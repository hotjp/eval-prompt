package domain

import (
	"time"
)

// AuditLog represents an audit log entry for tracking operations.
type AuditLog struct {
	ID        ID
	Operation string
	AssetID   ID
	UserID    ID
	Details   map[string]any
	CreatedAt time.Time
}

// NewAuditLog creates a new AuditLog entry.
func NewAuditLog(operation string, assetID, userID ID, details map[string]any) *AuditLog {
	return &AuditLog{
		ID:        NewAutoID(),
		Operation: operation,
		AssetID:   assetID,
		UserID:    userID,
		Details:   details,
		CreatedAt: time.Now(),
	}
}

// Validate validates the audit log entry.
func (a *AuditLog) Validate() error {
	if a.Operation == "" {
		return NewDomainError(ErrDomainRuleViolation, "operation is required")
	}
	return nil
}

// AuditLogSummary is a lightweight representation of an audit log entry.
type AuditLogSummary struct {
	ID        ID
	Operation string
	AssetID   ID
	UserID    ID
	CreatedAt time.Time
}
