package sync

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"rtk_simulation/pkg/devices/base"
)

// GroupSyncManager 管理設備群組的狀態同步
type GroupSyncManager struct {
	groups    map[string]*DeviceGroup
	devices   map[string]base.Device
	syncRules []GroupSyncRule
	running   bool
	mu        sync.RWMutex
	logger    *logrus.Entry
}

// DeviceGroup 設備群組
type DeviceGroup struct {
	ID           string
	Name         string
	Description  string
	DeviceIDs    []string
	GroupType    string // room, floor, building, zone, custom
	SyncEnabled  bool
	SyncInterval time.Duration
	LastSync     time.Time
	SharedState  map[string]interface{}
	mu           sync.RWMutex
}

// GroupSyncRule 群組同步規則
type GroupSyncRule struct {
	ID          string
	Name        string
	SourceGroup string
	TargetGroup string
	SyncType    string // state, config, telemetry, events
	Conditions  []GroupSyncCondition
	Transform   TransformFunc
	Enabled     bool
}

// GroupSyncCondition 群組同步條件
type GroupSyncCondition struct {
	Field    string
	Operator string
	Value    interface{}
}

// TransformFunc 轉換函數類型
type TransformFunc func(data map[string]interface{}) map[string]interface{}

// SyncEvent 同步事件
type SyncEvent struct {
	Type      string
	GroupID   string
	DeviceID  string
	Data      map[string]interface{}
	Timestamp time.Time
}

// NewGroupSyncManager 創建新的群組同步管理器
func NewGroupSyncManager() *GroupSyncManager {
	return &GroupSyncManager{
		groups:    make(map[string]*DeviceGroup),
		devices:   make(map[string]base.Device),
		syncRules: make([]GroupSyncRule, 0),
		logger:    logrus.WithField("component", "group_sync_manager"),
	}
}

// Start 啟動群組同步管理器
func (gsm *GroupSyncManager) Start(ctx context.Context) error {
	gsm.mu.Lock()
	defer gsm.mu.Unlock()

	if gsm.running {
		return fmt.Errorf("group sync manager is already running")
	}

	gsm.running = true
	gsm.logger.Info("Starting group sync manager")

	// 啟動同步循環
	go gsm.runSyncLoop(ctx)
	go gsm.runRuleProcessor(ctx)

	return nil
}

// Stop 停止群組同步管理器
func (gsm *GroupSyncManager) Stop() error {
	gsm.mu.Lock()
	defer gsm.mu.Unlock()

	if !gsm.running {
		return fmt.Errorf("group sync manager is not running")
	}

	gsm.running = false
	gsm.logger.Info("Stopping group sync manager")
	return nil
}

// CreateGroup 創建設備群組
func (gsm *GroupSyncManager) CreateGroup(id, name, groupType string) *DeviceGroup {
	gsm.mu.Lock()
	defer gsm.mu.Unlock()

	group := &DeviceGroup{
		ID:           id,
		Name:         name,
		GroupType:    groupType,
		DeviceIDs:    make([]string, 0),
		SyncEnabled:  true,
		SyncInterval: 10 * time.Second,
		LastSync:     time.Now(),
		SharedState:  make(map[string]interface{}),
	}

	gsm.groups[id] = group

	gsm.logger.WithFields(logrus.Fields{
		"group_id":   id,
		"group_name": name,
		"group_type": groupType,
	}).Info("Device group created")

	return group
}

// DeleteGroup 刪除設備群組
func (gsm *GroupSyncManager) DeleteGroup(groupID string) error {
	gsm.mu.Lock()
	defer gsm.mu.Unlock()

	if _, exists := gsm.groups[groupID]; !exists {
		return fmt.Errorf("group %s not found", groupID)
	}

	delete(gsm.groups, groupID)
	gsm.logger.WithField("group_id", groupID).Info("Device group deleted")
	return nil
}

