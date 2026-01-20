package schema

import "entgo.io/ent"

// GameRound holds the schema definition for the GameRound entity.
type GameRound struct {
	ent.Schema
}

// Fields of the GameRound.
func (GameRound) Fields() []ent.Field {
	return nil
}

// Edges of the GameRound.
func (GameRound) Edges() []ent.Edge {
	return nil
}
