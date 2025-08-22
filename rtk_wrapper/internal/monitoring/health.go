package monitoring

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"rtk_wrapper/internal/registry"
)

// HealthChecker 健康檢查器
type HealthChecker struct {
	mu          sync.RWMutex
	config      HealthConfig
	registry    *registry.Registry
	metrics     *MetricsCollector
	logger      *StructuredLogger
	performance *PerformanceOptimizer
	httpServer  *http.Server
	checks      map[string]HealthCheck
	lastResults map[string]HealthCheckResult
	startTime   time.Time
	active      int32
	ticker      *time.Ticker
}

// HealthConfig 健康檢查配置
type HealthConfig struct {
	Port          int           `yaml:"port" json:"port"`
	Path          string        `yaml:"path" json:"path"`
	CheckInterval time.Duration `yaml:"check_interval" json:"check_interval"`
	Timeout       time.Duration `yaml:"timeout" json:"timeout"`
	EnabledChecks []string      `yaml:"enabled_checks" json:"enabled_checks"`

	// 閾值配置
	CPUThreshold          float64       `yaml:"cpu_threshold" json:"cpu_threshold"`                     // CPU 使用率閾值
	MemoryThreshold       float64       `yaml:"memory_threshold" json:"memory_threshold"`               // 記憶體使用率閾值
	GoroutineThreshold    int           `yaml:"goroutine_threshold" json:"goroutine_threshold"`         // Goroutine 數量閾值
	ResponseTimeThreshold time.Duration `yaml:"response_time_threshold" json:"response_time_threshold"` // 響應時間閾值
	ErrorRateThreshold    float64       `yaml:"error_rate_threshold" json:"error_rate_threshold"`       // 錯誤率閾值

	// MQTT 健康檢查配置
	MQTTCheckTopic    string        `yaml:"mqtt_check_topic" json:"mqtt_check_topic"`
	MQTTCheckInterval time.Duration `yaml:"mqtt_check_interval" json:"mqtt_check_interval"`
	MQTTCheckTimeout  time.Duration `yaml:"mqtt_check_timeout" json:"mqtt_check_timeout"`
}

// HealthCheck 健康檢查接口
type HealthCheck interface {
	Name() string
	Check(ctx context.Context) HealthCheckResult
	Description() string
	Enabled() bool
}

// HealthCheckResult 健康檢查結果
type HealthCheckResult struct {
	Name      string                 `json:"name"`
	Healthy   bool                   `json:"healthy"`
	Message   string                 `json:"message,omitempty"`
	Details   map[string]interface{} `json:"details,omitempty"`
	Duration  time.Duration          `json:"duration"`
	Timestamp time.Time              `json:"timestamp"`
	Error     string                 `json:"error,omitempty"`
}

// HealthStatus 整體健康狀態
type HealthStatus struct {
	Healthy      bool                         `json:"healthy"`
	Timestamp    time.Time                    `json:"timestamp"`
	Uptime       string                       `json:"uptime"`
	Version      string                       `json:"version"`
	Components   map[string]ComponentHealth   `json:"components"`
	CheckResults map[string]HealthCheckResult `json:"check_results,omitempty"`
	SystemInfo   SystemInfo                   `json:"system_info"`
}

// ComponentHealth 組件健康狀態
type ComponentHealth struct {
	Name    string `json:"name"`
	Healthy bool   `json:"healthy"`
	Message string `json:"message,omitempty"`
}

// SystemInfo 系統信息
type SystemInfo struct {
	OS          string  `json:"os"`
	Arch        string  `json:"arch"`
	GoVersion   string  `json:"go_version"`
	CPUs        int     `json:"cpus"`
	Goroutines  int     `json:"goroutines"`
	MemoryUsage int64   `json:"memory_usage"`
	MemoryTotal int64   `json:"memory_total"`
	CPUUsage    float64 `json:"cpu_usage,omitempty"`
}

// NewHealthChecker 創建健康檢查器
func NewHealthChecker(
	config HealthConfig,
	registry *registry.Registry,
	metrics *MetricsCollector,
	logger *StructuredLogger,
	performance *PerformanceOptimizer,
) *HealthChecker {
	hc := &HealthChecker{
		config:      config,
		registry:    registry,
		metrics:     metrics,
		logger:      logger,
		performance: performance,
		checks:      make(map[string]HealthCheck),
		lastResults: make(map[string]HealthCheckResult),
		startTime:   time.Now(),
	}

	// 註冊預設健康檢查
	hc.registerDefaultChecks()

	// 設置 HTTP 服務器
	hc.setupHTTPServer()

	return hc
}

// Start 啟動健康檢查器
func (hc *HealthChecker) Start(ctx context.Context) error {
	if !atomic.CompareAndSwapInt32(&hc.active, 0, 1) {
		return nil
	}

	hc.logger.Info("Health checker starting", map[string]interface{}{
		"port":           hc.config.Port,
		"check_interval": hc.config.CheckInterval,
		"enabled_checks": len(hc.checks),
	})

	// 啟動 HTTP 服務器
	go func() {
		if err := hc.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			hc.logger.Error("Health check HTTP server error", err)
		}
	}()

	// 啟動定期健康檢查
	hc.ticker = time.NewTicker(hc.config.CheckInterval)
	go hc.runPeriodicChecks(ctx)

	hc.logger.Info("Health checker started successfully")
	return nil
}

