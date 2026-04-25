// Package authz implements L3-Authz layer: permission checks (RBAC/OpenFGA),
// rate limiting, and identity verification.
package authz

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"
)

// AuditEntry represents a single audit log entry.
type AuditEntry struct {
	Timestamp  time.Time              `json:"timestamp"`
	Actor      string                 `json:"actor"`       // Agent identity
	Operation  string                 `json:"operation"`   // AssetCreated, LabelPromoted, etc.
	ResourceID string                 `json:"resource_id"` // Asset ID or Label name
	AssetID    string                 `json:"asset_id,omitempty"`
	SnapshotID string                 `json:"snapshot_id,omitempty"`
	Details    map[string]interface{} `json:"details,omitempty"`
}

// AuditLogger records operations for audit compliance.
type AuditLogger struct {
	logger        *slog.Logger
	auditRecorder AuditRecorder
}

// AuditRecorder is the interface for recording audit entries.
type AuditRecorder interface {
	Record(ctx context.Context, entry *AuditEntry) error
}

// NewAuditLogger creates a new AuditLogger.
func NewAuditLogger(logger *slog.Logger) *AuditLogger {
	return &AuditLogger{
		logger: logger,
	}
}

// NewAuditLoggerWithRecorder creates a new AuditLogger with a custom recorder.
func NewAuditLoggerWithRecorder(logger *slog.Logger, recorder AuditRecorder) *AuditLogger {
	return &AuditLogger{
		logger:        logger,
		auditRecorder: recorder,
	}
}

// LogAssetCreated logs an asset creation event.
func (a *AuditLogger) LogAssetCreated(ctx context.Context, actor, assetID, name, bizLine string, tags []string) {
	entry := &AuditEntry{
		Timestamp:  time.Now(),
		Actor:      actor,
		Operation:  "AssetCreated",
		ResourceID: assetID,
		AssetID:    assetID,
		Details: map[string]interface{}{
			"name":     name,
			"asset_type": bizLine,
			"tags":     tags,
		},
	}
	a.record(ctx, entry)
}

// LogAssetUpdated logs an asset update event.
func (a *AuditLogger) LogAssetUpdated(ctx context.Context, actor, assetID, reason string) {
	entry := &AuditEntry{
		Timestamp:  time.Now(),
		Actor:      actor,
		Operation:  "AssetUpdated",
		ResourceID: assetID,
		AssetID:    assetID,
		Details: map[string]interface{}{
			"reason": reason,
		},
	}
	a.record(ctx, entry)
}

// LogAssetDeleted logs an asset deletion event.
func (a *AuditLogger) LogAssetDeleted(ctx context.Context, actor, assetID string) {
	entry := &AuditEntry{
		Timestamp:  time.Now(),
		Actor:      actor,
		Operation:  "AssetDeleted",
		ResourceID: assetID,
		AssetID:    assetID,
	}
	a.record(ctx, entry)
}

// LogLabelPromoted logs a label promotion event.
func (a *AuditLogger) LogLabelPromoted(ctx context.Context, actor, assetID, labelName, fromVersion, toVersion string, evalScore int) {
	entry := &AuditEntry{
		Timestamp:  time.Now(),
		Actor:      actor,
		Operation:  "LabelPromoted",
		ResourceID: labelName,
		AssetID:    assetID,
		Details: map[string]interface{}{
			"label_name":   labelName,
			"from_version": fromVersion,
			"to_version":   toVersion,
			"eval_score":   evalScore,
		},
	}
	a.record(ctx, entry)
}

// LogLabelMoved logs a label move event (internal move, not promotion).
func (a *AuditLogger) LogLabelMoved(ctx context.Context, actor, assetID, labelName, toVersion string) {
	entry := &AuditEntry{
		Timestamp:  time.Now(),
		Actor:      actor,
		Operation:  "LabelMoved",
		ResourceID: labelName,
		AssetID:    assetID,
		Details: map[string]interface{}{
			"label_name": labelName,
			"to_version": toVersion,
		},
	}
	a.record(ctx, entry)
}

// LogEvalTriggered logs an evaluation trigger event.
func (a *AuditLogger) LogEvalTriggered(ctx context.Context, actor, assetID, snapshotID, caseID string) {
	entry := &AuditEntry{
		Timestamp:  time.Now(),
		Actor:      actor,
		Operation:  "EvalTriggered",
		ResourceID: snapshotID,
		AssetID:    assetID,
		SnapshotID: snapshotID,
		Details: map[string]interface{}{
			"case_id": caseID,
		},
	}
	a.record(ctx, entry)
}

// LogEvalCompleted logs an evaluation completion event.
func (a *AuditLogger) LogEvalCompleted(ctx context.Context, actor, assetID, snapshotID, runID string, passed bool, score int) {
	entry := &AuditEntry{
		Timestamp:  time.Now(),
		Actor:      actor,
		Operation:  "EvalCompleted",
		ResourceID: runID,
		AssetID:    assetID,
		SnapshotID: snapshotID,
		Details: map[string]interface{}{
			"run_id": runID,
			"passed": passed,
			"score":  score,
		},
	}
	a.record(ctx, entry)
}

// record logs and optionally persists the audit entry.
func (a *AuditLogger) record(ctx context.Context, entry *AuditEntry) {
	// Log to structured logger
	if a.logger != nil {
		a.logger.Info("audit",
			slog.String("actor", entry.Actor),
			slog.String("operation", entry.Operation),
			slog.String("resource_id", entry.ResourceID),
			slog.Any("details", entry.Details),
		)
	}

	// Persist if recorder is configured
	if a.auditRecorder != nil {
		if err := a.auditRecorder.Record(ctx, entry); err != nil {
			if a.logger != nil {
				a.logger.Error("failed to persist audit entry", slog.Any("error", err))
			}
		}
	}
}

// AuditEntryJSON returns the audit entry as JSON for testing.
func (a *AuditEntry) AuditEntryJSON() ([]byte, error) {
	return json.Marshal(a)
}
