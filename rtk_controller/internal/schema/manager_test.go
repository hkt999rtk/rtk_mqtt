package schema

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"rtk_controller/internal/storage"
)

// MockStorage for testing
type MockStorage struct {
	mock.Mock
}

func (m *MockStorage) Set(key, value string) error {
	args := m.Called(key, value)
	return args.Error(0)
}

func (m *MockStorage) Get(key string) (string, error) {
	args := m.Called(key)
	return args.String(0), args.Error(1)
}

func (m *MockStorage) Delete(key string) error {
	args := m.Called(key)
	return args.Error(0)
}

func (m *MockStorage) Exists(key string) (bool, error) {
	args := m.Called(key)
	return args.Bool(0), args.Error(1)
}

func (m *MockStorage) Transaction(fn func(storage.Transaction) error) error {
	args := m.Called(fn)
	return args.Error(0)
}

func (m *MockStorage) View(fn func(storage.Transaction) error) error {
	args := m.Called(fn)
	return args.Error(0)
}

func (m *MockStorage) Close() error {
	args := m.Called()
	return args.Error(0)
}

func createTestSchemaFile(t *testing.T, dir, filename string, schema map[string]interface{}) string {
	schemaPath := filepath.Join(dir, filename)
	schemaBytes, err := json.MarshalIndent(schema, "", "  ")
	require.NoError(t, err)

	err = os.WriteFile(schemaPath, schemaBytes, 0644)
	require.NoError(t, err)

	return schemaPath
}

func TestNewManager(t *testing.T) {
	mockStorage := &MockStorage{}

	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "valid config with enabled schema",
			config: Config{
				Enabled:             true,
				SchemaFiles:         []string{},
				StrictValidation:    false,
				LogValidationErrors: true,
				CacheResults:        true,
				CacheSize:           1000,
				StoreResults:        false,
			},
			wantErr: false,
		},
		{
			name: "disabled schema manager",
			config: Config{
				Enabled: false,
			},
			wantErr: false,
		},
		{
			name: "invalid cache size",
			config: Config{
				Enabled:   true,
				CacheSize: -1,
			},
			wantErr: false, // Manager should handle this gracefully
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager, err := NewManager(tt.config, mockStorage)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, manager)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, manager)
				assert.Equal(t, tt.config.Enabled, manager.config.Enabled)
				assert.Equal(t, mockStorage, manager.storage)
			}
		})
	}
}

func TestManager_Initialize(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "schema_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create test schema files
	deviceSchema := map[string]interface{}{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"type":    "object",
		"properties": map[string]interface{}{
			"device_id": map[string]interface{}{
				"type": "string",
			},
			"timestamp": map[string]interface{}{
				"type": "integer",
			},
			"status": map[string]interface{}{
				"type": "string",
				"enum": []string{"online", "offline", "error"},
			},
		},
		"required": []string{"device_id", "timestamp", "status"},
	}

	commandSchema := map[string]interface{}{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"type":    "object",
		"properties": map[string]interface{}{
			"command_id": map[string]interface{}{
				"type": "string",
			},
			"action": map[string]interface{}{
				"type": "string",
			},
			"parameters": map[string]interface{}{
				"type": "object",
			},
		},
		"required": []string{"command_id", "action"},
	}

	deviceSchemaPath := createTestSchemaFile(t, tempDir, "device.json", deviceSchema)
	commandSchemaPath := createTestSchemaFile(t, tempDir, "command.json", commandSchema)

	tests := []struct {
		name        string
		config      Config
		schemaFiles []string
		wantErr     bool
	}{
		{
			name: "successful initialization with valid schemas",
			config: Config{
				Enabled:     true,
				SchemaFiles: []string{deviceSchemaPath, commandSchemaPath},
				CacheSize:   100,
			},
			wantErr: false,
		},
		{
			name: "initialization with non-existent schema file (strict mode)",
			config: Config{
				Enabled:          true,
				StrictValidation: true, // Enable strict mode to get error
				SchemaFiles:      []string{filepath.Join(tempDir, "non_existent.json")},
			},
			wantErr: true,
		},
		{
			name: "disabled schema manager",
			config: Config{
				Enabled: false,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStorage := &MockStorage{}

			manager, err := NewManager(tt.config, mockStorage)
			require.NoError(t, err)

			err = manager.Initialize()

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				if tt.config.Enabled {
					assert.NotNil(t, manager.validator)
				}
			}
		})
	}
}

