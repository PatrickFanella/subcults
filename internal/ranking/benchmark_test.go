package ranking

import (
	"testing"
	"time"
)

// BenchmarkTextWeight benchmarks the text weight calculation.
func BenchmarkTextWeight(b *testing.B) {
	for i := 0; i < b.N; i++ {
		TextWeight(0.75, 0.4)
	}
}

// BenchmarkProximityWeight benchmarks the proximity weight calculation.
func BenchmarkProximityWeight(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ProximityWeight(1500.0)
	}
}

// BenchmarkRecencyWeight benchmarks the recency weight calculation.
func BenchmarkRecencyWeight(b *testing.B) {
	startTime := time.Now().Add(6 * time.Hour)
	windowSpan := 24 * time.Hour

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		RecencyWeight(startTime, windowSpan)
	}
}

// BenchmarkTrustWeight benchmarks the trust weight calculation.
func BenchmarkTrustWeight(b *testing.B) {
	for i := 0; i < b.N; i++ {
		TrustWeight(0.8, true)
	}
}

// BenchmarkCompositeScoreScene benchmarks the scene composite score calculation.
func BenchmarkCompositeScoreScene(b *testing.B) {
	params := SceneParams{
		Text:         0.8,
		Proximity:    0.6,
		Trust:        0.7,
		TrustEnabled: true,
	}
	weights := DefaultWeights()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CompositeScoreScene(params, weights)
	}
}

// BenchmarkCompositeScoreEvent benchmarks the event composite score calculation.
func BenchmarkCompositeScoreEvent(b *testing.B) {
	params := EventParams{
		Recency:      0.5,
		Text:         0.8,
		Proximity:    0.6,
		Trust:        0.7,
		TrustEnabled: true,
	}
	weights := DefaultWeights()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CompositeScoreEvent(params, weights)
	}
}

// BenchmarkCompositeScoreScene_WithNilWeights benchmarks scene scoring with nil weights.
func BenchmarkCompositeScoreScene_WithNilWeights(b *testing.B) {
	params := SceneParams{
		Text:         0.8,
		Proximity:    0.6,
		Trust:        0.7,
		TrustEnabled: true,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CompositeScoreScene(params, nil)
	}
}

// BenchmarkCompositeScoreEvent_WithNilWeights benchmarks event scoring with nil weights.
func BenchmarkCompositeScoreEvent_WithNilWeights(b *testing.B) {
	params := EventParams{
		Recency:      0.5,
		Text:         0.8,
		Proximity:    0.6,
		Trust:        0.7,
		TrustEnabled: true,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CompositeScoreEvent(params, nil)
	}
}

// BenchmarkFullEventRanking benchmarks a complete event ranking workflow.
// This simulates calculating all components and the composite score.
func BenchmarkFullEventRanking(b *testing.B) {
	startTime := time.Now().Add(6 * time.Hour)
	windowSpan := 24 * time.Hour
	weights := DefaultWeights()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Calculate individual components (raw scores, not pre-weighted)
		recency := RecencyWeight(startTime, windowSpan)
		text := 0.85 // Raw text match score from database
		proximity := ProximityWeight(1500.0)
		trust := TrustWeight(0.75, true)

		// Calculate composite score (applies weights internally)
		params := EventParams{
			Recency:      recency,
			Text:         text,
			Proximity:    proximity,
			Trust:        trust,
			TrustEnabled: true,
		}
		_ = CompositeScoreEvent(params, weights)
	}
}

// BenchmarkFullSceneRanking benchmarks a complete scene ranking workflow.
func BenchmarkFullSceneRanking(b *testing.B) {
	weights := DefaultWeights()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Calculate individual components (raw scores, not pre-weighted)
		text := 0.85 // Raw text match score from database
		proximity := ProximityWeight(2500.0)
		trust := TrustWeight(0.65, true)

		// Calculate composite score (applies weights internally)
		params := SceneParams{
			Text:         text,
			Proximity:    proximity,
			Trust:        trust,
			TrustEnabled: true,
		}
		_ = CompositeScoreScene(params, weights)
	}
}
