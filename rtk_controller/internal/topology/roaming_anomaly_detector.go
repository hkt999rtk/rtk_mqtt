package topology

import (
	"context"
	"fmt"
	"log"
	"math"
	"sort"
	"sync"
	"time"

	"rtk_controller/internal/storage"
)

// RoamingAnomalyDetector detects abnormal roaming behavior patterns
type RoamingAnomalyDetector struct {
	// Data sources
	roamingDetector   *RoamingDetector
	connectionTracker *ConnectionHistoryTracker
	inferenceEngine   *RoamingInferenceEngine
	storage           *storage.TopologyStorage
	identityStorage   *storage.IdentityStorage

	// Anomaly tracking
	detectedAnomalies map[string]*AnomalyCase
	baselineProfiles  map[string]*ClientBaselineProfile
	alertRules        []AnomalyAlertRule
	mu                sync.RWMutex

	// Machine learning models
	models AnomalyDetectionModels

	// Configuration
	config AnomalyDetectorConfig

	// Background processing
	running bool
	cancel  context.CancelFunc

	// Statistics
	stats AnomalyDetectorStats
}

// AnomalyCase represents a detected anomaly with detailed analysis
type AnomalyCase struct {
	ID               string
	Type             AnomalyType
	MacAddress       string
	FriendlyName     string
	Title            string
	Description      string
	Severity         AnomalySeverity
	Status           AnomalyStatus
	FirstDetected    time.Time
	LastOccurrence   time.Time
	OccurrenceCount  int
	Evidence         []AnomalyEvidence
	Impact           AnomalyImpact
	RootCauses       []string
	Recommendations  []string
	RelatedAnomalies []string
	Resolution       *AnomalyResolution
	Metadata         map[string]interface{}
}

// AnomalyEvidence provides supporting evidence for detected anomalies
type AnomalyEvidence struct {
	Type        EvidenceType
	Timestamp   time.Time
	Value       interface{}
	Baseline    interface{}
	Deviation   float64
	Description string
	Confidence  float64
}

// AnomalyImpact describes the impact of an anomaly
type AnomalyImpact struct {
	UserExperience     ImpactLevel
	NetworkPerformance ImpactLevel
	SecurityRisk       ImpactLevel
	OperationalCost    ImpactLevel
	AffectedDevices    []string
	EstimatedDuration  time.Duration
}

// AnomalyResolution tracks anomaly resolution
type AnomalyResolution struct {
	ResolvedAt         time.Time
	ResolvedBy         string
	Resolution         string
	RootCause          string
	PreventiveMeasures []string
	EffectivenessScore float64
}

// ClientBaselineProfile represents normal behavior baseline for a client
type ClientBaselineProfile struct {
	MacAddress  string
	CreatedAt   time.Time
	LastUpdated time.Time
	SampleSize  int

	// Roaming patterns
	AverageRoamingFrequency float64
	RoamingFrequencyStdDev  float64
	PreferredAPs            []string
	PreferredTimeWindows    []TimeWindow
	SessionDurationMean     time.Duration
	SessionDurationStdDev   time.Duration

	// Signal patterns
	SignalQualityMean          float64
	SignalQualityStdDev        float64
	SignalStabilityMean        float64
	RoamingTriggerDistribution map[RoamingTrigger]float64

	// Behavioral patterns
	UsagePatterns           []UsagePattern
	ConnectivityReliability float64
	QualityExpectations     QualityProfile

	// Anomaly thresholds (adaptive)
	Thresholds AdaptiveThresholds
}

// TimeWindow represents a time range for pattern analysis
type TimeWindow struct {
	StartHour int
	EndHour   int
	DayOfWeek time.Weekday
	Frequency float64
}

// QualityProfile represents expected quality metrics
type QualityProfile struct {
	MinAcceptableRSSI   int
	MaxTolerableLatency time.Duration
	MinThroughput       float64
	MaxPacketLoss       float64
}

// AdaptiveThresholds hold adaptive anomaly detection thresholds
type AdaptiveThresholds struct {
	RoamingFrequencyThreshold float64
	SignalQualityThreshold    float64
	SessionStabilityThreshold float64
	PatternDeviationThreshold float64
	ConfidenceThreshold       float64
}

// AnomalyDetectionModels contains ML models for anomaly detection
type AnomalyDetectionModels struct {
	IsolationForest  *IsolationForestModel
	OneClassSVM      *OneClassSVMModel
	LSTMPrediction   *LSTMModel
	StatisticalModel *StatisticalAnomalyModel
}

