package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// Country holds the schema definition for the Country entity.
type Country struct {
	ent.Schema
}

func (Country) Mixin() []ent.Mixin {
	return []ent.Mixin{
		TimeMixin{},
	}
}

// Fields of the Country.
func (Country) Fields() []ent.Field {
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
		field.String("code").
			MaxLen(3).
			Unique().
			NotEmpty(),
		field.Text("description").
			Optional().
			Nillable(),
	}
}

// Edges of the Country.
func (Country) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("continent", Continent.Type).
			Ref("countries").
			Unique().
			Required(),
		edge.To("locations", Location.Type),
		edge.To("disciplines", Discipline.Type),
		edge.To("managed_by", User.Type),
	}
}
