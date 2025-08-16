package schema

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"rtk_controller/internal/storage"
)

// Manager handles JSON schema validation for MQTT messages
type Manager struct {
	validator    *Validator
	storage      storage.Storage
	config       Config
	
	// Statistics
	stats        *ValidationStats
	mu           sync.RWMutex
	
	// Validation results cache (optional)
	resultCache  map[string]*ValidationResult
	cacheSize    int
}

// Config holds schema validation configuration
type Config struct {
	Enabled           bool     `json:"enabled"`
	SchemaFiles       []string `json:"schema_files"`
	StrictValidation  bool     `json:"strict_validation"`
	LogValidationErrors bool   `json:"log_validation_errors"`
	CacheResults      bool     `json:"cache_results"`
	CacheSize         int      `json:"cache_size"`
	StoreResults      bool     `json:"store_results"`
}

// ValidationStats holds validation statistics
type ValidationStats struct {
	TotalValidations  int64              `json:"total_validations"`
	ValidMessages     int64              `json:"valid_messages"`
	InvalidMessages   int64              `json:"invalid_messages"`
	SkippedMessages   int64              `json:"skipped_messages"`
	SchemaStats       map[string]int64   `json:"schema_stats"`
	ErrorStats        map[string]int64   `json:"error_stats"`
	LastValidation    time.Time          `json:"last_validation"`
	ValidationRate    float64            `json:"validation_rate_per_sec"`
}

// NewManager creates a new schema manager
func NewManager(config Config, storage storage.Storage) (*Manager, error) {
	manager := &Manager{
		validator:   NewValidator(),
		storage:     storage,
		config:      config,
		stats:       &ValidationStats{
			SchemaStats: make(map[string]int64),
			ErrorStats:  make(map[string]int64),
		},
		cacheSize:   config.CacheSize,
	}
	
	if config.CacheResults && config.CacheSize > 0 {
		manager.resultCache = make(map[string]*ValidationResult)
	}
	
	return manager, nil
}

// Initialize loads schemas and prepares the manager
func (m *Manager) Initialize() error {
	if !m.config.Enabled {
		log.Info("Schema validation disabled")
		return nil
	}
	
	log.Info("Initializing JSON schema validation")
	
	// Load built-in schemas
	if err := m.validator.LoadBuiltinSchemas(); err != nil {
		return fmt.Errorf("failed to load built-in schemas: %w", err)
	}
	
	// Load external schema files
	for _, schemaFile := range m.config.SchemaFiles {
		if err := m.loadSchemaFile(schemaFile); err != nil {
			log.WithError(err).WithField("file", schemaFile).Error("Failed to load schema file")
			if m.config.StrictValidation {
				return fmt.Errorf("failed to load schema file %s: %w", schemaFile, err)
			}
		}
	}
	
	schemas := m.validator.GetLoadedSchemas()
	log.WithField("schemas", schemas).Info("Schema validation initialized")
	
	return nil
}

// ValidateMessage validates an MQTT message
func (m *Manager) ValidateMessage(topic string, payload []byte) (*ValidationResult, error) {
	if !m.config.Enabled {
		return &ValidationResult{Valid: true, Schema: "disabled"}, nil
	}
	
	startTime := time.Now()
	
	// Update statistics
	m.mu.Lock()
	m.stats.TotalValidations++
	m.stats.LastValidation = startTime
	m.mu.Unlock()
	
	// Check cache first (if enabled)
	cacheKey := m.getCacheKey(topic, payload)
	if m.config.CacheResults && m.resultCache != nil {
		if cached, exists := m.resultCache[cacheKey]; exists {
			return cached, nil
		}
	}
	
	// Validate by topic first, then by schema field
	var result *ValidationResult
	var err error
	
	// Try topic-based validation
	result, err = m.validator.ValidateByTopic(topic, payload)
	if err != nil {
		return nil, fmt.Errorf("topic-based validation failed: %w", err)
	}
	
	// If no schema matched from topic, try schema field validation
	if result.Schema == "unknown" || result.Schema == "no_schema" {
		schemaResult, schemaErr := m.validator.ValidateBySchemaField(payload)
		if schemaErr == nil && schemaResult.Schema != "no_schema" {
			result = schemaResult
		}
	}
	
	// Update statistics
	m.updateStats(result)
	
	// Log validation errors if enabled
	if !result.Valid && m.config.LogValidationErrors {
		log.WithFields(log.Fields{
			"topic":  topic,
			"schema": result.Schema,
			"errors": result.Errors,
		}).Warn("Message validation failed")
	}
	
	// Cache result if enabled
	if m.config.CacheResults && m.resultCache != nil {
		m.cacheResult(cacheKey, result)
	}
	
	// Store result if enabled
	if m.config.StoreResults {
		m.storeValidationResult(topic, result)
	}
	
	// Log validation time
	validationTime := time.Since(startTime)
	if validationTime > 10*time.Millisecond {
		log.WithFields(log.Fields{
			"topic":           topic,
			"schema":          result.Schema,
			"validation_time": validationTime,
		}).Debug("Slow validation detected")
	}
	
	return result, nil
}

