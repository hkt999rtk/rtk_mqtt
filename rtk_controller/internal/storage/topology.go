package storage

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"rtk_controller/pkg/types"
)

// TopologyStorage provides topology-specific storage operations
type TopologyStorage struct {
	storage Storage
}

// NewTopologyStorage creates a new topology storage
func NewTopologyStorage(storage Storage) *TopologyStorage {
	return &TopologyStorage{
		storage: storage,
	}
}

// Topology operations

// SaveTopology saves a network topology
func (ts *TopologyStorage) SaveTopology(topology *types.NetworkTopology) error {
	data, err := json.Marshal(topology)
	if err != nil {
		return fmt.Errorf("failed to marshal topology: %w", err)
	}

	key := fmt.Sprintf("topology:%s:%s", topology.Tenant, topology.Site)
	return ts.storage.Set(key, string(data))
}

// GetTopology retrieves a network topology
func (ts *TopologyStorage) GetTopology(tenant, site string) (*types.NetworkTopology, error) {
	key := fmt.Sprintf("topology:%s:%s", tenant, site)
	data, err := ts.storage.Get(key)
	if err != nil {
		return nil, fmt.Errorf("failed to get topology: %w", err)
	}

	var topology types.NetworkTopology
	if err := json.Unmarshal([]byte(data), &topology); err != nil {
		return nil, fmt.Errorf("failed to unmarshal topology: %w", err)
	}

	return &topology, nil
}

// DeleteTopology deletes a network topology
func (ts *TopologyStorage) DeleteTopology(tenant, site string) error {
	key := fmt.Sprintf("topology:%s:%s", tenant, site)
	return ts.storage.Delete(key)
}

// ListTopologies lists all topologies for a tenant
func (ts *TopologyStorage) ListTopologies(tenant string) ([]*types.NetworkTopology, error) {
	var topologies []*types.NetworkTopology

	err := ts.storage.View(func(tx Transaction) error {
		prefix := fmt.Sprintf("topology:%s:", tenant)
		return tx.IteratePrefix(prefix, func(key, value string) error {
			var topology types.NetworkTopology
			if err := json.Unmarshal([]byte(value), &topology); err != nil {
				return fmt.Errorf("failed to unmarshal topology %s: %w", key, err)
			}
			topologies = append(topologies, &topology)
			return nil
		})
	})

	return topologies, err
}

// Device operations

// SaveNetworkDevice saves a network device
func (ts *TopologyStorage) SaveNetworkDevice(device *types.NetworkDevice) error {
	data, err := json.Marshal(device)
	if err != nil {
		return fmt.Errorf("failed to marshal network device: %w", err)
	}

	key := fmt.Sprintf("network_device:%s", device.DeviceID)
	return ts.storage.Set(key, string(data))
}

// GetNetworkDevice retrieves a network device
func (ts *TopologyStorage) GetNetworkDevice(deviceID string) (*types.NetworkDevice, error) {
	key := fmt.Sprintf("network_device:%s", deviceID)
	data, err := ts.storage.Get(key)
	if err != nil {
		return nil, fmt.Errorf("failed to get network device: %w", err)
	}

	var device types.NetworkDevice
	if err := json.Unmarshal([]byte(data), &device); err != nil {
		return nil, fmt.Errorf("failed to unmarshal network device: %w", err)
	}

	return &device, nil
}

// Connection operations

// SaveDeviceConnection saves a device connection
func (ts *TopologyStorage) SaveDeviceConnection(connection *types.DeviceConnection) error {
	data, err := json.Marshal(connection)
	if err != nil {
		return fmt.Errorf("failed to marshal device connection: %w", err)
	}

	key := fmt.Sprintf("connection:%s", connection.ID)
	return ts.storage.Set(key, string(data))
}

