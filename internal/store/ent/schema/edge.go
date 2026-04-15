package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type Edge struct {
	ent.Schema
}

func (Edge) Fields() []ent.Field {
	return []ent.Field{
		field.String("source"),
		field.String("target"),
		field.String("relation"),
	}
}

func (Edge) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("source", "target", "relation").Unique(),
	}
}
