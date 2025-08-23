package client

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"sync"
	"time"

	// // "github.com/sirupsen/logrus"
	// // "rtk_simulation/pkg/config"
	"rtk_simulation/pkg/devices/base"
)

type Laptop struct {
	*base.BaseDevice

	batteryLevel     float64
	isCharging       bool
	powerMode        string
	screenBrightness int
	applications     []RunningApplication
	networkActivity  NetworkActivity
	systemResources  SystemResources
	mu               sync.RWMutex
}

type RunningApplication struct {
	Name         string    `json:"name"`
	Type         string    `json:"type"`
	CPUUsage     float64   `json:"cpu_usage"`
	MemoryUsage  float64   `json:"memory_usage"`
	NetworkUsage float64   `json:"network_usage"`
	StartTime    time.Time `json:"start_time"`
	Active       bool      `json:"active"`
}

type NetworkActivity struct {
	CurrentUpload   float64   `json:"current_upload"`   // Mbps
	CurrentDownload float64   `json:"current_download"` // Mbps
	TotalUpload     float64   `json:"total_upload"`     // MB
	TotalDownload   float64   `json:"total_download"`   // MB
	ConnectedSSID   string    `json:"connected_ssid"`
	SignalStrength  float64   `json:"signal_strength"`
	ConnectionType  string    `json:"connection_type"`
	LastActivity    time.Time `json:"last_activity"`
}

type SystemResources struct {
	CPUUsage       float64 `json:"cpu_usage"`
	MemoryUsage    float64 `json:"memory_usage"`
	DiskUsage      float64 `json:"disk_usage"`
	Temperature    float64 `json:"temperature"`
	FanSpeed       int     `json:"fan_speed"`
	NetworkLatency float64 `json:"network_latency"`
}

func NewLaptop(deviceConfig base.DeviceConfig, mqttConfig base.MQTTConfig) (*Laptop, error) {
	baseDevice := base.NewBaseDevice(deviceConfig)

	laptop := &Laptop{
		BaseDevice:       baseDevice,
		batteryLevel:     85.0,
		isCharging:       false,
		powerMode:        "balanced",
		screenBrightness: 75,
	}

	laptop.initializeApplications(deviceConfig)
	laptop.initializeNetworkActivity()
	laptop.initializeSystemResources()

	return laptop, nil
}

func (l *Laptop) initializeApplications(config base.DeviceConfig) {
	apps := []RunningApplication{
		{Name: "Web Browser", Type: "productivity", CPUUsage: 15.0, MemoryUsage: 1200.0, NetworkUsage: 2.5, Active: true},
		{Name: "Text Editor", Type: "productivity", CPUUsage: 2.0, MemoryUsage: 150.0, NetworkUsage: 0.1, Active: true},
		{Name: "Email Client", Type: "productivity", CPUUsage: 1.5, MemoryUsage: 200.0, NetworkUsage: 0.5, Active: false},
		{Name: "Video Conference", Type: "streaming", CPUUsage: 25.0, MemoryUsage: 800.0, NetworkUsage: 5.0, Active: false},
		{Name: "File Sync", Type: "productivity", CPUUsage: 3.0, MemoryUsage: 100.0, NetworkUsage: 1.0, Active: true},
		{Name: "Music Player", Type: "entertainment", CPUUsage: 2.0, MemoryUsage: 80.0, NetworkUsage: 0.2, Active: false},
		{Name: "Development IDE", Type: "productivity", CPUUsage: 12.0, MemoryUsage: 950.0, NetworkUsage: 0.8, Active: false},
	}

	// Set start times and select active applications
	now := time.Now()
	for i := range apps {
		apps[i].StartTime = now.Add(-time.Duration(l.getRandomInt(30, 300)) * time.Minute)
		if l.getRandomFloat(0, 100) < 60 {
			apps[i].Active = true
		}
	}

	l.applications = apps[:l.getRandomInt(3, len(apps))]
}

