// Package scene provides benchmarks for critical scene operations.
package scene

import (
	"fmt"
	"testing"
	"time"
)

// BenchmarkSceneRepository_Insert measures the performance of scene creation
func BenchmarkSceneRepository_Insert(b *testing.B) {
	repo := NewInMemorySceneRepository()

	b.Run("single_scene_insertion", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			now := time.Now()
			scene := &Scene{
				ID:            generateTestSceneID(i),
				Name:          "Test Scene",
				Description:   "A test scene for benchmarking",
				OwnerDID:      "did:plc:test123",
				CreatedAt:     &now,
				UpdatedAt:     &now,
				AllowPrecise:  false,
				CoarseGeohash: "dr5r",
				PrecisePoint:  &Point{Lat: 40.7128, Lng: -74.0060},
			}
			_ = repo.Insert(scene)
		}
	})

	b.Run("concurrent_scene_insertion", func(b *testing.B) {
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			var counter int64
			for pb.Next() {
				now := time.Now()
				// Use unique ID based on timestamp and a sequential counter to avoid collisions
				uniqueID := generateTestSceneID(int(time.Now().UnixNano()%1000000) + int(counter))
				scene := &Scene{
					ID:            uniqueID,
					Name:          "Test Scene",
					Description:   "A test scene for benchmarking",
					OwnerDID:      "did:plc:test123",
					CreatedAt:     &now,
					UpdatedAt:     &now,
					AllowPrecise:  false,
					CoarseGeohash: "dr5r",
					PrecisePoint:  &Point{Lat: 40.7128, Lng: -74.0060},
				}
				_ = repo.Insert(scene)
				counter++
			}
		})
	})
}

// BenchmarkSceneRepository_GetByID measures the performance of scene retrieval
func BenchmarkSceneRepository_GetByID(b *testing.B) {
	repo := NewInMemorySceneRepository()

	// Pre-populate with test data
	for i := 0; i < 1000; i++ {
		now := time.Now()
		scene := &Scene{
			ID:            generateTestSceneID(i),
			Name:          "Test Scene",
			Description:   "A test scene for benchmarking",
			OwnerDID:      "did:plc:test123",
			CreatedAt:     &now,
			UpdatedAt:     &now,
			AllowPrecise:  false,
			CoarseGeohash: "dr5r",
			PrecisePoint:  &Point{Lat: 40.7128, Lng: -74.0060},
		}
		_ = repo.Insert(scene)
	}

	b.Run("single_lookup", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			id := generateTestSceneID(i % 1000)
			_, _ = repo.GetByID(id)
		}
	})

	b.Run("concurrent_lookup", func(b *testing.B) {
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			var counter int64
			for pb.Next() {
				// Use modulo to cycle through existing test scenes
				id := generateTestSceneID(int(counter % 1000))
				_, _ = repo.GetByID(id)
				counter++
			}
		})
	})
}

// BenchmarkSceneRepository_ListByOwner measures the performance of owner-based queries
func BenchmarkSceneRepository_ListByOwner(b *testing.B) {
	repo := NewInMemorySceneRepository()

	// Pre-populate with test data for multiple owners
	for i := 0; i < 1000; i++ {
		now := time.Now()
		ownerDID := generateTestOwnerDID(i % 100) // 100 different owners, 10 scenes each
		scene := &Scene{
			ID:            generateTestSceneID(i),
			Name:          "Test Scene",
			Description:   "A test scene for benchmarking",
			OwnerDID:      ownerDID,
			CreatedAt:     &now,
			UpdatedAt:     &now,
			AllowPrecise:  false,
			CoarseGeohash: "dr5r",
			PrecisePoint:  &Point{Lat: 40.7128, Lng: -74.0060},
		}
		_ = repo.Insert(scene)
	}

	b.Run("list_by_owner", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			ownerDID := generateTestOwnerDID(i % 100)
			_, _ = repo.ListByOwner(ownerDID)
		}
	})
}

