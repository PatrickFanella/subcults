package post

import (
	"testing"
)

func TestValidateLabels(t *testing.T) {
	tests := []struct {
		name    string
		labels  []string
		wantErr bool
	}{
		{
			name:    "valid single label",
			labels:  []string{LabelHidden},
			wantErr: false,
		},
		{
			name:    "valid multiple labels",
			labels:  []string{LabelNSFW, LabelFlagged},
			wantErr: false,
		},
		{
			name:    "all valid labels",
			labels:  AllowedLabels,
			wantErr: false,
		},
		{
			name:    "empty labels list",
			labels:  []string{},
			wantErr: false,
		},
		{
			name:    "nil labels",
			labels:  nil,
			wantErr: false,
		},
		{
			name:    "invalid label",
			labels:  []string{"invalid"},
			wantErr: true,
		},
		{
			name:    "mix of valid and invalid",
			labels:  []string{LabelHidden, "invalid"},
			wantErr: true,
		},
		{
			name:    "case sensitive check",
			labels:  []string{"HIDDEN"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateLabels(tt.labels)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateLabels() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err != ErrInvalidLabel {
				t.Errorf("ValidateLabels() expected ErrInvalidLabel, got %v", err)
			}
		})
	}
}

func TestFilterPostsForUser_Hidden(t *testing.T) {
	authorDID := "did:example:alice"
	otherDID := "did:example:bob"

	tests := []struct {
		name             string
		post             *Post
		prefs            *UserPreferences
		viewerDID        string
		includeModerated bool
		want             bool
	}{
		{
			name: "hidden post excluded for public",
			post: &Post{
				ID:        "1",
				AuthorDID: authorDID,
				Text:      "Hidden post",
				Labels:    []string{LabelHidden},
			},
			prefs:            &UserPreferences{},
			viewerDID:        "",
			includeModerated: true,
			want:             false,
		},
		{
			name: "hidden post excluded for other users",
			post: &Post{
				ID:        "1",
				AuthorDID: authorDID,
				Text:      "Hidden post",
				Labels:    []string{LabelHidden},
			},
			prefs:            &UserPreferences{},
			viewerDID:        otherDID,
			includeModerated: true,
			want:             false,
		},
		{
			name: "hidden post visible to owner",
			post: &Post{
				ID:        "1",
				AuthorDID: authorDID,
				Text:      "Hidden post",
				Labels:    []string{LabelHidden},
			},
			prefs:            &UserPreferences{},
			viewerDID:        authorDID,
			includeModerated: true,
			want:             true,
		},
		{
			name: "non-hidden post visible to everyone",
			post: &Post{
				ID:        "1",
				AuthorDID: authorDID,
				Text:      "Public post",
				Labels:    []string{},
			},
			prefs:            &UserPreferences{},
			viewerDID:        "",
			includeModerated: true,
			want:             true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			posts := []*Post{tt.post}
			filtered := FilterPostsForUser(posts, tt.prefs, tt.viewerDID, tt.includeModerated)

			got := len(filtered) > 0
			if got != tt.want {
				t.Errorf("FilterPostsForUser() included=%v, want=%v", got, tt.want)
			}
		})
	}
}

