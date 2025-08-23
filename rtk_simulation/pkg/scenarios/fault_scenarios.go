package scenarios

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"rtk_simulation/pkg/devices/base"
)

// FaultScenarioManager 故障情境管理器
type FaultScenarioManager struct {
	scenarios       map[string]*FaultScenario
	activeScenarios map[string]*ActiveScenario
	devices         map[string]base.Device
	eventHistory    []FaultEvent
	running         bool
	mu              sync.RWMutex
	logger          *logrus.Entry
	config          *FaultConfig
}

// FaultScenario 故障情境定義
type FaultScenario struct {
	ID               string
	Name             string
	Type             string // device_failure, network_outage, power_failure, etc.
	Description      string
	Severity         string                 // low, medium, high, critical
	Probability      float64                // 0-1 發生機率
	Duration         time.Duration          // 故障持續時間
	RecoveryTime     time.Duration          // 恢復時間
	AffectedDevices  []string               // 受影響的設備 ID 或類型
	Conditions       []TriggerCondition     // 觸發條件
	Effects          []FaultEffect          // 故障效果
	RecoveryActions  []RecoveryAction       // 恢復動作
	CascadeScenarios []string               // 級聯故障情境
	Parameters       map[string]interface{} // 額外參數
}

// ActiveScenario 活動中的故障情境
type ActiveScenario struct {
	ScenarioID       string
	InstanceID       string
	StartTime        time.Time
	EndTime          time.Time
	State            string // active, recovering, resolved
	AffectedDevices  []string
	Impact           *ImpactAssessment
	RecoveryAttempts int
	LastUpdate       time.Time
}

// FaultEvent 故障事件
type FaultEvent struct {
	ID          string
	ScenarioID  string
	Type        string
	Severity    string
	Timestamp   time.Time
	DeviceID    string
	Description string
	Data        map[string]interface{}
}

// TriggerCondition 觸發條件
type TriggerCondition struct {
	Type       string // time, load, temperature, error_rate, etc.
	Operator   string // gt, lt, eq, ne, ge, le
	Threshold  interface{}
	Duration   time.Duration // 條件持續時間
	Parameters map[string]interface{}
}

// FaultEffect 故障效果
type FaultEffect struct {
	Type       string  // performance_degradation, service_unavailable, data_loss, etc.
	Target     string  // device, service, network
	Severity   float64 // 0-1
	Parameters map[string]interface{}
}

// RecoveryAction 恢復動作
type RecoveryAction struct {
	Type       string // restart, failover, reconfigure, replace
	Target     string
	Delay      time.Duration
	Priority   int
	Parameters map[string]interface{}
}

// ImpactAssessment 影響評估
type ImpactAssessment struct {
	TotalDevices        int
	AffectedDevices     int
	ServiceAvailability float64 // 0-100%
	DataLoss            bool
	EstimatedDowntime   time.Duration
	BusinessImpact      string
	RecoveryTime        time.Duration
}

// FaultConfig 故障配置
type FaultConfig struct {
	EnableAutoRecovery    bool
	MaxRecoveryAttempts   int
	ScenarioCheckInterval time.Duration
	EventRetention        time.Duration
	CascadeEnabled        bool
	RandomFailureRate     float64 // 隨機故障率
}

