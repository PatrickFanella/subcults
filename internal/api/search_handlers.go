// Package api provides HTTP handlers for the Subcults API.
package api

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/onnwee/subcults/internal/middleware"
	"github.com/onnwee/subcults/internal/post"
	"github.com/onnwee/subcults/internal/scene"
	"github.com/onnwee/subcults/internal/trust"
)

// SearchHandlers holds dependencies for search HTTP handlers.
type SearchHandlers struct {
	sceneRepo  scene.SceneRepository
	eventRepo  scene.EventRepository
	postRepo   post.PostRepository
	trustStore TrustScoreStore
}

// NewSearchHandlers creates a new SearchHandlers instance.
func NewSearchHandlers(sceneRepo scene.SceneRepository, postRepo post.PostRepository, trustStore TrustScoreStore, eventRepo scene.EventRepository) *SearchHandlers {
	return &SearchHandlers{
		sceneRepo:  sceneRepo,
		eventRepo:  eventRepo,
		postRepo:   postRepo,
		trustStore: trustStore,
	}
}

// SceneSearchResponse represents the response for scene search.
type SceneSearchResponse struct {
	Results    []*SceneSearchResult `json:"results"`
	NextCursor string               `json:"next_cursor,omitempty"`
	Count      int                  `json:"count"`
}

// SceneSearchResult represents a minimal scene result for search.
type SceneSearchResult struct {
	ID            string       `json:"id"`
	Name          string       `json:"name"`
	Description   string       `json:"description,omitempty"`
	JitteredPoint *scene.Point `json:"jittered_centroid,omitempty"` // Always jittered for privacy
	CoarseGeohash string       `json:"coarse_geohash"`
	Tags          []string     `json:"tags,omitempty"`
	Visibility    string       `json:"visibility"`
	TrustScore    *float64     `json:"trust_score,omitempty"` // Only if trust ranking enabled
}

// Constants for bbox validation
const (
	MaxBboxAreaDegrees                     = 10.0 // Max bbox area in square degrees (~1000km x 1000km at equator)
	MaxSearchLimit                         = 50   // Max results per page
	DefaultSearchLimit                     = 20   // Default results if not specified
	MaxGlobalLimit                         = 25
	maxGlobalScenes                        = 10
	maxGlobalEvents                        = 10
	maxGlobalPosts                         = 5
	defaultEventPastYearsForGlobalSearch   = 1
	defaultEventFutureYearsForGlobalSearch = 5
	defaultGlobalEventSearchRadiusDegrees  = 5.0
)

