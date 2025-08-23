package iot

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

type SecurityCamera struct {
	*base.BaseDevice

	recording   bool
	nightVision bool
	resolution  string
	framerate   int
	motion      MotionDetection
	audio       AudioDetection
	storage     StorageInfo
	stream      StreamInfo
	mu          sync.RWMutex
}

type MotionDetection struct {
	Enabled       bool      `json:"enabled"`
	Sensitivity   int       `json:"sensitivity"`
	LastDetection time.Time `json:"last_detection"`
	Zones         []Zone    `json:"zones"`
}

type AudioDetection struct {
	Enabled       bool      `json:"enabled"`
	Sensitivity   int       `json:"sensitivity"`
	LastDetection time.Time `json:"last_detection"`
	NoiseLevel    float64   `json:"noise_level"`
}

type Zone struct {
	Name   string `json:"name"`
	Active bool   `json:"active"`
	X1     int    `json:"x1"`
	Y1     int    `json:"y1"`
	X2     int    `json:"x2"`
	Y2     int    `json:"y2"`
}

type StorageInfo struct {
	TotalSpace      int64     `json:"total_space"`
	UsedSpace       int64     `json:"used_space"`
	Recordings      int       `json:"recordings"`
	OldestRecording time.Time `json:"oldest_recording"`
}

type StreamInfo struct {
	Active   bool   `json:"active"`
	Viewers  int    `json:"viewers"`
	Bitrate  int    `json:"bitrate"`
	Protocol string `json:"protocol"`
	URL      string `json:"url"`
}

func NewSecurityCamera(deviceConfig base.DeviceConfig, mqttConfig base.MQTTConfig) (*SecurityCamera, error) {
	baseDevice := base.NewBaseDevice(deviceConfig)

	camera := &SecurityCamera{
		BaseDevice:  baseDevice,
		recording:   false,
		nightVision: false,
		resolution:  "1080p",
		framerate:   30,
	}

	camera.initializeMotionDetection()
	camera.initializeAudioDetection()
	camera.initializeStorage()
	camera.initializeStream()

	return camera, nil
}

// Helper functions
func (sc *SecurityCamera) getRandomInt(min, max int) int {
	if min >= max {
		return min
	}
	return rand.Intn(max-min) + min
}

func (sc *SecurityCamera) getRandomFloat(min, max float64) float64 {
	return min + rand.Float64()*(max-min)
}

func (sc *SecurityCamera) initializeMotionDetection() {
	sc.motion = MotionDetection{
		Enabled:     true,
		Sensitivity: 7,
		Zones: []Zone{
			{Name: "main_area", Active: true, X1: 0, Y1: 0, X2: 1920, Y2: 1080},
		},
	}
}

func (sc *SecurityCamera) initializeAudioDetection() {
	sc.audio = AudioDetection{
		Enabled:     true,
		Sensitivity: 5,
		NoiseLevel:  30.0,
	}
}

func (sc *SecurityCamera) initializeStorage() {
	sc.storage = StorageInfo{
		TotalSpace:      64 * 1024 * 1024 * 1024, // 64GB
		UsedSpace:       5 * 1024 * 1024 * 1024,  // 5GB
		Recordings:      150,
		OldestRecording: time.Now().AddDate(0, 0, -30),
	}
}

func (sc *SecurityCamera) initializeStream() {
	sc.stream = StreamInfo{
		Active:   false,
		Viewers:  0,
		Bitrate:  2000,
		Protocol: "RTSP",
		URL:      fmt.Sprintf("rtsp://192.168.1.%d:554/stream", sc.getRandomInt(100, 200)),
	}
}

func (sc *SecurityCamera) Start(ctx context.Context) error {
	if err := sc.BaseDevice.Start(ctx); err != nil {
		return err
	}

	go sc.runMotionDetection(ctx)
	go sc.runAudioDetection(ctx)
	go sc.runStorageManagement(ctx)
	go sc.runStreamManagement(ctx)

	// Security camera started successfully
	return nil
}

func (sc *SecurityCamera) runMotionDetection(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			sc.detectMotion()
		}
	}
}

func (sc *SecurityCamera) runAudioDetection(ctx context.Context) {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			sc.detectAudio()
		}
	}
}

func (sc *SecurityCamera) runStorageManagement(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			sc.manageStorage()
		}
	}
}

func (sc *SecurityCamera) runStreamManagement(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			sc.updateStreamInfo()
		}
	}
}

func (sc *SecurityCamera) detectMotion() {
	if !sc.motion.Enabled {
		return
	}

	sc.mu.Lock()
	defer sc.mu.Unlock()

	if sc.getRandomFloat(0, 100) < float64(sc.motion.Sensitivity) {
		sc.motion.LastDetection = time.Now()

		if !sc.recording {
			sc.recording = true
		}

		event := base.Event{
			EventType: "motion_detected",
			Severity:  "info",
			Message:   "Motion detected in monitoring area",
			Extra: map[string]interface{}{
				"sensitivity": sc.motion.Sensitivity,
				"zone":        "main_area",
				"recording":   sc.recording,
			},
		}
		sc.PublishEvent(event)
	}
}

