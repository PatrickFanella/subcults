package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/onnwee/subcults/internal/membership"
	"github.com/onnwee/subcults/internal/middleware"
	"github.com/onnwee/subcults/internal/post"
	"github.com/onnwee/subcults/internal/scene"
)

// testUserDID is the DID used in tests to simulate an authenticated user.
const testUserDID = "did:plc:testuser123"

// withAuthContext attaches a test user DID to the request context.
func withAuthContext(r *http.Request) *http.Request {
	return r.WithContext(middleware.SetUserDID(r.Context(), testUserDID))
}

// Helper function to create a string pointer
func strPtr(s string) *string {
	return &s
}

// newTestPostHandlers creates a PostHandlers instance for testing with mock repositories.
func newTestPostHandlers() *PostHandlers {
	postRepo := post.NewInMemoryPostRepository()
	sceneRepo := scene.NewInMemorySceneRepository()
	membershipRepo := membership.NewInMemoryMembershipRepository()
	return NewPostHandlers(postRepo, sceneRepo, membershipRepo, nil) // nil metadata service for basic tests
}

// TestCreatePost_Success tests successful post creation.
func TestCreatePost_Success(t *testing.T) {
	handlers := newTestPostHandlers()

	sceneID := "scene123"
	reqBody := CreatePostRequest{
		SceneID: &sceneID,
		Text:    "This is a test post",
		Labels:  []string{}, // Empty labels for basic test
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/posts", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	req = withAuthContext(req)
	handlers.CreatePost(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d: %s", w.Code, w.Body.String())
	}

	var createdPost post.Post
	if err := json.NewDecoder(w.Body).Decode(&createdPost); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if createdPost.Text != "This is a test post" {
		t.Errorf("expected text 'This is a test post', got %s", createdPost.Text)
	}
	if createdPost.SceneID == nil || *createdPost.SceneID != sceneID {
		t.Errorf("expected scene_id '%s', got %v", sceneID, createdPost.SceneID)
	}
	if createdPost.ID == "" {
		t.Error("expected ID to be set")
	}
}

// TestCreatePost_WithEventID tests creating a post with event_id.
func TestCreatePost_WithEventID(t *testing.T) {
	handlers := newTestPostHandlers()

	eventID := "event123"
	reqBody := CreatePostRequest{
		EventID: &eventID,
		Text:    "Event announcement",
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/posts", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	req = withAuthContext(req)
	handlers.CreatePost(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d: %s", w.Code, w.Body.String())
	}

	var createdPost post.Post
	if err := json.NewDecoder(w.Body).Decode(&createdPost); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if createdPost.EventID == nil || *createdPost.EventID != eventID {
		t.Errorf("expected event_id '%s', got %v", eventID, createdPost.EventID)
	}
}

// TestCreatePost_MissingTarget tests that missing both sceneId and eventId returns 400.
func TestCreatePost_MissingTarget(t *testing.T) {
	handlers := newTestPostHandlers()

	reqBody := CreatePostRequest{
		Text: "This post has no target",
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/posts", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	req = withAuthContext(req)
	handlers.CreatePost(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}

	var errResp ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}

	if errResp.Error.Code != ErrCodeMissingTarget {
		t.Errorf("expected error code '%s', got '%s'", ErrCodeMissingTarget, errResp.Error.Code)
	}
}

// TestCreatePost_EmptyText tests that empty text returns validation error.
func TestCreatePost_EmptyText(t *testing.T) {
	handlers := newTestPostHandlers()

	sceneID := "scene123"
	reqBody := CreatePostRequest{
		SceneID: &sceneID,
		Text:    "   ",
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/posts", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	req = withAuthContext(req)
	handlers.CreatePost(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}

	var errResp ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}

	if errResp.Error.Code != ErrCodeValidation {
		t.Errorf("expected error code '%s', got '%s'", ErrCodeValidation, errResp.Error.Code)
	}
}

// TestCreatePost_TextTooLong tests that text exceeding 5000 chars returns validation error.
func TestCreatePost_TextTooLong(t *testing.T) {
	handlers := newTestPostHandlers()

	sceneID := "scene123"
	longText := strings.Repeat("a", 5001)
	reqBody := CreatePostRequest{
		SceneID: &sceneID,
		Text:    longText,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/posts", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	req = withAuthContext(req)
	handlers.CreatePost(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}

	var errResp ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}

	if errResp.Error.Code != ErrCodeValidation {
		t.Errorf("expected error code '%s', got '%s'", ErrCodeValidation, errResp.Error.Code)
	}
}

// TestCreatePost_TooManyAttachments tests that more than 6 attachments returns validation error.
func TestCreatePost_TooManyAttachments(t *testing.T) {
	handlers := newTestPostHandlers()

	sceneID := "scene123"
	attachments := make([]post.Attachment, 7)
	for i := range attachments {
		attachments[i] = post.Attachment{URL: "https://example.com/image.jpg", Type: "image"}
	}

	reqBody := CreatePostRequest{
		SceneID:     &sceneID,
		Text:        "Post with too many attachments",
		Attachments: attachments,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/posts", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	req = withAuthContext(req)
	handlers.CreatePost(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}

	var errResp ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}

	if errResp.Error.Code != ErrCodeValidation {
		t.Errorf("expected error code '%s', got '%s'", ErrCodeValidation, errResp.Error.Code)
	}
}

// TestCreatePost_XSSSanitization tests that HTML is escaped to prevent XSS.
func TestCreatePost_XSSSanitization(t *testing.T) {
	handlers := newTestPostHandlers()

	sceneID := "scene123"
	reqBody := CreatePostRequest{
		SceneID: &sceneID,
		Text:    "<script>alert('xss')</script>Hello",
		Labels:  []string{}, // Empty labels, we're testing text sanitization
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/posts", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	req = withAuthContext(req)
	handlers.CreatePost(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d", w.Code)
	}

	var createdPost post.Post
	if err := json.NewDecoder(w.Body).Decode(&createdPost); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Check that HTML was escaped
	if strings.Contains(createdPost.Text, "<script>") {
		t.Error("expected HTML to be escaped in text")
	}
	if !strings.Contains(createdPost.Text, "&lt;script&gt;") {
		t.Error("expected HTML entities in sanitized text")
	}
	if len(createdPost.Labels) > 0 && strings.Contains(createdPost.Labels[0], "<script>") {
		t.Error("expected HTML to be escaped in labels")
	}
}

// TestUpdatePost_Success tests successful post update.
func TestUpdatePost_Success(t *testing.T) {
	handlers := newTestPostHandlers()

	// Create a post first
	sceneID := "scene123"
	originalPost := &post.Post{
		SceneID:   &sceneID,
		AuthorDID: testUserDID,
		Text:      "Original text",
		Labels:    []string{"original"},
	}
	if err := handlers.repo.Create(originalPost); err != nil {
		t.Fatalf("failed to create post: %v", err)
	}

	// Update the post
	newText := "Updated text"
	reqBody := UpdatePostRequest{
		Text: &newText,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPatch, "/posts/"+originalPost.ID, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	req = withAuthContext(req)
	handlers.UpdatePost(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var updatedPost post.Post
	if err := json.NewDecoder(w.Body).Decode(&updatedPost); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if updatedPost.Text != "Updated text" {
		t.Errorf("expected text 'Updated text', got %s", updatedPost.Text)
	}
	if updatedPost.ID != originalPost.ID {
		t.Error("expected ID to remain the same")
	}
}

// TestUpdatePost_Labels tests updating labels.
func TestUpdatePost_Labels(t *testing.T) {
	handlers := newTestPostHandlers()

	// Create a post first
	sceneID := "scene123"
	originalPost := &post.Post{
		SceneID:   &sceneID,
		AuthorDID: testUserDID,
		Text:      "Test post",
		Labels:    []string{post.LabelNSFW}, // Use valid moderation label
	}
	if err := handlers.repo.Create(originalPost); err != nil {
		t.Fatalf("failed to create post: %v", err)
	}

	// Update labels with valid moderation labels
	newLabels := []string{post.LabelSpam, post.LabelFlagged}
	reqBody := UpdatePostRequest{
		Labels: &newLabels,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPatch, "/posts/"+originalPost.ID, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	req = withAuthContext(req)
	handlers.UpdatePost(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var updatedPost post.Post
	if err := json.NewDecoder(w.Body).Decode(&updatedPost); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(updatedPost.Labels) != 2 {
		t.Errorf("expected 2 labels, got %d", len(updatedPost.Labels))
	}
	if !updatedPost.HasLabel(post.LabelSpam) {
		t.Error("expected post to have spam label")
	}
	if !updatedPost.HasLabel(post.LabelFlagged) {
		t.Error("expected post to have flagged label")
	}
}

// TestUpdatePost_NotFound tests updating a non-existent post.
func TestUpdatePost_NotFound(t *testing.T) {
	handlers := newTestPostHandlers()

	newText := "Updated text"
	reqBody := UpdatePostRequest{
		Text: &newText,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPatch, "/posts/nonexistent", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	req = withAuthContext(req)
	handlers.UpdatePost(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}

	var errResp ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}

	if errResp.Error.Code != ErrCodeNotFound {
		t.Errorf("expected error code '%s', got '%s'", ErrCodeNotFound, errResp.Error.Code)
	}
}

// TestDeletePost_Success tests successful post deletion.
func TestDeletePost_Success(t *testing.T) {
	handlers := newTestPostHandlers()

	// Create a post first
	sceneID := "scene123"
	originalPost := &post.Post{
		SceneID:   &sceneID,
		AuthorDID: testUserDID,
		Text:      "Test post",
	}
	if err := handlers.repo.Create(originalPost); err != nil {
		t.Fatalf("failed to create post: %v", err)
	}

	// Delete the post
	req := httptest.NewRequest(http.MethodDelete, "/posts/"+originalPost.ID, nil)
	w := httptest.NewRecorder()

	req = withAuthContext(req)
	handlers.DeletePost(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected status 204, got %d", w.Code)
	}

	// Verify post is not retrievable (soft deleted)
	_, err := handlers.repo.GetByID(originalPost.ID)
	if err != post.ErrPostNotFound {
		t.Error("expected post to be soft deleted (not retrievable)")
	}
}

// TestDeletePost_SoftDeletedExclusion tests that soft-deleted post returns 404 on fetch.
func TestDeletePost_SoftDeletedExclusion(t *testing.T) {
	handlers := newTestPostHandlers()

	// Create a post first
	sceneID := "scene123"
	originalPost := &post.Post{
		SceneID:   &sceneID,
		AuthorDID: testUserDID,
		Text:      "Test post",
	}
	if err := handlers.repo.Create(originalPost); err != nil {
		t.Fatalf("failed to create post: %v", err)
	}

	// Delete the post
	req := httptest.NewRequest(http.MethodDelete, "/posts/"+originalPost.ID, nil)
	w := httptest.NewRecorder()
	req = withAuthContext(req)
	handlers.DeletePost(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected status 204, got %d", w.Code)
	}

	// Try to fetch the deleted post - should return 404
	// We verify via repository directly since GetPost handler is not in scope
	_, err := handlers.repo.GetByID(originalPost.ID)
	if err != post.ErrPostNotFound {
		t.Error("expected GetByID to return ErrPostNotFound for soft-deleted post")
	}

	// Also try to update the deleted post - should fail
	newText := "Updated"
	reqBody := UpdatePostRequest{Text: &newText}
	body, _ := json.Marshal(reqBody)
	req3 := httptest.NewRequest(http.MethodPatch, "/posts/"+originalPost.ID, bytes.NewReader(body))
	w3 := httptest.NewRecorder()
	req3 = withAuthContext(req3)
	handlers.UpdatePost(w3, req3)

	if w3.Code != http.StatusNotFound {
		t.Errorf("expected status 404 when updating deleted post, got %d", w3.Code)
	}
}

// TestDeletePost_NotFound tests deleting a non-existent post.
func TestDeletePost_NotFound(t *testing.T) {
	handlers := newTestPostHandlers()

	req := httptest.NewRequest(http.MethodDelete, "/posts/nonexistent", nil)
	w := httptest.NewRecorder()

	req = withAuthContext(req)
	handlers.DeletePost(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}

	var errResp ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}

	if errResp.Error.Code != ErrCodeNotFound {
		t.Errorf("expected error code '%s', got '%s'", ErrCodeNotFound, errResp.Error.Code)
	}
}

// TestDeletePost_AlreadyDeleted tests deleting an already deleted post.
func TestDeletePost_AlreadyDeleted(t *testing.T) {
	handlers := newTestPostHandlers()

	// Create and delete a post
	sceneID := "scene123"
	originalPost := &post.Post{
		SceneID:   &sceneID,
		AuthorDID: testUserDID,
		Text:      "Test post",
	}
	if err := handlers.repo.Create(originalPost); err != nil {
		t.Fatalf("failed to create post: %v", err)
	}
	if err := handlers.repo.Delete(originalPost.ID); err != nil {
		t.Fatalf("failed to delete post: %v", err)
	}

	// Try to delete again
	req := httptest.NewRequest(http.MethodDelete, "/posts/"+originalPost.ID, nil)
	w := httptest.NewRecorder()

	req = withAuthContext(req)
	handlers.DeletePost(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

// TestCreatePost_InvalidLabel tests that invalid labels are rejected.
func TestCreatePost_InvalidLabel(t *testing.T) {
	handlers := newTestPostHandlers()

	sceneID := "scene123"
	reqBody := CreatePostRequest{
		SceneID: &sceneID,
		Text:    "Test post",
		Labels:  []string{"invalid_label"},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/posts", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	req = withAuthContext(req)
	handlers.CreatePost(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}

	var errResp ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}

	if errResp.Error.Code != ErrCodeValidation {
		t.Errorf("expected error code '%s', got '%s'", ErrCodeValidation, errResp.Error.Code)
	}
	if errResp.Error.Message != "Invalid moderation label" {
		t.Errorf("expected error message 'Invalid moderation label', got '%s'", errResp.Error.Message)
	}
}

// TestCreatePost_ValidModerationLabels tests that valid moderation labels are accepted.
func TestCreatePost_ValidModerationLabels(t *testing.T) {
	handlers := newTestPostHandlers()

	sceneID := "scene123"

	tests := []struct {
		name   string
		labels []string
	}{
		{
			name:   "hidden label",
			labels: []string{post.LabelHidden},
		},
		{
			name:   "nsfw label",
			labels: []string{post.LabelNSFW},
		},
		{
			name:   "spam label",
			labels: []string{post.LabelSpam},
		},
		{
			name:   "flagged label",
			labels: []string{post.LabelFlagged},
		},
		{
			name:   "multiple valid labels",
			labels: []string{post.LabelNSFW, post.LabelFlagged},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reqBody := CreatePostRequest{
				SceneID: &sceneID,
				Text:    "Test post",
				Labels:  tt.labels,
			}

			body, err := json.Marshal(reqBody)
			if err != nil {
				t.Fatalf("failed to marshal request: %v", err)
			}

			req := httptest.NewRequest(http.MethodPost, "/posts", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			req = withAuthContext(req)
			handlers.CreatePost(w, req)

			if w.Code != http.StatusCreated {
				t.Errorf("expected status 201, got %d: %s", w.Code, w.Body.String())
			}

			var createdPost post.Post
			if err := json.NewDecoder(w.Body).Decode(&createdPost); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}

			if len(createdPost.Labels) != len(tt.labels) {
				t.Errorf("expected %d labels, got %d", len(tt.labels), len(createdPost.Labels))
			}
		})
	}
}

// TestUpdatePost_InvalidLabel tests that invalid labels are rejected on update.
func TestUpdatePost_InvalidLabel(t *testing.T) {
	handlers := newTestPostHandlers()

	// Create a post first
	sceneID := "scene123"
	originalPost := &post.Post{
		SceneID:   &sceneID,
		AuthorDID: testUserDID,
		Text:      "Test post",
		Labels:    []string{post.LabelNSFW},
	}
	if err := handlers.repo.Create(originalPost); err != nil {
		t.Fatalf("failed to create post: %v", err)
	}

	// Try to update with invalid label
	newLabels := []string{"invalid_label"}
	reqBody := UpdatePostRequest{
		Labels: &newLabels,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPatch, "/posts/"+originalPost.ID, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	req = withAuthContext(req)
	handlers.UpdatePost(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}

	var errResp ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}

	if errResp.Error.Code != ErrCodeValidation {
		t.Errorf("expected error code '%s', got '%s'", ErrCodeValidation, errResp.Error.Code)
	}
	if errResp.Error.Message != "Invalid moderation label" {
		t.Errorf("expected error message 'Invalid moderation label', got '%s'", errResp.Error.Message)
	}

	// Verify original labels are unchanged
	retrieved, err := handlers.repo.GetByID(originalPost.ID)
	if err != nil {
		t.Fatalf("failed to retrieve post: %v", err)
	}
	if len(retrieved.Labels) != 1 || retrieved.Labels[0] != post.LabelNSFW {
		t.Error("expected original labels to remain unchanged after failed update")
	}
}

// TestUpdatePost_ValidModerationLabels tests that valid moderation labels work on update.
func TestUpdatePost_ValidModerationLabels(t *testing.T) {
	handlers := newTestPostHandlers()

	// Create a post first
	sceneID := "scene123"
	originalPost := &post.Post{
		SceneID:   &sceneID,
		AuthorDID: testUserDID,
		Text:      "Test post",
		Labels:    []string{},
	}
	if err := handlers.repo.Create(originalPost); err != nil {
		t.Fatalf("failed to create post: %v", err)
	}

	// Update with valid moderation labels
	newLabels := []string{post.LabelHidden, post.LabelSpam}
	reqBody := UpdatePostRequest{
		Labels: &newLabels,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPatch, "/posts/"+originalPost.ID, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	req = withAuthContext(req)
	handlers.UpdatePost(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var updatedPost post.Post
	if err := json.NewDecoder(w.Body).Decode(&updatedPost); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(updatedPost.Labels) != 2 {
		t.Errorf("expected 2 labels, got %d", len(updatedPost.Labels))
	}
	if !updatedPost.HasLabel(post.LabelHidden) {
		t.Error("expected post to have hidden label")
	}
	if !updatedPost.HasLabel(post.LabelSpam) {
		t.Error("expected post to have spam label")
	}
}

// TestCreatePost_WithAttachments tests creating a post with attachments.
func TestCreatePost_WithAttachments(t *testing.T) {
	handlers := newTestPostHandlers()

	sceneID := "scene123"
	width := 1920
	height := 1080
	sizeBytes := int64(1024000)

	reqBody := CreatePostRequest{
		SceneID: &sceneID,
		Text:    "Post with image attachment",
		Attachments: []post.Attachment{
			{
				Key:       "posts/test/image1.jpg",
				Type:      "image/jpeg",
				SizeBytes: sizeBytes,
				Width:     &width,
				Height:    &height,
			},
		},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/posts", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	req = withAuthContext(req)
	handlers.CreatePost(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d: %s", w.Code, w.Body.String())
	}

	var createdPost post.Post
	if err := json.NewDecoder(w.Body).Decode(&createdPost); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Verify attachment metadata
	if len(createdPost.Attachments) != 1 {
		t.Fatalf("expected 1 attachment, got %d", len(createdPost.Attachments))
	}

	att := createdPost.Attachments[0]
	if att.Key != "posts/test/image1.jpg" {
		t.Errorf("expected key 'posts/test/image1.jpg', got %s", att.Key)
	}
	if att.Type != "image/jpeg" {
		t.Errorf("expected type 'image/jpeg', got %s", att.Type)
	}
	if att.SizeBytes != sizeBytes {
		t.Errorf("expected size %d, got %d", sizeBytes, att.SizeBytes)
	}
	if att.Width == nil || *att.Width != width {
		t.Errorf("expected width %d, got %v", width, att.Width)
	}
	if att.Height == nil || *att.Height != height {
		t.Errorf("expected height %d, got %v", height, att.Height)
	}
}

// TestCreatePost_WithMultipleAttachments tests creating a post with multiple attachments.
func TestCreatePost_WithMultipleAttachments(t *testing.T) {
	handlers := newTestPostHandlers()

	sceneID := "scene123"
	width1 := 1920
	height1 := 1080
	width2 := 800
	height2 := 600

	reqBody := CreatePostRequest{
		SceneID: &sceneID,
		Text:    "Post with multiple attachments",
		Attachments: []post.Attachment{
			{
				Key:       "posts/test/image1.jpg",
				Type:      "image/jpeg",
				SizeBytes: 1024000,
				Width:     &width1,
				Height:    &height1,
			},
			{
				Key:       "posts/test/image2.png",
				Type:      "image/png",
				SizeBytes: 512000,
				Width:     &width2,
				Height:    &height2,
			},
		},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/posts", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	req = withAuthContext(req)
	handlers.CreatePost(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d: %s", w.Code, w.Body.String())
	}

	var createdPost post.Post
	if err := json.NewDecoder(w.Body).Decode(&createdPost); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Verify both attachments
	if len(createdPost.Attachments) != 2 {
		t.Fatalf("expected 2 attachments, got %d", len(createdPost.Attachments))
	}
}

// TestCreatePost_WithAudioAttachment tests creating a post with audio attachment (no dimensions).
func TestCreatePost_WithAudioAttachment(t *testing.T) {
	handlers := newTestPostHandlers()

	sceneID := "scene123"
	duration := 180.5

	reqBody := CreatePostRequest{
		SceneID: &sceneID,
		Text:    "Post with audio attachment",
		Attachments: []post.Attachment{
			{
				Key:             "posts/test/audio.mp3",
				Type:            "audio/mpeg",
				SizeBytes:       5000000,
				DurationSeconds: &duration,
			},
		},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/posts", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	req = withAuthContext(req)
	handlers.CreatePost(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d: %s", w.Code, w.Body.String())
	}

	var createdPost post.Post
	if err := json.NewDecoder(w.Body).Decode(&createdPost); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Verify audio attachment (should not have width/height)
	if len(createdPost.Attachments) != 1 {
		t.Fatalf("expected 1 attachment, got %d", len(createdPost.Attachments))
	}

	att := createdPost.Attachments[0]
	if att.Type != "audio/mpeg" {
		t.Errorf("expected type 'audio/mpeg', got %s", att.Type)
	}
	if att.Width != nil {
		t.Errorf("audio attachment should not have width, got %v", att.Width)
	}
	if att.Height != nil {
		t.Errorf("audio attachment should not have height, got %v", att.Height)
	}
	if att.DurationSeconds == nil || *att.DurationSeconds != duration {
		t.Errorf("expected duration %f, got %v", duration, att.DurationSeconds)
	}
}
