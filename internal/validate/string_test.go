package validate

import (
	"errors"
	"regexp"
	"strings"
	"testing"
)

func TestString(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		constraints StringConstraints
		wantErr     error
		wantOutput  string
	}{
		{
			name:  "valid string within length constraints",
			input: "Hello World",
			constraints: StringConstraints{
				MinLength: 5,
				MaxLength: 20,
				TrimSpace: true,
			},
			wantErr:    nil,
			wantOutput: "Hello World",
		},
		{
			name:  "string too short",
			input: "Hi",
			constraints: StringConstraints{
				MinLength: 5,
				MaxLength: 20,
			},
			wantErr: ErrStringTooShort,
		},
		{
			name:  "string too long",
			input: strings.Repeat("a", 101),
			constraints: StringConstraints{
				MinLength: 1,
				MaxLength: 100,
			},
			wantErr: ErrStringTooLong,
		},
		{
			name:  "empty string not allowed",
			input: "",
			constraints: StringConstraints{
				AllowEmpty: false,
			},
			wantErr: ErrEmpty,
		},
		{
			name:  "empty string allowed",
			input: "",
			constraints: StringConstraints{
				AllowEmpty: true,
			},
			wantErr:    nil,
			wantOutput: "",
		},
		{
			name:  "whitespace trimmed",
			input: "  Hello  ",
			constraints: StringConstraints{
				TrimSpace: true,
			},
			wantErr:    nil,
			wantOutput: "Hello",
		},
		{
			name:  "SQL keyword detected",
			input: "Hello SELECT World",
			constraints: StringConstraints{
				CheckSQLKeywords: true,
			},
			wantErr: ErrSQLKeyword,
		},
		{
			name:  "SQL keyword in lowercase",
			input: "select * from users",
			constraints: StringConstraints{
				CheckSQLKeywords: true,
			},
			wantErr: ErrSQLKeyword,
		},
		{
			name:  "no SQL keyword",
			input: "This is a normal sentence",
			constraints: StringConstraints{
				CheckSQLKeywords: true,
			},
			wantErr:    nil,
			wantOutput: "This is a normal sentence",
		},
		{
			name:  "disallowed word detected",
			input: "Hello spam world",
			constraints: StringConstraints{
				DisallowedWords: []string{"spam", "scam"},
			},
			wantErr: errors.New("disallowed word"),
		},
		{
			name:  "pattern validation success",
			input: "valid-name_123",
			constraints: StringConstraints{
				AllowedPattern: mustCompile(`^[a-zA-Z0-9_\-]+$`),
			},
			wantErr:    nil,
			wantOutput: "valid-name_123",
		},
		{
			name:  "pattern validation failure",
			input: "invalid name!",
			constraints: StringConstraints{
				AllowedPattern: mustCompile(`^[a-zA-Z0-9_\-]+$`),
			},
			wantErr: ErrInvalidCharacters,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := String(tt.input, tt.constraints)
			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("String() error = nil, wantErr %v", tt.wantErr)
					return
				}
				if !errors.Is(err, tt.wantErr) && !strings.Contains(err.Error(), "disallowed word") {
					t.Errorf("String() error = %v, wantErr %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Errorf("String() unexpected error = %v", err)
				return
			}
			if got != tt.wantOutput {
				t.Errorf("String() = %q, want %q", got, tt.wantOutput)
			}
		})
	}
}

func TestSanitizeHTML(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "plain text unchanged",
			input: "Hello World",
			want:  "Hello World",
		},
		{
			name:  "script tag escaped",
			input: "<script>alert('xss')</script>",
			want:  "&lt;script&gt;alert(&#39;xss&#39;)&lt;/script&gt;",
		},
		{
			name:  "HTML entities escaped",
			input: `<div onclick="evil()">Click me</div>`,
			want:  "&lt;div onclick=&#34;evil()&#34;&gt;Click me&lt;/div&gt;",
		},
		{
			name:  "ampersand escaped",
			input: "Tom & Jerry",
			want:  "Tom &amp; Jerry",
		},
		{
			name:  "quotes escaped",
			input: `He said "hello"`,
			want:  "He said &#34;hello&#34;",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SanitizeHTML(tt.input)
			if got != tt.want {
				t.Errorf("SanitizeHTML() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSceneName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "valid scene name",
			input:   "My Cool Scene",
			wantErr: false,
		},
		{
			name:    "scene name with allowed characters",
			input:   "Scene-Name_v2.0",
			wantErr: false,
		},
		{
			name:    "scene name too short",
			input:   "",
			wantErr: true,
		},
		{
			name:    "scene name too long",
			input:   strings.Repeat("a", 65),
			wantErr: true,
		},
		{
			name:    "scene name with special characters",
			input:   "Scene@Name#123",
			wantErr: true,
		},
		{
			name:    "single character too short",
			input:   "X",
			wantErr: true,
		},
		{
			name:    "two characters too short",
			input:   "Ab",
			wantErr: true,
		},
		{
			name:    "three characters minimum",
			input:   "Abc",
			wantErr: false,
		},
		{
			name:    "DROP TABLE scenes - now allowed",
			input:   "DROP TABLE scenes",
			wantErr: false, // SQL keywords disabled for scene names
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SceneName(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("SceneName() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got == "" {
				t.Errorf("SceneName() returned empty string for valid input")
			}
		})
	}
}

