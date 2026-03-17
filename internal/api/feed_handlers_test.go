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

// createTestSceneForFeed creates a public scene for feed testing.
func createTestSceneForFeed(sceneRepo scene.SceneRepository, sceneID, ownerDID string) {
	testScene := &scene.Scene{
		ID:            sceneID,
		Name:          "Test Scene",
		OwnerDID:      ownerDID,
		Visibility:    scene.VisibilityPublic,
		CoarseGeohash: "9q8yy",
	}
	if err := sceneRepo.Insert(testScene); err != nil {
		panic(err)
	}
}

// seedTestPosts creates test posts in the repository for feed testing.
func seedTestPosts(repo post.PostRepository, sceneID, eventID string, count int) []*post.Post {
	posts := make([]*post.Post, count)
	now := time.Now()

	for i := 0; i < count; i++ {
		p := &post.Post{
			SceneID:   &sceneID,
			EventID:   &eventID,
			AuthorDID: "did:example:user1",
			Text:      "Test post " + string(rune('A'+i)),
			Labels:    []string{},
		}

		// Create posts with different timestamps for ordering
		// Each post is 1 hour older than the previous
		p.CreatedAt = now.Add(-time.Duration(i) * time.Hour)

		if err := repo.Create(p); err != nil {
			panic(err)
		}
		posts[i] = p
	}

	return posts
}

