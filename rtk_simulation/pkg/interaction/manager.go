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

// InteractionManager 設備互動管理器
type InteractionManager struct {
	devices      map[string]base.Device
	interactions []InteractionRule
	running      bool
	mu           sync.RWMutex
	logger       *logrus.Entry
}

// InteractionRule 互動規則
type InteractionRule struct {
	ID            string              `json:"id"`
	Name          string              `json:"name"`
	Trigger       TriggerCondition    `json:"trigger"`
	Actions       []InteractionAction `json:"actions"`
	Probability   float64             `json:"probability"` // 0.0-1.0
	Cooldown      time.Duration       `json:"cooldown"`    // 冷卻時間
	LastTriggered time.Time           `json:"last_triggered"`
	Enabled       bool                `json:"enabled"`
	Priority      int                 `json:"priority"` // 優先級，數字越大優先級越高
}

// TriggerCondition 觸發條件
type TriggerCondition struct {
	Type           string                 `json:"type"`            // event, time, state, manual
	DeviceID       string                 `json:"device_id"`       // 觸發設備 ID
	EventType      string                 `json:"event_type"`      // 事件類型
	TimePattern    string                 `json:"time_pattern"`    // 時間模式 (cron-like)
	StateCondition StateCondition         `json:"state_condition"` // 狀態條件
	Parameters     map[string]interface{} `json:"parameters"`
}

// StateCondition 狀態條件
type StateCondition struct {
	Field    string      `json:"field"`    // 狀態字段
	Operator string      `json:"operator"` // eq, ne, gt, lt, gte, lte
	Value    interface{} `json:"value"`    // 比較值
}

// InteractionAction 互動動作
type InteractionAction struct {
	Type       string                 `json:"type"`       // command, notification, automation
	TargetID   string                 `json:"target_id"`  // 目標設備 ID
	Command    string                 `json:"command"`    // 命令
	Parameters map[string]interface{} `json:"parameters"` // 命令參數
	Delay      time.Duration          `json:"delay"`      // 延遲執行
}

// NewInteractionManager 建立新的互動管理器
func NewInteractionManager() *InteractionManager {
	return &InteractionManager{
		devices:      make(map[string]base.Device),
		interactions: make([]InteractionRule, 0),
		running:      false,
		logger:       logrus.WithField("component", "interaction_manager"),
	}
}

// Start 啟動互動管理器
func (im *InteractionManager) Start(ctx context.Context) error {
	im.mu.Lock()
	defer im.mu.Unlock()

	if im.running {
		return fmt.Errorf("interaction manager is already running")
	}

	im.running = true
	im.logger.Info("Starting interaction manager")

	// 啟動規則處理器
	go im.runRuleProcessor(ctx)
	go im.runTimeBasedTriggers(ctx)

	return nil
}

// Stop 停止互動管理器
func (im *InteractionManager) Stop() error {
	im.mu.Lock()
	defer im.mu.Unlock()

	if !im.running {
		return fmt.Errorf("interaction manager is not running")
	}

	im.running = false
	im.logger.Info("Stopping interaction manager")

	return nil
}

// RegisterDevice 註冊設備
func (im *InteractionManager) RegisterDevice(deviceID string, device base.Device) {
	im.mu.Lock()
	defer im.mu.Unlock()

	im.devices[deviceID] = device
	im.logger.WithField("device_id", deviceID).Info("Device registered for interactions")
}

// UnregisterDevice 取消註冊設備
func (im *InteractionManager) UnregisterDevice(deviceID string) {
	im.mu.Lock()
	defer im.mu.Unlock()

	delete(im.devices, deviceID)
	im.logger.WithField("device_id", deviceID).Info("Device unregistered from interactions")
}

// AddInteractionRule 新增互動規則
func (im *InteractionManager) AddInteractionRule(rule InteractionRule) {
	im.mu.Lock()
	defer im.mu.Unlock()

	rule.Enabled = true
	im.interactions = append(im.interactions, rule)
	im.logger.WithFields(logrus.Fields{
		"rule_id":   rule.ID,
		"rule_name": rule.Name,
	}).Info("Interaction rule added")
}

// RemoveInteractionRule 移除互動規則
func (im *InteractionManager) RemoveInteractionRule(ruleID string) {
	im.mu.Lock()
	defer im.mu.Unlock()

	for i, rule := range im.interactions {
		if rule.ID == ruleID {
			im.interactions = append(im.interactions[:i], im.interactions[i+1:]...)
			im.logger.WithField("rule_id", ruleID).Info("Interaction rule removed")
			return
		}
	}
}

