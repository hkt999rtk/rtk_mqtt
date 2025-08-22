package mcp

import (
	"context"
	"fmt"
	"strings"
	"time"

	"rtk_controller/internal/workflow"
)

// WorkflowMCPAdapter adapts workflow engine for MCP protocol
type WorkflowMCPAdapter struct {
	workflowEngine *workflow.WorkflowEngine
	toolExports    map[string]MCPToolDefinition
}

// MCPToolDefinition defines an MCP tool exported from a workflow
type MCPToolDefinition struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
	WorkflowID  string                 `json:"workflowId"`
	Category    string                 `json:"category,omitempty"`
	Tags        []string               `json:"tags,omitempty"`
}

// NewWorkflowMCPAdapter creates a new workflow MCP adapter
func NewWorkflowMCPAdapter(workflowEngine *workflow.WorkflowEngine) *WorkflowMCPAdapter {
	return &WorkflowMCPAdapter{
		workflowEngine: workflowEngine,
		toolExports:    make(map[string]MCPToolDefinition),
	}
}

// Initialize initializes the MCP adapter and generates tool definitions
func (adapter *WorkflowMCPAdapter) Initialize() error {
	// Generate MCP tool definitions from available workflows
	if err := adapter.generateMCPTools(); err != nil {
		return fmt.Errorf("failed to generate MCP tools: %w", err)
	}

	return nil
}

// ExportWorkflowAsTool exports a single workflow as an MCP tool
func (adapter *WorkflowMCPAdapter) ExportWorkflowAsTool(workflowID string) (*MCPToolDefinition, error) {
	workflow, err := adapter.workflowEngine.GetWorkflow(workflowID)
	if err != nil {
		return nil, fmt.Errorf("workflow not found: %w", err)
	}

	// Generate MCP tool name
	mcpName := fmt.Sprintf("rtk_%s", strings.ReplaceAll(workflowID, "_", "-"))

	// Generate input schema based on workflow requirements
	inputSchema := adapter.generateInputSchema(workflow)

	// Create category based on workflow intent
	category := adapter.mapIntentToCategory(workflow.Intent.Primary)

	toolDef := MCPToolDefinition{
		Name:        mcpName,
		Description: workflow.Description,
		InputSchema: inputSchema,
		WorkflowID:  workflowID,
		Category:    category,
		Tags:        workflow.Metadata.Tags,
	}

	adapter.toolExports[mcpName] = toolDef
	return &toolDef, nil
}

// HandleToolInvocation handles MCP tool invocation by executing the corresponding workflow
func (adapter *WorkflowMCPAdapter) HandleToolInvocation(ctx context.Context, toolName string, params map[string]interface{}) (*workflow.WorkflowResult, error) {
	// Get tool definition
	toolDef, exists := adapter.toolExports[toolName]
	if !exists {
		return nil, fmt.Errorf("tool %s not found", toolName)
	}

	// Validate parameters against schema
	if err := adapter.validateParameters(params, toolDef.InputSchema); err != nil {
		return nil, fmt.Errorf("parameter validation failed: %w", err)
	}

	// Execute workflow
	result, err := adapter.workflowEngine.ExecuteWorkflow(ctx, toolDef.WorkflowID, params)
	if err != nil {
		return nil, fmt.Errorf("workflow execution failed: %w", err)
	}

	return result, nil
}

// GetAllMCPTools returns all exported MCP tools
func (adapter *WorkflowMCPAdapter) GetAllMCPTools() []MCPToolDefinition {
	tools := make([]MCPToolDefinition, 0, len(adapter.toolExports))
	for _, toolDef := range adapter.toolExports {
		tools = append(tools, toolDef)
	}
	return tools
}

// ConsolidateResults consolidates workflow execution results for MCP response
func (adapter *WorkflowMCPAdapter) ConsolidateResults(workflowResult *workflow.WorkflowResult) map[string]interface{} {
	consolidated := map[string]interface{}{
		"workflow_id": workflowResult.WorkflowID,
		"success":     workflowResult.Success,
		"duration_ms": workflowResult.Duration.Milliseconds(),
		"timestamp":   workflowResult.StartTime.Format(time.RFC3339),
		"session_id":  workflowResult.SessionID,
	}

	// Group results by category for better organization
	categories := make(map[string]interface{})

	for _, stepResult := range workflowResult.Steps {
		if stepResult.ToolResult != nil {
			categoryName := adapter.extractCategoryFromToolName(stepResult.ToolResult.ToolName)
			if categories[categoryName] == nil {
				categories[categoryName] = make(map[string]interface{})
			}
			categoryMap := categories[categoryName].(map[string]interface{})
			categoryMap[stepResult.ToolResult.ToolName] = stepResult.ToolResult.Data
		}
	}

	consolidated["results"] = categories

	// Generate summary and recommendations
	consolidated["summary"] = adapter.generateWorkflowSummary(workflowResult)
	consolidated["recommendations"] = adapter.extractRecommendations(workflowResult)

	// Add step execution details
	consolidated["execution_details"] = adapter.generateExecutionDetails(workflowResult)

	return consolidated
}

