# Audio Quality Metrics Integration Guide

## Overview

This guide explains how to integrate the audio quality metrics handlers into the API server and set up periodic metrics collection.

## API Handler Integration

### 1. Initialize Quality Metrics Handler

In `cmd/api/main.go`, add the quality metrics handler after initializing the stream handlers:

```go
// Initialize quality metrics repository
qualityMetricsRepo := stream.NewPostgresQualityMetricsRepository(db)

// Create quality metrics handler
qualityMetricsHandler := api.NewQualityMetricsHandler(
    roomService,
    qualityMetricsRepo,
    streamRepo,
    streamMetrics,
)
```

### 2. Register Quality Metrics Routes

Add the following routes to the stream routing section in `cmd/api/main.go`:

```go
mux.HandleFunc("/streams/", func(w http.ResponseWriter, r *http.Request) {
    pathParts := strings.Split(strings.TrimPrefix(r.URL.Path, "/streams/"), "/")

    // ... existing routes ...

    // Quality metrics routes
    
    // GET /streams/{id}/quality-metrics
    if len(pathParts) == 2 && pathParts[0] != "" && pathParts[1] == "quality-metrics" && r.Method == http.MethodGet {
        qualityMetricsHandler.GetStreamQualityMetrics(w, r)
        return
    }

    // POST /streams/{id}/quality-metrics/collect
    if len(pathParts) == 3 && pathParts[0] != "" && pathParts[1] == "quality-metrics" && pathParts[2] == "collect" && r.Method == http.MethodPost {
        qualityMetricsHandler.CollectStreamQualityMetrics(w, r)
        return
    }

    // GET /streams/{id}/quality-metrics/high-packet-loss
    if len(pathParts) == 3 && pathParts[0] != "" && pathParts[1] == "quality-metrics" && pathParts[2] == "high-packet-loss" && r.Method == http.MethodGet {
        qualityMetricsHandler.GetHighPacketLossParticipants(w, r)
        return
    }

    // GET /streams/{id}/participants/{participant_id}/quality-metrics
    if len(pathParts) == 4 && pathParts[0] != "" && pathParts[1] == "participants" && pathParts[3] == "quality-metrics" && r.Method == http.MethodGet {
        qualityMetricsHandler.GetParticipantQualityMetrics(w, r)
        return
    }

    // ... rest of stream routes ...
})
```

### 3. Apply Authentication Middleware

Ensure quality metrics endpoints are protected with authentication:

```go
// Wrap handler with authentication middleware
authenticatedQualityMetricsHandler := middleware.Authenticate(jwtSecret)(
    http.HandlerFunc(qualityMetricsHandler.CollectStreamQualityMetrics),
)
```

## Periodic Metrics Collection

### Background Job Approach

Create a background goroutine to periodically collect metrics from active streams:

```go
// In cmd/api/main.go, after server initialization

// Start background metrics collection
go func() {
    ticker := time.NewTicker(30 * time.Second) // Collect every 30 seconds
    defer ticker.Stop()

    for range ticker.C {
        collectQualityMetrics(
            context.Background(),
            streamRepo,
            qualityMetricsRepo,
            roomService,
            streamMetrics,
            logger,
        )
    }
}()

// Helper function
func collectQualityMetrics(
    ctx context.Context,
    streamRepo stream.SessionRepository,
    metricsRepo stream.QualityMetricsRepository,
    roomService *livekit.RoomService,
    metrics *stream.Metrics,
    logger *slog.Logger,
) {
    // NOTE: SessionRepository does not have a ListActive method.
    // You'll need to implement a method to get active streams, for example:
    // - Add ListActive() method to SessionRepository interface
    // - Query database directly: SELECT * FROM stream_sessions WHERE ended_at IS NULL
    // - Or maintain an in-memory cache of active streams
    //
    // Example implementation (requires adding to repository):
    //   activeStreams, err := streamRepo.ListActive(ctx)
    //
    // For now, this is a conceptual example showing the collection pattern.

    logger.Info("quality metrics collection skipped - ListActive not implemented")
    // Uncomment and implement once ListActive is available:
    /*
    activeStreams, err := streamRepo.ListActive(ctx)
    if err != nil {
        logger.Error("failed to list active streams", "error", err)
        return
    }

    for _, session := range activeStreams {
        // Get participants from LiveKit
        participants, err := roomService.GetAllParticipantStats(ctx, session.RoomName)
        if err != nil {
            logger.Error("failed to get participant stats",
                "stream_id", session.ID,
                "room_name", session.RoomName,
                "error", err)
            continue
        }

        // Collect metrics for each participant
        measuredAt := time.Now()
        for _, participant := range participants {
            qualityMetrics := extractQualityMetrics(session.ID, participant, measuredAt)

            // Store in database
            if err := metricsRepo.RecordMetrics(qualityMetrics); err != nil {
                logger.Error("failed to record quality metrics",
                    "participant_id", participant.Identity,
                    "error", err)
                continue
            }

            // Update Prometheus metrics
            updatePrometheusMetrics(metrics, qualityMetrics)

            // Check for alerts
            if qualityMetrics.HasPoorNetworkQuality() {
                metrics.IncQualityAlerts()
                logger.Warn("poor network quality detected",
                    "stream_id", session.ID,
                    "participant_id", participant.Identity,
                    "packet_loss", qualityMetrics.PacketLossPercent,
                    "jitter", qualityMetrics.JitterMs,
                    "rtt", qualityMetrics.RTTMs)
            }
        }
    }
}

// Helper to extract metrics (placeholder - see quality_metrics_handlers.go for full docs)
func extractQualityMetrics(streamID string, participant *livekit.ParticipantInfo, measuredAt time.Time) *stream.QualityMetrics {
    // NOTE: This is a placeholder. Actual extraction depends on LiveKit protocol version.
    // See internal/api/quality_metrics_handlers.go for detailed implementation notes.
    return &stream.QualityMetrics{
        StreamSessionID: streamID,
        ParticipantID:   participant.Identity,
        MeasuredAt:      measuredAt,
        // TODO: Extract actual metrics from participant.Tracks[].Stats
    }
}

// Helper to update Prometheus metrics
func updatePrometheusMetrics(metrics *stream.Metrics, qualityMetrics *stream.QualityMetrics) {
    if qualityMetrics.BitrateKbps != nil {
        metrics.ObserveAudioBitrate(*qualityMetrics.BitrateKbps)
    }
    if qualityMetrics.JitterMs != nil {
        metrics.ObserveAudioJitter(*qualityMetrics.JitterMs)
    }
    if qualityMetrics.PacketLossPercent != nil {
        metrics.ObserveAudioPacketLoss(*qualityMetrics.PacketLossPercent)
    }
    if qualityMetrics.AudioLevel != nil {
        metrics.ObserveAudioLevel(*qualityMetrics.AudioLevel)
    }
    if qualityMetrics.RTTMs != nil {
        metrics.ObserveNetworkRTT(*qualityMetrics.RTTMs)
    }
}
```

### External Monitoring Service

Alternatively, use an external service to poll the collection endpoint.

**Note**: You'll need to iterate through active streams. Here's a conceptual example:

```bash
# Bash script to collect metrics for all active streams
# This requires implementing an endpoint to list active streams

#!/bin/bash
API_TOKEN="your-jwt-token"
API_URL="http://api:8080"

# Get active stream IDs (requires implementing /streams?status=active endpoint)
STREAMS=$(curl -s "$API_URL/streams?status=active" -H "Authorization: Bearer $API_TOKEN" | jq -r '.streams[].id')

# Collect metrics for each stream
for stream_id in $STREAMS; do
    echo "Collecting metrics for stream $stream_id"
    curl -X POST "$API_URL/streams/$stream_id/quality-metrics/collect" \
        -H "Authorization: Bearer $API_TOKEN"
done
```

### Kubernetes CronJob

**Security Note**: The following example uses a pinned, trusted image with minimal permissions. Avoid using mutable tags like `latest` for security-sensitive workloads.

```yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: stream-quality-metrics-collector
spec:
  schedule: "*/1 * * * *"  # Every minute
  jobTemplate:
    spec:
      template:
        spec:
          serviceAccountName: metrics-collector  # Use dedicated service account with minimal permissions
          containers:
          - name: collector
            # Use a pinned, immutable image digest instead of mutable tags
            # Replace with your organization's trusted image
            image: curlimages/curl:8.5.0@sha256:4bfa3e2c0164fb103fb9bfd4dc956facce32b6c5d2f61e8a9f00f0f2f2b4c3c0  # Example digest - verify actual digest
            command:
            - /bin/sh
            - -c
            - |
              # Get active streams (requires implementing /streams?status=active endpoint)
              for stream_id in $(curl -s -H "Authorization: Bearer $API_TOKEN" \
                "http://api:8080/streams?status=active" | jq -r '.streams[].id'); do
                
                curl -X POST "http://api:8080/streams/$stream_id/quality-metrics/collect" \
                  -H "Authorization: Bearer $API_TOKEN"
              done
            env:
            - name: API_TOKEN
              valueFrom:
                secretKeyRef:
                  name: api-secrets
                  key: token
            securityContext:
              runAsNonRoot: true
              runAsUser: 1000
              allowPrivilegeEscalation: false
              readOnlyRootFilesystem: true
              capabilities:
                drop:
                - ALL
          restartPolicy: OnFailure
```