// Stop 停止健康檢查器
func (hc *HealthChecker) Stop() error {
	if !atomic.CompareAndSwapInt32(&hc.active, 1, 0) {
		return nil
	}

	hc.logger.Info("Health checker stopping")

	// 停止定期檢查
	if hc.ticker != nil {
		hc.ticker.Stop()
	}

	// 停止 HTTP 服務器
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := hc.httpServer.Shutdown(ctx); err != nil {
		hc.logger.Error("Health check HTTP server shutdown error", err)
	}

	hc.logger.Info("Health checker stopped successfully")
	return nil
}

// RegisterCheck 註冊健康檢查
func (hc *HealthChecker) RegisterCheck(check HealthCheck) {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	hc.checks[check.Name()] = check
	hc.logger.Debug("Health check registered", map[string]interface{}{
		"check_name":  check.Name(),
		"description": check.Description(),
	})
}

// runPeriodicChecks 運行定期健康檢查
func (hc *HealthChecker) runPeriodicChecks(ctx context.Context) {
	for {
		select {
		case <-hc.ticker.C:
			hc.runAllChecks(ctx)
		case <-ctx.Done():
			return
		}
	}
}

// runAllChecks 運行所有健康檢查
func (hc *HealthChecker) runAllChecks(ctx context.Context) {
	hc.mu.RLock()
	checks := make(map[string]HealthCheck)
	for name, check := range hc.checks {
		if hc.isCheckEnabled(name) && check.Enabled() {
			checks[name] = check
		}
	}
	hc.mu.RUnlock()

	results := make(map[string]HealthCheckResult)
	var wg sync.WaitGroup

	for name, check := range checks {
		wg.Add(1)
		go func(n string, c HealthCheck) {
			defer wg.Done()

			checkCtx, cancel := context.WithTimeout(ctx, hc.config.Timeout)
			defer cancel()

			start := time.Now()
			result := c.Check(checkCtx)
			result.Duration = time.Since(start)
			result.Timestamp = time.Now()

			results[n] = result
		}(name, check)
	}

	wg.Wait()

	// 更新結果
	hc.mu.Lock()
	for name, result := range results {
		hc.lastResults[name] = result
	}
	hc.mu.Unlock()

	// 記錄警告
	for _, result := range results {
		if !result.Healthy {
			hc.logger.Warn("Health check failed", map[string]interface{}{
				"check_name": result.Name,
				"message":    result.Message,
				"error":      result.Error,
				"duration":   result.Duration.String(),
			})
		}
	}
}

// GetHealthStatus 獲取健康狀態
func (hc *HealthChecker) GetHealthStatus() HealthStatus {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	status := HealthStatus{
		Healthy:      true,
		Timestamp:    time.Now(),
		Uptime:       time.Since(hc.startTime).String(),
		Version:      "1.0.0",
		Components:   make(map[string]ComponentHealth),
		CheckResults: make(map[string]HealthCheckResult),
		SystemInfo:   hc.getSystemInfo(),
	}

	// 複製檢查結果
	for name, result := range hc.lastResults {
		status.CheckResults[name] = result
		if !result.Healthy {
			status.Healthy = false
		}
	}

	// 獲取組件健康狀態
	if hc.performance != nil {
		perfStatus := hc.performance.GetHealthStatus()
		for name, component := range perfStatus.Components {
			status.Components[name] = component
			if !component.Healthy {
				status.Healthy = false
			}
		}
	}

	// 添加基本組件檢查
	if hc.registry != nil {
		regStats := hc.registry.GetRegistryStats()
		regHealth := ComponentHealth{
			Name:    "registry",
			Healthy: regStats["registered_wrappers"].(int) > 0,
			Message: fmt.Sprintf("Wrappers: %d", regStats["registered_wrappers"].(int)),
		}
		status.Components["registry"] = regHealth
	}

	if hc.metrics != nil {
		metricsStats := hc.metrics.GetMetricsSummary()
		metricsHealth := ComponentHealth{
			Name:    "metrics",
			Healthy: metricsStats["uptime"] != "",
			Message: fmt.Sprintf("Uptime: %s", metricsStats["uptime"]),
		}
		status.Components["metrics"] = metricsHealth
	}

	return status
}

// setupHTTPServer 設置 HTTP 服務器
func (hc *HealthChecker) setupHTTPServer() {
	mux := http.NewServeMux()

	// 健康檢查端點
	mux.HandleFunc(hc.config.Path, hc.healthHandler)
	mux.HandleFunc(hc.config.Path+"/detailed", hc.detailedHealthHandler)
	mux.HandleFunc(hc.config.Path+"/ready", hc.readinessHandler)
	mux.HandleFunc(hc.config.Path+"/live", hc.livenessHandler)

	hc.httpServer = &http.Server{
		Addr:    fmt.Sprintf(":%d", hc.config.Port),
		Handler: mux,
	}
}

