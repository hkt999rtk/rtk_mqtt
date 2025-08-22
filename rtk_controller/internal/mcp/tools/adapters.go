package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"rtk_controller/pkg/types"

	"github.com/sirupsen/logrus"
)

// ToolAdapter 將 LLM 工具適配為 MCP 工具
type ToolAdapter struct {
	llmTool types.LLMTool
	logger  *logrus.Logger

	// 快取的 schema 資訊
	name        string
	description string
	category    string
	parameters  map[string]interface{}
}

// NewToolAdapter 建立工具適配器
func NewToolAdapter(llmTool types.LLMTool, logger *logrus.Logger) *ToolAdapter {
	adapter := &ToolAdapter{
		llmTool: llmTool,
		logger:  logger,
	}

	// 初始化基本資訊
	adapter.name = llmTool.Name()
	adapter.description = llmTool.Description()
	adapter.category = extractCategoryFromName(llmTool.Name())
	adapter.parameters = generateDefaultParametersSchema(llmTool.Name())

	return adapter
}

// GetName 返回工具名稱
func (ta *ToolAdapter) GetName() string {
	return ta.name
}

// GetDescription 返回工具描述
func (ta *ToolAdapter) GetDescription() string {
	return ta.description
}

// GetCategory 返回工具分類
func (ta *ToolAdapter) GetCategory() string {
	return ta.category
}

// GetMCPSchema 返回 MCP 工具 schema
func (ta *ToolAdapter) GetMCPSchema() MCPTool {
	return MCPTool{
		Name:        ta.name,
		Description: ta.description,
		InputSchema: MCPInputSchema{
			Type:       "object",
			Properties: ta.parameters,
		},
	}
}

// Execute 執行工具
func (ta *ToolAdapter) Execute(ctx context.Context, arguments map[string]interface{}) (*MCPToolResult, error) {
	startTime := time.Now()

	ta.logger.WithFields(logrus.Fields{
		"tool":      ta.name,
		"arguments": arguments,
	}).Debug("Executing tool")

	// 驗證參數
	if err := ta.llmTool.Validate(arguments); err != nil {
		ta.logger.WithFields(logrus.Fields{
			"tool":  ta.name,
			"error": err,
		}).Warning("Tool parameter validation failed")

		return &MCPToolResult{
			Content: []MCPContent{
				{
					Type: "text",
					Text: fmt.Sprintf("Parameter validation failed: %v", err),
				},
			},
			IsError: true,
		}, nil
	}

	// 執行 LLM 工具
	result, err := ta.llmTool.Execute(ctx, arguments)
	executionTime := time.Since(startTime)

	ta.logger.WithFields(logrus.Fields{
		"tool":           ta.name,
		"execution_time": executionTime,
		"success":        err == nil && result != nil && result.Success,
	}).Debug("Tool execution completed")

	if err != nil {
		ta.logger.WithFields(logrus.Fields{
			"tool":  ta.name,
			"error": err,
		}).Error("Tool execution failed")

		return &MCPToolResult{
			Content: []MCPContent{
				{
					Type: "text",
					Text: fmt.Sprintf("Tool execution failed: %v", err),
				},
			},
			IsError: true,
		}, nil
	}

	// 轉換結果為 MCP 格式
	mcpResult := ta.convertLLMResultToMCPResult(result, executionTime)
	return mcpResult, nil
}