// AddDeviceToGroup 將設備添加到群組
func (gsm *GroupSyncManager) AddDeviceToGroup(deviceID, groupID string, device base.Device) error {
	gsm.mu.Lock()
	defer gsm.mu.Unlock()

	group, exists := gsm.groups[groupID]
	if !exists {
		return fmt.Errorf("group %s not found", groupID)
	}

	// 檢查設備是否已在群組中
	for _, id := range group.DeviceIDs {
		if id == deviceID {
			return fmt.Errorf("device %s already in group %s", deviceID, groupID)
		}
	}

	group.DeviceIDs = append(group.DeviceIDs, deviceID)
	gsm.devices[deviceID] = device

	gsm.logger.WithFields(logrus.Fields{
		"device_id": deviceID,
		"group_id":  groupID,
	}).Info("Device added to group")

	// 立即同步新設備的狀態
	go gsm.syncDeviceWithGroup(deviceID, groupID)

	return nil
}

// RemoveDeviceFromGroup 從群組中移除設備
func (gsm *GroupSyncManager) RemoveDeviceFromGroup(deviceID, groupID string) error {
	gsm.mu.Lock()
	defer gsm.mu.Unlock()

	group, exists := gsm.groups[groupID]
	if !exists {
		return fmt.Errorf("group %s not found", groupID)
	}

	// 從群組中移除設備
	newDeviceIDs := make([]string, 0)
	found := false
	for _, id := range group.DeviceIDs {
		if id != deviceID {
			newDeviceIDs = append(newDeviceIDs, id)
		} else {
			found = true
		}
	}

	if !found {
		return fmt.Errorf("device %s not in group %s", deviceID, groupID)
	}

	group.DeviceIDs = newDeviceIDs

	gsm.logger.WithFields(logrus.Fields{
		"device_id": deviceID,
		"group_id":  groupID,
	}).Info("Device removed from group")

	return nil
}

// AddSyncRule 添加同步規則
func (gsm *GroupSyncManager) AddSyncRule(rule GroupSyncRule) {
	gsm.mu.Lock()
	defer gsm.mu.Unlock()

	rule.Enabled = true
	gsm.syncRules = append(gsm.syncRules, rule)

	gsm.logger.WithFields(logrus.Fields{
		"rule_id":   rule.ID,
		"rule_name": rule.Name,
	}).Info("Sync rule added")
}

// runSyncLoop 運行同步循環
func (gsm *GroupSyncManager) runSyncLoop(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			gsm.performGroupSync()
		}
	}
}

// performGroupSync 執行群組同步
func (gsm *GroupSyncManager) performGroupSync() {
	gsm.mu.RLock()
	groups := make(map[string]*DeviceGroup)
	for id, group := range gsm.groups {
		groups[id] = group
	}
	gsm.mu.RUnlock()

	for groupID, group := range groups {
		if !group.SyncEnabled {
			continue
		}

		if time.Since(group.LastSync) >= group.SyncInterval {
			gsm.syncGroup(groupID)
		}
	}
}

// syncGroup 同步群組
func (gsm *GroupSyncManager) syncGroup(groupID string) {
	gsm.mu.RLock()
	group, exists := gsm.groups[groupID]
	if !exists {
		gsm.mu.RUnlock()
		return
	}
	deviceIDs := make([]string, len(group.DeviceIDs))
	copy(deviceIDs, group.DeviceIDs)
	gsm.mu.RUnlock()

	gsm.logger.WithField("group_id", groupID).Debug("Syncing group")

	// 收集所有設備的狀態
	groupState := gsm.collectGroupState(deviceIDs)

	// 更新群組共享狀態
	gsm.updateGroupSharedState(groupID, groupState)

	// 向所有設備廣播群組狀態
	gsm.broadcastGroupState(groupID, deviceIDs, groupState)

	// 更新最後同步時間
	gsm.mu.Lock()
	if g, exists := gsm.groups[groupID]; exists {
		g.LastSync = time.Now()
	}
	gsm.mu.Unlock()
}

