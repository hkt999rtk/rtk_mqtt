package topology

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"
)

// NetworkDiagnosticsRenderer provides various output formats for diagnostic reports
type NetworkDiagnosticsRenderer struct {
	config RendererConfig
}

// RendererConfig holds configuration for report rendering
type RendererConfig struct {
	ColorEnabled   bool
	MaxWidth       int
	CompactMode    bool
	ShowTimestamps bool
	ShowDetails    bool
	IncludeCharts  bool
	IncludeTrends  bool
}

// ReportFormat defines the output format for diagnostic reports
type ReportFormat string

const (
	FormatText     ReportFormat = "text"
	FormatJSON     ReportFormat = "json"
	FormatHTML     ReportFormat = "html"
	FormatMarkdown ReportFormat = "markdown"
	FormatCSV      ReportFormat = "csv"
	FormatXML      ReportFormat = "xml"
	FormatPDF      ReportFormat = "pdf"
)

// NewNetworkDiagnosticsRenderer creates a new diagnostics renderer
func NewNetworkDiagnosticsRenderer(config RendererConfig) *NetworkDiagnosticsRenderer {
	return &NetworkDiagnosticsRenderer{
		config: config,
	}
}

// RenderReport renders a diagnostic report in the specified format
func (ndr *NetworkDiagnosticsRenderer) RenderReport(
	report *NetworkDiagnosticReport,
	format ReportFormat,
	writer io.Writer,
) error {
	switch format {
	case FormatText:
		return ndr.renderText(report, writer)
	case FormatJSON:
		return ndr.renderJSON(report, writer)
	case FormatHTML:
		return ndr.renderHTML(report, writer)
	case FormatMarkdown:
		return ndr.renderMarkdown(report, writer)
	case FormatCSV:
		return ndr.renderCSV(report, writer)
	case FormatXML:
		return ndr.renderXML(report, writer)
	default:
		return fmt.Errorf("unsupported report format: %s", format)
	}
}

// renderText renders the report in plain text format
func (ndr *NetworkDiagnosticsRenderer) renderText(report *NetworkDiagnosticReport, writer io.Writer) error {
	// Header
	fmt.Fprintf(writer, "NETWORK DIAGNOSTIC REPORT\n")
	fmt.Fprintf(writer, "=========================\n\n")

	// Report metadata
	fmt.Fprintf(writer, "Report ID:      %s\n", report.ID)
	fmt.Fprintf(writer, "Generated:      %s\n", report.GeneratedAt.Format(time.RFC3339))
	fmt.Fprintf(writer, "Report Type:    %s\n", report.ReportType)
	fmt.Fprintf(writer, "Detail Level:   %s\n", report.DetailLevel)
	fmt.Fprintf(writer, "Time Window:    %s to %s (%v)\n",
		report.TimeWindow.StartTime.Format("2006-01-02 15:04:05"),
		report.TimeWindow.EndTime.Format("2006-01-02 15:04:05"),
		report.TimeWindow.Duration)
	fmt.Fprintf(writer, "\n")

	// Overall health summary
	fmt.Fprintf(writer, "OVERALL NETWORK HEALTH\n")
	fmt.Fprintf(writer, "======================\n")
	fmt.Fprintf(writer, "Health Status:  %s\n", strings.ToUpper(string(report.OverallHealth)))
	fmt.Fprintf(writer, "Health Score:   %.1f/100\n", report.HealthScore)
	fmt.Fprintf(writer, "\n")

	// Network overview
	ndr.renderNetworkOverview(report, writer)

	// Quality analysis
	ndr.renderQualityAnalysis(report, writer)

	// Performance analysis
	ndr.renderPerformanceAnalysis(report, writer)

	// Connectivity analysis
	ndr.renderConnectivityAnalysis(report, writer)

	// Issues
	ndr.renderIssues(report, writer)

	// Recommendations
	ndr.renderRecommendations(report, writer)

	// Predictions (if enabled)
	if len(report.Predictions) > 0 {
		ndr.renderPredictions(report, writer)
	}

	// Report statistics
	ndr.renderReportStats(report, writer)

	return nil
}

