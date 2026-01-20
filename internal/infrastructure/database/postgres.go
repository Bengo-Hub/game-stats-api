package database

import (
	"context"
	"fmt"
	"time"

	"github.com/bengobox/game-stats-api/ent"
	"github.com/bengobox/game-stats-api/internal/pkg/logger"
	_ "github.com/lib/pq"
)

// NewClient creates a new ent client.
func NewClient(dataSourceName string, debug bool) (*ent.Client, error) {
	client, err := ent.Open("postgres", dataSourceName)
	if err != nil {
		return nil, fmt.Errorf("failed opening connection to postgres: %v", err)
	}

	if debug {
		client = client.Debug()
	}

	// Run the auto migration tool.
	// In production, you might want to use a migration tool like 'atlas'.
	if err := client.Schema.Create(context.Background()); err != nil {
		return nil, fmt.Errorf("failed creating schema resources: %v", err)
	}

	logger.Info("Database connection established and schema migrated")
	return client, nil
}

// ConnectWithRetry attempts to connect to the database with retries.
func ConnectWithRetry(dataSourceName string, debug bool, maxRetries int) (*ent.Client, error) {
	var client *ent.Client
	var err error

	for i := 0; i < maxRetries; i++ {
		client, err = NewClient(dataSourceName, debug)
		if err == nil {
			return client, nil
		}

		logger.Warn("Failed to connect to database, retrying...",
			logger.Int("attempt", i+1),
			logger.Err(err))

		time.Sleep(5 * time.Second)
	}

	return nil, err
}
