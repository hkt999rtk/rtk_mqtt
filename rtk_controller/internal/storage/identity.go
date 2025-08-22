package storage

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"rtk_controller/pkg/types"
)

// IdentityStorage provides device identity storage operations
type IdentityStorage struct {
	storage Storage
}

// NewIdentityStorage creates a new identity storage
func NewIdentityStorage(storage Storage) *IdentityStorage {
	return &IdentityStorage{
		storage: storage,
	}
}

// Device Identity operations

// SaveDeviceIdentity saves a device identity
func (is *IdentityStorage) SaveDeviceIdentity(identity *types.DeviceIdentity) error {
	data, err := json.Marshal(identity)
	if err != nil {
		return fmt.Errorf("failed to marshal device identity: %w", err)
	}

	key := fmt.Sprintf("identity:%s", identity.MacAddress)
	return is.storage.Set(key, string(data))
}

// GetDeviceIdentity retrieves a device identity by MAC address
func (is *IdentityStorage) GetDeviceIdentity(macAddress string) (*types.DeviceIdentity, error) {
	key := fmt.Sprintf("identity:%s", macAddress)
	data, err := is.storage.Get(key)
	if err != nil {
		return nil, fmt.Errorf("failed to get device identity: %w", err)
	}

	var identity types.DeviceIdentity
	if err := json.Unmarshal([]byte(data), &identity); err != nil {
		return nil, fmt.Errorf("failed to unmarshal device identity: %w", err)
	}

	return &identity, nil
}

// DeleteDeviceIdentity deletes a device identity
func (is *IdentityStorage) DeleteDeviceIdentity(macAddress string) error {
	key := fmt.Sprintf("identity:%s", macAddress)
	return is.storage.Delete(key)
}

// ListDeviceIdentities lists device identities with optional filtering
func (is *IdentityStorage) ListDeviceIdentities(filter *types.DeviceIdentityFilter, limit, offset int) ([]*types.DeviceIdentity, int, error) {
	var identities []*types.DeviceIdentity
	var totalCount int

	err := is.storage.View(func(tx Transaction) error {
		prefix := "identity:"
		return tx.IteratePrefix(prefix, func(key, value string) error {
			var identity types.DeviceIdentity
			if err := json.Unmarshal([]byte(value), &identity); err != nil {
				return fmt.Errorf("failed to unmarshal identity %s: %w", key, err)
			}

			// Apply filters
			if filter != nil && !is.matchesFilter(&identity, filter) {
				return nil
			}

			totalCount++

			// Apply pagination
			if offset > 0 && totalCount <= offset {
				return nil
			}

			if limit > 0 && len(identities) >= limit {
				return nil
			}

			identities = append(identities, &identity)
			return nil
		})
	})

	return identities, totalCount, err
}

// ExistsDeviceIdentity checks if a device identity exists
func (is *IdentityStorage) ExistsDeviceIdentity(macAddress string) (bool, error) {
	key := fmt.Sprintf("identity:%s", macAddress)
	return is.storage.Exists(key)
}

// Detection Rules operations

// SaveDetectionRule saves a detection rule
func (is *IdentityStorage) SaveDetectionRule(rule *types.DetectionRule) error {
	data, err := json.Marshal(rule)
	if err != nil {
		return fmt.Errorf("failed to marshal detection rule: %w", err)
	}

	key := fmt.Sprintf("detection_rule:%s", rule.ID)
	return is.storage.Set(key, string(data))
}

// GetDetectionRule retrieves a detection rule by ID
func (is *IdentityStorage) GetDetectionRule(ruleID string) (*types.DetectionRule, error) {
	key := fmt.Sprintf("detection_rule:%s", ruleID)
	data, err := is.storage.Get(key)
	if err != nil {
		return nil, fmt.Errorf("failed to get detection rule: %w", err)
	}

	var rule types.DetectionRule
	if err := json.Unmarshal([]byte(data), &rule); err != nil {
		return nil, fmt.Errorf("failed to unmarshal detection rule: %w", err)
	}

	return &rule, nil
}

