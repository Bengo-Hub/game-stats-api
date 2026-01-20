package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// Entity reconciliation holds the schema definition for the EventReconciliation entity.
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
		field.String("name").
			NotEmpty(),
		field.Text("description").
			Optional(),
		field.Bool("is_active").
			Default(true),
	}
}

// Edges of the EventReconciliation.
func (EventReconciliation) Edges() []ent.Edge {
	return nil
}
