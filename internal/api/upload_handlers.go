// Package api provides HTTP handlers for upload operations.
package api

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/onnwee/subcults/internal/middleware"
	"github.com/onnwee/subcults/internal/upload"
)

// SignUploadRequest represents the request body for POST /uploads/sign.
type SignUploadRequest struct {
	ContentType string  `json:"contentType"`
	SizeBytes   int64   `json:"sizeBytes"`
	PostID      *string `json:"postId,omitempty"`
}

// SignUploadResponse represents the response for POST /uploads/sign.
type SignUploadResponse struct {
	URL       string `json:"url"`
	Key       string `json:"key"`
	ExpiresAt string `json:"expiresAt"` // ISO 8601 format
}

// UploadHandlers holds dependencies for upload HTTP handlers.
type UploadHandlers struct {
	uploadService *upload.Service
}

// NewUploadHandlers creates a new UploadHandlers instance.
func NewUploadHandlers(uploadService *upload.Service) *UploadHandlers {
	return &UploadHandlers{
		uploadService: uploadService,
	}
}

// SignUpload handles POST /uploads/sign - generates a pre-signed upload URL.
func (h *UploadHandlers) SignUpload(w http.ResponseWriter, r *http.Request) {
	// Parse request body
	var req SignUploadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeBadRequest)
		WriteError(w, ctx, http.StatusBadRequest, ErrCodeBadRequest, "Invalid JSON in request body")
		return
	}

	// Validate content type is provided
	if req.ContentType == "" {
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeValidation)
		WriteError(w, ctx, http.StatusBadRequest, ErrCodeValidation, "contentType is required")
		return
	}

	// Validate size is provided and positive
	if req.SizeBytes <= 0 {
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeValidation)
		WriteError(w, ctx, http.StatusBadRequest, ErrCodeValidation, "sizeBytes must be positive")
		return
	}

	// Generate signed URL
	signedURL, err := h.uploadService.GenerateSignedURL(r.Context(), upload.SignedURLRequest{
		ContentType: req.ContentType,
		SizeBytes:   req.SizeBytes,
		PostID:      req.PostID,
	})

	if err != nil {
		// Handle specific error types
		switch err {
		case upload.ErrUnsupportedType:
			ctx := middleware.SetErrorCode(r.Context(), ErrCodeUnsupportedType)
			WriteError(w, ctx, http.StatusBadRequest, ErrCodeUnsupportedType,
				"Unsupported content type. Allowed types: image/jpeg, image/png, audio/mpeg, audio/wav")
			return
		case upload.ErrFileTooLarge:
			ctx := middleware.SetErrorCode(r.Context(), ErrCodeValidation)
			WriteError(w, ctx, http.StatusBadRequest, ErrCodeValidation, "File size exceeds maximum allowed")
			return
		case upload.ErrInvalidPostID:
			ctx := middleware.SetErrorCode(r.Context(), ErrCodeValidation)
			WriteError(w, ctx, http.StatusBadRequest, ErrCodeValidation, "Invalid post ID")
			return
		default:
			slog.ErrorContext(r.Context(), "failed to generate signed URL", "error", err)
			ctx := middleware.SetErrorCode(r.Context(), ErrCodeInternal)
			WriteError(w, ctx, http.StatusInternalServerError, ErrCodeInternal, "Failed to generate signed URL")
			return
		}
	}

	// Return signed URL response
	response := SignUploadResponse{
		URL:       signedURL.URL,
		Key:       signedURL.Key,
		ExpiresAt: signedURL.ExpiresAt.Format("2006-01-02T15:04:05Z07:00"), // ISO 8601
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		slog.ErrorContext(r.Context(), "failed to encode response", "error", err)
		return
	}
}
