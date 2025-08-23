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

// SmartBulb 智慧燈泡模擬器
type SmartBulb struct {
	*base.BaseDevice

	// 燈泡狀態
	power      bool   // 電源狀態
	brightness int    // 亮度 (0-100%)
	color      Color  // RGB 顏色
	colorTemp  int    // 色溫 (2700K-6500K)
	mode       string // 模式: normal, party, sleep, reading, custom

	// 能耗資訊
	wattage    float64 // 功率消耗 (watts)
	maxWattage float64 // 最大功率
	efficiency float64 // 光效 (lumens/watt)

	// 環境感測
	ambientLight int       // 環境光感測 (lux)
	motionSensor bool      // 動作感測器
	lastMotion   time.Time // 最後動作檢測時間

	// 排程和自動化
	schedules []LightSchedule // 排程設定
	autoMode  bool            // 自動模式
	fadeTime  time.Duration   // 漸變時間

	// 使用統計
	onTime      time.Duration // 累計開啟時間
	switchCount int           // 開關次數
	lastOn      time.Time     // 最後開啟時間

	// 健康監控
	lifespan    time.Duration // 預期壽命
	usedLife    time.Duration // 已使用時間
	degradation float64       // 光衰減率

	// 日誌
}

// Color RGB 顏色結構
type Color struct {
	Red   int `json:"red"`   // 0-255
	Green int `json:"green"` // 0-255
	Blue  int `json:"blue"`  // 0-255
}

// LightSchedule 燈光排程
type LightSchedule struct {
	ID         string        `json:"id"`
	Name       string        `json:"name"`
	StartTime  string        `json:"start_time"` // HH:MM
	EndTime    string        `json:"end_time"`   // HH:MM
	Days       []string      `json:"days"`       // monday, tuesday, ...
	Brightness int           `json:"brightness"`
	Color      *Color        `json:"color,omitempty"`
	ColorTemp  int           `json:"color_temp,omitempty"`
	FadeTime   time.Duration `json:"fade_time"`
	Enabled    bool          `json:"enabled"`
}

// LightUsagePattern 使用模式
type LightUsagePattern struct {
	TimeRange   TimeRange     `json:"time_range"`
	Probability float64       `json:"probability"`
	Behavior    LightBehavior `json:"behavior"`
}

// TimeRange 時間範圍
type TimeRange struct {
	StartTime string   `json:"start_time"`
	EndTime   string   `json:"end_time"`
	Days      []string `json:"days"`
}

// LightBehavior 燈光行為
type LightBehavior struct {
	Power      bool   `json:"power"`
	Brightness int    `json:"brightness"`
	Color      *Color `json:"color,omitempty"`
	Mode       string `json:"mode"`
}

// NewSmartBulb 建立新的智慧燈泡模擬器
func NewSmartBulb(config base.DeviceConfig, mqttConfig base.MQTTConfig) (*SmartBulb, error) {
	baseDevice := base.NewBaseDeviceWithMQTT(config, mqttConfig)

	bulb := &SmartBulb{
		BaseDevice:   baseDevice,
		power:        false,
		brightness:   100,
		color:        Color{Red: 255, Green: 255, Blue: 255}, // 預設白光
		colorTemp:    3000,                                   // 暖白光
		mode:         "normal",
		maxWattage:   9.0,   // 9W LED 燈泡
		efficiency:   100.0, // 100 lumens/watt
		ambientLight: 100,   // 預設環境光
		autoMode:     true,
		fadeTime:     2 * time.Second,
		lifespan:     25000 * time.Hour, // 25000 小時壽命
		// logger removed
	}

	// 初始化排程
	bulb.initializeSchedules()

	return bulb, nil
}

// Start 啟動智慧燈泡模擬器
func (s *SmartBulb) Start(ctx context.Context) error {
	if err := s.BaseDevice.Start(ctx); err != nil {
		return err
	}

	// logger removed

	// 啟動燈泡特定的處理循環
	go s.lightControlLoop(ctx)
	go s.ambientSensingLoop(ctx)
	go s.scheduleCheckLoop(ctx)

	return nil
}

// Stop 停止智慧燈泡模擬器
func (s *SmartBulb) Stop() error {
	// logger removed

	// 記錄最後的使用時間
	if s.power {
		s.onTime += time.Since(s.lastOn)
	}

	return s.BaseDevice.Stop()
}

