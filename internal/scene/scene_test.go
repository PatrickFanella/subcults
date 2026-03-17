package scene

import (
	"testing"
)

func TestScene_EnforceLocationConsent(t *testing.T) {
	tests := []struct {
		name                string
		allowPrecise        bool
		hasPrecisePoint     bool
		wantPrecisePointNil bool
	}{
		{
			name:                "consent false with precise point - should clear",
			allowPrecise:        false,
			hasPrecisePoint:     true,
			wantPrecisePointNil: true,
		},
		{
			name:                "consent false without precise point - remains nil",
			allowPrecise:        false,
			hasPrecisePoint:     false,
			wantPrecisePointNil: true,
		},
		{
			name:                "consent true with precise point - should keep",
			allowPrecise:        true,
			hasPrecisePoint:     true,
			wantPrecisePointNil: false,
		},
		{
			name:                "consent true without precise point - remains nil",
			allowPrecise:        true,
			hasPrecisePoint:     false,
			wantPrecisePointNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scene := &Scene{
				ID:           "scene-1",
				Name:         "Test Scene",
				AllowPrecise: tt.allowPrecise,
			}
			if tt.hasPrecisePoint {
				scene.PrecisePoint = &Point{Lat: 40.7128, Lng: -74.0060}
			}

			scene.EnforceLocationConsent()

			if (scene.PrecisePoint == nil) != tt.wantPrecisePointNil {
				t.Errorf("EnforceLocationConsent() PrecisePoint = %v, wantNil = %v", scene.PrecisePoint, tt.wantPrecisePointNil)
			}
		})
	}
}

func TestEvent_EnforceLocationConsent(t *testing.T) {
	tests := []struct {
		name                string
		allowPrecise        bool
		hasPrecisePoint     bool
		wantPrecisePointNil bool
	}{
		{
			name:                "consent false with precise point - should clear",
			allowPrecise:        false,
			hasPrecisePoint:     true,
			wantPrecisePointNil: true,
		},
		{
			name:                "consent false without precise point - remains nil",
			allowPrecise:        false,
			hasPrecisePoint:     false,
			wantPrecisePointNil: true,
		},
		{
			name:                "consent true with precise point - should keep",
			allowPrecise:        true,
			hasPrecisePoint:     true,
			wantPrecisePointNil: false,
		},
		{
			name:                "consent true without precise point - remains nil",
			allowPrecise:        true,
			hasPrecisePoint:     false,
			wantPrecisePointNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := &Event{
				ID:           "event-1",
				SceneID:      "scene-1",
				Title:        "Test Event",
				AllowPrecise: tt.allowPrecise,
			}
			if tt.hasPrecisePoint {
				event.PrecisePoint = &Point{Lat: 40.7128, Lng: -74.0060}
			}

			event.EnforceLocationConsent()

			if (event.PrecisePoint == nil) != tt.wantPrecisePointNil {
				t.Errorf("EnforceLocationConsent() PrecisePoint = %v, wantNil = %v", event.PrecisePoint, tt.wantPrecisePointNil)
			}
		})
	}
}

func TestInMemorySceneRepository_Insert_WithoutConsent(t *testing.T) {
	repo := NewInMemorySceneRepository()

	// Insert a scene with precise point but without consent
	scene := &Scene{
		ID:           "scene-1",
		Name:         "Test Scene",
		AllowPrecise: false, // No consent
		PrecisePoint: &Point{Lat: 40.7128, Lng: -74.0060},
	}

	err := repo.Insert(scene)
	if err != nil {
		t.Fatalf("Insert() error = %v", err)
	}

	// Verify the original scene was not modified
	if scene.PrecisePoint == nil {
		t.Error("Insert() should not modify original scene's PrecisePoint")
	}

	// Retrieve and verify stored scene has nil precise point
	stored, err := repo.GetByID("scene-1")
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if stored == nil {
		t.Fatal("GetByID() returned nil scene")
	}
	if stored.PrecisePoint != nil {
		t.Error("Insert() with AllowPrecise=false should set PrecisePoint to nil in storage")
	}
}

