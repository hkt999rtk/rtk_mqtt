package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

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
	default:
		fmt.Printf("Unknown LLM subcommand: %s\n", args[0])
		cli.showLLMHelp()
	}
}

// showLLMHelp displays help for LLM commands
func (cli *InteractiveCLI) showLLMHelp() {
	fmt.Println("LLM Tool Execution Commands:")
	fmt.Println("  llm list                    - List available LLM tools")
	fmt.Println("  llm exec <tool> [params]    - Execute LLM tool")
	fmt.Println("  llm session <cmd> [args]    - Manage LLM sessions")
	fmt.Println("  llm history [limit]         - Show execution history")
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
		"", // device_id
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