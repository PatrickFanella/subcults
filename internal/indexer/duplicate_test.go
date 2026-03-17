package indexer

import (
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestDuplicate_DetectionByDIDAndRKey tests that duplicate records are properly identified.
// This tests the core duplicate detection logic needed for database upserts.
func TestDuplicate_DetectionByDIDAndRKey(t *testing.T) {
	filter := NewRecordFilter(NewFilterMetrics())

	// Create two identical records (same DID + RKey)
	record1Data := map[string]interface{}{"name": "Test Scene Version 1"}
	record1Bytes, _ := json.Marshal(record1Data)

	record2Data := map[string]interface{}{"name": "Test Scene Version 2"}
	record2Bytes, _ := json.Marshal(record2Data)

	msg1 := JetstreamMessage{
		DID:    "did:plc:test123",
		TimeUS: time.Now().UnixMicro(),
		Kind:   "commit",
		Commit: &AtProtoCommit{
			DID:        "did:plc:test123",
			Rev:        "rev1",
			Operation:  "create",
			Collection: CollectionScene,
			RKey:       "scene1",
			Record:     mustEncodeCBORForDup(record1Bytes),
		},
	}

	msg2 := JetstreamMessage{
		DID:    "did:plc:test123",
		TimeUS: time.Now().Add(1 * time.Second).UnixMicro(),
		Kind:   "commit",
		Commit: &AtProtoCommit{
			DID:        "did:plc:test123",
			Rev:        "rev2",
			Operation:  "update",
			Collection: CollectionScene,
			RKey:       "scene1", // Same RKey as msg1
			Record:     mustEncodeCBORForDup(record2Bytes),
		},
	}

	msg1CBOR, _ := EncodeCBOR(msg1)
	msg2CBOR, _ := EncodeCBOR(msg2)

	// Process both messages
	result1 := filter.FilterCBOR(msg1CBOR)
	result2 := filter.FilterCBOR(msg2CBOR)

	// Both should be valid
	if !result1.Valid {
		t.Errorf("First message should be valid: %v", result1.Error)
	}
	if !result2.Valid {
		t.Errorf("Second message should be valid: %v", result2.Error)
	}

	// Verify they have the same composite key (DID + RKey)
	if result1.DID != result2.DID {
		t.Errorf("DID mismatch: %s != %s", result1.DID, result2.DID)
	}
	if result1.RKey != result2.RKey {
		t.Errorf("RKey mismatch: %s != %s", result1.RKey, result2.RKey)
	}
	if result1.Collection != result2.Collection {
		t.Errorf("Collection mismatch: %s != %s", result1.Collection, result2.Collection)
	}

	// Verify operations differ
	if result1.Operation != "create" {
		t.Errorf("First operation should be 'create', got %s", result1.Operation)
	}
	if result2.Operation != "update" {
		t.Errorf("Second operation should be 'update', got %s", result2.Operation)
	}

	// In a real database scenario, this would be an upsert based on (DID, Collection, RKey)
	t.Logf("Duplicate key detected: DID=%s, Collection=%s, RKey=%s", result1.DID, result1.Collection, result1.RKey)
}

// TestDuplicate_TrackingInMemory tests in-memory duplicate tracking during ingestion.
// This simulates what would happen in the database layer.
func TestDuplicate_TrackingInMemory(t *testing.T) {
	type recordKey struct {
		DID        string
		Collection string
		RKey       string
	}

	// Simulate a simple in-memory record store
	records := make(map[recordKey][]byte)
	var mu sync.Mutex

	filter := NewRecordFilter(NewFilterMetrics())

	// Generate test messages with some duplicates
	messages := []JetstreamMessage{
		{
			DID:    "did:plc:user1",
			TimeUS: time.Now().UnixMicro(),
			Kind:   "commit",
			Commit: &AtProtoCommit{
				DID:        "did:plc:user1",
				Operation:  "create",
				Collection: CollectionScene,
				RKey:       "scene1",
				Record:     mustEncodeCBORForDup([]byte(`{"name":"Scene 1"}`)),
			},
		},
		{
			DID:    "did:plc:user1",
			TimeUS: time.Now().UnixMicro(),
			Kind:   "commit",
			Commit: &AtProtoCommit{
				DID:        "did:plc:user1",
				Operation:  "update",
				Collection: CollectionScene,
				RKey:       "scene1", // Duplicate key
				Record:     mustEncodeCBORForDup([]byte(`{"name":"Scene 1 Updated"}`)),
			},
		},
		{
			DID:    "did:plc:user2",
			TimeUS: time.Now().UnixMicro(),
			Kind:   "commit",
			Commit: &AtProtoCommit{
				DID:        "did:plc:user2",
				Operation:  "create",
				Collection: CollectionScene,
				RKey:       "scene2",
				Record:     mustEncodeCBORForDup([]byte(`{"name":"Scene 2"}`)),
			},
		},
		{
			DID:    "did:plc:user1",
			TimeUS: time.Now().UnixMicro(),
			Kind:   "commit",
			Commit: &AtProtoCommit{
				DID:        "did:plc:user1",
				Operation:  "create",
				Collection: CollectionEvent,
				RKey:       "scene1", // Same RKey but different collection
				Record:     mustEncodeCBORForDup([]byte(`{"name":"Event 1","sceneId":"scene1"}`)),
			},
		},
	}

	var creates, updates, upserts int

	for _, msg := range messages {
		cborData, _ := EncodeCBOR(msg)
		result := filter.FilterCBOR(cborData)

		if !result.Valid {
			t.Errorf("Message validation failed: %v", result.Error)
			continue
		}

		key := recordKey{
			DID:        result.DID,
			Collection: result.Collection,
			RKey:       result.RKey,
		}

		mu.Lock()
		if _, exists := records[key]; exists {
			// This is an update/upsert
			updates++
		} else {
			// This is a new record
			creates++
		}
		records[key] = result.Record
		upserts++
		mu.Unlock()
	}

	t.Logf("Record statistics:")
	t.Logf("  Total operations: %d", upserts)
	t.Logf("  New records: %d", creates)
	t.Logf("  Updates: %d", updates)
	t.Logf("  Unique records: %d", len(records))

	// Verify statistics
	if creates != 3 {
		t.Errorf("Expected 3 creates, got %d", creates)
	}
	if updates != 1 {
		t.Errorf("Expected 1 update, got %d", updates)
	}
	if len(records) != 3 {
		t.Errorf("Expected 3 unique records, got %d", len(records))
	}
}

// TestDuplicate_RaceConditions tests concurrent duplicate detection without races.
func TestDuplicate_RaceConditions(t *testing.T) {
	type recordKey struct {
		DID        string
		Collection string
		RKey       string
	}

	// Concurrent-safe record store
	records := sync.Map{}
	filter := NewRecordFilter(NewFilterMetrics())

	const numGoroutines = 10
	const messagesPerGoroutine = 100

	var wg sync.WaitGroup
	var totalProcessed int32

	// Each goroutine processes messages for different DIDs
	for g := 0; g < numGoroutines; g++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			for i := 0; i < messagesPerGoroutine; i++ {
				did := fmt.Sprintf("did:plc:user%d", goroutineID)
				rkey := fmt.Sprintf("scene%d", i%10) // Reuse some RKeys to create updates

				msg := JetstreamMessage{
					DID:    did,
					TimeUS: time.Now().UnixMicro(),
					Kind:   "commit",
					Commit: &AtProtoCommit{
						DID:        did,
						Operation:  "create",
						Collection: CollectionScene,
						RKey:       rkey,
						Record:     mustEncodeCBORForDup([]byte(fmt.Sprintf(`{"name":"Scene %d"}`, i))),
					},
				}

				cborData, _ := EncodeCBOR(msg)
				result := filter.FilterCBOR(cborData)

				if result.Valid {
					key := recordKey{
						DID:        result.DID,
						Collection: result.Collection,
						RKey:       result.RKey,
					}
					records.Store(key, result.Record)
					atomic.AddInt32(&totalProcessed, 1)
				}
			}
		}(g)
	}

	wg.Wait()

	// Count unique records
	var uniqueRecords int
	records.Range(func(key, value interface{}) bool {
		uniqueRecords++
		return true
	})

	processed := atomic.LoadInt32(&totalProcessed)
	expectedTotal := numGoroutines * messagesPerGoroutine

	t.Logf("Concurrent duplicate detection results:")
	t.Logf("  Total processed: %d", processed)
	t.Logf("  Expected total: %d", expectedTotal)
	t.Logf("  Unique records: %d", uniqueRecords)

	if processed != int32(expectedTotal) {
		t.Errorf("Processing count mismatch: %d != %d", processed, expectedTotal)
	}

	// Each goroutine had 10 unique RKeys, so we expect 10 * numGoroutines unique records
	expectedUnique := 10 * numGoroutines
	if uniqueRecords != expectedUnique {
		t.Errorf("Unique record count: %d, expected %d", uniqueRecords, expectedUnique)
	}
}

