package scenarios

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"rtk_simulation/pkg/devices/base"
)

// AutomationManager 自動化管理器
type AutomationManager struct {
	rules          map[string]*AutomationRule
	activeRules    map[string]*ActiveAutomation
	scenes         map[string]*Scene
	activeScenes   map[string]*ActiveScene
	devices        map[string]base.Device
	eventBus       *EventBus
	conditionCache map[string]interface{}
	running        bool
	mu             sync.RWMutex
	logger         *logrus.Entry
	config         *AutomationConfig
}

// AutomationRule 自動化規則
type AutomationRule struct {
	ID            string
	Name          string
	Description   string
	Triggers      []Trigger
	Conditions    []Condition
	Actions       []Action
	Enabled       bool
	Priority      int
	Cooldown      time.Duration
	LastTriggered time.Time
}

// Trigger 觸發器
type Trigger struct {
	ID       string
	Type     string // time, device_state, sensor, event, manual
	Source   string // device ID or event source
	Event    string
	Value    interface{}
	Operator string // eq, ne, gt, lt, ge, le, contains, matches
}

// Condition 條件
type Condition struct {
	ID       string
	Type     string // time, device_state, weather, presence, custom
	Source   string
	Property string
	Operator string
	Value    interface{}
	Logic    string // and, or, not
}

// Action 動作
type Action struct {
	ID         string
	Type       string // device_control, scene_activation, notification, delay
	Target     string // device ID or scene ID
	Command    string
	Parameters map[string]interface{}
	Delay      time.Duration
}

// Scene 場景
type Scene struct {
	ID           string
	Name         string
	Description  string
	Type         string // lighting, climate, security, entertainment, custom
	DeviceStates []DeviceState
	Transitions  []Transition
	Duration     time.Duration
	Enabled      bool
}

// DeviceState 設備狀態
type DeviceState struct {
	DeviceID   string
	DeviceType string
	State      string
	Properties map[string]interface{}
}

// Transition 轉換
type Transition struct {
	From     string
	To       string
	Duration time.Duration
	Curve    string // linear, ease-in, ease-out, ease-in-out
}

// ActiveAutomation 活動中的自動化
type ActiveAutomation struct {
	RuleID          string
	InstanceID      string
	TriggerTime     time.Time
	State           string // triggered, executing, completed, failed
	ExecutedActions []string
	FailedActions   []string
	LastUpdate      time.Time
}

// ActiveScene 活動中的場景
type ActiveScene struct {
	SceneID      string
	InstanceID   string
	StartTime    time.Time
	EndTime      time.Time
	State        string // active, transitioning, completed
	CurrentStep  int
	DeviceStates map[string]string
	LastUpdate   time.Time
}

// AutomationConfig 自動化配置
type AutomationConfig struct {
	EnableAutomation    bool
	EvaluationInterval  time.Duration
	MaxConcurrentRules  int
	EnableSceneBlending bool
	EventQueueSize      int
}

