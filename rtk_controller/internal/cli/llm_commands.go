package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"rtk_controller/internal/workflow"
	"rtk_controller/pkg/types"
)

// handleLLMCommand handles LLM tool execution commands
func (cli *InteractiveCLI) handleLLMCommand(args []string) {
	if len(args) == 0 {
		cli.showLLMHelp()
		return
	}

	switch args[0] {
	case "list", "ls":
		cli.listLLMTools()
	case "exec", "run":
		if len(args) < 2 {
			fmt.Println("Usage: llm exec <tool_name> [params]")
			fmt.Println("Example: llm exec topology.get_full")
			return
		}
		cli.executeLLMTool(args[1], args[2:])
	case "session":
		cli.handleLLMSession(args[1:])
	case "history":
		cli.showLLMHistory(args[1:])
	case "workflow":
		cli.handleWorkflowCommand(args[1:])
	case "validate":
		cli.handleValidateCommand(args[1:])
	case "query":
		if len(args) < 2 {
			fmt.Println("Usage: llm query <natural_language_input>")
			fmt.Println("Example: llm query WiFi is weak in my bedroom")
			return
		}
		cli.processNaturalLanguageQuery(strings.Join(args[1:], " "))
	default:
		fmt.Printf("Unknown LLM subcommand: %s\n", args[0])
		cli.showLLMHelp()
	}
}

// showLLMHelp displays help for LLM commands
func (cli *InteractiveCLI) showLLMHelp() {
	fmt.Println("LLM Tool Execution Commands:")
	fmt.Println("  llm list                    - List available LLM tools")
	fmt.Println("  llm exec <tool> [params]    - Execute LLM tool directly")
	fmt.Println("  llm session <cmd> [args]    - Manage LLM sessions")
	fmt.Println("  llm history [limit]         - Show execution history")
	fmt.Println("  llm workflow <cmd> [args]   - Manage workflows")
	fmt.Println("  llm validate <cmd> [args]   - Validate workflow configurations")
	fmt.Println("  llm query <input>           - Process natural language query")
	fmt.Println("")
	fmt.Println("Workflow Commands:")
	fmt.Println("  llm workflow list           - List available workflows")
	fmt.Println("  llm workflow show <id>      - Show workflow details")
	fmt.Println("  llm workflow exec <id>      - Execute specific workflow")
	fmt.Println("  llm workflow reload         - Reload workflow configuration")
	fmt.Println("")
	fmt.Println("Validation Commands:")
	fmt.Println("  llm validate config [file]  - Validate workflow config file")
	fmt.Println("  llm validate workflow <id>  - Validate specific workflow")
	fmt.Println("")
	fmt.Println("Available Tools:")
	fmt.Println("  topology.get_full           - Get complete network topology")
	fmt.Println("  clients.list               - List all network clients")
	fmt.Println("  qos.get_status             - Get QoS status")
	fmt.Println("  traffic.get_stats          - Get traffic statistics")
	fmt.Println("  network.speedtest_full     - Run comprehensive speed test")
	fmt.Println("  diagnostics.wan_connectivity - Test WAN connectivity")
	fmt.Println("")
	fmt.Println("Session Commands:")
	fmt.Println("  llm session create [device_id] - Create new session")
	fmt.Println("  llm session list               - List active sessions")
	fmt.Println("  llm session close <session_id> - Close session")
	fmt.Println("  llm session show <session_id>  - Show session details")
}

