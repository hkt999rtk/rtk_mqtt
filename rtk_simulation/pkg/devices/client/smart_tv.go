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

type SmartTV struct {
	*base.BaseDevice

	power           bool
	volume          int
	currentChannel  int
	currentInput    string
	screenSize      int
	resolution      string
	mediaPlayback   TVMediaPlayback
	networkActivity NetworkActivity
	apps            []TVApplication
	settings        TVSettings
	mu              sync.RWMutex
}

type TVMediaPlayback struct {
	Playing          bool      `json:"playing"`
	MediaType        string    `json:"media_type"`
	Title            string    `json:"title"`
	Duration         int       `json:"duration"`
	Position         int       `json:"position"`
	Quality          string    `json:"quality"`
	Bandwidth        float64   `json:"bandwidth"`
	AudioLanguage    string    `json:"audio_language"`
	SubtitleLanguage string    `json:"subtitle_language"`
	LastActivity     time.Time `json:"last_activity"`
}

type TVApplication struct {
	Name      string        `json:"name"`
	Type      string        `json:"type"`
	Active    bool          `json:"active"`
	DataUsage float64       `json:"data_usage"`
	StartTime time.Time     `json:"start_time"`
	UsageTime time.Duration `json:"usage_time"`
}

type TVSettings struct {
	Brightness      int    `json:"brightness"`
	Contrast        int    `json:"contrast"`
	ColorSaturation int    `json:"color_saturation"`
	PictureMode     string `json:"picture_mode"`
	AudioMode       string `json:"audio_mode"`
	EcoMode         bool   `json:"eco_mode"`
	HDREnabled      bool   `json:"hdr_enabled"`
	MotionSmoothing bool   `json:"motion_smoothing"`
}

func NewSmartTV(deviceConfig base.DeviceConfig, mqttConfig base.MQTTConfig) (*SmartTV, error) {
	baseDevice := base.NewBaseDevice(deviceConfig)

	tv := &SmartTV{
		BaseDevice:     baseDevice,
		power:          false,
		volume:         25,
		currentChannel: 1,
		currentInput:   "hdmi1",
		screenSize:     55,
		resolution:     "4K",
	}

	tv.initializeMediaPlayback()
	tv.initializeNetworkActivity()
	tv.initializeApplications()
	tv.initializeSettings()

	return tv, nil
}

// Helper functions
func (tv *SmartTV) getRandomInt(min, max int) int {
	if min >= max {
		return min
	}
	return rand.Intn(max-min) + min
}

func (tv *SmartTV) getRandomFloat(min, max float64) float64 {
	return min + rand.Float64()*(max-min)
}

func (tv *SmartTV) getRandomBool() bool {
	return rand.Intn(2) == 1
}

func (tv *SmartTV) initializeMediaPlayback() {
	tv.mediaPlayback = TVMediaPlayback{
		Playing:          false,
		MediaType:        "broadcast",
		Title:            "",
		Duration:         0,
		Position:         0,
		Quality:          "1080p",
		Bandwidth:        0.0,
		AudioLanguage:    "English",
		SubtitleLanguage: "Off",
		LastActivity:     time.Now(),
	}
}

func (tv *SmartTV) initializeNetworkActivity() {
	ssids := []string{"HomeNetwork", "HomeNetwork_5G", "LivingRoomWiFi"}
	tv.networkActivity = NetworkActivity{
		CurrentUpload:   0.1,
		CurrentDownload: 5.0,
		TotalUpload:     25.0,
		TotalDownload:   2500.0,
		ConnectedSSID:   ssids[tv.getRandomInt(0, len(ssids))],
		SignalStrength:  tv.getRandomFloat(-55, -25),
		ConnectionType:  "wifi",
		LastActivity:    time.Now(),
	}
}

