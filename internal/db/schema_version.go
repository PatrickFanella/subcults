// Package db provides database utilities and connection handling for Subcults.
package db

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
)

// MinSchemaVersion is the minimum schema version required for the application to start.
// This should be updated when migrations add features that running code depends on.
const MinSchemaVersion = 28

// SchemaVersionChecker queries and validates the application schema version.
type SchemaVersionChecker struct {
	db     *sql.DB
	logger *slog.Logger
}

// NewSchemaVersionChecker creates a new schema version checker.
func NewSchemaVersionChecker(db *sql.DB, logger *slog.Logger) *SchemaVersionChecker {
	if logger == nil {
		logger = slog.Default()
	}
	return &SchemaVersionChecker{db: db, logger: logger}
}

// SchemaInfo holds the current schema version and when it was applied.
type SchemaInfo struct {
	Version     int    `json:"version"`
	Description string `json:"description"`
	AppliedAt   string `json:"applied_at"`
}

// GetCurrentVersion returns the latest applied schema version.
// Returns 0 if the schema_version table doesn't exist or has no rows.
func (c *SchemaVersionChecker) GetCurrentVersion(ctx context.Context) (SchemaInfo, error) {
	var info SchemaInfo
	query := `SELECT version, description, applied_at::text FROM schema_version ORDER BY id DESC LIMIT 1`
	err := c.db.QueryRowContext(ctx, query).Scan(&info.Version, &info.Description, &info.AppliedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return SchemaInfo{}, nil
		}
		return SchemaInfo{}, fmt.Errorf("failed to get schema version: %w", err)
	}
	return info, nil
}

// EnsureCompatible checks that the current schema version meets the minimum requirement.
// Returns an error if the schema is too old for this application version.
func (c *SchemaVersionChecker) EnsureCompatible(ctx context.Context) error {
	info, err := c.GetCurrentVersion(ctx)
	if err != nil {
		return fmt.Errorf("schema compatibility check failed: %w", err)
	}

	if info.Version < MinSchemaVersion {
		return fmt.Errorf(
			"schema version %d is below minimum required version %d — run migrations before starting",
			info.Version, MinSchemaVersion,
		)
	}

	c.logger.Info("schema version compatible",
		slog.Int("current_version", info.Version),
		slog.Int("min_version", MinSchemaVersion),
		slog.String("description", info.Description),
	)
	return nil
}

// RecordVersion inserts a new schema version entry (called after migrations).
func (c *SchemaVersionChecker) RecordVersion(ctx context.Context, version int, description string) error {
	query := `INSERT INTO schema_version (version, description) VALUES ($1, $2)`
	_, err := c.db.ExecContext(ctx, query, version, description)
	if err != nil {
		return fmt.Errorf("failed to record schema version: %w", err)
	}
	return nil
}
