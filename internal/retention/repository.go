package retention

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// InMemoryRepository provides an in-memory implementation of the retention Repository.
// Used for testing and development.
type InMemoryRepository struct {
	mu               sync.RWMutex
	expiredCounts    map[string]int64
	deletedCounts    map[string]int64
	archivedCounts   map[string]int64
	userExports      map[string]*UserDataExport
	pendingDeletions []PendingDeletion
	deletedAccounts  []string
	logger           *slog.Logger
}

// NewInMemoryRepository creates a new in-memory retention repository.
func NewInMemoryRepository(logger *slog.Logger) *InMemoryRepository {
	if logger == nil {
		logger = slog.Default()
	}
	return &InMemoryRepository{
		expiredCounts:  make(map[string]int64),
		deletedCounts:  make(map[string]int64),
		archivedCounts: make(map[string]int64),
		userExports:    make(map[string]*UserDataExport),
		logger:         logger,
	}
}

// SetExpiredCount sets the number of expired records for an entity type (for testing).
func (r *InMemoryRepository) SetExpiredCount(entityType string, count int64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.expiredCounts[entityType] = count
}

// AddUserExport adds a user data export (for testing).
func (r *InMemoryRepository) AddUserExport(userDID string, export *UserDataExport) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.userExports[userDID] = export
}

// AddPendingDeletion queues a deletion (for testing).
func (r *InMemoryRepository) AddPendingDeletion(p PendingDeletion) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.pendingDeletions = append(r.pendingDeletions, p)
}

// DeletedAccounts returns the list of permanently deleted accounts (for testing).
func (r *InMemoryRepository) DeletedAccounts() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]string, len(r.deletedAccounts))
	copy(out, r.deletedAccounts)
	return out
}

// GetDeletedCount returns how many records were deleted for an entity type (for testing).
func (r *InMemoryRepository) GetDeletedCount(entityType string) int64 {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.deletedCounts[entityType]
}

// GetArchivedCount returns how many records were archived for an entity type (for testing).
func (r *InMemoryRepository) GetArchivedCount(entityType string) int64 {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.archivedCounts[entityType]
}

func (r *InMemoryRepository) CountExpiredRecords(_ context.Context, entityType string, _ time.Time) (int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.expiredCounts[entityType], nil
}

func (r *InMemoryRepository) DeleteExpiredRecords(_ context.Context, entityType string, _ time.Time, batchSize int) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	expired := r.expiredCounts[entityType]
	deleted := expired
	if deleted > int64(batchSize) {
		deleted = int64(batchSize)
	}
	r.deletedCounts[entityType] += deleted
	r.expiredCounts[entityType] -= deleted
	return deleted, nil
}

func (r *InMemoryRepository) ArchiveExpiredRecords(_ context.Context, entityType string, _ time.Time, batchSize int) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	expired := r.expiredCounts[entityType]
	archived := expired
	if archived > int64(batchSize) {
		archived = int64(batchSize)
	}
	r.archivedCounts[entityType] += archived
	return archived, nil
}

func (r *InMemoryRepository) ExportUserData(_ context.Context, userDID string) (*UserDataExport, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	export, ok := r.userExports[userDID]
	if !ok {
		return nil, fmt.Errorf("user not found: %s", userDID)
	}
	return export, nil
}

func (r *InMemoryRepository) ScheduleAccountDeletion(_ context.Context, userDID string, graceEnd time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.pendingDeletions = append(r.pendingDeletions, PendingDeletion{
		UserDID:     userDID,
		ScheduledAt: time.Now(),
		GraceEndsAt: graceEnd,
	})
	return nil
}

func (r *InMemoryRepository) ExecuteAccountDeletion(_ context.Context, userDID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.deletedAccounts = append(r.deletedAccounts, userDID)
	// Remove from pending
	for i, p := range r.pendingDeletions {
		if p.UserDID == userDID {
			r.pendingDeletions = append(r.pendingDeletions[:i], r.pendingDeletions[i+1:]...)
			break
		}
	}
	return nil
}

func (r *InMemoryRepository) GetPendingDeletions(_ context.Context) ([]PendingDeletion, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]PendingDeletion, len(r.pendingDeletions))
	copy(out, r.pendingDeletions)
	return out, nil
}

// Compile-time interface check
var _ Repository = (*InMemoryRepository)(nil)
