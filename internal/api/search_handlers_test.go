package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/onnwee/subcults/internal/audit"
	"github.com/onnwee/subcults/internal/scene"
	"github.com/onnwee/subcults/internal/stream"
)

// TestSearchEvents_Success tests successful event search with bbox and time range.
func TestSearchEvents_Success(t *testing.T) {
	eventRepo := scene.NewInMemoryEventRepository()
	sceneRepo := scene.NewInMemorySceneRepository()
	auditRepo := audit.NewInMemoryRepository()
	rsvpRepo := scene.NewInMemoryRSVPRepository()
	streamRepo := stream.NewInMemorySessionRepository()
	handlers := NewEventHandlers(eventRepo, sceneRepo, auditRepo, rsvpRepo, streamRepo, nil)

	// Create test events at different locations and times
	baseTime := time.Now().Add(24 * time.Hour)

	// Event 1: Inside bbox, within time range
	event1 := &scene.Event{
		ID:            uuid.New().String(),
		SceneID:       uuid.New().String(),
		Title:         "Event 1",
		AllowPrecise:  true,
		PrecisePoint:  &scene.Point{Lat: 40.7128, Lng: -74.0060}, // NYC
		CoarseGeohash: "dr5regw",
		Status:        "scheduled",
		StartsAt:      baseTime.Add(1 * time.Hour),
		CreatedAt:     &baseTime,
		UpdatedAt:     &baseTime,
	}

	// Event 2: Inside bbox, within time range (later than event1)
	event2 := &scene.Event{
		ID:            uuid.New().String(),
		SceneID:       uuid.New().String(),
		Title:         "Event 2",
		AllowPrecise:  true,
		PrecisePoint:  &scene.Point{Lat: 40.7589, Lng: -73.9851}, // Times Square
		CoarseGeohash: "dr5regw",
		Status:        "scheduled",
		StartsAt:      baseTime.Add(2 * time.Hour),
		CreatedAt:     &baseTime,
		UpdatedAt:     &baseTime,
	}

	// Event 3: Outside bbox (should not be returned)
	event3 := &scene.Event{
		ID:            uuid.New().String(),
		SceneID:       uuid.New().String(),
		Title:         "Event 3",
		AllowPrecise:  true,
		PrecisePoint:  &scene.Point{Lat: 34.0522, Lng: -118.2437}, // LA
		CoarseGeohash: "9q5",
		Status:        "scheduled",
		StartsAt:      baseTime.Add(1 * time.Hour),
		CreatedAt:     &baseTime,
		UpdatedAt:     &baseTime,
	}

	// Event 4: Inside bbox, but cancelled (should not be returned)
	event4 := &scene.Event{
		ID:            uuid.New().String(),
		SceneID:       uuid.New().String(),
		Title:         "Event 4",
		AllowPrecise:  true,
		PrecisePoint:  &scene.Point{Lat: 40.7128, Lng: -74.0060},
		CoarseGeohash: "dr5regw",
		Status:        "cancelled",
		StartsAt:      baseTime.Add(1 * time.Hour),
		CreatedAt:     &baseTime,
		UpdatedAt:     &baseTime,
		CancelledAt:   &baseTime,
	}

	// Insert events
	if err := eventRepo.Insert(event1); err != nil {
		t.Fatalf("failed to insert event1: %v", err)
	}
	if err := eventRepo.Insert(event2); err != nil {
		t.Fatalf("failed to insert event2: %v", err)
	}
	if err := eventRepo.Insert(event3); err != nil {
		t.Fatalf("failed to insert event3: %v", err)
	}
	if err := eventRepo.Insert(event4); err != nil {
		t.Fatalf("failed to insert event4: %v", err)
	}

	// Search with bbox covering NYC area
	// minLng=-74.1, minLat=40.6, maxLng=-73.9, maxLat=40.8
	from := baseTime.Format(time.RFC3339)
	to := baseTime.Add(3 * time.Hour).Format(time.RFC3339)

	url := fmt.Sprintf("/search/events?bbox=-74.1,40.6,-73.9,40.8&from=%s&to=%s&limit=10", from, to)
	req := httptest.NewRequest(http.MethodGet, url, nil)
	w := httptest.NewRecorder()

	handlers.SearchEvents(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var response SearchEventsResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Should return 2 events (event1 and event2), not event3 (outside bbox) or event4 (cancelled)
	if len(response.Events) != 2 {
		t.Errorf("expected 2 events, got %d", len(response.Events))
	}

	// Verify sorting: event1 should come before event2 (earlier start time)
	if len(response.Events) >= 2 {
		if !response.Events[0].StartsAt.Before(response.Events[1].StartsAt) {
			t.Error("events should be sorted by starts_at ascending")
		}
	}
}

// TestSearchEvents_BboxValidation tests bbox parameter validation.
func TestSearchEvents_BboxValidation(t *testing.T) {
	eventRepo := scene.NewInMemoryEventRepository()
	sceneRepo := scene.NewInMemorySceneRepository()
	auditRepo := audit.NewInMemoryRepository()
	rsvpRepo := scene.NewInMemoryRSVPRepository()
	streamRepo := stream.NewInMemorySessionRepository()
	handlers := NewEventHandlers(eventRepo, sceneRepo, auditRepo, rsvpRepo, streamRepo, nil)

	baseTime := time.Now().Add(24 * time.Hour)
	from := baseTime.Format(time.RFC3339)
	to := baseTime.Add(1 * time.Hour).Format(time.RFC3339)

	tests := []struct {
		name       string
		bbox       string
		wantStatus int
		wantCode   string
	}{
		{
			name:       "missing bbox",
			bbox:       "",
			wantStatus: http.StatusBadRequest,
			wantCode:   ErrCodeValidation,
		},
		{
			name:       "invalid bbox format",
			bbox:       "-74.1,40.6,-73.9", // missing one coordinate
			wantStatus: http.StatusBadRequest,
			wantCode:   ErrCodeValidation,
		},
		{
			name:       "invalid longitude",
			bbox:       "-200,40.6,-73.9,40.8", // lng out of range
			wantStatus: http.StatusBadRequest,
			wantCode:   ErrCodeValidation,
		},
		{
			name:       "invalid latitude",
			bbox:       "-74.1,100,-73.9,40.8", // lat out of range
			wantStatus: http.StatusBadRequest,
			wantCode:   ErrCodeValidation,
		},
		{
			name:       "minLng >= maxLng",
			bbox:       "-73.9,40.6,-74.1,40.8", // minLng > maxLng
			wantStatus: http.StatusBadRequest,
			wantCode:   ErrCodeValidation,
		},
		{
			name:       "minLat >= maxLat",
			bbox:       "-74.1,40.8,-73.9,40.6", // minLat > maxLat
			wantStatus: http.StatusBadRequest,
			wantCode:   ErrCodeValidation,
		},
		{
			name:       "non-numeric coordinate",
			bbox:       "invalid,40.6,-73.9,40.8",
			wantStatus: http.StatusBadRequest,
			wantCode:   ErrCodeValidation,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := fmt.Sprintf("/search/events?bbox=%s&from=%s&to=%s", tt.bbox, from, to)
			req := httptest.NewRequest(http.MethodGet, url, nil)
			w := httptest.NewRecorder()

			handlers.SearchEvents(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, w.Code)
			}

			var errResp ErrorResponse
			if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
				t.Fatalf("failed to decode error response: %v", err)
			}

			if errResp.Error.Code != tt.wantCode {
				t.Errorf("expected error code %s, got %s", tt.wantCode, errResp.Error.Code)
			}
		})
	}
}