// BenchmarkScene_EnforceLocationConsent measures the performance of privacy enforcement
func BenchmarkScene_EnforceLocationConsent(b *testing.B) {
	b.Run("with_precise_location", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			scene := &Scene{
				AllowPrecise: true,
				PrecisePoint: &Point{Lat: 40.7128, Lng: -74.0060},
			}
			scene.EnforceLocationConsent()
		}
	})

	b.Run("without_precise_location", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			scene := &Scene{
				AllowPrecise: false,
				PrecisePoint: &Point{Lat: 40.7128, Lng: -74.0060},
			}
			scene.EnforceLocationConsent()
		}
	})
}

// BenchmarkSearchScenes_10k measures SearchScenes performance with 10,000 scenes,
// providing an average/steady-state latency baseline for a first-page query against
// the p95 < 300ms latency budget. Real-world PostgreSQL performance relies on the
// indexes added in migration 000030 (idx_scenes_search_visible, idx_scenes_public_created).
func BenchmarkSearchScenes_10k(b *testing.B) {
	const datasetSize = 10_000

	repo := NewInMemorySceneRepository()
	now := time.Now()

	// Spread scenes across a geographic grid within NYC bbox to exercise bbox filtering.
	// Grid: 100 lat cells × 100 lng cells = 10,000 scenes.
	const (
		minLat = 40.60
		maxLat = 40.80
		minLng = -74.10
		maxLng = -73.90
	)
	latStep := (maxLat - minLat) / 100.0
	lngStep := (maxLng - minLng) / 100.0

	// Mix of public, members-only, and unlisted to match realistic distributions:
	// 70% public, 20% members-only (private), 10% unlisted (hidden).
	visibilities := []string{
		VisibilityPublic, VisibilityPublic, VisibilityPublic, VisibilityPublic,
		VisibilityPublic, VisibilityPublic, VisibilityPublic, VisibilityMembersOnly,
		VisibilityMembersOnly, VisibilityHidden,
	}

	// Vary names and tags to exercise text matching.
	genres := []string{"techno", "jazz", "rock", "ambient", "hip-hop", "folk", "metal", "soul", "punk", "reggae"}

	for i := 0; i < datasetSize; i++ {
		row := i / 100
		col := i % 100
		lat := minLat + float64(row)*latStep
		lng := minLng + float64(col)*lngStep
		genre := genres[i%len(genres)]
		vis := visibilities[i%len(visibilities)]

		s := &Scene{
			ID:            generateTestSceneID(i),
			Name:          fmt.Sprintf("%s Scene %d", genre, i),
			Description:   fmt.Sprintf("Underground %s collective", genre),
			OwnerDID:      generateTestOwnerDID(i % 500),
			AllowPrecise:  true,
			PrecisePoint:  &Point{Lat: lat, Lng: lng},
			CoarseGeohash: "dr5r",
			Tags:          []string{genre, "live-music"},
			Visibility:    vis,
			CreatedAt:     &now,
			UpdatedAt:     &now,
		}
		if err := repo.Insert(s); err != nil {
			b.Fatalf("failed to insert scene %d: %v", i, err)
		}
	}

	// Bbox covers roughly the bottom-left quarter of the grid (~2,500 candidates).
	searchBbox := SceneSearchOptions{
		MinLng: minLng,
		MinLat: minLat,
		MaxLng: minLng + (maxLng-minLng)/2,
		MaxLat: minLat + (maxLat-minLat)/2,
		Limit:  20,
	}

	b.Run("first_page_no_query", func(b *testing.B) {
		opts := searchBbox
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _, _ = repo.SearchScenes(opts)
		}
	})

	b.Run("first_page_text_query", func(b *testing.B) {
		opts := searchBbox
		opts.Query = "techno"
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _, _ = repo.SearchScenes(opts)
		}
	})

	b.Run("second_page_text_query", func(b *testing.B) {
		// Obtain a real cursor from the first page to test second-page latency.
		opts := searchBbox
		opts.Query = "techno"
		_, cursor, err := repo.SearchScenes(opts)
		if err != nil || cursor == "" {
			b.Skip("no second page available")
		}
		opts.Cursor = cursor
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _, _ = repo.SearchScenes(opts)
		}
	})

	b.Run("full_bbox_no_query", func(b *testing.B) {
		opts := SceneSearchOptions{
			MinLng: minLng,
			MinLat: minLat,
			MaxLng: maxLng,
			MaxLat: maxLat,
			Limit:  20,
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _, _ = repo.SearchScenes(opts)
		}
	})
}

