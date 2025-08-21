package llm

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// MetricsCollector collects and manages LLM tool execution metrics
type MetricsCollector struct {
	// Tool execution metrics
	toolExecutions map[string]*ToolMetrics
	toolMutex      sync.RWMutex

	// Session metrics
	sessionMetrics *SessionMetrics
	sessionMutex   sync.RWMutex

	// Engine metrics
	engineMetrics *EngineMetrics
	engineMutex   sync.RWMutex

	// Start time for uptime calculation
	startTime time.Time
}

// ToolMetrics represents metrics for a specific tool
type ToolMetrics struct {
	// Basic counters
	TotalExecutions   int64 `json:"total_executions"`
	SuccessfulExecs   int64 `json:"successful_executions"`
	FailedExecs       int64 `json:"failed_executions"`
	TimeoutExecs      int64 `json:"timeout_executions"`

	// Timing metrics
	TotalExecutionTime time.Duration `json:"total_execution_time"`
	MinExecutionTime   time.Duration `json:"min_execution_time"`
	MaxExecutionTime   time.Duration `json:"max_execution_time"`
	AvgExecutionTime   time.Duration `json:"avg_execution_time"`

	// Error tracking
	LastError     string    `json:"last_error,omitempty"`
	LastErrorTime time.Time `json:"last_error_time,omitempty"`

	// Performance tracking
	LastExecutionTime time.Duration `json:"last_execution_time"`
	LastUpdated       time.Time     `json:"last_updated"`
}

// SessionMetrics represents session-related metrics
type SessionMetrics struct {
	// Session counters
	TotalSessions    int64 `json:"total_sessions"`
	ActiveSessions   int64 `json:"active_sessions"`
	CompletedSessions int64 `json:"completed_sessions"`
	FailedSessions   int64 `json:"failed_sessions"`
	CancelledSessions int64 `json:"cancelled_sessions"`

	// Session timing
	TotalSessionTime  time.Duration `json:"total_session_time"`
	AvgSessionTime    time.Duration `json:"avg_session_time"`
	MaxConcurrentSessions int64     `json:"max_concurrent_sessions"`

	// Session lifecycle
	SessionsCreatedToday  int64     `json:"sessions_created_today"`
	LastSessionCreated    time.Time `json:"last_session_created"`
	LastSessionCompleted  time.Time `json:"last_session_completed"`
}

// EngineMetrics represents overall engine metrics
type EngineMetrics struct {
	// Engine status
	Uptime          time.Duration `json:"uptime"`
	StartTime       time.Time     `json:"start_time"`
	IsHealthy       bool          `json:"is_healthy"`
	LastHealthCheck time.Time     `json:"last_health_check"`

	// Resource usage
	RegisteredTools   int   `json:"registered_tools"`
	MemoryUsageBytes  int64 `json:"memory_usage_bytes"`
	GoroutineCount    int   `json:"goroutine_count"`

	// Performance indicators
	RequestsPerSecond float64 `json:"requests_per_second"`
	AvgResponseTime   time.Duration `json:"avg_response_time"`
	ErrorRate         float64 `json:"error_rate"`

	// System health
	DatabaseConnections int   `json:"database_connections"`
	LastUpdated        time.Time `json:"last_updated"`
}

// MetricsSummary provides a consolidated view of all metrics
type MetricsSummary struct {
	Timestamp      time.Time                    `json:"timestamp"`
	Engine         *EngineMetrics              `json:"engine"`
	Sessions       *SessionMetrics             `json:"sessions"`
	Tools          map[string]*ToolMetrics     `json:"tools"`
	TopTools       []*ToolUsageSummary         `json:"top_tools"`
	HealthStatus   string                      `json:"health_status"`
}

// ToolUsageSummary provides a summary of tool usage
type ToolUsageSummary struct {
	ToolName        string        `json:"tool_name"`
	ExecutionCount  int64         `json:"execution_count"`
	SuccessRate     float64       `json:"success_rate"`
	AvgExecTime     time.Duration `json:"avg_execution_time"`
	LastUsed        time.Time     `json:"last_used"`
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		toolExecutions: make(map[string]*ToolMetrics),
		sessionMetrics: &SessionMetrics{},
		engineMetrics: &EngineMetrics{
			StartTime: time.Now(),
			IsHealthy: true,
		},
		startTime: time.Now(),
	}
}

