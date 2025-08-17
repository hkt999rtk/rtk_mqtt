package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// Test tool structures for advanced testing scenarios
type TestResult struct {
	TestName    string      `json:"test_name"`
	Success     bool        `json:"success"`
	Message     string      `json:"message"`
	Data        interface{} `json:"data,omitempty"`
	Timestamp   string      `json:"timestamp"`
	Duration    string      `json:"duration"`
}

// AddTestTools adds testing-related tools to the MCP server
func AddTestTools(s *server.MCPServer) {
	log.Printf("🔧 Tool Registry: Starting to register test tools...")

	// Test input validation tool
	validationTool := mcp.NewTool("test_input_validation",
		mcp.WithDescription("Test input validation and sanitization with various edge cases"),
		mcp.WithString("input",
			mcp.Required(),
			mcp.Description("Input string to test validation against"),
		),
		mcp.WithString("test_type",
			mcp.Description("Type of validation test: sql_injection, xss, path_traversal, command_injection, unicode"),
		),
	)

	s.AddTool(validationTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		log.Printf("🧪 MCP Server: Received test_input_validation tool call request")

		input, err := request.RequireString("input")
		if err != nil {
			return mcp.NewToolResultError("input parameter is required"), nil
		}

		testType := request.GetString("test_type", "general")
		
		result := testInputValidation(input, testType)
		jsonData, _ := json.MarshalIndent(result, "", "  ")
		
		log.Printf("✅ MCP Server: test_input_validation completed - test_type: %s, success: %t", testType, result.Success)
		return mcp.NewToolResultText(string(jsonData)), nil
	})

	// Test error simulation tool
	errorTool := mcp.NewTool("test_error_simulation",
		mcp.WithDescription("Simulate various error conditions for testing error handling"),
		mcp.WithString("error_type",
			mcp.Required(),
			mcp.Description("Type of error to simulate: network_timeout, api_failure, invalid_response, rate_limit, service_unavailable"),
		),
		mcp.WithString("severity",
			mcp.Description("Error severity: low, medium, high, critical"),
		),
		mcp.WithBoolean("recoverable",
			mcp.Description("Whether the error should be recoverable (default: true)"),
		),
	)

	s.AddTool(errorTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		log.Printf("🧪 MCP Server: Received test_error_simulation tool call request")

		errorType, err := request.RequireString("error_type")
		if err != nil {
			return mcp.NewToolResultError("error_type parameter is required"), nil
		}

		severity := request.GetString("severity", "medium")
		recoverable := request.GetBool("recoverable", true)
		
		result := simulateError(errorType, severity, recoverable)
		jsonData, _ := json.MarshalIndent(result, "", "  ")
		
		log.Printf("✅ MCP Server: test_error_simulation completed - type: %s, severity: %s, recoverable: %t", 
			errorType, severity, recoverable)
		return mcp.NewToolResultText(string(jsonData)), nil
	})

	// Test tool chain complexity
	chainTool := mcp.NewTool("test_tool_chain",
		mcp.WithDescription("Test complex tool chaining scenarios with dependencies and conditional execution"),
		mcp.WithString("scenario",
			mcp.Required(),
			mcp.Description("Test scenario: sequential, parallel, conditional, circular_dependency, error_cascade"),
		),
		mcp.WithString("complexity",
			mcp.Description("Complexity level: simple, moderate, complex, extreme"),
		),
	)

	s.AddTool(chainTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		log.Printf("🧪 MCP Server: Received test_tool_chain tool call request")

		scenario, err := request.RequireString("scenario")
		if err != nil {
			return mcp.NewToolResultError("scenario parameter is required"), nil
		}

		complexity := request.GetString("complexity", "moderate")
		
		result := testToolChain(scenario, complexity)
		jsonData, _ := json.MarshalIndent(result, "", "  ")
		
		log.Printf("✅ MCP Server: test_tool_chain completed - scenario: %s, complexity: %s", scenario, complexity)
		return mcp.NewToolResultText(string(jsonData)), nil
	})

	log.Printf("✅ Tool Registry: Successfully registered test tools (3 tools registered)")
}

