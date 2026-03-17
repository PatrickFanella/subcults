package scene

import (
	"testing"
)

func TestRSVPRepository_Upsert_Create(t *testing.T) {
	repo := NewInMemoryRSVPRepository()

	rsvp := &RSVP{
		EventID: "event-1",
		UserID:  "user-1",
		Status:  "going",
	}

	// Create new RSVP
	if err := repo.Upsert(rsvp); err != nil {
		t.Fatalf("Upsert failed: %v", err)
	}

	// Retrieve and verify
	stored, err := repo.GetByEventAndUser("event-1", "user-1")
	if err != nil {
		t.Fatalf("GetByEventAndUser failed: %v", err)
	}

	if stored.Status != "going" {
		t.Errorf("Expected status 'going', got %s", stored.Status)
	}
	if stored.CreatedAt == nil {
		t.Error("Expected CreatedAt to be set")
	}
	if stored.UpdatedAt == nil {
		t.Error("Expected UpdatedAt to be set")
	}
}

func TestRSVPRepository_Upsert_Update(t *testing.T) {
	repo := NewInMemoryRSVPRepository()

	// Create initial RSVP with "maybe" status
	rsvp := &RSVP{
		EventID: "event-1",
		UserID:  "user-1",
		Status:  "maybe",
	}
	if err := repo.Upsert(rsvp); err != nil {
		t.Fatalf("Initial Upsert failed: %v", err)
	}

	// Update to "going" status
	rsvp.Status = "going"
	if err := repo.Upsert(rsvp); err != nil {
		t.Fatalf("Update Upsert failed: %v", err)
	}

	// Verify status was updated
	stored, err := repo.GetByEventAndUser("event-1", "user-1")
	if err != nil {
		t.Fatalf("GetByEventAndUser failed: %v", err)
	}

	if stored.Status != "going" {
		t.Errorf("Expected status 'going', got %s", stored.Status)
	}
}

func TestRSVPRepository_Upsert_Idempotent(t *testing.T) {
	repo := NewInMemoryRSVPRepository()

	rsvp := &RSVP{
		EventID: "event-1",
		UserID:  "user-1",
		Status:  "going",
	}

	// First upsert
	if err := repo.Upsert(rsvp); err != nil {
		t.Fatalf("First Upsert failed: %v", err)
	}

	// Second upsert with same status - should be idempotent
	if err := repo.Upsert(rsvp); err != nil {
		t.Fatalf("Second Upsert failed: %v", err)
	}

	// Verify only one RSVP exists
	stored, err := repo.GetByEventAndUser("event-1", "user-1")
	if err != nil {
		t.Fatalf("GetByEventAndUser failed: %v", err)
	}

	if stored.Status != "going" {
		t.Errorf("Expected status 'going', got %s", stored.Status)
	}
}

func TestRSVPRepository_Delete_Success(t *testing.T) {
	repo := NewInMemoryRSVPRepository()

	// Create RSVP
	rsvp := &RSVP{
		EventID: "event-1",
		UserID:  "user-1",
		Status:  "going",
	}
	if err := repo.Upsert(rsvp); err != nil {
		t.Fatalf("Upsert failed: %v", err)
	}

	// Delete RSVP
	if err := repo.Delete("event-1", "user-1"); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify RSVP is deleted
	_, err := repo.GetByEventAndUser("event-1", "user-1")
	if err != ErrRSVPNotFound {
		t.Errorf("Expected ErrRSVPNotFound, got %v", err)
	}
}

func TestRSVPRepository_Delete_NotFound(t *testing.T) {
	repo := NewInMemoryRSVPRepository()

	// Try to delete non-existent RSVP
	err := repo.Delete("event-1", "user-1")
	if err != ErrRSVPNotFound {
		t.Errorf("Expected ErrRSVPNotFound, got %v", err)
	}
}

func TestRSVPRepository_GetByEventAndUser_NotFound(t *testing.T) {
	repo := NewInMemoryRSVPRepository()

	// Try to get non-existent RSVP
	_, err := repo.GetByEventAndUser("event-1", "user-1")
	if err != ErrRSVPNotFound {
		t.Errorf("Expected ErrRSVPNotFound, got %v", err)
	}
}

