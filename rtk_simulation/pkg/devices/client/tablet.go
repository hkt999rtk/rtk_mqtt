package client

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"sync"
	"time"

	// "github.com/sirupsen/logrus"
	// "rtk_simulation/pkg/config"
	"rtk_simulation/pkg/devices/base"
)

type Tablet struct {
	*base.BaseDevice

	batteryLevel     float64
	isCharging       bool
	screenBrightness int
	orientation      string
	applications     []TabletApplication
	mediaPlayback    MediaPlayback
	networkActivity  NetworkActivity
	touchActivity    TouchActivity
	mu               sync.RWMutex
}

type TabletApplication struct {
	Name         string        `json:"name"`
	Type         string        `json:"type"`
	Active       bool          `json:"active"`
	Foreground   bool          `json:"foreground"`
	BatteryUsage float64       `json:"battery_usage"`
	DataUsage    float64       `json:"data_usage"`
	StartTime    time.Time     `json:"start_time"`
	UsageTime    time.Duration `json:"usage_time"`
}

type MediaPlayback struct {
	Playing      bool      `json:"playing"`
	MediaType    string    `json:"media_type"`
	Title        string    `json:"title"`
	Duration     int       `json:"duration"`
	Position     int       `json:"position"`
	Volume       int       `json:"volume"`
	Quality      string    `json:"quality"`
	Bandwidth    float64   `json:"bandwidth"`
	LastActivity time.Time `json:"last_activity"`
}

type TouchActivity struct {
	TouchesPerMinute int       `json:"touches_per_minute"`
	SwipesPerMinute  int       `json:"swipes_per_minute"`
	ZoomGestures     int       `json:"zoom_gestures"`
	LastTouch        time.Time `json:"last_touch"`
	ActiveRegions    []string  `json:"active_regions"`
}

func NewTablet(deviceConfig base.DeviceConfig, mqttConfig base.MQTTConfig) (*Tablet, error) {
	baseDevice := base.NewBaseDevice(deviceConfig)

	tablet := &Tablet{
		BaseDevice:       baseDevice,
		batteryLevel:     75.0,
		isCharging:       false,
		screenBrightness: 80,
		orientation:      "portrait",
	}

	tablet.initializeApplications()
	tablet.initializeMediaPlayback()
	tablet.initializeNetworkActivity()
	tablet.initializeTouchActivity()

	return tablet, nil
}

// Helper functions
func (t *Tablet) getRandomInt(min, max int) int {
	if min >= max {
		return min
	}
	return rand.Intn(max-min) + min
}

func (t *Tablet) getRandomFloat(min, max float64) float64 {
	return min + rand.Float64()*(max-min)
}

func (t *Tablet) getRandomBool() bool {
	return rand.Intn(2) == 1
}

func (t *Tablet) initializeApplications() {
	apps := []TabletApplication{
		{Name: "Web Browser", Type: "productivity", Active: true, Foreground: false, BatteryUsage: 8.0, DataUsage: 5.2},
		{Name: "Video Streaming", Type: "entertainment", Active: false, Foreground: false, BatteryUsage: 20.0, DataUsage: 25.0},
		{Name: "Social Media", Type: "social", Active: true, Foreground: true, BatteryUsage: 12.0, DataUsage: 8.0},
		{Name: "Photo Gallery", Type: "media", Active: false, Foreground: false, BatteryUsage: 5.0, DataUsage: 1.0},
		{Name: "E-Reader", Type: "productivity", Active: false, Foreground: false, BatteryUsage: 2.0, DataUsage: 0.5},
		{Name: "Games", Type: "entertainment", Active: false, Foreground: false, BatteryUsage: 15.0, DataUsage: 3.0},
		{Name: "Music Player", Type: "entertainment", Active: true, Foreground: false, BatteryUsage: 6.0, DataUsage: 2.0},
		{Name: "Email", Type: "productivity", Active: false, Foreground: false, BatteryUsage: 3.0, DataUsage: 1.5},
		{Name: "Weather", Type: "utility", Active: true, Foreground: false, BatteryUsage: 1.0, DataUsage: 0.3},
		{Name: "News", Type: "productivity", Active: false, Foreground: false, BatteryUsage: 7.0, DataUsage: 4.0},
	}

	now := time.Now()
	for i := range apps {
		apps[i].StartTime = now.Add(-time.Duration(t.getRandomInt(10, 180)) * time.Minute)
		apps[i].UsageTime = time.Duration(t.getRandomInt(5, 120)) * time.Minute
	}

	t.applications = apps[:t.getRandomInt(4, 8)]
}

