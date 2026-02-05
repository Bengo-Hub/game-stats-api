package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// GameEvent holds the schema definition for the GameEvent entity.
type GameEvent struct {
	ent.Schema
}

func (GameEvent) Mixin() []ent.Mixin {
	return []ent.Mixin{
		TimeMixin{},
	}
}

// Fields of the GameEvent.
func (GameEvent) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New).
			Immutable(),
		field.String("event_type").
			MaxLen(50).
			NotEmpty(),
		field.Int("minute"),
		field.Int("second"),
		field.Text("description").
			Optional(),
		field.JSON("metadata", map[string]interface{}{}).
			Optional(),
	}
}

// Edges of the GameEvent.
func (GameEvent) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("game", Game.Type).
			Ref("game_events").
			Unique().
			Required(),
		edge.From("player", Player.Type).
			Ref("game_events").
			Unique(),
	}
}