// AnomalyAlertRule defines rules for anomaly alerting
type AnomalyAlertRule struct {
	ID             string
	Name           string
	AnomalyTypes   []AnomalyType
	SeverityFilter AnomalySeverity
	ClientFilter   []string
	TimeWindow     time.Duration
	AlertThreshold int
	Enabled        bool
	Actions        []AlertAction
}

// AlertAction defines actions to take when anomaly rules trigger
type AlertAction struct {
	Type        ActionType
	Parameters  map[string]interface{}
	Destination string
}

// Enums for anomaly detection
type AnomalyStatus string

const (
	// StatusActive moved to constants.go
	// StatusResolved moved to constants.go
	StatusIgnored AnomalyStatus = "ignored"
	// StatusPending moved to constants.go
)

type EvidenceType string

const (
	EvidenceFrequency   EvidenceType = "frequency"
	EvidenceSignal      EvidenceType = "signal"
	EvidenceTiming      EvidenceType = "timing"
	EvidencePattern     EvidenceType = "pattern"
	EvidenceQuality     EvidenceType = "quality"
	EvidenceStatistical EvidenceType = "statistical"
)

type ImpactLevel string

const (
	ImpactNone ImpactLevel = "none"
	// ImpactLow moved to constants.go
	// ImpactMedium moved to constants.go
	// ImpactHigh moved to constants.go
	// ImpactCritical moved to constants.go
)

// AnomalyDetectorConfig holds anomaly detector configuration
type AnomalyDetectorConfig struct {
	// Detection settings
	EnableMLDetection          bool
	EnableStatisticalDetection bool
	EnablePatternDetection     bool
	EnableRealTimeDetection    bool

	// Baseline building
	BaselineLearningPeriod time.Duration
	MinSamplesForBaseline  int
	BaselineUpdateInterval time.Duration
	AdaptiveThresholds     bool

	// Detection parameters
	AnomalyConfidenceThreshold float64
	StatisticalConfidenceLevel float64
	PatternDeviationThreshold  float64
	RoamingFrequencyMultiplier float64

	// Timing parameters
	DetectionInterval time.Duration
	AnalysisWindow    time.Duration
	CooldownPeriod    time.Duration

	// Data retention
	AnomalyRetention  time.Duration
	BaselineRetention time.Duration
	EvidenceRetention time.Duration

	// Performance settings
	MaxActiveAnomalies int
	BatchSize          int
	ProcessingTimeout  time.Duration

	// Alerting settings
	EnableAlerting   bool
	AlertCooldown    time.Duration
	MaxAlertsPerHour int
}

// AnomalyDetectorStats holds detector statistics
type AnomalyDetectorStats struct {
	TotalAnomaliesDetected int64
	ActiveAnomalies        int64
	ResolvedAnomalies      int64
	IgnoredAnomalies       int64
	FalsePositives         int64
	TruePositives          int64
	DetectionAccuracy      float64
	AverageDetectionTime   time.Duration
	BaselineProfiles       int64
	LastDetectionRun       time.Time
	ProcessingErrors       int64
}

// NewRoamingAnomalyDetector creates a new roaming anomaly detector
func NewRoamingAnomalyDetector(
	roamingDetector *RoamingDetector,
	connectionTracker *ConnectionHistoryTracker,
	inferenceEngine *RoamingInferenceEngine,
	storage *storage.TopologyStorage,
	identityStorage *storage.IdentityStorage,
	config AnomalyDetectorConfig,
) *RoamingAnomalyDetector {

	detector := &RoamingAnomalyDetector{
		roamingDetector:   roamingDetector,
		connectionTracker: connectionTracker,
		inferenceEngine:   inferenceEngine,
		storage:           storage,
		identityStorage:   identityStorage,
		detectedAnomalies: make(map[string]*AnomalyCase),
		baselineProfiles:  make(map[string]*ClientBaselineProfile),
		alertRules:        []AnomalyAlertRule{},
		config:            config,
		stats:             AnomalyDetectorStats{},
	}

	// Initialize models
	detector.initializeModels()

	// Initialize alert rules
	detector.initializeAlertRules()

	return detector
}

// Start begins anomaly detection
func (rad *RoamingAnomalyDetector) Start() error {
	rad.mu.Lock()
	defer rad.mu.Unlock()

	if rad.running {
		return fmt.Errorf("roaming anomaly detector is already running")
	}

	ctx, cancel := context.WithCancel(context.Background())
	rad.cancel = cancel
	rad.running = true

	log.Printf("Starting roaming anomaly detector")

	// Start background processing
	go rad.anomalyDetectionLoop(ctx)
	go rad.baselineUpdateLoop(ctx)
	go rad.alertingLoop(ctx)
	go rad.cleanupLoop(ctx)

	// Load existing baselines
	if err := rad.loadExistingBaselines(); err != nil {
		log.Printf("Failed to load existing baselines: %v", err)
	}

	return nil
}

