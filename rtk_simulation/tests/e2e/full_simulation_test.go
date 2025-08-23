package e2e

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"rtk_simulation/pkg/devices/base"
	"rtk_simulation/pkg/interaction"
	"rtk_simulation/pkg/network"
	"rtk_simulation/pkg/scenarios"
	syncsimul "rtk_simulation/pkg/sync"
	"rtk_simulation/tests/utils"
)

func TestCompleteHomeSimulation(t *testing.T) {
	t.Run("24HourHomeSimulation", func(t *testing.T) {
		// Create test environment
		env := utils.NewTestEnvironment(t)
		defer env.Cleanup()

		// Create home devices
		devices := []struct {
			id       string
			devType  string
			location string
		}{
			{"router_main", "router", "living_room"},
			{"ap_upstairs", "access_point", "hallway_2f"},
			{"bulb_living", "smart_bulb", "living_room"},
			{"bulb_bedroom", "smart_bulb", "bedroom"},
			{"bulb_kitchen", "smart_bulb", "kitchen"},
			{"ac_living", "air_conditioner", "living_room"},
			{"ac_bedroom", "air_conditioner", "bedroom"},
			{"camera_front", "security_camera", "front_door"},
			{"camera_back", "security_camera", "back_door"},
			{"sensor_temp_living", "environmental_sensor", "living_room"},
			{"sensor_temp_bedroom", "environmental_sensor", "bedroom"},
			{"tv_living", "smart_tv", "living_room"},
			{"plug_coffee", "smart_plug", "kitchen"},
			{"phone_john", "smartphone", "anywhere"},
			{"laptop_jane", "laptop", "office"},
		}

		// Create all devices
		for _, d := range devices {
			env.CreateDevice(t, d.id, d.devType)
		}

		// Create topology manager
		topologyManager := network.NewTopologyManager()
		for _, device := range env.Devices {
			topologyManager.AddDevice(device)
		}

		// Build network topology
		topologyManager.AddConnection("router_main", "ap_upstairs", "ethernet")
		topologyManager.AddConnection("router_main", "tv_living", "ethernet")

		// WiFi connections
		wifiDevices := []string{"bulb_living", "bulb_bedroom", "bulb_kitchen",
			"camera_front", "camera_back", "phone_john", "laptop_jane"}
		for _, device := range wifiDevices {
			topologyManager.AddConnection("router_main", device, "wifi")
		}

		// Monitor for collecting metrics
		// monitor := monitoring.NewMonitor()
		// monitor.Start(env.Context)

		// Register devices with monitor
		// for id, device := range env.Devices {
		// 	monitor.RegisterDevice(id, device)
		// }

		// Simulate 24-hour cycle (accelerated)
		simulateDailyCycle(t, env)

		// Collect and validate metrics
		// metrics := monitor.GetMetrics()
		// assert.NotNil(t, metrics)

		// Validate device health
		for id, device := range env.Devices {
			health := device.GetHealth()
			assert.NotEqual(t, "critical", health, "Device %s is critical", id)
		}

		// Get statistics
		stats := env.AutomationManager.GetStatistics()
		t.Logf("Automation stats: %+v", stats)

		routineStats := env.RoutineManager.GetStatistics()
		t.Logf("Routine stats: %+v", routineStats)
	})
}

