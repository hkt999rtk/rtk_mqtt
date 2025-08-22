package homeassistant

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"rtk_wrapper/pkg/types"
)

// HomeAssistantWrapper Home Assistant 設備 wrapper 實現
type HomeAssistantWrapper struct {
	name    string
	version string
	config  HAConfig
	enabled bool
}

// HAConfig Home Assistant wrapper 配置
type HAConfig struct {
	Name                 string                 `yaml:"name"`
	Description          string                 `yaml:"description"`
	SupportedDeviceTypes []string               `yaml:"supported_device_types"`
	TopicPatterns        HATopicPatterns        `yaml:"topic_patterns"`
	PayloadRules         HAPayloadRules         `yaml:"payload_rules"`
	DeviceMapping        map[string]string      `yaml:"device_mapping"`
	ErrorHandling        HAErrorHandling        `yaml:"error_handling"`
	Settings             map[string]interface{} `yaml:"settings"`
}

// HATopicPatterns Home Assistant topic 模式配置
type HATopicPatterns struct {
	Uplink   []HATopicPattern `yaml:"uplink"`
	Downlink []HATopicPattern `yaml:"downlink"`
}

// HATopicPattern topic 模式定義
type HATopicPattern struct {
	Pattern         string `yaml:"pattern"`
	Priority        int    `yaml:"priority"`
	DeviceIDExtract string `yaml:"device_id_extract"`
	MessageType     string `yaml:"message_type"`
}

// HAPayloadRules payload 處理規則
type HAPayloadRules struct {
	StateTransform      HAStateTransform      `yaml:"state_transform"`
	AttributeExtraction HAAttributeExtraction `yaml:"attribute_extraction"`
}

// HAStateTransform 狀態轉換規則
type HAStateTransform struct {
	BooleanMapping  map[string]bool   `yaml:"boolean_mapping"`
	UnitConversions HAUnitConversions `yaml:"unit_conversions"`
}

// HAUnitConversions 單位轉換配置
type HAUnitConversions struct {
	Temperature HATemperatureConversion `yaml:"temperature"`
	Brightness  HABrightnessConversion  `yaml:"brightness"`
}

// HATemperatureConversion 溫度轉換配置
type HATemperatureConversion struct {
	FromUnit string `yaml:"from_unit"`
	ToUnit   string `yaml:"to_unit"`
	Formula  string `yaml:"formula"`
}

// HABrightnessConversion 亮度轉換配置
type HABrightnessConversion struct {
	FromRange []int `yaml:"from_range"`
	ToRange   []int `yaml:"to_range"`
}

// HAAttributeExtraction 屬性提取配置
type HAAttributeExtraction struct {
	Brightness       string              `yaml:"brightness"`
	ColorTemp        string              `yaml:"color_temp"`
	RGBColor         string              `yaml:"rgb_color"`
	PowerConsumption HAComputedAttribute `yaml:"power_consumption"`
}

// HAComputedAttribute 計算屬性配置
type HAComputedAttribute struct {
	Formula string   `yaml:"formula"`
	Sources []string `yaml:"sources"`
}

// HAErrorHandling 錯誤處理配置
type HAErrorHandling struct {
	InvalidJSON          HAErrorAction `yaml:"invalid_json"`
	MissingFields        HAErrorAction `yaml:"missing_fields"`
	TransformationErrors HAErrorAction `yaml:"transformation_errors"`
}

// HAErrorAction 錯誤處理動作配置
type HAErrorAction struct {
	Action     string                 `yaml:"action"`
	MaxRetries int                    `yaml:"max_retries"`
	Defaults   map[string]interface{} `yaml:"defaults"`
}

