package event

import (
	"context"

	"github.com/bengobox/game-stats-api/ent"
	"github.com/google/uuid"
)

type Repository interface {
	Create(ctx context.Context, event *ent.Event) (*ent.Event, error)
	GetByID(ctx context.Context, id uuid.UUID) (*ent.Event, error)
	GetBySlug(ctx context.Context, slug string) (*ent.Event, error)
	List(ctx context.Context, year int) ([]*ent.Event, error)
	Update(ctx context.Context, event *ent.Event) (*ent.Event, error)
	Delete(ctx context.Context, id uuid.UUID) error
}
