//go:build integration

package testutil

import (
	"database/sql"
	"testing"
	"time"
)

// SceneFixture holds parameters for inserting a test scene.
type SceneFixture struct {
	ID           string
	OwnerDID     string
	Name         string
	Description  string
	Geohash      string
	AllowPrecise bool
	Lng          *float64
	Lat          *float64
}

// DefaultScene returns a SceneFixture with sensible defaults.
func DefaultScene() SceneFixture {
	return SceneFixture{
		ID:          "test-scene-default",
		OwnerDID:    "did:plc:testowner",
		Name:        "Test Scene",
		Description: "A test scene for integration tests",
		Geohash:     "u4pruydqqvj",
	}
}

// InsertScene inserts a scene into the database. Returns the scene ID.
func InsertScene(t *testing.T, db *sql.DB, f SceneFixture) string {
	t.Helper()
	if f.AllowPrecise && f.Lng != nil && f.Lat != nil {
		_, err := db.Exec(`
			INSERT INTO scenes (id, owner_did, name, description, geohash, allow_precise, precise_point)
			VALUES ($1, $2, $3, $4, $5, TRUE, ST_SetSRID(ST_MakePoint($6, $7), 4326)::geography)
		`, f.ID, f.OwnerDID, f.Name, f.Description, f.Geohash, *f.Lng, *f.Lat)
		if err != nil {
			t.Fatalf("InsertScene(%s): %v", f.ID, err)
		}
	} else {
		_, err := db.Exec(`
			INSERT INTO scenes (id, owner_did, name, description, geohash, allow_precise)
			VALUES ($1, $2, $3, $4, $5, FALSE)
		`, f.ID, f.OwnerDID, f.Name, f.Description, f.Geohash)
		if err != nil {
			t.Fatalf("InsertScene(%s): %v", f.ID, err)
		}
	}
	return f.ID
}

// StreamSessionFixture holds parameters for inserting a test stream session.
type StreamSessionFixture struct {
	ID       string
	SceneID  string
	RoomName string
	HostDID  string
}

// InsertStreamSession inserts a stream session into the database. Returns the session ID.
func InsertStreamSession(t *testing.T, db *sql.DB, f StreamSessionFixture) string {
	t.Helper()
	_, err := db.Exec(`
		INSERT INTO stream_sessions (id, scene_id, room_name, host_did)
		VALUES ($1, $2, $3, $4)
	`, f.ID, f.SceneID, f.RoomName, f.HostDID)
	if err != nil {
		t.Fatalf("InsertStreamSession(%s): %v", f.ID, err)
	}
	return f.ID
}

// EventFixture holds parameters for inserting a test event.
type EventFixture struct {
	ID       string
	SceneID  string
	Title    string
	StartsAt time.Time
	EndsAt   time.Time
}

// InsertEvent inserts an event into the database. Returns the event ID.
func InsertEvent(t *testing.T, db *sql.DB, f EventFixture) string {
	t.Helper()
	_, err := db.Exec(`
		INSERT INTO events (id, scene_id, title, starts_at, ends_at)
		VALUES ($1, $2, $3, $4, $5)
	`, f.ID, f.SceneID, f.Title, f.StartsAt, f.EndsAt)
	if err != nil {
		t.Fatalf("InsertEvent(%s): %v", f.ID, err)
	}
	return f.ID
}
