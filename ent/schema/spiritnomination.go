package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// SpiritNomination holds the schema definition for the SpiritNomination entity.
type SpiritNomination struct {
	ent.Schema
}

// Fields of the SpiritNomination.
func (SpiritNomination) Fields() []ent.Field {
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

// Edges of the SpiritNomination.
func (SpiritNomination) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("spirit_score", SpiritScore.Type).
			Ref("spirit_nominations").
			Unique().
			Required(),
		edge.From("player", Player.Type).
			Ref("spirit_nominations").
			Unique().
			Required(),
	}
}
