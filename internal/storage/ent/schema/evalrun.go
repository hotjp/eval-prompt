package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// EvalRun represents a single evaluation execution.
type EvalRun struct {
	ent.Schema
}

// Fields of the EvalRun.
func (EvalRun) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").MaxLen(128).NotEmpty().Unique(),
		field.Enum("status").Values("pending", "running", "passed", "failed").Default("pending"),
		field.Float("deterministic_score").Optional(),
		field.Int("rubric_score").Optional(),
		field.JSON("rubric_details", []RubricCheckResult{}).Optional(),
		field.String("trace_path").MaxLen(512).Optional(),
		field.Int("token_input").Optional(),
		field.Int("token_output").Optional(),
		field.Int("duration_ms").Optional(),
		field.Time("created_at").Default(time.Now),
	}
}

// Edges of the EvalRun.
func (EvalRun) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("eval_case", EvalCase.Type).Ref("eval_runs").Unique().Required(),
		edge.From("snapshot", Snapshot.Type).Ref("eval_runs").Unique().Required(),
	}
}

// Indexes of the EvalRun.
func (EvalRun) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("status"),
	}
}

// RubricCheckResult represents the result of a single rubric check.
type RubricCheckResult struct {
	CheckID   string `json:"check_id"`
	Passed    bool   `json:"passed"`
	Score     int    `json:"score"`
	Details   string `json:"details,omitempty"`
}
