package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

// HTTPTransport HTTP REST API 傳輸
type HTTPTransport struct {
	mcpServer  *MCPServer
	httpServer *http.Server
	config     HTTPConfig
	logger     *logrus.Logger
}

// NewHTTPTransport 建立 HTTP 傳輸
func NewHTTPTransport(mcpServer *MCPServer, config HTTPConfig, logger *logrus.Logger) *HTTPTransport {
	return &HTTPTransport{
		mcpServer: mcpServer,
		config:    config,
		logger:    logger,
	}
}

// Start 啟動 HTTP 傳輸
func (t *HTTPTransport) Start(ctx context.Context) error {
	if !t.config.Enabled {
		t.logger.Info("HTTP transport disabled")
		return nil
	}

	// 建立 HTTP router
	mux := http.NewServeMux()

	// MCP API endpoints
	mux.HandleFunc("/mcp/tools", t.corsMiddleware(t.handleTools))
	mux.HandleFunc("/mcp/tools/call", t.corsMiddleware(t.handleToolCall))
	mux.HandleFunc("/mcp/resources", t.corsMiddleware(t.handleResources))
	mux.HandleFunc("/mcp/resources/read", t.corsMiddleware(t.handleResourceRead))
	mux.HandleFunc("/mcp/prompts", t.corsMiddleware(t.handlePrompts))
	mux.HandleFunc("/mcp/prompts/get", t.corsMiddleware(t.handlePromptGet))
	mux.HandleFunc("/mcp/initialize", t.corsMiddleware(t.handleInitialize))
	mux.HandleFunc("/mcp/health", t.corsMiddleware(t.handleHealth))
	mux.HandleFunc("/mcp/info", t.corsMiddleware(t.handleInfo))

	// 建立 HTTP server
	t.httpServer = &http.Server{
		Addr:         fmt.Sprintf("%s:%d", t.config.Host, t.config.Port),
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// 啟動伺服器
	go func() {
		var err error
		if t.config.TLS.Enabled {
			t.logger.WithFields(logrus.Fields{
				"host":      t.config.Host,
				"port":      t.config.Port,
				"cert_file": t.config.TLS.CertFile,
			}).Info("Starting HTTPS server")
			err = t.httpServer.ListenAndServeTLS(t.config.TLS.CertFile, t.config.TLS.KeyFile)
		} else {
			t.logger.WithFields(logrus.Fields{
				"host": t.config.Host,
				"port": t.config.Port,
			}).Info("Starting HTTP server")
			err = t.httpServer.ListenAndServe()
		}

		if err != nil && err != http.ErrServerClosed {
			t.logger.WithError(err).Error("HTTP server failed")
		}
	}()

	t.logger.WithFields(logrus.Fields{
		"address": t.httpServer.Addr,
		"tls":     t.config.TLS.Enabled,
	}).Info("HTTP transport started")

	return nil
}

// Stop 停止 HTTP 傳輸
func (t *HTTPTransport) Stop(ctx context.Context) error {
	if t.httpServer == nil {
		return nil
	}

	t.logger.Info("Stopping HTTP transport")

	err := t.httpServer.Shutdown(ctx)
	if err != nil {
		t.logger.WithError(err).Warning("HTTP server shutdown error")
	} else {
		t.logger.Info("HTTP transport stopped")
	}

	return err
}

// corsMiddleware CORS 中介軟體
func (t *HTTPTransport) corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 設定 CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Max-Age", "86400")

		// 處理 preflight requests
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		// 記錄請求
		t.logger.WithFields(logrus.Fields{
			"method": r.Method,
			"path":   r.URL.Path,
			"remote": r.RemoteAddr,
		}).Debug("HTTP request")

		next.ServeHTTP(w, r)
	}
}

// handleTools 處理工具列表請求
func (t *HTTPTransport) handleTools(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		t.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// 取得工具列表
	adapters := t.mcpServer.toolRegistry.List()
	tools := make([]interface{}, len(adapters))

	for i, adapter := range adapters {
		tools[i] = adapter.GetMCPSchema()
	}

	response := map[string]interface{}{
		"tools": tools,
	}

	t.writeJSON(w, http.StatusOK, response)
}

