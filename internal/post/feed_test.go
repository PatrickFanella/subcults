package post

import (
	"testing"
	"time"
)

// TestListByScene_BasicFunctionality tests basic scene feed retrieval.
func TestListByScene_BasicFunctionality(t *testing.T) {
	repo := NewInMemoryPostRepository()
	sceneID := "scene123"

	// Create 5 posts for this scene
	now := time.Now()
	for i := 0; i < 5; i++ {
		post := &Post{
			SceneID:   &sceneID,
			AuthorDID: "did:example:user1",
			Text:      "Post " + string(rune('A'+i)),
			CreatedAt: now.Add(-time.Duration(i) * time.Hour), // Each post 1 hour older
		}
		if err := repo.Create(post); err != nil {
			t.Fatalf("failed to create post: %v", err)
		}
	}

	// Retrieve all posts
	posts, nextCursor, err := repo.ListByScene(sceneID, 10, nil)
	if err != nil {
		t.Fatalf("ListByScene failed: %v", err)
	}

	if len(posts) != 5 {
		t.Errorf("expected 5 posts, got %d", len(posts))
	}

	if nextCursor != nil {
		t.Error("expected nil cursor (no more posts)")
	}

	// Verify ordering: newest first
	for i := 1; i < len(posts); i++ {
		if posts[i-1].CreatedAt.Before(posts[i].CreatedAt) {
			t.Errorf("posts not ordered correctly: post %d is older than post %d", i-1, i)
		}
	}
}

// TestListByScene_Pagination tests cursor-based pagination.
func TestListByScene_Pagination(t *testing.T) {
	repo := NewInMemoryPostRepository()
	sceneID := "scene123"

	// Create 25 posts
	now := time.Now()
	for i := 0; i < 25; i++ {
		post := &Post{
			SceneID:   &sceneID,
			AuthorDID: "did:example:user1",
			Text:      "Post " + string(rune('A'+i)),
			CreatedAt: now.Add(-time.Duration(i) * time.Hour),
		}
		if err := repo.Create(post); err != nil {
			t.Fatalf("failed to create post: %v", err)
		}
	}

	// Get first page
	page1, cursor1, err := repo.ListByScene(sceneID, 10, nil)
	if err != nil {
		t.Fatalf("ListByScene page 1 failed: %v", err)
	}

	if len(page1) != 10 {
		t.Errorf("expected 10 posts on page 1, got %d", len(page1))
	}

	if cursor1 == nil {
		t.Fatal("expected cursor to be set")
	}

	// Get second page
	page2, cursor2, err := repo.ListByScene(sceneID, 10, cursor1)
	if err != nil {
		t.Fatalf("ListByScene page 2 failed: %v", err)
	}

	if len(page2) != 10 {
		t.Errorf("expected 10 posts on page 2, got %d", len(page2))
	}

	if cursor2 == nil {
		t.Fatal("expected cursor to be set for page 2")
	}

	// Get third page
	page3, cursor3, err := repo.ListByScene(sceneID, 10, cursor2)
	if err != nil {
		t.Fatalf("ListByScene page 3 failed: %v", err)
	}

	if len(page3) != 5 {
		t.Errorf("expected 5 posts on page 3, got %d", len(page3))
	}

	if cursor3 != nil {
		t.Error("expected nil cursor on last page")
	}

	// Verify no duplicates across pages
	seen := make(map[string]bool)
	for _, p := range append(append(page1, page2...), page3...) {
		if seen[p.ID] {
			t.Errorf("duplicate post ID found: %s", p.ID)
		}
		seen[p.ID] = true
	}
}

// TestListByScene_HiddenExcluded tests that hidden posts are excluded.
func TestListByScene_HiddenExcluded(t *testing.T) {
	repo := NewInMemoryPostRepository()
	sceneID := "scene123"

	// Create 3 normal posts
	now := time.Now()
	for i := 0; i < 3; i++ {
		post := &Post{
			SceneID:   &sceneID,
			AuthorDID: "did:example:user1",
			Text:      "Normal post",
			Labels:    []string{},
			CreatedAt: now.Add(-time.Duration(i) * time.Hour),
		}
		if err := repo.Create(post); err != nil {
			t.Fatalf("failed to create post: %v", err)
		}
	}

	// Create 2 hidden posts
	for i := 0; i < 2; i++ {
		post := &Post{
			SceneID:   &sceneID,
			AuthorDID: "did:example:user1",
			Text:      "Hidden post",
			Labels:    []string{LabelHidden},
			CreatedAt: now.Add(time.Duration(i+1) * time.Hour), // Newer than normal posts
		}
		if err := repo.Create(post); err != nil {
			t.Fatalf("failed to create hidden post: %v", err)
		}
	}

	// Retrieve posts
	posts, _, err := repo.ListByScene(sceneID, 10, nil)
	if err != nil {
		t.Fatalf("ListByScene failed: %v", err)
	}

	// Should only return 3 normal posts
	if len(posts) != 3 {
		t.Errorf("expected 3 posts (hidden excluded), got %d", len(posts))
	}

	// Verify none have hidden label
	for _, p := range posts {
		if p.HasLabel(LabelHidden) {
			t.Errorf("found hidden post in results: %s", p.ID)
		}
	}
}

