// Package stream provides metrics for streaming session analytics.
package stream

import (
	"github.com/prometheus/client_golang/prometheus"
)

// Metrics names as constants for consistency.
const (
	MetricStreamJoins       = "stream_joins_total"
	MetricStreamLeaves      = "stream_leaves_total"
	MetricStreamJoinLatency = "stream_join_latency_seconds"

	// Audio quality metrics
	MetricAudioBitrate    = "stream_audio_bitrate_kbps"
	MetricAudioJitter     = "stream_audio_jitter_ms"
	MetricAudioPacketLoss = "stream_audio_packet_loss_percent"
	MetricAudioLevel      = "stream_audio_level"
	MetricNetworkRTT      = "stream_network_rtt_ms"
	MetricQualityAlerts   = "stream_quality_alerts_total"
	MetricHighPacketLoss  = "stream_high_packet_loss_total"
)

// Metrics contains Prometheus metrics for streaming sessions.
// All operations are thread-safe.
type Metrics struct {
	streamJoins       prometheus.Counter
	streamLeaves      prometheus.Counter
	streamJoinLatency prometheus.Histogram

	// Audio quality metrics
	audioBitrate    prometheus.Histogram
	audioJitter     prometheus.Histogram
	audioPacketLoss prometheus.Histogram
	audioLevel      prometheus.Histogram
	networkRTT      prometheus.Histogram
	qualityAlerts   prometheus.Counter
	highPacketLoss  prometheus.Counter
}

// NewMetrics creates and returns a new Metrics instance with all collectors initialized.
// The metrics are not registered; call Register to register them with a registry.
func NewMetrics() *Metrics {
	return &Metrics{
		streamJoins: prometheus.NewCounter(prometheus.CounterOpts{
			Name: MetricStreamJoins,
			Help: "Total number of stream join events",
		}),
		streamLeaves: prometheus.NewCounter(prometheus.CounterOpts{
			Name: MetricStreamLeaves,
			Help: "Total number of stream leave events",
		}),
		streamJoinLatency: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    MetricStreamJoinLatency,
			Help:    "Histogram of stream join completion latency in seconds (from token issuance to first audio track subscription)",
			Buckets: []float64{0.1, 0.25, 0.5, 1.0, 2.0, 5.0, 10.0, 30.0, 60.0},
		}),
		audioBitrate: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    MetricAudioBitrate,
			Help:    "Audio bitrate in kilobits per second",
			Buckets: []float64{16, 32, 64, 96, 128, 160, 192, 256, 320},
		}),
		audioJitter: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    MetricAudioJitter,
			Help:    "Audio jitter (packet delay variation) in milliseconds",
			Buckets: []float64{1, 5, 10, 20, 30, 50, 100, 200},
		}),
		audioPacketLoss: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    MetricAudioPacketLoss,
			Help:    "Audio packet loss percentage (0-100)",
			Buckets: []float64{0.1, 0.5, 1, 2, 5, 10, 20, 50},
		}),
		audioLevel: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    MetricAudioLevel,
			Help:    "Audio level (0.0-1.0, where 1.0 is loudest)",
			Buckets: []float64{0.0, 0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 1.0},
		}),
		networkRTT: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    MetricNetworkRTT,
			Help:    "Network round-trip time in milliseconds",
			Buckets: []float64{10, 25, 50, 100, 150, 200, 300, 500, 1000},
		}),
		qualityAlerts: prometheus.NewCounter(prometheus.CounterOpts{
			Name: MetricQualityAlerts,
			Help: "Total number of audio quality alerts triggered",
		}),
		highPacketLoss: prometheus.NewCounter(prometheus.CounterOpts{
			Name: MetricHighPacketLoss,
			Help: "Total number of high packet loss events (>5%)",
		}),
	}
}

// Register registers all metrics with the given registry.
// Returns an error if registration fails.
func (m *Metrics) Register(reg prometheus.Registerer) error {
	collectors := []prometheus.Collector{
		m.streamJoins,
		m.streamLeaves,
		m.streamJoinLatency,
		m.audioBitrate,
		m.audioJitter,
		m.audioPacketLoss,
		m.audioLevel,
		m.networkRTT,
		m.qualityAlerts,
		m.highPacketLoss,
	}

	for _, c := range collectors {
		if err := reg.Register(c); err != nil {
			return err
		}
	}
	return nil
}

// IncStreamJoins increments the stream joins counter.
func (m *Metrics) IncStreamJoins() {
	m.streamJoins.Inc()
}

// IncStreamLeaves increments the stream leaves counter.
func (m *Metrics) IncStreamLeaves() {
	m.streamLeaves.Inc()
}

// ObserveStreamJoinLatency records a stream join latency sample.
func (m *Metrics) ObserveStreamJoinLatency(seconds float64) {
	m.streamJoinLatency.Observe(seconds)
}

// ObserveAudioBitrate records an audio bitrate sample in kbps.
func (m *Metrics) ObserveAudioBitrate(kbps float64) {
	m.audioBitrate.Observe(kbps)
}

// ObserveAudioJitter records an audio jitter sample in milliseconds.
func (m *Metrics) ObserveAudioJitter(ms float64) {
	m.audioJitter.Observe(ms)
}

// ObserveAudioPacketLoss records a packet loss percentage sample.
// If packet loss exceeds 5%, also increments the high packet loss counter.
func (m *Metrics) ObserveAudioPacketLoss(percent float64) {
	m.audioPacketLoss.Observe(percent)
	if percent > 5.0 {
		m.highPacketLoss.Inc()
	}
}

// ObserveAudioLevel records an audio level sample (0.0-1.0).
func (m *Metrics) ObserveAudioLevel(level float64) {
	m.audioLevel.Observe(level)
}

// ObserveNetworkRTT records a network round-trip time sample in milliseconds.
func (m *Metrics) ObserveNetworkRTT(ms float64) {
	m.networkRTT.Observe(ms)
}

// IncQualityAlerts increments the quality alerts counter.
func (m *Metrics) IncQualityAlerts() {
	m.qualityAlerts.Inc()
}

// Collectors returns all Prometheus collectors for testing.
func (m *Metrics) Collectors() []prometheus.Collector {
	return []prometheus.Collector{
		m.streamJoins,
		m.streamLeaves,
		m.streamJoinLatency,
		m.audioBitrate,
		m.audioJitter,
		m.audioPacketLoss,
		m.audioLevel,
		m.networkRTT,
		m.qualityAlerts,
		m.highPacketLoss,
	}
}
