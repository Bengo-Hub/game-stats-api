package ranking

import (
	"time"

	"github.com/google/uuid"
)

// RankingCriteria defines how teams are ranked in a division
type RankingCriteria struct {
	PrimarySort   string   `json:"primary_sort"`   // "points", "win_percentage", "goal_diff"
	SecondarySort []string `json:"secondary_sort"` // Tiebreakers: ["goal_diff", "head_to_head", "goals_for"]
	PointsPerWin  int      `json:"points_per_win"`
	PointsPerDraw int      `json:"points_per_draw"`
	PointsPerLoss int      `json:"points_per_loss"`
}

// DefaultRankingCriteria returns the standard Ultimate ranking criteria
func DefaultRankingCriteria() RankingCriteria {
	return RankingCriteria{
		PrimarySort:   "points",
		SecondarySort: []string{"goal_diff", "goals_for", "head_to_head"},
		PointsPerWin:  3,
		PointsPerDraw: 1,
		PointsPerLoss: 0,
	}
}

// TeamStanding represents a team's standing in the division
type TeamStanding struct {
	Rank           int       `json:"rank"`
	TeamID         uuid.UUID `json:"team_id"`
	TeamName       string    `json:"team_name"`
	GamesPlayed    int       `json:"games_played"`
	Wins           int       `json:"wins"`
	Draws          int       `json:"draws"`
	Losses         int       `json:"losses"`
	GoalsFor       int       `json:"goals_for"`
	GoalsAgainst   int       `json:"goals_against"`
	GoalDifference int       `json:"goal_difference"`
	Points         int       `json:"points"`
	WinPercentage  float64   `json:"win_percentage"`
	SpiritAverage  *float64  `json:"spirit_average,omitempty"`
	LastUpdated    time.Time `json:"last_updated"`
}

// DivisionStandingsResponse contains all teams' standings
type DivisionStandingsResponse struct {
	DivisionID      uuid.UUID        `json:"division_id"`
	DivisionName    string           `json:"division_name"`
	Standings       []TeamStanding   `json:"standings"`
	RankingCriteria *RankingCriteria `json:"ranking_criteria,omitempty"`
	LastUpdated     time.Time        `json:"last_updated"`
}

// UpdateRankingCriteriaRequest updates division ranking rules
type UpdateRankingCriteriaRequest struct {
	PrimarySort   string   `json:"primary_sort" validate:"required,oneof=points win_percentage goal_diff"`
	SecondarySort []string `json:"secondary_sort"`
	PointsPerWin  int      `json:"points_per_win" validate:"required,min=0"`
	PointsPerDraw int      `json:"points_per_draw" validate:"required,min=0"`
	PointsPerLoss int      `json:"points_per_loss" validate:"min=0"`
}

// AdvanceTeamsRequest triggers team advancement to next round
type AdvanceTeamsRequest struct {
	DivisionID      uuid.UUID  `json:"division_id" validate:"required"`
	TopN            int        `json:"top_n" validate:"required,min=1"`
	TargetRoundID   uuid.UUID  `json:"target_round_id" validate:"required"`
	NotifyTeams     bool       `json:"notify_teams"`
	GenerateBracket bool       `json:"generate_bracket"`
	StartTime       *time.Time `json:"start_time,omitempty"`
	FieldID         *uuid.UUID `json:"field_id,omitempty"`
	GameDuration    int        `json:"game_duration,omitempty" validate:"omitempty,min=30"`
}

// AdvanceTeamsResponse returns the advancement result
type AdvanceTeamsResponse struct {
	AdvancedTeams []uuid.UUID `json:"advanced_teams"`
	TargetRoundID uuid.UUID   `json:"target_round_id"`
	GamesCreated  int         `json:"games_created,omitempty"`
	BracketID     *uuid.UUID  `json:"bracket_id,omitempty"`
	Message       string      `json:"message"`
}