// TestListByScene_DeletedExcluded tests that soft-deleted posts are excluded.
func TestListByScene_DeletedExcluded(t *testing.T) {
	repo := NewInMemoryPostRepository()
	sceneID := "scene123"

	// Create 5 posts
	var postIDs []string
	now := time.Now()
	for i := 0; i < 5; i++ {
		post := &Post{
			SceneID:   &sceneID,
			AuthorDID: "did:example:user1",
			Text:      "Post " + string(rune('A'+i)),
			CreatedAt: now.Add(-time.Duration(i) * time.Hour),
		}
		if err := repo.Create(post); err != nil {
			t.Fatalf("failed to create post: %v", err)
		}
		postIDs = append(postIDs, post.ID)
	}

	// Delete 2 posts
	if err := repo.Delete(postIDs[1]); err != nil {
		t.Fatalf("failed to delete post: %v", err)
	}
	if err := repo.Delete(postIDs[3]); err != nil {
		t.Fatalf("failed to delete post: %v", err)
	}

	// Retrieve posts
	posts, _, err := repo.ListByScene(sceneID, 10, nil)
	if err != nil {
		t.Fatalf("ListByScene failed: %v", err)
	}

	// Should only return 3 posts
	if len(posts) != 3 {
		t.Errorf("expected 3 posts (deleted excluded), got %d", len(posts))
	}

	// Verify deleted posts are not in results
	for _, p := range posts {
		if p.ID == postIDs[1] || p.ID == postIDs[3] {
			t.Errorf("found deleted post in results: %s", p.ID)
		}
	}
}

// TestListByEvent_BasicFunctionality tests basic event feed retrieval.
func TestListByEvent_BasicFunctionality(t *testing.T) {
	repo := NewInMemoryPostRepository()
	eventID := "event123"

	// Create 5 posts for this event
	now := time.Now()
	for i := 0; i < 5; i++ {
		post := &Post{
			SceneID:   strPtr("scene123"),
			EventID:   &eventID,
			AuthorDID: "did:example:user1",
			Text:      "Post " + string(rune('A'+i)),
			CreatedAt: now.Add(-time.Duration(i) * time.Hour),
		}
		if err := repo.Create(post); err != nil {
			t.Fatalf("failed to create post: %v", err)
		}
	}

	// Retrieve all posts
	posts, nextCursor, err := repo.ListByEvent(eventID, 10, nil)
	if err != nil {
		t.Fatalf("ListByEvent failed: %v", err)
	}

	if len(posts) != 5 {
		t.Errorf("expected 5 posts, got %d", len(posts))
	}

	if nextCursor != nil {
		t.Error("expected nil cursor (no more posts)")
	}

	// Verify all posts belong to this event
	for _, p := range posts {
		if p.EventID == nil || *p.EventID != eventID {
			t.Errorf("post %s does not belong to event %s", p.ID, eventID)
		}
	}
}

// TestListByScene_EmptyResult tests behavior when no posts exist.
func TestListByScene_EmptyResult(t *testing.T) {
	repo := NewInMemoryPostRepository()
	sceneID := "scene123"

	posts, cursor, err := repo.ListByScene(sceneID, 10, nil)
	if err != nil {
		t.Fatalf("ListByScene failed: %v", err)
	}

	if len(posts) != 0 {
		t.Errorf("expected 0 posts, got %d", len(posts))
	}

	if cursor != nil {
		t.Error("expected nil cursor for empty result")
	}
}

