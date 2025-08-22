package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"rtk_controller/internal/llm"

	"github.com/sirupsen/logrus"
)

// ResourceRegistry 資源註冊表
type ResourceRegistry struct {
	providers map[string]ResourceProvider
	mutex     sync.RWMutex
	logger    *logrus.Logger
}

// ResourceProvider 資源提供者介面
type ResourceProvider interface {
	ListResources(ctx context.Context) ([]*MCPResource, error)
	ReadResource(ctx context.Context, uri string) (*MCPResourceContent, error)
	GetResourceInfo() ResourceInfo
}

// ResourceInfo 資源提供者資訊
type ResourceInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Version     string `json:"version"`
}

// MCPResource MCP 資源定義
type MCPResource struct {
	URI         string `json:"uri"`
	Name        string `json:"name"`
	Description string `json:"description"`
	MimeType    string `json:"mimeType"`
}

// MCPResourceContent MCP 資源內容
type MCPResourceContent struct {
	URI      string `json:"uri"`
	MimeType string `json:"mimeType"`
	Text     string `json:"text,omitempty"`
	Data     []byte `json:"data,omitempty"`
}

// MCPResourceListResult MCP 資源列表結果
type MCPResourceListResult struct {
	Resources []*MCPResource `json:"resources"`
}

// MCPResourceReadResult MCP 資源讀取結果
type MCPResourceReadResult struct {
	Contents []MCPResourceContent `json:"contents"`
}

// NewResourceRegistry 建立資源註冊表
func NewResourceRegistry(logger *logrus.Logger) *ResourceRegistry {
	return &ResourceRegistry{
		providers: make(map[string]ResourceProvider),
		logger:    logger,
	}
}

// Register 註冊資源提供者
func (rr *ResourceRegistry) Register(name string, provider ResourceProvider) error {
	if name == "" {
		return fmt.Errorf("provider name cannot be empty")
	}

	if provider == nil {
		return fmt.Errorf("provider cannot be nil")
	}

	rr.mutex.Lock()
	defer rr.mutex.Unlock()

	if _, exists := rr.providers[name]; exists {
		return fmt.Errorf("provider %s already registered", name)
	}

	rr.providers[name] = provider
	rr.logger.WithField("provider", name).Debug("Resource provider registered")

	return nil
}

// Unregister 取消註冊資源提供者
func (rr *ResourceRegistry) Unregister(name string) error {
	rr.mutex.Lock()
	defer rr.mutex.Unlock()

	if _, exists := rr.providers[name]; !exists {
		return fmt.Errorf("provider %s not found", name)
	}

	delete(rr.providers, name)
	rr.logger.WithField("provider", name).Debug("Resource provider unregistered")

	return nil
}

// ListResources 列出所有資源
func (rr *ResourceRegistry) ListResources(ctx context.Context) ([]*MCPResource, error) {
	rr.mutex.RLock()
	defer rr.mutex.RUnlock()

	var allResources []*MCPResource

	for providerName, provider := range rr.providers {
		resources, err := provider.ListResources(ctx)
		if err != nil {
			rr.logger.WithFields(logrus.Fields{
				"provider": providerName,
				"error":    err,
			}).Warning("Failed to list resources from provider")
			continue
		}

		allResources = append(allResources, resources...)
	}

	return allResources, nil
}

// ReadResource 讀取資源
func (rr *ResourceRegistry) ReadResource(ctx context.Context, uri string) (*MCPResourceContent, error) {
	rr.mutex.RLock()
	defer rr.mutex.RUnlock()

	// 嘗試從所有提供者中讀取資源
	for providerName, provider := range rr.providers {
		content, err := provider.ReadResource(ctx, uri)
		if err == nil {
			rr.logger.WithFields(logrus.Fields{
				"provider": providerName,
				"uri":      uri,
			}).Debug("Resource read successfully")
			return content, nil
		}

		// 記錄但不中斷，繼續嘗試其他提供者
		rr.logger.WithFields(logrus.Fields{
			"provider": providerName,
			"uri":      uri,
			"error":    err,
		}).Debug("Provider cannot read resource")
	}

	return nil, fmt.Errorf("resource %s not found", uri)
}

// GetProviders 取得所有提供者資訊
func (rr *ResourceRegistry) GetProviders() map[string]ResourceInfo {
	rr.mutex.RLock()
	defer rr.mutex.RUnlock()

	providers := make(map[string]ResourceInfo)
	for name, provider := range rr.providers {
		providers[name] = provider.GetResourceInfo()
	}

	return providers
}

