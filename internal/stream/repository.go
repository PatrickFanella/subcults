// Package stream provides models and repository for managing stream sessions
// with AT Protocol record tracking for idempotent ingestion.
package stream

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Common errors for stream session operations.
var (
	ErrStreamNotFound = errors.New("stream session not found")
)

// Session represents a LiveKit audio room streaming session.
type Session struct {
	ID               string  `json:"id"`
	SceneID          *string `json:"scene_id,omitempty"`
	EventID          *string `json:"event_id,omitempty"`
	RoomName         string  `json:"room_name"`
	HostDID          string  `json:"host_did"`
	ParticipantCount int     `json:"participant_count"` // Deprecated: use ActiveParticipantCount

	// Denormalized participant count for efficient queries
	ActiveParticipantCount int `json:"active_participant_count"`

	// AT Protocol record tracking
	RecordDID  *string `json:"record_did,omitempty"`
	RecordRKey *string `json:"record_rkey,omitempty"`

	// Analytics tracking for historical queries
	JoinCount  int `json:"join_count"`  // Total number of join events
	LeaveCount int `json:"leave_count"` // Total number of leave events

	// Organizer controls
	// Backed by DB columns `is_locked` and `featured_participant` on stream_sessions
	// (see migrations/000022_add_stream_organizer_controls.up.sql).
	IsLocked            bool    `json:"is_locked"`                      // When true, new participants cannot join
	FeaturedParticipant *string `json:"featured_participant,omitempty"` // ParticipantID of featured/spotlighted speaker

	StartedAt time.Time  `json:"started_at"`
	EndedAt   *time.Time `json:"ended_at,omitempty"`
}

// UpsertResult tracks statistics for upsert operations.
type UpsertResult struct {
	Inserted bool   // True if new record was inserted
	ID       string // The UUID of the upserted record
}

// ActiveStreamInfo represents active stream information for event payload serialization.
// Only includes fields needed for frontend display.
type ActiveStreamInfo struct {
	StreamSessionID string    `json:"stream_session_id"`
	RoomName        string    `json:"room_name"`
	StartedAt       time.Time `json:"started_at"`
}

// SessionRepository defines the interface for stream session data operations.
type SessionRepository interface {
	// Upsert inserts a new session or updates existing one based on (record_did, record_rkey).
	// Returns UpsertResult indicating whether insert or update occurred.
	Upsert(session *Session) (*UpsertResult, error)

	// GetByID retrieves a session by its UUID.
	GetByID(id string) (*Session, error)

	// GetByRecordKey retrieves a session by its AT Protocol record key.
	GetByRecordKey(did, rkey string) (*Session, error)

	// CreateStreamSession creates a new stream session with automatic room naming and UUID generation.
	// One of sceneID or eventID must be provided. Returns the session ID and room name.
	CreateStreamSession(sceneID *string, eventID *string, hostDID string) (id string, roomName string, err error)

	// EndStreamSession marks a stream session as ended by setting ended_at timestamp.
	// Returns ErrStreamNotFound if session doesn't exist.
	// Idempotent: returns nil if session is already ended.
	EndStreamSession(id string) error

	// RecordJoin increments the join count for a stream session.
	// Returns ErrStreamNotFound if session doesn't exist.
	RecordJoin(id string) error

	// RecordLeave increments the leave count for a stream session.
	// Returns ErrStreamNotFound if session doesn't exist.
	RecordLeave(id string) error

	// UpdateActiveParticipantCount updates the denormalized active_participant_count.
	// Returns ErrStreamNotFound if session doesn't exist.
	UpdateActiveParticipantCount(id string, count int) error

	// SetLockStatus updates the is_locked status for a stream session.
	// Returns ErrStreamNotFound if session doesn't exist.
	SetLockStatus(id string, locked bool) error

	// SetFeaturedParticipant sets or clears the featured participant for a stream session.
	// Pass nil participantID to clear the featured participant.
	// Returns ErrStreamNotFound if session doesn't exist.
	SetFeaturedParticipant(id string, participantID *string) error

	// HasActiveStreamForScene checks if there's an active stream (ended_at IS NULL) for the given scene.
	HasActiveStreamForScene(sceneID string) (bool, error)

	// HasActiveStreamsForScenes returns a map of scene IDs to their active stream status.
	// Returns true for scenes with at least one active stream (ended_at IS NULL).
	// This is a batch operation to avoid N+1 queries.
	HasActiveStreamsForScenes(sceneIDs []string) (map[string]bool, error)

	// GetActiveStreamForEvent retrieves the active stream (ended_at IS NULL) for a given event.
	// Returns nil if no active stream exists for the event.
	GetActiveStreamForEvent(eventID string) (*ActiveStreamInfo, error)

	// GetActiveStreamsForEvents returns a map of event IDs to their active stream info.
	// Only includes events with active streams (ended_at IS NULL).
	// This is a batch operation to avoid N+1 queries.
	GetActiveStreamsForEvents(eventIDs []string) (map[string]*ActiveStreamInfo, error)
}

