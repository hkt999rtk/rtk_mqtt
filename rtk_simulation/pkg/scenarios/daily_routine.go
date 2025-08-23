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

// DailyRoutineManager 日常作息管理器
type DailyRoutineManager struct {
	routines       map[string]*DailyRoutine
	activeRoutines map[string]*ActiveRoutine
	devices        map[string]base.Device
	schedules      []*ScheduledEvent
	currentMode    string
	running        bool
	mu             sync.RWMutex
	logger         *logrus.Entry
	config         *RoutineConfig
}

// DailyRoutine 日常作息定義
type DailyRoutine struct {
	ID          string
	Name        string
	Type        string   // morning, daytime, evening, night
	StartTime   string   // HH:MM format
	EndTime     string   // HH:MM format
	Weekdays    []string // Mon, Tue, Wed, Thu, Fri, Sat, Sun
	Description string
	Actions     []RoutineAction
	Conditions  []RoutineCondition
	Priority    int
	Enabled     bool
}

// ActiveRoutine 活動中的作息
type ActiveRoutine struct {
	RoutineID       string
	InstanceID      string
	StartTime       time.Time
	EndTime         time.Time
	State           string // active, paused, completed
	ExecutedActions []string
	LastUpdate      time.Time
}

// RoutineAction 作息動作
type RoutineAction struct {
	ID         string
	Name       string
	DeviceType string // smart_bulb, air_conditioner, smart_plug, etc.
	DeviceID   string // specific device or "all" for device type
	Command    string
	Parameters map[string]interface{}
	Delay      time.Duration
	Condition  string // optional condition
}

// RoutineCondition 作息條件
type RoutineCondition struct {
	Type     string // time, weather, presence, temperature, etc.
	Operator string // gt, lt, eq, ne, ge, le
	Value    interface{}
	Duration time.Duration
}

// ScheduledEvent 排程事件
type ScheduledEvent struct {
	ID            string
	RoutineID     string
	ScheduledTime time.Time
	Executed      bool
	Result        string
}

// RoutineConfig 作息配置
type RoutineConfig struct {
	EnableAutoScheduling bool
	TimeZone             *time.Location
	DefaultTransition    time.Duration
	RandomVariation      time.Duration
	PresenceSimulation   bool
}

