package codec

import (
	"encoding/json"
	"fmt"
	"time"
)

// SchemaManager manages JSON schemas for message validation
type SchemaManager struct {
	schemas map[string]*Schema
}

// Schema represents a message schema
type Schema struct {
	ID          string      `json:"id"`
	Version     string      `json:"version"`
	Title       string      `json:"title"`
	Description string      `json:"description"`
	Type        string      `json:"type"`
	Properties  interface{} `json:"properties"`
	Required    []string    `json:"required,omitempty"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
}

// NewSchemaManager creates a new schema manager
func NewSchemaManager() *SchemaManager {
	sm := &SchemaManager{
		schemas: make(map[string]*Schema),
	}
	
	// Load default schemas
	sm.loadDefaultSchemas()
	
	return sm
}

// loadDefaultSchemas loads the default RTK message schemas
func (sm *SchemaManager) loadDefaultSchemas() {
	// State message schema
	sm.AddSchema(&Schema{
		ID:          "rtk.state/1.0",
		Version:     "1.0",
		Title:       "RTK Device State Message",
		Description: "Schema for device state messages",
		Type:        "object",
		Properties: map[string]interface{}{
			"status": map[string]interface{}{
				"type": "string",
				"enum": []string{"online", "offline", "error", "maintenance"},
			},
			"health": map[string]interface{}{
				"type": "string",
				"enum": []string{"healthy", "warning", "critical", "unknown"},
			},
			"last_seen": map[string]interface{}{
				"type":   "string",
				"format": "date-time",
			},
			"uptime_seconds": map[string]interface{}{
				"type":    "integer",
				"minimum": 0,
			},
			"properties": map[string]interface{}{
				"type": "object",
			},
			"diagnostics": map[string]interface{}{
				"type": "object",
			},
			"timestamp": map[string]interface{}{
				"type":   "string",
				"format": "date-time",
			},
		},
		Required:  []string{"status", "health", "timestamp"},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})

	// Telemetry message schema
	sm.AddSchema(&Schema{
		ID:          "rtk.telemetry/1.0",
		Version:     "1.0",
		Title:       "RTK Telemetry Message",
		Description: "Schema for telemetry messages",
		Type:        "object",
		Properties: map[string]interface{}{
			"metric": map[string]interface{}{
				"type":      "string",
				"minLength": 1,
			},
			"value": map[string]interface{}{
				"oneOf": []map[string]interface{}{
					{"type": "number"},
					{"type": "string"},
					{"type": "boolean"},
				},
			},
			"unit": map[string]interface{}{
				"type": "string",
			},
			"labels": map[string]interface{}{
				"type": "object",
				"patternProperties": map[string]interface{}{
					".*": map[string]interface{}{
						"type": "string",
					},
				},
			},
			"metadata": map[string]interface{}{
				"type": "object",
			},
			"timestamp": map[string]interface{}{
				"type":   "string",
				"format": "date-time",
			},
		},
		Required:  []string{"metric", "value", "timestamp"},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})

	// Event message schema
	sm.AddSchema(&Schema{
		ID:          "rtk.event/1.0",
		Version:     "1.0",
		Title:       "RTK Event Message",
		Description: "Schema for event messages",
		Type:        "object",
		Properties: map[string]interface{}{
			"event_id": map[string]interface{}{
				"type":      "string",
				"minLength": 1,
			},
			"type": map[string]interface{}{
				"type":      "string",
				"minLength": 1,
			},
			"level": map[string]interface{}{
				"type": "string",
				"enum": []string{"info", "warning", "error", "critical"},
			},
			"message": map[string]interface{}{
				"type":      "string",
				"minLength": 1,
			},
			"source": map[string]interface{}{
				"type": "string",
			},
			"category": map[string]interface{}{
				"type": "string",
			},
			"data": map[string]interface{}{
				"type": "object",
			},
			"timestamp": map[string]interface{}{
				"type":   "string",
				"format": "date-time",
			},
			"schema_id": map[string]interface{}{
				"type": "string",
			},
		},
		Required:  []string{"event_id", "type", "level", "message", "timestamp"},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})

	// WiFi roaming event schema
	sm.AddSchema(&Schema{
		ID:          "evt.wifi.roam_miss/1.0",
		Version:     "1.0",
		Title:       "WiFi Roaming Miss Event",
		Description: "Schema for WiFi roaming miss events",
		Type:        "object",
		Properties: map[string]interface{}{
			"event_id": map[string]interface{}{
				"type": "string",
			},
			"type": map[string]interface{}{
				"type": "string",
				"enum": []string{"wifi.roam_miss"},
			},
			"level": map[string]interface{}{
				"type": "string",
				"enum": []string{"warning", "error"},
			},
			"message": map[string]interface{}{
				"type": "string",
			},
			"source": map[string]interface{}{
				"type": "string",
			},
			"data": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"current_ap": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"bssid":     map[string]interface{}{"type": "string"},
							"ssid":      map[string]interface{}{"type": "string"},
							"channel":   map[string]interface{}{"type": "integer"},
							"frequency": map[string]interface{}{"type": "integer"},
							"rssi":      map[string]interface{}{"type": "integer"},
						},
						"required": []string{"bssid", "ssid"},
					},
					"target_ap": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"bssid":     map[string]interface{}{"type": "string"},
							"ssid":      map[string]interface{}{"type": "string"},
							"channel":   map[string]interface{}{"type": "integer"},
							"frequency": map[string]interface{}{"type": "integer"},
							"rssi":      map[string]interface{}{"type": "integer"},
						},
						"required": []string{"bssid", "ssid"},
					},
					"roam_reason": map[string]interface{}{
						"type": "string",
						"enum": []string{"poor_signal", "load_balancing", "better_ap", "manual"},
					},
					"roam_time_ms": map[string]interface{}{
						"type":    "integer",
						"minimum": 0,
					},
					"failure_reason": map[string]interface{}{
						"type": "string",
					},
				},
				"required": []string{"current_ap", "target_ap", "roam_reason"},
			},
			"timestamp": map[string]interface{}{
				"type":   "string",
				"format": "date-time",
			},
		},
		Required:  []string{"event_id", "type", "level", "message", "data", "timestamp"},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})

	// WiFi connection failure event schema
	sm.AddSchema(&Schema{
		ID:          "evt.wifi.connect_fail/1.0",
		Version:     "1.0",
		Title:       "WiFi Connection Failure Event",
		Description: "Schema for WiFi connection failure events",
		Type:        "object",
		Properties: map[string]interface{}{
			"event_id": map[string]interface{}{
				"type": "string",
			},
			"type": map[string]interface{}{
				"type": "string",
				"enum": []string{"wifi.connect_fail"},
			},
			"level": map[string]interface{}{
				"type": "string",
				"enum": []string{"error", "critical"},
			},
			"message": map[string]interface{}{
				"type": "string",
			},
			"source": map[string]interface{}{
				"type": "string",
			},
			"data": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"ssid": map[string]interface{}{
						"type": "string",
					},
					"bssid": map[string]interface{}{
						"type": "string",
					},
					"security_type": map[string]interface{}{
						"type": "string",
						"enum": []string{"open", "wep", "wpa", "wpa2", "wpa3"},
					},
					"failure_stage": map[string]interface{}{
						"type": "string",
						"enum": []string{"association", "authentication", "dhcp", "internet_check"},
					},
					"error_code": map[string]interface{}{
						"type": "integer",
					},
					"error_message": map[string]interface{}{
						"type": "string",
					},
					"retry_count": map[string]interface{}{
						"type":    "integer",
						"minimum": 0,
					},
					"signal_strength": map[string]interface{}{
						"type": "integer",
					},
				},
				"required": []string{"ssid", "failure_stage"},
			},
			"timestamp": map[string]interface{}{
				"type":   "string",
				"format": "date-time",
			},
		},
		Required:  []string{"event_id", "type", "level", "message", "data", "timestamp"},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})

	// ARP loss event schema
	sm.AddSchema(&Schema{
		ID:          "evt.wifi.arp_loss/1.0",
		Version:     "1.0",
		Title:       "ARP Loss Event",
		Description: "Schema for ARP packet loss events",
		Type:        "object",
		Properties: map[string]interface{}{
			"event_id": map[string]interface{}{
				"type": "string",
			},
			"type": map[string]interface{}{
				"type": "string",
				"enum": []string{"wifi.arp_loss"},
			},
			"level": map[string]interface{}{
				"type": "string",
				"enum": []string{"warning", "error"},
			},
			"message": map[string]interface{}{
				"type": "string",
			},
			"source": map[string]interface{}{
				"type": "string",
			},
			"data": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"target_ip": map[string]interface{}{
						"type":   "string",
						"format": "ipv4",
					},
					"gateway_ip": map[string]interface{}{
						"type":   "string",
						"format": "ipv4",
					},
					"interface": map[string]interface{}{
						"type": "string",
					},
					"packets_sent": map[string]interface{}{
						"type":    "integer",
						"minimum": 0,
					},
					"packets_lost": map[string]interface{}{
						"type":    "integer",
						"minimum": 0,
					},
					"loss_percentage": map[string]interface{}{
						"type":    "number",
						"minimum": 0,
						"maximum": 100,
					},
					"test_duration_ms": map[string]interface{}{
						"type":    "integer",
						"minimum": 0,
					},
				},
				"required": []string{"target_ip", "packets_sent", "packets_lost", "loss_percentage"},
			},
			"timestamp": map[string]interface{}{
				"type":   "string",
				"format": "date-time",
			},
		},
		Required:  []string{"event_id", "type", "level", "message", "data", "timestamp"},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})

	// Command message schema
	sm.AddSchema(&Schema{
		ID:          "rtk.command/1.0",
		Version:     "1.0",
		Title:       "RTK Command Message",
		Description: "Schema for command messages",
		Type:        "object",
		Properties: map[string]interface{}{
			"command_id": map[string]interface{}{
				"type":      "string",
				"minLength": 1,
			},
			"type": map[string]interface{}{
				"type":      "string",
				"minLength": 1,
			},
			"action": map[string]interface{}{
				"type":      "string",
				"minLength": 1,
			},
			"params": map[string]interface{}{
				"type": "object",
			},
			"timeout_seconds": map[string]interface{}{
				"type":    "integer",
				"minimum": 1,
			},
			"timestamp": map[string]interface{}{
				"type":   "string",
				"format": "date-time",
			},
		},
		Required:  []string{"command_id", "type", "action", "timestamp"},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})

	// Command response schema
	sm.AddSchema(&Schema{
		ID:          "rtk.command_response/1.0",
		Version:     "1.0",
		Title:       "RTK Command Response Message",
		Description: "Schema for command response messages",
		Type:        "object",
		Properties: map[string]interface{}{
			"command_id": map[string]interface{}{
				"type":      "string",
				"minLength": 1,
			},
			"request_id": map[string]interface{}{
				"type": "string",
			},
			"status": map[string]interface{}{
				"type": "string",
				"enum": []string{"success", "error", "timeout", "pending"},
			},
			"result": map[string]interface{}{
				"type": "object",
			},
			"error": map[string]interface{}{
				"type": "string",
			},
			"timestamp": map[string]interface{}{
				"type":   "string",
				"format": "date-time",
			},
			"process_time_ms": map[string]interface{}{
				"type":    "integer",
				"minimum": 0,
			},
		},
		Required:  []string{"command_id", "status", "timestamp"},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})
}

// AddSchema adds a schema to the manager
func (sm *SchemaManager) AddSchema(schema *Schema) {
	sm.schemas[schema.ID] = schema
}

// GetSchema retrieves a schema by ID
func (sm *SchemaManager) GetSchema(id string) (*Schema, bool) {
	schema, exists := sm.schemas[id]
	return schema, exists
}

// ListSchemas returns all available schemas
func (sm *SchemaManager) ListSchemas() map[string]*Schema {
	result := make(map[string]*Schema)
	for id, schema := range sm.schemas {
		result[id] = schema
	}
	return result
}

// RemoveSchema removes a schema by ID
func (sm *SchemaManager) RemoveSchema(id string) {
	delete(sm.schemas, id)
}

// ToJSONSchema converts a schema to JSON Schema format
func (s *Schema) ToJSONSchema() ([]byte, error) {
	jsonSchema := map[string]interface{}{
		"$schema":     "http://json-schema.org/draft-07/schema#",
		"$id":         s.ID,
		"title":       s.Title,
		"description": s.Description,
		"type":        s.Type,
		"properties":  s.Properties,
	}

	if len(s.Required) > 0 {
		jsonSchema["required"] = s.Required
	}

	return json.MarshalIndent(jsonSchema, "", "  ")
}

// ValidateSchema validates a schema definition
func (sm *SchemaManager) ValidateSchema(schema *Schema) error {
	if schema.ID == "" {
		return fmt.Errorf("schema ID is required")
	}

	if schema.Version == "" {
		return fmt.Errorf("schema version is required")
	}

	if schema.Type == "" {
		return fmt.Errorf("schema type is required")
	}

	if schema.Properties == nil {
		return fmt.Errorf("schema properties are required")
	}

	return nil
}

// GetSchemaForMessageType returns the appropriate schema for a message type
func (sm *SchemaManager) GetSchemaForMessageType(msgType MessageType, subType string) (*Schema, bool) {
	switch msgType {
	case MessageTypeState:
		return sm.GetSchema("rtk.state/1.0")
	case MessageTypeTelemetry:
		return sm.GetSchema("rtk.telemetry/1.0")
	case MessageTypeEvent:
		if subType != "" {
			// Try specific event schema first
			if schema, exists := sm.GetSchema(fmt.Sprintf("evt.%s/1.0", subType)); exists {
				return schema, true
			}
		}
		// Fall back to generic event schema
		return sm.GetSchema("rtk.event/1.0")
	case MessageTypeAttribute:
		return sm.GetSchema("rtk.attributes/1.0")
	case MessageTypeCommand:
		return sm.GetSchema("rtk.command/1.0")
	case MessageTypeLWT:
		return sm.GetSchema("rtk.lwt/1.0")
	default:
		return nil, false
	}
}

// ExportSchemas exports all schemas as JSON
func (sm *SchemaManager) ExportSchemas() ([]byte, error) {
	export := map[string]interface{}{
		"schemas":    sm.schemas,
		"exported_at": time.Now(),
	}

	return json.MarshalIndent(export, "", "  ")
}

// ImportSchemas imports schemas from JSON
func (sm *SchemaManager) ImportSchemas(data []byte) error {
	var importData struct {
		Schemas map[string]*Schema `json:"schemas"`
	}

	if err := json.Unmarshal(data, &importData); err != nil {
		return fmt.Errorf("failed to unmarshal schemas: %w", err)
	}

	for id, schema := range importData.Schemas {
		if err := sm.ValidateSchema(schema); err != nil {
			return fmt.Errorf("invalid schema %s: %w", id, err)
		}
		sm.AddSchema(schema)
	}

	return nil
}