// collectGroupState 收集群組狀態
func (gsm *GroupSyncManager) collectGroupState(deviceIDs []string) map[string]interface{} {
	groupState := map[string]interface{}{
		"devices":   make(map[string]interface{}),
		"summary":   make(map[string]interface{}),
		"timestamp": time.Now(),
	}

	deviceStates := groupState["devices"].(map[string]interface{})
	summary := groupState["summary"].(map[string]interface{})

	// 收集每個設備的狀態
	totalDevices := 0
	healthyDevices := 0
	totalCPU := 0.0
	totalMemory := 0.0

	for _, deviceID := range deviceIDs {
		gsm.mu.RLock()
		device, exists := gsm.devices[deviceID]
		gsm.mu.RUnlock()

		if !exists {
			continue
		}

		// 獲取設備狀態
		state := device.GenerateStatePayload()
		deviceStates[deviceID] = state

		// 更新統計
		totalDevices++
		if state.Health == "healthy" {
			healthyDevices++
		}
		totalCPU += state.CPUUsage
		totalMemory += state.MemoryUsage
	}

	// 計算群組摘要
	if totalDevices > 0 {
		summary["total_devices"] = totalDevices
		summary["healthy_devices"] = healthyDevices
		summary["average_cpu"] = totalCPU / float64(totalDevices)
		summary["average_memory"] = totalMemory / float64(totalDevices)
		summary["health_percentage"] = float64(healthyDevices) / float64(totalDevices) * 100
	}

	return groupState
}

// updateGroupSharedState 更新群組共享狀態
func (gsm *GroupSyncManager) updateGroupSharedState(groupID string, state map[string]interface{}) {
	gsm.mu.Lock()
	defer gsm.mu.Unlock()

	if group, exists := gsm.groups[groupID]; exists {
		group.mu.Lock()
		group.SharedState = state
		group.mu.Unlock()
	}
}

// broadcastGroupState 廣播群組狀態
func (gsm *GroupSyncManager) broadcastGroupState(groupID string, deviceIDs []string, state map[string]interface{}) {
	for _, deviceID := range deviceIDs {
		gsm.mu.RLock()
		device, exists := gsm.devices[deviceID]
		gsm.mu.RUnlock()

		if !exists {
			continue
		}

		// 發送群組狀態更新命令
		cmd := base.Command{
			ID:   fmt.Sprintf("group_sync_%d", time.Now().UnixNano()),
			Type: "group_state_update",
			Parameters: map[string]interface{}{
				"group_id":    groupID,
				"group_state": state,
				"timestamp":   time.Now(),
			},
			Timeout: 5 * time.Second,
		}

		go device.HandleCommand(cmd)
	}
}

// syncDeviceWithGroup 同步設備與群組
func (gsm *GroupSyncManager) syncDeviceWithGroup(deviceID, groupID string) {
	gsm.mu.RLock()
	group, groupExists := gsm.groups[groupID]
	device, deviceExists := gsm.devices[deviceID]
	gsm.mu.RUnlock()

	if !groupExists || !deviceExists {
		return
	}

	// 獲取群組共享狀態
	group.mu.RLock()
	sharedState := group.SharedState
	group.mu.RUnlock()

	// 發送群組狀態到新設備
	cmd := base.Command{
		ID:   fmt.Sprintf("device_sync_%d", time.Now().UnixNano()),
		Type: "sync_with_group",
		Parameters: map[string]interface{}{
			"group_id":     groupID,
			"shared_state": sharedState,
			"timestamp":    time.Now(),
		},
		Timeout: 5 * time.Second,
	}

	device.HandleCommand(cmd)
}

// runRuleProcessor 運行規則處理器
func (gsm *GroupSyncManager) runRuleProcessor(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			gsm.processSyncRules()
		}
	}
}

