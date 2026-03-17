package validate

import (
	"net"
	"strings"
	"testing"
)

func TestURL(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		constraints URLConstraints
		wantErr     bool
		errType     error
	}{
		{
			name:  "valid HTTPS URL",
			input: "https://example.com/path",
			constraints: URLConstraints{
				AllowedSchemes: []string{"https"},
				BlockPrivate:   false,
			},
			wantErr: false,
		},
		{
			name:  "valid HTTP URL",
			input: "http://example.com",
			constraints: URLConstraints{
				AllowedSchemes: []string{"http", "https"},
				BlockPrivate:   false,
			},
			wantErr: false,
		},
		{
			name:  "empty URL",
			input: "",
			constraints: URLConstraints{
				AllowedSchemes: []string{"https"},
			},
			wantErr: true,
			errType: ErrEmpty,
		},
		{
			name:  "disallowed scheme",
			input: "ftp://example.com",
			constraints: URLConstraints{
				AllowedSchemes: []string{"https"},
			},
			wantErr: true,
			errType: ErrDisallowedScheme,
		},
		{
			name:  "URL too long",
			input: "https://example.com/" + strings.Repeat("a", 2048),
			constraints: URLConstraints{
				AllowedSchemes: []string{"https"},
				MaxLength:      2048,
			},
			wantErr: true,
		},
		{
			name:  "localhost blocked",
			input: "https://localhost/admin",
			constraints: URLConstraints{
				AllowedSchemes: []string{"https"},
				BlockPrivate:   true,
			},
			wantErr: true,
			errType: ErrSSRFRisk,
		},
		{
			name:  "private IP blocked (10.x.x.x)",
			input: "https://10.0.0.1/internal",
			constraints: URLConstraints{
				AllowedSchemes: []string{"https"},
				BlockPrivate:   true,
			},
			wantErr: true,
			errType: ErrSSRFRisk,
		},
		{
			name:  "private IP blocked (192.168.x.x)",
			input: "https://192.168.1.1/router",
			constraints: URLConstraints{
				AllowedSchemes: []string{"https"},
				BlockPrivate:   true,
			},
			wantErr: true,
			errType: ErrSSRFRisk,
		},
		{
			name:  "private IP blocked (172.16-31.x.x)",
			input: "https://172.16.0.1/internal",
			constraints: URLConstraints{
				AllowedSchemes: []string{"https"},
				BlockPrivate:   true,
			},
			wantErr: true,
			errType: ErrSSRFRisk,
		},
		{
			name:  "domain allowlist - allowed",
			input: "https://api.example.com/data",
			constraints: URLConstraints{
				AllowedSchemes: []string{"https"},
				AllowedDomains: []string{"example.com"},
			},
			wantErr: false,
		},
		{
			name:  "domain allowlist - blocked",
			input: "https://evil.com/malware",
			constraints: URLConstraints{
				AllowedSchemes: []string{"https"},
				AllowedDomains: []string{"example.com"},
			},
			wantErr: true,
			errType: ErrDisallowedDomain,
		},
		{
			name:  "missing hostname",
			input: "https:///path",
			constraints: URLConstraints{
				AllowedSchemes: []string{"https"},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := URL(tt.input, tt.constraints)
			if (err != nil) != tt.wantErr {
				t.Errorf("URL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got == "" {
				t.Errorf("URL() returned empty string for valid input")
			}
		})
	}
}

func TestDefaultURLConstraints(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "HTTPS allowed with resolvable domain",
			input:   "https://www.google.com",
			wantErr: false,
		},
		{
			name:    "HTTP blocked by default",
			input:   "http://www.google.com",
			wantErr: true,
		},
		{
			name:    "localhost blocked by default",
			input:   "https://localhost",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := URL(tt.input, DefaultURLConstraints)
			if (err != nil) != tt.wantErr {
				t.Errorf("URL() with DefaultURLConstraints error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPublicWebURLConstraints(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "HTTPS allowed with resolvable domain",
			input:   "https://www.google.com",
			wantErr: false,
		},
		{
			name:    "HTTP allowed with resolvable domain",
			input:   "http://www.google.com",
			wantErr: false,
		},
		{
			name:    "localhost blocked",
			input:   "http://localhost",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := URL(tt.input, PublicWebURLConstraints)
			if (err != nil) != tt.wantErr {
				t.Errorf("URL() with PublicWebURLConstraints error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAttachmentURL(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "valid HTTPS URL with resolvable domain",
			input:   "https://www.google.com/image.jpg",
			wantErr: false,
		},
		{
			name:    "unresolvable domain rejected (SSRF fail-closed)",
			input:   "https://cdn.example.com/image.jpg",
			wantErr: true,
		},
		{
			name:    "HTTP not allowed",
			input:   "http://example.com/image.jpg",
			wantErr: true,
		},
		{
			name:    "localhost blocked",
			input:   "https://localhost/file.jpg",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := AttachmentURL(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("AttachmentURL() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMediaURL(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "valid HTTPS URL with resolvable domain",
			input:   "https://www.google.com/video.mp4",
			wantErr: false,
		},
		{
			name:    "HTTP allowed for media with resolvable domain",
			input:   "http://www.google.com/audio.mp3",
			wantErr: false,
		},
		{
			name:    "unresolvable domain rejected (SSRF fail-closed)",
			input:   "https://media.example.com/video.mp4",
			wantErr: true,
		},
		{
			name:    "localhost blocked",
			input:   "http://localhost/media.mp4",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := MediaURL(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("MediaURL() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestIsPrivateIP(t *testing.T) {
	tests := []struct {
		name string
		ip   string
		want bool
	}{
		// Loopback addresses
		{name: "IPv4 loopback", ip: "127.0.0.1", want: true},
		{name: "IPv6 loopback", ip: "::1", want: true},

		// Private IPv4 ranges
		{name: "10.x.x.x private", ip: "10.0.0.1", want: true},
		{name: "10.x.x.x private high", ip: "10.255.255.255", want: true},
		{name: "172.16.x.x private", ip: "172.16.0.1", want: true},
		{name: "172.31.x.x private", ip: "172.31.255.255", want: true},
		{name: "192.168.x.x private", ip: "192.168.1.1", want: true},

		// Link-local
		{name: "169.254.x.x link-local", ip: "169.254.169.254", want: true},

		// Unspecified addresses (0.0.0.0)
		{name: "0.0.0.0 unspecified", ip: "0.0.0.0", want: true},
		{name: "IPv6 unspecified", ip: "::", want: true},

		// Public IPs
		{name: "public IP 8.8.8.8", ip: "8.8.8.8", want: false},
		{name: "public IP 1.1.1.1", ip: "1.1.1.1", want: false},

		// Edge cases
		{name: "172.15.x.x not private", ip: "172.15.0.1", want: false},
		{name: "172.32.x.x not private", ip: "172.32.0.1", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := parseIP(tt.ip)
			if ip == nil {
				t.Fatalf("Failed to parse IP: %s", tt.ip)
			}
			got := isPrivateIP(ip)
			if got != tt.want {
				t.Errorf("isPrivateIP(%s) = %v, want %v", tt.ip, got, tt.want)
			}
		})
	}
}

// Helper to parse IP for testing
func parseIP(s string) net.IP {
	return net.ParseIP(s)
}
