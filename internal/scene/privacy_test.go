package scene

import (
	"testing"

	"github.com/onnwee/subcults/internal/geo"
)

// TestPrivacy_SceneFetch_WithoutConsent validates that scenes without consent
// never expose precise coordinates, only coarse geohash if available.
func TestPrivacy_SceneFetch_WithoutConsent(t *testing.T) {
	repo := NewInMemorySceneRepository()

	// Create a scene with precise coordinates but no consent
	scene := &Scene{
		ID:           "scene-no-consent",
		Name:         "Private Scene",
		Description:  "Should not expose precise location",
		AllowPrecise: false,
		PrecisePoint: &Point{Lat: 37.7749, Lng: -122.4194}, // San Francisco
	}

	err := repo.Insert(scene)
	if err != nil {
		t.Fatalf("Insert() error = %v", err)
	}

	// Fetch the scene - simulating a public API fetch
	fetched, err := repo.GetByID("scene-no-consent")
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	// Critical privacy assertion: PrecisePoint must be nil
	if fetched.PrecisePoint != nil {
		t.Errorf("Privacy violation: scene without consent exposed precise coordinates: %+v", fetched.PrecisePoint)
	}

	// Verify consent flag is preserved
	if fetched.AllowPrecise {
		t.Error("AllowPrecise should be false")
	}
}

// TestPrivacy_SceneFetch_WithConsent validates that scenes with consent
// do expose precise coordinates when requested.
func TestPrivacy_SceneFetch_WithConsent(t *testing.T) {
	repo := NewInMemorySceneRepository()

	expectedLat := 37.7749
	expectedLng := -122.4194

	// Create a scene with precise coordinates and explicit consent
	scene := &Scene{
		ID:           "scene-with-consent",
		Name:         "Public Scene",
		Description:  "Consented to precise location",
		AllowPrecise: true,
		PrecisePoint: &Point{Lat: expectedLat, Lng: expectedLng},
	}

	err := repo.Insert(scene)
	if err != nil {
		t.Fatalf("Insert() error = %v", err)
	}

	// Fetch the scene
	fetched, err := repo.GetByID("scene-with-consent")
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	// With consent, precise point should be available
	if fetched.PrecisePoint == nil {
		t.Fatal("With consent, PrecisePoint should not be nil")
	}

	// Verify coordinates are preserved
	if fetched.PrecisePoint.Lat != expectedLat || fetched.PrecisePoint.Lng != expectedLng {
		t.Errorf("Coordinates mismatch: got (%f, %f), want (%f, %f)",
			fetched.PrecisePoint.Lat, fetched.PrecisePoint.Lng, expectedLat, expectedLng)
	}

	// Verify consent flag is preserved
	if !fetched.AllowPrecise {
		t.Error("AllowPrecise should be true")
	}
}

// TestPrivacy_EventFetch_WithoutConsent validates that events without consent
// never expose precise coordinates.
func TestPrivacy_EventFetch_WithoutConsent(t *testing.T) {
	repo := NewInMemoryEventRepository()

	// Create an event with precise coordinates but no consent
	event := &Event{
		ID:           "event-no-consent",
		SceneID:      "scene-1",
		Title:        "Private Event",
		Description:  "Should not expose precise location",
		AllowPrecise: false,
		PrecisePoint: &Point{Lat: 40.7128, Lng: -74.0060}, // New York
	}

	err := repo.Insert(event)
	if err != nil {
		t.Fatalf("Insert() error = %v", err)
	}

	// Fetch the event - simulating a public API fetch
	fetched, err := repo.GetByID("event-no-consent")
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	// Critical privacy assertion: PrecisePoint must be nil
	if fetched.PrecisePoint != nil {
		t.Errorf("Privacy violation: event without consent exposed precise coordinates: %+v", fetched.PrecisePoint)
	}

	// Verify consent flag is preserved
	if fetched.AllowPrecise {
		t.Error("AllowPrecise should be false")
	}
}

