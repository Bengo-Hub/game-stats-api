package bracket

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/bengobox/game-stats-api/ent"
	"github.com/bengobox/game-stats-api/internal/infrastructure/cache"
	"github.com/google/uuid"
)

// Repository interfaces
type GameRepository interface {
	Create(ctx context.Context, game *ent.Game) (*ent.Game, error)
	GetByID(ctx context.Context, id uuid.UUID) (*ent.Game, error)
	Update(ctx context.Context, game *ent.Game) (*ent.Game, error)
	ListByRound(ctx context.Context, roundID uuid.UUID) ([]*ent.Game, error)
}

type GameRoundRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*ent.GameRound, error)
	Update(ctx context.Context, round *ent.GameRound) (*ent.GameRound, error)
}

type TeamRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*ent.Team, error)
}

type EventRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*ent.Event, error)
}

// Service handles bracket generation and management
type Service struct {
	gameRepo      GameRepository
	gameRoundRepo GameRoundRepository
	teamRepo      TeamRepository
	eventRepo     EventRepository
	cache         *cache.RedisClient
}

// NewService creates a new bracket service
func NewService(
	gameRepo GameRepository,
	gameRoundRepo GameRoundRepository,
	teamRepo TeamRepository,
	eventRepo EventRepository,
	cacheClient *cache.RedisClient,
) *Service {
	return &Service{
		gameRepo:      gameRepo,
		gameRoundRepo: gameRoundRepo,
		teamRepo:      teamRepo,
		eventRepo:     eventRepo,
		cache:         cacheClient,
	}
}

// GenerateBracket creates a tournament bracket structure and associated games
func (s *Service) GenerateBracket(ctx context.Context, req GenerateBracketRequest) (*GenerateBracketResponse, error) {
	// Validate event exists
	_, err := s.eventRepo.GetByID(ctx, req.EventID)
	if err != nil {
		return nil, fmt.Errorf("event not found: %w", err)
	}

	// Validate round exists
	round, err := s.gameRoundRepo.GetByID(ctx, req.RoundID)
	if err != nil {
		return nil, fmt.Errorf("round not found: %w", err)
	}

	// Validate round type is bracket
	if round.RoundType != "bracket" && round.RoundType != "semifinal" && round.RoundType != "final" {
		return nil, fmt.Errorf("round type must be bracket, semifinal, or final")
	}

	// Sort teams by seed
	teams := make([]TeamSeed, len(req.Teams))
	copy(teams, req.Teams)
	sort.Slice(teams, func(i, j int) bool {
		return teams[i].Seed < teams[j].Seed
	})

	// Validate number of teams is a power of 2 or adjust
	numTeams := len(teams)
	nextPowerOf2 := nextPowerOfTwo(numTeams)

	// If not a power of 2, add byes
	if numTeams != nextPowerOf2 {
		numByes := nextPowerOf2 - numTeams
		// Add bye teams (represented as nil)
		for i := 0; i < numByes; i++ {
			teams = append(teams, TeamSeed{
				Seed: numTeams + i + 1,
			})
		}
	}

	// Generate bracket based on type
	var bracketTree *BracketNode
	var gamesCreated []uuid.UUID

	switch req.BracketType {
	case BracketTypeSingleElimination:
		bracketTree, gamesCreated, err = s.generateSingleElimination(ctx, teams, req)
		if err != nil {
			return nil, fmt.Errorf("failed to generate single elimination bracket: %w", err)
		}
	case BracketTypeDoubleElimination:
		return nil, fmt.Errorf("double elimination not yet implemented")
	default:
		return nil, fmt.Errorf("unsupported bracket type: %s", req.BracketType)
	}

	// Calculate total rounds
	totalRounds := int(math.Log2(float64(len(teams))))

	return &GenerateBracketResponse{
		BracketID:    uuid.New(),
		EventID:      req.EventID,
		RoundID:      req.RoundID,
		BracketType:  req.BracketType,
		TotalRounds:  totalRounds,
		TotalGames:   len(gamesCreated),
		GamesCreated: gamesCreated,
		BracketTree:  bracketTree,
		CreatedAt:    time.Now(),
	}, nil
}

