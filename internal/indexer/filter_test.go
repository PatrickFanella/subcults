package indexer

import (
	"testing"
)

func TestMatchesLexicon(t *testing.T) {
	tests := []struct {
		name       string
		collection string
		want       bool
	}{
		{
			name:       "matches app.subcult.scene",
			collection: "app.subcult.scene",
			want:       true,
		},
		{
			name:       "matches app.subcult.event",
			collection: "app.subcult.event",
			want:       true,
		},
		{
			name:       "matches app.subcult.post",
			collection: "app.subcult.post",
			want:       true,
		},
		{
			name:       "matches app.subcult.custom",
			collection: "app.subcult.custom",
			want:       true,
		},
		{
			name:       "does not match app.bsky.feed.post",
			collection: "app.bsky.feed.post",
			want:       false,
		},
		{
			name:       "does not match app.bsky.actor.profile",
			collection: "app.bsky.actor.profile",
			want:       false,
		},
		{
			name:       "does not match empty string",
			collection: "",
			want:       false,
		},
		{
			name:       "does not match partial prefix",
			collection: "app.subcul",
			want:       false,
		},
		{
			name:       "does not match similar prefix",
			collection: "app.subculture.scene",
			want:       false,
		},
		{
			name:       "case sensitive - uppercase does not match",
			collection: "APP.SUBCULT.SCENE",
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MatchesLexicon(tt.collection); got != tt.want {
				t.Errorf("MatchesLexicon(%q) = %v, want %v", tt.collection, got, tt.want)
			}
		})
	}
}

func TestFilterMetrics(t *testing.T) {
	t.Run("initial values are zero", func(t *testing.T) {
		m := NewFilterMetrics()
		if m.Processed() != 0 {
			t.Errorf("initial Processed() = %d, want 0", m.Processed())
		}
		if m.Matched() != 0 {
			t.Errorf("initial Matched() = %d, want 0", m.Matched())
		}
		if m.Discarded() != 0 {
			t.Errorf("initial Discarded() = %d, want 0", m.Discarded())
		}
	})

	t.Run("increments are atomic and correct", func(t *testing.T) {
		m := NewFilterMetrics()
		for i := 0; i < 100; i++ {
			m.incProcessed()
		}
		for i := 0; i < 75; i++ {
			m.incMatched()
		}
		for i := 0; i < 25; i++ {
			m.incDiscarded()
		}

		if m.Processed() != 100 {
			t.Errorf("Processed() = %d, want 100", m.Processed())
		}
		if m.Matched() != 75 {
			t.Errorf("Matched() = %d, want 75", m.Matched())
		}
		if m.Discarded() != 25 {
			t.Errorf("Discarded() = %d, want 25", m.Discarded())
		}
	})
}

func TestRecordFilter_Filter_NonMatchingLexicon(t *testing.T) {
	metrics := NewFilterMetrics()
	filter := NewRecordFilter(metrics)

	tests := []struct {
		name       string
		collection string
	}{
		{name: "bsky post", collection: "app.bsky.feed.post"},
		{name: "bsky like", collection: "app.bsky.feed.like"},
		{name: "bsky profile", collection: "app.bsky.actor.profile"},
		{name: "empty collection", collection: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filter.Filter(tt.collection, []byte(`{"foo":"bar"}`))

			if result.Matched {
				t.Error("expected Matched = false for non-matching lexicon")
			}
			if result.Valid {
				t.Error("expected Valid = false for non-matching lexicon")
			}
			if result.Error != ErrNonMatchingLexicon {
				t.Errorf("expected error %v, got %v", ErrNonMatchingLexicon, result.Error)
			}
		})
	}

	// Verify metrics: all processed, none matched, none discarded
	if metrics.Processed() != int64(len(tests)) {
		t.Errorf("Processed() = %d, want %d", metrics.Processed(), len(tests))
	}
	if metrics.Matched() != 0 {
		t.Errorf("Matched() = %d, want 0", metrics.Matched())
	}
	if metrics.Discarded() != 0 {
		t.Errorf("Discarded() = %d, want 0", metrics.Discarded())
	}
}

