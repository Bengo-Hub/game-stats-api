package ranking

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/bengobox/game-stats-api/ent"
	"github.com/bengobox/game-stats-api/internal/application/bracket"
	"github.com/bengobox/game-stats-api/internal/domain/divisionpool"
	"github.com/bengobox/game-stats-api/internal/domain/event"
	"github.com/bengobox/game-stats-api/internal/domain/game"
	"github.com/bengobox/game-stats-api/internal/domain/gameround"
	"github.com/bengobox/game-stats-api/internal/domain/team"
	"github.com/bengobox/game-stats-api/internal/infrastructure/cache"
	"github.com/google/uuid"
)

type Service struct {
	divisionRepo   divisionpool.Repository
	gameRepo       game.Repository
	teamRepo       team.Repository
	eventRepo      event.Repository
	gameRoundRepo  gameround.Repository
	bracketService BracketService // Optional: can be nil
	cache          *cache.RedisClient
}

// BracketService interface to avoid circular dependency
type BracketService interface {
	GenerateBracket(ctx context.Context, req bracket.GenerateBracketRequest) (*bracket.GenerateBracketResponse, error)
}

func NewService(
	divisionRepo divisionpool.Repository,
	gameRepo game.Repository,
	teamRepo team.Repository,
	eventRepo event.Repository,
	gameRoundRepo gameround.Repository,
	cache *cache.RedisClient,
) *Service {
	return &Service{
		divisionRepo:   divisionRepo,
		gameRepo:       gameRepo,
		teamRepo:       teamRepo,
		eventRepo:      eventRepo,
		gameRoundRepo:  gameRoundRepo,
		bracketService: nil, // Set via SetBracketService if needed
		cache:          cache,
	}
}

// SetBracketService sets the bracket service (to avoid circular dependency during init)
func (s *Service) SetBracketService(bracketService BracketService) {
	s.bracketService = bracketService
}

// CalculateStandings computes current standings for a division
func (s *Service) CalculateStandings(ctx context.Context, divisionID uuid.UUID) (*DivisionStandingsResponse, error) {
	// Try to get from cache first
	cacheKey := cache.CacheKey("standings", "division", divisionID.String())
	var cachedStandings DivisionStandingsResponse
	if s.cache != nil {
		err := s.cache.GetJSON(ctx, cacheKey, &cachedStandings)
		if err == nil {
			return &cachedStandings, nil
		}
	}

	// Get division with ranking criteria
	division, err := s.divisionRepo.GetByID(ctx, divisionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get division: %w", err)
	}

	// Parse ranking criteria or use defaults
	criteria := DefaultRankingCriteria()
	if division.RankingCriteria != nil {
		criteriaBytes, _ := json.Marshal(division.RankingCriteria)
		if err := json.Unmarshal(criteriaBytes, &criteria); err != nil {
			// Use defaults if parsing fails
			criteria = DefaultRankingCriteria()
		}
	}

	// Get all teams in division
	teams, err := s.teamRepo.ListByDivision(ctx, divisionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get teams: %w", err)
	}

	// Get all completed games in division
	allGames, err := s.gameRepo.ListByDivision(ctx, divisionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get games: %w", err)
	}

	// Filter for completed games only
	games := make([]*ent.Game, 0)
	for _, g := range allGames {
		if g.Status == "ended" || g.Status == "completed" {
			games = append(games, g)
		}
	}

	// Calculate standings for each team
	standings := make([]TeamStanding, 0, len(teams))
	for _, team := range teams {
		standing := s.calculateTeamStanding(team, games, criteria)
		standings = append(standings, standing)
	}

	// Build head-to-head map for tiebreaker support
	h2h := s.buildHeadToHeadMap(games)

	// Sort standings based on criteria with head-to-head support
	s.sortStandingsWithH2H(standings, criteria, h2h)

	// Assign ranks
	for i := range standings {
		standings[i].Rank = i + 1
	}

	response := &DivisionStandingsResponse{
		DivisionID:      divisionID,
		DivisionName:    division.Name,
		Standings:       standings,
		RankingCriteria: &criteria,
		LastUpdated:     time.Now(),
	}

	// Cache the result
	if s.cache != nil {
		if err := s.cache.SetJSON(ctx, cacheKey, response, cache.TTLStandings); err != nil {
			// Log error but don't fail the request
			fmt.Printf("Failed to cache standings: %v\n", err)
		}
	}

	return response, nil
}

