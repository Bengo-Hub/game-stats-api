package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bengobox/game-stats-api/internal/application/auth"
	"github.com/bengobox/game-stats-api/internal/config"
	"github.com/bengobox/game-stats-api/internal/infrastructure/database"
	"github.com/bengobox/game-stats-api/internal/infrastructure/repository"
	"github.com/bengobox/game-stats-api/internal/pkg/logger"
	appHttp "github.com/bengobox/game-stats-api/internal/presentation/http"
	"github.com/bengobox/game-stats-api/internal/presentation/http/handlers"
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

// @host localhost:8080
// @BasePath /api/v1

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization

func main() {
	// 1. Load configuration
	cfg := config.Load()

	// 2. Initialize logger
	logger.Init(cfg.LogLevel, cfg.IsProduction())
	defer logger.Log.Sync()

	logger.Info("Starting DigiGameStats API",
		logger.String("env", cfg.Env),
		logger.String("port", cfg.Port))

	// 3. Connect to database
	client, err := database.ConnectWithRetry(cfg.DatabaseURL, !cfg.IsProduction(), 5)
	if err != nil {
		logger.Fatal("Failed to connect to database", logger.Err(err))
	}
	defer client.Close()

	// 4. Initialize repositories
	userRepo := repository.NewUserRepository(client)

	// Instantiate other repositories to ensure they are valid and compiled
	_ = repository.NewWorldRepository(client)
	_ = repository.NewContinentRepository(client)
	_ = repository.NewCountryRepository(client)
	_ = repository.NewLocationRepository(client)
	_ = repository.NewFieldRepository(client)
	_ = repository.NewDisciplineRepository(client)
	_ = repository.NewEventRepository(client)
	_ = repository.NewDivisionPoolRepository(client)
	_ = repository.NewTeamRepository(client)
	_ = repository.NewPlayerRepository(client)
	_ = repository.NewEventReconciliationRepository(client)
	_ = repository.NewScoringRepository(client)
	_ = repository.NewSpiritScoreRepository(client)
	_ = repository.NewAnalyticsEmbeddingRepository(client)
	_ = repository.NewMVPNominationRepository(client)
	_ = repository.NewSpiritNominationRepository(client)

	// 5. Initialize application services
	authService := auth.NewService(userRepo, cfg)

	// 6. Initialize HTTP handlers
	authHandler := handlers.NewAuthHandler(authService)

	// 7. Setup router
	router := appHttp.NewRouter(appHttp.RouterOptions{
		Config:      cfg,
		AuthHandler: authHandler,
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