// NewHomeAssistantWrapper 創建 Home Assistant wrapper
func NewHomeAssistantWrapper() *HomeAssistantWrapper {
	return &HomeAssistantWrapper{
		name:    "homeassistant",
		version: "1.0.0",
		enabled: true,
		config: HAConfig{
			Name:        "Home Assistant Wrapper",
			Description: "Converts Home Assistant MQTT messages to RTK format",
			SupportedDeviceTypes: []string{
				"light", "switch", "sensor", "climate", "cover", "binary_sensor", "fan", "lock",
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
					UnitConversions: HAUnitConversions{
						Temperature: HATemperatureConversion{
							FromUnit: "°F",
							ToUnit:   "°C",
							Formula:  "(x - 32) * 5/9",
						},
						Brightness: HABrightnessConversion{
							FromRange: []int{0, 255},
							ToRange:   []int{0, 100},
						},
					},
				},
				AttributeExtraction: HAAttributeExtraction{
					Brightness: "attributes.brightness",
					ColorTemp:  "attributes.color_temp",
					RGBColor:   "attributes.rgb_color",
					PowerConsumption: HAComputedAttribute{
						Formula: "voltage * current",
						Sources: []string{"voltage", "current"},
					},
				},
			},
			DeviceMapping: make(map[string]string),
			ErrorHandling: HAErrorHandling{
				InvalidJSON: HAErrorAction{
					Action:     "log_and_drop",
					MaxRetries: 3,
				},
				MissingFields: HAErrorAction{
					Action: "fill_defaults",
					Defaults: map[string]interface{}{
						"device_id": "unknown_ha_device",
						"timestamp": "current_time",
					},
				},
				TransformationErrors: HAErrorAction{
					Action: "log_and_forward_original",
				},
			},
		},
	}
}

// Name 返回 wrapper 名稱
func (w *HomeAssistantWrapper) Name() string {
	return w.name
}

// Version 返回 wrapper 版本
func (w *HomeAssistantWrapper) Version() string {
	return w.version
}

// SupportedDeviceTypes 返回支援的設備類型
func (w *HomeAssistantWrapper) SupportedDeviceTypes() []string {
	return w.config.SupportedDeviceTypes
}

// Transform 統一轉換入口
func (w *HomeAssistantWrapper) Transform(wrapperMsg *types.WrapperMessage) (interface{}, error) {
	if wrapperMsg.Direction == types.DirectionUplink {
		return w.transformUplink(wrapperMsg)
	} else {
		return w.transformDownlink(wrapperMsg)
	}
}

// transformUplink 上行轉換（HA 格式 → RTK 格式）
func (w *HomeAssistantWrapper) transformUplink(wrapperMsg *types.WrapperMessage) (*types.RTKMessage, error) {
	log.Printf("HomeAssistantWrapper: Processing uplink message from topic: %s", wrapperMsg.MQTTInfo.Topic)

	// 從 topic 提取設備資訊
	deviceInfo, err := w.extractDeviceInfo(wrapperMsg.MQTTInfo.Topic)
	if err != nil {
		return nil, fmt.Errorf("failed to extract device info: %w", err)
	}

	// 轉換 payload
	rtkPayload, err := w.convertHAToRTKPayload(deviceInfo, wrapperMsg.ParsedPayload)
	if err != nil {
		return nil, fmt.Errorf("failed to convert HA payload: %w", err)
	}

	// 建構 RTK 格式訊息
	schema := fmt.Sprintf("%s/1.0", deviceInfo.MessageType)
	rtkMsg := types.NewRTKMessage(schema, deviceInfo.DeviceID, rtkPayload)

	// 添加追蹤資訊
	rtkMsg.Trace = &types.TraceInfo{
		RequestID: fmt.Sprintf("ha_%d", time.Now().UnixNano()),
	}

	log.Printf("HomeAssistantWrapper: Converted %s → RTK format (device: %s, type: %s)",
		wrapperMsg.MQTTInfo.Topic, deviceInfo.DeviceID, deviceInfo.DeviceClass)

	return rtkMsg, nil
}

// transformDownlink 下行轉換（RTK 格式 → HA 格式）
func (w *HomeAssistantWrapper) transformDownlink(wrapperMsg *types.WrapperMessage) (*types.DeviceMessage, error) {
	log.Printf("HomeAssistantWrapper: Processing downlink message from topic: %s", wrapperMsg.MQTTInfo.Topic)

	// 解析 RTK 訊息
	rtkInfo, err := w.extractRTKInfo(wrapperMsg.MQTTInfo.Topic)
	if err != nil {
		return nil, fmt.Errorf("failed to extract RTK info: %w", err)
	}

	// 轉換為 HA 格式的 payload
	haPayload, err := w.convertRTKToHAPayload(rtkInfo, wrapperMsg.ParsedPayload)
	if err != nil {
		return nil, fmt.Errorf("failed to convert to HA payload: %w", err)
	}

	// 推斷 HA 設備類別和構建目標 topic
	deviceClass := w.inferHADeviceClass(rtkInfo.DeviceID)
	haTopic := w.buildHATopic(deviceClass, rtkInfo.DeviceID)

	// 建構設備訊息
	deviceMsg := types.NewDeviceMessage(haTopic, haPayload, 0, false)

	log.Printf("HomeAssistantWrapper: Converted RTK → %s (device: %s)", haTopic, rtkInfo.DeviceID)

	return deviceMsg, nil
}

