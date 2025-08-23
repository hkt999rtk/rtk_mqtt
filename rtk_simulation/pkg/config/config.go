package config

import (
	"fmt"
	"time"
	// "rtk_simulation/pkg/devices/base"
)

// SimulationConfig 模擬器主配置
type SimulationConfig struct {
	Simulation SimulationSettings `yaml:"simulation" mapstructure:"simulation"`
	MQTT       MQTTSettings       `yaml:"mqtt" mapstructure:"mqtt"`
	Network    NetworkSettings    `yaml:"network" mapstructure:"network"`
	Devices    DeviceConfigs      `yaml:"devices" mapstructure:"devices"`
	Scenarios  []ScenarioConfig   `yaml:"scenarios" mapstructure:"scenarios"`
	Logging    LoggingConfig      `yaml:"logging" mapstructure:"logging"`
}

// SimulationSettings 模擬器設定
type SimulationSettings struct {
	Name           string        `yaml:"name"`
	Duration       time.Duration `yaml:"duration"`
	RealTimeFactor float64       `yaml:"real_time_factor"` // 時間加速倍數
	MaxDevices     int           `yaml:"max_devices"`
	UpdateInterval time.Duration `yaml:"update_interval"`
	MetricsEnabled bool          `yaml:"metrics_enabled"`
	MetricsPort    int           `yaml:"metrics_port"`
}

// MQTTSettings MQTT 設定
type MQTTSettings struct {
	Broker       string        `yaml:"broker"`
	Port         int           `yaml:"port"`
	Username     string        `yaml:"username"`
	Password     string        `yaml:"password"`
	ClientPrefix string        `yaml:"client_prefix"`
	KeepAlive    time.Duration `yaml:"keep_alive"`
	CleanSession bool          `yaml:"clean_session"`
	QoS          byte          `yaml:"qos"`
	Retained     bool          `yaml:"retained"`
	TLSEnabled   bool          `yaml:"tls_enabled"`
}

// NetworkSettings 網路設定
type NetworkSettings struct {
	Topology          string             `yaml:"topology" mapstructure:"topology"` // single_router, mesh_network, hybrid
	Subnet            string             `yaml:"subnet" mapstructure:"subnet"`     // CIDR 格式
	DHCPPool          DHCPPoolConfig     `yaml:"dhcp_pool" mapstructure:"dhcp_pool"`
	InternetBandwidth int                `yaml:"internet_bandwidth" mapstructure:"internet_bandwidth"` // Mbps
	VLANs             []VLANConfig       `yaml:"vlans" mapstructure:"vlans"`
	Interference      InterferenceConfig `yaml:"interference" mapstructure:"interference"`
}

// DHCPPoolConfig DHCP 地址池配置
type DHCPPoolConfig struct {
	StartIP   string        `yaml:"start_ip" mapstructure:"start_ip"`
	EndIP     string        `yaml:"end_ip" mapstructure:"end_ip"`
	LeaseTime time.Duration `yaml:"lease_time" mapstructure:"lease_time"`
	Gateway   string        `yaml:"gateway" mapstructure:"gateway"`
	DNS       []string      `yaml:"dns" mapstructure:"dns"`
}

// VLANConfig VLAN 配置
type VLANConfig struct {
	ID     int    `yaml:"id"`
	Name   string `yaml:"name"`
	Subnet string `yaml:"subnet"`
	Access string `yaml:"access"` // unrestricted, restricted, isolated
}

// InterferenceConfig 干擾配置
type InterferenceConfig struct {
	Enabled         bool                 `yaml:"enabled"`
	Sources         []InterferenceSource `yaml:"sources"`
	EnvironmentType string               `yaml:"environment_type"` // home, office, industrial
	WallAttenuation float64              `yaml:"wall_attenuation"` // dB
}

// InterferenceSource 干擾源
type InterferenceSource struct {
	Type      string  `yaml:"type"`      // microwave, bluetooth, neighbor_wifi
	Frequency string  `yaml:"frequency"` // 2.4GHz, 5GHz
	Power     float64 `yaml:"power"`     // dBm
	Location  Coord   `yaml:"location"`
}

// DeviceConfig 基礎設備配置
type DeviceConfig struct {
	ID             string    `yaml:"id" mapstructure:"id"`
	Type           string    `yaml:"type" mapstructure:"type"`
	Tenant         string    `yaml:"tenant" mapstructure:"tenant"`
	Site           string    `yaml:"site" mapstructure:"site"`
	IPAddress      string    `yaml:"ip_address" mapstructure:"ip_address"`
	ConnectionType string    `yaml:"connection_type" mapstructure:"connection_type"`
	Firmware       string    `yaml:"firmware" mapstructure:"firmware"`
	Protocols      []string  `yaml:"protocols" mapstructure:"protocols"`
	Location       *Location `yaml:"location,omitempty" mapstructure:"location,omitempty"`
}

