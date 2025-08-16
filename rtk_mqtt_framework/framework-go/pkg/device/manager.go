package device

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// Manager manages device plugins and their lifecycle
type Manager struct {
	plugins   map[string]*PluginInstance
	factories map[string]PluginFactory
	mutex     sync.RWMutex
	logger    *logrus.Logger
	ctx       context.Context
	cancel    context.CancelFunc

	// Event handling
	eventHandlers []EventHandler
	eventQueue    chan *Event
	eventWorkers  int

	// Plugin monitoring
	healthChecker *HealthChecker
	metricsCollector *MetricsCollector
}

// PluginInstance represents a running plugin instance
type PluginInstance struct {
	Plugin   Plugin
	Config   *PluginConfig
	Info     *Info
	State    PluginState
	Metrics  *PluginMetrics
	StartTime time.Time
	mutex    sync.RWMutex
}

// PluginState represents the current state of a plugin
type PluginState int

const (
	StateUnknown PluginState = iota
	StateInitialized
	StateStarting
	StateRunning
	StateStopping
	StateStopped
	StateError
)

// String returns the string representation of plugin state
func (s PluginState) String() string {
	switch s {
	case StateUnknown:
		return "Unknown"
	case StateInitialized:
		return "Initialized"
	case StateStarting:
		return "Starting"
	case StateRunning:
		return "Running"
	case StateStopping:
		return "Stopping"
	case StateStopped:
		return "Stopped"
	case StateError:
		return "Error"
	default:
		return "Invalid"
	}
}

// ManagerConfig represents manager configuration
type ManagerConfig struct {
	EventWorkers     int           `json:"event_workers"`
	EventQueueSize   int           `json:"event_queue_size"`
	HealthCheckInterval time.Duration `json:"health_check_interval"`
	MetricsInterval  time.Duration `json:"metrics_interval"`
	PluginTimeout    time.Duration `json:"plugin_timeout"`
}

// NewManager creates a new plugin manager
func NewManager(config *ManagerConfig) *Manager {
	if config == nil {
		config = &ManagerConfig{
			EventWorkers:     4,
			EventQueueSize:   1000,
			HealthCheckInterval: 30 * time.Second,
			MetricsInterval:  60 * time.Second,
			PluginTimeout:    30 * time.Second,
		}
	}

	ctx, cancel := context.WithCancel(context.Background())

	manager := &Manager{
		plugins:       make(map[string]*PluginInstance),
		factories:     make(map[string]PluginFactory),
		logger:        logrus.New(),
		ctx:           ctx,
		cancel:        cancel,
		eventHandlers: make([]EventHandler, 0),
		eventQueue:    make(chan *Event, config.EventQueueSize),
		eventWorkers:  config.EventWorkers,
	}

	// Initialize health checker
	manager.healthChecker = NewHealthChecker(config.HealthCheckInterval)
	
	// Initialize metrics collector
	manager.metricsCollector = NewMetricsCollector(config.MetricsInterval)

	return manager
}

// RegisterFactory registers a plugin factory
func (m *Manager) RegisterFactory(name string, factory PluginFactory) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if _, exists := m.factories[name]; exists {
		return fmt.Errorf("plugin factory %s already registered", name)
	}

	m.factories[name] = factory
	m.logger.WithField("factory", name).Info("Plugin factory registered")
	return nil
}

// CreatePlugin creates a new plugin instance
func (m *Manager) CreatePlugin(name string, config *PluginConfig) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Check if plugin already exists
	if _, exists := m.plugins[name]; exists {
		return fmt.Errorf("plugin %s already exists", name)
	}

	// Find factory
	factory, exists := m.factories[config.Type]
	if !exists {
		return fmt.Errorf("plugin factory %s not found", config.Type)
	}

	// Create plugin instance
	plugin, err := factory.CreatePlugin()
	if err != nil {
		return fmt.Errorf("failed to create plugin %s: %w", name, err)
	}

	// Initialize plugin
	ctx, cancel := context.WithTimeout(m.ctx, 30*time.Second)
	defer cancel()

	if err := plugin.Initialize(ctx, config.Config); err != nil {
		return fmt.Errorf("failed to initialize plugin %s: %w", name, err)
	}

	// Create plugin instance
	instance := &PluginInstance{
		Plugin:    plugin,
		Config:    config,
		Info:      plugin.GetInfo(),
		State:     StateInitialized,
		StartTime: time.Now(),
		Metrics: &PluginMetrics{
			StartTime: time.Now(),
		},
	}

	m.plugins[name] = instance
	m.logger.WithField("plugin", name).Info("Plugin created")
	return nil
}

