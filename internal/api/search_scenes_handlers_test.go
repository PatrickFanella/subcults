package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/onnwee/subcults/internal/scene"
	"github.com/onnwee/subcults/internal/trust"
)

// TestSearchScenes_Success tests successful scene search.
func TestSearchScenes_Success(t *testing.T) {
	sceneRepo := scene.NewInMemorySceneRepository()
	handlers := NewSearchHandlers(sceneRepo, nil, nil, scene.NewInMemoryEventRepository())

	now := time.Now()

	// Create test scenes
	scene1 := &scene.Scene{
		ID:            uuid.New().String(),
		Name:          "Electronic Music Scene",
		Description:   "Underground techno parties",
		OwnerDID:      "did:plc:user1",
		AllowPrecise:  true,
		PrecisePoint:  &scene.Point{Lat: 40.7128, Lng: -74.0060}, // NYC
		CoarseGeohash: "dr5regw",
		Visibility:    scene.VisibilityPublic,
		CreatedAt:     &now,
		UpdatedAt:     &now,
	}

	scene2 := &scene.Scene{
		ID:            uuid.New().String(),
		Name:          "Jazz Collective",
		Description:   "Live jazz performances",
		OwnerDID:      "did:plc:user2",
		AllowPrecise:  true,
		PrecisePoint:  &scene.Point{Lat: 40.7589, Lng: -73.9851}, // Times Square
		CoarseGeohash: "dr5regw",
		Visibility:    scene.VisibilityPublic,
		CreatedAt:     &now,
		UpdatedAt:     &now,
	}

	if err := sceneRepo.Insert(scene1); err != nil {
		t.Fatalf("failed to insert scene1: %v", err)
	}
	if err := sceneRepo.Insert(scene2); err != nil {
		t.Fatalf("failed to insert scene2: %v", err)
	}

	// Create request with bbox and query
	req := httptest.NewRequest(http.MethodGet, "/search/scenes?q=electronic&bbox=-74.1,40.6,-73.9,40.8&limit=10", nil)
	w := httptest.NewRecorder()

	handlers.SearchScenes(w, req)

	// Check status code
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	// Parse response
	var response SceneSearchResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	// Should return 1 scene (electronic music)
	if response.Count != 1 {
		t.Errorf("expected 1 scene, got %d", response.Count)
	}

	if len(response.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(response.Results))
	}

	// Verify it's the correct scene
	if response.Results[0].ID != scene1.ID {
		t.Error("expected scene1 in results")
	}

	// Verify jittered coordinates are present
	if response.Results[0].JitteredPoint == nil {
		t.Error("expected jittered point to be present")
	}
}

// TestSearchScenes_Pagination tests cursor pagination.
func TestSearchScenes_Pagination(t *testing.T) {
	sceneRepo := scene.NewInMemorySceneRepository()
	handlers := NewSearchHandlers(sceneRepo, nil, nil, scene.NewInMemoryEventRepository())

	now := time.Now()

	// Create 5 scenes
	for i := 0; i < 5; i++ {
		s := &scene.Scene{
			ID:            uuid.New().String(),
			Name:          "Music Scene",
			OwnerDID:      "did:plc:user1",
			AllowPrecise:  true,
			PrecisePoint:  &scene.Point{Lat: 40.7128, Lng: -74.0060},
			CoarseGeohash: "dr5regw",
			Visibility:    scene.VisibilityPublic,
			CreatedAt:     &now,
			UpdatedAt:     &now,
		}
		if err := sceneRepo.Insert(s); err != nil {
			t.Fatalf("failed to insert scene: %v", err)
		}
	}

	// Get first page
	req1 := httptest.NewRequest(http.MethodGet, "/search/scenes?q=music&bbox=-74.1,40.6,-73.9,40.8&limit=2", nil)
	w1 := httptest.NewRecorder()
	handlers.SearchScenes(w1, req1)

	var response1 SceneSearchResponse
	if err := json.Unmarshal(w1.Body.Bytes(), &response1); err != nil {
		t.Fatalf("failed to parse response1: %v", err)
	}

	if len(response1.Results) != 2 {
		t.Errorf("expected 2 results in page 1, got %d", len(response1.Results))
	}

	if response1.NextCursor == "" {
		t.Fatal("expected next_cursor to be set")
	}

	// Get second page with cursor
	req2 := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/search/scenes?q=music&bbox=-74.1,40.6,-73.9,40.8&limit=2&cursor=%s", response1.NextCursor), nil)
	w2 := httptest.NewRecorder()
	handlers.SearchScenes(w2, req2)

	var response2 SceneSearchResponse
	if err := json.Unmarshal(w2.Body.Bytes(), &response2); err != nil {
		t.Fatalf("failed to parse response2: %v", err)
	}

	if len(response2.Results) != 2 {
		t.Errorf("expected 2 results in page 2, got %d", len(response2.Results))
	}

	// Verify no duplicates
	seenIDs := make(map[string]bool)
	for _, r := range response1.Results {
		seenIDs[r.ID] = true
	}
	for _, r := range response2.Results {
		if seenIDs[r.ID] {
			t.Errorf("duplicate scene ID %s in page 2", r.ID)
		}
	}
}

