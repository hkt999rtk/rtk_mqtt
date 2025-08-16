package config

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// Manager handles configuration management with hot-reload support
type Manager struct {
	config      *Config
	configPath  string
	mu          sync.RWMutex
	watchers    []func(*Config)
	watcher     *fsnotify.Watcher
	ctx         context.Context
	cancel      context.CancelFunc
}

// NewManager creates a new configuration manager
func NewManager(configPath string) (*Manager, error) {
	ctx, cancel := context.WithCancel(context.Background())
	
	manager := &Manager{
		configPath: configPath,
		watchers:   make([]func(*Config), 0),
		ctx:        ctx,
		cancel:     cancel,
	}
	
	// Load initial configuration
	config, err := LoadConfig(configPath)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to load initial config: %w", err)
	}
	
	manager.config = config
	
	// Setup file watcher
	if err := manager.setupWatcher(); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to setup config watcher: %w", err)
	}
	
	return manager, nil
}

// GetConfig returns a copy of the current configuration
func (m *Manager) GetConfig() *Config {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	// Return a deep copy to prevent external modifications
	return m.copyConfig(m.config)
}

// SetConfig updates the configuration (for testing purposes)
func (m *Manager) SetConfig(config *Config) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.config = config
	m.notifyWatchers(config)
}

// ReloadConfig reloads configuration from file
func (m *Manager) ReloadConfig() error {
	log.Info("Reloading configuration...")
	
	config, err := LoadConfig(m.configPath)
	if err != nil {
		return fmt.Errorf("failed to reload config: %w", err)
	}
	
	// Validate configuration
	if err := m.ValidateConfig(config); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}
	
	m.mu.Lock()
	oldConfig := m.config
	m.config = config
	m.mu.Unlock()
	
	log.Info("Configuration reloaded successfully")
	
	// Notify watchers of the change
	m.notifyWatchers(config)
	
	// Log configuration changes
	m.logConfigChanges(oldConfig, config)
	
	return nil
}

// UpdateConfigValue updates a single configuration value
func (m *Manager) UpdateConfigValue(key string, value interface{}) error {
	log.WithFields(log.Fields{
		"key":   key,
		"value": value,
	}).Info("Updating configuration value")
	
	// Use viper to set the value
	viper.Set(key, value)
	
	// Reload the full configuration
	return m.ReloadConfig()
}

// ValidateConfig validates configuration structure and values
func (m *Manager) ValidateConfig(config *Config) error {
	// Validate MQTT configuration
	if config.MQTT.Broker == "" {
		return fmt.Errorf("MQTT broker cannot be empty")
	}
	
	if config.MQTT.Port <= 0 || config.MQTT.Port > 65535 {
		return fmt.Errorf("MQTT port must be between 1 and 65535")
	}
	
	if config.MQTT.ClientID == "" {
		return fmt.Errorf("MQTT client ID cannot be empty")
	}
	
	// Validate retention settings
	if config.MQTT.Logging.RetentionSeconds < 0 {
		return fmt.Errorf("MQTT logging retention seconds cannot be negative")
	}
	
	// API and Console validation removed - using CLI only
	
	// Validate storage configuration
	if config.Storage.Path == "" {
		return fmt.Errorf("storage path cannot be empty")
	}
	
	// Validate schema configuration
	if config.Schema.Enabled && config.Schema.CacheSize < 0 {
		return fmt.Errorf("schema cache size cannot be negative")
	}
	
	log.Info("Configuration validation passed")
	return nil
}

// AddWatcher adds a configuration change watcher
func (m *Manager) AddWatcher(watcher func(*Config)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.watchers = append(m.watchers, watcher)
}

// RemoveWatcher removes a configuration change watcher (not implemented for simplicity)
func (m *Manager) RemoveWatcher(watcher func(*Config)) {
	// Implementation would require comparing function pointers, which is complex
	// For now, this is a placeholder
}

// Stop stops the configuration manager
func (m *Manager) Stop() {
	log.Info("Stopping configuration manager")
	
	if m.watcher != nil {
		m.watcher.Close()
	}
	
	m.cancel()
}

