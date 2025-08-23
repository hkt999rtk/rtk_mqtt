package network

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"sync"
	"time"

	// "github.com/sirupsen/logrus"
	// "rtk_simulation/pkg/config"
	"rtk_simulation/pkg/devices/base"
)

// WiFiConfig 定義WiFi配置
type WiFiConfig struct {
	Networks []WiFiNetwork `json:"networks"`
	Channels ChannelConfig `json:"channels"`
	Security WiFiSecurity  `json:"security"`
}

type WiFiNetwork struct {
	SSID       string `json:"ssid"`
	Band       string `json:"band"`
	Hidden     bool   `json:"hidden"`
	MaxClients int    `json:"max_clients"`
	TxPower    int    `json:"tx_power"`
}

type ChannelConfig struct {
	Channel2G  int  `json:"channel_2g"`
	Channel5G  int  `json:"channel_5g"`
	AutoSelect bool `json:"auto_select"`
	Width2G    int  `json:"width_2g"`
	Width5G    int  `json:"width_5g"`
}

type WiFiSecurity struct {
	Type       string `json:"type"`
	Password   string `json:"password"`
	Encryption string `json:"encryption"`
}

type SecurityConfig struct {
	WPSEnabled         bool     `json:"wps_enabled"`
	MACFiltering       bool     `json:"mac_filtering"`
	AccessControl      bool     `json:"access_control"`
	AllowedMACs        []string `json:"allowed_macs"`
	BlockedMACs        []string `json:"blocked_macs"`
	GuestNetwork       bool     `json:"guest_network"`
	BandSteering       bool     `json:"band_steering"`
	LoadBalancing      bool     `json:"load_balancing"`
	FastRoaming        bool     `json:"fast_roaming"`
	AuthenticationMode string   `json:"authentication_mode"`
}

type BandwidthUsage struct {
	Upload   float64 `json:"upload"`
	Download float64 `json:"download"`
	Total    float64 `json:"total"`
}

type AccessPoint struct {
	*base.BaseDevice

	wifiConfig       WiFiConfig
	connectedClients map[string]*ConnectedClient
	beaconConfig     BeaconConfig
	securityConfig   SecurityConfig
	signalStrength   map[string]float64
	channelStats     ChannelStatistics
	bandwidthUsage   BandwidthUsage
	mu               sync.RWMutex
}

type BeaconConfig struct {
	Interval    time.Duration `json:"interval"`
	TxPower     int           `json:"tx_power"`
	BeaconCount int64         `json:"beacon_count"`
	LastBeacon  time.Time     `json:"last_beacon"`
}

type ChannelStatistics struct {
	Channel      int     `json:"channel"`
	Utilization  float64 `json:"utilization"`
	NoiseLevel   float64 `json:"noise_level"`
	Interference float64 `json:"interference"`
	Efficiency   float64 `json:"efficiency"`
	RetryRate    float64 `json:"retry_rate"`
}

func NewAccessPoint(deviceConfig base.DeviceConfig, mqttConfig base.MQTTConfig) (*AccessPoint, error) {
	baseDevice := base.NewBaseDevice(deviceConfig)

	ap := &AccessPoint{
		BaseDevice:       baseDevice,
		connectedClients: make(map[string]*ConnectedClient),
		signalStrength:   make(map[string]float64),
	}

	if err := ap.initializeWiFiConfig(deviceConfig); err != nil {
		return nil, err
	}

	ap.initializeBeaconConfig()
	ap.initializeSecurityConfig(deviceConfig)
	ap.initializeChannelStats()

	return ap, nil
}

