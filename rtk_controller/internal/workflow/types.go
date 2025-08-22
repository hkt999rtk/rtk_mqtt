package workflow

import (
	"context"
	"time"

	"rtk_controller/internal/llm"
	"rtk_controller/pkg/types"
)

// WorkflowEngineInterface defines the main workflow engine operations
type WorkflowEngineInterface interface {
	Start(ctx context.Context) error
	Stop() error
	ProcessUserInput(ctx context.Context, userInput string, context map[string]string) (*WorkflowResult, error)
	ExecuteWorkflow(ctx context.Context, workflowID string, params map[string]interface{}) (*WorkflowResult, error)
	GetWorkflow(workflowID string) (*Workflow, error)
	ListWorkflows() []string
	ReloadConfiguration() error
}

// EngineConfig contains configuration for the workflow engine
type EngineConfig struct {
	DefaultTimeout         time.Duration `yaml:"default_timeout"`
	MaxConcurrentWorkflows int           `yaml:"max_concurrent_workflows"`
	RetryAttempts          int           `yaml:"retry_attempts"`
	ConfidenceThreshold    float64       `yaml:"confidence_threshold"`
	FallbackWorkflow       string        `yaml:"fallback_workflow"`
}

// DefaultEngineConfig returns sensible default configuration
func DefaultEngineConfig() *EngineConfig {
	return &EngineConfig{
		DefaultTimeout:         60 * time.Second,
		MaxConcurrentWorkflows: 5,
		RetryAttempts:          2,
		ConfidenceThreshold:    0.8,
		FallbackWorkflow:       "general_network_diagnosis",
	}
}

// Workflow represents a predefined diagnostic workflow
type Workflow struct {
	ID          string           `yaml:"id" json:"id"`
	Name        string           `yaml:"name" json:"name"`
	Description string           `yaml:"description" json:"description"`
	Intent      IntentMapping    `yaml:"intent" json:"intent"`
	Steps       []WorkflowStep   `yaml:"steps" json:"steps"`
	Metadata    WorkflowMetadata `yaml:"metadata" json:"metadata"`
}

// IntentMapping maps workflow to intent categories
type IntentMapping struct {
	Primary   string `yaml:"primary" json:"primary"`
	Secondary string `yaml:"secondary" json:"secondary"`
}

// WorkflowMetadata contains additional workflow information
type WorkflowMetadata struct {
	Version      string            `yaml:"version" json:"version"`
	Author       string            `yaml:"author" json:"author"`
	CreatedAt    time.Time         `yaml:"created_at" json:"created_at"`
	UpdatedAt    time.Time         `yaml:"updated_at" json:"updated_at"`
	Tags         []string          `yaml:"tags" json:"tags"`
	Requirements []string          `yaml:"requirements" json:"requirements"`
	ExtraData    map[string]string `yaml:"extra_data,omitempty" json:"extra_data,omitempty"`
}

// WorkflowStep represents a single step in a workflow
type WorkflowStep struct {
	ID         string                 `yaml:"id" json:"id"`
	Name       string                 `yaml:"name,omitempty" json:"name,omitempty"`
	Type       StepType               `yaml:"type" json:"type"`
	ToolName   string                 `yaml:"tool_name,omitempty" json:"tool_name,omitempty"`
	Parameters map[string]interface{} `yaml:"parameters,omitempty" json:"parameters,omitempty"`
	Condition  *StepCondition         `yaml:"condition,omitempty" json:"condition,omitempty"`
	SubSteps   []WorkflowStep         `yaml:"sub_steps,omitempty" json:"sub_steps,omitempty"`
	Timeout    *time.Duration         `yaml:"timeout,omitempty" json:"timeout,omitempty"`
	Retry      *RetryConfig           `yaml:"retry,omitempty" json:"retry,omitempty"`
	Optional   bool                   `yaml:"optional,omitempty" json:"optional,omitempty"`
}

// StepType defines the type of workflow step
type StepType string

const (
	StepTypeTool       StepType = "tool_call"
	StepTypeCondition  StepType = "condition"
	StepTypeParallel   StepType = "parallel"
	StepTypeSequential StepType = "sequential"
)

// StepCondition defines conditions for conditional execution
type StepCondition struct {
	Field    string      `yaml:"field" json:"field"`
	Operator string      `yaml:"operator" json:"operator"`
	Value    interface{} `yaml:"value" json:"value"`
}

