package topology

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"rtk_controller/internal/storage"
)

// RoamingDetector detects and analyzes WiFi client roaming behavior
type RoamingDetector struct {
	// Data sources
	wifiCollector   *WiFiClientCollector
	storage         *storage.TopologyStorage
	identityStorage *storage.IdentityStorage

	// Roaming state tracking
	clientStates  map[string]*ClientRoamingState
	roamingEvents []RoamingAnalysisEvent
	anomalies     []RoamingAnomaly
	mu            sync.RWMutex

	// Configuration
	config RoamingDetectorConfig

	// Background processing
	running bool
	cancel  context.CancelFunc

	// Statistics
	stats RoamingDetectorStats
}

// ClientRoamingState tracks roaming state for a specific client
type ClientRoamingState struct {
	MacAddress          string
	FriendlyName        string
	CurrentAP           string
	PreviousAP          string
	LastRoamTime        time.Time
	RoamingFrequency    float64 // roams per hour
	StableConnection    bool
	SignalStrengthTrend []int
	ConnectionQuality   ConnectionQuality
	RoamingPattern      RoamingPattern
	AnomalyScore        float64
}

// RoamingAnalysisEvent represents a detected roaming event with analysis
type RoamingAnalysisEvent struct {
	ID           string
	MacAddress   string
	FriendlyName string
	FromAP       string
	ToAP         string
	FromSSID     string
	ToSSID       string
	Timestamp    time.Time
	Duration     time.Duration
	RoamingType  RoamingType
	Trigger      RoamingTrigger
	SignalBefore int
	SignalAfter  int
	Quality      EventQuality
	IsAnomaly    bool
	Confidence   float64
	Context      RoamingContext
}

// RoamingAnomaly represents unusual roaming behavior
type RoamingAnomaly struct {
	ID             string
	Type           AnomalyType
	MacAddress     string
	FriendlyName   string
	Description    string
	Severity       AnomalySeverity
	FirstDetected  time.Time
	LastOccurrence time.Time
	Frequency      int
	Details        map[string]interface{}
	Resolved       bool
}

// ConnectionQuality holds connection quality metrics
type ConnectionQuality struct {
	AverageRSSI     int
	SignalStability float64
	ThroughputMbps  int
	LatencyMs       float64
	PacketLoss      float64
	QualityScore    float64 // 0-1 scale
}

// RoamingPattern identifies patterns in roaming behavior
// RoamingPattern moved to network_diagnostics.go to avoid duplication

// RoamingContext provides context about the roaming environment
type RoamingContext struct {
	NetworkLoad       float64
	APUtilization     map[string]float64
	InterferenceLevel int
	ClientCount       int
	TimeOfDay         string
	DayOfWeek         string
}

// Enums for roaming analysis
type RoamingType string

const (
	RoamingTypeSignalDriven  RoamingType = "signal_driven"
	RoamingTypeLoadBalancing RoamingType = "load_balancing"
	RoamingTypeBandSteering  RoamingType = "band_steering"
	RoamingTypeForced        RoamingType = "forced"
	RoamingTypeManual        RoamingType = "manual"
	RoamingTypeUnknown       RoamingType = "unknown"
)

type RoamingTrigger string

const (
	TriggerWeakSignal   RoamingTrigger = "weak_signal"
	TriggerBetterSignal RoamingTrigger = "better_signal"
	TriggerHighLoad     RoamingTrigger = "high_load"
	TriggerInterference RoamingTrigger = "interference"
	TriggerAPFailure    RoamingTrigger = "ap_failure"
	TriggerUserMovement RoamingTrigger = "user_movement"
	TriggerUnknown      RoamingTrigger = "unknown"
)

type EventQuality string

const (
	QualityExcellent EventQuality = "excellent"
	QualityGood      EventQuality = "good"
	QualityFair      EventQuality = "fair"
	QualityPoor      EventQuality = "poor"
)

type AnomalyType string

const (
	AnomalyExcessiveRoaming AnomalyType = "excessive_roaming"
	AnomalyPingPong         AnomalyType = "ping_pong"
	AnomalyStuckClient      AnomalyType = "stuck_client"
	AnomalyUnusualPattern   AnomalyType = "unusual_pattern"
	AnomalySignalAnomaly    AnomalyType = "signal_anomaly"
	AnomalyTimeAnomaly      AnomalyType = "time_anomaly"
)

