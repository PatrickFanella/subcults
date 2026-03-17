package alliance

import (
	"testing"
)

func strPtr(s string) *string {
	return &s
}

func TestAllianceRepository_Upsert_Insert(t *testing.T) {
	repo := NewInMemoryAllianceRepository()
	did := "did:plc:alice123"
	rkey := "alliance456"

	alliance := &Alliance{
		FromSceneID: "scene-1",
		ToSceneID:   "scene-2",
		Weight:      0.8,
		Status:      "active",
		RecordDID:   strPtr(did),
		RecordRKey:  strPtr(rkey),
	}

	result, err := repo.Upsert(alliance)
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

	if retrieved.Weight != 0.8 {
		t.Errorf("Expected weight 0.8, got %f", retrieved.Weight)
	}
}

func TestAllianceRepository_Upsert_Update(t *testing.T) {
	repo := NewInMemoryAllianceRepository()
	did := "did:plc:alice123"
	rkey := "alliance456"

	// First insert
	alliance := &Alliance{
		FromSceneID: "scene-1",
		ToSceneID:   "scene-2",
		Weight:      0.5,
		Status:      "active",
		RecordDID:   strPtr(did),
		RecordRKey:  strPtr(rkey),
	}

	result1, err := repo.Upsert(alliance)
	if err != nil {
		t.Fatalf("First upsert failed: %v", err)
	}

	if !result1.Inserted {
		t.Error("Expected insert on first upsert")
	}

	// Second upsert with same record key
	reasonText := "Updated reason"
	alliance2 := &Alliance{
		FromSceneID: "scene-1",
		ToSceneID:   "scene-3",
		Weight:      0.9,
		Status:      "active",
		Reason:      &reasonText,
		RecordDID:   strPtr(did),
		RecordRKey:  strPtr(rkey),
	}

	result2, err := repo.Upsert(alliance2)
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

	if retrieved.Weight != 0.9 {
		t.Errorf("Expected updated weight 0.9, got %f", retrieved.Weight)
	}

	if retrieved.ToSceneID != "scene-3" {
		t.Errorf("Expected ToSceneID 'scene-3', got %s", retrieved.ToSceneID)
	}
}

