// Package telemetry provides observability: logging, tracing, and metrics.
package telemetry

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

// Config holds OpenTelemetry configuration.
type Config struct {
	Enabled      bool
	ServiceName  string
	Exporter     string // "stdout", "jaeger"
	MetricsPort  int
	TracingPort  int
	SamplingRate float64
}

// DefaultOTELConfig returns default OpenTelemetry configuration.
func DefaultOTELConfig() Config {
	return Config{
		Enabled:     true,
		ServiceName: "eval-prompt",
		Exporter:    "stdout",
		MetricsPort: 9090,
		TracingPort: 4318,
		SamplingRate: 0.1, // 10% sampling
	}
}

// OTelProvider provides tracing and metrics.
type OTelProvider struct {
	tracer trace.Tracer
	meter  metric.Meter
	config Config
}

// NewOTelProvider creates a new OpenTelemetry provider.
func NewOTelProvider(cfg Config) (*OTelProvider, error) {
	p := &OTelProvider{config: cfg}

	// Create tracer and meter using global providers
	p.tracer = otel.Tracer(cfg.ServiceName)
	p.meter = otel.Meter(cfg.ServiceName)

	// Set text map propagator
	otel.SetTextMapPropagator(propagation.NewCompositePropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return p, nil
}

// Tracer returns the tracer.
func (p *OTelProvider) Tracer() trace.Tracer {
	return p.tracer
}

// Meter returns the meter.
func (p *OTelProvider) Meter() metric.Meter {
	return p.meter
}

// Shutdown gracefully shuts down the provider.
func (p *OTelProvider) Shutdown(ctx context.Context) error {
	return nil // Global providers are shutdown elsewhere
}

// StartSpan starts a new span with the given name.
func (p *OTelProvider) StartSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return p.tracer.Start(ctx, name, opts...)
}

// AddSpanAttributes adds attributes to the current span.
func AddSpanAttributes(ctx context.Context, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(attrs...)
}

// RecordError records an error on the current span.
func RecordError(ctx context.Context, err error) {
	span := trace.SpanFromContext(ctx)
	span.RecordError(err)
}

// MetricsCollector provides metrics collection.
type MetricsCollector struct {
	httpRequestsTotal   metric.Int64Counter
	httpRequestDuration metric.Float64Histogram
	evalRunsTotal       metric.Int64Counter
	evalRunsPassed      metric.Int64Counter
	promptAssetsTotal   metric.Int64UpDownCounter
}

// NewMetricsCollector creates a new metrics collector.
func NewMetricsCollector() (*MetricsCollector, error) {
	m := &MetricsCollector{}

	meter := otel.Meter("eval-prompt")

	var err error

	m.httpRequestsTotal, err = meter.Int64Counter(
		"http_requests_total",
		metric.WithDescription("Total HTTP requests"),
		metric.WithUnit("{request}"),
	)
	if err != nil {
		return nil, err
	}

	m.httpRequestDuration, err = meter.Float64Histogram(
		"http_request_duration_seconds",
		metric.WithDescription("HTTP request duration"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, err
	}

	m.evalRunsTotal, err = meter.Int64Counter(
		"eval_runs_total",
		metric.WithDescription("Total eval runs"),
		metric.WithUnit("{run}"),
	)
	if err != nil {
		return nil, err
	}

	m.evalRunsPassed, err = meter.Int64Counter(
		"eval_runs_passed",
		metric.WithDescription("Total passed eval runs"),
		metric.WithUnit("{run}"),
	)
	if err != nil {
		return nil, err
	}

	m.promptAssetsTotal, err = meter.Int64UpDownCounter(
		"prompt_assets_total",
		metric.WithDescription("Total prompt assets"),
		metric.WithUnit("{asset}"),
	)
	if err != nil {
		return nil, err
	}

	return m, nil
}

// RecordHTTPRequest records an HTTP request.
func (m *MetricsCollector) RecordHTTPRequest(method, path string, status int, duration time.Duration) {
	ctx := context.Background()

	m.httpRequestsTotal.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("method", method),
			attribute.String("path", path),
			attribute.Int("status", status),
		),
	)

	m.httpRequestDuration.Record(ctx, duration.Seconds(),
		metric.WithAttributes(
			attribute.String("method", method),
			attribute.String("path", path),
		),
	)
}

// RecordEvalRun records an eval run.
func (m *MetricsCollector) RecordEvalRun(status string) {
	ctx := context.Background()

	m.evalRunsTotal.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("status", status),
		),
	)

	if status == "passed" {
		m.evalRunsPassed.Add(ctx, 1)
	}
}

// RecordAssetOperation records a prompt asset operation.
func (m *MetricsCollector) RecordAssetOperation(op string) {
	ctx := context.Background()

	m.promptAssetsTotal.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("operation", op),
		),
	)
}

// PrometheusHandler returns an HTTP handler for Prometheus metrics.
func PrometheusHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(fmt.Sprintf(`# HELP eval_prompt_up Up indicator
# TYPE eval_prompt_up gauge
eval_prompt_up 1
# HELP eval_prompt_http_requests_total Total HTTP requests
# TYPE eval_prompt_http_requests_total counter
eval_prompt_http_requests_total{method="GET",path="/healthz",status="200"} 1
`)))
	})
}
