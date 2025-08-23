package network

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"rtk_simulation/pkg/devices/base"

	"github.com/sirupsen/logrus"
)

// MeshNode Mesh 節點設備模擬器
type MeshNode struct {
	*base.BaseDevice

	// Mesh 網路特有屬性
	meshID        string                      // Mesh 網路 ID
	meshRole      string                      // master, slave, repeater
	meshNeighbors map[string]*MeshNeighbor    // 鄰近的 mesh 節點
	meshRoutes    map[string]*MeshRoute       // 路由表
	backhaul      *MeshBackhaul               // 回程連接
	meshClients   map[string]*ConnectedClient // 連接的客戶端
	meshTopology  MeshTopology                // Mesh 拓撲資訊

	// 無線配置
	wifiNetworks  []MeshWiFiNetwork // 無線網路配置
	channelConfig MeshChannelConfig // 頻道配置

	// 效能指標
	meshMetrics     MeshMetrics     // Mesh 網路指標
	nodePerformance NodePerformance // 節點效能

	// 自動優化
	selfHealing   bool // 自愈功能
	autoChannel   bool // 自動頻道選擇
	loadBalancing bool // 負載均衡

	// 統計資訊
	trafficStats MeshTrafficStats // 流量統計
	hopCount     int              // 到主節點的跳數

	// 狀態
	health string // 設備健康狀態

	// 狀態同步
	syncMutex    sync.RWMutex
	lastSyncTime time.Time

	// 日誌
	logger *logrus.Entry
}

// MeshNeighbor Mesh 鄰居節點
type MeshNeighbor struct {
	NodeID      string    `json:"node_id"`
	MACAddress  string    `json:"mac_address"`
	IPAddress   string    `json:"ip_address"`
	LinkQuality float64   `json:"link_quality"` // 0-100%
	RSSI        int       `json:"rssi"`         // dBm
	Distance    int       `json:"distance"`     // 跳數
	LastSeen    time.Time `json:"last_seen"`
	Status      string    `json:"status"`    // active, inactive, lost
	Bandwidth   int       `json:"bandwidth"` // Mbps
	Latency     float64   `json:"latency"`   // ms
}

// MeshRoute Mesh 路由
type MeshRoute struct {
	Destination string    `json:"destination"`
	NextHop     string    `json:"next_hop"`
	Metric      int       `json:"metric"`
	Interface   string    `json:"interface"`
	Status      string    `json:"status"` // active, backup, invalid
	UpdateTime  time.Time `json:"update_time"`
}

// MeshBackhaul Mesh 回程連接
type MeshBackhaul struct {
	Type        string    `json:"type"`        // wireless, ethernet
	Interface   string    `json:"interface"`   // 5G, 2.4G, eth0
	Status      string    `json:"status"`      // connected, disconnected
	Speed       int       `json:"speed"`       // Mbps
	ParentNode  string    `json:"parent_node"` // 上級節點 ID
	ConnectTime time.Time `json:"connect_time"`
}

// MeshWiFiNetwork Mesh WiFi 網路配置
type MeshWiFiNetwork struct {
	Band        string  `json:"band"` // 2.4GHz, 5GHz, 6GHz
	SSID        string  `json:"ssid"`
	Channel     int     `json:"channel"`
	Width       int     `json:"width"` // 20, 40, 80, 160 MHz
	Mode        string  `json:"mode"`  // backhaul, fronthaul, dual
	MaxClients  int     `json:"max_clients"`
	ClientCount int     `json:"client_count"`
	Utilization float64 `json:"utilization"` // 0-100%
}

// MeshChannelConfig Mesh 頻道配置
type MeshChannelConfig struct {
	BackhaulChannel   int     `json:"backhaul_channel"`
	FronthaulChannel  int     `json:"fronthaul_channel"`
	AutoChannel       bool    `json:"auto_channel"`
	ChannelScanPeriod int     `json:"channel_scan_period"` // seconds
	Interference      float64 `json:"interference"`        // 0-100%
}

