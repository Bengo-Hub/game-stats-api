package seeds

import (
	"context"
	"fmt"

	"github.com/bengobox/game-stats-api/ent"
	"github.com/google/uuid"
)

// DefaultRound represents a template for a round
type DefaultRound struct {
	Name        string
	RoundType   string
	RoundNumber int
}

// GetDefaultRounds returns a list of common game rounds
func GetDefaultRounds() []DefaultRound {
	return []DefaultRound{
		{Name: "Pool A", RoundType: "pool", RoundNumber: 1},
		{Name: "Pool B", RoundType: "pool", RoundNumber: 1},
		{Name: "Pool C", RoundType: "pool", RoundNumber: 1},
		{Name: "Pool D", RoundType: "pool", RoundNumber: 1},
		{Name: "Crossover", RoundType: "crossover", RoundNumber: 2},
		{Name: "Pre-Quarters", RoundType: "bracket", RoundNumber: 3},
		{Name: "Quarter-Finals", RoundType: "bracket", RoundNumber: 4},
		{Name: "Semi-Finals", RoundType: "semifinal", RoundNumber: 5},
		{Name: "3rd Place Playoff", RoundType: "bracket", RoundNumber: 6},
		{Name: "Finals", RoundType: "final", RoundNumber: 6},
	}
}

// SeedGameRounds seeds common game rounds for an event
func SeedGameRounds(ctx context.Context, client *ent.Client, eventID uuid.UUID) error {
	rounds := GetDefaultRounds()

	for _, r := range rounds {
		_, err := client.GameRound.Create().
			SetName(r.Name).
			SetRoundType(r.RoundType).
			SetRoundNumber(r.RoundNumber).
			SetEventID(eventID).
			Save(ctx)
		if err != nil {
			return fmt.Errorf("failed to seed round %s: %w", r.Name, err)
		}
	}

	return nil
}
