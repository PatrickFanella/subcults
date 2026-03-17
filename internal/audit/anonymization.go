// Package audit provides audit logging functionality for tracking access to
// sensitive endpoints and operations for compliance and incident response.
package audit

import (
	"net"
	"strings"
	"time"
)

// AnonymizeIP anonymizes an IP address according to privacy requirements.
// For IPv4: replaces last octet with 0 (e.g., 192.168.1.100 → 192.168.1.0)
// For IPv6: replaces last 80 bits with zeros
// Returns the anonymized IP address string, or empty string if input is invalid.
func AnonymizeIP(ipStr string) string {
	if ipStr == "" {
		return ""
	}

	ip := net.ParseIP(ipStr)
	if ip == nil {
		return ""
	}

	// Check if IPv4 or IPv6
	if ip.To4() != nil {
		// IPv4: zero out last octet
		parts := strings.Split(ipStr, ".")
		if len(parts) != 4 {
			return ""
		}
		parts[3] = "0"
		return strings.Join(parts, ".")
	} else {
		// IPv6: zero out last 80 bits (keep first 48 bits)
		// IPv6 addresses are 128 bits total
		ipBytes := []byte(ip.To16())
		if len(ipBytes) != 16 {
			return ""
		}

		// Zero out bytes 6-15 (last 80 bits)
		for i := 6; i < 16; i++ {
			ipBytes[i] = 0
		}

		return net.IP(ipBytes).String()
	}
}

// IPAnonymizationCutoff returns the date before which IP addresses should be anonymized.
// Currently set to 90 days ago as per compliance requirements.
func IPAnonymizationCutoff() time.Time {
	return time.Now().UTC().Add(-90 * 24 * time.Hour)
}
