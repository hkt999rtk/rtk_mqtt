package monitoring

import (
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// MetricsCollector 定義指標收集器
type MetricsCollector struct {
	mu                 sync.RWMutex
	startTime          time.Time
	messagesProcessed  int64
	uplinkMessages     int64
	downlinkMessages   int64
	transformErrors    int64
	validationErrors   int64
	totalLatency       int64
	latencyCount       int64
	activeWrappers     int32
	wrapperStats       map[string]*WrapperMetrics
	errorStats         map[string]int64
	topicStats         map[string]int64
	deviceStats        map[string]*DeviceMetrics
	performanceHistory []PerformanceSnapshot
	maxHistorySize     int
}

// WrapperMetrics 定義 wrapper 專用指標
type WrapperMetrics struct {
	Name              string    `json:"name"`
	MessagesProcessed int64     `json:"messages_processed"`
	UplinkMessages    int64     `json:"uplink_messages"`
	DownlinkMessages  int64     `json:"downlink_messages"`
	TransformErrors   int64     `json:"transform_errors"`
	ValidationErrors  int64     `json:"validation_errors"`
	AverageLatency    float64   `json:"average_latency_ms"`
	LastActive        time.Time `json:"last_active"`
	ErrorRate         float64   `json:"error_rate"`
}

// DeviceMetrics 定義設備專用指標
type DeviceMetrics struct {
	DeviceID          string    `json:"device_id"`
	DeviceType        string    `json:"device_type"`
	MessagesProcessed int64     `json:"messages_processed"`
	LastSeen          time.Time `json:"last_seen"`
	Status            string    `json:"status"`
	UplinkCount       int64     `json:"uplink_count"`
	DownlinkCount     int64     `json:"downlink_count"`
}

// PerformanceSnapshot 定義性能快照
type PerformanceSnapshot struct {
	Timestamp      time.Time `json:"timestamp"`
	MessagesPerSec float64   `json:"messages_per_sec"`
	AverageLatency float64   `json:"average_latency_ms"`
	MemoryUsage    uint64    `json:"memory_usage_bytes"`
	GoroutineCount int       `json:"goroutine_count"`
	ErrorRate      float64   `json:"error_rate"`
}

// SystemMetrics 定義系統指標
type SystemMetrics struct {
	Uptime             time.Duration              `json:"uptime"`
	MessagesProcessed  int64                      `json:"messages_processed"`
	UplinkMessages     int64                      `json:"uplink_messages"`
	DownlinkMessages   int64                      `json:"downlink_messages"`
	TransformErrors    int64                      `json:"transform_errors"`
	ValidationErrors   int64                      `json:"validation_errors"`
	AverageLatency     float64                    `json:"average_latency_ms"`
	MessagesPerSecond  float64                    `json:"messages_per_second"`
	ActiveWrappers     int32                      `json:"active_wrappers"`
	ErrorRate          float64                    `json:"error_rate"`
	MemoryStats        runtime.MemStats           `json:"memory_stats"`
	GoroutineCount     int                        `json:"goroutine_count"`
	WrapperStats       map[string]*WrapperMetrics `json:"wrapper_stats"`
	DeviceStats        map[string]*DeviceMetrics  `json:"device_stats"`
	TopicStats         map[string]int64           `json:"topic_stats"`
	ErrorStats         map[string]int64           `json:"error_stats"`
	PerformanceHistory []PerformanceSnapshot      `json:"performance_history"`
}

// NewMetricsCollector 創建新的指標收集器
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		startTime:          time.Now(),
		wrapperStats:       make(map[string]*WrapperMetrics),
		errorStats:         make(map[string]int64),
		topicStats:         make(map[string]int64),
		deviceStats:        make(map[string]*DeviceMetrics),
		performanceHistory: make([]PerformanceSnapshot, 0),
		maxHistorySize:     100, // 保留最近 100 個快照
	}
}

