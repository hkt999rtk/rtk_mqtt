package config

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/viper"
)

// Config 定義 wrapper 主配置結構
type Config struct {
	Wrapper WrapperConfig `mapstructure:"wrapper"`
}

// WrapperConfig 定義 wrapper 配置
type WrapperConfig struct {
	Name        string            `mapstructure:"name"`
	Version     string            `mapstructure:"version"`
	MQTT        MQTTConfig        `mapstructure:"mqtt"`
	Logging     LoggingConfig     `mapstructure:"logging"`
	Monitoring  MonitoringConfig  `mapstructure:"monitoring"`
	Registry    RegistryConfig    `mapstructure:"registry"`
	Performance PerformanceConfig `mapstructure:"performance"`
	RTK         RTKConfig         `mapstructure:"rtk"`
}

// MQTTConfig 定義 MQTT 連接配置
type MQTTConfig struct {
	Broker       string          `mapstructure:"broker"`
	ClientID     string          `mapstructure:"client_id"`
	Username     string          `mapstructure:"username"`
	Password     string          `mapstructure:"password"`
	KeepAlive    int             `mapstructure:"keep_alive"`
	CleanSession bool            `mapstructure:"clean_session"`
	TLS          TLSConfig       `mapstructure:"tls"`
	Reconnect    ReconnectConfig `mapstructure:"reconnect"`
}

// TLSConfig 定義 TLS 配置
type TLSConfig struct {
	Enabled            bool   `mapstructure:"enabled"`
	CertFile           string `mapstructure:"cert_file"`
	KeyFile            string `mapstructure:"key_file"`
	CAFile             string `mapstructure:"ca_file"`
	InsecureSkipVerify bool   `mapstructure:"insecure_skip_verify"`
}

// ReconnectConfig 定義重連配置
type ReconnectConfig struct {
	Enabled      bool          `mapstructure:"enabled"`
	InitialDelay time.Duration `mapstructure:"initial_delay"`
	MaxDelay     time.Duration `mapstructure:"max_delay"`
	Multiplier   int           `mapstructure:"multiplier"`
	MaxAttempts  int           `mapstructure:"max_attempts"`
}

// LoggingConfig 定義日志配置
type LoggingConfig struct {
	Level    string         `mapstructure:"level"`
	Format   string         `mapstructure:"format"`
	Output   string         `mapstructure:"output"`
	Rotation RotationConfig `mapstructure:"rotation"`
}

// RotationConfig 定義日志輪轉配置
type RotationConfig struct {
	Enabled    bool   `mapstructure:"enabled"`
	MaxSize    string `mapstructure:"max_size"`
	MaxAge     string `mapstructure:"max_age"`
	MaxBackups int    `mapstructure:"max_backups"`
}

// MonitoringConfig 定義監控配置
type MonitoringConfig struct {
	Enabled         bool          `mapstructure:"enabled"`
	MetricsPort     int           `mapstructure:"metrics_port"`
	HealthCheckPort int           `mapstructure:"health_check_port"`
	CollectInterval time.Duration `mapstructure:"collect_interval"`
}

// RegistryConfig 定義註冊表配置
type RegistryConfig struct {
	AutoDiscovery    bool                  `mapstructure:"auto_discovery"`
	DiscoveryTimeout time.Duration         `mapstructure:"discovery_timeout"`
	Wrappers         []WrapperRegistration `mapstructure:"wrappers"`
}

// WrapperRegistration 定義 wrapper 註冊配置
type WrapperRegistration struct {
	Name       string `mapstructure:"name"`
	Enabled    bool   `mapstructure:"enabled"`
	ConfigFile string `mapstructure:"config_file"`
	Priority   int    `mapstructure:"priority"`
}

// PerformanceConfig 定義性能配置
type PerformanceConfig struct {
	WorkerPoolSize    int                   `mapstructure:"worker_pool_size"`
	MessageBufferSize int                   `mapstructure:"message_buffer_size"`
	BatchProcessing   BatchProcessingConfig `mapstructure:"batch_processing"`
}

// BatchProcessingConfig 定義批處理配置
type BatchProcessingConfig struct {
	Enabled       bool          `mapstructure:"enabled"`
	BatchSize     int           `mapstructure:"batch_size"`
	FlushInterval time.Duration `mapstructure:"flush_interval"`
}

// RTKConfig 定義 RTK 相關配置
type RTKConfig struct {
	DefaultTenant string         `mapstructure:"default_tenant"`
	DefaultSite   string         `mapstructure:"default_site"`
	TopicPrefix   string         `mapstructure:"topic_prefix"`
	QoS           QoSConfig      `mapstructure:"qos"`
	Retained      RetainedConfig `mapstructure:"retained"`
}

// QoSConfig 定義 QoS 配置
type QoSConfig struct {
	StateMessages     byte `mapstructure:"state_messages"`
	TelemetryMessages byte `mapstructure:"telemetry_messages"`
	EventMessages     byte `mapstructure:"event_messages"`
	CommandMessages   byte `mapstructure:"command_messages"`
}

// RetainedConfig 定義 retained 配置
type RetainedConfig struct {
	StateMessages bool `mapstructure:"state_messages"`
	AttrMessages  bool `mapstructure:"attr_messages"`
	LWTMessages   bool `mapstructure:"lwt_messages"`
	Others        bool `mapstructure:"others"`
}

// Load 載入配置文件
func Load(configFile string) (*Config, error) {
	viper.SetConfigFile(configFile)

	// 設置環境變數前綴
	viper.SetEnvPrefix("RTK_WRAPPER")
	viper.AutomaticEnv()

	// 設置默認值
	setDefaults()

	// 讀取配置文件
	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", configFile, err)
	}

	// 解析配置
	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// 驗證配置
	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &config, nil
}

