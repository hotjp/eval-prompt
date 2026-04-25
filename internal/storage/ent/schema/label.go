package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// Label represents a named pointer to an asset at a specific version (e.g., "prod", "dev").
type Label struct {
	ent.Schema
}

// Fields of the Label.
func (Label) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").MaxLen(128).NotEmpty().Unique(),
		field.String("name").MaxLen(32).NotEmpty(),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

// Edges of the Label.
func (Label) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("asset", Asset.Type).Ref("labels").Unique().Required(),
	}
}

// Indexes of the Label.
func (Label) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("name"),
	}
}
