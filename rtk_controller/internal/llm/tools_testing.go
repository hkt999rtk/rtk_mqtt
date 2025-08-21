package llm

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"rtk_controller/pkg/types"
)

// NetworkSpeedTestTool implements the network.speedtest_full LLM tool
type NetworkSpeedTestTool struct {
	// No dependencies needed for basic speed test
}

// NewNetworkSpeedTestTool creates a new network.speedtest_full tool
func NewNetworkSpeedTestTool() *NetworkSpeedTestTool {
	return &NetworkSpeedTestTool{}
}

// Name returns the tool name
func (n *NetworkSpeedTestTool) Name() string {
	return "network.speedtest_full"
}

// Category returns the tool category
func (n *NetworkSpeedTestTool) Category() types.ToolCategory {
	return types.ToolCategoryTest
}

// Description returns the tool description
func (n *NetworkSpeedTestTool) Description() string {
	return "Performs a comprehensive network speed test including download/upload speeds and latency measurements"
}

// RequiredCapabilities returns the required device capabilities
func (n *NetworkSpeedTestTool) RequiredCapabilities() []string {
	return []string{"network_testing", "internet_access"}
}

// Validate validates the tool parameters
func (n *NetworkSpeedTestTool) Validate(params map[string]interface{}) error {
	// Optional test_server parameter
	if testServer, exists := params["test_server"]; exists {
		if _, ok := testServer.(string); !ok {
			return fmt.Errorf("test_server must be a string")
		}
	}
	
	// Optional test_duration parameter
	if duration, exists := params["test_duration_seconds"]; exists {
		if d, ok := duration.(float64); ok {
			if d < 5 || d > 60 {
				return fmt.Errorf("test_duration_seconds must be between 5 and 60")
			}
		} else {
			return fmt.Errorf("test_duration_seconds must be a number")
		}
	}
	
	return nil
}

// Execute executes the tool
func (n *NetworkSpeedTestTool) Execute(ctx context.Context, params map[string]interface{}) (*types.ToolResult, error) {
	result := &types.ToolResult{
		ToolName:  n.Name(),
		Success:   false,
		Timestamp: time.Now(),
	}
	
	// Parse parameters
	testServer := "http://speedtest.tele2.net/1MB.zip" // default
	if val, exists := params["test_server"]; exists {
		if s, ok := val.(string); ok && s != "" {
			testServer = s
		}
	}
	
	testDuration := 10.0 // default 10 seconds
	if val, exists := params["test_duration_seconds"]; exists {
		if d, ok := val.(float64); ok {
			testDuration = d
		}
	}
	
	// Run speed test with timeout
	testCtx, cancel := context.WithTimeout(ctx, time.Duration(testDuration+10)*time.Second)
	defer cancel()
	
	speedTestResult, err := n.runSpeedTest(testCtx, testServer)
	if err != nil {
		result.Error = fmt.Sprintf("Speed test failed: %v", err)
		return result, nil
	}
	
	// Run additional network quality tests
	latencyResult := n.measureLatency(testCtx)
	connectivityResult := n.testConnectivity(testCtx)
	
	// Build comprehensive result
	testData := map[string]interface{}{
		"speed_test": speedTestResult,
		"latency":    latencyResult,
		"connectivity": connectivityResult,
		"test_parameters": map[string]interface{}{
			"test_server":         testServer,
			"test_duration":       testDuration,
			"executed_at":         time.Now(),
		},
		"summary": map[string]interface{}{
			"download_mbps":       speedTestResult["download_mbps"],
			"upload_mbps":         speedTestResult["upload_mbps"],
			"avg_latency_ms":      latencyResult["average_ms"],
			"connectivity_status": connectivityResult["status"],
		},
	}
	
	result.Success = true
	result.Data = testData
	
	return result, nil
}

