// Package diagnostics provides network diagnostic capabilities
package diagnostics

import (
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"rtk_controller/pkg/types"
)

// ================== Core Diagnostics ==================

// NetworkDiagnostics handles active network testing
type NetworkDiagnostics struct {
	mu            sync.RWMutex
	config        *Config
	lastResults   map[string]*types.NetworkDiagnostics
	testScheduler *TestScheduler
}

// Config for network diagnostics
type Config struct {
	EnableSpeedTest    bool          `json:"enable_speed_test"`
	EnableWANTest      bool          `json:"enable_wan_test"`
	EnableLatencyTest  bool          `json:"enable_latency_test"`
	TestInterval       time.Duration `json:"test_interval"`
	SpeedTestServers   []string      `json:"speed_test_servers"`
	LatencyTargets     []string      `json:"latency_targets"`
	DNSServers         []string      `json:"dns_servers"`
	MaxConcurrentTests int           `json:"max_concurrent_tests"`
}

// NewNetworkDiagnostics creates a new network diagnostics instance
func NewNetworkDiagnostics(config *Config) *NetworkDiagnostics {
	if config == nil {
		config = &Config{
			EnableSpeedTest:    true,
			EnableWANTest:      true,
			EnableLatencyTest:  true,
			TestInterval:       30 * time.Minute,
			DNSServers:         []string{"8.8.8.8", "1.1.1.1"},
			MaxConcurrentTests: 3,
		}
	}

	return &NetworkDiagnostics{
		config:        config,
		lastResults:   make(map[string]*types.NetworkDiagnostics),
		testScheduler: NewTestScheduler(config.TestInterval),
	}
}

