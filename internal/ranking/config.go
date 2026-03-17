package ranking

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"sync"
)

// activeWeightsCache holds the process-wide calibration weights set at startup.
var activeWeightsCache struct {
	mu      sync.RWMutex
	weights *Weights
}

// SetActiveWeights stores calibrated weights for process-wide use.
// Call once during application initialization after loading the calibration file.
// Thread-safe via mutex.
func SetActiveWeights(w *Weights) {
	activeWeightsCache.mu.Lock()
	defer activeWeightsCache.mu.Unlock()
	activeWeightsCache.weights = w
}

// GetActiveWeights returns the active calibration weights.
// Falls back to DefaultWeights() when SetActiveWeights has not been called.
// Thread-safe via mutex.
func GetActiveWeights() *Weights {
	activeWeightsCache.mu.RLock()
	defer activeWeightsCache.mu.RUnlock()
	if activeWeightsCache.weights != nil {
		return activeWeightsCache.weights
	}
	return DefaultWeights()
}

// SceneWeights defines the ranking weights for scene search.
type SceneWeights struct {
	TextMatch float64 `json:"text_match"` // Weight for text relevance (default: 0.4)
	Proximity float64 `json:"proximity"`  // Weight for geographic proximity (default: 0.3)
	Trust     float64 `json:"trust"`      // Weight for trust score (default: 0.1)
}

// EventWeights defines the ranking weights for event search.
type EventWeights struct {
	Recency   float64 `json:"recency"`    // Weight for time recency (default: 0.3)
	TextMatch float64 `json:"text_match"` // Weight for text relevance (default: 0.4)
	Proximity float64 `json:"proximity"`  // Weight for geographic proximity (default: 0.2)
	Trust     float64 `json:"trust"`      // Weight for trust score (default: 0.1)
}

// Weights holds all ranking weight configurations.
type Weights struct {
	Scene SceneWeights `json:"scene"` // Scene search weights
	Event EventWeights `json:"event"` // Event search weights
}

// CalibrationConfig represents the JSON structure of the calibration file.
type CalibrationConfig struct {
	Version string  `json:"version"` // Config version for future compatibility
	Weights Weights `json:"weights"` // Weight configurations
}

// DefaultWeights returns the default ranking weight configuration.
// These weights are optimized for balanced discovery across different ranking factors.
//
// Scene formula: composite_score = (text * 0.4) + (proximity * 0.3) + (trust * 0.1)
// - Prioritizes text match for targeted search
// - Geographic proximity for local discovery
// - Trust as a reputation signal
// - Max score without trust: 0.7, with trust: 0.8
//
// Event formula: composite_score = (recency * 0.3) + (text * 0.4) + (proximity * 0.2) + (trust * 0.1)
// - Prioritizes text match for targeted search
// - Recency favors events happening sooner
// - Geographic proximity for local discovery
// - Trust as a reputation signal
// - Max score without trust: 0.9, with trust: 1.0
func DefaultWeights() *Weights {
	return &Weights{
		Scene: SceneWeights{
			TextMatch: 0.4,
			Proximity: 0.3,
			Trust:     0.1,
		},
		Event: EventWeights{
			Recency:   0.3,
			TextMatch: 0.4,
			Proximity: 0.2,
			Trust:     0.1,
		},
	}
}

// LoadCalibration loads ranking weights from a JSON calibration file.
// If the file doesn't exist or can't be read, returns default weights with an error.
// The file is expected to be in JSON format matching CalibrationConfig structure.
// Partial configurations are merged with defaults for graceful degradation.
//
// Parameters:
//   - filePath: Path to the calibration JSON file
//
// Returns the loaded weights and any error encountered.
// On error, returns default weights to ensure graceful degradation.
func LoadCalibration(filePath string) (*Weights, error) {
	// Return defaults if no file path provided
	if filePath == "" {
		return DefaultWeights(), nil
	}

	// Read the calibration file
	data, err := os.ReadFile(filePath)
	if err != nil {
		slog.Warn("failed to read calibration file, using defaults",
			"path", filePath,
			"error", err)
		return DefaultWeights(), fmt.Errorf("failed to read calibration file: %w", err)
	}

	// Parse JSON
	var config CalibrationConfig
	if err := json.Unmarshal(data, &config); err != nil {
		slog.Warn("failed to parse calibration file, using defaults",
			"path", filePath,
			"error", err)
		return DefaultWeights(), fmt.Errorf("failed to parse calibration file: %w", err)
	}

	// Merge loaded weights with defaults to handle partial configurations
	defaults := DefaultWeights()
	merged := MergeCalibration(defaults, &config.Weights)
	logCalibrationOverrides(defaults, merged)

	return merged, nil
}

