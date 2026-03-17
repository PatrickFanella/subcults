// Package stream provides audio quality metrics models for monitoring stream health.
package stream

import (
	"time"
)

// QualityMetrics represents audio quality measurements for a stream participant.
// These metrics are used to monitor network conditions and adapt quality settings.
type QualityMetrics struct {
	ID              string `json:"id"`
	StreamSessionID string `json:"stream_session_id"`
	ParticipantID   string `json:"participant_id"`

	// Audio quality metrics
	BitrateKbps       *float64 `json:"bitrate_kbps,omitempty"`        // Audio bitrate in kilobits per second
	JitterMs          *float64 `json:"jitter_ms,omitempty"`           // Jitter (packet delay variation) in milliseconds
	PacketLossPercent *float64 `json:"packet_loss_percent,omitempty"` // Packet loss percentage (0-100)
	AudioLevel        *float64 `json:"audio_level,omitempty"`         // Audio level (0.0-1.0, where 1.0 is loudest)

	// Network quality indicators
	RTTMs *float64 `json:"rtt_ms,omitempty"` // Round-trip time in milliseconds

	MeasuredAt time.Time `json:"measured_at"`
}

// HasHighPacketLoss returns true if packet loss exceeds the threshold (5%).
// This is used to trigger quality degradation alerts.
func (q *QualityMetrics) HasHighPacketLoss() bool {
	return q.PacketLossPercent != nil && *q.PacketLossPercent > 5.0
}

// HasPoorNetworkQuality returns true if any network quality indicator suggests degradation.
// Criteria:
// - Packet loss > 5%
// - Jitter > 30ms
// - RTT > 300ms
func (q *QualityMetrics) HasPoorNetworkQuality() bool {
	if q.HasHighPacketLoss() {
		return true
	}

	if q.JitterMs != nil && *q.JitterMs > 30.0 {
		return true
	}

	if q.RTTMs != nil && *q.RTTMs > 300.0 {
		return true
	}

	return false
}

// QualityAlert represents an alert triggered by poor audio quality.
type QualityAlert struct {
	StreamSessionID string    `json:"stream_session_id"`
	ParticipantID   string    `json:"participant_id"`
	AlertType       string    `json:"alert_type"` // "high_packet_loss", "high_jitter", "high_rtt"
	Severity        string    `json:"severity"`   // "warning", "critical"
	Metric          string    `json:"metric"`     // Human-readable metric description
	Value           float64   `json:"value"`      // Metric value that triggered the alert
	Threshold       float64   `json:"threshold"`  // Threshold that was exceeded
	DetectedAt      time.Time `json:"detected_at"`
}

// QualityMetricsRepository defines the interface for quality metrics data operations.
type QualityMetricsRepository interface {
	// RecordMetrics stores audio quality metrics for a participant.
	RecordMetrics(metrics *QualityMetrics) error

	// GetLatestMetrics retrieves the most recent quality metrics for a participant.
	GetLatestMetrics(streamSessionID, participantID string) (*QualityMetrics, error)

	// GetMetricsBySession retrieves all quality metrics for a stream session.
	// Results are ordered by measured_at DESC, with optional limit.
	GetMetricsBySession(streamSessionID string, limit int) ([]*QualityMetrics, error)

	// GetMetricsTimeSeries retrieves quality metrics for a participant within a time range.
	// Useful for visualizing quality trends over time.
	GetMetricsTimeSeries(streamSessionID, participantID string, start, end time.Time) ([]*QualityMetrics, error)

	// GetParticipantsWithHighPacketLoss returns participants with recent packet loss > 5%.
	// Used to identify participants needing quality adaptation.
	GetParticipantsWithHighPacketLoss(streamSessionID string, sinceMinutes int) ([]string, error)
}
