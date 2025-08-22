package workflow

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v2"
)

// ConfigValidator provides validation for workflow configuration files
type ConfigValidator struct {
	validToolNames    map[string]bool
	validStepTypes    map[string]bool
	validConditionOps map[string]bool
}

// ValidationError represents a configuration validation error
type ValidationError struct {
	Field   string
	Message string
	Line    int // Optional: line number in YAML file
}

func (ve ValidationError) Error() string {
	if ve.Line > 0 {
		return fmt.Sprintf("line %d, field %s: %s", ve.Line, ve.Field, ve.Message)
	}
	return fmt.Sprintf("field %s: %s", ve.Field, ve.Message)
}

// ValidationResult contains the result of configuration validation
type ValidationResult struct {
	IsValid  bool
	Errors   []ValidationError
	Warnings []ValidationError
}

// NewConfigValidator creates a new configuration validator
func NewConfigValidator() *ConfigValidator {
	return &ConfigValidator{
		validToolNames: map[string]bool{
			"network_topology_scan":     true,
			"wifi_signal_analysis":      true,
			"wan_connectivity_test":     true,
			"lan_performance_test":      true,
			"device_interference_scan":  true,
			"qos_analysis":              true,
			"network_health_check":      true,
			"wifi_channel_analysis":     true,
			"mesh_topology_analysis":    true,
			"wifi_advanced_diagnostics": true,
		},
		validStepTypes: map[string]bool{
			"tool_call":  true,
			"condition":  true,
			"parallel":   true,
			"sequential": true,
		},
		validConditionOps: map[string]bool{
			"equals":             true,
			"not_equals":         true,
			"greater_than":       true,
			"less_than":          true,
			"greater_than_equal": true,
			"less_than_equal":    true,
			"contains":           true,
			"not_contains":       true,
			"exists":             true,
			"not_exists":         true,
		},
	}
}

// ValidateWorkflowFile validates a workflow configuration file
func (cv *ConfigValidator) ValidateWorkflowFile(filePath string) (*ValidationResult, error) {
	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return &ValidationResult{
			IsValid: false,
			Errors: []ValidationError{
				{Field: "file", Message: fmt.Sprintf("workflow file does not exist: %s", filePath)},
			},
		}, nil
	}

	// Read file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read workflow file: %w", err)
	}

	// Parse YAML
	var config struct {
		Workflows map[string]Workflow `yaml:"workflows"`
	}

	if err := yaml.Unmarshal(data, &config); err != nil {
		return &ValidationResult{
			IsValid: false,
			Errors: []ValidationError{
				{Field: "yaml", Message: fmt.Sprintf("invalid YAML syntax: %s", err.Error())},
			},
		}, nil
	}

	// Validate workflows
	return cv.ValidateWorkflows(config.Workflows), nil
}

// ValidateWorkflows validates a map of workflow definitions
func (cv *ConfigValidator) ValidateWorkflows(workflows map[string]Workflow) *ValidationResult {
	result := &ValidationResult{
		IsValid:  true,
		Errors:   []ValidationError{},
		Warnings: []ValidationError{},
	}

	if len(workflows) == 0 {
		result.Errors = append(result.Errors, ValidationError{
			Field:   "workflows",
			Message: "no workflows defined in configuration",
		})
		result.IsValid = false
		return result
	}

	for workflowID, workflow := range workflows {
		cv.validateWorkflow(workflowID, workflow, result)
	}

	return result
}

// validateWorkflow validates a single workflow definition
func (cv *ConfigValidator) validateWorkflow(workflowID string, workflow Workflow, result *ValidationResult) {
	// Validate workflow ID
	if workflowID == "" {
		result.Errors = append(result.Errors, ValidationError{
			Field:   "workflow.id",
			Message: "workflow ID cannot be empty",
		})
		result.IsValid = false
	}

	// Validate required fields
	if workflow.Name == "" {
		result.Errors = append(result.Errors, ValidationError{
			Field:   fmt.Sprintf("workflows.%s.name", workflowID),
			Message: "workflow name is required",
		})
		result.IsValid = false
	}

	if workflow.Description == "" {
		result.Warnings = append(result.Warnings, ValidationError{
			Field:   fmt.Sprintf("workflows.%s.description", workflowID),
			Message: "workflow description is recommended",
		})
	}

	if workflow.Intent.Primary == "" {
		result.Errors = append(result.Errors, ValidationError{
			Field:   fmt.Sprintf("workflows.%s.intent.primary", workflowID),
			Message: "primary intent must be specified",
		})
		result.IsValid = false
	}

	if len(workflow.Steps) == 0 {
		result.Errors = append(result.Errors, ValidationError{
			Field:   fmt.Sprintf("workflows.%s.steps", workflowID),
			Message: "workflow must have at least one step",
		})
		result.IsValid = false
	}

	// Validate intent mapping
	cv.validateIntentMapping(fmt.Sprintf("workflows.%s.intent", workflowID), workflow.Intent, result)

	// Validate steps
	for i, step := range workflow.Steps {
		cv.validateWorkflowStep(fmt.Sprintf("workflows.%s.steps[%d]", workflowID, i), step, result)
	}

	// Validate step dependencies
	cv.validateStepDependencies(workflowID, workflow.Steps, result)
}

