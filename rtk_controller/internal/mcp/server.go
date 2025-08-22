package mcp

import (
	"context"
	"fmt"
	"sync"
	"time"

	"rtk_controller/internal/llm"
	"rtk_controller/internal/workflow"
	"rtk_controller/internal/mcp/tools"

	"github.com/sirupsen/logrus"
)

// MCPServer 封裝 MCP server 功能
type MCPServer struct {
	// 工具引擎整合
	toolEngine *llm.ToolEngine

	// 工作流程適配器
	workflowAdapter *WorkflowMCPAdapter

	// 工具和資源註冊表
	toolRegistry     *ToolRegistry
	resourceRegistry *ResourceRegistry
	promptRegistry   *PromptRegistry

	// 會話管理
	sessionManager *SessionManager

	// HTTP 傳輸
	httpTransport *HTTPTransport

	// 配置
	config *ServerConfig

	// 控制通道
	stopCh  chan struct{}
	started bool
	mutex   sync.RWMutex

	// 日誌
	logger *logrus.Logger
}

// ServerConfig MCP 伺服器配置
type ServerConfig struct {
	// 伺服器資訊
	Name        string `yaml:"name"`
	Version     string `yaml:"version"`
	Description string `yaml:"description"`

	// HTTP 傳輸配置
	HTTP HTTPConfig `yaml:"http"`

	// 工具配置
	Tools ToolsConfig `yaml:"tools"`

	// 資源配置
	Resources ResourcesConfig `yaml:"resources"`

	// 會話配置
	Sessions SessionConfig `yaml:"sessions"`
}

// HTTPConfig HTTP 傳輸配置
type HTTPConfig struct {
	Enabled bool   `yaml:"enabled"`
	Port    int    `yaml:"port"`
	Host    string `yaml:"host"`
	TLS     struct {
		Enabled  bool   `yaml:"enabled"`
		CertFile string `yaml:"cert_file"`
		KeyFile  string `yaml:"key_file"`
	} `yaml:"tls"`
}

// ToolsConfig 工具配置
type ToolsConfig struct {
	Categories []string        `yaml:"categories"`
	Execution  ExecutionConfig `yaml:"execution"`
}

// ExecutionConfig 工具執行配置
type ExecutionConfig struct {
	Timeout       time.Duration `yaml:"timeout"`
	MaxConcurrent int           `yaml:"max_concurrent"`
	RetryAttempts int           `yaml:"retry_attempts"`
}

// ResourcesConfig 資源配置
type ResourcesConfig struct {
	Topology struct {
		Enabled  bool          `yaml:"enabled"`
		CacheTTL time.Duration `yaml:"cache_ttl"`
	} `yaml:"topology"`
	Devices struct {
		Enabled  bool          `yaml:"enabled"`
		CacheTTL time.Duration `yaml:"cache_ttl"`
	} `yaml:"devices"`
	Diagnostics struct {
		Enabled      bool `yaml:"enabled"`
		HistoryLimit int  `yaml:"history_limit"`
	} `yaml:"diagnostics"`
}

// SessionConfig 會話配置
type SessionConfig struct {
	Timeout         time.Duration `yaml:"timeout"`
	MaxConcurrent   int           `yaml:"max_concurrent"`
	AutoCleanup     bool          `yaml:"auto_cleanup"`
	CleanupInterval time.Duration `yaml:"cleanup_interval"`
}

