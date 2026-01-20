package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// Continent holds the schema definition for the Continent entity.
type Continent struct {
	ent.Schema
}

func (Continent) Mixin() []ent.Mixin {
	return []ent.Mixin{
		TimeMixin{},
	}
}

// Fields of the Continent.
func (Continent) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New).
			Immutable(),
		field.String("name").
			MaxLen(100).
			NotEmpty(),
		field.String("slug").
			MaxLen(100).
			Unique().
			NotEmpty(),
		field.Text("description").
			Optional().
			Nillable(),
		field.UUID("world_id", uuid.UUID{}),
	}
}

// Edges of the Continent.
func (Continent) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("world", World.Type).
			Ref("continents").
			Unique().
			Field("world_id").
			Required(),
		edge.To("countries", Country.Type),
		edge.To("managed_by", User.Type),
	}
}
