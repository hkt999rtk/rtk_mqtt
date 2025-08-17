package qos

import (
	"math"
	"sort"
	"sync"
	"time"

	"rtk_controller/pkg/types"
)

// TrafficAnalyzer analyzes network traffic patterns
type TrafficAnalyzer struct {
	mu                sync.RWMutex
	config            *Config
	trafficHistory    map[string]*DeviceTrafficHistory
	anomalies         map[string]*types.TrafficAnomaly
	currentStats      *types.TrafficStats
	anomalyDetector   *AnomalyDetector
	hotspotIdentifier *HotspotIdentifier
}

// Config for traffic analyzer
type Config struct {
	SamplingInterval   time.Duration `json:"sampling_interval"`
	HistoryRetention   time.Duration `json:"history_retention"`
	AnomalyThreshold   float64       `json:"anomaly_threshold"`
	TopTalkersCount    int           `json:"top_talkers_count"`
	EnableAutoDetect   bool          `json:"enable_auto_detect"`
	BandwidthCapacity  float64       `json:"bandwidth_capacity_mbps"`
}

// DeviceTrafficHistory stores historical traffic data for a device
type DeviceTrafficHistory struct {
	DeviceID     string
	Samples      []TrafficSample
	LastUpdated  time.Time
	AverageUpload   float64
	AverageDownload float64
	PeakUpload      float64
	PeakDownload    float64
}

// TrafficSample represents a single traffic measurement
type TrafficSample struct {
	Timestamp    time.Time
	UploadMbps   float64
	DownloadMbps float64
	Connections  int
	Bytes        int64
}

// NewTrafficAnalyzer creates a new traffic analyzer
func NewTrafficAnalyzer(config *Config) *TrafficAnalyzer {
	if config == nil {
		config = &Config{
			SamplingInterval:  30 * time.Second,
			HistoryRetention:  24 * time.Hour,
			AnomalyThreshold:  0.8, // 80% of capacity
			TopTalkersCount:   10,
			EnableAutoDetect:  true,
			BandwidthCapacity: 1000, // 1Gbps default
		}
	}

	return &TrafficAnalyzer{
		config:            config,
		trafficHistory:    make(map[string]*DeviceTrafficHistory),
		anomalies:         make(map[string]*types.TrafficAnomaly),
		anomalyDetector:   NewAnomalyDetector(config.AnomalyThreshold),
		hotspotIdentifier: NewHotspotIdentifier(),
		currentStats: &types.TrafficStats{
			UpdatedAt: time.Now().Unix(),
		},
	}
}

// UpdateTraffic updates traffic statistics for a device
func (ta *TrafficAnalyzer) UpdateTraffic(deviceID, deviceMAC string, upload, download float64, connections int) {
	ta.mu.Lock()
	defer ta.mu.Unlock()

	// Update or create device history
	history, exists := ta.trafficHistory[deviceID]
	if !exists {
		history = &DeviceTrafficHistory{
			DeviceID: deviceID,
			Samples:  []TrafficSample{},
		}
		ta.trafficHistory[deviceID] = history
	}

	// Add new sample
	sample := TrafficSample{
		Timestamp:    time.Now(),
		UploadMbps:   upload,
		DownloadMbps: download,
		Connections:  connections,
		Bytes:        int64((upload + download) * 125000), // Convert Mbps to bytes (approximate)
	}
	history.Samples = append(history.Samples, sample)
	history.LastUpdated = time.Now()

	// Update statistics
	ta.updateStatistics(history, sample)

	// Clean old samples
	ta.cleanOldSamples(history)

	// Detect anomalies
	if ta.config.EnableAutoDetect {
		ta.detectAnomalies(deviceID, history, sample)
	}
}

// updateStatistics updates device traffic statistics
func (ta *TrafficAnalyzer) updateStatistics(history *DeviceTrafficHistory, sample TrafficSample) {
	// Update peaks
	if sample.UploadMbps > history.PeakUpload {
		history.PeakUpload = sample.UploadMbps
	}
	if sample.DownloadMbps > history.PeakDownload {
		history.PeakDownload = sample.DownloadMbps
	}

	// Calculate averages
	if len(history.Samples) > 0 {
		var totalUpload, totalDownload float64
		for _, s := range history.Samples {
			totalUpload += s.UploadMbps
			totalDownload += s.DownloadMbps
		}
		history.AverageUpload = totalUpload / float64(len(history.Samples))
		history.AverageDownload = totalDownload / float64(len(history.Samples))
	}
}

