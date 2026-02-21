package repository

import (
	"context"
	"errors"
	"time"

	"github.com/bengobox/game-stats-api/ent"
	"github.com/bengobox/game-stats-api/ent/divisionpool"
	"github.com/bengobox/game-stats-api/ent/event"
	entfield "github.com/bengobox/game-stats-api/ent/field"
	"github.com/bengobox/game-stats-api/ent/game"
	"github.com/bengobox/game-stats-api/ent/gameround"
	domaingame "github.com/bengobox/game-stats-api/internal/domain/game"
	"github.com/google/uuid"
)

type gameRepository struct {
	client *ent.Client
}

// NewGameRepository creates a new game repository.
func NewGameRepository(client *ent.Client) *gameRepository {
	return &gameRepository{client: client}
}

func (r *gameRepository) Create(ctx context.Context, g *ent.Game) (*ent.Game, error) {
	query := r.client.Game.Create().
		SetName(g.Name).
		SetScheduledTime(g.ScheduledTime).
		SetAllocatedTimeMinutes(g.AllocatedTimeMinutes).
		SetStatus(g.Status).
		SetHomeTeamID(g.Edges.HomeTeam.ID).
		SetAwayTeamID(g.Edges.AwayTeam.ID).
		SetDivisionPoolID(g.Edges.DivisionPool.ID).
		SetFieldLocationID(g.Edges.FieldLocation.ID)

	if g.Edges.GameRound != nil {
		query.SetGameRoundID(g.Edges.GameRound.ID)
	}

	if g.Edges.Scorekeeper != nil {
		query.SetScorekeeperID(g.Edges.Scorekeeper.ID)
	}

	if g.FirstPullBy != nil {
		query.SetFirstPullBy(*g.FirstPullBy)
	}

	if g.Metadata != nil {
		query.SetMetadata(g.Metadata)
	}

	return query.Save(ctx)
}

func (r *gameRepository) GetByID(ctx context.Context, id uuid.UUID) (*ent.Game, error) {
	return r.client.Game.Query().
		Where(game.ID(id)).
		Only(ctx)
}

func (r *gameRepository) GetByIDWithRelations(ctx context.Context, id uuid.UUID) (*ent.Game, error) {
	return r.client.Game.Query().
		Where(game.ID(id)).
		WithHomeTeam().
		WithAwayTeam().
		WithDivisionPool().
		WithFieldLocation().
		WithGameRound().
		WithScorekeeper().
		WithScores(func(q *ent.ScoringQuery) {
			q.WithPlayer()
		}).
		WithGameEvents().
		WithSpiritScores().
		Only(ctx)
}

func (r *gameRepository) ListByDivision(ctx context.Context, divisionID uuid.UUID) ([]*ent.Game, error) {
	return r.client.Game.Query().
		Where(game.HasDivisionPoolWith(divisionpool.ID(divisionID))).
		Where(game.DeletedAtIsNil()).
		WithHomeTeam().
		WithAwayTeam().
		WithFieldLocation().
		WithGameRound().
		Order(ent.Asc(game.FieldScheduledTime)).
		All(ctx)
}

func (r *gameRepository) ListByRound(ctx context.Context, roundID uuid.UUID) ([]*ent.Game, error) {
	return r.client.Game.Query().
		Where(game.HasGameRoundWith(gameround.ID(roundID))).
		Where(game.DeletedAtIsNil()).
		WithHomeTeam().
		WithAwayTeam().
		WithGameRound().
		WithDivisionPool().
		WithFieldLocation().
		Order(ent.Asc(game.FieldScheduledTime)).
		All(ctx)
}

func (r *gameRepository) ListByStatus(ctx context.Context, status string) ([]*ent.Game, error) {
	return r.client.Game.Query().
		Where(game.Status(status)).
		Where(game.DeletedAtIsNil()).
		WithHomeTeam().
		WithAwayTeam().
		WithFieldLocation().
		Order(ent.Asc(game.FieldScheduledTime)).
		All(ctx)
}

func (r *gameRepository) ListByField(ctx context.Context, fieldID uuid.UUID) ([]*ent.Game, error) {
	return r.client.Game.Query().
		Where(game.HasFieldLocationWith(entfield.ID(fieldID))).
		Where(game.DeletedAtIsNil()).
		WithHomeTeam().
		WithAwayTeam().
		Order(ent.Asc(game.FieldScheduledTime)).
		All(ctx)
}

func (r *gameRepository) ListByDateRange(ctx context.Context, start, end time.Time) ([]*ent.Game, error) {
	return r.client.Game.Query().
		Where(
			game.And(
				game.ScheduledTimeGTE(start),
				game.ScheduledTimeLTE(end),
			),
		).
		Where(game.DeletedAtIsNil()).
		WithHomeTeam().
		WithAwayTeam().
		WithFieldLocation().
		WithGameRound().
		Order(ent.Asc(game.FieldScheduledTime)).
		All(ctx)
}

