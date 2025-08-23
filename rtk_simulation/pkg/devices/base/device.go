package base

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// BaseDevice 基礎設備實作，提供所有設備的共通功能
type BaseDevice struct {
	// 基本設備資訊
	id         string
	deviceType string
	macAddress string
	ipAddress  string
	tenant     string
	site       string
	location   *Location

	// 狀態管理
	running    bool
	health     string
	startTime  time.Time
	lastUpdate time.Time
	mu         sync.RWMutex

	// 硬體狀態
	cpuUsage    float64
	memoryUsage float64
	diskUsage   float64
	temperature float64
	batteryV    float64

	// 網路狀態
	networkInfo NetworkInfo
	bytesRx     int64
	bytesTx     int64
	packetsRx   int64
	packetsTx   int64
	rssi        int

	// 診斷資訊
	errorCount   int
	restartCount int
	lastError    *string
	errors       []DeviceError

	// 統計資訊
	stats DeviceStats

	// 配置
	firmware     string
	capabilities DeviceCapabilities

	// 日誌
	logger *logrus.Entry

	// MQTT 客戶端
	mqttClient *MQTTClient
	mqttConfig MQTTConfig

	// 內部狀態更新
	statusUpdateInterval time.Duration
}

// NewBaseDevice 建立新的基礎設備
func NewBaseDevice(config DeviceConfig) *BaseDevice {
	deviceID := config.ID
	if deviceID == "" {
		deviceID = generateDeviceID()
	}

	macAddr := config.MACAddress
	if macAddr == "" {
		macAddr = generateMACAddress()
	}

	logger := logrus.WithFields(logrus.Fields{
		"device_id":   deviceID,
		"device_type": config.Type,
		"tenant":      config.Tenant,
		"site":        config.Site,
	})

	return &BaseDevice{
		id:         deviceID,
		deviceType: config.Type,
		macAddress: macAddr,
		ipAddress:  config.IPAddress,
		tenant:     config.Tenant,
		site:       config.Site,
		location:   config.Location,

		health:      "ok",
		cpuUsage:    20.0 + rand.Float64()*30.0, // 20-50%
		memoryUsage: 30.0 + rand.Float64()*40.0, // 30-70%
		diskUsage:   20.0 + rand.Float64()*60.0, // 20-80%
		temperature: 30.0 + rand.Float64()*20.0, // 30-50°C
		batteryV:    config.BatteryV,

		networkInfo: NetworkInfo{
			IP:             config.IPAddress,
			Gateway:        config.Gateway,
			Netmask:        config.Netmask,
			MACAddress:     macAddr,
			ConnectionType: config.ConnectionType,
		},

		firmware: config.Firmware,
		capabilities: DeviceCapabilities{
			DeviceType: config.Type,
			Protocols:  config.Protocols,
		},

		logger:               logger,
		statusUpdateInterval: 5 * time.Second,
	}
}

// NewBaseDeviceWithMQTT 建立新的基礎設備（含 MQTT）
func NewBaseDeviceWithMQTT(config DeviceConfig, mqttConfig MQTTConfig) *BaseDevice {
	device := NewBaseDevice(config)
	device.mqttConfig = mqttConfig

	// 設定預設值
	if mqttConfig.Broker != "" {
		if mqttConfig.Port == 0 {
			device.mqttConfig.Port = 1883
		}
		if mqttConfig.KeepAlive == 0 {
			device.mqttConfig.KeepAlive = 60 * time.Second
		}
		if mqttConfig.ClientID == "" {
			device.mqttConfig.ClientID = fmt.Sprintf("rtk_sim_%s_%s", device.deviceType, device.id)
		}
	}

	return device
}

// DeviceConfig 設備配置結構
type DeviceConfig struct {
	ID             string
	Type           string
	MACAddress     string
	IPAddress      string
	Tenant         string
	Site           string
	Location       *Location
	Gateway        string
	Netmask        string
	ConnectionType string
	BatteryV       float64
	Firmware       string
	Protocols      []string
	Extra          map[string]interface{} // 設備特定配置
}

// 實作 Device 介面
func (d *BaseDevice) GetDeviceID() string {
	return d.id
}

func (d *BaseDevice) GetDeviceType() string {
	return d.deviceType
}