// GetDeviceConnection retrieves a device connection
func (ts *TopologyStorage) GetDeviceConnection(connectionID string) (*types.DeviceConnection, error) {
	key := fmt.Sprintf("connection:%s", connectionID)
	data, err := ts.storage.Get(key)
	if err != nil {
		return nil, fmt.Errorf("failed to get device connection: %w", err)
	}

	var connection types.DeviceConnection
	if err := json.Unmarshal([]byte(data), &connection); err != nil {
		return nil, fmt.Errorf("failed to unmarshal device connection: %w", err)
	}

	return &connection, nil
}

// ListDeviceConnections lists all connections for a device
func (ts *TopologyStorage) ListDeviceConnections(deviceID string) ([]*types.DeviceConnection, error) {
	var connections []*types.DeviceConnection

	err := ts.storage.View(func(tx Transaction) error {
		prefix := "connection:"
		return tx.IteratePrefix(prefix, func(key, value string) error {
			var connection types.DeviceConnection
			if err := json.Unmarshal([]byte(value), &connection); err != nil {
				return fmt.Errorf("failed to unmarshal connection %s: %w", key, err)
			}
			
			// Filter connections related to the specified device
			if connection.FromDeviceID == deviceID || connection.ToDeviceID == deviceID {
				connections = append(connections, &connection)
			}
			return nil
		})
	})

	return connections, err
}

// Gateway operations

// SaveGatewayInfo saves gateway information
func (ts *TopologyStorage) SaveGatewayInfo(gatewayInfo *types.GatewayInfo) error {
	data, err := json.Marshal(gatewayInfo)
	if err != nil {
		return fmt.Errorf("failed to marshal gateway info: %w", err)
	}

	key := fmt.Sprintf("gateway:%s", gatewayInfo.DeviceID)
	return ts.storage.Set(key, string(data))
}

// GetGatewayInfo retrieves gateway information
func (ts *TopologyStorage) GetGatewayInfo(deviceID string) (*types.GatewayInfo, error) {
	key := fmt.Sprintf("gateway:%s", deviceID)
	data, err := ts.storage.Get(key)
	if err != nil {
		return nil, fmt.Errorf("failed to get gateway info: %w", err)
	}

	var gatewayInfo types.GatewayInfo
	if err := json.Unmarshal([]byte(data), &gatewayInfo); err != nil {
		return nil, fmt.Errorf("failed to unmarshal gateway info: %w", err)
	}

	return &gatewayInfo, nil
}

// Utility operations

// CleanupOldConnections removes connections older than the specified duration
func (ts *TopologyStorage) CleanupOldConnections(maxAge time.Duration) (int, error) {
	cutoff := time.Now().Add(-maxAge).UnixMilli()
	var deletedCount int

	err := ts.storage.Transaction(func(tx Transaction) error {
		prefix := "connection:"
		var keysToDelete []string

		err := tx.IteratePrefix(prefix, func(key, value string) error {
			var connection types.DeviceConnection
			if err := json.Unmarshal([]byte(value), &connection); err != nil {
				// Delete malformed entries
				keysToDelete = append(keysToDelete, key)
				return nil
			}

			if connection.LastSeen < cutoff {
				keysToDelete = append(keysToDelete, key)
			}
			return nil
		})

		if err != nil {
			return err
		}

		// Delete old connections
		for _, key := range keysToDelete {
			if err := tx.Delete(key); err != nil {
				return err
			}
			deletedCount++
		}

		return nil
	})

	return deletedCount, err
}

// GetTopologyStats returns statistics about stored topology data
func (ts *TopologyStorage) GetTopologyStats() (map[string]int, error) {
	stats := make(map[string]int)

	err := ts.storage.View(func(tx Transaction) error {
		prefixes := []string{
			"topology:",
			"network_device:",
			"connection:",
			"gateway:",
		}

		for _, prefix := range prefixes {
			count := 0
			err := tx.IteratePrefix(prefix, func(key, value string) error {
				count++
				return nil
			})
			if err != nil {
				return err
			}
			
			// Extract the type name from prefix
			typeName := strings.TrimSuffix(prefix, ":")
			stats[typeName] = count
		}

		return nil
	})

	return stats, err
}