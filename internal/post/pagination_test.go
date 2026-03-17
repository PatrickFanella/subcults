package post

import (
	"fmt"
	"sort"
	"testing"
	"time"
)

// TestPostPagination_NoDuplicates tests that paginating through all posts
// produces no duplicates and captures all items.
func TestPostPagination_NoDuplicates(t *testing.T) {
	repo := NewInMemoryPostRepository()

	sceneID := "scene-123"
	totalPosts := 25
	expectedIDs := make(map[string]bool)

	// Create posts with varying content for scoring
	for i := 0; i < totalPosts; i++ {
		post := &Post{
			SceneID:   &sceneID,
			AuthorDID: "did:example:user1",
			Text:      fmt.Sprintf("Music post number %d with some content", i),
			Labels:    []string{},
			CreatedAt: time.Now().Add(-time.Duration(i) * time.Minute),
		}
		if err := repo.Create(post); err != nil {
			t.Fatalf("failed to create post %d: %v", i, err)
		}
		expectedIDs[post.ID] = true
	}

	// Paginate through all results
	pageSize := 10
	cursor := ""
	seenIDs := make(map[string]bool)
	pageCount := 0
	maxPages := 10

	for {
		pageCount++
		if pageCount > maxPages {
			t.Fatal("pagination exceeded max pages, possible infinite loop")
		}

		results, nextCursor, err := repo.SearchPosts("music", &sceneID, pageSize, cursor, nil)
		if err != nil {
			t.Fatalf("search failed on page %d: %v", pageCount, err)
		}

		// Check for duplicates
		for _, post := range results {
			if seenIDs[post.ID] {
				t.Errorf("duplicate post ID %s on page %d", post.ID, pageCount)
			}
			seenIDs[post.ID] = true
		}

		t.Logf("Page %d: %d results, cursor=%s", pageCount, len(results), nextCursor)

		if nextCursor == "" {
			break
		}
		cursor = nextCursor
	}

	// Verify we got all posts
	if len(seenIDs) != totalPosts {
		t.Errorf("expected %d unique posts, got %d", totalPosts, len(seenIDs))
	}

	for id := range expectedIDs {
		if !seenIDs[id] {
			t.Errorf("expected post ID %s was not returned", id)
		}
	}
}

// TestPostPagination_OrderingConsistency tests that pagination returns
// posts in consistent order across multiple runs.
func TestPostPagination_OrderingConsistency(t *testing.T) {
	repo := NewInMemoryPostRepository()

	sceneID := "scene-123"

	// Create posts
	for i := 0; i < 5; i++ {
		post := &Post{
			SceneID:   &sceneID,
			AuthorDID: "did:example:user1",
			Text:      fmt.Sprintf("Music post %d", i),
			Labels:    []string{},
			CreatedAt: time.Now().Add(-time.Duration(i) * time.Hour),
		}
		if err := repo.Create(post); err != nil {
			t.Fatalf("failed to create post: %v", err)
		}
	}

	// Run multiple times
	runs := 3
	var previousOrder []string

	for run := 1; run <= runs; run++ {
		var currentOrder []string
		cursor := ""

		for {
			results, nextCursor, err := repo.SearchPosts("music", &sceneID, 2, cursor, nil)
			if err != nil {
				t.Fatalf("run %d: search failed: %v", run, err)
			}

			for _, post := range results {
				currentOrder = append(currentOrder, post.ID)
			}

			if nextCursor == "" {
				break
			}
			cursor = nextCursor
		}

		if run > 1 {
			if len(currentOrder) != len(previousOrder) {
				t.Errorf("run %d: length mismatch: expected %d, got %d",
					run, len(previousOrder), len(currentOrder))
			}

			for i := 0; i < len(currentOrder) && i < len(previousOrder); i++ {
				if currentOrder[i] != previousOrder[i] {
					t.Errorf("run %d: ordering differs at position %d", run, i)
				}
			}
		}

		previousOrder = currentOrder
		t.Logf("Run %d order: %v", run, currentOrder)
	}
}

