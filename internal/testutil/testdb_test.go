//go:build integration

package testutil

import (
	"database/sql"
	"testing"
)

func TestNewTestDB_MigrationsApplied(t *testing.T) {
	tdb := NewTestDB(t)

	// Verify core tables exist
	tables := []string{
		"scenes", "events", "posts", "memberships", "alliances",
		"stream_sessions", "payment_records", "audit_logs",
		"telemetry_events", "client_error_logs",
	}

	for _, table := range tables {
		var exists bool
		err := tdb.DB.QueryRow(`
			SELECT EXISTS (
				SELECT 1 FROM information_schema.tables
				WHERE table_schema = 'public' AND table_name = $1
			)`, table).Scan(&exists)
		if err != nil {
			t.Fatalf("checking table %s: %v", table, err)
		}
		if !exists {
			t.Errorf("expected table %q to exist", table)
		}
	}
}

func TestNewTestDB_PostGISEnabled(t *testing.T) {
	tdb := NewTestDB(t)

	var version string
	err := tdb.DB.QueryRow("SELECT PostGIS_Version()").Scan(&version)
	if err != nil {
		t.Fatalf("PostGIS not enabled: %v", err)
	}
	t.Logf("PostGIS version: %s", version)
}

func TestNewTestDB_ScenePrivacyConstraint(t *testing.T) {
	tdb := NewTestDB(t)

	// allow_precise defaults to FALSE
	var allowPrecise bool
	err := tdb.DB.QueryRow(`
		SELECT column_default = 'false'
		FROM information_schema.columns
		WHERE table_name = 'scenes' AND column_name = 'allow_precise'
	`).Scan(&allowPrecise)
	if err != nil {
		t.Fatalf("checking allow_precise default: %v", err)
	}
	if !allowPrecise {
		t.Error("expected allow_precise to default to FALSE")
	}
}

func TestNewTestDB_PaymentStatusConstraint(t *testing.T) {
	tdb := NewTestDB(t)

	// Verify payment_records status CHECK constraint exists
	var constraintExists bool
	err := tdb.DB.QueryRow(`
		SELECT EXISTS (
			SELECT 1 FROM information_schema.check_constraints cc
			JOIN information_schema.constraint_column_usage ccu
				ON cc.constraint_name = ccu.constraint_name
			WHERE ccu.table_name = 'payment_records'
				AND ccu.column_name = 'status'
		)
	`).Scan(&constraintExists)
	if err != nil {
		t.Fatalf("checking payment status constraint: %v", err)
	}
	if !constraintExists {
		t.Error("expected CHECK constraint on payment_records.status")
	}
}

func TestNewTestDB_SceneInsertWithGeo(t *testing.T) {
	tdb := NewTestDB(t)

	tdb.ExecTx(t, func(tx *sql.Tx) {
		// Insert a scene with a geographic point
		_, err := tx.Exec(`
			INSERT INTO scenes (id, owner_did, name, description, genre, geohash, allow_precise, precise_point)
			VALUES ($1, $2, $3, $4, $5, $6, TRUE, ST_SetSRID(ST_MakePoint($7, $8), 4326)::geography)
		`, "test-scene-1", "did:plc:test123", "Test Scene", "A test scene", "electronic",
			"u4pruydqqvj", -73.935242, 40.730610)
		if err != nil {
			t.Fatalf("inserting scene with geo: %v", err)
		}

		// Verify we can query by proximity
		var count int
		err = tx.QueryRow(`
			SELECT COUNT(*) FROM scenes
			WHERE ST_DWithin(
				precise_point,
				ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography,
				1000
			)
		`, -73.935242, 40.730610).Scan(&count)
		if err != nil {
			t.Fatalf("querying by proximity: %v", err)
		}
		if count != 1 {
			t.Errorf("expected 1 scene within 1km, got %d", count)
		}
	})
}

func TestNewTestDB_ScenePrivacyEnforcement(t *testing.T) {
	tdb := NewTestDB(t)

	tdb.ExecTx(t, func(tx *sql.Tx) {
		// allow_precise=FALSE with precise_point should be rejected by CHECK
		_, err := tx.Exec(`
			INSERT INTO scenes (id, owner_did, name, description, genre, geohash, allow_precise, precise_point)
			VALUES ($1, $2, $3, $4, $5, $6, FALSE, ST_SetSRID(ST_MakePoint($7, $8), 4326)::geography)
		`, "test-scene-privacy", "did:plc:test456", "Privacy Test", "desc", "electronic",
			"u4pruydqqvj", -73.935242, 40.730610)
		if err == nil {
			t.Error("expected CHECK constraint violation: allow_precise=FALSE with precise_point set")
		}
	})
}

func TestNewTestDB_ClientErrorDedup(t *testing.T) {
	tdb := NewTestDB(t)

	tdb.ExecTx(t, func(tx *sql.Tx) {
		// Insert first error
		_, err := tx.Exec(`
			INSERT INTO client_error_logs (id, session_id, error_type, error_message, error_hash, occurred_at)
			VALUES ($1, $2, $3, $4, $5, NOW())
		`, "err-1", "sess-1", "TypeError", "null ref", "hash-abc")
		if err != nil {
			t.Fatalf("inserting first error: %v", err)
		}

		// Same error_hash + session_id should fail (UNIQUE constraint)
		_, err = tx.Exec(`
			INSERT INTO client_error_logs (id, session_id, error_type, error_message, error_hash, occurred_at)
			VALUES ($1, $2, $3, $4, $5, NOW())
		`, "err-2", "sess-1", "TypeError", "null ref", "hash-abc")
		if err == nil {
			t.Error("expected UNIQUE constraint violation on (error_hash, session_id)")
		}

		// Different session should succeed
		_, err = tx.Exec(`
			INSERT INTO client_error_logs (id, session_id, error_type, error_message, error_hash, occurred_at)
			VALUES ($1, $2, $3, $4, $5, NOW())
		`, "err-3", "sess-2", "TypeError", "null ref", "hash-abc")
		if err != nil {
			t.Fatalf("different session should succeed: %v", err)
		}
	})
}
