package schema

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/xeipuuv/gojsonschema"
	log "github.com/sirupsen/logrus"
)

// Validator provides JSON schema validation functionality
type Validator struct {
	schemas map[string]*gojsonschema.Schema
	mu      sync.RWMutex
}

// ValidationResult represents the result of schema validation
type ValidationResult struct {
	Valid  bool           `json:"valid"`
	Errors []string       `json:"errors,omitempty"`
	Schema string         `json:"schema"`
	Data   interface{}    `json:"data,omitempty"`
}

// IsValid returns whether the validation passed
func (vr *ValidationResult) IsValid() bool {
	return vr.Valid
}

// GetErrors returns validation errors
func (vr *ValidationResult) GetErrors() []string {
	return vr.Errors
}

// GetSchema returns the schema name used for validation
func (vr *ValidationResult) GetSchema() string {
	return vr.Schema
}

// NewValidator creates a new schema validator
func NewValidator() *Validator {
	return &Validator{
		schemas: make(map[string]*gojsonschema.Schema),
	}
}

// LoadSchemasFromFile loads schemas from a JSON file
func (v *Validator) LoadSchemasFromFile(filePath string) error {
	log.WithField("file", filePath).Info("Loading JSON schemas from file")
	
	// Load schema file
	schemaLoader := gojsonschema.NewReferenceLoader("file://" + filePath)
	schema, err := gojsonschema.NewSchema(schemaLoader)
	if err != nil {
		return fmt.Errorf("failed to load schema file: %w", err)
	}
	
	// For now, store the main schema as "main"
	v.mu.Lock()
	v.schemas["main"] = schema
	v.mu.Unlock()
	
	log.WithField("schemas_loaded", 1).Info("JSON schemas loaded successfully")
	return nil
}

// LoadSchemaFromString loads a single schema from JSON string
func (v *Validator) LoadSchemaFromString(name, schemaJSON string) error {
	schemaLoader := gojsonschema.NewStringLoader(schemaJSON)
	schema, err := gojsonschema.NewSchema(schemaLoader)
	if err != nil {
		return fmt.Errorf("failed to load schema %s: %w", name, err)
	}
	
	v.mu.Lock()
	v.schemas[name] = schema
	v.mu.Unlock()
	
	log.WithField("schema", name).Debug("Schema loaded")
	return nil
}

// LoadBuiltinSchemas loads built-in schemas for RTK protocol
func (v *Validator) LoadBuiltinSchemas() error {
	log.Info("Loading built-in RTK protocol schemas")
	
	// Define built-in schemas
	schemas := map[string]string{
		"state": stateSchema,
		"event": eventSchema,
		"command": commandSchema,
		"telemetry": telemetrySchema,
		"lwt": lwtSchema,
		"attr": attributeSchema,
	}
	
	for name, schemaJSON := range schemas {
		if err := v.LoadSchemaFromString(name, schemaJSON); err != nil {
			return fmt.Errorf("failed to load built-in schema %s: %w", name, err)
		}
	}
	
	log.WithField("schemas_loaded", len(schemas)).Info("Built-in schemas loaded successfully")
	return nil
}

