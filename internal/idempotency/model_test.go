package idempotency

import (
	"strings"
	"testing"
)

func TestValidateKey(t *testing.T) {
	tests := []struct {
		name      string
		key       string
		expectErr error
	}{
		{
			name:      "empty key",
			key:       "",
			expectErr: ErrInvalidKey,
		},
		{
			name:      "valid key",
			key:       "test-key-123",
			expectErr: nil,
		},
		{
			name:      "key at max length",
			key:       strings.Repeat("a", MaxKeyLength),
			expectErr: nil,
		},
		{
			name:      "key exceeds max length",
			key:       strings.Repeat("a", MaxKeyLength+1),
			expectErr: ErrKeyTooLong,
		},
		{
			name:      "uuid format key",
			key:       "550e8400-e29b-41d4-a716-446655440000",
			expectErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateKey(tt.key)
			if err != tt.expectErr {
				t.Errorf("ValidateKey() error = %v, expectErr %v", err, tt.expectErr)
			}
		})
	}
}

func TestComputeResponseHash(t *testing.T) {
	tests := []struct {
		name         string
		responseBody string
	}{
		{
			name:         "empty response",
			responseBody: "",
		},
		{
			name:         "simple json",
			responseBody: `{"status":"ok"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash := ComputeResponseHash(tt.responseBody)

			// Just verify it's not empty and has correct length (SHA256 is 64 hex chars)
			if len(hash) != 64 {
				t.Errorf("ComputeResponseHash() hash length = %d, want 64", len(hash))
			}

			// Verify consistency - same input produces same hash
			hash2 := ComputeResponseHash(tt.responseBody)
			if hash != hash2 {
				t.Errorf("ComputeResponseHash() not consistent: %s != %s", hash, hash2)
			}
		})
	}
}

func TestComputeResponseHash_Uniqueness(t *testing.T) {
	hash1 := ComputeResponseHash(`{"session_url":"https://example.com/1"}`)
	hash2 := ComputeResponseHash(`{"session_url":"https://example.com/2"}`)

	if hash1 == hash2 {
		t.Error("Different responses should produce different hashes")
	}
}
