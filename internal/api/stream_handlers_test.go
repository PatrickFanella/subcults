package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/onnwee/subcults/internal/audit"
	"github.com/onnwee/subcults/internal/middleware"
	"github.com/onnwee/subcults/internal/scene"
	"github.com/onnwee/subcults/internal/stream"
)

// Helper functions

func ptrString(s string) *string {
	return &s
}

// TestCreateStream_Success_WithSceneID tests successful stream creation with scene_id.
func TestCreateStream_Success_WithSceneID(t *testing.T) {
	streamRepo := stream.NewInMemorySessionRepository()
	sceneRepo := scene.NewInMemorySceneRepository()
	eventRepo := scene.NewInMemoryEventRepository()
	auditRepo := audit.NewInMemoryRepository()
	handlers := NewStreamHandlers(streamRepo, nil, nil, sceneRepo, eventRepo, auditRepo, nil, nil, nil)

	// Create a scene first
	testScene := &scene.Scene{
		ID:            uuid.New().String(),
		Name:          "Test Scene",
		OwnerDID:      "did:plc:test123",
		CoarseGeohash: "dr5regw",
		CreatedAt:     &time.Time{},
	}
	if err := sceneRepo.Insert(testScene); err != nil {
		t.Fatalf("failed to insert scene: %v", err)
	}

	reqBody := CreateStreamRequest{
		SceneID: ptrString(testScene.ID),
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/streams", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	ctx := middleware.SetUserDID(req.Context(), "did:plc:test123")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	handlers.CreateStream(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d: %s", w.Code, w.Body.String())
	}

	var response StreamSessionResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.ID == "" {
		t.Error("expected non-empty session ID")
	}

	if response.RoomName == "" {
		t.Error("expected non-empty room name")
	}

	if response.Status != "active" {
		t.Errorf("expected status 'active', got %s", response.Status)
	}

	if response.SceneID == nil || *response.SceneID != testScene.ID {
		t.Errorf("expected scene_id %s, got %v", testScene.ID, response.SceneID)
	}
}

// TestCreateStream_Success_WithEventID tests successful stream creation with event_id.
func TestCreateStream_Success_WithEventID(t *testing.T) {
	streamRepo := stream.NewInMemorySessionRepository()
	sceneRepo := scene.NewInMemorySceneRepository()
	eventRepo := scene.NewInMemoryEventRepository()
	auditRepo := audit.NewInMemoryRepository()
	handlers := NewStreamHandlers(streamRepo, nil, nil, sceneRepo, eventRepo, auditRepo, nil, nil, nil)

	// Create a scene and event
	testScene := &scene.Scene{
		ID:            uuid.New().String(),
		Name:          "Test Scene",
		OwnerDID:      "did:plc:test123",
		CoarseGeohash: "dr5regw",
		CreatedAt:     &time.Time{},
	}
	if err := sceneRepo.Insert(testScene); err != nil {
		t.Fatalf("failed to insert scene: %v", err)
	}

	startsAt := time.Now().Add(24 * time.Hour)
	testEvent := &scene.Event{
		ID:            uuid.New().String(),
		SceneID:       testScene.ID,
		Title:         "Test Event",
		CoarseGeohash: "dr5regw",
		StartsAt:      startsAt,
		Status:        "scheduled",
	}
	if err := eventRepo.Insert(testEvent); err != nil {
		t.Fatalf("failed to insert event: %v", err)
	}

	reqBody := CreateStreamRequest{
		EventID: ptrString(testEvent.ID),
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/streams", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	ctx := middleware.SetUserDID(req.Context(), "did:plc:test123")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	handlers.CreateStream(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d: %s", w.Code, w.Body.String())
	}

	var response StreamSessionResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.EventID == nil || *response.EventID != testEvent.ID {
		t.Errorf("expected event_id %s, got %v", testEvent.ID, response.EventID)
	}
}

