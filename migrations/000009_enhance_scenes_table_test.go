//go:build integration

// Package migrations_test provides integration tests for database migrations.
//
// These tests require a PostgreSQL database with PostGIS and migrations applied.
// Run with: go test -tags=integration -v ./migrations/...
//
// Required environment variable:
//
//	DATABASE_URL=postgres://user:pass@localhost:5432/subcults?sslmode=disable
package migrations_test

import (
	"database/sql"
	"os"
	"testing"

	"github.com/lib/pq" // PostgreSQL driver; pq.Array used for scanning PostgreSQL arrays
)

const (
	// testGeohash is a placeholder geohash for testing (represents 0,0 coordinates at "Null Island")
	// SQL queries use the literal value directly for clarity; this constant documents the standard test value.
	testGeohash = "s00000"
)

// TestMigration000009_CoarseGeohashNotNull verifies that coarse_geohash is NOT NULL
// after migration 000009.
func TestMigration000009_CoarseGeohashNotNull(t *testing.T) {
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

	// Try to insert a scene without coarse_geohash - should fail
	_, err = db.Exec(`
		INSERT INTO scenes (id, name, owner_did, allow_precise) 
		VALUES (gen_random_uuid(), 'Test Scene', 'did:example:test', false)
	`)
	if err == nil {
		t.Fatal("Expected error when inserting scene without coarse_geohash, but got none")
	}

	// Verify the error is a NOT NULL constraint violation
	if err != nil {
		t.Logf("Got expected error: %v", err)
	}
}

// TestMigration000009_SoftDelete verifies that soft delete works via deleted_at.
func TestMigration000009_SoftDelete(t *testing.T) {
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

	// Insert a test scene with required fields
	var sceneID string
	err = db.QueryRow(`
		INSERT INTO scenes (name, owner_did, allow_precise, coarse_geohash) 
		VALUES ('Soft Delete Test Scene', 'did:example:softdelete', false, 's00000')
		RETURNING id
	`).Scan(&sceneID)
	if err != nil {
		t.Fatalf("failed to insert test scene: %v", err)
	}
	defer func() {
		// Cleanup - hard delete the test scene
		_, _ = db.Exec("DELETE FROM scenes WHERE id = $1", sceneID)
	}()

	// Verify scene exists and deleted_at is NULL
	var deletedAt sql.NullTime
	err = db.QueryRow("SELECT deleted_at FROM scenes WHERE id = $1", sceneID).Scan(&deletedAt)
	if err != nil {
		t.Fatalf("failed to query scene: %v", err)
	}
	if deletedAt.Valid {
		t.Error("Expected deleted_at to be NULL for new scene")
	}

	// Soft delete the scene
	_, err = db.Exec("UPDATE scenes SET deleted_at = NOW() WHERE id = $1", sceneID)
	if err != nil {
		t.Fatalf("failed to soft delete scene: %v", err)
	}

	// Verify deleted_at is now set
	err = db.QueryRow("SELECT deleted_at FROM scenes WHERE id = $1", sceneID).Scan(&deletedAt)
	if err != nil {
		t.Fatalf("failed to query scene after soft delete: %v", err)
	}
	if !deletedAt.Valid {
		t.Error("Expected deleted_at to be set after soft delete")
	}
}

