package qos

import (
	"fmt"
	"time"

	"rtk_controller/pkg/types"
)

// PolicyEngine generates and evaluates QoS policies
type PolicyEngine struct {
	maxBandwidth float64
	policies     []PolicyRule
}

// PolicyRule represents a QoS policy rule
type PolicyRule struct {
	Name      string
	Condition func(*TrafficPattern) bool
	Action    func(*TrafficPattern) *types.QoSRecommendation
	Priority  int
}

// NewPolicyEngine creates a new policy engine
func NewPolicyEngine(maxBandwidth float64) *PolicyEngine {
	pe := &PolicyEngine{
		maxBandwidth: maxBandwidth,
	}

	// Define default policy rules
	pe.policies = []PolicyRule{
		{
			Name:     "streaming_optimization",
			Priority: 1,
			Condition: func(p *TrafficPattern) bool {
				return p.Type == "streaming"
			},
			Action: pe.streamingOptimization,
		},
		{
			Name:     "gaming_latency",
			Priority: 2,
			Condition: func(p *TrafficPattern) bool {
				return p.Type == "gaming"
			},
			Action: pe.gamingLatencyOptimization,
		},
		{
			Name:     "upload_heavy_scheduling",
			Priority: 3,
			Condition: func(p *TrafficPattern) bool {
				return p.Type == "upload_heavy"
			},
			Action: pe.uploadScheduling,
		},
		{
			Name:     "peak_hour_management",
			Priority: 4,
			Condition: func(p *TrafficPattern) bool {
				return len(p.PeakHours) > 3
			},
			Action: pe.peakHourManagement,
		},
	}

	return pe
}

// GenerateRecommendation generates QoS recommendation based on traffic pattern
func (pe *PolicyEngine) GenerateRecommendation(deviceID string, pattern *TrafficPattern) *types.QoSRecommendation {
	if pattern == nil {
		return nil
	}

	// Evaluate all policies and find matching ones
	for _, policy := range pe.policies {
		if policy.Condition(pattern) {
			return policy.Action(pattern)
		}
	}

	// Default recommendation if no specific policy matches
	return pe.defaultRecommendation(pattern)
}

// streamingOptimization creates recommendation for streaming traffic
func (pe *PolicyEngine) streamingOptimization(pattern *TrafficPattern) *types.QoSRecommendation {
	return &types.QoSRecommendation{
		ID:          fmt.Sprintf("rec-stream-%s-%d", pattern.DeviceID, time.Now().Unix()),
		Type:        "priority_queue",
		Reason:      "Streaming traffic detected",
		Description: "Create high-priority queue for streaming traffic to ensure smooth playback",
		Priority:    1,
		Impact:      "high",
		Rule: &types.QueueInfo{
			QueueID:      fmt.Sprintf("stream-q-%s", pattern.DeviceID),
			Priority:     8,
			BandwidthPct: 40,
		},
		Devices:   []string{pattern.DeviceID},
		CreatedAt: time.Now().Unix(),
	}
}

// gamingLatencyOptimization creates recommendation for gaming traffic
func (pe *PolicyEngine) gamingLatencyOptimization(pattern *TrafficPattern) *types.QoSRecommendation {
	return &types.QoSRecommendation{
		ID:          fmt.Sprintf("rec-game-%s-%d", pattern.DeviceID, time.Now().Unix()),
		Type:        "traffic_shaping",
		Reason:      "Gaming traffic detected - requires low latency",
		Description: "Prioritize gaming packets and minimize buffering to reduce latency",
		Priority:    1,
		Impact:      "high",
		Rule: &types.TrafficRule{
			RuleID:   fmt.Sprintf("game-rule-%s", pattern.DeviceID),
			Protocol: "udp",
			Action:   "prioritize",
			Priority: 9,
		},
		Devices:   []string{pattern.DeviceID},
		CreatedAt: time.Now().Unix(),
	}
}