// TestPrivacy_EventFetch_WithConsent validates that events with consent
// do expose precise coordinates when requested.
func TestPrivacy_EventFetch_WithConsent(t *testing.T) {
	repo := NewInMemoryEventRepository()

	expectedLat := 40.7128
	expectedLng := -74.0060

	// Create an event with precise coordinates and explicit consent
	event := &Event{
		ID:           "event-with-consent",
		SceneID:      "scene-1",
		Title:        "Public Event",
		Description:  "Consented to precise location",
		AllowPrecise: true,
		PrecisePoint: &Point{Lat: expectedLat, Lng: expectedLng},
	}

	err := repo.Insert(event)
	if err != nil {
		t.Fatalf("Insert() error = %v", err)
	}

	// Fetch the event
	fetched, err := repo.GetByID("event-with-consent")
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	// With consent, precise point should be available
	if fetched.PrecisePoint == nil {
		t.Fatal("With consent, PrecisePoint should not be nil")
	}

	// Verify coordinates are preserved
	if fetched.PrecisePoint.Lat != expectedLat || fetched.PrecisePoint.Lng != expectedLng {
		t.Errorf("Coordinates mismatch: got (%f, %f), want (%f, %f)",
			fetched.PrecisePoint.Lat, fetched.PrecisePoint.Lng, expectedLat, expectedLng)
	}

	// Verify consent flag is preserved
	if !fetched.AllowPrecise {
		t.Error("AllowPrecise should be true")
	}
}

// TestPrivacy_GeohashPrecision validates that only coarse geohash is used
// for public display without precise consent.
func TestPrivacy_GeohashPrecision(t *testing.T) {
	tests := []struct {
		name           string
		inputGeohash   string
		expectedOutput string
	}{
		{
			name:           "high precision geohash truncated to 6 chars",
			inputGeohash:   "9q8yyk8yuv9z", // Very precise (~few meters)
			expectedOutput: "9q8yyk",       // Coarse (~±0.61 km)
		},
		{
			name:           "medium precision geohash truncated to 6 chars",
			inputGeohash:   "dr5regw3p",
			expectedOutput: "dr5reg",
		},
		{
			name:           "already coarse geohash unchanged",
			inputGeohash:   "9q8yyk",
			expectedOutput: "9q8yyk",
		},
		{
			name:           "short geohash preserved",
			inputGeohash:   "9q8",
			expectedOutput: "9q8",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use the default privacy-preserving precision
			result := geo.RoundGeohash(tt.inputGeohash, geo.DefaultPrecision)
			if result != tt.expectedOutput {
				t.Errorf("RoundGeohash(%q, %d) = %q, want %q",
					tt.inputGeohash, geo.DefaultPrecision, result, tt.expectedOutput)
			}

			// Verify the output is coarse enough for privacy
			if len(result) > geo.DefaultPrecision {
				t.Errorf("Geohash too precise: length=%d, max allowed=%d", len(result), geo.DefaultPrecision)
			}
		})
	}
}

// TestPrivacy_ConsentRevocation validates that removing consent
// immediately removes precise coordinates.
func TestPrivacy_ConsentRevocation(t *testing.T) {
	repo := NewInMemorySceneRepository()

	// Step 1: Insert scene with consent
	scene := &Scene{
		ID:           "scene-consent-revoke",
		Name:         "Scene With Changing Consent",
		AllowPrecise: true,
		PrecisePoint: &Point{Lat: 51.5074, Lng: -0.1278}, // London
	}

	err := repo.Insert(scene)
	if err != nil {
		t.Fatalf("Insert() error = %v", err)
	}

	// Verify precise point is stored
	fetched, err := repo.GetByID("scene-consent-revoke")
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if fetched.PrecisePoint == nil {
		t.Fatal("With consent, PrecisePoint should not be nil")
	}

	// Step 2: Revoke consent (update with AllowPrecise=false)
	updatedScene := &Scene{
		ID:           "scene-consent-revoke",
		Name:         "Scene With Revoked Consent",
		AllowPrecise: false,                              // Consent revoked
		PrecisePoint: &Point{Lat: 51.5074, Lng: -0.1278}, // Still present in input
	}

	err = repo.Update(updatedScene)
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	// Step 3: Verify precise point is removed from storage
	reFetched, err := repo.GetByID("scene-consent-revoke")
	if err != nil {
		t.Fatalf("GetByID() after update error = %v", err)
	}

	// Critical privacy assertion: PrecisePoint must be nil after consent revocation
	if reFetched.PrecisePoint != nil {
		t.Errorf("Privacy violation: precise point persisted after consent revocation: %+v", reFetched.PrecisePoint)
	}

	if reFetched.AllowPrecise {
		t.Error("AllowPrecise should be false after revocation")
	}
}

