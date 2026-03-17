// Package trust provides trust score computation for scenes based on
// membership and alliance relationships.
package trust

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/onnwee/subcults/internal/jobs"
)

// DataSource provides membership and alliance data for trust score computation.
type DataSource interface {
	// GetMembershipsByScene returns all memberships for a scene.
	GetMembershipsByScene(sceneID string) ([]Membership, error)
	// GetAlliancesByScene returns all alliances where the scene is the source.
	GetAlliancesByScene(sceneID string) ([]Alliance, error)
	// AddAlliance adds an alliance to the data source.
	AddAlliance(a Alliance)
	// ClearAlliances removes all alliances for a scene.
	ClearAlliances(sceneID string)
}

// ScoreStore persists computed trust scores.
type ScoreStore interface {
	// SaveScore stores a computed trust score.
	SaveScore(score SceneTrustScore) error
	// GetScore retrieves a trust score by scene ID.
	GetScore(sceneID string) (*SceneTrustScore, error)
}

// RecomputeJobConfig configures the trust score recompute job.
type RecomputeJobConfig struct {
	// Interval is the duration between recompute cycles.
	Interval time.Duration
	// Logger for job activity.
	Logger *slog.Logger
	// Metrics for performance tracking.
	Metrics *Metrics
	// JobMetrics for centralized background job tracking.
	JobMetrics jobs.Reporter
	// Timeout for each recompute cycle.
	Timeout time.Duration
	// BatchSize is the maximum number of scenes to process in one batch.
	// Default: 500. Lower values reduce contention, higher values improve throughput.
	BatchSize int
	// MaxConcurrency limits parallel recompute operations within a batch.
	// Default: 5. Higher values increase throughput but may increase DB load.
	MaxConcurrency int
	// AdaptiveScheduling enables dynamic interval adjustment based on recompute cycle duration.
	// When enabled, interval increases when the recompute cycle duration exceeds LoadThresholdMs
	// and decreases when the cycle completes quickly. Default: false.
	AdaptiveScheduling bool
	// MinInterval is the minimum interval when adaptive scheduling is enabled.
	// Default: 10s.
	MinInterval time.Duration
	// MaxInterval is the maximum interval when adaptive scheduling is enabled.
	// Default: 5m.
	MaxInterval time.Duration
	// LoadThresholdMs is the average recompute cycle duration threshold (in ms) above
	// which the job considers the system under high load. Default: 100ms.
	LoadThresholdMs float64
}

// DefaultRecomputeInterval is the default interval between recompute cycles.
const DefaultRecomputeInterval = 30 * time.Second

// DefaultRecomputeTimeout is the default timeout for a single recompute cycle.
const DefaultRecomputeTimeout = 30 * time.Second

// DefaultBatchSize is the default number of scenes to process per batch.
const DefaultBatchSize = 500

// DefaultMaxConcurrency is the default max parallel recompute operations.
const DefaultMaxConcurrency = 5

// DefaultMinInterval is the default minimum interval for adaptive scheduling.
const DefaultMinInterval = 10 * time.Second

// DefaultMaxInterval is the default maximum interval for adaptive scheduling.
const DefaultMaxInterval = 5 * time.Minute

// DefaultLoadThresholdMs is the default high-load threshold in milliseconds.
const DefaultLoadThresholdMs = 100.0

// RecomputeJob periodically recalculates trust scores for dirty scenes.
type RecomputeJob struct {
	config       RecomputeJobConfig
	dirtyTracker *DirtyTracker
	dataSource   DataSource
	scoreStore   ScoreStore

	mu                sync.Mutex
	running           bool
	stopCh            chan struct{}
	doneCh            chan struct{}
	currentInterval   time.Duration
	recentDurations   []float64 // Ring buffer for adaptive scheduling
	recentDurationsIdx int
}

