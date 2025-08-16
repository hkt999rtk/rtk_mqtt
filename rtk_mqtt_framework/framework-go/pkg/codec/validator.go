package codec

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/xeipuuv/gojsonschema"
)

// Validator handles message validation
type Validator struct {
	structValidator *validator.Validate
	schemas         map[string]*gojsonschema.Schema
}

// ValidationError represents a validation error
type ValidationError struct {
	Field   string `json:"field"`
	Tag     string `json:"tag"`
	Value   string `json:"value"`
	Message string `json:"message"`
}

// ValidationResult contains validation results
type ValidationResult struct {
	Valid    bool               `json:"valid"`
	Errors   []*ValidationError `json:"errors,omitempty"`
	Warnings []string           `json:"warnings,omitempty"`
}

// NewValidator creates a new message validator
func NewValidator() *Validator {
	v := &Validator{
		structValidator: validator.New(),
		schemas:         make(map[string]*gojsonschema.Schema),
	}

	// Register custom validators
	v.registerCustomValidators()

	return v
}

// registerCustomValidators registers custom validation functions
func (v *Validator) registerCustomValidators() {
	// Register device status validator
	v.structValidator.RegisterValidation("device_status", func(fl validator.FieldLevel) bool {
		status := fl.Field().String()
		validStatuses := []string{"online", "offline", "error", "maintenance"}
		for _, validStatus := range validStatuses {
			if status == validStatus {
				return true
			}
		}
		return false
	})

	// Register health status validator
	v.structValidator.RegisterValidation("health_status", func(fl validator.FieldLevel) bool {
		health := fl.Field().String()
		validHealths := []string{"healthy", "warning", "critical", "unknown"}
		for _, validHealth := range validHealths {
			if health == validHealth {
				return true
			}
		}
		return false
	})

	// Register event level validator
	v.structValidator.RegisterValidation("event_level", func(fl validator.FieldLevel) bool {
		level := fl.Field().String()
		validLevels := []string{"info", "warning", "error", "critical"}
		for _, validLevel := range validLevels {
			if level == validLevel {
				return true
			}
		}
		return false
	})

	// Register command status validator
	v.structValidator.RegisterValidation("command_status", func(fl validator.FieldLevel) bool {
		status := fl.Field().String()
		validStatuses := []string{"success", "error", "timeout", "pending"}
		for _, validStatus := range validStatuses {
			if status == validStatus {
				return true
			}
		}
		return false
	})
}

// ValidateRTKMessage validates an RTK message structure
func (v *Validator) ValidateRTKMessage(ctx context.Context, msg *RTKMessage) *ValidationResult {
	result := &ValidationResult{Valid: true}

	// Basic structure validation
	if msg.MessageID == "" {
		result.addError("message_id", "required", "", "Message ID is required")
	}

	if msg.MessageType == "" {
		result.addError("message_type", "required", "", "Message type is required")
	}

	if msg.DeviceID == "" {
		result.addError("device_id", "required", "", "Device ID is required")
	}

	if msg.Topic == "" {
		result.addError("topic", "required", "", "Topic is required")
	}

	if msg.Timestamp.IsZero() {
		result.addError("timestamp", "required", "", "Timestamp is required")
	}

	// Validate timestamp is not too far in the future or past
	now := time.Now()
	if msg.Timestamp.After(now.Add(5 * time.Minute)) {
		result.addWarning("Timestamp is more than 5 minutes in the future")
	}
	if msg.Timestamp.Before(now.Add(-24 * time.Hour)) {
		result.addWarning("Timestamp is more than 24 hours in the past")
	}

	// Validate payload is valid JSON
	if len(msg.Payload) > 0 {
		var payload interface{}
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			result.addError("payload", "json", string(msg.Payload), "Payload must be valid JSON")
		}
	}

	// Message type specific validation
	switch msg.MessageType {
	case MessageTypeState:
		v.validateStateMessage(msg, result)
	case MessageTypeTelemetry:
		v.validateTelemetryMessage(msg, result)
	case MessageTypeEvent:
		v.validateEventMessage(msg, result)
	case MessageTypeAttribute:
		v.validateAttributeMessage(msg, result)
	case MessageTypeCommand:
		v.validateCommandMessage(msg, result)
	case MessageTypeLWT:
		v.validateLWTMessage(msg, result)
	default:
		result.addError("message_type", "invalid", string(msg.MessageType), "Unknown message type")
	}

	return result
}

// validateStateMessage validates a state message
func (v *Validator) validateStateMessage(msg *RTKMessage, result *ValidationResult) {
	var stateMsg StateMessage
	if err := json.Unmarshal(msg.Payload, &stateMsg); err != nil {
		result.addError("payload", "unmarshal", "", "Failed to unmarshal state message")
		return
	}

	// Validate using struct tags
	if err := v.structValidator.Struct(&stateMsg); err != nil {
		v.addValidationErrors(err, result)
	}

	// Additional validation
	if stateMsg.Status == "" {
		result.addError("status", "required", "", "Status is required")
	}

	if stateMsg.Health == "" {
		result.addError("health", "required", "", "Health is required")
	}

	if stateMsg.Timestamp.IsZero() {
		result.addError("timestamp", "required", "", "Timestamp is required")
	}
}