func (tv *SmartTV) initializeApplications() {
	apps := []TVApplication{
		{Name: "Netflix", Type: "streaming", Active: false, DataUsage: 25.0},
		{Name: "YouTube", Type: "streaming", Active: false, DataUsage: 20.0},
		{Name: "Amazon Prime", Type: "streaming", Active: false, DataUsage: 22.0},
		{Name: "Disney+", Type: "streaming", Active: false, DataUsage: 28.0},
		{Name: "Spotify", Type: "music", Active: false, DataUsage: 3.0},
		{Name: "Plex", Type: "media", Active: false, DataUsage: 30.0},
		{Name: "Weather", Type: "utility", Active: false, DataUsage: 0.5},
		{Name: "News", Type: "information", Active: false, DataUsage: 2.0},
	}

	now := time.Now()
	for i := range apps {
		apps[i].StartTime = now
		apps[i].UsageTime = 0
	}

	tv.apps = apps
}

func (tv *SmartTV) initializeSettings() {
	tv.settings = TVSettings{
		Brightness:      80,
		Contrast:        75,
		ColorSaturation: 70,
		PictureMode:     "Standard",
		AudioMode:       "Auto",
		EcoMode:         false,
		HDREnabled:      true,
		MotionSmoothing: false,
	}
}

func (tv *SmartTV) Start(ctx context.Context) error {
	if err := tv.BaseDevice.Start(ctx); err != nil {
		return err
	}

	go tv.runPowerManagement(ctx)
	go tv.runMediaPlayback(ctx)
	go tv.runNetworkActivity(ctx)
	go tv.runApplicationManagement(ctx)
	go tv.runChannelSwitching(ctx)

	// Smart TV started successfully
	return nil
}

func (tv *SmartTV) runPowerManagement(ctx context.Context) {
	ticker := time.NewTicker(2 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			tv.managePowerState()
		}
	}
}

func (tv *SmartTV) runMediaPlayback(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			tv.updateMediaPlayback()
		}
	}
}

func (tv *SmartTV) runNetworkActivity(ctx context.Context) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			tv.updateNetworkActivity()
		}
	}
}

func (tv *SmartTV) runApplicationManagement(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			tv.updateApplications()
		}
	}
}

func (tv *SmartTV) runChannelSwitching(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			tv.simulateChannelChange()
		}
	}
}

func (tv *SmartTV) managePowerState() {
	tv.mu.Lock()
	defer tv.mu.Unlock()

	// Simulate power state changes based on time of day
	hour := time.Now().Hour()

	if !tv.power {
		// Turn on during typical viewing hours
		if (hour >= 18 && hour <= 23) || (hour >= 7 && hour <= 9) {
			if tv.getRandomFloat(0, 100) < 15 {
				tv.power = true
				// Smart TV powered on

				event := base.Event{
					EventType: "power_on",
					Severity:  "info",
					Message:   "Smart TV turned on",

					Extra: map[string]interface{}{
						"power_state": tv.power,
						"hour":        hour,
					},
				}
				tv.PublishEvent(event)
			}
		}
	} else {
		// Turn off during late night/early morning hours
		if hour >= 1 && hour <= 6 {
			if tv.getRandomFloat(0, 100) < 25 {
				tv.power = false
				tv.mediaPlayback.Playing = false
				// Smart TV powered off

				event := base.Event{
					EventType: "power_off",
					Severity:  "info",
					Message:   "Smart TV turned off",

					Extra: map[string]interface{}{
						"power_state": tv.power,
						"hour":        hour,
					},
				}
				tv.PublishEvent(event)
			}
		}
	}
}

