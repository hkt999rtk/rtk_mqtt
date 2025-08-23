package network

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// TrafficSimulator 網路流量模擬器
type TrafficSimulator struct {
	devices    map[string]*DeviceTraffic
	flows      map[string]*TrafficFlow
	patterns   map[string]*TrafficPattern
	congestion map[string]*CongestionPoint
	running    bool
	mu         sync.RWMutex
	logger     *logrus.Entry
	stats      *TrafficStatistics
	config     *TrafficConfig
}

// DeviceTraffic 設備流量資訊
type DeviceTraffic struct {
	DeviceID       string
	IPAddress      string
	MACAddress     string
	ConnectionType string
	Bandwidth      int     // Mbps
	CurrentRx      float64 // 當前接收速率 Mbps
	CurrentTx      float64 // 當前發送速率 Mbps
	TotalRx        int64   // 總接收量 bytes
	TotalTx        int64   // 總發送量 bytes
	PacketsRx      int64
	PacketsTx      int64
	DroppedPackets int64
	Latency        float64 // ms
	Jitter         float64 // ms
	PacketLoss     float64 // %
	ActiveFlows    []string
	LastUpdate     time.Time
}

// TrafficFlow 流量流
type TrafficFlow struct {
	ID          string
	Source      string // 源設備 ID
	Destination string // 目標設備 ID
	Protocol    string // TCP, UDP, ICMP, etc.
	Port        int
	FlowType    string  // video, audio, web, file, iot, etc.
	Priority    int     // QoS 優先級
	Bandwidth   float64 // Mbps
	Duration    time.Duration
	StartTime   time.Time
	EndTime     time.Time
	BytesSent   int64
	PacketsSent int64
	State       string // active, paused, completed
	Pattern     string // 流量模式
}

// TrafficPattern 流量模式
type TrafficPattern struct {
	ID          string
	Name        string
	Type        string        // constant, burst, wave, random
	BaseRate    float64       // 基礎速率 Mbps
	PeakRate    float64       // 峰值速率 Mbps
	BurstSize   int           // 突發大小 bytes
	Period      time.Duration // 週期
	DutyCycle   float64       // 占空比 0-1
	Variation   float64       // 變化率 0-1
	Description string
}

// CongestionPoint 擁塞點
type CongestionPoint struct {
	ID            string
	Location      string  // 設備或連接 ID
	Severity      float64 // 0-1
	AffectedFlows []string
	StartTime     time.Time
	Duration      time.Duration
	PacketLoss    float64
	AddedLatency  float64
	Cause         string // overflow, collision, interference, etc.
}

// TrafficStatistics 流量統計
type TrafficStatistics struct {
	TotalBandwidth    float64
	UsedBandwidth     float64
	AverageThroughput float64
	PeakThroughput    float64
	TotalPackets      int64
	DroppedPackets    int64
	AverageLatency    float64
	MaxLatency        float64
	ActiveFlows       int
	CongestionEvents  int
	LastUpdate        time.Time
	mu                sync.RWMutex
}

// TrafficConfig 流量配置
type TrafficConfig struct {
	MaxBandwidth        float64       // 最大頻寬 Mbps
	DefaultLatency      float64       // 預設延遲 ms
	PacketLossRate      float64       // 封包遺失率 %
	JitterRange         float64       // 抖動範圍 ms
	CongestionThreshold float64       // 擁塞閾值 0-1
	UpdateInterval      time.Duration // 更新間隔
	EnableQoS           bool          // 啟用 QoS
	EnableShaping       bool          // 啟用流量整形
}

// NewTrafficSimulator 創建新的流量模擬器
func NewTrafficSimulator(config *TrafficConfig) *TrafficSimulator {
	if config == nil {
		config = &TrafficConfig{
			MaxBandwidth:        1000, // 1 Gbps
			DefaultLatency:      1,    // 1ms
			PacketLossRate:      0.01, // 0.01%
			JitterRange:         0.5,  // 0.5ms
			CongestionThreshold: 0.8,  // 80%
			UpdateInterval:      100 * time.Millisecond,
			EnableQoS:           true,
			EnableShaping:       true,
		}
	}

	return &TrafficSimulator{
		devices:    make(map[string]*DeviceTraffic),
		flows:      make(map[string]*TrafficFlow),
		patterns:   make(map[string]*TrafficPattern),
		congestion: make(map[string]*CongestionPoint),
		config:     config,
		logger:     logrus.WithField("component", "traffic_simulator"),
		stats: &TrafficStatistics{
			LastUpdate: time.Now(),
		},
	}
}

