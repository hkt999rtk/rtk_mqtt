package mqtt

import (
	"context"
	"testing"
	"time"
)

func TestSetDefaults(t *testing.T) {
	cfg := &Config{}
	SetDefaults(cfg)

	if cfg.BrokerPort != 1883 {
		t.Errorf("Expected default broker port 1883, got %d", cfg.BrokerPort)
	}

	if cfg.KeepAlive != 60*time.Second {
		t.Errorf("Expected default keep alive 60s, got %s", cfg.KeepAlive)
	}

	if cfg.ConnectTimeout != 30*time.Second {
		t.Errorf("Expected default connect timeout 30s, got %s", cfg.ConnectTimeout)
	}

	if cfg.RetryInterval != 5*time.Second {
		t.Errorf("Expected default retry interval 5s, got %s", cfg.RetryInterval)
	}

	if cfg.MaxRetryCount != 3 {
		t.Errorf("Expected default max retry count 3, got %d", cfg.MaxRetryCount)
	}

	if cfg.ClientID == "" {
		t.Error("Expected client ID to be generated")
	}
}

func TestNewRegistry(t *testing.T) {
	registry := NewRegistry()

	if registry == nil {
		t.Fatal("Expected registry to be created")
	}

	if registry.factories == nil {
		t.Error("Expected factories map to be initialized")
	}

	if len(registry.List()) != 0 {
		t.Error("Expected empty registry initially")
	}
}

func TestRegistryRegisterAndCreate(t *testing.T) {
	registry := NewRegistry()
	
	// Create a mock factory
	factory := &MockFactory{}
	
	// Register factory
	registry.Register("mock", factory)
	
	// Check if factory is registered
	factories := registry.List()
	if len(factories) != 1 {
		t.Errorf("Expected 1 factory, got %d", len(factories))
	}
	
	if factories[0] != "mock" {
		t.Errorf("Expected factory name 'mock', got '%s'", factories[0])
	}
	
	// Test creating client
	cfg := &Config{
		BrokerHost: "localhost",
		BrokerPort: 1883,
		ClientID:   "test_client",
	}
	
	client, err := registry.Create("mock", cfg)
	if err != nil {
		t.Errorf("Failed to create client: %v", err)
	}
	
	if client == nil {
		t.Error("Expected client to be created")
	}
	
	// Test creating with non-existent factory
	_, err = registry.Create("nonexistent", cfg)
	if err == nil {
		t.Error("Expected error for non-existent factory")
	}
}

func TestQoSString(t *testing.T) {
	tests := []struct {
		qos      QoS
		expected string
	}{
		{QoSAtMostOnce, "AtMostOnce"},
		{QoSAtLeastOnce, "AtLeastOnce"},
		{QoSExactlyOnce, "ExactlyOnce"},
		{QoS(99), "Unknown"},
	}

	for _, test := range tests {
		result := test.qos.String()
		if result != test.expected {
			t.Errorf("QoS(%d).String() = %s, expected %s", test.qos, result, test.expected)
		}
	}
}

func TestConnectionStatusString(t *testing.T) {
	tests := []struct {
		status   ConnectionStatus
		expected string
	}{
		{StatusDisconnected, "Disconnected"},
		{StatusConnecting, "Connecting"},
		{StatusConnected, "Connected"},
		{StatusReconnecting, "Reconnecting"},
		{StatusError, "Error"},
		{ConnectionStatus(99), "Unknown"},
	}

	for _, test := range tests {
		result := test.status.String()
		if result != test.expected {
			t.Errorf("ConnectionStatus(%d).String() = %s, expected %s", test.status, result, test.expected)
		}
	}
}

func TestErrorWithCause(t *testing.T) {
	baseErr := &Error{
		Code:    "TEST_ERROR",
		Message: "Test error message",
	}

	causeErr := &Error{
		Code:    "CAUSE_ERROR",
		Message: "Cause error message",
	}

	wrappedErr := baseErr.WithCause(causeErr)

	if wrappedErr.Code != baseErr.Code {
		t.Errorf("Expected code %s, got %s", baseErr.Code, wrappedErr.Code)
	}

	if wrappedErr.Message != baseErr.Message {
		t.Errorf("Expected message %s, got %s", baseErr.Message, wrappedErr.Message)
	}

	if wrappedErr.Cause != causeErr {
		t.Error("Expected cause to be set")
	}

	expectedError := baseErr.Message + ": " + causeErr.Error()
	if wrappedErr.Error() != expectedError {
		t.Errorf("Expected error %s, got %s", expectedError, wrappedErr.Error())
	}

	if wrappedErr.Unwrap() != causeErr {
		t.Error("Expected unwrap to return cause")
	}
}

func TestGenerateClientID(t *testing.T) {
	id1 := generateClientID()
	id2 := generateClientID()

	if id1 == id2 {
		t.Error("Expected different client IDs")
	}

	if len(id1) == 0 {
		t.Error("Expected non-empty client ID")
	}

	if !hasPrefix(id1, "rtk_client_") {
		t.Errorf("Expected client ID to start with 'rtk_client_', got %s", id1)
	}
}

// Helper function since strings.HasPrefix might not be available
func hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

// MockFactory for testing
type MockFactory struct{}

func (f *MockFactory) CreateClient(cfg *Config) (Client, error) {
	return &MockClient{config: cfg}, nil
}

func (f *MockFactory) GetName() string {
	return "mock"
}

func (f *MockFactory) GetVersion() string {
	return "1.0.0"
}

// MockClient for testing
type MockClient struct {
	config    *Config
	connected bool
}

func (c *MockClient) Connect(ctx context.Context) error {
	c.connected = true
	return nil
}

func (c *MockClient) Disconnect() error {
	c.connected = false
	return nil
}

func (c *MockClient) IsConnected() bool {
	return c.connected
}

func (c *MockClient) GetStatus() ConnectionStatus {
	if c.connected {
		return StatusConnected
	}
	return StatusDisconnected
}

func (c *MockClient) Publish(ctx context.Context, topic string, payload []byte, opts *PublishOptions) error {
	return nil
}

func (c *MockClient) PublishMessage(ctx context.Context, msg *Message) error {
	return nil
}

func (c *MockClient) Subscribe(ctx context.Context, topic string, handler MessageHandler, opts *SubscribeOptions) error {
	return nil
}

func (c *MockClient) Unsubscribe(ctx context.Context, topic string) error {
	return nil
}

func (c *MockClient) SetConnectionHandler(handler ConnectionHandler) {}

func (c *MockClient) SetDefaultMessageHandler(handler MessageHandler) {}

func (c *MockClient) GetStatistics() *Statistics {
	return &Statistics{}
}

func (c *MockClient) GetLastError() error {
	return nil
}

func (c *MockClient) GetConfig() *Config {
	return c.config
}

func (c *MockClient) UpdateConfig(cfg *Config) error {
	c.config = cfg
	return nil
}

func (c *MockClient) Start(ctx context.Context) error {
	return c.Connect(ctx)
}

func (c *MockClient) Stop() error {
	return c.Disconnect()
}