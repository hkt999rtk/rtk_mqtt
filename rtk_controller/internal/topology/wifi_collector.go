package topology

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"rtk_controller/internal/storage"
)

// WiFiClientCollector manages WiFi client state collection and analysis
type WiFiClientCollector struct {
	// Storage
	storage         *storage.TopologyStorage
	identityStorage *storage.IdentityStorage
	
	// WiFi client tracking
	clients    map[string]*WiFiClientState
	clientsMu  sync.RWMutex
	
	// Access point tracking
	accessPoints map[string]*AccessPointState
	apMu        sync.RWMutex
	
	// Configuration
	config WiFiCollectorConfig
	
	// Background processing
	running bool
	cancel  context.CancelFunc
	
	// Statistics
	stats WiFiCollectorStats
}

// WiFiClientState tracks the state of a WiFi client
type WiFiClientState struct {
	MacAddress      string
	FriendlyName    string
	CurrentAP       string
	CurrentSSID     string
	LastSeen        time.Time
	ConnectionTime  time.Time
	SignalHistory   []SignalMeasurement
	ConnectionCount int64
	TotalBytes      int64
	IsRoaming       bool
	RoamingHistory  []RoamingEvent
}

// AccessPointState tracks the state of an access point
type AccessPointState struct {
	DeviceID        string
	SSID           string
	BSSID          string
	Channel        int
	Band           string
	MaxClients     int
	ConnectedClients map[string]*WiFiClientInfo
	LastUpdate     time.Time
	SignalQuality  AccessPointQuality
}

// WiFiClientInfo holds detailed information about a connected client
type WiFiClientInfo struct {
	MacAddress     string
	IPAddress      string
	Hostname       string
	ConnectedAt    time.Time
	LastSeen       time.Time
	RSSI          int
	TxRate        int
	RxRate        int
	TxBytes       int64
	RxBytes       int64
	TxPackets     int64
	RxPackets     int64
	SignalStrength int
	NoiseLevel    int
	ConnectionTime int64
	Capabilities  []string
}

// SignalMeasurement represents a signal strength measurement
type SignalMeasurement struct {
	Timestamp    time.Time
	RSSI         int
	NoiseLevel   int
	TxRate       int
	RxRate       int
	APDeviceID   string
}

// RoamingEvent represents a client roaming between access points
type RoamingEvent struct {
	Timestamp     time.Time
	FromAP        string
	ToAP          string
	FromSSID      string
	ToSSID        string
	Reason        string
	Duration      time.Duration
	SignalBefore  int
	SignalAfter   int
}

// AccessPointQuality holds AP quality metrics
type AccessPointQuality struct {
	AverageRSSI      int
	ClientCount      int
	ChannelUtilization float64
	Interference     int
	QualityScore     float64
}

// WiFiCollectorConfig holds WiFi collector configuration
type WiFiCollectorConfig struct {
	// Collection intervals
	ClientUpdateInterval    time.Duration
	SignalSampleInterval   time.Duration
	QualityCheckInterval   time.Duration
	
	// Data retention
	SignalHistoryRetention time.Duration
	ClientOfflineTimeout   time.Duration
	RoamingHistoryRetention time.Duration
	
	// Analysis settings
	EnableRoamingDetection bool
	RoamingTimeThreshold   time.Duration
	WeakSignalThreshold    int
	QualityUpdateThreshold float64
	
	// Performance settings
	MaxClientsPerAP        int
	MaxSignalSamples       int
	BatchSize              int
}

// WiFiCollectorStats holds collection statistics
type WiFiCollectorStats struct {
	TotalClientsTracked    int64
	ActiveClients         int64
	TotalAccessPoints     int64
	ActiveAccessPoints    int64
	RoamingEventsDetected int64
	SignalSamplesCollected int64
	LastUpdate            time.Time
	ProcessingErrors      int64
}