func (ndr *NetworkDiagnosticsRenderer) renderNetworkOverview(report *NetworkDiagnosticReport, writer io.Writer) {
	fmt.Fprintf(writer, "NETWORK OVERVIEW\n")
	fmt.Fprintf(writer, "================\n")
	fmt.Fprintf(writer, "Total Devices:    %d\n", report.NetworkOverview.TotalDevices)
	fmt.Fprintf(writer, "Online Devices:   %d\n", report.NetworkOverview.OnlineDevices)
	fmt.Fprintf(writer, "Offline Devices:  %d\n", report.NetworkOverview.OfflineDevices)

	if len(report.NetworkOverview.DevicesByType) > 0 {
		fmt.Fprintf(writer, "\nDevices by Type:\n")
		for deviceType, count := range report.NetworkOverview.DevicesByType {
			fmt.Fprintf(writer, "  %-15s %d\n", deviceType+":", count)
		}
	}

	fmt.Fprintf(writer, "\nBandwidth Utilization:\n")
	bw := report.NetworkOverview.BandwidthUtilization
	fmt.Fprintf(writer, "  Total Capacity: %.1f Mbps\n", bw.TotalCapacity)
	fmt.Fprintf(writer, "  Used Bandwidth: %.1f Mbps\n", bw.UsedBandwidth)
	fmt.Fprintf(writer, "  Utilization:    %.1f%%\n", bw.UtilizationRate*100)
	fmt.Fprintf(writer, "  Peak Usage:     %.1f Mbps\n", bw.PeakUsage)
	fmt.Fprintf(writer, "\n")
}

func (ndr *NetworkDiagnosticsRenderer) renderQualityAnalysis(report *NetworkDiagnosticReport, writer io.Writer) {
	fmt.Fprintf(writer, "CONNECTION QUALITY ANALYSIS\n")
	fmt.Fprintf(writer, "===========================\n")
	fmt.Fprintf(writer, "Average Quality:  %.2f\n", report.QualityAnalysis.AverageQuality)

	if len(report.QualityAnalysis.QualityDistribution) > 0 {
		fmt.Fprintf(writer, "\nQuality Distribution:\n")
		for quality, count := range report.QualityAnalysis.QualityDistribution {
			bar := ndr.renderBar(count, 20)
			fmt.Fprintf(writer, "  %-10s %s %d\n", quality+":", bar, count)
		}
	}

	if len(report.QualityAnalysis.PoorQualityDevices) > 0 {
		fmt.Fprintf(writer, "\nPoor Quality Devices:\n")
		for _, device := range report.QualityAnalysis.PoorQualityDevices {
			fmt.Fprintf(writer, "  %s (Quality: %.2f)\n", device.DeviceID, device.Quality)
			for _, issue := range device.Issues {
				fmt.Fprintf(writer, "    - %s\n", issue)
			}
		}
	}
	fmt.Fprintf(writer, "\n")
}