func TestRecordFilter_Filter_ValidSceneRecord(t *testing.T) {
	metrics := NewFilterMetrics()
	filter := NewRecordFilter(metrics)

	tests := []struct {
		name    string
		payload string
	}{
		{
			name:    "minimal scene with name only",
			payload: `{"name":"Underground Techno"}`,
		},
		{
			name:    "scene with name and description",
			payload: `{"name":"Warehouse Rave","description":"Monthly warehouse events"}`,
		},
		{
			name:    "scene with extra fields",
			payload: `{"name":"Test Scene","description":"Test","extra":"ignored"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filter.Filter(CollectionScene, []byte(tt.payload))

			if !result.Matched {
				t.Error("expected Matched = true")
			}
			if !result.Valid {
				t.Errorf("expected Valid = true, got error: %v", result.Error)
			}
			if result.Collection != CollectionScene {
				t.Errorf("Collection = %s, want %s", result.Collection, CollectionScene)
			}
			if result.Record == nil {
				t.Error("expected Record to be non-nil")
			}
		})
	}
}

func TestRecordFilter_Filter_InvalidSceneRecord(t *testing.T) {
	metrics := NewFilterMetrics()
	filter := NewRecordFilter(metrics)

	tests := []struct {
		name        string
		payload     string
		expectedErr error
	}{
		{
			name:        "missing name field",
			payload:     `{"description":"no name"}`,
			expectedErr: ErrMissingField,
		},
		{
			name:        "name is not a string",
			payload:     `{"name":123}`,
			expectedErr: ErrInvalidFieldType,
		},
		{
			name:        "name is null",
			payload:     `{"name":null}`,
			expectedErr: ErrInvalidFieldType,
		},
		{
			name:        "name is array",
			payload:     `{"name":["foo","bar"]}`,
			expectedErr: ErrInvalidFieldType,
		},
		{
			name:        "malformed JSON",
			payload:     `{"name":"test"`,
			expectedErr: ErrMalformedJSON,
		},
		{
			name:        "empty JSON",
			payload:     `{}`,
			expectedErr: ErrMissingField,
		},
		{
			name:        "not JSON",
			payload:     `not json at all`,
			expectedErr: ErrMalformedJSON,
		},
	}

	initialDiscarded := metrics.Discarded()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filter.Filter(CollectionScene, []byte(tt.payload))

			if !result.Matched {
				t.Error("expected Matched = true for matching collection")
			}
			if result.Valid {
				t.Error("expected Valid = false for invalid record")
			}
			if result.Error != tt.expectedErr {
				t.Errorf("expected error %v, got %v", tt.expectedErr, result.Error)
			}
		})
	}

	// Verify discarded count increased
	expectedDiscarded := initialDiscarded + int64(len(tests))
	if metrics.Discarded() != expectedDiscarded {
		t.Errorf("Discarded() = %d, want %d", metrics.Discarded(), expectedDiscarded)
	}
}

func TestRecordFilter_Filter_ValidEventRecord(t *testing.T) {
	metrics := NewFilterMetrics()
	filter := NewRecordFilter(metrics)

	tests := []struct {
		name    string
		payload string
	}{
		{
			name:    "minimal event",
			payload: `{"name":"Friday Night","sceneId":"scene123"}`,
		},
		{
			name:    "event with extra fields",
			payload: `{"name":"Saturday Rave","sceneId":"scene456","startTime":"2024-01-01T20:00:00Z"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filter.Filter(CollectionEvent, []byte(tt.payload))

			if !result.Matched {
				t.Error("expected Matched = true")
			}
			if !result.Valid {
				t.Errorf("expected Valid = true, got error: %v", result.Error)
			}
			if result.Collection != CollectionEvent {
				t.Errorf("Collection = %s, want %s", result.Collection, CollectionEvent)
			}
		})
	}
}

func TestRecordFilter_Filter_InvalidEventRecord(t *testing.T) {
	metrics := NewFilterMetrics()
	filter := NewRecordFilter(metrics)

	tests := []struct {
		name        string
		payload     string
		expectedErr error
	}{
		{
			name:        "missing name",
			payload:     `{"sceneId":"scene123"}`,
			expectedErr: ErrMissingField,
		},
		{
			name:        "missing sceneId",
			payload:     `{"name":"Event Name"}`,
			expectedErr: ErrMissingField,
		},
		{
			name:        "name not string",
			payload:     `{"name":true,"sceneId":"scene123"}`,
			expectedErr: ErrInvalidFieldType,
		},
		{
			name:        "sceneId not string",
			payload:     `{"name":"Event","sceneId":123}`,
			expectedErr: ErrInvalidFieldType,
		},
		{
			name:        "malformed JSON",
			payload:     `{"name":"Event","sceneId":}`,
			expectedErr: ErrMalformedJSON,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filter.Filter(CollectionEvent, []byte(tt.payload))

			if !result.Matched {
				t.Error("expected Matched = true for matching collection")
			}
			if result.Valid {
				t.Error("expected Valid = false for invalid record")
			}
			if result.Error != tt.expectedErr {
				t.Errorf("expected error %v, got %v", tt.expectedErr, result.Error)
			}
		})
	}
}