// PredefinedRoutines 預定義日常作息
var PredefinedRoutines = []DailyRoutine{
	{
		ID:          "morning_routine",
		Name:        "Morning Routine",
		Type:        "morning",
		StartTime:   "07:00",
		EndTime:     "09:00",
		Weekdays:    []string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"},
		Description: "Morning wake-up and preparation routine",
		Actions: []RoutineAction{
			{
				ID:         "morning_lights_on",
				Name:       "Turn on bedroom lights",
				DeviceType: "smart_bulb",
				Command:    "turn_on",
				Parameters: map[string]interface{}{
					"brightness": 30,
					"color_temp": "warm",
				},
				Delay: 0,
			},
			{
				ID:         "morning_ac_adjust",
				Name:       "Adjust air conditioner",
				DeviceType: "air_conditioner",
				Command:    "set_temperature",
				Parameters: map[string]interface{}{
					"temperature": 24,
					"mode":        "auto",
				},
				Delay: 2 * time.Minute,
			},
			{
				ID:         "morning_coffee",
				Name:       "Start coffee maker",
				DeviceType: "smart_plug",
				DeviceID:   "coffee_maker_plug",
				Command:    "turn_on",
				Delay:      5 * time.Minute,
			},
			{
				ID:         "morning_news",
				Name:       "Turn on TV for news",
				DeviceType: "smart_tv",
				Command:    "turn_on",
				Parameters: map[string]interface{}{
					"channel": "news",
					"volume":  20,
				},
				Delay: 10 * time.Minute,
			},
			{
				ID:         "morning_brightness_increase",
				Name:       "Gradually increase brightness",
				DeviceType: "smart_bulb",
				Command:    "set_brightness",
				Parameters: map[string]interface{}{
					"brightness": 80,
					"transition": 600, // 10 minutes
				},
				Delay: 15 * time.Minute,
			},
		},
		Priority: 10,
		Enabled:  true,
	},
	{
		ID:          "daytime_routine",
		Name:        "Daytime Energy Saving",
		Type:        "daytime",
		StartTime:   "09:00",
		EndTime:     "17:00",
		Weekdays:    []string{"Mon", "Tue", "Wed", "Thu", "Fri"},
		Description: "Energy saving and security monitoring during work hours",
		Actions: []RoutineAction{
			{
				ID:         "daytime_lights_off",
				Name:       "Turn off unnecessary lights",
				DeviceType: "smart_bulb",
				Command:    "turn_off",
				Delay:      0,
			},
			{
				ID:         "daytime_ac_eco",
				Name:       "Set AC to eco mode",
				DeviceType: "air_conditioner",
				Command:    "set_mode",
				Parameters: map[string]interface{}{
					"mode":        "eco",
					"temperature": 26,
				},
				Delay: 5 * time.Minute,
			},
			{
				ID:         "daytime_security_on",
				Name:       "Enable security cameras",
				DeviceType: "security_camera",
				Command:    "enable_monitoring",
				Parameters: map[string]interface{}{
					"motion_detection": true,
					"recording":        true,
				},
				Delay: 0,
			},
			{
				ID:         "daytime_blinds_adjust",
				Name:       "Adjust smart blinds",
				DeviceType: "smart_blinds",
				Command:    "set_position",
				Parameters: map[string]interface{}{
					"position": 50, // 50% open
				},
				Delay: 10 * time.Minute,
			},
		},
		Priority: 8,
		Enabled:  true,
	},
	{
		ID:          "evening_routine",
		Name:        "Evening Home",
		Type:        "evening",
		StartTime:   "17:00",
		EndTime:     "23:00",
		Weekdays:    []string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"},
		Description: "Evening comfort and entertainment settings",
		Actions: []RoutineAction{
			{
				ID:         "evening_lights_on",
				Name:       "Turn on living room lights",
				DeviceType: "smart_bulb",
				DeviceID:   "living_room",
				Command:    "turn_on",
				Parameters: map[string]interface{}{
					"brightness": 70,
					"color_temp": "warm",
				},
				Delay: 0,
			},
			{
				ID:         "evening_ac_comfort",
				Name:       "Set AC to comfort mode",
				DeviceType: "air_conditioner",
				Command:    "set_temperature",
				Parameters: map[string]interface{}{
					"temperature": 23,
					"mode":        "cool",
				},
				Delay: 5 * time.Minute,
			},
			{
				ID:         "evening_kitchen_lights",
				Name:       "Turn on kitchen lights",
				DeviceType: "smart_bulb",
				DeviceID:   "kitchen",
				Command:    "turn_on",
				Parameters: map[string]interface{}{
					"brightness": 90,
				},
				Delay: 30 * time.Minute,
			},
			{
				ID:         "evening_entertainment",
				Name:       "Setup entertainment system",
				DeviceType: "smart_tv",
				Command:    "turn_on",
				Parameters: map[string]interface{}{
					"input": "streaming",
				},
				Delay: 1 * time.Hour,
			},
			{
				ID:         "evening_ambient",
				Name:       "Set ambient lighting",
				DeviceType: "smart_bulb",
				Command:    "set_scene",
				Parameters: map[string]interface{}{
					"scene": "relax",
					"color": "orange",
				},
				Delay: 2 * time.Hour,
			},
		},
		Priority: 9,
		Enabled:  true,
	},
	{
		ID:          "night_routine",
		Name:        "Night Sleep Mode",
		Type:        "night",
		StartTime:   "23:00",
		EndTime:     "07:00",
		Weekdays:    []string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"},
		Description: "Night time security and sleep environment",
		Actions: []RoutineAction{
			{
				ID:         "night_lights_dim",
				Name:       "Dim all lights",
				DeviceType: "smart_bulb",
				Command:    "set_brightness",
				Parameters: map[string]interface{}{
					"brightness": 10,
					"transition": 300, // 5 minutes
				},
				Delay: 0,
			},
			{
				ID:         "night_tv_off",
				Name:       "Turn off TV",
				DeviceType: "smart_tv",
				Command:    "turn_off",
				Delay:      5 * time.Minute,
			},
			{
				ID:         "night_ac_sleep",
				Name:       "Set AC to sleep mode",
				DeviceType: "air_conditioner",
				Command:    "set_mode",
				Parameters: map[string]interface{}{
					"mode":        "sleep",
					"temperature": 25,
				},
				Delay: 10 * time.Minute,
			},
			{
				ID:         "night_security_arm",
				Name:       "Arm security system",
				DeviceType: "security_camera",
				Command:    "arm_system",
				Parameters: map[string]interface{}{
					"mode":   "night",
					"alerts": true,
				},
				Delay: 15 * time.Minute,
			},
			{
				ID:         "night_lights_off",
				Name:       "Turn off all lights",
				DeviceType: "smart_bulb",
				Command:    "turn_off",
				Delay:      30 * time.Minute,
			},
			{
				ID:         "night_pathway_light",
				Name:       "Enable pathway night light",
				DeviceType: "smart_bulb",
				DeviceID:   "pathway",
				Command:    "turn_on",
				Parameters: map[string]interface{}{
					"brightness": 5,
					"color":      "red",
				},
				Delay: 31 * time.Minute,
			},
		},
		Priority: 10,
		Enabled:  true,
	},
}

