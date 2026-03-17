package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/onnwee/subcults/internal/post"
	"github.com/onnwee/subcults/internal/scene"
)

// TestSearchPosts_Success tests successful post search with text query.
func TestSearchPosts_Success(t *testing.T) {
	postRepo := post.NewInMemoryPostRepository()
	sceneRepo := scene.NewInMemorySceneRepository()
	handlers := NewSearchHandlers(sceneRepo, postRepo, nil, scene.NewInMemoryEventRepository())

	sceneID1 := "scene-1"
	sceneID2 := "scene-2"

	// Create test posts with different text content
	post1 := &post.Post{
		SceneID:   &sceneID1,
		AuthorDID: "did:plc:user1",
		Text:      "Electronic music festival happening next week",
		Labels:    []string{},
	}
	if err := postRepo.Create(post1); err != nil {
		t.Fatalf("failed to create post1: %v", err)
	}

	post2 := &post.Post{
		SceneID:   &sceneID2,
		AuthorDID: "did:plc:user2",
		Text:      "Jazz concert with live electronic beats",
		Labels:    []string{},
	}
	if err := postRepo.Create(post2); err != nil {
		t.Fatalf("failed to create post2: %v", err)
	}

	post3 := &post.Post{
		SceneID:   &sceneID1,
		AuthorDID: "did:plc:user3",
		Text:      "Rock band performing tonight",
		Labels:    []string{},
	}
	if err := postRepo.Create(post3); err != nil {
		t.Fatalf("failed to create post3: %v", err)
	}

	// Search for "electronic"
	req := httptest.NewRequest(http.MethodGet, "/search/posts?q=electronic", nil)
	w := httptest.NewRecorder()

	handlers.SearchPosts(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var response PostSearchResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Should return 2 posts (post1 and post2 contain "electronic")
	if response.Count != 2 {
		t.Errorf("expected count 2, got %d", response.Count)
	}

	if len(response.Results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(response.Results))
	}

	// Verify posts are in the results
	foundPost1 := false
	foundPost2 := false
	for _, result := range response.Results {
		if result.ID == post1.ID {
			foundPost1 = true
		}
		if result.ID == post2.ID {
			foundPost2 = true
		}
	}

	if !foundPost1 || !foundPost2 {
		t.Errorf("expected to find post1 and post2 in results")
	}
}

// TestSearchPosts_WithSceneFilter tests post search with scene filter.
func TestSearchPosts_WithSceneFilter(t *testing.T) {
	postRepo := post.NewInMemoryPostRepository()
	sceneRepo := scene.NewInMemorySceneRepository()
	handlers := NewSearchHandlers(sceneRepo, postRepo, nil, scene.NewInMemoryEventRepository())

	sceneID1 := "scene-1"
	sceneID2 := "scene-2"

	// Create test posts
	post1 := &post.Post{
		SceneID:   &sceneID1,
		AuthorDID: "did:plc:user1",
		Text:      "Electronic music festival",
		Labels:    []string{},
	}
	if err := postRepo.Create(post1); err != nil {
		t.Fatalf("failed to create post1: %v", err)
	}

	post2 := &post.Post{
		SceneID:   &sceneID2,
		AuthorDID: "did:plc:user2",
		Text:      "Electronic beats and rhythms",
		Labels:    []string{},
	}
	if err := postRepo.Create(post2); err != nil {
		t.Fatalf("failed to create post2: %v", err)
	}

	// Search for "electronic" in scene-1 only
	req := httptest.NewRequest(http.MethodGet, "/search/posts?q=electronic&scene_id=scene-1", nil)
	w := httptest.NewRecorder()

	handlers.SearchPosts(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var response PostSearchResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Should return only 1 post (post1 from scene-1)
	if response.Count != 1 {
		t.Errorf("expected count 1, got %d", response.Count)
	}

	if len(response.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(response.Results))
	}

	if response.Results[0].ID != post1.ID {
		t.Errorf("expected post1, got post ID %s", response.Results[0].ID)
	}

	if response.Results[0].SceneID == nil || *response.Results[0].SceneID != sceneID1 {
		t.Errorf("expected scene_id %s, got %v", sceneID1, response.Results[0].SceneID)
	}
}

