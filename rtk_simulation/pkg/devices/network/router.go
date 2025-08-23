package network

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"rtk_simulation/pkg/devices/base"

	"github.com/sirupsen/logrus"
)

// Router 路由器設備模擬器
type Router struct {
	*base.BaseDevice

	// 路由器特有屬性
	wifiNetworks     []RouterWiFiNetwork
	ethernetPorts    []EthernetPort
	connectedClients map[string]*ConnectedClient
	dhcpAssignments  map[string]DHCPAssignment

	// 效能指標
	totalBandwidth  int     // Mbps
	bandwidthUsed   int     // Mbps
	cpuLoad         float64 // 0-100%
	memoryUsed      float64 // 0-100%
	packetForwarded int64
	packetsDropped  int64

	// WiFi 管理
	channelUtilization map[int]float64 // 頻道使用率
	interference       map[int]float64 // 干擾強度

	// 安全功能
	firewallEnabled bool
	vpnConnections  []VPNConnection

	// 統計資訊
	trafficStats RouterTrafficStats
	wifiStats    RouterWiFiStats

	// 日誌
	logger *logrus.Entry
}

// 使用 access_point.go 中定義的 WiFiNetwork 和 WiFiSecurity

// EthernetPort 乙太網路端口
type EthernetPort struct {
	Number    int    `json:"number"`
	Speed     int    `json:"speed"`  // Mbps
	Duplex    string `json:"duplex"` // full, half
	Status    string `json:"status"` // up, down, auto
	Connected bool   `json:"connected"`
	Device    string `json:"device,omitempty"` // 連接的設備 ID
}

// ConnectedClient 已連接的客戶端
type ConnectedClient struct {
	MAC          string    `json:"mac"`
	IP           string    `json:"ip"`
	DeviceID     string    `json:"device_id,omitempty"`
	Interface    string    `json:"interface"` // wifi_2g, wifi_5g, ethernet
	ConnectTime  time.Time `json:"connect_time"`
	RSSI         int       `json:"rssi,omitempty"` // WiFi 信號強度
	BytesRx      int64     `json:"bytes_rx"`
	BytesTx      int64     `json:"bytes_tx"`
	LastActivity time.Time `json:"last_activity"`
}

// DHCPAssignment DHCP 分配記錄
type DHCPAssignment struct {
	IP        string    `json:"ip"`
	MAC       string    `json:"mac"`
	Hostname  string    `json:"hostname"`
	LeaseTime time.Time `json:"lease_time"`
	Active    bool      `json:"active"`
}

// VPNConnection VPN 連接
type VPNConnection struct {
	Type        string    `json:"type"` // pptp, l2tp, openvpn
	RemoteIP    string    `json:"remote_ip"`
	Status      string    `json:"status"` // connected, disconnected, connecting
	BytesRx     int64     `json:"bytes_rx"`
	BytesTx     int64     `json:"bytes_tx"`
	ConnectTime time.Time `json:"connect_time"`
}

// RouterTrafficStats 路由器流量統計
type RouterTrafficStats struct {
	TotalPackets     int64 `json:"total_packets"`
	ForwardedPackets int64 `json:"forwarded_packets"`
	DroppedPackets   int64 `json:"dropped_packets"`
	ErrorPackets     int64 `json:"error_packets"`
	TotalBytes       int64 `json:"total_bytes"`
}

// RouterWiFiNetwork 路由器WiFi網路配置
type RouterWiFiNetwork struct {
	SSID        string       `json:"ssid"`
	Band        string       `json:"band"`
	Channel     int          `json:"channel"`
	TxPower     int          `json:"tx_power"`
	Hidden      bool         `json:"hidden"`
	MaxClients  int          `json:"max_clients"`
	ClientCount int          `json:"client_count"`
	Enabled     bool         `json:"enabled"`
	Security    WiFiSecurity `json:"security"`
}

// RouterWiFiStats 路由器 WiFi 統計
type RouterWiFiStats struct {
	TotalClients       int                `json:"total_clients"`
	ClientsByBand      map[string]int     `json:"clients_by_band"`
	ChannelUtilization map[int]float64    `json:"channel_utilization"`
	Interference       map[string]float64 `json:"interference"`
	AverageRSSI        map[string]float64 `json:"average_rssi"`
}

