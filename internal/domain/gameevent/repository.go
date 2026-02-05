package gameevent

import (
	"context"

	"github.com/bengobox/game-stats-api/ent"
	"github.com/google/uuid"
)

type Repository interface {
	Create(ctx context.Context, event *ent.GameEvent) (*ent.GameEvent, error)
	GetByID(ctx context.Context, id uuid.UUID) (*ent.GameEvent, error)
	ListByGame(ctx context.Context, gameID uuid.UUID) ([]*ent.GameEvent, error)
	Update(ctx context.Context, event *ent.GameEvent) (*ent.GameEvent, error)
	Delete(ctx context.Context, id uuid.UUID) error
}