// cleanOldSamples removes samples older than retention period
func (ta *TrafficAnalyzer) cleanOldSamples(history *DeviceTrafficHistory) {
	cutoff := time.Now().Add(-ta.config.HistoryRetention)
	newSamples := []TrafficSample{}
	
	for _, sample := range history.Samples {
		if sample.Timestamp.After(cutoff) {
			newSamples = append(newSamples, sample)
		}
	}
	
	history.Samples = newSamples
}

// detectAnomalies detects traffic anomalies
func (ta *TrafficAnalyzer) detectAnomalies(deviceID string, history *DeviceTrafficHistory, sample TrafficSample) {
	anomalies := ta.anomalyDetector.Detect(deviceID, history, sample, ta.config.BandwidthCapacity)
	
	for _, anomaly := range anomalies {
		ta.anomalies[anomaly.ID] = anomaly
	}
}

// GetCurrentStats returns current traffic statistics
func (ta *TrafficAnalyzer) GetCurrentStats() *types.TrafficStats {
	ta.mu.RLock()
	defer ta.mu.RUnlock()

	stats := &types.TrafficStats{
		DeviceTraffic:        ta.getDeviceTraffic(),
		TopTalkers:           ta.getTopTalkers(),
		ProtocolDistribution: ta.getProtocolDistribution(),
		UpdatedAt:            time.Now().Unix(),
	}

	// Calculate total bandwidth usage
	for _, device := range stats.DeviceTraffic {
		stats.UsedBandwidthMbps += device.UploadMbps + device.DownloadMbps
	}
	stats.TotalBandwidthMbps = ta.config.BandwidthCapacity

	return stats
}

// getDeviceTraffic returns current device traffic information
func (ta *TrafficAnalyzer) getDeviceTraffic() []types.DeviceTrafficInfo {
	devices := []types.DeviceTrafficInfo{}

	for deviceID, history := range ta.trafficHistory {
		if len(history.Samples) == 0 {
			continue
		}

		// Get latest sample
		latest := history.Samples[len(history.Samples)-1]
		
		info := types.DeviceTrafficInfo{
			DeviceID:     deviceID,
			UploadMbps:   latest.UploadMbps,
			DownloadMbps: latest.DownloadMbps,
			TotalBytes:   latest.Bytes,
			ActiveConns:  latest.Connections,
			BandwidthPct: ((latest.UploadMbps + latest.DownloadMbps) / ta.config.BandwidthCapacity) * 100,
		}
		
		devices = append(devices, info)
	}

	return devices
}

// getTopTalkers identifies top bandwidth consumers
func (ta *TrafficAnalyzer) getTopTalkers() []types.TopTalkerInfo {
	talkers := []types.TopTalkerInfo{}

	// Collect all devices with their total bandwidth
	for deviceID, history := range ta.trafficHistory {
		if len(history.Samples) == 0 {
			continue
		}

		latest := history.Samples[len(history.Samples)-1]
		totalMbps := latest.UploadMbps + latest.DownloadMbps

		talker := types.TopTalkerInfo{
			DeviceID:    deviceID,
			TotalMbps:   totalMbps,
			TrafficType: "total",
		}
		
		talkers = append(talkers, talker)
	}

	// Sort by bandwidth usage
	sort.Slice(talkers, func(i, j int) bool {
		return talkers[i].TotalMbps > talkers[j].TotalMbps
	})

	// Assign ranks and limit to top N
	for i := range talkers {
		talkers[i].Rank = i + 1
		if i >= ta.config.TopTalkersCount {
			talkers = talkers[:ta.config.TopTalkersCount]
			break
		}
	}

	return talkers
}

// getProtocolDistribution returns traffic distribution by protocol
func (ta *TrafficAnalyzer) getProtocolDistribution() []types.ProtocolTrafficInfo {
	// This would require deep packet inspection or netflow data
	// For now, return simulated distribution
	return []types.ProtocolTrafficInfo{
		{Protocol: "HTTP/HTTPS", TotalMbps: ta.currentStats.UsedBandwidthMbps * 0.6, Percentage: 60},
		{Protocol: "Video Streaming", TotalMbps: ta.currentStats.UsedBandwidthMbps * 0.25, Percentage: 25},
		{Protocol: "Other TCP", TotalMbps: ta.currentStats.UsedBandwidthMbps * 0.1, Percentage: 10},
		{Protocol: "UDP", TotalMbps: ta.currentStats.UsedBandwidthMbps * 0.05, Percentage: 5},
	}
}

