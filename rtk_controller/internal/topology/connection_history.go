package topology

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"sync"
	"time"

	"rtk_controller/internal/storage"
)

// ConnectionHistoryTracker tracks and analyzes client connection history
type ConnectionHistoryTracker struct {
	// Storage
	storage         *storage.TopologyStorage
	identityStorage *storage.IdentityStorage

	// Connection tracking
	connections map[string]*ClientConnectionHistory
	sessions    map[string]*ConnectionSession
	mu          sync.RWMutex

	// Configuration
	config ConnectionHistoryConfig

	// Background processing
	running bool
	cancel  context.CancelFunc

	// Statistics
	stats ConnectionHistoryStats
}

// ClientConnectionHistory tracks complete connection history for a client
type ClientConnectionHistory struct {
	MacAddress    string
	FriendlyName  string
	FirstSeen     time.Time
	LastSeen      time.Time
	TotalSessions int64
	TotalDuration time.Duration
	Sessions      []ConnectionSession
	Patterns      ConnectionPattern
	Preferences   ClientPreferences
	Reliability   ReliabilityMetrics
}

// ConnectionSession represents a single connection session
type ConnectionSession struct {
	ID               string
	MacAddress       string
	DeviceID         string // AP or router device ID
	SSID             string
	Interface        string
	StartTime        time.Time
	EndTime          time.Time
	Duration         time.Duration
	DisconnectReason string
	Quality          SessionQuality
	Activities       []SessionActivity
	Location         string
	Context          SessionContext
}

// SessionQuality holds quality metrics for a connection session
type SessionQuality struct {
	AverageRSSI     int
	MinRSSI         int
	MaxRSSI         int
	SignalStability float64
	ThroughputMbps  float64
	LatencyMs       float64
	PacketLoss      float64
	ErrorRate       float64
	QualityScore    float64 // 0-1 scale
}

// SessionActivity represents activity during a connection session
type SessionActivity struct {
	Timestamp        time.Time
	ActivityType     ActivityType
	Duration         time.Duration
	BytesTransferred int64
	Description      string
	Metadata         map[string]interface{}
}

// SessionContext provides context about the session environment
type SessionContext struct {
	TimeOfDay        string
	DayOfWeek        string
	NetworkLoad      float64
	UserCount        int
	WeatherCondition string
	LocationContext  string
}

// ConnectionPattern identifies patterns in connection behavior
type ConnectionPattern struct {
	PreferredAPs        []string
	PreferredSSIDs      []string
	ConnectionTimes     map[string]float64 // hour -> frequency
	SessionDurations    []time.Duration
	RoamingFrequency    float64
	LocationPreferences map[string]float64
	DeviceUsagePattern  UsagePattern
	Predictability      float64
}

// ClientPreferences holds client connection preferences
type ClientPreferences struct {
	PreferredBand        string // 2.4G, 5G, 6G
	PreferredChannel     int
	PreferredSecurity    string
	MinSignalStrength    int
	MaxRoamingDelay      time.Duration
	BandwidthRequirement int
	LatencyRequirement   float64
}

// ReliabilityMetrics tracks connection reliability
type ReliabilityMetrics struct {
	ConnectionSuccess   float64 // successful connections / total attempts
	SessionStability    float64 // stable sessions / total sessions
	AverageSessionTime  time.Duration
	DisconnectFrequency float64 // disconnects per hour
	ReconnectTime       time.Duration
	QualityConsistency  float64
}

// UsagePattern identifies device usage patterns
type UsagePattern struct {
	Type             PatternType
	DailyUsageHours  []int
	WeeklyPattern    map[string]float64 // day -> usage intensity
	SeasonalPattern  map[string]float64 // month -> usage intensity
	ActivityTypes    []ActivityType
	BandwidthProfile BandwidthProfile
}

// BandwidthProfile describes bandwidth usage characteristics
type BandwidthProfile struct {
	AverageUsage     float64 // Mbps
	PeakUsage        float64 // Mbps
	BurstFrequency   float64 // bursts per hour
	ApplicationTypes []string
	TrafficPattern   string // steady, bursty, intermittent
}

// Enums for connection tracking
type ActivityType string

