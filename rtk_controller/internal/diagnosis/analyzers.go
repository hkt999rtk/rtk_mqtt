package diagnosis

import (
	"fmt"
	"time"

	"rtk_controller/pkg/types"
	"rtk_controller/pkg/utils"

	log "github.com/sirupsen/logrus"
)

// WiFiAnalyzer analyzes WiFi-related diagnosis data
type WiFiAnalyzer struct{}

func (a *WiFiAnalyzer) Name() string {
	return "wifi_analyzer"
}

func (a *WiFiAnalyzer) Type() string {
	return "builtin"
}

func (a *WiFiAnalyzer) SupportedTypes() []string {
	return []string{"wifi", "telemetry"}
}

func (a *WiFiAnalyzer) GetInfo() *types.AnalyzerInfo {
	return &types.AnalyzerInfo{
		Name:           a.Name(),
		Type:           a.Type(),
		Version:        "1.0.0",
		Description:    "Analyzes WiFi connectivity and performance issues",
		SupportedTypes: a.SupportedTypes(),
		Enabled:        true,
		Config:         make(map[string]interface{}),
	}
}

func (a *WiFiAnalyzer) Analyze(data []*types.DiagnosisData, config types.DiagnosisConfig) (*types.DiagnosisResult, error) {
	startTime := time.Now()

	result := &types.DiagnosisResult{
		ID:              utils.GenerateMessageID(),
		AnalyzerType:    a.Type(),
		AnalyzerName:    a.Name(),
		Status:          "analyzing",
		Confidence:      0.8,
		Issues:          []types.DiagnosisIssue{},
		Recommendations: []types.Recommendation{},
		Metrics:         make(map[string]interface{}),
		CreatedAt:       time.Now(),
	}

	if len(data) > 0 {
		result.DiagnosisID = data[0].ID
		result.DeviceID = data[0].DeviceID
	}

	log.WithFields(log.Fields{
		"analyzer":   a.Name(),
		"data_count": len(data),
	}).Debug("Starting WiFi analysis")

	// Analyze WiFi data
	var wifiData []*types.DiagnosisData
	for _, d := range data {
		if d.Type == "wifi" || (d.Type == "telemetry" && a.isWiFiTelemetry(d)) {
			wifiData = append(wifiData, d)
		}
	}

	if len(wifiData) == 0 {
		result.Status = "completed"
		result.Confidence = 0.1
		result.ExecutionTime = time.Since(startTime).Milliseconds()
		return result, nil
	}

	// Analyze signal strength issues
	a.analyzeSignalStrength(wifiData, result)

	// Analyze connectivity issues
	a.analyzeConnectivity(wifiData, result)

	// Analyze performance issues
	a.analyzePerformance(wifiData, result)

	// Generate metrics
	result.Metrics["analyzed_data_points"] = len(wifiData)
	result.Metrics["issues_found"] = len(result.Issues)

	result.Status = "completed"
	now := time.Now()
	result.CompletedAt = &now
	result.ExecutionTime = time.Since(startTime).Milliseconds()

	log.WithFields(log.Fields{
		"analyzer":          a.Name(),
		"issues":            len(result.Issues),
		"recommendations":   len(result.Recommendations),
		"execution_time_ms": result.ExecutionTime,
	}).Info("WiFi analysis completed")

	return result, nil
}

func (a *WiFiAnalyzer) isWiFiTelemetry(data *types.DiagnosisData) bool {
	// Check if telemetry data contains WiFi metrics
	for key := range data.Metrics {
		if key == "signal_strength" || key == "snr" || key == "link_quality" {
			return true
		}
	}
	return false
}