// MergeCalibration merges override weights with default weights.
// Only non-zero values from the override are applied.
// This allows partial overrides in the calibration file.
//
// Parameters:
//   - base: The base weights to start from (typically defaults)
//   - override: The override weights to merge in
//
// Returns a new Weights struct with merged values.
func MergeCalibration(base *Weights, override *Weights) *Weights {
	// Guard against nil base to avoid panics; fall back to defaults.
	if base == nil {
		return DefaultWeights()
	}

	// If there is no override provided, return a copy of the base.
	if override == nil {
		result := *base
		return &result
	}

	result := *base // Copy base

	// Merge scene weights
	if override.Scene.TextMatch != 0 {
		result.Scene.TextMatch = override.Scene.TextMatch
	}
	if override.Scene.Proximity != 0 {
		result.Scene.Proximity = override.Scene.Proximity
	}
	if override.Scene.Trust != 0 {
		result.Scene.Trust = override.Scene.Trust
	}

	// Merge event weights
	if override.Event.Recency != 0 {
		result.Event.Recency = override.Event.Recency
	}
	if override.Event.TextMatch != 0 {
		result.Event.TextMatch = override.Event.TextMatch
	}
	if override.Event.Proximity != 0 {
		result.Event.Proximity = override.Event.Proximity
	}
	if override.Event.Trust != 0 {
		result.Event.Trust = override.Event.Trust
	}

	return &result
}

// logCalibrationOverrides logs which weights were overridden from defaults.
func logCalibrationOverrides(defaults *Weights, loaded *Weights) {
	var overrides []string

	// Check scene weight overrides
	if loaded.Scene.TextMatch != defaults.Scene.TextMatch {
		overrides = append(overrides, fmt.Sprintf("scene.text_match: %.2f -> %.2f",
			defaults.Scene.TextMatch, loaded.Scene.TextMatch))
	}
	if loaded.Scene.Proximity != defaults.Scene.Proximity {
		overrides = append(overrides, fmt.Sprintf("scene.proximity: %.2f -> %.2f",
			defaults.Scene.Proximity, loaded.Scene.Proximity))
	}
	if loaded.Scene.Trust != defaults.Scene.Trust {
		overrides = append(overrides, fmt.Sprintf("scene.trust: %.2f -> %.2f",
			defaults.Scene.Trust, loaded.Scene.Trust))
	}

	// Check event weight overrides
	if loaded.Event.Recency != defaults.Event.Recency {
		overrides = append(overrides, fmt.Sprintf("event.recency: %.2f -> %.2f",
			defaults.Event.Recency, loaded.Event.Recency))
	}
	if loaded.Event.TextMatch != defaults.Event.TextMatch {
		overrides = append(overrides, fmt.Sprintf("event.text_match: %.2f -> %.2f",
			defaults.Event.TextMatch, loaded.Event.TextMatch))
	}
	if loaded.Event.Proximity != defaults.Event.Proximity {
		overrides = append(overrides, fmt.Sprintf("event.proximity: %.2f -> %.2f",
			defaults.Event.Proximity, loaded.Event.Proximity))
	}
	if loaded.Event.Trust != defaults.Event.Trust {
		overrides = append(overrides, fmt.Sprintf("event.trust: %.2f -> %.2f",
			defaults.Event.Trust, loaded.Event.Trust))
	}

	if len(overrides) > 0 {
		slog.Info("loaded ranking calibration with overrides",
			"overrides", overrides)
	} else {
		slog.Info("loaded ranking calibration (using all defaults)")
	}
}