// InMemorySessionRepository is an in-memory implementation of SessionRepository.
// Thread-safe via RWMutex.
type InMemorySessionRepository struct {
	mu       sync.RWMutex
	sessions map[string]*Session // UUID -> Session
	keys     map[string]string   // "did:rkey" -> UUID
}

// NewInMemorySessionRepository creates a new in-memory session repository.
func NewInMemorySessionRepository() *InMemorySessionRepository {
	return &InMemorySessionRepository{
		sessions: make(map[string]*Session),
		keys:     make(map[string]string),
	}
}

// makeKey creates a composite key from DID and rkey using a null byte separator to avoid collisions.
// AT Protocol DIDs contain colons (e.g., "did:plc:abc123"), so using a null byte prevents
// collisions like did="a:b" + rkey="c" vs did="a" + rkey="b:c" both producing "a:b:c".
func makeKey(did, rkey string) string {
	return did + "\x00" + rkey
}

// Upsert inserts a new session or updates existing one based on (record_did, record_rkey).
func (r *InMemorySessionRepository) Upsert(session *Session) (*UpsertResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	var inserted bool
	var id string

	// Check if session exists by record key
	if session.RecordDID != nil && session.RecordRKey != nil {
		key := makeKey(*session.RecordDID, *session.RecordRKey)
		existingID, exists := r.keys[key]

		if exists {
			// Update existing session
			existing := r.sessions[existingID]
			existing.SceneID = session.SceneID
			existing.EventID = session.EventID
			existing.RoomName = session.RoomName
			existing.HostDID = session.HostDID
			existing.ParticipantCount = session.ParticipantCount
			existing.EndedAt = session.EndedAt
			inserted = false
			id = existingID
		} else {
			// Insert new session
			if session.ID == "" {
				session.ID = uuid.New().String()
			}
			if session.StartedAt.IsZero() {
				session.StartedAt = now
			}

			sessionCopy := *session
			r.sessions[session.ID] = &sessionCopy
			r.keys[key] = session.ID
			inserted = true
			id = session.ID
		}
	} else {
		// No record key, always insert new with new UUID
		newID := uuid.New().String()
		session.ID = newID
		if session.StartedAt.IsZero() {
			session.StartedAt = now
		}

		sessionCopy := *session
		r.sessions[newID] = &sessionCopy
		inserted = true
		id = newID
	}

	return &UpsertResult{
		Inserted: inserted,
		ID:       id,
	}, nil
}

// GetByID retrieves a session by its UUID.
func (r *InMemorySessionRepository) GetByID(id string) (*Session, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	session, ok := r.sessions[id]
	if !ok {
		return nil, ErrStreamNotFound
	}

	sessionCopy := *session
	return &sessionCopy, nil
}

// GetByRecordKey retrieves a session by its AT Protocol record key.
func (r *InMemorySessionRepository) GetByRecordKey(did, rkey string) (*Session, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	key := makeKey(did, rkey)
	id, ok := r.keys[key]
	if !ok {
		return nil, ErrStreamNotFound
	}

	session := r.sessions[id]
	sessionCopy := *session
	return &sessionCopy, nil
}

// HasActiveStreamForScene checks if there's an active stream (ended_at IS NULL) for the given scene.
func (r *InMemorySessionRepository) HasActiveStreamForScene(sceneID string) (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, session := range r.sessions {
		if session.SceneID != nil && *session.SceneID == sceneID && session.EndedAt == nil {
			return true, nil
		}
	}
	return false, nil
}

// HasActiveStreamsForScenes returns a map of scene IDs to their active stream status.
// Returns true for scenes with at least one active stream (ended_at IS NULL).
// This is a batch operation to avoid N+1 queries.
func (r *InMemorySessionRepository) HasActiveStreamsForScenes(sceneIDs []string) (map[string]bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Create a set of scene IDs for efficient lookup
	sceneIDSet := make(map[string]bool, len(sceneIDs))
	for _, id := range sceneIDs {
		sceneIDSet[id] = true
	}

	// Initialize result map with false for all scenes
	result := make(map[string]bool, len(sceneIDs))
	for _, id := range sceneIDs {
		result[id] = false
	}

	// Check for active streams
	for _, session := range r.sessions {
		if session.SceneID != nil && sceneIDSet[*session.SceneID] && session.EndedAt == nil {
			result[*session.SceneID] = true
		}
	}

	return result, nil
}

