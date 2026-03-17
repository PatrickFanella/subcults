package telemetry

import (
	"context"
	"sync"

	"github.com/google/uuid"
)

// InMemoryStore is an in-memory implementation of Store for development and testing.
type InMemoryStore struct {
	mu           sync.RWMutex
	events       []TelemetryEvent
	errorLogs    []ClientErrorLog
	replayEvents []ReplayEvent
	// dedup tracks error_hash+session_id combinations to prevent duplicates
	dedup map[string]bool
}

// NewInMemoryStore creates a new in-memory telemetry store.
func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{
		events:       make([]TelemetryEvent, 0),
		errorLogs:    make([]ClientErrorLog, 0),
		replayEvents: make([]ReplayEvent, 0),
		dedup:        make(map[string]bool),
	}
}

// InsertEvents persists a batch of telemetry events.
func (s *InMemoryStore) InsertEvents(_ context.Context, events []TelemetryEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range events {
		if events[i].ID == "" {
			events[i].ID = uuid.New().String()
		}
		s.events = append(s.events, events[i])
	}
	return nil
}

// InsertClientError persists a client error log.
func (s *InMemoryStore) InsertClientError(_ context.Context, errLog ClientErrorLog) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	dedupKey := errLog.ErrorHash + "|" + errLog.SessionID
	if s.dedup[dedupKey] {
		return "", ErrDuplicateError
	}

	if errLog.ID == "" {
		errLog.ID = uuid.New().String()
	}
	s.dedup[dedupKey] = true
	s.errorLogs = append(s.errorLogs, errLog)
	return errLog.ID, nil
}

// InsertReplayEvents persists session replay events linked to an error log.
func (s *InMemoryStore) InsertReplayEvents(_ context.Context, errorLogID string, events []ReplayEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range events {
		if events[i].ID == "" {
			events[i].ID = uuid.New().String()
		}
		events[i].ErrorLogID = errorLogID
		s.replayEvents = append(s.replayEvents, events[i])
	}
	return nil
}

// GetEvents returns all stored events (for testing).
func (s *InMemoryStore) GetEvents() []TelemetryEvent {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]TelemetryEvent, len(s.events))
	copy(result, s.events)
	return result
}

// GetErrorLogs returns all stored error logs (for testing).
func (s *InMemoryStore) GetErrorLogs() []ClientErrorLog {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]ClientErrorLog, len(s.errorLogs))
	copy(result, s.errorLogs)
	return result
}

// GetReplayEvents returns all stored replay events (for testing).
func (s *InMemoryStore) GetReplayEvents() []ReplayEvent {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]ReplayEvent, len(s.replayEvents))
	copy(result, s.replayEvents)
	return result
}