func TestRSVPRepository_GetCountsByEvent_Empty(t *testing.T) {
	repo := NewInMemoryRSVPRepository()

	// Get counts for event with no RSVPs
	counts, err := repo.GetCountsByEvent("event-1")
	if err != nil {
		t.Fatalf("GetCountsByEvent failed: %v", err)
	}

	if counts.Going != 0 {
		t.Errorf("Expected Going count 0, got %d", counts.Going)
	}
	if counts.Maybe != 0 {
		t.Errorf("Expected Maybe count 0, got %d", counts.Maybe)
	}
}

func TestRSVPRepository_GetCountsByEvent_Multiple(t *testing.T) {
	repo := NewInMemoryRSVPRepository()

	// Create multiple RSVPs for same event
	rsvps := []*RSVP{
		{EventID: "event-1", UserID: "user-1", Status: "going"},
		{EventID: "event-1", UserID: "user-2", Status: "going"},
		{EventID: "event-1", UserID: "user-3", Status: "maybe"},
		{EventID: "event-1", UserID: "user-4", Status: "maybe"},
		{EventID: "event-1", UserID: "user-5", Status: "maybe"},
		// RSVPs for different event (should not be counted)
		{EventID: "event-2", UserID: "user-6", Status: "going"},
	}

	for _, rsvp := range rsvps {
		if err := repo.Upsert(rsvp); err != nil {
			t.Fatalf("Upsert failed: %v", err)
		}
	}

	// Get counts for event-1
	counts, err := repo.GetCountsByEvent("event-1")
	if err != nil {
		t.Fatalf("GetCountsByEvent failed: %v", err)
	}

	if counts.Going != 2 {
		t.Errorf("Expected Going count 2, got %d", counts.Going)
	}
	if counts.Maybe != 3 {
		t.Errorf("Expected Maybe count 3, got %d", counts.Maybe)
	}
}

func TestRSVPRepository_GetCountsByEvent_AfterDelete(t *testing.T) {
	repo := NewInMemoryRSVPRepository()

	// Create RSVPs
	rsvps := []*RSVP{
		{EventID: "event-1", UserID: "user-1", Status: "going"},
		{EventID: "event-1", UserID: "user-2", Status: "going"},
		{EventID: "event-1", UserID: "user-3", Status: "maybe"},
	}

	for _, rsvp := range rsvps {
		if err := repo.Upsert(rsvp); err != nil {
			t.Fatalf("Upsert failed: %v", err)
		}
	}

	// Delete one RSVP
	if err := repo.Delete("event-1", "user-1"); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Get counts - should reflect deletion
	counts, err := repo.GetCountsByEvent("event-1")
	if err != nil {
		t.Fatalf("GetCountsByEvent failed: %v", err)
	}

	if counts.Going != 1 {
		t.Errorf("Expected Going count 1 after delete, got %d", counts.Going)
	}
	if counts.Maybe != 1 {
		t.Errorf("Expected Maybe count 1, got %d", counts.Maybe)
	}
}

func TestRSVPRepository_GetCountsByEvent_AfterStatusChange(t *testing.T) {
	repo := NewInMemoryRSVPRepository()

	// Create RSVPs
	rsvps := []*RSVP{
		{EventID: "event-1", UserID: "user-1", Status: "going"},
		{EventID: "event-1", UserID: "user-2", Status: "going"},
		{EventID: "event-1", UserID: "user-3", Status: "maybe"},
	}

	for _, rsvp := range rsvps {
		if err := repo.Upsert(rsvp); err != nil {
			t.Fatalf("Upsert failed: %v", err)
		}
	}

	// Change user-1 from "going" to "maybe"
	updatedRSVP := &RSVP{
		EventID: "event-1",
		UserID:  "user-1",
		Status:  "maybe",
	}
	if err := repo.Upsert(updatedRSVP); err != nil {
		t.Fatalf("Upsert failed: %v", err)
	}

	// Get counts - should reflect status change
	counts, err := repo.GetCountsByEvent("event-1")
	if err != nil {
		t.Fatalf("GetCountsByEvent failed: %v", err)
	}

	if counts.Going != 1 {
		t.Errorf("Expected Going count 1 after status change, got %d", counts.Going)
	}
	if counts.Maybe != 2 {
		t.Errorf("Expected Maybe count 2 after status change, got %d", counts.Maybe)
	}
}
