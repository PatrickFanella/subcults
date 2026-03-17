package stream

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

var (
	// ErrQualityMetricsNotFound is returned when quality metrics are not found.
	ErrQualityMetricsNotFound = errors.New("quality metrics not found")
)

// PostgresQualityMetricsRepository implements QualityMetricsRepository using PostgreSQL.
type PostgresQualityMetricsRepository struct {
	db *sql.DB
}

// NewPostgresQualityMetricsRepository creates a new PostgresQualityMetricsRepository.
func NewPostgresQualityMetricsRepository(db *sql.DB) *PostgresQualityMetricsRepository {
	return &PostgresQualityMetricsRepository{db: db}
}

// RecordMetrics stores audio quality metrics for a participant.
func (r *PostgresQualityMetricsRepository) RecordMetrics(metrics *QualityMetrics) error {
	// Default to current time if MeasuredAt is zero value
	measuredAt := metrics.MeasuredAt
	if measuredAt.IsZero() {
		measuredAt = time.Now()
	}

	query := `
		INSERT INTO stream_quality_metrics (
			stream_session_id, participant_id,
			bitrate_kbps, jitter_ms, packet_loss_percent, audio_level, rtt_ms,
			measured_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id
	`

	var id string
	err := r.db.QueryRow(
		query,
		metrics.StreamSessionID,
		metrics.ParticipantID,
		metrics.BitrateKbps,
		metrics.JitterMs,
		metrics.PacketLossPercent,
		metrics.AudioLevel,
		metrics.RTTMs,
		measuredAt,
	).Scan(&id)

	if err != nil {
		return fmt.Errorf("failed to record quality metrics: %w", err)
	}

	metrics.ID = id
	metrics.MeasuredAt = measuredAt
	return nil
}

// GetLatestMetrics retrieves the most recent quality metrics for a participant.
func (r *PostgresQualityMetricsRepository) GetLatestMetrics(streamSessionID, participantID string) (*QualityMetrics, error) {
	query := `
		SELECT id, stream_session_id, participant_id,
		       bitrate_kbps, jitter_ms, packet_loss_percent, audio_level, rtt_ms,
		       measured_at
		FROM stream_quality_metrics
		WHERE stream_session_id = $1 AND participant_id = $2
		ORDER BY measured_at DESC
		LIMIT 1
	`

	metrics := &QualityMetrics{}
	err := r.db.QueryRow(query, streamSessionID, participantID).Scan(
		&metrics.ID,
		&metrics.StreamSessionID,
		&metrics.ParticipantID,
		&metrics.BitrateKbps,
		&metrics.JitterMs,
		&metrics.PacketLossPercent,
		&metrics.AudioLevel,
		&metrics.RTTMs,
		&metrics.MeasuredAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrQualityMetricsNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get latest metrics: %w", err)
	}

	return metrics, nil
}

// GetMetricsBySession retrieves all quality metrics for a stream session.
func (r *PostgresQualityMetricsRepository) GetMetricsBySession(streamSessionID string, limit int) ([]*QualityMetrics, error) {
	query := `
		SELECT id, stream_session_id, participant_id,
		       bitrate_kbps, jitter_ms, packet_loss_percent, audio_level, rtt_ms,
		       measured_at
		FROM stream_quality_metrics
		WHERE stream_session_id = $1
		ORDER BY measured_at DESC
		LIMIT $2
	`

	rows, err := r.db.Query(query, streamSessionID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get metrics by session: %w", err)
	}
	defer rows.Close()

	var metrics []*QualityMetrics
	for rows.Next() {
		m := &QualityMetrics{}
		err := rows.Scan(
			&m.ID,
			&m.StreamSessionID,
			&m.ParticipantID,
			&m.BitrateKbps,
			&m.JitterMs,
			&m.PacketLossPercent,
			&m.AudioLevel,
			&m.RTTMs,
			&m.MeasuredAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan quality metrics: %w", err)
		}
		metrics = append(metrics, m)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating quality metrics: %w", err)
	}

	return metrics, nil
}

// GetMetricsTimeSeries retrieves quality metrics for a participant within a time range.
func (r *PostgresQualityMetricsRepository) GetMetricsTimeSeries(streamSessionID, participantID string, start, end time.Time) ([]*QualityMetrics, error) {
	query := `
		SELECT id, stream_session_id, participant_id,
		       bitrate_kbps, jitter_ms, packet_loss_percent, audio_level, rtt_ms,
		       measured_at
		FROM stream_quality_metrics
		WHERE stream_session_id = $1 
		  AND participant_id = $2
		  AND measured_at >= $3
		  AND measured_at <= $4
		ORDER BY measured_at ASC
	`

	rows, err := r.db.Query(query, streamSessionID, participantID, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to get metrics time series: %w", err)
	}
	defer rows.Close()

	var metrics []*QualityMetrics
	for rows.Next() {
		m := &QualityMetrics{}
		err := rows.Scan(
			&m.ID,
			&m.StreamSessionID,
			&m.ParticipantID,
			&m.BitrateKbps,
			&m.JitterMs,
			&m.PacketLossPercent,
			&m.AudioLevel,
			&m.RTTMs,
			&m.MeasuredAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan quality metrics: %w", err)
		}
		metrics = append(metrics, m)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating quality metrics: %w", err)
	}

	return metrics, nil
}

// GetParticipantsWithHighPacketLoss returns participants with recent packet loss > 5%.
func (r *PostgresQualityMetricsRepository) GetParticipantsWithHighPacketLoss(streamSessionID string, sinceMinutes int) ([]string, error) {
	query := `
		SELECT DISTINCT participant_id
		FROM stream_quality_metrics
		WHERE stream_session_id = $1
		  AND packet_loss_percent > 5.0
		  AND measured_at >= NOW() - ($2 * INTERVAL '1 minute')
		ORDER BY participant_id
	`

	rows, err := r.db.Query(query, streamSessionID, sinceMinutes)
	if err != nil {
		return nil, fmt.Errorf("failed to get participants with high packet loss: %w", err)
	}
	defer rows.Close()

	var participants []string
	for rows.Next() {
		var participantID string
		if err := rows.Scan(&participantID); err != nil {
			return nil, fmt.Errorf("failed to scan participant ID: %w", err)
		}
		participants = append(participants, participantID)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating participants: %w", err)
	}

	return participants, nil
}
