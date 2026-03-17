// Package retention provides data retention enforcement for the Subcults platform.
// It implements scheduled cleanup of expired records according to the data retention policy,
// along with user account export and deletion capabilities.
package retention

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// RetentionTier defines how long different entity types are kept.
type RetentionTier struct {
	EntityType      string
	RetentionPeriod time.Duration
	ArchiveFirst    bool // If true, archive before deletion
}

// DefaultTiers returns the standard retention tiers per the data retention policy.
func DefaultTiers() []RetentionTier {
	return []RetentionTier{
		{EntityType: "scenes", RetentionPeriod: 2 * 365 * 24 * time.Hour, ArchiveFirst: true},
		{EntityType: "events", RetentionPeriod: 365 * 24 * time.Hour, ArchiveFirst: true},
		{EntityType: "posts", RetentionPeriod: 365 * 24 * time.Hour, ArchiveFirst: false},
		{EntityType: "audit_logs", RetentionPeriod: 90 * 24 * time.Hour, ArchiveFirst: false},
		{EntityType: "soft_deleted", RetentionPeriod: 30 * 24 * time.Hour, ArchiveFirst: false},
	}
}

// Repository defines the data operations needed by the retention service.
type Repository interface {
	// CountExpiredRecords returns the number of records past their retention period.
	CountExpiredRecords(ctx context.Context, entityType string, cutoff time.Time) (int64, error)
	// DeleteExpiredRecords removes records older than the cutoff (soft-deleted records only).
	DeleteExpiredRecords(ctx context.Context, entityType string, cutoff time.Time, batchSize int) (int64, error)
	// ArchiveExpiredRecords moves records to archive storage before deletion.
	ArchiveExpiredRecords(ctx context.Context, entityType string, cutoff time.Time, batchSize int) (int64, error)
	// ExportUserData returns all data belonging to a user as a serializable structure.
	ExportUserData(ctx context.Context, userDID string) (*UserDataExport, error)
	// ScheduleAccountDeletion marks an account for deletion after the grace period.
	ScheduleAccountDeletion(ctx context.Context, userDID string, graceEnd time.Time) error
	// ExecuteAccountDeletion permanently removes account data past the grace period.
	ExecuteAccountDeletion(ctx context.Context, userDID string) error
	// GetPendingDeletions returns accounts scheduled for deletion that are past grace.
	GetPendingDeletions(ctx context.Context) ([]PendingDeletion, error)
}

// PendingDeletion represents a scheduled account deletion.
type PendingDeletion struct {
	UserDID     string
	ScheduledAt time.Time
	GraceEndsAt time.Time
}

// UserDataExport contains all user data for export.
type UserDataExport struct {
	UserDID    string                   `json:"user_did"`
	ExportedAt time.Time                `json:"exported_at"`
	Scenes     []map[string]interface{} `json:"scenes"`
	Events     []map[string]interface{} `json:"events"`
	Posts      []map[string]interface{} `json:"posts"`
	Alliances  []map[string]interface{} `json:"alliances"`
}

// ServiceConfig configures the retention service.
type ServiceConfig struct {
	// Tiers defines retention periods per entity type.
	Tiers []RetentionTier
	// CheckInterval is how often the service runs its cleanup loop.
	CheckInterval time.Duration
	// BatchSize is the maximum number of records processed per batch.
	BatchSize int
	// GracePeriod is how long after account deletion request before hard delete.
	GracePeriod time.Duration
	Logger      *slog.Logger
}

// DefaultConfig returns the default retention service configuration.
func DefaultConfig() ServiceConfig {
	return ServiceConfig{
		Tiers:         DefaultTiers(),
		CheckInterval: 24 * time.Hour,
		BatchSize:     1000,
		GracePeriod:   30 * 24 * time.Hour,
		Logger:        slog.Default(),
	}
}

// Service runs periodic retention enforcement.
type Service struct {
	repo     Repository
	config   ServiceConfig
	logger   *slog.Logger
	stopChan chan struct{}
	doneChan chan struct{}
	mu       sync.Mutex
	running  bool
}

