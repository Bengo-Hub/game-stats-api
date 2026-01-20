package team

import (
	"context"

	"github.com/bengobox/game-stats-api/ent"
	"github.com/google/uuid"
)

type Repository interface {
	Create(ctx context.Context, team *ent.Team) (*ent.Team, error)
	GetByID(ctx context.Context, id uuid.UUID) (*ent.Team, error)
	ListByDivision(ctx context.Context, divisionID uuid.UUID) ([]*ent.Team, error)
	Update(ctx context.Context, team *ent.Team) (*ent.Team, error)
	Delete(ctx context.Context, id uuid.UUID) error
}
