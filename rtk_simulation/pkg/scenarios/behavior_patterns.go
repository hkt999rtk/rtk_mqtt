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

// BehaviorPatternManager 行為模式管理器
type BehaviorPatternManager struct {
	patterns       map[string]*BehaviorPattern
	activePatterns map[string]*ActivePattern
	devices        map[string]base.Device
	userProfiles   map[string]*UserProfile
	currentPattern string
	running        bool
	mu             sync.RWMutex
	logger         *logrus.Entry
	config         *BehaviorConfig
}

// BehaviorPattern 行為模式定義
type BehaviorPattern struct {
	ID          string
	Name        string
	Type        string // workday, weekend, vacation, party, guest, away
	Description string
	Activities  []Activity
	DeviceUsage map[string]DeviceUsagePattern
	NetworkLoad NetworkLoadPattern
	Duration    time.Duration
	Probability float64 // 發生機率
	Conditions  []BehaviorCondition
	Priority    int
	Enabled     bool
}

// ActivePattern 活動中的行為模式
type ActivePattern struct {
	PatternID          string
	InstanceID         string
	StartTime          time.Time
	EndTime            time.Time
	State              string // active, paused, completed
	CurrentActivity    string
	ExecutedActivities []string
	DeviceStates       map[string]string
	LastUpdate         time.Time
}

// Activity 活動
type Activity struct {
	ID           string
	Name         string
	Type         string // work, entertainment, cooking, cleaning, sleep, exercise
	Duration     time.Duration
	StartTime    string // HH:MM format, optional
	Devices      []DeviceInteraction
	NetworkUsage float64 // Mbps
	PowerUsage   float64 // Watts
	Probability  float64
}

// DeviceInteraction 設備互動
type DeviceInteraction struct {
	DeviceType string
	DeviceID   string
	Action     string
	Parameters map[string]interface{}
	Duration   time.Duration
	Frequency  string // once, periodic, continuous
}

// DeviceUsagePattern 設備使用模式
type DeviceUsagePattern struct {
	DeviceType           string
	UsageRate            float64 // 0-1
	PeakHours            []int   // 0-23
	AverageDuration      time.Duration
	InteractionFrequency int // per hour
}

// NetworkLoadPattern 網路負載模式
type NetworkLoadPattern struct {
	AverageBandwidth float64 // Mbps
	PeakBandwidth    float64
	PeakHours        []int
	TrafficTypes     map[string]float64 // streaming, gaming, browsing, etc.
}

// UserProfile 使用者檔案
type UserProfile struct {
	ID               string
	Name             string
	Type             string // adult, teenager, child, elderly
	WorkSchedule     string // regular, shift, remote, none
	Preferences      map[string]interface{}
	DeviceOwnership  []string
	ActivityPatterns map[string]float64 // activity type -> probability
	HomeHours        []int              // hours typically at home
}

// BehaviorCondition 行為條件
type BehaviorCondition struct {
	Type     string // day_of_week, time_of_day, weather, presence
	Operator string
	Value    interface{}
}

// BehaviorConfig 行為配置
type BehaviorConfig struct {
	EnableLearning      bool
	SimulationSpeed     float64
	RandomnessLevel     float64
	MultiUserSimulation bool
	AdaptivePatterns    bool
}