// TestPostPagination_ScoreTies tests that posts with identical scores
// are ordered deterministically by ID.
func TestPostPagination_ScoreTies(t *testing.T) {
	repo := NewInMemoryPostRepository()

	sceneID := "scene-123"
	postIDs := []string{}

	// Create posts with identical text (identical scores)
	for i := 0; i < 5; i++ {
		post := &Post{
			SceneID:   &sceneID,
			AuthorDID: "did:example:user1",
			Text:      "Identical music content", // Same text = same score
			Labels:    []string{},
			CreatedAt: time.Now(),
		}
		if err := repo.Create(post); err != nil {
			t.Fatalf("failed to create post: %v", err)
		}
		postIDs = append(postIDs, post.ID)
	}

	// Search
	results, _, err := repo.SearchPosts("music", &sceneID, 10, "", nil)
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}

	if len(results) != len(postIDs) {
		t.Fatalf("expected %d posts, got %d", len(postIDs), len(results))
	}

	// Verify ordering by ID (ascending when scores are tied)
	sortedExpected := make([]string, len(postIDs))
	copy(sortedExpected, postIDs)
	sort.Strings(sortedExpected)

	for i, post := range results {
		if post.ID != sortedExpected[i] {
			t.Errorf("position %d: expected %s, got %s (tie-breaking failed)",
				i, sortedExpected[i], post.ID)
		}
	}

	t.Logf("Score tie ordering: %v", sortedExpected)
}

// TestPostPagination_InsertionOrderIndependence tests that ordering within
// a single repository is deterministic regardless of when items were inserted.
func TestPostPagination_InsertionOrderIndependence(t *testing.T) {
	repo := NewInMemoryPostRepository()
	sceneID := "scene-123"

	// Create multiple posts with identical scores (same text)
	// They will be created at slightly different times but should still
	// be ordered deterministically by ID
	for i := 0; i < 10; i++ {
		post := &Post{
			SceneID:   &sceneID,
			AuthorDID: "did:example:user1",
			Text:      "music event", // Same text = same score
			Labels:    []string{},
		}
		if err := repo.Create(post); err != nil {
			t.Fatalf("failed to create post %d: %v", i, err)
		}
	}

	// Search multiple times - should return same order every time
	runs := 3
	var expectedOrder []string

	for run := 1; run <= runs; run++ {
		results, _, err := repo.SearchPosts("music", &sceneID, 20, "", nil)
		if err != nil {
			t.Fatalf("run %d: search failed: %v", run, err)
		}

		currentOrder := make([]string, len(results))
		for i, post := range results {
			currentOrder[i] = post.ID
		}

		if run == 1 {
			expectedOrder = currentOrder
			t.Logf("Run 1 order: %v", currentOrder)
		} else {
			// Verify exact same order
			if len(currentOrder) != len(expectedOrder) {
				t.Errorf("run %d: length mismatch", run)
			}

			for i := 0; i < len(currentOrder) && i < len(expectedOrder); i++ {
				if currentOrder[i] != expectedOrder[i] {
					t.Errorf("run %d: ordering differs at position %d: expected %s, got %s",
						run, i, expectedOrder[i], currentOrder[i])
				}
			}
			t.Logf("Run %d: order is consistent", run)
		}
	}
}