// Stop stops anomaly detection
func (rad *RoamingAnomalyDetector) Stop() error {
	rad.mu.Lock()
	defer rad.mu.Unlock()

	if !rad.running {
		return fmt.Errorf("roaming anomaly detector is not running")
	}

	rad.cancel()
	rad.running = false

	log.Printf("Roaming anomaly detector stopped")
	return nil
}

// DetectAnomaliesForClient detects anomalies for a specific client
func (rad *RoamingAnomalyDetector) DetectAnomaliesForClient(macAddress string) ([]*AnomalyCase, error) {
	rad.mu.Lock()
	defer rad.mu.Unlock()

	var anomalies []*AnomalyCase

	// Get client baseline
	baseline, hasBaseline := rad.baselineProfiles[macAddress]
	if !hasBaseline {
		// Build baseline first
		if err := rad.buildClientBaseline(macAddress); err != nil {
			return nil, fmt.Errorf("failed to build baseline: %w", err)
		}
		baseline = rad.baselineProfiles[macAddress]
	}

	// Get recent roaming events
	since := time.Now().Add(-rad.config.AnalysisWindow)
	events := rad.roamingDetector.GetRoamingEvents(since, macAddress)

	// Statistical anomaly detection
	if rad.config.EnableStatisticalDetection {
		statAnomalies := rad.detectStatisticalAnomalies(macAddress, events, baseline)
		anomalies = append(anomalies, statAnomalies...)
	}

	// Pattern-based anomaly detection
	if rad.config.EnablePatternDetection {
		patternAnomalies := rad.detectPatternAnomalies(macAddress, events, baseline)
		anomalies = append(anomalies, patternAnomalies...)
	}

	// ML-based anomaly detection
	if rad.config.EnableMLDetection {
		mlAnomalies := rad.detectMLAnomalies(macAddress, events, baseline)
		anomalies = append(anomalies, mlAnomalies...)
	}

	// Store detected anomalies
	for _, anomaly := range anomalies {
		rad.detectedAnomalies[anomaly.ID] = anomaly
		rad.stats.TotalAnomaliesDetected++
		rad.stats.ActiveAnomalies++
	}

	return anomalies, nil
}

// GetActiveAnomalies returns currently active anomalies
func (rad *RoamingAnomalyDetector) GetActiveAnomalies() []*AnomalyCase {
	rad.mu.RLock()
	defer rad.mu.RUnlock()

	var active []*AnomalyCase

	for _, anomaly := range rad.detectedAnomalies {
		if anomaly.Status == StatusActive {
			active = append(active, anomaly)
		}
	}

	// Sort by severity and last occurrence
	sort.Slice(active, func(i, j int) bool {
		if active[i].Severity != active[j].Severity {
			return rad.getSeverityWeight(active[i].Severity) > rad.getSeverityWeight(active[j].Severity)
		}
		return active[i].LastOccurrence.After(active[j].LastOccurrence)
	})

	return active
}

// GetAnomalyCase returns details of a specific anomaly case
func (rad *RoamingAnomalyDetector) GetAnomalyCase(anomalyID string) (*AnomalyCase, bool) {
	rad.mu.RLock()
	defer rad.mu.RUnlock()

	anomaly, exists := rad.detectedAnomalies[anomalyID]
	return anomaly, exists
}

// ResolveAnomaly marks an anomaly as resolved
func (rad *RoamingAnomalyDetector) ResolveAnomaly(
	anomalyID string,
	resolvedBy string,
	resolution string,
	rootCause string,
) error {

	rad.mu.Lock()
	defer rad.mu.Unlock()

	anomaly, exists := rad.detectedAnomalies[anomalyID]
	if !exists {
		return fmt.Errorf("anomaly not found: %s", anomalyID)
	}

	anomaly.Status = StatusResolved
	anomaly.Resolution = &AnomalyResolution{
		ResolvedAt:         time.Now(),
		ResolvedBy:         resolvedBy,
		Resolution:         resolution,
		RootCause:          rootCause,
		EffectivenessScore: 1.0, // Default, can be updated later
	}

	rad.stats.ActiveAnomalies--
	rad.stats.ResolvedAnomalies++

	log.Printf("Anomaly resolved: %s by %s", anomalyID, resolvedBy)
	return nil
}

