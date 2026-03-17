package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/onnwee/subcults/internal/telemetry"
)

func TestTelemetryHandlers_PostMetrics(t *testing.T) {
	handler := NewTelemetryHandlers(telemetry.NewInMemoryStore(), telemetry.NewMetrics())

	tests := []struct {
		name           string
		method         string
		body           interface{}
		expectedStatus int
		expectedError  string
	}{
		{
			name:   "valid metrics submission",
			method: http.MethodPost,
			body: TelemetryMetricsRequest{
				Metrics: []PerformanceMetric{
					{
						Name:           "LCP",
						Value:          1234.56,
						Rating:         "good",
						Delta:          1234.56,
						ID:             "test-id-1",
						NavigationType: "navigate",
						Timestamp:      time.Now().UnixMilli(),
					},
					{
						Name:           "FCP",
						Value:          789.01,
						Rating:         "good",
						Delta:          789.01,
						ID:             "test-id-2",
						NavigationType: "navigate",
						Timestamp:      time.Now().UnixMilli(),
					},
				},
				UserAgent: "Mozilla/5.0 Test Browser",
				URL:       "https://example.com/test",
			},
			expectedStatus: http.StatusAccepted,
		},
		{
			name:   "single metric",
			method: http.MethodPost,
			body: TelemetryMetricsRequest{
				Metrics: []PerformanceMetric{
					{
						Name:           "CLS",
						Value:          0.05,
						Rating:         "good",
						Delta:          0.05,
						ID:             "test-id-3",
						NavigationType: "navigate",
						Timestamp:      time.Now().UnixMilli(),
					},
				},
				UserAgent: "Mozilla/5.0 Test Browser",
				URL:       "https://example.com/test",
			},
			expectedStatus: http.StatusAccepted,
		},
		{
			name:           "invalid method GET",
			method:         http.MethodGet,
			body:           nil,
			expectedStatus: http.StatusMethodNotAllowed,
			expectedError:  ErrCodeBadRequest,
		},
		{
			name:           "invalid method PUT",
			method:         http.MethodPut,
			body:           nil,
			expectedStatus: http.StatusMethodNotAllowed,
			expectedError:  ErrCodeBadRequest,
		},
		{
			name:           "empty metrics array",
			method:         http.MethodPost,
			body:           TelemetryMetricsRequest{Metrics: []PerformanceMetric{}},
			expectedStatus: http.StatusBadRequest,
			expectedError:  ErrCodeBadRequest,
		},
		{
			name:           "invalid JSON body",
			method:         http.MethodPost,
			body:           "invalid json",
			expectedStatus: http.StatusBadRequest,
			expectedError:  ErrCodeBadRequest,
		},
		{
			name:           "malformed JSON",
			method:         http.MethodPost,
			body:           []byte(`{"metrics": [invalid]}`),
			expectedStatus: http.StatusBadRequest,
			expectedError:  ErrCodeBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var reqBody []byte
			var err error

			if tt.body != nil {
				switch v := tt.body.(type) {
				case []byte:
					reqBody = v
				case string:
					reqBody = []byte(v)
				default:
					reqBody, err = json.Marshal(tt.body)
					if err != nil {
						t.Fatalf("failed to marshal request body: %v", err)
					}
				}
			}

			req := httptest.NewRequest(tt.method, "/api/telemetry/metrics", bytes.NewBuffer(reqBody))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			handler.PostMetrics(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedStatus == http.StatusAccepted {
				var response map[string]interface{}
				if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}

				if status, ok := response["status"].(string); !ok || status != "accepted" {
					t.Errorf("expected status 'accepted', got %v", response["status"])
				}

				metricsReceived, ok := response["metrics_received"].(float64)
				if !ok {
					t.Errorf("expected metrics_received to be a number, got %v", response["metrics_received"])
				}

				expectedMetricsCount := len(tt.body.(TelemetryMetricsRequest).Metrics)
				if int(metricsReceived) != expectedMetricsCount {
					t.Errorf("expected %d metrics received, got %d", expectedMetricsCount, int(metricsReceived))
				}
			}

			if tt.expectedError != "" {
				var errResp ErrorResponse
				if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
					t.Fatalf("failed to decode error response: %v", err)
				}

				if errResp.Error.Code != tt.expectedError {
					t.Errorf("expected error code %s, got %s", tt.expectedError, errResp.Error.Code)
				}
			}
		})
	}
}

func TestTelemetryHandlers_PostMetrics_MetricValidation(t *testing.T) {
	handler := NewTelemetryHandlers(telemetry.NewInMemoryStore(), telemetry.NewMetrics())

	tests := []struct {
		name           string
		metric         PerformanceMetric
		expectedStatus int
	}{
		{
			name: "valid LCP metric",
			metric: PerformanceMetric{
				Name:           "LCP",
				Value:          2345.67,
				Rating:         "needs-improvement",
				Delta:          2345.67,
				ID:             "lcp-test-id",
				NavigationType: "navigate",
				Timestamp:      time.Now().UnixMilli(),
			},
			expectedStatus: http.StatusAccepted,
		},
		{
			name: "valid CLS metric with small value",
			metric: PerformanceMetric{
				Name:           "CLS",
				Value:          0.001,
				Rating:         "good",
				Delta:          0.001,
				ID:             "cls-test-id",
				NavigationType: "reload",
				Timestamp:      time.Now().UnixMilli(),
			},
			expectedStatus: http.StatusAccepted,
		},
		{
			name: "custom metric",
			metric: PerformanceMetric{
				Name:           "custom-metric",
				Value:          999.99,
				Rating:         "good",
				Delta:          999.99,
				ID:             "custom-test-id",
				NavigationType: "custom",
				Timestamp:      time.Now().UnixMilli(),
			},
			expectedStatus: http.StatusAccepted,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reqBody := TelemetryMetricsRequest{
				Metrics:   []PerformanceMetric{tt.metric},
				UserAgent: "Test Agent",
				URL:       "https://test.example.com",
			}

			body, err := json.Marshal(reqBody)
			if err != nil {
				t.Fatalf("failed to marshal request: %v", err)
			}

			req := httptest.NewRequest(http.MethodPost, "/api/telemetry/metrics", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			handler.PostMetrics(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}
