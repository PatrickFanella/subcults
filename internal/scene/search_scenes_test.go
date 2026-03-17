package scene

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
)

// TestSearchScenes_TextSearch tests text search filtering for scenes.
func TestSearchScenes_TextSearch(t *testing.T) {
	repo := NewInMemorySceneRepository()

	now := time.Now()

	// Create scenes with different names and descriptions
	scene1 := &Scene{
		ID:            uuid.New().String(),
		Name:          "Electronic Music Scene",
		Description:   "Underground techno parties",
		OwnerDID:      "did:plc:user1",
		AllowPrecise:  true,
		PrecisePoint:  &Point{Lat: 40.7128, Lng: -74.0060},
		CoarseGeohash: "dr5regw",
		Visibility:    VisibilityPublic,
		CreatedAt:     &now,
		UpdatedAt:     &now,
	}

	scene2 := &Scene{
		ID:            uuid.New().String(),
		Name:          "Jazz Collective",
		Description:   "Live jazz performances weekly",
		OwnerDID:      "did:plc:user2",
		AllowPrecise:  true,
		PrecisePoint:  &Point{Lat: 40.7128, Lng: -74.0060},
		CoarseGeohash: "dr5regw",
		Visibility:    VisibilityPublic,
		CreatedAt:     &now,
		UpdatedAt:     &now,
	}

	scene3 := &Scene{
		ID:            uuid.New().String(),
		Name:          "Rock Venue",
		Description:   "Electronic rock fusion events",
		OwnerDID:      "did:plc:user3",
		AllowPrecise:  true,
		PrecisePoint:  &Point{Lat: 40.7128, Lng: -74.0060},
		CoarseGeohash: "dr5regw",
		Tags:          []string{"rock", "electronic"},
		Visibility:    VisibilityPublic,
		CreatedAt:     &now,
		UpdatedAt:     &now,
	}

	// Insert scenes
	if err := repo.Insert(scene1); err != nil {
		t.Fatalf("failed to insert scene1: %v", err)
	}
	if err := repo.Insert(scene2); err != nil {
		t.Fatalf("failed to insert scene2: %v", err)
	}
	if err := repo.Insert(scene3); err != nil {
		t.Fatalf("failed to insert scene3: %v", err)
	}

	// Search with query "electronic" - should match scene1 (name) and scene3 (description + tags)
	results, _, err := repo.SearchScenes(SceneSearchOptions{
		MinLng: -74.1,
		MinLat: 40.6,
		MaxLng: -73.9,
		MaxLat: 40.8,
		Query:  "electronic",
		Limit:  10,
	})
	if err != nil {
		t.Fatalf("failed to search: %v", err)
	}

	// Should return 2 scenes (scene1 and scene3)
	if len(results) != 2 {
		t.Errorf("expected 2 scenes matching 'electronic', got %d", len(results))
	}

	// Verify the matched scenes
	foundIDs := make(map[string]bool)
	for _, s := range results {
		foundIDs[s.ID] = true
	}

	if !foundIDs[scene1.ID] {
		t.Error("expected scene1 (Electronic Music Scene) in results")
	}
	if !foundIDs[scene3.ID] {
		t.Error("expected scene3 (Rock Venue with electronic in description and tags) in results")
	}
	if foundIDs[scene2.ID] {
		t.Error("scene2 (Jazz Collective) should not match 'electronic'")
	}
}

