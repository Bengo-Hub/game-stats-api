package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// Discipline holds the schema definition for the Discipline entity.
type Discipline struct {
	ent.Schema
}

func (Discipline) Mixin() []ent.Mixin {
	return []ent.Mixin{
		TimeMixin{},
	}
}

// Fields of the Discipline.
func (Discipline) Fields() []ent.Field {
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
		field.String("rules_pdf_url").
			Optional().
			Nillable(),
	}
}

// Edges of the Discipline.
func (Discipline) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("country", Country.Type).
			Ref("disciplines").
			Unique().
			Required(),
		edge.To("events", Event.Type),
		edge.To("managed_by", User.Type),
	}
}