// MeshTopology Mesh 拓撲資訊
type MeshTopology struct {
	TopologyType   string `json:"topology_type"` // star, mesh, tree, hybrid
	NodeCount      int    `json:"node_count"`
	MaxHops        int    `json:"max_hops"`
	RootNode       string `json:"root_node"`
	BackupRoot     string `json:"backup_root"`
	RedundantPaths int    `json:"redundant_paths"`
}

// MeshMetrics Mesh 網路指標
type MeshMetrics struct {
	MeshThroughput  float64 `json:"mesh_throughput"`  // Mbps
	AverageLatency  float64 `json:"average_latency"`  // ms
	PacketLoss      float64 `json:"packet_loss"`      // %
	RoutingOverhead float64 `json:"routing_overhead"` // %
	MeshEfficiency  float64 `json:"mesh_efficiency"`  // %
	PathReliability float64 `json:"path_reliability"` // %
}

// NodePerformance 節點效能
type NodePerformance struct {
	CPUUsage         float64 `json:"cpu_usage"`         // %
	MemoryUsage      float64 `json:"memory_usage"`      // %
	Temperature      float64 `json:"temperature"`       // °C
	PowerConsumption float64 `json:"power_consumption"` // Watts
	Uptime           int64   `json:"uptime"`            // seconds
}

// MeshTrafficStats Mesh 流量統計
type MeshTrafficStats struct {
	LocalTraffic     int64 `json:"local_traffic"`     // bytes
	ForwardedTraffic int64 `json:"forwarded_traffic"` // bytes
	BackhaulTraffic  int64 `json:"backhaul_traffic"`  // bytes
	BroadcastTraffic int64 `json:"broadcast_traffic"` // bytes
	MulticastTraffic int64 `json:"multicast_traffic"` // bytes
	ControlTraffic   int64 `json:"control_traffic"`   // bytes
}

// NewMeshNode 創建新的 Mesh 節點
func NewMeshNode(config base.DeviceConfig, mqttConfig base.MQTTConfig) (*MeshNode, error) {
	baseDevice := base.NewBaseDeviceWithMQTT(config, mqttConfig)

	// 從配置中讀取 mesh 特定設置
	meshRole := "slave"
	if role, ok := config.Extra["mesh_role"].(string); ok {
		meshRole = role
	}

	meshID := "mesh_network_001"
	if id, ok := config.Extra["mesh_id"].(string); ok {
		meshID = id
	}

	selfHealing := true
	if sh, ok := config.Extra["self_healing"].(bool); ok {
		selfHealing = sh
	}

	autoChannel := true
	if ac, ok := config.Extra["auto_channel"].(bool); ok {
		autoChannel = ac
	}

	loadBalancing := true
	if lb, ok := config.Extra["load_balancing"].(bool); ok {
		loadBalancing = lb
	}

	mn := &MeshNode{
		BaseDevice:    baseDevice,
		meshID:        meshID,
		meshRole:      meshRole,
		meshNeighbors: make(map[string]*MeshNeighbor),
		meshRoutes:    make(map[string]*MeshRoute),
		meshClients:   make(map[string]*ConnectedClient),
		selfHealing:   selfHealing,
		autoChannel:   autoChannel,
		loadBalancing: loadBalancing,
		hopCount:      0,
		health:        "healthy",
		lastSyncTime:  time.Now(),
		logger: logrus.WithFields(logrus.Fields{
			"device_type": "mesh_node",
			"device_id":   config.ID,
			"mesh_id":     meshID,
		}),
	}

	// 初始化 WiFi 網路配置
	mn.initializeWiFiNetworks(config)

	// 初始化頻道配置
	mn.initializeChannelConfig()

	// 初始化拓撲資訊
	mn.initializeTopology()

	// 初始化效能指標
	mn.initializeMetrics()

	// 如果是主節點，設置回程連接
	if meshRole == "master" {
		mn.backhaul = &MeshBackhaul{
			Type:        "ethernet",
			Interface:   "eth0",
			Status:      "connected",
			Speed:       1000,
			ParentNode:  "gateway",
			ConnectTime: time.Now(),
		}
		mn.hopCount = 0
	} else {
		mn.backhaul = &MeshBackhaul{
			Type:        "wireless",
			Interface:   "5G",
			Status:      "searching",
			Speed:       0,
			ParentNode:  "",
			ConnectTime: time.Time{},
		}
		mn.hopCount = -1 // 未連接
	}

	return mn, nil
}

