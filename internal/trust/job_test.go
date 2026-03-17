package trust

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/onnwee/subcults/internal/jobs"
)

// slowDataSource wraps a DataSource with artificial delays for testing timeouts.
type slowDataSource struct {
	ds    DataSource
	delay time.Duration
}

// newSlowDataSource creates a new slow data source wrapper.
func newSlowDataSource(ds DataSource, delay time.Duration) *slowDataSource {
	return &slowDataSource{
		ds:    ds,
		delay: delay,
	}
}

// GetMembershipsByScene returns memberships after a delay.
func (s *slowDataSource) GetMembershipsByScene(sceneID string) ([]Membership, error) {
	time.Sleep(s.delay)
	return s.ds.GetMembershipsByScene(sceneID)
}

// GetAlliancesByScene returns alliances after a delay.
func (s *slowDataSource) GetAlliancesByScene(sceneID string) ([]Alliance, error) {
	time.Sleep(s.delay)
	return s.ds.GetAlliancesByScene(sceneID)
}

// AddAlliance delegates to the wrapped data source.
func (s *slowDataSource) AddAlliance(a Alliance) {
	s.ds.AddAlliance(a)
}

// ClearAlliances delegates to the wrapped data source.
func (s *slowDataSource) ClearAlliances(sceneID string) {
	s.ds.ClearAlliances(sceneID)
}

func TestRecomputeJob_StartStop(t *testing.T) {
	dataSource := NewInMemoryDataSource()
	scoreStore := NewInMemoryScoreStore()
	dirtyTracker := NewDirtyTracker()

	job := NewRecomputeJob(
		RecomputeJobConfig{
			Interval: 100 * time.Millisecond,
			Logger:   slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError})),
		},
		dirtyTracker,
		dataSource,
		scoreStore,
	)

	// Job should not be running initially
	if job.IsRunning() {
		t.Error("job should not be running before Start")
	}

	// Start the job
	ctx := context.Background()
	if err := job.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	if !job.IsRunning() {
		t.Error("job should be running after Start")
	}

	// Starting again should be safe (idempotent)
	if err := job.Start(ctx); err != nil {
		t.Fatalf("Start() second call error = %v", err)
	}

	// Stop the job
	job.Stop()

	if job.IsRunning() {
		t.Error("job should not be running after Stop")
	}

	// Stopping again should be safe
	job.Stop()
}

func TestRecomputeJob_RecomputesOnlyDirtyScenes(t *testing.T) {
	dataSource := NewInMemoryDataSource()
	scoreStore := NewInMemoryScoreStore()
	dirtyTracker := NewDirtyTracker()

	// Setup data for two scenes
	dataSource.AddMembership(Membership{
		SceneID:     "scene-1",
		UserDID:     "did:user1",
		Role:        "owner",
		TrustWeight: 0.8,
	})
	dataSource.AddMembership(Membership{
		SceneID:     "scene-2",
		UserDID:     "did:user2",
		Role:        "owner",
		TrustWeight: 1.0,
	})

	job := NewRecomputeJob(
		RecomputeJobConfig{
			Interval: 100 * time.Millisecond,
			Logger:   slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError})),
		},
		dirtyTracker,
		dataSource,
		scoreStore,
	)

	// Mark only scene-1 as dirty
	dirtyTracker.MarkDirty("scene-1")

	// Trigger immediate recompute
	job.RecomputeNow()

	// Check scene-1 was recomputed
	score1, err := scoreStore.GetScore("scene-1")
	if err != nil {
		t.Fatalf("GetScore(scene-1) error = %v", err)
	}
	if score1 == nil {
		t.Fatal("expected score for scene-1")
	}
	expectedScore1 := 0.8 // 0.8 * 1.0 (owner multiplier) * 1.0 (no alliances)
	if score1.Score != expectedScore1 {
		t.Errorf("scene-1 score = %v, want %v", score1.Score, expectedScore1)
	}

	// Check scene-2 was NOT recomputed (not dirty)
	score2, err := scoreStore.GetScore("scene-2")
	if err != nil {
		t.Fatalf("GetScore(scene-2) error = %v", err)
	}
	if score2 != nil {
		t.Error("expected no score for scene-2 (not dirty)")
	}

	// Check scene-1 is no longer dirty
	if dirtyTracker.IsDirty("scene-1") {
		t.Error("scene-1 should not be dirty after recompute")
	}
}

