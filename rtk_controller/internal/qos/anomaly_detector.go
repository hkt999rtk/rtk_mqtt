package qos

import (
	"fmt"
	"math"
	"time"

	"rtk_controller/pkg/types"
)

// AnomalyDetector detects traffic anomalies
type AnomalyDetector struct {
	threshold        float64
	baselineWindow   time.Duration
	spikeThreshold   float64
	sustainedWindow  time.Duration
}

// NewAnomalyDetector creates a new anomaly detector
func NewAnomalyDetector(threshold float64) *AnomalyDetector {
	return &AnomalyDetector{
		threshold:        threshold,
		baselineWindow:   1 * time.Hour,
		spikeThreshold:   2.0, // 200% of average
		sustainedWindow:  5 * time.Minute,
	}
}

// Detect detects anomalies in traffic patterns
func (ad *AnomalyDetector) Detect(deviceID string, history *DeviceTrafficHistory, sample TrafficSample, capacity float64) []*types.TrafficAnomaly {
	anomalies := []*types.TrafficAnomaly{}

	// Check for bandwidth spike
	if spike := ad.detectSpike(deviceID, history, sample); spike != nil {
		anomalies = append(anomalies, spike)
	}

	// Check for sustained high usage
	if sustained := ad.detectSustainedHigh(deviceID, history, capacity); sustained != nil {
		anomalies = append(anomalies, sustained)
	}

	// Check for unusual patterns
	if unusual := ad.detectUnusualPattern(deviceID, history, sample); unusual != nil {
		anomalies = append(anomalies, unusual)
	}

	// Check for capacity threshold
	if threshold := ad.detectCapacityThreshold(deviceID, sample, capacity); threshold != nil {
		anomalies = append(anomalies, threshold)
	}

	return anomalies
}

// detectSpike detects sudden traffic spikes
func (ad *AnomalyDetector) detectSpike(deviceID string, history *DeviceTrafficHistory, sample TrafficSample) *types.TrafficAnomaly {
	if len(history.Samples) < 5 {
		return nil
	}

	// Calculate recent average
	recentSamples := history.Samples[len(history.Samples)-5:]
	var avgUpload, avgDownload float64
	for _, s := range recentSamples {
		avgUpload += s.UploadMbps
		avgDownload += s.DownloadMbps
	}
	avgUpload /= float64(len(recentSamples))
	avgDownload /= float64(len(recentSamples))

	// Check for spike
	totalCurrent := sample.UploadMbps + sample.DownloadMbps
	totalAverage := avgUpload + avgDownload

	if totalAverage > 0 && totalCurrent > totalAverage*ad.spikeThreshold {
		return &types.TrafficAnomaly{
			ID:          fmt.Sprintf("spike-%s-%d", deviceID, time.Now().Unix()),
			DeviceID:    deviceID,
			Type:        "spike",
			Severity:    ad.calculateSeverity(totalCurrent / totalAverage),
			Description: fmt.Sprintf("Traffic spike detected: %.2f Mbps (%.1fx average)", totalCurrent, totalCurrent/totalAverage),
			Value:       totalCurrent,
			Threshold:   totalAverage * ad.spikeThreshold,
			StartTime:   time.Now().Unix(),
			Resolved:    false,
		}
	}

	return nil
}

// detectSustainedHigh detects sustained high bandwidth usage
func (ad *AnomalyDetector) detectSustainedHigh(deviceID string, history *DeviceTrafficHistory, capacity float64) *types.TrafficAnomaly {
	if len(history.Samples) < 10 {
		return nil
	}

	// Check recent samples for sustained high usage
	threshold := capacity * ad.threshold
	windowStart := time.Now().Add(-ad.sustainedWindow)
	highCount := 0
	totalSamples := 0

	for _, sample := range history.Samples {
		if sample.Timestamp.After(windowStart) {
			totalSamples++
			total := sample.UploadMbps + sample.DownloadMbps
			if total > threshold {
				highCount++
			}
		}
	}

	// If more than 80% of samples are high
	if totalSamples > 0 && float64(highCount)/float64(totalSamples) > 0.8 {
		avgUsage := (history.AverageUpload + history.AverageDownload)
		return &types.TrafficAnomaly{
			ID:          fmt.Sprintf("sustained-%s-%d", deviceID, time.Now().Unix()),
			DeviceID:    deviceID,
			Type:        "sustained_high",
			Severity:    "high",
			Description: fmt.Sprintf("Sustained high bandwidth usage: %.2f Mbps average", avgUsage),
			Value:       avgUsage,
			Threshold:   threshold,
			StartTime:   windowStart.Unix(),
			Resolved:    false,
		}
	}

	return nil
}