// generateMCPTools generates MCP tool definitions from all available workflows
func (adapter *WorkflowMCPAdapter) generateMCPTools() error {
	workflows := adapter.workflowEngine.ListWorkflows()

	for _, workflowID := range workflows {
		if _, err := adapter.ExportWorkflowAsTool(workflowID); err != nil {
			return fmt.Errorf("failed to export workflow %s: %w", workflowID, err)
		}
	}

	return nil
}

// generateInputSchema generates JSON schema for workflow parameters
func (adapter *WorkflowMCPAdapter) generateInputSchema(workflow *workflow.Workflow) map[string]interface{} {
	schema := map[string]interface{}{
		"type":       "object",
		"properties": make(map[string]interface{}),
		"required":   []string{},
	}

	properties := schema["properties"].(map[string]interface{})
	required := []string{}

	// Extract parameters from workflow steps
	parameterHints := adapter.extractParameterHints(workflow)

	for paramName, paramInfo := range parameterHints {
		properties[paramName] = map[string]interface{}{
			"type":        paramInfo.Type,
			"description": paramInfo.Description,
		}

		if paramInfo.Required {
			required = append(required, paramName)
		}

		if len(paramInfo.Examples) > 0 {
			properties[paramName].(map[string]interface{})["examples"] = paramInfo.Examples
		}
	}

	schema["required"] = required
	return schema
}

// ParameterHint represents extracted parameter information
type ParameterHint struct {
	Type        string   `json:"type"`
	Description string   `json:"description"`
	Required    bool     `json:"required"`
	Examples    []string `json:"examples,omitempty"`
}

// extractParameterHints extracts parameter hints from workflow definition
func (adapter *WorkflowMCPAdapter) extractParameterHints(workflow *workflow.Workflow) map[string]ParameterHint {
	hints := make(map[string]ParameterHint)

	// Common parameters based on workflow intent
	switch workflow.Intent.Primary {
	case "coverage_issues":
		hints["location1"] = ParameterHint{
			Type:        "string",
			Description: "First location for comparison (e.g., living room)",
			Required:    true,
			Examples:    []string{"living room", "bedroom", "kitchen"},
		}
		hints["location2"] = ParameterHint{
			Type:        "string",
			Description: "Second location for comparison (optional)",
			Required:    false,
			Examples:    []string{"bedroom", "garage", "office"},
		}
		hints["severity"] = ParameterHint{
			Type:        "string",
			Description: "Severity of coverage issue",
			Required:    false,
			Examples:    []string{"critical", "moderate", "minor"},
		}

	case "connectivity_issues":
		hints["device_type"] = ParameterHint{
			Type:        "string",
			Description: "Type of device experiencing issues",
			Required:    false,
			Examples:    []string{"smartphone", "laptop", "smart_tv"},
		}
		hints["frequency"] = ParameterHint{
			Type:        "string",
			Description: "How often disconnections occur",
			Required:    false,
			Examples:    []string{"constantly", "frequently", "occasionally"},
		}

	case "performance_problems":
		hints["speed_type"] = ParameterHint{
			Type:        "string",
			Description: "Type of speed issue",
			Required:    false,
			Examples:    []string{"download", "upload", "both"},
		}
		hints["device_type"] = ParameterHint{
			Type:        "string",
			Description: "Device experiencing slow speeds",
			Required:    false,
			Examples:    []string{"smartphone", "laptop", "streaming_device"},
		}

	case "device_issues":
		hints["device_id"] = ParameterHint{
			Type:        "string",
			Description: "ID or name of the device",
			Required:    true,
			Examples:    []string{"phone-001", "laptop-002", "iot-device-003"},
		}
	}

	return hints
}

// validateParameters validates parameters against JSON schema
func (adapter *WorkflowMCPAdapter) validateParameters(params map[string]interface{}, schema map[string]interface{}) error {
	// Basic validation - check required parameters
	required, ok := schema["required"].([]string)
	if ok {
		for _, reqParam := range required {
			if _, exists := params[reqParam]; !exists {
				return fmt.Errorf("required parameter '%s' is missing", reqParam)
			}
		}
	}

	// Type validation for properties
	properties, ok := schema["properties"].(map[string]interface{})
	if ok {
		for paramName, paramValue := range params {
			if propSchema, exists := properties[paramName]; exists {
				propSchemaMap := propSchema.(map[string]interface{})
				expectedType := propSchemaMap["type"].(string)

				if !adapter.validateParameterType(paramValue, expectedType) {
					return fmt.Errorf("parameter '%s' has invalid type, expected %s", paramName, expectedType)
				}
			}
		}
	}

	return nil
}

