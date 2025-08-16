package types

import (
	"time"
)

// DiagnosisData represents diagnostic data collected from devices
type DiagnosisData struct {
	ID          string                 `json:"id"`
	DeviceID    string                 `json:"device_id"`
	Type        string                 `json:"type"`        // wifi, network, system, connectivity, etc.
	Category    string                 `json:"category"`    // performance, error, warning, info
	Severity    string                 `json:"severity"`    // low, medium, high, critical
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Data        map[string]interface{} `json:"data"`
	Metrics     map[string]float64     `json:"metrics"`
	Tags        []string               `json:"tags"`
	Source      string                 `json:"source"`      // telemetry, event, command, manual
	Timestamp   int64                  `json:"timestamp"`   // Unix timestamp in milliseconds
	ProcessedAt *time.Time             `json:"processed_at,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
}

// DiagnosisResult represents the result of diagnosis analysis
type DiagnosisResult struct {
	ID             string                 `json:"id"`
	DiagnosisID    string                 `json:"diagnosis_id"`
	DeviceID       string                 `json:"device_id"`
	AnalyzerType   string                 `json:"analyzer_type"`   // builtin, plugin, external
	AnalyzerName   string                 `json:"analyzer_name"`
	Status         string                 `json:"status"`          // analyzing, completed, failed
	Confidence     float64                `json:"confidence"`      // 0.0 - 1.0
	Issues         []DiagnosisIssue       `json:"issues"`
	Recommendations []Recommendation      `json:"recommendations"`
	Metrics        map[string]interface{} `json:"metrics"`
	ExecutionTime  int64                  `json:"execution_time_ms"`
	Error          string                 `json:"error,omitempty"`
	CreatedAt      time.Time              `json:"created_at"`
	CompletedAt    *time.Time             `json:"completed_at,omitempty"`
}

// DiagnosisIssue represents an identified issue
type DiagnosisIssue struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`        // connectivity, performance, configuration, hardware
	Severity    string                 `json:"severity"`    // low, medium, high, critical
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Impact      string                 `json:"impact"`
	Likelihood  float64                `json:"likelihood"`  // 0.0 - 1.0
	Evidence    map[string]interface{} `json:"evidence"`
	AffectedComponents []string        `json:"affected_components"`
}

// Recommendation represents a suggested action
type Recommendation struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`        // immediate, scheduled, preventive
	Priority    string                 `json:"priority"`    // low, medium, high, urgent
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Actions     []RecommendedAction    `json:"actions"`
	EstimatedTime string               `json:"estimated_time"`
	RequiredSkills []string            `json:"required_skills"`
}

// RecommendedAction represents a specific action to take
type RecommendedAction struct {
	Type        string                 `json:"type"`        // command, manual, configuration
	Description string                 `json:"description"`
	Command     string                 `json:"command,omitempty"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
	Risk        string                 `json:"risk"`        // low, medium, high
}

// DiagnosisSession represents a complete diagnosis session
type DiagnosisSession struct {
	ID          string            `json:"id"`
	DeviceID    string            `json:"device_id"`
	Type        string            `json:"type"`        // scheduled, triggered, manual
	Status      string            `json:"status"`      // running, completed, failed, cancelled
	TriggerBy   string            `json:"trigger_by"`  // user, event, schedule, threshold
	StartTime   time.Time         `json:"start_time"`
	EndTime     *time.Time        `json:"end_time,omitempty"`
	Duration    int64             `json:"duration_ms"`
	DataCount   int               `json:"data_count"`
	ResultCount int               `json:"result_count"`
	IssueCount  int               `json:"issue_count"`
	Progress    float64           `json:"progress"`    // 0.0 - 1.0
	Config      DiagnosisConfig   `json:"config"`
	Summary     DiagnosisSummary  `json:"summary"`
}

// DiagnosisSummary provides high-level summary of diagnosis results
type DiagnosisSummary struct {
	OverallHealth   string             `json:"overall_health"`   // excellent, good, fair, poor, critical
	TotalIssues     int                `json:"total_issues"`
	CriticalIssues  int                `json:"critical_issues"`
	HighIssues      int                `json:"high_issues"`
	MediumIssues    int                `json:"medium_issues"`
	LowIssues       int                `json:"low_issues"`
	TopIssues       []DiagnosisIssue   `json:"top_issues"`
	KeyMetrics      map[string]float64 `json:"key_metrics"`
	Recommendations []Recommendation   `json:"urgent_recommendations"`
}

// DiagnosisConfig represents configuration for diagnosis analysis
type DiagnosisConfig struct {
	EnabledAnalyzers []string               `json:"enabled_analyzers"`
	AnalysisDepth    string                 `json:"analysis_depth"`    // basic, standard, deep
	TimeRange        TimeRange              `json:"time_range"`
	Filters          map[string]interface{} `json:"filters"`
	Thresholds       map[string]float64     `json:"thresholds"`
	OutputFormat     string                 `json:"output_format"`     // json, report, summary
}

