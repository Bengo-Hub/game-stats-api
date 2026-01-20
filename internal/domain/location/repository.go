package location

import (
	"context"

	"github.com/bengobox/game-stats-api/ent"
	"github.com/google/uuid"
)

type Repository interface {
	Create(ctx context.Context, location *ent.Location) (*ent.Location, error)
	GetByID(ctx context.Context, id uuid.UUID) (*ent.Location, error)
	GetBySlug(ctx context.Context, slug string) (*ent.Location, error)
	List(ctx context.Context) ([]*ent.Location, error)
	Update(ctx context.Context, location *ent.Location) (*ent.Location, error)
	Delete(ctx context.Context, id uuid.UUID) error
}
