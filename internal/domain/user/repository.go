package user

import (
	"context"

	"github.com/bengobox/game-stats-api/ent"
	"github.com/google/uuid"
)

type Repository interface {
	Create(ctx context.Context, user *ent.User) (*ent.User, error)
	GetByID(ctx context.Context, id uuid.UUID) (*ent.User, error)
	GetByEmail(ctx context.Context, email string) (*ent.User, error)
	List(ctx context.Context) ([]*ent.User, error)
	Update(ctx context.Context, user *ent.User) (*ent.User, error)
	Delete(ctx context.Context, id uuid.UUID) error
}
