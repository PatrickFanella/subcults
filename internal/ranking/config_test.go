package ranking

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"testing"
)

// TestDefaultWeights verifies the default weight configuration.
func TestDefaultWeights(t *testing.T) {
	weights := DefaultWeights()

	// Verify scene weights
	if weights.Scene.TextMatch != 0.4 {
		t.Errorf("expected scene text_match 0.4, got %f", weights.Scene.TextMatch)
	}
	if weights.Scene.Proximity != 0.3 {
		t.Errorf("expected scene proximity 0.3, got %f", weights.Scene.Proximity)
	}
	if weights.Scene.Trust != 0.1 {
		t.Errorf("expected scene trust 0.1, got %f", weights.Scene.Trust)
	}

	// Verify event weights
	if weights.Event.Recency != 0.3 {
		t.Errorf("expected event recency 0.3, got %f", weights.Event.Recency)
	}
	if weights.Event.TextMatch != 0.4 {
		t.Errorf("expected event text_match 0.4, got %f", weights.Event.TextMatch)
	}
	if weights.Event.Proximity != 0.2 {
		t.Errorf("expected event proximity 0.2, got %f", weights.Event.Proximity)
	}
	if weights.Event.Trust != 0.1 {
		t.Errorf("expected event trust 0.1, got %f", weights.Event.Trust)
	}
}

// TestLoadCalibration_DefaultFile tests loading the actual default calibration file.
func TestLoadCalibration_DefaultFile(t *testing.T) {
	// Try to load the default calibration file (relative to project root)
	configPath := filepath.Join("..", "..", "configs", "ranking.calibration.json")
	weights, err := LoadCalibration(configPath)

	// If file exists, it should load without error
	if _, statErr := os.Stat(configPath); statErr == nil {
		if err != nil {
			t.Fatalf("expected no error loading default calibration file, got: %v", err)
		}

		// Verify it loaded the default values
		defaults := DefaultWeights()
		if !weightsEqual(weights, defaults) {
			t.Errorf("loaded weights don't match defaults:\nloaded: %+v\ndefaults: %+v",
				weights, defaults)
		}
	} else {
		// File doesn't exist, should return defaults with error
		if err == nil {
			t.Error("expected error when file doesn't exist")
		}
		// Should still return defaults for graceful degradation
		defaults := DefaultWeights()
		if !weightsEqual(weights, defaults) {
			t.Error("should return defaults when file doesn't exist")
		}
	}
}

// TestLoadCalibration_EmptyPath tests loading with empty file path.
func TestLoadCalibration_EmptyPath(t *testing.T) {
	weights, err := LoadCalibration("")

	if err != nil {
		t.Errorf("expected no error with empty path, got: %v", err)
	}

	defaults := DefaultWeights()
	if !weightsEqual(weights, defaults) {
		t.Error("should return defaults when path is empty")
	}
}

// TestLoadCalibration_NonExistentFile tests loading a non-existent file.
func TestLoadCalibration_NonExistentFile(t *testing.T) {
	weights, err := LoadCalibration("/nonexistent/path/to/file.json")

	if err == nil {
		t.Error("expected error when file doesn't exist")
	}

	// Should still return defaults for graceful degradation
	defaults := DefaultWeights()
	if !weightsEqual(weights, defaults) {
		t.Error("should return defaults when file doesn't exist")
	}
}

// TestLoadCalibration_CustomWeights tests loading custom weight overrides.
func TestLoadCalibration_CustomWeights(t *testing.T) {
	// Create a temporary calibration file with custom weights
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "custom.json")

	customConfig := CalibrationConfig{
		Version: "1.0",
		Weights: Weights{
			Scene: SceneWeights{
				TextMatch: 0.5,
				Proximity: 0.3,
				Trust:     0.2,
			},
			Event: EventWeights{
				Recency:   0.4,
				TextMatch: 0.3,
				Proximity: 0.2,
				Trust:     0.1,
			},
		},
	}

	data, err := json.MarshalIndent(customConfig, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal config: %v", err)
	}

	if err := os.WriteFile(tmpFile, data, 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	// Load the custom calibration
	weights, err := LoadCalibration(tmpFile)
	if err != nil {
		t.Fatalf("expected no error loading custom file, got: %v", err)
	}

	// Verify custom values were loaded
	if weights.Scene.TextMatch != 0.5 {
		t.Errorf("expected scene text_match 0.5, got %f", weights.Scene.TextMatch)
	}
	if weights.Scene.Trust != 0.2 {
		t.Errorf("expected scene trust 0.2, got %f", weights.Scene.Trust)
	}
	if weights.Event.Recency != 0.4 {
		t.Errorf("expected event recency 0.4, got %f", weights.Event.Recency)
	}
	if weights.Event.TextMatch != 0.3 {
		t.Errorf("expected event text_match 0.3, got %f", weights.Event.TextMatch)
	}
}

// TestLoadCalibration_InvalidJSON tests loading invalid JSON.
func TestLoadCalibration_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "invalid.json")

	// Write invalid JSON
	if err := os.WriteFile(tmpFile, []byte("{invalid json}"), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	weights, err := LoadCalibration(tmpFile)

	if err == nil {
		t.Error("expected error when JSON is invalid")
	}

	// Should still return defaults for graceful degradation
	defaults := DefaultWeights()
	if !weightsEqual(weights, defaults) {
		t.Error("should return defaults when JSON is invalid")
	}
}

