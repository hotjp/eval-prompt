// Package telemetry provides observability: logging, tracing, and metrics.
// This is a stub implementation that provides no-op operations when OTel is unavailable.
package telemetry

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// OTelConfig holds OpenTelemetry configuration.
type OTelConfig struct {
	Enabled      bool
	ServiceName  string
	Exporter     string // "stdout", "jaeger"
	MetricsPort  int
	TracingPort  int
	SamplingRate float64
}

// DefaultOTELConfig returns default OpenTelemetry configuration.
func DefaultOTELConfig() OTelConfig {
	return OTelConfig{
		Enabled:      true,
		ServiceName:  "eval-prompt",
		Exporter:     "stdout",
		MetricsPort:  9090,
		TracingPort:  4318,
		SamplingRate: 0.1,
	}
}

// OTelProvider provides tracing and metrics (stub).
type OTelProvider struct {
	config OTelConfig
}

// NewOTelProvider creates a new OpenTelemetry provider (stub).
func NewOTelProvider(cfg OTelConfig) (*OTelProvider, error) {
	return &OTelProvider{config: cfg}, nil
}

// Tracer returns a no-op tracer.
func (p *OTelProvider) Tracer() interface{} {
	return nil
}

// Meter returns a no-op meter.
func (p *OTelProvider) Meter() interface{} {
	return nil
}

// Shutdown does nothing.
func (p *OTelProvider) Shutdown(ctx context.Context) error {
	return nil
}

// StartSpan starts a no-op span.
func (p *OTelProvider) StartSpan(ctx context.Context, name string) (context.Context, interface{}) {
	return ctx, nil
}

// MetricsCollector provides metrics collection (stub).
type MetricsCollector struct{}

// NewMetricsCollector creates a new metrics collector (stub).
func NewMetricsCollector() (*MetricsCollector, error) {
	return &MetricsCollector{}, nil
}

// RecordHTTPRequest does nothing.
func (m *MetricsCollector) RecordHTTPRequest(method, path string, status int, duration time.Duration) {
}

// RecordEvalRun does nothing.
func (m *MetricsCollector) RecordEvalRun(status string) {
}

// RecordAssetOperation does nothing.
func (m *MetricsCollector) RecordAssetOperation(op string) {
}

// PrometheusHandler returns a metrics handler (stub).
func PrometheusHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(fmt.Sprintf(`# HELP eval_prompt_up Up indicator
# TYPE eval_prompt_up gauge
eval_prompt_up 1
`)))
	})
}
