package topology

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"rtk_controller/internal/identity"
	"rtk_controller/internal/mqtt"
	"rtk_controller/internal/storage"
	"rtk_controller/pkg/types"
)

// Manager orchestrates all topology-related functionality
type Manager struct {
	// Core components
	storage           *storage.TopologyStorage
	identityStorage   *storage.IdentityStorage
	identityManager   *identity.Manager
	topologyProcessor *mqtt.TopologyProcessor
	deviceDiscovery   *DeviceDiscovery

	// Current topology state
	topology *types.NetworkTopology
	mu       sync.RWMutex

	// Configuration
	config ManagerConfig

	// Background processing
	running bool
	cancel  context.CancelFunc

	// Statistics
	stats ManagerStats
}

// ManagerConfig holds topology manager configuration
type ManagerConfig struct {
	// Core settings
	Tenant string
	Site   string

	// Update intervals
	TopologyUpdateInterval time.Duration
	MetricsUpdateInterval  time.Duration
	CleanupInterval        time.Duration

	// Data retention
	ConnectionHistoryRetention time.Duration
	MetricsRetention           time.Duration
	DeviceOfflineRetention     time.Duration

	// Processing settings
	EnableRealTimeUpdates      bool
	EnableConnectionInference  bool
	EnableMetricsCollection    bool
	EnableDeviceClassification bool

	// Discovery configuration
	DiscoveryConfig DiscoveryConfig
}

// ManagerStats holds topology manager statistics
type ManagerStats struct {
	// Topology stats
	TotalDevices      int64
	OnlineDevices     int64
	OfflineDevices    int64
	TotalConnections  int64
	ActiveConnections int64

	// Processing stats
	TopologyUpdates    int64
	ConnectionUpdates  int64
	DeviceUpdates      int64
	LastTopologyUpdate time.Time
	ProcessingErrors   int64

	// Discovery stats
	DiscoveryStats DiscoveryStats
}

// NewManager creates a new topology manager
func NewManager(
	topologyStorage *storage.TopologyStorage,
	identityStorage *storage.IdentityStorage,
	identityManager *identity.Manager,
	config ManagerConfig,
) (*Manager, error) {

	// Create topology processor
	processor := mqtt.NewTopologyProcessor(
		nil, // Schema manager will be set later
		topologyStorage,
		identityStorage,
	)

	// Create device discovery
	discovery := NewDeviceDiscovery(topologyStorage, processor, config.DiscoveryConfig)

	manager := &Manager{
		storage:           topologyStorage,
		identityStorage:   identityStorage,
		identityManager:   identityManager,
		topologyProcessor: processor,
		deviceDiscovery:   discovery,
		config:            config,
		stats:             ManagerStats{},
	}

	// Initialize topology
	if err := manager.initializeTopology(); err != nil {
		return nil, fmt.Errorf("failed to initialize topology: %w", err)
	}

	return manager, nil
}

// Start begins topology management
func (m *Manager) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.running {
		return fmt.Errorf("topology manager is already running")
	}

	ctx, cancel := context.WithCancel(context.Background())
	m.cancel = cancel
	m.running = true

	log.Printf("Starting topology manager for tenant: %s, site: %s", m.config.Tenant, m.config.Site)

	// Start device discovery
	if err := m.deviceDiscovery.Start(); err != nil {
		m.cancel()
		m.running = false
		return fmt.Errorf("failed to start device discovery: %w", err)
	}

	// Start background processing
	go m.topologyUpdateLoop(ctx)
	go m.metricsUpdateLoop(ctx)
	go m.cleanupLoop(ctx)

	log.Printf("Topology manager started successfully")
	return nil
}

// Stop stops topology management
func (m *Manager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.running {
		return fmt.Errorf("topology manager is not running")
	}

	// Stop device discovery
	if err := m.deviceDiscovery.Stop(); err != nil {
		log.Printf("Error stopping device discovery: %v", err)
	}

	// Stop background processing
	m.cancel()
	m.running = false

	log.Printf("Topology manager stopped")
	return nil
}

// GetTopology returns the current network topology
func (m *Manager) GetTopology() *types.NetworkTopology {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Return a copy of the topology
	if m.topology == nil {
		return nil
	}

	topology := *m.topology
	topology.Devices = make(map[string]*types.NetworkDevice)
	for id, device := range m.topology.Devices {
		deviceCopy := *device
		topology.Devices[id] = &deviceCopy
	}

	topology.Connections = make([]types.DeviceConnection, len(m.topology.Connections))
	copy(topology.Connections, m.topology.Connections)

	return &topology
}

