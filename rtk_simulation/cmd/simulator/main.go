package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"rtk_simulation/pkg/config"
	"rtk_simulation/pkg/devices"
	"rtk_simulation/pkg/devices/base"
	"rtk_simulation/pkg/interaction"
	"rtk_simulation/pkg/network"
	"rtk_simulation/pkg/sync"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	version = "1.0.0"
	commit  = "dev"
	date    = "unknown"
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "rtk-simulator",
	Short: "RTK Home Network Environment Simulator",
	Long: `RTK Home Network Environment Simulator
	
A comprehensive home network environment simulator that supports various IoT devices,
network equipment, and client devices using the RTK MQTT protocol.`,
	Version: fmt.Sprintf("%s (commit: %s, date: %s)", version, commit, date),
}

var runCmd = &cobra.Command{
	Use:   "run [config-file]",
	Short: "Run the network simulation",
	Long: `Run the network simulation with the specified configuration file.
If no config file is provided, it will look for config files in the following order:
- ./config.yaml
- ./configs/home_basic.yaml
- Use default configuration`,
	Args: cobra.MaximumNArgs(1),
	RunE: runSimulation,
}

var generateCmd = &cobra.Command{
	Use:   "generate [template-type]",
	Short: "Generate configuration templates",
	Long: `Generate configuration templates for different scenarios:
- home_basic: Basic home network with router and few IoT devices
- home_advanced: Advanced home network with mesh and multiple devices
- apartment: Apartment network setup  
- smart_home: Full smart home configuration with automation`,
	Args:      cobra.ExactArgs(1),
	ValidArgs: []string{"home_basic", "home_advanced", "apartment", "smart_home"},
	RunE:      generateConfig,
}

var validateCmd = &cobra.Command{
	Use:   "validate [config-file]",
	Short: "Validate configuration file",
	Long:  "Validate the syntax and content of a configuration file",
	Args:  cobra.ExactArgs(1),
	RunE:  validateConfig,
}

var (
	configFile  string
	outputFile  string
	verbose     bool
	logLevel    string
	dryRun      bool
	metricsPort int
)

func init() {
	// 設定全域 flags
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose logging")
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "info", "Log level (debug, info, warn, error)")

	// run 命令 flags
	runCmd.Flags().StringVarP(&configFile, "config", "c", "", "Configuration file path")
	runCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Validate configuration and exit")
	runCmd.Flags().IntVar(&metricsPort, "metrics-port", 0, "Override metrics port from config")

	// generate 命令 flags
	generateCmd.Flags().StringVarP(&outputFile, "output", "o", "", "Output file path (default: stdout)")

	// 添加子命令
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(generateCmd)
	rootCmd.AddCommand(validateCmd)
}

func runSimulation(cmd *cobra.Command, args []string) error {
	// 設定日誌
	setupLogging()

	// 載入配置
	configPath := findConfigFile(args)
	loader := config.NewLoader()

	cfg, err := loadConfiguration(loader, configPath)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %v", err)
	}

	logrus.WithFields(logrus.Fields{
		"config_file":     configPath,
		"simulation_name": cfg.Simulation.Name,
		"duration":        cfg.Simulation.Duration,
		"max_devices":     cfg.Simulation.MaxDevices,
	}).Info("Starting RTK Home Network Simulator")

	if dryRun {
		logrus.Info("Configuration is valid. Dry-run mode, exiting.")
		return nil
	}

	// 建立模擬器
	simulator, err := NewSimulator(cfg)
	if err != nil {
		return fmt.Errorf("failed to create simulator: %v", err)
	}

	// 設定信號處理
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		logrus.Info("Received shutdown signal, stopping simulation...")
		cancel()
	}()

	// 執行模擬
	return simulator.Run(ctx)
}

