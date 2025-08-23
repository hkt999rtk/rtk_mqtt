package interaction

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"rtk_simulation/pkg/devices/base"
)

// MeshInteractionManager 管理 Mesh 網路設備間的主動互動
type MeshInteractionManager struct {
	devices      map[string]base.Device
	meshDevices  map[string]MeshDeviceInfo
	interactions []MeshInteraction
	running      bool
	mu           sync.RWMutex
	logger       *logrus.Entry
}

// MeshDeviceInfo Mesh 設備資訊
type MeshDeviceInfo struct {
	DeviceID      string
	MeshRole      string   // master, slave, repeater
	Neighbors     []string // 鄰居設備 ID
	ParentNode    string   // 父節點 ID
	ChildNodes    []string // 子節點 ID
	LastHeartbeat time.Time
	Metrics       MeshMetrics
}

// MeshMetrics Mesh 網路指標
type MeshMetrics struct {
	LinkQuality    float64
	SignalStrength int
	Latency        float64
	PacketLoss     float64
	Throughput     float64
}

// MeshInteraction Mesh 設備間的互動
type MeshInteraction struct {
	Type         string // neighbor_discovery, route_update, health_check, load_balance
	SourceDevice string
	TargetDevice string
	Frequency    time.Duration
	LastExecuted time.Time
	Priority     int
	Data         map[string]interface{}
}

// NewMeshInteractionManager 創建新的 Mesh 互動管理器
func NewMeshInteractionManager() *MeshInteractionManager {
	return &MeshInteractionManager{
		devices:      make(map[string]base.Device),
		meshDevices:  make(map[string]MeshDeviceInfo),
		interactions: make([]MeshInteraction, 0),
		logger:       logrus.WithField("component", "mesh_interaction_manager"),
	}
}

// Start 啟動 Mesh 互動管理器
func (mim *MeshInteractionManager) Start(ctx context.Context) error {
	mim.mu.Lock()
	defer mim.mu.Unlock()

	if mim.running {
		return fmt.Errorf("mesh interaction manager is already running")
	}

	mim.running = true
	mim.logger.Info("Starting mesh interaction manager")

	// 啟動各種互動循環
	go mim.runNeighborDiscovery(ctx)
	go mim.runHealthCheck(ctx)
	go mim.runRouteOptimization(ctx)
	go mim.runLoadBalancing(ctx)
	go mim.runMeshSynchronization(ctx)

	return nil
}

// Stop 停止 Mesh 互動管理器
func (mim *MeshInteractionManager) Stop() error {
	mim.mu.Lock()
	defer mim.mu.Unlock()

	if !mim.running {
		return fmt.Errorf("mesh interaction manager is not running")
	}

	mim.running = false
	mim.logger.Info("Stopping mesh interaction manager")
	return nil
}

// RegisterMeshDevice 註冊 Mesh 設備
func (mim *MeshInteractionManager) RegisterMeshDevice(deviceID string, device base.Device, meshRole string) {
	mim.mu.Lock()
	defer mim.mu.Unlock()

	mim.devices[deviceID] = device
	mim.meshDevices[deviceID] = MeshDeviceInfo{
		DeviceID:      deviceID,
		MeshRole:      meshRole,
		Neighbors:     make([]string, 0),
		ChildNodes:    make([]string, 0),
		LastHeartbeat: time.Now(),
		Metrics: MeshMetrics{
			LinkQuality:    100,
			SignalStrength: -30,
			Latency:        1,
			PacketLoss:     0,
			Throughput:     500,
		},
	}

	mim.logger.WithFields(logrus.Fields{
		"device_id": deviceID,
		"mesh_role": meshRole,
	}).Info("Mesh device registered")
}

// runNeighborDiscovery 運行鄰居發現
func (mim *MeshInteractionManager) runNeighborDiscovery(ctx context.Context) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			mim.performNeighborDiscovery()
		}
	}
}

