package device

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// BasePlugin provides a base implementation for device plugins
type BasePlugin struct {
	info     *Info
	logger   *logrus.Logger
	mutex    sync.RWMutex
	state    *State
	config   json.RawMessage
	running  bool
	startTime time.Time

	// Plugin-specific data
	attributes  map[string]interface{}
	telemetry   map[string]*TelemetryData
	healthData  *Health
	lastCommand *Command
	
	// Callbacks for plugin-specific implementations
	startCallback    func(ctx context.Context) error
	stopCallback     func() error
	commandCallback  func(ctx context.Context, cmd *Command) (*CommandResponse, error)
	stateCallback    func(ctx context.Context) (*State, error)
	telemetryCallback func(ctx context.Context, metric string) (*TelemetryData, error)
	healthCallback   func(ctx context.Context) (*Health, error)
}

// BasePluginConfig represents base plugin configuration
type BasePluginConfig struct {
	Name            string                 `json:"name"`
	Type            string                 `json:"type"`
	Version         string                 `json:"version"`
	Description     string                 `json:"description"`
	Vendor          string                 `json:"vendor"`
	UpdateInterval  time.Duration          `json:"update_interval"`
	HealthInterval  time.Duration          `json:"health_interval"`
	Attributes      map[string]interface{} `json:"attributes,omitempty"`
	Metadata        map[string]string      `json:"metadata,omitempty"`
}

// NewBasePlugin creates a new base plugin
func NewBasePlugin(info *Info) *BasePlugin {
	return &BasePlugin{
		info:       info,
		logger:     logrus.New(),
		attributes: make(map[string]interface{}),
		telemetry:  make(map[string]*TelemetryData),
		state: &State{
			Status:      "offline",
			Health:      "unknown",
			Properties:  make(map[string]interface{}),
			Diagnostics: make(map[string]interface{}),
			Timestamp:   time.Now(),
		},
		healthData: &Health{
			Status:      "unknown",
			Score:       0.0,
			Checks:      make(map[string]HealthCheck),
			LastCheck:   time.Now(),
			Diagnostics: make(map[string]interface{}),
		},
	}
}

// GetInfo returns plugin information
func (p *BasePlugin) GetInfo() *Info {
	return p.info
}

// Initialize initializes the plugin
func (p *BasePlugin) Initialize(ctx context.Context, config json.RawMessage) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.config = config
	p.logger.WithField("plugin", p.info.Name).Info("Plugin initialized")
	return nil
}

// Start starts the plugin
func (p *BasePlugin) Start(ctx context.Context) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.running {
		return nil
	}

	p.startTime = time.Now()
	p.running = true

	// Update state
	p.state.Status = "online"
	p.state.Health = "healthy"
	p.state.LastSeen = time.Now()
	p.state.Uptime = 0
	p.state.Timestamp = time.Now()

	// Call plugin-specific start logic
	if p.startCallback != nil {
		if err := p.startCallback(ctx); err != nil {
			p.running = false
			p.state.Status = "error"
			p.state.Health = "critical"
			return err
		}
	}

	p.logger.WithField("plugin", p.info.Name).Info("Plugin started")
	return nil
}

// Stop stops the plugin
func (p *BasePlugin) Stop() error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if !p.running {
		return nil
	}

	// Call plugin-specific stop logic
	if p.stopCallback != nil {
		if err := p.stopCallback(); err != nil {
			p.logger.WithError(err).WithField("plugin", p.info.Name).Warn("Plugin stop callback failed")
		}
	}

	p.running = false

	// Update state
	p.state.Status = "offline"
	p.state.Health = "unknown"
	p.state.Timestamp = time.Now()

	p.logger.WithField("plugin", p.info.Name).Info("Plugin stopped")
	return nil
}

// GetState returns the current device state
func (p *BasePlugin) GetState(ctx context.Context) (*State, error) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	// Call plugin-specific state logic if available
	if p.stateCallback != nil {
		if state, err := p.stateCallback(ctx); err == nil {
			return state, nil
		}
	}

	// Return base state
	state := *p.state
	if p.running {
		state.Uptime = time.Since(p.startTime)
		state.LastSeen = time.Now()
	}
	state.Timestamp = time.Now()

	return &state, nil
}

// GetTelemetry returns telemetry data
func (p *BasePlugin) GetTelemetry(ctx context.Context, metric string) (*TelemetryData, error) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	// Call plugin-specific telemetry logic if available
	if p.telemetryCallback != nil {
		if data, err := p.telemetryCallback(ctx, metric); err == nil {
			return data, nil
		}
	}

	// Return cached telemetry data
	if data, exists := p.telemetry[metric]; exists {
		return data, nil
	}

	return nil, &Error{
		Code:    "TELEMETRY_NOT_FOUND",
		Message: "Telemetry metric not found: " + metric,
	}
}