func (ndr *NetworkDiagnosticsRenderer) renderPerformanceAnalysis(report *NetworkDiagnosticReport, writer io.Writer) {
	fmt.Fprintf(writer, "PERFORMANCE ANALYSIS\n")
	fmt.Fprintf(writer, "====================\n")

	latency := report.Performance.LatencyAnalysis
	fmt.Fprintf(writer, "Latency Analysis:\n")
	fmt.Fprintf(writer, "  Average:        %.1f ms\n", latency.AverageLatency)
	fmt.Fprintf(writer, "  95th Percentile: %.1f ms\n", latency.P95Latency)
	fmt.Fprintf(writer, "  99th Percentile: %.1f ms\n", latency.P99Latency)

	throughput := report.Performance.ThroughputAnalysis
	fmt.Fprintf(writer, "\nThroughput Analysis:\n")
	fmt.Fprintf(writer, "  Average:        %.1f Mbps\n", throughput.AverageThroughput)
	fmt.Fprintf(writer, "  Peak:           %.1f Mbps\n", throughput.PeakThroughput)

	if len(throughput.BottleneckDevices) > 0 {
		fmt.Fprintf(writer, "  Bottlenecks:    %s\n", strings.Join(throughput.BottleneckDevices, ", "))
	}

	packetLoss := report.Performance.PacketLossAnalysis
	fmt.Fprintf(writer, "\nPacket Loss Analysis:\n")
	fmt.Fprintf(writer, "  Average:        %.2f%%\n", packetLoss.AveragePacketLoss*100)
	fmt.Fprintf(writer, "  Maximum:        %.2f%%\n", packetLoss.MaxPacketLoss*100)

	if len(report.Performance.Bottlenecks) > 0 {
		fmt.Fprintf(writer, "\nPerformance Bottlenecks:\n")
		for _, bottleneck := range report.Performance.Bottlenecks {
			fmt.Fprintf(writer, "  %s (%s): %s (Severity: %.1f)\n",
				bottleneck.DeviceID, bottleneck.Type, bottleneck.Impact, bottleneck.Severity)
		}
	}

	fmt.Fprintf(writer, "\n")
}

func (ndr *NetworkDiagnosticsRenderer) renderConnectivityAnalysis(report *NetworkDiagnosticReport, writer io.Writer) {
	fmt.Fprintf(writer, "CONNECTIVITY ANALYSIS\n")
	fmt.Fprintf(writer, "=====================\n")
	fmt.Fprintf(writer, "Connection Success:   %.1f%%\n", report.Connectivity.ConnectionSuccess*100)
	fmt.Fprintf(writer, "Session Stability:    %.1f%%\n", report.Connectivity.SessionStability*100)

	if len(report.Connectivity.ConnectivityIssues) > 0 {
		fmt.Fprintf(writer, "\nConnectivity Issues:\n")
		for _, issue := range report.Connectivity.ConnectivityIssues {
			fmt.Fprintf(writer, "  %s: %s\n", issue.Type, issue.Description)
			fmt.Fprintf(writer, "    Impact: %s\n", issue.Impact)
			if len(issue.Devices) > 0 {
				fmt.Fprintf(writer, "    Affected: %s\n", strings.Join(issue.Devices, ", "))
			}
		}
	}

	if len(report.Connectivity.DeviceReliability) > 0 {
		fmt.Fprintf(writer, "\nDevice Reliability:\n")
		for _, device := range report.Connectivity.DeviceReliability {
			fmt.Fprintf(writer, "  %s: %.1f%% uptime (Score: %.2f)\n",
				device.DeviceID, device.UptimePercentage, device.ConnectionScore)
		}
	}

	fmt.Fprintf(writer, "\n")
}

func (ndr *NetworkDiagnosticsRenderer) renderIssues(report *NetworkDiagnosticReport, writer io.Writer) {
	if len(report.Issues) == 0 {
		return
	}

	fmt.Fprintf(writer, "IDENTIFIED ISSUES\n")
	fmt.Fprintf(writer, "=================\n")

	// Group issues by severity
	issuesBySeverity := make(map[IssueSeverity][]DiagnosticIssue)
	for _, issue := range report.Issues {
		issuesBySeverity[issue.Severity] = append(issuesBySeverity[issue.Severity], issue)
	}

	// Render issues in severity order
	severityOrder := []IssueSeverity{SeverityCritical, SeverityHigh, SeverityMedium, SeverityLow, SeverityInfo}

	for _, severity := range severityOrder {
		issues := issuesBySeverity[severity]
		if len(issues) == 0 {
			continue
		}

		fmt.Fprintf(writer, "\n%s Issues:\n", strings.ToUpper(string(severity)))
		for _, issue := range issues {
			fmt.Fprintf(writer, "  [%s] %s\n", issue.Type, issue.Title)
			fmt.Fprintf(writer, "    %s\n", issue.Description)
			if len(issue.AffectedDevices) > 0 {
				fmt.Fprintf(writer, "    Affected: %s\n", strings.Join(issue.AffectedDevices, ", "))
			}
			if !issue.FirstDetected.IsZero() {
				fmt.Fprintf(writer, "    First Detected: %s\n", issue.FirstDetected.Format("2006-01-02 15:04:05"))
			}
			if issue.Frequency > 1 {
				fmt.Fprintf(writer, "    Frequency: %d occurrences\n", issue.Frequency)
			}
		}
	}
	fmt.Fprintf(writer, "\n")
}

