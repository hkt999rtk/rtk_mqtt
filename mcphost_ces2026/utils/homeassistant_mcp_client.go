package utils

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
)

// HomeAssistantMCPClient implements MCPClient for Home Assistant MCP server using SSE
type HomeAssistantMCPClient struct {
	name              string
	baseURL           string
	sseEndpoint       string
	messagesEndpoint  string
	authToken         string
	client            *http.Client
	sessionID         string
	eventStream       io.ReadCloser
	mu                sync.RWMutex
	tools             []mcp.Tool
	lastToolsUpdate   time.Time
	connected         bool
	messageID         int
	responseChan      map[int]chan *mcp.JSONRPCResponse
	responseMu        sync.RWMutex
}

// NewHomeAssistantMCPClient creates a new Home Assistant MCP client
func NewHomeAssistantMCPClient(config map[string]interface{}) (*HomeAssistantMCPClient, error) {
	name, ok := config["name"].(string)
	if !ok {
		return nil, fmt.Errorf("name is required")
	}
	
	baseURL, ok := config["base_url"].(string)
	if !ok {
		return nil, fmt.Errorf("base_url is required")
	}
	
	sseEndpoint, ok := config["sse_endpoint"].(string)
	if !ok {
		sseEndpoint = "/mcp_server/sse"
	}
	
	messagesEndpoint, ok := config["messages_endpoint"].(string)
	if !ok {
		messagesEndpoint = "/mcp_server/messages"
	}
	
	// Get auth token from environment
	authTokenEnv, ok := config["auth_token_env"].(string)
	if !ok {
		authTokenEnv = "HA_TOKEN"
	}
	
	authToken := os.Getenv(authTokenEnv)
	if authToken == "" {
		log.Printf("⚠️ Home Assistant MCP: No auth token found in environment variable %s", authTokenEnv)
	}
	
	client := &HomeAssistantMCPClient{
		name:              name,
		baseURL:           baseURL,
		sseEndpoint:       sseEndpoint,
		messagesEndpoint:  messagesEndpoint,
		authToken:         authToken,
		client:            &http.Client{Timeout: 30 * time.Second},
		responseChan:      make(map[int]chan *mcp.JSONRPCResponse),
	}
	
	return client, nil
}

func (c *HomeAssistantMCPClient) GetName() string {
	return c.name
}

// connectSSE establishes SSE connection to Home Assistant MCP server
func (c *HomeAssistantMCPClient) connectSSE(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if c.connected && c.eventStream != nil {
		return nil
	}
	
	if c.authToken == "" {
		return fmt.Errorf("authentication token is required")
	}
	
	url := c.baseURL + c.sseEndpoint
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create SSE request: %w", err)
	}
	
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Authorization", "Bearer "+c.authToken)
	
	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect to SSE: %w", err)
	}
	
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return fmt.Errorf("SSE connection failed with status %d", resp.StatusCode)
	}
	
	c.eventStream = resp.Body
	c.connected = true
	
	// Start processing events
	go c.processEvents()
	
	log.Printf("✅ Home Assistant MCP: Connected to %s", url)
	return nil
}

// processEvents processes incoming SSE events
func (c *HomeAssistantMCPClient) processEvents() {
	defer func() {
		c.mu.Lock()
		c.connected = false
		if c.eventStream != nil {
			c.eventStream.Close()
		}
		c.mu.Unlock()
	}()
	
	scanner := bufio.NewScanner(c.eventStream)
	var eventData strings.Builder
	
	for scanner.Scan() {
		line := scanner.Text()
		
		if strings.HasPrefix(line, "data: ") {
			eventData.WriteString(line[6:])
		} else if line == "" {
			// End of event
			data := eventData.String()
			eventData.Reset()
			
			if data != "" {
				c.handleEvent(data)
			}
		} else if strings.HasPrefix(line, "id: ") {
			// Extract session ID if available
			if c.sessionID == "" {
				c.sessionID = line[4:]
				log.Printf("📍 Home Assistant MCP: Session ID: %s", c.sessionID)
			}
		}
	}
	
	if err := scanner.Err(); err != nil {
		log.Printf("❌ Home Assistant MCP: SSE stream error: %v", err)
	}
}

