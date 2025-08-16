package device

import (
	"context"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// HealthChecker monitors plugin health
type HealthChecker struct {
	interval time.Duration
	logger   *logrus.Logger
	mutex    sync.RWMutex
	results  map[string]*HealthResult
}

// HealthResult contains the result of a health check
type HealthResult struct {
	PluginName string                 `json:"plugin_name"`
	Health     *Health                `json:"health"`
	Error      error                  `json:"error,omitempty"`
	CheckTime  time.Time              `json:"check_time"`
	Duration   time.Duration          `json:"duration"`
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(interval time.Duration) *HealthChecker {
	return &HealthChecker{
		interval: interval,
		logger:   logrus.New(),
		results:  make(map[string]*HealthResult),
	}
}

// Start starts the health checker
func (h *HealthChecker) Start(ctx context.Context, manager *Manager) {
	ticker := time.NewTicker(h.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			h.checkAllPlugins(manager)
		case <-ctx.Done():
			return
		}
	}
}

// checkAllPlugins checks the health of all running plugins
func (h *HealthChecker) checkAllPlugins(manager *Manager) {
	runningPlugins := manager.GetRunningPlugins()

	for _, pluginName := range runningPlugins {
		go h.checkPlugin(manager, pluginName)
	}
}

// checkPlugin checks the health of a single plugin
func (h *HealthChecker) checkPlugin(manager *Manager, pluginName string) {
	startTime := time.Now()

	instance, err := manager.GetPlugin(pluginName)
	if err != nil {
		h.recordResult(pluginName, nil, err, startTime)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	health, err := instance.Plugin.GetHealth(ctx)
	h.recordResult(pluginName, health, err, startTime)

	// Update plugin state based on health
	if err != nil || (health != nil && health.Status == "critical") {
		instance.mutex.Lock()
		if instance.State == StateRunning {
			instance.State = StateError
			instance.Metrics.ErrorCount++
			instance.Metrics.LastError = "Health check failed"
			instance.Metrics.LastErrorTime = time.Now()
		}
		instance.mutex.Unlock()

		h.logger.WithField("plugin", pluginName).WithError(err).Warn("Plugin health check failed")
	}
}

// recordResult records a health check result
func (h *HealthChecker) recordResult(pluginName string, health *Health, err error, startTime time.Time) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	h.results[pluginName] = &HealthResult{
		PluginName: pluginName,
		Health:     health,
		Error:      err,
		CheckTime:  startTime,
		Duration:   time.Since(startTime),
	}
}

// GetHealth returns the health result for a plugin
func (h *HealthChecker) GetHealth(pluginName string) (*HealthResult, bool) {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	result, exists := h.results[pluginName]
	return result, exists
}

// GetAllHealth returns health results for all checked plugins
func (h *HealthChecker) GetAllHealth() map[string]*HealthResult {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	results := make(map[string]*HealthResult)
	for name, result := range h.results {
		results[name] = result
	}
	return results
}

// MetricsCollector collects plugin metrics
type MetricsCollector struct {
	interval time.Duration
	logger   *logrus.Logger
	mutex    sync.RWMutex
	snapshots map[string]*MetricsSnapshot
}

// MetricsSnapshot represents a snapshot of plugin metrics
type MetricsSnapshot struct {
	PluginName string         `json:"plugin_name"`
	Metrics    *PluginMetrics `json:"metrics"`
	Timestamp  time.Time      `json:"timestamp"`
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector(interval time.Duration) *MetricsCollector {
	return &MetricsCollector{
		interval:  interval,
		logger:    logrus.New(),
		snapshots: make(map[string]*MetricsSnapshot),
	}
}

// Start starts the metrics collector
func (m *MetricsCollector) Start(ctx context.Context, manager *Manager) {
	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.collectAllMetrics(manager)
		case <-ctx.Done():
			return
		}
	}
}

// collectAllMetrics collects metrics from all plugins
func (m *MetricsCollector) collectAllMetrics(manager *Manager) {
	plugins := manager.ListPlugins()

	for name, instance := range plugins {
		go m.collectPluginMetrics(name, instance)
	}
}

// collectPluginMetrics collects metrics from a single plugin
func (m *MetricsCollector) collectPluginMetrics(pluginName string, instance *PluginInstance) {
	instance.mutex.RLock()
	metrics := *instance.Metrics
	metrics.Uptime = time.Since(instance.StartTime)
	instance.mutex.RUnlock()

	m.mutex.Lock()
	m.snapshots[pluginName] = &MetricsSnapshot{
		PluginName: pluginName,
		Metrics:    &metrics,
		Timestamp:  time.Now(),
	}
	m.mutex.Unlock()
}

// GetMetrics returns the latest metrics for a plugin
func (m *MetricsCollector) GetMetrics(pluginName string) (*MetricsSnapshot, bool) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	snapshot, exists := m.snapshots[pluginName]
	return snapshot, exists
}

// GetAllMetrics returns metrics for all plugins
func (m *MetricsCollector) GetAllMetrics() map[string]*MetricsSnapshot {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	snapshots := make(map[string]*MetricsSnapshot)
	for name, snapshot := range m.snapshots {
		snapshots[name] = snapshot
	}
	return snapshots
}