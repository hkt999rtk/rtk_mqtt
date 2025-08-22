package mqtt

import "rtk_controller/internal/schema"

// SchemaValidatorAdapter adapts schema.Manager to implement SchemaValidator interface
type SchemaValidatorAdapter struct {
	manager *schema.Manager
}

// NewSchemaValidatorAdapter creates a new adapter
func NewSchemaValidatorAdapter(manager *schema.Manager) *SchemaValidatorAdapter {
	return &SchemaValidatorAdapter{manager: manager}
}

// ValidateMessage adapts schema manager's validation to MQTT interface
func (s *SchemaValidatorAdapter) ValidateMessage(topic string, payload []byte) (*ValidationResult, error) {
	result, err := s.manager.ValidateMessage(topic, payload)
	if err != nil {
		return nil, err
	}

	// Convert schema.ValidationResult to mqtt.ValidationResult
	return &ValidationResult{
		Valid:  result.Valid,
		Errors: result.Errors,
		Schema: result.Schema,
	}, nil
}
