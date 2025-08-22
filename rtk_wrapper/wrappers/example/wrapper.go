package example

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"rtk_wrapper/pkg/types"
)

// ExampleWrapper 範例設備 wrapper 實現
type ExampleWrapper struct {
	name    string
	version string
	config  ExampleConfig
	enabled bool
}

// ExampleConfig 範例 wrapper 配置
type ExampleConfig struct {
	Name             string                 `yaml:"name"`
	Description      string                 `yaml:"description"`
	SupportedDevices []string               `yaml:"supported_devices"`
	TopicPrefix      string                 `yaml:"topic_prefix"`
	DeviceIDExtract  string                 `yaml:"device_id_extract"`
	MessageMapping   MessageMapping         `yaml:"message_mapping"`
	FieldMapping     FieldMapping           `yaml:"field_mapping"`
	Settings         map[string]interface{} `yaml:"settings"`
}

// MessageMapping 訊息類型映射
type MessageMapping struct {
	StateMessage     string `yaml:"state_message"`
	TelemetryMessage string `yaml:"telemetry_message"`
	EventMessage     string `yaml:"event_message"`
	CommandMessage   string `yaml:"command_message"`
}

// FieldMapping 欄位映射
type FieldMapping struct {
	DeviceID    string            `yaml:"device_id"`
	Status      string            `yaml:"status"`
	Temperature string            `yaml:"temperature"`
	Humidity    string            `yaml:"humidity"`
	Power       string            `yaml:"power"`
	Custom      map[string]string `yaml:"custom"`
}

// NewExampleWrapper 創建範例 wrapper
func NewExampleWrapper() *ExampleWrapper {
	return &ExampleWrapper{
		name:    "example",
		version: "1.0.0",
		enabled: true,
		config: ExampleConfig{
			Name:        "Example Device Wrapper",
			Description: "A sample wrapper for demonstrating message transformation",
			SupportedDevices: []string{
				"sensor", "switch", "light", "climate",
			},
			TopicPrefix:     "example",
			DeviceIDExtract: "{device_id}",
			MessageMapping: MessageMapping{
				StateMessage:     "state",
				TelemetryMessage: "telemetry",
				EventMessage:     "event",
				CommandMessage:   "command",
			},
			FieldMapping: FieldMapping{
				DeviceID:    "device_id",
				Status:      "status",
				Temperature: "temperature",
				Humidity:    "humidity",
				Power:       "power",
				Custom:      make(map[string]string),
			},
		},
	}
}

// Name 返回 wrapper 名稱
func (w *ExampleWrapper) Name() string {
	return w.name
}

// Version 返回 wrapper 版本
func (w *ExampleWrapper) Version() string {
	return w.version
}

// SupportedDeviceTypes 返回支援的設備類型
func (w *ExampleWrapper) SupportedDeviceTypes() []string {
	return w.config.SupportedDevices
}

// Transform 統一轉換入口
func (w *ExampleWrapper) Transform(wrapperMsg *types.WrapperMessage) (interface{}, error) {
	if wrapperMsg.Direction == types.DirectionUplink {
		return w.transformUplink(wrapperMsg)
	} else {
		return w.transformDownlink(wrapperMsg)
	}
}