func (ndr *NetworkDiagnosticsRenderer) renderRecommendations(report *NetworkDiagnosticReport, writer io.Writer) {
	if len(report.Recommendations) == 0 {
		return
	}

	fmt.Fprintf(writer, "RECOMMENDATIONS\n")
	fmt.Fprintf(writer, "===============\n")

	// Group recommendations by priority
	recsByPriority := make(map[RecommendationPriority][]DiagnosticRecommendation)
	for _, rec := range report.Recommendations {
		recsByPriority[rec.Priority] = append(recsByPriority[rec.Priority], rec)
	}

	// Render recommendations in priority order
	priorityOrder := []RecommendationPriority{PriorityUrgent, PriorityHigh, PriorityMedium, PriorityLow}

	for _, priority := range priorityOrder {
		recommendations := recsByPriority[priority]
		if len(recommendations) == 0 {
			continue
		}

		fmt.Fprintf(writer, "\n%s Priority:\n", strings.ToUpper(string(priority)))
		for i, rec := range recommendations {
			fmt.Fprintf(writer, "  %d. [%s] %s\n", i+1, rec.Category, rec.Title)
			fmt.Fprintf(writer, "     %s\n", rec.Description)
			fmt.Fprintf(writer, "     Expected Impact: %s\n", rec.ExpectedImpact)
			fmt.Fprintf(writer, "     Confidence: %.0f%%\n", rec.Confidence*100)

			if len(rec.Actions) > 0 {
				fmt.Fprintf(writer, "     Actions:\n")
				for _, action := range rec.Actions {
					fmt.Fprintf(writer, "       - %s\n", action.Action)
					if len(action.Steps) > 0 && ndr.config.ShowDetails {
						for _, step := range action.Steps {
							fmt.Fprintf(writer, "         • %s\n", step)
						}
					}
				}
			}
		}
	}
	fmt.Fprintf(writer, "\n")
}

func (ndr *NetworkDiagnosticsRenderer) renderPredictions(report *NetworkDiagnosticReport, writer io.Writer) {
	fmt.Fprintf(writer, "PREDICTIVE INSIGHTS\n")
	fmt.Fprintf(writer, "===================\n")

	for _, prediction := range report.Predictions {
		fmt.Fprintf(writer, "[%s] %s\n", strings.ToUpper(string(prediction.Type)), prediction.Prediction)
		fmt.Fprintf(writer, "  Confidence: %.0f%%\n", prediction.Confidence*100)
		fmt.Fprintf(writer, "  Time Horizon: %v\n", prediction.TimeHorizon)
		fmt.Fprintf(writer, "  Likelihood: %.0f%%\n", prediction.Likelihood*100)

		if len(prediction.Evidence) > 0 && ndr.config.ShowDetails {
			fmt.Fprintf(writer, "  Evidence:\n")
			for _, evidence := range prediction.Evidence {
				fmt.Fprintf(writer, "    - %s (Weight: %.2f, Reliability: %.2f)\n",
					evidence.Source, evidence.Weight, evidence.Reliability)
			}
		}
		fmt.Fprintf(writer, "\n")
	}
}

