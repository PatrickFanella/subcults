//go:build integration

// Package testutil provides shared test infrastructure for integration tests.
package testutil

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// TestDB wraps a test Postgres container with helpers.
type TestDB struct {
	Container *postgres.PostgresContainer
	DB        *sql.DB
	ConnStr   string
}

// NewTestDB spins up a Postgres+PostGIS container and runs all migrations.
// Call cleanup in a t.Cleanup to tear down the container.
func NewTestDB(t *testing.T) *TestDB {
	t.Helper()
	ctx := context.Background()

	container, err := postgres.Run(ctx,
		"postgis/postgis:16-3.4-alpine",
		postgres.WithDatabase("subcults_test"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	if err != nil {
		t.Fatalf("failed to start postgres container: %v", err)
	}

	connStr, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("failed to get connection string: %v", err)
	}

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}

	if err := db.Ping(); err != nil {
		t.Fatalf("failed to ping database: %v", err)
	}

	tdb := &TestDB{
		Container: container,
		DB:        db,
		ConnStr:   connStr,
	}

	if err := tdb.runMigrations(t); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	t.Cleanup(func() {
		db.Close()
		if err := container.Terminate(ctx); err != nil {
			t.Logf("warning: failed to terminate container: %v", err)
		}
	})

	return tdb
}

// runMigrations applies all .up.sql migration files in order.
func (tdb *TestDB) runMigrations(t *testing.T) error {
	t.Helper()

	migrationsDir := findMigrationsDir()
	if migrationsDir == "" {
		return fmt.Errorf("migrations directory not found")
	}

	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return fmt.Errorf("reading migrations dir: %w", err)
	}

	var upFiles []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".up.sql") {
			upFiles = append(upFiles, e.Name())
		}
	}
	sort.Strings(upFiles)

	for _, f := range upFiles {
		content, err := os.ReadFile(filepath.Join(migrationsDir, f))
		if err != nil {
			return fmt.Errorf("reading migration %s: %w", f, err)
		}
		if _, err := tdb.DB.Exec(string(content)); err != nil {
			return fmt.Errorf("executing migration %s: %w", f, err)
		}
	}

	t.Logf("applied %d migrations", len(upFiles))
	return nil
}

// ExecTx runs fn inside a transaction and rolls back afterward, keeping the DB clean.
func (tdb *TestDB) ExecTx(t *testing.T, fn func(tx *sql.Tx)) {
	t.Helper()
	tx, err := tdb.DB.Begin()
	if err != nil {
		t.Fatalf("begin transaction: %v", err)
	}
	defer tx.Rollback() //nolint:errcheck
	fn(tx)
}

// findMigrationsDir walks up from the current file to find the migrations directory.
func findMigrationsDir() string {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		return ""
	}
	dir := filepath.Dir(thisFile)
	for i := 0; i < 10; i++ {
		candidate := filepath.Join(dir, "migrations")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
		dir = filepath.Dir(dir)
	}
	return ""
}
