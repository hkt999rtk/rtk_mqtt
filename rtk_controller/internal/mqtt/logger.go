package mqtt

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"rtk_controller/internal/config"
	"rtk_controller/internal/storage"
	"rtk_controller/pkg/utils"
)

// MQTTMessageLog represents a logged MQTT message
type MQTTMessageLog struct {
	ID          string `json:"id"`
	Timestamp   int64  `json:"timestamp"`
	Topic       string `json:"topic"`
	Payload     string `json:"payload"`
	QoS         byte   `json:"qos"`
	Retained    bool   `json:"retained"`
	ClientID    string `json:"client_id"`
	MessageSize int    `json:"message_size"`
}

// MessageLogger handles MQTT message logging
type MessageLogger struct {
	config      config.MQTTLogging
	storage     storage.Storage
	purgeWorker *PurgeWorker
	
	// Message buffer for batch processing
	buffer      chan *MQTTMessageLog
	batchBuffer []*MQTTMessageLog
	mu          sync.Mutex
	
	// Control channels
	ctx        context.Context
	cancel     context.CancelFunc
	done       chan struct{}
	
	// Statistics
	stats      *LoggingStats
}

// LoggingStats holds logging statistics
type LoggingStats struct {
	TotalMessages   int64     `json:"total_messages"`
	TotalBytes      int64     `json:"total_bytes"`
	MessagesPerSec  float64   `json:"messages_per_sec"`
	BytesPerSec     float64   `json:"bytes_per_sec"`
	DroppedMessages int64     `json:"dropped_messages"`
	PurgedMessages  int64     `json:"purged_messages"`
	LastPurgeTime   time.Time `json:"last_purge_time"`
	DatabaseSize    int64     `json:"database_size"`
}

// NewMessageLogger creates a new message logger
func NewMessageLogger(config config.MQTTLogging, storage storage.Storage) (*MessageLogger, error) {
	ctx, cancel := context.WithCancel(context.Background())
	
	logger := &MessageLogger{
		config:      config,
		storage:     storage,
		buffer:      make(chan *MQTTMessageLog, config.BatchSize*2),
		batchBuffer: make([]*MQTTMessageLog, 0, config.BatchSize),
		ctx:         ctx,
		cancel:      cancel,
		done:        make(chan struct{}),
		stats:       &LoggingStats{},
	}

	// Initialize purge worker
	purgeWorker, err := NewPurgeWorker(config, storage)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create purge worker: %w", err)
	}
	logger.purgeWorker = purgeWorker

	return logger, nil
}

// Start starts the message logger
func (ml *MessageLogger) Start(ctx context.Context) error {
	log.Info("Starting MQTT message logger")

	// Start purge worker
	if err := ml.purgeWorker.Start(ctx); err != nil {
		return fmt.Errorf("failed to start purge worker: %w", err)
	}

	// Start batch processor
	go ml.batchProcessor()

	log.WithFields(log.Fields{
		"retention_seconds": ml.config.RetentionSeconds,
		"batch_size":        ml.config.BatchSize,
		"max_message_size":  ml.config.MaxMessageSize,
	}).Info("MQTT message logger started")

	return nil
}

// Stop stops the message logger
func (ml *MessageLogger) Stop() {
	log.Info("Stopping MQTT message logger")
	
	ml.cancel()
	
	// Wait for batch processor to finish
	<-ml.done
	
	// Stop purge worker
	ml.purgeWorker.Stop()
	
	log.Info("MQTT message logger stopped")
}

// LogMessage logs an MQTT message
func (ml *MessageLogger) LogMessage(topic string, payload []byte, qos byte, retained bool) error {
	// Check if logging is enabled
	if !ml.config.Enabled {
		return nil
	}

	// Check if topic should be excluded
	for _, excludeTopic := range ml.config.ExcludeTopics {
		if utils.TopicMatches(excludeTopic, topic) {
			return nil
		}
	}

	// Check message size limit
	if len(payload) > ml.config.MaxMessageSize {
		log.WithFields(log.Fields{
			"topic": topic,
			"size":  len(payload),
			"limit": ml.config.MaxMessageSize,
		}).Warn("Message size exceeds limit, skipping log")
		
		ml.mu.Lock()
		ml.stats.DroppedMessages++
		ml.mu.Unlock()
		return nil
	}

	// Create log entry
	logEntry := &MQTTMessageLog{
		ID:          utils.GenerateMessageID(),
		Timestamp:   time.Now().UnixMilli(),
		Topic:       topic,
		Payload:     string(payload),
		QoS:         qos,
		Retained:    retained,
		ClientID:    utils.ExtractClientID(topic),
		MessageSize: len(payload),
	}

	// Try to add to buffer (non-blocking)
	select {
	case ml.buffer <- logEntry:
		// Successfully added to buffer
	default:
		// Buffer is full, drop message
		log.Warn("Message buffer full, dropping message")
		ml.mu.Lock()
		ml.stats.DroppedMessages++
		ml.mu.Unlock()
	}

	return nil
}

