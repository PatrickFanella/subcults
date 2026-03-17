package backfill

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"sync"

	"github.com/onnwee/subcults/internal/indexer"
)

// InMemoryCheckpointStore implements CheckpointStore for testing.
type InMemoryCheckpointStore struct {
	mu          sync.Mutex
	checkpoints map[int64]*Checkpoint
	nextID      int64
}

func newInMemoryCheckpointStore() *InMemoryCheckpointStore {
	return &InMemoryCheckpointStore{
		checkpoints: make(map[int64]*Checkpoint),
	}
}

func (s *InMemoryCheckpointStore) GetLatest(ctx context.Context, source string) (*Checkpoint, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var latest *Checkpoint
	for _, cp := range s.checkpoints {
		if cp.Source == source {
			if latest == nil || cp.ID > latest.ID {
				latest = cp
			}
		}
	}
	return latest, nil
}

func (s *InMemoryCheckpointStore) Create(ctx context.Context, source string) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nextID++
	s.checkpoints[s.nextID] = &Checkpoint{
		ID:     s.nextID,
		Source: source,
		Status: "running",
	}
	return s.nextID, nil
}

func (s *InMemoryCheckpointStore) Update(ctx context.Context, cp *Checkpoint) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	existing, ok := s.checkpoints[cp.ID]
	if !ok {
		return fmt.Errorf("checkpoint %d not found", cp.ID)
	}
	existing.CursorTS = cp.CursorTS
	existing.CAROffset = cp.CAROffset
	existing.RecordsProcessed = cp.RecordsProcessed
	existing.RecordsSkipped = cp.RecordsSkipped
	existing.ErrorsCount = cp.ErrorsCount
	return nil
}

func (s *InMemoryCheckpointStore) Complete(ctx context.Context, id int64, processed, skipped, errors int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp, ok := s.checkpoints[id]
	if !ok {
		return fmt.Errorf("checkpoint %d not found", id)
	}
	cp.Status = "completed"
	cp.RecordsProcessed = processed
	cp.RecordsSkipped = skipped
	cp.ErrorsCount = errors
	return nil
}

func (s *InMemoryCheckpointStore) Fail(ctx context.Context, id int64, processed, skipped, errors int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp, ok := s.checkpoints[id]
	if !ok {
		return fmt.Errorf("checkpoint %d not found", id)
	}
	cp.Status = "failed"
	cp.RecordsProcessed = processed
	cp.RecordsSkipped = skipped
	cp.ErrorsCount = errors
	return nil
}

func (s *InMemoryCheckpointStore) get(id int64) *Checkpoint {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.checkpoints[id]
}

func newTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func newTestFilter() *indexer.RecordFilter {
	return indexer.NewRecordFilter(indexer.NewFilterMetrics())
}

func newTestRepo() *indexer.InMemoryRecordRepository {
	return indexer.NewInMemoryRecordRepository(newTestLogger())
}
