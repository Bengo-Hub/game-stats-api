package gamemanagement

import (
	"time"

	"github.com/google/uuid"
)

// Game DTOs
type CreateGameRequest struct {
	Name                 *string                `json:"name,omitempty" validate:"omitempty,max=100"`
	ScheduledTime        time.Time              `json:"scheduled_time" validate:"required"`
	AllocatedTimeMinutes int                    `json:"allocated_time_minutes" validate:"required,min=1"`
	HomeTeamID           uuid.UUID              `json:"home_team_id" validate:"required"`
	AwayTeamID           uuid.UUID              `json:"away_team_id" validate:"required"`
	DivisionPoolID       uuid.UUID              `json:"division_pool_id" validate:"required"`
	FieldLocationID      *uuid.UUID             `json:"field_location_id,omitempty"`
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
	FieldLocationID      *uuid.UUID             `json:"field_location_id,omitempty"`
	GameRoundID          *uuid.UUID             `json:"game_round_id,omitempty"`
	HomeTeamID           *uuid.UUID             `json:"home_team_id,omitempty"`
	AwayTeamID           *uuid.UUID             `json:"away_team_id,omitempty"`
	FirstPullBy          *string                `json:"first_pull_by,omitempty"`
	Metadata             map[string]interface{} `json:"metadata,omitempty"`
}

type GameDTO struct {
	ID                   uuid.UUID              `json:"id"`
	Name                 string                 `json:"name"`
	ScheduledTime        time.Time              `json:"scheduledTime"`
	ActualStartTime      *time.Time             `json:"actualStartTime,omitempty"`
	ActualEndTime        *time.Time             `json:"actualEndTime,omitempty"`
	AllocatedTimeMinutes int                    `json:"allocatedTimeMinutes"`
	StoppageTimeSeconds  int                    `json:"stoppageTimeSeconds"`
	Status               string                 `json:"status"`
	HomeTeamScore        int                    `json:"homeTeamScore"`
	AwayTeamScore        int                    `json:"awayTeamScore"`
	FirstPullBy          *string                `json:"firstPullBy,omitempty"`
	Version              int                    `json:"version"`
	Metadata             map[string]interface{} `json:"metadata,omitempty"`
	HomeTeam             *TeamSummaryDTO        `json:"homeTeam,omitempty"`
	AwayTeam             *TeamSummaryDTO        `json:"awayTeam,omitempty"`
	FieldLocation        *FieldSummaryDTO       `json:"fieldLocation,omitempty"`
	GameRound            *GameRoundSummaryDTO   `json:"gameRound,omitempty"`
	Scorekeeper          *UserSummaryDTO        `json:"scorekeeper,omitempty"`
	EventID              uuid.UUID              `json:"eventId"`
	CreatedAt            time.Time              `json:"createdAt"`
	UpdatedAt            time.Time              `json:"updatedAt"`
}

type TeamSummaryDTO struct {
	ID             uuid.UUID `json:"id"`
	Name           string    `json:"name"`
	LogoURL        *string   `json:"logoUrl,omitempty"`
	PrimaryColor   *string   `json:"primaryColor,omitempty"`
	SecondaryColor *string   `json:"secondaryColor,omitempty"`
}

type FieldSummaryDTO struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

type GameRoundSummaryDTO struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	RoundType string    `json:"roundType"`
}

type UserSummaryDTO struct {
	ID    uuid.UUID `json:"id"`
	Name  string    `json:"name"`
	Email string    `json:"email"`
}

type PlayerSummaryDTO struct {
	ID           uuid.UUID `json:"id"`
	Name         string    `json:"name"`
	Gender       string    `json:"gender"`
	JerseyNumber *int      `json:"jerseyNumber,omitempty"`
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
	RoundType   string     `json:"round_type" validate:"required,oneof=pool crossover bracket semifinal final"`
	EventID     uuid.UUID  `json:"event_id" validate:"required"`
	RoundNumber *int       `json:"round_number,omitempty"`
	StartDate   *time.Time `json:"start_date,omitempty"`
	EndDate     *time.Time `json:"end_date,omitempty"`
	AutoAdvance bool       `json:"auto_advance,omitempty"`
	TopNTeams   *int       `json:"top_n_teams,omitempty"`
}

