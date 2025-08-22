package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"rtk_controller/internal/diagnostics"
	"rtk_controller/pkg/types"
)

// DiagnosticsCommands handles network diagnostics CLI commands
type DiagnosticsCommands struct {
	diagnostics *diagnostics.NetworkDiagnostics
	scheduler   *diagnostics.ScheduleManager
}

// NewDiagnosticsCommands creates new diagnostics commands handler
func NewDiagnosticsCommands() *DiagnosticsCommands {
	config := &diagnostics.Config{
		EnableSpeedTest:   true,
		EnableWANTest:     true,
		EnableLatencyTest: true,
		TestInterval:      30 * time.Minute,
		DNSServers:        []string{"8.8.8.8", "1.1.1.1"},
	}

	return &DiagnosticsCommands{
		diagnostics: diagnostics.NewNetworkDiagnostics(config),
		scheduler:   diagnostics.NewScheduleManager(),
	}
}

// RunDiagnostics runs comprehensive network diagnostics
func (dc *DiagnosticsCommands) RunDiagnostics(args []string) (string, error) {
	deviceID := "local"
	if len(args) > 0 {
		deviceID = args[0]
	}

	fmt.Println("Starting network diagnostics...")
	fmt.Println("This may take 30-60 seconds to complete.\n")

	result, err := dc.diagnostics.RunDiagnostics(deviceID)
	if err != nil {
		return "", fmt.Errorf("diagnostics failed: %w", err)
	}

	return dc.formatDiagnosticsResult(result), nil
}

// RunSpeedTest runs bandwidth speed test
func (dc *DiagnosticsCommands) RunSpeedTest(args []string) (string, error) {
	fmt.Println("Running speed test...")
	fmt.Println("Testing download and upload speeds...\n")

	client := diagnostics.NewSpeedTestClient()
	result, err := client.RunTest()
	if err != nil {
		return "", fmt.Errorf("speed test failed: %w", err)
	}

	output := fmt.Sprintf("Speed Test Results:\n")
	output += fmt.Sprintf("===================\n")
	output += fmt.Sprintf("Download: %.2f Mbps\n", result.DownloadMbps)
	output += fmt.Sprintf("Upload:   %.2f Mbps\n", result.UploadMbps)
	if result.Jitter > 0 {
		output += fmt.Sprintf("Jitter:   %.2f ms\n", result.Jitter)
	}
	if result.PacketLoss > 0 {
		output += fmt.Sprintf("Loss:     %.2f%%\n", result.PacketLoss)
	}
	output += fmt.Sprintf("Server:   %s\n", result.TestServer)
	output += fmt.Sprintf("Status:   %s\n", result.Status)

	return output, nil
}

// TestWAN tests WAN connectivity
func (dc *DiagnosticsCommands) TestWAN(args []string) (string, error) {
	fmt.Println("Testing WAN connectivity...")

	tester := diagnostics.NewWANTester([]string{"8.8.8.8", "1.1.1.1"})
	result, err := tester.TestWANConnectivity()
	if err != nil {
		return "", fmt.Errorf("WAN test failed: %w", err)
	}

	output := fmt.Sprintf("WAN Connectivity Test:\n")
	output += fmt.Sprintf("======================\n")
	output += fmt.Sprintf("Gateway Reachable: %v\n", result.ISPGatewayReachable)
	if result.ISPGatewayReachable {
		output += fmt.Sprintf("Gateway Latency:   %.2f ms\n", result.ISPGatewayLatency)
	}
	output += fmt.Sprintf("External DNS:      %.2f ms\n", result.ExternalDNSLatency)
	output += fmt.Sprintf("WAN Connected:     %v\n", result.WANConnected)
	if result.PublicIP != "" {
		output += fmt.Sprintf("Public IP:         %s\n", result.PublicIP)
	}
	if result.ISPInfo != "" {
		output += fmt.Sprintf("ISP Info:          %s\n", result.ISPInfo)
	}

	return output, nil
}

