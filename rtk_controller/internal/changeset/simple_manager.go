package changeset

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"rtk_controller/internal/command"
	"rtk_controller/internal/storage"
	"rtk_controller/pkg/types"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
)

// SimpleManager provides basic changeset management functionality
type SimpleManager struct {
	// Dependencies
	storage        storage.Storage
	commandManager *command.Manager
	
	// Active changesets in memory
	activeChangesets map[string]*types.Changeset
	mutex           sync.RWMutex
	
	// Configuration
	config *ManagerConfig
	
	// Control
	ctx     context.Context
	cancel  context.CancelFunc
	started bool
}

// ManagerConfig contains configuration for the changeset manager
type ManagerConfig struct {
	// MaxActiveChangesets limits the number of active changesets
	MaxActiveChangesets int
	
	// ChangesetTimeout is how long a changeset can remain active
	ChangesetTimeout time.Duration
	
	// AutoCleanup enables automatic cleanup of old changesets
	AutoCleanup bool
	
	// CleanupInterval is how often to run cleanup
	CleanupInterval time.Duration
	
	// RetentionDays is how long to keep completed changesets
	RetentionDays int
}

// DefaultManagerConfig returns sensible default configuration
func DefaultManagerConfig() *ManagerConfig {
	return &ManagerConfig{
		MaxActiveChangesets: 50,
		ChangesetTimeout:   2 * time.Hour,
		AutoCleanup:        true,
		CleanupInterval:    1 * time.Hour,
		RetentionDays:      30,
	}
}

// NewSimpleManager creates a new simple changeset manager
func NewSimpleManager(storage storage.Storage, commandManager *command.Manager) *SimpleManager {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &SimpleManager{
		storage:          storage,
		commandManager:   commandManager,
		activeChangesets: make(map[string]*types.Changeset),
		config:           DefaultManagerConfig(),
		ctx:              ctx,
		cancel:           cancel,
	}
}

// Start starts the changeset manager
func (m *SimpleManager) Start(ctx context.Context) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	if m.started {
		return fmt.Errorf("changeset manager already started")
	}
	
	// Load active changesets from storage
	if err := m.loadActiveChangesets(); err != nil {
		log.WithError(err).Warn("Failed to load active changesets")
	}
	
	// Start background workers
	if m.config.AutoCleanup {
		go m.cleanupWorker()
	}
	
	m.started = true
	log.Info("Simple changeset manager started")
	
	return nil
}

// Stop stops the changeset manager
func (m *SimpleManager) Stop() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	if !m.started {
		return nil
	}
	
	m.cancel()
	m.started = false
	
	log.Info("Simple changeset manager stopped")
	return nil
}

// CreateChangeset creates a new changeset
func (m *SimpleManager) CreateChangeset(ctx context.Context, options *types.ChangesetOptions) (*types.Changeset, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	// Check limits
	if len(m.activeChangesets) >= m.config.MaxActiveChangesets {
		return nil, fmt.Errorf("maximum active changesets exceeded (%d)", m.config.MaxActiveChangesets)
	}
	
	// Create changeset
	changeset := &types.Changeset{
		ID:        uuid.New().String(),
		Status:    types.ChangesetStatusDraft,
		CreatedAt: time.Now(),
		Commands:  make([]*types.Command, 0),
		Results:   make([]*types.CommandResult, 0),
		Metadata:  make(map[string]interface{}),
	}
	
	if options != nil {
		changeset.Description = options.Description
		changeset.CreatedBy = options.CreatedBy
		changeset.SessionID = options.SessionID
		changeset.TraceID = options.TraceID
		if options.Metadata != nil {
			for k, v := range options.Metadata {
				changeset.Metadata[k] = v
			}
		}
	}
	
	// Store in memory and persistence
	m.activeChangesets[changeset.ID] = changeset
	if err := m.persistChangeset(changeset); err != nil {
		delete(m.activeChangesets, changeset.ID)
		return nil, fmt.Errorf("failed to persist changeset: %w", err)
	}
	
	log.WithFields(log.Fields{
		"changeset_id": changeset.ID,
		"description":  changeset.Description,
		"session_id":   changeset.SessionID,
	}).Info("Created changeset")
	
	return changeset, nil
}