// RecordToolExecution records metrics for a tool execution
func (mc *MetricsCollector) RecordToolExecution(toolName string, duration time.Duration, success bool, errorMsg string) {
	mc.toolMutex.Lock()
	defer mc.toolMutex.Unlock()

	metrics, exists := mc.toolExecutions[toolName]
	if !exists {
		metrics = &ToolMetrics{
			MinExecutionTime: duration,
			MaxExecutionTime: duration,
		}
		mc.toolExecutions[toolName] = metrics
	}

	// Update counters
	metrics.TotalExecutions++
	if success {
		metrics.SuccessfulExecs++
	} else {
		metrics.FailedExecs++
		metrics.LastError = errorMsg
		metrics.LastErrorTime = time.Now()
	}

	// Update timing metrics
	metrics.TotalExecutionTime += duration
	metrics.LastExecutionTime = duration
	if duration < metrics.MinExecutionTime {
		metrics.MinExecutionTime = duration
	}
	if duration > metrics.MaxExecutionTime {
		metrics.MaxExecutionTime = duration
	}
	metrics.AvgExecutionTime = time.Duration(int64(metrics.TotalExecutionTime) / metrics.TotalExecutions)
	metrics.LastUpdated = time.Now()
}

// RecordToolTimeout records a tool timeout
func (mc *MetricsCollector) RecordToolTimeout(toolName string) {
	mc.toolMutex.Lock()
	defer mc.toolMutex.Unlock()

	metrics, exists := mc.toolExecutions[toolName]
	if !exists {
		metrics = &ToolMetrics{}
		mc.toolExecutions[toolName] = metrics
	}

	metrics.TimeoutExecs++
	metrics.LastError = "Tool execution timeout"
	metrics.LastErrorTime = time.Now()
	metrics.LastUpdated = time.Now()
}

// RecordSessionCreated records a new session creation
func (mc *MetricsCollector) RecordSessionCreated() {
	mc.sessionMutex.Lock()
	defer mc.sessionMutex.Unlock()

	mc.sessionMetrics.TotalSessions++
	mc.sessionMetrics.ActiveSessions++
	mc.sessionMetrics.LastSessionCreated = time.Now()

	// Update max concurrent sessions
	if mc.sessionMetrics.ActiveSessions > mc.sessionMetrics.MaxConcurrentSessions {
		mc.sessionMetrics.MaxConcurrentSessions = mc.sessionMetrics.ActiveSessions
	}

	// Update daily counter (simplified - resets at midnight would be more accurate)
	mc.sessionMetrics.SessionsCreatedToday++
}

// RecordSessionCompleted records a session completion
func (mc *MetricsCollector) RecordSessionCompleted(duration time.Duration, success bool) {
	mc.sessionMutex.Lock()
	defer mc.sessionMutex.Unlock()

	mc.sessionMetrics.ActiveSessions--
	if success {
		mc.sessionMetrics.CompletedSessions++
	} else {
		mc.sessionMetrics.FailedSessions++
	}

	mc.sessionMetrics.TotalSessionTime += duration
	if mc.sessionMetrics.CompletedSessions > 0 {
		mc.sessionMetrics.AvgSessionTime = time.Duration(
			int64(mc.sessionMetrics.TotalSessionTime) / mc.sessionMetrics.CompletedSessions,
		)
	}

	mc.sessionMetrics.LastSessionCompleted = time.Now()
}

// RecordSessionCancelled records a session cancellation
func (mc *MetricsCollector) RecordSessionCancelled() {
	mc.sessionMutex.Lock()
	defer mc.sessionMutex.Unlock()

	mc.sessionMetrics.ActiveSessions--
	mc.sessionMetrics.CancelledSessions++
}

// UpdateEngineMetrics updates engine-level metrics
func (mc *MetricsCollector) UpdateEngineMetrics(registeredTools int, memoryUsage int64, goroutineCount int) {
	mc.engineMutex.Lock()
	defer mc.engineMutex.Unlock()

	mc.engineMetrics.Uptime = time.Since(mc.startTime)
	mc.engineMetrics.RegisteredTools = registeredTools
	mc.engineMetrics.MemoryUsageBytes = memoryUsage
	mc.engineMetrics.GoroutineCount = goroutineCount
	mc.engineMetrics.LastUpdated = time.Now()

	// Calculate performance indicators
	mc.calculatePerformanceMetrics()
}