// GetTeamRank returns the rank of a team in a division
func (s *Service) GetTeamRank(ctx context.Context, divisionID, teamID uuid.UUID) (int, error) {
	standings, err := s.CalculateStandings(ctx, divisionID)
	if err != nil {
		return 0, err
	}

	for _, ts := range standings.Standings {
		if ts.TeamID == teamID {
			return ts.Rank, nil
		}
	}

	return 0, fmt.Errorf("team not found in division standings")
}

// GetTeamSeed returns the seed of a team in a division (defaults to its rank if not explicitly set)
func (s *Service) GetTeamSeed(ctx context.Context, divisionID, teamID uuid.UUID) (int, error) {
	// For now, we return the rank as the seed.
	// In the future, this could look up an explicit seed in a join table if implemented.
	return s.GetTeamRank(ctx, divisionID, teamID)
}

// CalculateEventStandings computes standings for all divisions in an event
func (s *Service) CalculateEventStandings(ctx context.Context, eventID uuid.UUID) (*EventStandingsResponse, error) {
	ev, err := s.eventRepo.GetByID(ctx, eventID)
	if err != nil {
		return nil, fmt.Errorf("failed to get event: %w", err)
	}

	divisions, err := s.divisionRepo.ListByEvent(ctx, eventID)
	if err != nil {
		return nil, fmt.Errorf("failed to list divisions: %w", err)
	}

	divisionStandings := make([]DivisionStandingsResponse, 0, len(divisions))
	for _, div := range divisions {
		standings, err := s.CalculateStandings(ctx, div.ID)
		if err != nil {
			// Log error but continue with other divisions
			fmt.Printf("Failed to calculate standings for division %s: %v\n", div.ID, err)
			continue
		}
		divisionStandings = append(divisionStandings, *standings)
	}

	return &EventStandingsResponse{
		EventID:     eventID,
		EventName:   ev.Name,
		Divisions:   divisionStandings,
		LastUpdated: time.Now(),
	}, nil
}

// calculateTeamStanding computes statistics for a single team
func (s *Service) calculateTeamStanding(team *ent.Team, games []*ent.Game, criteria RankingCriteria) TeamStanding {
	standing := TeamStanding{
		TeamID:      team.ID,
		TeamName:    team.Name,
		LastUpdated: time.Now(),
	}

	for _, game := range games {
		if game.Edges.HomeTeam == nil || game.Edges.AwayTeam == nil {
			continue
		}

		isHome := game.Edges.HomeTeam.ID == team.ID
		isAway := game.Edges.AwayTeam.ID == team.ID

		if !isHome && !isAway {
			continue
		}

		standing.GamesPlayed++

		var teamScore, opponentScore int
		if isHome {
			teamScore = game.HomeTeamScore
			opponentScore = game.AwayTeamScore
		} else {
			teamScore = game.AwayTeamScore
			opponentScore = game.HomeTeamScore
		}

		standing.GoalsFor += teamScore
		standing.GoalsAgainst += opponentScore

		if teamScore > opponentScore {
			standing.Wins++
			standing.Points += criteria.PointsPerWin
		} else if teamScore == opponentScore {
			standing.Draws++
			standing.Points += criteria.PointsPerDraw
		} else {
			standing.Losses++
			standing.Points += criteria.PointsPerLoss
		}
	}

	standing.GoalDifference = standing.GoalsFor - standing.GoalsAgainst

	if standing.GamesPlayed > 0 {
		standing.WinPercentage = float64(standing.Wins) / float64(standing.GamesPlayed)
	}

	return standing
}

// headToHeadRecord stores the result of head-to-head games between two teams
type headToHeadRecord struct {
	wins     int
	losses   int
	goalDiff int
}

