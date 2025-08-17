package utils

import (
	"context"
	"fmt"
)

// LLMProvider defines the interface for different LLM backends
type LLMProvider interface {
	// Chat sends a chat completion request and returns the response
	Chat(ctx context.Context, request ChatCompletionRequest) (*ChatCompletionResponse, error)
	
	// ChatStream sends a streaming chat completion request
	ChatStream(ctx context.Context, request ChatCompletionRequest) (<-chan ChatCompletionChunk, error)
	
	// GetModels returns the list of available models for this provider
	GetModels() []ModelInfo
	
	// GetName returns the name of this provider
	GetName() string
	
	// IsHealthy checks if the provider is healthy and reachable
	IsHealthy(ctx context.Context) error
}

// ChatCompletionRequest represents a unified chat completion request
type ChatCompletionRequest struct {
	Model       string           `json:"model"`
	Messages    []ChatMessage    `json:"messages"`
	Temperature float64          `json:"temperature,omitempty"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
	Stream      bool            `json:"stream,omitempty"`
	Tools       []ToolDefinition `json:"tools,omitempty"`
	ToolChoice  interface{}     `json:"tool_choice,omitempty"`
}

// ChatCompletionResponse represents a unified chat completion response
type ChatCompletionResponse struct {
	ID      string                   `json:"id"`
	Object  string                  `json:"object"`
	Created int64                   `json:"created"`
	Model   string                  `json:"model"`
	Choices []ChatCompletionChoice  `json:"choices"`
	Usage   ChatCompletionUsage     `json:"usage"`
}

// ChatCompletionChoice represents a choice in the completion response
type ChatCompletionChoice struct {
	Index        int              `json:"index"`
	Message      *ChatMessage     `json:"message,omitempty"`
	Delta        *ChatMessage     `json:"delta,omitempty"`
	FinishReason *string          `json:"finish_reason"`
	ToolCalls    []ToolCall       `json:"tool_calls,omitempty"`
}

// ChatCompletionChunk represents a streaming chunk
type ChatCompletionChunk struct {
	ID      string                   `json:"id"`
	Object  string                  `json:"object"`
	Created int64                   `json:"created"`
	Model   string                  `json:"model"`
	Choices []ChatCompletionChoice  `json:"choices"`
}

// ChatCompletionUsage represents token usage information
type ChatCompletionUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ToolDefinition represents a tool/function that can be called
type ToolDefinition struct {
	Type     string   `json:"type"`
	Function Function `json:"function"`
}

// Function represents a function definition
type Function struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// ToolCall represents a tool call in the response
type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function FunctionCall `json:"function"`
}

// FunctionCall represents a function call
type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// ModelInfo represents information about a model
type ModelInfo struct {
	ID       string `json:"id"`
	Object   string `json:"object"`
	Created  int64  `json:"created"`
	OwnedBy  string `json:"owned_by"`
	Provider string `json:"provider"`
}

// LLMProviderConfig represents configuration for LLM providers
type LLMProviderConfig struct {
	Name     string            `json:"name"`
	Type     string            `json:"type"`     // openai, ollama, lmstudio
	BaseURL  string            `json:"base_url"`
	APIKey   string            `json:"api_key,omitempty"`
	Models   []string          `json:"models,omitempty"`
	Settings map[string]interface{} `json:"settings,omitempty"`
	Enabled  bool              `json:"enabled"`
}

// LLMProviderRegistry manages multiple LLM providers
type LLMProviderRegistry struct {
	providers     map[string]LLMProvider
	defaultProvider string
}

// NewLLMProviderRegistry creates a new provider registry
func NewLLMProviderRegistry() *LLMProviderRegistry {
	return &LLMProviderRegistry{
		providers: make(map[string]LLMProvider),
	}
}

// RegisterProvider registers a new LLM provider
func (r *LLMProviderRegistry) RegisterProvider(name string, provider LLMProvider) {
	r.providers[name] = provider
	if r.defaultProvider == "" {
		r.defaultProvider = name
	}
}

// GetProvider returns a provider by name
func (r *LLMProviderRegistry) GetProvider(name string) (LLMProvider, error) {
	if name == "" {
		name = r.defaultProvider
	}
	
	provider, exists := r.providers[name]
	if !exists {
		return nil, fmt.Errorf("provider '%s' not found", name)
	}
	
	return provider, nil
}

// GetDefaultProvider returns the default provider
func (r *LLMProviderRegistry) GetDefaultProvider() (LLMProvider, error) {
	return r.GetProvider(r.defaultProvider)
}

// SetDefaultProvider sets the default provider
func (r *LLMProviderRegistry) SetDefaultProvider(name string) error {
	if _, exists := r.providers[name]; !exists {
		return fmt.Errorf("provider '%s' not found", name)
	}
	r.defaultProvider = name
	return nil
}

// ListProviders returns all registered provider names
func (r *LLMProviderRegistry) ListProviders() []string {
	names := make([]string, 0, len(r.providers))
	for name := range r.providers {
		names = append(names, name)
	}
	return names
}

// GetAllModels returns all models from all providers
func (r *LLMProviderRegistry) GetAllModels() []ModelInfo {
	var allModels []ModelInfo
	for _, provider := range r.providers {
		models := provider.GetModels()
		allModels = append(allModels, models...)
	}
	return allModels
}

// GetProviderCount returns the number of registered providers
func (r *LLMProviderRegistry) GetProviderCount() int {
	return len(r.providers)
}

// Global registry instance
var llmRegistry *LLMProviderRegistry

// InitializeLLMProviders initializes all LLM providers from configuration
func InitializeLLMProviders(configs []LLMProviderConfig) error {
	llmRegistry = NewLLMProviderRegistry()
	
	for _, config := range configs {
		if !config.Enabled {
			continue
		}
		
		var provider LLMProvider
		var err error
		
		switch config.Type {
		case "lmstudio":
			provider, err = NewLMStudioProvider(config)
		case "openai":
			provider, err = NewOpenAIProvider(config)
		case "ollama":
			provider, err = NewOllamaProvider(config)
		default:
			return fmt.Errorf("unsupported provider type: %s", config.Type)
		}
		
		if err != nil {
			return fmt.Errorf("failed to create provider %s: %w", config.Name, err)
		}
		
		llmRegistry.RegisterProvider(config.Name, provider)
	}
	
	return nil
}

// GetLLMRegistry returns the global LLM registry
func GetLLMRegistry() *LLMProviderRegistry {
	return llmRegistry
}