// listLLMTools lists all available LLM tools
func (cli *InteractiveCLI) listLLMTools() {
	fmt.Println("Available LLM Diagnostic Tools:")
	fmt.Println("================================")

	tools := cli.diagnosisManager.ListLLMTools()
	if len(tools) == 0 {
		fmt.Println("No LLM tools available (LLM engine not initialized)")
		return
	}

	categories := map[string][]string{
		"Read Tools": {},
		"Test Tools": {},
		"Act Tools":  {},
	}

	for _, toolName := range tools {
		switch {
		case strings.HasPrefix(toolName, "topology.") || strings.HasPrefix(toolName, "clients.") ||
			strings.HasPrefix(toolName, "qos.") || strings.HasPrefix(toolName, "traffic."):
			categories["Read Tools"] = append(categories["Read Tools"], toolName)
		case strings.HasPrefix(toolName, "network.") || strings.HasPrefix(toolName, "diagnostics."):
			categories["Test Tools"] = append(categories["Test Tools"], toolName)
		default:
			categories["Act Tools"] = append(categories["Act Tools"], toolName)
		}
	}

	for category, toolList := range categories {
		if len(toolList) > 0 {
			fmt.Printf("\n%s:\n", category)
			for _, tool := range toolList {
				fmt.Printf("  • %s\n", tool)
			}
		}
	}
	fmt.Printf("\nTotal: %d tools available\n", len(tools))
}

// executeLLMTool executes a specified LLM tool
func (cli *InteractiveCLI) executeLLMTool(toolName string, paramArgs []string) {
	// Parse parameters from command line arguments
	params := make(map[string]interface{})
	for _, arg := range paramArgs {
		if strings.Contains(arg, "=") {
			parts := strings.SplitN(arg, "=", 2)
			key := parts[0]
			value := parts[1]

			// Try to parse as different types
			if value == "true" {
				params[key] = true
			} else if value == "false" {
				params[key] = false
			} else if num, err := strconv.ParseFloat(value, 64); err == nil {
				params[key] = num
			} else {
				params[key] = value
			}
		}
	}

	fmt.Printf("Executing tool: %s\n", toolName)
	if len(params) > 0 {
		fmt.Printf("Parameters: %v\n", params)
	}
	fmt.Println(strings.Repeat("-", 50))

	// Create a temporary session for the tool execution
	session, err := cli.diagnosisManager.CreateLLMSession(
		context.Background(),
		"",         // device_id
		"cli-user", // user_id
		map[string]interface{}{
			"source": "interactive_cli",
			"tool":   toolName,
		},
	)
	if err != nil {
		fmt.Printf("Error creating LLM session: %v\n", err)
		return
	}

	startTime := time.Now()

	// Execute the tool
	result, err := cli.diagnosisManager.ExecuteLLMTool(
		context.Background(),
		session.SessionID,
		toolName,
		params,
	)

	executionTime := time.Since(startTime)

	if err != nil {
		fmt.Printf("❌ Tool execution failed: %v\n", err)
		cli.diagnosisManager.CloseLLMSession(session.SessionID, types.LLMSessionStatusFailed)
		return
	}

	// Display results
	cli.displayLLMResult(result, executionTime)

	// Close the session
	cli.diagnosisManager.CloseLLMSession(session.SessionID, types.LLMSessionStatusCompleted)
}

// displayLLMResult displays the result of an LLM tool execution
func (cli *InteractiveCLI) displayLLMResult(result *types.ToolResult, executionTime time.Duration) {
	if result == nil {
		fmt.Println("❌ No result returned")
		return
	}

	if result.Success {
		fmt.Printf("✅ Tool executed successfully in %v\n", executionTime)
	} else {
		fmt.Printf("❌ Tool execution failed: %s\n", result.Error)
		return
	}

	if result.Data != nil {
		fmt.Println("\n📊 Result Data:")
		fmt.Println(strings.Repeat("-", 30))

		// Pretty print JSON data
		jsonData, err := json.MarshalIndent(result.Data, "", "  ")
		if err != nil {
			fmt.Printf("Error formatting data: %v\n", err)
			fmt.Printf("Raw data: %+v\n", result.Data)
		} else {
			fmt.Println(string(jsonData))
		}
	}

	fmt.Printf("\n⏱️  Execution Time: %v\n", executionTime)
	fmt.Printf("🔢 Session ID: %s\n", result.SessionID)
	if result.TraceID != "" {
		fmt.Printf("🔍 Trace ID: %s\n", result.TraceID)
	}
}

