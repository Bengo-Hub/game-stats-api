package game

import (
	"context"
	"time"

	"github.com/bengobox/game-stats-api/ent"
	"github.com/google/uuid"
)

type Repository interface {
	Create(ctx context.Context, game *ent.Game) (*ent.Game, error)
	GetByID(ctx context.Context, id uuid.UUID) (*ent.Game, error)
	GetByIDWithRelations(ctx context.Context, id uuid.UUID) (*ent.Game, error)
	ListByDivision(ctx context.Context, divisionID uuid.UUID) ([]*ent.Game, error)
	ListByRound(ctx context.Context, roundID uuid.UUID) ([]*ent.Game, error)
	ListByStatus(ctx context.Context, status string) ([]*ent.Game, error)
	ListByField(ctx context.Context, fieldID uuid.UUID) ([]*ent.Game, error)
	ListByDateRange(ctx context.Context, start, end time.Time) ([]*ent.Game, error)
	List(ctx context.Context, limit, offset int) ([]*ent.Game, error)
	Update(ctx context.Context, game *ent.Game) (*ent.Game, error)
	UpdateWithVersion(ctx context.Context, id uuid.UUID, version int, updateFn func(*ent.GameUpdateOne) *ent.GameUpdateOne) (*ent.Game, error)
	Delete(ctx context.Context, id uuid.UUID) error
	CheckFieldConflict(ctx context.Context, fieldID uuid.UUID, scheduledTime time.Time, duration int) (bool, error)
}
