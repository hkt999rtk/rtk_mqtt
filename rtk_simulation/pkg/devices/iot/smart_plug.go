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

type SmartPlug struct {
	*base.BaseDevice

	power            bool
	powerConsumption float64 // watts
	voltage          float64 // volts
	current          float64 // amps
	energyUsage      PlugEnergyUsage
	schedule         []PlugSchedule
	connectedDevice  ConnectedDevice
	protections      SafetyProtections
	mu               sync.RWMutex
}

type PlugEnergyUsage struct {
	InstantPower  float64   `json:"instant_power"`  // watts
	DailyEnergy   float64   `json:"daily_energy"`   // wh
	WeeklyEnergy  float64   `json:"weekly_energy"`  // wh
	MonthlyEnergy float64   `json:"monthly_energy"` // wh
	DailyCost     float64   `json:"daily_cost"`     // usd
	LastReading   time.Time `json:"last_reading"`
	PeakPower     float64   `json:"peak_power"`    // watts
	AveragePower  float64   `json:"average_power"` // watts
}

type PlugSchedule struct {
	Name      string   `json:"name"`
	Enabled   bool     `json:"enabled"`
	Days      []string `json:"days"`
	StartTime string   `json:"start_time"`
	EndTime   string   `json:"end_time"`
	Action    string   `json:"action"` // on, off
	Repeat    bool     `json:"repeat"`
}

type ConnectedDevice struct {
	Name           string  `json:"name"`
	Type           string  `json:"type"`
	RatedPower     float64 `json:"rated_power"`     // watts
	EstimatedUsage float64 `json:"estimated_usage"` // hours per day
	DeviceStatus   string  `json:"device_status"`   // on, off, standby
}

type SafetyProtections struct {
	OverloadProtection bool       `json:"overload_protection"`
	MaxCurrent         float64    `json:"max_current"` // amps
	MaxPower           float64    `json:"max_power"`   // watts
	SurgeProtection    bool       `json:"surge_protection"`
	ChildLock          bool       `json:"child_lock"`
	AutoShutoff        bool       `json:"auto_shutoff"`
	ShutoffDelay       int        `json:"shutoff_delay"` // minutes
	LastOverload       *time.Time `json:"last_overload,omitempty"`
}

func NewSmartPlug(deviceConfig base.DeviceConfig, mqttConfig base.MQTTConfig) (*SmartPlug, error) {
	baseDevice := base.NewBaseDevice(deviceConfig)

	plug := &SmartPlug{
		BaseDevice:       baseDevice,
		power:            false,
		powerConsumption: 0.0,
		voltage:          120.0,
		current:          0.0,
	}

	plug.initializeEnergyUsage()
	plug.initializeSchedule()
	plug.initializeConnectedDevice()
	plug.initializeSafetyProtections()

	return plug, nil
}

// Helper functions
func (sp *SmartPlug) getRandomInt(min, max int) int {
	if min >= max {
		return min
	}
	return rand.Intn(max-min) + min
}

func (sp *SmartPlug) getRandomFloat(min, max float64) float64 {
	return min + rand.Float64()*(max-min)
}

func (sp *SmartPlug) initializeEnergyUsage() {
	sp.energyUsage = PlugEnergyUsage{
		InstantPower:  0.0,
		DailyEnergy:   0.0,
		WeeklyEnergy:  0.0,
		MonthlyEnergy: 0.0,
		DailyCost:     0.0,
		LastReading:   time.Now(),
		PeakPower:     0.0,
		AveragePower:  0.0,
	}
}

func (sp *SmartPlug) initializeSchedule() {
	sp.schedule = []PlugSchedule{
		{
			Name:      "morning_routine",
			Enabled:   true,
			Days:      []string{"monday", "tuesday", "wednesday", "thursday", "friday"},
			StartTime: "07:00",
			EndTime:   "09:00",
			Action:    "on",
			Repeat:    true,
		},
		{
			Name:      "evening_routine",
			Enabled:   true,
			Days:      []string{"monday", "tuesday", "wednesday", "thursday", "friday", "saturday", "sunday"},
			StartTime: "18:00",
			EndTime:   "23:00",
			Action:    "on",
			Repeat:    true,
		},
	}
}