// PredefinedPatterns 預定義行為模式
var PredefinedPatterns = []BehaviorPattern{
	{
		ID:          "workday_pattern",
		Name:        "Workday Pattern",
		Type:        "workday",
		Description: "Typical workday behavior with remote work",
		Activities: []Activity{
			{
				ID:        "morning_prep",
				Name:      "Morning Preparation",
				Type:      "work",
				Duration:  2 * time.Hour,
				StartTime: "07:00",
				Devices: []DeviceInteraction{
					{
						DeviceType: "smart_bulb",
						Action:     "gradual_on",
						Parameters: map[string]interface{}{"brightness": 80},
						Duration:   30 * time.Minute,
						Frequency:  "once",
					},
					{
						DeviceType: "air_conditioner",
						Action:     "set_temperature",
						Parameters: map[string]interface{}{"temperature": 24},
						Duration:   2 * time.Hour,
						Frequency:  "once",
					},
					{
						DeviceType: "smart_plug",
						DeviceID:   "coffee_maker",
						Action:     "turn_on",
						Duration:   20 * time.Minute,
						Frequency:  "once",
					},
				},
				NetworkUsage: 5.0,
				PowerUsage:   1500,
				Probability:  0.95,
			},
			{
				ID:        "remote_work",
				Name:      "Remote Work Session",
				Type:      "work",
				Duration:  8 * time.Hour,
				StartTime: "09:00",
				Devices: []DeviceInteraction{
					{
						DeviceType: "laptop",
						Action:     "intensive_use",
						Parameters: map[string]interface{}{"cpu_usage": 60},
						Duration:   8 * time.Hour,
						Frequency:  "continuous",
					},
					{
						DeviceType: "smart_bulb",
						DeviceID:   "office",
						Action:     "maintain_brightness",
						Parameters: map[string]interface{}{"brightness": 90},
						Duration:   8 * time.Hour,
						Frequency:  "continuous",
					},
				},
				NetworkUsage: 25.0,
				PowerUsage:   500,
				Probability:  0.8,
			},
			{
				ID:        "lunch_break",
				Name:      "Lunch Break",
				Type:      "cooking",
				Duration:  1 * time.Hour,
				StartTime: "12:00",
				Devices: []DeviceInteraction{
					{
						DeviceType: "smart_plug",
						DeviceID:   "microwave",
						Action:     "turn_on",
						Duration:   5 * time.Minute,
						Frequency:  "once",
					},
					{
						DeviceType: "smart_tv",
						Action:     "turn_on",
						Parameters: map[string]interface{}{"input": "streaming"},
						Duration:   45 * time.Minute,
						Frequency:  "once",
					},
				},
				NetworkUsage: 15.0,
				PowerUsage:   2000,
				Probability:  0.7,
			},
			{
				ID:        "evening_entertainment",
				Name:      "Evening Entertainment",
				Type:      "entertainment",
				Duration:  3 * time.Hour,
				StartTime: "19:00",
				Devices: []DeviceInteraction{
					{
						DeviceType: "smart_tv",
						Action:     "streaming",
						Parameters: map[string]interface{}{"quality": "4k"},
						Duration:   3 * time.Hour,
						Frequency:  "continuous",
					},
					{
						DeviceType: "smart_bulb",
						Action:     "dim_lights",
						Parameters: map[string]interface{}{"brightness": 40, "color": "warm"},
						Duration:   3 * time.Hour,
						Frequency:  "once",
					},
				},
				NetworkUsage: 30.0,
				PowerUsage:   300,
				Probability:  0.85,
			},
		},
		DeviceUsage: map[string]DeviceUsagePattern{
			"laptop": {
				DeviceType:           "laptop",
				UsageRate:            0.7,
				PeakHours:            []int{9, 10, 11, 14, 15, 16},
				AverageDuration:      4 * time.Hour,
				InteractionFrequency: 20,
			},
			"smart_tv": {
				DeviceType:           "smart_tv",
				UsageRate:            0.4,
				PeakHours:            []int{19, 20, 21, 22},
				AverageDuration:      2 * time.Hour,
				InteractionFrequency: 2,
			},
		},
		NetworkLoad: NetworkLoadPattern{
			AverageBandwidth: 20,
			PeakBandwidth:    50,
			PeakHours:        []int{9, 14, 20, 21},
			TrafficTypes: map[string]float64{
				"streaming": 0.4,
				"browsing":  0.3,
				"work":      0.2,
				"gaming":    0.1,
			},
		},
		Duration:    24 * time.Hour,
		Probability: 0.7,
		Priority:    8,
		Enabled:     true,
	},
	{
		ID:          "weekend_pattern",
		Name:        "Weekend Pattern",
		Type:        "weekend",
		Description: "Relaxed weekend behavior",
		Activities: []Activity{
			{
				ID:        "late_morning",
				Name:      "Late Morning",
				Type:      "sleep",
				Duration:  3 * time.Hour,
				StartTime: "07:00",
				Devices: []DeviceInteraction{
					{
						DeviceType: "smart_bulb",
						Action:     "keep_off",
						Duration:   2 * time.Hour,
						Frequency:  "once",
					},
					{
						DeviceType: "air_conditioner",
						Action:     "maintain_temperature",
						Parameters: map[string]interface{}{"temperature": 25},
						Duration:   3 * time.Hour,
						Frequency:  "continuous",
					},
				},
				NetworkUsage: 1.0,
				PowerUsage:   200,
				Probability:  0.8,
			},
			{
				ID:        "brunch",
				Name:      "Brunch Preparation",
				Type:      "cooking",
				Duration:  90 * time.Minute,
				StartTime: "10:00",
				Devices: []DeviceInteraction{
					{
						DeviceType: "smart_plug",
						DeviceID:   "kitchen_appliances",
						Action:     "sequential_use",
						Duration:   1 * time.Hour,
						Frequency:  "periodic",
					},
					{
						DeviceType: "smart_tv",
						Action:     "background_music",
						Parameters: map[string]interface{}{"volume": 30},
						Duration:   90 * time.Minute,
						Frequency:  "once",
					},
				},
				NetworkUsage: 5.0,
				PowerUsage:   3000,
				Probability:  0.7,
			},
			{
				ID:        "gaming_session",
				Name:      "Gaming Session",
				Type:      "entertainment",
				Duration:  4 * time.Hour,
				StartTime: "14:00",
				Devices: []DeviceInteraction{
					{
						DeviceType: "gaming_console",
						Action:     "intensive_gaming",
						Parameters: map[string]interface{}{"performance": "high"},
						Duration:   4 * time.Hour,
						Frequency:  "continuous",
					},
					{
						DeviceType: "smart_tv",
						Action:     "gaming_display",
						Parameters: map[string]interface{}{"mode": "game", "refresh": 120},
						Duration:   4 * time.Hour,
						Frequency:  "continuous",
					},
				},
				NetworkUsage: 40.0,
				PowerUsage:   600,
				Probability:  0.6,
			},
			{
				ID:        "movie_night",
				Name:      "Movie Night",
				Type:      "entertainment",
				Duration:  3 * time.Hour,
				StartTime: "20:00",
				Devices: []DeviceInteraction{
					{
						DeviceType: "smart_tv",
						Action:     "movie_streaming",
						Parameters: map[string]interface{}{"quality": "4k_hdr"},
						Duration:   150 * time.Minute,
						Frequency:  "once",
					},
					{
						DeviceType: "smart_bulb",
						Action:     "theater_mode",
						Parameters: map[string]interface{}{"brightness": 10, "color": "blue"},
						Duration:   3 * time.Hour,
						Frequency:  "once",
					},
				},
				NetworkUsage: 35.0,
				PowerUsage:   400,
				Probability:  0.75,
			},
		},
		NetworkLoad: NetworkLoadPattern{
			AverageBandwidth: 25,
			PeakBandwidth:    60,
			PeakHours:        []int{14, 15, 16, 20, 21, 22},
			TrafficTypes: map[string]float64{
				"streaming": 0.5,
				"gaming":    0.3,
				"browsing":  0.15,
				"social":    0.05,
			},
		},
		Duration:    24 * time.Hour,
		Probability: 0.9,
		Priority:    7,
		Enabled:     true,
	},
	{
		ID:          "vacation_pattern",
		Name:        "Vacation Pattern",
		Type:        "vacation",
		Description: "Away mode with minimal activity",
		Activities: []Activity{
			{
				ID:       "security_check",
				Name:     "Security Check",
				Type:     "security",
				Duration: 24 * time.Hour,
				Devices: []DeviceInteraction{
					{
						DeviceType: "security_camera",
						Action:     "continuous_monitoring",
						Parameters: map[string]interface{}{"sensitivity": "high"},
						Duration:   24 * time.Hour,
						Frequency:  "continuous",
					},
				},
				NetworkUsage: 2.0,
				PowerUsage:   50,
				Probability:  1.0,
			},
			{
				ID:        "presence_simulation",
				Name:      "Presence Simulation",
				Type:      "security",
				Duration:  4 * time.Hour,
				StartTime: "19:00",
				Devices: []DeviceInteraction{
					{
						DeviceType: "smart_bulb",
						Action:     "random_pattern",
						Parameters: map[string]interface{}{"rooms": []string{"living", "bedroom"}},
						Duration:   4 * time.Hour,
						Frequency:  "periodic",
					},
				},
				NetworkUsage: 0.1,
				PowerUsage:   100,
				Probability:  0.9,
			},
		},
		NetworkLoad: NetworkLoadPattern{
			AverageBandwidth: 2,
			PeakBandwidth:    5,
			PeakHours:        []int{},
			TrafficTypes: map[string]float64{
				"security": 0.9,
				"iot":      0.1,
			},
		},
		Duration:    24 * time.Hour,
		Probability: 1.0,
		Priority:    10,
		Enabled:     true,
	},
	{
		ID:          "party_pattern",
		Name:        "Party Pattern",
		Type:        "party",
		Description: "Party mode with high activity",
		Activities: []Activity{
			{
				ID:        "party_setup",
				Name:      "Party Setup",
				Type:      "entertainment",
				Duration:  1 * time.Hour,
				StartTime: "18:00",
				Devices: []DeviceInteraction{
					{
						DeviceType: "smart_bulb",
						Action:     "party_lights",
						Parameters: map[string]interface{}{"mode": "color_cycle", "speed": "fast"},
						Duration:   6 * time.Hour,
						Frequency:  "continuous",
					},
					{
						DeviceType: "smart_speaker",
						Action:     "party_music",
						Parameters: map[string]interface{}{"volume": 70, "playlist": "party"},
						Duration:   6 * time.Hour,
						Frequency:  "continuous",
					},
				},
				NetworkUsage: 20.0,
				PowerUsage:   1000,
				Probability:  1.0,
			},
			{
				ID:        "guest_devices",
				Name:      "Guest Device Connections",
				Type:      "entertainment",
				Duration:  5 * time.Hour,
				StartTime: "19:00",
				Devices: []DeviceInteraction{
					{
						DeviceType: "router",
						Action:     "guest_network",
						Parameters: map[string]interface{}{"max_devices": 20},
						Duration:   5 * time.Hour,
						Frequency:  "continuous",
					},
				},
				NetworkUsage: 100.0,
				PowerUsage:   200,
				Probability:  0.95,
			},
		},
		NetworkLoad: NetworkLoadPattern{
			AverageBandwidth: 80,
			PeakBandwidth:    150,
			PeakHours:        []int{19, 20, 21, 22, 23},
			TrafficTypes: map[string]float64{
				"streaming": 0.4,
				"social":    0.3,
				"browsing":  0.2,
				"gaming":    0.1,
			},
		},
		Duration:    6 * time.Hour,
		Probability: 0.1,
		Priority:    9,
		Enabled:     true,
	},
}

