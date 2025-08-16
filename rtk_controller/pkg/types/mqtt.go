package types

import "time"

// MQTTMessageLog represents a logged MQTT message
type MQTTMessageLog struct {
	ID          string `json:"id"`
	Timestamp   int64  `json:"timestamp"`   // Unix timestamp in milliseconds
	Topic       string `json:"topic"`
	Payload     string `json:"payload"`
	QoS         byte   `json:"qos"`
	Retained    bool   `json:"retained"`
	MessageSize int    `json:"message_size"`
	Direction   string `json:"direction"` // inbound, outbound
}

// MQTTLoggerStats represents MQTT logger statistics
type MQTTLoggerStats struct {
	TotalMessages    int64     `json:"total_messages"`
	InboundMessages  int64     `json:"inbound_messages"`
	OutboundMessages int64     `json:"outbound_messages"`
	TotalBytes       int64     `json:"total_bytes"`
	OldestMessage    int64     `json:"oldest_message"`  // Unix timestamp
	NewestMessage    int64     `json:"newest_message"`  // Unix timestamp
	LastPurge        int64     `json:"last_purge"`      // Unix timestamp
	PurgedMessages   int64     `json:"purged_messages"`
}

// DeviceInfo represents basic device identification information
type DeviceInfo struct {
	Tenant   string `json:"tenant"`
	Site     string `json:"site"`
	DeviceID string `json:"device_id"`
	Type     string `json:"type"`
}

// ExtendedDeviceInfo represents extended device information
type ExtendedDeviceInfo struct {
	DeviceInfo
	Location    string `json:"location,omitempty"`
	Model       string `json:"model,omitempty"`
	Firmware    string `json:"firmware,omitempty"`
	Hardware    string `json:"hardware,omitempty"`
	Description string `json:"description,omitempty"`
}

// DeviceStatus represents the status of a device
type DeviceStatus string

const (
	DeviceStatusOnline    DeviceStatus = "online"
	DeviceStatusOffline   DeviceStatus = "offline"
	DeviceStatusUnknown   DeviceStatus = "unknown"
	DeviceStatusError     DeviceStatus = "error"
)

// DeviceStateManager represents the complete state of a device with additional manager-specific fields
type DeviceStateManager struct {
	DeviceInfo
	Status    DeviceStatus           `json:"status"`
	Health    string                 `json:"health"`    // ok, warning, critical, unknown
	State     map[string]interface{} `json:"state"`
	LastSeen  time.Time              `json:"last_seen"`
	UpdatedAt time.Time              `json:"updated_at"`
}