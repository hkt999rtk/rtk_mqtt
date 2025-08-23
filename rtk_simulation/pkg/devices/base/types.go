package base

import (
	"context"
	"time"
)

// Device 定義所有設備的通用介面
type Device interface {
	// 基本設備資訊
	GetDeviceID() string
	GetDeviceType() string
	GetMACAddress() string
	GetIPAddress() string
	GetTenant() string
	GetSite() string

	// 生命週期管理
	Start(ctx context.Context) error
	Stop() error
	IsRunning() bool

	// 狀態管理
	UpdateStatus()
	GetHealth() string
	GetUptime() time.Duration

	// 網路資訊
	GetNetworkInfo() NetworkInfo

	// MQTT 消息生成
	GenerateStatePayload() StatePayload
	GenerateTelemetryData() map[string]TelemetryPayload
	GenerateEvents() []Event

	// 命令處理
	HandleCommand(cmd Command) error
}

// NetworkInfo 網路資訊結構
type NetworkInfo struct {
	IP             string `json:"ip"`
	Gateway        string `json:"gateway,omitempty"`
	Netmask        string `json:"netmask,omitempty"`
	MACAddress     string `json:"mac_address"`
	RSSI           int    `json:"rssi,omitempty"`          // WiFi 信號強度 (dBm)
	BytesRx        int64  `json:"bytes_rx,omitempty"`      // 接收位元組數
	BytesTx        int64  `json:"bytes_tx,omitempty"`      // 傳送位元組數
	PacketsRx      int64  `json:"packets_rx,omitempty"`    // 接收封包數
	PacketsTx      int64  `json:"packets_tx,omitempty"`    // 傳送封包數
	ParentDevice   string `json:"parent_device,omitempty"` // 父設備 ID
	ConnectionType string `json:"connection_type"`         // ethernet, wifi, zigbee, etc.
}

// StatePayload RTK MQTT 狀態訊息格式
type StatePayload struct {
	Health      string                 `json:"health"` // ok|warn|error
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
	Extra       map[string]interface{} `json:"extra,omitempty"` // 設備特定欄位
}

// DiagnosisInfo 診斷資訊
type DiagnosisInfo struct {
	LastError    *string `json:"last_error"`
	ErrorCount   int     `json:"error_count"`
	RestartCount int     `json:"restart_count"`
}

// TelemetryPayload 遙測資料格式
type TelemetryPayload map[string]interface{}

// Event 事件結構
type Event struct {
	EventType string                 `json:"event_type"`
	Severity  string                 `json:"severity"` // info|warning|error
	Message   string                 `json:"message"`
	Extra     map[string]interface{} `json:"extra,omitempty"`
}

// Command 命令結構
type Command struct {
	ID         string                 `json:"id"`
	Type       string                 `json:"type"`
	Action     string                 `json:"action,omitempty"`
	Payload    string                 `json:"payload,omitempty"`
	Parameters map[string]interface{} `json:"parameters,omitempty"`
	Timeout    time.Duration          `json:"timeout,omitempty"`
}

// CommandResult 命令執行結果
type CommandResult struct {
	CommandID string                 `json:"command_id"`
	Success   bool                   `json:"success"`
	Message   string                 `json:"message"`
	Data      map[string]interface{} `json:"data,omitempty"`
	Error     string                 `json:"error,omitempty"`
}

// CommandResponse 命令回應
type CommandResponse struct {
	CommandID string `json:"command_id"`
	Status    string `json:"status"`
	Data      string `json:"data,omitempty"`
	Error     string `json:"error,omitempty"`
}

// DeviceStats 設備統計資訊
type DeviceStats struct {
	MessagesPublished int64         `json:"messages_published"`
	MessagesFailed    int64         `json:"messages_failed"`
	LastActivity      time.Time     `json:"last_activity"`
	Errors            []DeviceError `json:"errors,omitempty"`
}

// DeviceError 設備錯誤記錄
type DeviceError struct {
	Timestamp time.Time `json:"timestamp"`
	Type      string    `json:"type"`
	Message   string    `json:"message"`
	Severity  string    `json:"severity"`
}

// DeviceCapabilities 設備能力聲明
type DeviceCapabilities struct {
	DeviceType     string                 `json:"device_type"`
	Diagnostics    []string               `json:"diagnostics,omitempty"` // 支援的診斷功能
	Controls       []string               `json:"controls,omitempty"`    // 支援的控制功能
	Sensors        []string               `json:"sensors,omitempty"`     // 支援的感測器
	Protocols      []string               `json:"protocols"`             // 支援的通信協議
	LLMIntegration *LLMCapabilities       `json:"llm_integration,omitempty"`
	Extra          map[string]interface{} `json:"extra,omitempty"`
}

// LLMCapabilities LLM 整合能力
type LLMCapabilities struct {
	Supported       bool     `json:"supported"`
	SessionTracking bool     `json:"session_tracking"`
	ToolCategories  []string `json:"tool_categories"` // read, test, act
}

// Location 設備位置資訊
type Location struct {
	Room        string `json:"room,omitempty"`
	Building    string `json:"building,omitempty"`
	Floor       string `json:"floor,omitempty"`
	Coordinates *Coord `json:"coordinates,omitempty"`
}

// Coord 座標資訊
type Coord struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z,omitempty"`
}
