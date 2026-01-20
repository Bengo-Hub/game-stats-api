package repository

import (
	"context"
	"time"

	"github.com/bengobox/game-stats-api/ent"
	"github.com/bengobox/game-stats-api/ent/country"
	"github.com/google/uuid"
)

type countryRepository struct {
	client *ent.Client
}

// NewCountryRepository creates a new country repository.
func NewCountryRepository(client *ent.Client) *countryRepository {
	return &countryRepository{client: client}
}

func (r *countryRepository) Create(ctx context.Context, c *ent.Country) (*ent.Country, error) {
	return r.client.Country.Create().
		SetName(c.Name).
		SetSlug(c.Slug).
		SetCode(c.Code).
		SetNillableDescription(c.Description).
		SetContinentID(c.Edges.Continent.ID).
		Save(ctx)
}

func (r *countryRepository) GetByID(ctx context.Context, id uuid.UUID) (*ent.Country, error) {
	return r.client.Country.Query().
		Where(country.ID(id)).
		WithContinent().
		Only(ctx)
}

func (r *countryRepository) GetBySlug(ctx context.Context, slug string) (*ent.Country, error) {
	return r.client.Country.Query().
		Where(country.SlugEQ(slug)).
		WithContinent().
		Only(ctx)
}

func (r *countryRepository) List(ctx context.Context) ([]*ent.Country, error) {
	return r.client.Country.Query().
		Where(country.DeletedAtIsNil()).
		WithContinent().
		All(ctx)
}

func (r *countryRepository) Update(ctx context.Context, c *ent.Country) (*ent.Country, error) {
	return r.client.Country.UpdateOneID(c.ID).
		SetName(c.Name).
		SetSlug(c.Slug).
		SetCode(c.Code).
		SetNillableDescription(c.Description).
		SetContinentID(c.Edges.Continent.ID).
		SetUpdatedAt(time.Now()).
		Save(ctx)
}

func (r *countryRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.client.Country.UpdateOneID(id).
		SetDeletedAt(time.Now()).
		Exec(ctx)
}
