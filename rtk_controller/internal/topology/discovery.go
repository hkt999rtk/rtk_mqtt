package topology

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"rtk_controller/internal/mqtt"
	"rtk_controller/internal/storage"
	"rtk_controller/pkg/types"
)

// DeviceDiscovery handles network device discovery and topology building
type DeviceDiscovery struct {
	storage           *storage.TopologyStorage
	topologyProcessor *mqtt.TopologyProcessor

	// Discovery configuration
	config DiscoveryConfig

	// Device registry
	devices map[string]*types.NetworkDevice
	mu      sync.RWMutex

	// Discovery state
	running bool
	cancel  context.CancelFunc

	// Statistics
	stats DiscoveryStats
}

// DiscoveryConfig holds device discovery configuration
type DiscoveryConfig struct {
	// Discovery intervals
	DeviceDiscoveryInterval time.Duration
	ConnectionScanInterval  time.Duration
	DeviceTimeout           time.Duration

	// Discovery methods
	EnableMQTTDiscovery     bool
	EnableNetworkScanning   bool
	EnableARPTableScanning  bool
	EnableDHCPLeaseScanning bool

	// Network scanning configuration
	NetworkRanges []string
	ScanPorts     []int
	ScanTimeout   time.Duration

	// Device classification
	EnableDeviceClassification bool
	ClassificationRules        []ClassificationRule
}

// ClassificationRule defines how to classify discovered devices
type ClassificationRule struct {
	ID         string
	Name       string
	Conditions []Condition
	DeviceType string
	Role       types.DeviceRole
	Confidence float64
}

// Condition represents a classification condition
type Condition struct {
	Field    string      // e.g., "manufacturer", "hostname", "open_ports"
	Operator string      // e.g., "equals", "contains", "matches", "in"
	Value    interface{} // The value to match against
}

// DiscoveryStats holds discovery statistics
type DiscoveryStats struct {
	TotalDevicesDiscovered int64
	DevicesOnline          int64
	DevicesOffline         int64
	ConnectionsDiscovered  int64
	LastDiscoveryRun       time.Time
	DiscoveryRunDuration   time.Duration
	DiscoveryErrors        int64
}

// NewDeviceDiscovery creates a new device discovery manager
func NewDeviceDiscovery(storage *storage.TopologyStorage, processor *mqtt.TopologyProcessor, config DiscoveryConfig) *DeviceDiscovery {
	return &DeviceDiscovery{
		storage:           storage,
		topologyProcessor: processor,
		config:            config,
		devices:           make(map[string]*types.NetworkDevice),
		stats:             DiscoveryStats{},
	}
}

// Start begins the device discovery process
func (dd *DeviceDiscovery) Start() error {
	dd.mu.Lock()
	defer dd.mu.Unlock()

	if dd.running {
		return fmt.Errorf("device discovery is already running")
	}

	ctx, cancel := context.WithCancel(context.Background())
	dd.cancel = cancel
	dd.running = true

	log.Printf("Starting device discovery with interval %v", dd.config.DeviceDiscoveryInterval)

	// Start discovery goroutines
	go dd.discoveryLoop(ctx)
	go dd.connectionScanLoop(ctx)

	// Load existing devices from storage
	if err := dd.loadExistingDevices(); err != nil {
		log.Printf("Failed to load existing devices: %v", err)
	}

	return nil
}

// Stop stops the device discovery process
func (dd *DeviceDiscovery) Stop() error {
	dd.mu.Lock()
	defer dd.mu.Unlock()

	if !dd.running {
		return fmt.Errorf("device discovery is not running")
	}

	dd.cancel()
	dd.running = false

	log.Printf("Device discovery stopped")
	return nil
}

// GetDiscoveredDevices returns all discovered devices
func (dd *DeviceDiscovery) GetDiscoveredDevices() map[string]*types.NetworkDevice {
	dd.mu.RLock()
	defer dd.mu.RUnlock()

	// Return copy of devices map
	devices := make(map[string]*types.NetworkDevice)
	for id, device := range dd.devices {
		devices[id] = device
	}

	return devices
}

// GetDevice returns a specific discovered device
func (dd *DeviceDiscovery) GetDevice(deviceID string) (*types.NetworkDevice, bool) {
	dd.mu.RLock()
	defer dd.mu.RUnlock()

	device, exists := dd.devices[deviceID]
	return device, exists
}