// DeleteDetectionRule deletes a detection rule
func (is *IdentityStorage) DeleteDetectionRule(ruleID string) error {
	key := fmt.Sprintf("detection_rule:%s", ruleID)
	return is.storage.Delete(key)
}

// ListDetectionRules lists all detection rules
func (is *IdentityStorage) ListDetectionRules() ([]*types.DetectionRule, error) {
	var rules []*types.DetectionRule

	err := is.storage.View(func(tx Transaction) error {
		prefix := "detection_rule:"
		return tx.IteratePrefix(prefix, func(key, value string) error {
			var rule types.DetectionRule
			if err := json.Unmarshal([]byte(value), &rule); err != nil {
				return fmt.Errorf("failed to unmarshal detection rule %s: %w", key, err)
			}
			rules = append(rules, &rule)
			return nil
		})
	})

	return rules, err
}

// Statistics operations

// GetIdentityStats returns device identity statistics
func (is *IdentityStorage) GetIdentityStats() (*types.DeviceIdentityStats, error) {
	stats := &types.DeviceIdentityStats{
		DeviceTypeStats:   make(map[string]int),
		ManufacturerStats: make(map[string]int),
		LocationStats:     make(map[string]int),
		CategoryStats:     make(map[string]int),
		LastUpdated:       time.Now(),
	}

	err := is.storage.View(func(tx Transaction) error {
		prefix := "identity:"
		return tx.IteratePrefix(prefix, func(key, value string) error {
			var identity types.DeviceIdentity
			if err := json.Unmarshal([]byte(value), &identity); err != nil {
				return fmt.Errorf("failed to unmarshal identity %s: %w", key, err)
			}

			stats.TotalDevices++

			if identity.AutoDetected {
				stats.AutoDetectedDevices++
			} else {
				stats.ManualDevices++
			}

			// Device type statistics
			if identity.DeviceType != "" {
				stats.DeviceTypeStats[identity.DeviceType]++
			}

			// Manufacturer statistics
			if identity.Manufacturer != "" {
				stats.ManufacturerStats[identity.Manufacturer]++
			}

			// Location statistics
			if identity.Location != "" {
				stats.LocationStats[identity.Location]++
			}

			// Category statistics
			if identity.Category != "" {
				stats.CategoryStats[identity.Category]++
			}

			return nil
		})
	})

	return stats, err
}

// Cleanup operations