// TestListByScene_TieBreaking tests ID-based tie-breaking for posts with same timestamp.
func TestListByScene_TieBreaking(t *testing.T) {
	repo := NewInMemoryPostRepository()
	sceneID := "scene123"

	// Create posts with identical timestamps but different IDs
	now := time.Now()
	var postIDs []string
	for i := 0; i < 3; i++ {
		post := &Post{
			SceneID:   &sceneID,
			AuthorDID: "did:example:user1",
			Text:      "Post " + string(rune('A'+i)),
		}
		if err := repo.Create(post); err != nil {
			t.Fatalf("failed to create post: %v", err)
		}
		// Manually override the created timestamp to be the same for all posts
		// This simulates posts created at exactly the same time
		retrievedPost, err := repo.GetByID(post.ID)
		if err != nil {
			t.Fatalf("failed to retrieve post: %v", err)
		}
		// Access the internal map to set timestamp (in production DB, this would be set by DB)
		repo.mu.Lock()
		repo.posts[retrievedPost.ID].CreatedAt = now
		repo.mu.Unlock()

		postIDs = append(postIDs, post.ID)
	}

	// Retrieve posts
	posts, _, err := repo.ListByScene(sceneID, 10, nil)
	if err != nil {
		t.Fatalf("ListByScene failed: %v", err)
	}

	if len(posts) != 3 {
		t.Fatalf("expected 3 posts, got %d", len(posts))
	}

	// Verify all have same timestamp
	for i := 1; i < len(posts); i++ {
		if !posts[i].CreatedAt.Equal(posts[0].CreatedAt) {
			t.Errorf("posts do not have same timestamp: %v vs %v", posts[i].CreatedAt, posts[0].CreatedAt)
		}
	}

	// Verify IDs are in ascending order (lexicographic)
	for i := 1; i < len(posts); i++ {
		if posts[i-1].ID >= posts[i].ID {
			t.Errorf("IDs not in ascending order: %s >= %s", posts[i-1].ID, posts[i].ID)
		}
	}
}

// TestListByScene_OtherSceneExcluded tests that posts from other scenes are excluded.
func TestListByScene_OtherSceneExcluded(t *testing.T) {
	repo := NewInMemoryPostRepository()
	targetScene := "scene123"
	otherScene := "scene456"

	// Create 3 posts for target scene
	now := time.Now()
	for i := 0; i < 3; i++ {
		post := &Post{
			SceneID:   &targetScene,
			AuthorDID: "did:example:user1",
			Text:      "Target scene post",
			CreatedAt: now.Add(-time.Duration(i) * time.Hour),
		}
		if err := repo.Create(post); err != nil {
			t.Fatalf("failed to create post: %v", err)
		}
	}

	// Create 2 posts for other scene
	for i := 0; i < 2; i++ {
		post := &Post{
			SceneID:   &otherScene,
			AuthorDID: "did:example:user1",
			Text:      "Other scene post",
			CreatedAt: now.Add(-time.Duration(i) * time.Hour),
		}
		if err := repo.Create(post); err != nil {
			t.Fatalf("failed to create post: %v", err)
		}
	}

	// Retrieve posts for target scene
	posts, _, err := repo.ListByScene(targetScene, 10, nil)
	if err != nil {
		t.Fatalf("ListByScene failed: %v", err)
	}

	// Should only return 3 posts from target scene
	if len(posts) != 3 {
		t.Errorf("expected 3 posts, got %d", len(posts))
	}

	// Verify all posts belong to target scene
	for _, p := range posts {
		if p.SceneID == nil || *p.SceneID != targetScene {
			t.Errorf("post %s does not belong to target scene", p.ID)
		}
	}
}

