package stream

import (
	"testing"
)

func TestQualityMetrics_HasHighPacketLoss(t *testing.T) {
	tests := []struct {
		name       string
		packetLoss *float64
		wantAlert  bool
	}{
		{
			name:       "nil packet loss",
			packetLoss: nil,
			wantAlert:  false,
		},
		{
			name:       "zero packet loss",
			packetLoss: ptrFloat64(0.0),
			wantAlert:  false,
		},
		{
			name:       "low packet loss (3%)",
			packetLoss: ptrFloat64(3.0),
			wantAlert:  false,
		},
		{
			name:       "threshold packet loss (5%)",
			packetLoss: ptrFloat64(5.0),
			wantAlert:  false, // Threshold is exclusive (> 5%)
		},
		{
			name:       "high packet loss (5.1%)",
			packetLoss: ptrFloat64(5.1),
			wantAlert:  true,
		},
		{
			name:       "very high packet loss (20%)",
			packetLoss: ptrFloat64(20.0),
			wantAlert:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := &QualityMetrics{
				PacketLossPercent: tt.packetLoss,
			}

			if got := q.HasHighPacketLoss(); got != tt.wantAlert {
				t.Errorf("HasHighPacketLoss() = %v, want %v", got, tt.wantAlert)
			}
		})
	}
}

func TestQualityMetrics_HasPoorNetworkQuality(t *testing.T) {
	tests := []struct {
		name     string
		metrics  *QualityMetrics
		wantPoor bool
	}{
		{
			name: "all metrics good",
			metrics: &QualityMetrics{
				PacketLossPercent: ptrFloat64(1.0),
				JitterMs:          ptrFloat64(10.0),
				RTTMs:             ptrFloat64(50.0),
			},
			wantPoor: false,
		},
		{
			name: "high packet loss",
			metrics: &QualityMetrics{
				PacketLossPercent: ptrFloat64(6.0),
				JitterMs:          ptrFloat64(10.0),
				RTTMs:             ptrFloat64(50.0),
			},
			wantPoor: true,
		},
		{
			name: "high jitter",
			metrics: &QualityMetrics{
				PacketLossPercent: ptrFloat64(1.0),
				JitterMs:          ptrFloat64(35.0),
				RTTMs:             ptrFloat64(50.0),
			},
			wantPoor: true,
		},
		{
			name: "high RTT",
			metrics: &QualityMetrics{
				PacketLossPercent: ptrFloat64(1.0),
				JitterMs:          ptrFloat64(10.0),
				RTTMs:             ptrFloat64(350.0),
			},
			wantPoor: true,
		},
		{
			name: "threshold jitter (30ms)",
			metrics: &QualityMetrics{
				PacketLossPercent: ptrFloat64(1.0),
				JitterMs:          ptrFloat64(30.0),
				RTTMs:             ptrFloat64(50.0),
			},
			wantPoor: false, // Threshold is exclusive (> 30ms)
		},
		{
			name: "threshold RTT (300ms)",
			metrics: &QualityMetrics{
				PacketLossPercent: ptrFloat64(1.0),
				JitterMs:          ptrFloat64(10.0),
				RTTMs:             ptrFloat64(300.0),
			},
			wantPoor: false, // Threshold is exclusive (> 300ms)
		},
		{
			name: "all nil metrics",
			metrics: &QualityMetrics{
				PacketLossPercent: nil,
				JitterMs:          nil,
				RTTMs:             nil,
			},
			wantPoor: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.metrics.HasPoorNetworkQuality(); got != tt.wantPoor {
				t.Errorf("HasPoorNetworkQuality() = %v, want %v", got, tt.wantPoor)
			}
		})
	}
}

// ptrFloat64 is a helper to create a pointer to a float64.
func ptrFloat64(f float64) *float64 {
	return &f
}