// TestSearchEvents_TimeRangeValidation tests time range parameter validation.
func TestSearchEvents_TimeRangeValidation(t *testing.T) {
	eventRepo := scene.NewInMemoryEventRepository()
	sceneRepo := scene.NewInMemorySceneRepository()
	auditRepo := audit.NewInMemoryRepository()
	rsvpRepo := scene.NewInMemoryRSVPRepository()
	streamRepo := stream.NewInMemorySessionRepository()
	handlers := NewEventHandlers(eventRepo, sceneRepo, auditRepo, rsvpRepo, streamRepo, nil)

	baseTime := time.Now().Add(24 * time.Hour)

	tests := []struct {
		name       string
		from       string
		to         string
		wantStatus int
		wantCode   string
	}{
		{
			name:       "missing from",
			from:       "",
			to:         baseTime.Format(time.RFC3339),
			wantStatus: http.StatusBadRequest,
			wantCode:   ErrCodeValidation,
		},
		{
			name:       "missing to",
			from:       baseTime.Format(time.RFC3339),
			to:         "",
			wantStatus: http.StatusBadRequest,
			wantCode:   ErrCodeValidation,
		},
		{
			name:       "invalid from format",
			from:       "2024-01-01",
			to:         baseTime.Format(time.RFC3339),
			wantStatus: http.StatusBadRequest,
			wantCode:   ErrCodeValidation,
		},
		{
			name:       "invalid to format",
			from:       baseTime.Format(time.RFC3339),
			to:         "invalid",
			wantStatus: http.StatusBadRequest,
			wantCode:   ErrCodeValidation,
		},
		{
			name:       "from after to",
			from:       baseTime.Add(2 * time.Hour).Format(time.RFC3339),
			to:         baseTime.Format(time.RFC3339),
			wantStatus: http.StatusBadRequest,
			wantCode:   ErrCodeInvalidTimeRange,
		},
		{
			name:       "from equals to",
			from:       baseTime.Format(time.RFC3339),
			to:         baseTime.Format(time.RFC3339),
			wantStatus: http.StatusBadRequest,
			wantCode:   ErrCodeInvalidTimeRange,
		},
		{
			name:       "window exceeds 30 days",
			from:       baseTime.Format(time.RFC3339),
			to:         baseTime.Add(31 * 24 * time.Hour).Format(time.RFC3339),
			wantStatus: http.StatusBadRequest,
			wantCode:   ErrCodeValidation,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := fmt.Sprintf("/search/events?bbox=-74.1,40.6,-73.9,40.8&from=%s&to=%s", tt.from, tt.to)
			req := httptest.NewRequest(http.MethodGet, url, nil)
			w := httptest.NewRecorder()

			handlers.SearchEvents(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d: %s", tt.wantStatus, w.Code, w.Body.String())
			}

			var errResp ErrorResponse
			if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
				t.Fatalf("failed to decode error response: %v", err)
			}

			if errResp.Error.Code != tt.wantCode {
				t.Errorf("expected error code %s, got %s", tt.wantCode, errResp.Error.Code)
			}
		})
	}
}

