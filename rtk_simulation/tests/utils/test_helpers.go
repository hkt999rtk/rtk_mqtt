package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
	"rtk_simulation/pkg/config"
	"rtk_simulation/pkg/devices"
	"rtk_simulation/pkg/devices/base"
	"rtk_simulation/pkg/scenarios"
	syncsimul "rtk_simulation/pkg/sync"
)

// TestEnvironment provides a complete test environment for simulations
type TestEnvironment struct {
	DeviceManager     *devices.DeviceManager
	AutomationManager *scenarios.AutomationManager
	RoutineManager    *scenarios.DailyRoutineManager
	ScriptEngine      *scenarios.ScriptEngine
	EventBus          *scenarios.EventBus
	SyncManager       *syncsimul.StateSync
	Context           context.Context
	Cancel            context.CancelFunc
	Devices           map[string]base.Device
	Logger            *logrus.Logger
	mu                sync.Mutex
}

// NewTestEnvironment creates a new test environment
func NewTestEnvironment(t *testing.T) *TestEnvironment {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel) // Reduce noise in tests

	deviceManager := devices.NewDeviceManager(logger)
	syncManager := &syncsimul.StateSync{}

	env := &TestEnvironment{
		DeviceManager:     deviceManager,
		AutomationManager: scenarios.NewAutomationManager(nil),
		RoutineManager:    scenarios.NewDailyRoutineManager(nil),
		ScriptEngine:      scenarios.NewScriptEngine(nil),
		EventBus:          scenarios.NewEventBus(1000),
		SyncManager:       syncManager,
		Context:           ctx,
		Cancel:            cancel,
		Devices:           make(map[string]base.Device),
		Logger:            logger,
	}

	// Start all managers
	require.NoError(t, env.AutomationManager.Start(ctx))
	require.NoError(t, env.RoutineManager.Start(ctx))
	require.NoError(t, env.ScriptEngine.Start(ctx))
	require.NoError(t, env.EventBus.Start(ctx))

	return env
}

// Cleanup stops all managers and cancels context
func (env *TestEnvironment) Cleanup() {
	for _, device := range env.Devices {
		device.Stop()
	}
	env.AutomationManager.Stop()
	env.RoutineManager.Stop()
	env.ScriptEngine.Stop()
	env.EventBus.Stop()
	// env.SyncManager.Stop() // StateSync doesn't have Stop method
	env.Cancel()
}

// CreateDevice creates a device and registers it with all managers
func (env *TestEnvironment) CreateDevice(t *testing.T, id, deviceType string) base.Device {
	env.mu.Lock()
	defer env.mu.Unlock()

	deviceConfig := config.DeviceConfig{
		ID:     id,
		Type:   deviceType,
		Tenant: "test",
		Site:   "test_site",
		Location: &config.Location{
			Room: "test_location",
		},
	}

	device, err := env.DeviceManager.CreateDevice(deviceConfig)
	require.NoError(t, err)

	env.Devices[id] = device
	env.AutomationManager.RegisterDevice(id, device)
	env.RoutineManager.RegisterDevice(id, device)
	env.ScriptEngine.RegisterDevice(id, device)

	if err := device.Start(env.Context); err != nil {
		t.Fatalf("Failed to start device %s: %v", id, err)
	}

	return device
}

// CreateTestConfig creates a test configuration file
func CreateTestConfig(t *testing.T, name string) string {
	simConfig := config.SimulationConfig{
		Simulation: config.SimulationSettings{
			Name:     name,
			Duration: 60 * time.Second,
		},
		Network: config.NetworkSettings{
			Topology: "single_router",
		},
		Devices: config.DeviceConfigs{
			NetworkDevices: []config.NetworkDeviceConfig{
				{
					DeviceConfig: config.DeviceConfig{
						ID:     "router_main",
						Type:   "router",
						Tenant: "test",
						Site:   "home",
						Location: &config.Location{
							Room: "living_room",
						},
					},
				},
			},
			IoTDevices: []config.IoTDeviceConfig{
				{
					DeviceConfig: config.DeviceConfig{
						ID:     "bulb_001",
						Type:   "smart_bulb",
						Tenant: "test",
						Site:   "home",
						Location: &config.Location{
							Room: "bedroom",
						},
					},
				},
				{
					DeviceConfig: config.DeviceConfig{
						ID:     "ac_001",
						Type:   "air_conditioner",
						Tenant: "test",
						Site:   "home",
						Location: &config.Location{
							Room: "living_room",
						},
					},
				},
			},
		},
	}

	// Create temp file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test_config.yaml")

	data, err := yaml.Marshal(simConfig)
	require.NoError(t, err)

	err = ioutil.WriteFile(configPath, data, 0644)
	require.NoError(t, err)

	return configPath
}

