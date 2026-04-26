package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// EvalWorkItem represents a single unit of work (Case × Run) in an execution.
type EvalWorkItem struct {
	ent.Schema
}

// Fields of the EvalWorkItem.
func (EvalWorkItem) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").MaxLen(128).NotEmpty().Unique(),
		field.String("execution_id").MaxLen(128).NotEmpty(),
		field.String("eval_case_id").MaxLen(128).NotEmpty(),
		field.Int("run_number").Default(1),
		field.Enum("status").Values("pending", "running", "completed", "failed", "cancelled").Default("pending"),
		field.String("prompt_hash").MaxLen(64).Optional(),
		field.Text("prompt_text").Optional(),
		field.Text("response").Optional(),
		field.String("model").MaxLen(64).Optional(),
		field.Float("temperature").Default(0.0),
		field.Int("tokens_in").Default(0),
		field.Int("tokens_out").Default(0),
		field.Int("duration_ms").Default(0),
		field.Text("error").Optional(),
		field.Time("created_at").Default(time.Now),
		field.Time("completed_at").Optional(),
	}
}

// Edges of the EvalWorkItem.
func (EvalWorkItem) Edges() []ent.Edge {
	return nil
}

// Indexes of the EvalWorkItem.
func (EvalWorkItem) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("execution_id"),
		index.Fields("eval_case_id"),
		index.Fields("status"),
		index.Fields("prompt_hash"),
	}
}
