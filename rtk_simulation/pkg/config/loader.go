package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

// Loader 配置載入器
type Loader struct {
	viper *viper.Viper
}

// NewLoader 建立新的配置載入器
func NewLoader() *Loader {
	v := viper.New()

	// 設定配置檔案類型
	v.SetConfigType("yaml")

	// 設定環境變數前綴
	v.SetEnvPrefix("RTK_SIM")
	v.AutomaticEnv()

	// 設定預設值
	setDefaultValues(v)

	return &Loader{viper: v}
}

// LoadFromFile 從檔案載入配置
func (l *Loader) LoadFromFile(configPath string) (*SimulationConfig, error) {
	// 檢查檔案是否存在
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("config file not found: %s", configPath)
	}

	// 設定配置檔案路徑
	l.viper.SetConfigFile(configPath)

	// 讀取配置檔案
	if err := l.viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %v", err)
	}

	var config SimulationConfig
	if err := l.viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %v", err)
	}

	// 驗證配置
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %v", err)
	}

	return &config, nil
}

// LoadFromBytes 從位元組載入配置
func (l *Loader) LoadFromBytes(data []byte) (*SimulationConfig, error) {
	var config SimulationConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal YAML: %v", err)
	}

	// 驗證配置
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %v", err)
	}

	return &config, nil
}

// LoadDefault 載入預設配置
func (l *Loader) LoadDefault() *SimulationConfig {
	return GetDefaultConfig()
}

// SaveToFile 儲存配置到檔案
func (l *Loader) SaveToFile(config *SimulationConfig, filePath string) error {
	// 建立目錄（如果不存在）
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %v", err)
	}

	// 序列化配置
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %v", err)
	}

	// 寫入檔案
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %v", err)
	}

	return nil
}

// Watch 監控配置檔案變化
func (l *Loader) Watch(configPath string, callback func(*SimulationConfig)) error {
	l.viper.SetConfigFile(configPath)

	l.viper.WatchConfig()
	l.viper.OnConfigChange(func(e fsnotify.Event) {
		var config SimulationConfig
		if err := l.viper.Unmarshal(&config); err == nil {
			if err := config.Validate(); err == nil {
				callback(&config)
			}
		}
	})

	return nil
}

// Merge 合併多個配置檔案
func (l *Loader) Merge(basePath string, overridePaths ...string) (*SimulationConfig, error) {
	// 載入基礎配置
	config, err := l.LoadFromFile(basePath)
	if err != nil {
		return nil, fmt.Errorf("failed to load base config: %v", err)
	}

	// 逐一合併覆蓋配置
	for _, overridePath := range overridePaths {
		if _, err := os.Stat(overridePath); os.IsNotExist(err) {
			continue // 跳過不存在的檔案
		}

		overrideConfig, err := l.LoadFromFile(overridePath)
		if err != nil {
			return nil, fmt.Errorf("failed to load override config %s: %v", overridePath, err)
		}

		// 合併配置
		if err := mergeConfigs(config, overrideConfig); err != nil {
			return nil, fmt.Errorf("failed to merge config %s: %v", overridePath, err)
		}
	}

	return config, nil
}

// FindConfigFile 尋找配置檔案
func (l *Loader) FindConfigFile(name string) (string, error) {
	searchPaths := []string{
		".",
		"./configs",
		"../configs",
		"/etc/rtk_simulation",
		"$HOME/.rtk_simulation",
	}

	for _, path := range searchPaths {
		// 展開環境變數
		expandedPath := os.ExpandEnv(path)

		configPath := filepath.Join(expandedPath, name)
		if _, err := os.Stat(configPath); err == nil {
			return configPath, nil
		}

		// 嘗試加上 .yaml 副檔名
		configPath = filepath.Join(expandedPath, name+".yaml")
		if _, err := os.Stat(configPath); err == nil {
			return configPath, nil
		}

		// 嘗試加上 .yml 副檔名
		configPath = filepath.Join(expandedPath, name+".yml")
		if _, err := os.Stat(configPath); err == nil {
			return configPath, nil
		}
	}

	return "", fmt.Errorf("config file not found: %s", name)
}

// GenerateTemplate 生成配置範本
func (l *Loader) GenerateTemplate(templateType string) (*SimulationConfig, error) {
	switch templateType {
	case "home_basic":
		return generateHomeBasicTemplate(), nil
	case "home_advanced":
		return generateHomeAdvancedTemplate(), nil
	case "apartment":
		return generateApartmentTemplate(), nil
	case "smart_home":
		return generateSmartHomeTemplate(), nil
	default:
		return nil, fmt.Errorf("unknown template type: %s", templateType)
	}
}

