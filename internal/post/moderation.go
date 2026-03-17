// Package post provides moderation functionality for content filtering.
package post

import (
	"errors"
	"slices"
)

// Moderation label constants define allowed labels for content moderation.
// These labels control visibility and filtering behavior in public feeds and search.
const (
	// LabelHidden marks content that should be excluded from all public feeds and search.
	// Hidden content is never visible to anyone except the post owner.
	LabelHidden = "hidden"

	// LabelNSFW marks adult/mature content that requires explicit user opt-in.
	// NSFW content is only visible to users with ShowNSFW preference enabled.
	LabelNSFW = "nsfw"

	// LabelFlagged marks content that has been flagged for review.
	// Flagged content is excluded from search but remains visible to the owner.
	LabelFlagged = "flagged"

	// LabelSpam marks content identified as spam.
	// Spam content is excluded from search but remains visible to the owner.
	LabelSpam = "spam"
)

// AllowedLabels is the exhaustive list of valid moderation labels.
// Any label applied to content must be in this list.
var AllowedLabels = []string{
	LabelHidden,
	LabelNSFW,
	LabelFlagged,
	LabelSpam,
}

// Common errors for moderation operations.
var (
	ErrInvalidLabel = errors.New("invalid moderation label")
)

// ValidateLabels checks that all provided labels are in the allowed list.
// Returns an error if any label is not recognized.
func ValidateLabels(labels []string) error {
	for _, label := range labels {
		if !slices.Contains(AllowedLabels, label) {
			return ErrInvalidLabel
		}
	}
	return nil
}

// UserPreferences represents user-specific settings that affect content filtering.
// This is a placeholder struct that can be extended with additional preferences.
type UserPreferences struct {
	// ShowNSFW indicates whether the user has opted in to view NSFW content.
	// Default is false (NSFW content is hidden by default).
	ShowNSFW bool `json:"show_nsfw"`
}

// FilterPostsForUser filters a list of posts based on moderation labels and user preferences.
// Returns a new slice containing only posts that should be visible to the user.
//
// Filtering rules:
//   - Posts with 'hidden' label are always excluded from public feeds/search
//   - Posts with 'nsfw' label are excluded unless user has ShowNSFW=true
//   - Posts with 'spam' or 'flagged' labels are excluded from search contexts
//     (context parameter controls this; use includeModerated=false for search)
//   - Post owner always sees their own posts regardless of labels (requires viewerDID)
func FilterPostsForUser(posts []*Post, prefs *UserPreferences, viewerDID string, includeModerated bool) []*Post {
	// Return empty slice for nil or empty input
	if len(posts) == 0 {
		return []*Post{}
	}

	// Default preferences if nil
	if prefs == nil {
		prefs = &UserPreferences{ShowNSFW: false}
	}

	filtered := make([]*Post, 0, len(posts))
	for _, post := range posts {
		if shouldIncludePost(post, prefs, viewerDID, includeModerated) {
			filtered = append(filtered, post)
		}
	}

	return filtered
}

// shouldIncludePost determines if a single post should be visible based on filtering rules.
func shouldIncludePost(post *Post, prefs *UserPreferences, viewerDID string, includeModerated bool) bool {
	if post == nil {
		return false
	}

	// Owner always sees their own posts
	isOwner := viewerDID != "" && post.AuthorDID == viewerDID

	// Check each label for filtering rules
	for _, label := range post.Labels {
		switch label {
		case LabelHidden:
			// Hidden posts never appear in public contexts
			if !isOwner {
				return false
			}
		case LabelNSFW:
			// NSFW posts require explicit opt-in or ownership
			if !isOwner && !prefs.ShowNSFW {
				return false
			}
		case LabelSpam, LabelFlagged:
			// Spam/flagged excluded from search contexts unless owner
			if !isOwner && !includeModerated {
				return false
			}
		}
	}

	return true
}

// HasLabel checks if a post has a specific moderation label.
func (p *Post) HasLabel(label string) bool {
	return slices.Contains(p.Labels, label)
}

// IsHidden returns true if the post has the 'hidden' label.
func (p *Post) IsHidden() bool {
	return p.HasLabel(LabelHidden)
}

// IsNSFW returns true if the post has the 'nsfw' label.
func (p *Post) IsNSFW() bool {
	return p.HasLabel(LabelNSFW)
}

// IsFlagged returns true if the post has the 'flagged' label.
func (p *Post) IsFlagged() bool {
	return p.HasLabel(LabelFlagged)
}

// IsSpam returns true if the post has the 'spam' label.
func (p *Post) IsSpam() bool {
	return p.HasLabel(LabelSpam)
}