// PredefinedScenarios 預定義故障情境
var PredefinedScenarios = []FaultScenario{
	{
		ID:           "power_outage",
		Name:         "Power Outage",
		Type:         "power_failure",
		Description:  "Complete power failure affecting all devices",
		Severity:     "critical",
		Probability:  0.01,
		Duration:     30 * time.Minute,
		RecoveryTime: 5 * time.Minute,
		Effects: []FaultEffect{
			{Type: "service_unavailable", Target: "all", Severity: 1.0},
			{Type: "data_loss", Target: "volatile", Severity: 0.8},
		},
		RecoveryActions: []RecoveryAction{
			{Type: "wait_power", Delay: 0},
			{Type: "restart", Target: "all", Delay: 30 * time.Second},
			{Type: "verify_services", Delay: 1 * time.Minute},
		},
	},
	{
		ID:              "router_failure",
		Name:            "Router Hardware Failure",
		Type:            "device_failure",
		Description:     "Main router hardware failure",
		Severity:        "high",
		Probability:     0.05,
		Duration:        2 * time.Hour,
		RecoveryTime:    30 * time.Minute,
		AffectedDevices: []string{"router"},
		Effects: []FaultEffect{
			{Type: "network_isolation", Target: "all", Severity: 1.0},
			{Type: "service_unavailable", Target: "internet", Severity: 1.0},
		},
		RecoveryActions: []RecoveryAction{
			{Type: "failover", Target: "backup_router", Delay: 1 * time.Minute},
			{Type: "reconfigure", Target: "network", Delay: 5 * time.Minute},
		},
		CascadeScenarios: []string{"client_disconnect", "iot_offline"},
	},
	{
		ID:           "ddos_attack",
		Name:         "DDoS Attack",
		Type:         "security_incident",
		Description:  "Distributed Denial of Service attack",
		Severity:     "high",
		Probability:  0.02,
		Duration:     1 * time.Hour,
		RecoveryTime: 15 * time.Minute,
		Effects: []FaultEffect{
			{Type: "performance_degradation", Target: "network", Severity: 0.9},
			{Type: "service_unavailable", Target: "external", Severity: 0.8},
		},
		RecoveryActions: []RecoveryAction{
			{Type: "enable_mitigation", Delay: 30 * time.Second},
			{Type: "block_sources", Delay: 1 * time.Minute},
			{Type: "scale_resources", Delay: 5 * time.Minute},
		},
	},
	{
		ID:           "memory_leak",
		Name:         "Memory Leak",
		Type:         "software_bug",
		Description:  "Memory leak causing performance degradation",
		Severity:     "medium",
		Probability:  0.1,
		Duration:     4 * time.Hour,
		RecoveryTime: 10 * time.Minute,
		Effects: []FaultEffect{
			{Type: "performance_degradation", Target: "device", Severity: 0.6},
			{Type: "service_degradation", Target: "application", Severity: 0.7},
		},
		RecoveryActions: []RecoveryAction{
			{Type: "restart", Target: "application", Delay: 0},
			{Type: "clear_cache", Delay: 30 * time.Second},
		},
	},
	{
		ID:           "firmware_corruption",
		Name:         "Firmware Corruption",
		Type:         "software_failure",
		Description:  "Device firmware corruption",
		Severity:     "high",
		Probability:  0.01,
		Duration:     3 * time.Hour,
		RecoveryTime: 1 * time.Hour,
		Effects: []FaultEffect{
			{Type: "device_malfunction", Target: "device", Severity: 1.0},
			{Type: "configuration_loss", Target: "device", Severity: 0.9},
		},
		RecoveryActions: []RecoveryAction{
			{Type: "firmware_recovery", Delay: 5 * time.Minute},
			{Type: "factory_reset", Delay: 15 * time.Minute},
			{Type: "reconfigure", Target: "device", Delay: 30 * time.Minute},
		},
	},
	{
		ID:           "overheating",
		Name:         "Device Overheating",
		Type:         "environmental",
		Description:  "Device overheating due to high temperature",
		Severity:     "medium",
		Probability:  0.05,
		Duration:     1 * time.Hour,
		RecoveryTime: 20 * time.Minute,
		Conditions: []TriggerCondition{
			{Type: "temperature", Operator: "gt", Threshold: 80.0},
		},
		Effects: []FaultEffect{
			{Type: "thermal_throttling", Target: "device", Severity: 0.5},
			{Type: "performance_degradation", Target: "device", Severity: 0.6},
		},
		RecoveryActions: []RecoveryAction{
			{Type: "reduce_load", Delay: 0},
			{Type: "enable_cooling", Delay: 30 * time.Second},
			{Type: "shutdown_if_critical", Delay: 5 * time.Minute},
		},
	},
	{
		ID:           "network_congestion",
		Name:         "Network Congestion",
		Type:         "network_issue",
		Description:  "Severe network congestion",
		Severity:     "medium",
		Probability:  0.15,
		Duration:     30 * time.Minute,
		RecoveryTime: 10 * time.Minute,
		Conditions: []TriggerCondition{
			{Type: "bandwidth_usage", Operator: "gt", Threshold: 0.9},
		},
		Effects: []FaultEffect{
			{Type: "high_latency", Target: "network", Severity: 0.7},
			{Type: "packet_loss", Target: "network", Severity: 0.5},
		},
		RecoveryActions: []RecoveryAction{
			{Type: "traffic_shaping", Delay: 0},
			{Type: "qos_adjustment", Delay: 1 * time.Minute},
		},
	},
	{
		ID:           "database_corruption",
		Name:         "Database Corruption",
		Type:         "data_issue",
		Description:  "Configuration database corruption",
		Severity:     "high",
		Probability:  0.02,
		Duration:     2 * time.Hour,
		RecoveryTime: 45 * time.Minute,
		Effects: []FaultEffect{
			{Type: "configuration_loss", Target: "system", Severity: 0.9},
			{Type: "service_unavailable", Target: "management", Severity: 0.8},
		},
		RecoveryActions: []RecoveryAction{
			{Type: "restore_backup", Delay: 10 * time.Minute},
			{Type: "rebuild_database", Delay: 30 * time.Minute},
		},
	},
}

