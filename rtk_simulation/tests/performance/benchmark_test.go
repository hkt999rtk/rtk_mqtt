package performance

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"rtk_simulation/pkg/config"
	"rtk_simulation/pkg/devices"
	"rtk_simulation/pkg/devices/base"
	"rtk_simulation/pkg/devices/iot"
	devicenetwork "rtk_simulation/pkg/devices/network"
	"rtk_simulation/pkg/network"
	"rtk_simulation/pkg/scenarios"
)

func BenchmarkDeviceCreation(b *testing.B) {
	b.Run("SmartBulb", func(b *testing.B) {
		config := base.DeviceConfig{
			ID:   "bulb_bench",
			Type: "smart_bulb",
		}
		mqttConfig := base.MQTTConfig{}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = iot.NewSmartBulb(config, mqttConfig)
		}
	})

	b.Run("AirConditioner", func(b *testing.B) {
		config := base.DeviceConfig{
			ID:   "ac_bench",
			Type: "air_conditioner",
		}
		mqttConfig := base.MQTTConfig{}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = iot.NewAirConditioner(config, mqttConfig)
		}
	})

	b.Run("Router", func(b *testing.B) {
		config := base.DeviceConfig{
			ID:   "router_bench",
			Type: "router",
		}
		mqttConfig := base.MQTTConfig{}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = devicenetwork.NewRouter(config, mqttConfig)
		}
	})

	b.Run("DeviceManager", func(b *testing.B) {
		logger := logrus.New()
		logger.SetLevel(logrus.WarnLevel)
		manager := devices.NewDeviceManager(logger)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			deviceConfig := config.DeviceConfig{
				ID:   fmt.Sprintf("device_%d", i),
				Type: "smart_bulb",
			}
			_, _ = manager.CreateDevice(deviceConfig)
		}
	})
}

func BenchmarkDeviceOperations(b *testing.B) {
	b.Run("StateGeneration", func(b *testing.B) {
		config := base.DeviceConfig{ID: "bench"}
		bulb, _ := iot.NewSmartBulb(config, base.MQTTConfig{})

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = bulb.GenerateStatePayload()
		}
	})

	b.Run("TelemetryGeneration", func(b *testing.B) {
		config := base.DeviceConfig{ID: "bench"}
		bulb, _ := iot.NewSmartBulb(config, base.MQTTConfig{})

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = bulb.GenerateTelemetryData()
		}
	})

	b.Run("EventGeneration", func(b *testing.B) {
		config := base.DeviceConfig{ID: "bench"}
		bulb, _ := iot.NewSmartBulb(config, base.MQTTConfig{})

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = bulb.GenerateEvents()
		}
	})

	b.Run("CommandHandling", func(b *testing.B) {
		config := base.DeviceConfig{ID: "bench"}
		bulb, _ := iot.NewSmartBulb(config, base.MQTTConfig{})

		cmd := base.Command{
			Type: "turn_on",
			Parameters: map[string]interface{}{
				"brightness": 80,
			},
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = bulb.HandleCommand(cmd)
		}
	})
}

func BenchmarkEventBus(b *testing.B) {
	b.Run("PublishEvent", func(b *testing.B) {
		eventBus := scenarios.NewEventBus(10000)
		ctx := context.Background()
		eventBus.Start(ctx)
		defer eventBus.Stop()

		event := scenarios.Event{
			Type:   "test",
			Source: "bench",
			Name:   "benchmark",
			Data:   "test_data",
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = eventBus.Publish(event)
		}
	})

	b.Run("EventProcessing", func(b *testing.B) {
		eventBus := scenarios.NewEventBus(10000)
		ctx := context.Background()
		eventBus.Start(ctx)
		defer eventBus.Stop()

		// Add handler
		handler := scenarios.EventHandler{
			ID:   "bench_handler",
			Name: "Benchmark Handler",
			Handler: func(e scenarios.Event) error {
				// Minimal processing
				return nil
			},
		}
		eventBus.Subscribe("*", handler)

		event := scenarios.Event{
			Type:   "test",
			Source: "bench",
			Name:   "benchmark",
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			eventBus.Publish(event)
		}
	})

	b.Run("ConcurrentPublish", func(b *testing.B) {
		eventBus := scenarios.NewEventBus(10000)
		ctx := context.Background()
		eventBus.Start(ctx)
		defer eventBus.Stop()

		b.RunParallel(func(pb *testing.PB) {
			event := scenarios.Event{
				Type:   "test",
				Source: "bench",
				Name:   "concurrent",
			}
			for pb.Next() {
				eventBus.Publish(event)
			}
		})
	})
}