// PredefinedRules 預定義自動化規則
var PredefinedRules = []AutomationRule{
	{
		ID:          "motion_lighting",
		Name:        "Motion Activated Lighting",
		Description: "Turn on lights when motion is detected",
		Triggers: []Trigger{
			{
				ID:     "motion_trigger",
				Type:   "sensor",
				Source: "motion_sensor",
				Event:  "motion_detected",
				Value:  true,
			},
		},
		Conditions: []Condition{
			{
				ID:       "time_condition",
				Type:     "time",
				Property: "hour",
				Operator: "ge",
				Value:    18, // After 6 PM
				Logic:    "and",
			},
			{
				ID:       "time_condition_2",
				Type:     "time",
				Property: "hour",
				Operator: "le",
				Value:    6, // Before 6 AM
				Logic:    "or",
			},
		},
		Actions: []Action{
			{
				ID:      "turn_on_lights",
				Type:    "device_control",
				Target:  "smart_bulb",
				Command: "turn_on",
				Parameters: map[string]interface{}{
					"brightness": 80,
					"transition": 2,
				},
			},
			{
				ID:      "auto_off_timer",
				Type:    "delay",
				Command: "wait",
				Delay:   10 * time.Minute,
			},
			{
				ID:      "turn_off_lights",
				Type:    "device_control",
				Target:  "smart_bulb",
				Command: "turn_off",
				Parameters: map[string]interface{}{
					"transition": 5,
				},
			},
		},
		Enabled:  true,
		Priority: 8,
		Cooldown: 1 * time.Minute,
	},
	{
		ID:          "temperature_control",
		Name:        "Smart Temperature Control",
		Description: "Adjust AC based on temperature and presence",
		Triggers: []Trigger{
			{
				ID:       "temp_trigger",
				Type:     "sensor",
				Source:   "temperature_sensor",
				Event:    "temperature_change",
				Value:    28,
				Operator: "gt",
			},
		},
		Conditions: []Condition{
			{
				ID:       "presence_condition",
				Type:     "presence",
				Property: "occupied",
				Operator: "eq",
				Value:    true,
				Logic:    "and",
			},
		},
		Actions: []Action{
			{
				ID:      "adjust_ac",
				Type:    "device_control",
				Target:  "air_conditioner",
				Command: "set_temperature",
				Parameters: map[string]interface{}{
					"temperature": 24,
					"mode":        "cool",
					"fan_speed":   "auto",
				},
			},
		},
		Enabled:  true,
		Priority: 7,
		Cooldown: 5 * time.Minute,
	},
	{
		ID:          "security_alert",
		Name:        "Security Alert System",
		Description: "Activate security measures on intrusion",
		Triggers: []Trigger{
			{
				ID:     "door_trigger",
				Type:   "device_state",
				Source: "door_sensor",
				Event:  "state_change",
				Value:  "open",
			},
			{
				ID:     "window_trigger",
				Type:   "device_state",
				Source: "window_sensor",
				Event:  "state_change",
				Value:  "open",
			},
		},
		Conditions: []Condition{
			{
				ID:       "armed_condition",
				Type:     "device_state",
				Source:   "security_system",
				Property: "armed",
				Operator: "eq",
				Value:    true,
				Logic:    "and",
			},
		},
		Actions: []Action{
			{
				ID:      "activate_alarm",
				Type:    "device_control",
				Target:  "security_alarm",
				Command: "trigger",
				Parameters: map[string]interface{}{
					"volume": 100,
					"type":   "intrusion",
				},
			},
			{
				ID:      "turn_on_all_lights",
				Type:    "scene_activation",
				Target:  "all_lights_on",
				Command: "activate",
			},
			{
				ID:      "send_notification",
				Type:    "notification",
				Command: "send",
				Parameters: map[string]interface{}{
					"message":  "Security breach detected!",
					"priority": "high",
				},
			},
		},
		Enabled:  true,
		Priority: 10,
		Cooldown: 10 * time.Second,
	},
	{
		ID:          "energy_saving",
		Name:        "Energy Saving Mode",
		Description: "Optimize energy usage when away",
		Triggers: []Trigger{
			{
				ID:       "presence_trigger",
				Type:     "presence",
				Event:    "all_away",
				Value:    true,
				Operator: "eq",
			},
		},
		Conditions: []Condition{},
		Actions: []Action{
			{
				ID:      "lights_off",
				Type:    "device_control",
				Target:  "all_lights",
				Command: "turn_off",
			},
			{
				ID:      "ac_eco_mode",
				Type:    "device_control",
				Target:  "air_conditioner",
				Command: "set_mode",
				Parameters: map[string]interface{}{
					"mode":        "eco",
					"temperature": 28,
				},
			},
			{
				ID:      "appliances_standby",
				Type:    "device_control",
				Target:  "smart_plug",
				Command: "standby",
			},
		},
		Enabled:  true,
		Priority: 6,
		Cooldown: 10 * time.Minute,
	},
}

