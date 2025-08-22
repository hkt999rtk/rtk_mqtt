package monitoring

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"rtk_wrapper/pkg/types"
)

// PerformanceOptimizer 性能優化器
type PerformanceOptimizer struct {
	mu                 sync.RWMutex
	metrics            *MetricsCollector
	logger             *StructuredLogger
	config             PerformanceConfig
	workerPool         *WorkerPool
	messageBuffer      *MessageBuffer
	circuitBreaker     *CircuitBreaker
	rateLimiter        *RateLimiter
	memoryManager      *MemoryManager
	connectionPool     *ConnectionPool
	startTime          time.Time
	optimizationActive int32
}

// PerformanceConfig 性能配置
type PerformanceConfig struct {
	// 工作池配置
	WorkerPoolSize  int           `yaml:"worker_pool_size" json:"worker_pool_size"`
	WorkerQueueSize int           `yaml:"worker_queue_size" json:"worker_queue_size"`
	WorkerTimeout   time.Duration `yaml:"worker_timeout" json:"worker_timeout"`

	// 訊息緩衝配置
	MessageBufferSize int           `yaml:"message_buffer_size" json:"message_buffer_size"`
	BatchSize         int           `yaml:"batch_size" json:"batch_size"`
	FlushInterval     time.Duration `yaml:"flush_interval" json:"flush_interval"`

	// 熔斷器配置
	CircuitBreakerThreshold    int           `yaml:"circuit_breaker_threshold" json:"circuit_breaker_threshold"`
	CircuitBreakerTimeout      time.Duration `yaml:"circuit_breaker_timeout" json:"circuit_breaker_timeout"`
	CircuitBreakerResetTimeout time.Duration `yaml:"circuit_breaker_reset_timeout" json:"circuit_breaker_reset_timeout"`

	// 限流器配置
	RateLimit       int           `yaml:"rate_limit" json:"rate_limit"`   // 每秒請求數
	BurstLimit      int           `yaml:"burst_limit" json:"burst_limit"` // 突發限制
	RateLimitWindow time.Duration `yaml:"rate_limit_window" json:"rate_limit_window"`

	// 記憶體管理配置
	GCInterval      time.Duration `yaml:"gc_interval" json:"gc_interval"`
	MemoryThreshold float64       `yaml:"memory_threshold" json:"memory_threshold"` // 記憶體使用閾值 (%)
	MaxMemoryUsage  int64         `yaml:"max_memory_usage" json:"max_memory_usage"` // 最大記憶體使用 (MB)

	// 連接池配置
	MaxConnections    int           `yaml:"max_connections" json:"max_connections"`
	ConnectionTimeout time.Duration `yaml:"connection_timeout" json:"connection_timeout"`
	IdleTimeout       time.Duration `yaml:"idle_timeout" json:"idle_timeout"`
}

// WorkerPool 工作池
type WorkerPool struct {
	workers    []Worker
	jobQueue   chan Job
	workerPool chan chan Job
	quit       chan bool
	mu         sync.RWMutex
	stats      WorkerPoolStats
}

// Worker 工作者
type Worker struct {
	id         int
	jobChannel chan Job
	workerPool chan chan Job
	quit       chan bool
	stats      WorkerStats
}

// Job 工作任務
type Job struct {
	ID        string
	Type      JobType
	Message   *types.RTKMessage
	Wrapper   string
	Direction types.MessageDirection
	Context   context.Context
	Callback  func(result JobResult)
	StartTime time.Time
}

// JobType 任務類型
type JobType string

const (
	JobTypeTransform JobType = "transform"
	JobTypeValidate  JobType = "validate"
	JobTypeRoute     JobType = "route"
	JobTypeBatch     JobType = "batch"
)

// JobResult 任務結果
type JobResult struct {
	Success        bool
	Result         interface{}
	Error          error
	ProcessingTime time.Duration
	WorkerID       int
}

// WorkerPoolStats 工作池統計
type WorkerPoolStats struct {
	TotalJobs     int64
	CompletedJobs int64
	FailedJobs    int64
	ActiveJobs    int64
	QueueLength   int64
	AverageTime   time.Duration
}

