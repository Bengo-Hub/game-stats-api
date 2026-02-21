package gamemanagement

import (
	"time"

	"github.com/google/uuid"
)

// Game DTOs
type CreateGameRequest struct {
	Name                 string                 `json:"name" validate:"required,max=100"`
	ScheduledTime        time.Time              `json:"scheduled_time" validate:"required"`
	AllocatedTimeMinutes int                    `json:"allocated_time_minutes" validate:"required,min=1"`
	HomeTeamID           uuid.UUID              `json:"home_team_id" validate:"required"`
	AwayTeamID           uuid.UUID              `json:"away_team_id" validate:"required"`
	DivisionPoolID       uuid.UUID              `json:"division_pool_id" validate:"required"`
	FieldLocationID      uuid.UUID              `json:"field_location_id" validate:"required"`
	GameRoundID          *uuid.UUID             `json:"game_round_id,omitempty"`
	ScorekeeperID        *uuid.UUID             `json:"scorekeeper_id,omitempty"`
	FirstPullBy          *string                `json:"first_pull_by,omitempty"`
	Metadata             map[string]interface{} `json:"metadata,omitempty"`
}

type UpdateGameRequest struct {
	Name                 *string                `json:"name,omitempty" validate:"omitempty,max=100"`
	ScheduledTime        *time.Time             `json:"scheduled_time,omitempty"`
	AllocatedTimeMinutes *int                   `json:"allocated_time_minutes,omitempty" validate:"omitempty,min=1"`
	ScorekeeperID        *uuid.UUID             `json:"scorekeeper_id,omitempty"`
	FirstPullBy          *string                `json:"first_pull_by,omitempty"`
	Metadata             map[string]interface{} `json:"metadata,omitempty"`
}

type GameDTO struct {
	ID                   uuid.UUID              `json:"id"`
	Name                 string                 `json:"name"`
	ScheduledTime        time.Time              `json:"scheduled_time"`
	ActualStartTime      *time.Time             `json:"actual_start_time,omitempty"`
	ActualEndTime        *time.Time             `json:"actual_end_time,omitempty"`
	AllocatedTimeMinutes int                    `json:"allocated_time_minutes"`
	StoppageTimeSeconds  int                    `json:"stoppage_time_seconds"`
	Status               string                 `json:"status"`
	HomeTeamScore        int                    `json:"home_team_score"`
	AwayTeamScore        int                    `json:"away_team_score"`
	FirstPullBy          *string                `json:"first_pull_by,omitempty"`
	Version              int                    `json:"version"`
	Metadata             map[string]interface{} `json:"metadata,omitempty"`
	HomeTeam             *TeamSummaryDTO        `json:"home_team,omitempty"`
	AwayTeam             *TeamSummaryDTO        `json:"away_team,omitempty"`
	FieldLocation        *FieldSummaryDTO       `json:"field_location,omitempty"`
	GameRound            *GameRoundSummaryDTO   `json:"game_round,omitempty"`
	Scorekeeper          *UserSummaryDTO        `json:"scorekeeper,omitempty"`
	CreatedAt            time.Time              `json:"created_at"`
	UpdatedAt            time.Time              `json:"updated_at"`
}

type TeamSummaryDTO struct {
	ID      uuid.UUID `json:"id"`
	Name    string    `json:"name"`
	LogoURL *string   `json:"logo_url,omitempty"`
}

type FieldSummaryDTO struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

type GameRoundSummaryDTO struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	RoundType string    `json:"round_type"`
}

type UserSummaryDTO struct {
	ID    uuid.UUID `json:"id"`
	Name  string    `json:"name"`
	Email string    `json:"email"`
}

// Game Timer DTOs
type StartGameRequest struct {
	FirstPullBy *string `json:"first_pull_by,omitempty"`
}

type RecordStoppageRequest struct {
	DurationSeconds int    `json:"duration_seconds" validate:"required,min=1"`
	Reason          string `json:"reason" validate:"required,max=255"`
}

// GameRound DTOs
type CreateGameRoundRequest struct {
	Name        string     `json:"name" validate:"required,max=100"`
	RoundType   string     `json:"round_type" validate:"required,oneof=pool bracket semifinal final"`
	EventID     uuid.UUID  `json:"event_id" validate:"required"`
	RoundNumber *int       `json:"round_number,omitempty"`
	StartDate   *time.Time `json:"start_date,omitempty"`
	EndDate     *time.Time `json:"end_date,omitempty"`
}