// HADeviceInfo Home Assistant 設備資訊
type HADeviceInfo struct {
	DeviceClass string
	DeviceID    string
	Location    string
	MessageType string
}

// RTKInfo RTK 訊息資訊
type RTKInfo struct {
	Tenant      string
	Site        string
	DeviceID    string
	MessageType string
}

// extractDeviceInfo 從 HA topic 提取設備資訊
func (w *HomeAssistantWrapper) extractDeviceInfo(topic string) (*HADeviceInfo, error) {
	parts := strings.Split(topic, "/")

	// homeassistant/{device_class}/{device_name}/state
	if len(parts) >= 4 && parts[0] == "homeassistant" {
		deviceInfo := &HADeviceInfo{
			DeviceClass: parts[1],
			MessageType: parts[len(parts)-1], // 最後一部分是訊息類型
		}

		if len(parts) == 4 {
			// homeassistant/{device_class}/{device_name}/state
			deviceInfo.DeviceID = parts[2]
		} else if len(parts) == 5 {
			// homeassistant/{device_class}/{location}/{device_name}/state
			deviceInfo.Location = parts[2]
			deviceInfo.DeviceID = fmt.Sprintf("%s_%s", parts[2], parts[3])
		} else {
			return nil, fmt.Errorf("unsupported HA topic format: %s", topic)
		}

		// 檢查是否有映射配置
		if mappedID, exists := w.config.DeviceMapping[deviceInfo.DeviceID]; exists {
			deviceInfo.DeviceID = mappedID
		}

		return deviceInfo, nil
	}

	return nil, fmt.Errorf("not a valid Home Assistant topic: %s", topic)
}

// extractRTKInfo 從 RTK topic 提取資訊
func (w *HomeAssistantWrapper) extractRTKInfo(topic string) (*RTKInfo, error) {
	// rtk/v1/{tenant}/{site}/{device_id}/{message_type}
	parts := strings.Split(topic, "/")
	if len(parts) < 6 || parts[0] != "rtk" || parts[1] != "v1" {
		return nil, fmt.Errorf("invalid RTK topic format: %s", topic)
	}

	return &RTKInfo{
		Tenant:      parts[2],
		Site:        parts[3],
		DeviceID:    parts[4],
		MessageType: strings.Join(parts[5:], "/"),
	}, nil
}

// convertHAToRTKPayload 轉換 HA payload 為 RTK 格式
func (w *HomeAssistantWrapper) convertHAToRTKPayload(deviceInfo *HADeviceInfo, payload *types.FlexiblePayload) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	// 根據設備類型處理不同的欄位
	switch deviceInfo.DeviceClass {
	case "light":
		w.convertLightPayload(result, payload)
	case "switch":
		w.convertSwitchPayload(result, payload)
	case "sensor":
		w.convertSensorPayload(result, payload)
	case "climate":
		w.convertClimatePayload(result, payload)
	case "cover":
		w.convertCoverPayload(result, payload)
	case "binary_sensor":
		w.convertBinarySensorPayload(result, payload)
	default:
		// 通用處理
		w.convertGenericPayload(result, payload)
	}

	// 確保有基本的健康狀態
	if _, exists := result["health"]; !exists {
		result["health"] = "ok"
	}

	return result, nil
}

// convertLightPayload 轉換燈光設備 payload
func (w *HomeAssistantWrapper) convertLightPayload(result map[string]interface{}, payload *types.FlexiblePayload) {
	// 電源狀態
	if state, exists := payload.GetString("state"); exists {
		result["power_state"] = w.normalizeState(state)
	}

	// 亮度
	if brightness, exists := payload.GetFloat64("brightness"); exists {
		// 從 HA 的 0-255 轉換為 RTK 的 0-100
		normalizedBrightness := (brightness / 255.0) * 100.0
		result["brightness"] = normalizedBrightness
	}

	// 色溫
	if colorTemp, exists := payload.GetFloat64("color_temp"); exists {
		result["color_temperature"] = colorTemp
	}

	// RGB 顏色
	if rgbColor, exists := payload.GetNested("rgb_color"); exists {
		result["rgb_color"] = rgbColor
	}

	// 功率
	if power, exists := payload.GetFloat64("power"); exists {
		result["power_consumption"] = power
	}
}