// NewWiFiClientCollector creates a new WiFi client collector
func NewWiFiClientCollector(
	storage *storage.TopologyStorage,
	identityStorage *storage.IdentityStorage,
	config WiFiCollectorConfig,
) *WiFiClientCollector {
	return &WiFiClientCollector{
		storage:         storage,
		identityStorage: identityStorage,
		clients:        make(map[string]*WiFiClientState),
		accessPoints:   make(map[string]*AccessPointState),
		config:         config,
		stats:          WiFiCollectorStats{},
	}
}

// Start begins WiFi client collection
func (wc *WiFiClientCollector) Start() error {
	wc.clientsMu.Lock()
	defer wc.clientsMu.Unlock()
	
	if wc.running {
		return fmt.Errorf("WiFi client collector is already running")
	}
	
	ctx, cancel := context.WithCancel(context.Background())
	wc.cancel = cancel
	wc.running = true
	
	log.Printf("Starting WiFi client collector")
	
	// Start background processing goroutines
	go wc.clientTrackingLoop(ctx)
	go wc.signalAnalysisLoop(ctx)
	go wc.qualityMonitoringLoop(ctx)
	go wc.cleanupLoop(ctx)
	
	// Load existing data
	if err := wc.loadExistingData(); err != nil {
		log.Printf("Failed to load existing WiFi data: %v", err)
	}
	
	return nil
}

// Stop stops WiFi client collection
func (wc *WiFiClientCollector) Stop() error {
	wc.clientsMu.Lock()
	defer wc.clientsMu.Unlock()
	
	if !wc.running {
		return fmt.Errorf("WiFi client collector is not running")
	}
	
	wc.cancel()
	wc.running = false
	
	log.Printf("WiFi client collector stopped")
	return nil
}

// ProcessWiFiClientsMessage processes incoming WiFi clients telemetry
func (wc *WiFiClientCollector) ProcessWiFiClientsMessage(
	deviceID string,
	interfaceName string,
	apInfo map[string]interface{},
	clients []map[string]interface{},
	timestamp int64,
) error {
	
	// Update access point state
	if err := wc.updateAccessPointState(deviceID, interfaceName, apInfo, timestamp); err != nil {
		log.Printf("Failed to update AP state: %v", err)
		wc.stats.ProcessingErrors++
	}
	
	// Process each client
	for _, clientData := range clients {
		if err := wc.processClientData(deviceID, clientData, timestamp); err != nil {
			log.Printf("Failed to process client data: %v", err)
			wc.stats.ProcessingErrors++
			continue
		}
	}
	
	wc.stats.LastUpdate = time.Now()
	return nil
}

// GetWiFiClientState returns current state of a WiFi client
func (wc *WiFiClientCollector) GetWiFiClientState(macAddress string) (*WiFiClientState, bool) {
	wc.clientsMu.RLock()
	defer wc.clientsMu.RUnlock()
	
	client, exists := wc.clients[macAddress]
	if !exists {
		return nil, false
	}
	
	// Return copy
	clientCopy := *client
	return &clientCopy, true
}

// GetAccessPointState returns current state of an access point
func (wc *WiFiClientCollector) GetAccessPointState(deviceID string) (*AccessPointState, bool) {
	wc.apMu.RLock()
	defer wc.apMu.RUnlock()
	
	ap, exists := wc.accessPoints[deviceID]
	if !exists {
		return nil, false
	}
	
	// Return copy
	apCopy := *ap
	apCopy.ConnectedClients = make(map[string]*WiFiClientInfo)
	for mac, client := range ap.ConnectedClients {
		clientCopy := *client
		apCopy.ConnectedClients[mac] = &clientCopy
	}
	
	return &apCopy, true
}

