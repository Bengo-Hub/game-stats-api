package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// EventReconciliation holds the schema definition for the EventReconciliation entity.
type EventReconciliation struct {
	ent.Schema
}

func (EventReconciliation) Mixin() []ent.Mixin {
	return []ent.Mixin{
		TimeMixin{},
	}
}

// Fields of the EventReconciliation.
func (EventReconciliation) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New).
			Immutable(),
		field.Time("reconciled_at").
			Default(time.Now),
		field.String("reconciled_by").
			NotEmpty(),
		field.String("status").
			Default("pending").
			NotEmpty(),
		field.Text("comments").
			Optional().
			Nillable(),
	}
}

// Edges of the EventReconciliation.
func (EventReconciliation) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("event", Event.Type).
			Ref("reconciliations").
			Unique().
			Required(),
	}
}