// TriggerEvent 觸發事件處理
func (im *InteractionManager) TriggerEvent(deviceID, eventType string, data map[string]interface{}) {
	im.mu.RLock()
	if !im.running {
		im.mu.RUnlock()
		return
	}

	// 複製需要檢查的規則
	rulesToCheck := make([]InteractionRule, 0)
	for _, rule := range im.interactions {
		if rule.Enabled && rule.Trigger.Type == "event" {
			if rule.Trigger.DeviceID == deviceID && rule.Trigger.EventType == eventType {
				rulesToCheck = append(rulesToCheck, rule)
			}
		}
	}
	im.mu.RUnlock()

	// 處理匹配的規則
	for _, rule := range rulesToCheck {
		go im.processRule(rule, deviceID, data)
	}
}

// runRuleProcessor 運行規則處理器
func (im *InteractionManager) runRuleProcessor(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			im.processStateBasedRules()
		}
	}
}

// runTimeBasedTriggers 運行基於時間的觸發器
func (im *InteractionManager) runTimeBasedTriggers(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			im.processTimeBasedRules()
		}
	}
}

// processRule 處理規則
func (im *InteractionManager) processRule(rule InteractionRule, triggerDeviceID string, eventData map[string]interface{}) {
	// 檢查冷卻時間
	if time.Since(rule.LastTriggered) < rule.Cooldown {
		return
	}

	// 檢查概率
	if rand.Float64() > rule.Probability {
		return
	}

	im.logger.WithFields(logrus.Fields{
		"rule_id":           rule.ID,
		"rule_name":         rule.Name,
		"trigger_device_id": triggerDeviceID,
	}).Info("Processing interaction rule")

	// 執行動作
	for _, action := range rule.Actions {
		go im.executeAction(action, rule, eventData)
	}

	// 更新最後觸發時間
	im.mu.Lock()
	for i := range im.interactions {
		if im.interactions[i].ID == rule.ID {
			im.interactions[i].LastTriggered = time.Now()
			break
		}
	}
	im.mu.Unlock()
}

// executeAction 執行動作
func (im *InteractionManager) executeAction(action InteractionAction, rule InteractionRule, eventData map[string]interface{}) {
	// 延遲執行
	if action.Delay > 0 {
		time.Sleep(action.Delay)
	}

	im.mu.RLock()
	targetDevice, exists := im.devices[action.TargetID]
	im.mu.RUnlock()

	if !exists {
		im.logger.WithField("target_id", action.TargetID).Warn("Target device not found")
		return
	}

	switch action.Type {
	case "command":
		im.executeCommand(targetDevice, action, eventData)
	case "notification":
		im.sendNotification(action, rule, eventData)
	case "automation":
		im.executeAutomation(action, rule, eventData)
	}
}

// executeCommand 執行設備命令
func (im *InteractionManager) executeCommand(device base.Device, action InteractionAction, eventData map[string]interface{}) {
	// 構建命令
	cmd := base.Command{
		ID:         fmt.Sprintf("auto_%d", time.Now().UnixNano()),
		Type:       action.Command,
		Parameters: action.Parameters,
		Timeout:    30 * time.Second,
	}

	// 執行命令
	if err := device.HandleCommand(cmd); err != nil {
		im.logger.WithFields(logrus.Fields{
			"device_id": device.GetDeviceID(),
			"command":   action.Command,
			"error":     err,
		}).Error("Failed to execute command")
	} else {
		im.logger.WithFields(logrus.Fields{
			"device_id": device.GetDeviceID(),
			"command":   action.Command,
		}).Info("Command executed successfully")
	}
}

// sendNotification 發送通知
func (im *InteractionManager) sendNotification(action InteractionAction, rule InteractionRule, eventData map[string]interface{}) {
	notification := map[string]interface{}{
		"type":      "interaction_notification",
		"rule_id":   rule.ID,
		"rule_name": rule.Name,
		"target_id": action.TargetID,
		"message":   action.Parameters["message"],
		"timestamp": time.Now(),
		"data":      eventData,
	}

	im.logger.WithFields(logrus.Fields{
		"notification": notification,
	}).Info("Notification sent")
}

// executeAutomation 執行自動化
func (im *InteractionManager) executeAutomation(action InteractionAction, rule InteractionRule, eventData map[string]interface{}) {
	automationType := action.Parameters["type"].(string)

	switch automationType {
	case "scene_activation":
		im.activateScene(action.Parameters["scene_name"].(string))
	case "schedule_adjustment":
		im.adjustSchedule(action.Parameters)
	case "security_mode":
		im.changeSecurityMode(action.Parameters["mode"].(string))
	}
}

