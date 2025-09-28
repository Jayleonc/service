package telemetry

import (
	"context"
	"sync"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/semconv/v1.24.0"
)

// Config controls OTLP tracing initialisation.
type Config struct {
	ServiceName string
	Endpoint    string
	Enabled     bool
}

// Provider wraps the tracer provider shutdown hook.
type Provider struct {
	shutdown func(context.Context) error
}

var (
	mu       sync.RWMutex
	provider *Provider
)

// NewProvider configures tracing based on cfg.
func NewProvider(ctx context.Context, cfg Config) (*Provider, error) {
	if !cfg.Enabled {
		return &Provider{}, nil
	}

	exporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithEndpoint(cfg.Endpoint), otlptracegrpc.WithInsecure())
	if err != nil {
		return nil, err
	}

	res, err := resource.Merge(resource.Default(), resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceName(cfg.ServiceName),
	))
	if err != nil {
		return nil, err
	}

	tp := trace.NewTracerProvider(
		trace.WithBatcher(exporter),
		trace.WithResource(res),
	)
	otel.SetTracerProvider(tp)

	return &Provider{shutdown: tp.Shutdown}, nil
}

// Init constructs a provider and stores it globally.
func Init(ctx context.Context, cfg Config) (*Provider, error) {
	p, err := NewProvider(ctx, cfg)
	if err != nil {
		return nil, err
	}
	SetDefault(p)
	return p, nil
}

// SetDefault records p as the global telemetry provider.
func SetDefault(p *Provider) {
	mu.Lock()
	defer mu.Unlock()
	provider = p
}

// Default returns the global telemetry provider.
func Default() *Provider {
	mu.RLock()
	defer mu.RUnlock()
	return provider
}

// Shutdown gracefully stops the underlying tracer provider.
func (p *Provider) Shutdown(ctx context.Context) error {
	if p == nil || p.shutdown == nil {
		return nil
	}
	return p.shutdown(ctx)
}
