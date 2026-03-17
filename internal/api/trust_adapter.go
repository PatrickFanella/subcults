// Package api provides HTTP handlers for the Subcults API.
package api

import (
	"github.com/onnwee/subcults/internal/trust"
)

// trustScoreStoreAdapter adapts trust.InMemoryScoreStore to implement TrustScoreStore interface.
type trustScoreStoreAdapter struct {
	store *trust.InMemoryScoreStore
}

// GetScore retrieves a trust score for a scene.
func (a *trustScoreStoreAdapter) GetScore(sceneID string) (*TrustScore, error) {
	score, err := a.store.GetScore(sceneID)
	if err != nil {
		return nil, err
	}
	if score == nil {
		return nil, nil
	}
	return &TrustScore{
		SceneID: score.SceneID,
		Score:   score.Score,
	}, nil
}

// NewTrustScoreStoreAdapter creates an adapter for the trust score store.
func NewTrustScoreStoreAdapter(store *trust.InMemoryScoreStore) TrustScoreStore {
	if store == nil {
		return nil
	}
	return &trustScoreStoreAdapter{store: store}
}