func (tv *SmartTV) updateMediaPlayback() {
	if !tv.power {
		return
	}

	tv.mu.Lock()
	defer tv.mu.Unlock()

	if tv.mediaPlayback.Playing {
		// Update playback position
		tv.mediaPlayback.Position += 30

		// Check if media finished
		if tv.mediaPlayback.Duration > 0 && tv.mediaPlayback.Position >= tv.mediaPlayback.Duration {
			tv.mediaPlayback.Playing = false
			tv.mediaPlayback.Position = 0
			tv.mediaPlayback.Title = ""
			tv.mediaPlayback.Duration = 0
		} else {
			// Sometimes pause or stop
			if tv.getRandomFloat(0, 100) < 3 {
				tv.mediaPlayback.Playing = false
			}
		}

		// Update bandwidth usage based on quality
		switch tv.mediaPlayback.Quality {
		case "480p":
			tv.mediaPlayback.Bandwidth = 3.0 + tv.getRandomFloat(-0.5, 1.0)
		case "720p":
			tv.mediaPlayback.Bandwidth = 5.0 + tv.getRandomFloat(-1.0, 2.0)
		case "1080p":
			tv.mediaPlayback.Bandwidth = 8.0 + tv.getRandomFloat(-2.0, 3.0)
		case "4K":
			tv.mediaPlayback.Bandwidth = 25.0 + tv.getRandomFloat(-5.0, 10.0)
		default:
			tv.mediaPlayback.Bandwidth = 5.0
		}
	} else {
		// Sometimes start playing
		if tv.getRandomFloat(0, 100) < 12 {
			tv.startRandomContent()
		}
		tv.mediaPlayback.Bandwidth = 0
	}

	tv.mediaPlayback.LastActivity = time.Now()
}

func (tv *SmartTV) startRandomContent() {
	contentTypes := []string{"movie", "tv_show", "documentary", "news", "sports"}
	contentType := contentTypes[tv.getRandomInt(0, len(contentTypes))]

	tv.mediaPlayback.Playing = true
	tv.mediaPlayback.MediaType = contentType
	tv.mediaPlayback.Position = 0

	switch contentType {
	case "movie":
		tv.mediaPlayback.Title = "Feature Film"
		tv.mediaPlayback.Duration = tv.getRandomInt(5400, 9000) // 90-150 minutes
	case "tv_show":
		tv.mediaPlayback.Title = "TV Episode"
		tv.mediaPlayback.Duration = tv.getRandomInt(1800, 3600) // 30-60 minutes
	case "documentary":
		tv.mediaPlayback.Title = "Documentary"
		tv.mediaPlayback.Duration = tv.getRandomInt(3000, 5400) // 50-90 minutes
	case "news":
		tv.mediaPlayback.Title = "News Broadcast"
		tv.mediaPlayback.Duration = tv.getRandomInt(1800, 3600) // 30-60 minutes
	case "sports":
		tv.mediaPlayback.Title = "Sports Event"
		tv.mediaPlayback.Duration = tv.getRandomInt(7200, 10800) // 2-3 hours
	}

	// Set quality based on content type and network conditions
	qualities := []string{"720p", "1080p", "4K"}
	if tv.networkActivity.SignalStrength > -40 {
		tv.mediaPlayback.Quality = qualities[2] // 4K
	} else if tv.networkActivity.SignalStrength > -60 {
		tv.mediaPlayback.Quality = qualities[1] // 1080p
	} else {
		tv.mediaPlayback.Quality = qualities[0] // 720p
	}
}

func (tv *SmartTV) updateNetworkActivity() {
	if !tv.power {
		tv.mu.Lock()
		tv.networkActivity.CurrentUpload = 0
		tv.networkActivity.CurrentDownload = 0
		tv.mu.Unlock()
		return
	}

	tv.mu.Lock()
	defer tv.mu.Unlock()

	// Calculate network usage from active apps and media
	totalDataUsage := 0.0
	for _, app := range tv.apps {
		if app.Active {
			totalDataUsage += app.DataUsage / 4.0 // 15-second intervals
		}
	}

	// Add media bandwidth
	totalDataUsage += tv.mediaPlayback.Bandwidth

	tv.networkActivity.CurrentUpload = totalDataUsage*0.05 + tv.getRandomFloat(0, 0.2)
	tv.networkActivity.CurrentDownload = totalDataUsage + tv.getRandomFloat(0, 2)

	// Accumulate totals
	tv.networkActivity.TotalUpload += tv.networkActivity.CurrentUpload / 4.0
	tv.networkActivity.TotalDownload += tv.networkActivity.CurrentDownload / 4.0

	// Update signal strength
	tv.networkActivity.SignalStrength += tv.getRandomFloat(-2, 2)
	if tv.networkActivity.SignalStrength > -20 {
		tv.networkActivity.SignalStrength = -20
	}
	if tv.networkActivity.SignalStrength < -80 {
		tv.networkActivity.SignalStrength = -80
	}

	tv.networkActivity.LastActivity = time.Now()

	// Check for network issues
	if tv.networkActivity.SignalStrength < -70 {
		if tv.getRandomFloat(0, 100) < 10 {
			event := base.Event{
				EventType: "network_quality_poor",
				Severity:  "warning",
				Message:   "Poor network signal affecting streaming quality",
				Extra: map[string]interface{}{
					"signal_strength": tv.networkActivity.SignalStrength,
					"current_quality": tv.mediaPlayback.Quality,
				},
			}
			tv.PublishEvent(event)
		}
	}
}

