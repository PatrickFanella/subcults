// Package stream provides repository for managing stream participants.
package stream

import (
	"errors"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Participant-specific errors
var (
	ErrParticipantNotFound      = errors.New("participant not found")
	ErrParticipantAlreadyActive = errors.New("participant already active in stream")
)

// ParticipantRepository defines the interface for participant data operations.
type ParticipantRepository interface {
	// RecordJoin records a participant joining a stream.
	// Returns the participant record and whether this is a reconnection.
	// If the participant is already active, returns ErrParticipantAlreadyActive.
	RecordJoin(streamSessionID, participantID, userDID string) (*Participant, bool, error)

	// RecordLeave marks a participant as having left the stream.
	// Sets left_at timestamp and updates the active participant count.
	// Returns ErrParticipantNotFound if participant doesn't exist or is already left.
	RecordLeave(streamSessionID, participantID string) error

	// GetActiveParticipants returns all currently active participants for a stream.
	// Active participants have left_at = NULL.
	GetActiveParticipants(streamSessionID string) ([]*Participant, error)

	// GetParticipantHistory returns all participants (active and past) for a stream.
	// Ordered by joined_at descending (most recent first).
	GetParticipantHistory(streamSessionID string) ([]*Participant, error)

	// GetActiveCount returns the count of currently active participants.
	GetActiveCount(streamSessionID string) (int, error)

	// UpdateSessionParticipantCount updates the denormalized active_participant_count
	// on the stream_sessions table. Should be called after join/leave operations.
	UpdateSessionParticipantCount(streamSessionID string, count int) error
}

// InMemoryParticipantRepository is an in-memory implementation of ParticipantRepository.
// Thread-safe via RWMutex.
type InMemoryParticipantRepository struct {
	mu           sync.RWMutex
	participants map[string]*Participant // participant.ID -> Participant
	// Index for quick lookup of active participants by stream and participant_id
	activeIndex map[string]map[string]string // streamSessionID -> participantID -> participant.ID
	sessionRepo SessionRepository            // Reference for updating denormalized count
}

// NewInMemoryParticipantRepository creates a new in-memory participant repository.
func NewInMemoryParticipantRepository(sessionRepo SessionRepository) *InMemoryParticipantRepository {
	return &InMemoryParticipantRepository{
		participants: make(map[string]*Participant),
		activeIndex:  make(map[string]map[string]string),
		sessionRepo:  sessionRepo,
	}
}

// RecordJoin records a participant joining a stream.
func (r *InMemoryParticipantRepository) RecordJoin(streamSessionID, participantID, userDID string) (*Participant, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	isReconnection := false

	// Check if participant is already active
	if streamActive, exists := r.activeIndex[streamSessionID]; exists {
		if _, active := streamActive[participantID]; active {
			return nil, false, ErrParticipantAlreadyActive
		}
	}

	// Check if this participant has been in this stream before (reconnection)
	// Find the maximum reconnection count from all previous records
	var reconnectionCount int
	for _, p := range r.participants {
		if p.StreamSessionID == streamSessionID && p.ParticipantID == participantID {
			isReconnection = true
			if candidate := p.ReconnectionCount + 1; candidate > reconnectionCount {
				reconnectionCount = candidate
			}
		}
	}

	// Create new participant record
	participant := &Participant{
		ID:                uuid.New().String(),
		StreamSessionID:   streamSessionID,
		ParticipantID:     participantID,
		UserDID:           userDID,
		JoinedAt:          now,
		LeftAt:            nil, // Active
		ReconnectionCount: reconnectionCount,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	// Store participant
	r.participants[participant.ID] = participant

	// Update active index
	if r.activeIndex[streamSessionID] == nil {
		r.activeIndex[streamSessionID] = make(map[string]string)
	}
	r.activeIndex[streamSessionID][participantID] = participant.ID

	// Update denormalized count
	activeCount := len(r.activeIndex[streamSessionID])
	if err := r.sessionRepo.UpdateActiveParticipantCount(streamSessionID, activeCount); err != nil {
		// Log but don't fail the operation
		// In production, this would be logged with proper observability
	}

	participantCopy := *participant
	return &participantCopy, isReconnection, nil
}

// RecordLeave marks a participant as having left the stream.
func (r *InMemoryParticipantRepository) RecordLeave(streamSessionID, participantID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Find active participant
	streamActive, exists := r.activeIndex[streamSessionID]
	if !exists {
		return ErrParticipantNotFound
	}

	participantRecordID, active := streamActive[participantID]
	if !active {
		return ErrParticipantNotFound
	}

	// Get participant record
	participant, exists := r.participants[participantRecordID]
	if !exists {
		return ErrParticipantNotFound
	}

	// Mark as left
	now := time.Now()
	participant.LeftAt = &now
	participant.UpdatedAt = now

	// Remove from active index
	delete(streamActive, participantID)
	if len(streamActive) == 0 {
		delete(r.activeIndex, streamSessionID)
	}

	// Update denormalized count
	activeCount := 0
	if remaining, exists := r.activeIndex[streamSessionID]; exists {
		activeCount = len(remaining)
	}
	if err := r.sessionRepo.UpdateActiveParticipantCount(streamSessionID, activeCount); err != nil {
		// Log but don't fail the operation
	}

	return nil
}

// GetActiveParticipants returns all currently active participants for a stream.
func (r *InMemoryParticipantRepository) GetActiveParticipants(streamSessionID string) ([]*Participant, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	streamActive, exists := r.activeIndex[streamSessionID]
	if !exists {
		return []*Participant{}, nil
	}

	result := make([]*Participant, 0, len(streamActive))
	for _, participantRecordID := range streamActive {
		if participant, exists := r.participants[participantRecordID]; exists {
			participantCopy := *participant
			result = append(result, &participantCopy)
		}
	}

	return result, nil
}

// GetParticipantHistory returns all participants (active and past) for a stream.
func (r *InMemoryParticipantRepository) GetParticipantHistory(streamSessionID string) ([]*Participant, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*Participant
	for _, participant := range r.participants {
		if participant.StreamSessionID == streamSessionID {
			participantCopy := *participant
			result = append(result, &participantCopy)
		}
	}

	// Sort by joined_at descending (most recent first)
	sort.Slice(result, func(i, j int) bool {
		return result[i].JoinedAt.After(result[j].JoinedAt)
	})

	return result, nil
}

// GetActiveCount returns the count of currently active participants.
func (r *InMemoryParticipantRepository) GetActiveCount(streamSessionID string) (int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if streamActive, exists := r.activeIndex[streamSessionID]; exists {
		return len(streamActive), nil
	}
	return 0, nil
}

// UpdateSessionParticipantCount updates the denormalized count on stream_sessions.
func (r *InMemoryParticipantRepository) UpdateSessionParticipantCount(streamSessionID string, count int) error {
	return r.sessionRepo.UpdateActiveParticipantCount(streamSessionID, count)
}