// TestCursorIntegrity_DeletedPostAfterCursor tests that cursor pagination remains stable
// when a post is deleted after the cursor is captured.
// Expected behavior: deleted post is skipped, no duplicates or unintended skips.
func TestCursorIntegrity_DeletedPostAfterCursor(t *testing.T) {
	repo := NewInMemoryPostRepository()
	sceneID := "scene123"

	// Create 10 posts with 1-hour intervals
	now := time.Now()
	var postIDs []string
	for i := 0; i < 10; i++ {
		post := &Post{
			SceneID:   &sceneID,
			AuthorDID: "did:example:user1",
			Text:      "Post " + string(rune('A'+i)),
		}
		if err := repo.Create(post); err != nil {
			t.Fatalf("failed to create post: %v", err)
		}
		// Manually set timestamp to simulate posts at different times
		repo.mu.Lock()
		repo.posts[post.ID].CreatedAt = now.Add(-time.Duration(i) * time.Hour)
		repo.mu.Unlock()

		postIDs = append(postIDs, post.ID)
	}

	// Get first page (limit 4)
	page1, cursor1, err := repo.ListByScene(sceneID, 4, nil)
	if err != nil {
		t.Fatalf("ListByScene page 1 failed: %v", err)
	}

	if len(page1) != 4 {
		t.Fatalf("expected 4 posts on page 1, got %d", len(page1))
	}

	// Capture IDs from page 1 for later verification
	page1IDs := make([]string, len(page1))
	for i, p := range page1 {
		page1IDs[i] = p.ID
	}

	// Delete the 6th post (index 5, which would be on page 2)
	if err := repo.Delete(postIDs[5]); err != nil {
		t.Fatalf("failed to delete post: %v", err)
	}

	// Get second page using cursor1 - should skip deleted post
	page2, cursor2, err := repo.ListByScene(sceneID, 4, cursor1)
	if err != nil {
		t.Fatalf("ListByScene page 2 failed: %v", err)
	}

	// Should still get 4 posts (skipping the deleted one)
	if len(page2) != 4 {
		t.Errorf("expected 4 posts on page 2 (skipping deleted), got %d", len(page2))
	}

	// Verify no post from page1 appears in page2 (no duplicates)
	for _, p1 := range page1 {
		for _, p2 := range page2 {
			if p1.ID == p2.ID {
				t.Errorf("duplicate post found: %s appeared in both page1 and page2", p1.ID)
			}
		}
	}

	// Get third page
	page3, cursor3, err := repo.ListByScene(sceneID, 4, cursor2)
	if err != nil {
		t.Fatalf("ListByScene page 3 failed: %v", err)
	}

	// Should get remaining posts (10 total - 1 deleted - 8 from pages 1&2 = 1 post)
	if len(page3) != 1 {
		t.Errorf("expected 1 post on page 3, got %d", len(page3))
	}

	if cursor3 != nil {
		t.Error("expected nil cursor on last page")
	}

	// Verify deleted post does not appear anywhere
	deletedID := postIDs[5]
	allRetrievedPosts := append(append(page1, page2...), page3...)
	for _, p := range allRetrievedPosts {
		if p.ID == deletedID {
			t.Errorf("deleted post %s appeared in results", deletedID)
		}
	}
}

// TestCursorIntegrity_HiddenPostAfterCursor tests that cursor pagination remains stable
// when a post is hidden via label change after the cursor is captured.
// Expected behavior: hidden post is skipped, no duplicates or unintended skips.
func TestCursorIntegrity_HiddenPostAfterCursor(t *testing.T) {
	repo := NewInMemoryPostRepository()
	sceneID := "scene123"

	// Create 10 posts with 1-hour intervals
	now := time.Now()
	var postIDs []string
	for i := 0; i < 10; i++ {
		post := &Post{
			SceneID:   &sceneID,
			AuthorDID: "did:example:user1",
			Text:      "Post " + string(rune('A'+i)),
			Labels:    []string{}, // Initially no labels
		}
		if err := repo.Create(post); err != nil {
			t.Fatalf("failed to create post: %v", err)
		}
		// Manually set timestamp
		repo.mu.Lock()
		repo.posts[post.ID].CreatedAt = now.Add(-time.Duration(i) * time.Hour)
		repo.mu.Unlock()

		postIDs = append(postIDs, post.ID)
	}

	// Get first page (limit 4)
	page1, cursor1, err := repo.ListByScene(sceneID, 4, nil)
	if err != nil {
		t.Fatalf("ListByScene page 1 failed: %v", err)
	}

	if len(page1) != 4 {
		t.Fatalf("expected 4 posts on page 1, got %d", len(page1))
	}

	// Get the post that would be on page 2 and hide it
	postToHide, err := repo.GetByID(postIDs[5])
	if err != nil {
		t.Fatalf("failed to get post to hide: %v", err)
	}
	postToHide.Labels = []string{LabelHidden}
	if err := repo.Update(postToHide); err != nil {
		t.Fatalf("failed to hide post: %v", err)
	}

	// Get second page using cursor1 - should skip hidden post
	page2, cursor2, err := repo.ListByScene(sceneID, 4, cursor1)
	if err != nil {
		t.Fatalf("ListByScene page 2 failed: %v", err)
	}

	// Should still get 4 posts (skipping the hidden one)
	if len(page2) != 4 {
		t.Errorf("expected 4 posts on page 2 (skipping hidden), got %d", len(page2))
	}

	// Verify no post from page1 appears in page2 (no duplicates)
	for _, p1 := range page1 {
		for _, p2 := range page2 {
			if p1.ID == p2.ID {
				t.Errorf("duplicate post found: %s appeared in both page1 and page2", p1.ID)
			}
		}
	}

	// Verify hidden post does not appear in page2
	hiddenID := postIDs[5]
	for _, p := range page2 {
		if p.ID == hiddenID {
			t.Errorf("hidden post %s appeared in page2", hiddenID)
		}
	}

	// Get third page
	page3, cursor3, err := repo.ListByScene(sceneID, 4, cursor2)
	if err != nil {
		t.Fatalf("ListByScene page 3 failed: %v", err)
	}

	// Should get remaining posts (10 total - 1 hidden - 8 from pages 1&2 = 1 post)
	if len(page3) != 1 {
		t.Errorf("expected 1 post on page 3, got %d", len(page3))
	}

	if cursor3 != nil {
		t.Error("expected nil cursor on last page")
	}

	// Verify hidden post does not appear anywhere
	allRetrievedPosts := append(append(page1, page2...), page3...)
	for _, p := range allRetrievedPosts {
		if p.ID == hiddenID {
			t.Errorf("hidden post %s appeared in results", hiddenID)
		}
	}
}