func (tv *SmartTV) updateApplications() {
	if !tv.power {
		return
	}

	tv.mu.Lock()
	defer tv.mu.Unlock()

	// Update running applications
	for i := range tv.apps {
		app := &tv.apps[i]

		if app.Active {
			app.UsageTime += 1 * time.Minute

			// Vary data usage
			app.DataUsage += tv.getRandomFloat(-1.0, 2.0)
			if app.DataUsage < 0 {
				app.DataUsage = 0
			}
		}

		// Random app state changes
		if tv.getRandomFloat(0, 100) < 8 {
			app.Active = !app.Active
			if app.Active {
				app.StartTime = time.Now()
				// TV app started: app.Name
			} else {
				// TV app stopped: app.Name
			}
		}
	}
}

func (tv *SmartTV) simulateChannelChange() {
	if !tv.power {
		return
	}

	tv.mu.Lock()
	defer tv.mu.Unlock()

	// Sometimes change channel or input
	if tv.getRandomFloat(0, 100) < 20 {
		if tv.getRandomBool() {
			// Change channel
			oldChannel := tv.currentChannel
			tv.currentChannel = tv.getRandomInt(1, 100)

			event := base.Event{
				EventType: "channel_change",
				Severity:  "info",
				Message:   fmt.Sprintf("Channel changed from %d to %d", oldChannel, tv.currentChannel),
				Extra: map[string]interface{}{
					"old_channel": oldChannel,
					"new_channel": tv.currentChannel,
				},
			}
			tv.PublishEvent(event)
		} else {
			// Change input
			inputs := []string{"hdmi1", "hdmi2", "hdmi3", "usb", "cast", "antenna"}
			oldInput := tv.currentInput
			tv.currentInput = inputs[tv.getRandomInt(0, len(inputs))]

			if oldInput != tv.currentInput {
				event := base.Event{
					EventType: "input_change",
					Severity:  "info",
					Message:   fmt.Sprintf("Input changed from %s to %s", oldInput, tv.currentInput),

					Extra: map[string]interface{}{
						"old_input": oldInput,
						"new_input": tv.currentInput,
					},
				}
				tv.PublishEvent(event)
			}
		}
	}
}

func (tv *SmartTV) GenerateStatePayload() base.StatePayload {
	tv.mu.RLock()
	defer tv.mu.RUnlock()

	baseState := tv.BaseDevice.GenerateStatePayload()

	if baseState.Extra == nil {
		baseState.Extra = make(map[string]interface{})
	}

	baseState.Extra["power"] = map[string]interface{}{
		"power_state":     tv.power,
		"volume":          tv.volume,
		"current_channel": tv.currentChannel,
		"current_input":   tv.currentInput,
	}

	baseState.Extra["display"] = map[string]interface{}{
		"screen_size": tv.screenSize,
		"resolution":  tv.resolution,
		"settings":    tv.settings,
	}

	baseState.Extra["applications"] = map[string]interface{}{
		"total_count":  len(tv.apps),
		"active_count": tv.getActiveAppCount(),
		"applications": tv.apps,
	}

	baseState.Extra["media_playback"] = tv.mediaPlayback
	baseState.Extra["network"] = tv.networkActivity

	return baseState
}