func generateConfig(cmd *cobra.Command, args []string) error {
	templateType := args[0]

	// Map template types to config files
	templateFiles := map[string]string{
		"home_basic":    "configs/home_basic.yaml",
		"home_advanced": "configs/home_advanced.yaml",
		"apartment":     "configs/apartment.yaml",
		"smart_home":    "configs/smart_home.yaml",
	}

	templateFile, ok := templateFiles[templateType]
	if !ok {
		return fmt.Errorf("unknown template type: %s", templateType)
	}

	// Check if template file exists
	if _, err := os.Stat(templateFile); os.IsNotExist(err) {
		// Fallback to using the loader's GenerateTemplate method
		loader := config.NewLoader()
		cfg, err := loader.GenerateTemplate(templateType)
		if err != nil {
			return fmt.Errorf("failed to generate template: %v", err)
		}

		if outputFile != "" {
			if err := loader.SaveToFile(cfg, outputFile); err != nil {
				return fmt.Errorf("failed to save config file: %v", err)
			}
			fmt.Printf("Configuration template saved to: %s\n", outputFile)
		} else {
			// 輸出到 stdout
			data, err := yamlMarshal(cfg)
			if err != nil {
				return fmt.Errorf("failed to marshal config: %v", err)
			}
			fmt.Print(string(data))
		}
		return nil
	}

	// Read the template file
	data, err := os.ReadFile(templateFile)
	if err != nil {
		return fmt.Errorf("failed to read template file: %v", err)
	}

	if outputFile != "" {
		// Copy to output file
		if err := os.WriteFile(outputFile, data, 0644); err != nil {
			return fmt.Errorf("failed to save config file: %v", err)
		}
		fmt.Printf("Configuration template '%s' saved to: %s\n", templateType, outputFile)
	} else {
		// Output to stdout
		fmt.Print(string(data))
	}

	return nil
}

func validateConfig(cmd *cobra.Command, args []string) error {
	configPath := args[0]

	loader := config.NewLoader()
	cfg, err := loader.LoadFromFile(configPath)
	if err != nil {
		return fmt.Errorf("configuration validation failed: %v", err)
	}

	fmt.Printf("Configuration file '%s' is valid\n", configPath)
	fmt.Printf("Simulation: %s\n", cfg.Simulation.Name)
	fmt.Printf("Network devices: %d\n", len(cfg.Devices.NetworkDevices))
	fmt.Printf("IoT devices: %d\n", len(cfg.Devices.IoTDevices))
	fmt.Printf("Client devices: %d\n", len(cfg.Devices.ClientDevices))
	fmt.Printf("Scenarios: %d\n", len(cfg.Scenarios))

	return nil
}

// Simulator 主模擬器
type Simulator struct {
	config             *config.SimulationConfig
	topologyManager    *network.TopologyManager
	interactionManager *interaction.InteractionManager
	stateSync          *sync.StateSync
	deviceManager      *devices.DeviceManager
	devices            map[string]base.Device
	logger             *logrus.Entry
}

// NewSimulator 建立新的模擬器
func NewSimulator(cfg *config.SimulationConfig) (*Simulator, error) {
	logger := logrus.WithField("component", "simulator")

	sim := &Simulator{
		config:             cfg,
		topologyManager:    network.NewTopologyManager(),
		interactionManager: interaction.NewInteractionManager(),
		stateSync:          sync.NewStateSync(),
		deviceManager:      devices.NewDeviceManager(logger.Logger),
		devices:            make(map[string]base.Device),
		logger:             logger,
	}

	// 設定 DHCP 池
	if err := sim.setupDHCP(); err != nil {
		return nil, fmt.Errorf("failed to setup DHCP: %v", err)
	}

	// 建立設備
	if err := sim.createDevices(); err != nil {
		return nil, fmt.Errorf("failed to create devices: %v", err)
	}

	return sim, nil
}

