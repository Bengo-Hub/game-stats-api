package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// Event holds the schema definition for the Event entity.
type Event struct {
	ent.Schema
}

func (Event) Mixin() []ent.Mixin {
	return []ent.Mixin{
		TimeMixin{},
	}
}

// Fields of the Event.
func (Event) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New).
			Immutable(),
		field.String("name").
			MaxLen(200).
			NotEmpty(),
		field.String("slug").
			MaxLen(200).
			Unique().
			NotEmpty(),
		field.Int("year"),
		field.Time("start_date"),
		field.Time("end_date"),
		field.String("status").
			Default("draft").
			NotEmpty(),
		field.Text("description").
			Optional().
			Nillable(),
		field.JSON("settings", map[string]interface{}{}).
			Optional(),
		// New fields for enhanced events
		field.Strings("categories").
			Optional().
			Comment("Event categories: outdoor, hat, beach, indoor, league"),
		field.String("logo_url").
			MaxLen(500).
			Optional().
			Nillable().
			Comment("URL to event logo image"),
		field.String("banner_url").
			MaxLen(500).
			Optional().
			Nillable().
			Comment("URL to event banner image"),
		field.Int("teams_count").
			Default(0).
			Comment("Denormalized count of teams for efficient queries"),
		field.Int("games_count").
			Default(0).
			Comment("Denormalized count of games for efficient queries"),
	}
}

// Edges of the Event.
func (Event) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("discipline", Discipline.Type).
			Ref("events").
			Unique().
			Required(),
		edge.From("location", Location.Type).
			Ref("events").
			Unique().
			Required(),
		edge.To("division_pools", DivisionPool.Type).
			Annotations(entsql.Annotation{
				OnDelete: entsql.Cascade,
			}),
		edge.To("reconciliations", EventReconciliation.Type).
			Annotations(entsql.Annotation{
				OnDelete: entsql.Cascade,
			}),
		edge.To("game_rounds", GameRound.Type),
		edge.To("managed_by", User.Type),
	}
}