// TestDuplicate_DeleteOperations tests handling of delete operations for duplicate keys.
func TestDuplicate_DeleteOperations(t *testing.T) {
	type recordKey struct {
		DID        string
		Collection string
		RKey       string
	}

	records := make(map[recordKey]bool) // true = exists, false = deleted
	var mu sync.Mutex

	filter := NewRecordFilter(NewFilterMetrics())

	// Sequence: create -> update -> delete -> create
	operations := []struct {
		operation string
		name      string
		expectErr bool
	}{
		{"create", "Initial Scene", false},
		{"update", "Updated Scene", false},
		{"delete", "", false},
		{"create", "Recreated Scene", false},
	}

	did := "did:plc:test"
	rkey := "scene1"

	for i, op := range operations {
		var msg JetstreamMessage

		if op.operation == "delete" {
			msg = JetstreamMessage{
				DID:    did,
				TimeUS: time.Now().UnixMicro(),
				Kind:   "commit",
				Commit: &AtProtoCommit{
					DID:        did,
					Operation:  op.operation,
					Collection: CollectionScene,
					RKey:       rkey,
					// No record data for deletes
				},
			}
		} else {
			msg = JetstreamMessage{
				DID:    did,
				TimeUS: time.Now().UnixMicro(),
				Kind:   "commit",
				Commit: &AtProtoCommit{
					DID:        did,
					Operation:  op.operation,
					Collection: CollectionScene,
					RKey:       rkey,
					Record:     mustEncodeCBORForDup([]byte(fmt.Sprintf(`{"name":"%s"}`, op.name))),
				},
			}
		}

		cborData, _ := EncodeCBOR(msg)
		result := filter.FilterCBOR(cborData)

		if !result.Valid {
			t.Errorf("Operation %d (%s) validation failed: %v", i, op.operation, result.Error)
			continue
		}

		key := recordKey{
			DID:        result.DID,
			Collection: result.Collection,
			RKey:       result.RKey,
		}

		mu.Lock()
		if result.Operation == "delete" {
			records[key] = false // Mark as deleted
		} else {
			records[key] = true // Mark as existing
		}
		mu.Unlock()

		t.Logf("Operation %d: %s completed", i, op.operation)
	}

	// Final state should be: record exists (recreated after delete)
	key := recordKey{DID: did, Collection: CollectionScene, RKey: rkey}
	mu.Lock()
	exists := records[key]
	mu.Unlock()

	if !exists {
		t.Error("Expected record to exist after final create")
	}
}