func TestFilterPostsForUser_NSFW(t *testing.T) {
	authorDID := "did:example:alice"
	otherDID := "did:example:bob"

	tests := []struct {
		name             string
		post             *Post
		prefs            *UserPreferences
		viewerDID        string
		includeModerated bool
		want             bool
	}{
		{
			name: "nsfw post excluded when ShowNSFW is false",
			post: &Post{
				ID:        "1",
				AuthorDID: authorDID,
				Text:      "Adult content",
				Labels:    []string{LabelNSFW},
			},
			prefs:            &UserPreferences{ShowNSFW: false},
			viewerDID:        otherDID,
			includeModerated: true,
			want:             false,
		},
		{
			name: "nsfw post visible when ShowNSFW is true",
			post: &Post{
				ID:        "1",
				AuthorDID: authorDID,
				Text:      "Adult content",
				Labels:    []string{LabelNSFW},
			},
			prefs:            &UserPreferences{ShowNSFW: true},
			viewerDID:        otherDID,
			includeModerated: true,
			want:             true,
		},
		{
			name: "nsfw post visible to owner regardless of preference",
			post: &Post{
				ID:        "1",
				AuthorDID: authorDID,
				Text:      "Adult content",
				Labels:    []string{LabelNSFW},
			},
			prefs:            &UserPreferences{ShowNSFW: false},
			viewerDID:        authorDID,
			includeModerated: true,
			want:             true,
		},
		{
			name: "nsfw post excluded for public when ShowNSFW is false (nil prefs)",
			post: &Post{
				ID:        "1",
				AuthorDID: authorDID,
				Text:      "Adult content",
				Labels:    []string{LabelNSFW},
			},
			prefs:            nil, // Defaults to ShowNSFW: false
			viewerDID:        "",
			includeModerated: true,
			want:             false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			posts := []*Post{tt.post}
			filtered := FilterPostsForUser(posts, tt.prefs, tt.viewerDID, tt.includeModerated)

			got := len(filtered) > 0
			if got != tt.want {
				t.Errorf("FilterPostsForUser() included=%v, want=%v", got, tt.want)
			}
		})
	}
}

func TestFilterPostsForUser_SpamAndFlagged(t *testing.T) {
	authorDID := "did:example:alice"
	otherDID := "did:example:bob"

	tests := []struct {
		name             string
		post             *Post
		prefs            *UserPreferences
		viewerDID        string
		includeModerated bool
		want             bool
	}{
		{
			name: "spam post excluded from search (includeModerated=false)",
			post: &Post{
				ID:        "1",
				AuthorDID: authorDID,
				Text:      "Spam content",
				Labels:    []string{LabelSpam},
			},
			prefs:            &UserPreferences{},
			viewerDID:        otherDID,
			includeModerated: false,
			want:             false,
		},
		{
			name: "spam post visible in feed (includeModerated=true)",
			post: &Post{
				ID:        "1",
				AuthorDID: authorDID,
				Text:      "Spam content",
				Labels:    []string{LabelSpam},
			},
			prefs:            &UserPreferences{},
			viewerDID:        otherDID,
			includeModerated: true,
			want:             true,
		},
		{
			name: "spam post visible to owner in search",
			post: &Post{
				ID:        "1",
				AuthorDID: authorDID,
				Text:      "Spam content",
				Labels:    []string{LabelSpam},
			},
			prefs:            &UserPreferences{},
			viewerDID:        authorDID,
			includeModerated: false,
			want:             true,
		},
		{
			name: "flagged post excluded from search (includeModerated=false)",
			post: &Post{
				ID:        "1",
				AuthorDID: authorDID,
				Text:      "Flagged content",
				Labels:    []string{LabelFlagged},
			},
			prefs:            &UserPreferences{},
			viewerDID:        otherDID,
			includeModerated: false,
			want:             false,
		},
		{
			name: "flagged post visible in feed (includeModerated=true)",
			post: &Post{
				ID:        "1",
				AuthorDID: authorDID,
				Text:      "Flagged content",
				Labels:    []string{LabelFlagged},
			},
			prefs:            &UserPreferences{},
			viewerDID:        otherDID,
			includeModerated: true,
			want:             true,
		},
		{
			name: "flagged post visible to owner in search",
			post: &Post{
				ID:        "1",
				AuthorDID: authorDID,
				Text:      "Flagged content",
				Labels:    []string{LabelFlagged},
			},
			prefs:            &UserPreferences{},
			viewerDID:        authorDID,
			includeModerated: false,
			want:             true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			posts := []*Post{tt.post}
			filtered := FilterPostsForUser(posts, tt.prefs, tt.viewerDID, tt.includeModerated)

			got := len(filtered) > 0
			if got != tt.want {
				t.Errorf("FilterPostsForUser() included=%v, want=%v", got, tt.want)
			}
		})
	}
}

