package repository

import (
	"context"
	"time"

	"github.com/bengobox/game-stats-api/ent"
	"github.com/bengobox/game-stats-api/ent/discipline"
	"github.com/google/uuid"
)

type disciplineRepository struct {
	client *ent.Client
}

// NewDisciplineRepository creates a new discipline repository.
func NewDisciplineRepository(client *ent.Client) *disciplineRepository {
	return &disciplineRepository{client: client}
}

func (r *disciplineRepository) Create(ctx context.Context, d *ent.Discipline) (*ent.Discipline, error) {
	return r.client.Discipline.Create().
		SetName(d.Name).
		SetSlug(d.Slug).
		SetNillableDescription(d.Description).
		SetNillableRulesPdfURL(d.RulesPdfURL).
		SetCountryID(d.Edges.Country.ID).
		Save(ctx)
}

func (r *disciplineRepository) GetByID(ctx context.Context, id uuid.UUID) (*ent.Discipline, error) {
	return r.client.Discipline.Query().
		Where(discipline.ID(id)).
		WithCountry().
		Only(ctx)
}

func (r *disciplineRepository) GetBySlug(ctx context.Context, slug string) (*ent.Discipline, error) {
	return r.client.Discipline.Query().
		Where(discipline.SlugEQ(slug)).
		WithCountry().
		Only(ctx)
}

func (r *disciplineRepository) List(ctx context.Context) ([]*ent.Discipline, error) {
	return r.client.Discipline.Query().
		Where(discipline.DeletedAtIsNil()).
		WithCountry().
		All(ctx)
}

func (r *disciplineRepository) Update(ctx context.Context, d *ent.Discipline) (*ent.Discipline, error) {
	return r.client.Discipline.UpdateOneID(d.ID).
		SetName(d.Name).
		SetSlug(d.Slug).
		SetNillableDescription(d.Description).
		SetNillableRulesPdfURL(d.RulesPdfURL).
		SetCountryID(d.Edges.Country.ID).
		SetUpdatedAt(time.Now()).
		Save(ctx)
}

func (r *disciplineRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.client.Discipline.UpdateOneID(id).
		SetDeletedAt(time.Now()).
		Exec(ctx)
}
