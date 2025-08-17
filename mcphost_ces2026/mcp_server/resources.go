package main

import (
	"context"
	"encoding/json"
	"log"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// AddExampleResources adds example resources to the MCP server
func AddExampleResources(s *server.MCPServer) {
	// Configuration resource
	resource := mcp.Resource{
		URI:         "config://current",
		Name:        "Current Configuration Settings",
		MIMEType:    "application/json",
		Description: "Display current server configuration",
	}

	s.AddResource(resource, func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		cfg := map[string]interface{}{
			"server": map[string]interface{}{
				"name":    config.Server.Name,
				"version": config.Server.Version,
				"status":  "running",
			},
			"lmstudio": map[string]interface{}{
				"base_url":    config.LMStudio.BaseURL,
				"model":       config.LMStudio.Model,
				"temperature": config.LMStudio.Temperature,
				"max_tokens":  config.LMStudio.MaxTokens,
			},
			"weatherapi": map[string]interface{}{
				"base_url": config.WeatherAPI.BaseURL,
				"api_key":  "***HIDDEN***", // Don't expose the actual API key
			},
		}

		data, err := json.MarshalIndent(cfg, "", "  ")
		if err != nil {
			return nil, err
		}

		return []mcp.ResourceContents{
			mcp.TextResourceContents{
				URI:      resource.URI,
				MIMEType: "application/json",
				Text:     string(data),
			},
		}, nil
	})

	log.Println("Registered resource: config://current")
}