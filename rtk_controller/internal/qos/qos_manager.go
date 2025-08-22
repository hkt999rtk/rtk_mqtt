package qos

import (
	"fmt"
	"sync"
	"time"

	"rtk_controller/pkg/types"
)

// QoSManager manages QoS policies and recommendations
type QoSManager struct {
	mu              sync.RWMutex
	config          *QoSConfig
	policies        map[string]*types.BandwidthRule
	trafficRules    map[string]*types.TrafficRule
	queues          map[string]*types.QueueInfo
	recommendations map[string]*types.QoSRecommendation
	trafficAnalyzer *TrafficAnalyzer
	policyEngine    *PolicyEngine
}

// QoSConfig contains QoS manager configuration
type QoSConfig struct {
	Enabled              bool          `json:"enabled"`
	AutoRecommendations  bool          `json:"auto_recommendations"`
	MaxBandwidthMbps     float64       `json:"max_bandwidth_mbps"`
	DefaultPriority      int           `json:"default_priority"`
	RecommendationWindow time.Duration `json:"recommendation_window"`
}

// NewQoSManager creates a new QoS manager
func NewQoSManager(config *QoSConfig) *QoSManager {
	if config == nil {
		config = &QoSConfig{
			Enabled:              true,
			AutoRecommendations:  true,
			MaxBandwidthMbps:     1000,
			DefaultPriority:      5,
			RecommendationWindow: 24 * time.Hour,
		}
	}

	analyzerConfig := &Config{
		BandwidthCapacity: config.MaxBandwidthMbps,
		EnableAutoDetect:  config.AutoRecommendations,
		TopTalkersCount:   10,
	}

	return &QoSManager{
		config:          config,
		policies:        make(map[string]*types.BandwidthRule),
		trafficRules:    make(map[string]*types.TrafficRule),
		queues:          make(map[string]*types.QueueInfo),
		recommendations: make(map[string]*types.QoSRecommendation),
		trafficAnalyzer: NewTrafficAnalyzer(analyzerConfig),
		policyEngine:    NewPolicyEngine(config.MaxBandwidthMbps),
	}
}

// GetQoSInfo returns current QoS information
func (qm *QoSManager) GetQoSInfo() *types.QoSInfo {
	qm.mu.RLock()
	defer qm.mu.RUnlock()

	// Convert maps to slices
	bandwidthCaps := []types.BandwidthRule{}
	for _, rule := range qm.policies {
		bandwidthCaps = append(bandwidthCaps, *rule)
	}

	trafficShaping := []types.TrafficRule{}
	for _, rule := range qm.trafficRules {
		trafficShaping = append(trafficShaping, *rule)
	}

	priorityQueues := []types.QueueInfo{}
	for _, queue := range qm.queues {
		priorityQueues = append(priorityQueues, *queue)
	}

	return &types.QoSInfo{
		Enabled:           qm.config.Enabled,
		BandwidthCaps:     bandwidthCaps,
		TrafficShaping:    trafficShaping,
		PriorityQueues:    priorityQueues,
		ActiveConnections: qm.getActiveConnections(),
		TrafficStats:      qm.trafficAnalyzer.GetCurrentStats(),
	}
}

// UpdateTraffic updates traffic statistics for a device
func (qm *QoSManager) UpdateTraffic(deviceID, deviceMAC string, upload, download float64, connections int) {
	qm.trafficAnalyzer.UpdateTraffic(deviceID, deviceMAC, upload, download, connections)

	// Generate recommendations if enabled
	if qm.config.AutoRecommendations {
		qm.generateRecommendations(deviceID)
	}
}

// AddBandwidthRule adds a bandwidth limitation rule
func (qm *QoSManager) AddBandwidthRule(rule *types.BandwidthRule) error {
	qm.mu.Lock()
	defer qm.mu.Unlock()

	if rule.RuleID == "" {
		rule.RuleID = fmt.Sprintf("bw-%d", time.Now().Unix())
	}

	// Validate rule
	if err := qm.validateBandwidthRule(rule); err != nil {
		return err
	}

	qm.policies[rule.RuleID] = rule
	return nil
}

