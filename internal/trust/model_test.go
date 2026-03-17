package trust

import (
	"math"
	"strconv"
	"testing"
)

func TestComputeTrustScore(t *testing.T) {
	tests := []struct {
		name        string
		memberships []Membership
		alliances   []Alliance
		want        float64
	}{
		{
			name:        "no memberships returns zero",
			memberships: []Membership{},
			alliances:   []Alliance{},
			want:        0.0,
		},
		{
			name:        "nil memberships returns zero",
			memberships: nil,
			alliances:   nil,
			want:        0.0,
		},
		{
			name: "single member no alliances",
			memberships: []Membership{
				{SceneID: "s1", UserDID: "did:user1", Role: "member", TrustWeight: 0.8},
			},
			alliances: []Alliance{},
			// avg_alliance = 1.0 (default), avg_membership = 0.8 * 0.5 = 0.4
			// score = 1.0 * 0.4 = 0.4
			want: 0.4,
		},
		{
			name: "single member with curator role",
			memberships: []Membership{
				{SceneID: "s1", UserDID: "did:user1", Role: "curator", TrustWeight: 0.8},
			},
			alliances: []Alliance{},
			// avg_alliance = 1.0, avg_membership = 0.8 * 0.8 = 0.64
			// score = 1.0 * 0.64 = 0.64
			want: 0.64,
		},
		{
			name: "single member with owner role",
			memberships: []Membership{
				{SceneID: "s1", UserDID: "did:user1", Role: "owner", TrustWeight: 0.5},
			},
			alliances: []Alliance{},
			// avg_alliance = 1.0, avg_membership = 0.5 * 1.0 = 0.5
			// score = 1.0 * 0.5 = 0.5
			want: 0.5,
		},
		{
			name: "single member with guest role",
			memberships: []Membership{
				{SceneID: "s1", UserDID: "did:user1", Role: "guest", TrustWeight: 1.0},
			},
			alliances: []Alliance{},
			// avg_alliance = 1.0, avg_membership = 1.0 * 0.3 = 0.3
			// score = 1.0 * 0.3 = 0.3
			want: 0.3,
		},
		{
			name: "multiple members different roles",
			memberships: []Membership{
				{SceneID: "s1", UserDID: "did:user1", Role: "member", TrustWeight: 0.6},  // 0.6 * 0.5 = 0.3
				{SceneID: "s1", UserDID: "did:user2", Role: "curator", TrustWeight: 0.8}, // 0.8 * 0.8 = 0.64
				{SceneID: "s1", UserDID: "did:user3", Role: "owner", TrustWeight: 1.0},   // 1.0 * 1.0 = 1.0
			},
			alliances: []Alliance{},
			// avg_membership = (0.3 + 0.64 + 1.0) / 3 = 1.94 / 3 = 0.6466...
			// score = 1.0 * 0.6466... = 0.6466...
			want: 1.94 / 3.0,
		},
		{
			name: "single member single alliance",
			memberships: []Membership{
				{SceneID: "s1", UserDID: "did:user1", Role: "member", TrustWeight: 1.0},
			},
			alliances: []Alliance{
				{FromSceneID: "s1", ToSceneID: "s2", Weight: 0.5},
			},
			// avg_alliance = 0.5, avg_membership = 1.0 * 0.5 = 0.5
			// score = 0.5 * 0.5 = 0.25
			want: 0.25,
		},
		{
			name: "multiple alliances",
			memberships: []Membership{
				{SceneID: "s1", UserDID: "did:user1", Role: "member", TrustWeight: 1.0},
			},
			alliances: []Alliance{
				{FromSceneID: "s1", ToSceneID: "s2", Weight: 0.4},
				{FromSceneID: "s1", ToSceneID: "s3", Weight: 0.6},
				{FromSceneID: "s1", ToSceneID: "s4", Weight: 0.8},
			},
			// avg_alliance = (0.4 + 0.6 + 0.8) / 3 = 0.6
			// avg_membership = 1.0 * 0.5 = 0.5
			// score = 0.6 * 0.5 = 0.3
			want: 0.3,
		},
		{
			name: "complex scenario",
			memberships: []Membership{
				{SceneID: "s1", UserDID: "did:user1", Role: "owner", TrustWeight: 0.9},   // 0.9 * 1.0 = 0.9
				{SceneID: "s1", UserDID: "did:user2", Role: "curator", TrustWeight: 0.7}, // 0.7 * 0.8 = 0.56
				{SceneID: "s1", UserDID: "did:user3", Role: "member", TrustWeight: 0.5},  // 0.5 * 0.5 = 0.25
				{SceneID: "s1", UserDID: "did:user4", Role: "guest", TrustWeight: 0.8},   // 0.8 * 0.3 = 0.24
			},
			alliances: []Alliance{
				{FromSceneID: "s1", ToSceneID: "s2", Weight: 0.9},
				{FromSceneID: "s1", ToSceneID: "s3", Weight: 0.7},
			},
			// avg_alliance = (0.9 + 0.7) / 2 = 0.8
			// avg_membership = (0.9 + 0.56 + 0.25 + 0.24) / 4 = 1.95 / 4 = 0.4875
			// score = 0.8 * 0.4875 = 0.39
			want: 0.8 * (1.95 / 4.0),
		},
		{
			name: "unknown role uses default multiplier",
			memberships: []Membership{
				{SceneID: "s1", UserDID: "did:user1", Role: "unknown_role", TrustWeight: 0.5},
			},
			alliances: []Alliance{},
			// avg_membership = 0.5 * 0.5 (default) = 0.25
			// score = 1.0 * 0.25 = 0.25
			want: 0.25,
		},
		{
			name: "zero trust weight member",
			memberships: []Membership{
				{SceneID: "s1", UserDID: "did:user1", Role: "member", TrustWeight: 0.0},
			},
			alliances: []Alliance{},
			want:      0.0,
		},
		{
			name: "zero weight alliance",
			memberships: []Membership{
				{SceneID: "s1", UserDID: "did:user1", Role: "member", TrustWeight: 1.0},
			},
			alliances: []Alliance{
				{FromSceneID: "s1", ToSceneID: "s2", Weight: 0.0},
			},
			want: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ComputeTrustScore(tt.memberships, tt.alliances)
			if math.Abs(got-tt.want) > 1e-9 {
				t.Errorf("ComputeTrustScore() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRoleMultipliers(t *testing.T) {
	// Verify role multipliers are correctly defined
	tests := []struct {
		role string
		want float64
	}{
		{role: "owner", want: 1.0},
		{role: "curator", want: 0.8},
		{role: "member", want: 0.5},
		{role: "guest", want: 0.3},
	}

	for _, tt := range tests {
		t.Run(tt.role, func(t *testing.T) {
			got := RoleMultiplier[tt.role]
			if got != tt.want {
				t.Errorf("RoleMultiplier[%q] = %v, want %v", tt.role, got, tt.want)
			}
		})
	}
}

func TestDirtyTracker(t *testing.T) {
	t.Run("initial state is empty", func(t *testing.T) {
		tracker := NewDirtyTracker()
		if tracker.DirtyCount() != 0 {
			t.Errorf("initial DirtyCount() = %d, want 0", tracker.DirtyCount())
		}
		if len(tracker.GetDirtyScenes()) != 0 {
			t.Errorf("initial GetDirtyScenes() len = %d, want 0", len(tracker.GetDirtyScenes()))
		}
	})

	t.Run("mark and check dirty", func(t *testing.T) {
		tracker := NewDirtyTracker()
		tracker.MarkDirty("scene-1")

		if !tracker.IsDirty("scene-1") {
			t.Error("expected scene-1 to be dirty")
		}
		if tracker.IsDirty("scene-2") {
			t.Error("expected scene-2 to not be dirty")
		}
		if tracker.DirtyCount() != 1 {
			t.Errorf("DirtyCount() = %d, want 1", tracker.DirtyCount())
		}
	})

	t.Run("clear dirty", func(t *testing.T) {
		tracker := NewDirtyTracker()
		tracker.MarkDirty("scene-1")
		tracker.ClearDirty("scene-1")

		if tracker.IsDirty("scene-1") {
			t.Error("expected scene-1 to not be dirty after clear")
		}
		if tracker.DirtyCount() != 0 {
			t.Errorf("DirtyCount() = %d, want 0", tracker.DirtyCount())
		}
	})

	t.Run("get dirty scenes", func(t *testing.T) {
		tracker := NewDirtyTracker()
		tracker.MarkDirty("scene-1")
		tracker.MarkDirty("scene-2")
		tracker.MarkDirty("scene-3")

		scenes := tracker.GetDirtyScenes()
		if len(scenes) != 3 {
			t.Errorf("GetDirtyScenes() len = %d, want 3", len(scenes))
		}

		// Check all scenes are present
		sceneMap := make(map[string]bool)
		for _, s := range scenes {
			sceneMap[s] = true
		}
		for _, expected := range []string{"scene-1", "scene-2", "scene-3"} {
			if !sceneMap[expected] {
				t.Errorf("expected %s in dirty scenes", expected)
			}
		}
	})

	t.Run("marking same scene twice is idempotent", func(t *testing.T) {
		tracker := NewDirtyTracker()
		tracker.MarkDirty("scene-1")
		tracker.MarkDirty("scene-1")

		if tracker.DirtyCount() != 1 {
			t.Errorf("DirtyCount() = %d, want 1 after double mark", tracker.DirtyCount())
		}
	})

	t.Run("clearing non-existent scene is safe", func(t *testing.T) {
		tracker := NewDirtyTracker()
		tracker.ClearDirty("nonexistent") // Should not panic

		if tracker.DirtyCount() != 0 {
			t.Errorf("DirtyCount() = %d, want 0", tracker.DirtyCount())
		}
	})
}

func TestDirtyTracker_Concurrency(t *testing.T) {
	tracker := NewDirtyTracker()
	done := make(chan bool)

	// Run concurrent operations
	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 100; j++ {
				sceneID := "scene-" + strconv.Itoa(id)
				tracker.MarkDirty(sceneID)
				tracker.IsDirty(sceneID)
				tracker.GetDirtyScenes()
				tracker.DirtyCount()
				if j%10 == 0 {
					tracker.ClearDirty(sceneID)
				}
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Just verify no panic occurred and state is consistent
	count := tracker.DirtyCount()
	scenes := tracker.GetDirtyScenes()
	if count != len(scenes) {
		t.Errorf("DirtyCount() = %d, but GetDirtyScenes() len = %d", count, len(scenes))
	}
}

func TestValidRole(t *testing.T) {
	tests := []struct {
		role string
		want bool
	}{
		{role: "owner", want: true},
		{role: "curator", want: true},
		{role: "member", want: true},
		{role: "guest", want: true},
		{role: "admin", want: false},
		{role: "moderator", want: false},
		{role: "", want: false},
		{role: "OWNER", want: false}, // case-sensitive
	}

	for _, tt := range tests {
		t.Run(tt.role, func(t *testing.T) {
			got := ValidRole(tt.role)
			if got != tt.want {
				t.Errorf("ValidRole(%q) = %v, want %v", tt.role, got, tt.want)
			}
		})
	}
}

func TestValidateTrustWeight(t *testing.T) {
	tests := []struct {
		name    string
		weight  float64
		wantErr bool
	}{
		{name: "zero is valid", weight: 0.0, wantErr: false},
		{name: "one is valid", weight: 1.0, wantErr: false},
		{name: "middle value is valid", weight: 0.5, wantErr: false},
		{name: "negative is invalid", weight: -0.1, wantErr: true},
		{name: "above one is invalid", weight: 1.1, wantErr: true},
		{name: "large negative is invalid", weight: -100.0, wantErr: true},
		{name: "large positive is invalid", weight: 100.0, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTrustWeight(tt.weight)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateTrustWeight(%v) error = %v, wantErr %v", tt.weight, err, tt.wantErr)
			}
			if tt.wantErr && err != ErrInvalidTrustWeight {
				t.Errorf("ValidateTrustWeight(%v) error = %v, want %v", tt.weight, err, ErrInvalidTrustWeight)
			}
		})
	}
}

func TestMembership_EffectiveWeight(t *testing.T) {
	tests := []struct {
		name       string
		membership Membership
		want       float64
	}{
		{
			name: "owner with full trust",
			membership: Membership{
				Role:        "owner",
				TrustWeight: 1.0,
			},
			want: 1.0, // 1.0 * 1.0
		},
		{
			name: "curator with 0.8 trust",
			membership: Membership{
				Role:        "curator",
				TrustWeight: 0.8,
			},
			want: 0.64, // 0.8 * 0.8
		},
		{
			name: "member with 0.5 trust",
			membership: Membership{
				Role:        "member",
				TrustWeight: 0.5,
			},
			want: 0.25, // 0.5 * 0.5
		},
		{
			name: "guest with full trust",
			membership: Membership{
				Role:        "guest",
				TrustWeight: 1.0,
			},
			want: 0.3, // 1.0 * 0.3
		},
		{
			name: "unknown role uses default multiplier",
			membership: Membership{
				Role:        "unknown",
				TrustWeight: 1.0,
			},
			want: 0.5, // 1.0 * 0.5 (default)
		},
		{
			name: "zero trust weight",
			membership: Membership{
				Role:        "owner",
				TrustWeight: 0.0,
			},
			want: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.membership.EffectiveWeight()
			if math.Abs(got-tt.want) > 1e-9 {
				t.Errorf("Membership.EffectiveWeight() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMembership_Validate(t *testing.T) {
	tests := []struct {
		name       string
		membership Membership
		wantErr    error
	}{
		{
			name: "valid owner membership",
			membership: Membership{
				Role:        "owner",
				TrustWeight: 1.0,
			},
			wantErr: nil,
		},
		{
			name: "valid guest membership",
			membership: Membership{
				Role:        "guest",
				TrustWeight: 0.3,
			},
			wantErr: nil,
		},
		{
			name: "invalid role",
			membership: Membership{
				Role:        "admin",
				TrustWeight: 0.5,
			},
			wantErr: ErrInvalidRole,
		},
		{
			name: "invalid trust weight too high",
			membership: Membership{
				Role:        "owner",
				TrustWeight: 1.5,
			},
			wantErr: ErrInvalidTrustWeight,
		},
		{
			name: "invalid trust weight negative",
			membership: Membership{
				Role:        "member",
				TrustWeight: -0.5,
			},
			wantErr: ErrInvalidTrustWeight,
		},
		{
			name: "both invalid, returns role error first",
			membership: Membership{
				Role:        "invalid",
				TrustWeight: 2.0,
			},
			wantErr: ErrInvalidRole,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.membership.Validate()
			if err != tt.wantErr {
				t.Errorf("Membership.Validate() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}
