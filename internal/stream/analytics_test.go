// Package stream provides tests for analytics functionality.
package stream

import (
	"testing"
	"time"
)

func TestInMemoryAnalyticsRepository_RecordParticipantEvent(t *testing.T) {
	sessionRepo := NewInMemorySessionRepository()
	repo := NewInMemoryAnalyticsRepository(sessionRepo)

	streamID := "test-stream-123"
	participantDID := "did:plc:user1"
	geo := "abcd"

	err := repo.RecordParticipantEvent(streamID, participantDID, "join", &geo)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	events, err := repo.GetParticipantEvents(streamID)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(events) != 1 {
		t.Fatalf("Expected 1 event, got %d", len(events))
	}

	if events[0].EventType != "join" {
		t.Errorf("Expected event type 'join', got %s", events[0].EventType)
	}

	if events[0].ParticipantDID != participantDID {
		t.Errorf("Expected participant %s, got %s", participantDID, events[0].ParticipantDID)
	}

	if events[0].GeohashPrefix == nil || *events[0].GeohashPrefix != geo {
		t.Errorf("Expected geohash prefix %s, got %v", geo, events[0].GeohashPrefix)
	}
}

func TestInMemoryAnalyticsRepository_RecordParticipantEvent_InvalidType(t *testing.T) {
	sessionRepo := NewInMemorySessionRepository()
	repo := NewInMemoryAnalyticsRepository(sessionRepo)

	err := repo.RecordParticipantEvent("stream-1", "did:plc:user1", "invalid", nil)
	if err == nil {
		t.Fatal("Expected error for invalid event type, got nil")
	}
}

func TestInMemoryAnalyticsRepository_GetParticipantEvents_Empty(t *testing.T) {
	sessionRepo := NewInMemorySessionRepository()
	repo := NewInMemoryAnalyticsRepository(sessionRepo)

	events, err := repo.GetParticipantEvents("non-existent-stream")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(events) != 0 {
		t.Errorf("Expected 0 events, got %d", len(events))
	}
}

func TestInMemoryAnalyticsRepository_GetParticipantEvents_Ordering(t *testing.T) {
	sessionRepo := NewInMemorySessionRepository()
	repo := NewInMemoryAnalyticsRepository(sessionRepo)

	streamID := "stream-1"

	// Record events with small delays to ensure different timestamps
	_ = repo.RecordParticipantEvent(streamID, "user1", "join", nil)
	time.Sleep(1 * time.Millisecond)
	_ = repo.RecordParticipantEvent(streamID, "user2", "join", nil)
	time.Sleep(1 * time.Millisecond)
	_ = repo.RecordParticipantEvent(streamID, "user1", "leave", nil)

	events, err := repo.GetParticipantEvents(streamID)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(events) != 3 {
		t.Fatalf("Expected 3 events, got %d", len(events))
	}

	// Verify ordering by occurred_at
	for i := 1; i < len(events); i++ {
		if events[i].OccurredAt.Before(events[i-1].OccurredAt) {
			t.Errorf("Events not ordered correctly: event %d occurred before event %d", i, i-1)
		}
	}
}

func TestInMemoryAnalyticsRepository_ComputeAnalytics_NoEvents(t *testing.T) {
	sessionRepo := NewInMemorySessionRepository()
	repo := NewInMemoryAnalyticsRepository(sessionRepo)

	// Create a stream session
	sceneID := "scene-1"
	streamID, _, err := sessionRepo.CreateStreamSession(&sceneID, nil, "did:plc:host")
	if err != nil {
		t.Fatalf("Failed to create stream: %v", err)
	}

	// End the stream
	time.Sleep(10 * time.Millisecond)
	err = sessionRepo.EndStreamSession(streamID)
	if err != nil {
		t.Fatalf("Failed to end stream: %v", err)
	}

	// Compute analytics
	analytics, err := repo.ComputeAnalytics(streamID)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify metrics for stream with no participants
	if analytics.PeakConcurrentListeners != 0 {
		t.Errorf("Expected 0 peak listeners, got %d", analytics.PeakConcurrentListeners)
	}

	if analytics.TotalUniqueParticipants != 0 {
		t.Errorf("Expected 0 unique participants, got %d", analytics.TotalUniqueParticipants)
	}

	if analytics.TotalJoinAttempts != 0 {
		t.Errorf("Expected 0 join attempts, got %d", analytics.TotalJoinAttempts)
	}

	if analytics.EngagementLagSeconds != nil {
		t.Errorf("Expected nil engagement lag, got %v", *analytics.EngagementLagSeconds)
	}

	if analytics.AvgListenDurationSeconds != nil {
		t.Errorf("Expected nil average duration, got %v", *analytics.AvgListenDurationSeconds)
	}
}