func (l *Laptop) initializeNetworkActivity() {
	ssids := []string{"HomeNetwork", "HomeNetwork_5G", "OfficeWiFi", "PublicWiFi"}
	l.networkActivity = NetworkActivity{
		CurrentUpload:   0.5,
		CurrentDownload: 2.0,
		TotalUpload:     150.0,
		TotalDownload:   1200.0,
		ConnectedSSID:   ssids[l.getRandomInt(0, len(ssids))],
		SignalStrength:  l.getRandomFloat(-60, -30),
		ConnectionType:  "wifi",
		LastActivity:    time.Now(),
	}
}

func (l *Laptop) initializeSystemResources() {
	l.systemResources = SystemResources{
		CPUUsage:       15.0,
		MemoryUsage:    45.0,
		DiskUsage:      65.0,
		Temperature:    45.0,
		FanSpeed:       1200,
		NetworkLatency: 25.0,
	}
}

func (l *Laptop) Start(ctx context.Context) error {
	if err := l.BaseDevice.Start(ctx); err != nil {
		return err
	}

	go l.runBatteryManagement(ctx)
	go l.runApplicationManagement(ctx)
	go l.runNetworkActivity(ctx)
	go l.runSystemMonitoring(ctx)
	go l.runPowerManagement(ctx)

	// Laptop started successfully
	return nil
}

func (l *Laptop) runBatteryManagement(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			l.updateBattery()
		}
	}
}

func (l *Laptop) runApplicationManagement(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			l.updateApplications()
		}
	}
}

func (l *Laptop) runNetworkActivity(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			l.updateNetworkActivity()
		}
	}
}

func (l *Laptop) runSystemMonitoring(ctx context.Context) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			l.updateSystemResources()
		}
	}
}

func (l *Laptop) runPowerManagement(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			l.managePowerSettings()
		}
	}
}

func (l *Laptop) updateBattery() {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.isCharging {
		if l.batteryLevel < 100 {
			chargeRate := l.getRandomFloat(1.0, 3.0)
			l.batteryLevel += chargeRate
			if l.batteryLevel > 100 {
				l.batteryLevel = 100
			}
		}

		// Sometimes unplug when fully charged or randomly
		if l.batteryLevel >= 100 || l.getRandomFloat(0, 100) < 5 {
			l.isCharging = false
		}
	} else {
		// Battery discharge based on usage and power mode
		dischargeRate := l.calculateDischargeRate()
		l.batteryLevel -= dischargeRate

		if l.batteryLevel <= 0 {
			l.batteryLevel = 0
			// Battery depleted - set offline status
		} else if l.batteryLevel <= 10 {
			event := base.Event{
				EventType: "battery_critical",
				Severity:  "critical",
				Message:   fmt.Sprintf("Battery critically low: %.1f%%", l.batteryLevel),
				Extra: map[string]interface{}{
					"battery_level": l.batteryLevel,
					"is_charging":   l.isCharging,
					"power_mode":    l.powerMode,
				},
			}
			l.PublishEvent(event) // suppress unused variable warning
		} else if l.batteryLevel <= 20 {
			event := base.Event{
				EventType: "battery_low",
				Severity:  "warning",
				Message:   fmt.Sprintf("Battery low: %.1f%%", l.batteryLevel),
				Extra: map[string]interface{}{
					"battery_level": l.batteryLevel,
					"is_charging":   l.isCharging,
				},
			}
			l.PublishEvent(event) // suppress unused variable warning
		}

		// Sometimes plug in charger when battery is low
		if l.batteryLevel < 30 && l.getRandomFloat(0, 100) < 20 {
			l.isCharging = true
		}
	}
}

