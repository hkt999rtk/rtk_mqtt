package utils

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

// LMStudioProvider implements LLMProvider interface for LM Studio
type LMStudioProvider struct {
	config   LLMProviderConfig
	client   *http.Client
	baseURL  string
	apiKey   string
	models   []string
}

// NewLMStudioProvider creates a new LM Studio provider
func NewLMStudioProvider(config LLMProviderConfig) (*LMStudioProvider, error) {
	models := config.Models
	if len(models) == 0 {
		models = []string{"local-model"}
	}
	
	return &LMStudioProvider{
		config:  config,
		client:  &http.Client{Timeout: 30 * time.Second},
		baseURL: config.BaseURL,
		apiKey:  config.APIKey,
		models:  models,
	}, nil
}

func (p *LMStudioProvider) Chat(ctx context.Context, request ChatCompletionRequest) (*ChatCompletionResponse, error) {
	log.Printf("🤖 LLM Provider: LM Studio processing request")
	log.Printf("   └─ Provider Name: %s", p.config.Name)
	log.Printf("   └─ Base URL: %s", p.baseURL)
	log.Printf("   └─ Model: %s", request.Model)
	log.Printf("   └─ Temperature: %.2f", request.Temperature)
	log.Printf("   └─ Max Tokens: %d", request.MaxTokens)
	log.Printf("   └─ Messages Count: %d", len(request.Messages))
	log.Printf("   └─ Tools Available: %d", len(request.Tools))
	
	// Check if LM Studio supports function calling
	if len(request.Tools) > 0 {
		log.Printf("🤖 LLM Provider: Tools detected - checking LM Studio compatibility")
		// LM Studio might not support OpenAI function calling format
		// We'll try to convert or handle this differently
		
		// Option 1: Try to send tools anyway and see if it works
		// Option 2: Convert to system prompt format
		// For now, let's try system prompt approach
		
		toolsPrompt := "You have access to the following tools. When you need to use a tool, respond with a JSON object containing tool_calls:\n"
		for _, tool := range request.Tools {
			toolsPrompt += fmt.Sprintf("- %s: %s\n", tool.Function.Name, tool.Function.Description)
		}
		toolsPrompt += "\nExample response when calling a tool:\n"
		toolsPrompt += `{"tool_calls": [{"function": {"name": "get_weather", "arguments": "{\"location\": \"Tokyo\"}"}}]}`
		
		// Log detailed tools schema for MCP analysis
		log.Printf("🔍 RAW_REQUEST: Tools Schema Details")
		for i, tool := range request.Tools {
			log.Printf("   └─ Tool %d: %s", i+1, tool.Function.Name)
			log.Printf("      └─ Description: %s", tool.Function.Description)
			if tool.Function.Parameters != nil {
				paramsJson, _ := json.Marshal(tool.Function.Parameters)
				log.Printf("      └─ Parameters: %s", string(paramsJson))
			}
		}
		log.Printf("🔍 RAW_REQUEST: Generated System Prompt")
		log.Printf("   └─ Prompt Length: %d characters", len(toolsPrompt))
		log.Printf("   └─ Full Prompt: %s", toolsPrompt)
		
		// Add tools instruction to system message
		if len(request.Messages) > 0 && request.Messages[0].Role == "system" {
			request.Messages[0].Content = request.Messages[0].Content + "\n\n" + toolsPrompt
		} else {
			// Insert system message at the beginning
			systemMsg := ChatMessage{
				Role:    "system",
				Content: toolsPrompt,
			}
			request.Messages = append([]ChatMessage{systemMsg}, request.Messages...)
		}
		log.Printf("🤖 LLM Provider: Added tools to system prompt (LM Studio compatibility mode)")
	}
	
	// Convert to LM Studio format (without tools field as LM Studio may not support it)
	lmReq := ChatRequest{
		Model:       request.Model,
		Messages:    request.Messages,
		Temperature: request.Temperature,
		MaxTokens:   request.MaxTokens,
	}
	
	jsonData, err := json.Marshal(lmReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}
	
	// Log complete request being sent to LLM
	log.Printf("🔍 RAW_REQUEST: Complete LLM Request")
	log.Printf("   └─ Request Size: %d bytes", len(jsonData))
	log.Printf("   └─ Full JSON Request: %s", string(jsonData))
	
	req, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/v1/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.apiKey)
	}
	
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	
	// Log complete raw response from LLM
	log.Printf("🔍 RAW_RESPONSE: Complete LLM Response")
	log.Printf("   └─ Response Status: %d", resp.StatusCode)
	log.Printf("   └─ Response Size: %d bytes", len(body))
	log.Printf("   └─ Raw Response Body: %s", string(body))
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}
	
	var lmResp ChatResponse
	if err := json.Unmarshal(body, &lmResp); err != nil {
		log.Printf("🔍 PARSING_DEBUG: Failed to unmarshal LLM response: %v", err)
		log.Printf("   └─ Raw response that failed: %s", string(body))
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}
	
	// Convert to unified format
	choices := make([]ChatCompletionChoice, len(lmResp.Choices))
	for i, choice := range lmResp.Choices {
		// Check if the response contains tool calls in JSON format
		toolCalls := []ToolCall{}
		content := choice.Message.Content
		
		// Try to detect JSON tool call format
		if strings.Contains(content, "tool_calls") {
			log.Printf("🤖 LLM Provider: Detected potential tool call in response")
			log.Printf("🔍 PARSING_DEBUG: Raw content from LLM: %s", content)
			
			// Clean up markdown code blocks and other formatting
			cleanContent := content
			// Remove markdown code block markers
			cleanContent = strings.ReplaceAll(cleanContent, "```json", "")
			cleanContent = strings.ReplaceAll(cleanContent, "```", "")
			// Remove any leading/trailing whitespace
			cleanContent = strings.TrimSpace(cleanContent)
			
			log.Printf("🤖 LLM Provider: Cleaned response content for parsing")
			log.Printf("   └─ Original length: %d characters", len(content))
			log.Printf("   └─ Cleaned length: %d characters", len(cleanContent))
			log.Printf("🔍 PARSING_DEBUG: Cleaned content: %s", cleanContent)
			
			// Character-by-character analysis for debugging
			log.Printf("🔍 PARSING_DEBUG: Character analysis of cleaned content:")
			for i, char := range []byte(cleanContent) {
				if i < 200 { // Only log first 200 characters to avoid spam
					log.Printf("   └─ Position %d: '%c' (ASCII: %d)", i, char, char)
				}
			}
			
			// Try to parse tool calls from the response
			var toolCallResponse struct {
				ToolCalls []struct {
					Function struct {
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
					} `json:"function"`
				} `json:"tool_calls"`
			}
			
			if err := json.Unmarshal([]byte(cleanContent), &toolCallResponse); err == nil {
				log.Printf("🤖 LLM Provider: Successfully parsed %d tool calls from LM Studio response", len(toolCallResponse.ToolCalls))
				for j, tc := range toolCallResponse.ToolCalls {
					toolCall := ToolCall{
						ID:   fmt.Sprintf("call_%d", j),
						Type: "function",
						Function: FunctionCall{
							Name:      tc.Function.Name,
							Arguments: tc.Function.Arguments,
						},
					}
					toolCalls = append(toolCalls, toolCall)
					log.Printf("   └─ Tool Call %d: %s", j+1, tc.Function.Name)
					log.Printf("      └─ Arguments: %s", tc.Function.Arguments)
				}
				// Clear content since we have tool calls
				content = ""
			} else {
				log.Printf("🤖 LLM Provider: Failed to parse tool calls from JSON response: %v", err)
				log.Printf("🔍 PARSING_DEBUG: JSON parsing error details:")
				log.Printf("   └─ Error: %v", err)
				log.Printf("   └─ Content that failed parsing: %s", cleanContent)
				
				// Try to identify the exact location of JSON syntax errors
				if jsonErr, ok := err.(*json.SyntaxError); ok {
					log.Printf("   └─ Syntax error at offset %d", jsonErr.Offset)
					if int(jsonErr.Offset) < len(cleanContent) && jsonErr.Offset > 0 {
						log.Printf("   └─ Character at error: '%c'", cleanContent[jsonErr.Offset-1])
					}
				}
			}
		}
		
		choices[i] = ChatCompletionChoice{
			Index:        choice.Index,
			Message:      &ChatMessage{Role: choice.Message.Role, Content: content},
			FinishReason: &choice.Reason,
			ToolCalls:    toolCalls,
		}
	}
	
	log.Printf("✅ LLM Provider: LM Studio response received successfully")
	log.Printf("   └─ Response ID: %s", lmResp.ID)
	log.Printf("   └─ Choices: %d", len(choices))
	log.Printf("   └─ Total Tokens Used: %d", lmResp.Usage.TotalTokens)
	if len(choices) > 0 {
		if len(choices[0].ToolCalls) > 0 {
			log.Printf("   └─ Tool Calls Detected: %d", len(choices[0].ToolCalls))
		} else {
			log.Printf("   └─ Completion Length: %d characters", len(choices[0].Message.Content))
		}
	}
	
	return &ChatCompletionResponse{
		ID:      lmResp.ID,
		Object:  lmResp.Object,
		Created: lmResp.Created,
		Model:   lmResp.Model,
		Choices: choices,
		Usage: ChatCompletionUsage{
			PromptTokens:     lmResp.Usage.PromptTokens,
			CompletionTokens: lmResp.Usage.CompletionTokens,
			TotalTokens:      lmResp.Usage.TotalTokens,
		},
	}, nil
}

