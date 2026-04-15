package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

type Principal struct {
	ent.Schema
}

func (Principal) Fields() []ent.Field {
	return []ent.Field{
		field.String("astra_id").
			Unique().
			NotEmpty().
			Comment("unique AStRA principal ID"),

		field.String("name").
			Default(""),

		field.String("trust").
			Default(""),

		field.String("builder").
			Default(""),

		field.JSON("metadata", map[string]string{}).
			Optional(),

		field.Time("created_at").
			Default(time.Now),
	}
}
