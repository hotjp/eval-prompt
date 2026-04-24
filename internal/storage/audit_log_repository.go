package storage

import (
	"context"

	"github.com/eval-prompt/internal/domain"
	"github.com/eval-prompt/internal/storage/ent"
	"github.com/eval-prompt/internal/storage/ent/auditlog"
)

// AuditLogRepository provides repository operations for AuditLog entities.
type AuditLogRepository struct {
	client *Client
}

// NewAuditLogRepository creates a new AuditLogRepository.
func NewAuditLogRepository(client *Client) *AuditLogRepository {
	return &AuditLogRepository{client: client}
}

// Create creates a new audit log entry in the database.
func (r *AuditLogRepository) Create(ctx context.Context, a *domain.AuditLog) error {
	detailsMap := make(map[string]interface{})
	for k, v := range a.Details {
		detailsMap[k] = v
	}

	_, err := r.client.ent.AuditLog.Create().
		SetID(a.ID.String()).
		SetOperation(a.Operation).
		SetAssetID(a.AssetID.String()).
		SetUserID(a.UserID.String()).
		SetDetails(detailsMap).
		SetCreatedAt(a.CreatedAt).
		Save(ctx)
	return err
}

// GetByID retrieves an audit log by its ID.
func (r *AuditLogRepository) GetByID(ctx context.Context, id string) (*domain.AuditLog, error) {
	entLog, err := r.client.ent.AuditLog.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	return r.toDomainAuditLog(entLog), nil
}

// GetByAssetID retrieves all audit logs for an asset.
func (r *AuditLogRepository) GetByAssetID(ctx context.Context, assetID string) ([]*domain.AuditLog, error) {
	entLogs, err := r.client.ent.AuditLog.Query().
		Where(auditlog.AssetIDEQ(assetID)).
		All(ctx)
	if err != nil {
		return nil, err
	}

	logs := make([]*domain.AuditLog, len(entLogs))
	for i, entLog := range entLogs {
		logs[i] = r.toDomainAuditLog(entLog)
	}
	return logs, nil
}

// List retrieves audit logs with pagination.
func (r *AuditLogRepository) List(ctx context.Context, offset, limit int) ([]*domain.AuditLog, int, error) {
	entLogs, err := r.client.ent.AuditLog.Query().
		Offset(offset).
		Limit(limit).
		All(ctx)
	if err != nil {
		return nil, 0, err
	}

	total, err := r.client.ent.AuditLog.Query().Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	logs := make([]*domain.AuditLog, len(entLogs))
	for i, entLog := range entLogs {
		logs[i] = r.toDomainAuditLog(entLog)
	}
	return logs, total, nil
}

// Delete deletes an audit log by its ID.
func (r *AuditLogRepository) Delete(ctx context.Context, id string) error {
	return r.client.ent.AuditLog.DeleteOneID(id).Exec(ctx)
}

// toDomainAuditLog converts an ent AuditLog to a domain AuditLog.
func (r *AuditLogRepository) toDomainAuditLog(e *ent.AuditLog) *domain.AuditLog {
	assetID := domain.ID{}
	if e.AssetID != "" {
		assetID = domain.MustNewID(e.AssetID)
	}

	userID := domain.ID{}
	if e.UserID != "" {
		userID = domain.MustNewID(e.UserID)
	}

	details := make(map[string]any)
	if e.Details != nil {
		details = e.Details
	}

	return &domain.AuditLog{
		ID:        domain.MustNewID(e.ID),
		Operation: e.Operation,
		AssetID:   assetID,
		UserID:    userID,
		Details:   details,
		CreatedAt: e.CreatedAt,
	}
}