func BenchmarkScriptEngine(b *testing.B) {
	b.Run("ScriptExecution", func(b *testing.B) {
		engine := scenarios.NewScriptEngine(nil)
		ctx := context.Background()
		engine.Start(ctx)
		defer engine.Stop()

		script := &scenarios.Script{
			ID:       "bench_script",
			Language: "simple",
			Steps: []scenarios.ScriptStep{
				{
					ID:   "step1",
					Type: "action",
					Action: &scenarios.ScriptAction{
						Type: "log",
						Parameters: map[string]interface{}{
							"message": "test",
						},
					},
				},
			},
			Enabled: true,
		}
		engine.LoadScript(script)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = engine.ExecuteScript("bench_script", nil)
		}
	})

	b.Run("ParallelScriptExecution", func(b *testing.B) {
		engine := scenarios.NewScriptEngine(&scenarios.ScriptConfig{
			MaxConcurrentScripts: 100,
		})
		ctx := context.Background()
		engine.Start(ctx)
		defer engine.Stop()

		script := &scenarios.Script{
			ID:       "parallel_script",
			Language: "simple",
			Steps: []scenarios.ScriptStep{
				{
					ID:   "parallel_steps",
					Type: "parallel",
					Steps: []scenarios.ScriptStep{
						{
							ID:   "step1",
							Type: "action",
							Action: &scenarios.ScriptAction{
								Type: "wait",
								Parameters: map[string]interface{}{
									"duration": 0.001,
								},
							},
						},
						{
							ID:   "step2",
							Type: "action",
							Action: &scenarios.ScriptAction{
								Type: "wait",
								Parameters: map[string]interface{}{
									"duration": 0.001,
								},
							},
						},
					},
				},
			},
			Enabled: true,
		}
		engine.LoadScript(script)

		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				engine.ExecuteScript("parallel_script", nil)
			}
		})
	})
}

func BenchmarkAutomation(b *testing.B) {
	b.Run("RuleEvaluation", func(b *testing.B) {
		manager := scenarios.NewAutomationManager(nil)
		ctx := context.Background()
		manager.Start(ctx)
		defer manager.Stop()

		// Register mock devices
		for i := 0; i < 10; i++ {
			device := &MockDevice{
				id:         fmt.Sprintf("device_%d", i),
				deviceType: "smart_bulb",
			}
			manager.RegisterDevice(device.id, device)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// This triggers rule evaluation internally
			event := scenarios.Event{
				Type:   "sensor",
				Source: "motion_sensor",
				Name:   "motion_detected",
				Data:   true,
			}
			manager.PublishEvent(event)
		}
	})

	b.Run("SceneActivation", func(b *testing.B) {
		manager := scenarios.NewAutomationManager(nil)
		ctx := context.Background()
		manager.Start(ctx)
		defer manager.Stop()

		// Register devices
		for i := 0; i < 5; i++ {
			device := &MockDevice{
				id:         fmt.Sprintf("bulb_%d", i),
				deviceType: "smart_bulb",
			}
			manager.RegisterDevice(device.id, device)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// _ = manager.ActivateScene("movie_night")
		}
	})
}

func BenchmarkConcurrentDevices(b *testing.B) {
	b.Run("100Devices", func(b *testing.B) {
		benchmarkConcurrentDevices(b, 100)
	})

	b.Run("500Devices", func(b *testing.B) {
		benchmarkConcurrentDevices(b, 500)
	})

	b.Run("1000Devices", func(b *testing.B) {
		benchmarkConcurrentDevices(b, 1000)
	})
}

func benchmarkConcurrentDevices(b *testing.B, numDevices int) {
	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel)
	manager := devices.NewDeviceManager(logger)

	// Create devices
	for i := 0; i < numDevices; i++ {
		deviceConfig := config.DeviceConfig{
			ID:   fmt.Sprintf("device_%d", i),
			Type: getDeviceType(i),
		}
		manager.CreateDevice(deviceConfig)
	}

	allDevices := manager.ListDevices()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var wg sync.WaitGroup
		wg.Add(len(allDevices))

		for _, device := range allDevices {
			go func(d base.Device) {
				defer wg.Done()
				_ = d.GenerateStatePayload()
				_ = d.GenerateTelemetryData()
				_ = d.GenerateEvents()
			}(device)
		}

		wg.Wait()
	}
}

func getDeviceType(index int) string {
	types := []string{"smart_bulb", "air_conditioner", "router", "security_camera", "smart_plug"}
	return types[index%len(types)]
}

func BenchmarkMemoryUsage(b *testing.B) {
	b.Run("DeviceMemory", func(b *testing.B) {
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		allocBefore := m.Alloc

		// Create many devices
		logger := logrus.New()
		logger.SetLevel(logrus.WarnLevel)
		manager := devices.NewDeviceManager(logger)
		for i := 0; i < 1000; i++ {
			deviceConfig := config.DeviceConfig{
				ID:   fmt.Sprintf("mem_device_%d", i),
				Type: "smart_bulb",
			}
			manager.CreateDevice(deviceConfig)
		}

		runtime.ReadMemStats(&m)
		allocAfter := m.Alloc

		memPerDevice := (allocAfter - allocBefore) / 1000
		b.Logf("Memory per device: %d bytes", memPerDevice)
	})

	b.Run("EventBusMemory", func(b *testing.B) {
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		allocBefore := m.Alloc

		eventBus := scenarios.NewEventBus(10000)
		ctx := context.Background()
		eventBus.Start(ctx)

		// Publish many events
		for i := 0; i < 10000; i++ {
			event := scenarios.Event{
				Type:   "test",
				Source: "mem_test",
				Name:   fmt.Sprintf("event_%d", i),
				Data:   fmt.Sprintf("data_%d", i),
			}
			eventBus.Publish(event)
		}

		runtime.ReadMemStats(&m)
		allocAfter := m.Alloc

		totalMem := allocAfter - allocBefore
		b.Logf("Total memory for 10000 events: %d bytes", totalMem)

		eventBus.Stop()
	})
}