// NewService creates a new retention service.
func NewService(repo Repository, config ServiceConfig) *Service {
	if config.Logger == nil {
		config.Logger = slog.Default()
	}
	if config.CheckInterval == 0 {
		config.CheckInterval = 24 * time.Hour
	}
	if config.BatchSize == 0 {
		config.BatchSize = 1000
	}
	if config.GracePeriod == 0 {
		config.GracePeriod = 30 * 24 * time.Hour
	}
	if len(config.Tiers) == 0 {
		config.Tiers = DefaultTiers()
	}
	return &Service{
		repo:     repo,
		config:   config,
		logger:   config.Logger,
		stopChan: make(chan struct{}),
		doneChan: make(chan struct{}),
	}
}

// Start begins the periodic retention enforcement loop.
func (s *Service) Start(ctx context.Context) {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return
	}
	s.running = true
	s.mu.Unlock()
	go s.loop(ctx)
}

// Stop gracefully stops the retention service.
func (s *Service) Stop() {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return
	}
	s.running = false
	s.mu.Unlock()
	close(s.stopChan)
	<-s.doneChan
}

func (s *Service) loop(ctx context.Context) {
	defer close(s.doneChan)
	ticker := time.NewTicker(s.config.CheckInterval)
	defer ticker.Stop()
	// Run immediately on start
	s.runCycle(ctx)
	for {
		select {
		case <-ctx.Done():
			s.logger.Info("retention service stopped (context cancelled)")
			return
		case <-s.stopChan:
			s.logger.Info("retention service stopped")
			return
		case <-ticker.C:
			s.runCycle(ctx)
		}
	}
}

// runCycle executes one full retention enforcement cycle.
func (s *Service) runCycle(ctx context.Context) {
	s.logger.Info("starting retention enforcement cycle")
	start := time.Now()
	var totalDeleted, totalArchived int64

	for _, tier := range s.config.Tiers {
		cutoff := time.Now().Add(-tier.RetentionPeriod)
		count, err := s.repo.CountExpiredRecords(ctx, tier.EntityType, cutoff)
		if err != nil {
			s.logger.Error("failed to count expired records",
				slog.String("entity_type", tier.EntityType),
				slog.String("error", err.Error()))
			continue
		}
		if count == 0 {
			continue
		}

		s.logger.Info("found expired records",
			slog.String("entity_type", tier.EntityType),
			slog.Int64("count", count),
			slog.Time("cutoff", cutoff))

		if tier.ArchiveFirst {
			archived, err := s.repo.ArchiveExpiredRecords(ctx, tier.EntityType, cutoff, s.config.BatchSize)
			if err != nil {
				s.logger.Error("failed to archive expired records",
					slog.String("entity_type", tier.EntityType),
					slog.String("error", err.Error()))
				continue
			}
			totalArchived += archived
		}

		deleted, err := s.repo.DeleteExpiredRecords(ctx, tier.EntityType, cutoff, s.config.BatchSize)
		if err != nil {
			s.logger.Error("failed to delete expired records",
				slog.String("entity_type", tier.EntityType),
				slog.String("error", err.Error()))
			continue
		}
		totalDeleted += deleted
	}

	// Process pending account deletions
	pendingDeleted, err := s.processAccountDeletions(ctx)
	if err != nil {
		s.logger.Error("failed to process account deletions", slog.String("error", err.Error()))
	}

	s.logger.Info("retention enforcement cycle completed",
		slog.Int64("records_deleted", totalDeleted),
		slog.Int64("records_archived", totalArchived),
		slog.Int64("accounts_deleted", pendingDeleted),
		slog.Duration("duration", time.Since(start)))
}

func (s *Service) processAccountDeletions(ctx context.Context) (int64, error) {
	pending, err := s.repo.GetPendingDeletions(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get pending deletions: %w", err)
	}

	var deleted int64
	now := time.Now()
	for _, p := range pending {
		if now.Before(p.GraceEndsAt) {
			continue // Still within grace period
		}
		if err := s.repo.ExecuteAccountDeletion(ctx, p.UserDID); err != nil {
			s.logger.Error("failed to delete account",
				slog.String("user_did", p.UserDID),
				slog.String("error", err.Error()))
			continue
		}
		s.logger.Info("account permanently deleted",
			slog.String("user_did", p.UserDID))
		deleted++
	}
	return deleted, nil
}
