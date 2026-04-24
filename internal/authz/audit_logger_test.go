package authz

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// mockAuditRecorder is a mock implementation of AuditRecorder for testing.
type mockAuditRecorder struct {
	entries []*AuditEntry
}

func (m *mockAuditRecorder) Record(ctx context.Context, entry *AuditEntry) error {
	m.entries = append(m.entries, entry)
	return nil
}

// mockAuditRecorderWithError simulates a recorder that fails
type mockAuditRecorderWithError struct {
	err    error
	called int
}

func (m *mockAuditRecorderWithError) Record(ctx context.Context, entry *AuditEntry) error {
	m.called++
	return m.err
}

func TestAuditLogger_LogAssetCreated(t *testing.T) {
	recorder := &mockAuditRecorder{}
	logger := NewAuditLoggerWithRecorder(nil, recorder)

	logger.LogAssetCreated(context.Background(), "agent-1", "asset-123", "TestAsset", "ml", []string{"tag1", "tag2"})

	if len(recorder.entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(recorder.entries))
	}

	entry := recorder.entries[0]
	if entry.Actor != "agent-1" {
		t.Errorf("expected actor 'agent-1', got %q", entry.Actor)
	}
	if entry.Operation != "AssetCreated" {
		t.Errorf("expected operation 'AssetCreated', got %q", entry.Operation)
	}
	if entry.AssetID != "asset-123" {
		t.Errorf("expected asset_id 'asset-123', got %q", entry.AssetID)
	}
	if entry.Details["name"] != "TestAsset" {
		t.Errorf("expected name 'TestAsset', got %v", entry.Details["name"])
	}
}

func TestAuditLogger_LogAssetUpdated(t *testing.T) {
	recorder := &mockAuditRecorder{}
	logger := NewAuditLoggerWithRecorder(nil, recorder)

	logger.LogAssetUpdated(context.Background(), "agent-1", "asset-123", "bug fix")

	if len(recorder.entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(recorder.entries))
	}

	entry := recorder.entries[0]
	if entry.Operation != "AssetUpdated" {
		t.Errorf("expected operation 'AssetUpdated', got %q", entry.Operation)
	}
	if entry.Details["reason"] != "bug fix" {
		t.Errorf("expected reason 'bug fix', got %v", entry.Details["reason"])
	}
}

func TestAuditLogger_LogAssetDeleted(t *testing.T) {
	recorder := &mockAuditRecorder{}
	logger := NewAuditLoggerWithRecorder(nil, recorder)

	logger.LogAssetDeleted(context.Background(), "agent-1", "asset-123")

	if len(recorder.entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(recorder.entries))
	}

	entry := recorder.entries[0]
	if entry.Operation != "AssetDeleted" {
		t.Errorf("expected operation 'AssetDeleted', got %q", entry.Operation)
	}
}

func TestAuditLogger_LogLabelPromoted(t *testing.T) {
	recorder := &mockAuditRecorder{}
	logger := NewAuditLoggerWithRecorder(nil, recorder)

	logger.LogLabelPromoted(context.Background(), "agent-1", "asset-123", "prod", "v1.0.0", "v2.0.0", 85)

	if len(recorder.entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(recorder.entries))
	}

	entry := recorder.entries[0]
	if entry.Operation != "LabelPromoted" {
		t.Errorf("expected operation 'LabelPromoted', got %q", entry.Operation)
	}
	if entry.ResourceID != "prod" {
		t.Errorf("expected resource_id 'prod', got %q", entry.ResourceID)
	}
	if entry.Details["from_version"] != "v1.0.0" {
		t.Errorf("expected from_version 'v1.0.0', got %v", entry.Details["from_version"])
	}
	if entry.Details["to_version"] != "v2.0.0" {
		t.Errorf("expected to_version 'v2.0.0', got %v", entry.Details["to_version"])
	}
	if entry.Details["eval_score"] != 85 {
		t.Errorf("expected eval_score 85, got %v", entry.Details["eval_score"])
	}
}