// testInputValidation performs input validation tests
func testInputValidation(input, testType string) TestResult {
	start := time.Now()
	result := TestResult{
		TestName:  fmt.Sprintf("input_validation_%s", testType),
		Timestamp: time.Now().Format("2006-01-02 15:04:05"),
	}

	// Define dangerous patterns
	sqlPatterns := []string{"'", "--", "DROP", "DELETE", "INSERT", "UPDATE", "UNION", "SELECT"}
	xssPatterns := []string{"<script>", "</script>", "javascript:", "onload=", "onerror="}
	pathPatterns := []string{"../", "..\\", "/etc/passwd", "C:\\Windows"}
	cmdPatterns := []string{"|", "&", ";", "$(", "`", "rm -rf", "format c:"}

	var detectedPatterns []string
	inputLower := strings.ToLower(input)

	switch testType {
	case "sql_injection":
		for _, pattern := range sqlPatterns {
			if strings.Contains(inputLower, strings.ToLower(pattern)) {
				detectedPatterns = append(detectedPatterns, pattern)
			}
		}
	case "xss":
		for _, pattern := range xssPatterns {
			if strings.Contains(inputLower, strings.ToLower(pattern)) {
				detectedPatterns = append(detectedPatterns, pattern)
			}
		}
	case "path_traversal":
		for _, pattern := range pathPatterns {
			if strings.Contains(inputLower, strings.ToLower(pattern)) {
				detectedPatterns = append(detectedPatterns, pattern)
			}
		}
	case "command_injection":
		for _, pattern := range cmdPatterns {
			if strings.Contains(input, pattern) {
				detectedPatterns = append(detectedPatterns, pattern)
			}
		}
	case "unicode":
		// Check for suspicious Unicode characters
		for _, r := range input {
			if r > 127 && (r < 160 || r > 255) {
				detectedPatterns = append(detectedPatterns, fmt.Sprintf("U+%04X", r))
			}
		}
	default:
		// General validation - check all patterns
		allPatterns := append(append(append(sqlPatterns, xssPatterns...), pathPatterns...), cmdPatterns...)
		for _, pattern := range allPatterns {
			if strings.Contains(inputLower, strings.ToLower(pattern)) {
				detectedPatterns = append(detectedPatterns, pattern)
			}
		}
	}

	result.Success = len(detectedPatterns) == 0
	if result.Success {
		result.Message = fmt.Sprintf("Input validation passed - no dangerous patterns detected for %s test", testType)
	} else {
		result.Message = fmt.Sprintf("Input validation failed - detected patterns: %v", detectedPatterns)
	}
	result.Data = map[string]interface{}{
		"input":             input,
		"test_type":         testType,
		"detected_patterns": detectedPatterns,
		"input_length":      len(input),
		"is_empty":          len(strings.TrimSpace(input)) == 0,
	}
	result.Duration = time.Since(start).String()

	return result
}

// simulateError simulates various error conditions
func simulateError(errorType, severity string, recoverable bool) TestResult {
	start := time.Now()
	result := TestResult{
		TestName:  fmt.Sprintf("error_simulation_%s", errorType),
		Timestamp: time.Now().Format("2006-01-02 15:04:05"),
	}

	// Simulate different error conditions
	switch errorType {
	case "network_timeout":
		time.Sleep(100 * time.Millisecond) // Simulate timeout delay
		result.Success = recoverable
		result.Message = fmt.Sprintf("Network timeout simulation - severity: %s, recoverable: %t", severity, recoverable)
		
	case "api_failure":
		result.Success = recoverable && severity != "critical"
		result.Message = fmt.Sprintf("API failure simulation - HTTP 500/503 error, severity: %s", severity)
		
	case "invalid_response":
		result.Success = recoverable
		result.Message = fmt.Sprintf("Invalid response simulation - malformed JSON, severity: %s", severity)
		
	case "rate_limit":
		result.Success = recoverable
		result.Message = fmt.Sprintf("Rate limit simulation - HTTP 429 error, severity: %s", severity)
		
	case "service_unavailable":
		result.Success = recoverable && severity != "critical"
		result.Message = fmt.Sprintf("Service unavailable simulation - HTTP 503 error, severity: %s", severity)
		
	default:
		result.Success = false
		result.Message = fmt.Sprintf("Unknown error type: %s", errorType)
	}

	result.Data = map[string]interface{}{
		"error_type":  errorType,
		"severity":    severity,
		"recoverable": recoverable,
		"error_code":  getErrorCode(errorType),
		"retry_after": getRetryDelay(severity),
	}
	result.Duration = time.Since(start).String()

	return result
}

