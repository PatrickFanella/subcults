// Package config provides configuration loading and validation for the API server.
// It uses koanf to merge environment variables with optional file overrides.
package config

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

// Config holds all configuration values for the API server.
type Config struct {
	// Server settings
	Port int    `koanf:"port"`
	Env  string `koanf:"env"`

	// Database
	DatabaseURL string `koanf:"database_url"`

	// JWT Authentication
	JWTSecret         string `koanf:"jwt_secret"`          // Legacy: single secret (backward compatibility)
	JWTSecretCurrent  string `koanf:"jwt_secret_current"`  // Current signing key
	JWTSecretPrevious string `koanf:"jwt_secret_previous"` // Previous key for rotation window

	// LiveKit (WebRTC)
	LiveKitURL       string `koanf:"livekit_url"`
	LiveKitAPIKey    string `koanf:"livekit_api_key"`
	LiveKitAPISecret string `koanf:"livekit_api_secret"`

	// Stripe
	StripeAPIKey                string  `koanf:"stripe_api_key"`
	StripeWebhookSecret         string  `koanf:"stripe_webhook_secret"`
	StripeOnboardingReturnURL   string  `koanf:"stripe_onboarding_return_url"`
	StripeOnboardingRefreshURL  string  `koanf:"stripe_onboarding_refresh_url"`
	StripeApplicationFeePercent float64 `koanf:"stripe_application_fee_percent"` // Platform fee as percentage (e.g., 5.0 for 5%)

	// MapTiler
	MapTilerAPIKey string `koanf:"maptiler_api_key"`

	// Jetstream (AT Protocol)
	JetstreamURL string `koanf:"jetstream_url"`

	// R2 (Cloudflare Object Storage)
	R2BucketName      string `koanf:"r2_bucket_name"`
	R2AccessKeyID     string `koanf:"r2_access_key_id"`
	R2SecretAccessKey string `koanf:"r2_secret_access_key"`
	R2Endpoint        string `koanf:"r2_endpoint"`
	R2MaxUploadSizeMB int    `koanf:"r2_max_upload_size_mb"` // Default: 15MB

	// Redis (Rate Limiting)
	RedisURL string `koanf:"redis_url"` // Optional: Redis connection URL for distributed rate limiting

	// Rate Limiting
	// InternalServiceToken, when non-empty, allows requests bearing the header
	// "X-Internal-Token: <value>" to bypass per-endpoint rate limiting.
	// Should be a long random secret shared only with trusted internal services.
	InternalServiceToken string `koanf:"internal_service_token"` // Optional: shared secret for internal service bypass

	// Feature Flags
	RankTrustEnabled bool `koanf:"rank_trust_enabled"` // Enable trust-weighted ranking in search/feed

	// Canary Deployment
	CanaryEnabled          bool    `koanf:"canary_enabled"`           // Enable canary deployment
	CanaryTrafficPercent   float64 `koanf:"canary_traffic_percent"`   // Percentage of traffic to route to canary (0-100)
	CanaryErrorThreshold   float64 `koanf:"canary_error_threshold"`   // Error rate threshold for auto-rollback (0-100)
	CanaryLatencyThreshold float64 `koanf:"canary_latency_threshold"` // Latency threshold in seconds for auto-rollback
	CanaryAutoRollback     bool    `koanf:"canary_auto_rollback"`     // Enable automatic rollback on threshold breach
	CanaryMonitoringWindow int     `koanf:"canary_monitoring_window"` // Monitoring window in seconds for metrics comparison
	CanaryVersion          string  `koanf:"canary_version"`           // Version identifier for canary deployment (e.g., "v1.2.0-canary")

	// Tracing (OpenTelemetry)
	TracingEnabled      bool    `koanf:"tracing_enabled"`       // Enable distributed tracing
	TracingExporterType string  `koanf:"tracing_exporter_type"` // Exporter type: otlp-http, otlp-grpc
	TracingOTLPEndpoint string  `koanf:"tracing_otlp_endpoint"` // OTLP endpoint URL
	TracingSampleRate   float64 `koanf:"tracing_sample_rate"`   // Sampling rate (0.0 to 1.0)
	TracingInsecure     bool    `koanf:"tracing_insecure"`      // Disable TLS for OTLP (dev only)

	// Profiling (pprof)
	ProfilingEnabled bool `koanf:"profiling_enabled"` // Enable pprof profiling endpoints (dev only, NEVER in production)

	// CORS (Cross-Origin Resource Sharing)
	CORSAllowedOrigins   string `koanf:"cors_allowed_origins"`   // Comma-separated list of allowed origins (no wildcards)
	CORSAllowedMethods   string `koanf:"cors_allowed_methods"`   // Comma-separated list of allowed HTTP methods
	CORSAllowedHeaders   string `koanf:"cors_allowed_headers"`   // Comma-separated list of allowed headers
	CORSAllowCredentials bool   `koanf:"cors_allow_credentials"` // Allow credentials (cookies, auth headers)
	CORSMaxAge           int    `koanf:"cors_max_age"`           // Preflight cache duration in seconds
}