// AddCommandToChangeset adds a command to an existing changeset
func (m *SimpleManager) AddCommandToChangeset(changesetID string, cmd *types.Command) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	changeset, exists := m.activeChangesets[changesetID]
	if !exists {
		return fmt.Errorf("changeset %s not found", changesetID)
	}
	
	if changeset.Status != types.ChangesetStatusDraft {
		return fmt.Errorf("cannot add commands to changeset in status %s", changeset.Status)
	}
	
	changeset.AddCommand(cmd)
	
	if err := m.persistChangeset(changeset); err != nil {
		return fmt.Errorf("failed to persist changeset: %w", err)
	}
	
	log.WithFields(log.Fields{
		"changeset_id": changesetID,
		"command_id":   cmd.ID,
		"operation":    cmd.Operation,
	}).Debug("Added command to changeset")
	
	return nil
}

// ExecuteChangeset executes all commands in a changeset
func (m *SimpleManager) ExecuteChangeset(ctx context.Context, changesetID string) error {
	m.mutex.Lock()
	changeset, exists := m.activeChangesets[changesetID]
	if !exists {
		m.mutex.Unlock()
		return fmt.Errorf("changeset %s not found", changesetID)
	}
	
	if changeset.Status != types.ChangesetStatusDraft {
		m.mutex.Unlock()
		return fmt.Errorf("changeset %s is not in draft status", changesetID)
	}
	
	// Update status
	changeset.Status = types.ChangesetStatusExecuting
	m.mutex.Unlock()
	
	log.WithFields(log.Fields{
		"changeset_id":   changesetID,
		"command_count": len(changeset.Commands),
	}).Info("Executing changeset")
	
	startTime := time.Now()
	var allSuccessful = true
	
	// Execute each command
	for _, cmd := range changeset.Commands {
		cmdStartTime := time.Now()
		
		// Execute command through command manager
		// Convert timeout from Duration to seconds
		timeoutSeconds := int(cmd.Timeout.Seconds())
		if timeoutSeconds <= 0 {
			timeoutSeconds = 30 // default timeout
		}
		
		// Use SendCommand instead of ExecuteCommand
		_, err := m.commandManager.SendCommand("default", "default", cmd.DeviceID, cmd.Operation, cmd.Args, timeoutSeconds)
		
		duration := time.Since(cmdStartTime)
		result := &types.CommandResult{
			CommandID:  cmd.ID,
			Success:    err == nil,
			ExecutedAt: time.Now(),
			Duration:   duration,
		}
		
		if err != nil {
			result.Message = err.Error()
			allSuccessful = false
			log.WithFields(log.Fields{
				"changeset_id": changesetID,
				"command_id":   cmd.ID,
				"error":        err.Error(),
			}).Error("Command execution failed")
		} else {
			result.Message = "Command executed successfully"
			result.Data = cmd.Result // If the command manager sets result data
		}
		
		changeset.AddResult(result)
	}
	
	// Update final status
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	executedAt := time.Now()
	changeset.ExecutedAt = &executedAt
	
	if allSuccessful {
		changeset.Status = types.ChangesetStatusCompleted
		log.WithFields(log.Fields{
			"changeset_id": changesetID,
			"duration":     time.Since(startTime),
		}).Info("Changeset executed successfully")
	} else {
		changeset.Status = types.ChangesetStatusFailed
		log.WithFields(log.Fields{
			"changeset_id": changesetID,
			"duration":     time.Since(startTime),
		}).Error("Changeset execution failed")
	}
	
	// Persist updated changeset
	if err := m.persistChangeset(changeset); err != nil {
		log.WithError(err).Error("Failed to persist changeset after execution")
	}
	
	return nil
}