// PredefinedScenes 預定義場景
var PredefinedScenes = []Scene{
	{
		ID:          "movie_night",
		Name:        "Movie Night",
		Description: "Optimal settings for watching movies",
		Type:        "entertainment",
		DeviceStates: []DeviceState{
			{
				DeviceType: "smart_bulb",
				State:      "on",
				Properties: map[string]interface{}{
					"brightness": 20,
					"color":      "warm",
				},
			},
			{
				DeviceType: "smart_tv",
				State:      "on",
				Properties: map[string]interface{}{
					"input":        "streaming",
					"picture_mode": "cinema",
				},
			},
			{
				DeviceType: "air_conditioner",
				State:      "on",
				Properties: map[string]interface{}{
					"temperature": 23,
					"mode":        "quiet",
				},
			},
		},
		Transitions: []Transition{
			{
				From:     "current",
				To:       "movie_night",
				Duration: 10 * time.Second,
				Curve:    "ease-in-out",
			},
		},
		Duration: 3 * time.Hour,
		Enabled:  true,
	},
	{
		ID:          "good_morning",
		Name:        "Good Morning",
		Description: "Wake up routine",
		Type:        "lighting",
		DeviceStates: []DeviceState{
			{
				DeviceType: "smart_bulb",
				DeviceID:   "bedroom",
				State:      "on",
				Properties: map[string]interface{}{
					"brightness": 30,
					"color_temp": 3000,
				},
			},
			{
				DeviceType: "smart_blinds",
				State:      "open",
				Properties: map[string]interface{}{
					"position": 100,
				},
			},
			{
				DeviceType: "air_conditioner",
				State:      "on",
				Properties: map[string]interface{}{
					"temperature": 24,
					"mode":        "auto",
				},
			},
		},
		Transitions: []Transition{
			{
				From:     "night",
				To:       "morning",
				Duration: 5 * time.Minute,
				Curve:    "ease-in",
			},
		},
		Duration: 30 * time.Minute,
		Enabled:  true,
	},
	{
		ID:          "romantic_dinner",
		Name:        "Romantic Dinner",
		Description: "Perfect ambiance for dinner",
		Type:        "lighting",
		DeviceStates: []DeviceState{
			{
				DeviceType: "smart_bulb",
				DeviceID:   "dining_room",
				State:      "on",
				Properties: map[string]interface{}{
					"brightness": 40,
					"color":      "warm_white",
				},
			},
			{
				DeviceType: "smart_speaker",
				State:      "on",
				Properties: map[string]interface{}{
					"playlist": "romantic",
					"volume":   30,
				},
			},
		},
		Duration: 2 * time.Hour,
		Enabled:  true,
	},
	{
		ID:          "all_lights_on",
		Name:        "All Lights On",
		Description: "Turn on all lights",
		Type:        "lighting",
		DeviceStates: []DeviceState{
			{
				DeviceType: "smart_bulb",
				State:      "on",
				Properties: map[string]interface{}{
					"brightness": 100,
				},
			},
		},
		Duration: 0,
		Enabled:  true,
	},
}

// NewAutomationManager 創建新的自動化管理器
func NewAutomationManager(config *AutomationConfig) *AutomationManager {
	if config == nil {
		config = &AutomationConfig{
			EnableAutomation:    true,
			EvaluationInterval:  1 * time.Second,
			MaxConcurrentRules:  10,
			EnableSceneBlending: true,
			EventQueueSize:      1000,
		}
	}

	am := &AutomationManager{
		rules:          make(map[string]*AutomationRule),
		activeRules:    make(map[string]*ActiveAutomation),
		scenes:         make(map[string]*Scene),
		activeScenes:   make(map[string]*ActiveScene),
		devices:        make(map[string]base.Device),
		conditionCache: make(map[string]interface{}),
		config:         config,
		logger:         logrus.WithField("component", "automation_manager"),
	}

	// 初始化事件總線
	am.eventBus = NewEventBus(config.EventQueueSize)

	// 載入預定義規則和場景
	am.loadPredefinedRules()
	am.loadPredefinedScenes()

	return am
}

