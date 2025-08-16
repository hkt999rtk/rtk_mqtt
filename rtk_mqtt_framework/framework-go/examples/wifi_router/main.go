package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rtk/mqtt-framework/pkg/codec"
	"github.com/rtk/mqtt-framework/pkg/config"
	"github.com/rtk/mqtt-framework/pkg/device"
	"github.com/rtk/mqtt-framework/pkg/mqtt"
	"github.com/rtk/mqtt-framework/pkg/topic"
)

// WiFiRouterPlugin implements a WiFi router device plugin
type WiFiRouterPlugin struct {
	*device.BasePlugin
	
	mqttClient   mqtt.Client
	codec        *codec.Codec
	topicBuilder *topic.Builder
	
	// Router configuration
	config          *RouterConfig
	connectedClients map[string]*ClientInfo
	networkStats    *NetworkStats
	
	// Channels for coordination
	stopChan      chan struct{}
	eventChan     chan *device.Event
	diagnosticsChan chan *DiagnosticEvent
}

// RouterConfig represents the router configuration
type RouterConfig struct {
	SSID              string        `json:"ssid"`
	Channel           int           `json:"channel"`
	MaxClients        int           `json:"max_clients"`
	DiagnosticInterval time.Duration `json:"diagnostic_interval"`
	EventReporting    bool          `json:"event_reporting"`
}

// ClientInfo represents information about a connected client
type ClientInfo struct {
	MAC           string    `json:"mac"`
	IP            string    `json:"ip"`
	Hostname      string    `json:"hostname"`
	ConnectedAt   time.Time `json:"connected_at"`
	RSSI          int       `json:"rssi"`
	TxBytes       uint64    `json:"tx_bytes"`
	RxBytes       uint64    `json:"rx_bytes"`
	LastActivity  time.Time `json:"last_activity"`
}

// NetworkStats represents network statistics
type NetworkStats struct {
	TotalClients    int       `json:"total_clients"`
	TotalTxBytes    uint64    `json:"total_tx_bytes"`
	TotalRxBytes    uint64    `json:"total_rx_bytes"`
	Uptime          time.Duration `json:"uptime"`
	LastUpdate      time.Time `json:"last_update"`
	ChannelUtilization float64 `json:"channel_utilization"`
}

// DiagnosticEvent represents a diagnostic event
type DiagnosticEvent struct {
	Type      string                 `json:"type"`
	Level     string                 `json:"level"`
	Message   string                 `json:"message"`
	Data      map[string]interface{} `json:"data"`
	Timestamp time.Time              `json:"timestamp"`
}

// NewWiFiRouterPlugin creates a new WiFi router plugin
func NewWiFiRouterPlugin() *WiFiRouterPlugin {
	info := &device.Info{
		Name:        "WiFi Router",
		Type:        "wifi_router",
		Version:     "1.0.0",
		Description: "WiFi router with diagnostic capabilities",
		Vendor:      "RTK",
	}

	plugin := &WiFiRouterPlugin{
		BasePlugin:       device.NewBasePlugin(info),
		connectedClients: make(map[string]*ClientInfo),
		networkStats:     &NetworkStats{},
		stopChan:         make(chan struct{}),
		eventChan:        make(chan *device.Event, 100),
		diagnosticsChan:  make(chan *DiagnosticEvent, 100),
	}

	// Set callbacks
	plugin.SetStartCallback(plugin.startRouter)
	plugin.SetStopCallback(plugin.stopRouter)
	plugin.SetCommandCallback(plugin.handleRouterCommand)
	plugin.SetTelemetryCallback(plugin.getRouterTelemetry)
	plugin.SetHealthCallback(plugin.getRouterHealth)

	return plugin
}

// Initialize initializes the router plugin
func (p *WiFiRouterPlugin) Initialize(ctx context.Context, configData json.RawMessage) error {
	if err := p.BasePlugin.Initialize(ctx, configData); err != nil {
		return err
	}

	// Parse router configuration
	var routerConfig RouterConfig
	if err := json.Unmarshal(configData, &routerConfig); err != nil {
		return fmt.Errorf("failed to parse router config: %w", err)
	}

	p.config = &routerConfig

	// Set default values
	if p.config.MaxClients == 0 {
		p.config.MaxClients = 50
	}
	if p.config.DiagnosticInterval == 0 {
		p.config.DiagnosticInterval = 60 * time.Second
	}

	p.GetLogger().WithField("config", p.config).Info("Router plugin initialized")
	return nil
}