// UpdateHealthStatus updates the engine health status
func (mc *MetricsCollector) UpdateHealthStatus(isHealthy bool) {
	mc.engineMutex.Lock()
	defer mc.engineMutex.Unlock()

	mc.engineMetrics.IsHealthy = isHealthy
	mc.engineMetrics.LastHealthCheck = time.Now()
}

// GetToolMetrics returns metrics for a specific tool
func (mc *MetricsCollector) GetToolMetrics(toolName string) *ToolMetrics {
	mc.toolMutex.RLock()
	defer mc.toolMutex.RUnlock()

	if metrics, exists := mc.toolExecutions[toolName]; exists {
		// Return a copy to avoid race conditions
		metricsCopy := *metrics
		return &metricsCopy
	}
	return nil
}

// GetSessionMetrics returns current session metrics
func (mc *MetricsCollector) GetSessionMetrics() *SessionMetrics {
	mc.sessionMutex.RLock()
	defer mc.sessionMutex.RUnlock()

	// Return a copy to avoid race conditions
	metricsCopy := *mc.sessionMetrics
	return &metricsCopy
}

// GetEngineMetrics returns current engine metrics
func (mc *MetricsCollector) GetEngineMetrics() *EngineMetrics {
	mc.engineMutex.RLock()
	defer mc.engineMutex.RUnlock()

	// Return a copy to avoid race conditions
	metricsCopy := *mc.engineMetrics
	metricsCopy.Uptime = time.Since(mc.startTime)
	return &metricsCopy
}

// GetMetricsSummary returns a comprehensive metrics summary
func (mc *MetricsCollector) GetMetricsSummary() *MetricsSummary {
	summary := &MetricsSummary{
		Timestamp:    time.Now(),
		Engine:       mc.GetEngineMetrics(),
		Sessions:     mc.GetSessionMetrics(),
		Tools:        make(map[string]*ToolMetrics),
		TopTools:     mc.getTopTools(5),
		HealthStatus: mc.getHealthStatus(),
	}

	// Copy tool metrics
	mc.toolMutex.RLock()
	for toolName, metrics := range mc.toolExecutions {
		metricsCopy := *metrics
		summary.Tools[toolName] = &metricsCopy
	}
	mc.toolMutex.RUnlock()

	return summary
}

// GetPrometheusMetrics returns metrics in Prometheus format
func (mc *MetricsCollector) GetPrometheusMetrics() string {
	summary := mc.GetMetricsSummary()
	
	var metrics []string
	
	// Engine metrics
	metrics = append(metrics, "# HELP llm_engine_uptime_seconds Engine uptime in seconds")
	metrics = append(metrics, "# TYPE llm_engine_uptime_seconds counter")
	metrics = append(metrics, fmt.Sprintf("llm_engine_uptime_seconds %d", int64(summary.Engine.Uptime.Seconds())))
	
	metrics = append(metrics, "# HELP llm_engine_registered_tools Number of registered tools")
	metrics = append(metrics, "# TYPE llm_engine_registered_tools gauge")
	metrics = append(metrics, fmt.Sprintf("llm_engine_registered_tools %d", summary.Engine.RegisteredTools))
	
	// Session metrics
	metrics = append(metrics, "# HELP llm_sessions_total Total number of sessions")
	metrics = append(metrics, "# TYPE llm_sessions_total counter")
	metrics = append(metrics, fmt.Sprintf("llm_sessions_total %d", summary.Sessions.TotalSessions))
	
	metrics = append(metrics, "# HELP llm_sessions_active Current number of active sessions")
	metrics = append(metrics, "# TYPE llm_sessions_active gauge")
	metrics = append(metrics, fmt.Sprintf("llm_sessions_active %d", summary.Sessions.ActiveSessions))
	
	// Tool metrics
	for toolName, toolMetrics := range summary.Tools {
		metrics = append(metrics, fmt.Sprintf("# HELP llm_tool_executions_total Total executions for tool %s", toolName))
		metrics = append(metrics, "# TYPE llm_tool_executions_total counter")
		metrics = append(metrics, fmt.Sprintf("llm_tool_executions_total{tool=\"%s\"} %d", toolName, toolMetrics.TotalExecutions))
		
		metrics = append(metrics, fmt.Sprintf("# HELP llm_tool_execution_duration_seconds Execution duration for tool %s", toolName))
		metrics = append(metrics, "# TYPE llm_tool_execution_duration_seconds histogram")
		metrics = append(metrics, fmt.Sprintf("llm_tool_execution_duration_seconds{tool=\"%s\"} %f", toolName, toolMetrics.AvgExecutionTime.Seconds()))
	}
	
	return strings.Join(metrics, "\n")
}