type AnomalySeverity string

const (
// SeverityLow moved to constants.go
// SeverityMedium moved to constants.go
// SeverityHigh moved to constants.go
// SeverityCritical moved to constants.go
)

type PatternType string

const (
	PatternNormal     PatternType = "normal"
	PatternCommuter   PatternType = "commuter"
	PatternStationary PatternType = "stationary"
	PatternMobile     PatternType = "mobile"
	PatternIrregular  PatternType = "irregular"
)

// RoamingDetectorConfig holds detector configuration
type RoamingDetectorConfig struct {
	// Detection thresholds
	RoamingTimeWindow         time.Duration
	ExcessiveRoamingThreshold int // roams per hour
	PingPongTimeThreshold     time.Duration
	WeakSignalThreshold       int // RSSI
	StrongSignalThreshold     int // RSSI

	// Analysis settings
	EnablePatternAnalysis    bool
	EnableAnomalyDetection   bool
	EnablePredictiveAnalysis bool
	LearningPeriod           time.Duration
	ConfidenceThreshold      float64

	// Processing intervals
	AnalysisInterval      time.Duration
	PatternUpdateInterval time.Duration
	AnomalyCheckInterval  time.Duration

	// Data retention
	EventRetention   time.Duration
	AnomalyRetention time.Duration
	PatternRetention time.Duration

	// Performance settings
	MaxEventsPerClient int
	MaxAnomalies       int
	BatchSize          int
}

// RoamingDetectorStats holds detector statistics
type RoamingDetectorStats struct {
	TotalRoamingEvents      int64
	NormalRoaming           int64
	AnomalousRoaming        int64
	PingPongEvents          int64
	ExcessiveRoamingClients int64
	PatternsIdentified      int64
	LastAnalysis            time.Time
	ProcessingErrors        int64
}

// NewRoamingDetector creates a new roaming detector
func NewRoamingDetector(
	wifiCollector *WiFiClientCollector,
	storage *storage.TopologyStorage,
	identityStorage *storage.IdentityStorage,
	config RoamingDetectorConfig,
) *RoamingDetector {
	return &RoamingDetector{
		wifiCollector:   wifiCollector,
		storage:         storage,
		identityStorage: identityStorage,
		clientStates:    make(map[string]*ClientRoamingState),
		roamingEvents:   []RoamingAnalysisEvent{},
		anomalies:       []RoamingAnomaly{},
		config:          config,
		stats:           RoamingDetectorStats{},
	}
}

// Start begins roaming detection and analysis
func (rd *RoamingDetector) Start() error {
	rd.mu.Lock()
	defer rd.mu.Unlock()

	if rd.running {
		return fmt.Errorf("roaming detector is already running")
	}

	ctx, cancel := context.WithCancel(context.Background())
	rd.cancel = cancel
	rd.running = true

	log.Printf("Starting roaming detector")

	// Start background analysis loops
	go rd.roamingAnalysisLoop(ctx)
	go rd.patternAnalysisLoop(ctx)
	go rd.anomalyDetectionLoop(ctx)
	go rd.cleanupLoop(ctx)

	return nil
}

// Stop stops roaming detection
func (rd *RoamingDetector) Stop() error {
	rd.mu.Lock()
	defer rd.mu.Unlock()

	if !rd.running {
		return fmt.Errorf("roaming detector is not running")
	}

	rd.cancel()
	rd.running = false

	log.Printf("Roaming detector stopped")
	return nil
}