// processStateBasedRules 處理基於狀態的規則
func (im *InteractionManager) processStateBasedRules() {
	im.mu.RLock()
	rulesToProcess := make([]InteractionRule, 0)
	for _, rule := range im.interactions {
		if rule.Enabled && rule.Trigger.Type == "state" {
			rulesToProcess = append(rulesToProcess, rule)
		}
	}
	im.mu.RUnlock()

	for _, rule := range rulesToProcess {
		im.checkStateCondition(rule)
	}
}

// processTimeBasedRules 處理基於時間的規則
func (im *InteractionManager) processTimeBasedRules() {
	now := time.Now()

	im.mu.RLock()
	rulesToProcess := make([]InteractionRule, 0)
	for _, rule := range im.interactions {
		if rule.Enabled && rule.Trigger.Type == "time" {
			if im.matchTimePattern(rule.Trigger.TimePattern, now) {
				rulesToProcess = append(rulesToProcess, rule)
			}
		}
	}
	im.mu.RUnlock()

	for _, rule := range rulesToProcess {
		go im.processRule(rule, "", map[string]interface{}{
			"trigger_time": now,
		})
	}
}

// checkStateCondition 檢查狀態條件
func (im *InteractionManager) checkStateCondition(rule InteractionRule) {
	im.mu.RLock()
	device, exists := im.devices[rule.Trigger.DeviceID]
	im.mu.RUnlock()

	if !exists {
		return
	}

	// 獲取設備狀態
	state := device.GenerateStatePayload()

	// 檢查狀態條件
	if im.evaluateStateCondition(rule.Trigger.StateCondition, state) {
		go im.processRule(rule, rule.Trigger.DeviceID, map[string]interface{}{
			"device_state": state,
		})
	}
}

// evaluateStateCondition 評估狀態條件
func (im *InteractionManager) evaluateStateCondition(condition StateCondition, state base.StatePayload) bool {
	// 這裡可以實作複雜的狀態條件評估邏輯
	// 目前提供基本實現
	return true
}

// matchTimePattern 匹配時間模式
func (im *InteractionManager) matchTimePattern(pattern string, t time.Time) bool {
	// 實作時間模式匹配 (簡化版本)
	// 支援 "HH:MM", "HH:MM-HH:MM" 格式
	switch pattern {
	case "morning":
		return t.Hour() >= 6 && t.Hour() < 12
	case "afternoon":
		return t.Hour() >= 12 && t.Hour() < 18
	case "evening":
		return t.Hour() >= 18 && t.Hour() < 22
	case "night":
		return t.Hour() >= 22 || t.Hour() < 6
	default:
		// 可以添加更複雜的時間模式解析
		return false
	}
}

// buildCommandPayload 構建命令參數
func (im *InteractionManager) buildCommandPayload(parameters map[string]interface{}, eventData map[string]interface{}) string {
	// 這裡可以實作模板替換和參數處理
	// 目前返回簡單的 JSON
	data := fmt.Sprintf(`{"parameters": %+v, "event_data": %+v}`, parameters, eventData)
	return data
}

// activateScene 激活場景
func (im *InteractionManager) activateScene(sceneName string) {
	im.logger.WithField("scene", sceneName).Info("Activating scene")
	// 實作場景激活邏輯
}

// adjustSchedule 調整排程
func (im *InteractionManager) adjustSchedule(parameters map[string]interface{}) {
	im.logger.WithField("parameters", parameters).Info("Adjusting schedule")
	// 實作排程調整邏輯
}

// changeSecurityMode 變更安全模式
func (im *InteractionManager) changeSecurityMode(mode string) {
	im.logger.WithField("mode", mode).Info("Changing security mode")
	// 實作安全模式變更邏輯
}

// GetStatistics 獲取互動統計
func (im *InteractionManager) GetStatistics() map[string]interface{} {
	im.mu.RLock()
	defer im.mu.RUnlock()

	stats := map[string]interface{}{
		"total_rules":        len(im.interactions),
		"enabled_rules":      0,
		"registered_devices": len(im.devices),
		"running":            im.running,
	}

	for _, rule := range im.interactions {
		if rule.Enabled {
			stats["enabled_rules"] = stats["enabled_rules"].(int) + 1
		}
	}

	return stats
}
