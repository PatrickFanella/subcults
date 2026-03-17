# Idempotency Key Cleanup

## Overview

Idempotency keys stored in the database need to be periodically cleaned up to prevent unbounded storage growth. Keys older than 24 hours are considered expired and can be safely deleted.

## Implementation

The cleanup functionality is provided by the `idempotency` package:

```go
import "github.com/onnwee/subcults/internal/idempotency"
```

### Manual Cleanup

To run cleanup manually (e.g., in a one-off script or scheduled job):

```go
repo := idempotency.NewInMemoryRepository() // or PostgresRepository in production
deleted, err := idempotency.CleanupOldKeys(repo, idempotency.DefaultExpiry)
if err != nil {
    log.Fatal(err)
}
log.Printf("Deleted %d old idempotency keys", deleted)
```

### Periodic Cleanup

To run cleanup automatically at regular intervals:

```go
stopChan := make(chan struct{})
go idempotency.RunPeriodicCleanup(
    repo,
    1*time.Hour,                    // Run every hour
    idempotency.DefaultExpiry,      // Delete keys older than 24h
    stopChan,
)

// When shutting down:
close(stopChan)
```

## Deployment Options

### Option 1: Cron Job (Recommended for Production)

Create a separate binary or script that connects to the database and runs cleanup:

```bash
#!/bin/bash
# /usr/local/bin/idempotency-cleanup.sh

# Run cleanup via database query
psql "$DATABASE_URL" <<EOF
DELETE FROM idempotency_keys 
WHERE created_at < NOW() - INTERVAL '24 hours';
EOF
```

Add to crontab to run every hour:

```cron
0 * * * * /usr/local/bin/idempotency-cleanup.sh
```

### Option 2: Built-in Periodic Cleanup

Add to `cmd/api/main.go` in the main function (after initializing idempotencyRepo):

```go
// Start background cleanup job
cleanupStopChan := make(chan struct{})
go idempotency.RunPeriodicCleanup(
    idempotencyRepo,
    1*time.Hour,
    idempotency.DefaultExpiry,
    cleanupStopChan,
)
defer close(cleanupStopChan)
```

### Option 3: Kubernetes CronJob

For Kubernetes deployments:

```yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: idempotency-cleanup
spec:
  schedule: "0 * * * *"  # Every hour
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: cleanup
            image: postgres:16
            command:
            - /bin/sh
            - -c
            - psql "$DATABASE_URL" -c "DELETE FROM idempotency_keys WHERE created_at < NOW() - INTERVAL '24 hours';"
            env:
            - name: DATABASE_URL
              valueFrom:
                secretKeyRef:
                  name: database-secret
                  key: url
          restartPolicy: OnFailure
```

## Database Query

For direct database cleanup (e.g., via psql or pgAdmin):

```sql
-- Delete idempotency keys older than 24 hours
DELETE FROM idempotency_keys 
WHERE created_at < NOW() - INTERVAL '24 hours';

-- Check how many keys would be deleted (dry run)
SELECT COUNT(*) FROM idempotency_keys 
WHERE created_at < NOW() - INTERVAL '24 hours';
```

## Monitoring

Monitor cleanup effectiveness:

```sql
-- Count of idempotency keys by age
SELECT 
    COUNT(*) AS total_keys,
    COUNT(*) FILTER (WHERE created_at < NOW() - INTERVAL '24 hours') AS expired_keys,
    COUNT(*) FILTER (WHERE created_at >= NOW() - INTERVAL '24 hours') AS active_keys
FROM idempotency_keys;

-- Oldest key age
SELECT 
    NOW() - MIN(created_at) AS oldest_key_age
FROM idempotency_keys;
```

## Configuration

The default expiry is 24 hours (`idempotency.DefaultExpiry`). This can be adjusted if needed:

```go
// Custom expiry (e.g., 48 hours)
customExpiry := 48 * time.Hour
deleted, err := idempotency.CleanupOldKeys(repo, customExpiry)
```

## Performance Considerations

- Cleanup runs incrementally and should not impact API performance
- For large datasets, consider partitioning the idempotency_keys table by created_at
- The cleanup uses an index on created_at for efficient deletion
- Running cleanup during off-peak hours can minimize database load