func (l *Laptop) calculateDischargeRate() float64 {
	baseRate := 0.5 // Base discharge rate per minute

	// Adjust for power mode
	switch l.powerMode {
	case "power_saver":
		baseRate *= 0.6
	case "high_performance":
		baseRate *= 1.8
	case "balanced":
		baseRate *= 1.0
	}

	// Adjust for screen brightness
	baseRate *= (float64(l.screenBrightness)/100.0)*0.3 + 0.7

	// Adjust for active applications
	cpuLoad := l.systemResources.CPUUsage / 100.0
	baseRate *= (1.0 + cpuLoad)

	return baseRate + l.getRandomFloat(-0.2, 0.2)
}

func (l *Laptop) updateApplications() {
	l.mu.Lock()
	defer l.mu.Unlock()

	totalCPU := 0.0
	totalMemory := 0.0
	totalNetwork := 0.0

	for i := range l.applications {
		app := &l.applications[i]

		if app.Active {
			// Simulate usage variations
			app.CPUUsage += l.getRandomFloat(-2, 2)
			if app.CPUUsage < 0 {
				app.CPUUsage = 0
			}
			if app.CPUUsage > 50 {
				app.CPUUsage = 50
			}

			app.MemoryUsage += l.getRandomFloat(-50, 50)
			if app.MemoryUsage < 50 {
				app.MemoryUsage = 50
			}

			app.NetworkUsage += l.getRandomFloat(-0.5, 0.5)
			if app.NetworkUsage < 0 {
				app.NetworkUsage = 0
			}

			totalCPU += app.CPUUsage
			totalMemory += app.MemoryUsage
			totalNetwork += app.NetworkUsage
		}

		// Sometimes applications become inactive or new ones start
		if l.getRandomFloat(0, 100) < 5 {
			app.Active = !app.Active
			if app.Active {
				app.StartTime = time.Now()
			}
		}
	}

	// Update system resources based on applications
	l.systemResources.CPUUsage = totalCPU
	l.systemResources.MemoryUsage = (totalMemory / 80.0) + 20.0 // Base system usage + apps
}

func (l *Laptop) updateNetworkActivity() {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Calculate network usage from applications
	totalAppNetwork := 0.0
	for _, app := range l.applications {
		if app.Active {
			totalAppNetwork += app.NetworkUsage
		}
	}

	l.networkActivity.CurrentUpload = totalAppNetwork*0.3 + l.getRandomFloat(0, 1)
	l.networkActivity.CurrentDownload = totalAppNetwork + l.getRandomFloat(0, 2)

	// Accumulate totals
	l.networkActivity.TotalUpload += l.networkActivity.CurrentUpload / 6.0 // 10-second intervals
	l.networkActivity.TotalDownload += l.networkActivity.CurrentDownload / 6.0

	// Update signal strength
	l.networkActivity.SignalStrength += l.getRandomFloat(-5, 5)
	if l.networkActivity.SignalStrength > -20 {
		l.networkActivity.SignalStrength = -20
	}
	if l.networkActivity.SignalStrength < -90 {
		l.networkActivity.SignalStrength = -90
	}

	// Update latency
	l.systemResources.NetworkLatency = 20.0 + l.getRandomFloat(-10, 30)
	if l.systemResources.NetworkLatency < 5 {
		l.systemResources.NetworkLatency = 5
	}

	l.networkActivity.LastActivity = time.Now()

	// Sometimes change WiFi network
	if l.getRandomFloat(0, 100) < 1 {
		ssids := []string{"HomeNetwork", "HomeNetwork_5G", "OfficeWiFi", "PublicWiFi"}
		l.networkActivity.ConnectedSSID = ssids[l.getRandomInt(0, len(ssids))]
	}
}