// TestMergeCalibration tests merging override weights with defaults.
func TestMergeCalibration(t *testing.T) {
	base := DefaultWeights()

	tests := []struct {
		name     string
		override *Weights
		validate func(*testing.T, *Weights)
	}{
		{
			name: "partial scene override",
			override: &Weights{
				Scene: SceneWeights{
					TextMatch: 0.6, // Override
					Proximity: 0.0, // Don't override (zero)
					Trust:     0.0, // Don't override (zero)
				},
				Event: EventWeights{}, // All zeros
			},
			validate: func(t *testing.T, result *Weights) {
				if result.Scene.TextMatch != 0.6 {
					t.Errorf("expected scene text_match 0.6, got %f", result.Scene.TextMatch)
				}
				if result.Scene.Proximity != 0.3 {
					t.Errorf("expected scene proximity unchanged at 0.3, got %f", result.Scene.Proximity)
				}
				if result.Scene.Trust != 0.1 {
					t.Errorf("expected scene trust unchanged at 0.1, got %f", result.Scene.Trust)
				}
			},
		},
		{
			name: "partial event override",
			override: &Weights{
				Scene: SceneWeights{},
				Event: EventWeights{
					Recency:   0.5, // Override
					TextMatch: 0.0, // Don't override (zero)
					Proximity: 0.0, // Don't override (zero)
					Trust:     0.0, // Don't override (zero)
				},
			},
			validate: func(t *testing.T, result *Weights) {
				if result.Event.Recency != 0.5 {
					t.Errorf("expected event recency 0.5, got %f", result.Event.Recency)
				}
				if result.Event.TextMatch != 0.4 {
					t.Errorf("expected event text_match unchanged at 0.4, got %f", result.Event.TextMatch)
				}
			},
		},
		{
			name: "full override",
			override: &Weights{
				Scene: SceneWeights{
					TextMatch: 0.5,
					Proximity: 0.4,
					Trust:     0.1,
				},
				Event: EventWeights{
					Recency:   0.25,
					TextMatch: 0.25,
					Proximity: 0.25,
					Trust:     0.25,
				},
			},
			validate: func(t *testing.T, result *Weights) {
				if result.Scene.TextMatch != 0.5 {
					t.Errorf("expected scene text_match 0.5, got %f", result.Scene.TextMatch)
				}
				if result.Scene.Proximity != 0.4 {
					t.Errorf("expected scene proximity 0.4, got %f", result.Scene.Proximity)
				}
				if result.Event.Recency != 0.25 {
					t.Errorf("expected event recency 0.25, got %f", result.Event.Recency)
				}
				if result.Event.Trust != 0.25 {
					t.Errorf("expected event trust 0.25, got %f", result.Event.Trust)
				}
			},
		},
		{
			name:     "no override (all zeros)",
			override: &Weights{},
			validate: func(t *testing.T, result *Weights) {
				if !weightsEqual(result, base) {
					t.Error("expected weights to remain unchanged when override is all zeros")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MergeCalibration(base, tt.override)
			tt.validate(t, result)

			// Verify base wasn't modified (immutability check)
			if !weightsEqual(base, DefaultWeights()) {
				t.Error("base weights should not be modified")
			}
		})
	}
}

// TestCompositeScoreWithCalibration tests that changing calibration affects composite scores.
func TestCompositeScoreWithCalibration(t *testing.T) {
	// Default weights
	defaultWeights := DefaultWeights()

	// Custom weights that prioritize trust
	trustWeights := &Weights{
		Scene: SceneWeights{
			TextMatch: 0.2,
			Proximity: 0.2,
			Trust:     0.6, // High trust weight
		},
		Event: EventWeights{
			Recency:   0.1,
			TextMatch: 0.2,
			Proximity: 0.1,
			Trust:     0.6, // High trust weight
		},
	}

	sceneParams := SceneParams{
		Text:         0.5,
		Proximity:    0.5,
		Trust:        1.0,
		TrustEnabled: true,
	}

	eventParams := EventParams{
		Recency:      0.5,
		Text:         0.5,
		Proximity:    0.5,
		Trust:        1.0,
		TrustEnabled: true,
	}

	// Calculate scores with default weights
	defaultSceneScore := CompositeScoreScene(sceneParams, defaultWeights)
	defaultEventScore := CompositeScoreEvent(eventParams, defaultWeights)

	// Calculate scores with custom weights
	trustSceneScore := CompositeScoreScene(sceneParams, trustWeights)
	trustEventScore := CompositeScoreEvent(eventParams, trustWeights)

	// Trust-weighted scores should be higher due to high trust component
	if trustSceneScore <= defaultSceneScore {
		t.Errorf("expected trust-weighted scene score (%f) > default score (%f)",
			trustSceneScore, defaultSceneScore)
	}

	if trustEventScore <= defaultEventScore {
		t.Errorf("expected trust-weighted event score (%f) > default score (%f)",
			trustEventScore, defaultEventScore)
	}
}

// weightsEqual compares two Weights structs for equality with floating point tolerance.
func weightsEqual(a, b *Weights) bool {
	const epsilon = 0.001

	return math.Abs(a.Scene.TextMatch-b.Scene.TextMatch) < epsilon &&
		math.Abs(a.Scene.Proximity-b.Scene.Proximity) < epsilon &&
		math.Abs(a.Scene.Trust-b.Scene.Trust) < epsilon &&
		math.Abs(a.Event.Recency-b.Event.Recency) < epsilon &&
		math.Abs(a.Event.TextMatch-b.Event.TextMatch) < epsilon &&
		math.Abs(a.Event.Proximity-b.Event.Proximity) < epsilon &&
		math.Abs(a.Event.Trust-b.Event.Trust) < epsilon
}