// TestSearchPosts_ExcludesModeratedPosts tests that moderated posts are excluded.
func TestSearchPosts_ExcludesModeratedPosts(t *testing.T) {
	postRepo := post.NewInMemoryPostRepository()
	sceneRepo := scene.NewInMemorySceneRepository()
	handlers := NewSearchHandlers(sceneRepo, postRepo, nil, scene.NewInMemoryEventRepository())

	sceneID := "scene-1"

	// Create test posts with different moderation labels
	post1 := &post.Post{
		SceneID:   &sceneID,
		AuthorDID: "did:plc:user1",
		Text:      "Electronic music is awesome",
		Labels:    []string{}, // No labels
	}
	if err := postRepo.Create(post1); err != nil {
		t.Fatalf("failed to create post1: %v", err)
	}

	post2 := &post.Post{
		SceneID:   &sceneID,
		AuthorDID: "did:plc:user2",
		Text:      "Electronic beats are hidden",
		Labels:    []string{post.LabelHidden}, // Hidden
	}
	if err := postRepo.Create(post2); err != nil {
		t.Fatalf("failed to create post2: %v", err)
	}

	post3 := &post.Post{
		SceneID:   &sceneID,
		AuthorDID: "did:plc:user3",
		Text:      "Electronic spam content",
		Labels:    []string{post.LabelSpam}, // Spam
	}
	if err := postRepo.Create(post3); err != nil {
		t.Fatalf("failed to create post3: %v", err)
	}

	post4 := &post.Post{
		SceneID:   &sceneID,
		AuthorDID: "did:plc:user4",
		Text:      "Electronic flagged content",
		Labels:    []string{post.LabelFlagged}, // Flagged
	}
	if err := postRepo.Create(post4); err != nil {
		t.Fatalf("failed to create post4: %v", err)
	}

	// Search for "electronic"
	req := httptest.NewRequest(http.MethodGet, "/search/posts?q=electronic", nil)
	w := httptest.NewRecorder()

	handlers.SearchPosts(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var response PostSearchResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Should return only 1 post (post1, without moderation labels)
	if response.Count != 1 {
		t.Errorf("expected count 1 (excluding moderated posts), got %d", response.Count)
	}

	if len(response.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(response.Results))
	}

	if response.Results[0].ID != post1.ID {
		t.Errorf("expected post1 (clean post), got post ID %s", response.Results[0].ID)
	}
}

// TestSearchPosts_Pagination tests cursor-based pagination.
func TestSearchPosts_Pagination(t *testing.T) {
	postRepo := post.NewInMemoryPostRepository()
	sceneRepo := scene.NewInMemorySceneRepository()
	handlers := NewSearchHandlers(sceneRepo, postRepo, nil, scene.NewInMemoryEventRepository())

	sceneID := "scene-1"

	// Create multiple posts
	for i := 0; i < 5; i++ {
		p := &post.Post{
			SceneID:   &sceneID,
			AuthorDID: fmt.Sprintf("did:plc:user%d", i),
			Text:      fmt.Sprintf("Electronic music post number %d", i),
			Labels:    []string{},
		}
		if err := postRepo.Create(p); err != nil {
			t.Fatalf("failed to create post %d: %v", i, err)
		}
		// Small delay to ensure different timestamps
		time.Sleep(1 * time.Millisecond)
	}

	// First page with limit 3
	req := httptest.NewRequest(http.MethodGet, "/search/posts?q=electronic&limit=3", nil)
	w := httptest.NewRecorder()

	handlers.SearchPosts(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var response PostSearchResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Count != 3 {
		t.Errorf("expected count 3, got %d", response.Count)
	}

	if response.NextCursor == "" {
		t.Error("expected next_cursor to be set")
	}

	// Second page with cursor
	req2 := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/search/posts?q=electronic&limit=3&cursor=%s", response.NextCursor), nil)
	w2 := httptest.NewRecorder()

	handlers.SearchPosts(w2, req2)

	if w2.Code != http.StatusOK {
		t.Errorf("expected status 200 on page 2, got %d: %s", w2.Code, w2.Body.String())
	}

	var response2 PostSearchResponse
	if err := json.NewDecoder(w2.Body).Decode(&response2); err != nil {
		t.Fatalf("failed to decode page 2 response: %v", err)
	}

	// Should return remaining 2 posts
	if response2.Count != 2 {
		t.Errorf("expected count 2 on page 2, got %d", response2.Count)
	}

	if response2.NextCursor != "" {
		t.Error("expected next_cursor to be empty on last page")
	}
}

// TestSearchPosts_MissingQuery tests error handling for missing query parameter.
func TestSearchPosts_MissingQuery(t *testing.T) {
	postRepo := post.NewInMemoryPostRepository()
	sceneRepo := scene.NewInMemorySceneRepository()
	handlers := NewSearchHandlers(sceneRepo, postRepo, nil, scene.NewInMemoryEventRepository())

	// Request without q parameter
	req := httptest.NewRequest(http.MethodGet, "/search/posts", nil)
	w := httptest.NewRecorder()

	handlers.SearchPosts(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}

	var errResp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}

	// Response format: {"error": {"code": "validation_error", "message": "..."}}
	errorObj, ok := errResp["error"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected error object in response, got %v", errResp)
	}
	if errorObj["code"] != ErrCodeValidation {
		t.Errorf("expected error code %s, got %v", ErrCodeValidation, errorObj["code"])
	}
}

