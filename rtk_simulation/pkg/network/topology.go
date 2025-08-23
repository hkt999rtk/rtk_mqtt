package network

import (
	"context"
	"fmt"
	"math/rand"
	"net"
	"sync"
	"time"

	"rtk_simulation/pkg/devices/base"

	"github.com/sirupsen/logrus"
)

// TopologyManager 網路拓撲管理器
type TopologyManager struct {
	devices     map[string]base.Device
	connections map[string]*Connection
	subnets     map[string]*Subnet
	dhcpPool    *DHCPPool
	mu          sync.RWMutex
	logger      *logrus.Entry
	running     bool
}

// Connection 網路連接
type Connection struct {
	ID         string           `json:"id"`
	Type       ConnectionType   `json:"type"`
	Device1    string           `json:"device1"`
	Device2    string           `json:"device2"`
	Status     ConnectionStatus `json:"status"`
	Bandwidth  int              `json:"bandwidth"`         // Mbps
	Latency    time.Duration    `json:"latency"`           // 延遲
	PacketLoss float64          `json:"packet_loss"`       // 封包遺失率 (0-1)
	RSSI       int              `json:"rssi,omitempty"`    // WiFi 信號強度 (dBm)
	Channel    int              `json:"channel,omitempty"` // WiFi 頻道
	Quality    int              `json:"quality"`           // 連接品質 (0-100)
	CreatedAt  time.Time        `json:"created_at"`
	UpdatedAt  time.Time        `json:"updated_at"`
}

// ConnectionType 連接類型
type ConnectionType string

const (
	ConnectionEthernet  ConnectionType = "ethernet"
	ConnectionWiFi      ConnectionType = "wifi"
	ConnectionMesh      ConnectionType = "mesh"
	ConnectionZigbee    ConnectionType = "zigbee"
	ConnectionBluetooth ConnectionType = "bluetooth"
)

// ConnectionStatus 連接狀態
type ConnectionStatus string

const (
	ConnectionActive       ConnectionStatus = "active"
	ConnectionInactive     ConnectionStatus = "inactive"
	ConnectionEstablishing ConnectionStatus = "establishing"
	ConnectionFailed       ConnectionStatus = "failed"
	ConnectionMaintenance  ConnectionStatus = "maintenance"
)

// Subnet 子網路
type Subnet struct {
	CIDR       string                 `json:"cidr"`
	Gateway    string                 `json:"gateway"`
	VLAN       int                    `json:"vlan,omitempty"`
	Devices    map[string]bool        `json:"devices"`
	Properties map[string]interface{} `json:"properties,omitempty"`
}

// DHCPPool DHCP 地址池
type DHCPPool struct {
	StartIP     net.IP                `json:"start_ip"`
	EndIP       net.IP                `json:"end_ip"`
	Gateway     net.IP                `json:"gateway"`
	DNS         []net.IP              `json:"dns"`
	LeaseTime   time.Duration         `json:"lease_time"`
	Assignments map[string]*DHCPLease `json:"assignments"`
	mu          sync.RWMutex
}

// DHCPLease DHCP 租用
type DHCPLease struct {
	IP        net.IP        `json:"ip"`
	MAC       string        `json:"mac"`
	DeviceID  string        `json:"device_id"`
	LeaseTime time.Duration `json:"lease_time"`
	StartTime time.Time     `json:"start_time"`
	Active    bool          `json:"active"`
}

// TopologyStats 拓撲統計
type TopologyStats struct {
	TotalDevices      int                    `json:"total_devices"`
	ActiveConnections int                    `json:"active_connections"`
	DevicesByType     map[string]int         `json:"devices_by_type"`
	ConnectionsByType map[ConnectionType]int `json:"connections_by_type"`
	AverageLatency    time.Duration          `json:"average_latency"`
	TotalBandwidth    int                    `json:"total_bandwidth"`
	PacketLossRate    float64                `json:"packet_loss_rate"`
}

// NewTopologyManager 建立新的拓撲管理器
func NewTopologyManager() *TopologyManager {
	return &TopologyManager{
		devices:     make(map[string]base.Device),
		connections: make(map[string]*Connection),
		subnets:     make(map[string]*Subnet),
		logger:      logrus.WithField("component", "topology_manager"),
	}
}

// Start 啟動拓撲管理器
func (tm *TopologyManager) Start(ctx context.Context) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if tm.running {
		return fmt.Errorf("topology manager is already running")
	}

	tm.running = true
	tm.logger.Info("Starting topology manager")

	// 啟動拓撲監控
	go tm.monitorTopology(ctx)

	return nil
}