**Security Best Practices**:
- Use image digests instead of mutable tags (`:latest`, `:8.5.0`)
- Create a dedicated ServiceAccount with minimal RBAC permissions
- Run as non-root user with restrictive security context
- Consider using a first-party helper image from your own registry
- Rotate API tokens regularly
- Use network policies to restrict pod egress to only the API server

## Database Migration

Before using quality metrics, run the migration:

```bash
# Apply migration
make migrate-up

# Or manually
./scripts/migrate.sh up
```

Verify the table was created:

```sql
SELECT * FROM stream_quality_metrics LIMIT 1;
```

## Testing

### Manual Test

1. Start an active stream
2. Call the collect endpoint:

```bash
curl -X POST http://localhost:8080/streams/{stream_id}/quality-metrics/collect \
  -H "Authorization: Bearer $JWT_TOKEN"
```

3. Verify metrics were recorded:

```bash
curl http://localhost:8080/streams/{stream_id}/quality-metrics \
  -H "Authorization: Bearer $JWT_TOKEN"
```

### Check Prometheus Metrics

```bash
curl http://localhost:8080/metrics | grep stream_audio
```

Expected output:
```
# HELP stream_audio_bitrate_kbps Audio bitrate in kilobits per second
# TYPE stream_audio_bitrate_kbps histogram
stream_audio_bitrate_kbps_bucket{le="16"} 0
stream_audio_bitrate_kbps_bucket{le="32"} 0
...
```

## Monitoring Setup

### Grafana Dashboard

Import the dashboard from `docs/grafana/audio-quality-dashboard.json` (to be created).

### Prometheus Alerts

Add the alert rules from `docs/OBSERVABILITY.md` to your Prometheus configuration.

### Log Monitoring

Quality alerts are logged with structured fields:

```json
{
  "level": "warn",
  "msg": "poor network quality detected",
  "stream_id": "uuid",
  "participant_id": "user-abc123",
  "packet_loss": 6.5,
  "jitter": 45.2,
  "rtt": 350.0
}
```

Set up log aggregation to track quality issues over time.

## Performance Considerations

### Collection Frequency

- **Default**: 30 seconds
- **Low-latency requirements**: 15 seconds
- **High volume**: 60 seconds

### Database Cleanup

Quality metrics can accumulate quickly. Set up a cleanup job:

```sql
-- Delete metrics older than 7 days
DELETE FROM stream_quality_metrics
WHERE measured_at < NOW() - INTERVAL '7 days';
```

Or use a cron job:

```bash
# Daily cleanup at 2 AM
0 2 * * * psql $DATABASE_URL -c "DELETE FROM stream_quality_metrics WHERE measured_at < NOW() - INTERVAL '7 days';"
```

### Indexing

The migration includes optimal indexes for common queries:

- `idx_quality_metrics_session`: Stream session lookups
- `idx_quality_metrics_participant`: Participant history
- `idx_quality_metrics_packet_loss`: High packet loss queries (partial index)

## Troubleshooting

### No Metrics Collected

1. Verify LiveKit connection:
   ```bash
   curl http://localhost:8080/health
   ```

2. Check stream is active:
   ```bash
   curl http://localhost:8080/streams/{id}
   ```

3. Verify LiveKit room exists:
   - Check LiveKit dashboard
   - Verify room name matches stream session

### High Database Load

- Reduce collection frequency
- Implement batching for metrics storage
- Enable connection pooling
- Consider time-series database (e.g., TimescaleDB)

### Missing Prometheus Metrics

- Verify metrics are registered at startup
- Check `/metrics` endpoint accessibility
- Ensure metrics are being updated in handler
- Verify Prometheus scrape config

## Next Steps

1. Implement client-side codec adaptation based on metrics
2. Add WebSocket notifications for quality alerts
3. Create quality trend visualizations
4. Implement predictive quality analysis
5. Add geographic quality correlation

## See Also

- [OBSERVABILITY.md](./OBSERVABILITY.md) - Full metrics documentation
- [ARCHITECTURE.md](./ARCHITECTURE.md) - System architecture
- [API Reference](./API_REFERENCE.md) - Complete API documentation
