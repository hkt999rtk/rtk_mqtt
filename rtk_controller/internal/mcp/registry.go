package mcp

import (
	"rtk_controller/internal/mcp/tools"
	"rtk_controller/pkg/types"

	"github.com/sirupsen/logrus"
)

// ToolRegistry 工具註冊表類型別名
type ToolRegistry = tools.ToolRegistry

// NewToolRegistry 建立工具註冊表
func NewToolRegistry(logger *logrus.Logger) *ToolRegistry {
	return tools.NewToolRegistry(logger)
}

// ToolAdapter 工具適配器類型別名
type ToolAdapter = tools.ToolAdapter

// NewToolAdapter 建立工具適配器
func NewToolAdapter(llmTool types.LLMTool, logger *logrus.Logger) *ToolAdapter {
	return tools.NewToolAdapter(llmTool, logger)
}