// WorkerStats 工作者統計
type WorkerStats struct {
	ID            int
	JobsProcessed int64
	JobsSucceeded int64
	JobsFailed    int64
	AverageTime   time.Duration
	LastJobTime   time.Time
}

// MessageBuffer 訊息緩衝
type MessageBuffer struct {
	mu            sync.RWMutex
	buffer        []*types.RTKMessage
	maxSize       int
	batchSize     int
	flushInterval time.Duration
	flushTimer    *time.Timer
	callback      func([]*types.RTKMessage)
	stats         BufferStats
}

// BufferStats 緩衝統計
type BufferStats struct {
	TotalMessages    int64
	BufferedMessages int64
	BatchesProcessed int64
	FlushCount       int64
	OverflowCount    int64
	AverageBatchSize float64
}

// CircuitBreaker 熔斷器
type CircuitBreaker struct {
	mu           sync.RWMutex
	state        CircuitBreakerState
	failureCount int64
	successCount int64
	threshold    int
	timeout      time.Duration
	resetTimeout time.Duration
	lastFailTime time.Time
	stats        CircuitBreakerStats
}

// CircuitBreakerState 熔斷器狀態
type CircuitBreakerState int

const (
	CircuitBreakerClosed CircuitBreakerState = iota
	CircuitBreakerOpen
	CircuitBreakerHalfOpen
)

// CircuitBreakerStats 熔斷器統計
type CircuitBreakerStats struct {
	State            CircuitBreakerState
	FailureCount     int64
	SuccessCount     int64
	TotalRequests    int64
	RejectedRequests int64
	LastFailTime     time.Time
	StateChanges     int64
}

// RateLimiter 限流器
type RateLimiter struct {
	mu         sync.RWMutex
	rate       int
	burst      int
	window     time.Duration
	tokens     int
	lastRefill time.Time
	requests   []time.Time
	stats      RateLimiterStats
}

// RateLimiterStats 限流器統計
type RateLimiterStats struct {
	TotalRequests    int64
	AllowedRequests  int64
	RejectedRequests int64
	CurrentTokens    int
	RefillCount      int64
	AverageRate      float64
}

// MemoryManager 記憶體管理器
type MemoryManager struct {
	mu          sync.RWMutex
	threshold   float64
	maxUsage    int64
	gcInterval  time.Duration
	lastGC      time.Time
	gcTicker    *time.Ticker
	stats       MemoryStats
	memoryPools map[string]*sync.Pool
}

// MemoryStats 記憶體統計
type MemoryStats struct {
	CurrentUsage      int64
	MaxUsage          int64
	GCCount           int64
	LastGCTime        time.Time
	GCDuration        time.Duration
	PoolHits          int64
	PoolMisses        int64
	AllocationCount   int64
	DeallocationCount int64
}

// ConnectionPool 連接池
type ConnectionPool struct {
	mu             sync.RWMutex
	connections    map[string]*Connection
	maxConnections int
	timeout        time.Duration
	idleTimeout    time.Duration
	cleanupTicker  *time.Ticker
	stats          ConnectionPoolStats
}

// Connection 連接
type Connection struct {
	ID         string
	CreatedAt  time.Time
	LastUsed   time.Time
	InUse      bool
	UseCount   int64
	ErrorCount int64
}

// ConnectionPoolStats 連接池統計
type ConnectionPoolStats struct {
	TotalConnections     int
	ActiveConnections    int
	IdleConnections      int
	ConnectionsCreated   int64
	ConnectionsDestroyed int64
	ConnectionHits       int64
	ConnectionMisses     int64
	AverageUseCount      float64
}