// GetDevice returns information about a specific device
func (m *Manager) GetDevice(deviceID string) (*types.NetworkDevice, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.topology == nil {
		return nil, fmt.Errorf("topology not initialized")
	}

	device, exists := m.topology.Devices[deviceID]
	if !exists {
		return nil, fmt.Errorf("device not found: %s", deviceID)
	}

	// Return a copy
	deviceCopy := *device
	return &deviceCopy, nil
}

// GetConnections returns all connections for a device
func (m *Manager) GetConnections(deviceID string) ([]types.DeviceConnection, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.topology == nil {
		return nil, fmt.Errorf("topology not initialized")
	}

	var connections []types.DeviceConnection
	for _, conn := range m.topology.Connections {
		if conn.FromDeviceID == deviceID || conn.ToDeviceID == deviceID {
			connections = append(connections, conn)
		}
	}

	return connections, nil
}

// ProcessTopologyMessage processes incoming topology-related MQTT messages
func (m *Manager) ProcessTopologyMessage(topic string, payload []byte) error {
	// Delegate to topology processor
	if err := m.topologyProcessor.ProcessMessage(topic, payload); err != nil {
		m.stats.ProcessingErrors++
		return fmt.Errorf("failed to process topology message: %w", err)
	}

	// Trigger topology update if real-time updates are enabled
	if m.config.EnableRealTimeUpdates {
		go m.updateTopology()
	}

	return nil
}

// AddDevice manually adds a device to the topology
func (m *Manager) AddDevice(device *types.NetworkDevice) error {
	// Add to discovery
	if err := m.deviceDiscovery.AddDevice(device); err != nil {
		return fmt.Errorf("failed to add device to discovery: %w", err)
	}

	// Update topology
	if err := m.updateTopology(); err != nil {
		return fmt.Errorf("failed to update topology: %w", err)
	}

	return nil
}

// GetStats returns topology manager statistics
func (m *Manager) GetStats() ManagerStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Update device counts
	if m.topology != nil {
		m.stats.TotalDevices = int64(len(m.topology.Devices))

		var online, offline int64
		for _, device := range m.topology.Devices {
			if device.Online {
				online++
			} else {
				offline++
			}
		}

		m.stats.OnlineDevices = online
		m.stats.OfflineDevices = offline
		m.stats.TotalConnections = int64(len(m.topology.Connections))

		var active int64
		now := time.Now().UnixMilli()
		for _, conn := range m.topology.Connections {
			// Consider connection active if seen within last 5 minutes
			if (now - conn.LastSeen) < 5*60*1000 {
				active++
			}
		}
		m.stats.ActiveConnections = active
	}

	// Include discovery stats
	m.stats.DiscoveryStats = m.deviceDiscovery.GetStats()

	return m.stats
}

// Private methods

func (m *Manager) initializeTopology() error {
	// Create or load existing topology
	topology, err := m.storage.GetTopology(m.config.Tenant, m.config.Site)
	if err != nil {
		// Create new topology
		topology = &types.NetworkTopology{
			ID:          fmt.Sprintf("%s-%s", m.config.Tenant, m.config.Site),
			Tenant:      m.config.Tenant,
			Site:        m.config.Site,
			Devices:     make(map[string]*types.NetworkDevice),
			Connections: []types.DeviceConnection{},
			UpdatedAt:   time.Now(),
		}

		if err := m.storage.SaveTopology(topology); err != nil {
			return fmt.Errorf("failed to save initial topology: %w", err)
		}

		log.Printf("Created new topology for tenant: %s, site: %s", m.config.Tenant, m.config.Site)
	} else {
		log.Printf("Loaded existing topology with %d devices", len(topology.Devices))
	}

	m.topology = topology
	return nil
}

func (m *Manager) topologyUpdateLoop(ctx context.Context) {
	ticker := time.NewTicker(m.config.TopologyUpdateInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := m.updateTopology(); err != nil {
				log.Printf("Failed to update topology: %v", err)
				m.stats.ProcessingErrors++
			}
		}
	}
}

func (m *Manager) metricsUpdateLoop(ctx context.Context) {
	if !m.config.EnableMetricsCollection {
		return
	}

	ticker := time.NewTicker(m.config.MetricsUpdateInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.updateMetrics()
		}
	}
}

func (m *Manager) cleanupLoop(ctx context.Context) {
	ticker := time.NewTicker(m.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.runCleanup()
		}
	}
}

func (m *Manager) updateTopology() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Get discovered devices
	discoveredDevices := m.deviceDiscovery.GetDiscoveredDevices()

	// Update topology with discovered devices
	for deviceID, device := range discoveredDevices {
		m.topology.Devices[deviceID] = device
	}

	// Update connections if inference is enabled
	if m.config.EnableConnectionInference {
		if err := m.inferConnections(); err != nil {
			log.Printf("Failed to infer connections: %v", err)
		}
	}

	// Update topology timestamp
	m.topology.UpdatedAt = time.Now()

	// Save topology
	if err := m.storage.SaveTopology(m.topology); err != nil {
		return fmt.Errorf("failed to save topology: %w", err)
	}

	m.stats.TopologyUpdates++
	m.stats.LastTopologyUpdate = time.Now()

	return nil
}

