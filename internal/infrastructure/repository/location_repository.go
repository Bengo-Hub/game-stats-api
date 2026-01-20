package repository

import (
	"context"
	"time"

	"github.com/bengobox/game-stats-api/ent"
	"github.com/bengobox/game-stats-api/ent/location"
	"github.com/google/uuid"
)

type locationRepository struct {
	client *ent.Client
}

// NewLocationRepository creates a new location repository.
func NewLocationRepository(client *ent.Client) *locationRepository {
	return &locationRepository{client: client}
}

func (r *locationRepository) Create(ctx context.Context, l *ent.Location) (*ent.Location, error) {
	return r.client.Location.Create().
		SetName(l.Name).
		SetSlug(l.Slug).
		SetNillableAddress(l.Address).
		SetNillableCity(l.City).
		SetNillableLatitude(l.Latitude).
		SetNillableLongitude(l.Longitude).
		SetNillableDescription(l.Description).
		SetCountryID(l.Edges.Country.ID).
		Save(ctx)
}

func (r *locationRepository) GetByID(ctx context.Context, id uuid.UUID) (*ent.Location, error) {
	return r.client.Location.Query().
		Where(location.ID(id)).
		WithCountry().
		Only(ctx)
}

func (r *locationRepository) GetBySlug(ctx context.Context, slug string) (*ent.Location, error) {
	return r.client.Location.Query().
		Where(location.SlugEQ(slug)).
		WithCountry().
		Only(ctx)
}

func (r *locationRepository) List(ctx context.Context) ([]*ent.Location, error) {
	return r.client.Location.Query().
		Where(location.DeletedAtIsNil()).
		WithCountry().
		All(ctx)
}

func (r *locationRepository) Update(ctx context.Context, l *ent.Location) (*ent.Location, error) {
	return r.client.Location.UpdateOneID(l.ID).
		SetName(l.Name).
		SetSlug(l.Slug).
		SetNillableAddress(l.Address).
		SetNillableCity(l.City).
		SetNillableLatitude(l.Latitude).
		SetNillableLongitude(l.Longitude).
		SetNillableDescription(l.Description).
		SetCountryID(l.Edges.Country.ID).
		SetUpdatedAt(time.Now()).
		Save(ctx)
}

func (r *locationRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.client.Location.UpdateOneID(id).
		SetDeletedAt(time.Now()).
		Exec(ctx)
}