// Start 啟動流量模擬器
func (ts *TrafficSimulator) Start(ctx context.Context) error {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	if ts.running {
		return fmt.Errorf("traffic simulator is already running")
	}

	ts.running = true
	ts.logger.Info("Starting traffic simulator")

	// 初始化預設流量模式
	ts.initializeDefaultPatterns()

	// 啟動模擬循環
	go ts.simulationLoop(ctx)
	go ts.flowGenerationLoop(ctx)
	go ts.congestionDetectionLoop(ctx)
	go ts.statisticsUpdateLoop(ctx)

	return nil
}

// Stop 停止流量模擬器
func (ts *TrafficSimulator) Stop() error {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	if !ts.running {
		return fmt.Errorf("traffic simulator is not running")
	}

	ts.running = false
	ts.logger.Info("Stopping traffic simulator")
	return nil
}

// RegisterDevice 註冊設備
func (ts *TrafficSimulator) RegisterDevice(deviceID, ipAddress, macAddress, connectionType string, bandwidth int) {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	ts.devices[deviceID] = &DeviceTraffic{
		DeviceID:       deviceID,
		IPAddress:      ipAddress,
		MACAddress:     macAddress,
		ConnectionType: connectionType,
		Bandwidth:      bandwidth,
		CurrentRx:      0,
		CurrentTx:      0,
		TotalRx:        0,
		TotalTx:        0,
		PacketsRx:      0,
		PacketsTx:      0,
		DroppedPackets: 0,
		Latency:        ts.config.DefaultLatency,
		Jitter:         0,
		PacketLoss:     0,
		ActiveFlows:    make([]string, 0),
		LastUpdate:     time.Now(),
	}

	ts.logger.WithFields(logrus.Fields{
		"device_id":  deviceID,
		"ip_address": ipAddress,
		"bandwidth":  bandwidth,
	}).Info("Device registered for traffic simulation")
}

// CreateFlow 創建流量流
func (ts *TrafficSimulator) CreateFlow(source, destination, protocol, flowType string, bandwidth float64) (*TrafficFlow, error) {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	// 檢查設備是否存在
	srcDevice, srcExists := ts.devices[source]
	dstDevice, dstExists := ts.devices[destination]
	if !srcExists || !dstExists {
		return nil, fmt.Errorf("source or destination device not found")
	}

	// 檢查頻寬限制
	if srcDevice.CurrentTx+bandwidth > float64(srcDevice.Bandwidth) {
		return nil, fmt.Errorf("insufficient bandwidth on source device")
	}
	if dstDevice.CurrentRx+bandwidth > float64(dstDevice.Bandwidth) {
		return nil, fmt.Errorf("insufficient bandwidth on destination device")
	}

	flowID := fmt.Sprintf("flow_%s_%s_%d", source, destination, time.Now().UnixNano())

	flow := &TrafficFlow{
		ID:          flowID,
		Source:      source,
		Destination: destination,
		Protocol:    protocol,
		FlowType:    flowType,
		Bandwidth:   bandwidth,
		StartTime:   time.Now(),
		State:       "active",
		Priority:    ts.getFlowPriority(flowType),
	}

	ts.flows[flowID] = flow

	// 更新設備流量
	srcDevice.CurrentTx += bandwidth
	srcDevice.ActiveFlows = append(srcDevice.ActiveFlows, flowID)
	dstDevice.CurrentRx += bandwidth
	dstDevice.ActiveFlows = append(dstDevice.ActiveFlows, flowID)

	ts.logger.WithFields(logrus.Fields{
		"flow_id":     flowID,
		"source":      source,
		"destination": destination,
		"bandwidth":   bandwidth,
	}).Debug("Traffic flow created")

	return flow, nil
}

// simulationLoop 主模擬循環
func (ts *TrafficSimulator) simulationLoop(ctx context.Context) {
	ticker := time.NewTicker(ts.config.UpdateInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			ts.updateTrafficFlows()
			ts.applyNetworkEffects()
		}
	}
}

