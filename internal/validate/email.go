package validate

import (
	"errors"
	"regexp"
	"strings"
)

// Email validation errors
var (
	ErrInvalidEmail = errors.New("invalid email format")
)

// emailPattern is a reasonable regex for basic email validation.
// For production use, this matches most common email formats.
// More strict validation happens at the SMTP level.
var emailPattern = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

// Email validates an email address format.
// Returns the normalized (lowercased, trimmed) email and an error if invalid.
func Email(email string) (string, error) {
	// Trim and lowercase
	email = strings.ToLower(strings.TrimSpace(email))

	// Check if empty
	if email == "" {
		return "", ErrEmpty
	}

	// Check length constraints (RFC 5321)
	if len(email) > 254 {
		return "", ErrStringTooLong
	}

	// Validate format
	if !emailPattern.MatchString(email) {
		return "", ErrInvalidEmail
	}

	// Additional checks
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return "", ErrInvalidEmail
	}

	localPart, domain := parts[0], parts[1]

	// Local part should not exceed 64 characters (RFC 5321)
	if len(localPart) > 64 {
		return "", ErrStringTooLong
	}

	// Domain should not exceed 255 characters (RFC 5321)
	if len(domain) > 255 {
		return "", ErrStringTooLong
	}

	// Domain should have at least one dot
	if !strings.Contains(domain, ".") {
		return "", ErrInvalidEmail
	}

	return email, nil
}