// startRouter starts the router operations
func (p *WiFiRouterPlugin) startRouter(ctx context.Context) error {
	p.GetLogger().Info("Starting WiFi router")

	// Initialize network stats
	p.networkStats.LastUpdate = time.Now()

	// Start diagnostic monitoring
	go p.diagnosticLoop()
	
	// Start event processing
	go p.eventProcessingLoop()
	
	// Start client simulation
	go p.clientSimulationLoop()

	return nil
}

// stopRouter stops the router operations
func (p *WiFiRouterPlugin) stopRouter() error {
	p.GetLogger().Info("Stopping WiFi router")

	close(p.stopChan)
	return nil
}

// diagnosticLoop runs diagnostic checks
func (p *WiFiRouterPlugin) diagnosticLoop() {
	ticker := time.NewTicker(p.config.DiagnosticInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			p.runDiagnostics()
		case <-p.stopChan:
			return
		}
	}
}

// runDiagnostics performs various diagnostic checks
func (p *WiFiRouterPlugin) runDiagnostics() {
	now := time.Now()

	// Update network stats
	p.updateNetworkStats()

	// Check for roaming events
	if rand.Float64() < 0.1 { // 10% chance of roaming event
		p.simulateRoamingEvent()
	}

	// Check for connection failures
	if rand.Float64() < 0.05 { // 5% chance of connection failure
		p.simulateConnectionFailure()
	}

	// Check for ARP loss
	if rand.Float64() < 0.08 { // 8% chance of ARP loss
		p.simulateARPLoss()
	}

	// Update plugin state
	p.UpdateState(map[string]interface{}{
		"last_diagnostic_run": now,
		"connected_clients":   len(p.connectedClients),
		"network_stats":       p.networkStats,
		"config":              p.config,
	})
}

// updateNetworkStats updates network statistics
func (p *WiFiRouterPlugin) updateNetworkStats() {
	now := time.Now()
	p.networkStats.TotalClients = len(p.connectedClients)
	p.networkStats.ChannelUtilization = 20.0 + rand.Float64()*30.0 // 20-50%
	p.networkStats.LastUpdate = now
	
	// Update client activity
	for _, client := range p.connectedClients {
		client.TxBytes += uint64(rand.Intn(1000000)) // 0-1MB
		client.RxBytes += uint64(rand.Intn(500000))  // 0-500KB
		client.LastActivity = now
		client.RSSI = -30 - rand.Intn(40) // -30 to -70 dBm
	}
	
	// Calculate total bytes
	var totalTx, totalRx uint64
	for _, client := range p.connectedClients {
		totalTx += client.TxBytes
		totalRx += client.RxBytes
	}
	p.networkStats.TotalTxBytes = totalTx
	p.networkStats.TotalRxBytes = totalRx
}

// simulateRoamingEvent simulates a WiFi roaming event
func (p *WiFiRouterPlugin) simulateRoamingEvent() {
	if len(p.connectedClients) == 0 {
		return
	}

	// Pick a random client
	var clientMAC string
	for mac := range p.connectedClients {
		clientMAC = mac
		break
	}

	eventData := map[string]interface{}{
		"current_ap": map[string]interface{}{
			"bssid":     "aa:bb:cc:dd:ee:ff",
			"ssid":      p.config.SSID,
			"channel":   p.config.Channel,
			"frequency": 2400 + p.config.Channel*5,
			"rssi":      -45,
		},
		"target_ap": map[string]interface{}{
			"bssid":     "ff:ee:dd:cc:bb:aa",
			"ssid":      p.config.SSID,
			"channel":   p.config.Channel + 5,
			"frequency": 2400 + (p.config.Channel+5)*5,
			"rssi":      -40,
		},
		"roam_reason":    "better_ap",
		"roam_time_ms":   rand.Intn(500) + 100, // 100-600ms
		"client_mac":     clientMAC,
	}

	// Generate success or failure
	var level, message string
	if rand.Float64() < 0.8 { // 80% success rate
		level = "info"
		message = "WiFi roaming completed successfully"
	} else {
		level = "warning"
		message = "WiFi roaming failed"
		eventData["failure_reason"] = "target_ap_unreachable"
	}

	event := &device.Event{
		ID:        fmt.Sprintf("roam_%d", time.Now().UnixNano()),
		Type:      "wifi.roam_miss",
		Level:     level,
		Message:   message,
		Source:    "wifi_radio",
		Category:  "roaming",
		Data:      eventData,
		Timestamp: time.Now(),
		DeviceID:  p.GetInfo().Name,
		SchemaID:  "evt.wifi.roam_miss/1.0",
	}

	select {
	case p.eventChan <- event:
	default:
		p.GetLogger().Warn("Event channel full, dropping roaming event")
	}
}

