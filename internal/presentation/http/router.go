package http

import (
	"net/http"

	"github.com/bengobox/game-stats-api/internal/config"
	"github.com/bengobox/game-stats-api/internal/presentation/http/handlers"
	"github.com/bengobox/game-stats-api/internal/presentation/http/middleware"
	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/rs/cors"
	httpSwagger "github.com/swaggo/http-swagger/v2"

	_ "github.com/bengobox/game-stats-api/docs"
)

type RouterOptions struct {
	Config             *config.Config
	AuthHandler        *handlers.AuthHandler
	SystemHandler      *handlers.SystemHandler
	GeographicHandler  *handlers.GeographicHandler
	GameHandler        *handlers.GameHandler
	GameRoundHandler   *handlers.GameRoundHandler
	SpiritScoreHandler *handlers.SpiritScoreHandler
	RankingHandler     *handlers.RankingHandler
	BracketHandler     *handlers.BracketHandler
	AnalyticsHandler   *handlers.AnalyticsHandler
	AdminHandler       *handlers.AdminHandler
	TeamHandler        *handlers.TeamHandler
	LeaderboardHandler *handlers.LeaderboardHandler
	EventHandler       *handlers.EventHandler
}

func NewRouter(opts RouterOptions) chi.Router {
	r := chi.NewRouter()

	// Initialize rate limiters
	defaultLimiter := middleware.DefaultRateLimiter()
	authLimiter := middleware.AuthRateLimiter()
	publicLimiter := middleware.PublicAPIRateLimiter()

	// Base middleware
	r.Use(chiMiddleware.Logger)
	r.Use(chiMiddleware.Recoverer)
	r.Use(chiMiddleware.RealIP) // Get real IP from proxy headers
	r.Use(middleware.RateLimit(defaultLimiter))
	r.Use(cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link", "X-Total-Count"},
		AllowCredentials: true,
		MaxAge:           300, // Maximum preflight cache duration
	}).Handler)

	// Root redirect to Swagger
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/swagger/index.html", http.StatusMovedPermanently)
	})

	// API v1
	r.Route("/api/v1", func(r chi.Router) {
		// Health check (no rate limit)
		r.Get("/health", opts.SystemHandler.Health)

		// ========================================
		// PUBLIC ROUTES (No authentication required)
		// View-only access for public pages
		// ========================================
		r.Group(func(r chi.Router) {
			r.Use(middleware.RateLimit(publicLimiter))

			// Auth routes (stricter rate limiting)
			r.Route("/auth", func(r chi.Router) {
				r.Use(middleware.RateLimit(authLimiter))
				r.Post("/login", opts.AuthHandler.Login)
				r.Post("/refresh", opts.AuthHandler.Refresh)
			})

			// Public game viewing (read-only)
			r.Route("/public", func(r chi.Router) {
				// Events/Discover - list and view events
				r.Get("/events", opts.EventHandler.ListEvents)
				r.Get("/events/{id}", opts.EventHandler.GetEvent)
				r.Get("/events/{event_id}/rounds", opts.GameRoundHandler.ListGameRounds)
				r.Get("/events/{id}/bracket", opts.BracketHandler.GetEventBracket)
				r.Get("/events/{id}/standings", opts.RankingHandler.GetDivisionStandings)
				r.Get("/events/{id}/crew", opts.EventHandler.GetEventCrew)

				// Live games - view games in progress
				r.Get("/games", opts.GameHandler.ListGames)
				r.Get("/games/{id}", opts.GameHandler.GetGame)
				r.Get("/games/{id}/timeline", opts.GameHandler.GetGameTimeline)
				r.Get("/games/{id}/scores", opts.GameHandler.GetGameScores)
				r.Get("/games/{id}/stream", opts.GameHandler.StreamGame) // SSE for live updates
				r.Get("/games/{id}/spirit", opts.SpiritScoreHandler.GetGameSpiritScores)

				// Divisions and standings
				r.Get("/divisions/{id}/standings", opts.RankingHandler.GetDivisionStandings)

				// Rounds and brackets
				r.Get("/rounds/{id}", opts.GameRoundHandler.GetGameRound)
				r.Get("/rounds/{id}/bracket", opts.BracketHandler.GetRoundBracket)

				// Teams (public info)
				r.Get("/teams", opts.TeamHandler.ListTeams)
				r.Get("/teams/{id}", opts.TeamHandler.GetTeam)
				r.Get("/teams/{id}/spirit-average", opts.SpiritScoreHandler.GetTeamSpiritAverage)

				// Leaderboards
				r.Get("/leaderboards/players", opts.LeaderboardHandler.GetPlayerLeaderboard)
				r.Get("/leaderboards/spirit", opts.LeaderboardHandler.GetSpiritLeaderboard)

				// Geographic metadata
				r.Get("/geographic/worlds", opts.GeographicHandler.ListWorlds)
				r.Get("/geographic/continents", opts.GeographicHandler.ListContinents)
				r.Get("/geographic/countries", opts.GeographicHandler.ListCountries)
			})
		})

		// ========================================
		// PROTECTED ROUTES (Authentication required)
		// ========================================
		r.Group(func(r chi.Router) {
			r.Use(middleware.AuthMiddleware(opts.Config.JWTSecret))
			r.Use(middleware.SetUserContext)

			r.Get("/me", opts.AuthHandler.Me)

			// Game management routes (with permission checks)
			r.Route("/games", func(r chi.Router) {
				// Read operations - require view_games permission
				r.Group(func(r chi.Router) {
					r.Use(middleware.RequirePermission(middleware.PermViewGames))
					r.Get("/", opts.GameHandler.ListGames)
					r.Get("/{id}", opts.GameHandler.GetGame)
					r.Get("/{id}/timeline", opts.GameHandler.GetGameTimeline)
					r.Get("/{id}/scores", opts.GameHandler.GetGameScores)
					r.Get("/{id}/stream", opts.GameHandler.StreamGame)
					r.Get("/{id}/spirit", opts.SpiritScoreHandler.GetGameSpiritScores)
				})

				// Create operations - require add_games permission
				r.Group(func(r chi.Router) {
					r.Use(middleware.RequirePermission(middleware.PermAddGames))
					r.Post("/", opts.GameHandler.ScheduleGame)
				})

				// Update operations - require change_games permission
				r.Group(func(r chi.Router) {
					r.Use(middleware.RequirePermission(middleware.PermChangeGames))
					r.Put("/{id}", opts.GameHandler.UpdateGame)
				})

				// Delete operations - require delete_games permission
				r.Group(func(r chi.Router) {
					r.Use(middleware.RequirePermission(middleware.PermDeleteGames))
					r.Delete("/{id}", opts.GameHandler.CancelGame)
				})

				// Score recording - require record_scores permission
				r.Group(func(r chi.Router) {
					r.Use(middleware.RequirePermission(middleware.PermRecordScore))
					r.Post("/{id}/start", opts.GameHandler.StartGame)
					r.Post("/{id}/finish", opts.GameHandler.FinishGame)
					r.Post("/{id}/end", opts.GameHandler.EndGame)
					r.Post("/{id}/stoppage", opts.GameHandler.RecordStoppage)
					r.Post("/{id}/score", opts.GameHandler.RecordScore)
				})

				// Spirit scores - require submit_spirit permission
				r.Group(func(r chi.Router) {
					r.Use(middleware.RequirePermission(middleware.PermSubmitSpirit))
					r.Post("/{id}/spirit", opts.SpiritScoreHandler.SubmitSpiritScore)
				})
			})

			// Event routes (rounds and brackets)
			r.Route("/events", func(r chi.Router) {
				// Read operations
				r.Group(func(r chi.Router) {
					r.Use(middleware.RequirePermission(middleware.PermViewEvents))
					r.Get("/{event_id}/rounds", opts.GameRoundHandler.ListGameRounds)
					r.Get("/{id}/bracket", opts.BracketHandler.GetEventBracket)
				})

				// Manage operations - require manage_events permission
				r.Group(func(r chi.Router) {
					r.Use(middleware.RequirePermission(middleware.PermManageEvents))
					r.Post("/{id}/generate-bracket", opts.BracketHandler.GenerateBracket)
				})
			})

			// Team routes
			r.Route("/teams", func(r chi.Router) {
				r.Use(middleware.RequirePermission(middleware.PermViewTeams))
				r.Get("/{id}/spirit-average", opts.SpiritScoreHandler.GetTeamSpiritAverage)
			})

			// Division ranking routes
			r.Route("/divisions", func(r chi.Router) {
				// Read operations
				r.Group(func(r chi.Router) {
					r.Use(middleware.RequirePermission(middleware.PermViewGames))
					r.Get("/{id}/standings", opts.RankingHandler.GetDivisionStandings)
				})

				// Manage operations
				r.Group(func(r chi.Router) {
					r.Use(middleware.RequirePermission(middleware.PermManageGames))
					r.Put("/{id}/criteria", opts.RankingHandler.UpdateRankingCriteria)
					r.Post("/advance", opts.RankingHandler.AdvanceTeams)
				})
			})

			// Round routes (game rounds and brackets)
			r.Route("/rounds", func(r chi.Router) {
				// Read operations
				r.Group(func(r chi.Router) {
					r.Use(middleware.RequirePermission(middleware.PermViewGames))
					r.Get("/{id}", opts.GameRoundHandler.GetGameRound)
					r.Get("/{id}/bracket", opts.BracketHandler.GetRoundBracket)
				})

				// Create/Update operations
				r.Group(func(r chi.Router) {
					r.Use(middleware.RequirePermission(middleware.PermManageGames))
					r.Post("/", opts.GameRoundHandler.CreateGameRound)
					r.Put("/{id}", opts.GameRoundHandler.UpdateGameRound)
				})
			})

			// Analytics routes
			r.Route("/analytics", func(r chi.Router) {
				r.Use(middleware.RequirePermission(middleware.PermViewAnalytics))
				r.Get("/health", opts.AnalyticsHandler.HealthCheck)
				r.Get("/dashboards", opts.AnalyticsHandler.ListDashboards)
				r.Get("/dashboards/{dashboard_uuid}", opts.AnalyticsHandler.GetDashboard)
				r.Post("/embed-token/{dashboard_uuid}", opts.AnalyticsHandler.GenerateEmbedToken)
				r.Get("/events/{event_id}/statistics", opts.AnalyticsHandler.GetEventStatistics)
				r.Post("/query", opts.AnalyticsHandler.NaturalLanguageQuery)
			})

			// Admin only routes
			r.Group(func(r chi.Router) {
				r.Use(middleware.AdminOnly)

				// Admin game management
				r.Route("/admin", func(r chi.Router) {
					// Game score updates
					r.Put("/games/{id}/score", opts.AdminHandler.UpdateGameScore)
					r.Get("/games/{id}/audit", opts.AdminHandler.GetGameAuditHistory)

					// Spirit score updates
					r.Put("/spirit-scores/{id}", opts.AdminHandler.UpdateSpiritScore)
				})
			})
		})
	})

	// Swagger documentation
	r.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"), // The url pointing to API definition
	))

	return r
}

// ChiRouter alias for return type convenience if needed
func GetRouter(opts RouterOptions) chi.Router {
	return NewRouter(opts)
}
