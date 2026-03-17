package livekit

import (
	"context"
	"testing"
)

func TestNewRoomService(t *testing.T) {
	tests := []struct {
		name      string
		url       string
		apiKey    string
		apiSecret string
		wantNil   bool
	}{
		{
			name:      "valid configuration",
			url:       "wss://livekit.example.com",
			apiKey:    "APIkey123",
			apiSecret: "secret456",
			wantNil:   false,
		},
		{
			name:      "empty URL",
			url:       "",
			apiKey:    "APIkey123",
			apiSecret: "secret456",
			wantNil:   true,
		},
		{
			name:      "empty API key",
			url:       "wss://livekit.example.com",
			apiKey:    "",
			apiSecret: "secret456",
			wantNil:   true,
		},
		{
			name:      "empty API secret",
			url:       "wss://livekit.example.com",
			apiKey:    "APIkey123",
			apiSecret: "",
			wantNil:   true,
		},
		{
			name:      "all empty",
			url:       "",
			apiKey:    "",
			apiSecret: "",
			wantNil:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewRoomService(tt.url, tt.apiKey, tt.apiSecret)
			if tt.wantNil && svc != nil {
				t.Error("expected nil RoomService for empty config")
			}
			if !tt.wantNil && svc == nil {
				t.Error("expected non-nil RoomService for valid config")
			}
		})
	}
}

// nilRoomService creates a RoomService with nil roomClient for guard-clause testing.
func nilRoomService() *RoomService {
	return &RoomService{roomClient: nil}
}

func TestRoomService_CreateRoom_NilClient(t *testing.T) {
	svc := nilRoomService()
	_, err := svc.CreateRoom(context.Background(), "test-room", 300, 10)
	if err != ErrRoomServiceNotConfigured {
		t.Errorf("expected ErrRoomServiceNotConfigured, got %v", err)
	}
}

func TestRoomService_DeleteRoom_NilClient(t *testing.T) {
	svc := nilRoomService()
	err := svc.DeleteRoom(context.Background(), "test-room")
	if err != ErrRoomServiceNotConfigured {
		t.Errorf("expected ErrRoomServiceNotConfigured, got %v", err)
	}
}

func TestRoomService_GetRoom_NilClient(t *testing.T) {
	svc := nilRoomService()
	_, err := svc.GetRoom(context.Background(), "test-room")
	if err != ErrRoomServiceNotConfigured {
		t.Errorf("expected ErrRoomServiceNotConfigured, got %v", err)
	}
}

func TestRoomService_MuteParticipantTrack_NilClient(t *testing.T) {
	svc := nilRoomService()
	err := svc.MuteParticipantTrack(context.Background(), "room", "participant", "track-sid", true)
	if err != ErrRoomServiceNotConfigured {
		t.Errorf("expected ErrRoomServiceNotConfigured, got %v", err)
	}
}

func TestRoomService_RemoveParticipant_NilClient(t *testing.T) {
	svc := nilRoomService()
	err := svc.RemoveParticipant(context.Background(), "room", "participant")
	if err != ErrRoomServiceNotConfigured {
		t.Errorf("expected ErrRoomServiceNotConfigured, got %v", err)
	}
}

func TestRoomService_UpdateParticipantMetadata_NilClient(t *testing.T) {
	svc := nilRoomService()
	err := svc.UpdateParticipantMetadata(context.Background(), "room", "participant", `{"featured": true}`)
	if err != ErrRoomServiceNotConfigured {
		t.Errorf("expected ErrRoomServiceNotConfigured, got %v", err)
	}
}

func TestRoomService_UpdateRoomMetadata_NilClient(t *testing.T) {
	svc := nilRoomService()
	err := svc.UpdateRoomMetadata(context.Background(), "room", `{"locked": true}`)
	if err != ErrRoomServiceNotConfigured {
		t.Errorf("expected ErrRoomServiceNotConfigured, got %v", err)
	}
}

func TestRoomService_GetParticipant_NilClient(t *testing.T) {
	svc := nilRoomService()
	_, err := svc.GetParticipant(context.Background(), "room", "participant")
	if err != ErrRoomServiceNotConfigured {
		t.Errorf("expected ErrRoomServiceNotConfigured, got %v", err)
	}
}

func TestRoomService_ListParticipants_NilClient(t *testing.T) {
	svc := nilRoomService()
	_, err := svc.ListParticipants(context.Background(), "room")
	if err != ErrRoomServiceNotConfigured {
		t.Errorf("expected ErrRoomServiceNotConfigured, got %v", err)
	}
}

func TestRoomService_GetParticipantStats_NilClient(t *testing.T) {
	svc := nilRoomService()
	_, err := svc.GetParticipantStats(context.Background(), "room", "participant")
	if err != ErrRoomServiceNotConfigured {
		t.Errorf("expected ErrRoomServiceNotConfigured, got %v", err)
	}
}

func TestRoomService_GetAllParticipantStats_NilClient(t *testing.T) {
	svc := nilRoomService()
	_, err := svc.GetAllParticipantStats(context.Background(), "room")
	if err != ErrRoomServiceNotConfigured {
		t.Errorf("expected ErrRoomServiceNotConfigured, got %v", err)
	}
}
