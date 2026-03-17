// Package api provides HTTP handlers for the Subcults API.
package api

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/onnwee/subcults/internal/attachment"
	"github.com/onnwee/subcults/internal/membership"
	"github.com/onnwee/subcults/internal/middleware"
	"github.com/onnwee/subcults/internal/post"
	"github.com/onnwee/subcults/internal/scene"
	"github.com/onnwee/subcults/internal/validate"
)

// Post text validation constraints
const (
	MaxPostTextLength = 5000
	MaxAttachments    = 6
)

// PostHandlers holds dependencies for post HTTP handlers.
type PostHandlers struct {
	repo            post.PostRepository
	sceneRepo       scene.SceneRepository
	membershipRepo  membership.MembershipRepository
	metadataService *attachment.MetadataService // Optional: for enriching attachment metadata
}

// NewPostHandlers creates a new PostHandlers instance.
// metadataService is optional and can be nil if attachment enrichment is not configured.
//
// Note: This constructor was updated to include metadataService as a new parameter.
// All existing callers have been updated. The service is optional (can be nil) to
// maintain graceful degradation when R2 is not configured.
func NewPostHandlers(repo post.PostRepository, sceneRepo scene.SceneRepository, membershipRepo membership.MembershipRepository, metadataService *attachment.MetadataService) *PostHandlers {
	return &PostHandlers{
		repo:            repo,
		sceneRepo:       sceneRepo,
		membershipRepo:  membershipRepo,
		metadataService: metadataService,
	}
}

// CreatePostRequest represents the request body for creating a post.
type CreatePostRequest struct {
	SceneID     *string           `json:"scene_id,omitempty"`
	EventID     *string           `json:"event_id,omitempty"`
	Text        string            `json:"text"`
	Attachments []post.Attachment `json:"attachments,omitempty"`
	Labels      []string          `json:"labels,omitempty"`
}

// UpdatePostRequest represents the request body for updating a post.
type UpdatePostRequest struct {
	Text        *string            `json:"text,omitempty"`
	Attachments *[]post.Attachment `json:"attachments,omitempty"`
	Labels      *[]string          `json:"labels,omitempty"`
}

// validatePostAttachments validates post attachments, including URL validation for SSRF protection.
func validatePostAttachments(attachments []post.Attachment) error {
	for i, att := range attachments {
		// Validate legacy URL field if present
		if att.URL != "" {
			if _, err := validate.MediaURL(att.URL); err != nil {
				return fmt.Errorf("attachment %d: invalid URL: %w", i, err)
			}
		}
	}
	return nil
}

// extractPostID extracts the post ID from the URL path.
// Returns the post ID and an error if the ID is missing or invalid.
func extractPostID(r *http.Request) (string, error) {
	pathParts := strings.Split(strings.TrimPrefix(r.URL.Path, "/posts/"), "/")
	if len(pathParts) == 0 || pathParts[0] == "" {
		return "", fmt.Errorf("post ID is required")
	}
	return pathParts[0], nil
}

