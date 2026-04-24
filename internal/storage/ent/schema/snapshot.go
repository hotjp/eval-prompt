package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// Snapshot represents a version of an asset.
type Snapshot struct {
	ent.Schema
}

// Fields of the Snapshot.
func (Snapshot) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").MaxLen(128).NotEmpty().Unique(),
		field.String("version").MaxLen(32).NotEmpty(),
		field.String("content_hash").MaxLen(64).NotEmpty(),
		field.String("commit_hash").MaxLen(40).Optional(),
		field.String("author").MaxLen(128).Optional(),
		field.String("reason").MaxLen(512).Optional(),
		field.String("model").MaxLen(64).Optional(),
		field.Float("temperature").Optional(),
		field.JSON("metrics", map[string]any{}).Optional(),
		field.Time("created_at").Default(time.Now),
	}
}

// Edges of the Snapshot.
func (Snapshot) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("asset", Asset.Type).Ref("snapshots").Unique().Required(),
		edge.To("eval_runs", EvalRun.Type),
		edge.To("labels", Label.Type),
	}
}

// Indexes of the Snapshot.
func (Snapshot) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("version"),
		index.Fields("commit_hash"),
	}
}
