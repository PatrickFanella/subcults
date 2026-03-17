package scene

import (
	"strconv"
	"testing"
	"time"
)

// --- SceneRepository: Delete ---

func TestSceneRepository_Delete_Success(t *testing.T) {
	repo := NewInMemorySceneRepository()

	scene := &Scene{
		ID:            "scene-del-1",
		Name:          "Delete Me",
		OwnerDID:      "did:plc:owner1",
		CoarseGeohash: "dr5regw",
		AllowPrecise:  false,
	}
	if err := repo.Insert(scene); err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	err := repo.Delete("scene-del-1")
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// GetByID should return ErrSceneDeleted for soft-deleted scene
	_, err = repo.GetByID("scene-del-1")
	if err != ErrSceneDeleted {
		t.Errorf("Expected ErrSceneDeleted after delete, got %v", err)
	}
}

func TestSceneRepository_Delete_NotFound(t *testing.T) {
	repo := NewInMemorySceneRepository()

	err := repo.Delete("nonexistent-id")
	if err != ErrSceneNotFound {
		t.Errorf("Expected ErrSceneNotFound, got %v", err)
	}
}

func TestSceneRepository_Delete_AlreadyDeleted(t *testing.T) {
	repo := NewInMemorySceneRepository()

	scene := &Scene{
		ID:            "scene-del-2",
		Name:          "Double Delete",
		OwnerDID:      "did:plc:owner1",
		CoarseGeohash: "dr5regw",
		AllowPrecise:  false,
	}
	if err := repo.Insert(scene); err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	// First delete
	if err := repo.Delete("scene-del-2"); err != nil {
		t.Fatalf("First delete failed: %v", err)
	}

	// Second delete should return ErrSceneDeleted
	err := repo.Delete("scene-del-2")
	if err != ErrSceneDeleted {
		t.Errorf("Expected ErrSceneDeleted on second delete, got %v", err)
	}
}

// --- SceneRepository: GetByRecordKey ---

func TestSceneRepository_GetByRecordKey_Success(t *testing.T) {
	repo := NewInMemorySceneRepository()

	did := "did:plc:alice123"
	rkey := "scene456"
	scene := &Scene{
		ID:            "scene-rk-1",
		Name:          "Record Key Scene",
		OwnerDID:      "did:plc:owner1",
		CoarseGeohash: "dr5regw",
		AllowPrecise:  true,
		PrecisePoint:  &Point{Lat: 40.7128, Lng: -74.006},
		RecordDID:     &did,
		RecordRKey:    &rkey,
	}

	// Must upsert to populate the keys map
	_, err := repo.Upsert(scene)
	if err != nil {
		t.Fatalf("Upsert failed: %v", err)
	}

	retrieved, err := repo.GetByRecordKey(did, rkey)
	if err != nil {
		t.Fatalf("GetByRecordKey failed: %v", err)
	}

	if retrieved.Name != "Record Key Scene" {
		t.Errorf("Expected name 'Record Key Scene', got %s", retrieved.Name)
	}
	if retrieved.PrecisePoint == nil {
		t.Error("Expected PrecisePoint to be set (consent is true)")
	}
}

func TestSceneRepository_GetByRecordKey_NotFound(t *testing.T) {
	repo := NewInMemorySceneRepository()

	_, err := repo.GetByRecordKey("did:plc:nobody", "no-rkey")
	if err != ErrSceneNotFound {
		t.Errorf("Expected ErrSceneNotFound, got %v", err)
	}
}