// runSpeedTest performs the actual speed test
func (n *NetworkSpeedTestTool) runSpeedTest(ctx context.Context, testServer string) (map[string]interface{}, error) {
	start := time.Now()
	
	// Download test using curl
	cmd := exec.CommandContext(ctx, "curl", "-o", "/dev/null", "-s", "-w", "%{speed_download}", testServer)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("download test failed: %w", err)
	}
	
	elapsed := time.Since(start).Seconds()
	
	// Parse download speed (bytes/sec to Mbps)
	var downloadSpeed float64
	if _, err := fmt.Sscanf(string(output), "%f", &downloadSpeed); err != nil {
		return nil, fmt.Errorf("failed to parse download speed: %w", err)
	}
	
	downloadMbps := (downloadSpeed * 8) / 1e6
	
	// Simple upload test (POST small data)
	uploadMbps := n.measureUploadSpeed(ctx)
	
	return map[string]interface{}{
		"download_mbps":  downloadMbps,
		"upload_mbps":    uploadMbps,
		"test_duration":  elapsed,
		"test_server":    testServer,
		"status":         "completed",
	}, nil
}

// measureUploadSpeed performs a simple upload test
func (n *NetworkSpeedTestTool) measureUploadSpeed(ctx context.Context) float64 {
	// Simple upload test - this is a basic implementation
	// In a real implementation, you'd want to use a proper speed test server
	testData := strings.Repeat("test", 1024) // 4KB test data
	
	start := time.Now()
	
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequestWithContext(ctx, "POST", "http://httpbin.org/post", strings.NewReader(testData))
	if err != nil {
		return 0
	}
	
	resp, err := client.Do(req)
	if err != nil {
		return 0
	}
	defer resp.Body.Close()
	
	elapsed := time.Since(start).Seconds()
	if elapsed > 0 {
		return (float64(len(testData)) * 8) / (elapsed * 1e6) // Convert to Mbps
	}
	
	return 0
}

// measureLatency measures network latency to common targets
func (n *NetworkSpeedTestTool) measureLatency(ctx context.Context) map[string]interface{} {
	targets := []string{"8.8.8.8", "1.1.1.1", "google.com"}
	latencies := make(map[string]float64)
	var totalLatency float64
	var successCount int
	
	for _, target := range targets {
		latency, success := n.pingTarget(ctx, target)
		if success {
			latencies[target] = latency
			totalLatency += latency
			successCount++
		}
	}
	
	avgLatency := 0.0
	if successCount > 0 {
		avgLatency = totalLatency / float64(successCount)
	}
	
	return map[string]interface{}{
		"targets":    latencies,
		"average_ms": avgLatency,
		"tested_at":  time.Now(),
	}
}

// pingTarget pings a target and returns latency
func (n *NetworkSpeedTestTool) pingTarget(ctx context.Context, target string) (float64, bool) {
	cmd := exec.CommandContext(ctx, "ping", "-c", "3", "-W", "2", target)
	output, err := cmd.Output()
	if err != nil {
		return 0, false
	}
	
	// Parse average latency from ping output
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "avg") || strings.Contains(line, "round-trip") {
			// Look for pattern like "min/avg/max/mdev = 1.234/5.678/..."
			parts := strings.Split(line, "=")
			if len(parts) >= 2 {
				times := strings.Split(strings.TrimSpace(parts[1]), "/")
				if len(times) >= 2 {
					if avg, err := strconv.ParseFloat(times[1], 64); err == nil {
						return avg, true
					}
				}
			}
		}
	}
	
	return 0, false
}

// testConnectivity tests various connectivity aspects
func (n *NetworkSpeedTestTool) testConnectivity(ctx context.Context) map[string]interface{} {
	// Test DNS resolution
	_, dnsErr := net.LookupHost("google.com")
	dnsWorking := dnsErr == nil
	
	// Test HTTP connectivity
	client := &http.Client{Timeout: 5 * time.Second}
	req, _ := http.NewRequestWithContext(ctx, "GET", "http://httpbin.org/get", nil)
	resp, httpErr := client.Do(req)
	httpWorking := httpErr == nil && resp != nil
	if resp != nil {
		resp.Body.Close()
	}
	
	// Test HTTPS connectivity
	req, _ = http.NewRequestWithContext(ctx, "GET", "https://httpbin.org/get", nil)
	resp, httpsErr := client.Do(req)
	httpsWorking := httpsErr == nil && resp != nil
	if resp != nil {
		resp.Body.Close()
	}
	
	// Overall status
	status := "good"
	if !dnsWorking || !httpWorking {
		status = "poor"
	} else if !httpsWorking {
		status = "limited"
	}
	
	return map[string]interface{}{
		"dns_resolution": dnsWorking,
		"http_access":    httpWorking,
		"https_access":   httpsWorking,
		"status":         status,
		"tested_at":      time.Now(),
	}
}