// Run 執行模擬
func (s *Simulator) Run(ctx context.Context) error {
	s.logger.Info("Starting simulation")

	// 啟動拓撲管理器
	if err := s.topologyManager.Start(ctx); err != nil {
		return fmt.Errorf("failed to start topology manager: %v", err)
	}
	defer s.topologyManager.Stop()

	// 啟動互動管理器
	if err := s.interactionManager.Start(ctx); err != nil {
		return fmt.Errorf("failed to start interaction manager: %v", err)
	}
	defer s.interactionManager.Stop()

	// 啟動狀態同步管理器
	if err := s.stateSync.Start(ctx); err != nil {
		return fmt.Errorf("failed to start state sync: %v", err)
	}
	defer s.stateSync.Stop()

	// 註冊所有設備到管理器
	for deviceID, device := range s.devices {
		s.interactionManager.RegisterDevice(deviceID, device)
		s.stateSync.RegisterDevice(deviceID, device)
	}

	// 載入預定義的規則
	s.loadInteractionRules()
	s.loadSyncRules()

	// 啟動所有設備
	for deviceID, device := range s.devices {
		if err := device.Start(ctx); err != nil {
			s.logger.WithError(err).Errorf("Failed to start device %s", deviceID)
			continue
		}
		s.logger.WithField("device_id", deviceID).Info("Device started")
	}

	// 建立設備連接
	if err := s.createConnections(); err != nil {
		s.logger.WithError(err).Warn("Failed to create some connections")
	}

	// 啟動監控和統計
	go s.monitorSimulation(ctx)

	// 等待模擬結束
	if s.config.Simulation.Duration > 0 {
		timer := time.NewTimer(s.config.Simulation.Duration)
		defer timer.Stop()

		select {
		case <-ctx.Done():
			s.logger.Info("Simulation cancelled")
		case <-timer.C:
			s.logger.Info("Simulation completed")
		}
	} else {
		<-ctx.Done()
		s.logger.Info("Simulation stopped")
	}

	// 停止所有設備並從互動管理器中取消註冊
	for deviceID, device := range s.devices {
		if err := device.Stop(); err != nil {
			s.logger.WithError(err).Errorf("Failed to stop device %s", deviceID)
		}
		s.interactionManager.UnregisterDevice(deviceID)
		s.stateSync.UnregisterDevice(deviceID)
	}

	return nil
}

// 內部方法
func (s *Simulator) setupDHCP() error {
	dhcp := s.config.Network.DHCPPool
	return s.topologyManager.SetDHCPPool(
		dhcp.StartIP,
		dhcp.EndIP,
		dhcp.Gateway,
		dhcp.DNS,
		dhcp.LeaseTime,
	)
}

func (s *Simulator) createDevices() error {
	// 建立網路設備
	for _, deviceCfg := range s.config.Devices.NetworkDevices {
		device, err := s.createNetworkDevice(deviceCfg)
		if err != nil {
			return fmt.Errorf("failed to create network device %s: %v", deviceCfg.ID, err)
		}
		s.devices[deviceCfg.ID] = device
		s.topologyManager.AddDevice(device)
	}

	// 建立 IoT 設備
	for _, deviceCfg := range s.config.Devices.IoTDevices {
		device, err := s.createIoTDevice(deviceCfg)
		if err != nil {
			return fmt.Errorf("failed to create IoT device %s: %v", deviceCfg.ID, err)
		}
		s.devices[deviceCfg.ID] = device
		s.topologyManager.AddDevice(device)
	}

	// 建立客戶端設備
	for _, deviceCfg := range s.config.Devices.ClientDevices {
		device, err := s.createClientDevice(deviceCfg)
		if err != nil {
			return fmt.Errorf("failed to create client device %s: %v", deviceCfg.ID, err)
		}
		s.devices[deviceCfg.ID] = device
		s.topologyManager.AddDevice(device)
	}

	return nil
}

func (s *Simulator) createNetworkDevice(cfg config.NetworkDeviceConfig) (base.Device, error) {
	// 轉換為通用 DeviceConfig
	deviceConfig := config.DeviceConfig{
		ID:             cfg.ID,
		Type:           cfg.Type,
		IPAddress:      cfg.IPAddress,
		Tenant:         cfg.Tenant,
		Site:           cfg.Site,
		ConnectionType: cfg.ConnectionType,
		Firmware:       cfg.Firmware,
		Protocols:      cfg.Protocols,
		Location:       cfg.Location,
	}

	return s.deviceManager.CreateDevice(deviceConfig)
}