// Configuration validation errors.
var (
	ErrMissingDatabaseURL                = errors.New("DATABASE_URL is required")
	ErrMissingJWTSecret                  = errors.New("JWT_SECRET, or JWT_SECRET_CURRENT is required")
	ErrMissingLiveKitURL                 = errors.New("LIVEKIT_URL is required")
	ErrMissingLiveKitAPIKey              = errors.New("LIVEKIT_API_KEY is required")
	ErrMissingLiveKitAPISecret           = errors.New("LIVEKIT_API_SECRET is required")
	ErrMissingStripeAPIKey               = errors.New("STRIPE_API_KEY is required")
	ErrMissingStripeWebhookSecret        = errors.New("STRIPE_WEBHOOK_SECRET is required")
	ErrMissingStripeOnboardingReturnURL  = errors.New("STRIPE_ONBOARDING_RETURN_URL is required")
	ErrMissingStripeOnboardingRefreshURL = errors.New("STRIPE_ONBOARDING_REFRESH_URL is required")
	ErrMissingMapTilerAPIKey             = errors.New("MAPTILER_API_KEY is required")
	ErrMissingJetstreamURL               = errors.New("JETSTREAM_URL is required")
	ErrMissingR2BucketName               = errors.New("R2_BUCKET_NAME is required")
	ErrMissingR2AccessKeyID              = errors.New("R2_ACCESS_KEY_ID is required")
	ErrMissingR2SecretAccessKey          = errors.New("R2_SECRET_ACCESS_KEY is required")
	ErrMissingR2Endpoint                 = errors.New("R2_ENDPOINT is required")
	ErrInvalidPort                       = errors.New("PORT must be a valid integer")
	ErrJWTSecretTooShort                 = errors.New("JWT secret must be at least 32 bytes")
)

// Default values for non-secret configuration.
const (
	DefaultPort                        = 8080
	DefaultEnv                         = "development"
	DefaultR2MaxUploadSizeMB           = 15
	DefaultRankTrustEnabled            = false
	DefaultStripeApplicationFeePercent = 5.0 // 5% platform fee by default
	DefaultCanaryEnabled               = false
	DefaultCanaryTrafficPercent        = 5.0  // Start with 5% canary traffic
	DefaultCanaryErrorThreshold        = 1.0  // 1% error rate triggers rollback
	DefaultCanaryLatencyThreshold      = 2.0  // 2 seconds p95 latency threshold
	DefaultCanaryAutoRollback          = true // Auto-rollback enabled by default
	DefaultCanaryMonitoringWindow      = 300  // 5 minutes monitoring window
	DefaultCanaryVersion               = "canary"
	DefaultTracingEnabled              = false
	DefaultTracingExporterType         = "otlp-http"
	DefaultTracingSampleRate           = 0.1   // 10% sampling in production
	DefaultTracingInsecure             = false
	DefaultProfilingEnabled            = false // NEVER enable in production (security risk)
	DefaultCORSAllowedOrigins          = ""    // Empty means CORS is disabled
	DefaultCORSAllowedMethods          = "GET,POST,PUT,PATCH,DELETE,OPTIONS"                // Standard REST methods
	DefaultCORSAllowedHeaders          = "Content-Type,Authorization,X-Request-ID"          // Essential headers
	DefaultCORSAllowCredentials        = true                                                // Allow cookies/auth by default
	DefaultCORSMaxAge                  = 3600                                                // 1 hour preflight cache
)