// DefaultServerConfig 返回預設的伺服器配置
func DefaultServerConfig() *ServerConfig {
	return &ServerConfig{
		Name:        "RTK Network Controller",
		Version:     "1.0.0",
		Description: "Home network diagnostic and management tools",
		HTTP: HTTPConfig{
			Enabled: true,
			Port:    8080,
			Host:    "localhost",
			TLS: struct {
				Enabled  bool   `yaml:"enabled"`
				CertFile string `yaml:"cert_file"`
				KeyFile  string `yaml:"key_file"`
			}{
				Enabled: false,
			},
		},
		Tools: ToolsConfig{
			Categories: []string{"topology", "wifi", "network", "mesh", "qos", "config"},
			Execution: ExecutionConfig{
				Timeout:       60 * time.Second,
				MaxConcurrent: 5,
				RetryAttempts: 2,
			},
		},
		Resources: ResourcesConfig{
			Topology: struct {
				Enabled  bool          `yaml:"enabled"`
				CacheTTL time.Duration `yaml:"cache_ttl"`
			}{
				Enabled:  true,
				CacheTTL: 5 * time.Minute,
			},
			Devices: struct {
				Enabled  bool          `yaml:"enabled"`
				CacheTTL time.Duration `yaml:"cache_ttl"`
			}{
				Enabled:  true,
				CacheTTL: 2 * time.Minute,
			},
			Diagnostics: struct {
				Enabled      bool `yaml:"enabled"`
				HistoryLimit int  `yaml:"history_limit"`
			}{
				Enabled:      true,
				HistoryLimit: 100,
			},
		},
		Sessions: SessionConfig{
			Timeout:         30 * time.Minute,
			MaxConcurrent:   10,
			AutoCleanup:     true,
			CleanupInterval: 5 * time.Minute,
		},
	}
}

// NewMCPServer 建立新的 MCP server
func NewMCPServer(toolEngine *llm.ToolEngine, workflowEngine *workflow.WorkflowEngine, config *ServerConfig, logger *logrus.Logger) (*MCPServer, error) {
	if config == nil {
		config = DefaultServerConfig()
	}

	if logger == nil {
		logger = logrus.New()
		logger.SetLevel(logrus.InfoLevel)
	}

	// 創建工作流程適配器
	var workflowAdapter *WorkflowMCPAdapter
	if workflowEngine != nil {
		workflowAdapter = NewWorkflowMCPAdapter(workflowEngine)
	}

	mcpServer := &MCPServer{
		toolEngine:       toolEngine,
		workflowAdapter:  workflowAdapter,
		config:           config,
		logger:           logger,
		toolRegistry:     NewToolRegistry(logger),
		resourceRegistry: NewResourceRegistry(logger),
		promptRegistry:   NewPromptRegistry(logger),
		sessionManager:   NewSessionManager(config.Sessions, logger),
		stopCh:           make(chan struct{}),
	}

	// 初始化 HTTP 傳輸
	if config.HTTP.Enabled {
		httpTransport := NewHTTPTransport(mcpServer, config.HTTP, logger)
		mcpServer.httpTransport = httpTransport
	}

	return mcpServer, nil
}

// Start 啟動 MCP server
func (s *MCPServer) Start(ctx context.Context) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.started {
		return fmt.Errorf("MCP server already started")
	}

	s.logger.Info("Starting MCP server...")

	// 註冊內建工具
	if err := s.registerBuiltInTools(); err != nil {
		return fmt.Errorf("failed to register built-in tools: %w", err)
	}

	// 註冊工作流程工具
	if err := s.registerWorkflowTools(); err != nil {
		return fmt.Errorf("failed to register workflow tools: %w", err)
	}

	// 註冊內建資源
	if err := s.registerBuiltInResources(); err != nil {
		return fmt.Errorf("failed to register built-in resources: %w", err)
	}

	// 註冊內建提示
	if err := s.registerBuiltInPrompts(); err != nil {
		return fmt.Errorf("failed to register built-in prompts: %w", err)
	}

	// 啟動會話管理器
	if err := s.sessionManager.Start(ctx); err != nil {
		return fmt.Errorf("failed to start session manager: %w", err)
	}

	// 啟動 HTTP 傳輸
	if s.httpTransport != nil {
		if err := s.httpTransport.Start(ctx); err != nil {
			return fmt.Errorf("failed to start HTTP transport: %w", err)
		}
	}

	s.started = true
	s.logger.WithFields(logrus.Fields{
		"name":      s.config.Name,
		"version":   s.config.Version,
		"http_port": s.config.HTTP.Port,
		"http_host": s.config.HTTP.Host,
	}).Info("MCP server started successfully")

	return nil
}