func TestInMemorySceneRepository_Insert_WithConsent(t *testing.T) {
	repo := NewInMemorySceneRepository()

	// Insert a scene with precise point and consent
	scene := &Scene{
		ID:           "scene-2",
		Name:         "Test Scene With Consent",
		AllowPrecise: true, // Has consent
		PrecisePoint: &Point{Lat: 40.7128, Lng: -74.0060},
	}

	err := repo.Insert(scene)
	if err != nil {
		t.Fatalf("Insert() error = %v", err)
	}

	// Retrieve and verify stored scene has precise point
	stored, err := repo.GetByID("scene-2")
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if stored == nil {
		t.Fatal("GetByID() returned nil scene")
	}
	if stored.PrecisePoint == nil {
		t.Error("Insert() with AllowPrecise=true should preserve PrecisePoint")
	}
	if stored.PrecisePoint != nil && (stored.PrecisePoint.Lat != 40.7128 || stored.PrecisePoint.Lng != -74.0060) {
		t.Errorf("Insert() PrecisePoint = %+v, want Lat=40.7128, Lng=-74.0060", stored.PrecisePoint)
	}
}

func TestInMemorySceneRepository_Update_WithoutConsent(t *testing.T) {
	repo := NewInMemorySceneRepository()

	// First insert with consent
	scene := &Scene{
		ID:           "scene-3",
		Name:         "Test Scene",
		AllowPrecise: true,
		PrecisePoint: &Point{Lat: 40.7128, Lng: -74.0060},
	}
	_ = repo.Insert(scene)

	// Now update removing consent but keeping precise point in input
	updatedScene := &Scene{
		ID:           "scene-3",
		Name:         "Updated Scene",
		AllowPrecise: false,                               // Consent removed
		PrecisePoint: &Point{Lat: 40.7128, Lng: -74.0060}, // Still has point in input
	}

	err := repo.Update(updatedScene)
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	// Retrieve and verify stored scene has nil precise point
	stored, err := repo.GetByID("scene-3")
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if stored == nil {
		t.Fatal("GetByID() returned nil scene")
	}
	if stored.PrecisePoint != nil {
		t.Error("Update() with AllowPrecise=false should set PrecisePoint to nil in storage")
	}
}

func TestInMemoryEventRepository_Insert_WithoutConsent(t *testing.T) {
	repo := NewInMemoryEventRepository()

	// Insert an event with precise point but without consent
	event := &Event{
		ID:           "event-1",
		SceneID:      "scene-1",
		Title:        "Test Event",
		AllowPrecise: false, // No consent
		PrecisePoint: &Point{Lat: 40.7128, Lng: -74.0060},
	}

	err := repo.Insert(event)
	if err != nil {
		t.Fatalf("Insert() error = %v", err)
	}

	// Verify the original event was not modified
	if event.PrecisePoint == nil {
		t.Error("Insert() should not modify original event's PrecisePoint")
	}

	// Retrieve and verify stored event has nil precise point
	stored, err := repo.GetByID("event-1")
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if stored == nil {
		t.Fatal("GetByID() returned nil event")
	}
	if stored.PrecisePoint != nil {
		t.Error("Insert() with AllowPrecise=false should set PrecisePoint to nil in storage")
	}
}

func TestInMemoryEventRepository_Insert_WithConsent(t *testing.T) {
	repo := NewInMemoryEventRepository()

	// Insert an event with precise point and consent
	event := &Event{
		ID:           "event-2",
		SceneID:      "scene-1",
		Title:        "Test Event With Consent",
		AllowPrecise: true, // Has consent
		PrecisePoint: &Point{Lat: 40.7128, Lng: -74.0060},
	}

	err := repo.Insert(event)
	if err != nil {
		t.Fatalf("Insert() error = %v", err)
	}

	// Retrieve and verify stored event has precise point
	stored, err := repo.GetByID("event-2")
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if stored == nil {
		t.Fatal("GetByID() returned nil event")
	}
	if stored.PrecisePoint == nil {
		t.Error("Insert() with AllowPrecise=true should preserve PrecisePoint")
	}
	if stored.PrecisePoint != nil && (stored.PrecisePoint.Lat != 40.7128 || stored.PrecisePoint.Lng != -74.0060) {
		t.Errorf("Insert() PrecisePoint = %+v, want Lat=40.7128, Lng=-74.0060", stored.PrecisePoint)
	}
}

