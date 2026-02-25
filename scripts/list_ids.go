//go:build ignore

package main

import (
	"context"
	"fmt"
	"log"

	"github.com/bengobox/game-stats-api/internal/config"
	"github.com/bengobox/game-stats-api/internal/infrastructure/database"
	"github.com/bengobox/game-stats-api/internal/pkg/logger"
)

func main() {
	cfg := config.Load()
	logger.Init(cfg.LogLevel, cfg.IsProduction())
	client, err := database.ConnectWithRetry(cfg.DatabaseURL, false, 3)
	if err != nil {
		log.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	fmt.Println("=== Locations ===")
	locs, _ := client.Location.Query().All(ctx)
	for _, l := range locs {
		fmt.Printf("ID: %s, Name: %s\n", l.ID, l.Name)
	}

	fmt.Println("\n=== Disciplines ===")
	discs, _ := client.Discipline.Query().All(ctx)
	for _, d := range discs {
		fmt.Printf("ID: %s, Name: %s\n", d.ID, d.Name)
	}

	fmt.Println("\n=== Categories ===")
	cats, _ := client.Category.Query().All(ctx)
	for _, c := range cats {
		fmt.Printf("ID: %s, Name: %s\n", c.ID, c.Name)
	}

	fmt.Println("\n=== Countries ===")
	countries, _ := client.Country.Query().All(ctx)
	for _, c := range countries {
		fmt.Printf("ID: %s, Name: %s, Slug: %s\n", c.ID, c.Name, c.Slug)
	}
}
