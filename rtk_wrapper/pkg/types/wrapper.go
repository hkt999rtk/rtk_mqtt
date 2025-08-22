package types

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// DeviceWrapper 定義設備包裝器接口
type DeviceWrapper interface {
	// 基本資訊
	Name() string
	Version() string
	SupportedDeviceTypes() []string

	// 訊息轉換
	Transform(wrapperMsg *WrapperMessage) (interface{}, error)

	// 生命週期管理
	Initialize(config WrapperConfig) error
	Start(ctx context.Context) error
	Stop() error

	// 健康檢查
	HealthCheck() error
}

// MessageTransformer 訊息轉換器接口（雙向轉換）
type MessageTransformer interface {
	// 上行轉換：設備格式 -> RTK 格式
	TransformUplink(wrapperMsg *WrapperMessage) (*RTKMessage, error)

	// 下行轉換：RTK 格式 -> 設備格式
	TransformDownlink(wrapperMsg *WrapperMessage) (*DeviceMessage, error)

	// 統一轉換入口（自動判斷方向）
	Transform(wrapperMsg *WrapperMessage) (interface{}, error)

	// 驗證訊息格式
	ValidateUplink(wrapperMsg *WrapperMessage) error
	ValidateDownlink(wrapperMsg *WrapperMessage) error

	// 支援的訊息格式檢測
	CanHandleUplink(topic string, payload *FlexiblePayload) bool   // 能否處理設備→RTK
	CanHandleDownlink(topic string, payload *FlexiblePayload) bool // 能否處理RTK→設備
}

// MQTT 原始訊息結構（直接對應 MQTT packet）
type MQTTRawMessage struct {
	Topic     string `json:"topic"`     // MQTT topic
	Payload   []byte `json:"payload"`   // MQTT payload（原始 bytes）
	QoS       byte   `json:"qos"`       // MQTT QoS
	Retained  bool   `json:"retained"`  // MQTT retained flag
	Timestamp int64  `json:"timestamp"` // Wrapper 接收時間戳
}

// 訊息方向定義
type MessageDirection int

const (
	DirectionUplink   MessageDirection = iota // Device → RTK (Non-RTK → RTK)
	DirectionDownlink                         // RTK → Device (RTK → Non-RTK)
)

// String implements the Stringer interface for MessageDirection
func (md MessageDirection) String() string {
	switch md {
	case DirectionUplink:
		return "uplink"
	case DirectionDownlink:
		return "downlink"
	default:
		return "unknown"
	}
}

// 訊息來源類型
type MessageSource int

const (
	SourceDevice MessageSource = iota // 來自設備（非RTK格式）
	SourceRTK                         // 來自RTK Controller（RTK格式）
)

// 統一的訊息處理結構
type WrapperMessage struct {
	// 基本 MQTT 資訊
	MQTTInfo MQTTRawMessage `json:"mqtt_info"`

	// 訊息方向和來源
	Direction MessageDirection `json:"direction"`
	Source    MessageSource    `json:"source"`

	// 解析後的 JSON payload（彈性處理）
	ParsedPayload *FlexiblePayload `json:"parsed_payload"`

	// 目標資訊（用於路由）
	Target *MessageTarget `json:"target,omitempty"`

	// Wrapper 添加的元資料
	Meta map[string]interface{} `json:"meta,omitempty"`
}

// 彈性 JSON 處理輔助類型（處理 MQTT payload 中的 JSON 內容）
type FlexiblePayload struct {
	Raw    json.RawMessage        `json:"-"` // 原始 JSON bytes
	Parsed map[string]interface{} `json:"-"` // 解析後的通用結構
}

// 提供彈性解析方法
func (fp *FlexiblePayload) UnmarshalJSON(data []byte) error {
	fp.Raw = data
	return json.Unmarshal(data, &fp.Parsed)
}

func (fp *FlexiblePayload) GetString(key string) (string, bool) {
	if val, exists := fp.Parsed[key]; exists {
		if str, ok := val.(string); ok {
			return str, true
		}
	}
	return "", false
}

func (fp *FlexiblePayload) GetFloat64(key string) (float64, bool) {
	if val, exists := fp.Parsed[key]; exists {
		if num, ok := val.(float64); ok {
			return num, true
		}
	}
	return 0, false
}

func (fp *FlexiblePayload) GetBool(key string) (bool, bool) {
	if val, exists := fp.Parsed[key]; exists {
		if b, ok := val.(bool); ok {
			return b, true
		}
	}
	return false, false
}