// GetClientBaseline returns baseline profile for a client
func (rad *RoamingAnomalyDetector) GetClientBaseline(macAddress string) (*ClientBaselineProfile, bool) {
	rad.mu.RLock()
	defer rad.mu.RUnlock()

	baseline, exists := rad.baselineProfiles[macAddress]
	if !exists {
		return nil, false
	}

	// Return copy
	baselineCopy := *baseline
	return &baselineCopy, true
}

// GetStats returns detector statistics
func (rad *RoamingAnomalyDetector) GetStats() AnomalyDetectorStats {
	rad.mu.RLock()
	defer rad.mu.RUnlock()

	stats := rad.stats
	stats.BaselineProfiles = int64(len(rad.baselineProfiles))

	// Calculate detection accuracy
	if stats.TruePositives+stats.FalsePositives > 0 {
		stats.DetectionAccuracy = float64(stats.TruePositives) / float64(stats.TruePositives+stats.FalsePositives)
	}

	return stats
}

// Private methods

func (rad *RoamingAnomalyDetector) anomalyDetectionLoop(ctx context.Context) {
	ticker := time.NewTicker(rad.config.DetectionInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			rad.runAnomalyDetection()
		}
	}
}

func (rad *RoamingAnomalyDetector) baselineUpdateLoop(ctx context.Context) {
	ticker := time.NewTicker(rad.config.BaselineUpdateInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			rad.updateBaselines()
		}
	}
}

func (rad *RoamingAnomalyDetector) alertingLoop(ctx context.Context) {
	if !rad.config.EnableAlerting {
		return
	}

	ticker := time.NewTicker(time.Minute) // Check alerts every minute
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			rad.processAlerts()
		}
	}
}

func (rad *RoamingAnomalyDetector) cleanupLoop(ctx context.Context) {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			rad.cleanupOldData()
		}
	}
}

func (rad *RoamingAnomalyDetector) runAnomalyDetection() {
	// Get active clients from WiFi collector
	activeClients := rad.roamingDetector.wifiCollector.GetActiveClients()

	rad.mu.Lock()
	defer rad.mu.Unlock()

	for macAddress := range activeClients {
		anomalies, err := rad.DetectAnomaliesForClient(macAddress)
		if err != nil {
			log.Printf("Failed to detect anomalies for client %s: %v", macAddress, err)
			rad.stats.ProcessingErrors++
			continue
		}

		if len(anomalies) > 0 {
			log.Printf("Detected %d anomalies for client %s", len(anomalies), macAddress)
		}
	}

	rad.stats.LastDetectionRun = time.Now()
}

func (rad *RoamingAnomalyDetector) detectStatisticalAnomalies(
	macAddress string,
	events []RoamingAnalysisEvent,
	baseline *ClientBaselineProfile,
) []*AnomalyCase {

	var anomalies []*AnomalyCase

	if len(events) == 0 {
		return anomalies
	}

	// Calculate current metrics
	roamingFrequency := float64(len(events)) / rad.config.AnalysisWindow.Hours()

	// Check roaming frequency anomaly
	if roamingFrequency > baseline.AverageRoamingFrequency+2*baseline.RoamingFrequencyStdDev {
		anomaly := rad.createAnomalyCase(
			AnomalyExcessiveRoaming,
			macAddress,
			"Excessive Roaming Frequency",
			fmt.Sprintf("Client roaming %.2f times per hour (baseline: %.2f ± %.2f)",
				roamingFrequency, baseline.AverageRoamingFrequency, baseline.RoamingFrequencyStdDev),
			rad.calculateSeverityFromDeviation(roamingFrequency, baseline.AverageRoamingFrequency, baseline.RoamingFrequencyStdDev),
		)

		// Add evidence
		anomaly.Evidence = append(anomaly.Evidence, AnomalyEvidence{
			Type:        EvidenceFrequency,
			Timestamp:   time.Now(),
			Value:       roamingFrequency,
			Baseline:    baseline.AverageRoamingFrequency,
			Deviation:   (roamingFrequency - baseline.AverageRoamingFrequency) / baseline.RoamingFrequencyStdDev,
			Description: "Roaming frequency significantly higher than baseline",
			Confidence:  0.9,
		})

		anomalies = append(anomalies, anomaly)
	}

	// Check signal quality anomalies
	signalQualities := make([]float64, 0, len(events))
	for _, event := range events {
		if event.SignalAfter > 0 {
			quality := rad.calculateSignalQuality(event.SignalAfter)
			signalQualities = append(signalQualities, quality)
		}
	}

	if len(signalQualities) > 0 {
		avgQuality := rad.calculateMean(signalQualities)
		if avgQuality < baseline.SignalQualityMean-2*baseline.SignalQualityStdDev {
			anomaly := rad.createAnomalyCase(
				AnomalySignalAnomaly,
				macAddress,
				"Poor Signal Quality",
				fmt.Sprintf("Average signal quality %.2f is significantly below baseline %.2f ± %.2f",
					avgQuality, baseline.SignalQualityMean, baseline.SignalQualityStdDev),
				rad.calculateSeverityFromDeviation(avgQuality, baseline.SignalQualityMean, baseline.SignalQualityStdDev),
			)

			anomaly.Evidence = append(anomaly.Evidence, AnomalyEvidence{
				Type:        EvidenceSignal,
				Timestamp:   time.Now(),
				Value:       avgQuality,
				Baseline:    baseline.SignalQualityMean,
				Deviation:   (baseline.SignalQualityMean - avgQuality) / baseline.SignalQualityStdDev,
				Description: "Signal quality significantly below baseline",
				Confidence:  0.8,
			})

			anomalies = append(anomalies, anomaly)
		}
	}

	return anomalies
}

