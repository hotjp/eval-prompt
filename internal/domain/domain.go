// Package domain implements L2-Domain layer: domain entities, state machines,
// event collection (Outbox), and business invariants.
// This layer has ZERO external dependencies - pure Go structs + standard library.
package domain

// Entity represents a domain entity with ULID-based ID.
type Entity struct {
	ID      ID
	Version int64
}

// AggregateRoot is the base for all aggregate roots.
type AggregateRoot struct {
	Entity
	events []DomainEventInterface
}

// RecordEvent records a domain event for later publishing via Outbox.
func (a *AggregateRoot) RecordEvent(event DomainEventInterface) {
	a.events = append(a.events, event)
}

// FlushEvents returns and clears recorded events.
func (a *AggregateRoot) FlushEvents() []DomainEventInterface {
	events := a.events
	a.events = nil
	return events
}

// DomainEvent is a legacy type alias for the event interface.
// Deprecated: Use DomainEventInterface instead.
type DomainEvent = DomainEventInterface

// NewDomainEvent creates a new aggregate root with auto-generated ID.
func NewAggregateRoot() *AggregateRoot {
	return &AggregateRoot{
		Entity: Entity{
			ID:      NewAutoID(),
			Version: 0,
		},
	}
}

// IncrementVersion increments the aggregate version.
func (a *AggregateRoot) IncrementVersion() {
	a.Version++
}