func (ap *AccessPoint) initializeWiFiConfig(config base.DeviceConfig) error {
	ap.wifiConfig = WiFiConfig{
		Networks: []WiFiNetwork{
			{
				SSID:       "HomeAP-2.4G",
				Band:       "2.4GHz",
				Hidden:     false,
				MaxClients: 50,
				TxPower:    20,
			},
			{
				SSID:       "HomeAP-5G",
				Band:       "5GHz",
				Hidden:     false,
				MaxClients: 30,
				TxPower:    23,
			},
		},
		Channels: ChannelConfig{
			Channel2G:  6,
			Channel5G:  36,
			AutoSelect: true,
			Width2G:    20,
			Width5G:    80,
		},
		Security: WiFiSecurity{
			Type:       "WPA3",
			Password:   "accesspoint123",
			Encryption: "AES",
		},
	}

	// 從配置中載入 WiFi 設定
	if config.Extra != nil {
		// 載入 SSID
		if ssid2g, ok := config.Extra["ssid_2g"].(string); ok {
			ap.wifiConfig.Networks[0].SSID = ssid2g
		}
		if ssid5g, ok := config.Extra["ssid_5g"].(string); ok {
			ap.wifiConfig.Networks[1].SSID = ssid5g
		}

		// 載入密碼
		if password, ok := config.Extra["wifi_password"].(string); ok {
			ap.wifiConfig.Security.Password = password
		}

		// 載入安全類型
		if secType, ok := config.Extra["security_type"].(string); ok {
			ap.wifiConfig.Security.Type = secType
		}

		// 載入頻道設定
		if ch2g, ok := config.Extra["channel_2g"].(float64); ok {
			ap.wifiConfig.Channels.Channel2G = int(ch2g)
		} else if ch2g, ok := config.Extra["channel_2g"].(int); ok {
			ap.wifiConfig.Channels.Channel2G = ch2g
		}

		if ch5g, ok := config.Extra["channel_5g"].(float64); ok {
			ap.wifiConfig.Channels.Channel5G = int(ch5g)
		} else if ch5g, ok := config.Extra["channel_5g"].(int); ok {
			ap.wifiConfig.Channels.Channel5G = ch5g
		}

		// 載入最大客戶端數
		if maxClients, ok := config.Extra["max_clients"].(float64); ok {
			for i := range ap.wifiConfig.Networks {
				ap.wifiConfig.Networks[i].MaxClients = int(maxClients)
			}
		} else if maxClients, ok := config.Extra["max_clients"].(int); ok {
			for i := range ap.wifiConfig.Networks {
				ap.wifiConfig.Networks[i].MaxClients = maxClients
			}
		}
	}

	return nil
}

func (ap *AccessPoint) initializeBeaconConfig() {
	ap.beaconConfig = BeaconConfig{
		Interval:    100 * time.Millisecond,
		TxPower:     20,
		BeaconCount: 0,
		LastBeacon:  time.Now(),
	}
}

func (ap *AccessPoint) initializeSecurityConfig(config base.DeviceConfig) {
	ap.securityConfig = SecurityConfig{
		WPSEnabled:         false,
		MACFiltering:       false,
		AccessControl:      true,
		GuestNetwork:       false,
		BandSteering:       true,
		LoadBalancing:      true,
		FastRoaming:        true,
		AuthenticationMode: "WPA3-Personal",
	}
}

func (ap *AccessPoint) initializeChannelStats() {
	ap.channelStats = ChannelStatistics{
		Channel:      ap.wifiConfig.Channels.Channel2G,
		Utilization:  0,
		NoiseLevel:   -95,
		Interference: 0,
		Efficiency:   100,
		RetryRate:    0,
	}
}

func (ap *AccessPoint) Start(ctx context.Context) error {
	if err := ap.BaseDevice.Start(ctx); err != nil {
		return err
	}

	go ap.runBeaconTransmission(ctx)
	go ap.runClientManagement(ctx)
	go ap.runChannelOptimization(ctx)
	go ap.runSecurityMonitoring(ctx)
	go ap.runBandwidthMonitoring(ctx)

	// Access Point started successfully
	return nil
}

func (ap *AccessPoint) runBeaconTransmission(ctx context.Context) {
	ticker := time.NewTicker(ap.beaconConfig.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			ap.transmitBeacon()
		}
	}
}

