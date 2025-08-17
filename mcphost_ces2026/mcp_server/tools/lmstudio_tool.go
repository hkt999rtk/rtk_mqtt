package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"mcphost_ces2026/utils"
)

// AddLLMTools adds LLM-related tools to the MCP server using the provider registry
func AddLLMTools(s *server.MCPServer) {
	log.Printf("🔧 Tool Registry: Starting to register LLM tools...")
	
	// Chat completion tool
	chatTool := mcp.NewTool("llm_chat",
		mcp.WithDescription("Chat with configured LLM provider"),
		mcp.WithString("prompt",
			mcp.Required(),
			mcp.Description("Prompt to send to the LLM"),
		),
		mcp.WithString("system_prompt",
			mcp.Description("System prompt (optional)"),
		),
		mcp.WithString("model",
			mcp.Description("Model to use (optional, uses default if not specified)"),
		),
		mcp.WithString("provider",
			mcp.Description("LLM provider to use (optional, uses default if not specified)"),
		),
		mcp.WithNumber("temperature",
			mcp.Description("Temperature parameter (0.0-2.0), controls creativity of response"),
		),
		mcp.WithNumber("max_tokens",
			mcp.Description("Maximum number of tokens"),
		),
	)

	log.Printf("🔧 Tool Registry: LLM tool created")
	log.Printf("   └─ Name: llm_chat")
	log.Printf("   └─ Description: Chat with configured LLM provider")
	log.Printf("   └─ Parameters: prompt (required), system_prompt, model, provider, temperature, max_tokens")
	
	// Get available providers count
	if llmRegistry != nil {
		allModels := llmRegistry.GetAllModels()
		log.Printf("   └─ Available Models: %d models across configured providers", len(allModels))
	} else {
		log.Printf("   └─ Available Models: Using legacy LM Studio configuration")
	}

	s.AddTool(chatTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		log.Printf("💬 MCP Server: Received llm_chat tool call request")

		prompt, err := request.RequireString("prompt")
		if err != nil {
			log.Printf("❌ MCP Server: llm_chat parameter error - prompt parameter is required")
			return mcp.NewToolResultError("prompt parameter is required"), nil
		}

		systemPrompt := request.GetString("system_prompt", "You are a helpful assistant. Please respond in English.")
		model := request.GetString("model", "")
		providerName := request.GetString("provider", "")
		temperature := request.GetFloat("temperature", 0.7)
		maxTokens := request.GetInt("max_tokens", 2048)

		log.Printf("🤖 MCP Server: Processing chat request - provider: %s, model: %s, prompt length: %d", providerName, model, len(prompt))

		// Get LLM provider
		provider, err := llmRegistry.GetProvider(providerName)
		if err != nil {
			log.Printf("❌ MCP Server: Failed to get LLM provider: %v", err)
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get LLM provider: %v", err)), nil
		}

		// Set default model if not specified
		if model == "" {
			models := provider.GetModels()
			if len(models) > 0 {
				model = models[0].ID
			} else {
				model = "default"
			}
		}

		// Build chat messages
		messages := []ChatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: prompt},
		}

		// Create chat request
		chatReq := ChatCompletionRequest{
			Model:       model,
			Messages:    messages,
			Temperature: temperature,
			MaxTokens:   maxTokens,
		}

		// Call LLM provider
		response, err := provider.Chat(ctx, chatReq)
		if err != nil {
			log.Printf("❌ MCP Server: LLM provider call failed: %v", err)
			return mcp.NewToolResultError(fmt.Sprintf("LLM provider call failed: %v", err)), nil
		}

		if len(response.Choices) == 0 {
			log.Printf("❌ MCP Server: No response choices returned")
			return mcp.NewToolResultError("No response choices returned"), nil
		}

		responseText := response.Choices[0].Message.Content
		log.Printf("✅ MCP Server: llm_chat tool call successful, returning %d character response", len(responseText))
		return mcp.NewToolResultText(responseText), nil
	})

	log.Printf("✅ Tool Registry: Successfully registered LLM tool 'llm_chat'")
	log.Printf("🔧 Tool Registry: LLM tools registration completed (1 tool registered)")
}

// callLMStudioAPI calls the LM Studio API with the provided messages
func callLMStudioAPI(ctx context.Context, messages []ChatMessage, temperature float64, maxTokens int) (string, error) {
	return callLMStudioAPIWithModel(ctx, messages, temperature, maxTokens, config.LMStudio.Model)
}

// callLMStudioAPIWithModel calls the LM Studio API with the provided messages and specific model
func callLMStudioAPIWithModel(ctx context.Context, messages []ChatMessage, temperature float64, maxTokens int, model string) (string, error) {
	// Create the request payload
	chatReq := ChatRequest{
		Model:       model,
		Messages:    messages,
		Temperature: temperature,
		MaxTokens:   maxTokens,
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(chatReq)
	if err != nil {
		return "", fmt.Errorf("request serialization failed: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", config.LMStudio.BaseURL+"/v1/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("request creation failed: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	if config.LMStudio.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+config.LMStudio.APIKey)
	}

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Send request
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request sending failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("response reading failed: %w", err)
	}

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API returned error status code: %d, body: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var chatResp ChatResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		return "", fmt.Errorf("response parsing failed: %w", err)
	}

	// Extract the response content
	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}

	return chatResp.Choices[0].Message.Content, nil
}