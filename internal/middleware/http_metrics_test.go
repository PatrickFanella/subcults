package middleware

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

func TestHTTPMetrics(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		path           string
		requestBody    string
		responseStatus int
		responseBody   string
		wantMetrics    bool // false if health check endpoint
	}{
		{
			name:           "GET request",
			method:         http.MethodGet,
			path:           "/events",
			requestBody:    "",
			responseStatus: http.StatusOK,
			responseBody:   `{"events":[]}`,
			wantMetrics:    true,
		},
		{
			name:           "POST request with body",
			method:         http.MethodPost,
			path:           "/events",
			requestBody:    `{"title":"Test Event"}`,
			responseStatus: http.StatusCreated,
			responseBody:   `{"id":"123"}`,
			wantMetrics:    true,
		},
		{
			name:           "404 error",
			method:         http.MethodGet,
			path:           "/notfound",
			requestBody:    "",
			responseStatus: http.StatusNotFound,
			responseBody:   `{"error":"not found"}`,
			wantMetrics:    true,
		},
		{
			name:           "Health check excluded",
			method:         http.MethodGet,
			path:           "/health",
			requestBody:    "",
			responseStatus: http.StatusOK,
			responseBody:   `{"status":"ok"}`,
			wantMetrics:    false,
		},
		{
			name:           "Ready check excluded",
			method:         http.MethodGet,
			path:           "/ready",
			requestBody:    "",
			responseStatus: http.StatusOK,
			responseBody:   `{"ready":true}`,
			wantMetrics:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create metrics
			m := NewMetrics()
			reg := prometheus.NewRegistry()
			if err := m.Register(reg); err != nil {
				t.Fatalf("Register() failed: %v", err)
			}

			// Create test handler
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.responseStatus)
				_, _ = w.Write([]byte(tt.responseBody))
			})

			// Wrap with metrics middleware
			wrapped := HTTPMetrics(m)(handler)

			// Create request
			var body io.Reader
			if tt.requestBody != "" {
				body = strings.NewReader(tt.requestBody)
			}
			req := httptest.NewRequest(tt.method, tt.path, body)
			if tt.requestBody != "" {
				req.Header.Set("Content-Length", strconv.Itoa(len(tt.requestBody)))
			}

			// Execute request
			rec := httptest.NewRecorder()
			wrapped.ServeHTTP(rec, req)

			// Verify response
			if rec.Code != tt.responseStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.responseStatus)
			}

			// Gather metrics
			metrics, err := reg.Gather()
			if err != nil {
				t.Fatalf("Gather() failed: %v", err)
			}

			// Check if metrics were recorded
			foundDuration := false
			foundTotal := false

			for _, mf := range metrics {
				if mf.GetName() == MetricHTTPRequestDuration {
					foundDuration = true
					if !tt.wantMetrics && len(mf.GetMetric()) > 0 {
						t.Errorf("expected no duration metrics for %s, but found some", tt.path)
					}
				}
				if mf.GetName() == MetricHTTPRequestsTotal {
					foundTotal = true
					if !tt.wantMetrics && len(mf.GetMetric()) > 0 {
						t.Errorf("expected no counter metrics for %s, but found some", tt.path)
					}
				}
			}

			if tt.wantMetrics {
				if !foundDuration {
					t.Error("duration metric not found")
				}
				if !foundTotal {
					t.Error("total metric not found")
				}
			}
		})
	}
}

func TestHTTPMetrics_Labels(t *testing.T) {
	// Create metrics
	m := NewMetrics()
	reg := prometheus.NewRegistry()
	if err := m.Register(reg); err != nil {
		t.Fatalf("Register() failed: %v", err)
	}

	// Create test handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	// Wrap with metrics middleware
	wrapped := HTTPMetrics(m)(handler)

	// Execute request
	req := httptest.NewRequest(http.MethodGet, "/events", nil)
	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)

	// Gather metrics
	metrics, err := reg.Gather()
	if err != nil {
		t.Fatalf("Gather() failed: %v", err)
	}

	// Find the total counter metric
	var totalMetric *dto.MetricFamily
	for i := range metrics {
		if metrics[i].GetName() == MetricHTTPRequestsTotal {
			totalMetric = metrics[i]
			break
		}
	}

	if totalMetric == nil {
		t.Fatal("total metric not found")
	}

	// Verify labels
	if len(totalMetric.GetMetric()) != 1 {
		t.Fatalf("expected 1 metric entry, got %d", len(totalMetric.GetMetric()))
	}

	metric := totalMetric.GetMetric()[0]
	labels := metric.GetLabel()

	// Check label values
	labelMap := make(map[string]string)
	for _, label := range labels {
		labelMap[label.GetName()] = label.GetValue()
	}

	if labelMap["method"] != "GET" {
		t.Errorf("method label = %s, want GET", labelMap["method"])
	}
	if labelMap["path"] != "/events" {
		t.Errorf("path label = %s, want /events", labelMap["path"])
	}
	if labelMap["status"] != "200" {
		t.Errorf("status label = %s, want 200", labelMap["status"])
	}
}

