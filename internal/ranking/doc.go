// Package ranking provides centralized ranking component calculations
// with calibration support for search and discovery features.
//
// Basic Usage:
//
//	// Load calibration (typically at startup)
//	weights, err := ranking.LoadCalibration("configs/ranking.calibration.json")
//	if err != nil {
//		log.Warn("using default weights", "error", err)
//	}
//
//	// Calculate scene ranking
//	sceneParams := ranking.SceneParams{
//		Text:         0.85,  // From database ts_rank
//		Proximity:    ranking.ProximityWeight(distanceMeters),
//		Trust:        0.7,   // From trust graph
//		TrustEnabled: config.RankTrustEnabled,
//	}
//	score := ranking.CompositeScoreScene(sceneParams, weights)
//
//	// Calculate event ranking
//	eventParams := ranking.EventParams{
//		Recency:      ranking.RecencyWeight(event.StartTime, windowSpan),
//		Text:         0.90,  // From database ts_rank
//		Proximity:    ranking.ProximityWeight(distanceMeters),
//		Trust:        0.6,   // From trust graph
//		TrustEnabled: config.RankTrustEnabled,
//	}
//	score := ranking.CompositeScoreEvent(eventParams, weights)
//
// Weight Functions:
//
// All weight functions return values in the [0, 1] range and are designed
// to be composable. Use them to calculate individual ranking components
// before combining them with composite score functions.
//
// Calibration:
//
// The calibration system allows deploy-time tuning of ranking weights via
// JSON configuration files loaded at startup. This enables A/B testing and
// optimization without code changes (but requires a redeploy or restart to
// pick up new configuration). See configs/ranking.calibration.json for the
// default configuration.
package ranking