// handleLLMSession handles LLM session management commands
func (cli *InteractiveCLI) handleLLMSession(args []string) {
	if len(args) == 0 {
		fmt.Println("Session subcommands: create, list, close, show")
		return
	}

	switch args[0] {
	case "create":
		cli.createLLMSession(args[1:])
	case "list":
		cli.listLLMSessions()
	case "close":
		if len(args) < 2 {
			fmt.Println("Usage: llm session close <session_id>")
			return
		}
		cli.closeLLMSession(args[1])
	case "show":
		if len(args) < 2 {
			fmt.Println("Usage: llm session show <session_id>")
			return
		}
		cli.showLLMSession(args[1])
	default:
		fmt.Printf("Unknown session subcommand: %s\n", args[0])
	}
}

// createLLMSession creates a new LLM session
func (cli *InteractiveCLI) createLLMSession(args []string) {
	var deviceID string
	if len(args) > 0 {
		deviceID = args[0]
	}

	session, err := cli.diagnosisManager.CreateLLMSession(
		context.Background(),
		deviceID,
		"cli-user",
		map[string]interface{}{
			"source": "interactive_cli",
			"manual": true,
		},
	)
	if err != nil {
		fmt.Printf("Error creating session: %v\n", err)
		return
	}

	fmt.Printf("✅ Created LLM session: %s\n", session.SessionID)
	if deviceID != "" {
		fmt.Printf("   Device ID: %s\n", deviceID)
	}
	fmt.Printf("   Created at: %s\n", session.CreatedAt.Format(time.RFC3339))
}

// listLLMSessions lists all active LLM sessions
func (cli *InteractiveCLI) listLLMSessions() {
	// This would require implementing a method to list sessions in diagnosis manager
	fmt.Println("Session listing is not yet fully implemented")
	fmt.Println("This would show all active LLM diagnostic sessions")
}

// closeLLMSession closes an LLM session
func (cli *InteractiveCLI) closeLLMSession(sessionID string) {
	err := cli.diagnosisManager.CloseLLMSession(sessionID, types.LLMSessionStatusCompleted)
	if err != nil {
		fmt.Printf("Error closing session: %v\n", err)
		return
	}
	fmt.Printf("✅ Closed session: %s\n", sessionID)
}

// showLLMSession shows details of an LLM session
func (cli *InteractiveCLI) showLLMSession(sessionID string) {
	session, err := cli.diagnosisManager.GetLLMSession(sessionID)
	if err != nil {
		fmt.Printf("Error retrieving session: %v\n", err)
		return
	}

	fmt.Printf("LLM Session Details: %s\n", sessionID)
	fmt.Println(strings.Repeat("=", 40))
	fmt.Printf("Status: %s\n", session.Status)
	fmt.Printf("Device ID: %s\n", session.DeviceID)
	fmt.Printf("User ID: %s\n", session.UserID)
	fmt.Printf("Created: %s\n", session.CreatedAt.Format(time.RFC3339))
	fmt.Printf("Updated: %s\n", session.UpdatedAt.Format(time.RFC3339))
	fmt.Printf("Trace ID: %s\n", session.TraceID)

	if len(session.ToolCalls) > 0 {
		fmt.Printf("\nTool Calls (%d):\n", len(session.ToolCalls))
		fmt.Println(strings.Repeat("-", 20))
		for i, call := range session.ToolCalls {
			fmt.Printf("%d. %s (%s)\n", i+1, call.ToolName, call.Status)
			if call.CompletedAt != nil {
				duration := call.CompletedAt.Sub(call.StartedAt)
				fmt.Printf("   Duration: %v\n", duration)
			}
		}
	}

	if len(session.Metadata) > 0 {
		fmt.Println("\nMetadata:")
		fmt.Println(strings.Repeat("-", 10))
		for key, value := range session.Metadata {
			fmt.Printf("  %s: %v\n", key, value)
		}
	}
}

// showLLMHistory shows LLM tool execution history
func (cli *InteractiveCLI) showLLMHistory(args []string) {
	limit := 10 // default limit
	if len(args) > 0 {
		if l, err := strconv.Atoi(args[0]); err == nil && l > 0 {
			limit = l
		}
	}

	// This would require implementing a method to get execution history
	fmt.Printf("Showing last %d LLM tool executions:\n", limit)
	fmt.Println("(History retrieval not yet fully implemented)")
	fmt.Println("This would show recent tool executions with results")
}