// GetActiveClients returns all currently active WiFi clients
func (wc *WiFiClientCollector) GetActiveClients() map[string]*WiFiClientState {
	wc.clientsMu.RLock()
	defer wc.clientsMu.RUnlock()
	
	activeClients := make(map[string]*WiFiClientState)
	now := time.Now()
	
	for mac, client := range wc.clients {
		if now.Sub(client.LastSeen) < wc.config.ClientOfflineTimeout {
			clientCopy := *client
			activeClients[mac] = &clientCopy
		}
	}
	
	return activeClients
}

// GetRoamingEvents returns recent roaming events
func (wc *WiFiClientCollector) GetRoamingEvents(since time.Time) []RoamingEvent {
	wc.clientsMu.RLock()
	defer wc.clientsMu.RUnlock()
	
	var events []RoamingEvent
	
	for _, client := range wc.clients {
		for _, event := range client.RoamingHistory {
			if event.Timestamp.After(since) {
				events = append(events, event)
			}
		}
	}
	
	return events
}

// GetStats returns collector statistics
func (wc *WiFiClientCollector) GetStats() WiFiCollectorStats {
	wc.clientsMu.RLock()
	wc.apMu.RLock()
	defer wc.clientsMu.RUnlock()
	defer wc.apMu.RUnlock()
	
	stats := wc.stats
	stats.TotalClientsTracked = int64(len(wc.clients))
	stats.TotalAccessPoints = int64(len(wc.accessPoints))
	
	// Count active clients and APs
	now := time.Now()
	var activeClients, activeAPs int64
	
	for _, client := range wc.clients {
		if now.Sub(client.LastSeen) < wc.config.ClientOfflineTimeout {
			activeClients++
		}
	}
	
	for _, ap := range wc.accessPoints {
		if now.Sub(ap.LastUpdate) < wc.config.ClientOfflineTimeout {
			activeAPs++
		}
	}
	
	stats.ActiveClients = activeClients
	stats.ActiveAccessPoints = activeAPs
	
	return stats
}

// Private methods

func (wc *WiFiClientCollector) updateAccessPointState(
	deviceID string,
	interfaceName string,
	apInfo map[string]interface{},
	timestamp int64,
) error {
	
	wc.apMu.Lock()
	defer wc.apMu.Unlock()
	
	ap, exists := wc.accessPoints[deviceID]
	if !exists {
		ap = &AccessPointState{
			DeviceID:         deviceID,
			ConnectedClients: make(map[string]*WiFiClientInfo),
		}
		wc.accessPoints[deviceID] = ap
	}
	
	// Update AP information
	if ssid, ok := apInfo["ssid"].(string); ok {
		ap.SSID = ssid
	}
	if bssid, ok := apInfo["bssid"].(string); ok {
		ap.BSSID = bssid
	}
	if channel, ok := apInfo["channel"].(float64); ok {
		ap.Channel = int(channel)
	}
	if band, ok := apInfo["band"].(string); ok {
		ap.Band = band
	}
	if maxClients, ok := apInfo["max_clients"].(float64); ok {
		ap.MaxClients = int(maxClients)
	}
	
	ap.LastUpdate = time.UnixMilli(timestamp)
	
	return nil
}

