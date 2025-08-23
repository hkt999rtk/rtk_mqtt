package sync

import (
	"context"
	"sync"
	"time"

	"rtk_simulation/pkg/devices/base"

	"github.com/sirupsen/logrus"
)

// StateSync 狀態同步管理器
type StateSync struct {
	devices      map[string]base.Device
	deviceStates map[string]base.StatePayload
	subscribers  map[string][]StateSubscriber
	stateHistory map[string][]StateSnapshot
	syncRules    []SyncRule
	running      bool
	mu           sync.RWMutex
	logger       *logrus.Entry
}

// StateSubscriber 狀態訂閱者
type StateSubscriber struct {
	DeviceID string
	Handler  StateChangeHandler
	Filter   StateFilter
}

// StateChangeHandler 狀態變化處理函數
type StateChangeHandler func(deviceID string, oldState, newState base.StatePayload)

// StateFilter 狀態過濾器
type StateFilter struct {
	Fields      []string               // 只關注特定字段
	Conditions  map[string]interface{} // 條件過濾
	MinInterval time.Duration          // 最小更新間隔
}

// StateSnapshot 狀態快照
type StateSnapshot struct {
	Timestamp time.Time         `json:"timestamp"`
	State     base.StatePayload `json:"state"`
	Source    string            `json:"source"`
}

// SyncRule 同步規則
type SyncRule struct {
	ID         string          `json:"id"`
	Name       string          `json:"name"`
	Source     StatePattern    `json:"source"`     // 源設備狀態模式
	Targets    []SyncAction    `json:"targets"`    // 目標同步動作
	Conditions []SyncCondition `json:"conditions"` // 同步條件
	Enabled    bool            `json:"enabled"`
	Priority   int             `json:"priority"`
	LastSync   time.Time       `json:"last_sync"`
	Delay      time.Duration   `json:"delay"` // 延遲同步
}

// StatePattern 狀態模式
type StatePattern struct {
	DeviceID   string      `json:"device_id"`
	DeviceType string      `json:"device_type,omitempty"`
	Field      string      `json:"field"`    // 狀態字段路徑
	Operator   string      `json:"operator"` // eq, ne, gt, lt, contains
	Value      interface{} `json:"value"`
	Site       string      `json:"site,omitempty"`
}

// SyncAction 同步動作
type SyncAction struct {
	TargetID   string                 `json:"target_id"`
	TargetType string                 `json:"target_type,omitempty"`
	Action     string                 `json:"action"` // set_state, send_command, trigger_event
	Parameters map[string]interface{} `json:"parameters"`
	Site       string                 `json:"site,omitempty"`
}

// SyncCondition 同步條件
type SyncCondition struct {
	DeviceID string      `json:"device_id"`
	Field    string      `json:"field"`
	Operator string      `json:"operator"`
	Value    interface{} `json:"value"`
	Required bool        `json:"required"`
}

// NewStateSync 創建狀態同步管理器
func NewStateSync() *StateSync {
	return &StateSync{
		devices:      make(map[string]base.Device),
		deviceStates: make(map[string]base.StatePayload),
		subscribers:  make(map[string][]StateSubscriber),
		stateHistory: make(map[string][]StateSnapshot),
		syncRules:    make([]SyncRule, 0),
		running:      false,
		logger:       logrus.WithField("component", "state_sync"),
	}
}

// Start 啟動狀態同步管理器
func (ss *StateSync) Start(ctx context.Context) error {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	if ss.running {
		return nil
	}

	ss.running = true
	ss.logger.Info("Starting state sync manager")

	// 啟動狀態收集器
	go ss.runStateCollector(ctx)

	// 啟動同步規則處理器
	go ss.runSyncProcessor(ctx)

	// 啟動狀態歷史清理器
	go ss.runHistoryCleanup(ctx)

	return nil
}

// Stop 停止狀態同步管理器
func (ss *StateSync) Stop() error {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	if !ss.running {
		return nil
	}

	ss.running = false
	ss.logger.Info("Stopping state sync manager")

	return nil
}

// RegisterDevice 註冊設備
func (ss *StateSync) RegisterDevice(deviceID string, device base.Device) {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	ss.devices[deviceID] = device
	ss.logger.WithField("device_id", deviceID).Info("Device registered for state sync")
}

// UnregisterDevice 取消註冊設備
func (ss *StateSync) UnregisterDevice(deviceID string) {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	delete(ss.devices, deviceID)
	delete(ss.deviceStates, deviceID)
	delete(ss.subscribers, deviceID)
	delete(ss.stateHistory, deviceID)

	ss.logger.WithField("device_id", deviceID).Info("Device unregistered from state sync")
}