func (ap *AccessPoint) runClientManagement(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			ap.updateClientConnections()
		}
	}
}

func (ap *AccessPoint) runChannelOptimization(ctx context.Context) {
	ticker := time.NewTicker(2 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			ap.optimizeChannel()
		}
	}
}

func (ap *AccessPoint) runSecurityMonitoring(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			ap.monitorSecurity()
		}
	}
}

func (ap *AccessPoint) runBandwidthMonitoring(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			ap.updateBandwidthUsage()
		}
	}
}

func (ap *AccessPoint) transmitBeacon() {
	ap.mu.Lock()
	defer ap.mu.Unlock()

	ap.beaconConfig.BeaconCount++
	ap.beaconConfig.LastBeacon = time.Now()

	for _, network := range ap.wifiConfig.Networks {
		if !network.Hidden {
			// ap.Logger.Debugf("Transmitted beacon for SSID: %s", network.SSID)
		}
	}
}

func (ap *AccessPoint) updateClientConnections() {
	ap.mu.Lock()
	defer ap.mu.Unlock()

	totalConnected := len(ap.connectedClients)
	maxClients := ap.getTotalMaxClients()

	if totalConnected < maxClients && ap.getRandomFloat(0, 100) < 20 {
		clientMAC := fmt.Sprintf("02:00:00:%02x:%02x:%02x",
			ap.getRandomInt(0, 255),
			ap.getRandomInt(0, 255),
			ap.getRandomInt(0, 255))

		band := "2.4GHz"
		if ap.getRandomBool() {
			band = "5GHz"
		}

		client := &ConnectedClient{
			MAC:          clientMAC,
			IP:           fmt.Sprintf("192.168.1.%d", ap.getRandomInt(100, 200)),
			Interface:    band,
			RSSI:         int(ap.getRandomFloat(-70, -30)),
			ConnectTime:  time.Now(),
			BytesRx:      0,
			BytesTx:      0,
			LastActivity: time.Now(),
		}

		ap.connectedClients[clientMAC] = client
		ap.signalStrength[clientMAC] = float64(client.RSSI)

		// ap.Logger.Infof("Client connected: %s (%s)", clientMAC, band)
	}

	if ap.getRandomFloat(0, 100) < 5 && len(ap.connectedClients) > 0 {
		for mac := range ap.connectedClients {
			delete(ap.connectedClients, mac)
			delete(ap.signalStrength, mac)
			// ap.Logger.Infof("Client disconnected: %s", mac)
			break
		}
	}

	for mac, client := range ap.connectedClients {
		newRSSI := float64(client.RSSI) + ap.getRandomFloat(-5, 5)
		if newRSSI > -20 {
			newRSSI = -20
		}
		if newRSSI < -90 {
			newRSSI = -90
		}
		client.RSSI = int(newRSSI)
		ap.signalStrength[mac] = newRSSI

		client.BytesRx += int64(ap.getRandomInt(1000, 50000))
		client.BytesTx += int64(ap.getRandomInt(500, 25000))
		// PacketsRx and PacketsTx not available in ConnectedClient
	}
}

func (ap *AccessPoint) optimizeChannel() {
	ap.mu.Lock()
	defer ap.mu.Unlock()

	if !ap.wifiConfig.Channels.AutoSelect {
		return
	}

	currentUtilization := ap.channelStats.Utilization
	currentInterference := ap.channelStats.Interference

	if currentUtilization > 70 || currentInterference > 50 {
		newChannel := ap.findBestChannel()
		if newChannel != ap.channelStats.Channel {
			oldChannel := ap.channelStats.Channel
			ap.channelStats.Channel = newChannel
			ap.wifiConfig.Channels.Channel2G = newChannel

			// ap.Logger.Infof("Channel optimized: %d -> %d", oldChannel, newChannel)

			event := base.Event{
				EventType: "channel_change",
				Severity:  "info",
				Message:   fmt.Sprintf("Channel changed from %d to %d due to interference", oldChannel, newChannel),
				Extra: map[string]interface{}{
					"old_channel":  oldChannel,
					"new_channel":  newChannel,
					"utilization":  currentUtilization,
					"interference": currentInterference,
				},
			}

			ap.PublishEvent(event)
		}
	}
}