// RecordMessageProcessed 記錄訊息處理
func (mc *MetricsCollector) RecordMessageProcessed(wrapperName, topic, deviceID, deviceType string, isUplink bool, latency time.Duration) {
	atomic.AddInt64(&mc.messagesProcessed, 1)
	atomic.AddInt64(&mc.totalLatency, latency.Nanoseconds()/1000000) // 轉換為毫秒
	atomic.AddInt64(&mc.latencyCount, 1)

	if isUplink {
		atomic.AddInt64(&mc.uplinkMessages, 1)
	} else {
		atomic.AddInt64(&mc.downlinkMessages, 1)
	}

	mc.mu.Lock()
	defer mc.mu.Unlock()

	// 更新 wrapper 統計
	if wrapperStats, exists := mc.wrapperStats[wrapperName]; exists {
		wrapperStats.MessagesProcessed++
		if isUplink {
			wrapperStats.UplinkMessages++
		} else {
			wrapperStats.DownlinkMessages++
		}
		wrapperStats.LastActive = time.Now()

		// 更新平均延遲
		if wrapperStats.MessagesProcessed > 0 {
			totalLatency := wrapperStats.AverageLatency * float64(wrapperStats.MessagesProcessed-1)
			wrapperStats.AverageLatency = (totalLatency + float64(latency.Nanoseconds()/1000000)) / float64(wrapperStats.MessagesProcessed)
		}
	} else {
		mc.wrapperStats[wrapperName] = &WrapperMetrics{
			Name:              wrapperName,
			MessagesProcessed: 1,
			UplinkMessages: func() int64 {
				if isUplink {
					return 1
				}
				return 0
			}(),
			DownlinkMessages: func() int64 {
				if !isUplink {
					return 1
				}
				return 0
			}(),
			AverageLatency: float64(latency.Nanoseconds() / 1000000),
			LastActive:     time.Now(),
		}
	}

	// 更新 topic 統計
	mc.topicStats[topic]++

	// 更新設備統計
	if deviceID != "" {
		if deviceStats, exists := mc.deviceStats[deviceID]; exists {
			deviceStats.MessagesProcessed++
			deviceStats.LastSeen = time.Now()
			deviceStats.Status = "active"
			if isUplink {
				deviceStats.UplinkCount++
			} else {
				deviceStats.DownlinkCount++
			}
		} else {
			mc.deviceStats[deviceID] = &DeviceMetrics{
				DeviceID:          deviceID,
				DeviceType:        deviceType,
				MessagesProcessed: 1,
				LastSeen:          time.Now(),
				Status:            "active",
				UplinkCount: func() int64 {
					if isUplink {
						return 1
					}
					return 0
				}(),
				DownlinkCount: func() int64 {
					if !isUplink {
						return 1
					}
					return 0
				}(),
			}
		}
	}
}

// RecordTransformError 記錄轉換錯誤
func (mc *MetricsCollector) RecordTransformError(wrapperName, errorType string) {
	atomic.AddInt64(&mc.transformErrors, 1)

	mc.mu.Lock()
	defer mc.mu.Unlock()

	// 更新 wrapper 錯誤統計
	if wrapperStats, exists := mc.wrapperStats[wrapperName]; exists {
		wrapperStats.TransformErrors++
		if wrapperStats.MessagesProcessed > 0 {
			wrapperStats.ErrorRate = float64(wrapperStats.TransformErrors+wrapperStats.ValidationErrors) / float64(wrapperStats.MessagesProcessed)
		}
	}

	// 更新錯誤類型統計
	errorKey := fmt.Sprintf("%s:%s", wrapperName, errorType)
	mc.errorStats[errorKey]++
}

// RecordValidationError 記錄驗證錯誤
func (mc *MetricsCollector) RecordValidationError(wrapperName, errorType string) {
	atomic.AddInt64(&mc.validationErrors, 1)

	mc.mu.Lock()
	defer mc.mu.Unlock()

	// 更新 wrapper 錯誤統計
	if wrapperStats, exists := mc.wrapperStats[wrapperName]; exists {
		wrapperStats.ValidationErrors++
		if wrapperStats.MessagesProcessed > 0 {
			wrapperStats.ErrorRate = float64(wrapperStats.TransformErrors+wrapperStats.ValidationErrors) / float64(wrapperStats.MessagesProcessed)
		}
	}

	// 更新錯誤類型統計
	errorKey := fmt.Sprintf("%s:validation:%s", wrapperName, errorType)
	mc.errorStats[errorKey]++
}

// SetActiveWrappers 設置活躍 wrapper 數量
func (mc *MetricsCollector) SetActiveWrappers(count int32) {
	atomic.StoreInt32(&mc.activeWrappers, count)
}

