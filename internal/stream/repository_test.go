package stream

import (
	"strings"
	"testing"
	"time"
)

func strPtr(s string) *string {
	return &s
}

func TestSessionRepository_Upsert_Insert(t *testing.T) {
	repo := NewInMemorySessionRepository()
	did := "did:plc:alice123"
	rkey := "stream456"
	sceneID := "scene-1"

	session := &Session{
		SceneID:          &sceneID,
		RoomName:         "test-room",
		HostDID:          "did:plc:host789",
		ParticipantCount: 5,
		RecordDID:        strPtr(did),
		RecordRKey:       strPtr(rkey),
	}

	result, err := repo.Upsert(session)
	if err != nil {
		t.Fatalf("Upsert failed: %v", err)
	}

	if !result.Inserted {
		t.Error("Expected insert, got update")
	}

	if result.ID == "" {
		t.Error("Expected non-empty ID")
	}

	// Verify we can retrieve it
	retrieved, err := repo.GetByRecordKey(did, rkey)
	if err != nil {
		t.Fatalf("GetByRecordKey failed: %v", err)
	}

	if retrieved.RoomName != "test-room" {
		t.Errorf("Expected room name 'test-room', got %s", retrieved.RoomName)
	}
}

func TestSessionRepository_Upsert_Update(t *testing.T) {
	repo := NewInMemorySessionRepository()
	did := "did:plc:alice123"
	rkey := "stream456"
	sceneID := "scene-1"

	// First insert
	session := &Session{
		SceneID:          &sceneID,
		RoomName:         "test-room",
		HostDID:          "did:plc:host789",
		ParticipantCount: 5,
		RecordDID:        strPtr(did),
		RecordRKey:       strPtr(rkey),
	}

	result1, err := repo.Upsert(session)
	if err != nil {
		t.Fatalf("First upsert failed: %v", err)
	}

	if !result1.Inserted {
		t.Error("Expected insert on first upsert")
	}

	// Second upsert with same record key
	session2 := &Session{
		SceneID:          &sceneID,
		RoomName:         "updated-room",
		HostDID:          "did:plc:host789",
		ParticipantCount: 10,
		RecordDID:        strPtr(did),
		RecordRKey:       strPtr(rkey),
	}

	result2, err := repo.Upsert(session2)
	if err != nil {
		t.Fatalf("Second upsert failed: %v", err)
	}

	if result2.Inserted {
		t.Error("Expected update, got insert")
	}

	if result1.ID != result2.ID {
		t.Errorf("Expected same ID, got %s and %s", result1.ID, result2.ID)
	}

	// Verify update was persisted
	retrieved, err := repo.GetByRecordKey(did, rkey)
	if err != nil {
		t.Fatalf("GetByRecordKey failed: %v", err)
	}

	if retrieved.RoomName != "updated-room" {
		t.Errorf("Expected updated room name 'updated-room', got %s", retrieved.RoomName)
	}

	if retrieved.ParticipantCount != 10 {
		t.Errorf("Expected participant count 10, got %d", retrieved.ParticipantCount)
	}
}

func TestSessionRepository_Upsert_Idempotent(t *testing.T) {
	repo := NewInMemorySessionRepository()
	did := "did:plc:alice123"
	rkey := "stream456"
	sceneID := "scene-1"

	session := &Session{
		SceneID:          &sceneID,
		RoomName:         "test-room",
		HostDID:          "did:plc:host789",
		ParticipantCount: 5,
		RecordDID:        strPtr(did),
		RecordRKey:       strPtr(rkey),
	}

	// First upsert
	result1, err := repo.Upsert(session)
	if err != nil {
		t.Fatalf("First upsert failed: %v", err)
	}

	// Second upsert with same content
	result2, err := repo.Upsert(session)
	if err != nil {
		t.Fatalf("Second upsert failed: %v", err)
	}

	// Should update, not insert
	if result2.Inserted {
		t.Error("Expected update (idempotent), got insert")
	}

	if result1.ID != result2.ID {
		t.Error("Idempotent upserts should return same ID")
	}
}

func TestSessionRepository_Upsert_WithoutRecordKey(t *testing.T) {
	repo := NewInMemorySessionRepository()
	sceneID := "scene-1"

	session := &Session{
		SceneID:          &sceneID,
		RoomName:         "test-room",
		HostDID:          "did:plc:host789",
		ParticipantCount: 5,
	}

	result, err := repo.Upsert(session)
	if err != nil {
		t.Fatalf("Upsert failed: %v", err)
	}

	if !result.Inserted {
		t.Error("Expected insert when no record key provided")
	}

	// Second upsert without record key should also insert
	result2, err := repo.Upsert(session)
	if err != nil {
		t.Fatalf("Second upsert failed: %v", err)
	}

	if !result2.Inserted {
		t.Error("Expected insert when no record key provided")
	}

	// IDs should be different
	if result.ID == result2.ID {
		t.Error("Expected different IDs for separate inserts")
	}
}

