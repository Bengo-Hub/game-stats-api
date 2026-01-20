package player

import (
	"context"

	"github.com/bengobox/game-stats-api/ent"
	"github.com/google/uuid"
)

type Repository interface {
	Create(ctx context.Context, player *ent.Player) (*ent.Player, error)
	GetByID(ctx context.Context, id uuid.UUID) (*ent.Player, error)
	ListByTeam(ctx context.Context, teamID uuid.UUID) ([]*ent.Player, error)
	SearchByName(ctx context.Context, name string, limit int) ([]*ent.Player, error)
	Update(ctx context.Context, player *ent.Player) (*ent.Player, error)
	Delete(ctx context.Context, id uuid.UUID) error
}