func (d *BaseDevice) GetMACAddress() string {
	return d.macAddress
}

func (d *BaseDevice) GetIPAddress() string {
	return d.ipAddress
}

func (d *BaseDevice) GetTenant() string {
	return d.tenant
}

func (d *BaseDevice) GetSite() string {
	return d.site
}

func (d *BaseDevice) Start(ctx context.Context) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.running {
		return fmt.Errorf("device %s is already running", d.id)
	}

	d.running = true
	d.startTime = time.Now()
	d.lastUpdate = time.Now()

	// 初始化並連接 MQTT 客戶端（如果配置存在）
	if d.mqttConfig.Broker != "" {
		// 設定客戶端 ID
		if d.mqttConfig.ClientID == "" {
			d.mqttConfig.ClientID = fmt.Sprintf("rtk_sim_%s_%s", d.deviceType, d.id)
		}

		// 建立 MQTT 客戶端
		d.mqttClient = NewMQTTClient(d.mqttConfig, d)

		// 連接到 MQTT Broker
		if err := d.mqttClient.Connect(); err != nil {
			d.logger.WithError(err).Error("Failed to connect to MQTT broker")
			// 繼續運行，但沒有 MQTT 功能
		}
	}

	d.logger.Info("Device started")

	// 啟動狀態更新循環
	go d.statusUpdateLoop(ctx)

	// 啟動 MQTT 發布循環（如果已連接）
	if d.mqttClient != nil && d.mqttClient.IsConnected() {
		go d.mqttPublishLoop(ctx)
	}

	return nil
}

func (d *BaseDevice) Stop() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if !d.running {
		return fmt.Errorf("device %s is not running", d.id)
	}

	d.running = false

	// 斷開 MQTT 連接
	if d.mqttClient != nil {
		d.mqttClient.Disconnect()
	}

	d.logger.Info("Device stopped")

	return nil
}

func (d *BaseDevice) IsRunning() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.running
}

func (d *BaseDevice) GetHealth() string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.health
}

func (d *BaseDevice) GetUptime() time.Duration {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if !d.running {
		return 0
	}
	return time.Since(d.startTime)
}

func (d *BaseDevice) GetNetworkInfo() NetworkInfo {
	d.mu.RLock()
	defer d.mu.RUnlock()

	d.networkInfo.BytesRx = d.bytesRx
	d.networkInfo.BytesTx = d.bytesTx
	d.networkInfo.RSSI = d.rssi

	return d.networkInfo
}

func (d *BaseDevice) UpdateStatus() {
	d.mu.Lock()
	defer d.mu.Unlock()

	// 更新硬體狀態（模擬變化）
	d.cpuUsage += (rand.Float64() - 0.5) * 10.0
	if d.cpuUsage < 5.0 {
		d.cpuUsage = 5.0
	} else if d.cpuUsage > 95.0 {
		d.cpuUsage = 95.0
	}

	d.memoryUsage += (rand.Float64() - 0.5) * 5.0
	if d.memoryUsage < 10.0 {
		d.memoryUsage = 10.0
	} else if d.memoryUsage > 90.0 {
		d.memoryUsage = 90.0
	}

	d.temperature += (rand.Float64() - 0.5) * 2.0
	if d.temperature < 25.0 {
		d.temperature = 25.0
	} else if d.temperature > 70.0 {
		d.temperature = 70.0
	}

	// 更新網路統計
	d.updateNetworkStats()

	// 根據狀態決定健康度
	d.updateHealth()

	d.lastUpdate = time.Now()
}

func (d *BaseDevice) GenerateStatePayload() StatePayload {
	d.mu.RLock()
	defer d.mu.RUnlock()

	uptime := int64(0)
	if d.running {
		uptime = int64(time.Since(d.startTime).Seconds())
	}

	return StatePayload{
		Health:      d.health,
		Firmware:    d.firmware,
		UptimeS:     uptime,
		CPUUsage:    d.cpuUsage,
		MemoryUsage: d.memoryUsage,
		DiskUsage:   d.diskUsage,
		TempC:       d.temperature,
		DeviceType:  d.deviceType,
		BatteryV:    d.batteryV,
		Net:         d.GetNetworkInfo(),
		Diagnosis: DiagnosisInfo{
			LastError:    d.lastError,
			ErrorCount:   d.errorCount,
			RestartCount: d.restartCount,
		},
	}
}