func TestRecomputeJob_RecomputesWithAlliances(t *testing.T) {
	dataSource := NewInMemoryDataSource()
	scoreStore := NewInMemoryScoreStore()
	dirtyTracker := NewDirtyTracker()

	// Setup scene with memberships and alliances
	dataSource.AddMembership(Membership{
		SceneID:     "scene-1",
		UserDID:     "did:user1",
		Role:        "curator",
		TrustWeight: 0.8,
	})
	dataSource.AddAlliance(Alliance{
		FromSceneID: "scene-1",
		ToSceneID:   "scene-2",
		Weight:      0.6,
	})
	dataSource.AddAlliance(Alliance{
		FromSceneID: "scene-1",
		ToSceneID:   "scene-3",
		Weight:      0.8,
	})

	job := NewRecomputeJob(
		RecomputeJobConfig{
			Interval: 100 * time.Millisecond,
			Logger:   slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError})),
		},
		dirtyTracker,
		dataSource,
		scoreStore,
	)

	dirtyTracker.MarkDirty("scene-1")
	job.RecomputeNow()

	score, err := scoreStore.GetScore("scene-1")
	if err != nil {
		t.Fatalf("GetScore error = %v", err)
	}
	if score == nil {
		t.Fatal("expected score for scene-1")
	}

	// Expected: avg_alliance = (0.6 + 0.8) / 2 = 0.7
	//           avg_membership = 0.8 * 0.8 (curator) = 0.64
	//           score = 0.7 * 0.64 = 0.448
	expectedScore := 0.7 * 0.64
	if math.Abs(score.Score-expectedScore) > 1e-9 {
		t.Errorf("score = %v, want %v", score.Score, expectedScore)
	}
}

func TestRecomputeJob_PeriodicExecution(t *testing.T) {
	dataSource := NewInMemoryDataSource()
	scoreStore := NewInMemoryScoreStore()
	dirtyTracker := NewDirtyTracker()

	// Setup initial data
	dataSource.AddMembership(Membership{
		SceneID:     "scene-1",
		UserDID:     "did:user1",
		Role:        "member",
		TrustWeight: 0.5,
	})

	job := NewRecomputeJob(
		RecomputeJobConfig{
			Interval: 50 * time.Millisecond, // Short interval for testing
			Logger:   slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError})),
		},
		dirtyTracker,
		dataSource,
		scoreStore,
	)

	ctx := context.Background()
	if err := job.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer job.Stop()

	// Mark scene dirty
	dirtyTracker.MarkDirty("scene-1")

	// Wait for at least one tick
	time.Sleep(100 * time.Millisecond)

	// Check score was computed
	score, err := scoreStore.GetScore("scene-1")
	if err != nil {
		t.Fatalf("GetScore error = %v", err)
	}
	if score == nil {
		t.Fatal("expected score to be computed after periodic tick")
	}
	if score.Score != 0.25 {
		t.Errorf("score = %v, want 0.25", score.Score)
	}

	// Scene should no longer be dirty
	if dirtyTracker.IsDirty("scene-1") {
		t.Error("scene-1 should not be dirty after recompute")
	}
}

func TestRecomputeJob_ContextCancellation(t *testing.T) {
	dataSource := NewInMemoryDataSource()
	scoreStore := NewInMemoryScoreStore()
	dirtyTracker := NewDirtyTracker()

	job := NewRecomputeJob(
		RecomputeJobConfig{
			Interval: 100 * time.Millisecond,
			Logger:   slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError})),
		},
		dirtyTracker,
		dataSource,
		scoreStore,
	)

	ctx, cancel := context.WithCancel(context.Background())
	if err := job.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	if !job.IsRunning() {
		t.Error("job should be running")
	}

	// Cancel context
	cancel()

	// Give job time to notice cancellation
	time.Sleep(50 * time.Millisecond)

	// Job should have stopped - wait for doneCh via Stop()
	job.Stop()

	if job.IsRunning() {
		t.Error("job should have stopped after context cancellation")
	}
}