// initializeWiFiNetworks 初始化 WiFi 網路配置
func (mn *MeshNode) initializeWiFiNetworks(config base.DeviceConfig) {
	// 5GHz 回程網路
	backhaul5G := MeshWiFiNetwork{
		Band:        "5GHz",
		SSID:        fmt.Sprintf("%s_backhaul", mn.meshID),
		Channel:     149,
		Width:       80,
		Mode:        "backhaul",
		MaxClients:  10,
		ClientCount: 0,
		Utilization: 0,
	}

	// 2.4GHz 前程網路
	fronthaul2G := MeshWiFiNetwork{
		Band:        "2.4GHz",
		SSID:        fmt.Sprintf("%s_2G", mn.meshID),
		Channel:     6,
		Width:       20,
		Mode:        "fronthaul",
		MaxClients:  32,
		ClientCount: 0,
		Utilization: 0,
	}

	// 5GHz 前程網路
	fronthaul5G := MeshWiFiNetwork{
		Band:        "5GHz",
		SSID:        fmt.Sprintf("%s_5G", mn.meshID),
		Channel:     36,
		Width:       80,
		Mode:        "fronthaul",
		MaxClients:  64,
		ClientCount: 0,
		Utilization: 0,
	}

	// 從配置中覆蓋設置
	if ssid2g, ok := config.Extra["ssid_2g"].(string); ok {
		fronthaul2G.SSID = ssid2g
	}
	if ssid5g, ok := config.Extra["ssid_5g"].(string); ok {
		fronthaul5G.SSID = ssid5g
	}

	mn.wifiNetworks = []MeshWiFiNetwork{backhaul5G, fronthaul2G, fronthaul5G}
}

// initializeChannelConfig 初始化頻道配置
func (mn *MeshNode) initializeChannelConfig() {
	mn.channelConfig = MeshChannelConfig{
		BackhaulChannel:   149,
		FronthaulChannel:  36,
		AutoChannel:       mn.autoChannel,
		ChannelScanPeriod: 300, // 5 minutes
		Interference:      getRandomFloat(0, 20),
	}
}

// initializeTopology 初始化拓撲資訊
func (mn *MeshNode) initializeTopology() {
	mn.meshTopology = MeshTopology{
		TopologyType:   "mesh",
		NodeCount:      1,
		MaxHops:        3,
		RootNode:       "",
		BackupRoot:     "",
		RedundantPaths: 0,
	}
}

// initializeMetrics 初始化效能指標
func (mn *MeshNode) initializeMetrics() {
	mn.meshMetrics = MeshMetrics{
		MeshThroughput:  getRandomFloat(100, 500),
		AverageLatency:  getRandomFloat(1, 10),
		PacketLoss:      getRandomFloat(0, 0.5),
		RoutingOverhead: getRandomFloat(5, 15),
		MeshEfficiency:  getRandomFloat(80, 95),
		PathReliability: getRandomFloat(95, 99.9),
	}

	mn.nodePerformance = NodePerformance{
		CPUUsage:         getRandomFloat(20, 40),
		MemoryUsage:      getRandomFloat(30, 50),
		Temperature:      getRandomFloat(35, 55),
		PowerConsumption: getRandomFloat(5, 15),
		Uptime:           0,
	}

	mn.trafficStats = MeshTrafficStats{
		LocalTraffic:     0,
		ForwardedTraffic: 0,
		BackhaulTraffic:  0,
		BroadcastTraffic: 0,
		MulticastTraffic: 0,
		ControlTraffic:   0,
	}
}

// Start 啟動 Mesh 節點
func (mn *MeshNode) Start(ctx context.Context) error {
	if err := mn.BaseDevice.Start(ctx); err != nil {
		return err
	}

	mn.logger.Info("Starting Mesh node")

	// 啟動 mesh 特定的 goroutines
	go mn.meshDiscoveryLoop(ctx)
	go mn.meshOptimizationLoop(ctx)
	go mn.meshSyncLoop(ctx)

	// 如果不是主節點，嘗試連接到網路
	if mn.meshRole != "master" {
		go mn.connectToMeshNetwork(ctx)
	}

	return nil
}

