package retention

import (
	"context"
	"log/slog"
	"testing"
	"time"
)

func TestService_RunCycle_DeletesExpiredRecords(t *testing.T) {
	repo := NewInMemoryRepository(slog.Default())
	repo.SetExpiredCount("posts", 50)
	repo.SetExpiredCount("audit_logs", 10)

	svc := NewService(repo, ServiceConfig{
		Tiers:     DefaultTiers(),
		BatchSize: 1000,
		Logger:    slog.Default(),
	})

	svc.runCycle(context.Background())

	if got := repo.GetDeletedCount("posts"); got != 50 {
		t.Errorf("expected 50 deleted posts, got %d", got)
	}
	if got := repo.GetDeletedCount("audit_logs"); got != 10 {
		t.Errorf("expected 10 deleted audit_logs, got %d", got)
	}
}

func TestService_RunCycle_ArchivesBeforeDelete(t *testing.T) {
	repo := NewInMemoryRepository(slog.Default())
	repo.SetExpiredCount("scenes", 20)

	svc := NewService(repo, ServiceConfig{
		Tiers:     DefaultTiers(),
		BatchSize: 1000,
		Logger:    slog.Default(),
	})

	svc.runCycle(context.Background())

	if got := repo.GetArchivedCount("scenes"); got != 20 {
		t.Errorf("expected 20 archived scenes, got %d", got)
	}
	if got := repo.GetDeletedCount("scenes"); got != 20 {
		t.Errorf("expected 20 deleted scenes, got %d", got)
	}
}

func TestService_ProcessAccountDeletions_PastGrace(t *testing.T) {
	repo := NewInMemoryRepository(slog.Default())
	repo.AddPendingDeletion(PendingDeletion{
		UserDID:     "did:plc:test123",
		ScheduledAt: time.Now().Add(-31 * 24 * time.Hour),
		GraceEndsAt: time.Now().Add(-1 * time.Hour),
	})

	svc := NewService(repo, ServiceConfig{
		Logger: slog.Default(),
	})

	svc.runCycle(context.Background())

	deleted := repo.DeletedAccounts()
	if len(deleted) != 1 || deleted[0] != "did:plc:test123" {
		t.Errorf("expected account deletion, got %v", deleted)
	}
}

func TestService_ProcessAccountDeletions_WithinGrace(t *testing.T) {
	repo := NewInMemoryRepository(slog.Default())
	repo.AddPendingDeletion(PendingDeletion{
		UserDID:     "did:plc:test456",
		ScheduledAt: time.Now(),
		GraceEndsAt: time.Now().Add(29 * 24 * time.Hour),
	})

	svc := NewService(repo, ServiceConfig{
		Logger: slog.Default(),
	})

	svc.runCycle(context.Background())

	deleted := repo.DeletedAccounts()
	if len(deleted) != 0 {
		t.Errorf("expected no deletions during grace period, got %v", deleted)
	}
}

func TestService_BatchSizeRespected(t *testing.T) {
	repo := NewInMemoryRepository(slog.Default())
	repo.SetExpiredCount("posts", 5000)

	svc := NewService(repo, ServiceConfig{
		Tiers: []RetentionTier{
			{EntityType: "posts", RetentionPeriod: time.Hour, ArchiveFirst: false},
		},
		BatchSize: 100,
		Logger:    slog.Default(),
	})

	svc.runCycle(context.Background())

	if got := repo.GetDeletedCount("posts"); got != 100 {
		t.Errorf("expected batch-limited 100 deletions, got %d", got)
	}
}

func TestDefaultTiers(t *testing.T) {
	tiers := DefaultTiers()
	if len(tiers) == 0 {
		t.Fatal("expected non-empty default tiers")
	}

	found := false
	for _, tier := range tiers {
		if tier.EntityType == "scenes" {
			found = true
			if !tier.ArchiveFirst {
				t.Error("scenes should archive before delete")
			}
			if tier.RetentionPeriod < 365*24*time.Hour {
				t.Error("scenes retention should be >= 1 year")
			}
		}
	}
	if !found {
		t.Error("expected 'scenes' in default tiers")
	}
}

func TestInMemoryRepository_ExportUserData(t *testing.T) {
	repo := NewInMemoryRepository(slog.Default())
	export := &UserDataExport{
		UserDID: "did:plc:user1",
		Scenes:  []map[string]interface{}{{"name": "test"}},
	}
	repo.AddUserExport("did:plc:user1", export)

	got, err := repo.ExportUserData(context.Background(), "did:plc:user1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.UserDID != "did:plc:user1" {
		t.Errorf("expected user DID 'did:plc:user1', got %q", got.UserDID)
	}
	if len(got.Scenes) != 1 {
		t.Errorf("expected 1 scene, got %d", len(got.Scenes))
	}
}

func TestInMemoryRepository_ExportUserData_NotFound(t *testing.T) {
	repo := NewInMemoryRepository(slog.Default())

	_, err := repo.ExportUserData(context.Background(), "did:plc:nonexistent")
	if err == nil {
		t.Error("expected error for non-existent user")
	}
}

func TestInMemoryRepository_ScheduleAndExecuteDeletion(t *testing.T) {
	repo := NewInMemoryRepository(slog.Default())
	ctx := context.Background()

	graceEnd := time.Now().Add(30 * 24 * time.Hour)
	if err := repo.ScheduleAccountDeletion(ctx, "did:plc:del1", graceEnd); err != nil {
		t.Fatalf("ScheduleAccountDeletion: %v", err)
	}

	pending, err := repo.GetPendingDeletions(ctx)
	if err != nil {
		t.Fatalf("GetPendingDeletions: %v", err)
	}
	if len(pending) != 1 {
		t.Fatalf("expected 1 pending deletion, got %d", len(pending))
	}

	if err := repo.ExecuteAccountDeletion(ctx, "did:plc:del1"); err != nil {
		t.Fatalf("ExecuteAccountDeletion: %v", err)
	}

	deleted := repo.DeletedAccounts()
	if len(deleted) != 1 || deleted[0] != "did:plc:del1" {
		t.Errorf("expected deleted account, got %v", deleted)
	}

	pending, _ = repo.GetPendingDeletions(ctx)
	if len(pending) != 0 {
		t.Errorf("expected 0 pending after execution, got %d", len(pending))
	}
}
