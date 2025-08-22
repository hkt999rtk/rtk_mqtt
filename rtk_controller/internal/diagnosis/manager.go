package diagnosis

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"rtk_controller/internal/config"
	"rtk_controller/internal/storage"
	"rtk_controller/pkg/types"
	"rtk_controller/pkg/utils"

	log "github.com/sirupsen/logrus"
)

// Manager handles diagnosis data collection and analysis
type Manager struct {
	config  config.DiagnosisConfig
	storage storage.Storage

	// Data collection
	dataQueue chan *types.DiagnosisData

	// Analysis sessions
	sessions map[string]*types.DiagnosisSession
	mu       sync.RWMutex

	// LLM sessions support (added for LLM tool integration)
	llmSessions map[string]*types.LLMSession
	llmMutex    sync.RWMutex
	toolEngine  ToolEngine // Interface to avoid circular dependency

	// Workflow management (for workflow-based diagnostics)
	workflowManager WorkflowManager // Interface to avoid circular dependency

	// Analyzers
	analyzers map[string]Analyzer

	// Background workers
	ctx    context.Context
	cancel context.CancelFunc
	done   chan struct{}

	// Statistics
	stats *types.DiagnosisStats
}

// ToolEngine interface to avoid circular dependency with llm package
type ToolEngine interface {
	CreateSession(ctx context.Context, options interface{}) (*types.LLMSession, error)
	ExecuteTool(ctx context.Context, sessionID, toolName string, params map[string]interface{}) (*types.ToolResult, error)
	GetSession(sessionID string) (*types.LLMSession, error)
	CloseSession(sessionID string, status types.LLMSessionStatus) error
	ListTools() []string
}

// WorkflowManager interface to avoid circular dependency with workflow package
type WorkflowManager interface {
	ProcessUserInput(ctx context.Context, userInput string, context map[string]string) (interface{}, error)
	ExecuteWorkflow(ctx context.Context, workflowID string, params map[string]interface{}) (interface{}, error)
	ListWorkflows() []string
	GetWorkflow(workflowID string) (interface{}, error)
	ReloadConfiguration() error
}

// Analyzer interface for diagnosis analysis
type Analyzer interface {
	Name() string
	Type() string
	SupportedTypes() []string
	Analyze(data []*types.DiagnosisData, config types.DiagnosisConfig) (*types.DiagnosisResult, error)
	GetInfo() *types.AnalyzerInfo
}

// NewManager creates a new diagnosis manager
func NewManager(config config.DiagnosisConfig, storage storage.Storage) *Manager {
	ctx, cancel := context.WithCancel(context.Background())

	manager := &Manager{
		config:      config,
		storage:     storage,
		dataQueue:   make(chan *types.DiagnosisData, 1000),
		sessions:    make(map[string]*types.DiagnosisSession),
		llmSessions: make(map[string]*types.LLMSession),
		analyzers:   make(map[string]Analyzer),
		ctx:         ctx,
		cancel:      cancel,
		done:        make(chan struct{}),
		stats:       &types.DiagnosisStats{},
	}

	// Register built-in analyzers
	manager.registerBuiltinAnalyzers()

	return manager
}

// Start starts the diagnosis manager
func (m *Manager) Start(ctx context.Context) error {
	log.Info("Starting diagnosis manager...")

	// Start background workers
	go m.dataProcessor()
	go m.sessionMonitor()
	go m.statsWorker()

	log.WithFields(log.Fields{
		"analyzers": len(m.analyzers),
		"enabled":   m.config.Enabled,
	}).Info("Diagnosis manager started")

	return nil
}

// Stop stops the diagnosis manager
func (m *Manager) Stop() {
	log.Info("Stopping diagnosis manager...")

	m.cancel()
	close(m.dataQueue)
	<-m.done

	log.Info("Diagnosis manager stopped")
}

