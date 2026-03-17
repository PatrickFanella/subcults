package stream

import (
	"strings"
	"testing"
	"time"
)

func TestGenerateParticipantID(t *testing.T) {
	tests := []struct {
		name         string
		did          string
		expectedBase string // Expected base (may be truncated)
		checkLen     bool   // Check if truncated to 48 chars
	}{
		{
			name:         "standard_did_format",
			did:          "did:plc:abc123xyz",
			expectedBase: "user-abc123xyz",
			checkLen:     false,
		},
		{
			name:         "long_identifier",
			did:          "did:plc:verylongidentifier1234567890abcdefghijklmnopqrstuvwxyz",
			expectedBase: "user-verylongidentifier1234567890abcdefghijklmnop",
			checkLen:     true, // Should be truncated to 48 chars + "user-" prefix
		},
		{
			name:         "short_did_two_parts",
			did:          "did:abc",
			expectedBase: "user-did:abc", // Falls back to using whole DID since only 2 parts
			checkLen:     false,
		},
		{
			name:         "malformed_did_no_colons",
			did:          "malformed",
			expectedBase: "user-malformed",
			checkLen:     false,
		},
		{
			name:         "did_with_multiple_colons",
			did:          "did:method:sub:identifier",
			expectedBase: "user-identifier",
			checkLen:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateParticipantID(tt.did)

			// Verify it starts with "user-"
			if !strings.HasPrefix(result, "user-") {
				t.Errorf("Expected participant ID to start with 'user-', got %s", result)
			}

			// For truncation tests, just verify the prefix and length
			if tt.checkLen {
				if !strings.HasPrefix(result, tt.expectedBase) {
					t.Errorf("Expected to start with %s, got %s", tt.expectedBase, result)
				}
				// Verify identifier part is truncated to 48 chars
				identifierPart := strings.TrimPrefix(result, "user-")
				if len(identifierPart) > 48 {
					t.Errorf("Expected identifier part <= 48 chars, got %d", len(identifierPart))
				}
			} else {
				if result != tt.expectedBase {
					t.Errorf("Expected %s, got %s", tt.expectedBase, result)
				}
			}

			// Verify total length is reasonable (max 53 chars: "user-" + 48)
			if len(result) > 53 {
				t.Errorf("Expected participant ID length <= 53, got %d", len(result))
			}
		})
	}
}

func TestGenerateParticipantID_Deterministic(t *testing.T) {
	did := "did:plc:test123"

	// Call multiple times and verify same result
	id1 := GenerateParticipantID(did)
	id2 := GenerateParticipantID(did)
	id3 := GenerateParticipantID(did)

	if id1 != id2 || id2 != id3 {
		t.Errorf("Expected deterministic IDs: %s, %s, %s", id1, id2, id3)
	}
}

func TestGenerateParticipantID_Uniqueness(t *testing.T) {
	dids := []string{
		"did:plc:alice123",
		"did:plc:bob456",
		"did:plc:charlie789",
	}

	ids := make(map[string]bool)
	for _, did := range dids {
		id := GenerateParticipantID(did)
		if ids[id] {
			t.Errorf("Expected unique IDs, got duplicate: %s", id)
		}
		ids[id] = true
	}

	if len(ids) != len(dids) {
		t.Errorf("Expected %d unique IDs, got %d", len(dids), len(ids))
	}
}

// TestParticipant_IsActive tests the IsActive method.
func TestParticipant_IsActive(t *testing.T) {
	now := time.Now()
	leftTime := now.Add(-1 * time.Hour)

	tests := []struct {
		name   string
		leftAt *time.Time
		want   bool
	}{
		{
			name:   "active_participant",
			leftAt: nil,
			want:   true,
		},
		{
			name:   "left_participant",
			leftAt: &leftTime,
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Participant{
				ID:              "participant-123",
				StreamSessionID: "session-456",
				ParticipantID:   "user-alice",
				UserDID:         "did:plc:alice123",
				JoinedAt:        now,
				LeftAt:          tt.leftAt,
			}

			got := p.IsActive()
			if got != tt.want {
				t.Errorf("IsActive() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestParticipantStateEvent_Structure tests the ParticipantStateEvent structure.
func TestParticipantStateEvent_Structure(t *testing.T) {
	event := &ParticipantStateEvent{
		Type:            "participant_joined",
		StreamSessionID: "session-789",
		ParticipantID:   "user-bob",
		UserDID:         "did:plc:bob456",
		Timestamp:       time.Now(),
		IsReconnection:  false,
		ActiveCount:     5,
	}

	if event.Type != "participant_joined" {
		t.Errorf("Type = %s, want participant_joined", event.Type)
	}
	if event.ActiveCount != 5 {
		t.Errorf("ActiveCount = %d, want 5", event.ActiveCount)
	}
	if event.IsReconnection {
		t.Error("IsReconnection should be false")
	}
}

// TestParticipantStateEvent_Reconnection tests reconnection event behavior.
// This test will be expanded when reconnection behavior logic is implemented.
func TestParticipantStateEvent_Reconnection(t *testing.T) {
	t.Skip("TODO: implement reconnection behavior test when reconnection logic is added")
}