// RollbackChangeset rolls back a changeset by executing rollback commands
func (m *SimpleManager) RollbackChangeset(ctx context.Context, changesetID string) error {
	m.mutex.Lock()
	changeset, exists := m.activeChangesets[changesetID]
	if !exists {
		m.mutex.Unlock()
		return fmt.Errorf("changeset %s not found", changesetID)
	}
	
	if !changeset.IsRollbackable() {
		m.mutex.Unlock()
		return fmt.Errorf("changeset %s cannot be rolled back", changesetID)
	}
	
	if len(changeset.RollbackCommands) == 0 {
		m.mutex.Unlock()
		return fmt.Errorf("no rollback commands defined for changeset %s", changesetID)
	}
	m.mutex.Unlock()
	
	log.WithFields(log.Fields{
		"changeset_id":         changesetID,
		"rollback_command_count": len(changeset.RollbackCommands),
	}).Info("Rolling back changeset")
	
	startTime := time.Now()
	var allSuccessful = true
	
	// Execute rollback commands in reverse order
	for i := len(changeset.RollbackCommands) - 1; i >= 0; i-- {
		cmd := changeset.RollbackCommands[i]
		
		// Convert timeout from Duration to seconds for rollback command
		timeoutSeconds := int(cmd.Timeout.Seconds())
		if timeoutSeconds <= 0 {
			timeoutSeconds = 30 // default timeout
		}
		
		// Use SendCommand instead of ExecuteCommand
		_, err := m.commandManager.SendCommand("default", "default", cmd.DeviceID, cmd.Operation, cmd.Args, timeoutSeconds)
		if err != nil {
			allSuccessful = false
			log.WithFields(log.Fields{
				"changeset_id": changesetID,
				"command_id":   cmd.ID,
				"error":        err.Error(),
			}).Error("Rollback command execution failed")
		}
	}
	
	// Update status
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	rolledBackAt := time.Now()
	changeset.RolledBackAt = &rolledBackAt
	
	if allSuccessful {
		changeset.Status = types.ChangesetStatusRolledBack
		log.WithFields(log.Fields{
			"changeset_id": changesetID,
			"duration":     time.Since(startTime),
		}).Info("Changeset rolled back successfully")
	} else {
		changeset.Status = types.ChangesetStatusRollbackFailed
		log.WithFields(log.Fields{
			"changeset_id": changesetID,
			"duration":     time.Since(startTime),
		}).Error("Changeset rollback failed")
	}
	
	// Persist updated changeset
	if err := m.persistChangeset(changeset); err != nil {
		log.WithError(err).Error("Failed to persist changeset after rollback")
	}
	
	return nil
}

// GetChangeset retrieves a changeset by ID
func (m *SimpleManager) GetChangeset(changesetID string) (*types.Changeset, error) {
	m.mutex.RLock()
	if changeset, exists := m.activeChangesets[changesetID]; exists {
		m.mutex.RUnlock()
		return changeset, nil
	}
	m.mutex.RUnlock()
	
	// Try to load from storage
	return m.loadChangeset(changesetID)
}

// ListChangesets lists changesets with optional filtering
func (m *SimpleManager) ListChangesets(filter *types.ChangesetFilter, limit, offset int) ([]*types.ChangesetSummary, int, error) {
	var summaries []*types.ChangesetSummary
	var total int
	
	// Get from active changesets
	m.mutex.RLock()
	for _, changeset := range m.activeChangesets {
		if m.matchesFilter(changeset, filter) {
			summaries = append(summaries, changeset.GetSummary())
			total++
		}
	}
	m.mutex.RUnlock()
	
	// TODO: Also query from storage for complete list
	
	// Apply pagination
	if offset > 0 && offset < len(summaries) {
		summaries = summaries[offset:]
	}
	if limit > 0 && len(summaries) > limit {
		summaries = summaries[:limit]
	}
	
	return summaries, total, nil
}

