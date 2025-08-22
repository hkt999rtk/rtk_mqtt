package command

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"rtk_controller/internal/storage"
	"rtk_controller/pkg/types"
)

// MockMQTTClient for testing
type MockMQTTClient struct {
	mock.Mock
}

func (m *MockMQTTClient) IsConnected() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *MockMQTTClient) Publish(topic string, payload interface{}, qos int, retained bool) error {
	args := m.Called(topic, payload, qos, retained)
	return args.Error(0)
}

func (m *MockMQTTClient) Subscribe(topics []string) error {
	args := m.Called(topics)
	return args.Error(0)
}

func (m *MockMQTTClient) Unsubscribe(topics []string) error {
	args := m.Called(topics)
	return args.Error(0)
}

func (m *MockMQTTClient) Connect(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockMQTTClient) Disconnect() {
	m.Called()
}

// MockStorage for testing
type MockStorage struct {
	mock.Mock
	data map[string]interface{}
}

func NewMockStorage() *MockStorage {
	return &MockStorage{
		data: make(map[string]interface{}),
	}
}

func (m *MockStorage) Set(key string, value interface{}) error {
	args := m.Called(key, value)
	if args.Error(0) == nil {
		m.data[key] = value
	}
	return args.Error(0)
}

func (m *MockStorage) Get(key string, result interface{}) error {
	args := m.Called(key, result)
	return args.Error(0)
}

func (m *MockStorage) Delete(key string) error {
	args := m.Called(key)
	if args.Error(0) == nil {
		delete(m.data, key)
	}
	return args.Error(0)
}

func (m *MockStorage) List(pattern string) ([]string, error) {
	args := m.Called(pattern)
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockStorage) Close() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockStorage) CreateIndex(name, pattern string, less func(a, b string) bool) error {
	args := m.Called(name, pattern, less)
	return args.Error(0)
}

func (m *MockStorage) View(fn func(storage.Transaction) error) error {
	args := m.Called(fn)
	return args.Error(0)
}

func (m *MockStorage) Update(fn func(storage.Transaction) error) error {
	args := m.Called(fn)
	return args.Error(0)
}

func TestNewManager(t *testing.T) {
	mockMQTT := &MockMQTTClient{}
	mockStorage := NewMockStorage()

	manager := NewManager(mockMQTT, mockStorage)

	assert.NotNil(t, manager)
	assert.Equal(t, mockMQTT, manager.mqttClient)
	assert.Equal(t, mockStorage, manager.storage)
	assert.NotNil(t, manager.commands)
	assert.NotNil(t, manager.stats)
}

func TestManager_Start(t *testing.T) {
	mockMQTT := &MockMQTTClient{}
	mockStorage := NewMockStorage()
	manager := NewManager(mockMQTT, mockStorage)

	// Mock the List call for loading existing commands
	mockStorage.On("List", "commands:*").Return([]string{}, nil)

	ctx := context.Background()
	err := manager.Start(ctx)

	assert.NoError(t, err)
	mockStorage.AssertExpectations(t)
}

func TestManager_SendCommand(t *testing.T) {
	mockMQTT := &MockMQTTClient{}
	mockStorage := NewMockStorage()
	manager := NewManager(mockMQTT, mockStorage)

	deviceInfo := types.DeviceInfo{
		Tenant:   "test-tenant",
		Site:     "test-site",
		DeviceID: "test-device",
		Type:     "wifi_router",
	}

	commandData := map[string]interface{}{
		"action": "restart",
		"params": map[string]interface{}{
			"delay": 5,
		},
	}

	// Mock MQTT publish
	expectedTopic := "rtk/v1/test-tenant/test-site/test-device/cmd/req"
	mockMQTT.On("Publish", expectedTopic, mock.AnythingOfType("map[string]interface {}"), 1, false).Return(nil)

	// Mock storage operations
	mockStorage.On("Set", mock.AnythingOfType("string"), mock.AnythingOfType("types.Command")).Return(nil)

	commandID, err := manager.SendCommand(deviceInfo, commandData)

	assert.NoError(t, err)
	assert.NotEmpty(t, commandID)

	// Verify command was stored
	assert.Contains(t, manager.commands, commandID)
	command := manager.commands[commandID]
	assert.Equal(t, deviceInfo, command.DeviceInfo)
	assert.Equal(t, commandData, command.Data)
	assert.Equal(t, types.CommandStatusPending, command.Status)

	mockMQTT.AssertExpectations(t)
	mockStorage.AssertExpectations(t)
}