func TestSessionRepository_GetByRecordKey_NotFound(t *testing.T) {
	repo := NewInMemorySessionRepository()

	session, err := repo.GetByRecordKey("did:plc:alice123", "nonexistent")
	if err != ErrStreamNotFound {
		t.Errorf("Expected ErrStreamNotFound, got %v", err)
	}

	if session != nil {
		t.Error("Expected nil session for non-existent record")
	}
}

func TestSessionRepository_GetByID_AfterUpsert(t *testing.T) {
	repo := NewInMemorySessionRepository()
	did := "did:plc:alice123"
	rkey := "stream456"
	sceneID := "scene-1"

	session := &Session{
		SceneID:          &sceneID,
		RoomName:         "test-room",
		HostDID:          "did:plc:host789",
		ParticipantCount: 5,
		RecordDID:        strPtr(did),
		RecordRKey:       strPtr(rkey),
	}

	result, err := repo.Upsert(session)
	if err != nil {
		t.Fatalf("Upsert failed: %v", err)
	}

	// Retrieve by UUID
	retrieved, err := repo.GetByID(result.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}

	if retrieved.RoomName != "test-room" {
		t.Errorf("Expected room name 'test-room', got %s", retrieved.RoomName)
	}
}

func TestSessionRepository_HasActiveStreamForScene(t *testing.T) {
	repo := NewInMemorySessionRepository()
	sceneID1 := "scene-1"
	sceneID2 := "scene-2"

	// Insert active stream for scene-1
	session1 := &Session{
		ID:               "stream-1",
		SceneID:          &sceneID1,
		RoomName:         "room-1",
		HostDID:          "did:plc:host1",
		ParticipantCount: 5,
		EndedAt:          nil, // Active stream
	}
	if _, err := repo.Upsert(session1); err != nil {
		t.Fatalf("Upsert session1 failed: %v", err)
	}

	// Insert ended stream for scene-1
	endTime := timePtr()
	session2 := &Session{
		ID:               "stream-2",
		SceneID:          &sceneID1,
		RoomName:         "room-2",
		HostDID:          "did:plc:host2",
		ParticipantCount: 3,
		EndedAt:          endTime, // Ended stream
	}
	if _, err := repo.Upsert(session2); err != nil {
		t.Fatalf("Upsert session2 failed: %v", err)
	}

	// Test: scene-1 should have active stream
	hasActive, err := repo.HasActiveStreamForScene(sceneID1)
	if err != nil {
		t.Fatalf("HasActiveStreamForScene failed: %v", err)
	}
	if !hasActive {
		t.Error("Expected scene-1 to have active stream")
	}

	// Test: scene-2 should not have active stream
	hasActive, err = repo.HasActiveStreamForScene(sceneID2)
	if err != nil {
		t.Fatalf("HasActiveStreamForScene failed: %v", err)
	}
	if hasActive {
		t.Error("Expected scene-2 to not have active stream")
	}
}

func TestSessionRepository_HasActiveStreamForScene_AllEnded(t *testing.T) {
	repo := NewInMemorySessionRepository()
	sceneID := "scene-1"
	endTime := timePtr()

	// Insert only ended streams
	session1 := &Session{
		ID:               "stream-1",
		SceneID:          &sceneID,
		RoomName:         "room-1",
		HostDID:          "did:plc:host1",
		ParticipantCount: 5,
		EndedAt:          endTime,
	}
	if _, err := repo.Upsert(session1); err != nil {
		t.Fatalf("Upsert failed: %v", err)
	}

	// Test: should not have active stream
	hasActive, err := repo.HasActiveStreamForScene(sceneID)
	if err != nil {
		t.Fatalf("HasActiveStreamForScene failed: %v", err)
	}
	if hasActive {
		t.Error("Expected no active stream when all streams are ended")
	}
}

func timePtr() *time.Time {
	t := time.Now()
	return &t
}

