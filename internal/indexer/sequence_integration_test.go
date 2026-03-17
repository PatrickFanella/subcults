//go:build integration

package indexer

import (
	"context"
	"database/sql"
	"os"
	"testing"

	_ "github.com/lib/pq" // PostgreSQL driver
)

// setupTestDB creates a test database connection and ensures the indexer_state table exists.
func setupTestDB(t *testing.T) (*sql.DB, func()) {
	t.Helper()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL not set; skipping integration test")
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}

	if err := db.Ping(); err != nil {
		t.Fatalf("failed to ping database: %v", err)
	}

	// Clean up any existing test data
	cleanup := func() {
		_, _ = db.Exec("DELETE FROM indexer_state")
		db.Close()
	}

	// Ensure table is empty for test
	_, err = db.Exec("DELETE FROM indexer_state")
	if err != nil {
		t.Fatalf("failed to clean indexer_state table: %v", err)
	}

	return db, cleanup
}

// TestPostgresSequenceTracker_EmptyTable verifies behavior when no state exists.
func TestPostgresSequenceTracker_EmptyTable(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	tracker := NewPostgresSequenceTracker(db, newTestLogger())
	ctx := context.Background()

	// GetLastSequence should return 0 for empty table
	seq, err := tracker.GetLastSequence(ctx)
	if err != nil {
		t.Fatalf("GetLastSequence() error = %v", err)
	}
	if seq != 0 {
		t.Errorf("GetLastSequence() = %d, want 0", seq)
	}
}

// TestPostgresSequenceTracker_InsertPath verifies initial insert behavior.
func TestPostgresSequenceTracker_InsertPath(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	tracker := NewPostgresSequenceTracker(db, newTestLogger())
	ctx := context.Background()

	// First update should insert
	err := tracker.UpdateSequence(ctx, 100)
	if err != nil {
		t.Fatalf("UpdateSequence() error = %v", err)
	}

	// Verify the value was inserted
	seq, err := tracker.GetLastSequence(ctx)
	if err != nil {
		t.Fatalf("GetLastSequence() error = %v", err)
	}
	if seq != 100 {
		t.Errorf("GetLastSequence() = %d, want 100", seq)
	}
}

// TestPostgresSequenceTracker_UpdatePath verifies normal update behavior.
func TestPostgresSequenceTracker_UpdatePath(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	tracker := NewPostgresSequenceTracker(db, newTestLogger())
	ctx := context.Background()

	// Insert initial value
	err := tracker.UpdateSequence(ctx, 100)
	if err != nil {
		t.Fatalf("UpdateSequence(100) error = %v", err)
	}

	// Update to higher value
	err = tracker.UpdateSequence(ctx, 200)
	if err != nil {
		t.Fatalf("UpdateSequence(200) error = %v", err)
	}

	// Verify the value was updated
	seq, err := tracker.GetLastSequence(ctx)
	if err != nil {
		t.Fatalf("GetLastSequence() error = %v", err)
	}
	if seq != 200 {
		t.Errorf("GetLastSequence() = %d, want 200", seq)
	}
}

// TestPostgresSequenceTracker_MonotonicBehavior verifies that lower sequences don't overwrite.
func TestPostgresSequenceTracker_MonotonicBehavior(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	tracker := NewPostgresSequenceTracker(db, newTestLogger())
	ctx := context.Background()

	// Insert initial value
	err := tracker.UpdateSequence(ctx, 500)
	if err != nil {
		t.Fatalf("UpdateSequence(500) error = %v", err)
	}

	// Try to update to lower value (should be ignored)
	err = tracker.UpdateSequence(ctx, 300)
	if err != nil {
		t.Fatalf("UpdateSequence(300) error = %v", err)
	}

	// Verify the value did NOT decrease
	seq, err := tracker.GetLastSequence(ctx)
	if err != nil {
		t.Fatalf("GetLastSequence() error = %v", err)
	}
	if seq != 500 {
		t.Errorf("GetLastSequence() = %d, want 500 (should not have decreased to 300)", seq)
	}

	// Try to update to same value (should be ignored)
	err = tracker.UpdateSequence(ctx, 500)
	if err != nil {
		t.Fatalf("UpdateSequence(500) error = %v", err)
	}

	// Verify the value is still 500
	seq, err = tracker.GetLastSequence(ctx)
	if err != nil {
		t.Fatalf("GetLastSequence() error = %v", err)
	}
	if seq != 500 {
		t.Errorf("GetLastSequence() = %d, want 500", seq)
	}

	// Update to higher value (should work)
	err = tracker.UpdateSequence(ctx, 700)
	if err != nil {
		t.Fatalf("UpdateSequence(700) error = %v", err)
	}

	// Verify the value increased
	seq, err = tracker.GetLastSequence(ctx)
	if err != nil {
		t.Fatalf("GetLastSequence() error = %v", err)
	}
	if seq != 700 {
		t.Errorf("GetLastSequence() = %d, want 700", seq)
	}
}

// TestPostgresSequenceTracker_OutOfOrderSequences tests realistic out-of-order message scenarios.
func TestPostgresSequenceTracker_OutOfOrderSequences(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	tracker := NewPostgresSequenceTracker(db, newTestLogger())
	ctx := context.Background()

	// Simulate out-of-order message processing
	sequences := []int64{100, 300, 200, 400, 150, 500}
	expectedSequence := int64(100) // Start with first value

	for _, seq := range sequences {
		err := tracker.UpdateSequence(ctx, seq)
		if err != nil {
			t.Fatalf("UpdateSequence(%d) error = %v", seq, err)
		}

		// Track expected value (max so far)
		if seq > expectedSequence {
			expectedSequence = seq
		}

		// Verify cursor is always at the maximum seen
		actual, err := tracker.GetLastSequence(ctx)
		if err != nil {
			t.Fatalf("GetLastSequence() error = %v", err)
		}
		if actual != expectedSequence {
			t.Errorf("After UpdateSequence(%d): GetLastSequence() = %d, want %d", seq, actual, expectedSequence)
		}
	}

	// Final value should be the maximum
	seq, err := tracker.GetLastSequence(ctx)
	if err != nil {
		t.Fatalf("GetLastSequence() error = %v", err)
	}
	if seq != 500 {
		t.Errorf("Final GetLastSequence() = %d, want 500", seq)
	}
}