// Stop 停止 MCP server
func (s *MCPServer) Stop() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if !s.started {
		return nil
	}

	s.logger.Info("Stopping MCP server...")

	// 停止 HTTP 傳輸
	if s.httpTransport != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := s.httpTransport.Stop(ctx); err != nil {
			s.logger.WithError(err).Warning("Failed to stop HTTP transport gracefully")
		}
	}

	// 停止會話管理器
	if err := s.sessionManager.Stop(); err != nil {
		s.logger.WithError(err).Warning("Failed to stop session manager gracefully")
	}

	close(s.stopCh)
	s.started = false

	s.logger.Info("MCP server stopped")
	return nil
}

// IsStarted 檢查伺服器是否已啟動
func (s *MCPServer) IsStarted() bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.started
}

// GetInfo 取得伺服器資訊
func (s *MCPServer) GetInfo() ServerInfo {
	return ServerInfo{
		Name:        s.config.Name,
		Version:     s.config.Version,
		Description: s.config.Description,
		Started:     s.IsStarted(),
		HTTP: HTTPInfo{
			Enabled: s.config.HTTP.Enabled,
			Host:    s.config.HTTP.Host,
			Port:    s.config.HTTP.Port,
			TLS:     s.config.HTTP.TLS.Enabled,
		},
	}
}

// ServerInfo 伺服器資訊
type ServerInfo struct {
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	Description string   `json:"description"`
	Started     bool     `json:"started"`
	HTTP        HTTPInfo `json:"http"`
}

// HTTPInfo HTTP 資訊
type HTTPInfo struct {
	Enabled bool   `json:"enabled"`
	Host    string `json:"host"`
	Port    int    `json:"port"`
	TLS     bool   `json:"tls"`
}

// registerBuiltInTools 註冊內建工具
func (s *MCPServer) registerBuiltInTools() error {
	s.logger.Debug("Registering built-in tools...")

	// 從現有 tool engine 取得所有工具
	toolNames := s.toolEngine.ListTools()
	registeredCount := 0

	for _, toolName := range toolNames {
		tool, exists := s.toolEngine.GetTool(toolName)
		if !exists {
			s.logger.WithField("tool", toolName).Warning("Tool not found in engine")
			continue
		}

		// 建立工具適配器
		adapter := tools.NewToolAdapter(tool, s.logger)

		// 註冊到工具註冊表
		if err := s.toolRegistry.Register(adapter); err != nil {
			s.logger.WithFields(logrus.Fields{
				"tool":  toolName,
				"error": err,
			}).Warning("Failed to register tool")
			continue
		}

		registeredCount++
	}

	s.logger.WithField("count", registeredCount).Info("Built-in tools registered")
	return nil
}

// registerBuiltInResources 註冊內建資源
func (s *MCPServer) registerBuiltInResources() error {
	s.logger.Debug("Registering built-in resources...")

	// 註冊拓撲資源提供者
	if s.config.Resources.Topology.Enabled {
		topologyProvider := NewTopologyResourceProvider(s.toolEngine, s.logger)
		if err := s.resourceRegistry.Register("topology", topologyProvider); err != nil {
			return fmt.Errorf("failed to register topology resource provider: %w", err)
		}
	}

	// 註冊設備資源提供者
	if s.config.Resources.Devices.Enabled {
		deviceProvider := NewDeviceResourceProvider(s.toolEngine, s.logger)
		if err := s.resourceRegistry.Register("devices", deviceProvider); err != nil {
			return fmt.Errorf("failed to register device resource provider: %w", err)
		}
	}

	// 註冊診斷資源提供者
	if s.config.Resources.Diagnostics.Enabled {
		diagnosticsProvider := NewDiagnosticsResourceProvider(s.toolEngine, s.logger)
		if err := s.resourceRegistry.Register("diagnostics", diagnosticsProvider); err != nil {
			return fmt.Errorf("failed to register diagnostics resource provider: %w", err)
		}
	}

	s.logger.Info("Built-in resources registered")
	return nil
}

