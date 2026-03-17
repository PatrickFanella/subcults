// Package stream provides in-memory implementation of analytics repository.
package stream

import (
	"errors"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
)

var (
	// ErrAnalyticsNotFound is returned when analytics for a stream session are not found.
	ErrAnalyticsNotFound = errors.New("analytics not found")
)

// InMemoryAnalyticsRepository is an in-memory implementation of AnalyticsRepository.
// Thread-safe via RWMutex.
type InMemoryAnalyticsRepository struct {
	mu          sync.RWMutex
	events      map[string][]*ParticipantEvent // stream_session_id -> events
	analytics   map[string]*Analytics          // stream_session_id -> analytics
	sessionRepo SessionRepository              // Reference to session repo for stream data
}

// NewInMemoryAnalyticsRepository creates a new in-memory analytics repository.
func NewInMemoryAnalyticsRepository(sessionRepo SessionRepository) *InMemoryAnalyticsRepository {
	return &InMemoryAnalyticsRepository{
		events:      make(map[string][]*ParticipantEvent),
		analytics:   make(map[string]*Analytics),
		sessionRepo: sessionRepo,
	}
}

// RecordParticipantEvent records a join or leave event for a participant.
func (r *InMemoryAnalyticsRepository) RecordParticipantEvent(streamSessionID, participantDID, eventType string, geohashPrefix *string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Validate event type
	if eventType != "join" && eventType != "leave" {
		return errors.New("event_type must be 'join' or 'leave'")
	}

	event := &ParticipantEvent{
		ID:              uuid.New().String(),
		StreamSessionID: streamSessionID,
		ParticipantDID:  participantDID,
		EventType:       eventType,
		GeohashPrefix:   geohashPrefix,
		OccurredAt:      time.Now(),
	}

	r.events[streamSessionID] = append(r.events[streamSessionID], event)
	return nil
}

// GetParticipantEvents retrieves all participant events for a stream session, ordered by occurred_at.
func (r *InMemoryAnalyticsRepository) GetParticipantEvents(streamSessionID string) ([]*ParticipantEvent, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	events := r.events[streamSessionID]
	if events == nil {
		return []*ParticipantEvent{}, nil
	}

	// Create a copy and sort by occurred_at
	result := make([]*ParticipantEvent, len(events))
	for i, e := range events {
		eventCopy := *e
		// Deep copy pointer fields to avoid sharing internal state
		if e.GeohashPrefix != nil {
			prefix := *e.GeohashPrefix
			eventCopy.GeohashPrefix = &prefix
		}
		result[i] = &eventCopy
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].OccurredAt.Before(result[j].OccurredAt)
	})

	return result, nil
}

