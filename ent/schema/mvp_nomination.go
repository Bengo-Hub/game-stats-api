package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// MVP_Nomination holds the schema definition for the MVP_Nomination entity.
type MVP_Nomination struct {
	ent.Schema
}

// Fields of the MVP_Nomination.
func (MVP_Nomination) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New).
			Immutable(),
		field.String("category").
			MaxLen(20).
			NotEmpty(),
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
	}
}

// Edges of the MVP_Nomination.
func (MVP_Nomination) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("spirit_score", SpiritScore.Type).
			Ref("mvp_nominations").
			Unique().
			Required(),
		edge.From("player", Player.Type).
			Ref("mvp_nominations").
			Unique().
			Required(),
	}
}
