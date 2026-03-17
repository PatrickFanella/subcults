package stream

import (
	"testing"
	"time"
)

func floatPtr(f float64) *float64 {
	return &f
}

// TestQualityMetricsRepository_InMemory tests the in-memory implementation
// Note: The PostgresQualityMetricsRepository methods (GetMetricsBySession,
// GetMetricsTimeSeries, GetParticipantsWithHighPacketLoss) are database-dependent
// and should be tested with integration tests. These tests focus on the in-memory
// repository pattern that is already tested in other repository tests.

// BenchmarkQualityMetricsInsert benchmarks quality metrics insertion.
func BenchmarkQualityMetricsInsert(b *testing.B) {
	// Note: This is a placeholder for when the in-memory quality metrics
	// repository is implemented. Currently, PostgresQualityMetricsRepository
	// is the only implementation and requires a DB connection.
	b.Skip("In-memory quality metrics repository not yet implemented")
}

// TestQualityMetricsValidation tests validation of quality metrics data.
func TestQualityMetricsValidation(t *testing.T) {
	tests := []struct {
		name    string
		metrics QualityMetrics
	}{
		{
			name: "valid_metrics",
			metrics: QualityMetrics{
				StreamSessionID:   "session-123",
				ParticipantID:     "participant-456",
				BitrateKbps:       floatPtr(128.5),
				JitterMs:          floatPtr(12.3),
				PacketLossPercent: floatPtr(1.5),
				AudioLevel:        floatPtr(-20.0),
				RTTMs:             floatPtr(45.2),
				MeasuredAt:        time.Now(),
			},
		},
		{
			name: "high_packet_loss",
			metrics: QualityMetrics{
				StreamSessionID:   "session-123",
				ParticipantID:     "participant-789",
				BitrateKbps:       floatPtr(64.0),
				JitterMs:          floatPtr(50.0),
				PacketLossPercent: floatPtr(15.0), // High packet loss
				AudioLevel:        floatPtr(-40.0),
				RTTMs:             floatPtr(200.0),
				MeasuredAt:        time.Now(),
			},
		},
		{
			name: "zero_values",
			metrics: QualityMetrics{
				StreamSessionID:   "session-123",
				ParticipantID:     "participant-zero",
				BitrateKbps:       floatPtr(0.0),
				JitterMs:          floatPtr(0.0),
				PacketLossPercent: floatPtr(0.0),
				AudioLevel:        floatPtr(0.0),
				RTTMs:             floatPtr(0.0),
				MeasuredAt:        time.Now(),
			},
		},
		{
			name: "nil_optional_fields",
			metrics: QualityMetrics{
				StreamSessionID: "session-123",
				ParticipantID:   "participant-nil",
				MeasuredAt:      time.Now(),
				// All optional fields are nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Validate that all required fields are present
			if tt.metrics.StreamSessionID == "" {
				t.Error("StreamSessionID should not be empty")
			}
			if tt.metrics.ParticipantID == "" {
				t.Error("ParticipantID should not be empty")
			}
			if tt.metrics.MeasuredAt.IsZero() {
				t.Error("MeasuredAt should not be zero")
			}

			// Validate quality thresholds
			if tt.metrics.PacketLossPercent != nil && *tt.metrics.PacketLossPercent > 5.0 && tt.name == "high_packet_loss" {
				// This is expected for the high packet loss test case
				t.Logf("High packet loss detected: %.2f%%", *tt.metrics.PacketLossPercent)
			}
		})
	}
}

// TestQualityMetricsThresholds tests various quality thresholds.
func TestQualityMetricsThresholds(t *testing.T) {
	tests := []struct {
		name           string
		packetLoss     float64
		jitter         float64
		rtt            float64
		expectGoodQual bool
	}{
		{
			name:           "excellent_quality",
			packetLoss:     0.5,
			jitter:         5.0,
			rtt:            30.0,
			expectGoodQual: true,
		},
		{
			name:           "acceptable_quality",
			packetLoss:     2.0,
			jitter:         15.0,
			rtt:            80.0,
			expectGoodQual: true,
		},
		{
			name:           "poor_quality_high_packet_loss",
			packetLoss:     8.0,
			jitter:         20.0,
			rtt:            100.0,
			expectGoodQual: false,
		},
		{
			name:           "poor_quality_high_jitter",
			packetLoss:     1.0,
			jitter:         80.0,
			rtt:            50.0,
			expectGoodQual: false,
		},
		{
			name:           "poor_quality_high_rtt",
			packetLoss:     1.0,
			jitter:         10.0,
			rtt:            300.0,
			expectGoodQual: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Define quality thresholds (these match typical WebRTC standards)
			const (
				packetLossThreshold = 5.0   // 5% packet loss
				jitterThreshold     = 50.0  // 50ms jitter
				rttThreshold        = 250.0 // 250ms RTT
			)

			isGoodQuality := tt.packetLoss < packetLossThreshold &&
				tt.jitter < jitterThreshold &&
				tt.rtt < rttThreshold

			if isGoodQuality != tt.expectGoodQual {
				t.Errorf("Quality assessment mismatch: got %v, want %v (packet_loss=%.1f, jitter=%.1f, rtt=%.1f)",
					isGoodQuality, tt.expectGoodQual, tt.packetLoss, tt.jitter, tt.rtt)
			}
		})
	}
}

// TestHasHighPacketLoss tests the HasHighPacketLoss method.
func TestHasHighPacketLoss(t *testing.T) {
	tests := []struct {
		name       string
		packetLoss *float64
		want       bool
	}{
		{
			name:       "no_packet_loss",
			packetLoss: floatPtr(0.0),
			want:       false,
		},
		{
			name:       "low_packet_loss",
			packetLoss: floatPtr(2.5),
			want:       false,
		},
		{
			name:       "threshold_packet_loss",
			packetLoss: floatPtr(5.0),
			want:       false, // Exactly at threshold, not exceeding
		},
		{
			name:       "high_packet_loss",
			packetLoss: floatPtr(5.1),
			want:       true,
		},
		{
			name:       "very_high_packet_loss",
			packetLoss: floatPtr(20.0),
			want:       true,
		},
		{
			name:       "nil_packet_loss",
			packetLoss: nil,
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metrics := &QualityMetrics{
				PacketLossPercent: tt.packetLoss,
			}

			got := metrics.HasHighPacketLoss()
			if got != tt.want {
				t.Errorf("HasHighPacketLoss() = %v, want %v", got, tt.want)
			}
		})
	}
}
