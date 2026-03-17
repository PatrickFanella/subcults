package validate

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"
)

// URL validation errors
var (
	ErrInvalidURL       = errors.New("invalid URL format")
	ErrDisallowedScheme = errors.New("URL scheme not allowed")
	ErrDisallowedDomain = errors.New("URL domain not allowed")
	ErrSSRFRisk         = errors.New("URL poses SSRF risk")
)

// URLConstraints defines validation constraints for URLs.
type URLConstraints struct {
	AllowedSchemes []string // e.g., []string{"https", "http"}
	AllowedDomains []string // If non-empty, only these domains are allowed
	BlockPrivate   bool     // Whether to block private/local IP addresses (SSRF protection)
	MaxLength      int      // Maximum URL length (0 = no limit)
}

// DefaultURLConstraints provides secure defaults for URL validation.
// Blocks private IPs and only allows HTTPS.
var DefaultURLConstraints = URLConstraints{
	AllowedSchemes: []string{"https"},
	AllowedDomains: nil, // Empty means all public domains allowed
	BlockPrivate:   true,
	MaxLength:      2048,
}

// PublicWebURLConstraints allows both HTTP and HTTPS for public web URLs.
var PublicWebURLConstraints = URLConstraints{
	AllowedSchemes: []string{"https", "http"},
	AllowedDomains: nil,
	BlockPrivate:   true,
	MaxLength:      2048,
}

// URL validates a URL against the given constraints.
// Returns the validated URL string and an error if validation fails.
func URL(urlStr string, constraints URLConstraints) (string, error) {
	// Trim whitespace
	urlStr = strings.TrimSpace(urlStr)

	// Check if empty
	if urlStr == "" {
		return "", ErrEmpty
	}

	// Check length
	if constraints.MaxLength > 0 && len(urlStr) > constraints.MaxLength {
		return "", fmt.Errorf("%w: URL exceeds %d characters", ErrStringTooLong, constraints.MaxLength)
	}

	// Parse URL
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrInvalidURL, err)
	}

	// Validate scheme
	if len(constraints.AllowedSchemes) > 0 {
		schemeAllowed := false
		for _, scheme := range constraints.AllowedSchemes {
			if parsedURL.Scheme == scheme {
				schemeAllowed = true
				break
			}
		}
		if !schemeAllowed {
			return "", fmt.Errorf("%w: got %q, allowed: %v", ErrDisallowedScheme, parsedURL.Scheme, constraints.AllowedSchemes)
		}
	}

	// Validate hostname
	hostname := parsedURL.Hostname()
	if hostname == "" {
		return "", fmt.Errorf("%w: missing hostname", ErrInvalidURL)
	}

	// Check domain allowlist if specified
	if len(constraints.AllowedDomains) > 0 {
		domainAllowed := false
		for _, domain := range constraints.AllowedDomains {
			// Check exact match or subdomain
			if hostname == domain || strings.HasSuffix(hostname, "."+domain) {
				domainAllowed = true
				break
			}
		}
		if !domainAllowed {
			return "", fmt.Errorf("%w: %q not in allowlist", ErrDisallowedDomain, hostname)
		}
	}

	// Check for SSRF risks (private IPs, localhost, etc.)
	if constraints.BlockPrivate {
		if err := checkSSRF(hostname); err != nil {
			return "", err
		}
	}

	return urlStr, nil
}

// checkSSRF checks if a hostname could be used for SSRF attacks.
// Blocks localhost, private IPs, link-local addresses, etc.
//
// Note: This performs validation at check-time, not request-time. For strongest
// SSRF protection, consumers should call this validation immediately before making
// HTTP requests to minimize DNS rebinding risks (TOCTOU).
func checkSSRF(hostname string) error {
	// Block localhost variations
	lower := strings.ToLower(hostname)
	if lower == "localhost" || lower == "localhost.localdomain" {
		return fmt.Errorf("%w: localhost not allowed", ErrSSRFRisk)
	}

	// Check if hostname is already an IP address (avoid unnecessary DNS lookup)
	if ip := net.ParseIP(hostname); ip != nil {
		if isPrivateIP(ip) {
			return fmt.Errorf("%w: private IP address not allowed", ErrSSRFRisk)
		}
		return nil
	}

	// Perform DNS resolution for domain names with timeout context
	// Note: DNS resolution can be slow (100ms+) and introduces TOCTOU risk
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	resolver := &net.Resolver{}
	ips, err := resolver.LookupIP(ctx, "ip", hostname)
	if err != nil {
		// Fail closed: reject URLs with unresolvable hostnames to prevent SSRF
		return fmt.Errorf("%w: failed to resolve hostname: %v", ErrSSRFRisk, err)
	}

	// Check each resolved IP
	for _, ip := range ips {
		if isPrivateIP(ip) {
			return fmt.Errorf("%w: private IP address %s", ErrSSRFRisk, ip.String())
		}
	}

	return nil
}

// isPrivateIP checks if an IP address is private, loopback, or link-local.
// This includes common SSRF bypass vectors.
func isPrivateIP(ip net.IP) bool {
	// Check for loopback (covers 127.0.0.0/8 and ::1)
	// IsLoopback handles various IPv6 loopback representations
	if ip.IsLoopback() {
		return true
	}

	// Check for link-local (covers 169.254.0.0/16 and fe80::/10)
	if ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return true
	}

	// Check for unspecified address (covers 0.0.0.0 and ::)
	if ip.IsUnspecified() {
		return true
	}

	// Check for private IPv4 ranges
	if ip4 := ip.To4(); ip4 != nil {
		// 10.0.0.0/8
		if ip4[0] == 10 {
			return true
		}
		// 172.16.0.0/12
		if ip4[0] == 172 && ip4[1] >= 16 && ip4[1] <= 31 {
			return true
		}
		// 192.168.0.0/16
		if ip4[0] == 192 && ip4[1] == 168 {
			return true
		}
		// 169.254.0.0/16 (link-local, redundant with IsLinkLocalUnicast but explicit)
		if ip4[0] == 169 && ip4[1] == 254 {
			return true
		}
	}

	// Check for private IPv6 ranges
	if ip.To4() == nil {
		// fc00::/7 (unique local addresses)
		if len(ip) == 16 && (ip[0]&0xfe) == 0xfc {
			return true
		}
	}

	return false
}

// AttachmentURL validates a URL for use in post attachments.
// Uses default constraints with SSRF protection.
func AttachmentURL(urlStr string) (string, error) {
	return URL(urlStr, DefaultURLConstraints)
}

// MediaURL validates a URL for media content with more permissive constraints.
// Allows both HTTP and HTTPS but still blocks private IPs.
func MediaURL(urlStr string) (string, error) {
	return URL(urlStr, PublicWebURLConstraints)
}