func (m *Manager) inferConnections() error {
	// Implement connection inference algorithm
	// This is a complex algorithm that analyzes:
	// 1. Device routing tables
	// 2. Bridge tables (for switches)
	// 3. ARP tables
	// 4. WiFi client associations
	// 5. Network topology patterns

	var newConnections []types.DeviceConnection

	// Example inference logic (simplified)
	for _, device := range m.topology.Devices {
		// Analyze device routing info to infer connections
		if device.RoutingInfo != nil {
			connections := m.inferConnectionsFromRouting(device)
			newConnections = append(newConnections, connections...)
		}

		// Analyze device bridge info to infer connections
		if device.BridgeInfo != nil {
			connections := m.inferConnectionsFromBridge(device)
			newConnections = append(newConnections, connections...)
		}
	}

	// Update topology connections
	m.topology.Connections = newConnections
	m.stats.ConnectionUpdates++

	return nil
}

func (m *Manager) inferConnectionsFromRouting(device *types.NetworkDevice) []types.DeviceConnection {
	var connections []types.DeviceConnection

	// Analyze routing table to find next-hop relationships
	for _, route := range device.RoutingInfo.RoutingTable {
		if route.Gateway != "" && route.Gateway != "0.0.0.0" {
			// Try to find the gateway device
			for _, otherDevice := range m.topology.Devices {
				if otherDevice.DeviceID == device.DeviceID {
					continue
				}

				// Check if this device has the gateway IP
				for _, iface := range otherDevice.Interfaces {
					for _, ipInfo := range iface.IPAddresses {
						if ipInfo.Address == route.Gateway {
							// Found connection
							connection := types.DeviceConnection{
								ID:             fmt.Sprintf("%s-%s-route", device.DeviceID, otherDevice.DeviceID),
								FromDeviceID:   device.DeviceID,
								ToDeviceID:     otherDevice.DeviceID,
								FromInterface:  route.Interface,
								ToInterface:    iface.Name,
								ConnectionType: "route",
								IsDirectLink:   false,
								LastSeen:       time.Now().UnixMilli(),
								Discovered:     time.Now().UnixMilli(),
							}
							connections = append(connections, connection)
						}
					}
				}
			}
		}
	}

	return connections
}

func (m *Manager) inferConnectionsFromBridge(device *types.NetworkDevice) []types.DeviceConnection {
	var connections []types.DeviceConnection

	// Analyze bridge table to find Layer 2 connections
	for _, entry := range device.BridgeInfo.BridgeTable {
		if !entry.IsLocal {
			// This MAC is learned on a specific interface, indicating a connection
			// Try to find the device with this MAC
			for _, otherDevice := range m.topology.Devices {
				if otherDevice.DeviceID == device.DeviceID {
					continue
				}

				if otherDevice.PrimaryMAC == entry.MacAddress {
					// Found connection
					connection := types.DeviceConnection{
						ID:             fmt.Sprintf("%s-%s-bridge", device.DeviceID, otherDevice.DeviceID),
						FromDeviceID:   device.DeviceID,
						ToDeviceID:     otherDevice.DeviceID,
						FromInterface:  entry.Interface,
						ConnectionType: "bridge",
						IsDirectLink:   true,
						LastSeen:       time.Now().UnixMilli(),
						Discovered:     time.Now().UnixMilli(),
					}
					connections = append(connections, connection)
				}
			}
		}
	}

	return connections
}

func (m *Manager) updateMetrics() {
	// Update device and connection metrics
	// This would typically involve:
	// 1. Collecting performance metrics from devices
	// 2. Calculating connection quality metrics
	// 3. Updating historical data

	m.stats.DeviceUpdates++
}

func (m *Manager) runCleanup() {
	// Clean up old data based on retention policies

	now := time.Now()
	cutoff := now.Add(-m.config.DeviceOfflineRetention)

	// Remove old offline devices
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.topology != nil {
		for deviceID, device := range m.topology.Devices {
			if !device.Online && time.UnixMilli(device.LastSeen).Before(cutoff) {
				delete(m.topology.Devices, deviceID)
				log.Printf("Removed old offline device: %s", deviceID)
			}
		}
	}

	// TODO: Clean up old connection history and metrics
}

// GetCurrentTopology returns the current network topology
func (m *Manager) GetCurrentTopology() (*types.NetworkTopology, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.topology, nil
}
