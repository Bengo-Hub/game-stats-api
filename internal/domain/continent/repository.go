package continent

import (
	"context"

	"github.com/bengobox/game-stats-api/ent"
	"github.com/google/uuid"
)

type Repository interface {
	Create(ctx context.Context, continent *ent.Continent) (*ent.Continent, error)
	GetByID(ctx context.Context, id uuid.UUID) (*ent.Continent, error)
	GetBySlug(ctx context.Context, slug string) (*ent.Continent, error)
	List(ctx context.Context) ([]*ent.Continent, error)
	Update(ctx context.Context, continent *ent.Continent) (*ent.Continent, error)
	Delete(ctx context.Context, id uuid.UUID) error
}
