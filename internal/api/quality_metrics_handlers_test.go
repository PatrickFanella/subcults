package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/onnwee/subcults/internal/middleware"
	"github.com/onnwee/subcults/internal/stream"
)

// mockQualityMetricsRepo is a test-only mock of stream.QualityMetricsRepository.
type mockQualityMetricsRepo struct {
	metrics       map[string][]*stream.QualityMetrics // keyed by session ID
	latestMetrics map[string]*stream.QualityMetrics   // keyed by "sessionID:participantID"
	highPLoss     map[string][]string                 // keyed by session ID
	recordErr     error
}

func newMockQualityMetricsRepo() *mockQualityMetricsRepo {
	return &mockQualityMetricsRepo{
		metrics:       make(map[string][]*stream.QualityMetrics),
		latestMetrics: make(map[string]*stream.QualityMetrics),
		highPLoss:     make(map[string][]string),
	}
}

func (m *mockQualityMetricsRepo) RecordMetrics(metrics *stream.QualityMetrics) error {
	if m.recordErr != nil {
		return m.recordErr
	}
	m.metrics[metrics.StreamSessionID] = append(m.metrics[metrics.StreamSessionID], metrics)
	return nil
}

func (m *mockQualityMetricsRepo) GetLatestMetrics(sessionID, participantID string) (*stream.QualityMetrics, error) {
	key := sessionID + ":" + participantID
	if met, ok := m.latestMetrics[key]; ok {
		return met, nil
	}
	return nil, stream.ErrQualityMetricsNotFound
}

func (m *mockQualityMetricsRepo) GetMetricsBySession(sessionID string, _ int) ([]*stream.QualityMetrics, error) {
	return m.metrics[sessionID], nil
}

func (m *mockQualityMetricsRepo) GetMetricsTimeSeries(_, _ string, _, _ time.Time) ([]*stream.QualityMetrics, error) {
	return nil, nil
}

func (m *mockQualityMetricsRepo) GetParticipantsWithHighPacketLoss(sessionID string, _ int) ([]string, error) {
	return m.highPLoss[sessionID], nil
}