func TestInMemoryAnalyticsRepository_ComputeAnalytics_WithEvents(t *testing.T) {
	sessionRepo := NewInMemorySessionRepository()
	repo := NewInMemoryAnalyticsRepository(sessionRepo)

	// Create a stream session
	sceneID := "scene-1"
	streamID, _, err := sessionRepo.CreateStreamSession(&sceneID, nil, "did:plc:host")
	if err != nil {
		t.Fatalf("Failed to create stream: %v", err)
	}

	// Simulate participants joining
	geo1 := "abcd"
	geo2 := "efgh"

	time.Sleep(5 * time.Millisecond) // Engagement lag
	_ = repo.RecordParticipantEvent(streamID, "user1", "join", &geo1)

	time.Sleep(5 * time.Millisecond)
	_ = repo.RecordParticipantEvent(streamID, "user2", "join", &geo2)

	time.Sleep(5 * time.Millisecond)
	_ = repo.RecordParticipantEvent(streamID, "user3", "join", &geo1) // Peak = 3

	time.Sleep(5 * time.Millisecond)
	_ = repo.RecordParticipantEvent(streamID, "user1", "leave", nil)

	time.Sleep(5 * time.Millisecond)
	_ = repo.RecordParticipantEvent(streamID, "user2", "leave", nil)

	// End the stream
	time.Sleep(5 * time.Millisecond)
	err = sessionRepo.EndStreamSession(streamID)
	if err != nil {
		t.Fatalf("Failed to end stream: %v", err)
	}

	// Compute analytics
	analytics, err := repo.ComputeAnalytics(streamID)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify metrics
	if analytics.PeakConcurrentListeners != 3 {
		t.Errorf("Expected peak 3 listeners, got %d", analytics.PeakConcurrentListeners)
	}

	if analytics.TotalUniqueParticipants != 3 {
		t.Errorf("Expected 3 unique participants, got %d", analytics.TotalUniqueParticipants)
	}

	if analytics.TotalJoinAttempts != 3 {
		t.Errorf("Expected 3 join attempts, got %d", analytics.TotalJoinAttempts)
	}

	if analytics.EngagementLagSeconds == nil {
		t.Fatal("Expected engagement lag to be set")
	}

	if *analytics.EngagementLagSeconds < 0 {
		t.Errorf("Expected positive engagement lag, got %d", *analytics.EngagementLagSeconds)
	}

	// Verify geographic distribution
	if len(analytics.GeographicDistribution) != 2 {
		t.Errorf("Expected 2 geographic regions, got %d", len(analytics.GeographicDistribution))
	}

	if analytics.GeographicDistribution["abcd"] != 2 {
		t.Errorf("Expected 2 participants from 'abcd', got %d", analytics.GeographicDistribution["abcd"])
	}

	if analytics.GeographicDistribution["efgh"] != 1 {
		t.Errorf("Expected 1 participant from 'efgh', got %d", analytics.GeographicDistribution["efgh"])
	}

	// Verify retention metrics (only 2 left, user3 still listening)
	if analytics.AvgListenDurationSeconds == nil {
		t.Fatal("Expected average duration to be set")
	}

	if *analytics.AvgListenDurationSeconds <= 0 {
		t.Errorf("Expected positive average duration, got %f", *analytics.AvgListenDurationSeconds)
	}

	if analytics.MedianListenDurationSeconds == nil {
		t.Fatal("Expected median duration to be set")
	}
}

