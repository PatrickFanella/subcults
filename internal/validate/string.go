// Package validate provides centralized input validation and sanitization utilities
// for the Subcults API. It includes protection against SQL injection, XSS, SSRF,
// and other common web vulnerabilities.
package validate

import (
	"errors"
	"fmt"
	"html"
	"regexp"
	"strings"
	"unicode/utf8"
)

// String validation errors
var (
	ErrStringTooShort    = errors.New("string is too short")
	ErrStringTooLong     = errors.New("string is too long")
	ErrInvalidCharacters = errors.New("string contains invalid characters")
	ErrSQLKeyword        = errors.New("string contains SQL keywords")
	ErrEmpty             = errors.New("string is empty")
)

// sceneNamePattern is a precompiled regex for allowed scene name characters.
var sceneNamePattern = regexp.MustCompile(`^[A-Za-z0-9 _\-\.]+$`)

// StringConstraints defines validation constraints for a string.
type StringConstraints struct {
	MinLength        int              // Minimum length (0 = no minimum)
	MaxLength        int              // Maximum length (0 = no maximum)
	AllowedPattern   *regexp.Regexp   // Optional regex pattern for allowed characters
	DisallowedWords  []string         // Optional list of disallowed words (case-insensitive)
	CheckSQLKeywords bool             // Whether to check for SQL keywords
	AllowEmpty       bool             // Whether empty strings are allowed
	TrimSpace        bool             // Whether to trim whitespace before validation
}

// String validates a string against the given constraints.
// Returns the validated (and optionally trimmed) string and an error if validation fails.
func String(s string, constraints StringConstraints) (string, error) {
	// Optionally trim whitespace
	if constraints.TrimSpace {
		s = strings.TrimSpace(s)
	}

	// Check if empty
	if s == "" {
		if !constraints.AllowEmpty {
			return "", ErrEmpty
		}
		return s, nil
	}

	// Get actual character count (not byte count)
	length := utf8.RuneCountInString(s)

	// Check minimum length
	if constraints.MinLength > 0 && length < constraints.MinLength {
		return "", fmt.Errorf("%w: got %d chars, need at least %d", ErrStringTooShort, length, constraints.MinLength)
	}

	// Check maximum length
	if constraints.MaxLength > 0 && length > constraints.MaxLength {
		return "", fmt.Errorf("%w: got %d chars, maximum is %d", ErrStringTooLong, length, constraints.MaxLength)
	}

	// Check allowed pattern
	if constraints.AllowedPattern != nil && !constraints.AllowedPattern.MatchString(s) {
		return "", fmt.Errorf("%w: does not match required pattern", ErrInvalidCharacters)
	}

	// Check SQL keywords if enabled
	if constraints.CheckSQLKeywords {
		if err := checkSQLKeywords(s); err != nil {
			return "", err
		}
	}

	// Check disallowed words
	if len(constraints.DisallowedWords) > 0 {
		upper := strings.ToUpper(s)
		for _, word := range constraints.DisallowedWords {
			if strings.Contains(upper, strings.ToUpper(word)) {
				return "", fmt.Errorf("string contains disallowed word: %q", word)
			}
		}
	}

	return s, nil
}

// checkSQLKeywords checks if the string contains common SQL keywords as standalone words.
// This is a basic heuristic check; parameterized queries are the real defense.
// Uses word boundary detection to avoid false positives (e.g., "Drop Zone" vs "DROP TABLE").
func checkSQLKeywords(s string) error {
	upper := strings.ToUpper(s)

	// Check for SQL comment patterns (these are always suspicious)
	commentPatterns := []string{"--", "/*", "*/", ";--"}
	for _, pattern := range commentPatterns {
		if strings.Contains(upper, pattern) {
			return fmt.Errorf("%w: contains %q", ErrSQLKeyword, pattern)
		}
	}

	// Check for SQL keywords with word boundary detection
	// Split into words to avoid false positives like "Drop Zone" or "Executive"
	words := strings.FieldsFunc(upper, func(r rune) bool {
		// Split on non-alphanumeric characters
		return !((r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_')
	})

	sqlKeywordSet := map[string]bool{
		"SELECT": true, "INSERT": true, "UPDATE": true, "DELETE": true,
		"DROP": true, "CREATE": true, "ALTER": true, "TRUNCATE": true,
		"EXEC": true, "EXECUTE": true, "UNION": true,
		// Removed common words: JOIN, WHERE, FROM (too many false positives)
		// These are less likely to indicate SQL injection in user-facing fields
	}

	for _, word := range words {
		if sqlKeywordSet[word] {
			return fmt.Errorf("%w: contains SQL keyword %q", ErrSQLKeyword, word)
		}
	}

	// Check for stored procedure prefixes
	if strings.Contains(upper, "XP_") || strings.Contains(upper, "SP_") {
		return fmt.Errorf("%w: contains stored procedure prefix", ErrSQLKeyword)
	}

	return nil
}

// SanitizeHTML escapes HTML special characters to prevent XSS attacks.
// This should be called on all user-generated text that will be displayed in HTML.
func SanitizeHTML(s string) string {
	return html.EscapeString(s)
}

// SanitizeString performs both validation and HTML sanitization.
// Returns the sanitized string and an error if validation fails.
func SanitizeString(s string, constraints StringConstraints) (string, error) {
	validated, err := String(s, constraints)
	if err != nil {
		return "", err
	}
	return SanitizeHTML(validated), nil
}

// SceneName validates a scene name according to Subcults requirements:
// - 3-64 characters
// - Letters, numbers, spaces, dash, underscore, period only
// - HTML sanitized
// Note: SQL keyword checking is disabled for scene names to avoid false positives
// with legitimate venue names like "Drop Zone" or "The Executive Lounge".
// Parameterized queries provide the primary SQL injection defense.
func SceneName(name string) (string, error) {
	return SanitizeString(name, StringConstraints{
		MinLength:        3,
		MaxLength:        64,
		AllowedPattern:   sceneNamePattern,
		CheckSQLKeywords: false, // Disabled to avoid false positives
		AllowEmpty:       false,
		TrimSpace:        true,
	})
}

// EventTitle validates an event title according to Subcults requirements:
// - 3-80 characters
// - HTML sanitized
// Note: SQL keyword checking is disabled for event titles to avoid false positives
// with legitimate event names. Parameterized queries provide the primary SQL injection defense.
func EventTitle(title string) (string, error) {
	return SanitizeString(title, StringConstraints{
		MinLength:        3,
		MaxLength:        80,
		CheckSQLKeywords: false, // Disabled to avoid false positives
		AllowEmpty:       false,
		TrimSpace:        true,
	})
}

// PostContent validates post content according to Subcults requirements:
// - Required (not empty)
// - Max 5000 characters
func PostContent(content string) (string, error) {
	return SanitizeString(content, StringConstraints{
		MinLength:        1,
		MaxLength:        5000,
		CheckSQLKeywords: false, // Allow more freedom in post content
		AllowEmpty:       false,
		TrimSpace:        true,
	})
}

// Description validates a description field:
// - Optional (can be empty)
// - Max 5000 characters
func Description(desc string) (string, error) {
	return SanitizeString(desc, StringConstraints{
		MinLength:        0,
		MaxLength:        5000,
		CheckSQLKeywords: false, // Allow more freedom in descriptions
		AllowEmpty:       true,
		TrimSpace:        true,
	})
}
