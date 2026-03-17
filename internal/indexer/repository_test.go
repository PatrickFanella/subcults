package indexer

import (
	"context"
	"fmt"
	"testing"
)

// TestInMemoryRepository_UpsertRecord tests basic insert and update operations.
func TestInMemoryRepository_UpsertRecord(t *testing.T) {
	repo := NewInMemoryRecordRepository(newTestLogger())
	ctx := context.Background()

	// Create a valid filter result
	record := &FilterResult{
		DID:        "did:plc:test123",
		Collection: CollectionScene,
		RKey:       "scene1",
		Rev:        "rev1",
		Operation:  "create",
		Valid:      true,
		Matched:    true,
		Record:     []byte(`{"name":"Test Scene"}`),
	}

	// First insert should create a new record
	id1, isNew1, err := repo.UpsertRecord(ctx, record)
	if err != nil {
		t.Fatalf("UpsertRecord() error = %v", err)
	}
	if !isNew1 {
		t.Error("Expected isNew=true for first insert")
	}
	if id1 == "" {
		t.Error("Expected non-empty record ID")
	}

	// Update with different rev should create new record in idempotency tracking
	record.Rev = "rev2"
	record.Operation = "update"
	record.Record = []byte(`{"name":"Updated Scene"}`)

	id2, isNew2, err := repo.UpsertRecord(ctx, record)
	if err != nil {
		t.Fatalf("UpsertRecord() error = %v", err)
	}
	if isNew2 {
		t.Error("Expected isNew=false for update operation")
	}
	if id2 == "" {
		t.Error("Expected non-empty record ID")
	}
}

// TestInMemoryRepository_Idempotency tests that duplicate records are not processed.
func TestInMemoryRepository_Idempotency(t *testing.T) {
	repo := NewInMemoryRecordRepository(newTestLogger())
	ctx := context.Background()

	record := &FilterResult{
		DID:        "did:plc:test456",
		Collection: CollectionEvent,
		RKey:       "event1",
		Rev:        "rev1",
		Operation:  "create",
		Valid:      true,
		Matched:    true,
		Record:     []byte(`{"name":"Test Event"}`),
	}

	// First insert
	id1, isNew1, err := repo.UpsertRecord(ctx, record)
	if err != nil {
		t.Fatalf("First UpsertRecord() error = %v", err)
	}
	if !isNew1 {
		t.Error("Expected isNew=true for first insert")
	}

	// Exact same record (same rev) should be skipped
	id2, isNew2, err := repo.UpsertRecord(ctx, record)
	if err != nil {
		t.Fatalf("Second UpsertRecord() error = %v", err)
	}
	if isNew2 {
		t.Error("Expected isNew=false for duplicate")
	}
	if id2 != "" {
		t.Error("Expected empty ID for skipped duplicate")
	}
	if id1 == id2 && id2 != "" {
		t.Log("IDs matched (OK if both non-empty)")
	}
}

// TestInMemoryRepository_DifferentCollections tests that same DID+RKey in different collections are separate.
func TestInMemoryRepository_DifferentCollections(t *testing.T) {
	repo := NewInMemoryRecordRepository(newTestLogger())
	ctx := context.Background()

	// Same DID and RKey, different collections
	record1 := &FilterResult{
		DID:        "did:plc:test789",
		Collection: CollectionScene,
		RKey:       "record1",
		Rev:        "rev1",
		Operation:  "create",
		Valid:      true,
		Matched:    true,
		Record:     []byte(`{"name":"Scene"}`),
	}

	record2 := &FilterResult{
		DID:        "did:plc:test789",
		Collection: CollectionEvent,
		RKey:       "record1", // Same RKey
		Rev:        "rev1",
		Operation:  "create",
		Valid:      true,
		Matched:    true,
		Record:     []byte(`{"name":"Event"}`),
	}

	// Both should be inserted as new records
	_, isNew1, err := repo.UpsertRecord(ctx, record1)
	if err != nil {
		t.Fatalf("UpsertRecord(scene) error = %v", err)
	}
	if !isNew1 {
		t.Error("Expected isNew=true for scene")
	}

	_, isNew2, err := repo.UpsertRecord(ctx, record2)
	if err != nil {
		t.Fatalf("UpsertRecord(event) error = %v", err)
	}
	if !isNew2 {
		t.Error("Expected isNew=true for event (different collection)")
	}
}