// handleEvent processes an individual SSE event
func (c *HomeAssistantMCPClient) handleEvent(data string) {
	if data == "" {
		return
	}
	
	var response mcp.JSONRPCResponse
	if err := json.Unmarshal([]byte(data), &response); err != nil {
		// Try to parse as error response
		var errResponse mcp.JSONRPCError
		if err2 := json.Unmarshal([]byte(data), &errResponse); err2 != nil {
			log.Printf("⚠️ Home Assistant MCP: Failed to parse message: %v", err)
			return
		}
		// Handle error response (convert to response format)
		response = mcp.JSONRPCResponse{
			JSONRPC: errResponse.JSONRPC,
			ID:      errResponse.ID,
			Result:  errResponse.Error,
		}
	}
	
	// Handle response messages
	// Extract the ID value
	if idValue, ok := response.ID.Value().(int); ok {
		c.responseMu.RLock()
		if ch, exists := c.responseChan[idValue]; exists {
			select {
			case ch <- &response:
			case <-time.After(1 * time.Second):
				log.Printf("⚠️ Home Assistant MCP: Response channel timeout for message %d", idValue)
			}
		}
		c.responseMu.RUnlock()
	}
}

// sendMessage sends a JSON-RPC message to Home Assistant MCP server
func (c *HomeAssistantMCPClient) sendMessage(ctx context.Context, method string, params interface{}) (*mcp.JSONRPCResponse, error) {
	if !c.connected {
		if err := c.connectSSE(ctx); err != nil {
			return nil, fmt.Errorf("failed to connect: %w", err)
		}
	}
	
	c.messageID++
	msgID := c.messageID
	
	requestID := mcp.NewRequestId(msgID)
	message := mcp.JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      requestID,
		Request: mcp.Request{
			Method: method,
		},
		Params:  params,
	}
	
	// Create response channel
	respChan := make(chan *mcp.JSONRPCResponse, 1)
	c.responseMu.Lock()
	c.responseChan[msgID] = respChan
	c.responseMu.Unlock()
	
	// Clean up response channel
	defer func() {
		c.responseMu.Lock()
		delete(c.responseChan, msgID)
		c.responseMu.Unlock()
		close(respChan)
	}()
	
	// Send message
	reqData, err := json.Marshal(message)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal message: %w", err)
	}
	
	url := c.baseURL + c.messagesEndpoint + "?sessionId=" + c.sessionID
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(reqData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.authToken)
	
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send message: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("message send failed with status %d", resp.StatusCode)
	}
	
	// Wait for response
	select {
	case response := <-respChan:
		return response, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(30 * time.Second):
		return nil, fmt.Errorf("timeout waiting for response")
	}
}

func (c *HomeAssistantMCPClient) GetTools(ctx context.Context) ([]mcp.Tool, error) {
	c.mu.RLock()
	if time.Since(c.lastToolsUpdate) < 5*time.Minute && len(c.tools) > 0 {
		defer c.mu.RUnlock()
		return c.tools, nil
	}
	c.mu.RUnlock()
	
	response, err := c.sendMessage(ctx, "tools/list", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get tools: %w", err)
	}
	
	// Check if response contains an error
	if errorResult, ok := response.Result.(map[string]interface{}); ok {
		if errorMsg, hasError := errorResult["error"]; hasError {
			return nil, fmt.Errorf("tools/list error: %v", errorMsg)
		}
	}
	
	var result struct {
		Tools []mcp.Tool `json:"tools"`
	}
	
	resultBytes, err := json.Marshal(response.Result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal tools response: %w", err)
	}
	
	if err := json.Unmarshal(resultBytes, &result); err != nil {
		return nil, fmt.Errorf("failed to parse tools response: %w", err)
	}
	
	c.mu.Lock()
	c.tools = result.Tools
	c.lastToolsUpdate = time.Now()
	c.mu.Unlock()
	
	log.Printf("🔧 Home Assistant MCP: Retrieved %d tools", len(result.Tools))
	return result.Tools, nil
}

