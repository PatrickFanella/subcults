// Package trust provides trust score computation for scenes based on
// membership and alliance relationships.
package trust

import "sync"

// InMemoryDataSource is an in-memory implementation of DataSource for testing.
type InMemoryDataSource struct {
	mu          sync.RWMutex
	memberships map[string][]Membership // sceneID -> memberships
	alliances   map[string][]Alliance   // sceneID -> alliances
}

// NewInMemoryDataSource creates a new in-memory data source.
func NewInMemoryDataSource() *InMemoryDataSource {
	return &InMemoryDataSource{
		memberships: make(map[string][]Membership),
		alliances:   make(map[string][]Alliance),
	}
}

// GetMembershipsByScene returns all memberships for a scene.
func (s *InMemoryDataSource) GetMembershipsByScene(sceneID string) ([]Membership, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	memberships := s.memberships[sceneID]
	// Return a copy to avoid external modification
	result := make([]Membership, len(memberships))
	copy(result, memberships)
	return result, nil
}

// GetAlliancesByScene returns all alliances where the scene is the source.
func (s *InMemoryDataSource) GetAlliancesByScene(sceneID string) ([]Alliance, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	alliances := s.alliances[sceneID]
	// Return a copy to avoid external modification
	result := make([]Alliance, len(alliances))
	copy(result, alliances)
	return result, nil
}

// AddMembership adds a membership to the data source.
func (s *InMemoryDataSource) AddMembership(m Membership) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.memberships[m.SceneID] = append(s.memberships[m.SceneID], m)
}

// AddAlliance adds an alliance to the data source.
func (s *InMemoryDataSource) AddAlliance(a Alliance) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.alliances[a.FromSceneID] = append(s.alliances[a.FromSceneID], a)
}

// ClearMemberships removes all memberships for a scene.
func (s *InMemoryDataSource) ClearMemberships(sceneID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.memberships, sceneID)
}

// ClearAlliances removes all alliances for a scene.
func (s *InMemoryDataSource) ClearAlliances(sceneID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.alliances, sceneID)
}

// InMemoryScoreStore is an in-memory implementation of ScoreStore for testing.
type InMemoryScoreStore struct {
	mu     sync.RWMutex
	scores map[string]SceneTrustScore // sceneID -> score
}

// NewInMemoryScoreStore creates a new in-memory score store.
func NewInMemoryScoreStore() *InMemoryScoreStore {
	return &InMemoryScoreStore{
		scores: make(map[string]SceneTrustScore),
	}
}

// SaveScore stores a computed trust score.
func (s *InMemoryScoreStore) SaveScore(score SceneTrustScore) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.scores[score.SceneID] = score
	return nil
}

// GetScore retrieves a trust score by scene ID.
func (s *InMemoryScoreStore) GetScore(sceneID string) (*SceneTrustScore, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	score, ok := s.scores[sceneID]
	if !ok {
		return nil, nil
	}
	// Return a copy to avoid external modification
	return &score, nil
}

// AllScores returns all stored scores (for testing).
func (s *InMemoryScoreStore) AllScores() map[string]SceneTrustScore {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make(map[string]SceneTrustScore, len(s.scores))
	for k, v := range s.scores {
		result[k] = v
	}
	return result
}