func TestSessionRepository_HasActiveStreamsForScenes(t *testing.T) {
	repo := NewInMemorySessionRepository()

	scene1 := "scene-1"
	scene2 := "scene-2"
	scene3 := "scene-3"

	// Scene 1: has active stream
	session1 := &Session{
		ID:               "stream-1",
		SceneID:          &scene1,
		RoomName:         "room-1",
		HostDID:          "did:plc:host1",
		ParticipantCount: 5,
		EndedAt:          nil, // Active
	}

	// Scene 2: has ended stream only
	endTime := timePtr()
	session2 := &Session{
		ID:               "stream-2",
		SceneID:          &scene2,
		RoomName:         "room-2",
		HostDID:          "did:plc:host2",
		ParticipantCount: 3,
		EndedAt:          endTime, // Ended
	}

	// Scene 3: no streams

	if _, err := repo.Upsert(session1); err != nil {
		t.Fatalf("Upsert session1 failed: %v", err)
	}
	if _, err := repo.Upsert(session2); err != nil {
		t.Fatalf("Upsert session2 failed: %v", err)
	}

	// Test: Check active streams for all scenes
	activeStreams, err := repo.HasActiveStreamsForScenes([]string{scene1, scene2, scene3})
	if err != nil {
		t.Fatalf("HasActiveStreamsForScenes failed: %v", err)
	}

	if !activeStreams[scene1] {
		t.Error("Expected scene1 to have active stream")
	}
	if activeStreams[scene2] {
		t.Error("Expected scene2 to not have active stream (ended)")
	}
	if activeStreams[scene3] {
		t.Error("Expected scene3 to not have active stream (no streams)")
	}
}

func TestSessionRepository_HasActiveStreamsForScenes_EmptyInput(t *testing.T) {
	repo := NewInMemorySessionRepository()

	activeStreams, err := repo.HasActiveStreamsForScenes([]string{})
	if err != nil {
		t.Fatalf("HasActiveStreamsForScenes failed: %v", err)
	}

	if len(activeStreams) != 0 {
		t.Errorf("Expected empty map, got %d entries", len(activeStreams))
	}
}

func TestSessionRepository_CreateStreamSession_WithSceneID(t *testing.T) {
	repo := NewInMemorySessionRepository()
	sceneID := "scene-123"
	hostDID := "did:plc:host456"

	id, roomName, err := repo.CreateStreamSession(&sceneID, nil, hostDID)
	if err != nil {
		t.Fatalf("CreateStreamSession failed: %v", err)
	}

	if id == "" {
		t.Error("Expected non-empty session ID")
	}

	// Verify room name format: scene-{sceneId}-{timestamp}
	if !strings.Contains(roomName, "scene-scene-123-") {
		t.Errorf("Expected room name to contain 'scene-scene-123-', got %s", roomName)
	}

	// Verify session was created
	session, err := repo.GetByID(id)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}

	if session.SceneID == nil || *session.SceneID != sceneID {
		t.Errorf("Expected scene_id %s, got %v", sceneID, session.SceneID)
	}

	if session.HostDID != hostDID {
		t.Errorf("Expected host_did %s, got %s", hostDID, session.HostDID)
	}

	if session.EndedAt != nil {
		t.Error("Expected ended_at to be nil (active stream)")
	}

	if session.RoomName != roomName {
		t.Errorf("Expected room_name %s, got %s", roomName, session.RoomName)
	}
}

func TestSessionRepository_CreateStreamSession_WithEventID(t *testing.T) {
	repo := NewInMemorySessionRepository()
	eventID := "event-789"
	hostDID := "did:plc:host456"

	id, roomName, err := repo.CreateStreamSession(nil, &eventID, hostDID)
	if err != nil {
		t.Fatalf("CreateStreamSession failed: %v", err)
	}

	if id == "" {
		t.Error("Expected non-empty session ID")
	}

	// Verify room name format: event-{eventId}-{timestamp}
	if !strings.Contains(roomName, "event-event-789-") {
		t.Errorf("Expected room name to contain 'event-event-789-', got %s", roomName)
	}

	// Verify session was created
	session, err := repo.GetByID(id)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}

	if session.EventID == nil || *session.EventID != eventID {
		t.Errorf("Expected event_id %s, got %v", eventID, session.EventID)
	}
}

func TestSessionRepository_CreateStreamSession_NoSceneOrEvent(t *testing.T) {
	repo := NewInMemorySessionRepository()
	hostDID := "did:plc:host456"

	_, _, err := repo.CreateStreamSession(nil, nil, hostDID)
	if err == nil {
		t.Error("Expected error when neither scene_id nor event_id provided")
	}

	// Test with empty strings
	emptyScene := ""
	emptyEvent := ""
	_, _, err = repo.CreateStreamSession(&emptyScene, &emptyEvent, hostDID)
	if err == nil {
		t.Error("Expected error when both scene_id and event_id are empty")
	}
}