func (s *Simulator) createIoTDevice(cfg config.IoTDeviceConfig) (base.Device, error) {
	// 轉換為通用 DeviceConfig
	deviceConfig := config.DeviceConfig{
		ID:             cfg.ID,
		Type:           cfg.Type,
		IPAddress:      cfg.IPAddress,
		Tenant:         cfg.Tenant,
		Site:           cfg.Site,
		ConnectionType: cfg.ConnectionType,
		Firmware:       cfg.Firmware,
		Protocols:      cfg.Protocols,
		Location:       cfg.Location,
	}

	return s.deviceManager.CreateDevice(deviceConfig)
}

func (s *Simulator) createClientDevice(cfg config.ClientDeviceConfig) (base.Device, error) {
	// 轉換為通用 DeviceConfig
	deviceConfig := config.DeviceConfig{
		ID:             cfg.ID,
		Type:           cfg.Type,
		IPAddress:      cfg.IPAddress,
		Tenant:         cfg.Tenant,
		Site:           cfg.Site,
		ConnectionType: cfg.ConnectionType,
		Firmware:       cfg.Firmware,
		Protocols:      cfg.Protocols,
		Location:       cfg.Location,
	}

	return s.deviceManager.CreateDevice(deviceConfig)
}

func (s *Simulator) createConnections() error {
	// 根據網路拓撲類型建立連接
	switch s.config.Network.Topology {
	case "single_router":
		return s.createSingleRouterTopology()
	case "mesh_network":
		return s.createMeshTopology()
	default:
		return fmt.Errorf("unsupported topology type: %s", s.config.Network.Topology)
	}
}

func (s *Simulator) createSingleRouterTopology() error {
	// 尋找路由器
	var routerID string
	for deviceID, device := range s.devices {
		if device.GetDeviceType() == "router" {
			routerID = deviceID
			break
		}
	}

	if routerID == "" {
		return fmt.Errorf("no router found for single router topology")
	}

	// 將所有設備連接到路由器
	for deviceID, device := range s.devices {
		if deviceID == routerID {
			continue
		}

		connType := network.ConnectionWiFi
		if device.GetNetworkInfo().ConnectionType == "ethernet" {
			connType = network.ConnectionEthernet
		}

		_, err := s.topologyManager.AddConnection(routerID, deviceID, connType)
		if err != nil {
			s.logger.WithError(err).Warnf("Failed to connect device %s to router", deviceID)
		}
	}

	return nil
}