// TestSearchEvents_Pagination tests cursor-based pagination.
func TestSearchEvents_Pagination(t *testing.T) {
	eventRepo := scene.NewInMemoryEventRepository()
	sceneRepo := scene.NewInMemorySceneRepository()
	auditRepo := audit.NewInMemoryRepository()
	rsvpRepo := scene.NewInMemoryRSVPRepository()
	streamRepo := stream.NewInMemorySessionRepository()
	handlers := NewEventHandlers(eventRepo, sceneRepo, auditRepo, rsvpRepo, streamRepo, nil)

	baseTime := time.Now().Add(24 * time.Hour)

	// Create 5 events
	for i := 0; i < 5; i++ {
		event := &scene.Event{
			ID:            uuid.New().String(),
			SceneID:       uuid.New().String(),
			Title:         fmt.Sprintf("Event %d", i+1),
			AllowPrecise:  true,
			PrecisePoint:  &scene.Point{Lat: 40.7128, Lng: -74.0060},
			CoarseGeohash: "dr5regw",
			Status:        "scheduled",
			StartsAt:      baseTime.Add(time.Duration(i) * time.Hour),
			CreatedAt:     &baseTime,
			UpdatedAt:     &baseTime,
		}
		if err := eventRepo.Insert(event); err != nil {
			t.Fatalf("failed to insert event: %v", err)
		}
	}

	from := baseTime.Format(time.RFC3339)
	to := baseTime.Add(6 * time.Hour).Format(time.RFC3339)

	// First page: limit=2
	url := fmt.Sprintf("/search/events?bbox=-74.1,40.6,-73.9,40.8&from=%s&to=%s&limit=2", from, to)
	req := httptest.NewRequest(http.MethodGet, url, nil)
	w := httptest.NewRecorder()

	handlers.SearchEvents(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var response1 SearchEventsResponse
	if err := json.NewDecoder(w.Body).Decode(&response1); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(response1.Events) != 2 {
		t.Errorf("expected 2 events in first page, got %d", len(response1.Events))
	}

	if response1.NextCursor == "" {
		t.Error("expected next_cursor to be set")
	}

	// Second page: use cursor
	url2 := fmt.Sprintf("/search/events?bbox=-74.1,40.6,-73.9,40.8&from=%s&to=%s&limit=2&cursor=%s", from, to, response1.NextCursor)
	req2 := httptest.NewRequest(http.MethodGet, url2, nil)
	w2 := httptest.NewRecorder()

	handlers.SearchEvents(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w2.Code, w2.Body.String())
	}

	var response2 SearchEventsResponse
	if err := json.NewDecoder(w2.Body).Decode(&response2); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(response2.Events) != 2 {
		t.Errorf("expected 2 events in second page, got %d", len(response2.Events))
	}

	// Events from page 2 should be different from page 1 (pagination works correctly)
	// Note: ordering is by composite score (recency, proximity, text match, trust), not start time
	if len(response1.Events) > 0 && len(response2.Events) > 0 {
		// The first event of page 2 should be different from any event in page 1
		firstFromPage2ID := response2.Events[0].ID
		for _, e := range response1.Events {
			if e.ID == firstFromPage2ID {
				t.Error("pagination should not repeat events: first event from page 2 found in page 1")
				break
			}
		}
	}

	// Verify no duplicate events between pages
	seenIDs := make(map[string]bool)
	for _, event := range response1.Events {
		seenIDs[event.ID] = true
	}
	for _, event := range response2.Events {
		if seenIDs[event.ID] {
			t.Errorf("duplicate event ID %s found across pages", event.ID)
		}
	}
}