func (l *Laptop) updateSystemResources() {
	l.mu.Lock()
	defer l.mu.Unlock()

	// CPU temperature and fan speed correlation
	targetTemp := 40.0 + (l.systemResources.CPUUsage / 100.0 * 30.0)
	tempDiff := targetTemp - l.systemResources.Temperature

	l.systemResources.Temperature += tempDiff*0.1 + l.getRandomFloat(-2, 2)
	if l.systemResources.Temperature < 30 {
		l.systemResources.Temperature = 30
	}
	if l.systemResources.Temperature > 85 {
		l.systemResources.Temperature = 85
	}

	// Fan speed based on temperature
	if l.systemResources.Temperature > 60 {
		l.systemResources.FanSpeed = int(1000 + (l.systemResources.Temperature-60)*100)
	} else {
		l.systemResources.FanSpeed = int(800 + l.systemResources.Temperature*10)
	}

	// Disk usage gradually increases
	if l.getRandomFloat(0, 100) < 10 {
		l.systemResources.DiskUsage += l.getRandomFloat(0, 0.5)
		if l.systemResources.DiskUsage > 95 {
			l.systemResources.DiskUsage = 95
		}
	}

	// Thermal throttling warning
	if l.systemResources.Temperature > 75 {
		event := base.Event{
			EventType: "thermal_warning",
			Severity:  "warning",
			Message:   fmt.Sprintf("High CPU temperature: %.1f°C", l.systemResources.Temperature),
			Extra: map[string]interface{}{
				"temperature": l.systemResources.Temperature,
				"fan_speed":   l.systemResources.FanSpeed,
				"cpu_usage":   l.systemResources.CPUUsage,
			},
		}
		l.PublishEvent(event)
	}
}

func (l *Laptop) managePowerSettings() {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Automatically adjust power mode based on battery level and usage
	if l.batteryLevel < 20 && l.powerMode != "power_saver" {
		l.powerMode = "power_saver"
		l.screenBrightness = 30

		event := base.Event{
			EventType: "power_mode_change",
			Severity:  "info",
			Message:   "Switched to power saver mode due to low battery",
			Extra: map[string]interface{}{
				"power_mode":        l.powerMode,
				"battery_level":     l.batteryLevel,
				"screen_brightness": l.screenBrightness,
			},
		}
		l.PublishEvent(event)
	} else if l.isCharging && l.batteryLevel > 50 && l.powerMode == "power_saver" {
		l.powerMode = "balanced"
		l.screenBrightness = 75
	}

	// Adjust screen brightness based on usage patterns
	hour := time.Now().Hour()
	if hour >= 20 || hour <= 7 {
		// Night time - reduce brightness
		if l.screenBrightness > 40 {
			l.screenBrightness -= 5
		}
	}
}

func (l *Laptop) GenerateStatePayload() base.StatePayload {
	l.mu.RLock()
	defer l.mu.RUnlock()

	baseState := l.BaseDevice.GenerateStatePayload()

	baseState.Extra["battery"] = map[string]interface{}{
		"level":       l.batteryLevel,
		"is_charging": l.isCharging,
		"power_mode":  l.powerMode,
	}

	baseState.Extra["display"] = map[string]interface{}{
		"screen_brightness": l.screenBrightness,
	}

	baseState.Extra["applications"] = map[string]interface{}{
		"total_count":  len(l.applications),
		"active_count": l.getActiveAppCount(),
		"applications": l.applications,
	}

	baseState.Extra["network"] = l.networkActivity
	baseState.Extra["system_resources"] = l.systemResources

	return baseState
}

func (l *Laptop) getActiveAppCount() int {
	count := 0
	for _, app := range l.applications {
		if app.Active {
			count++
		}
	}
	return count
}

