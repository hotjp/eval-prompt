package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// AuditLog represents an audit trail entry for operations.
type AuditLog struct {
	ent.Schema
}

// Fields of the AuditLog.
func (AuditLog) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").MaxLen(128).NotEmpty().Unique(),
		field.String("operation").MaxLen(64).NotEmpty(),
		field.String("asset_id").MaxLen(128).Optional(),
		field.String("user_id").MaxLen(128).Optional(),
		field.JSON("details", map[string]any{}).Optional(),
		field.Time("created_at").Default(time.Now),
	}
}

// Edges of the AuditLog.
func (AuditLog) Edges() []ent.Edge {
	return nil
}

// Indexes of the AuditLog.
func (AuditLog) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("operation"),
		index.Fields("asset_id"),
	}
}
