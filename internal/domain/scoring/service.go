package scoring

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

type ScoreService struct {
	repo Repository
}

func NewScoreService(repo Repository) *ScoreService {
	return &ScoreService{repo: repo}
}

// RecalculateTotals calculates the home and away scores for a game based on player scores.
func (s *ScoreService) RecalculateTotals(ctx context.Context, gameID uuid.UUID, homeTeamID, awayTeamID uuid.UUID) (int, int, error) {
	scores, err := s.repo.ListByGame(ctx, gameID)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to list scores: %w", err)
	}

	homeScore := 0
	awayScore := 0

	for _, score := range scores {
		if score.Edges.Player != nil {
			for _, t := range score.Edges.Player.Edges.Teams {
				if t.ID == homeTeamID {
					homeScore += score.Goals
					break
				} else if t.ID == awayTeamID {
					awayScore += score.Goals
					break
				}
			}
		}
	}

	return homeScore, awayScore, nil
}