// GetMetrics 獲取所有指標
func (mc *MetricsCollector) GetMetrics() *SystemMetrics {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	totalMessages := atomic.LoadInt64(&mc.messagesProcessed)
	totalLatency := atomic.LoadInt64(&mc.totalLatency)
	latencyCount := atomic.LoadInt64(&mc.latencyCount)

	var averageLatency float64
	if latencyCount > 0 {
		averageLatency = float64(totalLatency) / float64(latencyCount)
	}

	uptime := time.Since(mc.startTime)
	var messagesPerSecond float64
	if uptime.Seconds() > 0 {
		messagesPerSecond = float64(totalMessages) / uptime.Seconds()
	}

	totalErrors := atomic.LoadInt64(&mc.transformErrors) + atomic.LoadInt64(&mc.validationErrors)
	var errorRate float64
	if totalMessages > 0 {
		errorRate = float64(totalErrors) / float64(totalMessages)
	}

	// 深拷貝 map 以避免併發問題
	wrapperStatsCopy := make(map[string]*WrapperMetrics)
	for k, v := range mc.wrapperStats {
		wrapperStatsCopy[k] = &WrapperMetrics{
			Name:              v.Name,
			MessagesProcessed: v.MessagesProcessed,
			UplinkMessages:    v.UplinkMessages,
			DownlinkMessages:  v.DownlinkMessages,
			TransformErrors:   v.TransformErrors,
			ValidationErrors:  v.ValidationErrors,
			AverageLatency:    v.AverageLatency,
			LastActive:        v.LastActive,
			ErrorRate:         v.ErrorRate,
		}
	}

	deviceStatsCopy := make(map[string]*DeviceMetrics)
	for k, v := range mc.deviceStats {
		deviceStatsCopy[k] = &DeviceMetrics{
			DeviceID:          v.DeviceID,
			DeviceType:        v.DeviceType,
			MessagesProcessed: v.MessagesProcessed,
			LastSeen:          v.LastSeen,
			Status:            v.Status,
			UplinkCount:       v.UplinkCount,
			DownlinkCount:     v.DownlinkCount,
		}
	}

	topicStatsCopy := make(map[string]int64)
	for k, v := range mc.topicStats {
		topicStatsCopy[k] = v
	}

	errorStatsCopy := make(map[string]int64)
	for k, v := range mc.errorStats {
		errorStatsCopy[k] = v
	}

	// 複製性能歷史
	historyLength := len(mc.performanceHistory)
	historyCopy := make([]PerformanceSnapshot, historyLength)
	copy(historyCopy, mc.performanceHistory)

	return &SystemMetrics{
		Uptime:             uptime,
		MessagesProcessed:  totalMessages,
		UplinkMessages:     atomic.LoadInt64(&mc.uplinkMessages),
		DownlinkMessages:   atomic.LoadInt64(&mc.downlinkMessages),
		TransformErrors:    atomic.LoadInt64(&mc.transformErrors),
		ValidationErrors:   atomic.LoadInt64(&mc.validationErrors),
		AverageLatency:     averageLatency,
		MessagesPerSecond:  messagesPerSecond,
		ActiveWrappers:     atomic.LoadInt32(&mc.activeWrappers),
		ErrorRate:          errorRate,
		MemoryStats:        memStats,
		GoroutineCount:     runtime.NumGoroutine(),
		WrapperStats:       wrapperStatsCopy,
		DeviceStats:        deviceStatsCopy,
		TopicStats:         topicStatsCopy,
		ErrorStats:         errorStatsCopy,
		PerformanceHistory: historyCopy,
	}
}

// CollectPerformanceSnapshot 收集性能快照
func (mc *MetricsCollector) CollectPerformanceSnapshot() {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	metrics := mc.GetMetrics()

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	snapshot := PerformanceSnapshot{
		Timestamp:      time.Now(),
		MessagesPerSec: metrics.MessagesPerSecond,
		AverageLatency: metrics.AverageLatency,
		MemoryUsage:    memStats.Alloc,
		GoroutineCount: runtime.NumGoroutine(),
		ErrorRate:      metrics.ErrorRate,
	}

	mc.performanceHistory = append(mc.performanceHistory, snapshot)

	// 限制歷史記錄大小
	if len(mc.performanceHistory) > mc.maxHistorySize {
		mc.performanceHistory = mc.performanceHistory[1:]
	}
}

// GetPerformanceHistory 獲取性能歷史
func (mc *MetricsCollector) GetPerformanceHistory(minutes int) []PerformanceSnapshot {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	if minutes <= 0 {
		// 返回所有歷史記錄
		result := make([]PerformanceSnapshot, len(mc.performanceHistory))
		copy(result, mc.performanceHistory)
		return result
	}

	// 返回指定時間範圍內的記錄
	cutoff := time.Now().Add(-time.Duration(minutes) * time.Minute)
	var result []PerformanceSnapshot

	for _, snapshot := range mc.performanceHistory {
		if snapshot.Timestamp.After(cutoff) {
			result = append(result, snapshot)
		}
	}

	return result
}