// TestCreateStream_NoSceneOrEvent tests rejection when neither scene_id nor event_id provided.
func TestCreateStream_NoSceneOrEvent(t *testing.T) {
	streamRepo := stream.NewInMemorySessionRepository()
	sceneRepo := scene.NewInMemorySceneRepository()
	eventRepo := scene.NewInMemoryEventRepository()
	auditRepo := audit.NewInMemoryRepository()
	handlers := NewStreamHandlers(streamRepo, nil, nil, sceneRepo, eventRepo, auditRepo, nil, nil, nil)

	reqBody := CreateStreamRequest{}

	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/streams", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	ctx := middleware.SetUserDID(req.Context(), "did:plc:test123")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	handlers.CreateStream(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

// TestCreateStream_BothSceneAndEvent tests rejection when both scene_id and event_id are provided.
func TestCreateStream_BothSceneAndEvent(t *testing.T) {
	streamRepo := stream.NewInMemorySessionRepository()
	sceneRepo := scene.NewInMemorySceneRepository()
	eventRepo := scene.NewInMemoryEventRepository()
	auditRepo := audit.NewInMemoryRepository()
	handlers := NewStreamHandlers(streamRepo, nil, nil, sceneRepo, eventRepo, auditRepo, nil, nil, nil)

	// Create a scene and event
	testScene := &scene.Scene{
		ID:            uuid.New().String(),
		Name:          "Test Scene",
		OwnerDID:      "did:plc:test123",
		CoarseGeohash: "dr5regw",
		CreatedAt:     &time.Time{},
	}
	if err := sceneRepo.Insert(testScene); err != nil {
		t.Fatalf("failed to insert scene: %v", err)
	}

	startsAt := time.Now().Add(24 * time.Hour)
	testEvent := &scene.Event{
		ID:            uuid.New().String(),
		SceneID:       testScene.ID,
		Title:         "Test Event",
		CoarseGeohash: "dr5regw",
		StartsAt:      startsAt,
		Status:        "scheduled",
	}
	if err := eventRepo.Insert(testEvent); err != nil {
		t.Fatalf("failed to insert event: %v", err)
	}

	// Provide both scene_id and event_id
	reqBody := CreateStreamRequest{
		SceneID: ptrString(testScene.ID),
		EventID: ptrString(testEvent.ID),
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/streams", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	ctx := middleware.SetUserDID(req.Context(), "did:plc:test123")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	handlers.CreateStream(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d: %s", w.Code, w.Body.String())
	}

	var response ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Error.Code != ErrCodeValidation {
		t.Errorf("expected error code %s, got %s", ErrCodeValidation, response.Error.Code)
	}
}