func (s *Simulator) createMeshTopology() error {
	// 實作 Mesh 網路拓撲
	// 在 Mesh 網路中，設備可以相互連接形成網狀結構

	// 找出所有支援 Mesh 的設備（路由器、AP、Mesh 節點）
	var meshDevices []string
	for deviceID, device := range s.devices {
		deviceType := device.GetDeviceType()
		if deviceType == "router" || deviceType == "access_point" || deviceType == "mesh_node" {
			meshDevices = append(meshDevices, deviceID)
		}
	}

	if len(meshDevices) < 2 {
		return fmt.Errorf("insufficient mesh-capable devices for mesh topology (need at least 2, found %d)", len(meshDevices))
	}

	// 建立 Mesh 骨幹網路 - 每個 Mesh 設備至少連接到 2 個其他 Mesh 設備
	for i, deviceID := range meshDevices {
		// 連接到下一個設備（環形連接）
		nextIdx := (i + 1) % len(meshDevices)
		_, err := s.topologyManager.AddConnection(deviceID, meshDevices[nextIdx], network.ConnectionWiFi)
		if err != nil {
			s.logger.WithError(err).Warnf("Failed to create mesh connection between %s and %s", deviceID, meshDevices[nextIdx])
		}

		// 如果有 3 個以上的 Mesh 設備，建立額外的交叉連接
		if len(meshDevices) > 3 {
			crossIdx := (i + 2) % len(meshDevices)
			_, err := s.topologyManager.AddConnection(deviceID, meshDevices[crossIdx], network.ConnectionWiFi)
			if err != nil {
				s.logger.WithError(err).Warnf("Failed to create cross mesh connection between %s and %s", deviceID, meshDevices[crossIdx])
			}
		}
	}

	// 將非 Mesh 設備連接到最近的 Mesh 節點
	for deviceID, device := range s.devices {
		deviceType := device.GetDeviceType()
		// 跳過 Mesh 設備本身
		if deviceType == "router" || deviceType == "access_point" || deviceType == "mesh_node" {
			continue
		}

		// 連接到第一個可用的 Mesh 設備
		if len(meshDevices) > 0 {
			// 根據設備類型選擇連接類型
			connType := network.ConnectionWiFi
			if device.GetNetworkInfo().ConnectionType == "ethernet" {
				connType = network.ConnectionEthernet
			}

			// 簡單地連接到第一個 Mesh 設備，實際應該基於距離或負載均衡
			meshDevice := meshDevices[0]
			_, err := s.topologyManager.AddConnection(meshDevice, deviceID, connType)
			if err != nil {
				s.logger.WithError(err).Warnf("Failed to connect device %s to mesh node %s", deviceID, meshDevice)
			}
		}
	}

	return nil
}

// loadInteractionRules 載入互動規則
func (s *Simulator) loadInteractionRules() {
	s.logger.Info("Loading predefined interaction rules")

	// 載入所有預定義規則
	rules := interaction.GetPredefinedRules()

	// 根據模擬器中的設備過濾有效規則
	validRules := s.filterValidRules(rules)

	// 添加到互動管理器
	for _, rule := range validRules {
		s.interactionManager.AddInteractionRule(rule)
	}

	s.logger.WithField("rule_count", len(validRules)).Info("Interaction rules loaded")
}

// filterValidRules 過濾有效的互動規則
func (s *Simulator) filterValidRules(rules []interaction.InteractionRule) []interaction.InteractionRule {
	var validRules []interaction.InteractionRule

	for _, rule := range rules {
		// 檢查觸發設備是否存在
		if rule.Trigger.DeviceID != "" {
			if _, exists := s.devices[rule.Trigger.DeviceID]; !exists {
				s.logger.WithFields(logrus.Fields{
					"rule_id":   rule.ID,
					"device_id": rule.Trigger.DeviceID,
				}).Debug("Skipping rule - trigger device not found")
				continue
			}
		}

		// 檢查目標設備是否存在
		ruleValid := true
		for _, action := range rule.Actions {
			if action.TargetID != "" && action.TargetID != "system" {
				if _, exists := s.devices[action.TargetID]; !exists {
					s.logger.WithFields(logrus.Fields{
						"rule_id":   rule.ID,
						"target_id": action.TargetID,
					}).Debug("Skipping rule - target device not found")
					ruleValid = false
					break
				}
			}
		}

		if ruleValid {
			validRules = append(validRules, rule)
		}
	}

	return validRules
}

// loadSyncRules 載入同步規則
func (s *Simulator) loadSyncRules() {
	s.logger.Info("Loading predefined sync rules")

	// 載入所有預定義同步規則
	rules := sync.GetPredefinedSyncRules()

	// 根據模擬器中的設備過濾有效規則
	validRules := s.filterValidSyncRules(rules)

	// 添加到狀態同步管理器
	for _, rule := range validRules {
		s.stateSync.AddSyncRule(rule)
	}

	s.logger.WithField("rule_count", len(validRules)).Info("Sync rules loaded")
}

