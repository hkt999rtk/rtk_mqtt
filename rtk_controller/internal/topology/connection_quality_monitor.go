package topology

import (
	"context"
	"fmt"
	"log"
	"math"
	"sync"
	"time"

	"rtk_controller/internal/storage"
)

// ConnectionQualityMonitor monitors and analyzes connection quality for all network devices
type ConnectionQualityMonitor struct {
	// Data sources
	topologyManager     *Manager
	connectionTracker   *ConnectionHistoryTracker
	wifiCollector      *WiFiClientCollector
	realtimeUpdater    *RealtimeTopologyUpdater
	storage            *storage.TopologyStorage
	identityStorage    *storage.IdentityStorage
	
	// Quality tracking
	connectionMetrics  map[string]*ConnectionMetrics
	qualityHistory     map[string][]QualitySnapshot
	thresholds         QualityThresholds
	alerts             []QualityAlert
	mu                 sync.RWMutex
	
	// Configuration
	config QualityMonitorConfig
	
	// Background processing
	running bool
	cancel  context.CancelFunc
	
	// Statistics
	stats QualityMonitorStats
}

// ConnectionMetrics holds comprehensive quality metrics for a connection
type ConnectionMetrics struct {
	DeviceID           string
	MacAddress         string
	FriendlyName       string
	InterfaceType      string // wifi, ethernet, bridge
	
	// Signal metrics (for WiFi)
	SignalStrength     SignalMetrics
	
	// Performance metrics
	Throughput         ThroughputMetrics
	Latency           LatencyMetrics
	PacketLoss        PacketLossMetrics
	Jitter            JitterMetrics
	
	// Connection stability
	Stability         StabilityMetrics
	Reliability       ReliabilityMetrics
	
	// Quality scores
	OverallQuality    QualityScore
	TrendAnalysis     QualityTrend
	
	// Monitoring state
	LastUpdate        time.Time
	MonitoringActive  bool
	AlertsGenerated   int
}

// SignalMetrics holds WiFi signal-related metrics
type SignalMetrics struct {
	CurrentRSSI       int
	AverageRSSI       int
	MinRSSI          int
	MaxRSSI          int
	SignalVariance   float64
	SNR              float64 // Signal-to-Noise Ratio
	NoiseFloor       int
	SignalQuality    float64 // 0-1 scale
	FrequencyBand    string  // 2.4G, 5G, 6G
	Channel          int
	Bandwidth        int     // MHz
}

// ThroughputMetrics holds bandwidth and throughput data
type ThroughputMetrics struct {
	CurrentThroughputMbps  float64
	AverageThroughputMbps  float64
	PeakThroughputMbps     float64
	DownloadSpeedMbps      float64
	UploadSpeedMbps        float64
	ThroughputEfficiency   float64 // actual vs theoretical max
	BandwidthUtilization   float64 // percentage of available bandwidth
	ThroughputStability    float64 // consistency over time
}

// LatencyMetrics holds latency and response time data
type LatencyMetrics struct {
	CurrentLatencyMs    float64
	AverageLatencyMs    float64
	MinLatencyMs        float64
	MaxLatencyMs        float64
	LatencyVariance     float64
	RTTMs              float64 // Round-trip time
	DNSResolutionMs     float64
	TCPHandshakeMs      float64
	FirstByteMs         float64
}

// PacketLossMetrics holds packet loss statistics
type PacketLossMetrics struct {
	CurrentLossRate     float64 // percentage
	AverageLossRate     float64
	MaxLossRate         float64
	TotalPacketsSent    int64
	TotalPacketsLost    int64
	LossPatterns        []LossPattern
	RetransmissionRate  float64
	ErrorRate           float64
}

// JitterMetrics holds jitter and timing variation data
type JitterMetrics struct {
	CurrentJitterMs     float64
	AverageJitterMs     float64
	MaxJitterMs         float64
	JitterVariance      float64
	PacketDelayVariation float64
	TimingConsistency   float64 // 0-1 scale
}

// StabilityMetrics tracks connection stability
type StabilityMetrics struct {
	ConnectionUptime    time.Duration
	DisconnectionCount  int
	ReconnectionTime    time.Duration
	StabilityScore      float64 // 0-1 scale
	FlappingDetected    bool
	QualityConsistency  float64
	PerformanceVariation float64
}

// ReliabilityMetrics tracks connection reliability
// ReliabilityMetrics moved to connection_history.go to avoid duplication

// QualityScore represents overall connection quality
type QualityScore struct {
	Overall            float64 // 0-1 scale
	Signal             float64
	Performance        float64
	Stability          float64
	Reliability        float64
	UserExperience     float64
	Grade              QualityGrade
	LastCalculated     time.Time
}

// QualityTrend analyzes quality trends over time
type QualityTrend struct {
	TrendDirection     TrendDirection
	TrendStrength      float64 // 0-1 scale
	RecentImprovement  bool
	RecentDegradation  bool
	PredictedQuality   float64
	TrendAnalysis      string
	Confidence         float64
}

// QualitySnapshot captures quality at a specific point in time
type QualitySnapshot struct {
	Timestamp         time.Time
	OverallQuality    float64
	SignalQuality     float64
	ThroughputQuality float64
	LatencyQuality    float64
	StabilityQuality  float64
	Context           SnapshotContext
}

// SnapshotContext provides context for quality measurements
type SnapshotContext struct {
	NetworkLoad       float64
	UserCount         int
	TimeOfDay         string
	DayOfWeek         string
	WeatherCondition  string
	InterferenceLevel int
}

// QualityAlert represents a quality-related alert
type QualityAlert struct {
	ID                string
	Type              AlertType
	Severity          AlertSeverity
	DeviceID          string
	MacAddress        string
	FriendlyName      string
	Metric            string
	Threshold         float64
	CurrentValue      float64
	Message           string
	FirstDetected     time.Time
	LastOccurrence    time.Time
	Frequency         int
	Resolved          bool
	ResolutionTime    time.Time
	Details           map[string]interface{}
}

// LossPattern represents packet loss patterns
type LossPattern struct {
	Type              LossPatternType
	StartTime         time.Time
	Duration          time.Duration
	LossRate          float64
	AffectedTraffic   string
	Cause             string
}

// QualityThresholds defines quality thresholds for monitoring
type QualityThresholds struct {
	// Signal thresholds (dBm)
	ExcellentSignal   int     // -50
	GoodSignal        int     // -60
	FairSignal        int     // -70
	PoorSignal        int     // -80
	
	// Throughput thresholds (Mbps)
	ExcellentThroughput float64 // 100
	GoodThroughput      float64 // 50
	FairThroughput      float64 // 10
	PoorThroughput      float64 // 1
	
	// Latency thresholds (ms)
	ExcellentLatency    float64 // 10
	GoodLatency         float64 // 50
	FairLatency         float64 // 100
	PoorLatency         float64 // 200
	
	// Packet loss thresholds (%)
	ExcellentPacketLoss float64 // 0.1
	GoodPacketLoss      float64 // 0.5
	FairPacketLoss      float64 // 1.0
	PoorPacketLoss      float64 // 3.0
	
	// Jitter thresholds (ms)
	ExcellentJitter     float64 // 5
	GoodJitter          float64 // 20
	FairJitter          float64 // 50
	PoorJitter          float64 // 100
	
	// Stability thresholds
	MinUptimeForStable  time.Duration // 1 hour
	MaxDisconnectsPerHour int         // 3
	MinStabilityScore   float64       // 0.8
	
	// Alert thresholds
	QualityDegradationThreshold float64 // 0.2
	PerformanceDropThreshold    float64 // 0.3
	AlertCooldownPeriod        time.Duration // 5 minutes
}