// TestSearchScenes_BboxValidation tests bbox parameter validation.
func TestSearchScenes_BboxValidation(t *testing.T) {
	sceneRepo := scene.NewInMemorySceneRepository()
	handlers := NewSearchHandlers(sceneRepo, nil, nil, scene.NewInMemoryEventRepository())

	tests := []struct {
		name       string
		bbox       string
		expectCode int
	}{
		{
			name:       "valid bbox",
			bbox:       "-74.1,40.6,-73.9,40.8",
			expectCode: http.StatusOK,
		},
		{
			name:       "invalid format (missing value)",
			bbox:       "-74.1,40.6,-73.9",
			expectCode: http.StatusBadRequest,
		},
		{
			name:       "invalid latitude (out of range)",
			bbox:       "-74.1,100,-73.9,40.8",
			expectCode: http.StatusBadRequest,
		},
		{
			name:       "invalid longitude (out of range)",
			bbox:       "-200,40.6,-73.9,40.8",
			expectCode: http.StatusBadRequest,
		},
		{
			name:       "min >= max (longitude)",
			bbox:       "-73.9,40.6,-74.1,40.8",
			expectCode: http.StatusBadRequest,
		},
		{
			name:       "min >= max (latitude)",
			bbox:       "-74.1,40.8,-73.9,40.6",
			expectCode: http.StatusBadRequest,
		},
		{
			name:       "bbox too large",
			bbox:       "-180,-90,180,90",
			expectCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/search/scenes?q=test&bbox=%s", tt.bbox), nil)
			w := httptest.NewRecorder()

			handlers.SearchScenes(w, req)

			if w.Code != tt.expectCode {
				t.Errorf("expected status %d, got %d", tt.expectCode, w.Code)
			}
		})
	}
}

// TestSearchScenes_RequiresBbox tests that bbox parameter is required.
func TestSearchScenes_RequiresBbox(t *testing.T) {
	sceneRepo := scene.NewInMemorySceneRepository()
	handlers := NewSearchHandlers(sceneRepo, nil, nil, scene.NewInMemoryEventRepository())

	// Request with neither q nor bbox
	req := httptest.NewRequest(http.MethodGet, "/search/scenes", nil)
	w := httptest.NewRecorder()

	handlers.SearchScenes(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}

	// Request with only q (no bbox) should also fail
	req2 := httptest.NewRequest(http.MethodGet, "/search/scenes?q=test", nil)
	w2 := httptest.NewRecorder()

	handlers.SearchScenes(w2, req2)

	if w2.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 for q without bbox, got %d", w2.Code)
	}
}

