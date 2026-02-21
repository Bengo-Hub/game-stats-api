package gamemanagement

import (
	"context"
	"errors"
	"time"

	"github.com/bengobox/game-stats-api/ent"
	"github.com/bengobox/game-stats-api/internal/domain/divisionpool"
	"github.com/bengobox/game-stats-api/internal/domain/event"
	"github.com/bengobox/game-stats-api/internal/domain/field"
	"github.com/bengobox/game-stats-api/internal/domain/game"
	"github.com/bengobox/game-stats-api/internal/domain/gameevent"
	"github.com/bengobox/game-stats-api/internal/domain/gameround"
	"github.com/bengobox/game-stats-api/internal/domain/mvpnomination"
	"github.com/bengobox/game-stats-api/internal/domain/player"
	"github.com/bengobox/game-stats-api/internal/domain/scoring"
	"github.com/bengobox/game-stats-api/internal/domain/spiritnomination"
	"github.com/bengobox/game-stats-api/internal/domain/spiritscore"
	"github.com/bengobox/game-stats-api/internal/domain/team"
	"github.com/bengobox/game-stats-api/internal/domain/user"
	"github.com/google/uuid"
)

var (
	ErrGameNotFound      = errors.New("game not found")
	ErrGameRoundNotFound = errors.New("game round not found")
	ErrInvalidGameStatus = errors.New("invalid game status for this operation")
	ErrFieldConflict     = errors.New("field is already booked for this time slot")
	ErrVersionConflict   = errors.New("game has been modified by another user")
	ErrTeamNotInGame     = errors.New("team is not part of this game")
	ErrPlayerNotInTeam   = errors.New("player does not belong to this team")
	ErrUnauthorized      = errors.New("user not authorized for this operation")
)

type Service struct {
	gameRepo             game.Repository
	gameRoundRepo        gameround.Repository
	gameEventRepo        gameevent.Repository
	scoringRepo          scoring.Repository
	spiritScoreRepo      spiritscore.Repository
	mvpNominationRepo    mvpnomination.Repository
	spiritNominationRepo spiritnomination.Repository
	teamRepo             team.Repository
	playerRepo           player.Repository
	fieldRepo            field.Repository
	divisionRepo         divisionpool.Repository
	userRepo             user.Repository
	eventRepo            event.Repository
}

func NewService(
	gameRepo game.Repository,
	gameRoundRepo gameround.Repository,
	gameEventRepo gameevent.Repository,
	scoringRepo scoring.Repository,
	spiritScoreRepo spiritscore.Repository,
	mvpNominationRepo mvpnomination.Repository,
	spiritNominationRepo spiritnomination.Repository,
	teamRepo team.Repository,
	playerRepo player.Repository,
	fieldRepo field.Repository,
	divisionRepo divisionpool.Repository,
	userRepo user.Repository,
	eventRepo event.Repository,
) *Service {
	return &Service{
		gameRepo:             gameRepo,
		gameRoundRepo:        gameRoundRepo,
		gameEventRepo:        gameEventRepo,
		scoringRepo:          scoringRepo,
		spiritScoreRepo:      spiritScoreRepo,
		mvpNominationRepo:    mvpNominationRepo,
		spiritNominationRepo: spiritNominationRepo,
		teamRepo:             teamRepo,
		playerRepo:           playerRepo,
		fieldRepo:            fieldRepo,
		divisionRepo:         divisionRepo,
		userRepo:             userRepo,
		eventRepo:            eventRepo,
	}
}