// NewBehaviorPatternManager 創建新的行為模式管理器
func NewBehaviorPatternManager(config *BehaviorConfig) *BehaviorPatternManager {
	if config == nil {
		config = &BehaviorConfig{
			EnableLearning:      false,
			SimulationSpeed:     1.0,
			RandomnessLevel:     0.2,
			MultiUserSimulation: true,
			AdaptivePatterns:    true,
		}
	}

	bpm := &BehaviorPatternManager{
		patterns:       make(map[string]*BehaviorPattern),
		activePatterns: make(map[string]*ActivePattern),
		devices:        make(map[string]base.Device),
		userProfiles:   make(map[string]*UserProfile),
		currentPattern: "none",
		config:         config,
		logger:         logrus.WithField("component", "behavior_pattern_manager"),
	}

	// 載入預定義模式
	bpm.loadPredefinedPatterns()
	// 初始化使用者檔案
	bpm.initializeUserProfiles()

	return bpm
}

// loadPredefinedPatterns 載入預定義模式
func (bpm *BehaviorPatternManager) loadPredefinedPatterns() {
	for _, pattern := range PredefinedPatterns {
		p := pattern
		bpm.patterns[p.ID] = &p
	}
	bpm.logger.Infof("Loaded %d predefined behavior patterns", len(PredefinedPatterns))
}

