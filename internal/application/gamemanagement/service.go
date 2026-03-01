package gamemanagement

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/bengobox/game-stats-api/ent"
	"github.com/bengobox/game-stats-api/internal/domain/auth"
	"github.com/bengobox/game-stats-api/internal/domain/divisionpool"
	"github.com/bengobox/game-stats-api/internal/domain/event"
	"github.com/bengobox/game-stats-api/internal/domain/eventparticipation"
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
	"github.com/bengobox/game-stats-api/internal/infrastructure/cache"
	"github.com/google/uuid"
)

type AdvancementService interface {
	HandleGameEnded(ctx context.Context, gameID uuid.UUID) error
}

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
	participationRepo    eventparticipation.Repository
	scoreDomainService   *scoring.ScoreService
	permissionService    *auth.PermissionService
	advancementService   AdvancementService
	cache                *cache.RedisClient
	client               *ent.Client
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
	participationRepo eventparticipation.Repository,
	permissionService *auth.PermissionService,
	advancementService AdvancementService,
	cacheClient *cache.RedisClient,
	client *ent.Client,
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
		participationRepo:    participationRepo,
		scoreDomainService:   scoring.NewScoreService(scoringRepo),
		permissionService:    permissionService,
		advancementService:   advancementService,
		cache:                cacheClient,
		client:               client,
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

	// Validate field exists if provided
	var field *ent.Field
	if req.FieldLocationID != nil {
		f, err := s.fieldRepo.GetByID(ctx, *req.FieldLocationID)
		if err != nil {
			return nil, err
		}
		field = f

		// Check field conflict
		hasConflict, err := s.gameRepo.CheckFieldConflict(ctx, *req.FieldLocationID, req.ScheduledTime, req.AllocatedTimeMinutes)
		if err != nil {
			return nil, err
		}
		if hasConflict {
			return nil, ErrFieldConflict
		}
	}

	// Validate division pool
	division, err := s.divisionRepo.GetByID(ctx, req.DivisionPoolID)
	if err != nil {
		return nil, err
	}

	// Auto-generate game name if not provided
	var gameName string
	if req.Name == nil || *req.Name == "" {
		gameName = fmt.Sprintf("%s vs %s", homeTeam.Name, awayTeam.Name)
	} else {
		gameName = *req.Name
	}

	// Create game entity
	gameEntity := &ent.Game{
		Name:                 gameName,
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

	// Get scorekeeper if provided
	if req.ScorekeeperID != nil {
		scorekeeper, err := s.userRepo.GetByID(ctx, *req.ScorekeeperID)
		if err != nil {
			return nil, fmt.Errorf("scorekeeper not found: %w", err)
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

	dto := mapGameToDTO(game)
	s.cacheGame(ctx, dto)
	return dto, nil
}

func (s *Service) GetGame(ctx context.Context, id uuid.UUID) (*GameDTO, error) {
	// Try cache first
	if s.cache != nil {
		cacheKey := cache.CacheKey("game", id.String())
		var cached GameDTO
		if err := s.cache.GetJSON(ctx, cacheKey, &cached); err == nil && cached.ID != uuid.Nil {
			return &cached, nil
		}
	}

	game, err := s.gameRepo.GetByIDWithRelations(ctx, id)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, ErrGameNotFound
		}
		return nil, err
	}

	dto := mapGameToDTO(game)

	// Cache the result with status-based TTL
	s.cacheGame(ctx, dto)

	return dto, nil
}

func (s *Service) ListGames(ctx context.Context, filter ListGamesFilter) ([]*GameDTO, int, error) {
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
		GameRoundID:    filter.GameRoundID,
		Status:         filter.Status,
		FieldID:        filter.FieldID,
		StartDate:      filter.StartDate,
		EndDate:        filter.EndDate,
		RoundType:      filter.RoundType,
		TeamID:         filter.TeamID,
		Limit:          limit,
		Offset:         offset,
	}

	games, total, err := s.gameRepo.ListWithFilter(ctx, searchFilter)
	if err != nil {
		return nil, 0, err
	}

	result := make([]*GameDTO, len(games))
	for i, g := range games {
		// Automatic cancellation check for list view
		g, _ = s.checkAndCancelExpiredGame(ctx, g)
		dto := mapGameToDTO(g)
		// Cache individual games as we list them
		s.cacheGame(ctx, dto)
		result[i] = dto
	}

	return result, total, nil
}

