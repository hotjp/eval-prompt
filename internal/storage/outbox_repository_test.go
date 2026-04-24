package storage

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/eval-prompt/internal/domain"
	"github.com/eval-prompt/internal/storage/ent/enttest"
)

func TestOutboxRepository_Create(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file::memory:?_fk=1&_journal_mode=WAL")
	defer client.Close()

	repo := NewOutboxRepository(&Client{ent: client})
	ctx := context.Background()

	payload := []byte(`{"key": "value"}`)
	event := &domain.OutboxEvent{
		ID:             domain.MustNewID("01ARZ3NDEKTSV4RRFFQ69G5FAV"),
		AggregateType:  "Asset",
		AggregateID:    domain.MustNewID("01ARZ3NDEKTSV4RRFFQ69G5FAW"),
		EventType:      domain.EventType("AssetCreated"),
		Payload:        payload,
		OccurredAt:     domain.Time{}.Time(),
		IdempotencyKey: "key123",
		Status:         domain.EventStatusPending,
	}

	err := repo.Create(ctx, event)
	if err != nil {
		t.Fatalf("failed to create outbox event: %v", err)
	}

	got, err := repo.GetByID(ctx, event.ID.String())
	if err != nil {
		t.Fatalf("failed to get outbox event: %v", err)
	}
	if got.AggregateType != event.AggregateType {
		t.Errorf("expected aggregate type %q, got %q", event.AggregateType, got.AggregateType)
	}
	if got.EventType != event.EventType {
		t.Errorf("expected event type %q, got %q", event.EventType, got.EventType)
	}
}

func TestOutboxRepository_GetPendingEvents(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file::memory:?_fk=1&_journal_mode=WAL")
	defer client.Close()

	repo := NewOutboxRepository(&Client{ent: client})
	ctx := context.Background()

	// Create events with different statuses
	statuses := []domain.EventStatus{
		domain.EventStatusPending,
		domain.EventStatusPending,
		domain.EventStatusProcessed,
	}
	for i, status := range statuses {
		event := &domain.OutboxEvent{
			ID:             domain.NewAutoID(),
			AggregateType:  "Asset",
			AggregateID:    domain.NewAutoID(),
			EventType:      domain.EventType("TestEvent"),
			Payload:        []byte(`{}`),
			OccurredAt:     domain.Time{}.Time(),
			IdempotencyKey: "key" + string(rune('0'+i)),
			Status:         status,
		}
		err := repo.Create(ctx, event)
		if err != nil {
			t.Fatalf("failed to create event %d: %v", i, err)
		}
	}

	events, err := repo.GetPendingEvents(ctx, 10)
	if err != nil {
		t.Fatalf("failed to get pending events: %v", err)
	}
	if len(events) != 2 {
		t.Errorf("expected 2 pending events, got %d", len(events))
	}
}

func TestOutboxRepository_UpdateStatus(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file::memory:?_fk=1&_journal_mode=WAL")
	defer client.Close()

	repo := NewOutboxRepository(&Client{ent: client})
	ctx := context.Background()

	event := &domain.OutboxEvent{
		ID:             domain.MustNewID("01ARZ3NDEKTSV4RRFFQ69G5FAV"),
		AggregateType:  "Asset",
		AggregateID:    domain.NewAutoID(),
		EventType:      domain.EventType("TestEvent"),
		Payload:        []byte(`{}`),
		OccurredAt:     domain.Time{}.Time(),
		IdempotencyKey: "key123",
		Status:         domain.EventStatusPending,
	}
	err := repo.Create(ctx, event)
	if err != nil {
		t.Fatalf("failed to create event: %v", err)
	}

	err = repo.UpdateStatus(ctx, event.ID.String(), domain.EventStatusProcessed)
	if err != nil {
		t.Fatalf("failed to update status: %v", err)
	}

	got, err := repo.GetByID(ctx, event.ID.String())
	if err != nil {
		t.Fatalf("failed to get event: %v", err)
	}
	if got.Status != domain.EventStatusProcessed {
		t.Errorf("expected status %q, got %q", domain.EventStatusProcessed, got.Status)
	}
}

