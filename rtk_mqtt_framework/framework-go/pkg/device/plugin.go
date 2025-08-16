package device

import (
	"context"
	"encoding/json"
	"time"
)

// Plugin defines the interface that all device plugins must implement
type Plugin interface {
	// GetInfo returns basic information about the device plugin
	GetInfo() *Info

	// Initialize initializes the plugin with the given configuration
	Initialize(ctx context.Context, config json.RawMessage) error

	// Start starts the plugin operation
	Start(ctx context.Context) error

	// Stop stops the plugin operation
	Stop() error

	// GetState returns the current device state
	GetState(ctx context.Context) (*State, error)

	// GetTelemetry returns telemetry data for the specified metric
	GetTelemetry(ctx context.Context, metric string) (*TelemetryData, error)

	// GetAttributes returns device attributes
	GetAttributes(ctx context.Context) (map[string]interface{}, error)

	// HandleCommand handles incoming commands
	HandleCommand(ctx context.Context, cmd *Command) (*CommandResponse, error)

	// GetHealth returns device health status
	GetHealth(ctx context.Context) (*Health, error)
}

// Info contains basic information about a device plugin
type Info struct {
	Name        string            `json:"name"`
	Type        string            `json:"type"`
	Version     string            `json:"version"`
	Description string            `json:"description"`
	Vendor      string            `json:"vendor"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// State represents the current state of a device
type State struct {
	Status      string                 `json:"status"`      // online, offline, error, maintenance
	Health      string                 `json:"health"`      // healthy, warning, critical, unknown
	LastSeen    time.Time              `json:"last_seen"`
	Uptime      time.Duration          `json:"uptime"`
	Properties  map[string]interface{} `json:"properties,omitempty"`
	Diagnostics map[string]interface{} `json:"diagnostics,omitempty"`
	Timestamp   time.Time              `json:"timestamp"`
}

// TelemetryData represents telemetry information
type TelemetryData struct {
	Metric    string                 `json:"metric"`
	Value     interface{}            `json:"value"`
	Unit      string                 `json:"unit,omitempty"`
	Labels    map[string]string      `json:"labels,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

// Command represents a command sent to a device
type Command struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Action    string                 `json:"action"`
	Params    map[string]interface{} `json:"params,omitempty"`
	Timeout   time.Duration          `json:"timeout,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

// CommandResponse represents the response to a command
type CommandResponse struct {
	CommandID string                 `json:"command_id"`
	Status    string                 `json:"status"`    // success, error, timeout, pending
	Result    map[string]interface{} `json:"result,omitempty"`
	Error     string                 `json:"error,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

// Health represents device health information
type Health struct {
	Status      string                 `json:"status"`      // healthy, warning, critical, unknown
	Score       float64                `json:"score"`       // 0.0 to 1.0
	Checks      map[string]HealthCheck `json:"checks,omitempty"`
	LastCheck   time.Time              `json:"last_check"`
	NextCheck   time.Time              `json:"next_check,omitempty"`
	Diagnostics map[string]interface{} `json:"diagnostics,omitempty"`
}

// HealthCheck represents an individual health check
type HealthCheck struct {
	Name        string      `json:"name"`
	Status      string      `json:"status"`
	Value       interface{} `json:"value,omitempty"`
	Threshold   interface{} `json:"threshold,omitempty"`
	Message     string      `json:"message,omitempty"`
	LastChecked time.Time   `json:"last_checked"`
}

// Event represents a device event
type Event struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Level       string                 `json:"level"`       // info, warning, error, critical
	Message     string                 `json:"message"`
	Source      string                 `json:"source"`
	Category    string                 `json:"category,omitempty"`
	Data        map[string]interface{} `json:"data,omitempty"`
	Timestamp   time.Time              `json:"timestamp"`
	DeviceID    string                 `json:"device_id"`
	SchemaID    string                 `json:"schema_id,omitempty"`
}

// EventHandler defines the interface for handling device events
type EventHandler interface {
	HandleEvent(ctx context.Context, event *Event) error
}

// PluginFactory defines the interface for creating plugin instances
type PluginFactory interface {
	CreatePlugin() (Plugin, error)
	GetInfo() *FactoryInfo
}

// FactoryInfo contains information about a plugin factory
type FactoryInfo struct {
	Name         string   `json:"name"`
	Version      string   `json:"version"`
	SupportedTypes []string `json:"supported_types"`
	Description  string   `json:"description"`
}

// PluginConfig represents plugin configuration
type PluginConfig struct {
	Type     string          `json:"type" validate:"required"`
	Name     string          `json:"name" validate:"required"`
	Enabled  bool            `json:"enabled"`
	Config   json.RawMessage `json:"config,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// ValidationResult represents the result of plugin validation
type ValidationResult struct {
	Valid   bool     `json:"valid"`
	Errors  []string `json:"errors,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
}

// PluginMetrics contains metrics for a plugin
type PluginMetrics struct {
	StartTime         time.Time     `json:"start_time"`
	Uptime           time.Duration `json:"uptime"`
	CommandsHandled  uint64        `json:"commands_handled"`
	EventsGenerated  uint64        `json:"events_generated"`
	ErrorCount       uint64        `json:"error_count"`
	LastError        string        `json:"last_error,omitempty"`
	LastErrorTime    time.Time     `json:"last_error_time,omitempty"`
	TelemetryCount   uint64        `json:"telemetry_count"`
	StateUpdates     uint64        `json:"state_updates"`
}

// StatusReporter provides status reporting capabilities
type StatusReporter interface {
	GetStatus() string
	GetMetrics() *PluginMetrics
	GetLastError() error
}

// Lifecycle interface for managing plugin lifecycle
type Lifecycle interface {
	Start(ctx context.Context) error
	Stop() error
	Restart(ctx context.Context) error
	IsRunning() bool
	GetStatus() string
}

// Configuration interface for plugin configuration management
type Configuration interface {
	GetConfig() json.RawMessage
	UpdateConfig(config json.RawMessage) error
	ValidateConfig(config json.RawMessage) *ValidationResult
	GetConfigSchema() json.RawMessage
}