package audit

import (
	"testing"
)

func TestInMemoryRepository_HashChain(t *testing.T) {
	repo := NewInMemoryRepository()

	// Log first entry
	entry1 := LogEntry{
		UserDID:    "user1",
		EntityType: "scene",
		EntityID:   "scene-1",
		Action:     "scene_create",
		Outcome:    OutcomeSuccess,
	}
	log1, err := repo.LogAccess(entry1)
	if err != nil {
		t.Fatalf("LogAccess() error = %v", err)
	}

	// First entry should have empty previous hash
	if log1.PreviousHash != "" {
		t.Errorf("First log entry PreviousHash = %q, want empty string", log1.PreviousHash)
	}

	// Log second entry
	entry2 := LogEntry{
		UserDID:    "user1",
		EntityType: "scene",
		EntityID:   "scene-1",
		Action:     "scene_update",
		Outcome:    OutcomeSuccess,
	}
	log2, err := repo.LogAccess(entry2)
	if err != nil {
		t.Fatalf("LogAccess() error = %v", err)
	}

	// Second entry should have non-empty previous hash
	if log2.PreviousHash == "" {
		t.Error("Second log entry should have non-empty PreviousHash")
	}

	// Log third entry
	entry3 := LogEntry{
		UserDID:    "user2",
		EntityType: "event",
		EntityID:   "event-1",
		Action:     "event_create",
		Outcome:    OutcomeSuccess,
	}
	log3, err := repo.LogAccess(entry3)
	if err != nil {
		t.Fatalf("LogAccess() error = %v", err)
	}

	// Third entry should have different previous hash
	if log3.PreviousHash == "" {
		t.Error("Third log entry should have non-empty PreviousHash")
	}
	if log3.PreviousHash == log2.PreviousHash {
		t.Error("Third log entry PreviousHash should differ from second log entry's PreviousHash")
	}
}

func TestInMemoryRepository_GetLastHash(t *testing.T) {
	repo := NewInMemoryRepository()

	// Empty repository should have empty last hash
	hash, err := repo.GetLastHash()
	if err != nil {
		t.Fatalf("GetLastHash() error = %v", err)
	}
	if hash != "" {
		t.Errorf("GetLastHash() on empty repo = %q, want empty string", hash)
	}

	// Add one entry
	entry := LogEntry{
		UserDID:    "user1",
		EntityType: "scene",
		EntityID:   "scene-1",
		Action:     "scene_create",
		Outcome:    OutcomeSuccess,
	}
	_, err = repo.LogAccess(entry)
	if err != nil {
		t.Fatalf("LogAccess() error = %v", err)
	}

	// Now last hash should be set
	hash, err = repo.GetLastHash()
	if err != nil {
		t.Fatalf("GetLastHash() error = %v", err)
	}
	if hash == "" {
		t.Error("GetLastHash() should return non-empty hash after logging")
	}

	// Add another entry
	entry2 := LogEntry{
		UserDID:    "user2",
		EntityType: "event",
		EntityID:   "event-1",
		Action:     "event_create",
		Outcome:    OutcomeSuccess,
	}
	_, err = repo.LogAccess(entry2)
	if err != nil {
		t.Fatalf("LogAccess() error = %v", err)
	}

	// Last hash should be updated
	hash2, err := repo.GetLastHash()
	if err != nil {
		t.Fatalf("GetLastHash() error = %v", err)
	}
	if hash2 == hash {
		t.Error("GetLastHash() should return different hash after new entry")
	}
}

func TestInMemoryRepository_VerifyHashChain_EmptyRepo(t *testing.T) {
	repo := NewInMemoryRepository()

	valid, err := repo.VerifyHashChain()
	if err != nil {
		t.Fatalf("VerifyHashChain() error = %v", err)
	}
	if !valid {
		t.Error("VerifyHashChain() on empty repo should be valid")
	}
}

