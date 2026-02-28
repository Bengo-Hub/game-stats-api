package repository

import (
	"context"
	"time"

	"github.com/bengobox/game-stats-api/ent"
	"github.com/bengobox/game-stats-api/ent/event"
	"github.com/bengobox/game-stats-api/ent/gameround"
	"github.com/google/uuid"
)

type gameRoundRepository struct {
	client *ent.Client
}

// NewGameRoundRepository creates a new game round repository.
func NewGameRoundRepository(client *ent.Client) *gameRoundRepository {
	return &gameRoundRepository{client: client}
}

func (r *gameRoundRepository) Create(ctx context.Context, round *ent.GameRound) (*ent.GameRound, error) {
	query := r.client.GameRound.Create().
		SetName(round.Name).
		SetRoundType(round.RoundType).
		AddEventIDs(round.Edges.Events[0].ID).
		SetAutoAdvance(round.AutoAdvance)

	if round.RoundNumber != nil {
		query.SetRoundNumber(*round.RoundNumber)
	}

	if round.StartDate != nil {
		query.SetStartDate(*round.StartDate)
	}

	if round.EndDate != nil {
		query.SetEndDate(*round.EndDate)
	}

	if round.TopNTeams != nil {
		query.SetTopNTeams(*round.TopNTeams)
	}

	return query.Save(ctx)
}

func (r *gameRoundRepository) GetByID(ctx context.Context, id uuid.UUID) (*ent.GameRound, error) {
	return r.client.GameRound.Query().
		Where(gameround.ID(id)).
		WithEvents().
		Only(ctx)
}

func (r *gameRoundRepository) GetByIDWithGames(ctx context.Context, id uuid.UUID) (*ent.GameRound, error) {
	return r.client.GameRound.Query().
		Where(gameround.ID(id)).
		WithEvents().
		WithGames(func(q *ent.GameQuery) {
			q.WithHomeTeam().
				WithAwayTeam().
				WithFieldLocation()
		}).
		Only(ctx)
}

func (r *gameRoundRepository) ListByEvent(ctx context.Context, eventID uuid.UUID) ([]*ent.GameRound, error) {
	return r.client.GameRound.Query().
		Where(gameround.HasEventsWith(event.ID(eventID))).
		Where(gameround.DeletedAtIsNil()).
		WithEvents().
		Order(ent.Asc(gameround.FieldRoundNumber)).
		All(ctx)
}

func (r *gameRoundRepository) ListAll(ctx context.Context) ([]*ent.GameRound, error) {
	return r.client.GameRound.Query().
		Where(gameround.DeletedAtIsNil()).
		WithEvents().
		Order(ent.Asc(gameround.FieldRoundNumber)).
		All(ctx)
}

func (r *gameRoundRepository) Update(ctx context.Context, round *ent.GameRound) (*ent.GameRound, error) {
	query := r.client.GameRound.UpdateOneID(round.ID).
		SetName(round.Name).
		SetRoundType(round.RoundType).
		SetAutoAdvance(round.AutoAdvance).
		SetUpdatedAt(time.Now())

	if round.RoundNumber != nil {
		query.SetRoundNumber(*round.RoundNumber)
	}

	if round.StartDate != nil {
		query.SetStartDate(*round.StartDate)
	}

	if round.EndDate != nil {
		query.SetEndDate(*round.EndDate)
	}

	if round.TopNTeams != nil {
		query.SetTopNTeams(*round.TopNTeams)
	}

	return query.Save(ctx)
}

func (r *gameRoundRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.client.GameRound.UpdateOneID(id).
		SetDeletedAt(time.Now()).
		Exec(ctx)
}

func (r *gameRoundRepository) GetClient() *ent.Client {
	return r.client
}
