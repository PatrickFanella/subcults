package scene

import (
	"fmt"
	"math/rand"
	"sort"
	"testing"
	"time"

	"github.com/google/uuid"
)

// TestScenePagination_NoDuplicates tests that paginating through all scenes
// produces no duplicates and captures all items.
func TestScenePagination_NoDuplicates(t *testing.T) {
	repo := NewInMemorySceneRepository()

	// Create 25 scenes with varying scores to test pagination
	totalScenes := 25
	expectedIDs := make(map[string]bool)

	for i := 0; i < totalScenes; i++ {
		scene := &Scene{
			ID:            uuid.New().String(),
			Name:          fmt.Sprintf("Scene %d", i),
			OwnerDID:      "did:example:owner1",
			Visibility:    VisibilityPublic,
			AllowPrecise:  true,
			PrecisePoint:  &Point{Lat: 40.7128 + float64(i)*0.001, Lng: -74.0060 + float64(i)*0.001},
			CoarseGeohash: "dr5regw",
			Tags:          []string{"music"},
		}
		if err := repo.Insert(scene); err != nil {
			t.Fatalf("failed to insert scene %d: %v", i, err)
		}
		expectedIDs[scene.ID] = true
	}

	// Paginate through all results with limit of 10
	pageSize := 10
	var cursor string
	seenIDs := make(map[string]bool)
	pageCount := 0
	maxPages := 10 // Safety limit to prevent infinite loops

	for {
		pageCount++
		if pageCount > maxPages {
			t.Fatal("pagination exceeded max pages, possible infinite loop")
		}

		results, nextCursor, err := repo.SearchScenes(SceneSearchOptions{
			MinLng: -75.0,
			MinLat: 40.0,
			MaxLng: -73.0,
			MaxLat: 41.0,
			Query:  "music",
			Limit:  pageSize,
			Cursor: cursor,
		})
		if err != nil {
			t.Fatalf("failed to search scenes on page %d: %v", pageCount, err)
		}

		// Check for duplicates within this page
		for _, scene := range results {
			if seenIDs[scene.ID] {
				t.Errorf("duplicate scene ID %s found on page %d", scene.ID, pageCount)
			}
			seenIDs[scene.ID] = true
		}

		t.Logf("Page %d: %d results, cursor=%s", pageCount, len(results), nextCursor)

		// If no more results, we're done
		if nextCursor == "" {
			break
		}

		cursor = nextCursor
	}

	// Verify we got all scenes exactly once
	if len(seenIDs) != totalScenes {
		t.Errorf("expected %d unique scenes, got %d", totalScenes, len(seenIDs))
	}

	// Verify all expected IDs were returned
	for id := range expectedIDs {
		if !seenIDs[id] {
			t.Errorf("expected scene ID %s was not returned in pagination", id)
		}
	}
}

