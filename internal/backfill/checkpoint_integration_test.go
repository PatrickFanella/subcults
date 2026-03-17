//go:build integration

package backfill_test

import (
	"context"
	"log/slog"
	"testing"

	"github.com/onnwee/subcults/internal/backfill"
	"github.com/onnwee/subcults/internal/testutil"
)

func TestPostgresCheckpointStore_CreateAndGetLatest(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	store := backfill.NewPostgresCheckpointStore(tdb.DB, slog.Default())
	ctx := context.Background()

	id, err := store.Create(ctx, "jetstream")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if id == 0 {
		t.Fatal("expected non-zero checkpoint ID")
	}

	cp, err := store.GetLatest(ctx, "jetstream")
	if err != nil {
		t.Fatalf("GetLatest: %v", err)
	}
	if cp == nil {
		t.Fatal("expected checkpoint, got nil")
	}
	if cp.ID != id {
		t.Errorf("ID mismatch: got %d, want %d", cp.ID, id)
	}
	if cp.Source != "jetstream" {
		t.Errorf("Source mismatch: got %q, want %q", cp.Source, "jetstream")
	}
	if cp.Status != "running" {
		t.Errorf("Status mismatch: got %q, want %q", cp.Status, "running")
	}
}

func TestPostgresCheckpointStore_GetLatestReturnsNilWhenEmpty(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	store := backfill.NewPostgresCheckpointStore(tdb.DB, slog.Default())
	ctx := context.Background()

	cp, err := store.GetLatest(ctx, "nonexistent-source")
	if err != nil {
		t.Fatalf("GetLatest: %v", err)
	}
	if cp != nil {
		t.Errorf("expected nil checkpoint, got %+v", cp)
	}
}

func TestPostgresCheckpointStore_Update(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	store := backfill.NewPostgresCheckpointStore(tdb.DB, slog.Default())
	ctx := context.Background()

	id, err := store.Create(ctx, "car")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	cp := &backfill.Checkpoint{
		ID:               id,
		CursorTS:         1234567890,
		CAROffset:        4096,
		RecordsProcessed: 100,
		RecordsSkipped:   5,
		ErrorsCount:      2,
	}
	if err := store.Update(ctx, cp); err != nil {
		t.Fatalf("Update: %v", err)
	}

	got, err := store.GetLatest(ctx, "car")
	if err != nil {
		t.Fatalf("GetLatest: %v", err)
	}
	if got.CursorTS != 1234567890 {
		t.Errorf("CursorTS: got %d, want 1234567890", got.CursorTS)
	}
	if got.CAROffset != 4096 {
		t.Errorf("CAROffset: got %d, want 4096", got.CAROffset)
	}
	if got.RecordsProcessed != 100 {
		t.Errorf("RecordsProcessed: got %d, want 100", got.RecordsProcessed)
	}
}

func TestPostgresCheckpointStore_Complete(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	store := backfill.NewPostgresCheckpointStore(tdb.DB, slog.Default())
	ctx := context.Background()

	id, err := store.Create(ctx, "jetstream")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := store.Complete(ctx, id, 500, 10, 3); err != nil {
		t.Fatalf("Complete: %v", err)
	}

	cp, err := store.GetLatest(ctx, "jetstream")
	if err != nil {
		t.Fatalf("GetLatest: %v", err)
	}
	if cp.Status != "completed" {
		t.Errorf("Status: got %q, want %q", cp.Status, "completed")
	}
	if cp.RecordsProcessed != 500 {
		t.Errorf("RecordsProcessed: got %d, want 500", cp.RecordsProcessed)
	}
	if cp.CompletedAt == nil {
		t.Error("expected CompletedAt to be set")
	}
}

func TestPostgresCheckpointStore_Fail(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	store := backfill.NewPostgresCheckpointStore(tdb.DB, slog.Default())
	ctx := context.Background()

	id, err := store.Create(ctx, "jetstream")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := store.Fail(ctx, id, 200, 5, 50); err != nil {
		t.Fatalf("Fail: %v", err)
	}

	cp, err := store.GetLatest(ctx, "jetstream")
	if err != nil {
		t.Fatalf("GetLatest: %v", err)
	}
	if cp.Status != "failed" {
		t.Errorf("Status: got %q, want %q", cp.Status, "failed")
	}
	if cp.ErrorsCount != 50 {
		t.Errorf("ErrorsCount: got %d, want 50", cp.ErrorsCount)
	}
}

func TestPostgresCheckpointStore_GetLatestReturnsNewest(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	store := backfill.NewPostgresCheckpointStore(tdb.DB, slog.Default())
	ctx := context.Background()

	// Create two checkpoints for the same source
	id1, err := store.Create(ctx, "jetstream")
	if err != nil {
		t.Fatalf("Create first: %v", err)
	}
	if err := store.Complete(ctx, id1, 100, 0, 0); err != nil {
		t.Fatalf("Complete first: %v", err)
	}

	id2, err := store.Create(ctx, "jetstream")
	if err != nil {
		t.Fatalf("Create second: %v", err)
	}

	cp, err := store.GetLatest(ctx, "jetstream")
	if err != nil {
		t.Fatalf("GetLatest: %v", err)
	}
	if cp.ID != id2 {
		t.Errorf("expected latest checkpoint ID %d, got %d", id2, cp.ID)
	}
	if cp.Status != "running" {
		t.Errorf("expected status 'running', got %q", cp.Status)
	}
}