// loadPredefinedRules 載入預定義規則
func (am *AutomationManager) loadPredefinedRules() {
	for _, rule := range PredefinedRules {
		r := rule
		am.rules[r.ID] = &r
	}
	am.logger.Infof("Loaded %d predefined automation rules", len(PredefinedRules))
}

// loadPredefinedScenes 載入預定義場景
func (am *AutomationManager) loadPredefinedScenes() {
	for _, scene := range PredefinedScenes {
		s := scene
		am.scenes[s.ID] = &s
	}
	am.logger.Infof("Loaded %d predefined scenes", len(PredefinedScenes))
}

// Start 啟動自動化管理器
func (am *AutomationManager) Start(ctx context.Context) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	if am.running {
		return fmt.Errorf("automation manager is already running")
	}

	am.running = true
	am.logger.Info("Starting automation manager")

	// 啟動事件總線
	am.eventBus.Start(ctx)

	// 啟動管理循環
	go am.evaluationLoop(ctx)
	go am.executionLoop(ctx)
	go am.sceneManagementLoop(ctx)

	return nil
}

// Stop 停止自動化管理器
func (am *AutomationManager) Stop() error {
	am.mu.Lock()
	defer am.mu.Unlock()

	if !am.running {
		return fmt.Errorf("automation manager is not running")
	}

	am.running = false
	am.eventBus.Stop()
	am.logger.Info("Stopping automation manager")
	return nil
}

// RegisterDevice 註冊設備
func (am *AutomationManager) RegisterDevice(deviceID string, device base.Device) {
	am.mu.Lock()
	defer am.mu.Unlock()

	am.devices[deviceID] = device
	am.logger.WithField("device_id", deviceID).Debug("Device registered for automation")
}

// evaluationLoop 評估循環
func (am *AutomationManager) evaluationLoop(ctx context.Context) {
	ticker := time.NewTicker(am.config.EvaluationInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			am.evaluateRules()
		}
	}
}

// evaluateRules 評估規則
func (am *AutomationManager) evaluateRules() {
	am.mu.RLock()
	defer am.mu.RUnlock()

	for _, rule := range am.rules {
		if !rule.Enabled {
			continue
		}

		// 檢查冷卻時間
		if time.Since(rule.LastTriggered) < rule.Cooldown {
			continue
		}

		// 檢查觸發器
		if am.checkTriggers(rule.Triggers) {
			// 檢查條件
			if am.checkConditions(rule.Conditions) {
				// 觸發規則
				go am.triggerRule(rule)
			}
		}
	}
}

// checkTriggers 檢查觸發器
func (am *AutomationManager) checkTriggers(triggers []Trigger) bool {
	for _, trigger := range triggers {
		if am.evaluateTrigger(trigger) {
			return true // OR 邏輯
		}
	}
	return false
}

// evaluateTrigger 評估觸發器
func (am *AutomationManager) evaluateTrigger(trigger Trigger) bool {
	// 從事件總線或緩存中獲取值
	currentValue := am.getCurrentValue(trigger.Source, trigger.Event)

	return am.compareValues(currentValue, trigger.Value, trigger.Operator)
}

// checkConditions 檢查條件
func (am *AutomationManager) checkConditions(conditions []Condition) bool {
	if len(conditions) == 0 {
		return true
	}

	result := true
	for _, condition := range conditions {
		conditionMet := am.evaluateCondition(condition)

		switch condition.Logic {
		case "and":
			result = result && conditionMet
		case "or":
			result = result || conditionMet
		case "not":
			result = result && !conditionMet
		default:
			result = result && conditionMet
		}
	}
	return result
}

