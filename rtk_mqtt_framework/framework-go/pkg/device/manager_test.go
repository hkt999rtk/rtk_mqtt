package device

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"
)

func TestNewManager(t *testing.T) {
	config := &ManagerConfig{
		EventWorkers:        2,
		EventQueueSize:      100,
		HealthCheckInterval: 30 * time.Second,
		MetricsInterval:     60 * time.Second,
		PluginTimeout:       30 * time.Second,
	}
	
	manager := NewManager(config)
	
	if manager == nil {
		t.Fatal("Expected manager to be created")
	}
	
	if manager.plugins == nil {
		t.Error("Expected plugins map to be initialized")
	}
	
	if manager.factories == nil {
		t.Error("Expected factories map to be initialized")
	}
	
	if manager.eventWorkers != 2 {
		t.Errorf("Expected 2 event workers, got %d", manager.eventWorkers)
	}
	
	if cap(manager.eventQueue) != 100 {
		t.Errorf("Expected event queue size 100, got %d", cap(manager.eventQueue))
	}
}

func TestNewManagerWithDefaults(t *testing.T) {
	manager := NewManager(nil)
	
	if manager == nil {
		t.Fatal("Expected manager to be created with defaults")
	}
	
	if manager.eventWorkers != 4 {
		t.Errorf("Expected default 4 event workers, got %d", manager.eventWorkers)
	}
	
	if cap(manager.eventQueue) != 1000 {
		t.Errorf("Expected default event queue size 1000, got %d", cap(manager.eventQueue))
	}
}

func TestRegisterFactory(t *testing.T) {
	manager := NewManager(nil)
	factory := &MockPluginFactory{}
	
	err := manager.RegisterFactory("mock", factory)
	if err != nil {
		t.Fatalf("Failed to register factory: %v", err)
	}
	
	// Test duplicate registration
	err = manager.RegisterFactory("mock", factory)
	if err == nil {
		t.Error("Expected error for duplicate factory registration")
	}
}

func TestCreatePlugin(t *testing.T) {
	manager := NewManager(nil)
	factory := &MockPluginFactory{}
	
	// Register factory
	err := manager.RegisterFactory("mock", factory)
	if err != nil {
		t.Fatalf("Failed to register factory: %v", err)
	}
	
	// Create plugin
	config := &PluginConfig{
		Type:    "mock",
		Name:    "test_plugin",
		Enabled: true,
		Config:  json.RawMessage(`{"test": true}`),
	}
	
	err = manager.CreatePlugin("test_plugin", config)
	if err != nil {
		t.Fatalf("Failed to create plugin: %v", err)
	}
	
	// Verify plugin exists
	instance, err := manager.GetPlugin("test_plugin")
	if err != nil {
		t.Fatalf("Failed to get plugin: %v", err)
	}
	
	if instance == nil {
		t.Error("Expected plugin instance to exist")
	}
	
	if instance.State != StateInitialized {
		t.Errorf("Expected plugin state to be Initialized, got %s", instance.State)
	}
	
	// Test duplicate creation
	err = manager.CreatePlugin("test_plugin", config)
	if err == nil {
		t.Error("Expected error for duplicate plugin creation")
	}
	
	// Test creating with non-existent factory
	config.Type = "nonexistent"
	err = manager.CreatePlugin("test_plugin2", config)
	if err == nil {
		t.Error("Expected error for non-existent factory")
	}
}

func TestStartStopPlugin(t *testing.T) {
	manager := NewManager(nil)
	factory := &MockPluginFactory{}
	
	// Register factory and create plugin
	manager.RegisterFactory("mock", factory)
	config := &PluginConfig{
		Type:    "mock",
		Name:    "test_plugin",
		Enabled: true,
		Config:  json.RawMessage(`{}`),
	}
	manager.CreatePlugin("test_plugin", config)
	
	// Start plugin
	err := manager.StartPlugin("test_plugin")
	if err != nil {
		t.Fatalf("Failed to start plugin: %v", err)
	}
	
	// Verify plugin is running
	state, err := manager.GetPluginState("test_plugin")
	if err != nil {
		t.Fatalf("Failed to get plugin state: %v", err)
	}
	
	if state != StateRunning {
		t.Errorf("Expected plugin state to be Running, got %s", state)
	}
	
	// Test starting already running plugin
	err = manager.StartPlugin("test_plugin")
	if err == nil {
		t.Error("Expected error when starting already running plugin")
	}
	
	// Stop plugin
	err = manager.StopPlugin("test_plugin")
	if err != nil {
		t.Fatalf("Failed to stop plugin: %v", err)
	}
	
	// Verify plugin is stopped
	state, err = manager.GetPluginState("test_plugin")
	if err != nil {
		t.Fatalf("Failed to get plugin state: %v", err)
	}
	
	if state != StateStopped {
		t.Errorf("Expected plugin state to be Stopped, got %s", state)
	}
	
	// Test stopping non-running plugin
	err = manager.StopPlugin("test_plugin")
	if err == nil {
		t.Error("Expected error when stopping non-running plugin")
	}
}

