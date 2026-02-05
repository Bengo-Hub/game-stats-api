package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/bengobox/game-stats-api/internal/pkg/auth"
	"github.com/google/uuid"
)

// Role constants for RBAC
const (
	RoleAdmin        = "admin"
	RoleEventManager = "event_manager"
	RoleTeamManager  = "team_manager"
	RoleScorekeeper  = "scorekeeper"
	RoleSpectator    = "spectator"
)

// Permission constants
type Permission string

const (
	// Dashboard
	PermViewDashboard Permission = "view_dashboard"
	// Events
	PermViewEvents   Permission = "view_events"
	PermAddEvents    Permission = "add_events"
	PermChangeEvents Permission = "change_events"
	PermDeleteEvents Permission = "delete_events"
	PermManageEvents Permission = "manage_events"
	// Games
	PermViewGames   Permission = "view_games"
	PermAddGames    Permission = "add_games"
	PermChangeGames Permission = "change_games"
	PermDeleteGames Permission = "delete_games"
	PermManageGames Permission = "manage_games"
	PermRecordScore Permission = "record_scores"
	// Teams
	PermViewTeams   Permission = "view_teams"
	PermAddTeams    Permission = "add_teams"
	PermChangeTeams Permission = "change_teams"
	PermDeleteTeams Permission = "delete_teams"
	PermManageTeams Permission = "manage_teams"
	// Players
	PermViewPlayers   Permission = "view_players"
	PermAddPlayers    Permission = "add_players"
	PermChangePlayers Permission = "change_players"
	PermDeletePlayers Permission = "delete_players"
	PermManagePlayers Permission = "manage_players"
	// Spirit
	PermViewSpirit   Permission = "view_spirit"
	PermSubmitSpirit Permission = "submit_spirit"
	PermChangeSpirit Permission = "change_spirit"
	PermManageSpirit Permission = "manage_spirit"
	// Analytics
	PermViewAnalytics   Permission = "view_analytics"
	PermExportAnalytics Permission = "export_analytics"
	// Admin
	PermViewAdmin      Permission = "view_admin"
	PermManageUsers    Permission = "manage_users"
	PermManageRoles    Permission = "manage_roles"
	PermManageSettings Permission = "manage_settings"
)

// RolePermissions maps roles to their permissions
var RolePermissions = map[string][]Permission{
	RoleAdmin: {
		PermViewDashboard,
		PermViewEvents, PermAddEvents, PermChangeEvents, PermDeleteEvents, PermManageEvents,
		PermViewGames, PermAddGames, PermChangeGames, PermDeleteGames, PermManageGames, PermRecordScore,
		PermViewTeams, PermAddTeams, PermChangeTeams, PermDeleteTeams, PermManageTeams,
		PermViewPlayers, PermAddPlayers, PermChangePlayers, PermDeletePlayers, PermManagePlayers,
		PermViewSpirit, PermSubmitSpirit, PermChangeSpirit, PermManageSpirit,
		PermViewAnalytics, PermExportAnalytics,
		PermViewAdmin, PermManageUsers, PermManageRoles, PermManageSettings,
	},
	RoleEventManager: {
		PermViewDashboard,
		PermViewEvents, PermAddEvents, PermChangeEvents, PermManageEvents,
		PermViewGames, PermAddGames, PermChangeGames, PermManageGames, PermRecordScore,
		PermViewTeams, PermAddTeams, PermChangeTeams, PermManageTeams,
		PermViewPlayers, PermAddPlayers, PermChangePlayers,
		PermViewSpirit, PermSubmitSpirit,
		PermViewAnalytics, PermExportAnalytics,
	},
	RoleTeamManager: {
		PermViewDashboard,
		PermViewEvents,
		PermViewGames,
		PermViewTeams, PermChangeTeams,
		PermViewPlayers, PermAddPlayers, PermChangePlayers, PermDeletePlayers, PermManagePlayers,
		PermViewSpirit, PermSubmitSpirit,
		PermViewAnalytics,
	},
	RoleScorekeeper: {
		PermViewDashboard,
		PermViewEvents,
		PermViewGames, PermRecordScore,
		PermViewTeams,
		PermViewPlayers,
		PermViewSpirit, PermSubmitSpirit,
	},
	RoleSpectator: {
		PermViewDashboard,
		PermViewEvents,
		PermViewGames,
		PermViewTeams,
		PermViewPlayers,
		PermViewSpirit,
		PermViewAnalytics,
	},
}

// HasPermission checks if a role has a specific permission
func HasPermission(role string, permission Permission) bool {
	if role == RoleAdmin {
		return true // Admin has all permissions
	}
	permissions, ok := RolePermissions[role]
	if !ok {
		return false
	}
	for _, p := range permissions {
		if p == permission {
			return true
		}
	}
	return false
}

// ErrorResponse writes a JSON error response
func ErrorResponse(w http.ResponseWriter, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

// AdminOnly middleware ensures only admin users can access the endpoint
func AdminOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// First, check JWT claims from auth middleware
		claims, ok := r.Context().Value(UserContextKey).(*auth.Claims)
		if ok && strings.ToLower(claims.Role) == RoleAdmin {
			next.ServeHTTP(w, r)
			return
		}

		// Fallback: check user_role context value (for backward compatibility)
		role, ok := r.Context().Value("user_role").(string)
		if ok && strings.ToLower(role) == RoleAdmin {
			next.ServeHTTP(w, r)
			return
		}

		ErrorResponse(w, "Forbidden: Admin access required", http.StatusForbidden)
	})
}

// RequirePermission middleware ensures the user has the required permission
func RequirePermission(permission Permission) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := r.Context().Value(UserContextKey).(*auth.Claims)
			if !ok {
				ErrorResponse(w, "Unauthorized: No user context", http.StatusUnauthorized)
				return
			}

			if !HasPermission(claims.Role, permission) {
				ErrorResponse(w, "Forbidden: Insufficient permissions", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireAnyPermission middleware ensures the user has at least one of the required permissions
func RequireAnyPermission(permissions ...Permission) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := r.Context().Value(UserContextKey).(*auth.Claims)
			if !ok {
				ErrorResponse(w, "Unauthorized: No user context", http.StatusUnauthorized)
				return
			}

			for _, perm := range permissions {
				if HasPermission(claims.Role, perm) {
					next.ServeHTTP(w, r)
					return
				}
			}

			ErrorResponse(w, "Forbidden: Insufficient permissions", http.StatusForbidden)
		})
	}
}

// SetUserContext middleware extracts user information from JWT claims and sets in context
// Used for admin routes that need additional context values
func SetUserContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// First, try to get user from JWT claims (auth middleware should have set this)
		claims, ok := ctx.Value(UserContextKey).(*auth.Claims)
		if ok {
			// Extract user ID from claims (already uuid.UUID type)
			if claims.UserID != uuid.Nil {
				ctx = context.WithValue(ctx, "user_id", claims.UserID)
			}
			// Set role from claims
			ctx = context.WithValue(ctx, "user_role", claims.Role)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		// Fallback: Extract user info from headers (development only)
		userIDStr := r.Header.Get("X-User-ID")
		if userIDStr != "" {
			if userID, err := uuid.Parse(userIDStr); err == nil {
				ctx = context.WithValue(ctx, "user_id", userID)
			}
		}

		username := r.Header.Get("X-Username")
		if username != "" {
			ctx = context.WithValue(ctx, "username", username)
		}

		role := r.Header.Get("X-User-Role")
		if role != "" {
			ctx = context.WithValue(ctx, "user_role", role)
		}

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