func TestInMemorySceneRepository_GetByIDs(t *testing.T) {
	repo := NewInMemorySceneRepository()
	scene1 := &Scene{ID: "scene-1", Name: "One", OwnerDID: "did:one"}
	scene2 := &Scene{ID: "scene-2", Name: "Two", OwnerDID: "did:two"}
	sceneDeleted := &Scene{ID: "scene-deleted", Name: "Deleted", OwnerDID: "did:deleted"}
	if err := repo.Insert(scene1); err != nil {
		t.Fatalf("failed to insert scene1: %v", err)
	}
	if err := repo.Insert(scene2); err != nil {
		t.Fatalf("failed to insert scene2: %v", err)
	}
	if err := repo.Insert(sceneDeleted); err != nil {
		t.Fatalf("failed to insert sceneDeleted: %v", err)
	}
	if err := repo.Delete(sceneDeleted.ID); err != nil {
		t.Fatalf("failed to delete sceneDeleted: %v", err)
	}

	results, err := repo.GetByIDs([]string{scene1.ID, "missing", sceneDeleted.ID, scene2.ID})
	if err != nil {
		t.Fatalf("GetByIDs returned error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 scenes (excluding missing/deleted), got %d", len(results))
	}
	found := map[string]bool{}
	for _, result := range results {
		found[result.ID] = true
	}
	if !found[scene1.ID] || !found[scene2.ID] {
		t.Fatalf("expected scene-1 and scene-2 in results, got %+v", found)
	}

	emptyResults, err := repo.GetByIDs(nil)
	if err != nil {
		t.Fatalf("GetByIDs with nil ids returned error: %v", err)
	}
	if len(emptyResults) != 0 {
		t.Fatalf("expected empty result for nil ids, got %d", len(emptyResults))
	}
}

func TestInMemoryEventRepository_Update_WithoutConsent(t *testing.T) {
	repo := NewInMemoryEventRepository()

	// First insert with consent
	event := &Event{
		ID:           "event-3",
		SceneID:      "scene-1",
		Title:        "Test Event",
		AllowPrecise: true,
		PrecisePoint: &Point{Lat: 40.7128, Lng: -74.0060},
	}
	_ = repo.Insert(event)

	// Now update removing consent but keeping precise point in input
	updatedEvent := &Event{
		ID:           "event-3",
		SceneID:      "scene-1",
		Title:        "Updated Event",
		AllowPrecise: false,                               // Consent removed
		PrecisePoint: &Point{Lat: 40.7128, Lng: -74.0060}, // Still has point in input
	}

	err := repo.Update(updatedEvent)
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	// Retrieve and verify stored event has nil precise point
	stored, err := repo.GetByID("event-3")
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if stored == nil {
		t.Fatal("GetByID() returned nil event")
	}
	if stored.PrecisePoint != nil {
		t.Error("Update() with AllowPrecise=false should set PrecisePoint to nil in storage")
	}
}

func TestInMemorySceneRepository_GetByID_ReturnsNilForNonexistent(t *testing.T) {
	repo := NewInMemorySceneRepository()

	stored, err := repo.GetByID("nonexistent")
	if err != ErrSceneNotFound {
		t.Fatalf("GetByID() error = %v, want ErrSceneNotFound", err)
	}
	if stored != nil {
		t.Error("GetByID() should return nil for nonexistent scene")
	}
}

func TestInMemoryEventRepository_GetByID_ReturnsNilForNonexistent(t *testing.T) {
	repo := NewInMemoryEventRepository()

	stored, err := repo.GetByID("nonexistent")
	if err != ErrEventNotFound {
		t.Fatalf("GetByID() error = %v, want ErrEventNotFound", err)
	}
	if stored != nil {
		t.Error("GetByID() should return nil for nonexistent event")
	}
}