// uploadScheduling creates recommendation for upload-heavy traffic
func (pe *PolicyEngine) uploadScheduling(pattern *TrafficPattern) *types.QoSRecommendation {
	return &types.QoSRecommendation{
		ID:          fmt.Sprintf("rec-upload-%s-%d", pattern.DeviceID, time.Now().Unix()),
		Type:        "bandwidth_cap",
		Reason:      "Heavy upload traffic detected",
		Description: "Implement upload bandwidth cap to prevent network saturation",
		Priority:    2,
		Impact:      "medium",
		Rule: &types.BandwidthRule{
			RuleID:      fmt.Sprintf("upload-cap-%s", pattern.DeviceID),
			Target:      pattern.DeviceID,
			UploadLimit: int(pe.maxBandwidth * 0.3), // 30% of total
			Priority:    5,
			Enabled:     false,
		},
		Devices:   []string{pattern.DeviceID},
		CreatedAt: time.Now().Unix(),
	}
}

// peakHourManagement creates recommendation for peak hour traffic
func (pe *PolicyEngine) peakHourManagement(pattern *TrafficPattern) *types.QoSRecommendation {
	return &types.QoSRecommendation{
		ID:          fmt.Sprintf("rec-peak-%s-%d", pattern.DeviceID, time.Now().Unix()),
		Type:        "traffic_shaping",
		Reason:      fmt.Sprintf("High usage during peak hours: %v", pattern.PeakHours),
		Description: "Implement time-based traffic shaping during peak hours",
		Priority:    3,
		Impact:      "medium",
		Rule: &types.TrafficRule{
			RuleID:   fmt.Sprintf("peak-rule-%s", pattern.DeviceID),
			Protocol: "all",
			Action:   "throttle",
			Priority: 4,
		},
		Devices:   []string{pattern.DeviceID},
		CreatedAt: time.Now().Unix(),
	}
}

// defaultRecommendation creates a default recommendation
func (pe *PolicyEngine) defaultRecommendation(pattern *TrafficPattern) *types.QoSRecommendation {
	return &types.QoSRecommendation{
		ID:          fmt.Sprintf("rec-default-%s-%d", pattern.DeviceID, time.Now().Unix()),
		Type:        "priority_queue",
		Reason:      "Standard traffic pattern",
		Description: "Apply default QoS policy for fair bandwidth sharing",
		Priority:    5,
		Impact:      "low",
		Rule: &types.QueueInfo{
			QueueID:      fmt.Sprintf("default-q-%s", pattern.DeviceID),
			Priority:     5,
			BandwidthPct: 20,
		},
		Devices:   []string{pattern.DeviceID},
		CreatedAt: time.Now().Unix(),
	}
}

// EvaluatePolicies evaluates all policies against current traffic
func (pe *PolicyEngine) EvaluatePolicies(patterns map[string]*TrafficPattern) []*types.QoSRecommendation {
	recommendations := []*types.QoSRecommendation{}

	for deviceID, pattern := range patterns {
		if rec := pe.GenerateRecommendation(deviceID, pattern); rec != nil {
			recommendations = append(recommendations, rec)
		}
	}

	return recommendations
}

// AddCustomPolicy adds a custom policy rule
func (pe *PolicyEngine) AddCustomPolicy(rule PolicyRule) {
	pe.policies = append(pe.policies, rule)
}

// SimulatePolicy simulates the effect of applying a policy
func (pe *PolicyEngine) SimulatePolicy(rec *types.QoSRecommendation, currentStats *types.TrafficStats) *PolicySimulation {
	sim := &PolicySimulation{
		RecommendationID: rec.ID,
		Before:           currentStats,
	}

	// Simulate the effect based on recommendation type
	switch rec.Type {
	case "bandwidth_cap":
		sim.After = pe.simulateBandwidthCap(rec, currentStats)
	case "traffic_shaping":
		sim.After = pe.simulateTrafficShaping(rec, currentStats)
	case "priority_queue":
		sim.After = pe.simulatePriorityQueue(rec, currentStats)
	}

	// Calculate impact
	sim.ImpactScore = pe.calculateImpact(sim.Before, sim.After)
	sim.Description = pe.describeImpact(sim)

	return sim
}

// PolicySimulation represents the simulated effect of a policy
type PolicySimulation struct {
	RecommendationID string
	Before           *types.TrafficStats
	After            *types.TrafficStats
	ImpactScore      float64
	Description      string
}

