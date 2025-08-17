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
	
	// Network topology related fields
	NetworkInfo  *NetworkDeviceInfo     `json:"network_info,omitempty"`    // 網路拓撲資訊
	
	UpdatedAt    time.Time              `json:"updated_at"`
}

// NetworkDeviceInfo represents network topology information for a device
type NetworkDeviceInfo struct {
	PrimaryMAC    string            `json:"primary_mac,omitempty"`
	Hostname      string            `json:"hostname,omitempty"`
	Manufacturer  string            `json:"manufacturer,omitempty"`
	Model         string            `json:"model,omitempty"`
	Location      string            `json:"location,omitempty"`
	Role          string            `json:"role,omitempty"`          // gateway, access_point, switch, client, bridge, router
	Interfaces    []InterfaceInfo   `json:"interfaces,omitempty"`
	Capabilities  []string          `json:"capabilities,omitempty"` // routing, bridge, ap, client, nat, dhcp
	Connections   []ConnectionInfo  `json:"connections,omitempty"`  // 連接到其他設備的資訊
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

// InterfaceInfo represents basic interface information for device state
type InterfaceInfo struct {
	Name         string   `json:"name"`                     // eth0, wlan0, br0
	Type         string   `json:"type"`                     // ethernet, wifi, bridge
	MacAddress   string   `json:"mac_address,omitempty"`
	IPAddresses  []string `json:"ip_addresses,omitempty"`
	Status       string   `json:"status"`                   // up, down, dormant
	
	// WiFi specific fields
	SSID         string   `json:"ssid,omitempty"`
	BSSID        string   `json:"bssid,omitempty"`
	Channel      int      `json:"channel,omitempty"`
	Band         string   `json:"band,omitempty"`           // 2.4G, 5G, 6G
	RSSI         int      `json:"rssi,omitempty"`
	
	// Statistics
	TxBytes      int64    `json:"tx_bytes,omitempty"`
	RxBytes      int64    `json:"rx_bytes,omitempty"`
	LastUpdate   int64    `json:"last_update,omitempty"`
}

// ConnectionInfo represents connection information for device state
type ConnectionInfo struct {
	ConnectedDeviceID string  `json:"connected_device_id,omitempty"`
	ConnectionType    string  `json:"connection_type"`               // ethernet, wifi, bridge, route
	LocalInterface    string  `json:"local_interface,omitempty"`
	RemoteInterface   string  `json:"remote_interface,omitempty"`
	RSSI              int     `json:"rssi,omitempty"`
	LinkSpeed         int     `json:"link_speed,omitempty"`         // Mbps
	Latency           float64 `json:"latency,omitempty"`            // ms
	LastSeen          int64   `json:"last_seen"`
}