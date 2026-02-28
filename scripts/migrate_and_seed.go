//go:build ignore

package main

import (
	"context"
	"log"
	"time"

	"github.com/bengobox/game-stats-api/internal/config"
	"github.com/bengobox/game-stats-api/internal/infrastructure/database"
	"github.com/bengobox/game-stats-api/internal/infrastructure/migration"
	"github.com/bengobox/game-stats-api/internal/pkg/logger"
)

func main() {
	// 1. Load configuration
	cfg := config.Load()

	// 2. Initialize logger
	logger.Init(cfg.LogLevel, cfg.IsProduction())

	// 3. Connect to database (this will trigger Atlas migrations)
	logger.Info("Connecting to database and applying Atlas migrations...")
	client, err := database.ConnectWithRetry(cfg.DatabaseURL, false, 3)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer client.Close()

	// 4. Run data migration from legacy system (seeding)
	logger.Info("Starting data seeding from legacy fixtures...")
	migrator := migration.NewMigrator(client, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	defer cancel()

	fixturesDir := cfg.FixturesDir
	if fixturesDir == "" {
		fixturesDir = "./scripts/fixtures"
	}

	if err := migrator.RunAll(ctx, fixturesDir); err != nil {
		log.Fatalf("Data seeding failed: %v", err)
	}

	logger.Info("✓ Production reset: database migrated and seeded successfully")
}
