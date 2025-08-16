package integration

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"rtk_controller/internal/cli"
	"rtk_controller/internal/command"
	"rtk_controller/internal/config"
	"rtk_controller/internal/device"
	"rtk_controller/internal/diagnosis"
	mqtt_client "rtk_controller/internal/mqtt"
	"rtk_controller/internal/storage"
	"rtk_controller/pkg/types"
)

// TestCLIEnvironment sets up a complete CLI testing environment
type TestCLIEnvironment struct {
	Storage          storage.Storage
	MQTTClient       *MockMQTTClientForCLI
	DeviceManager    *device.Manager
	CommandManager   *command.Manager
	DiagnosisManager *diagnosis.Manager
	CLI              *cli.InteractiveCLI
	TempDir          string
}

// MockMQTTClientForCLI implements the MQTT client interface for CLI testing
type MockMQTTClientForCLI struct {
	connected       bool
	publishedMsgs   []PublishedMessage
	messageLogger   *MockMessageLogger
}

type PublishedMessage struct {
	Topic     string
	Payload   interface{}
	QoS       int
	Retained  bool
	Timestamp time.Time
}

type MockMessageLogger struct {
	messages []types.MQTTMessageLog
	stats    types.MQTTLoggerStats
}

func (m *MockMessageLogger) LogMessage(topic string, payload []byte, qos byte, retained bool) error {
	msg := types.MQTTMessageLog{
		ID:          fmt.Sprintf("msg-%d", len(m.messages)+1),
		Timestamp:   time.Now().UnixMilli(),
		Topic:       topic,
		Payload:     string(payload),
		QoS:         qos,
		Retained:    retained,
		MessageSize: len(payload),
	}
	m.messages = append(m.messages, msg)
	m.stats.TotalMessages++
	return nil
}

func (m *MockMessageLogger) GetMessages(offset, limit int) ([]types.MQTTMessageLog, error) {
	if offset >= len(m.messages) {
		return []types.MQTTMessageLog{}, nil
	}
	
	end := offset + limit
	if end > len(m.messages) {
		end = len(m.messages)
	}
	
	return m.messages[offset:end], nil
}

func (m *MockMessageLogger) GetStats() types.MQTTLoggerStats {
	return m.stats
}

func (m *MockMessageLogger) PurgeOldMessages() (int, error) {
	// For testing, don't actually purge
	return 0, nil
}

func (m *MockMQTTClientForCLI) IsConnected() bool {
	return m.connected
}

func (m *MockMQTTClientForCLI) Publish(topic string, payload interface{}, qos int, retained bool) error {
	if !m.connected {
		return fmt.Errorf("client not connected")
	}
	
	msg := PublishedMessage{
		Topic:     topic,
		Payload:   payload,
		QoS:       qos,
		Retained:  retained,
		Timestamp: time.Now(),
	}
	m.publishedMsgs = append(m.publishedMsgs, msg)
	return nil
}

func (m *MockMQTTClientForCLI) Subscribe(topics []string) error {
	return nil
}

func (m *MockMQTTClientForCLI) Unsubscribe(topics []string) error {
	return nil
}

func (m *MockMQTTClientForCLI) Connect(ctx context.Context) error {
	m.connected = true
	return nil
}

func (m *MockMQTTClientForCLI) Disconnect() {
	m.connected = false
}

func (m *MockMQTTClientForCLI) GetMessageLogger() mqtt_client.MessageLogger {
	return m.messageLogger
}

func (m *MockMQTTClientForCLI) SetDeviceManager(dm interface{}) {
	// Mock implementation
}

func (m *MockMQTTClientForCLI) SetEventProcessor(ep interface{}) {
	// Mock implementation
}

func (m *MockMQTTClientForCLI) SetSchemaValidator(sv interface{}) {
	// Mock implementation
}