func (rad *RoamingAnomalyDetector) detectPatternAnomalies(
	macAddress string,
	events []RoamingAnalysisEvent,
	baseline *ClientBaselineProfile,
) []*AnomalyCase {

	var anomalies []*AnomalyCase

	// Check for ping-pong roaming patterns
	pingPongCount := rad.detectPingPongPattern(events)
	if pingPongCount > 0 {
		anomaly := rad.createAnomalyCase(
			AnomalyPingPong,
			macAddress,
			"Ping-Pong Roaming Pattern",
			fmt.Sprintf("Detected %d ping-pong roaming sequences", pingPongCount),
			SeverityHigh,
		)

		anomaly.Evidence = append(anomaly.Evidence, AnomalyEvidence{
			Type:        EvidencePattern,
			Timestamp:   time.Now(),
			Value:       pingPongCount,
			Baseline:    0,
			Deviation:   float64(pingPongCount),
			Description: "Ping-pong roaming pattern detected",
			Confidence:  0.95,
		})

		anomalies = append(anomalies, anomaly)
	}

	// Check for unusual timing patterns
	if rad.detectUnusualTimingPattern(events, baseline) {
		anomaly := rad.createAnomalyCase(
			AnomalyTimeAnomaly,
			macAddress,
			"Unusual Roaming Timing",
			"Roaming pattern deviates significantly from normal time windows",
			SeverityMedium,
		)

		anomalies = append(anomalies, anomaly)
	}

	return anomalies
}

func (rad *RoamingAnomalyDetector) detectMLAnomalies(
	macAddress string,
	events []RoamingAnalysisEvent,
	baseline *ClientBaselineProfile,
) []*AnomalyCase {

	var anomalies []*AnomalyCase

	// TODO: Implement ML-based anomaly detection
	// This would use trained models to detect complex patterns

	// For now, implement a simple isolation forest-like approach
	features := rad.extractFeatures(events)
	if len(features) > 0 {
		anomalyScore := rad.models.StatisticalModel.PredictAnomalyScore(features)

		if anomalyScore > rad.config.AnomalyConfidenceThreshold {
			anomaly := rad.createAnomalyCase(
				AnomalyUnusualPattern,
				macAddress,
				"Unusual Behavior Pattern",
				fmt.Sprintf("ML model detected unusual pattern (score: %.2f)", anomalyScore),
				rad.calculateSeverityFromScore(anomalyScore),
			)

			anomaly.Evidence = append(anomaly.Evidence, AnomalyEvidence{
				Type:        EvidenceStatistical,
				Timestamp:   time.Now(),
				Value:       anomalyScore,
				Baseline:    rad.config.AnomalyConfidenceThreshold,
				Deviation:   anomalyScore - rad.config.AnomalyConfidenceThreshold,
				Description: "Machine learning model detected anomalous behavior",
				Confidence:  anomalyScore,
			})

			anomalies = append(anomalies, anomaly)
		}
	}

	return anomalies
}

