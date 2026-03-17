//go:build integration

package stream_test

import (
	"testing"
	"time"

	"github.com/onnwee/subcults/internal/stream"
	"github.com/onnwee/subcults/internal/testutil"
)

func setupStreamSession(t *testing.T, tdb *testutil.TestDB) string {
	t.Helper()
	// Insert a minimal scene first (stream_sessions references scenes)
	_, err := tdb.DB.Exec(`
		INSERT INTO scenes (id, owner_did, name, description, geohash, allow_precise)
		VALUES ($1, $2, $3, $4, $5, FALSE)
	`, "scene-for-stream", "did:plc:streamer", "Stream Scene", "A scene", "u4pruydqqvj")
	if err != nil {
		t.Fatalf("inserting scene: %v", err)
	}

	// Insert a stream session
	sessionID := "sess-quality-test"
	_, err = tdb.DB.Exec(`
		INSERT INTO stream_sessions (id, scene_id, room_name, host_did)
		VALUES ($1, $2, $3, $4)
	`, sessionID, "scene-for-stream", "room-quality", "did:plc:streamer")
	if err != nil {
		t.Fatalf("inserting stream session: %v", err)
	}
	return sessionID
}

func TestPostgresQualityMetrics_RecordAndGetLatest(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	repo := stream.NewPostgresQualityMetricsRepository(tdb.DB)
	sessionID := setupStreamSession(t, tdb)

	bitrate := 128.0
	jitter := 5.2
	packetLoss := 1.5
	audioLevel := 0.7
	rtt := 45.0

	m := &stream.QualityMetrics{
		StreamSessionID:   sessionID,
		ParticipantID:     "participant-1",
		BitrateKbps:       &bitrate,
		JitterMs:          &jitter,
		PacketLossPercent: &packetLoss,
		AudioLevel:        &audioLevel,
		RTTMs:             &rtt,
		MeasuredAt:        time.Now(),
	}

	if err := repo.RecordMetrics(m); err != nil {
		t.Fatalf("RecordMetrics: %v", err)
	}
	if m.ID == "" {
		t.Error("expected ID to be set after insert")
	}

	got, err := repo.GetLatestMetrics(sessionID, "participant-1")
	if err != nil {
		t.Fatalf("GetLatestMetrics: %v", err)
	}
	if got.ID != m.ID {
		t.Errorf("ID mismatch: got %s, want %s", got.ID, m.ID)
	}
	if got.BitrateKbps == nil || *got.BitrateKbps != 128.0 {
		t.Errorf("BitrateKbps mismatch: got %v", got.BitrateKbps)
	}
	if got.PacketLossPercent == nil || *got.PacketLossPercent != 1.5 {
		t.Errorf("PacketLossPercent mismatch: got %v", got.PacketLossPercent)
	}
}

func TestPostgresQualityMetrics_GetLatestReturnsNewest(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	repo := stream.NewPostgresQualityMetricsRepository(tdb.DB)
	sessionID := setupStreamSession(t, tdb)

	bitrate1 := 64.0
	bitrate2 := 128.0
	now := time.Now()

	if err := repo.RecordMetrics(&stream.QualityMetrics{
		StreamSessionID: sessionID,
		ParticipantID:   "p-latest",
		BitrateKbps:     &bitrate1,
		MeasuredAt:      now.Add(-10 * time.Second),
	}); err != nil {
		t.Fatalf("RecordMetrics first: %v", err)
	}

	if err := repo.RecordMetrics(&stream.QualityMetrics{
		StreamSessionID: sessionID,
		ParticipantID:   "p-latest",
		BitrateKbps:     &bitrate2,
		MeasuredAt:      now,
	}); err != nil {
		t.Fatalf("RecordMetrics second: %v", err)
	}

	got, err := repo.GetLatestMetrics(sessionID, "p-latest")
	if err != nil {
		t.Fatalf("GetLatestMetrics: %v", err)
	}
	if *got.BitrateKbps != 128.0 {
		t.Errorf("expected latest bitrate 128.0, got %f", *got.BitrateKbps)
	}
}

func TestPostgresQualityMetrics_GetLatestNotFound(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	repo := stream.NewPostgresQualityMetricsRepository(tdb.DB)

	_, err := repo.GetLatestMetrics("nonexistent-session", "nonexistent-participant")
	if err != stream.ErrQualityMetricsNotFound {
		t.Errorf("expected ErrQualityMetricsNotFound, got %v", err)
	}
}