// Enums for quality monitoring
type QualityGrade string
const (
	GradeExcellent QualityGrade = "excellent"
	GradeGood      QualityGrade = "good"
	GradeFair      QualityGrade = "fair"
	GradePoor      QualityGrade = "poor"
	GradeCritical  QualityGrade = "critical"
)

type TrendDirection string
const (
	TrendImproving  TrendDirection = "improving"
	TrendStable     TrendDirection = "stable"
	TrendDegrading  TrendDirection = "degrading"
	TrendVolatile   TrendDirection = "volatile"
	TrendUnknown    TrendDirection = "unknown"
)

type AlertType string
const (
	AlertQualityDegradation  AlertType = "quality_degradation"
	AlertSignalWeak         AlertType = "signal_weak"
	AlertHighLatency        AlertType = "high_latency"
	AlertHighPacketLoss     AlertType = "high_packet_loss"
	AlertLowThroughput      AlertType = "low_throughput"
	AlertHighJitter         AlertType = "high_jitter"
	AlertConnectionUnstable AlertType = "connection_unstable"
	AlertPerformanceDrop    AlertType = "performance_drop"
	AlertAnomalousPattern   AlertType = "anomalous_pattern"
)

type AlertSeverity string
// Severity constants moved to constants.go
// Use the package-level constants instead

type LossPatternType string
const (
	PatternBurst        LossPatternType = "burst"
	PatternRandom       LossPatternType = "random"
	PatternPeriodic     LossPatternType = "periodic"
	PatternProgressive  LossPatternType = "progressive"
)

// QualityMonitorConfig holds monitor configuration
type QualityMonitorConfig struct {
	// Monitoring intervals
	QualityCheckInterval    time.Duration
	MetricsCollectionInterval time.Duration
	TrendAnalysisInterval   time.Duration
	AlertCheckInterval      time.Duration
	
	// Measurement settings
	PingTestInterval       time.Duration
	ThroughputTestInterval time.Duration
	SignalScanInterval     time.Duration
	EnableActiveTesting    bool
	EnablePassiveMonitoring bool
	
	// History settings
	QualityHistorySize     int
	MetricsRetention       time.Duration
	AlertRetention         time.Duration
	SnapshotInterval       time.Duration
	
	// Analysis settings
	EnableTrendAnalysis    bool
	EnablePredictiveAlerts bool
	EnableMLAnalysis       bool
	TrendWindowSize        int
	
	// Performance settings
	MaxConcurrentTests     int
	TestTimeoutSeconds     int
	BatchSize             int
	
	// Thresholds (can be overridden)
	CustomThresholds      *QualityThresholds
}

// QualityMonitorStats holds monitoring statistics
type QualityMonitorStats struct {
	MonitoredConnections   int64
	QualityChecksPerformed int64
	AlertsGenerated        int64
	TestsExecuted          int64
	AverageQualityScore    float64
	ConnectionsWithPoorQuality int64
	LastQualityCheck       time.Time
	ProcessingErrors       int64
}

// NewConnectionQualityMonitor creates a new connection quality monitor
func NewConnectionQualityMonitor(
	topologyManager *Manager,
	connectionTracker *ConnectionHistoryTracker,
	wifiCollector *WiFiClientCollector,
	realtimeUpdater *RealtimeTopologyUpdater,
	storage *storage.TopologyStorage,
	identityStorage *storage.IdentityStorage,
	config QualityMonitorConfig,
) *ConnectionQualityMonitor {
	
	monitor := &ConnectionQualityMonitor{
		topologyManager:   topologyManager,
		connectionTracker: connectionTracker,
		wifiCollector:    wifiCollector,
		realtimeUpdater:  realtimeUpdater,
		storage:          storage,
		identityStorage:  identityStorage,
		connectionMetrics: make(map[string]*ConnectionMetrics),
		qualityHistory:   make(map[string][]QualitySnapshot),
		alerts:          []QualityAlert{},
		config:          config,
		stats:           QualityMonitorStats{},
	}
	
	// Initialize default thresholds
	monitor.initializeThresholds()
	
	return monitor
}

// Start begins connection quality monitoring
func (cqm *ConnectionQualityMonitor) Start() error {
	cqm.mu.Lock()
	defer cqm.mu.Unlock()
	
	if cqm.running {
		return fmt.Errorf("connection quality monitor is already running")
	}
	
	ctx, cancel := context.WithCancel(context.Background())
	cqm.cancel = cancel
	cqm.running = true
	
	log.Printf("Starting connection quality monitor")
	
	// Start background monitoring loops
	go cqm.qualityMonitoringLoop(ctx)
	go cqm.metricsCollectionLoop(ctx)
	go cqm.trendAnalysisLoop(ctx)
	go cqm.alertProcessingLoop(ctx)
	go cqm.cleanupLoop(ctx)
	
	// Initialize monitoring for existing connections
	if err := cqm.initializeExistingConnections(); err != nil {
		log.Printf("Failed to initialize existing connections: %v", err)
	}
	
	return nil
}

// Stop stops connection quality monitoring
func (cqm *ConnectionQualityMonitor) Stop() error {
	cqm.mu.Lock()
	defer cqm.mu.Unlock()
	
	if !cqm.running {
		return fmt.Errorf("connection quality monitor is not running")
	}
	
	cqm.cancel()
	cqm.running = false
	
	log.Printf("Connection quality monitor stopped")
	return nil
}

// MonitorConnection starts monitoring quality for a specific connection
func (cqm *ConnectionQualityMonitor) MonitorConnection(deviceID, macAddress string) error {
	cqm.mu.Lock()
	defer cqm.mu.Unlock()
	
	key := fmt.Sprintf("%s-%s", deviceID, macAddress)
	
	// Get friendly name
	friendlyName := macAddress
	if identity, err := cqm.identityStorage.GetDeviceIdentity(macAddress); err == nil {
		friendlyName = identity.FriendlyName
	}
	
	// Create connection metrics
	metrics := &ConnectionMetrics{
		DeviceID:         deviceID,
		MacAddress:       macAddress,
		FriendlyName:     friendlyName,
		LastUpdate:       time.Now(),
		MonitoringActive: true,
	}
	
	// Initialize metrics
	cqm.initializeConnectionMetrics(metrics)
	
	cqm.connectionMetrics[key] = metrics
	cqm.stats.MonitoredConnections++
	
	log.Printf("Started quality monitoring for connection: %s (%s)", friendlyName, macAddress)
	return nil
}

// StopMonitoring stops monitoring a specific connection
func (cqm *ConnectionQualityMonitor) StopMonitoring(deviceID, macAddress string) error {
	cqm.mu.Lock()
	defer cqm.mu.Unlock()
	
	key := fmt.Sprintf("%s-%s", deviceID, macAddress)
	
	if metrics, exists := cqm.connectionMetrics[key]; exists {
		metrics.MonitoringActive = false
		cqm.stats.MonitoredConnections--
		log.Printf("Stopped quality monitoring for connection: %s", metrics.FriendlyName)
	}
	
	return nil
}

// GetConnectionQuality returns quality metrics for a specific connection
func (cqm *ConnectionQualityMonitor) GetConnectionQuality(deviceID, macAddress string) (*ConnectionMetrics, bool) {
	cqm.mu.RLock()
	defer cqm.mu.RUnlock()
	
	key := fmt.Sprintf("%s-%s", deviceID, macAddress)
	metrics, exists := cqm.connectionMetrics[key]
	
	if !exists {
		return nil, false
	}
	
	// Return copy
	metricsCopy := *metrics
	return &metricsCopy, true
}

