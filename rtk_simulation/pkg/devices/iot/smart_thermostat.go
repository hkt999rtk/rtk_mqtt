package iot

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

type SmartThermostat struct {
	*base.BaseDevice

	currentTemp float64
	targetTemp  float64
	mode        string // heat, cool, auto, off
	fanMode     string // auto, on, circulate
	fanSpeed    int
	humidity    float64
	schedules   []ThermostatSchedule
	hvacSystem  HVACSystemStatus
	energyUsage EnergyUsageData
	sensors     []RemoteSensor
	mu          sync.RWMutex
}

type ThermostatSchedule struct {
	Name       string   `json:"name"`
	Days       []string `json:"days"`
	StartTime  string   `json:"start_time"`
	EndTime    string   `json:"end_time"`
	TargetTemp float64  `json:"target_temp"`
	Mode       string   `json:"mode"`
	Enabled    bool     `json:"enabled"`
}

type HVACSystemStatus struct {
	HeatingActive   bool      `json:"heating_active"`
	CoolingActive   bool      `json:"cooling_active"`
	FanRunning      bool      `json:"fan_running"`
	LastMaintenance time.Time `json:"last_maintenance"`
	FilterStatus    string    `json:"filter_status"`
	SystemHealth    string    `json:"system_health"`
}

type EnergyUsageData struct {
	DailyUsage       float64   `json:"daily_usage"`   // kWh
	MonthlyUsage     float64   `json:"monthly_usage"` // kWh
	YearlyUsage      float64   `json:"yearly_usage"`  // kWh
	Cost             float64   `json:"cost"`          // USD
	LastReading      time.Time `json:"last_reading"`
	EfficiencyRating float64   `json:"efficiency_rating"`
}

type RemoteSensor struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	Temperature  float64 `json:"temperature"`
	Humidity     float64 `json:"humidity"`
	Active       bool    `json:"active"`
	BatteryLevel float64 `json:"battery_level"`
}

func NewSmartThermostat(deviceConfig base.DeviceConfig, mqttConfig base.MQTTConfig) (*SmartThermostat, error) {
	baseDevice := base.NewBaseDevice(deviceConfig)

	thermostat := &SmartThermostat{
		BaseDevice:  baseDevice,
		currentTemp: 22.0,
		targetTemp:  22.0,
		mode:        "auto",
		fanMode:     "auto",
		fanSpeed:    1,
		humidity:    45.0,
	}

	thermostat.initializeSchedules()
	thermostat.initializeHVACSystem()
	thermostat.initializeEnergyUsage()
	thermostat.initializeRemoteSensors()

	return thermostat, nil
}

// Helper functions
func (st *SmartThermostat) getRandomFloat(min, max float64) float64 {
	return min + rand.Float64()*(max-min)
}

func (st *SmartThermostat) initializeSchedules() {
	st.schedules = []ThermostatSchedule{
		{
			Name:       "morning",
			Days:       []string{"monday", "tuesday", "wednesday", "thursday", "friday"},
			StartTime:  "06:00",
			EndTime:    "08:00",
			TargetTemp: 21.0,
			Mode:       "heat",
			Enabled:    true,
		},
		{
			Name:       "work_hours",
			Days:       []string{"monday", "tuesday", "wednesday", "thursday", "friday"},
			StartTime:  "08:00",
			EndTime:    "18:00",
			TargetTemp: 19.0,
			Mode:       "auto",
			Enabled:    true,
		},
		{
			Name:       "evening",
			Days:       []string{"monday", "tuesday", "wednesday", "thursday", "friday", "saturday", "sunday"},
			StartTime:  "18:00",
			EndTime:    "22:00",
			TargetTemp: 22.0,
			Mode:       "auto",
			Enabled:    true,
		},
		{
			Name:       "sleep",
			Days:       []string{"monday", "tuesday", "wednesday", "thursday", "friday", "saturday", "sunday"},
			StartTime:  "22:00",
			EndTime:    "06:00",
			TargetTemp: 20.0,
			Mode:       "auto",
			Enabled:    true,
		},
	}
}