// initializeUserProfiles 初始化使用者檔案
func (bpm *BehaviorPatternManager) initializeUserProfiles() {
	// 創建預設使用者檔案
	profiles := []UserProfile{
		{
			ID:           "user_adult_1",
			Name:         "Primary User",
			Type:         "adult",
			WorkSchedule: "remote",
			Preferences: map[string]interface{}{
				"wake_time":     "07:00",
				"sleep_time":    "23:00",
				"entertainment": "streaming",
			},
			DeviceOwnership: []string{"laptop", "smartphone", "smart_tv"},
			ActivityPatterns: map[string]float64{
				"work":          0.4,
				"entertainment": 0.3,
				"cooking":       0.1,
				"exercise":      0.1,
				"sleep":         0.1,
			},
			HomeHours: []int{0, 1, 2, 3, 4, 5, 6, 7, 17, 18, 19, 20, 21, 22, 23},
		},
		{
			ID:           "user_teenager_1",
			Name:         "Teenager",
			Type:         "teenager",
			WorkSchedule: "none",
			Preferences: map[string]interface{}{
				"wake_time":     "09:00",
				"sleep_time":    "01:00",
				"entertainment": "gaming",
			},
			DeviceOwnership: []string{"smartphone", "gaming_console", "laptop"},
			ActivityPatterns: map[string]float64{
				"entertainment": 0.5,
				"social":        0.3,
				"study":         0.1,
				"sleep":         0.1,
			},
			HomeHours: []int{0, 1, 15, 16, 17, 18, 19, 20, 21, 22, 23},
		},
	}

	for _, profile := range profiles {
		p := profile
		bpm.userProfiles[p.ID] = &p
	}
}