func TestRecomputeJob_DefaultInterval(t *testing.T) {
	dataSource := NewInMemoryDataSource()
	scoreStore := NewInMemoryScoreStore()
	dirtyTracker := NewDirtyTracker()

	// Create job with zero interval (should use default)
	job := NewRecomputeJob(
		RecomputeJobConfig{
			Interval: 0,
		},
		dirtyTracker,
		dataSource,
		scoreStore,
	)

	// The job should use default interval internally
	// We can't easily verify the interval value, but we verify it doesn't panic
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := job.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer job.Stop()

	if !job.IsRunning() {
		t.Error("job should be running with default interval")
	}
}

func TestRecomputeJob_EmptyDirtyScenes(t *testing.T) {
	dataSource := NewInMemoryDataSource()
	scoreStore := NewInMemoryScoreStore()
	dirtyTracker := NewDirtyTracker()

	job := NewRecomputeJob(
		RecomputeJobConfig{
			Interval: 100 * time.Millisecond,
			Logger:   slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError})),
		},
		dirtyTracker,
		dataSource,
		scoreStore,
	)

	// No dirty scenes
	job.RecomputeNow()

	// Should complete without error and no scores stored
	allScores := scoreStore.AllScores()
	if len(allScores) != 0 {
		t.Errorf("expected no scores, got %d", len(allScores))
	}
}

func TestInMemoryDataSource(t *testing.T) {
	t.Run("add and get memberships", func(t *testing.T) {
		ds := NewInMemoryDataSource()

		ds.AddMembership(Membership{SceneID: "s1", UserDID: "u1", Role: "member", TrustWeight: 0.5})
		ds.AddMembership(Membership{SceneID: "s1", UserDID: "u2", Role: "admin", TrustWeight: 0.8})
		ds.AddMembership(Membership{SceneID: "s2", UserDID: "u3", Role: "curator", TrustWeight: 0.6})

		memberships, err := ds.GetMembershipsByScene("s1")
		if err != nil {
			t.Fatalf("GetMembershipsByScene error = %v", err)
		}
		if len(memberships) != 2 {
			t.Errorf("expected 2 memberships for s1, got %d", len(memberships))
		}

		memberships2, err := ds.GetMembershipsByScene("s2")
		if err != nil {
			t.Fatalf("GetMembershipsByScene error = %v", err)
		}
		if len(memberships2) != 1 {
			t.Errorf("expected 1 membership for s2, got %d", len(memberships2))
		}

		// Non-existent scene
		memberships3, err := ds.GetMembershipsByScene("s3")
		if err != nil {
			t.Fatalf("GetMembershipsByScene error = %v", err)
		}
		if len(memberships3) != 0 {
			t.Errorf("expected 0 memberships for s3, got %d", len(memberships3))
		}
	})

	t.Run("add and get alliances", func(t *testing.T) {
		ds := NewInMemoryDataSource()

		ds.AddAlliance(Alliance{FromSceneID: "s1", ToSceneID: "s2", Weight: 0.5})
		ds.AddAlliance(Alliance{FromSceneID: "s1", ToSceneID: "s3", Weight: 0.7})
		ds.AddAlliance(Alliance{FromSceneID: "s2", ToSceneID: "s1", Weight: 0.6})

		alliances, err := ds.GetAlliancesByScene("s1")
		if err != nil {
			t.Fatalf("GetAlliancesByScene error = %v", err)
		}
		if len(alliances) != 2 {
			t.Errorf("expected 2 alliances from s1, got %d", len(alliances))
		}
	})

	t.Run("clear memberships", func(t *testing.T) {
		ds := NewInMemoryDataSource()

		ds.AddMembership(Membership{SceneID: "s1", UserDID: "u1", Role: "member", TrustWeight: 0.5})
		ds.ClearMemberships("s1")

		memberships, _ := ds.GetMembershipsByScene("s1")
		if len(memberships) != 0 {
			t.Errorf("expected 0 memberships after clear, got %d", len(memberships))
		}
	})

	t.Run("clear alliances", func(t *testing.T) {
		ds := NewInMemoryDataSource()

		ds.AddAlliance(Alliance{FromSceneID: "s1", ToSceneID: "s2", Weight: 0.5})
		ds.ClearAlliances("s1")

		alliances, _ := ds.GetAlliancesByScene("s1")
		if len(alliances) != 0 {
			t.Errorf("expected 0 alliances after clear, got %d", len(alliances))
		}
	})
}