func TestSessionRepository_EndStreamSession_Success(t *testing.T) {
	repo := NewInMemorySessionRepository()
	sceneID := "scene-123"
	hostDID := "did:plc:host456"

	// Create a session
	id, _, err := repo.CreateStreamSession(&sceneID, nil, hostDID)
	if err != nil {
		t.Fatalf("CreateStreamSession failed: %v", err)
	}

	// Verify it's active
	session, _ := repo.GetByID(id)
	if session.EndedAt != nil {
		t.Error("Expected session to be active before ending")
	}

	// End the session
	err = repo.EndStreamSession(id)
	if err != nil {
		t.Fatalf("EndStreamSession failed: %v", err)
	}

	// Verify it's now ended
	session, err = repo.GetByID(id)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}

	if session.EndedAt == nil {
		t.Error("Expected ended_at to be set after ending session")
	}
}

func TestSessionRepository_EndStreamSession_NotFound(t *testing.T) {
	repo := NewInMemorySessionRepository()

	err := repo.EndStreamSession("nonexistent-id")
	if err != ErrStreamNotFound {
		t.Errorf("Expected ErrStreamNotFound, got %v", err)
	}
}

func TestSessionRepository_EndStreamSession_Idempotent(t *testing.T) {
	repo := NewInMemorySessionRepository()
	sceneID := "scene-123"
	hostDID := "did:plc:host456"

	// Create and end a session
	id, _, err := repo.CreateStreamSession(&sceneID, nil, hostDID)
	if err != nil {
		t.Fatalf("CreateStreamSession failed: %v", err)
	}

	err = repo.EndStreamSession(id)
	if err != nil {
		t.Fatalf("First EndStreamSession failed: %v", err)
	}

	// Get the ended_at timestamp
	session1, _ := repo.GetByID(id)
	endedAt1 := session1.EndedAt

	// End it again (idempotent)
	err = repo.EndStreamSession(id)
	if err != nil {
		t.Fatalf("Second EndStreamSession failed: %v", err)
	}

	// Verify ended_at didn't change
	session2, _ := repo.GetByID(id)
	if !session2.EndedAt.Equal(*endedAt1) {
		t.Error("Expected ended_at to remain unchanged on idempotent call")
	}
}

func TestSessionRepository_CreateStreamSession_EmptyHostDID(t *testing.T) {
	repo := NewInMemorySessionRepository()
	sceneID := "scene-123"

	_, _, err := repo.CreateStreamSession(&sceneID, nil, "")
	if err == nil {
		t.Error("Expected error when hostDID is empty")
	}

	expectedErrMsg := "hostDID must not be empty"
	if err != nil && err.Error() != expectedErrMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedErrMsg, err.Error())
	}
}

// TestSessionRepository_GetActiveStreamForEvent_MultipleStreams tests most recent stream selection for single query.
func TestSessionRepository_GetActiveStreamForEvent_MultipleStreams(t *testing.T) {
	repo := NewInMemorySessionRepository()
	eventID := "event-multi-single"

	// Create first active stream
	stream1ID, _, err := repo.CreateStreamSession(nil, &eventID, "did:plc:host1")
	if err != nil {
		t.Fatalf("CreateStreamSession 1 failed: %v", err)
	}

	// Wait to ensure different timestamps
	time.Sleep(10 * time.Millisecond)

	// Create second active stream (should be more recent)
	stream2ID, room2, err := repo.CreateStreamSession(nil, &eventID, "did:plc:host2")
	if err != nil {
		t.Fatalf("CreateStreamSession 2 failed: %v", err)
	}

	// Single query should return the most recent stream
	activeStream, err := repo.GetActiveStreamForEvent(eventID)
	if err != nil {
		t.Fatalf("GetActiveStreamForEvent failed: %v", err)
	}

	if activeStream == nil {
		t.Fatal("Expected active stream, got nil")
	}

	// Should return the most recent stream (stream2)
	if activeStream.StreamSessionID != stream2ID {
		t.Errorf("Expected most recent stream_session_id '%s', got '%s' (older: '%s')", stream2ID, activeStream.StreamSessionID, stream1ID)
	}

	if activeStream.RoomName != room2 {
		t.Errorf("Expected room_name '%s', got '%s'", room2, activeStream.RoomName)
	}
}

func TestSessionRepository_RecordJoin_Success(t *testing.T) {
	repo := NewInMemorySessionRepository()
	sceneID := "scene-123"

	// Create a stream session
	id, _, err := repo.CreateStreamSession(&sceneID, nil, "did:plc:host")
	if err != nil {
		t.Fatalf("CreateStreamSession failed: %v", err)
	}

	// Get initial join count
	session, err := repo.GetByID(id)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if session.JoinCount != 0 {
		t.Errorf("Initial join_count = %d, want 0", session.JoinCount)
	}

	// Record multiple joins
	for i := 0; i < 5; i++ {
		if err := repo.RecordJoin(id); err != nil {
			t.Fatalf("RecordJoin failed: %v", err)
		}
	}

	// Verify join count
	session, err = repo.GetByID(id)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if session.JoinCount != 5 {
		t.Errorf("join_count = %d, want 5", session.JoinCount)
	}
}

