// Package membership provides models and repository for managing scene memberships
// with AT Protocol record tracking for idempotent ingestion.
package membership

import (
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/onnwee/subcults/internal/trust"
)

// Common errors for membership operations.
var (
	ErrMembershipNotFound = errors.New("membership not found")
)

// Membership represents a user's participation in a scene.
type Membership struct {
	ID          string  `json:"id"`
	SceneID     string  `json:"scene_id"`
	UserDID     string  `json:"user_did"`
	Role        string  `json:"role"`
	Status      string  `json:"status"`
	TrustWeight float64 `json:"trust_weight"`

	// AT Protocol record tracking
	RecordDID  *string `json:"record_did,omitempty"`
	RecordRKey *string `json:"record_rkey,omitempty"`

	Since     time.Time `json:"since"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// EffectiveWeight returns the effective trust weight for this membership.
// It computes: base TrustWeight * role multiplier.
func (m *Membership) EffectiveWeight() float64 {
	return m.TrustWeight * getRoleMultiplier(m.Role)
}

// getRoleMultiplier gets the multiplier for a role, using default if not found.
func getRoleMultiplier(role string) float64 {
	if multiplier, ok := trust.RoleMultiplier[role]; ok {
		return multiplier
	}
	return trust.DefaultRoleMultiplier
}

// ValidateRole validates the membership role.
// Returns an error if the role is invalid.
func (m *Membership) ValidateRole() error {
	if !trust.ValidRole(m.Role) {
		return trust.ErrInvalidRole
	}
	return nil
}

// ValidateTrustWeight validates the membership trust weight.
// Returns an error if the weight is out of bounds (0.0-1.0).
func (m *Membership) ValidateTrustWeight() error {
	return trust.ValidateTrustWeight(m.TrustWeight)
}

// Validate validates all membership fields that have constraints.
// Returns the first validation error encountered.
func (m *Membership) Validate() error {
	if err := m.ValidateRole(); err != nil {
		return err
	}
	if err := m.ValidateTrustWeight(); err != nil {
		return err
	}
	return nil
}

// UpsertResult tracks statistics for upsert operations.
type UpsertResult struct {
	Inserted bool   // True if new record was inserted
	ID       string // The UUID of the upserted record
}

// MembershipRepository defines the interface for membership data operations.
type MembershipRepository interface {
	// Upsert inserts a new membership or updates existing one based on (record_did, record_rkey).
	// Returns UpsertResult indicating whether insert or update occurred.
	Upsert(membership *Membership) (*UpsertResult, error)

	// GetByID retrieves a membership by its UUID.
	GetByID(id string) (*Membership, error)

	// GetByRecordKey retrieves a membership by its AT Protocol record key.
	GetByRecordKey(did, rkey string) (*Membership, error)

	// GetBySceneAndUser retrieves a membership by scene ID and user DID.
	GetBySceneAndUser(sceneID, userDID string) (*Membership, error)

	// UpdateStatus updates the status and optional since timestamp of a membership.
	// If since is nil, the timestamp is not updated.
	UpdateStatus(id, status string, since *time.Time) error

	// UpdateRole updates the role of a membership.
	// This affects the effective trust weight via the role multiplier.
	UpdateRole(id, role string) error

	// ListByScene retrieves all memberships for a scene, optionally filtered by status.
	// If status is empty, returns all memberships regardless of status.
	ListByScene(sceneID, status string) ([]*Membership, error)

	// CountByScenes returns a map of scene IDs to their membership counts.
	// Only counts memberships matching the specified status (empty string matches all).
	// This is a batch operation to avoid N+1 queries.
	CountByScenes(sceneIDs []string, status string) (map[string]int, error)
}

// InMemoryMembershipRepository is an in-memory implementation of MembershipRepository.
// Thread-safe via RWMutex.
type InMemoryMembershipRepository struct {
	mu          sync.RWMutex
	memberships map[string]*Membership // UUID -> Membership
	keys        map[string]string      // "did:rkey" -> UUID
}

// NewInMemoryMembershipRepository creates a new in-memory membership repository.
func NewInMemoryMembershipRepository() *InMemoryMembershipRepository {
	return &InMemoryMembershipRepository{
		memberships: make(map[string]*Membership),
		keys:        make(map[string]string),
	}
}

// makeKey creates a composite key from DID and rkey using a null byte separator to avoid collisions.
// AT Protocol DIDs contain colons (e.g., "did:plc:abc123"), so using a null byte prevents
// collisions like did="a:b" + rkey="c" vs did="a" + rkey="b:c" both producing "a:b:c".
func makeKey(did, rkey string) string {
	return did + "\x00" + rkey
}

// Upsert inserts a new membership or updates existing one based on (record_did, record_rkey).
func (r *InMemoryMembershipRepository) Upsert(membership *Membership) (*UpsertResult, error) {
	// Validate membership before upserting
	if err := membership.Validate(); err != nil {
		return nil, err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	var inserted bool
	var id string

	// Check if membership exists by record key
	if membership.RecordDID != nil && membership.RecordRKey != nil {
		key := makeKey(*membership.RecordDID, *membership.RecordRKey)
		existingID, exists := r.keys[key]

		if exists {
			// Update existing membership
			existing := r.memberships[existingID]
			existing.SceneID = membership.SceneID
			existing.UserDID = membership.UserDID
			existing.Role = membership.Role
			existing.Status = membership.Status
			existing.TrustWeight = membership.TrustWeight
			existing.UpdatedAt = now
			inserted = false
			id = existingID
		} else {
			// Insert new membership
			if membership.ID == "" {
				membership.ID = uuid.New().String()
			}
			if membership.Since.IsZero() {
				membership.Since = now
			}
			membership.CreatedAt = now
			membership.UpdatedAt = now

			membershipCopy := *membership
			r.memberships[membership.ID] = &membershipCopy
			r.keys[key] = membership.ID
			inserted = true
			id = membership.ID
		}
	} else {
		// No record key, always insert new with new UUID
		newID := uuid.New().String()
		membership.ID = newID
		if membership.Since.IsZero() {
			membership.Since = now
		}
		membership.CreatedAt = now
		membership.UpdatedAt = now

		membershipCopy := *membership
		r.memberships[newID] = &membershipCopy
		inserted = true
		id = newID
	}

	return &UpsertResult{
		Inserted: inserted,
		ID:       id,
	}, nil
}

// GetByID retrieves a membership by its UUID.
func (r *InMemoryMembershipRepository) GetByID(id string) (*Membership, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	membership, ok := r.memberships[id]
	if !ok {
		return nil, ErrMembershipNotFound
	}

	membershipCopy := *membership
	return &membershipCopy, nil
}

// GetByRecordKey retrieves a membership by its AT Protocol record key.
func (r *InMemoryMembershipRepository) GetByRecordKey(did, rkey string) (*Membership, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	key := makeKey(did, rkey)
	id, ok := r.keys[key]
	if !ok {
		return nil, ErrMembershipNotFound
	}

	membership := r.memberships[id]
	membershipCopy := *membership
	return &membershipCopy, nil
}

// GetBySceneAndUser retrieves a membership by scene ID and user DID.
func (r *InMemoryMembershipRepository) GetBySceneAndUser(sceneID, userDID string) (*Membership, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, membership := range r.memberships {
		if membership.SceneID == sceneID && membership.UserDID == userDID {
			membershipCopy := *membership
			return &membershipCopy, nil
		}
	}

	return nil, ErrMembershipNotFound
}

// UpdateStatus updates the status and optional since timestamp of a membership.
func (r *InMemoryMembershipRepository) UpdateStatus(id, status string, since *time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	membership, ok := r.memberships[id]
	if !ok {
		return ErrMembershipNotFound
	}

	membership.Status = status
	if since != nil {
		membership.Since = *since
	}
	membership.UpdatedAt = time.Now()

	return nil
}

// UpdateRole updates the role of a membership.
// Validates the role before updating.
func (r *InMemoryMembershipRepository) UpdateRole(id, role string) error {
	// Validate role before updating
	if !trust.ValidRole(role) {
		return trust.ErrInvalidRole
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	membership, ok := r.memberships[id]
	if !ok {
		return ErrMembershipNotFound
	}

	membership.Role = role
	membership.UpdatedAt = time.Now()

	return nil
}

// ListByScene retrieves all memberships for a scene, optionally filtered by status.
func (r *InMemoryMembershipRepository) ListByScene(sceneID, status string) ([]*Membership, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*Membership
	for _, membership := range r.memberships {
		if membership.SceneID == sceneID {
			if status == "" || membership.Status == status {
				membershipCopy := *membership
				result = append(result, &membershipCopy)
			}
		}
	}

	return result, nil
}

// CountByScenes returns a map of scene IDs to their membership counts.
// Only counts memberships matching the specified status (empty string matches all).
// This is a batch operation to avoid N+1 queries.
func (r *InMemoryMembershipRepository) CountByScenes(sceneIDs []string, status string) (map[string]int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Create a set of scene IDs for efficient lookup
	sceneIDSet := make(map[string]bool, len(sceneIDs))
	for _, id := range sceneIDs {
		sceneIDSet[id] = true
	}

	// Count memberships for each scene
	counts := make(map[string]int)
	for _, membership := range r.memberships {
		if sceneIDSet[membership.SceneID] {
			if status == "" || membership.Status == status {
				counts[membership.SceneID]++
			}
		}
	}

	return counts, nil
}
