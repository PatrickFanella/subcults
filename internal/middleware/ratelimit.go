// Package middleware provides HTTP middleware components for the API server.
package middleware

import (
	"context"
	"fmt"
	"crypto/subtle"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// RateLimitConfig defines the rate limiting configuration.
// Valid values:
//   - RequestsPerWindow: must be > 0
//   - WindowDuration: must be > 0
//   - BurstFactor: must be >= 1.0 when non-zero (0 disables burst)
//   - BurstWindow: duration of the burst sub-window; defaults to 10s when BurstFactor > 1.0
type RateLimitConfig struct {
	// RequestsPerWindow is the maximum number of requests allowed per window.
	// Must be > 0.
	RequestsPerWindow int
	// WindowDuration is the time window for the rate limit.
	// Must be > 0.
	WindowDuration time.Duration
	// BurstFactor, when > 1.0, allows brief spikes above the base rate.
	// For example, 1.5 allows 1.5x the base RequestsPerWindow during the
	// BurstWindow at the start of each main window. Set to 0 to disable.
	BurstFactor float64
	// BurstWindow is the duration of the burst sub-window within each main window.
	// Only relevant when BurstFactor > 1.0. Defaults to 10 seconds when not set.
	BurstWindow time.Duration
}

// Validate checks that the RateLimitConfig has valid values.
// Returns an error if RequestsPerWindow <= 0, WindowDuration <= 0, or BurstFactor < 1.0.
func (c RateLimitConfig) Validate() error {
	if c.RequestsPerWindow <= 0 {
		return fmt.Errorf("RequestsPerWindow must be > 0 (got %d)", c.RequestsPerWindow)
	}
	if c.WindowDuration <= 0 {
		return fmt.Errorf("WindowDuration must be > 0 (got %s)", c.WindowDuration)
	}
	if c.BurstFactor != 0 && c.BurstFactor < 1.0 {
		return fmt.Errorf("BurstFactor must be >= 1.0 when set (got %.2f)", c.BurstFactor)
	}
	return nil
}

// effectiveBurstWindow returns the burst sub-window duration, defaulting to 10s.
func (c RateLimitConfig) effectiveBurstWindow() time.Duration {
	if c.BurstWindow > 0 {
		return c.BurstWindow
	}
	return 10 * time.Second
}

// burstLimit returns the effective burst limit (requests allowed during burst window).
// Returns RequestsPerWindow when burst is disabled.
func (c RateLimitConfig) burstLimit() int {
	if c.BurstFactor > 1.0 {
		return int(float64(c.RequestsPerWindow) * c.BurstFactor)
	}
	return c.RequestsPerWindow
}

// defaultGlobalLimit is the default global rate limit (100 requests per minute).
var defaultGlobalLimit = RateLimitConfig{
	RequestsPerWindow: 100,
	WindowDuration:    time.Minute,
}

// defaultAuthLimit is the default auth endpoint rate limit (10 requests per minute).
var defaultAuthLimit = RateLimitConfig{
	RequestsPerWindow: 10,
	WindowDuration:    time.Minute,
}

// defaultSearchLimit is the default search endpoint rate limit (30 requests per minute).
var defaultSearchLimit = RateLimitConfig{
	RequestsPerWindow: 30,
	WindowDuration:    time.Minute,
}

// DefaultGlobalLimit returns a copy of the default global rate limit config.
func DefaultGlobalLimit() RateLimitConfig {
	return defaultGlobalLimit
}

// DefaultAuthLimit returns a copy of the default auth endpoint rate limit config.
func DefaultAuthLimit() RateLimitConfig {
	return defaultAuthLimit
}

// DefaultSearchLimit returns a copy of the default search endpoint rate limit config.
func DefaultSearchLimit() RateLimitConfig {
	return defaultSearchLimit
}

// RateLimitStore defines the interface for rate limit state storage.
// This allows for different backends (in-memory, Redis, etc.).
type RateLimitStore interface {
	// Allow checks if a request from the given key should be allowed.
	// Returns three values:
	// - allowed: true if request is allowed, false if rate limited
	// - remaining: number of requests remaining in current window
	// - retryAfter: number of seconds until the limit resets (relevant when allowed=false)
	Allow(ctx context.Context, key string, config RateLimitConfig) (allowed bool, remaining int, retryAfter int)
}

// bucket represents a rate limit bucket for a single key.
type bucket struct {
	count     int
	windowEnd time.Time
	burstEnd  time.Time // end of the burst sub-window; zero when burst is disabled
}

// InMemoryRateLimitStore implements RateLimitStore using an in-memory map.
// It uses a simple fixed window counter algorithm.
// Thread-safe for concurrent access.
type InMemoryRateLimitStore struct {
	mu         sync.RWMutex
	buckets    map[string]*bucket
	maxBuckets int
}

// defaultMaxBuckets is the maximum number of tracked keys before rejecting new ones.
const defaultMaxBuckets = 100000

// NewInMemoryRateLimitStore creates a new in-memory rate limit store.
func NewInMemoryRateLimitStore() *InMemoryRateLimitStore {
	return &InMemoryRateLimitStore{
		buckets:    make(map[string]*bucket),
		maxBuckets: defaultMaxBuckets,
	}
}

// Allow checks if a request from the given key should be allowed.
// Implements the RateLimitStore interface.
func (s *InMemoryRateLimitStore) Allow(ctx context.Context, key string, config RateLimitConfig) (bool, int, int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()

	b, exists := s.buckets[key]

	// Protect against memory exhaustion: reject new keys when at capacity
	if !exists && len(s.buckets) >= s.maxBuckets {
		return false, 0, 1
	}

	if !exists || now.After(b.windowEnd) {
		// New window: reset counts and set burst sub-window if configured.
		var burstEnd time.Time
		if config.BurstFactor > 1.0 {
			burstEnd = now.Add(config.effectiveBurstWindow())
		}
		s.buckets[key] = &bucket{
			count:     1,
			windowEnd: now.Add(config.WindowDuration),
			burstEnd:  burstEnd,
		}
		effectiveLimit := config.burstLimit()
		if config.BurstFactor <= 1.0 {
			effectiveLimit = config.RequestsPerWindow
		}
		remaining := effectiveLimit - 1
		return true, remaining, 0
	}

	// Determine effective limit: use burst limit during burst sub-window.
	effectiveLimit := config.RequestsPerWindow
	if config.BurstFactor > 1.0 && !b.burstEnd.IsZero() && now.Before(b.burstEnd) {
		effectiveLimit = config.burstLimit()
	}

	// Check if we're within the limit
	if b.count < effectiveLimit {
		b.count++
		remaining := effectiveLimit - b.count
		return true, remaining, 0
	}

	// Rate limited
	retryAfter := int(b.windowEnd.Sub(now).Seconds())
	if retryAfter <= 0 {
		retryAfter = 1
	}
	return false, 0, retryAfter
}

// Cleanup removes expired buckets to prevent memory leaks.
// This should be called periodically in production.
// Recommended cleanup interval is 2-5x the longest configured WindowDuration
// to balance memory usage and overhead.
func (s *InMemoryRateLimitStore) Cleanup() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	for key, b := range s.buckets {
		if now.After(b.windowEnd) {
			delete(s.buckets, key)
		}
	}
}