func (st *SmartThermostat) initializeHVACSystem() {
	st.hvacSystem = HVACSystemStatus{
		HeatingActive:   false,
		CoolingActive:   false,
		FanRunning:      false,
		LastMaintenance: time.Now().AddDate(0, -2, 0),
		FilterStatus:    "good",
		SystemHealth:    "healthy",
	}
}

func (st *SmartThermostat) initializeEnergyUsage() {
	st.energyUsage = EnergyUsageData{
		DailyUsage:       15.5,
		MonthlyUsage:     465.0,
		YearlyUsage:      5580.0,
		Cost:             0.12,
		LastReading:      time.Now(),
		EfficiencyRating: 8.5,
	}
}

func (st *SmartThermostat) initializeRemoteSensors() {
	st.sensors = []RemoteSensor{
		{
			ID:           "bedroom_sensor",
			Name:         "Bedroom",
			Temperature:  21.5,
			Humidity:     43.0,
			Active:       true,
			BatteryLevel: 85.0,
		},
		{
			ID:           "living_room_sensor",
			Name:         "Living Room",
			Temperature:  22.2,
			Humidity:     47.0,
			Active:       true,
			BatteryLevel: 92.0,
		},
	}
}

func (st *SmartThermostat) Start(ctx context.Context) error {
	if err := st.BaseDevice.Start(ctx); err != nil {
		return err
	}

	go st.runTemperatureControl(ctx)
	go st.runScheduleManager(ctx)
	go st.runEnergyMonitoring(ctx)
	go st.runMaintenanceCheck(ctx)
	go st.runRemoteSensorUpdate(ctx)

	// st.Logger.Info("Smart thermostat started successfully")
	return nil
}

func (st *SmartThermostat) runTemperatureControl(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			st.updateTemperatureControl()
		}
	}
}

func (st *SmartThermostat) runScheduleManager(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			st.checkSchedules()
		}
	}
}

func (st *SmartThermostat) runEnergyMonitoring(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			st.updateEnergyUsage()
		}
	}
}

func (st *SmartThermostat) runMaintenanceCheck(ctx context.Context) {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			st.checkMaintenanceSchedule()
		}
	}
}

func (st *SmartThermostat) runRemoteSensorUpdate(ctx context.Context) {
	ticker := time.NewTicker(2 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			st.updateRemoteSensors()
		}
	}
}

func (st *SmartThermostat) updateTemperatureControl() {
	st.mu.Lock()
	defer st.mu.Unlock()

	// Simulate temperature changes based on HVAC operation
	if st.hvacSystem.HeatingActive {
		st.currentTemp += st.getRandomFloat(0.1, 0.3)
	} else if st.hvacSystem.CoolingActive {
		st.currentTemp -= st.getRandomFloat(0.1, 0.3)
	} else {
		// Natural temperature drift
		st.currentTemp += st.getRandomFloat(-0.1, 0.1)
	}

	tempDiff := st.targetTemp - st.currentTemp
	tolerance := 0.5

	// Control logic
	switch st.mode {
	case "heat":
		if tempDiff > tolerance {
			st.hvacSystem.HeatingActive = true
			st.hvacSystem.CoolingActive = false
		} else {
			st.hvacSystem.HeatingActive = false
		}
	case "cool":
		if tempDiff < -tolerance {
			st.hvacSystem.CoolingActive = true
			st.hvacSystem.HeatingActive = false
		} else {
			st.hvacSystem.CoolingActive = false
		}
	case "auto":
		if tempDiff > tolerance {
			st.hvacSystem.HeatingActive = true
			st.hvacSystem.CoolingActive = false
		} else if tempDiff < -tolerance {
			st.hvacSystem.CoolingActive = true
			st.hvacSystem.HeatingActive = false
		} else {
			st.hvacSystem.HeatingActive = false
			st.hvacSystem.CoolingActive = false
		}
	case "off":
		st.hvacSystem.HeatingActive = false
		st.hvacSystem.CoolingActive = false
	}

	// Fan control
	if st.fanMode == "on" || st.hvacSystem.HeatingActive || st.hvacSystem.CoolingActive {
		st.hvacSystem.FanRunning = true
	} else if st.fanMode == "auto" {
		st.hvacSystem.FanRunning = st.hvacSystem.HeatingActive || st.hvacSystem.CoolingActive
	} else {
		st.hvacSystem.FanRunning = false
	}

	// Update humidity
	if st.hvacSystem.CoolingActive {
		st.humidity -= st.getRandomFloat(0, 2)
	} else if st.hvacSystem.HeatingActive {
		st.humidity += st.getRandomFloat(0, 1)
	}

	if st.humidity < 30 {
		st.humidity = 30
	}
	if st.humidity > 70 {
		st.humidity = 70
	}
}

