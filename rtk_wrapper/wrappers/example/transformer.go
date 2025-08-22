package example

import (
	"fmt"
	"log"
	"strings"

	"rtk_wrapper/pkg/types"
)

// ExampleTransformer 實現 MessageTransformer 接口
type ExampleTransformer struct {
	wrapper *ExampleWrapper
}

// NewExampleTransformer 創建範例 transformer
func NewExampleTransformer(wrapper *ExampleWrapper) *ExampleTransformer {
	return &ExampleTransformer{
		wrapper: wrapper,
	}
}

// TransformUplink 上行轉換：設備格式 -> RTK 格式
func (t *ExampleTransformer) TransformUplink(wrapperMsg *types.WrapperMessage) (*types.RTKMessage, error) {
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

// TransformDownlink 下行轉換：RTK 格式 -> 設備格式
func (t *ExampleTransformer) TransformDownlink(wrapperMsg *types.WrapperMessage) (*types.DeviceMessage, error) {
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
func (t *ExampleTransformer) Transform(wrapperMsg *types.WrapperMessage) (interface{}, error) {
	if wrapperMsg.Direction == types.DirectionUplink {
		return t.TransformUplink(wrapperMsg)
	} else if wrapperMsg.Direction == types.DirectionDownlink {
		return t.TransformDownlink(wrapperMsg)
	} else {
		return nil, fmt.Errorf("unknown message direction: %d", wrapperMsg.Direction)
	}
}

// ValidateUplink 驗證上行訊息格式
func (t *ExampleTransformer) ValidateUplink(wrapperMsg *types.WrapperMessage) error {
	// 檢查 topic 格式
	topic := wrapperMsg.MQTTInfo.Topic
	if !strings.HasPrefix(topic, t.wrapper.config.TopicPrefix) {
		return fmt.Errorf("topic does not match prefix: %s", t.wrapper.config.TopicPrefix)
	}

	// 檢查 topic 結構
	parts := strings.Split(topic, "/")
	if len(parts) < 3 {
		return fmt.Errorf("invalid topic structure, expected at least 3 parts: %s", topic)
	}

	// 檢查 payload 是否存在
	if wrapperMsg.ParsedPayload == nil {
		return fmt.Errorf("parsed payload is nil")
	}

	if len(wrapperMsg.ParsedPayload.Parsed) == 0 {
		log.Printf("ExampleTransformer: Warning - empty payload for topic: %s", topic)
	}

	return nil
}

// ValidateDownlink 驗證下行訊息格式
func (t *ExampleTransformer) ValidateDownlink(wrapperMsg *types.WrapperMessage) error {
	// 檢查 RTK topic 格式
	topic := wrapperMsg.MQTTInfo.Topic
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

	// 檢查 payload 是否存在
	if wrapperMsg.ParsedPayload == nil {
		return fmt.Errorf("parsed payload is nil")
	}

	return nil
}

// CanHandleUplink 能否處理上行訊息（設備→RTK）
func (t *ExampleTransformer) CanHandleUplink(topic string, payload *types.FlexiblePayload) bool {
	// 檢查 topic 前綴
	if !strings.HasPrefix(topic, t.wrapper.config.TopicPrefix) {
		return false
	}

	// 檢查 topic 結構
	parts := strings.Split(topic, "/")
	if len(parts) < 3 {
		return false
	}

	// 檢查訊息類型
	lastPart := parts[len(parts)-1]
	supportedTypes := []string{"state", "status", "telemetry", "sensor", "event", "alert"}

	for _, supportedType := range supportedTypes {
		if lastPart == supportedType {
			return true
		}
	}

	// 如果 payload 包含已知欄位，也可以處理
	if payload != nil {
		knownFields := []string{
			t.wrapper.config.FieldMapping.Status,
			t.wrapper.config.FieldMapping.Temperature,
			t.wrapper.config.FieldMapping.Humidity,
			t.wrapper.config.FieldMapping.Power,
		}

		for _, field := range knownFields {
			if field != "" {
				if _, exists := payload.GetNested(field); exists {
					return true
				}
			}
		}
	}

	return false
}

// CanHandleDownlink 能否處理下行訊息（RTK→設備）
func (t *ExampleTransformer) CanHandleDownlink(topic string, payload *types.FlexiblePayload) bool {
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

	// 從設備 ID 推斷是否可以處理
	deviceID := parts[4]
	deviceID = strings.ToLower(deviceID)

	// 檢查設備 ID 是否符合支援的設備類型
	for _, deviceType := range t.wrapper.config.SupportedDevices {
		if strings.Contains(deviceID, deviceType) {
			return true
		}
	}

	// 檢查 payload 是否包含已知命令
	if payload != nil {
		knownCommands := []string{"turn_on", "turn_off", "set_temperature", "set_brightness"}

		if command, exists := payload.GetString("command"); exists {
			for _, knownCmd := range knownCommands {
				if command == knownCmd {
					return true
				}
			}
		}
	}

	return false
}

// GetScore 獲取處理此訊息的分數（分數越高越適合）
func (t *ExampleTransformer) GetScore(wrapperMsg *types.WrapperMessage) int {
	score := 0
	topic := wrapperMsg.MQTTInfo.Topic

	if wrapperMsg.Direction == types.DirectionUplink {
		// 上行評分
		if strings.HasPrefix(topic, t.wrapper.config.TopicPrefix) {
			score += 50
		}

		parts := strings.Split(topic, "/")
		if len(parts) >= 3 {
			score += 20
		}

		// 根據 payload 內容加分
		if wrapperMsg.ParsedPayload != nil {
			if _, exists := wrapperMsg.ParsedPayload.GetString(t.wrapper.config.FieldMapping.Status); exists {
				score += 15
			}
			if _, exists := wrapperMsg.ParsedPayload.GetFloat64(t.wrapper.config.FieldMapping.Temperature); exists {
				score += 10
			}
			if _, exists := wrapperMsg.ParsedPayload.GetFloat64(t.wrapper.config.FieldMapping.Humidity); exists {
				score += 10
			}
		}

	} else if wrapperMsg.Direction == types.DirectionDownlink {
		// 下行評分
		if strings.HasPrefix(topic, "rtk/v1/") {
			score += 30
		}

		parts := strings.Split(topic, "/")
		if len(parts) >= 6 {
			score += 20
		}

		// 檢查設備 ID 匹配度
		if len(parts) >= 5 {
			deviceID := strings.ToLower(parts[4])
			for _, deviceType := range t.wrapper.config.SupportedDevices {
				if strings.Contains(deviceID, deviceType) {
					score += 25
					break
				}
			}
		}
	}

	return score
}

// SupportsDeviceType 檢查是否支援指定設備類型
func (t *ExampleTransformer) SupportsDeviceType(deviceType string) bool {
	for _, supported := range t.wrapper.config.SupportedDevices {
		if strings.EqualFold(supported, deviceType) {
			return true
		}
	}
	return false
}

// GetSupportedTopicPatterns 獲取支援的 topic 模式
func (t *ExampleTransformer) GetSupportedTopicPatterns() []string {
	patterns := make([]string, 0)

	// 上行模式
	patterns = append(patterns, fmt.Sprintf("%s/{device_id}/state", t.wrapper.config.TopicPrefix))
	patterns = append(patterns, fmt.Sprintf("%s/{device_id}/telemetry", t.wrapper.config.TopicPrefix))
	patterns = append(patterns, fmt.Sprintf("%s/{device_id}/event", t.wrapper.config.TopicPrefix))

	// 下行模式
	patterns = append(patterns, "rtk/v1/{tenant}/{site}/{device_id}/cmd/req")

	return patterns
}
