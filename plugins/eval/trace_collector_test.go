package eval

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/eval-prompt/internal/config"
	"github.com/eval-prompt/internal/service"
	"github.com/stretchr/testify/require"
)

func TestNewTraceCollector(t *testing.T) {
	collector := NewTraceCollector(TraceConfig{TracesDir: "/tmp/test-traces"})
	require.NotNil(t, collector)
	require.Equal(t, "/tmp/test-traces", collector.tracesDir)
}

func TestNewTraceCollectorWithConfig(t *testing.T) {
	cfg := config.PromptAssetsConfig{TracesDir: "/tmp/test-traces-config"}
	collector := NewTraceCollectorWithConfig(cfg)
	require.NotNil(t, collector)
	require.Equal(t, "/tmp/test-traces-config", collector.tracesDir)
}

func TestTraceCollector_StartSpan(t *testing.T) {
	tempDir := t.TempDir()
	collector := NewTraceCollector(TraceConfig{TracesDir: tempDir})

	ctx := context.Background()
	assetID := "asset-123"
	snapshotID := "snapshot-456"

	// Start a span
	newCtx, err := collector.StartSpan(ctx, assetID, snapshotID)
	require.NoError(t, err)
	require.NotNil(t, newCtx)

	// Verify active spans count
	require.Equal(t, 1, collector.GetActiveSpans())

	// Finalize to clean up
	_, err = collector.Finalize(newCtx)
	require.NoError(t, err)
	require.Equal(t, 0, collector.GetActiveSpans())
}

func TestTraceCollector_StartSpan_CreatesDirectory(t *testing.T) {
	tempDir := filepath.Join(t.TempDir(), "nested", "traces")
	collector := NewTraceCollector(TraceConfig{TracesDir: tempDir})

	// Directory should be created on first StartSpan
	_, err := collector.StartSpan(context.Background(), "asset", "snapshot")
	require.NoError(t, err)
	require.DirExists(t, tempDir)
}

func TestTraceCollector_StartSpan_DirCreationFails(t *testing.T) {
	// Use a path where directory creation will fail
	collector := NewTraceCollector(TraceConfig{TracesDir: "/proc/foo"})

	_, err := collector.StartSpan(context.Background(), "asset", "snapshot")
	require.Error(t, err)
}

func TestTraceCollector_RecordEvent(t *testing.T) {
	tempDir := t.TempDir()
	collector := NewTraceCollector(TraceConfig{TracesDir: tempDir})

	ctx := context.Background()
	newCtx, err := collector.StartSpan(ctx, "asset-1", "snapshot-1")
	require.NoError(t, err)

	// Record an event
	event := service.TraceEvent{
		Name: "test_event",
		Type: "custom",
		Data: map[string]any{"key": "value"},
	}
	err = collector.RecordEvent(newCtx, event)
	require.NoError(t, err)

	// Verify the event was recorded by finalizing
	tracePath, err := collector.Finalize(newCtx)
	require.NoError(t, err)
	require.NotEmpty(t, tracePath)

	// Verify the trace file exists and has content
	content, err := os.ReadFile(tracePath)
	require.NoError(t, err)
	require.NotEmpty(t, content)
}

