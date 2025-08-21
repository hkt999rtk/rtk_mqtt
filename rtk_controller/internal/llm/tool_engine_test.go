package llm

import (
	"context"
	"testing"
	"time"

	"rtk_controller/internal/command"
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

// MockTool implements types.LLMTool for testing
type MockTool struct {
	mock.Mock
	name     string
	category types.ToolCategory
}

func NewMockTool(name string) *MockTool {
	return &MockTool{
		name:     name,
		category: types.ToolCategoryTest,
	}
}

func (m *MockTool) Name() string {
	return m.name
}

func (m *MockTool) Category() types.ToolCategory {
	return m.category
}

func (m *MockTool) Execute(ctx context.Context, params map[string]interface{}) (*types.ToolResult, error) {
	args := m.Called(ctx, params)
	return args.Get(0).(*types.ToolResult), args.Error(1)
}

func (m *MockTool) Validate(params map[string]interface{}) error {
	args := m.Called(params)
	return args.Error(0)
}

func (m *MockTool) RequiredCapabilities() []string {
	args := m.Called()
	if args.Get(0) == nil {
		return []string{}
	}
	return args.Get(0).([]string)
}

func (m *MockTool) Description() string {
	args := m.Called()
	if args.Get(0) == nil {
		return "Mock tool for testing"
	}
	return args.Get(0).(string)
}

func setupTestEngine(t *testing.T) (*ToolEngine, *MockStorage) {
	mockStorage := NewMockStorage()
	mockStorage.On("Set", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)
	mockStorage.On("Get", mock.AnythingOfType("string")).Return("", nil)
	mockStorage.On("Delete", mock.AnythingOfType("string")).Return(nil)
	mockStorage.On("Exists", mock.AnythingOfType("string")).Return(false, nil)

	// Create minimal command manager for testing (pass nil for mqtt client since we're testing without MQTT)
	commandManager := command.NewManager(nil, mockStorage)
	
	// Create engine with nil managers (we'll test without topology/qos dependencies)
	engine := NewToolEngine(mockStorage, commandManager, nil, nil)
	
	return engine, mockStorage
}

func TestToolEngine_NewToolEngine(t *testing.T) {
	engine, _ := setupTestEngine(t)

	assert.NotNil(t, engine)
	assert.NotNil(t, engine.tools)
	assert.NotNil(t, engine.sessions)
	assert.NotNil(t, engine.config)
	assert.Equal(t, DefaultEngineConfig().SessionTimeout, engine.config.SessionTimeout)
	assert.Equal(t, DefaultEngineConfig().ToolTimeout, engine.config.ToolTimeout)
	assert.False(t, engine.started)
}

func TestToolEngine_Start(t *testing.T) {
	engine, _ := setupTestEngine(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// First start should succeed
	err := engine.Start(ctx)
	assert.NoError(t, err)
	assert.True(t, engine.started)
	
	// Stop to clean up
	engine.Stop()

	// Second start should fail after already started
	engine.started = true // Manually set to test the error condition
	err = engine.Start(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already started")
}

func TestToolEngine_Stop(t *testing.T) {
	engine, _ := setupTestEngine(t)
	ctx := context.Background()

	// Stop without start should not error
	err := engine.Stop()
	assert.NoError(t, err)

	// Start then stop
	err = engine.Start(ctx)
	require.NoError(t, err)

	err = engine.Stop()
	assert.NoError(t, err)
	assert.False(t, engine.started)
}

func TestToolEngine_RegisterTool(t *testing.T) {
	engine, _ := setupTestEngine(t)

	mockTool := NewMockTool("test.tool")

	// Register new tool should succeed
	err := engine.RegisterTool(mockTool)
	assert.NoError(t, err)

	// Register same tool again should fail
	err = engine.RegisterTool(mockTool)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already registered")

	// Register tool with empty name should fail
	emptyNameTool := NewMockTool("")
	err = engine.RegisterTool(emptyNameTool)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be empty")
}

func TestToolEngine_GetTool(t *testing.T) {
	engine, _ := setupTestEngine(t)

	mockTool := NewMockTool("test.tool")
	err := engine.RegisterTool(mockTool)
	require.NoError(t, err)

	// Get existing tool
	tool, exists := engine.GetTool("test.tool")
	assert.True(t, exists)
	assert.Equal(t, mockTool, tool)

	// Get non-existing tool
	tool, exists = engine.GetTool("non.existent")
	assert.False(t, exists)
	assert.Nil(t, tool)
}

func TestToolEngine_ListTools(t *testing.T) {
	engine, _ := setupTestEngine(t)

	// Initially empty
	tools := engine.ListTools()
	assert.Empty(t, tools)

	// Add some tools
	tool1 := NewMockTool("tool.one")
	tool2 := NewMockTool("tool.two")

	err := engine.RegisterTool(tool1)
	require.NoError(t, err)
	err = engine.RegisterTool(tool2)
	require.NoError(t, err)

	tools = engine.ListTools()
	assert.Len(t, tools, 2)
	assert.Contains(t, tools, "tool.one")
	assert.Contains(t, tools, "tool.two")
}

func TestToolEngine_CreateSession(t *testing.T) {
	engine, mockStorage := setupTestEngine(t)
	ctx := context.Background()

	mockStorage.On("Set", mock.MatchedBy(func(key string) bool {
		return key == "llm_session:"+key[12:] // Match session key pattern
	}), mock.AnythingOfType("string")).Return(nil)

	// Create session without options
	session, err := engine.CreateSession(ctx, nil)
	assert.NoError(t, err)
	assert.NotNil(t, session)
	assert.NotEmpty(t, session.SessionID)
	assert.NotEmpty(t, session.TraceID)
	assert.Equal(t, types.LLMSessionStatusActive, session.Status)
	assert.NotZero(t, session.CreatedAt)

	// Create session with options
	options := &SessionOptions{
		DeviceID: "device123",
		UserID:   "user456",
		Metadata: map[string]interface{}{
			"source": "test",
		},
	}

	session2, err := engine.CreateSession(ctx, options)
	assert.NoError(t, err)
	assert.Equal(t, "device123", session2.DeviceID)
	assert.Equal(t, "user456", session2.UserID)
	assert.Equal(t, "test", session2.Metadata["source"])
}

func TestToolEngine_CreateSession_Limits(t *testing.T) {
	engine, mockStorage := setupTestEngine(t)
	ctx := context.Background()

	// Set low session limit for testing
	engine.config.MaxConcurrentSessions = 2

	mockStorage.On("Set", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)

	// Create sessions up to limit
	session1, err := engine.CreateSession(ctx, nil)
	assert.NoError(t, err)
	assert.NotNil(t, session1)

	session2, err := engine.CreateSession(ctx, nil)
	assert.NoError(t, err)
	assert.NotNil(t, session2)

	// Third session should fail
	session3, err := engine.CreateSession(ctx, nil)
	assert.Error(t, err)
	assert.Nil(t, session3)
	assert.Contains(t, err.Error(), "maximum concurrent sessions exceeded")
}

func TestToolEngine_ExecuteTool(t *testing.T) {
	engine, mockStorage := setupTestEngine(t)
	ctx := context.Background()

	mockStorage.On("Set", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)

	// Create session
	session, err := engine.CreateSession(ctx, nil)
	require.NoError(t, err)

	// Register mock tool
	mockTool := NewMockTool("test.tool")
	mockTool.On("Validate", mock.AnythingOfType("map[string]interface {}")).Return(nil)
	mockTool.On("Execute", mock.Anything, mock.AnythingOfType("map[string]interface {}")).Return(
		&types.ToolResult{
			ToolName:  "test.tool",
			Success:   true,
			Data:      map[string]interface{}{"result": "success"},
			Timestamp: time.Now(),
		}, nil)

	err = engine.RegisterTool(mockTool)
	require.NoError(t, err)

	// Execute tool
	params := map[string]interface{}{"param1": "value1"}
	result, err := engine.ExecuteTool(ctx, session.SessionID, "test.tool", params)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success)
	assert.Equal(t, "test.tool", result.ToolName)
	assert.Equal(t, session.SessionID, result.SessionID)
	assert.Equal(t, session.TraceID, result.TraceID)

	// Verify tool call was recorded in session
	updatedSession, err := engine.GetSession(session.SessionID)
	require.NoError(t, err)
	assert.Len(t, updatedSession.ToolCalls, 1)
	assert.Equal(t, "test.tool", updatedSession.ToolCalls[0].ToolName)
	assert.Equal(t, types.ToolCallStatusCompleted, updatedSession.ToolCalls[0].Status)

	mockTool.AssertExpectations(t)
}

func TestToolEngine_ExecuteTool_UnknownTool(t *testing.T) {
	engine, mockStorage := setupTestEngine(t)
	ctx := context.Background()

	mockStorage.On("Set", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)

	// Create session
	session, err := engine.CreateSession(ctx, nil)
	require.NoError(t, err)

	// Execute unknown tool
	params := map[string]interface{}{}
	result, err := engine.ExecuteTool(ctx, session.SessionID, "unknown.tool", params)

	assert.Error(t, err)
	assert.Nil(t, result)

	// Check error type
	toolErr, ok := err.(*types.ToolError)
	assert.True(t, ok)
	assert.Equal(t, types.ToolErrorUnknownTool, toolErr.Code)
}

func TestToolEngine_ExecuteTool_ValidationFailure(t *testing.T) {
	engine, mockStorage := setupTestEngine(t)
	ctx := context.Background()

	mockStorage.On("Set", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)

	// Create session
	session, err := engine.CreateSession(ctx, nil)
	require.NoError(t, err)

	// Register mock tool that fails validation
	mockTool := NewMockTool("test.tool")
	mockTool.On("Validate", mock.AnythingOfType("map[string]interface {}")).Return(
		assert.AnError)

	err = engine.RegisterTool(mockTool)
	require.NoError(t, err)

	// Execute tool with invalid params
	params := map[string]interface{}{"invalid": "params"}
	result, err := engine.ExecuteTool(ctx, session.SessionID, "test.tool", params)

	assert.Error(t, err)
	assert.Nil(t, result)

	// Check error type
	toolErr, ok := err.(*types.ToolError)
	assert.True(t, ok)
	assert.Equal(t, types.ToolErrorInvalidParameters, toolErr.Code)

	mockTool.AssertExpectations(t)
}

func TestToolEngine_GetSession(t *testing.T) {
	engine, mockStorage := setupTestEngine(t)
	ctx := context.Background()

	mockStorage.On("Set", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)

	// Create session
	originalSession, err := engine.CreateSession(ctx, nil)
	require.NoError(t, err)

	// Get existing session from memory
	session, err := engine.GetSession(originalSession.SessionID)
	assert.NoError(t, err)
	assert.Equal(t, originalSession.SessionID, session.SessionID)

	// Get non-existing session
	_, err = engine.GetSession("non-existent-id")
	assert.Error(t, err)
}

func TestToolEngine_CloseSession(t *testing.T) {
	engine, mockStorage := setupTestEngine(t)
	ctx := context.Background()

	mockStorage.On("Set", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)

	// Create session
	session, err := engine.CreateSession(ctx, nil)
	require.NoError(t, err)

	// Close session
	err = engine.CloseSession(session.SessionID, types.LLMSessionStatusCompleted)
	assert.NoError(t, err)

	// Session should no longer be in active sessions
	_, exists := engine.sessions[session.SessionID]
	assert.False(t, exists)

	// Close non-existing session should fail
	err = engine.CloseSession("non-existent", types.LLMSessionStatusCompleted)
	assert.Error(t, err)
}

func TestToolEngine_SessionCleanup(t *testing.T) {
	engine, mockStorage := setupTestEngine(t)
	ctx := context.Background()

	// Set very short timeout for testing
	engine.config.SessionTimeout = 100 * time.Millisecond

	mockStorage.On("Set", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)

	// Create session
	session, err := engine.CreateSession(ctx, nil)
	require.NoError(t, err)

	// Manually set old update time
	session.UpdatedAt = time.Now().Add(-2 * time.Hour)
	engine.sessions[session.SessionID] = session

	// Run cleanup
	engine.cleanupExpiredSessions()

	// Session should be removed
	_, exists := engine.sessions[session.SessionID]
	assert.False(t, exists)
}

func TestDefaultEngineConfig(t *testing.T) {
	config := DefaultEngineConfig()

	assert.Equal(t, 30*time.Minute, config.SessionTimeout)
	assert.Equal(t, 60*time.Second, config.ToolTimeout)
	assert.Equal(t, 10, config.MaxConcurrentSessions)
	assert.True(t, config.EnableTracing)
}

// Benchmark tests
func BenchmarkToolEngine_CreateSession(b *testing.B) {
	engine, mockStorage := setupTestEngine(&testing.T{})
	ctx := context.Background()

	mockStorage.On("Set", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		session, err := engine.CreateSession(ctx, nil)
		if err != nil {
			b.Fatal(err)
		}
		// Clean up to avoid hitting session limits
		delete(engine.sessions, session.SessionID)
	}
}

func BenchmarkToolEngine_ExecuteTool(b *testing.B) {
	engine, mockStorage := setupTestEngine(&testing.T{})
	ctx := context.Background()

	mockStorage.On("Set", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)

	// Create session
	session, err := engine.CreateSession(ctx, nil)
	if err != nil {
		b.Fatal(err)
	}

	// Register fast mock tool
	mockTool := NewMockTool("bench.tool")
	mockTool.On("Validate", mock.AnythingOfType("map[string]interface {}")).Return(nil)
	mockTool.On("Execute", mock.AnythingOfType("*context.timerCtx"), mock.AnythingOfType("map[string]interface {}")).Return(
		&types.ToolResult{
			ToolName:  "bench.tool",
			Success:   true,
			Timestamp: time.Now(),
		}, nil)

	err = engine.RegisterTool(mockTool)
	if err != nil {
		b.Fatal(err)
	}

	params := map[string]interface{}{"test": "value"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := engine.ExecuteTool(ctx, session.SessionID, "bench.tool", params)
		if err != nil {
			b.Fatal(err)
		}
	}
}