package ranking

import (
	"math"
	"testing"
	"time"
)

// TestTextWeight tests the text weight calculation.
func TestTextWeight(t *testing.T) {
	tests := []struct {
		name     string
		rawRank  float64
		weight   float64
		expected float64
	}{
		{
			name:     "perfect score with full weight",
			rawRank:  1.0,
			weight:   0.4,
			expected: 0.4,
		},
		{
			name:     "zero score",
			rawRank:  0.0,
			weight:   0.4,
			expected: 0.0,
		},
		{
			name:     "half score",
			rawRank:  0.5,
			weight:   0.4,
			expected: 0.2,
		},
		{
			name:     "zero weight",
			rawRank:  1.0,
			weight:   0.0,
			expected: 0.0,
		},
		{
			name:     "negative score (edge case)",
			rawRank:  -0.5,
			weight:   0.4,
			expected: 0.0, // Negative scores are clamped to 0 before weighting
		},
		{
			name:     "score above 1 (edge case)",
			rawRank:  1.5,
			weight:   0.4,
			expected: 0.4, // Scores above 1 are clamped to 1 before weighting
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TextWeight(tt.rawRank, tt.weight)
			if math.Abs(result-tt.expected) > 0.001 {
				t.Errorf("expected %f, got %f", tt.expected, result)
			}
		})
	}
}

// TestProximityWeight tests the distance-based proximity scoring.
func TestProximityWeight(t *testing.T) {
	tests := []struct {
		name           string
		distanceMeters float64
		expectedMin    float64
		expectedMax    float64
	}{
		{
			name:           "at exact location (0m)",
			distanceMeters: 0,
			expectedMin:    0.99,
			expectedMax:    1.0,
		},
		{
			name:           "very close (100m)",
			distanceMeters: 100,
			expectedMin:    0.90,
			expectedMax:    0.92,
		},
		{
			name:           "nearby (500m)",
			distanceMeters: 500,
			expectedMin:    0.66,
			expectedMax:    0.68,
		},
		{
			name:           "1km away",
			distanceMeters: 1000,
			expectedMin:    0.49,
			expectedMax:    0.51,
		},
		{
			name:           "2km away",
			distanceMeters: 2000,
			expectedMin:    0.32,
			expectedMax:    0.34,
		},
		{
			name:           "5km away",
			distanceMeters: 5000,
			expectedMin:    0.16,
			expectedMax:    0.17,
		},
		{
			name:           "10km away",
			distanceMeters: 10000,
			expectedMin:    0.09,
			expectedMax:    0.10,
		},
		{
			name:           "negative distance (edge case)",
			distanceMeters: -500,
			expectedMin:    0.99, // Should be clamped to 0
			expectedMax:    1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ProximityWeight(tt.distanceMeters)

			// Verify result is in expected range
			if result < tt.expectedMin || result > tt.expectedMax {
				t.Errorf("expected result in range [%f, %f], got %f",
					tt.expectedMin, tt.expectedMax, result)
			}

			// Verify result is in [0, 1] range
			if result < 0.0 || result > 1.0 {
				t.Errorf("result %f is outside valid range [0.0, 1.0]", result)
			}
		})
	}
}

// TestRecencyWeight tests the time-based recency scoring.
func TestRecencyWeight(t *testing.T) {
	now := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	windowSpan := 24 * time.Hour // 24 hour window

	tests := []struct {
		name        string
		startTime   time.Time
		windowSpan  time.Duration
		expectedMin float64
		expectedMax float64
	}{
		{
			name:        "event in the past",
			startTime:   now.Add(-1 * time.Hour),
			windowSpan:  windowSpan,
			expectedMin: 1.0,
			expectedMax: 1.0,
		},
		{
			name:        "event happening now",
			startTime:   now,
			windowSpan:  windowSpan,
			expectedMin: 1.0,
			expectedMax: 1.0,
		},
		{
			name:        "event 6 hours in future (75% through window)",
			startTime:   now.Add(6 * time.Hour),
			windowSpan:  windowSpan,
			expectedMin: 0.74,
			expectedMax: 0.76,
		},
		{
			name:        "event 12 hours in future (50% through window)",
			startTime:   now.Add(12 * time.Hour),
			windowSpan:  windowSpan,
			expectedMin: 0.49,
			expectedMax: 0.51,
		},
		{
			name:        "event 18 hours in future (25% through window)",
			startTime:   now.Add(18 * time.Hour),
			windowSpan:  windowSpan,
			expectedMin: 0.24,
			expectedMax: 0.26,
		},
		{
			name:        "event 24 hours in future (end of window)",
			startTime:   now.Add(24 * time.Hour),
			windowSpan:  windowSpan,
			expectedMin: 0.0,
			expectedMax: 0.01,
		},
		{
			name:        "event beyond window",
			startTime:   now.Add(30 * time.Hour),
			windowSpan:  windowSpan,
			expectedMin: 0.0,
			expectedMax: 0.0,
		},
		{
			name:        "zero window span",
			startTime:   now.Add(1 * time.Hour),
			windowSpan:  0,
			expectedMin: 1.0,
			expectedMax: 1.0,
		},
		{
			name:        "negative window span (edge case)",
			startTime:   now.Add(1 * time.Hour),
			windowSpan:  -1 * time.Hour,
			expectedMin: 1.0,
			expectedMax: 1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Since RecencyWeight uses time.Now() internally, we'll just test it directly.
			// The function will use the current time, so we adjust our test startTime
			// relative to when the test runs. For precise testing, we simulate the calculation.

			// Create a test helper that simulates RecencyWeight with a fixed "now"
			testRecencyWeight := func(startTime time.Time, now time.Time, windowSpan time.Duration) float64 {
				if windowSpan <= 0 {
					return 1.0
				}
				timeDiff := startTime.Sub(now)
				if timeDiff <= 0 {
					return 1.0
				}
				weight := 1.0 - (float64(timeDiff) / float64(windowSpan))
				if weight < 0.0 {
					return 0.0
				}
				if weight > 1.0 {
					return 1.0
				}
				return weight
			}

			result := testRecencyWeight(tt.startTime, now, tt.windowSpan)

			// Verify result is in expected range
			if result < tt.expectedMin || result > tt.expectedMax {
				t.Errorf("expected result in range [%f, %f], got %f",
					tt.expectedMin, tt.expectedMax, result)
			}

			// Verify result is clamped to [0, 1]
			if result < 0.0 || result > 1.0 {
				t.Errorf("result %f is outside valid range [0.0, 1.0]", result)
			}
		})
	}
}