// NewPerformanceOptimizer 創建性能優化器
func NewPerformanceOptimizer(config PerformanceConfig, metrics *MetricsCollector, logger *StructuredLogger) *PerformanceOptimizer {
	po := &PerformanceOptimizer{
		config:    config,
		metrics:   metrics,
		logger:    logger,
		startTime: time.Now(),
	}

	// 初始化組件
	po.workerPool = NewWorkerPool(config.WorkerPoolSize, config.WorkerQueueSize, config.WorkerTimeout)
	po.messageBuffer = NewMessageBuffer(config.MessageBufferSize, config.BatchSize, config.FlushInterval)
	po.circuitBreaker = NewCircuitBreaker(config.CircuitBreakerThreshold, config.CircuitBreakerTimeout, config.CircuitBreakerResetTimeout)
	po.rateLimiter = NewRateLimiter(config.RateLimit, config.BurstLimit, config.RateLimitWindow)
	po.memoryManager = NewMemoryManager(config.MemoryThreshold, config.MaxMemoryUsage, config.GCInterval)
	po.connectionPool = NewConnectionPool(config.MaxConnections, config.ConnectionTimeout, config.IdleTimeout)

	return po
}

// Start 啟動性能優化器
func (po *PerformanceOptimizer) Start(ctx context.Context) error {
	if !atomic.CompareAndSwapInt32(&po.optimizationActive, 0, 1) {
		return nil
	}

	po.logger.Info("Performance optimizer starting")

	// 啟動各個組件
	if err := po.workerPool.Start(); err != nil {
		return err
	}

	if err := po.messageBuffer.Start(); err != nil {
		return err
	}

	po.memoryManager.Start()
	po.connectionPool.Start()

	po.logger.Info("Performance optimizer started successfully")
	return nil
}

// Stop 停止性能優化器
func (po *PerformanceOptimizer) Stop() error {
	if !atomic.CompareAndSwapInt32(&po.optimizationActive, 1, 0) {
		return nil
	}

	po.logger.Info("Performance optimizer stopping")

	// 停止各個組件
	po.workerPool.Stop()
	po.messageBuffer.Stop()
	po.memoryManager.Stop()
	po.connectionPool.Stop()

	po.logger.Info("Performance optimizer stopped successfully")
	return nil
}

// SubmitJob 提交工作任務
func (po *PerformanceOptimizer) SubmitJob(job Job) error {
	if atomic.LoadInt32(&po.optimizationActive) == 0 {
		return fmt.Errorf("performance optimizer not active")
	}

	// 檢查熔斷器
	if !po.circuitBreaker.Allow() {
		return fmt.Errorf("circuit breaker open")
	}

	// 檢查限流
	if !po.rateLimiter.Allow() {
		return fmt.Errorf("rate limit exceeded")
	}

	return po.workerPool.Submit(job)
}

// BufferMessage 緩衝訊息
func (po *PerformanceOptimizer) BufferMessage(message *types.RTKMessage) error {
	if atomic.LoadInt32(&po.optimizationActive) == 0 {
		return fmt.Errorf("performance optimizer not active")
	}

	return po.messageBuffer.Add(message)
}

// GetPerformanceStats 獲取性能統計
func (po *PerformanceOptimizer) GetPerformanceStats() map[string]interface{} {
	po.mu.RLock()
	defer po.mu.RUnlock()

	return map[string]interface{}{
		"uptime":              time.Since(po.startTime).String(),
		"optimization_active": atomic.LoadInt32(&po.optimizationActive) == 1,
		"worker_pool":         po.workerPool.GetStats(),
		"message_buffer":      po.messageBuffer.GetStats(),
		"circuit_breaker":     po.circuitBreaker.GetStats(),
		"rate_limiter":        po.rateLimiter.GetStats(),
		"memory_manager":      po.memoryManager.GetStats(),
		"connection_pool":     po.connectionPool.GetStats(),
	}
}

// OptimizeMemory 執行記憶體優化
func (po *PerformanceOptimizer) OptimizeMemory() {
	po.logger.Debug("Starting memory optimization")

	// 強制垃圾回收
	runtime.GC()

	// 清理記憶體池
	po.memoryManager.Cleanup()

	// 清理連接池
	po.connectionPool.Cleanup()

	po.logger.Debug("Memory optimization completed")
}