// Load reads configuration from environment variables and an optional config file.
// Environment variables take precedence over file values.
// Returns the loaded config and a slice of validation errors (empty if valid).
// If a config file path is provided and the file cannot be loaded, an error is returned.
func Load(configFilePath string) (*Config, []error) {
	k := koanf.New(".")
	var loadErrs []error

	// Load from YAML file first if provided (lower precedence)
	if configFilePath != "" {
		if err := k.Load(file.Provider(configFilePath), yaml.Parser()); err != nil {
			return nil, []error{fmt.Errorf("failed to load config file %s: %w", configFilePath, err)}
		}
	}

	// Parse port from env, collecting error if invalid
	// Try SUBCULT_PORT first, then PORT for backward compatibility
	port, portErr := getEnvIntOrDefaultMulti([]string{"SUBCULT_PORT", "PORT"}, k.Int("port"), DefaultPort)
	if portErr != nil {
		loadErrs = append(loadErrs, portErr)
	}

	// Parse R2 max upload size from env with default
	maxUploadSize, uploadSizeErr := getEnvIntOrDefault("R2_MAX_UPLOAD_SIZE_MB", k.Int("r2_max_upload_size_mb"), DefaultR2MaxUploadSizeMB)
	if uploadSizeErr != nil {
		loadErrs = append(loadErrs, uploadSizeErr)
	}

	// Parse trust ranking feature flag from env with default
	rankTrustEnabled := DefaultRankTrustEnabled
	if k.Exists("rank_trust_enabled") {
		rankTrustEnabled = k.Bool("rank_trust_enabled")
	}
	if val := os.Getenv("RANK_TRUST_ENABLED"); val != "" {
		// Env var takes precedence over file config
		valLower := strings.ToLower(val)
		switch valLower {
		case "true", "1", "yes", "on":
			rankTrustEnabled = true
		case "false", "0", "no", "off":
			rankTrustEnabled = false
		}
	}

	// Parse Stripe application fee percentage with default
	stripeFeePercent := DefaultStripeApplicationFeePercent
	if k.Exists("stripe_application_fee_percent") {
		stripeFeePercent = k.Float64("stripe_application_fee_percent")
	}
	if feePercentStr := os.Getenv("STRIPE_APPLICATION_FEE_PERCENT"); feePercentStr != "" {
		// Env var takes precedence over file config
		parsed, err := strconv.ParseFloat(feePercentStr, 64)
		if err != nil {
			loadErrs = append(loadErrs, fmt.Errorf("STRIPE_APPLICATION_FEE_PERCENT must be a valid float: %w", err))
		} else {
			stripeFeePercent = parsed
		}
	}

	// Parse tracing configuration
	tracingEnabled := DefaultTracingEnabled
	if k.Exists("tracing_enabled") {
		tracingEnabled = k.Bool("tracing_enabled")
	}
	if val := os.Getenv("TRACING_ENABLED"); val != "" {
		valLower := strings.ToLower(val)
		switch valLower {
		case "true", "1", "yes", "on":
			tracingEnabled = true
		case "false", "0", "no", "off":
			tracingEnabled = false
		}
	}

	tracingSampleRate := DefaultTracingSampleRate
	if k.Exists("tracing_sample_rate") {
		tracingSampleRate = k.Float64("tracing_sample_rate")
	}
	if sampleRateStr := os.Getenv("TRACING_SAMPLE_RATE"); sampleRateStr != "" {
		parsed, err := strconv.ParseFloat(sampleRateStr, 64)
		if err != nil {
			loadErrs = append(loadErrs, fmt.Errorf("TRACING_SAMPLE_RATE must be a valid float: %w", err))
		} else {
			tracingSampleRate = parsed
		}
	}

	tracingInsecure := DefaultTracingInsecure
	if k.Exists("tracing_insecure") {
		tracingInsecure = k.Bool("tracing_insecure")
	}
	if val := os.Getenv("TRACING_INSECURE"); val != "" {
		valLower := strings.ToLower(val)
		switch valLower {
		case "true", "1", "yes", "on":
			tracingInsecure = true
		case "false", "0", "no", "off":
			tracingInsecure = false
		}
	}

	// Parse profiling configuration (SECURITY: disabled by default, only for development)
	profilingEnabled := DefaultProfilingEnabled
	if k.Exists("profiling_enabled") {
		profilingEnabled = k.Bool("profiling_enabled")
	}
	if val := os.Getenv("PROFILING_ENABLED"); val != "" {
		valLower := strings.ToLower(val)
		switch valLower {
		case "true", "1", "yes", "on":
			profilingEnabled = true
		case "false", "0", "no", "off":
			profilingEnabled = false
		}
	}

	// Parse canary deployment configuration
	canaryEnabled := DefaultCanaryEnabled
	if k.Exists("canary_enabled") {
		canaryEnabled = k.Bool("canary_enabled")
	}
	if val := os.Getenv("CANARY_ENABLED"); val != "" {
		valLower := strings.ToLower(val)
		switch valLower {
		case "true", "1", "yes", "on":
			canaryEnabled = true
		case "false", "0", "no", "off":
			canaryEnabled = false
		}
	}

	canaryTrafficPercent := DefaultCanaryTrafficPercent
	if k.Exists("canary_traffic_percent") {
		canaryTrafficPercent = k.Float64("canary_traffic_percent")
	}
	if val := os.Getenv("CANARY_TRAFFIC_PERCENT"); val != "" {
		if parsed, err := strconv.ParseFloat(val, 64); err == nil {
			canaryTrafficPercent = parsed
		} else {
			loadErrs = append(loadErrs, fmt.Errorf("CANARY_TRAFFIC_PERCENT must be a valid float: %w", err))
		}
	}

	canaryErrorThreshold := DefaultCanaryErrorThreshold
	if k.Exists("canary_error_threshold") {
		canaryErrorThreshold = k.Float64("canary_error_threshold")
	}
	if val := os.Getenv("CANARY_ERROR_THRESHOLD"); val != "" {
		if parsed, err := strconv.ParseFloat(val, 64); err == nil {
			canaryErrorThreshold = parsed
		} else {
			loadErrs = append(loadErrs, fmt.Errorf("CANARY_ERROR_THRESHOLD must be a valid float: %w", err))
		}
	}

	canaryLatencyThreshold := DefaultCanaryLatencyThreshold
	if k.Exists("canary_latency_threshold") {
		canaryLatencyThreshold = k.Float64("canary_latency_threshold")
	}
	if val := os.Getenv("CANARY_LATENCY_THRESHOLD"); val != "" {
		if parsed, err := strconv.ParseFloat(val, 64); err == nil {
			canaryLatencyThreshold = parsed
		} else {
			loadErrs = append(loadErrs, fmt.Errorf("CANARY_LATENCY_THRESHOLD must be a valid float: %w", err))
		}
	}

	canaryAutoRollback := DefaultCanaryAutoRollback
	if k.Exists("canary_auto_rollback") {
		canaryAutoRollback = k.Bool("canary_auto_rollback")
	}
	if val := os.Getenv("CANARY_AUTO_ROLLBACK"); val != "" {
		valLower := strings.ToLower(val)
		switch valLower {
		case "true", "1", "yes", "on":
			canaryAutoRollback = true
		case "false", "0", "no", "off":
			canaryAutoRollback = false
		}
	}

	canaryMonitoringWindow := DefaultCanaryMonitoringWindow
	if k.Exists("canary_monitoring_window") {
		canaryMonitoringWindow = k.Int("canary_monitoring_window")
	}
	if val := os.Getenv("CANARY_MONITORING_WINDOW"); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil {
			canaryMonitoringWindow = parsed
		} else {
			loadErrs = append(loadErrs, fmt.Errorf("CANARY_MONITORING_WINDOW must be a valid integer: %w", err))
		}
	}

	canaryVersion := DefaultCanaryVersion
	if k.Exists("canary_version") {
		canaryVersion = k.String("canary_version")
	}
	if val := os.Getenv("CANARY_VERSION"); val != "" {
		canaryVersion = val
	}

	// Parse CORS configuration
	corsAllowedOrigins := getEnvOrDefault("CORS_ALLOWED_ORIGINS", k.String("cors_allowed_origins"), DefaultCORSAllowedOrigins)
	corsAllowedMethods := getEnvOrDefault("CORS_ALLOWED_METHODS", k.String("cors_allowed_methods"), DefaultCORSAllowedMethods)
	corsAllowedHeaders := getEnvOrDefault("CORS_ALLOWED_HEADERS", k.String("cors_allowed_headers"), DefaultCORSAllowedHeaders)

	corsAllowCredentials := DefaultCORSAllowCredentials
	if k.Exists("cors_allow_credentials") {
		corsAllowCredentials = k.Bool("cors_allow_credentials")
	}
	if val := os.Getenv("CORS_ALLOW_CREDENTIALS"); val != "" {
		valLower := strings.ToLower(val)
		switch valLower {
		case "true", "1", "yes", "on":
			corsAllowCredentials = true
		case "false", "0", "no", "off":
			corsAllowCredentials = false
		}
	}

	corsMaxAge, corsMaxAgeErr := getEnvIntOrDefault("CORS_MAX_AGE", k.Int("cors_max_age"), DefaultCORSMaxAge)
	if corsMaxAgeErr != nil {
		loadErrs = append(loadErrs, corsMaxAgeErr)
	}

	// Build config struct, with env vars taking precedence over file values
	cfg := &Config{
		Port:                        port,
		Env:                         getEnvOrDefaultMulti([]string{"SUBCULT_ENV", "ENV", "GO_ENV"}, k.String("env"), DefaultEnv),
		DatabaseURL:                 getEnvOrKoanf("DATABASE_URL", k, "database_url"),
		JWTSecret:                   getEnvOrKoanf("JWT_SECRET", k, "jwt_secret"),
		JWTSecretCurrent:            getEnvOrKoanf("JWT_SECRET_CURRENT", k, "jwt_secret_current"),
		JWTSecretPrevious:           getEnvOrKoanf("JWT_SECRET_PREVIOUS", k, "jwt_secret_previous"),
		LiveKitURL:                  getEnvOrKoanf("LIVEKIT_URL", k, "livekit_url"),
		LiveKitAPIKey:               getEnvOrKoanf("LIVEKIT_API_KEY", k, "livekit_api_key"),
		LiveKitAPISecret:            getEnvOrKoanf("LIVEKIT_API_SECRET", k, "livekit_api_secret"),
		StripeAPIKey:                getEnvOrKoanf("STRIPE_API_KEY", k, "stripe_api_key"),
		StripeWebhookSecret:         getEnvOrKoanf("STRIPE_WEBHOOK_SECRET", k, "stripe_webhook_secret"),
		StripeOnboardingReturnURL:   getEnvOrKoanf("STRIPE_ONBOARDING_RETURN_URL", k, "stripe_onboarding_return_url"),
		StripeOnboardingRefreshURL:  getEnvOrKoanf("STRIPE_ONBOARDING_REFRESH_URL", k, "stripe_onboarding_refresh_url"),
		StripeApplicationFeePercent: stripeFeePercent,
		MapTilerAPIKey:              getEnvOrKoanf("MAPTILER_API_KEY", k, "maptiler_api_key"),
		JetstreamURL:                getEnvOrKoanf("JETSTREAM_URL", k, "jetstream_url"),
		R2BucketName:                getEnvOrKoanf("R2_BUCKET_NAME", k, "r2_bucket_name"),
		R2AccessKeyID:               getEnvOrKoanf("R2_ACCESS_KEY_ID", k, "r2_access_key_id"),
		R2SecretAccessKey:           getEnvOrKoanf("R2_SECRET_ACCESS_KEY", k, "r2_secret_access_key"),
		R2Endpoint:                  getEnvOrKoanf("R2_ENDPOINT", k, "r2_endpoint"),
		R2MaxUploadSizeMB:           maxUploadSize,
		RedisURL:                    getEnvOrKoanf("REDIS_URL", k, "redis_url"),
		InternalServiceToken:        getEnvOrKoanf("INTERNAL_SERVICE_TOKEN", k, "internal_service_token"),
		RankTrustEnabled:            rankTrustEnabled,
		CanaryEnabled:               canaryEnabled,
		CanaryTrafficPercent:        canaryTrafficPercent,
		CanaryErrorThreshold:        canaryErrorThreshold,
		CanaryLatencyThreshold:      canaryLatencyThreshold,
		CanaryAutoRollback:          canaryAutoRollback,
		CanaryMonitoringWindow:      canaryMonitoringWindow,
		CanaryVersion:               canaryVersion,
		TracingEnabled:              tracingEnabled,
		TracingExporterType:         getEnvOrDefault("TRACING_EXPORTER_TYPE", k.String("tracing_exporter_type"), DefaultTracingExporterType),
		TracingOTLPEndpoint:         getEnvOrKoanf("TRACING_OTLP_ENDPOINT", k, "tracing_otlp_endpoint"),
		TracingSampleRate:           tracingSampleRate,
		TracingInsecure:             tracingInsecure,
		ProfilingEnabled:            profilingEnabled,
		CORSAllowedOrigins:          corsAllowedOrigins,
		CORSAllowedMethods:          corsAllowedMethods,
		CORSAllowedHeaders:          corsAllowedHeaders,
		CORSAllowCredentials:        corsAllowCredentials,
		CORSMaxAge:                  corsMaxAge,
	}

	// Validate and collect errors
	errs := cfg.Validate()
	errs = append(loadErrs, errs...)

	return cfg, errs
}