// GetAttributes returns device attributes
func (p *BasePlugin) GetAttributes(ctx context.Context) (map[string]interface{}, error) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	attributes := make(map[string]interface{})
	for k, v := range p.attributes {
		attributes[k] = v
	}

	return attributes, nil
}

// HandleCommand handles incoming commands
func (p *BasePlugin) HandleCommand(ctx context.Context, cmd *Command) (*CommandResponse, error) {
	p.mutex.Lock()
	p.lastCommand = cmd
	p.mutex.Unlock()

	// Call plugin-specific command logic if available
	if p.commandCallback != nil {
		return p.commandCallback(ctx, cmd)
	}

	// Default response for unsupported commands
	return &CommandResponse{
		CommandID: cmd.ID,
		Status:    "error",
		Error:     "Command not supported",
		Timestamp: time.Now(),
	}, nil
}

// GetHealth returns device health status
func (p *BasePlugin) GetHealth(ctx context.Context) (*Health, error) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	// Call plugin-specific health logic if available
	if p.healthCallback != nil {
		if health, err := p.healthCallback(ctx); err == nil {
			return health, nil
		}
	}

	// Return base health data
	health := *p.healthData
	health.LastCheck = time.Now()

	if p.running {
		health.Status = "healthy"
		health.Score = 1.0
		
		// Add basic health checks
		health.Checks = map[string]HealthCheck{
			"running": {
				Name:        "Running",
				Status:      "healthy",
				Value:       p.running,
				Message:     "Plugin is running",
				LastChecked: time.Now(),
			},
			"uptime": {
				Name:        "Uptime",
				Status:      "healthy",
				Value:       time.Since(p.startTime).String(),
				Message:     "Plugin uptime",
				LastChecked: time.Now(),
			},
		}
	} else {
		health.Status = "critical"
		health.Score = 0.0
		health.Checks = map[string]HealthCheck{
			"running": {
				Name:        "Running",
				Status:      "critical",
				Value:       p.running,
				Message:     "Plugin is not running",
				LastChecked: time.Now(),
			},
		}
	}

	return &health, nil
}

// SetStartCallback sets the callback for plugin-specific start logic
func (p *BasePlugin) SetStartCallback(callback func(ctx context.Context) error) {
	p.startCallback = callback
}

// SetStopCallback sets the callback for plugin-specific stop logic
func (p *BasePlugin) SetStopCallback(callback func() error) {
	p.stopCallback = callback
}

// SetCommandCallback sets the callback for plugin-specific command handling
func (p *BasePlugin) SetCommandCallback(callback func(ctx context.Context, cmd *Command) (*CommandResponse, error)) {
	p.commandCallback = callback
}

// SetStateCallback sets the callback for plugin-specific state retrieval
func (p *BasePlugin) SetStateCallback(callback func(ctx context.Context) (*State, error)) {
	p.stateCallback = callback
}

// SetTelemetryCallback sets the callback for plugin-specific telemetry retrieval
func (p *BasePlugin) SetTelemetryCallback(callback func(ctx context.Context, metric string) (*TelemetryData, error)) {
	p.telemetryCallback = callback
}

// SetHealthCallback sets the callback for plugin-specific health checks
func (p *BasePlugin) SetHealthCallback(callback func(ctx context.Context) (*Health, error)) {
	p.healthCallback = callback
}

// UpdateState updates the plugin state
func (p *BasePlugin) UpdateState(updates map[string]interface{}) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	for key, value := range updates {
		p.state.Properties[key] = value
	}
	p.state.Timestamp = time.Now()
}

// UpdateTelemetry updates telemetry data
func (p *BasePlugin) UpdateTelemetry(metric string, value interface{}, unit string) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.telemetry[metric] = &TelemetryData{
		Metric:    metric,
		Value:     value,
		Unit:      unit,
		Timestamp: time.Now(),
	}
}

// UpdateAttributes updates device attributes
func (p *BasePlugin) UpdateAttributes(attributes map[string]interface{}) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	for key, value := range attributes {
		p.attributes[key] = value
	}
}

// IsRunning returns whether the plugin is running
func (p *BasePlugin) IsRunning() bool {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	return p.running
}

// GetConfig returns the plugin configuration
func (p *BasePlugin) GetConfig() json.RawMessage {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	return p.config
}

// GetLogger returns the plugin logger
func (p *BasePlugin) GetLogger() *logrus.Logger {
	return p.logger
}

// Error represents a plugin-specific error
type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Cause   error  `json:"cause,omitempty"`
}

// Error implements the error interface
func (e *Error) Error() string {
	if e.Cause != nil {
		return e.Message + ": " + e.Cause.Error()
	}
	return e.Message
}

// Unwrap returns the underlying error
func (e *Error) Unwrap() error {
	return e.Cause
}