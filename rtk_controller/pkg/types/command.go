package types

import (
	"time"
)

// CommandStatus represents the status of a command
type CommandStatus string

const (
	CommandStatusPending   CommandStatus = "pending"
	CommandStatusSent      CommandStatus = "sent"
	CommandStatusAcked     CommandStatus = "ack"
	CommandStatusCompleted CommandStatus = "completed"
	CommandStatusFailed    CommandStatus = "failed"
	CommandStatusTimeout   CommandStatus = "timeout"
	CommandStatusCancelled CommandStatus = "cancelled"
)

// Command represents a command sent to a device
type Command struct {
	ID          string                 `json:"id"`
	DeviceID    string                 `json:"device_id"`
	Operation   string                 `json:"operation"`
	Args        map[string]interface{} `json:"args"`
	Timeout     time.Duration          `json:"timeout"`
	Status      CommandStatus          `json:"status"`
	Result      map[string]interface{} `json:"result,omitempty"`
	Error       *string                `json:"error,omitempty"`
	Progress    map[string]interface{} `json:"progress,omitempty"`
	Expectation string                 `json:"expectation"` // ack, result, none
	CreatedAt   time.Time              `json:"created_at"`
	SentAt      *time.Time             `json:"sent_at,omitempty"`
	AckedAt     *time.Time             `json:"acked_at,omitempty"`
	CompletedAt *time.Time             `json:"completed_at,omitempty"`
}

// CommandFilter represents filtering criteria for commands
type CommandFilter struct {
	DeviceID string `json:"device_id,omitempty"`
	Status   string `json:"status,omitempty"`
	Limit    int    `json:"limit,omitempty"`
}

// CommandStats represents command statistics
type CommandStats struct {
	TotalCommands     int       `json:"total_commands"`
	PendingCommands   int       `json:"pending_commands"`
	CompletedCommands int       `json:"completed_commands"`
	FailedCommands    int       `json:"failed_commands"`
	LastUpdated       time.Time `json:"last_updated"`
}
