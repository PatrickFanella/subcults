// Package alliance provides models and repository for managing alliances between scenes
// with AT Protocol record tracking for idempotent ingestion.
package alliance

import (
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Common errors for alliance operations.
var (
	ErrAllianceNotFound = errors.New("alliance not found")
	ErrAllianceDeleted  = errors.New("alliance deleted")
)

// Alliance represents a trust relationship between two scenes.
type Alliance struct {
	ID          string  `json:"id"`
	FromSceneID string  `json:"from_scene_id"`
	ToSceneID   string  `json:"to_scene_id"`
	Weight      float64 `json:"weight"`
	Status      string  `json:"status"`
	Reason      *string `json:"reason,omitempty"`

	// AT Protocol record tracking
	RecordDID  *string `json:"record_did,omitempty"`
	RecordRKey *string `json:"record_rkey,omitempty"`

	Since     time.Time  `json:"since"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
}

// UpsertResult tracks statistics for upsert operations.
type UpsertResult struct {
	Inserted bool   // True if new record was inserted
	ID       string // The UUID of the upserted record
}

// AllianceRepository defines the interface for alliance data operations.
type AllianceRepository interface {
	// Upsert inserts a new alliance or updates existing one based on (record_did, record_rkey).
	// Returns UpsertResult indicating whether insert or update occurred.
	Upsert(alliance *Alliance) (*UpsertResult, error)

	// GetByID retrieves an alliance by its UUID.
	// Returns ErrAllianceDeleted if alliance exists but is soft-deleted.
	GetByID(id string) (*Alliance, error)

	// GetByRecordKey retrieves an alliance by its AT Protocol record key.
	GetByRecordKey(did, rkey string) (*Alliance, error)

	// Insert creates a new alliance with the given data.
	Insert(alliance *Alliance) error

	// Update modifies an existing alliance.
	Update(alliance *Alliance) error

	// Delete soft-deletes an alliance by setting deleted_at.
	// Returns ErrAllianceDeleted if alliance is already deleted.
	Delete(id string) error
}

// InMemoryAllianceRepository is an in-memory implementation of AllianceRepository.
// Thread-safe via RWMutex.
type InMemoryAllianceRepository struct {
	mu        sync.RWMutex
	alliances map[string]*Alliance // UUID -> Alliance
	keys      map[string]string    // "did:rkey" -> UUID
}

// NewInMemoryAllianceRepository creates a new in-memory alliance repository.
func NewInMemoryAllianceRepository() *InMemoryAllianceRepository {
	return &InMemoryAllianceRepository{
		alliances: make(map[string]*Alliance),
		keys:      make(map[string]string),
	}
}

// makeKey creates a composite key from DID and rkey using a null byte separator to avoid collisions.
// AT Protocol DIDs contain colons (e.g., "did:plc:abc123"), so using a null byte prevents
// collisions like did="a:b" + rkey="c" vs did="a" + rkey="b:c" both producing "a:b:c".
func makeKey(did, rkey string) string {
	return did + "\x00" + rkey
}

// Upsert inserts a new alliance or updates existing one based on (record_did, record_rkey).
func (r *InMemoryAllianceRepository) Upsert(alliance *Alliance) (*UpsertResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	var inserted bool
	var id string

	// Check if alliance exists by record key
	if alliance.RecordDID != nil && alliance.RecordRKey != nil {
		key := makeKey(*alliance.RecordDID, *alliance.RecordRKey)
		existingID, exists := r.keys[key]

		if exists {
			// Update existing alliance
			existing := r.alliances[existingID]
			existing.FromSceneID = alliance.FromSceneID
			existing.ToSceneID = alliance.ToSceneID
			existing.Weight = alliance.Weight
			existing.Status = alliance.Status
			existing.Reason = alliance.Reason
			existing.UpdatedAt = now
			inserted = false
			id = existingID
		} else {
			// Insert new alliance
			if alliance.ID == "" {
				alliance.ID = uuid.New().String()
			}
			if alliance.Since.IsZero() {
				alliance.Since = now
			}
			alliance.CreatedAt = now
			alliance.UpdatedAt = now

			allianceCopy := *alliance
			r.alliances[alliance.ID] = &allianceCopy
			r.keys[key] = alliance.ID
			inserted = true
			id = alliance.ID
		}
	} else {
		// No record key, always insert new with new UUID
		newID := uuid.New().String()
		alliance.ID = newID
		if alliance.Since.IsZero() {
			alliance.Since = now
		}
		alliance.CreatedAt = now
		alliance.UpdatedAt = now

		allianceCopy := *alliance
		r.alliances[newID] = &allianceCopy
		inserted = true
		id = newID
	}

	return &UpsertResult{
		Inserted: inserted,
		ID:       id,
	}, nil
}

// GetByID retrieves an alliance by its UUID.
// Returns ErrAllianceDeleted if alliance exists but is soft-deleted.
func (r *InMemoryAllianceRepository) GetByID(id string) (*Alliance, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	alliance, ok := r.alliances[id]
	if !ok {
		return nil, ErrAllianceNotFound
	}

	if alliance.DeletedAt != nil {
		return nil, ErrAllianceDeleted
	}

	allianceCopy := *alliance
	return &allianceCopy, nil
}

// GetByRecordKey retrieves an alliance by its AT Protocol record key.
// Returns ErrAllianceDeleted if alliance exists but is soft-deleted.
func (r *InMemoryAllianceRepository) GetByRecordKey(did, rkey string) (*Alliance, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	key := makeKey(did, rkey)
	id, ok := r.keys[key]
	if !ok {
		return nil, ErrAllianceNotFound
	}

	alliance := r.alliances[id]
	if alliance.DeletedAt != nil {
		return nil, ErrAllianceDeleted
	}

	allianceCopy := *alliance
	return &allianceCopy, nil
}

// Insert creates a new alliance with the given data.
func (r *InMemoryAllianceRepository) Insert(alliance *Alliance) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Generate UUID if not set
	if alliance.ID == "" {
		alliance.ID = uuid.New().String()
	}

	// Set timestamps
	now := time.Now()
	alliance.CreatedAt = now
	alliance.UpdatedAt = now
	if alliance.Since.IsZero() {
		alliance.Since = now
	}

	// Store deep copy
	allianceCopy := *alliance
	r.alliances[alliance.ID] = &allianceCopy

	return nil
}

// Update modifies an existing alliance.
func (r *InMemoryAllianceRepository) Update(alliance *Alliance) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	existing, ok := r.alliances[alliance.ID]
	if !ok {
		return ErrAllianceNotFound
	}

	if existing.DeletedAt != nil {
		return ErrAllianceDeleted
	}

	// Create an updated copy to maintain consistent deep-copy behavior for stored data
	updated := *existing
	updated.Weight = alliance.Weight
	updated.Reason = alliance.Reason
	updated.Status = alliance.Status
	updated.UpdatedAt = time.Now()

	r.alliances[alliance.ID] = &updated

	return nil
}

// Delete soft-deletes an alliance by setting deleted_at.
// Returns ErrAllianceDeleted if alliance is already deleted.
func (r *InMemoryAllianceRepository) Delete(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	alliance, ok := r.alliances[id]
	if !ok {
		return ErrAllianceNotFound
	}

	if alliance.DeletedAt != nil {
		return ErrAllianceDeleted
	}

	now := time.Now()
	alliance.DeletedAt = &now

	return nil
}