func (l *Laptop) GenerateTelemetryData() map[string]base.TelemetryPayload {
	l.mu.RLock()
	defer l.mu.RUnlock()

	telemetry := l.BaseDevice.GenerateTelemetryData()

	telemetry["battery_level"] = base.TelemetryPayload{
		"timestamp": time.Now(),
		"value":     l.batteryLevel,
		"unit":      "percent",
		"tags": map[string]string{
			"metric": "battery_level",
		},
	}

	telemetry["cpu_usage"] = base.TelemetryPayload{
		"timestamp": time.Now(),
		"value":     l.systemResources.CPUUsage,
		"unit":      "percent",
		"tags": map[string]string{
			"metric": "cpu_usage",
		},
	}

	telemetry["memory_usage"] = base.TelemetryPayload{
		"timestamp": time.Now(),
		"value":     l.systemResources.MemoryUsage,
		"unit":      "percent",
		"tags": map[string]string{
			"metric": "memory_usage",
		},
	}

	telemetry["cpu_temperature"] = base.TelemetryPayload{
		"timestamp": time.Now(),
		"value":     l.systemResources.Temperature,
		"unit":      "celsius",
		"tags": map[string]string{
			"metric": "cpu_temperature",
		},
	}

	telemetry["network_upload"] = base.TelemetryPayload{
		"timestamp": time.Now(),
		"value":     l.networkActivity.CurrentUpload,
		"unit":      "mbps",
		"tags": map[string]string{
			"metric": "network_upload",
		},
	}

	telemetry["network_download"] = base.TelemetryPayload{
		"timestamp": time.Now(),
		"value":     l.networkActivity.CurrentDownload,
		"unit":      "mbps",
		"tags": map[string]string{
			"metric": "network_download",
		},
	}

	telemetry["network_latency"] = base.TelemetryPayload{
		"timestamp": time.Now(),
		"value":     l.systemResources.NetworkLatency,
		"unit":      "ms",
		"tags": map[string]string{
			"metric": "network_latency",
		},
	}

	return telemetry
}

func (l *Laptop) HandleCommand(cmd base.Command) error {
	// Handling command

	switch cmd.Action {
	case "set_power_mode":
		return l.setPowerMode(cmd)
	case "set_brightness":
		return l.setBrightness(cmd)
	case "launch_application":
		return l.launchApplication(cmd)
	case "close_application":
		return l.closeApplication(cmd)
	case "get_system_info":
		return l.getSystemInfo(cmd)
	case "connect_wifi":
		return l.connectWiFi(cmd)
	default:
		return l.BaseDevice.HandleCommand(cmd)
	}
}

func (l *Laptop) setPowerMode(cmd base.Command) error {
	var powerCmd struct {
		PowerMode string `json:"power_mode"`
	}

	if err := json.Unmarshal([]byte(cmd.Payload), &powerCmd); err != nil {
		return fmt.Errorf("invalid power mode setting: %v", err)
	}

	validModes := []string{"power_saver", "balanced", "high_performance"}
	valid := false
	for _, mode := range validModes {
		if powerCmd.PowerMode == mode {
			valid = true
			break
		}
	}

	if !valid {
		return fmt.Errorf("invalid power mode: %s", powerCmd.PowerMode)
	}

	l.mu.Lock()
	l.powerMode = powerCmd.PowerMode
	l.mu.Unlock()

	response := base.CommandResponse{
		CommandID: cmd.ID,
		Status:    "success",
		Data:      fmt.Sprintf("Power mode set to %s", powerCmd.PowerMode),
	}

	return l.PublishCommandResponse(response)
}

func (l *Laptop) setBrightness(cmd base.Command) error {
	var brightnessCmd struct {
		Brightness int `json:"brightness"`
	}

	if err := json.Unmarshal([]byte(cmd.Payload), &brightnessCmd); err != nil {
		return fmt.Errorf("invalid brightness setting: %v", err)
	}

	if brightnessCmd.Brightness < 0 || brightnessCmd.Brightness > 100 {
		return fmt.Errorf("brightness must be between 0 and 100")
	}

	l.mu.Lock()
	l.screenBrightness = brightnessCmd.Brightness
	l.mu.Unlock()

	response := base.CommandResponse{
		CommandID: cmd.ID,
		Status:    "success",
		Data:      fmt.Sprintf("Screen brightness set to %d%%", brightnessCmd.Brightness),
	}

	return l.PublishCommandResponse(response)
}

