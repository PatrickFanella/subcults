// Package livekit provides utilities for LiveKit integration.
package livekit

import (
	"context"
	"errors"
	"fmt"

	"github.com/livekit/protocol/livekit"
	lksdk "github.com/livekit/server-sdk-go/v2"
)

var (
	// ErrRoomServiceNotConfigured is returned when room service operations are attempted without proper configuration.
	ErrRoomServiceNotConfigured = errors.New("livekit room service not configured")

	// ErrRoomNotFound is returned when a requested room does not exist in LiveKit.
	ErrRoomNotFound = errors.New("room not found")
)

// RoomService provides operations for managing LiveKit rooms.
type RoomService struct {
	roomClient *lksdk.RoomServiceClient
	apiKey     string
	apiSecret  string
	url        string
}

// NewRoomService creates a new RoomService with the given configuration.
// Returns nil if apiKey, apiSecret, or url is empty (room control will not be available).
func NewRoomService(url, apiKey, apiSecret string) *RoomService {
	if url == "" || apiKey == "" || apiSecret == "" {
		return nil
	}

	roomClient := lksdk.NewRoomServiceClient(url, apiKey, apiSecret)

	return &RoomService{
		roomClient: roomClient,
		apiKey:     apiKey,
		apiSecret:  apiSecret,
		url:        url,
	}
}

// CreateRoom creates a new LiveKit room with the specified configuration.
// emptyTimeout is the duration in seconds after which an empty room will be automatically closed (0 = no timeout).
// maxParticipants is the maximum number of participants allowed (0 = unlimited).
func (s *RoomService) CreateRoom(ctx context.Context, roomName string, emptyTimeout, maxParticipants uint32) (*livekit.Room, error) {
	if s.roomClient == nil {
		return nil, ErrRoomServiceNotConfigured
	}

	req := &livekit.CreateRoomRequest{
		Name:            roomName,
		EmptyTimeout:    emptyTimeout,
		MaxParticipants: maxParticipants,
	}

	room, err := s.roomClient.CreateRoom(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create room: %w", err)
	}

	return room, nil
}

// DeleteRoom deletes a LiveKit room, disconnecting all participants.
func (s *RoomService) DeleteRoom(ctx context.Context, roomName string) error {
	if s.roomClient == nil {
		return ErrRoomServiceNotConfigured
	}

	req := &livekit.DeleteRoomRequest{
		Room: roomName,
	}

	_, err := s.roomClient.DeleteRoom(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to delete room: %w", err)
	}

	return nil
}

// GetRoom retrieves information about a specific LiveKit room.
// Returns ErrRoomNotFound if the room does not exist in LiveKit.
func (s *RoomService) GetRoom(ctx context.Context, roomName string) (*livekit.Room, error) {
	if s.roomClient == nil {
		return nil, ErrRoomServiceNotConfigured
	}

	resp, err := s.roomClient.ListRooms(ctx, &livekit.ListRoomsRequest{
		Names: []string{roomName},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get room: %w", err)
	}

	if len(resp.Rooms) == 0 {
		return nil, ErrRoomNotFound
	}

	return resp.Rooms[0], nil
}

// MuteParticipantTrack mutes a specific participant's track in a room.
func (s *RoomService) MuteParticipantTrack(ctx context.Context, roomName, participantIdentity string, trackSID string, muted bool) error {
	if s.roomClient == nil {
		return ErrRoomServiceNotConfigured
	}

	req := &livekit.MuteRoomTrackRequest{
		Room:     roomName,
		Identity: participantIdentity,
		TrackSid: trackSID,
		Muted:    muted,
	}

	_, err := s.roomClient.MutePublishedTrack(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to mute track: %w", err)
	}

	return nil
}

// RemoveParticipant removes (kicks) a participant from a room.
func (s *RoomService) RemoveParticipant(ctx context.Context, roomName, participantIdentity string) error {
	if s.roomClient == nil {
		return ErrRoomServiceNotConfigured
	}

	req := &livekit.RoomParticipantIdentity{
		Room:     roomName,
		Identity: participantIdentity,
	}

	_, err := s.roomClient.RemoveParticipant(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to remove participant: %w", err)
	}

	return nil
}

// UpdateParticipantMetadata updates metadata for a participant (used for featured participant).
func (s *RoomService) UpdateParticipantMetadata(ctx context.Context, roomName, participantIdentity, metadata string) error {
	if s.roomClient == nil {
		return ErrRoomServiceNotConfigured
	}

	req := &livekit.UpdateParticipantRequest{
		Room:     roomName,
		Identity: participantIdentity,
		Metadata: metadata,
	}

	_, err := s.roomClient.UpdateParticipant(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to update participant metadata: %w", err)
	}

	return nil
}

// UpdateRoomMetadata updates room metadata (used for lock status).
func (s *RoomService) UpdateRoomMetadata(ctx context.Context, roomName, metadata string) error {
	if s.roomClient == nil {
		return ErrRoomServiceNotConfigured
	}

	req := &livekit.UpdateRoomMetadataRequest{
		Room:     roomName,
		Metadata: metadata,
	}

	_, err := s.roomClient.UpdateRoomMetadata(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to update room metadata: %w", err)
	}

	return nil
}

// GetParticipant gets information about a specific participant.
func (s *RoomService) GetParticipant(ctx context.Context, roomName, participantIdentity string) (*livekit.ParticipantInfo, error) {
	if s.roomClient == nil {
		return nil, ErrRoomServiceNotConfigured
	}

	req := &livekit.RoomParticipantIdentity{
		Room:     roomName,
		Identity: participantIdentity,
	}

	participant, err := s.roomClient.GetParticipant(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get participant: %w", err)
	}

	return participant, nil
}

// ListParticipants lists all participants in a room.
func (s *RoomService) ListParticipants(ctx context.Context, roomName string) ([]*livekit.ParticipantInfo, error) {
	if s.roomClient == nil {
		return nil, ErrRoomServiceNotConfigured
	}

	req := &livekit.ListParticipantsRequest{
		Room: roomName,
	}

	resp, err := s.roomClient.ListParticipants(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to list participants: %w", err)
	}

	return resp.Participants, nil
}

// GetParticipantStats retrieves detailed statistics for a specific participant.
// Stats include bitrate, jitter, packet loss, and other quality metrics.
// Returns ErrRoomServiceNotConfigured if room service is not initialized.
func (s *RoomService) GetParticipantStats(ctx context.Context, roomName, participantIdentity string) (*livekit.ParticipantInfo, error) {
	if s.roomClient == nil {
		return nil, ErrRoomServiceNotConfigured
	}

	// Use GetParticipant which includes track statistics
	participant, err := s.GetParticipant(ctx, roomName, participantIdentity)
	if err != nil {
		return nil, err
	}

	return participant, nil
}

// GetAllParticipantStats retrieves statistics for all participants in a room.
// This is useful for monitoring overall room quality and detecting issues.
func (s *RoomService) GetAllParticipantStats(ctx context.Context, roomName string) ([]*livekit.ParticipantInfo, error) {
	if s.roomClient == nil {
		return nil, ErrRoomServiceNotConfigured
	}

	// ListParticipants returns full ParticipantInfo with track stats
	participants, err := s.ListParticipants(ctx, roomName)
	if err != nil {
		return nil, err
	}

	return participants, nil
}
