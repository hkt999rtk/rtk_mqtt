package types

import (
	"context"
	"time"
)

// LLMTool represents a diagnostic tool that can be executed by LLM
type LLMTool interface {
	// Name returns the tool name (e.g., "topology.get_full")
	Name() string

	// Category returns the tool category (Read, Test, Act)
	Category() ToolCategory

	// Execute runs the tool with given parameters
	Execute(ctx context.Context, params map[string]interface{}) (*ToolResult, error)

	// Validate checks if the parameters are valid
	Validate(params map[string]interface{}) error

	// RequiredCapabilities returns the required device capabilities
	RequiredCapabilities() []string

	// Description returns a brief description of what the tool does
	Description() string
}

// ToolCategory represents the type of diagnostic tool
type ToolCategory string

const (
	ToolCategoryRead ToolCategory = "Read"
	ToolCategoryTest ToolCategory = "Test"
	ToolCategoryAct  ToolCategory = "Act"
	ToolCategoryWiFi ToolCategory = "WiFi"
)

// ToolResult represents the result of a tool execution
type ToolResult struct {
	// ToolName is the name of the executed tool
	ToolName string `json:"tool_name"`

	// Success indicates if the tool execution was successful
	Success bool `json:"success"`

	// Data contains the tool-specific result data
	Data interface{} `json:"data"`

	// Error contains error information if execution failed
	Error string `json:"error,omitempty"`

	// ExecutionTime is how long the tool took to execute
	ExecutionTime time.Duration `json:"execution_time"`

	// Timestamp is when the tool was executed
	Timestamp time.Time `json:"timestamp"`

	// SessionID links this result to a diagnosis session
	SessionID string `json:"session_id,omitempty"`

	// TraceID for distributed tracing
	TraceID string `json:"trace_id,omitempty"`
}

// LLMSession represents an LLM diagnosis session
type LLMSession struct {
	// SessionID is the unique identifier for the session
	SessionID string `json:"session_id"`

	// TraceID for distributed tracing
	TraceID string `json:"trace_id"`

	// CreatedAt is when the session was created
	CreatedAt time.Time `json:"created_at"`

	// UpdatedAt is when the session was last updated
	UpdatedAt time.Time `json:"updated_at"`

	// Status represents the current session status
	Status LLMSessionStatus `json:"status"`

	// ToolCalls contains the sequence of tool calls in this session
	ToolCalls []ToolCall `json:"tool_calls"`

	// DeviceID is the target device for this session (if applicable)
	DeviceID string `json:"device_id,omitempty"`

	// UserID is the user who initiated this session
	UserID string `json:"user_id,omitempty"`

	// Metadata contains additional session information
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// LLMSessionStatus represents the status of an LLM session
type LLMSessionStatus string

const (
	LLMSessionStatusActive    LLMSessionStatus = "active"
	LLMSessionStatusCompleted LLMSessionStatus = "completed"
	LLMSessionStatusFailed    LLMSessionStatus = "failed"
	LLMSessionStatusCancelled LLMSessionStatus = "cancelled"
)

// ToolCall represents a single tool execution within a session
type ToolCall struct {
	// ID is the unique identifier for this tool call
	ID string `json:"id"`

	// ToolName is the name of the called tool
	ToolName string `json:"tool_name"`

	// Parameters are the parameters passed to the tool
	Parameters map[string]interface{} `json:"parameters"`

	// Result is the result of the tool execution
	Result *ToolResult `json:"result,omitempty"`

	// StartedAt is when the tool call started
	StartedAt time.Time `json:"started_at"`

	// CompletedAt is when the tool call completed
	CompletedAt *time.Time `json:"completed_at,omitempty"`

	// Status represents the current status of the tool call
	Status ToolCallStatus `json:"status"`
}

// ToolCallStatus represents the status of a tool call
type ToolCallStatus string

const (
	ToolCallStatusPending   ToolCallStatus = "pending"
	ToolCallStatusRunning   ToolCallStatus = "running"
	ToolCallStatusCompleted ToolCallStatus = "completed"
	ToolCallStatusFailed    ToolCallStatus = "failed"
	ToolCallStatusCancelled ToolCallStatus = "cancelled"
)

// DeviceCapabilities represents the capabilities of a device
type DeviceCapabilities struct {
	// DeviceID is the unique identifier for the device
	DeviceID string `json:"device_id"`

	// SupportedTools is the list of tools this device supports
	SupportedTools []string `json:"supported_tools"`

	// Capabilities contains device-specific capability information
	Capabilities map[string]interface{} `json:"capabilities"`

	// LastUpdated is when these capabilities were last updated
	LastUpdated time.Time `json:"last_updated"`

	// Version is the version of the capability information
	Version string `json:"version"`
}

// ToolError represents an error that occurred during tool execution
type ToolError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

func (e *ToolError) Error() string {
	if e.Details != "" {
		return e.Message + ": " + e.Details
	}
	return e.Message
}

// Common tool error codes
const (
	ToolErrorInvalidParameters = "INVALID_PARAMETERS"
	ToolErrorDeviceOffline     = "DEVICE_OFFLINE"
	ToolErrorTimeout           = "TIMEOUT"
	ToolErrorUnknownTool       = "UNKNOWN_TOOL"
	ToolErrorExecutionFailed   = "EXECUTION_FAILED"
	ToolErrorPermissionDenied  = "PERMISSION_DENIED"
)
