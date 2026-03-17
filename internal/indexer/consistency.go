package indexer

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"time"
)

// ConsistencyCheckResult represents the outcome of a consistency verification.
type ConsistencyCheckResult struct {
	TotalSampled   int
	Consistent     int
	Inconsistent   int
	Missing        int
	Errors         int
	Duration       time.Duration
	Mismatches     []Mismatch
}

// Score returns the consistency percentage (0.0 to 1.0).
func (r *ConsistencyCheckResult) Score() float64 {
	if r.TotalSampled == 0 {
		return 1.0
	}
	return float64(r.Consistent) / float64(r.TotalSampled)
}

// Mismatch describes a single inconsistency between local and remote records.
type Mismatch struct {
	Collection string
	DID        string
	RKey       string
	Field      string
	LocalVal   string
	RemoteVal  string
}

// LocalRecord represents a record stored in Postgres for comparison.
type LocalRecord struct {
	DID        string
	Collection string
	RKey       string
	Rev        string
	RecordJSON []byte
	UpdatedAt  time.Time
}

// ConsistencyChecker verifies data integrity between local Postgres and AT Protocol source.
type ConsistencyChecker struct {
	db         *sql.DB
	logger     *slog.Logger
	sampleSize int
}

// NewConsistencyChecker creates a new checker.
func NewConsistencyChecker(db *sql.DB, logger *slog.Logger, sampleSize int) *ConsistencyChecker {
	if logger == nil {
		logger = slog.Default()
	}
	if sampleSize <= 0 {
		sampleSize = 1000
	}
	return &ConsistencyChecker{
		db:         db,
		logger:     logger,
		sampleSize: sampleSize,
	}
}

// SampleRecords retrieves a random sample of records from the local database.
func (cc *ConsistencyChecker) SampleRecords(ctx context.Context) ([]LocalRecord, error) {
	if cc.db == nil {
		return nil, nil
	}
	// Use TABLESAMPLE for efficient random sampling, fallback to ORDER BY RANDOM()
	query := `SELECT did, collection, rkey, rev, record_json, updated_at
		FROM indexed_records
		ORDER BY RANDOM()
		LIMIT $1`

	rows, err := cc.db.QueryContext(ctx, query, cc.sampleSize)
	if err != nil {
		return nil, fmt.Errorf("sample records: %w", err)
	}
	defer rows.Close()

	var records []LocalRecord
	for rows.Next() {
		var r LocalRecord
		if err := rows.Scan(&r.DID, &r.Collection, &r.RKey, &r.Rev, &r.RecordJSON, &r.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan record: %w", err)
		}
		records = append(records, r)
	}
	return records, rows.Err()
}

// CompareRecord checks whether a local record matches the remote (source of truth).
// The remote fetcher is injected to support both Jetstream API and mock testing.
type RemoteFetcher func(ctx context.Context, did, collection, rkey string) ([]byte, string, error)

// Check runs consistency verification using the provided remote fetcher for comparison.
func (cc *ConsistencyChecker) Check(ctx context.Context, records []LocalRecord, fetcher RemoteFetcher) *ConsistencyCheckResult {
	start := time.Now()
	result := &ConsistencyCheckResult{}

	for _, local := range records {
		select {
		case <-ctx.Done():
			cc.logger.Info("consistency check cancelled")
			result.Duration = time.Since(start)
			return result
		default:
		}

		result.TotalSampled++
		remoteJSON, remoteRev, err := fetcher(ctx, local.DID, local.Collection, local.RKey)
		if err != nil {
			result.Errors++
			cc.logger.Debug("failed to fetch remote record",
				"did", local.DID, "collection", local.Collection, "rkey", local.RKey, "error", err,
			)
			continue
		}

		if remoteJSON == nil {
			result.Missing++
			result.Mismatches = append(result.Mismatches, Mismatch{
				Collection: local.Collection,
				DID:        local.DID,
				RKey:       local.RKey,
				Field:      "record",
				LocalVal:   "present",
				RemoteVal:  "missing",
			})
			continue
		}

		if local.Rev != remoteRev {
			result.Inconsistent++
			result.Mismatches = append(result.Mismatches, Mismatch{
				Collection: local.Collection,
				DID:        local.DID,
				RKey:       local.RKey,
				Field:      "rev",
				LocalVal:   local.Rev,
				RemoteVal:  remoteRev,
			})
			continue
		}

		result.Consistent++
	}

	result.Duration = time.Since(start)
	cc.logger.Info("consistency check complete",
		"sampled", result.TotalSampled,
		"consistent", result.Consistent,
		"inconsistent", result.Inconsistent,
		"missing", result.Missing,
		"errors", result.Errors,
		"score", fmt.Sprintf("%.2f%%", result.Score()*100),
	)
	return result
}

// MarkForReindex flags inconsistent records for re-indexing via the backfill system.
func (cc *ConsistencyChecker) MarkForReindex(ctx context.Context, mismatches []Mismatch) (int, error) {
	if cc.db == nil || len(mismatches) == 0 {
		return 0, nil
	}
	marked := 0
	for _, m := range mismatches {
		_, err := cc.db.ExecContext(ctx,
			`UPDATE indexed_records SET needs_reindex = TRUE, updated_at = NOW()
			WHERE did = $1 AND collection = $2 AND rkey = $3`,
			m.DID, m.Collection, m.RKey,
		)
		if err != nil {
			cc.logger.Warn("failed to mark for reindex",
				"did", m.DID, "collection", m.Collection, "rkey", m.RKey, "error", err,
			)
			continue
		}
		marked++
	}
	return marked, nil
}

// InMemoryConsistencyChecker is for testing without a database.
type InMemoryConsistencyChecker struct {
	records    []LocalRecord
	logger     *slog.Logger
	sampleSize int
}

// NewInMemoryConsistencyChecker creates a test checker with pre-loaded records.
func NewInMemoryConsistencyChecker(records []LocalRecord, logger *slog.Logger, sampleSize int) *InMemoryConsistencyChecker {
	if logger == nil {
		logger = slog.Default()
	}
	if sampleSize <= 0 {
		sampleSize = 1000
	}
	return &InMemoryConsistencyChecker{
		records:    records,
		logger:     logger,
		sampleSize: sampleSize,
	}
}

// SampleRecords returns a random sample from the in-memory records.
func (ic *InMemoryConsistencyChecker) SampleRecords() []LocalRecord {
	if len(ic.records) <= ic.sampleSize {
		return ic.records
	}
	// Fisher-Yates sample
	sampled := make([]LocalRecord, len(ic.records))
	copy(sampled, ic.records)
	for i := len(sampled) - 1; i > 0; i-- {
		j := rand.IntN(i + 1)
		sampled[i], sampled[j] = sampled[j], sampled[i]
	}
	return sampled[:ic.sampleSize]
}

// Check runs consistency verification using the in-memory records.
func (ic *InMemoryConsistencyChecker) Check(ctx context.Context, fetcher RemoteFetcher) *ConsistencyCheckResult {
	records := ic.SampleRecords()
	checker := &ConsistencyChecker{logger: ic.logger, sampleSize: ic.sampleSize}
	return checker.Check(ctx, records, fetcher)
}
