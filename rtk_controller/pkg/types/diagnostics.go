package types

// NetworkDiagnostics represents comprehensive network diagnostic results
type NetworkDiagnostics struct {
	DeviceID         string              `json:"device_id"`
	SpeedTest        *SpeedTestResult    `json:"speed_test,omitempty"`
	LatencyTest      *LatencyTestResult  `json:"latency_test,omitempty"`
	WANTest          *WANTestResult      `json:"wan_test,omitempty"`
	ConnectivityTest *ConnectivityResult `json:"connectivity_test,omitempty"`
	Timestamp        int64               `json:"timestamp"`
}

// SpeedTestResult contains bandwidth test results
type SpeedTestResult struct {
	DownloadMbps float64 `json:"download_mbps"`
	UploadMbps   float64 `json:"upload_mbps"`
	Jitter       float64 `json:"jitter_ms"`
	PacketLoss   float64 `json:"packet_loss_percent"`
	TestServer   string  `json:"test_server"`
	TestDuration int     `json:"test_duration_seconds"`
	Status       string  `json:"status"` // running, completed, failed
	Error        string  `json:"error,omitempty"`
}

// LatencyTestResult contains latency test results
type LatencyTestResult struct {
	Targets       []LatencyTarget `json:"targets"`
	OverallStatus string          `json:"overall_status"`
}

// LatencyTarget represents a single latency test target
type LatencyTarget struct {
	Target          string  `json:"target"` // IP or hostname
	Type            string  `json:"type"`   // gateway, dns, external
	AvgLatency      float64 `json:"avg_latency_ms"`
	MinLatency      float64 `json:"min_latency_ms"`
	MaxLatency      float64 `json:"max_latency_ms"`
	PacketLoss      float64 `json:"packet_loss_percent"`
	PacketsSent     int     `json:"packets_sent"`
	PacketsReceived int     `json:"packets_received"`
	Status          string  `json:"status"` // success, failed, timeout
}

// WANTestResult contains WAN connectivity test results
type WANTestResult struct {
	ISPGatewayReachable bool    `json:"isp_gateway_reachable"`
	ISPGatewayLatency   float64 `json:"isp_gateway_latency_ms"`
	ExternalDNSLatency  float64 `json:"external_dns_latency_ms"` // 8.8.8.8
	WANConnected        bool    `json:"wan_connected"`
	PublicIP            string  `json:"public_ip,omitempty"`
	ISPInfo             string  `json:"isp_info,omitempty"`
}

// ConnectivityResult contains connectivity test results
type ConnectivityResult struct {
	InternalReachability []InternalTarget `json:"internal_reachability"`
	ExternalReachability []ExternalTarget `json:"external_reachability"`
}

// InternalTarget represents an internal network target
type InternalTarget struct {
	DeviceID  string  `json:"device_id,omitempty"`
	IPAddress string  `json:"ip_address"`
	Reachable bool    `json:"reachable"`
	Latency   float64 `json:"latency_ms"`
	Method    string  `json:"method"` // ping, tcp_connect
}

// ExternalTarget represents an external network target
type ExternalTarget struct {
	Target     string  `json:"target"`
	Type       string  `json:"type"` // dns, web, speedtest_server
	Reachable  bool    `json:"reachable"`
	Latency    float64 `json:"latency_ms"`
	HTTPStatus int     `json:"http_status,omitempty"`
}

// DiagnosticsSummary provides a summary of diagnostic results
type DiagnosticsSummary struct {
	DeviceID         string  `json:"device_id"`
	Timestamp        int64   `json:"timestamp"`
	OverallHealth    string  `json:"overall_health"` // excellent, good, fair, poor
	DownloadSpeed    float64 `json:"download_speed_mbps"`
	UploadSpeed      float64 `json:"upload_speed_mbps"`
	AverageLatency   float64 `json:"average_latency_ms"`
	PacketLoss       float64 `json:"packet_loss_percent"`
	WANStatus        string  `json:"wan_status"`
	InternalDevices  int     `json:"internal_devices_reachable"`
	ExternalServices int     `json:"external_services_reachable"`
	Issues           []Issue `json:"issues,omitempty"`
}

// Issue represents a detected network issue
type Issue struct {
	Type        string `json:"type"`     // high_latency, packet_loss, low_bandwidth, wan_down
	Severity    string `json:"severity"` // critical, warning, info
	Description string `json:"description"`
	Suggestion  string `json:"suggestion,omitempty"`
}