// NewRouter 建立新的路由器模擬器
func NewRouter(config base.DeviceConfig, mqttConfig base.MQTTConfig) (*Router, error) {
	baseDevice := base.NewBaseDeviceWithMQTT(config, mqttConfig)

	logger := logrus.WithFields(logrus.Fields{
		"device_id":   config.ID,
		"device_type": "router",
	})

	router := &Router{
		BaseDevice:         baseDevice,
		connectedClients:   make(map[string]*ConnectedClient),
		dhcpAssignments:    make(map[string]DHCPAssignment),
		channelUtilization: make(map[int]float64),
		interference:       make(map[int]float64),
		totalBandwidth:     1000, // 1 Gbps 預設
		firewallEnabled:    true,
		logger:             logger,
	}

	// 初始化預設 WiFi 網路
	router.initializeWiFiNetworks()

	// 初始化乙太網路端口
	router.initializeEthernetPorts()

	return router, nil
}

// Start 啟動路由器模擬器
func (r *Router) Start(ctx context.Context) error {
	if err := r.BaseDevice.Start(ctx); err != nil {
		return err
	}

	r.logger.Info("Router simulator started")

	// 啟動路由器特定的處理循環
	go r.wifiManagementLoop(ctx)
	go r.clientManagementLoop(ctx)
	go r.trafficMonitorLoop(ctx)

	return nil
}

// Stop 停止路由器模擬器
func (r *Router) Stop() error {
	r.logger.Info("Stopping router simulator")
	return r.BaseDevice.Stop()
}

// GenerateStatePayload 生成路由器狀態數據
func (r *Router) GenerateStatePayload() base.StatePayload {
	basePayload := r.BaseDevice.GenerateStatePayload()

	// 添加路由器特定狀態
	basePayload.Extra = map[string]interface{}{
		"router_status": map[string]interface{}{
			"total_bandwidth_mbps": r.totalBandwidth,
			"bandwidth_used_mbps":  r.bandwidthUsed,
			"connected_clients":    len(r.connectedClients),
			"wifi_networks":        r.getWiFiNetworkSummary(),
			"ethernet_ports":       r.getEthernetPortSummary(),
			"firewall_enabled":     r.firewallEnabled,
			"dhcp_assignments":     len(r.dhcpAssignments),
		},
		"performance": map[string]interface{}{
			"cpu_load":          r.cpuLoad,
			"memory_used":       r.memoryUsed,
			"packets_forwarded": r.packetForwarded,
			"packets_dropped":   r.packetsDropped,
		},
	}

	return basePayload
}

// GenerateTelemetryData 生成路由器遙測數據
func (r *Router) GenerateTelemetryData() map[string]base.TelemetryPayload {
	telemetry := r.BaseDevice.GenerateTelemetryData()

	// 網路流量統計
	telemetry["network_traffic"] = base.TelemetryPayload{
		"total_packets":         r.trafficStats.TotalPackets,
		"forwarded_packets":     r.trafficStats.ForwardedPackets,
		"dropped_packets":       r.trafficStats.DroppedPackets,
		"error_packets":         r.trafficStats.ErrorPackets,
		"total_bytes":           r.trafficStats.TotalBytes,
		"bandwidth_utilization": float64(r.bandwidthUsed) / float64(r.totalBandwidth) * 100,
	}

	// WiFi 統計
	telemetry["wifi_status"] = base.TelemetryPayload{
		"total_clients":       r.wifiStats.TotalClients,
		"clients_by_band":     r.wifiStats.ClientsByBand,
		"channel_utilization": r.channelUtilization,
		"interference":        r.interference,
		"networks":            r.getWiFiNetworkDetails(),
	}

	// 客戶端統計
	telemetry["client_stats"] = base.TelemetryPayload{
		"total_connected":  len(r.connectedClients),
		"wifi_clients":     r.countWiFiClients(),
		"ethernet_clients": r.countEthernetClients(),
		"dhcp_leases":      len(r.dhcpAssignments),
	}

	// 端口統計
	telemetry["port_status"] = base.TelemetryPayload{
		"ethernet_ports":   r.getEthernetPortDetails(),
		"port_utilization": r.calculatePortUtilization(),
	}

	return telemetry
}