// SearchScenes handles GET /search/scenes - searches for scenes with ranking and pagination.
func (h *SearchHandlers) SearchScenes(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	query := r.URL.Query()

	// Get text search query (optional)
	q := strings.TrimSpace(query.Get("q"))

	// Optional reference point for proximity scoring
	var lat, lng *float64
	if latStr := strings.TrimSpace(query.Get("lat")); latStr != "" {
		parsedLat, err := strconv.ParseFloat(latStr, 64)
		if err != nil || parsedLat < -90 || parsedLat > 90 {
			ctx := middleware.SetErrorCode(r.Context(), ErrCodeValidation)
			WriteError(w, ctx, http.StatusBadRequest, ErrCodeValidation, "lat must be a valid latitude between -90 and 90")
			return
		}
		lat = &parsedLat
	}
	if lngStr := strings.TrimSpace(query.Get("lon")); lngStr != "" {
		parsedLng, err := strconv.ParseFloat(lngStr, 64)
		if err != nil || parsedLng < -180 || parsedLng > 180 {
			ctx := middleware.SetErrorCode(r.Context(), ErrCodeValidation)
			WriteError(w, ctx, http.StatusBadRequest, ErrCodeValidation, "lon must be a valid longitude between -180 and 180")
			return
		}
		lng = &parsedLng
	}
	if (lat == nil) != (lng == nil) {
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeValidation)
		WriteError(w, ctx, http.StatusBadRequest, ErrCodeValidation, "lat and lon must be provided together")
		return
	}

	// Get bbox parameters (required unless lat/lon is provided)
	bboxStr := query.Get("bbox")
	if bboxStr == "" && (lat == nil || lng == nil) {
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeValidation)
		WriteError(w, ctx, http.StatusBadRequest, ErrCodeValidation, "bbox parameter is required (or provide lat/lon)")
		return
	}

	// Parse bbox
	var minLng, minLat, maxLng, maxLat float64
	{
		if bboxStr != "" {
			parts := strings.Split(bboxStr, ",")
			if len(parts) != 4 {
				ctx := middleware.SetErrorCode(r.Context(), ErrCodeValidation)
				WriteError(w, ctx, http.StatusBadRequest, ErrCodeValidation, "bbox must be in format: minLng,minLat,maxLng,maxLat")
				return
			}

			var err error
			minLng, err = strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
			if err != nil {
				ctx := middleware.SetErrorCode(r.Context(), ErrCodeValidation)
				WriteError(w, ctx, http.StatusBadRequest, ErrCodeValidation, "Invalid minLng in bbox")
				return
			}

			minLat, err = strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
			if err != nil {
				ctx := middleware.SetErrorCode(r.Context(), ErrCodeValidation)
				WriteError(w, ctx, http.StatusBadRequest, ErrCodeValidation, "Invalid minLat in bbox")
				return
			}

			maxLng, err = strconv.ParseFloat(strings.TrimSpace(parts[2]), 64)
			if err != nil {
				ctx := middleware.SetErrorCode(r.Context(), ErrCodeValidation)
				WriteError(w, ctx, http.StatusBadRequest, ErrCodeValidation, "Invalid maxLng in bbox")
				return
			}

			maxLat, err = strconv.ParseFloat(strings.TrimSpace(parts[3]), 64)
			if err != nil {
				ctx := middleware.SetErrorCode(r.Context(), ErrCodeValidation)
				WriteError(w, ctx, http.StatusBadRequest, ErrCodeValidation, "Invalid maxLat in bbox")
				return
			}

			// Validate bbox coordinates
			if minLng < -180 || minLng > 180 || maxLng < -180 || maxLng > 180 {
				ctx := middleware.SetErrorCode(r.Context(), ErrCodeValidation)
				WriteError(w, ctx, http.StatusBadRequest, ErrCodeValidation, "Longitude must be between -180 and 180")
				return
			}

			if minLat < -90 || minLat > 90 || maxLat < -90 || maxLat > 90 {
				ctx := middleware.SetErrorCode(r.Context(), ErrCodeValidation)
				WriteError(w, ctx, http.StatusBadRequest, ErrCodeValidation, "Latitude must be between -90 and 90")
				return
			}

			if minLng >= maxLng {
				ctx := middleware.SetErrorCode(r.Context(), ErrCodeValidation)
				WriteError(w, ctx, http.StatusBadRequest, ErrCodeValidation, "minLng must be less than maxLng")
				return
			}

			if minLat >= maxLat {
				ctx := middleware.SetErrorCode(r.Context(), ErrCodeValidation)
				WriteError(w, ctx, http.StatusBadRequest, ErrCodeValidation, "minLat must be less than maxLat")
				return
			}

			// Validate bbox area (prevent wide scans)
			area := (maxLng - minLng) * (maxLat - minLat)
			if area > MaxBboxAreaDegrees {
				ctx := middleware.SetErrorCode(r.Context(), ErrCodeValidation)
				WriteError(w, ctx, http.StatusBadRequest, ErrCodeValidation, fmt.Sprintf("bbox area too large (max %.1f square degrees)", MaxBboxAreaDegrees))
				return
			}
		}
	}

	// Parse genre filter (comma-separated tags)
	var genres []string
	if genresStr := strings.TrimSpace(query.Get("genres")); genresStr != "" {
		for _, genre := range strings.Split(genresStr, ",") {
			if g := strings.TrimSpace(genre); g != "" {
				genres = append(genres, g)
			}
		}
	}

	// Get pagination parameters
	cursor := query.Get("cursor")

	limit := DefaultSearchLimit
	if limitStr := query.Get("limit"); limitStr != "" {
		var err error
		limit, err = strconv.Atoi(limitStr)
		if err != nil || limit < 1 {
			ctx := middleware.SetErrorCode(r.Context(), ErrCodeValidation)
			WriteError(w, ctx, http.StatusBadRequest, ErrCodeValidation, "limit must be a positive integer")
			return
		}
		if limit > MaxSearchLimit {
			limit = MaxSearchLimit
		}
	}
	offset := 0
	if offsetStr := query.Get("offset"); offsetStr != "" {
		var err error
		offset, err = strconv.Atoi(offsetStr)
		if err != nil || offset < 0 {
			ctx := middleware.SetErrorCode(r.Context(), ErrCodeValidation)
			WriteError(w, ctx, http.StatusBadRequest, ErrCodeValidation, "offset must be a non-negative integer")
			return
		}
	}

	// Execute search
	searchOpts := scene.SceneSearchOptions{
		MinLng: minLng,
		MinLat: minLat,
		MaxLng: maxLng,
		MaxLat: maxLat,
		Lat:    lat,
		Lng:    lng,
		Query:  q,
		Genres: genres,
		Limit:  limit,
		Offset: offset,
		Cursor: cursor,
	}

	trustEnabled := trust.IsRankingEnabled() && h.trustStore != nil
	if trustEnabled {
		searchOpts.TrustScores = make(map[string]float64)
	}

	results, nextCursor, err := h.sceneRepo.SearchScenes(searchOpts)

	if err != nil {
		slog.ErrorContext(r.Context(), "failed to search scenes", "error", err, "query", q, "bbox", bboxStr)
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeInternal)
		WriteError(w, ctx, http.StatusInternalServerError, ErrCodeInternal, "Failed to search scenes")
		return
	}

	if trustEnabled && len(results) > 0 {
		for _, s := range results {
			score, scoreErr := h.trustStore.GetScore(s.ID)
			if scoreErr != nil {
				slog.WarnContext(r.Context(), "failed to get trust score", "scene_id", s.ID, "error", scoreErr)
				continue
			}
			if score != nil {
				searchOpts.TrustScores[s.ID] = score.Score
			}
		}

		if len(searchOpts.TrustScores) > 0 {
			results, nextCursor, err = h.sceneRepo.SearchScenes(searchOpts)
			if err != nil {
				slog.ErrorContext(r.Context(), "failed to search scenes with trust scores", "error", err, "query", q, "bbox", bboxStr)
				ctx := middleware.SetErrorCode(r.Context(), ErrCodeInternal)
				WriteError(w, ctx, http.StatusInternalServerError, ErrCodeInternal, "Failed to search scenes")
				return
			}
		}
	}

	// Convert to search results with jittered coordinates
	searchResults := make([]*SceneSearchResult, 0, len(results))
	for _, s := range results {
		result := &SceneSearchResult{
			ID:            s.ID,
			Name:          s.Name,
			Description:   s.Description,
			CoarseGeohash: s.CoarseGeohash,
			Tags:          s.Tags,
			Visibility:    s.Visibility,
		}

		// Apply jitter to coordinates for privacy
		// Even if allow_precise is true, we jitter for public search results
		if s.PrecisePoint != nil {
			result.JitteredPoint = applyJitter(s.PrecisePoint)
		}
		if trustEnabled {
			if trustScore, ok := searchOpts.TrustScores[s.ID]; ok {
				scoreCopy := trustScore
				result.TrustScore = &scoreCopy
			}
		}

		searchResults = append(searchResults, result)
	}

	// Build response
	response := SceneSearchResponse{
		Results:    searchResults,
		NextCursor: nextCursor,
		Count:      len(searchResults),
	}

	// Return results
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		slog.ErrorContext(r.Context(), "failed to encode search response", "error", err)
		return
	}
}

