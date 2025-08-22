package workflow

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"rtk_controller/internal/llm"
	"rtk_controller/pkg/types"

	"github.com/google/uuid"
)

// WorkflowExecutor executes workflow steps and manages execution context
type WorkflowExecutor struct {
	toolEngine *llm.ToolEngine
	config     *EngineConfig
	registry   *WorkflowRegistry
	mutex      sync.RWMutex
}

// NewWorkflowExecutor creates a new workflow executor
func NewWorkflowExecutor(toolEngine *llm.ToolEngine, config *EngineConfig) *WorkflowExecutor {
	return &WorkflowExecutor{
		toolEngine: toolEngine,
		config:     config,
	}
}

// SetRegistry sets the workflow registry (for circular dependency resolution)
func (we *WorkflowExecutor) SetRegistry(registry *WorkflowRegistry) {
	we.registry = registry
}

// Execute executes a complete workflow
func (we *WorkflowExecutor) Execute(ctx context.Context, workflowID string, params map[string]interface{}) (*WorkflowResult, error) {
	startTime := time.Now()

	// Get workflow definition
	workflow, err := we.registry.GetWorkflow(workflowID)
	if err != nil {
		return &WorkflowResult{
			WorkflowID: workflowID,
			Success:    false,
			StartTime:  startTime,
			EndTime:    time.Now(),
			Duration:   time.Since(startTime),
			Error:      fmt.Sprintf("workflow not found: %v", err),
		}, err
	}

	// Create execution context
	execCtx := &ExecutionContext{
		Context:    ctx,
		WorkflowID: workflowID,
		SessionID:  uuid.New().String(),
		Parameters: params,
		Results:    make(map[string]interface{}),
		Metadata:   make(map[string]interface{}),
		StartTime:  startTime,
		ToolEngine: we.toolEngine,
	}

	// Initialize workflow result
	result := &WorkflowResult{
		WorkflowID: workflowID,
		SessionID:  execCtx.SessionID,
		StartTime:  startTime,
		Steps:      make([]StepResult, 0, len(workflow.Steps)),
		Success:    true,
		Metadata:   make(map[string]interface{}),
	}

	// Execute workflow steps
	for i, step := range workflow.Steps {
		stepResult, err := we.ExecuteStep(execCtx, step)
		result.Steps = append(result.Steps, stepResult)

		// Store step results in context for future steps
		execCtx.Results[step.ID] = stepResult

		if err != nil && !step.Optional {
			result.Success = false
			result.Error = fmt.Sprintf("step %d (%s) failed: %v", i, step.ID, err)
			break
		}
	}

	// Finalize result
	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)

	// Generate summary
	result.Summary = we.generateWorkflowSummary(result)

	// Add workflow metadata
	result.Metadata["workflow_name"] = workflow.Name
	result.Metadata["workflow_description"] = workflow.Description
	result.Metadata["total_steps"] = len(workflow.Steps)
	result.Metadata["successful_steps"] = we.countSuccessfulSteps(result.Steps)

	return result, nil
}

// ExecuteStep executes a single workflow step
func (we *WorkflowExecutor) ExecuteStep(ctx *ExecutionContext, step WorkflowStep) (StepResult, error) {
	stepStartTime := time.Now()

	stepResult := StepResult{
		StepID:    step.ID,
		StepName:  step.Name,
		ToolName:  step.ToolName,
		StartTime: stepStartTime,
		Success:   false,
	}

	// Check if step should be skipped based on condition
	if step.Condition != nil {
		shouldExecute, err := we.evaluateCondition(ctx, *step.Condition)
		if err != nil {
			stepResult.Error = fmt.Sprintf("condition evaluation failed: %v", err)
			stepResult.EndTime = time.Now()
			stepResult.Duration = time.Since(stepStartTime)
			return stepResult, err
		}

		if !shouldExecute {
			stepResult.Skipped = true
			stepResult.Success = true
			stepResult.EndTime = time.Now()
			stepResult.Duration = time.Since(stepStartTime)
			return stepResult, nil
		}
	}

	// Create step context with timeout
	stepCtx := ctx.Context
	if step.Timeout != nil {
		var cancel context.CancelFunc
		stepCtx, cancel = context.WithTimeout(ctx.Context, *step.Timeout)
		defer cancel()
	} else {
		var cancel context.CancelFunc
		stepCtx, cancel = context.WithTimeout(ctx.Context, we.config.DefaultTimeout)
		defer cancel()
	}

	// Execute step based on type
	var err error
	switch step.Type {
	case StepTypeTool:
		err = we.executeToolStep(stepCtx, ctx, step, &stepResult)
	case StepTypeParallel:
		err = we.executeParallelSteps(stepCtx, ctx, step.SubSteps, &stepResult)
	case StepTypeSequential:
		err = we.executeSequentialSteps(stepCtx, ctx, step.SubSteps, &stepResult)
	case StepTypeCondition:
		err = we.executeConditionalStep(stepCtx, ctx, step, &stepResult)
	default:
		err = fmt.Errorf("unknown step type: %s", step.Type)
	}

	// Handle retry if configured
	if err != nil && step.Retry != nil && step.Retry.MaxAttempts > 1 {
		err = we.executeWithRetry(stepCtx, ctx, step, &stepResult)
	}

	// Finalize step result
	stepResult.EndTime = time.Now()
	stepResult.Duration = time.Since(stepStartTime)
	stepResult.Success = (err == nil)

	if err != nil {
		stepResult.Error = err.Error()
	}

	return stepResult, err
}