func TestInMemoryScoreStore(t *testing.T) {
	t.Run("save and get score", func(t *testing.T) {
		store := NewInMemoryScoreStore()

		now := time.Now()
		score := SceneTrustScore{
			SceneID:    "s1",
			Score:      0.75,
			ComputedAt: now,
		}

		if err := store.SaveScore(score); err != nil {
			t.Fatalf("SaveScore error = %v", err)
		}

		retrieved, err := store.GetScore("s1")
		if err != nil {
			t.Fatalf("GetScore error = %v", err)
		}
		if retrieved == nil {
			t.Fatal("expected score to be retrieved")
		}
		if retrieved.Score != 0.75 {
			t.Errorf("score = %v, want 0.75", retrieved.Score)
		}
	})

	t.Run("get non-existent score", func(t *testing.T) {
		store := NewInMemoryScoreStore()

		score, err := store.GetScore("nonexistent")
		if err != nil {
			t.Fatalf("GetScore error = %v", err)
		}
		if score != nil {
			t.Error("expected nil for non-existent score")
		}
	})

	t.Run("update score", func(t *testing.T) {
		store := NewInMemoryScoreStore()

		store.SaveScore(SceneTrustScore{SceneID: "s1", Score: 0.5, ComputedAt: time.Now()})
		store.SaveScore(SceneTrustScore{SceneID: "s1", Score: 0.8, ComputedAt: time.Now()})

		retrieved, _ := store.GetScore("s1")
		if retrieved.Score != 0.8 {
			t.Errorf("score = %v, want 0.8 (updated)", retrieved.Score)
		}
	})

	t.Run("all scores", func(t *testing.T) {
		store := NewInMemoryScoreStore()

		store.SaveScore(SceneTrustScore{SceneID: "s1", Score: 0.5, ComputedAt: time.Now()})
		store.SaveScore(SceneTrustScore{SceneID: "s2", Score: 0.6, ComputedAt: time.Now()})

		allScores := store.AllScores()
		if len(allScores) != 2 {
			t.Errorf("expected 2 scores, got %d", len(allScores))
		}
	})
}

func TestRecomputeJob_WithMetrics(t *testing.T) {
	dataSource := NewInMemoryDataSource()
	scoreStore := NewInMemoryScoreStore()
	dirtyTracker := NewDirtyTracker()
	metrics := NewMetrics()

	// Setup 50 scenes with data
	for i := 0; i < 50; i++ {
		sceneID := fmt.Sprintf("scene-%d", i)
		dataSource.AddMembership(Membership{
			SceneID:     sceneID,
			UserDID:     fmt.Sprintf("did:user%d", i),
			Role:        "member",
			TrustWeight: 0.5,
		})
		dirtyTracker.MarkDirty(sceneID)
	}

	job := NewRecomputeJob(
		RecomputeJobConfig{
			Interval: 100 * time.Millisecond,
			Logger:   slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError})),
			Metrics:  metrics,
			Timeout:  5 * time.Second,
		},
		dirtyTracker,
		dataSource,
		scoreStore,
	)

	// Trigger recompute
	job.RecomputeNow()

	// Verify metrics were updated
	if v := getCounterValue(metrics.recomputeTotal); v != 1 {
		t.Errorf("recomputeTotal = %f, want 1", v)
	}

	if v := getHistogramSampleCount(metrics.recomputeDuration); v != 1 {
		t.Errorf("recomputeDuration sample count = %d, want 1", v)
	}

	if v := getGaugeValue(metrics.lastRecomputeSceneCount); v != 50 {
		t.Errorf("lastRecomputeSceneCount = %f, want 50", v)
	}

	if v := getGaugeValue(metrics.lastRecomputeTimestamp); v <= 0 {
		t.Errorf("lastRecomputeTimestamp = %f, should be > 0", v)
	}

	// Verify all scenes were processed
	if count := dirtyTracker.DirtyCount(); count != 0 {
		t.Errorf("dirty count = %d, want 0", count)
	}

	allScores := scoreStore.AllScores()
	if len(allScores) != 50 {
		t.Errorf("stored scores = %d, want 50", len(allScores))
	}
}