func (tv *SmartTV) getActiveAppCount() int {
	count := 0
	for _, app := range tv.apps {
		if app.Active {
			count++
		}
	}
	return count
}

func (tv *SmartTV) GenerateTelemetryData() map[string]base.TelemetryPayload {
	tv.mu.RLock()
	defer tv.mu.RUnlock()

	telemetry := tv.BaseDevice.GenerateTelemetryData()

	telemetry["power_state"] = base.TelemetryPayload{
		"timestamp": time.Now(),
		"value":     map[bool]float64{true: 1, false: 0}[tv.power],
		"unit":      "boolean",
		"tags": map[string]string{
			"metric": "power_state",
		},
	}

	telemetry["volume"] = base.TelemetryPayload{
		"timestamp": time.Now(),
		"value":     float64(tv.volume),
		"unit":      "percent",
		"tags": map[string]string{
			"metric": "volume",
		},
	}

	telemetry["active_applications"] = base.TelemetryPayload{
		"timestamp": time.Now(),
		"value":     float64(tv.getActiveAppCount()),
		"unit":      "count",
		"tags": map[string]string{
			"metric": "active_applications",
		},
	}

	telemetry["media_bandwidth"] = base.TelemetryPayload{
		"timestamp": time.Now(),
		"value":     tv.mediaPlayback.Bandwidth,
		"unit":      "mbps",
		"tags": map[string]string{
			"metric": "media_bandwidth",
		},
	}

	telemetry["network_download"] = base.TelemetryPayload{
		"timestamp": time.Now(),
		"value":     tv.networkActivity.CurrentDownload,
		"unit":      "mbps",
		"tags": map[string]string{
			"metric": "network_download",
		},
	}

	telemetry["signal_strength"] = base.TelemetryPayload{
		"timestamp": time.Now(),
		"value":     tv.networkActivity.SignalStrength,
		"unit":      "dbm",
		"tags": map[string]string{
			"metric": "signal_strength",
		},
	}

	return telemetry
}

func (tv *SmartTV) HandleCommand(cmd base.Command) error {
	// Handling command: cmd.Action

	switch cmd.Action {
	case "power_on":
		return tv.powerOn(cmd)
	case "power_off":
		return tv.powerOff(cmd)
	case "set_volume":
		return tv.setVolume(cmd)
	case "change_channel":
		return tv.changeChannel(cmd)
	case "change_input":
		return tv.changeInput(cmd)
	case "launch_app":
		return tv.launchApp(cmd)
	case "close_app":
		return tv.closeApp(cmd)
	case "start_media":
		return tv.startMedia(cmd)
	case "stop_media":
		return tv.stopMedia(cmd)
	case "adjust_settings":
		return tv.adjustSettings(cmd)
	default:
		return tv.BaseDevice.HandleCommand(cmd)
	}
}

func (tv *SmartTV) powerOn(cmd base.Command) error {
	tv.mu.Lock()
	tv.power = true
	tv.mu.Unlock()

	response := base.CommandResponse{
		CommandID: cmd.ID,
		Status:    "success",
		Data:      "Smart TV powered on",
	}

	return tv.PublishCommandResponse(response)
}

func (tv *SmartTV) powerOff(cmd base.Command) error {
	tv.mu.Lock()
	tv.power = false
	tv.mediaPlayback.Playing = false
	for i := range tv.apps {
		tv.apps[i].Active = false
	}
	tv.mu.Unlock()

	response := base.CommandResponse{
		CommandID: cmd.ID,
		Status:    "success",
		Data:      "Smart TV powered off",
	}

	return tv.PublishCommandResponse(response)
}

