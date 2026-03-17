// Package indexer provides filtering and processing of AT Protocol records
// for the Subcults Jetstream indexer.
package indexer

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"
)

// CleanupService handles periodic cleanup of old idempotency keys.
// This prevents unbounded growth of the ingestion_idempotency table.
type CleanupService struct {
	db              *sql.DB
	logger          *slog.Logger
	retentionPeriod time.Duration
	cleanupInterval time.Duration
	stopChan        chan struct{}
	doneChan        chan struct{}
}

// CleanupConfig contains configuration for the cleanup service.
type CleanupConfig struct {
	// RetentionPeriod is how long to keep idempotency keys before cleanup.
	// Default: 24 hours (keys older than this will be deleted).
	RetentionPeriod time.Duration

	// CleanupInterval is how often to run the cleanup job.
	// Default: 1 hour.
	CleanupInterval time.Duration
}

// DefaultCleanupConfig returns the default cleanup configuration.
func DefaultCleanupConfig() CleanupConfig {
	return CleanupConfig{
		RetentionPeriod: 24 * time.Hour,
		CleanupInterval: 1 * time.Hour,
	}
}

// NewCleanupService creates a new cleanup service.
func NewCleanupService(db *sql.DB, logger *slog.Logger, config CleanupConfig) *CleanupService {
	if logger == nil {
		logger = slog.Default()
	}

	// Apply defaults if not set
	if config.RetentionPeriod == 0 {
		config.RetentionPeriod = 24 * time.Hour
	}
	if config.CleanupInterval == 0 {
		config.CleanupInterval = 1 * time.Hour
	}

	return &CleanupService{
		db:              db,
		logger:          logger,
		retentionPeriod: config.RetentionPeriod,
		cleanupInterval: config.CleanupInterval,
		stopChan:        make(chan struct{}),
		doneChan:        make(chan struct{}),
	}
}

// Start begins the cleanup service.
// It runs in a background goroutine and performs cleanup at regular intervals.
func (s *CleanupService) Start(ctx context.Context) {
	go s.run(ctx)
}

// Stop gracefully stops the cleanup service.
func (s *CleanupService) Stop() {
	close(s.stopChan)
	<-s.doneChan
}

// run executes the cleanup loop.
func (s *CleanupService) run(ctx context.Context) {
	defer close(s.doneChan)

	ticker := time.NewTicker(s.cleanupInterval)
	defer ticker.Stop()

	s.logger.Info("cleanup service started",
		slog.Duration("retention_period", s.retentionPeriod),
		slog.Duration("cleanup_interval", s.cleanupInterval))

	// Run initial cleanup immediately
	if err := s.cleanup(ctx); err != nil {
		s.logger.Error("initial cleanup failed",
			slog.String("error", err.Error()))
	}

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("cleanup service stopping due to context cancellation")
			return
		case <-s.stopChan:
			s.logger.Info("cleanup service stopping")
			return
		case <-ticker.C:
			if err := s.cleanup(ctx); err != nil {
				s.logger.Error("cleanup failed",
					slog.String("error", err.Error()))
			}
		}
	}
}

// cleanup deletes old idempotency keys from the database.
//
// IMPORTANT: Cleanup retention strategy considerations:
// - Default 24h retention may be too short for long replay/backfill scenarios
// - If replaying from an old cursor (>24h), records can be reprocessed including deleted content
// - For production with replay requirements, consider:
//  1. Longer retention (e.g., 7-30 days) to cover expected replay windows
//  2. Indefinite retention for delete operations to prevent re-ingestion
//  3. Separate high-water mark / tombstone mechanism for correctness across long replays
//
// - Current implementation prioritizes privacy (minimal retention) over replay correctness
// - Adjust RetentionPeriod based on your replay patterns and correctness requirements
func (s *CleanupService) cleanup(ctx context.Context) error {
	start := time.Now()

	cutoffTime := time.Now().Add(-s.retentionPeriod)

	query := `
		DELETE FROM ingestion_idempotency
		WHERE created_at < $1
	`

	result, err := s.db.ExecContext(ctx, query, cutoffTime)
	if err != nil {
		return fmt.Errorf("failed to delete old idempotency keys: %w", err)
	}

	rowsDeleted, err := result.RowsAffected()
	if err != nil {
		s.logger.Warn("failed to get rows affected count",
			slog.String("error", err.Error()))
		rowsDeleted = -1
	}

	duration := time.Since(start)

	s.logger.Info("cleanup completed",
		slog.Int64("rows_deleted", rowsDeleted),
		slog.Duration("duration", duration),
		slog.Time("cutoff_time", cutoffTime))

	return nil
}

// InMemoryCleanupService provides an in-memory cleanup implementation for testing.
type InMemoryCleanupService struct {
	repo            *InMemoryRecordRepository
	logger          *slog.Logger
	retentionPeriod time.Duration
	cleanupInterval time.Duration
	stopChan        chan struct{}
	doneChan        chan struct{}
}

// NewInMemoryCleanupService creates a new in-memory cleanup service.
func NewInMemoryCleanupService(repo *InMemoryRecordRepository, logger *slog.Logger, config CleanupConfig) *InMemoryCleanupService {
	if logger == nil {
		logger = slog.Default()
	}

	// Apply defaults
	if config.RetentionPeriod == 0 {
		config.RetentionPeriod = 24 * time.Hour
	}
	if config.CleanupInterval == 0 {
		config.CleanupInterval = 1 * time.Hour
	}

	return &InMemoryCleanupService{
		repo:            repo,
		logger:          logger,
		retentionPeriod: config.RetentionPeriod,
		cleanupInterval: config.CleanupInterval,
		stopChan:        make(chan struct{}),
		doneChan:        make(chan struct{}),
	}
}

// Start begins the cleanup service.
func (s *InMemoryCleanupService) Start(ctx context.Context) {
	go s.run(ctx)
}

// Stop gracefully stops the cleanup service.
func (s *InMemoryCleanupService) Stop() {
	close(s.stopChan)
	<-s.doneChan
}

// run executes the cleanup loop.
func (s *InMemoryCleanupService) run(ctx context.Context) {
	defer close(s.doneChan)

	ticker := time.NewTicker(s.cleanupInterval)
	defer ticker.Stop()

	s.logger.Info("in-memory cleanup service started",
		slog.Duration("retention_period", s.retentionPeriod),
		slog.Duration("cleanup_interval", s.cleanupInterval))

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("in-memory cleanup service stopping due to context cancellation")
			return
		case <-s.stopChan:
			s.logger.Info("in-memory cleanup service stopping")
			return
		case <-ticker.C:
			// In-memory repository doesn't track timestamps, so cleanup is a no-op
			// This is intentional for testing simplicity
			s.logger.Debug("in-memory cleanup cycle (no-op)")
		}
	}
}