func TestInMemorySceneRepository_Insert_DeepCopyProtection(t *testing.T) {
	repo := NewInMemorySceneRepository()

	// Insert a scene with precise point and consent
	originalPoint := &Point{Lat: 40.7128, Lng: -74.0060}
	scene := &Scene{
		ID:           "scene-deep-copy",
		Name:         "Test Scene",
		AllowPrecise: true,
		PrecisePoint: originalPoint,
	}

	err := repo.Insert(scene)
	if err != nil {
		t.Fatalf("Insert() error = %v", err)
	}

	// Modify the original point after insertion
	originalPoint.Lat = 0.0
	originalPoint.Lng = 0.0

	// Retrieve and verify stored scene was NOT affected by the modification
	stored, err := repo.GetByID("scene-deep-copy")
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if stored == nil {
		t.Fatal("GetByID() returned nil scene")
	}
	if stored.PrecisePoint == nil {
		t.Fatal("stored scene PrecisePoint should not be nil")
	}
	if stored.PrecisePoint.Lat != 40.7128 || stored.PrecisePoint.Lng != -74.0060 {
		t.Errorf("Insert() should create deep copy; stored PrecisePoint = %+v, want Lat=40.7128, Lng=-74.0060", stored.PrecisePoint)
	}
}

func TestInMemoryEventRepository_Insert_DeepCopyProtection(t *testing.T) {
	repo := NewInMemoryEventRepository()

	// Insert an event with precise point and consent
	originalPoint := &Point{Lat: 40.7128, Lng: -74.0060}
	event := &Event{
		ID:           "event-deep-copy",
		SceneID:      "scene-1",
		Title:        "Test Event",
		AllowPrecise: true,
		PrecisePoint: originalPoint,
	}

	err := repo.Insert(event)
	if err != nil {
		t.Fatalf("Insert() error = %v", err)
	}

	// Modify the original point after insertion
	originalPoint.Lat = 0.0
	originalPoint.Lng = 0.0

	// Retrieve and verify stored event was NOT affected by the modification
	stored, err := repo.GetByID("event-deep-copy")
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if stored == nil {
		t.Fatal("GetByID() returned nil event")
	}
	if stored.PrecisePoint == nil {
		t.Fatal("stored event PrecisePoint should not be nil")
	}
	if stored.PrecisePoint.Lat != 40.7128 || stored.PrecisePoint.Lng != -74.0060 {
		t.Errorf("Insert() should create deep copy; stored PrecisePoint = %+v, want Lat=40.7128, Lng=-74.0060", stored.PrecisePoint)
	}
}

func strPtr(s string) *string {
	return &s
}

func TestSceneRepository_Upsert_Insert(t *testing.T) {
	repo := NewInMemorySceneRepository()
	did := "did:example:alice"
	rkey := "scene123"

	scene := &Scene{
		Name:         "Test Scene",
		Description:  "A test scene",
		AllowPrecise: false,
		RecordDID:    strPtr(did),
		RecordRKey:   strPtr(rkey),
	}

	result, err := repo.Upsert(scene)
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

	if retrieved.Name != "Test Scene" {
		t.Errorf("Expected name 'Test Scene', got %s", retrieved.Name)
	}
}

func TestSceneRepository_Upsert_Update(t *testing.T) {
	repo := NewInMemorySceneRepository()
	did := "did:example:alice"
	rkey := "scene123"

	// First insert
	scene := &Scene{
		Name:         "Original Name",
		Description:  "Original description",
		AllowPrecise: false,
		RecordDID:    strPtr(did),
		RecordRKey:   strPtr(rkey),
	}

	result1, err := repo.Upsert(scene)
	if err != nil {
		t.Fatalf("First upsert failed: %v", err)
	}

	if !result1.Inserted {
		t.Error("Expected insert on first upsert")
	}

	// Second upsert with same record key
	scene2 := &Scene{
		Name:         "Updated Name",
		Description:  "Updated description",
		AllowPrecise: false,
		RecordDID:    strPtr(did),
		RecordRKey:   strPtr(rkey),
	}

	result2, err := repo.Upsert(scene2)
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

	if retrieved.Name != "Updated Name" {
		t.Errorf("Expected updated name, got %s", retrieved.Name)
	}
}

