package schema

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewValidator(t *testing.T) {
	validator := NewValidator()
	assert.NotNil(t, validator)
	assert.Empty(t, validator.GetLoadedSchemas())
}

func TestLoadBuiltinSchemas(t *testing.T) {
	validator := NewValidator()

	err := validator.LoadBuiltinSchemas()
	require.NoError(t, err)

	schemas := validator.GetLoadedSchemas()
	assert.Len(t, schemas, 6)

	expectedSchemas := []string{"state", "event", "command", "telemetry", "lwt", "attr"}
	for _, expected := range expectedSchemas {
		assert.Contains(t, schemas, expected)
	}
}

func TestLoadSchemaFromString(t *testing.T) {
	validator := NewValidator()

	testSchema := `{
		"type": "object",
		"properties": {
			"name": {"type": "string"},
			"age": {"type": "number"}
		},
		"required": ["name"]
	}`

	err := validator.LoadSchemaFromString("test", testSchema)
	require.NoError(t, err)

	schemas := validator.GetLoadedSchemas()
	assert.Contains(t, schemas, "test")
}

func TestLoadSchemaFromString_InvalidJSON(t *testing.T) {
	validator := NewValidator()

	invalidSchema := `{"type": "object", "properties":}`

	err := validator.LoadSchemaFromString("invalid", invalidSchema)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load schema")
}

func TestValidateStateMessage(t *testing.T) {
	validator := NewValidator()
	err := validator.LoadBuiltinSchemas()
	require.NoError(t, err)

	validState := map[string]interface{}{
		"schema":   "state/1.0",
		"ts":       1234567890,
		"health":   "ok",
		"uptime_s": 3600,
		"version":  "1.2.3",
		"components": map[string]interface{}{
			"wifi": "ok",
			"cpu":  "warning",
		},
	}

	result, err := validator.Validate("state", validState)
	require.NoError(t, err)
	assert.True(t, result.IsValid())
	assert.Empty(t, result.GetErrors())
	assert.Equal(t, "state", result.GetSchema())
}

func TestValidateStateMessage_Missing_Required(t *testing.T) {
	validator := NewValidator()
	err := validator.LoadBuiltinSchemas()
	require.NoError(t, err)

	invalidState := map[string]interface{}{
		"schema": "state/1.0",
		"ts":     1234567890,
		// Missing required "health" field
	}

	result, err := validator.Validate("state", invalidState)
	require.NoError(t, err)
	assert.False(t, result.IsValid())
	assert.NotEmpty(t, result.GetErrors())
	assert.Contains(t, result.GetErrors()[0], "health")
}

func TestValidateEventMessage(t *testing.T) {
	validator := NewValidator()
	err := validator.LoadBuiltinSchemas()
	require.NoError(t, err)

	validEvent := map[string]interface{}{
		"schema":   "evt.wifi.roam_miss/1.0",
		"ts":       1234567890,
		"severity": "warning",
		"message":  "WiFi roaming failed",
	}

	result, err := validator.Validate("event", validEvent)
	require.NoError(t, err)
	assert.True(t, result.IsValid())
	assert.Empty(t, result.GetErrors())
}

func TestValidateEventMessage_InvalidSeverity(t *testing.T) {
	validator := NewValidator()
	err := validator.LoadBuiltinSchemas()
	require.NoError(t, err)

	invalidEvent := map[string]interface{}{
		"schema":   "evt.wifi.disconnect/1.0",
		"ts":       1234567890,
		"severity": "invalid", // Invalid severity level
		"message":  "WiFi disconnected",
	}

	result, err := validator.Validate("event", invalidEvent)
	require.NoError(t, err)
	assert.False(t, result.IsValid())
	assert.NotEmpty(t, result.GetErrors())
}

func TestValidateCommandMessage(t *testing.T) {
	validator := NewValidator()
	err := validator.LoadBuiltinSchemas()
	require.NoError(t, err)

	validCommand := map[string]interface{}{
		"id":         "cmd-123",
		"op":         "reboot",
		"schema":     "cmd.system.reboot/1.0",
		"ts":         1234567890,
		"timeout_ms": 30000,
		"expect":     "ack",
		"args": map[string]interface{}{
			"force": true,
		},
	}

	result, err := validator.Validate("command", validCommand)
	require.NoError(t, err)
	assert.True(t, result.IsValid())
}

func TestValidateCommandMessage_InvalidTimeout(t *testing.T) {
	validator := NewValidator()
	err := validator.LoadBuiltinSchemas()
	require.NoError(t, err)

	invalidCommand := map[string]interface{}{
		"id":         "cmd-124",
		"op":         "test",
		"schema":     "cmd.test/1.0",
		"ts":         1234567890,
		"timeout_ms": 500, // Below minimum of 1000ms
	}

	result, err := validator.Validate("command", invalidCommand)
	require.NoError(t, err)
	assert.False(t, result.IsValid())
	assert.NotEmpty(t, result.GetErrors())
}