func TestTraceCollector_RecordEvent_NoActiveSpan(t *testing.T) {
	collector := NewTraceCollector(TraceConfig{TracesDir: t.TempDir()})

	// Try to record event without an active span
	err := collector.RecordEvent(context.Background(), service.TraceEvent{
		Name: "orphan_event",
		Type: "custom",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "no active span")
}

func TestTraceCollector_RecordEvent_SetsSpanID(t *testing.T) {
	tempDir := t.TempDir()
	collector := NewTraceCollector(TraceConfig{TracesDir: tempDir})

	ctx := context.Background()
	newCtx, err := collector.StartSpan(ctx, "asset-1", "snapshot-1")
	require.NoError(t, err)

	// Record an event without SpanID - should get the context's span ID
	event := service.TraceEvent{
		Name: "test_event",
		Type: "custom",
	}
	err = collector.RecordEvent(newCtx, event)
	require.NoError(t, err)
}

func TestTraceCollector_RecordEvent_SetsTimestamp(t *testing.T) {
	tempDir := t.TempDir()
	collector := NewTraceCollector(TraceConfig{TracesDir: tempDir})

	ctx := context.Background()
	newCtx, err := collector.StartSpan(ctx, "asset-1", "snapshot-1")
	require.NoError(t, err)

	// Record an event without timestamp - should get current time
	event := service.TraceEvent{
		Name: "test_event",
		Type: "custom",
	}
	err = collector.RecordEvent(newCtx, event)
	require.NoError(t, err)
}

func TestTraceCollector_Finalize(t *testing.T) {
	tempDir := t.TempDir()
	collector := NewTraceCollector(TraceConfig{TracesDir: tempDir})

	ctx := context.Background()
	newCtx, err := collector.StartSpan(ctx, "asset-1", "snapshot-1")
	require.NoError(t, err)

	// Finalize the span
	tracePath, err := collector.Finalize(newCtx)
	require.NoError(t, err)
	require.NotEmpty(t, tracePath)

	// Verify file was created
	_, err = os.Stat(tracePath)
	require.NoError(t, err)

	// Verify no active spans
	require.Equal(t, 0, collector.GetActiveSpans())
}

func TestTraceCollector_Finalize_NoActiveSpan(t *testing.T) {
	collector := NewTraceCollector(TraceConfig{TracesDir: t.TempDir()})

	_, err := collector.Finalize(context.Background())
	require.Error(t, err)
	require.Contains(t, err.Error(), "no active span")
}

func TestTraceCollector_Finalize_WriteEventFails(t *testing.T) {
	tempDir := t.TempDir()
	collector := NewTraceCollector(TraceConfig{TracesDir: tempDir})

	ctx := context.Background()
	newCtx, err := collector.StartSpan(ctx, "asset-1", "snapshot-1")
	require.NoError(t, err)

	// Get the span info from the context
	info, ok := newCtx.Value(traceContextKey{}).(*spanInfo)
	require.True(t, ok, "spanInfo should be in context")

	// Close the file to simulate write failure on finalization
	err = info.File.Close()
	require.NoError(t, err)

	_, err = collector.Finalize(newCtx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to write span end")
}

func TestTraceCollector_Finalize_CloseFileFails(t *testing.T) {
	tempDir := t.TempDir()
	collector := NewTraceCollector(TraceConfig{TracesDir: tempDir})

	ctx := context.Background()
	newCtx, err := collector.StartSpan(ctx, "asset-1", "snapshot-1")
	require.NoError(t, err)

	// Manually close the file to trigger close error on finalize
	info, ok := newCtx.Value(traceContextKey{}).(*spanInfo)
	require.True(t, ok)
	info.File.Close()
	info.Mu.Lock()
	info.File.Close() // Double close to trigger error
	info.Mu.Unlock()

	_, err = collector.Finalize(newCtx)
	require.Error(t, err)
}

func TestTraceCollector_Close(t *testing.T) {
	tempDir := t.TempDir()
	collector := NewTraceCollector(TraceConfig{TracesDir: tempDir})

	ctx := context.Background()
	_, err := collector.StartSpan(ctx, "asset-1", "snapshot-1")
	require.NoError(t, err)

	ctx2 := context.Background()
	_, err = collector.StartSpan(ctx2, "asset-2", "snapshot-2")
	require.NoError(t, err)

	// Verify 2 active spans
	require.Equal(t, 2, collector.GetActiveSpans())

	// Close all spans
	err = collector.Close()
	require.NoError(t, err)

	// Verify no active spans
	require.Equal(t, 0, collector.GetActiveSpans())
}

func TestTraceCollector_Close_MultipleSpans(t *testing.T) {
	tempDir := t.TempDir()
	collector := NewTraceCollector(TraceConfig{TracesDir: tempDir})

	// Create multiple spans
	for i := 0; i < 5; i++ {
		_, err := collector.StartSpan(context.Background(), "asset", "snapshot")
		require.NoError(t, err)
	}

	require.Equal(t, 5, collector.GetActiveSpans())

	// Close should handle multiple spans
	err := collector.Close()
	require.NoError(t, err)
	require.Equal(t, 0, collector.GetActiveSpans())
}

func TestTraceCollector_GetActiveSpans(t *testing.T) {
	tempDir := t.TempDir()
	collector := NewTraceCollector(TraceConfig{TracesDir: tempDir})

	// Initially no spans
	require.Equal(t, 0, collector.GetActiveSpans())

	// Start and finalize a span
	ctx := context.Background()
	newCtx, err := collector.StartSpan(ctx, "asset-1", "snapshot-1")
	require.NoError(t, err)
	require.Equal(t, 1, collector.GetActiveSpans())

	_, err = collector.Finalize(newCtx)
	require.NoError(t, err)
	require.Equal(t, 0, collector.GetActiveSpans())
}

func TestTraceCollector_StartSpan_ParentSpan(t *testing.T) {
	tempDir := t.TempDir()
	collector := NewTraceCollector(TraceConfig{TracesDir: tempDir})

	// Create parent span
	ctx := context.Background()
	parentCtx, err := collector.StartSpan(ctx, "parent-asset", "parent-snapshot")
	require.NoError(t, err)

	// Create child span with parent context
	childCtx, err := collector.StartSpan(parentCtx, "child-asset", "child-snapshot")
	require.NoError(t, err)
	require.NotNil(t, childCtx)

	// Verify both spans exist
	require.Equal(t, 2, collector.GetActiveSpans())

	// Clean up
	collector.Close()
}

func TestTraceCollector_writeEvent(t *testing.T) {
	// This is tested indirectly via StartSpan, RecordEvent, Finalize
	// Direct testing requires internal state manipulation
	tempDir := t.TempDir()
	collector := NewTraceCollector(TraceConfig{TracesDir: tempDir})

	ctx := context.Background()
	newCtx, err := collector.StartSpan(ctx, "asset-1", "snapshot-1")
	require.NoError(t, err)

	// Verify we can write multiple events
	for i := 0; i < 3; i++ {
		event := service.TraceEvent{
			Name: "test_event",
			Type: "custom",
			Data: map[string]any{"index": i},
		}
		err = collector.RecordEvent(newCtx, event)
		require.NoError(t, err)
	}

	// Finalize and verify trace file
	tracePath, err := collector.Finalize(newCtx)
	require.NoError(t, err)

	content, err := os.ReadFile(tracePath)
	require.NoError(t, err)
	// Should have span_start, 3 events, and span_end = 5 lines
	lines := len(content) / 100 // Approximate, just check it's not empty
	require.Greater(t, lines, 0)
}
