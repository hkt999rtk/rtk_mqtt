package types

import (
	"time"
)

// Changeset represents a collection of changes that can be applied or rolled back together
type Changeset struct {
	// ID is the unique identifier for the changeset
	ID string `json:"id"`
	
	// Description describes what this changeset does
	Description string `json:"description"`
	
	// Commands are the commands that make up this changeset
	Commands []*Command `json:"commands"`
	
	// Status represents the current status of the changeset
	Status ChangesetStatus `json:"status"`
	
	// CreatedAt is when the changeset was created
	CreatedAt time.Time `json:"created_at"`
	
	// ExecutedAt is when the changeset was executed (if applicable)
	ExecutedAt *time.Time `json:"executed_at,omitempty"`
	
	// RolledBackAt is when the changeset was rolled back (if applicable)
	RolledBackAt *time.Time `json:"rolled_back_at,omitempty"`
	
	// CreatedBy indicates who created this changeset
	CreatedBy string `json:"created_by,omitempty"`
	
	// SessionID links this changeset to an LLM session
	SessionID string `json:"session_id,omitempty"`
	
	// TraceID for distributed tracing
	TraceID string `json:"trace_id,omitempty"`
	
	// Results contains the results of executing the commands
	Results []*CommandResult `json:"results,omitempty"`
	
	// RollbackCommands are the commands needed to rollback this changeset
	RollbackCommands []*Command `json:"rollback_commands,omitempty"`
	
	// Metadata contains additional changeset information
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// ChangesetStatus represents the status of a changeset
type ChangesetStatus string

const (
	ChangesetStatusDraft      ChangesetStatus = "draft"      // Created but not executed
	ChangesetStatusPending    ChangesetStatus = "pending"    // Waiting to be executed
	ChangesetStatusExecuting  ChangesetStatus = "executing"  // Currently being executed
	ChangesetStatusCompleted  ChangesetStatus = "completed"  // Successfully executed
	ChangesetStatusFailed     ChangesetStatus = "failed"     // Execution failed
	ChangesetStatusRolledBack ChangesetStatus = "rolled_back" // Successfully rolled back
	ChangesetStatusRollbackFailed ChangesetStatus = "rollback_failed" // Rollback failed
)

// CommandResult represents the result of executing a command within a changeset
type CommandResult struct {
	// CommandID is the ID of the executed command
	CommandID string `json:"command_id"`
	
	// Success indicates if the command was executed successfully
	Success bool `json:"success"`
	
	// Message contains success or error message
	Message string `json:"message"`
	
	// Data contains command-specific result data
	Data interface{} `json:"data,omitempty"`
	
	// ExecutedAt is when this command was executed
	ExecutedAt time.Time `json:"executed_at"`
	
	// Duration is how long the command took to execute
	Duration time.Duration `json:"duration"`
}

// ChangesetOptions contains options for creating a changeset
type ChangesetOptions struct {
	Description string
	CreatedBy   string
	SessionID   string
	TraceID     string
	Metadata    map[string]interface{}
}

// ChangesetSummary provides a summary view of a changeset
type ChangesetSummary struct {
	ID              string          `json:"id"`
	Description     string          `json:"description"`
	Status          ChangesetStatus `json:"status"`
	CommandCount    int             `json:"command_count"`
	SuccessCount    int             `json:"success_count"`
	FailureCount    int             `json:"failure_count"`
	CreatedAt       time.Time       `json:"created_at"`
	ExecutedAt      *time.Time      `json:"executed_at,omitempty"`
	RolledBackAt    *time.Time      `json:"rolled_back_at,omitempty"`
	CreatedBy       string          `json:"created_by,omitempty"`
	SessionID       string          `json:"session_id,omitempty"`
	ExecutionTime   time.Duration   `json:"execution_time"`
}

// IsExecuted returns true if the changeset has been executed
func (c *Changeset) IsExecuted() bool {
	return c.Status == ChangesetStatusCompleted || c.Status == ChangesetStatusFailed
}

// IsRollbackable returns true if the changeset can be rolled back
func (c *Changeset) IsRollbackable() bool {
	return c.Status == ChangesetStatusCompleted && len(c.RollbackCommands) > 0
}

// GetSummary returns a summary of the changeset
func (c *Changeset) GetSummary() *ChangesetSummary {
	summary := &ChangesetSummary{
		ID:           c.ID,
		Description:  c.Description,
		Status:       c.Status,
		CommandCount: len(c.Commands),
		CreatedAt:    c.CreatedAt,
		ExecutedAt:   c.ExecutedAt,
		RolledBackAt: c.RolledBackAt,
		CreatedBy:    c.CreatedBy,
		SessionID:    c.SessionID,
	}
	
	// Count successful and failed results
	for _, result := range c.Results {
		if result.Success {
			summary.SuccessCount++
		} else {
			summary.FailureCount++
		}
	}
	
	// Calculate execution time
	if c.ExecutedAt != nil {
		summary.ExecutionTime = c.ExecutedAt.Sub(c.CreatedAt)
	}
	
	return summary
}

// AddCommand adds a command to the changeset
func (c *Changeset) AddCommand(cmd *Command) {
	if c.Commands == nil {
		c.Commands = make([]*Command, 0)
	}
	c.Commands = append(c.Commands, cmd)
}

// AddRollbackCommand adds a rollback command to the changeset
func (c *Changeset) AddRollbackCommand(cmd *Command) {
	if c.RollbackCommands == nil {
		c.RollbackCommands = make([]*Command, 0)
	}
	c.RollbackCommands = append(c.RollbackCommands, cmd)
}

// AddResult adds a command execution result to the changeset
func (c *Changeset) AddResult(result *CommandResult) {
	if c.Results == nil {
		c.Results = make([]*CommandResult, 0)
	}
	c.Results = append(c.Results, result)
}

// ChangesetFilter represents filters for querying changesets
type ChangesetFilter struct {
	Status     []ChangesetStatus `json:"status,omitempty"`
	CreatedBy  string            `json:"created_by,omitempty"`
	SessionID  string            `json:"session_id,omitempty"`
	StartTime  *time.Time        `json:"start_time,omitempty"`
	EndTime    *time.Time        `json:"end_time,omitempty"`
}