func TestGetStreamQualityMetrics_Unauthorized(t *testing.T) {
	handler := NewQualityMetricsHandler(nil, newMockQualityMetricsRepo(), nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/streams/stream1/quality-metrics", nil)
	w := httptest.NewRecorder()

	handler.GetStreamQualityMetrics(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestGetStreamQualityMetrics_InvalidPath(t *testing.T) {
	handler := NewQualityMetricsHandler(nil, newMockQualityMetricsRepo(), nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/streams/quality-metrics", nil)
	ctx := middleware.SetUserDID(req.Context(), "did:plc:test")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	handler.GetStreamQualityMetrics(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestGetStreamQualityMetrics_Success(t *testing.T) {
	repo := newMockQualityMetricsRepo()
	bitrate := 128.0
	repo.metrics["stream1"] = []*stream.QualityMetrics{
		{StreamSessionID: "stream1", ParticipantID: "p1", BitrateKbps: &bitrate, MeasuredAt: time.Now()},
	}

	handler := NewQualityMetricsHandler(nil, repo, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/streams/stream1/quality-metrics", nil)
	ctx := middleware.SetUserDID(req.Context(), "did:plc:test")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	handler.GetStreamQualityMetrics(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if resp["stream_id"] != "stream1" {
		t.Errorf("expected stream_id stream1, got %v", resp["stream_id"])
	}
	if int(resp["count"].(float64)) != 1 {
		t.Errorf("expected count 1, got %v", resp["count"])
	}
}

func TestGetStreamQualityMetrics_WithLimit(t *testing.T) {
	repo := newMockQualityMetricsRepo()
	handler := NewQualityMetricsHandler(nil, repo, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/streams/stream1/quality-metrics?limit=50", nil)
	ctx := middleware.SetUserDID(req.Context(), "did:plc:test")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	handler.GetStreamQualityMetrics(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestGetParticipantQualityMetrics_Unauthorized(t *testing.T) {
	handler := NewQualityMetricsHandler(nil, newMockQualityMetricsRepo(), nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/streams/stream1/participants/p1/quality-metrics", nil)
	w := httptest.NewRecorder()

	handler.GetParticipantQualityMetrics(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestGetParticipantQualityMetrics_InvalidPath(t *testing.T) {
	handler := NewQualityMetricsHandler(nil, newMockQualityMetricsRepo(), nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/streams/stream1/quality-metrics", nil)
	ctx := middleware.SetUserDID(req.Context(), "did:plc:test")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	handler.GetParticipantQualityMetrics(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestGetParticipantQualityMetrics_NotFound(t *testing.T) {
	handler := NewQualityMetricsHandler(nil, newMockQualityMetricsRepo(), nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/streams/stream1/participants/p1/quality-metrics", nil)
	ctx := middleware.SetUserDID(req.Context(), "did:plc:test")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	handler.GetParticipantQualityMetrics(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestGetParticipantQualityMetrics_Success(t *testing.T) {
	repo := newMockQualityMetricsRepo()
	bitrate := 128.0
	repo.latestMetrics["stream1:p1"] = &stream.QualityMetrics{
		StreamSessionID: "stream1",
		ParticipantID:   "p1",
		BitrateKbps:     &bitrate,
		MeasuredAt:      time.Now(),
	}

	handler := NewQualityMetricsHandler(nil, repo, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/streams/stream1/participants/p1/quality-metrics", nil)
	ctx := middleware.SetUserDID(req.Context(), "did:plc:test")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	handler.GetParticipantQualityMetrics(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCollectStreamQualityMetrics_Unauthorized(t *testing.T) {
	handler := NewQualityMetricsHandler(nil, newMockQualityMetricsRepo(), nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/streams/stream1/quality-metrics/collect", nil)
	w := httptest.NewRecorder()

	handler.CollectStreamQualityMetrics(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestCollectStreamQualityMetrics_InvalidPath(t *testing.T) {
	handler := NewQualityMetricsHandler(nil, newMockQualityMetricsRepo(), nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/streams//quality-metrics/collect", nil)
	ctx := middleware.SetUserDID(req.Context(), "did:plc:test")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	handler.CollectStreamQualityMetrics(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestCollectStreamQualityMetrics_StreamNotFound(t *testing.T) {
	sessionRepo := stream.NewInMemorySessionRepository()
	handler := NewQualityMetricsHandler(nil, newMockQualityMetricsRepo(), sessionRepo, nil)

	req := httptest.NewRequest(http.MethodPost, "/streams/nonexistent/quality-metrics/collect", nil)
	ctx := middleware.SetUserDID(req.Context(), "did:plc:test")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	handler.CollectStreamQualityMetrics(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCollectStreamQualityMetrics_NotHost(t *testing.T) {
	sessionRepo := stream.NewInMemorySessionRepository()
	sceneID := "scene1"
	id, _, _ := sessionRepo.CreateStreamSession(&sceneID, nil, "did:plc:host")

	handler := NewQualityMetricsHandler(nil, newMockQualityMetricsRepo(), sessionRepo, nil)

	req := httptest.NewRequest(http.MethodPost, "/streams/"+id+"/quality-metrics/collect", nil)
	ctx := middleware.SetUserDID(req.Context(), "did:plc:nothost")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	handler.CollectStreamQualityMetrics(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCollectStreamQualityMetrics_StreamEnded(t *testing.T) {
	sessionRepo := stream.NewInMemorySessionRepository()
	sceneID := "scene1"
	id, _, _ := sessionRepo.CreateStreamSession(&sceneID, nil, "did:plc:host")
	_ = sessionRepo.EndStreamSession(id)

	handler := NewQualityMetricsHandler(nil, newMockQualityMetricsRepo(), sessionRepo, nil)

	req := httptest.NewRequest(http.MethodPost, "/streams/"+id+"/quality-metrics/collect", nil)
	ctx := middleware.SetUserDID(req.Context(), "did:plc:host")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	handler.CollectStreamQualityMetrics(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCollectStreamQualityMetrics_NoRoomService(t *testing.T) {
	sessionRepo := stream.NewInMemorySessionRepository()
	sceneID := "scene1"
	id, _, _ := sessionRepo.CreateStreamSession(&sceneID, nil, "did:plc:host")

	// nil room service
	handler := NewQualityMetricsHandler(nil, newMockQualityMetricsRepo(), sessionRepo, nil)

	req := httptest.NewRequest(http.MethodPost, "/streams/"+id+"/quality-metrics/collect", nil)
	ctx := middleware.SetUserDID(req.Context(), "did:plc:host")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	handler.CollectStreamQualityMetrics(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetHighPacketLossParticipants_Unauthorized(t *testing.T) {
	handler := NewQualityMetricsHandler(nil, newMockQualityMetricsRepo(), nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/streams/stream1/quality-metrics/high-packet-loss", nil)
	w := httptest.NewRecorder()

	handler.GetHighPacketLossParticipants(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestGetHighPacketLossParticipants_InvalidPath(t *testing.T) {
	handler := NewQualityMetricsHandler(nil, newMockQualityMetricsRepo(), nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/streams//quality-metrics/high-packet-loss", nil)
	ctx := middleware.SetUserDID(req.Context(), "did:plc:test")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	handler.GetHighPacketLossParticipants(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestGetHighPacketLossParticipants_Success(t *testing.T) {
	repo := newMockQualityMetricsRepo()
	repo.highPLoss["stream1"] = []string{"p1", "p2"}

	handler := NewQualityMetricsHandler(nil, repo, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/streams/stream1/quality-metrics/high-packet-loss", nil)
	ctx := middleware.SetUserDID(req.Context(), "did:plc:test")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	handler.GetHighPacketLossParticipants(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if int(resp["count"].(float64)) != 2 {
		t.Errorf("expected count 2, got %v", resp["count"])
	}
}

func TestGetHighPacketLossParticipants_WithSinceParam(t *testing.T) {
	repo := newMockQualityMetricsRepo()
	handler := NewQualityMetricsHandler(nil, repo, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/streams/stream1/quality-metrics/high-packet-loss?since_minutes=10", nil)
	ctx := middleware.SetUserDID(req.Context(), "did:plc:test")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	handler.GetHighPacketLossParticipants(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}