func (t *Tablet) initializeMediaPlayback() {
	t.mediaPlayback = MediaPlayback{
		Playing:      false,
		MediaType:    "video",
		Title:        "",
		Duration:     0,
		Position:     0,
		Volume:       65,
		Quality:      "720p",
		Bandwidth:    0.0,
		LastActivity: time.Now(),
	}
}

func (t *Tablet) initializeNetworkActivity() {
	ssids := []string{"HomeNetwork", "HomeNetwork_5G", "PublicWiFi", "MobileHotspot"}
	t.networkActivity = NetworkActivity{
		CurrentUpload:   0.2,
		CurrentDownload: 1.5,
		TotalUpload:     85.0,
		TotalDownload:   650.0,
		ConnectedSSID:   ssids[t.getRandomInt(0, len(ssids))],
		SignalStrength:  t.getRandomFloat(-65, -35),
		ConnectionType:  "wifi",
		LastActivity:    time.Now(),
	}
}

func (t *Tablet) initializeTouchActivity() {
	t.touchActivity = TouchActivity{
		TouchesPerMinute: 15,
		SwipesPerMinute:  8,
		ZoomGestures:     2,
		LastTouch:        time.Now(),
		ActiveRegions:    []string{"center", "bottom"},
	}
}

func (t *Tablet) Start(ctx context.Context) error {
	if err := t.BaseDevice.Start(ctx); err != nil {
		return err
	}

	go t.runBatteryManagement(ctx)
	go t.runApplicationManagement(ctx)
	go t.runMediaPlayback(ctx)
	go t.runNetworkActivity(ctx)
	go t.runTouchActivity(ctx)
	go t.runOrientationChanges(ctx)

	// Tablet started successfully
	return nil
}

func (t *Tablet) runBatteryManagement(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			t.updateBattery()
		}
	}
}

func (t *Tablet) runApplicationManagement(ctx context.Context) {
	ticker := time.NewTicker(45 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			t.updateApplications()
		}
	}
}

func (t *Tablet) runMediaPlayback(ctx context.Context) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			t.updateMediaPlayback()
		}
	}
}

func (t *Tablet) runNetworkActivity(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			t.updateNetworkActivity()
		}
	}
}

func (t *Tablet) runTouchActivity(ctx context.Context) {
	ticker := time.NewTicker(20 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			t.updateTouchActivity()
		}
	}
}

func (t *Tablet) runOrientationChanges(ctx context.Context) {
	ticker := time.NewTicker(2 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			t.updateOrientation()
		}
	}
}

func (t *Tablet) updateBattery() {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.isCharging {
		if t.batteryLevel < 100 {
			chargeRate := t.getRandomFloat(1.5, 4.0)
			t.batteryLevel += chargeRate
			if t.batteryLevel > 100 {
				t.batteryLevel = 100
			}
		}

		if t.batteryLevel >= 100 || t.getRandomFloat(0, 100) < 8 {
			t.isCharging = false
		}
	} else {
		dischargeRate := t.calculateDischargeRate()
		t.batteryLevel -= dischargeRate

		if t.batteryLevel <= 0 {
			t.batteryLevel = 0
			// Update status: offline - Battery depleted
		} else if t.batteryLevel <= 5 {
			event := base.Event{
				EventType: "battery_critical",
				Severity:  "critical",
				Message:   fmt.Sprintf("Tablet battery critically low: %.1f%%", t.batteryLevel),
				Extra: map[string]interface{}{
					"battery_level": t.batteryLevel,
					"is_charging":   t.isCharging,
				},
			}
			t.PublishEvent(event)
		}

		if t.batteryLevel < 25 && t.getRandomFloat(0, 100) < 30 {
			t.isCharging = true
		}
	}
}

