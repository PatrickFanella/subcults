package backfill

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/onnwee/subcults/internal/indexer"
)

// Runner orchestrates a backfill operation with checkpoint-based resume.
type Runner struct {
	config     Config
	repo       indexer.RecordRepository
	filter     *indexer.RecordFilter
	checkpoint CheckpointStore
	logger     *slog.Logger
}

// NewRunner creates a backfill runner.
func NewRunner(cfg Config, repo indexer.RecordRepository, filter *indexer.RecordFilter, checkpoint CheckpointStore) *Runner {
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 1000
	}
	return &Runner{
		config:     cfg,
		repo:       repo,
		filter:     filter,
		checkpoint: checkpoint,
		logger:     cfg.Logger,
	}
}

// Run executes the backfill operation.
func (r *Runner) Run(ctx context.Context) (*Result, error) {
	start := time.Now()
	switch r.config.Source {
	case "jetstream":
		return r.runJetstream(ctx, start)
	case "car":
		return r.runCAR(ctx, start)
	default:
		return nil, fmt.Errorf("unsupported source: %s", r.config.Source)
	}
}

func (r *Runner) runJetstream(ctx context.Context, start time.Time) (*Result, error) {
	var startTS int64
	if r.config.Resume {
		cp, err := r.checkpoint.GetLatest(ctx, "jetstream")
		if err != nil {
			return nil, fmt.Errorf("failed to get checkpoint: %w", err)
		}
		if cp != nil && cp.Status == "running" {
			startTS = cp.CursorTS
			r.logger.Info("resuming from checkpoint",
				"checkpoint_id", cp.ID,
				"cursor_ts", cp.CursorTS,
			)
		}
	}
	if startTS == 0 {
		startTS = r.config.StartTS
	}
	endTS := r.config.EndTS
	if endTS == 0 {
		endTS = time.Now().UnixMicro()
	}

	cpID, err := r.checkpoint.Create(ctx, "jetstream")
	if err != nil {
		return nil, fmt.Errorf("failed to create checkpoint: %w", err)
	}

	result := &Result{}
	r.logger.Info("starting Jetstream backfill",
		"start_ts", startTS,
		"end_ts", endTS,
		"dry_run", r.config.DryRun,
	)

	// TODO: Connect to Jetstream WebSocket with cursor=startTS
	// Process messages until endTS or context cancelled

	if err := r.checkpoint.Complete(ctx, cpID, result.RecordsProcessed, result.RecordsSkipped, result.Errors); err != nil {
		r.logger.Error("failed to mark checkpoint complete", "error", err)
	}
	result.Duration = time.Since(start)
	return result, nil
}

// ProcessRecord processes a single record through the filter and repository.
func (r *Runner) ProcessRecord(ctx context.Context, collection string, payload []byte, did, rkey, rev string) error {
	filterResult := r.filter.Filter(collection, payload)
	if !filterResult.Matched {
		return nil
	}
	filterResult.DID = did
	filterResult.RKey = rkey
	filterResult.Rev = rev
	filterResult.Operation = "create"
	if !filterResult.Valid {
		return fmt.Errorf("validation failed for %s/%s: %w", collection, rkey, filterResult.Error)
	}
	if r.config.DryRun {
		r.logger.Debug("dry-run: would upsert record",
			"collection", collection, "did", did, "rkey", rkey,
		)
		return nil
	}
	_, _, err := r.repo.UpsertRecord(ctx, &filterResult)
	return err
}

func (r *Runner) runCAR(ctx context.Context, start time.Time) (*Result, error) {
	cpID, err := r.checkpoint.Create(ctx, "car")
	if err != nil {
		return nil, fmt.Errorf("failed to create checkpoint: %w", err)
	}
	result := &Result{}
	r.logger.Info("starting CAR file import",
		"path", r.config.CARPath,
		"dry_run", r.config.DryRun,
	)

	f, err := os.Open(r.config.CARPath)
	if err != nil {
		_ = r.checkpoint.Fail(ctx, cpID, 0, 0, 1)
		return nil, fmt.Errorf("failed to open CAR file: %w", err)
	}
	defer f.Close()

	carReader, err := indexer.NewCARReader(f, r.logger)
	if err != nil {
		_ = r.checkpoint.Fail(ctx, cpID, 0, 0, 1)
		return nil, fmt.Errorf("failed to parse CAR header: %w", err)
	}

	importer := indexer.NewCARImporter(r.repo, r.filter, r.logger)
	importResult, err := importer.Import(ctx, carReader, r.config.DryRun)
	if err != nil {
		result.Errors = importResult.Errors
		_ = r.checkpoint.Fail(ctx, cpID, importResult.RecordsProcessed, importResult.RecordsSkipped, importResult.Errors)
		return nil, fmt.Errorf("CAR import failed: %w", err)
	}

	result.RecordsProcessed = importResult.RecordsProcessed
	result.RecordsSkipped = importResult.RecordsSkipped
	result.Errors = importResult.Errors

	if err := r.checkpoint.Complete(ctx, cpID, result.RecordsProcessed, result.RecordsSkipped, result.Errors); err != nil {
		r.logger.Error("failed to mark checkpoint complete", "error", err)
	}
	result.Duration = time.Since(start)
	return result, nil
}

// JetstreamMessage represents a message from the Jetstream replay stream.
type JetstreamMessage struct {
	DID        string          `json:"did"`
	TimeUS     int64           `json:"time_us"`
	Kind       string          `json:"kind"`
	Collection string          `json:"collection"`
	RKey       string          `json:"rkey"`
	Record     json.RawMessage `json:"record"`
	Rev        string          `json:"rev"`
	Operation  string          `json:"operation"`
}
