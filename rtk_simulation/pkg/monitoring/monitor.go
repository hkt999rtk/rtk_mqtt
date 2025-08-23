package monitoring

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// PerformanceMonitor 效能監控器
type PerformanceMonitor struct {
	metrics     map[string]*MetricCollector
	alerts      []Alert
	thresholds  map[string]Threshold
	aggregators map[string]*Aggregator
	running     bool
	mu          sync.RWMutex
	logger      *logrus.Entry
	config      *MonitorConfig
}

// MetricCollector 指標收集器
type MetricCollector struct {
	Name       string
	Type       string // counter, gauge, histogram
	Value      float64
	Samples    []Sample
	Tags       map[string]string
	LastUpdate time.Time
	Window     time.Duration
}

// Sample 樣本
type Sample struct {
	Value     float64
	Timestamp time.Time
}

// Alert 警報
type Alert struct {
	ID           string
	MetricName   string
	Severity     string // info, warning, critical
	Condition    string
	Threshold    float64
	CurrentValue float64
	Message      string
	Timestamp    time.Time
	Resolved     bool
}

// Threshold 閾值
type Threshold struct {
	MetricName string
	Warning    float64
	Critical   float64
	Duration   time.Duration
}

// Aggregator 聚合器
type Aggregator struct {
	Type       string // sum, avg, max, min, p50, p95, p99
	Window     time.Duration
	Values     []float64
	LastResult float64
}

// MonitorConfig 監控配置
type MonitorConfig struct {
	CollectInterval   time.Duration
	AggregateInterval time.Duration
	RetentionPeriod   time.Duration
	EnableAlerts      bool
}

// MetricData 指標數據
type MetricData struct {
	Name      string
	Value     float64
	Tags      map[string]string
	Timestamp time.Time
}

// NewPerformanceMonitor 創建新的效能監控器
func NewPerformanceMonitor(config *MonitorConfig) *PerformanceMonitor {
	if config == nil {
		config = &MonitorConfig{
			CollectInterval:   1 * time.Second,
			AggregateInterval: 10 * time.Second,
			RetentionPeriod:   1 * time.Hour,
			EnableAlerts:      true,
		}
	}

	return &PerformanceMonitor{
		metrics:     make(map[string]*MetricCollector),
		alerts:      make([]Alert, 0),
		thresholds:  make(map[string]Threshold),
		aggregators: make(map[string]*Aggregator),
		config:      config,
		logger:      logrus.WithField("component", "performance_monitor"),
	}
}

// Start 啟動監控器
func (pm *PerformanceMonitor) Start(ctx context.Context) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if pm.running {
		return fmt.Errorf("performance monitor is already running")
	}

	pm.running = true
	pm.logger.Info("Starting performance monitor")

	// 初始化預設指標和閾值
	pm.initializeDefaultMetrics()

	// 啟動監控循環
	go pm.collectionLoop(ctx)
	go pm.aggregationLoop(ctx)
	go pm.alertCheckLoop(ctx)
	go pm.cleanupLoop(ctx)

	return nil
}

// Stop 停止監控器
func (pm *PerformanceMonitor) Stop() error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if !pm.running {
		return fmt.Errorf("performance monitor is not running")
	}

	pm.running = false
	pm.logger.Info("Stopping performance monitor")
	return nil
}

// RecordMetric 記錄指標
func (pm *PerformanceMonitor) RecordMetric(name string, value float64, tags map[string]string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	collector, exists := pm.metrics[name]
	if !exists {
		collector = &MetricCollector{
			Name:       name,
			Type:       "gauge",
			Tags:       tags,
			Samples:    make([]Sample, 0),
			Window:     5 * time.Minute,
			LastUpdate: time.Now(),
		}
		pm.metrics[name] = collector
	}

	// 添加樣本
	sample := Sample{
		Value:     value,
		Timestamp: time.Now(),
	}
	collector.Samples = append(collector.Samples, sample)
	collector.Value = value
	collector.LastUpdate = time.Now()

	// 清理舊樣本
	pm.cleanOldSamples(collector)
}

