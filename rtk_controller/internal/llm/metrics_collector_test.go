package llm

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMetricsCollector_NewMetricsCollector(t *testing.T) {
	mc := NewMetricsCollector()

	assert.NotNil(t, mc)
	assert.NotNil(t, mc.toolExecutions)
	assert.NotNil(t, mc.sessionMetrics)
	assert.NotNil(t, mc.engineMetrics)
	assert.True(t, mc.engineMetrics.IsHealthy)
	assert.NotZero(t, mc.startTime)
}

func TestMetricsCollector_RecordToolExecution(t *testing.T) {
	mc := NewMetricsCollector()
	toolName := "test.tool"
	duration := 100 * time.Millisecond

	// Record successful execution
	mc.RecordToolExecution(toolName, duration, true, "")

	metrics := mc.GetToolMetrics(toolName)
	assert.NotNil(t, metrics)
	assert.Equal(t, int64(1), metrics.TotalExecutions)
	assert.Equal(t, int64(1), metrics.SuccessfulExecs)
	assert.Equal(t, int64(0), metrics.FailedExecs)
	assert.Equal(t, duration, metrics.AvgExecutionTime)
	assert.Equal(t, duration, metrics.MinExecutionTime)
	assert.Equal(t, duration, metrics.MaxExecutionTime)

	// Record failed execution
	mc.RecordToolExecution(toolName, 200*time.Millisecond, false, "test error")

	metrics = mc.GetToolMetrics(toolName)
	assert.Equal(t, int64(2), metrics.TotalExecutions)
	assert.Equal(t, int64(1), metrics.SuccessfulExecs)
	assert.Equal(t, int64(1), metrics.FailedExecs)
	assert.Equal(t, "test error", metrics.LastError)
	assert.Equal(t, 150*time.Millisecond, metrics.AvgExecutionTime)
	assert.Equal(t, duration, metrics.MinExecutionTime)
	assert.Equal(t, 200*time.Millisecond, metrics.MaxExecutionTime)
}

func TestMetricsCollector_RecordToolTimeout(t *testing.T) {
	mc := NewMetricsCollector()
	toolName := "timeout.tool"

	mc.RecordToolTimeout(toolName)

	metrics := mc.GetToolMetrics(toolName)
	assert.NotNil(t, metrics)
	assert.Equal(t, int64(1), metrics.TimeoutExecs)
	assert.Equal(t, "Tool execution timeout", metrics.LastError)
	assert.NotZero(t, metrics.LastErrorTime)
}

func TestMetricsCollector_SessionMetrics(t *testing.T) {
	mc := NewMetricsCollector()

	// Test session creation
	mc.RecordSessionCreated()
	sessionMetrics := mc.GetSessionMetrics()
	assert.Equal(t, int64(1), sessionMetrics.TotalSessions)
	assert.Equal(t, int64(1), sessionMetrics.ActiveSessions)
	assert.Equal(t, int64(1), sessionMetrics.MaxConcurrentSessions)

	// Create another session
	mc.RecordSessionCreated()
	sessionMetrics = mc.GetSessionMetrics()
	assert.Equal(t, int64(2), sessionMetrics.TotalSessions)
	assert.Equal(t, int64(2), sessionMetrics.ActiveSessions)
	assert.Equal(t, int64(2), sessionMetrics.MaxConcurrentSessions)

	// Complete one session
	mc.RecordSessionCompleted(5*time.Second, true)
	sessionMetrics = mc.GetSessionMetrics()
	assert.Equal(t, int64(1), sessionMetrics.ActiveSessions)
	assert.Equal(t, int64(1), sessionMetrics.CompletedSessions)
	assert.Equal(t, 5*time.Second, sessionMetrics.AvgSessionTime)

	// Cancel the other session
	mc.RecordSessionCancelled()
	sessionMetrics = mc.GetSessionMetrics()
	assert.Equal(t, int64(0), sessionMetrics.ActiveSessions)
	assert.Equal(t, int64(1), sessionMetrics.CancelledSessions)
}

func TestMetricsCollector_EngineMetrics(t *testing.T) {
	mc := NewMetricsCollector()

	mc.UpdateEngineMetrics(5, 1024*1024, 50)

	engineMetrics := mc.GetEngineMetrics()
	assert.Equal(t, 5, engineMetrics.RegisteredTools)
	assert.Equal(t, int64(1024*1024), engineMetrics.MemoryUsageBytes)
	assert.Equal(t, 50, engineMetrics.GoroutineCount)
	assert.True(t, engineMetrics.Uptime > 0)
}