// TestTransactionAtomicity_SimulatedRollback tests that failed operations don't leave partial state.
// Note: This is a conceptual test. Real transaction atomicity requires database support.
func TestTransactionAtomicity_SimulatedRollback(t *testing.T) {
	filter := NewRecordFilter(NewFilterMetrics())

	// Simulate a batch of operations that should be atomic
	type operation struct {
		msg        JetstreamMessage
		shouldFail bool
	}

	batch := []operation{
		{
			msg: JetstreamMessage{
				DID:    "did:plc:test",
				TimeUS: time.Now().UnixMicro(),
				Kind:   "commit",
				Commit: &AtProtoCommit{
					DID:        "did:plc:test",
					Operation:  "create",
					Collection: CollectionScene,
					RKey:       "scene1",
					Record:     mustEncodeCBORForDup([]byte(`{"name":"Scene 1"}`)),
				},
			},
			shouldFail: false,
		},
		{
			msg: JetstreamMessage{
				DID:    "did:plc:test",
				TimeUS: time.Now().UnixMicro(),
				Kind:   "commit",
				Commit: &AtProtoCommit{
					DID:        "did:plc:test",
					Operation:  "create",
					Collection: CollectionScene,
					RKey:       "scene2",
					// Missing required field - should fail validation
					Record: mustEncodeCBORForDup([]byte(`{"description":"No name"}`)),
				},
			},
			shouldFail: true,
		},
	}

	// Process batch with transaction semantics
	type recordKey struct {
		DID, Collection, RKey string
	}
	committedRecords := make(map[recordKey][]byte)
	stagingRecords := make(map[recordKey][]byte)

	var allValid = true
	for _, op := range batch {
		cborData, _ := EncodeCBOR(op.msg)
		result := filter.FilterCBOR(cborData)

		if !result.Valid {
			allValid = false
			if !op.shouldFail {
				t.Errorf("Unexpected validation failure: %v", result.Error)
			}
			break
		}

		if op.shouldFail {
			t.Error("Expected validation to fail, but it passed")
			allValid = false
			break
		}

		// Stage the record
		key := recordKey{
			DID:        result.DID,
			Collection: result.Collection,
			RKey:       result.RKey,
		}
		stagingRecords[key] = result.Record
	}

	// Commit or rollback
	if allValid {
		// Commit all staged records
		for k, v := range stagingRecords {
			committedRecords[k] = v
		}
		t.Log("Transaction committed")
	} else {
		// Rollback - discard staged records
		t.Log("Transaction rolled back due to validation error")
	}

	// Verify: no partial state should exist
	if len(committedRecords) != 0 {
		t.Errorf("Expected 0 committed records after rollback, got %d", len(committedRecords))
	}

	t.Logf("Final state: %d committed records", len(committedRecords))
}