// GenerateEvents 生成路由器事件
func (r *Router) GenerateEvents() []base.Event {
	var events []base.Event

	// 客戶端連接/斷開事件
	if rand.Float64() < 0.15 { // 15% 機率
		if rand.Float64() < 0.6 { // 60% 連接，40% 斷開
			events = append(events, r.generateClientConnectEvent())
		} else {
			events = append(events, r.generateClientDisconnectEvent())
		}
	}

	// 高負載警告
	if r.bandwidthUsed > int(float64(r.totalBandwidth)*0.8) { // 80% 以上
		events = append(events, base.Event{
			EventType: "router.high_bandwidth_usage",
			Severity:  "warning",
			Message:   fmt.Sprintf("Bandwidth usage is high: %d/%d Mbps", r.bandwidthUsed, r.totalBandwidth),
			Extra: map[string]interface{}{
				"bandwidth_used_mbps":  r.bandwidthUsed,
				"total_bandwidth_mbps": r.totalBandwidth,
				"utilization_percent":  float64(r.bandwidthUsed) / float64(r.totalBandwidth) * 100,
			},
		})
	}

	// WiFi 干擾檢測
	for channel, interference := range r.interference {
		if interference > 0.7 { // 70% 以上干擾
			events = append(events, base.Event{
				EventType: "router.wifi_interference",
				Severity:  "warning",
				Message:   fmt.Sprintf("High interference detected on channel %d", channel),
				Extra: map[string]interface{}{
					"channel":            channel,
					"interference_level": interference,
					"affected_clients":   r.getClientsOnChannel(channel),
				},
			})
		}
	}

	// DHCP 池即將用盡
	if len(r.dhcpAssignments) > 90 { // 假設池大小為 100
		events = append(events, base.Event{
			EventType: "router.dhcp_pool_exhaustion",
			Severity:  "warning",
			Message:   "DHCP pool is running low",
			Extra: map[string]interface{}{
				"active_leases": len(r.dhcpAssignments),
				"pool_size":     100,
				"remaining":     100 - len(r.dhcpAssignments),
			},
		})
	}

	return events
}

// HandleCommand 處理路由器命令
func (r *Router) HandleCommand(cmd base.Command) error {
	r.logger.WithField("command", cmd.Type).Info("Handling router command")

	switch cmd.Type {
	case "router.reboot":
		return r.handleReboot()
	case "router.wifi_scan":
		return r.handleWiFiScan()
	case "router.client_list":
		return r.handleClientList()
	case "router.bandwidth_test":
		return r.handleBandwidthTest()
	case "router.firmware_update":
		return r.handleFirmwareUpdate()
	case "router.reset_factory":
		return r.handleFactoryReset()
	default:
		return r.BaseDevice.HandleCommand(cmd)
	}
}

// 內部方法實作
func (r *Router) initializeWiFiNetworks() {
	r.wifiNetworks = []RouterWiFiNetwork{
		{
			SSID:        "HomeNetwork",
			Band:        "2.4GHz",
			Channel:     6,
			TxPower:     20,
			Hidden:      false,
			MaxClients:  15,
			ClientCount: 0,
			Enabled:     true,
			Security: WiFiSecurity{
				Type:       "WPA3",
				Password:   "homepassword123",
				Encryption: "AES",
			},
		},
		{
			SSID:        "HomeNetwork_5G",
			Band:        "5GHz",
			Channel:     36,
			TxPower:     23,
			Hidden:      false,
			MaxClients:  10,
			ClientCount: 0,
			Enabled:     true,
			Security: WiFiSecurity{
				Type:       "WPA3",
				Password:   "homepassword123",
				Encryption: "AES",
			},
		},
	}

	// 初始化頻道使用率
	r.channelUtilization[6] = 0.0  // 2.4GHz
	r.channelUtilization[36] = 0.0 // 5GHz

	// 初始化干擾程度
	r.interference[6] = 0.1   // 基礎干擾
	r.interference[36] = 0.05 // 5GHz 通常干擾較少
}

func (r *Router) initializeEthernetPorts() {
	r.ethernetPorts = []EthernetPort{
		{Number: 1, Speed: 1000, Duplex: "full", Status: "up", Connected: false},
		{Number: 2, Speed: 1000, Duplex: "full", Status: "up", Connected: false},
		{Number: 3, Speed: 1000, Duplex: "full", Status: "up", Connected: false},
		{Number: 4, Speed: 1000, Duplex: "full", Status: "up", Connected: false},
	}
}

// WiFi 管理循環
func (r *Router) wifiManagementLoop(ctx context.Context) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.updateWiFiStatus()
			r.optimizeChannels()
		}
	}
}