// performNeighborDiscovery 執行鄰居發現
func (mim *MeshInteractionManager) performNeighborDiscovery() {
	mim.mu.RLock()
	meshDevices := make([]string, 0, len(mim.meshDevices))
	for id := range mim.meshDevices {
		meshDevices = append(meshDevices, id)
	}
	mim.mu.RUnlock()

	// 模擬鄰居發現過程
	for _, deviceID := range meshDevices {
		mim.discoverNeighborsForDevice(deviceID)
	}
}

// discoverNeighborsForDevice 為特定設備發現鄰居
func (mim *MeshInteractionManager) discoverNeighborsForDevice(deviceID string) {
	mim.mu.Lock()
	defer mim.mu.Unlock()

	deviceInfo, exists := mim.meshDevices[deviceID]
	if !exists {
		return
	}

	// 清空現有鄰居列表
	deviceInfo.Neighbors = make([]string, 0)

	// 模擬發現鄰居設備
	for otherID, otherInfo := range mim.meshDevices {
		if otherID == deviceID {
			continue
		}

		// 根據角色和隨機因素決定是否成為鄰居
		if mim.canBeNeighbors(deviceInfo, otherInfo) {
			deviceInfo.Neighbors = append(deviceInfo.Neighbors, otherID)

			// 發送鄰居發現通知
			mim.sendNeighborDiscoveryNotification(deviceID, otherID)
		}
	}

	mim.meshDevices[deviceID] = deviceInfo

	mim.logger.WithFields(logrus.Fields{
		"device_id":      deviceID,
		"neighbor_count": len(deviceInfo.Neighbors),
	}).Debug("Neighbor discovery completed")
}

// canBeNeighbors 判斷兩個設備是否可以成為鄰居
func (mim *MeshInteractionManager) canBeNeighbors(device1, device2 MeshDeviceInfo) bool {
	// Master 節點可以連接到任何設備
	if device1.MeshRole == "master" || device2.MeshRole == "master" {
		return rand.Float64() > 0.3
	}

	// Slave 和 Repeater 之間的連接
	if (device1.MeshRole == "slave" && device2.MeshRole == "repeater") ||
		(device1.MeshRole == "repeater" && device2.MeshRole == "slave") {
		return rand.Float64() > 0.4
	}

	// Repeater 之間的連接
	if device1.MeshRole == "repeater" && device2.MeshRole == "repeater" {
		return rand.Float64() > 0.5
	}

	return rand.Float64() > 0.6
}

// sendNeighborDiscoveryNotification 發送鄰居發現通知
func (mim *MeshInteractionManager) sendNeighborDiscoveryNotification(sourceID, targetID string) {
	sourceDevice, sourceExists := mim.devices[sourceID]
	targetDevice, targetExists := mim.devices[targetID]

	if !sourceExists || !targetExists {
		return
	}

	// 創建鄰居發現命令
	cmd := base.Command{
		ID:   fmt.Sprintf("neighbor_discovery_%d", time.Now().UnixNano()),
		Type: "neighbor_discovered",
		Parameters: map[string]interface{}{
			"neighbor_id": targetID,
			"timestamp":   time.Now(),
			"metrics":     mim.meshDevices[targetID].Metrics,
		},
		Timeout: 5 * time.Second,
	}

	// 雙向通知
	go sourceDevice.HandleCommand(cmd)

	cmd.Parameters["neighbor_id"] = sourceID
	cmd.Parameters["metrics"] = mim.meshDevices[sourceID].Metrics
	go targetDevice.HandleCommand(cmd)
}

// runHealthCheck 運行健康檢查
func (mim *MeshInteractionManager) runHealthCheck(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			mim.performHealthCheck()
		}
	}
}

// performHealthCheck 執行健康檢查
func (mim *MeshInteractionManager) performHealthCheck() {
	mim.mu.RLock()
	meshDevices := make(map[string]MeshDeviceInfo)
	for id, info := range mim.meshDevices {
		meshDevices[id] = info
	}
	mim.mu.RUnlock()

	for deviceID, deviceInfo := range meshDevices {
		// 檢查每個鄰居的健康狀態
		for _, neighborID := range deviceInfo.Neighbors {
			mim.checkNeighborHealth(deviceID, neighborID)
		}

		// 更新設備的 heartbeat
		mim.mu.Lock()
		if info, exists := mim.meshDevices[deviceID]; exists {
			info.LastHeartbeat = time.Now()
			mim.meshDevices[deviceID] = info
		}
		mim.mu.Unlock()
	}
}

