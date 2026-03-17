package membership

import (
	"math"
	"testing"
	"time"
)

func strPtr(s string) *string {
	return &s
}

func TestMembershipRepository_Upsert_Insert(t *testing.T) {
	repo := NewInMemoryMembershipRepository()
	did := "did:plc:alice123"
	rkey := "membership456"

	membership := &Membership{
		SceneID:     "scene-1",
		UserDID:     "did:plc:user789",
		Role:        "member",
		Status:      "active",
		TrustWeight: 0.8,
		RecordDID:   strPtr(did),
		RecordRKey:  strPtr(rkey),
	}

	result, err := repo.Upsert(membership)
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

	if retrieved.Role != "member" {
		t.Errorf("Expected role 'member', got %s", retrieved.Role)
	}
}

func TestMembershipRepository_Upsert_Update(t *testing.T) {
	repo := NewInMemoryMembershipRepository()
	did := "did:plc:alice123"
	rkey := "membership456"

	// First insert
	membership := &Membership{
		SceneID:     "scene-1",
		UserDID:     "did:plc:user789",
		Role:        "member",
		Status:      "active",
		TrustWeight: 0.5,
		RecordDID:   strPtr(did),
		RecordRKey:  strPtr(rkey),
	}

	result1, err := repo.Upsert(membership)
	if err != nil {
		t.Fatalf("First upsert failed: %v", err)
	}

	if !result1.Inserted {
		t.Error("Expected insert on first upsert")
	}

	// Second upsert with same record key
	membership2 := &Membership{
		SceneID:     "scene-2",
		UserDID:     "did:plc:user789",
		Role:        "curator",
		Status:      "active",
		TrustWeight: 0.9,
		RecordDID:   strPtr(did),
		RecordRKey:  strPtr(rkey),
	}

	result2, err := repo.Upsert(membership2)
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

	if retrieved.Role != "curator" {
		t.Errorf("Expected updated role 'curator', got %s", retrieved.Role)
	}

	if retrieved.TrustWeight != 0.9 {
		t.Errorf("Expected updated trust weight 0.9, got %f", retrieved.TrustWeight)
	}
}