// CreatePost handles POST /posts - creates a new post.
func (h *PostHandlers) CreatePost(w http.ResponseWriter, r *http.Request) {
	var req CreatePostRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeBadRequest)
		WriteError(w, ctx, http.StatusBadRequest, ErrCodeBadRequest, "Invalid JSON in request body")
		return
	}

	// Validate at least one of sceneId/eventId is provided
	if req.SceneID == nil && req.EventID == nil {
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeMissingTarget)
		WriteError(w, ctx, http.StatusBadRequest, ErrCodeMissingTarget, "Either scene_id or event_id must be provided")
		return
	}

	// Validate and sanitize text
	validatedText, err := validate.PostContent(req.Text)
	if err != nil {
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeValidation)
		WriteError(w, ctx, http.StatusBadRequest, ErrCodeValidation, fmt.Sprintf("Invalid post text: %v", err))
		return
	}
	req.Text = validatedText

	// Validate attachments count
	if len(req.Attachments) > MaxAttachments {
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeValidation)
		WriteError(w, ctx, http.StatusBadRequest, ErrCodeValidation, "Maximum 6 attachments allowed")
		return
	}

	// Validate attachment URLs for SSRF protection
	if err := validatePostAttachments(req.Attachments); err != nil {
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeValidation)
		WriteError(w, ctx, http.StatusBadRequest, ErrCodeValidation, err.Error())
		return
	}

	// Sanitize and validate labels
	sanitizedLabels := make([]string, len(req.Labels))
	for i, label := range req.Labels {
		sanitizedLabels[i] = validate.SanitizeHTML(strings.TrimSpace(label))
	}

	// Validate that all labels are allowed
	if err := post.ValidateLabels(sanitizedLabels); err != nil {
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeValidation)
		WriteError(w, ctx, http.StatusBadRequest, ErrCodeValidation, "Invalid moderation label")
		return
	}

	// Get author DID from context (set by auth middleware)
	authorDID := middleware.GetUserDID(r.Context())
	if authorDID == "" {
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeAuthFailed)
		WriteError(w, ctx, http.StatusUnauthorized, ErrCodeAuthFailed, "authentication required")
		return
	}

	// Enrich attachments with metadata if service is configured
	// This fetches metadata from R2 and strips EXIF data for images
	enrichedAttachments := make([]post.Attachment, 0, len(req.Attachments))
	if h.metadataService != nil {
		for _, att := range req.Attachments {
			// Skip empty keys or attachments that already have all metadata
			if att.Key == "" {
				// Legacy URL-based attachment without key, keep as-is
				enrichedAttachments = append(enrichedAttachments, att)
				continue
			}

			// Enrich the attachment with metadata from R2
			enriched, err := h.metadataService.EnrichAttachment(r.Context(), att.Key)
			if err != nil {
				// Log the error but don't fail the request
				// Use the attachment as provided by the client
				slog.WarnContext(r.Context(), "failed to enrich attachment",
					"key", att.Key,
					"error", err)
				enrichedAttachments = append(enrichedAttachments, att)
				continue
			}

			enrichedAttachments = append(enrichedAttachments, *enriched)
		}
	} else {
		// No metadata service configured, use attachments as-is
		enrichedAttachments = req.Attachments
	}

	// Create post
	newPost := &post.Post{
		SceneID:     req.SceneID,
		EventID:     req.EventID,
		AuthorDID:   authorDID,
		Text:        req.Text,
		Attachments: enrichedAttachments,
		Labels:      sanitizedLabels,
	}

	if err := h.repo.Create(newPost); err != nil {
		slog.ErrorContext(r.Context(), "failed to create post", "error", err)
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeInternal)
		WriteError(w, ctx, http.StatusInternalServerError, ErrCodeInternal, "Failed to create post")
		return
	}

	// Return created post
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(newPost); err != nil {
		slog.ErrorContext(r.Context(), "failed to encode response", "error", err)
		return
	}
}