const (
	ActivityBrowsing   ActivityType = "browsing"
	ActivityStreaming  ActivityType = "streaming"
	ActivityGaming     ActivityType = "gaming"
	ActivityDownload   ActivityType = "download"
	ActivityUpload     ActivityType = "upload"
	ActivityVoIP       ActivityType = "voip"
	ActivityVideoCall  ActivityType = "video_call"
	ActivityIoT        ActivityType = "iot"
	ActivityBackground ActivityType = "background"
	ActivityUnknown    ActivityType = "unknown"
)

// ConnectionHistoryConfig holds tracker configuration
type ConnectionHistoryConfig struct {
	// Data retention
	SessionRetention        time.Duration
	ActivityRetention       time.Duration
	PatternRetention        time.Duration
	MaxSessionsPerClient    int
	MaxActivitiesPerSession int

	// Analysis settings
	EnablePatternAnalysis     bool
	EnablePreferenceAnalysis  bool
	EnableReliabilityTracking bool
	PatternUpdateInterval     time.Duration
	AnalysisWindow            time.Duration

	// Session detection
	SessionTimeoutThreshold time.Duration
	MinSessionDuration      time.Duration
	MaxSessionGap           time.Duration

	// Quality thresholds
	MinSignalQuality  int
	MinThroughputMbps float64
	MaxLatencyMs      float64
	MaxPacketLoss     float64

	// Performance settings
	BatchSize          int
	ProcessingInterval time.Duration
	CleanupInterval    time.Duration
}

// ConnectionHistoryStats holds tracker statistics
type ConnectionHistoryStats struct {
	TotalClients       int64
	ActiveSessions     int64
	CompletedSessions  int64
	TotalSessionTime   time.Duration
	AverageSessionTime time.Duration
	PatternsIdentified int64
	ReliabilityIssues  int64
	LastUpdate         time.Time
	ProcessingErrors   int64
}

// NewConnectionHistoryTracker creates a new connection history tracker
func NewConnectionHistoryTracker(
	storage *storage.TopologyStorage,
	identityStorage *storage.IdentityStorage,
	config ConnectionHistoryConfig,
) *ConnectionHistoryTracker {
	return &ConnectionHistoryTracker{
		storage:         storage,
		identityStorage: identityStorage,
		connections:     make(map[string]*ClientConnectionHistory),
		sessions:        make(map[string]*ConnectionSession),
		config:          config,
		stats:           ConnectionHistoryStats{},
	}
}

// Start begins connection history tracking
func (cht *ConnectionHistoryTracker) Start() error {
	cht.mu.Lock()
	defer cht.mu.Unlock()

	if cht.running {
		return fmt.Errorf("connection history tracker is already running")
	}

	ctx, cancel := context.WithCancel(context.Background())
	cht.cancel = cancel
	cht.running = true

	log.Printf("Starting connection history tracker")

	// Start background processing
	go cht.sessionTrackingLoop(ctx)
	go cht.patternAnalysisLoop(ctx)
	go cht.reliabilityAnalysisLoop(ctx)
	go cht.cleanupLoop(ctx)

	// Load existing data
	if err := cht.loadExistingData(); err != nil {
		log.Printf("Failed to load existing connection history: %v", err)
	}

	return nil
}

// Stop stops connection history tracking
func (cht *ConnectionHistoryTracker) Stop() error {
	cht.mu.Lock()
	defer cht.mu.Unlock()

	if !cht.running {
		return fmt.Errorf("connection history tracker is not running")
	}

	cht.cancel()
	cht.running = false

	log.Printf("Connection history tracker stopped")
	return nil
}