// filterValidSyncRules 過濾有效的同步規則
func (s *Simulator) filterValidSyncRules(rules []sync.SyncRule) []sync.SyncRule {
	var validRules []sync.SyncRule

	for _, rule := range rules {
		// 檢查源設備是否存在
		if rule.Source.DeviceID != "" {
			if _, exists := s.devices[rule.Source.DeviceID]; !exists {
				s.logger.WithFields(logrus.Fields{
					"rule_id":   rule.ID,
					"device_id": rule.Source.DeviceID,
				}).Debug("Skipping sync rule - source device not found")
				continue
			}
		}

		// 檢查目標設備是否存在
		ruleValid := true
		for _, action := range rule.Targets {
			if action.TargetID != "" && action.TargetID != "system" {
				if _, exists := s.devices[action.TargetID]; !exists {
					// 如果指定了目標類型而不是具體設備ID，檢查是否有該類型的設備
					if action.TargetType != "" {
						hasType := false
						for _, device := range s.devices {
							if device.GetDeviceType() == action.TargetType {
								hasType = true
								break
							}
						}
						if !hasType {
							s.logger.WithFields(logrus.Fields{
								"rule_id":     rule.ID,
								"target_type": action.TargetType,
							}).Debug("Skipping sync rule - no target device of specified type")
							ruleValid = false
							break
						}
					} else {
						s.logger.WithFields(logrus.Fields{
							"rule_id":   rule.ID,
							"target_id": action.TargetID,
						}).Debug("Skipping sync rule - target device not found")
						ruleValid = false
						break
					}
				}
			}
		}

		if ruleValid {
			validRules = append(validRules, rule)
		}
	}

	return validRules
}

func (s *Simulator) monitorSimulation(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// 網路拓撲統計
			topologyStats := s.topologyManager.GetStats()

			// 互動管理器統計
			interactionStats := s.interactionManager.GetStatistics()

			// 狀態同步統計
			syncStats := s.stateSync.GetStatistics()

			s.logger.WithFields(logrus.Fields{
				"total_devices":       topologyStats.TotalDevices,
				"active_connections":  topologyStats.ActiveConnections,
				"total_bandwidth":     topologyStats.TotalBandwidth,
				"average_latency":     topologyStats.AverageLatency,
				"packet_loss_rate":    topologyStats.PacketLossRate,
				"interaction_rules":   interactionStats["total_rules"],
				"enabled_rules":       interactionStats["enabled_rules"],
				"interaction_running": interactionStats["running"],
				"sync_rules":          syncStats["sync_rules"],
				"active_states":       syncStats["active_states"],
				"sync_running":        syncStats["running"],
				"history_records":     syncStats["total_history_records"],
			}).Info("Simulation statistics")
		}
	}
}

// 工具函數
func setupLogging() {
	if verbose {
		logrus.SetLevel(logrus.DebugLevel)
	} else {
		level, err := logrus.ParseLevel(logLevel)
		if err != nil {
			level = logrus.InfoLevel
		}
		logrus.SetLevel(level)
	}

	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
		ForceColors:   true,
	})
}

func findConfigFile(args []string) string {
	if len(args) > 0 {
		return args[0]
	}

	// 嘗試尋找預設配置檔案
	candidates := []string{
		"config.yaml",
		"configs/home_basic.yaml",
		"configs/config.yaml",
	}

	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}

	return ""
}

func loadConfiguration(loader *config.Loader, configPath string) (*config.SimulationConfig, error) {
	if configPath != "" {
		return loader.LoadFromFile(configPath)
	}

	// 嘗試尋找配置檔案
	foundPath, err := loader.FindConfigFile("config")
	if err == nil {
		logrus.WithField("config_file", foundPath).Info("Found configuration file")
		return loader.LoadFromFile(foundPath)
	}

	// 使用預設配置
	logrus.Info("Using default configuration")
	return loader.LoadDefault(), nil
}

func yamlMarshal(v interface{}) ([]byte, error) {
	// 實作 YAML 序列化
	data, err := yaml.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal YAML: %v", err)
	}
	return data, nil
}