// getEnvOrKoanf returns the environment variable value if set, otherwise the koanf value.
func getEnvOrKoanf(envKey string, k *koanf.Koanf, koanfKey string) string {
	if val := os.Getenv(envKey); val != "" {
		return val
	}
	return k.String(koanfKey)
}

// getEnvOrDefault returns the environment variable value if set, otherwise the koanf value, or default.
func getEnvOrDefault(envKey string, koanfVal string, defaultVal string) string {
	if val := os.Getenv(envKey); val != "" {
		return val
	}
	if koanfVal != "" {
		return koanfVal
	}
	return defaultVal
}

// getEnvOrDefaultMulti tries multiple environment variable keys in order.
// Returns the first non-empty value found, otherwise the koanf value, or default.
func getEnvOrDefaultMulti(envKeys []string, koanfVal string, defaultVal string) string {
	for _, key := range envKeys {
		if val := os.Getenv(key); val != "" {
			return val
		}
	}
	if koanfVal != "" {
		return koanfVal
	}
	return defaultVal
}

// getEnvIntOrDefault returns the environment variable as int if set, otherwise the koanf value, or default.
// Returns an error if the environment variable is set but cannot be parsed as an integer.
// Note: A port value of 0 from a YAML file will fall back to the default; port 0 is not supported in YAML files.
func getEnvIntOrDefault(envKey string, koanfVal int, defaultVal int) (int, error) {
	if val := os.Getenv(envKey); val != "" {
		i, err := strconv.Atoi(val)
		if err != nil {
			return 0, fmt.Errorf("%s must be a valid integer: %w", envKey, ErrInvalidPort)
		}
		return i, nil
	}
	if koanfVal != 0 {
		return koanfVal, nil
	}
	return defaultVal, nil
}