func TestAuditLogger_LogEvalTriggered(t *testing.T) {
	recorder := &mockAuditRecorder{}
	logger := NewAuditLoggerWithRecorder(nil, recorder)

	logger.LogEvalTriggered(context.Background(), "agent-1", "asset-123", "snap-456", "case-789")

	if len(recorder.entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(recorder.entries))
	}

	entry := recorder.entries[0]
	if entry.Operation != "EvalTriggered" {
		t.Errorf("expected operation 'EvalTriggered', got %q", entry.Operation)
	}
	if entry.SnapshotID != "snap-456" {
		t.Errorf("expected snapshot_id 'snap-456', got %q", entry.SnapshotID)
	}
	if entry.Details["case_id"] != "case-789" {
		t.Errorf("expected case_id 'case-789', got %v", entry.Details["case_id"])
	}
}

func TestAuditLogger_LogEvalCompleted(t *testing.T) {
	recorder := &mockAuditRecorder{}
	logger := NewAuditLoggerWithRecorder(nil, recorder)

	logger.LogEvalCompleted(context.Background(), "agent-1", "asset-123", "snap-456", "run-001", true, 90)

	if len(recorder.entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(recorder.entries))
	}

	entry := recorder.entries[0]
	if entry.Operation != "EvalCompleted" {
		t.Errorf("expected operation 'EvalCompleted', got %q", entry.Operation)
	}
	if entry.Details["passed"] != true {
		t.Errorf("expected passed true, got %v", entry.Details["passed"])
	}
	if entry.Details["score"] != 90 {
		t.Errorf("expected score 90, got %v", entry.Details["score"])
	}
}

func TestAuditLogger_Record_WritesSuccessfully(t *testing.T) {
	recorder := &mockAuditRecorder{}
	logger := NewAuditLoggerWithRecorder(nil, recorder)

	// Log multiple operations
	logger.LogAssetCreated(context.Background(), "agent-1", "asset-123", "TestAsset", "ml", []string{"tag1"})
	logger.LogAssetUpdated(context.Background(), "agent-1", "asset-123", "reason here")
	logger.LogLabelPromoted(context.Background(), "agent-1", "asset-123", "prod", "v1", "v2", 85)

	require.Len(t, recorder.entries, 3)
	require.Equal(t, "AssetCreated", recorder.entries[0].Operation)
	require.Equal(t, "AssetUpdated", recorder.entries[1].Operation)
	require.Equal(t, "LabelPromoted", recorder.entries[2].Operation)
}

func TestAuditLogger_LogLabelMoved_NoAssetID(t *testing.T) {
	recorder := &mockAuditRecorder{}
	logger := NewAuditLoggerWithRecorder(nil, recorder)

	logger.LogLabelMoved(context.Background(), "agent-1", "asset-123", "staging", "v3.0.0")

	require.Len(t, recorder.entries, 1)
	entry := recorder.entries[0]
	require.Equal(t, "LabelMoved", entry.Operation)
	require.Equal(t, "staging", entry.ResourceID)
	require.Equal(t, "asset-123", entry.AssetID)
	require.Equal(t, "v3.0.0", entry.Details["to_version"])
}

func TestAuditLogger_Record_ErrorHandling(t *testing.T) {
	// Test that record does not panic when recorder fails
	recorder := &mockAuditRecorderWithError{err: errors.New("db error")}
	logger := NewAuditLoggerWithRecorder(nil, recorder)

	// Should not panic, just log the error
	logger.LogAssetCreated(context.Background(), "agent-1", "asset-123", "TestAsset", "ml", []string{})

	// Verify recorder was called despite error (it just logs and continues)
	require.Equal(t, 1, recorder.called)
}

func TestAuditEntryJSON(t *testing.T) {
	entry := &AuditEntry{
		Timestamp:  time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		Actor:      "test-agent",
		Operation:  "TestOp",
		ResourceID: "res-123",
		AssetID:    "asset-456",
		Details: map[string]interface{}{
			"key": "value",
		},
	}

	data, err := entry.AuditEntryJSON()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var parsed AuditEntry
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if parsed.Actor != entry.Actor {
		t.Errorf("actor mismatch: got %q, want %q", parsed.Actor, entry.Actor)
	}
	if parsed.Operation != entry.Operation {
		t.Errorf("operation mismatch: got %q, want %q", parsed.Operation, entry.Operation)
	}
}
