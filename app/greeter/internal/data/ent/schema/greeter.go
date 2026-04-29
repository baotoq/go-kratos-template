package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

type Greeter struct {
	ent.Schema
}

func (Greeter) Fields() []ent.Field {
	return []ent.Field{
		field.String("hello"),
	}
}

func (Greeter) Edges() []ent.Edge {
	return nil
}
