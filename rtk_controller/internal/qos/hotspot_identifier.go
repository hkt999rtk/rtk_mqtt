package qos

import (
	"sort"
	"time"
)

// HotspotIdentifier identifies traffic hotspot devices
type HotspotIdentifier struct {
	hotspotThreshold float64 // Percentage of total bandwidth
	minDuration      time.Duration
	maxHotspots      int
}

// NewHotspotIdentifier creates a new hotspot identifier
func NewHotspotIdentifier() *HotspotIdentifier {
	return &HotspotIdentifier{
		hotspotThreshold: 0.2,  // 20% of total bandwidth
		minDuration:      5 * time.Minute,
		maxHotspots:      5,
	}
}

// Identify identifies traffic hotspot devices
func (hi *HotspotIdentifier) Identify(trafficHistory map[string]*DeviceTrafficHistory, capacity float64) []string {
	hotspots := []HotspotDevice{}
	now := time.Now()

	for deviceID, history := range trafficHistory {
		if len(history.Samples) == 0 {
			continue
		}

		// Calculate recent bandwidth usage
		recentUsage := hi.calculateRecentUsage(history, now)
		percentageUsage := (recentUsage / capacity) * 100

		// Check if device qualifies as hotspot
		if percentageUsage >= hi.hotspotThreshold*100 {
			duration := hi.calculateHighUsageDuration(history, capacity)
			
			if duration >= hi.minDuration {
				hotspots = append(hotspots, HotspotDevice{
					DeviceID:        deviceID,
					BandwidthMbps:   recentUsage,
					PercentageUsage: percentageUsage,
					Duration:        duration,
					Score:           hi.calculateHotspotScore(percentageUsage, duration),
				})
			}
		}
	}

	// Sort by score and return top N
	sort.Slice(hotspots, func(i, j int) bool {
		return hotspots[i].Score > hotspots[j].Score
	})

	deviceIDs := []string{}
	for i, hotspot := range hotspots {
		if i >= hi.maxHotspots {
			break
		}
		deviceIDs = append(deviceIDs, hotspot.DeviceID)
	}

	return deviceIDs
}

// HotspotDevice represents a traffic hotspot device
type HotspotDevice struct {
	DeviceID        string
	BandwidthMbps   float64
	PercentageUsage float64
	Duration        time.Duration
	Score           float64
}

// calculateRecentUsage calculates recent bandwidth usage
func (hi *HotspotIdentifier) calculateRecentUsage(history *DeviceTrafficHistory, now time.Time) float64 {
	cutoff := now.Add(-5 * time.Minute)
	var totalUsage float64
	count := 0

	for _, sample := range history.Samples {
		if sample.Timestamp.After(cutoff) {
			totalUsage += sample.UploadMbps + sample.DownloadMbps
			count++
		}
	}

	if count == 0 {
		return 0
	}

	return totalUsage / float64(count)
}

// calculateHighUsageDuration calculates how long device has been a hotspot
func (hi *HotspotIdentifier) calculateHighUsageDuration(history *DeviceTrafficHistory, capacity float64) time.Duration {
	if len(history.Samples) < 2 {
		return 0
	}

	threshold := capacity * hi.hotspotThreshold
	var startTime *time.Time
	var endTime time.Time

	// Find continuous high usage period
	for i := len(history.Samples) - 1; i >= 0; i-- {
		sample := history.Samples[i]
		usage := sample.UploadMbps + sample.DownloadMbps

		if usage >= threshold {
			if startTime == nil {
				endTime = sample.Timestamp
			}
			startTime = &sample.Timestamp
		} else if startTime != nil {
			// High usage period ended
			break
		}
	}

	if startTime == nil {
		return 0
	}

	return endTime.Sub(*startTime)
}

// calculateHotspotScore calculates a score for ranking hotspots
func (hi *HotspotIdentifier) calculateHotspotScore(percentageUsage float64, duration time.Duration) float64 {
	// Score based on both bandwidth usage and duration
	bandwidthScore := percentageUsage / 100.0
	durationScore := duration.Minutes() / 60.0 // Normalize to hours
	
	// Weight bandwidth more heavily than duration
	return bandwidthScore*0.7 + durationScore*0.3
}

// GetHotspotDetails returns detailed information about hotspots
func (hi *HotspotIdentifier) GetHotspotDetails(trafficHistory map[string]*DeviceTrafficHistory, capacity float64) []HotspotDetail {
	details := []HotspotDetail{}
	hotspotIDs := hi.Identify(trafficHistory, capacity)

	for _, deviceID := range hotspotIDs {
		history, exists := trafficHistory[deviceID]
		if !exists {
			continue
		}

		detail := HotspotDetail{
			DeviceID:        deviceID,
			CurrentUsage:    hi.calculateRecentUsage(history, time.Now()),
			AverageUsage:    history.AverageUpload + history.AverageDownload,
			PeakUsage:       history.PeakUpload + history.PeakDownload,
			Duration:        hi.calculateHighUsageDuration(history, capacity),
			TrafficType:     hi.identifyTrafficType(history),
			Impact:          hi.assessImpact(history, capacity),
			Recommendation:  hi.generateRecommendation(history, capacity),
		}

		details = append(details, detail)
	}

	return details
}

// HotspotDetail provides detailed information about a traffic hotspot
type HotspotDetail struct {
	DeviceID       string
	CurrentUsage   float64
	AverageUsage   float64
	PeakUsage      float64
	Duration       time.Duration
	TrafficType    string
	Impact         string
	Recommendation string
}

// identifyTrafficType identifies the type of traffic for a hotspot
func (hi *HotspotIdentifier) identifyTrafficType(history *DeviceTrafficHistory) string {
	if len(history.Samples) == 0 {
		return "unknown"
	}

	// Analyze upload/download ratio
	ratio := history.AverageDownload / (history.AverageUpload + 0.1)

	if ratio > 10 {
		return "download_heavy"
	} else if ratio < 0.5 {
		return "upload_heavy"
	} else if history.AverageUpload+history.AverageDownload > 50 {
		return "high_volume"
	}

	return "balanced"
}

// assessImpact assesses the impact of a hotspot on the network
func (hi *HotspotIdentifier) assessImpact(history *DeviceTrafficHistory, capacity float64) string {
	usage := history.AverageUpload + history.AverageDownload
	percentage := (usage / capacity) * 100

	if percentage > 50 {
		return "critical"
	} else if percentage > 30 {
		return "high"
	} else if percentage > 20 {
		return "medium"
	}

	return "low"
}

// generateRecommendation generates recommendations for handling hotspots
func (hi *HotspotIdentifier) generateRecommendation(history *DeviceTrafficHistory, capacity float64) string {
	usage := history.AverageUpload + history.AverageDownload
	percentage := (usage / capacity) * 100
	trafficType := hi.identifyTrafficType(history)

	switch {
	case percentage > 50:
		return "Implement immediate bandwidth cap or traffic shaping"
	case percentage > 30 && trafficType == "download_heavy":
		return "Consider download speed limiting during peak hours"
	case percentage > 30 && trafficType == "upload_heavy":
		return "Schedule large uploads during off-peak hours"
	case percentage > 20:
		return "Monitor closely and consider QoS prioritization"
	default:
		return "Continue monitoring"
	}
}