// AddTrafficRule adds a traffic shaping rule
func (qm *QoSManager) AddTrafficRule(rule *types.TrafficRule) error {
	qm.mu.Lock()
	defer qm.mu.Unlock()

	if rule.RuleID == "" {
		rule.RuleID = fmt.Sprintf("tr-%d", time.Now().Unix())
	}

	// Validate rule
	if err := qm.validateTrafficRule(rule); err != nil {
		return err
	}

	qm.trafficRules[rule.RuleID] = rule
	return nil
}

// AddQueue adds a priority queue
func (qm *QoSManager) AddQueue(queue *types.QueueInfo) error {
	qm.mu.Lock()
	defer qm.mu.Unlock()

	if queue.QueueID == "" {
		queue.QueueID = fmt.Sprintf("q-%d", time.Now().Unix())
	}

	// Validate queue
	if err := qm.validateQueue(queue); err != nil {
		return err
	}

	qm.queues[queue.QueueID] = queue
	return nil
}

// GetRecommendations returns current QoS recommendations
func (qm *QoSManager) GetRecommendations() []*types.QoSRecommendation {
	qm.mu.RLock()
	defer qm.mu.RUnlock()

	recommendations := []*types.QoSRecommendation{}
	cutoff := time.Now().Add(-qm.config.RecommendationWindow).Unix()

	for _, rec := range qm.recommendations {
		if rec.CreatedAt > cutoff {
			recommendations = append(recommendations, rec)
		}
	}

	return recommendations
}

// AnalyzeAndRecommend performs comprehensive analysis and generates recommendations
func (qm *QoSManager) AnalyzeAndRecommend() []*types.QoSRecommendation {
	recommendations := []*types.QoSRecommendation{}

	// Get current traffic statistics
	stats := qm.trafficAnalyzer.GetCurrentStats()

	// Analyze bandwidth usage
	if rec := qm.analyzeBandwidthUsage(stats); rec != nil {
		recommendations = append(recommendations, rec)
	}

	// Identify hotspots
	hotspots := qm.trafficAnalyzer.IdentifyHotspots()
	for _, deviceID := range hotspots {
		if rec := qm.recommendForHotspot(deviceID); rec != nil {
			recommendations = append(recommendations, rec)
		}
	}

	// Check for anomalies
	anomalies := qm.trafficAnalyzer.GetAnomalies()
	for _, anomaly := range anomalies {
		if rec := qm.recommendForAnomaly(anomaly); rec != nil {
			recommendations = append(recommendations, rec)
		}
	}

	// Store recommendations
	qm.mu.Lock()
	for _, rec := range recommendations {
		qm.recommendations[rec.ID] = rec
	}
	qm.mu.Unlock()

	return recommendations
}

// generateRecommendations generates recommendations for a specific device
func (qm *QoSManager) generateRecommendations(deviceID string) {
	// Analyze device traffic pattern
	pattern := qm.trafficAnalyzer.AnalyzeTrafficPattern(deviceID, 24*time.Hour)
	if pattern == nil {
		return
	}

	// Generate recommendation based on pattern
	rec := qm.policyEngine.GenerateRecommendation(deviceID, pattern)
	if rec != nil {
		qm.mu.Lock()
		qm.recommendations[rec.ID] = rec
		qm.mu.Unlock()
	}
}

// analyzeBandwidthUsage analyzes overall bandwidth usage
func (qm *QoSManager) analyzeBandwidthUsage(stats *types.TrafficStats) *types.QoSRecommendation {
	if stats.TotalBandwidthMbps == 0 {
		return nil
	}

	utilization := (stats.UsedBandwidthMbps / stats.TotalBandwidthMbps) * 100

	if utilization > 80 {
		return &types.QoSRecommendation{
			ID:          fmt.Sprintf("rec-bw-%d", time.Now().Unix()),
			Type:        "bandwidth_cap",
			Reason:      fmt.Sprintf("Network utilization at %.1f%%", utilization),
			Description: "Implement bandwidth caps for top consumers to prevent congestion",
			Priority:    1,
			Impact:      "high",
			CreatedAt:   time.Now().Unix(),
		}
	}

	return nil
}