// simulateConnectionFailure simulates a WiFi connection failure
func (p *WiFiRouterPlugin) simulateConnectionFailure() {
	eventData := map[string]interface{}{
		"ssid":            p.config.SSID,
		"bssid":           "aa:bb:cc:dd:ee:ff",
		"security_type":   "wpa2",
		"failure_stage":   []string{"association", "authentication", "dhcp"}[rand.Intn(3)],
		"error_code":      rand.Intn(1000),
		"error_message":   "Connection timeout",
		"retry_count":     rand.Intn(3),
		"signal_strength": -30 - rand.Intn(40),
	}

	event := &device.Event{
		ID:        fmt.Sprintf("conn_fail_%d", time.Now().UnixNano()),
		Type:      "wifi.connect_fail",
		Level:     "error",
		Message:   "WiFi connection failed",
		Source:    "wifi_radio",
		Category:  "connection",
		Data:      eventData,
		Timestamp: time.Now(),
		DeviceID:  p.GetInfo().Name,
		SchemaID:  "evt.wifi.connect_fail/1.0",
	}

	select {
	case p.eventChan <- event:
	default:
		p.GetLogger().Warn("Event channel full, dropping connection failure event")
	}
}

// simulateARPLoss simulates ARP packet loss
func (p *WiFiRouterPlugin) simulateARPLoss() {
	packetsSent := rand.Intn(50) + 10   // 10-60 packets
	packetsLost := rand.Intn(packetsSent/2) // Up to 50% loss
	lossPercentage := float64(packetsLost) / float64(packetsSent) * 100

	eventData := map[string]interface{}{
		"target_ip":         "192.168.1.1",
		"gateway_ip":        "192.168.1.1",
		"interface":         "wlan0",
		"packets_sent":      packetsSent,
		"packets_lost":      packetsLost,
		"loss_percentage":   lossPercentage,
		"test_duration_ms":  5000,
	}

	level := "info"
	if lossPercentage > 20 {
		level = "warning"
	}
	if lossPercentage > 50 {
		level = "error"
	}

	event := &device.Event{
		ID:        fmt.Sprintf("arp_loss_%d", time.Now().UnixNano()),
		Type:      "wifi.arp_loss",
		Level:     level,
		Message:   fmt.Sprintf("ARP packet loss detected: %.1f%%", lossPercentage),
		Source:    "network_monitor",
		Category:  "connectivity",
		Data:      eventData,
		Timestamp: time.Now(),
		DeviceID:  p.GetInfo().Name,
		SchemaID:  "evt.wifi.arp_loss/1.0",
	}

	select {
	case p.eventChan <- event:
	default:
		p.GetLogger().Warn("Event channel full, dropping ARP loss event")
	}
}

// clientSimulationLoop simulates client connections and disconnections
func (p *WiFiRouterPlugin) clientSimulationLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			p.simulateClientActivity()
		case <-p.stopChan:
			return
		}
	}
}

// simulateClientActivity simulates client connections and disconnections
func (p *WiFiRouterPlugin) simulateClientActivity() {
	// Random chance of new client connection
	if len(p.connectedClients) < p.config.MaxClients && rand.Float64() < 0.3 {
		p.addRandomClient()
	}

	// Random chance of client disconnection
	if len(p.connectedClients) > 0 && rand.Float64() < 0.2 {
		p.removeRandomClient()
	}
}

// addRandomClient adds a random client
func (p *WiFiRouterPlugin) addRandomClient() {
	mac := fmt.Sprintf("aa:bb:cc:dd:ee:%02x", rand.Intn(256))
	if _, exists := p.connectedClients[mac]; exists {
		return
	}

	client := &ClientInfo{
		MAC:           mac,
		IP:            fmt.Sprintf("192.168.1.%d", rand.Intn(254)+2),
		Hostname:      fmt.Sprintf("device-%d", rand.Intn(1000)),
		ConnectedAt:   time.Now(),
		RSSI:          -30 - rand.Intn(40),
		TxBytes:       0,
		RxBytes:       0,
		LastActivity:  time.Now(),
	}

	p.connectedClients[mac] = client
	p.GetLogger().WithField("client", client).Info("Client connected")
}

