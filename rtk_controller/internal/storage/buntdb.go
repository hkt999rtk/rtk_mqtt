package storage

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/tidwall/buntdb"
)

// BuntDBStorage implements Storage interface using BuntDB
type BuntDBStorage struct {
	db   *buntdb.DB
	path string
}

// NewBuntDB creates a new BuntDB storage instance
func NewBuntDB(dataPath string) (Storage, error) {
	// Ensure data directory exists
	if err := os.MkdirAll(dataPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	// Open main database
	dbPath := filepath.Join(dataPath, "controller.db")
	db, err := buntdb.Open(dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	storage := &BuntDBStorage{
		db:   db,
		path: dbPath,
	}

	// Create indices
	if err := storage.createIndices(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create indices: %w", err)
	}

	return storage, nil
}

// createIndices creates database indices for efficient querying
func (s *BuntDBStorage) createIndices() error {
	return s.db.Update(func(tx *buntdb.Tx) error {
		// Create index for MQTT message logs by timestamp
		err := tx.CreateIndex("mqtt_log_time", "mqtt_log:*", buntdb.IndexString)
		if err != nil && err != buntdb.ErrIndexExists {
			return err
		}

		// Create index for device data
		err = tx.CreateIndex("device_id", "device:*", buntdb.IndexString)
		if err != nil && err != buntdb.ErrIndexExists {
			return err
		}

		// Create index for commands
		err = tx.CreateIndex("command_time", "command:*", buntdb.IndexString)
		if err != nil && err != buntdb.ErrIndexExists {
			return err
		}

		// Create index for events
		err = tx.CreateIndex("event_time", "event:*", buntdb.IndexString)
		if err != nil && err != buntdb.ErrIndexExists {
			return err
		}

		return nil
	})
}

// Set stores a key-value pair
func (s *BuntDBStorage) Set(key, value string) error {
	return s.db.Update(func(tx *buntdb.Tx) error {
		_, _, err := tx.Set(key, value, nil)
		return err
	})
}

// Get retrieves a value by key
func (s *BuntDBStorage) Get(key string) (string, error) {
	var value string
	err := s.db.View(func(tx *buntdb.Tx) error {
		var err error
		value, err = tx.Get(key)
		return err
	})
	return value, err
}

// Delete removes a key
func (s *BuntDBStorage) Delete(key string) error {
	return s.db.Update(func(tx *buntdb.Tx) error {
		_, err := tx.Delete(key)
		return err
	})
}

// Exists checks if a key exists
func (s *BuntDBStorage) Exists(key string) (bool, error) {
	_, err := s.Get(key)
	if err == buntdb.ErrNotFound {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// Transaction executes a function within a transaction
func (s *BuntDBStorage) Transaction(fn func(Transaction) error) error {
	return s.db.Update(func(tx *buntdb.Tx) error {
		return fn(&BuntDBTransaction{tx: tx})
	})
}

// View executes a read-only function
func (s *BuntDBStorage) View(fn func(Transaction) error) error {
	return s.db.View(func(tx *buntdb.Tx) error {
		return fn(&BuntDBTransaction{tx: tx})
	})
}

// Close closes the database
func (s *BuntDBStorage) Close() error {
	return s.db.Close()
}

// BuntDBTransaction implements Transaction interface
type BuntDBTransaction struct {
	tx *buntdb.Tx
}

// Set stores a key-value pair in transaction
func (t *BuntDBTransaction) Set(key, value string) error {
	_, _, err := t.tx.Set(key, value, nil)
	return err
}

// Get retrieves a value by key in transaction
func (t *BuntDBTransaction) Get(key string) (string, error) {
	return t.tx.Get(key)
}

// Delete removes a key in transaction
func (t *BuntDBTransaction) Delete(key string) error {
	_, err := t.tx.Delete(key)
	return err
}

// Exists checks if a key exists in transaction
func (t *BuntDBTransaction) Exists(key string) (bool, error) {
	_, err := t.tx.Get(key)
	if err == buntdb.ErrNotFound {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// IterateRange iterates over keys in a range
func (t *BuntDBTransaction) IterateRange(startKey, endKey string, fn func(key, value string) error) error {
	return t.tx.AscendRange("", startKey, endKey, func(key, value string) bool {
		if err := fn(key, value); err != nil {
			if err == ErrStopIteration {
				return false
			}
			// Log error but continue iteration
			return true
		}
		return true
	})
}

// IteratePrefix iterates over keys with a prefix
func (t *BuntDBTransaction) IteratePrefix(prefix string, fn func(key, value string) error) error {
	return t.tx.AscendKeys(prefix+"*", func(key, value string) bool {
		if err := fn(key, value); err != nil {
			if err == ErrStopIteration {
				return false
			}
			// Log error but continue iteration
			return true
		}
		return true
	})
}

// DeleteRange deletes keys in a range
func (t *BuntDBTransaction) DeleteRange(startKey, endKey string) (int, error) {
	var deleted int

	// First collect keys to delete
	var keysToDelete []string
	err := t.tx.AscendRange("", startKey, endKey, func(key, value string) bool {
		keysToDelete = append(keysToDelete, key)
		return true
	})

	if err != nil {
		return 0, err
	}

	// Then delete them
	for _, key := range keysToDelete {
		if _, err := t.tx.Delete(key); err != nil {
			return deleted, err
		}
		deleted++
	}

	return deleted, nil
}