func TestSceneRepository_GetByRecordKey_DeepCopy(t *testing.T) {
	repo := NewInMemorySceneRepository()

	did := "did:plc:alice123"
	rkey := "scene789"
	scene := &Scene{
		ID:            "scene-rk-dc",
		Name:          "Deep Copy Scene",
		OwnerDID:      "did:plc:owner1",
		CoarseGeohash: "dr5regw",
		AllowPrecise:  true,
		PrecisePoint:  &Point{Lat: 40.7128, Lng: -74.006},
		RecordDID:     &did,
		RecordRKey:    &rkey,
	}

	if _, err := repo.Upsert(scene); err != nil {
		t.Fatalf("Upsert failed: %v", err)
	}

	retrieved, _ := repo.GetByRecordKey(did, rkey)

	// Mutate the returned copy
	retrieved.Name = "Mutated"
	retrieved.PrecisePoint.Lat = 0.0

	// Verify the repo's copy is unchanged
	original, _ := repo.GetByRecordKey(did, rkey)
	if original.Name != "Deep Copy Scene" {
		t.Error("Deep copy violated: mutation of returned scene affected repository")
	}
	if original.PrecisePoint.Lat != 40.7128 {
		t.Error("Deep copy violated: mutation of PrecisePoint affected repository")
	}
}

// --- SceneRepository: ExistsByOwnerAndName ---

func TestSceneRepository_ExistsByOwnerAndName_Exists(t *testing.T) {
	repo := NewInMemorySceneRepository()

	scene := &Scene{
		ID:            "scene-eon-1",
		Name:          "My Scene",
		OwnerDID:      "did:plc:owner1",
		CoarseGeohash: "dr5regw",
		AllowPrecise:  false,
	}
	if err := repo.Insert(scene); err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	exists, err := repo.ExistsByOwnerAndName("did:plc:owner1", "My Scene", "")
	if err != nil {
		t.Fatalf("ExistsByOwnerAndName failed: %v", err)
	}
	if !exists {
		t.Error("Expected scene to exist for owner")
	}
}

func TestSceneRepository_ExistsByOwnerAndName_CaseInsensitive(t *testing.T) {
	repo := NewInMemorySceneRepository()

	scene := &Scene{
		ID:            "scene-eon-ci",
		Name:          "My Scene",
		OwnerDID:      "did:plc:owner1",
		CoarseGeohash: "dr5regw",
		AllowPrecise:  false,
	}
	if err := repo.Insert(scene); err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	exists, err := repo.ExistsByOwnerAndName("did:plc:owner1", "MY SCENE", "")
	if err != nil {
		t.Fatalf("ExistsByOwnerAndName failed: %v", err)
	}
	if !exists {
		t.Error("Expected case-insensitive match")
	}

	exists, err = repo.ExistsByOwnerAndName("did:plc:owner1", "my scene", "")
	if err != nil {
		t.Fatalf("ExistsByOwnerAndName failed: %v", err)
	}
	if !exists {
		t.Error("Expected case-insensitive match (lowercase)")
	}
}

func TestSceneRepository_ExistsByOwnerAndName_DifferentOwner(t *testing.T) {
	repo := NewInMemorySceneRepository()

	scene := &Scene{
		ID:            "scene-eon-do",
		Name:          "Unique Scene",
		OwnerDID:      "did:plc:owner1",
		CoarseGeohash: "dr5regw",
		AllowPrecise:  false,
	}
	if err := repo.Insert(scene); err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	exists, err := repo.ExistsByOwnerAndName("did:plc:owner2", "Unique Scene", "")
	if err != nil {
		t.Fatalf("ExistsByOwnerAndName failed: %v", err)
	}
	if exists {
		t.Error("Expected scene to NOT exist for different owner")
	}
}

func TestSceneRepository_ExistsByOwnerAndName_ExcludeID(t *testing.T) {
	repo := NewInMemorySceneRepository()

	scene := &Scene{
		ID:            "scene-eon-ex",
		Name:          "Exclude Me",
		OwnerDID:      "did:plc:owner1",
		CoarseGeohash: "dr5regw",
		AllowPrecise:  false,
	}
	if err := repo.Insert(scene); err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	// Excluding its own ID should return false (useful for updates)
	exists, err := repo.ExistsByOwnerAndName("did:plc:owner1", "Exclude Me", "scene-eon-ex")
	if err != nil {
		t.Fatalf("ExistsByOwnerAndName failed: %v", err)
	}
	if exists {
		t.Error("Expected false when excludeID matches the scene")
	}
}