// GetHealthStatus 獲取健康狀態
func (po *PerformanceOptimizer) GetHealthStatus() HealthStatus {
	status := HealthStatus{
		Healthy:    true,
		Timestamp:  time.Now(),
		Components: make(map[string]ComponentHealth),
	}

	// 檢查工作池健康狀態
	workerStats := po.workerPool.GetStats()
	workerHealth := ComponentHealth{
		Name:    "worker_pool",
		Healthy: workerStats.QueueLength < int64(po.config.WorkerQueueSize*80/100), // 80% 閾值
		Message: fmt.Sprintf("Queue length: %d", workerStats.QueueLength),
	}
	status.Components["worker_pool"] = workerHealth

	// 檢查熔斷器狀態
	cbStats := po.circuitBreaker.GetStats()
	cbHealth := ComponentHealth{
		Name:    "circuit_breaker",
		Healthy: cbStats.State != CircuitBreakerOpen,
		Message: fmt.Sprintf("State: %v", cbStats.State),
	}
	status.Components["circuit_breaker"] = cbHealth

	// 檢查記憶體使用
	memStats := po.memoryManager.GetStats()
	memoryThresholdReached := float64(memStats.CurrentUsage)/float64(po.config.MaxMemoryUsage*1024*1024) > po.config.MemoryThreshold
	memHealth := ComponentHealth{
		Name:    "memory_manager",
		Healthy: !memoryThresholdReached,
		Message: fmt.Sprintf("Usage: %d MB", memStats.CurrentUsage/1024/1024),
	}
	status.Components["memory_manager"] = memHealth

	// 計算總體健康狀態
	for _, component := range status.Components {
		if !component.Healthy {
			status.Healthy = false
			break
		}
	}

	return status
}

// NewWorkerPool 創建工作池
func NewWorkerPool(size, queueSize int, timeout time.Duration) *WorkerPool {
	wp := &WorkerPool{
		jobQueue:   make(chan Job, queueSize),
		workerPool: make(chan chan Job, size),
		quit:       make(chan bool),
	}

	// 創建工作者
	wp.workers = make([]Worker, size)
	for i := 0; i < size; i++ {
		worker := Worker{
			id:         i,
			jobChannel: make(chan Job),
			workerPool: wp.workerPool,
			quit:       make(chan bool),
		}
		wp.workers[i] = worker
	}

	return wp
}

// Start 啟動工作池
func (wp *WorkerPool) Start() error {
	// 啟動所有工作者
	for i := range wp.workers {
		go wp.workers[i].Start()
	}

	// 啟動分發器
	go wp.dispatcher()

	return nil
}

// Stop 停止工作池
func (wp *WorkerPool) Stop() {
	close(wp.quit)

	// 停止所有工作者
	for i := range wp.workers {
		wp.workers[i].Stop()
	}
}

// Submit 提交任務
func (wp *WorkerPool) Submit(job Job) error {
	select {
	case wp.jobQueue <- job:
		atomic.AddInt64(&wp.stats.TotalJobs, 1)
		atomic.AddInt64(&wp.stats.QueueLength, 1)
		return nil
	default:
		return fmt.Errorf("job queue full")
	}
}

// dispatcher 任務分發器
func (wp *WorkerPool) dispatcher() {
	for {
		select {
		case job := <-wp.jobQueue:
			atomic.AddInt64(&wp.stats.QueueLength, -1)
			atomic.AddInt64(&wp.stats.ActiveJobs, 1)

			// 獲取可用工作者
			jobChannel := <-wp.workerPool

			// 分發任務
			select {
			case jobChannel <- job:
			case <-wp.quit:
				return
			}

		case <-wp.quit:
			return
		}
	}
}

// GetStats 獲取工作池統計
func (wp *WorkerPool) GetStats() WorkerPoolStats {
	wp.mu.RLock()
	defer wp.mu.RUnlock()
	return wp.stats
}

