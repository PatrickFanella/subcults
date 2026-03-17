// Package indexer provides filtering and processing of AT Protocol records
// for the Subcults Jetstream indexer.
package indexer

import (
	"errors"
	"time"
)

// Default values for WebSocket reconnection configuration.
const (
	DefaultBaseDelay        = 100 * time.Millisecond
	DefaultMaxDelay         = 30 * time.Second
	DefaultJitterFactor     = 0.5 // 50% jitter
	DefaultMaxRetryAttempts = 5   // Max retry attempts before alerting
)

// Configuration errors.
var (
	ErrEmptyURL        = errors.New("jetstream URL cannot be empty")
	ErrInvalidDelay    = errors.New("base delay must be positive")
	ErrInvalidMaxDelay = errors.New("max delay must be >= base delay")
	ErrInvalidJitter   = errors.New("jitter factor must be between 0 and 1")
)

// Config holds configuration for the Jetstream WebSocket client.
type Config struct {
	// URL is the Jetstream WebSocket endpoint URL.
	URL string

	// BaseDelay is the initial delay before first reconnect attempt.
	BaseDelay time.Duration

	// MaxDelay is the maximum delay between reconnect attempts.
	MaxDelay time.Duration

	// JitterFactor is the fraction of delay to randomize (0.0 to 1.0).
	// A value of 0.5 means the actual delay will be in [delay*0.75, delay*1.25].
	JitterFactor float64

	// MaxRetryAttempts is the maximum number of consecutive reconnection attempts
	// before logging an alert. Set to 0 to disable the limit.
	MaxRetryAttempts int64
}

// DefaultConfig returns a Config with sensible default values.
// The URL must be provided by the caller.
func DefaultConfig(url string) Config {
	return Config{
		URL:              url,
		BaseDelay:        DefaultBaseDelay,
		MaxDelay:         DefaultMaxDelay,
		JitterFactor:     DefaultJitterFactor,
		MaxRetryAttempts: DefaultMaxRetryAttempts,
	}
}

// Validate checks that the configuration is valid.
func (c Config) Validate() error {
	if c.URL == "" {
		return ErrEmptyURL
	}
	if c.BaseDelay <= 0 {
		return ErrInvalidDelay
	}
	if c.MaxDelay < c.BaseDelay {
		return ErrInvalidMaxDelay
	}
	if c.JitterFactor < 0 || c.JitterFactor > 1 {
		return ErrInvalidJitter
	}
	return nil
}