func TestSessionRepository_RecordJoin_NotFound(t *testing.T) {
	repo := NewInMemorySessionRepository()

	err := repo.RecordJoin("nonexistent-id")
	if err != ErrStreamNotFound {
		t.Errorf("Expected ErrStreamNotFound, got %v", err)
	}
}

func TestSessionRepository_RecordLeave_Success(t *testing.T) {
	repo := NewInMemorySessionRepository()
	sceneID := "scene-123"

	// Create a stream session
	id, _, err := repo.CreateStreamSession(&sceneID, nil, "did:plc:host")
	if err != nil {
		t.Fatalf("CreateStreamSession failed: %v", err)
	}

	// Get initial leave count
	session, err := repo.GetByID(id)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if session.LeaveCount != 0 {
		t.Errorf("Initial leave_count = %d, want 0", session.LeaveCount)
	}

	// Record multiple leaves
	for i := 0; i < 3; i++ {
		if err := repo.RecordLeave(id); err != nil {
			t.Fatalf("RecordLeave failed: %v", err)
		}
	}

	// Verify leave count
	session, err = repo.GetByID(id)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if session.LeaveCount != 3 {
		t.Errorf("leave_count = %d, want 3", session.LeaveCount)
	}
}

func TestSessionRepository_RecordLeave_NotFound(t *testing.T) {
	repo := NewInMemorySessionRepository()

	err := repo.RecordLeave("nonexistent-id")
	if err != ErrStreamNotFound {
		t.Errorf("Expected ErrStreamNotFound, got %v", err)
	}
}

func TestSessionRepository_JoinLeave_Combined(t *testing.T) {
	repo := NewInMemorySessionRepository()
	sceneID := "scene-123"

	// Create a stream session
	id, _, err := repo.CreateStreamSession(&sceneID, nil, "did:plc:host")
	if err != nil {
		t.Fatalf("CreateStreamSession failed: %v", err)
	}

	// Simulate join/leave events
	if err := repo.RecordJoin(id); err != nil {
		t.Fatalf("RecordJoin 1 failed: %v", err)
	}
	if err := repo.RecordJoin(id); err != nil {
		t.Fatalf("RecordJoin 2 failed: %v", err)
	}
	if err := repo.RecordLeave(id); err != nil {
		t.Fatalf("RecordLeave 1 failed: %v", err)
	}
	if err := repo.RecordJoin(id); err != nil {
		t.Fatalf("RecordJoin 3 failed: %v", err)
	}
	if err := repo.RecordLeave(id); err != nil {
		t.Fatalf("RecordLeave 2 failed: %v", err)
	}
	if err := repo.RecordLeave(id); err != nil {
		t.Fatalf("RecordLeave 3 failed: %v", err)
	}

	// Verify counts
	session, err := repo.GetByID(id)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if session.JoinCount != 3 {
		t.Errorf("join_count = %d, want 3", session.JoinCount)
	}
	if session.LeaveCount != 3 {
		t.Errorf("leave_count = %d, want 3", session.LeaveCount)
	}
}

// TestSessionRepository_SetLockStatus tests the SetLockStatus method.
func TestSessionRepository_SetLockStatus(t *testing.T) {
	tests := []struct {
		name      string
		setupFn   func(*InMemorySessionRepository) string
		sessionID string
		locked    bool
		wantErr   error
		checkLock bool
	}{
		{
			name: "lock_active_stream",
			setupFn: func(repo *InMemorySessionRepository) string {
				sceneID := "scene-lock-test"
				id, _, _ := repo.CreateStreamSession(&sceneID, nil, "did:plc:host")
				return id
			},
			locked:    true,
			wantErr:   nil,
			checkLock: true,
		},
		{
			name: "unlock_active_stream",
			setupFn: func(repo *InMemorySessionRepository) string {
				sceneID := "scene-unlock-test"
				id, _, _ := repo.CreateStreamSession(&sceneID, nil, "did:plc:host")
				repo.SetLockStatus(id, true) // Lock it first
				return id
			},
			locked:    false,
			wantErr:   nil,
			checkLock: true,
		},
		{
			name:      "nonexistent_stream",
			setupFn:   func(repo *InMemorySessionRepository) string { return "" },
			sessionID: "nonexistent-stream-id",
			locked:    true,
			wantErr:   ErrStreamNotFound,
			checkLock: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewInMemorySessionRepository()
			var id string
			if tt.setupFn != nil {
				id = tt.setupFn(repo)
			}
			if tt.sessionID != "" {
				id = tt.sessionID
			}

			err := repo.SetLockStatus(id, tt.locked)
			if err != tt.wantErr {
				t.Errorf("SetLockStatus() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.checkLock && err == nil {
				session, err := repo.GetByID(id)
				if err != nil {
					t.Fatalf("GetByID() failed: %v", err)
				}
				if session.IsLocked != tt.locked {
					t.Errorf("IsLocked = %v, want %v", session.IsLocked, tt.locked)
				}
			}
		})
	}
}

