package db

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"sync"
)

// InMemorySchemaVersionChecker is a test double for SchemaVersionChecker
// that doesn't require a real database connection.
type InMemorySchemaVersionChecker struct {
	mu       sync.RWMutex
	versions []SchemaInfo
	logger   *slog.Logger
}

// NewInMemorySchemaVersionChecker creates a new in-memory schema version checker.
func NewInMemorySchemaVersionChecker(logger *slog.Logger) *InMemorySchemaVersionChecker {
	if logger == nil {
		logger = slog.Default()
	}
	return &InMemorySchemaVersionChecker{logger: logger}
}

// GetCurrentVersion returns the latest version from the in-memory store.
func (c *InMemorySchemaVersionChecker) GetCurrentVersion(_ context.Context) (SchemaInfo, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if len(c.versions) == 0 {
		return SchemaInfo{}, nil
	}
	return c.versions[len(c.versions)-1], nil
}

// RecordVersion stores a new schema version in memory.
func (c *InMemorySchemaVersionChecker) RecordVersion(_ context.Context, version int, description string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.versions = append(c.versions, SchemaInfo{
		Version:     version,
		Description: description,
		AppliedAt:   "in-memory",
	})
	return nil
}

// EnsureCompatible checks the minimum schema version against recorded versions.
func (c *InMemorySchemaVersionChecker) EnsureCompatible(ctx context.Context) error {
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
	)
	return nil
}

// SchemaVersionStore defines the interface for schema version operations.
// Both SchemaVersionChecker (Postgres) and InMemorySchemaVersionChecker implement this.
type SchemaVersionStore interface {
	GetCurrentVersion(ctx context.Context) (SchemaInfo, error)
	RecordVersion(ctx context.Context, version int, description string) error
	EnsureCompatible(ctx context.Context) error
}

// Compile-time interface checks
var _ SchemaVersionStore = (*SchemaVersionChecker)(nil)
var _ SchemaVersionStore = (*InMemorySchemaVersionChecker)(nil)

// NewSchemaVersionStore creates the appropriate SchemaVersionStore based on whether
// a database connection is provided.
func NewSchemaVersionStore(db *sql.DB, logger *slog.Logger) SchemaVersionStore {
	if db == nil {
		return NewInMemorySchemaVersionChecker(logger)
	}
	return NewSchemaVersionChecker(db, logger)
}
