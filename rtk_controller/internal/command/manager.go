package command

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"rtk_controller/internal/mqtt"
	"rtk_controller/internal/storage"
	"rtk_controller/pkg/types"
	"rtk_controller/pkg/utils"

	log "github.com/sirupsen/logrus"
)

// Manager handles command processing and response tracking
type Manager struct {
	mqttClient *mqtt.Client
	storage    storage.Storage
	
	// Command tracking
	pendingCommands map[string]*types.DeviceCommand
	mu              sync.RWMutex
	
	// Background workers
	ctx    context.Context
	cancel context.CancelFunc
	done   chan struct{}
	
	// Statistics
	stats *CommandStats
}

// CommandStats represents command statistics
type CommandStats struct {
	TotalCommands     int            `json:"total_commands"`
	PendingCommands   int            `json:"pending_commands"`
	CompletedCommands int            `json:"completed_commands"`
	FailedCommands    int            `json:"failed_commands"`
	TimeoutCommands   int            `json:"timeout_commands"`
	StatusStats       map[string]int `json:"status_stats"`
	LastUpdated       time.Time      `json:"last_updated"`
}

// NewManager creates a new command manager
func NewManager(mqttClient *mqtt.Client, storage storage.Storage) *Manager {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &Manager{
		mqttClient:      mqttClient,
		storage:         storage,
		pendingCommands: make(map[string]*types.DeviceCommand),
		ctx:             ctx,
		cancel:          cancel,
		done:            make(chan struct{}),
		stats:           &CommandStats{StatusStats: make(map[string]int)},
	}
}

// Start starts the command manager
func (m *Manager) Start(ctx context.Context) error {
	log.Info("Starting command manager...")
	
	// Register MQTT handlers for command responses
	m.mqttClient.RegisterHandler("rtk/v1/+/+/+/cmd/ack", &CommandAckHandler{manager: m})
	m.mqttClient.RegisterHandler("rtk/v1/+/+/+/cmd/res", &CommandResultHandler{manager: m})
	
	// Load pending commands from storage
	if err := m.loadPendingCommands(); err != nil {
		log.WithError(err).Warn("Failed to load pending commands from storage")
	}
	
	// Start background workers
	go m.timeoutWorker()
	go m.statsWorker()
	
	log.WithField("pending_commands", len(m.pendingCommands)).Info("Command manager started")
	return nil
}

// Stop stops the command manager
func (m *Manager) Stop() {
	log.Info("Stopping command manager...")
	
	m.cancel()
	<-m.done
	
	// Save pending commands to storage
	if err := m.savePendingCommands(); err != nil {
		log.WithError(err).Error("Failed to save pending commands to storage")
	}
	
	log.Info("Command manager stopped")
}

// SendCommand sends a command to a device
func (m *Manager) SendCommand(tenant, site, deviceID, operation string, args map[string]interface{}, timeoutSeconds int) (*types.DeviceCommand, error) {
	// Create command
	command := &types.DeviceCommand{
		ID:        utils.GenerateMessageID(),
		DeviceID:  fmt.Sprintf("%s:%s:%s", tenant, site, deviceID),
		Operation: operation,
		Args:      args,
		TimeoutMS: int64(timeoutSeconds * 1000),
		Status:    "pending",
		CreatedAt: time.Now(),
	}
	
	// Store command
	if err := m.storeCommand(command); err != nil {
		return nil, fmt.Errorf("failed to store command: %w", err)
	}
	
	// Add to pending commands
	m.mu.Lock()
	m.pendingCommands[command.ID] = command
	m.mu.Unlock()
	
	// Build MQTT command payload
	payload := map[string]interface{}{
		"id":         command.ID,
		"op":         operation,
		"schema":     fmt.Sprintf("cmd.%s/1.0", operation),
		"args":       args,
		"timeout_ms": command.TimeoutMS,
		"expect":     "result",
		"reply_to":   nil,
		"ts":         time.Now().UnixMilli(),
	}
	
	// Publish command
	topic := utils.BuildTopic(tenant, site, deviceID, "cmd", "req")
	if err := m.mqttClient.Publish(topic, 1, false, payload); err != nil {
		// Remove from pending commands on publish failure
		m.mu.Lock()
		delete(m.pendingCommands, command.ID)
		m.mu.Unlock()
		
		// Update command status
		command.Status = "failed"
		command.Error = fmt.Sprintf("Failed to publish command: %v", err)
		m.storeCommand(command)
		
		return nil, fmt.Errorf("failed to publish command: %w", err)
	}
	
	// Update command status and timestamp
	command.Status = "sent"
	now := time.Now()
	command.SentAt = &now
	m.storeCommand(command)
	
	log.WithFields(log.Fields{
		"command_id": command.ID,
		"device_id":  deviceID,
		"operation":  operation,
		"topic":      topic,
	}).Info("Command sent to device")
	
	return command, nil
}

