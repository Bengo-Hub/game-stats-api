package bracket

import (
	"context"
	"testing"
	"time"

	"github.com/bengobox/game-stats-api/ent"
	"github.com/bengobox/game-stats-api/internal/domain/game"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock repositories
type MockGameRepository struct {
	mock.Mock
}

func (m *MockGameRepository) Create(ctx context.Context, game *ent.Game) (*ent.Game, error) {
	args := m.Called(ctx, game)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ent.Game), args.Error(1)
}

func (m *MockGameRepository) GetByID(ctx context.Context, id uuid.UUID) (*ent.Game, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ent.Game), args.Error(1)
}

func (m *MockGameRepository) Update(ctx context.Context, game *ent.Game) (*ent.Game, error) {
	args := m.Called(ctx, game)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ent.Game), args.Error(1)
}

func (m *MockGameRepository) ListByRound(ctx context.Context, roundID uuid.UUID) ([]*ent.Game, error) {
	args := m.Called(ctx, roundID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*ent.Game), args.Error(1)
}

func (m *MockGameRepository) ListWithFilter(ctx context.Context, filter game.SearchFilter) ([]*ent.Game, error) {
	return nil, nil
}

type MockGameRoundRepository struct {
	mock.Mock
}

func (m *MockGameRoundRepository) GetByID(ctx context.Context, id uuid.UUID) (*ent.GameRound, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ent.GameRound), args.Error(1)
}

func (m *MockGameRoundRepository) Update(ctx context.Context, round *ent.GameRound) (*ent.GameRound, error) {
	args := m.Called(ctx, round)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ent.GameRound), args.Error(1)
}

type MockTeamRepository struct {
	mock.Mock
}

func (m *MockTeamRepository) GetByID(ctx context.Context, id uuid.UUID) (*ent.Team, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ent.Team), args.Error(1)
}

type MockEventRepository struct {
	mock.Mock
}

func (m *MockEventRepository) GetByID(ctx context.Context, id uuid.UUID) (*ent.Event, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ent.Event), args.Error(1)
}