// Subscribe 訂閱設備狀態變化
func (ss *StateSync) Subscribe(deviceID string, subscriber StateSubscriber) {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	if ss.subscribers[deviceID] == nil {
		ss.subscribers[deviceID] = make([]StateSubscriber, 0)
	}

	ss.subscribers[deviceID] = append(ss.subscribers[deviceID], subscriber)
	ss.logger.WithFields(logrus.Fields{
		"device_id":     deviceID,
		"subscriber_id": subscriber.DeviceID,
	}).Info("State subscription added")
}

// AddSyncRule 添加同步規則
func (ss *StateSync) AddSyncRule(rule SyncRule) {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	rule.Enabled = true
	ss.syncRules = append(ss.syncRules, rule)

	ss.logger.WithFields(logrus.Fields{
		"rule_id":   rule.ID,
		"rule_name": rule.Name,
	}).Info("Sync rule added")
}

// UpdateState 更新設備狀態
func (ss *StateSync) UpdateState(deviceID string, state base.StatePayload) {
	ss.mu.Lock()

	oldState, exists := ss.deviceStates[deviceID]
	ss.deviceStates[deviceID] = state

	// 記錄狀態歷史
	snapshot := StateSnapshot{
		Timestamp: time.Now(),
		State:     state,
		Source:    "device_update",
	}

	if ss.stateHistory[deviceID] == nil {
		ss.stateHistory[deviceID] = make([]StateSnapshot, 0)
	}
	ss.stateHistory[deviceID] = append(ss.stateHistory[deviceID], snapshot)

	// 限制歷史記錄數量
	if len(ss.stateHistory[deviceID]) > 100 {
		ss.stateHistory[deviceID] = ss.stateHistory[deviceID][1:]
	}

	ss.mu.Unlock()

	// 通知訂閱者
	ss.notifySubscribers(deviceID, oldState, state, exists)

	// 觸發同步規則
	ss.triggerSyncRules(deviceID, state)
}

// GetDeviceState 獲取設備狀態
func (ss *StateSync) GetDeviceState(deviceID string) (base.StatePayload, bool) {
	ss.mu.RLock()
	defer ss.mu.RUnlock()

	state, exists := ss.deviceStates[deviceID]
	return state, exists
}

// GetStateHistory 獲取狀態歷史
func (ss *StateSync) GetStateHistory(deviceID string, limit int) []StateSnapshot {
	ss.mu.RLock()
	defer ss.mu.RUnlock()

	history, exists := ss.stateHistory[deviceID]
	if !exists {
		return nil
	}

	if limit <= 0 || limit > len(history) {
		limit = len(history)
	}

	result := make([]StateSnapshot, limit)
	copy(result, history[len(history)-limit:])
	return result
}

// runStateCollector 運行狀態收集器
func (ss *StateSync) runStateCollector(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			ss.collectDeviceStates()
		}
	}
}

// collectDeviceStates 收集設備狀態
func (ss *StateSync) collectDeviceStates() {
	ss.mu.RLock()
	devices := make(map[string]base.Device)
	for id, device := range ss.devices {
		devices[id] = device
	}
	ss.mu.RUnlock()

	for deviceID, device := range devices {
		state := device.GenerateStatePayload()
		ss.UpdateState(deviceID, state)
	}
}

// runSyncProcessor 運行同步處理器
func (ss *StateSync) runSyncProcessor(ctx context.Context) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			ss.processSyncRules()
		}
	}
}

// runHistoryCleanup 運行歷史清理器
func (ss *StateSync) runHistoryCleanup(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			ss.cleanupHistory()
		}
	}
}

// notifySubscribers 通知訂閱者
func (ss *StateSync) notifySubscribers(deviceID string, oldState, newState base.StatePayload, hasOldState bool) {
	ss.mu.RLock()
	subscribers := ss.subscribers[deviceID]
	ss.mu.RUnlock()

	if !hasOldState {
		oldState = base.StatePayload{}
	}

	for _, subscriber := range subscribers {
		if ss.matchesFilter(subscriber.Filter, oldState, newState) {
			go subscriber.Handler(deviceID, oldState, newState)
		}
	}
}

// triggerSyncRules 觸發同步規則
func (ss *StateSync) triggerSyncRules(deviceID string, state base.StatePayload) {
	ss.mu.RLock()
	rules := make([]SyncRule, len(ss.syncRules))
	copy(rules, ss.syncRules)
	ss.mu.RUnlock()

	for _, rule := range rules {
		if rule.Enabled && ss.matchesPattern(rule.Source, deviceID, state) {
			go ss.executeSyncRule(rule, deviceID, state)
		}
	}
}

// matchesFilter 檢查是否匹配過濾器
func (ss *StateSync) matchesFilter(filter StateFilter, oldState, newState base.StatePayload) bool {
	// 簡化實現，實際可以更複雜
	return true
}

