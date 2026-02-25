package category

import (
	"context"

	"github.com/bengobox/game-stats-api/ent"
	"github.com/google/uuid"
)

// Repository defines persistence operations for categories.
// Implemented using ent at the moment.

type Repository interface {
	Create(ctx context.Context, c *ent.Category) (*ent.Category, error)
	GetByID(ctx context.Context, id uuid.UUID) (*ent.Category, error)
	GetBySlug(ctx context.Context, slug string) (*ent.Category, error)
	List(ctx context.Context) ([]*ent.Category, error)
	Update(ctx context.Context, c *ent.Category) (*ent.Category, error)
	Delete(ctx context.Context, id uuid.UUID) error
}
