// Package scene provides models and repository for managing scenes and events
// with location privacy controls.
package scene

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/onnwee/subcults/internal/geo"
)

// EventRankingWeights defines the weights for different ranking components.
// These weights determine how much each factor contributes to the final composite score.
//
// The default formula is optimized for event discovery:
// - Text match (40%): Prioritizes query relevance for targeted search
// - Recency (30%): Favors events happening sooner for timely discovery
// - Proximity (20%): Considers geographic convenience
// - Trust (10%): Adds scene reputation signal without dominating results
//
// Formula: composite_score = (recency * 0.3) + (text * 0.4) + (proximity * 0.2) + (trust * 0.1)
//
// Note: When trust ranking is disabled via feature flag, trust weight is effectively 0,
// and the remaining components are normalized to sum to 0.9.
type EventRankingWeights struct {
	Recency   float64 // Time recency weight (default: 0.3)
	TextMatch float64 // Text match weight (default: 0.4)
	Proximity float64 // Proximity/geo weight (default: 0.2)
	Trust     float64 // Trust score weight (default: 0.1)
}

// DefaultEventRankingWeights returns the default ranking weights for event search.
var DefaultEventRankingWeights = EventRankingWeights{
	Recency:   0.3,
	TextMatch: 0.4,
	Proximity: 0.2,
	Trust:     0.1,
}

// CalculateRecencyWeight computes the time recency weight for an event.
// Formula: 1 - ((event_start - now) / window_span) clamped to [0, 1]
//
// Parameters:
//   - eventStart: The start time of the event
//   - now: Current time (reference point)
//   - windowSpan: The total time window duration (from - to)
//
// Returns a value between 0.0 (furthest in future) and 1.0 (happening now/past)
func CalculateRecencyWeight(eventStart time.Time, now time.Time, windowSpan time.Duration) float64 {
	if windowSpan <= 0 {
		return 1.0 // If no window span, consider all events equally recent
	}

	// Calculate time difference from now to event start
	timeDiff := eventStart.Sub(now)

	// If event is in the past or happening now, it's maximally recent
	if timeDiff <= 0 {
		return 1.0
	}

	// Calculate weight: 1 - (timeDiff / windowSpan)
	weight := 1.0 - (float64(timeDiff) / float64(windowSpan))

	// Clamp to [0, 1] range
	if weight < 0.0 {
		return 0.0
	}
	if weight > 1.0 {
		return 1.0
	}

	return weight
}

// CalculateTextMatchScore computes a simple text match score based on query presence.
// Returns 1.0 if query matches, 0.0 otherwise.
// For in-memory implementation, we use simple case-insensitive substring matching.
// In a real PostGIS implementation, this would use ts_rank with tsvector.
func CalculateTextMatchScore(event *Event, query string) float64 {
	if query == "" {
		return 1.0 // No query means all events match equally
	}

	queryLower := strings.ToLower(query)

	// Check title
	if strings.Contains(strings.ToLower(event.Title), queryLower) {
		return 1.0
	}

	// Check description
	if strings.Contains(strings.ToLower(event.Description), queryLower) {
		return 0.8 // Slightly lower weight for description match
	}

	// Check tags
	for _, tag := range event.Tags {
		if strings.Contains(strings.ToLower(tag), queryLower) {
			return 0.6 // Lower weight for tag match
		}
	}

	return 0.0 // No match
}

// CalculateProximityScore computes a distance-based proximity score.
// Returns a value between 0.0 (far) and 1.0 (close) based on distance from reference point.
// For simplicity in in-memory implementation, we use the center of the bbox as reference.
// In a real PostGIS implementation, this would use ST_Distance with proper geo calculations.
func CalculateProximityScore(event *Event, centerLat, centerLng float64) float64 {
	if event.PrecisePoint == nil {
		return 0.5 // Default score if no location
	}

	// Calculate simple Euclidean distance (not great circle, but good enough for in-memory)
	// For production with PostGIS, use ST_Distance with proper geodesic calculations
	latDiff := event.PrecisePoint.Lat - centerLat
	lngDiff := event.PrecisePoint.Lng - centerLng
	distance := math.Sqrt(latDiff*latDiff + lngDiff*lngDiff)

	// Normalize distance to [0, 1] range
	// Use a simple decay function: 1 / (1 + distance)
	// This gives 1.0 for distance=0, 0.5 for distance=1, etc.
	score := 1.0 / (1.0 + distance)

	return score
}