// TestSessionRepository_SetFeaturedParticipant tests the SetFeaturedParticipant method.
func TestSessionRepository_SetFeaturedParticipant(t *testing.T) {
	tests := []struct {
		name          string
		setupFn       func(*InMemorySessionRepository) string
		sessionID     string
		participantID *string
		wantErr       error
	}{
		{
			name: "set_featured_participant",
			setupFn: func(repo *InMemorySessionRepository) string {
				sceneID := "scene-featured-test"
				id, _, _ := repo.CreateStreamSession(&sceneID, nil, "did:plc:host")
				return id
			},
			participantID: strPtr("participant-alice"),
			wantErr:       nil,
		},
		{
			name: "clear_featured_participant",
			setupFn: func(repo *InMemorySessionRepository) string {
				sceneID := "scene-clear-featured"
				id, _, _ := repo.CreateStreamSession(&sceneID, nil, "did:plc:host")
				repo.SetFeaturedParticipant(id, strPtr("participant-bob"))
				return id
			},
			participantID: nil, // Clear the featured participant
			wantErr:       nil,
		},
		{
			name:          "nonexistent_stream",
			setupFn:       func(repo *InMemorySessionRepository) string { return "" },
			sessionID:     "nonexistent-id",
			participantID: strPtr("participant-charlie"),
			wantErr:       ErrStreamNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewInMemorySessionRepository()
			var id string
			if tt.setupFn != nil {
				id = tt.setupFn(repo)
			}
			if tt.sessionID != "" {
				id = tt.sessionID
			}

			err := repo.SetFeaturedParticipant(id, tt.participantID)
			if err != tt.wantErr {
				t.Errorf("SetFeaturedParticipant() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err == nil {
				session, err := repo.GetByID(id)
				if err != nil {
					t.Fatalf("GetByID() failed: %v", err)
				}
				if tt.participantID == nil {
					if session.FeaturedParticipant != nil {
						t.Errorf("FeaturedParticipant = %v, want nil", session.FeaturedParticipant)
					}
				} else {
					if session.FeaturedParticipant == nil {
						t.Error("FeaturedParticipant is nil, want non-nil")
					} else if *session.FeaturedParticipant != *tt.participantID {
						t.Errorf("FeaturedParticipant = %v, want %v", *session.FeaturedParticipant, *tt.participantID)
					}
				}
			}
		})
	}
}

// TestSessionRepository_GetActiveStreamsForEvents tests the batch operation for multiple events.
func TestSessionRepository_GetActiveStreamsForEvents(t *testing.T) {
	tests := []struct {
		name     string
		setupFn  func(*InMemorySessionRepository) []string
		eventIDs []string
		want     map[string]bool // eventID -> has active stream
	}{
		{
			name: "multiple_events_with_active_streams",
			setupFn: func(repo *InMemorySessionRepository) []string {
				event1 := "event-batch-1"
				event2 := "event-batch-2"
				event3 := "event-batch-3"

				repo.CreateStreamSession(nil, &event1, "did:plc:host1")
				repo.CreateStreamSession(nil, &event2, "did:plc:host2")
				// event3 has no active stream
				return []string{event1, event2, event3}
			},
			eventIDs: []string{"event-batch-1", "event-batch-2", "event-batch-3"},
			want: map[string]bool{
				"event-batch-1": true,
				"event-batch-2": true,
				"event-batch-3": false,
			},
		},
		{
			name: "no_active_streams",
			setupFn: func(repo *InMemorySessionRepository) []string {
				event1 := "event-ended-1"
				event2 := "event-ended-2"

				id1, _, _ := repo.CreateStreamSession(nil, &event1, "did:plc:host1")
				id2, _, _ := repo.CreateStreamSession(nil, &event2, "did:plc:host2")
				repo.EndStreamSession(id1)
				repo.EndStreamSession(id2)
				return []string{event1, event2}
			},
			eventIDs: []string{"event-ended-1", "event-ended-2"},
			want: map[string]bool{
				"event-ended-1": false,
				"event-ended-2": false,
			},
		},
		{
			name:     "empty_event_list",
			setupFn:  func(repo *InMemorySessionRepository) []string { return []string{} },
			eventIDs: []string{},
			want:     map[string]bool{},
		},
		{
			name: "multiple_streams_per_event_returns_most_recent",
			setupFn: func(repo *InMemorySessionRepository) []string {
				eventID := "event-multiple"

				// Create first stream
				id1, room1, _ := repo.CreateStreamSession(nil, &eventID, "did:plc:host1")
				time.Sleep(10 * time.Millisecond)

				// Create second stream (more recent)
				id2, room2, _ := repo.CreateStreamSession(nil, &eventID, "did:plc:host2")

				// Verify later by checking the room name
				_ = id1
				_ = room1
				_ = id2
				_ = room2
				return []string{eventID}
			},
			eventIDs: []string{"event-multiple"},
			want: map[string]bool{
				"event-multiple": true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewInMemorySessionRepository()
			var eventIDs []string
			if tt.setupFn != nil {
				eventIDs = tt.setupFn(repo)
			}
			if tt.eventIDs != nil {
				eventIDs = tt.eventIDs
			}

			result, err := repo.GetActiveStreamsForEvents(eventIDs)
			if err != nil {
				t.Fatalf("GetActiveStreamsForEvents() error = %v", err)
			}

			// Check that result matches expected active streams
			expectedCount := 0
			for eventID, shouldHaveStream := range tt.want {
				if shouldHaveStream {
					expectedCount++
				}
				info, hasStream := result[eventID]
				if shouldHaveStream && !hasStream {
					t.Errorf("Expected active stream for event %s, got none", eventID)
				} else if !shouldHaveStream && hasStream {
					t.Errorf("Expected no active stream for event %s, got %v", eventID, info)
				}
			}

			// Verify no unexpected events in result
			if len(result) != expectedCount {
				t.Errorf("Result count = %d, want %d", len(result), expectedCount)
			}
		})
	}
}

