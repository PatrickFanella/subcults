package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/onnwee/subcults/internal/audit"
	"github.com/onnwee/subcults/internal/scene"
	"github.com/onnwee/subcults/internal/stream"
)

// TestGetEvent_WithActiveStream tests that GetEvent returns active stream info.
func TestGetEvent_WithActiveStream(t *testing.T) {
	eventRepo := scene.NewInMemoryEventRepository()
	sceneRepo := scene.NewInMemorySceneRepository()
	auditRepo := audit.NewInMemoryRepository()
	rsvpRepo := scene.NewInMemoryRSVPRepository()
	streamRepo := stream.NewInMemorySessionRepository()
	handlers := NewEventHandlers(eventRepo, sceneRepo, auditRepo, rsvpRepo, streamRepo, nil)

	// Create test event
	eventID := uuid.New().String()
	startsAt := time.Now().Add(1 * time.Hour)
	testEvent := &scene.Event{
		ID:            eventID,
		SceneID:       uuid.New().String(),
		Title:         "Test Event with Stream",
		CoarseGeohash: "dr5regw",
		Status:        "scheduled",
		StartsAt:      startsAt,
		CreatedAt:     &startsAt,
		UpdatedAt:     &startsAt,
	}
	if err := eventRepo.Insert(testEvent); err != nil {
		t.Fatalf("failed to insert event: %v", err)
	}

	// Create active stream for the event
	eventIDPtr := &testEvent.ID
	streamID, roomName, err := streamRepo.CreateStreamSession(nil, eventIDPtr, "did:plc:host123")
	if err != nil {
		t.Fatalf("failed to create stream session: %v", err)
	}

	// Get the event
	req := httptest.NewRequest(http.MethodGet, "/events/"+eventID, nil)
	w := httptest.NewRecorder()

	handlers.GetEvent(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var response EventWithRSVPCounts
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Verify active stream is included
	if response.ActiveStream == nil {
		t.Fatal("expected active_stream to be present, got nil")
	}
	if response.ActiveStream.StreamSessionID != streamID {
		t.Errorf("expected stream_session_id '%s', got '%s'", streamID, response.ActiveStream.StreamSessionID)
	}
	if response.ActiveStream.RoomName != roomName {
		t.Errorf("expected room_name '%s', got '%s'", roomName, response.ActiveStream.RoomName)
	}
	if response.ActiveStream.StartedAt.IsZero() {
		t.Error("expected started_at to be set")
	}
}

// TestGetEvent_WithEndedStream tests that GetEvent returns nil for ended streams.
func TestGetEvent_WithEndedStream(t *testing.T) {
	eventRepo := scene.NewInMemoryEventRepository()
	sceneRepo := scene.NewInMemorySceneRepository()
	auditRepo := audit.NewInMemoryRepository()
	rsvpRepo := scene.NewInMemoryRSVPRepository()
	streamRepo := stream.NewInMemorySessionRepository()
	handlers := NewEventHandlers(eventRepo, sceneRepo, auditRepo, rsvpRepo, streamRepo, nil)

	// Create test event
	eventID := uuid.New().String()
	startsAt := time.Now().Add(1 * time.Hour)
	testEvent := &scene.Event{
		ID:            eventID,
		SceneID:       uuid.New().String(),
		Title:         "Test Event with Ended Stream",
		CoarseGeohash: "dr5regw",
		Status:        "scheduled",
		StartsAt:      startsAt,
		CreatedAt:     &startsAt,
		UpdatedAt:     &startsAt,
	}
	if err := eventRepo.Insert(testEvent); err != nil {
		t.Fatalf("failed to insert event: %v", err)
	}

	// Create and end stream for the event
	eventIDPtr := &testEvent.ID
	streamID, _, err := streamRepo.CreateStreamSession(nil, eventIDPtr, "did:plc:host123")
	if err != nil {
		t.Fatalf("failed to create stream session: %v", err)
	}
	if err := streamRepo.EndStreamSession(streamID); err != nil {
		t.Fatalf("failed to end stream session: %v", err)
	}

	// Get the event
	req := httptest.NewRequest(http.MethodGet, "/events/"+eventID, nil)
	w := httptest.NewRecorder()

	handlers.GetEvent(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var response EventWithRSVPCounts
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Verify active stream is nil for ended stream
	if response.ActiveStream != nil {
		t.Errorf("expected active_stream to be nil for ended stream, got %+v", response.ActiveStream)
	}
}

// TestGetEvent_WithNoStream tests that GetEvent returns nil when no stream exists.
func TestGetEvent_WithNoStream(t *testing.T) {
	eventRepo := scene.NewInMemoryEventRepository()
	sceneRepo := scene.NewInMemorySceneRepository()
	auditRepo := audit.NewInMemoryRepository()
	rsvpRepo := scene.NewInMemoryRSVPRepository()
	streamRepo := stream.NewInMemorySessionRepository()
	handlers := NewEventHandlers(eventRepo, sceneRepo, auditRepo, rsvpRepo, streamRepo, nil)

	// Create test event
	eventID := uuid.New().String()
	startsAt := time.Now().Add(1 * time.Hour)
	testEvent := &scene.Event{
		ID:            eventID,
		SceneID:       uuid.New().String(),
		Title:         "Test Event without Stream",
		CoarseGeohash: "dr5regw",
		Status:        "scheduled",
		StartsAt:      startsAt,
		CreatedAt:     &startsAt,
		UpdatedAt:     &startsAt,
	}
	if err := eventRepo.Insert(testEvent); err != nil {
		t.Fatalf("failed to insert event: %v", err)
	}

	// Get the event (no stream created)
	req := httptest.NewRequest(http.MethodGet, "/events/"+eventID, nil)
	w := httptest.NewRecorder()

	handlers.GetEvent(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var response EventWithRSVPCounts
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Verify active stream is nil when no stream exists
	if response.ActiveStream != nil {
		t.Errorf("expected active_stream to be nil when no stream exists, got %+v", response.ActiveStream)
	}
}

// TestSearchEvents_WithActiveStreams tests that SearchEvents includes active stream info via batch query.
func TestSearchEvents_WithActiveStreams(t *testing.T) {
	eventRepo := scene.NewInMemoryEventRepository()
	sceneRepo := scene.NewInMemorySceneRepository()
	auditRepo := audit.NewInMemoryRepository()
	rsvpRepo := scene.NewInMemoryRSVPRepository()
	streamRepo := stream.NewInMemorySessionRepository()
	handlers := NewEventHandlers(eventRepo, sceneRepo, auditRepo, rsvpRepo, streamRepo, nil)

	baseTime := time.Now().Add(24 * time.Hour)

	// Create 3 events
	events := make([]*scene.Event, 3)
	for i := 0; i < 3; i++ {
		eventID := uuid.New().String()
		events[i] = &scene.Event{
			ID:            eventID,
			SceneID:       uuid.New().String(),
			Title:         "Event " + string(rune('A'+i)),
			AllowPrecise:  true,
			PrecisePoint:  &scene.Point{Lat: 40.7128, Lng: -74.0060},
			CoarseGeohash: "dr5regw",
			Status:        "scheduled",
			StartsAt:      baseTime.Add(time.Duration(i) * time.Hour),
			CreatedAt:     &baseTime,
			UpdatedAt:     &baseTime,
		}
		if err := eventRepo.Insert(events[i]); err != nil {
			t.Fatalf("failed to insert event: %v", err)
		}
	}

	// Create active stream for event 0
	event0ID := events[0].ID
	streamID0, roomName0, err := streamRepo.CreateStreamSession(nil, &event0ID, "did:plc:host123")
	if err != nil {
		t.Fatalf("failed to create stream session 0: %v", err)
	}

	// Create active stream for event 2
	event2ID := events[2].ID
	streamID2, roomName2, err := streamRepo.CreateStreamSession(nil, &event2ID, "did:plc:host456")
	if err != nil {
		t.Fatalf("failed to create stream session 2: %v", err)
	}

	// Event 1 has no stream

	// Search events
	from := baseTime.Add(-1 * time.Hour)
	to := baseTime.Add(5 * time.Hour)
	url := fmt.Sprintf("/search/events?bbox=-75,-90,0,90&from=%s&to=%s&limit=10",
		from.Format(time.RFC3339), to.Format(time.RFC3339))
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

	if len(response.Events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(response.Events))
	}

	// Build a map of event ID to response index for easier verification
	eventIDToResponse := make(map[string]*EventWithRSVPCounts)
	for i := range response.Events {
		eventIDToResponse[response.Events[i].ID] = response.Events[i]
	}

	// Verify event 0 (Event A with stream) has active stream
	event0Response := eventIDToResponse[events[0].ID]
	if event0Response == nil {
		t.Fatal("event 0 not found in response")
	}
	if event0Response.ActiveStream == nil {
		t.Error("expected event 0 (Event A) to have active stream")
	} else {
		if event0Response.ActiveStream.StreamSessionID != streamID0 {
			t.Errorf("event 0: expected stream_session_id '%s', got '%s'", streamID0, event0Response.ActiveStream.StreamSessionID)
		}
		if event0Response.ActiveStream.RoomName != roomName0 {
			t.Errorf("event 0: expected room_name '%s', got '%s'", roomName0, event0Response.ActiveStream.RoomName)
		}
	}

	// Verify event 1 (Event B without stream) has no active stream
	event1Response := eventIDToResponse[events[1].ID]
	if event1Response == nil {
		t.Fatal("event 1 not found in response")
	}
	if event1Response.ActiveStream != nil {
		t.Errorf("expected event 1 (Event B) to have no active stream, got %+v", event1Response.ActiveStream)
	}

	// Verify event 2 (Event C with stream) has active stream
	event2Response := eventIDToResponse[events[2].ID]
	if event2Response == nil {
		t.Fatal("event 2 not found in response")
	}
	if event2Response.ActiveStream == nil {
		t.Error("expected event 2 (Event C) to have active stream")
	} else {
		if event2Response.ActiveStream.StreamSessionID != streamID2 {
			t.Errorf("event 2: expected stream_session_id '%s', got '%s'", streamID2, event2Response.ActiveStream.StreamSessionID)
		}
		if event2Response.ActiveStream.RoomName != roomName2 {
			t.Errorf("event 2: expected room_name '%s', got '%s'", roomName2, event2Response.ActiveStream.RoomName)
		}
	}
}

// TestSearchEvents_BatchQueryPerformance tests that batch query performs efficiently with many events.
func TestSearchEvents_BatchQueryPerformance(t *testing.T) {
	eventRepo := scene.NewInMemoryEventRepository()
	sceneRepo := scene.NewInMemorySceneRepository()
	auditRepo := audit.NewInMemoryRepository()
	rsvpRepo := scene.NewInMemoryRSVPRepository()
	streamRepo := stream.NewInMemorySessionRepository()
	handlers := NewEventHandlers(eventRepo, sceneRepo, auditRepo, rsvpRepo, streamRepo, nil)

	baseTime := time.Now().Add(24 * time.Hour)

	// Create 100 events
	const numEvents = 100
	for i := 0; i < numEvents; i++ {
		eventID := uuid.New().String()
		event := &scene.Event{
			ID:            eventID,
			SceneID:       uuid.New().String(),
			Title:         fmt.Sprintf("Event %d", i),
			AllowPrecise:  true,
			PrecisePoint:  &scene.Point{Lat: 40.7128, Lng: -74.0060},
			CoarseGeohash: "dr5regw",
			Status:        "scheduled",
			StartsAt:      baseTime.Add(time.Duration(i) * time.Minute),
			CreatedAt:     &baseTime,
			UpdatedAt:     &baseTime,
		}
		if err := eventRepo.Insert(event); err != nil {
			t.Fatalf("failed to insert event %d: %v", i, err)
		}

		// Create active stream for half of the events
		if i%2 == 0 {
			eventIDCopy := eventID
			_, _, err := streamRepo.CreateStreamSession(nil, &eventIDCopy, fmt.Sprintf("did:plc:host%d", i))
			if err != nil {
				t.Fatalf("failed to create stream session %d: %v", i, err)
			}
		}
	}

	// Search events and measure performance
	from := baseTime.Add(-1 * time.Hour)
	to := baseTime.Add(200 * time.Minute)
	url := fmt.Sprintf("/search/events?bbox=-75,-90,0,90&from=%s&to=%s&limit=100",
		from.Format(time.RFC3339), to.Format(time.RFC3339))
	req := httptest.NewRequest(http.MethodGet, url, nil)
	w := httptest.NewRecorder()

	startTime := time.Now()
	handlers.SearchEvents(w, req)
	elapsed := time.Since(startTime)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify latency is under 150ms target
	maxLatency := 150 * time.Millisecond
	if elapsed > maxLatency {
		t.Errorf("search latency %v exceeds target of %v", elapsed, maxLatency)
	}

	var response SearchEventsResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(response.Events) != numEvents {
		t.Errorf("expected %d events, got %d", numEvents, len(response.Events))
	}

	// Verify half of events have active streams
	activeCount := 0
	for _, event := range response.Events {
		if event.ActiveStream != nil {
			activeCount++
		}
	}
	expectedActive := numEvents / 2
	if activeCount != expectedActive {
		t.Errorf("expected %d events with active streams, got %d", expectedActive, activeCount)
	}

	t.Logf("Successfully searched %d events with batch stream query in %v", numEvents, elapsed)
}