// SetThreshold 設置閾值
func (pm *PerformanceMonitor) SetThreshold(metricName string, warning, critical float64) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	pm.thresholds[metricName] = Threshold{
		MetricName: metricName,
		Warning:    warning,
		Critical:   critical,
		Duration:   30 * time.Second,
	}
}

// initializeDefaultMetrics 初始化預設指標
func (pm *PerformanceMonitor) initializeDefaultMetrics() {
	// 系統指標閾值
	pm.SetThreshold("cpu_usage", 70, 90)
	pm.SetThreshold("memory_usage", 80, 95)
	pm.SetThreshold("disk_usage", 85, 95)

	// 網路指標閾值
	pm.SetThreshold("bandwidth_usage", 80, 95)
	pm.SetThreshold("packet_loss", 1, 5)
	pm.SetThreshold("latency", 100, 500)

	// 設備指標閾值
	pm.SetThreshold("device_temperature", 70, 85)
	pm.SetThreshold("error_rate", 1, 5)
}

// collectionLoop 收集循環
func (pm *PerformanceMonitor) collectionLoop(ctx context.Context) {
	ticker := time.NewTicker(pm.config.CollectInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// 收集系統指標
			pm.collectSystemMetrics()
		}
	}
}

// collectSystemMetrics 收集系統指標
func (pm *PerformanceMonitor) collectSystemMetrics() {
	// 這裡應該實現實際的系統指標收集
	// 目前使用模擬數據

	// CPU 使用率
	pm.RecordMetric("cpu_usage", 30+float64(time.Now().Unix()%40), map[string]string{"type": "system"})

	// 記憶體使用率
	pm.RecordMetric("memory_usage", 40+float64(time.Now().Unix()%30), map[string]string{"type": "system"})

	// 磁碟使用率
	pm.RecordMetric("disk_usage", 60+float64(time.Now().Unix()%20), map[string]string{"type": "system"})
}

// aggregationLoop 聚合循環
func (pm *PerformanceMonitor) aggregationLoop(ctx context.Context) {
	ticker := time.NewTicker(pm.config.AggregateInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			pm.aggregateMetrics()
		}
	}
}

// aggregateMetrics 聚合指標
func (pm *PerformanceMonitor) aggregateMetrics() {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	for name, collector := range pm.metrics {
		if len(collector.Samples) == 0 {
			continue
		}

		// 計算各種聚合值
		aggregatorKey := fmt.Sprintf("%s_avg", name)
		if _, exists := pm.aggregators[aggregatorKey]; !exists {
			pm.aggregators[aggregatorKey] = &Aggregator{
				Type:   "avg",
				Window: 1 * time.Minute,
				Values: make([]float64, 0),
			}
		}

		// 計算平均值
		sum := 0.0
		for _, sample := range collector.Samples {
			sum += sample.Value
		}
		avg := sum / float64(len(collector.Samples))
		pm.aggregators[aggregatorKey].LastResult = avg
	}
}

// alertCheckLoop 警報檢查循環
func (pm *PerformanceMonitor) alertCheckLoop(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if pm.config.EnableAlerts {
				pm.checkAlerts()
			}
		}
	}
}

// checkAlerts 檢查警報
func (pm *PerformanceMonitor) checkAlerts() {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	for metricName, threshold := range pm.thresholds {
		collector, exists := pm.metrics[metricName]
		if !exists {
			continue
		}

		// 檢查是否超過閾值
		if collector.Value > threshold.Critical {
			pm.createAlert(metricName, "critical", threshold.Critical, collector.Value)
		} else if collector.Value > threshold.Warning {
			pm.createAlert(metricName, "warning", threshold.Warning, collector.Value)
		} else {
			pm.resolveAlert(metricName)
		}
	}
}

