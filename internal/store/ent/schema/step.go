package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

type Step struct {
	ent.Schema
}

func (Step) Fields() []ent.Field {
	return []ent.Field{
		field.String("astra_id").
			Unique().
			NotEmpty().
			Comment("unique AStRA step ID"),

		field.String("command").
			Default(""),

		field.JSON("environment", map[string]string{}),

		field.String("Arch").
			Default("").
			Comment("Optional higher-level grouping: source, build, review, package"),

		field.String("timestamp"),

		field.JSON("metadata", map[string]string{}).
			Optional(),

		field.Enum("completeness").
			Values("complete", "incomplete").
			Default("complete"),

		field.Time("created_at").
			Default(time.Now),
	}
}