func TestNextPowerOfTwo(t *testing.T) {
	tests := []struct {
		name     string
		input    int
		expected int
	}{
		{"Zero", 0, 1},
		{"One", 1, 1},
		{"Two", 2, 2},
		{"Three", 3, 4},
		{"Four", 4, 4},
		{"Five", 5, 8},
		{"Seven", 7, 8},
		{"Eight", 8, 8},
		{"Nine", 9, 16},
		{"Fifteen", 15, 16},
		{"Sixteen", 16, 16},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := nextPowerOfTwo(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGenerateMatchups(t *testing.T) {
	tests := []struct {
		name        string
		teams       []TeamSeed
		totalRounds int
		expected    int // expected number of first round matchups
	}{
		{
			name: "4 teams",
			teams: []TeamSeed{
				{TeamID: uuid.New(), TeamName: "Team 1", Seed: 1},
				{TeamID: uuid.New(), TeamName: "Team 2", Seed: 2},
				{TeamID: uuid.New(), TeamName: "Team 3", Seed: 3},
				{TeamID: uuid.New(), TeamName: "Team 4", Seed: 4},
			},
			totalRounds: 2,
			expected:    2,
		},
		{
			name: "8 teams",
			teams: []TeamSeed{
				{TeamID: uuid.New(), TeamName: "Team 1", Seed: 1},
				{TeamID: uuid.New(), TeamName: "Team 2", Seed: 2},
				{TeamID: uuid.New(), TeamName: "Team 3", Seed: 3},
				{TeamID: uuid.New(), TeamName: "Team 4", Seed: 4},
				{TeamID: uuid.New(), TeamName: "Team 5", Seed: 5},
				{TeamID: uuid.New(), TeamName: "Team 6", Seed: 6},
				{TeamID: uuid.New(), TeamName: "Team 7", Seed: 7},
				{TeamID: uuid.New(), TeamName: "Team 8", Seed: 8},
			},
			totalRounds: 3,
			expected:    4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matchups := generateMatchups(tt.teams, tt.totalRounds)
			assert.Equal(t, tt.expected, len(matchups))

			// Verify standard seeding (1 vs last, 2 vs second-to-last, etc.)
			if len(matchups) > 0 {
				assert.Equal(t, 1, matchups[0].Team1Seed)
				assert.Equal(t, len(tt.teams), matchups[0].Team2Seed)
			}
		})
	}
}

func TestGenerateBracket_Success(t *testing.T) {
	// Setup
	mockGameRepo := new(MockGameRepository)
	mockRoundRepo := new(MockGameRoundRepository)
	mockTeamRepo := new(MockTeamRepository)
	mockEventRepo := new(MockEventRepository)

	service := NewService(mockGameRepo, mockRoundRepo, mockTeamRepo, mockEventRepo, nil)

	eventID := uuid.New()
	roundID := uuid.New()
	fieldID := uuid.New()

	teams := []TeamSeed{
		{TeamID: uuid.New(), TeamName: "Team Alpha", Seed: 1},
		{TeamID: uuid.New(), TeamName: "Team Beta", Seed: 2},
		{TeamID: uuid.New(), TeamName: "Team Gamma", Seed: 3},
		{TeamID: uuid.New(), TeamName: "Team Delta", Seed: 4},
	}

	request := GenerateBracketRequest{
		EventID:      eventID,
		BracketType:  BracketTypeSingleElimination,
		Teams:        teams,
		RoundID:      roundID,
		StartTime:    time.Now(),
		FieldID:      fieldID,
		GameDuration: 60,
	}

	// Setup mocks
	mockEventRepo.On("GetByID", mock.Anything, eventID).Return(&ent.Event{
		ID:   eventID,
		Name: "Test Tournament",
	}, nil)

	mockRoundRepo.On("GetByID", mock.Anything, roundID).Return(&ent.GameRound{
		ID:        roundID,
		Name:      "Bracket Round",
		RoundType: "bracket",
		Edges: ent.GameRoundEdges{
			Event: &ent.Event{ID: eventID},
		},
	}, nil)

	// Mock team lookups
	for _, team := range teams {
		mockTeamRepo.On("GetByID", mock.Anything, team.TeamID).Return(&ent.Team{
			ID:   team.TeamID,
			Name: team.TeamName,
		}, nil)
	}

	// Mock game creation (2 games for 4 teams)
	mockGameRepo.On("Create", mock.Anything, mock.Anything).Return(&ent.Game{
		ID:     uuid.New(),
		Name:   "Test Game",
		Status: "scheduled",
	}, nil)

	// Execute
	ctx := context.Background()
	response, err := service.GenerateBracket(ctx, request)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, eventID, response.EventID)
	assert.Equal(t, roundID, response.RoundID)
	assert.Equal(t, BracketTypeSingleElimination, response.BracketType)
	assert.Equal(t, 2, response.TotalRounds) // 4 teams = 2 rounds
	assert.Equal(t, 2, response.TotalGames)  // 2 first-round games
	assert.NotNil(t, response.BracketTree)

	mockEventRepo.AssertExpectations(t)
	mockRoundRepo.AssertExpectations(t)
	mockTeamRepo.AssertExpectations(t)
}

func TestGenerateBracket_NonPowerOfTwo(t *testing.T) {
	// Setup
	mockGameRepo := new(MockGameRepository)
	mockRoundRepo := new(MockGameRoundRepository)
	mockTeamRepo := new(MockTeamRepository)
	mockEventRepo := new(MockEventRepository)

	service := NewService(mockGameRepo, mockRoundRepo, mockTeamRepo, mockEventRepo, nil)

	eventID := uuid.New()
	roundID := uuid.New()
	fieldID := uuid.New()

	// 5 teams - should be padded to 8 with byes
	teams := []TeamSeed{
		{TeamID: uuid.New(), TeamName: "Team 1", Seed: 1},
		{TeamID: uuid.New(), TeamName: "Team 2", Seed: 2},
		{TeamID: uuid.New(), TeamName: "Team 3", Seed: 3},
		{TeamID: uuid.New(), TeamName: "Team 4", Seed: 4},
		{TeamID: uuid.New(), TeamName: "Team 5", Seed: 5},
	}

	request := GenerateBracketRequest{
		EventID:      eventID,
		BracketType:  BracketTypeSingleElimination,
		Teams:        teams,
		RoundID:      roundID,
		StartTime:    time.Now(),
		FieldID:      fieldID,
		GameDuration: 60,
	}

	// Setup mocks
	mockEventRepo.On("GetByID", mock.Anything, eventID).Return(&ent.Event{
		ID:   eventID,
		Name: "Test Tournament",
	}, nil)

	mockRoundRepo.On("GetByID", mock.Anything, roundID).Return(&ent.GameRound{
		ID:        roundID,
		Name:      "Bracket Round",
		RoundType: "bracket",
		Edges: ent.GameRoundEdges{
			Event: &ent.Event{ID: eventID},
		},
	}, nil)

	// Mock team lookups
	for _, team := range teams {
		mockTeamRepo.On("GetByID", mock.Anything, team.TeamID).Return(&ent.Team{
			ID:   team.TeamID,
			Name: team.TeamName,
		}, nil)
	}

	// Mock game creation
	mockGameRepo.On("Create", mock.Anything, mock.Anything).Return(&ent.Game{
		ID:     uuid.New(),
		Name:   "Test Game",
		Status: "scheduled",
	}, nil)

	// Execute
	ctx := context.Background()
	response, err := service.GenerateBracket(ctx, request)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, 3, response.TotalRounds) // 8 teams (padded) = 3 rounds
	// Should create games only for non-bye matchups
	assert.LessOrEqual(t, response.TotalGames, 4)
}