func TestPostgresQualityMetrics_GetMetricsBySession(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	repo := stream.NewPostgresQualityMetricsRepository(tdb.DB)
	sessionID := setupStreamSession(t, tdb)

	now := time.Now()
	for i := 0; i < 5; i++ {
		bitrate := float64(64 + i*16)
		if err := repo.RecordMetrics(&stream.QualityMetrics{
			StreamSessionID: sessionID,
			ParticipantID:   "p-session",
			BitrateKbps:     &bitrate,
			MeasuredAt:      now.Add(time.Duration(i) * time.Second),
		}); err != nil {
			t.Fatalf("RecordMetrics [%d]: %v", i, err)
		}
	}

	// Limit to 3
	metrics, err := repo.GetMetricsBySession(sessionID, 3)
	if err != nil {
		t.Fatalf("GetMetricsBySession: %v", err)
	}
	if len(metrics) != 3 {
		t.Fatalf("expected 3 metrics, got %d", len(metrics))
	}
	// Should be DESC order — newest first
	if *metrics[0].BitrateKbps < *metrics[1].BitrateKbps {
		t.Error("expected metrics in DESC order by measured_at")
	}
}

func TestPostgresQualityMetrics_GetMetricsTimeSeries(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	repo := stream.NewPostgresQualityMetricsRepository(tdb.DB)
	sessionID := setupStreamSession(t, tdb)

	now := time.Now()
	for i := 0; i < 5; i++ {
		bitrate := float64(64 + i*16)
		if err := repo.RecordMetrics(&stream.QualityMetrics{
			StreamSessionID: sessionID,
			ParticipantID:   "p-ts",
			BitrateKbps:     &bitrate,
			MeasuredAt:      now.Add(time.Duration(i) * time.Minute),
		}); err != nil {
			t.Fatalf("RecordMetrics [%d]: %v", i, err)
		}
	}

	// Query middle window (minutes 1-3)
	start := now.Add(1 * time.Minute)
	end := now.Add(3 * time.Minute)
	metrics, err := repo.GetMetricsTimeSeries(sessionID, "p-ts", start, end)
	if err != nil {
		t.Fatalf("GetMetricsTimeSeries: %v", err)
	}
	if len(metrics) != 3 {
		t.Fatalf("expected 3 metrics in window, got %d", len(metrics))
	}
	// Should be ASC order
	if *metrics[0].BitrateKbps > *metrics[1].BitrateKbps {
		t.Error("expected metrics in ASC order by measured_at")
	}
}

func TestPostgresQualityMetrics_GetParticipantsWithHighPacketLoss(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	repo := stream.NewPostgresQualityMetricsRepository(tdb.DB)
	sessionID := setupStreamSession(t, tdb)

	low := 2.0
	high := 10.0
	now := time.Now()

	// Low packet loss participant
	if err := repo.RecordMetrics(&stream.QualityMetrics{
		StreamSessionID:   sessionID,
		ParticipantID:     "p-good",
		PacketLossPercent: &low,
		MeasuredAt:        now,
	}); err != nil {
		t.Fatalf("RecordMetrics p-good: %v", err)
	}

	// High packet loss participant
	if err := repo.RecordMetrics(&stream.QualityMetrics{
		StreamSessionID:   sessionID,
		ParticipantID:     "p-bad",
		PacketLossPercent: &high,
		MeasuredAt:        now,
	}); err != nil {
		t.Fatalf("RecordMetrics p-bad: %v", err)
	}

	participants, err := repo.GetParticipantsWithHighPacketLoss(sessionID, 5)
	if err != nil {
		t.Fatalf("GetParticipantsWithHighPacketLoss: %v", err)
	}
	if len(participants) != 1 || participants[0] != "p-bad" {
		t.Errorf("expected [p-bad], got %v", participants)
	}
}

func TestPostgresQualityMetrics_ConstraintValidation(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	repo := stream.NewPostgresQualityMetricsRepository(tdb.DB)
	sessionID := setupStreamSession(t, tdb)

	tests := []struct {
		name    string
		metrics *stream.QualityMetrics
	}{
		{
			name: "negative bitrate rejected",
			metrics: func() *stream.QualityMetrics {
				v := -1.0
				return &stream.QualityMetrics{
					StreamSessionID: sessionID,
					ParticipantID:   "p-constraint",
					BitrateKbps:     &v,
					MeasuredAt:      time.Now(),
				}
			}(),
		},
		{
			name: "packet loss over 100 rejected",
			metrics: func() *stream.QualityMetrics {
				v := 101.0
				return &stream.QualityMetrics{
					StreamSessionID: sessionID,
					ParticipantID:   "p-constraint",
					PacketLossPercent: &v,
					MeasuredAt:        time.Now(),
				}
			}(),
		},
		{
			name: "audio level over 1 rejected",
			metrics: func() *stream.QualityMetrics {
				v := 1.5
				return &stream.QualityMetrics{
					StreamSessionID: sessionID,
					ParticipantID:   "p-constraint",
					AudioLevel:      &v,
					MeasuredAt:      time.Now(),
				}
			}(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := repo.RecordMetrics(tt.metrics)
			if err == nil {
				t.Error("expected CHECK constraint violation")
			}
		})
	}
}