// meshDiscoveryLoop Mesh 發現循環
func (mn *MeshNode) meshDiscoveryLoop(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			mn.discoverNeighbors()
			mn.updateRoutes()
		}
	}
}

// meshOptimizationLoop Mesh 優化循環
func (mn *MeshNode) meshOptimizationLoop(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if mn.autoChannel {
				mn.optimizeChannels()
			}
			if mn.loadBalancing {
				mn.balanceLoad()
			}
			if mn.selfHealing {
				mn.checkAndHeal()
			}
		}
	}
}

// meshSyncLoop Mesh 同步循環
func (mn *MeshNode) meshSyncLoop(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			mn.syncWithNeighbors()
		}
	}
}

// connectToMeshNetwork 連接到 Mesh 網路
func (mn *MeshNode) connectToMeshNetwork(ctx context.Context) {
	retryCount := 0
	maxRetries := 10

	for retryCount < maxRetries {
		select {
		case <-ctx.Done():
			return
		default:
			// 模擬搜尋和連接過程
			time.Sleep(time.Duration(2+rand.Intn(3)) * time.Second)

			// 模擬成功連接
			if rand.Float64() > 0.3 {
				mn.syncMutex.Lock()
				mn.backhaul.Status = "connected"
				mn.backhaul.Speed = 300 + rand.Intn(500)
				mn.backhaul.ParentNode = fmt.Sprintf("mesh_node_%d", rand.Intn(3))
				mn.backhaul.ConnectTime = time.Now()
				mn.hopCount = 1 + rand.Intn(3)
				mn.syncMutex.Unlock()

				mn.logger.Info("Successfully connected to mesh network")
				return
			}

			retryCount++
			mn.logger.Warnf("Failed to connect to mesh network, retry %d/%d", retryCount, maxRetries)
		}
	}

	mn.logger.Error("Failed to connect to mesh network after maximum retries")
}

// discoverNeighbors 發現鄰居節點
func (mn *MeshNode) discoverNeighbors() {
	mn.syncMutex.Lock()
	defer mn.syncMutex.Unlock()

	// 模擬發現新的鄰居
	if rand.Float64() > 0.7 && len(mn.meshNeighbors) < 5 {
		neighborID := fmt.Sprintf("mesh_node_%d", rand.Intn(10))
		if _, exists := mn.meshNeighbors[neighborID]; !exists {
			mn.meshNeighbors[neighborID] = &MeshNeighbor{
				NodeID:      neighborID,
				MACAddress:  fmt.Sprintf("aa:bb:cc:dd:ee:%02x", rand.Intn(256)),
				IPAddress:   fmt.Sprintf("192.168.1.%d", 100+rand.Intn(50)),
				LinkQuality: getRandomFloat(70, 100),
				RSSI:        -30 - rand.Intn(40),
				Distance:    1 + rand.Intn(3),
				LastSeen:    time.Now(),
				Status:      "active",
				Bandwidth:   100 + rand.Intn(400),
				Latency:     getRandomFloat(1, 10),
			}
			mn.logger.Infof("Discovered new mesh neighbor: %s", neighborID)
		}
	}

	// 更新現有鄰居狀態
	for id, neighbor := range mn.meshNeighbors {
		if time.Since(neighbor.LastSeen) > 30*time.Second {
			neighbor.Status = "inactive"
		} else {
			neighbor.LinkQuality = getRandomFloat(60, 100)
			neighbor.RSSI = -30 - rand.Intn(50)
			neighbor.Latency = getRandomFloat(1, 15)
			neighbor.LastSeen = time.Now()
		}

		// 移除失去連接的鄰居
		if time.Since(neighbor.LastSeen) > 60*time.Second {
			delete(mn.meshNeighbors, id)
			mn.logger.Infof("Lost mesh neighbor: %s", id)
		}
	}
}

