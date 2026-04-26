package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
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
		field.JSON("tags", []string{}).Optional(),
		field.String("content_hash").MaxLen(128).NotEmpty(),
		field.String("file_path").MaxLen(512).NotEmpty(),
		field.String("repo_path").MaxLen(512).Optional(), // repo isolation
		field.Enum("state").Values("created", "evaluating", "evaluated", "promoted", "archived").Default("created"),
		field.String("asset_type").MaxLen(64).Optional(),
		field.String("category").MaxLen(32).Optional(), // content/eval/metric
	}
}

// Edges of the Asset.
func (Asset) Edges() []ent.Edge {
	return nil
}

// Indexes of the Asset.
func (Asset) Indexes() []ent.Index {
	return nil
}