// detectUnusualPattern detects unusual traffic patterns
func (ad *AnomalyDetector) detectUnusualPattern(deviceID string, history *DeviceTrafficHistory, sample TrafficSample) *types.TrafficAnomaly {
	if len(history.Samples) < 20 {
		return nil
	}

	// Calculate standard deviation
	var values []float64
	for _, s := range history.Samples {
		values = append(values, s.UploadMbps+s.DownloadMbps)
	}
	
	mean, stdDev := calculateStats(values)
	current := sample.UploadMbps + sample.DownloadMbps

	// Check if current value is outside 3 standard deviations
	if math.Abs(current-mean) > 3*stdDev {
		return &types.TrafficAnomaly{
			ID:          fmt.Sprintf("unusual-%s-%d", deviceID, time.Now().Unix()),
			DeviceID:    deviceID,
			Type:        "unusual_pattern",
			Severity:    "medium",
			Description: fmt.Sprintf("Unusual traffic pattern detected: %.2f Mbps (mean: %.2f, stddev: %.2f)", current, mean, stdDev),
			Value:       current,
			Threshold:   mean + 3*stdDev,
			StartTime:   time.Now().Unix(),
			Resolved:    false,
		}
	}

	// Check for unusual upload/download ratio
	if sample.UploadMbps > 0 && sample.DownloadMbps > 0 {
		ratio := sample.UploadMbps / sample.DownloadMbps
		historicalRatio := history.AverageUpload / math.Max(history.AverageDownload, 0.1)
		
		if math.Abs(ratio-historicalRatio) > historicalRatio*2 {
			return &types.TrafficAnomaly{
				ID:          fmt.Sprintf("ratio-%s-%d", deviceID, time.Now().Unix()),
				DeviceID:    deviceID,
				Type:        "unusual_protocol",
				Severity:    "low",
				Description: fmt.Sprintf("Unusual upload/download ratio: %.2f (normal: %.2f)", ratio, historicalRatio),
				Value:       ratio,
				Threshold:   historicalRatio,
				StartTime:   time.Now().Unix(),
				Resolved:    false,
			}
		}
	}

	return nil
}

// detectCapacityThreshold detects when traffic exceeds capacity threshold
func (ad *AnomalyDetector) detectCapacityThreshold(deviceID string, sample TrafficSample, capacity float64) *types.TrafficAnomaly {
	total := sample.UploadMbps + sample.DownloadMbps
	threshold := capacity * ad.threshold

	if total > threshold {
		percentUsed := (total / capacity) * 100
		return &types.TrafficAnomaly{
			ID:          fmt.Sprintf("capacity-%s-%d", deviceID, time.Now().Unix()),
			DeviceID:    deviceID,
			Type:        "capacity_threshold",
			Severity:    ad.calculateCapacitySeverity(percentUsed),
			Description: fmt.Sprintf("Device using %.1f%% of network capacity", percentUsed),
			Value:       total,
			Threshold:   threshold,
			StartTime:   time.Now().Unix(),
			Resolved:    false,
		}
	}

	return nil
}

// calculateSeverity calculates anomaly severity based on magnitude
func (ad *AnomalyDetector) calculateSeverity(magnitude float64) string {
	if magnitude > 5 {
		return "critical"
	} else if magnitude > 3 {
		return "high"
	} else if magnitude > 2 {
		return "medium"
	}
	return "low"
}

// calculateCapacitySeverity calculates severity based on capacity usage
func (ad *AnomalyDetector) calculateCapacitySeverity(percentUsed float64) string {
	if percentUsed > 95 {
		return "critical"
	} else if percentUsed > 85 {
		return "high"
	} else if percentUsed > 75 {
		return "medium"
	}
	return "low"
}

// calculateStats calculates mean and standard deviation
func calculateStats(values []float64) (mean, stdDev float64) {
	if len(values) == 0 {
		return 0, 0
	}

	// Calculate mean
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	mean = sum / float64(len(values))

	// Calculate standard deviation
	sumSquares := 0.0
	for _, v := range values {
		diff := v - mean
		sumSquares += diff * diff
	}
	variance := sumSquares / float64(len(values))
	stdDev = math.Sqrt(variance)

	return mean, stdDev
}

// ResolveAnomaly marks an anomaly as resolved
func (ad *AnomalyDetector) ResolveAnomaly(anomaly *types.TrafficAnomaly) {
	anomaly.Resolved = true
	anomaly.EndTime = time.Now().Unix()
}

// GetAnomalyHistory returns historical anomalies
func (ad *AnomalyDetector) GetAnomalyHistory(deviceID string, duration time.Duration) []*types.TrafficAnomaly {
	// This would query from storage in a real implementation
	// For now, return empty slice
	return []*types.TrafficAnomaly{}
}