type UpdateGameRoundRequest struct {
	Name        *string    `json:"name,omitempty" validate:"omitempty,max=100"`
	RoundType   *string    `json:"round_type,omitempty" validate:"omitempty,oneof=pool crossover bracket semifinal final"`
	RoundNumber *int       `json:"round_number,omitempty"`
	StartDate   *time.Time `json:"start_date,omitempty"`
	EndDate     *time.Time `json:"end_date,omitempty"`
	AutoAdvance *bool      `json:"auto_advance,omitempty"`
	TopNTeams   *int       `json:"top_n_teams,omitempty"`
}

type GameRoundDTO struct {
	ID          uuid.UUID  `json:"id"`
	Name        string     `json:"name"`
	RoundType   string     `json:"roundType"`
	RoundNumber *int       `json:"roundNumber,omitempty"`
	StartDate   *time.Time `json:"startDate,omitempty"`
	EndDate     *time.Time `json:"endDate,omitempty"`
	EventID     uuid.UUID  `json:"eventId"`
	GamesCount  int        `json:"gamesCount,omitempty"`
	AutoAdvance bool       `json:"autoAdvance"`
	TopNTeams   *int       `json:"topNTeams,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
}

// Timeline DTOs
type GameTimelineDTO struct {
	Events []GameEventDTO `json:"events"`
}

type GameEventDTO struct {
	ID          uuid.UUID              `json:"id"`
	EventType   string                 `json:"eventType"`
	Minute      int                    `json:"minute"`
	Second      int                    `json:"second"`
	Description string                 `json:"description"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt   time.Time              `json:"createdAt"`
}

// Scoring DTOs
type RecordScoreRequest struct {
	PlayerID uuid.UUID `json:"player_id" validate:"required"`
	TeamID   uuid.UUID `json:"team_id" validate:"required"`
	Goals    int       `json:"goals" validate:"min=0"`
	Assists  int       `json:"assists" validate:"min=0"`
	Blocks   int       `json:"blocks" validate:"min=0"`
	Turns    int       `json:"turns" validate:"min=0"`
	Minute   *int      `json:"minute,omitempty"`
	Second   *int      `json:"second,omitempty"`
}

type ScoringDTO struct {
	ID           uuid.UUID  `json:"id"`
	PlayerID     uuid.UUID  `json:"playerId"`
	PlayerName   string     `json:"playerName,omitempty"`
	PlayerNumber *int       `json:"playerNumber,omitempty"`
	TeamID       *uuid.UUID `json:"teamId,omitempty"`
	TeamName     string     `json:"teamName,omitempty"`
	Goals        int        `json:"goals"`
	Assists      int        `json:"assists"`
	Blocks       int        `json:"blocks"`
	Turns        int        `json:"turns"`
	CreatedAt    time.Time  `json:"createdAt"`
	UpdatedAt    time.Time  `json:"updatedAt"`
}

type PlayerScore struct {
	PlayerID uuid.UUID `json:"player_id"`
	Goals    int       `json:"goals" validate:"min=0"`
	Assists  int       `json:"assists" validate:"min=0"`
	Blocks   int       `json:"blocks" validate:"min=0"`
	Turns    int       `json:"turns" validate:"min=0"`
}

type UpdateGameScoreRequest struct {
	HomeScore    int           `json:"home_score" validate:"min=0"`
	AwayScore    int           `json:"away_score" validate:"min=0"`
	PlayerScores []PlayerScore `json:"player_scores" validate:"dive"`
	Reason       string        `json:"reason" validate:"required,min=10"`
}

// Spirit Score DTOs
type SubmitSpiritScoreRequest struct {
	ScoredByTeamID         uuid.UUID  `json:"scored_by_team_id" validate:"required"`
	TeamID                 uuid.UUID  `json:"team_id" validate:"required"`
	RulesKnowledge         int        `json:"rules_knowledge" validate:"required,min=0,max=4"`
	FoulsBodyContact       int        `json:"fouls_body_contact" validate:"required,min=0,max=4"`
	FairMindedness         int        `json:"fair_mindedness" validate:"required,min=0,max=4"`
	Attitude               int        `json:"attitude" validate:"required,min=0,max=4"`
	Communication          int        `json:"communication" validate:"required,min=0,max=4"`
	Comments               *string    `json:"comments,omitempty"`
	MVPMaleNomination      *uuid.UUID `json:"mvp_male_id,omitempty"`
	MVPFemaleNomination    *uuid.UUID `json:"mvp_female_id,omitempty"`
	SpiritMaleNomination   *uuid.UUID `json:"spirit_male_id,omitempty"`
	SpiritFemaleNomination *uuid.UUID `json:"spirit_female_id,omitempty"`
}