func TestValidateTelemetryMessage(t *testing.T) {
	validator := NewValidator()
	err := validator.LoadBuiltinSchemas()
	require.NoError(t, err)

	validTelemetry := map[string]interface{}{
		"schema": "telemetry.system/1.0",
		"ts":     1234567890,
		"metrics": map[string]interface{}{
			"cpu_percent":    45.2,
			"memory_percent": 67.8,
			"temperature":    42.5,
		},
	}

	result, err := validator.Validate("telemetry", validTelemetry)
	require.NoError(t, err)
	assert.True(t, result.IsValid())
}

func TestValidateLWTMessage(t *testing.T) {
	validator := NewValidator()
	err := validator.LoadBuiltinSchemas()
	require.NoError(t, err)

	validLWT := map[string]interface{}{
		"status": "offline",
		"ts":     1234567890,
		"reason": "network timeout",
	}

	result, err := validator.Validate("lwt", validLWT)
	require.NoError(t, err)
	assert.True(t, result.IsValid())
}

func TestValidateAttributeMessage(t *testing.T) {
	validator := NewValidator()
	err := validator.LoadBuiltinSchemas()
	require.NoError(t, err)

	validAttr := map[string]interface{}{
		"schema":      "attr/1.0",
		"ts":          1234567890,
		"device_type": "wifi_router",
		"model":       "RTL8192EU",
		"hw_version":  "1.0",
		"fw_version":  "2.3.4",
	}

	result, err := validator.Validate("attr", validAttr)
	require.NoError(t, err)
	assert.True(t, result.IsValid())
}

func TestValidateJSONString(t *testing.T) {
	validator := NewValidator()
	err := validator.LoadBuiltinSchemas()
	require.NoError(t, err)

	stateJSON := `{
		"schema": "state/1.0",
		"ts": 1234567890,
		"health": "ok",
		"uptime_s": 7200
	}`

	result, err := validator.Validate("state", stateJSON)
	require.NoError(t, err)
	assert.True(t, result.IsValid())
}

func TestValidateJSONBytes(t *testing.T) {
	validator := NewValidator()
	err := validator.LoadBuiltinSchemas()
	require.NoError(t, err)

	stateData := map[string]interface{}{
		"schema":   "state/1.0",
		"ts":       1234567890,
		"health":   "critical",
		"uptime_s": 300,
	}

	jsonBytes, err := json.Marshal(stateData)
	require.NoError(t, err)

	result, err := validator.Validate("state", jsonBytes)
	require.NoError(t, err)
	assert.True(t, result.IsValid())
}

func TestValidateInvalidJSON(t *testing.T) {
	validator := NewValidator()
	err := validator.LoadBuiltinSchemas()
	require.NoError(t, err)

	invalidJSON := `{"schema": "state/1.0", "ts": 123,`

	result, err := validator.Validate("state", invalidJSON)
	require.NoError(t, err)
	assert.False(t, result.IsValid())
	assert.Contains(t, result.GetErrors()[0], "Invalid JSON")
}

func TestValidateSchemaNotFound(t *testing.T) {
	validator := NewValidator()

	data := map[string]interface{}{"test": "data"}

	result, err := validator.Validate("nonexistent", data)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "schema not found")
}

func TestInferSchemaFromTopic(t *testing.T) {
	validator := NewValidator()

	tests := []struct {
		topic    string
		expected string
	}{
		{"rtk/v1/tenant/site/device1/state", "state"},
		{"rtk/v1/tenant/site/device1/evt/wifi", "event"},
		{"rtk/v1/tenant/site/device1/cmd/req", "command"},
		{"rtk/v1/tenant/site/device1/cmd/ack", "command"},
		{"rtk/v1/tenant/site/device1/cmd/res", "command"},
		{"rtk/v1/tenant/site/device1/lwt", "lwt"},
		{"rtk/v1/tenant/site/device1/attr", "attr"},
		{"rtk/v1/tenant/site/device1/telemetry/cpu", "telemetry"},
		{"rtk/v1/tenant/site/device1/telemetry", "telemetry"},
		{"rtk/v1/tenant/site/device1/unknown", ""},
		{"invalid/topic/format", ""},
		{"not/rtk/protocol", ""},
	}

	for _, test := range tests {
		result := validator.inferSchemaFromTopic(test.topic)
		assert.Equal(t, test.expected, result, "Topic: %s", test.topic)
	}
}