// updateRoutes 更新路由表
func (mn *MeshNode) updateRoutes() {
	mn.syncMutex.Lock()
	defer mn.syncMutex.Unlock()

	// 為每個鄰居創建或更新路由
	for id, neighbor := range mn.meshNeighbors {
		routeKey := fmt.Sprintf("route_to_%s", id)

		if route, exists := mn.meshRoutes[routeKey]; exists {
			// 更新現有路由
			route.Metric = int(100 - neighbor.LinkQuality)
			route.Status = neighbor.Status
			route.UpdateTime = time.Now()
		} else {
			// 創建新路由
			mn.meshRoutes[routeKey] = &MeshRoute{
				Destination: neighbor.IPAddress,
				NextHop:     neighbor.IPAddress,
				Metric:      int(100 - neighbor.LinkQuality),
				Interface:   "mesh0",
				Status:      neighbor.Status,
				UpdateTime:  time.Now(),
			}
		}
	}

	// 清理過期路由
	for key, route := range mn.meshRoutes {
		if time.Since(route.UpdateTime) > 60*time.Second {
			delete(mn.meshRoutes, key)
		}
	}
}

// optimizeChannels 優化頻道選擇
func (mn *MeshNode) optimizeChannels() {
	mn.syncMutex.Lock()
	defer mn.syncMutex.Unlock()

	// 模擬頻道掃描和優化
	currentInterference := mn.channelConfig.Interference

	// 隨機模擬干擾變化
	mn.channelConfig.Interference = getRandomFloat(0, 30)

	// 如果干擾太高，切換頻道
	if mn.channelConfig.Interference > 25 {
		// 選擇新的頻道
		channels5G := []int{36, 40, 44, 48, 149, 153, 157, 161}
		channels2G := []int{1, 6, 11}

		for i, network := range mn.wifiNetworks {
			if network.Band == "5GHz" && network.Mode == "backhaul" {
				mn.wifiNetworks[i].Channel = channels5G[rand.Intn(len(channels5G))]
				mn.logger.Infof("Changed backhaul channel to %d due to interference", mn.wifiNetworks[i].Channel)
			} else if network.Band == "2.4GHz" {
				mn.wifiNetworks[i].Channel = channels2G[rand.Intn(len(channels2G))]
			}
		}

		mn.channelConfig.BackhaulChannel = mn.wifiNetworks[0].Channel
	}

	mn.logger.Debugf("Channel optimization: interference changed from %.1f%% to %.1f%%",
		currentInterference, mn.channelConfig.Interference)
}

// balanceLoad 負載均衡
func (mn *MeshNode) balanceLoad() {
	mn.syncMutex.Lock()
	defer mn.syncMutex.Unlock()

	// 更新網路使用率
	for i := range mn.wifiNetworks {
		mn.wifiNetworks[i].Utilization = getRandomFloat(10, 80)
		mn.wifiNetworks[i].ClientCount = rand.Intn(mn.wifiNetworks[i].MaxClients)
	}

	// 模擬流量重新分配
	totalTraffic := mn.trafficStats.LocalTraffic + mn.trafficStats.ForwardedTraffic
	if totalTraffic > 1000000000 { // > 1GB
		// 觸發負載均衡
		mn.logger.Info("Triggering load balancing due to high traffic")

		// 重新分配客戶端到不同的頻段
		for _, client := range mn.meshClients {
			if rand.Float64() > 0.5 {
				if client.Interface == "wifi_2g" {
					client.Interface = "wifi_5g"
				} else if client.Interface == "wifi_5g" {
					client.Interface = "wifi_2g"
				}
			}
		}
	}
}

// checkAndHeal 自愈檢查
func (mn *MeshNode) checkAndHeal() {
	mn.syncMutex.Lock()
	defer mn.syncMutex.Unlock()

	// 檢查回程連接
	if mn.backhaul.Status != "connected" && mn.meshRole != "master" {
		mn.logger.Warn("Backhaul disconnected, attempting to reconnect")
		go mn.connectToMeshNetwork(context.Background())
	}

	// 檢查路徑可靠性
	if mn.meshMetrics.PathReliability < 90 {
		mn.logger.Warn("Low path reliability detected, searching for alternative routes")

		// 尋找備用路由
		for _, route := range mn.meshRoutes {
			if route.Status == "backup" {
				route.Status = "active"
				mn.logger.Infof("Activated backup route to %s", route.Destination)
				break
			}
		}
	}

	// 更新自愈指標
	mn.meshMetrics.PathReliability = getRandomFloat(90, 99.9)
}

