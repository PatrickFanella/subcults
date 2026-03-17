package audit

import (
	"testing"
	"time"
)

func TestAnonymizeIP_IPv4(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "standard IPv4",
			input:    "192.168.1.100",
			expected: "192.168.1.0",
		},
		{
			name:     "IPv4 with last octet already 0",
			input:    "10.0.0.0",
			expected: "10.0.0.0",
		},
		{
			name:     "IPv4 with 255 in last octet",
			input:    "172.16.254.255",
			expected: "172.16.254.0",
		},
		{
			name:     "public IPv4",
			input:    "203.0.113.195",
			expected: "203.0.113.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := AnonymizeIP(tt.input)
			if result != tt.expected {
				t.Errorf("AnonymizeIP(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestAnonymizeIP_IPv6(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "standard IPv6",
			input:    "2001:0db8:85a3:0000:0000:8a2e:0370:7334",
			expected: "2001:db8:85a3::",
		},
		{
			name:     "compressed IPv6",
			input:    "2001:db8:85a3::8a2e:370:7334",
			expected: "2001:db8:85a3::",
		},
		{
			name:     "loopback IPv6",
			input:    "::1",
			expected: "::",
		},
		{
			name:     "IPv6 with all zeros in suffix",
			input:    "fe80::",
			expected: "fe80::",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := AnonymizeIP(tt.input)
			if result != tt.expected {
				t.Errorf("AnonymizeIP(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestAnonymizeIP_InvalidInput(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "empty string",
			input: "",
		},
		{
			name:  "invalid IP",
			input: "not-an-ip",
		},
		{
			name:  "partial IPv4",
			input: "192.168.1",
		},
		{
			name:  "too many octets",
			input: "192.168.1.1.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := AnonymizeIP(tt.input)
			if result != "" {
				t.Errorf("AnonymizeIP(%q) = %q, want empty string", tt.input, result)
			}
		})
	}
}

func TestIPAnonymizationCutoff(t *testing.T) {
	cutoff := IPAnonymizationCutoff()
	
	// Should be approximately 90 days ago
	expectedCutoff := time.Now().UTC().Add(-90 * 24 * time.Hour)
	
	// Allow 1 second tolerance for test execution time
	diff := cutoff.Sub(expectedCutoff)
	if diff < -1*time.Second || diff > 1*time.Second {
		t.Errorf("IPAnonymizationCutoff() = %v, expected approximately %v (diff: %v)", 
			cutoff, expectedCutoff, diff)
	}
	
	// Verify it's in UTC
	if cutoff.Location() != time.UTC {
		t.Errorf("IPAnonymizationCutoff() location = %v, want UTC", cutoff.Location())
	}
	
	// Verify it's in the past
	if cutoff.After(time.Now().UTC()) {
		t.Error("IPAnonymizationCutoff() should be in the past")
	}
}
