package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// Scoring holds the schema definition for the Scoring entity.
type Scoring struct {
	ent.Schema
}

func (Scoring) Mixin() []ent.Mixin {
	return []ent.Mixin{
		TimeMixin{},
	}
}

// Fields of the Scoring.
func (Scoring) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New).
			Immutable(),
		field.Int("goals").
			Default(0),
		field.Int("assists").
			Default(0),
		field.Int("blocks").
			Default(0),
		field.Int("turns").
			Default(0),
		field.Int("version").
			Default(1),
	}
}

// Edges of the Scoring.
func (Scoring) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("game", Game.Type).
			Ref("scores").
			Unique().
			Required(),
		edge.From("player", Player.Type).
			Ref("scores").
			Unique().
			Required(),
	}
}