func (wc *WiFiClientCollector) processClientData(
	apDeviceID string,
	clientData map[string]interface{},
	timestamp int64,
) error {
	
	macAddress, ok := clientData["mac_address"].(string)
	if !ok {
		return fmt.Errorf("missing or invalid mac_address")
	}
	
	wc.clientsMu.Lock()
	defer wc.clientsMu.Unlock()
	
	// Get or create client state
	client, exists := wc.clients[macAddress]
	if !exists {
		client = &WiFiClientState{
			MacAddress:     macAddress,
			SignalHistory:  []SignalMeasurement{},
			RoamingHistory: []RoamingEvent{},
		}
		wc.clients[macAddress] = client
		wc.stats.TotalClientsTracked++
	}
	
	// Get friendly name from identity storage
	if identity, err := wc.identityStorage.GetDeviceIdentity(macAddress); err == nil {
		client.FriendlyName = identity.FriendlyName
	}
	
	// Check for roaming
	previousAP := client.CurrentAP
	currentAP := apDeviceID
	
	if previousAP != "" && previousAP != currentAP && wc.config.EnableRoamingDetection {
		wc.detectRoaming(client, previousAP, currentAP, timestamp)
	}
	
	// Update client state
	client.CurrentAP = currentAP
	client.LastSeen = time.UnixMilli(timestamp)
	
	if connectedAt, ok := clientData["connected_at"].(float64); ok {
		client.ConnectionTime = time.UnixMilli(int64(connectedAt))
	}
	
	// Update signal measurement
	measurement := SignalMeasurement{
		Timestamp:  time.UnixMilli(timestamp),
		APDeviceID: apDeviceID,
	}
	
	if rssi, ok := clientData["rssi"].(float64); ok {
		measurement.RSSI = int(rssi)
	}
	if noiseLevel, ok := clientData["noise_level"].(float64); ok {
		measurement.NoiseLevel = int(noiseLevel)
	}
	if txRate, ok := clientData["tx_rate"].(float64); ok {
		measurement.TxRate = int(txRate)
	}
	if rxRate, ok := clientData["rx_rate"].(float64); ok {
		measurement.RxRate = int(rxRate)
	}
	
	// Add to signal history
	client.SignalHistory = append(client.SignalHistory, measurement)
	
	// Limit signal history size
	if len(client.SignalHistory) > wc.config.MaxSignalSamples {
		client.SignalHistory = client.SignalHistory[len(client.SignalHistory)-wc.config.MaxSignalSamples:]
	}
	
	wc.stats.SignalSamplesCollected++
	
	// Update AP client info
	wc.updateAPClientInfo(apDeviceID, macAddress, clientData, timestamp)
	
	return nil
}

func (wc *WiFiClientCollector) updateAPClientInfo(
	apDeviceID string,
	macAddress string,
	clientData map[string]interface{},
	timestamp int64,
) {
	
	wc.apMu.Lock()
	defer wc.apMu.Unlock()
	
	ap, exists := wc.accessPoints[apDeviceID]
	if !exists {
		return
	}
	
	clientInfo := &WiFiClientInfo{
		MacAddress: macAddress,
		LastSeen:   time.UnixMilli(timestamp),
	}
	
	// Extract client information
	if ipAddress, ok := clientData["ip_address"].(string); ok {
		clientInfo.IPAddress = ipAddress
	}
	if hostname, ok := clientData["hostname"].(string); ok {
		clientInfo.Hostname = hostname
	}
	if connectedAt, ok := clientData["connected_at"].(float64); ok {
		clientInfo.ConnectedAt = time.UnixMilli(int64(connectedAt))
	}
	if rssi, ok := clientData["rssi"].(float64); ok {
		clientInfo.RSSI = int(rssi)
	}
	if txRate, ok := clientData["tx_rate"].(float64); ok {
		clientInfo.TxRate = int(txRate)
	}
	if rxRate, ok := clientData["rx_rate"].(float64); ok {
		clientInfo.RxRate = int(rxRate)
	}
	if txBytes, ok := clientData["tx_bytes"].(float64); ok {
		clientInfo.TxBytes = int64(txBytes)
	}
	if rxBytes, ok := clientData["rx_bytes"].(float64); ok {
		clientInfo.RxBytes = int64(rxBytes)
	}
	if txPackets, ok := clientData["tx_packets"].(float64); ok {
		clientInfo.TxPackets = int64(txPackets)
	}
	if rxPackets, ok := clientData["rx_packets"].(float64); ok {
		clientInfo.RxPackets = int64(rxPackets)
	}
	if signalStrength, ok := clientData["signal_strength"].(float64); ok {
		clientInfo.SignalStrength = int(signalStrength)
	}
	if noiseLevel, ok := clientData["noise_level"].(float64); ok {
		clientInfo.NoiseLevel = int(noiseLevel)
	}
	if connectionTime, ok := clientData["connection_time"].(float64); ok {
		clientInfo.ConnectionTime = int64(connectionTime)
	}
	if capabilities, ok := clientData["capabilities"].([]interface{}); ok {
		for _, cap := range capabilities {
			if capStr, ok := cap.(string); ok {
				clientInfo.Capabilities = append(clientInfo.Capabilities, capStr)
			}
		}
	}
	
	ap.ConnectedClients[macAddress] = clientInfo
}