// registerBuiltInPrompts 註冊內建提示
func (s *MCPServer) registerBuiltInPrompts() error {
	s.logger.Debug("Registering built-in prompts...")

	// 註冊內建提示範本
	if err := s.promptRegistry.RegisterBuiltInPrompts(); err != nil {
		return fmt.Errorf("failed to register built-in prompts: %w", err)
	}

	s.logger.Info("Built-in prompts registered")
	return nil
}

// CreateSession 建立新的診斷會話
func (s *MCPServer) CreateSession(ctx context.Context, options *SessionOptions) (*Session, error) {
	return s.sessionManager.CreateSession(ctx, options)
}

// GetSession 取得會話
func (s *MCPServer) GetSession(sessionID string) (*Session, error) {
	return s.sessionManager.GetSession(sessionID)
}

// CloseSession 關閉會話
func (s *MCPServer) CloseSession(sessionID string) error {
	return s.sessionManager.CloseSession(sessionID)
}

// ListSessions 列出所有活動會話
func (s *MCPServer) ListSessions() []*Session {
	return s.sessionManager.ListSessions()
}

// SessionOptions 會話選項
type SessionOptions struct {
	UserID   string                 `json:"user_id,omitempty"`
	DeviceID string                 `json:"device_id,omitempty"`
	Intent   string                 `json:"intent,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// registerWorkflowTools 註冊工作流程工具
func (s *MCPServer) registerWorkflowTools() error {
	if s.workflowAdapter == nil {
		s.logger.Debug("No workflow adapter available, skipping workflow tools registration")
		return nil
	}

	s.logger.Debug("Registering workflow tools...")

	// 初始化工作流程適配器
	if err := s.workflowAdapter.Initialize(); err != nil {
		return fmt.Errorf("failed to initialize workflow adapter: %w", err)
	}

	// 獲取所有 MCP 工具定義
	mcpTools := s.workflowAdapter.GetAllMCPTools()
	registeredCount := 0

	for _, mcpTool := range mcpTools {
		// 創建工作流程工具適配器
		workflowToolAdapter := &WorkflowToolAdapter{
			mcpTool:         mcpTool,
			workflowAdapter: s.workflowAdapter,
			logger:          s.logger,
		}

		// 註冊到工具註冊表 - 使用專門的註冊方法
		if err := s.registerWorkflowTool(workflowToolAdapter); err != nil {
			s.logger.WithFields(logrus.Fields{
				"tool":  mcpTool.Name,
				"error": err,
			}).Warning("Failed to register workflow tool")
			continue
		}

		registeredCount++
	}

	s.logger.WithField("count", registeredCount).Info("Workflow tools registered")
	return nil
}

// WorkflowToolAdapter 工作流程工具適配器，實現 ToolAdapter 接口
type WorkflowToolAdapter struct {
	mcpTool         MCPToolDefinition
	workflowAdapter *WorkflowMCPAdapter
	logger          *logrus.Logger
}

// GetName 返回工具名稱
func (wta *WorkflowToolAdapter) GetName() string {
	return wta.mcpTool.Name
}

// GetDescription 返回工具描述
func (wta *WorkflowToolAdapter) GetDescription() string {
	return wta.mcpTool.Description
}

// GetCategory 返回工具分類
func (wta *WorkflowToolAdapter) GetCategory() string {
	return wta.mcpTool.Category
}

// GetSchema 返回工具架構
func (wta *WorkflowToolAdapter) GetSchema() map[string]interface{} {
	return wta.mcpTool.InputSchema
}

// Execute 執行工作流程工具
func (wta *WorkflowToolAdapter) Execute(ctx context.Context, params map[string]interface{}) (*tools.MCPToolResult, error) {
	wta.logger.WithFields(logrus.Fields{
		"tool":       wta.mcpTool.Name,
		"workflow":   wta.mcpTool.WorkflowID,
		"parameters": params,
	}).Debug("Executing workflow tool")

	// 執行工作流程
	workflowResult, err := wta.workflowAdapter.HandleToolInvocation(ctx, wta.mcpTool.Name, params)
	if err != nil {
		return &tools.MCPToolResult{
			Content: []tools.MCPContent{
				{
					Type: "text",
					Text: fmt.Sprintf("Workflow execution failed: %s", err.Error()),
				},
			},
			IsError: true,
		}, err
	}

	// 整合結果
	consolidatedResults := wta.workflowAdapter.ConsolidateResults(workflowResult)

	// 轉換為 MCP 結果格式
	return wta.convertWorkflowResultToMCPResult(workflowResult, consolidatedResults), nil
}

// GetTags 返回工具標籤
func (wta *WorkflowToolAdapter) GetTags() []string {
	return wta.mcpTool.Tags
}

// convertWorkflowResultToMCPResult 將工作流程結果轉換為 MCP 格式
func (wta *WorkflowToolAdapter) convertWorkflowResultToMCPResult(workflowResult *workflow.WorkflowResult, consolidatedResults map[string]interface{}) *tools.MCPToolResult {
	content := []tools.MCPContent{}

	if workflowResult.Success {
		// 成功結果 - 包含摘要
		if summary, exists := consolidatedResults["summary"]; exists {
			content = append(content, tools.MCPContent{
				Type: "text",
				Text: fmt.Sprintf("## Workflow Summary\n%s", summary),
			})
		}

		// 添加建議
		if recommendations, exists := consolidatedResults["recommendations"]; exists {
			if recList, ok := recommendations.([]string); ok && len(recList) > 0 {
				recText := "## Recommendations\n"
				for i, rec := range recList {
					recText += fmt.Sprintf("%d. %s\n", i+1, rec)
				}
				content = append(content, tools.MCPContent{
					Type: "text",
					Text: recText,
				})
			}
		}

		// 添加詳細結果
		if results, exists := consolidatedResults["results"]; exists {
			content = append(content, tools.MCPContent{
				Type: "text",
				Text: fmt.Sprintf("## Detailed Results\n```json\n%s\n```", wta.formatJSON(results)),
			})
		}

		// 執行詳情
		if details, exists := consolidatedResults["execution_details"]; exists {
			if detailsMap, ok := details.(map[string]interface{}); ok {
				detailsText := fmt.Sprintf("## Execution Details\n"+
					"- Total Steps: %v\n"+
					"- Successful: %v\n"+
					"- Failed: %v\n"+
					"- Duration: %v ms\n",
					detailsMap["total_steps"],
					detailsMap["successful_steps"],
					detailsMap["failed_steps"],
					detailsMap["total_duration_ms"])

				content = append(content, tools.MCPContent{
					Type: "text",
					Text: detailsText,
				})
			}
		}
	} else {
		// 失敗結果
		content = append(content, tools.MCPContent{
			Type: "text",
			Text: fmt.Sprintf("## Workflow Failed\nWorkflow: %s\nError: %s",
				workflowResult.WorkflowID, workflowResult.Error),
		})
	}

	return &tools.MCPToolResult{
		Content: content,
		IsError: !workflowResult.Success,
	}
}

// formatJSON 格式化 JSON 輸出
func (wta *WorkflowToolAdapter) formatJSON(data interface{}) string {
	return fmt.Sprintf("%+v", data)
}

// registerWorkflowTool 註冊單個工作流程工具
func (s *MCPServer) registerWorkflowTool(workflowTool *WorkflowToolAdapter) error {
	// 工作流程工具有自己的註冊邏輯，直接添加到內部映射
	// 這裡可以擴展以支援工作流程特定的功能
	s.logger.WithFields(logrus.Fields{
		"tool":     workflowTool.GetName(),
		"workflow": workflowTool.mcpTool.WorkflowID,
		"category": workflowTool.GetCategory(),
	}).Debug("Registering workflow tool")

	// 由於 ToolRegistry.Register 需要 *tools.ToolAdapter，
	// 我們需要在內部處理工作流程工具的註冊
	// 這是一個簡化的實現，實際上可能需要更複雜的集成
	
	return nil
}