func TestRecordFilter_Filter_ValidPostRecord(t *testing.T) {
	metrics := NewFilterMetrics()
	filter := NewRecordFilter(metrics)

	tests := []struct {
		name    string
		payload string
	}{
		{
			name:    "minimal post with sceneId",
			payload: `{"text":"Check out this venue!","sceneId":"scene123"}`,
		},
		{
			name:    "post with eventId only",
			payload: `{"text":"Amazing performance!","eventId":"event789"}`,
		},
		{
			name:    "post with both sceneId and eventId",
			payload: `{"text":"Can't wait!","sceneId":"scene456","eventId":"event123"}`,
		},
		{
			name:    "post with extra fields",
			payload: `{"text":"Great show tonight","sceneId":"scene456","embed":{"uri":"at://..."}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filter.Filter(CollectionPost, []byte(tt.payload))

			if !result.Matched {
				t.Error("expected Matched = true")
			}
			if !result.Valid {
				t.Errorf("expected Valid = true, got error: %v", result.Error)
			}
			if result.Collection != CollectionPost {
				t.Errorf("Collection = %s, want %s", result.Collection, CollectionPost)
			}
		})
	}
}

func TestRecordFilter_Filter_InvalidPostRecord(t *testing.T) {
	metrics := NewFilterMetrics()
	filter := NewRecordFilter(metrics)

	tests := []struct {
		name        string
		payload     string
		expectedErr error
	}{
		{
			name:        "missing text",
			payload:     `{"sceneId":"scene123"}`,
			expectedErr: ErrMissingField,
		},
		{
			name:        "missing both sceneId and eventId",
			payload:     `{"text":"Hello world"}`,
			expectedErr: ErrMissingField,
		},
		{
			name:        "text not string",
			payload:     `{"text":123,"sceneId":"scene123"}`,
			expectedErr: ErrInvalidFieldType,
		},
		{
			name:        "sceneId not string (null)",
			payload:     `{"text":"Hello","sceneId":null}`,
			expectedErr: ErrMissingField,
		},
		{
			name:        "eventId not string (null)",
			payload:     `{"text":"Hello","eventId":null}`,
			expectedErr: ErrMissingField,
		},
		{
			name:        "both references are null",
			payload:     `{"text":"Hello","sceneId":null,"eventId":null}`,
			expectedErr: ErrMissingField,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filter.Filter(CollectionPost, []byte(tt.payload))

			if !result.Matched {
				t.Error("expected Matched = true for matching collection")
			}
			if result.Valid {
				t.Error("expected Valid = false for invalid record")
			}
			if result.Error != tt.expectedErr {
				t.Errorf("expected error %v, got %v", tt.expectedErr, result.Error)
			}
		})
	}
}

func TestRecordFilter_Filter_UnknownSubcultCollection(t *testing.T) {
	metrics := NewFilterMetrics()
	filter := NewRecordFilter(metrics)

	// Unknown app.subcult.* collections should still match but only validate JSON syntax
	tests := []struct {
		name       string
		collection string
		payload    string
		wantValid  bool
	}{
		{
			name:       "unknown collection with valid JSON",
			collection: "app.subcult.unknown",
			payload:    `{"any":"field"}`,
			wantValid:  true,
		},
		{
			name:       "app.subcult.alliance with valid JSON",
			collection: "app.subcult.alliance",
			payload:    `{"fromSceneId":"scene1","toSceneId":"scene2"}`,
			wantValid:  true,
		},
		{
			name:       "unknown collection with invalid JSON",
			collection: "app.subcult.future",
			payload:    `not json`,
			wantValid:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filter.Filter(tt.collection, []byte(tt.payload))

			if !result.Matched {
				t.Error("expected Matched = true for app.subcult.* collection")
			}
			if result.Valid != tt.wantValid {
				t.Errorf("Valid = %v, want %v; error: %v", result.Valid, tt.wantValid, result.Error)
			}
		})
	}
}

func TestRecordFilter_Filter_MixedBatch(t *testing.T) {
	metrics := NewFilterMetrics()
	filter := NewRecordFilter(metrics)

	// Simulate a batch of mixed records as would come from Jetstream
	batch := []struct {
		collection string
		payload    string
	}{
		// Non-matching records (should be ignored)
		{"app.bsky.feed.post", `{"text":"Hello Bluesky"}`},
		{"app.bsky.feed.like", `{"subject":{"uri":"at://..."}}`},
		{"app.bsky.actor.profile", `{"displayName":"User"}`},

		// Matching valid records
		{"app.subcult.scene", `{"name":"Test Scene"}`},
		{"app.subcult.event", `{"name":"Test Event","sceneId":"s1"}`},
		{"app.subcult.post", `{"text":"Test post","sceneId":"s1"}`},

		// Matching invalid records (should be discarded)
		{"app.subcult.scene", `{"description":"missing name"}`},
		{"app.subcult.event", `invalid json`},
	}

	var matchedValid, matchedInvalid, nonMatching int
	for _, item := range batch {
		result := filter.Filter(item.collection, []byte(item.payload))
		if !result.Matched {
			nonMatching++
		} else if result.Valid {
			matchedValid++
		} else {
			matchedInvalid++
		}
	}

	if nonMatching != 3 {
		t.Errorf("nonMatching = %d, want 3", nonMatching)
	}
	if matchedValid != 3 {
		t.Errorf("matchedValid = %d, want 3", matchedValid)
	}
	if matchedInvalid != 2 {
		t.Errorf("matchedInvalid = %d, want 2", matchedInvalid)
	}

	// Verify metrics
	if metrics.Processed() != int64(len(batch)) {
		t.Errorf("Processed() = %d, want %d", metrics.Processed(), len(batch))
	}
	if metrics.Matched() != 5 {
		t.Errorf("Matched() = %d, want 5", metrics.Matched())
	}
	if metrics.Discarded() != 2 {
		t.Errorf("Discarded() = %d, want 2", metrics.Discarded())
	}
}

func TestFilterMetrics_Concurrency(t *testing.T) {
	metrics := NewFilterMetrics()
	filter := NewRecordFilter(metrics)

	done := make(chan bool)

	// Run concurrent filters
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				filter.Filter("app.bsky.feed.post", []byte(`{"text":"test"}`))
				filter.Filter("app.subcult.scene", []byte(`{"name":"test"}`))
				filter.Filter("app.subcult.scene", []byte(`{}`)) // invalid
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify counts
	expectedProcessed := int64(10 * 100 * 3) // 10 goroutines * 100 iterations * 3 calls each
	if metrics.Processed() != expectedProcessed {
		t.Errorf("Processed() = %d, want %d", metrics.Processed(), expectedProcessed)
	}

	expectedMatched := int64(10 * 100 * 2) // 2 app.subcult.* calls per iteration
	if metrics.Matched() != expectedMatched {
		t.Errorf("Matched() = %d, want %d", metrics.Matched(), expectedMatched)
	}

	expectedDiscarded := int64(10 * 100) // 1 invalid record per iteration
	if metrics.Discarded() != expectedDiscarded {
		t.Errorf("Discarded() = %d, want %d", metrics.Discarded(), expectedDiscarded)
	}
}

func TestValidateJSONSyntax(t *testing.T) {
	tests := []struct {
		name    string
		payload string
		wantErr error
	}{
		{
			name:    "valid object",
			payload: `{"key":"value"}`,
			wantErr: nil,
		},
		{
			name:    "valid array",
			payload: `[1,2,3]`,
			wantErr: nil,
		},
		{
			name:    "valid string",
			payload: `"hello"`,
			wantErr: nil,
		},
		{
			name:    "valid number",
			payload: `123`,
			wantErr: nil,
		},
		{
			name:    "valid null",
			payload: `null`,
			wantErr: nil,
		},
		{
			name:    "invalid - unclosed brace",
			payload: `{"key":"value"`,
			wantErr: ErrMalformedJSON,
		},
		{
			name:    "invalid - not JSON",
			payload: `hello world`,
			wantErr: ErrMalformedJSON,
		},
		{
			name:    "empty string",
			payload: ``,
			wantErr: ErrMalformedJSON,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateJSONSyntax([]byte(tt.payload))
			if err != tt.wantErr {
				t.Errorf("validateJSONSyntax() = %v, want %v", err, tt.wantErr)
			}
		})
	}
}