// CalculateCompositeScore computes the final composite ranking score for an event.
// Combines recency, text match, proximity, and optional trust scores using configured weights.
func CalculateCompositeScore(
	recencyWeight float64,
	textMatchScore float64,
	proximityScore float64,
	trustScore float64,
	weights EventRankingWeights,
	includeTrust bool,
) float64 {
	score := (recencyWeight * weights.Recency) +
		(textMatchScore * weights.TextMatch) +
		(proximityScore * weights.Proximity)

	if includeTrust {
		score += trustScore * weights.Trust
	}

	return score
}

// SceneRankingWeights defines the weights for different ranking components in scene search.
// These weights determine how much each factor contributes to the final composite score.
//
// The default formula is optimized for scene discovery:
// - Text match (60%): Prioritizes query relevance for targeted search
// - Proximity (25%): Considers geographic convenience
// - Trust (15%): Adds scene reputation signal without dominating results
//
// Formula: composite_score = (text * 0.6) + (proximity * 0.25) + (trust * 0.15)
//
// Note: When trust ranking is disabled via feature flag, trust weight is effectively 0,
// resulting in a maximum possible score of 0.85 (not 1.0).
type SceneRankingWeights struct {
	TextMatch float64 // Text match weight (default: 0.6)
	Proximity float64 // Proximity/geo weight (default: 0.25)
	Trust     float64 // Trust score weight (default: 0.15)
}

// DefaultSceneRankingWeights returns the default ranking weights for scene search.
var DefaultSceneRankingWeights = SceneRankingWeights{
	TextMatch: 0.6,
	Proximity: 0.25,
	Trust:     0.15,
}

// CalculateSceneTextMatchScore computes a simple text match score based on query presence.
// Returns a value between 0.0 and 1.0 based on where the match is found.
// For in-memory implementation, we use simple case-insensitive substring matching.
// In a real PostGIS implementation, this would use ts_rank with tsvector.
func CalculateSceneTextMatchScore(scene *Scene, query string) float64 {
	if query == "" {
		return 1.0 // No query means all scenes match equally
	}

	queryTerms := strings.Fields(strings.ToLower(query))
	if len(queryTerms) == 0 {
		return 1.0
	}

	name := strings.ToLower(scene.Name)
	description := strings.ToLower(scene.Description)
	tags := strings.ToLower(strings.Join(scene.Tags, " "))

	// Approximate FTS ranking: weighted term coverage by field
	// name=1.0, description=0.7, tags=0.5
	// final score normalized to [0,1]
	maxPerTerm := 1.0
	totalScore := 0.0
	for _, term := range queryTerms {
		termScore := 0.0
		if strings.Contains(name, term) {
			termScore = 1.0
		} else if strings.Contains(description, term) {
			termScore = 0.7
		} else if strings.Contains(tags, term) {
			termScore = 0.5
		}
		totalScore += termScore
	}

	score := totalScore / (float64(len(queryTerms)) * maxPerTerm)
	if score < 0.0 {
		return 0.0
	}
	if score > 1.0 {
		return 1.0
	}
	return score
}

// CalculateSceneProximityScore computes a distance-based proximity score for a scene.
// Returns a value between 0.0 (far) and 1.0 (close) based on distance from reference point.
// For simplicity in in-memory implementation, we use the center of the bbox as reference.
// In a real PostGIS implementation, this would use ST_Distance with proper geo calculations.
func CalculateSceneProximityScore(scene *Scene, centerLat, centerLng float64) float64 {
	// Geohash prefix similarity scoring, used even when precise location is not available.
	proximityScore := 0.5 // default
	if scene.CoarseGeohash != "" {
		centerGeohash := geo.Encode(centerLat, centerLng, len(scene.CoarseGeohash))
		matchedPrefix := 0
		for i := 0; i < len(scene.CoarseGeohash) && i < len(centerGeohash); i++ {
			if scene.CoarseGeohash[i] != centerGeohash[i] {
				break
			}
			matchedPrefix++
		}
		proximityScore = float64(matchedPrefix) / float64(len(scene.CoarseGeohash))
	}

	// If precise coordinates are available, blend geohash score with distance score.
	if scene.PrecisePoint != nil {
		latDiff := scene.PrecisePoint.Lat - centerLat
		lngDiff := scene.PrecisePoint.Lng - centerLng
		distance := math.Sqrt(latDiff*latDiff + lngDiff*lngDiff)
		distanceScore := 1.0 / (1.0 + distance)
		proximityScore = (proximityScore * 0.5) + (distanceScore * 0.5)
	}

	return proximityScore
}