// TestPrivacy_MultipleScenes_MixedConsent validates that in a batch scenario,
// consent is enforced independently for each scene.
func TestPrivacy_MultipleScenes_MixedConsent(t *testing.T) {
	repo := NewInMemorySceneRepository()

	scenes := []*Scene{
		{
			ID:           "scene-public-1",
			Name:         "Public Scene 1",
			AllowPrecise: true,
			PrecisePoint: &Point{Lat: 37.7749, Lng: -122.4194},
		},
		{
			ID:           "scene-private-1",
			Name:         "Private Scene 1",
			AllowPrecise: false,
			PrecisePoint: &Point{Lat: 40.7128, Lng: -74.0060},
		},
		{
			ID:           "scene-public-2",
			Name:         "Public Scene 2",
			AllowPrecise: true,
			PrecisePoint: &Point{Lat: 51.5074, Lng: -0.1278},
		},
		{
			ID:           "scene-private-2",
			Name:         "Private Scene 2",
			AllowPrecise: false,
			PrecisePoint: &Point{Lat: 48.8566, Lng: 2.3522},
		},
	}

	// Insert all scenes
	for _, s := range scenes {
		if err := repo.Insert(s); err != nil {
			t.Fatalf("Insert(%s) error = %v", s.ID, err)
		}
	}

	// Verify each scene independently
	for _, original := range scenes {
		fetched, err := repo.GetByID(original.ID)
		if err != nil {
			t.Fatalf("GetByID(%s) error = %v", original.ID, err)
		}

		if original.AllowPrecise {
			// Public scene - should have precise point
			if fetched.PrecisePoint == nil {
				t.Errorf("Scene %s: with consent, PrecisePoint should not be nil", original.ID)
			}
		} else {
			// Private scene - should NOT have precise point
			if fetched.PrecisePoint != nil {
				t.Errorf("Privacy violation: Scene %s without consent exposed precise coordinates: %+v",
					original.ID, fetched.PrecisePoint)
			}
		}
	}
}

// TestPrivacy_Upsert_PreservesConsent validates that upsert operations
// respect consent on both insert and update paths.
func TestPrivacy_Upsert_PreservesConsent(t *testing.T) {
	repo := NewInMemorySceneRepository()
	did := "did:example:privacy-test"
	rkey := "scene-upsert-privacy"

	// First upsert: no consent
	scene1 := &Scene{
		Name:         "Scene With No Consent",
		AllowPrecise: false,
		PrecisePoint: &Point{Lat: 37.7749, Lng: -122.4194},
		RecordDID:    strPtr(did),
		RecordRKey:   strPtr(rkey),
	}

	result1, err := repo.Upsert(scene1)
	if err != nil {
		t.Fatalf("First Upsert() error = %v", err)
	}

	// Verify no precise point stored
	fetched1, err := repo.GetByRecordKey(did, rkey)
	if err != nil {
		t.Fatalf("GetByRecordKey() error = %v", err)
	}
	if fetched1.PrecisePoint != nil {
		t.Errorf("Privacy violation: upsert insert without consent stored precise point: %+v", fetched1.PrecisePoint)
	}

	// Second upsert: add consent
	scene2 := &Scene{
		Name:         "Scene With Consent Added",
		AllowPrecise: true,
		PrecisePoint: &Point{Lat: 40.7128, Lng: -74.0060},
		RecordDID:    strPtr(did),
		RecordRKey:   strPtr(rkey),
	}

	result2, err := repo.Upsert(scene2)
	if err != nil {
		t.Fatalf("Second Upsert() error = %v", err)
	}

	if result2.Inserted {
		t.Error("Second upsert should be update, not insert")
	}

	if result1.ID != result2.ID {
		t.Error("Upsert should preserve ID")
	}

	// Verify precise point now stored
	fetched2, err := repo.GetByRecordKey(did, rkey)
	if err != nil {
		t.Fatalf("GetByRecordKey() after consent grant error = %v", err)
	}
	if fetched2.PrecisePoint == nil {
		t.Error("With consent, PrecisePoint should be stored")
	}

	// Third upsert: revoke consent
	scene3 := &Scene{
		Name:         "Scene With Consent Revoked",
		AllowPrecise: false,
		PrecisePoint: &Point{Lat: 40.7128, Lng: -74.0060}, // Still in input
		RecordDID:    strPtr(did),
		RecordRKey:   strPtr(rkey),
	}

	_, err = repo.Upsert(scene3)
	if err != nil {
		t.Fatalf("Third Upsert() error = %v", err)
	}

	// Verify precise point removed
	fetched3, err := repo.GetByRecordKey(did, rkey)
	if err != nil {
		t.Fatalf("GetByRecordKey() after consent revocation error = %v", err)
	}
	if fetched3.PrecisePoint != nil {
		t.Errorf("Privacy violation: upsert update after consent revocation stored precise point: %+v", fetched3.PrecisePoint)
	}
}

