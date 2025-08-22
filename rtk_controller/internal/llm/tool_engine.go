package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"rtk_controller/internal/command"
	"rtk_controller/internal/qos"
	"rtk_controller/internal/storage"
	"rtk_controller/internal/topology"
	"rtk_controller/pkg/types"

	"github.com/google/uuid"
)

// ToolEngine manages the execution of LLM diagnostic tools
type ToolEngine struct {
	// Tool registry maps tool names to tool implementations
	tools map[string]types.LLMTool

	// Active sessions tracks ongoing diagnosis sessions
	sessions     map[string]*types.LLMSession
	sessionMutex sync.RWMutex

	// Dependencies on existing managers
	storage         storage.Storage
	commandManager  *command.Manager
	topologyManager *topology.Manager
	qosManager      *qos.QoSManager

	// Metrics collection
	metricsCollector *MetricsCollector

	// Configuration
	config *EngineConfig

	// Control channels
	stopCh  chan struct{}
	started bool
	mutex   sync.RWMutex
}

// EngineConfig contains configuration for the tool engine
type EngineConfig struct {
	// SessionTimeout is how long sessions can remain active
	SessionTimeout time.Duration

	// ToolTimeout is the default timeout for tool execution
	ToolTimeout time.Duration

	// MaxConcurrentSessions limits the number of active sessions
	MaxConcurrentSessions int

	// EnableTracing enables distributed tracing
	EnableTracing bool
}

// DefaultEngineConfig returns a sensible default configuration
func DefaultEngineConfig() *EngineConfig {
	return &EngineConfig{
		SessionTimeout:        30 * time.Minute,
		ToolTimeout:           60 * time.Second,
		MaxConcurrentSessions: 10,
		EnableTracing:         true,
	}
}

// NewToolEngine creates a new LLM tool engine
func NewToolEngine(
	storage storage.Storage,
	commandManager *command.Manager,
	topologyManager *topology.Manager,
	qosManager *qos.QoSManager,
) *ToolEngine {
	return &ToolEngine{
		tools:            make(map[string]types.LLMTool),
		sessions:         make(map[string]*types.LLMSession),
		storage:          storage,
		commandManager:   commandManager,
		topologyManager:  topologyManager,
		qosManager:       qosManager,
		metricsCollector: NewMetricsCollector(),
		config:           DefaultEngineConfig(),
		stopCh:           make(chan struct{}),
	}
}

// Start starts the tool engine
func (e *ToolEngine) Start(ctx context.Context) error {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	if e.started {
		return fmt.Errorf("tool engine already started")
	}

	// Register built-in tools
	if err := e.registerBuiltInTools(); err != nil {
		return fmt.Errorf("failed to register built-in tools: %w", err)
	}

	// Start background workers
	go e.sessionCleanupWorker(ctx)

	e.started = true
	return nil
}

// Stop stops the tool engine
func (e *ToolEngine) Stop() error {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	if !e.started {
		return nil
	}

	close(e.stopCh)
	e.started = false
	return nil
}

// RegisterTool registers a new tool with the engine
func (e *ToolEngine) RegisterTool(tool types.LLMTool) error {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	name := tool.Name()
	if name == "" {
		return fmt.Errorf("tool name cannot be empty")
	}

	if _, exists := e.tools[name]; exists {
		return fmt.Errorf("tool %s already registered", name)
	}

	e.tools[name] = tool
	return nil
}

// GetTool retrieves a tool by name
func (e *ToolEngine) GetTool(name string) (types.LLMTool, bool) {
	e.mutex.RLock()
	defer e.mutex.RUnlock()

	tool, exists := e.tools[name]
	return tool, exists
}

// ListTools returns all registered tool names
func (e *ToolEngine) ListTools() []string {
	e.mutex.RLock()
	defer e.mutex.RUnlock()

	tools := make([]string, 0, len(e.tools))
	for name := range e.tools {
		tools = append(tools, name)
	}
	return tools
}