// TestCreateStream_Forbidden_NotSceneOwner tests permission check for scene ownership.
func TestCreateStream_Forbidden_NotSceneOwner(t *testing.T) {
	streamRepo := stream.NewInMemorySessionRepository()
	sceneRepo := scene.NewInMemorySceneRepository()
	eventRepo := scene.NewInMemoryEventRepository()
	auditRepo := audit.NewInMemoryRepository()
	handlers := NewStreamHandlers(streamRepo, nil, nil, sceneRepo, eventRepo, auditRepo, nil, nil, nil)

	// Create a scene owned by someone else
	testScene := &scene.Scene{
		ID:            uuid.New().String(),
		Name:          "Test Scene",
		OwnerDID:      "did:plc:owner123",
		CoarseGeohash: "dr5regw",
		CreatedAt:     &time.Time{},
	}
	if err := sceneRepo.Insert(testScene); err != nil {
		t.Fatalf("failed to insert scene: %v", err)
	}

	reqBody := CreateStreamRequest{
		SceneID: ptrString(testScene.ID),
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/streams", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	// Different user trying to create stream
	ctx := middleware.SetUserDID(req.Context(), "did:plc:different456")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	handlers.CreateStream(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected status 403, got %d: %s", w.Code, w.Body.String())
	}
}

// TestCreateStream_Forbidden_NotEventHost tests permission check for event host.
func TestCreateStream_Forbidden_NotEventHost(t *testing.T) {
	streamRepo := stream.NewInMemorySessionRepository()
	sceneRepo := scene.NewInMemorySceneRepository()
	eventRepo := scene.NewInMemoryEventRepository()
	auditRepo := audit.NewInMemoryRepository()
	handlers := NewStreamHandlers(streamRepo, nil, nil, sceneRepo, eventRepo, auditRepo, nil, nil, nil)

	// Create a scene and event owned by someone else
	testScene := &scene.Scene{
		ID:            uuid.New().String(),
		Name:          "Test Scene",
		OwnerDID:      "did:plc:owner123",
		CoarseGeohash: "dr5regw",
		CreatedAt:     &time.Time{},
	}
	if err := sceneRepo.Insert(testScene); err != nil {
		t.Fatalf("failed to insert scene: %v", err)
	}

	startsAt := time.Now().Add(24 * time.Hour)
	testEvent := &scene.Event{
		ID:            uuid.New().String(),
		SceneID:       testScene.ID,
		Title:         "Test Event",
		CoarseGeohash: "dr5regw",
		StartsAt:      startsAt,
		Status:        "scheduled",
	}
	if err := eventRepo.Insert(testEvent); err != nil {
		t.Fatalf("failed to insert event: %v", err)
	}

	reqBody := CreateStreamRequest{
		EventID: ptrString(testEvent.ID),
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/streams", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	// Different user trying to create stream
	ctx := middleware.SetUserDID(req.Context(), "did:plc:different456")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	handlers.CreateStream(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected status 403, got %d: %s", w.Code, w.Body.String())
	}
}

// TestEndStream_Success tests successful stream ending.
func TestEndStream_Success(t *testing.T) {
	streamRepo := stream.NewInMemorySessionRepository()
	sceneRepo := scene.NewInMemorySceneRepository()
	eventRepo := scene.NewInMemoryEventRepository()
	auditRepo := audit.NewInMemoryRepository()
	handlers := NewStreamHandlers(streamRepo, nil, nil, sceneRepo, eventRepo, auditRepo, nil, nil, nil)

	// Create a stream session
	sceneID := "scene-123"
	hostDID := "did:plc:host456"
	sessionID, _, err := streamRepo.CreateStreamSession(&sceneID, nil, hostDID)
	if err != nil {
		t.Fatalf("failed to create stream session: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/streams/"+sessionID+"/end", nil)
	ctx := middleware.SetUserDID(req.Context(), hostDID)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	handlers.EndStream(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var response StreamSessionResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Status != "ended" {
		t.Errorf("expected status 'ended', got %s", response.Status)
	}

	// Verify session was ended in repo
	session, err := streamRepo.GetByID(sessionID)
	if err != nil {
		t.Fatalf("failed to get session: %v", err)
	}
	if session.EndedAt == nil {
		t.Error("expected session to be ended")
	}
}

// TestEndStream_Forbidden_NotHost tests permission check for stream host.
func TestEndStream_Forbidden_NotHost(t *testing.T) {
	streamRepo := stream.NewInMemorySessionRepository()
	sceneRepo := scene.NewInMemorySceneRepository()
	eventRepo := scene.NewInMemoryEventRepository()
	auditRepo := audit.NewInMemoryRepository()
	handlers := NewStreamHandlers(streamRepo, nil, nil, sceneRepo, eventRepo, auditRepo, nil, nil, nil)

	// Create a stream session
	sceneID := "scene-123"
	hostDID := "did:plc:host456"
	sessionID, _, err := streamRepo.CreateStreamSession(&sceneID, nil, hostDID)
	if err != nil {
		t.Fatalf("failed to create stream session: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/streams/"+sessionID+"/end", nil)
	// Different user trying to end stream
	ctx := middleware.SetUserDID(req.Context(), "did:plc:different789")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	handlers.EndStream(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected status 403, got %d: %s", w.Code, w.Body.String())
	}

	// Verify session was NOT ended in repo
	session, err := streamRepo.GetByID(sessionID)
	if err != nil {
		t.Fatalf("failed to get session: %v", err)
	}
	if session.EndedAt != nil {
		t.Error("expected session to still be active")
	}
}

// TestEndStream_NotFound tests handling of non-existent stream.
func TestEndStream_NotFound(t *testing.T) {
	streamRepo := stream.NewInMemorySessionRepository()
	sceneRepo := scene.NewInMemorySceneRepository()
	eventRepo := scene.NewInMemoryEventRepository()
	auditRepo := audit.NewInMemoryRepository()
	handlers := NewStreamHandlers(streamRepo, nil, nil, sceneRepo, eventRepo, auditRepo, nil, nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/streams/nonexistent-id/end", nil)
	ctx := middleware.SetUserDID(req.Context(), "did:plc:test123")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	handlers.EndStream(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d: %s", w.Code, w.Body.String())
	}
}

// TestEndStream_Idempotent tests that ending an already-ended stream succeeds.
func TestEndStream_Idempotent(t *testing.T) {
	streamRepo := stream.NewInMemorySessionRepository()
	sceneRepo := scene.NewInMemorySceneRepository()
	eventRepo := scene.NewInMemoryEventRepository()
	auditRepo := audit.NewInMemoryRepository()
	handlers := NewStreamHandlers(streamRepo, nil, nil, sceneRepo, eventRepo, auditRepo, nil, nil, nil)

	// Create and end a stream session
	sceneID := "scene-123"
	hostDID := "did:plc:host456"
	sessionID, _, err := streamRepo.CreateStreamSession(&sceneID, nil, hostDID)
	if err != nil {
		t.Fatalf("failed to create stream session: %v", err)
	}

	// End it first time
	req1 := httptest.NewRequest(http.MethodPost, "/streams/"+sessionID+"/end", nil)
	ctx1 := middleware.SetUserDID(req1.Context(), hostDID)
	req1 = req1.WithContext(ctx1)
	w1 := httptest.NewRecorder()
	handlers.EndStream(w1, req1)

	if w1.Code != http.StatusOK {
		t.Errorf("expected status 200 on first end, got %d", w1.Code)
	}

	// End it again (idempotent)
	req2 := httptest.NewRequest(http.MethodPost, "/streams/"+sessionID+"/end", nil)
	ctx2 := middleware.SetUserDID(req2.Context(), hostDID)
	req2 = req2.WithContext(ctx2)
	w2 := httptest.NewRecorder()
	handlers.EndStream(w2, req2)

	if w2.Code != http.StatusOK {
		t.Errorf("expected status 200 on second end (idempotent), got %d", w2.Code)
	}
}

// TestJoinStream_Success tests successful join event recording.
func TestJoinStream_Success(t *testing.T) {
	streamRepo := stream.NewInMemorySessionRepository()
	sceneRepo := scene.NewInMemorySceneRepository()
	eventRepo := scene.NewInMemoryEventRepository()
	auditRepo := audit.NewInMemoryRepository()
	streamMetrics := stream.NewMetrics()
	handlers := NewStreamHandlers(streamRepo, nil, nil, sceneRepo, eventRepo, auditRepo, streamMetrics, nil, nil)

	// Create a scene and stream session first
	testScene := &scene.Scene{
		ID:            uuid.New().String(),
		Name:          "Test Scene",
		OwnerDID:      "did:plc:test123",
		CoarseGeohash: "dr5regw",
		CreatedAt:     &time.Time{},
	}
	if err := sceneRepo.Insert(testScene); err != nil {
		t.Fatalf("failed to insert scene: %v", err)
	}

	streamID, _, err := streamRepo.CreateStreamSession(ptrString(testScene.ID), nil, testScene.OwnerDID)
	if err != nil {
		t.Fatalf("failed to create stream: %v", err)
	}

	// Create join request with timing
	tokenTime := time.Now().Add(-2 * time.Second).Format(time.RFC3339)
	reqBody := map[string]interface{}{
		"token_issued_at": tokenTime,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/streams/"+streamID+"/join", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	// Add auth context
	ctx := middleware.SetUserDID(req.Context(), testScene.OwnerDID)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handlers.JoinStream(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	// Verify response
	var resp map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp["status"] != "joined" {
		t.Errorf("expected status 'joined', got %v", resp["status"])
	}

	if resp["join_count"] != float64(1) {
		t.Errorf("expected join_count 1, got %v", resp["join_count"])
	}

	// Verify database was updated
	session, err := streamRepo.GetByID(streamID)
	if err != nil {
		t.Fatalf("failed to get session: %v", err)
	}
	if session.JoinCount != 1 {
		t.Errorf("expected JoinCount 1, got %d", session.JoinCount)
	}
}

// TestJoinStream_NotFound tests join request for non-existent stream.
func TestJoinStream_NotFound(t *testing.T) {
	streamRepo := stream.NewInMemorySessionRepository()
	sceneRepo := scene.NewInMemorySceneRepository()
	eventRepo := scene.NewInMemoryEventRepository()
	auditRepo := audit.NewInMemoryRepository()
	streamMetrics := stream.NewMetrics()
	handlers := NewStreamHandlers(streamRepo, nil, nil, sceneRepo, eventRepo, auditRepo, streamMetrics, nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/streams/nonexistent-id/join", nil)
	ctx := middleware.SetUserDID(req.Context(), "did:plc:test123")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handlers.JoinStream(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rr.Code)
	}
}

// TestJoinStream_Unauthorized tests join request without auth.
func TestJoinStream_Unauthorized(t *testing.T) {
	streamRepo := stream.NewInMemorySessionRepository()
	sceneRepo := scene.NewInMemorySceneRepository()
	eventRepo := scene.NewInMemoryEventRepository()
	auditRepo := audit.NewInMemoryRepository()
	streamMetrics := stream.NewMetrics()
	handlers := NewStreamHandlers(streamRepo, nil, nil, sceneRepo, eventRepo, auditRepo, streamMetrics, nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/streams/some-id/join", nil)
	// No auth context

	rr := httptest.NewRecorder()
	handlers.JoinStream(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", rr.Code)
	}
}

// TestLeaveStream_Success tests successful leave event recording.
func TestLeaveStream_Success(t *testing.T) {
	streamRepo := stream.NewInMemorySessionRepository()
	sceneRepo := scene.NewInMemorySceneRepository()
	eventRepo := scene.NewInMemoryEventRepository()
	auditRepo := audit.NewInMemoryRepository()
	streamMetrics := stream.NewMetrics()
	handlers := NewStreamHandlers(streamRepo, nil, nil, sceneRepo, eventRepo, auditRepo, streamMetrics, nil, nil)

	// Create a scene and stream session first
	testScene := &scene.Scene{
		ID:            uuid.New().String(),
		Name:          "Test Scene",
		OwnerDID:      "did:plc:test123",
		CoarseGeohash: "dr5regw",
		CreatedAt:     &time.Time{},
	}
	if err := sceneRepo.Insert(testScene); err != nil {
		t.Fatalf("failed to insert scene: %v", err)
	}

	streamID, _, err := streamRepo.CreateStreamSession(ptrString(testScene.ID), nil, testScene.OwnerDID)
	if err != nil {
		t.Fatalf("failed to create stream: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/streams/"+streamID+"/leave", nil)

	// Add auth context
	ctx := middleware.SetUserDID(req.Context(), testScene.OwnerDID)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handlers.LeaveStream(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	// Verify response
	var resp map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp["status"] != "left" {
		t.Errorf("expected status 'left', got %v", resp["status"])
	}

	if resp["leave_count"] != float64(1) {
		t.Errorf("expected leave_count 1, got %v", resp["leave_count"])
	}

	// Verify database was updated
	session, err := streamRepo.GetByID(streamID)
	if err != nil {
		t.Fatalf("failed to get session: %v", err)
	}
	if session.LeaveCount != 1 {
		t.Errorf("expected LeaveCount 1, got %d", session.LeaveCount)
	}
}

// TestLeaveStream_NotFound tests leave request for non-existent stream.
func TestLeaveStream_NotFound(t *testing.T) {
	streamRepo := stream.NewInMemorySessionRepository()
	sceneRepo := scene.NewInMemorySceneRepository()
	eventRepo := scene.NewInMemoryEventRepository()
	auditRepo := audit.NewInMemoryRepository()
	streamMetrics := stream.NewMetrics()
	handlers := NewStreamHandlers(streamRepo, nil, nil, sceneRepo, eventRepo, auditRepo, streamMetrics, nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/streams/nonexistent-id/leave", nil)
	ctx := middleware.SetUserDID(req.Context(), "did:plc:test123")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handlers.LeaveStream(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rr.Code)
	}
}

// TestJoinLeave_Multiple tests multiple join/leave events increment correctly.
func TestJoinLeave_Multiple(t *testing.T) {
	streamRepo := stream.NewInMemorySessionRepository()
	sceneRepo := scene.NewInMemorySceneRepository()
	eventRepo := scene.NewInMemoryEventRepository()
	auditRepo := audit.NewInMemoryRepository()
	streamMetrics := stream.NewMetrics()
	handlers := NewStreamHandlers(streamRepo, nil, nil, sceneRepo, eventRepo, auditRepo, streamMetrics, nil, nil)

	// Create a scene and stream session first
	testScene := &scene.Scene{
		ID:            uuid.New().String(),
		Name:          "Test Scene",
		OwnerDID:      "did:plc:test123",
		CoarseGeohash: "dr5regw",
		CreatedAt:     &time.Time{},
	}
	if err := sceneRepo.Insert(testScene); err != nil {
		t.Fatalf("failed to insert scene: %v", err)
	}

	streamID, _, err := streamRepo.CreateStreamSession(ptrString(testScene.ID), nil, testScene.OwnerDID)
	if err != nil {
		t.Fatalf("failed to create stream: %v", err)
	}

	// Record 3 joins
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodPost, "/streams/"+streamID+"/join", nil)
		ctx := middleware.SetUserDID(req.Context(), testScene.OwnerDID)
		req = req.WithContext(ctx)
		rr := httptest.NewRecorder()
		handlers.JoinStream(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("join %d failed: %d", i+1, rr.Code)
		}
	}

	// Record 2 leaves
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodPost, "/streams/"+streamID+"/leave", nil)
		ctx := middleware.SetUserDID(req.Context(), testScene.OwnerDID)
		req = req.WithContext(ctx)
		rr := httptest.NewRecorder()
		handlers.LeaveStream(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("leave %d failed: %d", i+1, rr.Code)
		}
	}

	// Verify counts
	session, err := streamRepo.GetByID(streamID)
	if err != nil {
		t.Fatalf("failed to get session: %v", err)
	}
	if session.JoinCount != 3 {
		t.Errorf("expected JoinCount 3, got %d", session.JoinCount)
	}
	if session.LeaveCount != 2 {
		t.Errorf("expected LeaveCount 2, got %d", session.LeaveCount)
	}
}

// TestJoinStream_WithNilMetrics tests that join/leave handlers work correctly when metrics collection is disabled.
func TestJoinStream_WithNilMetrics(t *testing.T) {
	streamRepo := stream.NewInMemorySessionRepository()
	sceneRepo := scene.NewInMemorySceneRepository()
	eventRepo := scene.NewInMemoryEventRepository()
	auditRepo := audit.NewInMemoryRepository()
	// Pass nil for metrics to test the nil check path
	handlers := NewStreamHandlers(streamRepo, nil, nil, sceneRepo, eventRepo, auditRepo, nil, nil, nil)

	// Create a scene and stream session first
	testScene := &scene.Scene{
		ID:            uuid.New().String(),
		Name:          "Test Scene",
		OwnerDID:      "did:plc:test123",
		CoarseGeohash: "dr5regw",
		CreatedAt:     &time.Time{},
	}
	if err := sceneRepo.Insert(testScene); err != nil {
		t.Fatalf("failed to insert scene: %v", err)
	}

	streamID, _, err := streamRepo.CreateStreamSession(ptrString(testScene.ID), nil, testScene.OwnerDID)
	if err != nil {
		t.Fatalf("failed to create stream: %v", err)
	}

	// Test join with nil metrics
	joinReq := httptest.NewRequest(http.MethodPost, "/streams/"+streamID+"/join", nil)
	joinCtx := middleware.SetUserDID(joinReq.Context(), testScene.OwnerDID)
	joinReq = joinReq.WithContext(joinCtx)
	joinRR := httptest.NewRecorder()
	handlers.JoinStream(joinRR, joinReq)

	if joinRR.Code != http.StatusOK {
		t.Errorf("join failed with nil metrics: expected status 200, got %d: %s", joinRR.Code, joinRR.Body.String())
	}

	// Verify database was updated despite nil metrics
	session, err := streamRepo.GetByID(streamID)
	if err != nil {
		t.Fatalf("failed to get session: %v", err)
	}
	if session.JoinCount != 1 {
		t.Errorf("expected JoinCount 1, got %d", session.JoinCount)
	}

	// Test leave with nil metrics
	leaveReq := httptest.NewRequest(http.MethodPost, "/streams/"+streamID+"/leave", nil)
	leaveCtx := middleware.SetUserDID(leaveReq.Context(), testScene.OwnerDID)
	leaveReq = leaveReq.WithContext(leaveCtx)
	leaveRR := httptest.NewRecorder()
	handlers.LeaveStream(leaveRR, leaveReq)

	if leaveRR.Code != http.StatusOK {
		t.Errorf("leave failed with nil metrics: expected status 200, got %d: %s", leaveRR.Code, leaveRR.Body.String())
	}

	// Verify leave count was updated
	session, err = streamRepo.GetByID(streamID)
	if err != nil {
		t.Fatalf("failed to get session: %v", err)
	}
	if session.LeaveCount != 1 {
		t.Errorf("expected LeaveCount 1, got %d", session.LeaveCount)
	}
}

// TestGetStream_Success tests successful stream retrieval.
func TestGetStream_Success(t *testing.T) {
	streamRepo := stream.NewInMemorySessionRepository()
	sceneRepo := scene.NewInMemorySceneRepository()
	eventRepo := scene.NewInMemoryEventRepository()
	auditRepo := audit.NewInMemoryRepository()
	handlers := NewStreamHandlers(streamRepo, nil, nil, sceneRepo, eventRepo, auditRepo, nil, nil, nil)

	// Create a scene first
	testScene := &scene.Scene{
		ID:            uuid.New().String(),
		Name:          "Test Scene",
		OwnerDID:      "did:plc:test123",
		CoarseGeohash: "dr5regw",
		CreatedAt:     &time.Time{},
	}
	if err := sceneRepo.Insert(testScene); err != nil {
		t.Fatalf("failed to insert scene: %v", err)
	}

	// Create a stream session
	streamID, roomName, err := streamRepo.CreateStreamSession(ptrString(testScene.ID), nil, "did:plc:test123")
	if err != nil {
		t.Fatalf("failed to create stream: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/streams/"+streamID, nil)
	ctx := middleware.SetUserDID(req.Context(), "did:plc:test123")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	handlers.GetStream(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var response StreamSessionResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.ID != streamID {
		t.Errorf("expected stream ID %s, got %s", streamID, response.ID)
	}

	if response.RoomName != roomName {
		t.Errorf("expected room name %s, got %s", roomName, response.RoomName)
	}

	if response.Status != "active" {
		t.Errorf("expected status 'active', got %s", response.Status)
	}
}

// TestGetStream_NotFound tests retrieval of non-existent stream.
func TestGetStream_NotFound(t *testing.T) {
	streamRepo := stream.NewInMemorySessionRepository()
	sceneRepo := scene.NewInMemorySceneRepository()
	eventRepo := scene.NewInMemoryEventRepository()
	auditRepo := audit.NewInMemoryRepository()
	handlers := NewStreamHandlers(streamRepo, nil, nil, sceneRepo, eventRepo, auditRepo, nil, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/streams/"+uuid.New().String(), nil)
	ctx := middleware.SetUserDID(req.Context(), "did:plc:test123")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	handlers.GetStream(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d: %s", w.Code, w.Body.String())
	}
}

// TestGetStream_Unauthorized tests stream retrieval without authentication.
func TestGetStream_Unauthorized(t *testing.T) {
	streamRepo := stream.NewInMemorySessionRepository()
	sceneRepo := scene.NewInMemorySceneRepository()
	eventRepo := scene.NewInMemoryEventRepository()
	auditRepo := audit.NewInMemoryRepository()
	handlers := NewStreamHandlers(streamRepo, nil, nil, sceneRepo, eventRepo, auditRepo, nil, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/streams/"+uuid.New().String(), nil)
	w := httptest.NewRecorder()

	handlers.GetStream(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d: %s", w.Code, w.Body.String())
	}
}

// TestUpdateStream_Success tests successful stream metadata update.
func TestUpdateStream_Success(t *testing.T) {
	streamRepo := stream.NewInMemorySessionRepository()
	sceneRepo := scene.NewInMemorySceneRepository()
	eventRepo := scene.NewInMemoryEventRepository()
	auditRepo := audit.NewInMemoryRepository()
	handlers := NewStreamHandlers(streamRepo, nil, nil, sceneRepo, eventRepo, auditRepo, nil, nil, nil)

	// Create a scene first
	testScene := &scene.Scene{
		ID:            uuid.New().String(),
		Name:          "Test Scene",
		OwnerDID:      "did:plc:test123",
		CoarseGeohash: "dr5regw",
		CreatedAt:     &time.Time{},
	}
	if err := sceneRepo.Insert(testScene); err != nil {
		t.Fatalf("failed to insert scene: %v", err)
	}

	// Create a stream session
	streamID, _, err := streamRepo.CreateStreamSession(ptrString(testScene.ID), nil, "did:plc:test123")
	if err != nil {
		t.Fatalf("failed to create stream: %v", err)
	}

	reqBody := UpdateStreamRequest{
		Metadata: map[string]interface{}{
			"title":       "My Stream",
			"description": "A great stream",
		},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPatch, "/streams/"+streamID, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	ctx := middleware.SetUserDID(req.Context(), "did:plc:test123")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	handlers.UpdateStream(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var response StreamSessionResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.ID != streamID {
		t.Errorf("expected stream ID %s, got %s", streamID, response.ID)
	}
}

// TestUpdateStream_Forbidden tests metadata update by non-host.
func TestUpdateStream_Forbidden(t *testing.T) {
	streamRepo := stream.NewInMemorySessionRepository()
	sceneRepo := scene.NewInMemorySceneRepository()
	eventRepo := scene.NewInMemoryEventRepository()
	auditRepo := audit.NewInMemoryRepository()
	handlers := NewStreamHandlers(streamRepo, nil, nil, sceneRepo, eventRepo, auditRepo, nil, nil, nil)

	// Create a scene first
	testScene := &scene.Scene{
		ID:            uuid.New().String(),
		Name:          "Test Scene",
		OwnerDID:      "did:plc:test123",
		CoarseGeohash: "dr5regw",
		CreatedAt:     &time.Time{},
	}
	if err := sceneRepo.Insert(testScene); err != nil {
		t.Fatalf("failed to insert scene: %v", err)
	}

	// Create a stream session
	streamID, _, err := streamRepo.CreateStreamSession(ptrString(testScene.ID), nil, "did:plc:test123")
	if err != nil {
		t.Fatalf("failed to create stream: %v", err)
	}

	reqBody := UpdateStreamRequest{
		Metadata: map[string]interface{}{
			"title": "Hacked Stream",
		},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPatch, "/streams/"+streamID, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	ctx := middleware.SetUserDID(req.Context(), "did:plc:different456") // Different user
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	handlers.UpdateStream(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected status 403, got %d: %s", w.Code, w.Body.String())
	}
}

// TestCreateStream_ConcurrentStartPrevention_Scene tests duplicate prevention for scenes.
func TestCreateStream_ConcurrentStartPrevention_Scene(t *testing.T) {
	streamRepo := stream.NewInMemorySessionRepository()
	sceneRepo := scene.NewInMemorySceneRepository()
	eventRepo := scene.NewInMemoryEventRepository()
	auditRepo := audit.NewInMemoryRepository()
	handlers := NewStreamHandlers(streamRepo, nil, nil, sceneRepo, eventRepo, auditRepo, nil, nil, nil)

	// Create a scene first
	testScene := &scene.Scene{
		ID:            uuid.New().String(),
		Name:          "Test Scene",
		OwnerDID:      "did:plc:test123",
		CoarseGeohash: "dr5regw",
		CreatedAt:     &time.Time{},
	}
	if err := sceneRepo.Insert(testScene); err != nil {
		t.Fatalf("failed to insert scene: %v", err)
	}

	// Create first stream
	_, _, err := streamRepo.CreateStreamSession(ptrString(testScene.ID), nil, "did:plc:test123")
	if err != nil {
		t.Fatalf("failed to create first stream: %v", err)
	}

	// Attempt to create second stream for same scene
	reqBody := CreateStreamRequest{
		SceneID: ptrString(testScene.ID),
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/streams", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	ctx := middleware.SetUserDID(req.Context(), "did:plc:test123")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	handlers.CreateStream(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("expected status 409, got %d: %s", w.Code, w.Body.String())
	}
}

// TestCreateStream_ConcurrentStartPrevention_Event tests duplicate prevention for events.
func TestCreateStream_ConcurrentStartPrevention_Event(t *testing.T) {
	streamRepo := stream.NewInMemorySessionRepository()
	sceneRepo := scene.NewInMemorySceneRepository()
	eventRepo := scene.NewInMemoryEventRepository()
	auditRepo := audit.NewInMemoryRepository()
	handlers := NewStreamHandlers(streamRepo, nil, nil, sceneRepo, eventRepo, auditRepo, nil, nil, nil)

	// Create a scene and event
	testScene := &scene.Scene{
		ID:            uuid.New().String(),
		Name:          "Test Scene",
		OwnerDID:      "did:plc:test123",
		CoarseGeohash: "dr5regw",
		CreatedAt:     &time.Time{},
	}
	if err := sceneRepo.Insert(testScene); err != nil {
		t.Fatalf("failed to insert scene: %v", err)
	}

	startsAt := time.Now().Add(24 * time.Hour)
	testEvent := &scene.Event{
		ID:            uuid.New().String(),
		SceneID:       testScene.ID,
		Title:         "Test Event",
		CoarseGeohash: "dr5regw",
		StartsAt:      startsAt,
	}
	if err := eventRepo.Insert(testEvent); err != nil {
		t.Fatalf("failed to insert event: %v", err)
	}

	// Create first stream
	_, _, err := streamRepo.CreateStreamSession(nil, ptrString(testEvent.ID), "did:plc:test123")
	if err != nil {
		t.Fatalf("failed to create first stream: %v", err)
	}

	// Attempt to create second stream for same event
	reqBody := CreateStreamRequest{
		EventID: ptrString(testEvent.ID),
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/streams", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	ctx := middleware.SetUserDID(req.Context(), "did:plc:test123")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	handlers.CreateStream(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("expected status 409, got %d: %s", w.Code, w.Body.String())
	}
}