// Validate validates JSON data against a schema
func (v *Validator) Validate(schemaName string, data interface{}) (*ValidationResult, error) {
	v.mu.RLock()
	schema, exists := v.schemas[schemaName]
	v.mu.RUnlock()
	
	if !exists {
		return nil, fmt.Errorf("schema not found: %s", schemaName)
	}
	
	// Convert data to JSON if it's not already a string
	var jsonData interface{}
	switch d := data.(type) {
	case string:
		// Parse JSON string
		if err := json.Unmarshal([]byte(d), &jsonData); err != nil {
			return &ValidationResult{
				Valid:  false,
				Errors: []string{fmt.Sprintf("Invalid JSON: %v", err)},
				Schema: schemaName,
			}, nil
		}
	case []byte:
		// Parse JSON bytes
		if err := json.Unmarshal(d, &jsonData); err != nil {
			return &ValidationResult{
				Valid:  false,
				Errors: []string{fmt.Sprintf("Invalid JSON: %v", err)},
				Schema: schemaName,
			}, nil
		}
	default:
		jsonData = data
	}
	
	// Create document loader
	documentLoader := gojsonschema.NewGoLoader(jsonData)
	
	// Validate
	result, err := schema.Validate(documentLoader)
	if err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}
	
	// Build result
	validationResult := &ValidationResult{
		Valid:  result.Valid(),
		Schema: schemaName,
		Data:   jsonData,
	}
	
	if !result.Valid() {
		for _, desc := range result.Errors() {
			validationResult.Errors = append(validationResult.Errors, desc.String())
		}
	}
	
	return validationResult, nil
}

// ValidateByTopic determines schema from MQTT topic and validates
func (v *Validator) ValidateByTopic(topic string, payload []byte) (*ValidationResult, error) {
	schemaName := v.inferSchemaFromTopic(topic)
	if schemaName == "" {
		return &ValidationResult{
			Valid:  true, // Skip validation for unknown schemas
			Schema: "unknown",
		}, nil
	}
	
	return v.Validate(schemaName, payload)
}

// ValidateBySchemaField validates based on schema field in JSON payload
func (v *Validator) ValidateBySchemaField(payload []byte) (*ValidationResult, error) {
	// Parse payload to extract schema field
	var data map[string]interface{}
	if err := json.Unmarshal(payload, &data); err != nil {
		return &ValidationResult{
			Valid:  false,
			Errors: []string{fmt.Sprintf("Invalid JSON: %v", err)},
			Schema: "unknown",
		}, nil
	}
	
	// Extract schema field
	schemaField, ok := data["schema"].(string)
	if !ok {
		return &ValidationResult{
			Valid:  true, // Skip validation if no schema field
			Schema: "no_schema",
		}, nil
	}
	
	// Map schema field to schema name
	schemaName := v.mapSchemaFieldToName(schemaField)
	if schemaName == "" {
		return &ValidationResult{
			Valid:  true, // Skip validation for unknown schemas
			Schema: schemaField,
		}, nil
	}
	
	return v.Validate(schemaName, data)
}

// GetLoadedSchemas returns list of loaded schema names
func (v *Validator) GetLoadedSchemas() []string {
	v.mu.RLock()
	defer v.mu.RUnlock()
	
	var schemas []string
	for name := range v.schemas {
		schemas = append(schemas, name)
	}
	return schemas
}

// inferSchemaFromTopic determines appropriate schema based on MQTT topic
func (v *Validator) inferSchemaFromTopic(topic string) string {
	parts := strings.Split(topic, "/")
	if len(parts) < 6 {
		return ""
	}
	
	// Topic format: rtk/v1/{tenant}/{site}/{device_id}/{message_type}/...
	if parts[0] != "rtk" || parts[1] != "v1" {
		return ""
	}
	
	messageType := parts[5]
	
	switch messageType {
	case "state":
		return "state"
	case "evt":
		return "event"
	case "cmd":
		if len(parts) > 6 {
			switch parts[6] {
			case "req":
				return "command"
			case "ack", "res":
				return "command"
			}
		}
		return "command"
	case "lwt":
		return "lwt"
	case "attr":
		return "attr"
	case "telemetry":
		return "telemetry"
	default:
		if strings.HasPrefix(messageType, "telemetry") {
			return "telemetry"
		}
		return ""
	}
}