// recommendForHotspot generates recommendation for a traffic hotspot
func (qm *QoSManager) recommendForHotspot(deviceID string) *types.QoSRecommendation {
	return &types.QoSRecommendation{
		ID:          fmt.Sprintf("rec-hs-%s-%d", deviceID, time.Now().Unix()),
		Type:        "traffic_shaping",
		Reason:      fmt.Sprintf("Device %s identified as traffic hotspot", deviceID),
		Description: "Apply traffic shaping to limit impact on other devices",
		Priority:    2,
		Impact:      "medium",
		Devices:     []string{deviceID},
		Rule: &types.BandwidthRule{
			Target:        deviceID,
			UploadLimit:   100,
			DownloadLimit: 200,
			Priority:      3,
			Enabled:       false,
		},
		CreatedAt: time.Now().Unix(),
	}
}

// recommendForAnomaly generates recommendation for a traffic anomaly
func (qm *QoSManager) recommendForAnomaly(anomaly *types.TrafficAnomaly) *types.QoSRecommendation {
	var recType, description string

	switch anomaly.Type {
	case "spike":
		recType = "priority_queue"
		description = "Create low-priority queue for burst traffic"
	case "sustained_high":
		recType = "bandwidth_cap"
		description = "Implement bandwidth cap to ensure fair usage"
	case "unusual_protocol":
		recType = "traffic_shaping"
		description = "Monitor and potentially shape unusual traffic patterns"
	default:
		return nil
	}

	return &types.QoSRecommendation{
		ID:          fmt.Sprintf("rec-an-%s-%d", anomaly.DeviceID, time.Now().Unix()),
		Type:        recType,
		Reason:      anomaly.Description,
		Description: description,
		Priority:    3,
		Impact:      anomaly.Severity,
		Devices:     []string{anomaly.DeviceID},
		CreatedAt:   time.Now().Unix(),
	}
}

// getActiveConnections returns active network connections
func (qm *QoSManager) getActiveConnections() []types.ActiveConnection {
	// This would integrate with netstat or conntrack in real implementation
	// For now, return empty slice
	return []types.ActiveConnection{}
}

// validateBandwidthRule validates a bandwidth rule
func (qm *QoSManager) validateBandwidthRule(rule *types.BandwidthRule) error {
	if rule.Target == "" {
		return fmt.Errorf("target cannot be empty")
	}
	if rule.UploadLimit < 0 || rule.DownloadLimit < 0 {
		return fmt.Errorf("bandwidth limits cannot be negative")
	}
	if rule.UploadLimit+rule.DownloadLimit > int(qm.config.MaxBandwidthMbps) {
		return fmt.Errorf("total bandwidth exceeds maximum capacity")
	}
	return nil
}

// validateTrafficRule validates a traffic rule
func (qm *QoSManager) validateTrafficRule(rule *types.TrafficRule) error {
	validActions := map[string]bool{
		"allow": true, "block": true, "throttle": true, "prioritize": true,
	}
	if !validActions[rule.Action] {
		return fmt.Errorf("invalid action: %s", rule.Action)
	}

	validProtocols := map[string]bool{
		"tcp": true, "udp": true, "icmp": true, "all": true,
	}
	if !validProtocols[rule.Protocol] {
		return fmt.Errorf("invalid protocol: %s", rule.Protocol)
	}

	return nil
}

// validateQueue validates a queue configuration
func (qm *QoSManager) validateQueue(queue *types.QueueInfo) error {
	if queue.BandwidthPct < 0 || queue.BandwidthPct > 100 {
		return fmt.Errorf("bandwidth percentage must be between 0 and 100")
	}
	if queue.Priority < 0 || queue.Priority > 10 {
		return fmt.Errorf("priority must be between 0 and 10")
	}
	return nil
}

// ApplyRecommendation applies a QoS recommendation
func (qm *QoSManager) ApplyRecommendation(recID string) error {
	qm.mu.Lock()
	defer qm.mu.Unlock()

	rec, exists := qm.recommendations[recID]
	if !exists {
		return fmt.Errorf("recommendation %s not found", recID)
	}

	switch rec.Type {
	case "bandwidth_cap":
		if rule, ok := rec.Rule.(*types.BandwidthRule); ok {
			rule.Enabled = true
			qm.policies[rule.RuleID] = rule
		}
	case "traffic_shaping":
		if rule, ok := rec.Rule.(*types.TrafficRule); ok {
			qm.trafficRules[rule.RuleID] = rule
		}
	case "priority_queue":
		if queue, ok := rec.Rule.(*types.QueueInfo); ok {
			qm.queues[queue.QueueID] = queue
		}
	}

	return nil
}