func TestEventTitle(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "valid event title",
			input:   "Summer Music Festival 2024",
			wantErr: false,
		},
		{
			name:    "event title at max length",
			input:   strings.Repeat("a", 80),
			wantErr: false,
		},
		{
			name:    "event title too long",
			input:   strings.Repeat("a", 81),
			wantErr: true,
		},
		{
			name:    "empty event title",
			input:   "",
			wantErr: true,
		},
		{
			name:    "Concert with SQL pattern - now allowed",
			input:   "Concert; DROP TABLE events--",
			wantErr: false, // SQL keywords disabled for event titles
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := EventTitle(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("EventTitle() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got == "" {
				t.Errorf("EventTitle() returned empty string for valid input")
			}
		})
	}
}

func TestPostContent(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "valid post content",
			input:   "This is a great post about music!",
			wantErr: false,
		},
		{
			name:    "post content at max length",
			input:   strings.Repeat("a", 5000),
			wantErr: false,
		},
		{
			name:    "post content too long",
			input:   strings.Repeat("a", 5001),
			wantErr: true,
		},
		{
			name:    "empty post content",
			input:   "",
			wantErr: true,
		},
		{
			name:    "post content with HTML",
			input:   "Check out <b>this</b> cool thing!",
			wantErr: false, // Should not error, but HTML will be escaped
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := PostContent(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("PostContent() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if got == "" {
					t.Errorf("PostContent() returned empty string for valid input")
				}
				// Verify HTML is escaped
				if strings.Contains(tt.input, "<") && !strings.Contains(got, "&lt;") {
					t.Errorf("PostContent() did not escape HTML: got %q", got)
				}
			}
		})
	}
}

func TestDescription(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "valid description",
			input:   "This is a description.",
			wantErr: false,
		},
		{
			name:    "empty description allowed",
			input:   "",
			wantErr: false,
		},
		{
			name:    "description too long",
			input:   strings.Repeat("a", 5001),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Description(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Description() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestSQLKeywordWordBoundary tests that SQL keyword detection uses word boundaries
// to avoid false positives with legitimate names containing SQL keywords as substrings.
// Note: SQL keyword checking is now disabled for scene names and event titles to avoid
// frustrating users with legitimate venue/event names. The improved checkSQLKeywords
// function with word boundary detection is available for other use cases.
func TestSQLKeywordWordBoundary(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		// All of these should pass now since SQL keyword checking is disabled for scene names
		{
			name:    "Drop Zone Music Hall",
			input:   "Drop Zone Music Hall",
			wantErr: false,
		},
		{
			name:    "The Executive Lounge",
			input:   "The Executive Lounge",
			wantErr: false,
		},
		{
			name:    "From the Underground",
			input:   "From the Underground",
			wantErr: false,
		},
		{
			name:    "Join Together Festival",
			input:   "Join Together Festival",
			wantErr: false,
		},
		{
			name:    "Select Sounds Collective",
			input:   "Select Sounds Collective",
			wantErr: false,
		},
		{
			name:    "DELETE this scene",
			input:   "DELETE this scene",
			wantErr: false,
		},
		{
			name:    "DROP the beat",
			input:   "DROP the beat",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := SceneName(tt.input)
			hasErr := err != nil
			if hasErr != tt.wantErr {
				t.Errorf("SceneName(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

// TestSQLKeywordDetectionWithConstraints tests the SQL keyword detection directly
// with the CheckSQLKeywords constraint enabled, demonstrating the word boundary logic.
func TestSQLKeywordDetectionWithConstraints(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		// Should NOT trigger (legitimate names with SQL keywords as substrings)
		{
			name:    "Executive contains EXEC",
			input:   "The Executive",
			wantErr: false,
		},
		
		// Should trigger (actual SQL keywords as standalone words)
		{
			name:    "standalone SELECT",
			input:   "SELECT something",
			wantErr: true,
		},
		{
			name:    "standalone DELETE",
			input:   "DELETE this",
			wantErr: true,
		},
		{
			name:    "standalone DROP",
			input:   "DROP it",
			wantErr: true,
		},
		{
			name:    "SQL comment pattern",
			input:   "test -- comment",
			wantErr: true,
		},
		{
			name:    "stored procedure prefix",
			input:   "xp_cmdshell test",
			wantErr: true,
		},
	}

	constraints := StringConstraints{
		MinLength:        1,
		MaxLength:        100,
		CheckSQLKeywords: true,
		TrimSpace:        true,
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := String(tt.input, constraints)
			hasErr := err != nil
			if hasErr != tt.wantErr {
				t.Errorf("String(%q) with SQL keyword check error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

// Helper function for tests
func mustCompile(pattern string) *regexp.Regexp {
	return regexp.MustCompile(pattern)
}