func (wc *WiFiClientCollector) detectRoaming(
	client *WiFiClientState,
	fromAP string,
	toAP string,
	timestamp int64,
) {
	
	event := RoamingEvent{
		Timestamp: time.UnixMilli(timestamp),
		FromAP:    fromAP,
		ToAP:      toAP,
		Reason:    "automatic",
	}
	
	// Calculate roaming duration
	if !client.ConnectionTime.IsZero() {
		event.Duration = time.UnixMilli(timestamp).Sub(client.ConnectionTime)
	}
	
	// Get signal strength before and after
	if len(client.SignalHistory) > 0 {
		lastMeasurement := client.SignalHistory[len(client.SignalHistory)-1]
		event.SignalBefore = lastMeasurement.RSSI
	}
	
	// Add to roaming history
	client.RoamingHistory = append(client.RoamingHistory, event)
	client.IsRoaming = true
	
	// Limit roaming history size
	maxEvents := 100
	if len(client.RoamingHistory) > maxEvents {
		client.RoamingHistory = client.RoamingHistory[len(client.RoamingHistory)-maxEvents:]
	}
	
	wc.stats.RoamingEventsDetected++
	
	log.Printf("Roaming detected: client %s moved from AP %s to AP %s", 
		client.MacAddress, fromAP, toAP)
}

func (wc *WiFiClientCollector) clientTrackingLoop(ctx context.Context) {
	ticker := time.NewTicker(wc.config.ClientUpdateInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			wc.updateClientStates()
		}
	}
}

func (wc *WiFiClientCollector) signalAnalysisLoop(ctx context.Context) {
	ticker := time.NewTicker(wc.config.SignalSampleInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			wc.analyzeSignalQuality()
		}
	}
}

func (wc *WiFiClientCollector) qualityMonitoringLoop(ctx context.Context) {
	ticker := time.NewTicker(wc.config.QualityCheckInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			wc.updateAccessPointQuality()
		}
	}
}

func (wc *WiFiClientCollector) cleanupLoop(ctx context.Context) {
	ticker := time.NewTicker(time.Hour) // Run cleanup every hour
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			wc.cleanupOldData()
		}
	}
}

func (wc *WiFiClientCollector) updateClientStates() {
	// Update client connection status and detect disconnections
	now := time.Now()
	
	wc.clientsMu.Lock()
	defer wc.clientsMu.Unlock()
	
	for mac, client := range wc.clients {
		// Mark as not roaming if enough time has passed
		if client.IsRoaming && now.Sub(client.LastSeen) > wc.config.RoamingTimeThreshold {
			client.IsRoaming = false
		}
		
		// Remove from AP client lists if offline
		if now.Sub(client.LastSeen) > wc.config.ClientOfflineTimeout {
			wc.removeClientFromAPs(mac)
		}
	}
}

func (wc *WiFiClientCollector) analyzeSignalQuality() {
	// Analyze signal quality trends and detect issues
	wc.clientsMu.RLock()
	defer wc.clientsMu.RUnlock()
	
	for _, client := range wc.clients {
		if len(client.SignalHistory) < 3 {
			continue
		}
		
		// Check for weak signal
		recentMeasurements := client.SignalHistory[len(client.SignalHistory)-3:]
		totalRSSI := 0
		for _, measurement := range recentMeasurements {
			totalRSSI += measurement.RSSI
		}
		avgRSSI := totalRSSI / len(recentMeasurements)
		
		if avgRSSI < wc.config.WeakSignalThreshold {
			log.Printf("Weak signal detected for client %s: avg RSSI %d", 
				client.MacAddress, avgRSSI)
		}
	}
}