// GenerateStatePayload 生成智慧燈泡狀態數據
func (s *SmartBulb) GenerateStatePayload() base.StatePayload {
	basePayload := s.BaseDevice.GenerateStatePayload()

	// 計算當前功率消耗
	currentWattage := s.calculateCurrentWattage()

	// 計算光輸出
	lumens := s.calculateLumens()

	// 添加燈泡特定狀態
	basePayload.Extra = map[string]interface{}{
		"light_status": map[string]interface{}{
			"power":        s.power,
			"brightness":   s.brightness,
			"color":        s.color,
			"color_temp":   s.colorTemp,
			"mode":         s.mode,
			"wattage":      currentWattage,
			"lumens":       lumens,
			"fade_time_ms": s.fadeTime.Milliseconds(),
		},
		"sensors": map[string]interface{}{
			"ambient_light":   s.ambientLight,
			"motion_detected": s.motionSensor,
			"last_motion":     s.lastMotion,
		},
		"automation": map[string]interface{}{
			"auto_mode":        s.autoMode,
			"active_schedules": s.getActiveSchedules(),
		},
		"health": map[string]interface{}{
			"total_on_time_hours":   s.onTime.Hours(),
			"switch_count":          s.switchCount,
			"lifespan_used_percent": (s.usedLife.Hours() / s.lifespan.Hours()) * 100,
			"degradation_percent":   s.degradation * 100,
		},
	}

	return basePayload
}

// GenerateTelemetryData 生成智慧燈泡遙測數據
func (s *SmartBulb) GenerateTelemetryData() map[string]base.TelemetryPayload {
	telemetry := s.BaseDevice.GenerateTelemetryData()

	// 光照資訊
	telemetry["lighting"] = base.TelemetryPayload{
		"power":       s.power,
		"brightness":  s.brightness,
		"color_red":   s.color.Red,
		"color_green": s.color.Green,
		"color_blue":  s.color.Blue,
		"color_temp":  s.colorTemp,
		"wattage":     s.calculateCurrentWattage(),
		"lumens":      s.calculateLumens(),
		"efficiency":  s.efficiency,
	}

	// 環境感測
	telemetry["environment"] = base.TelemetryPayload{
		"ambient_light":   s.ambientLight,
		"motion_detected": s.motionSensor,
		"auto_mode":       s.autoMode,
	}

	// 能耗統計
	telemetry["energy"] = base.TelemetryPayload{
		"current_wattage":   s.calculateCurrentWattage(),
		"daily_kwh":         s.calculateDailyEnergy(),
		"monthly_kwh":       s.calculateMonthlyEnergy(),
		"efficiency_rating": s.getEfficiencyRating(),
	}

	// 使用統計
	telemetry["usage_stats"] = base.TelemetryPayload{
		"total_on_time_hours": s.onTime.Hours(),
		"switch_count":        s.switchCount,
		"average_brightness":  s.calculateAverageBrightness(),
		"most_used_mode":      s.getMostUsedMode(),
	}

	return telemetry
}

// GenerateEvents 生成智慧燈泡事件
func (s *SmartBulb) GenerateEvents() []base.Event {
	var events []base.Event

	// 動作感測事件
	if s.motionSensor && time.Since(s.lastMotion) < 5*time.Second {
		events = append(events, base.Event{
			EventType: "bulb.motion_detected",
			Severity:  "info",
			Message:   "Motion detected, auto-adjusting lighting",
			Extra: map[string]interface{}{
				"ambient_light":      s.ambientLight,
				"current_brightness": s.brightness,
				"auto_adjustment":    s.autoMode,
			},
		})
	}

	// 燈泡壽命警告
	lifeUsedPercent := (s.usedLife.Hours() / s.lifespan.Hours()) * 100
	if lifeUsedPercent > 80 && rand.Float64() < 0.1 { // 80% 壽命，10% 機率警告
		events = append(events, base.Event{
			EventType: "bulb.lifespan_warning",
			Severity:  "warning",
			Message:   fmt.Sprintf("Bulb lifespan is %d%% used", int(lifeUsedPercent)),
			Extra: map[string]interface{}{
				"lifespan_used_percent": lifeUsedPercent,
				"remaining_hours":       s.lifespan.Hours() - s.usedLife.Hours(),
				"switch_count":          s.switchCount,
			},
		})
	}

	// 高能耗警告
	currentWattage := s.calculateCurrentWattage()
	if currentWattage > s.maxWattage*0.9 { // 90% 以上功率
		events = append(events, base.Event{
			EventType: "bulb.high_power_consumption",
			Severity:  "info",
			Message:   "High power consumption detected",
			Extra: map[string]interface{}{
				"current_wattage": currentWattage,
				"max_wattage":     s.maxWattage,
				"brightness":      s.brightness,
				"color_temp":      s.colorTemp,
			},
		})
	}

	// 排程觸發事件
	if activeSchedule := s.getCurrentActiveSchedule(); activeSchedule != nil {
		events = append(events, base.Event{
			EventType: "bulb.schedule_triggered",
			Severity:  "info",
			Message:   fmt.Sprintf("Schedule '%s' activated", activeSchedule.Name),
			Extra: map[string]interface{}{
				"schedule_name":     activeSchedule.Name,
				"target_brightness": activeSchedule.Brightness,
				"target_color":      activeSchedule.Color,
				"fade_time_ms":      activeSchedule.FadeTime.Milliseconds(),
			},
		})
	}

	return events
}