// handleWorkflowCommand handles workflow management commands
func (cli *InteractiveCLI) handleWorkflowCommand(args []string) {
	if len(args) == 0 {
		cli.showWorkflowHelp()
		return
	}

	switch args[0] {
	case "list", "ls":
		cli.listWorkflows()
	case "show", "describe":
		if len(args) < 2 {
			fmt.Println("Usage: llm workflow show <workflow_id>")
			return
		}
		cli.showWorkflow(args[1])
	case "exec", "execute", "run":
		if len(args) < 2 {
			fmt.Println("Usage: llm workflow exec <workflow_id> [key=value ...]")
			return
		}
		cli.executeWorkflow(args[1], args[2:])
	case "reload":
		cli.reloadWorkflows()
	default:
		fmt.Printf("Unknown workflow subcommand: %s\n", args[0])
		cli.showWorkflowHelp()
	}
}

// showWorkflowHelp displays help for workflow commands
func (cli *InteractiveCLI) showWorkflowHelp() {
	fmt.Println("Workflow Management Commands:")
	fmt.Println("  llm workflow list                      - List available workflows")
	fmt.Println("  llm workflow show <id>                 - Show workflow details")
	fmt.Println("  llm workflow exec <id> [key=value ...] - Execute specific workflow")
	fmt.Println("  llm workflow reload                    - Reload workflow configuration")
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Println("  llm workflow list")
	fmt.Println("  llm workflow show weak_signal_coverage_diagnosis")
	fmt.Println("  llm workflow exec weak_signal_coverage_diagnosis location1=\"living room\" location2=\"bedroom\"")
}

// listWorkflows lists available workflows
func (cli *InteractiveCLI) listWorkflows() {
	// Get workflow manager from diagnosis manager
	// For now, this is a placeholder implementation
	workflowManager := cli.diagnosisManager.GetWorkflowManager()
	if workflowManager == nil {
		fmt.Println("Workflow manager not available")
		return
	}

	workflows := workflowManager.ListWorkflows()
	if len(workflows) == 0 {
		fmt.Println("No workflows available")
		return
	}

	fmt.Println("Available Workflows:")
	fmt.Println(strings.Repeat("=", 50))

	for _, workflowID := range workflows {
		workflowInterface, err := workflowManager.GetWorkflow(workflowID)
		if err != nil {
			fmt.Printf("✗ %s (error: %v)\n", workflowID, err)
			continue
		}

		// Type assertion to workflow.Workflow
		workflow, ok := workflowInterface.(*workflow.Workflow)
		if !ok {
			fmt.Printf("✗ %s (type assertion failed)\n", workflowID)
			continue
		}

		fmt.Printf("✓ %s\n", workflowID)
		fmt.Printf("  Name: %s\n", workflow.Name)
		fmt.Printf("  Description: %s\n", workflow.Description)
		fmt.Printf("  Intent: %s/%s\n", workflow.Intent.Primary, workflow.Intent.Secondary)
		fmt.Printf("  Steps: %d\n", len(workflow.Steps))
		if len(workflow.Metadata.Tags) > 0 {
			fmt.Printf("  Tags: %v\n", workflow.Metadata.Tags)
		}
		fmt.Println()
	}
}

