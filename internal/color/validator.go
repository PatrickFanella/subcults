// Package color provides color validation utilities including hex format validation
// and WCAG AA contrast ratio checking.
package color

import (
	"errors"
	"fmt"
	"html"
	"math"
	"regexp"
	"strconv"
	"strings"
)

// hexColorPattern matches valid hex color codes in format #RRGGBB (case insensitive).
var hexColorPattern = regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)

// Common validation errors
var (
	ErrInvalidHexFormat     = errors.New("invalid hex color format, expected #RRGGBB")
	ErrInsufficientContrast = errors.New("insufficient contrast ratio, minimum 4.5:1 required for WCAG AA")
)

// IsValidHexColor validates that a color string is in valid #RRGGBB format.
func IsValidHexColor(color string) bool {
	return hexColorPattern.MatchString(color)
}

// SanitizeColor sanitizes a color string to prevent script injection.
// Returns the original color if valid, or empty string if invalid.
func SanitizeColor(color string) string {
	// HTML escape to prevent any script injection
	sanitized := html.EscapeString(strings.TrimSpace(color))

	// Verify it's still a valid hex color after sanitization
	if !IsValidHexColor(sanitized) {
		return ""
	}

	return sanitized
}

// ValidateHexColor validates a hex color and returns an error if invalid.
func ValidateHexColor(color string) error {
	if !IsValidHexColor(color) {
		return fmt.Errorf("%w: got %q", ErrInvalidHexFormat, color)
	}
	return nil
}

// RGB represents a color in RGB color space with values 0-255.
type RGB struct {
	R, G, B uint8
}

// ParseHexColor parses a hex color string (#RRGGBB) into RGB components.
// Returns an error if the color is not in valid hex format.
func ParseHexColor(hexColor string) (RGB, error) {
	if !IsValidHexColor(hexColor) {
		return RGB{}, ErrInvalidHexFormat
	}

	// Remove the # prefix
	hexColor = strings.TrimPrefix(hexColor, "#")

	// Parse each component
	r, err := strconv.ParseUint(hexColor[0:2], 16, 8)
	if err != nil {
		return RGB{}, fmt.Errorf("failed to parse red component: %w", err)
	}

	g, err := strconv.ParseUint(hexColor[2:4], 16, 8)
	if err != nil {
		return RGB{}, fmt.Errorf("failed to parse green component: %w", err)
	}

	b, err := strconv.ParseUint(hexColor[4:6], 16, 8)
	if err != nil {
		return RGB{}, fmt.Errorf("failed to parse blue component: %w", err)
	}

	return RGB{R: uint8(r), G: uint8(g), B: uint8(b)}, nil
}

// relativeLuminance calculates the relative luminance of an RGB color
// according to WCAG 2.1 specification.
// https://www.w3.org/WAI/GL/wiki/Relative_luminance
func relativeLuminance(rgb RGB) float64 {
	// Convert to sRGB values (0.0 - 1.0)
	rsRGB := float64(rgb.R) / 255.0
	gsRGB := float64(rgb.G) / 255.0
	bsRGB := float64(rgb.B) / 255.0

	// Apply gamma correction
	var r, g, b float64
	if rsRGB <= 0.03928 {
		r = rsRGB / 12.92
	} else {
		r = math.Pow((rsRGB+0.055)/1.055, 2.4)
	}

	if gsRGB <= 0.03928 {
		g = gsRGB / 12.92
	} else {
		g = math.Pow((gsRGB+0.055)/1.055, 2.4)
	}

	if bsRGB <= 0.03928 {
		b = bsRGB / 12.92
	} else {
		b = math.Pow((bsRGB+0.055)/1.055, 2.4)
	}

	// Calculate luminance using ITU-R BT.709 coefficients
	return 0.2126*r + 0.7152*g + 0.0722*b
}

// ContrastRatio calculates the contrast ratio between two colors.
// Returns a value between 1.0 (no contrast) and 21.0 (maximum contrast).
// Based on WCAG 2.1 contrast ratio formula.
// https://www.w3.org/WAI/GL/wiki/Contrast_ratio
func ContrastRatio(color1, color2 RGB) float64 {
	l1 := relativeLuminance(color1)
	l2 := relativeLuminance(color2)

	// Ensure l1 is the lighter color
	if l1 < l2 {
		l1, l2 = l2, l1
	}

	// Calculate contrast ratio
	return (l1 + 0.05) / (l2 + 0.05)
}

// ValidateContrast validates that two hex colors have sufficient contrast
// for WCAG AA compliance (minimum 4.5:1 ratio).
// Returns the calculated ratio and an error if insufficient.
func ValidateContrast(textColor, bgColor string) (float64, error) {
	textRGB, err := ParseHexColor(textColor)
	if err != nil {
		return 0, fmt.Errorf("invalid text color: %w", err)
	}

	bgRGB, err := ParseHexColor(bgColor)
	if err != nil {
		return 0, fmt.Errorf("invalid background color: %w", err)
	}

	ratio := ContrastRatio(textRGB, bgRGB)

	// WCAG AA requires 4.5:1 for normal text
	if ratio < 4.5 {
		return ratio, fmt.Errorf("%w: got %.2f:1", ErrInsufficientContrast, ratio)
	}

	return ratio, nil
}