// Reset 重置所有指標
func (mc *MetricsCollector) Reset() {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	atomic.StoreInt64(&mc.messagesProcessed, 0)
	atomic.StoreInt64(&mc.uplinkMessages, 0)
	atomic.StoreInt64(&mc.downlinkMessages, 0)
	atomic.StoreInt64(&mc.transformErrors, 0)
	atomic.StoreInt64(&mc.validationErrors, 0)
	atomic.StoreInt64(&mc.totalLatency, 0)
	atomic.StoreInt64(&mc.latencyCount, 0)

	mc.startTime = time.Now()
	mc.wrapperStats = make(map[string]*WrapperMetrics)
	mc.errorStats = make(map[string]int64)
	mc.topicStats = make(map[string]int64)
	mc.deviceStats = make(map[string]*DeviceMetrics)
	mc.performanceHistory = make([]PerformanceSnapshot, 0)
}

// StartPerformanceCollector 啟動性能收集器
func (mc *MetricsCollector) StartPerformanceCollector(interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			mc.CollectPerformanceSnapshot()
		}
	}()
}

// HTTPHandler 提供 HTTP 接口查看指標
func (mc *MetricsCollector) HTTPHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		metrics := mc.GetMetrics()

		if err := json.NewEncoder(w).Encode(metrics); err != nil {
			http.Error(w, fmt.Sprintf("Failed to encode metrics: %v", err), http.StatusInternalServerError)
			return
		}
	}
}

// HealthCheckHandler 提供健康檢查接口
func (mc *MetricsCollector) HealthCheckHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		metrics := mc.GetMetrics()

		// 簡單的健康檢查邏輯
		status := "healthy"
		if metrics.ErrorRate > 0.10 { // 錯誤率超過 10%
			status = "degraded"
		}
		if metrics.ActiveWrappers == 0 {
			status = "unhealthy"
		}

		health := map[string]interface{}{
			"status":             status,
			"uptime":             metrics.Uptime.String(),
			"messages_processed": metrics.MessagesProcessed,
			"active_wrappers":    metrics.ActiveWrappers,
			"error_rate":         metrics.ErrorRate,
			"memory_usage_mb":    metrics.MemoryStats.Alloc / 1024 / 1024,
			"goroutine_count":    metrics.GoroutineCount,
		}

		statusCode := http.StatusOK
		if status == "degraded" {
			statusCode = http.StatusPartialContent
		} else if status == "unhealthy" {
			statusCode = http.StatusServiceUnavailable
		}

		w.WriteHeader(statusCode)
		json.NewEncoder(w).Encode(health)
	}
}

// GetMetricsSummary 獲取指標摘要
func (mc *MetricsCollector) GetMetricsSummary() map[string]interface{} {
	metrics := mc.GetMetrics()

	return map[string]interface{}{
		"total_messages_processed": metrics.MessagesProcessed,
		"uplink_messages":          metrics.UplinkMessages,
		"downlink_messages":        metrics.DownlinkMessages,
		"average_response_time":    time.Duration(metrics.AverageLatency * float64(time.Millisecond)),
		"error_rate":               metrics.ErrorRate,
		"active_wrappers":          metrics.ActiveWrappers,
		"uptime":                   metrics.Uptime.String(),
		"messages_per_second":      metrics.MessagesPerSecond,
		"transform_errors":         metrics.TransformErrors,
		"validation_errors":        metrics.ValidationErrors,
		"memory_usage_mb":          metrics.MemoryStats.Alloc / 1024 / 1024,
		"goroutine_count":          metrics.GoroutineCount,
	}
}

// RecordLogMessage 記錄日誌訊息
func (mc *MetricsCollector) RecordLogMessage(level string) {
	// 這是一個簡化實現，可以根據需要擴展
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if mc.errorStats == nil {
		mc.errorStats = make(map[string]int64)
	}

	mc.errorStats[fmt.Sprintf("log_%s", level)]++
}

// MetricsConfig 指標配置
type MetricsConfig struct {
	Port            int           `yaml:"port" json:"port"`
	Path            string        `yaml:"path" json:"path"`
	CollectInterval time.Duration `yaml:"collect_interval" json:"collect_interval"`
	HistorySize     int           `yaml:"history_size" json:"history_size"`
	EnableHTTP      bool          `yaml:"enable_http" json:"enable_http"`
}

// Start 啟動指標收集器
func (mc *MetricsCollector) Start() error {
	// 啟動性能收集器
	mc.StartPerformanceCollector(30 * time.Second)
	return nil
}

// Stop 停止指標收集器
func (mc *MetricsCollector) Stop() error {
	// 簡化實現，實際可能需要停止收集器
	return nil
}

// GetDefaultMetricsConfig 獲取預設指標配置
func GetDefaultMetricsConfig() MetricsConfig {
	return MetricsConfig{
		Port:            8080,
		Path:            "/metrics",
		CollectInterval: 30 * time.Second,
		HistorySize:     1440, // 24 hours of minute snapshots
		EnableHTTP:      true,
	}
}