// AnalyzeRoamingEvent analyzes a new roaming event
func (rd *RoamingDetector) AnalyzeRoamingEvent(event RoamingEvent) (*RoamingAnalysisEvent, error) {
	rd.mu.Lock()
	defer rd.mu.Unlock()

	// Get or create client state
	clientState, exists := rd.clientStates[event.FromAP]
	if !exists {
		clientState = &ClientRoamingState{
			MacAddress:          event.FromAP, // Using FromAP as client identifier
			SignalStrengthTrend: []int{},
		}
		rd.clientStates[event.FromAP] = clientState
	}

	// Get friendly name from identity storage
	if identity, err := rd.identityStorage.GetDeviceIdentity(event.FromAP); err == nil {
		clientState.FriendlyName = identity.FriendlyName
	}

	// Create analysis event
	analysisEvent := &RoamingAnalysisEvent{
		ID:           fmt.Sprintf("roam_%d_%s", event.Timestamp.UnixMilli(), event.FromAP),
		MacAddress:   event.FromAP,
		FriendlyName: clientState.FriendlyName,
		FromAP:       event.FromAP,
		ToAP:         event.ToAP,
		FromSSID:     event.FromSSID,
		ToSSID:       event.ToSSID,
		Timestamp:    event.Timestamp,
		Duration:     event.Duration,
		SignalBefore: event.SignalBefore,
		SignalAfter:  event.SignalAfter,
		Context:      rd.buildRoamingContext(),
	}

	// Analyze roaming type and trigger
	rd.analyzeRoamingType(analysisEvent, clientState)
	rd.analyzeRoamingTrigger(analysisEvent, clientState)
	rd.calculateEventQuality(analysisEvent)
	rd.calculateConfidence(analysisEvent)

	// Update client state
	rd.updateClientState(clientState, analysisEvent)

	// Check for anomalies
	if rd.config.EnableAnomalyDetection {
		rd.checkForAnomalies(clientState, analysisEvent)
	}

	// Store event
	rd.roamingEvents = append(rd.roamingEvents, *analysisEvent)

	// Limit event history
	if len(rd.roamingEvents) > rd.config.MaxEventsPerClient {
		rd.roamingEvents = rd.roamingEvents[len(rd.roamingEvents)-rd.config.MaxEventsPerClient:]
	}

	rd.stats.TotalRoamingEvents++
	if analysisEvent.IsAnomaly {
		rd.stats.AnomalousRoaming++
	} else {
		rd.stats.NormalRoaming++
	}

	log.Printf("Analyzed roaming event: %s -> %s (type: %s, trigger: %s, quality: %s)",
		analysisEvent.FromAP, analysisEvent.ToAP, analysisEvent.RoamingType,
		analysisEvent.Trigger, analysisEvent.Quality)

	return analysisEvent, nil
}

// GetRoamingEvents returns recent roaming analysis events
func (rd *RoamingDetector) GetRoamingEvents(since time.Time, macAddress string) []RoamingAnalysisEvent {
	rd.mu.RLock()
	defer rd.mu.RUnlock()

	var events []RoamingAnalysisEvent

	for _, event := range rd.roamingEvents {
		if event.Timestamp.After(since) {
			if macAddress == "" || event.MacAddress == macAddress {
				events = append(events, event)
			}
		}
	}

	return events
}

// GetAnomalies returns detected roaming anomalies
func (rd *RoamingDetector) GetAnomalies(resolved bool) []RoamingAnomaly {
	rd.mu.RLock()
	defer rd.mu.RUnlock()

	var anomalies []RoamingAnomaly

	for _, anomaly := range rd.anomalies {
		if anomaly.Resolved == resolved {
			anomalies = append(anomalies, anomaly)
		}
	}

	return anomalies
}

// GetClientRoamingState returns roaming state for a specific client
func (rd *RoamingDetector) GetClientRoamingState(macAddress string) (*ClientRoamingState, bool) {
	rd.mu.RLock()
	defer rd.mu.RUnlock()

	state, exists := rd.clientStates[macAddress]
	if !exists {
		return nil, false
	}

	// Return copy
	stateCopy := *state
	return &stateCopy, true
}

// GetStats returns detector statistics
func (rd *RoamingDetector) GetStats() RoamingDetectorStats {
	rd.mu.RLock()
	defer rd.mu.RUnlock()

	stats := rd.stats
	stats.ExcessiveRoamingClients = rd.countExcessiveRoamingClients()
	stats.PingPongEvents = rd.countPingPongEvents()

	return stats
}

// Private methods

func (rd *RoamingDetector) roamingAnalysisLoop(ctx context.Context) {
	ticker := time.NewTicker(rd.config.AnalysisInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			rd.performRoamingAnalysis()
		}
	}
}

func (rd *RoamingDetector) patternAnalysisLoop(ctx context.Context) {
	if !rd.config.EnablePatternAnalysis {
		return
	}

	ticker := time.NewTicker(rd.config.PatternUpdateInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			rd.analyzeRoamingPatterns()
		}
	}
}

func (rd *RoamingDetector) anomalyDetectionLoop(ctx context.Context) {
	if !rd.config.EnableAnomalyDetection {
		return
	}

	ticker := time.NewTicker(rd.config.AnomalyCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			rd.detectAnomalies()
		}
	}
}

