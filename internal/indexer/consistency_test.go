package indexer

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"testing"
)

func consistencyTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestConsistencyChecker_AllConsistent(t *testing.T) {
	records := []LocalRecord{
		{DID: "did:plc:user1", Collection: CollectionScene, RKey: "abc", Rev: "rev1"},
		{DID: "did:plc:user2", Collection: CollectionEvent, RKey: "def", Rev: "rev2"},
	}
	checker := NewInMemoryConsistencyChecker(records, consistencyTestLogger(), 100)

	fetcher := func(ctx context.Context, did, collection, rkey string) ([]byte, string, error) {
		for _, r := range records {
			if r.DID == did && r.Collection == collection && r.RKey == rkey {
				return []byte(`{}`), r.Rev, nil
			}
		}
		return nil, "", nil
	}

	result := checker.Check(context.Background(), fetcher)
	if result.TotalSampled != 2 {
		t.Errorf("expected 2 sampled, got %d", result.TotalSampled)
	}
	if result.Consistent != 2 {
		t.Errorf("expected 2 consistent, got %d", result.Consistent)
	}
	if result.Score() != 1.0 {
		t.Errorf("expected score 1.0, got %f", result.Score())
	}
}

func TestConsistencyChecker_DetectsRevMismatch(t *testing.T) {
	records := []LocalRecord{
		{DID: "did:plc:user1", Collection: CollectionScene, RKey: "abc", Rev: "rev1"},
	}
	checker := NewInMemoryConsistencyChecker(records, consistencyTestLogger(), 100)

	fetcher := func(ctx context.Context, did, collection, rkey string) ([]byte, string, error) {
		return []byte(`{}`), "rev999", nil // Different rev
	}

	result := checker.Check(context.Background(), fetcher)
	if result.Inconsistent != 1 {
		t.Errorf("expected 1 inconsistent, got %d", result.Inconsistent)
	}
	if len(result.Mismatches) != 1 {
		t.Fatalf("expected 1 mismatch, got %d", len(result.Mismatches))
	}
	if result.Mismatches[0].Field != "rev" {
		t.Errorf("expected field 'rev', got %s", result.Mismatches[0].Field)
	}
}

func TestConsistencyChecker_DetectsMissingRemote(t *testing.T) {
	records := []LocalRecord{
		{DID: "did:plc:user1", Collection: CollectionScene, RKey: "abc", Rev: "rev1"},
	}
	checker := NewInMemoryConsistencyChecker(records, consistencyTestLogger(), 100)

	fetcher := func(ctx context.Context, did, collection, rkey string) ([]byte, string, error) {
		return nil, "", nil // Not found remotely
	}

	result := checker.Check(context.Background(), fetcher)
	if result.Missing != 1 {
		t.Errorf("expected 1 missing, got %d", result.Missing)
	}
}

func TestConsistencyChecker_HandlesErrors(t *testing.T) {
	records := []LocalRecord{
		{DID: "did:plc:user1", Collection: CollectionScene, RKey: "abc", Rev: "rev1"},
	}
	checker := NewInMemoryConsistencyChecker(records, consistencyTestLogger(), 100)

	fetcher := func(ctx context.Context, did, collection, rkey string) ([]byte, string, error) {
		return nil, "", fmt.Errorf("network error")
	}

	result := checker.Check(context.Background(), fetcher)
	if result.Errors != 1 {
		t.Errorf("expected 1 error, got %d", result.Errors)
	}
}

func TestConsistencyChecker_CancelsOnContext(t *testing.T) {
	records := make([]LocalRecord, 100)
	for i := range records {
		records[i] = LocalRecord{DID: fmt.Sprintf("did:plc:user%d", i), Collection: CollectionScene, RKey: fmt.Sprintf("rkey%d", i), Rev: "rev1"}
	}
	checker := NewInMemoryConsistencyChecker(records, consistencyTestLogger(), 100)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	fetcher := func(ctx context.Context, did, collection, rkey string) ([]byte, string, error) {
		return []byte(`{}`), "rev1", nil
	}

	result := checker.Check(ctx, fetcher)
	// Should stop early due to cancelled context
	if result.TotalSampled >= len(records) {
		t.Errorf("expected early termination, but sampled all %d records", result.TotalSampled)
	}
}

func TestConsistencyChecker_EmptyRecords(t *testing.T) {
	checker := NewInMemoryConsistencyChecker(nil, consistencyTestLogger(), 100)

	fetcher := func(ctx context.Context, did, collection, rkey string) ([]byte, string, error) {
		return nil, "", nil
	}

	result := checker.Check(context.Background(), fetcher)
	if result.TotalSampled != 0 {
		t.Errorf("expected 0 sampled, got %d", result.TotalSampled)
	}
	if result.Score() != 1.0 {
		t.Errorf("expected score 1.0 for empty set, got %f", result.Score())
	}
}

func TestConsistencyCheckResult_Score(t *testing.T) {
	tests := []struct {
		name     string
		result   ConsistencyCheckResult
		expected float64
	}{
		{"all consistent", ConsistencyCheckResult{TotalSampled: 10, Consistent: 10}, 1.0},
		{"50% consistent", ConsistencyCheckResult{TotalSampled: 10, Consistent: 5}, 0.5},
		{"none consistent", ConsistencyCheckResult{TotalSampled: 10, Consistent: 0}, 0.0},
		{"empty", ConsistencyCheckResult{TotalSampled: 0, Consistent: 0}, 1.0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.result.Score(); got != tt.expected {
				t.Errorf("Score() = %v, want %v", got, tt.expected)
			}
		})
	}
}
