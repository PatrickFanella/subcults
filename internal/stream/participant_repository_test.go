package stream

import (
	"fmt"
	"testing"
	"time"
)

func TestInMemoryParticipantRepository_RecordJoin(t *testing.T) {
	sessionRepo := NewInMemorySessionRepository()
	repo := NewInMemoryParticipantRepository(sessionRepo)

	// Create a test stream session
	sceneID := "scene-123"
	hostDID := "did:plc:host123"
	streamID, _, err := sessionRepo.CreateStreamSession(&sceneID, nil, hostDID)
	if err != nil {
		t.Fatalf("Failed to create stream session: %v", err)
	}

	participantID := "user-abc123"
	userDID := "did:plc:abc123"

	// Test initial join
	t.Run("initial_join", func(t *testing.T) {
		participant, isReconnection, err := repo.RecordJoin(streamID, participantID, userDID)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if isReconnection {
			t.Error("Expected isReconnection to be false for initial join")
		}
		if participant.ParticipantID != participantID {
			t.Errorf("Expected participant_id %s, got %s", participantID, participant.ParticipantID)
		}
		if participant.UserDID != userDID {
			t.Errorf("Expected user_did %s, got %s", userDID, participant.UserDID)
		}
		if !participant.IsActive() {
			t.Error("Expected participant to be active")
		}
		if participant.ReconnectionCount != 0 {
			t.Errorf("Expected reconnection count 0, got %d", participant.ReconnectionCount)
		}

		// Verify active count updated
		count, err := repo.GetActiveCount(streamID)
		if err != nil {
			t.Fatalf("Failed to get active count: %v", err)
		}
		if count != 1 {
			t.Errorf("Expected active count 1, got %d", count)
		}

		// Verify denormalized count on session
		session, err := sessionRepo.GetByID(streamID)
		if err != nil {
			t.Fatalf("Failed to get session: %v", err)
		}
		if session.ActiveParticipantCount != 1 {
			t.Errorf("Expected session active_participant_count 1, got %d", session.ActiveParticipantCount)
		}
	})

	// Test duplicate join (should fail)
	t.Run("duplicate_join", func(t *testing.T) {
		_, _, err := repo.RecordJoin(streamID, participantID, userDID)
		if err != ErrParticipantAlreadyActive {
			t.Errorf("Expected ErrParticipantAlreadyActive, got %v", err)
		}
	})

	// Test leave and rejoin (reconnection)
	t.Run("reconnection", func(t *testing.T) {
		// First, leave
		err := repo.RecordLeave(streamID, participantID)
		if err != nil {
			t.Fatalf("Failed to record leave: %v", err)
		}

		// Verify active count decreased
		count, err := repo.GetActiveCount(streamID)
		if err != nil {
			t.Fatalf("Failed to get active count: %v", err)
		}
		if count != 0 {
			t.Errorf("Expected active count 0 after leave, got %d", count)
		}

		// Rejoin (should be marked as reconnection)
		participant, isReconnection, err := repo.RecordJoin(streamID, participantID, userDID)
		if err != nil {
			t.Fatalf("Expected no error on reconnection, got %v", err)
		}
		if !isReconnection {
			t.Error("Expected isReconnection to be true for rejoin")
		}
		if participant.ReconnectionCount != 1 {
			t.Errorf("Expected reconnection count 1, got %d", participant.ReconnectionCount)
		}

		// Verify active count increased
		count, err = repo.GetActiveCount(streamID)
		if err != nil {
			t.Fatalf("Failed to get active count: %v", err)
		}
		if count != 1 {
			t.Errorf("Expected active count 1 after rejoin, got %d", count)
		}
	})
}

func TestInMemoryParticipantRepository_RecordLeave(t *testing.T) {
	sessionRepo := NewInMemorySessionRepository()
	repo := NewInMemoryParticipantRepository(sessionRepo)

	// Create a test stream session
	sceneID := "scene-123"
	hostDID := "did:plc:host123"
	streamID, _, err := sessionRepo.CreateStreamSession(&sceneID, nil, hostDID)
	if err != nil {
		t.Fatalf("Failed to create stream session: %v", err)
	}

	participantID := "user-abc123"
	userDID := "did:plc:abc123"

	// Test leaving without joining
	t.Run("leave_without_join", func(t *testing.T) {
		err := repo.RecordLeave(streamID, participantID)
		if err != ErrParticipantNotFound {
			t.Errorf("Expected ErrParticipantNotFound, got %v", err)
		}
	})

	// Test normal leave
	t.Run("normal_leave", func(t *testing.T) {
		// First join
		_, _, err := repo.RecordJoin(streamID, participantID, userDID)
		if err != nil {
			t.Fatalf("Failed to join: %v", err)
		}

		// Then leave
		err = repo.RecordLeave(streamID, participantID)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		// Verify participant is no longer active
		active, err := repo.GetActiveParticipants(streamID)
		if err != nil {
			t.Fatalf("Failed to get active participants: %v", err)
		}
		if len(active) != 0 {
			t.Errorf("Expected 0 active participants, got %d", len(active))
		}
	})

	// Test double leave (should fail)
	t.Run("double_leave", func(t *testing.T) {
		err := repo.RecordLeave(streamID, participantID)
		if err != ErrParticipantNotFound {
			t.Errorf("Expected ErrParticipantNotFound on double leave, got %v", err)
		}
	})
}