func TestSceneRepository_ExistsByOwnerAndName_IgnoresDeleted(t *testing.T) {
	repo := NewInMemorySceneRepository()

	scene := &Scene{
		ID:            "scene-eon-del",
		Name:          "Deleted Scene",
		OwnerDID:      "did:plc:owner1",
		CoarseGeohash: "dr5regw",
		AllowPrecise:  false,
	}
	if err := repo.Insert(scene); err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	// Delete the scene
	if err := repo.Delete("scene-eon-del"); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Should return false because deleted scenes are excluded
	exists, err := repo.ExistsByOwnerAndName("did:plc:owner1", "Deleted Scene", "")
	if err != nil {
		t.Fatalf("ExistsByOwnerAndName failed: %v", err)
	}
	if exists {
		t.Error("Expected false for deleted scene")
	}
}

// --- EventRepository: Cancel ---

func TestEventRepository_Cancel_Success(t *testing.T) {
	repo := NewInMemoryEventRepository()

	event := &Event{
		ID:            "event-cancel-1",
		SceneID:       "scene-1",
		Title:         "Cancel This",
		CoarseGeohash: "dr5regw",
		Status:        "scheduled",
		StartsAt:      time.Now().Add(24 * time.Hour),
	}
	if err := repo.Insert(event); err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	reason := "weather conditions"
	err := repo.Cancel("event-cancel-1", &reason)
	if err != nil {
		t.Fatalf("Cancel failed: %v", err)
	}

	retrieved, err := repo.GetByID("event-cancel-1")
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}

	if retrieved.Status != "cancelled" {
		t.Errorf("Expected status 'cancelled', got %s", retrieved.Status)
	}
	if retrieved.CancelledAt == nil {
		t.Error("Expected CancelledAt to be set")
	}
	if retrieved.CancellationReason == nil || *retrieved.CancellationReason != "weather conditions" {
		t.Errorf("Expected cancellation reason 'weather conditions', got %v", retrieved.CancellationReason)
	}
}

func TestEventRepository_Cancel_NilReason(t *testing.T) {
	repo := NewInMemoryEventRepository()

	event := &Event{
		ID:            "event-cancel-nr",
		SceneID:       "scene-1",
		Title:         "Cancel No Reason",
		CoarseGeohash: "dr5regw",
		Status:        "scheduled",
		StartsAt:      time.Now().Add(24 * time.Hour),
	}
	if err := repo.Insert(event); err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	err := repo.Cancel("event-cancel-nr", nil)
	if err != nil {
		t.Fatalf("Cancel failed: %v", err)
	}

	retrieved, _ := repo.GetByID("event-cancel-nr")
	if retrieved.Status != "cancelled" {
		t.Errorf("Expected status 'cancelled', got %s", retrieved.Status)
	}
	if retrieved.CancellationReason != nil {
		t.Error("Expected nil cancellation reason")
	}
}

func TestEventRepository_Cancel_NotFound(t *testing.T) {
	repo := NewInMemoryEventRepository()

	err := repo.Cancel("nonexistent-event", nil)
	if err != ErrEventNotFound {
		t.Errorf("Expected ErrEventNotFound, got %v", err)
	}
}

func TestEventRepository_Cancel_Idempotent(t *testing.T) {
	repo := NewInMemoryEventRepository()

	event := &Event{
		ID:            "event-cancel-idem",
		SceneID:       "scene-1",
		Title:         "Idempotent Cancel",
		CoarseGeohash: "dr5regw",
		Status:        "scheduled",
		StartsAt:      time.Now().Add(24 * time.Hour),
	}
	if err := repo.Insert(event); err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	reason := "first cancel"
	if err := repo.Cancel("event-cancel-idem", &reason); err != nil {
		t.Fatalf("First cancel failed: %v", err)
	}

	retrieved1, _ := repo.GetByID("event-cancel-idem")
	cancelledAt1 := retrieved1.CancelledAt

	// Second cancel should be idempotent
	reason2 := "second cancel"
	if err := repo.Cancel("event-cancel-idem", &reason2); err != nil {
		t.Fatalf("Second cancel failed: %v", err)
	}

	retrieved2, _ := repo.GetByID("event-cancel-idem")
	if !retrieved2.CancelledAt.Equal(*cancelledAt1) {
		t.Error("Expected CancelledAt to remain unchanged on idempotent call")
	}
}