// NewTestCLIEnvironment creates a complete test environment for CLI testing
func NewTestCLIEnvironment(t *testing.T) *TestCLIEnvironment {
	tempDir, err := os.MkdirTemp("", "cli_integration_test")
	require.NoError(t, err)

	// Create storage
	storage, err := storage.NewBuntDB(filepath.Join(tempDir, "test.db"))
	require.NoError(t, err)

	// Create mock MQTT client
	mockMQTT := &MockMQTTClientForCLI{
		connected:     true,
		publishedMsgs: make([]PublishedMessage, 0),
		messageLogger: &MockMessageLogger{
			messages: make([]types.MQTTMessageLog, 0),
		},
	}

	// Create managers
	deviceManager := device.NewManager(storage)
	commandManager := command.NewManager(mockMQTT, storage)
	
	diagnosisConfig := config.DiagnosisConfig{
		Enabled: true,
		DefaultAnalyzers: []string{"builtin_wifi_analyzer"},
	}
	diagnosisManager := diagnosis.NewManager(diagnosisConfig, storage)

	// Create test config
	testConfig := &config.Config{
		MQTT: config.MQTTConfig{
			Broker:   "localhost",
			Port:     1883,
			ClientID: "test-cli-client",
		},
		Storage: config.StorageConfig{
			Path: tempDir,
		},
		Logging: config.LoggingConfig{
			Level: "info",
		},
	}

	// Create CLI
	interactiveCLI := cli.NewInteractiveCLI(
		testConfig,
		mockMQTT,
		storage,
		deviceManager,
		commandManager,
		diagnosisManager,
	)

	return &TestCLIEnvironment{
		Storage:          storage,
		MQTTClient:       mockMQTT,
		DeviceManager:    deviceManager,
		CommandManager:   commandManager,
		DiagnosisManager: diagnosisManager,
		CLI:              interactiveCLI,
		TempDir:          tempDir,
	}
}

func (env *TestCLIEnvironment) Cleanup() {
	env.Storage.Close()
	os.RemoveAll(env.TempDir)
}

// ExecuteCommand executes a CLI command and returns the output
func (env *TestCLIEnvironment) ExecuteCommand(command string) (string, error) {
	// Create a buffer to capture output
	var outputBuffer bytes.Buffer
	
	// Parse and execute the command
	args := strings.Fields(command)
	if len(args) == 0 {
		return "", fmt.Errorf("empty command")
	}

	// Redirect output for testing
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Execute command
	err := env.CLI.ExecuteCommand(args, &outputBuffer)

	// Restore stdout
	w.Close()
	os.Stdout = oldStdout

	// Read captured output
	output, _ := io.ReadAll(r)
	
	return string(output) + outputBuffer.String(), err
}

func TestCLI_DeviceCommands(t *testing.T) {
	env := NewTestCLIEnvironment(t)
	defer env.Cleanup()

	// Start managers
	ctx := context.Background()
	err := env.DeviceManager.Start(ctx)
	require.NoError(t, err)

	// Test device list (initially empty)
	output, err := env.ExecuteCommand("device list")
	assert.NoError(t, err)
	assert.Contains(t, output, "No devices found")

	// Register test devices
	testDevices := []types.ExtendedDeviceInfo{
		{
			DeviceInfo: types.DeviceInfo{
				Tenant:   "test-tenant",
				Site:     "test-site",
				DeviceID: "router-001",
				Type:     "wifi_router",
			},
			Location: "Office Floor 1",
			Model:    "RTK-Router-Pro",
			Firmware: "1.2.3",
		},
		{
			DeviceInfo: types.DeviceInfo{
				Tenant:   "test-tenant",
				Site:     "test-site",
				DeviceID: "switch-001",
				Type:     "smart_switch",
			},
			Location: "Server Room",
			Model:    "RTK-Switch-24",
			Firmware: "2.1.0",
		},
	}

	for _, device := range testDevices {
		err = env.DeviceManager.RegisterDevice(device)
		require.NoError(t, err)
	}

	// Test device list (should show devices)
	output, err = env.ExecuteCommand("device list")
	assert.NoError(t, err)
	assert.Contains(t, output, "router-001")
	assert.Contains(t, output, "switch-001")
	assert.Contains(t, output, "wifi_router")
	assert.Contains(t, output, "smart_switch")

	// Test device list with filter
	output, err = env.ExecuteCommand("device list --tenant test-tenant")
	assert.NoError(t, err)
	assert.Contains(t, output, "router-001")
	assert.Contains(t, output, "switch-001")

	// Test device info
	output, err = env.ExecuteCommand("device info test-tenant test-site router-001")
	assert.NoError(t, err)
	assert.Contains(t, output, "router-001")
	assert.Contains(t, output, "Office Floor 1")
	assert.Contains(t, output, "RTK-Router-Pro")
	assert.Contains(t, output, "1.2.3")

	// Test device info for non-existent device
	output, err = env.ExecuteCommand("device info test-tenant test-site non-existent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")

	// Test device state update
	stateData := map[string]interface{}{
		"cpu_usage":    45.2,
		"memory_usage": 67.8,
		"uptime":       3600,
	}
	
	err = env.DeviceManager.UpdateDeviceState(testDevices[0].DeviceInfo, stateData)
	require.NoError(t, err)

	// Test device state command
	output, err = env.ExecuteCommand("device state test-tenant test-site router-001")
	assert.NoError(t, err)
	assert.Contains(t, output, "45.2") // CPU usage
	assert.Contains(t, output, "67.8") // Memory usage
}

