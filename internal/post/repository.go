// Package post provides models and repository for managing posts
// with AT Protocol record tracking for idempotent ingestion.
package post

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Common errors for post operations.
var (
	ErrPostNotFound = errors.New("post not found")
	ErrPostDeleted  = errors.New("post has been deleted")
)

// PostScoreCursor represents the pagination cursor for post search results.
type PostScoreCursor struct {
	Score float64 `json:"score"` // Composite score of last post
	ID    string  `json:"id"`    // Post ID for stable ordering
}

// EncodePostScoreCursor encodes a post score cursor to a base64 string.
func EncodePostScoreCursor(score float64, id string) string {
	cursor := PostScoreCursor{Score: score, ID: id}
	data, _ := json.Marshal(cursor)
	return base64.URLEncoding.EncodeToString(data)
}

// DecodePostScoreCursor decodes a base64 post score cursor string.
// Returns (nil, nil) for empty input, or an error for invalid cursors.
func DecodePostScoreCursor(encoded string) (*PostScoreCursor, error) {
	if encoded == "" {
		return nil, nil
	}
	data, err := base64.URLEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("invalid cursor encoding: %w", err)
	}
	var cursor PostScoreCursor
	if err := json.Unmarshal(data, &cursor); err != nil {
		return nil, fmt.Errorf("invalid cursor format: %w", err)
	}
	return &cursor, nil
}

// Attachment represents a media attachment on a post with sanitized metadata.
// Supports both legacy URL-based attachments and new key-based attachments with metadata.
type Attachment struct {
	// Legacy field for backward compatibility
	URL string `json:"url,omitempty"`

	// New fields for enriched attachments
	Key       string `json:"key,omitempty"`        // R2 object key (e.g., "posts/uuid/file.jpg")
	Type      string `json:"type,omitempty"`       // MIME type (e.g., "image/jpeg")
	SizeBytes int64  `json:"size_bytes,omitempty"` // File size in bytes

	// Image-specific metadata (populated for image/* types)
	Width  *int `json:"width,omitempty"`  // Image width in pixels
	Height *int `json:"height,omitempty"` // Image height in pixels

	// Audio-specific metadata (populated for audio/* types)
	DurationSeconds *float64 `json:"duration_seconds,omitempty"` // Audio duration in seconds
}

// Post represents a content post within scenes/events.
type Post struct {
	ID          string       `json:"id"`
	SceneID     *string      `json:"scene_id,omitempty"`
	EventID     *string      `json:"event_id,omitempty"`
	AuthorDID   string       `json:"author_did"`
	Text        string       `json:"text"`
	Attachments []Attachment `json:"attachments,omitempty"`
	Labels      []string     `json:"labels,omitempty"`

	// AT Protocol record tracking
	RecordDID  *string `json:"record_did,omitempty"`
	RecordRKey *string `json:"record_rkey,omitempty"`

	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
}

// UpsertResult tracks statistics for upsert operations.
type UpsertResult struct {
	Inserted bool   // True if new record was inserted
	ID       string // The UUID of the upserted record
}

// FeedCursor represents a cursor for paginating through a feed.
// Uses (created_at, id) for stable pagination with tie-breaking.
type FeedCursor struct {
	CreatedAt time.Time `json:"created_at"`
	ID        string    `json:"id"`
}

