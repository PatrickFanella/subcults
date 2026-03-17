package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/onnwee/subcults/internal/middleware"
	"github.com/onnwee/subcults/internal/retention"
)

// AccountHandlers provides endpoints for user account data export and deletion.
type AccountHandlers struct {
	repo        retention.Repository
	gracePeriod time.Duration
}

// NewAccountHandlers creates new account handler instances.
func NewAccountHandlers(repo retention.Repository, gracePeriod time.Duration) *AccountHandlers {
	if gracePeriod == 0 {
		gracePeriod = 30 * 24 * time.Hour
	}
	return &AccountHandlers{
		repo:        repo,
		gracePeriod: gracePeriod,
	}
}

// ExportAccountData handles GET /api/account/export
// Returns all personal data for the authenticated user as JSON.
func (h *AccountHandlers) ExportAccountData(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		WriteError(w, r.Context(), http.StatusMethodNotAllowed, "method_not_allowed", "Only GET is allowed")
		return
	}

	userDID := middleware.GetUserDID(r.Context())
	if userDID == "" {
		WriteError(w, r.Context(), http.StatusUnauthorized, ErrCodeUnauthorized, "Authentication required")
		return
	}

	export, err := h.repo.ExportUserData(r.Context(), userDID)
	if err != nil {
		WriteError(w, r.Context(), http.StatusInternalServerError, "export_failed", "Failed to export user data")
		return
	}

	export.ExportedAt = time.Now().UTC()

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", "attachment; filename=\"subcults-data-export.json\"")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(export); err != nil {
		// Response already started, can't change status
		return
	}
}

// DeleteAccountRequest represents the request to delete an account.
type DeleteAccountRequest struct {
	Confirm bool `json:"confirm"`
}

// DeleteAccount handles POST /api/account/delete
// Schedules account deletion with a grace period.
func (h *AccountHandlers) DeleteAccount(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		WriteError(w, r.Context(), http.StatusMethodNotAllowed, "method_not_allowed", "Only POST is allowed")
		return
	}

	userDID := middleware.GetUserDID(r.Context())
	if userDID == "" {
		WriteError(w, r.Context(), http.StatusUnauthorized, ErrCodeUnauthorized, "Authentication required")
		return
	}

	var req DeleteAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, r.Context(), http.StatusBadRequest, ErrCodeBadRequest, "Invalid request body")
		return
	}

	if !req.Confirm {
		WriteError(w, r.Context(), http.StatusBadRequest, ErrCodeBadRequest,
			"Account deletion requires confirm: true")
		return
	}

	graceEnd := time.Now().Add(h.gracePeriod)
	if err := h.repo.ScheduleAccountDeletion(r.Context(), userDID, graceEnd); err != nil {
		WriteError(w, r.Context(), http.StatusInternalServerError, "deletion_failed",
			"Failed to schedule account deletion")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	resp := map[string]interface{}{
		"message":       "Account deletion scheduled",
		"grace_ends_at": graceEnd.UTC().Format(time.RFC3339),
		"note":          "You can cancel deletion by logging in before the grace period ends",
	}
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		return
	}
}