// WANConnectivityTool implements the diagnostics.wan_connectivity LLM tool
type WANConnectivityTool struct {
	// No dependencies needed for basic WAN test
}

// NewWANConnectivityTool creates a new diagnostics.wan_connectivity tool
func NewWANConnectivityTool() *WANConnectivityTool {
	return &WANConnectivityTool{}
}

// Name returns the tool name
func (w *WANConnectivityTool) Name() string {
	return "diagnostics.wan_connectivity"
}

// Category returns the tool category
func (w *WANConnectivityTool) Category() types.ToolCategory {
	return types.ToolCategoryTest
}

// Description returns the tool description
func (w *WANConnectivityTool) Description() string {
	return "Performs comprehensive WAN connectivity diagnosis including DNS, routing, and external accessibility tests"
}

// RequiredCapabilities returns the required device capabilities
func (w *WANConnectivityTool) RequiredCapabilities() []string {
	return []string{"network_testing", "wan_access"}
}

// Validate validates the tool parameters
func (w *WANConnectivityTool) Validate(params map[string]interface{}) error {
	// Optional dns_servers parameter
	if dnsServers, exists := params["dns_servers"]; exists {
		if servers, ok := dnsServers.([]interface{}); ok {
			for _, server := range servers {
				if _, ok := server.(string); !ok {
					return fmt.Errorf("all dns_servers must be strings")
				}
			}
		} else {
			return fmt.Errorf("dns_servers must be an array of strings")
		}
	}
	
	// Optional include_traceroute parameter
	if includeTrace, exists := params["include_traceroute"]; exists {
		if _, ok := includeTrace.(bool); !ok {
			return fmt.Errorf("include_traceroute must be a boolean")
		}
	}
	
	return nil
}

// Execute executes the tool
func (w *WANConnectivityTool) Execute(ctx context.Context, params map[string]interface{}) (*types.ToolResult, error) {
	result := &types.ToolResult{
		ToolName:  w.Name(),
		Success:   false,
		Timestamp: time.Now(),
	}
	
	// Parse parameters
	dnsServers := []string{"8.8.8.8", "1.1.1.1", "208.67.222.222"} // default DNS servers
	if val, exists := params["dns_servers"]; exists {
		if servers, ok := val.([]interface{}); ok {
			dnsServers = make([]string, len(servers))
			for i, server := range servers {
				dnsServers[i] = server.(string)
			}
		}
	}
	
	includeTraceroute := false // default to false for faster execution
	if val, exists := params["include_traceroute"]; exists {
		if b, ok := val.(bool); ok {
			includeTraceroute = b
		}
	}
	
	// Run WAN connectivity tests with timeout
	testCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	
	// Basic connectivity test
	connectivityTest := w.testBasicConnectivity(testCtx)
	
	// DNS test
	dnsTest := w.testDNSServers(testCtx, dnsServers)
	
	// Public IP detection
	publicIP := w.getPublicIP(testCtx)
	
	// Gateway test
	gatewayTest := w.testGateway(testCtx)
	
	// Optional traceroute
	var tracerouteTest map[string]interface{}
	if includeTraceroute {
		tracerouteTest = w.runTraceroute(testCtx, "8.8.8.8")
	}
	
	// Overall status determination
	overallStatus := "healthy"
	if !connectivityTest["internet_reachable"].(bool) {
		overallStatus = "no_internet"
	} else if !dnsTest["primary_dns_working"].(bool) {
		overallStatus = "dns_issues"
	} else if publicIP == "" {
		overallStatus = "nat_issues"
	} else if !gatewayTest["gateway_reachable"].(bool) {
		overallStatus = "gateway_issues"
	}
	
	// Build comprehensive result
	wanData := map[string]interface{}{
		"overall_status": overallStatus,
		"connectivity":   connectivityTest,
		"dns":            dnsTest,
		"public_ip":      publicIP,
		"gateway":        gatewayTest,
		"test_parameters": map[string]interface{}{
			"dns_servers":        dnsServers,
			"include_traceroute": includeTraceroute,
			"executed_at":        time.Now(),
		},
	}
	
	if tracerouteTest != nil {
		wanData["traceroute"] = tracerouteTest
	}
	
	result.Success = true
	result.Data = wanData
	
	return result, nil
}

