package utils

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
)

// MCPClient represents a connection to an external MCP server
type MCPClient interface {
	// GetTools returns the list of tools available on this MCP server
	GetTools(ctx context.Context) ([]mcp.Tool, error)
	
	// CallTool calls a tool on the MCP server
	CallTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error)
	
	// GetResources returns the list of resources available on this MCP server
	GetResources(ctx context.Context) ([]mcp.Resource, error)
	
	// GetPrompts returns the list of prompts available on this MCP server
	GetPrompts(ctx context.Context) ([]mcp.Prompt, error)
	
	// IsHealthy checks if the MCP server is healthy and reachable
	IsHealthy(ctx context.Context) error
	
	// GetName returns the name of this MCP server
	GetName() string
	
	// Close closes the connection to the MCP server
	Close() error
}

// HTTPMCPClient implements MCPClient for HTTP-based MCP servers
type HTTPMCPClient struct {
	name    string
	baseURL string
	client  *http.Client
	mu      sync.RWMutex
	tools   []mcp.Tool
	lastToolsUpdate time.Time
}

// NewHTTPMCPClient creates a new HTTP-based MCP client
func NewHTTPMCPClient(config MCPServerConfig) (*HTTPMCPClient, error) {
	if config.Type != "http" {
		return nil, fmt.Errorf("invalid type for HTTP client: %s", config.Type)
	}
	
	if config.URL == "" {
		return nil, fmt.Errorf("URL is required for HTTP MCP client")
	}
	
	// Validate URL
	if _, err := url.Parse(config.URL); err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}
	
	return &HTTPMCPClient{
		name:    config.Name,
		baseURL: config.URL,
		client:  &http.Client{Timeout: 30 * time.Second},
	}, nil
}

func (c *HTTPMCPClient) GetName() string {
	return c.name
}

func (c *HTTPMCPClient) GetTools(ctx context.Context) ([]mcp.Tool, error) {
	c.mu.RLock()
	// Cache tools for 5 minutes
	if time.Since(c.lastToolsUpdate) < 5*time.Minute && len(c.tools) > 0 {
		defer c.mu.RUnlock()
		return c.tools, nil
	}
	c.mu.RUnlock()
	
	// Need to fetch tools
	c.mu.Lock()
	defer c.mu.Unlock()
	
	// Double-check after acquiring write lock
	if time.Since(c.lastToolsUpdate) < 5*time.Minute && len(c.tools) > 0 {
		return c.tools, nil
	}
	
	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/tools/list", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("Content-Type", "application/json")
	
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned status %d", resp.StatusCode)
	}
	
	var result struct {
		Tools []mcp.Tool `json:"tools"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	
	c.tools = result.Tools
	c.lastToolsUpdate = time.Now()
	
	return c.tools, nil
}

func (c *HTTPMCPClient) CallTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	reqData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}
	
	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/tools/call", 
		bytes.NewBuffer(reqData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	httpReq.Header.Set("Content-Type", "application/json")
	
	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned status %d", resp.StatusCode)
	}
	
	var result mcp.CallToolResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	
	return &result, nil
}

func (c *HTTPMCPClient) GetResources(ctx context.Context) ([]mcp.Resource, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/resources/list", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("Content-Type", "application/json")
	
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned status %d", resp.StatusCode)
	}
	
	var result struct {
		Resources []mcp.Resource `json:"resources"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	
	return result.Resources, nil
}