type SpiritScoreDTO struct {
	ID                     uuid.UUID         `json:"id"`
	GameID                 uuid.UUID         `json:"gameId"`
	ScoredByTeam           *TeamSummaryDTO   `json:"scoredByTeam,omitempty"`
	Team                   *TeamSummaryDTO   `json:"team,omitempty"`
	SubmittedBy            *UserSummaryDTO   `json:"submittedBy,omitempty"`
	RulesKnowledge         int               `json:"rulesKnowledge"`
	FoulsBodyContact       int               `json:"foulsBodyContact"`
	FairMindedness         int               `json:"fairMindedness"`
	Attitude               int               `json:"attitude"`
	Communication          int               `json:"communication"`
	TotalScore             int               `json:"totalScore"`
	Comments               *string           `json:"comments,omitempty"`
	MVPMaleNomination      *PlayerSummaryDTO `json:"mvpMaleNomination,omitempty"`
	MVPFemaleNomination    *PlayerSummaryDTO `json:"mvpFemaleNomination,omitempty"`
	SpiritMaleNomination   *PlayerSummaryDTO `json:"spiritMaleNomination,omitempty"`
	SpiritFemaleNomination *PlayerSummaryDTO `json:"spiritFemaleNomination,omitempty"`
	CreatedAt              time.Time         `json:"createdAt"`
	UpdatedAt              time.Time         `json:"updatedAt"`
}

type TeamSpiritAverageDTO struct {
	TeamID                 uuid.UUID `json:"teamId"`
	TeamName               string    `json:"teamName"`
	GamesPlayed            int       `json:"gamesPlayed"`
	RulesKnowledge         float64   `json:"rulesKnowledge"`
	FoulsBodyContact       float64   `json:"foulsBodyContact"`
	FairMindedness         float64   `json:"fairMindedness"`
	Attitude               float64   `json:"attitude"`
	Communication          float64   `json:"communication"`
	AverageTotal           float64   `json:"averageTotal"`
	MVPNominationsCount    int       `json:"mvpNominationsCount"`
	SpiritNominationsCount int       `json:"spiritNominationsCount"`
}

// List filters
type ListGamesFilter struct {
	EventID        *uuid.UUID
	DivisionPoolID *uuid.UUID
	GameRoundID    *uuid.UUID
	Status         *string
	FieldID        *uuid.UUID
	StartDate      *time.Time
	EndDate        *time.Time
	RoundType      *string
	Limit          int
	Offset         int
}

// Bulk Operations DTOs
type BulkTransferRequest struct {
	Transfers     []PlayerTransfer `json:"transfers" validate:"required,dive"`
	EventID       uuid.UUID        `json:"event_id" validate:"required"` // Target Event
	SourceEventID uuid.UUID        `json:"source_event_id"`              // Optional Source Event
}

type PlayerTransfer struct {
	PlayerID   uuid.UUID `json:"playerId" validate:"required"`
	ToTeamID   uuid.UUID `json:"toTeamId" validate:"required"`
	FromTeamID uuid.UUID `json:"fromTeamId" validate:"required"`
	Role       *string   `json:"role,omitempty"`
	Status     *string   `json:"status,omitempty"`
}

type MassImportPlayersRequest struct {
	TeamID  uuid.UUID      `json:"team_id" validate:"required"`
	Players []ImportPlayer `json:"players" validate:"required,dive"`
	EventID *uuid.UUID     `json:"event_id,omitempty"`
}

type ImportPlayer struct {
	Name         string  `json:"name" validate:"required"`
	Email        *string `json:"email,omitempty"`
	Gender       string  `json:"gender" validate:"required,oneof=M F X"`
	JerseyNumber *int    `json:"jerseyNumber,omitempty"`
}