// updateTrafficFlows 更新流量流
func (ts *TrafficSimulator) updateTrafficFlows() {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	now := time.Now()
	completedFlows := make([]string, 0)

	for flowID, flow := range ts.flows {
		if flow.State != "active" {
			continue
		}

		// 計算傳輸的數據量
		duration := ts.config.UpdateInterval.Seconds()
		bytesTransferred := int64(flow.Bandwidth * 1000000 * duration / 8) // Mbps to bytes
		packetsTransferred := bytesTransferred / 1500                      // 假設 MTU 1500

		flow.BytesSent += bytesTransferred
		flow.PacketsSent += packetsTransferred

		// 更新設備統計
		if srcDevice, exists := ts.devices[flow.Source]; exists {
			srcDevice.TotalTx += bytesTransferred
			srcDevice.PacketsTx += packetsTransferred
			srcDevice.LastUpdate = now
		}

		if dstDevice, exists := ts.devices[flow.Destination]; exists {
			dstDevice.TotalRx += bytesTransferred
			dstDevice.PacketsRx += packetsTransferred
			dstDevice.LastUpdate = now
		}

		// 檢查流是否應該結束
		if flow.Duration > 0 && time.Since(flow.StartTime) >= flow.Duration {
			flow.State = "completed"
			flow.EndTime = now
			completedFlows = append(completedFlows, flowID)
		}
	}

	// 清理完成的流
	for _, flowID := range completedFlows {
		ts.cleanupFlow(flowID)
	}
}

// applyNetworkEffects 應用網路效應
func (ts *TrafficSimulator) applyNetworkEffects() {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	for _, device := range ts.devices {
		// 計算設備負載
		load := (device.CurrentRx + device.CurrentTx) / float64(device.Bandwidth)

		// 根據負載調整延遲和封包遺失
		if load > ts.config.CongestionThreshold {
			// 擁塞狀態
			congestionFactor := (load - ts.config.CongestionThreshold) / (1 - ts.config.CongestionThreshold)
			device.Latency = ts.config.DefaultLatency * (1 + congestionFactor*5)     // 最多增加 5 倍延遲
			device.PacketLoss = ts.config.PacketLossRate * (1 + congestionFactor*10) // 最多增加 10 倍封包遺失
			device.Jitter = ts.config.JitterRange * (1 + congestionFactor*3)
		} else {
			// 正常狀態
			device.Latency = ts.config.DefaultLatency + rand.Float64()*ts.config.JitterRange
			device.PacketLoss = ts.config.PacketLossRate * rand.Float64()
			device.Jitter = rand.Float64() * ts.config.JitterRange
		}

		// 模擬封包遺失
		if rand.Float64()*100 < device.PacketLoss {
			droppedPackets := int64(float64(device.PacketsRx+device.PacketsTx) * device.PacketLoss / 100)
			device.DroppedPackets += droppedPackets
		}
	}
}

// flowGenerationLoop 流量生成循環
func (ts *TrafficSimulator) flowGenerationLoop(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			ts.generateRandomFlows()
		}
	}
}

// generateRandomFlows 生成隨機流量
func (ts *TrafficSimulator) generateRandomFlows() {
	ts.mu.RLock()
	deviceIDs := make([]string, 0, len(ts.devices))
	for id := range ts.devices {
		deviceIDs = append(deviceIDs, id)
	}
	ts.mu.RUnlock()

	if len(deviceIDs) < 2 {
		return
	}

	// 隨機選擇源和目標
	numFlows := rand.Intn(3) + 1
	for i := 0; i < numFlows; i++ {
		source := deviceIDs[rand.Intn(len(deviceIDs))]
		destination := deviceIDs[rand.Intn(len(deviceIDs))]

		if source == destination {
			continue
		}

		// 隨機選擇流量類型
		flowTypes := []string{"web", "video", "iot", "file", "voice"}
		flowType := flowTypes[rand.Intn(len(flowTypes))]

		// 根據類型設定頻寬
		var bandwidth float64
		switch flowType {
		case "video":
			bandwidth = 5 + rand.Float64()*20 // 5-25 Mbps
		case "voice":
			bandwidth = 0.064 + rand.Float64()*0.256 // 64-320 kbps
		case "web":
			bandwidth = 0.5 + rand.Float64()*2 // 0.5-2.5 Mbps
		case "file":
			bandwidth = 10 + rand.Float64()*50 // 10-60 Mbps
		case "iot":
			bandwidth = 0.01 + rand.Float64()*0.1 // 10-110 kbps
		}

		// 創建流
		_, err := ts.CreateFlow(source, destination, "TCP", flowType, bandwidth)
		if err != nil {
			ts.logger.WithError(err).Debug("Failed to create random flow")
		}
	}
}