// showWorkflow shows details of a specific workflow
func (cli *InteractiveCLI) showWorkflow(workflowID string) {
	workflowManager := cli.diagnosisManager.GetWorkflowManager()
	if workflowManager == nil {
		fmt.Println("Workflow manager not available")
		return
	}

	workflowInterface, err := workflowManager.GetWorkflow(workflowID)
	if err != nil {
		fmt.Printf("Error retrieving workflow: %v\n", err)
		return
	}

	// Type assertion to workflow.Workflow
	workflow, ok := workflowInterface.(*workflow.Workflow)
	if !ok {
		fmt.Printf("Type assertion failed for workflow: %s\n", workflowID)
		return
	}

	fmt.Printf("Workflow Details: %s\n", workflowID)
	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("ID: %s\n", workflow.ID)
	fmt.Printf("Name: %s\n", workflow.Name)
	fmt.Printf("Description: %s\n", workflow.Description)
	fmt.Printf("Version: %s\n", workflow.Metadata.Version)
	fmt.Printf("Author: %s\n", workflow.Metadata.Author)

	fmt.Printf("\nIntent Mapping:\n")
	fmt.Printf("  Primary: %s\n", workflow.Intent.Primary)
	fmt.Printf("  Secondary: %s\n", workflow.Intent.Secondary)

	if len(workflow.Metadata.Tags) > 0 {
		fmt.Printf("\nTags: %v\n", workflow.Metadata.Tags)
	}

	if len(workflow.Metadata.Requirements) > 0 {
		fmt.Printf("\nRequirements: %v\n", workflow.Metadata.Requirements)
	}

	fmt.Printf("\nWorkflow Steps (%d):\n", len(workflow.Steps))
	fmt.Println(strings.Repeat("-", 40))
	cli.printWorkflowSteps(workflow.Steps, 0)
}

// printWorkflowSteps recursively prints workflow steps
func (cli *InteractiveCLI) printWorkflowSteps(steps []workflow.WorkflowStep, indent int) {
	prefix := strings.Repeat("  ", indent)

	for i, step := range steps {
		fmt.Printf("%s%d. [%s] %s\n", prefix, i+1, step.Type, step.ID)
		if step.Name != "" {
			fmt.Printf("%s   Name: %s\n", prefix, step.Name)
		}
		if step.ToolName != "" {
			fmt.Printf("%s   Tool: %s\n", prefix, step.ToolName)
		}
		if step.Timeout != nil {
			fmt.Printf("%s   Timeout: %v\n", prefix, *step.Timeout)
		}
		if step.Optional {
			fmt.Printf("%s   Optional: true\n", prefix)
		}
		if step.Condition != nil {
			fmt.Printf("%s   Condition: %s %s %v\n", prefix,
				step.Condition.Field, step.Condition.Operator, step.Condition.Value)
		}

		if len(step.SubSteps) > 0 {
			fmt.Printf("%s   Sub-steps:\n", prefix)
			cli.printWorkflowSteps(step.SubSteps, indent+2)
		}

		fmt.Println()
	}
}

// executeWorkflow executes a specific workflow
func (cli *InteractiveCLI) executeWorkflow(workflowID string, paramArgs []string) {
	workflowManager := cli.diagnosisManager.GetWorkflowManager()
	if workflowManager == nil {
		fmt.Println("Workflow manager not available")
		return
	}

	// Parse parameters from key=value format
	params := make(map[string]interface{})
	for _, arg := range paramArgs {
		parts := strings.SplitN(arg, "=", 2)
		if len(parts) == 2 {
			params[parts[0]] = parts[1]
		} else {
			fmt.Printf("Warning: Invalid parameter format '%s', expected key=value\n", arg)
		}
	}

	fmt.Printf("Executing workflow: %s\n", workflowID)
	if len(params) > 0 {
		fmt.Printf("Parameters: %v\n", params)
	}
	fmt.Println(strings.Repeat("=", 50))

	ctx := context.Background()
	resultInterface, err := workflowManager.ExecuteWorkflow(ctx, workflowID, params)
	if err != nil {
		fmt.Printf("Error executing workflow: %v\n", err)
		return
	}

	// Type assertion to workflow.WorkflowResult
	result, ok := resultInterface.(*workflow.WorkflowResult)
	if !ok {
		fmt.Printf("Type assertion failed for workflow result\n")
		return
	}

	cli.displayWorkflowResult(result)
}