// validateTelemetryMessage validates a telemetry message
func (v *Validator) validateTelemetryMessage(msg *RTKMessage, result *ValidationResult) {
	var telemetryMsg TelemetryMessage
	if err := json.Unmarshal(msg.Payload, &telemetryMsg); err != nil {
		result.addError("payload", "unmarshal", "", "Failed to unmarshal telemetry message")
		return
	}

	// Validate using struct tags
	if err := v.structValidator.Struct(&telemetryMsg); err != nil {
		v.addValidationErrors(err, result)
	}

	// Additional validation
	if telemetryMsg.Metric == "" {
		result.addError("metric", "required", "", "Metric name is required")
	}

	if telemetryMsg.Value == nil {
		result.addError("value", "required", "", "Metric value is required")
	}

	if telemetryMsg.Timestamp.IsZero() {
		result.addError("timestamp", "required", "", "Timestamp is required")
	}
}

// validateEventMessage validates an event message
func (v *Validator) validateEventMessage(msg *RTKMessage, result *ValidationResult) {
	var eventMsg EventMessage
	if err := json.Unmarshal(msg.Payload, &eventMsg); err != nil {
		result.addError("payload", "unmarshal", "", "Failed to unmarshal event message")
		return
	}

	// Validate using struct tags
	if err := v.structValidator.Struct(&eventMsg); err != nil {
		v.addValidationErrors(err, result)
	}

	// Additional validation
	if eventMsg.EventID == "" {
		result.addError("event_id", "required", "", "Event ID is required")
	}

	if eventMsg.Type == "" {
		result.addError("type", "required", "", "Event type is required")
	}

	if eventMsg.Level == "" {
		result.addError("level", "required", "", "Event level is required")
	}

	if eventMsg.Message == "" {
		result.addError("message", "required", "", "Event message is required")
	}

	if eventMsg.Timestamp.IsZero() {
		result.addError("timestamp", "required", "", "Timestamp is required")
	}

	// Validate schema ID if provided
	if eventMsg.SchemaID != "" && msg.SchemaID != eventMsg.SchemaID {
		result.addWarning("Schema ID mismatch between message header and payload")
	}
}

// validateAttributeMessage validates an attribute message
func (v *Validator) validateAttributeMessage(msg *RTKMessage, result *ValidationResult) {
	var attrMsg AttributeMessage
	if err := json.Unmarshal(msg.Payload, &attrMsg); err != nil {
		result.addError("payload", "unmarshal", "", "Failed to unmarshal attribute message")
		return
	}

	// Validate using struct tags
	if err := v.structValidator.Struct(&attrMsg); err != nil {
		v.addValidationErrors(err, result)
	}

	// Additional validation
	if attrMsg.Attributes == nil || len(attrMsg.Attributes) == 0 {
		result.addError("attributes", "required", "", "Attributes are required")
	}

	if attrMsg.Timestamp.IsZero() {
		result.addError("timestamp", "required", "", "Timestamp is required")
	}
}

// validateCommandMessage validates a command message
func (v *Validator) validateCommandMessage(msg *RTKMessage, result *ValidationResult) {
	// Check if it's a command request or response based on topic
	if strings.Contains(msg.Topic, "/cmd/req") {
		v.validateCommandRequest(msg, result)
	} else if strings.Contains(msg.Topic, "/cmd/res") || strings.Contains(msg.Topic, "/cmd/ack") {
		v.validateCommandResponse(msg, result)
	} else {
		result.addError("topic", "invalid", msg.Topic, "Unknown command topic type")
	}
}

// validateCommandRequest validates a command request
func (v *Validator) validateCommandRequest(msg *RTKMessage, result *ValidationResult) {
	var cmdMsg CommandMessage
	if err := json.Unmarshal(msg.Payload, &cmdMsg); err != nil {
		result.addError("payload", "unmarshal", "", "Failed to unmarshal command message")
		return
	}

	// Validate using struct tags
	if err := v.structValidator.Struct(&cmdMsg); err != nil {
		v.addValidationErrors(err, result)
	}

	// Additional validation
	if cmdMsg.CommandID == "" {
		result.addError("command_id", "required", "", "Command ID is required")
	}

	if cmdMsg.Type == "" {
		result.addError("type", "required", "", "Command type is required")
	}

	if cmdMsg.Action == "" {
		result.addError("action", "required", "", "Command action is required")
	}

	if cmdMsg.Timestamp.IsZero() {
		result.addError("timestamp", "required", "", "Timestamp is required")
	}
}

