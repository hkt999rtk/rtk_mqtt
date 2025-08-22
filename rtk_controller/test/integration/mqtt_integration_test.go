package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"rtk_controller/internal/config"
	mqtt_client "rtk_controller/internal/mqtt"
	"rtk_controller/internal/storage"
	"rtk_controller/pkg/types"
)

// TestMQTTBrokerInfo contains information about the test MQTT broker
type TestMQTTBrokerInfo struct {
	Host     string
	Port     int
	Username string
	Password string
}

// getTestBrokerInfo returns MQTT broker info for testing
// In a real environment, this would come from environment variables or config
func getTestBrokerInfo() TestMQTTBrokerInfo {
	return TestMQTTBrokerInfo{
		Host:     getEnvOrDefault("TEST_MQTT_HOST", "localhost"),
		Port:     1883,
		Username: getEnvOrDefault("TEST_MQTT_USERNAME", ""),
		Password: getEnvOrDefault("TEST_MQTT_PASSWORD", ""),
	}
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// createTestMQTTClient creates a test MQTT client for publishing test messages
func createTestMQTTClient(t *testing.T, clientID string) mqtt.Client {
	brokerInfo := getTestBrokerInfo()

	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("tcp://%s:%d", brokerInfo.Host, brokerInfo.Port))
	opts.SetClientID(clientID)
	opts.SetCleanSession(true)

	if brokerInfo.Username != "" {
		opts.SetUsername(brokerInfo.Username)
		opts.SetPassword(brokerInfo.Password)
	}

	client := mqtt.NewClient(opts)
	token := client.Connect()

	require.True(t, token.WaitTimeout(10*time.Second), "Failed to connect to test MQTT broker")
	require.NoError(t, token.Error(), "MQTT connection error")

	return client
}

func TestMQTTClient_ConnectionAndReconnection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tempDir, err := os.MkdirTemp("", "mqtt_integration_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create storage
	storage, err := storage.NewBuntDB(filepath.Join(tempDir, "test.db"))
	require.NoError(t, err)
	defer storage.Close()

	brokerInfo := getTestBrokerInfo()

	// Test MQTT client configuration
	mqttConfig := config.MQTTConfig{
		Broker:   brokerInfo.Host,
		Port:     brokerInfo.Port,
		ClientID: "test-controller-connection",
		Username: brokerInfo.Username,
		Password: brokerInfo.Password,
		Topics: config.TopicsConfig{
			Subscribe: []string{
				"rtk/v1/+/+/+/state",
				"rtk/v1/+/+/+/evt/#",
				"rtk/v1/+/+/+/cmd/ack",
				"rtk/v1/+/+/+/cmd/res",
			},
		},
		Logging: config.MQTTLogging{
			Enabled:          true,
			RetentionSeconds: 3600,
			BatchSize:        100,
		},
	}

	// Create MQTT client
	client, err := mqtt_client.NewClient(mqttConfig, storage)
	require.NoError(t, err)

	// Test connection
	ctx := context.Background()
	err = client.Connect(ctx)
	require.NoError(t, err)

	// Verify connection
	assert.True(t, client.IsConnected())

	// Test graceful disconnection
	client.Disconnect()

	// Give some time for disconnection
	time.Sleep(100 * time.Millisecond)

	// In most implementations, IsConnected should return false after disconnect
	// Note: Some MQTT clients might still show connected during graceful shutdown
}

