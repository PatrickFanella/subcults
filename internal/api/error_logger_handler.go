package api

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/onnwee/subcults/internal/middleware"
	"github.com/onnwee/subcults/internal/telemetry"
)

// ErrorLoggerHandlers provides the endpoint for client-side error collection.
type ErrorLoggerHandlers struct {
	store   telemetry.Store
	metrics *telemetry.Metrics
}

// NewErrorLoggerHandlers creates a new error logger handler.
func NewErrorLoggerHandlers(store telemetry.Store, metrics *telemetry.Metrics) *ErrorLoggerHandlers {
	return &ErrorLoggerHandlers{
		store:   store,
		metrics: metrics,
	}
}

// Maximum number of replay events per error report.
const maxReplayEvents = 100

// HandleClientError handles POST /api/log/client-error.
// Accepts client-side error reports with optional session replay data.
// Always returns 200 OK — error logging must never fail the client.
func (h *ErrorLoggerHandlers) HandleClientError(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodPost {
		ctx = middleware.SetErrorCode(ctx, ErrCodeBadRequest)
		WriteError(w, ctx, http.StatusMethodNotAllowed, ErrCodeBadRequest, "Method not allowed")
		return
	}

	var errLog telemetry.ClientErrorLog
	if err := json.NewDecoder(r.Body).Decode(&errLog); err != nil {
		// Even on bad JSON, return 200 — frontend error logging is fire-and-forget
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"success": false})
		return
	}

	// Validate required fields
	if errLog.SessionID == "" || errLog.ErrorMessage == "" || errLog.ErrorType == "" || errLog.OccurredAt == 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"success": false})
		return
	}

	// Extract user DID from JWT context if authenticated
	userDID := middleware.GetUserDID(r.Context())
	if userDID != "" && errLog.UserDID == "" {
		errLog.UserDID = userDID
	}

	// Compute dedup hash: SHA256(session_id + error_message)
	hash := sha256.Sum256([]byte(errLog.SessionID + errLog.ErrorMessage))
	errLog.ErrorHash = hex.EncodeToString(hash[:])

	// Cap replay events to prevent oversized payloads
	replayEvents := errLog.ReplayEvents
	if len(replayEvents) > maxReplayEvents {
		replayEvents = replayEvents[:maxReplayEvents]
	}
	errLog.ReplayEvents = nil // don't store replay events inline

	// Record metrics
	h.metrics.ClientErrorsTotal.WithLabelValues(errLog.ErrorType).Inc()

	// Persist error log
	errorID, err := h.store.InsertClientError(ctx, errLog)
	if err != nil {
		if errors.Is(err, telemetry.ErrDuplicateError) {
			h.metrics.ClientErrorsDeduped.Inc()
			slog.DebugContext(ctx, "duplicate client error skipped",
				"session_id", errLog.SessionID,
				"error_type", errLog.ErrorType,
			)
		} else {
			slog.ErrorContext(ctx, "failed to persist client error",
				"error", err,
				"session_id", errLog.SessionID,
				"error_type", errLog.ErrorType,
			)
		}

		// Always return success
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
		})
		return
	}

	// Log the error for Loki (structured logging for observability)
	slog.InfoContext(ctx, "client_error_received",
		"error_id", errorID,
		"session_id", errLog.SessionID,
		"error_type", errLog.ErrorType,
		"url", errLog.URL,
		"replay_event_count", len(replayEvents),
	)

	// Persist replay events if present
	if len(replayEvents) > 0 {
		if err := h.store.InsertReplayEvents(ctx, errorID, replayEvents); err != nil {
			slog.ErrorContext(ctx, "failed to persist replay events",
				"error", err,
				"error_log_id", errorID,
				"replay_count", len(replayEvents),
			)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"errorId": errorID,
	})
}
