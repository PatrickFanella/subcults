//go:build integration

// Package db provides database utilities and connection handling for Subcults.
//
// Integration tests in this package require a PostgreSQL database with PostGIS.
// Run with: go test -tags=integration -v ./internal/db/...
//
// Required environment variable:
//
//	DATABASE_URL=postgres://user:pass@localhost:5432/subcults?sslmode=disable
package db

import (
	"database/sql"
	"os"
	"testing"

	_ "github.com/lib/pq" // PostgreSQL driver
)

// TestPostGISVersion verifies that PostGIS is available and returns a version string.
// This is an integration test that requires a real database connection.
//
// To run this test:
//
//	export DATABASE_URL='postgres://user:pass@localhost:5432/subcults?sslmode=disable'
//	go test -tags=integration -v ./internal/db/...
func TestPostGISVersion(t *testing.T) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL not set; skipping integration test")
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		t.Fatalf("failed to ping database: %v", err)
	}

	var version string
	// VersionQuery is "SELECT PostGIS_Version()" - defined in db.go
	err = db.QueryRow(VersionQuery).Scan(&version)
	if err != nil {
		t.Logf("Hint: Ensure PostGIS is enabled with: CREATE EXTENSION IF NOT EXISTS postgis;")
		t.Fatalf("PostGIS version query failed: %v", err)
	}

	if version == "" {
		t.Error("PostGIS version returned empty string; expected a version like '3.4 USE_GEOS=1 USE_PROJ=1 USE_STATS=1'")
	}

	t.Logf("PostGIS version: %s", version)
}

// TestPostGISExtensionExists verifies that the PostGIS extension is installed.
func TestPostGISExtensionExists(t *testing.T) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL not set; skipping integration test")
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		t.Fatalf("failed to ping database: %v", err)
	}
	var extname string
	err = db.QueryRow("SELECT extname FROM pg_extension WHERE extname = 'postgis'").Scan(&extname)
	if err == sql.ErrNoRows {
		t.Fatal("PostGIS extension is not installed; run: CREATE EXTENSION IF NOT EXISTS postgis;")
	}
	if err != nil {
		t.Fatalf("failed to query pg_extension: %v", err)
	}

	if extname != "postgis" {
		t.Errorf("expected extension name 'postgis', got %q", extname)
	}
}