func (sp *SmartPlug) initializeConnectedDevice() {
	devices := []ConnectedDevice{
		{Name: "Desk Lamp", Type: "lighting", RatedPower: 60.0, EstimatedUsage: 6.0, DeviceStatus: "off"},
		{Name: "Coffee Maker", Type: "appliance", RatedPower: 1200.0, EstimatedUsage: 1.0, DeviceStatus: "off"},
		{Name: "Phone Charger", Type: "electronics", RatedPower: 18.0, EstimatedUsage: 8.0, DeviceStatus: "standby"},
		{Name: "Space Heater", Type: "heating", RatedPower: 1500.0, EstimatedUsage: 4.0, DeviceStatus: "off"},
		{Name: "Air Purifier", Type: "appliance", RatedPower: 45.0, EstimatedUsage: 12.0, DeviceStatus: "off"},
	}

	sp.connectedDevice = devices[sp.getRandomInt(0, len(devices))]
}

func (sp *SmartPlug) initializeSafetyProtections() {
	sp.protections = SafetyProtections{
		OverloadProtection: true,
		MaxCurrent:         15.0,   // 15 amps
		MaxPower:           1800.0, // 1800 watts
		SurgeProtection:    true,
		ChildLock:          false,
		AutoShutoff:        false,
		ShutoffDelay:       30,
	}
}

func (sp *SmartPlug) Start(ctx context.Context) error {
	if err := sp.BaseDevice.Start(ctx); err != nil {
		return err
	}

	go sp.runPowerMonitoring(ctx)
	go sp.runScheduleManager(ctx)
	go sp.runSafetyMonitoring(ctx)
	go sp.runEnergyTracking(ctx)

	// sp.Logger.Info("Smart plug started successfully")
	return nil
}

func (sp *SmartPlug) runPowerMonitoring(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			sp.updatePowerMeasurements()
		}
	}
}

func (sp *SmartPlug) runScheduleManager(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			sp.checkSchedules()
		}
	}
}

func (sp *SmartPlug) runSafetyMonitoring(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			sp.checkSafetyLimits()
		}
	}
}

func (sp *SmartPlug) runEnergyTracking(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			sp.updateEnergyUsage()
		}
	}
}

func (sp *SmartPlug) updatePowerMeasurements() {
	sp.mu.Lock()
	defer sp.mu.Unlock()

	if !sp.power {
		sp.powerConsumption = 0.0
		sp.current = 0.0
		sp.energyUsage.InstantPower = 0.0
		return
	}

	// Simulate realistic power consumption based on connected device
	basePower := sp.connectedDevice.RatedPower

	switch sp.connectedDevice.DeviceStatus {
	case "on":
		variation := sp.getRandomFloat(0.8, 1.2)
		sp.powerConsumption = basePower * variation
	case "standby":
		sp.powerConsumption = basePower * sp.getRandomFloat(0.01, 0.05)
	case "off":
		sp.powerConsumption = 0.0
	}

	// Add some random variation to simulate real-world fluctuations
	sp.powerConsumption += sp.getRandomFloat(-basePower*0.1, basePower*0.1)

	if sp.powerConsumption < 0 {
		sp.powerConsumption = 0
	}

	// Calculate current (P = V * I, so I = P / V)
	sp.current = sp.powerConsumption / sp.voltage
	sp.energyUsage.InstantPower = sp.powerConsumption

	// Update peak power
	if sp.powerConsumption > sp.energyUsage.PeakPower {
		sp.energyUsage.PeakPower = sp.powerConsumption
	}

	// Simulate device state changes
	if sp.getRandomFloat(0, 100) < 5 {
		states := []string{"on", "off", "standby"}
		sp.connectedDevice.DeviceStatus = states[sp.getRandomInt(0, len(states))]
	}
}

func (sp *SmartPlug) checkSchedules() {
	sp.mu.Lock()
	defer sp.mu.Unlock()

	now := time.Now()
	currentTime := now.Format("15:04")
	currentDay := now.Weekday().String()

	for _, schedule := range sp.schedule {
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
		if sp.isTimeInRange(currentTime, schedule.StartTime, schedule.EndTime) {
			newState := schedule.Action == "on"
			if sp.power != newState {
				sp.power = newState

				event := base.Event{
					EventType: "schedule_activated",
					Severity:  "info",
					Message:   fmt.Sprintf("Schedule '%s' activated - plug turned %s", schedule.Name, schedule.Action),
					Extra: map[string]interface{}{
						"schedule_name": schedule.Name,
						"action":        schedule.Action,
						"power_state":   sp.power,
					},
				}
				sp.PublishEvent(event)
			}
			break
		}
	}
}

