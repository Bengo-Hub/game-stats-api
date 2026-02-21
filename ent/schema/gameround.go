package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// GameRound holds the schema definition for the GameRound entity.
type GameRound struct {
	ent.Schema
}

func (GameRound) Mixin() []ent.Mixin {
	return []ent.Mixin{
		TimeMixin{},
	}
}

// Fields of the GameRound.
func (GameRound) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New).
			Immutable(),
		field.String("name").
			MaxLen(100).
			NotEmpty(),
		field.String("round_type").
			MaxLen(50).
			NotEmpty(),
		field.Int("round_number").
			Optional().
			Nillable(),
		field.Time("start_date").
			Optional().
			Nillable(),
		field.Time("end_date").
			Optional().
			Nillable(),
	}
}

// Edges of the GameRound.
func (GameRound) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("event", Event.Type).
			Ref("game_rounds").
			Unique().
			Required(),
		edge.To("games", Game.Type),
	}
}