// KeyFunc extracts a rate limit key from an HTTP request.
type KeyFunc func(r *http.Request) string

// IPKeyFunc returns a KeyFunc that uses the client's IP address.
func IPKeyFunc() KeyFunc {
	return func(r *http.Request) string {
		// Check X-Forwarded-For header first (for proxied requests)
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			// Use the last (rightmost) IP — the one added by the trusted reverse proxy
			parts := strings.Split(xff, ",")
			return strings.TrimSpace(parts[len(parts)-1])
		}
		// Check X-Real-IP header
		if xri := r.Header.Get("X-Real-IP"); xri != "" {
			return strings.TrimSpace(xri)
		}
		// Fall back to RemoteAddr (strip port properly for both IPv4 and IPv6)
		host, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			// RemoteAddr might not have a port
			return r.RemoteAddr
		}
		return host
	}
}

// UserKeyFunc returns a KeyFunc that uses the authenticated user's DID if available,
// falling back to IP address.
func UserKeyFunc() KeyFunc {
	ipFunc := IPKeyFunc()
	return func(r *http.Request) string {
		if did := GetUserDID(r.Context()); did != "" {
			return "user:" + did
		}
		return "ip:" + ipFunc(r)
	}
}

// RateLimiter is a middleware that limits request rates.
// It returns HTTP 429 Too Many Requests when the limit is exceeded.
// It also sets X-RateLimit-* headers to indicate quota status.
// Rate limit violations are logged via the logging middleware through error codes.
// If metrics is provided, rate limit events are tracked for observability.
func RateLimiter(store RateLimitStore, config RateLimitConfig, keyFunc KeyFunc, metrics *Metrics) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := keyFunc(r)
			allowed, remaining, retryAfter := store.Allow(r.Context(), key, config)

			// Determine key type for metrics
			keyType := "ip"
			if strings.HasPrefix(key, "user:") {
				keyType = "user"
			}

			// Track rate limit request in metrics
			if metrics != nil {
				metrics.IncRateLimitRequests(r.URL.Path, keyType)
			}

			// Set rate limit headers
			w.Header().Set("X-RateLimit-Limit", strconv.Itoa(config.RequestsPerWindow))
			w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))

			if !allowed {
				// Track rate limit violation in metrics
				if metrics != nil {
					metrics.IncRateLimitBlocked(r.URL.Path, keyType)
				}

				// Set error code for logging middleware
				// The logging middleware will automatically log this with error_code="rate_limit_exceeded"
				ctx := SetErrorCode(r.Context(), "rate_limit_exceeded")

				// Store rate limit details in context for logging
				ctx = SetRateLimitKey(ctx, key)
				r = r.WithContext(ctx)

				w.Header().Set("Retry-After", strconv.Itoa(retryAfter))
				// X-RateLimit-Reset should be a Unix timestamp per API conventions
				resetTime := time.Now().Add(time.Duration(retryAfter) * time.Second).Unix()
				w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(resetTime, 10))
				http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RateLimiterWithBypass creates a rate limiting middleware that can be skipped for
