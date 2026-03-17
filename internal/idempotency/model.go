// Package idempotency provides models and services for idempotency key management.
package idempotency

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"
)

// Status constants for idempotency keys.
//
// StatusCompleted indicates that the request has finished and a stable response
// has been persisted. This is the only status currently used by the Go code.
//
// StatusProcessing is reserved for future use when implementing proper concurrent
// request handling for idempotent operations (e.g., marking a key as "processing"
// while the first request is still in-flight to prevent the race condition described
// in the middleware). It is already referenced in the database schema CHECK constraint,
// so it must remain in sync with that constraint even if unused in the Go code.
// Do not remove without updating the migrations.
const (
	StatusProcessing = "processing"
	StatusCompleted  = "completed"
)

var (
	// ErrKeyNotFound is returned when an idempotency key is not found.
	ErrKeyNotFound = errors.New("idempotency key not found")

	// ErrKeyExists is returned when attempting to create a duplicate key.
	ErrKeyExists = errors.New("idempotency key already exists")

	// ErrInvalidKey is returned when the key is invalid.
	ErrInvalidKey = errors.New("invalid idempotency key")

	// ErrKeyTooLong is returned when the key exceeds maximum length.
	ErrKeyTooLong = errors.New("idempotency key exceeds maximum length of 64 characters")
)

// MaxKeyLength is the maximum allowed length for an idempotency key.
const MaxKeyLength = 64

// IdempotencyKey represents a stored idempotency key with cached response.
type IdempotencyKey struct {
	Key                string    `json:"key"`
	Method             string    `json:"method"`
	Route              string    `json:"route"`
	CreatedAt          time.Time `json:"created_at"`
	PaymentID          *string   `json:"payment_id,omitempty"`
	ResponseHash       string    `json:"response_hash"`
	Status             string    `json:"status"`
	ResponseBody       string    `json:"response_body"`
	ResponseStatusCode int       `json:"response_status_code"`
}

// ValidateKey checks if an idempotency key is valid.
// Returns ErrInvalidKey if the key is empty.
// Returns ErrKeyTooLong if the key exceeds MaxKeyLength.
func ValidateKey(key string) error {
	if key == "" {
		return ErrInvalidKey
	}
	if len(key) > MaxKeyLength {
		return ErrKeyTooLong
	}
	return nil
}

// ComputeResponseHash computes a SHA256 hash of the response body.
// This is used to verify response integrity when returning cached responses.
func ComputeResponseHash(responseBody string) string {
	hash := sha256.Sum256([]byte(responseBody))
	return hex.EncodeToString(hash[:])
}

// Repository defines methods for idempotency key persistence.
type Repository interface {
	// Get retrieves an idempotency key by its key value.
	// Returns ErrKeyNotFound if the key doesn't exist.
	Get(key string) (*IdempotencyKey, error)

	// Store saves a new idempotency key.
	// Returns ErrKeyExists if the key already exists.
	Store(record *IdempotencyKey) error

	// DeleteOlderThan removes idempotency keys older than the specified duration.
	// This is used for cleanup jobs to prevent unbounded storage growth.
	DeleteOlderThan(duration time.Duration) (int64, error)
}
