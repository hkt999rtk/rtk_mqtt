package workflow

import (
	"testing"
	"time"
)

func TestConfigValidator_ValidateWorkflows(t *testing.T) {
	validator := NewConfigValidator()

	tests := []struct {
		name            string
		workflows       map[string]Workflow
		expectValid     bool
		expectErrorsMin int
	}{
		{
			name: "Valid workflow",
			workflows: map[string]Workflow{
				"test_workflow": {
					ID:          "test_workflow",
					Name:        "Test Workflow",
					Description: "A test workflow",
					Intent: IntentMapping{
						Primary:   "network_diagnosis",
						Secondary: "connectivity_test",
					},
					Steps: []WorkflowStep{
						{
							ID:       "step1",
							Name:     "Test Step",
							Type:     StepTypeTool,
							ToolName: "network_topology_scan",
							Parameters: map[string]interface{}{
								"param1": "value1",
							},
						},
					},
					Metadata: WorkflowMetadata{
						Version:   "1.0.0",
						Author:    "Test Author",
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					},
				},
			},
			expectValid:     true,
			expectErrorsMin: 0,
		},
		{
			name: "Invalid workflow - missing name",
			workflows: map[string]Workflow{
				"invalid_workflow": {
					ID:          "invalid_workflow",
					Description: "Invalid workflow",
					Intent: IntentMapping{
						Primary: "network_diagnosis",
					},
					Steps: []WorkflowStep{
						{
							ID:   "step1",
							Type: StepTypeTool,
						},
					},
				},
			},
			expectValid:     false,
			expectErrorsMin: 1,
		},
		{
			name: "Invalid workflow - unknown tool",
			workflows: map[string]Workflow{
				"unknown_tool_workflow": {
					ID:          "unknown_tool_workflow",
					Name:        "Unknown Tool Workflow",
					Description: "Workflow with unknown tool",
					Intent: IntentMapping{
						Primary: "network_diagnosis",
					},
					Steps: []WorkflowStep{
						{
							ID:       "step1",
							Name:     "Unknown Tool Step",
							Type:     StepTypeTool,
							ToolName: "unknown_tool_name",
						},
					},
				},
			},
			expectValid:     false,
			expectErrorsMin: 1,
		},
		{
			name:            "Empty workflows map",
			workflows:       map[string]Workflow{},
			expectValid:     false,
			expectErrorsMin: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.ValidateWorkflows(tt.workflows)

			if result.IsValid != tt.expectValid {
				t.Errorf("ValidateWorkflows() IsValid = %v, want %v", result.IsValid, tt.expectValid)
			}

			if len(result.Errors) < tt.expectErrorsMin {
				t.Errorf("ValidateWorkflows() errors = %d, want >= %d", len(result.Errors), tt.expectErrorsMin)
			}

			// Print validation results for debugging
			if !result.IsValid {
				t.Logf("Validation errors:")
				for i, err := range result.Errors {
					t.Logf("  %d. %s", i+1, err.Error())
				}
			}
			if len(result.Warnings) > 0 {
				t.Logf("Validation warnings:")
				for i, warning := range result.Warnings {
					t.Logf("  %d. %s", i+1, warning.Error())
				}
			}
		})
	}
}