// TrackConnection tracks a new connection event
func (cht *ConnectionHistoryTracker) TrackConnection(
	macAddress string,
	deviceID string,
	ssid string,
	interfaceName string,
	timestamp time.Time,
) error {

	cht.mu.Lock()
	defer cht.mu.Unlock()

	// Get or create client history
	history, exists := cht.connections[macAddress]
	if !exists {
		history = &ClientConnectionHistory{
			MacAddress: macAddress,
			FirstSeen:  timestamp,
			Sessions:   []ConnectionSession{},
		}
		cht.connections[macAddress] = history
		cht.stats.TotalClients++
	}

	// Get friendly name
	if identity, err := cht.identityStorage.GetDeviceIdentity(macAddress); err == nil {
		history.FriendlyName = identity.FriendlyName
	}

	// Update last seen
	history.LastSeen = timestamp

	// Check for existing active session
	sessionKey := fmt.Sprintf("%s-%s", macAddress, deviceID)
	existingSession, hasActiveSession := cht.sessions[sessionKey]

	if hasActiveSession {
		// Update existing session
		existingSession.EndTime = timestamp
		existingSession.Duration = timestamp.Sub(existingSession.StartTime)
	} else {
		// Create new session
		session := &ConnectionSession{
			ID:         fmt.Sprintf("session_%d_%s", timestamp.UnixMilli(), macAddress),
			MacAddress: macAddress,
			DeviceID:   deviceID,
			SSID:       ssid,
			Interface:  interfaceName,
			StartTime:  timestamp,
			Activities: []SessionActivity{},
			Context:    cht.buildSessionContext(timestamp),
		}

		cht.sessions[sessionKey] = session
		cht.stats.ActiveSessions++

		log.Printf("New connection session started: %s -> %s (%s)",
			macAddress, deviceID, ssid)
	}

	return nil
}

// TrackDisconnection tracks a disconnection event
func (cht *ConnectionHistoryTracker) TrackDisconnection(
	macAddress string,
	deviceID string,
	reason string,
	timestamp time.Time,
) error {

	cht.mu.Lock()
	defer cht.mu.Unlock()

	sessionKey := fmt.Sprintf("%s-%s", macAddress, deviceID)
	session, exists := cht.sessions[sessionKey]

	if !exists {
		return fmt.Errorf("no active session found for %s -> %s", macAddress, deviceID)
	}

	// End the session
	session.EndTime = timestamp
	session.Duration = timestamp.Sub(session.StartTime)
	session.DisconnectReason = reason

	// Calculate session quality
	cht.calculateSessionQuality(session)

	// Add to client history
	if history, exists := cht.connections[macAddress]; exists {
		history.Sessions = append(history.Sessions, *session)
		history.TotalSessions++
		history.TotalDuration += session.Duration

		// Limit session history
		if len(history.Sessions) > cht.config.MaxSessionsPerClient {
			history.Sessions = history.Sessions[len(history.Sessions)-cht.config.MaxSessionsPerClient:]
		}

		// Update patterns and preferences
		if cht.config.EnablePatternAnalysis {
			cht.updateConnectionPattern(history)
		}

		if cht.config.EnablePreferenceAnalysis {
			cht.updateClientPreferences(history)
		}

		if cht.config.EnableReliabilityTracking {
			cht.updateReliabilityMetrics(history)
		}
	}

	// Remove from active sessions
	delete(cht.sessions, sessionKey)
	cht.stats.ActiveSessions--
	cht.stats.CompletedSessions++
	cht.stats.TotalSessionTime += session.Duration

	log.Printf("Connection session ended: %s -> %s, duration: %v, reason: %s",
		macAddress, deviceID, session.Duration, reason)

	return nil
}

// TrackActivity tracks activity during a connection session
func (cht *ConnectionHistoryTracker) TrackActivity(
	macAddress string,
	deviceID string,
	activityType ActivityType,
	duration time.Duration,
	bytesTransferred int64,
	timestamp time.Time,
) error {

	cht.mu.Lock()
	defer cht.mu.Unlock()

	sessionKey := fmt.Sprintf("%s-%s", macAddress, deviceID)
	session, exists := cht.sessions[sessionKey]

	if !exists {
		return fmt.Errorf("no active session found for activity tracking")
	}

	activity := SessionActivity{
		Timestamp:        timestamp,
		ActivityType:     activityType,
		Duration:         duration,
		BytesTransferred: bytesTransferred,
		Description:      string(activityType),
		Metadata:         make(map[string]interface{}),
	}

	session.Activities = append(session.Activities, activity)

	// Limit activity history
	if len(session.Activities) > cht.config.MaxActivitiesPerSession {
		session.Activities = session.Activities[len(session.Activities)-cht.config.MaxActivitiesPerSession:]
	}

	return nil
}