// NewDailyRoutineManager 創建新的日常作息管理器
func NewDailyRoutineManager(config *RoutineConfig) *DailyRoutineManager {
	if config == nil {
		loc, _ := time.LoadLocation("Local")
		config = &RoutineConfig{
			EnableAutoScheduling: true,
			TimeZone:             loc,
			DefaultTransition:    5 * time.Minute,
			RandomVariation:      10 * time.Minute,
			PresenceSimulation:   true,
		}
	}

	drm := &DailyRoutineManager{
		routines:       make(map[string]*DailyRoutine),
		activeRoutines: make(map[string]*ActiveRoutine),
		devices:        make(map[string]base.Device),
		schedules:      make([]*ScheduledEvent, 0),
		currentMode:    "auto",
		config:         config,
		logger:         logrus.WithField("component", "daily_routine_manager"),
	}

	// 載入預定義作息
	drm.loadPredefinedRoutines()

	return drm
}

// loadPredefinedRoutines 載入預定義作息
func (drm *DailyRoutineManager) loadPredefinedRoutines() {
	for _, routine := range PredefinedRoutines {
		r := routine // 避免迴圈變數問題
		drm.routines[r.ID] = &r
	}
	drm.logger.Infof("Loaded %d predefined daily routines", len(PredefinedRoutines))
}

// Start 啟動日常作息管理器
func (drm *DailyRoutineManager) Start(ctx context.Context) error {
	drm.mu.Lock()
	defer drm.mu.Unlock()

	if drm.running {
		return fmt.Errorf("daily routine manager is already running")
	}

	drm.running = true
	drm.logger.Info("Starting daily routine manager")

	// 啟動管理循環
	go drm.scheduleLoop(ctx)
	go drm.executionLoop(ctx)
	go drm.monitorLoop(ctx)

	return nil
}

// Stop 停止日常作息管理器
func (drm *DailyRoutineManager) Stop() error {
	drm.mu.Lock()
	defer drm.mu.Unlock()

	if !drm.running {
		return fmt.Errorf("daily routine manager is not running")
	}

	drm.running = false
	drm.logger.Info("Stopping daily routine manager")
	return nil
}

// RegisterDevice 註冊設備
func (drm *DailyRoutineManager) RegisterDevice(deviceID string, device base.Device) {
	drm.mu.Lock()
	defer drm.mu.Unlock()

	drm.devices[deviceID] = device
	drm.logger.WithField("device_id", deviceID).Debug("Device registered for daily routines")
}

// scheduleLoop 排程循環
func (drm *DailyRoutineManager) scheduleLoop(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			drm.updateSchedules()
		}
	}
}

// updateSchedules 更新排程
func (drm *DailyRoutineManager) updateSchedules() {
	drm.mu.Lock()
	defer drm.mu.Unlock()

	now := time.Now().In(drm.config.TimeZone)
	weekday := now.Format("Mon")
	currentTime := now.Format("15:04")

	for _, routine := range drm.routines {
		if !routine.Enabled {
			continue
		}

		// 檢查星期
		if !drm.isWeekdayMatch(weekday, routine.Weekdays) {
			continue
		}

		// 檢查時間
		if drm.isTimeInRange(currentTime, routine.StartTime, routine.EndTime) {
			// 檢查是否已經在執行
			if !drm.isRoutineActive(routine.ID) {
				drm.startRoutine(routine)
			}
		}
	}
}

// isWeekdayMatch 檢查星期是否匹配
func (drm *DailyRoutineManager) isWeekdayMatch(weekday string, weekdays []string) bool {
	for _, w := range weekdays {
		if w == weekday {
			return true
		}
	}
	return false
}