// TestScenePagination_OrderingConsistency tests that pagination returns
// scenes in consistent order across multiple runs.
func TestScenePagination_OrderingConsistency(t *testing.T) {
	repo := NewInMemorySceneRepository()

	// Create scenes with specific scores to test ordering
	scenes := []*Scene{
		{
			ID:            "scene-1",
			Name:          "High Score Scene",
			OwnerDID:      "did:example:owner1",
			Visibility:    VisibilityPublic,
			AllowPrecise:  true,
			PrecisePoint:  &Point{Lat: 40.7128, Lng: -74.0060}, // At center
			CoarseGeohash: "dr5regw",
			Tags:          []string{"music"},
		},
		{
			ID:            "scene-2",
			Name:          "Medium Score Scene",
			OwnerDID:      "did:example:owner1",
			Visibility:    VisibilityPublic,
			AllowPrecise:  true,
			PrecisePoint:  &Point{Lat: 40.72, Lng: -74.01}, // Further from center
			CoarseGeohash: "dr5regw",
			Tags:          []string{"music"},
		},
		{
			ID:            "scene-3",
			Name:          "Low Score Scene",
			OwnerDID:      "did:example:owner1",
			Visibility:    VisibilityPublic,
			AllowPrecise:  true,
			PrecisePoint:  &Point{Lat: 40.75, Lng: -74.05}, // Far from center
			CoarseGeohash: "dr5regw",
			Tags:          []string{"music"},
		},
	}

	for _, scene := range scenes {
		if err := repo.Insert(scene); err != nil {
			t.Fatalf("failed to insert scene %s: %v", scene.ID, err)
		}
	}

	// Run pagination multiple times and verify consistent ordering
	runs := 3
	var previousOrder []string

	for run := 1; run <= runs; run++ {
		var currentOrder []string
		cursor := ""

		for {
			results, nextCursor, err := repo.SearchScenes(SceneSearchOptions{
				MinLng: -75.0,
				MinLat: 40.0,
				MaxLng: -73.0,
				MaxLat: 41.0,
				Query:  "music",
				Limit:  2, // Small page size to test pagination
				Cursor: cursor,
			})
			if err != nil {
				t.Fatalf("run %d: failed to search scenes: %v", run, err)
			}

			for _, scene := range results {
				currentOrder = append(currentOrder, scene.ID)
			}

			if nextCursor == "" {
				break
			}
			cursor = nextCursor
		}

		if run > 1 {
			// Verify order matches previous run
			if len(currentOrder) != len(previousOrder) {
				t.Errorf("run %d: order length mismatch: expected %d, got %d",
					run, len(previousOrder), len(currentOrder))
			}

			for i := 0; i < len(currentOrder) && i < len(previousOrder); i++ {
				if currentOrder[i] != previousOrder[i] {
					t.Errorf("run %d: ordering inconsistency at position %d: expected %s, got %s",
						run, i, previousOrder[i], currentOrder[i])
				}
			}
		}

		previousOrder = currentOrder
		t.Logf("Run %d order: %v", run, currentOrder)
	}
}

// TestScenePagination_ScoreTies tests that scenes with identical composite scores
// are ordered deterministically by ID.
func TestScenePagination_ScoreTies(t *testing.T) {
	repo := NewInMemorySceneRepository()

	// Create scenes with identical characteristics except ID
	// to force score ties
	centerLat := 40.7128
	centerLng := -74.0060

	sceneIDs := []string{"scene-a", "scene-b", "scene-c", "scene-d", "scene-e"}

	for _, id := range sceneIDs {
		scene := &Scene{
			ID:            id,
			Name:          "Identical Score Scene",
			OwnerDID:      "did:example:owner1",
			Visibility:    VisibilityPublic,
			AllowPrecise:  true,
			PrecisePoint:  &Point{Lat: centerLat, Lng: centerLng}, // Same location
			CoarseGeohash: "dr5regw",
			Tags:          []string{"music"},
		}
		if err := repo.Insert(scene); err != nil {
			t.Fatalf("failed to insert scene %s: %v", id, err)
		}
	}

	// Retrieve all scenes in a single page
	results, _, err := repo.SearchScenes(SceneSearchOptions{
		MinLng: -75.0,
		MinLat: 40.0,
		MaxLng: -73.0,
		MaxLat: 41.0,
		Query:  "music",
		Limit:  10,
	})
	if err != nil {
		t.Fatalf("failed to search scenes: %v", err)
	}

	if len(results) != len(sceneIDs) {
		t.Fatalf("expected %d scenes, got %d", len(sceneIDs), len(results))
	}

	// Sort expected IDs for comparison
	sortedExpected := make([]string, len(sceneIDs))
	copy(sortedExpected, sceneIDs)
	sort.Strings(sortedExpected)

	// Verify results are ordered by ID (ascending) when scores are identical
	for i, scene := range results {
		if scene.ID != sortedExpected[i] {
			t.Errorf("position %d: expected ID %s, got %s (tie-breaking by ID failed)",
				i, sortedExpected[i], scene.ID)
		}
	}

	t.Logf("Score tie ordering (by ID): %v", sortedExpected)
}