func (sc *SecurityCamera) detectAudio() {
	if !sc.audio.Enabled {
		return
	}

	sc.mu.Lock()
	defer sc.mu.Unlock()

	sc.audio.NoiseLevel = 25.0 + sc.getRandomFloat(-10, 35)

	if sc.audio.NoiseLevel > float64(sc.audio.Sensitivity*10) {
		sc.audio.LastDetection = time.Now()

		event := base.Event{
			EventType: "audio_detected",
			Severity:  "info",
			Message:   fmt.Sprintf("Audio event detected: %.1f dB", sc.audio.NoiseLevel),
			Extra: map[string]interface{}{
				"noise_level": sc.audio.NoiseLevel,
				"sensitivity": sc.audio.Sensitivity,
				"recording":   sc.recording,
			},
		}
		sc.PublishEvent(event)
	}
}

func (sc *SecurityCamera) manageStorage() {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	if sc.recording {
		recordingSize := int64(sc.getRandomInt(50, 200) * 1024 * 1024) // 50-200MB
		sc.storage.UsedSpace += recordingSize
		sc.storage.Recordings++
	}

	usagePercent := float64(sc.storage.UsedSpace) / float64(sc.storage.TotalSpace) * 100

	if usagePercent > 90 {
		sc.cleanupOldRecordings()

		event := base.Event{
			EventType: "storage_full",
			Severity:  "warning",
			Message:   fmt.Sprintf("Storage usage high: %.1f%%", usagePercent),
			Extra: map[string]interface{}{
				"usage_percent": usagePercent,
				"used_space":    sc.storage.UsedSpace,
				"total_space":   sc.storage.TotalSpace,
				"recordings":    sc.storage.Recordings,
			},
		}
		sc.PublishEvent(event)
	}
}

func (sc *SecurityCamera) cleanupOldRecordings() {
	if sc.storage.Recordings > 10 {
		recordingsToDelete := sc.storage.Recordings / 4
		sc.storage.Recordings -= recordingsToDelete
		sc.storage.UsedSpace -= int64(recordingsToDelete * 100 * 1024 * 1024) // Average 100MB per recording
		sc.storage.OldestRecording = time.Now().AddDate(0, 0, -20)
	}
}

func (sc *SecurityCamera) updateStreamInfo() {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	if sc.stream.Active {
		sc.stream.Viewers = sc.getRandomInt(0, 3)

		if sc.stream.Viewers == 0 && sc.getRandomFloat(0, 100) < 20 {
			sc.stream.Active = false
		}
	} else {
		if sc.getRandomFloat(0, 100) < 10 {
			sc.stream.Active = true
			sc.stream.Viewers = sc.getRandomInt(1, 2)
		}
	}
}

func (sc *SecurityCamera) GenerateStatePayload() base.StatePayload {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	baseState := sc.BaseDevice.GenerateStatePayload()

	if baseState.Extra == nil {
		baseState.Extra = make(map[string]interface{})
	}

	baseState.Extra["camera"] = map[string]interface{}{
		"recording":    sc.recording,
		"night_vision": sc.nightVision,
		"resolution":   sc.resolution,
		"framerate":    sc.framerate,
	}

	baseState.Extra["motion_detection"] = sc.motion
	baseState.Extra["audio_detection"] = sc.audio
	baseState.Extra["storage"] = sc.storage
	baseState.Extra["stream"] = sc.stream

	return baseState
}

func (sc *SecurityCamera) GenerateTelemetryData() map[string]base.TelemetryPayload {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	telemetry := sc.BaseDevice.GenerateTelemetryData()

	telemetry["storage_usage"] = base.TelemetryPayload{
		"timestamp": time.Now(),
		"value":     float64(sc.storage.UsedSpace) / float64(sc.storage.TotalSpace) * 100,
		"unit":      "percent",
		"tags": map[string]string{
			"metric": "storage_usage",
		},
	}

	telemetry["recording_count"] = base.TelemetryPayload{
		"timestamp": time.Now(),
		"value":     float64(sc.storage.Recordings),
		"unit":      "count",
		"tags": map[string]string{
			"metric": "recording_count",
		},
	}

	telemetry["stream_viewers"] = base.TelemetryPayload{
		"timestamp": time.Now(),
		"value":     float64(sc.stream.Viewers),
		"unit":      "count",
		"tags": map[string]string{
			"metric": "stream_viewers",
		},
	}

	telemetry["noise_level"] = base.TelemetryPayload{
		"timestamp": time.Now(),
		"value":     sc.audio.NoiseLevel,
		"unit":      "db",
		"tags": map[string]string{
			"metric": "noise_level",
		},
	}

	return telemetry
}

