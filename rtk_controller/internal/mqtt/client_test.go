package mqtt

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	mqtt_client "github.com/eclipse/paho.mqtt.golang"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"rtk_controller/internal/config"
	"rtk_controller/internal/storage"
	"rtk_controller/pkg/types"
)

// MockMQTTClient implements mqtt.Client interface for testing
type MockMQTTClient struct {
	mock.Mock
	connected bool
}

func (m *MockMQTTClient) IsConnected() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *MockMQTTClient) IsConnectionOpen() bool {
	return m.connected
}

func (m *MockMQTTClient) Connect() mqtt_client.Token {
	args := m.Called()
	m.connected = true
	return args.Get(0).(mqtt_client.Token)
}

func (m *MockMQTTClient) Disconnect(quiesce uint) {
	m.Called(quiesce)
	m.connected = false
}

func (m *MockMQTTClient) Publish(topic string, qos byte, retained bool, payload interface{}) mqtt_client.Token {
	args := m.Called(topic, qos, retained, payload)
	return args.Get(0).(mqtt_client.Token)
}

func (m *MockMQTTClient) Subscribe(topic string, qos byte, callback mqtt_client.MessageHandler) mqtt_client.Token {
	args := m.Called(topic, qos, callback)
	return args.Get(0).(mqtt_client.Token)
}

func (m *MockMQTTClient) SubscribeMultiple(filters map[string]byte, callback mqtt_client.MessageHandler) mqtt_client.Token {
	args := m.Called(filters, callback)
	return args.Get(0).(mqtt_client.Token)
}

func (m *MockMQTTClient) Unsubscribe(topics ...string) mqtt_client.Token {
	args := m.Called(topics)
	return args.Get(0).(mqtt_client.Token)
}

func (m *MockMQTTClient) AddRoute(topic string, callback mqtt_client.MessageHandler) {
	m.Called(topic, callback)
}

func (m *MockMQTTClient) OptionsReader() mqtt_client.ClientOptionsReader {
	args := m.Called()
	return args.Get(0).(mqtt_client.ClientOptionsReader)
}

// MockToken implements mqtt.Token interface
type MockToken struct {
	mock.Mock
}

func (m *MockToken) Wait() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *MockToken) WaitTimeout(time.Duration) bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *MockToken) Done() <-chan struct{} {
	args := m.Called()
	return args.Get(0).(<-chan struct{})
}

func (m *MockToken) Error() error {
	args := m.Called()
	return args.Error(0)
}

// MockStorage implements storage interface for testing
type MockStorage struct {
	mock.Mock
}

func (m *MockStorage) Set(key string, value interface{}) error {
	args := m.Called(key, value)
	return args.Error(0)
}

func (m *MockStorage) Get(key string, result interface{}) error {
	args := m.Called(key, result)
	return args.Error(0)
}

func (m *MockStorage) Delete(key string) error {
	args := m.Called(key)
	return args.Error(0)
}

func (m *MockStorage) List(pattern string) ([]string, error) {
	args := m.Called(pattern)
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockStorage) Close() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockStorage) CreateIndex(name, pattern string, less func(a, b string) bool) error {
	args := m.Called(name, pattern, less)
	return args.Error(0)
}

func (m *MockStorage) View(fn func(storage.Transaction) error) error {
	args := m.Called(fn)
	return args.Error(0)
}

func (m *MockStorage) Update(fn func(storage.Transaction) error) error {
	args := m.Called(fn)
	return args.Error(0)
}

func TestClient_NewClient(t *testing.T) {
	tests := []struct {
		name    string
		config  config.MQTTConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: config.MQTTConfig{
				Broker:   "localhost",
				Port:     1883,
				ClientID: "test-client",
				Topics: config.TopicsConfig{
					Subscribe: []string{"test/topic"},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid broker",
			config: config.MQTTConfig{
				Broker:   "",
				Port:     1883,
				ClientID: "test-client",
			},
			wantErr: true,
		},
		{
			name: "invalid port",
			config: config.MQTTConfig{
				Broker:   "localhost",
				Port:     0,
				ClientID: "test-client",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStorage := &MockStorage{}

			client, err := NewClient(tt.config, mockStorage)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, client)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, client)
				assert.Equal(t, tt.config.Broker, client.config.Broker)
				assert.Equal(t, tt.config.Port, client.config.Port)
				assert.Equal(t, tt.config.ClientID, client.config.ClientID)
			}
		})
	}
}