// CreateSession creates a new LLM diagnosis session
func (e *ToolEngine) CreateSession(ctx context.Context, options interface{}) (*types.LLMSession, error) {
	e.sessionMutex.Lock()
	defer e.sessionMutex.Unlock()

	// Check session limits
	if len(e.sessions) >= e.config.MaxConcurrentSessions {
		return nil, fmt.Errorf("maximum concurrent sessions exceeded")
	}

	// Create new session
	session := &types.LLMSession{
		SessionID: uuid.New().String(),
		TraceID:   uuid.New().String(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Status:    types.LLMSessionStatusActive,
		ToolCalls: make([]types.ToolCall, 0),
		Metadata:  make(map[string]interface{}),
	}

	// Handle options with type assertion
	if options != nil {
		if sessionOpts, ok := options.(*SessionOptions); ok {
			if sessionOpts.DeviceID != "" {
				session.DeviceID = sessionOpts.DeviceID
			}
			if sessionOpts.UserID != "" {
				session.UserID = sessionOpts.UserID
			}
			if sessionOpts.Metadata != nil {
				for k, v := range sessionOpts.Metadata {
					session.Metadata[k] = v
				}
			}
		}
	}

	// Store session
	e.sessions[session.SessionID] = session

	// Persist to storage
	if err := e.persistSession(session); err != nil {
		delete(e.sessions, session.SessionID)
		return nil, fmt.Errorf("failed to persist session: %w", err)
	}

	// Record metrics
	e.metricsCollector.RecordSessionCreated()

	return session, nil
}

// ExecuteTool executes a tool within a session context
func (e *ToolEngine) ExecuteTool(ctx context.Context, sessionID, toolName string, params map[string]interface{}) (*types.ToolResult, error) {
	// Get session
	session, err := e.GetSession(sessionID)
	if err != nil {
		return nil, err
	}

	if session.Status != types.LLMSessionStatusActive {
		return nil, fmt.Errorf("session %s is not active", sessionID)
	}

	// Get tool
	tool, exists := e.GetTool(toolName)
	if !exists {
		return nil, &types.ToolError{
			Code:    types.ToolErrorUnknownTool,
			Message: fmt.Sprintf("tool %s not found", toolName),
		}
	}

	// Validate parameters
	if err := tool.Validate(params); err != nil {
		return nil, &types.ToolError{
			Code:    types.ToolErrorInvalidParameters,
			Message: "parameter validation failed",
			Details: err.Error(),
		}
	}

	// Create tool call record
	toolCall := types.ToolCall{
		ID:         uuid.New().String(),
		ToolName:   toolName,
		Parameters: params,
		StartedAt:  time.Now(),
		Status:     types.ToolCallStatusRunning,
	}

	// Add to session
	e.sessionMutex.Lock()
	session.ToolCalls = append(session.ToolCalls, toolCall)
	session.UpdatedAt = time.Now()
	e.sessionMutex.Unlock()

	// Execute tool with timeout
	ctx, cancel := context.WithTimeout(ctx, e.config.ToolTimeout)
	defer cancel()

	startTime := time.Now()
	result, err := tool.Execute(ctx, params)
	executionTime := time.Since(startTime)

	// Record tool execution metrics
	toolSuccess := (err == nil && result != nil && result.Success)
	errorMsg := ""
	if err != nil {
		errorMsg = err.Error()
	} else if result != nil && !result.Success {
		errorMsg = result.Error
	}
	e.metricsCollector.RecordToolExecution(toolName, executionTime, toolSuccess, errorMsg)

	// Update tool call with result
	e.sessionMutex.Lock()
	defer e.sessionMutex.Unlock()

	// Find the tool call in the session
	for i := range session.ToolCalls {
		if session.ToolCalls[i].ID == toolCall.ID {
			now := time.Now()
			session.ToolCalls[i].CompletedAt = &now
			session.UpdatedAt = now

			if err != nil {
				session.ToolCalls[i].Status = types.ToolCallStatusFailed
				result = &types.ToolResult{
					ToolName:      toolName,
					Success:       false,
					Error:         err.Error(),
					ExecutionTime: executionTime,
					Timestamp:     now,
					SessionID:     sessionID,
					TraceID:       session.TraceID,
				}
			} else {
				session.ToolCalls[i].Status = types.ToolCallStatusCompleted
				// Ensure result has session context
				if result != nil {
					result.SessionID = sessionID
					result.TraceID = session.TraceID
					result.ExecutionTime = executionTime
					result.Timestamp = now
				}
			}

			session.ToolCalls[i].Result = result
			break
		}
	}

	// Persist updated session
	if err := e.persistSession(session); err != nil {
		// Log error but don't fail the operation
		// TODO: Add proper logging
	}

	return result, err
}

// GetSession retrieves a session by ID
func (e *ToolEngine) GetSession(sessionID string) (*types.LLMSession, error) {
	e.sessionMutex.RLock()
	defer e.sessionMutex.RUnlock()

	session, exists := e.sessions[sessionID]
	if !exists {
		// Try to load from storage
		return e.loadSession(sessionID)
	}

	return session, nil
}

// CloseSession closes a diagnosis session
func (e *ToolEngine) CloseSession(sessionID string, status types.LLMSessionStatus) error {
	e.sessionMutex.Lock()
	defer e.sessionMutex.Unlock()

	session, exists := e.sessions[sessionID]
	if !exists {
		return fmt.Errorf("session %s not found", sessionID)
	}

	session.Status = status
	session.UpdatedAt = time.Now()

	// Calculate session duration
	sessionDuration := time.Since(session.CreatedAt)

	// Record session completion metrics
	switch status {
	case types.LLMSessionStatusCompleted:
		e.metricsCollector.RecordSessionCompleted(sessionDuration, true)
	case types.LLMSessionStatusFailed:
		e.metricsCollector.RecordSessionCompleted(sessionDuration, false)
	case types.LLMSessionStatusCancelled:
		e.metricsCollector.RecordSessionCancelled()
	}

	// Persist final state
	if err := e.persistSession(session); err != nil {
		return fmt.Errorf("failed to persist session final state: %w", err)
	}

	// Remove from active sessions
	delete(e.sessions, sessionID)

	return nil
}

// SessionOptions contains options for creating a new session
type SessionOptions struct {
	DeviceID string
	UserID   string
	Metadata map[string]interface{}
}

// registerBuiltInTools registers the built-in diagnostic tools
func (e *ToolEngine) registerBuiltInTools() error {
	// Register topology tools
	if e.topologyManager != nil {
		topologyGetFull := NewTopologyGetFullTool(e.topologyManager)
		if err := e.RegisterTool(topologyGetFull); err != nil {
			return fmt.Errorf("failed to register topology.get_full tool: %w", err)
		}

		clientsList := NewClientsListTool(e.topologyManager)
		if err := e.RegisterTool(clientsList); err != nil {
			return fmt.Errorf("failed to register clients.list tool: %w", err)
		}
	}

	// Register network/QoS tools
	if e.qosManager != nil {
		qosGetStatus := NewQoSGetStatusTool(e.qosManager)
		if err := e.RegisterTool(qosGetStatus); err != nil {
			return fmt.Errorf("failed to register qos.get_status tool: %w", err)
		}

		trafficGetStats := NewTrafficGetStatsTool(e.qosManager)
		if err := e.RegisterTool(trafficGetStats); err != nil {
			return fmt.Errorf("failed to register traffic.get_stats tool: %w", err)
		}
	}

	// Register test tools
	networkSpeedTest := NewNetworkSpeedTestTool()
	if err := e.RegisterTool(networkSpeedTest); err != nil {
		return fmt.Errorf("failed to register network.speedtest_full tool: %w", err)
	}

	wanConnectivity := NewWANConnectivityTool()
	if err := e.RegisterTool(wanConnectivity); err != nil {
		return fmt.Errorf("failed to register diagnostics.wan_connectivity tool: %w", err)
	}

	// Register WiFi advanced diagnostic tools
	wifiScanChannels := NewWiFiScanChannelsTool()
	if err := e.RegisterTool(wifiScanChannels); err != nil {
		return fmt.Errorf("failed to register wifi.scan_channels tool: %w", err)
	}

	wifiAnalyzeInterference := NewWiFiAnalyzeInterferenceTool()
	if err := e.RegisterTool(wifiAnalyzeInterference); err != nil {
		return fmt.Errorf("failed to register wifi.analyze_interference tool: %w", err)
	}

	wifiSpectrumUtilization := NewWiFiSpectrumUtilizationTool()
	if err := e.RegisterTool(wifiSpectrumUtilization); err != nil {
		return fmt.Errorf("failed to register wifi.spectrum_utilization tool: %w", err)
	}

	wifiSignalStrengthMap := NewWiFiSignalStrengthMapTool()
	if err := e.RegisterTool(wifiSignalStrengthMap); err != nil {
		return fmt.Errorf("failed to register wifi.signal_strength_map tool: %w", err)
	}

	wifiCoverageAnalysis := NewWiFiCoverageAnalysisTool()
	if err := e.RegisterTool(wifiCoverageAnalysis); err != nil {
		return fmt.Errorf("failed to register wifi.coverage_analysis tool: %w", err)
	}

	wifiRoamingOptimization := NewWiFiRoamingOptimizationTool()
	if err := e.RegisterTool(wifiRoamingOptimization); err != nil {
		return fmt.Errorf("failed to register wifi.roaming_optimization tool: %w", err)
	}

	wifiThroughputAnalysis := NewWiFiThroughputAnalysisTool()
	if err := e.RegisterTool(wifiThroughputAnalysis); err != nil {
		return fmt.Errorf("failed to register wifi.throughput_analysis tool: %w", err)
	}

	wifiLatencyProfiling := NewWiFiLatencyProfilingTool()
	if err := e.RegisterTool(wifiLatencyProfiling); err != nil {
		return fmt.Errorf("failed to register wifi.latency_profiling tool: %w", err)
	}

	// Register Mesh network diagnostic tools
	meshGetTopology := NewMeshGetTopologyTool()
	if err := e.RegisterTool(meshGetTopology); err != nil {
		return fmt.Errorf("failed to register mesh.get_topology tool: %w", err)
	}

	meshNodeRelationship := NewMeshNodeRelationshipTool()
	if err := e.RegisterTool(meshNodeRelationship); err != nil {
		return fmt.Errorf("failed to register mesh.node_relationship tool: %w", err)
	}

	meshPathOptimization := NewMeshPathOptimizationTool()
	if err := e.RegisterTool(meshPathOptimization); err != nil {
		return fmt.Errorf("failed to register mesh.path_optimization tool: %w", err)
	}

	meshBackhaulTest := NewMeshBackhaulTestTool()
	if err := e.RegisterTool(meshBackhaulTest); err != nil {
		return fmt.Errorf("failed to register mesh.backhaul_test tool: %w", err)
	}

	meshLoadBalancing := NewMeshLoadBalancingTool()
	if err := e.RegisterTool(meshLoadBalancing); err != nil {
		return fmt.Errorf("failed to register mesh.load_balancing tool: %w", err)
	}

	meshFailoverSimulation := NewMeshFailoverSimulationTool()
	if err := e.RegisterTool(meshFailoverSimulation); err != nil {
		return fmt.Errorf("failed to register mesh.failover_simulation tool: %w", err)
	}

	// Register Configuration Management tools
	configWiFiSettings := NewConfigWiFiSettingsTool()
	if err := e.RegisterTool(configWiFiSettings); err != nil {
		return fmt.Errorf("failed to register config.wifi_settings tool: %w", err)
	}

	configQoSPolicies := NewConfigQoSPoliciesTool()
	if err := e.RegisterTool(configQoSPolicies); err != nil {
		return fmt.Errorf("failed to register config.qos_policies tool: %w", err)
	}

	configSecuritySettings := NewConfigSecuritySettingsTool()
	if err := e.RegisterTool(configSecuritySettings); err != nil {
		return fmt.Errorf("failed to register config.security_settings tool: %w", err)
	}

	configBandSteering := NewConfigBandSteeringTool()
	if err := e.RegisterTool(configBandSteering); err != nil {
		return fmt.Errorf("failed to register config.band_steering tool: %w", err)
	}

	configAutoOptimize := NewConfigAutoOptimizeTool()
	if err := e.RegisterTool(configAutoOptimize); err != nil {
		return fmt.Errorf("failed to register config.auto_optimize tool: %w", err)
	}

	configValidateChanges := NewConfigValidateChangesTool()
	if err := e.RegisterTool(configValidateChanges); err != nil {
		return fmt.Errorf("failed to register config.validate_changes tool: %w", err)
	}

	configRollbackSafe := NewConfigRollbackSafeTool()
	if err := e.RegisterTool(configRollbackSafe); err != nil {
		return fmt.Errorf("failed to register config.rollback_safe tool: %w", err)
	}

	configImpactAnalysis := NewConfigImpactAnalysisTool()
	if err := e.RegisterTool(configImpactAnalysis); err != nil {
		return fmt.Errorf("failed to register config.impact_analysis tool: %w", err)
	}

	return nil
}

// sessionCleanupWorker periodically cleans up expired sessions
func (e *ToolEngine) sessionCleanupWorker(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-e.stopCh:
			return
		case <-ticker.C:
			e.cleanupExpiredSessions()
		}
	}
}

