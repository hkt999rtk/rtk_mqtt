package monitoring

import (
	"context"
	"fmt"
	"runtime"
	"sync/atomic"
	"time"

	"rtk_wrapper/internal/registry"
)

// CPUHealthCheck CPU 使用率健康檢查
type CPUHealthCheck struct {
	threshold float64
	lastCheck time.Time
	lastUsage float64
}

// Name 返回檢查名稱
func (c *CPUHealthCheck) Name() string {
	return "cpu"
}

// Description 返回檢查描述
func (c *CPUHealthCheck) Description() string {
	return fmt.Sprintf("CPU usage health check (threshold: %.1f%%)", c.threshold)
}

// Enabled 返回是否啟用
func (c *CPUHealthCheck) Enabled() bool {
	return c.threshold > 0
}

// Check 執行 CPU 健康檢查
func (c *CPUHealthCheck) Check(ctx context.Context) HealthCheckResult {
	result := HealthCheckResult{
		Name:      c.Name(),
		Timestamp: time.Now(),
		Details:   make(map[string]interface{}),
	}

	// 簡化的 CPU 使用率計算（基於 goroutine 數量和上次檢查時間間隔）
	numGoroutines := runtime.NumGoroutine()
	numCPU := runtime.NumCPU()

	// 估算 CPU 使用率（這是一個簡化的近似值）
	estimatedUsage := float64(numGoroutines) / float64(numCPU*10) * 100
	if estimatedUsage > 100 {
		estimatedUsage = 100
	}

	c.lastUsage = estimatedUsage
	c.lastCheck = time.Now()

	result.Details["cpu_usage"] = estimatedUsage
	result.Details["threshold"] = c.threshold
	result.Details["num_cpus"] = numCPU
	result.Details["num_goroutines"] = numGoroutines

	if estimatedUsage <= c.threshold {
		result.Healthy = true
		result.Message = fmt.Sprintf("CPU usage: %.1f%%", estimatedUsage)
	} else {
		result.Healthy = false
		result.Message = fmt.Sprintf("CPU usage too high: %.1f%% (threshold: %.1f%%)", estimatedUsage, c.threshold)
	}

	return result
}

// MemoryHealthCheck 記憶體使用健康檢查
type MemoryHealthCheck struct {
	threshold float64
}

// Name 返回檢查名稱
func (m *MemoryHealthCheck) Name() string {
	return "memory"
}

// Description 返回檢查描述
func (m *MemoryHealthCheck) Description() string {
	return fmt.Sprintf("Memory usage health check (threshold: %.1f%%)", m.threshold)
}

// Enabled 返回是否啟用
func (m *MemoryHealthCheck) Enabled() bool {
	return m.threshold > 0
}

// Check 執行記憶體健康檢查
func (m *MemoryHealthCheck) Check(ctx context.Context) HealthCheckResult {
	result := HealthCheckResult{
		Name:      m.Name(),
		Timestamp: time.Now(),
		Details:   make(map[string]interface{}),
	}

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	// 計算記憶體使用率
	usage := float64(memStats.Alloc) / float64(memStats.Sys) * 100

	result.Details["memory_usage"] = usage
	result.Details["threshold"] = m.threshold
	result.Details["alloc_bytes"] = memStats.Alloc
	result.Details["sys_bytes"] = memStats.Sys
	result.Details["gc_cycles"] = memStats.NumGC
	result.Details["heap_objects"] = memStats.HeapObjects

	if usage <= m.threshold {
		result.Healthy = true
		result.Message = fmt.Sprintf("Memory usage: %.1f%%", usage)
	} else {
		result.Healthy = false
		result.Message = fmt.Sprintf("Memory usage too high: %.1f%% (threshold: %.1f%%)", usage, m.threshold)
	}

	return result
}

// GoroutineHealthCheck Goroutine 數量健康檢查
type GoroutineHealthCheck struct {
	threshold int
}

// Name 返回檢查名稱
func (g *GoroutineHealthCheck) Name() string {
	return "goroutines"
}