// getEnvIntOrDefaultMulti tries multiple environment variable keys in order.
// Returns the first valid integer value found, otherwise the koanf value, or default.
// Returns an error if any environment variable is set but cannot be parsed as an integer.
func getEnvIntOrDefaultMulti(envKeys []string, koanfVal int, defaultVal int) (int, error) {
	for _, key := range envKeys {
		if val := os.Getenv(key); val != "" {
			i, err := strconv.Atoi(val)
			if err != nil {
				return 0, fmt.Errorf("%s must be a valid integer: %w", key, ErrInvalidPort)
			}
			return i, nil
		}
	}
	if koanfVal != 0 {
		return koanfVal, nil
	}
	return defaultVal, nil
}

// Validate checks that all required configuration values are present.
// Returns a slice of validation errors (empty if valid).
func (c *Config) Validate() []error {
	var errs []error

	if c.DatabaseURL == "" {
		errs = append(errs, ErrMissingDatabaseURL)
	}
	// JWT secret validation: require either legacy JWT_SECRET or JWT_SECRET_CURRENT
	if c.JWTSecret == "" && c.JWTSecretCurrent == "" {
		errs = append(errs, ErrMissingJWTSecret)
	} else {
		// Validate minimum length for whichever secret is active
		activeSecret := c.JWTSecretCurrent
		if activeSecret == "" {
			activeSecret = c.JWTSecret
		}
		if len(activeSecret) < 32 {
			errs = append(errs, ErrJWTSecretTooShort)
		}
	}
	if c.LiveKitURL == "" {
		errs = append(errs, ErrMissingLiveKitURL)
	}
	if c.LiveKitAPIKey == "" {
		errs = append(errs, ErrMissingLiveKitAPIKey)
	}
	if c.LiveKitAPISecret == "" {
		errs = append(errs, ErrMissingLiveKitAPISecret)
	}
	if c.StripeAPIKey == "" {
		errs = append(errs, ErrMissingStripeAPIKey)
	}
	if c.StripeWebhookSecret == "" {
		errs = append(errs, ErrMissingStripeWebhookSecret)
	}
	if c.StripeOnboardingReturnURL == "" {
		errs = append(errs, ErrMissingStripeOnboardingReturnURL)
	}
	if c.StripeOnboardingRefreshURL == "" {
		errs = append(errs, ErrMissingStripeOnboardingRefreshURL)
	}
	if c.MapTilerAPIKey == "" {
		errs = append(errs, ErrMissingMapTilerAPIKey)
	}
	if c.JetstreamURL == "" {
		errs = append(errs, ErrMissingJetstreamURL)
	}

	// R2 configuration is optional. Only validate fields if any R2 value is set.
	if c.R2BucketName != "" || c.R2AccessKeyID != "" || c.R2SecretAccessKey != "" || c.R2Endpoint != "" {
		if c.R2BucketName == "" {
			errs = append(errs, ErrMissingR2BucketName)
		}
		if c.R2AccessKeyID == "" {
			errs = append(errs, ErrMissingR2AccessKeyID)
		}
		if c.R2SecretAccessKey == "" {
			errs = append(errs, ErrMissingR2SecretAccessKey)
		}
		if c.R2Endpoint == "" {
			errs = append(errs, ErrMissingR2Endpoint)
		}
	}

	return errs
}