func (st *SmartThermostat) checkSchedules() {
	st.mu.Lock()
	defer st.mu.Unlock()

	now := time.Now()
	currentTime := now.Format("15:04")
	currentDay := now.Weekday().String()

	for _, schedule := range st.schedules {
		if !schedule.Enabled {
			continue
		}

		// Check if today matches schedule
		dayMatch := false
		for _, day := range schedule.Days {
			if day == currentDay {
				dayMatch = true
				break
			}
		}

		if !dayMatch {
			continue
		}

		// Check if current time is within schedule
		if st.isTimeInRange(currentTime, schedule.StartTime, schedule.EndTime) {
			if st.targetTemp != schedule.TargetTemp || st.mode != schedule.Mode {
				oldTemp := st.targetTemp
				oldMode := st.mode

				st.targetTemp = schedule.TargetTemp
				st.mode = schedule.Mode

				event := base.Event{
					EventType: "schedule_activated",
					Severity:  "info",
					Message:   fmt.Sprintf("Schedule '%s' activated", schedule.Name),
					Extra: map[string]interface{}{
						"schedule_name":   schedule.Name,
						"old_target_temp": oldTemp,
						"new_target_temp": st.targetTemp,
						"old_mode":        oldMode,
						"new_mode":        st.mode,
					},
				}
				st.PublishEvent(event)
			}
			break
		}
	}
}

func (st *SmartThermostat) isTimeInRange(current, start, end string) bool {
	currentTime, _ := time.Parse("15:04", current)
	startTime, _ := time.Parse("15:04", start)
	endTime, _ := time.Parse("15:04", end)

	if endTime.Before(startTime) {
		// Schedule crosses midnight
		return currentTime.After(startTime) || currentTime.Before(endTime)
	}

	return currentTime.After(startTime) && currentTime.Before(endTime)
}

func (st *SmartThermostat) updateEnergyUsage() {
	st.mu.Lock()
	defer st.mu.Unlock()

	// Calculate energy usage based on HVAC operation
	usage := 0.0
	if st.hvacSystem.HeatingActive {
		usage += st.getRandomFloat(2.0, 4.0)
	}
	if st.hvacSystem.CoolingActive {
		usage += st.getRandomFloat(3.0, 5.0)
	}
	if st.hvacSystem.FanRunning {
		usage += st.getRandomFloat(0.1, 0.3)
	}

	st.energyUsage.DailyUsage += usage / 6 // 10-minute intervals
	st.energyUsage.LastReading = time.Now()

	// Calculate cost (assuming $0.12 per kWh)
	st.energyUsage.Cost = st.energyUsage.DailyUsage * 0.12

	// Update efficiency rating based on usage patterns
	if st.energyUsage.DailyUsage > 20 {
		st.energyUsage.EfficiencyRating -= 0.1
	} else if st.energyUsage.DailyUsage < 10 {
		st.energyUsage.EfficiencyRating += 0.05
	}

	if st.energyUsage.EfficiencyRating < 1 {
		st.energyUsage.EfficiencyRating = 1
	}
	if st.energyUsage.EfficiencyRating > 10 {
		st.energyUsage.EfficiencyRating = 10
	}
}