// NewRecomputeJob creates a new trust score recompute job.
func NewRecomputeJob(
	config RecomputeJobConfig,
	dirtyTracker *DirtyTracker,
	dataSource DataSource,
	scoreStore ScoreStore,
) *RecomputeJob {
	if config.Interval == 0 {
		config.Interval = DefaultRecomputeInterval
	}
	if config.Logger == nil {
		config.Logger = slog.Default()
	}
	if config.Timeout == 0 {
		config.Timeout = DefaultRecomputeTimeout
	}
	if config.BatchSize <= 0 {
		config.BatchSize = DefaultBatchSize
	}
	if config.MaxConcurrency <= 0 {
		config.MaxConcurrency = DefaultMaxConcurrency
	}
	if config.MinInterval == 0 {
		config.MinInterval = DefaultMinInterval
	}
	if config.MaxInterval == 0 {
		config.MaxInterval = DefaultMaxInterval
	}
	if config.LoadThresholdMs == 0 {
		config.LoadThresholdMs = DefaultLoadThresholdMs
	}

	return &RecomputeJob{
		config:          config,
		dirtyTracker:    dirtyTracker,
		dataSource:      dataSource,
		scoreStore:      scoreStore,
		currentInterval: config.Interval,
		recentDurations: make([]float64, 10), // Track last 10 runs for adaptive scheduling
	}
}

// Start begins the periodic recompute job.
// Returns immediately; the job runs in a background goroutine.
func (j *RecomputeJob) Start(ctx context.Context) error {
	j.mu.Lock()
	if j.running {
		j.mu.Unlock()
		return nil
	}
	j.running = true
	j.stopCh = make(chan struct{})
	j.doneCh = make(chan struct{})
	j.mu.Unlock()

	go j.run(ctx)
	return nil
}

// Stop signals the recompute job to stop and waits for it to finish.
func (j *RecomputeJob) Stop() {
	j.mu.Lock()
	if !j.running {
		j.mu.Unlock()
		return
	}
	stopCh := j.stopCh
	doneCh := j.doneCh
	j.mu.Unlock()

	close(stopCh)
	<-doneCh

	j.mu.Lock()
	j.running = false
	j.mu.Unlock()
}

// IsRunning returns whether the job is currently running.
func (j *RecomputeJob) IsRunning() bool {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.running
}

// run is the main loop for the recompute job.
func (j *RecomputeJob) run(ctx context.Context) {
	defer close(j.doneCh)

	// Use current interval (may be adjusted dynamically)
	j.mu.Lock()
	interval := j.currentInterval
	j.mu.Unlock()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			j.config.Logger.Info("trust recompute job stopping due to context cancellation")
			return
		case <-j.stopCh:
			j.config.Logger.Info("trust recompute job stopping due to stop signal")
			return
		case <-ticker.C:
			j.recomputeDirtyScenes(ctx)

			// Adjust interval if adaptive scheduling is enabled
			if j.config.AdaptiveScheduling {
				j.mu.Lock()
				newInterval := j.currentInterval
				j.mu.Unlock()

				if newInterval != interval {
					interval = newInterval
					ticker.Reset(interval)
					j.config.Logger.Debug("adjusted recompute interval",
						"new_interval", interval)
				}
			}
		}
	}
}

