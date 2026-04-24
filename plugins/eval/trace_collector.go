// Package eval provides evaluation-related plugin implementations.
package eval

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/eval-prompt/internal/config"
	"github.com/eval-prompt/internal/service"
)

// traceContextKey is the key for storing trace info in context.
type traceContextKey struct{}

// spanInfo holds trace span information stored in context.
type spanInfo struct {
	SpanID     string
	ParentID   string
	AssetID    string
	SnapshotID string
	TracePath  string
	File       *os.File
	Mu         sync.Mutex
}

// TraceConfig holds trace collector configuration.
type TraceConfig struct {
	TracesDir string
}

// TraceCollector implements service.TraceCollector for collecting eval traces.
type TraceCollector struct {
	tracesDir string
	spans     map[string]*spanInfo
	mu        sync.RWMutex
}

// NewTraceCollector creates a new TraceCollector.
func NewTraceCollector(cfg TraceConfig) *TraceCollector {
	return &TraceCollector{
		tracesDir: cfg.TracesDir,
		spans:     make(map[string]*spanInfo),
	}
}

// NewTraceCollectorWithConfig creates a TraceCollector from PromptAssetsConfig.
func NewTraceCollectorWithConfig(cfg config.PromptAssetsConfig) *TraceCollector {
	return NewTraceCollector(TraceConfig{TracesDir: cfg.TracesDir})
}

// Ensure TraceCollector implements service.TraceCollector.
var _ service.TraceCollector = (*TraceCollector)(nil)

// StartSpan implements service.TraceCollector.
func (tc *TraceCollector) StartSpan(ctx context.Context, assetID, snapshotID string) (context.Context, error) {
	// Ensure traces directory exists
	if err := os.MkdirAll(tc.tracesDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create traces dir: %w", err)
	}

	// Generate trace file path
	traceFileName := fmt.Sprintf("%s_%s_%s.jsonl", assetID, snapshotID, uuid.New().String()[:8])
	tracePath := filepath.Join(tc.tracesDir, traceFileName)

	// Open trace file
	f, err := os.Create(tracePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create trace file: %w", err)
	}

	// Generate span ID
	spanID := uuid.New().String()

	// Get parent span ID from context if present
	parentID := ""
	if p, ok := ctx.Value(traceContextKey{}).(*spanInfo); ok {
		parentID = p.SpanID
	}

	// Create span info
	info := &spanInfo{
		SpanID:     spanID,
		ParentID:   parentID,
		AssetID:    assetID,
		SnapshotID: snapshotID,
		TracePath:  tracePath,
		File:       f,
	}

	// Record span start event
	startEvent := service.TraceEvent{
		SpanID:    spanID,
		ParentID:  parentID,
		Name:      "span_start",
		Timestamp: time.Now(),
		Type:      "span_start",
		Data: map[string]any{
			"asset_id":    assetID,
			"snapshot_id": snapshotID,
		},
	}
	if err := tc.writeEvent(info, startEvent); err != nil {
		f.Close()
		return nil, fmt.Errorf("failed to write span start: %w", err)
	}

	// Store span info
	tc.mu.Lock()
	tc.spans[spanID] = info
	tc.mu.Unlock()

	// Return context with span info
	return context.WithValue(ctx, traceContextKey{}, info), nil
}

// RecordEvent implements service.TraceCollector.
func (tc *TraceCollector) RecordEvent(ctx context.Context, event service.TraceEvent) error {
	info, ok := ctx.Value(traceContextKey{}).(*spanInfo)
	if !ok {
		return fmt.Errorf("no active span in context")
	}

	// Set span ID from context if not provided
	if event.SpanID == "" {
		event.SpanID = info.SpanID
	}

	// Set timestamp if not provided
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	return tc.writeEvent(info, event)
}

// Finalize implements service.TraceCollector.
func (tc *TraceCollector) Finalize(ctx context.Context) (string, error) {
	info, ok := ctx.Value(traceContextKey{}).(*spanInfo)
	if !ok {
		return "", fmt.Errorf("no active span in context")
	}

	// Write span end event
	endEvent := service.TraceEvent{
		SpanID:    info.SpanID,
		ParentID:  info.ParentID,
		Name:      "span_end",
		Timestamp: time.Now(),
		Type:      "span_end",
	}
	if err := tc.writeEvent(info, endEvent); err != nil {
		return "", fmt.Errorf("failed to write span end: %w", err)
	}

	// Close trace file
	info.Mu.Lock()
	defer info.Mu.Unlock()
	if err := info.File.Close(); err != nil {
		return "", fmt.Errorf("failed to close trace file: %w", err)
	}

	// Remove span from active spans
	tc.mu.Lock()
	delete(tc.spans, info.SpanID)
	tc.mu.Unlock()

	return info.TracePath, nil
}

// writeEvent writes a trace event to the trace file in JSONL format.
func (tc *TraceCollector) writeEvent(info *spanInfo, event service.TraceEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	info.Mu.Lock()
	defer info.Mu.Unlock()

	if _, err := info.File.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("failed to write event: %w", err)
	}

	return nil
}

// GetActiveSpans returns the number of active spans (for testing/monitoring).
func (tc *TraceCollector) GetActiveSpans() int {
	tc.mu.RLock()
	defer tc.mu.RUnlock()
	return len(tc.spans)
}

// Close closes all open trace files (for graceful shutdown).
func (tc *TraceCollector) Close() error {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	for _, info := range tc.spans {
		info.Mu.Lock()
		info.File.Close()
		info.Mu.Unlock()
	}
	tc.spans = make(map[string]*spanInfo)
	return nil
}
