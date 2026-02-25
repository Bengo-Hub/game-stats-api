package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// EventParticipation holds the schema definition for the EventParticipation entity.
type EventParticipation struct {
	ent.Schema
}

func (EventParticipation) Mixin() []ent.Mixin {
	return []ent.Mixin{
		TimeMixin{},
	}
}

// Fields of the EventParticipation.
func (EventParticipation) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New).
			Immutable(),
		field.String("role").
			Default("player").
			Comment("Role in the event: player, captain, spirit_captain, coach"),
		field.Int("jersey_number").
			Optional().
			Nillable(),
		field.String("position").
			Optional().
			Nillable(),
		field.Bool("is_captain").
			Default(false).
			Comment("Whether this player is the team captain in this event"),
		field.Bool("is_spirit_captain").
			Default(false).
			Comment("Whether this player is the spirit captain in this event"),
		field.String("status").
			Default("active").
			Comment("Status in the event: active, injured, transferred, inactive"),
		field.JSON("metadata", map[string]interface{}{}).
			Optional(),
	}
}

// Edges of the EventParticipation.
func (EventParticipation) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("event", Event.Type).
			Ref("participations").
			Unique().
			Required(),
		edge.From("team", Team.Type).
			Ref("participations").
			Unique().
			Required(),
		edge.From("player", Player.Type).
			Ref("participations").
			Unique().
			Required(),
	}
}
