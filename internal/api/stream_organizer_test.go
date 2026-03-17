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

// TestMuteParticipant_Unauthorized tests that non-host cannot mute participants.
func TestMuteParticipant_Unauthorized(t *testing.T) {
	streamRepo := stream.NewInMemorySessionRepository()
	sceneRepo := scene.NewInMemorySceneRepository()
	eventRepo := scene.NewInMemoryEventRepository()
	auditRepo := audit.NewInMemoryRepository()

	handlers := NewStreamHandlers(streamRepo, nil, nil, sceneRepo, eventRepo, auditRepo, nil, nil, nil)

	// Create a scene
	testScene := &scene.Scene{
		ID:            uuid.New().String(),
		Name:          "Test Scene",
		OwnerDID:      "did:plc:host123",
		CoarseGeohash: "dr5regw",
		CreatedAt:     &time.Time{},
	}
	if err := sceneRepo.Insert(testScene); err != nil {
		t.Fatalf("failed to insert scene: %v", err)
	}

	// Create a stream session
	streamID, _, err := streamRepo.CreateStreamSession(ptrString(testScene.ID), nil, "did:plc:host123")
	if err != nil {
		t.Fatalf("failed to create stream: %v", err)
	}

	// Try to mute as non-host
	reqBody := MuteParticipantRequest{
		Muted: true,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/streams/"+streamID+"/participants/user-participant1/mute", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	ctx := middleware.SetUserDID(req.Context(), "did:plc:otheruser")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	handlers.MuteParticipant(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected status 403, got %d: %s", w.Code, w.Body.String())
	}
}

// TestKickParticipant_Unauthorized tests that non-host cannot kick participants.
func TestKickParticipant_Unauthorized(t *testing.T) {
	streamRepo := stream.NewInMemorySessionRepository()
	sceneRepo := scene.NewInMemorySceneRepository()
	eventRepo := scene.NewInMemoryEventRepository()
	auditRepo := audit.NewInMemoryRepository()

	handlers := NewStreamHandlers(streamRepo, nil, nil, sceneRepo, eventRepo, auditRepo, nil, nil, nil)

	// Create a scene
	testScene := &scene.Scene{
		ID:            uuid.New().String(),
		Name:          "Test Scene",
		OwnerDID:      "did:plc:host123",
		CoarseGeohash: "dr5regw",
		CreatedAt:     &time.Time{},
	}
	if err := sceneRepo.Insert(testScene); err != nil {
		t.Fatalf("failed to insert scene: %v", err)
	}

	// Create a stream session
	streamID, _, err := streamRepo.CreateStreamSession(ptrString(testScene.ID), nil, "did:plc:host123")
	if err != nil {
		t.Fatalf("failed to create stream: %v", err)
	}

	// Try to kick as non-host
	req := httptest.NewRequest(http.MethodPost, "/streams/"+streamID+"/participants/user-participant1/kick", nil)
	ctx := middleware.SetUserDID(req.Context(), "did:plc:otheruser")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	handlers.KickParticipant(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected status 403, got %d: %s", w.Code, w.Body.String())
	}
}