// TestScenePagination_InsertionOrderIndependence tests that pagination ordering
// is independent of insertion order (shuffle resistance).
func TestScenePagination_InsertionOrderIndependence(t *testing.T) {
	// Create scenes with predictable scores
	scenesData := []struct {
		id       string
		name     string
		distance float64 // Distance from center (affects proximity score)
	}{
		{"scene-1", "Music Venue Alpha", 0.0},    // High proximity
		{"scene-2", "Music Venue Beta", 0.01},    // Medium proximity
		{"scene-3", "Music Venue Gamma", 0.05},   // Low proximity
		{"scene-4", "Music Venue Delta", 0.001},  // Very high proximity
		{"scene-5", "Music Venue Epsilon", 0.02}, // Medium-low proximity
	}

	centerLat := 40.7128
	centerLng := -74.0060

	// Run test with different insertion orders
	runs := 3

	var expectedOrder []string

	for run := 1; run <= runs; run++ {
		repo := NewInMemorySceneRepository()

		// Shuffle insertion order for each run
		shuffled := make([]int, len(scenesData))
		for i := range shuffled {
			shuffled[i] = i
		}
		r := rand.New(rand.NewSource(int64(run * 12345)))
		r.Shuffle(len(shuffled), func(i, j int) {
			shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
		})

		t.Logf("Run %d insertion order: %v", run, shuffled)

		// Insert scenes in shuffled order
		for _, idx := range shuffled {
			data := scenesData[idx]
			scene := &Scene{
				ID:            data.id,
				Name:          data.name,
				OwnerDID:      "did:example:owner1",
				Visibility:    VisibilityPublic,
				AllowPrecise:  true,
				PrecisePoint:  &Point{Lat: centerLat + data.distance, Lng: centerLng + data.distance},
				CoarseGeohash: "dr5regw",
				Tags:          []string{"music"},
			}
			if err := repo.Insert(scene); err != nil {
				t.Fatalf("run %d: failed to insert scene %s: %v", run, data.id, err)
			}
		}

		// Retrieve scenes
		results, _, err := repo.SearchScenes(SceneSearchOptions{
			MinLng: -75.0,
			MinLat: 40.0,
			MaxLng: -73.0,
			MaxLat: 41.0,
			Query:  "music",
			Limit:  10,
		})
		if err != nil {
			t.Fatalf("run %d: failed to search scenes: %v", run, err)
		}

		currentOrder := make([]string, len(results))
		for i, scene := range results {
			currentOrder[i] = scene.ID
		}

		if run == 1 {
			expectedOrder = currentOrder
			t.Logf("Run 1 result order: %v", currentOrder)
		} else {
			// Verify order matches first run despite different insertion order
			if len(currentOrder) != len(expectedOrder) {
				t.Errorf("run %d: order length mismatch: expected %d, got %d",
					run, len(expectedOrder), len(currentOrder))
			}

			for i := 0; i < len(currentOrder) && i < len(expectedOrder); i++ {
				if currentOrder[i] != expectedOrder[i] {
					t.Errorf("run %d: ordering differs at position %d: expected %s, got %s (insertion order affected result)",
						run, i, expectedOrder[i], currentOrder[i])
				}
			}
			t.Logf("Run %d result order: %v (matches expected)", run, currentOrder)
		}
	}
}

// TestSceneCursor_RoundTrip tests that cursor encoding and decoding preserves
// all fields with full precision.
func TestSceneCursor_RoundTrip(t *testing.T) {
	testCases := []struct {
		name  string
		score float64
		id    string
	}{
		{
			name:  "integer score",
			score: 1.0,
			id:    "scene-123",
		},
		{
			name:  "decimal score",
			score: 0.85432,
			id:    "scene-456",
		},
		{
			name:  "very small score",
			score: 0.000001,
			id:    "scene-789",
		},
		{
			name:  "zero score",
			score: 0.0,
			id:    "scene-000",
		},
		{
			name:  "high precision score",
			score: 0.123456789012345,
			id:    "scene-abc",
		},
		{
			name:  "UUID ID",
			score: 0.5,
			id:    uuid.New().String(),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Encode
			encoded := EncodeSceneCursor(tc.score, tc.id)
			if encoded == "" {
				t.Fatal("encoded cursor should not be empty")
			}

			// Decode
			decoded, err := DecodeSceneCursor(encoded)
			if err != nil {
				t.Fatalf("failed to decode cursor: %v", err)
			}

			// Verify score is preserved (within floating point precision)
			if decoded.Score != tc.score {
				t.Errorf("score mismatch: expected %.15f, got %.15f", tc.score, decoded.Score)
			}

			// Verify ID is preserved exactly
			if decoded.ID != tc.id {
				t.Errorf("ID mismatch: expected %s, got %s", tc.id, decoded.ID)
			}
		})
	}
}

