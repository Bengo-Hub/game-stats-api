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
	"github.com/bengobox/game-stats-api/ent/team"
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
		SetDivisionPoolID(g.Edges.DivisionPool.ID)

	if g.Edges.FieldLocation != nil {
		query.SetFieldLocationID(g.Edges.FieldLocation.ID)
	}

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
		WithEvent().
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

func (r *gameRepository) List(ctx context.Context, limit, offset int) ([]*ent.Game, int, error) {
	total, err := r.client.Game.Query().Where(game.DeletedAtIsNil()).Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	games, err := r.client.Game.Query().
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
	return games, total, err
}

func (r *gameRepository) ListWithFilter(ctx context.Context, filter domaingame.SearchFilter) ([]*ent.Game, int, error) {
	query := r.client.Game.Query().Where(game.DeletedAtIsNil())

	if filter.EventID != nil {
		// Need to join through division_pool to reach event
		query = query.Where(game.HasDivisionPoolWith(divisionpool.HasEventsWith(event.ID(*filter.EventID))))
	}

	if filter.DivisionPoolID != nil {
		query = query.Where(game.HasDivisionPoolWith(divisionpool.ID(*filter.DivisionPoolID)))
	}

	if filter.GameRoundID != nil {
		query = query.Where(game.HasGameRoundWith(gameround.ID(*filter.GameRoundID)))
	}

	if filter.RoundType != nil && *filter.RoundType != "" && *filter.RoundType != "all" {
		rt := *filter.RoundType
		// Handle common frontend mappings if necessary, or just use exact match
		// The frontend maps 'pool' to 'pool' or 'group'.
		if rt == "pool" {
			query = query.Where(game.HasGameRoundWith(gameround.RoundTypeIn("pool", "group")))
		} else if rt == "bracket" {
			query = query.Where(game.HasGameRoundWith(gameround.RoundTypeIn("bracket", "quarter", "semi", "quarterfinal", "semifinal")))
		} else if rt == "final" {
			query = query.Where(game.HasGameRoundWith(gameround.RoundTypeIn("final", "finals", "third_place", "third place")))
		} else {
			query = query.Where(game.HasGameRoundWith(gameround.RoundTypeEQ(rt)))
		}
	}

	if filter.Status != nil && *filter.Status != "" && *filter.Status != "all" {
		query = query.Where(game.Status(*filter.Status))
	}

	if filter.FieldID != nil {
		query = query.Where(game.HasFieldLocationWith(entfield.ID(*filter.FieldID)))
	}

	if filter.TeamID != nil {
		query = query.Where(game.Or(
			game.HasHomeTeamWith(team.ID(*filter.TeamID)),
			game.HasAwayTeamWith(team.ID(*filter.TeamID)),
		))
	}

	// Count total before applying pagination
	total, err := query.Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	if filter.Limit > 0 {
		query = query.Limit(filter.Limit)
	}
	if filter.Offset > 0 {
		query = query.Offset(filter.Offset)
	}

	games, err := query.
		WithHomeTeam().
		WithAwayTeam().
		WithFieldLocation().
		WithGameRound().
		All(ctx)
	return games, total, err
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
func (r *gameRepository) SyncGameScores(ctx context.Context, id uuid.UUID) (*ent.Game, error) {
	g, err := r.client.Game.Query().
		Where(game.ID(id)).
		WithHomeTeam().
		WithAwayTeam().
		WithScores(func(q *ent.ScoringQuery) {
			q.WithPlayer(func(pq *ent.PlayerQuery) {
				pq.WithTeams()
			})
		}).
		Only(ctx)
	if err != nil {
		return nil, err
	}

	homeScore := 0
	awayScore := 0
	for _, s := range g.Edges.Scores {
		if s.Edges.Player != nil {
			for _, team := range s.Edges.Player.Edges.Teams {
				if team.ID == g.Edges.HomeTeam.ID {
					homeScore += s.Goals
					break
				} else if team.ID == g.Edges.AwayTeam.ID {
					awayScore += s.Goals
					break
				}
			}
		}
	}

	return r.client.Game.UpdateOneID(id).
		SetHomeTeamScore(homeScore).
		SetAwayTeamScore(awayScore).
		Save(ctx)
}