func (t *Tablet) calculateDischargeRate() float64 {
	baseRate := 0.8 // Base discharge rate per minute for tablet

	// Adjust for screen brightness
	baseRate *= (float64(t.screenBrightness)/100.0)*0.4 + 0.6

	// Adjust for media playback
	if t.mediaPlayback.Playing {
		if t.mediaPlayback.MediaType == "video" {
			baseRate *= 2.5
		} else {
			baseRate *= 1.3
		}
	}

	// Adjust for active applications
	activeAppCount := 0
	highBatteryUsage := 0.0
	for _, app := range t.applications {
		if app.Active {
			activeAppCount++
			if app.BatteryUsage > 10 {
				highBatteryUsage += app.BatteryUsage / 100.0
			}
		}
	}
	baseRate *= (1.0 + float64(activeAppCount)*0.1 + highBatteryUsage)

	// Network activity impact
	networkImpact := (t.networkActivity.CurrentUpload + t.networkActivity.CurrentDownload) / 20.0
	baseRate *= (1.0 + networkImpact)

	return baseRate + t.getRandomFloat(-0.3, 0.3)
}

func (t *Tablet) updateApplications() {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Update application states
	for i := range t.applications {
		app := &t.applications[i]

		if app.Active {
			app.UsageTime += 45 * time.Second

			// Vary data usage
			app.DataUsage += t.getRandomFloat(-0.5, 1.5)
			if app.DataUsage < 0 {
				app.DataUsage = 0
			}
		}

		// Random app state changes
		if t.getRandomFloat(0, 100) < 10 {
			app.Active = !app.Active
			if app.Active {
				app.StartTime = time.Now()
			}
		}

		// Update foreground app
		if t.getRandomFloat(0, 100) < 15 {
			app.Foreground = t.getRandomBool() && app.Active
		}
	}

	// Ensure only one foreground app
	foregroundCount := 0
	for _, app := range t.applications {
		if app.Foreground {
			foregroundCount++
		}
	}
	if foregroundCount > 1 {
		for i := range t.applications {
			if t.applications[i].Foreground {
				t.applications[i].Foreground = false
				break
			}
		}
	}
}

func (t *Tablet) updateMediaPlayback() {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.mediaPlayback.Playing {
		// Update playback position
		t.mediaPlayback.Position += 15

		// Check if media finished
		if t.mediaPlayback.Duration > 0 && t.mediaPlayback.Position >= t.mediaPlayback.Duration {
			t.mediaPlayback.Playing = false
			t.mediaPlayback.Position = 0
			t.mediaPlayback.Title = ""
			t.mediaPlayback.Duration = 0
		} else {
			// Sometimes pause
			if t.getRandomFloat(0, 100) < 5 {
				t.mediaPlayback.Playing = false
			}
		}

		// Update bandwidth usage
		if t.mediaPlayback.MediaType == "video" {
			switch t.mediaPlayback.Quality {
			case "480p":
				t.mediaPlayback.Bandwidth = 2.5 + t.getRandomFloat(-0.5, 0.5)
			case "720p":
				t.mediaPlayback.Bandwidth = 5.0 + t.getRandomFloat(-1, 1)
			case "1080p":
				t.mediaPlayback.Bandwidth = 8.0 + t.getRandomFloat(-1.5, 1.5)
			}
		} else {
			t.mediaPlayback.Bandwidth = 0.3 + t.getRandomFloat(-0.1, 0.2)
		}
	} else {
		// Sometimes start playing
		if t.getRandomFloat(0, 100) < 8 {
			t.startRandomMedia()
		}
		t.mediaPlayback.Bandwidth = 0
	}

	t.mediaPlayback.LastActivity = time.Now()
}

