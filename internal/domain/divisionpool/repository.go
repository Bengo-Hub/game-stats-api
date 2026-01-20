package divisionpool

import (
	"context"

	"github.com/bengobox/game-stats-api/ent"
	"github.com/google/uuid"
)

type Repository interface {
	Create(ctx context.Context, pool *ent.DivisionPool) (*ent.DivisionPool, error)
	GetByID(ctx context.Context, id uuid.UUID) (*ent.DivisionPool, error)
	ListByEvent(ctx context.Context, eventID uuid.UUID) ([]*ent.DivisionPool, error)
	Update(ctx context.Context, pool *ent.DivisionPool) (*ent.DivisionPool, error)
	Delete(ctx context.Context, id uuid.UUID) error
}
