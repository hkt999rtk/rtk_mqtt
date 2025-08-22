package device

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"rtk_controller/internal/storage"
	"rtk_controller/pkg/types"
	"rtk_controller/pkg/utils"

	log "github.com/sirupsen/logrus"
)

// EventProcessor handles device event processing
type EventProcessor struct {
	storage storage.Storage

	// Event queue for processing
	eventQueue chan *types.DeviceEvent

	// Background workers
	ctx    context.Context
	cancel context.CancelFunc
	done   chan struct{}

	// Statistics
	stats *types.EventStats
	mu    sync.RWMutex

	// Event handlers
	handlers map[string]EventHandler
}

// EventHandler defines interface for event handling
type EventHandler interface {
	HandleEvent(event *types.DeviceEvent) error
	CanHandle(eventType string) bool
}

// NewEventProcessor creates a new event processor
func NewEventProcessor(storage storage.Storage) *EventProcessor {
	ctx, cancel := context.WithCancel(context.Background())

	return &EventProcessor{
		storage:    storage,
		eventQueue: make(chan *types.DeviceEvent, 1000),
		ctx:        ctx,
		cancel:     cancel,
		done:       make(chan struct{}),
		stats:      &types.EventStats{},
		handlers:   make(map[string]EventHandler),
	}
}

// Start starts the event processor
func (ep *EventProcessor) Start(ctx context.Context) error {
	log.Info("Starting event processor...")

	// Start worker goroutines
	go ep.eventWorker()
	go ep.statsWorker()

	log.Info("Event processor started")
	return nil
}

// Stop stops the event processor
func (ep *EventProcessor) Stop() {
	log.Info("Stopping event processor...")

	ep.cancel()
	close(ep.eventQueue)
	<-ep.done

	log.Info("Event processor stopped")
}

// ProcessEvent processes an event from MQTT message
func (ep *EventProcessor) ProcessEvent(topic string, payload []byte) error {
	// Parse topic to extract device and event information
	tenant, site, deviceID, messageType, subParts := utils.ExtractTopicParts(topic)
	if deviceID == "" || messageType != "evt" {
		return fmt.Errorf("invalid event topic format: %s", topic)
	}

	if len(subParts) == 0 {
		return fmt.Errorf("missing event type in topic: %s", topic)
	}

	eventType := subParts[0]

	// Parse JSON payload
	var data map[string]interface{}
	if err := json.Unmarshal(payload, &data); err != nil {
		return fmt.Errorf("failed to parse event JSON payload: %w", err)
	}

	// Create event object
	event := &types.DeviceEvent{
		ID:        utils.GenerateMessageID(),
		DeviceID:  fmt.Sprintf("%s:%s:%s", tenant, site, deviceID),
		EventType: eventType,
		Topic:     topic,
		Timestamp: time.Now().UnixMilli(),
		Processed: false,
		CreatedAt: time.Now(),
		Data:      data,
	}

	// Extract standard fields
	if severity, ok := data["severity"].(string); ok {
		event.Severity = severity
	} else {
		event.Severity = "info" // Default severity
	}

	if message, ok := data["message"].(string); ok {
		event.Message = message
	}

	// Override timestamp if provided in payload
	if ts, ok := data["ts"].(float64); ok {
		event.Timestamp = int64(ts)
	}

	// Queue event for processing
	select {
	case ep.eventQueue <- event:
		log.WithFields(log.Fields{
			"device_id":  deviceID,
			"event_type": eventType,
			"severity":   event.Severity,
		}).Debug("Event queued for processing")
		return nil
	default:
		// Queue is full, log error but don't block
		log.WithFields(log.Fields{
			"device_id":  deviceID,
			"event_type": eventType,
		}).Error("Event queue full, dropping event")
		return fmt.Errorf("event queue full")
	}
}

// RegisterHandler registers an event handler
func (ep *EventProcessor) RegisterHandler(name string, handler EventHandler) {
	log.WithField("handler", name).Info("Registering event handler")
	ep.handlers[name] = handler
}