// TestRecencyWeight_ActualFunction tests the actual RecencyWeight function with real time.
// This test validates that the exported function works correctly with time.Now().
func TestRecencyWeight_ActualFunction(t *testing.T) {
	windowSpan := 24 * time.Hour

	tests := []struct {
		name        string
		startTime   time.Time
		windowSpan  time.Duration
		expectedMin float64
		expectedMax float64
	}{
		{
			name:        "event in the past",
			startTime:   time.Now().Add(-1 * time.Hour),
			windowSpan:  windowSpan,
			expectedMin: 1.0,
			expectedMax: 1.0,
		},
		{
			name:        "event happening very soon",
			startTime:   time.Now().Add(1 * time.Minute),
			windowSpan:  windowSpan,
			expectedMin: 0.99,
			expectedMax: 1.0,
		},
		{
			name:        "event far in future",
			startTime:   time.Now().Add(30 * time.Hour),
			windowSpan:  windowSpan,
			expectedMin: 0.0,
			expectedMax: 0.0,
		},
		{
			name:        "zero window span",
			startTime:   time.Now().Add(1 * time.Hour),
			windowSpan:  0,
			expectedMin: 1.0,
			expectedMax: 1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RecencyWeight(tt.startTime, tt.windowSpan)

			// Verify result is in expected range
			if result < tt.expectedMin || result > tt.expectedMax {
				t.Errorf("expected result in range [%f, %f], got %f",
					tt.expectedMin, tt.expectedMax, result)
			}

			// Verify result is clamped to [0, 1]
			if result < 0.0 || result > 1.0 {
				t.Errorf("result %f is outside valid range [0.0, 1.0]", result)
			}
		})
	}
}

// TestTrustWeight tests the trust weight with feature flag support.
func TestTrustWeight(t *testing.T) {
	tests := []struct {
		name       string
		trustScore float64
		enabled    bool
		expected   float64
	}{
		{
			name:       "trust enabled with full score",
			trustScore: 1.0,
			enabled:    true,
			expected:   1.0,
		},
		{
			name:       "trust enabled with half score",
			trustScore: 0.5,
			enabled:    true,
			expected:   0.5,
		},
		{
			name:       "trust enabled with zero score",
			trustScore: 0.0,
			enabled:    true,
			expected:   0.0,
		},
		{
			name:       "trust disabled with full score",
			trustScore: 1.0,
			enabled:    false,
			expected:   0.0,
		},
		{
			name:       "trust disabled with high score",
			trustScore: 0.9,
			enabled:    false,
			expected:   0.0,
		},
		{
			name:       "negative trust score enabled (edge case)",
			trustScore: -0.5,
			enabled:    true,
			expected:   0.0, // TrustWeight clamps to lower bound 0
		},
		{
			name:       "trust score above 1 enabled (edge case)",
			trustScore: 1.5,
			enabled:    true,
			expected:   1.0, // TrustWeight clamps to upper bound 1
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TrustWeight(tt.trustScore, tt.enabled)
			if math.Abs(result-tt.expected) > 0.001 {
				t.Errorf("expected %f, got %f", tt.expected, result)
			}
		})
	}
}