// Stop 停止拓撲管理器
func (tm *TopologyManager) Stop() error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if !tm.running {
		return fmt.Errorf("topology manager is not running")
	}

	tm.running = false
	tm.logger.Info("Stopping topology manager")

	return nil
}

// AddDevice 添加設備到拓撲
func (tm *TopologyManager) AddDevice(device base.Device) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	deviceID := device.GetDeviceID()
	if _, exists := tm.devices[deviceID]; exists {
		return fmt.Errorf("device %s already exists", deviceID)
	}

	tm.devices[deviceID] = device
	tm.logger.WithField("device_id", deviceID).Info("Device added to topology")

	// 為設備分配 IP 地址（如果需要）
	if tm.dhcpPool != nil && device.GetIPAddress() == "" {
		if err := tm.assignIPAddress(device); err != nil {
			tm.logger.WithError(err).Warn("Failed to assign IP address")
		}
	}

	return nil
}

// RemoveDevice 從拓撲移除設備
func (tm *TopologyManager) RemoveDevice(deviceID string) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if _, exists := tm.devices[deviceID]; !exists {
		return fmt.Errorf("device %s not found", deviceID)
	}

	// 移除所有相關連接
	tm.removeDeviceConnections(deviceID)

	// 釋放 IP 地址
	if tm.dhcpPool != nil {
		tm.dhcpPool.ReleaseIP(deviceID)
	}

	delete(tm.devices, deviceID)
	tm.logger.WithField("device_id", deviceID).Info("Device removed from topology")

	return nil
}

// AddConnection 添加設備連接
func (tm *TopologyManager) AddConnection(device1ID, device2ID string, connType ConnectionType) (*Connection, error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	// 檢查設備是否存在
	if _, exists := tm.devices[device1ID]; !exists {
		return nil, fmt.Errorf("device %s not found", device1ID)
	}
	if _, exists := tm.devices[device2ID]; !exists {
		return nil, fmt.Errorf("device %s not found", device2ID)
	}

	connectionID := fmt.Sprintf("%s-%s", device1ID, device2ID)
	if _, exists := tm.connections[connectionID]; exists {
		return nil, fmt.Errorf("connection already exists between %s and %s", device1ID, device2ID)
	}

	connection := &Connection{
		ID:         connectionID,
		Type:       connType,
		Device1:    device1ID,
		Device2:    device2ID,
		Status:     ConnectionEstablishing,
		Bandwidth:  tm.getDefaultBandwidth(connType),
		Latency:    tm.getDefaultLatency(connType),
		PacketLoss: 0.0,
		Quality:    100,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	// 設定 WiFi 特定屬性
	if connType == ConnectionWiFi {
		connection.RSSI = -40 - (len(tm.connections) * 5) // 模擬信號衰減
		connection.Channel = tm.selectWiFiChannel()
	}

	tm.connections[connectionID] = connection
	tm.logger.WithFields(logrus.Fields{
		"connection_id": connectionID,
		"type":          connType,
		"device1":       device1ID,
		"device2":       device2ID,
	}).Info("Connection added")

	// 模擬連接建立過程
	go tm.establishConnection(connection)

	return connection, nil
}

// RemoveConnection 移除設備連接
func (tm *TopologyManager) RemoveConnection(connectionID string) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	connection, exists := tm.connections[connectionID]
	if !exists {
		return fmt.Errorf("connection %s not found", connectionID)
	}

	delete(tm.connections, connectionID)
	tm.logger.WithField("connection_id", connectionID).Info("Connection removed")

	// 通知相關設備連接已斷開
	tm.notifyConnectionStatus(connection, ConnectionInactive)

	return nil
}

// GetDevices 取得所有設備
func (tm *TopologyManager) GetDevices() map[string]base.Device {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	devices := make(map[string]base.Device)
	for id, device := range tm.devices {
		devices[id] = device
	}
	return devices
}

// GetConnections 取得所有連接
func (tm *TopologyManager) GetConnections() map[string]*Connection {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	connections := make(map[string]*Connection)
	for id, conn := range tm.connections {
		// 創建副本以避免並行修改
		connCopy := *conn
		connections[id] = &connCopy
	}
	return connections
}

// GetDeviceConnections 取得設備的所有連接
func (tm *TopologyManager) GetDeviceConnections(deviceID string) []*Connection {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	var connections []*Connection
	for _, conn := range tm.connections {
		if conn.Device1 == deviceID || conn.Device2 == deviceID {
			connCopy := *conn
			connections = append(connections, &connCopy)
		}
	}
	return connections
}