func TestConfigValidator_ValidateWorkflowStep(t *testing.T) {
	validator := NewConfigValidator()
	result := &ValidationResult{
		IsValid:  true,
		Errors:   []ValidationError{},
		Warnings: []ValidationError{},
	}

	tests := []struct {
		name        string
		step        WorkflowStep
		expectValid bool
	}{
		{
			name: "Valid tool step",
			step: WorkflowStep{
				ID:       "valid_step",
				Name:     "Valid Step",
				Type:     StepTypeTool,
				ToolName: "network_topology_scan",
			},
			expectValid: true,
		},
		{
			name: "Invalid step - unknown tool",
			step: WorkflowStep{
				ID:       "invalid_step",
				Name:     "Invalid Step",
				Type:     StepTypeTool,
				ToolName: "unknown_tool",
			},
			expectValid: false,
		},
		{
			name: "Invalid step - missing name",
			step: WorkflowStep{
				ID:   "no_name_step",
				Type: StepTypeTool,
			},
			expectValid: false,
		},
		{
			name: "Valid parallel step",
			step: WorkflowStep{
				ID:   "parallel_step",
				Name: "Parallel Step",
				Type: StepTypeParallel,
				SubSteps: []WorkflowStep{
					{
						ID:       "sub_step1",
						Name:     "Sub Step 1",
						Type:     StepTypeTool,
						ToolName: "network_topology_scan",
					},
				},
			},
			expectValid: true,
		},
		{
			name: "Invalid parallel step - no sub-steps",
			step: WorkflowStep{
				ID:   "empty_parallel_step",
				Name: "Empty Parallel Step",
				Type: StepTypeParallel,
			},
			expectValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset result for each test
			initialValid := result.IsValid
			initialErrorCount := len(result.Errors)

			validator.validateWorkflowStep("test", tt.step, result)

			hasNewErrors := len(result.Errors) > initialErrorCount
			isCurrentlyValid := result.IsValid && !hasNewErrors

			if isCurrentlyValid != tt.expectValid {
				t.Errorf("validateWorkflowStep() valid = %v, want %v", isCurrentlyValid, tt.expectValid)
				if len(result.Errors) > initialErrorCount {
					t.Logf("New validation errors:")
					for i := initialErrorCount; i < len(result.Errors); i++ {
						t.Logf("  %s", result.Errors[i].Error())
					}
				}
			}

			// Reset for next test
			result.IsValid = initialValid
			if !tt.expectValid {
				result.Errors = result.Errors[:initialErrorCount] // Remove errors added by this test
			}
		})
	}
}

func TestConfigValidator_ValidateIntentMapping(t *testing.T) {
	validator := NewConfigValidator()
	result := &ValidationResult{
		IsValid:  true,
		Errors:   []ValidationError{},
		Warnings: []ValidationError{},
	}

	tests := []struct {
		name        string
		intent      IntentMapping
		expectValid bool
	}{
		{
			name: "Valid intent mapping",
			intent: IntentMapping{
				Primary:   "network_diagnosis",
				Secondary: "connectivity_test",
			},
			expectValid: true,
		},
		{
			name: "Invalid intent mapping - empty primary",
			intent: IntentMapping{
				Primary:   "",
				Secondary: "connectivity_test",
			},
			expectValid: false,
		},
		{
			name: "Valid intent mapping - only primary",
			intent: IntentMapping{
				Primary: "network_diagnosis",
			},
			expectValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			initialValid := result.IsValid
			initialErrorCount := len(result.Errors)

			validator.validateIntentMapping("test", tt.intent, result)

			hasNewErrors := len(result.Errors) > initialErrorCount
			isCurrentlyValid := result.IsValid && !hasNewErrors

			if isCurrentlyValid != tt.expectValid {
				t.Errorf("validateIntentMapping() valid = %v, want %v", isCurrentlyValid, tt.expectValid)
				if hasNewErrors {
					t.Logf("New validation errors:")
					for i := initialErrorCount; i < len(result.Errors); i++ {
						t.Logf("  %s", result.Errors[i].Error())
					}
				}
			}

			// Reset for next test
			result.IsValid = initialValid
			if !tt.expectValid {
				result.Errors = result.Errors[:initialErrorCount]
			}
		})
	}
}

func TestConfigValidator_AddRemoveValidTool(t *testing.T) {
	validator := NewConfigValidator()

	// Test adding a new valid tool
	customTool := "custom_diagnostic_tool"
	validator.AddValidTool(customTool)

	// Create a workflow that uses the custom tool
	workflow := Workflow{
		ID:   "custom_workflow",
		Name: "Custom Workflow",
		Intent: IntentMapping{
			Primary: "custom_diagnosis",
		},
		Steps: []WorkflowStep{
			{
				ID:       "step1",
				Name:     "Custom Step",
				Type:     StepTypeTool,
				ToolName: customTool,
			},
		},
	}

	workflows := map[string]Workflow{"custom_workflow": workflow}
	result := validator.ValidateWorkflows(workflows)

	if !result.IsValid {
		t.Errorf("Workflow with added custom tool should be valid, but got errors: %v", result.Errors)
	}

	// Test removing the tool
	validator.RemoveValidTool(customTool)
	result = validator.ValidateWorkflows(workflows)

	if result.IsValid {
		t.Errorf("Workflow with removed custom tool should be invalid")
	}
}