// RunDiagnostics performs comprehensive network diagnostics
func (nd *NetworkDiagnostics) RunDiagnostics(deviceID string) (*types.NetworkDiagnostics, error) {
	result := &types.NetworkDiagnostics{
		DeviceID:  deviceID,
		Timestamp: time.Now().Unix(),
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	errors := []error{}

	// Run SpeedTest
	if nd.config.EnableSpeedTest {
		wg.Add(1)
		go func() {
			defer wg.Done()
			speedResult, err := runSpeedTest()
			mu.Lock()
			if err != nil {
				errors = append(errors, fmt.Errorf("speed test: %w", err))
			} else {
				result.SpeedTest = speedResult
			}
			mu.Unlock()
		}()
	}

	// Run WAN Test
	if nd.config.EnableWANTest {
		wg.Add(1)
		go func() {
			defer wg.Done()
			wanResult, err := runWANTest(nd.config.DNSServers)
			mu.Lock()
			if err != nil {
				errors = append(errors, fmt.Errorf("wan test: %w", err))
			} else {
				result.WANTest = wanResult
			}
			mu.Unlock()
		}()
	}

	// Run Latency Test
	if nd.config.EnableLatencyTest {
		wg.Add(1)
		go func() {
			defer wg.Done()
			latencyResult, err := nd.runLatencyTest()
			mu.Lock()
			if err != nil {
				errors = append(errors, fmt.Errorf("latency test: %w", err))
			} else {
				result.LatencyTest = latencyResult
			}
			mu.Unlock()
		}()
	}

	wg.Wait()

	// Store result
	nd.mu.Lock()
	nd.lastResults[deviceID] = result
	nd.mu.Unlock()

	if len(errors) > 0 {
		return result, fmt.Errorf("diagnostics completed with errors: %v", errors)
	}

	return result, nil
}

// GetLastResult returns the last diagnostic result for a device
func (nd *NetworkDiagnostics) GetLastResult(deviceID string) (*types.NetworkDiagnostics, bool) {
	nd.mu.RLock()
	defer nd.mu.RUnlock()
	result, exists := nd.lastResults[deviceID]
	return result, exists
}

// runLatencyTest measures latency to various targets
func (nd *NetworkDiagnostics) runLatencyTest() (*types.LatencyTestResult, error) {
	result := &types.LatencyTestResult{
		Targets:       []types.LatencyTarget{},
		OverallStatus: "success",
	}

	// Test DNS servers
	for _, dns := range nd.config.DNSServers {
		target := testLatency(dns, "dns")
		result.Targets = append(result.Targets, target)
		if target.Status != "success" && result.OverallStatus == "success" {
			result.OverallStatus = "partial"
		}
	}

	return result, nil
}

// ================== Speed Test ==================

// runSpeedTest performs bandwidth speed test
func runSpeedTest() (*types.SpeedTestResult, error) {
	result := &types.SpeedTestResult{
		Status:       "running",
		TestDuration: 10,
		TestServer:   "speedtest.tele2.net",
	}

	// Simple download test using curl
	testFile := "http://speedtest.tele2.net/1MB.zip"
	start := time.Now()

	cmd := exec.Command("curl", "-o", "/dev/null", "-s", "-w", "%{speed_download}", testFile)
	output, err := cmd.Output()
	if err != nil {
		result.Status = "failed"
		result.Error = "curl test failed"
		return result, err
	}

	elapsed := time.Since(start).Seconds()

	// Parse download speed (bytes/sec to Mbps)
	var downloadSpeed float64
	fmt.Sscanf(string(output), "%f", &downloadSpeed)
	result.DownloadMbps = (downloadSpeed * 8) / 1e6
	result.Status = "completed"
	result.TestDuration = int(elapsed)

	return result, nil
}

// ================== WAN Test ==================

// runWANTest checks WAN connectivity
func runWANTest(dnsServers []string) (*types.WANTestResult, error) {
	result := &types.WANTestResult{}

	// Check WAN connectivity
	result.WANConnected = checkInternetConnectivity()

	// Test External DNS
	for _, dns := range dnsServers {
		latency, reachable := testReachability(dns)
		if reachable {
			result.ExternalDNSLatency = latency
			break
		}
	}

	// Get Public IP
	if result.WANConnected {
		result.PublicIP = getPublicIP()
	}

	return result, nil
}

// checkInternetConnectivity verifies internet connectivity
func checkInternetConnectivity() bool {
	// Try DNS resolution
	_, err := net.LookupHost("google.com")
	return err == nil
}

// testReachability tests if a target is reachable and measures latency
func testReachability(target string) (float64, bool) {
	// Use ping to test reachability
	cmd := exec.Command("ping", "-c", "3", "-W", "2", target)
	output, err := cmd.Output()
	if err != nil {
		return 0, false
	}

	// Parse average latency
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "min/avg/max") || strings.Contains(line, "round-trip") {
			// Extract average RTT
			re := regexp.MustCompile(`(\d+\.?\d*)/(\d+\.?\d*)/(\d+\.?\d*)`)
			if matches := re.FindStringSubmatch(line); len(matches) > 2 {
				avg, _ := strconv.ParseFloat(matches[2], 64)
				return avg, true
			}
		}
	}

	return 0, true
}

// getPublicIP retrieves the public IP address
func getPublicIP() string {
	// Try multiple services
	services := []string{
		"https://api.ipify.org?format=text",
		"https://icanhazip.com",
	}

	client := &http.Client{Timeout: 5 * time.Second}

	for _, service := range services {
		resp, err := client.Get(service)
		if err == nil {
			defer resp.Body.Close()
			body, err := ioutil.ReadAll(resp.Body)
			if err == nil {
				ip := strings.TrimSpace(string(body))
				// Validate IP
				if net.ParseIP(ip) != nil {
					return ip
				}
			}
		}
	}

	return ""
}

