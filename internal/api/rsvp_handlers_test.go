package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/onnwee/subcults/internal/middleware"
	"github.com/onnwee/subcults/internal/scene"
)

func TestCreateOrUpdateRSVP_Success(t *testing.T) {
	rsvpRepo := scene.NewInMemoryRSVPRepository()
	eventRepo := scene.NewInMemoryEventRepository()
	handlers := NewRSVPHandlers(rsvpRepo, eventRepo)

	// Create a future event
	futureTime := time.Now().Add(24 * time.Hour)
	event := &scene.Event{
		ID:            "event-1",
		SceneID:       "scene-1",
		Title:         "Test Event",
		CoarseGeohash: "dr5regw",
		StartsAt:      futureTime,
	}
	if err := eventRepo.Insert(event); err != nil {
		t.Fatalf("Failed to insert event: %v", err)
	}

	// Create request
	reqBody := RSVPRequest{Status: "going"}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/events/event-1/rsvp", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	// Add user DID to context (simulating auth middleware)
	ctx := middleware.SetUserDID(req.Context(), "did:plc:user1")
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	// Call handler
	handlers.CreateOrUpdateRSVP(w, req)

	// Verify response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify response doesn't contain user_id (privacy requirement)
	var response RSVPResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}
	if response.Status != "going" {
		t.Errorf("Expected response status 'going', got %s", response.Status)
	}
	if response.EventID != "event-1" {
		t.Errorf("Expected response event_id 'event-1', got %s", response.EventID)
	}
	// Verify user_id is not in the raw JSON response (privacy check)
	if bytes.Contains(w.Body.Bytes(), []byte("user_id")) {
		t.Error("Response should not contain 'user_id' field (privacy violation)")
	}

	// Verify RSVP was created in repository
	stored, err := rsvpRepo.GetByEventAndUser("event-1", "did:plc:user1")
	if err != nil {
		t.Fatalf("Failed to get RSVP: %v", err)
	}
	if stored.Status != "going" {
		t.Errorf("Expected status 'going', got %s", stored.Status)
	}
}

func TestCreateOrUpdateRSVP_UpdateStatus(t *testing.T) {
	rsvpRepo := scene.NewInMemoryRSVPRepository()
	eventRepo := scene.NewInMemoryEventRepository()
	handlers := NewRSVPHandlers(rsvpRepo, eventRepo)

	// Create a future event
	futureTime := time.Now().Add(24 * time.Hour)
	event := &scene.Event{
		ID:            "event-1",
		SceneID:       "scene-1",
		Title:         "Test Event",
		CoarseGeohash: "dr5regw",
		StartsAt:      futureTime,
	}
	if err := eventRepo.Insert(event); err != nil {
		t.Fatalf("Failed to insert event: %v", err)
	}

	// Create initial RSVP with "maybe"
	initialRSVP := &scene.RSVP{
		EventID: "event-1",
		UserID:  "did:plc:user1",
		Status:  "maybe",
	}
	if err := rsvpRepo.Upsert(initialRSVP); err != nil {
		t.Fatalf("Failed to create initial RSVP: %v", err)
	}

	// Update to "going"
	reqBody := RSVPRequest{Status: "going"}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/events/event-1/rsvp", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	ctx := middleware.SetUserDID(req.Context(), "did:plc:user1")
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handlers.CreateOrUpdateRSVP(w, req)

	// Verify response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify status was updated
	stored, err := rsvpRepo.GetByEventAndUser("event-1", "did:plc:user1")
	if err != nil {
		t.Fatalf("Failed to get RSVP: %v", err)
	}
	if stored.Status != "going" {
		t.Errorf("Expected status 'going', got %s", stored.Status)
	}
}