// testToolChain tests complex tool chaining scenarios
func testToolChain(scenario, complexity string) TestResult {
	start := time.Now()
	result := TestResult{
		TestName:  fmt.Sprintf("tool_chain_%s", scenario),
		Timestamp: time.Now().Format("2006-01-02 15:04:05"),
	}

	var steps []string
	var dependencies []string

	switch scenario {
	case "sequential":
		steps = []string{"step1: validate_input", "step2: process_data", "step3: generate_output"}
		dependencies = []string{"step2 depends on step1", "step3 depends on step2"}
		
	case "parallel":
		steps = []string{"step1a: fetch_weather", "step1b: fetch_time", "step2: combine_results"}
		dependencies = []string{"step2 depends on step1a and step1b"}
		
	case "conditional":
		steps = []string{"step1: check_condition", "step2a: path_true", "step2b: path_false", "step3: merge_paths"}
		dependencies = []string{"step2a/step2b depends on step1 condition", "step3 depends on chosen path"}
		
	case "circular_dependency":
		steps = []string{"step1: init", "step2: process", "step3: validate", "step1: retry"}
		dependencies = []string{"CIRCULAR: step1->step2->step3->step1"}
		result.Success = false // Circular dependencies should fail
		
	case "error_cascade":
		steps = []string{"step1: fail", "step2: handle_error", "step3: cascade_failure"}
		dependencies = []string{"step2 handles step1 failure", "step3 fails due to cascade"}
		result.Success = false // Error cascades should be caught
		
	default:
		result.Success = false
		result.Message = fmt.Sprintf("Unknown scenario: %s", scenario)
		result.Duration = time.Since(start).String()
		return result
	}

	// Simulate complexity based processing time
	processingTime := getProcessingTime(complexity)
	time.Sleep(processingTime)

	if result.Success != false { // Only set success if not already set to false
		result.Success = true
	}
	
	result.Message = fmt.Sprintf("Tool chain test completed - scenario: %s, complexity: %s", scenario, complexity)
	result.Data = map[string]interface{}{
		"scenario":     scenario,
		"complexity":   complexity,
		"steps":        steps,
		"dependencies": dependencies,
		"total_steps":  len(steps),
		"processing_time": processingTime.String(),
	}
	result.Duration = time.Since(start).String()

	return result
}

// Helper functions
func getErrorCode(errorType string) int {
	switch errorType {
	case "network_timeout":
		return 408
	case "api_failure":
		return 500
	case "invalid_response":
		return 502
	case "rate_limit":
		return 429
	case "service_unavailable":
		return 503
	default:
		return 500
	}
}

func getRetryDelay(severity string) string {
	switch severity {
	case "low":
		return "1s"
	case "medium":
		return "5s"
	case "high":
		return "30s"
	case "critical":
		return "no_retry"
	default:
		return "5s"
	}
}

func getProcessingTime(complexity string) time.Duration {
	switch complexity {
	case "simple":
		return 10 * time.Millisecond
	case "moderate":
		return 50 * time.Millisecond
	case "complex":
		return 100 * time.Millisecond
	case "extreme":
		return 200 * time.Millisecond
	default:
		return 50 * time.Millisecond
	}
}