func TestMetricsCollector_HealthStatus(t *testing.T) {
	mc := NewMetricsCollector()

	// Update health status to set last health check time
	mc.UpdateHealthStatus(true)
	summary := mc.GetMetricsSummary()
	assert.Equal(t, "healthy", summary.HealthStatus)

	// Mark as unhealthy
	mc.UpdateHealthStatus(false)
	summary = mc.GetMetricsSummary()
	assert.Equal(t, "unhealthy", summary.HealthStatus)

	// Mark as healthy but with high error rate
	mc.UpdateHealthStatus(true)
	// Simulate high error rate by adding failed executions
	for i := 0; i < 20; i++ {
		mc.RecordToolExecution("error.tool", 100*time.Millisecond, false, "error")
	}
	mc.RecordToolExecution("success.tool", 100*time.Millisecond, true, "")
	
	// Update engine metrics to trigger error rate calculation
	mc.UpdateEngineMetrics(2, 1024*1024, 10)

	summary = mc.GetMetricsSummary()
	assert.Equal(t, "degraded", summary.HealthStatus)
}

func TestMetricsCollector_GetMetricsSummary(t *testing.T) {
	mc := NewMetricsCollector()

	// Add some test data
	mc.RecordToolExecution("tool1", 100*time.Millisecond, true, "")
	mc.RecordToolExecution("tool2", 200*time.Millisecond, true, "")
	mc.RecordToolExecution("tool1", 150*time.Millisecond, false, "error")
	mc.RecordSessionCreated()
	mc.RecordSessionCompleted(5*time.Second, true)
	mc.UpdateEngineMetrics(2, 1024*1024, 25)

	summary := mc.GetMetricsSummary()

	assert.NotNil(t, summary)
	assert.NotZero(t, summary.Timestamp)
	assert.NotNil(t, summary.Engine)
	assert.NotNil(t, summary.Sessions)
	assert.Len(t, summary.Tools, 2)
	assert.Len(t, summary.TopTools, 2)

	// Check that tool1 is ranked higher (more executions)
	assert.Equal(t, "tool1", summary.TopTools[0].ToolName)
	assert.Equal(t, int64(2), summary.TopTools[0].ExecutionCount)
	assert.Equal(t, 0.5, summary.TopTools[0].SuccessRate)
}

func TestMetricsCollector_PrometheusMetrics(t *testing.T) {
	mc := NewMetricsCollector()

	// Add some test data
	mc.RecordToolExecution("test.tool", 100*time.Millisecond, true, "")
	mc.RecordSessionCreated()
	mc.UpdateEngineMetrics(1, 1024*1024, 10)

	prometheus := mc.GetPrometheusMetrics()

	assert.Contains(t, prometheus, "llm_engine_uptime_seconds")
	assert.Contains(t, prometheus, "llm_engine_registered_tools")
	assert.Contains(t, prometheus, "llm_sessions_total")
	assert.Contains(t, prometheus, "llm_sessions_active")
	assert.Contains(t, prometheus, "llm_tool_executions_total")
	assert.Contains(t, prometheus, "tool=\"test.tool\"")
}

func TestMetricsCollector_Reset(t *testing.T) {
	mc := NewMetricsCollector()

	// Add some data
	mc.RecordToolExecution("test.tool", 100*time.Millisecond, true, "")
	mc.RecordSessionCreated()
	mc.UpdateEngineMetrics(1, 1024*1024, 10)

	// Verify data exists
	assert.NotNil(t, mc.GetToolMetrics("test.tool"))
	assert.Equal(t, int64(1), mc.GetSessionMetrics().TotalSessions)

	// Reset and verify clean state
	mc.Reset()

	assert.Nil(t, mc.GetToolMetrics("test.tool"))
	assert.Equal(t, int64(0), mc.GetSessionMetrics().TotalSessions)
	assert.True(t, mc.GetEngineMetrics().IsHealthy)
}

func TestMetricsCollector_ConcurrentAccess(t *testing.T) {
	mc := NewMetricsCollector()
	
	// Test concurrent access doesn't panic
	done := make(chan bool, 10)
	
	for i := 0; i < 10; i++ {
		go func(id int) {
			defer func() { done <- true }()
			
			toolName := fmt.Sprintf("tool%d", id%3)
			for j := 0; j < 100; j++ {
				mc.RecordToolExecution(toolName, time.Millisecond, true, "")
				mc.RecordSessionCreated()
				mc.RecordSessionCompleted(time.Second, true)
				_ = mc.GetMetricsSummary()
			}
		}(i)
	}
	
	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
	
	// Verify final state is consistent
	summary := mc.GetMetricsSummary()
	assert.NotNil(t, summary)
	assert.True(t, summary.Sessions.TotalSessions > 0)
	assert.Len(t, summary.Tools, 3) // tool0, tool1, tool2
}

// Benchmark tests
func BenchmarkMetricsCollector_RecordToolExecution(b *testing.B) {
	mc := NewMetricsCollector()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mc.RecordToolExecution("bench.tool", 100*time.Millisecond, true, "")
	}
}

func BenchmarkMetricsCollector_GetMetricsSummary(b *testing.B) {
	mc := NewMetricsCollector()
	
	// Pre-populate with some data
	for i := 0; i < 1000; i++ {
		mc.RecordToolExecution("tool1", 100*time.Millisecond, true, "")
		mc.RecordToolExecution("tool2", 200*time.Millisecond, true, "")
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = mc.GetMetricsSummary()
	}
}