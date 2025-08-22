package types

import (
	"time"
)

// Device represents a complete device with all its data
type Device struct {
	DeviceID     string            `json:"device_id"`
	Tenant       string            `json:"tenant"`
	Site         string            `json:"site"`
	DeviceType   string            `json:"device_type"`
	Health       string            `json:"health"` // ok, warning, critical, unknown
	Online       bool              `json:"online"`
	FirstSeen    *time.Time        `json:"first_seen,omitempty"`
	LastSeen     *time.Time        `json:"last_seen,omitempty"`
	State        *DeviceStateData  `json:"state,omitempty"`
	Attributes   *DeviceAttributes `json:"attributes,omitempty"`
	EventCount   int               `json:"event_count"`
	CommandCount int               `json:"command_count"`
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
}

// DeviceStateData represents device state information
type DeviceStateData struct {
	Schema     *string                `json:"schema,omitempty"`
	Timestamp  *int64                 `json:"timestamp,omitempty"` // Unix timestamp in milliseconds
	Health     *string                `json:"health,omitempty"`
	UptimeS    *int64                 `json:"uptime_s,omitempty"`
	Version    *string                `json:"version,omitempty"`
	Components map[string]interface{} `json:"components,omitempty"`
}

// DeviceAttributes represents device attributes
type DeviceAttributes struct {
	Schema     *string `json:"schema,omitempty"`
	Timestamp  *int64  `json:"timestamp,omitempty"`
	DeviceType *string `json:"device_type,omitempty"`
	Model      *string `json:"model,omitempty"`
	HwVersion  *string `json:"hw_version,omitempty"`
	FwVersion  *string `json:"fw_version,omitempty"`
}

// ExtendedDeviceFilter represents extended filtering criteria for devices
type ExtendedDeviceFilter struct {
	Tenant string `json:"tenant,omitempty"`
	Site   string `json:"site,omitempty"`
	Status string `json:"status,omitempty"` // online, offline
}

// DeviceHistoryEntry represents a device history entry
type DeviceHistoryEntry struct {
	Timestamp int64  `json:"timestamp"`
	Event     string `json:"event"`
	Details   string `json:"details"`
}

// ExtendedDeviceStats represents extended device statistics
type ExtendedDeviceStats struct {
	TotalDevices   int       `json:"total_devices"`
	OnlineDevices  int       `json:"online_devices"`
	OfflineDevices int       `json:"offline_devices"`
	TotalEvents    int       `json:"total_events"`
	LastUpdated    time.Time `json:"last_updated"`
}