// PostRepository defines the interface for post data operations.
type PostRepository interface {
	// Upsert inserts a new post or updates existing one based on (record_did, record_rkey).
	// Returns UpsertResult indicating whether insert or update occurred.
	Upsert(post *Post) (*UpsertResult, error)

	// Create inserts a new post with a generated UUID.
	Create(post *Post) error

	// Update updates an existing post.
	Update(post *Post) error

	// Delete soft-deletes a post by setting deleted_at timestamp.
	Delete(id string) error

	// GetByID retrieves a post by its UUID, excluding soft-deleted posts.
	GetByID(id string) (*Post, error)

	// GetByRecordKey retrieves a post by its AT Protocol record key.
	GetByRecordKey(did, rkey string) (*Post, error)

	// ListByScene retrieves posts for a scene with cursor-based pagination.
	// Returns posts ordered by created_at DESC, id ASC (tie-breaker).
	// Excludes soft-deleted posts and posts with 'hidden' label.
	// If cursor is nil, starts from the most recent post.
	// Returns posts, next cursor (nil if no more), and error.
	ListByScene(sceneID string, limit int, cursor *FeedCursor) ([]*Post, *FeedCursor, error)

	// ListByEvent retrieves posts for an event with cursor-based pagination.
	// Returns posts ordered by created_at DESC, id ASC (tie-breaker).
	// Excludes soft-deleted posts and posts with 'hidden' label.
	// If cursor is nil, starts from the most recent post.
	// Returns posts, next cursor (nil if no more), and error.
	ListByEvent(eventID string, limit int, cursor *FeedCursor) ([]*Post, *FeedCursor, error)

	// SearchPosts searches for posts by text query with optional scene filter.
	// Returns posts ordered by (score DESC, id ASC) for stable pagination.
	// Excludes soft-deleted posts and posts with moderation labels (hidden, spam, flagged).
	// If cursor is empty, starts from the highest scored post.
	// Returns posts, next cursor (empty if no more), and error.
	SearchPosts(query string, sceneID *string, limit int, cursor string, trustScores map[string]float64) ([]*Post, string, error)
}

// InMemoryPostRepository is an in-memory implementation of PostRepository.
// Thread-safe via RWMutex.
type InMemoryPostRepository struct {
	mu    sync.RWMutex
	posts map[string]*Post  // UUID -> Post
	keys  map[string]string // "did:rkey" -> UUID
}

// NewInMemoryPostRepository creates a new in-memory post repository.
func NewInMemoryPostRepository() *InMemoryPostRepository {
	return &InMemoryPostRepository{
		posts: make(map[string]*Post),
		keys:  make(map[string]string),
	}
}

// makeKey creates a composite key from DID and rkey using a null byte separator to avoid collisions.
// AT Protocol DIDs contain colons (e.g., "did:plc:abc123"), so using a null byte prevents
// collisions like did="a:b" + rkey="c" vs did="a" + rkey="b:c" both producing "a:b:c".
func makeKey(did, rkey string) string {
	return did + "\x00" + rkey
}

// Upsert inserts a new post or updates existing one based on (record_did, record_rkey).
func (r *InMemoryPostRepository) Upsert(post *Post) (*UpsertResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	var inserted bool
	var id string

	// Check if post exists by record key
	if post.RecordDID != nil && post.RecordRKey != nil {
		key := makeKey(*post.RecordDID, *post.RecordRKey)
		existingID, exists := r.keys[key]

		if exists {
			// Update existing post
			existing := r.posts[existingID]
			existing.SceneID = post.SceneID
			existing.EventID = post.EventID
			existing.AuthorDID = post.AuthorDID
			existing.Text = post.Text
			existing.Attachments = post.Attachments
			existing.Labels = post.Labels
			existing.UpdatedAt = now
			inserted = false
			id = existingID
		} else {
			// Insert new post
			if post.ID == "" {
				post.ID = uuid.New().String()
			}
			post.CreatedAt = now
			post.UpdatedAt = now

			postCopy := *post
			r.posts[post.ID] = &postCopy
			r.keys[key] = post.ID
			inserted = true
			id = post.ID
		}
	} else {
		// No record key, always insert new with new UUID
		newID := uuid.New().String()
		post.ID = newID
		post.CreatedAt = now
		post.UpdatedAt = now

		postCopy := *post
		r.posts[newID] = &postCopy
		inserted = true
		id = newID
	}

	return &UpsertResult{
		Inserted: inserted,
		ID:       id,
	}, nil
}