func (c *HomeAssistantMCPClient) CallTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	response, err := c.sendMessage(ctx, "tools/call", request)
	if err != nil {
		return nil, fmt.Errorf("failed to call tool: %w", err)
	}
	
	// Check if response contains an error
	if errorResult, ok := response.Result.(map[string]interface{}); ok {
		if errorMsg, hasError := errorResult["error"]; hasError {
			return nil, fmt.Errorf("tools/call error: %v", errorMsg)
		}
	}
	
	var result mcp.CallToolResult
	resultBytes, err := json.Marshal(response.Result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal tool call response: %w", err)
	}
	
	if err := json.Unmarshal(resultBytes, &result); err != nil {
		return nil, fmt.Errorf("failed to parse tool call response: %w", err)
	}
	
	return &result, nil
}

func (c *HomeAssistantMCPClient) GetResources(ctx context.Context) ([]mcp.Resource, error) {
	response, err := c.sendMessage(ctx, "resources/list", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get resources: %w", err)
	}
	
	// Check if response contains an error
	if errorResult, ok := response.Result.(map[string]interface{}); ok {
		if errorMsg, hasError := errorResult["error"]; hasError {
			return nil, fmt.Errorf("resources/list error: %v", errorMsg)
		}
	}
	
	var result struct {
		Resources []mcp.Resource `json:"resources"`
	}
	
	resultBytes, err := json.Marshal(response.Result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal resources response: %w", err)
	}
	
	if err := json.Unmarshal(resultBytes, &result); err != nil {
		return nil, fmt.Errorf("failed to parse resources response: %w", err)
	}
	
	return result.Resources, nil
}

func (c *HomeAssistantMCPClient) GetPrompts(ctx context.Context) ([]mcp.Prompt, error) {
	response, err := c.sendMessage(ctx, "prompts/list", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get prompts: %w", err)
	}
	
	// Check if response contains an error
	if errorResult, ok := response.Result.(map[string]interface{}); ok {
		if errorMsg, hasError := errorResult["error"]; hasError {
			return nil, fmt.Errorf("prompts/list error: %v", errorMsg)
		}
	}
	
	var result struct {
		Prompts []mcp.Prompt `json:"prompts"`
	}
	
	resultBytes, err := json.Marshal(response.Result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal prompts response: %w", err)
	}
	
	if err := json.Unmarshal(resultBytes, &result); err != nil {
		return nil, fmt.Errorf("failed to parse prompts response: %w", err)
	}
	
	return result.Prompts, nil
}

func (c *HomeAssistantMCPClient) IsHealthy(ctx context.Context) error {
	if c.authToken == "" {
		return fmt.Errorf("no authentication token available")
	}
	
	// Test SSE connection
	if err := c.connectSSE(ctx); err != nil {
		return fmt.Errorf("SSE connection failed: %w", err)
	}
	
	// Test basic JSON-RPC call
	_, err := c.sendMessage(ctx, "tools/list", nil)
	if err != nil {
		return fmt.Errorf("JSON-RPC test failed: %w", err)
	}
	
	return nil
}

func (c *HomeAssistantMCPClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	c.connected = false
	
	if c.eventStream != nil {
		c.eventStream.Close()
		c.eventStream = nil
	}
	
	// Close all response channels
	c.responseMu.Lock()
	for id, ch := range c.responseChan {
		close(ch)
		delete(c.responseChan, id)
	}
	c.responseMu.Unlock()
	
	log.Printf("🔌 Home Assistant MCP: Connection closed")
	return nil
}

// NewHomeAssistantMCPClientFromConfig creates a Home Assistant MCP client from MCPServerConfig
func NewHomeAssistantMCPClientFromConfig(config MCPServerConfig) (*HomeAssistantMCPClient, error) {
	clientConfig := map[string]interface{}{
		"name":                config.Name,
		"base_url":           config.URL,
		"sse_endpoint":       "/mcp_server/sse",
		"messages_endpoint":  "/mcp_server/messages", 
		"auth_token_env":     "HA_TOKEN",
	}
	
	return NewHomeAssistantMCPClient(clientConfig)
}