// TestSearchEvents_LimitValidation tests limit parameter validation.
func TestSearchEvents_LimitValidation(t *testing.T) {
	eventRepo := scene.NewInMemoryEventRepository()
	sceneRepo := scene.NewInMemorySceneRepository()
	auditRepo := audit.NewInMemoryRepository()
	rsvpRepo := scene.NewInMemoryRSVPRepository()
	streamRepo := stream.NewInMemorySessionRepository()
	handlers := NewEventHandlers(eventRepo, sceneRepo, auditRepo, rsvpRepo, streamRepo, nil)

	baseTime := time.Now().Add(24 * time.Hour)
	from := baseTime.Format(time.RFC3339)
	to := baseTime.Add(1 * time.Hour).Format(time.RFC3339)

	tests := []struct {
		name       string
		limit      string
		wantStatus int
	}{
		{
			name:       "limit below minimum",
			limit:      "0",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "limit above maximum",
			limit:      "101",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid limit",
			limit:      "invalid",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "valid limit",
			limit:      "25",
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := fmt.Sprintf("/search/events?bbox=-74.1,40.6,-73.9,40.8&from=%s&to=%s&limit=%s", from, to, tt.limit)
			req := httptest.NewRequest(http.MethodGet, url, nil)
			w := httptest.NewRecorder()

			handlers.SearchEvents(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d: %s", tt.wantStatus, w.Code, w.Body.String())
			}
		})
	}
}

