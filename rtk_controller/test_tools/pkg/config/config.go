package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
	"rtk_test_tools/pkg/types"
)

// LoadTestConfig 從檔案載入測試配置
func LoadTestConfig(filename string) (*types.TestConfig, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %v", filename, err)
	}
	
	var config types.TestConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file %s: %v", filename, err)
	}
	
	// 設置預設值
	if err := setDefaults(&config); err != nil {
		return nil, fmt.Errorf("failed to set defaults: %v", err)
	}
	
	// 驗證配置
	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("config validation failed: %v", err)
	}
	
	return &config, nil
}

// setDefaults 設置預設值
func setDefaults(config *types.TestConfig) error {
	// MQTT 預設值
	if config.MQTT.Port == 0 {
		config.MQTT.Port = 1883
	}
	
	if config.MQTT.ClientIDPrefix == "" {
		config.MQTT.ClientIDPrefix = "rtk_test"
	}
	
	// 測試設定預設值
	if config.Test.DurationS == 0 {
		config.Test.DurationS = 300 // 5分鐘
	}
	
	if config.Test.LogLevel == "" {
		config.Test.LogLevel = "info"
	}
	
	// 設備預設值
	for i := range config.Devices {
		device := &config.Devices[i]
		
		if device.Tenant == "" {
			device.Tenant = "test"
		}
		
		if device.Site == "" {
			device.Site = "lab"
		}
		
		// 間隔預設值
		if device.Intervals.StateS == 0 {
			device.Intervals.StateS = 30
		}
		
		if device.Intervals.TelemetryS == 0 {
			device.Intervals.TelemetryS = 10
		}
		
		if device.Intervals.EventS == 0 {
			device.Intervals.EventS = 20
		}
		
		// 設備屬性預設值
		if device.Properties == nil {
			device.Properties = make(map[string]interface{})
		}
		
		// 根據設備類型設置預設屬性
		switch device.Type {
		case "gateway":
			setGatewayDefaults(device)
		case "iot_sensor":
			setIoTSensorDefaults(device)
		case "nic":
			setNICDefaults(device)
		case "switch":
			setSwitchDefaults(device)
		}
	}
	
	return nil
}

// setGatewayDefaults 設置 Gateway 預設屬性
func setGatewayDefaults(device *types.DeviceConfig) {
	if _, ok := device.Properties["ip"]; !ok {
		device.Properties["ip"] = "192.168.1.1"
	}
	
	if _, ok := device.Properties["gateway"]; !ok {
		device.Properties["gateway"] = "10.0.1.1"
	}
	
	if _, ok := device.Properties["firmware"]; !ok {
		device.Properties["firmware"] = "1.2.3"
	}
}

// setIoTSensorDefaults 設置 IoT 感測器預設屬性
func setIoTSensorDefaults(device *types.DeviceConfig) {
	if _, ok := device.Properties["ip"]; !ok {
		device.Properties["ip"] = "10.0.1.105"
	}
	
	if _, ok := device.Properties["firmware"]; !ok {
		device.Properties["firmware"] = "1.0.2"
	}
}

// setNICDefaults 設置 NIC 預設屬性
func setNICDefaults(device *types.DeviceConfig) {
	if _, ok := device.Properties["ip"]; !ok {
		device.Properties["ip"] = "10.0.1.100"
	}
	
	if _, ok := device.Properties["netmask"]; !ok {
		device.Properties["netmask"] = "255.255.255.0"
	}
	
	if _, ok := device.Properties["gateway"]; !ok {
		device.Properties["gateway"] = "10.0.1.1"
	}
	
	if _, ok := device.Properties["interface_name"]; !ok {
		device.Properties["interface_name"] = "eth0"
	}
	
	if _, ok := device.Properties["firmware"]; !ok {
		device.Properties["firmware"] = "2.3.1"
	}
}

// setSwitchDefaults 設置 Switch 預設屬性
func setSwitchDefaults(device *types.DeviceConfig) {
	if _, ok := device.Properties["ip"]; !ok {
		device.Properties["ip"] = "10.0.1.10"
	}
	
	if _, ok := device.Properties["gateway"]; !ok {
		device.Properties["gateway"] = "10.0.1.1"
	}
	
	if _, ok := device.Properties["port_count"]; !ok {
		device.Properties["port_count"] = 24
	}
	
	if _, ok := device.Properties["firmware"]; !ok {
		device.Properties["firmware"] = "3.1.4"
	}
}

