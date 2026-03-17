package post

import (
	"testing"
)

func strPtr(s string) *string {
	return &s
}

func TestPostRepository_Upsert_Insert(t *testing.T) {
	repo := NewInMemoryPostRepository()
	did := "did:example:alice"
	rkey := "post123"

	post := &Post{
		AuthorDID:  did,
		Text:       "Hello world",
		RecordDID:  strPtr(did),
		RecordRKey: strPtr(rkey),
	}

	result, err := repo.Upsert(post)
	if err != nil {
		t.Fatalf("Upsert failed: %v", err)
	}

	if !result.Inserted {
		t.Error("Expected insert, got update")
	}

	if result.ID == "" {
		t.Error("Expected non-empty ID")
	}

	// Verify we can retrieve it
	retrieved, err := repo.GetByRecordKey(did, rkey)
	if err != nil {
		t.Fatalf("GetByRecordKey failed: %v", err)
	}

	if retrieved.Text != "Hello world" {
		t.Errorf("Expected text 'Hello world', got %s", retrieved.Text)
	}
}

func TestPostRepository_Upsert_Update(t *testing.T) {
	repo := NewInMemoryPostRepository()
	did := "did:example:alice"
	rkey := "post123"

	// First insert
	post := &Post{
		AuthorDID:  did,
		Text:       "Hello world",
		RecordDID:  strPtr(did),
		RecordRKey: strPtr(rkey),
	}

	result1, err := repo.Upsert(post)
	if err != nil {
		t.Fatalf("First upsert failed: %v", err)
	}

	if !result1.Inserted {
		t.Error("Expected insert on first upsert")
	}

	// Second upsert with same record key
	post2 := &Post{
		AuthorDID:  did,
		Text:       "Updated text",
		RecordDID:  strPtr(did),
		RecordRKey: strPtr(rkey),
	}

	result2, err := repo.Upsert(post2)
	if err != nil {
		t.Fatalf("Second upsert failed: %v", err)
	}

	if result2.Inserted {
		t.Error("Expected update, got insert")
	}

	if result1.ID != result2.ID {
		t.Errorf("Expected same ID, got %s and %s", result1.ID, result2.ID)
	}

	// Verify update was persisted
	retrieved, err := repo.GetByRecordKey(did, rkey)
	if err != nil {
		t.Fatalf("GetByRecordKey failed: %v", err)
	}

	if retrieved.Text != "Updated text" {
		t.Errorf("Expected updated text, got %s", retrieved.Text)
	}
}

func TestPostRepository_Upsert_Idempotent(t *testing.T) {
	repo := NewInMemoryPostRepository()
	did := "did:example:alice"
	rkey := "post123"

	post := &Post{
		AuthorDID:  did,
		Text:       "Same content",
		RecordDID:  strPtr(did),
		RecordRKey: strPtr(rkey),
	}

	// First upsert
	result1, err := repo.Upsert(post)
	if err != nil {
		t.Fatalf("First upsert failed: %v", err)
	}

	// Second upsert with same content
	result2, err := repo.Upsert(post)
	if err != nil {
		t.Fatalf("Second upsert failed: %v", err)
	}

	// Should update, not insert
	if result2.Inserted {
		t.Error("Expected update (idempotent), got insert")
	}

	if result1.ID != result2.ID {
		t.Error("Idempotent upserts should return same ID")
	}
}

func TestPostRepository_Upsert_WithoutRecordKey(t *testing.T) {
	repo := NewInMemoryPostRepository()

	post := &Post{
		AuthorDID: "did:example:alice",
		Text:      "No record key",
	}

	result, err := repo.Upsert(post)
	if err != nil {
		t.Fatalf("Upsert failed: %v", err)
	}

	if !result.Inserted {
		t.Error("Expected insert when no record key provided")
	}

	// Second upsert without record key should also insert
	result2, err := repo.Upsert(post)
	if err != nil {
		t.Fatalf("Second upsert failed: %v", err)
	}

	if !result2.Inserted {
		t.Error("Expected insert when no record key provided")
	}

	// IDs should be different
	if result.ID == result2.ID {
		t.Error("Expected different IDs for separate inserts")
	}
}

func TestPostRepository_GetByRecordKey_NotFound(t *testing.T) {
	repo := NewInMemoryPostRepository()

	post, err := repo.GetByRecordKey("did:example:alice", "nonexistent")
	if err != ErrPostNotFound {
		t.Errorf("Expected ErrPostNotFound, got %v", err)
	}

	if post != nil {
		t.Error("Expected nil post for non-existent record")
	}
}

func TestPostRepository_GetByID_AfterUpsert(t *testing.T) {
	repo := NewInMemoryPostRepository()
	did := "did:example:alice"
	rkey := "post123"

	post := &Post{
		AuthorDID:  did,
		Text:       "Test post",
		RecordDID:  strPtr(did),
		RecordRKey: strPtr(rkey),
	}

	result, err := repo.Upsert(post)
	if err != nil {
		t.Fatalf("Upsert failed: %v", err)
	}

	// Retrieve by UUID
	retrieved, err := repo.GetByID(result.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}

	if retrieved.Text != "Test post" {
		t.Errorf("Expected text 'Test post', got %s", retrieved.Text)
	}
}