// syncWithNeighbors 與鄰居同步
func (mn *MeshNode) syncWithNeighbors() {
	mn.syncMutex.Lock()
	defer mn.syncMutex.Unlock()

	// 更新同步時間
	mn.lastSyncTime = time.Now()

	// 更新拓撲資訊
	mn.meshTopology.NodeCount = len(mn.meshNeighbors) + 1
	mn.meshTopology.RedundantPaths = 0

	// 計算冗餘路徑
	destinations := make(map[string]int)
	for _, route := range mn.meshRoutes {
		destinations[route.Destination]++
	}
	for _, count := range destinations {
		if count > 1 {
			mn.meshTopology.RedundantPaths++
		}
	}

	// 更新效能指標
	mn.updatePerformanceMetrics()
}

// updatePerformanceMetrics 更新效能指標
func (mn *MeshNode) updatePerformanceMetrics() {
	// 更新 CPU 和記憶體使用率
	mn.nodePerformance.CPUUsage = getRandomFloat(15, 60)
	mn.nodePerformance.MemoryUsage = getRandomFloat(25, 70)
	mn.nodePerformance.Temperature = getRandomFloat(35, 65)
	mn.nodePerformance.PowerConsumption = getRandomFloat(5, 20)
	mn.nodePerformance.Uptime = int64(mn.BaseDevice.GetUptime().Seconds())

	// 更新 mesh 指標
	mn.meshMetrics.MeshThroughput = getRandomFloat(100, 800)
	mn.meshMetrics.AverageLatency = getRandomFloat(1, 15)
	mn.meshMetrics.PacketLoss = getRandomFloat(0, 1)
	mn.meshMetrics.RoutingOverhead = getRandomFloat(5, 20)
	mn.meshMetrics.MeshEfficiency = getRandomFloat(75, 95)

	// 更新流量統計
	mn.trafficStats.LocalTraffic += int64(rand.Intn(1000000))
	mn.trafficStats.ForwardedTraffic += int64(rand.Intn(2000000))
	mn.trafficStats.BackhaulTraffic += int64(rand.Intn(1500000))
	mn.trafficStats.BroadcastTraffic += int64(rand.Intn(100000))
	mn.trafficStats.MulticastTraffic += int64(rand.Intn(200000))
	mn.trafficStats.ControlTraffic += int64(rand.Intn(50000))
}

// GenerateStatePayload 生成狀態負載
func (mn *MeshNode) GenerateStatePayload() base.StatePayload {
	mn.syncMutex.RLock()
	defer mn.syncMutex.RUnlock()

	state := mn.BaseDevice.GenerateStatePayload()

	// 添加 mesh 特定的狀態資訊
	if state.Extra == nil {
		state.Extra = make(map[string]interface{})
	}

	state.Extra["mesh"] = map[string]interface{}{
		"mesh_id":        mn.meshID,
		"mesh_role":      mn.meshRole,
		"hop_count":      mn.hopCount,
		"neighbor_count": len(mn.meshNeighbors),
		"route_count":    len(mn.meshRoutes),
		"client_count":   len(mn.meshClients),
		"backhaul":       mn.backhaul,
		"topology":       mn.meshTopology,
		"self_healing":   mn.selfHealing,
		"auto_channel":   mn.autoChannel,
		"load_balancing": mn.loadBalancing,
	}

	state.Extra["wifi_networks"] = mn.wifiNetworks
	state.Extra["channel_config"] = mn.channelConfig

	return state
}