// processSyncRules 處理同步規則
func (gsm *GroupSyncManager) processSyncRules() {
	gsm.mu.RLock()
	rules := make([]GroupSyncRule, len(gsm.syncRules))
	copy(rules, gsm.syncRules)
	gsm.mu.RUnlock()

	for _, rule := range rules {
		if !rule.Enabled {
			continue
		}

		gsm.processRule(rule)
	}
}

// processRule 處理單個規則
func (gsm *GroupSyncManager) processRule(rule GroupSyncRule) {
	gsm.mu.RLock()
	sourceGroup, sourceExists := gsm.groups[rule.SourceGroup]
	_, targetExists := gsm.groups[rule.TargetGroup]
	gsm.mu.RUnlock()

	if !sourceExists || !targetExists {
		return
	}

	// 獲取源群組狀態
	sourceGroup.mu.RLock()
	sourceState := sourceGroup.SharedState
	sourceGroup.mu.RUnlock()

	// 檢查條件
	if !gsm.checkSyncConditions(rule.Conditions, sourceState) {
		return
	}

	// 應用轉換
	var syncData map[string]interface{}
	if rule.Transform != nil {
		syncData = rule.Transform(sourceState)
	} else {
		syncData = sourceState
	}

	// 同步到目標群組
	gsm.syncGroupToGroup(rule.TargetGroup, syncData, rule.SyncType)
}

// checkSyncConditions 檢查同步條件
func (gsm *GroupSyncManager) checkSyncConditions(conditions []GroupSyncCondition, state map[string]interface{}) bool {
	for _, condition := range conditions {
		if !gsm.evaluateCondition(condition, state) {
			return false
		}
	}
	return true
}

// evaluateCondition 評估條件
func (gsm *GroupSyncManager) evaluateCondition(condition GroupSyncCondition, state map[string]interface{}) bool {
	// 簡化的條件評估
	// 實際應用中可以實現更複雜的邏輯
	return true
}

// syncGroupToGroup 群組間同步
func (gsm *GroupSyncManager) syncGroupToGroup(targetGroupID string, data map[string]interface{}, syncType string) {
	gsm.mu.RLock()
	targetGroup, exists := gsm.groups[targetGroupID]
	if !exists {
		gsm.mu.RUnlock()
		return
	}
	deviceIDs := make([]string, len(targetGroup.DeviceIDs))
	copy(deviceIDs, targetGroup.DeviceIDs)
	gsm.mu.RUnlock()

	// 根據同步類型處理數據
	var syncData map[string]interface{}
	switch syncType {
	case "state":
		syncData = map[string]interface{}{
			"type": "state_sync",
			"data": data,
		}
	case "config":
		syncData = map[string]interface{}{
			"type": "config_sync",
			"data": data,
		}
	case "telemetry":
		syncData = map[string]interface{}{
			"type": "telemetry_sync",
			"data": data,
		}
	case "events":
		syncData = map[string]interface{}{
			"type": "event_sync",
			"data": data,
		}
	default:
		syncData = data
	}

	// 向目標群組的所有設備發送同步數據
	for _, deviceID := range deviceIDs {
		gsm.mu.RLock()
		device, exists := gsm.devices[deviceID]
		gsm.mu.RUnlock()

		if !exists {
			continue
		}

		cmd := base.Command{
			ID:   fmt.Sprintf("group_to_group_sync_%d", time.Now().UnixNano()),
			Type: "cross_group_sync",
			Parameters: map[string]interface{}{
				"source_group": targetGroupID,
				"sync_type":    syncType,
				"sync_data":    syncData,
				"timestamp":    time.Now(),
			},
			Timeout: 5 * time.Second,
		}

		go device.HandleCommand(cmd)
	}

	gsm.logger.WithFields(logrus.Fields{
		"target_group": targetGroupID,
		"sync_type":    syncType,
	}).Debug("Cross-group sync performed")
}

