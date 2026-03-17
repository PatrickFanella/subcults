# Upsert Operations and Statistics

This package provides utilities for tracking statistics during upsert operations in the ingestion pipeline.

## Usage Example

```go
package main

import (
    "log/slog"
    "os"
    
    "github.com/onnwee/subcults/internal/post"
    "github.com/onnwee/subcults/internal/stats"
)

func main() {
    logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
    
    // Create repository and stats tracker
    repo := post.NewInMemoryPostRepository()
    postStats := stats.NewUpsertStats()
    
    // Process records from ingestion
    did := "did:example:alice"
    rkey := "post123"
    
    // First ingestion - insert
    p1 := &post.Post{
        AuthorDID:  did,
        Text:       "Hello world",
        RecordDID:  &did,
        RecordRKey: &rkey,
    }
    
    result, err := repo.Upsert(p1)
    if err != nil {
        logger.Error("upsert failed", "error", err)
        return
    }
    
    // Track the result
    if result.Inserted {
        postStats.RecordInsert()
    } else {
        postStats.RecordUpdate()
    }
    
    // Second ingestion - update (same record key)
    p2 := &post.Post{
        AuthorDID:  did,
        Text:       "Updated text",
        RecordDID:  &did,
        RecordRKey: &rkey,
    }
    
    result, err = repo.Upsert(p2)
    if err != nil {
        logger.Error("upsert failed", "error", err)
        return
    }
    
    if result.Inserted {
        postStats.RecordInsert()
    } else {
        postStats.RecordUpdate()
    }
    
    // Log summary statistics
    postStats.LogSummary(logger, "posts")
    // Output: {"time":"...","level":"INFO","msg":"upsert statistics","entity":"posts","inserted":1,"updated":1,"total":2}
}
```

## Thread Safety

All stats operations are thread-safe using atomic operations. You can safely use the same `UpsertStats` instance across multiple goroutines:

```go
var wg sync.WaitGroup
stats := stats.NewUpsertStats()

for i := 0; i < 100; i++ {
    wg.Add(1)
    go func() {
        defer wg.Done()
        result, _ := repo.Upsert(record)
        if result.Inserted {
            stats.RecordInsert()
        } else {
            stats.RecordUpdate()
        }
    }()
}

wg.Wait()
stats.LogSummary(logger, "records")
```

## Integration with Indexer

The stats package is designed to be used in the Jetstream indexer to track ingestion progress:

```go
type Ingester struct {
    sceneRepo      scene.SceneRepository
    eventRepo      scene.EventRepository
    postRepo       post.PostRepository
    membershipRepo membership.MembershipRepository
    allianceRepo   alliance.AllianceRepository
    streamRepo     stream.SessionRepository
    
    sceneStats      *stats.UpsertStats
    eventStats      *stats.UpsertStats
    postStats       *stats.UpsertStats
    membershipStats *stats.UpsertStats
    allianceStats   *stats.UpsertStats
    streamStats     *stats.UpsertStats
    
    logger *slog.Logger
}

func (i *Ingester) LogStats() {
    i.sceneStats.LogSummary(i.logger, "scenes")
    i.eventStats.LogSummary(i.logger, "events")
    i.postStats.LogSummary(i.logger, "posts")
    i.membershipStats.LogSummary(i.logger, "memberships")
    i.allianceStats.LogSummary(i.logger, "alliances")
    i.streamStats.LogSummary(i.logger, "streams")
}
```

## Periodic Reporting

For long-running ingestion processes, you can periodically log and reset statistics:

```go
ticker := time.NewTicker(1 * time.Minute)
defer ticker.Stop()

for {
    select {
    case <-ticker.C:
        stats.LogSummary(logger, "posts")
        stats.Reset()
    case <-ctx.Done():
        // Final summary before shutdown
        stats.LogSummary(logger, "posts")
        return
    }
}
```