func (sp *SmartPlug) isTimeInRange(current, start, end string) bool {
	currentTime, _ := time.Parse("15:04", current)
	startTime, _ := time.Parse("15:04", start)
	endTime, _ := time.Parse("15:04", end)

	if endTime.Before(startTime) {
		// Schedule crosses midnight
		return currentTime.After(startTime) || currentTime.Before(endTime)
	}

	return currentTime.After(startTime) && currentTime.Before(endTime)
}

func (sp *SmartPlug) checkSafetyLimits() {
	sp.mu.Lock()
	defer sp.mu.Unlock()

	if !sp.protections.OverloadProtection {
		return
	}

	// Check current overload
	if sp.current > sp.protections.MaxCurrent {
		sp.handleOverload("current", sp.current, sp.protections.MaxCurrent)
		return
	}

	// Check power overload
	if sp.powerConsumption > sp.protections.MaxPower {
		sp.handleOverload("power", sp.powerConsumption, sp.protections.MaxPower)
		return
	}

	// Simulate random surge events (very rare)
	if sp.getRandomFloat(0, 100) < 0.1 {
		sp.handleSurgeEvent()
	}
}

func (sp *SmartPlug) handleOverload(overloadType string, currentValue, limit float64) {
	now := time.Now()
	sp.power = false
	sp.protections.LastOverload = &now

	event := base.Event{
		EventType: "overload_protection",
		Severity:  "critical",
		Message:   fmt.Sprintf("%s overload detected - plug automatically turned off", overloadType),
		Extra: map[string]interface{}{
			"overload_type":    overloadType,
			"current_value":    currentValue,
			"limit":            limit,
			"auto_shutoff":     true,
			"connected_device": sp.connectedDevice.Name,
		},
	}
	sp.PublishEvent(event)
}

func (sp *SmartPlug) handleSurgeEvent() {
	if !sp.protections.SurgeProtection {
		return
	}

	event := base.Event{
		EventType: "surge_detected",
		Severity:  "warning",
		Message:   "Power surge detected and suppressed",

		Extra: map[string]interface{}{
			"surge_protection": sp.protections.SurgeProtection,
			"connected_device": sp.connectedDevice.Name,
		},
	}
	sp.PublishEvent(event)
}

func (sp *SmartPlug) updateEnergyUsage() {
	sp.mu.Lock()
	defer sp.mu.Unlock()

	now := time.Now()
	timeDiff := now.Sub(sp.energyUsage.LastReading).Minutes()

	if timeDiff > 0 {
		// Energy = Power × Time (in watt-hours)
		energyIncrement := sp.powerConsumption * (timeDiff / 60.0)

		sp.energyUsage.DailyEnergy += energyIncrement
		sp.energyUsage.WeeklyEnergy += energyIncrement
		sp.energyUsage.MonthlyEnergy += energyIncrement

		// Calculate cost (assuming $0.12 per kWh)
		sp.energyUsage.DailyCost = (sp.energyUsage.DailyEnergy / 1000.0) * 0.12

		// Update average power
		if sp.energyUsage.DailyEnergy > 0 {
			hoursInDay := float64(now.Sub(now.Truncate(24 * time.Hour)).Hours())
			if hoursInDay > 0 {
				sp.energyUsage.AveragePower = sp.energyUsage.DailyEnergy / hoursInDay
			}
		}
	}

	sp.energyUsage.LastReading = now
}

func (sp *SmartPlug) GenerateStatePayload() base.StatePayload {
	sp.mu.RLock()
	defer sp.mu.RUnlock()

	baseState := sp.BaseDevice.GenerateStatePayload()

	if baseState.Extra == nil {
		baseState.Extra = make(map[string]interface{})
	}

	baseState.Extra["plug"] = map[string]interface{}{
		"power":             sp.power,
		"power_consumption": sp.powerConsumption,
		"voltage":           sp.voltage,
		"current":           sp.current,
	}

	baseState.Extra["connected_device"] = sp.connectedDevice
	baseState.Extra["energy_usage"] = sp.energyUsage
	baseState.Extra["safety_protections"] = sp.protections
	baseState.Extra["schedule_count"] = len(sp.schedule)

	return baseState
}