// testLatency tests latency to a single target
func testLatency(target, targetType string) types.LatencyTarget {
	result := types.LatencyTarget{
		Target:      target,
		Type:        targetType,
		PacketsSent: 4,
		Status:      "success",
	}

	// Use ping command
	cmd := exec.Command("ping", "-c", "4", "-W", "2", target)
	output, err := cmd.CombinedOutput()

	if err != nil {
		result.Status = "failed"
		return result
	}

	// Parse ping output
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		// Parse packet loss
		if strings.Contains(line, "packet loss") {
			re := regexp.MustCompile(`(\d+)\.?\d*% packet loss`)
			if matches := re.FindStringSubmatch(line); len(matches) > 1 {
				loss, _ := strconv.ParseFloat(matches[1], 64)
				result.PacketLoss = loss
				result.PacketsReceived = int(float64(result.PacketsSent) * (100 - loss) / 100)
			}
		}

		// Parse RTT statistics
		if strings.Contains(line, "min/avg/max") || strings.Contains(line, "round-trip") {
			re := regexp.MustCompile(`(\d+\.?\d*)/(\d+\.?\d*)/(\d+\.?\d*)`)
			if matches := re.FindStringSubmatch(line); len(matches) > 3 {
				result.MinLatency, _ = strconv.ParseFloat(matches[1], 64)
				result.AvgLatency, _ = strconv.ParseFloat(matches[2], 64)
				result.MaxLatency, _ = strconv.ParseFloat(matches[3], 64)
			}
		}
	}

	if result.PacketLoss == 100 {
		result.Status = "timeout"
	}

	return result
}

// MeasurePacketLoss measures packet loss to a target
func MeasurePacketLoss(target string, count int) (float64, error) {
	cmd := exec.Command("ping", "-c", fmt.Sprintf("%d", count), "-W", "2", target)
	output, err := cmd.Output()

	// Even if ping returns error, we might have partial output
	if output == nil {
		return 100, err
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "packet loss") {
			re := regexp.MustCompile(`(\d+)\.?\d*% packet loss`)
			if matches := re.FindStringSubmatch(line); len(matches) > 1 {
				loss, _ := strconv.ParseFloat(matches[1], 64)
				return loss, nil
			}
		}
	}

	return 0, nil
}

// MeasureJitter measures network jitter
func MeasureJitter(target string, count int) (float64, error) {
	cmd := exec.Command("ping", "-c", fmt.Sprintf("%d", count), target)
	output, err := cmd.Output()
	if err != nil {
		return 0, err
	}

	// Parse RTT values and calculate jitter
	lines := strings.Split(string(output), "\n")
	var rtts []float64

	for _, line := range lines {
		if strings.Contains(line, "time=") {
			re := regexp.MustCompile(`time=(\d+\.?\d*) ms`)
			if matches := re.FindStringSubmatch(line); len(matches) > 1 {
				rtt, _ := strconv.ParseFloat(matches[1], 64)
				rtts = append(rtts, rtt)
			}
		}
	}

	if len(rtts) < 2 {
		return 0, fmt.Errorf("insufficient samples")
	}

	// Calculate jitter (variation in RTT)
	var sum float64
	for i := 1; i < len(rtts); i++ {
		diff := rtts[i] - rtts[i-1]
		if diff < 0 {
			diff = -diff
		}
		sum += diff
	}

	return sum / float64(len(rtts)-1), nil
}

// ================== Test Scheduler ==================

// TestScheduler handles periodic test scheduling
type TestScheduler struct {
	interval time.Duration
	active   bool
	mu       sync.RWMutex
	stopCh   chan struct{}
}

// NewTestScheduler creates a new test scheduler
func NewTestScheduler(interval time.Duration) *TestScheduler {
	if interval < time.Minute {
		interval = 30 * time.Minute
	}

	return &TestScheduler{
		interval: interval,
		active:   false,
	}
}

// Start begins scheduled test execution
func (s *TestScheduler) Start(ctx context.Context, testFunc func()) {
	s.mu.Lock()
	if s.active {
		s.mu.Unlock()
		return
	}

	s.active = true
	s.stopCh = make(chan struct{})
	s.mu.Unlock()

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	// Run first test immediately
	go testFunc()

	for {
		select {
		case <-ctx.Done():
			s.Stop()
			return
		case <-s.stopCh:
			return
		case <-ticker.C:
			go testFunc()
		}
	}
}

// Stop halts scheduled tests
func (s *TestScheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.active && s.stopCh != nil {
		close(s.stopCh)
		s.active = false
	}
}