func TestSceneRepository_Upsert_EnforcesLocationConsent(t *testing.T) {
	repo := NewInMemorySceneRepository()
	did := "did:example:alice"
	rkey := "scene123"

	// Insert with precise point but consent=false
	scene := &Scene{
		Name:         "Test Scene",
		AllowPrecise: false,
		PrecisePoint: &Point{Lat: 40.7128, Lng: -74.0060},
		RecordDID:    strPtr(did),
		RecordRKey:   strPtr(rkey),
	}

	result, err := repo.Upsert(scene)
	if err != nil {
		t.Fatalf("Upsert failed: %v", err)
	}

	// Retrieve and verify consent was enforced
	retrieved, err := repo.GetByRecordKey(did, rkey)
	if err != nil {
		t.Fatalf("GetByRecordKey failed: %v", err)
	}

	if retrieved.PrecisePoint != nil {
		t.Error("Expected PrecisePoint to be nil when consent is false")
	}

	// Update with consent=true
	scene2 := &Scene{
		Name:         "Test Scene",
		AllowPrecise: true,
		PrecisePoint: &Point{Lat: 40.7128, Lng: -74.0060},
		RecordDID:    strPtr(did),
		RecordRKey:   strPtr(rkey),
	}

	result2, err := repo.Upsert(scene2)
	if err != nil {
		t.Fatalf("Second upsert failed: %v", err)
	}

	if result2.Inserted {
		t.Error("Expected update, got insert")
	}

	if result.ID != result2.ID {
		t.Error("Expected same ID for update")
	}

	// Verify precise point is now preserved
	retrieved2, err := repo.GetByRecordKey(did, rkey)
	if err != nil {
		t.Fatalf("GetByRecordKey failed: %v", err)
	}

	if retrieved2.PrecisePoint == nil {
		t.Error("Expected PrecisePoint to be preserved when consent is true")
	}
}