// generateSingleElimination creates a single elimination bracket
func (s *Service) generateSingleElimination(ctx context.Context, teams []TeamSeed, req GenerateBracketRequest) (*BracketNode, []uuid.UUID, error) {
	numTeams := len(teams)
	totalRounds := int(math.Log2(float64(numTeams)))

	// Generate matchups using standard seeding
	matchups := generateMatchups(teams, totalRounds)

	// Create games for first round
	gamesCreated := []uuid.UUID{}
	gameNodes := make(map[string]*BracketNode)

	currentTime := req.StartTime

	// Create first round games
	firstRoundMatchups := filterMatchupsByRound(matchups, 1)
	for _, matchup := range firstRoundMatchups {
		nodeKey := fmt.Sprintf("r%d-p%d", matchup.Round, matchup.Position)

		node := &BracketNode{
			ID:       uuid.New(),
			Round:    matchup.Round,
			Position: matchup.Position,
			Status:   "scheduled",
		}

		// Set team 1 (higher seed)
		if matchup.Team1ID != uuid.Nil {
			team1, err := s.teamRepo.GetByID(ctx, matchup.Team1ID)
			if err != nil {
				return nil, nil, fmt.Errorf("team1 not found: %w", err)
			}
			node.Team1ID = &matchup.Team1ID
			node.Team1Name = team1.Name
			node.Team1Seed = &matchup.Team1Seed
		} else {
			// Bye - team advances automatically
			node.Team1Name = "BYE"
		}

		// Set team 2 (lower seed)
		if matchup.Team2ID != uuid.Nil {
			team2, err := s.teamRepo.GetByID(ctx, matchup.Team2ID)
			if err != nil {
				return nil, nil, fmt.Errorf("team2 not found: %w", err)
			}
			node.Team2ID = &matchup.Team2ID
			node.Team2Name = team2.Name
			node.Team2Seed = &matchup.Team2Seed
		} else {
			// Bye - team1 advances automatically
			node.Team2Name = "BYE"
			if node.Team1ID != nil {
				node.WinnerID = node.Team1ID
				node.Status = "completed"
			}
		}

		// Create game only if both teams are real (no byes)
		if node.Team1ID != nil && node.Team2ID != nil {
			game, err := s.createBracketGame(ctx, req, *node.Team1ID, *node.Team2ID, currentTime)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to create game: %w", err)
			}

			node.GameID = &game.ID
			node.ScheduledTime = &currentTime
			gamesCreated = append(gamesCreated, game.ID)

			// Increment time for next game
			currentTime = currentTime.Add(time.Duration(req.GameDuration+15) * time.Minute)
		}

		gameNodes[nodeKey] = node
	}

	// Build bracket tree structure (placeholder nodes for future rounds)
	root := s.buildBracketTree(gameNodes, totalRounds)

	return root, gamesCreated, nil
}

// createBracketGame creates a game entity for a bracket match
func (s *Service) createBracketGame(ctx context.Context, req GenerateBracketRequest, team1ID, team2ID uuid.UUID, scheduledTime time.Time) (*ent.Game, error) {
	team1, err := s.teamRepo.GetByID(ctx, team1ID)
	if err != nil {
		return nil, err
	}

	team2, err := s.teamRepo.GetByID(ctx, team2ID)
	if err != nil {
		return nil, err
	}

	gameName := fmt.Sprintf("%s vs %s", team1.Name, team2.Name)

	game := &ent.Game{
		Name:                 gameName,
		ScheduledTime:        scheduledTime,
		AllocatedTimeMinutes: req.GameDuration,
		Status:               "scheduled",
		HomeTeamScore:        0,
		AwayTeamScore:        0,
		Version:              1,
	}

	// Note: In a real implementation, you'd use the Ent client's builder pattern
	// This is a simplified version
	createdGame, err := s.gameRepo.Create(ctx, game)
	if err != nil {
		return nil, fmt.Errorf("failed to create game: %w", err)
	}

	return createdGame, nil
}