func TestCLI_CommandManagement(t *testing.T) {
	env := NewTestCLIEnvironment(t)
	defer env.Cleanup()

	// Start managers
	ctx := context.Background()
	err := env.CommandManager.Start(ctx)
	require.NoError(t, err)

	// Register a test device
	deviceInfo := types.DeviceInfo{
		Tenant:   "test-tenant",
		Site:     "test-site",
		DeviceID: "router-001",
		Type:     "wifi_router",
	}

	testDevice := types.ExtendedDeviceInfo{
		DeviceInfo: deviceInfo,
		Location:   "Test Location",
	}
	
	err = env.DeviceManager.RegisterDevice(testDevice)
	require.NoError(t, err)

	// Test command list (initially empty)
	output, err := env.ExecuteCommand("command list")
	assert.NoError(t, err)
	assert.Contains(t, output, "No commands found")

	// Test sending a command
	output, err = env.ExecuteCommand("command send test-tenant test-site router-001 restart")
	assert.NoError(t, err)
	assert.Contains(t, output, "Command sent successfully")

	// Verify command was published to MQTT
	assert.Len(t, env.MQTTClient.publishedMsgs, 1)
	publishedMsg := env.MQTTClient.publishedMsgs[0]
	assert.Contains(t, publishedMsg.Topic, "rtk/v1/test-tenant/test-site/router-001/cmd/req")

	// Test command list (should show the command)
	output, err = env.ExecuteCommand("command list")
	assert.NoError(t, err)
	assert.Contains(t, output, "restart")
	assert.Contains(t, output, "pending")

	// Test command status
	commands := env.CommandManager.ListCommands("")
	require.Len(t, commands, 1)
	commandID := commands[0].ID

	output, err = env.ExecuteCommand(fmt.Sprintf("command status %s", commandID))
	assert.NoError(t, err)
	assert.Contains(t, output, commandID)
	assert.Contains(t, output, "pending")

	// Test command with parameters
	output, err = env.ExecuteCommand("command send test-tenant test-site router-001 reboot --delay 30")
	assert.NoError(t, err)
	assert.Contains(t, output, "Command sent successfully")

	// Verify parameter was included
	assert.Len(t, env.MQTTClient.publishedMsgs, 2)
	secondMsg := env.MQTTClient.publishedMsgs[1]
	assert.Contains(t, secondMsg.Topic, "cmd/req")

	// Test invalid command
	output, err = env.ExecuteCommand("command send invalid-tenant invalid-site invalid-device invalid-action")
	assert.Error(t, err)
}

func TestCLI_SystemCommands(t *testing.T) {
	env := NewTestCLIEnvironment(t)
	defer env.Cleanup()

	// Test system status
	output, err := env.ExecuteCommand("system status")
	assert.NoError(t, err)
	assert.Contains(t, output, "RTK Controller System Status")
	assert.Contains(t, output, "MQTT")
	assert.Contains(t, output, "connected") // Mock MQTT is always connected

	// Test system health
	output, err = env.ExecuteCommand("system health")
	assert.NoError(t, err)
	assert.Contains(t, output, "System Health Check")
	assert.Contains(t, output, "healthy")

	// Test system stats
	output, err = env.ExecuteCommand("system stats")
	assert.NoError(t, err)
	assert.Contains(t, output, "System Statistics")
	assert.Contains(t, output, "METRIC")
	assert.Contains(t, output, "VALUE")

	// Test config show
	output, err = env.ExecuteCommand("config show")
	assert.NoError(t, err)
	assert.Contains(t, output, "localhost") // MQTT broker
	assert.Contains(t, output, "1883")      // MQTT port

	// Test config show with section
	output, err = env.ExecuteCommand("config show --section mqtt")
	assert.NoError(t, err)
	assert.Contains(t, output, "localhost")
	assert.Contains(t, output, "1883")
}