// recomputeDirtyScenes processes all dirty scenes and updates their trust scores.
// Uses batching and concurrency control for improved throughput.
func (j *RecomputeJob) recomputeDirtyScenes(parentCtx context.Context) {
	dirtyScenes := j.dirtyTracker.GetDirtyScenes()
	if len(dirtyScenes) == 0 {
		return
	}

	// Create context with timeout derived from parent
	ctx, cancel := context.WithTimeout(parentCtx, j.config.Timeout)
	defer cancel()

	// Track metrics
	startTime := time.Now()
	totalScenes := len(dirtyScenes)
	var successCount int
	var varianceSum float64
	var varianceCount int

	j.config.Logger.Info("recomputing trust scores",
		"dirty_count", totalScenes,
		"batch_size", j.config.BatchSize,
		"max_concurrency", j.config.MaxConcurrency)

	// Process in batches
	for batchStart := 0; batchStart < totalScenes; batchStart += j.config.BatchSize {
		batchEnd := batchStart + j.config.BatchSize
		if batchEnd > totalScenes {
			batchEnd = totalScenes
		}
		batch := dirtyScenes[batchStart:batchEnd]

		// Process batch with concurrency control
		batchStartTime := time.Now()
		batchSuccess := j.processBatch(ctx, batch, &varianceSum, &varianceCount)
		successCount += batchSuccess
		batchDuration := time.Since(batchStartTime).Seconds() * 1000 // Convert to ms

		// Track batch metrics
		if j.config.Metrics != nil {
			j.config.Metrics.ObserveBatchDuration(batchDuration)
			if batchDuration > 0 {
				entitiesPerSec := float64(len(batch)) / (batchDuration / 1000.0)
				j.config.Metrics.ObserveEntitiesPerSecond(entitiesPerSec)
			}
		}

		j.config.Logger.Debug("batch completed",
			"batch_idx", batchStart/j.config.BatchSize,
			"batch_size", len(batch),
			"batch_success", batchSuccess,
			"batch_duration_ms", batchDuration)

		// Check for timeout
		select {
		case <-ctx.Done():
			j.config.Logger.Error("trust recompute timeout exceeded",
				"processed", batchEnd,
				"total", totalScenes,
				"timeout", j.config.Timeout)
			if j.config.Metrics != nil {
				j.config.Metrics.IncRecomputeErrors()
			}
			if j.config.JobMetrics != nil {
				j.config.JobMetrics.IncJobErrors(jobs.JobTypeTrustRecompute, "timeout")
			}

			// Record job completion metrics even for timeout
			duration := time.Since(startTime).Seconds()
			if j.config.Metrics != nil {
				j.config.Metrics.ObserveRecomputeDuration(duration)
			}
			if j.config.JobMetrics != nil {
				j.config.JobMetrics.IncJobsTotal(jobs.JobTypeTrustRecompute, jobs.StatusFailure)
				j.config.JobMetrics.ObserveJobDuration(jobs.JobTypeTrustRecompute, duration)
			}
			return
		default:
		}
	}

	// Calculate metrics
	duration := time.Since(startTime).Seconds()
	avgVariance := 0.0
	if varianceCount > 0 {
		avgVariance = varianceSum / float64(varianceCount)
	}

	// Determine job status
	status := jobs.StatusSuccess
	if successCount < totalScenes {
		status = jobs.StatusFailure
	}

	// Update metrics
	if j.config.Metrics != nil {
		j.config.Metrics.IncRecomputeTotal()
		j.config.Metrics.ObserveRecomputeDuration(duration)
		j.config.Metrics.SetLastRecomputeTimestamp(float64(time.Now().Unix()))
		j.config.Metrics.SetLastRecomputeSceneCount(float64(successCount))
	}

	// Update centralized job metrics
	if j.config.JobMetrics != nil {
		j.config.JobMetrics.IncJobsTotal(jobs.JobTypeTrustRecompute, status)
		j.config.JobMetrics.ObserveJobDuration(jobs.JobTypeTrustRecompute, duration)
	}

	// Adjust interval based on duration (adaptive scheduling)
	if j.config.AdaptiveScheduling {
		j.adjustInterval(duration)
	}

	// Completion log with required fields
	j.config.Logger.Info("trust recompute completed",
		"duration_seconds", duration,
		"scenes_processed", successCount,
		"scenes_failed", totalScenes-successCount,
		"avg_weight_variance", avgVariance)
}

