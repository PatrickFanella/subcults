// Package api provides HTTP API utilities including standardized error handling.
package api

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/onnwee/subcults/internal/middleware"
)

// Common error codes used throughout the API.
const (
	// ErrCodeValidation indicates input validation failure.
	ErrCodeValidation = "validation_error"

	// ErrCodeAuthFailed indicates authentication failure.
	ErrCodeAuthFailed = "auth_failed"

	// ErrCodeNotFound indicates the requested resource was not found.
	ErrCodeNotFound = "not_found"

	// ErrCodeRateLimited indicates rate limit exceeded.
	ErrCodeRateLimited = "rate_limited"

	// ErrCodeInternal indicates an internal server error.
	ErrCodeInternal = "internal_error"

	// ErrCodeForbidden indicates the request is forbidden.
	ErrCodeForbidden = "forbidden"

	// ErrCodeConflict indicates a conflict with the current state.
	ErrCodeConflict = "conflict"

	// ErrCodeBadRequest indicates a malformed request.
	ErrCodeBadRequest = "bad_request"

	// ErrCodeInvalidPalette indicates an invalid palette configuration.
	ErrCodeInvalidPalette = "invalid_palette"

	// ErrCodeSceneDeleted indicates the scene has been deleted.
	ErrCodeSceneDeleted = "scene_deleted"

	// ErrCodeInvalidSceneName indicates scene name validation failure.
	ErrCodeInvalidSceneName = "invalid_scene_name"

	// ErrCodeDuplicateSceneName indicates scene name already exists for owner.
	ErrCodeDuplicateSceneName = "duplicate_scene_name"

	// ErrCodeInvalidTimeRange indicates event start time is not before end time.
	ErrCodeInvalidTimeRange = "invalid_time_range"

	// ErrCodeMissingTarget indicates post must have at least one of scene_id or event_id.
	ErrCodeMissingTarget = "missing_target"

	// ErrCodeUnsupportedType indicates an unsupported content type for upload.
	ErrCodeUnsupportedType = "unsupported_type"

	// ErrCodeInvalidWeight indicates alliance weight must be between 0.0 and 1.0.
	ErrCodeInvalidWeight = "invalid_weight"

	// ErrCodeAllianceDeleted indicates the alliance has been deleted.
	ErrCodeAllianceDeleted = "alliance_deleted"

	// ErrCodeSelfAlliance indicates attempt to create alliance where from_scene_id == to_scene_id.
	ErrCodeSelfAlliance = "self_alliance"

	// ErrCodeSceneNotFound indicates the scene was not found.
	ErrCodeSceneNotFound = "scene_not_found"

	// ErrCodeUnauthorized is an alias of ErrCodeAuthFailed for 401-style auth errors.
	ErrCodeUnauthorized = ErrCodeAuthFailed

	// ErrCodePaymentNotFound indicates the payment record was not found.
	ErrCodePaymentNotFound = "payment_not_found"
)

// ErrorResponse represents the standard error response format.
// All API errors return JSON in this structure: {"error": {"code": "...", "message": "..."}}
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail contains the error code and human-readable message.
type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// WriteError writes a standardized JSON error response.
// It writes the appropriate HTTP status code and returns a JSON error body.
//
// Format: {"error": {"code": "error_code", "message": "Error description"}}
//
// The error_code will be automatically logged by the logging middleware
// for all 4xx and 5xx responses if you call SetErrorCode on the context
// and pass the updated context to WriteError.
//
// Example:
//
//	ctx := middleware.SetErrorCode(r.Context(), api.ErrCodeNotFound)
//	WriteError(w, ctx, http.StatusNotFound, api.ErrCodeNotFound, "Scene not found")
//
// Or in a handler with middleware:
//
//	func handler(w http.ResponseWriter, r *http.Request) {
//	    ctx := middleware.SetErrorCode(r.Context(), api.ErrCodeNotFound)
//	    api.WriteError(w, ctx, http.StatusNotFound, api.ErrCodeNotFound, "Scene not found")
//	}
func WriteError(w http.ResponseWriter, ctx context.Context, status int, code, message string) {
	// Update the context in the response writer if supported (for logging middleware)
	middleware.UpdateResponseContext(w, ctx)

	// Create error response
	errResp := ErrorResponse{
		Error: ErrorDetail{
			Code:    code,
			Message: message,
		},
	}

	// Marshal to JSON
	data, err := json.Marshal(errResp)
	if err != nil {
		// Fallback to plain text if JSON marshaling fails
		slog.ErrorContext(ctx, "failed to marshal error response", "error", err)
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("Internal server error"))
		return
	}

	// Write response
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if _, err := w.Write(data); err != nil {
		slog.ErrorContext(ctx, "failed to write error response", "error", err)
	}
}

// StatusCodeMapping returns the recommended HTTP status code for common error codes.
// This is a convenience function to map error codes to HTTP status codes.
func StatusCodeMapping(code string) int {
	switch code {
	case ErrCodeValidation:
		return http.StatusBadRequest
	case ErrCodeAuthFailed:
		return http.StatusUnauthorized
	case ErrCodeNotFound:
		return http.StatusNotFound
	case ErrCodeRateLimited:
		return http.StatusTooManyRequests
	case ErrCodeForbidden:
		return http.StatusForbidden
	case ErrCodeConflict:
		return http.StatusConflict
	case ErrCodeBadRequest:
		return http.StatusBadRequest
	case ErrCodeInternal:
		return http.StatusInternalServerError
	default:
		return http.StatusInternalServerError
	}
}