func TestManager_ValidateMessage(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "schema_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create a test schema
	deviceSchema := map[string]interface{}{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"type":    "object",
		"properties": map[string]interface{}{
			"device_id": map[string]interface{}{
				"type": "string",
			},
			"timestamp": map[string]interface{}{
				"type": "integer",
			},
			"status": map[string]interface{}{
				"type": "string",
				"enum": []string{"online", "offline", "error"},
			},
			"data": map[string]interface{}{
				"type": "object",
			},
		},
		"required": []string{"device_id", "timestamp", "status"},
	}

	schemaPath := createTestSchemaFile(t, tempDir, "device.json", deviceSchema)

	config := Config{
		Enabled:             true,
		SchemaFiles:         []string{schemaPath},
		StrictValidation:    false,
		LogValidationErrors: true,
		CacheResults:        true,
		CacheSize:           100,
		StoreResults:        false,
	}

	mockStorage := &MockStorage{}
	manager, err := NewManager(config, mockStorage)
	require.NoError(t, err)

	err = manager.Initialize()
	require.NoError(t, err)

	tests := []struct {
		name         string
		topic        string
		message      map[string]interface{}
		expectValid  bool
		expectCached bool
	}{
		{
			name:  "valid device message",
			topic: "rtk/v1/tenant1/site1/device1/state",
			message: map[string]interface{}{
				"schema":   "state/1.0",
				"ts":       1640995200,
				"health":   "ok",
				"uptime_s": 3600,
				"version":  "1.0.0",
				"components": map[string]interface{}{
					"cpu":    "ok",
					"memory": "warning",
				},
			},
			expectValid:  true,
			expectCached: true,
		},
		{
			name:  "invalid device message - missing required field",
			topic: "rtk/v1/tenant1/site1/device1/state",
			message: map[string]interface{}{
				"schema": "state/1.0",
				"ts":     1640995200,
				// missing required "health" field
			},
			expectValid:  false,
			expectCached: true,
		},
		{
			name:  "invalid device message - wrong type",
			topic: "rtk/v1/tenant1/site1/device1/state",
			message: map[string]interface{}{
				"schema": "state/1.0",
				"ts":     "invalid_timestamp", // should be integer
				"health": "ok",
			},
			expectValid:  false,
			expectCached: true,
		},
		{
			name:  "invalid device message - invalid enum value",
			topic: "rtk/v1/tenant1/site1/device1/state",
			message: map[string]interface{}{
				"schema": "state/1.0",
				"ts":     1640995200,
				"health": "invalid_status", // not in enum (ok, warning, critical, unknown)
			},
			expectValid:  false,
			expectCached: true,
		},
		{
			name:  "message for unknown topic type",
			topic: "rtk/v1/tenant1/site1/device1/unknown",
			message: map[string]interface{}{
				"some": "data",
			},
			expectValid:  true, // Should pass when no schema matches
			expectCached: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			messageBytes, err := json.Marshal(tt.message)
			require.NoError(t, err)

			result, err := manager.ValidateMessage(tt.topic, messageBytes)
			require.NoError(t, err)

			assert.Equal(t, tt.expectValid, result.Valid)

			if !tt.expectValid {
				assert.NotEmpty(t, result.Errors)
			} else {
				assert.Empty(t, result.Errors)
			}

			// Test caching if enabled
			if config.CacheResults && tt.expectCached {
				// Validate the same message again to test cache hit
				result2, err := manager.ValidateMessage(tt.topic, messageBytes)
				require.NoError(t, err)
				assert.Equal(t, result.Valid, result2.Valid)
			}
		})
	}
}

func TestManager_ValidateMessage_StrictMode(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "schema_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	deviceSchema := map[string]interface{}{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"type":    "object",
		"properties": map[string]interface{}{
			"device_id": map[string]interface{}{
				"type": "string",
			},
			"status": map[string]interface{}{
				"type": "string",
			},
		},
		"required": []string{"device_id", "status"},
	}

	schemaPath := createTestSchemaFile(t, tempDir, "device.json", deviceSchema)

	config := Config{
		Enabled:          true,
		SchemaFiles:      []string{schemaPath},
		StrictValidation: true, // Strict mode enabled
		CacheResults:     false,
	}

	mockStorage := &MockStorage{}
	manager, err := NewManager(config, mockStorage)
	require.NoError(t, err)

	err = manager.Initialize()
	require.NoError(t, err)

	tests := []struct {
		name        string
		topic       string
		message     map[string]interface{}
		expectValid bool
	}{
		{
			name:  "valid message",
			topic: "rtk/v1/tenant1/site1/device1/state",
			message: map[string]interface{}{
				"schema": "state/1.0",
				"ts":     1640995200,
				"health": "ok",
			},
			expectValid: true,
		},
		{
			name:  "unknown topic in strict mode",
			topic: "unknown/topic/format",
			message: map[string]interface{}{
				"some": "data",
			},
			expectValid: false, // Should fail in strict mode
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			messageBytes, err := json.Marshal(tt.message)
			require.NoError(t, err)

			result, err := manager.ValidateMessage(tt.topic, messageBytes)
			require.NoError(t, err)
			assert.Equal(t, tt.expectValid, result.Valid)
		})
	}
}

