package storage

import "errors"

// ErrStopIteration is returned to stop iteration
var ErrStopIteration = errors.New("stop iteration")

// Storage defines the interface for data storage operations
type Storage interface {
	// Basic operations
	Set(key, value string) error
	Get(key string) (string, error)
	Delete(key string) error
	Exists(key string) (bool, error)

	// Transaction operations
	Transaction(fn func(Transaction) error) error
	View(fn func(Transaction) error) error

	// Close the storage
	Close() error
}

// Transaction defines the interface for transactional operations
type Transaction interface {
	// Basic operations within transaction
	Set(key, value string) error
	Get(key string) (string, error)
	Delete(key string) error
	Exists(key string) (bool, error)

	// Iteration operations
	IterateRange(startKey, endKey string, fn func(key, value string) error) error
	IteratePrefix(prefix string, fn func(key, value string) error) error

	// Batch operations
	DeleteRange(startKey, endKey string) (int, error)
}
