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
	query := r.client.Team.Create().
		SetName(t.Name).
		SetNillableInitialSeed(t.InitialSeed).
		SetNillableFinalPlacement(t.FinalPlacement).
		SetNillableLogoURL(t.LogoURL).
		SetMetadata(t.Metadata).
		SetDivisionPoolID(t.Edges.DivisionPool.ID)

	if t.Edges.HomeLocation != nil {
		query.SetHomeLocationID(t.Edges.HomeLocation.ID)
	}

	return query.Save(ctx)
}

func (r *teamRepository) GetByID(ctx context.Context, id uuid.UUID) (*ent.Team, error) {
	return r.client.Team.Query().
		Where(team.ID(id)).
		WithDivisionPool().
		WithHomeLocation().
		WithPlayers().
		Only(ctx)
}

func (r *teamRepository) ListByDivision(ctx context.Context, divisionID uuid.UUID) ([]*ent.Team, error) {
	return r.client.Team.Query().
		Where(team.HasDivisionPoolWith(divisionpool.ID(divisionID))).
		Where(team.DeletedAtIsNil()).
		WithDivisionPool().
		All(ctx)
}

func (r *teamRepository) Update(ctx context.Context, t *ent.Team) (*ent.Team, error) {
	query := r.client.Team.UpdateOneID(t.ID).
		SetName(t.Name).
		SetNillableInitialSeed(t.InitialSeed).
		SetNillableFinalPlacement(t.FinalPlacement).
		SetNillableLogoURL(t.LogoURL).
		SetMetadata(t.Metadata).
		SetDivisionPoolID(t.Edges.DivisionPool.ID).
		SetUpdatedAt(time.Now())

	if t.Edges.HomeLocation != nil {
		query.SetHomeLocationID(t.Edges.HomeLocation.ID)
	} else {
		query.ClearHomeLocation()
	}

	return query.Save(ctx)
}

func (r *teamRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.client.Team.UpdateOneID(id).
		SetDeletedAt(time.Now()).
		Exec(ctx)
}
