package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCSPReportHandler_ValidReport(t *testing.T) {
	handler := CSPReportHandler()

	body := `{"csp-report":{"document-uri":"https://subcults.subcult.tv/","blocked-uri":"https://evil.com/script.js","violated-directive":"script-src 'self'","effective-directive":"script-src","original-policy":"default-src 'self'","status-code":200}}`

	req := httptest.NewRequest(http.MethodPost, "/api/csp-report", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/csp-report")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
}

func TestCSPReportHandler_InvalidJSON(t *testing.T) {
	handler := CSPReportHandler()

	req := httptest.NewRequest(http.MethodPost, "/api/csp-report", strings.NewReader("not json"))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestCSPReportHandler_WrongMethod(t *testing.T) {
	handler := CSPReportHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/csp-report", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}

func TestCSPReportHandler_EmptyBody(t *testing.T) {
	handler := CSPReportHandler()

	req := httptest.NewRequest(http.MethodPost, "/api/csp-report", strings.NewReader(""))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	// Empty body fails JSON unmarshal
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}
