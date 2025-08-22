package tools

import (
	"fmt"
	"sync"

	"github.com/sirupsen/logrus"
)

// ToolRegistry 工具註冊表
type ToolRegistry struct {
	tools  map[string]*ToolAdapter
	mutex  sync.RWMutex
	logger *logrus.Logger
}

// NewToolRegistry 建立新的工具註冊表
func NewToolRegistry(logger *logrus.Logger) *ToolRegistry {
	return &ToolRegistry{
		tools:  make(map[string]*ToolAdapter),
		logger: logger,
	}
}

// Register 註冊工具
func (r *ToolRegistry) Register(adapter *ToolAdapter) error {
	if adapter == nil {
		return fmt.Errorf("adapter cannot be nil")
	}

	name := adapter.GetName()
	if name == "" {
		return fmt.Errorf("tool name cannot be empty")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	if _, exists := r.tools[name]; exists {
		return fmt.Errorf("tool %s already registered", name)
	}

	r.tools[name] = adapter
	r.logger.WithField("tool", name).Debug("Tool registered")

	return nil
}

// Unregister 取消註冊工具
func (r *ToolRegistry) Unregister(name string) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if _, exists := r.tools[name]; !exists {
		return fmt.Errorf("tool %s not found", name)
	}

	delete(r.tools, name)
	r.logger.WithField("tool", name).Debug("Tool unregistered")

	return nil
}

// Get 取得工具
func (r *ToolRegistry) Get(name string) (*ToolAdapter, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	adapter, exists := r.tools[name]
	if !exists {
		return nil, fmt.Errorf("tool %s not found", name)
	}

	return adapter, nil
}

// List 列出所有工具
func (r *ToolRegistry) List() []*ToolAdapter {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	adapters := make([]*ToolAdapter, 0, len(r.tools))
	for _, adapter := range r.tools {
		adapters = append(adapters, adapter)
	}

	return adapters
}

// ListNames 列出所有工具名稱
func (r *ToolRegistry) ListNames() []string {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}

	return names
}

// Count 取得工具數量
func (r *ToolRegistry) Count() int {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	return len(r.tools)
}

// Clear 清空所有工具
func (r *ToolRegistry) Clear() {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.tools = make(map[string]*ToolAdapter)
	r.logger.Debug("All tools cleared")
}

// GetToolsByCategory 根據分類取得工具
func (r *ToolRegistry) GetToolsByCategory(category string) []*ToolAdapter {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	var adapters []*ToolAdapter
	for _, adapter := range r.tools {
		if adapter.GetCategory() == category {
			adapters = append(adapters, adapter)
		}
	}

	return adapters
}

// GetCategories 取得所有分類
func (r *ToolRegistry) GetCategories() []string {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	categorySet := make(map[string]bool)
	for _, adapter := range r.tools {
		category := adapter.GetCategory()
		if category != "" {
			categorySet[category] = true
		}
	}

	categories := make([]string, 0, len(categorySet))
	for category := range categorySet {
		categories = append(categories, category)
	}

	return categories
}