// --- EventRepository: GetByRecordKey ---

func TestEventRepository_GetByRecordKey_Success(t *testing.T) {
	repo := NewInMemoryEventRepository()

	did := "did:plc:alice123"
	rkey := "event456"
	event := &Event{
		ID:            "event-rk-1",
		SceneID:       "scene-1",
		Title:         "Record Key Event",
		CoarseGeohash: "dr5regw",
		AllowPrecise:  true,
		PrecisePoint:  &Point{Lat: 40.7128, Lng: -74.006},
		Status:        "scheduled",
		StartsAt:      time.Now().Add(24 * time.Hour),
		RecordDID:     &did,
		RecordRKey:    &rkey,
	}

	// Must upsert to populate the keys map
	if _, err := repo.Upsert(event); err != nil {
		t.Fatalf("Upsert failed: %v", err)
	}

	retrieved, err := repo.GetByRecordKey(did, rkey)
	if err != nil {
		t.Fatalf("GetByRecordKey failed: %v", err)
	}

	if retrieved.Title != "Record Key Event" {
		t.Errorf("Expected title 'Record Key Event', got %s", retrieved.Title)
	}
	if retrieved.PrecisePoint == nil {
		t.Error("Expected PrecisePoint to be set (consent is true)")
	}
}

func TestEventRepository_GetByRecordKey_NotFound(t *testing.T) {
	repo := NewInMemoryEventRepository()

	_, err := repo.GetByRecordKey("did:plc:nobody", "no-rkey")
	if err != ErrEventNotFound {
		t.Errorf("Expected ErrEventNotFound, got %v", err)
	}
}

func TestEventRepository_GetByRecordKey_DeepCopy(t *testing.T) {
	repo := NewInMemoryEventRepository()

	did := "did:plc:alice123"
	rkey := "event789"
	event := &Event{
		ID:            "event-rk-dc",
		SceneID:       "scene-1",
		Title:         "Deep Copy Event",
		CoarseGeohash: "dr5regw",
		AllowPrecise:  true,
		PrecisePoint:  &Point{Lat: 40.7128, Lng: -74.006},
		Status:        "scheduled",
		StartsAt:      time.Now().Add(24 * time.Hour),
		RecordDID:     &did,
		RecordRKey:    &rkey,
	}

	if _, err := repo.Upsert(event); err != nil {
		t.Fatalf("Upsert failed: %v", err)
	}

	retrieved, _ := repo.GetByRecordKey(did, rkey)

	// Mutate the returned copy
	retrieved.Title = "Mutated"
	retrieved.PrecisePoint.Lat = 0.0

	// Verify the repo's copy is unchanged
	original, _ := repo.GetByRecordKey(did, rkey)
	if original.Title != "Deep Copy Event" {
		t.Error("Deep copy violated: mutation of returned event affected repository")
	}
	if original.PrecisePoint.Lat != 40.7128 {
		t.Error("Deep copy violated: mutation of PrecisePoint affected repository")
	}
}

// --- SceneRepository: Stripe Onboarding Status ---

func TestSceneRepository_StripeOnboardingStatus_Update(t *testing.T) {
	repo := NewInMemorySceneRepository()

	acctID := strconv.FormatInt(1234567890, 10)
	scene := &Scene{
		ID:                      "scene-stripe-1",
		Name:                    "Stripe Scene",
		OwnerDID:                "did:plc:owner1",
		CoarseGeohash:           "dr5regw",
		AllowPrecise:            false,
		ConnectedAccountID:      &acctID,
		ConnectedAccountStatus:  "pending",
	}

	if err := repo.Insert(scene); err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	// Update status to active
	scene.ConnectedAccountStatus = "active"
	now := time.Now()
	scene.AccountOnboardedAt = &now

	if err := repo.Update(scene); err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	retrieved, err := repo.GetByID("scene-stripe-1")
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}

	if retrieved.ConnectedAccountStatus != "active" {
		t.Errorf("Expected status 'active', got '%s'", retrieved.ConnectedAccountStatus)
	}
	if retrieved.AccountOnboardedAt == nil {
		t.Error("Expected AccountOnboardedAt to be set")
	}
}

