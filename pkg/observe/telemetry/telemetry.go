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

// Config 用于控制 OTLP 链路追踪的初始化行为。
type Config struct {
	ServiceName string
	Endpoint    string
	Enabled     bool
}

// Provider 封装追踪提供器的关闭逻辑。
type Provider struct {
	shutdown func(context.Context) error
}

var (
	mu       sync.RWMutex
	provider *Provider
)

// NewProvider 根据配置初始化链路追踪。
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

// Init 构造追踪提供器并注册为全局实例。
func Init(ctx context.Context, cfg Config) (*Provider, error) {
	p, err := NewProvider(ctx, cfg)
	if err != nil {
		return nil, err
	}
	SetDefault(p)
	return p, nil
}

// SetDefault 将 p 记录为全局默认的链路追踪提供器。
func SetDefault(p *Provider) {
	mu.Lock()
	defer mu.Unlock()
	provider = p
}

// Default 返回当前的全局链路追踪提供器。
func Default() *Provider {
	mu.RLock()
	defer mu.RUnlock()
	return provider
}

// Shutdown 优雅地关闭底层的追踪提供器。
func (p *Provider) Shutdown(ctx context.Context) error {
	if p == nil || p.shutdown == nil {
		return nil
	}
	return p.shutdown(ctx)
}