// requests where bypassFunc returns true (e.g., trusted internal services).
// When bypassed, rate limit headers are not set and the request is passed through directly.
// If bypassFunc is nil, the middleware behaves identically to RateLimiter.
func RateLimiterWithBypass(store RateLimitStore, config RateLimitConfig, keyFunc KeyFunc, metrics *Metrics, bypassFunc func(*http.Request) bool) func(http.Handler) http.Handler {
	limited := RateLimiter(store, config, keyFunc, metrics)
	return func(next http.Handler) http.Handler {
		limitedNext := limited(next)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if bypassFunc != nil && bypassFunc(r) {
				next.ServeHTTP(w, r)
				return
			}
			limitedNext.ServeHTTP(w, r)
		})
	}
}

// InternalServiceBypassFunc returns a bypass function that allows requests carrying
// the header "X-Internal-Token: <secret>" to skip rate limiting. This is intended
// for trusted internal service-to-service calls.
// An empty secret disables the bypass (returns false for every request).
func InternalServiceBypassFunc(secret string) func(*http.Request) bool {
	if secret == "" {
		return func(*http.Request) bool { return false }
	}
	return func(r *http.Request) bool {
		token := r.Header.Get("X-Internal-Token")
		return subtle.ConstantTimeCompare([]byte(token), []byte(secret)) == 1
	}
}

// userTierKey is the context key used to store the user tier.
type userTierKey struct{}

// SetUserTier stores the user's tier (e.g., "free", "pro") in the request context.
// The tier is used by TieredRateLimiter / ProTierLimitSelector to select the
// appropriate rate limit configuration.
func SetUserTier(ctx context.Context, tier string) context.Context {
	return context.WithValue(ctx, userTierKey{}, tier)
}

// GetUserTier retrieves the user tier from the context set by SetUserTier.
// Returns an empty string when no tier has been stored.
func GetUserTier(ctx context.Context) string {
	if tier, ok := ctx.Value(userTierKey{}).(string); ok {
		return tier
	}
	return ""
}

// TieredRateLimiter creates a rate limiting middleware whose limit configuration is
// determined per-request by limitSelector. This enables different quotas for
// different user tiers (e.g., free vs. pro).
// The middleware otherwise behaves identically to RateLimiter.
func TieredRateLimiter(store RateLimitStore, limitSelector func(*http.Request) RateLimitConfig, keyFunc KeyFunc, metrics *Metrics) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			config := limitSelector(r)
			key := keyFunc(r)
			allowed, remaining, retryAfter := store.Allow(r.Context(), key, config)

			keyType := "ip"
			if strings.HasPrefix(key, "user:") {
				keyType = "user"
			}

			if metrics != nil {
				metrics.IncRateLimitRequests(r.URL.Path, keyType)
			}

			w.Header().Set("X-RateLimit-Limit", strconv.Itoa(config.RequestsPerWindow))
			w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))

			if !allowed {
				if metrics != nil {
					metrics.IncRateLimitBlocked(r.URL.Path, keyType)
				}

				ctx := SetErrorCode(r.Context(), "rate_limit_exceeded")
				ctx = SetRateLimitKey(ctx, key)
				r = r.WithContext(ctx)

				w.Header().Set("Retry-After", strconv.Itoa(retryAfter))
				resetTime := time.Now().Add(time.Duration(retryAfter) * time.Second).Unix()
				w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(resetTime, 10))
				http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// ProTierLimitSelector returns a limit-selector function for use with TieredRateLimiter.
// Requests whose context contains tier "pro" (set via SetUserTier) receive proLimit;
// all other requests receive defaultLimit.
func ProTierLimitSelector(defaultLimit, proLimit RateLimitConfig) func(*http.Request) RateLimitConfig {
	return func(r *http.Request) RateLimitConfig {
		if GetUserTier(r.Context()) == "pro" {
			return proLimit
		}
		return defaultLimit
	}
}
// It uses a sliding window counter approach for accurate rate limiting.
// Thread-safe and suitable for distributed systems.
type RedisRateLimitStore struct {
	client  *redis.Client
	metrics *Metrics
}