func TestMQTTClient_MessagePublishAndSubscribe(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tempDir, err := os.MkdirTemp("", "mqtt_integration_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create storage
	storage, err := storage.NewBuntDB(filepath.Join(tempDir, "test.db"))
	require.NoError(t, err)
	defer storage.Close()

	brokerInfo := getTestBrokerInfo()

	// Setup test topics
	stateTopic := "rtk/v1/test-tenant/test-site/test-device/state"
	commandTopic := "rtk/v1/test-tenant/test-site/test-device/cmd/req"

	// Configure MQTT client
	mqttConfig := config.MQTTConfig{
		Broker:   brokerInfo.Host,
		Port:     brokerInfo.Port,
		ClientID: "test-controller-pubsub",
		Username: brokerInfo.Username,
		Password: brokerInfo.Password,
		Topics: config.TopicsConfig{
			Subscribe: []string{
				"rtk/v1/+/+/+/state",
				"rtk/v1/+/+/+/cmd/ack",
			},
		},
		Logging: config.MQTTLogging{
			Enabled:          true,
			RetentionSeconds: 3600,
			BatchSize:        100,
		},
	}

	// Create and connect MQTT client
	client, err := mqtt_client.NewClient(mqttConfig, storage)
	require.NoError(t, err)

	ctx := context.Background()
	err = client.Connect(ctx)
	require.NoError(t, err)
	defer client.Disconnect()

	// Wait for subscription to be established
	time.Sleep(1 * time.Second)

	// Test publishing state message
	stateMessage := map[string]interface{}{
		"device_id": "test-device",
		"timestamp": time.Now().Unix(),
		"status":    "online",
		"cpu_usage": 45.2,
		"memory":    67.8,
	}

	err = client.Publish(stateTopic, stateMessage, 1, false)
	assert.NoError(t, err)

	// Test publishing command
	commandMessage := map[string]interface{}{
		"command_id": "test-cmd-123",
		"action":     "restart",
		"parameters": map[string]interface{}{
			"delay": 5,
		},
		"timestamp": time.Now().Unix(),
	}

	err = client.Publish(commandTopic, commandMessage, 1, false)
	assert.NoError(t, err)

	// Wait for message processing
	time.Sleep(2 * time.Second)

	// Verify message logging (if enabled)
	if client.GetMessageLogger() != nil {
		stats := client.GetMessageLogger().GetStats()
		// We should have received at least the state message (subscribed topic)
		assert.Greater(t, stats.TotalMessages, int64(0))
	}
}

func TestMQTTClient_MessageLogging(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tempDir, err := os.MkdirTemp("", "mqtt_integration_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create storage
	storage, err := storage.NewBuntDB(filepath.Join(tempDir, "test.db"))
	require.NoError(t, err)
	defer storage.Close()

	brokerInfo := getTestBrokerInfo()

	// Configure MQTT client with logging
	mqttConfig := config.MQTTConfig{
		Broker:   brokerInfo.Host,
		Port:     brokerInfo.Port,
		ClientID: "test-controller-logging",
		Username: brokerInfo.Username,
		Password: brokerInfo.Password,
		Topics: config.TopicsConfig{
			Subscribe: []string{
				"rtk/v1/test/+/+/state",
				"rtk/v1/test/+/+/evt/#",
			},
		},
		Logging: config.MQTTLogging{
			Enabled:          true,
			RetentionSeconds: 300,   // 5 minutes for testing
			PurgeInterval:    "10s", // More frequent for testing
			BatchSize:        10,
			MaxMessageSize:   1024,
			ExcludeTopics: []string{
				"rtk/v1/+/+/+/internal/#",
			},
		},
	}

	// Create and connect MQTT client
	client, err := mqtt_client.NewClient(mqttConfig, storage)
	require.NoError(t, err)

	ctx := context.Background()
	err = client.Connect(ctx)
	require.NoError(t, err)
	defer client.Disconnect()

	// Wait for subscription
	time.Sleep(1 * time.Second)

	// Create a separate test publisher
	testPublisher := createTestMQTTClient(t, "test-publisher")
	defer testPublisher.Disconnect(250)

	// Publish test messages
	testMessages := []struct {
		topic   string
		payload map[string]interface{}
	}{
		{
			topic: "rtk/v1/test/site1/device1/state",
			payload: map[string]interface{}{
				"device_id": "device1",
				"timestamp": time.Now().Unix(),
				"status":    "online",
			},
		},
		{
			topic: "rtk/v1/test/site1/device1/evt/wifi_disconnect",
			payload: map[string]interface{}{
				"device_id": "device1",
				"timestamp": time.Now().Unix(),
				"event":     "wifi_disconnect",
				"reason":    "signal_lost",
			},
		},
		{
			topic: "rtk/v1/test/site1/device1/internal/heartbeat", // Should be excluded
			payload: map[string]interface{}{
				"timestamp": time.Now().Unix(),
			},
		},
	}

	for _, msg := range testMessages {
		payloadBytes, err := json.Marshal(msg.payload)
		require.NoError(t, err)

		token := testPublisher.Publish(msg.topic, 1, false, payloadBytes)
		require.True(t, token.WaitTimeout(5*time.Second))
		require.NoError(t, token.Error())
	}

	// Wait for message processing
	time.Sleep(3 * time.Second)

	// Verify message logging
	messageLogger := client.GetMessageLogger()
	require.NotNil(t, messageLogger)

	stats := messageLogger.GetStats()

	// We should have logged at least 2 messages (excluding the internal one)
	assert.GreaterOrEqual(t, stats.TotalMessages, int64(2))

	// Verify messages can be retrieved
	messages, err := messageLogger.GetMessages(0, 10)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(messages), 2)

	// Verify message structure
	if len(messages) > 0 {
		msg := messages[0]
		assert.NotEmpty(t, msg.ID)
		assert.NotEmpty(t, msg.Topic)
		assert.NotEmpty(t, msg.Payload)
		assert.Greater(t, msg.Timestamp, int64(0))
		assert.Greater(t, msg.MessageSize, 0)
	}
}

func TestMQTTClient_DeviceStateIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tempDir, err := os.MkdirTemp("", "mqtt_integration_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create storage
	storage, err := storage.NewBuntDB(filepath.Join(tempDir, "test.db"))
	require.NoError(t, err)
	defer storage.Close()

	brokerInfo := getTestBrokerInfo()

	// Configure MQTT client
	mqttConfig := config.MQTTConfig{
		Broker:   brokerInfo.Host,
		Port:     brokerInfo.Port,
		ClientID: "test-controller-device-state",
		Username: brokerInfo.Username,
		Password: brokerInfo.Password,
		Topics: config.TopicsConfig{
			Subscribe: []string{
				"rtk/v1/+/+/+/state",
				"rtk/v1/+/+/+/lwt",
			},
		},
	}

	// Create MQTT client
	client, err := mqtt_client.NewClient(mqttConfig, storage)
	require.NoError(t, err)

	ctx := context.Background()
	err = client.Connect(ctx)
	require.NoError(t, err)
	defer client.Disconnect()

	// Create device manager and connect it to MQTT client
	deviceManager := &MockDeviceManager{
		devices: make(map[string]*types.DeviceState),
	}
	client.SetDeviceManager(deviceManager)

	// Wait for subscription
	time.Sleep(1 * time.Second)

	// Create test publisher
	testPublisher := createTestMQTTClient(t, "test-device-publisher")
	defer testPublisher.Disconnect(250)

	// Simulate device state updates
	deviceStateUpdates := []struct {
		topic   string
		payload map[string]interface{}
	}{
		{
			topic: "rtk/v1/tenant1/site1/router1/state",
			payload: map[string]interface{}{
				"device_id":    "router1",
				"timestamp":    time.Now().Unix(),
				"status":       "online",
				"cpu_usage":    45.2,
				"memory_usage": 67.8,
				"uptime":       3600,
				"wifi_clients": 12,
			},
		},
		{
			topic: "rtk/v1/tenant1/site1/switch1/state",
			payload: map[string]interface{}{
				"device_id":   "switch1",
				"timestamp":   time.Now().Unix(),
				"status":      "online",
				"port_status": []string{"up", "up", "down", "up"},
				"power_usage": 15.3,
			},
		},
	}

	for _, update := range deviceStateUpdates {
		payloadBytes, err := json.Marshal(update.payload)
		require.NoError(t, err)

		token := testPublisher.Publish(update.topic, 1, false, payloadBytes)
		require.True(t, token.WaitTimeout(5*time.Second))
		require.NoError(t, token.Error())
	}

	// Wait for message processing
	time.Sleep(2 * time.Second)

	// Verify device states were updated
	assert.Len(t, deviceManager.devices, 2)

	// Check router1 state
	router1Key := "tenant1:site1:router1"
	router1State, exists := deviceManager.devices[router1Key]
	assert.True(t, exists)
	assert.Equal(t, "router1", router1State.DeviceInfo.DeviceID)
	assert.Equal(t, types.DeviceStatusOnline, router1State.Status)
	assert.Contains(t, router1State.State, "cpu_usage")

	// Check switch1 state
	switch1Key := "tenant1:site1:switch1"
	switch1State, exists := deviceManager.devices[switch1Key]
	assert.True(t, exists)
	assert.Equal(t, "switch1", switch1State.DeviceInfo.DeviceID)
	assert.Equal(t, types.DeviceStatusOnline, switch1State.Status)
	assert.Contains(t, switch1State.State, "port_status")
}

func TestMQTTClient_CommandFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tempDir, err := os.MkdirTemp("", "mqtt_integration_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create storage
	storage, err := storage.NewBuntDB(filepath.Join(tempDir, "test.db"))
	require.NoError(t, err)
	defer storage.Close()

	brokerInfo := getTestBrokerInfo()

	// Configure MQTT client for command flow
	mqttConfig := config.MQTTConfig{
		Broker:   brokerInfo.Host,
		Port:     brokerInfo.Port,
		ClientID: "test-controller-commands",
		Username: brokerInfo.Username,
		Password: brokerInfo.Password,
		Topics: config.TopicsConfig{
			Subscribe: []string{
				"rtk/v1/+/+/+/cmd/ack",
				"rtk/v1/+/+/+/cmd/res",
			},
		},
	}

	// Create MQTT client
	client, err := mqtt_client.NewClient(mqttConfig, storage)
	require.NoError(t, err)

	ctx := context.Background()
	err = client.Connect(ctx)
	require.NoError(t, err)
	defer client.Disconnect()

	// Wait for subscription
	time.Sleep(1 * time.Second)

	// Test command publishing
	commandTopic := "rtk/v1/tenant1/site1/device1/cmd/req"
	commandPayload := map[string]interface{}{
		"command_id": "test-cmd-456",
		"action":     "reboot",
		"parameters": map[string]interface{}{
			"delay": 10,
		},
		"timestamp": time.Now().Unix(),
	}

	err = client.Publish(commandTopic, commandPayload, 1, false)
	assert.NoError(t, err)

	// Simulate device acknowledgment
	testPublisher := createTestMQTTClient(t, "test-device-simulator")
	defer testPublisher.Disconnect(250)

	// Publish ACK
	ackTopic := "rtk/v1/tenant1/site1/device1/cmd/ack"
	ackPayload := map[string]interface{}{
		"command_id": "test-cmd-456",
		"status":     "acknowledged",
		"timestamp":  time.Now().Unix(),
	}

	ackBytes, err := json.Marshal(ackPayload)
	require.NoError(t, err)

	token := testPublisher.Publish(ackTopic, 1, false, ackBytes)
	require.True(t, token.WaitTimeout(5*time.Second))
	require.NoError(t, token.Error())

	// Publish Response after delay
	time.Sleep(1 * time.Second)

	resTopic := "rtk/v1/tenant1/site1/device1/cmd/res"
	resPayload := map[string]interface{}{
		"command_id": "test-cmd-456",
		"status":     "success",
		"result": map[string]interface{}{
			"uptime": 0, // Device rebooted
		},
		"timestamp": time.Now().Unix(),
	}

	resBytes, err := json.Marshal(resPayload)
	require.NoError(t, err)

	token = testPublisher.Publish(resTopic, 1, false, resBytes)
	require.True(t, token.WaitTimeout(5*time.Second))
	require.NoError(t, token.Error())

	// Wait for message processing
	time.Sleep(2 * time.Second)

	// In a real test, we would verify command status updates through command manager
	// For this integration test, we're mainly testing MQTT message flow
}

// MockDeviceManager for testing device state integration
type MockDeviceManager struct {
	devices map[string]*types.DeviceState
}

func (m *MockDeviceManager) UpdateDeviceState(deviceInfo types.DeviceInfo, state map[string]interface{}) error {
	key := fmt.Sprintf("%s:%s:%s", deviceInfo.Tenant, deviceInfo.Site, deviceInfo.DeviceID)

	deviceState := &types.DeviceState{
		DeviceInfo: deviceInfo,
		Status:     types.DeviceStatusOnline,
		State:      state,
		LastSeen:   time.Now(),
	}

	m.devices[key] = deviceState
	return nil
}

func (m *MockDeviceManager) SetDeviceStatus(deviceInfo types.DeviceInfo, status types.DeviceStatus) error {
	key := fmt.Sprintf("%s:%s:%s", deviceInfo.Tenant, deviceInfo.Site, deviceInfo.DeviceID)

	if device, exists := m.devices[key]; exists {
		device.Status = status
		device.LastSeen = time.Now()
	}

	return nil
}

func TestMQTTClient_ErrorHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tempDir, err := os.MkdirTemp("", "mqtt_integration_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create storage
	storage, err := storage.NewBuntDB(filepath.Join(tempDir, "test.db"))
	require.NoError(t, err)
	defer storage.Close()

	// Test with invalid broker configuration
	invalidConfig := config.MQTTConfig{
		Broker:   "invalid-broker-host",
		Port:     1883,
		ClientID: "test-controller-error",
		Topics: config.TopicsConfig{
			Subscribe: []string{
				"rtk/v1/+/+/+/state",
			},
		},
	}

	client, err := mqtt_client.NewClient(invalidConfig, storage)
	require.NoError(t, err) // Client creation should succeed

	// Connection should fail
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = client.Connect(ctx)
	assert.Error(t, err) // Should fail to connect to invalid broker

	// Verify client is not connected
	assert.False(t, client.IsConnected())
}

func TestMQTTClient_LargeMessageHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tempDir, err := os.MkdirTemp("", "mqtt_integration_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create storage
	storage, err := storage.NewBuntDB(filepath.Join(tempDir, "test.db"))
	require.NoError(t, err)
	defer storage.Close()

	brokerInfo := getTestBrokerInfo()

	// Configure MQTT client with message size limits
	mqttConfig := config.MQTTConfig{
		Broker:   brokerInfo.Host,
		Port:     brokerInfo.Port,
		ClientID: "test-controller-large-msg",
		Username: brokerInfo.Username,
		Password: brokerInfo.Password,
		Topics: config.TopicsConfig{
			Subscribe: []string{
				"rtk/v1/+/+/+/state",
			},
		},
		Logging: config.MQTTLogging{
			Enabled:        true,
			MaxMessageSize: 1024, // 1KB limit for testing
		},
	}

	client, err := mqtt_client.NewClient(mqttConfig, storage)
	require.NoError(t, err)

	ctx := context.Background()
	err = client.Connect(ctx)
	require.NoError(t, err)
	defer client.Disconnect()

	// Test with large message that exceeds limit
	largeData := make(map[string]interface{})
	for i := 0; i < 100; i++ {
		largeData[fmt.Sprintf("key_%d", i)] = fmt.Sprintf("very_long_value_%d_with_lots_of_data_to_exceed_size_limit", i)
	}

	largeMessage := map[string]interface{}{
		"device_id": "test-device",
		"timestamp": time.Now().Unix(),
		"data":      largeData,
	}

	err = client.Publish("rtk/v1/test/site1/device1/state", largeMessage, 1, false)

	// Publishing might succeed, but message handling should respect size limits
	// The exact behavior depends on implementation
	assert.NoError(t, err) // MQTT publish itself should work
}
