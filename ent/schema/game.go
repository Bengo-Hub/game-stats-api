package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// Game holds the schema definition for the Game entity.
type Game struct {
	ent.Schema
}

func (Game) Mixin() []ent.Mixin {
	return []ent.Mixin{
		TimeMixin{},
	}
}

// Fields of the Game.
func (Game) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New).
			Immutable(),
		field.String("name").
			MaxLen(100).
			Unique().
			NotEmpty(),
		field.Time("scheduled_time"),
		field.Time("actual_start_time").
			Optional().
			Nillable(),
		field.Time("actual_end_time").
			Optional().
			Nillable(),
		field.Int("allocated_time_minutes"),
		field.Int("stoppage_time_seconds").
			Default(0),
		field.String("status").
			Default("scheduled").
			NotEmpty(),
		field.Int("home_team_score").
			Default(0),
		field.Int("away_team_score").
			Default(0),
		field.String("first_pull_by").
			Optional(),
		field.Int("version").
			Default(1),
		field.JSON("metadata", map[string]interface{}{}).
			Optional(),
	}
}

// Edges of the Game.
func (Game) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("game_round", GameRound.Type).
			Ref("games").
			Unique(),
		edge.From("home_team", Team.Type).
			Ref("home_games").
			Unique().
			Required(),
		edge.From("away_team", Team.Type).
			Ref("away_games").
			Unique().
			Required(),
		edge.From("division_pool", DivisionPool.Type).
			Ref("games").
			Unique().
			Required(),
		edge.From("field_location", Field.Type).
			Ref("games").
			Unique().
			Required(),
		edge.From("scorekeeper", User.Type).
			Ref("officiated_games").
			Unique(),
		edge.To("scores", Scoring.Type),
		edge.To("game_events", GameEvent.Type),
		edge.To("spirit_scores", SpiritScore.Type),
	}
}
