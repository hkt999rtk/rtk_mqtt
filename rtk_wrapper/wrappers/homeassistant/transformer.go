package homeassistant

import (
	"fmt"
	"strings"

	"rtk_wrapper/pkg/types"
)

// HATransformer 實現 MessageTransformer 接口
type HATransformer struct {
	wrapper *HomeAssistantWrapper
}

// NewHATransformer 創建 Home Assistant transformer
func NewHATransformer(wrapper *HomeAssistantWrapper) *HATransformer {
	return &HATransformer{
		wrapper: wrapper,
	}
}

// TransformUplink 上行轉換：HA 格式 -> RTK 格式
func (t *HATransformer) TransformUplink(wrapperMsg *types.WrapperMessage) (*types.RTKMessage, error) {
	if wrapperMsg.Direction != types.DirectionUplink {
		return nil, fmt.Errorf("message direction is not uplink")
	}

	// 驗證上行訊息
	if err := t.ValidateUplink(wrapperMsg); err != nil {
		return nil, fmt.Errorf("uplink validation failed: %w", err)
	}

	// 委託給 wrapper 的實現
	return t.wrapper.transformUplink(wrapperMsg)
}

// TransformDownlink 下行轉換：RTK 格式 -> HA 格式
func (t *HATransformer) TransformDownlink(wrapperMsg *types.WrapperMessage) (*types.DeviceMessage, error) {
	if wrapperMsg.Direction != types.DirectionDownlink {
		return nil, fmt.Errorf("message direction is not downlink")
	}

	// 驗證下行訊息
	if err := t.ValidateDownlink(wrapperMsg); err != nil {
		return nil, fmt.Errorf("downlink validation failed: %w", err)
	}

	// 委託給 wrapper 的實現
	return t.wrapper.transformDownlink(wrapperMsg)
}

// Transform 統一轉換入口（自動判斷方向）
func (t *HATransformer) Transform(wrapperMsg *types.WrapperMessage) (interface{}, error) {
	if wrapperMsg.Direction == types.DirectionUplink {
		return t.TransformUplink(wrapperMsg)
	} else if wrapperMsg.Direction == types.DirectionDownlink {
		return t.TransformDownlink(wrapperMsg)
	} else {
		return nil, fmt.Errorf("unknown message direction: %d", wrapperMsg.Direction)
	}
}

// ValidateUplink 驗證上行訊息格式
func (t *HATransformer) ValidateUplink(wrapperMsg *types.WrapperMessage) error {
	topic := wrapperMsg.MQTTInfo.Topic

	// 檢查 Home Assistant topic 格式
	if !strings.HasPrefix(topic, "homeassistant/") {
		return fmt.Errorf("not a Home Assistant topic: %s", topic)
	}

	// 檢查 topic 結構
	parts := strings.Split(topic, "/")
	if len(parts) < 4 {
		return fmt.Errorf("invalid HA topic structure, expected at least 4 parts: %s", topic)
	}

	// 檢查設備類型是否支援
	deviceClass := parts[1]
	if !t.isDeviceClassSupported(deviceClass) {
		return fmt.Errorf("unsupported device class: %s", deviceClass)
	}

	// 檢查訊息類型
	messageType := parts[len(parts)-1]
	supportedMessageTypes := []string{"state", "attributes", "config", "availability"}
	if !contains(supportedMessageTypes, messageType) {
		return fmt.Errorf("unsupported message type: %s", messageType)
	}

	// 檢查 payload
	if wrapperMsg.ParsedPayload == nil {
		return fmt.Errorf("parsed payload is nil")
	}

	return nil
}

// ValidateDownlink 驗證下行訊息格式
func (t *HATransformer) ValidateDownlink(wrapperMsg *types.WrapperMessage) error {
	topic := wrapperMsg.MQTTInfo.Topic

	// 檢查 RTK topic 格式
	if !strings.HasPrefix(topic, "rtk/v1/") {
		return fmt.Errorf("not a valid RTK topic: %s", topic)
	}

	// 檢查 RTK topic 結構
	parts := strings.Split(topic, "/")
	if len(parts) < 6 {
		return fmt.Errorf("invalid RTK topic structure, expected at least 6 parts: %s", topic)
	}

	// 檢查是否為命令訊息
	messageType := strings.Join(parts[5:], "/")
	if !strings.HasPrefix(messageType, "cmd/") {
		return fmt.Errorf("not a command message: %s", messageType)
	}

	// 檢查 payload
	if wrapperMsg.ParsedPayload == nil {
		return fmt.Errorf("parsed payload is nil")
	}

	return nil
}