func (st *SmartThermostat) checkMaintenanceSchedule() {
	st.mu.Lock()
	defer st.mu.Unlock()

	now := time.Now()
	daysSinceLastMaintenance := int(now.Sub(st.hvacSystem.LastMaintenance).Hours() / 24)

	// Filter status
	if daysSinceLastMaintenance > 90 {
		st.hvacSystem.FilterStatus = "needs_replacement"
		event := base.Event{
			EventType: "filter_replacement_due",
			Severity:  "warning",
			Message:   "HVAC filter needs replacement",
			Extra: map[string]interface{}{
				"days_since_last_maintenance": daysSinceLastMaintenance,
				"filter_status":               st.hvacSystem.FilterStatus,
			},
		}
		st.PublishEvent(event)
	} else if daysSinceLastMaintenance > 60 {
		st.hvacSystem.FilterStatus = "dirty"
	}

	// System health
	if daysSinceLastMaintenance > 365 {
		st.hvacSystem.SystemHealth = "needs_maintenance"
		event := base.Event{
			EventType: "maintenance_due",
			Severity:  "warning",
			Message:   "HVAC system maintenance is due",
			Extra: map[string]interface{}{
				"days_since_last_maintenance": daysSinceLastMaintenance,
				"system_health":               st.hvacSystem.SystemHealth,
			},
		}
		st.PublishEvent(event)
	} else if daysSinceLastMaintenance > 300 {
		st.hvacSystem.SystemHealth = "fair"
	}
}

func (st *SmartThermostat) updateRemoteSensors() {
	st.mu.Lock()
	defer st.mu.Unlock()

	for i := range st.sensors {
		sensor := &st.sensors[i]

		// Update temperature readings
		sensor.Temperature += st.getRandomFloat(-0.5, 0.5)
		sensor.Humidity += st.getRandomFloat(-2, 2)

		// Battery drain
		sensor.BatteryLevel -= st.getRandomFloat(0.01, 0.05)
		if sensor.BatteryLevel < 0 {
			sensor.BatteryLevel = 0
			sensor.Active = false
		}

		// Low battery alert
		if sensor.BatteryLevel < 20 && sensor.BatteryLevel > 19 {
			event := base.Event{
				EventType: "sensor_battery_low",
				Severity:  "warning",
				Message:   fmt.Sprintf("Remote sensor '%s' has low battery", sensor.Name),

				Extra: map[string]interface{}{
					"sensor_id":     sensor.ID,
					"sensor_name":   sensor.Name,
					"battery_level": sensor.BatteryLevel,
				},
			}
			st.PublishEvent(event)
		}
	}
}

func (st *SmartThermostat) GenerateStatePayload() base.StatePayload {
	st.mu.RLock()
	defer st.mu.RUnlock()

	baseState := st.BaseDevice.GenerateStatePayload()

	if baseState.Extra == nil {
		baseState.Extra = make(map[string]interface{})
	}

	baseState.Extra["thermostat"] = map[string]interface{}{
		"current_temp": st.currentTemp,
		"target_temp":  st.targetTemp,
		"mode":         st.mode,
		"fan_mode":     st.fanMode,
		"fan_speed":    st.fanSpeed,
		"humidity":     st.humidity,
	}

	baseState.Extra["hvac_system"] = st.hvacSystem
	baseState.Extra["energy_usage"] = st.energyUsage
	baseState.Extra["remote_sensors"] = st.sensors

	return baseState
}