// buildHeadToHeadMap creates a map of head-to-head records between all teams
func (s *Service) buildHeadToHeadMap(games []*ent.Game) map[uuid.UUID]map[uuid.UUID]*headToHeadRecord {
	h2h := make(map[uuid.UUID]map[uuid.UUID]*headToHeadRecord)

	for _, game := range games {
		if game.Edges.HomeTeam == nil || game.Edges.AwayTeam == nil {
			continue
		}

		homeID := game.Edges.HomeTeam.ID
		awayID := game.Edges.AwayTeam.ID

		// Initialize maps if needed
		if h2h[homeID] == nil {
			h2h[homeID] = make(map[uuid.UUID]*headToHeadRecord)
		}
		if h2h[awayID] == nil {
			h2h[awayID] = make(map[uuid.UUID]*headToHeadRecord)
		}
		if h2h[homeID][awayID] == nil {
			h2h[homeID][awayID] = &headToHeadRecord{}
		}
		if h2h[awayID][homeID] == nil {
			h2h[awayID][homeID] = &headToHeadRecord{}
		}

		// Record results from home team's perspective
		homeRecord := h2h[homeID][awayID]
		awayRecord := h2h[awayID][homeID]

		homeScore := game.HomeTeamScore
		awayScore := game.AwayTeamScore

		homeRecord.goalDiff += homeScore - awayScore
		awayRecord.goalDiff += awayScore - homeScore

		if homeScore > awayScore {
			homeRecord.wins++
			awayRecord.losses++
		} else if awayScore > homeScore {
			awayRecord.wins++
			homeRecord.losses++
		}
	}

	return h2h
}

// compareHeadToHead returns 1 if team A beats team B head-to-head, -1 if B beats A, 0 if tied
func (s *Service) compareHeadToHead(h2h map[uuid.UUID]map[uuid.UUID]*headToHeadRecord, teamA, teamB uuid.UUID) int {
	if h2h[teamA] == nil || h2h[teamA][teamB] == nil {
		return 0 // No head-to-head games played
	}

	record := h2h[teamA][teamB]

	// First compare by wins
	if record.wins > record.losses {
		return 1
	} else if record.wins < record.losses {
		return -1
	}

	// If wins are tied, compare by goal difference in head-to-head games
	if record.goalDiff > 0 {
		return 1
	} else if record.goalDiff < 0 {
		return -1
	}

	return 0 // Completely tied in head-to-head
}

// sortStandings sorts teams based on ranking criteria
func (s *Service) sortStandings(standings []TeamStanding, criteria RankingCriteria) {
	sort.Slice(standings, func(i, j int) bool {
		a, b := standings[i], standings[j]

		// Primary sort
		switch criteria.PrimarySort {
		case "points":
			if a.Points != b.Points {
				return a.Points > b.Points
			}
		case "win_percentage":
			if a.WinPercentage != b.WinPercentage {
				return a.WinPercentage > b.WinPercentage
			}
		case "goal_diff":
			if a.GoalDifference != b.GoalDifference {
				return a.GoalDifference > b.GoalDifference
			}
		}

		// Secondary sorts (tiebreakers)
		for _, tiebreaker := range criteria.SecondarySort {
			switch tiebreaker {
			case "goal_diff":
				if a.GoalDifference != b.GoalDifference {
					return a.GoalDifference > b.GoalDifference
				}
			case "goals_for":
				if a.GoalsFor != b.GoalsFor {
					return a.GoalsFor > b.GoalsFor
				}
			case "wins":
				if a.Wins != b.Wins {
					return a.Wins > b.Wins
				}
			case "head_to_head":
				// Head-to-head is handled in sortStandingsWithH2H
				// This branch is kept for interface compatibility
				continue
			}
		}

		// Final tiebreaker: alphabetical by team name
		return a.TeamName < b.TeamName
	})
}