func (s *Service) UpdateGame(ctx context.Context, id uuid.UUID, req UpdateGameRequest) (*GameDTO, error) {
	game, err := s.gameRepo.GetByID(ctx, id)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, ErrGameNotFound
		}
		return nil, err
	}

	// Allow updates to scheduled or in-progress games for rescheduling
	if game.Status != "scheduled" && game.Status != "in_progress" {
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

	if req.HomeTeamID != nil {
		team, err := s.teamRepo.GetByID(ctx, *req.HomeTeamID)
		if err != nil {
			return nil, fmt.Errorf("home team not found: %w", err)
		}

		// If both are provided, update both at once. Otherwise, ensure they aren't the same as existing.
		awayID := game.Edges.AwayTeam.ID
		if req.AwayTeamID != nil {
			awayID = *req.AwayTeamID
		}
		if team.ID == awayID {
			return nil, fmt.Errorf("home and away teams must be different")
		}

		game.Edges.HomeTeam = team
	}

	if req.AwayTeamID != nil {
		team, err := s.teamRepo.GetByID(ctx, *req.AwayTeamID)
		if err != nil {
			return nil, fmt.Errorf("away team not found: %w", err)
		}

		homeID := game.Edges.HomeTeam.ID
		if req.HomeTeamID != nil {
			homeID = *req.HomeTeamID
		}
		if team.ID == homeID {
			return nil, fmt.Errorf("home and away teams must be different")
		}

		game.Edges.AwayTeam = team
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

	if req.FieldLocationID != nil {
		field, err := s.fieldRepo.GetByID(ctx, *req.FieldLocationID)
		if err != nil {
			return nil, err
		}
		game.Edges.FieldLocation = field
	}

	if req.GameRoundID != nil {
		round, err := s.gameRoundRepo.GetByID(ctx, *req.GameRoundID)
		if err != nil {
			return nil, err
		}
		game.Edges.GameRound = round
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

func (s *Service) CancelGame(ctx context.Context, id uuid.UUID) (*GameDTO, error) {
	game, err := s.gameRepo.GetByID(ctx, id)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, ErrGameNotFound
		}
		return nil, err
	}

	// Update status to canceled instead of deleting
	updated, err := s.gameRepo.UpdateWithVersion(ctx, id, game.Version, func(u *ent.GameUpdateOne) *ent.GameUpdateOne {
		return u.SetStatus("canceled")
	})
	if err != nil {
		return nil, err
	}

	// Invalidate cache
	s.invalidateGameCache(ctx, id)

	// Fetch fresh with relations for response
	result, err := s.gameRepo.GetByIDWithRelations(ctx, updated.ID)
	if err != nil {
		return nil, err
	}

	return mapGameToDTO(result), nil
}

func (s *Service) DeleteGame(ctx context.Context, id uuid.UUID) error {
	game, err := s.gameRepo.GetByID(ctx, id)
	if err != nil {
		if ent.IsNotFound(err) {
			return ErrGameNotFound
		}
		return err
	}

	// Restriction: Only allow deleting cancelled games
	if game.Status != "canceled" {
		return fmt.Errorf("only cancelled games can be permanently deleted (current status: %s)", game.Status)
	}

	// Perform hard delete
	if err := s.gameRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete game: %w", err)
	}

	// Invalidate cache
	s.invalidateGameCache(ctx, id)

	return nil
}