func TestGenerateBracket_InvalidRoundType(t *testing.T) {
	// Setup
	mockGameRepo := new(MockGameRepository)
	mockRoundRepo := new(MockGameRoundRepository)
	mockTeamRepo := new(MockTeamRepository)
	mockEventRepo := new(MockEventRepository)

	service := NewService(mockGameRepo, mockRoundRepo, mockTeamRepo, mockEventRepo, nil)

	eventID := uuid.New()
	roundID := uuid.New()
	fieldID := uuid.New()

	teams := []TeamSeed{
		{TeamID: uuid.New(), TeamName: "Team 1", Seed: 1},
		{TeamID: uuid.New(), TeamName: "Team 2", Seed: 2},
	}

	request := GenerateBracketRequest{
		EventID:      eventID,
		BracketType:  BracketTypeSingleElimination,
		Teams:        teams,
		RoundID:      roundID,
		StartTime:    time.Now(),
		FieldID:      fieldID,
		GameDuration: 60,
	}

	// Setup mocks
	mockEventRepo.On("GetByID", mock.Anything, eventID).Return(&ent.Event{
		ID:   eventID,
		Name: "Test Tournament",
	}, nil)

	// Round with wrong type
	mockRoundRepo.On("GetByID", mock.Anything, roundID).Return(&ent.GameRound{
		ID:        roundID,
		Name:      "Pool Round",
		RoundType: "pool", // Wrong type - should be bracket/semifinal/final
		Edges: ent.GameRoundEdges{
			Event: &ent.Event{ID: eventID},
		},
	}, nil)

	// Execute
	ctx := context.Background()
	response, err := service.GenerateBracket(ctx, request)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Contains(t, err.Error(), "round type must be")
}

func TestGetBracket_Success(t *testing.T) {
	// Setup
	mockGameRepo := new(MockGameRepository)
	mockRoundRepo := new(MockGameRoundRepository)
	mockTeamRepo := new(MockTeamRepository)
	mockEventRepo := new(MockEventRepository)

	service := NewService(mockGameRepo, mockRoundRepo, mockTeamRepo, mockEventRepo, nil)

	roundID := uuid.New()
	eventID := uuid.New()

	// Setup mocks
	mockRoundRepo.On("GetByID", mock.Anything, roundID).Return(&ent.GameRound{
		ID:        roundID,
		Name:      "Bracket Round",
		RoundType: "bracket",
		Edges: ent.GameRoundEdges{
			Event: &ent.Event{
				ID:   eventID,
				Name: "Test Tournament",
			},
		},
	}, nil)

	scheduledTime := time.Now()
	mockGames := []*ent.Game{
		{
			ID:            uuid.New(),
			Name:          "Game 1",
			Status:        "completed",
			ScheduledTime: scheduledTime,
			HomeTeamScore: 15,
			AwayTeamScore: 10,
			Edges: ent.GameEdges{
				HomeTeam: &ent.Team{ID: uuid.New(), Name: "Team A"},
				AwayTeam: &ent.Team{ID: uuid.New(), Name: "Team B"},
			},
		},
		{
			ID:            uuid.New(),
			Name:          "Game 2",
			Status:        "scheduled",
			ScheduledTime: scheduledTime.Add(time.Hour),
			Edges: ent.GameEdges{
				HomeTeam: &ent.Team{ID: uuid.New(), Name: "Team C"},
				AwayTeam: &ent.Team{ID: uuid.New(), Name: "Team D"},
			},
		},
	}

	mockGameRepo.On("ListByRound", mock.Anything, roundID).Return(mockGames, nil)

	// Execute
	ctx := context.Background()
	response, err := service.GetBracket(ctx, roundID)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, eventID, response.EventID)
	assert.Equal(t, roundID, response.RoundID)
	assert.Equal(t, 2, response.TotalGames)
	assert.NotNil(t, response.BracketTree)

	mockRoundRepo.AssertExpectations(t)
	mockGameRepo.AssertExpectations(t)
}

func TestCalculateTotalRounds(t *testing.T) {
	tests := []struct {
		name     string
		numGames int
		expected int
	}{
		{"No games", 0, 0},
		{"One game (final)", 1, 1},
		{"Two games (semis)", 2, 2},
		{"Three games", 3, 2},
		{"Four games (quarters)", 4, 3},
		{"Seven games", 7, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateTotalRounds(tt.numGames)
			assert.Equal(t, tt.expected, result)
		})
	}
}