func (t *Tablet) startRandomMedia() {
	mediaTypes := []string{"video", "music", "podcast"}
	mediaType := mediaTypes[t.getRandomInt(0, len(mediaTypes))]

	t.mediaPlayback.Playing = true
	t.mediaPlayback.MediaType = mediaType
	t.mediaPlayback.Position = 0

	switch mediaType {
	case "video":
		t.mediaPlayback.Title = "Video Content"
		t.mediaPlayback.Duration = t.getRandomInt(600, 3600) // 10 min to 1 hour
		qualities := []string{"480p", "720p", "1080p"}
		t.mediaPlayback.Quality = qualities[t.getRandomInt(0, len(qualities))]
	case "music":
		t.mediaPlayback.Title = "Music Track"
		t.mediaPlayback.Duration = t.getRandomInt(180, 300) // 3-5 minutes
		t.mediaPlayback.Quality = "high"
	case "podcast":
		t.mediaPlayback.Title = "Podcast Episode"
		t.mediaPlayback.Duration = t.getRandomInt(1800, 3600) // 30 min to 1 hour
		t.mediaPlayback.Quality = "standard"
	}
}

func (t *Tablet) updateNetworkActivity() {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Calculate network usage from apps and media
	totalDataUsage := 0.0
	for _, app := range t.applications {
		if app.Active {
			totalDataUsage += app.DataUsage / 6.0 // 10-second intervals
		}
	}

	// Add media bandwidth
	totalDataUsage += t.mediaPlayback.Bandwidth

	t.networkActivity.CurrentUpload = totalDataUsage*0.2 + t.getRandomFloat(0, 0.5)
	t.networkActivity.CurrentDownload = totalDataUsage + t.getRandomFloat(0, 1)

	// Accumulate totals
	t.networkActivity.TotalUpload += t.networkActivity.CurrentUpload / 6.0
	t.networkActivity.TotalDownload += t.networkActivity.CurrentDownload / 6.0

	// Update signal strength
	t.networkActivity.SignalStrength += t.getRandomFloat(-3, 3)
	if t.networkActivity.SignalStrength > -20 {
		t.networkActivity.SignalStrength = -20
	}
	if t.networkActivity.SignalStrength < -90 {
		t.networkActivity.SignalStrength = -90
	}

	t.networkActivity.LastActivity = time.Now()
}

func (t *Tablet) updateTouchActivity() {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Adjust touch activity based on active apps and media playback
	baseActivity := 10

	for _, app := range t.applications {
		if app.Foreground {
			switch app.Type {
			case "entertainment":
				baseActivity += 15
			case "social":
				baseActivity += 20
			case "games":
				baseActivity += 35
			default:
				baseActivity += 8
			}
			break
		}
	}

	if t.mediaPlayback.Playing {
		baseActivity -= 5 // Less touching during media playback
	}

	t.touchActivity.TouchesPerMinute = baseActivity + t.getRandomInt(-5, 10)
	t.touchActivity.SwipesPerMinute = baseActivity/3 + t.getRandomInt(-2, 5)
	t.touchActivity.ZoomGestures = t.getRandomInt(0, 3)

	if t.touchActivity.TouchesPerMinute < 0 {
		t.touchActivity.TouchesPerMinute = 0
	}
	if t.touchActivity.SwipesPerMinute < 0 {
		t.touchActivity.SwipesPerMinute = 0
	}

	// Update active regions
	regions := []string{"top", "center", "bottom", "left", "right"}
	t.touchActivity.ActiveRegions = []string{}
	for i := 0; i < t.getRandomInt(1, 4); i++ {
		region := regions[t.getRandomInt(0, len(regions))]
		t.touchActivity.ActiveRegions = append(t.touchActivity.ActiveRegions, region)
	}

	t.touchActivity.LastTouch = time.Now()
}