// NewFaultScenarioManager 創建新的故障情境管理器
func NewFaultScenarioManager(config *FaultConfig) *FaultScenarioManager {
	if config == nil {
		config = &FaultConfig{
			EnableAutoRecovery:    true,
			MaxRecoveryAttempts:   3,
			ScenarioCheckInterval: 10 * time.Second,
			EventRetention:        24 * time.Hour,
			CascadeEnabled:        true,
			RandomFailureRate:     0.001,
		}
	}

	fsm := &FaultScenarioManager{
		scenarios:       make(map[string]*FaultScenario),
		activeScenarios: make(map[string]*ActiveScenario),
		devices:         make(map[string]base.Device),
		eventHistory:    make([]FaultEvent, 0),
		config:          config,
		logger:          logrus.WithField("component", "fault_scenario_manager"),
	}

	// 載入預定義情境
	fsm.loadPredefinedScenarios()

	return fsm
}

// loadPredefinedScenarios 載入預定義情境
func (fsm *FaultScenarioManager) loadPredefinedScenarios() {
	for _, scenario := range PredefinedScenarios {
		s := scenario // 避免迴圈變數問題
		fsm.scenarios[s.ID] = &s
	}
	fsm.logger.Infof("Loaded %d predefined fault scenarios", len(PredefinedScenarios))
}

// Start 啟動故障情境管理器
func (fsm *FaultScenarioManager) Start(ctx context.Context) error {
	fsm.mu.Lock()
	defer fsm.mu.Unlock()

	if fsm.running {
		return fmt.Errorf("fault scenario manager is already running")
	}

	fsm.running = true
	fsm.logger.Info("Starting fault scenario manager")

	// 啟動管理循環
	go fsm.scenarioCheckLoop(ctx)
	go fsm.recoveryLoop(ctx)
	go fsm.randomFailureLoop(ctx)
	go fsm.eventCleanupLoop(ctx)

	return nil
}

// Stop 停止故障情境管理器
func (fsm *FaultScenarioManager) Stop() error {
	fsm.mu.Lock()
	defer fsm.mu.Unlock()

	if !fsm.running {
		return fmt.Errorf("fault scenario manager is not running")
	}

	fsm.running = false
	fsm.logger.Info("Stopping fault scenario manager")
	return nil
}

// RegisterDevice 註冊設備
func (fsm *FaultScenarioManager) RegisterDevice(deviceID string, device base.Device) {
	fsm.mu.Lock()
	defer fsm.mu.Unlock()

	fsm.devices[deviceID] = device
	fsm.logger.WithField("device_id", deviceID).Debug("Device registered for fault scenarios")
}