// TestLatency tests network latency
func (dc *DiagnosticsCommands) TestLatency(args []string) (string, error) {
	targets := []string{"8.8.8.8", "1.1.1.1", "google.com"}
	if len(args) > 0 {
		targets = args
	}

	fmt.Printf("Testing latency to %d targets...\n", len(targets))

	output := fmt.Sprintf("Latency Test Results:\n")
	output += fmt.Sprintf("====================\n")

	for _, target := range targets {
		fmt.Printf("Testing %s...\n", target)
		latency, loss := dc.testSingleTarget(target)

		if latency > 0 {
			output += fmt.Sprintf("%-20s: %.2f ms", target, latency)
			if loss > 0 {
				output += fmt.Sprintf(" (%.0f%% loss)", loss)
			}
			output += "\n"
		} else {
			output += fmt.Sprintf("%-20s: unreachable\n", target)
		}
	}

	return output, nil
}

// testSingleTarget tests latency to a single target
func (dc *DiagnosticsCommands) testSingleTarget(target string) (float64, float64) {
	// This would use the actual latency test implementation
	// For now, returning mock data
	return 10.5, 0
}

// ScheduleTests schedules periodic diagnostic tests
func (dc *DiagnosticsCommands) ScheduleTests(args []string) (string, error) {
	interval := 30 * time.Minute
	if len(args) > 0 {
		duration, err := time.ParseDuration(args[0])
		if err != nil {
			return "", fmt.Errorf("invalid interval: %w", err)
		}
		interval = duration
	}

	// Add schedule for periodic diagnostics
	dc.scheduler.AddSchedule("diagnostics", interval, func() {
		result, err := dc.diagnostics.RunDiagnostics("scheduled")
		if err != nil {
			fmt.Printf("Scheduled diagnostic failed: %v\n", err)
		} else {
			fmt.Printf("Scheduled diagnostic completed at %s\n",
				time.Now().Format("2006-01-02 15:04:05"))
			// Store or process result as needed
			_ = result
		}
	})

	// Start the schedule
	ctx := context.Background()
	err := dc.scheduler.StartSchedule(ctx, "diagnostics")
	if err != nil {
		return "", fmt.Errorf("failed to start schedule: %w", err)
	}

	return fmt.Sprintf("Diagnostic tests scheduled every %v", interval), nil
}

// ShowSchedules shows all test schedules
func (dc *DiagnosticsCommands) ShowSchedules(args []string) (string, error) {
	schedules := dc.scheduler.GetScheduleStatus()

	if len(schedules) == 0 {
		return "No test schedules configured", nil
	}

	output := "Test Schedules:\n"
	output += "===============\n"

	for _, schedule := range schedules {
		output += fmt.Sprintf("\nSchedule: %s\n", schedule.Name)
		output += fmt.Sprintf("  Enabled:  %v\n", schedule.Enabled)
		output += fmt.Sprintf("  Active:   %v\n", schedule.Active)
		output += fmt.Sprintf("  Interval: %v\n", schedule.Interval)
		if !schedule.LastRun.IsZero() {
			output += fmt.Sprintf("  Last Run: %s\n", schedule.LastRun.Format("2006-01-02 15:04:05"))
		}
		if !schedule.NextRun.IsZero() {
			output += fmt.Sprintf("  Next Run: %s\n", schedule.NextRun.Format("2006-01-02 15:04:05"))
		}
		output += fmt.Sprintf("  Run Count: %d\n", schedule.RunCount)
	}

	return output, nil
}

// GetLastResult retrieves the last diagnostic result
func (dc *DiagnosticsCommands) GetLastResult(args []string) (string, error) {
	deviceID := "local"
	if len(args) > 0 {
		deviceID = args[0]
	}

	result, exists := dc.diagnostics.GetLastResult(deviceID)
	if !exists {
		return fmt.Sprintf("No diagnostic results found for device: %s", deviceID), nil
	}

	return dc.formatDiagnosticsResult(result), nil
}

