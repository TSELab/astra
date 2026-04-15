package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

type Resource struct {
	ent.Schema
}

func (Resource) Fields() []ent.Field {
	return []ent.Field{
		field.String("astra_id").
			Unique().
			NotEmpty().
			Comment("unique AStRA resource ID"),

		field.String("type").
			Default(""),

		field.String("uri").
			Default(""),

		field.String("format").
			Default(""),

		field.JSON("metadata", map[string]string{}).
			Optional(),

		field.Time("created_at").
			Default(time.Now),
	}
}
