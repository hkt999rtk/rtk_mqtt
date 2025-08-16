package device

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"rtk_controller/internal/storage"
	"rtk_controller/pkg/types"
	"rtk_controller/pkg/utils"

	log "github.com/sirupsen/logrus"
)

// Manager handles device lifecycle and state management
type Manager struct {
	storage storage.Storage
	
	// Device state cache
	devices map[string]*types.DeviceState
	mu      sync.RWMutex
	
	// Background workers
	ctx    context.Context
	cancel context.CancelFunc
	done   chan struct{}
	
	// Statistics
	stats *types.DeviceStats
}

// NewManager creates a new device manager
func NewManager(storage storage.Storage) *Manager {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &Manager{
		storage: storage,
		devices: make(map[string]*types.DeviceState),
		ctx:     ctx,
		cancel:  cancel,
		done:    make(chan struct{}),
		stats:   &types.DeviceStats{},
	}
}

// Start starts the device manager
func (m *Manager) Start(ctx context.Context) error {
	log.Info("Starting device manager...")
	
	// Load existing devices from storage
	if err := m.loadDevicesFromStorage(); err != nil {
		return fmt.Errorf("failed to load devices from storage: %w", err)
	}
	
	// Start background workers
	go m.deviceCleanupWorker()
	go m.statsWorker()
	
	log.WithField("device_count", len(m.devices)).Info("Device manager started")
	return nil
}

// Stop stops the device manager
func (m *Manager) Stop() {
	log.Info("Stopping device manager...")
	
	m.cancel()
	<-m.done
	
	// Save current state to storage
	if err := m.saveDevicesToStorage(); err != nil {
		log.WithError(err).Error("Failed to save devices to storage on shutdown")
	}
	
	log.Info("Device manager stopped")
}

// UpdateDeviceState updates device state from MQTT message
func (m *Manager) UpdateDeviceState(topic string, payload []byte) error {
	// Parse topic to extract device information
	tenant, site, deviceID, messageType, subParts := utils.ExtractTopicParts(topic)
	if deviceID == "" {
		return fmt.Errorf("invalid topic format: %s", topic)
	}
	
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// Get or create device
	device := m.getOrCreateDevice(tenant, site, deviceID)
	device.LastSeen = time.Now().UnixMilli()
	device.Online = true
	device.UpdatedAt = time.Now()
	
	// Parse JSON payload
	var data map[string]interface{}
	if err := json.Unmarshal(payload, &data); err != nil {
		return fmt.Errorf("failed to parse JSON payload: %w", err)
	}
	
	// Update device based on message type
	switch messageType {
	case "state":
		return m.updateDeviceFromState(device, data)
	case "attr":
		return m.updateDeviceFromAttributes(device, data)
	case "lwt":
		return m.updateDeviceFromLWT(device, topic, payload, data)
	default:
		if len(subParts) > 0 && subParts[0] == "telemetry" {
			return m.updateDeviceFromTelemetry(device, subParts[1:], data)
		}
	}
	
	return nil
}

// getOrCreateDevice gets existing device or creates a new one
func (m *Manager) getOrCreateDevice(tenant, site, deviceID string) *types.DeviceState {
	key := fmt.Sprintf("%s:%s:%s", tenant, site, deviceID)
	
	if device, exists := m.devices[key]; exists {
		return device
	}
	
	// Create new device
	device := &types.DeviceState{
		ID:         deviceID,
		Tenant:     tenant,
		Site:       site,
		Health:     "unknown",
		LastSeen:   time.Now().UnixMilli(),
		Components: make(map[string]interface{}),
		Attributes: make(map[string]interface{}),
		Telemetry:  make(map[string]interface{}),
		Online:     true,
		UpdatedAt:  time.Now(),
	}
	
	m.devices[key] = device
	return device
}

// updateDeviceFromState updates device from state message
func (m *Manager) updateDeviceFromState(device *types.DeviceState, data map[string]interface{}) error {
	if health, ok := data["health"].(string); ok {
		device.Health = health
	}
	
	if uptimeS, ok := data["uptime_s"].(float64); ok {
		device.UptimeS = int64(uptimeS)
	}
	
	if version, ok := data["version"].(string); ok {
		device.Version = version
	}
	
	if components, ok := data["components"].(map[string]interface{}); ok {
		device.Components = components
	}
	
	return nil
}

// updateDeviceFromAttributes updates device from attributes message
func (m *Manager) updateDeviceFromAttributes(device *types.DeviceState, data map[string]interface{}) error {
	// Merge attributes
	for key, value := range data {
		device.Attributes[key] = value
	}
	
	// Extract device type if present
	if deviceType, ok := data["device_type"].(string); ok {
		device.DeviceType = deviceType
	}
	
	return nil
}

// updateDeviceFromTelemetry updates device from telemetry message
func (m *Manager) updateDeviceFromTelemetry(device *types.DeviceState, subParts []string, data map[string]interface{}) error {
	if len(subParts) == 0 {
		return nil
	}
	
	metricName := subParts[0]
	device.Telemetry[metricName] = data
	
	return nil
}