// validateCommandResponse validates a command response
func (v *Validator) validateCommandResponse(msg *RTKMessage, result *ValidationResult) {
	var responseMsg CommandResponseMessage
	if err := json.Unmarshal(msg.Payload, &responseMsg); err != nil {
		result.addError("payload", "unmarshal", "", "Failed to unmarshal command response")
		return
	}

	// Validate using struct tags
	if err := v.structValidator.Struct(&responseMsg); err != nil {
		v.addValidationErrors(err, result)
	}

	// Additional validation
	if responseMsg.CommandID == "" {
		result.addError("command_id", "required", "", "Command ID is required")
	}

	if responseMsg.Status == "" {
		result.addError("status", "required", "", "Response status is required")
	}

	if responseMsg.Timestamp.IsZero() {
		result.addError("timestamp", "required", "", "Timestamp is required")
	}

	// Validate status-specific requirements
	if responseMsg.Status == "error" && responseMsg.Error == "" {
		result.addWarning("Error status should include an error message")
	}
}

// validateLWTMessage validates a Last Will Testament message
func (v *Validator) validateLWTMessage(msg *RTKMessage, result *ValidationResult) {
	var lwtMsg LWTMessage
	if err := json.Unmarshal(msg.Payload, &lwtMsg); err != nil {
		result.addError("payload", "unmarshal", "", "Failed to unmarshal LWT message")
		return
	}

	// Validate using struct tags
	if err := v.structValidator.Struct(&lwtMsg); err != nil {
		v.addValidationErrors(err, result)
	}

	// Additional validation
	if lwtMsg.Status == "" {
		result.addError("status", "required", "", "LWT status is required")
	}

	if lwtMsg.Timestamp.IsZero() {
		result.addError("timestamp", "required", "", "Timestamp is required")
	}
}

// LoadSchema loads a JSON schema for validation
func (v *Validator) LoadSchema(schemaID string, schemaData []byte) error {
	schemaLoader := gojsonschema.NewBytesLoader(schemaData)
	schema, err := gojsonschema.NewSchema(schemaLoader)
	if err != nil {
		return fmt.Errorf("failed to load schema %s: %w", schemaID, err)
	}

	v.schemas[schemaID] = schema
	return nil
}

// ValidateWithSchema validates a message against a JSON schema
func (v *Validator) ValidateWithSchema(ctx context.Context, msg *RTKMessage) *ValidationResult {
	result := &ValidationResult{Valid: true}

	if msg.SchemaID == "" {
		result.addWarning("No schema ID provided for validation")
		return result
	}

	schema, exists := v.schemas[msg.SchemaID]
	if !exists {
		result.addError("schema", "not_found", msg.SchemaID, "Schema not found")
		return result
	}

	payloadLoader := gojsonschema.NewBytesLoader(msg.Payload)
	validationResult, err := schema.Validate(payloadLoader)
	if err != nil {
		result.addError("schema", "validation_error", "", err.Error())
		return result
	}

	if !validationResult.Valid() {
		for _, err := range validationResult.Errors() {
			result.addError(err.Field(), "schema", err.Value().(string), err.Description())
		}
	}

	return result
}

// addValidationErrors converts validator errors to ValidationError
func (v *Validator) addValidationErrors(err error, result *ValidationResult) {
	if validationErrors, ok := err.(validator.ValidationErrors); ok {
		for _, fieldErr := range validationErrors {
			result.addError(
				fieldErr.Field(),
				fieldErr.Tag(),
				fmt.Sprintf("%v", fieldErr.Value()),
				v.getErrorMessage(fieldErr),
			)
		}
	}
}

// getErrorMessage returns a human-readable error message
func (v *Validator) getErrorMessage(fieldErr validator.FieldError) string {
	switch fieldErr.Tag() {
	case "required":
		return fmt.Sprintf("%s is required", fieldErr.Field())
	case "device_status":
		return "Status must be one of: online, offline, error, maintenance"
	case "health_status":
		return "Health must be one of: healthy, warning, critical, unknown"
	case "event_level":
		return "Level must be one of: info, warning, error, critical"
	case "command_status":
		return "Status must be one of: success, error, timeout, pending"
	default:
		return fmt.Sprintf("%s failed validation: %s", fieldErr.Field(), fieldErr.Tag())
	}
}

// Helper methods for ValidationResult

func (r *ValidationResult) addError(field, tag, value, message string) {
	r.Valid = false
	r.Errors = append(r.Errors, &ValidationError{
		Field:   field,
		Tag:     tag,
		Value:   value,
		Message: message,
	})
}

func (r *ValidationResult) addWarning(message string) {
	r.Warnings = append(r.Warnings, message)
}

// HasErrors returns true if validation has errors
func (r *ValidationResult) HasErrors() bool {
	return len(r.Errors) > 0
}

// HasWarnings returns true if validation has warnings
func (r *ValidationResult) HasWarnings() bool {
	return len(r.Warnings) > 0
}

// GetErrorMessages returns all error messages
func (r *ValidationResult) GetErrorMessages() []string {
	messages := make([]string, len(r.Errors))
	for i, err := range r.Errors {
		messages[i] = err.Message
	}
	return messages
}