// GetQualityHistory returns quality history for a connection
func (cqm *ConnectionQualityMonitor) GetQualityHistory(deviceID, macAddress string, since time.Time) []QualitySnapshot {
	cqm.mu.RLock()
	defer cqm.mu.RUnlock()
	
	key := fmt.Sprintf("%s-%s", deviceID, macAddress)
	history, exists := cqm.qualityHistory[key]
	
	if !exists {
		return []QualitySnapshot{}
	}
	
	var filteredHistory []QualitySnapshot
	for _, snapshot := range history {
		if snapshot.Timestamp.After(since) {
			filteredHistory = append(filteredHistory, snapshot)
		}
	}
	
	return filteredHistory
}

// GetQualityAlerts returns quality alerts
func (cqm *ConnectionQualityMonitor) GetQualityAlerts(resolved bool) []QualityAlert {
	cqm.mu.RLock()
	defer cqm.mu.RUnlock()
	
	var alerts []QualityAlert
	for _, alert := range cqm.alerts {
		if alert.Resolved == resolved {
			alerts = append(alerts, alert)
		}
	}
	
	return alerts
}

// GetAllConnectionQuality returns quality metrics for all monitored connections
func (cqm *ConnectionQualityMonitor) GetAllConnectionQuality() map[string]*ConnectionMetrics {
	cqm.mu.RLock()
	defer cqm.mu.RUnlock()
	
	result := make(map[string]*ConnectionMetrics)
	for key, metrics := range cqm.connectionMetrics {
		if metrics.MonitoringActive {
			metricsCopy := *metrics
			result[key] = &metricsCopy
		}
	}
	
	return result
}

// GetQualityReport generates a comprehensive quality report
func (cqm *ConnectionQualityMonitor) GetQualityReport(period time.Duration) (*QualityReport, error) {
	since := time.Now().Add(-period)
	
	report := &QualityReport{
		Period:      period,
		GeneratedAt: time.Now(),
		Connections: make(map[string]ConnectionQualitySummary),
	}
	
	cqm.mu.RLock()
	defer cqm.mu.RUnlock()
	
	// Analyze each connection
	var totalQuality float64
	var connectionCount int
	
	for key, metrics := range cqm.connectionMetrics {
		if !metrics.MonitoringActive {
			continue
		}
		
		summary := cqm.generateConnectionSummary(metrics, since)
		report.Connections[key] = summary
		
		totalQuality += summary.AverageQuality
		connectionCount++
		
		if summary.AverageQuality < 0.5 {
			report.PoorQualityConnections++
		}
	}
	
	if connectionCount > 0 {
		report.OverallAverageQuality = totalQuality / float64(connectionCount)
	}
	
	// Add alerts summary
	report.ActiveAlerts = len(cqm.GetQualityAlerts(false))
	report.ResolvedAlerts = len(cqm.GetQualityAlerts(true))
	
	return report, nil
}

// QualityReport holds comprehensive quality analysis
type QualityReport struct {
	Period                 time.Duration
	GeneratedAt            time.Time
	OverallAverageQuality  float64
	Connections           map[string]ConnectionQualitySummary
	PoorQualityConnections int
	ActiveAlerts          int
	ResolvedAlerts        int
	Recommendations       []string
}

// ConnectionQualitySummary holds summary quality information for a connection
type ConnectionQualitySummary struct {
	DeviceID           string
	FriendlyName       string
	AverageQuality     float64
	MinQuality         float64
	MaxQuality         float64
	CurrentGrade       QualityGrade
	TrendDirection     TrendDirection
	AlertCount         int
	IssuesIdentified   []string
	Recommendations    []string
}

// GetStats returns quality monitor statistics
func (cqm *ConnectionQualityMonitor) GetStats() QualityMonitorStats {
	cqm.mu.RLock()
	defer cqm.mu.RUnlock()
	
	stats := cqm.stats
	
	// Calculate average quality score
	var totalQuality float64
	var count int
	
	for _, metrics := range cqm.connectionMetrics {
		if metrics.MonitoringActive {
			totalQuality += metrics.OverallQuality.Overall
			count++
		}
	}
	
	if count > 0 {
		stats.AverageQualityScore = totalQuality / float64(count)
	}
	
	// Count poor quality connections
	stats.ConnectionsWithPoorQuality = 0
	for _, metrics := range cqm.connectionMetrics {
		if metrics.MonitoringActive && metrics.OverallQuality.Overall < 0.5 {
			stats.ConnectionsWithPoorQuality++
		}
	}
	
	return stats
}

// Private methods

func (cqm *ConnectionQualityMonitor) initializeThresholds() {
	if cqm.config.CustomThresholds != nil {
		cqm.thresholds = *cqm.config.CustomThresholds
	} else {
		// Default thresholds
		cqm.thresholds = QualityThresholds{
			ExcellentSignal:   -50,
			GoodSignal:        -60,
			FairSignal:        -70,
			PoorSignal:        -80,
			ExcellentThroughput: 100,
			GoodThroughput:      50,
			FairThroughput:      10,
			PoorThroughput:      1,
			ExcellentLatency:    10,
			GoodLatency:         50,
			FairLatency:         100,
			PoorLatency:         200,
			ExcellentPacketLoss: 0.1,
			GoodPacketLoss:      0.5,
			FairPacketLoss:      1.0,
			PoorPacketLoss:      3.0,
			ExcellentJitter:     5,
			GoodJitter:          20,
			FairJitter:          50,
			PoorJitter:          100,
			MinUptimeForStable:  time.Hour,
			MaxDisconnectsPerHour: 3,
			MinStabilityScore:   0.8,
			QualityDegradationThreshold: 0.2,
			PerformanceDropThreshold:    0.3,
			AlertCooldownPeriod:        5 * time.Minute,
		}
	}
}

func (cqm *ConnectionQualityMonitor) initializeConnectionMetrics(metrics *ConnectionMetrics) {
	// Initialize all metric structures with default values
	metrics.SignalStrength = SignalMetrics{
		SignalQuality: 0.5,
	}
	
	metrics.Throughput = ThroughputMetrics{
		ThroughputEfficiency: 0.5,
		BandwidthUtilization: 0.5,
		ThroughputStability:  0.5,
	}
	
	metrics.Latency = LatencyMetrics{
		CurrentLatencyMs: 50,
		AverageLatencyMs: 50,
		MinLatencyMs:     10,
		MaxLatencyMs:     100,
	}
	
	metrics.PacketLoss = PacketLossMetrics{
		LossPatterns: []LossPattern{},
	}
	
	metrics.Jitter = JitterMetrics{
		TimingConsistency: 0.5,
	}
	
	metrics.Stability = StabilityMetrics{
		StabilityScore:     0.5,
		QualityConsistency: 0.5,
	}
	
	metrics.Reliability = ReliabilityMetrics{
		// ReliabilityScore: 0.5,
		ConnectionSuccess: 0.9,
	}
	
	metrics.OverallQuality = QualityScore{
		Overall:     0.5,
		Signal:      0.5,
		Performance: 0.5,
		Stability:   0.5,
		Reliability: 0.5,
		Grade:       GradeFair,
	}
	
	metrics.TrendAnalysis = QualityTrend{
		TrendDirection: TrendUnknown,
		Confidence:     0.5,
	}
}

