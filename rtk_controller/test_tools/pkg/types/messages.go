package types

// BaseMessage 基礎消息結構，符合 SPEC.md 規範
type BaseMessage struct {
	Schema    string      `json:"schema"`
	Timestamp string      `json:"ts"`
	Payload   interface{} `json:"payload"`
	Trace     *TraceInfo  `json:"trace,omitempty"`
}

// TraceInfo LLM 診斷追蹤資訊
type TraceInfo struct {
	ReqID         string `json:"req_id,omitempty"`
	CorrelationID string `json:"correlation_id,omitempty"`
	SessionID     string `json:"session_id,omitempty"`
	TraceID       string `json:"trace_id,omitempty"`
}

// StatePayload 狀態消息 payload
type StatePayload struct {
	Health      string                 `json:"health"`              // ok|warn|error
	Firmware    string                 `json:"fw"`
	UptimeS     int64                  `json:"uptime_s"`
	CPUUsage    float64                `json:"cpu_usage"`
	MemoryUsage float64                `json:"memory_usage"`
	DiskUsage   float64                `json:"disk_usage,omitempty"`
	TempC       float64                `json:"temperature_c,omitempty"`
	DeviceType  string                 `json:"device_type"`
	BatteryV    float64                `json:"battery_v,omitempty"`
	Net         NetworkInfo            `json:"net"`
	Diagnosis   DiagnosisInfo          `json:"diagnosis"`
	Extra       map[string]interface{} `json:",inline"` // 設備特定欄位
}

// NetworkInfo 網路資訊
type NetworkInfo struct {
	IP           string  `json:"ip"`
	Gateway      string  `json:"gateway,omitempty"`
	Netmask      string  `json:"netmask,omitempty"`
	RSSI         int     `json:"rssi,omitempty"`
	BytesRx      int64   `json:"bytes_rx,omitempty"`
	BytesTx      int64   `json:"bytes_tx,omitempty"`
	ParentDevice string  `json:"parent_device,omitempty"`
}

// DiagnosisInfo 診斷資訊
type DiagnosisInfo struct {
	LastError    *string `json:"last_error"`
	ErrorCount   int     `json:"error_count"`
	RestartCount int     `json:"restart_count"`
}

// TelemetryPayload 遙測消息 payload（通用）
type TelemetryPayload map[string]interface{}

// EventPayload 事件消息 payload
type EventPayload struct {
	EventType string                 `json:"event_type"`
	Severity  string                 `json:"severity"` // info|warning|error
	Extra     map[string]interface{} `json:",inline"`  // 事件特定欄位
}

// WiFiStatus WiFi 狀態資訊
type WiFiStatus struct {
	APMode      bool `json:"ap_mode"`
	ClientCount int  `json:"client_count"`
	Channel2G   int  `json:"channel_2g"`
	Channel5G   int  `json:"channel_5g,omitempty"`
	TxPower2G   int  `json:"tx_power_2g"`
	TxPower5G   int  `json:"tx_power_5g,omitempty"`
}

// InterfaceInfo 介面資訊
type InterfaceInfo struct {
	Name       string `json:"name"`
	MACAddress string `json:"mac_address"`
	SpeedMbps  int    `json:"speed_mbps"`
	Duplex     string `json:"duplex"`
	LinkStatus string `json:"link_status"`
}

// PortInfo 端口資訊
type PortInfo struct {
	Status    string `json:"status"`
	SpeedMbps *int   `json:"speed_mbps"`
	Duplex    *string `json:"duplex"`
}

// DeviceConfig 設備配置
type DeviceConfig struct {
	ID         string            `yaml:"id"`
	Type       string            `yaml:"type"`        // gateway|iot_sensor|nic|switch
	Tenant     string            `yaml:"tenant"`
	Site       string            `yaml:"site"`
	Enabled    bool              `yaml:"enabled"`
	Intervals  IntervalConfig    `yaml:"intervals"`
	Properties map[string]interface{} `yaml:"properties"`
}

// IntervalConfig 消息發送間隔配置
type IntervalConfig struct {
	StateS     int `yaml:"state_seconds"`     // 狀態消息間隔
	TelemetryS int `yaml:"telemetry_seconds"` // 遙測消息間隔
	EventS     int `yaml:"event_seconds"`     // 事件檢查間隔
}

// TestConfig 測試配置
type TestConfig struct {
	MQTT    MQTTConfig     `yaml:"mqtt"`
	Devices []DeviceConfig `yaml:"devices"`
	Test    TestSettings   `yaml:"test"`
}

// MQTTConfig MQTT 連接配置
type MQTTConfig struct {
	Broker   string `yaml:"broker"`
	Port     int    `yaml:"port"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	ClientIDPrefix string `yaml:"client_id_prefix"`
}

// TestSettings 測試設定
type TestSettings struct {
	DurationS int  `yaml:"duration_seconds"`
	Verbose   bool `yaml:"verbose"`
	LogLevel  string `yaml:"log_level"`
}