// executeToolStep executes a tool call step
func (we *WorkflowExecutor) executeToolStep(stepCtx context.Context, execCtx *ExecutionContext, step WorkflowStep, result *StepResult) error {
	// Prepare parameters by merging step parameters with execution context parameters
	params := we.mergeParameters(step.Parameters, execCtx.Parameters, execCtx.Results)

	// Create LLM tool session
	session, err := execCtx.ToolEngine.CreateSession(stepCtx, &llm.SessionOptions{
		DeviceID: "workflow_executor",
		UserID:   "system",
		Metadata: map[string]interface{}{
			"workflow_id": execCtx.WorkflowID,
			"session_id":  execCtx.SessionID,
			"step_id":     step.ID,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create tool session: %w", err)
	}
	defer execCtx.ToolEngine.CloseSession(session.SessionID, types.LLMSessionStatusCompleted)

	// Execute tool
	toolResult, err := execCtx.ToolEngine.ExecuteTool(stepCtx, session.SessionID, step.ToolName, params)
	if err != nil {
		return fmt.Errorf("tool execution failed: %w", err)
	}

	// Store tool result
	result.ToolResult = toolResult

	return nil
}

// executeParallelSteps executes sub-steps in parallel
func (we *WorkflowExecutor) executeParallelSteps(stepCtx context.Context, execCtx *ExecutionContext, subSteps []WorkflowStep, result *StepResult) error {
	var wg sync.WaitGroup
	subResults := make([]StepResult, len(subSteps))
	errors := make([]error, len(subSteps))

	// Execute all sub-steps in parallel
	for i, subStep := range subSteps {
		wg.Add(1)
		go func(index int, step WorkflowStep) {
			defer wg.Done()

			subResult, err := we.ExecuteStep(execCtx, step)
			subResults[index] = subResult
			errors[index] = err
		}(i, subStep)
	}

	// Wait for all sub-steps to complete
	wg.Wait()

	// Check for errors
	var combinedError error
	for i, err := range errors {
		if err != nil && !subSteps[i].Optional {
			if combinedError == nil {
				combinedError = fmt.Errorf("parallel execution failed")
			}
			combinedError = fmt.Errorf("%v; step %s: %v", combinedError, subSteps[i].ID, err)
		}
	}

	// Store sub-results
	result.SubSteps = subResults

	return combinedError
}

// executeSequentialSteps executes sub-steps sequentially
func (we *WorkflowExecutor) executeSequentialSteps(stepCtx context.Context, execCtx *ExecutionContext, subSteps []WorkflowStep, result *StepResult) error {
	subResults := make([]StepResult, 0, len(subSteps))

	for i, subStep := range subSteps {
		subResult, err := we.ExecuteStep(execCtx, subStep)
		subResults = append(subResults, subResult)

		// Store sub-step result in execution context
		execCtx.Results[subStep.ID] = subResult

		if err != nil && !subStep.Optional {
			result.SubSteps = subResults
			return fmt.Errorf("sequential step %d (%s) failed: %w", i, subStep.ID, err)
		}
	}

	result.SubSteps = subResults
	return nil
}

// executeConditionalStep executes a conditional step
func (we *WorkflowExecutor) executeConditionalStep(stepCtx context.Context, execCtx *ExecutionContext, step WorkflowStep, result *StepResult) error {
	if step.Condition == nil {
		return fmt.Errorf("conditional step missing condition")
	}

	shouldExecute, err := we.evaluateCondition(execCtx, *step.Condition)
	if err != nil {
		return fmt.Errorf("condition evaluation failed: %w", err)
	}

	if !shouldExecute {
		result.Skipped = true
		return nil
	}

	// Execute sub-steps if condition is true
	if len(step.SubSteps) > 0 {
		return we.executeSequentialSteps(stepCtx, execCtx, step.SubSteps, result)
	}

	return nil
}

// executeWithRetry executes a step with retry logic
func (we *WorkflowExecutor) executeWithRetry(stepCtx context.Context, execCtx *ExecutionContext, step WorkflowStep, result *StepResult) error {
	var lastErr error

	for attempt := 1; attempt <= step.Retry.MaxAttempts; attempt++ {
		result.RetryCount = attempt - 1

		// Execute step
		switch step.Type {
		case StepTypeTool:
			lastErr = we.executeToolStep(stepCtx, execCtx, step, result)
		case StepTypeParallel:
			lastErr = we.executeParallelSteps(stepCtx, execCtx, step.SubSteps, result)
		case StepTypeSequential:
			lastErr = we.executeSequentialSteps(stepCtx, execCtx, step.SubSteps, result)
		}

		if lastErr == nil {
			return nil
		}

		// Check if we should retry based on retry conditions
		if !we.shouldRetry(lastErr, step.Retry.Conditions) {
			break
		}

		// Wait before retry (except for last attempt)
		if attempt < step.Retry.MaxAttempts {
			time.Sleep(time.Duration(step.Retry.BackoffMs) * time.Millisecond)
		}
	}

	return lastErr
}

// evaluateCondition evaluates a step condition
func (we *WorkflowExecutor) evaluateCondition(ctx *ExecutionContext, condition StepCondition) (bool, error) {
	// Get value from execution context results
	value := we.getValueFromPath(ctx.Results, condition.Field)

	switch condition.Operator {
	case "exists":
		return value != nil, nil
	case "not_exists":
		return value == nil, nil
	case "eq":
		return value == condition.Value, nil
	case "ne":
		return value != condition.Value, nil
	case "gt":
		return we.compareValues(value, condition.Value) > 0, nil
	case "lt":
		return we.compareValues(value, condition.Value) < 0, nil
	case "gte":
		return we.compareValues(value, condition.Value) >= 0, nil
	case "lte":
		return we.compareValues(value, condition.Value) <= 0, nil
	case "contains":
		return we.containsValue(value, condition.Value), nil
	default:
		return false, fmt.Errorf("unknown condition operator: %s", condition.Operator)
	}
}

// mergeParameters merges step parameters with execution context
func (we *WorkflowExecutor) mergeParameters(stepParams, execParams map[string]interface{}, results map[string]interface{}) map[string]interface{} {
	merged := make(map[string]interface{})

	// Start with execution parameters
	for k, v := range execParams {
		merged[k] = v
	}

	// Add step parameters (override execution parameters)
	for k, v := range stepParams {
		// Check if value is a reference to previous results
		if strVal, ok := v.(string); ok && strings.HasPrefix(strVal, "${") && strings.HasSuffix(strVal, "}") {
			// Extract reference path
			path := strings.TrimSuffix(strings.TrimPrefix(strVal, "${"), "}")
			if refValue := we.getValueFromPath(results, path); refValue != nil {
				merged[k] = refValue
			} else {
				merged[k] = v // Keep original if reference not found
			}
		} else {
			merged[k] = v
		}
	}

	return merged
}

// getValueFromPath gets a value from nested map using dot notation
func (we *WorkflowExecutor) getValueFromPath(data map[string]interface{}, path string) interface{} {
	parts := strings.Split(path, ".")
	current := data

	for _, part := range parts {
		if current == nil {
			return nil
		}

		if val, exists := current[part]; exists {
			switch v := val.(type) {
			case map[string]interface{}:
				current = v
			case StepResult:
				// Convert StepResult to map for further navigation
				current = map[string]interface{}{
					"success":     v.Success,
					"error":       v.Error,
					"tool_result": v.ToolResult,
					"sub_steps":   v.SubSteps,
				}
			default:
				if len(parts) == 1 {
					return val
				}
				return nil
			}
		} else {
			return nil
		}
	}

	return current
}

// compareValues compares two values (basic implementation)
func (we *WorkflowExecutor) compareValues(a, b interface{}) int {
	// This is a simplified comparison - would need more robust implementation
	switch va := a.(type) {
	case int:
		if vb, ok := b.(int); ok {
			if va > vb {
				return 1
			} else if va < vb {
				return -1
			}
			return 0
		}
	case float64:
		if vb, ok := b.(float64); ok {
			if va > vb {
				return 1
			} else if va < vb {
				return -1
			}
			return 0
		}
	case string:
		if vb, ok := b.(string); ok {
			return strings.Compare(va, vb)
		}
	}

	return 0
}

// containsValue checks if a contains b
func (we *WorkflowExecutor) containsValue(a, b interface{}) bool {
	switch va := a.(type) {
	case string:
		if vb, ok := b.(string); ok {
			return strings.Contains(va, vb)
		}
	case []interface{}:
		for _, item := range va {
			if item == b {
				return true
			}
		}
	}

	return false
}

// shouldRetry determines if a step should be retried based on error and retry conditions
func (we *WorkflowExecutor) shouldRetry(err error, conditions []string) bool {
	if len(conditions) == 0 {
		return true // Retry all errors by default
	}

	errStr := err.Error()
	for _, condition := range conditions {
		if strings.Contains(errStr, condition) {
			return true
		}
	}

	return false
}

// generateWorkflowSummary generates a summary of workflow execution
func (we *WorkflowExecutor) generateWorkflowSummary(result *WorkflowResult) string {
	if !result.Success {
		return fmt.Sprintf("Workflow failed: %s", result.Error)
	}

	successfulSteps := we.countSuccessfulSteps(result.Steps)
	totalSteps := len(result.Steps)

	return fmt.Sprintf("Workflow completed successfully. %d/%d steps executed successfully in %v",
		successfulSteps, totalSteps, result.Duration)
}

// countSuccessfulSteps counts the number of successful steps
func (we *WorkflowExecutor) countSuccessfulSteps(steps []StepResult) int {
	count := 0
	for _, step := range steps {
		if step.Success || step.Skipped {
			count++
		}
	}
	return count
}