// buildBracketTree constructs the tree structure from nodes
func (s *Service) buildBracketTree(nodes map[string]*BracketNode, totalRounds int) *BracketNode {
	// Start from the final (root)
	finalKey := fmt.Sprintf("r%d-p1", totalRounds)

	// Create placeholder for final if it doesn't exist
	if _, exists := nodes[finalKey]; !exists {
		nodes[finalKey] = &BracketNode{
			ID:        uuid.New(),
			Round:     totalRounds,
			Position:  1,
			Status:    "pending",
			Team1Name: "TBD",
			Team2Name: "TBD",
		}
	}

	root := nodes[finalKey]

	// Recursively attach children
	s.attachChildren(root, nodes)

	return root
}

// attachChildren recursively attaches child nodes
func (s *Service) attachChildren(node *BracketNode, nodes map[string]*BracketNode) {
	if node.Round == 1 {
		return // Leaf nodes have no children
	}

	previousRound := node.Round - 1
	leftPosition := (node.Position-1)*2 + 1
	rightPosition := (node.Position-1)*2 + 2

	leftKey := fmt.Sprintf("r%d-p%d", previousRound, leftPosition)
	rightKey := fmt.Sprintf("r%d-p%d", previousRound, rightPosition)

	// Create placeholder nodes if they don't exist
	if _, exists := nodes[leftKey]; !exists {
		nodes[leftKey] = &BracketNode{
			ID:        uuid.New(),
			Round:     previousRound,
			Position:  leftPosition,
			Status:    "pending",
			Team1Name: "TBD",
			Team2Name: "TBD",
		}
	}

	if _, exists := nodes[rightKey]; !exists {
		nodes[rightKey] = &BracketNode{
			ID:        uuid.New(),
			Round:     previousRound,
			Position:  rightPosition,
			Status:    "pending",
			Team1Name: "TBD",
			Team2Name: "TBD",
		}
	}

	node.LeftChildNode = nodes[leftKey]
	node.RightChildNode = nodes[rightKey]

	// Recursively attach to children
	s.attachChildren(node.LeftChildNode, nodes)
	s.attachChildren(node.RightChildNode, nodes)
}

// GetBracket retrieves the bracket structure for a round
func (s *Service) GetBracket(ctx context.Context, roundID uuid.UUID) (*GetBracketResponse, error) {
	// Try to get from cache first (if cache is available)
	cacheKey := cache.CacheKey("bracket", "round", roundID.String())
	if s.cache != nil {
		var cachedBracket GetBracketResponse
		err := s.cache.GetJSON(ctx, cacheKey, &cachedBracket)
		if err == nil {
			return &cachedBracket, nil
		}
	}

	// Validate round exists
	round, err := s.gameRoundRepo.GetByID(ctx, roundID)
	if err != nil {
		return nil, fmt.Errorf("round not found: %w", err)
	}

	// Get all games for this round
	games, err := s.gameRepo.ListByRound(ctx, roundID)
	if err != nil {
		return nil, fmt.Errorf("failed to get games: %w", err)
	}

	// Build bracket tree from games
	bracketTree := s.buildBracketFromGames(games)

	// Calculate total rounds
	totalRounds := calculateTotalRounds(len(games))

	response := &GetBracketResponse{
		EventID:     round.Edges.Event.ID,
		RoundID:     roundID,
		BracketType: BracketTypeSingleElimination,
		TotalRounds: totalRounds,
		TotalGames:  len(games),
		BracketTree: bracketTree,
		UpdatedAt:   time.Now(),
	}

	// Cache the result (if cache is available)
	if s.cache != nil {
		if err := s.cache.SetJSON(ctx, cacheKey, response, cache.TTLBracket); err != nil {
			// Log but don't fail - cache is non-critical
			fmt.Printf("Failed to cache bracket: %v\n", err)
		}
	}

	return response, nil
}