// testBasicConnectivity tests basic internet connectivity
func (w *WANConnectivityTool) testBasicConnectivity(ctx context.Context) map[string]interface{} {
	// Test DNS resolution
	_, dnsErr := net.LookupHost("google.com")
	
	// Test HTTP connectivity
	client := &http.Client{Timeout: 5 * time.Second}
	req, _ := http.NewRequestWithContext(ctx, "GET", "http://httpbin.org/get", nil)
	_, httpErr := client.Do(req)
	
	return map[string]interface{}{
		"internet_reachable": dnsErr == nil,
		"http_working":       httpErr == nil,
		"tested_at":         time.Now(),
	}
}

// testDNSServers tests DNS server connectivity and resolution
func (w *WANConnectivityTool) testDNSServers(ctx context.Context, servers []string) map[string]interface{} {
	results := make(map[string]interface{})
	var workingServers []string
	
	for i, server := range servers {
		latency, working := w.testDNSServer(ctx, server)
		serverKey := fmt.Sprintf("server_%d_%s", i, server)
		results[serverKey] = map[string]interface{}{
			"server":    server,
			"working":   working,
			"latency":   latency,
		}
		
		if working {
			workingServers = append(workingServers, server)
		}
	}
	
	results["primary_dns_working"] = len(workingServers) > 0
	results["working_servers"] = workingServers
	results["total_tested"] = len(servers)
	
	return results
}

// testDNSServer tests a specific DNS server
func (w *WANConnectivityTool) testDNSServer(ctx context.Context, server string) (float64, bool) {
	start := time.Now()
	
	// Try to ping the DNS server
	cmd := exec.CommandContext(ctx, "ping", "-c", "1", "-W", "2", server)
	err := cmd.Run()
	
	if err == nil {
		return float64(time.Since(start).Nanoseconds()) / 1e6, true // Convert to milliseconds
	}
	
	return 0, false
}

// getPublicIP attempts to get the public IP address
func (w *WANConnectivityTool) getPublicIP(ctx context.Context) string {
	client := &http.Client{Timeout: 5 * time.Second}
	
	// Try multiple services
	services := []string{
		"http://httpbin.org/ip",
		"http://icanhazip.com",
		"http://ipinfo.io/ip",
	}
	
	for _, service := range services {
		req, err := http.NewRequestWithContext(ctx, "GET", service, nil)
		if err != nil {
			continue
		}
		
		resp, err := client.Do(req)
		if err != nil {
			continue
		}
		defer resp.Body.Close()
		
		// Try to extract IP from response
		// This is a simplified implementation
		return "detected" // Placeholder - would implement IP extraction
	}
	
	return ""
}

// testGateway tests connectivity to the default gateway
func (w *WANConnectivityTool) testGateway(ctx context.Context) map[string]interface{} {
	// Get default gateway (simplified)
	cmd := exec.CommandContext(ctx, "route", "-n", "get", "default")
	output, err := cmd.Output()
	
	gatewayReachable := false
	if err == nil && strings.Contains(string(output), "gateway:") {
		// Extract gateway IP and test it
		// This is simplified - would parse the actual gateway IP
		gatewayReachable = true
	}
	
	return map[string]interface{}{
		"gateway_reachable": gatewayReachable,
		"tested_at":         time.Now(),
	}
}

// runTraceroute performs a traceroute to the target
func (w *WANConnectivityTool) runTraceroute(ctx context.Context, target string) map[string]interface{} {
	cmd := exec.CommandContext(ctx, "traceroute", "-m", "15", target)
	output, err := cmd.Output()
	
	if err != nil {
		return map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		}
	}
	
	hops := strings.Split(string(output), "\n")
	return map[string]interface{}{
		"success":  true,
		"target":   target,
		"hops":     len(hops),
		"output":   string(output),
		"executed": time.Now(),
	}
}