package repository

import (
	"context"
	"time"

	"github.com/bengobox/game-stats-api/ent"
	"github.com/bengobox/game-stats-api/ent/game"
	"github.com/bengobox/game-stats-api/ent/spiritscore"
	"github.com/bengobox/game-stats-api/ent/team"
	"github.com/google/uuid"
)

type spiritScoreRepository struct {
	client *ent.Client
}

// NewSpiritScoreRepository creates a new spirit score repository.
func NewSpiritScoreRepository(client *ent.Client) *spiritScoreRepository {
	return &spiritScoreRepository{client: client}
}

func (r *spiritScoreRepository) Create(ctx context.Context, s *ent.SpiritScore) (*ent.SpiritScore, error) {
	return r.client.SpiritScore.Create().
		SetRulesKnowledge(s.RulesKnowledge).
		SetFoulsBodyContact(s.FoulsBodyContact).
		SetFairMindedness(s.FairMindedness).
		SetAttitude(s.Attitude).
		SetCommunication(s.Communication).
		SetNillableComments(s.Comments).
		SetGameID(s.Edges.Game.ID).
		SetScoredByTeamID(s.Edges.ScoredByTeam.ID).
		SetTeamID(s.Edges.Team.ID).
		SetSubmittedByID(s.Edges.SubmittedBy.ID).
		Save(ctx)
}

func (r *spiritScoreRepository) GetByID(ctx context.Context, id uuid.UUID) (*ent.SpiritScore, error) {
	return r.client.SpiritScore.Query().
		Where(spiritscore.ID(id)).
		WithGame().
		WithScoredByTeam().
		WithTeam().
		WithSubmittedBy().
		Only(ctx)
}

func (r *spiritScoreRepository) ListByGame(ctx context.Context, gameID uuid.UUID) ([]*ent.SpiritScore, error) {
	return r.client.SpiritScore.Query().
		Where(spiritscore.HasGameWith(game.ID(gameID))).
		Where(spiritscore.DeletedAtIsNil()).
		WithScoredByTeam().
		WithTeam().
		All(ctx)
}

func (r *spiritScoreRepository) ListByTeam(ctx context.Context, teamID uuid.UUID) ([]*ent.SpiritScore, error) {
	return r.client.SpiritScore.Query().
		Where(spiritscore.HasTeamWith(team.ID(teamID))).
		Where(spiritscore.DeletedAtIsNil()).
		WithGame().
		WithScoredByTeam().
		All(ctx)
}

func (r *spiritScoreRepository) Update(ctx context.Context, s *ent.SpiritScore) (*ent.SpiritScore, error) {
	return r.client.SpiritScore.UpdateOneID(s.ID).
		SetRulesKnowledge(s.RulesKnowledge).
		SetFoulsBodyContact(s.FoulsBodyContact).
		SetFairMindedness(s.FairMindedness).
		SetAttitude(s.Attitude).
		SetCommunication(s.Communication).
		SetNillableComments(s.Comments).
		SetUpdatedAt(time.Now()).
		Save(ctx)
}

func (r *spiritScoreRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.client.SpiritScore.UpdateOneID(id).
		SetDeletedAt(time.Now()).
		Exec(ctx)
}