// applyJitter applies deterministic geohash-based jitter to a point for privacy.
// Uses the point's geohash to generate a stable offset that prevents exact location
// exposure while maintaining determinism (same coordinates = same jittered result).
func applyJitter(point *scene.Point) *scene.Point {
	// Calculate geohash at precision 8 (approximately 20m x 20m cell)
	// This provides stable jitter based on the location's geohash cell
	geohash := encodeGeohash(point.Lat, point.Lng, 8)

	// Use geohash to generate deterministic offset
	// Hash the geohash string to get a numeric seed
	var hash int64
	for i, c := range geohash {
		hash = hash*31 + int64(c) + int64(i)
	}

	// Ensure positive seed
	if hash < 0 {
		hash = -hash
	}

	// Generate offset in range [0.005, 0.015] degrees (~500m to 1.5km at equator)
	// This is sufficient to hide exact location while keeping scenes discoverable
	latOffset := 0.005 + float64(hash%1000)/100000.0
	lngOffset := 0.005 + float64((hash/1000)%1000)/100000.0

	// Apply offset in a deterministic direction based on hash
	latSign := 1.0
	if (hash/1000000)%2 == 0 {
		latSign = -1.0
	}
	lngSign := 1.0
	if (hash/2000000)%2 == 0 {
		lngSign = -1.0
	}

	return &scene.Point{
		Lat: point.Lat + latOffset*latSign,
		Lng: point.Lng + lngOffset*lngSign,
	}
}