// processNaturalLanguageQuery processes natural language input through workflow engine
func (cli *InteractiveCLI) processNaturalLanguageQuery(userInput string) {
	workflowManager := cli.diagnosisManager.GetWorkflowManager()
	if workflowManager == nil {
		fmt.Println("Workflow manager not available")
		return
	}

	fmt.Printf("Processing query: %s\n", userInput)
	fmt.Println(strings.Repeat("=", 50))

	ctx := context.Background()
	resultInterface, err := workflowManager.ProcessUserInput(ctx, userInput, map[string]string{
		"source":    "cli",
		"timestamp": time.Now().Format(time.RFC3339),
	})

	if err != nil {
		fmt.Printf("Error processing query: %v\n", err)
		return
	}

	// Type assertion to workflow.WorkflowResult
	result, ok := resultInterface.(*workflow.WorkflowResult)
	if !ok {
		fmt.Printf("Type assertion failed for workflow result\n")
		return
	}

	cli.displayWorkflowResult(result)
}

// displayWorkflowResult displays workflow execution results
func (cli *InteractiveCLI) displayWorkflowResult(result *workflow.WorkflowResult) {
	if result == nil {
		fmt.Println("No result available")
		return
	}

	status := "✓ SUCCESS"
	if !result.Success {
		status = "✗ FAILED"
	}

	fmt.Printf("\nWorkflow Result: %s [%s]\n", result.WorkflowID, status)
	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("Duration: %v\n", result.Duration)
	fmt.Printf("Steps: %d/%d successful\n", cli.countSuccessfulSteps(result.Steps), len(result.Steps))

	if result.Summary != "" {
		fmt.Printf("Summary: %s\n", result.Summary)
	}

	if result.Error != "" {
		fmt.Printf("Error: %s\n", result.Error)
	}

	// Show step details
	if len(result.Steps) > 0 {
		fmt.Println("\nStep Results:")
		fmt.Println(strings.Repeat("-", 40))
		for i, step := range result.Steps {
			stepStatus := "✓"
			if !step.Success {
				if step.Skipped {
					stepStatus = "⏭"
				} else {
					stepStatus = "✗"
				}
			}

			fmt.Printf("  %d. %s [%s] (%v)\n",
				i+1, step.StepID, stepStatus, step.Duration)

			if step.Error != "" {
				fmt.Printf("     Error: %s\n", step.Error)
			}

			if step.ToolResult != nil && step.ToolResult.Success {
				fmt.Printf("     Tool: %s - Success\n", step.ToolResult.ToolName)
				if len(fmt.Sprintf("%v", step.ToolResult.Data)) < 200 {
					// Show brief data preview for small results
					fmt.Printf("     Data: %v\n", step.ToolResult.Data)
				}
			}
		}
	}

	// Show metadata if available
	if len(result.Metadata) > 0 {
		fmt.Println("\nMetadata:")
		fmt.Println(strings.Repeat("-", 20))
		for key, value := range result.Metadata {
			fmt.Printf("  %s: %v\n", key, value)
		}
	}
}

// reloadWorkflows reloads workflow configuration
func (cli *InteractiveCLI) reloadWorkflows() {
	workflowManager := cli.diagnosisManager.GetWorkflowManager()
	if workflowManager == nil {
		fmt.Println("Workflow manager not available")
		return
	}

	fmt.Println("Reloading workflow configuration...")
	err := workflowManager.ReloadConfiguration()
	if err != nil {
		fmt.Printf("Error reloading workflows: %v\n", err)
		return
	}

	workflows := workflowManager.ListWorkflows()
	fmt.Printf("✓ Successfully reloaded %d workflows\n", len(workflows))
}

// countSuccessfulSteps counts successful steps in a workflow result
func (cli *InteractiveCLI) countSuccessfulSteps(steps []workflow.StepResult) int {
	count := 0
	for _, step := range steps {
		if step.Success || step.Skipped {
			count++
		}
	}
	return count
}