func (tv *SmartTV) setVolume(cmd base.Command) error {
	var volumeCmd struct {
		Volume int `json:"volume"`
	}

	if err := json.Unmarshal([]byte(cmd.Payload), &volumeCmd); err != nil {
		return fmt.Errorf("invalid volume setting: %v", err)
	}

	if volumeCmd.Volume < 0 || volumeCmd.Volume > 100 {
		return fmt.Errorf("volume must be between 0 and 100")
	}

	tv.mu.Lock()
	tv.volume = volumeCmd.Volume
	tv.mu.Unlock()

	response := base.CommandResponse{
		CommandID: cmd.ID,
		Status:    "success",
		Data:      fmt.Sprintf("Volume set to %d", volumeCmd.Volume),
	}

	return tv.PublishCommandResponse(response)
}

func (tv *SmartTV) changeChannel(cmd base.Command) error {
	if !tv.power {
		return fmt.Errorf("TV is powered off")
	}

	var channelCmd struct {
		Channel int `json:"channel"`
	}

	if err := json.Unmarshal([]byte(cmd.Payload), &channelCmd); err != nil {
		return fmt.Errorf("invalid channel setting: %v", err)
	}

	if channelCmd.Channel < 1 || channelCmd.Channel > 999 {
		return fmt.Errorf("channel must be between 1 and 999")
	}

	tv.mu.Lock()
	tv.currentChannel = channelCmd.Channel
	tv.currentInput = "antenna"
	tv.mu.Unlock()

	response := base.CommandResponse{
		CommandID: cmd.ID,
		Status:    "success",
		Data:      fmt.Sprintf("Channel changed to %d", channelCmd.Channel),
	}

	return tv.PublishCommandResponse(response)
}

func (tv *SmartTV) changeInput(cmd base.Command) error {
	if !tv.power {
		return fmt.Errorf("TV is powered off")
	}

	var inputCmd struct {
		Input string `json:"input"`
	}

	if err := json.Unmarshal([]byte(cmd.Payload), &inputCmd); err != nil {
		return fmt.Errorf("invalid input setting: %v", err)
	}

	validInputs := []string{"hdmi1", "hdmi2", "hdmi3", "usb", "cast", "antenna"}
	valid := false
	for _, validInput := range validInputs {
		if inputCmd.Input == validInput {
			valid = true
			break
		}
	}

	if !valid {
		return fmt.Errorf("invalid input: %s", inputCmd.Input)
	}

	tv.mu.Lock()
	tv.currentInput = inputCmd.Input
	tv.mu.Unlock()

	response := base.CommandResponse{
		CommandID: cmd.ID,
		Status:    "success",
		Data:      fmt.Sprintf("Input changed to %s", inputCmd.Input),
	}

	return tv.PublishCommandResponse(response)
}

func (tv *SmartTV) launchApp(cmd base.Command) error {
	if !tv.power {
		return fmt.Errorf("TV is powered off")
	}

	var appCmd struct {
		Name string `json:"name"`
	}

	if err := json.Unmarshal([]byte(cmd.Payload), &appCmd); err != nil {
		return fmt.Errorf("invalid app launch request: %v", err)
	}

	tv.mu.Lock()
	defer tv.mu.Unlock()

	for i, app := range tv.apps {
		if app.Name == appCmd.Name {
			tv.apps[i].Active = true
			tv.apps[i].StartTime = time.Now()
			response := base.CommandResponse{
				CommandID: cmd.ID,
				Status:    "success",
				Data:      fmt.Sprintf("App '%s' launched", appCmd.Name),
			}
			return tv.PublishCommandResponse(response)
		}
	}

	return fmt.Errorf("app '%s' not found", appCmd.Name)
}

