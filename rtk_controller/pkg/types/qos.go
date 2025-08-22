package types

// QoSInfo represents Quality of Service configuration and statistics
type QoSInfo struct {
	Enabled           bool               `json:"enabled"`
	BandwidthCaps     []BandwidthRule    `json:"bandwidth_caps"`
	TrafficShaping    []TrafficRule      `json:"traffic_shaping"`
	PriorityQueues    []QueueInfo        `json:"priority_queues"`
	ActiveConnections []ActiveConnection `json:"active_connections"`
	TrafficStats      *TrafficStats      `json:"traffic_stats"`
}

// BandwidthRule defines bandwidth limits for specific targets
type BandwidthRule struct {
	RuleID        string `json:"rule_id"`
	Target        string `json:"target"` // device_mac, ip_range, all
	UploadLimit   int    `json:"upload_limit_mbps"`
	DownloadLimit int    `json:"download_limit_mbps"`
	Priority      int    `json:"priority"`
	Enabled       bool   `json:"enabled"`
}

// TrafficRule defines traffic shaping rules
type TrafficRule struct {
	RuleID     string `json:"rule_id"`
	Protocol   string `json:"protocol"` // tcp, udp, icmp
	Ports      []int  `json:"ports,omitempty"`
	Action     string `json:"action"` // allow, block, throttle, prioritize
	Priority   int    `json:"priority"`
	BytesMatch int64  `json:"bytes_matched"`
}

// QueueInfo represents priority queue information
type QueueInfo struct {
	QueueID        string  `json:"queue_id"`
	Priority       int     `json:"priority"`
	BandwidthPct   float64 `json:"bandwidth_percent"`
	CurrentLoad    float64 `json:"current_load_percent"`
	PacketsQueued  int     `json:"packets_queued"`
	PacketsDropped int     `json:"packets_dropped"`
}

// ActiveConnection represents an active network connection
type ActiveConnection struct {
	ConnectionID  string `json:"connection_id"`
	SourceIP      string `json:"source_ip"`
	SourcePort    int    `json:"source_port"`
	DestIP        string `json:"dest_ip"`
	DestPort      int    `json:"dest_port"`
	Protocol      string `json:"protocol"`
	State         string `json:"state"` // established, time_wait, etc.
	BytesSent     int64  `json:"bytes_sent"`
	BytesReceived int64  `json:"bytes_received"`
	Duration      int64  `json:"duration_seconds"`
	QueueID       string `json:"queue_id,omitempty"`
}

// TrafficStats contains overall traffic statistics
type TrafficStats struct {
	TotalBandwidthMbps   float64               `json:"total_bandwidth_mbps"`
	UsedBandwidthMbps    float64               `json:"used_bandwidth_mbps"`
	DeviceTraffic        []DeviceTrafficInfo   `json:"device_traffic"`
	TopTalkers           []TopTalkerInfo       `json:"top_talkers"`
	ProtocolDistribution []ProtocolTrafficInfo `json:"protocol_distribution"`
	UpdatedAt            int64                 `json:"updated_at"`
}

// DeviceTrafficInfo contains traffic information for a specific device
type DeviceTrafficInfo struct {
	DeviceID     string  `json:"device_id"`
	DeviceMAC    string  `json:"device_mac"`
	FriendlyName string  `json:"friendly_name,omitempty"`
	UploadMbps   float64 `json:"upload_mbps"`
	DownloadMbps float64 `json:"download_mbps"`
	TotalBytes   int64   `json:"total_bytes"`
	ActiveConns  int     `json:"active_connections"`
	BandwidthPct float64 `json:"bandwidth_percent"`
}

// TopTalkerInfo identifies top bandwidth consumers
type TopTalkerInfo struct {
	DeviceID     string  `json:"device_id"`
	FriendlyName string  `json:"friendly_name,omitempty"`
	TotalMbps    float64 `json:"total_mbps"`
	TrafficType  string  `json:"traffic_type"` // upload, download, total
	Rank         int     `json:"rank"`
}

// ProtocolTrafficInfo shows traffic distribution by protocol
type ProtocolTrafficInfo struct {
	Protocol    string  `json:"protocol"`
	TotalMbps   float64 `json:"total_mbps"`
	Percentage  float64 `json:"percentage"`
	PacketCount int64   `json:"packet_count"`
}

// TrafficAnomaly represents detected traffic anomalies
type TrafficAnomaly struct {
	ID          string  `json:"id"`
	DeviceID    string  `json:"device_id"`
	Type        string  `json:"type"`     // spike, sustained_high, unusual_protocol
	Severity    string  `json:"severity"` // low, medium, high, critical
	Description string  `json:"description"`
	Value       float64 `json:"value"`
	Threshold   float64 `json:"threshold"`
	StartTime   int64   `json:"start_time"`
	EndTime     int64   `json:"end_time,omitempty"`
	Resolved    bool    `json:"resolved"`
}

// QoSRecommendation provides QoS policy recommendations
type QoSRecommendation struct {
	ID          string      `json:"id"`
	Type        string      `json:"type"` // bandwidth_cap, traffic_shaping, priority_queue
	Reason      string      `json:"reason"`
	Description string      `json:"description"`
	Priority    int         `json:"priority"`
	Impact      string      `json:"impact"`         // low, medium, high
	Rule        interface{} `json:"rule,omitempty"` // BandwidthRule, TrafficRule, or QueueInfo
	Devices     []string    `json:"affected_devices,omitempty"`
	CreatedAt   int64       `json:"created_at"`
}