// GetCommand returns a command by ID
func (m *Manager) GetCommand(commandID string) (*types.DeviceCommand, error) {
	// Check pending commands first
	m.mu.RLock()
	if cmd, exists := m.pendingCommands[commandID]; exists {
		m.mu.RUnlock()
		// Return a copy
		cmdCopy := *cmd
		return &cmdCopy, nil
	}
	m.mu.RUnlock()
	
	// Load from storage
	var command types.DeviceCommand
	err := m.storage.View(func(tx storage.Transaction) error {
		key := fmt.Sprintf("command:%s", commandID)
		value, err := tx.Get(key)
		if err != nil {
			return err
		}
		
		return json.Unmarshal([]byte(value), &command)
	})
	
	if err != nil {
		return nil, fmt.Errorf("command not found: %s", commandID)
	}
	
	return &command, nil
}

// ListCommands returns commands matching the filter
func (m *Manager) ListCommands(deviceID string, status string, limit int, offset int) ([]*types.DeviceCommand, int, error) {
	var commands []*types.DeviceCommand
	var total int
	
	err := m.storage.View(func(tx storage.Transaction) error {
		return tx.IteratePrefix("command:", func(key, value string) error {
			var command types.DeviceCommand
			if err := json.Unmarshal([]byte(value), &command); err != nil {
				log.WithError(err).Warn("Failed to unmarshal command")
				return nil // Continue iteration
			}
			
			// Apply filters
			if deviceID != "" && command.DeviceID != deviceID {
				return nil
			}
			
			if status != "" && command.Status != status {
				return nil
			}
			
			total++
			
			// Apply pagination
			if total > offset && (limit <= 0 || len(commands) < limit) {
				commands = append(commands, &command)
			}
			
			return nil
		})
	})
	
	return commands, total, err
}

// GetStats returns command statistics
func (m *Manager) GetStats() *CommandStats {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	// Return a copy of current stats
	stats := *m.stats
	return &stats
}

// HandleCommandAck handles command acknowledgment
func (m *Manager) HandleCommandAck(topic string, payload []byte) error {
	// Parse ACK payload
	var ackData map[string]interface{}
	if err := json.Unmarshal(payload, &ackData); err != nil {
		return fmt.Errorf("failed to parse ACK payload: %w", err)
	}
	
	commandID, ok := ackData["id"].(string)
	if !ok {
		return fmt.Errorf("missing or invalid command ID in ACK")
	}
	
	m.mu.Lock()
	command, exists := m.pendingCommands[commandID]
	if !exists {
		m.mu.Unlock()
		return fmt.Errorf("command not found: %s", commandID)
	}
	
	// Update command status
	command.Status = "ack"
	m.mu.Unlock()
	
	// Store updated command
	if err := m.storeCommand(command); err != nil {
		log.WithError(err).Error("Failed to store command ACK")
	}
	
	log.WithFields(log.Fields{
		"command_id": commandID,
		"device_id":  command.DeviceID,
	}).Info("Command acknowledged by device")
	
	return nil
}

// HandleCommandResult handles command result
func (m *Manager) HandleCommandResult(topic string, payload []byte) error {
	// Parse result payload
	var resultData map[string]interface{}
	if err := json.Unmarshal(payload, &resultData); err != nil {
		return fmt.Errorf("failed to parse result payload: %w", err)
	}
	
	commandID, ok := resultData["id"].(string)
	if !ok {
		return fmt.Errorf("missing or invalid command ID in result")
	}
	
	m.mu.Lock()
	command, exists := m.pendingCommands[commandID]
	if !exists {
		m.mu.Unlock()
		return fmt.Errorf("command not found: %s", commandID)
	}
	
	// Remove from pending commands
	delete(m.pendingCommands, commandID)
	m.mu.Unlock()
	
	// Update command with result
	command.Status = "completed"
	now := time.Now()
	command.CompletedAt = &now
	command.Result = resultData
	
	// Check for error in result
	if errorMsg, ok := resultData["error"].(string); ok && errorMsg != "" {
		command.Status = "failed"
		command.Error = errorMsg
	}
	
	// Store updated command
	if err := m.storeCommand(command); err != nil {
		log.WithError(err).Error("Failed to store command result")
	}
	
	log.WithFields(log.Fields{
		"command_id": commandID,
		"device_id":  command.DeviceID,
		"status":     command.Status,
	}).Info("Command completed")
	
	return nil
}

