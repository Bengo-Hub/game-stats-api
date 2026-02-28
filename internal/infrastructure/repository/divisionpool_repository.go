package repository

import (
	"context"
	"time"

	"github.com/bengobox/game-stats-api/ent"
	"github.com/bengobox/game-stats-api/ent/divisionpool"
	"github.com/bengobox/game-stats-api/ent/event"
	"github.com/google/uuid"
)

type divisionPoolRepository struct {
	client *ent.Client
}

// NewDivisionPoolRepository creates a new division pool repository.
func NewDivisionPoolRepository(client *ent.Client) *divisionPoolRepository {
	return &divisionPoolRepository{client: client}
}

func (r *divisionPoolRepository) Create(ctx context.Context, p *ent.DivisionPool) (*ent.DivisionPool, error) {
	return r.client.DivisionPool.Create().
		SetName(p.Name).
		SetDivisionType(p.DivisionType).
		SetNillableMaxTeams(p.MaxTeams).
		SetRankingCriteria(p.RankingCriteria).
		SetNillableDescription(p.Description).
		AddEventIDs(p.Edges.Events[0].ID).
		Save(ctx)
}

func (r *divisionPoolRepository) GetByID(ctx context.Context, id uuid.UUID) (*ent.DivisionPool, error) {
	return r.client.DivisionPool.Query().
		Where(divisionpool.ID(id)).
		WithEvents().
		WithTeams().
		WithTargetRound().
		Only(ctx)
}

func (r *divisionPoolRepository) ListByEvent(ctx context.Context, eventID uuid.UUID) ([]*ent.DivisionPool, error) {
	return r.client.DivisionPool.Query().
		Where(divisionpool.HasEventsWith(event.ID(eventID))).
		Where(divisionpool.DeletedAtIsNil()).
		WithTeams().
		All(ctx)
}

func (r *divisionPoolRepository) Update(ctx context.Context, p *ent.DivisionPool) (*ent.DivisionPool, error) {
	return r.client.DivisionPool.UpdateOneID(p.ID).
		SetName(p.Name).
		SetDivisionType(p.DivisionType).
		SetNillableMaxTeams(p.MaxTeams).
		SetRankingCriteria(p.RankingCriteria).
		SetNillableDescription(p.Description).
		AddEventIDs(p.Edges.Events[0].ID).
		SetUpdatedAt(time.Now()).
		Save(ctx)
}

func (r *divisionPoolRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.client.DivisionPool.UpdateOneID(id).
		SetDeletedAt(time.Now()).
		Exec(ctx)
}

func (r *divisionPoolRepository) ListAll(ctx context.Context) ([]*ent.DivisionPool, error) {
	return r.client.DivisionPool.Query().
		Where(divisionpool.DeletedAtIsNil()).
		All(ctx)
}