func TestMembershipRepository_Upsert_Idempotent(t *testing.T) {
	repo := NewInMemoryMembershipRepository()
	did := "did:plc:alice123"
	rkey := "membership456"

	membership := &Membership{
		SceneID:     "scene-1",
		UserDID:     "did:plc:user789",
		Role:        "member",
		Status:      "active",
		TrustWeight: 0.5,
		RecordDID:   strPtr(did),
		RecordRKey:  strPtr(rkey),
	}

	// First upsert
	result1, err := repo.Upsert(membership)
	if err != nil {
		t.Fatalf("First upsert failed: %v", err)
	}

	// Second upsert with same content
	result2, err := repo.Upsert(membership)
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

func TestMembershipRepository_Upsert_WithoutRecordKey(t *testing.T) {
	repo := NewInMemoryMembershipRepository()

	membership := &Membership{
		SceneID:     "scene-1",
		UserDID:     "did:plc:user789",
		Role:        "member",
		Status:      "active",
		TrustWeight: 0.5,
	}

	result, err := repo.Upsert(membership)
	if err != nil {
		t.Fatalf("Upsert failed: %v", err)
	}

	if !result.Inserted {
		t.Error("Expected insert when no record key provided")
	}

	// Second upsert without record key should also insert
	result2, err := repo.Upsert(membership)
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

func TestMembershipRepository_GetByRecordKey_NotFound(t *testing.T) {
	repo := NewInMemoryMembershipRepository()

	membership, err := repo.GetByRecordKey("did:plc:alice123", "nonexistent")
	if err != ErrMembershipNotFound {
		t.Errorf("Expected ErrMembershipNotFound, got %v", err)
	}

	if membership != nil {
		t.Error("Expected nil membership for non-existent record")
	}
}

func TestMembershipRepository_GetByID_AfterUpsert(t *testing.T) {
	repo := NewInMemoryMembershipRepository()
	did := "did:plc:alice123"
	rkey := "membership456"

	membership := &Membership{
		SceneID:     "scene-1",
		UserDID:     "did:plc:user789",
		Role:        "member",
		Status:      "active",
		TrustWeight: 0.5,
		RecordDID:   strPtr(did),
		RecordRKey:  strPtr(rkey),
	}

	result, err := repo.Upsert(membership)
	if err != nil {
		t.Fatalf("Upsert failed: %v", err)
	}

	// Retrieve by UUID
	retrieved, err := repo.GetByID(result.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}

	if retrieved.Role != "member" {
		t.Errorf("Expected role 'member', got %s", retrieved.Role)
	}
}

func TestMembershipRepository_GetBySceneAndUser(t *testing.T) {
	repo := NewInMemoryMembershipRepository()

	membership := &Membership{
		SceneID:     "scene-123",
		UserDID:     "did:plc:user456",
		Role:        "curator",
		Status:      "active",
		TrustWeight: 0.8,
	}

	result, err := repo.Upsert(membership)
	if err != nil {
		t.Fatalf("Upsert failed: %v", err)
	}

	// Retrieve by scene and user
	retrieved, err := repo.GetBySceneAndUser("scene-123", "did:plc:user456")
	if err != nil {
		t.Fatalf("GetBySceneAndUser failed: %v", err)
	}

	if retrieved.ID != result.ID {
		t.Errorf("Expected ID %s, got %s", result.ID, retrieved.ID)
	}

	if retrieved.Role != "curator" {
		t.Errorf("Expected role 'curator', got %s", retrieved.Role)
	}

	// Try with non-existent combination
	_, err = repo.GetBySceneAndUser("scene-123", "did:plc:other")
	if err != ErrMembershipNotFound {
		t.Errorf("Expected ErrMembershipNotFound, got %v", err)
	}

	_, err = repo.GetBySceneAndUser("other-scene", "did:plc:user456")
	if err != ErrMembershipNotFound {
		t.Errorf("Expected ErrMembershipNotFound, got %v", err)
	}
}

func TestMembershipRepository_UpdateStatus(t *testing.T) {
	repo := NewInMemoryMembershipRepository()

	membership := &Membership{
		SceneID:     "scene-123",
		UserDID:     "did:plc:user456",
		Role:        "member",
		Status:      "pending",
		TrustWeight: 0.5,
	}

	result, err := repo.Upsert(membership)
	if err != nil {
		t.Fatalf("Upsert failed: %v", err)
	}

	// Update status to active
	now := time.Now()
	err = repo.UpdateStatus(result.ID, "active", &now)
	if err != nil {
		t.Fatalf("UpdateStatus failed: %v", err)
	}

	// Verify update
	retrieved, err := repo.GetByID(result.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}

	if retrieved.Status != "active" {
		t.Errorf("Expected status 'active', got %s", retrieved.Status)
	}

	if !retrieved.Since.Equal(now) {
		t.Errorf("Expected since %v, got %v", now, retrieved.Since)
	}

	// Update status without changing since
	err = repo.UpdateStatus(result.ID, "rejected", nil)
	if err != nil {
		t.Fatalf("UpdateStatus failed: %v", err)
	}

	retrieved, err = repo.GetByID(result.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}

	if retrieved.Status != "rejected" {
		t.Errorf("Expected status 'rejected', got %s", retrieved.Status)
	}

	if !retrieved.Since.Equal(now) {
		t.Errorf("Expected since to remain %v, got %v", now, retrieved.Since)
	}

	// Try updating non-existent membership
	err = repo.UpdateStatus("nonexistent", "active", nil)
	if err != ErrMembershipNotFound {
		t.Errorf("Expected ErrMembershipNotFound, got %v", err)
	}
}

func TestMembershipRepository_ListByScene(t *testing.T) {
	repo := NewInMemoryMembershipRepository()

	// Create multiple memberships
	memberships := []*Membership{
		{
			SceneID:     "scene-123",
			UserDID:     "did:plc:user1",
			Role:        "member",
			Status:      "active",
			TrustWeight: 0.5,
		},
		{
			SceneID:     "scene-123",
			UserDID:     "did:plc:user2",
			Role:        "curator",
			Status:      "active",
			TrustWeight: 0.8,
		},
		{
			SceneID:     "scene-123",
			UserDID:     "did:plc:user3",
			Role:        "member",
			Status:      "pending",
			TrustWeight: 0.5,
		},
		{
			SceneID:     "scene-456",
			UserDID:     "did:plc:user4",
			Role:        "member",
			Status:      "active",
			TrustWeight: 0.6,
		},
	}

	for _, m := range memberships {
		if _, err := repo.Upsert(m); err != nil {
			t.Fatalf("Upsert failed: %v", err)
		}
	}

	// List all memberships for scene-123
	allMembers, err := repo.ListByScene("scene-123", "")
	if err != nil {
		t.Fatalf("ListByScene failed: %v", err)
	}

	if len(allMembers) != 3 {
		t.Errorf("Expected 3 memberships, got %d", len(allMembers))
	}

	// List only active memberships for scene-123
	activeMembers, err := repo.ListByScene("scene-123", "active")
	if err != nil {
		t.Fatalf("ListByScene failed: %v", err)
	}

	if len(activeMembers) != 2 {
		t.Errorf("Expected 2 active memberships, got %d", len(activeMembers))
	}

	// List pending memberships for scene-123
	pendingMembers, err := repo.ListByScene("scene-123", "pending")
	if err != nil {
		t.Fatalf("ListByScene failed: %v", err)
	}

	if len(pendingMembers) != 1 {
		t.Errorf("Expected 1 pending membership, got %d", len(pendingMembers))
	}

	// List memberships for scene-456
	scene456Members, err := repo.ListByScene("scene-456", "")
	if err != nil {
		t.Fatalf("ListByScene failed: %v", err)
	}

	if len(scene456Members) != 1 {
		t.Errorf("Expected 1 membership for scene-456, got %d", len(scene456Members))
	}

	// List memberships for non-existent scene
	nonExistent, err := repo.ListByScene("nonexistent", "")
	if err != nil {
		t.Fatalf("ListByScene failed: %v", err)
	}

	if len(nonExistent) != 0 {
		t.Errorf("Expected 0 memberships for non-existent scene, got %d", len(nonExistent))
	}
}

func TestMembershipRepository_CountByScenes(t *testing.T) {
	repo := NewInMemoryMembershipRepository()

	// Create memberships for different scenes
	scene1 := "scene-1"
	scene2 := "scene-2"
	scene3 := "scene-3"

	// Scene 1: 2 active, 1 pending
	m1 := &Membership{ID: "m1", SceneID: scene1, UserDID: "user1", Role: "member", TrustWeight: 0.5, Status: "active"}
	m2 := &Membership{ID: "m2", SceneID: scene1, UserDID: "user2", Role: "member", TrustWeight: 0.5, Status: "active"}
	m3 := &Membership{ID: "m3", SceneID: scene1, UserDID: "user3", Role: "member", TrustWeight: 0.5, Status: "pending"}

	// Scene 2: 1 active
	m4 := &Membership{ID: "m4", SceneID: scene2, UserDID: "user4", Role: "member", TrustWeight: 0.5, Status: "active"}

	// Scene 3: no memberships

	for _, m := range []*Membership{m1, m2, m3, m4} {
		if _, err := repo.Upsert(m); err != nil {
			t.Fatalf("Upsert failed: %v", err)
		}
	}

	// Test: Count active memberships for all scenes
	counts, err := repo.CountByScenes([]string{scene1, scene2, scene3}, "active")
	if err != nil {
		t.Fatalf("CountByScenes failed: %v", err)
	}

	if counts[scene1] != 2 {
		t.Errorf("Expected 2 active members for scene1, got %d", counts[scene1])
	}
	if counts[scene2] != 1 {
		t.Errorf("Expected 1 active member for scene2, got %d", counts[scene2])
	}
	if counts[scene3] != 0 {
		t.Errorf("Expected 0 active members for scene3, got %d", counts[scene3])
	}

	// Test: Count all memberships (including pending)
	allCounts, err := repo.CountByScenes([]string{scene1, scene2}, "")
	if err != nil {
		t.Fatalf("CountByScenes failed: %v", err)
	}

	if allCounts[scene1] != 3 {
		t.Errorf("Expected 3 total members for scene1, got %d", allCounts[scene1])
	}
	if allCounts[scene2] != 1 {
		t.Errorf("Expected 1 total member for scene2, got %d", allCounts[scene2])
	}
}

func TestMembershipRepository_CountByScenes_EmptyInput(t *testing.T) {
	repo := NewInMemoryMembershipRepository()

	counts, err := repo.CountByScenes([]string{}, "active")
	if err != nil {
		t.Fatalf("CountByScenes failed: %v", err)
	}

	if len(counts) != 0 {
		t.Errorf("Expected empty map, got %d entries", len(counts))
	}
}

func TestMembershipRepository_UpdateRole(t *testing.T) {
	repo := NewInMemoryMembershipRepository()

	// Create a membership
	m := &Membership{
		ID:          "test-id",
		SceneID:     "scene-1",
		UserDID:     "user1",
		Role:        "member",
		TrustWeight: 0.5,
		Status:      "active",
	}

	result, err := repo.Upsert(m)
	if err != nil {
		t.Fatalf("Upsert failed: %v", err)
	}

	// Update role to curator
	err = repo.UpdateRole(result.ID, "curator")
	if err != nil {
		t.Fatalf("UpdateRole failed: %v", err)
	}

	// Verify the role was updated
	updated, err := repo.GetByID(result.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}

	if updated.Role != "curator" {
		t.Errorf("Expected role 'curator', got %q", updated.Role)
	}

	// Try invalid role
	err = repo.UpdateRole(result.ID, "invalid_role")
	if err == nil {
		t.Error("Expected error for invalid role, got nil")
	}

	// Try non-existent membership
	err = repo.UpdateRole("non-existent-id", "owner")
	if err == nil {
		t.Error("Expected error for non-existent membership, got nil")
	}
}

func TestMembership_EffectiveWeight(t *testing.T) {
	tests := []struct {
		name        string
		role        string
		trustWeight float64
		want        float64
	}{
		{
			name:        "owner with full trust",
			role:        "owner",
			trustWeight: 1.0,
			want:        1.0,
		},
		{
			name:        "curator with 0.8 trust",
			role:        "curator",
			trustWeight: 0.8,
			want:        0.64,
		},
		{
			name:        "member with 0.5 trust",
			role:        "member",
			trustWeight: 0.5,
			want:        0.25,
		},
		{
			name:        "guest with full trust",
			role:        "guest",
			trustWeight: 1.0,
			want:        0.3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Membership{
				Role:        tt.role,
				TrustWeight: tt.trustWeight,
			}
			got := m.EffectiveWeight()
			if math.Abs(got-tt.want) > 1e-9 {
				t.Errorf("EffectiveWeight() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMembership_Validate(t *testing.T) {
	tests := []struct {
		name        string
		membership  *Membership
		expectError bool
	}{
		{
			name: "valid membership",
			membership: &Membership{
				Role:        "member",
				TrustWeight: 0.5,
			},
			expectError: false,
		},
		{
			name: "invalid role",
			membership: &Membership{
				Role:        "invalid",
				TrustWeight: 0.5,
			},
			expectError: true,
		},
		{
			name: "invalid trust weight too high",
			membership: &Membership{
				Role:        "member",
				TrustWeight: 1.5,
			},
			expectError: true,
		},
		{
			name: "invalid trust weight negative",
			membership: &Membership{
				Role:        "member",
				TrustWeight: -0.1,
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.membership.Validate()
			if (err != nil) != tt.expectError {
				t.Errorf("Validate() error = %v, expectError = %v", err, tt.expectError)
			}
		})
	}
}

func TestMembershipRepository_Upsert_ValidationError(t *testing.T) {
	repo := NewInMemoryMembershipRepository()

	// Test invalid role
	m1 := &Membership{
		SceneID:     "scene-1",
		UserDID:     "user1",
		Role:        "invalid_role",
		TrustWeight: 0.5,
		Status:      "active",
	}
	_, err := repo.Upsert(m1)
	if err == nil {
		t.Error("Expected validation error for invalid role, got nil")
	}

	// Test invalid trust weight
	m2 := &Membership{
		SceneID:     "scene-1",
		UserDID:     "user1",
		Role:        "member",
		TrustWeight: 1.5,
		Status:      "active",
	}
	_, err = repo.Upsert(m2)
	if err == nil {
		t.Error("Expected validation error for invalid trust weight, got nil")
	}
}