func (ap *AccessPoint) findBestChannel() int {
	channels := []int{1, 6, 11}
	bestChannel := channels[0]
	bestScore := 0.0

	for _, channel := range channels {
		interference := ap.getRandomFloat(0, 100)
		utilization := ap.getRandomFloat(0, 100)
		score := 100 - (interference + utilization)

		if score > bestScore {
			bestScore = score
			bestChannel = channel
		}
	}

	return bestChannel
}

func (ap *AccessPoint) monitorSecurity() {
	ap.mu.Lock()
	defer ap.mu.Unlock()

	if ap.getRandomFloat(0, 100) < 1 {
		event := base.Event{
			EventType: "security_alert",
			Severity:  "warning",
			Message:   "Suspicious connection attempt detected",
			Extra: map[string]interface{}{
				"attempt_count": ap.getRandomInt(3, 10),
				"blocked":       true,
			},
		}

		ap.PublishEvent(event)
	}

	if ap.getRandomFloat(0, 100) < 0.5 {
		event := base.Event{
			EventType: "deauth_attack",
			Severity:  "critical",
			Message:   "Deauthentication attack detected",
			Extra: map[string]interface{}{
				"affected_clients": ap.getRandomInt(1, 5),
				"mitigation":       "enabled",
			},
		}

		ap.PublishEvent(event)
	}
}

func (ap *AccessPoint) updateBandwidthUsage() {
	ap.mu.Lock()
	defer ap.mu.Unlock()

	var totalRx, totalTx int64
	for _, client := range ap.connectedClients {
		totalRx += client.BytesRx
		totalTx += client.BytesTx
	}

	ap.bandwidthUsage = BandwidthUsage{
		Upload:   float64(totalTx) / (1024 * 1024),
		Download: float64(totalRx) / (1024 * 1024),
		Total:    float64(totalRx+totalTx) / (1024 * 1024),
	}

	totalClients := len(ap.connectedClients)
	if totalClients > 0 {
		ap.channelStats.Utilization = ap.getRandomFloat(10, 90)
		ap.channelStats.NoiseLevel = ap.getRandomFloat(-95, -75)
		ap.channelStats.Interference = ap.getRandomFloat(0, 30)
		ap.channelStats.Efficiency = 100 - ap.channelStats.Interference
		ap.channelStats.RetryRate = ap.getRandomFloat(0, 10)
	} else {
		ap.channelStats.Utilization = 0
		ap.channelStats.RetryRate = 0
	}
}

func (ap *AccessPoint) getTotalMaxClients() int {
	total := 0
	for _, network := range ap.wifiConfig.Networks {
		total += network.MaxClients
	}
	return total
}

func (ap *AccessPoint) GenerateStatePayload() base.StatePayload {
	ap.mu.RLock()
	defer ap.mu.RUnlock()

	baseState := ap.BaseDevice.GenerateStatePayload()

	if baseState.Extra == nil {
		baseState.Extra = make(map[string]interface{})
	}

	baseState.Extra["wifi"] = map[string]interface{}{
		"ssid_count":        len(ap.wifiConfig.Networks),
		"connected_clients": len(ap.connectedClients),
		"max_clients":       ap.getTotalMaxClients(),
		"current_channel":   ap.channelStats.Channel,
		"beacon_count":      ap.beaconConfig.BeaconCount,
	}

	baseState.Extra["security"] = map[string]interface{}{
		"encryption":    ap.wifiConfig.Security.Type,
		"wps_enabled":   ap.securityConfig.WPSEnabled,
		"mac_filtering": ap.securityConfig.MACFiltering,
		"guest_network": ap.securityConfig.GuestNetwork,
	}

	return baseState
}