func (s *Service) checkAndCancelExpiredGame(ctx context.Context, g *ent.Game) (*ent.Game, error) {
	if g.Status != "scheduled" {
		return g, nil
	}

	now := time.Now()
	y, m, d := now.Date()
	startOfToday := time.Date(y, m, d, 0, 0, 0, 0, now.Location())

	// If scheduled date is before today, it's expired
	if g.ScheduledTime.Before(startOfToday) {
		updated, err := s.gameRepo.UpdateWithVersion(ctx, g.ID, g.Version, func(u *ent.GameUpdateOne) *ent.GameUpdateOne {
			return u.SetStatus("canceled")
		})
		if err != nil {
			return g, err
		}
		// Clear cache if exists
		s.invalidateGameCache(ctx, g.ID)
		return updated, nil
	}

	return g, nil
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

	// Verify scorekeeper or higher privilege (admin/event_manager)
	isScorekeeper := game.Edges.Scorekeeper != nil && game.Edges.Scorekeeper.ID == userID

	// Get role from context
	userRole, _ := ctx.Value("user_role").(string)
	isAdminOrEventManager := userRole == "admin" || userRole == "event_manager"

	if !isScorekeeper && !isAdminOrEventManager {
		return nil, ErrUnauthorized
	}

	// Can only start scheduled games
	if game.Status != "scheduled" {
		return nil, ErrInvalidGameStatus
	}

	now := time.Now()
	// Validation: scheduled time must be less than or equal to the current time
	if now.Before(game.ScheduledTime) {
		return nil, fmt.Errorf("cannot start game before scheduled time: %s", game.ScheduledTime.Format(time.Kitchen))
	}

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

	s.invalidateGameCache(ctx, id)
	dto := mapGameToDTO(result)
	s.cacheGame(ctx, dto)
	return dto, nil
}

func (s *Service) EndGame(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*GameDTO, error) {
	game, err := s.gameRepo.GetByID(ctx, id)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, ErrGameNotFound
		}
		return nil, err
	}

	// Get role from context
	userRole, _ := ctx.Value("user_role").(string)
	isAdminOrEventManager := userRole == "admin" || userRole == "event_manager"

	// Verify scorekeeper or admin
	isScorekeeper := game.Edges.Scorekeeper != nil && game.Edges.Scorekeeper.ID == userID

	if !isScorekeeper && !isAdminOrEventManager {
		return nil, ErrUnauthorized
	}

	return s.endGameInternal(ctx, id, "Game time expired")
}