// removeRandomClient removes a random client
func (p *WiFiRouterPlugin) removeRandomClient() {
	for mac, client := range p.connectedClients {
		delete(p.connectedClients, mac)
		p.GetLogger().WithField("client", client).Info("Client disconnected")
		break
	}
}

// eventProcessingLoop processes events and publishes them
func (p *WiFiRouterPlugin) eventProcessingLoop() {
	for {
		select {
		case event := <-p.eventChan:
			p.publishEvent(event)
		case <-p.stopChan:
			return
		}
	}
}

// publishEvent publishes an event via MQTT
func (p *WiFiRouterPlugin) publishEvent(event *device.Event) {
	if p.mqttClient == nil || p.codec == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	msg, err := p.codec.EncodeEvent(ctx, event)
	if err != nil {
		p.GetLogger().WithError(err).Error("Failed to encode event")
		return
	}

	mqttMsg := &mqtt.Message{
		Topic:     msg.Topic,
		Payload:   msg.Payload,
		QoS:       mqtt.QoS(msg.QoS),
		Retained:  msg.Retained,
		Timestamp: time.Now(),
	}

	if err := p.mqttClient.PublishMessage(ctx, mqttMsg); err != nil {
		p.GetLogger().WithError(err).Error("Failed to publish event")
		return
	}

	p.GetLogger().WithField("event_type", event.Type).Debug("Event published")
}

// handleRouterCommand handles incoming commands
func (p *WiFiRouterPlugin) handleRouterCommand(ctx context.Context, cmd *device.Command) (*device.CommandResponse, error) {
	p.GetLogger().WithField("command", cmd.Action).Info("Handling command")

	switch cmd.Action {
	case "get_clients":
		return p.handleGetClients(cmd)
	case "get_stats":
		return p.handleGetStats(cmd)
	case "restart_radio":
		return p.handleRestartRadio(cmd)
	case "set_channel":
		return p.handleSetChannel(cmd)
	case "run_diagnostics":
		return p.handleRunDiagnostics(cmd)
	default:
		return &device.CommandResponse{
			CommandID: cmd.ID,
			Status:    "error",
			Error:     "Unknown command: " + cmd.Action,
			Timestamp: time.Now(),
		}, nil
	}
}

// handleGetClients handles get clients command
func (p *WiFiRouterPlugin) handleGetClients(cmd *device.Command) (*device.CommandResponse, error) {
	clients := make([]ClientInfo, 0, len(p.connectedClients))
	for _, client := range p.connectedClients {
		clients = append(clients, *client)
	}

	return &device.CommandResponse{
		CommandID: cmd.ID,
		Status:    "success",
		Result: map[string]interface{}{
			"client_count": len(clients),
			"clients":      clients,
		},
		Timestamp: time.Now(),
	}, nil
}

// handleGetStats handles get stats command
func (p *WiFiRouterPlugin) handleGetStats(cmd *device.Command) (*device.CommandResponse, error) {
	return &device.CommandResponse{
		CommandID: cmd.ID,
		Status:    "success",
		Result: map[string]interface{}{
			"network_stats": p.networkStats,
			"config":        p.config,
		},
		Timestamp: time.Now(),
	}, nil
}

// handleRestartRadio handles restart radio command
func (p *WiFiRouterPlugin) handleRestartRadio(cmd *device.Command) (*device.CommandResponse, error) {
	// Simulate radio restart by clearing all clients
	clientCount := len(p.connectedClients)
	p.connectedClients = make(map[string]*ClientInfo)

	p.GetLogger().Info("WiFi radio restarted")

	return &device.CommandResponse{
		CommandID: cmd.ID,
		Status:    "success",
		Result: map[string]interface{}{
			"message":               "WiFi radio restarted successfully",
			"disconnected_clients":  clientCount,
		},
		Timestamp: time.Now(),
	}, nil
}