// DeleteChangeset deletes a changeset
func (m *SimpleManager) DeleteChangeset(changesetID string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	changeset, exists := m.activeChangesets[changesetID]
	if !exists {
		return fmt.Errorf("changeset %s not found", changesetID)
	}
	
	if changeset.Status == types.ChangesetStatusExecuting {
		return fmt.Errorf("cannot delete changeset %s while executing", changesetID)
	}
	
	// Remove from memory and storage
	delete(m.activeChangesets, changesetID)
	
	if err := m.deleteChangesetFromStorage(changesetID); err != nil {
		log.WithError(err).Warn("Failed to delete changeset from storage")
	}
	
	log.WithField("changeset_id", changesetID).Info("Deleted changeset")
	return nil
}

// Private helper methods

func (m *SimpleManager) persistChangeset(changeset *types.Changeset) error {
	key := fmt.Sprintf("changeset:%s", changeset.ID)
	data, err := json.Marshal(changeset)
	if err != nil {
		return fmt.Errorf("failed to marshal changeset: %w", err)
	}
	return m.storage.Set(key, string(data))
}

func (m *SimpleManager) loadChangeset(changesetID string) (*types.Changeset, error) {
	key := fmt.Sprintf("changeset:%s", changesetID)
	
	data, err := m.storage.Get(key)
	if err != nil {
		return nil, fmt.Errorf("changeset %s not found", changesetID)
	}
	
	var changeset types.Changeset
	if err := json.Unmarshal([]byte(data), &changeset); err != nil {
		return nil, fmt.Errorf("failed to unmarshal changeset: %w", err)
	}
	
	return &changeset, nil
}

func (m *SimpleManager) loadActiveChangesets() error {
	// TODO: Implement loading active changesets from storage on startup
	return nil
}

func (m *SimpleManager) deleteChangesetFromStorage(changesetID string) error {
	key := fmt.Sprintf("changeset:%s", changesetID)
	return m.storage.Delete(key)
}

func (m *SimpleManager) matchesFilter(changeset *types.Changeset, filter *types.ChangesetFilter) bool {
	if filter == nil {
		return true
	}
	
	// Status filter
	if len(filter.Status) > 0 {
		found := false
		for _, status := range filter.Status {
			if changeset.Status == status {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	
	// CreatedBy filter
	if filter.CreatedBy != "" && changeset.CreatedBy != filter.CreatedBy {
		return false
	}
	
	// SessionID filter
	if filter.SessionID != "" && changeset.SessionID != filter.SessionID {
		return false
	}
	
	// Time range filter
	if filter.StartTime != nil && changeset.CreatedAt.Before(*filter.StartTime) {
		return false
	}
	if filter.EndTime != nil && changeset.CreatedAt.After(*filter.EndTime) {
		return false
	}
	
	return true
}

func (m *SimpleManager) cleanupWorker() {
	ticker := time.NewTicker(m.config.CleanupInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			m.cleanupOldChangesets()
		}
	}
}

func (m *SimpleManager) cleanupOldChangesets() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	cutoff := time.Now().Add(-time.Duration(m.config.RetentionDays) * 24 * time.Hour)
	
	for id, changeset := range m.activeChangesets {
		// Remove old completed/failed changesets
		if (changeset.Status == types.ChangesetStatusCompleted || 
		    changeset.Status == types.ChangesetStatusFailed ||
		    changeset.Status == types.ChangesetStatusRolledBack) &&
		   changeset.CreatedAt.Before(cutoff) {
			
			delete(m.activeChangesets, id)
			
			// Also remove from storage
			if err := m.deleteChangesetFromStorage(id); err != nil {
				log.WithError(err).WithField("changeset_id", id).
					Warn("Failed to delete old changeset from storage")
			} else {
				log.WithField("changeset_id", id).Debug("Cleaned up old changeset")
			}
		}
	}
}