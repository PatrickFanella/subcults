// Package payment provides webhook event tracking for idempotency.
package payment

import (
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ErrEventAlreadyProcessed is returned when attempting to process a duplicate webhook event.
var ErrEventAlreadyProcessed = errors.New("webhook event already processed")

// WebhookEvent represents a processed webhook event for idempotency tracking.
type WebhookEvent struct {
	ID          string
	EventID     string // Stripe event ID
	EventType   string // Stripe event type
	ProcessedAt time.Time
}

// WebhookRepository defines methods for webhook event tracking.
type WebhookRepository interface {
	// RecordEvent records a webhook event as processed.
	// Returns ErrEventAlreadyProcessed if the event was already recorded.
	// Idempotent: calling with the same event_id multiple times returns ErrEventAlreadyProcessed.
	RecordEvent(eventID, eventType string) error

	// HasProcessed checks if an event has already been processed.
	HasProcessed(eventID string) (bool, error)
}

// InMemoryWebhookRepository implements WebhookRepository with in-memory storage.
type InMemoryWebhookRepository struct {
	mu     sync.RWMutex
	events map[string]*WebhookEvent // Maps event_id -> WebhookEvent
}

// NewInMemoryWebhookRepository creates a new in-memory webhook repository.
func NewInMemoryWebhookRepository() *InMemoryWebhookRepository {
	return &InMemoryWebhookRepository{
		events: make(map[string]*WebhookEvent),
	}
}

// RecordEvent records a webhook event as processed.
func (r *InMemoryWebhookRepository) RecordEvent(eventID, eventType string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if already processed
	if _, exists := r.events[eventID]; exists {
		return ErrEventAlreadyProcessed
	}

	// Record the event
	event := &WebhookEvent{
		ID:          uuid.New().String(),
		EventID:     eventID,
		EventType:   eventType,
		ProcessedAt: time.Now(),
	}
	r.events[eventID] = event

	return nil
}

// HasProcessed checks if an event has already been processed.
func (r *InMemoryWebhookRepository) HasProcessed(eventID string) (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, exists := r.events[eventID]
	return exists, nil
}