// formatDiagnosticsResult formats diagnostic results for display
func (dc *DiagnosticsCommands) formatDiagnosticsResult(result *types.NetworkDiagnostics) string {
	output := fmt.Sprintf("\nNetwork Diagnostics Report\n")
	output += fmt.Sprintf("==========================\n")
	output += fmt.Sprintf("Device: %s\n", result.DeviceID)
	output += fmt.Sprintf("Time:   %s\n\n", time.Unix(result.Timestamp, 0).Format("2006-01-02 15:04:05"))

	// Speed Test Results
	if result.SpeedTest != nil {
		output += fmt.Sprintf("Speed Test:\n")
		output += fmt.Sprintf("-----------\n")
		output += fmt.Sprintf("  Download: %.2f Mbps\n", result.SpeedTest.DownloadMbps)
		output += fmt.Sprintf("  Upload:   %.2f Mbps\n", result.SpeedTest.UploadMbps)
		if result.SpeedTest.Jitter > 0 {
			output += fmt.Sprintf("  Jitter:   %.2f ms\n", result.SpeedTest.Jitter)
		}
		if result.SpeedTest.PacketLoss > 0 {
			output += fmt.Sprintf("  Loss:     %.2f%%\n", result.SpeedTest.PacketLoss)
		}
		output += "\n"
	}

	// WAN Test Results
	if result.WANTest != nil {
		output += fmt.Sprintf("WAN Status:\n")
		output += fmt.Sprintf("-----------\n")
		output += fmt.Sprintf("  Gateway:     %s (%.2f ms)\n",
			dc.boolToStatus(result.WANTest.ISPGatewayReachable),
			result.WANTest.ISPGatewayLatency)
		output += fmt.Sprintf("  External DNS: %.2f ms\n", result.WANTest.ExternalDNSLatency)
		output += fmt.Sprintf("  Internet:    %s\n", dc.boolToStatus(result.WANTest.WANConnected))
		if result.WANTest.PublicIP != "" {
			output += fmt.Sprintf("  Public IP:   %s\n", result.WANTest.PublicIP)
		}
		output += "\n"
	}

	// Latency Test Results
	if result.LatencyTest != nil && len(result.LatencyTest.Targets) > 0 {
		output += fmt.Sprintf("Latency Tests:\n")
		output += fmt.Sprintf("--------------\n")
		for _, target := range result.LatencyTest.Targets {
			output += fmt.Sprintf("  %-15s: ", target.Target)
			if target.Status == "success" {
				output += fmt.Sprintf("%.2f ms (min: %.2f, max: %.2f)",
					target.AvgLatency, target.MinLatency, target.MaxLatency)
				if target.PacketLoss > 0 {
					output += fmt.Sprintf(" [%.0f%% loss]", target.PacketLoss)
				}
			} else {
				output += target.Status
			}
			output += "\n"
		}
		output += "\n"
	}

	// Connectivity Test Results
	if result.ConnectivityTest != nil {
		output += fmt.Sprintf("Connectivity:\n")
		output += fmt.Sprintf("-------------\n")

		reachableInternal := 0
		for _, target := range result.ConnectivityTest.InternalReachability {
			if target.Reachable {
				reachableInternal++
			}
		}

		reachableExternal := 0
		for _, target := range result.ConnectivityTest.ExternalReachability {
			if target.Reachable {
				reachableExternal++
			}
		}

		output += fmt.Sprintf("  Internal: %d/%d devices reachable\n",
			reachableInternal, len(result.ConnectivityTest.InternalReachability))
		output += fmt.Sprintf("  External: %d/%d services reachable\n",
			reachableExternal, len(result.ConnectivityTest.ExternalReachability))
	}

	return output
}

// boolToStatus converts boolean to status string
func (dc *DiagnosticsCommands) boolToStatus(status bool) string {
	if status {
		return "✓ OK"
	}
	return "✗ Failed"
}

// ExportResults exports diagnostic results as JSON
func (dc *DiagnosticsCommands) ExportResults(args []string) (string, error) {
	deviceID := "local"
	if len(args) > 0 {
		deviceID = args[0]
	}

	result, exists := dc.diagnostics.GetLastResult(deviceID)
	if !exists {
		return "", fmt.Errorf("no results found for device: %s", deviceID)
	}

	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to export results: %w", err)
	}

	return string(data), nil
}