// processBatch processes a batch of scenes with controlled concurrency.
// Returns the number of successfully processed scenes.
func (j *RecomputeJob) processBatch(ctx context.Context, sceneIDs []string, varianceSum *float64, varianceCount *int) int {
	// Semaphore for concurrency control
	sem := make(chan struct{}, j.config.MaxConcurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var successCount int

	for _, sceneID := range sceneIDs {
		sceneID := sceneID // Capture for goroutine

		// Check context cancellation
		select {
		case <-ctx.Done():
			// Context cancelled, stop spawning new goroutines
			goto done
		default:
		}

		// Acquire semaphore (non-blocking with context check)
		select {
		case sem <- struct{}{}:
		case <-ctx.Done():
			goto done
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() { <-sem }() // Release semaphore

			// Get previous score for variance calculation
			var previousScore float64
			var hasPreviousScore bool
			if prevScoreObj, err := j.scoreStore.GetScore(sceneID); err == nil && prevScoreObj != nil {
				previousScore = prevScoreObj.Score
				hasPreviousScore = true
			}

			// Recompute scene
			newScore, err := j.recomputeSceneWithScore(sceneID)
			if err != nil {
				j.config.Logger.Error("failed to recompute trust score",
					"scene_id", sceneID,
					"error", err)
				if j.config.Metrics != nil {
					j.config.Metrics.IncRecomputeErrors()
				}
				if j.config.JobMetrics != nil {
					j.config.JobMetrics.IncJobErrors(jobs.JobTypeTrustRecompute, "recompute_error")
				}
				return
			}

			// Calculate variance for this scene
			if hasPreviousScore {
				variance := abs(newScore - previousScore)
				mu.Lock()
				*varianceSum += variance
				*varianceCount++
				mu.Unlock()
			}

			j.dirtyTracker.ClearDirty(sceneID)

			mu.Lock()
			successCount++
			mu.Unlock()
		}()
	}

done:
	// Wait for all goroutines to complete
	wg.Wait()

	return successCount
}

// adjustInterval adjusts the recompute interval based on observed duration.
// Implements adaptive scheduling per issue #172.
func (j *RecomputeJob) adjustInterval(duration float64) {
	j.mu.Lock()
	defer j.mu.Unlock()

	// Update recent durations ring buffer
	j.recentDurations[j.recentDurationsIdx] = duration
	j.recentDurationsIdx = (j.recentDurationsIdx + 1) % len(j.recentDurations)

	// Calculate average duration
	var sum float64
	count := 0
	for _, d := range j.recentDurations {
		if d > 0 {
			sum += d
			count++
		}
	}
	if count == 0 {
		return
	}
	avgDuration := sum / float64(count)

	// Convert load threshold to seconds
	loadThresholdSec := j.config.LoadThresholdMs / 1000.0

	// Adjust interval based on average duration
	newInterval := j.currentInterval
	if avgDuration > loadThresholdSec {
		// High load: increase interval (back off)
		newInterval = time.Duration(float64(j.currentInterval) * 1.5)
		if newInterval > j.config.MaxInterval {
			newInterval = j.config.MaxInterval
		}
	} else {
		// Low load: decrease interval (speed up)
		newInterval = time.Duration(float64(j.currentInterval) * 0.8)
		if newInterval < j.config.MinInterval {
			newInterval = j.config.MinInterval
		}
	}

	if newInterval != j.currentInterval {
		j.config.Logger.Info("adaptive scheduling: adjusting interval",
			"old_interval", j.currentInterval,
			"new_interval", newInterval,
			"avg_duration_sec", avgDuration,
			"load_threshold_sec", loadThresholdSec)
		j.currentInterval = newInterval
	}
}

// GetCurrentInterval returns the current recompute interval (for testing).
func (j *RecomputeJob) GetCurrentInterval() time.Duration {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.currentInterval
}

// abs returns the absolute value of a float64.
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// recomputeSceneWithScore calculates and stores the trust score for a single scene.
// Returns the new score for variance calculation.
func (j *RecomputeJob) recomputeSceneWithScore(sceneID string) (float64, error) {
	memberships, err := j.dataSource.GetMembershipsByScene(sceneID)
	if err != nil {
		return 0, err
	}

	alliances, err := j.dataSource.GetAlliancesByScene(sceneID)
	if err != nil {
		return 0, err
	}

	score := ComputeTrustScore(memberships, alliances)

	trustScore := SceneTrustScore{
		SceneID:    sceneID,
		Score:      score,
		ComputedAt: time.Now(),
	}

	if err := j.scoreStore.SaveScore(trustScore); err != nil {
		return 0, err
	}

	j.config.Logger.Debug("trust score recomputed",
		"scene_id", sceneID,
		"score", score,
		"memberships", len(memberships),
		"alliances", len(alliances))

	return score, nil
}

// RecomputeNow immediately recomputes all dirty scenes without waiting for the ticker.
// This is useful for testing or forcing immediate updates.
func (j *RecomputeJob) RecomputeNow() {
	j.recomputeDirtyScenes(context.Background())
}