func TestClient_Connect(t *testing.T) {
	mockStorage := &MockStorage{}
	mockToken := &MockToken{}

	config := config.MQTTConfig{
		Broker:   "localhost",
		Port:     1883,
		ClientID: "test-client",
		Topics: config.TopicsConfig{
			Subscribe: []string{"test/topic"},
		},
	}

	client, err := NewClient(config, mockStorage)
	assert.NoError(t, err)

	// Replace the MQTT client with our mock
	mockMQTTClient := &MockMQTTClient{}
	client.client = mockMQTTClient

	mockToken.On("Wait").Return(true)
	mockToken.On("Error").Return(nil)
	mockMQTTClient.On("Connect").Return(mockToken)
	mockMQTTClient.On("IsConnected").Return(true)
	mockMQTTClient.On("Subscribe", "test/topic", byte(0), mock.AnythingOfType("mqtt.MessageHandler")).Return(mockToken)

	ctx := context.Background()
	err = client.Connect(ctx)

	assert.NoError(t, err)
	mockMQTTClient.AssertExpectations(t)
	mockToken.AssertExpectations(t)
}

func TestClient_Publish(t *testing.T) {
	mockStorage := &MockStorage{}
	mockToken := &MockToken{}

	config := config.MQTTConfig{
		Broker:   "localhost",
		Port:     1883,
		ClientID: "test-client",
	}

	client, err := NewClient(config, mockStorage)
	assert.NoError(t, err)

	// Replace the MQTT client with our mock
	mockMQTTClient := &MockMQTTClient{}
	client.client = mockMQTTClient

	testPayload := map[string]interface{}{
		"test": "data",
	}
	expectedPayload, _ := json.Marshal(testPayload)

	mockToken.On("Wait").Return(true)
	mockToken.On("Error").Return(nil)
	mockMQTTClient.On("Publish", "test/topic", byte(0), false, string(expectedPayload)).Return(mockToken)

	err = client.Publish("test/topic", testPayload, 0, false)

	assert.NoError(t, err)
	mockMQTTClient.AssertExpectations(t)
	mockToken.AssertExpectations(t)
}

func TestClient_IsConnected(t *testing.T) {
	mockStorage := &MockStorage{}

	config := config.MQTTConfig{
		Broker:   "localhost",
		Port:     1883,
		ClientID: "test-client",
	}

	client, err := NewClient(config, mockStorage)
	assert.NoError(t, err)

	// Replace the MQTT client with our mock
	mockMQTTClient := &MockMQTTClient{}
	client.client = mockMQTTClient

	tests := []struct {
		name     string
		expected bool
	}{
		{
			name:     "connected",
			expected: true,
		},
		{
			name:     "disconnected",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockMQTTClient.On("IsConnected").Return(tt.expected).Once()

			result := client.IsConnected()

			assert.Equal(t, tt.expected, result)
		})
	}

	mockMQTTClient.AssertExpectations(t)
}

func TestClient_MessageLogging(t *testing.T) {
	mockStorage := &MockStorage{}

	config := config.MQTTConfig{
		Broker:   "localhost",
		Port:     1883,
		ClientID: "test-client",
		Logging: config.MQTTLogging{
			Enabled:          true,
			RetentionSeconds: 3600,
			BatchSize:        100,
		},
	}

	client, err := NewClient(config, mockStorage)
	assert.NoError(t, err)
	assert.NotNil(t, client.messageLogger)

	// Test message logger statistics
	stats := client.GetMessageLogger().GetStats()
	assert.NotNil(t, stats)
	assert.Equal(t, int64(0), stats.TotalMessages)
	assert.Equal(t, int64(0), stats.PurgedMessages)
}

func TestClient_Disconnect(t *testing.T) {
	mockStorage := &MockStorage{}

	config := config.MQTTConfig{
		Broker:   "localhost",
		Port:     1883,
		ClientID: "test-client",
	}

	client, err := NewClient(config, mockStorage)
	assert.NoError(t, err)

	// Replace the MQTT client with our mock
	mockMQTTClient := &MockMQTTClient{}
	client.client = mockMQTTClient

	mockMQTTClient.On("Disconnect", uint(250)).Return()

	client.Disconnect()

	mockMQTTClient.AssertExpectations(t)
}

func TestClient_ValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  config.MQTTConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			config: config.MQTTConfig{
				Broker:   "localhost",
				Port:     1883,
				ClientID: "test-client",
			},
			wantErr: false,
		},
		{
			name: "empty broker",
			config: config.MQTTConfig{
				Broker:   "",
				Port:     1883,
				ClientID: "test-client",
			},
			wantErr: true,
			errMsg:  "broker cannot be empty",
		},
		{
			name: "invalid port - zero",
			config: config.MQTTConfig{
				Broker:   "localhost",
				Port:     0,
				ClientID: "test-client",
			},
			wantErr: true,
			errMsg:  "port must be between 1 and 65535",
		},
		{
			name: "invalid port - too high",
			config: config.MQTTConfig{
				Broker:   "localhost",
				Port:     70000,
				ClientID: "test-client",
			},
			wantErr: true,
			errMsg:  "port must be between 1 and 65535",
		},
		{
			name: "empty client ID",
			config: config.MQTTConfig{
				Broker:   "localhost",
				Port:     1883,
				ClientID: "",
			},
			wantErr: true,
			errMsg:  "client ID cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStorage := &MockStorage{}

			_, err := NewClient(tt.config, mockStorage)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