func BenchmarkTopologyOperations(b *testing.B) {
	b.Run("PathFinding", func(b *testing.B) {
		manager := network.NewTopologyManager()

		// Create mesh topology
		for i := 0; i < 20; i++ {
			device := &MockDevice{
				id:         fmt.Sprintf("node_%d", i),
				deviceType: "mesh_node",
			}
			manager.AddDevice(device)
		}

		// Add connections (mesh-like)
		for i := 0; i < 20; i++ {
			for j := i + 1; j < min(i+4, 20); j++ {
				manager.AddConnection(
					fmt.Sprintf("node_%d", i),
					fmt.Sprintf("node_%d", j),
					"wifi",
				)
			}
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = manager.GetDeviceConnections("node_0")
		}
	})

	b.Run("NeighborDiscovery", func(b *testing.B) {
		manager := network.NewTopologyManager()

		// Create star topology
		central := &MockDevice{id: "central", deviceType: "router"}
		manager.AddDevice(central)

		for i := 0; i < 50; i++ {
			device := &MockDevice{
				id:         fmt.Sprintf("device_%d", i),
				deviceType: "smart_device",
			}
			manager.AddDevice(device)
			manager.AddConnection("central", device.id, "wifi")
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = manager.GetDeviceConnections("central")
		}
	})
}

func BenchmarkScenarioIntegration(b *testing.B) {
	b.Run("FullScenario", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			// Create all managers
			automationManager := scenarios.NewAutomationManager(nil)
			dailyRoutineManager := scenarios.NewDailyRoutineManager(nil)
			scriptEngine := scenarios.NewScriptEngine(nil)
			eventBus := scenarios.NewEventBus(1000)

			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)

			// Start all
			automationManager.Start(ctx)
			dailyRoutineManager.Start(ctx)
			scriptEngine.Start(ctx)
			eventBus.Start(ctx)

			// Create and register devices
			for j := 0; j < 10; j++ {
				device := &MockDevice{
					id:         fmt.Sprintf("device_%d", j),
					deviceType: "smart_bulb",
				}
				automationManager.RegisterDevice(device.id, device)
				dailyRoutineManager.RegisterDevice(device.id, device)
				scriptEngine.RegisterDevice(device.id, device)
			}

			// Trigger various operations
			// automationManager.ActivateScene("movie_night")
			dailyRoutineManager.ManualTriggerRoutine("morning_routine")

			// Publish events
			for j := 0; j < 10; j++ {
				event := scenarios.Event{
					Type:   "test",
					Source: "bench",
					Name:   fmt.Sprintf("event_%d", j),
				}
				eventBus.Publish(event)
			}

			// Clean up
			cancel()
			automationManager.Stop()
			dailyRoutineManager.Stop()
			scriptEngine.Stop()
			eventBus.Stop()
		}
	})
}

// MockDevice for benchmarking
type MockDevice struct {
	id         string
	deviceType string
}

func (m *MockDevice) GetDeviceID() string                     { return m.id }
func (m *MockDevice) GetDeviceType() string                   { return m.deviceType }
func (m *MockDevice) GetHealth() string                       { return "healthy" }
func (m *MockDevice) SetHealth(health string)                 {}
func (m *MockDevice) Start(ctx context.Context) error         { return nil }
func (m *MockDevice) Stop() error                             { return nil }
func (m *MockDevice) GenerateStatePayload() base.StatePayload { return base.StatePayload{} }
func (m *MockDevice) GenerateTelemetryData() map[string]base.TelemetryPayload {
	return map[string]base.TelemetryPayload{}
}
func (m *MockDevice) GenerateEvents() []base.Event         { return []base.Event{} }
func (m *MockDevice) HandleCommand(cmd base.Command) error { return nil }
func (m *MockDevice) GetNetworkInfo() base.NetworkInfo     { return base.NetworkInfo{} }
func (m *MockDevice) UpdateCPUUsage(usage float64)         {}
func (m *MockDevice) UpdateMemoryUsage(usage float64)      {}
func (m *MockDevice) UpdateTemperature(temp float64)       {}
func (m *MockDevice) PublishEvent(event base.Event) error  { return nil }
func (m *MockDevice) GetIPAddress() string                 { return "192.168.1.1" }
func (m *MockDevice) GetMACAddress() string                { return "00:00:00:00:00:00" }
func (m *MockDevice) GetSite() string                      { return "test_site" }
func (m *MockDevice) GetTenant() string                    { return "test_tenant" }
func (m *MockDevice) GetUptime() time.Duration             { return time.Hour }
func (m *MockDevice) IsRunning() bool                      { return true }
func (m *MockDevice) UpdateStatus()                        {}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
