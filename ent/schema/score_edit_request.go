package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// ScoreEditRequest holds the schema definition for the ScoreEditRequest entity.
type ScoreEditRequest struct {
	ent.Schema
}

// Fields of the ScoreEditRequest.
func (ScoreEditRequest) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New).
			Immutable(),
		field.UUID("game_id", uuid.UUID{}),
		field.UUID("requested_by_id", uuid.UUID{}),
		field.Int("previous_home_score"),
		field.Int("previous_away_score"),
		field.Int("new_home_score"),
		field.Int("new_away_score"),
		field.String("reason").
			NotEmpty(),
		field.String("status").
			Default("pending").
			Comment("pending, approved, rejected"),
		field.String("rejection_reason").
			Optional(),
		field.UUID("reviewed_by_id", uuid.UUID{}).
			Optional(),
		field.Time("reviewed_at").
			Optional(),
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now),
		field.JSON("player_scores", []map[string]interface{}{}).
			Optional(),
	}
}

// Edges of the ScoreEditRequest.
func (ScoreEditRequest) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("game", Game.Type).
			Ref("score_edit_requests").
			Field("game_id").
			Unique().
			Required(),
		edge.From("requested_by", User.Type).
			Ref("score_edit_requests").
			Field("requested_by_id").
			Unique().
			Required(),
		edge.To("reviewed_by", User.Type).
			Field("reviewed_by_id").
			Unique(),
	}
}
