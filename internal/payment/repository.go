// Package payment provides repository for payment record persistence.
package payment

import (
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ErrPaymentRecordNotFound is returned when a payment record is not found.
var ErrPaymentRecordNotFound = errors.New("payment record not found")

// ErrInvalidStatusTransition is returned when an invalid status transition is attempted.
var ErrInvalidStatusTransition = errors.New("invalid status transition")

// ErrDuplicateSessionID is returned when attempting to create a payment with a duplicate session ID.
var ErrDuplicateSessionID = errors.New("duplicate session ID")

// ErrSessionIDImmutable is returned when attempting to change the session ID of an existing payment.
var ErrSessionIDImmutable = errors.New("session ID cannot be changed")

// ErrPaymentIntentMismatch is returned when marking a payment as completed with a different payment intent ID.
var ErrPaymentIntentMismatch = errors.New("payment intent ID mismatch")

// PaymentRepository defines methods for payment record persistence.
type PaymentRepository interface {
	// GetByID retrieves a payment record by ID.
	GetByID(id string) (*PaymentRecord, error)

	// GetBySessionID retrieves a payment record by session ID.
	GetBySessionID(sessionID string) (*PaymentRecord, error)

	// CreatePending creates a new payment record in pending status.
	// Returns ErrDuplicateSessionID if a record with the same session_id already exists.
	CreatePending(record *PaymentRecord) error

	// MarkCompleted transitions a payment from pending to succeeded status.
	// Returns ErrPaymentRecordNotFound if the session doesn't exist.
	// Returns ErrInvalidStatusTransition if the payment is not in pending status.
	// Returns ErrPaymentIntentMismatch if already succeeded with a different payment intent ID.
	// Idempotent: returns nil if already in succeeded status with the same payment intent ID.
	MarkCompleted(sessionID, paymentIntentID string) error

	// MarkFailed transitions a payment from pending to failed status.
	// Returns ErrPaymentRecordNotFound if the session doesn't exist.
	// Returns ErrInvalidStatusTransition if the payment is not in pending status.
	// Idempotent: returns nil if already in failed status with the same reason.
	MarkFailed(sessionID, reason string) error

	// MarkCanceled transitions a payment from pending to canceled status.
	// Returns ErrPaymentRecordNotFound if the session doesn't exist.
	// Returns ErrInvalidStatusTransition if the payment is not in pending status.
	// Idempotent: returns nil if already in canceled status.
	MarkCanceled(sessionID string) error

	// MarkRefunded transitions a payment from succeeded to refunded status.
	// Returns ErrPaymentRecordNotFound if the session doesn't exist.
	// Returns ErrInvalidStatusTransition if the payment is not in succeeded status.
	// Idempotent: returns nil if already in refunded status.
	MarkRefunded(sessionID string) error
}

// InMemoryPaymentRepository implements PaymentRepository with in-memory storage.
type InMemoryPaymentRepository struct {
	mu       sync.RWMutex
	records  map[string]*PaymentRecord
	sessions map[string]string // Maps session_id -> record ID for uniqueness
}

// NewInMemoryPaymentRepository creates a new in-memory payment repository.
func NewInMemoryPaymentRepository() *InMemoryPaymentRepository {
	return &InMemoryPaymentRepository{
		records:  make(map[string]*PaymentRecord),
		sessions: make(map[string]string),
	}
}

// isValidStatusTransition validates if a status transition is allowed.
// Valid transitions:
//
//	pending -> succeeded, failed, canceled
//	succeeded -> refunded
//	All other transitions are invalid (e.g., succeeded -> failed, failed -> succeeded)
func isValidStatusTransition(from, to string) bool {
	switch from {
	case StatusPending:
		return to == StatusSucceeded || to == StatusFailed || to == StatusCanceled
	case StatusSucceeded:
		return to == StatusRefunded
	default:
		return false
	}
}

// insert is a private method that adds a new payment record.
// This should not be exposed in the interface to enforce state machine transitions.
func (r *InMemoryPaymentRepository) insert(record *PaymentRecord) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check for duplicate session ID
	if _, exists := r.sessions[record.SessionID]; exists {
		return ErrDuplicateSessionID
	}

	if record.ID == "" {
		record.ID = uuid.New().String()
	}

	// Apply default currency if not set
	if record.Currency == "" {
		record.Currency = "usd"
	}

	// Set timestamps for new record
	now := time.Now()
	if record.CreatedAt == nil {
		record.CreatedAt = &now
	}
	if record.UpdatedAt == nil {
		record.UpdatedAt = &now
	}

	// Deep copy to prevent external mutation
	copied := record.DeepCopy()
	r.records[record.ID] = copied
	r.sessions[record.SessionID] = record.ID

	return nil
}

// GetByID retrieves a payment record by ID.
func (r *InMemoryPaymentRepository) GetByID(id string) (*PaymentRecord, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	record, ok := r.records[id]
	if !ok {
		return nil, ErrPaymentRecordNotFound
	}

	// Deep copy to prevent external mutation
	return record.DeepCopy(), nil
}

// GetBySessionID retrieves a payment record by session ID.
func (r *InMemoryPaymentRepository) GetBySessionID(sessionID string) (*PaymentRecord, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Use sessions map for O(1) lookup
	recordID, ok := r.sessions[sessionID]
	if !ok {
		return nil, ErrPaymentRecordNotFound
	}

	record := r.records[recordID]
	// Deep copy to prevent external mutation
	return record.DeepCopy(), nil
}

