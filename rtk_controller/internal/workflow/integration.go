package workflow

import (
	"context"
	"fmt"

	"rtk_controller/internal/command"
	"rtk_controller/internal/llm"
	"rtk_controller/internal/qos"
	"rtk_controller/internal/storage"
	"rtk_controller/internal/topology"
)

// WorkflowManager manages the integration between workflow engine and existing systems
type WorkflowManager struct {
	engine *WorkflowEngine
}

// NewWorkflowManager creates a new workflow manager with full system integration
func NewWorkflowManager(
	storage storage.Storage,
	commandManager *command.Manager,
	topologyManager *topology.Manager,
	qosManager *qos.QoSManager,
) (*WorkflowManager, error) {

	// Create LLM tool engine (reuse existing pattern from diagnosis manager)
	toolEngine := llm.NewToolEngine(storage, commandManager, topologyManager, qosManager)

	// Create workflow engine with default config
	config := DefaultEngineConfig()
	workflowEngine, err := NewWorkflowEngine(toolEngine, storage, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create workflow engine: %w", err)
	}

	return &WorkflowManager{
		engine: workflowEngine,
	}, nil
}

// Start starts the workflow manager
func (wm *WorkflowManager) Start(ctx context.Context) error {
	return wm.engine.Start(ctx)
}

// Stop stops the workflow manager
func (wm *WorkflowManager) Stop() error {
	return wm.engine.Stop()
}

// GetEngine returns the workflow engine for direct access
func (wm *WorkflowManager) GetEngine() *WorkflowEngine {
	return wm.engine
}

// ProcessUserInput is a convenience method for processing natural language input
func (wm *WorkflowManager) ProcessUserInput(ctx context.Context, userInput string, context map[string]string) (*WorkflowResult, error) {
	return wm.engine.ProcessUserInput(ctx, userInput, context)
}

// ExecuteWorkflow is a convenience method for executing a specific workflow
func (wm *WorkflowManager) ExecuteWorkflow(ctx context.Context, workflowID string, params map[string]interface{}) (*WorkflowResult, error) {
	return wm.engine.ExecuteWorkflow(ctx, workflowID, params)
}

// ListWorkflows returns available workflows
func (wm *WorkflowManager) ListWorkflows() []string {
	return wm.engine.ListWorkflows()
}

// GetWorkflow returns workflow definition
func (wm *WorkflowManager) GetWorkflow(workflowID string) (*Workflow, error) {
	return wm.engine.GetWorkflow(workflowID)
}

// GetMetrics returns workflow execution metrics
func (wm *WorkflowManager) GetMetrics() *WorkflowMetrics {
	return wm.engine.GetMetrics()
}

// ReloadConfiguration reloads workflow and intent configurations
func (wm *WorkflowManager) ReloadConfiguration() error {
	return wm.engine.ReloadConfiguration()
}

// Helper methods for integration with existing CLI system

// FormatWorkflowResult formats workflow result for CLI display
func FormatWorkflowResult(result *WorkflowResult) string {
	if result == nil {
		return "No result available"
	}

	status := "✓ SUCCESS"
	if !result.Success {
		status = "✗ FAILED"
	}

	output := fmt.Sprintf("Workflow: %s [%s]\n", result.WorkflowID, status)
	output += fmt.Sprintf("Duration: %v\n", result.Duration)
	output += fmt.Sprintf("Steps: %d/%d successful\n", countSuccessfulSteps(result.Steps), len(result.Steps))

	if result.Summary != "" {
		output += fmt.Sprintf("Summary: %s\n", result.Summary)
	}

	if result.Error != "" {
		output += fmt.Sprintf("Error: %s\n", result.Error)
	}

	// Show step details
	if len(result.Steps) > 0 {
		output += "\nSteps:\n"
		for i, step := range result.Steps {
			stepStatus := "✓"
			if !step.Success {
				if step.Skipped {
					stepStatus = "⏭"
				} else {
					stepStatus = "✗"
				}
			}

			stepInfo := fmt.Sprintf("  %d. %s [%s] (%v)",
				i+1, step.StepID, stepStatus, step.Duration)

			if step.Error != "" {
				stepInfo += fmt.Sprintf(" - %s", step.Error)
			}

			output += stepInfo + "\n"
		}
	}

	return output
}

// FormatWorkflowList formats workflow list for CLI display
func FormatWorkflowList(workflows map[string]*Workflow) string {
	if len(workflows) == 0 {
		return "No workflows available"
	}

	output := "Available Workflows:\n\n"

	for id, workflow := range workflows {
		output += fmt.Sprintf("ID: %s\n", id)
		output += fmt.Sprintf("Name: %s\n", workflow.Name)
		output += fmt.Sprintf("Description: %s\n", workflow.Description)
		output += fmt.Sprintf("Intent: %s/%s\n", workflow.Intent.Primary, workflow.Intent.Secondary)
		output += fmt.Sprintf("Steps: %d\n", len(workflow.Steps))

		if len(workflow.Metadata.Tags) > 0 {
			output += fmt.Sprintf("Tags: %v\n", workflow.Metadata.Tags)
		}

		output += "\n"
	}

	return output
}

// countSuccessfulSteps counts successful steps in a workflow result
func countSuccessfulSteps(steps []StepResult) int {
	count := 0
	for _, step := range steps {
		if step.Success || step.Skipped {
			count++
		}
	}
	return count
}