// TestSearchEvents_EmptyResults tests search with no matching events.
func TestSearchEvents_EmptyResults(t *testing.T) {
	eventRepo := scene.NewInMemoryEventRepository()
	sceneRepo := scene.NewInMemorySceneRepository()
	auditRepo := audit.NewInMemoryRepository()
	rsvpRepo := scene.NewInMemoryRSVPRepository()
	streamRepo := stream.NewInMemorySessionRepository()
	handlers := NewEventHandlers(eventRepo, sceneRepo, auditRepo, rsvpRepo, streamRepo, nil)

	baseTime := time.Now().Add(24 * time.Hour)
	from := baseTime.Format(time.RFC3339)
	to := baseTime.Add(1 * time.Hour).Format(time.RFC3339)

	url := fmt.Sprintf("/search/events?bbox=-74.1,40.6,-73.9,40.8&from=%s&to=%s", from, to)
	req := httptest.NewRequest(http.MethodGet, url, nil)
	w := httptest.NewRecorder()

	handlers.SearchEvents(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var response SearchEventsResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Events == nil {
		t.Error("events array should not be nil, even when empty")
	}

	if len(response.Events) != 0 {
		t.Errorf("expected 0 events, got %d", len(response.Events))
	}

	if response.NextCursor != "" {
		t.Error("next_cursor should be empty when there are no more results")
	}
}

// mockTrustScoreStore is a simple in-memory trust store for testing.
type mockTrustScoreStore struct {
	scores map[string]float64
}

func newMockTrustScoreStore() *mockTrustScoreStore {
	return &mockTrustScoreStore{
		scores: make(map[string]float64),
	}
}

func (m *mockTrustScoreStore) SetScore(sceneID string, score float64) {
	m.scores[sceneID] = score
}

func (m *mockTrustScoreStore) GetScore(sceneID string) (*TrustScore, error) {
	score, ok := m.scores[sceneID]
	if !ok {
		return nil, nil
	}
	return &TrustScore{
		SceneID: sceneID,
		Score:   score,
	}, nil
}

// TestSearchEvents_WithTrustRanking tests trust-weighted ranking in the handler.
func TestSearchEvents_WithTrustRanking(t *testing.T) {
	eventRepo := scene.NewInMemoryEventRepository()
	sceneRepo := scene.NewInMemorySceneRepository()
	auditRepo := audit.NewInMemoryRepository()
	rsvpRepo := scene.NewInMemoryRSVPRepository()
	streamRepo := stream.NewInMemorySessionRepository()

	// Create trust store with different scores
	trustStore := newMockTrustScoreStore()
	trustStore.SetScore("scene-low-trust", 0.3)
	trustStore.SetScore("scene-high-trust", 0.9)

	handlers := NewEventHandlers(eventRepo, sceneRepo, auditRepo, rsvpRepo, streamRepo, trustStore)

	baseTime := time.Now().Add(24 * time.Hour)

	// Create two events with identical characteristics except scene trust
	// Both have same time, location, and title
	event1 := &scene.Event{
		ID:            uuid.New().String(),
		SceneID:       "scene-low-trust",
		Title:         "Music Event",
		AllowPrecise:  true,
		PrecisePoint:  &scene.Point{Lat: 40.7128, Lng: -74.0060},
		CoarseGeohash: "dr5regw",
		Status:        "scheduled",
		StartsAt:      baseTime.Add(1 * time.Hour),
		CreatedAt:     &baseTime,
		UpdatedAt:     &baseTime,
	}

	event2 := &scene.Event{
		ID:            uuid.New().String(),
		SceneID:       "scene-high-trust",
		Title:         "Music Event",
		AllowPrecise:  true,
		PrecisePoint:  &scene.Point{Lat: 40.7128, Lng: -74.0060},
		CoarseGeohash: "dr5regw",
		Status:        "scheduled",
		StartsAt:      baseTime.Add(1 * time.Hour), // Same time
		CreatedAt:     &baseTime,
		UpdatedAt:     &baseTime,
	}

	// Insert events
	if err := eventRepo.Insert(event1); err != nil {
		t.Fatalf("failed to insert event1: %v", err)
	}
	if err := eventRepo.Insert(event2); err != nil {
		t.Fatalf("failed to insert event2: %v", err)
	}

	// Search with trust ranking enabled
	from := baseTime.Format(time.RFC3339)
	to := baseTime.Add(3 * time.Hour).Format(time.RFC3339)

	url := fmt.Sprintf("/search/events?bbox=-74.1,40.6,-73.9,40.8&from=%s&to=%s&q=music&limit=10", from, to)
	req := httptest.NewRequest(http.MethodGet, url, nil)
	w := httptest.NewRecorder()

	handlers.SearchEvents(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var response SearchEventsResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(response.Events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(response.Events))
	}

	// High trust scene should rank first due to trust weighting
	if response.Events[0].Event.SceneID != "scene-high-trust" {
		t.Errorf("expected high-trust scene to rank first, got scene_id=%s", response.Events[0].Event.SceneID)
	}

	if response.Events[1].Event.SceneID != "scene-low-trust" {
		t.Errorf("expected low-trust scene to rank second, got scene_id=%s", response.Events[1].Event.SceneID)
	}
}

// TestSearchEvents_TrustRankingWithPagination tests cursor stability with trust ranking.
func TestSearchEvents_TrustRankingWithPagination(t *testing.T) {
	eventRepo := scene.NewInMemoryEventRepository()
	sceneRepo := scene.NewInMemorySceneRepository()
	auditRepo := audit.NewInMemoryRepository()
	rsvpRepo := scene.NewInMemoryRSVPRepository()
	streamRepo := stream.NewInMemorySessionRepository()

	// Create trust store
	trustStore := newMockTrustScoreStore()
	trustStore.SetScore("scene1", 0.5)
	trustStore.SetScore("scene2", 0.7)
	trustStore.SetScore("scene3", 0.9)

	handlers := NewEventHandlers(eventRepo, sceneRepo, auditRepo, rsvpRepo, streamRepo, trustStore)

	baseTime := time.Now().Add(24 * time.Hour)

	// Create 3 events with different trust scores
	for i, sceneID := range []string{"scene1", "scene2", "scene3"} {
		event := &scene.Event{
			ID:            uuid.New().String(),
			SceneID:       sceneID,
			Title:         "Music Event",
			AllowPrecise:  true,
			PrecisePoint:  &scene.Point{Lat: 40.7128, Lng: -74.0060},
			CoarseGeohash: "dr5regw",
			Status:        "scheduled",
			StartsAt:      baseTime.Add(time.Duration(i) * time.Hour),
			CreatedAt:     &baseTime,
			UpdatedAt:     &baseTime,
		}
		if err := eventRepo.Insert(event); err != nil {
			t.Fatalf("failed to insert event: %v", err)
		}
	}

	from := baseTime.Format(time.RFC3339)
	to := baseTime.Add(6 * time.Hour).Format(time.RFC3339)

	// Get first page (limit=2)
	url1 := fmt.Sprintf("/search/events?bbox=-74.1,40.6,-73.9,40.8&from=%s&to=%s&q=music&limit=2", from, to)
	req1 := httptest.NewRequest(http.MethodGet, url1, nil)
	w1 := httptest.NewRecorder()

	handlers.SearchEvents(w1, req1)

	if w1.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w1.Code, w1.Body.String())
	}

	var response1 SearchEventsResponse
	if err := json.NewDecoder(w1.Body).Decode(&response1); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(response1.Events) != 2 {
		t.Fatalf("expected 2 events in first page, got %d", len(response1.Events))
	}

	if response1.NextCursor == "" {
		t.Error("expected next_cursor to be set")
	}

	// Get second page with cursor
	url2 := fmt.Sprintf("/search/events?bbox=-74.1,40.6,-73.9,40.8&from=%s&to=%s&q=music&limit=2&cursor=%s", from, to, response1.NextCursor)
	req2 := httptest.NewRequest(http.MethodGet, url2, nil)
	w2 := httptest.NewRecorder()

	handlers.SearchEvents(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w2.Code, w2.Body.String())
	}

	var response2 SearchEventsResponse
	if err := json.NewDecoder(w2.Body).Decode(&response2); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(response2.Events) != 1 {
		t.Fatalf("expected 1 event in second page, got %d", len(response2.Events))
	}

	// Verify no duplicate events between pages
	seenIDs := make(map[string]bool)
	for _, event := range response1.Events {
		seenIDs[event.Event.ID] = true
	}
	for _, event := range response2.Events {
		if seenIDs[event.Event.ID] {
			t.Errorf("duplicate event ID %s found across pages", event.Event.ID)
		}
	}
}

