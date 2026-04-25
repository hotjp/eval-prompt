package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// Asset represents a prompt asset in the system.
type Asset struct {
	ent.Schema
}

// Fields of the Asset.
func (Asset) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").MaxLen(128).NotEmpty().Unique(),
		field.String("name").MaxLen(100).NotEmpty(),
		field.Text("description"),
		field.String("biz_line").MaxLen(64).Optional(),
		field.JSON("tags", []string{}).Optional(),
		field.String("content_hash").MaxLen(64).NotEmpty(),
		field.String("file_path").MaxLen(512).NotEmpty(),
		field.Enum("state").Values("created", "evaluating", "evaluated", "promoted", "archived").Default("created"),
		field.Time("created_at").Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

// Edges of the Asset.
func (Asset) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("labels", Label.Type),
		edge.To("eval_cases", EvalCase.Type),
		edge.To("adaptations", ModelAdaptation.Type),
	}
}

// Indexes of the Asset.
func (Asset) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("biz_line"),
		index.Fields("state"),
	}
}
