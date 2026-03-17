package validate

import (
	"testing"
)

func TestMIMEType(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		allowedTypes []string
		want         string
		wantErr      bool
	}{
		{
			name:         "valid JPEG",
			input:        "image/jpeg",
			allowedTypes: AllowedImageTypes,
			want:         "image/jpeg",
			wantErr:      false,
		},
		{
			name:         "valid PNG",
			input:        "image/png",
			allowedTypes: AllowedImageTypes,
			want:         "image/png",
			wantErr:      false,
		},
		{
			name:         "case insensitive",
			input:        "IMAGE/JPEG",
			allowedTypes: AllowedImageTypes,
			want:         "image/jpeg",
			wantErr:      false,
		},
		{
			name:         "whitespace trimmed",
			input:        "  image/png  ",
			allowedTypes: AllowedImageTypes,
			want:         "image/png",
			wantErr:      false,
		},
		{
			name:         "empty MIME type",
			input:        "",
			allowedTypes: AllowedImageTypes,
			want:         "",
			wantErr:      true,
		},
		{
			name:         "disallowed type",
			input:        "application/x-executable",
			allowedTypes: AllowedImageTypes,
			want:         "",
			wantErr:      true,
		},
		{
			name:         "audio type allowed",
			input:        "audio/mpeg",
			allowedTypes: AllowedAudioTypes,
			want:         "audio/mpeg",
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := MIMEType(tt.input, tt.allowedTypes)
			if (err != nil) != tt.wantErr {
				t.Errorf("MIMEType() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("MIMEType() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFileSize(t *testing.T) {
	tests := []struct {
		name        string
		sizeBytes   int64
		constraints FileConstraints
		wantErr     bool
		errType     error
	}{
		{
			name:      "valid size",
			sizeBytes: 1024 * 1024, // 1MB
			constraints: FileConstraints{
				MaxSizeBytes: 10 * 1024 * 1024, // 10MB
			},
			wantErr: false,
		},
		{
			name:      "size at max",
			sizeBytes: 10 * 1024 * 1024, // 10MB
			constraints: FileConstraints{
				MaxSizeBytes: 10 * 1024 * 1024, // 10MB
			},
			wantErr: false,
		},
		{
			name:      "size too large",
			sizeBytes: 11 * 1024 * 1024, // 11MB
			constraints: FileConstraints{
				MaxSizeBytes: 10 * 1024 * 1024, // 10MB
			},
			wantErr: true,
			errType: ErrFileTooLarge,
		},
		{
			name:      "size too small",
			sizeBytes: 100,
			constraints: FileConstraints{
				MinSizeBytes: 1024,
			},
			wantErr: true,
			errType: ErrFileTooSmall,
		},
		{
			name:      "negative size",
			sizeBytes: -1,
			constraints: FileConstraints{
				MaxSizeBytes: 10 * 1024 * 1024,
			},
			wantErr: true,
		},
		{
			name:      "zero size",
			sizeBytes: 0,
			constraints: FileConstraints{
				MaxSizeBytes: 10 * 1024 * 1024,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := FileSize(tt.sizeBytes, tt.constraints)
			if (err != nil) != tt.wantErr {
				t.Errorf("FileSize() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestFile(t *testing.T) {
	tests := []struct {
		name        string
		mimeType    string
		sizeBytes   int64
		constraints FileConstraints
		wantErr     bool
	}{
		{
			name:      "valid image file",
			mimeType:  "image/jpeg",
			sizeBytes: 2 * 1024 * 1024, // 2MB
			constraints: FileConstraints{
				AllowedTypes: AllowedImageTypes,
				MaxSizeBytes: 10 * 1024 * 1024, // 10MB
			},
			wantErr: false,
		},
		{
			name:      "invalid MIME type",
			mimeType:  "application/x-executable",
			sizeBytes: 1024,
			constraints: FileConstraints{
				AllowedTypes: AllowedImageTypes,
				MaxSizeBytes: 10 * 1024 * 1024,
			},
			wantErr: true,
		},
		{
			name:      "file too large",
			mimeType:  "image/png",
			sizeBytes: 50 * 1024 * 1024, // 50MB
			constraints: FileConstraints{
				AllowedTypes: AllowedImageTypes,
				MaxSizeBytes: 10 * 1024 * 1024, // 10MB
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := File(tt.mimeType, tt.sizeBytes, tt.constraints)
			if (err != nil) != tt.wantErr {
				t.Errorf("File() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestImageFile(t *testing.T) {
	tests := []struct {
		name      string
		mimeType  string
		sizeBytes int64
		wantErr   bool
	}{
		{
			name:      "valid JPEG",
			mimeType:  "image/jpeg",
			sizeBytes: 5 * 1024 * 1024, // 5MB
			wantErr:   false,
		},
		{
			name:      "valid PNG",
			mimeType:  "image/png",
			sizeBytes: 3 * 1024 * 1024, // 3MB
			wantErr:   false,
		},
		{
			name:      "valid GIF",
			mimeType:  "image/gif",
			sizeBytes: 2 * 1024 * 1024, // 2MB
			wantErr:   false,
		},
		{
			name:      "valid WebP",
			mimeType:  "image/webp",
			sizeBytes: 4 * 1024 * 1024, // 4MB
			wantErr:   false,
		},
		{
			name:      "image too large",
			mimeType:  "image/jpeg",
			sizeBytes: 15 * 1024 * 1024, // 15MB
			wantErr:   true,
		},
		{
			name:      "not an image",
			mimeType:  "audio/mpeg",
			sizeBytes: 1024,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ImageFile(tt.mimeType, tt.sizeBytes)
			if (err != nil) != tt.wantErr {
				t.Errorf("ImageFile() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAudioFile(t *testing.T) {
	tests := []struct {
		name      string
		mimeType  string
		sizeBytes int64
		wantErr   bool
	}{
		{
			name:      "valid MP3",
			mimeType:  "audio/mpeg",
			sizeBytes: 10 * 1024 * 1024, // 10MB
			wantErr:   false,
		},
		{
			name:      "valid WAV",
			mimeType:  "audio/wav",
			sizeBytes: 20 * 1024 * 1024, // 20MB
			wantErr:   false,
		},
		{
			name:      "valid OGG",
			mimeType:  "audio/ogg",
			sizeBytes: 15 * 1024 * 1024, // 15MB
			wantErr:   false,
		},
		{
			name:      "audio too large",
			mimeType:  "audio/mpeg",
			sizeBytes: 60 * 1024 * 1024, // 60MB
			wantErr:   true,
		},
		{
			name:      "not audio",
			mimeType:  "image/jpeg",
			sizeBytes: 1024,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := AudioFile(tt.mimeType, tt.sizeBytes)
			if (err != nil) != tt.wantErr {
				t.Errorf("AudioFile() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestVideoFile(t *testing.T) {
	tests := []struct {
		name      string
		mimeType  string
		sizeBytes int64
		wantErr   bool
	}{
		{
			name:      "valid MP4",
			mimeType:  "video/mp4",
			sizeBytes: 100 * 1024 * 1024, // 100MB
			wantErr:   false,
		},
		{
			name:      "valid WebM",
			mimeType:  "video/webm",
			sizeBytes: 200 * 1024 * 1024, // 200MB
			wantErr:   false,
		},
		{
			name:      "video too large",
			mimeType:  "video/mp4",
			sizeBytes: 600 * 1024 * 1024, // 600MB
			wantErr:   true,
		},
		{
			name:      "not video",
			mimeType:  "audio/mpeg",
			sizeBytes: 1024,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := VideoFile(tt.mimeType, tt.sizeBytes)
			if (err != nil) != tt.wantErr {
				t.Errorf("VideoFile() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