// GetClientHistory returns connection history for a specific client
func (cht *ConnectionHistoryTracker) GetClientHistory(macAddress string) (*ClientConnectionHistory, bool) {
	cht.mu.RLock()
	defer cht.mu.RUnlock()

	history, exists := cht.connections[macAddress]
	if !exists {
		return nil, false
	}

	// Return copy
	historyCopy := *history
	historyCopy.Sessions = make([]ConnectionSession, len(history.Sessions))
	copy(historyCopy.Sessions, history.Sessions)

	return &historyCopy, true
}

// GetActiveSessions returns all currently active sessions
func (cht *ConnectionHistoryTracker) GetActiveSessions() map[string]*ConnectionSession {
	cht.mu.RLock()
	defer cht.mu.RUnlock()

	sessions := make(map[string]*ConnectionSession)
	for key, session := range cht.sessions {
		sessionCopy := *session
		sessions[key] = &sessionCopy
	}

	return sessions
}

// GetSessionHistory returns recent sessions with optional filtering
func (cht *ConnectionHistoryTracker) GetSessionHistory(
	since time.Time,
	macAddress string,
	deviceID string,
) []ConnectionSession {

	cht.mu.RLock()
	defer cht.mu.RUnlock()

	var sessions []ConnectionSession

	for _, history := range cht.connections {
		if macAddress != "" && history.MacAddress != macAddress {
			continue
		}

		for _, session := range history.Sessions {
			if session.StartTime.Before(since) {
				continue
			}

			if deviceID != "" && session.DeviceID != deviceID {
				continue
			}

			sessions = append(sessions, session)
		}
	}

	// Sort by start time (newest first)
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].StartTime.After(sessions[j].StartTime)
	})

	return sessions
}

// GetReliabilityReport generates a reliability report for a client or AP
func (cht *ConnectionHistoryTracker) GetReliabilityReport(
	macAddress string,
	deviceID string,
	period time.Duration,
) (*ReliabilityReport, error) {

	since := time.Now().Add(-period)
	sessions := cht.GetSessionHistory(since, macAddress, deviceID)

	if len(sessions) == 0 {
		return nil, fmt.Errorf("no sessions found for the specified period")
	}

	report := &ReliabilityReport{
		Period:        period,
		TotalSessions: len(sessions),
		MacAddress:    macAddress,
		DeviceID:      deviceID,
		GeneratedAt:   time.Now(),
	}

	// Calculate metrics
	var totalDuration time.Duration
	var qualitySum float64
	disconnectReasons := make(map[string]int)

	for _, session := range sessions {
		totalDuration += session.Duration
		qualitySum += session.Quality.QualityScore
		disconnectReasons[session.DisconnectReason]++
	}

	report.AverageSessionDuration = totalDuration / time.Duration(len(sessions))
	report.AverageQuality = qualitySum / float64(len(sessions))
	report.DisconnectReasons = disconnectReasons

	// Calculate success rate (sessions > minimum duration)
	successfulSessions := 0
	for _, session := range sessions {
		if session.Duration > cht.config.MinSessionDuration {
			successfulSessions++
		}
	}
	report.SuccessRate = float64(successfulSessions) / float64(len(sessions))

	return report, nil
}

// ReliabilityReport holds reliability analysis results
type ReliabilityReport struct {
	Period                 time.Duration
	TotalSessions          int
	MacAddress             string
	DeviceID               string
	AverageSessionDuration time.Duration
	AverageQuality         float64
	SuccessRate            float64
	DisconnectReasons      map[string]int
	GeneratedAt            time.Time
}

// GetStats returns tracker statistics
func (cht *ConnectionHistoryTracker) GetStats() ConnectionHistoryStats {
	cht.mu.RLock()
	defer cht.mu.RUnlock()

	stats := cht.stats

	// Calculate average session time
	if stats.CompletedSessions > 0 {
		stats.AverageSessionTime = stats.TotalSessionTime / time.Duration(stats.CompletedSessions)
	}

	return stats
}

// Private methods

func (cht *ConnectionHistoryTracker) sessionTrackingLoop(ctx context.Context) {
	ticker := time.NewTicker(cht.config.ProcessingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			cht.processSessionTimeouts()
		}
	}
}

func (cht *ConnectionHistoryTracker) patternAnalysisLoop(ctx context.Context) {
	if !cht.config.EnablePatternAnalysis {
		return
	}

	ticker := time.NewTicker(cht.config.PatternUpdateInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			cht.analyzeConnectionPatterns()
		}
	}
}

