package backfill

import (
"context"
"database/sql"
"fmt"
"log/slog"
"time"
)

// Config holds backfill operation configuration.
type Config struct {
	Source    string
	StartTS  int64
	EndTS    int64
	CARPath  string
	BatchSize int
	DryRun   bool
	Resume   bool
	Logger   *slog.Logger
}

// Result contains the outcome of a backfill run.
type Result struct {
	RecordsProcessed int64
	RecordsSkipped   int64
	Errors           int64
	Duration         time.Duration
}

// Checkpoint tracks backfill progress for resumability.
type Checkpoint struct {
	ID               int64
	Source           string
	CursorTS         int64
	CAROffset        int64
	Status           string
	RecordsProcessed int64
	RecordsSkipped   int64
	ErrorsCount      int64
	StartedAt        *time.Time
	UpdatedAt        time.Time
	CompletedAt      *time.Time
}

// CheckpointStore persists backfill progress.
type CheckpointStore interface {
	GetLatest(ctx context.Context, source string) (*Checkpoint, error)
	Create(ctx context.Context, source string) (int64, error)
	Update(ctx context.Context, cp *Checkpoint) error
	Complete(ctx context.Context, id int64, processed, skipped, errors int64) error
	Fail(ctx context.Context, id int64, processed, skipped, errors int64) error
}

// PostgresCheckpointStore implements CheckpointStore with the backfill_checkpoints table.
type PostgresCheckpointStore struct {
	db     *sql.DB
	logger *slog.Logger
}

// NewPostgresCheckpointStore creates a new checkpoint store.
func NewPostgresCheckpointStore(db *sql.DB, logger *slog.Logger) *PostgresCheckpointStore {
	if logger == nil {
		logger = slog.Default()
	}
	return &PostgresCheckpointStore{db: db, logger: logger}
}

func (s *PostgresCheckpointStore) GetLatest(ctx context.Context, source string) (*Checkpoint, error) {
	query := `SELECT id, source, cursor_ts, car_offset, status,
		records_processed, records_skipped, errors_count,
		started_at, updated_at, completed_at
		FROM backfill_checkpoints
		WHERE source = $1
		ORDER BY id DESC LIMIT 1`
	var cp Checkpoint
	err := s.db.QueryRowContext(ctx, query, source).Scan(
&cp.ID, &cp.Source, &cp.CursorTS, &cp.CAROffset, &cp.Status,
		&cp.RecordsProcessed, &cp.RecordsSkipped, &cp.ErrorsCount,
		&cp.StartedAt, &cp.UpdatedAt, &cp.CompletedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get latest checkpoint: %w", err)
	}
	return &cp, nil
}

func (s *PostgresCheckpointStore) Create(ctx context.Context, source string) (int64, error) {
	now := time.Now()
	var id int64
	err := s.db.QueryRowContext(ctx,
`INSERT INTO backfill_checkpoints (source, status, started_at, updated_at)
		VALUES ($1, 'running', $2, $2) RETURNING id`,
source, now,
).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("create checkpoint: %w", err)
	}
	return id, nil
}

func (s *PostgresCheckpointStore) Update(ctx context.Context, cp *Checkpoint) error {
	_, err := s.db.ExecContext(ctx,
`UPDATE backfill_checkpoints
		SET cursor_ts = $2, car_offset = $3,
			records_processed = $4, records_skipped = $5, errors_count = $6,
			updated_at = NOW()
		WHERE id = $1`,
cp.ID, cp.CursorTS, cp.CAROffset,
cp.RecordsProcessed, cp.RecordsSkipped, cp.ErrorsCount,
)
	if err != nil {
		return fmt.Errorf("update checkpoint: %w", err)
	}
	return nil
}

func (s *PostgresCheckpointStore) Complete(ctx context.Context, id int64, processed, skipped, errors int64) error {
	_, err := s.db.ExecContext(ctx,
`UPDATE backfill_checkpoints
		SET status = 'completed', records_processed = $2, records_skipped = $3,
			errors_count = $4, completed_at = NOW(), updated_at = NOW()
		WHERE id = $1`,
id, processed, skipped, errors,
)
	if err != nil {
		return fmt.Errorf("complete checkpoint: %w", err)
	}
	return nil
}

func (s *PostgresCheckpointStore) Fail(ctx context.Context, id int64, processed, skipped, errors int64) error {
	_, err := s.db.ExecContext(ctx,
`UPDATE backfill_checkpoints
		SET status = 'failed', records_processed = $2, records_skipped = $3,
			errors_count = $4, updated_at = NOW()
		WHERE id = $1`,
id, processed, skipped, errors,
)
	if err != nil {
		return fmt.Errorf("fail checkpoint: %w", err)
	}
	return nil
}
