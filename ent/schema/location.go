package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// Location holds the schema definition for the Location entity.
type Location struct {
	ent.Schema
}

func (Location) Mixin() []ent.Mixin {
	return []ent.Mixin{
		TimeMixin{},
	}
}

// Fields of the Location.
func (Location) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New).
			Immutable(),
		field.String("name").
			MaxLen(200).
			NotEmpty(),
		field.String("slug").
			MaxLen(200).
			Unique().
			NotEmpty(),
		field.String("address").
			MaxLen(500).
			Optional().
			Nillable(),
		field.String("city").
			MaxLen(100).
			Optional().
			Nillable(),
		field.Float("latitude").
			Optional().
			Nillable(),
		field.Float("longitude").
			Optional().
			Nillable(),
		field.Text("description").
			Optional().
			Nillable(),
	}
}

// Edges of the Location.
func (Location) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("country", Country.Type).
			Ref("locations").
			Unique().
			Required(),
		edge.To("fields", Field.Type),
		edge.To("events", Event.Type),
		edge.To("teams", Team.Type),
	}
}