func TestManager_ValidateMessage_Disabled(t *testing.T) {
	config := Config{
		Enabled: false,
	}

	mockStorage := &MockStorage{}
	manager, err := NewManager(config, mockStorage)
	require.NoError(t, err)

	err = manager.Initialize()
	require.NoError(t, err)

	// When disabled, all messages should be valid
	messageBytes, err := json.Marshal(map[string]interface{}{
		"any": "data",
	})
	require.NoError(t, err)

	result, err := manager.ValidateMessage("any/topic", messageBytes)
	require.NoError(t, err)

	assert.True(t, result.Valid)
	assert.Empty(t, result.Errors)
}

func TestManager_GetValidationStats(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "schema_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	deviceSchema := map[string]interface{}{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"type":    "object",
		"properties": map[string]interface{}{
			"device_id": map[string]interface{}{
				"type": "string",
			},
		},
		"required": []string{"device_id"},
	}

	schemaPath := createTestSchemaFile(t, tempDir, "device.json", deviceSchema)

	config := Config{
		Enabled:     true,
		SchemaFiles: []string{schemaPath},
		CacheSize:   100,
	}

	mockStorage := &MockStorage{}
	manager, err := NewManager(config, mockStorage)
	require.NoError(t, err)

	err = manager.Initialize()
	require.NoError(t, err)

	// Initially, stats should be zero
	stats := manager.GetStats()
	assert.Equal(t, int64(0), stats.TotalValidations)
	assert.Equal(t, int64(0), stats.ValidMessages)
	assert.Equal(t, int64(0), stats.InvalidMessages)

	// Perform some validations
	validMessage, _ := json.Marshal(map[string]interface{}{
		"device_id": "device1",
	})
	invalidMessage, _ := json.Marshal(map[string]interface{}{
		"invalid": "message",
	})

	manager.ValidateMessage("rtk/v1/tenant1/site1/device1/state", validMessage)
	manager.ValidateMessage("rtk/v1/tenant1/site1/device1/state", invalidMessage)
	manager.ValidateMessage("rtk/v1/tenant1/site1/device1/state", validMessage) // Should be cache hit

	// Check updated stats
	stats = manager.GetStats()
	assert.Equal(t, int64(3), stats.TotalValidations)
	assert.Equal(t, int64(2), stats.ValidMessages) // First and third validation
	assert.Equal(t, int64(1), stats.InvalidMessages)
}

func TestManager_CacheValidation(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "schema_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	deviceSchema := map[string]interface{}{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"type":    "object",
		"properties": map[string]interface{}{
			"device_id": map[string]interface{}{
				"type": "string",
			},
		},
		"required": []string{"device_id"},
	}

	schemaPath := createTestSchemaFile(t, tempDir, "device.json", deviceSchema)

	config := Config{
		Enabled:      true,
		SchemaFiles:  []string{schemaPath},
		CacheResults: true,
		CacheSize:    100,
	}

	mockStorage := &MockStorage{}
	manager, err := NewManager(config, mockStorage)
	require.NoError(t, err)

	err = manager.Initialize()
	require.NoError(t, err)

	// Perform validations and verify caching behavior
	messageBytes, _ := json.Marshal(map[string]interface{}{
		"device_id": "device1",
	})

	// First validation
	result1, err := manager.ValidateMessage("rtk/v1/tenant1/site1/device1/state", messageBytes)
	require.NoError(t, err)
	assert.True(t, result1.Valid)

	// Second validation should use cache (if implemented)
	result2, err := manager.ValidateMessage("rtk/v1/tenant1/site1/device1/state", messageBytes)
	require.NoError(t, err)
	assert.True(t, result2.Valid)
	assert.Equal(t, result1.Valid, result2.Valid)
}