// UpdatePost handles PATCH /posts/{id} - updates an existing post.
func (h *PostHandlers) UpdatePost(w http.ResponseWriter, r *http.Request) {
	// Extract post ID from URL path
	postID, err := extractPostID(r)
	if err != nil {
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeBadRequest)
		WriteError(w, ctx, http.StatusBadRequest, ErrCodeBadRequest, "Post ID is required")
		return
	}

	// Verify authenticated user
	userDID := middleware.GetUserDID(r.Context())
	if userDID == "" {
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeAuthFailed)
		WriteError(w, ctx, http.StatusUnauthorized, ErrCodeAuthFailed, "Authentication required")
		return
	}

	// Parse request body
	var req UpdatePostRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeBadRequest)
		WriteError(w, ctx, http.StatusBadRequest, ErrCodeBadRequest, "Invalid JSON in request body")
		return
	}

	// Get existing post
	existingPost, err := h.repo.GetByID(postID)
	if err != nil {
		if err == post.ErrPostNotFound {
			ctx := middleware.SetErrorCode(r.Context(), ErrCodeNotFound)
			WriteError(w, ctx, http.StatusNotFound, ErrCodeNotFound, "Post not found")
			return
		}
		slog.ErrorContext(r.Context(), "failed to retrieve post", "error", err, "post_id", postID)
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeInternal)
		WriteError(w, ctx, http.StatusInternalServerError, ErrCodeInternal, "Failed to retrieve post")
		return
	}

	// Verify ownership
	if existingPost.AuthorDID != userDID {
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeForbidden)
		WriteError(w, ctx, http.StatusForbidden, ErrCodeForbidden, "You do not have permission to update this post")
		return
	}

	// Apply updates
	if req.Text != nil {
		validatedText, err := validate.PostContent(*req.Text)
		if err != nil {
			ctx := middleware.SetErrorCode(r.Context(), ErrCodeValidation)
			WriteError(w, ctx, http.StatusBadRequest, ErrCodeValidation, fmt.Sprintf("Invalid post text: %v", err))
			return
		}
		existingPost.Text = validatedText
	}

	if req.Attachments != nil {
		if len(*req.Attachments) > MaxAttachments {
			ctx := middleware.SetErrorCode(r.Context(), ErrCodeValidation)
			WriteError(w, ctx, http.StatusBadRequest, ErrCodeValidation, "Maximum 6 attachments allowed")
			return
		}
		// Validate attachment URLs
		if err := validatePostAttachments(*req.Attachments); err != nil {
			ctx := middleware.SetErrorCode(r.Context(), ErrCodeValidation)
			WriteError(w, ctx, http.StatusBadRequest, ErrCodeValidation, err.Error())
			return
		}
		existingPost.Attachments = *req.Attachments
	}

	if req.Labels != nil {
		sanitizedLabels := make([]string, len(*req.Labels))
		for i, label := range *req.Labels {
			sanitizedLabels[i] = validate.SanitizeHTML(strings.TrimSpace(label))
		}

		// Validate that all labels are allowed
		if err := post.ValidateLabels(sanitizedLabels); err != nil {
			ctx := middleware.SetErrorCode(r.Context(), ErrCodeValidation)
			WriteError(w, ctx, http.StatusBadRequest, ErrCodeValidation, "Invalid moderation label")
			return
		}

		existingPost.Labels = sanitizedLabels
	}

	// Update in repository
	if err := h.repo.Update(existingPost); err != nil {
		if err == post.ErrPostDeleted {
			ctx := middleware.SetErrorCode(r.Context(), ErrCodeNotFound)
			WriteError(w, ctx, http.StatusNotFound, ErrCodeNotFound, "Post not found")
			return
		}
		slog.ErrorContext(r.Context(), "failed to update post", "error", err, "post_id", postID)
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeInternal)
		WriteError(w, ctx, http.StatusInternalServerError, ErrCodeInternal, "Failed to update post")
		return
	}

	// Return updated post (existingPost has been modified in-place)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(existingPost); err != nil {
		slog.ErrorContext(r.Context(), "failed to encode response", "error", err)
		return
	}
}

// DeletePost handles DELETE /posts/{id} - soft-deletes a post.
func (h *PostHandlers) DeletePost(w http.ResponseWriter, r *http.Request) {
	// Extract post ID from URL path
	postID, err := extractPostID(r)
	if err != nil {
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeBadRequest)
		WriteError(w, ctx, http.StatusBadRequest, ErrCodeBadRequest, "Post ID is required")
		return
	}

	// Verify authenticated user
	userDID := middleware.GetUserDID(r.Context())
	if userDID == "" {
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeAuthFailed)
		WriteError(w, ctx, http.StatusUnauthorized, ErrCodeAuthFailed, "Authentication required")
		return
	}

	// Get existing post to verify ownership
	existingPost, err := h.repo.GetByID(postID)
	if err != nil {
		if err == post.ErrPostNotFound {
			ctx := middleware.SetErrorCode(r.Context(), ErrCodeNotFound)
			WriteError(w, ctx, http.StatusNotFound, ErrCodeNotFound, "Post not found")
			return
		}
		slog.ErrorContext(r.Context(), "failed to retrieve post", "error", err, "post_id", postID)
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeInternal)
		WriteError(w, ctx, http.StatusInternalServerError, ErrCodeInternal, "Failed to retrieve post")
		return
	}

	// Verify ownership
	if existingPost.AuthorDID != userDID {
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeForbidden)
		WriteError(w, ctx, http.StatusForbidden, ErrCodeForbidden, "You do not have permission to delete this post")
		return
	}

	// Soft delete the post
	if err := h.repo.Delete(postID); err != nil {
		if err == post.ErrPostNotFound {
			ctx := middleware.SetErrorCode(r.Context(), ErrCodeNotFound)
			WriteError(w, ctx, http.StatusNotFound, ErrCodeNotFound, "Post not found")
			return
		}
		slog.ErrorContext(r.Context(), "failed to delete post", "error", err, "post_id", postID)
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeInternal)
		WriteError(w, ctx, http.StatusInternalServerError, ErrCodeInternal, "Failed to delete post")
		return
	}

	// Return success with no content
	w.WriteHeader(http.StatusNoContent)
}

