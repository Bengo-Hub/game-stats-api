package spiritnomination

import (
	"context"

	"github.com/bengobox/game-stats-api/ent"
	"github.com/google/uuid"
)

type Repository interface {
	Create(ctx context.Context, n *ent.SpiritNomination) (*ent.SpiritNomination, error)
	GetByID(ctx context.Context, id uuid.UUID) (*ent.SpiritNomination, error)
	ListBySpiritScore(ctx context.Context, spiritScoreID uuid.UUID) ([]*ent.SpiritNomination, error)
	CountByTeam(ctx context.Context, teamID uuid.UUID) (int, error)
	Delete(ctx context.Context, id uuid.UUID) error
}