// updateDeviceFromLWT updates device from Last Will Testament
func (m *Manager) updateDeviceFromLWT(device *types.DeviceState, topic string, payload []byte, data map[string]interface{}) error {
	device.Online = false
	device.LastWill = &types.LastWillMessage{
		Topic:     topic,
		Payload:   string(payload),
		QoS:       0, // Will be updated by caller if needed
		Retained:  false,
		Timestamp: time.Now().UnixMilli(),
	}
	
	// Update health status based on LWT
	if device.Health == "ok" {
		device.Health = "unknown"
	}
	
	return nil
}

// GetDevice returns a device by ID
func (m *Manager) GetDevice(tenant, site, deviceID string) (*types.DeviceState, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	key := fmt.Sprintf("%s:%s:%s", tenant, site, deviceID)
	device, exists := m.devices[key]
	if !exists {
		return nil, fmt.Errorf("device not found: %s", deviceID)
	}
	
	// Return a copy to avoid race conditions
	deviceCopy := *device
	return &deviceCopy, nil
}

// ListDevices returns devices matching the filter
func (m *Manager) ListDevices(filter *types.DeviceFilter, limit int, offset int) ([]*types.DeviceState, int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	var filtered []*types.DeviceState
	
	for _, device := range m.devices {
		if m.deviceMatchesFilter(device, filter) {
			deviceCopy := *device
			filtered = append(filtered, &deviceCopy)
		}
	}
	
	total := len(filtered)
	
	// Apply pagination
	start := offset
	if start > total {
		start = total
	}
	
	end := start + limit
	if limit <= 0 || end > total {
		end = total
	}
	
	return filtered[start:end], total, nil
}

// deviceMatchesFilter checks if device matches the filter criteria
func (m *Manager) deviceMatchesFilter(device *types.DeviceState, filter *types.DeviceFilter) bool {
	if filter == nil {
		return true
	}
	
	if filter.Tenant != "" && device.Tenant != filter.Tenant {
		return false
	}
	
	if filter.Site != "" && device.Site != filter.Site {
		return false
	}
	
	if filter.DeviceType != "" && device.DeviceType != filter.DeviceType {
		return false
	}
	
	if len(filter.Health) > 0 {
		found := false
		for _, health := range filter.Health {
			if device.Health == health {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	
	if filter.Online != nil && device.Online != *filter.Online {
		return false
	}
	
	if filter.LastSeenGT != nil && device.LastSeen <= *filter.LastSeenGT {
		return false
	}
	
	if filter.LastSeenLT != nil && device.LastSeen >= *filter.LastSeenLT {
		return false
	}
	
	return true
}

// GetStats returns device statistics
func (m *Manager) GetStats() *types.DeviceStats {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	// Return a copy of current stats
	stats := *m.stats
	return &stats
}

// deviceCleanupWorker removes offline devices after timeout
func (m *Manager) deviceCleanupWorker() {
	defer close(m.done)
	
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	
	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			m.cleanupOfflineDevices()
		}
	}
}

// cleanupOfflineDevices marks devices as offline if they haven't been seen recently
func (m *Manager) cleanupOfflineDevices() {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	now := time.Now().UnixMilli()
	offlineThreshold := int64(5 * 60 * 1000) // 5 minutes
	
	for _, device := range m.devices {
		if device.Online && (now-device.LastSeen) > offlineThreshold {
			device.Online = false
			device.UpdatedAt = time.Now()
			
			log.WithFields(log.Fields{
				"device_id": device.ID,
				"last_seen": time.UnixMilli(device.LastSeen).Format(time.RFC3339),
			}).Info("Device marked as offline due to inactivity")
		}
	}
}

// statsWorker updates device statistics periodically
func (m *Manager) statsWorker() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			m.updateStats()
		}
	}
}

// updateStats calculates current device statistics
func (m *Manager) updateStats() {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	stats := &types.DeviceStats{
		HealthStats:     make(map[string]int),
		DeviceTypeStats: make(map[string]int),
		LastUpdated:     time.Now(),
	}
	
	for _, device := range m.devices {
		stats.TotalDevices++
		
		if device.Online {
			stats.OnlineDevices++
		} else {
			stats.OfflineDevices++
		}
		
		stats.HealthStats[device.Health]++
		
		if device.DeviceType != "" {
			stats.DeviceTypeStats[device.DeviceType]++
		}
	}
	
	m.stats = stats
}

// loadDevicesFromStorage loads device state from persistent storage
func (m *Manager) loadDevicesFromStorage() error {
	return m.storage.View(func(tx storage.Transaction) error {
		return tx.IteratePrefix("device:", func(key, value string) error {
			var device types.DeviceState
			if err := json.Unmarshal([]byte(value), &device); err != nil {
				log.WithError(err).Warnf("Failed to unmarshal device: %s", key)
				return nil // Continue iteration
			}
			
			deviceKey := fmt.Sprintf("%s:%s:%s", device.Tenant, device.Site, device.ID)
			m.devices[deviceKey] = &device
			
			return nil
		})
	})
}

// saveDevicesToStorage saves current device state to persistent storage
func (m *Manager) saveDevicesToStorage() error {
	return m.storage.Transaction(func(tx storage.Transaction) error {
		for key, device := range m.devices {
			data, err := json.Marshal(device)
			if err != nil {
				log.WithError(err).Errorf("Failed to marshal device: %s", key)
				continue
			}
			
			storageKey := fmt.Sprintf("device:%s", key)
			if err := tx.Set(storageKey, string(data)); err != nil {
				return fmt.Errorf("failed to save device %s: %w", key, err)
			}
		}
		return nil
	})
}