// CreateStreamSession creates a new stream session with automatic room naming and UUID generation.
// One of sceneID or eventID must be provided. Returns the session ID and room name.
func (r *InMemorySessionRepository) CreateStreamSession(sceneID *string, eventID *string, hostDID string) (id string, roomName string, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Validate that hostDID is not empty
	if hostDID == "" {
		return "", "", errors.New("hostDID must not be empty")
	}

	// Validate that at least one of sceneID or eventID is provided
	if (sceneID == nil || *sceneID == "") && (eventID == nil || *eventID == "") {
		return "", "", errors.New("either scene_id or event_id must be provided")
	}

	// Generate room name using naming convention: scene-{sceneId}-{timestamp} or event-{eventId}-{timestamp}
	now := time.Now()
	timestamp := now.Unix()

	if sceneID != nil && *sceneID != "" {
		roomName = fmt.Sprintf("scene-%s-%d", *sceneID, timestamp)
	} else {
		roomName = fmt.Sprintf("event-%s-%d", *eventID, timestamp)
	}

	// Create new session
	newID := uuid.New().String()
	session := &Session{
		ID:                     newID,
		SceneID:                sceneID,
		EventID:                eventID,
		RoomName:               roomName,
		HostDID:                hostDID,
		ParticipantCount:       0,
		ActiveParticipantCount: 0,
		JoinCount:              0,
		LeaveCount:             0,
		IsLocked:               false,
		FeaturedParticipant:    nil,
		StartedAt:              now,
		EndedAt:                nil, // Active stream
	}

	r.sessions[newID] = session
	return newID, roomName, nil
}

// EndStreamSession marks a stream session as ended by setting ended_at timestamp.
// Returns ErrStreamNotFound if session doesn't exist.
// Idempotent: returns nil if session is already ended.
func (r *InMemorySessionRepository) EndStreamSession(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	session, ok := r.sessions[id]
	if !ok {
		return ErrStreamNotFound
	}

	// Idempotent: if already ended, return success
	if session.EndedAt != nil {
		return nil
	}

	// Set ended_at timestamp
	now := time.Now()
	session.EndedAt = &now

	return nil
}

// RecordJoin increments the join count for a stream session.
// Returns ErrStreamNotFound if session doesn't exist.
func (r *InMemorySessionRepository) RecordJoin(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	session, ok := r.sessions[id]
	if !ok {
		return ErrStreamNotFound
	}

	session.JoinCount++
	return nil
}

// RecordLeave increments the leave count for a stream session.
// Returns ErrStreamNotFound if session doesn't exist.
func (r *InMemorySessionRepository) RecordLeave(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	session, ok := r.sessions[id]
	if !ok {
		return ErrStreamNotFound
	}

	session.LeaveCount++
	return nil
}

// UpdateActiveParticipantCount updates the denormalized active_participant_count.
// Returns ErrStreamNotFound if session doesn't exist.
func (r *InMemorySessionRepository) UpdateActiveParticipantCount(id string, count int) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	session, ok := r.sessions[id]
	if !ok {
		return ErrStreamNotFound
	}

	session.ActiveParticipantCount = count
	return nil
}

// SetLockStatus updates the is_locked status for a stream session.
// Returns ErrStreamNotFound if session doesn't exist.
func (r *InMemorySessionRepository) SetLockStatus(id string, locked bool) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	session, ok := r.sessions[id]
	if !ok {
		return ErrStreamNotFound
	}

	session.IsLocked = locked
	return nil
}

// SetFeaturedParticipant sets or clears the featured participant for a stream session.
// Pass nil participantID to clear the featured participant.
// Returns ErrStreamNotFound if session doesn't exist.
func (r *InMemorySessionRepository) SetFeaturedParticipant(id string, participantID *string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	session, ok := r.sessions[id]
	if !ok {
		return ErrStreamNotFound
	}

	session.FeaturedParticipant = participantID
	return nil
}

// HasActiveStreamForScene checks if there's an active stream (ended_at IS NULL) for the given scene.
// Returns nil if no active stream exists for the event.
// If multiple active streams exist, returns the most recent by started_at.
func (r *InMemorySessionRepository) GetActiveStreamForEvent(eventID string) (*ActiveStreamInfo, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var best *ActiveStreamInfo

	for _, session := range r.sessions {
		if session.EventID != nil && *session.EventID == eventID && session.EndedAt == nil {
			// If multiple active streams exist for this event, use the most recent one
			if best == nil || session.StartedAt.After(best.StartedAt) {
				best = &ActiveStreamInfo{
					StreamSessionID: session.ID,
					RoomName:        session.RoomName,
					StartedAt:       session.StartedAt,
				}
			}
		}
	}

	return best, nil
}

// GetActiveStreamsForEvents returns a map of event IDs to their active stream info.
// Only includes events with active streams (ended_at IS NULL).
// This is a batch operation to avoid N+1 queries.
func (r *InMemorySessionRepository) GetActiveStreamsForEvents(eventIDs []string) (map[string]*ActiveStreamInfo, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Create a set of event IDs for efficient lookup
	eventIDSet := make(map[string]bool, len(eventIDs))
	for _, id := range eventIDs {
		eventIDSet[id] = true
	}

	// Initialize result map
	result := make(map[string]*ActiveStreamInfo)

	// Find active streams for each event
	for _, session := range r.sessions {
		if session.EventID != nil && eventIDSet[*session.EventID] && session.EndedAt == nil {
			// If multiple active streams exist for an event, use the most recent one
			eventID := *session.EventID
			existing, exists := result[eventID]
			if !exists || session.StartedAt.After(existing.StartedAt) {
				result[eventID] = &ActiveStreamInfo{
					StreamSessionID: session.ID,
					RoomName:        session.RoomName,
					StartedAt:       session.StartedAt,
				}
			}
		}
	}

	return result, nil
}