// checkNeighborHealth 檢查鄰居健康狀態
func (mim *MeshInteractionManager) checkNeighborHealth(deviceID, neighborID string) {
	mim.mu.RLock()
	device, deviceExists := mim.devices[deviceID]
	neighbor, neighborExists := mim.devices[neighborID]
	mim.mu.RUnlock()

	if !deviceExists || !neighborExists {
		return
	}

	// 發送健康檢查請求
	healthCmd := base.Command{
		ID:   fmt.Sprintf("health_check_%d", time.Now().UnixNano()),
		Type: "health_check",
		Parameters: map[string]interface{}{
			"requester_id": deviceID,
			"timestamp":    time.Now(),
		},
		Timeout: 5 * time.Second,
	}

	// 發送健康檢查
	go func() {
		if err := neighbor.HandleCommand(healthCmd); err == nil {
			// 更新鄰居健康指標
			mim.updateNeighborMetrics(deviceID, neighborID, true)
		} else {
			mim.updateNeighborMetrics(deviceID, neighborID, false)
		}
	}()

	// 模擬雙向健康檢查
	healthCmd.Parameters["requester_id"] = neighborID
	go device.HandleCommand(healthCmd)
}

// updateNeighborMetrics 更新鄰居指標
func (mim *MeshInteractionManager) updateNeighborMetrics(deviceID, neighborID string, healthy bool) {
	mim.mu.Lock()
	defer mim.mu.Unlock()

	if info, exists := mim.meshDevices[deviceID]; exists {
		if healthy {
			info.Metrics.LinkQuality = min(100, info.Metrics.LinkQuality+rand.Float64()*5)
			info.Metrics.PacketLoss = max(0, info.Metrics.PacketLoss-rand.Float64()*0.5)
		} else {
			info.Metrics.LinkQuality = max(0, info.Metrics.LinkQuality-rand.Float64()*10)
			info.Metrics.PacketLoss = min(10, info.Metrics.PacketLoss+rand.Float64()*2)
		}

		info.Metrics.Latency = 1 + rand.Float64()*10
		info.Metrics.SignalStrength = -30 - rand.Intn(40)
		info.Metrics.Throughput = 100 + rand.Float64()*400

		mim.meshDevices[deviceID] = info
	}
}

// runRouteOptimization 運行路由優化
func (mim *MeshInteractionManager) runRouteOptimization(ctx context.Context) {
	ticker := time.NewTicker(45 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			mim.optimizeRoutes()
		}
	}
}

// optimizeRoutes 優化路由
func (mim *MeshInteractionManager) optimizeRoutes() {
	mim.mu.RLock()
	meshDevices := make(map[string]MeshDeviceInfo)
	for id, info := range mim.meshDevices {
		meshDevices[id] = info
	}
	mim.mu.RUnlock()

	// 找出所有 master 節點
	masterNodes := make([]string, 0)
	for id, info := range meshDevices {
		if info.MeshRole == "master" {
			masterNodes = append(masterNodes, id)
		}
	}

	// 為每個非 master 節點優化到 master 的路由
	for deviceID, deviceInfo := range meshDevices {
		if deviceInfo.MeshRole != "master" && len(masterNodes) > 0 {
			mim.optimizeRouteToMaster(deviceID, masterNodes[0])
		}
	}
}

