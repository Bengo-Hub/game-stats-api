package migrate

import (
	"embed"
	"log"

	atlasmigrate "ariga.io/atlas/sql/migrate"
)

//go:embed migrations/*.sql migrations/atlas.sum
var migrations embed.FS

// Dir is the migration directory.
var Dir atlasmigrate.Dir

func init() {
	var err error
	// Use NewDirFS to support embedded migrations in production
	Dir, err = atlasmigrate.NewDirFS(migrations, "migrations")
	if err != nil {
		log.Fatalf("failed to create migration dir from embed: %v", err)
	}
}