// Helper methods

func (mc *MetricsCollector) calculatePerformanceMetrics() {
	// Calculate requests per second (simplified)
	totalExecs := int64(0)
	totalErrors := int64(0)
	
	for _, metrics := range mc.toolExecutions {
		totalExecs += metrics.TotalExecutions
		totalErrors += metrics.FailedExecs + metrics.TimeoutExecs
	}
	
	uptimeSeconds := mc.engineMetrics.Uptime.Seconds()
	if uptimeSeconds > 0 {
		mc.engineMetrics.RequestsPerSecond = float64(totalExecs) / uptimeSeconds
	}
	
	// Calculate error rate
	if totalExecs > 0 {
		mc.engineMetrics.ErrorRate = float64(totalErrors) / float64(totalExecs)
	}
}

func (mc *MetricsCollector) getTopTools(limit int) []*ToolUsageSummary {
	type toolUsage struct {
		name    string
		metrics *ToolMetrics
	}
	
	mc.toolMutex.RLock()
	defer mc.toolMutex.RUnlock()
	
	// Collect all tools
	tools := make([]toolUsage, 0, len(mc.toolExecutions))
	for name, metrics := range mc.toolExecutions {
		tools = append(tools, toolUsage{name: name, metrics: metrics})
	}
	
	// Sort by execution count (simple bubble sort for small datasets)
	for i := 0; i < len(tools)-1; i++ {
		for j := 0; j < len(tools)-i-1; j++ {
			if tools[j].metrics.TotalExecutions < tools[j+1].metrics.TotalExecutions {
				tools[j], tools[j+1] = tools[j+1], tools[j]
			}
		}
	}
	
	// Convert to summary format
	summary := make([]*ToolUsageSummary, 0, limit)
	for i, tool := range tools {
		if i >= limit {
			break
		}
		
		successRate := float64(0)
		if tool.metrics.TotalExecutions > 0 {
			successRate = float64(tool.metrics.SuccessfulExecs) / float64(tool.metrics.TotalExecutions)
		}
		
		summary = append(summary, &ToolUsageSummary{
			ToolName:       tool.name,
			ExecutionCount: tool.metrics.TotalExecutions,
			SuccessRate:    successRate,
			AvgExecTime:    tool.metrics.AvgExecutionTime,
			LastUsed:       tool.metrics.LastUpdated,
		})
	}
	
	return summary
}

func (mc *MetricsCollector) getHealthStatus() string {
	mc.engineMutex.RLock()
	defer mc.engineMutex.RUnlock()
	
	if !mc.engineMetrics.IsHealthy {
		return "unhealthy"
	}
	
	// Check error rate
	if mc.engineMetrics.ErrorRate > 0.1 { // More than 10% error rate
		return "degraded"
	}
	
	// Check if engine is responding
	if time.Since(mc.engineMetrics.LastHealthCheck) > 5*time.Minute {
		return "unknown"
	}
	
	return "healthy"
}

// Reset resets all metrics (useful for testing)
func (mc *MetricsCollector) Reset() {
	mc.toolMutex.Lock()
	mc.sessionMutex.Lock()
	mc.engineMutex.Lock()
	defer mc.toolMutex.Unlock()
	defer mc.sessionMutex.Unlock()
	defer mc.engineMutex.Unlock()

	mc.toolExecutions = make(map[string]*ToolMetrics)
	mc.sessionMetrics = &SessionMetrics{}
	mc.engineMetrics = &EngineMetrics{
		StartTime: time.Now(),
		IsHealthy: true,
	}
	mc.startTime = time.Now()
}