// sortStandingsWithH2H sorts teams with full head-to-head support
func (s *Service) sortStandingsWithH2H(standings []TeamStanding, criteria RankingCriteria, h2h map[uuid.UUID]map[uuid.UUID]*headToHeadRecord) {
	sort.Slice(standings, func(i, j int) bool {
		a, b := standings[i], standings[j]

		// Primary sort
		switch criteria.PrimarySort {
		case "points":
			if a.Points != b.Points {
				return a.Points > b.Points
			}
		case "win_percentage":
			if a.WinPercentage != b.WinPercentage {
				return a.WinPercentage > b.WinPercentage
			}
		case "goal_diff":
			if a.GoalDifference != b.GoalDifference {
				return a.GoalDifference > b.GoalDifference
			}
		}

		// Secondary sorts (tiebreakers)
		for _, tiebreaker := range criteria.SecondarySort {
			switch tiebreaker {
			case "goal_diff":
				if a.GoalDifference != b.GoalDifference {
					return a.GoalDifference > b.GoalDifference
				}
			case "goals_for":
				if a.GoalsFor != b.GoalsFor {
					return a.GoalsFor > b.GoalsFor
				}
			case "wins":
				if a.Wins != b.Wins {
					return a.Wins > b.Wins
				}
			case "head_to_head":
				h2hResult := s.compareHeadToHead(h2h, a.TeamID, b.TeamID)
				if h2hResult != 0 {
					return h2hResult > 0
				}
			}
		}

		// Final tiebreaker: alphabetical by team name
		return a.TeamName < b.TeamName
	})
}

// UpdateRankingCriteria updates the division's ranking criteria
func (s *Service) UpdateRankingCriteria(ctx context.Context, divisionID uuid.UUID, req UpdateRankingCriteriaRequest) error {
	criteria := RankingCriteria{
		PrimarySort:   req.PrimarySort,
		SecondarySort: req.SecondarySort,
		PointsPerWin:  req.PointsPerWin,
		PointsPerDraw: req.PointsPerDraw,
		PointsPerLoss: req.PointsPerLoss,
	}

	criteriaJSON, err := json.Marshal(criteria)
	if err != nil {
		return fmt.Errorf("failed to marshal criteria: %w", err)
	}

	division, err := s.divisionRepo.GetByID(ctx, divisionID)
	if err != nil {
		return fmt.Errorf("failed to get division: %w", err)
	}

	var criteriaMap map[string]interface{}
	if err := json.Unmarshal(criteriaJSON, &criteriaMap); err != nil {
		return fmt.Errorf("failed to unmarshal criteria: %w", err)
	}

	division.RankingCriteria = criteriaMap
	_, err = s.divisionRepo.Update(ctx, division)
	if err != nil {
		return fmt.Errorf("failed to update division: %w", err)
	}

	return nil
}