// CollectDiagnosisData collects diagnosis data from various sources
func (m *Manager) CollectDiagnosisData(deviceID, dataType, category, source string, data map[string]interface{}, metrics map[string]float64) error {
	if !m.config.Enabled {
		return nil
	}

	diagData := &types.DiagnosisData{
		ID:        utils.GenerateMessageID(),
		DeviceID:  deviceID,
		Type:      dataType,
		Category:  category,
		Source:    source,
		Data:      data,
		Metrics:   metrics,
		Timestamp: time.Now().UnixMilli(),
		CreatedAt: time.Now(),
	}

	// Determine severity based on metrics and data
	diagData.Severity = m.determineSeverity(diagData)

	// Generate title and description
	diagData.Title = m.generateTitle(diagData)
	diagData.Description = m.generateDescription(diagData)

	// Queue for processing
	select {
	case m.dataQueue <- diagData:
		log.WithFields(log.Fields{
			"device_id": deviceID,
			"type":      dataType,
			"category":  category,
			"severity":  diagData.Severity,
		}).Debug("Diagnosis data collected")
		return nil
	default:
		log.Warn("Diagnosis data queue full, dropping data")
		return fmt.Errorf("diagnosis data queue full")
	}
}

// ProcessTelemetryData processes telemetry data for diagnosis
func (m *Manager) ProcessTelemetryData(deviceID, metricName string, data map[string]interface{}) error {
	// Extract metrics
	metrics := make(map[string]float64)
	for key, value := range data {
		if floatVal, ok := value.(float64); ok {
			metrics[key] = floatVal
		}
	}

	return m.CollectDiagnosisData(deviceID, "telemetry", "performance", "telemetry", data, metrics)
}

// ProcessEventData processes event data for diagnosis
func (m *Manager) ProcessEventData(deviceID, eventType string, eventData map[string]interface{}) error {
	category := "info"
	if severity, ok := eventData["severity"].(string); ok {
		switch severity {
		case "error", "critical":
			category = "error"
		case "warning":
			category = "warning"
		}
	}

	return m.CollectDiagnosisData(deviceID, eventType, category, "event", eventData, nil)
}

// StartDiagnosisSession starts a new diagnosis session for a device
func (m *Manager) StartDiagnosisSession(deviceID, triggerBy string, config types.DiagnosisConfig) (*types.DiagnosisSession, error) {
	session := &types.DiagnosisSession{
		ID:        utils.GenerateMessageID(),
		DeviceID:  deviceID,
		Type:      "manual",
		Status:    "running",
		TriggerBy: triggerBy,
		StartTime: time.Now(),
		Progress:  0.0,
		Config:    config,
	}

	m.mu.Lock()
	m.sessions[session.ID] = session
	m.mu.Unlock()

	// Start analysis in background
	go m.runDiagnosisSession(session)

	log.WithFields(log.Fields{
		"session_id": session.ID,
		"device_id":  deviceID,
		"trigger_by": triggerBy,
	}).Info("Diagnosis session started")

	return session, nil
}

// GetDiagnosisSession returns a diagnosis session by ID
func (m *Manager) GetDiagnosisSession(sessionID string) (*types.DiagnosisSession, error) {
	m.mu.RLock()
	session, exists := m.sessions[sessionID]
	m.mu.RUnlock()

	if !exists {
		// Try to load from storage
		return m.loadSessionFromStorage(sessionID)
	}

	// Return a copy
	sessionCopy := *session
	return &sessionCopy, nil
}

// ListDiagnosisData returns diagnosis data matching the filter
func (m *Manager) ListDiagnosisData(filter *types.DiagnosisFilter, limit int, offset int) ([]*types.DiagnosisData, int, error) {
	var dataList []*types.DiagnosisData
	var total int

	err := m.storage.View(func(tx storage.Transaction) error {
		return tx.IteratePrefix("diagnosis_data:", func(key, value string) error {
			var data types.DiagnosisData
			if err := json.Unmarshal([]byte(value), &data); err != nil {
				log.WithError(err).Warn("Failed to unmarshal diagnosis data")
				return nil
			}

			if m.dataMatchesFilter(&data, filter) {
				total++

				// Apply pagination
				if total > offset && (limit <= 0 || len(dataList) < limit) {
					dataList = append(dataList, &data)
				}
			}

			return nil
		})
	})

	return dataList, total, err
}

// GetStats returns diagnosis statistics
func (m *Manager) GetStats() *types.DiagnosisStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Return a copy of current stats
	stats := *m.stats
	return &stats
}

// RegisterAnalyzer registers a new analyzer
func (m *Manager) RegisterAnalyzer(analyzer Analyzer) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.analyzers[analyzer.Name()] = analyzer

	log.WithFields(log.Fields{
		"name": analyzer.Name(),
		"type": analyzer.Type(),
	}).Info("Analyzer registered")
}

