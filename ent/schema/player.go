package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// Player holds the schema definition for the Player entity.
type Player struct {
	ent.Schema
}

func (Player) Mixin() []ent.Mixin {
	return []ent.Mixin{
		TimeMixin{},
	}
}

// Fields of the Player.
func (Player) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New).
			Immutable(),
		field.String("name").
			MaxLen(200).
			NotEmpty(),
		field.String("email").
			Unique().
			Optional().
			Nillable(),
		field.String("gender").
			MaxLen(10).
			NotEmpty(),
		field.Time("date_of_birth").
			Optional().
			Nillable(),
		field.Int("jersey_number").
			Optional().
			Nillable(),
		field.String("profile_image_url").
			Optional().
			Nillable(),
		field.Bool("is_captain").
			Default(false).
			Comment("Whether this player is the team captain"),
		field.Bool("is_spirit_captain").
			Default(false).
			Comment("Whether this player is the spirit captain"),
		field.JSON("metadata", map[string]interface{}{}).
			Optional(),
	}
}

// Edges of the Player.
func (Player) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("team", Team.Type).
			Ref("players").
			Unique().
			Required(),
		edge.To("scores", Scoring.Type),
		edge.To("game_events", GameEvent.Type),
		edge.To("mvp_nominations", MVP_Nomination.Type),
		edge.To("spirit_nominations", SpiritNomination.Type),
	}
}