// handleToolCall 處理工具調用請求
func (t *HTTPTransport) handleToolCall(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		t.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var request struct {
		Params struct {
			Name      string                 `json:"name"`
			Arguments map[string]interface{} `json:"arguments"`
		} `json:"params"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		t.writeError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	// 取得工具適配器
	adapter, err := t.mcpServer.toolRegistry.Get(request.Params.Name)
	if err != nil {
		t.writeError(w, http.StatusNotFound, fmt.Sprintf("Tool not found: %s", request.Params.Name))
		return
	}

	// 執行工具
	result, err := adapter.Execute(r.Context(), request.Params.Arguments)
	if err != nil {
		t.writeError(w, http.StatusInternalServerError, "Tool execution failed: "+err.Error())
		return
	}

	t.writeJSON(w, http.StatusOK, result)
}

// handleResources 處理資源列表請求
func (t *HTTPTransport) handleResources(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		t.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// 取得資源列表
	resources, err := t.mcpServer.resourceRegistry.ListResources(r.Context())
	if err != nil {
		t.writeError(w, http.StatusInternalServerError, "Failed to list resources: "+err.Error())
		return
	}

	response := MCPResourceListResult{
		Resources: resources,
	}

	t.writeJSON(w, http.StatusOK, response)
}

// handleResourceRead 處理資源讀取請求
func (t *HTTPTransport) handleResourceRead(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		t.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var request struct {
		Params struct {
			URI string `json:"uri"`
		} `json:"params"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		t.writeError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	// 讀取資源
	content, err := t.mcpServer.resourceRegistry.ReadResource(r.Context(), request.Params.URI)
	if err != nil {
		t.writeError(w, http.StatusNotFound, "Resource not found: "+err.Error())
		return
	}

	response := MCPResourceReadResult{
		Contents: []MCPResourceContent{*content},
	}

	t.writeJSON(w, http.StatusOK, response)
}

// handlePrompts 處理提示列表請求
func (t *HTTPTransport) handlePrompts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		t.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// 取得提示列表
	prompts, err := t.mcpServer.promptRegistry.ListPrompts(r.Context())
	if err != nil {
		t.writeError(w, http.StatusInternalServerError, "Failed to list prompts: "+err.Error())
		return
	}

	response := MCPListPromptsResult{
		Prompts: prompts,
	}

	t.writeJSON(w, http.StatusOK, response)
}

// handlePromptGet 處理提示取得請求
func (t *HTTPTransport) handlePromptGet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		t.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var request MCPGetPromptRequest

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		t.writeError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	// 取得提示內容
	result, err := t.mcpServer.promptRegistry.GetPrompt(r.Context(), request.Params.Name, request.Params.Arguments)
	if err != nil {
		t.writeError(w, http.StatusNotFound, "Prompt not found: "+err.Error())
		return
	}

	t.writeJSON(w, http.StatusOK, result)
}

// handleInitialize 處理初始化請求
func (t *HTTPTransport) handleInitialize(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		t.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// MCP 初始化回應
	response := map[string]interface{}{
		"protocolVersion": "2024-11-05",
		"capabilities": map[string]interface{}{
			"tools":     map[string]interface{}{},
			"resources": map[string]interface{}{},
			"prompts":   map[string]interface{}{},
		},
		"serverInfo": map[string]interface{}{
			"name":    t.mcpServer.config.Name,
			"version": t.mcpServer.config.Version,
		},
	}

	t.writeJSON(w, http.StatusOK, response)
}

// handleHealth 處理健康檢查請求
func (t *HTTPTransport) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		t.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// 檢查服務狀態
	isHealthy := t.mcpServer.IsStarted()
	status := "healthy"
	if !isHealthy {
		status = "unhealthy"
	}

	response := map[string]interface{}{
		"status":    status,
		"timestamp": time.Now(),
		"checks": map[string]interface{}{
			"server_started": isHealthy,
			"tools_count":    t.mcpServer.toolRegistry.Count(),
			"sessions_count": len(t.mcpServer.sessionManager.ListSessions()),
		},
	}

	statusCode := http.StatusOK
	if !isHealthy {
		statusCode = http.StatusServiceUnavailable
	}

	t.writeJSON(w, statusCode, response)
}

// handleInfo 處理伺服器資訊請求
func (t *HTTPTransport) handleInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		t.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	info := t.mcpServer.GetInfo()

	// 添加額外資訊
	response := map[string]interface{}{
		"server":    info,
		"tools":     t.mcpServer.toolRegistry.Count(),
		"resources": len(t.mcpServer.resourceRegistry.GetProviders()),
		"prompts":   t.mcpServer.promptRegistry.GetTemplateCount(),
		"sessions":  len(t.mcpServer.sessionManager.ListSessions()),
		"timestamp": time.Now(),
	}

	t.writeJSON(w, http.StatusOK, response)
}

// writeJSON 寫入 JSON 回應
func (t *HTTPTransport) writeJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		t.logger.WithError(err).Error("Failed to encode JSON response")
	}
}

// writeError 寫入錯誤回應
func (t *HTTPTransport) writeError(w http.ResponseWriter, statusCode int, message string) {
	t.logger.WithFields(logrus.Fields{
		"status":  statusCode,
		"message": message,
	}).Warning("HTTP error response")

	response := map[string]interface{}{
		"error": map[string]interface{}{
			"code":    statusCode,
			"message": message,
		},
		"timestamp": time.Now(),
	}

	t.writeJSON(w, statusCode, response)
}