// GenerateRandomDeviceConfig generates random device configurations
func GenerateRandomDeviceConfig(count int) []config.DeviceConfig {
	deviceTypes := []string{
		"smart_bulb",
		"air_conditioner",
		"router",
		"switch",
		"access_point",
		"security_camera",
		"smart_plug",
		"environmental_sensor",
		"smart_thermostat",
		"smart_tv",
		"smartphone",
		"laptop",
		"tablet",
	}

	locations := []string{
		"living_room",
		"bedroom",
		"kitchen",
		"bathroom",
		"garage",
		"office",
		"basement",
		"attic",
		"hallway",
		"outdoor",
	}

	configs := make([]config.DeviceConfig, count)
	for i := 0; i < count; i++ {
		configs[i] = config.DeviceConfig{
			ID:     fmt.Sprintf("device_%06d", i),
			Type:   deviceTypes[rand.Intn(len(deviceTypes))],
			Tenant: "test_tenant",
			Site:   "test_site",
			Location: &config.Location{
				Room: locations[rand.Intn(len(locations))],
			},
		}
	}

	return configs
}

// CreateTestScript creates a test script for the script engine
func CreateTestScript(name string, steps int) *scenarios.Script {
	script := &scenarios.Script{
		ID:          name,
		Name:        name,
		Description: "Test script",
		Language:    "simple",
		Steps:       make([]scenarios.ScriptStep, steps),
		Variables:   make(map[string]interface{}),
		Enabled:     true,
	}

	for i := 0; i < steps; i++ {
		script.Steps[i] = scenarios.ScriptStep{
			ID:   fmt.Sprintf("step_%d", i),
			Type: "action",
			Action: &scenarios.ScriptAction{
				Type: "log",
				Parameters: map[string]interface{}{
					"message": fmt.Sprintf("Step %d", i),
				},
			},
		}
	}

	return script
}

// CreateTestAutomationRule creates a test automation rule
func CreateTestAutomationRule(name string) *scenarios.AutomationRule {
	return &scenarios.AutomationRule{
		ID:          name,
		Name:        name,
		Description: "Test automation rule",
		Triggers: []scenarios.Trigger{
			{
				ID:       "trigger_1",
				Type:     "time",
				Event:    "time_reached",
				Value:    "12:00",
				Operator: "eq",
			},
		},
		Conditions: []scenarios.Condition{
			{
				ID:       "condition_1",
				Type:     "device_state",
				Source:   "test_device",
				Property: "power",
				Operator: "eq",
				Value:    true,
			},
		},
		Actions: []scenarios.Action{
			{
				ID:      "action_1",
				Type:    "device_control",
				Target:  "smart_bulb",
				Command: "turn_on",
				Parameters: map[string]interface{}{
					"brightness": 80,
				},
			},
		},
		Enabled:  true,
		Priority: 5,
		Cooldown: 1 * time.Minute,
	}
}

// WaitForCondition waits for a condition to be true with timeout
func WaitForCondition(t *testing.T, timeout time.Duration, checkFunc func() bool, message string) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if checkFunc() {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("Timeout waiting for condition: %s", message)
}

// AssertEventReceived checks if an event was received within timeout
func AssertEventReceived(t *testing.T, eventChan <-chan scenarios.Event, timeout time.Duration) scenarios.Event {
	select {
	case event := <-eventChan:
		return event
	case <-time.After(timeout):
		t.Fatal("Timeout waiting for event")
		return scenarios.Event{}
	}
}

// LoadFixture loads a fixture file from the fixtures directory
func LoadFixture(t *testing.T, name string) []byte {
	fixturePath := filepath.Join("../fixtures", name)
	data, err := ioutil.ReadFile(fixturePath)
	require.NoError(t, err)
	return data
}

// CompareJSON compares two JSON objects for equality
func CompareJSON(t *testing.T, expected, actual interface{}) {
	expectedJSON, err := json.MarshalIndent(expected, "", "  ")
	require.NoError(t, err)

	actualJSON, err := json.MarshalIndent(actual, "", "  ")
	require.NoError(t, err)

	require.JSONEq(t, string(expectedJSON), string(actualJSON))
}

// MeasurePerformance measures the performance of a function
func MeasurePerformance(name string, fn func()) time.Duration {
	start := time.Now()
	fn()
	duration := time.Since(start)
	fmt.Printf("Performance [%s]: %v\n", name, duration)
	return duration
}

// GenerateTestData generates various test data
type TestData struct {
	Devices  []config.DeviceConfig
	Scripts  []*scenarios.Script
	Rules    []*scenarios.AutomationRule
	Scenes   []*scenarios.Scene
	Routines []*scenarios.DailyRoutine
}