// TestSearchScenes_LimitValidation tests limit parameter validation.
func TestSearchScenes_LimitValidation(t *testing.T) {
	sceneRepo := scene.NewInMemorySceneRepository()
	handlers := NewSearchHandlers(sceneRepo, nil, nil, scene.NewInMemoryEventRepository())

	now := time.Now()

	// Create test scene
	s := &scene.Scene{
		ID:            uuid.New().String(),
		Name:          "Music Scene",
		OwnerDID:      "did:plc:user1",
		AllowPrecise:  true,
		PrecisePoint:  &scene.Point{Lat: 40.7128, Lng: -74.0060},
		CoarseGeohash: "dr5regw",
		Visibility:    scene.VisibilityPublic,
		CreatedAt:     &now,
		UpdatedAt:     &now,
	}
	if err := sceneRepo.Insert(s); err != nil {
		t.Fatalf("failed to insert scene: %v", err)
	}

	tests := []struct {
		name       string
		limit      string
		expectCode int
	}{
		{
			name:       "valid limit",
			limit:      "10",
			expectCode: http.StatusOK,
		},
		{
			name:       "limit exceeds max (should cap to 50)",
			limit:      "100",
			expectCode: http.StatusOK,
		},
		{
			name:       "invalid limit (not a number)",
			limit:      "abc",
			expectCode: http.StatusBadRequest,
		},
		{
			name:       "invalid limit (negative)",
			limit:      "-1",
			expectCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/search/scenes?q=music&bbox=-122.5,37.5,-122.0,38.0&limit=%s", tt.limit), nil)
			w := httptest.NewRecorder()

			handlers.SearchScenes(w, req)

			if w.Code != tt.expectCode {
				t.Errorf("expected status %d, got %d", tt.expectCode, w.Code)
			}
		})
	}
}

// TestSearchScenes_HiddenScenesExcluded tests that hidden scenes are excluded from search results.
func TestSearchScenes_HiddenScenesExcluded(t *testing.T) {
	sceneRepo := scene.NewInMemorySceneRepository()
	handlers := NewSearchHandlers(sceneRepo, nil, nil, scene.NewInMemoryEventRepository())

	now := time.Now()

	// Public scene
	publicScene := &scene.Scene{
		ID:            uuid.New().String(),
		Name:          "Public Scene",
		OwnerDID:      "did:plc:user1",
		AllowPrecise:  true,
		PrecisePoint:  &scene.Point{Lat: 40.7128, Lng: -74.0060},
		CoarseGeohash: "dr5regw",
		Visibility:    scene.VisibilityPublic,
		CreatedAt:     &now,
		UpdatedAt:     &now,
	}

	// Hidden scene
	hiddenScene := &scene.Scene{
		ID:            uuid.New().String(),
		Name:          "Hidden Scene",
		OwnerDID:      "did:plc:user2",
		AllowPrecise:  true,
		PrecisePoint:  &scene.Point{Lat: 40.7128, Lng: -74.0060},
		CoarseGeohash: "dr5regw",
		Visibility:    scene.VisibilityHidden,
		CreatedAt:     &now,
		UpdatedAt:     &now,
	}

	if err := sceneRepo.Insert(publicScene); err != nil {
		t.Fatalf("failed to insert publicScene: %v", err)
	}
	if err := sceneRepo.Insert(hiddenScene); err != nil {
		t.Fatalf("failed to insert hiddenScene: %v", err)
	}

	// Search should only return public scene
	req := httptest.NewRequest(http.MethodGet, "/search/scenes?q=scene&bbox=-74.1,40.6,-73.9,40.8", nil)
	w := httptest.NewRecorder()

	handlers.SearchScenes(w, req)

	var response SceneSearchResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if response.Count != 1 {
		t.Errorf("expected 1 scene (public only), got %d", response.Count)
	}

	if len(response.Results) > 0 && response.Results[0].ID != publicScene.ID {
		t.Error("expected only public scene in results")
	}
}

