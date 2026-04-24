// Package telemetry provides observability: logging, tracing, and metrics.
package telemetry

import (
	"context"
	"io"
	"log/slog"
	"os"
	"time"
)

// Config holds logger configuration.
type Config struct {
	Level  string // debug, info, warn, error
	Format string // json, text
}

// DefaultConfig returns default logger configuration.
func DefaultConfig() Config {
	return Config{
		Level:  "info",
		Format: "json",
	}
}

// Logger provides structured logging with trace context.
type Logger struct {
	*slog.Logger
	traceIDKey string
	spanIDKey  string
}

// NewLogger creates a new Logger with JSON handler.
func NewLogger(cfg Config) *Logger {
	var level slog.Level
	switch cfg.Level {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	var handler slog.Handler
	if cfg.Format == "text" {
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level:     level,
			AddSource: false,
		})
	} else {
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level:     level,
			AddSource: false,
		})
	}

	return &Logger{
		Logger:    slog.New(handler),
		traceIDKey: "trace_id",
		spanIDKey:  "span_id",
	}
}

// WithTrace adds trace context to the logger.
func (l *Logger) WithTrace(ctx context.Context) *slog.Logger {
	logger := l.Logger

	// Extract trace_id and span_id from context if available
	if traceID := ctx.Value(l.traceIDKey); traceID != nil {
		logger = logger.With(l.traceIDKey, traceID)
	}
	if spanID := ctx.Value(l.spanIDKey); spanID != nil {
		logger = logger.With(l.spanIDKey, spanID)
	}

	return logger
}

// WithLayer adds the layer field to the logger.
func (l *Logger) WithLayer(layer string) *slog.Logger {
	return l.Logger.With("layer", layer)
}

// WithAssetID adds the asset_id field to the logger.
func (l *Logger) WithAssetID(assetID string) *slog.Logger {
	return l.Logger.With("asset_id", assetID)
}

// WithRequestID adds the request_id field to the logger.
func (l *Logger) WithRequestID(requestID string) *slog.Logger {
	return l.Logger.With("request_id", requestID)
}

// Info logs at info level with layer field.
func (l *Logger) Info(msg string, layer string, args ...any) {
	l.Logger.Info(msg, append([]any{"layer", layer}, args...)...)
}

// Error logs at error level with layer field.
func (l *Logger) Error(msg string, layer string, args ...any) {
	l.Logger.Error(msg, append([]any{"layer", layer}, args...)...)
}

// Debug logs at debug level with layer field.
func (l *Logger) Debug(msg string, layer string, args ...any) {
	l.Logger.Debug(msg, append([]any{"layer", layer}, args...)...)
}

// Warn logs at warn level with layer field.
func (l *Logger) Warn(msg string, layer string, args ...any) {
	l.Logger.Warn(msg, append([]any{"layer", layer}, args...)...)
}

// LogRequest logs an HTTP request with relevant fields.
func (l *Logger) LogRequest(layer, method, path string, status int, duration time.Duration) {
	l.Logger.Info("http_request",
		"layer", layer,
		"method", method,
		"path", path,
		"status", status,
		"duration_ms", duration.Milliseconds(),
	)
}

// SetGlobal sets the global logger.
func SetGlobal(logger *Logger) {
	slog.SetDefault(logger.Logger)
}

// GetGlobal returns the global logger.
func GetGlobal() *Logger {
	return &Logger{
		Logger: slog.Default(),
	}
}
