package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/onnwee/subcults/internal/post"
	"github.com/onnwee/subcults/internal/scene"
)

func TestSearchGlobal_MixedResultsWithTypeCaps(t *testing.T) {
	sceneRepo := scene.NewInMemorySceneRepository()
	eventRepo := scene.NewInMemoryEventRepository()
	postRepo := post.NewInMemoryPostRepository()
	handlers := NewSearchHandlers(sceneRepo, postRepo, nil, eventRepo)

	now := time.Now()
	for i := 0; i < 12; i++ {
		s := &scene.Scene{
			ID:            uuid.New().String(),
			Name:          fmt.Sprintf("Music Scene %d", i),
			Description:   "music",
			OwnerDID:      "did:plc:owner",
			AllowPrecise:  true,
			PrecisePoint:  &scene.Point{Lat: 40.7, Lng: -74.0},
			CoarseGeohash: "dr5regw",
			Visibility:    scene.VisibilityPublic,
			CreatedAt:     &now,
			UpdatedAt:     &now,
		}
		if err := sceneRepo.Insert(s); err != nil {
			t.Fatalf("failed to insert scene: %v", err)
		}
	}

	for i := 0; i < 12; i++ {
		e := &scene.Event{
			ID:            uuid.New().String(),
			SceneID:       uuid.New().String(),
			Title:         fmt.Sprintf("Music Event %d", i),
			AllowPrecise:  true,
			PrecisePoint:  &scene.Point{Lat: 40.7, Lng: -74.0},
			CoarseGeohash: "dr5regw",
			Status:        "scheduled",
			StartsAt:      now.Add(24 * time.Hour),
			CreatedAt:     &now,
			UpdatedAt:     &now,
		}
		if err := eventRepo.Insert(e); err != nil {
			t.Fatalf("failed to insert event: %v", err)
		}
	}

	for i := 0; i < 7; i++ {
		p := &post.Post{
			AuthorDID: "did:plc:author",
			Text:      fmt.Sprintf("music post %d", i),
		}
		if err := postRepo.Create(p); err != nil {
			t.Fatalf("failed to create post: %v", err)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/search/global?q=music", nil)
	w := httptest.NewRecorder()

	handlers.SearchGlobal(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var response GlobalSearchResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Count != 25 {
		t.Fatalf("expected 25 total results, got %d", response.Count)
	}

	sceneCount, eventCount, postCount := 0, 0, 0
	for _, result := range response.Results {
		switch result.Type {
		case "scene":
			sceneCount++
		case "event":
			eventCount++
		case "post":
			postCount++
		}
	}

	if sceneCount != 10 {
		t.Errorf("expected 10 scene results, got %d", sceneCount)
	}
	if eventCount != 10 {
		t.Errorf("expected 10 event results, got %d", eventCount)
	}
	if postCount != 5 {
		t.Errorf("expected 5 post results, got %d", postCount)
	}
	if response.NextCursor == "" {
		t.Error("expected next_cursor to be set when additional type results exist")
	}
}

func TestSearchGlobal_PaginationCursor(t *testing.T) {
	sceneRepo := scene.NewInMemorySceneRepository()
	eventRepo := scene.NewInMemoryEventRepository()
	postRepo := post.NewInMemoryPostRepository()
	handlers := NewSearchHandlers(sceneRepo, postRepo, nil, eventRepo)

	now := time.Now()
	for i := 0; i < 11; i++ {
		s := &scene.Scene{
			ID:            uuid.New().String(),
			Name:          fmt.Sprintf("Music Scene %d", i),
			OwnerDID:      "did:plc:owner",
			AllowPrecise:  true,
			PrecisePoint:  &scene.Point{Lat: 40.7, Lng: -74.0},
			CoarseGeohash: "dr5regw",
			Visibility:    scene.VisibilityPublic,
			CreatedAt:     &now,
			UpdatedAt:     &now,
		}
		if err := sceneRepo.Insert(s); err != nil {
			t.Fatalf("failed to insert scene: %v", err)
		}
	}

	req1 := httptest.NewRequest(http.MethodGet, "/search/global?q=music", nil)
	w1 := httptest.NewRecorder()
	handlers.SearchGlobal(w1, req1)

	if w1.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w1.Code)
	}

	var page1 GlobalSearchResponse
	if err := json.NewDecoder(w1.Body).Decode(&page1); err != nil {
		t.Fatalf("failed to decode first page: %v", err)
	}

	if len(page1.Results) != 10 {
		t.Fatalf("expected 10 results in first page, got %d", len(page1.Results))
	}
	if page1.NextCursor == "" {
		t.Fatal("expected next_cursor in first page")
	}

	req2 := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/search/global?q=music&cursor=%s", page1.NextCursor), nil)
	w2 := httptest.NewRecorder()
	handlers.SearchGlobal(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w2.Code)
	}

	var page2 GlobalSearchResponse
	if err := json.NewDecoder(w2.Body).Decode(&page2); err != nil {
		t.Fatalf("failed to decode second page: %v", err)
	}

	if len(page2.Results) != 1 {
		t.Fatalf("expected 1 result in second page, got %d", len(page2.Results))
	}
	if page2.NextCursor != "" {
		t.Errorf("expected empty next_cursor on final page, got %q", page2.NextCursor)
	}
}

func TestSearchGlobal_Validation(t *testing.T) {
	handlers := NewSearchHandlers(scene.NewInMemorySceneRepository(), post.NewInMemoryPostRepository(), nil, scene.NewInMemoryEventRepository())

	req := httptest.NewRequest(http.MethodGet, "/search/global", nil)
	w := httptest.NewRecorder()
	handlers.SearchGlobal(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 when q is missing, got %d", w.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/search/global?q=music&cursor=not-base64", nil)
	w = httptest.NewRecorder()
	handlers.SearchGlobal(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 for invalid cursor, got %d", w.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/search/global?q=music&limit=5", nil)
	w = httptest.NewRecorder()
	handlers.SearchGlobal(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 when limit is provided, got %d", w.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/search/global?q=music&lat=40.7", nil)
	w = httptest.NewRecorder()
	handlers.SearchGlobal(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 when lat/lon pair is invalid, got %d", w.Code)
	}
}