// TriggerScenario 觸發故障情境
func (fsm *FaultScenarioManager) TriggerScenario(scenarioID string, targetDevices []string) error {
	fsm.mu.Lock()
	defer fsm.mu.Unlock()

	scenario, exists := fsm.scenarios[scenarioID]
	if !exists {
		return fmt.Errorf("scenario %s not found", scenarioID)
	}

	// 創建活動情境實例
	instanceID := fmt.Sprintf("%s_%d", scenarioID, time.Now().UnixNano())

	// 確定受影響的設備
	affectedDevices := fsm.determineAffectedDevices(scenario, targetDevices)

	// 評估影響
	impact := fsm.assessImpact(scenario, affectedDevices)

	activeScenario := &ActiveScenario{
		ScenarioID:       scenarioID,
		InstanceID:       instanceID,
		StartTime:        time.Now(),
		EndTime:          time.Now().Add(scenario.Duration),
		State:            "active",
		AffectedDevices:  affectedDevices,
		Impact:           impact,
		RecoveryAttempts: 0,
		LastUpdate:       time.Now(),
	}

	fsm.activeScenarios[instanceID] = activeScenario

	// 應用故障效果
	fsm.applyFaultEffects(scenario, affectedDevices)

	// 記錄事件
	fsm.recordEvent(FaultEvent{
		ID:          fmt.Sprintf("event_%d", time.Now().UnixNano()),
		ScenarioID:  scenarioID,
		Type:        "scenario_triggered",
		Severity:    scenario.Severity,
		Timestamp:   time.Now(),
		Description: fmt.Sprintf("Fault scenario '%s' triggered", scenario.Name),
		Data: map[string]interface{}{
			"instance_id":      instanceID,
			"affected_devices": len(affectedDevices),
			"duration":         scenario.Duration.String(),
		},
	})

	fsm.logger.WithFields(logrus.Fields{
		"scenario_id":      scenarioID,
		"instance_id":      instanceID,
		"affected_devices": len(affectedDevices),
	}).Warn("Fault scenario triggered")

	// 觸發級聯故障
	if fsm.config.CascadeEnabled && len(scenario.CascadeScenarios) > 0 {
		go fsm.triggerCascadeScenarios(scenario.CascadeScenarios, affectedDevices)
	}

	return nil
}

// scenarioCheckLoop 情境檢查循環
func (fsm *FaultScenarioManager) scenarioCheckLoop(ctx context.Context) {
	ticker := time.NewTicker(fsm.config.ScenarioCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			fsm.checkScenarioConditions()
			fsm.updateActiveScenarios()
		}
	}
}

// checkScenarioConditions 檢查情境條件
func (fsm *FaultScenarioManager) checkScenarioConditions() {
	fsm.mu.RLock()
	scenarios := make([]*FaultScenario, 0, len(fsm.scenarios))
	for _, s := range fsm.scenarios {
		scenarios = append(scenarios, s)
	}
	fsm.mu.RUnlock()

	for _, scenario := range scenarios {
		if fsm.shouldTriggerScenario(scenario) {
			go fsm.TriggerScenario(scenario.ID, nil)
		}
	}
}

// shouldTriggerScenario 判斷是否應該觸發情境
func (fsm *FaultScenarioManager) shouldTriggerScenario(scenario *FaultScenario) bool {
	// 檢查機率
	if rand.Float64() > scenario.Probability {
		return false
	}

	// 檢查是否已經有相同情境在運行
	fsm.mu.RLock()
	for _, active := range fsm.activeScenarios {
		if active.ScenarioID == scenario.ID && active.State == "active" {
			fsm.mu.RUnlock()
			return false
		}
	}
	fsm.mu.RUnlock()

	// 檢查觸發條件
	for _, condition := range scenario.Conditions {
		if !fsm.evaluateCondition(condition) {
			return false
		}
	}

	return true
}

// evaluateCondition 評估條件
func (fsm *FaultScenarioManager) evaluateCondition(condition TriggerCondition) bool {
	// 這裡應該實現具體的條件評估邏輯
	// 目前返回隨機結果用於演示
	return rand.Float64() > 0.5
}

