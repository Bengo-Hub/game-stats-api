package schema

import "entgo.io/ent"

// Scoring holds the schema definition for the Scoring entity.
type Scoring struct {
	ent.Schema
}

// Fields of the Scoring.
func (Scoring) Fields() []ent.Field {
	return nil
}

// Edges of the Scoring.
func (Scoring) Edges() []ent.Edge {
	return nil
}