// 內部函數
func setDefaultValues(v *viper.Viper) {
	// 模擬設定預設值
	v.SetDefault("simulation.name", "default_simulation")
	v.SetDefault("simulation.duration", "1h")
	v.SetDefault("simulation.real_time_factor", 1.0)
	v.SetDefault("simulation.max_devices", 100)
	v.SetDefault("simulation.update_interval", "5s")
	v.SetDefault("simulation.metrics_enabled", true)
	v.SetDefault("simulation.metrics_port", 8080)

	// MQTT 設定預設值
	v.SetDefault("mqtt.broker", "localhost")
	v.SetDefault("mqtt.port", 1883)
	v.SetDefault("mqtt.client_prefix", "rtk_sim")
	v.SetDefault("mqtt.keep_alive", "60s")
	v.SetDefault("mqtt.clean_session", true)
	v.SetDefault("mqtt.qos", 1)
	v.SetDefault("mqtt.retained", false)
	v.SetDefault("mqtt.tls_enabled", false)

	// 網路設定預設值
	v.SetDefault("network.topology", "single_router")
	v.SetDefault("network.subnet", "192.168.1.0/24")
	v.SetDefault("network.internet_bandwidth", 100)
	v.SetDefault("network.dhcp_pool.start_ip", "192.168.1.100")
	v.SetDefault("network.dhcp_pool.end_ip", "192.168.1.200")
	v.SetDefault("network.dhcp_pool.lease_time", "24h")
	v.SetDefault("network.dhcp_pool.gateway", "192.168.1.1")
	v.SetDefault("network.dhcp_pool.dns", []string{"8.8.8.8", "8.8.4.4"})

	// 日誌設定預設值
	v.SetDefault("logging.level", "info")
	v.SetDefault("logging.format", "text")
	v.SetDefault("logging.output", "stdout")
	v.SetDefault("logging.device_specific", false)
}

func mergeConfigs(base, override *SimulationConfig) error {
	// 這裡實作配置合併邏輯
	// 可以使用 reflect 包來動態合併，或手動處理重要欄位

	// 模擬設定合併
	if override.Simulation.Name != "" {
		base.Simulation.Name = override.Simulation.Name
	}
	if override.Simulation.Duration > 0 {
		base.Simulation.Duration = override.Simulation.Duration
	}
	if override.Simulation.RealTimeFactor > 0 {
		base.Simulation.RealTimeFactor = override.Simulation.RealTimeFactor
	}

	// MQTT 設定合併
	if override.MQTT.Broker != "" {
		base.MQTT.Broker = override.MQTT.Broker
	}
	if override.MQTT.Port > 0 {
		base.MQTT.Port = override.MQTT.Port
	}

	// 設備配置合併（追加）
	base.Devices.NetworkDevices = append(base.Devices.NetworkDevices, override.Devices.NetworkDevices...)
	base.Devices.IoTDevices = append(base.Devices.IoTDevices, override.Devices.IoTDevices...)
	base.Devices.ClientDevices = append(base.Devices.ClientDevices, override.Devices.ClientDevices...)

	// 情境配置合併（追加）
	base.Scenarios = append(base.Scenarios, override.Scenarios...)

	return nil
}

func generateHomeBasicTemplate() *SimulationConfig {
	config := GetDefaultConfig()
	config.Simulation.Name = "home_basic"

	// 添加基本家庭設備
	config.Devices.NetworkDevices = []NetworkDeviceConfig{
		{
			DeviceConfig: DeviceConfig{
				ID:             "main_router",
				Type:           "router",
				Tenant:         "home",
				Site:           "living_room",
				IPAddress:      "192.168.1.1",
				ConnectionType: "ethernet",
				Firmware:       "1.0.0",
				Protocols:      []string{"wifi", "ethernet", "mqtt"},
			},
			Model:         "Basic Router",
			WiFiBands:     []string{"2.4GHz", "5GHz"},
			EthernetPorts: 4,
			MaxThroughput: 100,
		},
	}

	config.Devices.IoTDevices = []IoTDeviceConfig{
		{
			DeviceConfig: DeviceConfig{
				ID:             "living_room_light",
				Type:           "smart_bulb",
				Tenant:         "home",
				Site:           "living_room",
				ConnectionType: "wifi",
				Firmware:       "1.0.0",
				Protocols:      []string{"wifi", "mqtt"},
			},
			Category:     "lighting",
			PowerSource:  "ac",
			SensorTypes:  []string{"brightness"},
			ControlTypes: []string{"power", "brightness", "color"},
		},
	}

	return config
}

func generateHomeAdvancedTemplate() *SimulationConfig {
	config := generateHomeBasicTemplate()
	config.Simulation.Name = "home_advanced"

	// 添加更多設備...
	return config
}

func generateApartmentTemplate() *SimulationConfig {
	config := GetDefaultConfig()
	config.Simulation.Name = "apartment"
	// 添加公寓專用配置...
	return config
}

func generateSmartHomeTemplate() *SimulationConfig {
	config := GetDefaultConfig()
	config.Simulation.Name = "smart_home"
	// 添加智慧家庭配置...
	return config
}