func (rd *RoamingDetector) cleanupLoop(ctx context.Context) {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			rd.cleanupOldData()
		}
	}
}

func (rd *RoamingDetector) performRoamingAnalysis() {
	// Get recent roaming events from WiFi collector
	since := time.Now().Add(-rd.config.AnalysisInterval)
	roamingEvents := rd.wifiCollector.GetRoamingEvents(since)

	for _, event := range roamingEvents {
		if _, err := rd.AnalyzeRoamingEvent(event); err != nil {
			log.Printf("Failed to analyze roaming event: %v", err)
			rd.stats.ProcessingErrors++
		}
	}

	rd.stats.LastAnalysis = time.Now()
}

func (rd *RoamingDetector) analyzeRoamingType(event *RoamingAnalysisEvent, clientState *ClientRoamingState) {
	// Determine roaming type based on signal strength and timing

	signalDiff := event.SignalAfter - event.SignalBefore

	if signalDiff > 10 {
		event.RoamingType = RoamingTypeSignalDriven
	} else if event.SignalBefore < rd.config.WeakSignalThreshold {
		event.RoamingType = RoamingTypeSignalDriven
	} else if event.Duration < 30*time.Second {
		event.RoamingType = RoamingTypeLoadBalancing
	} else {
		event.RoamingType = RoamingTypeUnknown
	}
}

func (rd *RoamingDetector) analyzeRoamingTrigger(event *RoamingAnalysisEvent, clientState *ClientRoamingState) {
	// Determine what triggered the roaming event

	if event.SignalBefore < rd.config.WeakSignalThreshold {
		event.Trigger = TriggerWeakSignal
	} else if event.SignalAfter > event.SignalBefore+10 {
		event.Trigger = TriggerBetterSignal
	} else if rd.isHighNetworkLoad(event.Context) {
		event.Trigger = TriggerHighLoad
	} else {
		event.Trigger = TriggerUnknown
	}
}

func (rd *RoamingDetector) calculateEventQuality(event *RoamingAnalysisEvent) {
	// Calculate overall quality of the roaming event

	score := 0.5 // Base score

	// Signal improvement
	signalImprovement := event.SignalAfter - event.SignalBefore
	if signalImprovement > 15 {
		score += 0.3
	} else if signalImprovement > 5 {
		score += 0.1
	} else if signalImprovement < -10 {
		score -= 0.3
	}

	// Duration (shorter is better for most cases)
	if event.Duration < 10*time.Second {
		score += 0.2
	} else if event.Duration > 60*time.Second {
		score -= 0.2
	}

	// Final signal strength
	if event.SignalAfter > rd.config.StrongSignalThreshold {
		score += 0.1
	} else if event.SignalAfter < rd.config.WeakSignalThreshold {
		score -= 0.2
	}

	// Assign quality based on score
	if score >= 0.8 {
		event.Quality = QualityExcellent
	} else if score >= 0.6 {
		event.Quality = QualityGood
	} else if score >= 0.4 {
		event.Quality = QualityFair
	} else {
		event.Quality = QualityPoor
	}
}

func (rd *RoamingDetector) calculateConfidence(event *RoamingAnalysisEvent) {
	// Calculate confidence in the analysis

	confidence := 0.7 // Base confidence

	// More confidence with clear signal patterns
	signalDiff := event.SignalAfter - event.SignalBefore
	if abs(signalDiff) > 15 {
		confidence += 0.2
	}

	// More confidence with consistent timing
	if event.Duration > 5*time.Second && event.Duration < 30*time.Second {
		confidence += 0.1
	}

	event.Confidence = confidence
}

func (rd *RoamingDetector) updateClientState(clientState *ClientRoamingState, event *RoamingAnalysisEvent) {
	// Update client roaming state based on new event

	clientState.PreviousAP = clientState.CurrentAP
	clientState.CurrentAP = event.ToAP
	clientState.LastRoamTime = event.Timestamp

	// Update signal trend
	clientState.SignalStrengthTrend = append(clientState.SignalStrengthTrend, event.SignalAfter)
	if len(clientState.SignalStrengthTrend) > 10 {
		clientState.SignalStrengthTrend = clientState.SignalStrengthTrend[1:]
	}

	// Calculate roaming frequency
	rd.calculateRoamingFrequency(clientState)

	// Update connection quality
	rd.updateConnectionQuality(clientState)

	// Calculate anomaly score
	clientState.AnomalyScore = rd.calculateAnomalyScore(clientState)
}

