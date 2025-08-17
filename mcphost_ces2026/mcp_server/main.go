package main

import (
	"context"
	"flag"
	"fmt"
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
	var port int
	flag.StringVar(&configPath, "config", "config.json", "Configuration file path")
	flag.IntVar(&port, "port", 8081, "HTTP server port")
	flag.Parse()

	log.Printf("🛠️ MCP Server starting...")

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

	// Override port if specified via command line
	if port != 8081 {
		config.HTTPServer.Port = port
	}

	// Create and start MCP server
	mcpServer := createMCPServer()
	
	// Setup HTTP routes for MCP protocol
	mux := http.NewServeMux()
	
	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	
	// MCP protocol endpoints will be added by server.go
	startMCPHTTPServer(mux, mcpServer)
	
	// Start HTTP server
	addr := fmt.Sprintf("%s:%d", "0.0.0.0", config.HTTPServer.Port)
	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	go func() {
		log.Printf("🌐 MCP Server listening on %s", addr)
		log.Printf("📚 Available endpoints:")
		log.Printf("   - GET  /health")
		log.Printf("   - POST /tools/list")
		log.Printf("   - POST /tools/call")
		log.Printf("   - POST /resources/list")
		log.Printf("   - POST /prompts/list")
		
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
		log.Printf("✅ MCP Server shutdown complete")
	}
}