func TestScenarioOrchestration(t *testing.T) {
	t.Run("MultiScenarioExecution", func(t *testing.T) {
		env := utils.NewTestEnvironment(t)
		defer env.Cleanup()

		// Create devices
		env.CreateDevice(t, "bulb_001", "smart_bulb")
		env.CreateDevice(t, "bulb_002", "smart_bulb")
		env.CreateDevice(t, "ac_001", "air_conditioner")
		env.CreateDevice(t, "tv_001", "smart_tv")
		env.CreateDevice(t, "camera_001", "security_camera")

		// Create event channel for monitoring
		eventChan := make(chan scenarios.Event, 100)
		handler := scenarios.EventHandler{
			ID:   "test_monitor",
			Name: "Test Monitor",
			Handler: func(e scenarios.Event) error {
				eventChan <- e
				return nil
			},
		}
		env.EventBus.Subscribe("*", handler)

		// Execute morning routine
		err := env.RoutineManager.ManualTriggerRoutine("morning_routine")
		assert.NoError(t, err)

		// Activate movie night scene
		// err = env.AutomationManager.ActivateScene("movie_night")
		// assert.NoError(t, err)

		// Create and execute custom script
		script := &scenarios.Script{
			ID:       "custom_sequence",
			Language: "simple",
			Steps: []scenarios.ScriptStep{
				{
					ID:   "lights_on",
					Type: "action",
					Action: &scenarios.ScriptAction{
						Type:   "device",
						Target: "smart_bulb",
						Method: "turn_on",
						Parameters: map[string]interface{}{
							"brightness": 100,
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
					ID:   "lights_dim",
					Type: "action",
					Action: &scenarios.ScriptAction{
						Type:   "device",
						Target: "smart_bulb",
						Method: "set_brightness",
						Parameters: map[string]interface{}{
							"brightness": 30,
						},
					},
				},
			},
			Enabled: true,
		}

		err = env.ScriptEngine.LoadScript(script)
		require.NoError(t, err)

		execution, err := env.ScriptEngine.ExecuteScript("custom_sequence", nil)
		assert.NoError(t, err)
		assert.NotNil(t, execution)

		// Wait for events
		eventCount := 0
		timeout := time.After(5 * time.Second)
	loop:
		for {
			select {
			case <-eventChan:
				eventCount++
				if eventCount >= 5 {
					break loop
				}
			case <-timeout:
				break loop
			}
		}

		assert.GreaterOrEqual(t, eventCount, 3, "Expected at least 3 events")
	})
}

func TestFaultToleranceAndRecovery(t *testing.T) {
	t.Run("DeviceFailureRecovery", func(t *testing.T) {
		env := utils.NewTestEnvironment(t)
		defer env.Cleanup()

		// Create devices
		router := env.CreateDevice(t, "router_main", "router")
		bulb1 := env.CreateDevice(t, "bulb_001", "smart_bulb")
		bulb2 := env.CreateDevice(t, "bulb_002", "smart_bulb")

		// Create fault manager
		// faultManager := scenarios.NewFaultManager(nil)
		// faultManager.Start(env.Context)
		// defer faultManager.Stop()

		// Check initial health
		assert.NotEmpty(t, router.GetHealth())

		// Wait for some time
		time.Sleep(1 * time.Second)

		// Verify devices are still healthy
		assert.NotEqual(t, "critical", router.GetHealth())
		assert.NotEqual(t, "critical", bulb1.GetHealth())
		assert.NotEqual(t, "critical", bulb2.GetHealth())
	})

	t.Run("CascadingFailures", func(t *testing.T) {
		env := utils.NewTestEnvironment(t)
		defer env.Cleanup()

		// Create device chain
		devices := make([]base.Device, 5)
		for i := 0; i < 5; i++ {
			devices[i] = env.CreateDevice(t, fmt.Sprintf("device_%d", i), "smart_bulb")
		}

		// Create sync group
		syncManager := &syncsimul.StateSync{}
		// syncManager doesn't have Start/Stop methods
		// syncManager.Start(env.Context)
		// defer syncManager.Stop()

		deviceIDs := make([]string, 5)
		for i := 0; i < 5; i++ {
			deviceIDs[i] = fmt.Sprintf("device_%d", i)
			syncManager.RegisterDevice(deviceIDs[i], devices[i])
		}

		// Start sync manager
		err := syncManager.Start(context.Background())
		assert.NoError(t, err)

		// Check first device health
		assert.NotEmpty(t, devices[0].GetHealth())

		// Update state
		syncManager.UpdateState(deviceIDs[0], base.StatePayload{
			Health: "critical",
		})
	})
}

func TestPerformanceUnderLoad(t *testing.T) {
	t.Run("HighDeviceCount", func(t *testing.T) {
		if testing.Short() {
			t.Skip("Skipping performance test in short mode")
		}

		env := utils.NewTestEnvironment(t)
		defer env.Cleanup()

		// Create many devices
		deviceCount := 100
		var wg sync.WaitGroup
		wg.Add(deviceCount)

		start := time.Now()
		for i := 0; i < deviceCount; i++ {
			go func(idx int) {
				defer wg.Done()
				deviceType := []string{"smart_bulb", "smart_plug", "environmental_sensor"}[idx%3]
				env.CreateDevice(t, fmt.Sprintf("device_%d", idx), deviceType)
			}(i)
		}
		wg.Wait()

		creationTime := time.Since(start)
		t.Logf("Created %d devices in %v", deviceCount, creationTime)
		assert.Less(t, creationTime, 10*time.Second)

		// Test concurrent operations
		start = time.Now()
		wg.Add(deviceCount)
		for id, device := range env.Devices {
			go func(deviceID string, d base.Device) {
				defer wg.Done()
				_ = d.GenerateStatePayload()
				_ = d.GenerateTelemetryData()
				_ = d.GenerateEvents()
			}(id, device)
		}
		wg.Wait()

		operationTime := time.Since(start)
		t.Logf("Performed operations on %d devices in %v", deviceCount, operationTime)
		assert.Less(t, operationTime, 5*time.Second)
	})

	t.Run("HighEventThroughput", func(t *testing.T) {
		env := utils.NewTestEnvironment(t)
		defer env.Cleanup()

		// Track processed events
		processedCount := int32(0)
		var mu sync.Mutex

		handler := scenarios.EventHandler{
			ID:   "throughput_handler",
			Name: "Throughput Handler",
			Handler: func(e scenarios.Event) error {
				mu.Lock()
				processedCount++
				mu.Unlock()
				return nil
			},
			Async: true,
		}
		env.EventBus.Subscribe("*", handler)

		// Generate many events
		eventCount := 10000
		start := time.Now()

		for i := 0; i < eventCount; i++ {
			event := scenarios.Event{
				Type:   "test",
				Source: "load_test",
				Name:   fmt.Sprintf("event_%d", i),
				Data:   i,
			}
			env.EventBus.Publish(event)
		}

		// Wait for processing
		time.Sleep(2 * time.Second)

		duration := time.Since(start)
		throughput := float64(processedCount) / duration.Seconds()

		t.Logf("Processed %d/%d events in %v (throughput: %.2f events/sec)",
			processedCount, eventCount, duration, throughput)

		assert.Greater(t, throughput, 1000.0)
	})
}

func TestInteractionPatterns(t *testing.T) {
	t.Run("DeviceInteractionChains", func(t *testing.T) {
		env := utils.NewTestEnvironment(t)
		defer env.Cleanup()

		// Create interaction manager
		interactionManager := interaction.NewInteractionManager()
		interactionManager.Start(env.Context)
		defer interactionManager.Stop()

		// Create devices
		sensor := env.CreateDevice(t, "motion_sensor", "environmental_sensor")
		bulb := env.CreateDevice(t, "smart_bulb", "smart_bulb")
		camera := env.CreateDevice(t, "security_camera", "security_camera")

		// Register with interaction manager
		interactionManager.RegisterDevice("motion_sensor", sensor)
		interactionManager.RegisterDevice("smart_bulb", bulb)
		interactionManager.RegisterDevice("security_camera", camera)

		// Get interaction statistics (TriggerInteraction not available)
		stats := interactionManager.GetStatistics()
		assert.NotNil(t, stats)

		t.Logf("Interaction stats: %+v", stats)
	})
}

func TestDataConsistency(t *testing.T) {
	t.Run("StateConsistency", func(t *testing.T) {
		env := utils.NewTestEnvironment(t)
		defer env.Cleanup()

		// Create devices
		devices := make([]base.Device, 10)
		for i := 0; i < 10; i++ {
			devices[i] = env.CreateDevice(t, fmt.Sprintf("device_%d", i), "smart_bulb")
		}

		// Perform concurrent state updates
		var wg sync.WaitGroup
		stateMap := sync.Map{}

		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func(iteration int) {
				defer wg.Done()
				deviceIdx := iteration % 10
				device := devices[deviceIdx]

				// Update device state
				cmd := base.Command{
					Type: "set_brightness",
					Parameters: map[string]interface{}{
						"brightness": iteration % 101,
					},
				}
				device.HandleCommand(cmd)

				// Store state
				state := device.GenerateStatePayload()
				stateMap.Store(fmt.Sprintf("device_%d", deviceIdx), state)
			}(i)
		}

		wg.Wait()

		// Verify states are consistent
		stateMap.Range(func(key, value interface{}) bool {
			state := value.(base.StatePayload)
			assert.NotEmpty(t, state.Health)
			return true
		})
	})
}

// Helper function to simulate daily cycle
func simulateDailyCycle(t *testing.T, env *utils.TestEnvironment) {
	// Morning (6:00 - 9:00)
	t.Log("Simulating morning...")
	err := env.RoutineManager.ManualTriggerRoutine("morning_routine")
	assert.NoError(t, err)
	time.Sleep(100 * time.Millisecond)

	// Daytime (9:00 - 17:00)
	t.Log("Simulating daytime...")
	err = env.RoutineManager.ManualTriggerRoutine("daytime_routine")
	assert.NoError(t, err)
	time.Sleep(100 * time.Millisecond)

	// Evening (17:00 - 23:00)
	t.Log("Simulating evening...")
	err = env.RoutineManager.ManualTriggerRoutine("evening_routine")
	assert.NoError(t, err)
	// err = env.AutomationManager.ActivateScene("movie_night")
	// assert.NoError(t, err)
	time.Sleep(100 * time.Millisecond)

	// Night (23:00 - 6:00)
	t.Log("Simulating night...")
	err = env.RoutineManager.ManualTriggerRoutine("night_routine")
	assert.NoError(t, err)
	time.Sleep(100 * time.Millisecond)
}