// TestSearchEvents_TrustStoreError tests that trust store errors don't fail the request.
func TestSearchEvents_TrustStoreError(t *testing.T) {
	eventRepo := scene.NewInMemoryEventRepository()
	sceneRepo := scene.NewInMemorySceneRepository()
	auditRepo := audit.NewInMemoryRepository()
	rsvpRepo := scene.NewInMemoryRSVPRepository()
	streamRepo := stream.NewInMemorySessionRepository()

	// Create a failing trust store
	trustStore := &failingTrustStore{}

	handlers := NewEventHandlers(eventRepo, sceneRepo, auditRepo, rsvpRepo, streamRepo, trustStore)

	baseTime := time.Now().Add(24 * time.Hour)

	event := &scene.Event{
		ID:            uuid.New().String(),
		SceneID:       "scene1",
		Title:         "Music Event",
		AllowPrecise:  true,
		PrecisePoint:  &scene.Point{Lat: 40.7128, Lng: -74.0060},
		CoarseGeohash: "dr5regw",
		Status:        "scheduled",
		StartsAt:      baseTime.Add(1 * time.Hour),
		CreatedAt:     &baseTime,
		UpdatedAt:     &baseTime,
	}

	if err := eventRepo.Insert(event); err != nil {
		t.Fatalf("failed to insert event: %v", err)
	}

	from := baseTime.Format(time.RFC3339)
	to := baseTime.Add(3 * time.Hour).Format(time.RFC3339)

	url := fmt.Sprintf("/search/events?bbox=-74.1,40.6,-73.9,40.8&from=%s&to=%s&q=music&limit=10", from, to)
	req := httptest.NewRequest(http.MethodGet, url, nil)
	w := httptest.NewRecorder()

	handlers.SearchEvents(w, req)

	// Should still succeed despite trust store errors
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var response SearchEventsResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Should still return the event
	if len(response.Events) != 1 {
		t.Errorf("expected 1 event, got %d", len(response.Events))
	}
}

// failingTrustStore always returns errors for testing error handling.
type failingTrustStore struct{}

func (f *failingTrustStore) GetScore(sceneID string) (*TrustScore, error) {
	return nil, fmt.Errorf("simulated trust store error")
}

type batchErrorSceneRepo struct {
	*scene.InMemorySceneRepository
	batchCalls   int
	getByIDCalls int
}

func (r *batchErrorSceneRepo) GetByIDs(ids []string) ([]*scene.Scene, error) {
	r.batchCalls++
	return nil, fmt.Errorf("simulated batch scene fetch error")
}

func (r *batchErrorSceneRepo) GetByID(id string) (*scene.Scene, error) {
	r.getByIDCalls++
	return r.InMemorySceneRepository.GetByID(id)
}