// LogSummary returns a summary of the configuration suitable for logging.
// All secrets are masked to prevent accidental exposure.
func (c *Config) LogSummary() map[string]string {
	return map[string]string{
		"port":                          fmt.Sprintf("%d", c.Port),
		"env":                           c.Env,
		"database_url":                  maskDatabaseURL(c.DatabaseURL),
		"jwt_secret":                    maskSecret(c.JWTSecret),
		"jwt_secret_current":            maskSecret(c.JWTSecretCurrent),
		"jwt_secret_previous":           maskSecret(c.JWTSecretPrevious),
		"livekit_url":                   c.LiveKitURL,
		"livekit_api_key":               maskSecret(c.LiveKitAPIKey),
		"livekit_api_secret":            maskSecret(c.LiveKitAPISecret),
		"stripe_api_key":                maskStripeKey(c.StripeAPIKey),
		"stripe_webhook_secret":         maskSecret(c.StripeWebhookSecret),
		"stripe_onboarding_return_url":  c.StripeOnboardingReturnURL,
		"stripe_onboarding_refresh_url": c.StripeOnboardingRefreshURL,
		"maptiler_api_key":              maskSecret(c.MapTilerAPIKey),
		"jetstream_url":                 c.JetstreamURL,
		"r2_bucket_name":                c.R2BucketName,
		"r2_access_key_id":              maskSecret(c.R2AccessKeyID),
		"r2_secret_access_key":          maskSecret(c.R2SecretAccessKey),
		"r2_endpoint":                   c.R2Endpoint,
		"r2_max_upload_size_mb":         fmt.Sprintf("%d", c.R2MaxUploadSizeMB),
		"redis_url":                     maskDatabaseURL(c.RedisURL),
		"internal_service_token":        maskSecret(c.InternalServiceToken),
		"rank_trust_enabled":            fmt.Sprintf("%t", c.RankTrustEnabled),
		"canary_enabled":                fmt.Sprintf("%t", c.CanaryEnabled),
		"canary_traffic_percent":        fmt.Sprintf("%.2f", c.CanaryTrafficPercent),
		"canary_error_threshold":        fmt.Sprintf("%.2f", c.CanaryErrorThreshold),
		"canary_latency_threshold":      fmt.Sprintf("%.2f", c.CanaryLatencyThreshold),
		"canary_auto_rollback":          fmt.Sprintf("%t", c.CanaryAutoRollback),
		"canary_monitoring_window":      fmt.Sprintf("%d", c.CanaryMonitoringWindow),
		"canary_version":                c.CanaryVersion,
		"tracing_enabled":               fmt.Sprintf("%t", c.TracingEnabled),
		"tracing_exporter_type":         c.TracingExporterType,
		"tracing_otlp_endpoint":         c.TracingOTLPEndpoint,
		"tracing_sample_rate":           fmt.Sprintf("%.2f", c.TracingSampleRate),
		"tracing_insecure":              fmt.Sprintf("%t", c.TracingInsecure),
		"profiling_enabled":             fmt.Sprintf("%t", c.ProfilingEnabled),
		"cors_allowed_origins":          c.CORSAllowedOrigins,
		"cors_allowed_methods":          c.CORSAllowedMethods,
		"cors_allowed_headers":          c.CORSAllowedHeaders,
		"cors_allow_credentials":        fmt.Sprintf("%t", c.CORSAllowCredentials),
		"cors_max_age":                  fmt.Sprintf("%d", c.CORSMaxAge),
	}
}

