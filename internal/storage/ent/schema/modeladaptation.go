package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// ModelAdaptation represents a prompt adaptation for a different model.
type ModelAdaptation struct {
	ent.Schema
}

// Fields of the ModelAdaptation.
func (ModelAdaptation) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").MaxLen(128).NotEmpty().Unique(),
		field.String("prompt_id").MaxLen(128).NotEmpty(),
		field.String("source_model").MaxLen(64).NotEmpty(),
		field.String("target_model").MaxLen(64).NotEmpty(),
		field.Text("adapted_content").NotEmpty(),
		field.JSON("param_adjustments", map[string]float64{}).Optional(),
		field.JSON("format_changes", []string{}).Optional(),
		field.Float("eval_score").Optional(),
		field.String("eval_run_id").MaxLen(128).Optional(),
		field.Time("created_at").Default(time.Now),
	}
}

// Edges of the ModelAdaptation.
func (ModelAdaptation) Edges() []ent.Edge {
	return nil
}

// Indexes of the ModelAdaptation.
func (ModelAdaptation) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("source_model"),
		index.Fields("target_model"),
	}
}
