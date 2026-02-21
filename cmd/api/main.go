package main

import (
	"context"
	"database/sql"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bengobox/game-stats-api/docs"
	_ "github.com/bengobox/game-stats-api/docs"
	"github.com/bengobox/game-stats-api/internal/application/admin"
	"github.com/bengobox/game-stats-api/internal/application/analytics"
	"github.com/bengobox/game-stats-api/internal/application/auth"
	"github.com/bengobox/game-stats-api/internal/application/bracket"
	"github.com/bengobox/game-stats-api/internal/application/gamemanagement"
	"github.com/bengobox/game-stats-api/internal/application/metadata"
	"github.com/bengobox/game-stats-api/internal/application/ranking"
	"github.com/bengobox/game-stats-api/internal/application/sse"
	"github.com/bengobox/game-stats-api/internal/config"
	"github.com/bengobox/game-stats-api/internal/infrastructure/cache"
	"github.com/bengobox/game-stats-api/internal/infrastructure/database"
	"github.com/bengobox/game-stats-api/internal/infrastructure/migration"
	"github.com/bengobox/game-stats-api/internal/infrastructure/repository"
	"github.com/bengobox/game-stats-api/internal/pkg/logger"
	appHttp "github.com/bengobox/game-stats-api/internal/presentation/http"
	"github.com/bengobox/game-stats-api/internal/presentation/http/handlers"
	_ "github.com/lib/pq"
)

// @title DigiGameStats API
// @version 1.0
// @description API for DigiGameStats reimplementation.
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization

func main() {
	// 1. Load configuration
	cfg := config.Load()

	// 2. Initialize logger
	logger.Init(cfg.LogLevel, cfg.IsProduction())
	defer logger.Log.Sync()

	// 2.1 Setup Swagger Info
	docs.SwaggerInfo.Host = cfg.SwaggerHost
	if cfg.IsProduction() {
		docs.SwaggerInfo.Schemes = []string{"https"}
	} else {
		docs.SwaggerInfo.Schemes = []string{"http"}
	}

	logger.Info("Starting DigiGameStats API",
		logger.String("env", cfg.Env),
		logger.String("port", cfg.Port))

	// 3. Connect to database
	client, err := database.ConnectWithRetry(cfg.DatabaseURL, !cfg.IsProduction(), 5)
	if err != nil {
		logger.Fatal("Failed to connect to database", logger.Err(err))
	}
	defer client.Close()

	// 3.0.1. Open raw SQL connection for analytics queries
	rawDB, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		logger.Fatal("Failed to open raw database connection", logger.Err(err))
	}
	defer rawDB.Close()

	// 3.1. Connect to Redis
	redisClient, err := cache.NewRedisClient(cfg.RedisURL)
	if err != nil {
		logger.Fatal("Failed to connect to Redis", logger.Err(err))
	}
	defer redisClient.Close()
	logger.Info("Connected to Redis", logger.String("url", cfg.RedisURL))

	// 3.2. Run data migration from legacy system (idempotent)
	if cfg.RunMigration {
		logger.Info("Running data migration from legacy Django fixtures...")
		migrator := migration.NewMigrator(client)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()

		fixturesDir := cfg.FixturesDir
		if fixturesDir == "" {
			fixturesDir = "./scripts/fixtures"
		}

		if err := migrator.RunAll(ctx, fixturesDir); err != nil {
			logger.Error("Data migration failed (continuing anyway)", logger.Err(err))
		} else {
			logger.Info("✓ Data migration completed successfully")
		}
	} else {
		logger.Info("Skipping data migration (RUN_MIGRATION=false)")
	}

	// 4. Initialize repositories
	userRepo := repository.NewUserRepository(client)
	worldRepo := repository.NewWorldRepository(client)
	continentRepo := repository.NewContinentRepository(client)
	countryRepo := repository.NewCountryRepository(client)

	// Game management repositories
	gameRepo := repository.NewGameRepository(client)
	gameRoundRepo := repository.NewGameRoundRepository(client)
	gameEventRepo := repository.NewGameEventRepository(client)
	scoringRepo := repository.NewScoringRepository(client)
	spiritScoreRepo := repository.NewSpiritScoreRepository(client)
	mvpNominationRepo := repository.NewMVPNominationRepository(client)
	spiritNominationRepo := repository.NewSpiritNominationRepository(client)
	teamRepo := repository.NewTeamRepository(client)
	playerRepo := repository.NewPlayerRepository(client)
	fieldRepo := repository.NewFieldRepository(client)
	divisionRepo := repository.NewDivisionPoolRepository(client)

	// Event repository for bracket generation
	eventRepo := repository.NewEventRepository(client)

	// Instantiate other repositories to ensure they are valid and compiled
	_ = repository.NewLocationRepository(client)
	_ = repository.NewDisciplineRepository(client)
	_ = repository.NewEventReconciliationRepository(client)
	_ = repository.NewAnalyticsEmbeddingRepository(client)

	// 5. Initialize application services
	authService := auth.NewService(userRepo, cfg)
	metadataService := metadata.NewService(worldRepo, continentRepo, countryRepo)
	gameManagementService := gamemanagement.NewService(
		gameRepo,
		gameRoundRepo,
		gameEventRepo,
		scoringRepo,
		spiritScoreRepo,
		mvpNominationRepo,
		spiritNominationRepo,
		teamRepo,
		playerRepo,
		fieldRepo,
		divisionRepo,
		userRepo,
		eventRepo,
	)

	// Initialize SSE broker for real-time updates
	sseBroker := sse.NewBroker()
	defer sseBroker.Shutdown()

	// Initialize ranking service with cache
	rankingService := ranking.NewService(
		divisionRepo,
		gameRepo,
		teamRepo,
		gameRoundRepo,
		redisClient,
	)

	// Initialize bracket service with cache
	bracketService := bracket.NewService(
		gameRepo,
		gameRoundRepo,
		teamRepo,
		eventRepo,
		redisClient,
	)

	// Initialize analytics service with Metabase client (adapter)
	metabaseClient := analytics.NewMetabaseClient(
		cfg.MetabaseBaseURL,
		cfg.MetabaseUsername,
		cfg.MetabasePassword,
	)
	analyticsService := analytics.NewService(metabaseClient, client)

	// Initialize Ollama client and text-to-SQL service
	ollamaClient := analytics.NewOllamaClient(
		cfg.OllamaBaseURL,
		cfg.OllamaModel,
	)
	textToSQLService := analytics.NewTextToSQLService(ollamaClient, client, rawDB)

	// Initialize audit repository
	auditRepo := repository.NewInMemoryAuditRepository()

	// Initialize admin service
	adminService := admin.NewScoreAdminService(gameRepo, spiritScoreRepo, auditRepo, redisClient)

	// 6. Initialize HTTP handlers
	authHandler := handlers.NewAuthHandler(authService, cfg.JWTSecret)
	systemHandler := handlers.NewSystemHandler()
	geographicHandler := handlers.NewGeographicHandler(metadataService)
	gameHandler := handlers.NewGameHandler(gameManagementService, sseBroker)
	gameRoundHandler := handlers.NewGameRoundHandler(gameManagementService)
	spiritScoreHandler := handlers.NewSpiritScoreHandler(gameManagementService)
	rankingHandler := handlers.NewRankingHandler(rankingService)
	bracketHandler := handlers.NewBracketHandler(bracketService)
	analyticsHandler := handlers.NewAnalyticsHandler(analyticsService, textToSQLService)
	adminHandler := handlers.NewAdminHandler(adminService)
	adminUsersHandler := handlers.NewAdminUsersHandler(userRepo, client)
	settingsHandler := handlers.NewSettingsHandler(userRepo)
	teamHandler := handlers.NewTeamHandler(client)
	leaderboardHandler := handlers.NewLeaderboardHandler(client)
	eventHandler := handlers.NewEventHandler(client)
	mediaHandler := handlers.NewMediaHandler(cfg.UploadsDir, cfg.ApiBaseURL)

	// 7. Setup router
	router := appHttp.NewRouter(appHttp.RouterOptions{
		Config:             cfg,
		AuthHandler:        authHandler,
		SystemHandler:      systemHandler,
		GeographicHandler:  geographicHandler,
		GameHandler:        gameHandler,
		GameRoundHandler:   gameRoundHandler,
		SpiritScoreHandler: spiritScoreHandler,
		RankingHandler:     rankingHandler,
		BracketHandler:     bracketHandler,
		AnalyticsHandler:   analyticsHandler,
		AdminHandler:       adminHandler,
		AdminUsersHandler:  adminUsersHandler,
		SettingsHandler:    settingsHandler,
		TeamHandler:        teamHandler,
		LeaderboardHandler: leaderboardHandler,
		EventHandler:       eventHandler,
		MediaHandler:       mediaHandler,
	})

	// 8. Start server
	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	go func() {
		logger.Info("Server listening", logger.String("addr", server.Addr))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("ListenAndServe failed", logger.Err(err))
		}
	}()

	// Wait for interrupt signal
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	logger.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("Server forced to shutdown", logger.Err(err))
	}

	logger.Info("Server exited gracefully")
}