// Location 設備位置資訊
type Location struct {
	Room        string `yaml:"room,omitempty"`
	Building    string `yaml:"building,omitempty"`
	Floor       string `yaml:"floor,omitempty"`
	Coordinates *Coord `yaml:"coordinates,omitempty"`
}

// Coord 座標資訊
type Coord struct {
	X float64 `yaml:"x"`
	Y float64 `yaml:"y"`
	Z float64 `yaml:"z,omitempty"`
}

// DeviceConfigs 設備配置
type DeviceConfigs struct {
	NetworkDevices []NetworkDeviceConfig `yaml:"network_devices" mapstructure:"network_devices"`
	IoTDevices     []IoTDeviceConfig     `yaml:"iot_devices" mapstructure:"iot_devices"`
	ClientDevices  []ClientDeviceConfig  `yaml:"client_devices" mapstructure:"client_devices"`
}

// NetworkDeviceConfig 網路設備配置
type NetworkDeviceConfig struct {
	DeviceConfig  `yaml:",inline" mapstructure:",squash"`
	Model         string      `yaml:"model"`
	WiFiBands     []string    `yaml:"wifi_bands"`
	EthernetPorts int         `yaml:"ethernet_ports"`
	PoESupport    bool        `yaml:"poe_support"`
	MaxThroughput int         `yaml:"max_throughput"` // Mbps
	WiFiConfig    WiFiConfig  `yaml:"wifi_config"`
	QoSConfig     QoSConfig   `yaml:"qos_config"`
	MeshConfig    *MeshConfig `yaml:"mesh_config,omitempty"`
}

// WiFiConfig WiFi 配置
type WiFiConfig struct {
	Networks []WiFiNetworkConfig `yaml:"networks"`
	Channels WiFiChannelConfig   `yaml:"channels"`
	Security WiFiSecurityConfig  `yaml:"security"`
}

// WiFiNetworkConfig WiFi 網路配置
type WiFiNetworkConfig struct {
	SSID       string `yaml:"ssid"`
	Band       string `yaml:"band"` // 2.4GHz, 5GHz, 6GHz
	Hidden     bool   `yaml:"hidden"`
	MaxClients int    `yaml:"max_clients"`
	TxPower    int    `yaml:"tx_power"` // dBm
}

// WiFiChannelConfig WiFi 頻道配置
type WiFiChannelConfig struct {
	Channel2G  int  `yaml:"channel_2g"`
	Channel5G  int  `yaml:"channel_5g"`
	AutoSelect bool `yaml:"auto_select"`
	Width2G    int  `yaml:"width_2g"` // 20, 40 MHz
	Width5G    int  `yaml:"width_5g"` // 20, 40, 80, 160 MHz
}

// WiFiSecurityConfig WiFi 安全配置
type WiFiSecurityConfig struct {
	Type       string `yaml:"type"` // WPA2, WPA3, WEP, Open
	Password   string `yaml:"password"`
	Encryption string `yaml:"encryption"` // AES, TKIP
}

// QoSConfig QoS 配置
type QoSConfig struct {
	Enabled        bool           `yaml:"enabled"`
	Rules          []QoSRule      `yaml:"rules"`
	BandwidthLimit BandwidthLimit `yaml:"bandwidth_limit"`
}

// QoSRule QoS 規則
type QoSRule struct {
	Name      string `yaml:"name"`
	Protocol  string `yaml:"protocol"` // TCP, UDP, ICMP
	Port      int    `yaml:"port"`
	Priority  int    `yaml:"priority"`  // 1-8
	Bandwidth int    `yaml:"bandwidth"` // Kbps
}

// BandwidthLimit 頻寬限制
type BandwidthLimit struct {
	Upload   int `yaml:"upload"`   // Mbps
	Download int `yaml:"download"` // Mbps
}

// MeshConfig Mesh 網路配置
type MeshConfig struct {
	Enabled      bool     `yaml:"enabled"`
	Role         string   `yaml:"role"` // router, node, satellite
	ParentDevice string   `yaml:"parent_device"`
	BackhaulBand string   `yaml:"backhaul_band"` // 2.4GHz, 5GHz, dedicated
	AutoConnect  bool     `yaml:"auto_connect"`
	MeshNodes    []string `yaml:"mesh_nodes"`
}