func (ndr *NetworkDiagnosticsRenderer) renderReportStats(report *NetworkDiagnosticReport, writer io.Writer) {
	fmt.Fprintf(writer, "REPORT STATISTICS\n")
	fmt.Fprintf(writer, "=================\n")
	fmt.Fprintf(writer, "Generation Time:  %v\n", report.ReportStats.GenerationTime)
	fmt.Fprintf(writer, "Data Points:      %d\n", report.ReportStats.DataPoints)
	fmt.Fprintf(writer, "Analyzed Devices: %d\n", report.ReportStats.AnalyzedDevices)
	fmt.Fprintf(writer, "\n")
}

// renderJSON renders the report in JSON format
func (ndr *NetworkDiagnosticsRenderer) renderJSON(report *NetworkDiagnosticReport, writer io.Writer) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}

// renderMarkdown renders the report in Markdown format
func (ndr *NetworkDiagnosticsRenderer) renderMarkdown(report *NetworkDiagnosticReport, writer io.Writer) error {
	fmt.Fprintf(writer, "# Network Diagnostic Report\n\n")

	// Report metadata
	fmt.Fprintf(writer, "**Report ID:** %s  \n", report.ID)
	fmt.Fprintf(writer, "**Generated:** %s  \n", report.GeneratedAt.Format(time.RFC3339))
	fmt.Fprintf(writer, "**Report Type:** %s  \n", report.ReportType)
	fmt.Fprintf(writer, "**Detail Level:** %s  \n", report.DetailLevel)
	fmt.Fprintf(writer, "**Time Window:** %s to %s (%v)  \n\n",
		report.TimeWindow.StartTime.Format("2006-01-02 15:04:05"),
		report.TimeWindow.EndTime.Format("2006-01-02 15:04:05"),
		report.TimeWindow.Duration)

	// Overall health
	fmt.Fprintf(writer, "## Overall Network Health\n\n")
	fmt.Fprintf(writer, "- **Health Status:** %s\n", strings.ToTitle(string(report.OverallHealth)))
	fmt.Fprintf(writer, "- **Health Score:** %.1f/100\n\n", report.HealthScore)

	// Network overview
	fmt.Fprintf(writer, "## Network Overview\n\n")
	fmt.Fprintf(writer, "| Metric | Value |\n")
	fmt.Fprintf(writer, "|--------|-------|\n")
	fmt.Fprintf(writer, "| Total Devices | %d |\n", report.NetworkOverview.TotalDevices)
	fmt.Fprintf(writer, "| Online Devices | %d |\n", report.NetworkOverview.OnlineDevices)
	fmt.Fprintf(writer, "| Offline Devices | %d |\n", report.NetworkOverview.OfflineDevices)
	fmt.Fprintf(writer, "\n")

	// Device breakdown
	if len(report.NetworkOverview.DevicesByType) > 0 {
		fmt.Fprintf(writer, "### Devices by Type\n\n")
		fmt.Fprintf(writer, "| Type | Count |\n")
		fmt.Fprintf(writer, "|------|-------|\n")
		for deviceType, count := range report.NetworkOverview.DevicesByType {
			fmt.Fprintf(writer, "| %s | %d |\n", strings.Title(deviceType), count)
		}
		fmt.Fprintf(writer, "\n")
	}

	// Quality analysis
	fmt.Fprintf(writer, "## Connection Quality Analysis\n\n")
	fmt.Fprintf(writer, "- **Average Quality:** %.2f\n\n", report.QualityAnalysis.AverageQuality)

	if len(report.QualityAnalysis.QualityDistribution) > 0 {
		fmt.Fprintf(writer, "### Quality Distribution\n\n")
		fmt.Fprintf(writer, "| Quality Level | Count |\n")
		fmt.Fprintf(writer, "|---------------|-------|\n")
		for quality, count := range report.QualityAnalysis.QualityDistribution {
			fmt.Fprintf(writer, "| %s | %d |\n", strings.Title(quality), count)
		}
		fmt.Fprintf(writer, "\n")
	}

	// Performance analysis
	fmt.Fprintf(writer, "## Performance Analysis\n\n")
	latency := report.Performance.LatencyAnalysis
	fmt.Fprintf(writer, "### Latency\n\n")
	fmt.Fprintf(writer, "- **Average:** %.1f ms\n", latency.AverageLatency)
	fmt.Fprintf(writer, "- **95th Percentile:** %.1f ms\n", latency.P95Latency)
	fmt.Fprintf(writer, "- **99th Percentile:** %.1f ms\n\n", latency.P99Latency)

	// Issues
	if len(report.Issues) > 0 {
		fmt.Fprintf(writer, "## Identified Issues\n\n")

		for _, issue := range report.Issues {
			fmt.Fprintf(writer, "### %s - %s\n\n", strings.ToUpper(string(issue.Severity)), issue.Title)
			fmt.Fprintf(writer, "**Type:** %s  \n", issue.Type)
			fmt.Fprintf(writer, "**Description:** %s  \n", issue.Description)

			if len(issue.AffectedDevices) > 0 {
				fmt.Fprintf(writer, "**Affected Devices:** %s  \n", strings.Join(issue.AffectedDevices, ", "))
			}

			if !issue.FirstDetected.IsZero() {
				fmt.Fprintf(writer, "**First Detected:** %s  \n", issue.FirstDetected.Format("2006-01-02 15:04:05"))
			}
			fmt.Fprintf(writer, "\n")
		}
	}

	// Recommendations
	if len(report.Recommendations) > 0 {
		fmt.Fprintf(writer, "## Recommendations\n\n")

		for i, rec := range report.Recommendations {
			fmt.Fprintf(writer, "### %d. %s\n\n", i+1, rec.Title)
			fmt.Fprintf(writer, "**Category:** %s  \n", rec.Category)
			fmt.Fprintf(writer, "**Priority:** %s  \n", strings.ToUpper(string(rec.Priority)))
			fmt.Fprintf(writer, "**Description:** %s  \n", rec.Description)
			fmt.Fprintf(writer, "**Expected Impact:** %s  \n", rec.ExpectedImpact)
			fmt.Fprintf(writer, "**Confidence:** %.0f%%  \n\n", rec.Confidence*100)
		}
	}

	return nil
}