// optimizeRouteToMaster 優化到主節點的路由
func (mim *MeshInteractionManager) optimizeRouteToMaster(deviceID, masterID string) {
	mim.mu.RLock()
	device, exists := mim.devices[deviceID]
	master, masterExists := mim.devices[masterID]
	mim.mu.RUnlock()

	if !exists || !masterExists {
		return
	}

	// 發送路由優化命令
	routeCmd := base.Command{
		ID:   fmt.Sprintf("route_optimize_%d", time.Now().UnixNano()),
		Type: "optimize_route",
		Parameters: map[string]interface{}{
			"target":       masterID,
			"optimization": "shortest_path",
			"timestamp":    time.Now(),
		},
		Timeout: 10 * time.Second,
	}

	go device.HandleCommand(routeCmd)

	// 通知 master 節點
	notifyCmd := base.Command{
		ID:   fmt.Sprintf("route_notify_%d", time.Now().UnixNano()),
		Type: "route_updated",
		Parameters: map[string]interface{}{
			"source":    deviceID,
			"timestamp": time.Now(),
		},
		Timeout: 5 * time.Second,
	}

	go master.HandleCommand(notifyCmd)
}

// runLoadBalancing 運行負載均衡
func (mim *MeshInteractionManager) runLoadBalancing(ctx context.Context) {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			mim.performLoadBalancing()
		}
	}
}

// performLoadBalancing 執行負載均衡
func (mim *MeshInteractionManager) performLoadBalancing() {
	mim.mu.RLock()
	meshDevices := make(map[string]MeshDeviceInfo)
	for id, info := range mim.meshDevices {
		meshDevices[id] = info
	}
	mim.mu.RUnlock()

	// 識別高負載和低負載設備
	highLoadDevices := make([]string, 0)
	lowLoadDevices := make([]string, 0)

	for id, info := range meshDevices {
		if info.Metrics.Throughput > 400 {
			highLoadDevices = append(highLoadDevices, id)
		} else if info.Metrics.Throughput < 200 {
			lowLoadDevices = append(lowLoadDevices, id)
		}
	}

	// 執行負載均衡
	if len(highLoadDevices) > 0 && len(lowLoadDevices) > 0 {
		mim.rebalanceLoad(highLoadDevices, lowLoadDevices)
	}
}

// rebalanceLoad 重新平衡負載
func (mim *MeshInteractionManager) rebalanceLoad(highLoad, lowLoad []string) {
	for _, highLoadID := range highLoad {
		if len(lowLoad) == 0 {
			break
		}

		// 選擇一個低負載設備
		targetID := lowLoad[rand.Intn(len(lowLoad))]

		mim.mu.RLock()
		sourceDevice, sourceExists := mim.devices[highLoadID]
		targetDevice, targetExists := mim.devices[targetID]
		mim.mu.RUnlock()

		if !sourceExists || !targetExists {
			continue
		}

		// 發送負載轉移命令
		balanceCmd := base.Command{
			ID:   fmt.Sprintf("load_balance_%d", time.Now().UnixNano()),
			Type: "transfer_load",
			Parameters: map[string]interface{}{
				"source":      highLoadID,
				"target":      targetID,
				"load_amount": rand.Intn(50) + 50,
				"timestamp":   time.Now(),
			},
			Timeout: 10 * time.Second,
		}

		go sourceDevice.HandleCommand(balanceCmd)
		go targetDevice.HandleCommand(balanceCmd)

		mim.logger.WithFields(logrus.Fields{
			"source": highLoadID,
			"target": targetID,
		}).Info("Load balancing performed")
	}
}

// runMeshSynchronization 運行 Mesh 同步
func (mim *MeshInteractionManager) runMeshSynchronization(ctx context.Context) {
	ticker := time.NewTicker(20 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			mim.synchronizeMeshNetwork()
		}
	}
}

// synchronizeMeshNetwork 同步 Mesh 網路
func (mim *MeshInteractionManager) synchronizeMeshNetwork() {
	mim.mu.RLock()
	meshDevices := make(map[string]MeshDeviceInfo)
	for id, info := range mim.meshDevices {
		meshDevices[id] = info
	}
	mim.mu.RUnlock()

	// 構建網路拓撲快照
	topology := mim.buildTopologySnapshot()

	// 向所有設備廣播拓撲更新
	for deviceID := range meshDevices {
		mim.broadcastTopologyUpdate(deviceID, topology)
	}
}