func (cqm *ConnectionQualityMonitor) qualityMonitoringLoop(ctx context.Context) {
	ticker := time.NewTicker(cqm.config.QualityCheckInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			cqm.performQualityCheck()
		}
	}
}

func (cqm *ConnectionQualityMonitor) metricsCollectionLoop(ctx context.Context) {
	ticker := time.NewTicker(cqm.config.MetricsCollectionInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			cqm.collectMetrics()
		}
	}
}

func (cqm *ConnectionQualityMonitor) trendAnalysisLoop(ctx context.Context) {
	if !cqm.config.EnableTrendAnalysis {
		return
	}
	
	ticker := time.NewTicker(cqm.config.TrendAnalysisInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			cqm.analyzeTrends()
		}
	}
}

func (cqm *ConnectionQualityMonitor) alertProcessingLoop(ctx context.Context) {
	ticker := time.NewTicker(cqm.config.AlertCheckInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			cqm.processAlerts()
		}
	}
}

func (cqm *ConnectionQualityMonitor) cleanupLoop(ctx context.Context) {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			cqm.cleanupOldData()
		}
	}
}

func (cqm *ConnectionQualityMonitor) performQualityCheck() {
	cqm.mu.Lock()
	defer cqm.mu.Unlock()
	
	for key, metrics := range cqm.connectionMetrics {
		if !metrics.MonitoringActive {
			continue
		}
		
		// Update quality metrics
		cqm.updateQualityMetrics(metrics)
		
		// Calculate overall quality score
		cqm.calculateOverallQuality(metrics)
		
		// Create quality snapshot
		cqm.createQualitySnapshot(key, metrics)
		
		cqm.stats.QualityChecksPerformed++
	}
	
	cqm.stats.LastQualityCheck = time.Now()
}

func (cqm *ConnectionQualityMonitor) collectMetrics() {
	// Collect metrics from various sources
	
	// Collect WiFi signal metrics
	cqm.collectWiFiMetrics()
	
	// Collect performance metrics
	if cqm.config.EnableActiveTesting {
		cqm.collectPerformanceMetrics()
	}
	
	// Collect stability metrics
	cqm.collectStabilityMetrics()
}

func (cqm *ConnectionQualityMonitor) collectWiFiMetrics() {
	// Get WiFi client states from collector
	// TODO: Implement GetAllClientStates method
	clientStates := make(map[string]interface{}) // cqm.wifiCollector.GetAllClientStates()
	
	for macAddress, _ := range clientStates {
		key := fmt.Sprintf("%s-%s", "ap", macAddress)
		
		if metrics, exists := cqm.connectionMetrics[key]; exists && metrics.MonitoringActive {
			// Update signal metrics
			// TODO: Get actual signal strength from clientState
			// metrics.SignalStrength.CurrentRSSI = clientState.SignalStrength
			// metrics.SignalStrength.SignalQuality = cqm.calculateSignalQuality(clientState.SignalStrength)
			// TODO: Get actual frequency band and channel from clientState
			// metrics.SignalStrength.FrequencyBand = clientState.FrequencyBand
			// metrics.SignalStrength.Channel = clientState.Channel
			
			// Update signal history for variance calculation
			// TODO: Get actual signal strength from clientState
			// cqm.updateSignalHistory(metrics, clientState.SignalStrength)
			
			metrics.LastUpdate = time.Now()
		}
	}
}

func (cqm *ConnectionQualityMonitor) collectPerformanceMetrics() {
	// Collect throughput, latency, and other performance metrics
	// This would typically involve running network tests
	
	for _, metrics := range cqm.connectionMetrics {
		if !metrics.MonitoringActive {
			continue
		}
		
		// Simulate performance collection
		// In a real implementation, this would run actual network tests
		cqm.updatePerformanceMetrics(metrics)
	}
}

func (cqm *ConnectionQualityMonitor) collectStabilityMetrics() {
	// Collect stability and reliability metrics from connection history
	
	for _, metrics := range cqm.connectionMetrics {
		if !metrics.MonitoringActive {
			continue
		}
		
		// Get connection history
		if history, exists := cqm.connectionTracker.GetClientHistory(metrics.MacAddress); exists {
			cqm.updateStabilityFromHistory(metrics, history)
		}
	}
}

func (cqm *ConnectionQualityMonitor) updateQualityMetrics(metrics *ConnectionMetrics) {
	// Update various quality metrics
	
	// Calculate signal statistics
	// if len(metrics.SignalStrength.SignalQuality) > 0 {
	cqm.calculateSignalStatistics(metrics)
	// }
	
	// Update throughput statistics
	cqm.updateThroughputStatistics(metrics)
	
	// Update latency statistics
	cqm.updateLatencyStatistics(metrics)
	
	// Update packet loss statistics
	cqm.updatePacketLossStatistics(metrics)
	
	// Update jitter statistics
	cqm.updateJitterStatistics(metrics)
}

func (cqm *ConnectionQualityMonitor) calculateOverallQuality(metrics *ConnectionMetrics) {
	// Calculate overall quality score based on all metrics
	
	signalScore := cqm.calculateSignalScore(metrics)
	performanceScore := cqm.calculatePerformanceScore(metrics)
	stabilityScore := cqm.calculateStabilityScore(metrics)
	reliabilityScore := cqm.calculateReliabilityScore(metrics)
	
	// Weighted average
	overallScore := (signalScore*0.2 + performanceScore*0.4 + stabilityScore*0.2 + reliabilityScore*0.2)
	
	metrics.OverallQuality.Overall = overallScore
	metrics.OverallQuality.Signal = signalScore
	metrics.OverallQuality.Performance = performanceScore
	metrics.OverallQuality.Stability = stabilityScore
	metrics.OverallQuality.Reliability = reliabilityScore
	metrics.OverallQuality.LastCalculated = time.Now()
	
	// Assign grade
	metrics.OverallQuality.Grade = cqm.calculateQualityGrade(overallScore)
	
	// Calculate user experience score
	metrics.OverallQuality.UserExperience = cqm.calculateUserExperienceScore(metrics)
}

func (cqm *ConnectionQualityMonitor) calculateSignalScore(metrics *ConnectionMetrics) float64 {
	rssi := metrics.SignalStrength.CurrentRSSI
	
	if rssi >= cqm.thresholds.ExcellentSignal {
		return 1.0
	} else if rssi >= cqm.thresholds.GoodSignal {
		return 0.8
	} else if rssi >= cqm.thresholds.FairSignal {
		return 0.6
	} else if rssi >= cqm.thresholds.PoorSignal {
		return 0.4
	} else {
		return 0.2
	}
}

func (cqm *ConnectionQualityMonitor) calculatePerformanceScore(metrics *ConnectionMetrics) float64 {
	// Combine throughput and latency scores
	throughputScore := cqm.calculateThroughputScore(metrics.Throughput.CurrentThroughputMbps)
	latencyScore := cqm.calculateLatencyScore(metrics.Latency.CurrentLatencyMs)
	packetLossScore := cqm.calculatePacketLossScore(metrics.PacketLoss.CurrentLossRate)
	jitterScore := cqm.calculateJitterScore(metrics.Jitter.CurrentJitterMs)
	
	// Weighted average
	return (throughputScore*0.4 + latencyScore*0.3 + packetLossScore*0.2 + jitterScore*0.1)
}

func (cqm *ConnectionQualityMonitor) calculateStabilityScore(metrics *ConnectionMetrics) float64 {
	return metrics.Stability.StabilityScore
}