func (cht *ConnectionHistoryTracker) reliabilityAnalysisLoop(ctx context.Context) {
	if !cht.config.EnableReliabilityTracking {
		return
	}

	ticker := time.NewTicker(cht.config.PatternUpdateInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			cht.analyzeReliability()
		}
	}
}

func (cht *ConnectionHistoryTracker) cleanupLoop(ctx context.Context) {
	ticker := time.NewTicker(cht.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			cht.cleanupOldData()
		}
	}
}

func (cht *ConnectionHistoryTracker) processSessionTimeouts() {
	// Check for sessions that have timed out
	now := time.Now()

	cht.mu.Lock()
	defer cht.mu.Unlock()

	for _, session := range cht.sessions {
		timeSinceUpdate := now.Sub(session.EndTime)
		if timeSinceUpdate > cht.config.SessionTimeoutThreshold {
			// Session has timed out
			cht.TrackDisconnection(session.MacAddress, session.DeviceID, "timeout", now)
		}
	}
}

func (cht *ConnectionHistoryTracker) calculateSessionQuality(session *ConnectionSession) {
	// Calculate quality metrics for the session

	if len(session.Activities) == 0 {
		session.Quality.QualityScore = 0.5 // Default score
		return
	}

	// Calculate throughput from activities
	var totalBytes int64
	for _, activity := range session.Activities {
		totalBytes += activity.BytesTransferred
	}

	if session.Duration > 0 {
		throughputMbps := float64(totalBytes*8) / (1024 * 1024) / session.Duration.Seconds()
		session.Quality.ThroughputMbps = throughputMbps
	}

	// Calculate quality score
	score := 0.5 // Base score

	// Bonus for good throughput
	if session.Quality.ThroughputMbps > cht.config.MinThroughputMbps {
		score += 0.3
	}

	// Bonus for long session (indicates stability)
	if session.Duration > time.Hour {
		score += 0.2
	}

	session.Quality.QualityScore = score
}

func (cht *ConnectionHistoryTracker) updateConnectionPattern(history *ClientConnectionHistory) {
	// Update connection patterns based on session history

	if len(history.Sessions) < 3 {
		return // Need more data
	}

	// Analyze preferred APs and SSIDs
	apCounts := make(map[string]int)
	ssidCounts := make(map[string]int)

	for _, session := range history.Sessions {
		apCounts[session.DeviceID]++
		ssidCounts[session.SSID]++
	}

	// Get top preferences
	history.Patterns.PreferredAPs = cht.getTopKeys(apCounts, 3)
	history.Patterns.PreferredSSIDs = cht.getTopKeys(ssidCounts, 3)

	// Analyze time patterns
	timeCounts := make(map[string]int)
	for _, session := range history.Sessions {
		hour := fmt.Sprintf("%02d", session.StartTime.Hour())
		timeCounts[hour]++
	}

	history.Patterns.ConnectionTimes = make(map[string]float64)
	total := len(history.Sessions)
	for hour, count := range timeCounts {
		history.Patterns.ConnectionTimes[hour] = float64(count) / float64(total)
	}

	cht.stats.PatternsIdentified++
}

func (cht *ConnectionHistoryTracker) updateClientPreferences(history *ClientConnectionHistory) {
	// Update client preferences based on behavior patterns

	if len(history.Sessions) < 5 {
		return // Need more data
	}

	// Analyze session quality preferences
	var totalQuality float64
	var qualityCount int

	for _, session := range history.Sessions {
		if session.Quality.QualityScore > 0 {
			totalQuality += session.Quality.QualityScore
			qualityCount++
		}
	}

	if qualityCount > 0 {
		avgQuality := totalQuality / float64(qualityCount)
		// Set preferences based on quality expectations
		if avgQuality > 0.8 {
			history.Preferences.MinSignalStrength = -50
			history.Preferences.BandwidthRequirement = 50
		} else {
			history.Preferences.MinSignalStrength = -70
			history.Preferences.BandwidthRequirement = 10
		}
	}
}