// convertLLMResultToMCPResult 轉換 LLM 結果為 MCP 結果
func (ta *ToolAdapter) convertLLMResultToMCPResult(result *types.ToolResult, executionTime time.Duration) *MCPToolResult {
	content := []MCPContent{}

	if result.Success {
		// 成功結果
		if result.Data != nil {
			// 將結果資料序列化為 JSON
			dataJSON, err := json.MarshalIndent(result.Data, "", "  ")
			if err != nil {
				content = append(content, MCPContent{
					Type: "text",
					Text: fmt.Sprintf("Failed to serialize result data: %v", err),
				})
			} else {
				content = append(content, MCPContent{
					Type: "text",
					Text: string(dataJSON),
				})
			}
		}

		// 錯誤資訊
		if !result.Success && result.Error != "" {
			content = append(content, MCPContent{
				Type: "text",
				Text: fmt.Sprintf("\n## Error\n%s", result.Error),
			})
		}

		// 執行資訊
		content = append(content, MCPContent{
			Type: "text",
			Text: fmt.Sprintf("\n## Execution Info\n- Tool: %s\n- Duration: %v\n- Success: %t",
				ta.name, executionTime, result.Success),
		})

	} else {
		// 錯誤結果
		content = append(content, MCPContent{
			Type: "text",
			Text: fmt.Sprintf("## Error\n%s", result.Error),
		})

		// 執行資訊
		content = append(content, MCPContent{
			Type: "text",
			Text: fmt.Sprintf("\n## Execution Info\n- Tool: %s\n- Duration: %v\n- Success: %t",
				ta.name, executionTime, result.Success),
		})
	}

	return &MCPToolResult{
		Content: content,
		IsError: !result.Success,
	}
}

// extractCategoryFromName 從工具名稱提取分類
func extractCategoryFromName(name string) string {
	parts := strings.Split(name, ".")
	if len(parts) > 0 {
		return parts[0]
	}
	return "general"
}

// generateDefaultParametersSchema 為工具生成預設參數 schema
func generateDefaultParametersSchema(toolName string) map[string]interface{} {
	properties := make(map[string]interface{})

	// 根據工具名稱提供一些通用參數
	switch {
	case strings.Contains(toolName, "topology"):
		properties["include_inactive"] = map[string]interface{}{
			"type":        "boolean",
			"description": "Include inactive devices in the topology",
			"default":     false,
		}
		properties["detail_level"] = map[string]interface{}{
			"type":        "string",
			"description": "Level of detail in the topology data",
			"enum":        []string{"basic", "full", "detailed"},
			"default":     "full",
		}
	case strings.Contains(toolName, "speedtest"):
		properties["target"] = map[string]interface{}{
			"type":        "string",
			"description": "Target for speed test",
			"enum":        []string{"wan", "lan", "client"},
			"default":     "wan",
		}
		properties["duration"] = map[string]interface{}{
			"type":        "integer",
			"description": "Test duration in seconds",
			"default":     30,
		}
	case strings.Contains(toolName, "wifi"):
		properties["interface"] = map[string]interface{}{
			"type":        "string",
			"description": "WiFi interface to analyze",
		}
		properties["band"] = map[string]interface{}{
			"type":        "string",
			"description": "WiFi band to focus on",
			"enum":        []string{"2.4GHz", "5GHz", "6GHz", "all"},
			"default":     "all",
		}
	case strings.Contains(toolName, "ping"):
		properties["target"] = map[string]interface{}{
			"type":        "string",
			"description": "Target host to ping",
			"default":     "8.8.8.8",
		}
		properties["count"] = map[string]interface{}{
			"type":        "integer",
			"description": "Number of ping packets",
			"default":     5,
		}
	case strings.Contains(toolName, "clients"):
		properties["active_only"] = map[string]interface{}{
			"type":        "boolean",
			"description": "Show only active clients",
			"default":     true,
		}
		properties["include_history"] = map[string]interface{}{
			"type":        "boolean",
			"description": "Include connection history",
			"default":     false,
		}
	default:
		// 通用參數
		properties["verbose"] = map[string]interface{}{
			"type":        "boolean",
			"description": "Enable verbose output",
			"default":     false,
		}
	}

	return properties
}

// MCPTool MCP 工具定義
type MCPTool struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema MCPInputSchema `json:"inputSchema"`
}

// MCPInputSchema MCP 輸入 schema
type MCPInputSchema struct {
	Type       string                 `json:"type"`
	Properties map[string]interface{} `json:"properties"`
	Required   []string               `json:"required,omitempty"`
}

// MCPToolResult MCP 工具執行結果
type MCPToolResult struct {
	Content []MCPContent `json:"content"`
	IsError bool         `json:"isError,omitempty"`
}

// MCPContent MCP 內容
type MCPContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// MCPCallToolRequest MCP 工具調用請求
type MCPCallToolRequest struct {
	Params MCPCallToolParams `json:"params"`
}

// MCPCallToolParams MCP 工具調用參數
type MCPCallToolParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
}