func TestSceneRepository_StripeOnboardingStatus_DefaultValues(t *testing.T) {
	repo := NewInMemorySceneRepository()

	scene := &Scene{
		ID:            "scene-stripe-default",
		Name:          "New Scene",
		OwnerDID:      "did:plc:owner1",
		CoarseGeohash: "dr5regw",
		AllowPrecise:  false,
	}

	if err := repo.Insert(scene); err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	retrieved, _ := repo.GetByID("scene-stripe-default")

	// Default status should be empty string (as struct zero value)
	// The database will enforce DEFAULT 'pending'
	if retrieved.ConnectedAccountStatus != "" && retrieved.ConnectedAccountStatus != "pending" {
		t.Errorf("Unexpected ConnectedAccountStatus: '%s'", retrieved.ConnectedAccountStatus)
	}
	if retrieved.AccountOnboardedAt != nil {
		t.Error("Expected AccountOnboardedAt to be nil for new scene")
	}
}

// --- SceneRepository: Moderation Status ---

func TestSceneRepository_ModerationStatus_HiddenScene(t *testing.T) {
	repo := NewInMemorySceneRepository()

	scene := &Scene{
		ID:                  "scene-mod-1",
		Name:                "Moderated Scene",
		OwnerDID:            "did:plc:owner1",
		CoarseGeohash:       "dr5regw",
		AllowPrecise:        false,
		ModerationStatus:    "hidden",
		ModerationReason:    strPtr("Spam content"),
		ModeratedBy:         strPtr("did:plc:admin1"),
		ModerationTimestamp: &[]time.Time{time.Now()}[0],
	}

	if err := repo.Insert(scene); err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	retrieved, err := repo.GetByID("scene-mod-1")
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}

	if retrieved.ModerationStatus != "hidden" {
		t.Errorf("Expected status 'hidden', got '%s'", retrieved.ModerationStatus)
	}
	if retrieved.ModerationReason == nil || *retrieved.ModerationReason != "Spam content" {
		t.Errorf("Expected reason 'Spam content', got %v", retrieved.ModerationReason)
	}
	if retrieved.ModeratedBy == nil || *retrieved.ModeratedBy != "did:plc:admin1" {
		t.Errorf("Expected moderated_by 'did:plc:admin1', got %v", retrieved.ModeratedBy)
	}
	if retrieved.ModerationTimestamp == nil {
		t.Error("Expected ModerationTimestamp to be set")
	}
}

func TestSceneRepository_ModerationStatus_RemoveModeration(t *testing.T) {
	repo := NewInMemorySceneRepository()

	scene := &Scene{
		ID:                  "scene-mod-2",
		Name:                "Moderated Scene",
		OwnerDID:            "did:plc:owner1",
		CoarseGeohash:       "dr5regw",
		AllowPrecise:        false,
		ModerationStatus:    "suspended",
		ModerationReason:    strPtr("Severe violation"),
		ModeratedBy:         strPtr("did:plc:admin1"),
		ModerationTimestamp: &[]time.Time{time.Now()}[0],
	}

	if err := repo.Insert(scene); err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	// Remove moderation: revert to visible status
	scene.ModerationStatus = "visible"
	scene.ModerationReason = nil
	scene.ModeratedBy = nil
	scene.ModerationTimestamp = nil

	if err := repo.Update(scene); err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	retrieved, _ := repo.GetByID("scene-mod-2")

	if retrieved.ModerationStatus != "visible" {
		t.Errorf("Expected status 'visible', got '%s'", retrieved.ModerationStatus)
	}
	if retrieved.ModeratedBy != nil {
		t.Errorf("Expected moderated_by to be nil, got %v", retrieved.ModeratedBy)
	}
	if retrieved.ModerationTimestamp != nil {
		t.Error("Expected ModerationTimestamp to be nil after removal")
	}
}