// Start 啟動行為模式管理器
func (bpm *BehaviorPatternManager) Start(ctx context.Context) error {
	bpm.mu.Lock()
	defer bpm.mu.Unlock()

	if bpm.running {
		return fmt.Errorf("behavior pattern manager is already running")
	}

	bpm.running = true
	bpm.logger.Info("Starting behavior pattern manager")

	// 啟動管理循環
	go bpm.patternSelectionLoop(ctx)
	go bpm.activityExecutionLoop(ctx)
	go bpm.behaviorSimulationLoop(ctx)

	return nil
}

// Stop 停止行為模式管理器
func (bpm *BehaviorPatternManager) Stop() error {
	bpm.mu.Lock()
	defer bpm.mu.Unlock()

	if !bpm.running {
		return fmt.Errorf("behavior pattern manager is not running")
	}

	bpm.running = false
	bpm.logger.Info("Stopping behavior pattern manager")
	return nil
}

// RegisterDevice 註冊設備
func (bpm *BehaviorPatternManager) RegisterDevice(deviceID string, device base.Device) {
	bpm.mu.Lock()
	defer bpm.mu.Unlock()

	bpm.devices[deviceID] = device
	bpm.logger.WithField("device_id", deviceID).Debug("Device registered for behavior patterns")
}

// patternSelectionLoop 模式選擇循環
func (bpm *BehaviorPatternManager) patternSelectionLoop(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			bpm.selectPattern()
		}
	}
}