// IsActive returns whether scheduler is running
func (s *TestScheduler) IsActive() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.active
}

// ================== Schedule Manager ==================

// ScheduleManager manages multiple test schedules
type ScheduleManager struct {
	schedules map[string]*TestSchedule
	mu        sync.RWMutex
}

// TestSchedule represents a scheduled test configuration
type TestSchedule struct {
	Name      string
	Interval  time.Duration
	Enabled   bool
	LastRun   time.Time
	NextRun   time.Time
	RunCount  int
	TestFunc  func()
	scheduler *TestScheduler
}

// NewScheduleManager creates a new schedule manager
func NewScheduleManager() *ScheduleManager {
	return &ScheduleManager{
		schedules: make(map[string]*TestSchedule),
	}
}

// AddSchedule adds a new test schedule
func (m *ScheduleManager) AddSchedule(name string, interval time.Duration, testFunc func()) {
	m.mu.Lock()
	defer m.mu.Unlock()

	schedule := &TestSchedule{
		Name:      name,
		Interval:  interval,
		Enabled:   true,
		TestFunc:  testFunc,
		scheduler: NewTestScheduler(interval),
		NextRun:   time.Now().Add(interval),
	}

	m.schedules[name] = schedule
}

// StartSchedule starts a specific test schedule
func (m *ScheduleManager) StartSchedule(ctx context.Context, name string) error {
	m.mu.RLock()
	schedule, exists := m.schedules[name]
	m.mu.RUnlock()

	if !exists {
		return fmt.Errorf("schedule %s not found", name)
	}

	if !schedule.Enabled {
		return fmt.Errorf("schedule %s is disabled", name)
	}

	go schedule.scheduler.Start(ctx, func() {
		m.mu.Lock()
		schedule.LastRun = time.Now()
		schedule.NextRun = time.Now().Add(schedule.Interval)
		schedule.RunCount++
		m.mu.Unlock()

		schedule.TestFunc()
	})

	return nil
}

// GetScheduleStatus returns status of all schedules
func (m *ScheduleManager) GetScheduleStatus() []ScheduleStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var status []ScheduleStatus
	for _, schedule := range m.schedules {
		status = append(status, ScheduleStatus{
			Name:     schedule.Name,
			Enabled:  schedule.Enabled,
			Active:   schedule.scheduler.IsActive(),
			Interval: schedule.Interval,
			LastRun:  schedule.LastRun,
			NextRun:  schedule.NextRun,
			RunCount: schedule.RunCount,
		})
	}

	return status
}

// ScheduleStatus represents the status of a test schedule
type ScheduleStatus struct {
	Name     string        `json:"name"`
	Enabled  bool          `json:"enabled"`
	Active   bool          `json:"active"`
	Interval time.Duration `json:"interval"`
	LastRun  time.Time     `json:"last_run"`
	NextRun  time.Time     `json:"next_run"`
	RunCount int           `json:"run_count"`
}

// ================== WAN Tester (for external use) ==================

// WANTester handles WAN connectivity testing
type WANTester struct {
	dnsServers []string
}

// NewWANTester creates a new WAN tester
func NewWANTester(dnsServers []string) *WANTester {
	if len(dnsServers) == 0 {
		dnsServers = []string{"8.8.8.8", "1.1.1.1"}
	}
	return &WANTester{
		dnsServers: dnsServers,
	}
}

// TestWANConnectivity performs comprehensive WAN connectivity test
func (w *WANTester) TestWANConnectivity() (*types.WANTestResult, error) {
	return runWANTest(w.dnsServers)
}

// ================== Speed Test Client (for external use) ==================

// SpeedTestClient handles bandwidth speed testing
type SpeedTestClient struct{}

// NewSpeedTestClient creates a new speed test client
func NewSpeedTestClient() *SpeedTestClient {
	return &SpeedTestClient{}
}

// RunTest performs a speed test
func (c *SpeedTestClient) RunTest() (*types.SpeedTestResult, error) {
	return runSpeedTest()
}