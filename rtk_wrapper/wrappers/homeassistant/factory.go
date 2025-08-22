package homeassistant

import (
	"fmt"
	"io/ioutil"
	"strings"

	"gopkg.in/yaml.v3"

	"rtk_wrapper/internal/registry"
	"rtk_wrapper/pkg/types"
)

// RegisterHomeAssistantWrapper 註冊 Home Assistant wrapper 到註冊表
func RegisterHomeAssistantWrapper(reg *registry.Registry, configFile string) error {
	// 創建 wrapper 實例
	wrapper := NewHomeAssistantWrapper()

	// 載入配置（如果提供了配置文件）
	if configFile != "" {
		config, err := loadHAConfig(configFile)
		if err != nil {
			return fmt.Errorf("failed to load HA wrapper config: %w", err)
		}

		// 更新 wrapper 配置
		wrapper.config = *config
	}

	// 創建 transformer
	transformer := NewHATransformer(wrapper)

	// 註冊 wrapper 和 transformer
	if err := reg.RegisterWrapper(wrapper.Name(), wrapper, transformer); err != nil {
		return fmt.Errorf("failed to register HA wrapper: %w", err)
	}

	// 註冊上行路由規則
	for _, pattern := range wrapper.config.TopicPatterns.Uplink {
		// 根據設備類型創建 payload 規則
		payloadRules := createPayloadRulesForDeviceClass(pattern.DeviceIDExtract)

		// 轉換 topic 模式為 MQTT wildcard 格式
		mqttPattern := convertToMQTTPattern(pattern.Pattern)

		if err := reg.RegisterUplinkRoute(
			wrapper.Name(),
			pattern.Priority,
			mqttPattern,
			payloadRules,
			wrapper.SupportedDeviceTypes(),
		); err != nil {
			return fmt.Errorf("failed to register uplink route %s: %w", pattern.Pattern, err)
		}
	}

	// 註冊下行路由規則
	for _, pattern := range wrapper.config.TopicPatterns.Downlink {
		mqttPattern := convertToMQTTPattern(pattern.Pattern)

		if err := reg.RegisterDownlinkRoute(
			wrapper.Name(),
			pattern.Priority,
			mqttPattern,
			wrapper.SupportedDeviceTypes(),
		); err != nil {
			return fmt.Errorf("failed to register downlink route %s: %w", pattern.Pattern, err)
		}
	}

	return nil
}

// loadHAConfig 載入 Home Assistant wrapper 配置
func loadHAConfig(configFile string) (*HAConfig, error) {
	data, err := ioutil.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config HAConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config YAML: %w", err)
	}

	return &config, nil
}

// convertToMQTTPattern 轉換配置中的模式為 MQTT wildcard 模式
func convertToMQTTPattern(pattern string) string {
	// 將 {variable} 替換為 MQTT wildcard +
	// homeassistant/{device_class}/{device_name}/state -> homeassistant/+/+/state

	// 簡化實現：將所有 {xxx} 替換為 +
	result := pattern

	// 處理常見模式
	replacements := map[string]string{
		"{device_class}": "+",
		"{device_name}":  "+",
		"{device_id}":    "+",
		"{location}":     "+",
		"{tenant}":       "+",
		"{site}":         "+",
		"{message_type}": "+",
	}

	for old, new := range replacements {
		result = strings.ReplaceAll(result, old, new)
	}

	return result
}

// createPayloadRulesForDeviceClass 為設備類型創建 payload 規則
func createPayloadRulesForDeviceClass(deviceClass string) []registry.PayloadRule {
	// 根據設備類型返回相應的驗證規則
	switch deviceClass {
	case "light":
		return []registry.PayloadRule{
			{
				FieldPath:    "state",
				ExpectedType: "string",
				Required:     false,
			},
			{
				FieldPath:    "brightness",
				ExpectedType: "number",
				Required:     false,
			},
		}

	case "switch":
		return []registry.PayloadRule{
			{
				FieldPath:    "state",
				ExpectedType: "string",
				Required:     false,
			},
		}

	case "sensor":
		return []registry.PayloadRule{
			{
				FieldPath:    "state",
				ExpectedType: "string",
				Required:     false,
			},
			{
				FieldPath:    "value",
				ExpectedType: "number",
				Required:     false,
			},
		}

	case "climate":
		return []registry.PayloadRule{
			{
				FieldPath:    "current_temperature",
				ExpectedType: "number",
				Required:     false,
			},
			{
				FieldPath:    "temperature",
				ExpectedType: "number",
				Required:     false,
			},
		}

	default:
		// 通用規則
		return []registry.PayloadRule{
			{
				FieldPath:    "state",
				ExpectedType: "string",
				Required:     false,
			},
		}
	}
}