func TestRemovePlugin(t *testing.T) {
	manager := NewManager(nil)
	factory := &MockPluginFactory{}
	
	// Register factory and create plugin
	manager.RegisterFactory("mock", factory)
	config := &PluginConfig{
		Type:    "mock",
		Name:    "test_plugin",
		Enabled: true,
		Config:  json.RawMessage(`{}`),
	}
	manager.CreatePlugin("test_plugin", config)
	manager.StartPlugin("test_plugin")
	
	// Remove plugin
	err := manager.RemovePlugin("test_plugin")
	if err != nil {
		t.Fatalf("Failed to remove plugin: %v", err)
	}
	
	// Verify plugin is removed
	_, err = manager.GetPlugin("test_plugin")
	if err == nil {
		t.Error("Expected error when getting removed plugin")
	}
	
	// Test removing non-existent plugin
	err = manager.RemovePlugin("nonexistent")
	if err == nil {
		t.Error("Expected error when removing non-existent plugin")
	}
}

func TestListPlugins(t *testing.T) {
	manager := NewManager(nil)
	factory := &MockPluginFactory{}
	
	// Register factory
	manager.RegisterFactory("mock", factory)
	
	// Initially no plugins
	plugins := manager.ListPlugins()
	if len(plugins) != 0 {
		t.Errorf("Expected 0 plugins, got %d", len(plugins))
	}
	
	// Create some plugins
	for i := 0; i < 3; i++ {
		config := &PluginConfig{
			Type:    "mock",
			Name:    fmt.Sprintf("plugin_%d", i),
			Enabled: true,
			Config:  json.RawMessage(`{}`),
		}
		manager.CreatePlugin(config.Name, config)
	}
	
	// Check plugin count
	plugins = manager.ListPlugins()
	if len(plugins) != 3 {
		t.Errorf("Expected 3 plugins, got %d", len(plugins))
	}
}

func TestHandleCommand(t *testing.T) {
	manager := NewManager(nil)
	factory := &MockPluginFactory{}
	
	// Register factory and create plugin
	manager.RegisterFactory("mock", factory)
	config := &PluginConfig{
		Type:    "mock",
		Name:    "test_plugin",
		Enabled: true,
		Config:  json.RawMessage(`{}`),
	}
	manager.CreatePlugin("test_plugin", config)
	manager.StartPlugin("test_plugin")
	
	// Send command
	cmd := &Command{
		ID:        "cmd_001",
		Type:      "test",
		Action:    "ping",
		Params:    map[string]interface{}{},
		Timestamp: time.Now(),
	}
	
	response, err := manager.HandleCommand("test_plugin", cmd)
	if err != nil {
		t.Fatalf("Failed to handle command: %v", err)
	}
	
	if response == nil {
		t.Error("Expected command response")
	}
	
	if response.CommandID != cmd.ID {
		t.Errorf("Expected command ID %s, got %s", cmd.ID, response.CommandID)
	}
	
	// Test command to non-existent plugin
	_, err = manager.HandleCommand("nonexistent", cmd)
	if err == nil {
		t.Error("Expected error for command to non-existent plugin")
	}
	
	// Test command to stopped plugin
	manager.StopPlugin("test_plugin")
	response, err = manager.HandleCommand("test_plugin", cmd)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	
	if response.Status != "error" {
		t.Errorf("Expected error status for stopped plugin, got %s", response.Status)
	}
}

func TestGetDeviceState(t *testing.T) {
	manager := NewManager(nil)
	factory := &MockPluginFactory{}
	
	// Register factory and create plugin
	manager.RegisterFactory("mock", factory)
	config := &PluginConfig{
		Type:    "mock",
		Name:    "test_plugin",
		Enabled: true,
		Config:  json.RawMessage(`{}`),
	}
	manager.CreatePlugin("test_plugin", config)
	manager.StartPlugin("test_plugin")
	
	// Get device state
	state, err := manager.GetDeviceState("test_plugin")
	if err != nil {
		t.Fatalf("Failed to get device state: %v", err)
	}
	
	if state == nil {
		t.Error("Expected device state")
	}
	
	// Test getting state from non-existent plugin
	_, err = manager.GetDeviceState("nonexistent")
	if err == nil {
		t.Error("Expected error for non-existent plugin")
	}
}

