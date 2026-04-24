package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// OutboxEvent represents a domain event stored for reliable delivery.
type OutboxEvent struct {
	ent.Schema
}

// Fields of the OutboxEvent.
func (OutboxEvent) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").MaxLen(128).NotEmpty().Unique(),
		field.String("aggregate_type").MaxLen(64).NotEmpty(),
		field.String("aggregate_id").MaxLen(128).NotEmpty(),
		field.String("event_type").MaxLen(128).NotEmpty(),
		field.JSON("payload", map[string]any{}),
		field.Time("occurred_at").Default(time.Now),
		field.String("idempotency_key").MaxLen(256).Unique(),
		field.Enum("status").Values("pending", "processed", "failed").Default("pending"),
		field.Int("retry_count").Default(0),
		field.Time("created_at").Default(time.Now),
	}
}

// Edges of the OutboxEvent.
func (OutboxEvent) Edges() []ent.Edge {
	return nil
}

// Indexes of the OutboxEvent.
func (OutboxEvent) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("status"),
		index.Fields("aggregate_id"),
	}
}