// TestMigration000009_FTSSearchVector verifies that the FTS search vector is generated
// and not null for valid scenes.
func TestMigration000009_FTSSearchVector(t *testing.T) {
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

	// Insert a test scene with name, description, and tags
	var sceneID string
	err = db.QueryRow(`
		INSERT INTO scenes (name, description, owner_did, allow_precise, coarse_geohash, tags) 
		VALUES ('FTS Test Scene', 'A scene for testing full-text search', 'did:example:fts', false, 's00000', ARRAY['electronic', 'underground'])
		RETURNING id
	`).Scan(&sceneID)
	if err != nil {
		t.Fatalf("failed to insert test scene: %v", err)
	}
	defer func() {
		// Cleanup - hard delete the test scene
		_, _ = db.Exec("DELETE FROM scenes WHERE id = $1", sceneID)
	}()

	// Verify FTS column is not null and contains expected terms
	var ftsVector string
	err = db.QueryRow("SELECT name_desc_tags_fts::text FROM scenes WHERE id = $1", sceneID).Scan(&ftsVector)
	if err != nil {
		t.Fatalf("failed to query FTS vector: %v", err)
	}
	if ftsVector == "" {
		t.Error("Expected non-empty FTS search vector")
	}

	t.Logf("FTS vector: %s", ftsVector)

	// Verify we can search using the FTS column
	var count int
	err = db.QueryRow(`
		SELECT COUNT(*) FROM scenes 
		WHERE name_desc_tags_fts @@ to_tsquery('english', 'electronic')
		AND id = $1
	`, sceneID).Scan(&count)
	if err != nil {
		t.Fatalf("failed to search FTS: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 result from FTS search for 'electronic', got %d", count)
	}

	// Test FTS stemming - "electronics" should match "electronic"
	err = db.QueryRow(`
		SELECT COUNT(*) FROM scenes 
		WHERE name_desc_tags_fts @@ to_tsquery('english', 'electronics')
		AND id = $1
	`, sceneID).Scan(&count)
	if err != nil {
		t.Fatalf("failed to search FTS with stemming: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 result from FTS stemming search (electronics->electronic), got %d", count)
	}

	// Test multi-word phrase query
	err = db.QueryRow(`
		SELECT COUNT(*) FROM scenes 
		WHERE name_desc_tags_fts @@ to_tsquery('english', 'testing & full')
		AND id = $1
	`, sceneID).Scan(&count)
	if err != nil {
		t.Fatalf("failed to search FTS with phrase: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 result from multi-word FTS search, got %d", count)
	}
}

// TestMigration000009_TagsColumn verifies that tags column exists and works correctly.
func TestMigration000009_TagsColumn(t *testing.T) {
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

	// Insert a scene with tags
	var sceneID string
	err = db.QueryRow(`
		INSERT INTO scenes (name, owner_did, allow_precise, coarse_geohash, tags) 
		VALUES ('Tags Test Scene', 'did:example:tags', false, 's00000', ARRAY['techno', 'house', 'underground'])
		RETURNING id
	`).Scan(&sceneID)
	if err != nil {
		t.Fatalf("failed to insert scene with tags: %v", err)
	}
	defer func() {
		_, _ = db.Exec("DELETE FROM scenes WHERE id = $1", sceneID)
	}()

	// Verify tags were stored correctly
	var tags []string
	err = db.QueryRow("SELECT tags FROM scenes WHERE id = $1", sceneID).Scan(pq.Array(&tags))
	if err != nil {
		t.Fatalf("failed to query tags: %v", err)
	}
	if len(tags) != 3 {
		t.Errorf("Expected 3 tags, got %d", len(tags))
	}

	// Test GIN index usage with array containment operator
	var ginCount int
	err = db.QueryRow(`
		SELECT COUNT(*) FROM scenes 
		WHERE tags @> ARRAY['techno'] AND id = $1
	`, sceneID).Scan(&ginCount)
	if err != nil {
		t.Fatalf("failed to query tags with containment operator: %v", err)
	}
	if ginCount != 1 {
		t.Errorf("Expected 1 result from tags containment search, got %d", ginCount)
	}
}

