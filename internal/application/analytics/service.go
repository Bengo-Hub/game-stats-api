package analytics

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/bengobox/game-stats-api/ent"
	"github.com/bengobox/game-stats-api/ent/divisionpool"
	"github.com/bengobox/game-stats-api/ent/event"
	"github.com/bengobox/game-stats-api/ent/game"
	"github.com/bengobox/game-stats-api/ent/player"
	"github.com/bengobox/game-stats-api/ent/spiritscore"
	"github.com/bengobox/game-stats-api/ent/team"
	"github.com/google/uuid"
)

// Service handles analytics operations and Superset integration
type Service struct {
	supersetClient SupersetClientInterface
	dbClient       *ent.Client
}

// NewService creates a new analytics service
func NewService(supersetClient SupersetClientInterface, dbClient *ent.Client) *Service {
	return &Service{
		supersetClient: supersetClient,
		dbClient:       dbClient,
	}
}

// GenerateEmbedTokenRequest contains parameters for token generation
type GenerateEmbedTokenRequest struct {
	DashboardUUID uuid.UUID   `json:"dashboard_uuid" validate:"required"`
	UserID        uuid.UUID   `json:"user_id" validate:"required"`
	EventID       *uuid.UUID  `json:"event_id,omitempty"`
	TeamIDs       []uuid.UUID `json:"team_ids,omitempty"`
	Username      string      `json:"username" validate:"required,email"`
	FirstName     string      `json:"first_name" validate:"required"`
	LastName      string      `json:"last_name" validate:"required"`
}

// GenerateEmbedTokenResponse contains the generated token
type GenerateEmbedTokenResponse struct {
	Token         string    `json:"token"`
	DashboardUUID uuid.UUID `json:"dashboard_uuid"`
	ExpiresAt     time.Time `json:"expires_at"`
}

// ListDashboardsResponse contains available dashboards
type ListDashboardsResponse struct {
	Dashboards []Dashboard `json:"dashboards"`
	Total      int         `json:"total"`
}

// GenerateEmbedToken creates a guest token for embedding Superset dashboards
func (s *Service) GenerateEmbedToken(ctx context.Context, req GenerateEmbedTokenRequest) (*GenerateEmbedTokenResponse, error) {
	// Authenticate with Superset
	loginResp, err := s.supersetClient.Login(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to authenticate with Superset: %w", err)
	}

	// Build row-level security rules based on user permissions
	rlsRules := s.buildRLSRules(req.EventID, req.TeamIDs)

	// Create guest token request
	guestReq := GuestTokenRequest{
		Resources: []Resource{
			{
				Type: "dashboard",
				ID:   req.DashboardUUID.String(),
			},
		},
		RLS: rlsRules,
		User: GuestUser{
			Username:  req.Username,
			FirstName: req.FirstName,
			LastName:  req.LastName,
		},
	}

	// Generate guest token
	token, err := s.supersetClient.GenerateGuestToken(ctx, loginResp.AccessToken, guestReq)
	if err != nil {
		return nil, fmt.Errorf("failed to generate guest token: %w", err)
	}

	// Guest tokens typically expire in 5 minutes
	expiresAt := time.Now().Add(5 * time.Minute)

	return &GenerateEmbedTokenResponse{
		Token:         token,
		DashboardUUID: req.DashboardUUID,
		ExpiresAt:     expiresAt,
	}, nil
}

// ListDashboards retrieves all available Superset dashboards
func (s *Service) ListDashboards(ctx context.Context) (*ListDashboardsResponse, error) {
	// Authenticate with Superset
	loginResp, err := s.supersetClient.Login(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to authenticate with Superset: %w", err)
	}

	// Get dashboards
	dashboards, err := s.supersetClient.GetDashboards(ctx, loginResp.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to list dashboards: %w", err)
	}

	return &ListDashboardsResponse{
		Dashboards: dashboards,
		Total:      len(dashboards),
	}, nil
}

// GetDashboard retrieves a specific dashboard by UUID
func (s *Service) GetDashboard(ctx context.Context, dashboardUUID uuid.UUID) (*Dashboard, error) {
	// Authenticate with Superset
	loginResp, err := s.supersetClient.Login(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to authenticate with Superset: %w", err)
	}

	// Get dashboard
	dashboard, err := s.supersetClient.GetDashboard(ctx, loginResp.AccessToken, dashboardUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to get dashboard: %w", err)
	}

	return dashboard, nil
}