// IoTDeviceConfig IoT 設備配置
type IoTDeviceConfig struct {
	DeviceConfig     `yaml:",inline" mapstructure:",squash"`
	Category         string                 `yaml:"category"`          // lighting, climate, security, entertainment
	PowerSource      string                 `yaml:"power_source"`      // battery, ac, poe
	BatteryCapacity  float64                `yaml:"battery_capacity"`  // mAh
	PowerConsumption float64                `yaml:"power_consumption"` // watts
	SensorTypes      []string               `yaml:"sensor_types"`
	ControlTypes     []string               `yaml:"control_types"`
	UsagePatterns    []UsagePatternConfig   `yaml:"usage_patterns"`
	AutomationRules  []AutomationRuleConfig `yaml:"automation_rules"`
}

// UsagePatternConfig 使用模式配置
type UsagePatternConfig struct {
	Name        string          `yaml:"name"`
	TimeRange   TimeRangeConfig `yaml:"time_range"`
	Probability float64         `yaml:"probability"` // 0.0-1.0
	Behavior    BehaviorConfig  `yaml:"behavior"`
}

// TimeRangeConfig 時間範圍配置
type TimeRangeConfig struct {
	StartTime string   `yaml:"start_time"` // HH:MM
	EndTime   string   `yaml:"end_time"`   // HH:MM
	Days      []string `yaml:"days"`       // monday, tuesday, ...
}

// BehaviorConfig 行為配置
type BehaviorConfig struct {
	PowerState  string                 `yaml:"power_state"` // on, off, auto
	Brightness  int                    `yaml:"brightness"`  // 0-100
	Temperature float64                `yaml:"temperature"` // 設定溫度
	Mode        string                 `yaml:"mode"`
	Custom      map[string]interface{} `yaml:"custom"`
}

// AutomationRuleConfig 自動化規則配置
type AutomationRuleConfig struct {
	Name       string            `yaml:"name"`
	Trigger    TriggerConfig     `yaml:"trigger"`
	Conditions []ConditionConfig `yaml:"conditions"`
	Actions    []ActionConfig    `yaml:"actions"`
}

// TriggerConfig 觸發器配置
type TriggerConfig struct {
	Type       string                 `yaml:"type"` // time, sensor, device_state, event
	Parameters map[string]interface{} `yaml:"parameters"`
}

// ConditionConfig 條件配置
type ConditionConfig struct {
	Type       string                 `yaml:"type"` // time, weather, device_state
	Parameters map[string]interface{} `yaml:"parameters"`
}

// ActionConfig 動作配置
type ActionConfig struct {
	DeviceID   string                 `yaml:"device_id"`
	Action     string                 `yaml:"action"`
	Parameters map[string]interface{} `yaml:"parameters"`
}

// ClientDeviceConfig 客戶端設備配置
type ClientDeviceConfig struct {
	DeviceConfig    `yaml:",inline" mapstructure:",squash"`
	DeviceClass     string                `yaml:"device_class"` // smartphone, laptop, tablet, desktop
	OperatingSystem string                `yaml:"operating_system"`
	UsageProfile    UsageProfileConfig    `yaml:"usage_profile"`
	Applications    []ApplicationConfig   `yaml:"applications"`
	NetworkBehavior NetworkBehaviorConfig `yaml:"network_behavior"`
}

// UsageProfileConfig 使用配置文件
type UsageProfileConfig struct {
	Profile     string            `yaml:"profile"`     // light, moderate, heavy, gaming
	DailyHours  float64           `yaml:"daily_hours"` // 每日使用時數
	PeakHours   []TimeRangeConfig `yaml:"peak_hours"`  // 高峰使用時段
	IdleTimeout time.Duration     `yaml:"idle_timeout"`
}

// ApplicationConfig 應用程式配置
type ApplicationConfig struct {
	Name      string  `yaml:"name"`
	Type      string  `yaml:"type"`      // web, streaming, gaming, productivity
	Bandwidth int     `yaml:"bandwidth"` // Kbps
	Frequency float64 `yaml:"frequency"` // 使用頻率 0.0-1.0
	Duration  int     `yaml:"duration"`  // 平均使用時長 (分鐘)
}

// NetworkBehaviorConfig 網路行為配置
type NetworkBehaviorConfig struct {
	PreferredBand  string               `yaml:"preferred_band"` // 2.4GHz, 5GHz, auto
	RoamingEnabled bool                 `yaml:"roaming_enabled"`
	PowerSaveMode  bool                 `yaml:"power_save_mode"`
	ConnectTimeout time.Duration        `yaml:"connect_timeout"`
	RetryAttempts  int                  `yaml:"retry_attempts"`
	BandwidthUsage BandwidthUsageConfig `yaml:"bandwidth_usage"`
}

// BandwidthUsageConfig 頻寬使用配置
type BandwidthUsageConfig struct {
	TypicalUpload   int `yaml:"typical_upload"`   // Kbps
	TypicalDownload int `yaml:"typical_download"` // Kbps
	PeakUpload      int `yaml:"peak_upload"`      // Kbps
	PeakDownload    int `yaml:"peak_download"`    // Kbps
	BurstDuration   int `yaml:"burst_duration"`   // seconds
}

