package payment

import (
	"fmt"
	"sync"
	"testing"
)

// TestRecordEvent_Success tests recording a new event.
func TestRecordEvent_Success(t *testing.T) {
	repo := NewInMemoryWebhookRepository()

	err := repo.RecordEvent("evt_test123", "payment_intent.succeeded")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify event was recorded
	hasProcessed, err := repo.HasProcessed("evt_test123")
	if err != nil {
		t.Fatalf("failed to check processed status: %v", err)
	}
	if !hasProcessed {
		t.Error("event should be marked as processed")
	}
}

// TestRecordEvent_Duplicate tests that duplicate events return ErrEventAlreadyProcessed.
func TestRecordEvent_Duplicate(t *testing.T) {
	repo := NewInMemoryWebhookRepository()

	// Record first time - should succeed
	err := repo.RecordEvent("evt_duplicate", "checkout.session.completed")
	if err != nil {
		t.Fatalf("first record failed: %v", err)
	}

	// Record second time - should return ErrEventAlreadyProcessed
	err = repo.RecordEvent("evt_duplicate", "checkout.session.completed")
	if err != ErrEventAlreadyProcessed {
		t.Errorf("expected ErrEventAlreadyProcessed, got %v", err)
	}
}

// TestHasProcessed_NotFound tests checking for an event that doesn't exist.
func TestHasProcessed_NotFound(t *testing.T) {
	repo := NewInMemoryWebhookRepository()

	hasProcessed, err := repo.HasProcessed("evt_nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hasProcessed {
		t.Error("event should not be marked as processed")
	}
}

// TestRecordEvent_DifferentTypes tests recording events with different types.
func TestRecordEvent_DifferentTypes(t *testing.T) {
	repo := NewInMemoryWebhookRepository()

	events := []struct {
		id        string
		eventType string
	}{
		{"evt_001", "payment_intent.succeeded"},
		{"evt_002", "payment_intent.payment_failed"},
		{"evt_003", "checkout.session.completed"},
		{"evt_004", "account.updated"},
	}

	for _, e := range events {
		err := repo.RecordEvent(e.id, e.eventType)
		if err != nil {
			t.Errorf("failed to record event %s: %v", e.id, err)
		}
	}

	// Verify all events were recorded
	for _, e := range events {
		hasProcessed, err := repo.HasProcessed(e.id)
		if err != nil {
			t.Fatalf("failed to check event %s: %v", e.id, err)
		}
		if !hasProcessed {
			t.Errorf("event %s should be marked as processed", e.id)
		}
	}
}

// TestRecordEvent_ConcurrentWrites tests thread safety with concurrent writes.
func TestRecordEvent_ConcurrentWrites(t *testing.T) {
	repo := NewInMemoryWebhookRepository()

	const numGoroutines = 100
	const numEventsPerGoroutine = 10

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Launch concurrent goroutines that record events
	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			defer wg.Done()
			for j := 0; j < numEventsPerGoroutine; j++ {
				eventID := fmt.Sprintf("evt_%d_%d", goroutineID, j)
				err := repo.RecordEvent(eventID, "test.event")
				if err != nil {
					t.Errorf("goroutine %d failed to record event: %v", goroutineID, err)
				}
			}
		}(i)
	}

	wg.Wait()

	// Verify total number of recorded events
	repo.mu.RLock()
	totalEvents := len(repo.events)
	repo.mu.RUnlock()

	expectedTotal := numGoroutines * numEventsPerGoroutine
	if totalEvents != expectedTotal {
		t.Errorf("expected %d events, got %d", expectedTotal, totalEvents)
	}
}

// TestRecordEvent_ConcurrentDuplicates tests thread safety with concurrent duplicate attempts.
func TestRecordEvent_ConcurrentDuplicates(t *testing.T) {
	repo := NewInMemoryWebhookRepository()

	const numGoroutines = 50
	const eventID = "evt_concurrent_duplicate"

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	successCount := 0
	duplicateCount := 0
	var countMutex sync.Mutex

	// Launch concurrent goroutines that try to record the same event
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			err := repo.RecordEvent(eventID, "test.event")

			countMutex.Lock()
			if err == nil {
				successCount++
			} else if err == ErrEventAlreadyProcessed {
				duplicateCount++
			} else {
				t.Errorf("unexpected error: %v", err)
			}
			countMutex.Unlock()
		}()
	}

	wg.Wait()

	// Exactly one goroutine should succeed, the rest should get duplicates
	if successCount != 1 {
		t.Errorf("expected exactly 1 success, got %d", successCount)
	}
	if duplicateCount != numGoroutines-1 {
		t.Errorf("expected %d duplicates, got %d", numGoroutines-1, duplicateCount)
	}
}

// TestRecordEvent_ConcurrentReadWrite tests concurrent reads and writes.
func TestRecordEvent_ConcurrentReadWrite(t *testing.T) {
	repo := NewInMemoryWebhookRepository()

	const numWriters = 50
	const numReaders = 50
	const numEventsPerWriter = 10

	var wg sync.WaitGroup

	// Launch writer goroutines
	wg.Add(numWriters)
	for i := 0; i < numWriters; i++ {
		go func(writerID int) {
			defer wg.Done()
			for j := 0; j < numEventsPerWriter; j++ {
				eventID := fmt.Sprintf("evt_writer_%d_%d", writerID, j)
				_ = repo.RecordEvent(eventID, "test.event")
			}
		}(i)
	}

	// Launch reader goroutines
	wg.Add(numReaders)
	for i := 0; i < numReaders; i++ {
		go func(readerID int) {
			defer wg.Done()
			for j := 0; j < numEventsPerWriter; j++ {
				eventID := fmt.Sprintf("evt_writer_%d_%d", readerID%numWriters, j)
				_, _ = repo.HasProcessed(eventID)
			}
		}(i)
	}

	wg.Wait()

	// Verify final state - all writer events should be recorded
	for i := 0; i < numWriters; i++ {
		for j := 0; j < numEventsPerWriter; j++ {
			eventID := fmt.Sprintf("evt_writer_%d_%d", i, j)
			hasProcessed, err := repo.HasProcessed(eventID)
			if err != nil {
				t.Fatalf("failed to check event %s: %v", eventID, err)
			}
			if !hasProcessed {
				t.Errorf("event %s should be marked as processed", eventID)
			}
		}
	}
}

// TestRecordEvent_EmptyEventID tests behavior with empty event IDs.
func TestRecordEvent_EmptyEventID(t *testing.T) {
	repo := NewInMemoryWebhookRepository()

	// Record with empty event ID - should succeed (no validation in current impl)
	err := repo.RecordEvent("", "test.event")
	if err != nil {
		t.Errorf("expected no error for empty event ID, got %v", err)
	}

	// Verify it was recorded
	hasProcessed, err := repo.HasProcessed("")
	if err != nil {
		t.Fatalf("failed to check empty event: %v", err)
	}
	if !hasProcessed {
		t.Error("empty event ID should be marked as processed")
	}

	// Duplicate should fail
	err = repo.RecordEvent("", "test.event")
	if err != ErrEventAlreadyProcessed {
		t.Errorf("expected ErrEventAlreadyProcessed for duplicate empty ID, got %v", err)
	}
}