// updateActiveScenarios 更新活動情境
func (fsm *FaultScenarioManager) updateActiveScenarios() {
	fsm.mu.Lock()
	defer fsm.mu.Unlock()

	now := time.Now()
	completed := make([]string, 0)

	for instanceID, active := range fsm.activeScenarios {
		if active.State == "active" && now.After(active.EndTime) {
			active.State = "recovering"
			active.LastUpdate = now

			// 開始恢復過程
			if fsm.config.EnableAutoRecovery {
				go fsm.startRecovery(instanceID)
			}
		}

		if active.State == "resolved" {
			completed = append(completed, instanceID)
		}
	}

	// 清理已解決的情境
	for _, instanceID := range completed {
		delete(fsm.activeScenarios, instanceID)
	}
}

// recoveryLoop 恢復循環
func (fsm *FaultScenarioManager) recoveryLoop(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			fsm.processRecoveries()
		}
	}
}

// processRecoveries 處理恢復
func (fsm *FaultScenarioManager) processRecoveries() {
	fsm.mu.RLock()
	recovering := make([]*ActiveScenario, 0)
	for _, active := range fsm.activeScenarios {
		if active.State == "recovering" {
			recovering = append(recovering, active)
		}
	}
	fsm.mu.RUnlock()

	for _, active := range recovering {
		fsm.attemptRecovery(active.InstanceID)
	}
}

// startRecovery 開始恢復
func (fsm *FaultScenarioManager) startRecovery(instanceID string) {
	fsm.mu.RLock()
	active, exists := fsm.activeScenarios[instanceID]
	if !exists {
		fsm.mu.RUnlock()
		return
	}

	scenario, scenarioExists := fsm.scenarios[active.ScenarioID]
	fsm.mu.RUnlock()

	if !scenarioExists {
		return
	}

	fsm.logger.WithFields(logrus.Fields{
		"instance_id": instanceID,
		"scenario_id": active.ScenarioID,
	}).Info("Starting recovery process")

	// 執行恢復動作
	for _, action := range scenario.RecoveryActions {
		time.Sleep(action.Delay)
		fsm.executeRecoveryAction(action, active.AffectedDevices)
	}
}

// attemptRecovery 嘗試恢復
func (fsm *FaultScenarioManager) attemptRecovery(instanceID string) {
	fsm.mu.Lock()
	defer fsm.mu.Unlock()

	active, exists := fsm.activeScenarios[instanceID]
	if !exists || active.State != "recovering" {
		return
	}

	active.RecoveryAttempts++

	// 檢查恢復是否成功
	if fsm.verifyRecovery(active) {
		active.State = "resolved"
		active.LastUpdate = time.Now()

		fsm.recordEvent(FaultEvent{
			ID:          fmt.Sprintf("event_%d", time.Now().UnixNano()),
			ScenarioID:  active.ScenarioID,
			Type:        "scenario_resolved",
			Severity:    "info",
			Timestamp:   time.Now(),
			Description: "Fault scenario resolved",
			Data: map[string]interface{}{
				"instance_id":       instanceID,
				"recovery_attempts": active.RecoveryAttempts,
				"total_duration":    time.Since(active.StartTime).String(),
			},
		})

		fsm.logger.WithFields(logrus.Fields{
			"instance_id":       instanceID,
			"recovery_attempts": active.RecoveryAttempts,
		}).Info("Fault scenario resolved")

	} else if active.RecoveryAttempts >= fsm.config.MaxRecoveryAttempts {
		fsm.logger.WithFields(logrus.Fields{
			"instance_id": instanceID,
			"attempts":    active.RecoveryAttempts,
		}).Error("Recovery failed after maximum attempts")
	}
}

// randomFailureLoop 隨機故障循環
func (fsm *FaultScenarioManager) randomFailureLoop(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if rand.Float64() < fsm.config.RandomFailureRate {
				fsm.triggerRandomFailure()
			}
		}
	}
}

// triggerRandomFailure 觸發隨機故障
func (fsm *FaultScenarioManager) triggerRandomFailure() {
	fsm.mu.RLock()
	scenarioIDs := make([]string, 0, len(fsm.scenarios))
	for id := range fsm.scenarios {
		scenarioIDs = append(scenarioIDs, id)
	}
	fsm.mu.RUnlock()

	if len(scenarioIDs) > 0 {
		randomScenario := scenarioIDs[rand.Intn(len(scenarioIDs))]
		fsm.TriggerScenario(randomScenario, nil)
	}
}