// Create inserts a new post with a generated UUID.
func (r *InMemoryPostRepository) Create(post *Post) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	post.ID = uuid.New().String()
	post.CreatedAt = now
	post.UpdatedAt = now

	postCopy := *post
	r.posts[post.ID] = &postCopy

	// If record key is provided, track it
	if post.RecordDID != nil && post.RecordRKey != nil {
		key := makeKey(*post.RecordDID, *post.RecordRKey)
		r.keys[key] = post.ID
	}

	return nil
}

// Update updates an existing post.
func (r *InMemoryPostRepository) Update(post *Post) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	existing, ok := r.posts[post.ID]
	if !ok {
		return ErrPostNotFound
	}

	// Don't allow updating deleted posts
	if existing.DeletedAt != nil {
		return ErrPostDeleted
	}

	// Update mutable fields
	existing.Text = post.Text
	existing.Attachments = post.Attachments
	existing.Labels = post.Labels
	existing.UpdatedAt = time.Now()

	return nil
}

// Delete soft-deletes a post by setting deleted_at timestamp.
func (r *InMemoryPostRepository) Delete(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	post, ok := r.posts[id]
	if !ok {
		return ErrPostNotFound
	}

	// Already deleted - treat as not found for idempotency
	if post.DeletedAt != nil {
		return ErrPostNotFound
	}

	now := time.Now()
	post.DeletedAt = &now

	return nil
}

// GetByID retrieves a post by its UUID, excluding soft-deleted posts.
func (r *InMemoryPostRepository) GetByID(id string) (*Post, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	post, ok := r.posts[id]
	if !ok {
		return nil, ErrPostNotFound
	}

	// Exclude soft-deleted posts
	if post.DeletedAt != nil {
		return nil, ErrPostNotFound
	}

	postCopy := *post
	return &postCopy, nil
}

// GetByRecordKey retrieves a post by its AT Protocol record key.
func (r *InMemoryPostRepository) GetByRecordKey(did, rkey string) (*Post, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	key := makeKey(did, rkey)
	id, ok := r.keys[key]
	if !ok {
		return nil, ErrPostNotFound
	}

	post := r.posts[id]
	postCopy := *post
	return &postCopy, nil
}

// ListByScene retrieves posts for a scene with cursor-based pagination.
func (r *InMemoryPostRepository) ListByScene(sceneID string, limit int, cursor *FeedCursor) ([]*Post, *FeedCursor, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Collect all non-deleted posts for this scene
	var candidates []*Post
	for _, post := range r.posts {
		// Skip deleted posts
		if post.DeletedAt != nil {
			continue
		}

		// Skip posts not in this scene
		if post.SceneID == nil || *post.SceneID != sceneID {
			continue
		}

		// Skip hidden posts
		if post.HasLabel(LabelHidden) {
			continue
		}

		// Apply cursor filter if provided
		if cursor != nil {
			// Skip posts that are newer or at/before the cursor position
			if post.CreatedAt.After(cursor.CreatedAt) {
				continue
			}
			if post.CreatedAt.Equal(cursor.CreatedAt) && post.ID <= cursor.ID {
				continue
			}
		}

		candidates = append(candidates, post)
	}

	// Sort by created_at DESC, then by ID ASC for tie-breaking
	// This ensures stable pagination
	sortPostsByCreatedDesc(candidates)

	// Apply limit and determine next cursor
	var results []*Post
	var nextCursor *FeedCursor

	if len(candidates) > limit {
		results = candidates[:limit]
		// Next cursor points to the last returned post
		lastPost := results[len(results)-1]
		nextCursor = &FeedCursor{
			CreatedAt: lastPost.CreatedAt,
			ID:        lastPost.ID,
		}
	} else {
		results = candidates
		// No more posts, cursor is nil
	}

	// Return deep copies to prevent external mutation
	copies := make([]*Post, len(results))
	for i, p := range results {
		postCopy := *p
		copies[i] = &postCopy
	}

	return copies, nextCursor, nil
}