func (t *Tablet) updateOrientation() {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Change orientation occasionally
	if t.getRandomFloat(0, 100) < 15 {
		orientations := []string{"portrait", "landscape", "portrait_upside_down", "landscape_left"}
		newOrientation := orientations[t.getRandomInt(0, len(orientations))]

		if newOrientation != t.orientation {
			oldOrientation := t.orientation
			t.orientation = newOrientation

			event := base.Event{
				EventType: "orientation_change",
				Severity:  "info",
				Message:   fmt.Sprintf("Tablet orientation changed from %s to %s", oldOrientation, newOrientation),
				Extra: map[string]interface{}{
					"old_orientation": oldOrientation,
					"new_orientation": newOrientation,
				},
			}
			t.PublishEvent(event)
		}
	}
}

func (t *Tablet) GenerateStatePayload() base.StatePayload {
	t.mu.RLock()
	defer t.mu.RUnlock()

	baseState := t.BaseDevice.GenerateStatePayload()

	if baseState.Extra == nil {
		baseState.Extra = make(map[string]interface{})
	}

	baseState.Extra["battery"] = map[string]interface{}{
		"level":       t.batteryLevel,
		"is_charging": t.isCharging,
	}

	baseState.Extra["display"] = map[string]interface{}{
		"screen_brightness": t.screenBrightness,
		"orientation":       t.orientation,
	}

	baseState.Extra["applications"] = map[string]interface{}{
		"total_count":    len(t.applications),
		"active_count":   t.getActiveAppCount(),
		"foreground_app": t.getForegroundApp(),
		"applications":   t.applications,
	}

	baseState.Extra["media_playback"] = t.mediaPlayback
	baseState.Extra["network"] = t.networkActivity
	baseState.Extra["touch_activity"] = t.touchActivity

	return baseState
}

func (t *Tablet) getActiveAppCount() int {
	count := 0
	for _, app := range t.applications {
		if app.Active {
			count++
		}
	}
	return count
}

func (t *Tablet) getForegroundApp() string {
	for _, app := range t.applications {
		if app.Foreground {
			return app.Name
		}
	}
	return "None"
}

func (t *Tablet) GenerateTelemetryData() map[string]base.TelemetryPayload {
	t.mu.RLock()
	defer t.mu.RUnlock()

	telemetry := t.BaseDevice.GenerateTelemetryData()

	telemetry["battery_level"] = base.TelemetryPayload{
		"timestamp": time.Now(),
		"value":     t.batteryLevel,
		"unit":      "percent",
		"tags": map[string]string{
			"metric": "battery_level",
		},
	}

	telemetry["screen_brightness"] = base.TelemetryPayload{
		"timestamp": time.Now(),
		"value":     float64(t.screenBrightness),
		"unit":      "percent",
		"tags": map[string]string{
			"metric": "screen_brightness",
		},
	}

	telemetry["active_applications"] = base.TelemetryPayload{
		"timestamp": time.Now(),
		"value":     float64(t.getActiveAppCount()),
		"unit":      "count",
		"tags": map[string]string{
			"metric": "active_applications",
		},
	}

	telemetry["touches_per_minute"] = base.TelemetryPayload{
		"timestamp": time.Now(),
		"value":     float64(t.touchActivity.TouchesPerMinute),
		"unit":      "count",
		"tags": map[string]string{
			"metric": "touches_per_minute",
		},
	}

	telemetry["network_download"] = base.TelemetryPayload{
		"timestamp": time.Now(),
		"value":     t.networkActivity.CurrentDownload,
		"unit":      "mbps",
		"tags": map[string]string{
			"metric": "network_download",
		},
	}

	telemetry["media_bandwidth"] = base.TelemetryPayload{
		"timestamp": time.Now(),
		"value":     t.mediaPlayback.Bandwidth,
		"unit":      "mbps",
		"tags": map[string]string{
			"metric": "media_bandwidth",
		},
	}

	return telemetry
}