func TestInMemoryParticipantRepository_GetActiveParticipants(t *testing.T) {
	sessionRepo := NewInMemorySessionRepository()
	repo := NewInMemoryParticipantRepository(sessionRepo)

	// Create a test stream session
	sceneID := "scene-123"
	hostDID := "did:plc:host123"
	streamID, _, err := sessionRepo.CreateStreamSession(&sceneID, nil, hostDID)
	if err != nil {
		t.Fatalf("Failed to create stream session: %v", err)
	}

	// Test with no participants
	t.Run("no_participants", func(t *testing.T) {
		active, err := repo.GetActiveParticipants(streamID)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if len(active) != 0 {
			t.Errorf("Expected 0 active participants, got %d", len(active))
		}
	})

	// Add multiple participants
	t.Run("multiple_participants", func(t *testing.T) {
		participants := []struct {
			participantID string
			userDID       string
		}{
			{"user-abc123", "did:plc:abc123"},
			{"user-def456", "did:plc:def456"},
			{"user-ghi789", "did:plc:ghi789"},
		}

		// Join all participants
		for _, p := range participants {
			_, _, err := repo.RecordJoin(streamID, p.participantID, p.userDID)
			if err != nil {
				t.Fatalf("Failed to join participant %s: %v", p.participantID, err)
			}
		}

		// Get active participants
		active, err := repo.GetActiveParticipants(streamID)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if len(active) != len(participants) {
			t.Errorf("Expected %d active participants, got %d", len(participants), len(active))
		}

		// Verify all participants are present
		foundIDs := make(map[string]bool)
		for _, p := range active {
			foundIDs[p.ParticipantID] = true
		}
		for _, p := range participants {
			if !foundIDs[p.participantID] {
				t.Errorf("Expected to find participant %s in active list", p.participantID)
			}
		}

		// Leave one participant
		err = repo.RecordLeave(streamID, participants[0].participantID)
		if err != nil {
			t.Fatalf("Failed to leave: %v", err)
		}

		// Verify active count decreased
		active, err = repo.GetActiveParticipants(streamID)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if len(active) != len(participants)-1 {
			t.Errorf("Expected %d active participants after leave, got %d", len(participants)-1, len(active))
		}
	})
}

func TestInMemoryParticipantRepository_GetParticipantHistory(t *testing.T) {
	sessionRepo := NewInMemorySessionRepository()
	repo := NewInMemoryParticipantRepository(sessionRepo)

	// Create a test stream session
	sceneID := "scene-123"
	hostDID := "did:plc:host123"
	streamID, _, err := sessionRepo.CreateStreamSession(&sceneID, nil, hostDID)
	if err != nil {
		t.Fatalf("Failed to create stream session: %v", err)
	}

	participants := []struct {
		participantID string
		userDID       string
	}{
		{"user-abc123", "did:plc:abc123"},
		{"user-def456", "did:plc:def456"},
		{"user-ghi789", "did:plc:ghi789"},
	}

	// Join all participants with small delays to ensure distinct timestamps
	for _, p := range participants {
		_, _, err := repo.RecordJoin(streamID, p.participantID, p.userDID)
		if err != nil {
			t.Fatalf("Failed to join participant %s: %v", p.participantID, err)
		}
		time.Sleep(10 * time.Millisecond)
	}

	// Leave first participant
	err = repo.RecordLeave(streamID, participants[0].participantID)
	if err != nil {
		t.Fatalf("Failed to leave: %v", err)
	}

	// Get history
	history, err := repo.GetParticipantHistory(streamID)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Should have all 3 participants (including the one who left)
	if len(history) != len(participants) {
		t.Errorf("Expected %d participants in history, got %d", len(participants), len(history))
	}

	// Verify sorted by joined_at descending (most recent first)
	for i := 0; i < len(history)-1; i++ {
		if history[i].JoinedAt.Before(history[i+1].JoinedAt) {
			t.Error("Expected history to be sorted by joined_at descending")
		}
	}

	// Verify first participant has left_at set
	foundLeft := false
	for _, p := range history {
		if p.ParticipantID == participants[0].participantID {
			if p.LeftAt == nil {
				t.Error("Expected first participant to have left_at set")
			}
			foundLeft = true
		}
	}
	if !foundLeft {
		t.Error("Expected to find first participant in history")
	}
}

func TestInMemoryParticipantRepository_GetActiveCount(t *testing.T) {
	sessionRepo := NewInMemorySessionRepository()
	repo := NewInMemoryParticipantRepository(sessionRepo)

	// Create a test stream session
	sceneID := "scene-123"
	hostDID := "did:plc:host123"
	streamID, _, err := sessionRepo.CreateStreamSession(&sceneID, nil, hostDID)
	if err != nil {
		t.Fatalf("Failed to create stream session: %v", err)
	}

	// Test with no participants
	count, err := repo.GetActiveCount(streamID)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if count != 0 {
		t.Errorf("Expected count 0, got %d", count)
	}

	// Add participants and verify count
	for i := 1; i <= 5; i++ {
		participantID := fmt.Sprintf("user-%d", i)
		userDID := fmt.Sprintf("did:plc:%d", i)
		_, _, err := repo.RecordJoin(streamID, participantID, userDID)
		if err != nil {
			t.Fatalf("Failed to join participant: %v", err)
		}

		count, err := repo.GetActiveCount(streamID)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if count != i {
			t.Errorf("Expected count %d, got %d", i, count)
		}
	}

	// Leave one participant
	err = repo.RecordLeave(streamID, "user-1")
	if err != nil {
		t.Fatalf("Failed to leave: %v", err)
	}

	count, err = repo.GetActiveCount(streamID)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if count != 4 {
		t.Errorf("Expected count 4 after leave, got %d", count)
	}
}