// ScenarioConfig 情境配置
type ScenarioConfig struct {
	Name        string              `yaml:"name"`
	Description string              `yaml:"description"`
	Duration    time.Duration       `yaml:"duration"`
	Events      []ScenarioEvent     `yaml:"events"`
	Conditions  []ScenarioCondition `yaml:"conditions"`
}

// ScenarioEvent 情境事件
type ScenarioEvent struct {
	Time        time.Duration          `yaml:"time"`   // 事件觸發時間
	Type        string                 `yaml:"type"`   // device_failure, network_congestion, power_outage
	Target      string                 `yaml:"target"` // 目標設備或網路
	Parameters  map[string]interface{} `yaml:"parameters"`
	Probability float64                `yaml:"probability"` // 事件發生機率
}

// ScenarioCondition 情境條件
type ScenarioCondition struct {
	Type       string                 `yaml:"type"` // time_of_day, day_of_week, weather
	Parameters map[string]interface{} `yaml:"parameters"`
}

// LoggingConfig 日誌配置
type LoggingConfig struct {
	Level          string `yaml:"level"`  // debug, info, warn, error
	Format         string `yaml:"format"` // text, json
	Output         string `yaml:"output"` // stdout, file
	File           string `yaml:"file,omitempty"`
	MaxSize        int    `yaml:"max_size"` // MB
	MaxAge         int    `yaml:"max_age"`  // days
	MaxBackups     int    `yaml:"max_backups"`
	Compress       bool   `yaml:"compress"`
	DeviceSpecific bool   `yaml:"device_specific"` // 是否為每個設備創建單獨日誌
}

// Validate 驗證配置
func (c *SimulationConfig) Validate() error {
	if c.Simulation.Name == "" {
		return fmt.Errorf("simulation name is required")
	}

	if c.MQTT.Broker == "" {
		return fmt.Errorf("MQTT broker is required")
	}

	if c.MQTT.Port <= 0 {
		return fmt.Errorf("MQTT port must be positive")
	}

	if c.Network.Subnet == "" {
		return fmt.Errorf("network subnet is required")
	}

	// 驗證設備配置
	if err := c.validateDevices(); err != nil {
		return fmt.Errorf("device validation failed: %v", err)
	}

	return nil
}

func (c *SimulationConfig) validateDevices() error {
	deviceIDs := make(map[string]bool)

	// 檢查網路設備
	for _, device := range c.Devices.NetworkDevices {
		if device.ID == "" {
			return fmt.Errorf("network device ID is required")
		}
		if deviceIDs[device.ID] {
			return fmt.Errorf("duplicate device ID: %s", device.ID)
		}
		deviceIDs[device.ID] = true
	}

	// 檢查 IoT 設備
	for _, device := range c.Devices.IoTDevices {
		if device.ID == "" {
			return fmt.Errorf("IoT device ID is required")
		}
		if deviceIDs[device.ID] {
			return fmt.Errorf("duplicate device ID: %s", device.ID)
		}
		deviceIDs[device.ID] = true
	}

	// 檢查客戶端設備
	for _, device := range c.Devices.ClientDevices {
		if device.ID == "" {
			return fmt.Errorf("client device ID is required")
		}
		if deviceIDs[device.ID] {
			return fmt.Errorf("duplicate device ID: %s", device.ID)
		}
		deviceIDs[device.ID] = true
	}

	return nil
}

// GetDefaultConfig 取得預設配置
func GetDefaultConfig() *SimulationConfig {
	return &SimulationConfig{
		Simulation: SimulationSettings{
			Name:           "default_simulation",
			Duration:       1 * time.Hour,
			RealTimeFactor: 1.0,
			MaxDevices:     100,
			UpdateInterval: 5 * time.Second,
			MetricsEnabled: true,
			MetricsPort:    8080,
		},
		MQTT: MQTTSettings{
			Broker:       "localhost",
			Port:         1883,
			ClientPrefix: "rtk_sim",
			KeepAlive:    60 * time.Second,
			CleanSession: true,
			QoS:          1,
			Retained:     false,
			TLSEnabled:   false,
		},
		Network: NetworkSettings{
			Topology:          "single_router",
			Subnet:            "192.168.1.0/24",
			InternetBandwidth: 100,
			DHCPPool: DHCPPoolConfig{
				StartIP:   "192.168.1.100",
				EndIP:     "192.168.1.200",
				LeaseTime: 24 * time.Hour,
				Gateway:   "192.168.1.1",
				DNS:       []string{"8.8.8.8", "8.8.4.4"},
			},
		},
		Logging: LoggingConfig{
			Level:          "info",
			Format:         "text",
			Output:         "stdout",
			DeviceSpecific: false,
		},
	}
}
