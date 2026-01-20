package mvpnomination

import (
	"context"

	"github.com/bengobox/game-stats-api/ent"
	"github.com/google/uuid"
)

type Repository interface {
	Create(ctx context.Context, n *ent.MVP_Nomination) (*ent.MVP_Nomination, error)
	GetByID(ctx context.Context, id uuid.UUID) (*ent.MVP_Nomination, error)
	ListBySpiritScore(ctx context.Context, spiritScoreID uuid.UUID) ([]*ent.MVP_Nomination, error)
	Delete(ctx context.Context, id uuid.UUID) error
}