func TestInMemoryRepository_VerifyHashChain_Valid(t *testing.T) {
	repo := NewInMemoryRepository()

	// Add multiple entries
	entries := []LogEntry{
		{UserDID: "user1", EntityType: "scene", EntityID: "scene-1", Action: "scene_create", Outcome: OutcomeSuccess},
		{UserDID: "user1", EntityType: "scene", EntityID: "scene-1", Action: "scene_update", Outcome: OutcomeSuccess},
		{UserDID: "user2", EntityType: "event", EntityID: "event-1", Action: "event_create", Outcome: OutcomeSuccess},
		{UserDID: "user3", EntityType: "payment", EntityID: "pay-1", Action: "payment_create", Outcome: OutcomeSuccess},
		{UserDID: "user3", EntityType: "payment", EntityID: "pay-1", Action: "payment_success", Outcome: OutcomeSuccess},
	}

	for _, entry := range entries {
		_, err := repo.LogAccess(entry)
		if err != nil {
			t.Fatalf("LogAccess() error = %v", err)
		}
	}

	// Verify the chain
	valid, err := repo.VerifyHashChain()
	if err != nil {
		t.Fatalf("VerifyHashChain() error = %v", err)
	}
	if !valid {
		t.Error("VerifyHashChain() should be valid for untampered chain")
	}
}

func TestInMemoryRepository_VerifyHashChain_TamperedData(t *testing.T) {
	repo := NewInMemoryRepository()

	// Add entries
	entry1 := LogEntry{UserDID: "user1", EntityType: "scene", EntityID: "scene-1", Action: "scene_create", Outcome: OutcomeSuccess}
	log1, err := repo.LogAccess(entry1)
	if err != nil {
		t.Fatalf("LogAccess() error = %v", err)
	}

	entry2 := LogEntry{UserDID: "user1", EntityType: "scene", EntityID: "scene-1", Action: "scene_update", Outcome: OutcomeSuccess}
	_, err = repo.LogAccess(entry2)
	if err != nil {
		t.Fatalf("LogAccess() error = %v", err)
	}

	// Tamper with the first entry's action
	// Note: This test is specific to InMemoryRepository implementation.
	// For PostgresAuditRepository, tampering detection would be tested
	// by directly modifying database records via SQL.
	repo.mu.Lock()
	repo.logs[log1.ID].Action = "scene_delete" // Tamper
	repo.mu.Unlock()

	// Verify should fail
	valid, err := repo.VerifyHashChain()
	if err != nil {
		t.Fatalf("VerifyHashChain() error = %v", err)
	}
	if valid {
		t.Error("VerifyHashChain() should be invalid for tampered data")
	}
}

func TestInMemoryRepository_OutcomeField(t *testing.T) {
	repo := NewInMemoryRepository()

	// Test success outcome
	entry1 := LogEntry{
		UserDID:    "user1",
		EntityType: "payment",
		EntityID:   "pay-1",
		Action:     "payment_create",
		Outcome:    OutcomeSuccess,
	}
	log1, err := repo.LogAccess(entry1)
	if err != nil {
		t.Fatalf("LogAccess() error = %v", err)
	}
	if log1.Outcome != OutcomeSuccess {
		t.Errorf("LogAccess() Outcome = %q, want %q", log1.Outcome, OutcomeSuccess)
	}

	// Test failure outcome
	entry2 := LogEntry{
		UserDID:    "user1",
		EntityType: "payment",
		EntityID:   "pay-1",
		Action:     "payment_failure",
		Outcome:    OutcomeFailure,
	}
	log2, err := repo.LogAccess(entry2)
	if err != nil {
		t.Fatalf("LogAccess() error = %v", err)
	}
	if log2.Outcome != OutcomeFailure {
		t.Errorf("LogAccess() Outcome = %q, want %q", log2.Outcome, OutcomeFailure)
	}

	// Test default to success when outcome not provided
	entry3 := LogEntry{
		UserDID:    "user2",
		EntityType: "scene",
		EntityID:   "scene-1",
		Action:     "scene_create",
		Outcome:    "", // Empty should default to success
	}
	log3, err := repo.LogAccess(entry3)
	if err != nil {
		t.Fatalf("LogAccess() error = %v", err)
	}
	if log3.Outcome != OutcomeSuccess {
		t.Errorf("LogAccess() with empty Outcome = %q, want %q (default)", log3.Outcome, OutcomeSuccess)
	}
}
