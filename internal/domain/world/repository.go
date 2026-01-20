package world

import (
	"context"

	"github.com/bengobox/game-stats-api/ent"
	"github.com/google/uuid"
)

type Repository interface {
	Create(ctx context.Context, world *ent.World) (*ent.World, error)
	GetByID(ctx context.Context, id uuid.UUID) (*ent.World, error)
	GetBySlug(ctx context.Context, slug string) (*ent.World, error)
	List(ctx context.Context) ([]*ent.World, error)
	Update(ctx context.Context, world *ent.World) (*ent.World, error)
	Delete(ctx context.Context, id uuid.UUID) error
}
