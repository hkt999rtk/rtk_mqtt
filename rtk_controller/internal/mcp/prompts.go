package mcp

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/sirupsen/logrus"
)

// PromptRegistry 提示範本註冊表
type PromptRegistry struct {
	templates map[string]*PromptTemplate
	mutex     sync.RWMutex
	logger    *logrus.Logger
}

// PromptTemplate 提示範本
type PromptTemplate struct {
	Name        string              `json:"name"`
	Description string              `json:"description"`
	Template    string              `json:"template"`
	Arguments   []MCPPromptArgument `json:"arguments"`
}

// MCPPromptArgument MCP 提示參數
type MCPPromptArgument struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
}

// MCPPrompt MCP 提示定義
type MCPPrompt struct {
	Name        string              `json:"name"`
	Description string              `json:"description"`
	Arguments   []MCPPromptArgument `json:"arguments"`
}

// MCPPromptMessage MCP 提示訊息
type MCPPromptMessage struct {
	Role    string            `json:"role"`
	Content MCPMessageContent `json:"content"`
}

// MCPMessageContent MCP 訊息內容
type MCPMessageContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// MCPGetPromptResult MCP 取得提示結果
type MCPGetPromptResult struct {
	Description string             `json:"description"`
	Messages    []MCPPromptMessage `json:"messages"`
}

// MCPListPromptsResult MCP 列出提示結果
type MCPListPromptsResult struct {
	Prompts []MCPPrompt `json:"prompts"`
}

// MCPGetPromptRequest MCP 取得提示請求
type MCPGetPromptRequest struct {
	Params MCPGetPromptParams `json:"params"`
}

// MCPGetPromptParams MCP 取得提示參數
type MCPGetPromptParams struct {
	Name      string            `json:"name"`
	Arguments map[string]string `json:"arguments,omitempty"`
}

// NewPromptRegistry 建立提示註冊表
func NewPromptRegistry(logger *logrus.Logger) *PromptRegistry {
	return &PromptRegistry{
		templates: make(map[string]*PromptTemplate),
		logger:    logger,
	}
}

// Register 註冊提示範本
func (pr *PromptRegistry) Register(template *PromptTemplate) error {
	if template == nil {
		return fmt.Errorf("template cannot be nil")
	}

	if template.Name == "" {
		return fmt.Errorf("template name cannot be empty")
	}

	pr.mutex.Lock()
	defer pr.mutex.Unlock()

	if _, exists := pr.templates[template.Name]; exists {
		return fmt.Errorf("template %s already registered", template.Name)
	}

	pr.templates[template.Name] = template
	pr.logger.WithField("template", template.Name).Debug("Prompt template registered")

	return nil
}

// Unregister 取消註冊提示範本
func (pr *PromptRegistry) Unregister(name string) error {
	pr.mutex.Lock()
	defer pr.mutex.Unlock()

	if _, exists := pr.templates[name]; !exists {
		return fmt.Errorf("template %s not found", name)
	}

	delete(pr.templates, name)
	pr.logger.WithField("template", name).Debug("Prompt template unregistered")

	return nil
}

// ListPrompts 列出所有提示
func (pr *PromptRegistry) ListPrompts(ctx context.Context) ([]MCPPrompt, error) {
	pr.mutex.RLock()
	defer pr.mutex.RUnlock()

	prompts := make([]MCPPrompt, 0, len(pr.templates))

	for _, template := range pr.templates {
		prompts = append(prompts, MCPPrompt{
			Name:        template.Name,
			Description: template.Description,
			Arguments:   template.Arguments,
		})
	}

	return prompts, nil
}

// GetPrompt 取得提示內容
func (pr *PromptRegistry) GetPrompt(ctx context.Context, name string, arguments map[string]string) (*MCPGetPromptResult, error) {
	pr.mutex.RLock()
	defer pr.mutex.RUnlock()

	template, exists := pr.templates[name]
	if !exists {
		return nil, fmt.Errorf("prompt template '%s' not found", name)
	}

	// 替換範本變數
	content := pr.replaceTemplateVariables(template.Template, arguments)

	return &MCPGetPromptResult{
		Description: template.Description,
		Messages: []MCPPromptMessage{
			{
				Role: "user",
				Content: MCPMessageContent{
					Type: "text",
					Text: content,
				},
			},
		},
	}, nil
}

