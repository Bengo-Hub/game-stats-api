package country

import (
	"context"

	"github.com/bengobox/game-stats-api/ent"
	"github.com/google/uuid"
)

type Repository interface {
	Create(ctx context.Context, country *ent.Country) (*ent.Country, error)
	GetByID(ctx context.Context, id uuid.UUID) (*ent.Country, error)
	GetBySlug(ctx context.Context, slug string) (*ent.Country, error)
	List(ctx context.Context) ([]*ent.Country, error)
	Update(ctx context.Context, country *ent.Country) (*ent.Country, error)
	Delete(ctx context.Context, id uuid.UUID) error
}