func TestSearchScenes_DisableProximityNeutralizesDistanceBias(t *testing.T) {
	repo := NewInMemorySceneRepository()
	now := time.Now()

	sceneFar := &Scene{
		ID:            "scene-a",
		Name:          "Music Scene",
		Description:   "music",
		OwnerDID:      "did:plc:user1",
		AllowPrecise:  true,
		PrecisePoint:  &Point{Lat: 40.7, Lng: -74.0},
		CoarseGeohash: "dr5regw",
		Visibility:    VisibilityPublic,
		CreatedAt:     &now,
		UpdatedAt:     &now,
	}
	sceneNearZero := &Scene{
		ID:            "scene-b",
		Name:          "Music Scene",
		Description:   "music",
		OwnerDID:      "did:plc:user2",
		AllowPrecise:  true,
		PrecisePoint:  &Point{Lat: 0.1, Lng: 0.1},
		CoarseGeohash: "s000000",
		Visibility:    VisibilityPublic,
		CreatedAt:     &now,
		UpdatedAt:     &now,
	}
	if err := repo.Insert(sceneFar); err != nil {
		t.Fatalf("failed to insert sceneFar: %v", err)
	}
	if err := repo.Insert(sceneNearZero); err != nil {
		t.Fatalf("failed to insert sceneNearZero: %v", err)
	}

	results, _, err := repo.SearchScenes(SceneSearchOptions{
		Query:            "music",
		Limit:            10,
		DisableProximity: true,
	})
	if err != nil {
		t.Fatalf("failed to search scenes: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	found := map[string]bool{}
	for _, result := range results {
		found[result.ID] = true
	}
	if !found["scene-a"] || !found["scene-b"] {
		t.Fatalf("expected both scenes when proximity is disabled, got %+v", found)
	}
}

// TestSearchScenes_Pagination tests cursor pagination for scene search.
func TestSearchScenes_Pagination(t *testing.T) {
	repo := NewInMemorySceneRepository()

	now := time.Now()

	// Create 5 scenes at the same location
	for i := 0; i < 5; i++ {
		scene := &Scene{
			ID:            uuid.New().String(),
			Name:          "Music Scene",
			Description:   "Description",
			OwnerDID:      "did:plc:user1",
			AllowPrecise:  true,
			PrecisePoint:  &Point{Lat: 40.7128, Lng: -74.0060},
			CoarseGeohash: "dr5regw",
			Visibility:    VisibilityPublic,
			CreatedAt:     &now,
			UpdatedAt:     &now,
		}
		if err := repo.Insert(scene); err != nil {
			t.Fatalf("failed to insert scene: %v", err)
		}
	}

	// Get first page (limit=2)
	results1, cursor1, err := repo.SearchScenes(SceneSearchOptions{
		MinLng: -74.1,
		MinLat: 40.6,
		MaxLng: -73.9,
		MaxLat: 40.8,
		Query:  "music",
		Limit:  2,
	})
	if err != nil {
		t.Fatalf("failed to search: %v", err)
	}

	if len(results1) != 2 {
		t.Errorf("expected 2 scenes in page 1, got %d", len(results1))
	}

	if cursor1 == "" {
		t.Fatal("expected cursor1 to be set")
	}

	// Get second page with cursor
	results2, cursor2, err := repo.SearchScenes(SceneSearchOptions{
		MinLng: -74.1,
		MinLat: 40.6,
		MaxLng: -73.9,
		MaxLat: 40.8,
		Query:  "music",
		Limit:  2,
		Cursor: cursor1,
	})
	if err != nil {
		t.Fatalf("failed to search with cursor: %v", err)
	}

	if len(results2) != 2 {
		t.Errorf("expected 2 scenes in page 2, got %d", len(results2))
	}

	// Get third page
	results3, _, err := repo.SearchScenes(SceneSearchOptions{
		MinLng: -74.1,
		MinLat: 40.6,
		MaxLng: -73.9,
		MaxLat: 40.8,
		Query:  "music",
		Limit:  2,
		Cursor: cursor2,
	})
	if err != nil {
		t.Fatalf("failed to search with cursor2: %v", err)
	}

	if len(results3) != 1 {
		t.Errorf("expected 1 scene in page 3, got %d", len(results3))
	}

	// Verify no duplicates across pages
	seenIDs := make(map[string]bool)
	allResults := append(append(results1, results2...), results3...)
	for _, scene := range allResults {
		if seenIDs[scene.ID] {
			t.Errorf("duplicate scene ID %s found across pages", scene.ID)
		}
		seenIDs[scene.ID] = true
	}

	// Should have all 5 scenes
	if len(seenIDs) != 5 {
		t.Errorf("expected 5 unique scenes across all pages, got %d", len(seenIDs))
	}
}

// TestSearchScenes_BboxFilter tests bounding box filtering.
func TestSearchScenes_BboxFilter(t *testing.T) {
	repo := NewInMemorySceneRepository()

	now := time.Now()

	// Scene inside bbox
	sceneInside := &Scene{
		ID:            uuid.New().String(),
		Name:          "Inside Scene",
		OwnerDID:      "did:plc:user1",
		AllowPrecise:  true,
		PrecisePoint:  &Point{Lat: 40.7128, Lng: -74.0060}, // NYC
		CoarseGeohash: "dr5regw",
		Visibility:    VisibilityPublic,
		CreatedAt:     &now,
		UpdatedAt:     &now,
	}

	// Scene outside bbox
	sceneOutside := &Scene{
		ID:            uuid.New().String(),
		Name:          "Outside Scene",
		OwnerDID:      "did:plc:user2",
		AllowPrecise:  true,
		PrecisePoint:  &Point{Lat: 34.0522, Lng: -118.2437}, // LA
		CoarseGeohash: "9q5ct2",
		Visibility:    VisibilityPublic,
		CreatedAt:     &now,
		UpdatedAt:     &now,
	}

	if err := repo.Insert(sceneInside); err != nil {
		t.Fatalf("failed to insert sceneInside: %v", err)
	}
	if err := repo.Insert(sceneOutside); err != nil {
		t.Fatalf("failed to insert sceneOutside: %v", err)
	}

	// Search with NYC bbox
	results, _, err := repo.SearchScenes(SceneSearchOptions{
		MinLng: -74.1,
		MinLat: 40.6,
		MaxLng: -73.9,
		MaxLat: 40.8,
		Query:  "",
		Limit:  10,
	})
	if err != nil {
		t.Fatalf("failed to search: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("expected 1 scene in bbox, got %d", len(results))
	}

	if results[0].ID != sceneInside.ID {
		t.Error("expected sceneInside to be in results")
	}
}

// TestSearchScenes_TrustScoreIntegration tests trust score weighting in ranking.
func TestSearchScenes_TrustScoreIntegration(t *testing.T) {
	repo := NewInMemorySceneRepository()

	now := time.Now()

	// Create two identical scenes except for ID
	scene1 := &Scene{
		ID:            "scene-low-trust",
		Name:          "Music Scene",
		OwnerDID:      "did:plc:user1",
		AllowPrecise:  true,
		PrecisePoint:  &Point{Lat: 40.7128, Lng: -74.0060},
		CoarseGeohash: "dr5regw",
		Visibility:    VisibilityPublic,
		CreatedAt:     &now,
		UpdatedAt:     &now,
	}

	scene2 := &Scene{
		ID:            "scene-high-trust",
		Name:          "Music Scene",
		OwnerDID:      "did:plc:user2",
		AllowPrecise:  true,
		PrecisePoint:  &Point{Lat: 40.7128, Lng: -74.0060},
		CoarseGeohash: "dr5regw",
		Visibility:    VisibilityPublic,
		CreatedAt:     &now,
		UpdatedAt:     &now,
	}

	if err := repo.Insert(scene1); err != nil {
		t.Fatalf("failed to insert scene1: %v", err)
	}
	if err := repo.Insert(scene2); err != nil {
		t.Fatalf("failed to insert scene2: %v", err)
	}

	// Search WITHOUT trust scores - should be ordered by ID (stable sort)
	resultsWithoutTrust, _, err := repo.SearchScenes(SceneSearchOptions{
		MinLng: -74.1,
		MinLat: 40.6,
		MaxLng: -73.9,
		MaxLat: 40.8,
		Query:  "music",
		Limit:  10,
	})
	if err != nil {
		t.Fatalf("failed to search: %v", err)
	}

	if len(resultsWithoutTrust) != 2 {
		t.Fatalf("expected 2 scenes, got %d", len(resultsWithoutTrust))
	}

	// Search WITH trust scores - high trust scene should rank first
	trustScores := map[string]float64{
		"scene-low-trust":  0.3,
		"scene-high-trust": 0.9,
	}

	resultsWithTrust, _, err := repo.SearchScenes(SceneSearchOptions{
		MinLng:      -74.1,
		MinLat:      40.6,
		MaxLng:      -73.9,
		MaxLat:      40.8,
		Query:       "music",
		Limit:       10,
		TrustScores: trustScores,
	})
	if err != nil {
		t.Fatalf("failed to search with trust: %v", err)
	}

	if len(resultsWithTrust) != 2 {
		t.Fatalf("expected 2 scenes, got %d", len(resultsWithTrust))
	}

	// High trust scene should rank first
	if resultsWithTrust[0].ID != "scene-high-trust" {
		t.Errorf("expected high-trust scene to rank first with trust scores, got %s", resultsWithTrust[0].ID)
	}
}

// TestSearchScenes_HiddenScenesExcluded tests that hidden scenes are excluded from search.
func TestSearchScenes_HiddenScenesExcluded(t *testing.T) {
	repo := NewInMemorySceneRepository()

	now := time.Now()

	// Public scene
	publicScene := &Scene{
		ID:            uuid.New().String(),
		Name:          "Public Scene",
		OwnerDID:      "did:plc:user1",
		AllowPrecise:  true,
		PrecisePoint:  &Point{Lat: 40.7128, Lng: -74.0060},
		CoarseGeohash: "dr5regw",
		Visibility:    VisibilityPublic,
		CreatedAt:     &now,
		UpdatedAt:     &now,
	}

	// Hidden scene
	hiddenScene := &Scene{
		ID:            uuid.New().String(),
		Name:          "Hidden Scene",
		OwnerDID:      "did:plc:user2",
		AllowPrecise:  true,
		PrecisePoint:  &Point{Lat: 40.7128, Lng: -74.0060},
		CoarseGeohash: "dr5regw",
		Visibility:    VisibilityHidden,
		CreatedAt:     &now,
		UpdatedAt:     &now,
	}

	if err := repo.Insert(publicScene); err != nil {
		t.Fatalf("failed to insert publicScene: %v", err)
	}
	if err := repo.Insert(hiddenScene); err != nil {
		t.Fatalf("failed to insert hiddenScene: %v", err)
	}

	// Search should only return public scene
	results, _, err := repo.SearchScenes(SceneSearchOptions{
		MinLng: -74.1,
		MinLat: 40.6,
		MaxLng: -73.9,
		MaxLat: 40.8,
		Query:  "scene",
		Limit:  10,
	})
	if err != nil {
		t.Fatalf("failed to search: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("expected 1 scene (public only), got %d", len(results))
	}

	if results[0].ID != publicScene.ID {
		t.Error("expected only public scene in results")
	}
}

// TestSearchScenes_DeletedScenesExcluded tests that deleted scenes are excluded from search.
func TestSearchScenes_DeletedScenesExcluded(t *testing.T) {
	repo := NewInMemorySceneRepository()

	now := time.Now()

	// Active scene
	activeScene := &Scene{
		ID:            uuid.New().String(),
		Name:          "Active Scene",
		OwnerDID:      "did:plc:user1",
		AllowPrecise:  true,
		PrecisePoint:  &Point{Lat: 40.7128, Lng: -74.0060},
		CoarseGeohash: "dr5regw",
		Visibility:    VisibilityPublic,
		CreatedAt:     &now,
		UpdatedAt:     &now,
	}

	// Deleted scene
	deletedScene := &Scene{
		ID:            uuid.New().String(),
		Name:          "Deleted Scene",
		OwnerDID:      "did:plc:user2",
		AllowPrecise:  true,
		PrecisePoint:  &Point{Lat: 40.7128, Lng: -74.0060},
		CoarseGeohash: "dr5regw",
		Visibility:    VisibilityPublic,
		CreatedAt:     &now,
		UpdatedAt:     &now,
		DeletedAt:     &now,
	}

	if err := repo.Insert(activeScene); err != nil {
		t.Fatalf("failed to insert activeScene: %v", err)
	}
	if err := repo.Insert(deletedScene); err != nil {
		t.Fatalf("failed to insert deletedScene: %v", err)
	}

	// Search should only return active scene
	results, _, err := repo.SearchScenes(SceneSearchOptions{
		MinLng: -74.1,
		MinLat: 40.6,
		MaxLng: -73.9,
		MaxLat: 40.8,
		Query:  "scene",
		Limit:  10,
	})
	if err != nil {
		t.Fatalf("failed to search: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("expected 1 scene (active only), got %d", len(results))
	}

	if results[0].ID != activeScene.ID {
		t.Error("expected only active scene in results")
	}
}

func TestSearchScenes_GenresFilter(t *testing.T) {
	repo := NewInMemorySceneRepository()
	now := time.Now()

	technoScene := &Scene{
		ID:            uuid.New().String(),
		Name:          "Warehouse Scene",
		OwnerDID:      "did:plc:user1",
		AllowPrecise:  true,
		PrecisePoint:  &Point{Lat: 40.7128, Lng: -74.0060},
		CoarseGeohash: "dr5regw",
		Tags:          []string{"techno"},
		Visibility:    VisibilityPublic,
		CreatedAt:     &now,
		UpdatedAt:     &now,
	}
	jazzScene := &Scene{
		ID:            uuid.New().String(),
		Name:          "Jazz Scene",
		OwnerDID:      "did:plc:user2",
		AllowPrecise:  true,
		PrecisePoint:  &Point{Lat: 40.7128, Lng: -74.0060},
		CoarseGeohash: "dr5regw",
		Tags:          []string{"jazz"},
		Visibility:    VisibilityPublic,
		CreatedAt:     &now,
		UpdatedAt:     &now,
	}

	if err := repo.Insert(technoScene); err != nil {
		t.Fatalf("failed to insert techno scene: %v", err)
	}
	if err := repo.Insert(jazzScene); err != nil {
		t.Fatalf("failed to insert jazz scene: %v", err)
	}

	results, _, err := repo.SearchScenes(SceneSearchOptions{
		MinLng: -74.1,
		MinLat: 40.6,
		MaxLng: -73.9,
		MaxLat: 40.8,
		Genres: []string{"techno"},
		Limit:  10,
		Offset: 0,
		Cursor: "",
		Query:  "",
	})
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}

	if len(results) != 1 || results[0].ID != technoScene.ID {
		t.Fatalf("expected only techno scene, got %d results", len(results))
	}
}

func TestSearchScenes_OffsetPagination(t *testing.T) {
	repo := NewInMemorySceneRepository()
	now := time.Now()

	for i := 0; i < 4; i++ {
		s := &Scene{
			ID:            fmt.Sprintf("scene-%d", i),
			Name:          "Music Scene",
			OwnerDID:      "did:plc:user1",
			AllowPrecise:  true,
			PrecisePoint:  &Point{Lat: 40.7128, Lng: -74.0060},
			CoarseGeohash: "dr5regw",
			Tags:          []string{"music"},
			Visibility:    VisibilityPublic,
			CreatedAt:     &now,
			UpdatedAt:     &now,
		}
		if err := repo.Insert(s); err != nil {
			t.Fatalf("failed to insert scene: %v", err)
		}
	}

	results, _, err := repo.SearchScenes(SceneSearchOptions{
		MinLng: -74.1,
		MinLat: 40.6,
		MaxLng: -73.9,
		MaxLat: 40.8,
		Query:  "music",
		Limit:  2,
		Offset: 1,
	})
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 results after offset, got %d", len(results))
	}
	if results[0].ID != "scene-1" {
		t.Fatalf("expected first offset result to be scene-1, got %s", results[0].ID)
	}
}
