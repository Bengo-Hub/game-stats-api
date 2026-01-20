package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// World holds the schema definition for the World entity.
type World struct {
	ent.Schema
}

func (World) Mixin() []ent.Mixin {
	return []ent.Mixin{
		TimeMixin{},
	}
}

// Fields of the World.
func (World) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New).
			Immutable(),
		field.String("name").
			MaxLen(100).
			Unique().
			NotEmpty(),
		field.String("slug").
			MaxLen(100).
			Unique().
			NotEmpty(),
		field.Text("description").
			Optional(),
	}
}

// Edges of the World.
func (World) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("continents", Continent.Type),
	}
}