// transformUplink 上行轉換（設備格式 → RTK 格式）
func (w *ExampleWrapper) transformUplink(wrapperMsg *types.WrapperMessage) (*types.RTKMessage, error) {
	log.Printf("ExampleWrapper: Processing uplink message from topic: %s", wrapperMsg.MQTTInfo.Topic)

	// 從 topic 提取設備 ID
	deviceID, err := w.extractDeviceID(wrapperMsg.MQTTInfo.Topic)
	if err != nil {
		return nil, fmt.Errorf("failed to extract device ID: %w", err)
	}

	// 推斷訊息類型
	messageType := w.inferMessageType(wrapperMsg.MQTTInfo.Topic, wrapperMsg.ParsedPayload)

	// 轉換 payload
	rtkPayload, err := w.convertToRTKPayload(wrapperMsg.ParsedPayload)
	if err != nil {
		return nil, fmt.Errorf("failed to convert payload: %w", err)
	}

	// 建構 RTK 格式訊息
	schema := fmt.Sprintf("%s/1.0", messageType)
	rtkMsg := types.NewRTKMessage(schema, deviceID, rtkPayload)

	// 添加追蹤資訊
	rtkMsg.Trace = &types.TraceInfo{
		RequestID: fmt.Sprintf("example_%d", time.Now().UnixNano()),
	}

	log.Printf("ExampleWrapper: Converted %s → RTK format (device: %s, schema: %s)",
		wrapperMsg.MQTTInfo.Topic, deviceID, schema)

	return rtkMsg, nil
}

// transformDownlink 下行轉換（RTK 格式 → 設備格式）
func (w *ExampleWrapper) transformDownlink(wrapperMsg *types.WrapperMessage) (*types.DeviceMessage, error) {
	log.Printf("ExampleWrapper: Processing downlink message from topic: %s", wrapperMsg.MQTTInfo.Topic)

	// 解析 RTK 訊息
	deviceID, err := w.extractRTKDeviceID(wrapperMsg.MQTTInfo.Topic)
	if err != nil {
		return nil, fmt.Errorf("failed to extract RTK device ID: %w", err)
	}

	// 轉換為設備格式的 payload
	devicePayload, err := w.convertToDevicePayload(wrapperMsg.ParsedPayload)
	if err != nil {
		return nil, fmt.Errorf("failed to convert to device payload: %w", err)
	}

	// 建構設備訊息
	deviceTopic := fmt.Sprintf("%s/%s/set", w.config.TopicPrefix, deviceID)
	deviceMsg := types.NewDeviceMessage(deviceTopic, devicePayload, 0, false)

	log.Printf("ExampleWrapper: Converted RTK → %s (device: %s)", deviceTopic, deviceID)

	return deviceMsg, nil
}

// extractDeviceID 從 topic 提取設備 ID
func (w *ExampleWrapper) extractDeviceID(topic string) (string, error) {
	// 範例 topic 格式: example/{device_id}/state
	parts := strings.Split(topic, "/")
	if len(parts) < 3 {
		return "", fmt.Errorf("invalid topic format: %s", topic)
	}

	if parts[0] != w.config.TopicPrefix {
		return "", fmt.Errorf("topic prefix mismatch: expected %s, got %s", w.config.TopicPrefix, parts[0])
	}

	return parts[1], nil
}

// extractRTKDeviceID 從 RTK topic 提取設備 ID
func (w *ExampleWrapper) extractRTKDeviceID(topic string) (string, error) {
	// RTK topic 格式: rtk/v1/{tenant}/{site}/{device_id}/{message_type}
	parts := strings.Split(topic, "/")
	if len(parts) < 6 {
		return "", fmt.Errorf("invalid RTK topic format: %s", topic)
	}

	return parts[4], nil
}

// inferMessageType 推斷訊息類型
func (w *ExampleWrapper) inferMessageType(topic string, payload *types.FlexiblePayload) string {
	// 從 topic 最後一部分推斷
	parts := strings.Split(topic, "/")
	if len(parts) > 0 {
		lastPart := parts[len(parts)-1]

		switch lastPart {
		case "state", "status":
			return "state"
		case "telemetry", "sensor":
			return "telemetry/sensor"
		case "event", "alert":
			return "evt/alert"
		default:
			return "state" // 預設為狀態訊息
		}
	}

	return "state"
}