func TestOutboxRepository_IncrementRetryCount(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file::memory:?_fk=1&_journal_mode=WAL")
	defer client.Close()

	repo := NewOutboxRepository(&Client{ent: client})
	ctx := context.Background()

	event := &domain.OutboxEvent{
		ID:             domain.MustNewID("01ARZ3NDEKTSV4RRFFQ69G5FAV"),
		AggregateType:  "Asset",
		AggregateID:    domain.NewAutoID(),
		EventType:      domain.EventType("TestEvent"),
		Payload:        []byte(`{}`),
		OccurredAt:     domain.Time{}.Time(),
		IdempotencyKey: "key123",
		Status:         domain.EventStatusPending,
		RetryCount:     0,
	}
	err := repo.Create(ctx, event)
	if err != nil {
		t.Fatalf("failed to create event: %v", err)
	}

	err = repo.IncrementRetryCount(ctx, event.ID.String())
	if err != nil {
		t.Fatalf("failed to increment retry count: %v", err)
	}

	got, err := repo.GetByID(ctx, event.ID.String())
	if err != nil {
		t.Fatalf("failed to get event: %v", err)
	}
	if got.RetryCount != 1 {
		t.Errorf("expected retry count 1, got %d", got.RetryCount)
	}
}

func TestOutboxRepository_Delete(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file::memory:?_fk=1&_journal_mode=WAL")
	defer client.Close()

	repo := NewOutboxRepository(&Client{ent: client})
	ctx := context.Background()

	event := &domain.OutboxEvent{
		ID:             domain.MustNewID("01ARZ3NDEKTSV4RRFFQ69G5FAV"),
		AggregateType:  "Asset",
		AggregateID:    domain.NewAutoID(),
		EventType:      domain.EventType("TestEvent"),
		Payload:        []byte(`{}`),
		OccurredAt:     domain.Time{}.Time(),
		IdempotencyKey: "key123",
		Status:         domain.EventStatusProcessed,
	}
	err := repo.Create(ctx, event)
	if err != nil {
		t.Fatalf("failed to create event: %v", err)
	}

	err = repo.Delete(ctx, event.ID.String())
	if err != nil {
		t.Fatalf("failed to delete event: %v", err)
	}

	_, err = repo.GetByID(ctx, event.ID.String())
	if err == nil {
		t.Error("expected error when getting deleted event")
	}
}

func TestOutboxRepository_ToDomainOutboxEvent(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file::memory:?_fk=1&_journal_mode=WAL")
	defer client.Close()

	repo := NewOutboxRepository(&Client{ent: client})
	ctx := context.Background()

	payloadMap := map[string]interface{}{"key": "value"}
	payloadBytes, _ := json.Marshal(payloadMap)

	event := &domain.OutboxEvent{
		ID:             domain.MustNewID("01ARZ3NDEKTSV4RRFFQ69G5FAV"),
		AggregateType:  "Asset",
		AggregateID:    domain.MustNewID("01ARZ3NDEKTSV4RRFFQ69G5FAW"),
		EventType:      domain.EventType("AssetCreated"),
		Payload:        payloadBytes,
		OccurredAt:     domain.Time{}.Time(),
		IdempotencyKey: "key123",
		Status:         domain.EventStatusPending,
		RetryCount:     2,
	}

	err := repo.Create(ctx, event)
	if err != nil {
		t.Fatalf("failed to create event: %v", err)
	}

	got, err := repo.GetByID(ctx, event.ID.String())
	if err != nil {
		t.Fatalf("failed to get event: %v", err)
	}

	if got.ID.String() != event.ID.String() {
		t.Errorf("expected ID %q, got %q", event.ID.String(), got.ID.String())
	}
	if got.AggregateType != event.AggregateType {
		t.Errorf("expected aggregate type %q, got %q", event.AggregateType, got.AggregateType)
	}
	if got.EventType != event.EventType {
		t.Errorf("expected event type %q, got %q", event.EventType, got.EventType)
	}
	if got.Status != event.Status {
		t.Errorf("expected status %q, got %q", event.Status, got.Status)
	}
	if got.RetryCount != event.RetryCount {
		t.Errorf("expected retry count %d, got %d", event.RetryCount, got.RetryCount)
	}
}