func (st *SmartThermostat) GenerateTelemetryData() map[string]base.TelemetryPayload {
	st.mu.RLock()
	defer st.mu.RUnlock()

	telemetry := st.BaseDevice.GenerateTelemetryData()

	telemetry["current_temperature"] = base.TelemetryPayload{
		"timestamp": time.Now(),

		"value": st.currentTemp,
		"unit":  "celsius",
		"tags": map[string]string{
			"sensor": "thermostat",
		},
	}

	telemetry["target_temperature"] = base.TelemetryPayload{
		"timestamp": time.Now(),

		"value": st.targetTemp,
		"unit":  "celsius",
		"tags": map[string]string{
			"setting": "target",
		},
	}

	telemetry["humidity"] = base.TelemetryPayload{
		"timestamp": time.Now(),

		"value": st.humidity,
		"unit":  "percent",
		"tags": map[string]string{
			"sensor": "humidity",
		},
	}

	telemetry["energy_usage"] = base.TelemetryPayload{
		"timestamp": time.Now(),

		"value": st.energyUsage.DailyUsage,
		"unit":  "kwh",
		"tags": map[string]string{
			"period": "daily",
		},
	}

	telemetry["efficiency_rating"] = base.TelemetryPayload{
		"timestamp": time.Now(),

		"value": st.energyUsage.EfficiencyRating,
		"unit":  "rating",
		"tags": map[string]string{
			"metric": "efficiency",
		},
	}

	return telemetry
}

func (st *SmartThermostat) HandleCommand(cmd base.Command) error {
	// st.Logger.Infof("Handling command: %s", cmd.Action)

	switch cmd.Action {
	case "set_temperature":
		return st.setTemperature(cmd)
	case "set_mode":
		return st.setMode(cmd)
	case "set_fan_mode":
		return st.setFanMode(cmd)
	case "create_schedule":
		return st.createSchedule(cmd)
	case "update_schedule":
		return st.updateSchedule(cmd)
	case "delete_schedule":
		return st.deleteSchedule(cmd)
	case "get_energy_usage":
		return st.getEnergyUsage(cmd)
	case "reset_filter":
		return st.resetFilter(cmd)
	default:
		return st.BaseDevice.HandleCommand(cmd)
	}
}

func (st *SmartThermostat) setTemperature(cmd base.Command) error {
	var tempCmd struct {
		TargetTemp float64 `json:"target_temp"`
	}

	if err := json.Unmarshal([]byte(cmd.Payload), &tempCmd); err != nil {
		return fmt.Errorf("invalid temperature setting: %v", err)
	}

	if tempCmd.TargetTemp < 10 || tempCmd.TargetTemp > 35 {
		return fmt.Errorf("temperature must be between 10°C and 35°C")
	}

	st.mu.Lock()
	st.targetTemp = tempCmd.TargetTemp
	st.mu.Unlock()

	response := base.CommandResponse{
		CommandID: cmd.ID,
		Status:    "success",
		Data:      fmt.Sprintf("Target temperature set to %.1f°C", tempCmd.TargetTemp),
	}

	return st.PublishCommandResponse(response)
}

func (st *SmartThermostat) setMode(cmd base.Command) error {
	var modeCmd struct {
		Mode string `json:"mode"`
	}

	if err := json.Unmarshal([]byte(cmd.Payload), &modeCmd); err != nil {
		return fmt.Errorf("invalid mode setting: %v", err)
	}

	validModes := []string{"heat", "cool", "auto", "off"}
	valid := false
	for _, validMode := range validModes {
		if modeCmd.Mode == validMode {
			valid = true
			break
		}
	}

	if !valid {
		return fmt.Errorf("invalid mode: %s. Valid modes: %v", modeCmd.Mode, validModes)
	}

	st.mu.Lock()
	st.mode = modeCmd.Mode
	st.mu.Unlock()

	response := base.CommandResponse{
		CommandID: cmd.ID,
		Status:    "success",
		Data:      fmt.Sprintf("Mode set to %s", modeCmd.Mode),
	}

	return st.PublishCommandResponse(response)
}