// Start 啟動工作者
func (w *Worker) Start() {
	go func() {
		for {
			w.workerPool <- w.jobChannel

			select {
			case job := <-w.jobChannel:
				w.processJob(job)
			case <-w.quit:
				return
			}
		}
	}()
}

// Stop 停止工作者
func (w *Worker) Stop() {
	close(w.quit)
}

// processJob 處理工作任務
func (w *Worker) processJob(job Job) {
	start := time.Now()

	result := JobResult{
		Success:        true,
		WorkerID:       w.id,
		ProcessingTime: 0,
	}

	// 執行任務邏輯（簡化實現）
	switch job.Type {
	case JobTypeTransform:
		// 執行消息轉換
		result.Result = job.Message
	case JobTypeValidate:
		// 執行驗證
		result.Result = true
	case JobTypeRoute:
		// 執行路由
		result.Result = job.Wrapper
	case JobTypeBatch:
		// 執行批處理
		result.Result = "batch_processed"
	default:
		result.Success = false
		result.Error = fmt.Errorf("unknown job type: %s", job.Type)
	}

	result.ProcessingTime = time.Since(start)
	w.stats.JobsProcessed++
	w.stats.LastJobTime = time.Now()

	if result.Success {
		w.stats.JobsSucceeded++
	} else {
		w.stats.JobsFailed++
	}

	// 回調結果
	if job.Callback != nil {
		job.Callback(result)
	}
}

// NewMessageBuffer 創建訊息緩衝
func NewMessageBuffer(maxSize, batchSize int, flushInterval time.Duration) *MessageBuffer {
	return &MessageBuffer{
		buffer:        make([]*types.RTKMessage, 0, maxSize),
		maxSize:       maxSize,
		batchSize:     batchSize,
		flushInterval: flushInterval,
	}
}

// Start 啟動訊息緩衝
func (mb *MessageBuffer) Start() error {
	mb.flushTimer = time.NewTimer(mb.flushInterval)
	go mb.flushLoop()
	return nil
}

// Stop 停止訊息緩衝
func (mb *MessageBuffer) Stop() {
	if mb.flushTimer != nil {
		mb.flushTimer.Stop()
	}
}

// Add 添加訊息到緩衝
func (mb *MessageBuffer) Add(message *types.RTKMessage) error {
	mb.mu.Lock()
	defer mb.mu.Unlock()

	if len(mb.buffer) >= mb.maxSize {
		mb.stats.OverflowCount++
		return fmt.Errorf("buffer overflow")
	}

	mb.buffer = append(mb.buffer, message)
	mb.stats.TotalMessages++
	mb.stats.BufferedMessages++

	// 檢查是否需要flush
	if len(mb.buffer) >= mb.batchSize {
		go mb.flush()
	}

	return nil
}

// flushLoop 刷新循環
func (mb *MessageBuffer) flushLoop() {
	for {
		select {
		case <-mb.flushTimer.C:
			mb.flush()
			mb.flushTimer.Reset(mb.flushInterval)
		}
	}
}

// flush 刷新緩衝
func (mb *MessageBuffer) flush() {
	mb.mu.Lock()
	defer mb.mu.Unlock()

	if len(mb.buffer) == 0 {
		return
	}

	batch := make([]*types.RTKMessage, len(mb.buffer))
	copy(batch, mb.buffer)
	mb.buffer = mb.buffer[:0]

	mb.stats.BatchesProcessed++
	mb.stats.FlushCount++
	mb.stats.BufferedMessages = int64(len(mb.buffer))
	mb.stats.AverageBatchSize = float64(mb.stats.TotalMessages) / float64(mb.stats.BatchesProcessed)

	if mb.callback != nil {
		go mb.callback(batch)
	}
}

// GetStats 獲取緩衝統計
func (mb *MessageBuffer) GetStats() BufferStats {
	mb.mu.RLock()
	defer mb.mu.RUnlock()
	return mb.stats
}

// NewCircuitBreaker 創建熔斷器
func NewCircuitBreaker(threshold int, timeout, resetTimeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		state:        CircuitBreakerClosed,
		threshold:    threshold,
		timeout:      timeout,
		resetTimeout: resetTimeout,
	}
}