// TestCursorIntegrity_NewPostBeforeCursor tests cursor behavior when a new post
// is inserted with a timestamp that would place it chronologically earlier than
// the cursor position (i.e., newer than current cursor).
// Expected behavior: The post should NOT appear in ongoing pagination using existing cursors.
// It only appears when refreshing from the beginning (no cursor).
func TestCursorIntegrity_NewPostBeforeCursor(t *testing.T) {
	repo := NewInMemoryPostRepository()
	sceneID := "scene123"

	// Create 8 posts with 1-hour intervals
	// Post 0: now - 0h (newest)
	// Post 1: now - 1h
	// Post 2: now - 2h
	// Post 3: now - 3h  <- end of page 1
	// Post 4: now - 4h
	// Post 5: now - 5h
	// Post 6: now - 6h
	// Post 7: now - 7h  <- end of page 2
	now := time.Now()
	for i := 0; i < 8; i++ {
		post := &Post{
			SceneID:   &sceneID,
			AuthorDID: "did:example:user1",
			Text:      "Post " + string(rune('A'+i)),
		}
		if err := repo.Create(post); err != nil {
			t.Fatalf("failed to create post: %v", err)
		}
		// Manually set timestamp
		repo.mu.Lock()
		repo.posts[post.ID].CreatedAt = now.Add(-time.Duration(i) * time.Hour)
		repo.mu.Unlock()
	}

	// Get first page (limit 4) - should return posts 0,1,2,3
	page1, cursor1, err := repo.ListByScene(sceneID, 4, nil)
	if err != nil {
		t.Fatalf("ListByScene page 1 failed: %v", err)
	}

	if len(page1) != 4 {
		t.Fatalf("expected 4 posts on page 1, got %d", len(page1))
	}

	// Capture page1 IDs
	page1IDs := make(map[string]bool)
	for _, p := range page1 {
		page1IDs[p.ID] = true
	}

	// Insert a new post with timestamp that would make it appear between
	// posts we've already seen on page 1 (newer than cursor1)
	// This simulates someone posting "now" but the server timestamp or user-provided
	// timestamp places it slightly in the past
	newPost := &Post{
		SceneID:   &sceneID,
		AuthorDID: "did:example:user1",
		Text:      "New post inserted after page1 retrieved",
	}
	if err := repo.Create(newPost); err != nil {
		t.Fatalf("failed to create new post: %v", err)
	}
	// Set timestamp to be between post 2 and post 3
	repo.mu.Lock()
	repo.posts[newPost.ID].CreatedAt = now.Add(-2*time.Hour - 30*time.Minute)
	repo.mu.Unlock()

	// Get second page using cursor1 - should NOT include the new post
	// because new post timestamp is newer than cursor1 timestamp
	page2, cursor2, err := repo.ListByScene(sceneID, 4, cursor1)
	if err != nil {
		t.Fatalf("ListByScene page 2 failed: %v", err)
	}

	// Should get 4 posts (original posts 4,5,6,7)
	if len(page2) != 4 {
		t.Errorf("expected 4 posts on page 2, got %d", len(page2))
	}

	// Verify the new post does not appear in page2
	for _, p := range page2 {
		if p.ID == newPost.ID {
			t.Errorf("new post %s incorrectly appeared in page2 (should be excluded by cursor)", newPost.ID)
		}
		// Also verify no duplicates from page1
		if page1IDs[p.ID] {
			t.Errorf("duplicate post %s from page1 appeared in page2", p.ID)
		}
	}

	// Cursor2 should be nil (no more posts after page 2, since we have exactly 8 posts)
	if cursor2 != nil {
		t.Errorf("expected nil cursor2 (no more pages), got cursor with ID=%s", cursor2.ID)
	}

	// Now refresh from the beginning - new post SHOULD appear
	refreshPage1, _, err := repo.ListByScene(sceneID, 10, nil)
	if err != nil {
		t.Fatalf("ListByScene refresh failed: %v", err)
	}

	// Should now see 9 posts (original 8 + 1 new)
	if len(refreshPage1) != 9 {
		t.Errorf("expected 9 posts on refresh, got %d", len(refreshPage1))
	}

	// Verify new post appears in refreshed results
	foundNewPost := false
	for _, p := range refreshPage1 {
		if p.ID == newPost.ID {
			foundNewPost = true
			break
		}
	}
	if !foundNewPost {
		t.Error("new post not found in refreshed results (should appear when starting from beginning)")
	}
}