// validateConfig 驗證配置
func validateConfig(config *types.TestConfig) error {
	// 驗證 MQTT 配置
	if config.MQTT.Broker == "" {
		return fmt.Errorf("MQTT broker cannot be empty")
	}
	
	if config.MQTT.Port <= 0 || config.MQTT.Port > 65535 {
		return fmt.Errorf("MQTT port must be between 1 and 65535")
	}
	
	// 驗證測試設定
	if config.Test.DurationS <= 0 {
		return fmt.Errorf("test duration must be positive")
	}
	
	// 驗證設備配置
	if len(config.Devices) == 0 {
		return fmt.Errorf("at least one device must be configured")
	}
	
	deviceIDs := make(map[string]bool)
	for _, device := range config.Devices {
		// 檢查重複 ID
		if deviceIDs[device.ID] {
			return fmt.Errorf("duplicate device ID: %s", device.ID)
		}
		deviceIDs[device.ID] = true
		
		// 驗證個別設備
		if err := validateDeviceConfig(device); err != nil {
			return fmt.Errorf("device %s: %v", device.ID, err)
		}
	}
	
	return nil
}

// validateDeviceConfig 驗證單一設備配置
func validateDeviceConfig(device types.DeviceConfig) error {
	if device.ID == "" {
		return fmt.Errorf("device ID cannot be empty")
	}
	
	if device.Type == "" {
		return fmt.Errorf("device type cannot be empty")
	}
	
	validTypes := []string{"gateway", "iot_sensor", "nic", "switch"}
	typeValid := false
	for _, validType := range validTypes {
		if device.Type == validType {
			typeValid = true
			break
		}
	}
	
	if !typeValid {
		return fmt.Errorf("invalid device type: %s, must be one of %v", device.Type, validTypes)
	}
	
	if device.Tenant == "" {
		return fmt.Errorf("tenant cannot be empty")
	}
	
	if device.Site == "" {
		return fmt.Errorf("site cannot be empty")
	}
	
	if device.Intervals.StateS <= 0 {
		return fmt.Errorf("state interval must be positive")
	}
	
	if device.Intervals.TelemetryS <= 0 {
		return fmt.Errorf("telemetry interval must be positive")
	}
	
	if device.Intervals.EventS <= 0 {
		return fmt.Errorf("event interval must be positive")
	}
	
	return nil
}

// SaveExampleConfig 儲存範例配置檔案
func SaveExampleConfig(filename string) error {
	config := &types.TestConfig{
		MQTT: types.MQTTConfig{
			Broker:         "localhost",
			Port:           1883,
			Username:       "",
			Password:       "",
			ClientIDPrefix: "rtk_test",
		},
		Test: types.TestSettings{
			DurationS: 600,
			Verbose:   true,
			LogLevel:  "info",
		},
		Devices: []types.DeviceConfig{
			{
				ID:      "gateway01",
				Type:    "gateway",
				Tenant:  "tenant1",
				Site:    "site1",
				Enabled: true,
				Intervals: types.IntervalConfig{
					StateS:     30,
					TelemetryS: 10,
					EventS:     20,
				},
				Properties: map[string]interface{}{
					"ip":       "192.168.1.1",
					"gateway":  "10.0.1.1",
					"firmware": "1.2.3",
				},
			},
			{
				ID:      "iot_sensor_01",
				Type:    "iot_sensor",
				Tenant:  "tenant1",
				Site:    "site1",
				Enabled: true,
				Intervals: types.IntervalConfig{
					StateS:     30,
					TelemetryS: 10,
					EventS:     20,
				},
				Properties: map[string]interface{}{
					"ip":       "10.0.1.105",
					"firmware": "1.0.2",
				},
			},
			{
				ID:      "nic01",
				Type:    "nic",
				Tenant:  "tenant1",
				Site:    "site1",
				Enabled: true,
				Intervals: types.IntervalConfig{
					StateS:     30,
					TelemetryS: 10,
					EventS:     20,
				},
				Properties: map[string]interface{}{
					"ip":             "10.0.1.100",
					"netmask":        "255.255.255.0",
					"gateway":        "10.0.1.1",
					"interface_name": "eth0",
					"firmware":       "2.3.1",
				},
			},
			{
				ID:      "switch01",
				Type:    "switch",
				Tenant:  "tenant1",
				Site:    "site1",
				Enabled: true,
				Intervals: types.IntervalConfig{
					StateS:     30,
					TelemetryS: 10,
					EventS:     20,
				},
				Properties: map[string]interface{}{
					"ip":         "10.0.1.10",
					"gateway":    "10.0.1.1",
					"port_count": 24,
					"firmware":   "3.1.4",
				},
			},
		},
	}
	
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %v", err)
	}
	
	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %v", err)
	}
	
	return nil
}