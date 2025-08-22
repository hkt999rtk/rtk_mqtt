package example

import (
	"fmt"
	"io/ioutil"

	"gopkg.in/yaml.v3"

	"rtk_wrapper/internal/registry"
	"rtk_wrapper/pkg/types"
)

// RegisterExampleWrapper 註冊範例 wrapper 到註冊表
func RegisterExampleWrapper(reg *registry.Registry, configFile string) error {
	// 創建 wrapper 實例
	wrapper := NewExampleWrapper()

	// 載入配置（如果提供了配置文件）
	if configFile != "" {
		config, err := loadConfig(configFile)
		if err != nil {
			return fmt.Errorf("failed to load example wrapper config: %w", err)
		}

		// 更新 wrapper 配置
		wrapper.config = *config
	}

	// 創建 transformer
	transformer := NewExampleTransformer(wrapper)

	// 註冊 wrapper 和 transformer
	if err := reg.RegisterWrapper(wrapper.Name(), wrapper, transformer); err != nil {
		return fmt.Errorf("failed to register example wrapper: %w", err)
	}

	// 註冊上行路由規則
	uplinkRules := []registry.PayloadRule{
		{
			FieldPath:    "status",
			ExpectedType: "string",
			Required:     false,
		},
		{
			FieldPath:    "temperature",
			ExpectedType: "number",
			Required:     false,
		},
	}

	// 上行路由：example/{device_id}/state
	if err := reg.RegisterUplinkRoute(
		wrapper.Name(),
		100,
		"example/+/state",
		uplinkRules,
		wrapper.SupportedDeviceTypes(),
	); err != nil {
		return fmt.Errorf("failed to register uplink route: %w", err)
	}

	// 上行路由：example/{device_id}/telemetry
	if err := reg.RegisterUplinkRoute(
		wrapper.Name(),
		95,
		"example/+/telemetry",
		uplinkRules,
		wrapper.SupportedDeviceTypes(),
	); err != nil {
		return fmt.Errorf("failed to register uplink telemetry route: %w", err)
	}

	// 上行路由：example/{device_id}/event
	if err := reg.RegisterUplinkRoute(
		wrapper.Name(),
		90,
		"example/+/event",
		uplinkRules,
		wrapper.SupportedDeviceTypes(),
	); err != nil {
		return fmt.Errorf("failed to register uplink event route: %w", err)
	}

	// 註冊下行路由規則
	if err := reg.RegisterDownlinkRoute(
		wrapper.Name(),
		100,
		"rtk/v1/+/+/+/cmd/req",
		wrapper.SupportedDeviceTypes(),
	); err != nil {
		return fmt.Errorf("failed to register downlink route: %w", err)
	}

	return nil
}

// loadConfig 載入範例 wrapper 配置
func loadConfig(configFile string) (*ExampleConfig, error) {
	data, err := ioutil.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config ExampleConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config YAML: %w", err)
	}

	return &config, nil
}

// CreateExampleWrapper 創建並配置範例 wrapper（便利函數）
func CreateExampleWrapper(configFile string) (types.DeviceWrapper, types.MessageTransformer, error) {
	wrapper := NewExampleWrapper()

	if configFile != "" {
		config, err := loadConfig(configFile)
		if err != nil {
			return nil, nil, err
		}
		wrapper.config = *config
	}

	transformer := NewExampleTransformer(wrapper)

	return wrapper, transformer, nil
}