// TestCursorIntegrity_MultipleMutations tests cursor stability under multiple concurrent
// mutations (delete + hide + insert).
// Expected behavior: pagination continues correctly, no duplicates, no unintended skips.
func TestCursorIntegrity_MultipleMutations(t *testing.T) {
	repo := NewInMemoryPostRepository()
	sceneID := "scene123"

	// Create 15 posts with 1-hour intervals
	now := time.Now()
	var postIDs []string
	for i := 0; i < 15; i++ {
		post := &Post{
			SceneID:   &sceneID,
			AuthorDID: "did:example:user1",
			Text:      "Post " + string(rune('A'+i)),
			Labels:    []string{},
		}
		if err := repo.Create(post); err != nil {
			t.Fatalf("failed to create post: %v", err)
		}
		// Manually set timestamp
		repo.mu.Lock()
		repo.posts[post.ID].CreatedAt = now.Add(-time.Duration(i) * time.Hour)
		repo.mu.Unlock()

		postIDs = append(postIDs, post.ID)
	}

	// Get first page (limit 5)
	page1, cursor1, err := repo.ListByScene(sceneID, 5, nil)
	if err != nil {
		t.Fatalf("ListByScene page 1 failed: %v", err)
	}

	if len(page1) != 5 {
		t.Fatalf("expected 5 posts on page 1, got %d", len(page1))
	}

	// Capture page1 IDs
	page1IDs := make(map[string]bool)
	for _, p := range page1 {
		page1IDs[p.ID] = true
	}

	// Perform multiple mutations:
	// 1. Delete post at index 6 (would be on page 2)
	if err := repo.Delete(postIDs[6]); err != nil {
		t.Fatalf("failed to delete post 6: %v", err)
	}

	// 2. Hide post at index 8 (would be on page 2)
	postToHide, err := repo.GetByID(postIDs[8])
	if err != nil {
		t.Fatalf("failed to get post 8: %v", err)
	}
	postToHide.Labels = []string{LabelHidden}
	if err := repo.Update(postToHide); err != nil {
		t.Fatalf("failed to hide post 8: %v", err)
	}

	// 3. Insert new post with timestamp before cursor
	newPost := &Post{
		SceneID:   &sceneID,
		AuthorDID: "did:example:user1",
		Text:      "New post inserted",
	}
	if err := repo.Create(newPost); err != nil {
		t.Fatalf("failed to create new post: %v", err)
	}
	// Set timestamp to be between existing posts
	repo.mu.Lock()
	repo.posts[newPost.ID].CreatedAt = now.Add(-3*time.Hour - 30*time.Minute)
	repo.mu.Unlock()

	// Get second page - should skip deleted and hidden, not show new post
	page2, cursor2, err := repo.ListByScene(sceneID, 5, cursor1)
	if err != nil {
		t.Fatalf("ListByScene page 2 failed: %v", err)
	}

	// Should get 5 posts (skipping deleted and hidden)
	if len(page2) != 5 {
		t.Errorf("expected 5 posts on page 2, got %d", len(page2))
	}

	// Verify no duplicates from page1
	for _, p := range page2 {
		if page1IDs[p.ID] {
			t.Errorf("duplicate post %s found in page2", p.ID)
		}
		// Verify deleted and hidden posts not in page2
		if p.ID == postIDs[6] {
			t.Errorf("deleted post %s appeared in page2", p.ID)
		}
		if p.ID == postIDs[8] {
			t.Errorf("hidden post %s appeared in page2", p.ID)
		}
		// Verify new post not in page2
		if p.ID == newPost.ID {
			t.Errorf("new post %s incorrectly appeared in page2", newPost.ID)
		}
	}

	// Get third page
	page3, cursor3, err := repo.ListByScene(sceneID, 5, cursor2)
	if err != nil {
		t.Fatalf("ListByScene page 3 failed: %v", err)
	}

	// Should get remaining posts (15 original - 1 deleted - 1 hidden - 10 from pages 1&2 = 3 posts)
	if len(page3) != 3 {
		t.Errorf("expected 3 posts on page 3, got %d", len(page3))
	}

	if cursor3 != nil {
		t.Error("expected nil cursor on last page")
	}

	// Verify no deleted/hidden posts anywhere
	allRetrievedPosts := append(append(page1, page2...), page3...)
	for _, p := range allRetrievedPosts {
		if p.ID == postIDs[6] {
			t.Errorf("deleted post %s appeared in results", postIDs[6])
		}
		if p.ID == postIDs[8] {
			t.Errorf("hidden post %s appeared in results", postIDs[8])
		}
		if p.ID == newPost.ID {
			t.Errorf("new post %s appeared in paginated results (should only appear on refresh)", newPost.ID)
		}
	}

	// Verify total count is correct (13 unique posts: 15 - 1 deleted - 1 hidden)
	uniqueIDs := make(map[string]bool)
	for _, p := range allRetrievedPosts {
		uniqueIDs[p.ID] = true
	}
	if len(uniqueIDs) != 13 {
		t.Errorf("expected 13 unique posts, got %d", len(uniqueIDs))
	}
}

