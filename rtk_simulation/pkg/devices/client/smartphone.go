package client

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"rtk_simulation/pkg/devices/base"
	// "github.com/sirupsen/logrus"
)

// Smartphone 智慧手機模擬器
type Smartphone struct {
	*base.BaseDevice

	// 基本資訊
	model        string // 手機型號
	os           string // 作業系統 (iOS, Android)
	osVersion    string // 系統版本
	manufacturer string // 製造商

	// 網路連接
	wifiConnected   bool   // WiFi 連接狀態
	mobileConnected bool   // 行動網路連接狀態
	preferredBand   string // 偏好頻段 (2.4GHz, 5GHz)
	currentSSID     string // 目前連接的 WiFi SSID
	rssi            int    // WiFi 信號強度

	// 電池狀態
	batteryLevel  int  // 電池電量 (0-100%)
	charging      bool // 充電狀態
	batteryHealth int  // 電池健康度 (0-100%)
	powerSaveMode bool // 省電模式

	// 使用狀態
	screenOn     bool       // 螢幕狀態
	inCall       bool       // 通話中
	activeApps   []AppUsage // 使用中的應用程式
	lastActivity time.Time  // 最後活動時間

	// 位置和移動
	location      Location // 目前位置
	moving        bool     // 移動狀態
	movementSpeed float64  // 移動速度 (km/h)

	// 網路使用
	dataUsage DataUsage // 數據使用情況
	wifiUsage DataUsage // WiFi 使用情況

	// 應用程式狀態
	installedApps []AppInfo      // 安裝的應用程式
	notifications []Notification // 通知

	// 使用模式
	usageProfile    UsageProfile    // 使用配置
	behaviorPattern BehaviorPattern // 行為模式

	// 安全狀態
	locked        bool      // 鎖定狀態
	biometricAuth bool      // 生物識別認證
	lastUnlock    time.Time // 最後解鎖時間

	// MQTT 客戶端
	mqttClient *base.MQTTClient

	// 日誌
}

// AppUsage 應用程式使用情況
type AppUsage struct {
	Name         string        `json:"name"`
	PackageName  string        `json:"package_name"`
	Category     string        `json:"category"`
	StartTime    time.Time     `json:"start_time"`
	Duration     time.Duration `json:"duration"`
	DataUsed     int64         `json:"data_used"`    // bytes
	BatteryUsed  float64       `json:"battery_used"` // percentage
	InForeground bool          `json:"in_foreground"`
}

// AppInfo 應用程式資訊
type AppInfo struct {
	Name        string    `json:"name"`
	PackageName string    `json:"package_name"`
	Category    string    `json:"category"`
	Version     string    `json:"version"`
	Size        int64     `json:"size"` // bytes
	InstallDate time.Time `json:"install_date"`
	LastUsed    time.Time `json:"last_used"`
	Permissions []string  `json:"permissions"`
}

// Location 位置資訊
type Location struct {
	Latitude  float64   `json:"latitude"`
	Longitude float64   `json:"longitude"`
	Altitude  float64   `json:"altitude"`
	Accuracy  float64   `json:"accuracy"` // meters
	Timestamp time.Time `json:"timestamp"`
	Address   string    `json:"address,omitempty"`
}

// DataUsage 數據使用情況
type DataUsage struct {
	TotalBytes    int64     `json:"total_bytes"`
	UploadBytes   int64     `json:"upload_bytes"`
	DownloadBytes int64     `json:"download_bytes"`
	ResetDate     time.Time `json:"reset_date"`
	DailyLimit    int64     `json:"daily_limit,omitempty"`
	MonthlyLimit  int64     `json:"monthly_limit,omitempty"`
}