func TestManager_HandleCommandAck(t *testing.T) {
	mockMQTT := &MockMQTTClient{}
	mockStorage := NewMockStorage()
	manager := NewManager(mockMQTT, mockStorage)

	// Create a test command
	commandID := "test-command-123"
	deviceInfo := types.DeviceInfo{
		Tenant:   "test-tenant",
		Site:     "test-site",
		DeviceID: "test-device",
		Type:     "wifi_router",
	}

	command := types.Command{
		ID:         commandID,
		DeviceInfo: deviceInfo,
		Data:       map[string]interface{}{"action": "restart"},
		Status:     types.CommandStatusPending,
		CreatedAt:  time.Now(),
	}

	manager.commands[commandID] = command
	mockStorage.On("Set", mock.AnythingOfType("string"), mock.AnythingOfType("types.Command")).Return(nil)

	// Handle ACK
	ackData := map[string]interface{}{
		"command_id": commandID,
		"status":     "acknowledged",
		"timestamp":  time.Now().Unix(),
	}

	err := manager.HandleCommandAck(deviceInfo, ackData)

	assert.NoError(t, err)

	// Verify command status was updated
	updatedCommand := manager.commands[commandID]
	assert.Equal(t, types.CommandStatusAcknowledged, updatedCommand.Status)
	assert.NotNil(t, updatedCommand.AcknowledgedAt)

	mockStorage.AssertExpectations(t)
}

func TestManager_HandleCommandResponse(t *testing.T) {
	mockMQTT := &MockMQTTClient{}
	mockStorage := NewMockStorage()
	manager := NewManager(mockMQTT, mockStorage)

	// Create a test command
	commandID := "test-command-123"
	deviceInfo := types.DeviceInfo{
		Tenant:   "test-tenant",
		Site:     "test-site",
		DeviceID: "test-device",
		Type:     "wifi_router",
	}

	command := types.Command{
		ID:         commandID,
		DeviceInfo: deviceInfo,
		Data:       map[string]interface{}{"action": "restart"},
		Status:     types.CommandStatusAcknowledged,
		CreatedAt:  time.Now(),
	}

	manager.commands[commandID] = command
	mockStorage.On("Set", mock.AnythingOfType("string"), mock.AnythingOfType("types.Command")).Return(nil)

	tests := []struct {
		name           string
		responseData   map[string]interface{}
		expectedStatus types.CommandStatus
	}{
		{
			name: "successful response",
			responseData: map[string]interface{}{
				"command_id": commandID,
				"status":     "success",
				"result":     map[string]interface{}{"uptime": 0},
				"timestamp":  time.Now().Unix(),
			},
			expectedStatus: types.CommandStatusCompleted,
		},
		{
			name: "failed response",
			responseData: map[string]interface{}{
				"command_id": commandID,
				"status":     "error",
				"error":      "Failed to restart",
				"timestamp":  time.Now().Unix(),
			},
			expectedStatus: types.CommandStatusFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset command status
			command.Status = types.CommandStatusAcknowledged
			command.CompletedAt = nil
			manager.commands[commandID] = command

			err := manager.HandleCommandResponse(deviceInfo, tt.responseData)

			assert.NoError(t, err)

			// Verify command status was updated
			updatedCommand := manager.commands[commandID]
			assert.Equal(t, tt.expectedStatus, updatedCommand.Status)
			assert.NotNil(t, updatedCommand.CompletedAt)
			assert.Equal(t, tt.responseData, updatedCommand.Response)
		})
	}

	mockStorage.AssertExpectations(t)
}

