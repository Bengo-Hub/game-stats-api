package repository

import (
	"context"
	"time"

	"github.com/bengobox/game-stats-api/ent"
	"github.com/bengobox/game-stats-api/ent/player"
	"github.com/bengobox/game-stats-api/ent/team"
	"github.com/google/uuid"
)

type playerRepository struct {
	client *ent.Client
}

// NewPlayerRepository creates a new player repository.
func NewPlayerRepository(client *ent.Client) *playerRepository {
	return &playerRepository{client: client}
}

func (r *playerRepository) Create(ctx context.Context, p *ent.Player) (*ent.Player, error) {
	return r.client.Player.Create().
		SetName(p.Name).
		SetNillableEmail(p.Email).
		SetGender(p.Gender).
		SetNillableDateOfBirth(p.DateOfBirth).
		SetNillableJerseyNumber(p.JerseyNumber).
		SetNillableProfileImageURL(p.ProfileImageURL).
		SetMetadata(p.Metadata).
		SetTeamID(p.Edges.Team.ID).
		Save(ctx)
}

func (r *playerRepository) GetByID(ctx context.Context, id uuid.UUID) (*ent.Player, error) {
	return r.client.Player.Query().
		Where(player.ID(id)).
		WithTeam().
		Only(ctx)
}

func (r *playerRepository) ListByTeam(ctx context.Context, teamID uuid.UUID) ([]*ent.Player, error) {
	return r.client.Player.Query().
		Where(player.HasTeamWith(team.ID(teamID))).
		Where(player.DeletedAtIsNil()).
		All(ctx)
}

func (r *playerRepository) SearchByName(ctx context.Context, name string, limit int) ([]*ent.Player, error) {
	// Simple ILIKE search for now.
	// In production with pg_trgm, we could use % similarity or word similarity.
	return r.client.Player.Query().
		Where(player.NameContainsFold(name)).
		Where(player.DeletedAtIsNil()).
		Limit(limit).
		All(ctx)
}

func (r *playerRepository) Update(ctx context.Context, p *ent.Player) (*ent.Player, error) {
	return r.client.Player.UpdateOneID(p.ID).
		SetName(p.Name).
		SetNillableEmail(p.Email).
		SetGender(p.Gender).
		SetNillableDateOfBirth(p.DateOfBirth).
		SetNillableJerseyNumber(p.JerseyNumber).
		SetNillableProfileImageURL(p.ProfileImageURL).
		SetMetadata(p.Metadata).
		SetTeamID(p.Edges.Team.ID).
		SetUpdatedAt(time.Now()).
		Save(ctx)
}

func (r *playerRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.client.Player.UpdateOneID(id).
		SetDeletedAt(time.Now()).
		Exec(ctx)
}