// eventCleanupLoop 事件清理循環
func (fsm *FaultScenarioManager) eventCleanupLoop(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			fsm.cleanupOldEvents()
		}
	}
}

// cleanupOldEvents 清理舊事件
func (fsm *FaultScenarioManager) cleanupOldEvents() {
	fsm.mu.Lock()
	defer fsm.mu.Unlock()

	cutoff := time.Now().Add(-fsm.config.EventRetention)
	newHistory := make([]FaultEvent, 0)

	for _, event := range fsm.eventHistory {
		if event.Timestamp.After(cutoff) {
			newHistory = append(newHistory, event)
		}
	}

	fsm.eventHistory = newHistory
}

// 輔助方法

// determineAffectedDevices 確定受影響的設備
func (fsm *FaultScenarioManager) determineAffectedDevices(scenario *FaultScenario, targetDevices []string) []string {
	if len(targetDevices) > 0 {
		return targetDevices
	}

	affected := make([]string, 0)

	if len(scenario.AffectedDevices) > 0 {
		// 使用情境定義的設備
		for _, pattern := range scenario.AffectedDevices {
			if pattern == "all" {
				for id := range fsm.devices {
					affected = append(affected, id)
				}
			} else {
				// 匹配設備類型或 ID
				for id, device := range fsm.devices {
					if device.GetDeviceType() == pattern || id == pattern {
						affected = append(affected, id)
					}
				}
			}
		}
	} else {
		// 隨機選擇一些設備
		for id := range fsm.devices {
			if rand.Float64() < 0.3 {
				affected = append(affected, id)
			}
		}
	}

	return affected
}

// assessImpact 評估影響
func (fsm *FaultScenarioManager) assessImpact(scenario *FaultScenario, affectedDevices []string) *ImpactAssessment {
	impact := &ImpactAssessment{
		TotalDevices:      len(fsm.devices),
		AffectedDevices:   len(affectedDevices),
		EstimatedDowntime: scenario.Duration,
		RecoveryTime:      scenario.RecoveryTime,
	}

	// 計算服務可用性
	if impact.TotalDevices > 0 {
		impact.ServiceAvailability = float64(impact.TotalDevices-impact.AffectedDevices) / float64(impact.TotalDevices) * 100
	}

	// 評估數據丟失風險
	for _, effect := range scenario.Effects {
		if effect.Type == "data_loss" {
			impact.DataLoss = true
			break
		}
	}

	// 評估業務影響
	switch scenario.Severity {
	case "critical":
		impact.BusinessImpact = "Complete service outage"
	case "high":
		impact.BusinessImpact = "Major service degradation"
	case "medium":
		impact.BusinessImpact = "Moderate service impact"
	case "low":
		impact.BusinessImpact = "Minor service impact"
	default:
		impact.BusinessImpact = "Unknown impact"
	}

	return impact
}

// applyFaultEffects 應用故障效果
func (fsm *FaultScenarioManager) applyFaultEffects(scenario *FaultScenario, affectedDevices []string) {
	for _, effect := range scenario.Effects {
		for _, deviceID := range affectedDevices {
			if device, exists := fsm.devices[deviceID]; exists {
				fsm.applyEffectToDevice(effect, device)
			}
		}
	}
}

// applyEffectToDevice 應用效果到設備
func (fsm *FaultScenarioManager) applyEffectToDevice(effect FaultEffect, device base.Device) {
	cmd := base.Command{
		ID:   fmt.Sprintf("fault_effect_%d", time.Now().UnixNano()),
		Type: "apply_fault",
		Parameters: map[string]interface{}{
			"effect_type": effect.Type,
			"severity":    effect.Severity,
			"parameters":  effect.Parameters,
		},
		Timeout: 5 * time.Second,
	}

	err := device.HandleCommand(cmd)
	if err != nil {
		fsm.logger.WithError(err).WithField("device_id", device.GetDeviceID()).
			Error("Failed to apply fault effect")
	}
}