func TestCLI_LogCommands(t *testing.T) {
	env := NewTestCLIEnvironment(t)
	defer env.Cleanup()

	// Add some test messages to the message logger
	messageLogger := env.MQTTClient.GetMessageLogger()
	testMessages := []struct {
		topic   string
		payload string
	}{
		{"rtk/v1/test/site1/device1/state", `{"status": "online"}`},
		{"rtk/v1/test/site1/device1/evt/wifi_disconnect", `{"reason": "signal_lost"}`},
		{"rtk/v1/test/site1/device2/state", `{"status": "offline"}`},
	}

	for _, msg := range testMessages {
		err := messageLogger.LogMessage(msg.topic, []byte(msg.payload), 0, false)
		require.NoError(t, err)
	}

	// Test log list
	output, err := env.ExecuteCommand("log list")
	assert.NoError(t, err)
	assert.Contains(t, output, "rtk/v1/test/site1/device1/state")
	assert.Contains(t, output, "online")

	// Test log list with limit
	output, err = env.ExecuteCommand("log list --limit 2")
	assert.NoError(t, err)
	// Should contain at most 2 messages worth of output

	// Test log stats
	output, err = env.ExecuteCommand("log stats")
	assert.NoError(t, err)
	assert.Contains(t, output, "Total Messages: 3")

	// Test log filter
	output, err = env.ExecuteCommand("log list --topic-filter */state")
	assert.NoError(t, err)
	assert.Contains(t, output, "state")
	// Should not contain event messages

	// Test download command
	output, err = env.ExecuteCommand("download 3600")
	assert.NoError(t, err)
	assert.Contains(t, output, "messages downloaded")
}

func TestCLI_DiagnosisCommands(t *testing.T) {
	env := NewTestCLIEnvironment(t)
	defer env.Cleanup()

	// Start diagnosis manager
	ctx := context.Background()
	err := env.DiagnosisManager.Start(ctx)
	require.NoError(t, err)

	// Test diagnosis list (initially empty)
	output, err := env.ExecuteCommand("diagnosis list")
	assert.NoError(t, err)
	assert.Contains(t, output, "No diagnosis sessions found")

	// Register a test device first
	deviceInfo := types.DeviceInfo{
		Tenant:   "test-tenant",
		Site:     "test-site",
		DeviceID: "router-001",
		Type:     "wifi_router",
	}

	testDevice := types.ExtendedDeviceInfo{
		DeviceInfo: deviceInfo,
		Location:   "Test Location",
	}
	
	err = env.DeviceManager.RegisterDevice(testDevice)
	require.NoError(t, err)

	// Test diagnosis start
	output, err = env.ExecuteCommand("diagnosis start test-tenant test-site router-001 wifi_disconnect")
	assert.NoError(t, err)
	assert.Contains(t, output, "Diagnosis session started")

	// Test diagnosis list (should show the session)
	output, err = env.ExecuteCommand("diagnosis list")
	assert.NoError(t, err)
	assert.Contains(t, output, "wifi_disconnect")
	assert.Contains(t, output, "router-001")

	// Test diagnosis analyzers list
	output, err = env.ExecuteCommand("diagnosis analyzers")
	assert.NoError(t, err)
	assert.Contains(t, output, "Available analyzers")

	// Test analyzer info
	output, err = env.ExecuteCommand("diagnosis analyzer builtin_wifi_analyzer")
	assert.NoError(t, err)
	assert.Contains(t, output, "builtin_wifi_analyzer")
}

func TestCLI_HelpCommands(t *testing.T) {
	env := NewTestCLIEnvironment(t)
	defer env.Cleanup()

	// Test main help
	output, err := env.ExecuteCommand("help")
	assert.NoError(t, err)
	assert.Contains(t, output, "Available commands")
	assert.Contains(t, output, "device")
	assert.Contains(t, output, "command")
	assert.Contains(t, output, "system")

	// Test device help
	output, err = env.ExecuteCommand("device help")
	assert.NoError(t, err)
	assert.Contains(t, output, "Device management commands")
	assert.Contains(t, output, "list")
	assert.Contains(t, output, "info")

	// Test command help
	output, err = env.ExecuteCommand("command help")
	assert.NoError(t, err)
	assert.Contains(t, output, "Command management")
	assert.Contains(t, output, "send")
	assert.Contains(t, output, "list")

	// Test help for specific command
	output, err = env.ExecuteCommand("help device list")
	assert.NoError(t, err)
	assert.Contains(t, output, "List all devices")
}