// handleSetChannel handles set channel command
func (p *WiFiRouterPlugin) handleSetChannel(cmd *device.Command) (*device.CommandResponse, error) {
	channelFloat, ok := cmd.Params["channel"].(float64)
	if !ok {
		return &device.CommandResponse{
			CommandID: cmd.ID,
			Status:    "error",
			Error:     "Missing or invalid channel parameter",
			Timestamp: time.Now(),
		}, nil
	}

	channel := int(channelFloat)
	if channel < 1 || channel > 14 {
		return &device.CommandResponse{
			CommandID: cmd.ID,
			Status:    "error",
			Error:     "Channel must be between 1 and 14",
			Timestamp: time.Now(),
		}, nil
	}

	oldChannel := p.config.Channel
	p.config.Channel = channel

	return &device.CommandResponse{
		CommandID: cmd.ID,
		Status:    "success",
		Result: map[string]interface{}{
			"old_channel": oldChannel,
			"new_channel": channel,
		},
		Timestamp: time.Now(),
	}, nil
}

// handleRunDiagnostics handles run diagnostics command
func (p *WiFiRouterPlugin) handleRunDiagnostics(cmd *device.Command) (*device.CommandResponse, error) {
	// Run diagnostics immediately
	p.runDiagnostics()

	return &device.CommandResponse{
		CommandID: cmd.ID,
		Status:    "success",
		Result: map[string]interface{}{
			"message": "Diagnostics completed",
			"timestamp": time.Now(),
		},
		Timestamp: time.Now(),
	}, nil
}

// getRouterTelemetry returns telemetry for a specific metric
func (p *WiFiRouterPlugin) getRouterTelemetry(ctx context.Context, metric string) (*device.TelemetryData, error) {
	switch metric {
	case "client_count":
		return &device.TelemetryData{
			Metric:    metric,
			Value:     len(p.connectedClients),
			Unit:      "count",
			Timestamp: time.Now(),
		}, nil
	case "channel_utilization":
		return &device.TelemetryData{
			Metric:    metric,
			Value:     p.networkStats.ChannelUtilization,
			Unit:      "percent",
			Timestamp: time.Now(),
		}, nil
	case "total_tx_bytes":
		return &device.TelemetryData{
			Metric:    metric,
			Value:     p.networkStats.TotalTxBytes,
			Unit:      "bytes",
			Timestamp: time.Now(),
		}, nil
	case "total_rx_bytes":
		return &device.TelemetryData{
			Metric:    metric,
			Value:     p.networkStats.TotalRxBytes,
			Unit:      "bytes",
			Timestamp: time.Now(),
		}, nil
	default:
		return nil, fmt.Errorf("metric not found: %s", metric)
	}
}

// getRouterHealth returns router health status
func (p *WiFiRouterPlugin) getRouterHealth(ctx context.Context) (*device.Health, error) {
	health := &device.Health{
		Status:      "healthy",
		Score:       1.0,
		Checks:      make(map[string]device.HealthCheck),
		LastCheck:   time.Now(),
		Diagnostics: make(map[string]interface{}),
	}

	// Check client count
	clientCount := len(p.connectedClients)
	if clientCount > int(float64(p.config.MaxClients)*0.9) {
		health.Status = "warning"
		health.Score = 0.7
	}

	health.Checks["client_capacity"] = device.HealthCheck{
		Name:        "Client Capacity",
		Status:      health.Status,
		Value:       fmt.Sprintf("%d/%d", clientCount, p.config.MaxClients),
		Message:     fmt.Sprintf("Client utilization: %.1f%%", float64(clientCount)/float64(p.config.MaxClients)*100),
		LastChecked: time.Now(),
	}

	// Check channel utilization
	if p.networkStats.ChannelUtilization > 80 {
		health.Status = "warning"
		if health.Score > 0.6 {
			health.Score = 0.6
		}
	}

	channelCheck := device.HealthCheck{
		Name:        "Channel Utilization",
		Status:      "healthy",
		Value:       fmt.Sprintf("%.1f%%", p.networkStats.ChannelUtilization),
		Message:     "Channel utilization within normal range",
		LastChecked: time.Now(),
	}

	if p.networkStats.ChannelUtilization > 80 {
		channelCheck.Status = "warning"
		channelCheck.Message = "High channel utilization"
	}

	health.Checks["channel_utilization"] = channelCheck

	// Add diagnostics
	health.Diagnostics["connected_clients"] = clientCount
	health.Diagnostics["channel"] = p.config.Channel
	health.Diagnostics["ssid"] = p.config.SSID
	health.Diagnostics["uptime"] = time.Since(p.networkStats.LastUpdate).String()

	return health, nil
}