func (c *HTTPMCPClient) GetPrompts(ctx context.Context) ([]mcp.Prompt, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/prompts/list", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("Content-Type", "application/json")
	
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned status %d", resp.StatusCode)
	}
	
	var result struct {
		Prompts []mcp.Prompt `json:"prompts"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	
	return result.Prompts, nil
}

func (c *HTTPMCPClient) IsHealthy(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/health", nil)
	if err != nil {
		return err
	}
	
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check failed with status %d", resp.StatusCode)
	}
	
	return nil
}

func (c *HTTPMCPClient) Close() error {
	// Nothing to close for HTTP client
	return nil
}

// MCPClientRegistry manages connections to multiple external MCP servers
type MCPClientRegistry struct {
	clients map[string]MCPClient
	mu      sync.RWMutex
}

// NewMCPClientRegistry creates a new MCP client registry
func NewMCPClientRegistry() *MCPClientRegistry {
	return &MCPClientRegistry{
		clients: make(map[string]MCPClient),
	}
}

// RegisterClient registers a new MCP client
func (r *MCPClientRegistry) RegisterClient(name string, client MCPClient) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.clients[name] = client
}

// GetClient returns a client by name
func (r *MCPClientRegistry) GetClient(name string) (MCPClient, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	client, exists := r.clients[name]
	if !exists {
		return nil, fmt.Errorf("MCP client '%s' not found", name)
	}
	
	return client, nil
}

// ListClients returns all registered client names
func (r *MCPClientRegistry) ListClients() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	names := make([]string, 0, len(r.clients))
	for name := range r.clients {
		names = append(names, name)
	}
	return names
}

// GetAllTools returns all tools from all registered MCP clients
func (r *MCPClientRegistry) GetAllTools(ctx context.Context) (map[string][]mcp.Tool, error) {
	r.mu.RLock()
	clients := make(map[string]MCPClient, len(r.clients))
	for name, client := range r.clients {
		clients[name] = client
	}
	r.mu.RUnlock()
	
	result := make(map[string][]mcp.Tool)
	errors := make([]error, 0)
	
	for name, client := range clients {
		tools, err := client.GetTools(ctx)
		if err != nil {
			log.Printf("⚠️ Failed to get tools from MCP server '%s': %v", name, err)
			errors = append(errors, fmt.Errorf("server %s: %w", name, err))
			continue
		}
		result[name] = tools
	}
	
	if len(result) == 0 && len(errors) > 0 {
		return nil, fmt.Errorf("failed to get tools from any server: %v", errors)
	}
	
	return result, nil
}

// CallToolOnServer calls a tool on a specific MCP server
func (r *MCPClientRegistry) CallToolOnServer(ctx context.Context, serverName string, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := r.GetClient(serverName)
	if err != nil {
		return nil, err
	}
	
	return client.CallTool(ctx, request)
}

// Close closes all registered clients
func (r *MCPClientRegistry) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	var errors []error
	for name, client := range r.clients {
		if err := client.Close(); err != nil {
			errors = append(errors, fmt.Errorf("failed to close client %s: %w", name, err))
		}
	}
	
	if len(errors) > 0 {
		return fmt.Errorf("errors while closing clients: %v", errors)
	}
	
	return nil
}

// Global registry instance
var mcpRegistry *MCPClientRegistry

// InitializeMCPClients initializes all MCP clients from configuration
func InitializeMCPClients(configs []MCPServerConfig) error {
	mcpRegistry = NewMCPClientRegistry()
	
	for _, config := range configs {
		if !config.Enabled {
			continue
		}
		
		if config.Type == "local" {
			// Local server doesn't need a client - it's handled by the main MCP server
			log.Printf("📍 MCP Registry: Local server '%s' registered (handled internally)", config.Name)
			continue
		}
		
		var client MCPClient
		var err error
		
		switch config.Type {
		case "http":
			client, err = NewHTTPMCPClient(config)
		case "homeassistant":
			client, err = NewHomeAssistantMCPClientFromConfig(config)
		default:
			return fmt.Errorf("unsupported MCP server type: %s", config.Type)
		}
		
		if err != nil {
			return fmt.Errorf("failed to create MCP client for server %s: %w", config.Name, err)
		}
		
		// Test connectivity
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		if err := client.IsHealthy(ctx); err != nil {
			log.Printf("⚠️ MCP Registry: Server '%s' is not healthy: %v", config.Name, err)
		} else {
			log.Printf("✅ MCP Registry: Server '%s' is healthy", config.Name)
		}
		cancel()
		
		mcpRegistry.RegisterClient(config.Name, client)
		log.Printf("🔗 MCP Registry: Registered client for server '%s' (%s)", config.Name, config.Type)
	}
	
	return nil
}

// GetMCPRegistry returns the global MCP client registry
func GetMCPRegistry() *MCPClientRegistry {
	return mcpRegistry
}