func TestCLI_ErrorHandling(t *testing.T) {
	env := NewTestCLIEnvironment(t)
	defer env.Cleanup()

	// Test invalid command
	output, err := env.ExecuteCommand("invalid-command")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Unknown command")

	// Test command with missing arguments
	output, err = env.ExecuteCommand("device info")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "required")

	// Test command with invalid arguments
	output, err = env.ExecuteCommand("device info invalid-tenant")
	assert.Error(t, err)

	// Test command when MQTT is disconnected
	env.MQTTClient.connected = false
	output, err = env.ExecuteCommand("command send test-tenant test-site device-001 restart")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not connected")
}

func TestCLI_OutputFormatting(t *testing.T) {
	env := NewTestCLIEnvironment(t)
	defer env.Cleanup()

	// Register test devices with different states
	devices := []types.ExtendedDeviceInfo{
		{
			DeviceInfo: types.DeviceInfo{
				Tenant:   "tenant1",
				Site:     "site1",
				DeviceID: "device1",
				Type:     "wifi_router",
			},
			Location: "Location 1",
		},
		{
			DeviceInfo: types.DeviceInfo{
				Tenant:   "tenant1",
				Site:     "site1",
				DeviceID: "device2",
				Type:     "smart_switch",
			},
			Location: "Location 2",
		},
	}

	for _, device := range devices {
		err := env.DeviceManager.RegisterDevice(device)
		require.NoError(t, err)
	}

	// Test table format (default)
	output, err := env.ExecuteCommand("device list")
	assert.NoError(t, err)
	assert.Contains(t, output, "DEVICE ID")
	assert.Contains(t, output, "TYPE")
	assert.Contains(t, output, "STATUS")
	assert.Contains(t, output, "device1")
	assert.Contains(t, output, "device2")

	// Test JSON format
	output, err = env.ExecuteCommand("device list --format json")
	assert.NoError(t, err)
	assert.Contains(t, output, `"device_id"`)
	assert.Contains(t, output, `"type"`)
	assert.Contains(t, output, "device1")

	// Test CSV format
	output, err = env.ExecuteCommand("device list --format csv")
	assert.NoError(t, err)
	assert.Contains(t, output, ",") // CSV separator
	assert.Contains(t, output, "device1")
}

func TestCLI_InteractiveFeatures(t *testing.T) {
	env := NewTestCLIEnvironment(t)
	defer env.Cleanup()

	// Test command completion
	completions := env.CLI.GetCompletions("dev")
	assert.Contains(t, completions, "device")

	completions = env.CLI.GetCompletions("device l")
	assert.Contains(t, completions, "device list")

	// Test command history (would be more complex in real implementation)
	history := env.CLI.GetCommandHistory()
	assert.NotNil(t, history)

	// Test command validation
	isValid := env.CLI.ValidateCommand("device list")
	assert.True(t, isValid)

	isValid = env.CLI.ValidateCommand("invalid command")
	assert.False(t, isValid)
}

func TestCLI_ConcurrentOperations(t *testing.T) {
	env := NewTestCLIEnvironment(t)
	defer env.Cleanup()

	// Start managers
	ctx := context.Background()
	err := env.DeviceManager.Start(ctx)
	require.NoError(t, err)
	err = env.CommandManager.Start(ctx)
	require.NoError(t, err)

	// Register test device
	deviceInfo := types.DeviceInfo{
		Tenant:   "test-tenant",
		Site:     "test-site",
		DeviceID: "router-001",
		Type:     "wifi_router",
	}

	testDevice := types.ExtendedDeviceInfo{
		DeviceInfo: deviceInfo,
	}
	
	err = env.DeviceManager.RegisterDevice(testDevice)
	require.NoError(t, err)

	// Test concurrent command execution
	const numCommands = 5
	done := make(chan bool, numCommands)
	
	for i := 0; i < numCommands; i++ {
		go func(id int) {
			defer func() { done <- true }()
			
			command := fmt.Sprintf("command send test-tenant test-site router-001 test-action-%d", id)
			output, err := env.ExecuteCommand(command)
			
			assert.NoError(t, err)
			assert.Contains(t, output, "Command sent successfully")
		}(i)
	}

	// Wait for all commands to complete
	for i := 0; i < numCommands; i++ {
		<-done
	}

	// Verify all commands were processed
	commands := env.CommandManager.ListCommands("")
	assert.Len(t, commands, numCommands)
	
	// Verify all MQTT messages were published
	assert.Len(t, env.MQTTClient.publishedMsgs, numCommands)
}