// congestionDetectionLoop 擁塞檢測循環
func (ts *TrafficSimulator) congestionDetectionLoop(ctx context.Context) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			ts.detectCongestion()
		}
	}
}

// detectCongestion 檢測擁塞
func (ts *TrafficSimulator) detectCongestion() {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	now := time.Now()

	for deviceID, device := range ts.devices {
		load := (device.CurrentRx + device.CurrentTx) / float64(device.Bandwidth)

		if load > ts.config.CongestionThreshold {
			congestionID := fmt.Sprintf("congestion_%s_%d", deviceID, now.UnixNano())

			// 檢查是否已經存在擁塞點
			existingCongestion := false
			for _, cp := range ts.congestion {
				if cp.Location == deviceID && time.Since(cp.StartTime) < 10*time.Second {
					existingCongestion = true
					cp.Duration = time.Since(cp.StartTime)
					cp.Severity = load
					break
				}
			}

			if !existingCongestion {
				ts.congestion[congestionID] = &CongestionPoint{
					ID:            congestionID,
					Location:      deviceID,
					Severity:      load,
					AffectedFlows: device.ActiveFlows,
					StartTime:     now,
					PacketLoss:    device.PacketLoss,
					AddedLatency:  device.Latency - ts.config.DefaultLatency,
					Cause:         "bandwidth_overload",
				}

				ts.logger.WithFields(logrus.Fields{
					"device_id": deviceID,
					"load":      load,
					"severity":  load,
				}).Warn("Congestion detected")
			}
		}
	}

	// 清理過期的擁塞點
	for id, cp := range ts.congestion {
		if time.Since(cp.StartTime) > 30*time.Second {
			delete(ts.congestion, id)
		}
	}
}

// statisticsUpdateLoop 統計更新循環
func (ts *TrafficSimulator) statisticsUpdateLoop(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			ts.updateStatistics()
		}
	}
}

// updateStatistics 更新統計
func (ts *TrafficSimulator) updateStatistics() {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	ts.stats.mu.Lock()
	defer ts.stats.mu.Unlock()

	// 重置統計
	ts.stats.TotalBandwidth = 0
	ts.stats.UsedBandwidth = 0
	ts.stats.TotalPackets = 0
	ts.stats.DroppedPackets = 0
	ts.stats.ActiveFlows = len(ts.flows)
	ts.stats.CongestionEvents = len(ts.congestion)

	totalLatency := 0.0
	deviceCount := 0

	// 計算統計
	for _, device := range ts.devices {
		ts.stats.TotalBandwidth += float64(device.Bandwidth)
		ts.stats.UsedBandwidth += device.CurrentRx + device.CurrentTx
		ts.stats.TotalPackets += device.PacketsRx + device.PacketsTx
		ts.stats.DroppedPackets += device.DroppedPackets

		totalLatency += device.Latency
		deviceCount++

		if device.Latency > ts.stats.MaxLatency {
			ts.stats.MaxLatency = device.Latency
		}
	}

	if deviceCount > 0 {
		ts.stats.AverageLatency = totalLatency / float64(deviceCount)
	}

	// 計算吞吐量
	if ts.stats.TotalBandwidth > 0 {
		ts.stats.AverageThroughput = ts.stats.UsedBandwidth / ts.stats.TotalBandwidth * 100
	}

	if ts.stats.UsedBandwidth > ts.stats.PeakThroughput {
		ts.stats.PeakThroughput = ts.stats.UsedBandwidth
	}

	ts.stats.LastUpdate = time.Now()
}

