package repository

import (
	"context"
	"time"

	"github.com/bengobox/game-stats-api/ent"
	"github.com/bengobox/game-stats-api/ent/event"
	"github.com/google/uuid"
)

type eventRepository struct {
	client *ent.Client
}

// NewEventRepository creates a new event repository.
func NewEventRepository(client *ent.Client) *eventRepository {
	return &eventRepository{client: client}
}

func (r *eventRepository) Create(ctx context.Context, e *ent.Event) (*ent.Event, error) {
	return r.client.Event.Create().
		SetName(e.Name).
		SetSlug(e.Slug).
		SetYear(e.Year).
		SetStartDate(e.StartDate).
		SetEndDate(e.EndDate).
		SetStatus(e.Status).
		SetNillableDescription(e.Description).
		SetSettings(e.Settings).
		SetDisciplineID(e.Edges.Discipline.ID).
		SetLocationID(e.Edges.Location.ID).
		Save(ctx)
}

func (r *eventRepository) GetByID(ctx context.Context, id uuid.UUID) (*ent.Event, error) {
	return r.client.Event.Query().
		Where(event.ID(id)).
		WithDiscipline().
		WithLocation().
		WithDivisionPools().
		Only(ctx)
}

func (r *eventRepository) GetBySlug(ctx context.Context, slug string) (*ent.Event, error) {
	return r.client.Event.Query().
		Where(event.SlugEQ(slug)).
		WithDiscipline().
		WithLocation().
		WithDivisionPools().
		Only(ctx)
}

func (r *eventRepository) List(ctx context.Context, year int) ([]*ent.Event, error) {
	query := r.client.Event.Query().
		Where(event.DeletedAtIsNil())

	if year > 0 {
		query = query.Where(event.YearEQ(year))
	}

	return query.
		WithDiscipline().
		WithLocation().
		All(ctx)
}

func (r *eventRepository) Update(ctx context.Context, e *ent.Event) (*ent.Event, error) {
	return r.client.Event.UpdateOneID(e.ID).
		SetName(e.Name).
		SetSlug(e.Slug).
		SetYear(e.Year).
		SetStartDate(e.StartDate).
		SetEndDate(e.EndDate).
		SetStatus(e.Status).
		SetNillableDescription(e.Description).
		SetSettings(e.Settings).
		SetDisciplineID(e.Edges.Discipline.ID).
		SetLocationID(e.Edges.Location.ID).
		SetUpdatedAt(time.Now()).
		Save(ctx)
}

func (r *eventRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.client.Event.UpdateOneID(id).
		SetDeletedAt(time.Now()).
		Exec(ctx)
}
