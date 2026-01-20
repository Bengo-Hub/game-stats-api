package field

import (
	"context"

	"github.com/bengobox/game-stats-api/ent"
	"github.com/google/uuid"
)

type Repository interface {
	Create(ctx context.Context, field *ent.Field) (*ent.Field, error)
	GetByID(ctx context.Context, id uuid.UUID) (*ent.Field, error)
	List(ctx context.Context) ([]*ent.Field, error)
	Update(ctx context.Context, field *ent.Field) (*ent.Field, error)
	Delete(ctx context.Context, id uuid.UUID) error
}