// createAlert 創建警報
func (pm *PerformanceMonitor) createAlert(metricName, severity string, threshold, value float64) {
	// 檢查是否已存在未解決的警報
	for i, alert := range pm.alerts {
		if alert.MetricName == metricName && !alert.Resolved {
			// 更新現有警報
			pm.alerts[i].CurrentValue = value
			pm.alerts[i].Timestamp = time.Now()
			return
		}
	}

	// 創建新警報
	alert := Alert{
		ID:           fmt.Sprintf("alert_%d", time.Now().UnixNano()),
		MetricName:   metricName,
		Severity:     severity,
		Threshold:    threshold,
		CurrentValue: value,
		Message:      fmt.Sprintf("%s exceeded %s threshold: %.2f > %.2f", metricName, severity, value, threshold),
		Timestamp:    time.Now(),
		Resolved:     false,
	}

	pm.alerts = append(pm.alerts, alert)

	pm.logger.WithFields(logrus.Fields{
		"metric":    metricName,
		"severity":  severity,
		"value":     value,
		"threshold": threshold,
	}).Warn("Alert triggered")
}

// resolveAlert 解決警報
func (pm *PerformanceMonitor) resolveAlert(metricName string) {
	for i, alert := range pm.alerts {
		if alert.MetricName == metricName && !alert.Resolved {
			pm.alerts[i].Resolved = true
			pm.logger.WithField("metric", metricName).Info("Alert resolved")
		}
	}
}

// cleanupLoop 清理循環
func (pm *PerformanceMonitor) cleanupLoop(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			pm.cleanupOldData()
		}
	}
}

// cleanupOldData 清理舊數據
func (pm *PerformanceMonitor) cleanupOldData() {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	cutoff := time.Now().Add(-pm.config.RetentionPeriod)

	// 清理指標樣本
	for _, collector := range pm.metrics {
		pm.cleanOldSamples(collector)
	}

	// 清理舊警報
	newAlerts := make([]Alert, 0)
	for _, alert := range pm.alerts {
		if alert.Timestamp.After(cutoff) || !alert.Resolved {
			newAlerts = append(newAlerts, alert)
		}
	}
	pm.alerts = newAlerts
}

// cleanOldSamples 清理舊樣本
func (pm *PerformanceMonitor) cleanOldSamples(collector *MetricCollector) {
	cutoff := time.Now().Add(-collector.Window)
	newSamples := make([]Sample, 0)

	for _, sample := range collector.Samples {
		if sample.Timestamp.After(cutoff) {
			newSamples = append(newSamples, sample)
		}
	}

	collector.Samples = newSamples
}

// GetMetrics 獲取指標
func (pm *PerformanceMonitor) GetMetrics() map[string]*MetricCollector {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	metrics := make(map[string]*MetricCollector)
	for k, v := range pm.metrics {
		metrics[k] = v
	}
	return metrics
}

// GetAlerts 獲取警報
func (pm *PerformanceMonitor) GetAlerts(includeResolved bool) []Alert {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	alerts := make([]Alert, 0)
	for _, alert := range pm.alerts {
		if !alert.Resolved || includeResolved {
			alerts = append(alerts, alert)
		}
	}
	return alerts
}

// GetMetricHistory 獲取指標歷史
func (pm *PerformanceMonitor) GetMetricHistory(metricName string, duration time.Duration) []Sample {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	collector, exists := pm.metrics[metricName]
	if !exists {
		return []Sample{}
	}

	cutoff := time.Now().Add(-duration)
	history := make([]Sample, 0)

	for _, sample := range collector.Samples {
		if sample.Timestamp.After(cutoff) {
			history = append(history, sample)
		}
	}

	return history
}

// GetStatistics 獲取統計資訊
func (pm *PerformanceMonitor) GetStatistics() map[string]interface{} {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	stats := map[string]interface{}{
		"total_metrics":     len(pm.metrics),
		"active_alerts":     0,
		"total_alerts":      len(pm.alerts),
		"total_thresholds":  len(pm.thresholds),
		"total_aggregators": len(pm.aggregators),
	}

	// 統計活動警報
	for _, alert := range pm.alerts {
		if !alert.Resolved {
			stats["active_alerts"] = stats["active_alerts"].(int) + 1
		}
	}

	return stats
}