// initializeDefaultPatterns 初始化預設流量模式
func (ts *TrafficSimulator) initializeDefaultPatterns() {
	patterns := []TrafficPattern{
		{
			ID:          "constant",
			Name:        "Constant Rate",
			Type:        "constant",
			BaseRate:    10,
			PeakRate:    10,
			Period:      1 * time.Second,
			DutyCycle:   1.0,
			Variation:   0,
			Description: "Constant rate traffic",
		},
		{
			ID:          "burst",
			Name:        "Burst Traffic",
			Type:        "burst",
			BaseRate:    1,
			PeakRate:    100,
			BurstSize:   1024 * 1024, // 1MB
			Period:      5 * time.Second,
			DutyCycle:   0.2,
			Variation:   0.1,
			Description: "Periodic burst traffic",
		},
		{
			ID:          "wave",
			Name:        "Wave Pattern",
			Type:        "wave",
			BaseRate:    5,
			PeakRate:    50,
			Period:      30 * time.Second,
			DutyCycle:   0.5,
			Variation:   0.2,
			Description: "Sinusoidal wave traffic pattern",
		},
		{
			ID:          "random",
			Name:        "Random Traffic",
			Type:        "random",
			BaseRate:    10,
			PeakRate:    30,
			Period:      1 * time.Second,
			DutyCycle:   0.7,
			Variation:   0.5,
			Description: "Random traffic pattern",
		},
	}

	for _, pattern := range patterns {
		ts.patterns[pattern.ID] = &pattern
	}
}

// ApplyTrafficPattern 應用流量模式
func (ts *TrafficSimulator) ApplyTrafficPattern(flowID, patternID string) error {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	flow, flowExists := ts.flows[flowID]
	pattern, patternExists := ts.patterns[patternID]

	if !flowExists {
		return fmt.Errorf("flow %s not found", flowID)
	}
	if !patternExists {
		return fmt.Errorf("pattern %s not found", patternID)
	}

	flow.Pattern = patternID

	// 根據模式調整流量
	switch pattern.Type {
	case "constant":
		flow.Bandwidth = pattern.BaseRate
	case "burst":
		// 模擬突發
		if rand.Float64() < pattern.DutyCycle {
			flow.Bandwidth = pattern.PeakRate
		} else {
			flow.Bandwidth = pattern.BaseRate
		}
	case "wave":
		// 正弦波模式
		elapsed := time.Since(flow.StartTime).Seconds()
		phase := (elapsed / pattern.Period.Seconds()) * 2 * math.Pi
		flow.Bandwidth = pattern.BaseRate + (pattern.PeakRate-pattern.BaseRate)*((math.Sin(phase)+1)/2)
	case "random":
		// 隨機模式
		flow.Bandwidth = pattern.BaseRate + rand.Float64()*(pattern.PeakRate-pattern.BaseRate)
	}

	return nil
}

// cleanupFlow 清理流
func (ts *TrafficSimulator) cleanupFlow(flowID string) {
	flow, exists := ts.flows[flowID]
	if !exists {
		return
	}

	// 更新設備流量
	if srcDevice, exists := ts.devices[flow.Source]; exists {
		srcDevice.CurrentTx -= flow.Bandwidth
		// 移除流 ID
		newFlows := make([]string, 0)
		for _, id := range srcDevice.ActiveFlows {
			if id != flowID {
				newFlows = append(newFlows, id)
			}
		}
		srcDevice.ActiveFlows = newFlows
	}

	if dstDevice, exists := ts.devices[flow.Destination]; exists {
		dstDevice.CurrentRx -= flow.Bandwidth
		// 移除流 ID
		newFlows := make([]string, 0)
		for _, id := range dstDevice.ActiveFlows {
			if id != flowID {
				newFlows = append(newFlows, id)
			}
		}
		dstDevice.ActiveFlows = newFlows
	}

	delete(ts.flows, flowID)
}

// getFlowPriority 獲取流優先級
func (ts *TrafficSimulator) getFlowPriority(flowType string) int {
	priorities := map[string]int{
		"voice":   7, // 最高優先級
		"video":   6,
		"iot":     5,
		"web":     4,
		"file":    3,
		"backup":  2,
		"bulk":    1,
		"default": 0,
	}

	if priority, exists := priorities[flowType]; exists {
		return priority
	}
	return 0
}