// StartPlugin starts a plugin
func (m *Manager) StartPlugin(name string) error {
	m.mutex.Lock()
	instance, exists := m.plugins[name]
	if !exists {
		m.mutex.Unlock()
		return fmt.Errorf("plugin %s not found", name)
	}
	m.mutex.Unlock()

	instance.mutex.Lock()
	defer instance.mutex.Unlock()

	if instance.State == StateRunning {
		return fmt.Errorf("plugin %s is already running", name)
	}

	instance.State = StateStarting

	ctx, cancel := context.WithTimeout(m.ctx, 30*time.Second)
	defer cancel()

	if err := instance.Plugin.Start(ctx); err != nil {
		instance.State = StateError
		return fmt.Errorf("failed to start plugin %s: %w", name, err)
	}

	instance.State = StateRunning
	instance.StartTime = time.Now()
	m.logger.WithField("plugin", name).Info("Plugin started")
	return nil
}

// StopPlugin stops a plugin
func (m *Manager) StopPlugin(name string) error {
	m.mutex.Lock()
	instance, exists := m.plugins[name]
	if !exists {
		m.mutex.Unlock()
		return fmt.Errorf("plugin %s not found", name)
	}
	m.mutex.Unlock()

	return m.stopPluginInternal(instance, name)
}

// stopPluginInternal stops a plugin without acquiring the manager mutex
func (m *Manager) stopPluginInternal(instance *PluginInstance, name string) error {
	instance.mutex.Lock()
	defer instance.mutex.Unlock()

	if instance.State != StateRunning {
		return fmt.Errorf("plugin %s is not running", name)
	}

	instance.State = StateStopping

	if err := instance.Plugin.Stop(); err != nil {
		instance.State = StateError
		return fmt.Errorf("failed to stop plugin %s: %w", name, err)
	}

	instance.State = StateStopped
	m.logger.WithField("plugin", name).Info("Plugin stopped")
	return nil
}

// RemovePlugin removes a plugin
func (m *Manager) RemovePlugin(name string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	instance, exists := m.plugins[name]
	if !exists {
		return fmt.Errorf("plugin %s not found", name)
	}

	// Stop plugin if running
	if instance.State == StateRunning {
		if err := m.stopPluginInternal(instance, name); err != nil {
			m.logger.WithError(err).WithField("plugin", name).Warn("Failed to stop plugin during removal")
		}
	}

	delete(m.plugins, name)
	m.logger.WithField("plugin", name).Info("Plugin removed")
	return nil
}

// GetPlugin returns a plugin instance
func (m *Manager) GetPlugin(name string) (*PluginInstance, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	instance, exists := m.plugins[name]
	if !exists {
		return nil, fmt.Errorf("plugin %s not found", name)
	}

	return instance, nil
}

// ListPlugins returns all plugin instances
func (m *Manager) ListPlugins() map[string]*PluginInstance {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	result := make(map[string]*PluginInstance)
	for name, instance := range m.plugins {
		result[name] = instance
	}
	return result
}

// GetPluginState returns the state of a plugin
func (m *Manager) GetPluginState(name string) (PluginState, error) {
	instance, err := m.GetPlugin(name)
	if err != nil {
		return StateUnknown, err
	}

	instance.mutex.RLock()
	defer instance.mutex.RUnlock()
	return instance.State, nil
}

// GetPluginInfo returns information about a plugin
func (m *Manager) GetPluginInfo(name string) (*Info, error) {
	instance, err := m.GetPlugin(name)
	if err != nil {
		return nil, err
	}

	return instance.Info, nil
}

// HandleCommand sends a command to a plugin
func (m *Manager) HandleCommand(name string, cmd *Command) (*CommandResponse, error) {
	instance, err := m.GetPlugin(name)
	if err != nil {
		return nil, err
	}

	instance.mutex.RLock()
	if instance.State != StateRunning {
		instance.mutex.RUnlock()
		return &CommandResponse{
			CommandID: cmd.ID,
			Status:    "error",
			Error:     "plugin is not running",
			Timestamp: time.Now(),
		}, nil
	}
	instance.mutex.RUnlock()

	ctx, cancel := context.WithTimeout(m.ctx, 30*time.Second)
	defer cancel()

	response, err := instance.Plugin.HandleCommand(ctx, cmd)
	if err != nil {
		return &CommandResponse{
			CommandID: cmd.ID,
			Status:    "error",
			Error:     err.Error(),
			Timestamp: time.Now(),
		}, nil
	}

	// Update metrics
	instance.mutex.Lock()
	instance.Metrics.CommandsHandled++
	instance.mutex.Unlock()

	return response, nil
}

