package monitoring

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"rtk_wrapper/internal/registry"
	"rtk_wrapper/pkg/types"
)

// Monitor 綜合監控系統
type Monitor struct {
	mu              sync.RWMutex
	config          MonitorConfig
	metrics         *MetricsCollector
	logger          *StructuredLogger
	performance     *PerformanceOptimizer
	health          *HealthChecker
	registry        *registry.Registry
	startTime       time.Time
	active          int32
	shutdownTimeout time.Duration
}

// MonitorConfig 監控配置
type MonitorConfig struct {
	// 指標收集配置
	Metrics MetricsConfig `yaml:"metrics" json:"metrics"`

	// 日誌配置
	Logging LogConfig `yaml:"logging" json:"logging"`

	// 性能優化配置
	Performance PerformanceConfig `yaml:"performance" json:"performance"`

	// 健康檢查配置
	Health HealthConfig `yaml:"health" json:"health"`

	// 通用配置
	ShutdownTimeout time.Duration `yaml:"shutdown_timeout" json:"shutdown_timeout"`

	// 功能開關
	EnableMetrics     bool `yaml:"enable_metrics" json:"enable_metrics"`
	EnableLogging     bool `yaml:"enable_logging" json:"enable_logging"`
	EnablePerformance bool `yaml:"enable_performance" json:"enable_performance"`
	EnableHealth      bool `yaml:"enable_health" json:"enable_health"`
}

// NewMonitor 創建監控系統
func NewMonitor(config MonitorConfig, registry *registry.Registry) (*Monitor, error) {
	monitor := &Monitor{
		config:          config,
		registry:        registry,
		startTime:       time.Now(),
		shutdownTimeout: config.ShutdownTimeout,
	}

	var err error

	// 初始化指標收集器
	if config.EnableMetrics {
		monitor.metrics = NewMetricsCollector()
	}

	// 初始化日誌系統
	if config.EnableLogging {
		monitor.logger, err = NewStructuredLogger(config.Logging, monitor.metrics)
		if err != nil {
			return nil, fmt.Errorf("failed to create logger: %w", err)
		}
	}

	// 初始化性能優化器
	if config.EnablePerformance {
		monitor.performance = NewPerformanceOptimizer(config.Performance, monitor.metrics, monitor.logger)
	}

	// 初始化健康檢查器
	if config.EnableHealth {
		monitor.health = NewHealthChecker(config.Health, registry, monitor.metrics, monitor.logger, monitor.performance)
	}

	return monitor, nil
}

// Start 啟動監控系統
func (m *Monitor) Start(ctx context.Context) error {
	if !atomic.CompareAndSwapInt32(&m.active, 0, 1) {
		return fmt.Errorf("monitor already started")
	}

	if m.logger != nil {
		m.logger.Info("Monitor system starting")
	}

	// 啟動指標收集器
	if m.metrics != nil {
		if err := m.metrics.Start(); err != nil {
			return fmt.Errorf("failed to start metrics collector: %w", err)
		}
	}

	// 啟動性能優化器
	if m.performance != nil {
		if err := m.performance.Start(ctx); err != nil {
			return fmt.Errorf("failed to start performance optimizer: %w", err)
		}
	}

	// 啟動健康檢查器
	if m.health != nil {
		if err := m.health.Start(ctx); err != nil {
			return fmt.Errorf("failed to start health checker: %w", err)
		}
	}

	if m.logger != nil {
		m.logger.Info("Monitor system started successfully", map[string]interface{}{
			"metrics":     m.config.EnableMetrics,
			"logging":     m.config.EnableLogging,
			"performance": m.config.EnablePerformance,
			"health":      m.config.EnableHealth,
		})
	}

	return nil
}

