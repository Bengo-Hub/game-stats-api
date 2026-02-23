package eventparticipation

import (
	"context"

	"github.com/bengobox/game-stats-api/ent"
	"github.com/google/uuid"
)

type Repository interface {
	Create(ctx context.Context, ep *ent.EventParticipation) (*ent.EventParticipation, error)
	GetByID(ctx context.Context, id uuid.UUID) (*ent.EventParticipation, error)
	ListByEvent(ctx context.Context, eventID uuid.UUID) ([]*ent.EventParticipation, error)
	ListByPlayer(ctx context.Context, playerID uuid.UUID) ([]*ent.EventParticipation, error)
	Update(ctx context.Context, ep *ent.EventParticipation) (*ent.EventParticipation, error)
	Delete(ctx context.Context, id uuid.UUID) error
}