// encodeGeohash encodes a lat/lng coordinate to a geohash string.
// This is a simplified implementation for jitter calculation.
// For production, use a proper geohash library.
func encodeGeohash(lat, lng float64, precision int) string {
	const base32 = "0123456789bcdefghjkmnpqrstuvwxyz"

	minLat, maxLat := -90.0, 90.0
	minLng, maxLng := -180.0, 180.0

	var geohash strings.Builder
	bits := 0
	bit := 0
	ch := 0

	for geohash.Len() < precision {
		if bits%2 == 0 {
			// longitude
			mid := (minLng + maxLng) / 2
			if lng > mid {
				ch |= (1 << (4 - bit))
				minLng = mid
			} else {
				maxLng = mid
			}
		} else {
			// latitude
			mid := (minLat + maxLat) / 2
			if lat > mid {
				ch |= (1 << (4 - bit))
				minLat = mid
			} else {
				maxLat = mid
			}
		}

		bits++
		bit++

		if bit == 5 {
			geohash.WriteByte(base32[ch])
			bit = 0
			ch = 0
		}
	}

	return geohash.String()
}

// PostSearchResponse represents the response for post search.
type PostSearchResponse struct {
	Results    []*PostSearchResult `json:"results"`
	NextCursor string              `json:"next_cursor,omitempty"`
	Count      int                 `json:"count"`
}

// PostSearchResult represents a minimal post result for search.
type PostSearchResult struct {
	ID         string   `json:"id"`
	Excerpt    string   `json:"excerpt"` // First 160 chars
	SceneID    *string  `json:"scene_id,omitempty"`
	TrustScore *float64 `json:"trust_score,omitempty"` // Only if trust ranking enabled
	CreatedAt  string   `json:"created_at"`            // ISO 8601 format
}

// GlobalSearchResult represents one item in mixed global search results.
type GlobalSearchResult struct {
	Type  string                   `json:"type"`
	Scene *SceneSearchResult       `json:"scene,omitempty"`
	Event *GlobalEventSearchResult `json:"event,omitempty"`
	Post  *PostSearchResult        `json:"post,omitempty"`
}

// GlobalEventSearchResult represents a minimal event result for global search.
type GlobalEventSearchResult struct {
	ID        string `json:"id"`
	SceneID   string `json:"scene_id"`
	Title     string `json:"title"`
	StartsAt  string `json:"starts_at"`
	CreatedAt string `json:"created_at"`
}

// GlobalSearchResponse represents the response for global search.
type GlobalSearchResponse struct {
	Results    []*GlobalSearchResult `json:"results"`
	NextCursor string                `json:"next_cursor,omitempty"`
	Count      int                   `json:"count"`
}

