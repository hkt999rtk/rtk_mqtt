package workflow

import (
	"context"
	"fmt"
	"sync"
	"time"

	"rtk_controller/internal/llm"
	"rtk_controller/internal/storage"

	"github.com/google/uuid"
)

// WorkflowEngine implementation
type WorkflowEngine struct {
	registry   *WorkflowRegistry
	executor   *WorkflowExecutor
	classifier *IntentClassifier
	toolEngine *llm.ToolEngine
	storage    storage.Storage
	config     *EngineConfig

	// Runtime state
	started bool
	stopCh  chan struct{}
	mutex   sync.RWMutex

	// Metrics
	metrics *WorkflowMetrics
}

// NewWorkflowEngine creates a new workflow engine
func NewWorkflowEngine(
	toolEngine *llm.ToolEngine,
	storage storage.Storage,
	config *EngineConfig,
) (*WorkflowEngine, error) {
	if config == nil {
		config = DefaultEngineConfig()
	}

	engine := &WorkflowEngine{
		toolEngine: toolEngine,
		storage:    storage,
		config:     config,
		started:    false,
		stopCh:     make(chan struct{}),
		metrics:    NewWorkflowMetrics(),
	}

	// Initialize components
	registry := NewWorkflowRegistry(storage)
	executor := NewWorkflowExecutor(toolEngine, config)
	classifier := NewIntentClassifier(config)

	// Set circular dependencies
	executor.SetRegistry(registry)

	engine.registry = registry
	engine.executor = executor
	engine.classifier = classifier

	return engine, nil
}

// Start starts the workflow engine
func (we *WorkflowEngine) Start(ctx context.Context) error {
	we.mutex.Lock()
	defer we.mutex.Unlock()

	if we.started {
		return fmt.Errorf("workflow engine already started")
	}

	// Start components
	if err := we.toolEngine.Start(ctx); err != nil {
		return fmt.Errorf("failed to start tool engine: %w", err)
	}

	// Load default workflows
	if err := we.loadDefaultWorkflows(); err != nil {
		return fmt.Errorf("failed to load default workflows: %w", err)
	}

	// Load intent definitions from config
	intentConfigPath := "configs/intent_classification.yaml"
	if err := we.classifier.LoadIntentDefinitions(intentConfigPath); err != nil {
		// Note: Using built-in definitions as fallback
		fmt.Printf("Warning: Failed to load intent classification config, using built-in definitions: %v\n", err)
	}

	// Start background workers
	go we.metricsCollectionWorker(ctx)

	we.started = true
	return nil
}

// Stop stops the workflow engine
func (we *WorkflowEngine) Stop() error {
	we.mutex.Lock()
	defer we.mutex.Unlock()

	if !we.started {
		return nil
	}

	close(we.stopCh)

	if err := we.toolEngine.Stop(); err != nil {
		return fmt.Errorf("failed to stop tool engine: %w", err)
	}

	we.started = false
	return nil
}

// ProcessUserInput processes natural language input and executes appropriate workflow
func (we *WorkflowEngine) ProcessUserInput(ctx context.Context, userInput string, context map[string]string) (*WorkflowResult, error) {
	we.mutex.RLock()
	if !we.started {
		we.mutex.RUnlock()
		return nil, fmt.Errorf("workflow engine not started")
	}
	we.mutex.RUnlock()

	// Step 1: Classify user intent
	intentReq := &IntentRequest{
		UserInput: userInput,
		Context:   context,
		Metadata: map[string]string{
			"session_id": uuid.New().String(),
			"timestamp":  time.Now().Format(time.RFC3339),
		},
	}

	intentResp, err := we.classifier.ClassifyIntent(ctx, intentReq)
	if err != nil {
		return nil, fmt.Errorf("intent classification failed: %w", err)
	}

	// Check confidence threshold
	if intentResp.Confidence < we.config.ConfidenceThreshold {
		// Use fallback workflow
		if we.config.FallbackWorkflow != "" {
			intentResp.WorkflowID = we.config.FallbackWorkflow
		} else {
			return nil, fmt.Errorf("intent classification confidence too low: %.2f < %.2f",
				intentResp.Confidence, we.config.ConfidenceThreshold)
		}
	}

	// Step 2: Execute workflow
	result, err := we.ExecuteWorkflow(ctx, intentResp.WorkflowID, intentResp.Parameters)
	if err != nil {
		return nil, fmt.Errorf("workflow execution failed: %w", err)
	}

	// Add intent information to result metadata
	if result.Metadata == nil {
		result.Metadata = make(map[string]interface{})
	}
	result.Metadata["intent_classification"] = intentResp
	result.Metadata["user_input"] = userInput

	return result, nil
}

