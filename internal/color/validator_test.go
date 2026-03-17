package color

import (
	"math"
	"testing"
)

func TestIsValidHexColor(t *testing.T) {
	tests := []struct {
		name  string
		color string
		want  bool
	}{
		{
			name:  "valid lowercase hex",
			color: "#ff0000",
			want:  true,
		},
		{
			name:  "valid uppercase hex",
			color: "#FF0000",
			want:  true,
		},
		{
			name:  "valid mixed case hex",
			color: "#FfAa00",
			want:  true,
		},
		{
			name:  "missing hash",
			color: "ff0000",
			want:  false,
		},
		{
			name:  "too short",
			color: "#fff",
			want:  false,
		},
		{
			name:  "too long",
			color: "#ff00000",
			want:  false,
		},
		{
			name:  "invalid characters",
			color: "#gggggg",
			want:  false,
		},
		{
			name:  "empty string",
			color: "",
			want:  false,
		},
		{
			name:  "with spaces",
			color: "#ff 00 00",
			want:  false,
		},
		{
			name:  "script tag attempt",
			color: "<script>alert(1)</script>",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsValidHexColor(tt.color)
			if got != tt.want {
				t.Errorf("IsValidHexColor(%q) = %v, want %v", tt.color, got, tt.want)
			}
		})
	}
}

func TestSanitizeColor(t *testing.T) {
	tests := []struct {
		name  string
		color string
		want  string
	}{
		{
			name:  "valid color unchanged",
			color: "#ff0000",
			want:  "#ff0000",
		},
		{
			name:  "valid color with whitespace trimmed",
			color: "  #ff0000  ",
			want:  "#ff0000",
		},
		{
			name:  "invalid format returns empty",
			color: "invalid",
			want:  "",
		},
		{
			name:  "script tag returns empty",
			color: "<script>alert(1)</script>",
			want:  "",
		},
		{
			name:  "html entity injection returns empty",
			color: "#ff&lt;script&gt;",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SanitizeColor(tt.color)
			if got != tt.want {
				t.Errorf("SanitizeColor(%q) = %q, want %q", tt.color, got, tt.want)
			}
		})
	}
}

func TestValidateHexColor(t *testing.T) {
	tests := []struct {
		name    string
		color   string
		wantErr bool
	}{
		{
			name:    "valid hex color",
			color:   "#ff0000",
			wantErr: false,
		},
		{
			name:    "invalid hex color",
			color:   "not-a-color",
			wantErr: true,
		},
		{
			name:    "missing hash",
			color:   "ff0000",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateHexColor(tt.color)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateHexColor(%q) error = %v, wantErr %v", tt.color, err, tt.wantErr)
			}
		})
	}
}

func TestParseHexColor(t *testing.T) {
	tests := []struct {
		name    string
		hex     string
		want    RGB
		wantErr bool
	}{
		{
			name: "red",
			hex:  "#ff0000",
			want: RGB{R: 255, G: 0, B: 0},
		},
		{
			name: "green",
			hex:  "#00ff00",
			want: RGB{R: 0, G: 255, B: 0},
		},
		{
			name: "blue",
			hex:  "#0000ff",
			want: RGB{R: 0, G: 0, B: 255},
		},
		{
			name: "white",
			hex:  "#ffffff",
			want: RGB{R: 255, G: 255, B: 255},
		},
		{
			name: "black",
			hex:  "#000000",
			want: RGB{R: 0, G: 0, B: 0},
		},
		{
			name: "gray",
			hex:  "#808080",
			want: RGB{R: 128, G: 128, B: 128},
		},
		{
			name:    "invalid format",
			hex:     "invalid",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseHexColor(tt.hex)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseHexColor(%q) error = %v, wantErr %v", tt.hex, err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("ParseHexColor(%q) = %+v, want %+v", tt.hex, got, tt.want)
			}
		})
	}
}

func TestContrastRatio(t *testing.T) {
	tests := []struct {
		name   string
		color1 RGB
		color2 RGB
		want   float64
	}{
		{
			name:   "black on white - maximum contrast",
			color1: RGB{R: 0, G: 0, B: 0},       // black
			color2: RGB{R: 255, G: 255, B: 255}, // white
			want:   21.0,
		},
		{
			name:   "white on black - same as black on white",
			color1: RGB{R: 255, G: 255, B: 255}, // white
			color2: RGB{R: 0, G: 0, B: 0},       // black
			want:   21.0,
		},
		{
			name:   "identical colors - minimum contrast",
			color1: RGB{R: 128, G: 128, B: 128},
			color2: RGB{R: 128, G: 128, B: 128},
			want:   1.0,
		},
		{
			name:   "dark blue on white - should pass WCAG AA",
			color1: RGB{R: 0, G: 0, B: 139},     // dark blue
			color2: RGB{R: 255, G: 255, B: 255}, // white
			want:   15.3,                        // approximate
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ContrastRatio(tt.color1, tt.color2)

			// Allow small floating point differences
			if math.Abs(got-tt.want) > 0.1 {
				t.Errorf("ContrastRatio() = %.2f, want %.2f", got, tt.want)
			}
		})
	}
}

func TestValidateContrast(t *testing.T) {
	tests := []struct {
		name      string
		textColor string
		bgColor   string
		wantErr   bool
		minRatio  float64
	}{
		{
			name:      "black on white - passes",
			textColor: "#000000",
			bgColor:   "#ffffff",
			wantErr:   false,
			minRatio:  21.0,
		},
		{
			name:      "white on black - passes",
			textColor: "#ffffff",
			bgColor:   "#000000",
			wantErr:   false,
			minRatio:  21.0,
		},
		{
			name:      "dark gray on white - passes",
			textColor: "#595959",
			bgColor:   "#ffffff",
			wantErr:   false,
			minRatio:  4.5,
		},
		{
			name:      "light gray on white - fails",
			textColor: "#cccccc",
			bgColor:   "#ffffff",
			wantErr:   true,
		},
		{
			name:      "yellow on white - fails",
			textColor: "#ffff00",
			bgColor:   "#ffffff",
			wantErr:   true,
		},
		{
			name:      "dark blue on black - fails",
			textColor: "#0066ff",
			bgColor:   "#000000",
			wantErr:   true,
		},
		{
			name:      "invalid text color",
			textColor: "invalid",
			bgColor:   "#ffffff",
			wantErr:   true,
		},
		{
			name:      "invalid background color",
			textColor: "#000000",
			bgColor:   "invalid",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ratio, err := ValidateContrast(tt.textColor, tt.bgColor)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateContrast() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && ratio < tt.minRatio {
				t.Errorf("ValidateContrast() ratio = %.2f, want >= %.2f", ratio, tt.minRatio)
			}
		})
	}
}