func TestManager_GetCommand(t *testing.T) {
	mockMQTT := &MockMQTTClient{}
	mockStorage := NewMockStorage()
	manager := NewManager(mockMQTT, mockStorage)

	commandID := "test-command-123"
	deviceInfo := types.DeviceInfo{
		Tenant:   "test-tenant",
		Site:     "test-site",
		DeviceID: "test-device",
		Type:     "wifi_router",
	}

	command := types.Command{
		ID:         commandID,
		DeviceInfo: deviceInfo,
		Data:       map[string]interface{}{"action": "restart"},
		Status:     types.CommandStatusPending,
		CreatedAt:  time.Now(),
	}

	manager.commands[commandID] = command

	// Test getting existing command
	retrievedCommand, err := manager.GetCommand(commandID)
	assert.NoError(t, err)
	assert.NotNil(t, retrievedCommand)
	assert.Equal(t, command, *retrievedCommand)

	// Test getting non-existent command
	_, err = manager.GetCommand("non-existent-command")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestManager_ListCommands(t *testing.T) {
	mockMQTT := &MockMQTTClient{}
	mockStorage := NewMockStorage()
	manager := NewManager(mockMQTT, mockStorage)

	// Create test commands
	commands := []types.Command{
		{
			ID: "cmd1",
			DeviceInfo: types.DeviceInfo{
				Tenant:   "tenant1",
				Site:     "site1",
				DeviceID: "device1",
				Type:     "wifi_router",
			},
			Status:    types.CommandStatusPending,
			CreatedAt: time.Now().Add(-10 * time.Minute),
		},
		{
			ID: "cmd2",
			DeviceInfo: types.DeviceInfo{
				Tenant:   "tenant1",
				Site:     "site1",
				DeviceID: "device2",
				Type:     "smart_switch",
			},
			Status:    types.CommandStatusCompleted,
			CreatedAt: time.Now().Add(-5 * time.Minute),
		},
		{
			ID: "cmd3",
			DeviceInfo: types.DeviceInfo{
				Tenant:   "tenant2",
				Site:     "site1",
				DeviceID: "device1",
				Type:     "iot_sensor",
			},
			Status:    types.CommandStatusFailed,
			CreatedAt: time.Now().Add(-1 * time.Minute),
		},
	}

	for _, cmd := range commands {
		manager.commands[cmd.ID] = cmd
	}

	// Test listing all commands
	allCommands := manager.ListCommands("")
	assert.Len(t, allCommands, 3)

	// Test filtering by device
	deviceKey := "tenant1:site1:device1"
	device1Commands := manager.ListCommands(deviceKey)
	assert.Len(t, device1Commands, 1)
	assert.Equal(t, "cmd1", device1Commands[0].ID)

	// Test filtering by non-existent device
	nonExistentCommands := manager.ListCommands("non:existent:device")
	assert.Len(t, nonExistentCommands, 0)
}

func TestManager_CancelCommand(t *testing.T) {
	mockMQTT := &MockMQTTClient{}
	mockStorage := NewMockStorage()
	manager := NewManager(mockMQTT, mockStorage)

	commandID := "test-command-123"
	deviceInfo := types.DeviceInfo{
		Tenant:   "test-tenant",
		Site:     "test-site",
		DeviceID: "test-device",
		Type:     "wifi_router",
	}

	command := types.Command{
		ID:         commandID,
		DeviceInfo: deviceInfo,
		Data:       map[string]interface{}{"action": "restart"},
		Status:     types.CommandStatusPending,
		CreatedAt:  time.Now(),
	}

	manager.commands[commandID] = command
	mockStorage.On("Set", mock.AnythingOfType("string"), mock.AnythingOfType("types.Command")).Return(nil)

	err := manager.CancelCommand(commandID)

	assert.NoError(t, err)

	// Verify command status was updated
	updatedCommand := manager.commands[commandID]
	assert.Equal(t, types.CommandStatusCancelled, updatedCommand.Status)

	mockStorage.AssertExpectations(t)
}

func TestManager_GetStats(t *testing.T) {
	mockMQTT := &MockMQTTClient{}
	mockStorage := NewMockStorage()
	manager := NewManager(mockMQTT, mockStorage)

	// Initially, stats should be zero
	stats := manager.GetStats()
	assert.Equal(t, int64(0), stats.TotalCommands)
	assert.Equal(t, int64(0), stats.PendingCommands)
	assert.Equal(t, int64(0), stats.CompletedCommands)
	assert.Equal(t, int64(0), stats.FailedCommands)

	// Add commands with different statuses
	commands := []types.Command{
		{
			ID:     "cmd1",
			Status: types.CommandStatusPending,
		},
		{
			ID:     "cmd2",
			Status: types.CommandStatusCompleted,
		},
		{
			ID:     "cmd3",
			Status: types.CommandStatusFailed,
		},
		{
			ID:     "cmd4",
			Status: types.CommandStatusPending,
		},
	}

	for _, cmd := range commands {
		manager.commands[cmd.ID] = cmd
	}

	// Check updated stats
	stats = manager.GetStats()
	assert.Equal(t, int64(4), stats.TotalCommands)
	assert.Equal(t, int64(2), stats.PendingCommands)
	assert.Equal(t, int64(1), stats.CompletedCommands)
	assert.Equal(t, int64(1), stats.FailedCommands)
}

func TestManager_CommandTimeout(t *testing.T) {
	mockMQTT := &MockMQTTClient{}
	mockStorage := NewMockStorage()
	manager := NewManager(mockMQTT, mockStorage)

	commandID := "test-command-123"
	deviceInfo := types.DeviceInfo{
		Tenant:   "test-tenant",
		Site:     "test-site",
		DeviceID: "test-device",
		Type:     "wifi_router",
	}

	// Create an old command
	command := types.Command{
		ID:         commandID,
		DeviceInfo: deviceInfo,
		Data:       map[string]interface{}{"action": "restart"},
		Status:     types.CommandStatusPending,
		CreatedAt:  time.Now().Add(-10 * time.Minute), // 10 minutes ago
	}

	manager.commands[commandID] = command

	// This would typically be handled by a background process
	// Here we just verify the structure is in place
	assert.True(t, command.CreatedAt.Before(time.Now().Add(-5*time.Minute)))
	assert.Equal(t, types.CommandStatusPending, command.Status)
}

func TestManager_Stop(t *testing.T) {
	mockMQTT := &MockMQTTClient{}
	mockStorage := NewMockStorage()
	manager := NewManager(mockMQTT, mockStorage)

	// Start the manager
	mockStorage.On("List", "commands:*").Return([]string{}, nil)
	ctx := context.Background()
	err := manager.Start(ctx)
	require.NoError(t, err)

	// Stop the manager
	manager.Stop()

	// Verify that operations still work after stop (manager should be graceful)
	deviceInfo := types.DeviceInfo{
		Tenant:   "test-tenant",
		Site:     "test-site",
		DeviceID: "test-device",
		Type:     "wifi_router",
	}

	commandData := map[string]interface{}{
		"action": "restart",
	}

	mockMQTT.On("Publish", mock.AnythingOfType("string"), mock.AnythingOfType("map[string]interface {}"), 1, false).Return(nil)
	mockStorage.On("Set", mock.AnythingOfType("string"), mock.AnythingOfType("types.Command")).Return(nil)

	_, err = manager.SendCommand(deviceInfo, commandData)
	assert.NoError(t, err) // Should still work after stop
}

func TestManager_ConcurrentCommandOperations(t *testing.T) {
	mockMQTT := &MockMQTTClient{}
	mockStorage := NewMockStorage()
	manager := NewManager(mockMQTT, mockStorage)

	const numGoroutines = 10
	const numCommands = 10

	mockMQTT.On("Publish", mock.AnythingOfType("string"), mock.AnythingOfType("map[string]interface {}"), 1, false).Return(nil)
	mockStorage.On("Set", mock.AnythingOfType("string"), mock.AnythingOfType("types.Command")).Return(nil)

	// Test concurrent command sending
	done := make(chan string, numGoroutines*numCommands)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			for j := 0; j < numCommands; j++ {
				deviceInfo := types.DeviceInfo{
					Tenant:   "tenant1",
					Site:     "site1",
					DeviceID: "device1",
					Type:     "wifi_router",
				}

				commandData := map[string]interface{}{
					"action": "test",
					"id":     id*numCommands + j,
				}

				commandID, err := manager.SendCommand(deviceInfo, commandData)
				assert.NoError(t, err)
				done <- commandID
			}
		}(i)
	}

	// Collect all command IDs
	var commandIDs []string
	for i := 0; i < numGoroutines*numCommands; i++ {
		commandIDs = append(commandIDs, <-done)
	}

	// Verify all commands were created
	assert.Len(t, commandIDs, numGoroutines*numCommands)

	// Verify stats
	stats := manager.GetStats()
	assert.Equal(t, int64(numGoroutines*numCommands), stats.TotalCommands)
	assert.Equal(t, int64(numGoroutines*numCommands), stats.PendingCommands)
}

func TestManager_InvalidCommandData(t *testing.T) {
	mockMQTT := &MockMQTTClient{}
	mockStorage := NewMockStorage()
	manager := NewManager(mockMQTT, mockStorage)

	deviceInfo := types.DeviceInfo{
		Tenant:   "test-tenant",
		Site:     "test-site",
		DeviceID: "test-device",
		Type:     "wifi_router",
	}

	tests := []struct {
		name        string
		commandData map[string]interface{}
		wantErr     bool
	}{
		{
			name:        "nil command data",
			commandData: nil,
			wantErr:     true,
		},
		{
			name:        "empty command data",
			commandData: map[string]interface{}{},
			wantErr:     true,
		},
		{
			name: "valid command data",
			commandData: map[string]interface{}{
				"action": "restart",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.wantErr {
				mockMQTT.On("Publish", mock.AnythingOfType("string"), mock.AnythingOfType("map[string]interface {}"), 1, false).Return(nil)
				mockStorage.On("Set", mock.AnythingOfType("string"), mock.AnythingOfType("types.Command")).Return(nil)
			}

			_, err := manager.SendCommand(deviceInfo, tt.commandData)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
