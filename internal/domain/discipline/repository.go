package discipline

import (
	"context"

	"github.com/bengobox/game-stats-api/ent"
	"github.com/google/uuid"
)

type Repository interface {
	Create(ctx context.Context, discipline *ent.Discipline) (*ent.Discipline, error)
	GetByID(ctx context.Context, id uuid.UUID) (*ent.Discipline, error)
	GetBySlug(ctx context.Context, slug string) (*ent.Discipline, error)
	List(ctx context.Context) ([]*ent.Discipline, error)
	Update(ctx context.Context, discipline *ent.Discipline) (*ent.Discipline, error)
	Delete(ctx context.Context, id uuid.UUID) error
}