// ValidateJSON validates raw JSON data against a specific schema
func (m *Manager) ValidateJSON(schemaName string, jsonData []byte) (*ValidationResult, error) {
	if !m.config.Enabled {
		return &ValidationResult{Valid: true, Schema: "disabled"}, nil
	}
	
	result, err := m.validator.Validate(schemaName, jsonData)
	if err != nil {
		return nil, err
	}
	
	m.updateStats(result)
	return result, nil
}

// GetStats returns current validation statistics
func (m *Manager) GetStats() *ValidationStats {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	// Calculate validation rate
	if m.stats.TotalValidations > 0 && !m.stats.LastValidation.IsZero() {
		duration := time.Since(m.stats.LastValidation).Seconds()
		if duration > 0 {
			m.stats.ValidationRate = float64(m.stats.TotalValidations) / duration
		}
	}
	
	// Return copy of stats
	stats := *m.stats
	stats.SchemaStats = make(map[string]int64)
	stats.ErrorStats = make(map[string]int64)
	
	for k, v := range m.stats.SchemaStats {
		stats.SchemaStats[k] = v
	}
	for k, v := range m.stats.ErrorStats {
		stats.ErrorStats[k] = v
	}
	
	return &stats
}

// GetLoadedSchemas returns list of loaded schemas
func (m *Manager) GetLoadedSchemas() []string {
	return m.validator.GetLoadedSchemas()
}

// ReloadSchemas reloads all schemas
func (m *Manager) ReloadSchemas() error {
	log.Info("Reloading JSON schemas")
	
	// Create new validator
	newValidator := NewValidator()
	
	// Load built-in schemas
	if err := newValidator.LoadBuiltinSchemas(); err != nil {
		return fmt.Errorf("failed to reload built-in schemas: %w", err)
	}
	
	// Load external schema files
	for _, schemaFile := range m.config.SchemaFiles {
		if err := m.loadSchemaFileToValidator(newValidator, schemaFile); err != nil {
			log.WithError(err).WithField("file", schemaFile).Error("Failed to reload schema file")
			if m.config.StrictValidation {
				return fmt.Errorf("failed to reload schema file %s: %w", schemaFile, err)
			}
		}
	}
	
	// Replace validator
	m.validator = newValidator
	
	// Clear cache
	if m.resultCache != nil {
		m.resultCache = make(map[string]*ValidationResult)
	}
	
	log.Info("Schemas reloaded successfully")
	return nil
}

// Private methods

func (m *Manager) loadSchemaFile(filePath string) error {
	return m.loadSchemaFileToValidator(m.validator, filePath)
}

func (m *Manager) loadSchemaFileToValidator(validator *Validator, filePath string) error {
	// Check file extension to determine loading method
	ext := filepath.Ext(filePath)
	
	switch ext {
	case ".json":
		return validator.LoadSchemasFromFile(filePath)
	default:
		return fmt.Errorf("unsupported schema file format: %s", ext)
	}
}

func (m *Manager) updateStats(result *ValidationResult) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// Update schema stats
	m.stats.SchemaStats[result.Schema]++
	
	if result.Valid {
		m.stats.ValidMessages++
	} else if len(result.Errors) > 0 {
		m.stats.InvalidMessages++
		
		// Update error stats
		for _, err := range result.Errors {
			m.stats.ErrorStats[err]++
		}
	} else {
		m.stats.SkippedMessages++
	}
}

func (m *Manager) getCacheKey(topic string, payload []byte) string {
	// Simple cache key based on topic and payload hash
	// For production, consider using a proper hash function
	return fmt.Sprintf("%s_%d", topic, len(payload))
}

func (m *Manager) cacheResult(key string, result *ValidationResult) {
	// Implement simple LRU-like cache
	if len(m.resultCache) >= m.cacheSize {
		// Remove oldest entry (simple implementation)
		for k := range m.resultCache {
			delete(m.resultCache, k)
			break
		}
	}
	
	m.resultCache[key] = result
}

func (m *Manager) storeValidationResult(topic string, result *ValidationResult) {
	// Store validation result in database
	go func() {
		err := m.storage.Transaction(func(tx storage.Transaction) error {
			key := fmt.Sprintf("validation:%d:%s", time.Now().UnixMilli(), topic)
			data, err := json.Marshal(result)
			if err != nil {
				return err
			}
			return tx.Set(key, string(data))
		})
		
		if err != nil {
			log.WithError(err).Error("Failed to store validation result")
		}
	}()
}

// ValidationMiddleware can be used as MQTT message middleware
type ValidationMiddleware struct {
	manager *Manager
}

// NewValidationMiddleware creates validation middleware
func NewValidationMiddleware(manager *Manager) *ValidationMiddleware {
	return &ValidationMiddleware{manager: manager}
}

// ProcessMessage processes and validates MQTT message
func (vm *ValidationMiddleware) ProcessMessage(topic string, payload []byte) (bool, error) {
	result, err := vm.manager.ValidateMessage(topic, payload)
	if err != nil {
		return false, fmt.Errorf("validation error: %w", err)
	}
	
	// Return true if message is valid, false if invalid
	return result.Valid, nil
}