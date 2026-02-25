package repository

import (
	"context"
	"time"

	"github.com/bengobox/game-stats-api/ent"
	"github.com/bengobox/game-stats-api/ent/game"
	"github.com/bengobox/game-stats-api/ent/player"
	"github.com/bengobox/game-stats-api/ent/scoreeditrequest"
	"github.com/bengobox/game-stats-api/ent/scoring"
	"github.com/google/uuid"
)

type scoringRepository struct {
	client *ent.Client
}

// NewScoringRepository creates a new scoring repository.
func NewScoringRepository(client *ent.Client) *scoringRepository {
	return &scoringRepository{client: client}
}

func (r *scoringRepository) Create(ctx context.Context, s *ent.Scoring) (*ent.Scoring, error) {
	return r.client.Scoring.Create().
		SetGoals(s.Goals).
		SetAssists(s.Assists).
		SetBlocks(s.Blocks).
		SetTurns(s.Turns).
		SetVersion(s.Version).
		SetGameID(s.Edges.Game.ID).
		SetPlayerID(s.Edges.Player.ID).
		Save(ctx)
}

func (r *scoringRepository) GetByID(ctx context.Context, id uuid.UUID) (*ent.Scoring, error) {
	return r.client.Scoring.Query().
		Where(scoring.ID(id)).
		WithGame().
		WithPlayer(func(pq *ent.PlayerQuery) {
			pq.WithTeams() // Load teams for team detection
		}).
		Only(ctx)
}

func (r *scoringRepository) ListByGame(ctx context.Context, gameID uuid.UUID) ([]*ent.Scoring, error) {
	return r.client.Scoring.Query().
		Where(scoring.HasGameWith(game.ID(gameID))).
		Where(scoring.DeletedAtIsNil()).
		WithPlayer(func(pq *ent.PlayerQuery) {
			pq.WithTeams() // Load teams for team detection in scoring calculations
		}).
		All(ctx)
}

func (r *scoringRepository) ListByPlayer(ctx context.Context, playerID uuid.UUID) ([]*ent.Scoring, error) {
	return r.client.Scoring.Query().
		Where(scoring.HasPlayerWith(player.ID(playerID))).
		Where(scoring.DeletedAtIsNil()).
		WithGame().
		All(ctx)
}

func (r *scoringRepository) Update(ctx context.Context, s *ent.Scoring) (*ent.Scoring, error) {
	return r.client.Scoring.UpdateOneID(s.ID).
		SetGoals(s.Goals).
		SetAssists(s.Assists).
		SetBlocks(s.Blocks).
		SetTurns(s.Turns).
		SetVersion(s.Version).
		SetUpdatedAt(time.Now()).
		Save(ctx)
}

func (r *scoringRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.client.Scoring.UpdateOneID(id).
		SetDeletedAt(time.Now()).
		Exec(ctx)
}

func (r *scoringRepository) CreateScoreEditRequest(ctx context.Context, req *ent.ScoreEditRequest) (*ent.ScoreEditRequest, error) {
	return r.client.ScoreEditRequest.Create().
		SetGameID(req.GameID).
		SetRequestedByID(req.RequestedByID).
		SetPreviousHomeScore(req.PreviousHomeScore).
		SetPreviousAwayScore(req.PreviousAwayScore).
		SetNewHomeScore(req.NewHomeScore).
		SetNewAwayScore(req.NewAwayScore).
		SetReason(req.Reason).
		SetStatus(req.Status).
		Save(ctx)
}

func (r *scoringRepository) GetScoreEditRequestByID(ctx context.Context, id uuid.UUID) (*ent.ScoreEditRequest, error) {
	return r.client.ScoreEditRequest.Query().
		Where(scoreeditrequest.ID(id)).
		WithGame().
		WithRequestedBy().
		Only(ctx)
}

func (r *scoringRepository) ListScoreEditRequests(ctx context.Context, status string) ([]*ent.ScoreEditRequest, error) {
	query := r.client.ScoreEditRequest.Query().
		WithGame().
		WithRequestedBy().
		Order(ent.Desc(scoreeditrequest.FieldCreatedAt))

	if status != "" {
		query = query.Where(scoreeditrequest.StatusEQ(status))
	}

	return query.All(ctx)
}

func (r *scoringRepository) UpdateScoreEditRequest(ctx context.Context, req *ent.ScoreEditRequest) (*ent.ScoreEditRequest, error) {
	update := r.client.ScoreEditRequest.UpdateOneID(req.ID).
		SetStatus(req.Status).
		SetUpdatedAt(time.Now())

	if req.RejectionReason != "" {
		update.SetRejectionReason(req.RejectionReason)
	}
	if req.ReviewedByID != uuid.Nil {
		update.SetReviewedByID(req.ReviewedByID)
		update.SetReviewedAt(time.Now())
	}

	return update.Save(ctx)
}