// ListByEvent retrieves posts for an event with cursor-based pagination.
func (r *InMemoryPostRepository) ListByEvent(eventID string, limit int, cursor *FeedCursor) ([]*Post, *FeedCursor, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Collect all non-deleted posts for this event
	var candidates []*Post
	for _, post := range r.posts {
		// Skip deleted posts
		if post.DeletedAt != nil {
			continue
		}

		// Skip posts not in this event
		if post.EventID == nil || *post.EventID != eventID {
			continue
		}

		// Skip hidden posts
		if post.HasLabel(LabelHidden) {
			continue
		}

		// Apply cursor filter if provided
		if cursor != nil {
			// Skip posts that are newer or at/before the cursor position
			if post.CreatedAt.After(cursor.CreatedAt) {
				continue
			}
			if post.CreatedAt.Equal(cursor.CreatedAt) && post.ID <= cursor.ID {
				continue
			}
		}

		candidates = append(candidates, post)
	}

	// Sort by created_at DESC, then by ID ASC for tie-breaking
	sortPostsByCreatedDesc(candidates)

	// Apply limit and determine next cursor
	var results []*Post
	var nextCursor *FeedCursor

	if len(candidates) > limit {
		results = candidates[:limit]
		// Next cursor points to the last returned post
		lastPost := results[len(results)-1]
		nextCursor = &FeedCursor{
			CreatedAt: lastPost.CreatedAt,
			ID:        lastPost.ID,
		}
	} else {
		results = candidates
		// No more posts, cursor is nil
	}

	// Return deep copies to prevent external mutation
	copies := make([]*Post, len(results))
	for i, p := range results {
		postCopy := *p
		copies[i] = &postCopy
	}

	return copies, nextCursor, nil
}