func (t *Tablet) HandleCommand(cmd base.Command) error {
	// Handling command: cmd.Action

	switch cmd.Action {
	case "set_brightness":
		return t.setBrightness(cmd)
	case "launch_application":
		return t.launchApplication(cmd)
	case "close_application":
		return t.closeApplication(cmd)
	case "start_media":
		return t.startMediaCommand(cmd)
	case "stop_media":
		return t.stopMedia(cmd)
	case "set_volume":
		return t.setVolume(cmd)
	case "rotate_screen":
		return t.rotateScreen(cmd)
	case "connect_wifi":
		return t.connectWiFi(cmd)
	default:
		return t.BaseDevice.HandleCommand(cmd)
	}
}

func (t *Tablet) setBrightness(cmd base.Command) error {
	var brightnessCmd struct {
		Brightness int `json:"brightness"`
	}

	if err := json.Unmarshal([]byte(cmd.Payload), &brightnessCmd); err != nil {
		return fmt.Errorf("invalid brightness setting: %v", err)
	}

	if brightnessCmd.Brightness < 0 || brightnessCmd.Brightness > 100 {
		return fmt.Errorf("brightness must be between 0 and 100")
	}

	t.mu.Lock()
	t.screenBrightness = brightnessCmd.Brightness
	t.mu.Unlock()

	response := base.CommandResponse{
		CommandID: cmd.ID,
		Status:    "success",
		Data:      fmt.Sprintf("Screen brightness set to %d%%", brightnessCmd.Brightness),
	}

	return t.PublishCommandResponse(response)
}

