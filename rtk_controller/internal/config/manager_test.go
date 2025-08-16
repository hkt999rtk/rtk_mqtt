package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewManager(t *testing.T) {
	// Create temporary config file
	tempDir, err := os.MkdirTemp("", "config_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	configPath := filepath.Join(tempDir, "test_config.yaml")
	configContent := `
mqtt:
  broker: "localhost"
  port: 1883
  client_id: "test-client"

storage:
  path: "test_data"

logging:
  level: "info"
  format: "json"
`

	err = os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "valid config file",
			path:    configPath,
			wantErr: false,
		},
		{
			name:    "non-existent config file",
			path:    filepath.Join(tempDir, "non_existent.yaml"),
			wantErr: false, // Should work with defaults
		},
		{
			name:    "invalid path",
			path:    "/invalid/path/config.yaml",
			wantErr: false, // Should work with defaults
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager, err := NewManager(tt.path)
			
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, manager)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, manager)
				
				// Verify config was loaded
				config := manager.GetConfig()
				assert.NotNil(t, config)
				assert.NotEmpty(t, config.MQTT.Broker)
				assert.Greater(t, config.MQTT.Port, 0)
				
				// Clean up
				if manager != nil {
					manager.Stop()
				}
			}
		})
	}
}

func TestManager_GetConfig(t *testing.T) {
	// Create temporary config file
	tempDir, err := os.MkdirTemp("", "config_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	configPath := filepath.Join(tempDir, "test_config.yaml")
	configContent := `
mqtt:
  broker: "test-broker"
  port: 1884
  client_id: "test-client-123"
  username: "test-user"
  password: "test-pass"
  tls:
    enabled: true
    cert_file: "cert.pem"
    key_file: "key.pem"
  topics:
    subscribe:
      - "test/topic1"
      - "test/topic2"
  logging:
    enabled: true
    retention_seconds: 7200
    batch_size: 500

storage:
  path: "test_data"
  backup_dir: "test_backups"
  backup_count: 5

diagnosis:
  enabled: true
  default_analyzers:
    - "test_analyzer"

schema:
  enabled: true
  schema_files:
    - "test_schema.json"
  strict_validation: true

logging:
  level: "debug"
  format: "text"
  file: "test.log"
  max_size: 50
  max_backups: 3
  max_age: 7
  compress: false
  audit: true
  performance: true
`

	err = os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	manager, err := NewManager(configPath)
	require.NoError(t, err)
	defer manager.Stop()

	config := manager.GetConfig()
	require.NotNil(t, config)

	// Test MQTT config
	assert.Equal(t, "test-broker", config.MQTT.Broker)
	assert.Equal(t, 1884, config.MQTT.Port)
	assert.Equal(t, "test-client-123", config.MQTT.ClientID)
	assert.Equal(t, "test-user", config.MQTT.Username)
	assert.Equal(t, "test-pass", config.MQTT.Password)
	assert.True(t, config.MQTT.TLS.Enabled)
	assert.Equal(t, "cert.pem", config.MQTT.TLS.CertFile)
	assert.Equal(t, "key.pem", config.MQTT.TLS.KeyFile)
	assert.Equal(t, []string{"test/topic1", "test/topic2"}, config.MQTT.Topics.Subscribe)
	assert.True(t, config.MQTT.Logging.Enabled)
	assert.Equal(t, 7200, config.MQTT.Logging.RetentionSeconds)
	assert.Equal(t, 500, config.MQTT.Logging.BatchSize)

	// Test Storage config
	assert.Equal(t, "test_data", config.Storage.Path)
	assert.Equal(t, "test_backups", config.Storage.BackupDir)
	assert.Equal(t, 5, config.Storage.BackupCount)

	// Test Diagnosis config
	assert.True(t, config.Diagnosis.Enabled)
	assert.Equal(t, []string{"test_analyzer"}, config.Diagnosis.DefaultAnalyzers)

	// Test Schema config
	assert.True(t, config.Schema.Enabled)
	assert.Equal(t, []string{"test_schema.json"}, config.Schema.SchemaFiles)
	assert.True(t, config.Schema.StrictValidation)

	// Test Logging config
	assert.Equal(t, "debug", config.Logging.Level)
	assert.Equal(t, "text", config.Logging.Format)
	assert.Equal(t, "test.log", config.Logging.File)
	assert.Equal(t, 50, config.Logging.MaxSize)
	assert.Equal(t, 3, config.Logging.MaxBackups)
	assert.Equal(t, 7, config.Logging.MaxAge)
	assert.False(t, config.Logging.Compress)
	assert.True(t, config.Logging.Audit)
	assert.True(t, config.Logging.Performance)
}

func TestManager_SetConfig(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "config_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	configPath := filepath.Join(tempDir, "test_config.yaml")
	
	manager, err := NewManager(configPath)
	require.NoError(t, err)
	defer manager.Stop()

	// Create a new config
	newConfig := &Config{
		MQTT: MQTTConfig{
			Broker:   "new-broker",
			Port:     1885,
			ClientID: "new-client",
		},
		Storage: StorageConfig{
			Path: "new_data",
		},
		Logging: LoggingConfig{
			Level:  "error",
			Format: "json",
		},
	}

	// Set new config
	manager.SetConfig(newConfig)

	// Verify config was updated
	config := manager.GetConfig()
	assert.Equal(t, "new-broker", config.MQTT.Broker)
	assert.Equal(t, 1885, config.MQTT.Port)
	assert.Equal(t, "new-client", config.MQTT.ClientID)
	assert.Equal(t, "new_data", config.Storage.Path)
	assert.Equal(t, "error", config.Logging.Level)
	assert.Equal(t, "json", config.Logging.Format)
}

func TestManager_ValidateConfig(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "config_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	configPath := filepath.Join(tempDir, "test_config.yaml")
	
	manager, err := NewManager(configPath)
	require.NoError(t, err)
	defer manager.Stop()

	tests := []struct {
		name    string
		config  *Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			config: &Config{
				MQTT: MQTTConfig{
					Broker:   "localhost",
					Port:     1883,
					ClientID: "test-client",
					Logging: MQTTLogging{
						RetentionSeconds: 3600,
					},
				},
				Storage: StorageConfig{
					Path: "data",
				},
				Schema: SchemaConfig{
					CacheSize: 1000,
				},
			},
			wantErr: false,
		},
		{
			name: "empty broker",
			config: &Config{
				MQTT: MQTTConfig{
					Broker:   "",
					Port:     1883,
					ClientID: "test-client",
				},
			},
			wantErr: true,
			errMsg:  "MQTT broker cannot be empty",
		},
		{
			name: "invalid port - zero",
			config: &Config{
				MQTT: MQTTConfig{
					Broker:   "localhost",
					Port:     0,
					ClientID: "test-client",
				},
			},
			wantErr: true,
			errMsg:  "MQTT port must be between 1 and 65535",
		},
		{
			name: "invalid port - too high",
			config: &Config{
				MQTT: MQTTConfig{
					Broker:   "localhost",
					Port:     70000,
					ClientID: "test-client",
				},
			},
			wantErr: true,
			errMsg:  "MQTT port must be between 1 and 65535",
		},
		{
			name: "empty client ID",
			config: &Config{
				MQTT: MQTTConfig{
					Broker:   "localhost",
					Port:     1883,
					ClientID: "",
				},
			},
			wantErr: true,
			errMsg:  "MQTT client ID cannot be empty",
		},
		{
			name: "negative retention seconds",
			config: &Config{
				MQTT: MQTTConfig{
					Broker:   "localhost",
					Port:     1883,
					ClientID: "test-client",
					Logging: MQTTLogging{
						RetentionSeconds: -1,
					},
				},
			},
			wantErr: true,
			errMsg:  "MQTT logging retention seconds cannot be negative",
		},
		{
			name: "empty storage path",
			config: &Config{
				MQTT: MQTTConfig{
					Broker:   "localhost",
					Port:     1883,
					ClientID: "test-client",
				},
				Storage: StorageConfig{
					Path: "",
				},
			},
			wantErr: true,
			errMsg:  "storage path cannot be empty",
		},
		{
			name: "negative schema cache size",
			config: &Config{
				MQTT: MQTTConfig{
					Broker:   "localhost",
					Port:     1883,
					ClientID: "test-client",
				},
				Storage: StorageConfig{
					Path: "data",
				},
				Schema: SchemaConfig{
					Enabled:   true,
					CacheSize: -1,
				},
			},
			wantErr: true,
			errMsg:  "schema cache size cannot be negative",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := manager.ValidateConfig(tt.config)
			
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestManager_GetConfigStats(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "config_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	configPath := filepath.Join(tempDir, "test_config.yaml")
	configContent := `
mqtt:
  broker: "localhost"
  port: 1883
  client_id: "test-client"

storage:
  path: "data"

diagnosis:
  enabled: true

schema:
  enabled: false

logging:
  level: "info"
`

	err = os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	manager, err := NewManager(configPath)
	require.NoError(t, err)
	defer manager.Stop()

	stats := manager.GetConfigStats()
	require.NotNil(t, stats)

	assert.Equal(t, configPath, stats["config_file"])
	assert.Equal(t, 0, stats["watchers_count"]) // No watchers added yet
	assert.Equal(t, true, stats["mqtt_enabled"])
	assert.Equal(t, false, stats["schema_enabled"])
	assert.Equal(t, true, stats["diagnosis_enabled"])

	// Check file stats
	assert.Contains(t, stats, "file_size")
	assert.Contains(t, stats, "file_modified")
}

func TestManager_AddWatcher(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "config_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	configPath := filepath.Join(tempDir, "test_config.yaml")
	
	manager, err := NewManager(configPath)
	require.NoError(t, err)
	defer manager.Stop()

	// Test adding watchers
	var watcherCalled bool
	watcher := func(config *Config) {
		watcherCalled = true
	}

	manager.AddWatcher(watcher)

	// Check stats after adding watcher
	stats := manager.GetConfigStats()
	assert.Equal(t, 1, stats["watchers_count"])

	// Trigger watcher by setting new config
	newConfig := manager.GetConfig()
	newConfig.MQTT.ClientID = "new-client-id"
	manager.SetConfig(newConfig)

	// Give some time for the watcher to be called (it runs in goroutine)
	time.Sleep(100 * time.Millisecond)
	assert.True(t, watcherCalled)
}

func TestManager_ConfigWatcher(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping file watcher test in short mode")
	}

	tempDir, err := os.MkdirTemp("", "config_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	configPath := filepath.Join(tempDir, "test_config.yaml")
	initialConfig := `
mqtt:
  broker: "localhost"
  port: 1883
  client_id: "initial-client"

logging:
  level: "info"
`

	err = os.WriteFile(configPath, []byte(initialConfig), 0644)
	require.NoError(t, err)

	manager, err := NewManager(configPath)
	require.NoError(t, err)
	defer manager.Stop()

	// Add a watcher to detect config changes
	configChanged := make(chan *Config, 1)
	manager.AddWatcher(func(config *Config) {
		select {
		case configChanged <- config:
		default:
			// Channel full, skip
		}
	})

	// Verify initial config
	config := manager.GetConfig()
	assert.Equal(t, "initial-client", config.MQTT.ClientID)
	assert.Equal(t, "info", config.Logging.Level)

	// Update config file
	updatedConfig := `
mqtt:
  broker: "localhost"
  port: 1883
  client_id: "updated-client"

logging:
  level: "debug"
`

	err = os.WriteFile(configPath, []byte(updatedConfig), 0644)
	require.NoError(t, err)

	// Wait for file watcher to detect change and reload config
	select {
	case newConfig := <-configChanged:
		assert.Equal(t, "updated-client", newConfig.MQTT.ClientID)
		assert.Equal(t, "debug", newConfig.Logging.Level)
	case <-time.After(5 * time.Second):
		t.Fatal("Config change was not detected within timeout")
	}

	// Verify config was actually updated in manager
	currentConfig := manager.GetConfig()
	assert.Equal(t, "updated-client", currentConfig.MQTT.ClientID)
	assert.Equal(t, "debug", currentConfig.Logging.Level)
}

func TestManager_ReloadConfig(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "config_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	configPath := filepath.Join(tempDir, "test_config.yaml")
	initialConfig := `
mqtt:
  broker: "localhost"
  port: 1883
  client_id: "initial-client"

logging:
  level: "info"
`

	err = os.WriteFile(configPath, []byte(initialConfig), 0644)
	require.NoError(t, err)

	manager, err := NewManager(configPath)
	require.NoError(t, err)
	defer manager.Stop()

	// Verify initial config
	config := manager.GetConfig()
	assert.Equal(t, "initial-client", config.MQTT.ClientID)

	// Update config file
	updatedConfig := `
mqtt:
  broker: "localhost"
  port: 1883
  client_id: "reloaded-client"

logging:
  level: "debug"
`

	err = os.WriteFile(configPath, []byte(updatedConfig), 0644)
	require.NoError(t, err)

	// Manually reload config
	err = manager.ReloadConfig()
	assert.NoError(t, err)

	// Verify config was updated
	config = manager.GetConfig()
	assert.Equal(t, "reloaded-client", config.MQTT.ClientID)
	assert.Equal(t, "debug", config.Logging.Level)
}

func TestManager_UpdateConfigValue(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "config_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	configPath := filepath.Join(tempDir, "test_config.yaml")
	initialConfig := `
mqtt:
  broker: "localhost"
  port: 1883
  client_id: "test-client"

logging:
  level: "info"
`

	err = os.WriteFile(configPath, []byte(initialConfig), 0644)
	require.NoError(t, err)

	manager, err := NewManager(configPath)
	require.NoError(t, err)
	defer manager.Stop()

	// Update a single config value
	err = manager.UpdateConfigValue("logging.level", "debug")
	assert.NoError(t, err)

	// Verify value was updated
	config := manager.GetConfig()
	assert.Equal(t, "debug", config.Logging.Level)
}

func TestManager_Stop(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "config_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	configPath := filepath.Join(tempDir, "test_config.yaml")
	
	manager, err := NewManager(configPath)
	require.NoError(t, err)

	// Get initial config
	config := manager.GetConfig()
	assert.NotNil(t, config)

	// Stop the manager
	manager.Stop()

	// Verify operations still work after stop (should be graceful)
	config = manager.GetConfig()
	assert.NotNil(t, config)
}