// isTimeInRange 檢查時間是否在範圍內
func (drm *DailyRoutineManager) isTimeInRange(current, start, end string) bool {
	// 簡單的時間比較
	if start <= end {
		return current >= start && current < end
	}
	// 跨天的情況（如 23:00 到 07:00）
	return current >= start || current < end
}

// isRoutineActive 檢查作息是否活動中
func (drm *DailyRoutineManager) isRoutineActive(routineID string) bool {
	for _, active := range drm.activeRoutines {
		if active.RoutineID == routineID && active.State == "active" {
			return true
		}
	}
	return false
}

// startRoutine 開始作息
func (drm *DailyRoutineManager) startRoutine(routine *DailyRoutine) {
	instanceID := fmt.Sprintf("%s_%d", routine.ID, time.Now().UnixNano())

	// 添加隨機變化
	variation := time.Duration(0)
	if drm.config.RandomVariation > 0 {
		variation = time.Duration(rand.Int63n(int64(drm.config.RandomVariation)))
	}

	active := &ActiveRoutine{
		RoutineID:       routine.ID,
		InstanceID:      instanceID,
		StartTime:       time.Now().Add(variation),
		State:           "active",
		ExecutedActions: make([]string, 0),
		LastUpdate:      time.Now(),
	}

	drm.activeRoutines[instanceID] = active

	drm.logger.WithFields(logrus.Fields{
		"routine_id":   routine.ID,
		"routine_name": routine.Name,
		"instance_id":  instanceID,
	}).Info("Daily routine started")

	// 執行動作
	go drm.executeRoutineActions(routine, active)
}

// executeRoutineActions 執行作息動作
func (drm *DailyRoutineManager) executeRoutineActions(routine *DailyRoutine, active *ActiveRoutine) {
	for _, action := range routine.Actions {
		// 等待延遲
		if action.Delay > 0 {
			time.Sleep(action.Delay)
		}

		// 檢查是否仍然活動
		drm.mu.RLock()
		if active.State != "active" {
			drm.mu.RUnlock()
			break
		}
		drm.mu.RUnlock()

		// 執行動作
		drm.executeAction(action)

		// 記錄已執行的動作
		drm.mu.Lock()
		active.ExecutedActions = append(active.ExecutedActions, action.ID)
		active.LastUpdate = time.Now()
		drm.mu.Unlock()
	}

	// 完成作息
	drm.mu.Lock()
	active.State = "completed"
	active.EndTime = time.Now()
	drm.mu.Unlock()

	drm.logger.WithField("routine_id", routine.ID).Info("Daily routine completed")
}

// executeAction 執行動作
func (drm *DailyRoutineManager) executeAction(action RoutineAction) {
	drm.mu.RLock()
	defer drm.mu.RUnlock()

	// 查找目標設備
	targetDevices := drm.findTargetDevices(action)

	for _, device := range targetDevices {
		cmd := base.Command{
			ID:         fmt.Sprintf("routine_%s_%d", action.ID, time.Now().UnixNano()),
			Type:       action.Command,
			Parameters: action.Parameters,
			Timeout:    10 * time.Second,
		}

		err := device.HandleCommand(cmd)
		if err != nil {
			drm.logger.WithError(err).WithFields(logrus.Fields{
				"action_id":   action.ID,
				"device_id":   device.GetDeviceID(),
				"device_type": device.GetDeviceType(),
			}).Error("Failed to execute routine action")
		} else {
			drm.logger.WithFields(logrus.Fields{
				"action_id":   action.ID,
				"action_name": action.Name,
				"device_id":   device.GetDeviceID(),
			}).Debug("Routine action executed")
		}
	}
}

// findTargetDevices 查找目標設備
func (drm *DailyRoutineManager) findTargetDevices(action RoutineAction) []base.Device {
	devices := make([]base.Device, 0)

	if action.DeviceID != "" && action.DeviceID != "all" {
		// 特定設備
		if device, exists := drm.devices[action.DeviceID]; exists {
			devices = append(devices, device)
		}
	} else if action.DeviceType != "" {
		// 設備類型
		for _, device := range drm.devices {
			if device.GetDeviceType() == action.DeviceType {
				devices = append(devices, device)
			}
		}
	}

	return devices
}

// executionLoop 執行循環
func (drm *DailyRoutineManager) executionLoop(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			drm.checkActiveRoutines()
		}
	}
}