// GetStatistics 獲取統計資訊
func (ts *TrafficSimulator) GetStatistics() *TrafficStatistics {
	ts.stats.mu.RLock()
	defer ts.stats.mu.RUnlock()

	// 返回統計副本
	return &TrafficStatistics{
		TotalBandwidth:    ts.stats.TotalBandwidth,
		UsedBandwidth:     ts.stats.UsedBandwidth,
		AverageThroughput: ts.stats.AverageThroughput,
		PeakThroughput:    ts.stats.PeakThroughput,
		TotalPackets:      ts.stats.TotalPackets,
		DroppedPackets:    ts.stats.DroppedPackets,
		AverageLatency:    ts.stats.AverageLatency,
		MaxLatency:        ts.stats.MaxLatency,
		ActiveFlows:       ts.stats.ActiveFlows,
		CongestionEvents:  ts.stats.CongestionEvents,
		LastUpdate:        ts.stats.LastUpdate,
	}
}

// GetDeviceTraffic 獲取設備流量資訊
func (ts *TrafficSimulator) GetDeviceTraffic(deviceID string) (*DeviceTraffic, error) {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	device, exists := ts.devices[deviceID]
	if !exists {
		return nil, fmt.Errorf("device %s not found", deviceID)
	}

	// 返回副本
	return &DeviceTraffic{
		DeviceID:       device.DeviceID,
		IPAddress:      device.IPAddress,
		MACAddress:     device.MACAddress,
		ConnectionType: device.ConnectionType,
		Bandwidth:      device.Bandwidth,
		CurrentRx:      device.CurrentRx,
		CurrentTx:      device.CurrentTx,
		TotalRx:        device.TotalRx,
		TotalTx:        device.TotalTx,
		PacketsRx:      device.PacketsRx,
		PacketsTx:      device.PacketsTx,
		DroppedPackets: device.DroppedPackets,
		Latency:        device.Latency,
		Jitter:         device.Jitter,
		PacketLoss:     device.PacketLoss,
		ActiveFlows:    append([]string{}, device.ActiveFlows...),
		LastUpdate:     device.LastUpdate,
	}, nil
}

// SimulateNetworkEvent 模擬網路事件
func (ts *TrafficSimulator) SimulateNetworkEvent(eventType string, parameters map[string]interface{}) {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	switch eventType {
	case "bandwidth_throttle":
		// 限制頻寬
		if deviceID, ok := parameters["device_id"].(string); ok {
			if device, exists := ts.devices[deviceID]; exists {
				if newBandwidth, ok := parameters["bandwidth"].(int); ok {
					device.Bandwidth = newBandwidth
					ts.logger.WithFields(logrus.Fields{
						"device_id":     deviceID,
						"new_bandwidth": newBandwidth,
					}).Info("Bandwidth throttled")
				}
			}
		}

	case "packet_storm":
		// 封包風暴
		if deviceID, ok := parameters["device_id"].(string); ok {
			if device, exists := ts.devices[deviceID]; exists {
				storm := int64(10000000) // 10M packets
				device.PacketsRx += storm
				device.PacketsTx += storm
				device.DroppedPackets += storm / 10 // 10% drop
				ts.logger.WithField("device_id", deviceID).Warn("Packet storm simulated")
			}
		}

	case "link_failure":
		// 連接失敗
		if deviceID, ok := parameters["device_id"].(string); ok {
			// 停止設備的所有流
			for flowID, flow := range ts.flows {
				if flow.Source == deviceID || flow.Destination == deviceID {
					flow.State = "failed"
					ts.cleanupFlow(flowID)
				}
			}
			ts.logger.WithField("device_id", deviceID).Error("Link failure simulated")
		}

	case "congestion_injection":
		// 注入擁塞
		if location, ok := parameters["location"].(string); ok {
			congestionID := fmt.Sprintf("injected_%s_%d", location, time.Now().UnixNano())
			severity := 0.9
			if s, ok := parameters["severity"].(float64); ok {
				severity = s
			}

			ts.congestion[congestionID] = &CongestionPoint{
				ID:           congestionID,
				Location:     location,
				Severity:     severity,
				StartTime:    time.Now(),
				PacketLoss:   severity * 10,
				AddedLatency: severity * 100,
				Cause:        "injected",
			}
			ts.logger.WithFields(logrus.Fields{
				"location": location,
				"severity": severity,
			}).Warn("Congestion injected")
		}
	}
}