// 客戶端管理循環
func (r *Router) clientManagementLoop(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.updateClientActivity()
			r.cleanupInactiveClients()
		}
	}
}

// 流量監控循環
func (r *Router) trafficMonitorLoop(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.updateTrafficStats()
			r.updateBandwidthUsage()
		}
	}
}

// 實作其他輔助方法...
func (r *Router) updateWiFiStatus() {
	// 更新 WiFi 網路狀態
	for i := range r.wifiNetworks {
		network := &r.wifiNetworks[i]

		// 模擬客戶端數量變化
		if rand.Float64() < 0.3 { // 30% 機率變化
			if rand.Float64() < 0.5 && network.ClientCount > 0 {
				network.ClientCount--
			} else if network.ClientCount < network.MaxClients {
				network.ClientCount++
			}
		}

		// 更新頻道使用率
		utilization := float64(network.ClientCount) / float64(network.MaxClients)
		r.channelUtilization[network.Channel] = utilization

		// 模擬干擾變化
		baseInterference := 0.1
		if network.Band == "2.4GHz" {
			baseInterference = 0.2 // 2.4GHz 干擾較多
		}
		r.interference[network.Channel] = baseInterference + rand.Float64()*0.3
	}
}

func (r *Router) optimizeChannels() {
	// 簡單的頻道優化邏輯
	for i := range r.wifiNetworks {
		network := &r.wifiNetworks[i]
		currentInterference := r.interference[network.Channel]

		if currentInterference > 0.6 { // 干擾過高時嘗試切換頻道
			newChannel := r.findBestChannel(network.Band)
			if newChannel != network.Channel {
				r.logger.WithFields(logrus.Fields{
					"ssid":         network.SSID,
					"old_channel":  network.Channel,
					"new_channel":  newChannel,
					"interference": currentInterference,
				}).Info("Switching WiFi channel due to interference")

				network.Channel = newChannel
			}
		}
	}
}

func (r *Router) findBestChannel(band string) int {
	// 簡化的頻道選擇邏輯
	if band == "2.4GHz" {
		channels := []int{1, 6, 11}
		return channels[rand.Intn(len(channels))]
	} else {
		channels := []int{36, 40, 44, 48, 149, 153, 157, 161}
		return channels[rand.Intn(len(channels))]
	}
}

// 實作其他必要的輔助方法...
func (r *Router) getWiFiNetworkSummary() []map[string]interface{} {
	var summary []map[string]interface{}
	for _, network := range r.wifiNetworks {
		summary = append(summary, map[string]interface{}{
			"ssid":         network.SSID,
			"band":         network.Band,
			"channel":      network.Channel,
			"enabled":      network.Enabled,
			"client_count": network.ClientCount,
			"max_clients":  network.MaxClients,
		})
	}
	return summary
}

func (r *Router) getEthernetPortSummary() []map[string]interface{} {
	var summary []map[string]interface{}
	for _, port := range r.ethernetPorts {
		summary = append(summary, map[string]interface{}{
			"number":    port.Number,
			"speed":     port.Speed,
			"status":    port.Status,
			"connected": port.Connected,
		})
	}
	return summary
}

// 其他必要的方法實作省略...
func (r *Router) countWiFiClients() int                     { return 0 }
func (r *Router) countEthernetClients() int                 { return 0 }
func (r *Router) getWiFiNetworkDetails() interface{}        { return nil }
func (r *Router) getEthernetPortDetails() interface{}       { return nil }
func (r *Router) calculatePortUtilization() interface{}     { return nil }
func (r *Router) generateClientConnectEvent() base.Event    { return base.Event{} }
func (r *Router) generateClientDisconnectEvent() base.Event { return base.Event{} }
func (r *Router) getClientsOnChannel(channel int) []string  { return []string{} }
func (r *Router) updateClientActivity()                     {}
func (r *Router) cleanupInactiveClients()                   {}
func (r *Router) updateTrafficStats()                       {}
func (r *Router) updateBandwidthUsage()                     {}
func (r *Router) handleReboot() error                       { return nil }
func (r *Router) handleWiFiScan() error                     { return nil }
func (r *Router) handleClientList() error                   { return nil }
func (r *Router) handleBandwidthTest() error                { return nil }
func (r *Router) handleFirmwareUpdate() error               { return nil }
func (r *Router) handleFactoryReset() error                 { return nil }
