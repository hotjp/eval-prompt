package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// EvalExecution represents an eval execution batch.
type EvalExecution struct {
	ent.Schema
}

// Fields of the EvalExecution.
func (EvalExecution) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").MaxLen(128).NotEmpty().Unique(),
		field.String("asset_id").MaxLen(128).NotEmpty(),
		field.String("snapshot_id").MaxLen(128),
		field.Enum("mode").Values("single", "batch", "matrix").Default("single"),
		field.Int("runs_per_case").Default(1),
		field.JSON("case_ids", []string{}),
		field.Int("total_runs"),
		field.Int("completed_runs").Default(0),
		field.Int("failed_runs").Default(0),
		field.Enum("status").Values("pending", "running", "completed", "partial_failure", "failed", "cancelled").Default("pending"),
		field.Int("concurrency").Default(1),
		field.String("model").MaxLen(64).Optional(),
		field.Float("temperature").Default(0.0),
		field.Time("created_at").Default(time.Now),
		field.Time("started_at").Optional(),
		field.Time("completed_at").Optional(),
	}
}

// Edges of the EvalExecution.
func (EvalExecution) Edges() []ent.Edge {
	return nil
}

// Indexes of the EvalExecution.
func (EvalExecution) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("asset_id"),
		index.Fields("status"),
		index.Fields("created_at"),
	}
}
