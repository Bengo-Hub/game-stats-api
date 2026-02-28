package bracket

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"

	"entgo.io/ent/dialect/sql"
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
	client        *ent.Client
}

// NewService creates a new bracket service
func NewService(
	gameRepo GameRepository,
	gameRoundRepo GameRoundRepository,
	teamRepo TeamRepository,
	eventRepo EventRepository,
	cacheClient *cache.RedisClient,
	client *ent.Client,
) *Service {
	return &Service{
		gameRepo:      gameRepo,
		gameRoundRepo: gameRoundRepo,
		teamRepo:      teamRepo,
		eventRepo:     eventRepo,
		cache:         cacheClient,
		client:        client,
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

	// Validate division pool exists
	_, err = s.client.DivisionPool.Get(ctx, req.DivisionPoolID)
	if err != nil {
		return nil, fmt.Errorf("division pool not found: %w", err)
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

// GetEventRounds retrieves all game rounds for an event
func (s *Service) GetEventRounds(ctx context.Context, eventID uuid.UUID) ([]*ent.GameRound, error) {
	// Need to query via ent.Client since GameRoundRepository doesn't have ListByEvent
	return s.client.GameRound.Query().
		Where(func(s *sql.Selector) {
			s.Where(sql.In(
				s.C("id"),
				sql.Select("game_round_id").
					From(sql.Table("event_game_rounds")).
					Where(sql.EQ("event_id", eventID)),
			))
		}).
		All(ctx)
}

// generateSingleElimination creates a single elimination bracket
func (s *Service) generateSingleElimination(ctx context.Context, teams []TeamSeed, req GenerateBracketRequest) (*BracketNode, []uuid.UUID, error) {
	numTeams := len(teams)
	totalRounds := int(math.Log2(float64(numTeams)))

	// Generate matchups using standard seeding
	matchups := generateMatchups(teams, totalRounds)

	// Create games for all rounds
	gamesCreated := []uuid.UUID{}
	gameNodes := make(map[string]*BracketNode)

	currentTime := req.StartTime

	// Create nodes for all rounds
	for r := 1; r <= totalRounds; r++ {
		numGames := 1 << (totalRounds - r)

		for p := 1; p <= numGames; p++ {
			nodeKey := fmt.Sprintf("r%d-p%d", r, p)

			node := &BracketNode{
				ID:       uuid.New(),
				Round:    r,
				Position: p,
				Status:   "scheduled",
			}

			// Find team assignments for round 1
			if r == 1 {
				matchFound := false
				for _, m := range matchups {
					if m.Round == 1 && m.Position == p {
						matchFound = true
						if m.Team1ID != uuid.Nil {
							team1, err := s.teamRepo.GetByID(ctx, m.Team1ID)
							if err == nil {
								node.Team1ID = &m.Team1ID
								node.Team1Name = team1.Name
								node.Team1Seed = &m.Team1Seed
							}
						} else {
							node.Team1Name = "BYE"
						}

						if m.Team2ID != uuid.Nil {
							team2, err := s.teamRepo.GetByID(ctx, m.Team2ID)
							if err == nil {
								node.Team2ID = &m.Team2ID
								node.Team2Name = team2.Name
								node.Team2Seed = &m.Team2Seed
							}
						} else {
							node.Team2Name = "BYE"
							// Bye logic: if team1 exists, they win
							if node.Team1ID != nil {
								node.WinnerID = node.Team1ID
							}
						}
						break
					}
				}
				if !matchFound {
					node.Team1Name = "TBD"
					node.Team2Name = "TBD"
				}
			} else {
				node.Team1Name = "TBD"
				node.Team2Name = "TBD"
				node.Status = "pending"
			}

			// Create bracket game entity if not a bye-win already
			// For later rounds, we create them as "pending" or "scheduled" TBD games
			game, err := s.createBracketGameEntity(ctx, req, node, currentTime)
			if err == nil && game != nil {
				node.GameID = &game.ID
				node.ScheduledTime = &currentTime
				gamesCreated = append(gamesCreated, game.ID)

				// Increment time for next game
				currentTime = currentTime.Add(time.Duration(req.GameDuration+15) * time.Minute)
			}

			gameNodes[nodeKey] = node
		}
	}

	// Build bracket tree structure
	root := s.buildBracketTree(gameNodes, totalRounds)

	return root, gamesCreated, nil
}

// createBracketGameEntity creates a game record for any round
func (s *Service) createBracketGameEntity(ctx context.Context, req GenerateBracketRequest, node *BracketNode, scheduledTime time.Time) (*ent.Game, error) {
	gameName := fmt.Sprintf("Round %d, Match %d", node.Round, node.Position)
	if node.Team1Name != "" && node.Team2Name != "" && node.Team1Name != "TBD" && node.Team2Name != "TBD" {
		gameName = fmt.Sprintf("%s vs %s", node.Team1Name, node.Team2Name)
	}

	game := &ent.Game{
		Name:                 gameName,
		ScheduledTime:        scheduledTime,
		AllocatedTimeMinutes: req.GameDuration,
		Status:               node.Status,
		HomeTeamScore:        0,
		AwayTeamScore:        0,
		Version:              1,
		Edges: ent.GameEdges{
			DivisionPool:  &ent.DivisionPool{ID: req.DivisionPoolID},
			FieldLocation: &ent.Field{ID: req.FieldID},
			GameRound:     &ent.GameRound{ID: req.RoundID},
		},
	}

	if node.Team1ID != nil {
		game.Edges.HomeTeam = &ent.Team{ID: *node.Team1ID}
	}
	if node.Team2ID != nil {
		game.Edges.AwayTeam = &ent.Team{ID: *node.Team2ID}
	}

	// Skip if it's a bye
	if node.Team1Name == "BYE" || node.Team2Name == "BYE" {
		return nil, nil
	}

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

	var eventID uuid.UUID
	if len(round.Edges.Events) > 0 {
		eventID = round.Edges.Events[0].ID
	}

	response := &GetBracketResponse{
		EventID:     eventID,
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

// GetEventBracketAll retrieves a combined bracket structure for an entire event
func (s *Service) GetEventBracketAll(ctx context.Context, eventID uuid.UUID) (*GetBracketResponse, error) {
	// 1. Get all rounds for the event
	rounds, err := s.GetEventRounds(ctx, eventID)
	if err != nil {
		return nil, fmt.Errorf("failed to get event rounds: %w", err)
	}

	// 2. Filter for bracket-type rounds and collect their IDs
	var roundIDs []uuid.UUID
	for _, r := range rounds {
		if r.RoundType == "bracket" || r.RoundType == "semifinal" || r.RoundType == "final" {
			roundIDs = append(roundIDs, r.ID)
		}
	}

	if len(roundIDs) == 0 {
		return nil, fmt.Errorf("no bracket rounds found for event")
	}

	// 3. Get all games for all bracket rounds
	var allGames []*ent.Game
	for _, rid := range roundIDs {
		games, err := s.gameRepo.ListByRound(ctx, rid)
		if err == nil && len(games) > 0 {
			allGames = append(allGames, games...)
		}
	}

	if len(allGames) == 0 {
		return &GetBracketResponse{
			EventID:     eventID,
			BracketType: BracketTypeSingleElimination,
			TotalRounds: 0,
			TotalGames:  0,
			UpdatedAt:   time.Now(),
		}, nil
	}

	// 4. Build single bracket tree from all games
	bracketTree := s.buildBracketFromGames(allGames)
	totalRounds := calculateTotalRounds(len(allGames))

	return &GetBracketResponse{
		EventID:     eventID,
		BracketType: BracketTypeSingleElimination,
		TotalRounds: totalRounds,
		TotalGames:  len(allGames),
		BracketTree: bracketTree,
		UpdatedAt:   time.Now(),
	}, nil
}

// buildBracketFromGames constructs bracket tree from existing games
func (s *Service) buildBracketFromGames(games []*ent.Game) *BracketNode {
	if len(games) == 0 {
		return nil
	}

	totalRounds := calculateTotalRounds(len(games))
	nodes := make(map[string]*BracketNode)

	// Build all nodes from games first
	// We need to determine the round and position for each game.
	// In a single elimination bracket with N games (N+1 teams),
	// round R has 2^(totalRounds-R) games.
	// We'll sort by scheduled time and assign rounds/positions based on a standard progression.
	sort.Slice(games, func(i, j int) bool {
		return games[i].ScheduledTime.Before(games[j].ScheduledTime)
	})

	// Assign games to rounds based on power-of-2 counts
	remainingGames := games
	for r := 1; r <= totalRounds; r++ {
		gamesInRound := 1 << (totalRounds - r)
		if gamesInRound > len(remainingGames) {
			gamesInRound = len(remainingGames)
		}

		roundGames := remainingGames[:gamesInRound]
		remainingGames = remainingGames[gamesInRound:]

		for p, game := range roundGames {
			node := &BracketNode{
				ID:            uuid.New(),
				GameID:        &game.ID,
				Round:         r,
				Position:      p + 1,
				Status:        game.Status,
				ScheduledTime: &game.ScheduledTime,
			}

			if game.Edges.HomeTeam != nil {
				node.Team1ID = &game.Edges.HomeTeam.ID
				node.Team1Name = game.Edges.HomeTeam.Name
			}
			if game.Edges.AwayTeam != nil {
				node.Team2ID = &game.Edges.AwayTeam.ID
				node.Team2Name = game.Edges.AwayTeam.Name
			}

			if game.Status == "completed" {
				t1, t2 := game.HomeTeamScore, game.AwayTeamScore
				node.Team1Score, node.Team2Score = &t1, &t2
				if t1 > t2 {
					node.WinnerID = node.Team1ID
				} else if t2 > t1 {
					node.WinnerID = node.Team2ID
				}
			}

			key := fmt.Sprintf("r%d-p%d", r, p+1)
			nodes[key] = node
		}
	}

	// Connect the nodes into a tree
	return s.buildBracketTree(nodes, totalRounds)
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