// validateIntentMapping validates an intent mapping
func (cv *ConfigValidator) validateIntentMapping(fieldPath string, intent IntentMapping, result *ValidationResult) {
	if intent.Primary == "" {
		result.Errors = append(result.Errors, ValidationError{
			Field:   fmt.Sprintf("%s.primary", fieldPath),
			Message: "primary intent cannot be empty",
		})
		result.IsValid = false
		return
	}

	// Check for reasonable intent length
	if len(intent.Primary) > 200 {
		result.Warnings = append(result.Warnings, ValidationError{
			Field:   fmt.Sprintf("%s.primary", fieldPath),
			Message: "primary intent description is very long, consider shortening",
		})
	}

	if len(intent.Secondary) > 200 {
		result.Warnings = append(result.Warnings, ValidationError{
			Field:   fmt.Sprintf("%s.secondary", fieldPath),
			Message: "secondary intent description is very long, consider shortening",
		})
	}
}

// validateWorkflowStep validates a workflow step
func (cv *ConfigValidator) validateWorkflowStep(fieldPath string, step WorkflowStep, result *ValidationResult) {
	// Validate step name
	if step.Name == "" {
		result.Errors = append(result.Errors, ValidationError{
			Field:   fmt.Sprintf("%s.name", fieldPath),
			Message: "step name is required",
		})
		result.IsValid = false
	}

	// Validate step type
	if step.Type != "" && !cv.validStepTypes[string(step.Type)] {
		result.Errors = append(result.Errors, ValidationError{
			Field:   fmt.Sprintf("%s.type", fieldPath),
			Message: fmt.Sprintf("invalid step type: %s. Valid types: %s", step.Type, cv.getValidStepTypes()),
		})
		result.IsValid = false
	}

	// Validate tool if present
	if step.ToolName != "" && !cv.validToolNames[step.ToolName] {
		result.Errors = append(result.Errors, ValidationError{
			Field:   fmt.Sprintf("%s.tool_name", fieldPath),
			Message: fmt.Sprintf("unknown tool: %s. Valid tools: %s", step.ToolName, cv.getValidToolNames()),
		})
		result.IsValid = false
	}

	// Validate nested steps for parallel/sequential types
	if (step.Type == "parallel" || step.Type == "sequential") && len(step.SubSteps) == 0 {
		result.Errors = append(result.Errors, ValidationError{
			Field:   fmt.Sprintf("%s.sub_steps", fieldPath),
			Message: fmt.Sprintf("%s step must have nested steps", step.Type),
		})
		result.IsValid = false
	}

	// Validate nested steps
	for i, nestedStep := range step.SubSteps {
		cv.validateWorkflowStep(fmt.Sprintf("%s.sub_steps[%d]", fieldPath, i), nestedStep, result)
	}

	// Validate conditional step
	if step.Type == "conditional" {
		cv.validateConditionalStep(fieldPath, step, result)
	}

	// Validate retry configuration
	if step.Retry != nil {
		cv.validateRetryConfig(fmt.Sprintf("%s.retry", fieldPath), *step.Retry, result)
	}

	// Validate parameters
	if step.Parameters != nil {
		cv.validateParameters(fmt.Sprintf("%s.parameters", fieldPath), step.Parameters, result)
	}
}

// validateConditionalStep validates conditional step configuration
func (cv *ConfigValidator) validateConditionalStep(fieldPath string, step WorkflowStep, result *ValidationResult) {
	if step.Condition == nil {
		result.Errors = append(result.Errors, ValidationError{
			Field:   fmt.Sprintf("%s.condition", fieldPath),
			Message: "conditional step must have condition defined",
		})
		result.IsValid = false
		return
	}

	condition := *step.Condition

	// Validate condition operator
	if !cv.validConditionOps[condition.Operator] {
		result.Errors = append(result.Errors, ValidationError{
			Field:   fmt.Sprintf("%s.condition.operator", fieldPath),
			Message: fmt.Sprintf("invalid condition operator: %s. Valid operators: %s", condition.Operator, cv.getValidConditionOps()),
		})
		result.IsValid = false
	}

	// Validate field path
	if condition.Field == "" {
		result.Errors = append(result.Errors, ValidationError{
			Field:   fmt.Sprintf("%s.condition.field", fieldPath),
			Message: "condition field path is required",
		})
		result.IsValid = false
	}

	// Validate that conditional step has then/else branches
	if len(step.SubSteps) == 0 && step.ToolName == "" {
		result.Warnings = append(result.Warnings, ValidationError{
			Field:   fmt.Sprintf("%s", fieldPath),
			Message: "conditional step has no action when condition is true",
		})
	}
}