func (rad *RoamingAnomalyDetector) buildClientBaseline(macAddress string) error {
	// Get connection history
	history, exists := rad.connectionTracker.GetClientHistory(macAddress)
	if !exists {
		return fmt.Errorf("no connection history found for client %s", macAddress)
	}

	if len(history.Sessions) < rad.config.MinSamplesForBaseline {
		return fmt.Errorf("insufficient data for baseline: %d sessions (need %d)",
			len(history.Sessions), rad.config.MinSamplesForBaseline)
	}

	// Get roaming events
	since := time.Now().Add(-rad.config.BaselineLearningPeriod)
	events := rad.roamingDetector.GetRoamingEvents(since, macAddress)

	// Build baseline profile
	baseline := &ClientBaselineProfile{
		MacAddress:  macAddress,
		CreatedAt:   time.Now(),
		LastUpdated: time.Now(),
		SampleSize:  len(events),
	}

	// Calculate roaming frequency statistics
	if len(events) > 0 {
		frequencies := rad.calculateRoamingFrequencies(events, rad.config.BaselineLearningPeriod)
		baseline.AverageRoamingFrequency = rad.calculateMean(frequencies)
		baseline.RoamingFrequencyStdDev = rad.calculateStdDev(frequencies, baseline.AverageRoamingFrequency)
	}

	// Calculate signal quality statistics
	signalQualities := make([]float64, 0, len(events))
	for _, event := range events {
		if event.SignalAfter > 0 {
			quality := rad.calculateSignalQuality(event.SignalAfter)
			signalQualities = append(signalQualities, quality)
		}
	}

	if len(signalQualities) > 0 {
		baseline.SignalQualityMean = rad.calculateMean(signalQualities)
		baseline.SignalQualityStdDev = rad.calculateStdDev(signalQualities, baseline.SignalQualityMean)
	}

	// Calculate session duration statistics
	durations := make([]float64, 0, len(history.Sessions))
	for _, session := range history.Sessions {
		durations = append(durations, float64(session.Duration.Milliseconds()))
	}

	if len(durations) > 0 {
		meanDuration := rad.calculateMean(durations)
		baseline.SessionDurationMean = time.Duration(meanDuration) * time.Millisecond
		stdDevDuration := rad.calculateStdDev(durations, meanDuration)
		baseline.SessionDurationStdDev = time.Duration(stdDevDuration) * time.Millisecond
	}

	// Set adaptive thresholds
	baseline.Thresholds = AdaptiveThresholds{
		RoamingFrequencyThreshold: baseline.AverageRoamingFrequency + 2*baseline.RoamingFrequencyStdDev,
		SignalQualityThreshold:    baseline.SignalQualityMean - 2*baseline.SignalQualityStdDev,
		ConfidenceThreshold:       rad.config.AnomalyConfidenceThreshold,
	}

	rad.baselineProfiles[macAddress] = baseline
	rad.stats.BaselineProfiles++

	log.Printf("Built baseline profile for client %s: freq=%.2f±%.2f, quality=%.2f±%.2f",
		macAddress, baseline.AverageRoamingFrequency, baseline.RoamingFrequencyStdDev,
		baseline.SignalQualityMean, baseline.SignalQualityStdDev)

	return nil
}

func (rad *RoamingAnomalyDetector) createAnomalyCase(
	anomalyType AnomalyType,
	macAddress string,
	title string,
	description string,
	severity AnomalySeverity,
) *AnomalyCase {

	anomaly := &AnomalyCase{
		ID:              fmt.Sprintf("anomaly_%d_%s", time.Now().UnixMilli(), macAddress),
		Type:            anomalyType,
		MacAddress:      macAddress,
		Title:           title,
		Description:     description,
		Severity:        severity,
		Status:          StatusActive,
		FirstDetected:   time.Now(),
		LastOccurrence:  time.Now(),
		OccurrenceCount: 1,
		Evidence:        []AnomalyEvidence{},
		Metadata:        make(map[string]interface{}),
	}

	// Get friendly name
	if identity, err := rad.identityStorage.GetDeviceIdentity(macAddress); err == nil {
		anomaly.FriendlyName = identity.FriendlyName
	}

	// Calculate impact
	anomaly.Impact = rad.calculateAnomalyImpact(anomaly)

	// Generate recommendations
	anomaly.Recommendations = rad.generateRecommendations(anomaly)

	return anomaly
}

func (rad *RoamingAnomalyDetector) calculateSignalQuality(rssi int) float64 {
	// Convert RSSI to quality score (0-1)
	if rssi >= -50 {
		return 1.0
	} else if rssi <= -90 {
		return 0.0
	} else {
		return float64(rssi+90) / 40.0
	}
}