// GetAnalyzers returns all registered analyzers
func (m *Manager) GetAnalyzers() []*types.AnalyzerInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var analyzers []*types.AnalyzerInfo
	for _, analyzer := range m.analyzers {
		analyzers = append(analyzers, analyzer.GetInfo())
	}

	return analyzers
}

// dataProcessor processes diagnosis data from the queue
func (m *Manager) dataProcessor() {
	defer close(m.done)

	for {
		select {
		case <-m.ctx.Done():
			return
		case data, ok := <-m.dataQueue:
			if !ok {
				return // Channel closed
			}

			m.processDiagnosisData(data)
		}
	}
}

// processDiagnosisData processes a single diagnosis data entry
func (m *Manager) processDiagnosisData(data *types.DiagnosisData) {
	// Store data
	if err := m.storeDiagnosisData(data); err != nil {
		log.WithError(err).Error("Failed to store diagnosis data")
		return
	}

	// Check if this data triggers automatic analysis
	if m.shouldTriggerAnalysis(data) {
		config := types.DiagnosisConfig{
			EnabledAnalyzers: m.config.DefaultAnalyzers,
			AnalysisDepth:    "standard",
			TimeRange: types.TimeRange{
				Duration: &[]int{1}[0], // Last 1 hour
			},
		}

		session, err := m.StartDiagnosisSession(data.DeviceID, "automatic", config)
		if err != nil {
			log.WithError(err).Error("Failed to start automatic diagnosis session")
		} else {
			log.WithFields(log.Fields{
				"session_id": session.ID,
				"device_id":  data.DeviceID,
				"trigger":    data.Type,
			}).Info("Automatic diagnosis triggered")
		}
	}
}

// runDiagnosisSession runs a complete diagnosis session
func (m *Manager) runDiagnosisSession(session *types.DiagnosisSession) {
	startTime := time.Now()

	defer func() {
		session.EndTime = &startTime
		session.Duration = time.Since(startTime).Milliseconds()

		// Save session to storage
		if err := m.storeSession(session); err != nil {
			log.WithError(err).Error("Failed to store diagnosis session")
		}

		// Remove from active sessions
		m.mu.Lock()
		delete(m.sessions, session.ID)
		m.mu.Unlock()
	}()

	// Collect diagnosis data for analysis
	data, err := m.collectDataForSession(session)
	if err != nil {
		session.Status = "failed"
		log.WithError(err).Error("Failed to collect data for diagnosis session")
		return
	}

	session.DataCount = len(data)
	session.Progress = 0.1

	// Run enabled analyzers
	var results []*types.DiagnosisResult
	for i, analyzerName := range session.Config.EnabledAnalyzers {
		analyzer, exists := m.analyzers[analyzerName]
		if !exists {
			log.WithField("analyzer", analyzerName).Warn("Analyzer not found")
			continue
		}

		log.WithFields(log.Fields{
			"session_id": session.ID,
			"analyzer":   analyzerName,
		}).Info("Running analyzer")

		result, err := analyzer.Analyze(data, session.Config)
		if err != nil {
			log.WithError(err).WithField("analyzer", analyzerName).Error("Analyzer failed")
			continue
		}

		results = append(results, result)
		session.Progress = 0.1 + (0.8 * float64(i+1) / float64(len(session.Config.EnabledAnalyzers)))

		// Store result
		if err := m.storeResult(result); err != nil {
			log.WithError(err).Error("Failed to store diagnosis result")
		}
	}

	session.ResultCount = len(results)
	session.Progress = 0.9

	// Generate summary
	session.Summary = m.generateSummary(results)
	session.IssueCount = session.Summary.TotalIssues
	session.Status = "completed"
	session.Progress = 1.0

	log.WithFields(log.Fields{
		"session_id":   session.ID,
		"device_id":    session.DeviceID,
		"duration_ms":  session.Duration,
		"data_count":   session.DataCount,
		"result_count": session.ResultCount,
		"issue_count":  session.IssueCount,
	}).Info("Diagnosis session completed")
}

// Helper methods (implementation details)
func (m *Manager) determineSeverity(data *types.DiagnosisData) string {
	// Simple severity determination based on category and metrics
	switch data.Category {
	case "error":
		return "high"
	case "warning":
		return "medium"
	default:
		return "low"
	}
}

func (m *Manager) generateTitle(data *types.DiagnosisData) string {
	return fmt.Sprintf("%s diagnosis data from %s", data.Type, data.DeviceID)
}

