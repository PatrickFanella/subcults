package middleware

import (
	"testing"
)

func TestNormalizePath(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		// Static routes - no normalization
		{
			name:     "root path",
			path:     "/",
			expected: "/",
		},
		{
			name:     "events collection",
			path:     "/events",
			expected: "/events",
		},
		{
			name:     "scenes collection",
			path:     "/scenes",
			expected: "/scenes",
		},
		{
			name:     "search events",
			path:     "/search/events",
			expected: "/search/events",
		},
		{
			name:     "health endpoint",
			path:     "/health",
			expected: "/health",
		},
		{
			name:     "ready endpoint",
			path:     "/ready",
			expected: "/ready",
		},
		{
			name:     "metrics endpoint",
			path:     "/metrics",
			expected: "/metrics",
		},

		// Events patterns
		{
			name:     "event by id",
			path:     "/events/123",
			expected: "/events/{id}",
		},
		{
			name:     "event by uuid",
			path:     "/events/550e8400-e29b-41d4-a716-446655440000",
			expected: "/events/{id}",
		},
		{
			name:     "event cancel",
			path:     "/events/123/cancel",
			expected: "/events/{id}/cancel",
		},
		{
			name:     "event rsvp",
			path:     "/events/456/rsvp",
			expected: "/events/{id}/rsvp",
		},
		{
			name:     "event feed",
			path:     "/events/789/feed",
			expected: "/events/{id}/feed",
		},

		// Scenes patterns
		{
			name:     "scene by id",
			path:     "/scenes/abc123",
			expected: "/scenes/{id}",
		},
		{
			name:     "scene feed",
			path:     "/scenes/xyz789/feed",
			expected: "/scenes/{id}/feed",
		},

		// Streams patterns
		{
			name:     "stream by id",
			path:     "/streams/stream-123",
			expected: "/streams/{id}",
		},
		{
			name:     "stream end",
			path:     "/streams/stream-456/end",
			expected: "/streams/{id}/end",
		},
		{
			name:     "stream join",
			path:     "/streams/stream-789/join",
			expected: "/streams/{id}/join",
		},
		{
			name:     "stream leave",
			path:     "/streams/stream-abc/leave",
			expected: "/streams/{id}/leave",
		},
		{
			name:     "stream analytics",
			path:     "/streams/stream-def/analytics",
			expected: "/streams/{id}/analytics",
		},
		{
			name:     "stream lock",
			path:     "/streams/stream-ghi/lock",
			expected: "/streams/{id}/lock",
		},
		{
			name:     "stream featured participant",
			path:     "/streams/stream-jkl/featured_participant",
			expected: "/streams/{id}/featured_participant",
		},
		{
			name:     "stream participants",
			path:     "/streams/stream-mno/participants",
			expected: "/streams/{id}/participants",
		},
		{
			name:     "stream participant mute",
			path:     "/streams/stream-123/participants/participant-456/mute",
			expected: "/streams/{id}/participants/{participant_id}/mute",
		},
		{
			name:     "stream participant kick",
			path:     "/streams/stream-123/participants/participant-789/kick",
			expected: "/streams/{id}/participants/{participant_id}/kick",
		},

		// Posts patterns
		{
			name:     "post by id",
			path:     "/posts/post-123",
			expected: "/posts/{id}",
		},

		// Trust patterns
		{
			name:     "trust by did",
			path:     "/trust/did:plc:abc123def456",
			expected: "/trust/{id}",
		},
		{
			name:     "trust by simple id",
			path:     "/trust/user-123",
			expected: "/trust/{id}",
		},

		// Static payment routes
		{
			name:     "payments onboard",
			path:     "/payments/onboard",
			expected: "/payments/onboard",
		},
		{
			name:     "payments checkout",
			path:     "/payments/checkout",
			expected: "/payments/checkout",
		},
		{
			name:     "payments status",
			path:     "/payments/status",
			expected: "/payments/status",
		},

		// Edge cases
		{
			name:     "trailing slash on collection",
			path:     "/events/",
			expected: "/events/",
		},
		{
			name:     "unknown route",
			path:     "/unknown/path",
			expected: "/unknown/path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizePath(tt.path)
			if result != tt.expected {
				t.Errorf("normalizePath(%q) = %q, want %q", tt.path, result, tt.expected)
			}
		})
	}
}

func TestNormalizePath_CardinalityControl(t *testing.T) {
	// Test that different IDs normalize to the same pattern
	paths := []string{
		"/events/1",
		"/events/2",
		"/events/999",
		"/events/550e8400-e29b-41d4-a716-446655440000",
		"/events/abc-def-ghi",
	}

	expected := "/events/{id}"
	seen := make(map[string]bool)

	for _, path := range paths {
		result := normalizePath(path)
		if result != expected {
			t.Errorf("normalizePath(%q) = %q, want %q", path, result, expected)
		}
		seen[result] = true
	}

	// Should all normalize to the same pattern (low cardinality)
	if len(seen) != 1 {
		t.Errorf("Expected all paths to normalize to single pattern, got %d patterns: %v", len(seen), seen)
	}
}
