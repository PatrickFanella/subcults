// Package tracing provides OpenTelemetry distributed tracing setup and utilities
// for the Subcults API server.
package tracing

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"
)

// Config holds the configuration for distributed tracing.
type Config struct {
	// ServiceName identifies this service in traces
	ServiceName string

	// Enabled controls whether tracing is active
	Enabled bool

	// Environment (development, staging, production)
	Environment string

	// ExporterType determines which exporter to use (otlp-grpc, otlp-http)
	ExporterType string

	// OTLPEndpoint is the endpoint for OTLP exporter (HTTP or gRPC)
	OTLPEndpoint string

	// SamplingRate is the fraction of traces to sample (0.0 to 1.0)
	// e.g., 0.1 means 10% of traces will be sampled
	SamplingRate float64

	// InsecureMode disables TLS for OTLP connection (dev only)
	InsecureMode bool
}

// Provider manages the OpenTelemetry tracer provider.
type Provider struct {
	tp     *sdktrace.TracerProvider
	config Config
}

// NewProvider creates and configures a new OpenTelemetry tracer provider.
func NewProvider(cfg Config) (*Provider, error) {
	if !cfg.Enabled {
		slog.Info("tracing disabled")
		return &Provider{config: cfg}, nil
	}

	// Validate config
	if cfg.ServiceName == "" {
		return nil, fmt.Errorf("service name is required")
	}
	if cfg.SamplingRate < 0 || cfg.SamplingRate > 1 {
		return nil, fmt.Errorf("sampling rate must be between 0 and 1, got %f", cfg.SamplingRate)
	}

	// Create resource with service information
	res, err := resource.New(
		context.Background(),
		resource.WithAttributes(
			semconv.ServiceName(cfg.ServiceName),
			semconv.ServiceVersion("0.0.1"),
			attribute.String("environment", cfg.Environment),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Create exporter based on configuration
	var exporter sdktrace.SpanExporter
	switch cfg.ExporterType {
	case "otlp-grpc":
		exporter, err = createOTLPGRPCExporter(cfg)
	case "otlp-http", "":
		exporter, err = createOTLPHTTPExporter(cfg)
	default:
		return nil, fmt.Errorf("unsupported exporter type: %s", cfg.ExporterType)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to create exporter: %w", err)
	}

	// Create sampler based on configuration
	var sampler sdktrace.Sampler
	if cfg.SamplingRate == 1.0 {
		sampler = sdktrace.AlwaysSample()
	} else if cfg.SamplingRate == 0.0 {
		sampler = sdktrace.NeverSample()
	} else {
		sampler = sdktrace.TraceIDRatioBased(cfg.SamplingRate)
	}

	// Create tracer provider with batch span processor
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sampler),
		sdktrace.WithBatcher(exporter,
			sdktrace.WithBatchTimeout(5*time.Second),
			sdktrace.WithMaxExportBatchSize(512),
		),
	)

	// Set global tracer provider
	otel.SetTracerProvider(tp)

	// Set global propagator to W3C Trace Context
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	slog.Info("tracing initialized",
		"service", cfg.ServiceName,
		"exporter", cfg.ExporterType,
		"endpoint", maskEndpoint(cfg.OTLPEndpoint),
		"sampling_rate", cfg.SamplingRate,
		"environment", cfg.Environment,
	)

	return &Provider{
		tp:     tp,
		config: cfg,
	}, nil
}

// maskEndpoint masks sensitive information in OTLP endpoint URLs.
// It redacts userinfo (username:password) if present in the URL.
func maskEndpoint(endpoint string) string {
	if endpoint == "" {
		return "<not set>"
	}

	// Try to parse as URL to check for userinfo
	// Handle both full URLs (http://...) and host:port format
	var fullURL string
	if strings.Contains(endpoint, "://") {
		fullURL = endpoint
	} else {
		// Assume http for parsing purposes
		fullURL = "http://" + endpoint
	}

	parsed, err := url.Parse(fullURL)
	if err != nil {
		// If parsing fails, just return the endpoint as-is
		return endpoint
	}

	// Check if there's userinfo (credentials)
	if parsed.User != nil {
		// Mask the password
		username := parsed.User.Username()
		if username != "" {
			// Reconstruct without credentials
			parsed.User = nil
			maskedURL := parsed.String()
			// Remove the http:// prefix if it wasn't in the original
			if !strings.Contains(endpoint, "://") {
				maskedURL = strings.TrimPrefix(maskedURL, "http://")
			}
			return maskedURL + " (credentials redacted)"
		}
	}

	return endpoint
}

// createOTLPHTTPExporter creates an OTLP HTTP exporter.
func createOTLPHTTPExporter(cfg Config) (sdktrace.SpanExporter, error) {
	opts := []otlptracehttp.Option{}

	if cfg.OTLPEndpoint != "" {
		opts = append(opts, otlptracehttp.WithEndpoint(cfg.OTLPEndpoint))
	}

	if cfg.InsecureMode {
		opts = append(opts, otlptracehttp.WithInsecure())
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return otlptracehttp.New(ctx, opts...)
}

// createOTLPGRPCExporter creates an OTLP gRPC exporter.
func createOTLPGRPCExporter(cfg Config) (sdktrace.SpanExporter, error) {
	opts := []otlptracegrpc.Option{}

	if cfg.OTLPEndpoint != "" {
		opts = append(opts, otlptracegrpc.WithEndpoint(cfg.OTLPEndpoint))
	}

	if cfg.InsecureMode {
		opts = append(opts, otlptracegrpc.WithInsecure())
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return otlptracegrpc.New(ctx, opts...)
}

// Shutdown gracefully shuts down the tracer provider, flushing any pending spans.
func (p *Provider) Shutdown(ctx context.Context) error {
	if p.tp == nil {
		return nil
	}

	slog.Info("shutting down tracer provider")
	if err := p.tp.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to shutdown tracer provider: %w", err)
	}
	return nil
}

// Tracer returns a tracer for the given name.
func (p *Provider) Tracer(name string) trace.Tracer {
	if p.tp == nil {
		return otel.Tracer(name)
	}
	return p.tp.Tracer(name)
}

// IsEnabled returns whether tracing is enabled.
func (p *Provider) IsEnabled() bool {
	return p.config.Enabled
}