// GenerateTelemetryData 生成遙測數據
func (mn *MeshNode) GenerateTelemetryData() map[string]base.TelemetryPayload {
	mn.syncMutex.RLock()
	defer mn.syncMutex.RUnlock()

	telemetry := mn.BaseDevice.GenerateTelemetryData()

	// 添加 mesh 特定的遙測數據
	telemetry["mesh_metrics"] = base.TelemetryPayload{
		"mesh_throughput":  mn.meshMetrics.MeshThroughput,
		"average_latency":  mn.meshMetrics.AverageLatency,
		"packet_loss":      mn.meshMetrics.PacketLoss,
		"routing_overhead": mn.meshMetrics.RoutingOverhead,
		"mesh_efficiency":  mn.meshMetrics.MeshEfficiency,
		"path_reliability": mn.meshMetrics.PathReliability,
	}

	telemetry["node_performance"] = base.TelemetryPayload{
		"cpu_usage":         mn.nodePerformance.CPUUsage,
		"memory_usage":      mn.nodePerformance.MemoryUsage,
		"temperature":       mn.nodePerformance.Temperature,
		"power_consumption": mn.nodePerformance.PowerConsumption,
		"uptime":            mn.nodePerformance.Uptime,
	}

	telemetry["traffic_stats"] = base.TelemetryPayload{
		"local_traffic":     mn.trafficStats.LocalTraffic,
		"forwarded_traffic": mn.trafficStats.ForwardedTraffic,
		"backhaul_traffic":  mn.trafficStats.BackhaulTraffic,
		"broadcast_traffic": mn.trafficStats.BroadcastTraffic,
		"multicast_traffic": mn.trafficStats.MulticastTraffic,
		"control_traffic":   mn.trafficStats.ControlTraffic,
	}

	telemetry["last_sync_time"] = base.TelemetryPayload{
		"timestamp": mn.lastSyncTime.Unix(),
	}

	// 添加鄰居健康狀態
	neighborHealth := make(base.TelemetryPayload)
	for id, neighbor := range mn.meshNeighbors {
		neighborHealth[id] = map[string]interface{}{
			"link_quality": neighbor.LinkQuality,
			"rssi":         neighbor.RSSI,
			"latency":      neighbor.Latency,
			"status":       neighbor.Status,
		}
	}
	telemetry["neighbor_health"] = neighborHealth

	return telemetry
}

// GenerateEvents 生成事件
func (mn *MeshNode) GenerateEvents() []base.Event {
	events := mn.BaseDevice.GenerateEvents()

	// Mesh 特定事件
	if rand.Float64() > 0.95 {
		events = append(events, base.Event{
			EventType: "mesh_topology_change",
			Severity:  "info",
			Message: fmt.Sprintf("Mesh topology changed: %d neighbors, %d routes",
				len(mn.meshNeighbors), len(mn.meshRoutes)),
			Extra: map[string]interface{}{
				"neighbor_count": len(mn.meshNeighbors),
				"route_count":    len(mn.meshRoutes),
				"hop_count":      mn.hopCount,
			},
		})
	}

	if rand.Float64() > 0.98 {
		events = append(events, base.Event{
			EventType: "mesh_optimization",
			Severity:  "info",
			Message:   "Mesh network optimization performed",
			Extra: map[string]interface{}{
				"channel_interference": mn.channelConfig.Interference,
				"mesh_efficiency":      mn.meshMetrics.MeshEfficiency,
			},
		})
	}

	if mn.meshMetrics.PacketLoss > 2 && rand.Float64() > 0.9 {
		events = append(events, base.Event{
			EventType: "high_packet_loss",
			Severity:  "warning",
			Message:   fmt.Sprintf("High packet loss detected: %.2f%%", mn.meshMetrics.PacketLoss),
			Extra: map[string]interface{}{
				"packet_loss":      mn.meshMetrics.PacketLoss,
				"path_reliability": mn.meshMetrics.PathReliability,
			},
		})
	}

	if mn.backhaul.Status != "connected" && mn.meshRole != "master" {
		events = append(events, base.Event{
			EventType: "backhaul_disconnected",
			Severity:  "error",
			Message:   "Mesh backhaul connection lost",
			Extra: map[string]interface{}{
				"previous_parent": mn.backhaul.ParentNode,
				"reconnecting":    true,
			},
		})
	}

	return events
}

