package bracket

import (
	"time"

	"github.com/google/uuid"
)

// BracketType represents the type of bracket structure
type BracketType string

const (
	BracketTypeSingleElimination BracketType = "single_elimination"
	BracketTypeDoubleElimination BracketType = "double_elimination"
)

// BracketNode represents a single match in the bracket tree
type BracketNode struct {
	ID             uuid.UUID    `json:"id"`
	GameID         *uuid.UUID   `json:"game_id,omitempty"`
	Round          int          `json:"round"`
	Position       int          `json:"position"`
	Team1ID        *uuid.UUID   `json:"team1_id,omitempty"`
	Team1Name      string       `json:"team1_name"`
	Team1Seed      *int         `json:"team1_seed,omitempty"`
	Team1Score     *int         `json:"team1_score,omitempty"`
	Team2ID        *uuid.UUID   `json:"team2_id,omitempty"`
	Team2Name      string       `json:"team2_name"`
	Team2Seed      *int         `json:"team2_seed,omitempty"`
	Team2Score     *int         `json:"team2_score,omitempty"`
	WinnerID       *uuid.UUID   `json:"winner_id,omitempty"`
	ScheduledTime  *time.Time   `json:"scheduled_time,omitempty"`
	Status         string       `json:"status"`
	LeftChildNode  *BracketNode `json:"left_child,omitempty"`
	RightChildNode *BracketNode `json:"right_child,omitempty"`
}

// GenerateBracketRequest represents the request to generate a tournament bracket
type GenerateBracketRequest struct {
	EventID      uuid.UUID   `json:"event_id" validate:"required"`
	BracketType  BracketType `json:"bracket_type" validate:"required,oneof=single_elimination double_elimination"`
	Teams        []TeamSeed  `json:"teams" validate:"required,min=2,dive"`
	RoundID      uuid.UUID   `json:"round_id" validate:"required"`
	StartTime    time.Time   `json:"start_time" validate:"required"`
	FieldID      uuid.UUID   `json:"field_id" validate:"required"`
	GameDuration int         `json:"game_duration" validate:"required,min=30"`
}

// TeamSeed represents a team with its seed position
type TeamSeed struct {
	TeamID   uuid.UUID `json:"team_id" validate:"required"`
	TeamName string    `json:"team_name" validate:"required"`
	Seed     int       `json:"seed" validate:"required,min=1"`
}

// GenerateBracketResponse represents the response after generating a bracket
type GenerateBracketResponse struct {
	BracketID    uuid.UUID    `json:"bracket_id"`
	EventID      uuid.UUID    `json:"event_id"`
	RoundID      uuid.UUID    `json:"round_id"`
	BracketType  BracketType  `json:"bracket_type"`
	TotalRounds  int          `json:"total_rounds"`
	TotalGames   int          `json:"total_games"`
	GamesCreated []uuid.UUID  `json:"games_created"`
	BracketTree  *BracketNode `json:"bracket_tree"`
	CreatedAt    time.Time    `json:"created_at"`
}

// GetBracketResponse represents the response for retrieving a bracket
type GetBracketResponse struct {
	EventID     uuid.UUID    `json:"event_id"`
	RoundID     uuid.UUID    `json:"round_id"`
	BracketType BracketType  `json:"bracket_type"`
	TotalRounds int          `json:"total_rounds"`
	TotalGames  int          `json:"total_games"`
	BracketTree *BracketNode `json:"bracket_tree"`
	UpdatedAt   time.Time    `json:"updated_at"`
}

// UpdateBracketNodeRequest represents the request to update a bracket node after a game
type UpdateBracketNodeRequest struct {
	GameID     uuid.UUID `json:"game_id" validate:"required"`
	WinnerID   uuid.UUID `json:"winner_id" validate:"required"`
	Team1Score int       `json:"team1_score" validate:"min=0"`
	Team2Score int       `json:"team2_score" validate:"min=0"`
}

// Matchup represents a potential game matchup for bracket generation
type Matchup struct {
	Team1ID   uuid.UUID
	Team1Seed int
	Team2ID   uuid.UUID
	Team2Seed int
	Round     int
	Position  int
}