// convertSwitchPayload 轉換開關設備 payload
func (w *HomeAssistantWrapper) convertSwitchPayload(result map[string]interface{}, payload *types.FlexiblePayload) {
	// 電源狀態
	if state, exists := payload.GetString("state"); exists {
		result["power_state"] = w.normalizeState(state)
	}

	// 能源監控
	if energy, exists := payload.GetFloat64("energy"); exists {
		result["energy_consumption"] = energy
	}

	if power, exists := payload.GetFloat64("power"); exists {
		result["power_consumption"] = power
	}

	if voltage, exists := payload.GetFloat64("voltage"); exists {
		result["voltage"] = voltage
	}

	if current, exists := payload.GetFloat64("current"); exists {
		result["current"] = current
	}
}

// convertSensorPayload 轉換感測器 payload
func (w *HomeAssistantWrapper) convertSensorPayload(result map[string]interface{}, payload *types.FlexiblePayload) {
	// 溫度
	if temp, exists := payload.GetFloat64("temperature"); exists {
		result["temperature"] = temp
	}

	// 濕度
	if humidity, exists := payload.GetFloat64("humidity"); exists {
		result["humidity"] = humidity
	}

	// 氣壓
	if pressure, exists := payload.GetFloat64("pressure"); exists {
		result["pressure"] = pressure
	}

	// 照度
	if illuminance, exists := payload.GetFloat64("illuminance"); exists {
		result["illuminance"] = illuminance
	}

	// 電池電量
	if battery, exists := payload.GetFloat64("battery"); exists {
		result["battery_level"] = battery
	}

	// 通用數值
	if value, exists := payload.GetFloat64("value"); exists {
		result["sensor_value"] = value
	}

	// 狀態
	if state, exists := payload.GetString("state"); exists {
		result["sensor_state"] = state
	}
}

// convertClimatePayload 轉換空調設備 payload
func (w *HomeAssistantWrapper) convertClimatePayload(result map[string]interface{}, payload *types.FlexiblePayload) {
	// 當前溫度
	if currentTemp, exists := payload.GetFloat64("current_temperature"); exists {
		result["current_temperature"] = currentTemp
	}

	// 目標溫度
	if targetTemp, exists := payload.GetFloat64("temperature"); exists {
		result["target_temperature"] = targetTemp
	}

	// 運行模式
	if mode, exists := payload.GetString("mode"); exists {
		result["hvac_mode"] = mode
	}

	// 風扇模式
	if fanMode, exists := payload.GetString("fan_mode"); exists {
		result["fan_mode"] = fanMode
	}

	// 電源狀態
	if state, exists := payload.GetString("state"); exists {
		result["power_state"] = w.normalizeState(state)
	}
}

// convertCoverPayload 轉換窗簾/捲簾設備 payload
func (w *HomeAssistantWrapper) convertCoverPayload(result map[string]interface{}, payload *types.FlexiblePayload) {
	// 狀態
	if state, exists := payload.GetString("state"); exists {
		result["cover_state"] = state
	}

	// 位置
	if position, exists := payload.GetFloat64("position"); exists {
		result["position"] = position
	}

	// 傾斜角度
	if tiltPosition, exists := payload.GetFloat64("tilt_position"); exists {
		result["tilt_position"] = tiltPosition
	}
}

// convertBinarySensorPayload 轉換二元感測器 payload
func (w *HomeAssistantWrapper) convertBinarySensorPayload(result map[string]interface{}, payload *types.FlexiblePayload) {
	// 狀態
	if state, exists := payload.GetString("state"); exists {
		result["binary_state"] = w.normalizeState(state)
	}

	// 設備類型
	if deviceClass, exists := payload.GetString("device_class"); exists {
		result["sensor_type"] = deviceClass
	}
}

// convertGenericPayload 轉換通用 payload
func (w *HomeAssistantWrapper) convertGenericPayload(result map[string]interface{}, payload *types.FlexiblePayload) {
	// 直接複製所有欄位
	for key, value := range payload.Parsed {
		result[key] = value
	}
}

