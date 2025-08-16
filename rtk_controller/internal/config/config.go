package config

import (
	"fmt"
	"os"

	"github.com/spf13/viper"
)

// Config represents the complete configuration structure
type Config struct {
	MQTT      MQTTConfig      `mapstructure:"mqtt"`
	Storage   StorageConfig   `mapstructure:"storage"`
	Diagnosis DiagnosisConfig `mapstructure:"diagnosis"`
	Schema    SchemaConfig    `mapstructure:"schema"`
	Logging   LoggingConfig   `mapstructure:"logging"`
}

// MQTTConfig holds MQTT client configuration
type MQTTConfig struct {
	Broker   string       `mapstructure:"broker"`
	Port     int          `mapstructure:"port"`
	ClientID string       `mapstructure:"client_id"`
	Username string       `mapstructure:"username"`
	Password string       `mapstructure:"password"`
	TLS      TLSConfig    `mapstructure:"tls"`
	Topics   TopicsConfig `mapstructure:"topics"`
	Logging  MQTTLogging  `mapstructure:"logging"`
}

// TLSConfig holds TLS configuration
type TLSConfig struct {
	Enabled    bool   `mapstructure:"enabled"`
	CertFile   string `mapstructure:"cert_file"`
	KeyFile    string `mapstructure:"key_file"`
	CAFile     string `mapstructure:"ca_file"`
	SkipVerify bool   `mapstructure:"skip_verify"`
}

// TopicsConfig holds topic subscription configuration
type TopicsConfig struct {
	Subscribe []string `mapstructure:"subscribe"`
}

// MQTTLogging holds MQTT message logging configuration
type MQTTLogging struct {
	Enabled           bool     `mapstructure:"enabled"`
	RetentionSeconds  int      `mapstructure:"retention_seconds"`
	PurgeInterval     string   `mapstructure:"purge_interval"`
	BatchSize         int      `mapstructure:"batch_size"`
	MaxMessageSize    int      `mapstructure:"max_message_size"`
	ExcludeTopics     []string `mapstructure:"exclude_topics"`
}

// API and Console configs removed - using CLI only

// StorageConfig holds database configuration
type StorageConfig struct {
	Path        string `mapstructure:"path"`
	BackupDir   string `mapstructure:"backup_dir"`
	BackupCount int    `mapstructure:"backup_count"`
}

// DiagnosisConfig holds diagnosis system configuration
type DiagnosisConfig struct {
	Enabled          bool             `mapstructure:"enabled"`
	DefaultAnalyzers []string         `mapstructure:"default_analyzers"`
	Analyzers        []AnalyzerConfig `mapstructure:"analyzers"`
	Routing          RoutingConfig    `mapstructure:"routing"`
}

// SchemaConfig holds JSON schema validation configuration
type SchemaConfig struct {
	Enabled              bool     `mapstructure:"enabled"`
	SchemaFiles          []string `mapstructure:"schema_files"`
	StrictValidation     bool     `mapstructure:"strict_validation"`
	LogValidationErrors  bool     `mapstructure:"log_validation_errors"`
	CacheResults         bool     `mapstructure:"cache_results"`
	CacheSize            int      `mapstructure:"cache_size"`
	StoreResults         bool     `mapstructure:"store_results"`
}

// AnalyzerConfig holds individual analyzer configuration
type AnalyzerConfig struct {
	Name    string                 `mapstructure:"name"`
	Type    string                 `mapstructure:"type"`
	Path    string                 `mapstructure:"path"`
	Command string                 `mapstructure:"command"`
	Args    []string               `mapstructure:"args"`
	Timeout string                 `mapstructure:"timeout"`
	Enabled bool                   `mapstructure:"enabled"`
	Config  map[string]interface{} `mapstructure:"config"`
}

// RoutingConfig holds event routing configuration
type RoutingConfig struct {
	Rules map[string][]string `mapstructure:"rules"`
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Level      string `mapstructure:"level"`
	Format     string `mapstructure:"format"`
	File       string `mapstructure:"file"`
	MaxSize    int    `mapstructure:"max_size"`    // MB
	MaxBackups int    `mapstructure:"max_backups"` // number of backups
	MaxAge     int    `mapstructure:"max_age"`     // days
	Compress   bool   `mapstructure:"compress"`    // compress rotated files
	Audit      bool   `mapstructure:"audit"`       // enable audit logging
	Performance bool  `mapstructure:"performance"` // enable performance logging
}

// LoadConfig loads configuration from file
func LoadConfig(configPath string) (*Config, error) {
	// Set default values
	viper.SetDefault("mqtt.broker", "localhost")
	viper.SetDefault("mqtt.port", 1883)
	viper.SetDefault("mqtt.client_id", "rtk-controller")
	viper.SetDefault("mqtt.topics.subscribe", []string{
		"rtk/v1/+/+/+/state",
		"rtk/v1/+/+/+/evt/#",
		"rtk/v1/+/+/+/lwt",
		"rtk/v1/+/+/+/cmd/ack",
		"rtk/v1/+/+/+/cmd/res",
		"rtk/v1/+/+/+/attr",
	})
	viper.SetDefault("mqtt.logging.enabled", true)
	viper.SetDefault("mqtt.logging.retention_seconds", 3600)
	viper.SetDefault("mqtt.logging.purge_interval", "1m")
	viper.SetDefault("mqtt.logging.batch_size", 1000)
	viper.SetDefault("mqtt.logging.max_message_size", 1048576)
	viper.SetDefault("mqtt.logging.exclude_topics", []string{
		"rtk/v1/+/+/+/telemetry/heartbeat",
		"rtk/v1/+/+/+/internal/#",
	})

	// API and Console defaults removed

	viper.SetDefault("storage.path", "data")
	viper.SetDefault("storage.backup_dir", "data/backups")
	viper.SetDefault("storage.backup_count", 7)

	viper.SetDefault("diagnosis.enabled", true)
	viper.SetDefault("diagnosis.default_analyzers", []string{
		"wifi_analyzer",
		"network_analyzer", 
		"system_analyzer",
	})

	viper.SetDefault("schema.enabled", true)
	viper.SetDefault("schema.schema_files", []string{
		"wifi_diagnosis_schemas.json",
	})
	viper.SetDefault("schema.strict_validation", false)
	viper.SetDefault("schema.log_validation_errors", true)
	viper.SetDefault("schema.cache_results", true)
	viper.SetDefault("schema.cache_size", 1000)
	viper.SetDefault("schema.store_results", false)

	viper.SetDefault("logging.level", "info")
	viper.SetDefault("logging.format", "json")
	viper.SetDefault("logging.file", "logs/controller.log")
	viper.SetDefault("logging.max_size", 100)
	viper.SetDefault("logging.max_backups", 7)
	viper.SetDefault("logging.max_age", 30)
	viper.SetDefault("logging.compress", true)
	viper.SetDefault("logging.audit", true)
	viper.SetDefault("logging.performance", false)

	// Configure viper
	viper.SetConfigFile(configPath)
	viper.AutomaticEnv()

	// Read configuration file if it exists
	if _, err := os.Stat(configPath); err == nil {
		if err := viper.ReadInConfig(); err != nil {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	// Unmarshal to struct
	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &config, nil
}

// GetAnalyzerConfig returns configuration for a specific analyzer
func (c *Config) GetAnalyzerConfig(name string) map[string]interface{} {
	for _, analyzer := range c.Diagnosis.Analyzers {
		if analyzer.Name == name {
			return analyzer.Config
		}
	}
	return make(map[string]interface{})
}