func TestInMemoryAnalyticsRepository_ComputeAnalytics_MultipleJoins(t *testing.T) {
	sessionRepo := NewInMemorySessionRepository()
	repo := NewInMemoryAnalyticsRepository(sessionRepo)

	// Create a stream session
	sceneID := "scene-1"
	streamID, _, err := sessionRepo.CreateStreamSession(&sceneID, nil, "did:plc:host")
	if err != nil {
		t.Fatalf("Failed to create stream: %v", err)
	}

	// Same user joins multiple times
	time.Sleep(5 * time.Millisecond)
	_ = repo.RecordParticipantEvent(streamID, "user1", "join", nil)
	time.Sleep(5 * time.Millisecond)
	_ = repo.RecordParticipantEvent(streamID, "user1", "leave", nil)
	time.Sleep(5 * time.Millisecond)
	_ = repo.RecordParticipantEvent(streamID, "user1", "join", nil) // Re-join

	// End the stream
	time.Sleep(5 * time.Millisecond)
	err = sessionRepo.EndStreamSession(streamID)
	if err != nil {
		t.Fatalf("Failed to end stream: %v", err)
	}

	// Compute analytics
	analytics, err := repo.ComputeAnalytics(streamID)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify that unique participants = 1, but join attempts = 2
	if analytics.TotalUniqueParticipants != 1 {
		t.Errorf("Expected 1 unique participant, got %d", analytics.TotalUniqueParticipants)
	}

	if analytics.TotalJoinAttempts != 2 {
		t.Errorf("Expected 2 join attempts, got %d", analytics.TotalJoinAttempts)
	}
}

func TestInMemoryAnalyticsRepository_GetAnalytics_NotFound(t *testing.T) {
	sessionRepo := NewInMemorySessionRepository()
	repo := NewInMemoryAnalyticsRepository(sessionRepo)

	_, err := repo.GetAnalytics("non-existent-stream")
	if err != ErrAnalyticsNotFound {
		t.Errorf("Expected ErrAnalyticsNotFound, got %v", err)
	}
}

func TestInMemoryAnalyticsRepository_GetAnalytics_Success(t *testing.T) {
	sessionRepo := NewInMemorySessionRepository()
	repo := NewInMemoryAnalyticsRepository(sessionRepo)

	// Create and end a stream
	sceneID := "scene-1"
	streamID, _, err := sessionRepo.CreateStreamSession(&sceneID, nil, "did:plc:host")
	if err != nil {
		t.Fatalf("Failed to create stream: %v", err)
	}

	err = sessionRepo.EndStreamSession(streamID)
	if err != nil {
		t.Fatalf("Failed to end stream: %v", err)
	}

	// Compute analytics
	computed, err := repo.ComputeAnalytics(streamID)
	if err != nil {
		t.Fatalf("Failed to compute analytics: %v", err)
	}

	// Retrieve analytics
	retrieved, err := repo.GetAnalytics(streamID)
	if err != nil {
		t.Fatalf("Failed to get analytics: %v", err)
	}

	// Verify they match
	if retrieved.StreamSessionID != computed.StreamSessionID {
		t.Errorf("Expected stream ID %s, got %s", computed.StreamSessionID, retrieved.StreamSessionID)
	}

	if retrieved.PeakConcurrentListeners != computed.PeakConcurrentListeners {
		t.Errorf("Expected peak %d, got %d", computed.PeakConcurrentListeners, retrieved.PeakConcurrentListeners)
	}
}

