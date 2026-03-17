package indexer

import "testing"

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr error
	}{
		{
			name:    "valid default config",
			config:  DefaultConfig("wss://jetstream.example.com"),
			wantErr: nil,
		},
		{
			name: "valid custom config",
			config: Config{
				URL:          "wss://jetstream.example.com",
				BaseDelay:    50,
				MaxDelay:     100,
				JitterFactor: 0.25,
			},
			wantErr: nil,
		},
		{
			name: "empty URL",
			config: Config{
				URL:          "",
				BaseDelay:    100,
				MaxDelay:     200,
				JitterFactor: 0.5,
			},
			wantErr: ErrEmptyURL,
		},
		{
			name: "zero base delay",
			config: Config{
				URL:          "wss://test.example.com",
				BaseDelay:    0,
				MaxDelay:     200,
				JitterFactor: 0.5,
			},
			wantErr: ErrInvalidDelay,
		},
		{
			name: "negative base delay",
			config: Config{
				URL:          "wss://test.example.com",
				BaseDelay:    -1,
				MaxDelay:     200,
				JitterFactor: 0.5,
			},
			wantErr: ErrInvalidDelay,
		},
		{
			name: "max delay less than base delay",
			config: Config{
				URL:          "wss://test.example.com",
				BaseDelay:    200,
				MaxDelay:     100,
				JitterFactor: 0.5,
			},
			wantErr: ErrInvalidMaxDelay,
		},
		{
			name: "negative jitter factor",
			config: Config{
				URL:          "wss://test.example.com",
				BaseDelay:    100,
				MaxDelay:     200,
				JitterFactor: -0.1,
			},
			wantErr: ErrInvalidJitter,
		},
		{
			name: "jitter factor greater than 1",
			config: Config{
				URL:          "wss://test.example.com",
				BaseDelay:    100,
				MaxDelay:     200,
				JitterFactor: 1.1,
			},
			wantErr: ErrInvalidJitter,
		},
		{
			name: "jitter factor exactly 0",
			config: Config{
				URL:          "wss://test.example.com",
				BaseDelay:    100,
				MaxDelay:     200,
				JitterFactor: 0,
			},
			wantErr: nil,
		},
		{
			name: "jitter factor exactly 1",
			config: Config{
				URL:          "wss://test.example.com",
				BaseDelay:    100,
				MaxDelay:     200,
				JitterFactor: 1,
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if err != tt.wantErr {
				t.Errorf("Config.Validate() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestDefaultConfig(t *testing.T) {
	url := "wss://jetstream.example.com"
	config := DefaultConfig(url)

	if config.URL != url {
		t.Errorf("DefaultConfig().URL = %s, want %s", config.URL, url)
	}
	if config.BaseDelay != DefaultBaseDelay {
		t.Errorf("DefaultConfig().BaseDelay = %v, want %v", config.BaseDelay, DefaultBaseDelay)
	}
	if config.MaxDelay != DefaultMaxDelay {
		t.Errorf("DefaultConfig().MaxDelay = %v, want %v", config.MaxDelay, DefaultMaxDelay)
	}
	if config.JitterFactor != DefaultJitterFactor {
		t.Errorf("DefaultConfig().JitterFactor = %v, want %v", config.JitterFactor, DefaultJitterFactor)
	}
}