// AddDevice adds a manually discovered device
func (dd *DeviceDiscovery) AddDevice(device *types.NetworkDevice) error {
	dd.mu.Lock()
	defer dd.mu.Unlock()

	// Validate device
	if device.DeviceID == "" {
		return fmt.Errorf("device ID cannot be empty")
	}

	// Set discovery timestamps
	now := time.Now().UnixMilli()
	// Set LastSeen timestamp
	device.LastSeen = now
	device.LastSeen = now
	device.Online = true

	// Store device
	dd.devices[device.DeviceID] = device

	// Save to storage
	if err := dd.storage.SaveNetworkDevice(device); err != nil {
		log.Printf("Failed to save manually added device: %v", err)
		return err
	}

	dd.stats.TotalDevicesDiscovered++
	dd.stats.DevicesOnline++

	log.Printf("Manually added device: %s (%s)", device.DeviceID, device.DeviceType)
	return nil
}

// GetStats returns discovery statistics
func (dd *DeviceDiscovery) GetStats() DiscoveryStats {
	dd.mu.RLock()
	defer dd.mu.RUnlock()

	return dd.stats
}

// ProcessMQTTDiscoveryMessage processes MQTT-based device discovery
func (dd *DeviceDiscovery) ProcessMQTTDiscoveryMessage(deviceID string, device *types.NetworkDevice) error {
	dd.mu.Lock()
	defer dd.mu.Unlock()

	// Check if this is a new device
	existing, exists := dd.devices[deviceID]

	if !exists {
		// New device discovered
		// New device - just set LastSeen
		device.LastSeen = time.Now().UnixMilli()
		dd.stats.TotalDevicesDiscovered++
		dd.stats.DevicesOnline++

		log.Printf("New device discovered via MQTT: %s (%s)", deviceID, device.DeviceType)
	} else {
		// Update existing device
		// Keep existing timestamps

		// Update online status
		if !existing.Online && device.Online {
			dd.stats.DevicesOnline++
			dd.stats.DevicesOffline--
		}
	}

	// Store updated device
	dd.devices[deviceID] = device

	// Apply device classification if enabled
	if dd.config.EnableDeviceClassification {
		dd.classifyDevice(device)
	}

	return nil
}

// Private methods

func (dd *DeviceDiscovery) discoveryLoop(ctx context.Context) {
	ticker := time.NewTicker(dd.config.DeviceDiscoveryInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			dd.runDiscovery()
		}
	}
}

func (dd *DeviceDiscovery) connectionScanLoop(ctx context.Context) {
	ticker := time.NewTicker(dd.config.ConnectionScanInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			dd.scanConnections()
		}
	}
}

func (dd *DeviceDiscovery) runDiscovery() {
	startTime := time.Now()
	dd.stats.LastDiscoveryRun = startTime

	log.Printf("Running device discovery scan...")

	// Run enabled discovery methods
	if dd.config.EnableNetworkScanning {
		dd.runNetworkScanning()
	}

	if dd.config.EnableARPTableScanning {
		dd.runARPTableScanning()
	}

	if dd.config.EnableDHCPLeaseScanning {
		dd.runDHCPLeaseScanning()
	}

	// Update device timeouts
	dd.updateDeviceTimeouts()

	dd.stats.DiscoveryRunDuration = time.Since(startTime)
	log.Printf("Discovery scan completed in %v", dd.stats.DiscoveryRunDuration)
}

func (dd *DeviceDiscovery) runNetworkScanning() {
	// Implement network range scanning
	// This would typically involve:
	// 1. Ping sweep of configured network ranges
	// 2. Port scanning of responsive hosts
	// 3. Service detection and fingerprinting

	log.Printf("Running network scanning for ranges: %v", dd.config.NetworkRanges)

	// For now, this is a placeholder
	// Real implementation would use network scanning libraries
}

func (dd *DeviceDiscovery) runARPTableScanning() {
	// Implement ARP table scanning
	// This would read the system ARP table to discover local devices

	log.Printf("Scanning ARP table for device discovery")

	// Placeholder implementation
	// Real implementation would parse /proc/net/arp on Linux
	// or use 'arp -a' command on other systems
}

func (dd *DeviceDiscovery) runDHCPLeaseScanning() {
	// Implement DHCP lease file scanning
	// This would read DHCP server lease files to discover devices

	log.Printf("Scanning DHCP leases for device discovery")

	// Placeholder implementation
	// Real implementation would parse DHCP lease files
	// from common locations like /var/lib/dhcp/dhcpd.leases
}