// TestCursorIntegrity_EventFeedDeletedPost tests cursor stability for event feeds
// when posts are deleted after cursor is captured.
func TestCursorIntegrity_EventFeedDeletedPost(t *testing.T) {
	repo := NewInMemoryPostRepository()
	eventID := "event123"
	sceneID := "scene123"

	// Create 10 posts for this event
	now := time.Now()
	var postIDs []string
	for i := 0; i < 10; i++ {
		post := &Post{
			SceneID:   &sceneID,
			EventID:   &eventID,
			AuthorDID: "did:example:user1",
			Text:      "Post " + string(rune('A'+i)),
		}
		if err := repo.Create(post); err != nil {
			t.Fatalf("failed to create post: %v", err)
		}
		// Manually set timestamp
		repo.mu.Lock()
		repo.posts[post.ID].CreatedAt = now.Add(-time.Duration(i) * time.Hour)
		repo.mu.Unlock()

		postIDs = append(postIDs, post.ID)
	}

	// Get first page (limit 4)
	page1, cursor1, err := repo.ListByEvent(eventID, 4, nil)
	if err != nil {
		t.Fatalf("ListByEvent page 1 failed: %v", err)
	}

	if len(page1) != 4 {
		t.Fatalf("expected 4 posts on page 1, got %d", len(page1))
	}

	// Delete a post that would be on page 2
	if err := repo.Delete(postIDs[5]); err != nil {
		t.Fatalf("failed to delete post: %v", err)
	}

	// Get second page - should skip deleted post
	page2, cursor2, err := repo.ListByEvent(eventID, 4, cursor1)
	if err != nil {
		t.Fatalf("ListByEvent page 2 failed: %v", err)
	}

	// Should get 4 posts (skipping deleted)
	if len(page2) != 4 {
		t.Errorf("expected 4 posts on page 2, got %d", len(page2))
	}

	// Verify no duplicates
	for _, p1 := range page1 {
		for _, p2 := range page2 {
			if p1.ID == p2.ID {
				t.Errorf("duplicate post %s in page1 and page2", p1.ID)
			}
		}
	}

	// Get third page
	page3, cursor3, err := repo.ListByEvent(eventID, 4, cursor2)
	if err != nil {
		t.Fatalf("ListByEvent page 3 failed: %v", err)
	}

	// Should get remaining post (10 - 1 deleted - 8 from pages 1&2 = 1)
	if len(page3) != 1 {
		t.Errorf("expected 1 post on page 3, got %d", len(page3))
	}

	if cursor3 != nil {
		t.Error("expected nil cursor on last page")
	}

	// Verify deleted post not in results
	allPosts := append(append(page1, page2...), page3...)
	for _, p := range allPosts {
		if p.ID == postIDs[5] {
			t.Errorf("deleted post %s appeared in results", postIDs[5])
		}
	}
}