func (d *BaseDevice) GenerateTelemetryData() map[string]TelemetryPayload {
	d.mu.RLock()
	defer d.mu.RUnlock()

	telemetry := make(map[string]TelemetryPayload)

	// 系統遙測
	telemetry["system"] = TelemetryPayload{
		"cpu_usage":    d.cpuUsage,
		"memory_usage": d.memoryUsage,
		"disk_usage":   d.diskUsage,
		"temperature":  d.temperature,
	}

	// 網路遙測
	telemetry["network"] = TelemetryPayload{
		"bytes_rx":   d.bytesRx,
		"bytes_tx":   d.bytesTx,
		"packets_rx": d.packetsRx,
		"packets_tx": d.packetsTx,
		"rssi":       d.rssi,
	}

	// 電池遙測（如果有）
	if d.batteryV > 0 {
		telemetry["battery"] = TelemetryPayload{
			"voltage":    d.batteryV,
			"percentage": d.calculateBatteryPercentage(),
		}
	}

	return telemetry
}

func (d *BaseDevice) GenerateEvents() []Event {
	d.mu.RLock()
	defer d.mu.RUnlock()

	var events []Event

	// 健康狀態變化事件
	if d.health == "warn" && rand.Float64() < 0.1 {
		events = append(events, Event{
			EventType: "system.warning",
			Severity:  "warning",
			Message:   "System performance degraded",
			Extra: map[string]interface{}{
				"cpu_usage":    d.cpuUsage,
				"memory_usage": d.memoryUsage,
				"temperature":  d.temperature,
			},
		})
	}

	// 低電量事件（電池設備）
	if d.batteryV > 0 && d.batteryV < 3.3 && rand.Float64() < 0.2 {
		events = append(events, Event{
			EventType: "system.battery_low",
			Severity:  "warning",
			Message:   "Battery level is low",
			Extra: map[string]interface{}{
				"battery_voltage":    d.batteryV,
				"battery_percentage": d.calculateBatteryPercentage(),
			},
		})
	}

	return events
}

func (d *BaseDevice) HandleCommand(cmd Command) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.logger.WithField("command", cmd.Type).Info("Handling command")

	switch cmd.Type {
	case "device.reboot":
		return d.handleReboot()
	case "device.reset":
		return d.handleReset()
	case "device.get_status":
		return d.handleGetStatus()
	default:
		err := fmt.Errorf("unsupported command: %s", cmd.Type)
		d.addError("command", err.Error(), "warning")
		return err
	}
}

// 內部方法
func (d *BaseDevice) statusUpdateLoop(ctx context.Context) {
	ticker := time.NewTicker(d.statusUpdateInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if !d.IsRunning() {
				return
			}
			d.UpdateStatus()
		}
	}
}

func (d *BaseDevice) updateNetworkStats() {
	// 模擬網路活動
	rxInc := int64(rand.Intn(10000)) // 0-10KB
	txInc := int64(rand.Intn(5000))  // 0-5KB

	d.bytesRx += rxInc
	d.bytesTx += txInc
	d.packetsRx += rxInc / 1024
	d.packetsTx += txInc / 1024

	// 更新 RSSI（WiFi 設備）
	if d.networkInfo.ConnectionType == "wifi" {
		d.rssi += rand.Intn(10) - 5 // ±5 dBm
		if d.rssi > -30 {
			d.rssi = -30
		} else if d.rssi < -90 {
			d.rssi = -90
		}
	}
}

func (d *BaseDevice) updateHealth() {
	if d.cpuUsage > 90 || d.memoryUsage > 90 || d.temperature > 65 {
		d.health = "error"
	} else if d.cpuUsage > 80 || d.memoryUsage > 80 || d.temperature > 55 {
		d.health = "warn"
	} else {
		d.health = "ok"
	}
}

func (d *BaseDevice) calculateBatteryPercentage() float64 {
	if d.batteryV <= 0 {
		return 0
	}
	// 假設 3.2V-4.2V 範圍
	percentage := (d.batteryV - 3.2) / (4.2 - 3.2) * 100
	if percentage < 0 {
		return 0
	}
	if percentage > 100 {
		return 100
	}
	return percentage
}

