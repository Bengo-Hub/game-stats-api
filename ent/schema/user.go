package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// User holds the schema definition for the User entity.
type User struct {
	ent.Schema
}

func (User) Mixin() []ent.Mixin {
	return []ent.Mixin{
		TimeMixin{},
	}
}

// Fields of the User.
func (User) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New).
			Immutable(),
		field.String("email").
			NotEmpty().
			Unique(),
		field.String("password_hash").
			Sensitive().
			NotEmpty(),
		field.String("full_name").
			MaxLen(200).
			NotEmpty(),
		field.String("role").
			NotEmpty(),
		field.Bool("is_active").
			Default(true),
		field.Time("last_login_at").
			Optional().
			Nillable(),
	}
}

// Edges of the User.
func (User) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("managed_continent", Continent.Type).
			Ref("managed_by").
			Unique(),
		edge.From("managed_country", Country.Type).
			Ref("managed_by").
			Unique(),
		edge.From("managed_discipline", Discipline.Type).
			Ref("managed_by").
			Unique(),
		edge.From("managed_event", Event.Type).
			Ref("managed_by").
			Unique(),
		edge.From("managed_team", Team.Type).
			Ref("managed_by").
			Unique(),
		edge.To("officiated_games", Game.Type),
		edge.To("submitted_spirit_scores", SpiritScore.Type),
	}
}