func TestSearchEvents_StatusSceneOrganizerAndParentScene(t *testing.T) {
	eventRepo := scene.NewInMemoryEventRepository()
	sceneRepo := scene.NewInMemorySceneRepository()
	auditRepo := audit.NewInMemoryRepository()
	rsvpRepo := scene.NewInMemoryRSVPRepository()
	streamRepo := stream.NewInMemorySessionRepository()
	handlers := NewEventHandlers(eventRepo, sceneRepo, auditRepo, rsvpRepo, streamRepo, nil)

	baseTime := time.Now().Add(24 * time.Hour)
	organizerDID := "did:plc:organizer123"
	otherDID := "did:plc:other456"

	organizerScene := &scene.Scene{
		ID:           uuid.New().String(),
		Name:         "Organizer Scene",
		OwnerDID:     organizerDID,
		AllowPrecise: true,
	}
	otherScene := &scene.Scene{
		ID:           uuid.New().String(),
		Name:         "Other Scene",
		OwnerDID:     otherDID,
		AllowPrecise: true,
	}
	if err := sceneRepo.Insert(organizerScene); err != nil {
		t.Fatalf("failed to insert organizer scene: %v", err)
	}
	if err := sceneRepo.Insert(otherScene); err != nil {
		t.Fatalf("failed to insert other scene: %v", err)
	}

	upcomingEvent := &scene.Event{
		ID:            uuid.New().String(),
		SceneID:       organizerScene.ID,
		Title:         "Organizer Upcoming Event",
		Description:   "test event",
		AllowPrecise:  true,
		PrecisePoint:  &scene.Point{Lat: 40.7128, Lng: -74.0060},
		CoarseGeohash: "dr5regw",
		Status:        "scheduled",
		StartsAt:      baseTime.Add(2 * time.Hour),
		CreatedAt:     &baseTime,
		UpdatedAt:     &baseTime,
	}
	cancelledEvent := &scene.Event{
		ID:            uuid.New().String(),
		SceneID:       organizerScene.ID,
		Title:         "Organizer Cancelled Event",
		Description:   "cancelled event",
		AllowPrecise:  true,
		PrecisePoint:  &scene.Point{Lat: 40.7129, Lng: -74.0059},
		CoarseGeohash: "dr5regw",
		Status:        "cancelled",
		StartsAt:      baseTime.Add(3 * time.Hour),
		CreatedAt:     &baseTime,
		UpdatedAt:     &baseTime,
	}
	otherOrganizerEvent := &scene.Event{
		ID:            uuid.New().String(),
		SceneID:       otherScene.ID,
		Title:         "Other Organizer Event",
		Description:   "other organizer",
		AllowPrecise:  true,
		PrecisePoint:  &scene.Point{Lat: 40.7130, Lng: -74.0058},
		CoarseGeohash: "dr5regw",
		Status:        "scheduled",
		StartsAt:      baseTime.Add(4 * time.Hour),
		CreatedAt:     &baseTime,
		UpdatedAt:     &baseTime,
	}
	for _, event := range []*scene.Event{upcomingEvent, cancelledEvent, otherOrganizerEvent} {
		if err := eventRepo.Insert(event); err != nil {
			t.Fatalf("failed to insert event %s: %v", event.ID, err)
		}
	}

	startDate := baseTime.Add(-1 * time.Hour).Format(time.RFC3339)
	endDate := baseTime.Add(8 * time.Hour).Format(time.RFC3339)

	searchQuery := url.Values{}
	searchQuery.Set("bbox", "-74.2,40.6,-73.8,40.9")
	searchQuery.Set("start_date", startDate)
	searchQuery.Set("end_date", endDate)
	searchQuery.Set("status", "upcoming")
	searchQuery.Set("organizer", organizerDID)
	searchQuery.Set("scene_id", organizerScene.ID)
	req := httptest.NewRequest(http.MethodGet, "/search/events?"+searchQuery.Encode(), nil)
	w := httptest.NewRecorder()
	handlers.SearchEvents(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var response SearchEventsResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(response.Events) != 1 {
		t.Fatalf("expected 1 event for organizer+scene+upcoming filters, got %d", len(response.Events))
	}
	if response.Events[0].ID != upcomingEvent.ID {
		t.Fatalf("expected upcoming event %s, got %s", upcomingEvent.ID, response.Events[0].ID)
	}
	if response.Events[0].Scene == nil || response.Events[0].Scene.ID != organizerScene.ID {
		t.Fatalf("expected parent scene info for scene %s", organizerScene.ID)
	}

	cancelledQuery := url.Values{}
	cancelledQuery.Set("bbox", "-74.2,40.6,-73.8,40.9")
	cancelledQuery.Set("from", startDate)
	cancelledQuery.Set("to", endDate)
	cancelledQuery.Set("status", "cancelled")
	cancelledQuery.Set("scene_id", organizerScene.ID)
	cancelledReq := httptest.NewRequest(http.MethodGet, "/search/events?"+cancelledQuery.Encode(), nil)
	cancelledW := httptest.NewRecorder()
	handlers.SearchEvents(cancelledW, cancelledReq)

	if cancelledW.Code != http.StatusOK {
		t.Fatalf("expected status 200 for cancelled filter, got %d: %s", cancelledW.Code, cancelledW.Body.String())
	}
	var cancelledResponse SearchEventsResponse
	if err := json.NewDecoder(cancelledW.Body).Decode(&cancelledResponse); err != nil {
		t.Fatalf("failed to decode cancelled response: %v", err)
	}
	if len(cancelledResponse.Events) != 1 || cancelledResponse.Events[0].ID != cancelledEvent.ID {
		t.Fatalf("expected cancelled event %s when filtering cancelled status", cancelledEvent.ID)
	}

	mismatchQuery := url.Values{}
	mismatchQuery.Set("bbox", "-74.2,40.6,-73.8,40.9")
	mismatchQuery.Set("start_date", startDate)
	mismatchQuery.Set("end_date", endDate)
	mismatchQuery.Set("status", "upcoming")
	mismatchQuery.Set("organizer", organizerDID)
	mismatchQuery.Set("scene_id", otherScene.ID)
	mismatchReq := httptest.NewRequest(http.MethodGet, "/search/events?"+mismatchQuery.Encode(), nil)
	mismatchW := httptest.NewRecorder()
	handlers.SearchEvents(mismatchW, mismatchReq)
	if mismatchW.Code != http.StatusOK {
		t.Fatalf("expected status 200 for organizer+scene mismatch, got %d: %s", mismatchW.Code, mismatchW.Body.String())
	}
	var mismatchResponse SearchEventsResponse
	if err := json.NewDecoder(mismatchW.Body).Decode(&mismatchResponse); err != nil {
		t.Fatalf("failed to decode mismatch response: %v", err)
	}
	if len(mismatchResponse.Events) != 0 {
		t.Fatalf("expected 0 events when scene_id is outside organizer scope, got %d", len(mismatchResponse.Events))
	}
}

func TestSearchEvents_BatchSceneFetchFallbackToGetByID(t *testing.T) {
	eventRepo := scene.NewInMemoryEventRepository()
	sceneRepo := &batchErrorSceneRepo{InMemorySceneRepository: scene.NewInMemorySceneRepository()}
	auditRepo := audit.NewInMemoryRepository()
	rsvpRepo := scene.NewInMemoryRSVPRepository()
	streamRepo := stream.NewInMemorySessionRepository()
	handlers := NewEventHandlers(eventRepo, sceneRepo, auditRepo, rsvpRepo, streamRepo, nil)

	baseTime := time.Now().Add(24 * time.Hour)
	parentScene := &scene.Scene{
		ID:           uuid.New().String(),
		Name:         "Fallback Scene",
		OwnerDID:     "did:plc:fallback",
		AllowPrecise: true,
	}
	if err := sceneRepo.Insert(parentScene); err != nil {
		t.Fatalf("failed to insert scene: %v", err)
	}
	event := &scene.Event{
		ID:            uuid.New().String(),
		SceneID:       parentScene.ID,
		Title:         "Fallback Event",
		AllowPrecise:  true,
		PrecisePoint:  &scene.Point{Lat: 40.7128, Lng: -74.0060},
		CoarseGeohash: "dr5regw",
		Status:        "scheduled",
		StartsAt:      baseTime.Add(1 * time.Hour),
		CreatedAt:     &baseTime,
		UpdatedAt:     &baseTime,
	}
	if err := eventRepo.Insert(event); err != nil {
		t.Fatalf("failed to insert event: %v", err)
	}

	query := url.Values{}
	query.Set("bbox", "-74.2,40.6,-73.8,40.9")
	query.Set("from", baseTime.Add(-1*time.Hour).Format(time.RFC3339))
	query.Set("to", baseTime.Add(3*time.Hour).Format(time.RFC3339))
	req := httptest.NewRequest(http.MethodGet, "/search/events?"+query.Encode(), nil)
	w := httptest.NewRecorder()
	handlers.SearchEvents(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}
	if sceneRepo.batchCalls == 0 {
		t.Fatal("expected batch scene fetch to be attempted")
	}
	if sceneRepo.getByIDCalls == 0 {
		t.Fatal("expected fallback GetByID calls after batch fetch error")
	}

	var response SearchEventsResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(response.Events) != 1 || response.Events[0].Scene == nil || response.Events[0].Scene.ID != parentScene.ID {
		t.Fatalf("expected scene payload to be populated via fallback, got %+v", response.Events)
	}
}