// TimeRange represents a time range for analysis
type TimeRange struct {
	StartTime *int64 `json:"start_time,omitempty"` // Unix timestamp in milliseconds
	EndTime   *int64 `json:"end_time,omitempty"`   // Unix timestamp in milliseconds
	Duration  *int   `json:"duration_hours,omitempty"` // Duration in hours from now
}

// DiagnosisFilter represents filtering criteria for diagnosis data
type DiagnosisFilter struct {
	DeviceID    string   `json:"device_id,omitempty"`
	Type        []string `json:"type,omitempty"`
	Category    []string `json:"category,omitempty"`
	Severity    []string `json:"severity,omitempty"`
	Source      []string `json:"source,omitempty"`
	StartTime   *int64   `json:"start_time,omitempty"`
	EndTime     *int64   `json:"end_time,omitempty"`
	Tags        []string `json:"tags,omitempty"`
}

// DiagnosisStats represents diagnosis statistics
type DiagnosisStats struct {
	TotalDiagnoses    int                `json:"total_diagnoses"`
	ActiveSessions    int                `json:"active_sessions"`
	CompletedSessions int                `json:"completed_sessions"`
	FailedSessions    int                `json:"failed_sessions"`
	TypeStats         map[string]int     `json:"type_stats"`
	SeverityStats     map[string]int     `json:"severity_stats"`
	DeviceStats       map[string]int     `json:"device_stats"`
	AnalyzerStats     map[string]int     `json:"analyzer_stats"`
	LastUpdated       time.Time          `json:"last_updated"`
}

// AnalyzerInfo represents information about an analyzer
type AnalyzerInfo struct {
	Name        string   `json:"name"`
	Type        string   `json:"type"`        // builtin, plugin, external
	Version     string   `json:"version"`
	Description string   `json:"description"`
	SupportedTypes []string `json:"supported_types"`
	Enabled     bool     `json:"enabled"`
	Config      map[string]interface{} `json:"config"`
}

// WiFiDiagnosisData represents WiFi-specific diagnosis data
type WiFiDiagnosisData struct {
	SSID         string  `json:"ssid"`
	BSSID        string  `json:"bssid"`
	Channel      int     `json:"channel"`
	Frequency    float64 `json:"frequency_mhz"`
	SignalStrength int   `json:"signal_strength_dbm"`
	NoiseLevel   int     `json:"noise_level_dbm"`
	SNR          float64 `json:"snr_db"`
	LinkQuality  int     `json:"link_quality_percent"`
	TxRate       float64 `json:"tx_rate_mbps"`
	RxRate       float64 `json:"rx_rate_mbps"`
	PacketLoss   float64 `json:"packet_loss_percent"`
	Latency      float64 `json:"latency_ms"`
	AuthType     string  `json:"auth_type"`
	Encryption   string  `json:"encryption"`
	ConnectedTime int64  `json:"connected_time_seconds"`
}

// NetworkDiagnosisData represents network-specific diagnosis data
type NetworkDiagnosisData struct {
	Interface    string  `json:"interface"`
	IPAddress    string  `json:"ip_address"`
	SubnetMask   string  `json:"subnet_mask"`
	Gateway      string  `json:"gateway"`
	DNSServers   []string `json:"dns_servers"`
	MTU          int     `json:"mtu"`
	Speed        int64   `json:"speed_mbps"`
	Duplex       string  `json:"duplex"`
	LinkStatus   string  `json:"link_status"`
	TxBytes      int64   `json:"tx_bytes"`
	RxBytes      int64   `json:"rx_bytes"`
	TxPackets    int64   `json:"tx_packets"`
	RxPackets    int64   `json:"rx_packets"`
	TxErrors     int64   `json:"tx_errors"`
	RxErrors     int64   `json:"rx_errors"`
	TxDropped    int64   `json:"tx_dropped"`
	RxDropped    int64   `json:"rx_dropped"`
}

// SystemDiagnosisData represents system-specific diagnosis data
type SystemDiagnosisData struct {
	CPUUsage      float64 `json:"cpu_usage_percent"`
	MemoryUsage   float64 `json:"memory_usage_percent"`
	DiskUsage     float64 `json:"disk_usage_percent"`
	LoadAverage   []float64 `json:"load_average"`
	Uptime        int64   `json:"uptime_seconds"`
	Temperature   float64 `json:"temperature_celsius"`
	Voltage       float64 `json:"voltage_volts"`
	ProcessCount  int     `json:"process_count"`
	ConnectionCount int   `json:"connection_count"`
	ErrorRate     float64 `json:"error_rate_percent"`
}