func (ap *AccessPoint) GenerateTelemetryData() map[string]base.TelemetryPayload {
	ap.mu.RLock()
	defer ap.mu.RUnlock()

	telemetry := ap.BaseDevice.GenerateTelemetryData()

	telemetry["connected_clients"] = base.TelemetryPayload{
		"timestamp": time.Now(),
		"value":     float64(len(ap.connectedClients)),
		"unit":      "count",
		"tags": map[string]string{
			"metric": "connected_clients",
		},
	}

	telemetry["channel_utilization"] = base.TelemetryPayload{
		"timestamp": time.Now(),
		"value":     ap.channelStats.Utilization,
		"unit":      "percent",
		"tags": map[string]string{
			"metric": "channel_utilization",
		},
	}

	telemetry["signal_quality"] = base.TelemetryPayload{
		"timestamp": time.Now(),
		"value":     ap.getAverageSignalStrength(),
		"unit":      "dbm",
		"tags": map[string]string{
			"metric": "signal_quality",
		},
	}

	telemetry["interference_level"] = base.TelemetryPayload{
		"timestamp": time.Now(),
		"value":     ap.channelStats.Interference,
		"unit":      "percent",
		"tags": map[string]string{
			"metric": "interference_level",
		},
	}

	telemetry["bandwidth_utilization"] = base.TelemetryPayload{
		"timestamp": time.Now(),
		"value":     ap.bandwidthUsage.Total,
		"unit":      "mbps",
		"tags": map[string]string{
			"metric": "bandwidth_utilization",
		},
	}

	return telemetry
}

func (ap *AccessPoint) getAverageSignalStrength() float64 {
	if len(ap.signalStrength) == 0 {
		return 0
	}

	total := 0.0
	for _, strength := range ap.signalStrength {
		total += strength
	}

	return total / float64(len(ap.signalStrength))
}

func (ap *AccessPoint) HandleCommand(cmd base.Command) error {
	// Handle command: cmd.Action

	switch cmd.Action {
	case "get_wifi_status":
		return ap.sendWiFiStatus(cmd)
	case "set_wifi_config":
		return ap.setWiFiConfig(cmd)
	case "get_client_list":
		return ap.sendClientList(cmd)
	case "disconnect_client":
		return ap.disconnectClient(cmd)
	case "change_channel":
		return ap.changeChannel(cmd)
	case "enable_guest_network":
		return ap.enableGuestNetwork(cmd)
	case "update_security":
		return ap.updateSecurity(cmd)
	default:
		return ap.BaseDevice.HandleCommand(cmd)
	}
}

func (ap *AccessPoint) sendWiFiStatus(cmd base.Command) error {
	ap.mu.RLock()
	wifiData, _ := json.Marshal(ap.wifiConfig)
	ap.mu.RUnlock()

	response := base.CommandResponse{
		CommandID: cmd.ID,
		Status:    "success",
		Data:      string(wifiData),
	}

	return ap.PublishCommandResponse(response)
}

func (ap *AccessPoint) setWiFiConfig(cmd base.Command) error {
	var wifiConfig WiFiConfig
	if err := json.Unmarshal([]byte(cmd.Payload), &wifiConfig); err != nil {
		return fmt.Errorf("invalid WiFi config: %v", err)
	}

	ap.mu.Lock()
	ap.wifiConfig = wifiConfig
	ap.mu.Unlock()

	response := base.CommandResponse{
		CommandID: cmd.ID,
		Status:    "success",
		Data:      "WiFi configuration updated",
	}

	return ap.PublishCommandResponse(response)
}

func (ap *AccessPoint) sendClientList(cmd base.Command) error {
	ap.mu.RLock()
	clientData, _ := json.Marshal(ap.connectedClients)
	ap.mu.RUnlock()

	response := base.CommandResponse{
		CommandID: cmd.ID,
		Status:    "success",
		Data:      string(clientData),
	}

	return ap.PublishCommandResponse(response)
}