// TestPostCursor_RoundTrip tests cursor encoding and decoding for posts.
// Post cursors use base64-encoded JSON format.
func TestPostCursor_RoundTrip(t *testing.T) {
	testCases := []struct {
		name  string
		score float64
		id    string
	}{
		{
			name:  "integer score",
			score: 1.0,
			id:    "post-123",
		},
		{
			name:  "decimal score",
			score: 0.85432,
			id:    "post-456",
		},
		{
			name:  "high precision",
			score: 0.123456,
			id:    "post-abc",
		},
		{
			name:  "zero score",
			score: 0.0,
			id:    "post-000",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Encode
			encoded := EncodePostScoreCursor(tc.score, tc.id)

			// Decode
			decoded, err := DecodePostScoreCursor(encoded)
			if err != nil {
				t.Fatalf("failed to decode cursor: %v", err)
			}
			if decoded == nil {
				t.Fatal("decoded cursor is nil")
			}

			// Verify exact score round-trip
			if decoded.Score != tc.score {
				t.Errorf("score mismatch: expected %v, got %v", tc.score, decoded.Score)
			}

			if decoded.ID != tc.id {
				t.Errorf("ID mismatch: expected %s, got %s", tc.id, decoded.ID)
			}
		})
	}

	// Test empty cursor
	decoded, err := DecodePostScoreCursor("")
	if err != nil {
		t.Fatalf("empty cursor should not error: %v", err)
	}
	if decoded != nil {
		t.Error("empty cursor should decode to nil")
	}

	// Test invalid cursor
	_, err = DecodePostScoreCursor("not-valid-base64!!!")
	if err == nil {
		t.Error("invalid cursor should return error")
	}
}

// TestPostPagination_EmptyResults tests pagination with no results.
func TestPostPagination_EmptyResults(t *testing.T) {
	repo := NewInMemoryPostRepository()

	sceneID := "scene-123"

	results, cursor, err := repo.SearchPosts("nonexistent", &sceneID, 10, "", nil)
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}

	if cursor != "" {
		t.Errorf("expected empty cursor, got %s", cursor)
	}
}

// TestPostPagination_LastPageEmptyCursor tests that last page has empty cursor.
func TestPostPagination_LastPageEmptyCursor(t *testing.T) {
	repo := NewInMemoryPostRepository()

	sceneID := "scene-123"

	// Create exactly 5 posts
	for i := 0; i < 5; i++ {
		post := &Post{
			SceneID:   &sceneID,
			AuthorDID: "did:example:user1",
			Text:      fmt.Sprintf("Music post %d", i),
			Labels:    []string{},
			CreatedAt: time.Now(),
		}
		if err := repo.Create(post); err != nil {
			t.Fatalf("failed to create post: %v", err)
		}
	}

	// Request all in one page
	results, cursor, err := repo.SearchPosts("music", &sceneID, 10, "", nil)
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}

	if len(results) != 5 {
		t.Errorf("expected 5 results, got %d", len(results))
	}

	if cursor != "" {
		t.Errorf("expected empty cursor for last page, got %s", cursor)
	}
}

// TestPostPagination_SingleItemPage tests single item pagination.
func TestPostPagination_SingleItemPage(t *testing.T) {
	repo := NewInMemoryPostRepository()

	sceneID := "scene-123"

	post := &Post{
		SceneID:   &sceneID,
		AuthorDID: "did:example:user1",
		Text:      "Single music post",
		Labels:    []string{},
		CreatedAt: time.Now(),
	}
	if err := repo.Create(post); err != nil {
		t.Fatalf("failed to create post: %v", err)
	}

	results, cursor, err := repo.SearchPosts("music", &sceneID, 10, "", nil)
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}

	if cursor != "" {
		t.Errorf("expected empty cursor, got %s", cursor)
	}

	if results[0].ID != post.ID {
		t.Errorf("expected post ID %s, got %s", post.ID, results[0].ID)
	}
}