// Game Management
func (s *Service) ScheduleGame(ctx context.Context, req CreateGameRequest) (*GameDTO, error) {
	// Validate teams exist and are different
	if req.HomeTeamID == req.AwayTeamID {
		return nil, errors.New("home and away teams must be different")
	}

	homeTeam, err := s.teamRepo.GetByID(ctx, req.HomeTeamID)
	if err != nil {
		return nil, err
	}

	awayTeam, err := s.teamRepo.GetByID(ctx, req.AwayTeamID)
	if err != nil {
		return nil, err
	}

	// Validate field exists
	field, err := s.fieldRepo.GetByID(ctx, req.FieldLocationID)
	if err != nil {
		return nil, err
	}

	// Check field conflict
	hasConflict, err := s.gameRepo.CheckFieldConflict(ctx, req.FieldLocationID, req.ScheduledTime, req.AllocatedTimeMinutes)
	if err != nil {
		return nil, err
	}
	if hasConflict {
		return nil, ErrFieldConflict
	}

	// Validate division pool
	division, err := s.divisionRepo.GetByID(ctx, req.DivisionPoolID)
	if err != nil {
		return nil, err
	}

	// Create game entity
	gameEntity := &ent.Game{
		Name:                 req.Name,
		ScheduledTime:        req.ScheduledTime,
		AllocatedTimeMinutes: req.AllocatedTimeMinutes,
		Status:               "scheduled",
		FirstPullBy:          req.FirstPullBy,
		Metadata:             req.Metadata,
		Edges: ent.GameEdges{
			HomeTeam:      homeTeam,
			AwayTeam:      awayTeam,
			FieldLocation: field,
			DivisionPool:  division,
		},
	}

	if req.GameRoundID != nil {
		round, err := s.gameRoundRepo.GetByID(ctx, *req.GameRoundID)
		if err != nil {
			return nil, err
		}
		gameEntity.Edges.GameRound = round
	}

	if req.ScorekeeperID != nil {
		scorekeeper, err := s.userRepo.GetByID(ctx, *req.ScorekeeperID)
		if err != nil {
			return nil, err
		}
		gameEntity.Edges.Scorekeeper = scorekeeper
	}

	created, err := s.gameRepo.Create(ctx, gameEntity)
	if err != nil {
		return nil, err
	}

	// Fetch with relations for DTO
	game, err := s.gameRepo.GetByIDWithRelations(ctx, created.ID)
	if err != nil {
		return nil, err
	}

	return mapGameToDTO(game), nil
}

func (s *Service) GetGame(ctx context.Context, id uuid.UUID) (*GameDTO, error) {
	game, err := s.gameRepo.GetByIDWithRelations(ctx, id)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, ErrGameNotFound
		}
		return nil, err
	}

	return mapGameToDTO(game), nil
}

func (s *Service) ListGames(ctx context.Context, filter ListGamesFilter) ([]*GameDTO, error) {
	// Set default pagination if not provided
	limit := filter.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}

	searchFilter := game.SearchFilter{
		EventID:        filter.EventID,
		DivisionPoolID: filter.DivisionPoolID,
		Status:         filter.Status,
		FieldID:        filter.FieldID,
		StartDate:      filter.StartDate,
		EndDate:        filter.EndDate,
		Limit:          limit,
		Offset:         offset,
	}

	games, err := s.gameRepo.ListWithFilter(ctx, searchFilter)
	if err != nil {
		return nil, err
	}

	result := make([]*GameDTO, len(games))
	for i, g := range games {
		result[i] = mapGameToDTO(g)
	}

	return result, nil
}

func (s *Service) UpdateGame(ctx context.Context, id uuid.UUID, req UpdateGameRequest) (*GameDTO, error) {
	game, err := s.gameRepo.GetByID(ctx, id)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, ErrGameNotFound
		}
		return nil, err
	}

	// Only allow updates to scheduled games
	if game.Status != "scheduled" {
		return nil, ErrInvalidGameStatus
	}

	if req.Name != nil {
		game.Name = *req.Name
	}
	if req.ScheduledTime != nil {
		game.ScheduledTime = *req.ScheduledTime
	}
	if req.AllocatedTimeMinutes != nil {
		game.AllocatedTimeMinutes = *req.AllocatedTimeMinutes
	}
	if req.FirstPullBy != nil {
		game.FirstPullBy = req.FirstPullBy
	}
	if req.Metadata != nil {
		game.Metadata = req.Metadata
	}

	if req.ScorekeeperID != nil {
		scorekeeper, err := s.userRepo.GetByID(ctx, *req.ScorekeeperID)
		if err != nil {
			return nil, err
		}
		game.Edges.Scorekeeper = scorekeeper
	}

	updated, err := s.gameRepo.Update(ctx, game)
	if err != nil {
		return nil, err
	}

	// Fetch with relations
	result, err := s.gameRepo.GetByIDWithRelations(ctx, updated.ID)
	if err != nil {
		return nil, err
	}

	return mapGameToDTO(result), nil
}

func (s *Service) CancelGame(ctx context.Context, id uuid.UUID) error {
	game, err := s.gameRepo.GetByID(ctx, id)
	if err != nil {
		if ent.IsNotFound(err) {
			return ErrGameNotFound
		}
		return err
	}

	// Can only cancel scheduled or in-progress games
	if game.Status != "scheduled" && game.Status != "in_progress" {
		return ErrInvalidGameStatus
	}

	return s.gameRepo.Delete(ctx, id)
}

