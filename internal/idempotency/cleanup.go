// Package idempotency provides cleanup utilities for idempotency key management.
package idempotency

import (
	"log/slog"
	"time"
)

// DefaultExpiry is the default duration after which idempotency keys expire.
// Set to 24 hours as per requirements.
const DefaultExpiry = 24 * time.Hour

// CleanupOldKeys removes idempotency keys older than the specified duration.
// This function should be called periodically (e.g., via cron job) to prevent unbounded growth.
// Returns the number of keys deleted and any error encountered.
func CleanupOldKeys(repo Repository, expiry time.Duration) (int64, error) {
	deleted, err := repo.DeleteOlderThan(expiry)
	if err != nil {
		slog.Error("failed to cleanup old idempotency keys", "error", err)
		return 0, err
	}

	if deleted > 0 {
		slog.Info("cleaned up old idempotency keys", "deleted", deleted, "older_than", expiry)
	}

	return deleted, nil
}

// RunPeriodicCleanup runs the cleanup job periodically at the specified interval.
// This function blocks and should typically be run in a goroutine.
// It will continue running until the provided stop channel is closed.
//
// Example usage:
//
//	stopChan := make(chan struct{})
//	go idempotency.RunPeriodicCleanup(repo, 1*time.Hour, idempotency.DefaultExpiry, stopChan)
//	// ... later when shutting down
//	close(stopChan)
func RunPeriodicCleanup(repo Repository, interval time.Duration, expiry time.Duration, stopChan <-chan struct{}) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Run cleanup immediately on start
	if _, err := CleanupOldKeys(repo, expiry); err != nil {
		slog.Error("initial cleanup failed", "error", err)
	}

	for {
		select {
		case <-ticker.C:
			if _, err := CleanupOldKeys(repo, expiry); err != nil {
				slog.Error("periodic cleanup failed", "error", err)
			}
		case <-stopChan:
			slog.Info("stopping periodic cleanup")
			return
		}
	}
}