// ExecuteWorkflow executes a specific workflow by ID
func (we *WorkflowEngine) ExecuteWorkflow(ctx context.Context, workflowID string, params map[string]interface{}) (*WorkflowResult, error) {
	we.mutex.RLock()
	if !we.started {
		we.mutex.RUnlock()
		return nil, fmt.Errorf("workflow engine not started")
	}
	we.mutex.RUnlock()

	// Get workflow definition
	_, err := we.registry.GetWorkflow(workflowID)
	if err != nil {
		return nil, fmt.Errorf("workflow not found: %w", err)
	}

	// Create session ID for this execution
	sessionID := uuid.New().String()

	// Execute workflow
	result, err := we.executor.Execute(ctx, workflowID, params)
	if err != nil {
		// Record failed execution
		we.metrics.RecordExecution(workflowID, time.Since(result.StartTime), false)
		return result, err
	}

	// Record successful execution
	we.metrics.RecordExecution(workflowID, result.Duration, result.Success)

	// Add session information
	result.SessionID = sessionID

	// Store result if storage is available
	we.storeWorkflowResult(result)

	return result, nil
}

// GetWorkflow retrieves a workflow by ID
func (we *WorkflowEngine) GetWorkflow(workflowID string) (*Workflow, error) {
	return we.registry.GetWorkflow(workflowID)
}

// ListWorkflows returns all available workflow IDs
func (we *WorkflowEngine) ListWorkflows() []string {
	return we.registry.ListWorkflows()
}

// ReloadConfiguration reloads workflow definitions and intent classifications
func (we *WorkflowEngine) ReloadConfiguration() error {
	we.mutex.Lock()
	defer we.mutex.Unlock()

	// Reload workflows
	if err := we.loadDefaultWorkflows(); err != nil {
		return fmt.Errorf("failed to reload workflows: %w", err)
	}

	// Reload intent definitions
	if err := we.classifier.LoadIntentDefinitions(""); err != nil {
		return fmt.Errorf("failed to reload intent definitions: %w", err)
	}

	return nil
}

// GetMetrics returns current workflow metrics
func (we *WorkflowEngine) GetMetrics() *WorkflowMetrics {
	we.mutex.RLock()
	defer we.mutex.RUnlock()
	return we.metrics
}

// loadDefaultWorkflows loads workflow definitions from configuration
func (we *WorkflowEngine) loadDefaultWorkflows() error {
	// Try to load from workflows.yaml configuration file
	return we.registry.LoadWorkflows("")
}

// storeWorkflowResult stores workflow execution result
func (we *WorkflowEngine) storeWorkflowResult(result *WorkflowResult) {
	if we.storage == nil {
		return
	}

	key := fmt.Sprintf("workflow_result:%s:%s", result.WorkflowID, result.SessionID)

	// Since we don't have SetWithTTL in the storage interface, use regular Set
	// This could be enhanced later when storage interface supports TTL
	we.storage.Set(key, fmt.Sprintf("%+v", result))
}

// metricsCollectionWorker periodically collects and updates metrics
func (we *WorkflowEngine) metricsCollectionWorker(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-we.stopCh:
			return
		case <-ticker.C:
			we.updateMetrics()
		}
	}
}

// updateMetrics updates internal metrics
func (we *WorkflowEngine) updateMetrics() {
	// This will be implemented with actual metrics collection
	// For now, just update timestamp
	we.mutex.Lock()
	defer we.mutex.Unlock()

	// Update workflow stats from storage if available
	if we.storage != nil {
		// Implementation will be added later
	}
}

// NewWorkflowMetrics creates a new workflow metrics instance
func NewWorkflowMetrics() *WorkflowMetrics {
	return &WorkflowMetrics{
		WorkflowStats: make(map[string]*WorkflowStats),
	}
}

// RecordExecution records a workflow execution in metrics
func (wm *WorkflowMetrics) RecordExecution(workflowID string, duration time.Duration, success bool) {
	wm.TotalExecutions++

	if success {
		wm.SuccessfulExecutions++
	} else {
		wm.FailedExecutions++
	}

	// Update average execution time
	if wm.TotalExecutions == 1 {
		wm.AverageExecutionTime = duration
	} else {
		// Rolling average
		wm.AverageExecutionTime = time.Duration(
			(int64(wm.AverageExecutionTime)*(wm.TotalExecutions-1) + int64(duration)) / wm.TotalExecutions,
		)
	}

	// Update workflow-specific stats
	if _, exists := wm.WorkflowStats[workflowID]; !exists {
		wm.WorkflowStats[workflowID] = &WorkflowStats{
			WorkflowID: workflowID,
		}
	}

	stats := wm.WorkflowStats[workflowID]
	stats.ExecutionCount++
	stats.LastExecuted = time.Now()

	if success {
		stats.SuccessRate = float64(stats.ExecutionCount) / float64(stats.ExecutionCount)
	}

	// Update average execution time for this workflow
	if stats.ExecutionCount == 1 {
		stats.AverageExecutionTime = duration
	} else {
		stats.AverageExecutionTime = time.Duration(
			(int64(stats.AverageExecutionTime)*(stats.ExecutionCount-1) + int64(duration)) / stats.ExecutionCount,
		)
	}
}