type UpdateGameRoundRequest struct {
	Name        *string    `json:"name,omitempty" validate:"omitempty,max=100"`
	RoundType   *string    `json:"round_type,omitempty" validate:"omitempty,oneof=pool bracket semifinal final"`
	RoundNumber *int       `json:"round_number,omitempty"`
	StartDate   *time.Time `json:"start_date,omitempty"`
	EndDate     *time.Time `json:"end_date,omitempty"`
}

type GameRoundDTO struct {
	ID          uuid.UUID  `json:"id"`
	Name        string     `json:"name"`
	RoundType   string     `json:"round_type"`
	RoundNumber *int       `json:"round_number,omitempty"`
	StartDate   *time.Time `json:"start_date,omitempty"`
	EndDate     *time.Time `json:"end_date,omitempty"`
	EventID     uuid.UUID  `json:"event_id"`
	GamesCount  int        `json:"games_count,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// Timeline DTOs
type GameTimelineDTO struct {
	Events []GameEventDTO `json:"events"`
}

type GameEventDTO struct {
	ID          uuid.UUID              `json:"id"`
	EventType   string                 `json:"event_type"`
	Minute      int                    `json:"minute"`
	Second      int                    `json:"second"`
	Description string                 `json:"description"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
}

// Scoring DTOs
type RecordScoreRequest struct {
	PlayerID uuid.UUID `json:"player_id" validate:"required"`
	Goals    int       `json:"goals" validate:"min=0"`
	Assists  int       `json:"assists" validate:"min=0"`
	Blocks   int       `json:"blocks" validate:"min=0"`
	Turns    int       `json:"turns" validate:"min=0"`
	Minute   *int      `json:"minute,omitempty"`
	Second   *int      `json:"second,omitempty"`
}

type ScoringDTO struct {
	ID           uuid.UUID  `json:"id"`
	PlayerID     uuid.UUID  `json:"player_id"`
	PlayerName   string     `json:"player_name,omitempty"`
	PlayerNumber *int       `json:"player_number,omitempty"`
	TeamID       *uuid.UUID `json:"team_id,omitempty"`
	TeamName     string     `json:"team_name,omitempty"`
	Goals        int        `json:"goals"`
	Assists      int        `json:"assists"`
	Blocks       int        `json:"blocks"`
	Turns        int        `json:"turns"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// Spirit Score DTOs
type SubmitSpiritScoreRequest struct {
	ScoredByTeamID   uuid.UUID  `json:"scored_by_team_id" validate:"required"`
	TeamID           uuid.UUID  `json:"team_id" validate:"required"`
	RulesKnowledge   int        `json:"rules_knowledge" validate:"required,min=0,max=4"`
	FoulsBodyContact int        `json:"fouls_body_contact" validate:"required,min=0,max=4"`
	FairMindedness   int        `json:"fair_mindedness" validate:"required,min=0,max=4"`
	Attitude         int        `json:"attitude" validate:"required,min=0,max=4"`
	Communication    int        `json:"communication" validate:"required,min=0,max=4"`
	Comments         *string    `json:"comments,omitempty"`
	MVPNomination    *uuid.UUID `json:"mvp_nomination,omitempty"`
	SpiritNomination *uuid.UUID `json:"spirit_nomination,omitempty"`
}

type SpiritScoreDTO struct {
	ID               uuid.UUID       `json:"id"`
	GameID           uuid.UUID       `json:"game_id"`
	ScoredByTeam     *TeamSummaryDTO `json:"scored_by_team,omitempty"`
	Team             *TeamSummaryDTO `json:"team,omitempty"`
	SubmittedBy      *UserSummaryDTO `json:"submitted_by,omitempty"`
	RulesKnowledge   int             `json:"rules_knowledge"`
	FoulsBodyContact int             `json:"fouls_body_contact"`
	FairMindedness   int             `json:"fair_mindedness"`
	Attitude         int             `json:"attitude"`
	Communication    int             `json:"communication"`
	TotalScore       int             `json:"total_score"`
	Comments         *string         `json:"comments,omitempty"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
}

type TeamSpiritAverageDTO struct {
	TeamID           uuid.UUID `json:"team_id"`
	TeamName         string    `json:"team_name"`
	GamesPlayed      int       `json:"games_played"`
	RulesKnowledge   float64   `json:"rules_knowledge"`
	FoulsBodyContact float64   `json:"fouls_body_contact"`
	FairMindedness   float64   `json:"fair_mindedness"`
	Attitude         float64   `json:"attitude"`
	Communication    float64   `json:"communication"`
	AverageTotal     float64   `json:"average_total"`
}

// List filters
type ListGamesFilter struct {
	EventID        *uuid.UUID
	DivisionPoolID *uuid.UUID
	Status         *string
	FieldID        *uuid.UUID
	StartDate      *time.Time
	EndDate        *time.Time
	Limit          int
	Offset         int
}
