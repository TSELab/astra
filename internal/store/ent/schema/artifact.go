package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

type Artifact struct {
	ent.Schema
}

func (Artifact) Fields() []ent.Field {
	return []ent.Field{
		field.String("astra_id").
			Unique().
			NotEmpty().
			Comment("unique AStRA artifact ID"),

		field.String("name").
			Default(""),

		field.String("kind").
			Default("").
			Comment("e.g. source_file, repo_state, binary, sbom"),

		field.String("uri").
			Default("").
			Comment("Optional original locator/path/URL"),

		field.String("version").
			Default(""),

		field.String("hash").
			Default("").
			Comment("Optional content hash"),

		field.Int64("size").
			Default(0),

		field.JSON("metadata", map[string]string{}).
			Optional(),

		field.Enum("completeness").
			Values("complete", "incomplete").
			Default("complete"),

		field.Time("created_at").
			Default(time.Now),
	}
}
