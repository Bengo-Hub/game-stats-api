package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// AnalyticsEmbedding holds the schema definition for the AnalyticsEmbedding entity.
type AnalyticsEmbedding struct {
	ent.Schema
}

func (AnalyticsEmbedding) Mixin() []ent.Mixin {
	return []ent.Mixin{
		TimeMixin{},
	}
}

// Fields of the AnalyticsEmbedding.
func (AnalyticsEmbedding) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New).
			Immutable(),
		field.String("entity_type").
			NotEmpty(),
		field.UUID("entity_id", uuid.UUID{}),
		field.JSON("embedding", []float32{}).
			Optional(),
		field.Text("content").
			Optional(),
		field.JSON("metadata", map[string]interface{}{}).
			Optional(),
	}
}

// Edges of the AnalyticsEmbedding.
func (AnalyticsEmbedding) Edges() []ent.Edge {
	return nil
}