func (m *Manager) generateDescription(data *types.DiagnosisData) string {
	return fmt.Sprintf("Diagnosis data collected from %s via %s source", data.Type, data.Source)
}

func (m *Manager) shouldTriggerAnalysis(data *types.DiagnosisData) bool {
	// Trigger analysis for high severity issues
	return data.Severity == "high" || data.Category == "error"
}

func (m *Manager) dataMatchesFilter(data *types.DiagnosisData, filter *types.DiagnosisFilter) bool {
	if filter == nil {
		return true
	}

	if filter.DeviceID != "" && data.DeviceID != filter.DeviceID {
		return false
	}

	if len(filter.Type) > 0 {
		found := false
		for _, t := range filter.Type {
			if data.Type == t {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Add more filter checks as needed
	return true
}

func (m *Manager) collectDataForSession(session *types.DiagnosisSession) ([]*types.DiagnosisData, error) {
	var data []*types.DiagnosisData

	// Build time range for data collection
	endTime := time.Now().UnixMilli()
	startTime := endTime - int64(*session.Config.TimeRange.Duration)*3600*1000 // Convert hours to milliseconds

	filter := &types.DiagnosisFilter{
		DeviceID:  session.DeviceID,
		StartTime: &startTime,
		EndTime:   &endTime,
	}

	data, _, err := m.ListDiagnosisData(filter, 1000, 0)
	return data, err
}

func (m *Manager) generateSummary(results []*types.DiagnosisResult) types.DiagnosisSummary {
	summary := types.DiagnosisSummary{
		OverallHealth: "good",
		KeyMetrics:    make(map[string]float64),
	}

	// Aggregate issues from all results
	var allIssues []types.DiagnosisIssue
	for _, result := range results {
		allIssues = append(allIssues, result.Issues...)
	}

	// Count issues by severity
	for _, issue := range allIssues {
		summary.TotalIssues++
		switch issue.Severity {
		case "critical":
			summary.CriticalIssues++
		case "high":
			summary.HighIssues++
		case "medium":
			summary.MediumIssues++
		case "low":
			summary.LowIssues++
		}
	}

	// Determine overall health
	if summary.CriticalIssues > 0 {
		summary.OverallHealth = "critical"
	} else if summary.HighIssues > 2 {
		summary.OverallHealth = "poor"
	} else if summary.HighIssues > 0 || summary.MediumIssues > 3 {
		summary.OverallHealth = "fair"
	} else if summary.MediumIssues > 0 || summary.LowIssues > 5 {
		summary.OverallHealth = "good"
	} else {
		summary.OverallHealth = "excellent"
	}

	return summary
}

func (m *Manager) registerBuiltinAnalyzers() {
	// Register built-in analyzers
	m.RegisterAnalyzer(&WiFiAnalyzer{})
	m.RegisterAnalyzer(&NetworkAnalyzer{})
	m.RegisterAnalyzer(&SystemAnalyzer{})
}

// Storage methods
func (m *Manager) storeDiagnosisData(data *types.DiagnosisData) error {
	return m.storage.Transaction(func(tx storage.Transaction) error {
		key := fmt.Sprintf("diagnosis_data:%d:%s", data.Timestamp, data.ID)
		jsonData, err := json.Marshal(data)
		if err != nil {
			return err
		}
		return tx.Set(key, string(jsonData))
	})
}

func (m *Manager) storeResult(result *types.DiagnosisResult) error {
	return m.storage.Transaction(func(tx storage.Transaction) error {
		key := fmt.Sprintf("diagnosis_result:%s", result.ID)
		jsonData, err := json.Marshal(result)
		if err != nil {
			return err
		}
		return tx.Set(key, string(jsonData))
	})
}

func (m *Manager) storeSession(session *types.DiagnosisSession) error {
	return m.storage.Transaction(func(tx storage.Transaction) error {
		key := fmt.Sprintf("diagnosis_session:%s", session.ID)
		jsonData, err := json.Marshal(session)
		if err != nil {
			return err
		}
		return tx.Set(key, string(jsonData))
	})
}

func (m *Manager) loadSessionFromStorage(sessionID string) (*types.DiagnosisSession, error) {
	var session types.DiagnosisSession

	err := m.storage.View(func(tx storage.Transaction) error {
		key := fmt.Sprintf("diagnosis_session:%s", sessionID)
		value, err := tx.Get(key)
		if err != nil {
			return err
		}
		return json.Unmarshal([]byte(value), &session)
	})

	if err != nil {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}

	return &session, nil
}

// Background workers
func (m *Manager) sessionMonitor() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			m.checkSessionTimeouts()
		}
	}
}