// TestTransactionAtomicity_AllOrNothing tests that batch operations are atomic.
func TestTransactionAtomicity_AllOrNothing(t *testing.T) {
	filter := NewRecordFilter(NewFilterMetrics())

	// Test case 1: All valid - should commit all
	validBatch := []JetstreamMessage{
		{
			DID:    "did:plc:test1",
			TimeUS: time.Now().UnixMicro(),
			Kind:   "commit",
			Commit: &AtProtoCommit{
				DID:        "did:plc:test1",
				Operation:  "create",
				Collection: CollectionScene,
				RKey:       "scene1",
				Record:     mustEncodeCBORForDup([]byte(`{"name":"Scene 1"}`)),
			},
		},
		{
			DID:    "did:plc:test2",
			TimeUS: time.Now().UnixMicro(),
			Kind:   "commit",
			Commit: &AtProtoCommit{
				DID:        "did:plc:test2",
				Operation:  "create",
				Collection: CollectionScene,
				RKey:       "scene2",
				Record:     mustEncodeCBORForDup([]byte(`{"name":"Scene 2"}`)),
			},
		},
	}

	committed := processBatchAtomically(t, filter, validBatch)
	if committed != len(validBatch) {
		t.Errorf("Valid batch: expected %d committed, got %d", len(validBatch), committed)
	}

	// Test case 2: One invalid - should commit none
	invalidBatch := []JetstreamMessage{
		{
			DID:    "did:plc:test3",
			TimeUS: time.Now().UnixMicro(),
			Kind:   "commit",
			Commit: &AtProtoCommit{
				DID:        "did:plc:test3",
				Operation:  "create",
				Collection: CollectionScene,
				RKey:       "scene3",
				Record:     mustEncodeCBORForDup([]byte(`{"name":"Scene 3"}`)),
			},
		},
		{
			DID:    "did:plc:test4",
			TimeUS: time.Now().UnixMicro(),
			Kind:   "commit",
			Commit: &AtProtoCommit{
				DID:        "did:plc:test4",
				Operation:  "create",
				Collection: CollectionScene,
				RKey:       "scene4",
				Record:     mustEncodeCBORForDup([]byte(`{"invalid":"no name field"}`)),
			},
		},
	}

	committed = processBatchAtomically(t, filter, invalidBatch)
	if committed != 0 {
		t.Errorf("Invalid batch: expected 0 committed, got %d", committed)
	}
}

// Helper function to process a batch atomically (simulated)
func processBatchAtomically(t *testing.T, filter *RecordFilter, batch []JetstreamMessage) int {
	t.Helper()

	type recordKey struct {
		DID, Collection, RKey string
	}
	staging := make(map[recordKey][]byte)

	// Validate all operations first
	for _, msg := range batch {
		cborData, _ := EncodeCBOR(msg)
		result := filter.FilterCBOR(cborData)

		if !result.Valid {
			// Validation failed - rollback entire batch
			t.Logf("Batch validation failed: %v", result.Error)
			return 0
		}

		key := recordKey{
			DID:        result.DID,
			Collection: result.Collection,
			RKey:       result.RKey,
		}
		staging[key] = result.Record
	}

	// All valid - commit
	return len(staging)
}

// Helper function
func mustEncodeCBORForDup(data []byte) []byte {
	// First unmarshal the JSON
	var v interface{}
	if err := json.Unmarshal(data, &v); err != nil {
		panic(err)
	}
	// Then encode to CBOR
	result, err := EncodeCBOR(v)
	if err != nil {
		panic(err)
	}
	return result
}