// renderHTML renders the report in HTML format
func (ndr *NetworkDiagnosticsRenderer) renderHTML(report *NetworkDiagnosticReport, writer io.Writer) error {
	fmt.Fprintf(writer, `<!DOCTYPE html>
<html>
<head>
    <title>Network Diagnostic Report</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; }
        h1, h2, h3 { color: #333; }
        .header { background: #f5f5f5; padding: 20px; border-radius: 5px; }
        .health-score { font-size: 24px; font-weight: bold; }
        .health-excellent { color: #28a745; }
        .health-good { color: #6c757d; }
        .health-fair { color: #ffc107; }
        .health-poor { color: #fd7e14; }
        .health-critical { color: #dc3545; }
        table { border-collapse: collapse; width: 100%%; margin: 20px 0; }
        th, td { border: 1px solid #ddd; padding: 8px; text-align: left; }
        th { background-color: #f2f2f2; }
        .issue-critical { background-color: #f8d7da; }
        .issue-high { background-color: #f1c0c7; }
        .issue-medium { background-color: #fff3cd; }
        .issue-low { background-color: #d1ecf1; }
        .issue-info { background-color: #e2e3e5; }
    </style>
</head>
<body>
`)

	// Header
	fmt.Fprintf(writer, `<div class="header">
        <h1>Network Diagnostic Report</h1>
        <p><strong>Report ID:</strong> %s</p>
        <p><strong>Generated:</strong> %s</p>
        <p><strong>Time Window:</strong> %s to %s (%v)</p>
    </div>
`, report.ID,
		report.GeneratedAt.Format(time.RFC3339),
		report.TimeWindow.StartTime.Format("2006-01-02 15:04:05"),
		report.TimeWindow.EndTime.Format("2006-01-02 15:04:05"),
		report.TimeWindow.Duration)

	// Overall health
	healthClass := fmt.Sprintf("health-%s", report.OverallHealth)
	fmt.Fprintf(writer, `<h2>Overall Network Health</h2>
    <p class="health-score %s">Status: %s (Score: %.1f/100)</p>
`, healthClass, strings.ToUpper(string(report.OverallHealth)), report.HealthScore)

	// Network overview table
	fmt.Fprintf(writer, `<h2>Network Overview</h2>
    <table>
        <tr><th>Metric</th><th>Value</th></tr>
        <tr><td>Total Devices</td><td>%d</td></tr>
        <tr><td>Online Devices</td><td>%d</td></tr>
        <tr><td>Offline Devices</td><td>%d</td></tr>
    </table>
`, report.NetworkOverview.TotalDevices, report.NetworkOverview.OnlineDevices, report.NetworkOverview.OfflineDevices)

	// Issues table
	if len(report.Issues) > 0 {
		fmt.Fprintf(writer, `<h2>Identified Issues</h2>
        <table>
            <tr><th>Severity</th><th>Type</th><th>Title</th><th>Description</th></tr>
`)
		for _, issue := range report.Issues {
			issueClass := fmt.Sprintf("issue-%s", issue.Severity)
			fmt.Fprintf(writer, `            <tr class="%s">
                <td>%s</td><td>%s</td><td>%s</td><td>%s</td>
            </tr>
`, issueClass, strings.ToUpper(string(issue.Severity)), issue.Type, issue.Title, issue.Description)
		}
		fmt.Fprintf(writer, "        </table>\n")
	}

	fmt.Fprintf(writer, "</body></html>")
	return nil
}