func (rd *RoamingDetector) calculateRoamingFrequency(clientState *ClientRoamingState) {
	// Calculate roaming frequency over the last hour
	since := time.Now().Add(-time.Hour)
	count := 0

	for _, event := range rd.roamingEvents {
		if event.MacAddress == clientState.MacAddress && event.Timestamp.After(since) {
			count++
		}
	}

	clientState.RoamingFrequency = float64(count)
}

func (rd *RoamingDetector) updateConnectionQuality(clientState *ClientRoamingState) {
	// Update connection quality metrics

	if len(clientState.SignalStrengthTrend) > 0 {
		// Calculate average RSSI
		total := 0
		for _, rssi := range clientState.SignalStrengthTrend {
			total += rssi
		}
		clientState.ConnectionQuality.AverageRSSI = total / len(clientState.SignalStrengthTrend)

		// Calculate signal stability (variance)
		variance := 0.0
		avg := float64(clientState.ConnectionQuality.AverageRSSI)
		for _, rssi := range clientState.SignalStrengthTrend {
			diff := float64(rssi) - avg
			variance += diff * diff
		}
		variance /= float64(len(clientState.SignalStrengthTrend))
		clientState.ConnectionQuality.SignalStability = 1.0 / (1.0 + variance/100.0)

		// Calculate overall quality score
		qualityScore := 0.5
		if clientState.ConnectionQuality.AverageRSSI > rd.config.StrongSignalThreshold {
			qualityScore += 0.3
		}
		qualityScore += clientState.ConnectionQuality.SignalStability * 0.2

		clientState.ConnectionQuality.QualityScore = qualityScore
	}
}

func (rd *RoamingDetector) calculateAnomalyScore(clientState *ClientRoamingState) float64 {
	score := 0.0

	// Excessive roaming
	if clientState.RoamingFrequency > float64(rd.config.ExcessiveRoamingThreshold) {
		score += 0.4
	}

	// Poor signal quality
	if clientState.ConnectionQuality.QualityScore < 0.3 {
		score += 0.3
	}

	// Low signal stability
	if clientState.ConnectionQuality.SignalStability < 0.5 {
		score += 0.3
	}

	return score
}

func (rd *RoamingDetector) checkForAnomalies(clientState *ClientRoamingState, event *RoamingAnalysisEvent) {
	// Check for various types of anomalies

	// Excessive roaming
	if clientState.RoamingFrequency > float64(rd.config.ExcessiveRoamingThreshold) {
		rd.createAnomaly(AnomalyExcessiveRoaming, clientState,
			fmt.Sprintf("Client roaming %d times per hour", int(clientState.RoamingFrequency)),
			SeverityMedium)
	}

	// Ping-pong roaming
	if clientState.PreviousAP == event.ToAP &&
		time.Since(clientState.LastRoamTime) < rd.config.PingPongTimeThreshold {
		rd.createAnomaly(AnomalyPingPong, clientState,
			fmt.Sprintf("Client ping-ponging between APs %s and %s", clientState.PreviousAP, event.ToAP),
			SeverityHigh)
	}

	// Signal anomaly
	if event.SignalAfter < event.SignalBefore-20 {
		rd.createAnomaly(AnomalySignalAnomaly, clientState,
			fmt.Sprintf("Roaming resulted in significantly worse signal: %d -> %d",
				event.SignalBefore, event.SignalAfter),
			SeverityMedium)
	}
}

func (rd *RoamingDetector) createAnomaly(
	anomalyType AnomalyType,
	clientState *ClientRoamingState,
	description string,
	severity AnomalySeverity,
) {

	anomaly := RoamingAnomaly{
		ID:             fmt.Sprintf("anomaly_%d_%s", time.Now().UnixMilli(), clientState.MacAddress),
		Type:           anomalyType,
		MacAddress:     clientState.MacAddress,
		FriendlyName:   clientState.FriendlyName,
		Description:    description,
		Severity:       severity,
		FirstDetected:  time.Now(),
		LastOccurrence: time.Now(),
		Frequency:      1,
		Details:        make(map[string]interface{}),
	}

	// Check if similar anomaly already exists
	for i, existing := range rd.anomalies {
		if existing.Type == anomaly.Type && existing.MacAddress == anomaly.MacAddress && !existing.Resolved {
			// Update existing anomaly
			rd.anomalies[i].LastOccurrence = time.Now()
			rd.anomalies[i].Frequency++
			return
		}
	}

	// Add new anomaly
	rd.anomalies = append(rd.anomalies, anomaly)

	// Limit anomaly history
	if len(rd.anomalies) > rd.config.MaxAnomalies {
		rd.anomalies = rd.anomalies[1:]
	}

	log.Printf("Roaming anomaly detected: %s for client %s", anomaly.Type, anomaly.MacAddress)
}