func TestEventHandling(t *testing.T) {
	manager := NewManager(nil)
	
	// Add event handler
	eventHandler := &MockEventHandler{}
	manager.AddEventHandler(eventHandler)
	
	// Start manager to enable event processing
	err := manager.Start()
	if err != nil {
		t.Fatalf("Failed to start manager: %v", err)
	}
	defer manager.Stop()
	
	// Publish event
	event := &Event{
		ID:        "event_001",
		Type:      "test",
		Level:     "info",
		Message:   "Test event",
		Source:    "test",
		Timestamp: time.Now(),
	}
	
	manager.PublishEvent(event)
	
	// Give some time for event processing
	time.Sleep(100 * time.Millisecond)
	
	// Check if event was handled
	if len(eventHandler.HandledEvents) != 1 {
		t.Errorf("Expected 1 handled event, got %d", len(eventHandler.HandledEvents))
	}
	
	if len(eventHandler.HandledEvents) > 0 {
		handledEvent := eventHandler.HandledEvents[0]
		if handledEvent.ID != event.ID {
			t.Errorf("Expected event ID %s, got %s", event.ID, handledEvent.ID)
		}
	}
}

func TestGetRunningPlugins(t *testing.T) {
	manager := NewManager(nil)
	factory := &MockPluginFactory{}
	
	// Register factory
	manager.RegisterFactory("mock", factory)
	
	// Create and start some plugins
	pluginNames := []string{"plugin1", "plugin2", "plugin3"}
	for _, name := range pluginNames {
		config := &PluginConfig{
			Type:    "mock",
			Name:    name,
			Enabled: true,
			Config:  json.RawMessage(`{}`),
		}
		manager.CreatePlugin(name, config)
		manager.StartPlugin(name)
	}
	
	// Stop one plugin
	manager.StopPlugin("plugin2")
	
	// Get running plugins
	runningPlugins := manager.GetRunningPlugins()
	
	if len(runningPlugins) != 2 {
		t.Errorf("Expected 2 running plugins, got %d", len(runningPlugins))
	}
	
	// Check that stopped plugin is not in the list
	for _, name := range runningPlugins {
		if name == "plugin2" {
			t.Error("Expected stopped plugin not to be in running list")
		}
	}
}

// Mock implementations for testing

type MockPluginFactory struct{}

func (f *MockPluginFactory) CreatePlugin() (Plugin, error) {
	return &MockPlugin{}, nil
}

func (f *MockPluginFactory) GetInfo() *FactoryInfo {
	return &FactoryInfo{
		Name:         "mock",
		Version:      "1.0.0",
		SupportedTypes: []string{"mock"},
		Description:  "Mock plugin for testing",
	}
}

type MockPlugin struct {
	initialized bool
	running     bool
}

func (p *MockPlugin) GetInfo() *Info {
	return &Info{
		Name:        "Mock Plugin",
		Type:        "mock",
		Version:     "1.0.0",
		Description: "Mock plugin for testing",
		Vendor:      "Test",
	}
}

func (p *MockPlugin) Initialize(ctx context.Context, config json.RawMessage) error {
	p.initialized = true
	return nil
}

func (p *MockPlugin) Start(ctx context.Context) error {
	if !p.initialized {
		return fmt.Errorf("plugin not initialized")
	}
	p.running = true
	return nil
}

func (p *MockPlugin) Stop() error {
	p.running = false
	return nil
}

func (p *MockPlugin) GetState(ctx context.Context) (*State, error) {
	status := "offline"
	if p.running {
		status = "online"
	}
	
	return &State{
		Status:    status,
		Health:    "healthy",
		LastSeen:  time.Now(),
		Uptime:    time.Hour,
		Timestamp: time.Now(),
		Properties: map[string]interface{}{
			"mock": true,
		},
	}, nil
}

func (p *MockPlugin) GetTelemetry(ctx context.Context, metric string) (*TelemetryData, error) {
	return &TelemetryData{
		Metric:    metric,
		Value:     42.0,
		Unit:      "mock_unit",
		Timestamp: time.Now(),
	}, nil
}

func (p *MockPlugin) GetAttributes(ctx context.Context) (map[string]interface{}, error) {
	return map[string]interface{}{
		"type":    "mock",
		"version": "1.0.0",
	}, nil
}

func (p *MockPlugin) HandleCommand(ctx context.Context, cmd *Command) (*CommandResponse, error) {
	return &CommandResponse{
		CommandID: cmd.ID,
		Status:    "success",
		Result: map[string]interface{}{
			"action": cmd.Action,
			"mock":   true,
		},
		Timestamp: time.Now(),
	}, nil
}

func (p *MockPlugin) GetHealth(ctx context.Context) (*Health, error) {
	return &Health{
		Status:    "healthy",
		Score:     1.0,
		LastCheck: time.Now(),
		Checks: map[string]HealthCheck{
			"mock": {
				Name:        "Mock Check",
				Status:      "healthy",
				Message:     "Mock plugin is healthy",
				LastChecked: time.Now(),
			},
		},
	}, nil
}

type MockEventHandler struct {
	HandledEvents []*Event
}

func (h *MockEventHandler) HandleEvent(ctx context.Context, event *Event) error {
	h.HandledEvents = append(h.HandledEvents, event)
	return nil
}

