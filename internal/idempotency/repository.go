// Package idempotency provides repository implementations for idempotency key storage.
package idempotency

import (
	"sync"
	"time"
)

// InMemoryRepository implements Repository with in-memory storage.
type InMemoryRepository struct {
	mu   sync.RWMutex
	keys map[string]*IdempotencyKey
}

// NewInMemoryRepository creates a new in-memory idempotency key repository.
func NewInMemoryRepository() *InMemoryRepository {
	return &InMemoryRepository{
		keys: make(map[string]*IdempotencyKey),
	}
}

// Get retrieves an idempotency key by its key value.
// Returns ErrKeyNotFound if the key doesn't exist.
func (r *InMemoryRepository) Get(key string) (*IdempotencyKey, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	record, ok := r.keys[key]
	if !ok {
		return nil, ErrKeyNotFound
	}

	// Return a copy to prevent external mutation
	return r.copyRecord(record), nil
}

// Store saves a new idempotency key.
// Returns ErrKeyExists if the key already exists.
func (r *InMemoryRepository) Store(record *IdempotencyKey) error {
	if err := ValidateKey(record.Key); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if key already exists
	if _, exists := r.keys[record.Key]; exists {
		return ErrKeyExists
	}

	// Set created_at if not provided
	if record.CreatedAt.IsZero() {
		record.CreatedAt = time.Now()
	}

	// Store a copy to prevent external mutation
	r.keys[record.Key] = r.copyRecord(record)

	return nil
}

// DeleteOlderThan removes idempotency keys older than the specified duration.
// Returns the number of keys deleted.
func (r *InMemoryRepository) DeleteOlderThan(duration time.Duration) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	cutoffTime := time.Now().Add(-duration)
	deleted := int64(0)

	for key, record := range r.keys {
		if record.CreatedAt.Before(cutoffTime) {
			delete(r.keys, key)
			deleted++
		}
	}

	return deleted, nil
}

// copyRecord creates a deep copy of an IdempotencyKey.
func (r *InMemoryRepository) copyRecord(record *IdempotencyKey) *IdempotencyKey {
	if record == nil {
		return nil
	}

	copied := &IdempotencyKey{
		Key:                record.Key,
		Method:             record.Method,
		Route:              record.Route,
		CreatedAt:          record.CreatedAt,
		ResponseHash:       record.ResponseHash,
		Status:             record.Status,
		ResponseBody:       record.ResponseBody,
		ResponseStatusCode: record.ResponseStatusCode,
	}

	if record.PaymentID != nil {
		paymentID := *record.PaymentID
		copied.PaymentID = &paymentID
	}

	return copied
}