func TestHTTPMetrics_ResponseSize(t *testing.T) {
	// Create metrics
	m := NewMetrics()
	reg := prometheus.NewRegistry()
	if err := m.Register(reg); err != nil {
		t.Fatalf("Register() failed: %v", err)
	}

	responseBody := "This is a test response"
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(responseBody))
	})

	// Wrap with metrics middleware
	wrapped := HTTPMetrics(m)(handler)

	// Execute request
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)

	// Gather metrics
	metrics, err := reg.Gather()
	if err != nil {
		t.Fatalf("Gather() failed: %v", err)
	}

	// Find the response size metric
	var responseSizeMetric *dto.MetricFamily
	for i := range metrics {
		if metrics[i].GetName() == MetricHTTPResponseSizeBytes {
			responseSizeMetric = metrics[i]
			break
		}
	}

	if responseSizeMetric == nil {
		t.Fatal("response size metric not found")
	}

	// Verify that the metric recorded observations
	if len(responseSizeMetric.GetMetric()) != 1 {
		t.Fatalf("expected 1 metric entry, got %d", len(responseSizeMetric.GetMetric()))
	}

	// Verify response size was recorded
	histogram := responseSizeMetric.GetMetric()[0].GetHistogram()
	if histogram == nil {
		t.Fatal("expected histogram, got nil")
	}

	if histogram.GetSampleCount() != 1 {
		t.Errorf("sample count = %d, want 1", histogram.GetSampleCount())
	}

	// The sum should equal the response body size
	expectedSize := float64(len(responseBody))
	if histogram.GetSampleSum() != expectedSize {
		t.Errorf("sample sum = %f, want %f", histogram.GetSampleSum(), expectedSize)
	}
}

func TestMetricsResponseWriter_MultipleWrites(t *testing.T) {
	rec := httptest.NewRecorder()
	mrw := newMetricsResponseWriter(rec)

	// Write multiple times
	n1, err := mrw.Write([]byte("Hello "))
	if err != nil {
		t.Fatalf("Write() failed: %v", err)
	}
	n2, err := mrw.Write([]byte("World"))
	if err != nil {
		t.Fatalf("Write() failed: %v", err)
	}

	expectedSize := int64(n1 + n2)
	if mrw.size != expectedSize {
		t.Errorf("size = %d, want %d", mrw.size, expectedSize)
	}
}

func TestMetricsResponseWriter_WriteHeaderOnce(t *testing.T) {
	rec := httptest.NewRecorder()
	mrw := newMetricsResponseWriter(rec)

	// Call WriteHeader multiple times
	mrw.WriteHeader(http.StatusCreated)
	mrw.WriteHeader(http.StatusInternalServerError) // Should be ignored

	if mrw.statusCode != http.StatusCreated {
		t.Errorf("statusCode = %d, want %d", mrw.statusCode, http.StatusCreated)
	}
}

func TestObserveHTTPRequest(t *testing.T) {
	m := NewMetrics()
	reg := prometheus.NewRegistry()
	if err := m.Register(reg); err != nil {
		t.Fatalf("Register() failed: %v", err)
	}

	// Record some observations
	m.ObserveHTTPRequest("GET", "/events", "200", 0.123, 100, 500)
	m.ObserveHTTPRequest("POST", "/events", "201", 0.456, 200, 300)
	m.ObserveHTTPRequest("GET", "/events", "200", 0.789, 150, 600)

	// Gather metrics
	metrics, err := reg.Gather()
	if err != nil {
		t.Fatalf("Gather() failed: %v", err)
	}

	// Verify all 4 metrics are present
	metricNames := map[string]bool{
		MetricHTTPRequestDuration:   false,
		MetricHTTPRequestsTotal:     false,
		MetricHTTPRequestSizeBytes:  false,
		MetricHTTPResponseSizeBytes: false,
	}

	for _, mf := range metrics {
		if _, ok := metricNames[mf.GetName()]; ok {
			metricNames[mf.GetName()] = true
		}
	}

	for name, found := range metricNames {
		if !found {
			t.Errorf("metric %s not found", name)
		}
	}

	// Verify the counter has the right number of distinct label sets
	var totalMetric *dto.MetricFamily
	for i := range metrics {
		if metrics[i].GetName() == MetricHTTPRequestsTotal {
			totalMetric = metrics[i]
			break
		}
	}

	if totalMetric == nil {
		t.Fatal("total metric not found")
	}

	// Should have 2 distinct label sets: GET/200 and POST/201
	if len(totalMetric.GetMetric()) != 2 {
		t.Errorf("expected 2 label sets, got %d", len(totalMetric.GetMetric()))
	}
}