func (l *Laptop) launchApplication(cmd base.Command) error {
	var appCmd struct {
		Name string `json:"name"`
		Type string `json:"type"`
	}

	if err := json.Unmarshal([]byte(cmd.Payload), &appCmd); err != nil {
		return fmt.Errorf("invalid application launch request: %v", err)
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	// Check if application is already running
	for i, app := range l.applications {
		if app.Name == appCmd.Name {
			l.applications[i].Active = true
			response := base.CommandResponse{
				CommandID: cmd.ID,
				Status:    "success",
				Data:      fmt.Sprintf("Application '%s' activated", appCmd.Name),
			}
			return l.PublishCommandResponse(response)
		}
	}

	// Launch new application
	newApp := RunningApplication{
		Name:         appCmd.Name,
		Type:         appCmd.Type,
		CPUUsage:     l.getRandomFloat(1, 10),
		MemoryUsage:  l.getRandomFloat(100, 500),
		NetworkUsage: l.getRandomFloat(0, 2),
		StartTime:    time.Now(),
		Active:       true,
	}

	l.applications = append(l.applications, newApp)

	response := base.CommandResponse{
		CommandID: cmd.ID,
		Status:    "success",
		Data:      fmt.Sprintf("Application '%s' launched", appCmd.Name),
	}

	return l.PublishCommandResponse(response)
}

func (l *Laptop) closeApplication(cmd base.Command) error {
	var appCmd struct {
		Name string `json:"name"`
	}

	if err := json.Unmarshal([]byte(cmd.Payload), &appCmd); err != nil {
		return fmt.Errorf("invalid application close request: %v", err)
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	for i, app := range l.applications {
		if app.Name == appCmd.Name && app.Active {
			l.applications[i].Active = false
			response := base.CommandResponse{
				CommandID: cmd.ID,
				Status:    "success",
				Data:      fmt.Sprintf("Application '%s' closed", appCmd.Name),
			}
			return l.PublishCommandResponse(response)
		}
	}

	return fmt.Errorf("application '%s' not found or not active", appCmd.Name)
}

func (l *Laptop) getSystemInfo(cmd base.Command) error {
	l.mu.RLock()
	systemInfo := map[string]interface{}{
		"battery":           map[string]interface{}{"level": l.batteryLevel, "is_charging": l.isCharging},
		"power_mode":        l.powerMode,
		"screen_brightness": l.screenBrightness,
		"system_resources":  l.systemResources,
		"network":           l.networkActivity,
		"active_apps":       l.getActiveAppCount(),
	}
	l.mu.RUnlock()

	data, _ := json.Marshal(systemInfo)
	response := base.CommandResponse{
		CommandID: cmd.ID,
		Status:    "success",
		Data:      string(data),
	}

	return l.PublishCommandResponse(response)
}

func (l *Laptop) connectWiFi(cmd base.Command) error {
	var wifiCmd struct {
		SSID     string `json:"ssid"`
		Password string `json:"password,omitempty"`
	}

	if err := json.Unmarshal([]byte(cmd.Payload), &wifiCmd); err != nil {
		return fmt.Errorf("invalid WiFi connection request: %v", err)
	}

	l.mu.Lock()
	l.networkActivity.ConnectedSSID = wifiCmd.SSID
	l.networkActivity.SignalStrength = l.getRandomFloat(-60, -30)
	l.networkActivity.ConnectionType = "wifi"
	l.mu.Unlock()

	response := base.CommandResponse{
		CommandID: cmd.ID,
		Status:    "success",
		Data:      fmt.Sprintf("Connected to WiFi network: %s", wifiCmd.SSID),
	}

	return l.PublishCommandResponse(response)
}

// Helper functions
func (l *Laptop) getRandomInt(min, max int) int {
	return min + rand.Intn(max-min+1)
}

func (l *Laptop) getRandomFloat(min, max float64) float64 {
	return min + rand.Float64()*(max-min)
}