func (sc *SecurityCamera) HandleCommand(cmd base.Command) error {
	// Handling command: cmd.Action

	switch cmd.Action {
	case "start_recording":
		return sc.startRecording(cmd)
	case "stop_recording":
		return sc.stopRecording(cmd)
	case "toggle_night_vision":
		return sc.toggleNightVision(cmd)
	case "set_motion_sensitivity":
		return sc.setMotionSensitivity(cmd)
	case "get_recordings":
		return sc.getRecordings(cmd)
	case "delete_recordings":
		return sc.deleteRecordings(cmd)
	case "start_stream":
		return sc.startStream(cmd)
	case "stop_stream":
		return sc.stopStream(cmd)
	default:
		return sc.BaseDevice.HandleCommand(cmd)
	}
}

func (sc *SecurityCamera) startRecording(cmd base.Command) error {
	sc.mu.Lock()
	sc.recording = true
	sc.mu.Unlock()

	response := base.CommandResponse{
		CommandID: cmd.ID,
		Status:    "success",
		Data:      "Recording started",
	}

	return sc.PublishCommandResponse(response)
}

func (sc *SecurityCamera) stopRecording(cmd base.Command) error {
	sc.mu.Lock()
	sc.recording = false
	sc.mu.Unlock()

	response := base.CommandResponse{
		CommandID: cmd.ID,
		Status:    "success",
		Data:      "Recording stopped",
	}

	return sc.PublishCommandResponse(response)
}

func (sc *SecurityCamera) toggleNightVision(cmd base.Command) error {
	sc.mu.Lock()
	sc.nightVision = !sc.nightVision
	state := sc.nightVision
	sc.mu.Unlock()

	response := base.CommandResponse{
		CommandID: cmd.ID,
		Status:    "success",
		Data:      fmt.Sprintf("Night vision %s", map[bool]string{true: "enabled", false: "disabled"}[state]),
	}

	return sc.PublishCommandResponse(response)
}

func (sc *SecurityCamera) setMotionSensitivity(cmd base.Command) error {
	var sensitivityCmd struct {
		Sensitivity int `json:"sensitivity"`
	}

	if err := json.Unmarshal([]byte(cmd.Payload), &sensitivityCmd); err != nil {
		return fmt.Errorf("invalid sensitivity setting: %v", err)
	}

	if sensitivityCmd.Sensitivity < 1 || sensitivityCmd.Sensitivity > 10 {
		return fmt.Errorf("sensitivity must be between 1 and 10")
	}

	sc.mu.Lock()
	sc.motion.Sensitivity = sensitivityCmd.Sensitivity
	sc.mu.Unlock()

	response := base.CommandResponse{
		CommandID: cmd.ID,
		Status:    "success",
		Data:      fmt.Sprintf("Motion sensitivity set to %d", sensitivityCmd.Sensitivity),
	}

	return sc.PublishCommandResponse(response)
}

func (sc *SecurityCamera) getRecordings(cmd base.Command) error {
	sc.mu.RLock()
	storageData, _ := json.Marshal(sc.storage)
	sc.mu.RUnlock()

	response := base.CommandResponse{
		CommandID: cmd.ID,
		Status:    "success",
		Data:      string(storageData),
	}

	return sc.PublishCommandResponse(response)
}

func (sc *SecurityCamera) deleteRecordings(cmd base.Command) error {
	var deleteCmd struct {
		OlderThan string `json:"older_than"` // "7d", "30d", "all"
	}

	if err := json.Unmarshal([]byte(cmd.Payload), &deleteCmd); err != nil {
		return fmt.Errorf("invalid delete request: %v", err)
	}

	sc.mu.Lock()
	defer sc.mu.Unlock()

	var deletedCount int
	switch deleteCmd.OlderThan {
	case "all":
		deletedCount = sc.storage.Recordings
		sc.storage.Recordings = 0
		sc.storage.UsedSpace = 0
	case "30d":
		deletedCount = sc.storage.Recordings / 2
		sc.storage.Recordings -= deletedCount
		sc.storage.UsedSpace -= int64(deletedCount * 100 * 1024 * 1024)
	default:
		deletedCount = sc.storage.Recordings / 4
		sc.storage.Recordings -= deletedCount
		sc.storage.UsedSpace -= int64(deletedCount * 100 * 1024 * 1024)
	}

	response := base.CommandResponse{
		CommandID: cmd.ID,
		Status:    "success",
		Data:      fmt.Sprintf("%d recordings deleted", deletedCount),
	}

	return sc.PublishCommandResponse(response)
}

func (sc *SecurityCamera) startStream(cmd base.Command) error {
	sc.mu.Lock()
	sc.stream.Active = true
	sc.stream.Viewers = 1
	sc.mu.Unlock()

	response := base.CommandResponse{
		CommandID: cmd.ID,
		Status:    "success",
		Data:      fmt.Sprintf("Stream started: %s", sc.stream.URL),
	}

	return sc.PublishCommandResponse(response)
}

func (sc *SecurityCamera) stopStream(cmd base.Command) error {
	sc.mu.Lock()
	sc.stream.Active = false
	sc.stream.Viewers = 0
	sc.mu.Unlock()

	response := base.CommandResponse{
		CommandID: cmd.ID,
		Status:    "success",
		Data:      "Stream stopped",
	}

	return sc.PublishCommandResponse(response)
}