func TestFilterPostsForUser_MultipleLabels(t *testing.T) {
	authorDID := "did:example:alice"
	otherDID := "did:example:bob"

	tests := []struct {
		name             string
		post             *Post
		prefs            *UserPreferences
		viewerDID        string
		includeModerated bool
		want             bool
	}{
		{
			name: "hidden+nsfw post excluded (hidden takes precedence)",
			post: &Post{
				ID:        "1",
				AuthorDID: authorDID,
				Text:      "Hidden and NSFW",
				Labels:    []string{LabelHidden, LabelNSFW},
			},
			prefs:            &UserPreferences{ShowNSFW: true},
			viewerDID:        otherDID,
			includeModerated: true,
			want:             false,
		},
		{
			name: "nsfw+spam post excluded when NSFW off",
			post: &Post{
				ID:        "1",
				AuthorDID: authorDID,
				Text:      "NSFW and Spam",
				Labels:    []string{LabelNSFW, LabelSpam},
			},
			prefs:            &UserPreferences{ShowNSFW: false},
			viewerDID:        otherDID,
			includeModerated: true,
			want:             false,
		},
		{
			name: "nsfw+spam post excluded in search even with NSFW on",
			post: &Post{
				ID:        "1",
				AuthorDID: authorDID,
				Text:      "NSFW and Spam",
				Labels:    []string{LabelNSFW, LabelSpam},
			},
			prefs:            &UserPreferences{ShowNSFW: true},
			viewerDID:        otherDID,
			includeModerated: false,
			want:             false,
		},
		{
			name: "owner sees hidden+nsfw+spam post",
			post: &Post{
				ID:        "1",
				AuthorDID: authorDID,
				Text:      "All labels",
				Labels:    []string{LabelHidden, LabelNSFW, LabelSpam},
			},
			prefs:            &UserPreferences{ShowNSFW: false},
			viewerDID:        authorDID,
			includeModerated: false,
			want:             true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			posts := []*Post{tt.post}
			filtered := FilterPostsForUser(posts, tt.prefs, tt.viewerDID, tt.includeModerated)

			got := len(filtered) > 0
			if got != tt.want {
				t.Errorf("FilterPostsForUser() included=%v, want=%v", got, tt.want)
			}
		})
	}
}

func TestFilterPostsForUser_MultiplePosts(t *testing.T) {
	authorDID := "did:example:alice"

	posts := []*Post{
		{
			ID:        "1",
			AuthorDID: authorDID,
			Text:      "Public post",
			Labels:    []string{},
		},
		{
			ID:        "2",
			AuthorDID: authorDID,
			Text:      "Hidden post",
			Labels:    []string{LabelHidden},
		},
		{
			ID:        "3",
			AuthorDID: authorDID,
			Text:      "NSFW post",
			Labels:    []string{LabelNSFW},
		},
		{
			ID:        "4",
			AuthorDID: authorDID,
			Text:      "Spam post",
			Labels:    []string{LabelSpam},
		},
	}

	tests := []struct {
		name             string
		prefs            *UserPreferences
		viewerDID        string
		includeModerated bool
		wantCount        int
		wantIDs          []string
	}{
		{
			name:             "public user in search context",
			prefs:            &UserPreferences{ShowNSFW: false},
			viewerDID:        "",
			includeModerated: false,
			wantCount:        1,
			wantIDs:          []string{"1"}, // Only public post
		},
		{
			name:             "public user in feed context",
			prefs:            &UserPreferences{ShowNSFW: false},
			viewerDID:        "",
			includeModerated: true,
			wantCount:        2,
			wantIDs:          []string{"1", "4"}, // Public and spam
		},
		{
			name:             "nsfw user in search context",
			prefs:            &UserPreferences{ShowNSFW: true},
			viewerDID:        "",
			includeModerated: false,
			wantCount:        2,
			wantIDs:          []string{"1", "3"}, // Public and NSFW
		},
		{
			name:             "nsfw user in feed context",
			prefs:            &UserPreferences{ShowNSFW: true},
			viewerDID:        "",
			includeModerated: true,
			wantCount:        3,
			wantIDs:          []string{"1", "3", "4"}, // Public, NSFW, and spam
		},
		{
			name:             "owner sees all in search",
			prefs:            &UserPreferences{ShowNSFW: false},
			viewerDID:        authorDID,
			includeModerated: false,
			wantCount:        4,
			wantIDs:          []string{"1", "2", "3", "4"}, // All posts
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filtered := FilterPostsForUser(posts, tt.prefs, tt.viewerDID, tt.includeModerated)

			if len(filtered) != tt.wantCount {
				t.Errorf("FilterPostsForUser() returned %d posts, want %d", len(filtered), tt.wantCount)
			}

			// Check that we got the expected post IDs
			gotIDs := make([]string, len(filtered))
			for i, post := range filtered {
				gotIDs[i] = post.ID
			}

			for _, wantID := range tt.wantIDs {
				found := false
				for _, gotID := range gotIDs {
					if gotID == wantID {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("FilterPostsForUser() missing post ID %s, got %v", wantID, gotIDs)
				}
			}
		})
	}
}