func TestAllianceRepository_Upsert_Idempotent(t *testing.T) {
	repo := NewInMemoryAllianceRepository()
	did := "did:plc:alice123"
	rkey := "alliance456"

	alliance := &Alliance{
		FromSceneID: "scene-1",
		ToSceneID:   "scene-2",
		Weight:      0.5,
		Status:      "active",
		RecordDID:   strPtr(did),
		RecordRKey:  strPtr(rkey),
	}

	// First upsert
	result1, err := repo.Upsert(alliance)
	if err != nil {
		t.Fatalf("First upsert failed: %v", err)
	}

	// Second upsert with same content
	result2, err := repo.Upsert(alliance)
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

func TestAllianceRepository_Upsert_WithoutRecordKey(t *testing.T) {
	repo := NewInMemoryAllianceRepository()

	alliance := &Alliance{
		FromSceneID: "scene-1",
		ToSceneID:   "scene-2",
		Weight:      0.5,
		Status:      "active",
	}

	result, err := repo.Upsert(alliance)
	if err != nil {
		t.Fatalf("Upsert failed: %v", err)
	}

	if !result.Inserted {
		t.Error("Expected insert when no record key provided")
	}

	// Second upsert without record key should also insert
	result2, err := repo.Upsert(alliance)
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

func TestAllianceRepository_GetByRecordKey_NotFound(t *testing.T) {
	repo := NewInMemoryAllianceRepository()

	alliance, err := repo.GetByRecordKey("did:plc:alice123", "nonexistent")
	if err != ErrAllianceNotFound {
		t.Errorf("Expected ErrAllianceNotFound, got %v", err)
	}

	if alliance != nil {
		t.Error("Expected nil alliance for non-existent record")
	}
}

func TestAllianceRepository_GetByID_AfterUpsert(t *testing.T) {
	repo := NewInMemoryAllianceRepository()
	did := "did:plc:alice123"
	rkey := "alliance456"

	alliance := &Alliance{
		FromSceneID: "scene-1",
		ToSceneID:   "scene-2",
		Weight:      0.5,
		Status:      "active",
		RecordDID:   strPtr(did),
		RecordRKey:  strPtr(rkey),
	}

	result, err := repo.Upsert(alliance)
	if err != nil {
		t.Fatalf("Upsert failed: %v", err)
	}

	// Retrieve by UUID
	retrieved, err := repo.GetByID(result.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}

	if retrieved.Weight != 0.5 {
		t.Errorf("Expected weight 0.5, got %f", retrieved.Weight)
	}
}

func TestAllianceRepository_Insert(t *testing.T) {
	repo := NewInMemoryAllianceRepository()

	t.Run("insert with generated ID", func(t *testing.T) {
		a := &Alliance{
			FromSceneID: "scene-1",
			ToSceneID:   "scene-2",
			Weight:      0.7,
			Status:      "active",
		}

		err := repo.Insert(a)
		if err != nil {
			t.Fatalf("Insert failed: %v", err)
		}
		if a.ID == "" {
			t.Error("Expected UUID to be generated")
		}
		if a.CreatedAt.IsZero() {
			t.Error("Expected CreatedAt to be set")
		}
		if a.Since.IsZero() {
			t.Error("Expected Since to be set")
		}

		retrieved, err := repo.GetByID(a.ID)
		if err != nil {
			t.Fatalf("GetByID failed: %v", err)
		}
		if retrieved.Weight != 0.7 {
			t.Errorf("Expected weight 0.7, got %f", retrieved.Weight)
		}
	})

	t.Run("insert with preset ID", func(t *testing.T) {
		a := &Alliance{
			ID:          "custom-id-123",
			FromSceneID: "scene-3",
			ToSceneID:   "scene-4",
			Weight:      0.5,
			Status:      "pending",
		}

		err := repo.Insert(a)
		if err != nil {
			t.Fatalf("Insert failed: %v", err)
		}
		if a.ID != "custom-id-123" {
			t.Errorf("Expected custom-id-123, got %s", a.ID)
		}
	})

	t.Run("insert returns deep copy", func(t *testing.T) {
		a := &Alliance{
			FromSceneID: "scene-5",
			ToSceneID:   "scene-6",
			Weight:      0.9,
			Status:      "active",
		}
		err := repo.Insert(a)
		if err != nil {
			t.Fatalf("Insert failed: %v", err)
		}

		// Mutating the original should not affect stored copy
		a.Weight = 0.1
		retrieved, err := repo.GetByID(a.ID)
		if err != nil {
			t.Fatalf("GetByID failed: %v", err)
		}
		if retrieved.Weight != 0.9 {
			t.Errorf("Expected stored weight 0.9 (deep copy), got %f", retrieved.Weight)
		}
	})
}

func TestAllianceRepository_Update(t *testing.T) {
	repo := NewInMemoryAllianceRepository()

	t.Run("update existing alliance", func(t *testing.T) {
		a := &Alliance{
			FromSceneID: "scene-1",
			ToSceneID:   "scene-2",
			Weight:      0.5,
			Status:      "active",
		}
		_ = repo.Insert(a)

		reason := "trust increased"
		update := &Alliance{
			ID:     a.ID,
			Weight: 0.9,
			Status: "verified",
			Reason: &reason,
		}
		err := repo.Update(update)
		if err != nil {
			t.Fatalf("Update failed: %v", err)
		}

		retrieved, err := repo.GetByID(a.ID)
		if err != nil {
			t.Fatalf("GetByID failed: %v", err)
		}
		if retrieved.Weight != 0.9 {
			t.Errorf("Expected weight 0.9, got %f", retrieved.Weight)
		}
		if retrieved.Status != "verified" {
			t.Errorf("Expected status verified, got %s", retrieved.Status)
		}
		if retrieved.Reason == nil || *retrieved.Reason != "trust increased" {
			t.Error("Expected reason to be updated")
		}
		if retrieved.UpdatedAt.Before(retrieved.CreatedAt) {
			t.Error("UpdatedAt should be after CreatedAt")
		}
	})

	t.Run("update non-existent alliance", func(t *testing.T) {
		err := repo.Update(&Alliance{ID: "non-existent"})
		if err != ErrAllianceNotFound {
			t.Errorf("Expected ErrAllianceNotFound, got %v", err)
		}
	})

	t.Run("update deleted alliance", func(t *testing.T) {
		a := &Alliance{
			FromSceneID: "scene-3",
			ToSceneID:   "scene-4",
			Weight:      0.5,
			Status:      "active",
		}
		_ = repo.Insert(a)
		_ = repo.Delete(a.ID)

		err := repo.Update(&Alliance{ID: a.ID, Weight: 0.9, Status: "active"})
		if err != ErrAllianceDeleted {
			t.Errorf("Expected ErrAllianceDeleted, got %v", err)
		}
	})
}

func TestAllianceRepository_Delete(t *testing.T) {
	repo := NewInMemoryAllianceRepository()

	t.Run("delete existing alliance", func(t *testing.T) {
		a := &Alliance{
			FromSceneID: "scene-1",
			ToSceneID:   "scene-2",
			Weight:      0.5,
			Status:      "active",
		}
		_ = repo.Insert(a)

		err := repo.Delete(a.ID)
		if err != nil {
			t.Fatalf("Delete failed: %v", err)
		}

		_, err = repo.GetByID(a.ID)
		if err != ErrAllianceDeleted {
			t.Errorf("Expected ErrAllianceDeleted after delete, got %v", err)
		}
	})

	t.Run("delete non-existent alliance", func(t *testing.T) {
		err := repo.Delete("non-existent")
		if err != ErrAllianceNotFound {
			t.Errorf("Expected ErrAllianceNotFound, got %v", err)
		}
	})

	t.Run("delete already deleted alliance", func(t *testing.T) {
		a := &Alliance{
			FromSceneID: "scene-3",
			ToSceneID:   "scene-4",
			Weight:      0.5,
			Status:      "active",
		}
		_ = repo.Insert(a)
		_ = repo.Delete(a.ID)

		err := repo.Delete(a.ID)
		if err != ErrAllianceDeleted {
			t.Errorf("Expected ErrAllianceDeleted on double delete, got %v", err)
		}
	})
}
