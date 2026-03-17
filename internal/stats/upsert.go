// Package stats provides utilities for tracking upsert operation statistics.
package stats

import (
	"fmt"
	"log/slog"
	"sync/atomic"
)

// UpsertStats tracks cumulative statistics for upsert operations.
// All operations are thread-safe using atomic counters.
type UpsertStats struct {
	inserted int64 // Total records inserted
	updated  int64 // Total records updated
}

// NewUpsertStats creates a new UpsertStats instance.
func NewUpsertStats() *UpsertStats {
	return &UpsertStats{}
}

// RecordInsert increments the inserted counter.
func (s *UpsertStats) RecordInsert() {
	atomic.AddInt64(&s.inserted, 1)
}

// RecordUpdate increments the updated counter.
func (s *UpsertStats) RecordUpdate() {
	atomic.AddInt64(&s.updated, 1)
}

// Inserted returns the total number of inserts.
func (s *UpsertStats) Inserted() int64 {
	return atomic.LoadInt64(&s.inserted)
}

// Updated returns the total number of updates.
func (s *UpsertStats) Updated() int64 {
	return atomic.LoadInt64(&s.updated)
}

// Total returns the total number of upsert operations (inserts + updates).
func (s *UpsertStats) Total() int64 {
	return s.Inserted() + s.Updated()
}

// Reset resets all counters to zero.
func (s *UpsertStats) Reset() {
	atomic.StoreInt64(&s.inserted, 0)
	atomic.StoreInt64(&s.updated, 0)
}

// String returns a human-readable summary of the statistics.
func (s *UpsertStats) String() string {
	return fmt.Sprintf("inserted=%d updated=%d total=%d", s.Inserted(), s.Updated(), s.Total())
}

// LogSummary logs a summary of upsert statistics at INFO level.
// Useful for periodic reporting during ingestion.
func (s *UpsertStats) LogSummary(logger *slog.Logger, entity string) {
	logger.Info("upsert statistics",
		"entity", entity,
		"inserted", s.Inserted(),
		"updated", s.Updated(),
		"total", s.Total(),
	)
}
