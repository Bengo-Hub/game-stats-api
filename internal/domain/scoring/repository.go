package scoring

import (
	"context"

	"github.com/bengobox/game-stats-api/ent"
	"github.com/google/uuid"
)

type Repository interface {
	Create(ctx context.Context, s *ent.Scoring) (*ent.Scoring, error)
	GetByID(ctx context.Context, id uuid.UUID) (*ent.Scoring, error)
	ListByGame(ctx context.Context, gameID uuid.UUID) ([]*ent.Scoring, error)
	ListByPlayer(ctx context.Context, playerID uuid.UUID) ([]*ent.Scoring, error)
	Update(ctx context.Context, s *ent.Scoring) (*ent.Scoring, error)
	Delete(ctx context.Context, id uuid.UUID) error
}