// buildTopologySnapshot 構建拓撲快照
func (mim *MeshInteractionManager) buildTopologySnapshot() map[string]interface{} {
	mim.mu.RLock()
	defer mim.mu.RUnlock()

	topology := map[string]interface{}{
		"timestamp":   time.Now(),
		"node_count":  len(mim.meshDevices),
		"nodes":       make(map[string]interface{}),
		"connections": make([]map[string]string, 0),
	}

	// 添加節點資訊
	for id, info := range mim.meshDevices {
		topology["nodes"].(map[string]interface{})[id] = map[string]interface{}{
			"role":      info.MeshRole,
			"neighbors": info.Neighbors,
			"metrics":   info.Metrics,
		}

		// 添加連接資訊
		for _, neighborID := range info.Neighbors {
			connection := map[string]string{
				"from": id,
				"to":   neighborID,
			}
			topology["connections"] = append(topology["connections"].([]map[string]string), connection)
		}
	}

	return topology
}

// broadcastTopologyUpdate 廣播拓撲更新
func (mim *MeshInteractionManager) broadcastTopologyUpdate(deviceID string, topology map[string]interface{}) {
	mim.mu.RLock()
	device, exists := mim.devices[deviceID]
	mim.mu.RUnlock()

	if !exists {
		return
	}

	// 發送拓撲更新命令
	updateCmd := base.Command{
		ID:   fmt.Sprintf("topology_update_%d", time.Now().UnixNano()),
		Type: "topology_sync",
		Parameters: map[string]interface{}{
			"topology":  topology,
			"timestamp": time.Now(),
		},
		Timeout: 5 * time.Second,
	}

	go device.HandleCommand(updateCmd)
}

// TriggerMeshEvent 觸發 Mesh 事件
func (mim *MeshInteractionManager) TriggerMeshEvent(eventType string, sourceDevice string, data map[string]interface{}) {
	mim.mu.RLock()
	deviceInfo, exists := mim.meshDevices[sourceDevice]
	mim.mu.RUnlock()

	if !exists {
		return
	}

	switch eventType {
	case "node_failure":
		mim.handleNodeFailure(sourceDevice, deviceInfo)
	case "link_degradation":
		mim.handleLinkDegradation(sourceDevice, data)
	case "congestion":
		mim.handleCongestion(sourceDevice, data)
	case "security_threat":
		mim.handleSecurityThreat(sourceDevice, data)
	}
}

// handleNodeFailure 處理節點故障
func (mim *MeshInteractionManager) handleNodeFailure(failedNode string, deviceInfo MeshDeviceInfo) {
	mim.logger.WithField("failed_node", failedNode).Warn("Handling node failure")

	// 通知所有鄰居節點
	for _, neighborID := range deviceInfo.Neighbors {
		mim.mu.RLock()
		neighbor, exists := mim.devices[neighborID]
		mim.mu.RUnlock()

		if exists {
			failureCmd := base.Command{
				ID:   fmt.Sprintf("node_failure_%d", time.Now().UnixNano()),
				Type: "node_failed",
				Parameters: map[string]interface{}{
					"failed_node": failedNode,
					"timestamp":   time.Now(),
					"action":      "reroute",
				},
				Timeout: 5 * time.Second,
			}
			go neighbor.HandleCommand(failureCmd)
		}
	}

	// 觸發自愈過程
	mim.initiateSelfHealing(failedNode)
}

// handleLinkDegradation 處理鏈路降級
func (mim *MeshInteractionManager) handleLinkDegradation(deviceID string, data map[string]interface{}) {
	mim.logger.WithFields(logrus.Fields{
		"device": deviceID,
		"data":   data,
	}).Warn("Handling link degradation")

	// 更新鏈路質量指標
	mim.mu.Lock()
	if info, exists := mim.meshDevices[deviceID]; exists {
		info.Metrics.LinkQuality = max(0, info.Metrics.LinkQuality-20)
		info.Metrics.PacketLoss = min(10, info.Metrics.PacketLoss+2)
		mim.meshDevices[deviceID] = info
	}
	mim.mu.Unlock()

	// 尋找替代路由
	mim.findAlternativeRoutes(deviceID)
}