type globalSearchCursor struct {
	SceneCursor string `json:"scene_cursor,omitempty"`
	EventCursor string `json:"event_cursor,omitempty"`
	PostCursor  string `json:"post_cursor,omitempty"`
}

// SearchGlobal handles GET /search/global - unified search across scenes, events, and posts.
func (h *SearchHandlers) SearchGlobal(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	q := strings.TrimSpace(query.Get("q"))
	if q == "" {
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeValidation)
		WriteError(w, ctx, http.StatusBadRequest, ErrCodeValidation, "q parameter is required")
		return
	}

	cursorState, err := decodeGlobalSearchCursor(query.Get("cursor"))
	if err != nil {
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeValidation)
		WriteError(w, ctx, http.StatusBadRequest, ErrCodeValidation, "invalid cursor")
		return
	}

	if query.Get("limit") != "" {
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeValidation)
		WriteError(w, ctx, http.StatusBadRequest, ErrCodeValidation, "limit is not supported for global search")
		return
	}

	var lat, lng *float64
	if latStr := strings.TrimSpace(query.Get("lat")); latStr != "" {
		parsedLat, parseErr := strconv.ParseFloat(latStr, 64)
		if parseErr != nil || parsedLat < -90 || parsedLat > 90 {
			ctx := middleware.SetErrorCode(r.Context(), ErrCodeValidation)
			WriteError(w, ctx, http.StatusBadRequest, ErrCodeValidation, "lat must be a valid latitude between -90 and 90")
			return
		}
		lat = &parsedLat
	}
	if lngStr := strings.TrimSpace(query.Get("lon")); lngStr != "" {
		parsedLng, parseErr := strconv.ParseFloat(lngStr, 64)
		if parseErr != nil || parsedLng < -180 || parsedLng > 180 {
			ctx := middleware.SetErrorCode(r.Context(), ErrCodeValidation)
			WriteError(w, ctx, http.StatusBadRequest, ErrCodeValidation, "lon must be a valid longitude between -180 and 180")
			return
		}
		lng = &parsedLng
	}
	if (lat == nil) != (lng == nil) {
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeValidation)
		WriteError(w, ctx, http.StatusBadRequest, ErrCodeValidation, "lat and lon must be provided together")
		return
	}

	sceneResults := make([]*scene.Scene, 0)
	sceneNextCursor := ""
	sceneResults, sceneNextCursor, err = h.sceneRepo.SearchScenes(scene.SceneSearchOptions{
		Lat:              lat,
		Lng:              lng,
		Query:            q,
		Limit:            maxGlobalScenes,
		Cursor:           cursorState.SceneCursor,
		DisableProximity: lat == nil && lng == nil,
	})
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to search scenes for global search", "error", err)
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeInternal)
		WriteError(w, ctx, http.StatusInternalServerError, ErrCodeInternal, "Failed to search")
		return
	}

	eventResults := make([]*scene.Event, 0)
	eventNextCursor := ""
	searchNow := time.Now()
	from := searchNow.AddDate(-defaultEventPastYearsForGlobalSearch, 0, 0)
	to := searchNow.AddDate(defaultEventFutureYearsForGlobalSearch, 0, 0)
	if lat != nil && lng != nil {
		// Uses degree offsets for a lightweight approximate radius window.
		// At higher latitudes longitudinal distance per degree shrinks, so this
		// is an intentionally coarse filter for global text search.
		minLng := *lng - defaultGlobalEventSearchRadiusDegrees
		maxLng := *lng + defaultGlobalEventSearchRadiusDegrees
		minLat := *lat - defaultGlobalEventSearchRadiusDegrees
		maxLat := *lat + defaultGlobalEventSearchRadiusDegrees
		if minLng < -180 {
			minLng = -180
		}
		if maxLng > 180 {
			maxLng = 180
		}
		if minLat < -90 {
			minLat = -90
		}
		if maxLat > 90 {
			maxLat = 90
		}
		eventResults, eventNextCursor, err = h.eventRepo.SearchEvents(scene.EventSearchOptions{
			MinLng:           minLng,
			MinLat:           minLat,
			MaxLng:           maxLng,
			MaxLat:           maxLat,
			From:             from,
			To:               to,
			Query:            q,
			Limit:            maxGlobalEvents,
			Cursor:           cursorState.EventCursor,
			DisableProximity: false,
		})
	} else {
		eventResults, eventNextCursor, err = h.eventRepo.SearchEvents(scene.EventSearchOptions{
			MinLng:           -180,
			MinLat:           -90,
			MaxLng:           180,
			MaxLat:           90,
			From:             from,
			To:               to,
			Query:            q,
			Limit:            maxGlobalEvents,
			Cursor:           cursorState.EventCursor,
			DisableProximity: true,
		})
	}
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to search events for global search", "error", err)
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeInternal)
		WriteError(w, ctx, http.StatusInternalServerError, ErrCodeInternal, "Failed to search")
		return
	}

	postResults := make([]*post.Post, 0)
	postNextCursor := ""
	if h.postRepo != nil {
		postResults, postNextCursor, err = h.postRepo.SearchPosts(q, nil, maxGlobalPosts, cursorState.PostCursor, nil)
		if err != nil {
			slog.ErrorContext(r.Context(), "failed to search posts for global search", "error", err)
			ctx := middleware.SetErrorCode(r.Context(), ErrCodeInternal)
			WriteError(w, ctx, http.StatusInternalServerError, ErrCodeInternal, "Failed to search")
			return
		}
	}

	type scoredGlobalResult struct {
		result *GlobalSearchResult
		score  float64
		key    string
	}
	scored := make([]scoredGlobalResult, 0, len(sceneResults)+len(eventResults)+len(postResults))
	for i, s := range sceneResults {
		sceneResult := &SceneSearchResult{
			ID:            s.ID,
			Name:          s.Name,
			Description:   s.Description,
			CoarseGeohash: s.CoarseGeohash,
			Tags:          s.Tags,
			Visibility:    s.Visibility,
		}
		if s.PrecisePoint != nil {
			sceneResult.JitteredPoint = applyJitter(s.PrecisePoint)
		}
		scored = append(scored, scoredGlobalResult{
			result: &GlobalSearchResult{Type: "scene", Scene: sceneResult},
			score:  globalNormalizedScore(i, len(sceneResults)),
			key:    "scene:" + s.ID,
		})
	}
	for i, e := range eventResults {
		createdAt := ""
		if e.CreatedAt != nil {
			createdAt = e.CreatedAt.Format(time.RFC3339)
		}
		scored = append(scored, scoredGlobalResult{
			result: &GlobalSearchResult{
				Type: "event",
				Event: &GlobalEventSearchResult{
					ID:        e.ID,
					SceneID:   e.SceneID,
					Title:     e.Title,
					StartsAt:  e.StartsAt.Format(time.RFC3339),
					CreatedAt: createdAt,
				},
			},
			score: globalNormalizedScore(i, len(eventResults)),
			key:   "event:" + e.ID,
		})
	}
	for i, p := range postResults {
		scored = append(scored, scoredGlobalResult{
			result: &GlobalSearchResult{
				Type: "post",
				Post: &PostSearchResult{
					ID:        p.ID,
					Excerpt:   makeExcerpt(p.Text, 160),
					SceneID:   p.SceneID,
					CreatedAt: p.CreatedAt.Format(time.RFC3339),
				},
			},
			score: globalNormalizedScore(i, len(postResults)),
			key:   "post:" + p.ID,
		})
	}

	sort.Slice(scored, func(i, j int) bool {
		if scored[i].score == scored[j].score {
			return scored[i].key < scored[j].key
		}
		return scored[i].score > scored[j].score
	})

	results := make([]*GlobalSearchResult, 0, min(MaxGlobalLimit, len(scored)))
	for _, item := range scored {
		if len(results) >= MaxGlobalLimit {
			break
		}
		results = append(results, item.result)
	}

	nextCursor := ""
	if sceneNextCursor != "" || eventNextCursor != "" || postNextCursor != "" {
		nextCursor = encodeGlobalSearchCursor(globalSearchCursor{
			SceneCursor: sceneNextCursor,
			EventCursor: eventNextCursor,
			PostCursor:  postNextCursor,
		})
	}

	response := GlobalSearchResponse{
		Results:    results,
		NextCursor: nextCursor,
		Count:      len(results),
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		slog.ErrorContext(r.Context(), "failed to encode global search response", "error", err)
	}
}