// CreateHomeAssistantWrapper 創建並配置 Home Assistant wrapper（便利函數）
func CreateHomeAssistantWrapper(configFile string) (types.DeviceWrapper, types.MessageTransformer, error) {
	wrapper := NewHomeAssistantWrapper()

	if configFile != "" {
		config, err := loadHAConfig(configFile)
		if err != nil {
			return nil, nil, err
		}
		wrapper.config = *config
	}

	transformer := NewHATransformer(wrapper)

	return wrapper, transformer, nil
}

// GetDefaultHAConfig 獲取預設的 Home Assistant 配置
func GetDefaultHAConfig() *HAConfig {
	return &HAConfig{
		Name:        "Home Assistant Wrapper",
		Description: "Converts Home Assistant MQTT messages to RTK format",
		SupportedDeviceTypes: []string{
			"light", "switch", "sensor", "climate", "cover",
			"binary_sensor", "fan", "lock",
		},
		TopicPatterns: HATopicPatterns{
			Uplink: []HATopicPattern{
				{
					Pattern:         "homeassistant/{device_class}/{device_name}/state",
					Priority:        100,
					DeviceIDExtract: "{device_name}",
					MessageType:     "state",
				},
				{
					Pattern:         "homeassistant/{device_class}/{device_name}/attributes",
					Priority:        95,
					DeviceIDExtract: "{device_name}",
					MessageType:     "attr",
				},
				{
					Pattern:         "homeassistant/{device_class}/{location}/{device_name}/state",
					Priority:        90,
					DeviceIDExtract: "{location}_{device_name}",
					MessageType:     "state",
				},
			},
			Downlink: []HATopicPattern{
				{
					Pattern:     "rtk/v1/{tenant}/{site}/{device_id}/cmd/req",
					Priority:    100,
					MessageType: "command",
				},
			},
		},
		PayloadRules: HAPayloadRules{
			StateTransform: HAStateTransform{
				BooleanMapping: map[string]bool{
					"on":  true,
					"off": false,
					"ON":  true,
					"OFF": false,
				},
			},
		},
		DeviceMapping: make(map[string]string),
	}
}

// ValidateHAConfig 驗證 Home Assistant 配置
func ValidateHAConfig(config *HAConfig) error {
	if config.Name == "" {
		return fmt.Errorf("wrapper name is required")
	}

	if len(config.SupportedDeviceTypes) == 0 {
		return fmt.Errorf("at least one supported device type is required")
	}

	if len(config.TopicPatterns.Uplink) == 0 {
		return fmt.Errorf("at least one uplink pattern is required")
	}

	// 驗證 topic 模式
	for _, pattern := range config.TopicPatterns.Uplink {
		if pattern.Pattern == "" {
			return fmt.Errorf("uplink pattern cannot be empty")
		}
		if pattern.Priority < 0 || pattern.Priority > 1000 {
			return fmt.Errorf("pattern priority must be between 0 and 1000")
		}
	}

	for _, pattern := range config.TopicPatterns.Downlink {
		if pattern.Pattern == "" {
			return fmt.Errorf("downlink pattern cannot be empty")
		}
		if pattern.Priority < 0 || pattern.Priority > 1000 {
			return fmt.Errorf("pattern priority must be between 0 and 1000")
		}
	}

	return nil
}

// BuildHAConfigFromTemplate 從模板建構 HA 配置
func BuildHAConfigFromTemplate(deviceTypes []string, customPatterns map[string][]HATopicPattern) *HAConfig {
	config := GetDefaultHAConfig()

	if len(deviceTypes) > 0 {
		config.SupportedDeviceTypes = deviceTypes
	}

	// 添加自定義模式
	if uplink, exists := customPatterns["uplink"]; exists {
		config.TopicPatterns.Uplink = append(config.TopicPatterns.Uplink, uplink...)
	}

	if downlink, exists := customPatterns["downlink"]; exists {
		config.TopicPatterns.Downlink = append(config.TopicPatterns.Downlink, downlink...)
	}

	return config
}