// convertRTKToHAPayload 轉換 RTK payload 為 HA 格式
func (w *HomeAssistantWrapper) convertRTKToHAPayload(rtkInfo *RTKInfo, payload *types.FlexiblePayload) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	// 從 RTK 命令格式解析
	if command, exists := payload.GetString("command"); exists {
		w.processRTKCommand(result, command, payload)
	} else {
		// 直接轉換 payload 內容
		if payloadData, exists := payload.GetNested("payload"); exists {
			if payloadMap, ok := payloadData.(map[string]interface{}); ok {
				for key, value := range payloadMap {
					result[key] = value
				}
			}
		}
	}

	return result, nil
}

// processRTKCommand 處理 RTK 命令
func (w *HomeAssistantWrapper) processRTKCommand(result map[string]interface{}, command string, payload *types.FlexiblePayload) {
	switch command {
	case "turn_on":
		result["state"] = "ON"
		// 如果有亮度參數
		if brightness, exists := payload.GetFloat64("brightness"); exists {
			// 從 RTK 的 0-100 轉換為 HA 的 0-255
			haBrightness := (brightness / 100.0) * 255.0
			result["brightness"] = haBrightness
		}

	case "turn_off":
		result["state"] = "OFF"

	case "set_brightness":
		if brightness, exists := payload.GetFloat64("brightness"); exists {
			haBrightness := (brightness / 100.0) * 255.0
			result["brightness"] = haBrightness
		}

	case "set_temperature":
		if temp, exists := payload.GetFloat64("temperature"); exists {
			result["temperature"] = temp
		}

	case "set_position":
		if position, exists := payload.GetFloat64("position"); exists {
			result["position"] = position
		}

	default:
		// 未知命令，嘗試直接轉換 parameters
		if params, exists := payload.GetNested("parameters"); exists {
			if paramsMap, ok := params.(map[string]interface{}); ok {
				for key, value := range paramsMap {
					result[key] = value
				}
			}
		}
	}
}

// inferHADeviceClass 從設備 ID 推斷 HA 設備類別
func (w *HomeAssistantWrapper) inferHADeviceClass(deviceID string) string {
	deviceID = strings.ToLower(deviceID)

	if strings.Contains(deviceID, "light") || strings.Contains(deviceID, "lamp") {
		return "light"
	} else if strings.Contains(deviceID, "switch") || strings.Contains(deviceID, "outlet") {
		return "switch"
	} else if strings.Contains(deviceID, "sensor") {
		return "sensor"
	} else if strings.Contains(deviceID, "climate") || strings.Contains(deviceID, "thermostat") {
		return "climate"
	} else if strings.Contains(deviceID, "cover") || strings.Contains(deviceID, "blind") {
		return "cover"
	} else if strings.Contains(deviceID, "fan") {
		return "fan"
	}

	return "switch" // 預設為開關
}

// buildHATopic 構建 HA topic
func (w *HomeAssistantWrapper) buildHATopic(deviceClass, deviceID string) string {
	// 簡化處理：homeassistant/{device_class}/{device_id}/set
	return fmt.Sprintf("homeassistant/%s/%s/set", deviceClass, deviceID)
}

// normalizeState 正規化狀態值
func (w *HomeAssistantWrapper) normalizeState(state string) string {
	state = strings.ToLower(state)
	switch state {
	case "on", "true", "1", "open", "active":
		return "on"
	case "off", "false", "0", "closed", "inactive":
		return "off"
	default:
		return state
	}
}

// Initialize 初始化 wrapper
func (w *HomeAssistantWrapper) Initialize(config types.WrapperConfig) error {
	w.enabled = config.Enabled
	log.Printf("HomeAssistantWrapper: Initialized (enabled: %t)", w.enabled)
	return nil
}

// Start 啟動 wrapper
func (w *HomeAssistantWrapper) Start(ctx context.Context) error {
	if !w.enabled {
		return fmt.Errorf("wrapper is disabled")
	}

	log.Printf("HomeAssistantWrapper: Started")

	go func() {
		<-ctx.Done()
		log.Printf("HomeAssistantWrapper: Context cancelled, stopping...")
	}()

	return nil
}

// Stop 停止 wrapper
func (w *HomeAssistantWrapper) Stop() error {
	log.Printf("HomeAssistantWrapper: Stopped")
	return nil
}

// HealthCheck 健康檢查
func (w *HomeAssistantWrapper) HealthCheck() error {
	if !w.enabled {
		return fmt.Errorf("wrapper is disabled")
	}
	return nil
}
