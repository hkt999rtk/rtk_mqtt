package iot

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"time"

	"rtk_simulation/pkg/devices/base"
	// "github.com/sirupsen/logrus"
)

// AirConditioner 空調設備模擬器
type AirConditioner struct {
	*base.BaseDevice

	// 空調狀態
	power      bool    // 電源狀態
	mode       string  // 模式: cool, heat, dry, fan, auto
	targetTemp float64 // 目標溫度 (°C)
	fanSpeed   int     // 風速 1-5
	swingH     bool    // 水平擺風
	swingV     bool    // 垂直擺風
	turboMode  bool    // 強力模式
	sleepMode  bool    // 睡眠模式
	ecoMode    bool    // 節能模式

	// 環境感測
	roomTemp     float64 // 室內溫度
	roomHumidity float64 // 室內濕度 (%)
	outdoorTemp  float64 // 室外溫度

	// 效能資訊
	coolingCapacity float64 // 製冷量 (BTU/hr)
	heatingCapacity float64 // 制熱量 (BTU/hr)
	energyRating    string  // 能效等級
	cop             float64 // 性能係數
	seer            float64 // 季節性能效比

	// 功率和能耗
	ratedPower    float64 // 額定功率 (W)
	currentPower  float64 // 當前功率 (W)
	dailyEnergy   float64 // 日耗電量 (kWh)
	monthlyEnergy float64 // 月耗電量 (kWh)

	// 運行統計
	compressorOn  bool          // 壓縮機狀態
	totalRunTime  time.Duration // 累計運行時間
	cycleCount    int           // 開關循環次數
	lastCycleTime time.Time     // 上次循環時間

	// 維護資訊
	filterStatus     string    // 濾網狀態: clean, dirty, replace
	lastMaintenance  time.Time // 上次維護時間
	maintenanceHours int       // 維護週期 (小時)

	// 排程和自動化
	schedules []ACSchedule // 排程設定
	autoMode  bool         // 自動模式
	timerOn   *time.Time   // 定時開機
	timerOff  *time.Time   // 定時關機

	// MQTT 客戶端
	mqttClient *base.MQTTClient

	// 日誌
}

// ACSchedule 空調排程
type ACSchedule struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	StartTime  string   `json:"start_time"` // HH:MM
	EndTime    string   `json:"end_time"`   // HH:MM
	Days       []string `json:"days"`
	Mode       string   `json:"mode"`
	TargetTemp float64  `json:"target_temp"`
	FanSpeed   int      `json:"fan_speed"`
	Enabled    bool     `json:"enabled"`
}

// ThermalModel 熱力學模型
type ThermalModel struct {
	RoomVolume     float64 // 房間體積 (m³)
	Insulation     float64 // 隔熱係數
	HeatGeneration float64 // 內部熱源 (W)
	AirExchange    float64 // 換氣率 (次/小時)
}

// NewAirConditioner 建立新的空調模擬器
func NewAirConditioner(config base.DeviceConfig, mqttConfig base.MQTTConfig) (*AirConditioner, error) {
	baseDevice := base.NewBaseDevice(config)

	// 建立 MQTT 客戶端
	mqttClient := base.NewMQTTClient(mqttConfig, baseDevice)

	ac := &AirConditioner{
		BaseDevice:       baseDevice,
		power:            false,
		mode:             "cool",
		targetTemp:       24.0,
		fanSpeed:         3,
		roomTemp:         26.0,
		roomHumidity:     60.0,
		outdoorTemp:      30.0,
		coolingCapacity:  12000, // 12,000 BTU/hr (約 3.5kW)
		heatingCapacity:  10000, // 10,000 BTU/hr
		energyRating:     "A++",
		cop:              3.5,
		seer:             16.0,
		ratedPower:       1200.0, // 1.2kW
		filterStatus:     "clean",
		maintenanceHours: 2000, // 2000 小時維護週期
		autoMode:         true,
		mqttClient:       mqttClient,
		// logger removed
	}

	// 初始化排程
	ac.initializeSchedules()

	return ac, nil
}