// validateRetryConfig validates retry configuration
func (cv *ConfigValidator) validateRetryConfig(fieldPath string, retry RetryConfig, result *ValidationResult) {
	if retry.MaxAttempts <= 0 {
		result.Errors = append(result.Errors, ValidationError{
			Field:   fmt.Sprintf("%s.max_attempts", fieldPath),
			Message: "retry max_attempts must be greater than 0",
		})
		result.IsValid = false
	}

	if retry.MaxAttempts > 10 {
		result.Warnings = append(result.Warnings, ValidationError{
			Field:   fmt.Sprintf("%s.max_attempts", fieldPath),
			Message: "retry max_attempts is very high, consider reducing",
		})
	}

	if retry.BackoffMs < 0 {
		result.Errors = append(result.Errors, ValidationError{
			Field:   fmt.Sprintf("%s.backoff_ms", fieldPath),
			Message: "retry backoff_ms cannot be negative",
		})
		result.IsValid = false
	}
}

// validateParameters validates step parameters
func (cv *ConfigValidator) validateParameters(fieldPath string, params map[string]interface{}, result *ValidationResult) {
	// Basic parameter validation
	for key, value := range params {
		if key == "" {
			result.Errors = append(result.Errors, ValidationError{
				Field:   fieldPath,
				Message: "parameter key cannot be empty",
			})
			result.IsValid = false
		}

		// Check for reasonable parameter value types
		switch value.(type) {
		case string, int, float64, bool, []interface{}, map[interface{}]interface{}:
			// Valid types
		default:
			result.Warnings = append(result.Warnings, ValidationError{
				Field:   fmt.Sprintf("%s.%s", fieldPath, key),
				Message: "parameter value has unusual type",
			})
		}
	}
}

// validateStepDependencies checks for circular dependencies and invalid references
func (cv *ConfigValidator) validateStepDependencies(workflowID string, steps []WorkflowStep, result *ValidationResult) {
	stepNames := make(map[string]bool)

	// Collect all step names
	for _, step := range steps {
		cv.collectStepNames(step, stepNames)
	}

	// Check dependencies
	for i, step := range steps {
		cv.checkStepDependencies(fmt.Sprintf("workflows.%s.steps[%d]", workflowID, i), step, stepNames, result)
	}
}

// collectStepNames recursively collects all step names
func (cv *ConfigValidator) collectStepNames(step WorkflowStep, stepNames map[string]bool) {
	if step.Name != "" {
		stepNames[step.Name] = true
	}

	for _, nestedStep := range step.SubSteps {
		cv.collectStepNames(nestedStep, stepNames)
	}
}

// checkStepDependencies checks if step dependencies are valid
func (cv *ConfigValidator) checkStepDependencies(fieldPath string, step WorkflowStep, stepNames map[string]bool, result *ValidationResult) {
	// Note: Current WorkflowStep struct doesn't have DependsOn field
	// This is a placeholder for future dependency validation

	// Check nested steps
	for i, nestedStep := range step.SubSteps {
		cv.checkStepDependencies(fmt.Sprintf("%s.sub_steps[%d]", fieldPath, i), nestedStep, stepNames, result)
	}
}

// Helper methods to get valid values as comma-separated strings
func (cv *ConfigValidator) getValidToolNames() string {
	var tools []string
	for tool := range cv.validToolNames {
		tools = append(tools, tool)
	}
	return strings.Join(tools, ", ")
}

func (cv *ConfigValidator) getValidStepTypes() string {
	var types []string
	for stepType := range cv.validStepTypes {
		types = append(types, stepType)
	}
	return strings.Join(types, ", ")
}

func (cv *ConfigValidator) getValidConditionOps() string {
	var ops []string
	for op := range cv.validConditionOps {
		ops = append(ops, op)
	}
	return strings.Join(ops, ", ")
}

// ValidateIntentPattern validates intent pattern syntax
func (cv *ConfigValidator) ValidateIntentPattern(pattern string) error {
	// Check for valid regex pattern if it looks like one
	if strings.Contains(pattern, "^") || strings.Contains(pattern, "$") || strings.Contains(pattern, ".*") {
		_, err := regexp.Compile(pattern)
		if err != nil {
			return fmt.Errorf("invalid regex pattern: %w", err)
		}
	}
	return nil
}

// AddValidTool adds a new valid tool name to the validator
func (cv *ConfigValidator) AddValidTool(toolName string) {
	cv.validToolNames[toolName] = true
}

// RemoveValidTool removes a tool name from the validator
func (cv *ConfigValidator) RemoveValidTool(toolName string) {
	delete(cv.validToolNames, toolName)
}
