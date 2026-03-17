package scene

import (
	"math"
	"testing"
	"time"
)

// TestCalculateRecencyWeight tests the recency weight calculation.
func TestCalculateRecencyWeight(t *testing.T) {
	now := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	windowSpan := 24 * time.Hour // 24 hour window

	tests := []struct {
		name        string
		eventStart  time.Time
		expectedMin float64
		expectedMax float64
	}{
		{
			name:        "event in the past",
			eventStart:  now.Add(-1 * time.Hour),
			expectedMin: 1.0,
			expectedMax: 1.0,
		},
		{
			name:        "event happening now",
			eventStart:  now,
			expectedMin: 1.0,
			expectedMax: 1.0,
		},
		{
			name:        "event 6 hours in future (75% through window)",
			eventStart:  now.Add(6 * time.Hour),
			expectedMin: 0.74,
			expectedMax: 0.76,
		},
		{
			name:        "event 12 hours in future (50% through window)",
			eventStart:  now.Add(12 * time.Hour),
			expectedMin: 0.49,
			expectedMax: 0.51,
		},
		{
			name:        "event 18 hours in future (25% through window)",
			eventStart:  now.Add(18 * time.Hour),
			expectedMin: 0.24,
			expectedMax: 0.26,
		},
		{
			name:        "event 24 hours in future (end of window)",
			eventStart:  now.Add(24 * time.Hour),
			expectedMin: 0.0,
			expectedMax: 0.01,
		},
		{
			name:        "event beyond window",
			eventStart:  now.Add(30 * time.Hour),
			expectedMin: 0.0,
			expectedMax: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			weight := CalculateRecencyWeight(tt.eventStart, now, windowSpan)

			if weight < tt.expectedMin || weight > tt.expectedMax {
				t.Errorf("expected weight in range [%f, %f], got %f",
					tt.expectedMin, tt.expectedMax, weight)
			}

			// Verify weight is clamped to [0, 1]
			if weight < 0.0 || weight > 1.0 {
				t.Errorf("weight %f is outside valid range [0.0, 1.0]", weight)
			}
		})
	}
}

// TestCalculateRecencyWeight_ZeroWindowSpan tests edge case of zero window span.
func TestCalculateRecencyWeight_ZeroWindowSpan(t *testing.T) {
	now := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	eventStart := now.Add(1 * time.Hour)

	weight := CalculateRecencyWeight(eventStart, now, 0)

	if weight != 1.0 {
		t.Errorf("expected weight 1.0 for zero window span, got %f", weight)
	}
}

// TestCalculateTextMatchScore tests text matching for events.
func TestCalculateTextMatchScore(t *testing.T) {
	event := &Event{
		Title:       "Electronic Music Night",
		Description: "Join us for an amazing techno party",
		Tags:        []string{"electronic", "techno", "underground"},
	}

	tests := []struct {
		name          string
		query         string
		expectedScore float64
	}{
		{
			name:          "empty query",
			query:         "",
			expectedScore: 1.0,
		},
		{
			name:          "exact title match",
			query:         "electronic music",
			expectedScore: 1.0,
		},
		{
			name:          "case insensitive title match",
			query:         "ELECTRONIC",
			expectedScore: 1.0,
		},
		{
			name:          "partial title match",
			query:         "music",
			expectedScore: 1.0,
		},
		{
			name:          "description match",
			query:         "techno party",
			expectedScore: 0.8,
		},
		{
			name:          "tag match",
			query:         "underground",
			expectedScore: 0.6,
		},
		{
			name:          "no match",
			query:         "jazz",
			expectedScore: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := CalculateTextMatchScore(event, tt.query)
			if score != tt.expectedScore {
				t.Errorf("expected score %f, got %f", tt.expectedScore, score)
			}
		})
	}
}

// TestCalculateProximityScore tests distance-based proximity scoring.
func TestCalculateProximityScore(t *testing.T) {
	// Center point (NYC)
	centerLat := 40.7128
	centerLng := -74.0060

	tests := []struct {
		name        string
		event       *Event
		expectedMin float64
	}{
		{
			name: "event at center point",
			event: &Event{
				PrecisePoint: &Point{Lat: centerLat, Lng: centerLng},
			},
			expectedMin: 0.99, // Should be very close to 1.0
		},
		{
			name: "event nearby (0.01 degrees away)",
			event: &Event{
				PrecisePoint: &Point{Lat: centerLat + 0.01, Lng: centerLng + 0.01},
			},
			expectedMin: 0.95, // Still very high score
		},
		{
			name: "event further away (0.1 degrees away)",
			event: &Event{
				PrecisePoint: &Point{Lat: centerLat + 0.1, Lng: centerLng + 0.1},
			},
			expectedMin: 0.8, // Moderate score
		},
		{
			name: "event without location",
			event: &Event{
				PrecisePoint: nil,
			},
			expectedMin: 0.49, // Default score
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := CalculateProximityScore(tt.event, centerLat, centerLng)

			if score < tt.expectedMin {
				t.Errorf("expected score >= %f, got %f", tt.expectedMin, score)
			}

			// Verify score is in [0, 1]
			if score < 0.0 || score > 1.0 {
				t.Errorf("score %f is outside valid range [0.0, 1.0]", score)
			}
		})
	}
}