// TestGetSceneFeed_Success tests successful scene feed retrieval.
func TestGetSceneFeed_Success(t *testing.T) {
	handlers := newTestPostHandlers()

	sceneID := "scene123"

	// Create a public scene for testing
	createTestSceneForFeed(handlers.sceneRepo, sceneID, "did:example:owner")

	// Seed 5 posts
	seedTestPosts(handlers.repo, sceneID, "event123", 5)

	req := httptest.NewRequest(http.MethodGet, "/scenes/"+sceneID+"/feed", nil)
	w := httptest.NewRecorder()

	handlers.GetSceneFeed(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var response FeedResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(response.Posts) != 5 {
		t.Errorf("expected 5 posts, got %d", len(response.Posts))
	}

	// Verify ordering: newest first
	for i := 1; i < len(response.Posts); i++ {
		if response.Posts[i-1].CreatedAt.Before(response.Posts[i].CreatedAt) {
			t.Errorf("posts not ordered correctly: post %d is older than post %d", i-1, i)
		}
	}
}

// TestGetSceneFeed_Pagination tests cursor-based pagination.
func TestGetSceneFeed_Pagination(t *testing.T) {
	handlers := newTestPostHandlers()

	sceneID := "scene123"

	// Create a public scene for testing
	createTestSceneForFeed(handlers.sceneRepo, "scene123", "did:example:owner")

	// Seed 25 posts
	seedTestPosts(handlers.repo, sceneID, "event123", 25)

	// First page: limit=10
	req := httptest.NewRequest(http.MethodGet, "/scenes/"+sceneID+"/feed?limit=10", nil)
	w := httptest.NewRecorder()

	handlers.GetSceneFeed(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var page1 FeedResponse
	if err := json.NewDecoder(w.Body).Decode(&page1); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(page1.Posts) != 10 {
		t.Errorf("expected 10 posts on page 1, got %d", len(page1.Posts))
	}

	if page1.NextCursor == nil {
		t.Fatal("expected next_cursor to be set")
	}

	// Second page: use cursor from first page
	cursorStr := encodeCursorString(page1.NextCursor)
	req2 := httptest.NewRequest(http.MethodGet, "/scenes/"+sceneID+"/feed?limit=10&cursor="+cursorStr, nil)
	w2 := httptest.NewRecorder()

	handlers.GetSceneFeed(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w2.Code, w2.Body.String())
	}

	var page2 FeedResponse
	if err := json.NewDecoder(w2.Body).Decode(&page2); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(page2.Posts) != 10 {
		t.Errorf("expected 10 posts on page 2, got %d", len(page2.Posts))
	}

	// Verify no overlap between pages
	for _, p1 := range page1.Posts {
		for _, p2 := range page2.Posts {
			if p1.ID == p2.ID {
				t.Errorf("found duplicate post ID %s across pages", p1.ID)
			}
		}
	}

	// Verify continuity: last post of page 1 should be newer than first post of page 2
	lastP1 := page1.Posts[len(page1.Posts)-1]
	firstP2 := page2.Posts[0]

	if lastP1.CreatedAt.Before(firstP2.CreatedAt) {
		t.Error("pagination ordering broken: page 1 last post is older than page 2 first post")
	}
}

// TestGetSceneFeed_HiddenPostsExcluded tests that posts with 'hidden' label are excluded.
func TestGetSceneFeed_HiddenPostsExcluded(t *testing.T) {
	handlers := newTestPostHandlers()

	sceneID := "scene123"

	// Create a public scene for testing
	createTestSceneForFeed(handlers.sceneRepo, "scene123", "did:example:owner")

	// Create 3 normal posts
	normalPosts := seedTestPosts(handlers.repo, sceneID, "event123", 3)

	// Create 2 hidden posts
	for i := 0; i < 2; i++ {
		hiddenPost := &post.Post{
			SceneID:   &sceneID,
			EventID:   strPtr("event123"),
			AuthorDID: "did:example:user1",
			Text:      "Hidden post",
			Labels:    []string{post.LabelHidden},
			CreatedAt: time.Now().Add(time.Duration(i) * time.Hour), // Newer than normal posts
		}
		if err := handlers.repo.Create(hiddenPost); err != nil {
			t.Fatalf("failed to create hidden post: %v", err)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/scenes/"+sceneID+"/feed", nil)
	w := httptest.NewRecorder()

	handlers.GetSceneFeed(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var response FeedResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Should only return the 3 normal posts, not the 2 hidden ones
	if len(response.Posts) != 3 {
		t.Errorf("expected 3 posts (hidden excluded), got %d", len(response.Posts))
	}

	// Verify none of the returned posts have the hidden label
	for _, p := range response.Posts {
		if p.HasLabel(post.LabelHidden) {
			t.Errorf("found hidden post in feed: %s", p.ID)
		}
	}

	// Verify the returned posts are the normal ones
	returnedIDs := make(map[string]bool)
	for _, p := range response.Posts {
		returnedIDs[p.ID] = true
	}

	for _, np := range normalPosts {
		if !returnedIDs[np.ID] {
			t.Errorf("normal post %s not found in feed", np.ID)
		}
	}
}

// TestGetSceneFeed_DeletedPostsExcluded tests that soft-deleted posts are excluded.
func TestGetSceneFeed_DeletedPostsExcluded(t *testing.T) {
	handlers := newTestPostHandlers()

	sceneID := "scene123"

	// Create a public scene for testing
	createTestSceneForFeed(handlers.sceneRepo, "scene123", "did:example:owner")

	// Create 5 posts
	posts := seedTestPosts(handlers.repo, sceneID, "event123", 5)

	// Delete 2 posts
	if err := handlers.repo.Delete(posts[1].ID); err != nil {
		t.Fatalf("failed to delete post: %v", err)
	}
	if err := handlers.repo.Delete(posts[3].ID); err != nil {
		t.Fatalf("failed to delete post: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/scenes/"+sceneID+"/feed", nil)
	w := httptest.NewRecorder()

	handlers.GetSceneFeed(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var response FeedResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Should only return 3 posts (5 - 2 deleted)
	if len(response.Posts) != 3 {
		t.Errorf("expected 3 posts (deleted excluded), got %d", len(response.Posts))
	}

	// Verify the deleted posts are not in the feed
	for _, p := range response.Posts {
		if p.ID == posts[1].ID || p.ID == posts[3].ID {
			t.Errorf("found deleted post in feed: %s", p.ID)
		}
	}
}

// TestGetEventFeed_Success tests successful event feed retrieval.
func TestGetEventFeed_Success(t *testing.T) {
	handlers := newTestPostHandlers()

	eventID := "event123"

	// Create posts for this event
	for i := 0; i < 5; i++ {
		p := &post.Post{
			SceneID:   strPtr("scene123"),
			EventID:   &eventID,
			AuthorDID: "did:example:user1",
			Text:      "Event post " + string(rune('A'+i)),
			Labels:    []string{},
			CreatedAt: time.Now().Add(-time.Duration(i) * time.Hour),
		}
		if err := handlers.repo.Create(p); err != nil {
			t.Fatalf("failed to create post: %v", err)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/events/"+eventID+"/feed", nil)
	w := httptest.NewRecorder()

	handlers.GetEventFeed(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var response FeedResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(response.Posts) != 5 {
		t.Errorf("expected 5 posts, got %d", len(response.Posts))
	}

	// Verify all posts belong to this event
	for _, p := range response.Posts {
		if p.EventID == nil || *p.EventID != eventID {
			t.Errorf("post %s does not belong to event %s", p.ID, eventID)
		}
	}
}

// TestGetSceneFeed_InvalidLimit tests validation of limit parameter.
func TestGetSceneFeed_InvalidLimit(t *testing.T) {
	handlers := newTestPostHandlers()

	sceneID := "scene123"

	// Create a public scene for testing
	createTestSceneForFeed(handlers.sceneRepo, sceneID, "did:example:owner")

	tests := []struct {
		name       string
		limit      string
		expectCode int
	}{
		{"negative limit", "-1", http.StatusBadRequest},
		{"zero limit", "0", http.StatusBadRequest},
		{"non-numeric limit", "abc", http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/scenes/"+sceneID+"/feed?limit="+tt.limit, nil)
			w := httptest.NewRecorder()

			handlers.GetSceneFeed(w, req)

			if w.Code != tt.expectCode {
				t.Errorf("expected status %d, got %d", tt.expectCode, w.Code)
			}
		})
	}
}

// TestGetSceneFeed_MaxLimit tests that limit is capped at 100.
func TestGetSceneFeed_MaxLimit(t *testing.T) {
	handlers := newTestPostHandlers()

	sceneID := "scene123"

	// Create a public scene for testing
	createTestSceneForFeed(handlers.sceneRepo, "scene123", "did:example:owner")

	// Seed 150 posts
	seedTestPosts(handlers.repo, sceneID, "event123", 150)

	// Request with limit > 100
	req := httptest.NewRequest(http.MethodGet, "/scenes/"+sceneID+"/feed?limit=200", nil)
	w := httptest.NewRecorder()

	handlers.GetSceneFeed(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var response FeedResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Should be capped at 100
	if len(response.Posts) != 100 {
		t.Errorf("expected 100 posts (max limit), got %d", len(response.Posts))
	}
}

// TestGetSceneFeed_EmptyFeed tests behavior when there are no posts.
func TestGetSceneFeed_EmptyFeed(t *testing.T) {
	handlers := newTestPostHandlers()

	sceneID := "scene123"

	// Create a public scene for testing
	createTestSceneForFeed(handlers.sceneRepo, sceneID, "did:example:owner")

	req := httptest.NewRequest(http.MethodGet, "/scenes/"+sceneID+"/feed", nil)
	w := httptest.NewRecorder()

	handlers.GetSceneFeed(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var response FeedResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(response.Posts) != 0 {
		t.Errorf("expected 0 posts, got %d", len(response.Posts))
	}

	if response.NextCursor != nil {
		t.Error("expected next_cursor to be nil for empty feed")
	}
}

// encodeCursorString is a helper to encode a cursor for testing.
func encodeCursorString(cursor *post.FeedCursor) string {
	if cursor == nil {
		return ""
	}
	// Encode as "created_at_unix_nano:id"
	return fmt.Sprintf("%d:%s", cursor.CreatedAt.UnixNano(), cursor.ID)
}