// TestSetFeaturedParticipant_Success tests setting a featured participant.
func TestSetFeaturedParticipant_Success(t *testing.T) {
	streamRepo := stream.NewInMemorySessionRepository()
	sceneRepo := scene.NewInMemorySceneRepository()
	eventRepo := scene.NewInMemoryEventRepository()
	auditRepo := audit.NewInMemoryRepository()

	handlers := NewStreamHandlers(streamRepo, nil, nil, sceneRepo, eventRepo, auditRepo, nil, nil, nil)

	// Create a scene
	testScene := &scene.Scene{
		ID:            uuid.New().String(),
		Name:          "Test Scene",
		OwnerDID:      "did:plc:host123",
		CoarseGeohash: "dr5regw",
		CreatedAt:     &time.Time{},
	}
	if err := sceneRepo.Insert(testScene); err != nil {
		t.Fatalf("failed to insert scene: %v", err)
	}

	// Create a stream session
	streamID, _, err := streamRepo.CreateStreamSession(ptrString(testScene.ID), nil, "did:plc:host123")
	if err != nil {
		t.Fatalf("failed to create stream: %v", err)
	}

	// Set featured participant
	participantID := "user-participant1"
	reqBody := SetFeaturedParticipantRequest{
		ParticipantID: &participantID,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPatch, "/streams/"+streamID+"/featured_participant", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	ctx := middleware.SetUserDID(req.Context(), "did:plc:host123")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	handlers.SetFeaturedParticipant(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify featured participant was set
	session, err := streamRepo.GetByID(streamID)
	if err != nil {
		t.Fatalf("failed to get stream session: %v", err)
	}

	if session.FeaturedParticipant == nil || *session.FeaturedParticipant != participantID {
		t.Errorf("expected featured participant %s, got %v", participantID, session.FeaturedParticipant)
	}
}

// TestSetFeaturedParticipant_Clear tests clearing the featured participant.
func TestSetFeaturedParticipant_Clear(t *testing.T) {
	streamRepo := stream.NewInMemorySessionRepository()
	sceneRepo := scene.NewInMemorySceneRepository()
	eventRepo := scene.NewInMemoryEventRepository()
	auditRepo := audit.NewInMemoryRepository()

	handlers := NewStreamHandlers(streamRepo, nil, nil, sceneRepo, eventRepo, auditRepo, nil, nil, nil)

	// Create a scene
	testScene := &scene.Scene{
		ID:            uuid.New().String(),
		Name:          "Test Scene",
		OwnerDID:      "did:plc:host123",
		CoarseGeohash: "dr5regw",
		CreatedAt:     &time.Time{},
	}
	if err := sceneRepo.Insert(testScene); err != nil {
		t.Fatalf("failed to insert scene: %v", err)
	}

	// Create a stream session
	streamID, _, err := streamRepo.CreateStreamSession(ptrString(testScene.ID), nil, "did:plc:host123")
	if err != nil {
		t.Fatalf("failed to create stream: %v", err)
	}

	// Set featured participant first
	participant := "user-participant1"
	if err := streamRepo.SetFeaturedParticipant(streamID, &participant); err != nil {
		t.Fatalf("failed to set featured participant: %v", err)
	}

	// Clear featured participant
	reqBody := SetFeaturedParticipantRequest{
		ParticipantID: nil,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPatch, "/streams/"+streamID+"/featured_participant", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	ctx := middleware.SetUserDID(req.Context(), "did:plc:host123")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	handlers.SetFeaturedParticipant(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify featured participant was cleared
	session, err := streamRepo.GetByID(streamID)
	if err != nil {
		t.Fatalf("failed to get stream session: %v", err)
	}

	if session.FeaturedParticipant != nil {
		t.Errorf("expected featured participant to be nil, got %v", session.FeaturedParticipant)
	}
}

// TestLockStream_Success tests locking and unlocking a stream.
func TestLockStream_Success(t *testing.T) {
	streamRepo := stream.NewInMemorySessionRepository()
	sceneRepo := scene.NewInMemorySceneRepository()
	eventRepo := scene.NewInMemoryEventRepository()
	auditRepo := audit.NewInMemoryRepository()

	handlers := NewStreamHandlers(streamRepo, nil, nil, sceneRepo, eventRepo, auditRepo, nil, nil, nil)

	// Create a scene
	testScene := &scene.Scene{
		ID:            uuid.New().String(),
		Name:          "Test Scene",
		OwnerDID:      "did:plc:host123",
		CoarseGeohash: "dr5regw",
		CreatedAt:     &time.Time{},
	}
	if err := sceneRepo.Insert(testScene); err != nil {
		t.Fatalf("failed to insert scene: %v", err)
	}

	// Create a stream session
	streamID, _, err := streamRepo.CreateStreamSession(ptrString(testScene.ID), nil, "did:plc:host123")
	if err != nil {
		t.Fatalf("failed to create stream: %v", err)
	}

	// Lock the stream
	reqBody := LockStreamRequest{
		Locked: true,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPatch, "/streams/"+streamID+"/lock", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	ctx := middleware.SetUserDID(req.Context(), "did:plc:host123")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	handlers.LockStream(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify stream is locked
	session, err := streamRepo.GetByID(streamID)
	if err != nil {
		t.Fatalf("failed to get stream session: %v", err)
	}

	if !session.IsLocked {
		t.Error("expected stream to be locked")
	}
}

// TestLockStream_Unauthorized tests that non-host cannot lock stream.
func TestLockStream_Unauthorized(t *testing.T) {
	streamRepo := stream.NewInMemorySessionRepository()
	sceneRepo := scene.NewInMemorySceneRepository()
	eventRepo := scene.NewInMemoryEventRepository()
	auditRepo := audit.NewInMemoryRepository()

	handlers := NewStreamHandlers(streamRepo, nil, nil, sceneRepo, eventRepo, auditRepo, nil, nil, nil)

	// Create a scene
	testScene := &scene.Scene{
		ID:            uuid.New().String(),
		Name:          "Test Scene",
		OwnerDID:      "did:plc:host123",
		CoarseGeohash: "dr5regw",
		CreatedAt:     &time.Time{},
	}
	if err := sceneRepo.Insert(testScene); err != nil {
		t.Fatalf("failed to insert scene: %v", err)
	}

	// Create a stream session
	streamID, _, err := streamRepo.CreateStreamSession(ptrString(testScene.ID), nil, "did:plc:host123")
	if err != nil {
		t.Fatalf("failed to create stream: %v", err)
	}

	// Try to lock as non-host
	reqBody := LockStreamRequest{
		Locked: true,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPatch, "/streams/"+streamID+"/lock", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	ctx := middleware.SetUserDID(req.Context(), "did:plc:otheruser")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	handlers.LockStream(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected status 403, got %d: %s", w.Code, w.Body.String())
	}

	// Verify stream is not locked
	session, err := streamRepo.GetByID(streamID)
	if err != nil {
		t.Fatalf("failed to get stream session: %v", err)
	}

	if session.IsLocked {
		t.Error("expected stream to not be locked")
	}
}