// FeedResponse represents the JSON response for feed endpoints.
type FeedResponse struct {
	Posts      []*post.Post     `json:"posts"`
	NextCursor *post.FeedCursor `json:"next_cursor,omitempty"`
}

// parseCursor parses cursor from query parameter.
// Returns nil if cursor is not provided or invalid.
func parseCursor(cursorStr string) *post.FeedCursor {
	if cursorStr == "" {
		return nil
	}

	// Cursor format: "created_at_unix_nano:id"
	// Example: "1234567890123456789:uuid-here"
	parts := strings.Split(cursorStr, ":")
	if len(parts) != 2 {
		return nil
	}

	// Parse timestamp (Unix nanoseconds)
	timestamp, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return nil
	}

	return &post.FeedCursor{
		CreatedAt: time.Unix(0, timestamp),
		ID:        parts[1],
	}
}

// canAccessScene checks if a user can access a scene based on visibility rules.
// Returns true if access is allowed, false otherwise.
func (h *PostHandlers) canAccessScene(s *scene.Scene, requesterDID string) (bool, error) {
	// Owner always has access
	if s.IsOwner(requesterDID) {
		return true, nil
	}

	// Check visibility rules
	switch s.Visibility {
	case scene.VisibilityPublic:
		// Public scenes are accessible to everyone
		return true, nil

	case scene.VisibilityMembersOnly:
		// Members-only scenes require active membership
		if requesterDID == "" {
			return false, nil
		}

		// Check if requester is an active member
		m, err := h.membershipRepo.GetBySceneAndUser(s.ID, requesterDID)
		if err != nil {
			// Not a member or error retrieving membership
			if err == membership.ErrMembershipNotFound {
				return false, nil
			}
			return false, err
		}

		// Only active members can access
		return m.Status == "active", nil

	case scene.VisibilityHidden:
		// Hidden scenes only accessible to owner (already checked above)
		return false, nil

	default:
		// Unknown visibility mode - deny access for safety
		slog.Warn("unknown visibility mode", "visibility", s.Visibility, "scene_id", s.ID)
		return false, nil
	}
}