// BenchmarkSearchEvents_10k measures SearchEvents performance with 10,000 events,
// approximating a first-page query under the p95 < 300ms latency budget.
// This benchmark reports average time/op; real-world PostgreSQL performance
// relies on idx_events_upcoming_geohash and idx_events_scene_upcoming added
// in migration 000030.
func BenchmarkSearchEvents_10k(b *testing.B) {
	const datasetSize = 10_000

	repo := NewInMemoryEventRepository()
	baseTime := time.Now().Add(24 * time.Hour)

	genres := []string{"techno", "jazz", "rock", "ambient", "hip-hop", "folk", "metal", "soul", "punk", "reggae"}

	// Spread events over 30 days, across NYC bbox.
	const (
		minLat = 40.60
		maxLat = 40.80
		minLng = -74.10
		maxLng = -73.90
	)
	latStep := (maxLat - minLat) / 100.0
	lngStep := (maxLng - minLng) / 100.0

	for i := 0; i < datasetSize; i++ {
		row := i / 100
		col := i % 100
		lat := minLat + float64(row)*latStep
		lng := minLng + float64(col)*lngStep
		genre := genres[i%len(genres)]
		now := baseTime

		e := &Event{
			ID:            generateTestEventID(i),
			SceneID:       generateTestSceneID(i % 500),
			Title:         fmt.Sprintf("%s Night %d", genre, i),
			Description:   fmt.Sprintf("Underground %s event", genre),
			AllowPrecise:  true,
			PrecisePoint:  &Point{Lat: lat, Lng: lng},
			CoarseGeohash: "dr5r",
			Tags:          []string{genre},
			Status:        "scheduled",
			StartsAt:      baseTime.Add(time.Duration(i%720) * time.Hour), // spread over 30 days
			CreatedAt:     &now,
			UpdatedAt:     &now,
		}
		if err := repo.Insert(e); err != nil {
			b.Fatalf("failed to insert event %d: %v", i, err)
		}
	}

	from := baseTime
	to := baseTime.Add(7 * 24 * time.Hour) // 7-day window

	searchOpts := EventSearchOptions{
		MinLng: minLng,
		MinLat: minLat,
		MaxLng: minLng + (maxLng-minLng)/2,
		MaxLat: minLat + (maxLat-minLat)/2,
		From:   from,
		To:     to,
		Limit:  20,
	}

	b.Run("first_page_no_query", func(b *testing.B) {
		opts := searchOpts
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _, _ = repo.SearchEvents(opts)
		}
	})

	b.Run("first_page_text_query", func(b *testing.B) {
		opts := searchOpts
		opts.Query = "techno"
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _, _ = repo.SearchEvents(opts)
		}
	})

	b.Run("second_page_text_query", func(b *testing.B) {
		opts := searchOpts
		opts.Query = "techno"
		_, cursor, err := repo.SearchEvents(opts)
		if err != nil || cursor == "" {
			b.Skip("no second page available")
		}
		opts.Cursor = cursor
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _, _ = repo.SearchEvents(opts)
		}
	})
}

// Helper functions for generating test data

func generateTestEventID(i int) string {
	return generateID("event", i)
}

func generateTestSceneID(i int) string {
	return generateID("scene", i)
}

func generateTestOwnerDID(i int) string {
	return generateID("did:plc:owner", i)
}

func generateID(prefix string, i int) string {
	return prefix + string(rune(i%26+'a')) + string(rune((i/26)%26+'a')) + string(rune((i/676)%26+'a'))
}
