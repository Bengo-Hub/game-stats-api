package eventreconciliation

import (
	"context"

	"github.com/bengobox/game-stats-api/ent"
	"github.com/google/uuid"
)

type Repository interface {
	Create(ctx context.Context, r *ent.EventReconciliation) (*ent.EventReconciliation, error)
	GetByID(ctx context.Context, id uuid.UUID) (*ent.EventReconciliation, error)
	ListByEvent(ctx context.Context, eventID uuid.UUID) ([]*ent.EventReconciliation, error)
	Update(ctx context.Context, r *ent.EventReconciliation) (*ent.EventReconciliation, error)
	Delete(ctx context.Context, id uuid.UUID) error
}
