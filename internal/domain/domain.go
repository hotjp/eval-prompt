// Package domain implements L2-Domain layer: domain entities, state machines,
// event collection (Outbox), and business invariants.
// This layer has ZERO external dependencies - pure Go structs + standard library.
package domain

// Entity represents a domain entity with ULID-based ID.
type Entity struct {
	ID      string
	Version int64
}

// AggregateRoot is the base for all aggregate roots.
type AggregateRoot struct {
	Entity
	events []DomainEvent
}

// DomainEvent represents a domain event for Outbox pattern.
type DomainEvent struct {
	EventID          string                 `json:"event_id"`
	AggregateType    string                 `json:"aggregate_type"`
	AggregateID      string                 `json:"aggregate_id"`
	EventType        string                 `json:"event_type"`
	Payload          map[string]interface{} `json:"payload"`
	OccurredAt       string                 `json:"occurred_at"`
	IdempotencyKey   string                 `json:"idempotency_key"`
	Version          int64                  `json:"version"`
}

// RecordEvent records a domain event for later publishing via Outbox.
func (a *AggregateRoot) RecordEvent(event DomainEvent) {
	a.events = append(a.events, event)
}

// FlushEvents returns and clears recorded events.
func (a *AggregateRoot) FlushEvents() []DomainEvent {
	events := a.events
	a.events = nil
	return events
}