// healthHandler 基本健康檢查處理器
func (hc *HealthChecker) healthHandler(w http.ResponseWriter, r *http.Request) {
	status := hc.GetHealthStatus()

	w.Header().Set("Content-Type", "application/json")
	if status.Healthy {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	// 簡化響應
	response := map[string]interface{}{
		"healthy":   status.Healthy,
		"timestamp": status.Timestamp,
		"uptime":    status.Uptime,
	}

	json.NewEncoder(w).Encode(response)
}

// detailedHealthHandler 詳細健康檢查處理器
func (hc *HealthChecker) detailedHealthHandler(w http.ResponseWriter, r *http.Request) {
	status := hc.GetHealthStatus()

	w.Header().Set("Content-Type", "application/json")
	if status.Healthy {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	json.NewEncoder(w).Encode(status)
}

// readinessHandler 就緒檢查處理器
func (hc *HealthChecker) readinessHandler(w http.ResponseWriter, r *http.Request) {
	// 檢查關鍵組件是否就緒
	ready := atomic.LoadInt32(&hc.active) == 1

	if hc.registry != nil {
		stats := hc.registry.GetRegistryStats()
		ready = ready && stats["registered_wrappers"].(int) > 0
	}

	w.Header().Set("Content-Type", "application/json")
	if ready {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	response := map[string]interface{}{
		"ready":     ready,
		"timestamp": time.Now(),
	}

	json.NewEncoder(w).Encode(response)
}

// livenessHandler 存活檢查處理器
func (hc *HealthChecker) livenessHandler(w http.ResponseWriter, r *http.Request) {
	alive := atomic.LoadInt32(&hc.active) == 1

	w.Header().Set("Content-Type", "application/json")
	if alive {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	response := map[string]interface{}{
		"alive":     alive,
		"timestamp": time.Now(),
		"uptime":    time.Since(hc.startTime).String(),
	}

	json.NewEncoder(w).Encode(response)
}

// getSystemInfo 獲取系統信息
func (hc *HealthChecker) getSystemInfo() SystemInfo {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	return SystemInfo{
		OS:          runtime.GOOS,
		Arch:        runtime.GOARCH,
		GoVersion:   runtime.Version(),
		CPUs:        runtime.NumCPU(),
		Goroutines:  runtime.NumGoroutine(),
		MemoryUsage: int64(memStats.Alloc),
		MemoryTotal: int64(memStats.Sys),
	}
}

// isCheckEnabled 檢查是否啟用了指定的健康檢查
func (hc *HealthChecker) isCheckEnabled(checkName string) bool {
	if len(hc.config.EnabledChecks) == 0 {
		return true // 如果沒有配置，預設全部啟用
	}

	for _, enabled := range hc.config.EnabledChecks {
		if enabled == checkName {
			return true
		}
	}
	return false
}

// registerDefaultChecks 註冊預設健康檢查
func (hc *HealthChecker) registerDefaultChecks() {
	// CPU 使用率檢查
	hc.RegisterCheck(&CPUHealthCheck{
		threshold: hc.config.CPUThreshold,
	})

	// 記憶體使用檢查
	hc.RegisterCheck(&MemoryHealthCheck{
		threshold: hc.config.MemoryThreshold,
	})

	// Goroutine 數量檢查
	hc.RegisterCheck(&GoroutineHealthCheck{
		threshold: hc.config.GoroutineThreshold,
	})

	// 註冊表檢查
	if hc.registry != nil {
		hc.RegisterCheck(&RegistryHealthCheck{
			registry: hc.registry,
		})
	}

	// 指標檢查
	if hc.metrics != nil {
		hc.RegisterCheck(&MetricsHealthCheck{
			metrics:               hc.metrics,
			responseTimeThreshold: hc.config.ResponseTimeThreshold,
			errorRateThreshold:    hc.config.ErrorRateThreshold,
		})
	}

	// 性能優化器檢查
	if hc.performance != nil {
		hc.RegisterCheck(&PerformanceHealthCheck{
			performance: hc.performance,
		})
	}
}

// GetDefaultHealthConfig 獲取預設健康檢查配置
func GetDefaultHealthConfig() HealthConfig {
	return HealthConfig{
		Port:                  8081,
		Path:                  "/health",
		CheckInterval:         30 * time.Second,
		Timeout:               10 * time.Second,
		EnabledChecks:         []string{"cpu", "memory", "goroutines", "registry", "metrics", "performance"},
		CPUThreshold:          80.0,
		MemoryThreshold:       85.0,
		GoroutineThreshold:    10000,
		ResponseTimeThreshold: 5 * time.Second,
		ErrorRateThreshold:    5.0,
		MQTTCheckTopic:        "rtk/v1/health/ping",
		MQTTCheckInterval:     60 * time.Second,
		MQTTCheckTimeout:      10 * time.Second,
	}
}