func TestFilterPostsForUser_EdgeCases(t *testing.T) {
	t.Run("empty posts list", func(t *testing.T) {
		posts := []*Post{}
		filtered := FilterPostsForUser(posts, nil, "", true)
		if len(filtered) != 0 {
			t.Errorf("Expected empty result for empty input")
		}
	})

	t.Run("nil posts list", func(t *testing.T) {
		var posts []*Post
		filtered := FilterPostsForUser(posts, nil, "", true)
		if filtered == nil {
			t.Errorf("Expected non-nil result")
		}
		if len(filtered) != 0 {
			t.Errorf("Expected empty result for nil input")
		}
	})

	t.Run("nil post in list is filtered out", func(t *testing.T) {
		posts := []*Post{
			{ID: "1", AuthorDID: "did:example:alice", Text: "Test"},
			nil,
			{ID: "2", AuthorDID: "did:example:bob", Text: "Test 2"},
		}
		filtered := FilterPostsForUser(posts, nil, "", true)
		if len(filtered) != 2 {
			t.Errorf("Expected 2 posts, got %d", len(filtered))
		}
	})
}

func TestPost_LabelHelperMethods(t *testing.T) {
	post := &Post{
		ID:        "1",
		AuthorDID: "did:example:alice",
		Text:      "Test",
		Labels:    []string{LabelHidden, LabelNSFW},
	}

	if !post.HasLabel(LabelHidden) {
		t.Error("Expected post to have hidden label")
	}
	if !post.HasLabel(LabelNSFW) {
		t.Error("Expected post to have nsfw label")
	}
	if post.HasLabel(LabelSpam) {
		t.Error("Expected post not to have spam label")
	}

	if !post.IsHidden() {
		t.Error("Expected IsHidden() to return true")
	}
	if !post.IsNSFW() {
		t.Error("Expected IsNSFW() to return true")
	}
	if post.IsSpam() {
		t.Error("Expected IsSpam() to return false")
	}
	if post.IsFlagged() {
		t.Error("Expected IsFlagged() to return false")
	}
}

func TestPost_LabelHelperMethods_NoLabels(t *testing.T) {
	post := &Post{
		ID:        "1",
		AuthorDID: "did:example:alice",
		Text:      "Test",
		Labels:    []string{},
	}

	if post.IsHidden() {
		t.Error("Expected IsHidden() to return false")
	}
	if post.IsNSFW() {
		t.Error("Expected IsNSFW() to return false")
	}
	if post.IsSpam() {
		t.Error("Expected IsSpam() to return false")
	}
	if post.IsFlagged() {
		t.Error("Expected IsFlagged() to return false")
	}
}