// GetSceneFeed handles GET /scenes/{id}/feed - retrieves posts for a scene with pagination.
func (h *PostHandlers) GetSceneFeed(w http.ResponseWriter, r *http.Request) {
	// Extract scene ID from URL path
	pathParts := strings.Split(strings.TrimPrefix(r.URL.Path, "/scenes/"), "/")
	if len(pathParts) < 2 || pathParts[0] == "" {
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeBadRequest)
		WriteError(w, ctx, http.StatusBadRequest, ErrCodeBadRequest, "Scene ID is required")
		return
	}
	sceneID := pathParts[0]

	// Get the scene to check visibility
	foundScene, err := h.sceneRepo.GetByID(sceneID)
	if err != nil {
		// Use uniform error message - same as "not found" to prevent enumeration
		if err == scene.ErrSceneNotFound || err == scene.ErrSceneDeleted {
			slog.DebugContext(r.Context(), "scene not found or deleted", "scene_id", sceneID)
			ctx := middleware.SetErrorCode(r.Context(), ErrCodeNotFound)
			WriteError(w, ctx, http.StatusNotFound, ErrCodeNotFound, "Scene not found")
			return
		}
		slog.ErrorContext(r.Context(), "failed to retrieve scene", "error", err, "scene_id", sceneID)
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeInternal)
		WriteError(w, ctx, http.StatusInternalServerError, ErrCodeInternal, "Failed to retrieve scene")
		return
	}

	// Get requester DID (empty if not authenticated)
	requesterDID := middleware.GetUserDID(r.Context())

	// Check visibility permissions
	canAccess, err := h.canAccessScene(foundScene, requesterDID)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to check scene access", "error", err, "scene_id", sceneID)
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeInternal)
		WriteError(w, ctx, http.StatusInternalServerError, ErrCodeInternal, "Failed to check access permissions")
		return
	}

	if !canAccess {
		// Use uniform error message - same as "not found" to prevent enumeration
		slog.DebugContext(r.Context(), "scene access denied",
			"scene_id", sceneID,
			"visibility", foundScene.Visibility,
			"requester_did", requesterDID,
			"is_owner", foundScene.IsOwner(requesterDID))
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeNotFound)
		WriteError(w, ctx, http.StatusNotFound, ErrCodeNotFound, "Scene not found")
		return
	}

	// Parse query parameters
	limitStr := r.URL.Query().Get("limit")
	cursorStr := r.URL.Query().Get("cursor")

	// Default limit is 20, max is 100
	limit := 20
	if limitStr != "" {
		parsedLimit, err := strconv.Atoi(limitStr)
		if err != nil || parsedLimit < 1 {
			ctx := middleware.SetErrorCode(r.Context(), ErrCodeValidation)
			WriteError(w, ctx, http.StatusBadRequest, ErrCodeValidation, "Invalid limit parameter")
			return
		}
		if parsedLimit > 100 {
			parsedLimit = 100
		}
		limit = parsedLimit
	}

	// Parse cursor
	cursor := parseCursor(cursorStr)

	// Fetch posts from repository
	posts, nextCursor, err := h.repo.ListByScene(sceneID, limit, cursor)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to list scene posts", "error", err, "scene_id", sceneID)
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeInternal)
		WriteError(w, ctx, http.StatusInternalServerError, ErrCodeInternal, "Failed to retrieve posts")
		return
	}

	// Build response
	response := FeedResponse{
		Posts:      posts,
		NextCursor: nextCursor,
	}

	// Return feed
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		slog.ErrorContext(r.Context(), "failed to encode response", "error", err)
		return
	}
}

// GetEventFeed handles GET /events/{id}/feed - retrieves posts for an event with pagination.
func (h *PostHandlers) GetEventFeed(w http.ResponseWriter, r *http.Request) {
	// Extract event ID from URL path
	pathParts := strings.Split(strings.TrimPrefix(r.URL.Path, "/events/"), "/")
	if len(pathParts) < 2 || pathParts[0] == "" {
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeBadRequest)
		WriteError(w, ctx, http.StatusBadRequest, ErrCodeBadRequest, "Event ID is required")
		return
	}
	eventID := pathParts[0]

	// Parse query parameters
	limitStr := r.URL.Query().Get("limit")
	cursorStr := r.URL.Query().Get("cursor")

	// Default limit is 20, max is 100
	limit := 20
	if limitStr != "" {
		parsedLimit, err := strconv.Atoi(limitStr)
		if err != nil || parsedLimit < 1 {
			ctx := middleware.SetErrorCode(r.Context(), ErrCodeValidation)
			WriteError(w, ctx, http.StatusBadRequest, ErrCodeValidation, "Invalid limit parameter")
			return
		}
		if parsedLimit > 100 {
			parsedLimit = 100
		}
		limit = parsedLimit
	}

	// Parse cursor
	cursor := parseCursor(cursorStr)

	// Fetch posts from repository
	posts, nextCursor, err := h.repo.ListByEvent(eventID, limit, cursor)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to list event posts", "error", err, "event_id", eventID)
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeInternal)
		WriteError(w, ctx, http.StatusInternalServerError, ErrCodeInternal, "Failed to retrieve posts")
		return
	}

	// Build response
	response := FeedResponse{
		Posts:      posts,
		NextCursor: nextCursor,
	}

	// Return feed
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		slog.ErrorContext(r.Context(), "failed to encode response", "error", err)
		return
	}
}