// RetryConfig defines retry behavior for a step
type RetryConfig struct {
	MaxAttempts int      `yaml:"max_attempts" json:"max_attempts"`
	BackoffMs   int      `yaml:"backoff_ms" json:"backoff_ms"`
	Conditions  []string `yaml:"conditions,omitempty" json:"conditions,omitempty"`
}

// IntentRequest represents a request for intent classification
type IntentRequest struct {
	UserInput             string                 `json:"user_input"`
	Context               map[string]string      `json:"context,omitempty"`
	DeviceInfo            *DeviceContext         `json:"device_info,omitempty"`
	Metadata              map[string]string      `json:"metadata,omitempty"`
	ManualIntentOverride  *ManualIntentOverride  `json:"manual_intent_override,omitempty"`
	RequireConfirmation   bool                   `json:"require_confirmation,omitempty"`
}

// ManualIntentOverride allows manual specification of intent (Risk Mitigation 8.1)
type ManualIntentOverride struct {
	PrimaryIntent   string                 `json:"primary_intent"`
	SecondaryIntent string                 `json:"secondary_intent"`
	Parameters      map[string]interface{} `json:"parameters,omitempty"`
	Reason          string                 `json:"reason,omitempty"`
}

// IntentResponse represents the result of intent classification
type IntentResponse struct {
	PrimaryIntent   string                 `json:"primary_intent"`
	SecondaryIntent string                 `json:"secondary_intent"`
	Confidence      float64                `json:"confidence"`
	Parameters      map[string]interface{} `json:"parameters"`
	WorkflowID      string                 `json:"workflow_id"`
	Reasoning       string                 `json:"reasoning,omitempty"`
}

// DeviceContext contains device information for intent classification
type DeviceContext struct {
	DeviceID     string   `json:"device_id"`
	DeviceType   string   `json:"device_type"`
	Location     string   `json:"location,omitempty"`
	Capabilities []string `json:"capabilities,omitempty"`
}

// WorkflowResult represents the result of workflow execution
type WorkflowResult struct {
	WorkflowID string                 `json:"workflow_id"`
	SessionID  string                 `json:"session_id,omitempty"`
	Success    bool                   `json:"success"`
	StartTime  time.Time              `json:"start_time"`
	EndTime    time.Time              `json:"end_time"`
	Duration   time.Duration          `json:"duration"`
	Steps      []StepResult           `json:"steps"`
	Summary    string                 `json:"summary,omitempty"`
	Error      string                 `json:"error,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// StepResult represents the result of a single workflow step
type StepResult struct {
	StepID     string            `json:"step_id"`
	StepName   string            `json:"step_name,omitempty"`
	ToolName   string            `json:"tool_name,omitempty"`
	Success    bool              `json:"success"`
	StartTime  time.Time         `json:"start_time"`
	EndTime    time.Time         `json:"end_time"`
	Duration   time.Duration     `json:"duration"`
	ToolResult *types.ToolResult `json:"tool_result,omitempty"`
	SubSteps   []StepResult      `json:"sub_steps,omitempty"`
	Error      string            `json:"error,omitempty"`
	Skipped    bool              `json:"skipped,omitempty"`
	RetryCount int               `json:"retry_count,omitempty"`
}

// ExecutionContext contains context information during workflow execution
type ExecutionContext struct {
	Context    context.Context
	WorkflowID string
	SessionID  string
	Parameters map[string]interface{}
	Results    map[string]interface{} // Results from previous steps
	Metadata   map[string]interface{}
	StartTime  time.Time
	ToolEngine *llm.ToolEngine
}

// WorkflowMetrics contains metrics for workflow execution
type WorkflowMetrics struct {
	TotalExecutions      int64                     `json:"total_executions"`
	SuccessfulExecutions int64                     `json:"successful_executions"`
	FailedExecutions     int64                     `json:"failed_executions"`
	AverageExecutionTime time.Duration             `json:"average_execution_time"`
	IntentAccuracy       float64                   `json:"intent_accuracy"`
	WorkflowStats        map[string]*WorkflowStats `json:"workflow_stats"`
}

// WorkflowStats contains statistics for individual workflows
type WorkflowStats struct {
	WorkflowID           string        `json:"workflow_id"`
	ExecutionCount       int64         `json:"execution_count"`
	SuccessRate          float64       `json:"success_rate"`
	AverageExecutionTime time.Duration `json:"average_execution_time"`
	LastExecuted         time.Time     `json:"last_executed"`
}
