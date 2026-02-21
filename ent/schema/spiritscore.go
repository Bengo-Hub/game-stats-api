package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// SpiritScore holds the schema definition for the SpiritScore entity.
type SpiritScore struct {
	ent.Schema
}

func (SpiritScore) Mixin() []ent.Mixin {
	return []ent.Mixin{
		TimeMixin{},
	}
}

// Fields of the SpiritScore.
func (SpiritScore) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New).
			Immutable(),
		field.Int("rules_knowledge"),
		field.Int("fouls_body_contact"),
		field.Int("fair_mindedness"),
		field.Int("attitude"),
		field.Int("communication"),
		field.Text("comments").
			Optional().
			Nillable(),
	}
}

// Edges of the SpiritScore.
func (SpiritScore) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("game", Game.Type).
			Ref("spirit_scores").
			Unique().
			Required(),
		edge.From("scored_by_team", Team.Type).
			Ref("spirit_scores_given").
			Unique().
			Required(),
		edge.From("team", Team.Type).
			Ref("spirit_scores_received").
			Unique().
			Required(),
		edge.From("submitted_by", User.Type).
			Ref("submitted_spirit_scores").
			Unique().
			Required(),
		edge.To("mvp_nominations", MVP_Nomination.Type),
		edge.To("spirit_nominations", SpiritNomination.Type),
	}
}