// TestScenePagination_EmptyResults tests pagination behavior with no results.
func TestScenePagination_EmptyResults(t *testing.T) {
	repo := NewInMemorySceneRepository()

	results, cursor, err := repo.SearchScenes(SceneSearchOptions{
		MinLng: -75.0,
		MinLat: 40.0,
		MaxLng: -73.0,
		MaxLat: 41.0,
		Query:  "nonexistent",
		Limit:  10,
	})
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}

	if cursor != "" {
		t.Errorf("expected empty cursor for no results, got %s", cursor)
	}
}

// TestScenePagination_LastPageEmptyCursor tests that the last page returns
// an empty next_cursor.
func TestScenePagination_LastPageEmptyCursor(t *testing.T) {
	repo := NewInMemorySceneRepository()

	// Create exactly 5 scenes
	for i := 0; i < 5; i++ {
		scene := &Scene{
			ID:            fmt.Sprintf("scene-%d", i),
			Name:          fmt.Sprintf("Scene %d", i),
			OwnerDID:      "did:example:owner1",
			Visibility:    VisibilityPublic,
			AllowPrecise:  true,
			PrecisePoint:  &Point{Lat: 40.7128, Lng: -74.0060},
			CoarseGeohash: "dr5regw",
			Tags:          []string{"music"},
		}
		if err := repo.Insert(scene); err != nil {
			t.Fatalf("failed to insert scene: %v", err)
		}
	}

	// Request with limit exactly matching total count
	results, cursor, err := repo.SearchScenes(SceneSearchOptions{
		MinLng: -75.0,
		MinLat: 40.0,
		MaxLng: -73.0,
		MaxLat: 41.0,
		Query:  "music",
		Limit:  5,
	})
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}

	if len(results) != 5 {
		t.Errorf("expected 5 results, got %d", len(results))
	}

	// Since we got exactly limit items, we need to check if cursor is empty
	// The last page should have empty cursor
	if cursor != "" {
		t.Errorf("expected empty cursor for last page, got %s", cursor)
	}
}

// TestScenePagination_SingleItemPage tests pagination with a single item.
func TestScenePagination_SingleItemPage(t *testing.T) {
	repo := NewInMemorySceneRepository()

	scene := &Scene{
		ID:            "scene-single",
		Name:          "Single Scene",
		OwnerDID:      "did:example:owner1",
		Visibility:    VisibilityPublic,
		AllowPrecise:  true,
		PrecisePoint:  &Point{Lat: 40.7128, Lng: -74.0060},
		CoarseGeohash: "dr5regw",
		Tags:          []string{"music"},
	}
	if err := repo.Insert(scene); err != nil {
		t.Fatalf("failed to insert scene: %v", err)
	}

	results, cursor, err := repo.SearchScenes(SceneSearchOptions{
		MinLng: -75.0,
		MinLat: 40.0,
		MaxLng: -73.0,
		MaxLat: 41.0,
		Query:  "music",
		Limit:  10,
	})
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}

	if cursor != "" {
		t.Errorf("expected empty cursor for single item, got %s", cursor)
	}

	if results[0].ID != scene.ID {
		t.Errorf("expected scene ID %s, got %s", scene.ID, results[0].ID)
	}
}

