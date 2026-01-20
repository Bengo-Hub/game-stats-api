package repository

import (
	"context"

	"github.com/bengobox/game-stats-api/ent"
	"github.com/bengobox/game-stats-api/ent/mvp_nomination"
	"github.com/bengobox/game-stats-api/ent/spiritscore"
	"github.com/google/uuid"
)

type mvpNominationRepository struct {
	client *ent.Client
}

// NewMVPNominationRepository creates a new MVP nomination repository.
func NewMVPNominationRepository(client *ent.Client) *mvpNominationRepository {
	return &mvpNominationRepository{client: client}
}

func (r *mvpNominationRepository) Create(ctx context.Context, n *ent.MVP_Nomination) (*ent.MVP_Nomination, error) {
	return r.client.MVP_Nomination.Create().
		SetCategory(n.Category).
		SetSpiritScoreID(n.Edges.SpiritScore.ID).
		SetPlayerID(n.Edges.Player.ID).
		Save(ctx)
}

func (r *mvpNominationRepository) GetByID(ctx context.Context, id uuid.UUID) (*ent.MVP_Nomination, error) {
	return r.client.MVP_Nomination.Query().
		Where(mvp_nomination.ID(id)).
		WithSpiritScore().
		WithPlayer().
		Only(ctx)
}

func (r *mvpNominationRepository) ListBySpiritScore(ctx context.Context, spiritScoreID uuid.UUID) ([]*ent.MVP_Nomination, error) {
	return r.client.MVP_Nomination.Query().
		Where(mvp_nomination.HasSpiritScoreWith(spiritscore.ID(spiritScoreID))).
		WithPlayer().
		All(ctx)
}

func (r *mvpNominationRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.client.MVP_Nomination.DeleteOneID(id).Exec(ctx)
}