// TestCalculateCompositeScore tests the composite ranking score calculation.
func TestCalculateCompositeScore(t *testing.T) {
	weights := DefaultEventRankingWeights

	tests := []struct {
		name         string
		recency      float64
		textMatch    float64
		proximity    float64
		trust        float64
		includeTrust bool
		expected     float64
	}{
		{
			name:         "all perfect scores without trust",
			recency:      1.0,
			textMatch:    1.0,
			proximity:    1.0,
			trust:        1.0,
			includeTrust: false,
			expected:     0.9, // 0.3 + 0.4 + 0.2 = 0.9
		},
		{
			name:         "all perfect scores with trust",
			recency:      1.0,
			textMatch:    1.0,
			proximity:    1.0,
			trust:        1.0,
			includeTrust: true,
			expected:     1.0, // 0.3 + 0.4 + 0.2 + 0.1 = 1.0
		},
		{
			name:         "all zero scores",
			recency:      0.0,
			textMatch:    0.0,
			proximity:    0.0,
			trust:        0.0,
			includeTrust: false,
			expected:     0.0,
		},
		{
			name:         "mixed scores without trust",
			recency:      0.5,
			textMatch:    0.8,
			proximity:    0.6,
			trust:        0.7,
			includeTrust: false,
			expected:     0.59, // (0.5*0.3) + (0.8*0.4) + (0.6*0.2) = 0.59
		},
		{
			name:         "mixed scores with trust",
			recency:      0.5,
			textMatch:    0.8,
			proximity:    0.6,
			trust:        0.7,
			includeTrust: true,
			expected:     0.66, // (0.5*0.3) + (0.8*0.4) + (0.6*0.2) + (0.7*0.1) = 0.66
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := CalculateCompositeScore(
				tt.recency,
				tt.textMatch,
				tt.proximity,
				tt.trust,
				weights,
				tt.includeTrust,
			)

			// Allow small floating point tolerance
			if math.Abs(score-tt.expected) > 0.001 {
				t.Errorf("expected score %f, got %f", tt.expected, score)
			}
		})
	}
}

// TestCalculateCompositeScore_CustomWeights tests custom weight configuration.
func TestCalculateCompositeScore_CustomWeights(t *testing.T) {
	// Custom weights that sum to 1.0
	customWeights := EventRankingWeights{
		Recency:   0.5,
		TextMatch: 0.3,
		Proximity: 0.1,
		Trust:     0.1,
	}

	score := CalculateCompositeScore(1.0, 1.0, 1.0, 1.0, customWeights, true)

	if math.Abs(score-1.0) > 0.001 {
		t.Errorf("expected score 1.0 with custom weights, got %f", score)
	}
}

// TestCalculateSceneTextMatchScore tests text match scoring for scenes.
func TestCalculateSceneTextMatchScore(t *testing.T) {
	tests := []struct {
		name          string
		scene         *Scene
		query         string
		expectedScore float64
	}{
		{
			name: "exact name match",
			scene: &Scene{
				Name:        "Electronic Music",
				Description: "Rock concert",
				Tags:        []string{"jazz"},
			},
			query:         "electronic",
			expectedScore: 1.0,
		},
		{
			name: "description match",
			scene: &Scene{
				Name:        "Music Venue",
				Description: "Electronic music events",
				Tags:        []string{"rock"},
			},
			query:         "electronic",
			expectedScore: 0.7,
		},
		{
			name: "tag match",
			scene: &Scene{
				Name:        "Music Venue",
				Description: "Live performances",
				Tags:        []string{"electronic", "rock"},
			},
			query:         "electronic",
			expectedScore: 0.5,
		},
		{
			name: "no match",
			scene: &Scene{
				Name:        "Jazz Venue",
				Description: "Live jazz",
				Tags:        []string{"jazz"},
			},
			query:         "electronic",
			expectedScore: 0.0,
		},
		{
			name: "empty query matches all",
			scene: &Scene{
				Name: "Any Scene",
			},
			query:         "",
			expectedScore: 1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := CalculateSceneTextMatchScore(tt.scene, tt.query)
			if score != tt.expectedScore {
				t.Errorf("expected score %.2f, got %.2f", tt.expectedScore, score)
			}
		})
	}
}