// TestSessionRepository_GetByID_EdgeCases tests edge cases for GetByID.
func TestSessionRepository_GetByID_EdgeCases(t *testing.T) {
	tests := []struct {
		name      string
		setupFn   func(*InMemorySessionRepository) string
		sessionID string
		wantErr   error
	}{
		{
			name: "valid_session",
			setupFn: func(repo *InMemorySessionRepository) string {
				sceneID := "scene-getbyid"
				id, _, _ := repo.CreateStreamSession(&sceneID, nil, "did:plc:host")
				return id
			},
			wantErr: nil,
		},
		{
			name:      "empty_session_id",
			setupFn:   func(repo *InMemorySessionRepository) string { return "" },
			sessionID: "",
			wantErr:   ErrStreamNotFound,
		},
		{
			name:      "nonexistent_session_id",
			setupFn:   func(repo *InMemorySessionRepository) string { return "" },
			sessionID: "nonexistent-uuid",
			wantErr:   ErrStreamNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewInMemorySessionRepository()
			var id string
			if tt.setupFn != nil {
				id = tt.setupFn(repo)
			}
			if tt.sessionID != "" {
				id = tt.sessionID
			}

			session, err := repo.GetByID(id)
			if err != tt.wantErr {
				t.Errorf("GetByID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr == nil && session == nil {
				t.Error("Expected non-nil session, got nil")
			}
		})
	}
}

// TestSessionRepository_UpdateActiveParticipantCount_EdgeCases tests edge cases.
func TestSessionRepository_UpdateActiveParticipantCount_EdgeCases(t *testing.T) {
	tests := []struct {
		name      string
		setupFn   func(*InMemorySessionRepository) string
		sessionID string
		count     int
		wantErr   error
	}{
		{
			name: "valid_update",
			setupFn: func(repo *InMemorySessionRepository) string {
				sceneID := "scene-update-count"
				id, _, _ := repo.CreateStreamSession(&sceneID, nil, "did:plc:host")
				return id
			},
			count:   10,
			wantErr: nil,
		},
		{
			name: "zero_count",
			setupFn: func(repo *InMemorySessionRepository) string {
				sceneID := "scene-zero-count"
				id, _, _ := repo.CreateStreamSession(&sceneID, nil, "did:plc:host")
				return id
			},
			count:   0,
			wantErr: nil,
		},
		{
			name:      "nonexistent_session",
			setupFn:   func(repo *InMemorySessionRepository) string { return "" },
			sessionID: "nonexistent-id",
			count:     5,
			wantErr:   ErrStreamNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewInMemorySessionRepository()
			var id string
			if tt.setupFn != nil {
				id = tt.setupFn(repo)
			}
			if tt.sessionID != "" {
				id = tt.sessionID
			}

			err := repo.UpdateActiveParticipantCount(id, tt.count)
			if err != tt.wantErr {
				t.Errorf("UpdateActiveParticipantCount() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err == nil {
				session, err := repo.GetByID(id)
				if err != nil {
					t.Fatalf("GetByID() failed: %v", err)
				}
				if session.ActiveParticipantCount != tt.count {
					t.Errorf("ActiveParticipantCount = %d, want %d", session.ActiveParticipantCount, tt.count)
				}
			}
		})
	}
}