// storeCommand stores command in database
func (m *Manager) storeCommand(command *types.DeviceCommand) error {
	return m.storage.Transaction(func(tx storage.Transaction) error {
		key := fmt.Sprintf("command:%s", command.ID)
		data, err := json.Marshal(command)
		if err != nil {
			return fmt.Errorf("failed to marshal command: %w", err)
		}
		
		return tx.Set(key, string(data))
	})
}

// timeoutWorker checks for command timeouts
func (m *Manager) timeoutWorker() {
	defer close(m.done)
	
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			m.checkTimeouts()
		}
	}
}

// checkTimeouts checks and handles command timeouts
func (m *Manager) checkTimeouts() {
	now := time.Now()
	var timedOutCommands []*types.DeviceCommand
	
	m.mu.Lock()
	for commandID, command := range m.pendingCommands {
		// Check if command has timed out
		timeoutDuration := time.Duration(command.TimeoutMS) * time.Millisecond
		if command.SentAt != nil && now.Sub(*command.SentAt) > timeoutDuration {
			timedOutCommands = append(timedOutCommands, command)
			delete(m.pendingCommands, commandID)
		}
	}
	m.mu.Unlock()
	
	// Handle timed out commands
	for _, command := range timedOutCommands {
		command.Status = "timeout"
		command.Error = "Command execution timeout"
		now := time.Now()
		command.CompletedAt = &now
		
		if err := m.storeCommand(command); err != nil {
			log.WithError(err).Error("Failed to store timeout command")
		}
		
		log.WithFields(log.Fields{
			"command_id": command.ID,
			"device_id":  command.DeviceID,
			"operation":  command.Operation,
		}).Warn("Command timed out")
	}
}

// statsWorker updates command statistics periodically
func (m *Manager) statsWorker() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			m.updateStats()
		}
	}
}

// updateStats calculates current command statistics
func (m *Manager) updateStats() {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	stats := &CommandStats{
		StatusStats: make(map[string]int),
		LastUpdated: time.Now(),
	}
	
	// Count pending commands
	stats.PendingCommands = len(m.pendingCommands)
	
	// Count all commands from storage
	err := m.storage.View(func(tx storage.Transaction) error {
		return tx.IteratePrefix("command:", func(key, value string) error {
			var command types.DeviceCommand
			if err := json.Unmarshal([]byte(value), &command); err != nil {
				return nil // Continue on error
			}
			
			stats.TotalCommands++
			stats.StatusStats[command.Status]++
			
			switch command.Status {
			case "completed":
				stats.CompletedCommands++
			case "failed":
				stats.FailedCommands++
			case "timeout":
				stats.TimeoutCommands++
			}
			
			return nil
		})
	})
	
	if err != nil {
		log.WithError(err).Error("Failed to update command statistics")
		return
	}
	
	m.stats = stats
}

// loadPendingCommands loads pending commands from storage
func (m *Manager) loadPendingCommands() error {
	return m.storage.View(func(tx storage.Transaction) error {
		return tx.IteratePrefix("command:", func(key, value string) error {
			var command types.DeviceCommand
			if err := json.Unmarshal([]byte(value), &command); err != nil {
				log.WithError(err).Warn("Failed to unmarshal command")
				return nil // Continue iteration
			}
			
			// Only load commands that are still pending execution
			if command.Status == "pending" || command.Status == "sent" || command.Status == "ack" {
				m.pendingCommands[command.ID] = &command
			}
			
			return nil
		})
	})
}

// savePendingCommands saves pending commands to storage
func (m *Manager) savePendingCommands() error {
	for _, command := range m.pendingCommands {
		if err := m.storeCommand(command); err != nil {
			log.WithError(err).Errorf("Failed to save pending command: %s", command.ID)
		}
	}
	return nil
}

// CommandAckHandler handles command acknowledgment messages
type CommandAckHandler struct {
	manager *Manager
}

func (h *CommandAckHandler) HandleMessage(topic string, payload []byte) error {
	return h.manager.HandleCommandAck(topic, payload)
}

// CommandResultHandler handles command result messages
type CommandResultHandler struct {
	manager *Manager
}

func (h *CommandResultHandler) HandleMessage(topic string, payload []byte) error {
	return h.manager.HandleCommandResult(topic, payload)
}