func (wc *WiFiClientCollector) updateAccessPointQuality() {
	// Update access point quality metrics
	wc.apMu.Lock()
	defer wc.apMu.Unlock()
	
	for _, ap := range wc.accessPoints {
		quality := &ap.SignalQuality
		
		// Calculate average RSSI for connected clients
		if len(ap.ConnectedClients) > 0 {
			totalRSSI := 0
			for _, client := range ap.ConnectedClients {
				totalRSSI += client.RSSI
			}
			quality.AverageRSSI = totalRSSI / len(ap.ConnectedClients)
		}
		
		quality.ClientCount = len(ap.ConnectedClients)
		
		// Calculate quality score
		quality.QualityScore = wc.calculateQualityScore(quality)
		
		log.Printf("AP %s quality: clients=%d, avg_rssi=%d, score=%.2f",
			ap.DeviceID, quality.ClientCount, quality.AverageRSSI, quality.QualityScore)
	}
}

func (wc *WiFiClientCollector) calculateQualityScore(quality *AccessPointQuality) float64 {
	score := 1.0
	
	// Penalty for weak signal
	if quality.AverageRSSI < -70 {
		score -= 0.3
	} else if quality.AverageRSSI < -60 {
		score -= 0.1
	}
	
	// Penalty for high client count (congestion)
	if quality.ClientCount > 20 {
		score -= 0.2
	} else if quality.ClientCount > 10 {
		score -= 0.1
	}
	
	// Penalty for channel utilization
	if quality.ChannelUtilization > 0.8 {
		score -= 0.2
	} else if quality.ChannelUtilization > 0.6 {
		score -= 0.1
	}
	
	if score < 0 {
		score = 0
	}
	
	return score
}

func (wc *WiFiClientCollector) cleanupOldData() {
	now := time.Now()
	
	// Clean up old signal history
	wc.clientsMu.Lock()
	for _, client := range wc.clients {
		cutoff := now.Add(-wc.config.SignalHistoryRetention)
		var filteredHistory []SignalMeasurement
		
		for _, measurement := range client.SignalHistory {
			if measurement.Timestamp.After(cutoff) {
				filteredHistory = append(filteredHistory, measurement)
			}
		}
		
		client.SignalHistory = filteredHistory
		
		// Clean up old roaming history
		cutoff = now.Add(-wc.config.RoamingHistoryRetention)
		var filteredRoaming []RoamingEvent
		
		for _, event := range client.RoamingHistory {
			if event.Timestamp.After(cutoff) {
				filteredRoaming = append(filteredRoaming, event)
			}
		}
		
		client.RoamingHistory = filteredRoaming
	}
	wc.clientsMu.Unlock()
	
	// Clean up offline clients from AP lists
	wc.apMu.Lock()
	for _, ap := range wc.accessPoints {
		cutoff := now.Add(-wc.config.ClientOfflineTimeout)
		
		for mac, client := range ap.ConnectedClients {
			if client.LastSeen.Before(cutoff) {
				delete(ap.ConnectedClients, mac)
			}
		}
	}
	wc.apMu.Unlock()
	
	log.Printf("Completed WiFi data cleanup")
}

func (wc *WiFiClientCollector) removeClientFromAPs(macAddress string) {
	wc.apMu.Lock()
	defer wc.apMu.Unlock()
	
	for _, ap := range wc.accessPoints {
		delete(ap.ConnectedClients, macAddress)
	}
}

func (wc *WiFiClientCollector) loadExistingData() error {
	// Load existing WiFi client and AP data from storage
	// This would typically involve reading from the database
	
	log.Printf("Loading existing WiFi data from storage")
	
	// TODO: Implement loading from storage
	// For now, start with empty state
	
	return nil
}