func (a *WiFiAnalyzer) analyzeSignalStrength(data []*types.DiagnosisData, result *types.DiagnosisResult) {
	var signalValues []float64

	for _, d := range data {
		if signal, ok := d.Metrics["signal_strength"]; ok {
			signalValues = append(signalValues, signal)
		}
	}

	if len(signalValues) == 0 {
		return
	}

	// Calculate average signal strength
	var sum float64
	for _, val := range signalValues {
		sum += val
	}
	avgSignal := sum / float64(len(signalValues))

	// Analyze signal strength (assuming dBm values, more negative = weaker)
	if avgSignal < -80 {
		issue := types.DiagnosisIssue{
			ID:                 utils.GenerateMessageID(),
			Type:               "connectivity",
			Severity:           "high",
			Title:              "Poor WiFi Signal Strength",
			Description:        fmt.Sprintf("Average signal strength is %.1f dBm, which is below recommended level", avgSignal),
			Impact:             "May cause connection drops and poor performance",
			Likelihood:         0.9,
			Evidence:           map[string]interface{}{"avg_signal_strength": avgSignal},
			AffectedComponents: []string{"wifi", "connectivity"},
		}

		recommendation := types.Recommendation{
			ID:          utils.GenerateMessageID(),
			Type:        "immediate",
			Priority:    "high",
			Title:       "Improve WiFi Signal Strength",
			Description: "Move device closer to access point or install signal repeater",
			Actions: []types.RecommendedAction{
				{
					Type:        "manual",
					Description: "Relocate device closer to WiFi access point",
					Risk:        "low",
				},
				{
					Type:        "manual",
					Description: "Check for physical obstructions blocking WiFi signal",
					Risk:        "low",
				},
			},
			EstimatedTime:  "15 minutes",
			RequiredSkills: []string{"basic"},
		}

		result.Issues = append(result.Issues, issue)
		result.Recommendations = append(result.Recommendations, recommendation)
	}
}

func (a *WiFiAnalyzer) analyzeConnectivity(data []*types.DiagnosisData, result *types.DiagnosisResult) {
	var disconnectionCount int
	var connectionEvents int

	for _, d := range data {
		if d.Category == "error" && d.Type == "wifi" {
			disconnectionCount++
		}
		if d.Type == "wifi" {
			connectionEvents++
		}
	}

	if connectionEvents > 0 {
		disconnectionRate := float64(disconnectionCount) / float64(connectionEvents)

		if disconnectionRate > 0.1 { // More than 10% disconnection rate
			issue := types.DiagnosisIssue{
				ID:                 utils.GenerateMessageID(),
				Type:               "connectivity",
				Severity:           "medium",
				Title:              "Frequent WiFi Disconnections",
				Description:        fmt.Sprintf("Disconnection rate is %.1f%%, indicating connectivity instability", disconnectionRate*100),
				Impact:             "Service interruptions and reduced reliability",
				Likelihood:         0.8,
				Evidence:           map[string]interface{}{"disconnection_rate": disconnectionRate},
				AffectedComponents: []string{"wifi", "connectivity"},
			}

			recommendation := types.Recommendation{
				ID:          utils.GenerateMessageID(),
				Type:        "scheduled",
				Priority:    "medium",
				Title:       "Investigate WiFi Stability",
				Description: "Check WiFi configuration and network stability",
				Actions: []types.RecommendedAction{
					{
						Type:        "command",
						Description: "Run WiFi diagnostics",
						Command:     "run_diagnostics",
						Parameters:  map[string]interface{}{"type": "wifi"},
						Risk:        "low",
					},
				},
				EstimatedTime:  "30 minutes",
				RequiredSkills: []string{"intermediate"},
			}

			result.Issues = append(result.Issues, issue)
			result.Recommendations = append(result.Recommendations, recommendation)
		}
	}
}

func (a *WiFiAnalyzer) analyzePerformance(data []*types.DiagnosisData, result *types.DiagnosisResult) {
	var throughputValues []float64
	var latencyValues []float64

	for _, d := range data {
		if throughput, ok := d.Metrics["throughput"]; ok {
			throughputValues = append(throughputValues, throughput)
		}
		if latency, ok := d.Metrics["latency"]; ok {
			latencyValues = append(latencyValues, latency)
		}
	}

	// Analyze throughput
	if len(throughputValues) > 0 {
		var sum float64
		for _, val := range throughputValues {
			sum += val
		}
		avgThroughput := sum / float64(len(throughputValues))

		if avgThroughput < 10 { // Less than 10 Mbps
			issue := types.DiagnosisIssue{
				ID:                 utils.GenerateMessageID(),
				Type:               "performance",
				Severity:           "medium",
				Title:              "Low WiFi Throughput",
				Description:        fmt.Sprintf("Average throughput is %.1f Mbps, below expected performance", avgThroughput),
				Impact:             "Slow data transfer and poor user experience",
				Likelihood:         0.7,
				Evidence:           map[string]interface{}{"avg_throughput_mbps": avgThroughput},
				AffectedComponents: []string{"wifi", "performance"},
			}

			result.Issues = append(result.Issues, issue)
		}
	}

	// Analyze latency
	if len(latencyValues) > 0 {
		var sum float64
		for _, val := range latencyValues {
			sum += val
		}
		avgLatency := sum / float64(len(latencyValues))

		if avgLatency > 100 { // More than 100ms
			issue := types.DiagnosisIssue{
				ID:                 utils.GenerateMessageID(),
				Type:               "performance",
				Severity:           "medium",
				Title:              "High WiFi Latency",
				Description:        fmt.Sprintf("Average latency is %.1f ms, higher than optimal", avgLatency),
				Impact:             "Delayed responses and poor real-time performance",
				Likelihood:         0.6,
				Evidence:           map[string]interface{}{"avg_latency_ms": avgLatency},
				AffectedComponents: []string{"wifi", "performance"},
			}

			result.Issues = append(result.Issues, issue)
		}
	}
}

