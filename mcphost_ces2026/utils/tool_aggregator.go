package utils

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// ToolAggregator manages local and remote tools
type ToolAggregator struct {
	localServer  *server.MCPServer
	mcpRegistry  *MCPClientRegistry
}

// NewToolAggregator creates a new tool aggregator
func NewToolAggregator() *ToolAggregator {
	return &ToolAggregator{
		mcpRegistry: GetMCPRegistry(),
	}
}

// AddRemoteTools adds tools from external MCP servers to the local MCP server
func (ta *ToolAggregator) AddRemoteTools(s *server.MCPServer) error {
	if ta.mcpRegistry == nil {
		log.Printf("📍 Tool Aggregator: No MCP registry available, using local tools only")
		return nil
	}
	
	ctx := context.Background()
	remoteTools, err := ta.mcpRegistry.GetAllTools(ctx)
	if err != nil {
		log.Printf("⚠️ Tool Aggregator: Failed to get remote tools: %v", err)
		return err
	}
	
	toolCount := 0
	for serverName, tools := range remoteTools {
		log.Printf("🔄 MCP Integration: Processing tools from server '%s'", serverName)
		
		for _, tool := range tools {
			// Create a namespaced tool name to avoid conflicts
			namespacedName := fmt.Sprintf("%s_%s", serverName, tool.Name)
			
			log.Printf("🔄 MCP Integration: Registering remote tool")
			log.Printf("   └─ Server: %s", serverName)
			log.Printf("   └─ Original Name: %s", tool.Name)
			log.Printf("   └─ Namespaced Name: %s", namespacedName)
			log.Printf("   └─ Description: %s", tool.Description)
			
			// Create a new tool with the namespaced name
			namespacedTool := mcp.NewTool(namespacedName,
				mcp.WithDescription(fmt.Sprintf("[%s] %s", serverName, tool.Description)),
			)
			
			// Copy input schema from original tool
			if tool.InputSchema.Type != "" {
				namespacedTool.InputSchema = tool.InputSchema
				log.Printf("   └─ Input Schema: %s", tool.InputSchema.Type)
			}
			
			// Create tool handler that routes to the appropriate MCP server
			originalToolName := tool.Name
			s.AddTool(namespacedTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				startTime := time.Now()
				log.Printf("🔄 MCP Server Call: Starting remote tool execution")
				log.Printf("   └─ Tool: %s -> %s", namespacedName, originalToolName)
				log.Printf("   └─ Target Server: %s", serverName)
				log.Printf("   └─ Arguments: %+v", request.Params.Arguments)
				
				// Create a new request with the original tool name
				originalRequest := mcp.CallToolRequest{
					Params: mcp.CallToolParams{
						Name:      originalToolName, // Use original name for remote server
						Arguments: request.Params.Arguments,
					},
				}
				
				// Route to the appropriate MCP server
				result, err := ta.mcpRegistry.CallToolOnServer(ctx, serverName, originalRequest)
				duration := time.Since(startTime)
				
				if err != nil {
					log.Printf("❌ MCP Server Call: Remote tool execution failed")
					log.Printf("   └─ Tool: %s", originalToolName)
					log.Printf("   └─ Server: %s", serverName)
					log.Printf("   └─ Duration: %v", duration)
					log.Printf("   └─ Error: %v", err)
					return mcp.NewToolResultError(fmt.Sprintf("Remote tool call failed: %v", err)), nil
				}
				
				// Log successful response details
				log.Printf("✅ MCP Server Call: Remote tool execution successful")
				log.Printf("   └─ Tool: %s", originalToolName)
				log.Printf("   └─ Server: %s", serverName)
				log.Printf("   └─ Duration: %v", duration)
				if result != nil {
					log.Printf("   └─ Result Type: %T", result)
					if len(result.Content) > 0 {
						contentLen := 0
						for _, content := range result.Content {
							if textContent, ok := content.(mcp.TextContent); ok {
								contentLen += len(textContent.Text)
							}
						}
						log.Printf("   └─ Response Length: %d characters", contentLen)
					}
				}
				
				return result, nil
			})
			
			toolCount++
		}
		
		log.Printf("🔗 Tool Aggregator: Added %d tools from server '%s'", len(tools), serverName)
	}
	
	if toolCount > 0 {
		log.Printf("✅ Tool Aggregator: Successfully added %d remote tools from %d servers", 
			toolCount, len(remoteTools))
	} else {
		log.Printf("📍 Tool Aggregator: No remote tools available")
	}
	
	return nil
}

// GetToolInfo returns information about all available tools (local + remote)
func (ta *ToolAggregator) GetToolInfo(ctx context.Context) (map[string][]ToolInfo, error) {
	result := make(map[string][]ToolInfo)
	
	// Add local tools info
	result["local"] = []ToolInfo{
		{Name: "llm_chat", Description: "Chat with configured LLM provider", Type: "local"},
		{Name: "get_weather", Description: "Get weather information", Type: "local"},
		{Name: "get_current_time", Description: "Get current time", Type: "local"},
	}
	
	// Add remote tools info
	if ta.mcpRegistry != nil {
		remoteTools, err := ta.mcpRegistry.GetAllTools(ctx)
		if err == nil {
			for serverName, tools := range remoteTools {
				serverTools := make([]ToolInfo, len(tools))
				for i, tool := range tools {
					serverTools[i] = ToolInfo{
						Name:        fmt.Sprintf("%s_%s", serverName, tool.Name),
						Description: tool.Description,
						Type:        "remote",
						Server:      serverName,
						OriginalName: tool.Name,
					}
				}
				result[serverName] = serverTools
			}
		}
	}
	
	return result, nil
}

// ToolInfo represents information about a tool
type ToolInfo struct {
	Name         string `json:"name"`
	Description  string `json:"description"`
	Type         string `json:"type"`         // "local" or "remote"
	Server       string `json:"server,omitempty"`
	OriginalName string `json:"original_name,omitempty"`
}

// RouteToolCall routes a tool call to the appropriate handler
func (ta *ToolAggregator) RouteToolCall(ctx context.Context, toolName string, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Check if it's a remote tool (contains underscore with server name)
	parts := strings.SplitN(toolName, "_", 2)
	if len(parts) == 2 && ta.mcpRegistry != nil {
		serverName := parts[0]
		originalToolName := parts[1]
		
		// Check if the server exists
		if _, err := ta.mcpRegistry.GetClient(serverName); err == nil {
			log.Printf("🔧 Tool Router: Routing '%s' to server '%s' (original: '%s')", 
				toolName, serverName, originalToolName)
			
			// Create request with original tool name
			originalRequest := mcp.CallToolRequest{
				Params: mcp.CallToolParams{
					Name:      originalToolName,
					Arguments: request.Params.Arguments,
				},
			}
			
			return ta.mcpRegistry.CallToolOnServer(ctx, serverName, originalRequest)
		}
	}
	
	// If not a remote tool or server not found, it should be handled by local tools
	return nil, fmt.Errorf("tool '%s' not found in aggregator routing", toolName)
}

// Global tool aggregator instance
var toolAggregator *ToolAggregator

// InitializeToolAggregator initializes the global tool aggregator
func InitializeToolAggregator() {
	toolAggregator = NewToolAggregator()
	log.Printf("🔧 Tool Aggregator: Initialized")
}

// GetToolAggregator returns the global tool aggregator
func GetToolAggregator() *ToolAggregator {
	return toolAggregator
}