func TestSceneRepository_Upsert_Idempotent(t *testing.T) {
	repo := NewInMemorySceneRepository()
	did := "did:example:alice"
	rkey := "scene123"

	scene := &Scene{
		Name:         "Same Scene",
		Description:  "Same description",
		AllowPrecise: false,
		RecordDID:    strPtr(did),
		RecordRKey:   strPtr(rkey),
	}

	// First upsert
	result1, err := repo.Upsert(scene)
	if err != nil {
		t.Fatalf("First upsert failed: %v", err)
	}

	// Second upsert with same content
	result2, err := repo.Upsert(scene)
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

func TestEventRepository_Upsert_Insert(t *testing.T) {
	repo := NewInMemoryEventRepository()
	did := "did:example:alice"
	rkey := "event123"

	event := &Event{
		SceneID:      "scene-1",
		Title:        "Test Event",
		Description:  "A test event",
		AllowPrecise: false,
		RecordDID:    strPtr(did),
		RecordRKey:   strPtr(rkey),
	}

	result, err := repo.Upsert(event)
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

	if retrieved.Title != "Test Event" {
		t.Errorf("Expected name 'Test Event', got %s", retrieved.Title)
	}
}

func TestEventRepository_Upsert_Update(t *testing.T) {
	repo := NewInMemoryEventRepository()
	did := "did:example:alice"
	rkey := "event123"

	// First insert
	event := &Event{
		SceneID:      "scene-1",
		Title:        "Original Event",
		Description:  "Original description",
		AllowPrecise: false,
		RecordDID:    strPtr(did),
		RecordRKey:   strPtr(rkey),
	}

	result1, err := repo.Upsert(event)
	if err != nil {
		t.Fatalf("First upsert failed: %v", err)
	}

	if !result1.Inserted {
		t.Error("Expected insert on first upsert")
	}

	// Second upsert with same record key
	event2 := &Event{
		SceneID:      "scene-2",
		Title:        "Updated Event",
		Description:  "Updated description",
		AllowPrecise: false,
		RecordDID:    strPtr(did),
		RecordRKey:   strPtr(rkey),
	}

	result2, err := repo.Upsert(event2)
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

	if retrieved.Title != "Updated Event" {
		t.Errorf("Expected updated name, got %s", retrieved.Title)
	}

	if retrieved.SceneID != "scene-2" {
		t.Errorf("Expected SceneID 'scene-2', got %s", retrieved.SceneID)
	}
}

func TestEventRepository_Upsert_EnforcesLocationConsent(t *testing.T) {
	repo := NewInMemoryEventRepository()
	did := "did:example:alice"
	rkey := "event123"

	// Insert with precise point but consent=false
	event := &Event{
		SceneID:      "scene-1",
		Title:        "Test Event",
		AllowPrecise: false,
		PrecisePoint: &Point{Lat: 40.7128, Lng: -74.0060},
		RecordDID:    strPtr(did),
		RecordRKey:   strPtr(rkey),
	}

	result, err := repo.Upsert(event)
	if err != nil {
		t.Fatalf("Upsert failed: %v", err)
	}

	// Retrieve and verify consent was enforced
	retrieved, err := repo.GetByRecordKey(did, rkey)
	if err != nil {
		t.Fatalf("GetByRecordKey failed: %v", err)
	}

	if retrieved.PrecisePoint != nil {
		t.Error("Expected PrecisePoint to be nil when consent is false")
	}

	// Update with consent=true
	event2 := &Event{
		SceneID:      "scene-1",
		Title:        "Test Event",
		AllowPrecise: true,
		PrecisePoint: &Point{Lat: 40.7128, Lng: -74.0060},
		RecordDID:    strPtr(did),
		RecordRKey:   strPtr(rkey),
	}

	result2, err := repo.Upsert(event2)
	if err != nil {
		t.Fatalf("Second upsert failed: %v", err)
	}

	if result2.Inserted {
		t.Error("Expected update, got insert")
	}

	if result.ID != result2.ID {
		t.Error("Expected same ID for update")
	}

	// Verify precise point is now preserved
	retrieved2, err := repo.GetByRecordKey(did, rkey)
	if err != nil {
		t.Fatalf("GetByRecordKey failed: %v", err)
	}

	if retrieved2.PrecisePoint == nil {
		t.Error("Expected PrecisePoint to be preserved when consent is true")
	}
}

func TestSceneRepository_ListByOwner(t *testing.T) {
	repo := NewInMemorySceneRepository()

	// Create scenes for different owners
	owner1 := "did:plc:owner1"
	owner2 := "did:plc:owner2"

	scene1 := &Scene{
		ID:            "scene-1",
		Name:          "Scene 1",
		OwnerDID:      owner1,
		CoarseGeohash: "dr5regw",
		AllowPrecise:  false,
	}
	scene2 := &Scene{
		ID:            "scene-2",
		Name:          "Scene 2",
		OwnerDID:      owner1,
		CoarseGeohash: "dr5regw",
		AllowPrecise:  false,
	}
	scene3 := &Scene{
		ID:            "scene-3",
		Name:          "Scene 3",
		OwnerDID:      owner2,
		CoarseGeohash: "dr5regw",
		AllowPrecise:  false,
	}

	// Insert scenes
	if err := repo.Insert(scene1); err != nil {
		t.Fatalf("Insert scene1 failed: %v", err)
	}
	if err := repo.Insert(scene2); err != nil {
		t.Fatalf("Insert scene2 failed: %v", err)
	}
	if err := repo.Insert(scene3); err != nil {
		t.Fatalf("Insert scene3 failed: %v", err)
	}

	// Test: List scenes for owner1
	scenes, err := repo.ListByOwner(owner1)
	if err != nil {
		t.Fatalf("ListByOwner failed: %v", err)
	}
	if len(scenes) != 2 {
		t.Errorf("Expected 2 scenes for owner1, got %d", len(scenes))
	}

	// Test: List scenes for owner2
	scenes, err = repo.ListByOwner(owner2)
	if err != nil {
		t.Fatalf("ListByOwner failed: %v", err)
	}
	if len(scenes) != 1 {
		t.Errorf("Expected 1 scene for owner2, got %d", len(scenes))
	}

	// Test: List scenes for owner with no scenes
	scenes, err = repo.ListByOwner("did:plc:noowner")
	if err != nil {
		t.Fatalf("ListByOwner failed: %v", err)
	}
	if len(scenes) != 0 {
		t.Errorf("Expected 0 scenes for unknown owner, got %d", len(scenes))
	}
}

func TestSceneRepository_ListByOwner_ExcludesDeleted(t *testing.T) {
	repo := NewInMemorySceneRepository()

	owner := "did:plc:owner1"

	scene1 := &Scene{
		ID:            "scene-1",
		Name:          "Scene 1",
		OwnerDID:      owner,
		CoarseGeohash: "dr5regw",
		AllowPrecise:  false,
	}
	scene2 := &Scene{
		ID:            "scene-2",
		Name:          "Scene 2",
		OwnerDID:      owner,
		CoarseGeohash: "dr5regw",
		AllowPrecise:  false,
	}

	// Insert scenes
	if err := repo.Insert(scene1); err != nil {
		t.Fatalf("Insert scene1 failed: %v", err)
	}
	if err := repo.Insert(scene2); err != nil {
		t.Fatalf("Insert scene2 failed: %v", err)
	}

	// Delete scene1
	if err := repo.Delete("scene-1"); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// List scenes - should only return non-deleted scene
	scenes, err := repo.ListByOwner(owner)
	if err != nil {
		t.Fatalf("ListByOwner failed: %v", err)
	}
	if len(scenes) != 1 {
		t.Errorf("Expected 1 non-deleted scene, got %d", len(scenes))
	}
	if scenes[0].ID != "scene-2" {
		t.Errorf("Expected scene-2, got %s", scenes[0].ID)
	}
}

// TestRSVPRepository_GetCountsForEvents_BatchQuery tests batch RSVP counting.
func TestRSVPRepository_GetCountsForEvents_BatchQuery(t *testing.T) {
	repo := NewInMemoryRSVPRepository()

	// Create RSVPs for multiple events
	event1ID := "event-1"
	event2ID := "event-2"
	event3ID := "event-3"

	// Event 1: 3 going, 2 maybe
	rsvps1 := []*RSVP{
		{EventID: event1ID, UserID: "user1", Status: "going"},
		{EventID: event1ID, UserID: "user2", Status: "going"},
		{EventID: event1ID, UserID: "user3", Status: "going"},
		{EventID: event1ID, UserID: "user4", Status: "maybe"},
		{EventID: event1ID, UserID: "user5", Status: "maybe"},
	}

	// Event 2: 1 going, 0 maybe
	rsvps2 := []*RSVP{
		{EventID: event2ID, UserID: "user6", Status: "going"},
	}

	// Event 3: no RSVPs

	// Insert all RSVPs
	for _, rsvp := range append(rsvps1, rsvps2...) {
		if err := repo.Upsert(rsvp); err != nil {
			t.Fatalf("Upsert failed: %v", err)
		}
	}

	// Batch query
	eventIDs := []string{event1ID, event2ID, event3ID}
	countsMap, err := repo.GetCountsForEvents(eventIDs)
	if err != nil {
		t.Fatalf("GetCountsForEvents failed: %v", err)
	}

	// Verify event 1 counts
	if counts, ok := countsMap[event1ID]; !ok {
		t.Error("Expected event 1 to have counts")
	} else {
		if counts.Going != 3 {
			t.Errorf("Event 1: expected 3 going, got %d", counts.Going)
		}
		if counts.Maybe != 2 {
			t.Errorf("Event 1: expected 2 maybe, got %d", counts.Maybe)
		}
	}

	// Verify event 2 counts
	if counts, ok := countsMap[event2ID]; !ok {
		t.Error("Expected event 2 to have counts")
	} else {
		if counts.Going != 1 {
			t.Errorf("Event 2: expected 1 going, got %d", counts.Going)
		}
		if counts.Maybe != 0 {
			t.Errorf("Event 2: expected 0 maybe, got %d", counts.Maybe)
		}
	}

	// Verify event 3 counts (no RSVPs)
	if counts, ok := countsMap[event3ID]; !ok {
		t.Error("Expected event 3 to have counts")
	} else {
		if counts.Going != 0 {
			t.Errorf("Event 3: expected 0 going, got %d", counts.Going)
		}
		if counts.Maybe != 0 {
			t.Errorf("Event 3: expected 0 maybe, got %d", counts.Maybe)
		}
	}
}

// TestRSVPRepository_GetCountsForEvents_EmptyInput tests empty input handling.
func TestRSVPRepository_GetCountsForEvents_EmptyInput(t *testing.T) {
	repo := NewInMemoryRSVPRepository()

	countsMap, err := repo.GetCountsForEvents([]string{})
	if err != nil {
		t.Fatalf("GetCountsForEvents failed: %v", err)
	}

	if len(countsMap) != 0 {
		t.Errorf("Expected empty map for empty input, got %d entries", len(countsMap))
	}
}