func (cht *ConnectionHistoryTracker) updateReliabilityMetrics(history *ClientConnectionHistory) {
	// Update reliability metrics based on session history

	if len(history.Sessions) == 0 {
		return
	}

	// Calculate connection success rate
	successfulSessions := 0
	var totalDuration time.Duration

	for _, session := range history.Sessions {
		if session.Duration > cht.config.MinSessionDuration {
			successfulSessions++
		}
		totalDuration += session.Duration
	}

	history.Reliability.ConnectionSuccess = float64(successfulSessions) / float64(len(history.Sessions))
	history.Reliability.AverageSessionTime = totalDuration / time.Duration(len(history.Sessions))

	// Calculate session stability
	stableSessions := 0
	for _, session := range history.Sessions {
		if session.Quality.QualityScore > 0.6 {
			stableSessions++
		}
	}

	history.Reliability.SessionStability = float64(stableSessions) / float64(len(history.Sessions))

	// Check for reliability issues
	if history.Reliability.ConnectionSuccess < 0.8 || history.Reliability.SessionStability < 0.7 {
		cht.stats.ReliabilityIssues++
	}
}

func (cht *ConnectionHistoryTracker) analyzeConnectionPatterns() {
	// Analyze patterns across all clients

	for _, history := range cht.connections {
		cht.updateConnectionPattern(history)
	}
}

func (cht *ConnectionHistoryTracker) analyzeReliability() {
	// Analyze reliability across all clients

	for _, history := range cht.connections {
		cht.updateReliabilityMetrics(history)
	}
}

func (cht *ConnectionHistoryTracker) cleanupOldData() {
	// Clean up old session and activity data

	now := time.Now()
	sessionCutoff := now.Add(-cht.config.SessionRetention)
	activityCutoff := now.Add(-cht.config.ActivityRetention)

	cht.mu.Lock()
	defer cht.mu.Unlock()

	for _, history := range cht.connections {
		// Clean up old sessions
		var filteredSessions []ConnectionSession
		for _, session := range history.Sessions {
			if session.StartTime.After(sessionCutoff) {
				// Clean up old activities within the session
				var filteredActivities []SessionActivity
				for _, activity := range session.Activities {
					if activity.Timestamp.After(activityCutoff) {
						filteredActivities = append(filteredActivities, activity)
					}
				}
				session.Activities = filteredActivities
				filteredSessions = append(filteredSessions, session)
			}
		}
		history.Sessions = filteredSessions
	}

	log.Printf("Cleaned up old connection history data")
}

func (cht *ConnectionHistoryTracker) buildSessionContext(timestamp time.Time) SessionContext {
	// Build context information for the session

	return SessionContext{
		TimeOfDay:       fmt.Sprintf("%02d:00", timestamp.Hour()),
		DayOfWeek:       timestamp.Weekday().String(),
		NetworkLoad:     0.5, // TODO: Get actual network load
		UserCount:       len(cht.connections),
		LocationContext: "home", // TODO: Determine location
	}
}

func (cht *ConnectionHistoryTracker) getTopKeys(counts map[string]int, limit int) []string {
	// Get top keys by count

	type keyCount struct {
		key   string
		count int
	}

	var pairs []keyCount
	for key, count := range counts {
		pairs = append(pairs, keyCount{key, count})
	}

	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].count > pairs[j].count
	})

	var result []string
	for i, pair := range pairs {
		if i >= limit {
			break
		}
		result = append(result, pair.key)
	}

	return result
}

func (cht *ConnectionHistoryTracker) loadExistingData() error {
	// Load existing connection history from storage

	log.Printf("Loading existing connection history from storage")

	// TODO: Implement loading from storage
	// For now, start with empty state

	return nil
}

// SaveToStorage saves connection history to persistent storage
func (cht *ConnectionHistoryTracker) SaveToStorage() error {
	cht.mu.RLock()
	defer cht.mu.RUnlock()

	// Save connection histories
	for macAddress, history := range cht.connections {
		_, err := json.Marshal(history)
		if err != nil {
			return fmt.Errorf("failed to marshal connection history for %s: %w", macAddress, err)
		}

		_ = fmt.Sprintf("connection_history:%s", macAddress)
		// TODO: Implement storage.Set method
		// if err := cht.storage.Set(key, string(data)); err != nil {
		//	return fmt.Errorf("failed to save connection history for %s: %w", macAddress, err)
		// }
	}

	log.Printf("Saved connection history for %d clients", len(cht.connections))
	return nil
}