func GenerateTestData(deviceCount, scriptCount, ruleCount int) *TestData {
	data := &TestData{
		Devices:  GenerateRandomDeviceConfig(deviceCount),
		Scripts:  make([]*scenarios.Script, scriptCount),
		Rules:    make([]*scenarios.AutomationRule, ruleCount),
		Scenes:   make([]*scenarios.Scene, 0),
		Routines: make([]*scenarios.DailyRoutine, 0),
	}

	for i := 0; i < scriptCount; i++ {
		data.Scripts[i] = CreateTestScript(fmt.Sprintf("script_%d", i), 5)
	}

	for i := 0; i < ruleCount; i++ {
		data.Rules[i] = CreateTestAutomationRule(fmt.Sprintf("rule_%d", i))
	}

	// Add some predefined scenes
	data.Scenes = append(data.Scenes, &scenarios.Scene{
		ID:          "test_scene",
		Name:        "Test Scene",
		Description: "A test scene",
		Type:        "custom",
		DeviceStates: []scenarios.DeviceState{
			{
				DeviceType: "smart_bulb",
				State:      "on",
				Properties: map[string]interface{}{
					"brightness": 100,
				},
			},
		},
		Enabled: true,
	})

	// Add some routines
	data.Routines = append(data.Routines, &scenarios.DailyRoutine{
		ID:        "test_routine",
		Name:      "Test Routine",
		Type:      "custom",
		StartTime: "09:00",
		EndTime:   "17:00",
		Weekdays:  []string{"Mon", "Tue", "Wed", "Thu", "Fri"},
		Actions: []scenarios.RoutineAction{
			{
				ID:         "test_action",
				Name:       "Test Action",
				DeviceType: "smart_bulb",
				Command:    "turn_on",
				Parameters: map[string]interface{}{
					"brightness": 80,
				},
			},
		},
		Enabled: true,
	})

	return data
}

// SimulationValidator validates simulation results
type SimulationValidator struct {
	t               *testing.T
	expectedDevices int
	expectedEvents  int
	receivedEvents  []scenarios.Event
	deviceStates    map[string]base.StatePayload
}

func NewSimulationValidator(t *testing.T) *SimulationValidator {
	return &SimulationValidator{
		t:              t,
		receivedEvents: make([]scenarios.Event, 0),
		deviceStates:   make(map[string]base.StatePayload),
	}
}

func (v *SimulationValidator) RecordEvent(event scenarios.Event) {
	v.receivedEvents = append(v.receivedEvents, event)
}

func (v *SimulationValidator) RecordDeviceState(deviceID string, state base.StatePayload) {
	v.deviceStates[deviceID] = state
}

func (v *SimulationValidator) Validate() {
	// Validate device count
	if v.expectedDevices > 0 {
		require.GreaterOrEqual(v.t, len(v.deviceStates), v.expectedDevices,
			"Expected at least %d devices, got %d", v.expectedDevices, len(v.deviceStates))
	}

	// Validate event count
	if v.expectedEvents > 0 {
		require.GreaterOrEqual(v.t, len(v.receivedEvents), v.expectedEvents,
			"Expected at least %d events, got %d", v.expectedEvents, len(v.receivedEvents))
	}

	// Validate all devices are healthy
	for deviceID, state := range v.deviceStates {
		require.NotEqual(v.t, "critical", state.Health,
			"Device %s is in critical state", deviceID)
	}
}

// CreateMockMQTTBroker creates a mock MQTT broker for testing
func CreateMockMQTTBroker(t *testing.T, port int) func() {
	// This is a placeholder for a real mock broker implementation
	// In a real scenario, you would start a lightweight MQTT broker here
	return func() {
		// Cleanup function
	}
}

// CaptureLogOutput captures log output during test execution
func CaptureLogOutput(fn func()) string {
	// Redirect stdout to capture logs
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	fn()

	w.Close()
	os.Stdout = old

	output, _ := ioutil.ReadAll(r)
	return string(output)
}

// AssertNoError is a helper to assert no error with proper message
func AssertNoError(t *testing.T, err error, message string) {
	if err != nil {
		t.Fatalf("%s: %v", message, err)
	}
}

// RandomString generates a random string of specified length
func RandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

// GenerateMACAddress generates a random MAC address
func GenerateMACAddress() string {
	mac := make([]byte, 6)
	rand.Read(mac)
	return fmt.Sprintf("%02x:%02x:%02x:%02x:%02x:%02x",
		mac[0], mac[1], mac[2], mac[3], mac[4], mac[5])
}

// GenerateIPAddress generates a random IP address in the specified subnet
func GenerateIPAddress(subnet string) string {
	// Simple implementation for 192.168.x.x subnet
	return fmt.Sprintf("192.168.%d.%d", rand.Intn(255), rand.Intn(255))
}
