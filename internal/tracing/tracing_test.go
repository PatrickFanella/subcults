package tracing

import (
	"context"
	"testing"
	"time"
)

func TestNewProvider_Disabled(t *testing.T) {
	cfg := Config{
		ServiceName: "test-service",
		Enabled:     false,
	}

	provider, err := NewProvider(cfg)
	if err != nil {
		t.Fatalf("expected no error for disabled tracing, got %v", err)
	}

	if provider == nil {
		t.Fatal("expected provider to be non-nil")
	}

	if provider.IsEnabled() {
		t.Error("expected tracing to be disabled")
	}
}

func TestNewProvider_MissingServiceName(t *testing.T) {
	cfg := Config{
		Enabled:      true,
		SamplingRate: 0.1,
	}

	_, err := NewProvider(cfg)
	if err == nil {
		t.Fatal("expected error for missing service name")
	}
}

func TestNewProvider_InvalidSamplingRate(t *testing.T) {
	tests := []struct {
		name string
		rate float64
	}{
		{"negative", -0.1},
		{"greater than 1", 1.5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{
				ServiceName:  "test-service",
				Enabled:      true,
				SamplingRate: tt.rate,
			}

			_, err := NewProvider(cfg)
			if err == nil {
				t.Fatalf("expected error for sampling rate %f", tt.rate)
			}
		})
	}
}

func TestNewProvider_ValidConfig(t *testing.T) {
	tests := []struct {
		name         string
		exporterType string
		samplingRate float64
		endpoint     string
		insecure     bool
	}{
		{
			name:         "otlp-http with 10% sampling",
			exporterType: "otlp-http",
			samplingRate: 0.1,
			endpoint:     "localhost:4318",
			insecure:     true,
		},
		{
			name:         "otlp-grpc with 100% sampling",
			exporterType: "otlp-grpc",
			samplingRate: 1.0,
			endpoint:     "localhost:4317",
			insecure:     true,
		},
		{
			name:         "default exporter with 0% sampling",
			exporterType: "",
			samplingRate: 0.0,
			endpoint:     "",
			insecure:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{
				ServiceName:  "test-service",
				Enabled:      true,
				Environment:  "test",
				ExporterType: tt.exporterType,
				OTLPEndpoint: tt.endpoint,
				SamplingRate: tt.samplingRate,
				InsecureMode: tt.insecure,
			}

			provider, err := NewProvider(cfg)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !provider.IsEnabled() {
				t.Error("expected tracing to be enabled")
			}

			// Test shutdown
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			if err := provider.Shutdown(ctx); err != nil {
				t.Errorf("unexpected shutdown error: %v", err)
			}
		})
	}
}

func TestNewProvider_UnsupportedExporter(t *testing.T) {
	cfg := Config{
		ServiceName:  "test-service",
		Enabled:      true,
		ExporterType: "unsupported",
		SamplingRate: 0.1,
	}

	_, err := NewProvider(cfg)
	if err == nil {
		t.Fatal("expected error for unsupported exporter type")
	}
}

func TestProvider_Tracer(t *testing.T) {
	cfg := Config{
		ServiceName:  "test-service",
		Enabled:      true,
		Environment:  "test",
		ExporterType: "otlp-http",
		SamplingRate: 1.0,
		InsecureMode: true,
	}

	provider, err := NewProvider(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = provider.Shutdown(ctx)
	}()

	tracer := provider.Tracer("test-tracer")
	if tracer == nil {
		t.Fatal("expected non-nil tracer")
	}

	// Test that tracer can create spans
	ctx := context.Background()
	_, span := tracer.Start(ctx, "test-span")
	if span == nil {
		t.Fatal("expected non-nil span")
	}
	span.End()
}

func TestProvider_Shutdown_Nil(t *testing.T) {
	provider := &Provider{}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Should not error on nil tp
	if err := provider.Shutdown(ctx); err != nil {
		t.Errorf("unexpected error on shutdown with nil tp: %v", err)
	}
}