// TestInMemoryRepository_DeleteRecord tests record deletion.
// Note: Idempotency keys are intentionally NOT cleaned up on delete to prevent
// re-ingestion of deleted content. A new revision can still be inserted.
func TestInMemoryRepository_DeleteRecord(t *testing.T) {
	repo := NewInMemoryRecordRepository(newTestLogger())
	ctx := context.Background()

	record := &FilterResult{
		DID:        "did:plc:deletetest",
		Collection: CollectionPost,
		RKey:       "post1",
		Rev:        "rev1",
		Operation:  "create",
		Valid:      true,
		Matched:    true,
		Record:     []byte(`{"text":"Test post","sceneId":"scene1"}`),
	}

	// Insert record
	firstID, _, err := repo.UpsertRecord(ctx, record)
	if err != nil {
		t.Fatalf("UpsertRecord() error = %v", err)
	}

	// Delete record
	err = repo.DeleteRecord(ctx, record.DID, record.Collection, record.RKey)
	if err != nil {
		t.Fatalf("DeleteRecord() error = %v", err)
	}

	// Trying to insert the same revision again should be skipped (idempotency key preserved)
	recordID, isNew, err := repo.UpsertRecord(ctx, record)
	if err != nil {
		t.Fatalf("UpsertRecord() same revision after delete error = %v", err)
	}
	if recordID != "" {
		t.Error("Expected empty recordID for duplicate revision after delete")
	}
	if isNew {
		t.Error("Expected isNew=false for duplicate revision (idempotency key preserved)")
	}

	// However, inserting a NEW revision should work and reuse the same stable ID
	record.Rev = "rev2"
	recordID, isNew, err = repo.UpsertRecord(ctx, record)
	if err != nil {
		t.Fatalf("UpsertRecord() new revision error = %v", err)
	}
	// With stable IDs, we expect the same ID to be returned
	if recordID != firstID {
		t.Errorf("Expected same stable ID after deletion and reinsertion, got %s != %s", recordID, firstID)
	}
	// Since the record key already existed (was deleted), isNew should be false
	if isNew {
		t.Error("Expected isNew=false for existing record key (stable ID reused)")
	}
	if recordID == "" {
		t.Error("Expected non-empty recordID for new revision")
	}
}

// TestInMemoryRepository_CheckIdempotencyKey tests idempotency key checking.
func TestInMemoryRepository_CheckIdempotencyKey(t *testing.T) {
	repo := NewInMemoryRecordRepository(newTestLogger())
	ctx := context.Background()

	record := &FilterResult{
		DID:        "did:plc:idemptest",
		Collection: CollectionScene, // Changed from CollectionMembership
		RKey:       "scene1",        // Changed from member1
		Rev:        "rev1",
		Operation:  "create",
		Valid:      true,
		Matched:    true,
		Record:     []byte(`{"name":"Test Scene"}`),
	}

	// Generate the same key the repository would use
	expectedKey := generateIdempotencyKey(record.DID, record.Collection, record.RKey, record.Rev)

	// Key should not exist initially
	exists, err := repo.CheckIdempotencyKey(ctx, expectedKey)
	if err != nil {
		t.Fatalf("CheckIdempotencyKey() error = %v", err)
	}
	if exists {
		t.Error("Expected key to not exist before upsert")
	}

	// Insert record
	_, _, err = repo.UpsertRecord(ctx, record)
	if err != nil {
		t.Fatalf("UpsertRecord() error = %v", err)
	}

	// Key should exist now
	exists, err = repo.CheckIdempotencyKey(ctx, expectedKey)
	if err != nil {
		t.Fatalf("CheckIdempotencyKey() after upsert error = %v", err)
	}
	if !exists {
		t.Error("Expected key to exist after upsert")
	}
}

// TestInMemoryRepository_InvalidRecords tests handling of invalid records.
func TestInMemoryRepository_InvalidRecords(t *testing.T) {
	repo := NewInMemoryRecordRepository(newTestLogger())
	ctx := context.Background()

	tests := []struct {
		name   string
		record *FilterResult
	}{
		{
			name: "invalid record",
			record: &FilterResult{
				DID:        "did:plc:test",
				Collection: CollectionScene,
				RKey:       "scene1",
				Rev:        "rev1",
				Valid:      false, // Invalid
				Matched:    true,
			},
		},
		{
			name: "unmatched record",
			record: &FilterResult{
				DID:        "did:plc:test",
				Collection: "app.bsky.feed.post",
				RKey:       "post1",
				Rev:        "rev1",
				Valid:      true,
				Matched:    false, // Unmatched
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := repo.UpsertRecord(ctx, tt.record)
			if err == nil {
				t.Error("Expected error for invalid/unmatched record")
			}
		})
	}
}