// cleanupExpiredSessions removes expired sessions
func (e *ToolEngine) cleanupExpiredSessions() {
	e.sessionMutex.Lock()
	defer e.sessionMutex.Unlock()

	now := time.Now()
	for sessionID, session := range e.sessions {
		if session.Status == types.LLMSessionStatusActive &&
			now.Sub(session.UpdatedAt) > e.config.SessionTimeout {
			session.Status = types.LLMSessionStatusCancelled
			session.UpdatedAt = now

			// Try to persist final state
			e.persistSession(session)

			// Remove from active sessions
			delete(e.sessions, sessionID)
		}
	}
}

// persistSession saves a session to storage
func (e *ToolEngine) persistSession(session *types.LLMSession) error {
	key := fmt.Sprintf("llm_session:%s", session.SessionID)
	data, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}
	return e.storage.Set(key, string(data))
}

// loadSession loads a session from storage
func (e *ToolEngine) loadSession(sessionID string) (*types.LLMSession, error) {
	key := fmt.Sprintf("llm_session:%s", sessionID)

	data, err := e.storage.Get(key)
	if err != nil {
		return nil, fmt.Errorf("session %s not found", sessionID)
	}

	var session types.LLMSession
	if err := json.Unmarshal([]byte(data), &session); err != nil {
		return nil, fmt.Errorf("failed to unmarshal session: %w", err)
	}

	// Add to active sessions if still active
	if session.Status == types.LLMSessionStatusActive {
		e.sessions[sessionID] = &session
	}

	return &session, nil
}

