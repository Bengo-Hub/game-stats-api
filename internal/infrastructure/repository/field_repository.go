package repository

import (
	"context"
	"time"

	"github.com/bengobox/game-stats-api/ent"
	entfield "github.com/bengobox/game-stats-api/ent/field"
	"github.com/google/uuid"
)

type fieldRepository struct {
	client *ent.Client
}

// NewFieldRepository creates a new field repository.
func NewFieldRepository(client *ent.Client) *fieldRepository {
	return &fieldRepository{client: client}
}

func (r *fieldRepository) Create(ctx context.Context, f *ent.Field) (*ent.Field, error) {
	return r.client.Field.Create().
		SetName(f.Name).
		SetNillableCapacity(f.Capacity).
		SetNillableSurfaceType(f.SurfaceType).
		SetMetadata(f.Metadata).
		SetLocationID(f.Edges.Location.ID).
		Save(ctx)
}

func (r *fieldRepository) GetByID(ctx context.Context, id uuid.UUID) (*ent.Field, error) {
	return r.client.Field.Query().
		Where(entfield.ID(id)).
		WithLocation().
		Only(ctx)
}

func (r *fieldRepository) List(ctx context.Context) ([]*ent.Field, error) {
	return r.client.Field.Query().
		Where(entfield.DeletedAtIsNil()).
		WithLocation().
		All(ctx)
}

func (r *fieldRepository) Update(ctx context.Context, f *ent.Field) (*ent.Field, error) {
	return r.client.Field.UpdateOneID(f.ID).
		SetName(f.Name).
		SetNillableCapacity(f.Capacity).
		SetNillableSurfaceType(f.SurfaceType).
		SetMetadata(f.Metadata).
		SetLocationID(f.Edges.Location.ID).
		SetUpdatedAt(time.Now()).
		Save(ctx)
}

func (r *fieldRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.client.Field.UpdateOneID(id).
		SetDeletedAt(time.Now()).
		Exec(ctx)
}