func (cqm *ConnectionQualityMonitor) calculateReliabilityScore(metrics *ConnectionMetrics) float64 {
	// TODO: Calculate reliability score from metrics
	return 0.5 // Default score
}

func (cqm *ConnectionQualityMonitor) calculateThroughputScore(throughput float64) float64 {
	if throughput >= cqm.thresholds.ExcellentThroughput {
		return 1.0
	} else if throughput >= cqm.thresholds.GoodThroughput {
		return 0.8
	} else if throughput >= cqm.thresholds.FairThroughput {
		return 0.6
	} else if throughput >= cqm.thresholds.PoorThroughput {
		return 0.4
	} else {
		return 0.2
	}
}

func (cqm *ConnectionQualityMonitor) calculateLatencyScore(latency float64) float64 {
	if latency <= cqm.thresholds.ExcellentLatency {
		return 1.0
	} else if latency <= cqm.thresholds.GoodLatency {
		return 0.8
	} else if latency <= cqm.thresholds.FairLatency {
		return 0.6
	} else if latency <= cqm.thresholds.PoorLatency {
		return 0.4
	} else {
		return 0.2
	}
}

func (cqm *ConnectionQualityMonitor) calculatePacketLossScore(lossRate float64) float64 {
	if lossRate <= cqm.thresholds.ExcellentPacketLoss {
		return 1.0
	} else if lossRate <= cqm.thresholds.GoodPacketLoss {
		return 0.8
	} else if lossRate <= cqm.thresholds.FairPacketLoss {
		return 0.6
	} else if lossRate <= cqm.thresholds.PoorPacketLoss {
		return 0.4
	} else {
		return 0.2
	}
}

func (cqm *ConnectionQualityMonitor) calculateJitterScore(jitter float64) float64 {
	if jitter <= cqm.thresholds.ExcellentJitter {
		return 1.0
	} else if jitter <= cqm.thresholds.GoodJitter {
		return 0.8
	} else if jitter <= cqm.thresholds.FairJitter {
		return 0.6
	} else if jitter <= cqm.thresholds.PoorJitter {
		return 0.4
	} else {
		return 0.2
	}
}

func (cqm *ConnectionQualityMonitor) calculateQualityGrade(score float64) QualityGrade {
	if score >= 0.9 {
		return GradeExcellent
	} else if score >= 0.7 {
		return GradeGood
	} else if score >= 0.5 {
		return GradeFair
	} else if score >= 0.3 {
		return GradePoor
	} else {
		return GradeCritical
	}
}

func (cqm *ConnectionQualityMonitor) calculateUserExperienceScore(metrics *ConnectionMetrics) float64 {
	// Calculate user experience based on application-specific requirements
	
	baseScore := metrics.OverallQuality.Overall
	
	// Penalties for poor performance
	if metrics.Latency.CurrentLatencyMs > 100 {
		baseScore -= 0.2
	}
	if metrics.PacketLoss.CurrentLossRate > 1.0 {
		baseScore -= 0.3
	}
	if metrics.Jitter.CurrentJitterMs > 50 {
		baseScore -= 0.1
	}
	
	// Ensure score is between 0 and 1
	if baseScore < 0 {
		baseScore = 0
	}
	if baseScore > 1 {
		baseScore = 1
	}
	
	return baseScore
}

func (cqm *ConnectionQualityMonitor) createQualitySnapshot(key string, metrics *ConnectionMetrics) {
	snapshot := QualitySnapshot{
		Timestamp:         time.Now(),
		OverallQuality:    metrics.OverallQuality.Overall,
		SignalQuality:     metrics.OverallQuality.Signal,
		ThroughputQuality: cqm.calculateThroughputScore(metrics.Throughput.CurrentThroughputMbps),
		LatencyQuality:    cqm.calculateLatencyScore(metrics.Latency.CurrentLatencyMs),
		StabilityQuality:  metrics.OverallQuality.Stability,
		Context:           cqm.buildSnapshotContext(),
	}
	
	// Add to history
	if _, exists := cqm.qualityHistory[key]; !exists {
		cqm.qualityHistory[key] = []QualitySnapshot{}
	}
	
	cqm.qualityHistory[key] = append(cqm.qualityHistory[key], snapshot)
	
	// Limit history size
	if len(cqm.qualityHistory[key]) > cqm.config.QualityHistorySize {
		cqm.qualityHistory[key] = cqm.qualityHistory[key][1:]
	}
}

func (cqm *ConnectionQualityMonitor) buildSnapshotContext() SnapshotContext {
	now := time.Now()
	
	return SnapshotContext{
		NetworkLoad:       0.5, // TODO: Get actual network load
		UserCount:         int(cqm.stats.MonitoredConnections),
		TimeOfDay:         fmt.Sprintf("%02d:00", now.Hour()),
		DayOfWeek:         now.Weekday().String(),
		InterferenceLevel: 0, // TODO: Calculate interference level
	}
}

func (cqm *ConnectionQualityMonitor) analyzeTrends() {
	for key, metrics := range cqm.connectionMetrics {
		if !metrics.MonitoringActive {
			continue
		}
		
		// Analyze quality trends
		if history, exists := cqm.qualityHistory[key]; exists && len(history) >= cqm.config.TrendWindowSize {
			cqm.updateQualityTrend(metrics, history)
		}
	}
}

func (cqm *ConnectionQualityMonitor) updateQualityTrend(metrics *ConnectionMetrics, history []QualitySnapshot) {
	if len(history) < 2 {
		return
	}
	
	// Calculate trend over recent history
	recentHistory := history
	if len(history) > cqm.config.TrendWindowSize {
		recentHistory = history[len(history)-cqm.config.TrendWindowSize:]
	}
	
	// Linear regression to determine trend
	trend := cqm.calculateLinearTrend(recentHistory)
	
	metrics.TrendAnalysis.TrendStrength = math.Abs(trend)
	metrics.TrendAnalysis.Confidence = cqm.calculateTrendConfidence(recentHistory)
	
	if trend > 0.01 {
		metrics.TrendAnalysis.TrendDirection = TrendImproving
		metrics.TrendAnalysis.RecentImprovement = true
		metrics.TrendAnalysis.RecentDegradation = false
	} else if trend < -0.01 {
		metrics.TrendAnalysis.TrendDirection = TrendDegrading
		metrics.TrendAnalysis.RecentImprovement = false
		metrics.TrendAnalysis.RecentDegradation = true
	} else {
		metrics.TrendAnalysis.TrendDirection = TrendStable
		metrics.TrendAnalysis.RecentImprovement = false
		metrics.TrendAnalysis.RecentDegradation = false
	}
	
	// Check for volatility
	variance := cqm.calculateVariance(recentHistory)
	if variance > 0.05 {
		metrics.TrendAnalysis.TrendDirection = TrendVolatile
	}
	
	// Predict future quality
	if len(recentHistory) > 0 {
		lastQuality := recentHistory[len(recentHistory)-1].OverallQuality
		metrics.TrendAnalysis.PredictedQuality = lastQuality + (trend * 5) // Predict 5 time periods ahead
	}
	
	// Generate trend analysis description
	metrics.TrendAnalysis.TrendAnalysis = cqm.generateTrendAnalysis(metrics.TrendAnalysis)
}

func (cqm *ConnectionQualityMonitor) calculateLinearTrend(history []QualitySnapshot) float64 {
	if len(history) < 2 {
		return 0
	}
	
	n := float64(len(history))
	var sumX, sumY, sumXY, sumX2 float64
	
	for i, snapshot := range history {
		x := float64(i)
		y := snapshot.OverallQuality
		
		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
	}
	
	// Calculate slope (trend)
	slope := (n*sumXY - sumX*sumY) / (n*sumX2 - sumX*sumX)
	
	return slope
}