// mapSchemaFieldToName maps schema field value to internal schema name
func (v *Validator) mapSchemaFieldToName(schemaField string) string {
	// Handle versioned schema fields like "state/1.0", "evt.wifi.roam_miss/1.0"
	parts := strings.Split(schemaField, "/")
	if len(parts) < 1 {
		return ""
	}
	
	schemaType := parts[0]
	
	// Extract base type from dotted schemas
	if strings.Contains(schemaType, ".") {
		typeParts := strings.Split(schemaType, ".")
		schemaType = typeParts[0]
	}
	
	switch schemaType {
	case "state":
		return "state"
	case "evt":
		return "event"
	case "cmd":
		return "command"
	case "lwt":
		return "lwt"
	case "attr":
		return "attr"
	case "telemetry":
		return "telemetry"
	default:
		return ""
	}
}

// Built-in schema definitions
const stateSchema = `{
  "type": "object",
  "properties": {
    "schema": {
      "type": "string",
      "pattern": "^state/[0-9]+\\.[0-9]+$"
    },
    "ts": {
      "type": "integer",
      "description": "Unix timestamp in milliseconds"
    },
    "health": {
      "type": "string",
      "enum": ["ok", "warning", "critical", "unknown"]
    },
    "uptime_s": {
      "type": "integer",
      "minimum": 0,
      "description": "Device uptime in seconds"
    },
    "version": {
      "type": "string",
      "description": "Device firmware/software version"
    },
    "components": {
      "type": "object",
      "description": "Component status map"
    }
  },
  "required": ["schema", "ts", "health"]
}`

const eventSchema = `{
  "type": "object",
  "properties": {
    "schema": {
      "type": "string",
      "pattern": "^evt\\."
    },
    "ts": {
      "type": "integer",
      "description": "Unix timestamp in milliseconds"
    },
    "severity": {
      "type": "string",
      "enum": ["info", "warning", "error", "critical"]
    },
    "message": {
      "type": "string",
      "description": "Human readable event message"
    }
  },
  "required": ["schema", "ts", "severity"]
}`

const commandSchema = `{
  "type": "object",
  "properties": {
    "id": {
      "type": "string",
      "description": "Unique command identifier"
    },
    "op": {
      "type": "string",
      "description": "Operation name"
    },
    "schema": {
      "type": "string",
      "pattern": "^cmd\\."
    },
    "args": {
      "type": "object",
      "description": "Command arguments"
    },
    "timeout_ms": {
      "type": "integer",
      "minimum": 1000,
      "maximum": 300000,
      "description": "Command timeout in milliseconds"
    },
    "expect": {
      "type": "string",
      "enum": ["ack", "result", "none"]
    },
    "ts": {
      "type": "integer",
      "description": "Unix timestamp in milliseconds"
    }
  },
  "required": ["id", "op", "schema", "ts"]
}`

const telemetrySchema = `{
  "type": "object",
  "properties": {
    "schema": {
      "type": "string",
      "pattern": "^telemetry\\."
    },
    "ts": {
      "type": "integer",
      "description": "Unix timestamp in milliseconds"
    },
    "metrics": {
      "type": "object",
      "description": "Telemetry metrics data"
    }
  },
  "required": ["schema", "ts"]
}`

const lwtSchema = `{
  "type": "object",
  "properties": {
    "status": {
      "type": "string",
      "enum": ["online", "offline"]
    },
    "ts": {
      "type": "integer",
      "description": "Unix timestamp in milliseconds"
    },
    "reason": {
      "type": "string",
      "description": "Reason for status change"
    }
  },
  "required": ["status", "ts"]
}`

const attributeSchema = `{
  "type": "object",
  "properties": {
    "schema": {
      "type": "string",
      "pattern": "^attr/[0-9]+\\.[0-9]+$"
    },
    "ts": {
      "type": "integer",
      "description": "Unix timestamp in milliseconds"
    },
    "device_type": {
      "type": "string",
      "description": "Type of device"
    },
    "model": {
      "type": "string",
      "description": "Device model"
    },
    "hw_version": {
      "type": "string",
      "description": "Hardware version"
    },
    "fw_version": {
      "type": "string", 
      "description": "Firmware version"
    }
  },
  "required": ["schema", "ts"]
}`