// replaceTemplateVariables 替換範本變數
func (pr *PromptRegistry) replaceTemplateVariables(template string, arguments map[string]string) string {
	content := template

	for argName, argValue := range arguments {
		placeholder := fmt.Sprintf("{{%s}}", argName)
		content = strings.ReplaceAll(content, placeholder, argValue)
	}

	return content
}

// RegisterBuiltInPrompts 註冊內建提示範本
func (pr *PromptRegistry) RegisterBuiltInPrompts() error {
	builtInPrompts := []*PromptTemplate{
		{
			Name:        "intent_classification",
			Description: "Classify user's network problem intent",
			Template: `You are a home network diagnostic expert. Based on the user's description, classify their problem into one of these intents:

A. no_internet - Complete loss of internet connectivity
B. slow_speed - Internet speed slower than expected  
C. unstable_disconnect - Intermittent disconnections
D. weak_signal_coverage - Poor WiFi signal in certain areas
E. realtime_latency - High latency affecting real-time applications
F. device_specific_issue - Specific device connection problems
G. roaming_issue - Devices not switching to optimal access points
H. mesh_backhaul_issue - Mesh network backhaul problems
I. dhcp_dns_issue_advanced - DHCP/DNS configuration issues

User description: "{{user_input}}"

Respond with:
1. Primary intent: [letter and name]
2. Confidence: [0-100]%
3. Reasoning: [brief explanation]
4. Suggested diagnostic tools: [list of tools]`,
			Arguments: []MCPPromptArgument{
				{
					Name:        "user_input",
					Description: "User's description of the network problem",
					Required:    true,
				},
			},
		},
		{
			Name:        "diagnostic_report",
			Description: "Generate comprehensive diagnostic report",
			Template: `Based on the diagnostic tool results, generate a comprehensive network diagnostic report.

Tool Results:
{{tool_results}}

Network Context:
{{network_context}}

Please provide:

## 🔍 FINDINGS
- List key findings from the diagnostic data
- Highlight any anomalies or concerning metrics

## 🎯 ROOT CAUSE ANALYSIS  
- Identify the most likely root cause(s)
- Explain the technical reasoning

## 💡 RECOMMENDATIONS
### Immediate Actions:
- Steps that can be taken right now

### Configuration Changes:
- Specific settings to modify (with dry-run commands)

### Long-term Improvements:
- Infrastructure or equipment recommendations

## 📊 TECHNICAL DETAILS
- Include relevant metrics and measurements
- Reference specific tool outputs

## 🔄 FOLLOW-UP
- Suggested monitoring or re-testing steps
- Timeline for reassessment`,
			Arguments: []MCPPromptArgument{
				{
					Name:        "tool_results",
					Description: "JSON results from diagnostic tools",
					Required:    true,
				},
				{
					Name:        "network_context",
					Description: "Additional network context information",
					Required:    false,
				},
			},
		},
		{
			Name:        "troubleshooting_guide",
			Description: "Generate step-by-step troubleshooting guide",
			Template: `Create a step-by-step troubleshooting guide for the following network issue:

Problem: {{problem_description}}
Intent: {{intent}}
Current Status: {{current_status}}

Generate a detailed troubleshooting guide with:

## 🔧 IMMEDIATE CHECKS
1. [List immediate verification steps]
2. [Quick fixes that can be tried]

## 🔍 DIAGNOSTIC STEPS  
1. [Systematic diagnostic procedures]
2. [Tools to run and what to look for]

## ⚠️ COMMON ISSUES
- [List common causes for this type of problem]
- [How to identify each cause]

## 🛠️ SOLUTIONS
### Basic Solutions:
- [Simple fixes for common causes]

### Advanced Solutions:
- [More complex troubleshooting steps]

## 📞 WHEN TO ESCALATE
- [Conditions that require professional help]
- [Information to gather before escalating]

Make the guide suitable for {{user_level}} users.`,
			Arguments: []MCPPromptArgument{
				{
					Name:        "problem_description",
					Description: "Description of the network problem",
					Required:    true,
				},
				{
					Name:        "intent",
					Description: "Classified problem intent",
					Required:    true,
				},
				{
					Name:        "current_status",
					Description: "Current network status information",
					Required:    false,
				},
				{
					Name:        "user_level",
					Description: "User technical level (beginner/intermediate/advanced)",
					Required:    false,
				},
			},
		},
		{
			Name:        "wifi_optimization",
			Description: "Generate WiFi optimization recommendations",
			Template: `Based on the WiFi diagnostic results, provide optimization recommendations.

WiFi Status:
{{wifi_status}}

Signal Analysis:
{{signal_analysis}}

Interference Data:
{{interference_data}}

Current Configuration:
{{current_config}}

## 📶 SIGNAL OPTIMIZATION
### Current Signal Quality:
- [Analyze current signal strength and coverage]

### Optimization Recommendations:
- [Specific recommendations for signal improvement]

## 🔄 CHANNEL OPTIMIZATION
### Current Channel Usage:
- [Analyze current channel assignments]

### Recommended Changes:
- [Specific channel recommendations with reasoning]

## ⚙️ CONFIGURATION RECOMMENDATIONS
### Power Settings:
- [Transmit power optimization recommendations]

### Advanced Settings:
- [802.11r/k/v, band steering, etc.]

## 🏠 PHYSICAL PLACEMENT
- [Access point placement recommendations]
- [Coverage area optimization]

## 📊 EXPECTED IMPROVEMENTS
- [Quantify expected performance gains]
- [Before/after comparison predictions]`,
			Arguments: []MCPPromptArgument{
				{
					Name:        "wifi_status",
					Description: "Current WiFi status information",
					Required:    true,
				},
				{
					Name:        "signal_analysis",
					Description: "WiFi signal analysis results",
					Required:    false,
				},
				{
					Name:        "interference_data",
					Description: "Interference analysis data",
					Required:    false,
				},
				{
					Name:        "current_config",
					Description: "Current WiFi configuration",
					Required:    false,
				},
			},
		},
		{
			Name:        "network_summary",
			Description: "Generate network health summary",
			Template: `Generate a comprehensive network health summary.

Topology Data:
{{topology_data}}

Performance Metrics:
{{performance_metrics}}

Device Status:
{{device_status}}

Recent Issues:
{{recent_issues}}

## 📊 NETWORK HEALTH OVERVIEW
### Overall Status: [Excellent/Good/Fair/Poor]
- [Brief summary of overall network health]

### Key Metrics:
- [List important performance indicators]

## 🏢 NETWORK INFRASTRUCTURE
### Devices Summary:
- [Count and status of network devices]

### Connectivity:
- [Network topology and connection quality]

## 📈 PERFORMANCE ANALYSIS
### Speed and Latency:
- [Internet and internal network performance]

### WiFi Performance:
- [Wireless network quality and coverage]

## ⚠️ IDENTIFIED ISSUES
### Active Issues:
- [Current problems requiring attention]

### Potential Concerns:
- [Issues that may develop into problems]

## 🔮 RECOMMENDATIONS
### Immediate Actions:
- [Steps to take now]

### Preventive Measures:
- [Steps to prevent future issues]

### Upgrade Suggestions:
- [Infrastructure improvements to consider]`,
			Arguments: []MCPPromptArgument{
				{
					Name:        "topology_data",
					Description: "Network topology information",
					Required:    true,
				},
				{
					Name:        "performance_metrics",
					Description: "Network performance data",
					Required:    false,
				},
				{
					Name:        "device_status",
					Description: "Status of network devices",
					Required:    false,
				},
				{
					Name:        "recent_issues",
					Description: "Recent network issues or problems",
					Required:    false,
				},
			},
		},
	}

	// 註冊所有內建提示
	for _, template := range builtInPrompts {
		if err := pr.Register(template); err != nil {
			pr.logger.WithFields(logrus.Fields{
				"template": template.Name,
				"error":    err,
			}).Warning("Failed to register built-in prompt template")
			continue
		}
	}

	pr.logger.WithField("count", len(builtInPrompts)).Info("Built-in prompt templates registered")
	return nil
}

// GetTemplateCount 取得範本數量
func (pr *PromptRegistry) GetTemplateCount() int {
	pr.mutex.RLock()
	defer pr.mutex.RUnlock()

	return len(pr.templates)
}

// GetTemplateNames 取得所有範本名稱
func (pr *PromptRegistry) GetTemplateNames() []string {
	pr.mutex.RLock()
	defer pr.mutex.RUnlock()

	names := make([]string, 0, len(pr.templates))
	for name := range pr.templates {
		names = append(names, name)
	}

	return names
}