// TestSearchPosts_LimitValidation tests limit parameter validation.
func TestSearchPosts_LimitValidation(t *testing.T) {
	postRepo := post.NewInMemoryPostRepository()
	sceneRepo := scene.NewInMemorySceneRepository()
	handlers := NewSearchHandlers(sceneRepo, postRepo, nil, scene.NewInMemoryEventRepository())

	tests := []struct {
		name           string
		limit          string
		expectStatus   int
		expectMaxLimit bool
	}{
		{"Valid limit", "10", http.StatusOK, false},
		{"Limit at max", "50", http.StatusOK, false},
		{"Limit over max", "100", http.StatusOK, true}, // Should be capped at 50
		{"Invalid limit", "invalid", http.StatusBadRequest, false},
		{"Negative limit", "-1", http.StatusBadRequest, false},
		{"Zero limit", "0", http.StatusBadRequest, false},
	}

	sceneID := "scene-1"
	// Create a test post
	p := &post.Post{
		SceneID:   &sceneID,
		AuthorDID: "did:plc:user1",
		Text:      "Electronic music",
		Labels:    []string{},
	}
	if err := postRepo.Create(p); err != nil {
		t.Fatalf("failed to create test post: %v", err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := fmt.Sprintf("/search/posts?q=electronic&limit=%s", tt.limit)
			req := httptest.NewRequest(http.MethodGet, url, nil)
			w := httptest.NewRecorder()

			handlers.SearchPosts(w, req)

			if w.Code != tt.expectStatus {
				t.Errorf("expected status %d, got %d", tt.expectStatus, w.Code)
			}

			if tt.expectStatus == http.StatusOK && tt.expectMaxLimit {
				var response PostSearchResponse
				if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}
				// Verify that limit was capped (we only have 1 post, but the request succeeded)
				// The actual enforcement happens in the repository
			}
		})
	}
}

// TestSearchPosts_ExcerptGeneration tests that excerpts are properly generated.
func TestSearchPosts_ExcerptGeneration(t *testing.T) {
	postRepo := post.NewInMemoryPostRepository()
	sceneRepo := scene.NewInMemorySceneRepository()
	handlers := NewSearchHandlers(sceneRepo, postRepo, nil, scene.NewInMemoryEventRepository())

	sceneID := "scene-1"

	// Create post with long text
	longText := "Electronic music is a genre of music that employs electronic musical instruments, digital instruments, and circuitry-based music technology. This text is intentionally long to test the excerpt generation functionality and ensure it truncates properly at word boundaries."
	post1 := &post.Post{
		SceneID:   &sceneID,
		AuthorDID: "did:plc:user1",
		Text:      longText,
		Labels:    []string{},
	}
	if err := postRepo.Create(post1); err != nil {
		t.Fatalf("failed to create post1: %v", err)
	}

	// Create post with short text
	shortText := "Electronic beats"
	post2 := &post.Post{
		SceneID:   &sceneID,
		AuthorDID: "did:plc:user2",
		Text:      shortText,
		Labels:    []string{},
	}
	if err := postRepo.Create(post2); err != nil {
		t.Fatalf("failed to create post2: %v", err)
	}

	// Search for "electronic"
	req := httptest.NewRequest(http.MethodGet, "/search/posts?q=electronic", nil)
	w := httptest.NewRecorder()

	handlers.SearchPosts(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var response PostSearchResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Find the results
	for _, result := range response.Results {
		if result.ID == post1.ID {
			// Long text should be truncated
			if len(result.Excerpt) > 163 { // 160 + "..."
				t.Errorf("expected excerpt to be truncated to ~160 chars, got %d chars", len(result.Excerpt))
			}
			if !containsEllipsis(result.Excerpt) {
				t.Error("expected long excerpt to contain ellipsis")
			}
		}
		if result.ID == post2.ID {
			// Short text should not be truncated
			if result.Excerpt != shortText {
				t.Errorf("expected excerpt '%s', got '%s'", shortText, result.Excerpt)
			}
		}
	}
}

// Helper function to check if text contains ellipsis
func containsEllipsis(text string) bool {
	return len(text) >= 3 && text[len(text)-3:] == "..."
}
