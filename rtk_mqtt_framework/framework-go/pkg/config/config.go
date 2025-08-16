package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"

	"github.com/rtk/mqtt-framework/pkg/mqtt"
)

// Config represents the complete framework configuration
type Config struct {
	MQTT    mqtt.Config `json:"mqtt" yaml:"mqtt" validate:"required"`
	Device  DeviceConfig `json:"device" yaml:"device" validate:"required"`
	Logging LoggingConfig `json:"logging" yaml:"logging"`
	Plugins PluginsConfig `json:"plugins" yaml:"plugins"`
}

// DeviceConfig represents device-specific configuration
type DeviceConfig struct {
	Type     string `json:"type" yaml:"type" validate:"required"`
	Tenant   string `json:"tenant" yaml:"tenant" validate:"required"`
	Site     string `json:"site" yaml:"site" validate:"required"`
	DeviceID string `json:"device_id" yaml:"device_id" validate:"required"`
	
	// Device metadata
	Name        string            `json:"name,omitempty" yaml:"name,omitempty"`
	Description string            `json:"description,omitempty" yaml:"description,omitempty"`
	Version     string            `json:"version,omitempty" yaml:"version,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	
	// Reporting intervals
	StateInterval      time.Duration `json:"state_interval" yaml:"state_interval"`
	TelemetryInterval  time.Duration `json:"telemetry_interval" yaml:"telemetry_interval"`
	HeartbeatInterval  time.Duration `json:"heartbeat_interval" yaml:"heartbeat_interval"`
}

// LoggingConfig represents logging configuration
type LoggingConfig struct {
	Level  string `json:"level" yaml:"level"`
	Format string `json:"format" yaml:"format"` // json, text
	Output string `json:"output" yaml:"output"` // stdout, stderr, file path
}

// PluginsConfig represents plugin configuration
type PluginsConfig struct {
	Directory string                 `json:"directory" yaml:"directory"`
	Enabled   []string               `json:"enabled" yaml:"enabled"`
	Config    map[string]interface{} `json:"config" yaml:"config"`
}

// Manager handles configuration loading and validation
type Manager struct {
	validator *validator.Validate
}

// NewManager creates a new configuration manager
func NewManager() *Manager {
	return &Manager{
		validator: validator.New(),
	}
}

// Load loads configuration from various sources
func (m *Manager) Load() (*Config, error) {
	v := viper.New()
	
	// Set defaults
	m.setDefaults(v)
	
	// Configure viper
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	v.AddConfigPath("./config")
	v.AddConfigPath("/etc/rtk-mqtt")
	v.AddConfigPath("$HOME/.rtk-mqtt")
	
	// Environment variables
	v.SetEnvPrefix("RTK")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()
	
	// Read configuration file
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	}
	
	// Unmarshal into struct
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}
	
	// Validate configuration
	if err := m.validator.Struct(&cfg); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}
	
	return &cfg, nil
}

// LoadFromFile loads configuration from a specific file
func (m *Manager) LoadFromFile(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", filename, err)
	}
	
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file %s: %w", filename, err)
	}
	
	// Set defaults
	m.setConfigDefaults(&cfg)
	
	// Validate configuration
	if err := m.validator.Struct(&cfg); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}
	
	return &cfg, nil
}

// LoadFromBytes loads configuration from byte data
func (m *Manager) LoadFromBytes(data []byte) (*Config, error) {
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}
	
	// Set defaults
	m.setConfigDefaults(&cfg)
	
	// Validate configuration
	if err := m.validator.Struct(&cfg); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}
	
	return &cfg, nil
}

// Save saves configuration to a file
func (m *Manager) Save(cfg *Config, filename string) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	
	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file %s: %w", filename, err)
	}
	
	return nil
}

// setDefaults sets default values using viper
func (m *Manager) setDefaults(v *viper.Viper) {
	// MQTT defaults
	v.SetDefault("mqtt.broker_port", 1883)
	v.SetDefault("mqtt.keep_alive", "60s")
	v.SetDefault("mqtt.clean_session", true)
	v.SetDefault("mqtt.connect_timeout", "30s")
	v.SetDefault("mqtt.retry_interval", "5s")
	v.SetDefault("mqtt.max_retry_count", 3)
	
	// Device defaults
	v.SetDefault("device.state_interval", "30s")
	v.SetDefault("device.telemetry_interval", "60s")
	v.SetDefault("device.heartbeat_interval", "300s")
	
	// Logging defaults
	v.SetDefault("logging.level", "info")
	v.SetDefault("logging.format", "text")
	v.SetDefault("logging.output", "stdout")
	
	// Plugins defaults
	v.SetDefault("plugins.directory", "./plugins")
}

// setConfigDefaults sets default values directly on config struct
func (m *Manager) setConfigDefaults(cfg *Config) {
	// MQTT defaults
	mqtt.SetDefaults(&cfg.MQTT)
	
	// Device defaults
	if cfg.Device.StateInterval == 0 {
		cfg.Device.StateInterval = 30 * time.Second
	}
	if cfg.Device.TelemetryInterval == 0 {
		cfg.Device.TelemetryInterval = 60 * time.Second
	}
	if cfg.Device.HeartbeatInterval == 0 {
		cfg.Device.HeartbeatInterval = 300 * time.Second
	}
	
	// Logging defaults
	if cfg.Logging.Level == "" {
		cfg.Logging.Level = "info"
	}
	if cfg.Logging.Format == "" {
		cfg.Logging.Format = "text"
	}
	if cfg.Logging.Output == "" {
		cfg.Logging.Output = "stdout"
	}
	
	// Plugins defaults
	if cfg.Plugins.Directory == "" {
		cfg.Plugins.Directory = "./plugins"
	}
}

// Convenience functions

// Load loads configuration using the default manager
func Load() (*Config, error) {
	manager := NewManager()
	return manager.Load()
}

// LoadFromFile loads configuration from a file using the default manager
func LoadFromFile(filename string) (*Config, error) {
	manager := NewManager()
	return manager.LoadFromFile(filename)
}

// LoadFromBytes loads configuration from bytes using the default manager
func LoadFromBytes(data []byte) (*Config, error) {
	manager := NewManager()
	return manager.LoadFromBytes(data)
}

// Save saves configuration to a file using the default manager
func Save(cfg *Config, filename string) error {
	manager := NewManager()
	return manager.Save(cfg, filename)
}