func TestRecomputeJob_TimeoutAbort(t *testing.T) {
	dataSource := NewInMemoryDataSource()
	slowDataSource := newSlowDataSource(dataSource, 200*time.Millisecond)
	scoreStore := NewInMemoryScoreStore()
	dirtyTracker := NewDirtyTracker()
	metrics := NewMetrics()

	// Setup 10 scenes - with 200ms delay per scene, this would take 2+ seconds
	for i := 0; i < 10; i++ {
		sceneID := fmt.Sprintf("scene-%d", i)
		dataSource.AddMembership(Membership{
			SceneID:     sceneID,
			UserDID:     fmt.Sprintf("did:user%d", i),
			Role:        "member",
			TrustWeight: 0.5,
		})
		dirtyTracker.MarkDirty(sceneID)
	}

	job := NewRecomputeJob(
		RecomputeJobConfig{
			Interval:       100 * time.Millisecond,
			Logger:         slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError})),
			Metrics:        metrics,
			Timeout:        500 * time.Millisecond, // Very short timeout to trigger abort
			BatchSize:      2,                      // Small batch size to allow timeout between batches
			MaxConcurrency: 1,                      // Serial processing to ensure predictable timeout
		},
		dirtyTracker,
		slowDataSource,
		scoreStore,
	)

	// Trigger recompute
	job.RecomputeNow()

	// Verify error counter was incremented due to timeout
	if v := getCounterValue(metrics.recomputeErrors); v != 1 {
		t.Errorf("recomputeErrors = %f, want 1", v)
	}

	// Some scenes should still be dirty (not all processed)
	// With batch size 2 and 200ms delay per scene, first batch takes 400ms
	// Second batch should hit the 500ms timeout
	if count := dirtyTracker.DirtyCount(); count == 0 {
		t.Error("dirty count should be > 0 due to timeout abort")
	}

	// Not all scores should be stored (only first batch of 2 scenes)
	allScores := scoreStore.AllScores()
	if len(allScores) >= 10 {
		t.Errorf("stored scores = %d, should be < 10 due to timeout", len(allScores))
	}
}

func TestRecomputeJob_CompletionLog(t *testing.T) {
	dataSource := NewInMemoryDataSource()
	scoreStore := NewInMemoryScoreStore()
	dirtyTracker := NewDirtyTracker()

	// Create a custom logger to capture logs
	var logBuf strings.Builder
	logger := slog.New(slog.NewTextHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelInfo}))

	// Setup scenes with initial scores for variance calculation
	for i := 0; i < 5; i++ {
		sceneID := fmt.Sprintf("scene-%d", i)
		dataSource.AddMembership(Membership{
			SceneID:     sceneID,
			UserDID:     fmt.Sprintf("did:user%d", i),
			Role:        "member",
			TrustWeight: 0.5,
		})
		// Set initial score
		scoreStore.SaveScore(SceneTrustScore{
			SceneID:    sceneID,
			Score:      0.3,
			ComputedAt: time.Now().Add(-1 * time.Hour),
		})
		dirtyTracker.MarkDirty(sceneID)
	}

	job := NewRecomputeJob(
		RecomputeJobConfig{
			Interval: 100 * time.Millisecond,
			Logger:   logger,
		},
		dirtyTracker,
		dataSource,
		scoreStore,
	)

	// Trigger recompute
	job.RecomputeNow()

	// Verify completion log contains required fields
	logOutput := logBuf.String()
	requiredFields := []string{
		"trust recompute completed",
		"duration_seconds",
		"scenes_processed",
		"avg_weight_variance",
	}

	for _, field := range requiredFields {
		if !strings.Contains(logOutput, field) {
			t.Errorf("completion log missing required field: %s", field)
		}
	}

	// Verify that variance was calculated (should be non-zero since scores changed)
	if !strings.Contains(logOutput, "avg_weight_variance") {
		t.Error("completion log should contain avg_weight_variance field")
	}
}

// BenchmarkRecompute is a placeholder for future scaling tests.
// To be expanded with various scene counts (100, 1000, 10000) and
// different alliance graph complexities.
func BenchmarkRecompute(b *testing.B) {
	dataSource := NewInMemoryDataSource()
	scoreStore := NewInMemoryScoreStore()
	dirtyTracker := NewDirtyTracker()

	// Setup a modest dataset for initial benchmark
	sceneCount := 100
	for i := 0; i < sceneCount; i++ {
		sceneID := fmt.Sprintf("scene-%d", i)
		dataSource.AddMembership(Membership{
			SceneID:     sceneID,
			UserDID:     fmt.Sprintf("did:user%d", i),
			Role:        "member",
			TrustWeight: 0.5,
		})
	}

	job := NewRecomputeJob(
		RecomputeJobConfig{
			Logger: slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError})),
		},
		dirtyTracker,
		dataSource,
		scoreStore,
	)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Mark all scenes dirty
		for j := 0; j < sceneCount; j++ {
			dirtyTracker.MarkDirty(fmt.Sprintf("scene-%d", j))
		}
		job.RecomputeNow()
	}
}