// executeRecoveryAction 執行恢復動作
func (fsm *FaultScenarioManager) executeRecoveryAction(action RecoveryAction, affectedDevices []string) {
	fsm.logger.WithFields(logrus.Fields{
		"action_type": action.Type,
		"target":      action.Target,
	}).Debug("Executing recovery action")

	for _, deviceID := range affectedDevices {
		if device, exists := fsm.devices[deviceID]; exists {
			cmd := base.Command{
				ID:         fmt.Sprintf("recovery_%d", time.Now().UnixNano()),
				Type:       action.Type,
				Parameters: action.Parameters,
				Timeout:    30 * time.Second,
			}

			device.HandleCommand(cmd)
		}
	}
}

// verifyRecovery 驗證恢復
func (fsm *FaultScenarioManager) verifyRecovery(active *ActiveScenario) bool {
	// 檢查所有受影響設備的健康狀態
	for _, deviceID := range active.AffectedDevices {
		if device, exists := fsm.devices[deviceID]; exists {
			if device.GetHealth() != "healthy" {
				return false
			}
		}
	}
	return true
}

// triggerCascadeScenarios 觸發級聯故障
func (fsm *FaultScenarioManager) triggerCascadeScenarios(cascadeScenarios []string, affectedDevices []string) {
	time.Sleep(5 * time.Second) // 延遲觸發級聯故障

	for _, scenarioID := range cascadeScenarios {
		fsm.logger.WithField("scenario_id", scenarioID).Info("Triggering cascade scenario")
		fsm.TriggerScenario(scenarioID, affectedDevices)
	}
}

// recordEvent 記錄事件
func (fsm *FaultScenarioManager) recordEvent(event FaultEvent) {
	fsm.mu.Lock()
	defer fsm.mu.Unlock()

	fsm.eventHistory = append(fsm.eventHistory, event)
}

// GetActiveScenarios 獲取活動情境
func (fsm *FaultScenarioManager) GetActiveScenarios() []*ActiveScenario {
	fsm.mu.RLock()
	defer fsm.mu.RUnlock()

	scenarios := make([]*ActiveScenario, 0, len(fsm.activeScenarios))
	for _, scenario := range fsm.activeScenarios {
		scenarios = append(scenarios, scenario)
	}
	return scenarios
}

// GetEventHistory 獲取事件歷史
func (fsm *FaultScenarioManager) GetEventHistory(limit int) []FaultEvent {
	fsm.mu.RLock()
	defer fsm.mu.RUnlock()

	if limit <= 0 || limit > len(fsm.eventHistory) {
		limit = len(fsm.eventHistory)
	}

	// 返回最新的事件
	start := len(fsm.eventHistory) - limit
	if start < 0 {
		start = 0
	}

	result := make([]FaultEvent, limit)
	copy(result, fsm.eventHistory[start:])
	return result
}

// GetImpactReport 獲取影響報告
func (fsm *FaultScenarioManager) GetImpactReport() map[string]interface{} {
	fsm.mu.RLock()
	defer fsm.mu.RUnlock()

	report := map[string]interface{}{
		"total_scenarios":  len(fsm.scenarios),
		"active_scenarios": len(fsm.activeScenarios),
		"total_events":     len(fsm.eventHistory),
		"scenarios_by_severity": map[string]int{
			"critical": 0,
			"high":     0,
			"medium":   0,
			"low":      0,
		},
		"current_impact": map[string]interface{}{
			"affected_devices":     0,
			"service_availability": 100.0,
		},
	}

	// 統計嚴重程度
	severityCount := report["scenarios_by_severity"].(map[string]int)
	for _, scenario := range fsm.scenarios {
		severityCount[scenario.Severity]++
	}

	// 計算當前影響
	totalAffected := 0
	for _, active := range fsm.activeScenarios {
		if active.State == "active" {
			totalAffected += len(active.AffectedDevices)
		}
	}

	currentImpact := report["current_impact"].(map[string]interface{})
	currentImpact["affected_devices"] = totalAffected
	if len(fsm.devices) > 0 {
		currentImpact["service_availability"] = float64(len(fsm.devices)-totalAffected) / float64(len(fsm.devices)) * 100
	}

	return report
}