// Start 啟動空調模擬器
func (ac *AirConditioner) Start(ctx context.Context) error {
	if err := ac.BaseDevice.Start(ctx); err != nil {
		return err
	}

	// 連接 MQTT
	if err := ac.mqttClient.Connect(); err != nil {
		// logger removed.Warn("Failed to connect MQTT, continuing without MQTT")
	}

	// logger removed

	// 啟動空調特定的處理循環
	go ac.thermalControlLoop(ctx)
	go ac.environmentSensingLoop(ctx)
	go ac.scheduleCheckLoop(ctx)
	go ac.maintenanceCheckLoop(ctx)
	go ac.publishingLoop(ctx)

	return nil
}

// Stop 停止空調模擬器
func (ac *AirConditioner) Stop() error {
	// logger removed

	// 記錄運行時間
	if ac.power {
		ac.totalRunTime += time.Since(ac.lastCycleTime)
	}

	// 斷開 MQTT
	if ac.mqttClient != nil {
		ac.mqttClient.Disconnect()
	}

	return ac.BaseDevice.Stop()
}

// GenerateStatePayload 生成空調狀態數據
func (ac *AirConditioner) GenerateStatePayload() base.StatePayload {
	basePayload := ac.BaseDevice.GenerateStatePayload()

	// 添加空調特定狀態
	basePayload.Extra = map[string]interface{}{
		"ac_status": map[string]interface{}{
			"power":            ac.power,
			"mode":             ac.mode,
			"target_temp":      ac.targetTemp,
			"fan_speed":        ac.fanSpeed,
			"swing_horizontal": ac.swingH,
			"swing_vertical":   ac.swingV,
			"turbo_mode":       ac.turboMode,
			"sleep_mode":       ac.sleepMode,
			"eco_mode":         ac.ecoMode,
			"compressor_on":    ac.compressorOn,
		},
		"environment": map[string]interface{}{
			"room_temp":       ac.roomTemp,
			"room_humidity":   ac.roomHumidity,
			"outdoor_temp":    ac.outdoorTemp,
			"temp_difference": math.Abs(ac.roomTemp - ac.targetTemp),
		},
		"performance": map[string]interface{}{
			"current_power_w":    ac.currentPower,
			"rated_power_w":      ac.ratedPower,
			"cooling_capacity":   ac.coolingCapacity,
			"energy_rating":      ac.energyRating,
			"cop":                ac.cop,
			"efficiency_percent": ac.calculateEfficiency(),
		},
		"maintenance": map[string]interface{}{
			"filter_status":           ac.filterStatus,
			"last_maintenance":        ac.lastMaintenance,
			"hours_since_maintenance": ac.getHoursSinceMaintenance(),
			"next_maintenance_hours":  ac.getHoursToMaintenance(),
		},
	}

	return basePayload
}

// GenerateTelemetryData 生成空調遙測數據
func (ac *AirConditioner) GenerateTelemetryData() map[string]base.TelemetryPayload {
	telemetry := ac.BaseDevice.GenerateTelemetryData()

	// 溫控資訊
	telemetry["climate_control"] = base.TelemetryPayload{
		"room_temp":         ac.roomTemp,
		"target_temp":       ac.targetTemp,
		"outdoor_temp":      ac.outdoorTemp,
		"room_humidity":     ac.roomHumidity,
		"mode":              ac.mode,
		"fan_speed":         ac.fanSpeed,
		"compressor_status": ac.compressorOn,
		"temp_delta":        ac.roomTemp - ac.targetTemp,
	}

	// 能耗統計
	telemetry["energy_consumption"] = base.TelemetryPayload{
		"current_power_w":    ac.currentPower,
		"daily_energy_kwh":   ac.dailyEnergy,
		"monthly_energy_kwh": ac.monthlyEnergy,
		"efficiency_cop":     ac.cop,
		"energy_rating":      ac.energyRating,
		"power_factor":       ac.calculatePowerFactor(),
	}

	// 運行統計
	telemetry["operation_stats"] = base.TelemetryPayload{
		"total_runtime_hours": ac.totalRunTime.Hours(),
		"cycle_count":         ac.cycleCount,
		"compressor_cycles":   ac.getCompressorCycles(),
		"average_runtime":     ac.getAverageRuntime(),
		"duty_cycle_percent":  ac.calculateDutyCycle(),
	}

	// 維護狀態
	telemetry["maintenance_status"] = base.TelemetryPayload{
		"filter_condition":        ac.getFilterCondition(),
		"hours_since_maintenance": ac.getHoursSinceMaintenance(),
		"maintenance_due_percent": ac.getMaintenanceDuePercent(),
		"estimated_filter_life":   ac.getFilterLifeRemaining(),
	}

	return telemetry
}

