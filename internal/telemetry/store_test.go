package telemetry

import (
	"context"
	"testing"
)

func TestInMemoryStore_InsertEvents(t *testing.T) {
	store := NewInMemoryStore()
	ctx := context.Background()

	events := []TelemetryEvent{
		{SessionID: "sess-1", Name: "page_view", Timestamp: 1000},
		{SessionID: "sess-1", Name: "click", Timestamp: 2000},
	}

	if err := store.InsertEvents(ctx, events); err != nil {
		t.Fatalf("InsertEvents failed: %v", err)
	}

	stored := store.GetEvents()
	if len(stored) != 2 {
		t.Fatalf("expected 2 events, got %d", len(stored))
	}
	if stored[0].ID == "" {
		t.Error("expected event ID to be generated")
	}
	if stored[0].Name != "page_view" {
		t.Errorf("expected event name 'page_view', got %q", stored[0].Name)
	}
}

func TestInMemoryStore_InsertClientError(t *testing.T) {
	store := NewInMemoryStore()
	ctx := context.Background()

	errLog := ClientErrorLog{
		SessionID:    "sess-1",
		ErrorType:    "TypeError",
		ErrorMessage: "Cannot read null",
		ErrorHash:    "abc123",
		OccurredAt:   1000,
	}

	id, err := store.InsertClientError(ctx, errLog)
	if err != nil {
		t.Fatalf("InsertClientError failed: %v", err)
	}
	if id == "" {
		t.Error("expected non-empty error ID")
	}

	logs := store.GetErrorLogs()
	if len(logs) != 1 {
		t.Fatalf("expected 1 error log, got %d", len(logs))
	}
}

func TestInMemoryStore_InsertClientError_Duplicate(t *testing.T) {
	store := NewInMemoryStore()
	ctx := context.Background()

	errLog := ClientErrorLog{
		SessionID:    "sess-1",
		ErrorType:    "TypeError",
		ErrorMessage: "Cannot read null",
		ErrorHash:    "abc123",
		OccurredAt:   1000,
	}

	_, err := store.InsertClientError(ctx, errLog)
	if err != nil {
		t.Fatalf("first InsertClientError failed: %v", err)
	}

	// Same hash + session should return ErrDuplicateError
	_, err = store.InsertClientError(ctx, errLog)
	if err != ErrDuplicateError {
		t.Errorf("expected ErrDuplicateError, got %v", err)
	}

	// Different session, same hash should succeed
	errLog.SessionID = "sess-2"
	id, err := store.InsertClientError(ctx, errLog)
	if err != nil {
		t.Fatalf("InsertClientError with different session failed: %v", err)
	}
	if id == "" {
		t.Error("expected non-empty error ID")
	}
}

func TestInMemoryStore_InsertReplayEvents(t *testing.T) {
	store := NewInMemoryStore()
	ctx := context.Background()

	errLog := ClientErrorLog{
		SessionID:    "sess-1",
		ErrorType:    "Error",
		ErrorMessage: "test error",
		ErrorHash:    "hash1",
		OccurredAt:   1000,
	}
	errorID, _ := store.InsertClientError(ctx, errLog)

	replays := []ReplayEvent{
		{EventType: "click", EventTimestamp: 900},
		{EventType: "navigation", EventTimestamp: 950},
	}

	if err := store.InsertReplayEvents(ctx, errorID, replays); err != nil {
		t.Fatalf("InsertReplayEvents failed: %v", err)
	}

	stored := store.GetReplayEvents()
	if len(stored) != 2 {
		t.Fatalf("expected 2 replay events, got %d", len(stored))
	}
	if stored[0].ErrorLogID != errorID {
		t.Errorf("expected error_log_id %q, got %q", errorID, stored[0].ErrorLogID)
	}
	if stored[0].ID == "" {
		t.Error("expected replay event ID to be generated")
	}
}