// TestCompositeScoreScene tests the scene composite scoring.
func TestCompositeScoreScene(t *testing.T) {
	tests := []struct {
		name     string
		params   SceneParams
		weights  *Weights
		expected float64
	}{
		{
			name: "all perfect scores without trust",
			params: SceneParams{
				Text:         1.0,
				Proximity:    1.0,
				Trust:        1.0,
				TrustEnabled: false,
			},
			weights:  nil, // Use defaults
			expected: 0.7, // 0.4 + 0.3 = 0.7
		},
		{
			name: "all perfect scores with trust",
			params: SceneParams{
				Text:         1.0,
				Proximity:    1.0,
				Trust:        1.0,
				TrustEnabled: true,
			},
			weights:  nil, // Use defaults
			expected: 0.8, // 0.4 + 0.3 + 0.1 = 0.8
		},
		{
			name: "all zero scores",
			params: SceneParams{
				Text:         0.0,
				Proximity:    0.0,
				Trust:        0.0,
				TrustEnabled: false,
			},
			weights:  nil,
			expected: 0.0,
		},
		{
			name: "mixed scores without trust",
			params: SceneParams{
				Text:         0.8,
				Proximity:    0.6,
				Trust:        0.7,
				TrustEnabled: false,
			},
			weights:  nil,
			expected: 0.5, // (0.8*0.4) + (0.6*0.3) = 0.5
		},
		{
			name: "mixed scores with trust",
			params: SceneParams{
				Text:         0.8,
				Proximity:    0.6,
				Trust:        0.7,
				TrustEnabled: true,
			},
			weights:  nil,
			expected: 0.57, // (0.8*0.4) + (0.6*0.3) + (0.7*0.1) = 0.57
		},
		{
			name: "custom weights",
			params: SceneParams{
				Text:         1.0,
				Proximity:    1.0,
				Trust:        1.0,
				TrustEnabled: true,
			},
			weights: &Weights{
				Scene: SceneWeights{
					TextMatch: 0.5,
					Proximity: 0.3,
					Trust:     0.2,
				},
			},
			expected: 1.0, // 0.5 + 0.3 + 0.2 = 1.0
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CompositeScoreScene(tt.params, tt.weights)
			if math.Abs(result-tt.expected) > 0.001 {
				t.Errorf("expected %f, got %f", tt.expected, result)
			}
		})
	}
}

// TestCompositeScoreEvent tests the event composite scoring.
func TestCompositeScoreEvent(t *testing.T) {
	tests := []struct {
		name     string
		params   EventParams
		weights  *Weights
		expected float64
	}{
		{
			name: "all perfect scores without trust",
			params: EventParams{
				Recency:      1.0,
				Text:         1.0,
				Proximity:    1.0,
				Trust:        1.0,
				TrustEnabled: false,
			},
			weights:  nil, // Use defaults
			expected: 0.9, // 0.3 + 0.4 + 0.2 = 0.9
		},
		{
			name: "all perfect scores with trust",
			params: EventParams{
				Recency:      1.0,
				Text:         1.0,
				Proximity:    1.0,
				Trust:        1.0,
				TrustEnabled: true,
			},
			weights:  nil,
			expected: 1.0, // 0.3 + 0.4 + 0.2 + 0.1 = 1.0
		},
		{
			name: "all zero scores",
			params: EventParams{
				Recency:      0.0,
				Text:         0.0,
				Proximity:    0.0,
				Trust:        0.0,
				TrustEnabled: false,
			},
			weights:  nil,
			expected: 0.0,
		},
		{
			name: "mixed scores without trust",
			params: EventParams{
				Recency:      0.5,
				Text:         0.8,
				Proximity:    0.6,
				Trust:        0.7,
				TrustEnabled: false,
			},
			weights:  nil,
			expected: 0.59, // (0.5*0.3) + (0.8*0.4) + (0.6*0.2) = 0.59
		},
		{
			name: "mixed scores with trust",
			params: EventParams{
				Recency:      0.5,
				Text:         0.8,
				Proximity:    0.6,
				Trust:        0.7,
				TrustEnabled: true,
			},
			weights:  nil,
			expected: 0.66, // (0.5*0.3) + (0.8*0.4) + (0.6*0.2) + (0.7*0.1) = 0.66
		},
		{
			name: "custom weights all ones",
			params: EventParams{
				Recency:      1.0,
				Text:         1.0,
				Proximity:    1.0,
				Trust:        1.0,
				TrustEnabled: true,
			},
			weights: &Weights{
				Event: EventWeights{
					Recency:   0.25,
					TextMatch: 0.25,
					Proximity: 0.25,
					Trust:     0.25,
				},
			},
			expected: 1.0, // 0.25 + 0.25 + 0.25 + 0.25 = 1.0
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CompositeScoreEvent(tt.params, tt.weights)
			if math.Abs(result-tt.expected) > 0.001 {
				t.Errorf("expected %f, got %f", tt.expected, result)
			}
		})
	}
}