func (p *LMStudioProvider) ChatStream(ctx context.Context, request ChatCompletionRequest) (<-chan ChatCompletionChunk, error) {
	// TODO: Implement streaming for LM Studio
	return nil, fmt.Errorf("streaming not implemented for LM Studio provider")
}

func (p *LMStudioProvider) GetModels() []ModelInfo {
	// Try to get models dynamically from LM Studio first
	if dynamicModels := p.getDynamicModels(); len(dynamicModels) > 0 {
		log.Printf("🔄 LM Studio Provider: Using dynamic models from LM Studio backend")
		log.Printf("   └─ Found %d models", len(dynamicModels))
		for _, model := range dynamicModels {
			log.Printf("   └─ Dynamic Model: %s", model.ID)
		}
		return dynamicModels
	}
	
	// Fallback to configured models if dynamic query fails
	log.Printf("⚠️ LM Studio Provider: Dynamic model query failed, using configured models")
	models := make([]ModelInfo, len(p.models))
	for i, model := range p.models {
		models[i] = ModelInfo{
			ID:       model,
			Object:   "model",
			Created:  time.Now().Unix(),
			OwnedBy:  "lmstudio",
			Provider: "lmstudio",
		}
	}
	log.Printf("   └─ Configured models: %d", len(models))
	return models
}

