package audit

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Repository defines the interface for audit log operations.
type Repository interface {
	// LogAccess records an access event to the audit log.
	// Returns the created audit log entry.
	LogAccess(entry LogEntry) (*AuditLog, error)

	// QueryByEntity retrieves audit logs for a specific entity, sorted by time (newest first).
	// Limit specifies the maximum number of entries to return (0 = no limit).
	QueryByEntity(entityType, entityID string, limit int) ([]*AuditLog, error)

	// QueryByUser retrieves audit logs for a specific user, sorted by time (newest first).
	// Limit specifies the maximum number of entries to return (0 = no limit).
	QueryByUser(userDID string, limit int) ([]*AuditLog, error)

	// GetLastHash returns the hash of the most recent audit log entry.
	// Returns empty string if no logs exist.
	GetLastHash() (string, error)

	// VerifyHashChain verifies the integrity of the hash chain.
	// Returns true if the chain is valid, false otherwise.
	VerifyHashChain() (bool, error)
}

// InMemoryRepository is an in-memory implementation of Repository.
// Used for testing and development. Thread-safe via RWMutex.
type InMemoryRepository struct {
	mu       sync.RWMutex
	logs     map[string]*AuditLog
	order    []string // Maintain insertion order for queries
	lastHash string   // Hash of the most recent log entry
}

// NewInMemoryRepository creates a new in-memory audit repository.
func NewInMemoryRepository() *InMemoryRepository {
	return &InMemoryRepository{
		logs:     make(map[string]*AuditLog),
		order:    make([]string, 0),
		lastHash: "",
	}
}

// computeHash computes the SHA-256 hash of a log entry combined with the previous hash.
// This creates a tamper-evident chain where any modification to a log entry or its order
// will invalidate all subsequent hashes.
func computeHash(log *AuditLog) string {
	// Concatenate all fields to create a string representation
	data := fmt.Sprintf("%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s",
		log.ID,
		log.UserDID,
		log.EntityType,
		log.EntityID,
		log.Action,
		log.Outcome,
		log.CreatedAt.Format(time.RFC3339Nano),
		log.RequestID,
		log.IPAddress,
		log.UserAgent,
		log.PreviousHash,
	)

	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// LogAccess records an access event to the audit log with hash chain support.
func (r *InMemoryRepository) LogAccess(entry LogEntry) (*AuditLog, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Default outcome to success if not provided
	outcome := entry.Outcome
	if outcome == "" {
		outcome = OutcomeSuccess
	}

	log := &AuditLog{
		ID:           uuid.New().String(),
		UserDID:      entry.UserDID,
		EntityType:   entry.EntityType,
		EntityID:     entry.EntityID,
		Action:       entry.Action,
		Outcome:      outcome,
		CreatedAt:    time.Now().UTC(),
		RequestID:    entry.RequestID,
		IPAddress:    entry.IPAddress,
		UserAgent:    entry.UserAgent,
		PreviousHash: r.lastHash, // Link to previous log entry
	}

	// Compute hash for this log entry (including previous hash)
	currentHash := computeHash(log)

	// Store the log and update last hash
	r.logs[log.ID] = log
	r.order = append(r.order, log.ID)
	r.lastHash = currentHash

	// Return a copy to prevent external modification
	logCopy := *log
	return &logCopy, nil
}

// QueryByEntity retrieves audit logs for a specific entity, sorted by time (newest first).
func (r *InMemoryRepository) QueryByEntity(entityType, entityID string, limit int) ([]*AuditLog, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var results []*AuditLog

	// Iterate in reverse order (newest first)
	for i := len(r.order) - 1; i >= 0; i-- {
		id := r.order[i]
		log := r.logs[id]

		if log.EntityType == entityType && log.EntityID == entityID {
			// Create a copy to prevent external modification
			logCopy := *log
			results = append(results, &logCopy)

			if limit > 0 && len(results) >= limit {
				break
			}
		}
	}

	return results, nil
}

// QueryByUser retrieves audit logs for a specific user, sorted by time (newest first).
func (r *InMemoryRepository) QueryByUser(userDID string, limit int) ([]*AuditLog, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var results []*AuditLog

	// Iterate in reverse order (newest first)
	for i := len(r.order) - 1; i >= 0; i-- {
		id := r.order[i]
		log := r.logs[id]

		if log.UserDID == userDID {
			// Create a copy to prevent external modification
			logCopy := *log
			results = append(results, &logCopy)

			if limit > 0 && len(results) >= limit {
				break
			}
		}
	}

	return results, nil
}

// GetLastHash returns the hash of the most recent audit log entry.
// Returns empty string if no logs exist.
func (r *InMemoryRepository) GetLastHash() (string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.lastHash, nil
}

// VerifyHashChain verifies the integrity of the hash chain.
// Returns true if the chain is valid, false otherwise.
// This checks that each log entry's hash correctly links to the previous entry.
func (r *InMemoryRepository) VerifyHashChain() (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if len(r.order) == 0 {
		return true, nil // Empty chain is valid
	}

	var expectedPreviousHash string

	// Iterate through logs in insertion order
	for _, id := range r.order {
		log := r.logs[id]

		// Verify that the previous hash matches what we expect
		if log.PreviousHash != expectedPreviousHash {
			return false, nil
		}

		// Compute what the hash should be for this entry
		expectedPreviousHash = computeHash(log)
	}

	// Verify that the last hash matches the stored lastHash
	if expectedPreviousHash != r.lastHash {
		return false, nil
	}

	return true, nil
}