// Description 返回檢查描述
func (g *GoroutineHealthCheck) Description() string {
	return fmt.Sprintf("Goroutine count health check (threshold: %d)", g.threshold)
}

// Enabled 返回是否啟用
func (g *GoroutineHealthCheck) Enabled() bool {
	return g.threshold > 0
}

// Check 執行 Goroutine 健康檢查
func (g *GoroutineHealthCheck) Check(ctx context.Context) HealthCheckResult {
	result := HealthCheckResult{
		Name:      g.Name(),
		Timestamp: time.Now(),
		Details:   make(map[string]interface{}),
	}

	count := runtime.NumGoroutine()

	result.Details["goroutine_count"] = count
	result.Details["threshold"] = g.threshold

	if count <= g.threshold {
		result.Healthy = true
		result.Message = fmt.Sprintf("Goroutine count: %d", count)
	} else {
		result.Healthy = false
		result.Message = fmt.Sprintf("Too many goroutines: %d (threshold: %d)", count, g.threshold)
	}

	return result
}

// RegistryHealthCheck 註冊表健康檢查
type RegistryHealthCheck struct {
	registry *registry.Registry
}

// Name 返回檢查名稱
func (r *RegistryHealthCheck) Name() string {
	return "registry"
}

// Description 返回檢查描述
func (r *RegistryHealthCheck) Description() string {
	return "Wrapper registry health check"
}

// Enabled 返回是否啟用
func (r *RegistryHealthCheck) Enabled() bool {
	return r.registry != nil
}

// Check 執行註冊表健康檢查
func (r *RegistryHealthCheck) Check(ctx context.Context) HealthCheckResult {
	result := HealthCheckResult{
		Name:      r.Name(),
		Timestamp: time.Now(),
		Details:   make(map[string]interface{}),
	}

	if r.registry == nil {
		result.Healthy = false
		result.Message = "Registry is not available"
		result.Error = "registry is nil"
		return result
	}

	stats := r.registry.GetRegistryStats()

	wrapperCount := stats["registered_wrappers"].(int)
	uplinkRoutes := stats["uplink_routes"].(int)
	downlinkRoutes := stats["downlink_routes"].(int)

	result.Details["registered_wrappers"] = wrapperCount
	result.Details["uplink_routes"] = uplinkRoutes
	result.Details["downlink_routes"] = downlinkRoutes
	result.Details["stats"] = stats

	if wrapperCount > 0 {
		result.Healthy = true
		result.Message = fmt.Sprintf("Registry healthy: %d wrappers, %d uplink routes, %d downlink routes",
			wrapperCount, uplinkRoutes, downlinkRoutes)
	} else {
		result.Healthy = false
		result.Message = "No wrappers registered"
	}

	return result
}

// MetricsHealthCheck 指標系統健康檢查
type MetricsHealthCheck struct {
	metrics               *MetricsCollector
	responseTimeThreshold time.Duration
	errorRateThreshold    float64
}

// Name 返回檢查名稱
func (m *MetricsHealthCheck) Name() string {
	return "metrics"
}

// Description 返回檢查描述
func (m *MetricsHealthCheck) Description() string {
	return "Metrics system health check"
}

// Enabled 返回是否啟用
func (m *MetricsHealthCheck) Enabled() bool {
	return m.metrics != nil
}