// TriggerGroupEvent 觸發群組事件
func (gsm *GroupSyncManager) TriggerGroupEvent(groupID string, eventType string, data map[string]interface{}) {
	gsm.mu.RLock()
	group, exists := gsm.groups[groupID]
	if !exists {
		gsm.mu.RUnlock()
		return
	}
	deviceIDs := make([]string, len(group.DeviceIDs))
	copy(deviceIDs, group.DeviceIDs)
	gsm.mu.RUnlock()

	event := SyncEvent{
		Type:      eventType,
		GroupID:   groupID,
		Data:      data,
		Timestamp: time.Now(),
	}

	// 向群組中的所有設備廣播事件
	for _, deviceID := range deviceIDs {
		gsm.mu.RLock()
		device, exists := gsm.devices[deviceID]
		gsm.mu.RUnlock()

		if !exists {
			continue
		}

		cmd := base.Command{
			ID:   fmt.Sprintf("group_event_%d", time.Now().UnixNano()),
			Type: "group_event",
			Parameters: map[string]interface{}{
				"event": event,
			},
			Timeout: 5 * time.Second,
		}

		go device.HandleCommand(cmd)
	}

	gsm.logger.WithFields(logrus.Fields{
		"group_id":     groupID,
		"event_type":   eventType,
		"device_count": len(deviceIDs),
	}).Info("Group event triggered")
}

// GetGroupStatistics 獲取群組統計
func (gsm *GroupSyncManager) GetGroupStatistics() map[string]interface{} {
	gsm.mu.RLock()
	defer gsm.mu.RUnlock()

	stats := map[string]interface{}{
		"total_groups":       len(gsm.groups),
		"total_devices":      len(gsm.devices),
		"total_sync_rules":   len(gsm.syncRules),
		"enabled_sync_rules": 0,
		"group_types":        make(map[string]int),
		"devices_per_group":  make(map[string]int),
	}

	// 統計啟用的規則
	for _, rule := range gsm.syncRules {
		if rule.Enabled {
			stats["enabled_sync_rules"] = stats["enabled_sync_rules"].(int) + 1
		}
	}

	// 統計群組類型和設備分佈
	groupTypes := stats["group_types"].(map[string]int)
	devicesPerGroup := stats["devices_per_group"].(map[string]int)

	for groupID, group := range gsm.groups {
		// 統計群組類型
		if _, exists := groupTypes[group.GroupType]; !exists {
			groupTypes[group.GroupType] = 0
		}
		groupTypes[group.GroupType]++

		// 統計每個群組的設備數
		devicesPerGroup[groupID] = len(group.DeviceIDs)
	}

	return stats
}

// GetGroupState 獲取群組狀態
func (gsm *GroupSyncManager) GetGroupState(groupID string) (map[string]interface{}, error) {
	gsm.mu.RLock()
	group, exists := gsm.groups[groupID]
	gsm.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("group %s not found", groupID)
	}

	group.mu.RLock()
	defer group.mu.RUnlock()

	return group.SharedState, nil
}

// UpdateGroupConfig 更新群組配置
func (gsm *GroupSyncManager) UpdateGroupConfig(groupID string, config map[string]interface{}) error {
	gsm.mu.Lock()
	group, exists := gsm.groups[groupID]
	gsm.mu.Unlock()

	if !exists {
		return fmt.Errorf("group %s not found", groupID)
	}

	// 更新群組配置
	if syncEnabled, ok := config["sync_enabled"].(bool); ok {
		group.SyncEnabled = syncEnabled
	}

	if syncInterval, ok := config["sync_interval"].(time.Duration); ok {
		group.SyncInterval = syncInterval
	}

	if description, ok := config["description"].(string); ok {
		group.Description = description
	}

	gsm.logger.WithFields(logrus.Fields{
		"group_id": groupID,
		"config":   config,
	}).Info("Group configuration updated")

	// 立即觸發同步
	go gsm.syncGroup(groupID)

	return nil
}
