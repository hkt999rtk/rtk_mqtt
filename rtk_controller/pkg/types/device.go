package types

import (
	"time"
)

// DeviceState represents the state of a device
type DeviceState struct {
	ID           string                 `json:"id"`
	Tenant       string                 `json:"tenant"`
	Site         string                 `json:"site"`
	DeviceType   string                 `json:"device_type"`
	Health       string                 `json:"health"`       // ok, warning, critical, unknown
	LastSeen     int64                  `json:"last_seen"`    // Unix timestamp in milliseconds
	UptimeS      int64                  `json:"uptime_s"`     // Uptime in seconds
	Version      string                 `json:"version"`
	Components   map[string]interface{} `json:"components"`
	Attributes   map[string]interface{} `json:"attributes"`
	Telemetry    map[string]interface{} `json:"telemetry"`
	Online       bool                   `json:"online"`
	LastWill     *LastWillMessage       `json:"last_will,omitempty"`
	UpdatedAt    time.Time              `json:"updated_at"`
}

// DeviceEvent represents an event from a device
type DeviceEvent struct {
	ID          string                 `json:"id"`
	DeviceID    string                 `json:"device_id"`
	EventType   string                 `json:"event_type"`
	Severity    string                 `json:"severity"`    // info, warning, error, critical
	Message     string                 `json:"message"`
	Data        map[string]interface{} `json:"data"`
	Timestamp   int64                  `json:"timestamp"`   // Unix timestamp in milliseconds
	Topic       string                 `json:"topic"`
	Processed   bool                   `json:"processed"`
	ProcessedAt *time.Time             `json:"processed_at,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
}

// LastWillMessage represents a device's last will testament
type LastWillMessage struct {
	Topic     string `json:"topic"`
	Payload   string `json:"payload"`
	QoS       byte   `json:"qos"`
	Retained  bool   `json:"retained"`
	Timestamp int64  `json:"timestamp"`
}

// DeviceCommand represents a command sent to a device
type DeviceCommand struct {
	ID          string                 `json:"id"`
	DeviceID    string                 `json:"device_id"`
	Operation   string                 `json:"operation"`
	Args        map[string]interface{} `json:"args"`
	TimeoutMS   int64                  `json:"timeout_ms"`
	Status      string                 `json:"status"`      // pending, sent, ack, completed, failed, timeout
	Result      map[string]interface{} `json:"result,omitempty"`
	Error       string                 `json:"error,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	SentAt      *time.Time             `json:"sent_at,omitempty"`
	CompletedAt *time.Time             `json:"completed_at,omitempty"`
}

// DeviceFilter represents filtering criteria for devices
type DeviceFilter struct {
	Tenant     string   `json:"tenant,omitempty"`
	Site       string   `json:"site,omitempty"`
	DeviceType string   `json:"device_type,omitempty"`
	Health     []string `json:"health,omitempty"`
	Online     *bool    `json:"online,omitempty"`
	LastSeenGT *int64   `json:"last_seen_gt,omitempty"` // Greater than timestamp
	LastSeenLT *int64   `json:"last_seen_lt,omitempty"` // Less than timestamp
}

// EventFilter represents filtering criteria for events
type EventFilter struct {
	DeviceID    string   `json:"device_id,omitempty"`
	EventType   string   `json:"event_type,omitempty"`
	Severity    []string `json:"severity,omitempty"`
	Processed   *bool    `json:"processed,omitempty"`
	StartTime   *int64   `json:"start_time,omitempty"`
	EndTime     *int64   `json:"end_time,omitempty"`
}

// DeviceStats represents device statistics
type DeviceStats struct {
	TotalDevices    int            `json:"total_devices"`
	OnlineDevices   int            `json:"online_devices"`
	OfflineDevices  int            `json:"offline_devices"`
	HealthStats     map[string]int `json:"health_stats"`     // health status -> count
	DeviceTypeStats map[string]int `json:"device_type_stats"` // device type -> count
	LastUpdated     time.Time      `json:"last_updated"`
}

// EventStats represents event statistics
type EventStats struct {
	TotalEvents    int            `json:"total_events"`
	SeverityStats  map[string]int `json:"severity_stats"`  // severity -> count
	EventTypeStats map[string]int `json:"event_type_stats"` // event type -> count
	ProcessedCount int            `json:"processed_count"`
	PendingCount   int            `json:"pending_count"`
	LastUpdated    time.Time      `json:"last_updated"`
}