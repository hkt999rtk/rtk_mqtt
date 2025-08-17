package main

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"

	"github.com/mark3labs/mcp-go/server"
)

// Global MCP server instance for local tool execution
var localMCPServer *server.MCPServer

// createMCPServer creates a complete MCP server with all tools, resources, and prompts
func createMCPServer() *server.MCPServer {
	s := server.NewMCPServer(
		config.Server.Name,
		config.Server.Version,
		server.WithInstructions(config.Server.Description),
	)

	// TODO: Add local tools
	// AddWeatherTools(s)
	// AddTimeTools(s)
	// AddTestTools(s)
	// AddLLMTools(s)
	
	// TODO: Add resources
	// AddExampleResources(s)
	
	// TODO: Add prompts
	// AddExamplePrompts(s)

	// Store globally for tool execution access
	localMCPServer = s
	
	log.Printf("🛠️ MCP Server: Created with local tools, resources, and prompts")
	return s
}

// startMCPHTTPServer sets up HTTP endpoints for MCP protocol
func startMCPHTTPServer(mux *http.ServeMux, mcpServer *server.MCPServer) {
	// Generic JSON-RPC endpoint for MCP protocol
	mux.HandleFunc("/mcp", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read request body", http.StatusBadRequest)
			return
		}
		
		ctx := context.Background()
		response := mcpServer.HandleMessage(ctx, body)
		
		w.Header().Set("Content-Type", "application/json")
		responseData, _ := json.Marshal(response)
		w.Write(responseData)
		
		log.Printf("🔧 MCP: Handled JSON-RPC request")
	})
	
	// Convenience endpoints for direct HTTP access
	mux.HandleFunc("/tools/list", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		
		// Create JSON-RPC request for tools/list
		request := map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      1,
			"method":  "tools/list",
		}
		
		requestData, _ := json.Marshal(request)
		ctx := context.Background()
		response := mcpServer.HandleMessage(ctx, requestData)
		
		w.Header().Set("Content-Type", "application/json")
		responseData, _ := json.Marshal(response)
		w.Write(responseData)
		
		log.Printf("🔧 Tools: Listed tools")
	})
	
	mux.HandleFunc("/tools/call", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read request body", http.StatusBadRequest)
			return
		}
		
		// Parse the tool call request
		var toolCall struct {
			Name      string                 `json:"name"`
			Arguments map[string]interface{} `json:"arguments"`
		}
		
		if err := json.Unmarshal(body, &toolCall); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
		
		// Create JSON-RPC request for tools/call
		request := map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      1,
			"method":  "tools/call",
			"params": map[string]interface{}{
				"name":      toolCall.Name,
				"arguments": toolCall.Arguments,
			},
		}
		
		requestData, _ := json.Marshal(request)
		ctx := context.Background()
		response := mcpServer.HandleMessage(ctx, requestData)
		
		w.Header().Set("Content-Type", "application/json")
		responseData, _ := json.Marshal(response)
		w.Write(responseData)
		
		log.Printf("🔧 Tool Call: %s", toolCall.Name)
	})
	
	log.Printf("✅ MCP HTTP Server: All endpoints registered")
}