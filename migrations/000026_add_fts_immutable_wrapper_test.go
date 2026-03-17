//go:build integration

// Package migrations_test provides integration tests for FTS functionality.
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

	_ "github.com/lib/pq" // PostgreSQL driver; imported for side-effects (driver registration)
)

// TestMigration000026_FTSImmutableWrapper verifies that the IMMUTABLE wrapper function
// exists and can be used for full-text search queries.
func TestMigration000026_FTSImmutableWrapper(t *testing.T) {
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

	// Verify the IMMUTABLE wrapper function exists
	var funcExists bool
	err = db.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM pg_proc p
			JOIN pg_namespace n ON p.pronamespace = n.oid
			WHERE p.proname = 'to_tsvector_immutable'
			AND n.nspname = 'public'
		)
	`).Scan(&funcExists)
	if err != nil {
		t.Fatalf("failed to check function existence: %v", err)
	}
	if !funcExists {
		t.Fatal("to_tsvector_immutable function does not exist")
	}

	// Verify function is marked IMMUTABLE
	var volatility string
	err = db.QueryRow(`
		SELECT p.provolatile FROM pg_proc p
		JOIN pg_namespace n ON p.pronamespace = n.oid
		WHERE p.proname = 'to_tsvector_immutable'
		AND n.nspname = 'public'
	`).Scan(&volatility)
	if err != nil {
		t.Fatalf("failed to check function volatility: %v", err)
	}
	// 'i' = IMMUTABLE, 's' = STABLE, 'v' = VOLATILE
	if volatility != "i" {
		t.Errorf("Expected function to be IMMUTABLE (i), got %s", volatility)
	}
}

// TestMigration000026_ScenesFTSIndex verifies the FTS index on scenes table works correctly.
func TestMigration000026_ScenesFTSIndex(t *testing.T) {
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

	// Insert test scene with searchable content
	var sceneID string
	err = db.QueryRow(`
		INSERT INTO scenes (name, description, owner_did, allow_precise, coarse_geohash, tags)
		VALUES (
			'Underground Techno Warehouse',
			'Late night electronic music events in industrial spaces',
			'did:example:ftstest',
			false,
			's00000',
			ARRAY['techno', 'warehouse', 'industrial']
		)
		RETURNING id
	`).Scan(&sceneID)
	if err != nil {
		t.Fatalf("failed to insert test scene: %v", err)
	}
	defer func() {
		_, _ = db.Exec("DELETE FROM scenes WHERE id = $1", sceneID)
	}()

	// Test FTS search on name
	var count int
	err = db.QueryRow(`
		SELECT COUNT(*) FROM scenes
		WHERE to_tsvector_immutable(
			COALESCE(name, '') || ' ' ||
			COALESCE(description, '') || ' ' ||
			COALESCE(array_to_string(tags, ' '), '')
		) @@ to_tsquery('english', 'techno')
		AND id = $1
	`, sceneID).Scan(&count)
	if err != nil {
		t.Fatalf("failed to search by name: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 result for 'techno' search, got %d", count)
	}

	// Test FTS search on description
	err = db.QueryRow(`
		SELECT COUNT(*) FROM scenes
		WHERE to_tsvector_immutable(
			COALESCE(name, '') || ' ' ||
			COALESCE(description, '') || ' ' ||
			COALESCE(array_to_string(tags, ' '), '')
		) @@ to_tsquery('english', 'electronic')
		AND id = $1
	`, sceneID).Scan(&count)
	if err != nil {
		t.Fatalf("failed to search by description: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 result for 'electronic' search, got %d", count)
	}

	// Test FTS search on tags
	err = db.QueryRow(`
		SELECT COUNT(*) FROM scenes
		WHERE to_tsvector_immutable(
			COALESCE(name, '') || ' ' ||
			COALESCE(description, '') || ' ' ||
			COALESCE(array_to_string(tags, ' '), '')
		) @@ to_tsquery('english', 'warehouse')
		AND id = $1
	`, sceneID).Scan(&count)
	if err != nil {
		t.Fatalf("failed to search by tags: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 result for 'warehouse' search, got %d", count)
	}

	// Test FTS with multi-word query
	err = db.QueryRow(`
		SELECT COUNT(*) FROM scenes
		WHERE to_tsvector_immutable(
			COALESCE(name, '') || ' ' ||
			COALESCE(description, '') || ' ' ||
			COALESCE(array_to_string(tags, ' '), '')
		) @@ to_tsquery('english', 'underground & electronic')
		AND id = $1
	`, sceneID).Scan(&count)
	if err != nil {
		t.Fatalf("failed to search with multi-word query: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 result for 'underground & electronic' search, got %d", count)
	}
}

// TestMigration000026_EventsFTSIndex verifies the FTS index on events table works correctly.
func TestMigration000026_EventsFTSIndex(t *testing.T) {
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

	// Create test scene first (required FK)
	var sceneID string
	err = db.QueryRow(`
		INSERT INTO scenes (name, owner_did, allow_precise, coarse_geohash)
		VALUES ('FTS Event Test Scene', 'did:example:eventfts', false, 's00000')
		RETURNING id
	`).Scan(&sceneID)
	if err != nil {
		t.Fatalf("failed to insert test scene: %v", err)
	}
	defer func() {
		_, _ = db.Exec("DELETE FROM scenes WHERE id = $1", sceneID)
	}()

	// Insert test event
	var eventID string
	err = db.QueryRow(`
		INSERT INTO events (scene_id, title, coarse_geohash, starts_at, tags)
		VALUES (
			$1,
			'Midnight Bass Session',
			's00000',
			NOW() + INTERVAL '1 day',
			ARRAY['bass', 'dubstep', 'midnight']
		)
		RETURNING id
	`, sceneID).Scan(&eventID)
	if err != nil {
		t.Fatalf("failed to insert test event: %v", err)
	}
	defer func() {
		_, _ = db.Exec("DELETE FROM events WHERE id = $1", eventID)
	}()

	// Test FTS search on title
	var count int
	err = db.QueryRow(`
		SELECT COUNT(*) FROM events
		WHERE to_tsvector_immutable(
			COALESCE(title, '') || ' ' ||
			COALESCE(array_to_string(tags, ' '), '')
		) @@ to_tsquery('english', 'midnight')
		AND id = $1
	`, eventID).Scan(&count)
	if err != nil {
		t.Fatalf("failed to search by title: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 result for 'midnight' search, got %d", count)
	}

	// Test FTS search on tags
	err = db.QueryRow(`
		SELECT COUNT(*) FROM events
		WHERE to_tsvector_immutable(
			COALESCE(title, '') || ' ' ||
			COALESCE(array_to_string(tags, ' '), '')
		) @@ to_tsquery('english', 'bass')
		AND id = $1
	`, eventID).Scan(&count)
	if err != nil {
		t.Fatalf("failed to search by tags: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 result for 'bass' search, got %d", count)
	}
}

// TestMigration000026_PostsFTSIndex verifies the FTS index on posts table works correctly.
func TestMigration000026_PostsFTSIndex(t *testing.T) {
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

	// Create test scene first (required FK)
	var sceneID string
	err = db.QueryRow(`
		INSERT INTO scenes (name, owner_did, allow_precise, coarse_geohash)
		VALUES ('FTS Post Test Scene', 'did:example:postfts', false, 's00000')
		RETURNING id
	`).Scan(&sceneID)
	if err != nil {
		t.Fatalf("failed to insert test scene: %v", err)
	}
	defer func() {
		_, _ = db.Exec("DELETE FROM scenes WHERE id = $1", sceneID)
	}()

	// Insert test post
	var postID string
	err = db.QueryRow(`
		INSERT INTO posts (scene_id, author_did, text)
		VALUES (
			$1,
			'did:example:author',
			'Amazing underground techno set last night! The warehouse was packed and the vibes were incredible.'
		)
		RETURNING id
	`, sceneID).Scan(&postID)
	if err != nil {
		t.Fatalf("failed to insert test post: %v", err)
	}
	defer func() {
		_, _ = db.Exec("DELETE FROM posts WHERE id = $1", postID)
	}()

	// Test FTS search on text
	var count int
	err = db.QueryRow(`
		SELECT COUNT(*) FROM posts
		WHERE to_tsvector_immutable(COALESCE(text, ''))
		@@ to_tsquery('english', 'techno')
		AND id = $1
	`, postID).Scan(&count)
	if err != nil {
		t.Fatalf("failed to search by text: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 result for 'techno' search, got %d", count)
	}

	// Test FTS with multi-word search
	err = db.QueryRow(`
		SELECT COUNT(*) FROM posts
		WHERE to_tsvector_immutable(COALESCE(text, ''))
		@@ to_tsquery('english', 'warehouse & underground')
		AND id = $1
	`, postID).Scan(&count)
	if err != nil {
		t.Fatalf("failed to search with multi-word: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 result for 'warehouse & underground' search, got %d", count)
	}
}

// TestMigration000026_FTSIndexExists verifies that all FTS indexes exist in the schema.
func TestMigration000026_FTSIndexExists(t *testing.T) {
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

	// Check for scenes FTS index
	expectedIndexes := []string{
		"idx_scenes_name_desc_tags_fts",
		"idx_events_title_tags_fts",
		"idx_posts_text_fts",
	}

	for _, indexName := range expectedIndexes {
		var exists bool
		err = db.QueryRow(`
			SELECT EXISTS(
				SELECT 1 FROM pg_indexes
				WHERE schemaname = 'public'
				AND indexname = $1
			)
		`, indexName).Scan(&exists)
		if err != nil {
			t.Fatalf("failed to check index %s: %v", indexName, err)
		}
		if !exists {
			t.Errorf("Expected index %s to exist", indexName)
		}
	}
}

// TestMigration000026_FTSStemming verifies that PostgreSQL FTS automatic word stemming
// works correctly as documented in the README. Tests that stem variants match.
func TestMigration000026_FTSStemming(t *testing.T) {
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

	// Create test scene first (required FK)
	var sceneID string
	err = db.QueryRow(`
		INSERT INTO scenes (name, owner_did, allow_precise, coarse_geohash)
		VALUES ('Stemming Test Scene', 'did:example:stemming', false, 's00000')
		RETURNING id
	`).Scan(&sceneID)
	if err != nil {
		t.Fatalf("failed to insert test scene: %v", err)
	}
	defer func() {
		_, _ = db.Exec("DELETE FROM scenes WHERE id = $1", sceneID)
	}()

	// Test 1: Insert post with "warehouses" and search for "warehouse"
	var postID1 string
	err = db.QueryRow(`
		INSERT INTO posts (scene_id, author_did, text)
		VALUES (
			$1,
			'did:example:author1',
			'The industrial warehouses downtown have incredible acoustics for underground shows.'
		)
		RETURNING id
	`, sceneID).Scan(&postID1)
	if err != nil {
		t.Fatalf("failed to insert test post 1: %v", err)
	}
	defer func() {
		_, _ = db.Exec("DELETE FROM posts WHERE id = $1", postID1)
	}()

	// Search for "warehouse" (singular) should match "warehouses" (plural)
	var count int
	err = db.QueryRow(`
		SELECT COUNT(*) FROM posts
		WHERE to_tsvector_immutable(COALESCE(text, ''))
		@@ to_tsquery('english', 'warehouse')
		AND id = $1
	`, postID1).Scan(&count)
	if err != nil {
		t.Fatalf("failed to search for 'warehouse': %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 result when searching 'warehouse' to match 'warehouses', got %d", count)
	}

	// Test 2: Insert post with "electronic" and search for "electronics"
	var postID2 string
	err = db.QueryRow(`
		INSERT INTO posts (scene_id, author_did, text)
		VALUES (
			$1,
			'did:example:author2',
			'Our collective focuses on electronic music production and performance.'
		)
		RETURNING id
	`, sceneID).Scan(&postID2)
	if err != nil {
		t.Fatalf("failed to insert test post 2: %v", err)
	}
	defer func() {
		_, _ = db.Exec("DELETE FROM posts WHERE id = $1", postID2)
	}()

	// Search for "electronics" (plural) should match "electronic" (singular)
	err = db.QueryRow(`
		SELECT COUNT(*) FROM posts
		WHERE to_tsvector_immutable(COALESCE(text, ''))
		@@ to_tsquery('english', 'electronics')
		AND id = $1
	`, postID2).Scan(&count)
	if err != nil {
		t.Fatalf("failed to search for 'electronics': %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 result when searching 'electronics' to match 'electronic', got %d", count)
	}

	// Test 3: Verify stemming with verb forms - "warehousing" should match "warehouse"
	var postID3 string
	err = db.QueryRow(`
		INSERT INTO posts (scene_id, author_did, text)
		VALUES (
			$1,
			'did:example:author3',
			'We are warehousing our equipment at the venue for the weekend.'
		)
		RETURNING id
	`, sceneID).Scan(&postID3)
	if err != nil {
		t.Fatalf("failed to insert test post 3: %v", err)
	}
	defer func() {
		_, _ = db.Exec("DELETE FROM posts WHERE id = $1", postID3)
	}()

	// Search for "warehouse" should match "warehousing" (verb form)
	err = db.QueryRow(`
		SELECT COUNT(*) FROM posts
		WHERE to_tsvector_immutable(COALESCE(text, ''))
		@@ to_tsquery('english', 'warehouse')
		AND id = $1
	`, postID3).Scan(&count)
	if err != nil {
		t.Fatalf("failed to search for 'warehouse' matching 'warehousing': %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 result when searching 'warehouse' to match 'warehousing', got %d", count)
	}
}