// TestSearchScenes_TrustRankingFlag tests that trust ranking can be disabled.
func TestSearchScenes_TrustRankingFlag(t *testing.T) {
	sceneRepo := scene.NewInMemorySceneRepository()

	// Create a mock trust store
	mockTrustStore := newMockTrustScoreStore()
	mockTrustStore.SetScore("scene1", 0.9)
	mockTrustStore.SetScore("scene2", 0.3)

	handlers := NewSearchHandlers(sceneRepo, nil, mockTrustStore, scene.NewInMemoryEventRepository())

	now := time.Now()

	// Create two scenes with same text/proximity scores
	scene1 := &scene.Scene{
		ID:            "scene1",
		Name:          "Music Scene",
		OwnerDID:      "did:plc:user1",
		AllowPrecise:  true,
		PrecisePoint:  &scene.Point{Lat: 40.7128, Lng: -74.0060},
		CoarseGeohash: "dr5regw",
		Visibility:    scene.VisibilityPublic,
		CreatedAt:     &now,
		UpdatedAt:     &now,
	}

	scene2 := &scene.Scene{
		ID:            "scene2",
		Name:          "Music Scene",
		OwnerDID:      "did:plc:user2",
		AllowPrecise:  true,
		PrecisePoint:  &scene.Point{Lat: 40.7128, Lng: -74.0060},
		CoarseGeohash: "dr5regw",
		Visibility:    scene.VisibilityPublic,
		CreatedAt:     &now,
		UpdatedAt:     &now,
	}

	if err := sceneRepo.Insert(scene1); err != nil {
		t.Fatalf("failed to insert scene1: %v", err)
	}
	if err := sceneRepo.Insert(scene2); err != nil {
		t.Fatalf("failed to insert scene2: %v", err)
	}

	// With trust ranking disabled, scenes should be ordered by ID (stable sort)
	trust.SetRankingEnabled(false)
	defer trust.SetRankingEnabled(false) // Reset after test

	req := httptest.NewRequest(http.MethodGet, "/search/scenes?q=music&bbox=-74.1,40.6,-73.9,40.8", nil)
	w := httptest.NewRecorder()

	handlers.SearchScenes(w, req)

	var response SceneSearchResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if len(response.Results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(response.Results))
	}

	// Verify trust scores are not included when disabled
	for _, result := range response.Results {
		if result.TrustScore != nil {
			t.Error("trust score should not be included when trust ranking is disabled")
		}
	}
}

func TestSearchScenes_LatLonAndGenresFilter(t *testing.T) {
	sceneRepo := scene.NewInMemorySceneRepository()
	handlers := NewSearchHandlers(sceneRepo, nil, nil, scene.NewInMemoryEventRepository())
	now := time.Now()

	techno := &scene.Scene{
		ID:            uuid.New().String(),
		Name:          "Warehouse",
		OwnerDID:      "did:plc:user1",
		AllowPrecise:  true,
		PrecisePoint:  &scene.Point{Lat: 40.7128, Lng: -74.0060},
		CoarseGeohash: "dr5regw",
		Tags:          []string{"techno"},
		Visibility:    scene.VisibilityPublic,
		CreatedAt:     &now,
		UpdatedAt:     &now,
	}
	jazz := &scene.Scene{
		ID:            uuid.New().String(),
		Name:          "Jazz Club",
		OwnerDID:      "did:plc:user2",
		AllowPrecise:  true,
		PrecisePoint:  &scene.Point{Lat: 40.7129, Lng: -74.0059},
		CoarseGeohash: "dr5regw",
		Tags:          []string{"jazz"},
		Visibility:    scene.VisibilityPublic,
		CreatedAt:     &now,
		UpdatedAt:     &now,
	}
	if err := sceneRepo.Insert(techno); err != nil {
		t.Fatalf("failed to insert techno scene: %v", err)
	}
	if err := sceneRepo.Insert(jazz); err != nil {
		t.Fatalf("failed to insert jazz scene: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/search/scenes?lat=40.7128&lon=-74.0060&genres=techno", nil)
	w := httptest.NewRecorder()
	handlers.SearchScenes(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var response SceneSearchResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if response.Count != 1 || response.Results[0].ID != techno.ID {
		t.Fatalf("expected only techno scene, got %d results", response.Count)
	}
}
