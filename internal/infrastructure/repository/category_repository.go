package repository

import (
	"context"
	"time"

	"github.com/bengobox/game-stats-api/ent"
	"github.com/bengobox/game-stats-api/ent/category"
	"github.com/google/uuid"
)

type categoryRepository struct {
	client *ent.Client
}

// NewCategoryRepository creates a new category repository.
func NewCategoryRepository(client *ent.Client) *categoryRepository {
	return &categoryRepository{client: client}
}

func (r *categoryRepository) Create(ctx context.Context, c *ent.Category) (*ent.Category, error) {
	return r.client.Category.Create().
		SetName(c.Name).
		SetSlug(c.Slug).
		SetNillableDescription(c.Description).
		Save(ctx)
}

func (r *categoryRepository) GetByID(ctx context.Context, id uuid.UUID) (*ent.Category, error) {
	return r.client.Category.Query().
		Where(category.ID(id)).
		Only(ctx)
}

func (r *categoryRepository) GetBySlug(ctx context.Context, slug string) (*ent.Category, error) {
	return r.client.Category.Query().
		Where(category.SlugEQ(slug)).
		Only(ctx)
}

func (r *categoryRepository) List(ctx context.Context) ([]*ent.Category, error) {
	return r.client.Category.Query().
		Where(category.DeletedAtIsNil()).
		All(ctx)
}

func (r *categoryRepository) Update(ctx context.Context, c *ent.Category) (*ent.Category, error) {
	return r.client.Category.UpdateOneID(c.ID).
		SetName(c.Name).
		SetSlug(c.Slug).
		SetNillableDescription(c.Description).
		SetUpdatedAt(time.Now()).
		Save(ctx)
}

func (r *categoryRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.client.Category.UpdateOneID(id).
		SetDeletedAt(time.Now()).
		Exec(ctx)
}