// TestInMemoryRepository_ConcurrentAccess tests thread safety.
func TestInMemoryRepository_ConcurrentAccess(t *testing.T) {
	repo := NewInMemoryRecordRepository(newTestLogger())
	ctx := context.Background()

	const numGoroutines = 10
	const recordsPerGoroutine = 100

	errors := make(chan error, numGoroutines*recordsPerGoroutine)
	done := make(chan bool, numGoroutines)

	for g := 0; g < numGoroutines; g++ {
		go func(goroutineID int) {
			for i := 0; i < recordsPerGoroutine; i++ {
				record := &FilterResult{
					DID:        "did:plc:concurrent",
					Collection: CollectionPost, // Changed from CollectionAlliance
					RKey:       generateRKey(goroutineID, i),
					Rev:        "rev1",
					Operation:  "create",
					Valid:      true,
					Matched:    true,
					Record:     []byte(`{"text":"Test","sceneId":"scene1"}`),
				}

				_, _, err := repo.UpsertRecord(ctx, record)
				if err != nil {
					errors <- err
				}
			}
			done <- true
		}(g)
	}

	// Wait for all goroutines
	for i := 0; i < numGoroutines; i++ {
		<-done
	}
	close(errors)

	// Check for errors
	errorCount := 0
	for err := range errors {
		t.Errorf("Concurrent access error: %v", err)
		errorCount++
	}

	if errorCount > 0 {
		t.Fatalf("Got %d errors during concurrent access", errorCount)
	}
}

// TestGenerateIdempotencyKey tests the idempotency key generation.
func TestGenerateIdempotencyKey(t *testing.T) {
	tests := []struct {
		name       string
		did        string
		collection string
		rkey       string
		rev        string
		wantSame   bool
	}{
		{
			name:       "identical inputs produce same key",
			did:        "did:plc:test",
			collection: CollectionScene,
			rkey:       "scene1",
			rev:        "rev1",
			wantSame:   true,
		},
		{
			name:       "different rev produces different key",
			did:        "did:plc:test",
			collection: CollectionScene,
			rkey:       "scene1",
			rev:        "rev2",
			wantSame:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key1 := generateIdempotencyKey(tt.did, tt.collection, tt.rkey, "rev1")
			key2 := generateIdempotencyKey(tt.did, tt.collection, tt.rkey, tt.rev)

			if tt.wantSame && key1 != key2 {
				t.Errorf("Expected same keys, got %s and %s", key1, key2)
			}
			if !tt.wantSame && key1 == key2 {
				t.Errorf("Expected different keys, but both are %s", key1)
			}

			// Verify key is a valid hex string
			if len(key1) != 64 { // SHA256 produces 32 bytes = 64 hex chars
				t.Errorf("Expected key length 64, got %d", len(key1))
			}
		})
	}
}

// TestTransactionAtomicity_Simulation tests simulated transaction rollback behavior.
func TestTransactionAtomicity_Simulation(t *testing.T) {
	repo := NewInMemoryRecordRepository(newTestLogger())
	ctx := context.Background()

	// Simulate a batch operation where one record fails
	// In a real database, the entire transaction would rollback
	records := []*FilterResult{
		{
			DID:        "did:plc:batch1",
			Collection: CollectionScene,
			RKey:       "scene1",
			Rev:        "rev1",
			Valid:      true,
			Matched:    true,
			Record:     []byte(`{"name":"Scene 1"}`),
		},
		{
			DID:        "did:plc:batch2",
			Collection: CollectionScene,
			RKey:       "scene2",
			Rev:        "rev1",
			Valid:      false, // This will fail
			Matched:    true,
			Record:     []byte(`{"name":"Scene 2"}`),
		},
		{
			DID:        "did:plc:batch3",
			Collection: CollectionScene,
			RKey:       "scene3",
			Rev:        "rev1",
			Valid:      true,
			Matched:    true,
			Record:     []byte(`{"name":"Scene 3"}`),
		},
	}

	var successCount int
	var failureCount int

	// Process each record individually (no batch transaction in in-memory)
	for _, record := range records {
		_, _, err := repo.UpsertRecord(ctx, record)
		if err != nil {
			failureCount++
		} else {
			successCount++
		}
	}

	// With in-memory repo, we can't rollback, so some records succeed
	t.Logf("Success: %d, Failures: %d", successCount, failureCount)

	if failureCount != 1 {
		t.Errorf("Expected 1 failure, got %d", failureCount)
	}
	if successCount != 2 {
		t.Errorf("Expected 2 successes, got %d", successCount)
	}
}

// Helper function to generate unique rkeys
func generateRKey(goroutineID, index int) string {
	return fmt.Sprintf("g%d-r%d", goroutineID, index)
}