// TestPostPagination_WithTrustScores tests pagination stability with trust scores.
func TestPostPagination_WithTrustScores(t *testing.T) {
	repo := NewInMemoryPostRepository()

	sceneID := "scene-123"

	// Create posts with identical text
	postIDs := []string{}
	for i := 0; i < 5; i++ {
		post := &Post{
			SceneID:   &sceneID,
			AuthorDID: "did:example:user1",
			Text:      "Music event content",
			Labels:    []string{},
			CreatedAt: time.Now(),
		}
		if err := repo.Create(post); err != nil {
			t.Fatalf("failed to create post: %v", err)
		}
		postIDs = append(postIDs, post.ID)
	}

	trustScores := map[string]float64{
		sceneID: 0.8,
	}

	// Run pagination multiple times with trust scores
	runs := 3
	var expectedOrder []string

	for run := 1; run <= runs; run++ {
		var currentOrder []string
		cursor := ""

		for {
			results, nextCursor, err := repo.SearchPosts("music", &sceneID, 2, cursor, trustScores)
			if err != nil {
				t.Fatalf("run %d: search failed: %v", run, err)
			}

			for _, post := range results {
				currentOrder = append(currentOrder, post.ID)
			}

			if nextCursor == "" {
				break
			}
			cursor = nextCursor
		}

		if run == 1 {
			expectedOrder = currentOrder
			t.Logf("Run 1 order with trust: %v", currentOrder)
		} else {
			for i := 0; i < len(currentOrder) && i < len(expectedOrder); i++ {
				if currentOrder[i] != expectedOrder[i] {
					t.Errorf("run %d: ordering differs at position %d (trust scores affected stability)",
						run, i)
				}
			}
		}
	}
}

// TestPostPagination_HiddenExcluded tests that hidden posts are excluded.
func TestPostPagination_HiddenExcluded(t *testing.T) {
	repo := NewInMemoryPostRepository()

	sceneID := "scene-123"

	// Create visible posts
	visibleIDs := []string{}
	for i := 0; i < 3; i++ {
		post := &Post{
			SceneID:   &sceneID,
			AuthorDID: "did:example:user1",
			Text:      fmt.Sprintf("Music post %d", i),
			Labels:    []string{},
			CreatedAt: time.Now(),
		}
		if err := repo.Create(post); err != nil {
			t.Fatalf("failed to create post: %v", err)
		}
		visibleIDs = append(visibleIDs, post.ID)
	}

	// Create hidden posts
	for i := 0; i < 2; i++ {
		post := &Post{
			SceneID:   &sceneID,
			AuthorDID: "did:example:user1",
			Text:      "Hidden music post",
			Labels:    []string{LabelHidden},
			CreatedAt: time.Now(),
		}
		if err := repo.Create(post); err != nil {
			t.Fatalf("failed to create hidden post: %v", err)
		}
	}

	// Search should only return visible posts
	results, _, err := repo.SearchPosts("music", &sceneID, 10, "", nil)
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}

	if len(results) != 3 {
		t.Errorf("expected 3 visible posts, got %d", len(results))
	}

	// Verify no hidden posts in results
	for _, post := range results {
		if post.HasLabel(LabelHidden) {
			t.Errorf("found hidden post in results: %s", post.ID)
		}
	}
}

// TestPostPagination_DeletedExcluded tests that deleted posts are excluded.
func TestPostPagination_DeletedExcluded(t *testing.T) {
	repo := NewInMemoryPostRepository()

	sceneID := "scene-123"

	// Create posts
	posts := []*Post{}
	for i := 0; i < 5; i++ {
		post := &Post{
			SceneID:   &sceneID,
			AuthorDID: "did:example:user1",
			Text:      fmt.Sprintf("Music post %d", i),
			Labels:    []string{},
			CreatedAt: time.Now(),
		}
		if err := repo.Create(post); err != nil {
			t.Fatalf("failed to create post: %v", err)
		}
		posts = append(posts, post)
	}

	// Delete some posts
	if err := repo.Delete(posts[1].ID); err != nil {
		t.Fatalf("failed to delete post: %v", err)
	}
	if err := repo.Delete(posts[3].ID); err != nil {
		t.Fatalf("failed to delete post: %v", err)
	}

	// Search should return only non-deleted posts
	results, _, err := repo.SearchPosts("music", &sceneID, 10, "", nil)
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}

	if len(results) != 3 {
		t.Errorf("expected 3 non-deleted posts, got %d", len(results))
	}

	// Verify deleted posts not in results
	for _, post := range results {
		if post.ID == posts[1].ID || post.ID == posts[3].ID {
			t.Errorf("found deleted post in results: %s", post.ID)
		}
	}
}
