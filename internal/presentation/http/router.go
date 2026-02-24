package http

import (
	"net/http"
	"os"

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
	AdminUsersHandler  *handlers.AdminUsersHandler
	SettingsHandler    *handlers.SettingsHandler
	TeamHandler        *handlers.TeamHandler
	LeaderboardHandler *handlers.LeaderboardHandler
	EventHandler       *handlers.EventHandler
	MediaHandler       *handlers.MediaHandler
	BulkHandler        *handlers.BulkHandler
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

	// Static files for uploads
	// Construct the absolute path for the uploads directory
	uploadsDir := opts.Config.UploadsDir
	if uploadsDir == "" {
		uploadsDir = "./uploads"
	}

	// Create directory if it doesn't exist
	if _, err := os.Stat(uploadsDir); os.IsNotExist(err) {
		os.MkdirAll(uploadsDir, 0755)
	}

	fileServer := http.FileServer(http.Dir(uploadsDir))
	r.Handle("/uploads/*", http.StripPrefix("/uploads/", fileServer))

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
				r.Get("/events/{id}/standings", opts.RankingHandler.GetEventStandings)
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

				// Players
				r.Get("/players", opts.TeamHandler.ListPlayers)
				r.Get("/players/{id}", opts.TeamHandler.GetPlayer)
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

			r.Route("/events", func(r chi.Router) {
				// Read operations
				r.Group(func(r chi.Router) {
					r.Use(middleware.RequirePermission(middleware.PermViewEvents))
					r.Get("/", opts.EventHandler.ListEvents)
					r.Get("/{id}", opts.EventHandler.GetEvent)
					r.Get("/{event_id}/rounds", opts.GameRoundHandler.ListGameRounds)
					r.Get("/{id}/bracket", opts.BracketHandler.GetEventBracket)
				})

				// Manage operations - require manage_events permission
				r.Group(func(r chi.Router) {
					r.Use(middleware.RequirePermission(middleware.PermManageEvents))
					r.Post("/", opts.EventHandler.CreateEvent)
					r.Put("/{id}", opts.EventHandler.UpdateEvent)
					r.Post("/{id}/divisions", opts.EventHandler.CreateDivisionPool)
					r.Post("/{id}/generate-bracket", opts.BracketHandler.GenerateBracket)
				})
			})

			r.Route("/teams", func(r chi.Router) {
				r.With(middleware.RequirePermission(middleware.PermViewTeams)).Get("/", opts.TeamHandler.ListTeams)
				r.With(middleware.RequirePermission(middleware.PermManageTeams)).Post("/", opts.TeamHandler.CreateTeam)

				r.Route("/{id}", func(r chi.Router) {
					r.With(middleware.RequirePermission(middleware.PermViewTeams)).Get("/", opts.TeamHandler.GetTeam)
					r.With(middleware.RequirePermission(middleware.PermManageTeams)).Put("/", opts.TeamHandler.UpdateTeam)
					r.With(middleware.RequirePermission(middleware.PermViewTeams)).Get("/spirit-average", opts.SpiritScoreHandler.GetTeamSpiritAverage)

					r.Route("/players", func(r chi.Router) {
						r.With(middleware.RequirePermission(middleware.PermManageTeams)).Post("/", opts.TeamHandler.CreatePlayer)
						r.With(middleware.RequirePermission(middleware.PermManageTeams)).Put("/{playerId}", opts.TeamHandler.UpdatePlayer)
						r.With(middleware.RequirePermission(middleware.PermManageTeams)).Post("/upload", opts.TeamHandler.BulkImportPlayers)
					})
				})
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

				// Admin-only operations
				r.Route("/admin", func(r chi.Router) {
					r.Use(middleware.AdminOnly)

					r.Route("/games", func(r chi.Router) {
						r.Put("/{id}/score", opts.AdminHandler.UpdateGameScore)
						r.Post("/{id}/sync", opts.AdminHandler.SyncGameScores)
						r.Get("/{id}/audit", opts.AdminHandler.GetGameAuditHistory)
					})

					r.Route("/score-edits", func(r chi.Router) {
						r.Get("/", opts.AdminHandler.GetScoreEdits)
						r.Get("/pending", opts.AdminHandler.ListScoreEditRequests)
						r.Post("/{id}/review", opts.AdminHandler.ReviewScoreEdit)
					})

					r.Route("/users", func(r chi.Router) {
						r.Get("/", opts.AdminUsersHandler.ListUsers)
						r.Post("/", opts.AdminUsersHandler.CreateUser)
						r.Get("/{id}", opts.AdminUsersHandler.GetUser)
						r.Put("/{id}", opts.AdminUsersHandler.UpdateUser)
						r.Delete("/{id}", opts.AdminUsersHandler.DeleteUser)
						r.Put("/{id}/role", opts.AdminUsersHandler.UpdateUserRole)
						r.Post("/roles/scoped", opts.AdminUsersHandler.AssignScopedRole)
						r.Get("/{id}/roles/scoped", opts.AdminUsersHandler.ListUserScopedRoles)
					})

					r.Route("/spirit-scores", func(r chi.Router) {
						r.Put("/{id}", opts.AdminHandler.UpdateSpiritScore)
					})

					// Score corrections
					r.Route("/games/{id}", func(r chi.Router) {
						r.Put("/score", opts.AdminHandler.UpdateGameScore)
						r.Get("/audit", opts.AdminHandler.GetGameAuditHistory)
						r.Post("/sync", opts.AdminHandler.SyncGameScores)
					})

					// Audit logs
					r.Get("/audit-logs", opts.AdminUsersHandler.GetAuditLogs)

					// System health
					r.Get("/system/health", opts.AdminUsersHandler.GetSystemHealth)
				})
			})

			// Settings routes (authenticated users)
			r.Route("/settings", func(r chi.Router) {
				r.Get("/profile", opts.SettingsHandler.GetProfile)
				r.Put("/profile", opts.SettingsHandler.UpdateProfile)
				r.Put("/password", opts.SettingsHandler.ChangePassword)
				r.Delete("/account", opts.SettingsHandler.DeleteAccount)
			})

			// Media upload route
			r.Post("/upload", opts.MediaHandler.Upload)

			// Bulk operations
			r.Route("/bulk", func(r chi.Router) {
				r.Use(middleware.RequirePermission(middleware.PermManageTeams))
				r.Post("/players/transfer", opts.BulkHandler.BulkTransferPlayers)
				r.Post("/players/import", opts.BulkHandler.MassImportPlayers)
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