func (tv *SmartTV) closeApp(cmd base.Command) error {
	var appCmd struct {
		Name string `json:"name"`
	}

	if err := json.Unmarshal([]byte(cmd.Payload), &appCmd); err != nil {
		return fmt.Errorf("invalid app close request: %v", err)
	}

	tv.mu.Lock()
	defer tv.mu.Unlock()

	for i, app := range tv.apps {
		if app.Name == appCmd.Name && app.Active {
			tv.apps[i].Active = false
			response := base.CommandResponse{
				CommandID: cmd.ID,
				Status:    "success",
				Data:      fmt.Sprintf("App '%s' closed", appCmd.Name),
			}
			return tv.PublishCommandResponse(response)
		}
	}

	return fmt.Errorf("app '%s' not found or not active", appCmd.Name)
}

func (tv *SmartTV) startMedia(cmd base.Command) error {
	if !tv.power {
		return fmt.Errorf("TV is powered off")
	}

	var mediaCmd struct {
		MediaType string `json:"media_type"`
		Title     string `json:"title"`
		Duration  int    `json:"duration"`
		Quality   string `json:"quality"`
	}

	if err := json.Unmarshal([]byte(cmd.Payload), &mediaCmd); err != nil {
		return fmt.Errorf("invalid media start request: %v", err)
	}

	tv.mu.Lock()
	tv.mediaPlayback.Playing = true
	tv.mediaPlayback.MediaType = mediaCmd.MediaType
	tv.mediaPlayback.Title = mediaCmd.Title
	tv.mediaPlayback.Duration = mediaCmd.Duration
	tv.mediaPlayback.Position = 0
	if mediaCmd.Quality != "" {
		tv.mediaPlayback.Quality = mediaCmd.Quality
	}
	tv.mu.Unlock()

	response := base.CommandResponse{
		CommandID: cmd.ID,
		Status:    "success",
		Data:      fmt.Sprintf("Started playing %s: %s", mediaCmd.MediaType, mediaCmd.Title),
	}

	return tv.PublishCommandResponse(response)
}

func (tv *SmartTV) stopMedia(cmd base.Command) error {
	tv.mu.Lock()
	tv.mediaPlayback.Playing = false
	tv.mediaPlayback.Bandwidth = 0
	tv.mu.Unlock()

	response := base.CommandResponse{
		CommandID: cmd.ID,
		Status:    "success",
		Data:      "Media playback stopped",
	}

	return tv.PublishCommandResponse(response)
}

func (tv *SmartTV) adjustSettings(cmd base.Command) error {
	var settingsCmd struct {
		Brightness      *int    `json:"brightness,omitempty"`
		Contrast        *int    `json:"contrast,omitempty"`
		ColorSaturation *int    `json:"color_saturation,omitempty"`
		PictureMode     *string `json:"picture_mode,omitempty"`
		AudioMode       *string `json:"audio_mode,omitempty"`
		EcoMode         *bool   `json:"eco_mode,omitempty"`
		HDREnabled      *bool   `json:"hdr_enabled,omitempty"`
	}

	if err := json.Unmarshal([]byte(cmd.Payload), &settingsCmd); err != nil {
		return fmt.Errorf("invalid settings adjustment: %v", err)
	}

	tv.mu.Lock()
	defer tv.mu.Unlock()

	if settingsCmd.Brightness != nil {
		tv.settings.Brightness = *settingsCmd.Brightness
	}
	if settingsCmd.Contrast != nil {
		tv.settings.Contrast = *settingsCmd.Contrast
	}
	if settingsCmd.ColorSaturation != nil {
		tv.settings.ColorSaturation = *settingsCmd.ColorSaturation
	}
	if settingsCmd.PictureMode != nil {
		tv.settings.PictureMode = *settingsCmd.PictureMode
	}
	if settingsCmd.AudioMode != nil {
		tv.settings.AudioMode = *settingsCmd.AudioMode
	}
	if settingsCmd.EcoMode != nil {
		tv.settings.EcoMode = *settingsCmd.EcoMode
	}
	if settingsCmd.HDREnabled != nil {
		tv.settings.HDREnabled = *settingsCmd.HDREnabled
	}

	response := base.CommandResponse{
		CommandID: cmd.ID,
		Status:    "success",
		Data:      "TV settings updated",
	}

	return tv.PublishCommandResponse(response)
}