// buildRLSRules constructs row-level security rules based on user context
func (s *Service) buildRLSRules(eventID *uuid.UUID, teamIDs []uuid.UUID) []RLSRule {
	var rules []RLSRule

	// Event-level filtering
	if eventID != nil && *eventID != uuid.Nil {
		rules = append(rules, RLSRule{
			Clause: fmt.Sprintf("event_id = '%s'", eventID.String()),
		})
	}

	// Team-level filtering
	if len(teamIDs) > 0 {
		teamIDStrings := make([]string, len(teamIDs))
		for i, id := range teamIDs {
			teamIDStrings[i] = id.String()
		}
		teamIDsJoined := strings.Join(teamIDStrings, "','")
		rules = append(rules, RLSRule{
			Clause: fmt.Sprintf("team_id IN ('%s')", teamIDsJoined),
		})
	}

	return rules
}

// HealthCheck verifies Superset connectivity
func (s *Service) HealthCheck(ctx context.Context) error {
	return s.supersetClient.HealthCheck(ctx)
}

// GetEventStatistics retrieves comprehensive event analytics
func (s *Service) GetEventStatistics(ctx context.Context, eventID uuid.UUID) (*EventStatistics, error) {
	stats := &EventStatistics{
		EventID: eventID,
	}

	// Count games by status using the division pools linked to this event
	divisionPools, err := s.dbClient.DivisionPool.Query().
		Where(divisionpool.HasEventWith(event.ID(eventID))).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get division pools: %w", err)
	}

	var poolIDs []uuid.UUID
	for _, dp := range divisionPools {
		poolIDs = append(poolIDs, dp.ID)
	}

	if len(poolIDs) > 0 {
		// Count total games
		totalGames, err := s.dbClient.Game.Query().
			Where(game.HasDivisionPoolWith(divisionpool.IDIn(poolIDs...))).
			Count(ctx)
		if err == nil {
			stats.TotalGames = totalGames
		}

		// Count completed games
		completedGames, err := s.dbClient.Game.Query().
			Where(
				game.HasDivisionPoolWith(divisionpool.IDIn(poolIDs...)),
				game.StatusIn("ended", "finished"),
			).
			Count(ctx)
		if err == nil {
			stats.CompletedGames = completedGames
		}

		// Count scheduled games
		scheduledGames, err := s.dbClient.Game.Query().
			Where(
				game.HasDivisionPoolWith(divisionpool.IDIn(poolIDs...)),
				game.Status("scheduled"),
			).
			Count(ctx)
		if err == nil {
			stats.ScheduledGames = scheduledGames
		}

		// Count in-progress games
		inProgressGames, err := s.dbClient.Game.Query().
			Where(
				game.HasDivisionPoolWith(divisionpool.IDIn(poolIDs...)),
				game.Status("in_progress"),
			).
			Count(ctx)
		if err == nil {
			stats.InProgressGames = inProgressGames
		}

		// Count teams
		totalTeams, err := s.dbClient.Team.Query().
			Where(team.HasDivisionPoolWith(divisionpool.IDIn(poolIDs...))).
			Count(ctx)
		if err == nil {
			stats.TotalTeams = totalTeams
		}

		// Count players (players on teams in this event)
		totalPlayers, err := s.dbClient.Player.Query().
			Where(player.HasTeamWith(team.HasDivisionPoolWith(divisionpool.IDIn(poolIDs...)))).
			Count(ctx)
		if err == nil {
			stats.TotalPlayers = totalPlayers
		}

		// Calculate average spirit score
		spiritScores, err := s.dbClient.SpiritScore.Query().
			Where(spiritscore.HasGameWith(game.HasDivisionPoolWith(divisionpool.IDIn(poolIDs...)))).
			All(ctx)
		if err == nil && len(spiritScores) > 0 {
			var totalSpirit int
			for _, ss := range spiritScores {
				totalSpirit += ss.RulesKnowledge + ss.FoulsBodyContact + ss.FairMindedness + ss.Attitude + ss.Communication
			}
			stats.AverageSpiritScore = float64(totalSpirit) / float64(len(spiritScores))
		}
	}

	return stats, nil
}

// EventStatistics contains key metrics for an event
type EventStatistics struct {
	EventID            uuid.UUID   `json:"event_id"`
	TotalGames         int         `json:"total_games"`
	CompletedGames     int         `json:"completed_games"`
	ScheduledGames     int         `json:"scheduled_games"`
	InProgressGames    int         `json:"in_progress_games"`
	TotalTeams         int         `json:"total_teams"`
	TotalPlayers       int         `json:"total_players"`
	AverageSpiritScore float64     `json:"average_spirit_score"`
	TopScorer          *PlayerStat `json:"top_scorer,omitempty"`
}

// PlayerStat represents player statistics
type PlayerStat struct {
	PlayerID    uuid.UUID `json:"player_id"`
	PlayerName  string    `json:"player_name"`
	TeamID      uuid.UUID `json:"team_id"`
	TeamName    string    `json:"team_name"`
	Goals       int       `json:"goals"`
	Assists     int       `json:"assists"`
	GamesPlayed int       `json:"games_played"`
}