func globalNormalizedScore(idx int, total int) float64 {
	if total <= 0 {
		return 0
	}
	return 1.0 - (float64(idx) / float64(total))
}

func decodeGlobalSearchCursor(cursor string) (globalSearchCursor, error) {
	if strings.TrimSpace(cursor) == "" {
		return globalSearchCursor{}, nil
	}
	decoded, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return globalSearchCursor{}, err
	}
	var state globalSearchCursor
	if err := json.Unmarshal(decoded, &state); err != nil {
		return globalSearchCursor{}, err
	}
	return state, nil
}

func encodeGlobalSearchCursor(state globalSearchCursor) string {
	encoded, err := json.Marshal(state)
	if err != nil {
		slog.Error("failed to encode global search cursor", "error", err)
		return ""
	}
	return base64.RawURLEncoding.EncodeToString(encoded)
}

// SearchPosts handles GET /search/posts - searches for posts with text relevance and scene filter.
func (h *SearchHandlers) SearchPosts(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	query := r.URL.Query()

	// Get text search query (required)
	q := strings.TrimSpace(query.Get("q"))
	if q == "" {
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeValidation)
		WriteError(w, ctx, http.StatusBadRequest, ErrCodeValidation, "q parameter is required")
		return
	}

	// Get optional scene filter
	var sceneID *string
	if sceneIDStr := query.Get("scene_id"); sceneIDStr != "" {
		sceneID = &sceneIDStr
	}

	// Get optional type filter (text, image, video, audio, link)
	postType := strings.ToLower(strings.TrimSpace(query.Get("type")))
	if postType != "" && postType != "text" && postType != "image" && postType != "video" && postType != "audio" && postType != "link" {
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeValidation)
		WriteError(w, ctx, http.StatusBadRequest, ErrCodeValidation, "type must be one of: text, image, video, audio, link")
		return
	}

	// Get optional temporal filters (RFC3339 timestamps)
	var createdAfter, createdBefore *time.Time
	if createdAfterStr := query.Get("created_after"); createdAfterStr != "" {
		t, err := time.Parse(time.RFC3339, createdAfterStr)
		if err != nil {
			ctx := middleware.SetErrorCode(r.Context(), ErrCodeValidation)
			WriteError(w, ctx, http.StatusBadRequest, ErrCodeValidation, "created_after must be an RFC3339 timestamp")
			return
		}
		createdAfter = &t
	}
	if createdBeforeStr := query.Get("created_before"); createdBeforeStr != "" {
		t, err := time.Parse(time.RFC3339, createdBeforeStr)
		if err != nil {
			ctx := middleware.SetErrorCode(r.Context(), ErrCodeValidation)
			WriteError(w, ctx, http.StatusBadRequest, ErrCodeValidation, "created_before must be an RFC3339 timestamp")
			return
		}
		createdBefore = &t
	}

	// Get pagination parameters
	cursor := query.Get("cursor")

	limit := DefaultSearchLimit
	if limitStr := query.Get("limit"); limitStr != "" {
		var err error
		limit, err = strconv.Atoi(limitStr)
		if err != nil || limit < 1 {
			ctx := middleware.SetErrorCode(r.Context(), ErrCodeValidation)
			WriteError(w, ctx, http.StatusBadRequest, ErrCodeValidation, "limit must be a positive integer")
			return
		}
		if limit > MaxSearchLimit {
			limit = MaxSearchLimit
		}
	}

	// Trust scores are not yet implemented for post search
	// Pass nil to use text relevance only
	var trustScores map[string]float64 = nil

	// Execute search
	results, nextCursor, err := h.postRepo.SearchPosts(q, sceneID, limit, cursor, trustScores)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to search posts", "error", err, "query", q, "scene_id", sceneID)
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeInternal)
		WriteError(w, ctx, http.StatusInternalServerError, ErrCodeInternal, "Failed to search posts")
		return
	}

	// Apply post-search filters (type and temporal)
	filtered := make([]*post.Post, 0, len(results))
	for _, p := range results {
		// Apply temporal filters
		if createdAfter != nil && p.CreatedAt.Before(*createdAfter) {
			continue
		}
		if createdBefore != nil && p.CreatedAt.After(*createdBefore) {
			continue
		}

		// Apply type filter
		if postType != "" {
			if !matchesPostType(p, postType) {
				continue
			}
		}

		filtered = append(filtered, p)
	}

	// Convert to search results with excerpts
	searchResults := make([]*PostSearchResult, 0, len(filtered))
	for _, p := range filtered {
		result := &PostSearchResult{
			ID:        p.ID,
			Excerpt:   makeExcerpt(p.Text, 160),
			SceneID:   p.SceneID,
			CreatedAt: p.CreatedAt.Format("2006-01-02T15:04:05Z07:00"), // ISO 8601
		}

		// Note: Trust score integration is not yet implemented for post search
		// trust_score field will always be nil and is omitted from JSON response

		searchResults = append(searchResults, result)
	}

	// Build response
	response := PostSearchResponse{
		Results:    searchResults,
		NextCursor: nextCursor,
		Count:      len(searchResults),
	}

	// Return results
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		slog.ErrorContext(r.Context(), "failed to encode search response", "error", err)
		return
	}
}

