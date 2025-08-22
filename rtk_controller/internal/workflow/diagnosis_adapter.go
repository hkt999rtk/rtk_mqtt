package workflow

import (
	"context"
)

// DiagnosisWorkflowAdapter adapts WorkflowEngine to satisfy the diagnosis.WorkflowManager interface
type DiagnosisWorkflowAdapter struct {
	engine *WorkflowEngine
}

// NewDiagnosisWorkflowAdapter creates a new adapter
func NewDiagnosisWorkflowAdapter(engine *WorkflowEngine) *DiagnosisWorkflowAdapter {
	return &DiagnosisWorkflowAdapter{
		engine: engine,
	}
}

// ProcessUserInput processes user input and returns interface{}
func (adapter *DiagnosisWorkflowAdapter) ProcessUserInput(ctx context.Context, userInput string, context map[string]string) (interface{}, error) {
	return adapter.engine.ProcessUserInput(ctx, userInput, context)
}

// ExecuteWorkflow executes a workflow and returns interface{}
func (adapter *DiagnosisWorkflowAdapter) ExecuteWorkflow(ctx context.Context, workflowID string, params map[string]interface{}) (interface{}, error) {
	return adapter.engine.ExecuteWorkflow(ctx, workflowID, params)
}

// ListWorkflows returns available workflow IDs
func (adapter *DiagnosisWorkflowAdapter) ListWorkflows() []string {
	return adapter.engine.ListWorkflows()
}

// GetWorkflow returns workflow definition as interface{}
func (adapter *DiagnosisWorkflowAdapter) GetWorkflow(workflowID string) (interface{}, error) {
	return adapter.engine.GetWorkflow(workflowID)
}

// ReloadConfiguration reloads workflow configuration
func (adapter *DiagnosisWorkflowAdapter) ReloadConfiguration() error {
	return adapter.engine.ReloadConfiguration()
}