// HandleCommand 處理智慧燈泡命令
func (s *SmartBulb) HandleCommand(cmd base.Command) error {
	// logger removed.Info("Handling smart bulb command")

	switch cmd.Type {
	case "bulb.set_power":
		return s.handleSetPower(cmd)
	case "bulb.set_brightness":
		return s.handleSetBrightness(cmd)
	case "bulb.set_color":
		return s.handleSetColor(cmd)
	case "bulb.set_color_temp":
		return s.handleSetColorTemp(cmd)
	case "bulb.set_mode":
		return s.handleSetMode(cmd)
	case "bulb.toggle":
		return s.handleToggle(cmd)
	case "bulb.fade_to":
		return s.handleFadeTo(cmd)
	case "bulb.create_schedule":
		return s.handleCreateSchedule(cmd)
	case "bulb.delete_schedule":
		return s.handleDeleteSchedule(cmd)
	default:
		return s.BaseDevice.HandleCommand(cmd)
	}
}

// 內部方法實作
func (s *SmartBulb) initializeSchedules() {
	s.schedules = []LightSchedule{
		{
			ID:         "evening_warm",
			Name:       "Evening Warm Light",
			StartTime:  "18:00",
			EndTime:    "22:00",
			Days:       []string{"monday", "tuesday", "wednesday", "thursday", "friday"},
			Brightness: 70,
			ColorTemp:  2700,
			FadeTime:   5 * time.Second,
			Enabled:    true,
		},
		{
			ID:         "night_mode",
			Name:       "Night Mode",
			StartTime:  "22:00",
			EndTime:    "06:00",
			Days:       []string{"monday", "tuesday", "wednesday", "thursday", "friday", "saturday", "sunday"},
			Brightness: 10,
			Color:      &Color{Red: 255, Green: 100, Blue: 0}, // 橙色
			FadeTime:   3 * time.Second,
			Enabled:    true,
		},
	}
}

// 燈光控制循環
func (s *SmartBulb) lightControlLoop(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.updateLightStatus()
			s.processAutoMode()
		}
	}
}

// 環境感測循環
func (s *SmartBulb) ambientSensingLoop(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.updateAmbientSensing()
		}
	}
}

// 排程檢查循環
func (s *SmartBulb) scheduleCheckLoop(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.checkSchedules()
		}
	}
}

// 輔助方法
func (s *SmartBulb) calculateCurrentWattage() float64 {
	if !s.power {
		return 0.0
	}

	// 根據亮度計算功率消耗
	basePower := s.maxWattage * (float64(s.brightness) / 100.0)

	// 考慮顏色對功率的影響 (彩色通常比白光耗電)
	if s.color.Red != 255 || s.color.Green != 255 || s.color.Blue != 255 {
		basePower *= 1.1 // 彩色模式增加 10% 功耗
	}

	return basePower
}

func (s *SmartBulb) calculateLumens() float64 {
	if !s.power {
		return 0.0
	}

	// 計算光輸出 (lumens = watts * efficiency * brightness_factor * degradation)
	wattage := s.calculateCurrentWattage()
	brightnessFactor := float64(s.brightness) / 100.0
	degradationFactor := 1.0 - s.degradation

	return wattage * s.efficiency * brightnessFactor * degradationFactor
}

func (s *SmartBulb) updateLightStatus() {
	// 更新使用時間
	if s.power {
		s.usedLife += time.Second
		if s.lastOn.IsZero() {
			s.lastOn = time.Now()
		}
	} else {
		if !s.lastOn.IsZero() {
			s.onTime += time.Since(s.lastOn)
			s.lastOn = time.Time{}
		}
	}

	// 計算光衰減
	hoursUsed := s.usedLife.Hours()
	s.degradation = math.Min(hoursUsed/s.lifespan.Hours()*0.3, 0.3) // 最大 30% 衰減
}