// NewRedisRateLimitStore creates a new Redis-backed rate limit store.
func NewRedisRateLimitStore(client *redis.Client) *RedisRateLimitStore {
	return &RedisRateLimitStore{
		client:  client,
		metrics: nil,
	}
}

// NewRedisRateLimitStoreWithMetrics creates a new Redis-backed rate limit store with metrics.
func NewRedisRateLimitStoreWithMetrics(client *redis.Client, metrics *Metrics) *RedisRateLimitStore {
	return &RedisRateLimitStore{
		client:  client,
		metrics: metrics,
	}
}

// Allow checks if a request from the given key should be allowed using Redis.
// Implements the RateLimitStore interface with a sliding window algorithm.
// When BurstFactor > 1.0 in config, the burst limit (RequestsPerWindow * BurstFactor)
// is used as the effective limit for the Redis sliding window. Full sub-window burst
// tracking is only available with the in-memory store.
func (s *RedisRateLimitStore) Allow(ctx context.Context, key string, config RateLimitConfig) (bool, int, int) {
	// Use burst limit when configured; Redis uses a single sliding window.
	effectiveLimit := config.burstLimit()
	// Use a Lua script for atomic operations
	// This implements a sliding window counter algorithm
	luaScript := `
		local key = KEYS[1]
		local limit = tonumber(ARGV[1])
		local window = tonumber(ARGV[2])
		local now = tonumber(ARGV[3])
		
		-- Remove old entries outside the window
		redis.call('ZREMRANGEBYSCORE', key, '-inf', now - window)
		
		-- Count current requests in window
		local current = redis.call('ZCARD', key)
		
		if current < limit then
			-- Use a per-key sequence to ensure unique members for concurrent requests
			local seqKey = key .. ':seq'
			local seq = redis.call('INCR', seqKey)
			redis.call('EXPIRE', seqKey, window + 10)
			local member = tostring(now) .. '-' .. tostring(seq)
			-- Add current request with timestamp as score and unique member
			redis.call('ZADD', key, now, member)
			-- Set expiry on the key (window duration + buffer)
			redis.call('EXPIRE', key, window + 10)
			-- Return: allowed=1, remaining=limit-current-1
			return {1, limit - current - 1, 0}
		else
			-- Get the oldest request timestamp to calculate retry-after
			local oldest = redis.call('ZRANGE', key, 0, 0, 'WITHSCORES')
			local retryAfter = math.ceil((tonumber(oldest[2]) + window) - now)
			if retryAfter < 1 then
				retryAfter = 1
			end
			-- Return: allowed=0, remaining=0, retryAfter
			return {0, 0, retryAfter}
		end
	`

	now := time.Now().Unix()
	windowSeconds := int64(config.WindowDuration.Seconds())

	result, err := s.client.Eval(ctx, luaScript, []string{key}, effectiveLimit, windowSeconds, now).Result()
	if err != nil {
		// Track Redis error in metrics if available
		if s.metrics != nil {
			s.metrics.IncRateLimitRedisErrors()
		}
		// On Redis error, fail open (allow request)
		// This prevents Redis outages from taking down the entire API
		return true, effectiveLimit, 0
	}

	// Parse result from Lua script with safe type assertions
	resultSlice, ok := result.([]interface{})
	if !ok || len(resultSlice) != 3 {
		// Track parsing error in metrics if available
		if s.metrics != nil {
			s.metrics.IncRateLimitRedisErrors()
		}
		// Invalid result format, fail open
		return true, effectiveLimit, 0
	}

	allowedVal, ok := resultSlice[0].(int64)
	if !ok {
		// Track type assertion error in metrics if available
		if s.metrics != nil {
			s.metrics.IncRateLimitRedisErrors()
		}
		// Unexpected type for allowed flag, fail open
		return true, effectiveLimit, 0
	}
	remainingVal, ok := resultSlice[1].(int64)
	if !ok {
		// Track type assertion error in metrics if available
		if s.metrics != nil {
			s.metrics.IncRateLimitRedisErrors()
		}
		// Unexpected type for remaining count, fail open
		return true, effectiveLimit, 0
	}
	retryAfterVal, ok := resultSlice[2].(int64)
	if !ok {
		// Track type assertion error in metrics if available
		if s.metrics != nil {
			s.metrics.IncRateLimitRedisErrors()
		}
		// Unexpected type for retry-after, fail open
		return true, effectiveLimit, 0
	}

	allowed := allowedVal == 1
	remaining := int(remainingVal)
	retryAfter := int(retryAfterVal)

	return allowed, remaining, retryAfter
}