// getDynamicModels queries LM Studio's /v1/models endpoint to get actually loaded models
func (p *LMStudioProvider) getDynamicModels() []ModelInfo {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	req, err := http.NewRequestWithContext(ctx, "GET", p.baseURL+"/v1/models", nil)
	if err != nil {
		log.Printf("❌ LM Studio Provider: Failed to create models request: %v", err)
		return nil
	}
	
	if p.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.apiKey)
	}
	
	resp, err := p.client.Do(req)
	if err != nil {
		log.Printf("❌ LM Studio Provider: Failed to query models endpoint: %v", err)
		return nil
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		log.Printf("❌ LM Studio Provider: Models endpoint returned status %d", resp.StatusCode)
		return nil
	}
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("❌ LM Studio Provider: Failed to read models response: %v", err)
		return nil
	}
	
	// Parse LM Studio models response
	var modelsResponse struct {
		Data []struct {
			ID      string `json:"id"`
			Object  string `json:"object"`
			Created int64  `json:"created"`
			OwnedBy string `json:"owned_by"`
		} `json:"data"`
		Object string `json:"object"`
	}
	
	if err := json.Unmarshal(body, &modelsResponse); err != nil {
		log.Printf("❌ LM Studio Provider: Failed to parse models response: %v", err)
		log.Printf("❌ LM Studio Provider: Response body: %s", string(body))
		return nil
	}
	
	// Convert to our ModelInfo format
	models := make([]ModelInfo, 0, len(modelsResponse.Data))
	for _, model := range modelsResponse.Data {
		models = append(models, ModelInfo{
			ID:       model.ID,
			Object:   "model",
			Created:  model.Created,
			OwnedBy:  "lmstudio",
			Provider: "lmstudio",
		})
	}
	
	log.Printf("✅ LM Studio Provider: Successfully retrieved %d dynamic models", len(models))
	return models
}

func (p *LMStudioProvider) GetName() string {
	return p.config.Name
}

func (p *LMStudioProvider) IsHealthy(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", p.baseURL+"/v1/models", nil)
	if err != nil {
		return err
	}
	
	if p.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.apiKey)
	}
	
	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check failed with status %d", resp.StatusCode)
	}
	
	return nil
}

// OpenAIProvider implements LLMProvider interface for OpenAI API
type OpenAIProvider struct {
	config   LLMProviderConfig
	client   *http.Client
	baseURL  string
	apiKey   string
	models   []string
}

// NewOpenAIProvider creates a new OpenAI provider
func NewOpenAIProvider(config LLMProviderConfig) (*OpenAIProvider, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("OpenAI API key is required")
	}
	
	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	
	models := config.Models
	if len(models) == 0 {
		models = []string{"gpt-3.5-turbo", "gpt-4"}
	}
	
	return &OpenAIProvider{
		config:  config,
		client:  &http.Client{Timeout: 30 * time.Second},
		baseURL: baseURL,
		apiKey:  config.APIKey,
		models:  models,
	}, nil
}

