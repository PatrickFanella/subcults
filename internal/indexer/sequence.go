// Package indexer provides filtering and processing of AT Protocol records
// for the Subcults Jetstream indexer.
package indexer

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"sync"
)

// SequenceTracker manages the last processed sequence number (cursor) for resume functionality.
// The sequence number is typically the TimeUS value from Jetstream messages.
type SequenceTracker interface {
	// GetLastSequence retrieves the last successfully processed sequence number.
	// Returns 0 if no sequence has been recorded yet.
	GetLastSequence(ctx context.Context) (int64, error)

	// UpdateSequence updates the last processed sequence number.
	// This should be called after successfully processing a message.
	UpdateSequence(ctx context.Context, sequence int64) error
}

// PostgresSequenceTracker implements SequenceTracker using the indexer_state table.
type PostgresSequenceTracker struct {
	db     *sql.DB
	logger *slog.Logger
}

// NewPostgresSequenceTracker creates a new PostgresSequenceTracker.
func NewPostgresSequenceTracker(db *sql.DB, logger *slog.Logger) *PostgresSequenceTracker {
	if logger == nil {
		logger = slog.Default()
	}
	return &PostgresSequenceTracker{
		db:     db,
		logger: logger,
	}
}

// GetLastSequence retrieves the last processed cursor from the database.
func (t *PostgresSequenceTracker) GetLastSequence(ctx context.Context) (int64, error) {
	var cursor int64
	query := `SELECT cursor FROM indexer_state ORDER BY id DESC LIMIT 1`
	err := t.db.QueryRowContext(ctx, query).Scan(&cursor)
	if err != nil {
		if err == sql.ErrNoRows {
			// No state exists yet, return 0
			return 0, nil
		}
		return 0, fmt.Errorf("failed to get last sequence: %w", err)
	}
	return cursor, nil
}

// UpdateSequence updates the cursor in the database.
// Only updates if the new sequence is greater than the current one (monotonic).
func (t *PostgresSequenceTracker) UpdateSequence(ctx context.Context, sequence int64) error {
	// Use GREATEST to ensure monotonic updates - only update if new sequence is greater
	query := `UPDATE indexer_state 
	          SET cursor = GREATEST(cursor, $1), last_updated = NOW() 
	          WHERE id = (SELECT id FROM indexer_state ORDER BY id DESC LIMIT 1) 
	          AND $1 > cursor`
	result, err := t.db.ExecContext(ctx, query, sequence)
	if err != nil {
		return fmt.Errorf("failed to update sequence: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		// Either no row exists, or sequence is not greater than current
		// Check if row exists
		var exists bool
		checkQuery := `SELECT EXISTS(SELECT 1 FROM indexer_state LIMIT 1)`
		err = t.db.QueryRowContext(ctx, checkQuery).Scan(&exists)
		if err != nil {
			return fmt.Errorf("failed to check if state exists: %w", err)
		}

		if !exists {
			// No row exists, insert one
			insertQuery := `INSERT INTO indexer_state (cursor, last_updated) VALUES ($1, NOW())`
			_, err = t.db.ExecContext(ctx, insertQuery, sequence)
			if err != nil {
				return fmt.Errorf("failed to insert initial sequence: %w", err)
			}
			t.logger.Debug("inserted initial sequence cursor",
				slog.Int64("cursor", sequence))
		} else {
			// Row exists but sequence is not greater - skip update (monotonic behavior)
			t.logger.Debug("skipped sequence update (not greater than current)",
				slog.Int64("sequence", sequence))
		}
	} else {
		t.logger.Debug("updated sequence cursor",
			slog.Int64("cursor", sequence))
	}

	return nil
}

// InMemorySequenceTracker implements SequenceTracker using in-memory storage.
// This is useful for testing and development.
type InMemorySequenceTracker struct {
	mu       sync.RWMutex
	sequence int64
	logger   *slog.Logger
}

// NewInMemorySequenceTracker creates a new InMemorySequenceTracker.
func NewInMemorySequenceTracker(logger *slog.Logger) *InMemorySequenceTracker {
	if logger == nil {
		logger = slog.Default()
	}
	return &InMemorySequenceTracker{
		logger: logger,
	}
}

// GetLastSequence retrieves the last processed sequence from memory.
func (t *InMemorySequenceTracker) GetLastSequence(ctx context.Context) (int64, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.sequence, nil
}

// UpdateSequence updates the sequence in memory.
func (t *InMemorySequenceTracker) UpdateSequence(ctx context.Context, sequence int64) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Only update if new sequence is greater (monotonically increasing)
	if sequence > t.sequence {
		t.sequence = sequence
		t.logger.Debug("updated sequence cursor",
			slog.Int64("cursor", sequence))
	}

	return nil
}
