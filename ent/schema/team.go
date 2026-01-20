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
			Unique().
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
		field.JSON("metadata", map[string]interface{}{}).
			Optional(),
	}
}

// Edges of the Team.
func (Team) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("division_pool", DivisionPool.Type).
			Ref("teams").
			Unique().
			Required(),
		edge.From("home_location", Location.Type).
			Ref("teams").
			Unique(),
		edge.To("players", Player.Type),
		edge.To("managed_by", User.Type),
		edge.To("home_games", Game.Type),
		edge.To("away_games", Game.Type),
		edge.To("spirit_scores_given", SpiritScore.Type),
		edge.To("spirit_scores_received", SpiritScore.Type),
	}
}