// GetStats 取得拓撲統計資訊
func (tm *TopologyManager) GetStats() TopologyStats {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	stats := TopologyStats{
		TotalDevices:      len(tm.devices),
		ActiveConnections: 0,
		DevicesByType:     make(map[string]int),
		ConnectionsByType: make(map[ConnectionType]int),
		TotalBandwidth:    0,
	}

	// 統計設備類型
	for _, device := range tm.devices {
		deviceType := device.GetDeviceType()
		stats.DevicesByType[deviceType]++
	}

	// 統計連接
	var totalLatency time.Duration
	var totalPacketLoss float64
	for _, conn := range tm.connections {
		stats.ConnectionsByType[conn.Type]++
		if conn.Status == ConnectionActive {
			stats.ActiveConnections++
			stats.TotalBandwidth += conn.Bandwidth
			totalLatency += conn.Latency
			totalPacketLoss += conn.PacketLoss
		}
	}

	// 計算平均值
	if stats.ActiveConnections > 0 {
		stats.AverageLatency = totalLatency / time.Duration(stats.ActiveConnections)
		stats.PacketLossRate = totalPacketLoss / float64(stats.ActiveConnections)
	}

	return stats
}

// SetDHCPPool 設定 DHCP 地址池
func (tm *TopologyManager) SetDHCPPool(startIP, endIP, gateway string, dns []string, leaseTime time.Duration) error {
	start := net.ParseIP(startIP)
	end := net.ParseIP(endIP)
	gw := net.ParseIP(gateway)

	if start == nil {
		return fmt.Errorf("invalid start IP address: %s", startIP)
	}
	if end == nil {
		return fmt.Errorf("invalid end IP address: %s", endIP)
	}
	if gw == nil {
		return fmt.Errorf("invalid gateway IP address: %s", gateway)
	}

	dnsIPs := make([]net.IP, len(dns))
	for i, dnsAddr := range dns {
		dnsIPs[i] = net.ParseIP(dnsAddr)
		if dnsIPs[i] == nil {
			return fmt.Errorf("invalid DNS address: %s", dnsAddr)
		}
	}

	tm.dhcpPool = &DHCPPool{
		StartIP:     start,
		EndIP:       end,
		Gateway:     gw,
		DNS:         dnsIPs,
		LeaseTime:   leaseTime,
		Assignments: make(map[string]*DHCPLease),
	}

	tm.logger.Info("DHCP pool configured")
	return nil
}

// 內部方法
func (tm *TopologyManager) monitorTopology(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if !tm.running {
				return
			}
			tm.updateConnectionQuality()
		}
	}
}

func (tm *TopologyManager) updateConnectionQuality() {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	for _, conn := range tm.connections {
		if conn.Status != ConnectionActive {
			continue
		}

		// 模擬網路狀況變化
		conn.UpdatedAt = time.Now()

		// 隨機調整延遲
		if conn.Type == ConnectionWiFi {
			conn.Latency += time.Duration((rand.Intn(10) - 5)) * time.Millisecond
			if conn.Latency < 1*time.Millisecond {
				conn.Latency = 1 * time.Millisecond
			}

			// 調整 RSSI
			conn.RSSI += rand.Intn(10) - 5
			if conn.RSSI > -20 {
				conn.RSSI = -20
			} else if conn.RSSI < -90 {
				conn.RSSI = -90
			}

			// 根據 RSSI 計算品質
			conn.Quality = tm.calculateWiFiQuality(conn.RSSI)
		}

		// 調整封包遺失率
		if conn.Quality < 50 {
			conn.PacketLoss = float64(50-conn.Quality) / 1000.0
		} else {
			conn.PacketLoss = 0.0
		}
	}
}

func (tm *TopologyManager) establishConnection(conn *Connection) {
	// 模擬連接建立過程
	time.Sleep(2 * time.Second)

	tm.mu.Lock()
	conn.Status = ConnectionActive
	conn.UpdatedAt = time.Now()
	tm.mu.Unlock()

	tm.logger.WithField("connection_id", conn.ID).Info("Connection established")
	tm.notifyConnectionStatus(conn, ConnectionActive)
}