// TopologyResourceProvider 拓撲資源提供者
type TopologyResourceProvider struct {
	toolEngine *llm.ToolEngine
	logger     *logrus.Logger
	cache      map[string]cachedResource
	cacheMutex sync.RWMutex
	cacheTTL   time.Duration
}

type cachedResource struct {
	content   *MCPResourceContent
	timestamp time.Time
}

// NewTopologyResourceProvider 建立拓撲資源提供者
func NewTopologyResourceProvider(toolEngine *llm.ToolEngine, logger *logrus.Logger) *TopologyResourceProvider {
	return &TopologyResourceProvider{
		toolEngine: toolEngine,
		logger:     logger,
		cache:      make(map[string]cachedResource),
		cacheTTL:   5 * time.Minute,
	}
}

// GetResourceInfo 取得資源提供者資訊
func (trp *TopologyResourceProvider) GetResourceInfo() ResourceInfo {
	return ResourceInfo{
		Name:        "topology",
		Description: "Network topology resources",
		Version:     "1.0.0",
	}
}

// ListResources 列出拓撲資源
func (trp *TopologyResourceProvider) ListResources(ctx context.Context) ([]*MCPResource, error) {
	return []*MCPResource{
		{
			URI:         "topology://network/current",
			Name:        "Current Network Topology",
			Description: "Real-time network topology information",
			MimeType:    "application/json",
		},
		{
			URI:         "topology://devices/list",
			Name:        "Device List",
			Description: "List of all network devices",
			MimeType:    "application/json",
		},
		{
			URI:         "topology://connections/graph",
			Name:        "Connection Graph",
			Description: "Network connection graph",
			MimeType:    "application/json",
		},
	}, nil
}

// ReadResource 讀取拓撲資源
func (trp *TopologyResourceProvider) ReadResource(ctx context.Context, uri string) (*MCPResourceContent, error) {
	// 檢查快取
	if content := trp.getFromCache(uri); content != nil {
		return content, nil
	}

	switch uri {
	case "topology://network/current":
		return trp.getCurrentTopology(ctx, uri)
	case "topology://devices/list":
		return trp.getDeviceList(ctx, uri)
	case "topology://connections/graph":
		return trp.getConnectionGraph(ctx, uri)
	default:
		return nil, fmt.Errorf("unknown topology resource URI: %s", uri)
	}
}