func (cqm *ConnectionQualityMonitor) calculateTrendConfidence(history []QualitySnapshot) float64 {
	if len(history) < 3 {
		return 0.5
	}
	
	// Calculate R-squared for confidence
	trend := cqm.calculateLinearTrend(history)
	
	var totalVariation, explainedVariation float64
	var meanQuality float64
	
	// Calculate mean
	for _, snapshot := range history {
		meanQuality += snapshot.OverallQuality
	}
	meanQuality /= float64(len(history))
	
	// Calculate variations
	for i, snapshot := range history {
		predicted := trend * float64(i)
		totalVariation += math.Pow(snapshot.OverallQuality-meanQuality, 2)
		explainedVariation += math.Pow(predicted-meanQuality, 2)
	}
	
	if totalVariation == 0 {
		return 1.0
	}
	
	rSquared := explainedVariation / totalVariation
	return math.Max(0, math.Min(1, rSquared))
}

func (cqm *ConnectionQualityMonitor) calculateVariance(history []QualitySnapshot) float64 {
	if len(history) < 2 {
		return 0
	}
	
	var mean float64
	for _, snapshot := range history {
		mean += snapshot.OverallQuality
	}
	mean /= float64(len(history))
	
	var variance float64
	for _, snapshot := range history {
		variance += math.Pow(snapshot.OverallQuality-mean, 2)
	}
	variance /= float64(len(history))
	
	return variance
}

func (cqm *ConnectionQualityMonitor) generateTrendAnalysis(trend QualityTrend) string {
	switch trend.TrendDirection {
	case TrendImproving:
		return fmt.Sprintf("Connection quality is improving (confidence: %.0f%%)", trend.Confidence*100)
	case TrendDegrading:
		return fmt.Sprintf("Connection quality is degrading (confidence: %.0f%%)", trend.Confidence*100)
	case TrendStable:
		return fmt.Sprintf("Connection quality is stable (confidence: %.0f%%)", trend.Confidence*100)
	case TrendVolatile:
		return "Connection quality is volatile with significant fluctuations"
	default:
		return "Insufficient data for trend analysis"
	}
}

func (cqm *ConnectionQualityMonitor) processAlerts() {
	for _, metrics := range cqm.connectionMetrics {
		if !metrics.MonitoringActive {
			continue
		}
		
		// Check for quality degradation
		cqm.checkQualityDegradation(metrics)
		
		// Check threshold violations
		cqm.checkThresholdViolations(metrics)
		
		// Check trend-based alerts
		cqm.checkTrendAlerts(metrics)
	}
}

func (cqm *ConnectionQualityMonitor) checkQualityDegradation(metrics *ConnectionMetrics) {
	// Check for significant quality degradation
	if metrics.TrendAnalysis.RecentDegradation && 
	   metrics.TrendAnalysis.TrendStrength > cqm.thresholds.QualityDegradationThreshold {
		
		cqm.createQualityAlert(
			AlertQualityDegradation,
			metrics,
			"overall_quality",
			cqm.thresholds.QualityDegradationThreshold,
			metrics.OverallQuality.Overall,
			fmt.Sprintf("Connection quality degrading: %s", metrics.TrendAnalysis.TrendAnalysis),
			SeverityWarning,
		)
	}
}

func (cqm *ConnectionQualityMonitor) checkThresholdViolations(metrics *ConnectionMetrics) {
	// Check signal strength
	if metrics.SignalStrength.CurrentRSSI < cqm.thresholds.PoorSignal {
		cqm.createQualityAlert(
			AlertSignalWeak,
			metrics,
			"signal_strength",
			float64(cqm.thresholds.PoorSignal),
			float64(metrics.SignalStrength.CurrentRSSI),
			fmt.Sprintf("Signal strength is weak: %d dBm", metrics.SignalStrength.CurrentRSSI),
			SeverityWarning,
		)
	}
	
	// Check latency
	if metrics.Latency.CurrentLatencyMs > cqm.thresholds.PoorLatency {
		cqm.createQualityAlert(
			AlertHighLatency,
			metrics,
			"latency",
			cqm.thresholds.PoorLatency,
			metrics.Latency.CurrentLatencyMs,
			fmt.Sprintf("High latency detected: %.1f ms", metrics.Latency.CurrentLatencyMs),
			SeverityError,
		)
	}
	
	// Check packet loss
	if metrics.PacketLoss.CurrentLossRate > cqm.thresholds.PoorPacketLoss {
		cqm.createQualityAlert(
			AlertHighPacketLoss,
			metrics,
			"packet_loss",
			cqm.thresholds.PoorPacketLoss,
			metrics.PacketLoss.CurrentLossRate,
			fmt.Sprintf("High packet loss detected: %.1f%%", metrics.PacketLoss.CurrentLossRate),
			SeverityError,
		)
	}
	
	// Check throughput
	if metrics.Throughput.CurrentThroughputMbps < cqm.thresholds.PoorThroughput {
		cqm.createQualityAlert(
			AlertLowThroughput,
			metrics,
			"throughput",
			cqm.thresholds.PoorThroughput,
			metrics.Throughput.CurrentThroughputMbps,
			fmt.Sprintf("Low throughput detected: %.1f Mbps", metrics.Throughput.CurrentThroughputMbps),
			SeverityWarning,
		)
	}
	
	// Check jitter
	if metrics.Jitter.CurrentJitterMs > cqm.thresholds.PoorJitter {
		cqm.createQualityAlert(
			AlertHighJitter,
			metrics,
			"jitter",
			cqm.thresholds.PoorJitter,
			metrics.Jitter.CurrentJitterMs,
			fmt.Sprintf("High jitter detected: %.1f ms", metrics.Jitter.CurrentJitterMs),
			SeverityWarning,
		)
	}
}

func (cqm *ConnectionQualityMonitor) checkTrendAlerts(metrics *ConnectionMetrics) {
	// Check for performance drops
	if metrics.TrendAnalysis.TrendDirection == TrendDegrading &&
	   metrics.TrendAnalysis.TrendStrength > cqm.thresholds.PerformanceDropThreshold {
		
		cqm.createQualityAlert(
			AlertPerformanceDrop,
			metrics,
			"performance_trend",
			cqm.thresholds.PerformanceDropThreshold,
			metrics.TrendAnalysis.TrendStrength,
			fmt.Sprintf("Significant performance drop detected: %s", metrics.TrendAnalysis.TrendAnalysis),
			SeverityError,
		)
	}
	
	// Check for volatile connections
	if metrics.TrendAnalysis.TrendDirection == TrendVolatile {
		cqm.createQualityAlert(
			AlertConnectionUnstable,
			metrics,
			"stability",
			cqm.thresholds.MinStabilityScore,
			metrics.Stability.StabilityScore,
			"Connection showing volatile quality patterns",
			SeverityWarning,
		)
	}
}