// selectPattern 選擇模式
func (bpm *BehaviorPatternManager) selectPattern() {
	bpm.mu.Lock()
	defer bpm.mu.Unlock()

	now := time.Now()
	weekday := now.Weekday()

	// 根據時間和條件選擇合適的模式
	var selectedPattern *BehaviorPattern
	maxPriority := -1

	for _, pattern := range bpm.patterns {
		if !pattern.Enabled {
			continue
		}

		// 檢查條件
		if !bpm.checkPatternConditions(pattern) {
			continue
		}

		// 根據類型和時間判斷
		switch pattern.Type {
		case "workday":
			if weekday >= time.Monday && weekday <= time.Friday {
				if pattern.Priority > maxPriority {
					selectedPattern = pattern
					maxPriority = pattern.Priority
				}
			}
		case "weekend":
			if weekday == time.Saturday || weekday == time.Sunday {
				if pattern.Priority > maxPriority {
					selectedPattern = pattern
					maxPriority = pattern.Priority
				}
			}
		case "vacation":
			// 假期模式需要特殊觸發
			continue
		case "party":
			// 派對模式需要特殊觸發
			continue
		}
	}

	// 應用選中的模式
	if selectedPattern != nil && !bpm.isPatternActive(selectedPattern.ID) {
		bpm.startPattern(selectedPattern)
	}
}

// checkPatternConditions 檢查模式條件
func (bpm *BehaviorPatternManager) checkPatternConditions(pattern *BehaviorPattern) bool {
	for _, condition := range pattern.Conditions {
		if !bpm.evaluateCondition(condition) {
			return false
		}
	}
	return true
}

// evaluateCondition 評估條件
func (bpm *BehaviorPatternManager) evaluateCondition(condition BehaviorCondition) bool {
	// 簡化的條件評估
	switch condition.Type {
	case "day_of_week":
		weekday := time.Now().Weekday().String()
		return weekday == condition.Value.(string)
	case "time_of_day":
		hour := time.Now().Hour()
		targetHour := condition.Value.(int)
		switch condition.Operator {
		case "gt":
			return hour > targetHour
		case "lt":
			return hour < targetHour
		case "eq":
			return hour == targetHour
		}
	}
	return true
}

// isPatternActive 檢查模式是否活動
func (bpm *BehaviorPatternManager) isPatternActive(patternID string) bool {
	for _, active := range bpm.activePatterns {
		if active.PatternID == patternID && active.State == "active" {
			return true
		}
	}
	return false
}

// startPattern 開始模式
func (bpm *BehaviorPatternManager) startPattern(pattern *BehaviorPattern) {
	instanceID := fmt.Sprintf("%s_%d", pattern.ID, time.Now().UnixNano())

	active := &ActivePattern{
		PatternID:          pattern.ID,
		InstanceID:         instanceID,
		StartTime:          time.Now(),
		State:              "active",
		ExecutedActivities: make([]string, 0),
		DeviceStates:       make(map[string]string),
		LastUpdate:         time.Now(),
	}

	if pattern.Duration > 0 {
		active.EndTime = time.Now().Add(pattern.Duration)
	}

	bpm.activePatterns[instanceID] = active
	bpm.currentPattern = pattern.Type

	bpm.logger.WithFields(logrus.Fields{
		"pattern_id":   pattern.ID,
		"pattern_name": pattern.Name,
		"instance_id":  instanceID,
	}).Info("Behavior pattern started")

	// 執行模式活動
	go bpm.executePatternActivities(pattern, active)
}

