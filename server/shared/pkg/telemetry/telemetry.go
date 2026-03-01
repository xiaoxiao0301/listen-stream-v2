// Package telemetry provides full OpenTelemetry integration for Listen Stream services.
// It supports traces (Jaeger via OTLP gRPC), metrics (Prometheus), and context helpers.
//
// Usage:
//
//	p, shutdown, err := telemetry.Init(ctx, &telemetry.Config{
//	    ServiceName:    "auth-svc",
//	    ServiceVersion: "1.0.0",
//	    Environment:    "production",
//	    OTLPEndpoint:   "otel-collector:4317",
//	    Enabled:        true,
//	})
//	defer shutdown(context.Background())
package telemetry

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Config holds telemetry configuration.
type Config struct {
	ServiceName    string
	ServiceVersion string
	Environment    string // "development", "staging", "production"
	OTLPEndpoint   string // gRPC endpoint, e.g. "otel-collector:4317"
	Enabled        bool

	// Deprecated fields kept for backward compatibility
	Endpoint string
}

// Provider wraps TracerProvider and MeterProvider.
type Provider struct {
	tracer        trace.Tracer
	meter         metric.Meter
	traceProvider *sdktrace.TracerProvider
	meterProvider *sdkmetric.MeterProvider
	cfg           *Config
}

// ShutdownFunc flushes and shuts down telemetry providers.
type ShutdownFunc func(context.Context) error

// Init initialises OpenTelemetry trace and metric providers.
// Returns a Provider and a shutdown function that must be deferred.
func Init(ctx context.Context, cfg *Config) (*Provider, ShutdownFunc, error) {
	if cfg.OTLPEndpoint == "" && cfg.Endpoint != "" {
		cfg.OTLPEndpoint = cfg.Endpoint // backward compat
	}
	if cfg.Environment == "" {
		cfg.Environment = "development"
	}
	if !cfg.Enabled || cfg.OTLPEndpoint == "" {
		return newNoopProvider(cfg), func(_ context.Context) error { return nil }, nil
	}

	// Build resource ────────────────────────────────────────────────────────
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(cfg.ServiceName),
			semconv.ServiceVersion(cfg.ServiceVersion),
			semconv.DeploymentEnvironment(cfg.Environment),
			attribute.String("service.namespace", "listen-stream"),
		),
		resource.WithProcessPID(),
		resource.WithHost(),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("build otel resource: %w", err)
	}

	// gRPC connection to OTel Collector ─────────────────────────────────────
	conn, err := grpc.NewClient(cfg.OTLPEndpoint,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("connect to otel collector %s: %w", cfg.OTLPEndpoint, err)
	}
	// conn.Close() is called inside the shutdown func below

	// Trace exporter ─────────────────────────────────────────────────────────
	traceExporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithGRPCConn(conn),
		otlptracegrpc.WithTimeout(10*time.Second),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("create trace exporter: %w", err)
	}

	samplingRate := 0.1 // 10% sampling — override to 1.0 in development
	if cfg.Environment != "production" {
		samplingRate = 1.0
	}
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(traceExporter,
			sdktrace.WithBatchTimeout(5*time.Second),
			sdktrace.WithMaxExportBatchSize(512),
		),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(
			sdktrace.ParentBased(sdktrace.TraceIDRatioBased(samplingRate)),
		),
	)
	otel.SetTracerProvider(tracerProvider)

	// Metrics exporter (Prometheus pull) ─────────────────────────────────────
	metricExporter, err := prometheus.New(
		prometheus.WithNamespace(sanitizeName(cfg.ServiceName)),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("create prometheus exporter: %w", err)
	}
	meterProvider := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(metricExporter),
	)
	otel.SetMeterProvider(meterProvider)

	p := &Provider{
		tracer:        otel.Tracer(cfg.ServiceName),
		meter:         otel.Meter(cfg.ServiceName),
		traceProvider: tracerProvider,
		meterProvider: meterProvider,
		cfg:           cfg,
	}

	shutdown := func(ctx context.Context) error {
		var errs []error
		if err := tracerProvider.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("shutdown trace provider: %w", err))
		}
		if err := meterProvider.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("shutdown metric provider: %w", err))
		}
		if err := conn.Close(); err != nil {
			errs = append(errs, fmt.Errorf("close otel grpc conn: %w", err))
		}
		if len(errs) > 0 {
			return fmt.Errorf("telemetry shutdown errors: %v", errs)
		}
		return nil
	}

	return p, shutdown, nil
}

// ─── Tracer helpers ──────────────────────────────────────────────────────────

// Tracer returns the global OTel tracer.
func (p *Provider) Tracer() trace.Tracer { return p.tracer }

// StartSpan starts a new span and injects it into the returned context.
func (p *Provider) StartSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, Span) {
	ctx, s := p.tracer.Start(ctx, name, opts...)
	return ctx, &otelSpan{span: s}
}

// ─── Meter helpers ───────────────────────────────────────────────────────────

// Meter returns the global OTel meter.
func (p *Provider) Meter() metric.Meter { return p.meter }

// NewHTTPRequestCounter creates a counter for HTTP requests.
func (p *Provider) NewHTTPRequestCounter() (metric.Int64Counter, error) {
	return p.meter.Int64Counter(
		"http_requests_total",
		metric.WithDescription("Total HTTP requests"),
		metric.WithUnit("{request}"),
	)
}