// GetAnomalies returns current traffic anomalies
func (ta *TrafficAnalyzer) GetAnomalies() []*types.TrafficAnomaly {
	ta.mu.RLock()
	defer ta.mu.RUnlock()

	anomalies := []*types.TrafficAnomaly{}
	for _, anomaly := range ta.anomalies {
		if !anomaly.Resolved {
			anomalies = append(anomalies, anomaly)
		}
	}

	return anomalies
}

// IdentifyHotspots identifies traffic hotspot devices
func (ta *TrafficAnalyzer) IdentifyHotspots() []string {
	ta.mu.RLock()
	defer ta.mu.RUnlock()

	return ta.hotspotIdentifier.Identify(ta.trafficHistory, ta.config.BandwidthCapacity)
}

// AnalyzeTrafficPattern analyzes traffic patterns over time
func (ta *TrafficAnalyzer) AnalyzeTrafficPattern(deviceID string, duration time.Duration) *TrafficPattern {
	ta.mu.RLock()
	defer ta.mu.RUnlock()

	history, exists := ta.trafficHistory[deviceID]
	if !exists {
		return nil
	}

	return analyzePattern(history, duration)
}

// TrafficPattern represents analyzed traffic patterns
type TrafficPattern struct {
	DeviceID        string
	PeakHours       []int     // Hours of day with peak usage
	AverageDaily    float64   // Average daily bandwidth
	Consistency     float64   // How consistent the pattern is (0-1)
	Type            string    // streaming, browsing, gaming, mixed
	Recommendation  string
}

// analyzePattern analyzes traffic patterns for a device
func analyzePattern(history *DeviceTrafficHistory, duration time.Duration) *TrafficPattern {
	pattern := &TrafficPattern{
		DeviceID: history.DeviceID,
	}

	if len(history.Samples) < 10 {
		pattern.Type = "insufficient_data"
		return pattern
	}

	// Analyze peak hours
	hourlyUsage := make(map[int]float64)
	cutoff := time.Now().Add(-duration)
	
	for _, sample := range history.Samples {
		if sample.Timestamp.After(cutoff) {
			hour := sample.Timestamp.Hour()
			hourlyUsage[hour] += sample.UploadMbps + sample.DownloadMbps
		}
	}

	// Find peak hours
	for hour, usage := range hourlyUsage {
		if usage > history.AverageUpload+history.AverageDownload {
			pattern.PeakHours = append(pattern.PeakHours, hour)
		}
	}

	// Determine traffic type based on patterns
	pattern.Type = determineTrafficType(history)
	
	// Generate recommendation
	pattern.Recommendation = generateRecommendation(pattern.Type, history)

	return pattern
}

// determineTrafficType determines the type of traffic pattern
func determineTrafficType(history *DeviceTrafficHistory) string {
	if len(history.Samples) == 0 {
		return "unknown"
	}

	// Calculate download/upload ratio
	ratio := history.AverageDownload / math.Max(history.AverageUpload, 0.1)

	// High download, low upload = streaming/browsing
	if ratio > 10 {
		if history.PeakDownload > 20 {
			return "streaming"
		}
		return "browsing"
	}

	// Balanced traffic = gaming or video calls
	if ratio > 0.5 && ratio < 2 {
		return "gaming"
	}

	// High upload = content creation/backup
	if ratio < 0.5 {
		return "upload_heavy"
	}

	return "mixed"
}

// generateRecommendation generates QoS recommendations based on traffic type
func generateRecommendation(trafficType string, history *DeviceTrafficHistory) string {
	switch trafficType {
	case "streaming":
		return "Prioritize download bandwidth, implement buffering optimization"
	case "gaming":
		return "Minimize latency, prioritize consistent bandwidth"
	case "upload_heavy":
		return "Allocate upload bandwidth quota, consider off-peak scheduling"
	case "browsing":
		return "Standard QoS policy sufficient"
	default:
		return "Monitor for pattern changes"
	}
}