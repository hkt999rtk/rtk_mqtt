package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"mcphost_ces2026/utils"
)

var config *utils.Config

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "config.json", "Configuration file path")
	flag.Parse()

	log.Printf("🌐 MCP Client starting...")

	// Load configuration
	cfg, err := utils.LoadConfig(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			log.Printf("Configuration file not found, using default settings")
			config = utils.GetDefaultConfig()
		} else {
			log.Fatalf("Failed to load configuration: %v", err)
		}
	} else {
		config = cfg
	}

	// Start HTTP server
	port := config.HTTPServer.Port
	host := config.HTTPServer.Host

	mux := http.NewServeMux()
	
	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	
	// Models endpoint
	mux.HandleFunc("/v1/models", func(w http.ResponseWriter, r *http.Request) {
		models := []map[string]interface{}{
			{
				"id":       "mcp-client",
				"object":   "model",
				"created":  time.Now().Unix(),
				"owned_by": "mcp-client",
			},
		}
		
		response := map[string]interface{}{
			"object": "list",
			"data":   models,
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})
	
	// Simple completions endpoint
	mux.HandleFunc("/v1/completions", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read request body", http.StatusBadRequest)
			return
		}
		
		var req struct {
			Model     string `json:"model"`
			Prompt    string `json:"prompt"`
			MaxTokens int    `json:"max_tokens"`
		}
		
		if err := json.Unmarshal(body, &req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
		
		// Simple response for now
		response := map[string]interface{}{
			"id":      fmt.Sprintf("comp-%d", time.Now().Unix()),
			"object":  "text_completion",
			"created": time.Now().Unix(),
			"model":   req.Model,
			"choices": []map[string]interface{}{
				{
					"text":          "Hello! This is a simple MCP Client response.",
					"index":         0,
					"finish_reason": "stop",
				},
			},
			"usage": map[string]interface{}{
				"prompt_tokens":     10,
				"completion_tokens": 8,
				"total_tokens":      18,
			},
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		
		log.Printf("🔧 Completion: %s - %s", req.Model, req.Prompt[:min(50, len(req.Prompt))])
	})

	// Start HTTP server
	addr := fmt.Sprintf("%s:%d", host, port)
	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	go func() {
		log.Printf("🌐 MCP Client listening on %s", addr)
		log.Printf("📚 Available endpoints:")
		log.Printf("   - GET  /health")
		log.Printf("   - GET  /v1/models")
		log.Printf("   - POST /v1/completions")
		
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server failed: %v", err)
		}
	}()

	// Wait for interrupt signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	<-sigCh
	log.Println("🔄 Received termination signal, shutting down...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("❌ Server shutdown error: %v", err)
	} else {
		log.Printf("✅ MCP Client shutdown complete")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}