// GenerateEvents 生成空調事件
func (ac *AirConditioner) GenerateEvents() []base.Event {
	var events []base.Event

	// 目標溫度達成事件
	tempDiff := math.Abs(ac.roomTemp - ac.targetTemp)
	if tempDiff <= 0.5 && ac.power && rand.Float64() < 0.1 { // 溫差小於0.5度
		events = append(events, base.Event{
			EventType: "ac.target_temp_reached",
			Severity:  "info",
			Message:   "Target temperature reached",
			Extra: map[string]interface{}{
				"target_temp":   ac.targetTemp,
				"current_temp":  ac.roomTemp,
				"mode":          ac.mode,
				"time_to_reach": ac.calculateTimeToReach(),
			},
		})
	}

	// 濾網維護提醒
	if ac.filterStatus == "dirty" && rand.Float64() < 0.05 { // 5% 機率
		events = append(events, base.Event{
			EventType: "ac.filter_maintenance_required",
			Severity:  "warning",
			Message:   "Air filter requires cleaning or replacement",
			Extra: map[string]interface{}{
				"filter_status":           ac.filterStatus,
				"hours_since_maintenance": ac.getHoursSinceMaintenance(),
				"maintenance_due_percent": ac.getMaintenanceDuePercent(),
				"performance_impact":      ac.getFilterPerformanceImpact(),
			},
		})
	}

	// 高能耗警告
	if ac.currentPower > ac.ratedPower*1.1 { // 超過額定功率 10%
		events = append(events, base.Event{
			EventType: "ac.high_power_consumption",
			Severity:  "warning",
			Message:   "Power consumption exceeds normal range",
			Extra: map[string]interface{}{
				"current_power":   ac.currentPower,
				"rated_power":     ac.ratedPower,
				"excess_percent":  ((ac.currentPower / ac.ratedPower) - 1.0) * 100,
				"possible_causes": ac.getDiagnosticReasons(),
			},
		})
	}

	// 室外溫度極端警告
	if ac.outdoorTemp > 40.0 || ac.outdoorTemp < -10.0 {
		events = append(events, base.Event{
			EventType: "ac.extreme_outdoor_temp",
			Severity:  "warning",
			Message:   fmt.Sprintf("Extreme outdoor temperature: %.1f°C", ac.outdoorTemp),
			Extra: map[string]interface{}{
				"outdoor_temp":       ac.outdoorTemp,
				"efficiency_impact":  ac.calculateTemperatureImpact(),
				"recommended_action": ac.getRecommendedAction(),
			},
		})
	}

	// 壓縮機頻繁啟動警告
	if ac.cycleCount > 50 && time.Since(ac.lastCycleTime) < time.Hour { // 一小時內超過50次
		events = append(events, base.Event{
			EventType: "ac.frequent_cycling",
			Severity:  "warning",
			Message:   "Compressor cycling too frequently",
			Extra: map[string]interface{}{
				"cycle_count":     ac.cycleCount,
				"time_window":     "1 hour",
				"possible_causes": []string{"oversized_unit", "poor_insulation", "thermostat_issue"},
				"efficiency_loss": "15-25%",
			},
		})
	}

	return events
}

