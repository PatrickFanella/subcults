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

// TestGetStreamAnalytics_Success tests successful analytics retrieval.
func TestGetStreamAnalytics_Success(t *testing.T) {
	streamRepo := stream.NewInMemorySessionRepository()
	analyticsRepo := stream.NewInMemoryAnalyticsRepository(streamRepo)
	sceneRepo := scene.NewInMemorySceneRepository()
	eventRepo := scene.NewInMemoryEventRepository()
	auditRepo := audit.NewInMemoryRepository()
	handlers := NewStreamHandlers(streamRepo, nil, analyticsRepo, sceneRepo, eventRepo, auditRepo, nil, nil, nil)

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
	sceneID := testScene.ID
	streamID, _, err := streamRepo.CreateStreamSession(&sceneID, nil, "did:plc:test123")
	if err != nil {
		t.Fatalf("failed to create stream: %v", err)
	}

	// Record some participant events
	geo1 := "abcd"
	_ = analyticsRepo.RecordParticipantEvent(streamID, "did:plc:user1", "join", &geo1)
	time.Sleep(10 * time.Millisecond)
	_ = analyticsRepo.RecordParticipantEvent(streamID, "did:plc:user2", "join", &geo1)
	time.Sleep(10 * time.Millisecond)
	_ = analyticsRepo.RecordParticipantEvent(streamID, "did:plc:user1", "leave", nil)

	// End the stream
	time.Sleep(10 * time.Millisecond)
	if err := streamRepo.EndStreamSession(streamID); err != nil {
		t.Fatalf("failed to end stream: %v", err)
	}

	// Compute analytics
	_, err = analyticsRepo.ComputeAnalytics(streamID)
	if err != nil {
		t.Fatalf("failed to compute analytics: %v", err)
	}

	// Request analytics
	req := httptest.NewRequest(http.MethodGet, "/streams/"+streamID+"/analytics", nil)
	ctx := middleware.SetUserDID(req.Context(), "did:plc:test123")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	handlers.GetStreamAnalytics(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var analytics stream.Analytics
	if err := json.NewDecoder(w.Body).Decode(&analytics); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Verify analytics
	if analytics.StreamSessionID != streamID {
		t.Errorf("expected stream ID %s, got %s", streamID, analytics.StreamSessionID)
	}

	if analytics.PeakConcurrentListeners != 2 {
		t.Errorf("expected peak 2, got %d", analytics.PeakConcurrentListeners)
	}

	if analytics.TotalUniqueParticipants != 2 {
		t.Errorf("expected 2 unique participants, got %d", analytics.TotalUniqueParticipants)
	}

	if analytics.TotalJoinAttempts != 2 {
		t.Errorf("expected 2 join attempts, got %d", analytics.TotalJoinAttempts)
	}

	if analytics.GeographicDistribution["abcd"] != 2 {
		t.Errorf("expected 2 participants from 'abcd', got %d", analytics.GeographicDistribution["abcd"])
	}
}

// TestGetStreamAnalytics_Unauthorized tests analytics retrieval by non-host.
func TestGetStreamAnalytics_Unauthorized(t *testing.T) {
	streamRepo := stream.NewInMemorySessionRepository()
	analyticsRepo := stream.NewInMemoryAnalyticsRepository(streamRepo)
	sceneRepo := scene.NewInMemorySceneRepository()
	eventRepo := scene.NewInMemoryEventRepository()
	auditRepo := audit.NewInMemoryRepository()
	handlers := NewStreamHandlers(streamRepo, nil, analyticsRepo, sceneRepo, eventRepo, auditRepo, nil, nil, nil)

	// Create a scene and stream
	testScene := &scene.Scene{
		ID:            uuid.New().String(),
		Name:          "Test Scene",
		OwnerDID:      "did:plc:owner",
		CoarseGeohash: "dr5regw",
		CreatedAt:     &time.Time{},
	}
	if err := sceneRepo.Insert(testScene); err != nil {
		t.Fatalf("failed to insert scene: %v", err)
	}

	sceneID := testScene.ID
	streamID, _, err := streamRepo.CreateStreamSession(&sceneID, nil, "did:plc:owner")
	if err != nil {
		t.Fatalf("failed to create stream: %v", err)
	}

	// End the stream and compute analytics
	_ = streamRepo.EndStreamSession(streamID)
	_, _ = analyticsRepo.ComputeAnalytics(streamID)

	// Request analytics as different user
	req := httptest.NewRequest(http.MethodGet, "/streams/"+streamID+"/analytics", nil)
	ctx := middleware.SetUserDID(req.Context(), "did:plc:other")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	handlers.GetStreamAnalytics(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected status 403, got %d: %s", w.Code, w.Body.String())
	}
}

// TestGetStreamAnalytics_NotComputedYet tests analytics retrieval before computation.
func TestGetStreamAnalytics_NotComputedYet(t *testing.T) {
	streamRepo := stream.NewInMemorySessionRepository()
	analyticsRepo := stream.NewInMemoryAnalyticsRepository(streamRepo)
	sceneRepo := scene.NewInMemorySceneRepository()
	eventRepo := scene.NewInMemoryEventRepository()
	auditRepo := audit.NewInMemoryRepository()
	handlers := NewStreamHandlers(streamRepo, nil, analyticsRepo, sceneRepo, eventRepo, auditRepo, nil, nil, nil)

	// Create a scene and stream
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

	sceneID := testScene.ID
	streamID, _, err := streamRepo.CreateStreamSession(&sceneID, nil, "did:plc:test123")
	if err != nil {
		t.Fatalf("failed to create stream: %v", err)
	}

	// End the stream but DON'T compute analytics
	_ = streamRepo.EndStreamSession(streamID)

	// Request analytics
	req := httptest.NewRequest(http.MethodGet, "/streams/"+streamID+"/analytics", nil)
	ctx := middleware.SetUserDID(req.Context(), "did:plc:test123")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	handlers.GetStreamAnalytics(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d: %s", w.Code, w.Body.String())
	}
}

// TestGetStreamAnalytics_StreamNotEnded tests analytics retrieval for active stream.
func TestGetStreamAnalytics_StreamNotEnded(t *testing.T) {
	streamRepo := stream.NewInMemorySessionRepository()
	analyticsRepo := stream.NewInMemoryAnalyticsRepository(streamRepo)
	sceneRepo := scene.NewInMemorySceneRepository()
	eventRepo := scene.NewInMemoryEventRepository()
	auditRepo := audit.NewInMemoryRepository()
	handlers := NewStreamHandlers(streamRepo, nil, analyticsRepo, sceneRepo, eventRepo, auditRepo, nil, nil, nil)

	// Create a scene and stream
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

	sceneID := testScene.ID
	streamID, _, err := streamRepo.CreateStreamSession(&sceneID, nil, "did:plc:test123")
	if err != nil {
		t.Fatalf("failed to create stream: %v", err)
	}

	// DON'T end the stream

	// Request analytics
	req := httptest.NewRequest(http.MethodGet, "/streams/"+streamID+"/analytics", nil)
	ctx := middleware.SetUserDID(req.Context(), "did:plc:test123")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	handlers.GetStreamAnalytics(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d: %s", w.Code, w.Body.String())
	}
}

// TestJoinStreamWithGeohash tests join event recording with geographic data.
func TestJoinStreamWithGeohash(t *testing.T) {
	streamRepo := stream.NewInMemorySessionRepository()
	analyticsRepo := stream.NewInMemoryAnalyticsRepository(streamRepo)
	sceneRepo := scene.NewInMemorySceneRepository()
	eventRepo := scene.NewInMemoryEventRepository()
	auditRepo := audit.NewInMemoryRepository()
	handlers := NewStreamHandlers(streamRepo, nil, analyticsRepo, sceneRepo, eventRepo, auditRepo, nil, nil, nil)

	// Create a scene and stream
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

	sceneID := testScene.ID
	streamID, _, err := streamRepo.CreateStreamSession(&sceneID, nil, "did:plc:test123")
	if err != nil {
		t.Fatalf("failed to create stream: %v", err)
	}

	// Join with geohash prefix
	geohash := "dr5regw3"
	reqBody := JoinStreamRequest{
		GeohashPrefix: &geohash,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/streams/"+streamID+"/join", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	ctx := middleware.SetUserDID(req.Context(), "did:plc:user1")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	handlers.JoinStream(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify event was recorded
	events, err := analyticsRepo.GetParticipantEvents(streamID)
	if err != nil {
		t.Fatalf("failed to get events: %v", err)
	}

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	// Verify geohash was truncated to 4 characters for privacy
	if events[0].GeohashPrefix == nil || *events[0].GeohashPrefix != "dr5r" {
		t.Errorf("expected geohash prefix 'dr5r', got %v", events[0].GeohashPrefix)
	}
}

// TestEndStreamComputesAnalytics tests that ending a stream automatically computes analytics.
func TestEndStreamComputesAnalytics(t *testing.T) {
	streamRepo := stream.NewInMemorySessionRepository()
	analyticsRepo := stream.NewInMemoryAnalyticsRepository(streamRepo)
	sceneRepo := scene.NewInMemorySceneRepository()
	eventRepo := scene.NewInMemoryEventRepository()
	auditRepo := audit.NewInMemoryRepository()
	handlers := NewStreamHandlers(streamRepo, nil, analyticsRepo, sceneRepo, eventRepo, auditRepo, nil, nil, nil)

	// Create a scene and stream
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

	sceneID := testScene.ID
	streamID, _, err := streamRepo.CreateStreamSession(&sceneID, nil, "did:plc:test123")
	if err != nil {
		t.Fatalf("failed to create stream: %v", err)
	}

	// Record some events
	_ = analyticsRepo.RecordParticipantEvent(streamID, "did:plc:user1", "join", nil)

	// End the stream
	req := httptest.NewRequest(http.MethodPost, "/streams/"+streamID+"/end", nil)
	ctx := middleware.SetUserDID(req.Context(), "did:plc:test123")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	handlers.EndStream(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify analytics were computed
	analytics, err := analyticsRepo.GetAnalytics(streamID)
	if err != nil {
		t.Fatalf("expected analytics to be computed, got error: %v", err)
	}

	if analytics.TotalJoinAttempts != 1 {
		t.Errorf("expected 1 join attempt in analytics, got %d", analytics.TotalJoinAttempts)
	}
}