// handleCongestion 處理擁塞
func (mim *MeshInteractionManager) handleCongestion(deviceID string, data map[string]interface{}) {
	mim.logger.WithFields(logrus.Fields{
		"device": deviceID,
		"data":   data,
	}).Info("Handling network congestion")

	// 觸發負載均衡
	mim.performLoadBalancing()
}

// handleSecurityThreat 處理安全威脅
func (mim *MeshInteractionManager) handleSecurityThreat(deviceID string, data map[string]interface{}) {
	mim.logger.WithFields(logrus.Fields{
		"device": deviceID,
		"threat": data,
	}).Error("Security threat detected")

	// 隔離受影響的節點
	mim.isolateNode(deviceID)

	// 通知所有設備
	mim.broadcastSecurityAlert(deviceID, data)
}

// initiateSelfHealing 啟動自愈過程
func (mim *MeshInteractionManager) initiateSelfHealing(failedNode string) {
	mim.logger.WithField("failed_node", failedNode).Info("Initiating self-healing process")

	// 重新計算網路拓撲
	mim.performNeighborDiscovery()

	// 優化路由
	mim.optimizeRoutes()
}

// findAlternativeRoutes 尋找替代路由
func (mim *MeshInteractionManager) findAlternativeRoutes(deviceID string) {
	mim.logger.WithField("device", deviceID).Info("Finding alternative routes")

	// 觸發路由重新計算
	mim.optimizeRoutes()
}

// isolateNode 隔離節點
func (mim *MeshInteractionManager) isolateNode(deviceID string) {
	mim.mu.Lock()
	defer mim.mu.Unlock()

	if info, exists := mim.meshDevices[deviceID]; exists {
		// 清空鄰居列表
		info.Neighbors = make([]string, 0)
		mim.meshDevices[deviceID] = info

		mim.logger.WithField("device", deviceID).Warn("Node isolated due to security threat")
	}
}

// broadcastSecurityAlert 廣播安全警報
func (mim *MeshInteractionManager) broadcastSecurityAlert(threatSource string, threatData map[string]interface{}) {
	mim.mu.RLock()
	devices := make(map[string]base.Device)
	for id, device := range mim.devices {
		devices[id] = device
	}
	mim.mu.RUnlock()

	alertCmd := base.Command{
		ID:   fmt.Sprintf("security_alert_%d", time.Now().UnixNano()),
		Type: "security_alert",
		Parameters: map[string]interface{}{
			"threat_source": threatSource,
			"threat_data":   threatData,
			"timestamp":     time.Now(),
			"action":        "heighten_security",
		},
		Timeout: 5 * time.Second,
	}

	for _, device := range devices {
		go device.HandleCommand(alertCmd)
	}
}

// GetMeshStatistics 獲取 Mesh 網路統計
func (mim *MeshInteractionManager) GetMeshStatistics() map[string]interface{} {
	mim.mu.RLock()
	defer mim.mu.RUnlock()

	stats := map[string]interface{}{
		"total_mesh_devices":   len(mim.meshDevices),
		"total_connections":    0,
		"average_link_quality": 0.0,
		"average_latency":      0.0,
		"roles": map[string]int{
			"master":   0,
			"slave":    0,
			"repeater": 0,
		},
	}

	totalLinkQuality := 0.0
	totalLatency := 0.0
	connectionCount := 0

	for _, info := range mim.meshDevices {
		// 計算連接數
		connectionCount += len(info.Neighbors)

		// 累加指標
		totalLinkQuality += info.Metrics.LinkQuality
		totalLatency += info.Metrics.Latency

		// 統計角色
		if roleCount, exists := stats["roles"].(map[string]int)[info.MeshRole]; exists {
			stats["roles"].(map[string]int)[info.MeshRole] = roleCount + 1
		}
	}

	stats["total_connections"] = connectionCount / 2 // 除以2因為是雙向連接

	if len(mim.meshDevices) > 0 {
		stats["average_link_quality"] = totalLinkQuality / float64(len(mim.meshDevices))
		stats["average_latency"] = totalLatency / float64(len(mim.meshDevices))
	}

	return stats
}

// Helper functions
func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