// Allow 檢查是否允許請求
func (cb *CircuitBreaker) Allow() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.stats.TotalRequests++

	switch cb.state {
	case CircuitBreakerClosed:
		return true
	case CircuitBreakerOpen:
		if time.Since(cb.lastFailTime) > cb.resetTimeout {
			cb.state = CircuitBreakerHalfOpen
			cb.stats.StateChanges++
			return true
		}
		cb.stats.RejectedRequests++
		return false
	case CircuitBreakerHalfOpen:
		return true
	}

	return false
}

// RecordSuccess 記錄成功
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.successCount++
	cb.stats.SuccessCount++

	if cb.state == CircuitBreakerHalfOpen {
		cb.state = CircuitBreakerClosed
		cb.failureCount = 0
		cb.stats.StateChanges++
	}
}

// RecordFailure 記錄失敗
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failureCount++
	cb.stats.FailureCount++
	cb.lastFailTime = time.Now()
	cb.stats.LastFailTime = cb.lastFailTime

	if cb.failureCount >= int64(cb.threshold) && cb.state == CircuitBreakerClosed {
		cb.state = CircuitBreakerOpen
		cb.stats.StateChanges++
	}
}

// GetStats 獲取熔斷器統計
func (cb *CircuitBreaker) GetStats() CircuitBreakerStats {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.stats
}

// NewRateLimiter 創建限流器
func NewRateLimiter(rate, burst int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		rate:       rate,
		burst:      burst,
		window:     window,
		tokens:     burst,
		lastRefill: time.Now(),
		requests:   make([]time.Time, 0),
	}
}

// Allow 檢查是否允許請求
func (rl *RateLimiter) Allow() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	rl.refillTokens()
	rl.stats.TotalRequests++

	if rl.tokens > 0 {
		rl.tokens--
		rl.requests = append(rl.requests, time.Now())
		rl.stats.AllowedRequests++
		rl.stats.CurrentTokens = rl.tokens
		return true
	}

	rl.stats.RejectedRequests++
	return false
}

// refillTokens 補充令牌
func (rl *RateLimiter) refillTokens() {
	now := time.Now()
	elapsed := now.Sub(rl.lastRefill)

	if elapsed > time.Second {
		tokensToAdd := int(elapsed.Seconds()) * rl.rate
		rl.tokens += tokensToAdd
		if rl.tokens > rl.burst {
			rl.tokens = rl.burst
		}
		rl.lastRefill = now
		rl.stats.RefillCount++
	}

	// 清理舊請求
	cutoff := now.Add(-rl.window)
	var validRequests []time.Time
	for _, req := range rl.requests {
		if req.After(cutoff) {
			validRequests = append(validRequests, req)
		}
	}
	rl.requests = validRequests

	// 計算平均速率
	if len(rl.requests) > 0 && rl.window > 0 {
		rl.stats.AverageRate = float64(len(rl.requests)) / rl.window.Seconds()
	}
}

// GetStats 獲取限流器統計
func (rl *RateLimiter) GetStats() RateLimiterStats {
	rl.mu.RLock()
	defer rl.mu.RUnlock()
	return rl.stats
}

// NewMemoryManager 創建記憶體管理器
func NewMemoryManager(threshold float64, maxUsage int64, gcInterval time.Duration) *MemoryManager {
	return &MemoryManager{
		threshold:   threshold,
		maxUsage:    maxUsage * 1024 * 1024, // 轉換為字節
		gcInterval:  gcInterval,
		memoryPools: make(map[string]*sync.Pool),
	}
}

// Start 啟動記憶體管理器
func (mm *MemoryManager) Start() {
	mm.gcTicker = time.NewTicker(mm.gcInterval)
	go mm.gcLoop()
}

// Stop 停止記憶體管理器
func (mm *MemoryManager) Stop() {
	if mm.gcTicker != nil {
		mm.gcTicker.Stop()
	}
}