// renderCSV renders the report in CSV format (issues and recommendations)
func (ndr *NetworkDiagnosticsRenderer) renderCSV(report *NetworkDiagnosticReport, writer io.Writer) error {
	// CSV header for issues
	fmt.Fprintf(writer, "Section,Type,Severity,Title,Description,AffectedDevices,FirstDetected\n")

	// Issues
	for _, issue := range report.Issues {
		fmt.Fprintf(writer, "Issue,%s,%s,\"%s\",\"%s\",\"%s\",%s\n",
			issue.Type,
			issue.Severity,
			strings.ReplaceAll(issue.Title, "\"", "\"\""),
			strings.ReplaceAll(issue.Description, "\"", "\"\""),
			strings.Join(issue.AffectedDevices, ";"),
			issue.FirstDetected.Format("2006-01-02 15:04:05"))
	}

	// Recommendations
	for _, rec := range report.Recommendations {
		fmt.Fprintf(writer, "Recommendation,%s,%s,\"%s\",\"%s\",,\n",
			rec.Category,
			rec.Priority,
			strings.ReplaceAll(rec.Title, "\"", "\"\""),
			strings.ReplaceAll(rec.Description, "\"", "\"\""))
	}

	return nil
}

// renderXML renders the report in XML format
func (ndr *NetworkDiagnosticsRenderer) renderXML(report *NetworkDiagnosticReport, writer io.Writer) error {
	fmt.Fprintf(writer, `<?xml version="1.0" encoding="UTF-8"?>
<networkDiagnosticReport>
    <metadata>
        <id>%s</id>
        <generatedAt>%s</generatedAt>
        <reportType>%s</reportType>
        <detailLevel>%s</detailLevel>
    </metadata>
    <overallHealth>
        <status>%s</status>
        <score>%.1f</score>
    </overallHealth>
    <networkOverview>
        <totalDevices>%d</totalDevices>
        <onlineDevices>%d</onlineDevices>
        <offlineDevices>%d</offlineDevices>
    </networkOverview>
`,
		report.ID,
		report.GeneratedAt.Format(time.RFC3339),
		report.ReportType,
		report.DetailLevel,
		report.OverallHealth,
		report.HealthScore,
		report.NetworkOverview.TotalDevices,
		report.NetworkOverview.OnlineDevices,
		report.NetworkOverview.OfflineDevices)

	// Issues
	if len(report.Issues) > 0 {
		fmt.Fprintf(writer, "    <issues>\n")
		for _, issue := range report.Issues {
			fmt.Fprintf(writer, `        <issue>
            <type>%s</type>
            <severity>%s</severity>
            <title>%s</title>
            <description>%s</description>
        </issue>
`, issue.Type, issue.Severity, ndr.escapeXML(issue.Title), ndr.escapeXML(issue.Description))
		}
		fmt.Fprintf(writer, "    </issues>\n")
	}

	fmt.Fprintf(writer, "</networkDiagnosticReport>\n")
	return nil
}