func (m *Manager) checkSessionTimeouts() {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	for sessionID, session := range m.sessions {
		// Timeout sessions running for more than 10 minutes
		if session.Status == "running" && now.Sub(session.StartTime) > 10*time.Minute {
			session.Status = "failed"
			session.Progress = 1.0
			endTime := now
			session.EndTime = &endTime
			session.Duration = now.Sub(session.StartTime).Milliseconds()

			log.WithField("session_id", sessionID).Warn("Diagnosis session timed out")
		}
	}
}

func (m *Manager) statsWorker() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			m.updateStats()
		}
	}
}

func (m *Manager) updateStats() {
	m.mu.Lock()
	defer m.mu.Unlock()

	stats := &types.DiagnosisStats{
		TypeStats:     make(map[string]int),
		SeverityStats: make(map[string]int),
		DeviceStats:   make(map[string]int),
		AnalyzerStats: make(map[string]int),
		LastUpdated:   time.Now(),
	}

	// Count active sessions
	for _, session := range m.sessions {
		if session.Status == "running" {
			stats.ActiveSessions++
		}
	}

	// Count stored data and sessions
	err := m.storage.View(func(tx storage.Transaction) error {
		// Count diagnosis data
		tx.IteratePrefix("diagnosis_data:", func(key, value string) error {
			var data types.DiagnosisData
			if err := json.Unmarshal([]byte(value), &data); err == nil {
				stats.TotalDiagnoses++
				stats.TypeStats[data.Type]++
				stats.SeverityStats[data.Severity]++
				stats.DeviceStats[data.DeviceID]++
			}
			return nil
		})

		// Count sessions
		tx.IteratePrefix("diagnosis_session:", func(key, value string) error {
			var session types.DiagnosisSession
			if err := json.Unmarshal([]byte(value), &session); err == nil {
				if session.Status == "completed" {
					stats.CompletedSessions++
				} else if session.Status == "failed" {
					stats.FailedSessions++
				}
			}
			return nil
		})

		return nil
	})

	if err != nil {
		log.WithError(err).Error("Failed to update diagnosis statistics")
		return
	}

	m.stats = stats
}

// LLM Support Methods

// SetToolEngine sets the LLM tool engine for this diagnosis manager
func (m *Manager) SetToolEngine(engine ToolEngine) {
	m.llmMutex.Lock()
	defer m.llmMutex.Unlock()
	m.toolEngine = engine
}

// CreateLLMSession creates a new LLM diagnosis session
func (m *Manager) CreateLLMSession(ctx context.Context, deviceID, userID string, metadata map[string]interface{}) (*types.LLMSession, error) {
	if m.toolEngine == nil {
		return nil, fmt.Errorf("LLM tool engine not configured")
	}

	options := map[string]interface{}{
		"DeviceID": deviceID,
		"UserID":   userID,
		"Metadata": metadata,
	}

	session, err := m.toolEngine.CreateSession(ctx, options)
	if err != nil {
		return nil, fmt.Errorf("failed to create LLM session: %w", err)
	}

	// Store in local cache
	m.llmMutex.Lock()
	m.llmSessions[session.SessionID] = session
	m.llmMutex.Unlock()

	log.WithFields(log.Fields{
		"session_id": session.SessionID,
		"device_id":  deviceID,
		"user_id":    userID,
	}).Info("Created LLM diagnosis session")

	return session, nil
}

