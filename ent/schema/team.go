package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// Team holds the schema definition for the Team entity.
type Team struct {
	ent.Schema
}

func (Team) Mixin() []ent.Mixin {
	return []ent.Mixin{
		TimeMixin{},
	}
}

// Fields of the Team.
func (Team) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New).
			Immutable(),
		field.String("name").
			MaxLen(100).
			NotEmpty(),
		field.Int("initial_seed").
			Optional().
			Nillable(),
		field.Int("final_placement").
			Optional().
			Nillable(),
		field.String("logo_url").
			Optional().
			Nillable(),
		field.String("primary_color").
			Optional().
			Nillable(),
		field.String("secondary_color").
			Optional().
			Nillable(),
		field.String("contact_email").
			Optional().
			Nillable(),
		field.String("contact_phone").
			Optional().
			Nillable(),
		field.JSON("metadata", map[string]interface{}{}).
			Optional(),
		field.UUID("division_pool_id", uuid.UUID{}),
		field.UUID("home_location_id", uuid.UUID{}).
			Optional().
			Nillable(),
	}
}

// Edges of the Team.
func (Team) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("division_pool", DivisionPool.Type).
			Ref("teams").
			Field("division_pool_id").
			Unique().
			Required(),
		edge.From("home_location", Location.Type).
			Ref("teams").
			Field("home_location_id").
			Unique(),
		edge.To("players", Player.Type),
		edge.To("managed_by", User.Type),
		edge.To("home_games", Game.Type),
		edge.To("away_games", Game.Type),
		edge.To("spirit_scores_given", SpiritScore.Type),
		edge.To("spirit_scores_received", SpiritScore.Type),
		edge.To("participations", EventParticipation.Type),
	}
}