func (dd *DeviceDiscovery) scanConnections() {
	// Scan for device connections
	// This involves analyzing routing tables, bridge tables, etc.

	log.Printf("Scanning device connections...")

	// For each discovered device, try to determine its connections
	dd.mu.RLock()
	devices := make([]*types.NetworkDevice, 0, len(dd.devices))
	for _, device := range dd.devices {
		devices = append(devices, device)
	}
	dd.mu.RUnlock()

	for _, device := range devices {
		dd.analyzeDeviceConnections(device)
	}
}

func (dd *DeviceDiscovery) analyzeDeviceConnections(device *types.NetworkDevice) {
	// Analyze device connections based on:
	// 1. Routing table information
	// 2. Bridge table information
	// 3. ARP table entries
	// 4. Network topology inference

	// This is a complex algorithm that would involve:
	// - Layer 2 connectivity analysis (bridge tables, switch forwarding)
	// - Layer 3 connectivity analysis (routing tables, next-hop analysis)
	// - WiFi client association analysis
	// - Network trace analysis

	// For now, this is a placeholder
	log.Printf("Analyzing connections for device: %s", device.DeviceID)
}

func (dd *DeviceDiscovery) updateDeviceTimeouts() {
	// Mark devices as offline if they haven't been seen recently
	now := time.Now().UnixMilli()
	timeout := dd.config.DeviceTimeout.Milliseconds()

	dd.mu.Lock()
	defer dd.mu.Unlock()

	for _, device := range dd.devices {
		if device.Online && (now-device.LastSeen) > timeout {
			device.Online = false
			dd.stats.DevicesOnline--
			dd.stats.DevicesOffline++

			log.Printf("Device %s marked as offline (timeout)", device.DeviceID)

			// Save updated status
			if err := dd.storage.SaveNetworkDevice(device); err != nil {
				log.Printf("Failed to save device offline status: %v", err)
			}
		}
	}
}

func (dd *DeviceDiscovery) loadExistingDevices() error {
	// Load devices from storage
	// TODO: Implement GetAllNetworkDevices
	// devices, err := dd.storage.GetAllNetworkDevices()
	var devices []*types.NetworkDevice
	var err error
	if err != nil {
		return fmt.Errorf("failed to load devices from storage: %w", err)
	}

	for _, device := range devices {
		dd.devices[device.DeviceID] = device
		dd.stats.TotalDevicesDiscovered++

		if device.Online {
			dd.stats.DevicesOnline++
		} else {
			dd.stats.DevicesOffline++
		}
	}

	log.Printf("Loaded %d existing devices from storage", len(devices))
	return nil
}

func (dd *DeviceDiscovery) classifyDevice(device *types.NetworkDevice) {
	// Apply classification rules to determine device type and role

	for _, rule := range dd.config.ClassificationRules {
		if dd.matchesRule(device, rule) {
			// Apply rule classification
			if device.DeviceType == "" || rule.Confidence > 0.8 {
				device.DeviceType = rule.DeviceType
			}

			if device.Role == "" || rule.Confidence > 0.8 {
				device.Role = rule.Role
			}

			log.Printf("Device %s classified as %s/%s by rule %s (confidence: %.2f)",
				device.DeviceID, device.DeviceType, device.Role, rule.Name, rule.Confidence)
			break
		}
	}
}

func (dd *DeviceDiscovery) matchesRule(device *types.NetworkDevice, rule ClassificationRule) bool {
	// Check if device matches all conditions in the rule

	for _, condition := range rule.Conditions {
		if !dd.matchesCondition(device, condition) {
			return false
		}
	}

	return true
}

func (dd *DeviceDiscovery) matchesCondition(device *types.NetworkDevice, condition Condition) bool {
	// Get field value from device
	var fieldValue interface{}

	switch condition.Field {
	case "manufacturer":
		fieldValue = device.Manufacturer
	case "hostname":
		fieldValue = device.Hostname
	case "device_type":
		fieldValue = device.DeviceType
	case "primary_mac":
		fieldValue = device.PrimaryMAC
	case "capabilities":
		fieldValue = device.Capabilities
	default:
		return false
	}

	// Apply operator
	switch condition.Operator {
	case "equals":
		return fieldValue == condition.Value
	case "contains":
		if str, ok := fieldValue.(string); ok {
			if substr, ok := condition.Value.(string); ok {
				return len(str) > 0 && len(substr) > 0 &&
					fmt.Sprintf("%v", str) == fmt.Sprintf("%v", substr)
			}
		}
		return false
	case "in":
		if slice, ok := condition.Value.([]interface{}); ok {
			for _, item := range slice {
				if fieldValue == item {
					return true
				}
			}
		}
		return false
	default:
		return false
	}
}
