package auth

import (
	"context"

	"github.com/bengobox/game-stats-api/ent"
	"github.com/bengobox/game-stats-api/ent/game"
	"github.com/bengobox/game-stats-api/ent/scopedrole"
	"github.com/google/uuid"
)

// PermissionService centralizes authorization logic
type PermissionService struct {
	client *ent.Client
}

// NewPermissionService creates a new permission service
func NewPermissionService(client *ent.Client) *PermissionService {
	return &PermissionService{client: client}
}

// CheckPermission checks if a user has a specific role for a given scope
func (s *PermissionService) CheckPermission(ctx context.Context, userID uuid.UUID, requiredRole string, scopeType string, scopeID uuid.UUID) (bool, error) {
	// 1. System Admins have global access
	u, err := s.client.User.Get(ctx, userID)
	if err != nil {
		return false, err
	}
	if u.Role == "admin" {
		return true, nil
	}

	// 2. Check for exact scoped role match
	exists, err := s.client.ScopedRole.Query().
		Where(
			scopedrole.UserID(userID),
			scopedrole.Role(requiredRole),
			scopedrole.ScopeType(scopeType),
			scopedrole.ScopeID(scopeID),
		).
		Exist(ctx)
	if err != nil {
		return false, err
	}
	if exists {
		return true, nil
	}

	// 3. Handle hierarchical/inherited permissions

	// 'event_manager' implicitly has 'game_admin' and 'scorekeeper' for all games in the event
	if scopeType == "game" && (requiredRole == "game_admin" || requiredRole == "scorekeeper") {
		g, err := s.client.Game.Query().
			Where(game.ID(scopeID)).
			WithDivisionPool(func(q *ent.DivisionPoolQuery) {
				q.WithEvents()
			}).
			Only(ctx)
		if err == nil && g.Edges.DivisionPool != nil && len(g.Edges.DivisionPool.Edges.Events) > 0 {
			eventID := g.Edges.DivisionPool.Edges.Events[0].ID
			isEventManager, _ := s.CheckPermission(ctx, userID, "event_manager", "event", eventID)
			if isEventManager {
				return true, nil
			}
		}
	}

	// 'game_admin' implicitly has 'scorekeeper' for that specifically assigned game
	if scopeType == "game" && requiredRole == "scorekeeper" {
		isGameAdmin, _ := s.CheckPermission(ctx, userID, "game_admin", "game", scopeID)
		if isGameAdmin {
			return true, nil
		}
	}

	return false, nil
}

// GetUserRoles returns all scoped roles for a user
func (s *PermissionService) GetUserRoles(ctx context.Context, userID uuid.UUID) ([]*ent.ScopedRole, error) {
	return s.client.ScopedRole.Query().
		Where(scopedrole.UserID(userID)).
		All(ctx)
}

// IsScorekeeperForGame checks if a user is authorized to record scores for a game
func (s *PermissionService) IsScorekeeperForGame(ctx context.Context, userID uuid.UUID, gameID uuid.UUID) (bool, error) {
	// Check if explicitly assigned as scorekeeper for this game
	isGameScorekeeper, err := s.CheckPermission(ctx, userID, "scorekeeper", "game", gameID)
	if err != nil {
		return false, err
	}
	if isGameScorekeeper {
		return true, nil
	}

	// Check if assigned as scorekeeper for the entire event
	g, err := s.client.Game.Query().
		Where(game.ID(gameID)).
		WithDivisionPool(func(q *ent.DivisionPoolQuery) {
			q.WithEvents()
		}).
		Only(ctx)
	if err == nil && g.Edges.DivisionPool != nil && len(g.Edges.DivisionPool.Edges.Events) > 0 {
		eventID := g.Edges.DivisionPool.Edges.Events[0].ID
		isEventScorekeeper, _ := s.CheckPermission(ctx, userID, "scorekeeper", "event", eventID)
		if isEventScorekeeper {
			return true, nil
		}
	}

	return false, nil
}