func (tm *TopologyManager) removeDeviceConnections(deviceID string) {
	var toRemove []string
	for connID, conn := range tm.connections {
		if conn.Device1 == deviceID || conn.Device2 == deviceID {
			toRemove = append(toRemove, connID)
		}
	}

	for _, connID := range toRemove {
		delete(tm.connections, connID)
	}
}

func (tm *TopologyManager) assignIPAddress(device base.Device) error {
	if tm.dhcpPool == nil {
		return fmt.Errorf("DHCP pool not configured")
	}

	ip, err := tm.dhcpPool.AssignIP(device.GetDeviceID(), device.GetMACAddress())
	if err != nil {
		return err
	}

	tm.logger.WithFields(logrus.Fields{
		"device_id": device.GetDeviceID(),
		"ip":        ip.String(),
	}).Info("IP address assigned")

	return nil
}

func (tm *TopologyManager) notifyConnectionStatus(conn *Connection, status ConnectionStatus) {
	// 這裡可以實作事件通知機制
	tm.logger.WithFields(logrus.Fields{
		"connection_id": conn.ID,
		"status":        status,
	}).Debug("Connection status changed")
}

func (tm *TopologyManager) getDefaultBandwidth(connType ConnectionType) int {
	switch connType {
	case ConnectionEthernet:
		return 1000 // 1 Gbps
	case ConnectionWiFi:
		return 100 // 100 Mbps
	case ConnectionMesh:
		return 50 // 50 Mbps
	case ConnectionZigbee:
		return 1 // 1 Mbps
	case ConnectionBluetooth:
		return 2 // 2 Mbps
	default:
		return 10
	}
}

func (tm *TopologyManager) getDefaultLatency(connType ConnectionType) time.Duration {
	switch connType {
	case ConnectionEthernet:
		return 1 * time.Millisecond
	case ConnectionWiFi:
		return 5 * time.Millisecond
	case ConnectionMesh:
		return 10 * time.Millisecond
	case ConnectionZigbee:
		return 20 * time.Millisecond
	case ConnectionBluetooth:
		return 15 * time.Millisecond
	default:
		return 5 * time.Millisecond
	}
}

func (tm *TopologyManager) selectWiFiChannel() int {
	// 簡單的頻道選擇邏輯
	channels := []int{1, 6, 11, 36, 40, 44, 48}
	return channels[len(tm.connections)%len(channels)]
}

func (tm *TopologyManager) calculateWiFiQuality(rssi int) int {
	if rssi >= -30 {
		return 100
	} else if rssi >= -50 {
		return 80
	} else if rssi >= -70 {
		return 60
	} else if rssi >= -80 {
		return 40
	} else {
		return 20
	}
}

// DHCP 地址池方法
func (pool *DHCPPool) AssignIP(deviceID, macAddress string) (net.IP, error) {
	pool.mu.Lock()
	defer pool.mu.Unlock()

	// 檢查是否已經有租用
	if lease, exists := pool.Assignments[deviceID]; exists && lease.Active {
		return lease.IP, nil
	}

	// 尋找可用的 IP 地址
	start := pool.ipToInt(pool.StartIP)
	end := pool.ipToInt(pool.EndIP)

	for i := start; i <= end; i++ {
		ip := pool.intToIP(i)
		if !pool.isIPAssigned(ip) {
			lease := &DHCPLease{
				IP:        ip,
				MAC:       macAddress,
				DeviceID:  deviceID,
				LeaseTime: pool.LeaseTime,
				StartTime: time.Now(),
				Active:    true,
			}
			pool.Assignments[deviceID] = lease
			return ip, nil
		}
	}

	return nil, fmt.Errorf("no available IP addresses in DHCP pool")
}

func (pool *DHCPPool) ReleaseIP(deviceID string) {
	pool.mu.Lock()
	defer pool.mu.Unlock()

	if lease, exists := pool.Assignments[deviceID]; exists {
		lease.Active = false
	}
}

func (pool *DHCPPool) isIPAssigned(ip net.IP) bool {
	for _, lease := range pool.Assignments {
		if lease.Active && lease.IP.Equal(ip) {
			return true
		}
	}
	return false
}

func (pool *DHCPPool) ipToInt(ip net.IP) uint32 {
	ipv4 := ip.To4()
	if ipv4 == nil {
		return 0
	}
	return uint32(ipv4[0])<<24 + uint32(ipv4[1])<<16 + uint32(ipv4[2])<<8 + uint32(ipv4[3])
}

func (pool *DHCPPool) intToIP(n uint32) net.IP {
	return net.IPv4(byte(n>>24), byte(n>>16), byte(n>>8), byte(n))
}