func (sp *SmartPlug) GenerateTelemetryData() map[string]base.TelemetryPayload {
	sp.mu.RLock()
	defer sp.mu.RUnlock()

	telemetry := sp.BaseDevice.GenerateTelemetryData()

	telemetry["power_consumption"] = base.TelemetryPayload{
		"timestamp": time.Now(),
		"value":     sp.powerConsumption,
		"unit":      "watts",
		"tags": map[string]string{
			"metric": "power_consumption",
		},
	}

	telemetry["current"] = base.TelemetryPayload{
		"timestamp": time.Now(),
		"value":     sp.current,
		"unit":      "amps",
		"tags": map[string]string{
			"metric": "current",
		},
	}

	telemetry["voltage"] = base.TelemetryPayload{
		"timestamp": time.Now(),
		"value":     sp.voltage,
		"unit":      "volts",
		"tags": map[string]string{
			"metric": "voltage",
		},
	}

	telemetry["daily_energy"] = base.TelemetryPayload{
		"timestamp": time.Now(),
		"value":     sp.energyUsage.DailyEnergy,
		"unit":      "wh",
		"tags": map[string]string{
			"metric": "daily_energy",
		},
	}

	telemetry["daily_cost"] = base.TelemetryPayload{
		"timestamp": time.Now(),
		"value":     sp.energyUsage.DailyCost,
		"unit":      "usd",
		"tags": map[string]string{
			"metric": "daily_cost",
		},
	}

	return telemetry
}

func (sp *SmartPlug) HandleCommand(cmd base.Command) error {
	// sp.Logger.Infof("Handling command: %s", cmd.Action)

	switch cmd.Action {
	case "power_on":
		return sp.powerOn(cmd)
	case "power_off":
		return sp.powerOff(cmd)
	case "toggle_power":
		return sp.togglePower(cmd)
	case "create_schedule":
		return sp.createSchedule(cmd)
	case "update_schedule":
		return sp.updateSchedule(cmd)
	case "delete_schedule":
		return sp.deleteSchedule(cmd)
	case "get_energy_usage":
		return sp.getEnergyUsage(cmd)
	case "reset_energy_usage":
		return sp.resetEnergyUsage(cmd)
	case "set_safety_limits":
		return sp.setSafetyLimits(cmd)
	default:
		return sp.BaseDevice.HandleCommand(cmd)
	}
}

func (sp *SmartPlug) powerOn(cmd base.Command) error {
	if sp.protections.ChildLock {
		return fmt.Errorf("child lock is enabled")
	}

	sp.mu.Lock()
	sp.power = true
	sp.mu.Unlock()

	response := base.CommandResponse{
		CommandID: cmd.ID,
		Status:    "success",
		Data:      "Plug turned on",
	}

	return sp.PublishCommandResponse(response)
}

func (sp *SmartPlug) powerOff(cmd base.Command) error {
	sp.mu.Lock()
	sp.power = false
	sp.mu.Unlock()

	response := base.CommandResponse{
		CommandID: cmd.ID,
		Status:    "success",
		Data:      "Plug turned off",
	}

	return sp.PublishCommandResponse(response)
}

func (sp *SmartPlug) togglePower(cmd base.Command) error {
	if sp.protections.ChildLock && !sp.power {
		return fmt.Errorf("child lock is enabled")
	}

	sp.mu.Lock()
	sp.power = !sp.power
	state := sp.power
	sp.mu.Unlock()

	response := base.CommandResponse{
		CommandID: cmd.ID,
		Status:    "success",
		Data:      fmt.Sprintf("Plug turned %s", map[bool]string{true: "on", false: "off"}[state]),
	}

	return sp.PublishCommandResponse(response)
}