// maskSecret masks a secret value, showing only the last 4 characters preceded by ****
// If the secret is shorter than 8 characters, it's fully masked.
func maskSecret(s string) string {
	if s == "" {
		return "<not set>"
	}
	if len(s) < 8 {
		return "****"
	}
	return "****" + s[len(s)-4:]
}

// maskStripeKey masks a Stripe API key, preserving the prefix (sk_live_, sk_test_, etc.)
func maskStripeKey(s string) string {
	if s == "" {
		return "<not set>"
	}

	// Stripe keys have format like sk_live_..., sk_test_..., pk_live_..., etc.
	parts := strings.SplitN(s, "_", 3)
	if len(parts) == 3 {
		return parts[0] + "_" + parts[1] + "_****"
	}

	// Fallback to generic masking
	return maskSecret(s)
}

// maskDatabaseURL masks the password in a database URL.
// Supports both postgres:// and postgresql:// schemes.
func maskDatabaseURL(s string) string {
	if s == "" {
		return "<not set>"
	}

	// Look for password pattern: user:password@host
	// Simple approach: find :// and then mask between : and @
	schemeEnd := strings.Index(s, "://")
	if schemeEnd == -1 {
		return maskSecret(s)
	}

	rest := s[schemeEnd+3:]
	atIndex := strings.Index(rest, "@")
	if atIndex == -1 {
		return s // No credentials in URL
	}

	colonIndex := strings.Index(rest[:atIndex], ":")
	if colonIndex == -1 {
		return s // No password (only username)
	}

	// Reconstruct URL with masked password
	scheme := s[:schemeEnd+3]
	user := rest[:colonIndex]
	hostAndPath := rest[atIndex:]

	return scheme + user + ":****" + hostAndPath
}