func TestCreateOrUpdateRSVP_Idempotent(t *testing.T) {
	rsvpRepo := scene.NewInMemoryRSVPRepository()
	eventRepo := scene.NewInMemoryEventRepository()
	handlers := NewRSVPHandlers(rsvpRepo, eventRepo)

	// Create a future event
	futureTime := time.Now().Add(24 * time.Hour)
	event := &scene.Event{
		ID:            "event-1",
		SceneID:       "scene-1",
		Title:         "Test Event",
		CoarseGeohash: "dr5regw",
		StartsAt:      futureTime,
	}
	if err := eventRepo.Insert(event); err != nil {
		t.Fatalf("Failed to insert event: %v", err)
	}

	// First RSVP with "going"
	reqBody := RSVPRequest{Status: "going"}
	body, _ := json.Marshal(reqBody)

	// First request
	req1 := httptest.NewRequest("POST", "/events/event-1/rsvp", bytes.NewReader(body))
	req1.Header.Set("Content-Type", "application/json")
	ctx1 := middleware.SetUserDID(req1.Context(), "did:plc:user1")
	req1 = req1.WithContext(ctx1)

	w1 := httptest.NewRecorder()
	handlers.CreateOrUpdateRSVP(w1, req1)

	// Verify first response
	if w1.Code != http.StatusOK {
		t.Errorf("Expected status 200 on first RSVP, got %d: %s", w1.Code, w1.Body.String())
	}

	// Second request with same status (idempotent test)
	body2, _ := json.Marshal(reqBody)
	req2 := httptest.NewRequest("POST", "/events/event-1/rsvp", bytes.NewReader(body2))
	req2.Header.Set("Content-Type", "application/json")
	ctx2 := middleware.SetUserDID(req2.Context(), "did:plc:user1")
	req2 = req2.WithContext(ctx2)

	w2 := httptest.NewRecorder()
	handlers.CreateOrUpdateRSVP(w2, req2)

	// Verify second response is also 200 (idempotent)
	if w2.Code != http.StatusOK {
		t.Errorf("Expected status 200 on duplicate RSVP (idempotent), got %d: %s", w2.Code, w2.Body.String())
	}

	// Verify RSVP still exists with correct status
	stored, err := rsvpRepo.GetByEventAndUser("event-1", "did:plc:user1")
	if err != nil {
		t.Fatalf("Failed to get RSVP: %v", err)
	}
	if stored.Status != "going" {
		t.Errorf("Expected status 'going', got %s", stored.Status)
	}

	// Verify no duplicate was created (should still be just one RSVP)
	counts, err := rsvpRepo.GetCountsByEvent("event-1")
	if err != nil {
		t.Fatalf("Failed to get counts: %v", err)
	}
	if counts.Going != 1 {
		t.Errorf("Expected 1 'going' RSVP, got %d (duplicate may have been created)", counts.Going)
	}
}