// Game Timer System
func (s *Service) StartGame(ctx context.Context, id uuid.UUID, userID uuid.UUID, req StartGameRequest) (*GameDTO, error) {
	game, err := s.gameRepo.GetByIDWithRelations(ctx, id)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, ErrGameNotFound
		}
		return nil, err
	}

	// Verify scorekeeper or admin permission
	if game.Edges.Scorekeeper == nil || game.Edges.Scorekeeper.ID != userID {
		// Admin check handled by middleware - only scorekeeper can proceed here
		return nil, ErrUnauthorized
	}

	// Can only start scheduled games
	if game.Status != "scheduled" {
		return nil, ErrInvalidGameStatus
	}

	now := time.Now()
	expectedEnd := now.Add(time.Duration(game.AllocatedTimeMinutes) * time.Minute)

	// Update game
	updated, err := s.gameRepo.UpdateWithVersion(ctx, id, game.Version, func(update *ent.GameUpdateOne) *ent.GameUpdateOne {
		return update.
			SetStatus("in_progress").
			SetActualStartTime(now).
			SetActualEndTime(expectedEnd)
	})
	if err != nil {
		return nil, err
	}

	// Create game start event
	_, err = s.gameEventRepo.Create(ctx, &ent.GameEvent{
		EventType:   "game_started",
		Minute:      0,
		Second:      0,
		Description: "Game started",
		Edges: ent.GameEventEdges{
			Game: updated,
		},
	})
	if err != nil {
		return nil, err
	}

	// SSE events are broadcast via the stream handler when clients poll for updates
	// Auto-finish is handled by the game timer background worker

	result, err := s.gameRepo.GetByIDWithRelations(ctx, updated.ID)
	if err != nil {
		return nil, err
	}

	return mapGameToDTO(result), nil
}

func (s *Service) FinishGame(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*GameDTO, error) {
	game, err := s.gameRepo.GetByIDWithRelations(ctx, id)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, ErrGameNotFound
		}
		return nil, err
	}

	// Verify scorekeeper or admin
	if game.Edges.Scorekeeper == nil || game.Edges.Scorekeeper.ID != userID {
		return nil, ErrUnauthorized
	}

	// Can only finish in-progress games
	if game.Status != "in_progress" {
		return nil, ErrInvalidGameStatus
	}

	// Update to finished status (scores can still be edited)
	updated, err := s.gameRepo.UpdateWithVersion(ctx, id, game.Version, func(update *ent.GameUpdateOne) *ent.GameUpdateOne {
		return update.SetStatus("finished")
	})
	if err != nil {
		return nil, err
	}

	// Create game finished event
	elapsed := time.Since(*game.ActualStartTime)
	minute := int(elapsed.Minutes())
	second := int(elapsed.Seconds()) % 60

	_, err = s.gameEventRepo.Create(ctx, &ent.GameEvent{
		EventType:   "game_finished",
		Minute:      minute,
		Second:      second,
		Description: "Game time expired",
		Edges: ent.GameEventEdges{
			Game: updated,
		},
	})
	if err != nil {
		return nil, err
	}

	// SSE events are broadcast via the stream handler when clients poll for updates

	result, err := s.gameRepo.GetByIDWithRelations(ctx, updated.ID)
	if err != nil {
		return nil, err
	}

	return mapGameToDTO(result), nil
}

func (s *Service) EndGame(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*GameDTO, error) {
	game, err := s.gameRepo.GetByIDWithRelations(ctx, id)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, ErrGameNotFound
		}
		return nil, err
	}

	// Verify scorekeeper or admin
	if game.Edges.Scorekeeper == nil || game.Edges.Scorekeeper.ID != userID {
		return nil, ErrUnauthorized
	}

	// Can only end finished games
	if game.Status != "finished" {
		return nil, ErrInvalidGameStatus
	}

	// Final submission - no more edits allowed
	updated, err := s.gameRepo.UpdateWithVersion(ctx, id, game.Version, func(update *ent.GameUpdateOne) *ent.GameUpdateOne {
		return update.SetStatus("ended")
	})
	if err != nil {
		return nil, err
	}

	// Create game ended event
	_, err = s.gameEventRepo.Create(ctx, &ent.GameEvent{
		EventType:   "game_ended",
		Minute:      0,
		Second:      0,
		Description: "Game finalized by scorekeeper",
		Edges: ent.GameEventEdges{
			Game: updated,
		},
	})
	if err != nil {
		return nil, err
	}

	// SSE events are broadcast via the stream handler when clients poll for updates
	// Ranking recalculation is triggered automatically via event-driven architecture

	result, err := s.gameRepo.GetByIDWithRelations(ctx, updated.ID)
	if err != nil {
		return nil, err
	}

	return mapGameToDTO(result), nil
}

