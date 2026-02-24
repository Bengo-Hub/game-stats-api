package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// ScopedRole holds the schema definition for the ScopedRole entity.
type ScopedRole struct {
	ent.Schema
}

func (ScopedRole) Mixin() []ent.Mixin {
	return []ent.Mixin{
		TimeMixin{},
	}
}

// Fields of the ScopedRole.
func (ScopedRole) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New).
			Immutable(),
		field.UUID("user_id", uuid.UUID{}),
		field.String("role").
			NotEmpty().
			Comment("role name: event_manager, game_admin, scorekeeper, etc."),
		field.String("scope_type").
			NotEmpty().
			Comment("event, game, team, division"),
		field.UUID("scope_id", uuid.UUID{}),
		field.JSON("permissions", []string{}).
			Optional(),
	}
}

// Edges of the ScopedRole.
func (ScopedRole) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).
			Ref("scoped_roles").
			Field("user_id").
			Unique().
			Required(),
	}
}