// Helper methods

func (ndr *NetworkDiagnosticsRenderer) renderBar(value int, maxWidth int) string {
	if value <= 0 {
		return strings.Repeat("░", maxWidth)
	}

	// Normalize to max width
	normalized := value
	if value > maxWidth {
		normalized = maxWidth
	}

	filled := strings.Repeat("█", normalized)
	empty := strings.Repeat("░", maxWidth-normalized)

	return filled + empty
}

func (ndr *NetworkDiagnosticsRenderer) escapeXML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "'", "&apos;")
	return s
}

// RenderReportSummary renders a summary of multiple reports
func (ndr *NetworkDiagnosticsRenderer) RenderReportSummary(
	summaries []ReportSummary,
	format ReportFormat,
	writer io.Writer,
) error {
	switch format {
	case FormatText:
		return ndr.renderSummaryText(summaries, writer)
	case FormatJSON:
		encoder := json.NewEncoder(writer)
		encoder.SetIndent("", "  ")
		return encoder.Encode(summaries)
	case FormatMarkdown:
		return ndr.renderSummaryMarkdown(summaries, writer)
	default:
		return fmt.Errorf("unsupported format for summary: %s", format)
	}
}

func (ndr *NetworkDiagnosticsRenderer) renderSummaryText(summaries []ReportSummary, writer io.Writer) error {
	fmt.Fprintf(writer, "DIAGNOSTIC REPORTS SUMMARY\n")
	fmt.Fprintf(writer, "==========================\n\n")

	if len(summaries) == 0 {
		fmt.Fprintf(writer, "No reports available.\n")
		return nil
	}

	fmt.Fprintf(writer, "%-20s %-15s %-12s %-10s %-6s\n",
		"Generated", "Report Type", "Detail Level", "Health", "Issues")
	fmt.Fprintf(writer, "%s\n", strings.Repeat("-", 70))

	for _, summary := range summaries {
		fmt.Fprintf(writer, "%-20s %-15s %-12s %-10.1f %-6d\n",
			summary.GeneratedAt.Format("2006-01-02 15:04"),
			summary.ReportType,
			summary.DetailLevel,
			summary.HealthScore,
			summary.IssueCount)
	}

	return nil
}

func (ndr *NetworkDiagnosticsRenderer) renderSummaryMarkdown(summaries []ReportSummary, writer io.Writer) error {
	fmt.Fprintf(writer, "# Diagnostic Reports Summary\n\n")

	if len(summaries) == 0 {
		fmt.Fprintf(writer, "No reports available.\n")
		return nil
	}

	fmt.Fprintf(writer, "| Generated | Report Type | Detail Level | Health Score | Issues |\n")
	fmt.Fprintf(writer, "|-----------|-------------|--------------|--------------|--------|\n")

	for _, summary := range summaries {
		fmt.Fprintf(writer, "| %s | %s | %s | %.1f | %d |\n",
			summary.GeneratedAt.Format("2006-01-02 15:04"),
			summary.ReportType,
			summary.DetailLevel,
			summary.HealthScore,
			summary.IssueCount)
	}

	return nil
}