// update is a private method that updates an existing payment record with status transition validation.
// This should not be exposed in the interface to enforce state machine transitions.
func (r *InMemoryPaymentRepository) update(record *PaymentRecord) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	existing, ok := r.records[record.ID]
	if !ok {
		return ErrPaymentRecordNotFound
	}

	// Prevent changing session ID
	if existing.SessionID != record.SessionID {
		return ErrSessionIDImmutable
	}

	// Validate status transition if status changed
	if existing.Status != record.Status {
		if !isValidStatusTransition(existing.Status, record.Status) {
			return ErrInvalidStatusTransition
		}
	}

	// Update the UpdatedAt timestamp
	now := time.Now()
	record.UpdatedAt = &now

	// Deep copy to prevent external mutation
	copied := record.DeepCopy()
	r.records[record.ID] = copied

	return nil
}

// CreatePending creates a new payment record in pending status.
// Returns ErrDuplicateSessionID if a record with the same session_id already exists.
func (r *InMemoryPaymentRepository) CreatePending(record *PaymentRecord) error {
	// Force status to pending for safety
	record.Status = StatusPending
	return r.insert(record)
}

// MarkCompleted transitions a payment from pending to succeeded status.
// Returns ErrPaymentRecordNotFound if the session doesn't exist.
// Returns ErrInvalidStatusTransition if the payment is not in pending status.
// Returns ErrPaymentIntentMismatch if already succeeded with a different payment intent ID.
// Idempotent: returns nil if already in succeeded status with the same payment intent ID.
func (r *InMemoryPaymentRepository) MarkCompleted(sessionID, paymentIntentID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	recordID, ok := r.sessions[sessionID]
	if !ok {
		return ErrPaymentRecordNotFound
	}

	record := r.records[recordID]

	// Idempotent: if already succeeded, check payment intent ID
	if record.Status == StatusSucceeded {
		if record.PaymentIntentID != nil && *record.PaymentIntentID != paymentIntentID {
			return ErrPaymentIntentMismatch
		}
		return nil
	}

	// Validate status transition
	if !isValidStatusTransition(record.Status, StatusSucceeded) {
		return ErrInvalidStatusTransition
	}

	// Update status and payment intent ID
	record.Status = StatusSucceeded
	record.PaymentIntentID = &paymentIntentID
	now := time.Now()
	record.UpdatedAt = &now

	return nil
}

// MarkFailed transitions a payment from pending to failed status.
// Returns ErrPaymentRecordNotFound if the session doesn't exist.
// Returns ErrInvalidStatusTransition if the payment is not in pending status.
// Idempotent: returns nil if already in failed status with the same reason.
func (r *InMemoryPaymentRepository) MarkFailed(sessionID, reason string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	recordID, ok := r.sessions[sessionID]
	if !ok {
		return ErrPaymentRecordNotFound
	}

	record := r.records[recordID]

	// Idempotent: if already failed with same reason, return success
	if record.Status == StatusFailed {
		if record.FailureReason != nil && *record.FailureReason == reason {
			return nil
		}
		// If already failed but with different reason, update the reason
		record.FailureReason = &reason
		now := time.Now()
		record.UpdatedAt = &now
		return nil
	}

	// Validate status transition
	if !isValidStatusTransition(record.Status, StatusFailed) {
		return ErrInvalidStatusTransition
	}

	// Update status and failure reason
	record.Status = StatusFailed
	record.FailureReason = &reason
	now := time.Now()
	record.UpdatedAt = &now

	return nil
}

// MarkCanceled transitions a payment from pending to canceled status.
// Returns ErrPaymentRecordNotFound if the session doesn't exist.
// Returns ErrInvalidStatusTransition if the payment is not in pending status.
// Idempotent: returns nil if already in canceled status.
func (r *InMemoryPaymentRepository) MarkCanceled(sessionID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	recordID, ok := r.sessions[sessionID]
	if !ok {
		return ErrPaymentRecordNotFound
	}

	record := r.records[recordID]

	// Idempotent: if already canceled, return success
	if record.Status == StatusCanceled {
		return nil
	}

	// Validate status transition
	if !isValidStatusTransition(record.Status, StatusCanceled) {
		return ErrInvalidStatusTransition
	}

	// Update status
	record.Status = StatusCanceled
	now := time.Now()
	record.UpdatedAt = &now

	return nil
}

// MarkRefunded transitions a payment from succeeded to refunded status.
// Returns ErrPaymentRecordNotFound if the session doesn't exist.
// Returns ErrInvalidStatusTransition if the payment is not in succeeded status.
// Idempotent: returns nil if already in refunded status.
func (r *InMemoryPaymentRepository) MarkRefunded(sessionID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	recordID, ok := r.sessions[sessionID]
	if !ok {
		return ErrPaymentRecordNotFound
	}

	record := r.records[recordID]

	// Idempotent: if already refunded, return success
	if record.Status == StatusRefunded {
		return nil
	}

	// Validate status transition
	if !isValidStatusTransition(record.Status, StatusRefunded) {
		return ErrInvalidStatusTransition
	}

	// Update status
	record.Status = StatusRefunded
	now := time.Now()
	record.UpdatedAt = &now

	return nil
}