// evaluateCondition 評估條件
func (am *AutomationManager) evaluateCondition(condition Condition) bool {
	currentValue := am.getCurrentValue(condition.Source, condition.Property)
	return am.compareValues(currentValue, condition.Value, condition.Operator)
}

// getCurrentValue 獲取當前值
func (am *AutomationManager) getCurrentValue(source, property string) interface{} {
	// 從緩存或設備獲取當前值
	cacheKey := fmt.Sprintf("%s.%s", source, property)
	if value, exists := am.conditionCache[cacheKey]; exists {
		return value
	}

	// 特殊處理時間條件
	if source == "" && property == "hour" {
		return time.Now().Hour()
	}

	// 從設備獲取
	if device, exists := am.devices[source]; exists {
		// 這裡需要根據設備類型獲取屬性
		return device.GetHealth() // 簡化示例
	}

	return nil
}

// compareValues 比較值
func (am *AutomationManager) compareValues(current, target interface{}, operator string) bool {
	switch operator {
	case "eq":
		return current == target
	case "ne":
		return current != target
	case "gt":
		return am.compareNumeric(current, target) > 0
	case "lt":
		return am.compareNumeric(current, target) < 0
	case "ge":
		return am.compareNumeric(current, target) >= 0
	case "le":
		return am.compareNumeric(current, target) <= 0
	default:
		return false
	}
}

// compareNumeric 數值比較
func (am *AutomationManager) compareNumeric(a, b interface{}) float64 {
	var aFloat, bFloat float64

	switch v := a.(type) {
	case int:
		aFloat = float64(v)
	case float64:
		aFloat = v
	default:
		return 0
	}

	switch v := b.(type) {
	case int:
		bFloat = float64(v)
	case float64:
		bFloat = v
	default:
		return 0
	}

	return aFloat - bFloat
}

// triggerRule 觸發規則
func (am *AutomationManager) triggerRule(rule *AutomationRule) {
	am.mu.Lock()
	rule.LastTriggered = time.Now()
	am.mu.Unlock()

	instanceID := fmt.Sprintf("%s_%d", rule.ID, time.Now().UnixNano())

	active := &ActiveAutomation{
		RuleID:          rule.ID,
		InstanceID:      instanceID,
		TriggerTime:     time.Now(),
		State:           "triggered",
		ExecutedActions: make([]string, 0),
		FailedActions:   make([]string, 0),
		LastUpdate:      time.Now(),
	}

	am.mu.Lock()
	am.activeRules[instanceID] = active
	am.mu.Unlock()

	am.logger.WithFields(logrus.Fields{
		"rule_id":   rule.ID,
		"rule_name": rule.Name,
	}).Info("Automation rule triggered")

	// 執行動作
	am.executeActions(rule.Actions, active)

	// 更新狀態
	am.mu.Lock()
	active.State = "completed"
	active.LastUpdate = time.Now()
	am.mu.Unlock()
}

// executeActions 執行動作
func (am *AutomationManager) executeActions(actions []Action, active *ActiveAutomation) {
	for _, action := range actions {
		// 處理延遲
		if action.Delay > 0 {
			time.Sleep(action.Delay)
		}

		err := am.executeAction(action)

		am.mu.Lock()
		if err != nil {
			active.FailedActions = append(active.FailedActions, action.ID)
			am.logger.WithError(err).WithField("action_id", action.ID).Error("Failed to execute action")
		} else {
			active.ExecutedActions = append(active.ExecutedActions, action.ID)
		}
		active.LastUpdate = time.Now()
		am.mu.Unlock()
	}
}

// executeAction 執行單個動作
func (am *AutomationManager) executeAction(action Action) error {
	switch action.Type {
	case "device_control":
		return am.executeDeviceControl(action)
	case "scene_activation":
		return am.activateScene(action.Target)
	case "notification":
		return am.sendNotification(action)
	case "delay":
		time.Sleep(action.Delay)
		return nil
	default:
		return fmt.Errorf("unknown action type: %s", action.Type)
	}
}

