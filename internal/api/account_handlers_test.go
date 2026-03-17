package api

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/onnwee/subcults/internal/middleware"
	"github.com/onnwee/subcults/internal/retention"
)

func newAccountTestHandlers() (*AccountHandlers, *retention.InMemoryRepository) {
	repo := retention.NewInMemoryRepository(slog.Default())
	handlers := NewAccountHandlers(repo, 30*24*time.Hour)
	return handlers, repo
}

func TestExportAccountData_Success(t *testing.T) {
	handlers, repo := newAccountTestHandlers()

	repo.AddUserExport("did:plc:test123", &retention.UserDataExport{
		UserDID: "did:plc:test123",
		Scenes:  []map[string]interface{}{{"name": "Test Scene"}},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/account/export", nil)
	ctx := middleware.SetUserDID(req.Context(), "did:plc:test123")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	handlers.ExportAccountData(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	contentDisp := w.Header().Get("Content-Disposition")
	if contentDisp != `attachment; filename="subcults-data-export.json"` {
		t.Errorf("expected Content-Disposition header, got %q", contentDisp)
	}

	var export retention.UserDataExport
	if err := json.NewDecoder(w.Body).Decode(&export); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if export.UserDID != "did:plc:test123" {
		t.Errorf("expected user DID did:plc:test123, got %s", export.UserDID)
	}
	if len(export.Scenes) != 1 {
		t.Errorf("expected 1 scene, got %d", len(export.Scenes))
	}
}

func TestExportAccountData_Unauthorized(t *testing.T) {
	handlers, _ := newAccountTestHandlers()

	req := httptest.NewRequest(http.MethodGet, "/api/account/export", nil)
	w := httptest.NewRecorder()

	handlers.ExportAccountData(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}
}

func TestExportAccountData_WrongMethod(t *testing.T) {
	handlers, _ := newAccountTestHandlers()

	req := httptest.NewRequest(http.MethodPost, "/api/account/export", nil)
	w := httptest.NewRecorder()

	handlers.ExportAccountData(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", w.Code)
	}
}

func TestDeleteAccount_Success(t *testing.T) {
	handlers, repo := newAccountTestHandlers()

	body, _ := json.Marshal(DeleteAccountRequest{Confirm: true})
	req := httptest.NewRequest(http.MethodPost, "/api/account/delete", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	ctx := middleware.SetUserDID(req.Context(), "did:plc:test123")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	handlers.DeleteAccount(w, req)

	if w.Code != http.StatusAccepted {
		t.Errorf("expected status 202, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp["message"] != "Account deletion scheduled" {
		t.Errorf("unexpected message: %v", resp["message"])
	}

	// Verify deletion was scheduled in repo
	pending, _ := repo.GetPendingDeletions(nil)
	if len(pending) != 1 {
		t.Errorf("expected 1 pending deletion, got %d", len(pending))
	}
}

func TestDeleteAccount_Unauthorized(t *testing.T) {
	handlers, _ := newAccountTestHandlers()

	body, _ := json.Marshal(DeleteAccountRequest{Confirm: true})
	req := httptest.NewRequest(http.MethodPost, "/api/account/delete", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handlers.DeleteAccount(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}
}

func TestDeleteAccount_WrongMethod(t *testing.T) {
	handlers, _ := newAccountTestHandlers()

	req := httptest.NewRequest(http.MethodGet, "/api/account/delete", nil)
	w := httptest.NewRecorder()

	handlers.DeleteAccount(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", w.Code)
	}
}

func TestDeleteAccount_NoConfirm(t *testing.T) {
	handlers, _ := newAccountTestHandlers()

	body, _ := json.Marshal(DeleteAccountRequest{Confirm: false})
	req := httptest.NewRequest(http.MethodPost, "/api/account/delete", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	ctx := middleware.SetUserDID(req.Context(), "did:plc:test123")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	handlers.DeleteAccount(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDeleteAccount_InvalidBody(t *testing.T) {
	handlers, _ := newAccountTestHandlers()

	req := httptest.NewRequest(http.MethodPost, "/api/account/delete", bytes.NewReader([]byte("invalid")))
	req.Header.Set("Content-Type", "application/json")
	ctx := middleware.SetUserDID(req.Context(), "did:plc:test123")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	handlers.DeleteAccount(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestNewAccountHandlers_DefaultGracePeriod(t *testing.T) {
	repo := retention.NewInMemoryRepository(slog.Default())
	handlers := NewAccountHandlers(repo, 0) // 0 means use default

	if handlers.gracePeriod != 30*24*time.Hour {
		t.Errorf("expected default grace period of 30 days, got %v", handlers.gracePeriod)
	}
}