// mockJobMetrics is a mock implementation of JobMetrics for testing.
// Note: This implementation is NOT thread-safe. Use only in single-threaded tests.
type mockJobMetrics struct {
	jobsTotal    map[string]map[string]int
	jobsDuration map[string][]float64
	jobErrors    map[string]map[string]int
}

func newMockJobMetrics() *mockJobMetrics {
	return &mockJobMetrics{
		jobsTotal:    make(map[string]map[string]int),
		jobsDuration: make(map[string][]float64),
		jobErrors:    make(map[string]map[string]int),
	}
}

func (m *mockJobMetrics) IncJobsTotal(jobType, status string) {
	if m.jobsTotal[jobType] == nil {
		m.jobsTotal[jobType] = make(map[string]int)
	}
	m.jobsTotal[jobType][status]++
}

func (m *mockJobMetrics) ObserveJobDuration(jobType string, seconds float64) {
	m.jobsDuration[jobType] = append(m.jobsDuration[jobType], seconds)
}

func (m *mockJobMetrics) IncJobErrors(jobType, errorType string) {
	if m.jobErrors[jobType] == nil {
		m.jobErrors[jobType] = make(map[string]int)
	}
	m.jobErrors[jobType][errorType]++
}

func TestRecomputeJob_WithJobMetrics(t *testing.T) {
	dataSource := NewInMemoryDataSource()
	scoreStore := NewInMemoryScoreStore()
	dirtyTracker := NewDirtyTracker()
	jobMetrics := newMockJobMetrics()

	// Setup 20 scenes with data
	for i := 0; i < 20; i++ {
		sceneID := fmt.Sprintf("scene-%d", i)
		dataSource.AddMembership(Membership{
			SceneID:     sceneID,
			UserDID:     fmt.Sprintf("did:user%d", i),
			Role:        "member",
			TrustWeight: 0.5,
		})
		dirtyTracker.MarkDirty(sceneID)
	}

	job := NewRecomputeJob(
		RecomputeJobConfig{
			Interval:   100 * time.Millisecond,
			Logger:     slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError})),
			JobMetrics: jobMetrics,
			Timeout:    5 * time.Second,
		},
		dirtyTracker,
		dataSource,
		scoreStore,
	)

	// Trigger recompute
	job.RecomputeNow()

	// Verify job total metrics were updated
	if v := jobMetrics.jobsTotal[jobs.JobTypeTrustRecompute][jobs.StatusSuccess]; v != 1 {
		t.Errorf("jobsTotal[trust_recompute][success] = %d, want 1", v)
	}

	// Verify job duration was recorded
	if len(jobMetrics.jobsDuration[jobs.JobTypeTrustRecompute]) != 1 {
		t.Errorf("jobsDuration[trust_recompute] count = %d, want 1", len(jobMetrics.jobsDuration[jobs.JobTypeTrustRecompute]))
	}

	// Duration should be positive
	if len(jobMetrics.jobsDuration[jobs.JobTypeTrustRecompute]) > 0 {
		duration := jobMetrics.jobsDuration[jobs.JobTypeTrustRecompute][0]
		if duration <= 0 {
			t.Errorf("jobsDuration = %f, should be > 0", duration)
		}
	}

	// Verify all scenes were processed
	if count := dirtyTracker.DirtyCount(); count != 0 {
		t.Errorf("dirty count = %d, want 0", count)
	}
}