// TestCursorIntegrity_IdenticalTimestampsPagination tests cursor pagination with posts
// that have identical timestamps, validating the ID-based tie-breaking logic works
// correctly across page boundaries.
func TestCursorIntegrity_IdenticalTimestampsPagination(t *testing.T) {
	repo := NewInMemoryPostRepository()
	sceneID := "scene123"

	// Create 12 posts, all with the same timestamp
	now := time.Now()
	var postIDs []string
	for i := 0; i < 12; i++ {
		post := &Post{
			SceneID:   &sceneID,
			AuthorDID: "did:example:user1",
			Text:      "Post " + string(rune('A'+i)),
		}
		if err := repo.Create(post); err != nil {
			t.Fatalf("failed to create post: %v", err)
		}
		// Set all posts to the same timestamp
		repo.mu.Lock()
		repo.posts[post.ID].CreatedAt = now
		repo.mu.Unlock()

		postIDs = append(postIDs, post.ID)
	}

	// Get first page (limit 5)
	page1, cursor1, err := repo.ListByScene(sceneID, 5, nil)
	if err != nil {
		t.Fatalf("ListByScene page 1 failed: %v", err)
	}

	if len(page1) != 5 {
		t.Fatalf("expected 5 posts on page 1, got %d", len(page1))
	}

	// Verify all posts on page 1 have the same timestamp
	for i := 1; i < len(page1); i++ {
		if !page1[i].CreatedAt.Equal(page1[0].CreatedAt) {
			t.Errorf("posts on page 1 don't have same timestamp: %v vs %v", page1[i].CreatedAt, page1[0].CreatedAt)
		}
	}

	// Verify IDs on page 1 are in ascending order (lexicographic)
	for i := 1; i < len(page1); i++ {
		if page1[i-1].ID >= page1[i].ID {
			t.Errorf("IDs on page 1 not in ascending order: %s >= %s", page1[i-1].ID, page1[i].ID)
		}
	}

	// Get second page using cursor1
	page2, cursor2, err := repo.ListByScene(sceneID, 5, cursor1)
	if err != nil {
		t.Fatalf("ListByScene page 2 failed: %v", err)
	}

	if len(page2) != 5 {
		t.Errorf("expected 5 posts on page 2, got %d", len(page2))
	}

	// Verify all posts on page 2 have the same timestamp
	for i := 1; i < len(page2); i++ {
		if !page2[i].CreatedAt.Equal(page2[0].CreatedAt) {
			t.Errorf("posts on page 2 don't have same timestamp: %v vs %v", page2[i].CreatedAt, page2[0].CreatedAt)
		}
	}

	// Verify IDs on page 2 are in ascending order
	for i := 1; i < len(page2); i++ {
		if page2[i-1].ID >= page2[i].ID {
			t.Errorf("IDs on page 2 not in ascending order: %s >= %s", page2[i-1].ID, page2[i].ID)
		}
	}

	// Verify no duplicates between page 1 and page 2
	page1IDs := make(map[string]bool)
	for _, p := range page1 {
		page1IDs[p.ID] = true
	}
	for _, p := range page2 {
		if page1IDs[p.ID] {
			t.Errorf("duplicate post %s found in both page1 and page2", p.ID)
		}
	}

	// Verify IDs across pages are properly ordered (last ID of page1 < first ID of page2)
	lastPage1ID := page1[len(page1)-1].ID
	firstPage2ID := page2[0].ID
	if lastPage1ID >= firstPage2ID {
		t.Errorf("pagination boundary violated: last page1 ID (%s) >= first page2 ID (%s)", lastPage1ID, firstPage2ID)
	}

	// Get third page
	page3, cursor3, err := repo.ListByScene(sceneID, 5, cursor2)
	if err != nil {
		t.Fatalf("ListByScene page 3 failed: %v", err)
	}

	// Should get remaining 2 posts
	if len(page3) != 2 {
		t.Errorf("expected 2 posts on page 3, got %d", len(page3))
	}

	if cursor3 != nil {
		t.Error("expected nil cursor on last page")
	}

	// Verify IDs on page 3 are in ascending order
	if len(page3) == 2 && page3[0].ID >= page3[1].ID {
		t.Errorf("IDs on page 3 not in ascending order: %s >= %s", page3[0].ID, page3[1].ID)
	}

	// Verify total unique posts = 12
	allPosts := append(append(page1, page2...), page3...)
	uniqueIDs := make(map[string]bool)
	for _, p := range allPosts {
		if uniqueIDs[p.ID] {
			t.Errorf("duplicate post ID found: %s", p.ID)
		}
		uniqueIDs[p.ID] = true
	}
	if len(uniqueIDs) != 12 {
		t.Errorf("expected 12 unique posts, got %d", len(uniqueIDs))
	}
}