func (s *Service) CompleteGame(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*GameDTO, error) {
	game, err := s.gameRepo.GetByIDWithRelations(ctx, id)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, ErrGameNotFound
		}
		return nil, err
	}

	// Get role from context
	userRole, _ := ctx.Value("user_role").(string)
	isAdminOrEventManager := userRole == "admin" || userRole == "event_manager"

	// Verify scorekeeper or admin
	isScorekeeper := game.Edges.Scorekeeper != nil && game.Edges.Scorekeeper.ID == userID

	if !isScorekeeper && !isAdminOrEventManager {
		return nil, ErrUnauthorized
	}

	// Can only complete ended games
	if game.Status != "ended" {
		return nil, ErrInvalidGameStatus
	}

	// Final submission - no more edits allowed
	updated, err := s.gameRepo.UpdateWithVersion(ctx, id, game.Version, func(update *ent.GameUpdateOne) *ent.GameUpdateOne {
		return update.SetStatus("completed")
	})
	if err != nil {
		return nil, err
	}

	// Create game completed event
	_, err = s.gameEventRepo.Create(ctx, &ent.GameEvent{
		EventType:   "game_completed",
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
	// Ranking recalculation and automated advancement
	if s.advancementService != nil {
		go func() {
			ctx := context.Background() // Use fresh context for background task
			if err := s.advancementService.HandleGameEnded(ctx, updated.ID); err != nil {
				// Log error (pro-active logging should be here)
			}
		}()
	}

	result, err := s.gameRepo.GetByIDWithRelations(ctx, updated.ID)
	if err != nil {
		return nil, err
	}

	s.invalidateGameCache(ctx, id)
	dto := mapGameToDTO(result)
	s.cacheGame(ctx, dto)
	return dto, nil
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

	if g.Edges.Event != nil {
		dto.EventID = g.Edges.Event.ID
	}

	if g.Edges.HomeTeam != nil {
		dto.HomeTeam = &TeamSummaryDTO{
			ID:             g.Edges.HomeTeam.ID,
			Name:           g.Edges.HomeTeam.Name,
			LogoURL:        g.Edges.HomeTeam.LogoURL,
			PrimaryColor:   g.Edges.HomeTeam.PrimaryColor,
			SecondaryColor: g.Edges.HomeTeam.SecondaryColor,
		}
	}

	if g.Edges.AwayTeam != nil {
		dto.AwayTeam = &TeamSummaryDTO{
			ID:             g.Edges.AwayTeam.ID,
			Name:           g.Edges.AwayTeam.Name,
			LogoURL:        g.Edges.AwayTeam.LogoURL,
			PrimaryColor:   g.Edges.AwayTeam.PrimaryColor,
			SecondaryColor: g.Edges.AwayTeam.SecondaryColor,
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

// cacheGame stores a game DTO in Redis with a TTL based on its status.
func (s *Service) cacheGame(ctx context.Context, dto *GameDTO) {
	if s.cache == nil || dto == nil {
		return
	}
	cacheKey := cache.CacheKey("game", dto.ID.String())

	ttl := cache.TTLGameLive // default for scheduled / in_progress / ended
	if dto.Status == "completed" || dto.Status == "canceled" {
		ttl = cache.TTLGameCompleted
	}
	_ = s.cache.SetJSON(ctx, cacheKey, dto, ttl)
}

// invalidateGameCache removes a game's cached DTO from Redis.
func (s *Service) invalidateGameCache(ctx context.Context, id uuid.UUID) {
	if s.cache == nil {
		return
	}
	cacheKey := cache.CacheKey("game", id.String())
	_ = s.cache.Delete(ctx, cacheKey)
}

// Bulk Operations

func (s *Service) BulkTransferPlayers(ctx context.Context, req BulkTransferRequest) error {
	for _, t := range req.Transfers {
		// 1. Handle Event Participation
		// If SourceEventID is provided, remove the player from the source team in that event
		if req.SourceEventID != uuid.Nil {
			participations, err := s.participationRepo.ListByPlayer(ctx, t.PlayerID)
			if err == nil {
				for _, ep := range participations {
					// If participation matches source event and source team, delete it
					if ep.Edges.Event.ID == req.SourceEventID && ep.Edges.Team.ID == t.FromTeamID {
						_ = s.participationRepo.Delete(ctx, ep.ID)
					}
				}
			}
		}

		// 2. Create/Update participation in target event
		// Check if player already has participation in target event
		var existingEP *ent.EventParticipation
		targetParticipations, _ := s.participationRepo.ListByPlayer(ctx, t.PlayerID)
		for _, ep := range targetParticipations {
			if ep.Edges.Event.ID == req.EventID {
				existingEP = ep
				break
			}
		}

		role := "player"
		if t.Role != nil {
			role = *t.Role
		}
		status := "active"
		if t.Status != nil {
			status = *t.Status
		}

		if existingEP != nil {
			// Update existing participation to new team
			// Note: Repository Update might not support changing team.
			// If not, we delete and recreate.
			_ = s.participationRepo.Delete(ctx, existingEP.ID)
		}

		// Create new participation
		_, err := s.participationRepo.Create(ctx, &ent.EventParticipation{
			Role:   role,
			Status: status,
			Edges: ent.EventParticipationEdges{
				Player: &ent.Player{ID: t.PlayerID},
				Team:   &ent.Team{ID: t.ToTeamID},
				Event:  &ent.Event{ID: req.EventID},
			},
		})
		if err != nil {
			return fmt.Errorf("failed to create participation for player %s: %w", t.PlayerID, err)
		}

		// 3. Update Player's global Team associations
		p, err := s.playerRepo.GetByID(ctx, t.PlayerID)
		if err == nil {
			// Update player's team list
			// In our repo, Update adds teams.
			// We should probably have a way to Remove teams too, but for now let's just ensure they are in the target team
			p.Edges.Teams = []*ent.Team{{ID: t.ToTeamID}}
			_, _ = s.playerRepo.Update(ctx, p)
		}
	}
	return nil
}

func (s *Service) MassImportPlayers(ctx context.Context, req MassImportPlayersRequest) ([]uuid.UUID, error) {
	var createdIDs []uuid.UUID
	for _, ip := range req.Players {
		// Create player
		p := &ent.Player{
			Name:         ip.Name,
			Email:        ip.Email,
			Gender:       ip.Gender,
			JerseyNumber: ip.JerseyNumber,
			Edges: ent.PlayerEdges{
				Teams: []*ent.Team{{ID: req.TeamID}},
			},
		}

		newPlayer, err := s.playerRepo.Create(ctx, p)
		if err != nil {
			return createdIDs, err
		}
		createdIDs = append(createdIDs, newPlayer.ID)

		// Create participation if EventID is provided
		if req.EventID != nil {
			_, _ = s.participationRepo.Create(ctx, &ent.EventParticipation{
				Role:            "player",
				Status:          "active",
				JerseyNumber:    ip.JerseyNumber,
				IsCaptain:       false,
				IsSpiritCaptain: false,
				Edges: ent.EventParticipationEdges{
					Player: &ent.Player{ID: newPlayer.ID},
					Team:   &ent.Team{ID: req.TeamID},
					Event:  &ent.Event{ID: *req.EventID},
				},
			})
		}
	}
	return createdIDs, nil
}

func (s *Service) DeleteGameRound(ctx context.Context, id uuid.UUID) error {
	// Check if exists
	_, err := s.gameRoundRepo.GetByID(ctx, id)
	if err != nil {
		return ErrGameRoundNotFound
	}

	return s.gameRoundRepo.Delete(ctx, id)
}

func (s *Service) ListAllDivisions(ctx context.Context) ([]*ent.DivisionPool, error) {
	return s.divisionRepo.ListAll(ctx)
}

func (s *Service) AutoEndExpiredGames(ctx context.Context) (int, error) {
	games, err := s.gameRepo.ListByStatus(ctx, "in_progress")
	if err != nil {
		return 0, err
	}

	endedCount := 0
	now := time.Now()

	for _, g := range games {
		if g.ActualStartTime == nil {
			continue
		}

		// Calculate expiration: StartTime + AllocatedMin + StoppageSec + 5min Grace
		allocatedDuration := time.Duration(g.AllocatedTimeMinutes) * time.Minute
		stoppageDuration := time.Duration(g.StoppageTimeSeconds) * time.Second
		gracePeriod := 5 * time.Minute

		expirationTime := g.ActualStartTime.Add(allocatedDuration).Add(stoppageDuration).Add(gracePeriod)

		if now.After(expirationTime) {
			// Auto-end the game
			_, err := s.endGameInternal(ctx, g.ID, "Game automatically ended by system (grace period expired)")
			if err != nil {
				// Log error but continue with other games
				continue
			}
			endedCount++
		}
	}

	return endedCount, nil
}

// endGameInternal is the core logic of EndGame without the auth check
func (s *Service) endGameInternal(ctx context.Context, id uuid.UUID, description string) (*GameDTO, error) {
	game, err := s.gameRepo.GetByIDWithRelations(ctx, id)
	if err != nil {
		return nil, err
	}

	if game.Status != "in_progress" {
		return nil, ErrInvalidGameStatus
	}

	updated, err := s.gameRepo.UpdateWithVersion(ctx, id, game.Version, func(update *ent.GameUpdateOne) *ent.GameUpdateOne {
		return update.SetStatus("ended")
	})
	if err != nil {
		return nil, err
	}

	// Create game ended event
	elapsed := time.Since(*game.ActualStartTime)
	minute := int(elapsed.Minutes())
	second := int(elapsed.Seconds()) % 60

	_, err = s.gameEventRepo.Create(ctx, &ent.GameEvent{
		EventType:   "game_ended",
		Minute:      minute,
		Second:      second,
		Description: description,
		Edges: ent.GameEventEdges{
			Game: updated,
		},
	})
	if err != nil {
		return nil, err
	}

	result, err := s.gameRepo.GetByIDWithRelations(ctx, updated.ID)
	if err != nil {
		return nil, err
	}

	s.invalidateGameCache(ctx, id)
	dto := mapGameToDTO(result)
	s.cacheGame(ctx, dto)
	return dto, nil
}

func (s *Service) ListDivisionsByEvent(ctx context.Context, eventID uuid.UUID) ([]*ent.DivisionPool, error) {
	return s.divisionRepo.ListByEvent(ctx, eventID)
}