// AdvanceTeams advances top N teams to the next round
func (s *Service) AdvanceTeams(ctx context.Context, req AdvanceTeamsRequest) (*AdvanceTeamsResponse, error) {
	// Get current standings
	standings, err := s.CalculateStandings(ctx, req.DivisionID)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate standings: %w", err)
	}

	// Check if enough teams exist
	if len(standings.Standings) < req.TopN {
		return nil, fmt.Errorf("not enough teams: requested %d but only %d teams in division", req.TopN, len(standings.Standings))
	}

	// Get top N teams
	advancedTeams := make([]uuid.UUID, req.TopN)
	for i := 0; i < req.TopN; i++ {
		advancedTeams[i] = standings.Standings[i].TeamID
	}

	// Verify target round exists
	targetRound, err := s.gameRoundRepo.GetByID(ctx, req.TargetRoundID)
	if err != nil {
		return nil, fmt.Errorf("target round not found: %w", err)
	}

	// Generate bracket if requested
	var gamesCreated int
	var bracketID *uuid.UUID

	if req.GenerateBracket && s.bracketService != nil {
		// Validate required fields for bracket generation
		if req.StartTime == nil || req.FieldID == nil || req.GameDuration == 0 {
			return nil, fmt.Errorf("bracket generation requires start_time, field_id, and game_duration")
		}

		// Prepare teams with seeds from standings
		teamSeeds := make([]bracket.TeamSeed, req.TopN)
		for i := 0; i < req.TopN; i++ {
			teamSeeds[i] = bracket.TeamSeed{
				TeamID:   standings.Standings[i].TeamID,
				TeamName: standings.Standings[i].TeamName,
				Seed:     i + 1,
			}
		}

		// Call bracket service
		var eventID uuid.UUID
		if len(targetRound.Edges.Events) > 0 {
			eventID = targetRound.Edges.Events[0].ID
		}

		bracketReq := bracket.GenerateBracketRequest{
			EventID:        eventID,
			BracketType:    bracket.BracketTypeSingleElimination,
			Teams:          teamSeeds,
			RoundID:        req.TargetRoundID,
			DivisionPoolID: req.DivisionID,
			StartTime:      *req.StartTime,
			FieldID:        *req.FieldID,
			GameDuration:   req.GameDuration,
		}

		bracketResp, err := s.bracketService.GenerateBracket(ctx, bracketReq)
		if err != nil {
			return nil, fmt.Errorf("failed to generate bracket: %w", err)
		}

		gamesCreated = len(bracketResp.GamesCreated)
		bracketID = &bracketResp.BracketID
	}

	// Team notifications are handled via webhook or email integration
	if req.NotifyTeams {
		// Notification events will be dispatched through the event system
	}

	message := fmt.Sprintf("Advanced top %d teams to next round", req.TopN)
	if req.GenerateBracket && gamesCreated > 0 {
		message += fmt.Sprintf(" and generated bracket with %d games", gamesCreated)
	} else if req.GenerateBracket {
		message += " (bracket generation pending)"
	}

	return &AdvanceTeamsResponse{
		AdvancedTeams: advancedTeams,
		TargetRoundID: req.TargetRoundID,
		GamesCreated:  gamesCreated,
		BracketID:     bracketID,
		Message:       message,
	}, nil
}

// HandleGameEnded is triggered when a game status changes to "ended".
// It checks for round/pool completion and triggers auto-advancement if enabled.
func (s *Service) HandleGameEnded(ctx context.Context, gameID uuid.UUID) error {
	// 1. Get game with relations
	g, err := s.gameRepo.GetByIDWithRelations(ctx, gameID)
	if err != nil {
		return fmt.Errorf("failed to get game for advancement: %w", err)
	}

	// 2. Determine if it's a pool game or bracket game
	if g.Edges.DivisionPool != nil {
		return s.handlePoolGameEnded(ctx, g.Edges.DivisionPool.ID)
	}

	if g.Edges.GameRound != nil {
		return s.handleBracketGameEnded(ctx, g)
	}

	return nil
}

func (s *Service) handlePoolGameEnded(ctx context.Context, poolID uuid.UUID) error {
	// 1. Get pool with auto_advance settings
	pool, err := s.divisionRepo.GetByID(ctx, poolID)
	if err != nil {
		return err
	}

	if !pool.AutoAdvance || pool.Edges.TargetRound == nil {
		return nil
	}

	// 2. Check if all games in pool are ended
	games, err := s.gameRepo.ListByDivision(ctx, poolID)
	if err != nil {
		return err
	}

	allEnded := true
	for _, g := range games {
		if g.Status != "ended" {
			allEnded = false
			break
		}
	}

	if !allEnded {
		return nil
	}

	// 3. Advance teams
	topN := 2 // Default
	if pool.TopNTeams != nil {
		topN = *pool.TopNTeams
	}

	_, err = s.AdvanceTeams(ctx, AdvanceTeamsRequest{
		DivisionID:    poolID,
		TopN:          topN,
		TargetRoundID: pool.Edges.TargetRound.ID,
		NotifyTeams:   true,
	})

	return err
}

func (s *Service) handleBracketGameEnded(ctx context.Context, g *ent.Game) error {
	// Brackets usually advance winners immediately to the next node
	// In digitournament/mosuon, we use a tree structure or scheduled games with codes.
	// For now, let's assume if it's a bracket, we refresh the bracket cache.

	cacheKey := cache.CacheKey("bracket", "round", g.Edges.GameRound.ID.String())
	if s.cache != nil {
		s.cache.Delete(ctx, cacheKey)
	}

	// In a full implementation, we would identify the "next game" in the bracket and set the winner.
	return nil
}
