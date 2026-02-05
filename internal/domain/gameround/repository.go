package gameround

import (
	"context"

	"github.com/bengobox/game-stats-api/ent"
	"github.com/google/uuid"
)

type Repository interface {
	Create(ctx context.Context, round *ent.GameRound) (*ent.GameRound, error)
	GetByID(ctx context.Context, id uuid.UUID) (*ent.GameRound, error)
	GetByIDWithGames(ctx context.Context, id uuid.UUID) (*ent.GameRound, error)
	ListByEvent(ctx context.Context, eventID uuid.UUID) ([]*ent.GameRound, error)
	Update(ctx context.Context, round *ent.GameRound) (*ent.GameRound, error)
	Delete(ctx context.Context, id uuid.UUID) error
}