// executePatternActivities 執行模式活動
func (bpm *BehaviorPatternManager) executePatternActivities(pattern *BehaviorPattern, active *ActivePattern) {
	for _, activity := range pattern.Activities {
		// 檢查機率
		if rand.Float64() > activity.Probability {
			continue
		}

		// 檢查是否仍然活動
		bpm.mu.RLock()
		if active.State != "active" {
			bpm.mu.RUnlock()
			break
		}
		bpm.mu.RUnlock()

		// 執行活動
		bpm.executeActivity(activity, active)

		// 記錄執行的活動
		bpm.mu.Lock()
		active.ExecutedActivities = append(active.ExecutedActivities, activity.ID)
		active.CurrentActivity = activity.Name
		active.LastUpdate = time.Now()
		bpm.mu.Unlock()

		// 活動持續時間
		if activity.Duration > 0 {
			time.Sleep(time.Duration(float64(activity.Duration) / bpm.config.SimulationSpeed))
		}
	}

	// 完成模式
	bpm.mu.Lock()
	active.State = "completed"
	active.EndTime = time.Now()
	bpm.mu.Unlock()

	bpm.logger.WithField("pattern_id", pattern.ID).Info("Behavior pattern completed")
}

// executeActivity 執行活動
func (bpm *BehaviorPatternManager) executeActivity(activity Activity, active *ActivePattern) {
	bpm.logger.WithFields(logrus.Fields{
		"activity_id":   activity.ID,
		"activity_name": activity.Name,
		"activity_type": activity.Type,
	}).Debug("Executing activity")

	// 執行設備互動
	for _, interaction := range activity.Devices {
		bpm.executeDeviceInteraction(interaction, active)
	}

	// 模擬網路使用
	if activity.NetworkUsage > 0 {
		bpm.simulateNetworkUsage(activity.NetworkUsage)
	}

	// 模擬電力使用
	if activity.PowerUsage > 0 {
		bpm.simulatePowerUsage(activity.PowerUsage)
	}
}

// executeDeviceInteraction 執行設備互動
func (bpm *BehaviorPatternManager) executeDeviceInteraction(interaction DeviceInteraction, active *ActivePattern) {
	bpm.mu.RLock()
	defer bpm.mu.RUnlock()

	// 查找目標設備
	targetDevices := bpm.findTargetDevices(interaction)

	for _, device := range targetDevices {
		cmd := base.Command{
			ID:         fmt.Sprintf("behavior_%s_%d", interaction.Action, time.Now().UnixNano()),
			Type:       interaction.Action,
			Parameters: interaction.Parameters,
			Timeout:    10 * time.Second,
		}

		err := device.HandleCommand(cmd)
		if err != nil {
			bpm.logger.WithError(err).WithField("device_id", device.GetDeviceID()).
				Error("Failed to execute device interaction")
		} else {
			// 更新設備狀態
			active.DeviceStates[device.GetDeviceID()] = interaction.Action
		}
	}
}

// findTargetDevices 查找目標設備
func (bpm *BehaviorPatternManager) findTargetDevices(interaction DeviceInteraction) []base.Device {
	devices := make([]base.Device, 0)

	if interaction.DeviceID != "" {
		// 特定設備
		if device, exists := bpm.devices[interaction.DeviceID]; exists {
			devices = append(devices, device)
		}
	} else if interaction.DeviceType != "" {
		// 設備類型
		for _, device := range bpm.devices {
			if device.GetDeviceType() == interaction.DeviceType {
				devices = append(devices, device)
			}
		}
	}

	return devices
}

// simulateNetworkUsage 模擬網路使用
func (bpm *BehaviorPatternManager) simulateNetworkUsage(bandwidth float64) {
	// 這裡可以與網路流量模擬器整合
	bpm.logger.WithField("bandwidth", bandwidth).Debug("Simulating network usage")
}

// simulatePowerUsage 模擬電力使用
func (bpm *BehaviorPatternManager) simulatePowerUsage(power float64) {
	// 這裡可以與電力模擬系統整合
	bpm.logger.WithField("power", power).Debug("Simulating power usage")
}

// activityExecutionLoop 活動執行循環
func (bpm *BehaviorPatternManager) activityExecutionLoop(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			bpm.checkActivePatterns()
		}
	}
}

