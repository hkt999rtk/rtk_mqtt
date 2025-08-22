package simulator

import (
	"fmt"
	"rtk_test_tools/pkg/types"
)

// CreateSimulator 根據配置創建設備模擬器
func CreateSimulator(config types.DeviceConfig, mqttConfig types.MQTTConfig, verbose bool) (DeviceSimulator, error) {
	if !config.Enabled {
		return nil, fmt.Errorf("device %s is disabled", config.ID)
	}
	
	switch config.Type {
	case "gateway":
		return NewGatewaySimulator(config, mqttConfig, verbose)
	case "iot_sensor":
		return NewIoTSensorSimulator(config, mqttConfig, verbose)
	case "nic":
		return NewNICSimulator(config, mqttConfig, verbose)
	case "switch":
		return NewSwitchSimulator(config, mqttConfig, verbose)
	default:
		return nil, fmt.Errorf("unsupported device type: %s", config.Type)
	}
}

// ValidateDeviceConfig 驗證設備配置
func ValidateDeviceConfig(config types.DeviceConfig) error {
	if config.ID == "" {
		return fmt.Errorf("device ID cannot be empty")
	}
	
	if config.Type == "" {
		return fmt.Errorf("device type cannot be empty for device %s", config.ID)
	}
	
	if config.Tenant == "" {
		return fmt.Errorf("tenant cannot be empty for device %s", config.ID)
	}
	
	if config.Site == "" {
		return fmt.Errorf("site cannot be empty for device %s", config.ID)
	}
	
	// 驗證間隔配置
	if config.Intervals.StateS <= 0 {
		return fmt.Errorf("state interval must be positive for device %s", config.ID)
	}
	
	if config.Intervals.TelemetryS <= 0 {
		return fmt.Errorf("telemetry interval must be positive for device %s", config.ID)
	}
	
	if config.Intervals.EventS <= 0 {
		return fmt.Errorf("event interval must be positive for device %s", config.ID)
	}
	
	// 驗證設備類型
	validTypes := map[string]bool{
		"gateway":    true,
		"iot_sensor": true,
		"nic":        true,
		"switch":     true,
	}
	
	if !validTypes[config.Type] {
		return fmt.Errorf("invalid device type %s for device %s", config.Type, config.ID)
	}
	
	return nil
}