// GetJWTSecrets returns the current and previous JWT secrets for rotation support.
// Returns (currentSecret, previousSecret).
// For backward compatibility, if JWT_SECRET is set and JWT_SECRET_CURRENT is not,
// JWT_SECRET is used as the current secret.
func (c *Config) GetJWTSecrets() (current, previous string) {
	// Prefer JWT_SECRET_CURRENT if set
	if c.JWTSecretCurrent != "" {
		return c.JWTSecretCurrent, c.JWTSecretPrevious
	}
	// Fallback to legacy JWT_SECRET
	return c.JWTSecret, ""
}

// LogValue implements slog.LogValuer so that Config is automatically safe to log.
// All sensitive fields are masked; non-sensitive fields are included as-is.
// This prevents accidental secret exposure when logging the config struct, e.g.:
//
//	slog.Info("config loaded", "config", cfg)
func (c *Config) LogValue() slog.Value {
	return slog.GroupValue(
		// Core environment & server
		slog.Int("port", c.Port),
		slog.String("env", c.Env),

		// Databases & caches (URLs masked)
		slog.String("database_url", maskDatabaseURL(c.DatabaseURL)),
		slog.String("redis_url", maskDatabaseURL(c.RedisURL)),

		// JWT secrets (masked)
		slog.String("jwt_secret", maskSecret(c.JWTSecret)),
		slog.String("jwt_secret_current", maskSecret(c.JWTSecretCurrent)),
		slog.String("jwt_secret_previous", maskSecret(c.JWTSecretPrevious)),

		// LiveKit streaming (URL visible, credentials masked)
		slog.String("livekit_url", c.LiveKitURL),
		slog.String("livekit_api_key", maskSecret(c.LiveKitAPIKey)),
		slog.String("livekit_api_secret", maskSecret(c.LiveKitAPISecret)),

		// Stripe (non-secret config visible, secrets masked)
		slog.String("stripe_api_key", maskStripeKey(c.StripeAPIKey)),
		slog.String("stripe_webhook_secret", maskSecret(c.StripeWebhookSecret)),
		slog.String("stripe_onboarding_return_url", c.StripeOnboardingReturnURL),
		slog.String("stripe_onboarding_refresh_url", c.StripeOnboardingRefreshURL),
		slog.Float64("stripe_application_fee_percent", c.StripeApplicationFeePercent),

		// MapTiler (API key masked)
		slog.String("maptiler_api_key", maskSecret(c.MapTilerAPIKey)),

		// Cloudflare R2 (identifiers visible, credentials masked)
		slog.String("r2_access_key_id", maskSecret(c.R2AccessKeyID)),
		slog.String("r2_secret_access_key", maskSecret(c.R2SecretAccessKey)),
		slog.String("r2_bucket_name", c.R2BucketName),
		slog.String("r2_endpoint", c.R2Endpoint),
		slog.Int("r2_max_upload_size_mb", c.R2MaxUploadSizeMB),

		// Internal services
		slog.String("internal_service_token", maskSecret(c.InternalServiceToken)),
		slog.String("jetstream_url", c.JetstreamURL),

		// Feature flags / behavior toggles
		slog.Bool("rank_trust_enabled", c.RankTrustEnabled),
		slog.Bool("profiling_enabled", c.ProfilingEnabled),

		// Canary configuration (non-secret, operational visibility)
		slog.Bool("canary_enabled", c.CanaryEnabled),
		slog.Float64("canary_traffic_percent", c.CanaryTrafficPercent),
		slog.Float64("canary_error_threshold", c.CanaryErrorThreshold),
		slog.Float64("canary_latency_threshold", c.CanaryLatencyThreshold),
		slog.Bool("canary_auto_rollback", c.CanaryAutoRollback),
		slog.Int("canary_monitoring_window", c.CanaryMonitoringWindow),
		slog.String("canary_version", c.CanaryVersion),

		// Tracing configuration (non-secret)
		slog.Bool("tracing_enabled", c.TracingEnabled),
		slog.String("tracing_exporter_type", c.TracingExporterType),
		slog.String("tracing_otlp_endpoint", c.TracingOTLPEndpoint),
		slog.Float64("tracing_sample_rate", c.TracingSampleRate),
		slog.Bool("tracing_insecure", c.TracingInsecure),

		// CORS configuration (important for security visibility)
		slog.String("cors_allowed_origins", c.CORSAllowedOrigins),
		slog.String("cors_allowed_methods", c.CORSAllowedMethods),
		slog.String("cors_allowed_headers", c.CORSAllowedHeaders),
		slog.Bool("cors_allow_credentials", c.CORSAllowCredentials),
		slog.Int("cors_max_age", c.CORSMaxAge),
	)
}