func (ap *AccessPoint) disconnectClient(cmd base.Command) error {
	var clientDisconnect struct {
		MACAddress string `json:"mac_address"`
	}

	if err := json.Unmarshal([]byte(cmd.Payload), &clientDisconnect); err != nil {
		return fmt.Errorf("invalid disconnect request: %v", err)
	}

	ap.mu.Lock()
	defer ap.mu.Unlock()

	if _, exists := ap.connectedClients[clientDisconnect.MACAddress]; !exists {
		return fmt.Errorf("client not found: %s", clientDisconnect.MACAddress)
	}

	delete(ap.connectedClients, clientDisconnect.MACAddress)
	delete(ap.signalStrength, clientDisconnect.MACAddress)

	response := base.CommandResponse{
		CommandID: cmd.ID,
		Status:    "success",
		Data:      fmt.Sprintf("Client %s disconnected", clientDisconnect.MACAddress),
	}

	return ap.PublishCommandResponse(response)
}

func (ap *AccessPoint) changeChannel(cmd base.Command) error {
	var channelChange struct {
		Channel int `json:"channel"`
	}

	if err := json.Unmarshal([]byte(cmd.Payload), &channelChange); err != nil {
		return fmt.Errorf("invalid channel change request: %v", err)
	}

	ap.mu.Lock()
	// oldChannel := ap.channelStats.Channel
	ap.channelStats.Channel = channelChange.Channel
	ap.wifiConfig.Channels.Channel2G = channelChange.Channel
	ap.mu.Unlock()

	// Channel changed from oldChannel to channelChange.Channel

	response := base.CommandResponse{
		CommandID: cmd.ID,
		Status:    "success",
		Data:      fmt.Sprintf("Channel changed to %d", channelChange.Channel),
	}

	return ap.PublishCommandResponse(response)
}

func (ap *AccessPoint) enableGuestNetwork(cmd base.Command) error {
	var guestConfig struct {
		Enabled  bool   `json:"enabled"`
		SSID     string `json:"ssid,omitempty"`
		Password string `json:"password,omitempty"`
	}

	if err := json.Unmarshal([]byte(cmd.Payload), &guestConfig); err != nil {
		return fmt.Errorf("invalid guest network config: %v", err)
	}

	ap.mu.Lock()
	ap.securityConfig.GuestNetwork = guestConfig.Enabled
	ap.mu.Unlock()

	response := base.CommandResponse{
		CommandID: cmd.ID,
		Status:    "success",
		Data:      fmt.Sprintf("Guest network %s", map[bool]string{true: "enabled", false: "disabled"}[guestConfig.Enabled]),
	}

	return ap.PublishCommandResponse(response)
}

func (ap *AccessPoint) updateSecurity(cmd base.Command) error {
	var securityUpdate struct {
		Type      string `json:"type,omitempty"`
		Password  string `json:"password,omitempty"`
		MACFilter *bool  `json:"mac_filter,omitempty"`
	}

	if err := json.Unmarshal([]byte(cmd.Payload), &securityUpdate); err != nil {
		return fmt.Errorf("invalid security update: %v", err)
	}

	ap.mu.Lock()
	defer ap.mu.Unlock()

	if securityUpdate.Type != "" {
		ap.wifiConfig.Security.Type = securityUpdate.Type
	}
	if securityUpdate.Password != "" {
		ap.wifiConfig.Security.Password = securityUpdate.Password
	}
	if securityUpdate.MACFilter != nil {
		ap.securityConfig.MACFiltering = *securityUpdate.MACFilter
	}

	response := base.CommandResponse{
		CommandID: cmd.ID,
		Status:    "success",
		Data:      "Security configuration updated",
	}

	return ap.PublishCommandResponse(response)
}

// Helper functions
func (ap *AccessPoint) getRandomFloat(min, max float64) float64 {
	return min + rand.Float64()*(max-min)
}

func (ap *AccessPoint) getRandomInt(min, max int) int {
	return min + rand.Intn(max-min+1)
}

func (ap *AccessPoint) getRandomBool() bool {
	return rand.Float64() < 0.5
}
