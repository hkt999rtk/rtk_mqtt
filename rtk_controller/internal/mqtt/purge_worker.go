package mqtt

import (
	"context"
	"fmt"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"rtk_controller/internal/config"
	"rtk_controller/internal/storage"
)

// PurgeWorker handles automatic cleanup of old MQTT message logs
type PurgeWorker struct {
	config  config.MQTTLogging
	storage storage.Storage

	// Control channels
	ctx    context.Context
	cancel context.CancelFunc
	done   chan struct{}
	mu     sync.Mutex

	// Statistics
	lastPurgeTime   time.Time
	totalPurged     int64
	lastPurgedCount int64
}

// NewPurgeWorker creates a new purge worker
func NewPurgeWorker(config config.MQTTLogging, storage storage.Storage) (*PurgeWorker, error) {
	ctx, cancel := context.WithCancel(context.Background())

	worker := &PurgeWorker{
		config:  config,
		storage: storage,
		ctx:     ctx,
		cancel:  cancel,
		done:    make(chan struct{}),
	}

	return worker, nil
}

// Start starts the purge worker
func (pw *PurgeWorker) Start(ctx context.Context) error {
	if !pw.config.Enabled {
		log.Info("MQTT message logging disabled, purge worker not started")
		return nil
	}

	log.WithFields(log.Fields{
		"retention_seconds": pw.config.RetentionSeconds,
		"purge_interval":    pw.config.PurgeInterval,
	}).Info("Starting MQTT message purge worker")

	// Parse purge interval
	interval, err := time.ParseDuration(pw.config.PurgeInterval)
	if err != nil {
		return fmt.Errorf("invalid purge interval: %w", err)
	}

	// Start purge routine
	go pw.purgeRoutine(interval)

	return nil
}

// Stop stops the purge worker
func (pw *PurgeWorker) Stop() {
	log.Info("Stopping MQTT message purge worker")

	pw.cancel()

	// Wait for routine to finish
	<-pw.done

	log.Info("MQTT message purge worker stopped")
}

// purgeRoutine runs the periodic purge process
func (pw *PurgeWorker) purgeRoutine(interval time.Duration) {
	defer close(pw.done)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Run initial purge
	pw.runPurge()

	for {
		select {
		case <-pw.ctx.Done():
			return
		case <-ticker.C:
			pw.runPurge()
		}
	}
}

// runPurge performs the actual purge operation
func (pw *PurgeWorker) runPurge() {
	start := time.Now()

	log.Debug("Starting MQTT message log purge")

	// Calculate cutoff time
	cutoffTime := time.Now().Add(-time.Duration(pw.config.RetentionSeconds) * time.Second)
	cutoffTimestamp := cutoffTime.UnixMilli()

	// Purge old messages
	purgedCount, err := pw.purgeOldMessages(cutoffTimestamp)
	if err != nil {
		log.WithError(err).Error("Failed to purge old MQTT messages")
		return
	}

	duration := time.Since(start)

	// Update statistics
	pw.mu.Lock()
	pw.lastPurgeTime = start
	pw.totalPurged += int64(purgedCount)
	pw.lastPurgedCount = int64(purgedCount)
	pw.mu.Unlock()

	if purgedCount > 0 {
		log.WithFields(log.Fields{
			"purged_count": purgedCount,
			"duration_ms":  duration.Milliseconds(),
			"cutoff_time":  cutoffTime.Format(time.RFC3339),
		}).Info("Purged old MQTT message logs")
	} else {
		log.Debug("No old MQTT message logs to purge")
	}
}

// purgeOldMessages deletes messages older than the given timestamp
func (pw *PurgeWorker) purgeOldMessages(cutoffTimestamp int64) (int, error) {
	var totalDeleted int

	err := pw.storage.Transaction(func(tx storage.Transaction) error {
		// Define range for old messages
		startKey := "mqtt_log:"
		endKey := fmt.Sprintf("mqtt_log:%d:", cutoffTimestamp)

		// Delete messages in batches to avoid long transactions
		batchSize := 1000
		for {
			deleted, err := pw.deleteMessageBatch(tx, startKey, endKey, batchSize)
			if err != nil {
				return fmt.Errorf("failed to delete message batch: %w", err)
			}

			totalDeleted += deleted

			// If we deleted fewer than batch size, we're done
			if deleted < batchSize {
				break
			}
		}

		return nil
	})

	return totalDeleted, err
}

// deleteMessageBatch deletes a batch of messages within the given range
func (pw *PurgeWorker) deleteMessageBatch(tx storage.Transaction, startKey, endKey string, batchSize int) (int, error) {
	var keysToDelete []string

	// Collect keys to delete (limited by batch size)
	err := tx.IterateRange(startKey, endKey, func(key, value string) error {
		keysToDelete = append(keysToDelete, key)

		// Stop when we reach batch size
		if len(keysToDelete) >= batchSize {
			return storage.ErrStopIteration
		}

		return nil
	})

	if err != nil {
		return 0, err
	}

	// Delete collected keys
	for _, key := range keysToDelete {
		if err := tx.Delete(key); err != nil {
			return len(keysToDelete), fmt.Errorf("failed to delete key %s: %w", key, err)
		}
	}

	return len(keysToDelete), nil
}

// GetStats returns purge worker statistics
func (pw *PurgeWorker) GetStats() (lastPurgeTime time.Time, totalPurged, lastPurgedCount int64) {
	pw.mu.Lock()
	defer pw.mu.Unlock()

	return pw.lastPurgeTime, pw.totalPurged, pw.lastPurgedCount
}

// ForcePurge manually triggers a purge operation
func (pw *PurgeWorker) ForcePurge() (int, error) {
	log.Info("Manual purge triggered")

	// Calculate cutoff time
	cutoffTime := time.Now().Add(-time.Duration(pw.config.RetentionSeconds) * time.Second)
	cutoffTimestamp := cutoffTime.UnixMilli()

	// Purge old messages
	purgedCount, err := pw.purgeOldMessages(cutoffTimestamp)
	if err != nil {
		return 0, fmt.Errorf("manual purge failed: %w", err)
	}

	// Update statistics
	pw.mu.Lock()
	pw.lastPurgeTime = time.Now()
	pw.totalPurged += int64(purgedCount)
	pw.lastPurgedCount = int64(purgedCount)
	pw.mu.Unlock()

	log.WithFields(log.Fields{
		"purged_count": purgedCount,
		"cutoff_time":  cutoffTime.Format(time.RFC3339),
	}).Info("Manual purge completed")

	return purgedCount, nil
}