// ExecuteLLMTool executes an LLM tool within a session
func (m *Manager) ExecuteLLMTool(ctx context.Context, sessionID, toolName string, params map[string]interface{}) (*types.ToolResult, error) {
	if m.toolEngine == nil {
		return nil, fmt.Errorf("LLM tool engine not configured")
	}

	// Check if session exists locally
	m.llmMutex.RLock()
	session, exists := m.llmSessions[sessionID]
	m.llmMutex.RUnlock()

	if !exists {
		// Try to get from tool engine
		var err error
		session, err = m.toolEngine.GetSession(sessionID)
		if err != nil {
			return nil, fmt.Errorf("session %s not found: %w", sessionID, err)
		}

		// Add to local cache
		m.llmMutex.Lock()
		m.llmSessions[sessionID] = session
		m.llmMutex.Unlock()
	}

	log.WithFields(log.Fields{
		"session_id": sessionID,
		"tool_name":  toolName,
		"device_id":  session.DeviceID,
	}).Info("Executing LLM tool")

	result, err := m.toolEngine.ExecuteTool(ctx, sessionID, toolName, params)
	if err != nil {
		log.WithFields(log.Fields{
			"session_id": sessionID,
			"tool_name":  toolName,
			"error":      err.Error(),
		}).Error("LLM tool execution failed")
		return nil, err
	}

	log.WithFields(log.Fields{
		"session_id":     sessionID,
		"tool_name":      toolName,
		"success":        result.Success,
		"execution_time": result.ExecutionTime,
	}).Info("LLM tool execution completed")

	return result, nil
}

// GetLLMSession retrieves an LLM session by ID
func (m *Manager) GetLLMSession(sessionID string) (*types.LLMSession, error) {
	// Try local cache first
	m.llmMutex.RLock()
	if session, exists := m.llmSessions[sessionID]; exists {
		m.llmMutex.RUnlock()
		return session, nil
	}
	m.llmMutex.RUnlock()

	// Try tool engine
	if m.toolEngine != nil {
		session, err := m.toolEngine.GetSession(sessionID)
		if err == nil {
			// Add to local cache
			m.llmMutex.Lock()
			m.llmSessions[sessionID] = session
			m.llmMutex.Unlock()
			return session, nil
		}
	}

	return nil, fmt.Errorf("LLM session %s not found", sessionID)
}

// CloseLLMSession closes an LLM diagnosis session
func (m *Manager) CloseLLMSession(sessionID string, status types.LLMSessionStatus) error {
	if m.toolEngine == nil {
		return fmt.Errorf("LLM tool engine not configured")
	}

	err := m.toolEngine.CloseSession(sessionID, status)
	if err != nil {
		return fmt.Errorf("failed to close LLM session: %w", err)
	}

	// Remove from local cache
	m.llmMutex.Lock()
	delete(m.llmSessions, sessionID)
	m.llmMutex.Unlock()

	log.WithFields(log.Fields{
		"session_id": sessionID,
		"status":     string(status),
	}).Info("Closed LLM diagnosis session")

	return nil
}

// ListLLMTools returns all available LLM diagnostic tools
func (m *Manager) ListLLMTools() []string {
	if m.toolEngine == nil {
		return []string{}
	}
	return m.toolEngine.ListTools()
}

// GetLLMSessionHistory returns recent LLM sessions for a device
func (m *Manager) GetLLMSessionHistory(deviceID string, limit int) ([]*types.LLMSession, error) {
	var sessions []*types.LLMSession

	// Get from local cache
	m.llmMutex.RLock()
	for _, session := range m.llmSessions {
		if deviceID == "" || session.DeviceID == deviceID {
			sessions = append(sessions, session)
		}
	}
	m.llmMutex.RUnlock()

	// TODO: Also load from storage for complete history

	// Sort by creation time (most recent first)
	// Simple bubble sort for small datasets
	for i := 0; i < len(sessions)-1; i++ {
		for j := 0; j < len(sessions)-i-1; j++ {
			if sessions[j].CreatedAt.Before(sessions[j+1].CreatedAt) {
				sessions[j], sessions[j+1] = sessions[j+1], sessions[j]
			}
		}
	}

	// Apply limit
	if limit > 0 && len(sessions) > limit {
		sessions = sessions[:limit]
	}

	return sessions, nil
}

// EnableLLMSupport enables LLM support for this diagnosis manager
func (m *Manager) EnableLLMSupport(engine ToolEngine) error {
	m.SetToolEngine(engine)

	log.Info("LLM support enabled for diagnosis manager")
	return nil
}

// SetWorkflowManager sets the workflow manager for this diagnosis manager
func (m *Manager) SetWorkflowManager(manager WorkflowManager) {
	m.workflowManager = manager
}

// GetWorkflowManager returns the workflow manager
func (m *Manager) GetWorkflowManager() WorkflowManager {
	return m.workflowManager
}

// EnableWorkflowSupport enables workflow-based diagnostics for this diagnosis manager
func (m *Manager) EnableWorkflowSupport(manager WorkflowManager) error {
	m.SetWorkflowManager(manager)

	log.Info("Workflow-based diagnosis support enabled")
	return nil
}