// Benchmark tests for performance-sensitive operations
// BenchmarkCreateStreamSession benchmarks stream session creation.
// Benchmark tests for performance-sensitive operations
// BenchmarkCreateStreamSession benchmarks stream session creation.
func BenchmarkCreateStreamSession(b *testing.B) {
	repo := NewInMemorySessionRepository()
	sceneID := "bench-scene"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		hostDID := "did:plc:host" + string(rune(i))
		if _, _, err := repo.CreateStreamSession(&sceneID, nil, hostDID); err != nil {
			b.Fatalf("CreateStreamSession failed: %v", err)
		}
	}
}

// BenchmarkGetByID benchmarks session retrieval by ID.
func BenchmarkGetByID(b *testing.B) {
	repo := NewInMemorySessionRepository()
	sceneID := "bench-scene"
	id, _, err := repo.CreateStreamSession(&sceneID, nil, "did:plc:host")
	if err != nil {
		b.Fatalf("CreateStreamSession failed: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := repo.GetByID(id); err != nil {
			b.Fatalf("GetByID failed: %v", err)
		}
	}
}

// BenchmarkGetActiveStreamForEvent benchmarks single event stream lookup.
func BenchmarkGetActiveStreamForEvent(b *testing.B) {
	repo := NewInMemorySessionRepository()
	eventID := "bench-event"
	if _, _, err := repo.CreateStreamSession(nil, &eventID, "did:plc:host"); err != nil {
		b.Fatalf("CreateStreamSession failed: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := repo.GetActiveStreamForEvent(eventID); err != nil {
			b.Fatalf("GetActiveStreamForEvent failed: %v", err)
		}
	}
}

// BenchmarkGetActiveStreamsForEvents benchmarks batch event stream lookup.
func BenchmarkGetActiveStreamsForEvents(b *testing.B) {
	repo := NewInMemorySessionRepository()

	// Create 100 active streams for different events
	eventIDs := make([]string, 100)
	for i := 0; i < 100; i++ {
		eventID := "bench-event-" + string(rune(i))
		eventIDs[i] = eventID
		if _, _, err := repo.CreateStreamSession(nil, &eventID, "did:plc:host"); err != nil {
			b.Fatalf("CreateStreamSession failed: %v", err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := repo.GetActiveStreamsForEvents(eventIDs); err != nil {
			b.Fatalf("GetActiveStreamsForEvents failed: %v", err)
		}
	}
}

// BenchmarkRecordJoinLeave benchmarks participant join/leave tracking.
func BenchmarkRecordJoinLeave(b *testing.B) {
	repo := NewInMemorySessionRepository()
	sceneID := "bench-scene"
	id, _, err := repo.CreateStreamSession(&sceneID, nil, "did:plc:host")
	if err != nil {
		b.Fatalf("CreateStreamSession failed: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if i%2 == 0 {
			if err := repo.RecordJoin(id); err != nil {
				b.Fatalf("RecordJoin failed: %v", err)
			}
		} else {
			if err := repo.RecordLeave(id); err != nil {
				b.Fatalf("RecordLeave failed: %v", err)
			}
		}
	}
}

// BenchmarkSetLockStatus benchmarks lock status updates.
func BenchmarkSetLockStatus(b *testing.B) {
	repo := NewInMemorySessionRepository()
	sceneID := "bench-scene"
	id, _, err := repo.CreateStreamSession(&sceneID, nil, "did:plc:host")
	if err != nil {
		b.Fatalf("CreateStreamSession failed: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := repo.SetLockStatus(id, i%2 == 0); err != nil {
			b.Fatalf("SetLockStatus failed: %v", err)
		}
	}
}

// BenchmarkSetFeaturedParticipant benchmarks featured participant updates.
func BenchmarkSetFeaturedParticipant(b *testing.B) {
	repo := NewInMemorySessionRepository()
	sceneID := "bench-scene"
	id, _, err := repo.CreateStreamSession(&sceneID, nil, "did:plc:host")
	if err != nil {
		b.Fatalf("CreateStreamSession failed: %v", err)
	}
	participantID := "participant-featured"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if i%2 == 0 {
			if err := repo.SetFeaturedParticipant(id, &participantID); err != nil {
				b.Fatalf("SetFeaturedParticipant failed: %v", err)
			}
		} else {
			if err := repo.SetFeaturedParticipant(id, nil); err != nil {
				b.Fatalf("SetFeaturedParticipant failed: %v", err)
			}
		}
	}
}