// convertToRTKPayload 轉換為 RTK payload 格式
func (w *ExampleWrapper) convertToRTKPayload(payload *types.FlexiblePayload) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	// 設備健康狀態
	if status, exists := payload.GetString(w.config.FieldMapping.Status); exists {
		result["health"] = w.normalizeStatus(status)
	} else {
		result["health"] = "unknown"
	}

	// 溫度
	if temp, exists := payload.GetFloat64(w.config.FieldMapping.Temperature); exists {
		result["temperature"] = temp
	}

	// 濕度
	if humidity, exists := payload.GetFloat64(w.config.FieldMapping.Humidity); exists {
		result["humidity"] = humidity
	}

	// 電源狀態
	if power, exists := payload.GetString(w.config.FieldMapping.Power); exists {
		result["power_state"] = w.normalizePowerState(power)
	}

	// 複製其他自定義欄位
	for originalField, rtkField := range w.config.FieldMapping.Custom {
		if value, exists := payload.GetNested(originalField); exists {
			result[rtkField] = value
		}
	}

	// 如果沒有任何有效欄位，至少包含基本資訊
	if len(result) == 0 {
		result["raw_data"] = payload.Parsed
	}

	return result, nil
}

// convertToDevicePayload 轉換為設備 payload 格式
func (w *ExampleWrapper) convertToDevicePayload(payload *types.FlexiblePayload) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	// 從 RTK payload 轉換回設備格式
	if command, exists := payload.GetString("command"); exists {
		result["action"] = command
	}

	if value, exists := payload.GetNested("value"); exists {
		result["value"] = value
	}

	if brightness, exists := payload.GetFloat64("brightness"); exists {
		result["brightness"] = brightness
	}

	if temp, exists := payload.GetFloat64("temperature"); exists {
		result["set_temperature"] = temp
	}

	// 如果沒有任何已知欄位，直接轉發原始資料
	if len(result) == 0 {
		result = payload.Parsed
	}

	return result, nil
}

// normalizeStatus 正規化狀態值
func (w *ExampleWrapper) normalizeStatus(status string) string {
	status = strings.ToLower(status)
	switch status {
	case "online", "connected", "active", "on", "ok", "good":
		return "ok"
	case "offline", "disconnected", "inactive", "off", "error", "bad":
		return "error"
	case "warning", "warn", "degraded":
		return "warning"
	default:
		return "unknown"
	}
}

// normalizePowerState 正規化電源狀態
func (w *ExampleWrapper) normalizePowerState(power string) string {
	power = strings.ToLower(power)
	switch power {
	case "on", "true", "1", "enabled", "active":
		return "on"
	case "off", "false", "0", "disabled", "inactive":
		return "off"
	default:
		return "unknown"
	}
}

// Initialize 初始化 wrapper
func (w *ExampleWrapper) Initialize(config types.WrapperConfig) error {
	w.enabled = config.Enabled

	// 這裡可以載入特定的配置
	if settings := config.Settings; settings != nil {
		// 處理自定義設定
		if prefix, ok := settings["topic_prefix"].(string); ok {
			w.config.TopicPrefix = prefix
		}
	}

	log.Printf("ExampleWrapper: Initialized (enabled: %t)", w.enabled)
	return nil
}

// Start 啟動 wrapper
func (w *ExampleWrapper) Start(ctx context.Context) error {
	if !w.enabled {
		return fmt.Errorf("wrapper is disabled")
	}

	log.Printf("ExampleWrapper: Started")

	// 在實際實現中，這裡可能會啟動背景工作或定時器
	go func() {
		<-ctx.Done()
		log.Printf("ExampleWrapper: Context cancelled, stopping...")
	}()

	return nil
}

// Stop 停止 wrapper
func (w *ExampleWrapper) Stop() error {
	log.Printf("ExampleWrapper: Stopped")
	return nil
}

// HealthCheck 健康檢查
func (w *ExampleWrapper) HealthCheck() error {
	if !w.enabled {
		return fmt.Errorf("wrapper is disabled")
	}

	// 檢查 wrapper 是否正常運作
	return nil
}
