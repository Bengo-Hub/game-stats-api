package repository

import (
	"context"
	"time"

	"github.com/bengobox/game-stats-api/ent"
	"github.com/bengobox/game-stats-api/ent/divisionpool"
	"github.com/bengobox/game-stats-api/ent/team"
	"github.com/google/uuid"
)

type teamRepository struct {
	client *ent.Client
}

// NewTeamRepository creates a new team repository.
func NewTeamRepository(client *ent.Client) *teamRepository {
	return &teamRepository{client: client}
}

func (r *teamRepository) Create(ctx context.Context, t *ent.Team) (*ent.Team, error) {
	builder := r.client.Team.Create().
		SetName(t.Name).
		SetNillableLogoURL(t.LogoURL).
		SetMetadata(t.Metadata)

	if len(t.Edges.DivisionPools) > 0 {
		ids := make([]uuid.UUID, len(t.Edges.DivisionPools))
		for i, dp := range t.Edges.DivisionPools {
			ids[i] = dp.ID
		}
		builder.AddDivisionPoolIDs(ids...)
	}

	if t.Edges.HomeLocation != nil {
		builder.SetHomeLocationID(t.Edges.HomeLocation.ID)
	}

	return builder.Save(ctx)
}

func (r *teamRepository) GetByID(ctx context.Context, id uuid.UUID) (*ent.Team, error) {
	return r.client.Team.Query().
		Where(team.ID(id)).
		WithDivisionPools(func(query *ent.DivisionPoolQuery) {
			query.WithEvents()
		}).
		WithHomeLocation().
		WithPlayers().
		Only(ctx)
}

func (r *teamRepository) ListByDivision(ctx context.Context, divisionID uuid.UUID) ([]*ent.Team, error) {
	return r.client.Team.Query().
		Where(team.HasDivisionPoolsWith(divisionpool.ID(divisionID))).
		Where(team.DeletedAtIsNil()).
		WithDivisionPools(func(query *ent.DivisionPoolQuery) {
			query.WithEvents()
		}).
		All(ctx)
}

func (r *teamRepository) Update(ctx context.Context, t *ent.Team) (*ent.Team, error) {
	updater := r.client.Team.UpdateOneID(t.ID).
		SetName(t.Name).
		SetNillableLogoURL(t.LogoURL).
		SetMetadata(t.Metadata).
		SetUpdatedAt(time.Now())

	if len(t.Edges.DivisionPools) > 0 {
		ids := make([]uuid.UUID, len(t.Edges.DivisionPools))
		for i, dp := range t.Edges.DivisionPools {
			ids[i] = dp.ID
		}
		updater.AddDivisionPoolIDs(ids...)
	}

	return updater.Save(ctx)
}

func (r *teamRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.client.Team.UpdateOneID(id).
		SetDeletedAt(time.Now()).
		Exec(ctx)
}