func (t *Tablet) launchApplication(cmd base.Command) error {
	var appCmd struct {
		Name string `json:"name"`
		Type string `json:"type"`
	}

	if err := json.Unmarshal([]byte(cmd.Payload), &appCmd); err != nil {
		return fmt.Errorf("invalid application launch request: %v", err)
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	// Check if application exists and activate it
	for i, app := range t.applications {
		if app.Name == appCmd.Name {
			t.applications[i].Active = true
			t.applications[i].Foreground = true
			t.applications[i].StartTime = time.Now()

			// Remove foreground from other apps
			for j := range t.applications {
				if j != i {
					t.applications[j].Foreground = false
				}
			}

			response := base.CommandResponse{
				CommandID: cmd.ID,
				Status:    "success",
				Data:      fmt.Sprintf("Application '%s' launched and brought to foreground", appCmd.Name),
			}
			return t.PublishCommandResponse(response)
		}
	}

	// Create new application
	newApp := TabletApplication{
		Name:         appCmd.Name,
		Type:         appCmd.Type,
		Active:       true,
		Foreground:   true,
		BatteryUsage: t.getRandomFloat(2, 15),
		DataUsage:    t.getRandomFloat(0.5, 5),
		StartTime:    time.Now(),
		UsageTime:    0,
	}

	// Remove foreground from other apps
	for i := range t.applications {
		t.applications[i].Foreground = false
	}

	t.applications = append(t.applications, newApp)

	response := base.CommandResponse{
		CommandID: cmd.ID,
		Status:    "success",
		Data:      fmt.Sprintf("Application '%s' launched", appCmd.Name),
	}

	return t.PublishCommandResponse(response)
}

func (t *Tablet) closeApplication(cmd base.Command) error {
	var appCmd struct {
		Name string `json:"name"`
	}

	if err := json.Unmarshal([]byte(cmd.Payload), &appCmd); err != nil {
		return fmt.Errorf("invalid application close request: %v", err)
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	for i, app := range t.applications {
		if app.Name == appCmd.Name && app.Active {
			t.applications[i].Active = false
			t.applications[i].Foreground = false
			response := base.CommandResponse{
				CommandID: cmd.ID,
				Status:    "success",
				Data:      fmt.Sprintf("Application '%s' closed", appCmd.Name),
			}
			return t.PublishCommandResponse(response)
		}
	}

	return fmt.Errorf("application '%s' not found or not active", appCmd.Name)
}

func (t *Tablet) startMediaCommand(cmd base.Command) error {
	var mediaCmd struct {
		MediaType string `json:"media_type"`
		Title     string `json:"title"`
		Duration  int    `json:"duration"`
		Quality   string `json:"quality"`
	}

	if err := json.Unmarshal([]byte(cmd.Payload), &mediaCmd); err != nil {
		return fmt.Errorf("invalid media start request: %v", err)
	}

	t.mu.Lock()
	t.mediaPlayback.Playing = true
	t.mediaPlayback.MediaType = mediaCmd.MediaType
	t.mediaPlayback.Title = mediaCmd.Title
	t.mediaPlayback.Duration = mediaCmd.Duration
	t.mediaPlayback.Position = 0
	if mediaCmd.Quality != "" {
		t.mediaPlayback.Quality = mediaCmd.Quality
	}
	t.mu.Unlock()

	response := base.CommandResponse{
		CommandID: cmd.ID,
		Status:    "success",
		Data:      fmt.Sprintf("Started playing %s: %s", mediaCmd.MediaType, mediaCmd.Title),
	}

	return t.PublishCommandResponse(response)
}

func (t *Tablet) stopMedia(cmd base.Command) error {
	t.mu.Lock()
	t.mediaPlayback.Playing = false
	t.mediaPlayback.Bandwidth = 0
	t.mu.Unlock()

	response := base.CommandResponse{
		CommandID: cmd.ID,
		Status:    "success",
		Data:      "Media playback stopped",
	}

	return t.PublishCommandResponse(response)
}

func (t *Tablet) setVolume(cmd base.Command) error {
	var volumeCmd struct {
		Volume int `json:"volume"`
	}

	if err := json.Unmarshal([]byte(cmd.Payload), &volumeCmd); err != nil {
		return fmt.Errorf("invalid volume setting: %v", err)
	}

	if volumeCmd.Volume < 0 || volumeCmd.Volume > 100 {
		return fmt.Errorf("volume must be between 0 and 100")
	}

	t.mu.Lock()
	t.mediaPlayback.Volume = volumeCmd.Volume
	t.mu.Unlock()

	response := base.CommandResponse{
		CommandID: cmd.ID,
		Status:    "success",
		Data:      fmt.Sprintf("Volume set to %d%%", volumeCmd.Volume),
	}

	return t.PublishCommandResponse(response)
}

func (t *Tablet) rotateScreen(cmd base.Command) error {
	var rotateCmd struct {
		Orientation string `json:"orientation"`
	}

	if err := json.Unmarshal([]byte(cmd.Payload), &rotateCmd); err != nil {
		return fmt.Errorf("invalid rotation request: %v", err)
	}

	validOrientations := []string{"portrait", "landscape", "portrait_upside_down", "landscape_left"}
	valid := false
	for _, orientation := range validOrientations {
		if rotateCmd.Orientation == orientation {
			valid = true
			break
		}
	}

	if !valid {
		return fmt.Errorf("invalid orientation: %s", rotateCmd.Orientation)
	}

	t.mu.Lock()
	t.orientation = rotateCmd.Orientation
	t.mu.Unlock()

	response := base.CommandResponse{
		CommandID: cmd.ID,
		Status:    "success",
		Data:      fmt.Sprintf("Screen rotated to %s", rotateCmd.Orientation),
	}

	return t.PublishCommandResponse(response)
}

func (t *Tablet) connectWiFi(cmd base.Command) error {
	var wifiCmd struct {
		SSID     string `json:"ssid"`
		Password string `json:"password,omitempty"`
	}

	if err := json.Unmarshal([]byte(cmd.Payload), &wifiCmd); err != nil {
		return fmt.Errorf("invalid WiFi connection request: %v", err)
	}

	t.mu.Lock()
	t.networkActivity.ConnectedSSID = wifiCmd.SSID
	t.networkActivity.SignalStrength = t.getRandomFloat(-65, -30)
	t.networkActivity.ConnectionType = "wifi"
	t.mu.Unlock()

	response := base.CommandResponse{
		CommandID: cmd.ID,
		Status:    "success",
		Data:      fmt.Sprintf("Connected to WiFi network: %s", wifiCmd.SSID),
	}

	return t.PublishCommandResponse(response)
}
