package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// DivisionPool holds the schema definition for the DivisionPool entity.
type DivisionPool struct {
	ent.Schema
}

func (DivisionPool) Mixin() []ent.Mixin {
	return []ent.Mixin{
		TimeMixin{},
	}
}

// Fields of the DivisionPool.
func (DivisionPool) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New).
			Immutable(),
		field.String("name").
			MaxLen(100).
			NotEmpty(),
		field.String("division_type").
			MaxLen(50).
			NotEmpty(),
		field.Int("max_teams").
			Optional().
			Nillable(),
		field.JSON("ranking_criteria", map[string]interface{}{}).
			Optional(),
		field.Text("description").
			Optional().
			Nillable(),
	}
}

// Edges of the DivisionPool.
func (DivisionPool) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("event", Event.Type).
			Ref("division_pools").
			Unique().
			Required(),
		edge.To("teams", Team.Type),
		edge.To("games", Game.Type),
	}
}
