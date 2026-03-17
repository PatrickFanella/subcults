// Package ranking provides centralized ranking component calculations
// with calibration support for search and discovery features.
package ranking

import (
	"time"
)

// TextWeight computes a weighted text ranking score.
// Parameters:
//   - rawRank: The raw text match score (typically from database ts_rank or similar)
//   - w: The weight to apply to the raw rank
//
// Returns a weighted score normalized to the [0, 1] range. The raw rank and weight
// are expected to be in the [0, 1] range; any out-of-range inputs are clamped.
func TextWeight(rawRank float64, w float64) float64 {
	// Clamp rawRank to [0, 1]
	if rawRank < 0.0 {
		rawRank = 0.0
	} else if rawRank > 1.0 {
		rawRank = 1.0
	}

	// Clamp weight to [0, 1]
	if w < 0.0 {
		w = 0.0
	} else if w > 1.0 {
		w = 1.0
	}

	return rawRank * w
}

// ProximityWeight computes a distance-based proximity score normalized to [0, 1].
// Uses a hyperbolic decay function to convert distance to a proximity score.
//
// Parameters:
//   - distanceMeters: The distance in meters from the reference point
//
// Returns a value between 0.0 (far) and 1.0 (very close).
// Formula: 1 / (1 + (distance / 1000)) - gives 1.0 at 0m, 0.5 at ~1km, 0.33 at ~2km, decays gradually
func ProximityWeight(distanceMeters float64) float64 {
	if distanceMeters < 0 {
		distanceMeters = 0 // Clamp negative distances
	}

	// Normalize distance to kilometers and apply hyperbolic decay function
	// This gives: 1.0 at 0m, 0.5 at 1000m, 0.33 at 2000m, etc.
	distanceKm := distanceMeters / 1000.0
	score := 1.0 / (1.0 + distanceKm)

	return score
}

// RecencyWeight computes a time-based recency score normalized to [0, 1].
// Events happening sooner receive higher scores.
//
// Parameters:
//   - startTime: The start time of the event
//   - windowSpan: The total time window duration being searched
//
// Returns a value between 0.0 (furthest in future) and 1.0 (happening now/past).
// Formula: 1 - ((event_start - now) / window_span) clamped to [0, 1]
func RecencyWeight(startTime time.Time, windowSpan time.Duration) float64 {
	now := time.Now()

	if windowSpan <= 0 {
		return 1.0 // If no window span, consider all events equally recent
	}

	// Calculate time difference from now to event start
	timeDiff := startTime.Sub(now)

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

// TrustWeight computes the trust component score with feature flag support.
// When trust ranking is disabled, returns 0. Otherwise returns the trust score clamped to [0, 1].
//
// Parameters:
//   - trustScore: The computed trust score (expected to be in [0, 1] range)
//   - enabled: Whether trust-based ranking is enabled
//
// Returns trustScore (clamped to [0, 1]) if enabled is true, otherwise 0.
func TrustWeight(trustScore float64, enabled bool) float64 {
	if !enabled {
		return 0.0
	}

	// Clamp trustScore to [0, 1] to ensure contract compliance
	if trustScore < 0.0 {
		return 0.0
	}
	if trustScore > 1.0 {
		return 1.0
	}

	return trustScore
}

// SceneParams holds the parameters for computing a scene composite score.
type SceneParams struct {
	Text         float64 // Text match score [0, 1]
	Proximity    float64 // Proximity score [0, 1]
	Trust        float64 // Trust score [0, 1]
	TrustEnabled bool    // Whether trust ranking is enabled
}

// EventParams holds the parameters for computing an event composite score.
type EventParams struct {
	Text         float64 // Text match score [0, 1]
	Proximity    float64 // Proximity score [0, 1]
	Recency      float64 // Recency score [0, 1]
	Trust        float64 // Trust score [0, 1]
	TrustEnabled bool    // Whether trust ranking is enabled
}

// CompositeScoreScene computes the final composite ranking score for a scene.
// Uses the calibrated weights to combine text match, proximity, and optional trust scores.
//
// Default formula (without trust): composite_score = (text * 0.4) + (proximity * 0.3) + (trust_weight * 0.1)
// When trust is disabled, the trust component is 0, making max score 0.7 instead of 0.8.
//
// Parameters:
//   - params: The component scores and feature flags
//   - weights: The calibrated weight configuration (optional, uses default if nil)
//
// Returns the composite score (typically in [0, 0.7-0.8] range depending on trust flag).
func CompositeScoreScene(params SceneParams, weights *Weights) float64 {
	if weights == nil {
		weights = GetActiveWeights()
	}

	score := (params.Text * weights.Scene.TextMatch) +
		(params.Proximity * weights.Scene.Proximity)

	if params.TrustEnabled {
		score += params.Trust * weights.Scene.Trust
	}

	return score
}

// CompositeScoreEvent computes the final composite ranking score for an event.
// Uses the calibrated weights to combine recency, text match, proximity, and optional trust scores.
//
// Default formula: composite_score = (recency * 0.3) + (text * 0.4) + (proximity * 0.2) + (trust_weight * 0.1)
// When trust is disabled, the trust component is 0, making max score 0.9 instead of 1.0.
//
// Parameters:
//   - params: The component scores and feature flags
//   - weights: The calibrated weight configuration (optional, uses default if nil)
//
// Returns the composite score (typically in [0, 0.9-1.0] range depending on trust flag).
func CompositeScoreEvent(params EventParams, weights *Weights) float64 {
	if weights == nil {
		weights = GetActiveWeights()
	}

	score := (params.Recency * weights.Event.Recency) +
		(params.Text * weights.Event.TextMatch) +
		(params.Proximity * weights.Event.Proximity)

	if params.TrustEnabled {
		score += params.Trust * weights.Event.Trust
	}

	return score
}
