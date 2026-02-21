package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// AnalyticSearch holds the schema definition for the AnalyticSearch entity.
type AnalyticSearch struct {
	ent.Schema
}

func (AnalyticSearch) Mixin() []ent.Mixin {
	return []ent.Mixin{
		TimeMixin{},
	}
}

// Fields of the AnalyticSearch.
func (AnalyticSearch) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New).
			Immutable(),
		field.Text("query").
			NotEmpty(),
		field.Text("explanation").
			Optional(),
		field.Text("generated_sql").
			Optional(),
		field.JSON("metadata", map[string]interface{}{}).
			Optional(),
	}
}

// Edges of the AnalyticSearch.
func (AnalyticSearch) Edges() []ent.Edge {
	return nil
}
