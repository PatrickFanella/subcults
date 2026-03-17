// Package trust provides trust score computation for scenes based on
// membership and alliance relationships.
package trust

import (
	"errors"
	"sync"
	"time"
)

// RoleMultiplier defines the trust weight multiplier for different membership roles.
// Multipliers affect how much each member's trust weight contributes to scene trust score.
// - owner: Scene owner (highest authority)
// - curator: Content curator (elevated privileges)
// - member: Regular member (baseline)
// - guest: Limited access member (lowest weight)
var RoleMultiplier = map[string]float64{
	"owner":   1.0,
	"curator": 0.8,
	"member":  0.5,
	"guest":   0.3,
}

// DefaultRoleMultiplier is used when a role is not found in the RoleMultiplier map.
const DefaultRoleMultiplier = 0.5

// Valid role constants
const (
	RoleOwner   = "owner"
	RoleCurator = "curator"
	RoleMember  = "member"
	RoleGuest   = "guest"
)

// Validation errors
var (
	ErrInvalidRole        = errors.New("invalid role: must be owner, curator, member, or guest")
	ErrInvalidTrustWeight = errors.New("invalid trust weight: must be between 0.0 and 1.0")
)

// ValidRole checks if a role string is valid.
// Returns true if the role is one of: owner, curator, member, guest.
func ValidRole(role string) bool {
	_, exists := RoleMultiplier[role]
	return exists
}

// ValidateTrustWeight checks if a trust weight is within valid bounds (0.0-1.0).
// Returns ErrInvalidTrustWeight if the weight is out of bounds.
func ValidateTrustWeight(weight float64) error {
	if weight < 0.0 || weight > 1.0 {
		return ErrInvalidTrustWeight
	}
	return nil
}

// Membership represents a user's membership in a scene with their role.
type Membership struct {
	SceneID     string  `json:"scene_id"`
	UserDID     string  `json:"user_did"`
	Role        string  `json:"role"`
	TrustWeight float64 `json:"trust_weight"` // Base trust weight (0.0-1.0)
}

// EffectiveWeight returns the effective trust weight for this membership.
// It is computed as base TrustWeight multiplied by the role multiplier.
// This value is used in trust score calculations.
func (m *Membership) EffectiveWeight() float64 {
	multiplier, ok := RoleMultiplier[m.Role]
	if !ok {
		multiplier = DefaultRoleMultiplier
	}
	return m.TrustWeight * multiplier
}

// Validate checks if the membership has valid role and trust weight.
// Returns an error if either field is invalid.
func (m *Membership) Validate() error {
	if !ValidRole(m.Role) {
		return ErrInvalidRole
	}
	if err := ValidateTrustWeight(m.TrustWeight); err != nil {
		return err
	}
	return nil
}

// Alliance represents a trust relationship between two scenes.
type Alliance struct {
	FromSceneID string  `json:"from_scene_id"`
	ToSceneID   string  `json:"to_scene_id"`
	Weight      float64 `json:"weight"` // Trust weight (0.0-1.0)
}

// SceneTrustScore represents the computed trust score for a scene.
type SceneTrustScore struct {
	SceneID    string    `json:"scene_id"`
	Score      float64   `json:"score"`
	ComputedAt time.Time `json:"computed_at"`
}

// ComputeTrustScore calculates the trust score for a scene using the formula:
// score = avg(alliance weights) * avg(effective membership weights)
//
// Effective weight = base trust_weight * role multiplier
// If there are no alliances, alliance average defaults to 1.0.
// If there are no memberships, the score is 0.0.
func ComputeTrustScore(memberships []Membership, alliances []Alliance) float64 {
	if len(memberships) == 0 {
		return 0.0
	}

	// Calculate average alliance weight
	allianceAvg := 1.0
	if len(alliances) > 0 {
		var allianceSum float64
		for _, a := range alliances {
			allianceSum += a.Weight
		}
		allianceAvg = allianceSum / float64(len(alliances))
	}

	// Calculate average effective membership weight
	var membershipSum float64
	for _, m := range memberships {
		membershipSum += m.EffectiveWeight()
	}
	membershipAvg := membershipSum / float64(len(memberships))

	return allianceAvg * membershipAvg
}

// DirtyTracker tracks which scenes have pending changes that require
// trust score recomputation. Thread-safe via RWMutex.
type DirtyTracker struct {
	mu         sync.RWMutex
	dirtyFlags map[string]time.Time // sceneID -> time marked dirty
}

// NewDirtyTracker creates a new DirtyTracker instance.
func NewDirtyTracker() *DirtyTracker {
	return &DirtyTracker{
		dirtyFlags: make(map[string]time.Time),
	}
}

// MarkDirty marks a scene as needing trust score recomputation.
func (t *DirtyTracker) MarkDirty(sceneID string) {
	t.mu.Lock()
	t.dirtyFlags[sceneID] = time.Now()
	t.mu.Unlock()
}

// ClearDirty removes the dirty flag for a scene after recomputation.
func (t *DirtyTracker) ClearDirty(sceneID string) {
	t.mu.Lock()
	delete(t.dirtyFlags, sceneID)
	t.mu.Unlock()
}

// GetDirtyScenes returns a list of scene IDs that are marked dirty.
// Returns a copy to avoid external modification.
func (t *DirtyTracker) GetDirtyScenes() []string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	scenes := make([]string, 0, len(t.dirtyFlags))
	for sceneID := range t.dirtyFlags {
		scenes = append(scenes, sceneID)
	}
	return scenes
}

// IsDirty checks if a specific scene is marked as dirty.
func (t *DirtyTracker) IsDirty(sceneID string) bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	_, exists := t.dirtyFlags[sceneID]
	return exists
}

// DirtyCount returns the number of scenes marked as dirty.
func (t *DirtyTracker) DirtyCount() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return len(t.dirtyFlags)
}