func (rad *RoamingAnomalyDetector) detectPingPongPattern(events []RoamingAnalysisEvent) int {
	// Detect ping-pong roaming (rapid switching between same APs)

	if len(events) < 4 {
		return 0
	}

	pingPongCount := 0
	for i := 0; i < len(events)-3; i++ {
		// Check for A->B->A->B pattern
		if events[i].FromAP == events[i+2].FromAP &&
			events[i].ToAP == events[i+2].ToAP &&
			events[i+1].FromAP == events[i+3].FromAP &&
			events[i+1].ToAP == events[i+3].ToAP &&
			events[i].FromAP == events[i+1].ToAP {

			// Check timing (should be rapid)
			timeDelta := events[i+3].Timestamp.Sub(events[i].Timestamp)
			if timeDelta < 5*time.Minute {
				pingPongCount++
			}
		}
	}

	return pingPongCount
}

func (rad *RoamingAnomalyDetector) detectUnusualTimingPattern(
	events []RoamingAnalysisEvent,
	baseline *ClientBaselineProfile,
) bool {

	// Check if roaming times deviate from baseline patterns

	hourCounts := make(map[int]int)
	for _, event := range events {
		hour := event.Timestamp.Hour()
		hourCounts[hour]++
	}

	// Compare with baseline time windows
	for _, window := range baseline.PreferredTimeWindows {
		expectedFrequency := window.Frequency
		actualFrequency := 0.0

		for hour := window.StartHour; hour <= window.EndHour; hour++ {
			actualFrequency += float64(hourCounts[hour])
		}

		if len(events) > 0 {
			actualFrequency /= float64(len(events))
		}

		// Check for significant deviation
		if math.Abs(actualFrequency-expectedFrequency) > rad.config.PatternDeviationThreshold {
			return true
		}
	}

	return false
}

func (rad *RoamingAnomalyDetector) extractFeatures(events []RoamingAnalysisEvent) []float64 {
	// Extract features for ML-based anomaly detection

	if len(events) == 0 {
		return nil
	}

	features := make([]float64, 10) // 10 features

	// Feature 1: Roaming frequency
	features[0] = float64(len(events))

	// Feature 2: Average signal improvement
	var totalImprovement float64
	for _, event := range events {
		totalImprovement += float64(event.SignalAfter - event.SignalBefore)
	}
	features[1] = totalImprovement / float64(len(events))

	// Feature 3: Time variance
	if len(events) > 1 {
		timeDiffs := make([]float64, len(events)-1)
		for i := 1; i < len(events); i++ {
			timeDiffs[i-1] = float64(events[i].Timestamp.Sub(events[i-1].Timestamp).Seconds())
		}
		features[2] = rad.calculateStdDev(timeDiffs, rad.calculateMean(timeDiffs))
	}

	// Features 4-10: Additional patterns (placeholder)
	for i := 3; i < 10; i++ {
		features[i] = 0.5 // Placeholder values
	}

	return features
}

// Helper functions

func (rad *RoamingAnomalyDetector) calculateMean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}

	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func (rad *RoamingAnomalyDetector) calculateStdDev(values []float64, mean float64) float64 {
	if len(values) == 0 {
		return 0
	}

	variance := 0.0
	for _, v := range values {
		diff := v - mean
		variance += diff * diff
	}
	variance /= float64(len(values))

	return math.Sqrt(variance)
}

func (rad *RoamingAnomalyDetector) calculateRoamingFrequencies(
	events []RoamingAnalysisEvent,
	period time.Duration,
) []float64 {

	// Calculate roaming frequency over sliding windows
	windowSize := time.Hour
	frequencies := []float64{}

	if len(events) == 0 {
		return frequencies
	}

	startTime := events[0].Timestamp
	endTime := time.Now()

	for current := startTime; current.Before(endTime); current = current.Add(windowSize) {
		windowEnd := current.Add(windowSize)
		count := 0

		for _, event := range events {
			if event.Timestamp.After(current) && event.Timestamp.Before(windowEnd) {
				count++
			}
		}

		frequencies = append(frequencies, float64(count))
	}

	return frequencies
}

func (rad *RoamingAnomalyDetector) calculateSeverityFromDeviation(
	value, baseline, stdDev float64,
) AnomalySeverity {

	deviation := math.Abs(value-baseline) / stdDev

	if deviation > 4 {
		return SeverityCritical
	} else if deviation > 3 {
		return SeverityHigh
	} else if deviation > 2 {
		return SeverityMedium
	} else {
		return SeverityLow
	}
}

func (rad *RoamingAnomalyDetector) calculateSeverityFromScore(score float64) AnomalySeverity {
	if score > 0.9 {
		return SeverityCritical
	} else if score > 0.8 {
		return SeverityHigh
	} else if score > 0.7 {
		return SeverityMedium
	} else {
		return SeverityLow
	}
}