// CalculateSceneCompositeScore computes the final composite ranking score for a scene.
// Combines text match, proximity, and optional trust scores using configured weights.
func CalculateSceneCompositeScore(
	textMatchScore float64,
	proximityScore float64,
	trustScore float64,
	weights SceneRankingWeights,
	includeTrust bool,
) float64 {
	score := (textMatchScore * weights.TextMatch) +
		(proximityScore * weights.Proximity)

	if includeTrust {
		score += trustScore * weights.Trust
	}

	return score
}

// SceneCursor represents the pagination cursor for scene search results.
type SceneCursor struct {
	Score float64 `json:"score"` // Composite score of last scene
	ID    string  `json:"id"`    // Scene ID for stable ordering
}

// EncodeSceneCursor encodes a cursor to a base64 string for pagination.
func EncodeSceneCursor(score float64, id string) string {
	cursor := SceneCursor{
		Score: score,
		ID:    id,
	}
	data, _ := json.Marshal(cursor)
	return base64.URLEncoding.EncodeToString(data)
}

// DecodeSceneCursor decodes a base64 cursor string back to SceneCursor.
// Returns error if cursor is invalid or cannot be decoded.
func DecodeSceneCursor(encoded string) (*SceneCursor, error) {
	if encoded == "" {
		return nil, nil
	}

	data, err := base64.URLEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("invalid cursor encoding: %w", err)
	}

	var cursor SceneCursor
	if err := json.Unmarshal(data, &cursor); err != nil {
		return nil, fmt.Errorf("invalid cursor format: %w", err)
	}

	return &cursor, nil
}

// EventScoreCursor represents the pagination cursor for ranked event search results.
type EventScoreCursor struct {
	Score float64 `json:"score"` // Composite score of last event
	ID    string  `json:"id"`    // Event ID for stable ordering
}

// EncodeEventScoreCursor encodes a score-based event cursor to a base64 string.
func EncodeEventScoreCursor(score float64, id string) string {
	cursor := EventScoreCursor{Score: score, ID: id}
	data, _ := json.Marshal(cursor)
	return base64.URLEncoding.EncodeToString(data)
}

// DecodeEventScoreCursor decodes a base64 event score cursor string.
// Returns (nil, nil) for empty input, or an error for invalid cursors.
func DecodeEventScoreCursor(encoded string) (*EventScoreCursor, error) {
	if encoded == "" {
		return nil, nil
	}
	data, err := base64.URLEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("invalid cursor encoding: %w", err)
	}
	var cursor EventScoreCursor
	if err := json.Unmarshal(data, &cursor); err != nil {
		return nil, fmt.Errorf("invalid cursor format: %w", err)
	}
	return &cursor, nil
}

// EventTimeCursor represents the pagination cursor for time-ordered event search results.
type EventTimeCursor struct {
	StartsAt time.Time `json:"starts_at"` // StartsAt of last event (second precision)
	ID       string    `json:"id"`        // Event ID for stable ordering
}

// EncodeEventTimeCursor encodes a time-based event cursor to a base64 string.
func EncodeEventTimeCursor(startsAt time.Time, id string) string {
	cursor := EventTimeCursor{StartsAt: startsAt.Truncate(time.Second), ID: id}
	data, _ := json.Marshal(cursor)
	return base64.URLEncoding.EncodeToString(data)
}

// DecodeEventTimeCursor decodes a base64 event time cursor string.
// Returns (nil, nil) for empty input, or an error for invalid cursors.
func DecodeEventTimeCursor(encoded string) (*EventTimeCursor, error) {
	if encoded == "" {
		return nil, nil
	}
	data, err := base64.URLEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("invalid cursor encoding: %w", err)
	}
	var cursor EventTimeCursor
	if err := json.Unmarshal(data, &cursor); err != nil {
		return nil, fmt.Errorf("invalid cursor format: %w", err)
	}
	return &cursor, nil
}