// GetConfigStats returns configuration statistics
func (m *Manager) GetConfigStats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	stats := map[string]interface{}{
		"config_file":       m.configPath,
		"watchers_count":    len(m.watchers),
		"mqtt_enabled":      m.config.MQTT.Broker != "",
		"schema_enabled":    m.config.Schema.Enabled,
		"diagnosis_enabled": m.config.Diagnosis.Enabled,
	}
	
	// File info
	if info, err := os.Stat(m.configPath); err == nil {
		stats["file_size"] = info.Size()
		stats["file_modified"] = info.ModTime()
	}
	
	return stats
}

// Private methods

func (m *Manager) setupWatcher() error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create file watcher: %w", err)
	}
	
	m.watcher = watcher
	
	// Add configuration file to watcher
	if err := watcher.Add(m.configPath); err != nil {
		return fmt.Errorf("failed to add config file to watcher: %w", err)
	}
	
	// Start watching in background
	go m.watchConfigFile()
	
	log.WithField("config_file", m.configPath).Info("Configuration file watcher started")
	return nil
}

func (m *Manager) watchConfigFile() {
	for {
		select {
		case <-m.ctx.Done():
			return
			
		case event, ok := <-m.watcher.Events:
			if !ok {
				return
			}
			
			if event.Has(fsnotify.Write) {
				log.WithField("file", event.Name).Info("Configuration file changed")
				
				// Debounce multiple write events
				time.Sleep(100 * time.Millisecond)
				
				if err := m.ReloadConfig(); err != nil {
					log.WithError(err).Error("Failed to reload configuration")
				}
			}
			
		case err, ok := <-m.watcher.Errors:
			if !ok {
				return
			}
			log.WithError(err).Error("Configuration file watcher error")
		}
	}
}

func (m *Manager) notifyWatchers(config *Config) {
	for _, watcher := range m.watchers {
		go func(w func(*Config)) {
			defer func() {
				if r := recover(); r != nil {
					log.WithField("error", r).Error("Configuration watcher panic")
				}
			}()
			w(m.copyConfig(config))
		}(watcher)
	}
}

func (m *Manager) copyConfig(config *Config) *Config {
	// For simplicity, we'll create a new config by reloading from file
	// In production, you might want to implement a proper deep copy
	newConfig, _ := LoadConfig(m.configPath)
	return newConfig
}

func (m *Manager) logConfigChanges(oldConfig, newConfig *Config) {
	changes := make(map[string]interface{})
	
	// Compare key configuration values
	if oldConfig.MQTT.Broker != newConfig.MQTT.Broker {
		changes["mqtt.broker"] = map[string]string{
			"old": oldConfig.MQTT.Broker,
			"new": newConfig.MQTT.Broker,
		}
	}
	
	if oldConfig.MQTT.Port != newConfig.MQTT.Port {
		changes["mqtt.port"] = map[string]int{
			"old": oldConfig.MQTT.Port,
			"new": newConfig.MQTT.Port,
		}
	}
	
	if oldConfig.MQTT.Logging.RetentionSeconds != newConfig.MQTT.Logging.RetentionSeconds {
		changes["mqtt.logging.retention_seconds"] = map[string]int{
			"old": oldConfig.MQTT.Logging.RetentionSeconds,
			"new": newConfig.MQTT.Logging.RetentionSeconds,
		}
	}
	
	if oldConfig.Logging.Level != newConfig.Logging.Level {
		changes["logging.level"] = map[string]string{
			"old": oldConfig.Logging.Level,
			"new": newConfig.Logging.Level,
		}
		
		// Apply new log level immediately
		if level, err := log.ParseLevel(newConfig.Logging.Level); err == nil {
			log.SetLevel(level)
			log.WithField("level", newConfig.Logging.Level).Info("Log level updated")
		}
	}
	
	if len(changes) > 0 {
		log.WithField("changes", changes).Info("Configuration changes detected")
	}
}