// executeDeviceControl 執行設備控制
func (am *AutomationManager) executeDeviceControl(action Action) error {
	am.mu.RLock()
	defer am.mu.RUnlock()

	// 查找目標設備
	if action.Target == "all_lights" {
		// 控制所有燈光
		for _, device := range am.devices {
			if device.GetDeviceType() == "smart_bulb" {
				cmd := base.Command{
					ID:         fmt.Sprintf("auto_%s_%d", action.Command, time.Now().UnixNano()),
					Type:       action.Command,
					Parameters: action.Parameters,
					Timeout:    5 * time.Second,
				}
				device.HandleCommand(cmd)
			}
		}
		return nil
	}

	// 特定設備
	device, exists := am.devices[action.Target]
	if !exists {
		// 嘗試按類型查找
		for _, d := range am.devices {
			if d.GetDeviceType() == action.Target {
				device = d
				break
			}
		}
		if device == nil {
			return fmt.Errorf("device %s not found", action.Target)
		}
	}

	cmd := base.Command{
		ID:         fmt.Sprintf("auto_%s_%d", action.Command, time.Now().UnixNano()),
		Type:       action.Command,
		Parameters: action.Parameters,
		Timeout:    5 * time.Second,
	}

	return device.HandleCommand(cmd)
}

// sendNotification 發送通知
func (am *AutomationManager) sendNotification(action Action) error {
	am.logger.WithFields(logrus.Fields{
		"message":  action.Parameters["message"],
		"priority": action.Parameters["priority"],
	}).Info("Sending notification")
	return nil
}

// activateScene 啟動場景
func (am *AutomationManager) activateScene(sceneID string) error {
	am.mu.RLock()
	scene, exists := am.scenes[sceneID]
	am.mu.RUnlock()

	if !exists {
		return fmt.Errorf("scene %s not found", sceneID)
	}

	if !scene.Enabled {
		return fmt.Errorf("scene %s is disabled", sceneID)
	}

	instanceID := fmt.Sprintf("%s_%d", sceneID, time.Now().UnixNano())

	active := &ActiveScene{
		SceneID:      sceneID,
		InstanceID:   instanceID,
		StartTime:    time.Now(),
		State:        "active",
		CurrentStep:  0,
		DeviceStates: make(map[string]string),
		LastUpdate:   time.Now(),
	}

	if scene.Duration > 0 {
		active.EndTime = time.Now().Add(scene.Duration)
	}

	am.mu.Lock()
	am.activeScenes[instanceID] = active
	am.mu.Unlock()

	am.logger.WithFields(logrus.Fields{
		"scene_id":   sceneID,
		"scene_name": scene.Name,
	}).Info("Scene activated")

	// 應用場景設置
	go am.applySceneSettings(scene, active)

	return nil
}

// applySceneSettings 應用場景設置
func (am *AutomationManager) applySceneSettings(scene *Scene, active *ActiveScene) {
	// 應用轉換
	if len(scene.Transitions) > 0 {
		transition := scene.Transitions[0]
		am.applyTransition(transition)
	}

	// 應用設備狀態
	for _, deviceState := range scene.DeviceStates {
		am.applyDeviceState(deviceState, active)
	}

	// 更新狀態
	am.mu.Lock()
	active.State = "completed"
	active.LastUpdate = time.Now()
	am.mu.Unlock()
}

// applyTransition 應用轉換
func (am *AutomationManager) applyTransition(transition Transition) {
	// 簡化的轉換實現
	steps := int(transition.Duration.Seconds())
	interval := transition.Duration / time.Duration(steps)

	for i := 0; i < steps; i++ {
		// 這裡可以實現漸變效果
		time.Sleep(interval)
	}
}