// NewHTTPDurationHistogram creates a histogram for HTTP request durations.
func (p *Provider) NewHTTPDurationHistogram() (metric.Float64Histogram, error) {
	return p.meter.Float64Histogram(
		"http_request_duration_seconds",
		metric.WithDescription("HTTP request latency in seconds"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(
			0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0,
		),
	)
}

// NewActiveConnectionsGauge creates a gauge for active WebSocket connections.
func (p *Provider) NewActiveConnectionsGauge() (metric.Int64UpDownCounter, error) {
	return p.meter.Int64UpDownCounter(
		"websocket_active_connections",
		metric.WithDescription("Active WebSocket connections"),
		metric.WithUnit("{connection}"),
	)
}

// NewCacheHitCounter creates hit/miss counters for a cache layer.
func (p *Provider) NewCacheHitCounter(layer string) (hit metric.Int64Counter, miss metric.Int64Counter, err error) {
	hit, err = p.meter.Int64Counter(
		fmt.Sprintf("cache_%s_hits_total", layer),
		metric.WithDescription(fmt.Sprintf("Cache hits for %s layer", layer)),
	)
	if err != nil {
		return nil, nil, err
	}
	miss, err = p.meter.Int64Counter(
		fmt.Sprintf("cache_%s_misses_total", layer),
		metric.WithDescription(fmt.Sprintf("Cache misses for %s layer", layer)),
	)
	return hit, miss, err
}

// ─── Span interface ──────────────────────────────────────────────────────────

// Span represents a tracing span.
type Span interface {
	End()
	SetAttribute(key string, value interface{})
	SetError(err error)
	AddEvent(name string, attributes map[string]interface{})
	TraceID() string
}

// otelSpan wraps an OTel trace.Span.
type otelSpan struct{ span trace.Span }

func (s *otelSpan) End() { s.span.End() }

func (s *otelSpan) SetAttribute(key string, value interface{}) {
	s.span.SetAttributes(anyAttr(key, value))
}

func (s *otelSpan) SetError(err error) {
	if err != nil {
		s.span.RecordError(err)
		s.span.SetStatus(codes.Error, err.Error())
	}
}

func (s *otelSpan) AddEvent(name string, attributes map[string]interface{}) {
	attrs := make([]attribute.KeyValue, 0, len(attributes))
	for k, v := range attributes {
		attrs = append(attrs, anyAttr(k, v))
	}
	s.span.AddEvent(name, trace.WithAttributes(attrs...))
}

func (s *otelSpan) TraceID() string {
	if sc := s.span.SpanContext(); sc.IsValid() {
		return sc.TraceID().String()
	}
	return ""
}

// ─── Context helpers ─────────────────────────────────────────────────────────

// TraceIDFromContext extracts the trace ID string from context.
func TraceIDFromContext(ctx context.Context) string {
	if sc := trace.SpanFromContext(ctx).SpanContext(); sc.IsValid() {
		return sc.TraceID().String()
	}
	return ""
}

// SpanIDFromContext extracts the span ID string from context.
func SpanIDFromContext(ctx context.Context) string {
	if sc := trace.SpanFromContext(ctx).SpanContext(); sc.IsValid() {
		return sc.SpanID().String()
	}
	return ""
}

// ─── Noop provider ───────────────────────────────────────────────────────────

func newNoopProvider(cfg *Config) *Provider {
	return &Provider{
		tracer: otel.Tracer(cfg.ServiceName),
		meter:  otel.Meter(cfg.ServiceName),
		cfg:    cfg,
	}
}

// ─── Internal helpers ────────────────────────────────────────────────────────

func anyAttr(key string, value interface{}) attribute.KeyValue {
	switch v := value.(type) {
	case string:
		return attribute.String(key, v)
	case int:
		return attribute.Int(key, v)
	case int64:
		return attribute.Int64(key, v)
	case float64:
		return attribute.Float64(key, v)
	case bool:
		return attribute.Bool(key, v)
	default:
		return attribute.String(key, fmt.Sprintf("%v", v))
	}
}

func sanitizeName(s string) string {
	out := make([]byte, len(s))
	for i := range s {
		c := s[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') {
			out[i] = c
		} else {
			out[i] = '_'
		}
	}
	return string(out)
}

// ─── Legacy shim (backward compatibility) ────────────────────────────────────

// Tracer is the legacy tracer type kept for code that uses NewTracer.
// Deprecated: use Init() and Provider.StartSpan().
type Tracer struct{ p *Provider }

// NewTracer creates a tracer using the legacy API.
// Deprecated: use Init() instead.
func NewTracer(cfg *Config) (*Tracer, error) {
	p := newNoopProvider(cfg)
	return &Tracer{p: p}, nil
}

// StartSpan starts a new span (legacy API).
func (t *Tracer) StartSpan(ctx context.Context, name string) (context.Context, Span) {
	return t.p.StartSpan(ctx, name)
}

// Close is a no-op for the legacy tracer.
func (t *Tracer) Close() error { return nil }

// Metrics provides metrics collection (legacy API).
// Deprecated: use Init() and Provider metrics helpers.
type Metrics struct{ enabled bool }

// NewMetrics creates a metrics collector.
// Deprecated: use Init() instead.
func NewMetrics(cfg *Config) (*Metrics, error) {
	return &Metrics{enabled: cfg.Enabled}, nil
}

func (m *Metrics) RecordCounter(name string, value int64, labels map[string]string)   {}
func (m *Metrics) RecordHistogram(name string, value float64, labels map[string]string) {}
func (m *Metrics) RecordGauge(name string, value float64, labels map[string]string)   {}
func (m *Metrics) Close() error                                                        { return nil }