func TestCreateOrUpdateRSVP_InvalidStatus(t *testing.T) {
	rsvpRepo := scene.NewInMemoryRSVPRepository()
	eventRepo := scene.NewInMemoryEventRepository()
	handlers := NewRSVPHandlers(rsvpRepo, eventRepo)

	// Create request with invalid status
	reqBody := RSVPRequest{Status: "invalid"}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/events/event-1/rsvp", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	ctx := middleware.SetUserDID(req.Context(), "did:plc:user1")
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handlers.CreateOrUpdateRSVP(w, req)

	// Verify error response
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestCreateOrUpdateRSVP_PastEvent(t *testing.T) {
	rsvpRepo := scene.NewInMemoryRSVPRepository()
	eventRepo := scene.NewInMemoryEventRepository()
	handlers := NewRSVPHandlers(rsvpRepo, eventRepo)

	// Create a past event
	pastTime := time.Now().Add(-24 * time.Hour)
	event := &scene.Event{
		ID:            "event-1",
		SceneID:       "scene-1",
		Title:         "Past Event",
		CoarseGeohash: "dr5regw",
		StartsAt:      pastTime,
	}
	if err := eventRepo.Insert(event); err != nil {
		t.Fatalf("Failed to insert event: %v", err)
	}

	// Try to RSVP to past event
	reqBody := RSVPRequest{Status: "going"}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/events/event-1/rsvp", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	ctx := middleware.SetUserDID(req.Context(), "did:plc:user1")
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handlers.CreateOrUpdateRSVP(w, req)

	// Verify error response
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestCreateOrUpdateRSVP_EventNotFound(t *testing.T) {
	rsvpRepo := scene.NewInMemoryRSVPRepository()
	eventRepo := scene.NewInMemoryEventRepository()
	handlers := NewRSVPHandlers(rsvpRepo, eventRepo)

	// Try to RSVP to non-existent event
	reqBody := RSVPRequest{Status: "going"}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/events/non-existent/rsvp", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	ctx := middleware.SetUserDID(req.Context(), "did:plc:user1")
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handlers.CreateOrUpdateRSVP(w, req)

	// Verify error response
	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestCreateOrUpdateRSVP_Unauthenticated(t *testing.T) {
	rsvpRepo := scene.NewInMemoryRSVPRepository()
	eventRepo := scene.NewInMemoryEventRepository()
	handlers := NewRSVPHandlers(rsvpRepo, eventRepo)

	// Create request without user DID in context
	reqBody := RSVPRequest{Status: "going"}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/events/event-1/rsvp", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	handlers.CreateOrUpdateRSVP(w, req)

	// Verify error response
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestDeleteRSVP_Success(t *testing.T) {
	rsvpRepo := scene.NewInMemoryRSVPRepository()
	eventRepo := scene.NewInMemoryEventRepository()
	handlers := NewRSVPHandlers(rsvpRepo, eventRepo)

	// Create a future event
	futureTime := time.Now().Add(24 * time.Hour)
	event := &scene.Event{
		ID:            "event-1",
		SceneID:       "scene-1",
		Title:         "Test Event",
		CoarseGeohash: "dr5regw",
		StartsAt:      futureTime,
	}
	if err := eventRepo.Insert(event); err != nil {
		t.Fatalf("Failed to insert event: %v", err)
	}

	// Create RSVP
	rsvp := &scene.RSVP{
		EventID: "event-1",
		UserID:  "did:plc:user1",
		Status:  "going",
	}
	if err := rsvpRepo.Upsert(rsvp); err != nil {
		t.Fatalf("Failed to create RSVP: %v", err)
	}

	// Delete RSVP
	req := httptest.NewRequest("DELETE", "/events/event-1/rsvp", nil)
	ctx := middleware.SetUserDID(req.Context(), "did:plc:user1")
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handlers.DeleteRSVP(w, req)

	// Verify response
	if w.Code != http.StatusNoContent {
		t.Errorf("Expected status 204, got %d: %s", w.Code, w.Body.String())
	}

	// Verify RSVP was deleted
	_, err := rsvpRepo.GetByEventAndUser("event-1", "did:plc:user1")
	if err != scene.ErrRSVPNotFound {
		t.Errorf("Expected ErrRSVPNotFound, got %v", err)
	}
}

func TestDeleteRSVP_NotFound(t *testing.T) {
	rsvpRepo := scene.NewInMemoryRSVPRepository()
	eventRepo := scene.NewInMemoryEventRepository()
	handlers := NewRSVPHandlers(rsvpRepo, eventRepo)

	// Create a future event
	futureTime := time.Now().Add(24 * time.Hour)
	event := &scene.Event{
		ID:            "event-1",
		SceneID:       "scene-1",
		Title:         "Test Event",
		CoarseGeohash: "dr5regw",
		StartsAt:      futureTime,
	}
	if err := eventRepo.Insert(event); err != nil {
		t.Fatalf("Failed to insert event: %v", err)
	}

	// Try to delete non-existent RSVP
	req := httptest.NewRequest("DELETE", "/events/event-1/rsvp", nil)
	ctx := middleware.SetUserDID(req.Context(), "did:plc:user1")
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handlers.DeleteRSVP(w, req)

	// Verify error response
	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestDeleteRSVP_PastEvent(t *testing.T) {
	rsvpRepo := scene.NewInMemoryRSVPRepository()
	eventRepo := scene.NewInMemoryEventRepository()
	handlers := NewRSVPHandlers(rsvpRepo, eventRepo)

	// Create a past event
	pastTime := time.Now().Add(-24 * time.Hour)
	event := &scene.Event{
		ID:            "event-1",
		SceneID:       "scene-1",
		Title:         "Past Event",
		CoarseGeohash: "dr5regw",
		StartsAt:      pastTime,
	}
	if err := eventRepo.Insert(event); err != nil {
		t.Fatalf("Failed to insert event: %v", err)
	}

	// Create RSVP
	rsvp := &scene.RSVP{
		EventID: "event-1",
		UserID:  "did:plc:user1",
		Status:  "going",
	}
	if err := rsvpRepo.Upsert(rsvp); err != nil {
		t.Fatalf("Failed to create RSVP: %v", err)
	}

	// Try to delete RSVP for past event
	req := httptest.NewRequest("DELETE", "/events/event-1/rsvp", nil)
	ctx := middleware.SetUserDID(req.Context(), "did:plc:user1")
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handlers.DeleteRSVP(w, req)

	// Verify error response
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestDeleteRSVP_Unauthenticated(t *testing.T) {
	rsvpRepo := scene.NewInMemoryRSVPRepository()
	eventRepo := scene.NewInMemoryEventRepository()
	handlers := NewRSVPHandlers(rsvpRepo, eventRepo)

	// Try to delete without user DID in context
	req := httptest.NewRequest("DELETE", "/events/event-1/rsvp", nil)

	w := httptest.NewRecorder()
	handlers.DeleteRSVP(w, req)

	// Verify error response
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}