// simulateBandwidthCap simulates the effect of bandwidth capping
func (pe *PolicyEngine) simulateBandwidthCap(rec *types.QoSRecommendation, stats *types.TrafficStats) *types.TrafficStats {
	simulated := *stats // Copy stats

	if rule, ok := rec.Rule.(*types.BandwidthRule); ok {
		// Reduce bandwidth for affected devices
		for i, device := range simulated.DeviceTraffic {
			for _, affectedID := range rec.Devices {
				if device.DeviceID == affectedID {
					if rule.UploadLimit > 0 {
						simulated.DeviceTraffic[i].UploadMbps = min(device.UploadMbps, float64(rule.UploadLimit))
					}
					if rule.DownloadLimit > 0 {
						simulated.DeviceTraffic[i].DownloadMbps = min(device.DownloadMbps, float64(rule.DownloadLimit))
					}
				}
			}
		}

		// Recalculate total usage
		simulated.UsedBandwidthMbps = 0
		for _, device := range simulated.DeviceTraffic {
			simulated.UsedBandwidthMbps += device.UploadMbps + device.DownloadMbps
		}
	}

	return &simulated
}

// simulateTrafficShaping simulates the effect of traffic shaping
func (pe *PolicyEngine) simulateTrafficShaping(rec *types.QoSRecommendation, stats *types.TrafficStats) *types.TrafficStats {
	simulated := *stats // Copy stats

	if rule, ok := rec.Rule.(*types.TrafficRule); ok {
		// Simulate the effect based on action
		switch rule.Action {
		case "throttle":
			// Reduce bandwidth by 30% for affected devices
			for i, device := range simulated.DeviceTraffic {
				for _, affectedID := range rec.Devices {
					if device.DeviceID == affectedID {
						simulated.DeviceTraffic[i].UploadMbps *= 0.7
						simulated.DeviceTraffic[i].DownloadMbps *= 0.7
					}
				}
			}
		case "block":
			// Remove traffic from affected devices
			for i, device := range simulated.DeviceTraffic {
				for _, affectedID := range rec.Devices {
					if device.DeviceID == affectedID {
						simulated.DeviceTraffic[i].UploadMbps = 0
						simulated.DeviceTraffic[i].DownloadMbps = 0
					}
				}
			}
		}
	}

	return &simulated
}

// simulatePriorityQueue simulates the effect of priority queuing
func (pe *PolicyEngine) simulatePriorityQueue(rec *types.QoSRecommendation, stats *types.TrafficStats) *types.TrafficStats {
	simulated := *stats // Copy stats

	// Priority queuing doesn't change total bandwidth but redistributes it
	// This is a simplified simulation
	if queue, ok := rec.Rule.(*types.QueueInfo); ok {
		allocatedBandwidth := stats.TotalBandwidthMbps * (queue.BandwidthPct / 100)

		// Ensure high-priority devices get their allocated bandwidth
		for i, device := range simulated.DeviceTraffic {
			for _, affectedID := range rec.Devices {
				if device.DeviceID == affectedID {
					// Guarantee minimum bandwidth
					totalNeeded := device.UploadMbps + device.DownloadMbps
					if totalNeeded > allocatedBandwidth {
						ratio := allocatedBandwidth / totalNeeded
						simulated.DeviceTraffic[i].UploadMbps *= ratio
						simulated.DeviceTraffic[i].DownloadMbps *= ratio
					}
				}
			}
		}
	}

	return &simulated
}

// calculateImpact calculates the impact score of a policy
func (pe *PolicyEngine) calculateImpact(before, after *types.TrafficStats) float64 {
	if before.UsedBandwidthMbps == 0 {
		return 0
	}

	// Calculate percentage change in bandwidth usage
	change := (before.UsedBandwidthMbps - after.UsedBandwidthMbps) / before.UsedBandwidthMbps
	return change * 100
}

// describeImpact describes the impact of a policy
func (pe *PolicyEngine) describeImpact(sim *PolicySimulation) string {
	if sim.ImpactScore > 20 {
		return fmt.Sprintf("High impact: %.1f%% bandwidth reduction", sim.ImpactScore)
	} else if sim.ImpactScore > 10 {
		return fmt.Sprintf("Medium impact: %.1f%% bandwidth reduction", sim.ImpactScore)
	} else if sim.ImpactScore > 0 {
		return fmt.Sprintf("Low impact: %.1f%% bandwidth reduction", sim.ImpactScore)
	}
	return "No significant impact on bandwidth usage"
}

// min returns the minimum of two float64 values
func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