func (s *SmartBulb) updateAmbientSensing() {
	// 模擬環境光變化
	hour := time.Now().Hour()
	var baseLight int

	if hour >= 6 && hour < 8 {
		baseLight = 200 + rand.Intn(100) // 清晨
	} else if hour >= 8 && hour < 18 {
		baseLight = 500 + rand.Intn(300) // 白天
	} else if hour >= 18 && hour < 20 {
		baseLight = 200 + rand.Intn(100) // 傍晚
	} else {
		baseLight = 10 + rand.Intn(50) // 夜晚
	}

	s.ambientLight = baseLight

	// 模擬動作感測
	if rand.Float64() < 0.05 { // 5% 機率檢測到動作
		s.motionSensor = true
		s.lastMotion = time.Now()
	} else if time.Since(s.lastMotion) > 30*time.Second {
		s.motionSensor = false
	}
}

func (s *SmartBulb) processAutoMode() {
	if !s.autoMode {
		return
	}

	// 根據環境光自動調整亮度
	if s.power {
		targetBrightness := s.calculateAutoBrightness()
		if abs(targetBrightness-s.brightness) > 10 { // 變化超過 10% 才調整
			s.brightness = targetBrightness
		}
	}

	// 動作感測自動開燈
	if s.motionSensor && !s.power && s.ambientLight < 100 {
		s.power = true
		s.brightness = 50 // 中等亮度
		s.switchCount++
		s.lastOn = time.Now()
	}
}

func (s *SmartBulb) calculateAutoBrightness() int {
	// 根據環境光計算目標亮度
	if s.ambientLight > 500 {
		return 30 // 白天低亮度
	} else if s.ambientLight > 200 {
		return 50 // 中等亮度
	} else if s.ambientLight > 50 {
		return 70 // 較高亮度
	} else {
		return 90 // 夜晚高亮度
	}
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// 命令處理方法
func (s *SmartBulb) handleSetPower(cmd base.Command) error {
	power, ok := cmd.Parameters["power"].(bool)
	if !ok {
		return fmt.Errorf("invalid power parameter")
	}

	oldPower := s.power
	s.power = power

	if power && !oldPower {
		s.switchCount++
		s.lastOn = time.Now()
	} else if !power && oldPower {
		s.onTime += time.Since(s.lastOn)
		s.lastOn = time.Time{}
	}

	// logger removed.Info("Power state changed")
	return nil
}

func (s *SmartBulb) handleSetBrightness(cmd base.Command) error {
	brightness, ok := cmd.Parameters["brightness"].(float64)
	if !ok {
		return fmt.Errorf("invalid brightness parameter")
	}

	if brightness < 0 || brightness > 100 {
		return fmt.Errorf("brightness must be between 0 and 100")
	}

	s.brightness = int(brightness)
	// logger removed.Info("Brightness changed")
	return nil
}

func (s *SmartBulb) handleSetColor(cmd base.Command) error {
	colorMap, ok := cmd.Parameters["color"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid color parameter")
	}

	red, _ := colorMap["red"].(float64)
	green, _ := colorMap["green"].(float64)
	blue, _ := colorMap["blue"].(float64)

	s.color = Color{
		Red:   int(red),
		Green: int(green),
		Blue:  int(blue),
	}

	// logger removed.Info("Color changed")
	return nil
}

// 實作其他必要的方法...
func (s *SmartBulb) handleSetColorTemp(cmd base.Command) error   { return nil }
func (s *SmartBulb) handleSetMode(cmd base.Command) error        { return nil }
func (s *SmartBulb) handleToggle(cmd base.Command) error         { return nil }
func (s *SmartBulb) handleFadeTo(cmd base.Command) error         { return nil }
func (s *SmartBulb) handleCreateSchedule(cmd base.Command) error { return nil }
func (s *SmartBulb) handleDeleteSchedule(cmd base.Command) error { return nil }
func (s *SmartBulb) checkSchedules()                             {}
func (s *SmartBulb) getActiveSchedules() []string                { return []string{} }
func (s *SmartBulb) getCurrentActiveSchedule() *LightSchedule    { return nil }
func (s *SmartBulb) calculateDailyEnergy() float64               { return 0.0 }
func (s *SmartBulb) calculateMonthlyEnergy() float64             { return 0.0 }
func (s *SmartBulb) getEfficiencyRating() string                 { return "A+" }
func (s *SmartBulb) calculateAverageBrightness() float64         { return 70.0 }
func (s *SmartBulb) getMostUsedMode() string                     { return "normal" }