// HandleCommand 處理命令
func (mn *MeshNode) HandleCommand(cmd base.Command) error {
	mn.syncMutex.Lock()
	defer mn.syncMutex.Unlock()

	switch cmd.Type {
	case "reboot":
		mn.logger.Info("Rebooting mesh node")
		// 模擬重啟
		mn.health = "rebooting"
		go func() {
			time.Sleep(30 * time.Second)
			mn.syncMutex.Lock()
			mn.health = "healthy"
			mn.backhaul.Status = "searching"
			mn.syncMutex.Unlock()
			go mn.connectToMeshNetwork(context.Background())
		}()
		return nil

	case "optimize_mesh":
		mn.logger.Info("Optimizing mesh network")
		mn.optimizeChannels()
		mn.balanceLoad()
		return nil

	case "reset_mesh":
		mn.logger.Info("Resetting mesh configuration")
		mn.meshNeighbors = make(map[string]*MeshNeighbor)
		mn.meshRoutes = make(map[string]*MeshRoute)
		mn.hopCount = -1
		if mn.meshRole != "master" {
			mn.backhaul.Status = "searching"
			go mn.connectToMeshNetwork(context.Background())
		}
		return nil

	case "set_mesh_role":
		if role, ok := cmd.Parameters["role"].(string); ok {
			mn.meshRole = role
			mn.logger.Infof("Changed mesh role to: %s", role)
			return nil
		}
		return fmt.Errorf("missing role parameter")

	case "enable_feature":
		if feature, ok := cmd.Parameters["feature"].(string); ok {
			switch feature {
			case "self_healing":
				mn.selfHealing = true
			case "auto_channel":
				mn.autoChannel = true
			case "load_balancing":
				mn.loadBalancing = true
			default:
				return fmt.Errorf("unknown feature: %s", feature)
			}
			mn.logger.Infof("Enabled feature: %s", feature)
			return nil
		}
		return fmt.Errorf("missing feature parameter")

	case "disable_feature":
		if feature, ok := cmd.Parameters["feature"].(string); ok {
			switch feature {
			case "self_healing":
				mn.selfHealing = false
			case "auto_channel":
				mn.autoChannel = false
			case "load_balancing":
				mn.loadBalancing = false
			default:
				return fmt.Errorf("unknown feature: %s", feature)
			}
			mn.logger.Infof("Disabled feature: %s", feature)
			return nil
		}
		return fmt.Errorf("missing feature parameter")

	default:
		// 傳遞給基礎設備處理
		return mn.BaseDevice.HandleCommand(cmd)
	}
}

// AddNeighbor 添加鄰居節點 (用於設備互動)
func (mn *MeshNode) AddNeighbor(neighborID string, neighbor *MeshNeighbor) {
	mn.syncMutex.Lock()
	defer mn.syncMutex.Unlock()

	mn.meshNeighbors[neighborID] = neighbor
	mn.logger.Infof("Added mesh neighbor: %s", neighborID)
}

// RemoveNeighbor 移除鄰居節點
func (mn *MeshNode) RemoveNeighbor(neighborID string) {
	mn.syncMutex.Lock()
	defer mn.syncMutex.Unlock()

	delete(mn.meshNeighbors, neighborID)
	mn.logger.Infof("Removed mesh neighbor: %s", neighborID)
}

// GetNeighbors 獲取所有鄰居節點
func (mn *MeshNode) GetNeighbors() map[string]*MeshNeighbor {
	mn.syncMutex.RLock()
	defer mn.syncMutex.RUnlock()

	// 返回副本以避免並發問題
	neighbors := make(map[string]*MeshNeighbor)
	for k, v := range mn.meshNeighbors {
		neighbors[k] = v
	}
	return neighbors
}

// GetMeshMetrics 獲取 mesh 指標
func (mn *MeshNode) GetMeshMetrics() MeshMetrics {
	mn.syncMutex.RLock()
	defer mn.syncMutex.RUnlock()

	return mn.meshMetrics
}

// SetBackhaulStatus 設置回程連接狀態 (用於設備互動)
func (mn *MeshNode) SetBackhaulStatus(status string, parentNode string) {
	mn.syncMutex.Lock()
	defer mn.syncMutex.Unlock()

	mn.backhaul.Status = status
	mn.backhaul.ParentNode = parentNode
	if status == "connected" {
		mn.backhaul.ConnectTime = time.Now()
	}
}

// 輔助函數
func getRandomFloat(min, max float64) float64 {
	return min + rand.Float64()*(max-min)
}

func getRandomInt(min, max int) int {
	return min + rand.Intn(max-min+1)
}

func getRandomBool() bool {
	return rand.Float64() > 0.5
}