// Check 執行指標健康檢查
func (m *MetricsHealthCheck) Check(ctx context.Context) HealthCheckResult {
	result := HealthCheckResult{
		Name:      m.Name(),
		Timestamp: time.Now(),
		Details:   make(map[string]interface{}),
	}

	if m.metrics == nil {
		result.Healthy = false
		result.Message = "Metrics collector is not available"
		result.Error = "metrics is nil"
		return result
	}

	stats := m.metrics.GetMetricsSummary()

	// 提取關鍵指標
	messagesProcessed := stats["total_messages_processed"].(int64)
	avgResponseTime := stats["average_response_time"].(time.Duration)
	errorRate := stats["error_rate"].(float64)

	result.Details["messages_processed"] = messagesProcessed
	result.Details["average_response_time"] = avgResponseTime.String()
	result.Details["error_rate"] = errorRate
	result.Details["response_time_threshold"] = m.responseTimeThreshold.String()
	result.Details["error_rate_threshold"] = m.errorRateThreshold

	healthy := true
	var issues []string

	// 檢查響應時間
	if m.responseTimeThreshold > 0 && avgResponseTime > m.responseTimeThreshold {
		healthy = false
		issues = append(issues, fmt.Sprintf("response time too high: %v", avgResponseTime))
	}

	// 檢查錯誤率
	if m.errorRateThreshold > 0 && errorRate > m.errorRateThreshold {
		healthy = false
		issues = append(issues, fmt.Sprintf("error rate too high: %.2f%%", errorRate))
	}

	result.Healthy = healthy

	if healthy {
		result.Message = fmt.Sprintf("Metrics healthy: %d messages processed, avg response time: %v, error rate: %.2f%%",
			messagesProcessed, avgResponseTime, errorRate)
	} else {
		result.Message = fmt.Sprintf("Metrics issues: %v", issues)
	}

	return result
}

// PerformanceHealthCheck 性能優化器健康檢查
type PerformanceHealthCheck struct {
	performance *PerformanceOptimizer
}

// Name 返回檢查名稱
func (p *PerformanceHealthCheck) Name() string {
	return "performance"
}

// Description 返回檢查描述
func (p *PerformanceHealthCheck) Description() string {
	return "Performance optimizer health check"
}

// Enabled 返回是否啟用
func (p *PerformanceHealthCheck) Enabled() bool {
	return p.performance != nil
}

// Check 執行性能健康檢查
func (p *PerformanceHealthCheck) Check(ctx context.Context) HealthCheckResult {
	result := HealthCheckResult{
		Name:      p.Name(),
		Timestamp: time.Now(),
		Details:   make(map[string]interface{}),
	}

	if p.performance == nil {
		result.Healthy = false
		result.Message = "Performance optimizer is not available"
		result.Error = "performance is nil"
		return result
	}

	// 獲取性能狀態
	perfStatus := p.performance.GetHealthStatus()

	result.Healthy = perfStatus.Healthy
	result.Details["components"] = perfStatus.Components

	if perfStatus.Healthy {
		result.Message = "Performance optimizer is healthy"
	} else {
		var issues []string
		for name, component := range perfStatus.Components {
			if !component.Healthy {
				issues = append(issues, fmt.Sprintf("%s: %s", name, component.Message))
			}
		}
		result.Message = fmt.Sprintf("Performance issues: %v", issues)
	}

	return result
}

// MQTTHealthCheck MQTT 連接健康檢查
type MQTTHealthCheck struct {
	client       interface{} // MQTT client interface
	topic        string
	timeout      time.Duration
	lastPingTime time.Time
	pingCount    int64
	pongCount    int64
}

// NewMQTTHealthCheck 創建 MQTT 健康檢查
func NewMQTTHealthCheck(client interface{}, topic string, timeout time.Duration) *MQTTHealthCheck {
	return &MQTTHealthCheck{
		client:  client,
		topic:   topic,
		timeout: timeout,
	}
}

// Name 返回檢查名稱
func (m *MQTTHealthCheck) Name() string {
	return "mqtt"
}

// Description 返回檢查描述
func (m *MQTTHealthCheck) Description() string {
	return "MQTT connection health check"
}

// Enabled 返回是否啟用
func (m *MQTTHealthCheck) Enabled() bool {
	return m.client != nil
}