// NetworkAnalyzer analyzes network-related diagnosis data
type NetworkAnalyzer struct{}

func (a *NetworkAnalyzer) Name() string {
	return "network_analyzer"
}

func (a *NetworkAnalyzer) Type() string {
	return "builtin"
}

func (a *NetworkAnalyzer) SupportedTypes() []string {
	return []string{"network", "telemetry"}
}

func (a *NetworkAnalyzer) GetInfo() *types.AnalyzerInfo {
	return &types.AnalyzerInfo{
		Name:           a.Name(),
		Type:           a.Type(),
		Version:        "1.0.0",
		Description:    "Analyzes network connectivity and performance issues",
		SupportedTypes: a.SupportedTypes(),
		Enabled:        true,
		Config:         make(map[string]interface{}),
	}
}

func (a *NetworkAnalyzer) Analyze(data []*types.DiagnosisData, config types.DiagnosisConfig) (*types.DiagnosisResult, error) {
	startTime := time.Now()

	result := &types.DiagnosisResult{
		ID:              utils.GenerateMessageID(),
		AnalyzerType:    a.Type(),
		AnalyzerName:    a.Name(),
		Status:          "analyzing",
		Confidence:      0.8,
		Issues:          []types.DiagnosisIssue{},
		Recommendations: []types.Recommendation{},
		Metrics:         make(map[string]interface{}),
		CreatedAt:       time.Now(),
	}

	if len(data) > 0 {
		result.DiagnosisID = data[0].ID
		result.DeviceID = data[0].DeviceID
	}

	// Filter network-related data
	var networkData []*types.DiagnosisData
	for _, d := range data {
		if d.Type == "network" || (d.Type == "telemetry" && a.isNetworkTelemetry(d)) {
			networkData = append(networkData, d)
		}
	}

	if len(networkData) == 0 {
		result.Status = "completed"
		result.Confidence = 0.1
		result.ExecutionTime = time.Since(startTime).Milliseconds()
		return result, nil
	}

	// Analyze packet loss
	a.analyzePacketLoss(networkData, result)

	// Analyze error rates
	a.analyzeErrorRates(networkData, result)

	result.Metrics["analyzed_data_points"] = len(networkData)
	result.Metrics["issues_found"] = len(result.Issues)

	result.Status = "completed"
	now := time.Now()
	result.CompletedAt = &now
	result.ExecutionTime = time.Since(startTime).Milliseconds()

	return result, nil
}

func (a *NetworkAnalyzer) isNetworkTelemetry(data *types.DiagnosisData) bool {
	for key := range data.Metrics {
		if key == "packet_loss" || key == "tx_errors" || key == "rx_errors" {
			return true
		}
	}
	return false
}

func (a *NetworkAnalyzer) analyzePacketLoss(data []*types.DiagnosisData, result *types.DiagnosisResult) {
	var packetLossValues []float64

	for _, d := range data {
		if loss, ok := d.Metrics["packet_loss"]; ok {
			packetLossValues = append(packetLossValues, loss)
		}
	}

	if len(packetLossValues) > 0 {
		var sum float64
		for _, val := range packetLossValues {
			sum += val
		}
		avgPacketLoss := sum / float64(len(packetLossValues))

		if avgPacketLoss > 1.0 { // More than 1% packet loss
			severity := "medium"
			if avgPacketLoss > 5.0 {
				severity = "high"
			}

			issue := types.DiagnosisIssue{
				ID:                 utils.GenerateMessageID(),
				Type:               "connectivity",
				Severity:           severity,
				Title:              "High Packet Loss",
				Description:        fmt.Sprintf("Average packet loss is %.2f%%, indicating network issues", avgPacketLoss),
				Impact:             "Data transmission failures and poor performance",
				Likelihood:         0.9,
				Evidence:           map[string]interface{}{"avg_packet_loss_percent": avgPacketLoss},
				AffectedComponents: []string{"network", "connectivity"},
			}

			result.Issues = append(result.Issues, issue)
		}
	}
}

