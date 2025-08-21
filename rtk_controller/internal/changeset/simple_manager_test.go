package changeset

import (
	"context"
	"fmt"
	"testing"
	"time"

	"rtk_controller/internal/storage"
	"rtk_controller/pkg/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockStorage implements storage.Storage for testing
type MockStorage struct {
	mock.Mock
	data map[string]string
}

func NewMockStorage() *MockStorage {
	return &MockStorage{
		data: make(map[string]string),
	}
}

func (m *MockStorage) Set(key, value string) error {
	args := m.Called(key, value)
	if args.Error(0) == nil {
		m.data[key] = value
	}
	return args.Error(0)
}

func (m *MockStorage) Get(key string) (string, error) {
	args := m.Called(key)
	if args.Error(1) != nil {
		return "", args.Error(1)
	}
	if value, exists := m.data[key]; exists {
		return value, nil
	}
	return "", args.Error(1)
}

func (m *MockStorage) Delete(key string) error {
	args := m.Called(key)
	if args.Error(0) == nil {
		delete(m.data, key)
	}
	return args.Error(0)
}

func (m *MockStorage) Exists(key string) (bool, error) {
	args := m.Called(key)
	_, exists := m.data[key]
	return exists, args.Error(0)
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


func setupTestManager(t *testing.T) (*SimpleManager, *MockStorage) {
	mockStorage := NewMockStorage()

	// Set up default mock responses
	mockStorage.On("Set", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)
	mockStorage.On("Get", mock.AnythingOfType("string")).Return("", nil)
	mockStorage.On("Delete", mock.AnythingOfType("string")).Return(nil)
	mockStorage.On("Exists", mock.AnythingOfType("string")).Return(false, nil)

	// Create a simple manager without command manager for basic testing
	manager := NewSimpleManager(mockStorage, nil)

	return manager, mockStorage
}

func TestSimpleManager_NewSimpleManager(t *testing.T) {
	mockStorage := NewMockStorage()

	manager := NewSimpleManager(mockStorage, nil)

	assert.NotNil(t, manager)
	assert.NotNil(t, manager.activeChangesets)
	assert.NotNil(t, manager.config)
	assert.Equal(t, DefaultManagerConfig().MaxActiveChangesets, manager.config.MaxActiveChangesets)
	assert.False(t, manager.started)
}

func TestSimpleManager_Start(t *testing.T) {
	manager, _ := setupTestManager(t)
	ctx := context.Background()

	// First start should succeed
	err := manager.Start(ctx)
	assert.NoError(t, err)
	assert.True(t, manager.started)

	// Second start should fail
	err = manager.Start(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already started")

	// Clean up
	manager.Stop()
}

func TestSimpleManager_Stop(t *testing.T) {
	manager, _ := setupTestManager(t)
	ctx := context.Background()

	// Stop without start should not error
	err := manager.Stop()
	assert.NoError(t, err)

	// Start then stop
	err = manager.Start(ctx)
	require.NoError(t, err)

	err = manager.Stop()
	assert.NoError(t, err)
	assert.False(t, manager.started)
}

func TestSimpleManager_CreateChangeset(t *testing.T) {
	manager, _ := setupTestManager(t)
	ctx := context.Background()

	// Test creating changeset without options
	changeset, err := manager.CreateChangeset(ctx, nil)
	assert.NoError(t, err)
	assert.NotNil(t, changeset)
	assert.NotEmpty(t, changeset.ID)
	assert.Equal(t, types.ChangesetStatusDraft, changeset.Status)
	assert.NotZero(t, changeset.CreatedAt)

	// Test creating changeset with options
	options := &types.ChangesetOptions{
		Description: "Test changeset",
		CreatedBy:   "test-user",
		SessionID:   "session-123",
		TraceID:     "trace-456",
		Metadata: map[string]interface{}{
			"source": "test",
		},
	}

	changeset2, err := manager.CreateChangeset(ctx, options)
	assert.NoError(t, err)
	assert.Equal(t, "Test changeset", changeset2.Description)
	assert.Equal(t, "test-user", changeset2.CreatedBy)
	assert.Equal(t, "session-123", changeset2.SessionID)
	assert.Equal(t, "trace-456", changeset2.TraceID)
	assert.Equal(t, "test", changeset2.Metadata["source"])
}

func TestSimpleManager_CreateChangeset_Limits(t *testing.T) {
	manager, _ := setupTestManager(t)
	ctx := context.Background()

	// Set low limit for testing
	manager.config.MaxActiveChangesets = 2

	// Create changesets up to limit
	changeset1, err := manager.CreateChangeset(ctx, nil)
	assert.NoError(t, err)
	assert.NotNil(t, changeset1)

	changeset2, err := manager.CreateChangeset(ctx, nil)
	assert.NoError(t, err)
	assert.NotNil(t, changeset2)

	// Third changeset should fail
	changeset3, err := manager.CreateChangeset(ctx, nil)
	assert.Error(t, err)
	assert.Nil(t, changeset3)
	assert.Contains(t, err.Error(), "maximum active changesets exceeded")
}

func TestSimpleManager_AddCommandToChangeset(t *testing.T) {
	manager, _ := setupTestManager(t)
	ctx := context.Background()

	// Create changeset
	changeset, err := manager.CreateChangeset(ctx, nil)
	require.NoError(t, err)

	// Add command
	cmd := &types.Command{
		ID:        "cmd-1",
		Operation: "wifi.update_settings",
		DeviceID:  "device-123",
		Args: map[string]interface{}{
			"ssid": "new-ssid",
		},
		Timeout: 30 * time.Second,
	}

	err = manager.AddCommandToChangeset(changeset.ID, cmd)
	assert.NoError(t, err)

	// Verify command was added
	updatedChangeset, err := manager.GetChangeset(changeset.ID)
	require.NoError(t, err)
	assert.Len(t, updatedChangeset.Commands, 1)
	assert.Equal(t, "cmd-1", updatedChangeset.Commands[0].ID)

	// Test adding to non-existent changeset
	err = manager.AddCommandToChangeset("non-existent", cmd)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestSimpleManager_GetChangeset(t *testing.T) {
	manager, _ := setupTestManager(t)
	ctx := context.Background()

	// Create changeset
	changeset, err := manager.CreateChangeset(ctx, nil)
	require.NoError(t, err)

	// Get existing changeset
	retrieved, err := manager.GetChangeset(changeset.ID)
	assert.NoError(t, err)
	assert.Equal(t, changeset.ID, retrieved.ID)

	// Get non-existing changeset
	_, err = manager.GetChangeset("non-existent")
	assert.Error(t, err)
}

func TestSimpleManager_DeleteChangeset(t *testing.T) {
	manager, _ := setupTestManager(t)
	ctx := context.Background()

	// Create changeset
	changeset, err := manager.CreateChangeset(ctx, nil)
	require.NoError(t, err)

	// Delete changeset
	err = manager.DeleteChangeset(changeset.ID)
	assert.NoError(t, err)

	// Verify it's deleted
	_, exists := manager.activeChangesets[changeset.ID]
	assert.False(t, exists)

	// Delete non-existing changeset
	err = manager.DeleteChangeset("non-existent")
	assert.Error(t, err)
}

func TestSimpleManager_ListChangesets(t *testing.T) {
	manager, _ := setupTestManager(t)
	ctx := context.Background()

	// Create some changesets
	changeset1, err := manager.CreateChangeset(ctx, &types.ChangesetOptions{
		CreatedBy: "user1",
	})
	require.NoError(t, err)

	_, err = manager.CreateChangeset(ctx, &types.ChangesetOptions{
		CreatedBy: "user2",
	})
	require.NoError(t, err)

	// List all changesets
	summaries, total, err := manager.ListChangesets(nil, 10, 0)
	assert.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Len(t, summaries, 2)

	// List with filter
	filter := &types.ChangesetFilter{
		CreatedBy: "user1",
	}
	summaries, total, err = manager.ListChangesets(filter, 10, 0)
	assert.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Len(t, summaries, 1)
	assert.Equal(t, changeset1.ID, summaries[0].ID)

	// List with pagination
	summaries, total, err = manager.ListChangesets(nil, 1, 1)
	assert.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Len(t, summaries, 1)
}

func TestDefaultManagerConfig(t *testing.T) {
	config := DefaultManagerConfig()

	assert.Equal(t, 50, config.MaxActiveChangesets)
	assert.Equal(t, 2*time.Hour, config.ChangesetTimeout)
	assert.True(t, config.AutoCleanup)
	assert.Equal(t, 1*time.Hour, config.CleanupInterval)
	assert.Equal(t, 30, config.RetentionDays)
}

// Benchmark tests
func BenchmarkSimpleManager_CreateChangeset(b *testing.B) {
	manager, _ := setupTestManager(&testing.T{})
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		changeset, err := manager.CreateChangeset(ctx, nil)
		if err != nil {
			b.Fatal(err)
		}
		// Clean up to avoid hitting limits
		delete(manager.activeChangesets, changeset.ID)
	}
}

func BenchmarkSimpleManager_AddCommandToChangeset(b *testing.B) {
	manager, _ := setupTestManager(&testing.T{})
	ctx := context.Background()

	changeset, err := manager.CreateChangeset(ctx, nil)
	if err != nil {
		b.Fatal(err)
	}

	cmd := &types.Command{
		ID:        "bench-cmd",
		Operation: "test.operation",
		DeviceID:  "bench-device",
		Args:      map[string]interface{}{"test": "value"},
		Timeout:   30 * time.Second,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cmd.ID = fmt.Sprintf("bench-cmd-%d", i)
		err := manager.AddCommandToChangeset(changeset.ID, cmd)
		if err != nil {
			b.Fatal(err)
		}
	}
}