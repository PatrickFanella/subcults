package db

import (
	"context"
	"log/slog"
	"testing"
)

func TestInMemory_GetCurrentVersion_Empty(t *testing.T) {
	checker := NewInMemorySchemaVersionChecker(slog.Default())

	info, err := checker.GetCurrentVersion(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.Version != 0 {
		t.Errorf("expected version 0 for empty store, got %d", info.Version)
	}
}

func TestInMemory_RecordAndGetVersion(t *testing.T) {
	checker := NewInMemorySchemaVersionChecker(slog.Default())
	ctx := context.Background()

	if err := checker.RecordVersion(ctx, 28, "schema_version tracking"); err != nil {
		t.Fatalf("RecordVersion: %v", err)
	}

	info, err := checker.GetCurrentVersion(ctx)
	if err != nil {
		t.Fatalf("GetCurrentVersion: %v", err)
	}
	if info.Version != 28 {
		t.Errorf("expected version 28, got %d", info.Version)
	}
	if info.Description != "schema_version tracking" {
		t.Errorf("expected description 'schema_version tracking', got %q", info.Description)
	}
}

func TestInMemory_EnsureCompatible_OK(t *testing.T) {
	checker := NewInMemorySchemaVersionChecker(slog.Default())
	ctx := context.Background()

	if err := checker.RecordVersion(ctx, MinSchemaVersion, "test"); err != nil {
		t.Fatal(err)
	}

	if err := checker.EnsureCompatible(ctx); err != nil {
		t.Errorf("expected compatible, got error: %v", err)
	}
}

func TestInMemory_EnsureCompatible_TooOld(t *testing.T) {
	checker := NewInMemorySchemaVersionChecker(slog.Default())
	ctx := context.Background()

	if err := checker.RecordVersion(ctx, MinSchemaVersion-1, "old"); err != nil {
		t.Fatal(err)
	}

	err := checker.EnsureCompatible(ctx)
	if err == nil {
		t.Error("expected error for old schema, got nil")
	}
}

func TestInMemory_EnsureCompatible_NoVersions(t *testing.T) {
	checker := NewInMemorySchemaVersionChecker(slog.Default())
	ctx := context.Background()

	err := checker.EnsureCompatible(ctx)
	if err == nil {
		t.Error("expected error for empty schema_version, got nil")
	}
}

func TestInMemory_GetCurrentVersion_ReturnsLatest(t *testing.T) {
	checker := NewInMemorySchemaVersionChecker(slog.Default())
	ctx := context.Background()

	for i := 1; i <= 30; i++ {
		if err := checker.RecordVersion(ctx, i, ""); err != nil {
			t.Fatal(err)
		}
	}

	info, err := checker.GetCurrentVersion(ctx)
	if err != nil {
		t.Fatalf("GetCurrentVersion: %v", err)
	}
	if info.Version != 30 {
		t.Errorf("expected version 30 (latest), got %d", info.Version)
	}
}

func TestNewSchemaVersionStore_NilDB(t *testing.T) {
	store := NewSchemaVersionStore(nil, slog.Default())
	if _, ok := store.(*InMemorySchemaVersionChecker); !ok {
		t.Error("expected InMemorySchemaVersionChecker when db is nil")
	}
}