func (sp *SmartPlug) createSchedule(cmd base.Command) error {
	var schedule PlugSchedule
	if err := json.Unmarshal([]byte(cmd.Payload), &schedule); err != nil {
		return fmt.Errorf("invalid schedule data: %v", err)
	}

	sp.mu.Lock()
	sp.schedule = append(sp.schedule, schedule)
	sp.mu.Unlock()

	response := base.CommandResponse{
		CommandID: cmd.ID,
		Status:    "success",
		Data:      fmt.Sprintf("Schedule '%s' created", schedule.Name),
	}

	return sp.PublishCommandResponse(response)
}

func (sp *SmartPlug) updateSchedule(cmd base.Command) error {
	var scheduleUpdate struct {
		Name     string       `json:"name"`
		Schedule PlugSchedule `json:"schedule"`
	}

	if err := json.Unmarshal([]byte(cmd.Payload), &scheduleUpdate); err != nil {
		return fmt.Errorf("invalid schedule update: %v", err)
	}

	sp.mu.Lock()
	defer sp.mu.Unlock()

	for i, schedule := range sp.schedule {
		if schedule.Name == scheduleUpdate.Name {
			sp.schedule[i] = scheduleUpdate.Schedule
			response := base.CommandResponse{
				CommandID: cmd.ID,
				Status:    "success",
				Data:      fmt.Sprintf("Schedule '%s' updated", scheduleUpdate.Name),
			}
			return sp.PublishCommandResponse(response)
		}
	}

	return fmt.Errorf("schedule '%s' not found", scheduleUpdate.Name)
}

func (sp *SmartPlug) deleteSchedule(cmd base.Command) error {
	var deleteCmd struct {
		Name string `json:"name"`
	}

	if err := json.Unmarshal([]byte(cmd.Payload), &deleteCmd); err != nil {
		return fmt.Errorf("invalid delete request: %v", err)
	}

	sp.mu.Lock()
	defer sp.mu.Unlock()

	for i, schedule := range sp.schedule {
		if schedule.Name == deleteCmd.Name {
			sp.schedule = append(sp.schedule[:i], sp.schedule[i+1:]...)
			response := base.CommandResponse{
				CommandID: cmd.ID,
				Status:    "success",
				Data:      fmt.Sprintf("Schedule '%s' deleted", deleteCmd.Name),
			}
			return sp.PublishCommandResponse(response)
		}
	}

	return fmt.Errorf("schedule '%s' not found", deleteCmd.Name)
}

func (sp *SmartPlug) getEnergyUsage(cmd base.Command) error {
	sp.mu.RLock()
	energyData, _ := json.Marshal(sp.energyUsage)
	sp.mu.RUnlock()

	response := base.CommandResponse{
		CommandID: cmd.ID,
		Status:    "success",
		Data:      string(energyData),
	}

	return sp.PublishCommandResponse(response)
}

func (sp *SmartPlug) resetEnergyUsage(cmd base.Command) error {
	sp.mu.Lock()
	sp.initializeEnergyUsage()
	sp.mu.Unlock()

	response := base.CommandResponse{
		CommandID: cmd.ID,
		Status:    "success",
		Data:      "Energy usage counters reset",
	}

	return sp.PublishCommandResponse(response)
}

func (sp *SmartPlug) setSafetyLimits(cmd base.Command) error {
	var limits struct {
		MaxCurrent  *float64 `json:"max_current,omitempty"`
		MaxPower    *float64 `json:"max_power,omitempty"`
		ChildLock   *bool    `json:"child_lock,omitempty"`
		AutoShutoff *bool    `json:"auto_shutoff,omitempty"`
	}

	if err := json.Unmarshal([]byte(cmd.Payload), &limits); err != nil {
		return fmt.Errorf("invalid safety limits: %v", err)
	}

	sp.mu.Lock()
	defer sp.mu.Unlock()

	if limits.MaxCurrent != nil {
		sp.protections.MaxCurrent = *limits.MaxCurrent
	}
	if limits.MaxPower != nil {
		sp.protections.MaxPower = *limits.MaxPower
	}
	if limits.ChildLock != nil {
		sp.protections.ChildLock = *limits.ChildLock
	}
	if limits.AutoShutoff != nil {
		sp.protections.AutoShutoff = *limits.AutoShutoff
	}

	response := base.CommandResponse{
		CommandID: cmd.ID,
		Status:    "success",
		Data:      "Safety limits updated",
	}

	return sp.PublishCommandResponse(response)
}