func (st *SmartThermostat) setFanMode(cmd base.Command) error {
	var fanCmd struct {
		FanMode string `json:"fan_mode"`
	}

	if err := json.Unmarshal([]byte(cmd.Payload), &fanCmd); err != nil {
		return fmt.Errorf("invalid fan mode setting: %v", err)
	}

	validModes := []string{"auto", "on", "circulate"}
	valid := false
	for _, validMode := range validModes {
		if fanCmd.FanMode == validMode {
			valid = true
			break
		}
	}

	if !valid {
		return fmt.Errorf("invalid fan mode: %s. Valid modes: %v", fanCmd.FanMode, validModes)
	}

	st.mu.Lock()
	st.fanMode = fanCmd.FanMode
	st.mu.Unlock()

	response := base.CommandResponse{
		CommandID: cmd.ID,
		Status:    "success",
		Data:      fmt.Sprintf("Fan mode set to %s", fanCmd.FanMode),
	}

	return st.PublishCommandResponse(response)
}

func (st *SmartThermostat) createSchedule(cmd base.Command) error {
	var schedule ThermostatSchedule
	if err := json.Unmarshal([]byte(cmd.Payload), &schedule); err != nil {
		return fmt.Errorf("invalid schedule data: %v", err)
	}

	st.mu.Lock()
	st.schedules = append(st.schedules, schedule)
	st.mu.Unlock()

	response := base.CommandResponse{
		CommandID: cmd.ID,
		Status:    "success",
		Data:      fmt.Sprintf("Schedule '%s' created", schedule.Name),
	}

	return st.PublishCommandResponse(response)
}

func (st *SmartThermostat) updateSchedule(cmd base.Command) error {
	var scheduleUpdate struct {
		Name     string             `json:"name"`
		Schedule ThermostatSchedule `json:"schedule"`
	}

	if err := json.Unmarshal([]byte(cmd.Payload), &scheduleUpdate); err != nil {
		return fmt.Errorf("invalid schedule update: %v", err)
	}

	st.mu.Lock()
	defer st.mu.Unlock()

	for i, schedule := range st.schedules {
		if schedule.Name == scheduleUpdate.Name {
			st.schedules[i] = scheduleUpdate.Schedule
			response := base.CommandResponse{
				CommandID: cmd.ID,
				Status:    "success",
				Data:      fmt.Sprintf("Schedule '%s' updated", scheduleUpdate.Name),
			}
			return st.PublishCommandResponse(response)
		}
	}

	return fmt.Errorf("schedule '%s' not found", scheduleUpdate.Name)
}

func (st *SmartThermostat) deleteSchedule(cmd base.Command) error {
	var deleteCmd struct {
		Name string `json:"name"`
	}

	if err := json.Unmarshal([]byte(cmd.Payload), &deleteCmd); err != nil {
		return fmt.Errorf("invalid delete request: %v", err)
	}

	st.mu.Lock()
	defer st.mu.Unlock()

	for i, schedule := range st.schedules {
		if schedule.Name == deleteCmd.Name {
			st.schedules = append(st.schedules[:i], st.schedules[i+1:]...)
			response := base.CommandResponse{
				CommandID: cmd.ID,
				Status:    "success",
				Data:      fmt.Sprintf("Schedule '%s' deleted", deleteCmd.Name),
			}
			return st.PublishCommandResponse(response)
		}
	}

	return fmt.Errorf("schedule '%s' not found", deleteCmd.Name)
}

func (st *SmartThermostat) getEnergyUsage(cmd base.Command) error {
	st.mu.RLock()
	energyData, _ := json.Marshal(st.energyUsage)
	st.mu.RUnlock()

	response := base.CommandResponse{
		CommandID: cmd.ID,
		Status:    "success",
		Data:      string(energyData),
	}

	return st.PublishCommandResponse(response)
}

func (st *SmartThermostat) resetFilter(cmd base.Command) error {
	st.mu.Lock()
	st.hvacSystem.FilterStatus = "good"
	st.hvacSystem.LastMaintenance = time.Now()
	st.mu.Unlock()

	response := base.CommandResponse{
		CommandID: cmd.ID,
		Status:    "success",
		Data:      "Filter status reset - marked as replaced",
	}

	return st.PublishCommandResponse(response)
}