func (s *Service) RecordStoppage(ctx context.Context, id uuid.UUID, userID uuid.UUID, req RecordStoppageRequest) (*GameDTO, error) {
	game, err := s.gameRepo.GetByIDWithRelations(ctx, id)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, ErrGameNotFound
		}
		return nil, err
	}

	// Verify scorekeeper
	if game.Edges.Scorekeeper == nil || game.Edges.Scorekeeper.ID != userID {
		return nil, ErrUnauthorized
	}

	// Can only record stoppages during in-progress games
	if game.Status != "in_progress" {
		return nil, ErrInvalidGameStatus
	}

	// Update game with stoppage time
	newStoppageTime := game.StoppageTimeSeconds + req.DurationSeconds
	newEndTime := game.ActualEndTime.Add(time.Duration(req.DurationSeconds) * time.Second)

	updated, err := s.gameRepo.UpdateWithVersion(ctx, id, game.Version, func(update *ent.GameUpdateOne) *ent.GameUpdateOne {
		return update.
			SetStoppageTimeSeconds(newStoppageTime).
			SetActualEndTime(newEndTime)
	})
	if err != nil {
		return nil, err
	}

	// Create stoppage event
	elapsed := time.Since(*game.ActualStartTime)
	minute := int(elapsed.Minutes())
	second := int(elapsed.Seconds()) % 60

	_, err = s.gameEventRepo.Create(ctx, &ent.GameEvent{
		EventType:   "stoppage_recorded",
		Minute:      minute,
		Second:      second,
		Description: req.Reason,
		Metadata: map[string]interface{}{
			"duration_seconds": req.DurationSeconds,
		},
		Edges: ent.GameEventEdges{
			Game: updated,
		},
	})
	if err != nil {
		return nil, err
	}

	// SSE events are broadcast via the stream handler when clients poll for updates

	result, err := s.gameRepo.GetByIDWithRelations(ctx, updated.ID)
	if err != nil {
		return nil, err
	}

	return mapGameToDTO(result), nil
}

// Game Timeline
func (s *Service) GetGameTimeline(ctx context.Context, id uuid.UUID) (*GameTimelineDTO, error) {
	events, err := s.gameEventRepo.ListByGame(ctx, id)
	if err != nil {
		return nil, err
	}

	timeline := &GameTimelineDTO{
		Events: make([]GameEventDTO, len(events)),
	}

	for i, event := range events {
		timeline.Events[i] = GameEventDTO{
			ID:          event.ID,
			EventType:   event.EventType,
			Minute:      event.Minute,
			Second:      event.Second,
			Description: event.Description,
			Metadata:    event.Metadata,
			CreatedAt:   event.CreatedAt,
		}
	}

	return timeline, nil
}

// Mapper functions
func mapGameToDTO(g *ent.Game) *GameDTO {
	dto := &GameDTO{
		ID:                   g.ID,
		Name:                 g.Name,
		ScheduledTime:        g.ScheduledTime,
		ActualStartTime:      g.ActualStartTime,
		ActualEndTime:        g.ActualEndTime,
		AllocatedTimeMinutes: g.AllocatedTimeMinutes,
		StoppageTimeSeconds:  g.StoppageTimeSeconds,
		Status:               g.Status,
		HomeTeamScore:        g.HomeTeamScore,
		AwayTeamScore:        g.AwayTeamScore,
		FirstPullBy:          g.FirstPullBy,
		Version:              g.Version,
		Metadata:             g.Metadata,
		CreatedAt:            g.CreatedAt,
		UpdatedAt:            g.UpdatedAt,
	}

	if g.Edges.HomeTeam != nil {
		dto.HomeTeam = &TeamSummaryDTO{
			ID:      g.Edges.HomeTeam.ID,
			Name:    g.Edges.HomeTeam.Name,
			LogoURL: g.Edges.HomeTeam.LogoURL,
		}
	}

	if g.Edges.AwayTeam != nil {
		dto.AwayTeam = &TeamSummaryDTO{
			ID:      g.Edges.AwayTeam.ID,
			Name:    g.Edges.AwayTeam.Name,
			LogoURL: g.Edges.AwayTeam.LogoURL,
		}
	}

	if g.Edges.FieldLocation != nil {
		dto.FieldLocation = &FieldSummaryDTO{
			ID:   g.Edges.FieldLocation.ID,
			Name: g.Edges.FieldLocation.Name,
		}
	}

	if g.Edges.GameRound != nil {
		dto.GameRound = &GameRoundSummaryDTO{
			ID:        g.Edges.GameRound.ID,
			Name:      g.Edges.GameRound.Name,
			RoundType: g.Edges.GameRound.RoundType,
		}
	}

	if g.Edges.Scorekeeper != nil {
		dto.Scorekeeper = &UserSummaryDTO{
			ID:    g.Edges.Scorekeeper.ID,
			Name:  g.Edges.Scorekeeper.Name,
			Email: g.Edges.Scorekeeper.Email,
		}
	}

	return dto
}