// setDefaults 設置默認配置值
func setDefaults() {
	// MQTT 默認值
	viper.SetDefault("wrapper.mqtt.broker", "tcp://localhost:1883")
	viper.SetDefault("wrapper.mqtt.client_id", "rtk_wrapper")
	viper.SetDefault("wrapper.mqtt.keep_alive", 60)
	viper.SetDefault("wrapper.mqtt.clean_session", true)

	// TLS 默認值
	viper.SetDefault("wrapper.mqtt.tls.enabled", false)
	viper.SetDefault("wrapper.mqtt.tls.insecure_skip_verify", false)

	// 重連默認值
	viper.SetDefault("wrapper.mqtt.reconnect.enabled", true)
	viper.SetDefault("wrapper.mqtt.reconnect.initial_delay", "1s")
	viper.SetDefault("wrapper.mqtt.reconnect.max_delay", "30s")
	viper.SetDefault("wrapper.mqtt.reconnect.multiplier", 2)
	viper.SetDefault("wrapper.mqtt.reconnect.max_attempts", 10)

	// 日志默認值
	viper.SetDefault("wrapper.logging.level", "info")
	viper.SetDefault("wrapper.logging.format", "text")
	viper.SetDefault("wrapper.logging.output", "stdout")
	viper.SetDefault("wrapper.logging.rotation.enabled", false)
	viper.SetDefault("wrapper.logging.rotation.max_size", "100MB")
	viper.SetDefault("wrapper.logging.rotation.max_age", "7d")
	viper.SetDefault("wrapper.logging.rotation.max_backups", 3)

	// 監控默認值
	viper.SetDefault("wrapper.monitoring.enabled", true)
	viper.SetDefault("wrapper.monitoring.metrics_port", 8080)
	viper.SetDefault("wrapper.monitoring.health_check_port", 8081)
	viper.SetDefault("wrapper.monitoring.collect_interval", "30s")

	// 註冊表默認值
	viper.SetDefault("wrapper.registry.auto_discovery", true)
	viper.SetDefault("wrapper.registry.discovery_timeout", "5s")

	// 性能默認值
	viper.SetDefault("wrapper.performance.worker_pool_size", 10)
	viper.SetDefault("wrapper.performance.message_buffer_size", 1000)
	viper.SetDefault("wrapper.performance.batch_processing.enabled", false)
	viper.SetDefault("wrapper.performance.batch_processing.batch_size", 50)
	viper.SetDefault("wrapper.performance.batch_processing.flush_interval", "100ms")

	// RTK 默認值
	viper.SetDefault("wrapper.rtk.default_tenant", "home")
	viper.SetDefault("wrapper.rtk.default_site", "main")
	viper.SetDefault("wrapper.rtk.topic_prefix", "rtk/v1")
	viper.SetDefault("wrapper.rtk.qos.state_messages", 1)
	viper.SetDefault("wrapper.rtk.qos.telemetry_messages", 0)
	viper.SetDefault("wrapper.rtk.qos.event_messages", 1)
	viper.SetDefault("wrapper.rtk.qos.command_messages", 2)
	viper.SetDefault("wrapper.rtk.retained.state_messages", true)
	viper.SetDefault("wrapper.rtk.retained.attr_messages", true)
	viper.SetDefault("wrapper.rtk.retained.lwt_messages", true)
	viper.SetDefault("wrapper.rtk.retained.others", false)
}

// validateConfig 驗證配置
func validateConfig(config *Config) error {
	// 驗證 MQTT broker
	if config.Wrapper.MQTT.Broker == "" {
		return fmt.Errorf("mqtt broker is required")
	}

	// 驗證 client ID
	if config.Wrapper.MQTT.ClientID == "" {
		return fmt.Errorf("mqtt client_id is required")
	}

	// 驗證日志級別
	validLogLevels := map[string]bool{
		"trace": true, "debug": true, "info": true,
		"warn": true, "error": true, "fatal": true,
	}
	if !validLogLevels[config.Wrapper.Logging.Level] {
		return fmt.Errorf("invalid log level: %s", config.Wrapper.Logging.Level)
	}

	// 驗證日志格式
	if config.Wrapper.Logging.Format != "text" && config.Wrapper.Logging.Format != "json" {
		return fmt.Errorf("invalid log format: %s (must be 'text' or 'json')", config.Wrapper.Logging.Format)
	}

	// 驗證監控端口
	if config.Wrapper.Monitoring.Enabled {
		if config.Wrapper.Monitoring.MetricsPort < 1 || config.Wrapper.Monitoring.MetricsPort > 65535 {
			return fmt.Errorf("invalid metrics port: %d", config.Wrapper.Monitoring.MetricsPort)
		}
		if config.Wrapper.Monitoring.HealthCheckPort < 1 || config.Wrapper.Monitoring.HealthCheckPort > 65535 {
			return fmt.Errorf("invalid health check port: %d", config.Wrapper.Monitoring.HealthCheckPort)
		}
	}

	// 驗證 RTK 配置
	if config.Wrapper.RTK.DefaultTenant == "" {
		return fmt.Errorf("rtk default_tenant is required")
	}
	if config.Wrapper.RTK.DefaultSite == "" {
		return fmt.Errorf("rtk default_site is required")
	}
	if config.Wrapper.RTK.TopicPrefix == "" {
		return fmt.Errorf("rtk topic_prefix is required")
	}

	return nil
}

// GetEnvOrDefault 獲取環境變數或默認值
func GetEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