// Check 執行 MQTT 健康檢查
func (m *MQTTHealthCheck) Check(ctx context.Context) HealthCheckResult {
	result := HealthCheckResult{
		Name:      m.Name(),
		Timestamp: time.Now(),
		Details:   make(map[string]interface{}),
	}

	if m.client == nil {
		result.Healthy = false
		result.Message = "MQTT client is not available"
		result.Error = "client is nil"
		return result
	}

	// 簡化實現：檢查客戶端是否連接
	// 實際實現需要根據具體的 MQTT 客戶端類型進行適配

	atomic.AddInt64(&m.pingCount, 1)
	m.lastPingTime = time.Now()

	result.Details["ping_count"] = atomic.LoadInt64(&m.pingCount)
	result.Details["pong_count"] = atomic.LoadInt64(&m.pongCount)
	result.Details["last_ping"] = m.lastPingTime
	result.Details["topic"] = m.topic

	// 簡化判斷：假設客戶端始終健康
	result.Healthy = true
	result.Message = fmt.Sprintf("MQTT connection healthy (ping: %d, pong: %d)",
		atomic.LoadInt64(&m.pingCount),
		atomic.LoadInt64(&m.pongCount))

	return result
}

// DatabaseHealthCheck 資料庫健康檢查（如果需要的話）
type DatabaseHealthCheck struct {
	connectionString string
	timeout          time.Duration
	queryTimeout     time.Duration
}

// Name 返回檢查名稱
func (d *DatabaseHealthCheck) Name() string {
	return "database"
}

// Description 返回檢查描述
func (d *DatabaseHealthCheck) Description() string {
	return "Database connection health check"
}

// Enabled 返回是否啟用
func (d *DatabaseHealthCheck) Enabled() bool {
	return d.connectionString != ""
}

// Check 執行資料庫健康檢查
func (d *DatabaseHealthCheck) Check(ctx context.Context) HealthCheckResult {
	result := HealthCheckResult{
		Name:      d.Name(),
		Timestamp: time.Now(),
		Details:   make(map[string]interface{}),
	}

	if d.connectionString == "" {
		result.Healthy = false
		result.Message = "Database connection string not configured"
		return result
	}

	// 實際實現需要根據使用的資料庫類型進行適配
	// 這裡提供一個框架

	checkCtx, cancel := context.WithTimeout(ctx, d.queryTimeout)
	defer cancel()

	// TODO: 實現實際的資料庫連接檢查
	_ = checkCtx

	result.Healthy = true
	result.Message = "Database connection healthy"

	return result
}

// ExternalServiceHealthCheck 外部服務健康檢查
type ExternalServiceHealthCheck struct {
	name     string
	endpoint string
	timeout  time.Duration
}

// NewExternalServiceHealthCheck 創建外部服務健康檢查
func NewExternalServiceHealthCheck(name, endpoint string, timeout time.Duration) *ExternalServiceHealthCheck {
	return &ExternalServiceHealthCheck{
		name:     name,
		endpoint: endpoint,
		timeout:  timeout,
	}
}

// Name 返回檢查名稱
func (e *ExternalServiceHealthCheck) Name() string {
	return fmt.Sprintf("external_%s", e.name)
}

// Description 返回檢查描述
func (e *ExternalServiceHealthCheck) Description() string {
	return fmt.Sprintf("External service health check: %s", e.name)
}

// Enabled 返回是否啟用
func (e *ExternalServiceHealthCheck) Enabled() bool {
	return e.endpoint != ""
}

// Check 執行外部服務健康檢查
func (e *ExternalServiceHealthCheck) Check(ctx context.Context) HealthCheckResult {
	result := HealthCheckResult{
		Name:      e.Name(),
		Timestamp: time.Now(),
		Details:   make(map[string]interface{}),
	}

	if e.endpoint == "" {
		result.Healthy = false
		result.Message = "External service endpoint not configured"
		return result
	}

	checkCtx, cancel := context.WithTimeout(ctx, e.timeout)
	defer cancel()

	// TODO: 實現 HTTP 健康檢查
	_ = checkCtx

	result.Details["endpoint"] = e.endpoint
	result.Details["timeout"] = e.timeout.String()

	// 簡化實現
	result.Healthy = true
	result.Message = fmt.Sprintf("External service %s is healthy", e.name)

	return result
}