func TestInMemoryAnalyticsRepository_ComputeAnalytics_RetentionCalculations(t *testing.T) {
	sessionRepo := NewInMemorySessionRepository()
	repo := NewInMemoryAnalyticsRepository(sessionRepo)

	// Create a stream session
	sceneID := "scene-1"
	streamID, _, err := sessionRepo.CreateStreamSession(&sceneID, nil, "did:plc:host")
	if err != nil {
		t.Fatalf("Failed to create stream: %v", err)
	}

	// User1: stays 100ms, User2: stays 200ms, User3: stays 150ms
	_ = repo.RecordParticipantEvent(streamID, "user1", "join", nil)
	time.Sleep(100 * time.Millisecond)
	_ = repo.RecordParticipantEvent(streamID, "user1", "leave", nil)

	_ = repo.RecordParticipantEvent(streamID, "user2", "join", nil)
	time.Sleep(200 * time.Millisecond)
	_ = repo.RecordParticipantEvent(streamID, "user2", "leave", nil)

	_ = repo.RecordParticipantEvent(streamID, "user3", "join", nil)
	time.Sleep(150 * time.Millisecond)
	_ = repo.RecordParticipantEvent(streamID, "user3", "leave", nil)

	err = sessionRepo.EndStreamSession(streamID)
	if err != nil {
		t.Fatalf("Failed to end stream: %v", err)
	}

	// Compute analytics
	analytics, err := repo.ComputeAnalytics(streamID)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Average should be (100 + 200 + 150) / 3 = 150ms = 0.15s
	if analytics.AvgListenDurationSeconds == nil {
		t.Fatal("Expected average duration to be set")
	}

	avgMs := *analytics.AvgListenDurationSeconds * 1000
	if avgMs < 140 || avgMs > 160 {
		t.Errorf("Expected average ~150ms, got %.0fms", avgMs)
	}

	// Median should be 150ms = 0.15s (middle value when sorted: [100, 150, 200])
	if analytics.MedianListenDurationSeconds == nil {
		t.Fatal("Expected median duration to be set")
	}

	medianMs := *analytics.MedianListenDurationSeconds * 1000
	if medianMs < 140 || medianMs > 160 {
		t.Errorf("Expected median ~150ms, got %.0fms", medianMs)
	}
}

// TestInMemoryAnalyticsRepository_ComputeAnalytics_ParticipantsNeverLeave tests analytics for participants still listening when stream ends.
func TestInMemoryAnalyticsRepository_ComputeAnalytics_ParticipantsNeverLeave(t *testing.T) {
	sessionRepo := NewInMemorySessionRepository()
	repo := NewInMemoryAnalyticsRepository(sessionRepo)

	// Create a stream session
	sceneID := "scene-1"
	streamID, _, err := sessionRepo.CreateStreamSession(&sceneID, nil, "did:plc:host")
	if err != nil {
		t.Fatalf("Failed to create stream: %v", err)
	}

	// User1: joins but never leaves (still listening)
	_ = repo.RecordParticipantEvent(streamID, "user1", "join", nil)
	time.Sleep(50 * time.Millisecond)

	// User2: joins, leaves after 100ms
	_ = repo.RecordParticipantEvent(streamID, "user2", "join", nil)
	time.Sleep(50 * time.Millisecond)

	// User3: joins while user2 is still present (peak = 3)
	_ = repo.RecordParticipantEvent(streamID, "user3", "join", nil)
	time.Sleep(50 * time.Millisecond)

	// User2 leaves (concurrent drops to 2)
	_ = repo.RecordParticipantEvent(streamID, "user2", "leave", nil)
	time.Sleep(50 * time.Millisecond)

	err = sessionRepo.EndStreamSession(streamID)
	if err != nil {
		t.Fatalf("Failed to end stream: %v", err)
	}

	// Compute analytics
	analytics, err := repo.ComputeAnalytics(streamID)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify metrics
	if analytics.PeakConcurrentListeners != 3 {
		t.Errorf("Expected peak 3, got %d", analytics.PeakConcurrentListeners)
	}

	if analytics.TotalUniqueParticipants != 3 {
		t.Errorf("Expected 3 unique participants, got %d", analytics.TotalUniqueParticipants)
	}

	// Retention metrics should only include user2 who explicitly left
	// user1 and user3 are still listening, so they're excluded from retention
	if analytics.AvgListenDurationSeconds == nil {
		t.Fatal("Expected average duration to be set")
	}

	// Should have exactly 1 duration (user2 only): ~100ms (from join to leave: 50ms + 50ms)
	avgMs := *analytics.AvgListenDurationSeconds * 1000
	if avgMs < 90 || avgMs > 110 {
		t.Errorf("Expected average ~100ms (user2 only), got %.0fms", avgMs)
	}

	// Median should equal average when there's only one sample
	if analytics.MedianListenDurationSeconds == nil {
		t.Fatal("Expected median duration to be set")
	}

	medianMs := *analytics.MedianListenDurationSeconds * 1000
	if medianMs < 90 || medianMs > 110 {
		t.Errorf("Expected median ~100ms (user2 only), got %.0fms", medianMs)
	}
}