// HandleCommand 處理空調命令
func (ac *AirConditioner) HandleCommand(cmd base.Command) error {
	// logger removed.Info("Handling air conditioner command")

	switch cmd.Type {
	case "ac.set_power":
		return ac.handleSetPower(cmd)
	case "ac.set_mode":
		return ac.handleSetMode(cmd)
	case "ac.set_temperature":
		return ac.handleSetTemperature(cmd)
	case "ac.set_fan_speed":
		return ac.handleSetFanSpeed(cmd)
	case "ac.set_swing":
		return ac.handleSetSwing(cmd)
	case "ac.set_timer":
		return ac.handleSetTimer(cmd)
	case "ac.create_schedule":
		return ac.handleCreateSchedule(cmd)
	case "ac.maintenance_reset":
		return ac.handleMaintenanceReset(cmd)
	case "ac.diagnostic":
		return ac.handleDiagnostic(cmd)
	default:
		return ac.BaseDevice.HandleCommand(cmd)
	}
}

// 內部方法實作
func (ac *AirConditioner) initializeSchedules() {
	ac.schedules = []ACSchedule{
		{
			ID:         "sleep_mode",
			Name:       "Sleep Schedule",
			StartTime:  "22:00",
			EndTime:    "06:00",
			Days:       []string{"monday", "tuesday", "wednesday", "thursday", "friday", "saturday", "sunday"},
			Mode:       "cool",
			TargetTemp: 26.0,
			FanSpeed:   1,
			Enabled:    true,
		},
		{
			ID:         "work_hours",
			Name:       "Daytime Comfort",
			StartTime:  "08:00",
			EndTime:    "18:00",
			Days:       []string{"monday", "tuesday", "wednesday", "thursday", "friday"},
			Mode:       "cool",
			TargetTemp: 24.0,
			FanSpeed:   3,
			Enabled:    true,
		},
	}
}

// 熱力控制循環
func (ac *AirConditioner) thermalControlLoop(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			ac.updateThermalModel()
			ac.controlCompressor()
			ac.updatePowerConsumption()
		}
	}
}

// 環境感測循環
func (ac *AirConditioner) environmentSensingLoop(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			ac.updateEnvironmentSensors()
		}
	}
}

// 排程檢查循環
func (ac *AirConditioner) scheduleCheckLoop(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			ac.checkSchedules()
			ac.checkTimers()
		}
	}
}

// 維護檢查循環
func (ac *AirConditioner) maintenanceCheckLoop(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			ac.updateMaintenanceStatus()
		}
	}
}

// MQTT 發布循環
func (ac *AirConditioner) publishingLoop(ctx context.Context) {
	stateTicker := time.NewTicker(30 * time.Second)
	telemetryTicker := time.NewTicker(20 * time.Second)
	defer stateTicker.Stop()
	defer telemetryTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-stateTicker.C:
			if ac.mqttClient != nil && ac.mqttClient.IsConnected() {
				if err := ac.mqttClient.PublishState(ac.GenerateStatePayload()); err != nil {
					// logger removed.Warn("Failed to publish state")
				}
			}
		case <-telemetryTicker.C:
			if ac.mqttClient != nil && ac.mqttClient.IsConnected() {
				telemetryData := ac.GenerateTelemetryData()
				for metric, payload := range telemetryData {
					if err := ac.mqttClient.PublishTelemetry(metric, payload); err != nil {
						// logger removed.Warnf("Failed to publish telemetry: %s", metric)
					}
				}

				// 發布事件
				events := ac.GenerateEvents()
				for _, event := range events {
					if err := ac.mqttClient.PublishEvent(event.EventType, event); err != nil {
						// logger removed.Warnf("Failed to publish event: %s", event.EventType)
					}
				}
			}
		}
	}
}

// 輔助方法
func (ac *AirConditioner) updateThermalModel() {
	if !ac.power {
		// 空調關閉時，室溫趨向室外溫度
		tempDiff := ac.outdoorTemp - ac.roomTemp
		ac.roomTemp += tempDiff * 0.02 // 緩慢變化
		return
	}

	// 計算冷卻/加熱效果
	tempDiff := ac.roomTemp - ac.targetTemp

	switch ac.mode {
	case "cool":
		if tempDiff > 0 && ac.compressorOn {
			// 製冷中
			coolingRate := ac.calculateCoolingRate()
			ac.roomTemp -= coolingRate
		}
	case "heat":
		if tempDiff < 0 && ac.compressorOn {
			// 制熱中
			heatingRate := ac.calculateHeatingRate()
			ac.roomTemp += heatingRate
		}
	case "dry":
		// 除濕模式，輕微降溫
		if ac.compressorOn {
			ac.roomTemp -= 0.1
			ac.roomHumidity -= 2.0
		}
	}

	// 限制溫度範圍
	if ac.roomTemp < 16.0 {
		ac.roomTemp = 16.0
	} else if ac.roomTemp > 35.0 {
		ac.roomTemp = 35.0
	}
}