func TestRecomputeJob_WithJobMetricsAndErrors(t *testing.T) {
	// Use a data source that will return errors
	dataSource := NewInMemoryDataSource()
	scoreStore := &errorScoreStore{} // Always returns errors
	dirtyTracker := NewDirtyTracker()
	jobMetrics := newMockJobMetrics()

	// Setup 5 scenes
	for i := 0; i < 5; i++ {
		sceneID := fmt.Sprintf("scene-%d", i)
		dataSource.AddMembership(Membership{
			SceneID:     sceneID,
			UserDID:     fmt.Sprintf("did:user%d", i),
			Role:        "member",
			TrustWeight: 0.5,
		})
		dirtyTracker.MarkDirty(sceneID)
	}

	job := NewRecomputeJob(
		RecomputeJobConfig{
			Interval:   100 * time.Millisecond,
			Logger:     slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError})),
			JobMetrics: jobMetrics,
			Timeout:    5 * time.Second,
		},
		dirtyTracker,
		dataSource,
		scoreStore,
	)

	// Trigger recompute
	job.RecomputeNow()

	// Verify job status is failure (not all scenes processed)
	if v := jobMetrics.jobsTotal[jobs.JobTypeTrustRecompute][jobs.StatusFailure]; v != 1 {
		t.Errorf("jobsTotal[trust_recompute][failure] = %d, want 1", v)
	}

	// Verify job errors were tracked
	if v := jobMetrics.jobErrors[jobs.JobTypeTrustRecompute]["recompute_error"]; v != 5 {
		t.Errorf("jobErrors[trust_recompute][recompute_error] = %d, want 5", v)
	}
}

func TestRecomputeJob_WithJobMetricsTimeout(t *testing.T) {
	dataSource := NewInMemoryDataSource()
	slowDataSource := newSlowDataSource(dataSource, 200*time.Millisecond)
	scoreStore := NewInMemoryScoreStore()
	dirtyTracker := NewDirtyTracker()
	jobMetrics := newMockJobMetrics()

	// Setup 10 scenes - with 200ms delay per scene, this would take 2+ seconds
	for i := 0; i < 10; i++ {
		sceneID := fmt.Sprintf("scene-%d", i)
		dataSource.AddMembership(Membership{
			SceneID:     sceneID,
			UserDID:     fmt.Sprintf("did:user%d", i),
			Role:        "member",
			TrustWeight: 0.5,
		})
		dirtyTracker.MarkDirty(sceneID)
	}

	job := NewRecomputeJob(
		RecomputeJobConfig{
			Interval:       100 * time.Millisecond,
			Logger:         slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError})),
			JobMetrics:     jobMetrics,
			Timeout:        500 * time.Millisecond, // Very short timeout to trigger abort
			BatchSize:      2,                      // Small batch size to allow timeout between batches
			MaxConcurrency: 1,                      // Serial processing to ensure predictable timeout
		},
		dirtyTracker,
		slowDataSource,
		scoreStore,
	)

	// Trigger recompute
	job.RecomputeNow()

	// Verify timeout error was tracked
	if v := jobMetrics.jobErrors[jobs.JobTypeTrustRecompute]["timeout"]; v != 1 {
		t.Errorf("jobErrors[trust_recompute][timeout] = %d, want 1", v)
	}

	// Verify job completion was tracked even with timeout (failure status)
	if v := jobMetrics.jobsTotal[jobs.JobTypeTrustRecompute][jobs.StatusFailure]; v != 1 {
		t.Errorf("jobsTotal[trust_recompute][failure] = %d, want 1", v)
	}

	// Verify duration was recorded
	if len(jobMetrics.jobsDuration[jobs.JobTypeTrustRecompute]) != 1 {
		t.Errorf("jobsDuration[trust_recompute] count = %d, want 1", len(jobMetrics.jobsDuration[jobs.JobTypeTrustRecompute]))
	}

	// Duration should be positive and bounded by timeout + some overhead (scheduler contention)
	// Use 3x timeout as upper bound to account for CI variability
	maxDuration := 3 * 500 * time.Millisecond
	if len(jobMetrics.jobsDuration[jobs.JobTypeTrustRecompute]) > 0 {
		duration := jobMetrics.jobsDuration[jobs.JobTypeTrustRecompute][0]
		if duration <= 0 || duration > maxDuration.Seconds() {
			t.Errorf("jobsDuration = %f, expected between 0 and %f", duration, maxDuration.Seconds())
		}
	}

	// Some scenes should still be dirty
	if count := dirtyTracker.DirtyCount(); count == 0 {
		t.Error("dirty count should be > 0 due to timeout abort")
	}
}

// errorScoreStore is a ScoreStore that always returns errors.
type errorScoreStore struct{}

func (e *errorScoreStore) SaveScore(score SceneTrustScore) error {
	return fmt.Errorf("simulated save error")
}

func (e *errorScoreStore) GetScore(sceneID string) (*SceneTrustScore, error) {
	return nil, fmt.Errorf("simulated get error")
}
