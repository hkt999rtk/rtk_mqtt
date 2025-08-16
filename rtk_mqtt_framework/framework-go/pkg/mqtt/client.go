package mqtt

import (
	"context"
	"time"
)

// Client defines the interface for MQTT client implementations
type Client interface {
	// Connection management
	Connect(ctx context.Context) error
	Disconnect() error
	IsConnected() bool
	GetStatus() ConnectionStatus

	// Message operations
	Publish(ctx context.Context, topic string, payload []byte, opts *PublishOptions) error
	PublishMessage(ctx context.Context, msg *Message) error
	Subscribe(ctx context.Context, topic string, handler MessageHandler, opts *SubscribeOptions) error
	Unsubscribe(ctx context.Context, topic string) error

	// Event handlers
	SetConnectionHandler(handler ConnectionHandler)
	SetDefaultMessageHandler(handler MessageHandler)

	// Statistics and monitoring
	GetStatistics() *Statistics
	GetLastError() error

	// Configuration
	GetConfig() *Config
	UpdateConfig(cfg *Config) error

	// Lifecycle
	Start(ctx context.Context) error
	Stop() error
}

// ClientFactory defines the interface for creating MQTT clients
type ClientFactory interface {
	CreateClient(cfg *Config) (Client, error)
	GetName() string
	GetVersion() string
}

// Registry manages available MQTT client implementations
type Registry struct {
	factories map[string]ClientFactory
}

// NewRegistry creates a new client factory registry
func NewRegistry() *Registry {
	return &Registry{
		factories: make(map[string]ClientFactory),
	}
}

// Register registers a client factory
func (r *Registry) Register(name string, factory ClientFactory) {
	r.factories[name] = factory
}

// Create creates a new client using the specified factory
func (r *Registry) Create(factoryName string, cfg *Config) (Client, error) {
	factory, exists := r.factories[factoryName]
	if !exists {
		return nil, &Error{
			Code:    "FACTORY_NOT_FOUND",
			Message: "MQTT client factory not found: " + factoryName,
		}
	}
	return factory.CreateClient(cfg)
}

// List returns the names of all registered factories
func (r *Registry) List() []string {
	names := make([]string, 0, len(r.factories))
	for name := range r.factories {
		names = append(names, name)
	}
	return names
}

// Default registry instance
var defaultRegistry = NewRegistry()

// RegisterFactory registers a client factory in the default registry
func RegisterFactory(name string, factory ClientFactory) {
	defaultRegistry.Register(name, factory)
}

// NewClient creates a new MQTT client using the default registry
func NewClient(cfg *Config) (Client, error) {
	return NewClientWithFactory("paho", cfg)
}

// NewClientWithFactory creates a new MQTT client using the specified factory
func NewClientWithFactory(factoryName string, cfg *Config) (Client, error) {
	return defaultRegistry.Create(factoryName, cfg)
}

// SetDefaults sets default values for MQTT configuration
func SetDefaults(cfg *Config) {
	if cfg.BrokerPort == 0 {
		cfg.BrokerPort = 1883
	}
	if cfg.KeepAlive == 0 {
		cfg.KeepAlive = 60 * time.Second
	}
	if cfg.ConnectTimeout == 0 {
		cfg.ConnectTimeout = 30 * time.Second
	}
	if cfg.RetryInterval == 0 {
		cfg.RetryInterval = 5 * time.Second
	}
	if cfg.MaxRetryCount == 0 {
		cfg.MaxRetryCount = 3
	}
	if cfg.ClientID == "" {
		cfg.ClientID = generateClientID()
	}
}

// generateClientID generates a unique client ID
func generateClientID() string {
	return "rtk_client_" + generateRandomString(8)
}

// generateRandomString generates a random string of specified length
func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(result)
}