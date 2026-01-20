package schema

import "entgo.io/ent"

// SpiritScore holds the schema definition for the SpiritScore entity.
type SpiritScore struct {
	ent.Schema
}

// Fields of the SpiritScore.
func (SpiritScore) Fields() []ent.Field {
	return nil
}

// Edges of the SpiritScore.
func (SpiritScore) Edges() []ent.Edge {
	return nil
}
