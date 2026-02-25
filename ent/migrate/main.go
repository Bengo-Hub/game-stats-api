//go:build ignore

package main

import (
	"context"
	"log"
	"os"

	"github.com/bengobox/game-stats-api/ent/migrate"

	atlasmigrate "ariga.io/atlas/sql/migrate"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/sql/schema"
	_ "github.com/lib/pq"
)

func main() {
	ctx := context.Background()
	// Create a local migration directory able to write to "ent/migrate/migrations".
	dir, err := atlasmigrate.NewLocalDir("ent/migrate/migrations")
	if err != nil {
		log.Fatalf("failed creating atlas migration directory: %v", err)
	}
	// Migrate diff options.
	opts := []schema.MigrateOption{
		schema.WithDir(dir),                         // provide migration directory
		schema.WithMigrationMode(schema.ModeReplay), // provide migration mode
		schema.WithDialect(dialect.Postgres),        // Ent dialect to use
		schema.WithFormatter(atlasmigrate.DefaultFormatter),
	}
	if len(os.Args) != 2 {
		log.Fatalln("migration name is required. use: 'go run -mod=mod ent/migrate/main.go <name>'")
	}
	// Generate migrations using Atlas support for MySQL (or other dialects).
	err = migrate.NamedDiff(ctx, "postgres://postgres:postgres@localhost:5432/game_stats?sslmode=disable", os.Args[1], opts...)
	if err != nil {
		log.Fatalf("failed generating migration: %v", err)
	}
}
