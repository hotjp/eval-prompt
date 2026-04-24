package storage

import (
	"context"

	"github.com/eval-prompt/internal/domain"
	"github.com/eval-prompt/internal/storage/ent"
	"github.com/eval-prompt/internal/storage/ent/outboxevent"
)

// OutboxRepository provides repository operations for OutboxEvent entities.
type OutboxRepository struct {
	client *Client
}

// NewOutboxRepository creates a new OutboxRepository.
func NewOutboxRepository(client *Client) *OutboxRepository {
	return &OutboxRepository{client: client}
}

// Create creates a new outbox event in the database.
func (r *OutboxRepository) Create(ctx context.Context, e *domain.OutboxEvent) error {
	_, err := r.client.ent.OutboxEvent.Create().
		SetID(e.ID.String()).
		SetAggregateType(e.AggregateType).
		SetAggregateID(e.AggregateID.String()).
		SetEventType(string(e.EventType)).
		SetPayload(string(e.Payload)).
		SetOccurredAt(e.OccurredAt).
		SetIdempotencyKey(e.IdempotencyKey).
		SetStatus(r.statusToEnt(e.Status)).
		SetRetryCount(e.RetryCount).
		Save(ctx)
	return err
}

// GetByID retrieves an outbox event by its ID.
func (r *OutboxRepository) GetByID(ctx context.Context, id string) (*domain.OutboxEvent, error) {
	entEvent, err := r.client.ent.OutboxEvent.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	return r.toDomain(entEvent), nil
}

// GetByAggregateID retrieves all outbox events for an aggregate.
func (r *OutboxRepository) GetByAggregateID(ctx context.Context, aggregateID string) ([]*domain.OutboxEvent, error) {
	entEvents, err := r.client.ent.OutboxEvent.Query().
		Where(outboxevent.AggregateIDEQ(aggregateID)).
		All(ctx)
	if err != nil {
		return nil, err
	}

	events := make([]*domain.OutboxEvent, len(entEvents))
	for i, entEvent := range entEvents {
		events[i] = r.toDomain(entEvent)
	}
	return events, nil
}

// GetPending retrieves pending outbox events for processing.
func (r *OutboxRepository) GetPending(ctx context.Context, limit int) ([]*domain.OutboxEvent, error) {
	entEvents, err := r.client.ent.OutboxEvent.Query().
		Where(outboxevent.StatusEQ(outboxevent.StatusPending)).
		Limit(limit).
		All(ctx)
	if err != nil {
		return nil, err
	}

	events := make([]*domain.OutboxEvent, len(entEvents))
	for i, entEvent := range entEvents {
		events[i] = r.toDomain(entEvent)
	}
	return events, nil
}

// UpdateStatus updates the status of an outbox event.
func (r *OutboxRepository) UpdateStatus(ctx context.Context, id string, status domain.EventStatus) error {
	_, err := r.client.ent.OutboxEvent.UpdateOneID(id).
		SetStatus(r.statusToEnt(status)).
		Save(ctx)
	return err
}

// IncrementRetryCount increments the retry count for an event.
func (r *OutboxRepository) IncrementRetryCount(ctx context.Context, id string) error {
	event, err := r.client.ent.OutboxEvent.Get(ctx, id)
	if err != nil {
		return err
	}

	_, err = r.client.ent.OutboxEvent.UpdateOneID(id).
		SetRetryCount(event.RetryCount + 1).
		SetStatus(outboxevent.StatusFailed).
		Save(ctx)
	return err
}

// Delete deletes an outbox event by ID.
func (r *OutboxRepository) Delete(ctx context.Context, id string) error {
	return r.client.ent.OutboxEvent.DeleteOneID(id).Exec(ctx)
}

// statusToEnt converts a domain EventStatus to an ent status.
func (r *OutboxRepository) statusToEnt(status domain.EventStatus) outboxevent.Status {
	switch status {
	case domain.EventStatusPending:
		return outboxevent.StatusPending
	case domain.EventStatusProcessed:
		return outboxevent.StatusProcessed
	case domain.EventStatusFailed:
		return outboxevent.StatusFailed
	default:
		return outboxevent.StatusPending
	}
}

// statusFromEnt converts an ent status to a domain EventStatus.
func (r *OutboxRepository) statusFromEnt(status outboxevent.Status) domain.EventStatus {
	switch status {
	case outboxevent.StatusPending:
		return domain.EventStatusPending
	case outboxevent.StatusProcessed:
		return domain.EventStatusProcessed
	case outboxevent.StatusFailed:
		return domain.EventStatusFailed
	default:
		return domain.EventStatusPending
	}
}

// toDomain converts an ent OutboxEvent to a domain OutboxEvent.
func (r *OutboxRepository) toDomain(e *ent.OutboxEvent) *domain.OutboxEvent {
	return &domain.OutboxEvent{
		ID:             domain.MustNewID(e.ID),
		AggregateType:  e.AggregateType,
		AggregateID:   domain.MustNewID(e.AggregateID),
		EventType:    domain.EventType(e.EventType),
		Payload:       []byte(e.Payload["_raw"].(string)),
		OccurredAt:   e.OccurredAt,
		IdempotencyKey: e.IdempotencyKey,
		Status:       r.statusFromEnt(e.Status),
		RetryCount:   e.RetryCount,
	}
}
