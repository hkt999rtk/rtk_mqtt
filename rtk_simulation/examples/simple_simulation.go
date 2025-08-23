package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"rtk_simulation/pkg/config"
	"rtk_simulation/pkg/devices"
	"rtk_simulation/pkg/devices/base"
	"rtk_simulation/pkg/scenarios"

	"github.com/sirupsen/logrus"
)

func main() {
	// Setup logger
	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)

	// Create simulation config
	cfg := &config.SimulationConfig{
		Devices: config.DeviceConfigs{
			IoTDevices: []config.IoTDeviceConfig{
				{
					DeviceConfig: config.DeviceConfig{
						ID:       "bulb_001",
						Type:     "smart_bulb",
						Tenant:   "demo",
						Site:     "home",
						Location: &config.Location{Room: "bedroom"},
					},
				},
				{
					DeviceConfig: config.DeviceConfig{
						ID:       "ac_001",
						Type:     "air_conditioner",
						Tenant:   "demo",
						Site:     "home",
						Location: &config.Location{Room: "living_room"},
					},
				},
			},
			NetworkDevices: []config.NetworkDeviceConfig{
				{
					DeviceConfig: config.DeviceConfig{
						ID:       "router_main",
						Type:     "router",
						Tenant:   "demo",
						Site:     "home",
						Location: &config.Location{Room: "living_room"},
					},
				},
			},
		},
	}

	// Create device manager
	deviceManager := devices.NewDeviceManager(logger)

	// Create devices
	createdDevices := make(map[string]base.Device)
	allDevices := []config.DeviceConfig{}
	for _, netDev := range cfg.Devices.NetworkDevices {
		allDevices = append(allDevices, netDev.DeviceConfig)
	}
	for _, iotDev := range cfg.Devices.IoTDevices {
		allDevices = append(allDevices, iotDev.DeviceConfig)
	}

	for _, deviceCfg := range allDevices {
		device, err := deviceManager.CreateDevice(deviceCfg)
		if err != nil {
			log.Printf("Failed to create device %s: %v", deviceCfg.ID, err)
			continue
		}
		createdDevices[deviceCfg.ID] = device
		log.Printf("Created device: %s (%s)", deviceCfg.ID, deviceCfg.Type)
	}

	// Create scenario managers
	automationManager := scenarios.NewAutomationManager(nil)
	dailyRoutineManager := scenarios.NewDailyRoutineManager(nil)
	scriptEngine := scenarios.NewScriptEngine(nil)
	eventBus := scenarios.NewEventBus(1000)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Start all managers
	if err := automationManager.Start(ctx); err != nil {
		log.Fatal("Failed to start automation manager:", err)
	}
	if err := dailyRoutineManager.Start(ctx); err != nil {
		log.Fatal("Failed to start daily routine manager:", err)
	}
	if err := scriptEngine.Start(ctx); err != nil {
		log.Fatal("Failed to start script engine:", err)
	}
	if err := eventBus.Start(ctx); err != nil {
		log.Fatal("Failed to start event bus:", err)
	}

	// Register devices with managers
	for id, device := range createdDevices {
		automationManager.RegisterDevice(id, device)
		dailyRoutineManager.RegisterDevice(id, device)
		scriptEngine.RegisterDevice(id, device)
		log.Printf("Registered device %s with all managers", id)
	}

	// Start devices
	for id, device := range createdDevices {
		if err := device.Start(ctx); err != nil {
			log.Printf("Failed to start device %s: %v", id, err)
		} else {
			log.Printf("Started device: %s", id)
		}
	}

	// Create and execute a simple script
	script := &scenarios.Script{
		ID:       "demo_script",
		Name:     "Demo Script",
		Language: "simple",
		Steps: []scenarios.ScriptStep{
			{
				ID:   "turn_on_lights",
				Type: "action",
				Action: &scenarios.ScriptAction{
					Type:   "device",
					Target: "bulb_001",
					Method: "turn_on",
					Parameters: map[string]interface{}{
						"brightness": 80,
					},
				},
			},
			{
				ID:    "wait",
				Type:  "action",
				Delay: 2 * time.Second,
				Action: &scenarios.ScriptAction{
					Type: "wait",
					Parameters: map[string]interface{}{
						"duration": 2,
					},
				},
			},
			{
				ID:   "adjust_ac",
				Type: "action",
				Action: &scenarios.ScriptAction{
					Type:   "device",
					Target: "ac_001",
					Method: "set_temperature",
					Parameters: map[string]interface{}{
						"temperature": 24,
					},
				},
			},
		},
		Enabled: true,
	}

	// Load and execute script
	if err := scriptEngine.LoadScript(script); err != nil {
		log.Printf("Failed to load script: %v", err)
	} else {
		log.Println("Script loaded successfully")

		if execution, err := scriptEngine.ExecuteScript("demo_script", nil); err != nil {
			log.Printf("Failed to execute script: %v", err)
		} else {
			log.Printf("Script execution started: %s", execution.ExecutionID)
		}
	}

	// Trigger morning routine
	if err := dailyRoutineManager.ManualTriggerRoutine("morning_routine"); err != nil {
		log.Printf("Failed to trigger morning routine: %v", err)
	} else {
		log.Println("Morning routine triggered")
	}

	// Run simulation for a short time
	log.Println("Running simulation for 10 seconds...")
	time.Sleep(10 * time.Second)

	// Get statistics
	fmt.Println("\n=== Simulation Statistics ===")

	// Automation stats
	autoStats := automationManager.GetStatistics()
	fmt.Printf("Automation - Total Rules: %v, Active Rules: %v\n",
		autoStats["total_rules"], autoStats["active_rules"])

	// Routine stats
	routineStats := dailyRoutineManager.GetStatistics()
	fmt.Printf("Routines - Total: %v, Active: %v, Mode: %v\n",
		routineStats["total_routines"], routineStats["active_routines"],
		routineStats["current_mode"])

	// Script stats
	scriptStats := scriptEngine.GetStatistics()
	fmt.Printf("Scripts - Total: %v, Running: %v\n",
		scriptStats["total_scripts"], scriptStats["running_scripts"])

	// Generate device states
	fmt.Println("\n=== Device States ===")
	for id, device := range createdDevices {
		state := device.GenerateStatePayload()
		fmt.Printf("%s: Health=%s, CPU=%.1f%%, Memory=%.1f%%\n",
			id, state.Health, state.CPUUsage, state.MemoryUsage)
	}

	// Stop all devices
	for id, device := range createdDevices {
		if err := device.Stop(); err != nil {
			log.Printf("Failed to stop device %s: %v", id, err)
		}
	}

	// Stop all managers
	automationManager.Stop()
	dailyRoutineManager.Stop()
	scriptEngine.Stop()
	eventBus.Stop()

	log.Println("Simulation completed successfully!")
}