// ComputeAnalytics calculates and stores analytics for a stream session.
func (r *InMemoryAnalyticsRepository) ComputeAnalytics(streamSessionID string) (*Analytics, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Get the stream session
	session, err := r.sessionRepo.GetByID(streamSessionID)
	if err != nil {
		return nil, err
	}

	// Get all participant events
	events := r.events[streamSessionID]
	if events == nil {
		events = []*ParticipantEvent{}
	}

	// Sort events by time
	sortedEvents := make([]*ParticipantEvent, len(events))
	copy(sortedEvents, events)
	sort.Slice(sortedEvents, func(i, j int) bool {
		return sortedEvents[i].OccurredAt.Before(sortedEvents[j].OccurredAt)
	})

	// Calculate stream duration
	streamDuration := 0
	if session.EndedAt != nil {
		streamDuration = int(session.EndedAt.Sub(session.StartedAt).Seconds())
	}

	// Calculate engagement lag (time to first join)
	var engagementLag *int
	for _, event := range sortedEvents {
		if event.EventType == "join" {
			lag := int(event.OccurredAt.Sub(session.StartedAt).Seconds())
			if lag < 0 {
				lag = 0 // Handle clock skew
			}
			engagementLag = &lag
			break
		}
	}

	// Track concurrent listeners and unique participants
	concurrent := 0
	peakConcurrent := 0
	uniqueParticipants := make(map[string]bool)
	participantJoinTimes := make(map[string]time.Time)
	listenDurations := []float64{}
	geoDistribution := make(map[string]int)
	totalJoins := 0

	// Track unique participants per geohash for privacy-safe geographic distribution
	geoParticipants := make(map[string]map[string]bool)

	for _, event := range sortedEvents {
		if event.EventType == "join" {
			totalJoins++

			// Only increment concurrent count when this participant was not already joined
			if _, alreadyJoined := participantJoinTimes[event.ParticipantDID]; !alreadyJoined {
				concurrent++
				if concurrent > peakConcurrent {
					peakConcurrent = concurrent
				}
			}
			uniqueParticipants[event.ParticipantDID] = true
			participantJoinTimes[event.ParticipantDID] = event.OccurredAt

			// Track geographic distribution (privacy-safe) - deduplicate by participant DID
			if event.GeohashPrefix != nil && *event.GeohashPrefix != "" {
				prefix := *event.GeohashPrefix
				if geoParticipants[prefix] == nil {
					geoParticipants[prefix] = make(map[string]bool)
				}
				geoParticipants[prefix][event.ParticipantDID] = true
			}
		} else if event.EventType == "leave" {
			concurrent--
			if concurrent < 0 {
				concurrent = 0 // Handle inconsistent data
			}

			// Calculate listen duration for this participant
			if joinTime, ok := participantJoinTimes[event.ParticipantDID]; ok {
				duration := event.OccurredAt.Sub(joinTime).Seconds()
				if duration > 0 {
					listenDurations = append(listenDurations, duration)
				}
				delete(participantJoinTimes, event.ParticipantDID) // Remove to handle re-joins
			}
		}
	}

	// Convert unique participants per geohash to counts
	for prefix, participants := range geoParticipants {
		geoDistribution[prefix] = len(participants)
	}

	// Calculate retention metrics
	var avgDuration *float64
	var medianDuration *float64
	if len(listenDurations) > 0 {
		// Calculate average
		sum := 0.0
		for _, d := range listenDurations {
			sum += d
		}
		avg := sum / float64(len(listenDurations))
		avgDuration = &avg

		// Calculate median
		sort.Float64s(listenDurations)
		mid := len(listenDurations) / 2
		if len(listenDurations)%2 == 0 {
			median := (listenDurations[mid-1] + listenDurations[mid]) / 2
			medianDuration = &median
		} else {
			medianDuration = &listenDurations[mid]
		}
	}

	// Create analytics object
	analytics := &Analytics{
		ID:                          uuid.New().String(),
		StreamSessionID:             streamSessionID,
		PeakConcurrentListeners:     peakConcurrent,
		TotalUniqueParticipants:     len(uniqueParticipants),
		TotalJoinAttempts:           totalJoins,
		StreamDurationSeconds:       streamDuration,
		EngagementLagSeconds:        engagementLag,
		AvgListenDurationSeconds:    avgDuration,
		MedianListenDurationSeconds: medianDuration,
		GeographicDistribution:      geoDistribution,
		ComputedAt:                  time.Now(),
	}

	// Store analytics
	r.analytics[streamSessionID] = analytics

	return analytics, nil
}

// GetAnalytics retrieves the computed analytics for a stream session.
func (r *InMemoryAnalyticsRepository) GetAnalytics(streamSessionID string) (*Analytics, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	analytics, ok := r.analytics[streamSessionID]
	if !ok {
		return nil, ErrAnalyticsNotFound
	}

	// Return a copy
	analyticsCopy := *analytics

	// Deep copy the geographic distribution map
	if analytics.GeographicDistribution != nil {
		analyticsCopy.GeographicDistribution = make(map[string]int, len(analytics.GeographicDistribution))
		for k, v := range analytics.GeographicDistribution {
			analyticsCopy.GeographicDistribution[k] = v
		}
	}

	return &analyticsCopy, nil
}