func (fp *FlexiblePayload) GetNested(path ...string) (interface{}, bool) {
	current := fp.Parsed
	for _, key := range path {
		if val, exists := current[key]; exists {
			if nested, ok := val.(map[string]interface{}); ok {
				current = nested
			} else {
				return val, true // 最後一個層級
			}
		} else {
			return nil, false
		}
	}
	return current, true
}

// 訊息目標資訊
type MessageTarget struct {
	DeviceID   string            `json:"device_id"`
	DeviceType string            `json:"device_type"`
	Topic      string            `json:"topic"`
	Meta       map[string]string `json:"meta,omitempty"`
}

// RTK 格式訊息
type RTKMessage struct {
	Schema    string                 `json:"schema"`
	Timestamp int64                  `json:"ts"`
	DeviceID  string                 `json:"device_id"`
	Payload   map[string]interface{} `json:"payload"`
	Trace     *TraceInfo             `json:"trace,omitempty"`
}

// 設備格式訊息
type DeviceMessage struct {
	Topic    string                 `json:"topic"`
	Payload  map[string]interface{} `json:"payload"`
	QoS      byte                   `json:"qos"`
	Retained bool                   `json:"retained"`
}

// 追蹤資訊
type TraceInfo struct {
	RequestID string `json:"req_id,omitempty"`
	SessionID string `json:"session_id,omitempty"`
}

// Wrapper 配置
type WrapperConfig struct {
	Name       string                 `json:"name"`
	Version    string                 `json:"version"`
	Enabled    bool                   `json:"enabled"`
	Priority   int                    `json:"priority"`
	ConfigFile string                 `json:"config_file"`
	Settings   map[string]interface{} `json:"settings,omitempty"`
}

// 訊息創建工廠函數
func NewWrapperMessage(mqttMsg MQTTRawMessage, direction MessageDirection, source MessageSource) (*WrapperMessage, error) {
	var payload FlexiblePayload

	// 嘗試解析 MQTT payload 為 JSON
	if len(mqttMsg.Payload) > 0 {
		payload.Raw = json.RawMessage(mqttMsg.Payload)
		if err := json.Unmarshal(mqttMsg.Payload, &payload.Parsed); err != nil {
			return nil, fmt.Errorf("failed to parse MQTT payload as JSON: %w", err)
		}
	}

	return &WrapperMessage{
		MQTTInfo:      mqttMsg,
		Direction:     direction,
		Source:        source,
		ParsedPayload: &payload,
		Meta:          make(map[string]interface{}),
	}, nil
}

// 便利函數：創建上行訊息（設備 → RTK）
func NewUplinkMessage(mqttMsg MQTTRawMessage) (*WrapperMessage, error) {
	return NewWrapperMessage(mqttMsg, DirectionUplink, SourceDevice)
}

// 便利函數：創建下行訊息（RTK → 設備）
func NewDownlinkMessage(mqttMsg MQTTRawMessage) (*WrapperMessage, error) {
	return NewWrapperMessage(mqttMsg, DirectionDownlink, SourceRTK)
}

// 檢查是否為 RTK 格式的訊息
func IsRTKMessage(topic string) bool {
	return len(topic) >= 7 && topic[:7] == "rtk/v1/"
}

// 自動判斷訊息方向和來源
func NewWrapperMessageAuto(mqttMsg MQTTRawMessage) (*WrapperMessage, error) {
	if IsRTKMessage(mqttMsg.Topic) {
		// RTK 格式訊息，下行方向
		return NewWrapperMessage(mqttMsg, DirectionDownlink, SourceRTK)
	} else {
		// 非 RTK 格式訊息，上行方向
		return NewWrapperMessage(mqttMsg, DirectionUplink, SourceDevice)
	}
}

// 創建 RTK 訊息
func NewRTKMessage(schema, deviceID string, payload map[string]interface{}) *RTKMessage {
	return &RTKMessage{
		Schema:    schema,
		Timestamp: time.Now().UnixMilli(),
		DeviceID:  deviceID,
		Payload:   payload,
	}
}

// 創建設備訊息
func NewDeviceMessage(topic string, payload map[string]interface{}, qos byte, retained bool) *DeviceMessage {
	return &DeviceMessage{
		Topic:    topic,
		Payload:  payload,
		QoS:      qos,
		Retained: retained,
	}
}