func (a *NetworkAnalyzer) analyzeErrorRates(data []*types.DiagnosisData, result *types.DiagnosisResult) {
	var txErrors, rxErrors []float64

	for _, d := range data {
		if tx, ok := d.Metrics["tx_errors"]; ok {
			txErrors = append(txErrors, tx)
		}
		if rx, ok := d.Metrics["rx_errors"]; ok {
			rxErrors = append(rxErrors, rx)
		}
	}

	// Check for high error rates
	if len(txErrors) > 0 || len(rxErrors) > 0 {
		var totalErrors float64
		totalErrors += sumFloat64(txErrors)
		totalErrors += sumFloat64(rxErrors)

		if totalErrors > 100 { // More than 100 total errors
			issue := types.DiagnosisIssue{
				ID:                 utils.GenerateMessageID(),
				Type:               "connectivity",
				Severity:           "medium",
				Title:              "High Network Error Rate",
				Description:        fmt.Sprintf("Total network errors: %.0f", totalErrors),
				Impact:             "Network instability and data corruption",
				Likelihood:         0.8,
				Evidence:           map[string]interface{}{"total_errors": totalErrors},
				AffectedComponents: []string{"network", "hardware"},
			}

			result.Issues = append(result.Issues, issue)
		}
	}
}

// SystemAnalyzer analyzes system-related diagnosis data
type SystemAnalyzer struct{}

func (a *SystemAnalyzer) Name() string {
	return "system_analyzer"
}

func (a *SystemAnalyzer) Type() string {
	return "builtin"
}

func (a *SystemAnalyzer) SupportedTypes() []string {
	return []string{"system", "telemetry"}
}

func (a *SystemAnalyzer) GetInfo() *types.AnalyzerInfo {
	return &types.AnalyzerInfo{
		Name:           a.Name(),
		Type:           a.Type(),
		Version:        "1.0.0",
		Description:    "Analyzes system performance and resource utilization",
		SupportedTypes: a.SupportedTypes(),
		Enabled:        true,
		Config:         make(map[string]interface{}),
	}
}

func (a *SystemAnalyzer) Analyze(data []*types.DiagnosisData, config types.DiagnosisConfig) (*types.DiagnosisResult, error) {
	startTime := time.Now()

	result := &types.DiagnosisResult{
		ID:              utils.GenerateMessageID(),
		AnalyzerType:    a.Type(),
		AnalyzerName:    a.Name(),
		Status:          "analyzing",
		Confidence:      0.9,
		Issues:          []types.DiagnosisIssue{},
		Recommendations: []types.Recommendation{},
		Metrics:         make(map[string]interface{}),
		CreatedAt:       time.Now(),
	}

	if len(data) > 0 {
		result.DiagnosisID = data[0].ID
		result.DeviceID = data[0].DeviceID
	}

	// Filter system-related data
	var systemData []*types.DiagnosisData
	for _, d := range data {
		if d.Type == "system" || (d.Type == "telemetry" && a.isSystemTelemetry(d)) {
			systemData = append(systemData, d)
		}
	}

	if len(systemData) == 0 {
		result.Status = "completed"
		result.Confidence = 0.1
		result.ExecutionTime = time.Since(startTime).Milliseconds()
		return result, nil
	}

	// Analyze resource usage
	a.analyzeResourceUsage(systemData, result)

	// Analyze system health
	a.analyzeSystemHealth(systemData, result)

	result.Metrics["analyzed_data_points"] = len(systemData)
	result.Metrics["issues_found"] = len(result.Issues)

	result.Status = "completed"
	now := time.Now()
	result.CompletedAt = &now
	result.ExecutionTime = time.Since(startTime).Milliseconds()

	return result, nil
}

func (a *SystemAnalyzer) isSystemTelemetry(data *types.DiagnosisData) bool {
	for key := range data.Metrics {
		if key == "cpu_usage" || key == "memory_usage" || key == "disk_usage" {
			return true
		}
	}
	return false
}