// checkActiveRoutines 檢查活動作息
func (drm *DailyRoutineManager) checkActiveRoutines() {
	drm.mu.Lock()
	defer drm.mu.Unlock()

	completed := make([]string, 0)

	for instanceID, active := range drm.activeRoutines {
		if active.State == "completed" {
			// 清理完成的作息
			if time.Since(active.EndTime) > 5*time.Minute {
				completed = append(completed, instanceID)
			}
		}
	}

	// 刪除完成的作息
	for _, instanceID := range completed {
		delete(drm.activeRoutines, instanceID)
	}
}

// monitorLoop 監控循環
func (drm *DailyRoutineManager) monitorLoop(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			drm.updateCurrentMode()
		}
	}
}

// updateCurrentMode 更新當前模式
func (drm *DailyRoutineManager) updateCurrentMode() {
	drm.mu.Lock()
	defer drm.mu.Unlock()

	now := time.Now().In(drm.config.TimeZone)
	hour := now.Hour()

	// 根據時間判斷當前模式
	if hour >= 7 && hour < 9 {
		drm.currentMode = "morning"
	} else if hour >= 9 && hour < 17 {
		drm.currentMode = "daytime"
	} else if hour >= 17 && hour < 23 {
		drm.currentMode = "evening"
	} else {
		drm.currentMode = "night"
	}

	drm.logger.WithField("mode", drm.currentMode).Debug("Current mode updated")
}

// GetCurrentMode 獲取當前模式
func (drm *DailyRoutineManager) GetCurrentMode() string {
	drm.mu.RLock()
	defer drm.mu.RUnlock()
	return drm.currentMode
}

// GetActiveRoutines 獲取活動作息
func (drm *DailyRoutineManager) GetActiveRoutines() []*ActiveRoutine {
	drm.mu.RLock()
	defer drm.mu.RUnlock()

	routines := make([]*ActiveRoutine, 0, len(drm.activeRoutines))
	for _, routine := range drm.activeRoutines {
		routines = append(routines, routine)
	}
	return routines
}

// ManualTriggerRoutine 手動觸發作息
func (drm *DailyRoutineManager) ManualTriggerRoutine(routineID string) error {
	drm.mu.RLock()
	routine, exists := drm.routines[routineID]
	drm.mu.RUnlock()

	if !exists {
		return fmt.Errorf("routine %s not found", routineID)
	}

	drm.mu.Lock()
	defer drm.mu.Unlock()

	if drm.isRoutineActive(routineID) {
		return fmt.Errorf("routine %s is already active", routineID)
	}

	drm.startRoutine(routine)
	return nil
}

// PauseRoutine 暫停作息
func (drm *DailyRoutineManager) PauseRoutine(instanceID string) error {
	drm.mu.Lock()
	defer drm.mu.Unlock()

	active, exists := drm.activeRoutines[instanceID]
	if !exists {
		return fmt.Errorf("active routine %s not found", instanceID)
	}

	active.State = "paused"
	active.LastUpdate = time.Now()

	drm.logger.WithField("instance_id", instanceID).Info("Routine paused")
	return nil
}

// ResumeRoutine 恢復作息
func (drm *DailyRoutineManager) ResumeRoutine(instanceID string) error {
	drm.mu.Lock()
	defer drm.mu.Unlock()

	active, exists := drm.activeRoutines[instanceID]
	if !exists {
		return fmt.Errorf("active routine %s not found", instanceID)
	}

	if active.State != "paused" {
		return fmt.Errorf("routine %s is not paused", instanceID)
	}

	active.State = "active"
	active.LastUpdate = time.Now()

	drm.logger.WithField("instance_id", instanceID).Info("Routine resumed")
	return nil
}

// GetStatistics 獲取統計資訊
func (drm *DailyRoutineManager) GetStatistics() map[string]interface{} {
	drm.mu.RLock()
	defer drm.mu.RUnlock()

	stats := map[string]interface{}{
		"total_routines":  len(drm.routines),
		"active_routines": len(drm.activeRoutines),
		"current_mode":    drm.currentMode,
		"enabled_count":   0,
		"executed_today":  0,
	}

	// 統計啟用的作息
	for _, routine := range drm.routines {
		if routine.Enabled {
			stats["enabled_count"] = stats["enabled_count"].(int) + 1
		}
	}

	// 統計今日執行的作息
	today := time.Now().Truncate(24 * time.Hour)
	for _, active := range drm.activeRoutines {
		if active.StartTime.After(today) {
			stats["executed_today"] = stats["executed_today"].(int) + 1
		}
	}

	return stats
}