// CanHandleUplink 能否處理上行訊息（HA→RTK）
func (t *HATransformer) CanHandleUplink(topic string, payload *types.FlexiblePayload) bool {
	// 檢查 topic 前綴
	if !strings.HasPrefix(topic, "homeassistant/") {
		return false
	}

	parts := strings.Split(topic, "/")
	if len(parts) < 4 {
		return false
	}

	// 檢查設備類型
	deviceClass := parts[1]
	if !t.isDeviceClassSupported(deviceClass) {
		return false
	}

	// 檢查訊息類型
	messageType := parts[len(parts)-1]
	supportedMessageTypes := []string{"state", "attributes", "availability"}
	if !contains(supportedMessageTypes, messageType) {
		return false
	}

	// 進階檢查：根據設備類型驗證 payload 內容
	if payload != nil {
		return t.validatePayloadForDeviceClass(deviceClass, payload)
	}

	return true
}

// CanHandleDownlink 能否處理下行訊息（RTK→HA）
func (t *HATransformer) CanHandleDownlink(topic string, payload *types.FlexiblePayload) bool {
	// 檢查是否為 RTK 命令格式
	if !strings.HasPrefix(topic, "rtk/v1/") {
		return false
	}

	parts := strings.Split(topic, "/")
	if len(parts) < 6 {
		return false
	}

	// 檢查訊息類型
	messageType := strings.Join(parts[5:], "/")
	if !strings.HasPrefix(messageType, "cmd/") {
		return false
	}

	// 從設備 ID 推斷是否為 HA 設備
	deviceID := parts[4]
	deviceClass := t.wrapper.inferHADeviceClass(deviceID)

	if !t.isDeviceClassSupported(deviceClass) {
		return false
	}

	// 檢查命令是否為 HA 支援的命令
	if payload != nil {
		return t.validateHACommand(payload)
	}

	return true
}

// isDeviceClassSupported 檢查設備類型是否支援
func (t *HATransformer) isDeviceClassSupported(deviceClass string) bool {
	for _, supported := range t.wrapper.config.SupportedDeviceTypes {
		if deviceClass == supported {
			return true
		}
	}
	return false
}

// validatePayloadForDeviceClass 根據設備類型驗證 payload
func (t *HATransformer) validatePayloadForDeviceClass(deviceClass string, payload *types.FlexiblePayload) bool {
	switch deviceClass {
	case "light":
		// 燈光設備應該有 state, brightness, color 等欄位之一
		expectedFields := []string{"state", "brightness", "color_temp", "rgb_color"}
		return t.hasAnyField(payload, expectedFields)

	case "switch":
		// 開關設備應該有 state 或能源相關欄位
		expectedFields := []string{"state", "power", "energy", "voltage", "current"}
		return t.hasAnyField(payload, expectedFields)

	case "sensor":
		// 感測器應該有數值或狀態欄位
		expectedFields := []string{"state", "temperature", "humidity", "pressure", "battery", "value"}
		return t.hasAnyField(payload, expectedFields)

	case "climate":
		// 空調設備應該有溫度相關欄位
		expectedFields := []string{"current_temperature", "temperature", "mode", "fan_mode"}
		return t.hasAnyField(payload, expectedFields)

	case "cover":
		// 窗簾設備應該有狀態或位置欄位
		expectedFields := []string{"state", "position", "tilt_position"}
		return t.hasAnyField(payload, expectedFields)

	case "binary_sensor":
		// 二元感測器應該有狀態欄位
		expectedFields := []string{"state"}
		return t.hasAnyField(payload, expectedFields)

	default:
		// 其他設備類型，寬鬆檢查
		return len(payload.Parsed) > 0
	}
}