// SearchPosts searches for posts by text query with optional scene filter.
// Returns posts ordered by (score DESC, id ASC) for stable pagination.
func (r *InMemoryPostRepository) SearchPosts(query string, sceneID *string, limit int, cursor string, trustScores map[string]float64) ([]*Post, string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Normalize query for case-insensitive matching
	queryLower := strings.ToLower(strings.TrimSpace(query))
	if queryLower == "" {
		return []*Post{}, "", nil
	}

	// Collect matching posts
	type scoredPost struct {
		post      *Post
		textScore float64
		score     float64
	}
	var candidates []scoredPost

	for _, post := range r.posts {
		// Skip deleted posts
		if post.DeletedAt != nil {
			continue
		}

		// Skip moderated posts (hidden, spam, flagged)
		if post.HasLabel(LabelHidden) || post.HasLabel(LabelSpam) || post.HasLabel(LabelFlagged) {
			continue
		}

		// Apply scene filter if provided
		if sceneID != nil {
			if post.SceneID == nil || *post.SceneID != *sceneID {
				continue
			}
		}

		// Calculate text relevance score (simple substring match)
		textLower := strings.ToLower(post.Text)
		textScore := 0.0
		if strings.Contains(textLower, queryLower) {
			// Exact substring match
			textScore = 1.0
		} else {
			// Check for individual word matches
			queryWords := strings.Fields(queryLower)
			matchCount := 0
			for _, word := range queryWords {
				if strings.Contains(textLower, word) {
					matchCount++
				}
			}
			if matchCount > 0 {
				textScore = float64(matchCount) / float64(len(queryWords))
			}
		}

		// Skip posts with no text match
		if textScore == 0.0 {
			continue
		}

		// Calculate recency score (exponential decay over 30 days)
		// Posts from the last 24h get score ~1.0, 1 week old ~0.8, 1 month old ~0.1
		const decayHalfLife = 7 * 24 * time.Hour // 7 days
		postAge := time.Since(post.CreatedAt)
		recencyScore := 1.0
		if postAge > 0 {
			// Exponential decay: score = exp(-ln(2) * age / halfLife)
			recencyScore = float64(int(1000*math.Exp(-0.693147*postAge.Hours()/decayHalfLife.Hours()))) / 1000.0
			if recencyScore < 0.01 {
				recencyScore = 0.01 // Floor to avoid zero scores
			}
		}

		// Calculate trust score from alliance-based trust weights (keyed by scene ID).
		// Falls back to 0.0 when trust scores are unavailable or post has no scene.
		trustScore := 0.0
		if trustScores != nil && post.SceneID != nil {
			if ts, ok := trustScores[*post.SceneID]; ok {
				trustScore = ts
			}
		}

		// Calculate composite score per issue #333:
		// relevance (0.4) + recency (0.3) + engagement (0.2) + trust (0.1)
		// engagementScore is 0.0 until per-post engagement metrics are available.
		score := (textScore * 0.4) + (recencyScore * 0.3) + (trustScore * 0.1)

		candidates = append(candidates, scoredPost{
			post:      post,
			textScore: textScore,
			score:     score,
		})
	}

	// Sort by score DESC, then by ID ASC for tie-breaking
	sort.Slice(candidates, func(i, j int) bool {
		// Sort by score DESC (higher scores first)
		if candidates[i].score > candidates[j].score {
			return true
		}
		if candidates[i].score < candidates[j].score {
			return false
		}
		// Tie-break by ID ASC (lexicographic order) when scores are equal
		return candidates[i].post.ID < candidates[j].post.ID
	})

	// Apply cursor filter if provided (base64-encoded JSON)
	if cursor != "" {
		decodedCursor, err := DecodePostScoreCursor(cursor)
		if err != nil {
			return nil, "", fmt.Errorf("invalid cursor: %w", err)
		}
		if decodedCursor != nil {
			cursorScore := decodedCursor.Score
			cursorID := decodedCursor.ID
			// In (score DESC, id ASC) order, items after the cursor are:
			// - posts with strictly lower score than the cursor score, or
			// - posts with the same score but an ID greater than the cursor ID.
			filtered := make([]scoredPost, 0, len(candidates))
			for _, candidate := range candidates {
				if candidate.score < cursorScore || (candidate.score == cursorScore && candidate.post.ID > cursorID) {
					filtered = append(filtered, candidate)
				}
			}
			candidates = filtered
		}
	}

	// Apply limit and generate next cursor
	var results []*Post
	var nextCursor string

	if len(candidates) > limit {
		results = make([]*Post, limit)
		for i := 0; i < limit; i++ {
			results[i] = candidates[i].post
		}
		// Next cursor points to the last returned post
		lastCandidate := candidates[limit-1]
		nextCursor = EncodePostScoreCursor(lastCandidate.score, lastCandidate.post.ID)
	} else {
		results = make([]*Post, len(candidates))
		for i, candidate := range candidates {
			results[i] = candidate.post
		}
		// No more posts, cursor is empty
	}

	// Return deep copies to prevent external mutation
	copies := make([]*Post, len(results))
	for i, p := range results {
		postCopy := *p
		copies[i] = &postCopy
	}

	return copies, nextCursor, nil
}

// sortPostsByCreatedDesc sorts posts by created_at DESC, then by ID ASC for tie-breaking.
// This provides stable ordering for cursor-based pagination.
// Uses sort.Slice with O(n log n) introsort for efficient sorting of large result sets.
func sortPostsByCreatedDesc(posts []*Post) {
	sort.Slice(posts, func(i, j int) bool {
		// Sort by created_at DESC (newer first)
		if posts[i].CreatedAt.After(posts[j].CreatedAt) {
			return true
		}
		if posts[i].CreatedAt.Before(posts[j].CreatedAt) {
			return false
		}
		// Tie-break by ID ASC (lexicographic order) when timestamps are equal
		return posts[i].ID < posts[j].ID
	})
}