// TestPrivacy_CoarseGeohash_PublicAPI validates that the public API
// should return coarse geohash for scenes without precise consent.
// This is a documentation test showing the expected behavior.
func TestPrivacy_CoarseGeohash_PublicAPI(t *testing.T) {
	t.Run("scene without consent returns coarse geohash only", func(t *testing.T) {
		// This test documents the expected behavior:
		// 1. Scene has allow_precise=false
		// 2. Scene may have a coarse geohash stored (e.g., "9q8yyk")
		// 3. API should return only the coarse geohash, not precise coordinates
		// 4. Frontend can use the coarse geohash for map clustering

		scene := &Scene{
			ID:           "scene-coarse-only",
			Name:         "Scene with Coarse Location",
			AllowPrecise: false,
			PrecisePoint: nil, // No precise point with consent=false
		}

		// This scene would have a coarse_geohash field (e.g., "9q8yyk")
		// but no precise coordinates. The geohash provides ~±0.61 km accuracy,
		// sufficient for regional discovery without pinpointing exact venues.

		if scene.AllowPrecise {
			t.Error("Scene should not have precise consent")
		}

		if scene.PrecisePoint != nil {
			t.Error("Scene without consent should not have precise point")
		}

		// In a real API response, this would include:
		// {
		//   "id": "scene-coarse-only",
		//   "name": "Scene with Coarse Location",
		//   "allow_precise": false,
		//   "coarse_geohash": "9q8yyk",    // ~±0.61 km precision
		//   "precise_point": null          // Must be null
		// }
	})
}

// TestPrivacy_EXIF_Placeholder is a placeholder test for EXIF stripping functionality.
// EXIF stripping is now implemented in internal/image package.
func TestPrivacy_EXIF_Placeholder(t *testing.T) {
	// NOTE: EXIF stripping is implemented in internal/image package.
	// This placeholder now directs to the actual implementation.
	// See internal/image/processor_test.go for comprehensive EXIF stripping tests.
	//
	// The image.Process() function:
	// 1. Loads test images with embedded EXIF data (GPS coordinates, device info, timestamps)
	// 2. Processes images through bimg (libvips) to strip metadata
	// 3. Verifies all EXIF metadata is stripped from the output
	// 4. Verifies GPS coordinates are removed
	// 5. Verifies device identifiers are removed
	// 6. Verifies timestamps are removed
	// 7. Verifies image dimensions and quality are preserved
	//
	// Integration with media upload pipeline is tracked separately.
	// For now, API handlers should call image.Process() before storing uploaded images.
	t.Skip("EXIF stripping implemented in internal/image - integration with upload pipeline pending")
}

// TestPrivacy_LocationJitter_Placeholder is a placeholder test for location jitter functionality.
// Location jitter is planned but not yet implemented.
func TestPrivacy_LocationJitter_Placeholder(t *testing.T) {
	t.Skip("Location jitter not yet implemented - tracked in Privacy & Safety Epic #6")

	// When implemented, this test should:
	// 1. Take a precise coordinate (e.g., 37.7749, -122.4194)
	// 2. Apply deterministic jitter based on geohash
	// 3. Verify jittered coordinate is within acceptable bounds (e.g., ±500m)
	// 4. Verify jitter is deterministic (same input -> same output)
	// 5. Verify jitter prevents triangulation attacks
	// 6. Verify jitter is only applied for public display, not storage

	// Example assertions when implemented:
	// originalLat := 37.7749
	// originalLng := -122.4194
	// jitteredPoint := geo.ApplyJitter(originalLat, originalLng, geo.DefaultPrecision)
	//
	// // Jitter should move coordinates slightly
	// if jitteredPoint.Lat == originalLat && jitteredPoint.Lng == originalLng {
	//     t.Error("Jitter should modify coordinates")
	// }
	//
	// // Jitter should be within bounds (e.g., ±500m ~= ±0.0045 degrees at equator)
	// maxDelta := 0.01 // ~1.1 km, generous for testing
	// latDelta := math.Abs(jitteredPoint.Lat - originalLat)
	// lngDelta := math.Abs(jitteredPoint.Lng - originalLng)
	// if latDelta > maxDelta || lngDelta > maxDelta {
	//     t.Errorf("Jitter too large: latDelta=%f, lngDelta=%f, max=%f", latDelta, lngDelta, maxDelta)
	// }
	//
	// // Jitter should be deterministic
	// jitteredPoint2 := geo.ApplyJitter(originalLat, originalLng, geo.DefaultPrecision)
	// if jitteredPoint.Lat != jitteredPoint2.Lat || jitteredPoint.Lng != jitteredPoint2.Lng {
	//     t.Error("Jitter should be deterministic")
	// }
}