// Stop 停止監控系統
func (m *Monitor) Stop() error {
	if !atomic.CompareAndSwapInt32(&m.active, 1, 0) {
		return fmt.Errorf("monitor not running")
	}

	if m.logger != nil {
		m.logger.Info("Monitor system stopping")
	}

	// 創建帶超時的上下文
	_, cancel := context.WithTimeout(context.Background(), m.shutdownTimeout)
	defer cancel()

	var shutdownErrors []error

	// 停止健康檢查器
	if m.health != nil {
		if err := m.health.Stop(); err != nil {
			shutdownErrors = append(shutdownErrors, fmt.Errorf("health checker shutdown error: %w", err))
		}
	}

	// 停止性能優化器
	if m.performance != nil {
		if err := m.performance.Stop(); err != nil {
			shutdownErrors = append(shutdownErrors, fmt.Errorf("performance optimizer shutdown error: %w", err))
		}
	}

	// 停止指標收集器
	if m.metrics != nil {
		if err := m.metrics.Stop(); err != nil {
			shutdownErrors = append(shutdownErrors, fmt.Errorf("metrics collector shutdown error: %w", err))
		}
	}

	if len(shutdownErrors) > 0 {
		errorMsg := "Monitor shutdown completed with errors:"
		for _, err := range shutdownErrors {
			errorMsg += fmt.Sprintf("\n  - %v", err)
		}
		if m.logger != nil {
			m.logger.Warn(errorMsg)
		}
		return fmt.Errorf(errorMsg)
	}

	if m.logger != nil {
		m.logger.Info("Monitor system stopped successfully")
	}

	return nil
}

// GetStatus 獲取監控系統狀態
func (m *Monitor) GetStatus() MonitorStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()

	status := MonitorStatus{
		Active:     atomic.LoadInt32(&m.active) == 1,
		StartTime:  m.startTime,
		Uptime:     time.Since(m.startTime).String(),
		Components: map[string]ComponentStatus{},
	}

	// 獲取各組件狀態
	if m.metrics != nil {
		status.Components["metrics"] = ComponentStatus{
			Name:    "metrics",
			Enabled: m.config.EnableMetrics,
			Healthy: true,
			Details: m.metrics.GetMetricsSummary(),
		}
	}

	if m.logger != nil {
		status.Components["logger"] = ComponentStatus{
			Name:    "logger",
			Enabled: m.config.EnableLogging,
			Healthy: true,
			Details: m.logger.GetLogStats(),
		}
	}

	if m.performance != nil {
		perfStats := m.performance.GetPerformanceStats()
		perfHealthy := perfStats["optimization_active"].(bool)
		status.Components["performance"] = ComponentStatus{
			Name:    "performance",
			Enabled: m.config.EnablePerformance,
			Healthy: perfHealthy,
			Details: perfStats,
		}
	}

	if m.health != nil {
		healthStatus := m.health.GetHealthStatus()
		status.Components["health"] = ComponentStatus{
			Name:    "health",
			Enabled: m.config.EnableHealth,
			Healthy: healthStatus.Healthy,
			Details: map[string]interface{}{
				"health_status": healthStatus,
			},
		}
	}

	return status
}

// MonitorStatus 監控系統狀態
type MonitorStatus struct {
	Active     bool                       `json:"active"`
	StartTime  time.Time                  `json:"start_time"`
	Uptime     string                     `json:"uptime"`
	Components map[string]ComponentStatus `json:"components"`
}