// checkActivePatterns 檢查活動模式
func (bpm *BehaviorPatternManager) checkActivePatterns() {
	bpm.mu.Lock()
	defer bpm.mu.Unlock()

	now := time.Now()
	completed := make([]string, 0)

	for instanceID, active := range bpm.activePatterns {
		// 檢查是否過期
		if !active.EndTime.IsZero() && now.After(active.EndTime) {
			active.State = "completed"
		}

		// 清理完成的模式
		if active.State == "completed" && time.Since(active.EndTime) > 10*time.Minute {
			completed = append(completed, instanceID)
		}
	}

	// 刪除完成的模式
	for _, instanceID := range completed {
		delete(bpm.activePatterns, instanceID)
	}
}

// behaviorSimulationLoop 行為模擬循環
func (bpm *BehaviorPatternManager) behaviorSimulationLoop(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			bpm.simulateUserBehavior()
		}
	}
}

// simulateUserBehavior 模擬使用者行為
func (bpm *BehaviorPatternManager) simulateUserBehavior() {
	if !bpm.config.MultiUserSimulation {
		return
	}

	bpm.mu.RLock()
	defer bpm.mu.RUnlock()

	// 為每個使用者模擬行為
	for _, profile := range bpm.userProfiles {
		// 根據使用者檔案生成隨機行為
		if rand.Float64() < bpm.config.RandomnessLevel {
			bpm.generateRandomUserAction(profile)
		}
	}
}

// generateRandomUserAction 生成隨機使用者動作
func (bpm *BehaviorPatternManager) generateRandomUserAction(profile *UserProfile) {
	// 根據使用者偏好生成動作
	for activityType, probability := range profile.ActivityPatterns {
		if rand.Float64() < probability*0.1 { // 10% 的基礎機率
			bpm.logger.WithFields(logrus.Fields{
				"user_id":       profile.ID,
				"activity_type": activityType,
			}).Debug("Random user action generated")
		}
	}
}

// TriggerPattern 觸發特定模式
func (bpm *BehaviorPatternManager) TriggerPattern(patternID string) error {
	bpm.mu.RLock()
	pattern, exists := bpm.patterns[patternID]
	bpm.mu.RUnlock()

	if !exists {
		return fmt.Errorf("pattern %s not found", patternID)
	}

	bpm.mu.Lock()
	defer bpm.mu.Unlock()

	if bpm.isPatternActive(patternID) {
		return fmt.Errorf("pattern %s is already active", patternID)
	}

	bpm.startPattern(pattern)
	return nil
}

// GetCurrentPattern 獲取當前模式
func (bpm *BehaviorPatternManager) GetCurrentPattern() string {
	bpm.mu.RLock()
	defer bpm.mu.RUnlock()
	return bpm.currentPattern
}

// GetActivePatterns 獲取活動模式
func (bpm *BehaviorPatternManager) GetActivePatterns() []*ActivePattern {
	bpm.mu.RLock()
	defer bpm.mu.RUnlock()

	patterns := make([]*ActivePattern, 0, len(bpm.activePatterns))
	for _, pattern := range bpm.activePatterns {
		patterns = append(patterns, pattern)
	}
	return patterns
}

// GetStatistics 獲取統計資訊
func (bpm *BehaviorPatternManager) GetStatistics() map[string]interface{} {
	bpm.mu.RLock()
	defer bpm.mu.RUnlock()

	stats := map[string]interface{}{
		"total_patterns":  len(bpm.patterns),
		"active_patterns": len(bpm.activePatterns),
		"current_pattern": bpm.currentPattern,
		"user_profiles":   len(bpm.userProfiles),
		"enabled_count":   0,
	}

	// 統計啟用的模式
	for _, pattern := range bpm.patterns {
		if pattern.Enabled {
			stats["enabled_count"] = stats["enabled_count"].(int) + 1
		}
	}

	// 統計活動類型分佈
	activityTypes := make(map[string]int)
	for _, active := range bpm.activePatterns {
		if active.CurrentActivity != "" {
			activityTypes[active.CurrentActivity]++
		}
	}
	stats["activity_distribution"] = activityTypes

	return stats
}
