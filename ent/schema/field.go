package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// Field holds the schema definition for the Field entity.
type Field struct {
	ent.Schema
}

func (Field) Mixin() []ent.Mixin {
	return []ent.Mixin{
		TimeMixin{},
	}
}

// Fields of the Field.
func (Field) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New).
			Immutable(),
		field.String("name").
			MaxLen(100).
			NotEmpty(),
		field.Int("capacity").
			Optional().
			Nillable(),
		field.String("surface_type").
			MaxLen(50).
			Optional().
			Nillable(),
		field.JSON("metadata", map[string]interface{}{}).
			Optional(),
	}
}

// Edges of the Field.
func (Field) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("location", Location.Type).
			Ref("fields").
			Unique().
			Required(),
		edge.To("games", Game.Type),
	}
}
