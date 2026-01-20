package repository

import (
	"context"

	"github.com/bengobox/game-stats-api/ent"
	"github.com/bengobox/game-stats-api/ent/spiritnomination"
	"github.com/bengobox/game-stats-api/ent/spiritscore"
	"github.com/google/uuid"
)

type spiritNominationRepository struct {
	client *ent.Client
}

// NewSpiritNominationRepository creates a new spirit nomination repository.
func NewSpiritNominationRepository(client *ent.Client) *spiritNominationRepository {
	return &spiritNominationRepository{client: client}
}

func (r *spiritNominationRepository) Create(ctx context.Context, n *ent.SpiritNomination) (*ent.SpiritNomination, error) {
	return r.client.SpiritNomination.Create().
		SetCategory(n.Category).
		SetSpiritScoreID(n.Edges.SpiritScore.ID).
		SetPlayerID(n.Edges.Player.ID).
		Save(ctx)
}

func (r *spiritNominationRepository) GetByID(ctx context.Context, id uuid.UUID) (*ent.SpiritNomination, error) {
	return r.client.SpiritNomination.Query().
		Where(spiritnomination.ID(id)).
		WithSpiritScore().
		WithPlayer().
		Only(ctx)
}

func (r *spiritNominationRepository) ListBySpiritScore(ctx context.Context, spiritScoreID uuid.UUID) ([]*ent.SpiritNomination, error) {
	return r.client.SpiritNomination.Query().
		Where(spiritnomination.HasSpiritScoreWith(spiritscore.ID(spiritScoreID))).
		WithPlayer().
		All(ctx)
}

func (r *spiritNominationRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.client.SpiritNomination.DeleteOneID(id).Exec(ctx)
}