// ComponentStatus 組件狀態
type ComponentStatus struct {
	Name    string                 `json:"name"`
	Enabled bool                   `json:"enabled"`
	Healthy bool                   `json:"healthy"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// GetMetrics 獲取指標收集器
func (m *Monitor) GetMetrics() *MetricsCollector {
	return m.metrics
}

// GetLogger 獲取日誌記錄器
func (m *Monitor) GetLogger() *StructuredLogger {
	return m.logger
}

// GetPerformanceOptimizer 獲取性能優化器
func (m *Monitor) GetPerformanceOptimizer() *PerformanceOptimizer {
	return m.performance
}

// GetHealthChecker 獲取健康檢查器
func (m *Monitor) GetHealthChecker() *HealthChecker {
	return m.health
}

// RecordMessage 記錄訊息處理
func (m *Monitor) RecordMessage(wrapper, direction, topic string, payloadSize int, processingTime time.Duration, success bool, err error) {
	// 記錄指標
	if m.metrics != nil {
		if success {
			isUplink := direction == "uplink" || direction == types.DirectionUplink.String()
			m.metrics.RecordMessageProcessed(wrapper, topic, "", "", isUplink, processingTime)
		} else {
			m.metrics.RecordTransformError(wrapper, direction)
		}
	}

	// 記錄日誌
	if m.logger != nil {
		ctx := m.logger.NewLogContext()
		ctx.WrapperName = wrapper
		if direction == "uplink" || direction == types.DirectionUplink.String() {
			ctx.Direction = types.DirectionUplink
		} else {
			ctx.Direction = types.DirectionDownlink
		}
		ctx.Topic = topic
		ctx.PayloadSize = int64(payloadSize)

		messageLogger := m.logger.NewMessageLogger(ctx)

		if success {
			messageLogger.LogMessageProcessed(processingTime)
		} else {
			messageLogger.LogTransformError(err, topic)
		}

		m.logger.ReleaseLogContext(ctx)
	}
}

// RecordWrapperRegistration 記錄包裝器註冊
func (m *Monitor) RecordWrapperRegistration(wrapperName string, success bool) {
	if m.logger != nil {
		if success {
			m.logger.Info("Wrapper registered successfully", map[string]interface{}{
				"wrapper_name": wrapperName,
			})
		} else {
			m.logger.Error("Wrapper registration failed", nil, map[string]interface{}{
				"wrapper_name": wrapperName,
			})
		}
	}
}

// OptimizeMemory 執行記憶體優化
func (m *Monitor) OptimizeMemory() {
	if m.performance != nil {
		m.performance.OptimizeMemory()
	}
}

// IsHealthy 檢查系統是否健康
func (m *Monitor) IsHealthy() bool {
	if m.health != nil {
		return m.health.GetHealthStatus().Healthy
	}
	return atomic.LoadInt32(&m.active) == 1
}

// GetSystemMetrics 獲取系統指標摘要
func (m *Monitor) GetSystemMetrics() map[string]interface{} {
	summary := map[string]interface{}{
		"monitor_active": atomic.LoadInt32(&m.active) == 1,
		"uptime":         time.Since(m.startTime).String(),
	}

	if m.metrics != nil {
		metricsSummary := m.metrics.GetMetricsSummary()
		for k, v := range metricsSummary {
			summary[fmt.Sprintf("metrics_%s", k)] = v
		}
	}

	if m.performance != nil {
		perfStats := m.performance.GetPerformanceStats()
		for k, v := range perfStats {
			summary[fmt.Sprintf("performance_%s", k)] = v
		}
	}

	if m.health != nil {
		summary["system_healthy"] = m.health.GetHealthStatus().Healthy
	}

	return summary
}

// GetDefaultMonitorConfig 獲取預設監控配置
func GetDefaultMonitorConfig() MonitorConfig {
	return MonitorConfig{
		Metrics:           GetDefaultMetricsConfig(),
		Logging:           GetDefaultLogConfig(),
		Performance:       GetDefaultPerformanceConfig(),
		Health:            GetDefaultHealthConfig(),
		ShutdownTimeout:   30 * time.Second,
		EnableMetrics:     true,
		EnableLogging:     true,
		EnablePerformance: true,
		EnableHealth:      true,
	}
}

// ValidateMonitorConfig 驗證監控配置
func ValidateMonitorConfig(config MonitorConfig) error {
	if config.ShutdownTimeout <= 0 {
		return fmt.Errorf("shutdown timeout must be positive")
	}

	if !config.EnableMetrics && !config.EnableLogging && !config.EnablePerformance && !config.EnableHealth {
		return fmt.Errorf("at least one monitoring component must be enabled")
	}

	return nil
}
