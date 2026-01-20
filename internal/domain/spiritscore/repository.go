package spiritscore

import (
	"context"

	"github.com/bengobox/game-stats-api/ent"
	"github.com/google/uuid"
)

type Repository interface {
	Create(ctx context.Context, s *ent.SpiritScore) (*ent.SpiritScore, error)
	GetByID(ctx context.Context, id uuid.UUID) (*ent.SpiritScore, error)
	ListByGame(ctx context.Context, gameID uuid.UUID) ([]*ent.SpiritScore, error)
	ListByTeam(ctx context.Context, teamID uuid.UUID) ([]*ent.SpiritScore, error)
	Update(ctx context.Context, s *ent.SpiritScore) (*ent.SpiritScore, error)
	Delete(ctx context.Context, id uuid.UUID) error
}