// batchProcessor processes messages in batches
func (ml *MessageLogger) batchProcessor() {
	defer close(ml.done)
	
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ml.ctx.Done():
			// Process remaining messages
			ml.flushBatch()
			return
			
		case message := <-ml.buffer:
			ml.batchBuffer = append(ml.batchBuffer, message)
			
			// Flush if batch is full
			if len(ml.batchBuffer) >= ml.config.BatchSize {
				ml.flushBatch()
			}
			
		case <-ticker.C:
			// Flush batch periodically
			if len(ml.batchBuffer) > 0 {
				ml.flushBatch()
			}
		}
	}
}

// flushBatch writes accumulated messages to storage
func (ml *MessageLogger) flushBatch() {
	if len(ml.batchBuffer) == 0 {
		return
	}

	// Write to storage
	if err := ml.writeBatch(ml.batchBuffer); err != nil {
		log.WithError(err).Error("Failed to write message batch to storage")
	} else {
		// Update statistics
		ml.mu.Lock()
		ml.stats.TotalMessages += int64(len(ml.batchBuffer))
		for _, msg := range ml.batchBuffer {
			ml.stats.TotalBytes += int64(msg.MessageSize)
		}
		ml.mu.Unlock()
	}

	// Clear batch buffer
	ml.batchBuffer = ml.batchBuffer[:0]
}

// writeBatch writes a batch of messages to storage
func (ml *MessageLogger) writeBatch(messages []*MQTTMessageLog) error {
	return ml.storage.Transaction(func(tx storage.Transaction) error {
		for _, msg := range messages {
			key := fmt.Sprintf("mqtt_log:%d:%s", msg.Timestamp, msg.ID)
			data, err := json.Marshal(msg)
			if err != nil {
				return fmt.Errorf("failed to marshal message: %w", err)
			}
			
			if err := tx.Set(key, string(data)); err != nil {
				return fmt.Errorf("failed to store message: %w", err)
			}
		}
		return nil
	})
}

// GetStats returns current logging statistics
func (ml *MessageLogger) GetStats() *LoggingStats {
	ml.mu.Lock()
	defer ml.mu.Unlock()
	
	// Create a copy of stats
	stats := *ml.stats
	
	// TODO: Calculate rates (messages/sec, bytes/sec)
	// This would require tracking time windows
	
	return &stats
}

// QueryMessages queries messages from storage
func (ml *MessageLogger) QueryMessages(startTime, endTime int64, filters map[string]string, limit int) ([]*MQTTMessageLog, error) {
	var messages []*MQTTMessageLog
	
	err := ml.storage.View(func(tx storage.Transaction) error {
		// Create iterator for time range
		startKey := fmt.Sprintf("mqtt_log:%d:", startTime)
		endKey := fmt.Sprintf("mqtt_log:%d:", endTime+1)
		
		return tx.IterateRange(startKey, endKey, func(key, value string) error {
			var msg MQTTMessageLog
			if err := json.Unmarshal([]byte(value), &msg); err != nil {
				log.WithError(err).Warn("Failed to unmarshal message log")
				return nil // Continue iteration
			}
			
			// Apply filters
			if ml.applyFilters(&msg, filters) {
				messages = append(messages, &msg)
				
				// Check limit
				if limit > 0 && len(messages) >= limit {
					return storage.ErrStopIteration
				}
			}
			
			return nil
		})
	})
	
	return messages, err
}

// applyFilters checks if a message matches the given filters
func (ml *MessageLogger) applyFilters(msg *MQTTMessageLog, filters map[string]string) bool {
	for key, value := range filters {
		switch key {
		case "topic_filter":
			if !utils.TopicMatches(value, msg.Topic) {
				return false
			}
		case "device_filter":
			if !utils.DeviceMatches(value, msg.ClientID) {
				return false
			}
		case "message_type":
			// Extract message type from topic (e.g., state, evt, cmd)
			msgType := utils.ExtractMessageType(msg.Topic)
			if msgType != value {
				return false
			}
		case "min_size":
			// TODO: Parse and compare message size
		}
	}
	return true
}