package identity

import (
	"log"
	"rtk_controller/pkg/types"
)

func (m *Manager) initializeDefaultRules() {
	defaultRules := []types.DetectionRule{
		{
			ID:          "apple-devices",
			Name:        "Apple Devices",
			Description: "Detect Apple devices by MAC prefix",
			Enabled:     true,
			Priority:    100,
			Conditions: []types.DetectionCondition{
				{Type: "mac_prefix", Operator: "starts_with", Value: "00:03:93"},
			},
			Action: types.DetectionAction{
				SetDeviceType:   "apple_device",
				SetManufacturer: "Apple",
				Confidence:      0.9,
			},
		},
		{
			ID:          "samsung-devices",
			Name:        "Samsung Devices",
			Description: "Detect Samsung devices by MAC prefix",
			Enabled:     true,
			Priority:    90,
			Conditions: []types.DetectionCondition{
				{Type: "mac_prefix", Operator: "starts_with", Value: "00:08:22"},
			},
			Action: types.DetectionAction{
				SetDeviceType:   "samsung_device",
				SetManufacturer: "Samsung",
				Confidence:      0.85,
			},
		},
	}

	for i := range defaultRules {
		if err := m.AddDetectionRule(&defaultRules[i]); err != nil {
			log.Printf("Failed to add default detection rule %s: %v", defaultRules[i].ID, err)
		}
	}
}