func (rad *RoamingAnomalyDetector) getSeverityWeight(severity AnomalySeverity) int {
	switch severity {
	case SeverityCritical:
		return 4
	case SeverityHigh:
		return 3
	case SeverityMedium:
		return 2
	case SeverityLow:
		return 1
	default:
		return 0
	}
}

func (rad *RoamingAnomalyDetector) calculateAnomalyImpact(anomaly *AnomalyCase) AnomalyImpact {
	// Calculate impact based on anomaly type and severity

	impact := AnomalyImpact{
		AffectedDevices: []string{anomaly.MacAddress},
	}

	switch anomaly.Type {
	case AnomalyExcessiveRoaming:
		impact.UserExperience = ImpactHigh
		impact.NetworkPerformance = ImpactMedium
		impact.SecurityRisk = ImpactLow
	case AnomalyPingPong:
		impact.UserExperience = ImpactHigh
		impact.NetworkPerformance = ImpactHigh
		impact.SecurityRisk = ImpactLow
	case AnomalySignalAnomaly:
		impact.UserExperience = ImpactMedium
		impact.NetworkPerformance = ImpactLow
		impact.SecurityRisk = ImpactLow
	case AnomalyStuckClient:
		impact.UserExperience = ImpactMedium
		impact.NetworkPerformance = ImpactLow
		impact.SecurityRisk = ImpactMedium
	}

	return impact
}

func (rad *RoamingAnomalyDetector) generateRecommendations(anomaly *AnomalyCase) []string {
	var recommendations []string

	switch anomaly.Type {
	case AnomalyExcessiveRoaming:
		recommendations = append(recommendations,
			"Check AP placement and coverage",
			"Review roaming thresholds on APs",
			"Analyze interference sources",
			"Consider client device settings",
		)
	case AnomalyPingPong:
		recommendations = append(recommendations,
			"Adjust roaming thresholds",
			"Check for overlapping coverage areas",
			"Review band steering configuration",
			"Consider disabling fast roaming temporarily",
		)
	case AnomalySignalAnomaly:
		recommendations = append(recommendations,
			"Check for interference sources",
			"Verify AP antenna alignment",
			"Consider AP power adjustment",
			"Check for hardware issues",
		)
	}

	return recommendations
}

// Placeholder implementations for ML models
type IsolationForestModel struct{}
type OneClassSVMModel struct{}
type LSTMModel struct{}

type StatisticalAnomalyModel struct{}

func (sam *StatisticalAnomalyModel) PredictAnomalyScore(features []float64) float64 {
	// Simple statistical anomaly score based on feature deviations
	if len(features) == 0 {
		return 0
	}

	mean := 0.0
	for _, f := range features {
		mean += f
	}
	mean /= float64(len(features))

	variance := 0.0
	for _, f := range features {
		diff := f - mean
		variance += diff * diff
	}
	variance /= float64(len(features))

	// Simple anomaly score based on variance
	return math.Min(variance/10.0, 1.0)
}

func (rad *RoamingAnomalyDetector) initializeModels() {
	rad.models = AnomalyDetectionModels{
		StatisticalModel: &StatisticalAnomalyModel{},
	}
}

func (rad *RoamingAnomalyDetector) initializeAlertRules() {
	// Initialize default alert rules
	rad.alertRules = []AnomalyAlertRule{
		{
			ID:             "high_severity_alerts",
			Name:           "High Severity Anomaly Alerts",
			AnomalyTypes:   []AnomalyType{AnomalyExcessiveRoaming, AnomalyPingPong},
			SeverityFilter: SeverityHigh,
			AlertThreshold: 1,
			Enabled:        true,
		},
	}
}

func (rad *RoamingAnomalyDetector) updateBaselines() {
	// Update existing baselines with new data
	// Implementation would refresh baseline profiles periodically
}

func (rad *RoamingAnomalyDetector) processAlerts() {
	// Process alert rules and send notifications
	// Implementation would check alert rules and trigger actions
}

func (rad *RoamingAnomalyDetector) cleanupOldData() {
	// Clean up old anomalies and baselines
	now := time.Now()
	cutoff := now.Add(-rad.config.AnomalyRetention)

	rad.mu.Lock()
	defer rad.mu.Unlock()

	for id, anomaly := range rad.detectedAnomalies {
		if anomaly.Status == StatusResolved && anomaly.LastOccurrence.Before(cutoff) {
			delete(rad.detectedAnomalies, id)
		}
	}

	log.Printf("Cleaned up old anomaly data")
}

func (rad *RoamingAnomalyDetector) loadExistingBaselines() error {
	// Load existing baseline profiles from storage
	log.Printf("Loading existing baseline profiles from storage")
	// TODO: Implement loading from storage
	return nil
}