// handleValidateCommand handles workflow configuration validation commands
func (cli *InteractiveCLI) handleValidateCommand(args []string) {
	if len(args) == 0 {
		fmt.Println("Usage:")
		fmt.Println("  llm validate config                    - Validate default workflow config")
		fmt.Println("  llm validate config <file_path>        - Validate specific config file")
		fmt.Println("  llm validate workflow <workflow_id>    - Validate specific workflow")
		return
	}

	switch args[0] {
	case "config":
		filePath := "configs/workflows.yaml"
		if len(args) > 1 {
			filePath = args[1]
		}
		cli.validateWorkflowConfig(filePath)
	case "workflow":
		if len(args) < 2 {
			fmt.Println("Usage: llm validate workflow <workflow_id>")
			return
		}
		cli.validateSpecificWorkflow(args[1])
	default:
		fmt.Printf("Unknown validate subcommand: %s\n", args[0])
		fmt.Println("Available subcommands: config, workflow")
	}
}

// validateWorkflowConfig validates a workflow configuration file
func (cli *InteractiveCLI) validateWorkflowConfig(filePath string) {
	validator := workflow.NewConfigValidator()
	
	fmt.Printf("Validating workflow configuration: %s\n", filePath)
	fmt.Println(strings.Repeat("=", 50))
	
	result, err := validator.ValidateWorkflowFile(filePath)
	if err != nil {
		fmt.Printf("✗ Failed to validate file: %v\n", err)
		return
	}
	
	if result.IsValid {
		fmt.Printf("✓ Configuration is valid!\n")
		if len(result.Warnings) > 0 {
			fmt.Printf("\nWarnings (%d):\n", len(result.Warnings))
			for i, warning := range result.Warnings {
				fmt.Printf("  %d. %s\n", i+1, warning.Error())
			}
		}
	} else {
		fmt.Printf("✗ Configuration is invalid!\n")
		
		if len(result.Errors) > 0 {
			fmt.Printf("\nErrors (%d):\n", len(result.Errors))
			for i, err := range result.Errors {
				fmt.Printf("  %d. %s\n", i+1, err.Error())
			}
		}
		
		if len(result.Warnings) > 0 {
			fmt.Printf("\nWarnings (%d):\n", len(result.Warnings))
			for i, warning := range result.Warnings {
				fmt.Printf("  %d. %s\n", i+1, warning.Error())
			}
		}
	}
}

// validateSpecificWorkflow validates a specific workflow by ID
func (cli *InteractiveCLI) validateSpecificWorkflow(workflowID string) {
	workflowManager := cli.diagnosisManager.GetWorkflowManager()
	if workflowManager == nil {
		fmt.Println("Workflow manager not available")
		return
	}
	
	workflowInterface, err := workflowManager.GetWorkflow(workflowID)
	if err != nil {
		fmt.Printf("✗ Failed to get workflow '%s': %v\n", workflowID, err)
		return
	}
	
	// Type assertion to workflow.Workflow
	workflowObj, ok := workflowInterface.(*workflow.Workflow)
	if !ok {
		fmt.Printf("✗ Type assertion failed for workflow: %s\n", workflowID)
		return
	}
	
	fmt.Printf("Validating workflow: %s\n", workflowID)
	fmt.Println(strings.Repeat("=", 50))
	
	// Create a map with just this workflow for validation
	workflows := map[string]workflow.Workflow{
		workflowID: *workflowObj,
	}
	
	validator := workflow.NewConfigValidator()
	result := validator.ValidateWorkflows(workflows)
	
	if result.IsValid {
		fmt.Printf("✓ Workflow '%s' is valid!\n", workflowID)
		if len(result.Warnings) > 0 {
			fmt.Printf("\nWarnings (%d):\n", len(result.Warnings))
			for i, warning := range result.Warnings {
				fmt.Printf("  %d. %s\n", i+1, warning.Error())
			}
		}
	} else {
		fmt.Printf("✗ Workflow '%s' is invalid!\n", workflowID)
		
		if len(result.Errors) > 0 {
			fmt.Printf("\nErrors (%d):\n", len(result.Errors))
			for i, err := range result.Errors {
				fmt.Printf("  %d. %s\n", i+1, err.Error())
			}
		}
		
		if len(result.Warnings) > 0 {
			fmt.Printf("\nWarnings (%d):\n", len(result.Warnings))
			for i, warning := range result.Warnings {
				fmt.Printf("  %d. %s\n", i+1, warning.Error())
			}
		}
	}
}