// SetMQTTClient sets the MQTT client for the plugin
func (p *WiFiRouterPlugin) SetMQTTClient(client mqtt.Client) {
	p.mqttClient = client
}

// SetCodec sets the message codec for the plugin
func (p *WiFiRouterPlugin) SetCodec(codec *codec.Codec) {
	p.codec = codec
}

func main() {
	// Load configuration
	cfg, err := config.LoadFromFile("config.yaml")
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Create MQTT client
	mqttClient, err := mqtt.NewClient(&cfg.MQTT)
	if err != nil {
		log.Fatalf("Failed to create MQTT client: %v", err)
	}

	// Create codec
	codec := codec.NewCodec()
	codec.SetDeviceInfo(cfg.Device.Tenant, cfg.Device.Site, cfg.Device.DeviceID)

	// Create topic builder
	topicBuilder := topic.NewBuilder(cfg.Device.Tenant, cfg.Device.Site, cfg.Device.DeviceID)

	// Create router plugin
	plugin := NewWiFiRouterPlugin()
	plugin.SetMQTTClient(mqttClient)
	plugin.SetCodec(codec)

	// Load plugin configuration
	pluginConfigData := json.RawMessage(`{
		"ssid": "RTK_Demo_Network",
		"channel": 6,
		"max_clients": 50,
		"diagnostic_interval": "60s",
		"event_reporting": true
	}`)

	// Initialize plugin
	ctx := context.Background()
	if err := plugin.Initialize(ctx, pluginConfigData); err != nil {
		log.Fatalf("Failed to initialize plugin: %v", err)
	}

	// Connect to MQTT broker
	if err := mqttClient.Connect(ctx); err != nil {
		log.Fatalf("Failed to connect to MQTT broker: %v", err)
	}
	defer mqttClient.Disconnect()

	log.Println("Connected to MQTT broker")

	// Subscribe to command topics
	commandTopic := topicBuilder.CommandRequest()
	err = mqttClient.Subscribe(ctx, commandTopic, func(ctx context.Context, msg *mqtt.Message) error {
		// Decode command
		rtkMsg, err := codec.Decode(ctx, msg.Topic, msg.Payload)
		if err != nil {
			log.Printf("Failed to decode message: %v", err)
			return err
		}

		cmd, err := codec.DecodeCommand(ctx, rtkMsg)
		if err != nil {
			log.Printf("Failed to decode command: %v", err)
			return err
		}

		// Handle command
		response, err := plugin.HandleCommand(ctx, cmd)
		if err != nil {
			log.Printf("Command handling failed: %v", err)
			return err
		}

		// Publish response
		responseMsg, err := codec.EncodeCommandResponse(ctx, response)
		if err != nil {
			log.Printf("Failed to encode response: %v", err)
			return err
		}

		mqttResponse := &mqtt.Message{
			Topic:     responseMsg.Topic,
			Payload:   responseMsg.Payload,
			QoS:       mqtt.QoS(responseMsg.QoS),
			Retained:  responseMsg.Retained,
			Timestamp: time.Now(),
		}

		return mqttClient.PublishMessage(ctx, mqttResponse)
	}, nil)

	if err != nil {
		log.Fatalf("Failed to subscribe to commands: %v", err)
	}

	log.Printf("Subscribed to commands on: %s", commandTopic)

	// Start plugin
	if err := plugin.Start(ctx); err != nil {
		log.Fatalf("Failed to start plugin: %v", err)
	}

	log.Println("WiFi router plugin started")

	// Publish initial state
	state, err := plugin.GetState(ctx)
	if err == nil {
		stateMsg, err := codec.EncodeState(ctx, state)
		if err == nil {
			mqttState := &mqtt.Message{
				Topic:     stateMsg.Topic,
				Payload:   stateMsg.Payload,
				QoS:       mqtt.QoS(stateMsg.QoS),
				Retained:  stateMsg.Retained,
				Timestamp: time.Now(),
			}
			mqttClient.PublishMessage(ctx, mqttState)
		}
	}

	// Set up signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	log.Println("WiFi router is running. Press Ctrl+C to stop.")
	<-sigChan

	log.Println("Shutting down...")

	// Stop plugin
	if err := plugin.Stop(); err != nil {
		log.Printf("Error stopping plugin: %v", err)
	}

	log.Println("WiFi router stopped")
}