// validateParameterType validates parameter type
func (adapter *WorkflowMCPAdapter) validateParameterType(value interface{}, expectedType string) bool {
	switch expectedType {
	case "string":
		_, ok := value.(string)
		return ok
	case "number":
		switch value.(type) {
		case int, int64, float64:
			return true
		default:
			return false
		}
	case "boolean":
		_, ok := value.(bool)
		return ok
	case "object":
		_, ok := value.(map[string]interface{})
		return ok
	case "array":
		_, ok := value.([]interface{})
		return ok
	default:
		return true // Allow unknown types
	}
}

// mapIntentToCategory maps workflow intent to MCP category
func (adapter *WorkflowMCPAdapter) mapIntentToCategory(intent string) string {
	switch intent {
	case "coverage_issues":
		return "wifi_diagnosis"
	case "connectivity_issues":
		return "network_diagnosis"
	case "performance_problems":
		return "performance_analysis"
	case "device_issues":
		return "device_management"
	default:
		return "general_diagnosis"
	}
}

// extractCategoryFromToolName extracts category from tool name
func (adapter *WorkflowMCPAdapter) extractCategoryFromToolName(toolName string) string {
	parts := strings.Split(toolName, ".")
	if len(parts) > 0 {
		return parts[0]
	}
	return "misc"
}

// generateWorkflowSummary generates a summary of workflow execution
func (adapter *WorkflowMCPAdapter) generateWorkflowSummary(result *workflow.WorkflowResult) string {
	if !result.Success {
		return fmt.Sprintf("Workflow %s failed: %s", result.WorkflowID, result.Error)
	}

	successfulSteps := 0
	for _, step := range result.Steps {
		if step.Success || step.Skipped {
			successfulSteps++
		}
	}

	return fmt.Sprintf("Workflow %s completed successfully. %d/%d steps executed in %v",
		result.WorkflowID, successfulSteps, len(result.Steps), result.Duration)
}

// extractRecommendations extracts recommendations from workflow results
func (adapter *WorkflowMCPAdapter) extractRecommendations(result *workflow.WorkflowResult) []string {
	var recommendations []string

	// Extract recommendations from tool results
	for _, step := range result.Steps {
		if step.ToolResult != nil && step.ToolResult.Data != nil {
			if data, ok := step.ToolResult.Data.(map[string]interface{}); ok {
				// Look for recommendations in various formats
				if recs, exists := data["recommendations"]; exists {
					switch recsList := recs.(type) {
					case []interface{}:
						for _, rec := range recsList {
							if recStr, ok := rec.(string); ok {
								recommendations = append(recommendations, recStr)
							} else if recMap, ok := rec.(map[string]interface{}); ok {
								if solution, exists := recMap["solution"]; exists {
									recommendations = append(recommendations, fmt.Sprintf("%v", solution))
								}
							}
						}
					case []string:
						recommendations = append(recommendations, recsList...)
					case string:
						recommendations = append(recommendations, recsList)
					}
				}
			}
		}
	}

	// Add default recommendations if none found
	if len(recommendations) == 0 {
		if !result.Success {
			recommendations = append(recommendations, "Check network configuration and retry")
		} else {
			recommendations = append(recommendations, "Network diagnosis completed successfully")
		}
	}

	return recommendations
}

// generateExecutionDetails generates detailed execution information
func (adapter *WorkflowMCPAdapter) generateExecutionDetails(result *workflow.WorkflowResult) map[string]interface{} {
	details := map[string]interface{}{
		"total_steps":       len(result.Steps),
		"successful_steps":  0,
		"failed_steps":      0,
		"skipped_steps":     0,
		"total_duration_ms": result.Duration.Milliseconds(),
		"steps":             make([]map[string]interface{}, 0, len(result.Steps)),
	}

	successfulSteps := 0
	failedSteps := 0
	skippedSteps := 0

	stepDetails := details["steps"].([]map[string]interface{})

	for _, step := range result.Steps {
		stepDetail := map[string]interface{}{
			"step_id":     step.StepID,
			"step_name":   step.StepName,
			"success":     step.Success,
			"skipped":     step.Skipped,
			"duration_ms": step.Duration.Milliseconds(),
		}

		if step.ToolName != "" {
			stepDetail["tool_name"] = step.ToolName
		}

		if step.Error != "" {
			stepDetail["error"] = step.Error
		}

		if step.Success || step.Skipped {
			successfulSteps++
		}
		if step.Skipped {
			skippedSteps++
		}
		if !step.Success && !step.Skipped {
			failedSteps++
		}

		stepDetails = append(stepDetails, stepDetail)
	}

	details["successful_steps"] = successfulSteps
	details["failed_steps"] = failedSteps
	details["skipped_steps"] = skippedSteps
	details["steps"] = stepDetails

	return details
}