// makeExcerpt creates a text excerpt of the specified length.
// Truncates at word boundary if possible.
func makeExcerpt(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}

	// Try to truncate at word boundary
	truncated := text[:maxLen]
	lastSpace := strings.LastIndex(truncated, " ")
	if lastSpace > maxLen/2 {
		// If we found a space in the second half, use it
		return truncated[:lastSpace] + "..."
	}

	// Otherwise just truncate at max length
	return truncated + "..."
}

// matchesPostType checks if a post matches the specified type filter.
// Type filters:
// - "text": posts with no attachments or only text content
// - "image": posts with at least one image/* attachment
// - "video": posts with at least one video/* attachment
// - "audio": posts with at least one audio/* attachment
// - "link": posts with at least one link attachment (non-media MIME type)
func matchesPostType(p *post.Post, postType string) bool {
	switch postType {
	case "text":
		// Text posts have no attachments or only legacy/unknown types
		if len(p.Attachments) == 0 {
			return true
		}
		for _, att := range p.Attachments {
			// If any attachment has a recognized media type, it's not a text-only post
			mimeType := strings.ToLower(att.Type)
			if strings.HasPrefix(mimeType, "image/") ||
				strings.HasPrefix(mimeType, "video/") ||
				strings.HasPrefix(mimeType, "audio/") {
				return false
			}
		}
		return true
	case "image":
		// Image posts have at least one image/* attachment
		for _, att := range p.Attachments {
			if strings.HasPrefix(strings.ToLower(att.Type), "image/") {
				return true
			}
		}
		return false
	case "video":
		// Video posts have at least one video/* attachment
		for _, att := range p.Attachments {
			if strings.HasPrefix(strings.ToLower(att.Type), "video/") {
				return true
			}
		}
		return false
	case "audio":
		// Audio posts have at least one audio/* attachment
		for _, att := range p.Attachments {
			if strings.HasPrefix(strings.ToLower(att.Type), "audio/") {
				return true
			}
		}
		return false
	case "link":
		// Link posts have at least one attachment that is not an image, video, or audio
		for _, att := range p.Attachments {
			mimeType := strings.ToLower(att.Type)
			if att.URL != "" && !strings.HasPrefix(mimeType, "image/") &&
				!strings.HasPrefix(mimeType, "video/") &&
				!strings.HasPrefix(mimeType, "audio/") {
				return true
			}
		}
		return false
	default:
		return true // Unknown type, include by default
	}
}