// buildBracketFromGames constructs bracket tree from existing games
// Uses game scheduled times to determine bracket structure
func (s *Service) buildBracketFromGames(games []*ent.Game) *BracketNode {
	if len(games) == 0 {
		return nil
	}

	// Sort games by scheduled time to determine rounds
	sort.Slice(games, func(i, j int) bool {
		return games[i].ScheduledTime.Before(games[j].ScheduledTime)
	})

	// Build nodes from games
	nodes := make([]*BracketNode, len(games))
	for i, game := range games {
		node := &BracketNode{
			ID:            uuid.New(),
			GameID:        &game.ID,
			Round:         i + 1, // Simplified - should calculate based on bracket structure
			Position:      i + 1,
			Status:        game.Status,
			ScheduledTime: &game.ScheduledTime,
		}

		// Set teams if available
		if game.Edges.HomeTeam != nil {
			node.Team1ID = &game.Edges.HomeTeam.ID
			node.Team1Name = game.Edges.HomeTeam.Name
		}

		if game.Edges.AwayTeam != nil {
			node.Team2ID = &game.Edges.AwayTeam.ID
			node.Team2Name = game.Edges.AwayTeam.Name
		}

		// Set scores
		if game.Status == "completed" {
			team1Score := game.HomeTeamScore
			team2Score := game.AwayTeamScore
			node.Team1Score = &team1Score
			node.Team2Score = &team2Score

			// Determine winner
			if team1Score > team2Score {
				node.WinnerID = node.Team1ID
			} else if team2Score > team1Score {
				node.WinnerID = node.Team2ID
			}
		}

		nodes[i] = node
	}

	// Return first node as root - tree structure built from game order
	if len(nodes) > 0 {
		return nodes[0]
	}

	return nil
}

// Helper functions

// nextPowerOfTwo returns the next power of 2 >= n
func nextPowerOfTwo(n int) int {
	if n <= 0 {
		return 1
	}
	power := 1
	for power < n {
		power *= 2
	}
	return power
}

// generateMatchups creates bracket matchups using standard seeding
func generateMatchups(teams []TeamSeed, totalRounds int) []Matchup {
	matchups := []Matchup{}
	numTeams := len(teams)

	// First round matchups using standard bracket seeding
	// 1 vs 16, 8 vs 9, 5 vs 12, 4 vs 13, 6 vs 11, 3 vs 14, 7 vs 10, 2 vs 15
	firstRoundGames := numTeams / 2

	for i := 0; i < firstRoundGames; i++ {
		team1Idx := i
		team2Idx := numTeams - 1 - i

		matchup := Matchup{
			Team1ID:   teams[team1Idx].TeamID,
			Team1Seed: teams[team1Idx].Seed,
			Team2ID:   teams[team2Idx].TeamID,
			Team2Seed: teams[team2Idx].Seed,
			Round:     1,
			Position:  i + 1,
		}

		matchups = append(matchups, matchup)
	}

	return matchups
}

// filterMatchupsByRound returns matchups for a specific round
func filterMatchupsByRound(matchups []Matchup, round int) []Matchup {
	filtered := []Matchup{}
	for _, m := range matchups {
		if m.Round == round {
			filtered = append(filtered, m)
		}
	}
	return filtered
}

// calculateTotalRounds calculates total rounds from number of games
func calculateTotalRounds(numGames int) int {
	if numGames == 0 {
		return 0
	}

	// For single elimination: total teams = numGames + 1
	totalTeams := numGames + 1
	return int(math.Ceil(math.Log2(float64(totalTeams))))
}