// TestEventPagination_NoDuplicates tests event pagination for duplicates.
func TestEventPagination_NoDuplicates(t *testing.T) {
	repo := NewInMemoryEventRepository()

	baseTime := time.Now().Add(24 * time.Hour)
	totalEvents := 25
	expectedIDs := make(map[string]bool)

	for i := 0; i < totalEvents; i++ {
		event := &Event{
			ID:            uuid.New().String(),
			SceneID:       "scene-123",
			Title:         fmt.Sprintf("Event %d", i),
			AllowPrecise:  true,
			PrecisePoint:  &Point{Lat: 40.7128, Lng: -74.0060},
			CoarseGeohash: "dr5regw",
			Status:        "scheduled",
			StartsAt:      baseTime.Add(time.Duration(i) * time.Hour),
			CreatedAt:     &baseTime,
			UpdatedAt:     &baseTime,
		}
		if err := repo.Insert(event); err != nil {
			t.Fatalf("failed to insert event %d: %v", i, err)
		}
		expectedIDs[event.ID] = true
	}

	// Paginate through all results
	pageSize := 10
	var cursor string
	seenIDs := make(map[string]bool)
	pageCount := 0
	maxPages := 10

	from := baseTime.Add(-1 * time.Hour)
	to := baseTime.Add(30 * time.Hour)

	for {
		pageCount++
		if pageCount > maxPages {
			t.Fatal("pagination exceeded max pages")
		}

		results, nextCursor, err := repo.SearchEvents(EventSearchOptions{
			MinLng: -75.0,
			MinLat: 40.0,
			MaxLng: -73.0,
			MaxLat: 41.0,
			From:   from,
			To:     to,
			Limit:  pageSize,
			Cursor: cursor,
		})
		if err != nil {
			t.Fatalf("search failed on page %d: %v", pageCount, err)
		}

		for _, event := range results {
			if seenIDs[event.ID] {
				t.Errorf("duplicate event ID %s on page %d", event.ID, pageCount)
			}
			seenIDs[event.ID] = true
		}

		if nextCursor == "" {
			break
		}
		cursor = nextCursor
	}

	if len(seenIDs) != totalEvents {
		t.Errorf("expected %d unique events, got %d", totalEvents, len(seenIDs))
	}
}

// TestEventPagination_ScoreTies tests event tie-breaking by ID.
func TestEventPagination_ScoreTies(t *testing.T) {
	repo := NewInMemoryEventRepository()

	baseTime := time.Now().Add(24 * time.Hour)
	eventIDs := []string{"event-a", "event-b", "event-c", "event-d"}

	// Create events with identical scores
	for _, id := range eventIDs {
		event := &Event{
			ID:            id,
			SceneID:       "scene-123",
			Title:         "Identical Event",
			AllowPrecise:  true,
			PrecisePoint:  &Point{Lat: 40.7128, Lng: -74.0060},
			CoarseGeohash: "dr5regw",
			Status:        "scheduled",
			StartsAt:      baseTime.Add(1 * time.Hour), // Same time
			CreatedAt:     &baseTime,
			UpdatedAt:     &baseTime,
		}
		if err := repo.Insert(event); err != nil {
			t.Fatalf("failed to insert event %s: %v", id, err)
		}
	}

	from := baseTime.Add(-1 * time.Hour)
	to := baseTime.Add(3 * time.Hour)

	results, _, err := repo.SearchEvents(EventSearchOptions{
		MinLng: -75.0,
		MinLat: 40.0,
		MaxLng: -73.0,
		MaxLat: 41.0,
		From:   from,
		To:     to,
		Limit:  10,
	})
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}

	if len(results) != len(eventIDs) {
		t.Fatalf("expected %d events, got %d", len(eventIDs), len(results))
	}

	// Verify ordering by ID (ascending)
	sortedExpected := make([]string, len(eventIDs))
	copy(sortedExpected, eventIDs)
	sort.Strings(sortedExpected)

	for i, event := range results {
		if event.ID != sortedExpected[i] {
			t.Errorf("position %d: expected %s, got %s", i, sortedExpected[i], event.ID)
		}
	}
}

// TestCursorDecodeComplexity verifies cursor decode is O(1) by timing it.
// This is a performance test, not a strict unit test.
func TestCursorDecodeComplexity(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping performance test in short mode")
	}

	// Generate a cursor
	cursor := EncodeSceneCursor(0.12345, "scene-test-id")

	// Decode multiple times and measure
	iterations := 10000
	start := time.Now()

	for i := 0; i < iterations; i++ {
		_, err := DecodeSceneCursor(cursor)
		if err != nil {
			t.Fatalf("decode failed: %v", err)
		}
	}

	elapsed := time.Since(start)
	avgTime := elapsed / time.Duration(iterations)

	t.Logf("Decoded %d cursors in %v (avg: %v per decode)", iterations, elapsed, avgTime)

	// Verify average time is reasonable (< 1ms per decode is well within O(1))
	if avgTime > time.Millisecond {
		t.Errorf("cursor decode too slow: %v per operation (expected < 1ms)", avgTime)
	}
}