func (r *gameRepository) List(ctx context.Context, limit, offset int) ([]*ent.Game, error) {
	return r.client.Game.Query().
		Where(game.DeletedAtIsNil()).
		WithHomeTeam().
		WithAwayTeam().
		WithFieldLocation().
		WithGameRound().
		WithDivisionPool().
		Order(ent.Desc(game.FieldScheduledTime)).
		Limit(limit).
		Offset(offset).
		All(ctx)
}

func (r *gameRepository) ListWithFilter(ctx context.Context, filter domaingame.SearchFilter) ([]*ent.Game, error) {
	query := r.client.Game.Query().Where(game.DeletedAtIsNil())

	if filter.EventID != nil {
		// Need to join through division_pool to reach event
		query = query.Where(game.HasDivisionPoolWith(divisionpool.HasEventWith(event.ID(*filter.EventID))))
	}

	if filter.DivisionPoolID != nil {
		query = query.Where(game.HasDivisionPoolWith(divisionpool.ID(*filter.DivisionPoolID)))
	}

	if filter.Status != nil && *filter.Status != "" && *filter.Status != "all" {
		query = query.Where(game.Status(*filter.Status))
	}

	if filter.FieldID != nil {
		query = query.Where(game.HasFieldLocationWith(entfield.ID(*filter.FieldID)))
	}

	if filter.StartDate != nil && filter.EndDate != nil {
		query = query.Where(
			game.And(
				game.ScheduledTimeGTE(*filter.StartDate),
				game.ScheduledTimeLTE(*filter.EndDate),
			),
		)
	}

	// Default sort by schedule time
	query = query.Order(ent.Asc(game.FieldScheduledTime))

	if filter.Limit > 0 {
		query = query.Limit(filter.Limit)
	}
	if filter.Offset > 0 {
		query = query.Offset(filter.Offset)
	}

	return query.
		WithHomeTeam().
		WithAwayTeam().
		WithFieldLocation().
		WithGameRound().
		All(ctx)
}

func (r *gameRepository) Update(ctx context.Context, g *ent.Game) (*ent.Game, error) {
	query := r.client.Game.UpdateOneID(g.ID).
		SetName(g.Name).
		SetScheduledTime(g.ScheduledTime).
		SetAllocatedTimeMinutes(g.AllocatedTimeMinutes).
		SetStoppageTimeSeconds(g.StoppageTimeSeconds).
		SetStatus(g.Status).
		SetHomeTeamScore(g.HomeTeamScore).
		SetAwayTeamScore(g.AwayTeamScore).
		SetVersion(g.Version + 1).
		SetUpdatedAt(time.Now())

	if g.ActualStartTime != nil {
		query.SetActualStartTime(*g.ActualStartTime)
	}

	if g.ActualEndTime != nil {
		query.SetActualEndTime(*g.ActualEndTime)
	}

	if g.FirstPullBy != nil {
		query.SetFirstPullBy(*g.FirstPullBy)
	}

	if g.Metadata != nil {
		query.SetMetadata(g.Metadata)
	}

	if g.Edges.Scorekeeper != nil {
		query.SetScorekeeperID(g.Edges.Scorekeeper.ID)
	}

	return query.Save(ctx)
}

func (r *gameRepository) UpdateWithVersion(ctx context.Context, id uuid.UUID, version int, updateFn func(*ent.GameUpdateOne) *ent.GameUpdateOne) (*ent.Game, error) {
	tx, err := r.client.Tx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// Check current version
	current, err := tx.Game.Query().Where(game.ID(id)).Only(ctx)
	if err != nil {
		return nil, err
	}

	if current.Version != version {
		return nil, errors.New("version conflict: game has been modified")
	}

	// Apply updates with version increment
	query := tx.Game.UpdateOneID(id).
		SetVersion(version + 1).
		SetUpdatedAt(time.Now())

	query = updateFn(query)

	updated, err := query.Save(ctx)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return updated, nil
}

func (r *gameRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.client.Game.UpdateOneID(id).
		SetDeletedAt(time.Now()).
		Exec(ctx)
}

func (r *gameRepository) CheckFieldConflict(ctx context.Context, fieldID uuid.UUID, scheduledTime time.Time, duration int) (bool, error) {
	endTime := scheduledTime.Add(time.Duration(duration) * time.Minute)

	count, err := r.client.Game.Query().
		Where(
			game.And(
				game.HasFieldLocationWith(entfield.ID(fieldID)),
				game.StatusNEQ("cancelled"),
				game.DeletedAtIsNil(),
				game.Or(
					// New game starts during existing game
					game.And(
						game.ScheduledTimeLTE(scheduledTime),
						game.ScheduledTimeGTE(scheduledTime),
					),
					// New game ends during existing game
					game.And(
						game.ScheduledTimeLTE(endTime),
						game.ScheduledTimeGTE(endTime),
					),
					// New game completely overlaps existing game
					game.And(
						game.ScheduledTimeGTE(scheduledTime),
						game.ScheduledTimeLTE(endTime),
					),
				),
			),
		).
		Count(ctx)

	if err != nil {
		return false, err
	}

	return count > 0, nil
}