func TestManager_GetLoadedSchemas(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "schema_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	deviceSchema := map[string]interface{}{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"type":    "object",
		"title":   "Device State Schema",
	}

	commandSchema := map[string]interface{}{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"type":    "object",
		"title":   "Command Schema",
	}

	deviceSchemaPath := createTestSchemaFile(t, tempDir, "device.json", deviceSchema)
	commandSchemaPath := createTestSchemaFile(t, tempDir, "command.json", commandSchema)

	config := Config{
		Enabled:     true,
		SchemaFiles: []string{deviceSchemaPath, commandSchemaPath},
	}

	mockStorage := &MockStorage{}
	manager, err := NewManager(config, mockStorage)
	require.NoError(t, err)

	err = manager.Initialize()
	require.NoError(t, err)

	loadedSchemas := manager.GetLoadedSchemas()
	require.NotNil(t, loadedSchemas)

	// Should have built-in schemas plus the two loaded schemas
	assert.GreaterOrEqual(t, len(loadedSchemas), 2)
}

func TestManager_ReloadSchemas(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "schema_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create initial schema
	initialSchema := map[string]interface{}{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"type":    "object",
		"properties": map[string]interface{}{
			"device_id": map[string]interface{}{
				"type": "string",
			},
		},
		"required": []string{"device_id"},
	}

	schemaPath := createTestSchemaFile(t, tempDir, "device.json", initialSchema)

	config := Config{
		Enabled:     true,
		SchemaFiles: []string{schemaPath},
	}

	mockStorage := &MockStorage{}
	manager, err := NewManager(config, mockStorage)
	require.NoError(t, err)

	err = manager.Initialize()
	require.NoError(t, err)

	// Verify initial schema works
	validMessageBytes, _ := json.Marshal(map[string]interface{}{
		"device_id": "device1",
	})
	result, err := manager.ValidateMessage("rtk/v1/tenant1/site1/device1/state", validMessageBytes)
	require.NoError(t, err)
	assert.True(t, result.Valid)

	// Update schema file with additional required field
	updatedSchema := map[string]interface{}{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"type":    "object",
		"properties": map[string]interface{}{
			"device_id": map[string]interface{}{
				"type": "string",
			},
			"timestamp": map[string]interface{}{
				"type": "integer",
			},
		},
		"required": []string{"device_id", "timestamp"},
	}

	// Overwrite schema file
	schemaBytes, err := json.MarshalIndent(updatedSchema, "", "  ")
	require.NoError(t, err)
	err = os.WriteFile(schemaPath, schemaBytes, 0644)
	require.NoError(t, err)

	// Reload schemas
	err = manager.ReloadSchemas()
	require.NoError(t, err)

	// Same message should now be invalid due to missing timestamp
	result, err = manager.ValidateMessage("rtk/v1/tenant1/site1/device1/state", validMessageBytes)
	require.NoError(t, err)
	assert.False(t, result.Valid)

	// Message with timestamp should be valid
	messageWithTimestampBytes, _ := json.Marshal(map[string]interface{}{
		"device_id": "device1",
		"timestamp": 1640995200,
	})
	result, err = manager.ValidateMessage("rtk/v1/tenant1/site1/device1/state", messageWithTimestampBytes)
	require.NoError(t, err)
	assert.True(t, result.Valid)
}

func TestManager_ValidateJSON(t *testing.T) {
	config := Config{
		Enabled: true,
	}

	mockStorage := &MockStorage{}
	manager, err := NewManager(config, mockStorage)
	require.NoError(t, err)

	err = manager.Initialize()
	require.NoError(t, err)

	// Test validating JSON directly with built-in schemas
	validStateJSON := `{
		"schema": "state/1.0",
		"ts": 1234567890,
		"health": "ok"
	}`

	result, err := manager.ValidateJSON("state", []byte(validStateJSON))
	require.NoError(t, err)
	assert.True(t, result.Valid)
	assert.Equal(t, "state", result.Schema)

	// Test disabled manager
	disabledConfig := Config{
		Enabled: false,
	}

	disabledManager, err := NewManager(disabledConfig, mockStorage)
	require.NoError(t, err)

	result, err = disabledManager.ValidateJSON("state", []byte(validStateJSON))
	require.NoError(t, err)
	assert.True(t, result.Valid)
	assert.Equal(t, "disabled", result.Schema)
}