func (p *OpenAIProvider) Chat(ctx context.Context, request ChatCompletionRequest) (*ChatCompletionResponse, error) {
	log.Printf("🤖 LLM Provider: OpenAI processing request")
	log.Printf("   └─ Provider Name: %s", p.config.Name)
	log.Printf("   └─ Base URL: %s", p.baseURL)
	log.Printf("   └─ Model: %s", request.Model)
	log.Printf("   └─ Temperature: %.2f", request.Temperature)
	log.Printf("   └─ Max Tokens: %d", request.MaxTokens)
	log.Printf("   └─ Messages Count: %d", len(request.Messages))
	log.Printf("   └─ Tools Available: %d", len(request.Tools))
	
	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}
	
	req, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}
	
	var response ChatCompletionResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}
	
	log.Printf("✅ LLM Provider: OpenAI response received successfully")
	log.Printf("   └─ Response ID: %s", response.ID)
	log.Printf("   └─ Choices: %d", len(response.Choices))
	if len(response.Choices) > 0 && response.Choices[0].Message != nil {
		log.Printf("   └─ Completion Length: %d characters", len(response.Choices[0].Message.Content))
	}
	log.Printf("   └─ Total Tokens Used: %d", response.Usage.TotalTokens)
	
	return &response, nil
}

func (p *OpenAIProvider) ChatStream(ctx context.Context, request ChatCompletionRequest) (<-chan ChatCompletionChunk, error) {
	// TODO: Implement streaming for OpenAI
	return nil, fmt.Errorf("streaming not implemented for OpenAI provider")
}

func (p *OpenAIProvider) GetModels() []ModelInfo {
	models := make([]ModelInfo, len(p.models))
	for i, model := range p.models {
		models[i] = ModelInfo{
			ID:       model,
			Object:   "model",
			Created:  time.Now().Unix(),
			OwnedBy:  "openai",
			Provider: "openai",
		}
	}
	return models
}

func (p *OpenAIProvider) GetName() string {
	return p.config.Name
}

func (p *OpenAIProvider) IsHealthy(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", p.baseURL+"/models", nil)
	if err != nil {
		return err
	}
	
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	
	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check failed with status %d", resp.StatusCode)
	}
	
	return nil
}

// OllamaProvider implements LLMProvider interface for Ollama
type OllamaProvider struct {
	config   LLMProviderConfig
	client   *http.Client
	baseURL  string
	models   []string
}

// NewOllamaProvider creates a new Ollama provider
func NewOllamaProvider(config LLMProviderConfig) (*OllamaProvider, error) {
	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}
	
	models := config.Models
	if len(models) == 0 {
		models = []string{"llama2"}
	}
	
	return &OllamaProvider{
		config:  config,
		client:  &http.Client{Timeout: 60 * time.Second}, // Ollama can be slower
		baseURL: baseURL,
		models:  models,
	}, nil
}

func (p *OllamaProvider) Chat(ctx context.Context, request ChatCompletionRequest) (*ChatCompletionResponse, error) {
	// Convert to Ollama format
	ollamaReq := map[string]interface{}{
		"model":    request.Model,
		"messages": request.Messages,
		"stream":   false,
	}
	
	if request.Temperature > 0 {
		ollamaReq["options"] = map[string]interface{}{
			"temperature": request.Temperature,
		}
	}
	
	jsonData, err := json.Marshal(ollamaReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}
	
	req, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/api/chat", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("Content-Type", "application/json")
	
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}
	
	var ollamaResp map[string]interface{}
	if err := json.Unmarshal(body, &ollamaResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}
	
	// Convert to unified format
	message, ok := ollamaResp["message"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid response format")
	}
	
	content, ok := message["content"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid message content")
	}
	
	finishReason := "stop"
	return &ChatCompletionResponse{
		ID:      fmt.Sprintf("ollama-%d", time.Now().Unix()),
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   request.Model,
		Choices: []ChatCompletionChoice{
			{
				Index: 0,
				Message: &ChatMessage{
					Role:    "assistant",
					Content: content,
				},
				FinishReason: &finishReason,
			},
		},
		Usage: ChatCompletionUsage{
			PromptTokens:     EstimateTokens(fmt.Sprintf("%v", request.Messages)),
			CompletionTokens: EstimateTokens(content),
			TotalTokens:      EstimateTokens(fmt.Sprintf("%v", request.Messages)) + EstimateTokens(content),
		},
	}, nil
}

func (p *OllamaProvider) ChatStream(ctx context.Context, request ChatCompletionRequest) (<-chan ChatCompletionChunk, error) {
	// TODO: Implement streaming for Ollama
	return nil, fmt.Errorf("streaming not implemented for Ollama provider")
}

func (p *OllamaProvider) GetModels() []ModelInfo {
	models := make([]ModelInfo, len(p.models))
	for i, model := range p.models {
		models[i] = ModelInfo{
			ID:       model,
			Object:   "model",
			Created:  time.Now().Unix(),
			OwnedBy:  "ollama",
			Provider: "ollama",
		}
	}
	return models
}

func (p *OllamaProvider) GetName() string {
	return p.config.Name
}

func (p *OllamaProvider) IsHealthy(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", p.baseURL+"/api/tags", nil)
	if err != nil {
		return err
	}
	
	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check failed with status %d", resp.StatusCode)
	}
	
	return nil
}