// GetDeviceState returns the current state of a device
func (m *Manager) GetDeviceState(name string) (*State, error) {
	instance, err := m.GetPlugin(name)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(m.ctx, 10*time.Second)
	defer cancel()

	return instance.Plugin.GetState(ctx)
}

// GetDeviceTelemetry returns telemetry data for a device
func (m *Manager) GetDeviceTelemetry(name string, metric string) (*TelemetryData, error) {
	instance, err := m.GetPlugin(name)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(m.ctx, 10*time.Second)
	defer cancel()

	return instance.Plugin.GetTelemetry(ctx, metric)
}

// AddEventHandler adds an event handler
func (m *Manager) AddEventHandler(handler EventHandler) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.eventHandlers = append(m.eventHandlers, handler)
}

// PublishEvent publishes an event
func (m *Manager) PublishEvent(event *Event) {
	select {
	case m.eventQueue <- event:
		// Event queued successfully
	default:
		m.logger.Warn("Event queue is full, dropping event")
	}
}

// Start starts the manager
func (m *Manager) Start() error {
	m.logger.Info("Starting plugin manager")

	// Start event workers
	for i := 0; i < m.eventWorkers; i++ {
		go m.eventWorker()
	}

	// Start health checker
	go m.healthChecker.Start(m.ctx, m)

	// Start metrics collector
	go m.metricsCollector.Start(m.ctx, m)

	return nil
}

// Stop stops the manager
func (m *Manager) Stop() error {
	m.logger.Info("Stopping plugin manager")

	// Cancel context to stop all goroutines
	m.cancel()

	// Stop all running plugins
	m.mutex.RLock()
	var wg sync.WaitGroup
	for name, instance := range m.plugins {
		if instance.State == StateRunning {
			wg.Add(1)
			go func(name string) {
				defer wg.Done()
				if err := m.StopPlugin(name); err != nil {
					m.logger.WithError(err).WithField("plugin", name).Error("Failed to stop plugin")
				}
			}(name)
		}
	}
	m.mutex.RUnlock()

	// Wait for all plugins to stop
	wg.Wait()

	return nil
}

// eventWorker processes events from the event queue
func (m *Manager) eventWorker() {
	for {
		select {
		case event := <-m.eventQueue:
			m.handleEvent(event)
		case <-m.ctx.Done():
			return
		}
	}
}

// handleEvent processes a single event
func (m *Manager) handleEvent(event *Event) {
	m.mutex.RLock()
	handlers := make([]EventHandler, len(m.eventHandlers))
	copy(handlers, m.eventHandlers)
	m.mutex.RUnlock()

	for _, handler := range handlers {
		if err := handler.HandleEvent(m.ctx, event); err != nil {
			m.logger.WithError(err).Error("Event handler failed")
		}
	}
}

// GetRunningPlugins returns the names of all running plugins
func (m *Manager) GetRunningPlugins() []string {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	var running []string
	for name, instance := range m.plugins {
		instance.mutex.RLock()
		if instance.State == StateRunning {
			running = append(running, name)
		}
		instance.mutex.RUnlock()
	}
	return running
}

// GetPluginMetrics returns metrics for a plugin
func (m *Manager) GetPluginMetrics(name string) (*PluginMetrics, error) {
	instance, err := m.GetPlugin(name)
	if err != nil {
		return nil, err
	}

	instance.mutex.RLock()
	defer instance.mutex.RUnlock()

	metrics := *instance.Metrics
	metrics.Uptime = time.Since(instance.StartTime)
	return &metrics, nil
}

// LoadPluginsFromConfig loads plugins from configuration
func (m *Manager) LoadPluginsFromConfig(configs []PluginConfig) error {
	for _, config := range configs {
		if !config.Enabled {
			continue
		}

		if err := m.CreatePlugin(config.Name, &config); err != nil {
			m.logger.WithError(err).WithField("plugin", config.Name).Error("Failed to create plugin")
			continue
		}

		if err := m.StartPlugin(config.Name); err != nil {
			m.logger.WithError(err).WithField("plugin", config.Name).Error("Failed to start plugin")
		}
	}

	return nil
}