// CleanupOldIdentities removes identities that haven't been seen for a specified duration
func (is *IdentityStorage) CleanupOldIdentities(maxAge time.Duration) (int, error) {
	cutoff := time.Now().Add(-maxAge)
	var deletedCount int

	err := is.storage.Transaction(func(tx Transaction) error {
		prefix := "identity:"
		var keysToDelete []string

		err := tx.IteratePrefix(prefix, func(key, value string) error {
			var identity types.DeviceIdentity
			if err := json.Unmarshal([]byte(value), &identity); err != nil {
				// Delete malformed entries
				keysToDelete = append(keysToDelete, key)
				return nil
			}

			if identity.LastSeen.Before(cutoff) {
				keysToDelete = append(keysToDelete, key)
			}
			return nil
		})

		if err != nil {
			return err
		}

		// Delete old identities
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

// Search operations

// SearchDeviceIdentities searches device identities by text
func (is *IdentityStorage) SearchDeviceIdentities(query string, limit int) ([]*types.DeviceIdentity, error) {
	var identities []*types.DeviceIdentity
	query = strings.ToLower(query)

	err := is.storage.View(func(tx Transaction) error {
		prefix := "identity:"
		return tx.IteratePrefix(prefix, func(key, value string) error {
			if limit > 0 && len(identities) >= limit {
				return ErrStopIteration
			}

			var identity types.DeviceIdentity
			if err := json.Unmarshal([]byte(value), &identity); err != nil {
				return nil // Skip malformed entries
			}

			// Search in various fields
			searchFields := []string{
				strings.ToLower(identity.MacAddress),
				strings.ToLower(identity.FriendlyName),
				strings.ToLower(identity.DeviceType),
				strings.ToLower(identity.Manufacturer),
				strings.ToLower(identity.Model),
				strings.ToLower(identity.Location),
				strings.ToLower(identity.Owner),
			}

			// Add tags to search fields
			for _, tag := range identity.Tags {
				searchFields = append(searchFields, strings.ToLower(tag))
			}

			// Check if query matches any field
			for _, field := range searchFields {
				if strings.Contains(field, query) {
					identities = append(identities, &identity)
					break
				}
			}

			return nil
		})
	})

	if err == ErrStopIteration {
		err = nil
	}

	return identities, err
}

// Batch operations

// SaveMultipleIdentities saves multiple device identities in a single transaction
func (is *IdentityStorage) SaveMultipleIdentities(identities []*types.DeviceIdentity) error {
	return is.storage.Transaction(func(tx Transaction) error {
		for _, identity := range identities {
			data, err := json.Marshal(identity)
			if err != nil {
				return fmt.Errorf("failed to marshal identity %s: %w", identity.MacAddress, err)
			}

			key := fmt.Sprintf("identity:%s", identity.MacAddress)
			if err := tx.Set(key, string(data)); err != nil {
				return fmt.Errorf("failed to save identity %s: %w", identity.MacAddress, err)
			}
		}
		return nil
	})
}

// DeleteMultipleIdentities deletes multiple device identities in a single transaction
func (is *IdentityStorage) DeleteMultipleIdentities(macAddresses []string) error {
	return is.storage.Transaction(func(tx Transaction) error {
		for _, macAddress := range macAddresses {
			key := fmt.Sprintf("identity:%s", macAddress)
			if err := tx.Delete(key); err != nil {
				return fmt.Errorf("failed to delete identity %s: %w", macAddress, err)
			}
		}
		return nil
	})
}

// Export/Import operations

// ExportIdentities exports all device identities
func (is *IdentityStorage) ExportIdentities() ([]*types.DeviceIdentity, error) {
	var identities []*types.DeviceIdentity

	err := is.storage.View(func(tx Transaction) error {
		prefix := "identity:"
		return tx.IteratePrefix(prefix, func(key, value string) error {
			var identity types.DeviceIdentity
			if err := json.Unmarshal([]byte(value), &identity); err != nil {
				return fmt.Errorf("failed to unmarshal identity %s: %w", key, err)
			}
			identities = append(identities, &identity)
			return nil
		})
	})

	return identities, err
}

// Private helper methods

func (is *IdentityStorage) matchesFilter(identity *types.DeviceIdentity, filter *types.DeviceIdentityFilter) bool {
	// Device type filter
	if filter.DeviceType != "" && !strings.EqualFold(identity.DeviceType, filter.DeviceType) {
		return false
	}

	// Manufacturer filter
	if filter.Manufacturer != "" && !strings.EqualFold(identity.Manufacturer, filter.Manufacturer) {
		return false
	}

	// Location filter
	if filter.Location != "" && !strings.EqualFold(identity.Location, filter.Location) {
		return false
	}

	// Owner filter
	if filter.Owner != "" && !strings.EqualFold(identity.Owner, filter.Owner) {
		return false
	}

	// Category filter
	if filter.Category != "" && !strings.EqualFold(identity.Category, filter.Category) {
		return false
	}

	// Auto-detected filter
	if filter.AutoDetected != nil && identity.AutoDetected != *filter.AutoDetected {
		return false
	}

	// Tags filter (identity must have all specified tags)
	if len(filter.Tags) > 0 {
		for _, filterTag := range filter.Tags {
			found := false
			for _, identityTag := range identity.Tags {
				if strings.EqualFold(identityTag, filterTag) {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
	}

	// Last seen filters
	lastSeenUnix := identity.LastSeen.Unix()
	if filter.LastSeenAfter != nil && lastSeenUnix < *filter.LastSeenAfter {
		return false
	}
	if filter.LastSeenBefore != nil && lastSeenUnix > *filter.LastSeenBefore {
		return false
	}

	return true
}
