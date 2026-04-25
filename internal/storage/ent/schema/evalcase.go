package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// EvalCase represents a test case for evaluating a prompt asset.
type EvalCase struct {
	ent.Schema
}

// Fields of the EvalCase.
func (EvalCase) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").MaxLen(128).NotEmpty().Unique(),
		field.String("name").MaxLen(128).NotEmpty(),
		field.Text("prompt").NotEmpty(),
		field.Bool("should_trigger").Default(true),
		field.Text("expected_output").Optional(),
		field.JSON("rubric", Rubric{}).Optional(),
		field.Time("created_at").Default(time.Now),
	}
}

// Edges of the EvalCase.
func (EvalCase) Edges() []ent.Edge {
	return nil
}

// Indexes of the EvalCase.
func (EvalCase) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("name"),
	}
}

// Rubric defines the evaluation rubric structure.
type Rubric struct {
	MaxScore int           `json:"max_score"`
	Checks   []RubricCheck `json:"checks"`
}

// RubricCheck defines a single check in the rubric.
type RubricCheck struct {
	ID          string `json:"id"`
	Description string `json:"description"`
	Weight      int    `json:"weight"`
}