func (d *BaseDevice) addError(errorType, message, severity string) {
	error := DeviceError{
		Timestamp: time.Now(),
		Type:      errorType,
		Message:   message,
		Severity:  severity,
	}

	d.errors = append(d.errors, error)
	d.errorCount++

	// 保持最近的 100 個錯誤
	if len(d.errors) > 100 {
		d.errors = d.errors[1:]
	}

	if severity == "error" {
		d.lastError = &message
	}
}

func (d *BaseDevice) handleReboot() error {
	d.logger.Info("Rebooting device")
	d.restartCount++

	// 模擬重啟過程
	time.Sleep(2 * time.Second)

	// 重置一些狀態
	d.cpuUsage = 20.0 + rand.Float64()*20.0
	d.memoryUsage = 30.0 + rand.Float64()*30.0
	d.health = "ok"
	d.startTime = time.Now()

	return nil
}

func (d *BaseDevice) handleReset() error {
	d.logger.Info("Resetting device to factory defaults")

	// 重置錯誤計數
	d.errorCount = 0
	d.lastError = nil
	d.errors = []DeviceError{}

	return nil
}

func (d *BaseDevice) handleGetStatus() error {
	d.logger.Info("Status requested")
	return nil
}

// 工具函數
func generateDeviceID() string {
	return uuid.New().String()[:8]
}

func generateMACAddress() string {
	mac := make([]byte, 6)
	rand.Read(mac)
	return fmt.Sprintf("%02x%02x%02x%02x%02x%02x",
		mac[0], mac[1], mac[2], mac[3], mac[4], mac[5])
}

// PublishCommandResponse 發布命令回應
func (d *BaseDevice) PublishCommandResponse(response CommandResponse) error {
	if d.mqttClient == nil || !d.mqttClient.IsConnected() {
		d.logger.Debug("MQTT client not connected, skipping command response publish")
		return nil
	}

	result := CommandResult{
		CommandID: response.CommandID,
		Success:   response.Status == "success",
		Message:   response.Data,
	}

	if response.Error != "" {
		result.Error = response.Error
		result.Success = false
	}

	return d.mqttClient.PublishCommandResult(result)
}

// PublishEvent 發布事件
func (d *BaseDevice) PublishEvent(event Event) error {
	if d.mqttClient == nil || !d.mqttClient.IsConnected() {
		d.logger.Debug("MQTT client not connected, skipping event publish")
		return nil
	}

	// 從事件類型中提取主要類型
	eventType := event.EventType
	if eventType == "" {
		eventType = "generic"
	}

	return d.mqttClient.PublishEvent(eventType, event)
}

// SetMQTTConfig 設定 MQTT 配置
func (d *BaseDevice) SetMQTTConfig(config MQTTConfig) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.mqttConfig = config
}

// mqttPublishLoop MQTT 定期發布循環
func (d *BaseDevice) mqttPublishLoop(ctx context.Context) {
	stateTicker := time.NewTicker(30 * time.Second)
	telemetryTicker := time.NewTicker(10 * time.Second)
	eventTicker := time.NewTicker(5 * time.Second)

	defer stateTicker.Stop()
	defer telemetryTicker.Stop()
	defer eventTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-stateTicker.C:
			if d.mqttClient != nil && d.mqttClient.IsConnected() {
				state := d.GenerateStatePayload()
				if err := d.mqttClient.PublishState(state); err != nil {
					d.logger.WithError(err).Error("Failed to publish state")
				}
			}
		case <-telemetryTicker.C:
			if d.mqttClient != nil && d.mqttClient.IsConnected() {
				telemetry := d.GenerateTelemetryData()
				for metric, data := range telemetry {
					if err := d.mqttClient.PublishTelemetry(metric, data); err != nil {
						d.logger.WithError(err).Error("Failed to publish telemetry")
					}
				}
			}
		case <-eventTicker.C:
			if d.mqttClient != nil && d.mqttClient.IsConnected() {
				events := d.GenerateEvents()
				for _, event := range events {
					if err := d.PublishEvent(event); err != nil {
						d.logger.WithError(err).Error("Failed to publish event")
					}
				}
			}
		}
	}
}