// Notification 通知
type Notification struct {
	ID        string    `json:"id"`
	AppName   string    `json:"app_name"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
	Priority  string    `json:"priority"` // low, normal, high, urgent
	Category  string    `json:"category"`
	Read      bool      `json:"read"`
}

// UsageProfile 使用配置
type UsageProfile struct {
	Profile       string        `json:"profile"` // light, moderate, heavy, gaming
	DailyHours    float64       `json:"daily_hours"`
	PeakHours     []TimeRange   `json:"peak_hours"`
	IdleTimeout   time.Duration `json:"idle_timeout"`
	AppCategories []string      `json:"app_categories"`
}

// BehaviorPattern 行為模式
type BehaviorPattern struct {
	WakeUpTime  string      `json:"wake_up_time"`
	SleepTime   string      `json:"sleep_time"`
	WorkHours   TimeRange   `json:"work_hours"`
	BreakTimes  []TimeRange `json:"break_times"`
	WeekendDiff bool        `json:"weekend_different"`
	Predictable bool        `json:"predictable"`
}

// TimeRange 時間範圍
type TimeRange struct {
	StartTime string   `json:"start_time"`
	EndTime   string   `json:"end_time"`
	Days      []string `json:"days"`
}

// NewSmartphone 建立新的智慧手機模擬器
func NewSmartphone(config base.DeviceConfig, mqttConfig base.MQTTConfig) (*Smartphone, error) {
	baseDevice := base.NewBaseDevice(config)

	// 建立 MQTT 客戶端
	mqttClient := base.NewMQTTClient(mqttConfig, baseDevice)

	phone := &Smartphone{
		BaseDevice:      baseDevice,
		model:           "Generic Smartphone",
		os:              "Android",
		osVersion:       "13.0",
		manufacturer:    "Generic",
		wifiConnected:   true,
		mobileConnected: true,
		preferredBand:   "5GHz",
		currentSSID:     "HomeNetwork_5G",
		rssi:            -45,
		batteryLevel:    75,
		charging:        false,
		batteryHealth:   95,
		powerSaveMode:   false,
		screenOn:        false,
		locked:          true,
		biometricAuth:   true,
		location: Location{
			Latitude:  25.0330, // 台北
			Longitude: 121.5654,
			Accuracy:  10.0,
		},
		mqttClient: mqttClient,
	}

	// 初始化使用配置
	phone.initializeUsageProfile()

	// 初始化安裝的應用程式
	phone.initializeInstalledApps()

	return phone, nil
}

// Start 啟動智慧手機模擬器
func (s *Smartphone) Start(ctx context.Context) error {
	if err := s.BaseDevice.Start(ctx); err != nil {
		return err
	}

	// 連接 MQTT
	if err := s.mqttClient.Connect(); err != nil {
	}

	// 啟動手機特定的處理循環
	go s.batteryManagementLoop(ctx)
	go s.networkManagementLoop(ctx)
	go s.usageSimulationLoop(ctx)
	go s.locationUpdateLoop(ctx)
	go s.publishingLoop(ctx)

	return nil
}

// Stop 停止智慧手機模擬器
func (s *Smartphone) Stop() error {

	// 斷開 MQTT
	if s.mqttClient != nil {
		s.mqttClient.Disconnect()
	}

	return s.BaseDevice.Stop()
}

// GenerateStatePayload 生成智慧手機狀態數據
func (s *Smartphone) GenerateStatePayload() base.StatePayload {
	basePayload := s.BaseDevice.GenerateStatePayload()

	// 添加手機特定狀態
	basePayload.Extra = map[string]interface{}{
		"device_info": map[string]interface{}{
			"model":        s.model,
			"os":           s.os,
			"os_version":   s.osVersion,
			"manufacturer": s.manufacturer,
		},
		"connectivity": map[string]interface{}{
			"wifi_connected":   s.wifiConnected,
			"mobile_connected": s.mobileConnected,
			"current_ssid":     s.currentSSID,
			"rssi":             s.rssi,
			"preferred_band":   s.preferredBand,
		},
		"battery": map[string]interface{}{
			"level":          s.batteryLevel,
			"charging":       s.charging,
			"health":         s.batteryHealth,
			"power_save":     s.powerSaveMode,
			"estimated_time": s.getEstimatedBatteryTime(),
		},
		"usage": map[string]interface{}{
			"screen_on":     s.screenOn,
			"locked":        s.locked,
			"in_call":       s.inCall,
			"active_apps":   len(s.activeApps),
			"last_activity": s.lastActivity,
		},
		"location": map[string]interface{}{
			"latitude":  s.location.Latitude,
			"longitude": s.location.Longitude,
			"accuracy":  s.location.Accuracy,
			"moving":    s.moving,
			"speed":     s.movementSpeed,
		},
	}

	return basePayload
}

// GenerateTelemetryData 生成智慧手機遙測數據
func (s *Smartphone) GenerateTelemetryData() map[string]base.TelemetryPayload {
	telemetry := s.BaseDevice.GenerateTelemetryData()

	// 電池統計
	telemetry["battery_stats"] = base.TelemetryPayload{
		"level":           s.batteryLevel,
		"health":          s.batteryHealth,
		"charging":        s.charging,
		"power_save_mode": s.powerSaveMode,
		"drain_rate":      s.calculateBatteryDrainRate(),
		"charge_cycles":   s.getBatteryCycles(),
		"temperature":     s.getBatteryTemperature(),
	}

	// 網路統計
	telemetry["network_stats"] = base.TelemetryPayload{
		"wifi_data_mb":       s.wifiUsage.TotalBytes / 1024 / 1024,
		"mobile_data_mb":     s.dataUsage.TotalBytes / 1024 / 1024,
		"signal_strength":    s.rssi,
		"connection_quality": s.getConnectionQuality(),
		"roaming_events":     s.getRoamingEvents(),
		"connection_drops":   s.getConnectionDrops(),
	}

	// 使用統計
	telemetry["usage_stats"] = base.TelemetryPayload{
		"screen_time_hours": s.getScreenTimeToday(),
		"app_switches":      s.getAppSwitches(),
		"notifications":     len(s.notifications),
		"calls_today":       s.getCallsToday(),
		"top_apps":          s.getTopApps(),
		"usage_pattern":     s.getCurrentUsagePattern(),
	}

	// 位置統計
	telemetry["location_stats"] = base.TelemetryPayload{
		"latitude":          s.location.Latitude,
		"longitude":         s.location.Longitude,
		"accuracy":          s.location.Accuracy,
		"moving":            s.moving,
		"speed_kmh":         s.movementSpeed,
		"distance_today_km": s.getDistanceToday(),
		"places_visited":    s.getPlacesVisited(),
	}

	return telemetry
}

// GenerateEvents 生成智慧手機事件
func (s *Smartphone) GenerateEvents() []base.Event {
	var events []base.Event

	// 低電量警告
	if s.batteryLevel <= 20 && !s.charging && rand.Float64() < 0.1 {
		events = append(events, base.Event{
			EventType: "phone.battery_low",
			Severity:  "warning",
			Message:   fmt.Sprintf("Battery level is low: %d%%", s.batteryLevel),
			Extra: map[string]interface{}{
				"battery_level":   s.batteryLevel,
				"power_save_mode": s.powerSaveMode,
				"estimated_time":  s.getEstimatedBatteryTime(),
			},
		})
	}

	// WiFi 連接狀態變化
	if rand.Float64() < 0.02 { // 2% 機率
		if s.wifiConnected && rand.Float64() < 0.3 { // 30% 斷開
			events = append(events, base.Event{
				EventType: "phone.wifi_disconnected",
				Severity:  "info",
				Message:   "WiFi connection lost",
				Extra: map[string]interface{}{
					"previous_ssid": s.currentSSID,
					"rssi":          s.rssi,
					"fallback":      "mobile_data",
				},
			})
		} else if !s.wifiConnected && rand.Float64() < 0.7 { // 70% 連接
			events = append(events, base.Event{
				EventType: "phone.wifi_connected",
				Severity:  "info",
				Message:   "WiFi connection established",
				Extra: map[string]interface{}{
					"ssid":          s.currentSSID,
					"rssi":          s.rssi,
					"security_type": "WPA3",
				},
			})
		}
	}

	// 應用程式崩潰
	if len(s.activeApps) > 0 && rand.Float64() < 0.01 { // 1% 機率
		app := s.activeApps[rand.Intn(len(s.activeApps))]
		events = append(events, base.Event{
			EventType: "phone.app_crash",
			Severity:  "warning",
			Message:   fmt.Sprintf("Application %s crashed", app.Name),
			Extra: map[string]interface{}{
				"app_name":     app.Name,
				"package_name": app.PackageName,
				"crash_reason": "out_of_memory",
				"runtime":      time.Since(app.StartTime).Minutes(),
			},
		})
	}

	// 收到通知
	if rand.Float64() < 0.15 { // 15% 機率
		notificationTypes := []string{"message", "email", "social", "news", "system"}
		notifType := notificationTypes[rand.Intn(len(notificationTypes))]

		events = append(events, base.Event{
			EventType: "phone.notification_received",
			Severity:  "info",
			Message:   "New notification received",
			Extra: map[string]interface{}{
				"type":       notifType,
				"app":        s.getRandomApp(),
				"priority":   "normal",
				"actionable": rand.Float64() < 0.4,
			},
		})
	}

	// 位置變化
	if s.moving && rand.Float64() < 0.05 { // 5% 機率
		events = append(events, base.Event{
			EventType: "phone.location_update",
			Severity:  "info",
			Message:   "Significant location change detected",
			Extra: map[string]interface{}{
				"latitude":    s.location.Latitude,
				"longitude":   s.location.Longitude,
				"accuracy":    s.location.Accuracy,
				"speed":       s.movementSpeed,
				"travel_mode": s.getTravelMode(),
			},
		})
	}

	return events
}

// HandleCommand 處理智慧手機命令
func (s *Smartphone) HandleCommand(cmd base.Command) error {

	switch cmd.Type {
	case "phone.unlock":
		return s.handleUnlock(cmd)
	case "phone.lock":
		return s.handleLock(cmd)
	case "phone.launch_app":
		return s.handleLaunchApp(cmd)
	case "phone.make_call":
		return s.handleMakeCall(cmd)
	case "phone.connect_wifi":
		return s.handleConnectWiFi(cmd)
	case "phone.enable_power_save":
		return s.handleEnablePowerSave(cmd)
	case "phone.update_location":
		return s.handleUpdateLocation(cmd)
	default:
		return s.BaseDevice.HandleCommand(cmd)
	}
}

// 內部方法實作
func (s *Smartphone) initializeUsageProfile() {
	s.usageProfile = UsageProfile{
		Profile:    "moderate",
		DailyHours: 4.5,
		PeakHours: []TimeRange{
			{StartTime: "08:00", EndTime: "09:00", Days: []string{"monday", "tuesday", "wednesday", "thursday", "friday"}},
			{StartTime: "12:00", EndTime: "13:00", Days: []string{"monday", "tuesday", "wednesday", "thursday", "friday"}},
			{StartTime: "19:00", EndTime: "22:00", Days: []string{"monday", "tuesday", "wednesday", "thursday", "friday", "saturday", "sunday"}},
		},
		IdleTimeout:   30 * time.Second,
		AppCategories: []string{"social", "entertainment", "productivity", "communication"},
	}

	s.behaviorPattern = BehaviorPattern{
		WakeUpTime:  "07:00",
		SleepTime:   "23:00",
		WorkHours:   TimeRange{StartTime: "09:00", EndTime: "18:00"},
		WeekendDiff: true,
		Predictable: true,
	}
}

func (s *Smartphone) initializeInstalledApps() {
	s.installedApps = []AppInfo{
		{Name: "Messages", PackageName: "com.android.messaging", Category: "communication", Version: "1.0"},
		{Name: "Phone", PackageName: "com.android.dialer", Category: "communication", Version: "1.0"},
		{Name: "Chrome", PackageName: "com.android.chrome", Category: "web", Version: "110.0"},
		{Name: "WhatsApp", PackageName: "com.whatsapp", Category: "social", Version: "2.23"},
		{Name: "YouTube", PackageName: "com.google.android.youtube", Category: "entertainment", Version: "18.0"},
		{Name: "Gmail", PackageName: "com.google.android.gm", Category: "productivity", Version: "2023.01"},
		{Name: "Maps", PackageName: "com.google.android.apps.maps", Category: "navigation", Version: "11.0"},
		{Name: "Camera", PackageName: "com.android.camera2", Category: "multimedia", Version: "1.0"},
	}
}

// 循環處理方法
func (s *Smartphone) batteryManagementLoop(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.updateBatteryStatus()
		}
	}
}

func (s *Smartphone) networkManagementLoop(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.updateNetworkStatus()
		}
	}
}

func (s *Smartphone) usageSimulationLoop(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.simulateUsagePattern()
		}
	}
}

func (s *Smartphone) locationUpdateLoop(ctx context.Context) {
	ticker := time.NewTicker(2 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.updateLocation()
		}
	}
}

// MQTT 發布循環
func (s *Smartphone) publishingLoop(ctx context.Context) {
	stateTicker := time.NewTicker(45 * time.Second)
	telemetryTicker := time.NewTicker(25 * time.Second)
	defer stateTicker.Stop()
	defer telemetryTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-stateTicker.C:
			if s.mqttClient != nil && s.mqttClient.IsConnected() {
				if err := s.mqttClient.PublishState(s.GenerateStatePayload()); err != nil {
				}
			}
		case <-telemetryTicker.C:
			if s.mqttClient != nil && s.mqttClient.IsConnected() {
				telemetryData := s.GenerateTelemetryData()
				for metric, payload := range telemetryData {
					if err := s.mqttClient.PublishTelemetry(metric, payload); err != nil {
					}
				}

				// 發布事件
				events := s.GenerateEvents()
				for _, event := range events {
					if err := s.mqttClient.PublishEvent(event.EventType, event); err != nil {
					}
				}
			}
		}
	}
}

// 輔助方法實作
func (s *Smartphone) updateBatteryStatus() {
	if s.charging {
		// 充電中
		if s.batteryLevel < 100 {
			s.batteryLevel += rand.Intn(3) + 1 // 1-3% 每分鐘
			if s.batteryLevel > 100 {
				s.batteryLevel = 100
			}
		}
	} else {
		// 消耗電池
		drainRate := 1 // 基礎消耗
		if s.screenOn {
			drainRate += 2 // 螢幕開啟增加消耗
		}
		if len(s.activeApps) > 0 {
			drainRate += len(s.activeApps) // 每個活躍應用增加消耗
		}
		if s.inCall {
			drainRate += 3 // 通話中增加消耗
		}
		if s.powerSaveMode {
			drainRate = drainRate / 2 // 省電模式減半
		}

		s.batteryLevel -= drainRate
		if s.batteryLevel < 0 {
			s.batteryLevel = 0
		}

		// 自動啟用省電模式
		if s.batteryLevel <= 15 && !s.powerSaveMode {
			s.powerSaveMode = true
		}
	}
}

func (s *Smartphone) updateNetworkStatus() {
	// 模擬網路狀態變化
	if s.wifiConnected {
		// WiFi 信號強度變化
		s.rssi += rand.Intn(10) - 5 // ±5 dBm
		if s.rssi > -20 {
			s.rssi = -20
		} else if s.rssi < -90 {
			s.rssi = -90
		}

		// WiFi 斷線機率
		if s.rssi < -80 && rand.Float64() < 0.1 {
			s.wifiConnected = false
			s.currentSSID = ""
		}
	} else {
		// 嘗試重新連接 WiFi
		if rand.Float64() < 0.3 { // 30% 機率
			s.wifiConnected = true
			s.currentSSID = "HomeNetwork_5G"
			s.rssi = -45 + rand.Intn(20) // -45 to -65 dBm
		}
	}
}

// 實作其他必要的方法...
func (s *Smartphone) simulateUsagePattern()              {}
func (s *Smartphone) updateLocation()                    {}
func (s *Smartphone) getEstimatedBatteryTime() string    { return "3h 45m" }
func (s *Smartphone) calculateBatteryDrainRate() float64 { return 2.5 }
func (s *Smartphone) getBatteryCycles() int              { return 450 }
func (s *Smartphone) getBatteryTemperature() float64     { return 32.5 }
func (s *Smartphone) getConnectionQuality() string       { return "excellent" }
func (s *Smartphone) getRoamingEvents() int              { return 2 }
func (s *Smartphone) getConnectionDrops() int            { return 1 }
func (s *Smartphone) getScreenTimeToday() float64        { return 4.2 }
func (s *Smartphone) getAppSwitches() int                { return 45 }
func (s *Smartphone) getCallsToday() int                 { return 3 }
func (s *Smartphone) getTopApps() []string               { return []string{"Chrome", "WhatsApp", "YouTube"} }
func (s *Smartphone) getCurrentUsagePattern() string     { return "moderate" }
func (s *Smartphone) getDistanceToday() float64          { return 12.5 }
func (s *Smartphone) getPlacesVisited() int              { return 4 }
func (s *Smartphone) getRandomApp() string               { return "WhatsApp" }
func (s *Smartphone) getTravelMode() string              { return "walking" }

// 命令處理方法
func (s *Smartphone) handleUnlock(cmd base.Command) error          { return nil }
func (s *Smartphone) handleLock(cmd base.Command) error            { return nil }
func (s *Smartphone) handleLaunchApp(cmd base.Command) error       { return nil }
func (s *Smartphone) handleMakeCall(cmd base.Command) error        { return nil }
func (s *Smartphone) handleConnectWiFi(cmd base.Command) error     { return nil }
func (s *Smartphone) handleEnablePowerSave(cmd base.Command) error { return nil }
func (s *Smartphone) handleUpdateLocation(cmd base.Command) error  { return nil }
