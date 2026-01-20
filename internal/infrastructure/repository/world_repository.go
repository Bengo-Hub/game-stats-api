package repository

import (
	"context"
	"time"

	"github.com/bengobox/game-stats-api/ent"
	"github.com/bengobox/game-stats-api/ent/world"
	"github.com/google/uuid"
)

type worldRepository struct {
	client *ent.Client
}

// NewWorldRepository creates a new world repository.
func NewWorldRepository(client *ent.Client) *worldRepository {
	return &worldRepository{client: client}
}

func (r *worldRepository) Create(ctx context.Context, w *ent.World) (*ent.World, error) {
	return r.client.World.Create().
		SetName(w.Name).
		SetSlug(w.Slug).
		SetNillableDescription(w.Description).
		Save(ctx)
}

func (r *worldRepository) GetByID(ctx context.Context, id uuid.UUID) (*ent.World, error) {
	return r.client.World.Get(ctx, id)
}

func (r *worldRepository) GetBySlug(ctx context.Context, slug string) (*ent.World, error) {
	return r.client.World.Query().
		Where(world.SlugEQ(slug)).
		Only(ctx)
}

func (r *worldRepository) List(ctx context.Context) ([]*ent.World, error) {
	return r.client.World.Query().
		Where(world.DeletedAtIsNil()).
		All(ctx)
}

func (r *worldRepository) Update(ctx context.Context, w *ent.World) (*ent.World, error) {
	return r.client.World.UpdateOneID(w.ID).
		SetName(w.Name).
		SetSlug(w.Slug).
		SetNillableDescription(w.Description).
		SetUpdatedAt(time.Now()).
		Save(ctx)
}

func (r *worldRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.client.World.UpdateOneID(id).
		SetDeletedAt(time.Now()).
		Exec(ctx)
}