// GetEvent returns an event by ID
func (ep *EventProcessor) GetEvent(eventID string) (*types.DeviceEvent, error) {
	var event types.DeviceEvent

	err := ep.storage.View(func(tx storage.Transaction) error {
		key := fmt.Sprintf("event:%s", eventID)
		value, err := tx.Get(key)
		if err != nil {
			return err
		}

		return json.Unmarshal([]byte(value), &event)
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get event: %w", err)
	}

	return &event, nil
}

// ListEvents returns events matching the filter
func (ep *EventProcessor) ListEvents(filter *types.EventFilter, limit int, offset int) ([]*types.DeviceEvent, int, error) {
	var events []*types.DeviceEvent
	var total int

	err := ep.storage.View(func(tx storage.Transaction) error {
		// Build time range for iteration
		var startKey, endKey string

		if filter != nil && filter.StartTime != nil {
			startKey = fmt.Sprintf("event:%d:", *filter.StartTime)
		} else {
			startKey = "event:"
		}

		if filter != nil && filter.EndTime != nil {
			endKey = fmt.Sprintf("event:%d:", *filter.EndTime+1)
		} else {
			endKey = "event:~" // Use tilde for end of range
		}

		return tx.IterateRange(startKey, endKey, func(key, value string) error {
			var event types.DeviceEvent
			if err := json.Unmarshal([]byte(value), &event); err != nil {
				log.WithError(err).Warn("Failed to unmarshal event")
				return nil // Continue iteration
			}

			if ep.eventMatchesFilter(&event, filter) {
				total++

				// Apply pagination
				if total > offset && (limit <= 0 || len(events) < limit) {
					events = append(events, &event)
				}
			}

			return nil
		})
	})

	return events, total, err
}

// eventMatchesFilter checks if event matches the filter criteria
func (ep *EventProcessor) eventMatchesFilter(event *types.DeviceEvent, filter *types.EventFilter) bool {
	if filter == nil {
		return true
	}

	if filter.DeviceID != "" && event.DeviceID != filter.DeviceID {
		return false
	}

	if filter.EventType != "" && event.EventType != filter.EventType {
		return false
	}

	if len(filter.Severity) > 0 {
		found := false
		for _, severity := range filter.Severity {
			if event.Severity == severity {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	if filter.Processed != nil && event.Processed != *filter.Processed {
		return false
	}

	if filter.StartTime != nil && event.Timestamp < *filter.StartTime {
		return false
	}

	if filter.EndTime != nil && event.Timestamp > *filter.EndTime {
		return false
	}

	return true
}

// GetStats returns event statistics
func (ep *EventProcessor) GetStats() *types.EventStats {
	ep.mu.RLock()
	defer ep.mu.RUnlock()

	// Return a copy of current stats
	stats := *ep.stats
	return &stats
}

// eventWorker processes events from the queue
func (ep *EventProcessor) eventWorker() {
	defer close(ep.done)

	for {
		select {
		case <-ep.ctx.Done():
			return
		case event, ok := <-ep.eventQueue:
			if !ok {
				return // Channel closed
			}

			ep.processEvent(event)
		}
	}
}

// processEvent processes a single event
func (ep *EventProcessor) processEvent(event *types.DeviceEvent) {
	log.WithFields(log.Fields{
		"event_id":   event.ID,
		"device_id":  event.DeviceID,
		"event_type": event.EventType,
		"severity":   event.Severity,
	}).Debug("Processing event")

	// Store event in database
	if err := ep.storeEvent(event); err != nil {
		log.WithError(err).Error("Failed to store event")
		return
	}

	// Process with registered handlers
	for handlerName, handler := range ep.handlers {
		if handler.CanHandle(event.EventType) {
			if err := handler.HandleEvent(event); err != nil {
				log.WithFields(log.Fields{
					"handler":    handlerName,
					"event_id":   event.ID,
					"event_type": event.EventType,
				}).WithError(err).Error("Event handler failed")
			}
		}
	}

	// Mark event as processed
	event.Processed = true
	now := time.Now()
	event.ProcessedAt = &now

	// Update event in storage
	if err := ep.storeEvent(event); err != nil {
		log.WithError(err).Error("Failed to update processed event")
	}

	log.WithFields(log.Fields{
		"event_id":   event.ID,
		"event_type": event.EventType,
	}).Debug("Event processed successfully")
}

// storeEvent stores event in database
func (ep *EventProcessor) storeEvent(event *types.DeviceEvent) error {
	return ep.storage.Transaction(func(tx storage.Transaction) error {
		key := fmt.Sprintf("event:%d:%s", event.Timestamp, event.ID)
		data, err := json.Marshal(event)
		if err != nil {
			return fmt.Errorf("failed to marshal event: %w", err)
		}

		return tx.Set(key, string(data))
	})
}

// statsWorker updates event statistics periodically
func (ep *EventProcessor) statsWorker() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ep.ctx.Done():
			return
		case <-ticker.C:
			ep.updateStats()
		}
	}
}

// updateStats calculates current event statistics
func (ep *EventProcessor) updateStats() {
	ep.mu.Lock()
	defer ep.mu.Unlock()

	stats := &types.EventStats{
		SeverityStats:  make(map[string]int),
		EventTypeStats: make(map[string]int),
		LastUpdated:    time.Now(),
	}

	// Count events in storage
	err := ep.storage.View(func(tx storage.Transaction) error {
		return tx.IteratePrefix("event:", func(key, value string) error {
			var event types.DeviceEvent
			if err := json.Unmarshal([]byte(value), &event); err != nil {
				return nil // Continue on error
			}

			stats.TotalEvents++
			stats.SeverityStats[event.Severity]++
			stats.EventTypeStats[event.EventType]++

			if event.Processed {
				stats.ProcessedCount++
			} else {
				stats.PendingCount++
			}

			return nil
		})
	})

	if err != nil {
		log.WithError(err).Error("Failed to update event statistics")
		return
	}

	ep.stats = stats
}

// DefaultEventHandler is a basic event handler implementation
type DefaultEventHandler struct{}

// HandleEvent handles events with default behavior
func (h *DefaultEventHandler) HandleEvent(event *types.DeviceEvent) error {
	switch event.Severity {
	case "critical", "error":
		log.WithFields(log.Fields{
			"device_id":  event.DeviceID,
			"event_type": event.EventType,
			"severity":   event.Severity,
			"message":    event.Message,
		}).Error("Critical device event")
	case "warning":
		log.WithFields(log.Fields{
			"device_id":  event.DeviceID,
			"event_type": event.EventType,
			"message":    event.Message,
		}).Warn("Device warning event")
	default:
		log.WithFields(log.Fields{
			"device_id":  event.DeviceID,
			"event_type": event.EventType,
			"message":    event.Message,
		}).Info("Device event")
	}

	return nil
}

// CanHandle returns true if this handler can handle the event type
func (h *DefaultEventHandler) CanHandle(eventType string) bool {
	// Default handler handles all event types
	return true
}
