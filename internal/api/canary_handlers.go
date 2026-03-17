// Package api provides HTTP handlers for canary deployment management.
package api

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/onnwee/subcults/internal/middleware"
)

// CanaryHandler handles canary deployment management endpoints.
type CanaryHandler struct {
	router *middleware.CanaryRouter
	logger *slog.Logger
}

// NewCanaryHandler creates a new canary handler.
func NewCanaryHandler(router *middleware.CanaryRouter, logger *slog.Logger) *CanaryHandler {
	return &CanaryHandler{
		router: router,
		logger: logger,
	}
}

// GetMetrics returns current canary deployment metrics.
// GET /canary/metrics
func (h *CanaryHandler) GetMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	snapshot := h.router.GetMetrics()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(snapshot); err != nil {
		h.logger.Error("failed to encode canary metrics", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

// Rollback triggers a manual rollback of the canary deployment.
// POST /canary/rollback
func (h *CanaryHandler) Rollback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse request body for rollback reason
	var req struct {
		Reason string `json:"reason"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		req.Reason = "manual_rollback" // Default reason
	}

	if req.Reason == "" {
		req.Reason = "manual_rollback"
	}

	h.router.Rollback(req.Reason)

	h.logger.Info("manual canary rollback triggered",
		"reason", req.Reason,
	)

	response := map[string]interface{}{
		"success": true,
		"message": "Canary deployment rolled back",
		"reason":  req.Reason,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error("failed to encode rollback response", "error", err)
	}
}

// ResetMetrics resets the canary metrics window.
// POST /canary/metrics/reset
func (h *CanaryHandler) ResetMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	h.router.ResetMetrics()

	h.logger.Info("canary metrics window reset")

	response := map[string]interface{}{
		"success": true,
		"message": "Canary metrics reset",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error("failed to encode reset response", "error", err)
	}
}