// GetMetrics returns the current metrics summary
func (e *ToolEngine) GetMetrics() *MetricsSummary {
	e.mutex.RLock()
	registeredTools := len(e.tools)
	e.mutex.RUnlock()

	// Update engine metrics
	e.metricsCollector.UpdateEngineMetrics(registeredTools, 0, 0) // TODO: Add actual memory/goroutine counts

	return e.metricsCollector.GetMetricsSummary()
}

// GetToolMetrics returns metrics for a specific tool
func (e *ToolEngine) GetToolMetrics(toolName string) *ToolMetrics {
	return e.metricsCollector.GetToolMetrics(toolName)
}

// GetSessionMetrics returns current session metrics
func (e *ToolEngine) GetSessionMetrics() *SessionMetrics {
	return e.metricsCollector.GetSessionMetrics()
}

// GetEngineMetrics returns current engine metrics
func (e *ToolEngine) GetEngineMetrics() *EngineMetrics {
	e.mutex.RLock()
	registeredTools := len(e.tools)
	e.mutex.RUnlock()

	e.metricsCollector.UpdateEngineMetrics(registeredTools, 0, 0)
	return e.metricsCollector.GetEngineMetrics()
}

// GetPrometheusMetrics returns metrics in Prometheus format
func (e *ToolEngine) GetPrometheusMetrics() string {
	return e.metricsCollector.GetPrometheusMetrics()
}

// UpdateHealthStatus updates the engine health status
func (e *ToolEngine) UpdateHealthStatus(isHealthy bool) {
	e.metricsCollector.UpdateHealthStatus(isHealthy)
}
