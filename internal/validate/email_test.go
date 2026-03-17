package validate

import (
	"strings"
	"testing"
)

func TestEmail(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:    "valid email",
			input:   "user@example.com",
			want:    "user@example.com",
			wantErr: false,
		},
		{
			name:    "valid email with subdomain",
			input:   "user@mail.example.com",
			want:    "user@mail.example.com",
			wantErr: false,
		},
		{
			name:    "valid email with plus",
			input:   "user+tag@example.com",
			want:    "user+tag@example.com",
			wantErr: false,
		},
		{
			name:    "valid email with dots",
			input:   "first.last@example.com",
			want:    "first.last@example.com",
			wantErr: false,
		},
		{
			name:    "email normalized to lowercase",
			input:   "User@Example.COM",
			want:    "user@example.com",
			wantErr: false,
		},
		{
			name:    "email with whitespace trimmed",
			input:   "  user@example.com  ",
			want:    "user@example.com",
			wantErr: false,
		},
		{
			name:    "empty email",
			input:   "",
			want:    "",
			wantErr: true,
		},
		{
			name:    "missing @",
			input:   "userexample.com",
			want:    "",
			wantErr: true,
		},
		{
			name:    "missing domain",
			input:   "user@",
			want:    "",
			wantErr: true,
		},
		{
			name:    "missing local part",
			input:   "@example.com",
			want:    "",
			wantErr: true,
		},
		{
			name:    "missing TLD",
			input:   "user@example",
			want:    "",
			wantErr: true,
		},
		{
			name:    "multiple @",
			input:   "user@@example.com",
			want:    "",
			wantErr: true,
		},
		{
			name:    "local part too long",
			input:   strings.Repeat("a", 65) + "@example.com",
			want:    "",
			wantErr: true,
		},
		{
			name:    "total length too long",
			input:   "user@" + strings.Repeat("a", 250) + ".com",
			want:    "",
			wantErr: true,
		},
		{
			name:    "invalid characters",
			input:   "user name@example.com",
			want:    "",
			wantErr: true,
		},
		{
			name:    "valid international domain",
			input:   "user@example.co.uk",
			want:    "user@example.co.uk",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Email(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Email() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Email() = %q, want %q", got, tt.want)
			}
		})
	}
}