// matchesPattern 檢查是否匹配狀態模式
func (ss *StateSync) matchesPattern(pattern StatePattern, deviceID string, state base.StatePayload) bool {
	if pattern.DeviceID != "" && pattern.DeviceID != deviceID {
		return false
	}

	// 簡化的字段匹配邏輯
	value := ss.getStateFieldValue(state, pattern.Field)
	return ss.compareValues(value, pattern.Operator, pattern.Value)
}

// executeSyncRule 執行同步規則
func (ss *StateSync) executeSyncRule(rule SyncRule, sourceDeviceID string, sourceState base.StatePayload) {
	// 檢查條件
	if !ss.checkSyncConditions(rule.Conditions) {
		return
	}

	// 延遲執行
	if rule.Delay > 0 {
		time.Sleep(rule.Delay)
	}

	ss.logger.WithFields(logrus.Fields{
		"rule_id":       rule.ID,
		"source_device": sourceDeviceID,
	}).Info("Executing sync rule")

	// 執行同步動作
	for _, action := range rule.Targets {
		ss.executeSyncAction(action, sourceDeviceID, sourceState)
	}

	// 更新最後同步時間
	ss.mu.Lock()
	for i := range ss.syncRules {
		if ss.syncRules[i].ID == rule.ID {
			ss.syncRules[i].LastSync = time.Now()
			break
		}
	}
	ss.mu.Unlock()
}

// executeSyncAction 執行同步動作
func (ss *StateSync) executeSyncAction(action SyncAction, sourceDeviceID string, sourceState base.StatePayload) {
	ss.mu.RLock()
	targetDevice, exists := ss.devices[action.TargetID]
	ss.mu.RUnlock()

	if !exists {
		ss.logger.WithField("target_id", action.TargetID).Warn("Target device not found for sync action")
		return
	}

	switch action.Action {
	case "send_command":
		ss.sendSyncCommand(targetDevice, action, sourceState)
	case "trigger_event":
		ss.triggerSyncEvent(targetDevice, action, sourceState)
	case "set_state":
		ss.setSyncState(targetDevice, action, sourceState)
	}
}

// sendSyncCommand 發送同步命令
func (ss *StateSync) sendSyncCommand(device base.Device, action SyncAction, sourceState base.StatePayload) {
	cmd := base.Command{
		ID:         ss.generateCommandID(),
		Type:       action.Parameters["command"].(string),
		Parameters: action.Parameters,
		Timeout:    30 * time.Second,
	}

	if err := device.HandleCommand(cmd); err != nil {
		ss.logger.WithError(err).Warn("Failed to execute sync command")
	}
}

// triggerSyncEvent 觸發同步事件
func (ss *StateSync) triggerSyncEvent(device base.Device, action SyncAction, sourceState base.StatePayload) {
	// 實作事件觸發邏輯
	ss.logger.WithField("target_device", device.GetDeviceID()).Info("Sync event triggered")
}

// setSyncState 設置同步狀態
func (ss *StateSync) setSyncState(device base.Device, action SyncAction, sourceState base.StatePayload) {
	// 實作狀態設置邏輯
	ss.logger.WithField("target_device", device.GetDeviceID()).Info("Sync state set")
}

// 輔助方法
func (ss *StateSync) getStateFieldValue(state base.StatePayload, field string) interface{} {
	// 簡化實現，實際需要支援嵌套字段路徑
	switch field {
	case "health":
		return state.Health
	case "uptime":
		return state.UptimeS
	default:
		return nil
	}
}

func (ss *StateSync) compareValues(value interface{}, operator string, target interface{}) bool {
	switch operator {
	case "eq":
		return value == target
	case "ne":
		return value != target
	default:
		return false
	}
}

func (ss *StateSync) checkSyncConditions(conditions []SyncCondition) bool {
	// 簡化實現
	return true
}

func (ss *StateSync) generateCommandID() string {
	return "sync_" + time.Now().Format("20060102150405")
}

func (ss *StateSync) processSyncRules() {
	// 週期性處理同步規則
}

func (ss *StateSync) cleanupHistory() {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	cutoff := time.Now().Add(-24 * time.Hour)
	for deviceID, history := range ss.stateHistory {
		filtered := make([]StateSnapshot, 0)
		for _, snapshot := range history {
			if snapshot.Timestamp.After(cutoff) {
				filtered = append(filtered, snapshot)
			}
		}
		ss.stateHistory[deviceID] = filtered
	}
}

// GetStatistics 獲取同步統計
func (ss *StateSync) GetStatistics() map[string]interface{} {
	ss.mu.RLock()
	defer ss.mu.RUnlock()

	stats := map[string]interface{}{
		"registered_devices": len(ss.devices),
		"active_states":      len(ss.deviceStates),
		"sync_rules":         len(ss.syncRules),
		"subscribers":        len(ss.subscribers),
		"running":            ss.running,
	}

	// 計算歷史記錄統計
	totalHistory := 0
	for _, history := range ss.stateHistory {
		totalHistory += len(history)
	}
	stats["total_history_records"] = totalHistory

	return stats
}