func TestValidateByTopic(t *testing.T) {
	validator := NewValidator()
	err := validator.LoadBuiltinSchemas()
	require.NoError(t, err)

	topic := "rtk/v1/test/site1/device1/state"
	payload := []byte(`{
		"schema": "state/1.0",
		"ts": 1234567890,
		"health": "ok"
	}`)

	result, err := validator.ValidateByTopic(topic, payload)
	require.NoError(t, err)
	assert.True(t, result.IsValid())
	assert.Equal(t, "state", result.GetSchema())
}

func TestValidateByTopic_UnknownSchema(t *testing.T) {
	validator := NewValidator()

	topic := "rtk/v1/test/site1/device1/unknown"
	payload := []byte(`{"test": "data"}`)

	result, err := validator.ValidateByTopic(topic, payload)
	require.NoError(t, err)
	assert.True(t, result.IsValid()) // Should skip validation
	assert.Equal(t, "unknown", result.GetSchema())
}

func TestMapSchemaFieldToName(t *testing.T) {
	validator := NewValidator()

	tests := []struct {
		schemaField string
		expected    string
	}{
		{"state/1.0", "state"},
		{"evt.wifi.roam_miss/1.0", "event"},
		{"cmd.system.reboot/1.0", "command"},
		{"lwt/1.0", "lwt"},
		{"attr/1.0", "attr"},
		{"telemetry.cpu/1.0", "telemetry"},
		{"unknown/1.0", ""},
		{"invalid", ""},
	}

	for _, test := range tests {
		result := validator.mapSchemaFieldToName(test.schemaField)
		assert.Equal(t, test.expected, result, "Schema field: %s", test.schemaField)
	}
}

func TestValidateBySchemaField(t *testing.T) {
	validator := NewValidator()
	err := validator.LoadBuiltinSchemas()
	require.NoError(t, err)

	payload := []byte(`{
		"schema": "state/1.0",
		"ts": 1234567890,
		"health": "ok"
	}`)

	result, err := validator.ValidateBySchemaField(payload)
	require.NoError(t, err)
	assert.True(t, result.IsValid())
	assert.Equal(t, "state", result.GetSchema())
}

func TestValidateBySchemaField_NoSchemaField(t *testing.T) {
	validator := NewValidator()

	payload := []byte(`{
		"ts": 1234567890,
		"data": "test"
	}`)

	result, err := validator.ValidateBySchemaField(payload)
	require.NoError(t, err)
	assert.True(t, result.IsValid()) // Should skip validation
	assert.Equal(t, "no_schema", result.GetSchema())
}

func TestValidateBySchemaField_InvalidJSON(t *testing.T) {
	validator := NewValidator()

	invalidPayload := []byte(`{"schema": "state/1.0", "ts":`)

	result, err := validator.ValidateBySchemaField(invalidPayload)
	require.NoError(t, err)
	assert.False(t, result.IsValid())
	assert.Contains(t, result.GetErrors()[0], "Invalid JSON")
}

func TestValidationResult_Methods(t *testing.T) {
	result := &ValidationResult{
		Valid:  false,
		Errors: []string{"error1", "error2"},
		Schema: "test",
		Data:   map[string]interface{}{"key": "value"},
	}

	assert.False(t, result.IsValid())
	assert.Equal(t, []string{"error1", "error2"}, result.GetErrors())
	assert.Equal(t, "test", result.GetSchema())
}

func TestConcurrentValidation(t *testing.T) {
	validator := NewValidator()
	err := validator.LoadBuiltinSchemas()
	require.NoError(t, err)

	// Test concurrent validation from multiple goroutines
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(id int) {
			stateData := map[string]interface{}{
				"schema":   "state/1.0",
				"ts":       1234567890 + id,
				"health":   "ok",
				"uptime_s": 3600 + id,
			}

			result, err := validator.Validate("state", stateData)
			assert.NoError(t, err)
			assert.True(t, result.IsValid())
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}

func BenchmarkValidateState(b *testing.B) {
	validator := NewValidator()
	err := validator.LoadBuiltinSchemas()
	require.NoError(b, err)

	stateData := map[string]interface{}{
		"schema":   "state/1.0",
		"ts":       1234567890,
		"health":   "ok",
		"uptime_s": 3600,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, err := validator.Validate("state", stateData)
		if err != nil || !result.IsValid() {
			b.Fatal("Validation failed")
		}
	}
}

func BenchmarkValidateByTopic(b *testing.B) {
	validator := NewValidator()
	err := validator.LoadBuiltinSchemas()
	require.NoError(b, err)

	topic := "rtk/v1/test/site1/device1/state"
	payload := []byte(`{
		"schema": "state/1.0",
		"ts": 1234567890,
		"health": "ok"
	}`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, err := validator.ValidateByTopic(topic, payload)
		if err != nil || !result.IsValid() {
			b.Fatal("Validation failed")
		}
	}
}
