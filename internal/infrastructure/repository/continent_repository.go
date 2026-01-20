package repository

import (
	"context"
	"time"

	"github.com/bengobox/game-stats-api/ent"
	"github.com/bengobox/game-stats-api/ent/continent"
	"github.com/google/uuid"
)

type continentRepository struct {
	client *ent.Client
}

// NewContinentRepository creates a new continent repository.
func NewContinentRepository(client *ent.Client) *continentRepository {
	return &continentRepository{client: client}
}

func (r *continentRepository) Create(ctx context.Context, c *ent.Continent) (*ent.Continent, error) {
	return r.client.Continent.Create().
		SetName(c.Name).
		SetSlug(c.Slug).
		SetNillableDescription(c.Description).
		SetWorldID(c.Edges.World.ID).
		Save(ctx)
}

func (r *continentRepository) GetByID(ctx context.Context, id uuid.UUID) (*ent.Continent, error) {
	return r.client.Continent.Query().
		Where(continent.ID(id)).
		WithWorld().
		Only(ctx)
}

func (r *continentRepository) GetBySlug(ctx context.Context, slug string) (*ent.Continent, error) {
	return r.client.Continent.Query().
		Where(continent.SlugEQ(slug)).
		WithWorld().
		Only(ctx)
}

func (r *continentRepository) List(ctx context.Context) ([]*ent.Continent, error) {
	return r.client.Continent.Query().
		Where(continent.DeletedAtIsNil()).
		WithWorld().
		All(ctx)
}

func (r *continentRepository) Update(ctx context.Context, c *ent.Continent) (*ent.Continent, error) {
	return r.client.Continent.UpdateOneID(c.ID).
		SetName(c.Name).
		SetSlug(c.Slug).
		SetNillableDescription(c.Description).
		SetWorldID(c.Edges.World.ID).
		SetUpdatedAt(time.Now()).
		Save(ctx)
}

func (r *continentRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.client.Continent.UpdateOneID(id).
		SetDeletedAt(time.Now()).
		Exec(ctx)
}