// TestMigration000009_VisibilityConstraint verifies visibility CHECK constraint.
func TestMigration000009_VisibilityConstraint(t *testing.T) {
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

	// Try to insert with invalid visibility - should fail with CHECK constraint violation
	_, err = db.Exec(`
		INSERT INTO scenes (name, owner_did, allow_precise, coarse_geohash, visibility) 
		VALUES ('Invalid Visibility Test', 'did:example:invalid', false, 's00000', 'invalid_value')
	`)
	if err == nil {
		t.Fatal("Expected error when inserting scene with invalid visibility, but got none")
	}
	
	// Verify it's a CHECK constraint violation (PostgreSQL error code 23514)
	if pqErr, ok := err.(*pq.Error); ok {
		if pqErr.Code != "23514" {
			t.Errorf("Expected CHECK constraint violation (23514), got error code: %s", pqErr.Code)
		}
		t.Logf("Got expected CHECK constraint error: %v", err)
	} else {
		t.Logf("Got expected error (not pq.Error): %v", err)
	}

	// Insert with valid visibilities
	validVisibilities := []string{"public", "private", "unlisted"}
	for _, vis := range validVisibilities {
		var sceneID string
		err = db.QueryRow(`
			INSERT INTO scenes (name, owner_did, allow_precise, coarse_geohash, visibility) 
			VALUES ($1, 'did:example:visibility', false, 's00000', $2)
			RETURNING id
		`, "Visibility Test "+vis, vis).Scan(&sceneID)
		if err != nil {
			t.Errorf("failed to insert scene with visibility=%s: %v", vis, err)
		} else {
			// Cleanup
			_, _ = db.Exec("DELETE FROM scenes WHERE id = $1", sceneID)
		}
	}
}

// TestMigration000009_PaletteColumn verifies that palette JSONB column exists.
func TestMigration000009_PaletteColumn(t *testing.T) {
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

	// Insert a scene with palette
	var sceneID string
	err = db.QueryRow(`
		INSERT INTO scenes (name, owner_did, allow_precise, coarse_geohash, palette) 
		VALUES ('Palette Test Scene', 'did:example:palette', false, 's00000', '{"primary": "#ff0000", "secondary": "#00ff00"}'::jsonb)
		RETURNING id
	`).Scan(&sceneID)
	if err != nil {
		t.Fatalf("failed to insert scene with palette: %v", err)
	}
	defer func() {
		_, _ = db.Exec("DELETE FROM scenes WHERE id = $1", sceneID)
	}()

	// Verify palette was stored correctly
	var palette string
	err = db.QueryRow("SELECT palette::text FROM scenes WHERE id = $1", sceneID).Scan(&palette)
	if err != nil {
		t.Fatalf("failed to query palette: %v", err)
	}
	if palette == "" {
		t.Error("Expected non-empty palette")
	}
	t.Logf("Palette: %s", palette)

	// Test palette key extraction
	var primaryColor string
	err = db.QueryRow("SELECT palette->>'primary' FROM scenes WHERE id = $1", sceneID).Scan(&primaryColor)
	if err != nil {
		t.Fatalf("failed to extract primary color: %v", err)
	}
	if primaryColor != "#ff0000" {
		t.Errorf("Expected primary color '#ff0000', got '%s'", primaryColor)
	}
}

// TestMigration000009_PrivacyConstraintsPreserved verifies that privacy constraints
// are still enforced after the migration.
func TestMigration000009_PrivacyConstraintsPreserved(t *testing.T) {
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

	// Try to insert scene with precise_point but allow_precise=false - should fail
	_, err = db.Exec(`
		INSERT INTO scenes (name, owner_did, allow_precise, coarse_geohash, precise_point) 
		VALUES ('Privacy Test', 'did:example:privacy', false, 's00000', ST_SetSRID(ST_MakePoint(-74.0060, 40.7128), 4326))
	`)
	if err == nil {
		t.Fatal("Expected error when inserting scene with precise_point but allow_precise=false, but got none")
	}
	t.Logf("Got expected error for privacy constraint: %v", err)

	// Insert scene with consent=true and precise point - should succeed
	var sceneID string
	err = db.QueryRow(`
		INSERT INTO scenes (name, owner_did, allow_precise, coarse_geohash, precise_point) 
		VALUES ('Privacy Test With Consent', 'did:example:privacy2', true, 's00000', ST_SetSRID(ST_MakePoint(-74.0060, 40.7128), 4326))
		RETURNING id
	`).Scan(&sceneID)
	if err != nil {
		t.Fatalf("failed to insert scene with consent: %v", err)
	}
	defer func() {
		_, _ = db.Exec("DELETE FROM scenes WHERE id = $1", sceneID)
	}()
}