func (a *SystemAnalyzer) analyzeResourceUsage(data []*types.DiagnosisData, result *types.DiagnosisResult) {
	var cpuUsage, memUsage, diskUsage []float64

	for _, d := range data {
		if cpu, ok := d.Metrics["cpu_usage"]; ok {
			cpuUsage = append(cpuUsage, cpu)
		}
		if mem, ok := d.Metrics["memory_usage"]; ok {
			memUsage = append(memUsage, mem)
		}
		if disk, ok := d.Metrics["disk_usage"]; ok {
			diskUsage = append(diskUsage, disk)
		}
	}

	// Analyze CPU usage
	if len(cpuUsage) > 0 {
		avgCPU := sumFloat64(cpuUsage) / float64(len(cpuUsage))
		if avgCPU > 80 {
			severity := "medium"
			if avgCPU > 95 {
				severity = "high"
			}

			issue := types.DiagnosisIssue{
				ID:                 utils.GenerateMessageID(),
				Type:               "performance",
				Severity:           severity,
				Title:              "High CPU Usage",
				Description:        fmt.Sprintf("Average CPU usage is %.1f%%, indicating high system load", avgCPU),
				Impact:             "Reduced system responsiveness and performance",
				Likelihood:         0.9,
				Evidence:           map[string]interface{}{"avg_cpu_usage_percent": avgCPU},
				AffectedComponents: []string{"cpu", "system"},
			}

			result.Issues = append(result.Issues, issue)
		}
	}

	// Analyze memory usage
	if len(memUsage) > 0 {
		avgMem := sumFloat64(memUsage) / float64(len(memUsage))
		if avgMem > 85 {
			severity := "medium"
			if avgMem > 95 {
				severity = "high"
			}

			issue := types.DiagnosisIssue{
				ID:                 utils.GenerateMessageID(),
				Type:               "performance",
				Severity:           severity,
				Title:              "High Memory Usage",
				Description:        fmt.Sprintf("Average memory usage is %.1f%%, approaching system limits", avgMem),
				Impact:             "Risk of out-of-memory errors and system instability",
				Likelihood:         0.8,
				Evidence:           map[string]interface{}{"avg_memory_usage_percent": avgMem},
				AffectedComponents: []string{"memory", "system"},
			}

			result.Issues = append(result.Issues, issue)
		}
	}

	// Analyze disk usage
	if len(diskUsage) > 0 {
		avgDisk := sumFloat64(diskUsage) / float64(len(diskUsage))
		if avgDisk > 90 {
			severity := "high"
			if avgDisk > 98 {
				severity = "critical"
			}

			issue := types.DiagnosisIssue{
				ID:                 utils.GenerateMessageID(),
				Type:               "performance",
				Severity:           severity,
				Title:              "High Disk Usage",
				Description:        fmt.Sprintf("Average disk usage is %.1f%%, critical storage level", avgDisk),
				Impact:             "Risk of system failure and data loss",
				Likelihood:         0.9,
				Evidence:           map[string]interface{}{"avg_disk_usage_percent": avgDisk},
				AffectedComponents: []string{"storage", "system"},
			}

			result.Issues = append(result.Issues, issue)
		}
	}
}

func (a *SystemAnalyzer) analyzeSystemHealth(data []*types.DiagnosisData, result *types.DiagnosisResult) {
	var temperatures []float64
	var errorCounts []float64

	for _, d := range data {
		if temp, ok := d.Metrics["temperature"]; ok {
			temperatures = append(temperatures, temp)
		}
		if errors, ok := d.Metrics["error_rate"]; ok {
			errorCounts = append(errorCounts, errors)
		}
	}

	// Check temperature
	if len(temperatures) > 0 {
		avgTemp := sumFloat64(temperatures) / float64(len(temperatures))
		if avgTemp > 85 { // Assuming Celsius
			severity := "medium"
			if avgTemp > 95 {
				severity = "high"
			}

			issue := types.DiagnosisIssue{
				ID:                 utils.GenerateMessageID(),
				Type:               "hardware",
				Severity:           severity,
				Title:              "High System Temperature",
				Description:        fmt.Sprintf("Average temperature is %.1f°C, above normal operating range", avgTemp),
				Impact:             "Risk of hardware damage and system instability",
				Likelihood:         0.7,
				Evidence:           map[string]interface{}{"avg_temperature_celsius": avgTemp},
				AffectedComponents: []string{"hardware", "cooling"},
			}

			result.Issues = append(result.Issues, issue)
		}
	}
}

// Helper function
func sumFloat64(values []float64) float64 {
	var sum float64
	for _, val := range values {
		sum += val
	}
	return sum
}