func (ac *AirConditioner) calculateCoolingRate() float64 {
	// 基礎冷卻率取決於風速和能力
	baseRate := 0.2 * float64(ac.fanSpeed) / 5.0 // 風速影響

	// 室外溫度影響效率
	tempImpact := 1.0 - (ac.outdoorTemp-25.0)*0.02 // 室外溫度越高效率越低
	if tempImpact < 0.5 {
		tempImpact = 0.5
	}

	// 濾網狀態影響
	filterImpact := 1.0
	if ac.filterStatus == "dirty" {
		filterImpact = 0.8
	} else if ac.filterStatus == "replace" {
		filterImpact = 0.6
	}

	return baseRate * tempImpact * filterImpact
}

func (ac *AirConditioner) calculateHeatingRate() float64 {
	// 與冷卻類似但考慮不同因素
	baseRate := 0.15 * float64(ac.fanSpeed) / 5.0

	// 室外溫度影響 (加熱時室外溫度越低效率越低)
	tempImpact := 1.0 + (ac.outdoorTemp+10.0)*0.01
	if tempImpact > 1.5 {
		tempImpact = 1.5
	}

	return baseRate * tempImpact
}

// 實作其他必要的方法...
func (ac *AirConditioner) controlCompressor()                  {}
func (ac *AirConditioner) updatePowerConsumption()             {}
func (ac *AirConditioner) updateEnvironmentSensors()           {}
func (ac *AirConditioner) checkSchedules()                     {}
func (ac *AirConditioner) checkTimers()                        {}
func (ac *AirConditioner) updateMaintenanceStatus()            {}
func (ac *AirConditioner) calculateEfficiency() float64        { return 85.0 }
func (ac *AirConditioner) getHoursSinceMaintenance() float64   { return 100.0 }
func (ac *AirConditioner) getHoursToMaintenance() float64      { return 1900.0 }
func (ac *AirConditioner) calculatePowerFactor() float64       { return 0.95 }
func (ac *AirConditioner) getCompressorCycles() int            { return 25 }
func (ac *AirConditioner) getAverageRuntime() float64          { return 45.0 }
func (ac *AirConditioner) calculateDutyCycle() float64         { return 60.0 }
func (ac *AirConditioner) getFilterCondition() int             { return 85 }
func (ac *AirConditioner) getMaintenanceDuePercent() float64   { return 5.0 }
func (ac *AirConditioner) getFilterLifeRemaining() int         { return 90 }
func (ac *AirConditioner) calculateTimeToReach() time.Duration { return 15 * time.Minute }
func (ac *AirConditioner) getFilterPerformanceImpact() float64 { return 15.0 }
func (ac *AirConditioner) getDiagnosticReasons() []string      { return []string{"high_outdoor_temp"} }
func (ac *AirConditioner) calculateTemperatureImpact() float64 { return 20.0 }
func (ac *AirConditioner) getRecommendedAction() string        { return "reduce_target_temp" }

// 命令處理方法
func (ac *AirConditioner) handleSetPower(cmd base.Command) error         { return nil }
func (ac *AirConditioner) handleSetMode(cmd base.Command) error          { return nil }
func (ac *AirConditioner) handleSetTemperature(cmd base.Command) error   { return nil }
func (ac *AirConditioner) handleSetFanSpeed(cmd base.Command) error      { return nil }
func (ac *AirConditioner) handleSetSwing(cmd base.Command) error         { return nil }
func (ac *AirConditioner) handleSetTimer(cmd base.Command) error         { return nil }
func (ac *AirConditioner) handleCreateSchedule(cmd base.Command) error   { return nil }
func (ac *AirConditioner) handleMaintenanceReset(cmd base.Command) error { return nil }
func (ac *AirConditioner) handleDiagnostic(cmd base.Command) error       { return nil }