// applyDeviceState 應用設備狀態
func (am *AutomationManager) applyDeviceState(deviceState DeviceState, active *ActiveScene) {
	am.mu.RLock()
	defer am.mu.RUnlock()

	// 查找目標設備
	for _, device := range am.devices {
		if (deviceState.DeviceID != "" && device.GetDeviceID() == deviceState.DeviceID) ||
			(deviceState.DeviceType != "" && device.GetDeviceType() == deviceState.DeviceType) {

			// 構建命令
			cmd := base.Command{
				ID:         fmt.Sprintf("scene_%d", time.Now().UnixNano()),
				Type:       deviceState.State,
				Parameters: deviceState.Properties,
				Timeout:    5 * time.Second,
			}

			err := device.HandleCommand(cmd)
			if err == nil {
				active.DeviceStates[device.GetDeviceID()] = deviceState.State
			}
		}
	}
}

// executionLoop 執行循環
func (am *AutomationManager) executionLoop(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			am.cleanupCompletedAutomations()
		}
	}
}

// cleanupCompletedAutomations 清理完成的自動化
func (am *AutomationManager) cleanupCompletedAutomations() {
	am.mu.Lock()
	defer am.mu.Unlock()

	// 清理完成的規則
	for id, active := range am.activeRules {
		if active.State == "completed" && time.Since(active.TriggerTime) > 5*time.Minute {
			delete(am.activeRules, id)
		}
	}
}

// sceneManagementLoop 場景管理循環
func (am *AutomationManager) sceneManagementLoop(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			am.checkActiveScenes()
		}
	}
}

// checkActiveScenes 檢查活動場景
func (am *AutomationManager) checkActiveScenes() {
	am.mu.Lock()
	defer am.mu.Unlock()

	now := time.Now()

	for id, active := range am.activeScenes {
		// 檢查是否過期
		if !active.EndTime.IsZero() && now.After(active.EndTime) {
			active.State = "completed"
		}

		// 清理完成的場景
		if active.State == "completed" && time.Since(active.StartTime) > 10*time.Minute {
			delete(am.activeScenes, id)
		}
	}
}

// PublishEvent 發布事件
func (am *AutomationManager) PublishEvent(event Event) {
	am.eventBus.Publish(event)

	// 更新條件緩存
	am.mu.Lock()
	cacheKey := fmt.Sprintf("%s.%s", event.Source, event.Type)
	am.conditionCache[cacheKey] = event.Data
	am.mu.Unlock()
}

// GetActiveRules 獲取活動規則
func (am *AutomationManager) GetActiveRules() []*ActiveAutomation {
	am.mu.RLock()
	defer am.mu.RUnlock()

	rules := make([]*ActiveAutomation, 0, len(am.activeRules))
	for _, rule := range am.activeRules {
		rules = append(rules, rule)
	}
	return rules
}

// GetActiveScenes 獲取活動場景
func (am *AutomationManager) GetActiveScenes() []*ActiveScene {
	am.mu.RLock()
	defer am.mu.RUnlock()

	scenes := make([]*ActiveScene, 0, len(am.activeScenes))
	for _, scene := range am.activeScenes {
		scenes = append(scenes, scene)
	}
	return scenes
}

// GetStatistics 獲取統計資訊
func (am *AutomationManager) GetStatistics() map[string]interface{} {
	am.mu.RLock()
	defer am.mu.RUnlock()

	stats := map[string]interface{}{
		"total_rules":      len(am.rules),
		"enabled_rules":    0,
		"active_rules":     len(am.activeRules),
		"total_scenes":     len(am.scenes),
		"active_scenes":    len(am.activeScenes),
		"event_queue_size": am.eventBus.QueueSize(),
	}

	// 統計啟用的規則
	for _, rule := range am.rules {
		if rule.Enabled {
			stats["enabled_rules"] = stats["enabled_rules"].(int) + 1
		}
	}

	return stats
}
