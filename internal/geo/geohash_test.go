package geo

import "testing"

func TestRoundGeohash(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		precision int
		want      string
	}{
		// Basic truncation cases
		{
			name:      "truncate to default precision 6",
			input:     "9q8yyk8yuv",
			precision: DefaultPrecision,
			want:      "9q8yyk",
		},
		{
			name:      "truncate to precision 5",
			input:     "9q8yyk8yuv",
			precision: 5,
			want:      "9q8yy",
		},
		{
			name:      "truncate to precision 4",
			input:     "9q8yyk8yuv",
			precision: 4,
			want:      "9q8y",
		},
		// Edge cases with input length
		{
			name:      "input shorter than precision - return as is",
			input:     "9q8",
			precision: 6,
			want:      "9q8",
		},
		{
			name:      "input equal to precision - return as is",
			input:     "9q8yyk",
			precision: 6,
			want:      "9q8yyk",
		},
		{
			name:      "input exactly one char longer",
			input:     "9q8yyk8",
			precision: 6,
			want:      "9q8yyk",
		},
		{
			name:      "single character input",
			input:     "9",
			precision: 6,
			want:      "9",
		},
		// Empty and invalid input cases
		{
			name:      "empty input returns empty",
			input:     "",
			precision: 6,
			want:      "",
		},
		{
			name:      "invalid character - letter a",
			input:     "9q8ayk",
			precision: 6,
			want:      "",
		},
		{
			name:      "invalid character - letter i",
			input:     "9q8iyk",
			precision: 6,
			want:      "",
		},
		{
			name:      "invalid character - letter l",
			input:     "9q8lyk",
			precision: 6,
			want:      "",
		},
		{
			name:      "invalid character - letter o",
			input:     "9q8oyk",
			precision: 6,
			want:      "",
		},
		{
			name:      "invalid character - special char",
			input:     "9q8-yk",
			precision: 6,
			want:      "",
		},
		{
			name:      "invalid character - space",
			input:     "9q8 yk",
			precision: 6,
			want:      "",
		},
		// Case handling
		{
			name:      "uppercase input normalized to lowercase",
			input:     "9Q8YYK8YUV",
			precision: 6,
			want:      "9q8yyk",
		},
		{
			name:      "mixed case input normalized to lowercase",
			input:     "9Q8yYk8YuV",
			precision: 6,
			want:      "9q8yyk",
		},
		// Precision edge cases
		{
			name:      "precision 0 returns empty",
			input:     "9q8yyk",
			precision: 0,
			want:      "",
		},
		{
			name:      "negative precision returns empty",
			input:     "9q8yyk",
			precision: -1,
			want:      "",
		},
		{
			name:      "precision 1",
			input:     "9q8yyk",
			precision: 1,
			want:      "9",
		},
		// Real-world geohash examples
		{
			name:      "San Francisco geohash",
			input:     "9q8yy",
			precision: 4,
			want:      "9q8y",
		},
		{
			name:      "New York geohash",
			input:     "dr5regw3p",
			precision: 6,
			want:      "dr5reg",
		},
		{
			name:      "London geohash",
			input:     "gcpvj0du",
			precision: 5,
			want:      "gcpvj",
		},
		// All valid characters
		{
			name:      "all valid digits",
			input:     "0123456789",
			precision: 10,
			want:      "0123456789",
		},
		{
			name:      "all valid letters",
			input:     "bcdefghjkmnpqrstuvwxyz",
			precision: 22,
			want:      "bcdefghjkmnpqrstuvwxyz",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RoundGeohash(tt.input, tt.precision)
			if got != tt.want {
				t.Errorf("RoundGeohash(%q, %d) = %q, want %q", tt.input, tt.precision, got, tt.want)
			}
		})
	}
}

func TestRoundGeohash_Consistency(t *testing.T) {
	// Test that the same input always produces the same output
	input := "9q8yyk8yuv"
	precision := 6

	first := RoundGeohash(input, precision)
	for i := 0; i < 100; i++ {
		result := RoundGeohash(input, precision)
		if result != first {
			t.Errorf("RoundGeohash inconsistent: first=%q, iteration %d=%q", first, i, result)
		}
	}
}

func TestDefaultPrecision(t *testing.T) {
	// Verify the default precision constant is 6
	if DefaultPrecision != 6 {
		t.Errorf("DefaultPrecision = %d, want 6", DefaultPrecision)
	}
}

func TestEncode(t *testing.T) {
	tests := []struct {
		name      string
		lat       float64
		lng       float64
		precision int
		want      string
	}{
		{
			name:      "Seattle",
			lat:       47.6062,
			lng:       -122.3321,
			precision: 6,
			want:      "c23nb6",
		},
		{
			name:      "Berlin",
			lat:       52.5200,
			lng:       13.4050,
			precision: 6,
			want:      "u33dc0",
		},
		{
			name:      "London",
			lat:       51.5074,
			lng:       -0.1278,
			precision: 6,
			want:      "gcpvj0",
		},
		{
			name:      "precision 5",
			lat:       47.6062,
			lng:       -122.3321,
			precision: 5,
			want:      "c23nb",
		},
		{
			name:      "default precision",
			lat:       47.6062,
			lng:       -122.3321,
			precision: 0, // Should use DefaultPrecision = 6
			want:      "c23nb6",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Encode(tt.lat, tt.lng, tt.precision)
			if got != tt.want {
				t.Errorf("Encode(%f, %f, %d) = %q, want %q", tt.lat, tt.lng, tt.precision, got, tt.want)
			}
		})
	}
}