// TestCalculateSceneProximityScore tests proximity scoring for scenes.
func TestCalculateSceneProximityScore(t *testing.T) {
	centerLat := 40.7128
	centerLng := -74.0060

	tests := []struct {
		name        string
		scene       *Scene
		expectedMin float64
		expectedMax float64
	}{
		{
			name: "exact center location without geohash",
			scene: &Scene{
				PrecisePoint: &Point{Lat: 40.7128, Lng: -74.0060},
			},
			// Blended: 0.5*defaultGeohash(0.5) + 0.5*distance(1.0) = 0.75
			expectedMin: 0.75,
			expectedMax: 0.75,
		},
		{
			name: "nearby location without geohash",
			scene: &Scene{
				PrecisePoint: &Point{Lat: 40.7, Lng: -74.0},
			},
			// Blended: 0.5*defaultGeohash(0.5) + 0.5*distance(~0.986) ≈ 0.74
			expectedMin: 0.70,
			expectedMax: 0.80,
		},
		{
			name: "no location",
			scene: &Scene{
				PrecisePoint: nil,
			},
			expectedMin: 0.5,
			expectedMax: 0.5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := CalculateSceneProximityScore(tt.scene, centerLat, centerLng)
			if score < tt.expectedMin || score > tt.expectedMax {
				t.Errorf("expected score between %.2f and %.2f, got %.2f", tt.expectedMin, tt.expectedMax, score)
			}
		})
	}
}

// TestCalculateSceneCompositeScore tests composite score calculation.
func TestCalculateSceneCompositeScore(t *testing.T) {
	weights := DefaultSceneRankingWeights

	tests := []struct {
		name         string
		textMatch    float64
		proximity    float64
		trust        float64
		includeTrust bool
		expected     float64
	}{
		{
			name:         "all perfect scores with trust",
			textMatch:    1.0,
			proximity:    1.0,
			trust:        1.0,
			includeTrust: true,
			expected:     1.0, // 0.6 + 0.25 + 0.15
		},
		{
			name:         "all perfect scores without trust",
			textMatch:    1.0,
			proximity:    1.0,
			trust:        1.0,
			includeTrust: false,
			expected:     0.85, // 0.6 + 0.25 (no trust)
		},
		{
			name:         "half scores with trust",
			textMatch:    0.5,
			proximity:    0.5,
			trust:        0.5,
			includeTrust: true,
			expected:     0.5, // (0.6 + 0.25 + 0.15) * 0.5
		},
		{
			name:         "text match only",
			textMatch:    1.0,
			proximity:    0.0,
			trust:        0.0,
			includeTrust: false,
			expected:     0.6, // 1.0 * 0.6
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := CalculateSceneCompositeScore(
				tt.textMatch,
				tt.proximity,
				tt.trust,
				weights,
				tt.includeTrust,
			)
			// Allow small floating point variance
			if math.Abs(score-tt.expected) > 0.001 {
				t.Errorf("expected score %.3f, got %.3f", tt.expected, score)
			}
		})
	}
}

// TestSceneCursorEncoding tests cursor encoding and decoding.
func TestSceneCursorEncoding(t *testing.T) {
	score := 0.85
	id := "scene-123"

	// Encode cursor
	encoded := EncodeSceneCursor(score, id)
	if encoded == "" {
		t.Fatal("encoded cursor should not be empty")
	}

	// Decode cursor
	decoded, err := DecodeSceneCursor(encoded)
	if err != nil {
		t.Fatalf("failed to decode cursor: %v", err)
	}

	if decoded.Score != score {
		t.Errorf("expected score %.2f, got %.2f", score, decoded.Score)
	}

	if decoded.ID != id {
		t.Errorf("expected ID %s, got %s", id, decoded.ID)
	}
}

// TestDecodeSceneCursor_InvalidInput tests cursor decoding with invalid input.
func TestDecodeSceneCursor_InvalidInput(t *testing.T) {
	tests := []struct {
		name   string
		cursor string
	}{
		{
			name:   "empty cursor",
			cursor: "",
		},
		{
			name:   "invalid base64",
			cursor: "not-base64!@#",
		},
		{
			name:   "invalid json",
			cursor: "aW52YWxpZCBqc29u", // base64 of "invalid json"
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decoded, err := DecodeSceneCursor(tt.cursor)
			if tt.cursor == "" {
				// Empty cursor should return nil without error
				if err != nil {
					t.Errorf("empty cursor should not error, got: %v", err)
				}
				if decoded != nil {
					t.Error("empty cursor should return nil")
				}
			} else {
				// Invalid cursors should error
				if err == nil {
					t.Error("expected error for invalid cursor")
				}
			}
		})
	}
}