// validateHACommand 驗證 HA 命令
func (t *HATransformer) validateHACommand(payload *types.FlexiblePayload) bool {
	// 檢查是否有命令欄位
	if command, exists := payload.GetString("command"); exists {
		// 檢查是否為 HA 支援的命令
		supportedCommands := []string{
			"turn_on", "turn_off", "toggle",
			"set_brightness", "set_color", "set_temperature",
			"open", "close", "stop", "set_position",
		}
		return contains(supportedCommands, command)
	}

	// 檢查是否有服務調用資訊
	if service, exists := payload.GetString("service"); exists {
		// 檢查是否為 HA 服務
		return strings.Contains(service, ".")
	}

	// 檢查是否有直接的控制參數
	controlFields := []string{"state", "brightness", "temperature", "position"}
	return t.hasAnyField(payload, controlFields)
}

// hasAnyField 檢查 payload 是否包含任意指定欄位
func (t *HATransformer) hasAnyField(payload *types.FlexiblePayload, fields []string) bool {
	for _, field := range fields {
		if _, exists := payload.GetNested(field); exists {
			return true
		}
	}
	return false
}

// GetScore 獲取處理此訊息的分數（分數越高越適合）
func (t *HATransformer) GetScore(wrapperMsg *types.WrapperMessage) int {
	score := 0
	topic := wrapperMsg.MQTTInfo.Topic

	if wrapperMsg.Direction == types.DirectionUplink {
		// 上行評分
		if strings.HasPrefix(topic, "homeassistant/") {
			score += 80 // HA topic 高分
		}

		parts := strings.Split(topic, "/")
		if len(parts) >= 4 {
			score += 20

			// 檢查設備類型支援度
			deviceClass := parts[1]
			if t.isDeviceClassSupported(deviceClass) {
				score += 30
			}

			// 檢查訊息類型
			messageType := parts[len(parts)-1]
			if messageType == "state" {
				score += 20 // 狀態訊息最重要
			} else if messageType == "attributes" {
				score += 15
			}
		}

		// 根據 payload 內容加分
		if wrapperMsg.ParsedPayload != nil {
			if t.validatePayloadForDeviceClass(parts[1], wrapperMsg.ParsedPayload) {
				score += 25
			}
		}

	} else if wrapperMsg.Direction == types.DirectionDownlink {
		// 下行評分
		if strings.HasPrefix(topic, "rtk/v1/") {
			score += 50
		}

		parts := strings.Split(topic, "/")
		if len(parts) >= 6 {
			score += 20

			// 檢查是否為命令
			messageType := strings.Join(parts[5:], "/")
			if strings.HasPrefix(messageType, "cmd/") {
				score += 30
			}

			// 根據設備 ID 推斷適合度
			deviceID := parts[4]
			deviceClass := t.wrapper.inferHADeviceClass(deviceID)
			if t.isDeviceClassSupported(deviceClass) {
				score += 25
			}
		}

		// 根據命令內容加分
		if wrapperMsg.ParsedPayload != nil {
			if t.validateHACommand(wrapperMsg.ParsedPayload) {
				score += 20
			}
		}
	}

	return score
}

// SupportsDeviceType 檢查是否支援指定設備類型
func (t *HATransformer) SupportsDeviceType(deviceType string) bool {
	return t.isDeviceClassSupported(deviceType)
}

// GetSupportedTopicPatterns 獲取支援的 topic 模式
func (t *HATransformer) GetSupportedTopicPatterns() []string {
	patterns := make([]string, 0)

	// 上行模式
	for _, deviceType := range t.wrapper.config.SupportedDeviceTypes {
		patterns = append(patterns, fmt.Sprintf("homeassistant/%s/{device_name}/state", deviceType))
		patterns = append(patterns, fmt.Sprintf("homeassistant/%s/{device_name}/attributes", deviceType))
		patterns = append(patterns, fmt.Sprintf("homeassistant/%s/{location}/{device_name}/state", deviceType))
	}

	// 下行模式
	patterns = append(patterns, "rtk/v1/{tenant}/{site}/{device_id}/cmd/req")

	return patterns
}

// GetDeviceClassFromTopic 從 topic 提取設備類別
func (t *HATransformer) GetDeviceClassFromTopic(topic string) string {
	parts := strings.Split(topic, "/")
	if len(parts) >= 2 && parts[0] == "homeassistant" {
		return parts[1]
	}
	return ""
}

// BuildTargetTopic 構建目標 topic
func (t *HATransformer) BuildTargetTopic(deviceClass, deviceID string) string {
	return t.wrapper.buildHATopic(deviceClass, deviceID)
}

// contains 輔助函數：檢查切片是否包含元素
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