// gcLoop 垃圾收集循環
func (mm *MemoryManager) gcLoop() {
	for {
		select {
		case <-mm.gcTicker.C:
			mm.performGC()
		}
	}
}

// performGC 執行垃圾收集
func (mm *MemoryManager) performGC() {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	mm.stats.CurrentUsage = int64(memStats.Alloc)

	if float64(mm.stats.CurrentUsage)/float64(mm.maxUsage) > mm.threshold {
		runtime.GC()
		mm.stats.GCCount++
		mm.stats.LastGCTime = time.Now()
	}
}

// Cleanup 清理記憶體池
func (mm *MemoryManager) Cleanup() {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	// 清理記憶體池（簡化實現）
	for name := range mm.memoryPools {
		mm.memoryPools[name] = &sync.Pool{}
	}
}

// GetStats 獲取記憶體統計
func (mm *MemoryManager) GetStats() MemoryStats {
	mm.mu.RLock()
	defer mm.mu.RUnlock()
	return mm.stats
}

// NewConnectionPool 創建連接池
func NewConnectionPool(maxConnections int, timeout, idleTimeout time.Duration) *ConnectionPool {
	cp := &ConnectionPool{
		connections:    make(map[string]*Connection),
		maxConnections: maxConnections,
		timeout:        timeout,
		idleTimeout:    idleTimeout,
	}

	cp.cleanupTicker = time.NewTicker(idleTimeout)
	return cp
}

// Start 啟動連接池
func (cp *ConnectionPool) Start() {
	go cp.cleanupLoop()
}

// Stop 停止連接池
func (cp *ConnectionPool) Stop() {
	if cp.cleanupTicker != nil {
		cp.cleanupTicker.Stop()
	}
}

// cleanupLoop 清理循環
func (cp *ConnectionPool) cleanupLoop() {
	for {
		select {
		case <-cp.cleanupTicker.C:
			cp.Cleanup()
		}
	}
}

// Cleanup 清理閒置連接
func (cp *ConnectionPool) Cleanup() {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	now := time.Now()
	for id, conn := range cp.connections {
		if !conn.InUse && now.Sub(conn.LastUsed) > cp.idleTimeout {
			delete(cp.connections, id)
			cp.stats.ConnectionsDestroyed++
		}
	}

	cp.updateStats()
}

// updateStats 更新統計
func (cp *ConnectionPool) updateStats() {
	cp.stats.TotalConnections = len(cp.connections)
	cp.stats.ActiveConnections = 0
	cp.stats.IdleConnections = 0

	totalUseCount := int64(0)
	for _, conn := range cp.connections {
		if conn.InUse {
			cp.stats.ActiveConnections++
		} else {
			cp.stats.IdleConnections++
		}
		totalUseCount += conn.UseCount
	}

	if len(cp.connections) > 0 {
		cp.stats.AverageUseCount = float64(totalUseCount) / float64(len(cp.connections))
	}
}

// GetStats 獲取連接池統計
func (cp *ConnectionPool) GetStats() ConnectionPoolStats {
	cp.mu.RLock()
	defer cp.mu.RUnlock()
	return cp.stats
}

// GetDefaultPerformanceConfig 獲取預設性能配置
func GetDefaultPerformanceConfig() PerformanceConfig {
	return PerformanceConfig{
		WorkerPoolSize:             10,
		WorkerQueueSize:            1000,
		WorkerTimeout:              30 * time.Second,
		MessageBufferSize:          10000,
		BatchSize:                  100,
		FlushInterval:              time.Second,
		CircuitBreakerThreshold:    10,
		CircuitBreakerTimeout:      30 * time.Second,
		CircuitBreakerResetTimeout: 60 * time.Second,
		RateLimit:                  1000,
		BurstLimit:                 100,
		RateLimitWindow:            time.Second,
		GCInterval:                 5 * time.Minute,
		MemoryThreshold:            0.8,
		MaxMemoryUsage:             1024,
		MaxConnections:             100,
		ConnectionTimeout:          30 * time.Second,
		IdleTimeout:                5 * time.Minute,
	}
}