func (cqm *ConnectionQualityMonitor) createQualityAlert(
	alertType AlertType,
	metrics *ConnectionMetrics,
	metric string,
	threshold float64,
	currentValue float64,
	message string,
	severity AlertSeverity,
) {
	
	now := time.Now()
	
	// Check if similar alert already exists recently
	for i, existing := range cqm.alerts {
		if existing.Type == alertType && 
		   existing.MacAddress == metrics.MacAddress && 
		   !existing.Resolved &&
		   now.Sub(existing.LastOccurrence) < cqm.thresholds.AlertCooldownPeriod {
			// Update existing alert
			cqm.alerts[i].LastOccurrence = now
			cqm.alerts[i].Frequency++
			cqm.alerts[i].CurrentValue = currentValue
			return
		}
	}
	
	// Create new alert
	alert := QualityAlert{
		ID:             fmt.Sprintf("quality_alert_%d_%s", now.UnixMilli(), metrics.MacAddress),
		Type:           alertType,
		Severity:       severity,
		DeviceID:       metrics.DeviceID,
		MacAddress:     metrics.MacAddress,
		FriendlyName:   metrics.FriendlyName,
		Metric:         metric,
		Threshold:      threshold,
		CurrentValue:   currentValue,
		Message:        message,
		FirstDetected:  now,
		LastOccurrence: now,
		Frequency:      1,
		Details:        make(map[string]interface{}),
	}
	
	cqm.alerts = append(cqm.alerts, alert)
	metrics.AlertsGenerated++
	cqm.stats.AlertsGenerated++
	
	log.Printf("Quality alert created: %s for %s (%s)", alertType, metrics.FriendlyName, message)
	
	// Publish alert through real-time updater
	if cqm.realtimeUpdater != nil {
		updateEvent := TopologyUpdateEvent{
			Type:      EventAnomalyDetected,
			DeviceID:  metrics.DeviceID,
			Timestamp: now,
			Priority:  PriorityHigh,
			Metadata: map[string]interface{}{
				"alert_type":    alertType,
				"alert_message": message,
				"severity":      severity,
			},
		}
		
		go cqm.realtimeUpdater.PublishUpdate(updateEvent)
	}
}

func (cqm *ConnectionQualityMonitor) cleanupOldData() {
	now := time.Now()
	
	cqm.mu.Lock()
	defer cqm.mu.Unlock()
	
	// Clean up old quality history
	for key, history := range cqm.qualityHistory {
		cutoff := now.Add(-cqm.config.MetricsRetention)
		var filteredHistory []QualitySnapshot
		
		for _, snapshot := range history {
			if snapshot.Timestamp.After(cutoff) {
				filteredHistory = append(filteredHistory, snapshot)
			}
		}
		
		cqm.qualityHistory[key] = filteredHistory
	}
	
	// Clean up old alerts
	cutoff := now.Add(-cqm.config.AlertRetention)
	var filteredAlerts []QualityAlert
	
	for _, alert := range cqm.alerts {
		if alert.LastOccurrence.After(cutoff) || !alert.Resolved {
			filteredAlerts = append(filteredAlerts, alert)
		}
	}
	
	cqm.alerts = filteredAlerts
	
	log.Printf("Cleaned up old quality monitoring data")
}

func (cqm *ConnectionQualityMonitor) initializeExistingConnections() error {
	// Initialize monitoring for existing active connections
	activeSessions := cqm.connectionTracker.GetActiveSessions()
	
	for _, session := range activeSessions {
		if err := cqm.MonitorConnection(session.DeviceID, session.MacAddress); err != nil {
			log.Printf("Failed to initialize monitoring for %s: %v", session.MacAddress, err)
		}
	}
	
	log.Printf("Initialized quality monitoring for %d existing connections", len(activeSessions))
	return nil
}

func (cqm *ConnectionQualityMonitor) generateConnectionSummary(metrics *ConnectionMetrics, since time.Time) ConnectionQualitySummary {
	// Get quality history for the period
	history := cqm.GetQualityHistory(metrics.DeviceID, metrics.MacAddress, since)
	
	summary := ConnectionQualitySummary{
		DeviceID:         metrics.DeviceID,
		FriendlyName:     metrics.FriendlyName,
		CurrentGrade:     metrics.OverallQuality.Grade,
		TrendDirection:   metrics.TrendAnalysis.TrendDirection,
		AlertCount:       metrics.AlertsGenerated,
		IssuesIdentified: []string{},
		Recommendations:  []string{},
	}
	
	if len(history) > 0 {
		// Calculate statistics
		var totalQuality, minQuality, maxQuality float64
		minQuality = 1.0
		
		for _, snapshot := range history {
			totalQuality += snapshot.OverallQuality
			if snapshot.OverallQuality < minQuality {
				minQuality = snapshot.OverallQuality
			}
			if snapshot.OverallQuality > maxQuality {
				maxQuality = snapshot.OverallQuality
			}
		}
		
		summary.AverageQuality = totalQuality / float64(len(history))
		summary.MinQuality = minQuality
		summary.MaxQuality = maxQuality
	} else {
		summary.AverageQuality = metrics.OverallQuality.Overall
		summary.MinQuality = metrics.OverallQuality.Overall
		summary.MaxQuality = metrics.OverallQuality.Overall
	}
	
	// Identify issues and recommendations
	cqm.identifyIssuesAndRecommendations(metrics, &summary)
	
	return summary
}

func (cqm *ConnectionQualityMonitor) identifyIssuesAndRecommendations(metrics *ConnectionMetrics, summary *ConnectionQualitySummary) {
	// Identify common issues
	if metrics.SignalStrength.CurrentRSSI < cqm.thresholds.FairSignal {
		summary.IssuesIdentified = append(summary.IssuesIdentified, "Weak WiFi signal")
		summary.Recommendations = append(summary.Recommendations, "Consider moving closer to AP or adding WiFi extender")
	}
	
	if metrics.Latency.CurrentLatencyMs > cqm.thresholds.FairLatency {
		summary.IssuesIdentified = append(summary.IssuesIdentified, "High latency")
		summary.Recommendations = append(summary.Recommendations, "Check network congestion and interference")
	}
	
	if metrics.PacketLoss.CurrentLossRate > cqm.thresholds.FairPacketLoss {
		summary.IssuesIdentified = append(summary.IssuesIdentified, "Packet loss detected")
		summary.Recommendations = append(summary.Recommendations, "Investigate network hardware and interference sources")
	}
	
	if metrics.Throughput.CurrentThroughputMbps < cqm.thresholds.FairThroughput {
		summary.IssuesIdentified = append(summary.IssuesIdentified, "Low throughput")
		summary.Recommendations = append(summary.Recommendations, "Check bandwidth allocation and network capacity")
	}
	
	if metrics.TrendAnalysis.TrendDirection == TrendDegrading {
		summary.IssuesIdentified = append(summary.IssuesIdentified, "Quality degrading over time")
		summary.Recommendations = append(summary.Recommendations, "Monitor for hardware issues or environmental changes")
	}
	
	if metrics.Stability.StabilityScore < 0.7 {
		summary.IssuesIdentified = append(summary.IssuesIdentified, "Connection instability")
		summary.Recommendations = append(summary.Recommendations, "Check for interference and connection reliability")
	}
}

// Helper methods for metric calculations

func (cqm *ConnectionQualityMonitor) calculateSignalQuality(rssi int) float64 {
	// Convert RSSI to quality score (0-1)
	if rssi >= -50 {
		return 1.0
	} else if rssi <= -90 {
		return 0.0
	} else {
		// Linear interpolation
		return float64(rssi+90) / 40.0
	}
}

func (cqm *ConnectionQualityMonitor) updateSignalHistory(metrics *ConnectionMetrics, rssi int) {
	// Update signal strength statistics
	if metrics.SignalStrength.MinRSSI == 0 || rssi < metrics.SignalStrength.MinRSSI {
		metrics.SignalStrength.MinRSSI = rssi
	}
	if rssi > metrics.SignalStrength.MaxRSSI {
		metrics.SignalStrength.MaxRSSI = rssi
	}
	
	// Update average (simple moving average for now)
	if metrics.SignalStrength.AverageRSSI == 0 {
		metrics.SignalStrength.AverageRSSI = rssi
	} else {
		metrics.SignalStrength.AverageRSSI = (metrics.SignalStrength.AverageRSSI + rssi) / 2
	}
}