// getCurrentTopology 獲取當前拓撲
func (trp *TopologyResourceProvider) getCurrentTopology(ctx context.Context, uri string) (*MCPResourceContent, error) {
	// 使用拓撲工具獲取資料
	tool, exists := trp.toolEngine.GetTool("topology.get_full")
	if !exists {
		return nil, fmt.Errorf("topology.get_full tool not available")
	}

	result, err := tool.Execute(ctx, map[string]interface{}{
		"include_inactive": false,
		"detail_level":     "full",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get topology: %w", err)
	}

	if !result.Success {
		return nil, fmt.Errorf("topology tool failed: %s", result.Error)
	}

	data, err := json.MarshalIndent(result.Data, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal topology data: %w", err)
	}

	content := &MCPResourceContent{
		URI:      uri,
		MimeType: "application/json",
		Text:     string(data),
	}

	// 存入快取
	trp.putToCache(uri, content)

	return content, nil
}

// getDeviceList 獲取設備列表
func (trp *TopologyResourceProvider) getDeviceList(ctx context.Context, uri string) (*MCPResourceContent, error) {
	// 使用客戶端列表工具獲取資料
	tool, exists := trp.toolEngine.GetTool("clients.list")
	if !exists {
		return nil, fmt.Errorf("clients.list tool not available")
	}

	result, err := tool.Execute(ctx, map[string]interface{}{
		"active_only":     false,
		"include_history": true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get device list: %w", err)
	}

	if !result.Success {
		return nil, fmt.Errorf("clients.list tool failed: %s", result.Error)
	}

	data, err := json.MarshalIndent(result.Data, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal device data: %w", err)
	}

	content := &MCPResourceContent{
		URI:      uri,
		MimeType: "application/json",
		Text:     string(data),
	}

	// 存入快取
	trp.putToCache(uri, content)

	return content, nil
}

// getConnectionGraph 獲取連接圖
func (trp *TopologyResourceProvider) getConnectionGraph(ctx context.Context, uri string) (*MCPResourceContent, error) {
	// 這裡可以使用其他工具來獲取連接圖資料
	// 暫時返回基本的連接資訊
	data := map[string]interface{}{
		"type":        "connection_graph",
		"timestamp":   time.Now(),
		"description": "Network connection graph data",
		"data":        "Connection graph data would be here",
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal connection graph: %w", err)
	}

	content := &MCPResourceContent{
		URI:      uri,
		MimeType: "application/json",
		Text:     string(jsonData),
	}

	// 存入快取
	trp.putToCache(uri, content)

	return content, nil
}

// getFromCache 從快取獲取資源
func (trp *TopologyResourceProvider) getFromCache(uri string) *MCPResourceContent {
	trp.cacheMutex.RLock()
	defer trp.cacheMutex.RUnlock()

	cached, exists := trp.cache[uri]
	if !exists {
		return nil
	}

	// 檢查是否過期
	if time.Since(cached.timestamp) > trp.cacheTTL {
		return nil
	}

	return cached.content
}

// putToCache 存入快取
func (trp *TopologyResourceProvider) putToCache(uri string, content *MCPResourceContent) {
	trp.cacheMutex.Lock()
	defer trp.cacheMutex.Unlock()

	trp.cache[uri] = cachedResource{
		content:   content,
		timestamp: time.Now(),
	}
}

// DeviceResourceProvider 設備資源提供者 (佔位符實作)
type DeviceResourceProvider struct {
	toolEngine *llm.ToolEngine
	logger     *logrus.Logger
}

// NewDeviceResourceProvider 建立設備資源提供者
func NewDeviceResourceProvider(toolEngine *llm.ToolEngine, logger *logrus.Logger) *DeviceResourceProvider {
	return &DeviceResourceProvider{
		toolEngine: toolEngine,
		logger:     logger,
	}
}

// GetResourceInfo 取得資源提供者資訊
func (drp *DeviceResourceProvider) GetResourceInfo() ResourceInfo {
	return ResourceInfo{
		Name:        "devices",
		Description: "Device-specific resources",
		Version:     "1.0.0",
	}
}

// ListResources 列出設備資源
func (drp *DeviceResourceProvider) ListResources(ctx context.Context) ([]*MCPResource, error) {
	return []*MCPResource{
		{
			URI:         "devices://status/all",
			Name:        "All Device Status",
			Description: "Status information for all devices",
			MimeType:    "application/json",
		},
	}, nil
}

// ReadResource 讀取設備資源
func (drp *DeviceResourceProvider) ReadResource(ctx context.Context, uri string) (*MCPResourceContent, error) {
	switch uri {
	case "devices://status/all":
		data := map[string]interface{}{
			"type":      "device_status",
			"timestamp": time.Now(),
			"devices":   []interface{}{},
		}

		jsonData, _ := json.MarshalIndent(data, "", "  ")
		return &MCPResourceContent{
			URI:      uri,
			MimeType: "application/json",
			Text:     string(jsonData),
		}, nil
	default:
		return nil, fmt.Errorf("unknown device resource URI: %s", uri)
	}
}

// DiagnosticsResourceProvider 診斷資源提供者 (佔位符實作)
type DiagnosticsResourceProvider struct {
	toolEngine *llm.ToolEngine
	logger     *logrus.Logger
}

// NewDiagnosticsResourceProvider 建立診斷資源提供者
func NewDiagnosticsResourceProvider(toolEngine *llm.ToolEngine, logger *logrus.Logger) *DiagnosticsResourceProvider {
	return &DiagnosticsResourceProvider{
		toolEngine: toolEngine,
		logger:     logger,
	}
}

// GetResourceInfo 取得資源提供者資訊
func (drp *DiagnosticsResourceProvider) GetResourceInfo() ResourceInfo {
	return ResourceInfo{
		Name:        "diagnostics",
		Description: "Diagnostic history and reports",
		Version:     "1.0.0",
	}
}

// ListResources 列出診斷資源
func (drp *DiagnosticsResourceProvider) ListResources(ctx context.Context) ([]*MCPResource, error) {
	return []*MCPResource{
		{
			URI:         "diagnostics://history/recent",
			Name:        "Recent Diagnostic History",
			Description: "Recent diagnostic test results",
			MimeType:    "application/json",
		},
	}, nil
}

// ReadResource 讀取診斷資源
func (drp *DiagnosticsResourceProvider) ReadResource(ctx context.Context, uri string) (*MCPResourceContent, error) {
	switch uri {
	case "diagnostics://history/recent":
		data := map[string]interface{}{
			"type":      "diagnostic_history",
			"timestamp": time.Now(),
			"history":   []interface{}{},
		}

		jsonData, _ := json.MarshalIndent(data, "", "  ")
		return &MCPResourceContent{
			URI:      uri,
			MimeType: "application/json",
			Text:     string(jsonData),
		}, nil
	default:
		return nil, fmt.Errorf("unknown diagnostics resource URI: %s", uri)
	}
}