func (rd *RoamingDetector) analyzeRoamingPatterns() {
	// Analyze patterns in roaming behavior
	rd.stats.PatternsIdentified++

	for _, clientState := range rd.clientStates {
		rd.updateRoamingPattern(clientState)
	}
}

func (rd *RoamingDetector) updateRoamingPattern(clientState *ClientRoamingState) {
	// Update roaming pattern for a client

	// Analyze frequency patterns
	if clientState.RoamingFrequency < 1 {
		clientState.RoamingPattern.Pattern = string(PatternStationary)
	} else if clientState.RoamingFrequency > 5 {
		clientState.RoamingPattern.Pattern = string(PatternMobile)
	} else {
		clientState.RoamingPattern.Pattern = string(PatternNormal)
	}

	// TODO: Implement more sophisticated pattern analysis
	// - Time-based patterns
	// - Location-based patterns
	// - Predictive analysis
}

func (rd *RoamingDetector) detectAnomalies() {
	// Periodic anomaly detection

	for _, clientState := range rd.clientStates {
		if clientState.AnomalyScore > 0.7 {
			// High anomaly score, investigate further
			rd.investigateClientAnomaly(clientState)
		}
	}
}

func (rd *RoamingDetector) investigateClientAnomaly(clientState *ClientRoamingState) {
	// Investigate potential anomaly for a client

	// TODO: Implement detailed anomaly investigation
	log.Printf("Investigating anomaly for client %s (score: %.2f)",
		clientState.MacAddress, clientState.AnomalyScore)
}

func (rd *RoamingDetector) cleanupOldData() {
	// Clean up old roaming events and anomalies
	now := time.Now()

	// Clean up old events
	cutoff := now.Add(-rd.config.EventRetention)
	var filteredEvents []RoamingAnalysisEvent

	for _, event := range rd.roamingEvents {
		if event.Timestamp.After(cutoff) {
			filteredEvents = append(filteredEvents, event)
		}
	}

	rd.roamingEvents = filteredEvents

	// Clean up old anomalies
	cutoff = now.Add(-rd.config.AnomalyRetention)
	var filteredAnomalies []RoamingAnomaly

	for _, anomaly := range rd.anomalies {
		if anomaly.LastOccurrence.After(cutoff) || !anomaly.Resolved {
			filteredAnomalies = append(filteredAnomalies, anomaly)
		}
	}

	rd.anomalies = filteredAnomalies

	log.Printf("Cleaned up old roaming data")
}

func (rd *RoamingDetector) buildRoamingContext() RoamingContext {
	// Build context information for roaming analysis

	now := time.Now()

	return RoamingContext{
		NetworkLoad:       0.5, // TODO: Get actual network load
		APUtilization:     make(map[string]float64),
		InterferenceLevel: 0,
		ClientCount:       len(rd.clientStates),
		TimeOfDay:         fmt.Sprintf("%02d:00", now.Hour()),
		DayOfWeek:         now.Weekday().String(),
	}
}

func (rd *RoamingDetector) isHighNetworkLoad(context RoamingContext) bool {
	return context.NetworkLoad > 0.8
}

func (rd *RoamingDetector) countExcessiveRoamingClients() int64 {
	count := int64(0)

	for _, clientState := range rd.clientStates {
		if clientState.RoamingFrequency > float64(rd.config.ExcessiveRoamingThreshold) {
			count++
		}
	}

	return count
}

func (rd *RoamingDetector) countPingPongEvents() int64 {
	count := int64(0)

	for _, anomaly := range rd.anomalies {
		if anomaly.Type == AnomalyPingPong && !anomaly.Resolved {
			count++
		}
	}

	return count
}

// Helper function
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