func (cqm *ConnectionQualityMonitor) updatePerformanceMetrics(metrics *ConnectionMetrics) {
	// Simulate performance metric updates
	// In a real implementation, this would collect actual measurements
	
	// Update throughput metrics (placeholder values)
	metrics.Throughput.CurrentThroughputMbps = 25.0 + (math.Mod(float64(time.Now().Unix()), 50)) // 25-75 Mbps
	if metrics.Throughput.AverageThroughputMbps == 0 {
		metrics.Throughput.AverageThroughputMbps = metrics.Throughput.CurrentThroughputMbps
	} else {
		metrics.Throughput.AverageThroughputMbps = (metrics.Throughput.AverageThroughputMbps + metrics.Throughput.CurrentThroughputMbps) / 2
	}
	
	// Update latency metrics (placeholder values)
	metrics.Latency.CurrentLatencyMs = 20.0 + (math.Mod(float64(time.Now().Unix()), 30)) // 20-50 ms
	if metrics.Latency.AverageLatencyMs == 0 {
		metrics.Latency.AverageLatencyMs = metrics.Latency.CurrentLatencyMs
	} else {
		metrics.Latency.AverageLatencyMs = (metrics.Latency.AverageLatencyMs + metrics.Latency.CurrentLatencyMs) / 2
	}
	
	// Update packet loss (placeholder values)
	metrics.PacketLoss.CurrentLossRate = math.Mod(float64(time.Now().Unix()), 2.0) // 0-2%
	
	// Update jitter (placeholder values)
	metrics.Jitter.CurrentJitterMs = 5.0 + (math.Mod(float64(time.Now().Unix()), 15)) // 5-20 ms
}

func (cqm *ConnectionQualityMonitor) updateStabilityFromHistory(metrics *ConnectionMetrics, history *ClientConnectionHistory) {
	// Calculate stability metrics from connection history
	
	if len(history.Sessions) > 0 {
		// Calculate uptime and disconnection patterns
		var totalUptime time.Duration
		disconnections := 0
		
		for _, session := range history.Sessions {
			totalUptime += session.Duration
			if session.DisconnectReason != "" && session.DisconnectReason != "normal" {
				disconnections++
			}
		}
		
		// Update stability score based on session quality
		avgSessionDuration := totalUptime / time.Duration(len(history.Sessions))
		if avgSessionDuration > cqm.thresholds.MinUptimeForStable {
			metrics.Stability.StabilityScore = 0.8
		} else {
			metrics.Stability.StabilityScore = 0.6
		}
		
		// Adjust for disconnection frequency
		if disconnections > 0 {
			disconnectionRate := float64(disconnections) / float64(len(history.Sessions))
			metrics.Stability.StabilityScore *= (1.0 - disconnectionRate*0.5)
		}
		
		// Update reliability metrics
// 		metrics.Reliability.SuccessRate = 1.0 - float64(disconnections)/float64(len(history.Sessions))
// 		metrics.Reliability.ReliabilityScore = metrics.Reliability.SuccessRate * 0.8
		
		if totalUptime > 0 {
			// TODO: Add AvailabilityPercent field to ReliabilityMetrics
			// metrics.Reliability.AvailabilityPercent = float64(totalUptime) / float64(time.Since(history.FirstSeen)) * 100
		}
	}
}

func (cqm *ConnectionQualityMonitor) calculateSignalStatistics(metrics *ConnectionMetrics) {
	// Calculate signal variance and other statistics
	// This is a placeholder - in real implementation, would maintain signal history
	
	rssiRange := metrics.SignalStrength.MaxRSSI - metrics.SignalStrength.MinRSSI
	if rssiRange > 20 {
		metrics.SignalStrength.SignalVariance = float64(rssiRange) / 10.0
	} else {
		metrics.SignalStrength.SignalVariance = 2.0
	}
}

func (cqm *ConnectionQualityMonitor) updateThroughputStatistics(metrics *ConnectionMetrics) {
	// Update throughput-related statistics
	
	current := metrics.Throughput.CurrentThroughputMbps
	if current > metrics.Throughput.PeakThroughputMbps {
		metrics.Throughput.PeakThroughputMbps = current
	}
	
	// Calculate efficiency (placeholder)
	theoreticalMax := 100.0 // Assume 100 Mbps theoretical max
	metrics.Throughput.ThroughputEfficiency = current / theoreticalMax
	
	// Calculate stability (placeholder)
	if metrics.Throughput.AverageThroughputMbps > 0 {
		deviation := math.Abs(current - metrics.Throughput.AverageThroughputMbps)
		metrics.Throughput.ThroughputStability = 1.0 - (deviation / metrics.Throughput.AverageThroughputMbps)
	}
}

func (cqm *ConnectionQualityMonitor) updateLatencyStatistics(metrics *ConnectionMetrics) {
	// Update latency-related statistics
	
	current := metrics.Latency.CurrentLatencyMs
	if metrics.Latency.MinLatencyMs == 0 || current < metrics.Latency.MinLatencyMs {
		metrics.Latency.MinLatencyMs = current
	}
	if current > metrics.Latency.MaxLatencyMs {
		metrics.Latency.MaxLatencyMs = current
	}
	
	// Calculate variance (placeholder)
	if metrics.Latency.AverageLatencyMs > 0 {
		deviation := math.Abs(current - metrics.Latency.AverageLatencyMs)
		metrics.Latency.LatencyVariance = deviation * deviation
	}
}

func (cqm *ConnectionQualityMonitor) updatePacketLossStatistics(metrics *ConnectionMetrics) {
	// Update packet loss statistics
	
	current := metrics.PacketLoss.CurrentLossRate
	if current > metrics.PacketLoss.MaxLossRate {
		metrics.PacketLoss.MaxLossRate = current
	}
	
	// Update average
	if metrics.PacketLoss.AverageLossRate == 0 {
		metrics.PacketLoss.AverageLossRate = current
	} else {
		metrics.PacketLoss.AverageLossRate = (metrics.PacketLoss.AverageLossRate + current) / 2
	}
	
	// Simulate packet counts (placeholder)
	metrics.PacketLoss.TotalPacketsSent += 100
	lostPackets := int64(float64(100) * current / 100.0)
	metrics.PacketLoss.TotalPacketsLost += lostPackets
}

func (cqm *ConnectionQualityMonitor) updateJitterStatistics(metrics *ConnectionMetrics) {
	// Update jitter statistics
	
	current := metrics.Jitter.CurrentJitterMs
	if current > metrics.Jitter.MaxJitterMs {
		metrics.Jitter.MaxJitterMs = current
	}
	
	// Update average
	if metrics.Jitter.AverageJitterMs == 0 {
		metrics.Jitter.AverageJitterMs = current
	} else {
		metrics.Jitter.AverageJitterMs = (metrics.Jitter.AverageJitterMs + current) / 2
	}
	
	// Calculate consistency (placeholder)
	if metrics.Jitter.AverageJitterMs > 0 {
		variation := math.Abs(current - metrics.Jitter.AverageJitterMs)
		metrics.Jitter.TimingConsistency = 1.0 - (variation / metrics.Jitter.AverageJitterMs)
	}
}