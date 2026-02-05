package migration

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/bengobox/game-stats-api/ent"
	"github.com/bengobox/game-stats-api/ent/world"
	"github.com/bengobox/game-stats-api/internal/pkg/logger"
	"github.com/google/uuid"
)

// Migrator handles idempotent data migrations from legacy Django system
type Migrator struct {
	client    *ent.Client
	idMapping *IDMapping
	loader    *FixtureLoader
}

// NewMigrator creates a new migrator instance
func NewMigrator(client *ent.Client) *Migrator {
	return &Migrator{
		client:    client,
		idMapping: NewIDMapping(),
	}
}

// RunAll executes all migration scripts idempotently
func (m *Migrator) RunAll(ctx context.Context, fixturesDir string) error {
	logger.Info("Starting data migration from legacy system...",
		logger.String("fixtures_dir", fixturesDir))

	// Initialize fixture loader
	m.loader = NewFixtureLoader(fixturesDir)

	// Validate fixtures before migration
	if err := m.loader.ValidateFixtures(); err != nil {
		logger.Warn("Fixture validation warnings", logger.Err(err))
	}

	// Migration order matters due to foreign key dependencies
	migrations := []struct {
		name string
		fn   func(context.Context, string) error
	}{
		// Users (needed for spirit score submitted_by)
		{"users", m.migrateUsers},

		// Geographic hierarchy
		{"world", m.migrateWorld},
		{"continents", m.migrateContinents},
		{"countries", m.migrateCountries},
		{"locations", m.migrateLocations},

		// Event management
		{"disciplines", m.migrateDisciplines},
		{"events", m.migrateEvents},
		{"division_pools", m.migrateDivisionPools},
		{"fields", m.migrateFields},
		{"game_rounds", m.migrateGameRounds},

		// Teams and players
		{"teams", m.migrateTeams},
		{"players", m.migratePlayers},

		// Games and scoring
		{"games", m.migrateGames},
		{"scoring", m.migrateScoring},

		// Spirit scores (with MVP/Spirit nominations)
		{"spirit_scores", m.migrateSpiritScores},
	}

	startTime := time.Now()
	successCount := 0
	var failedMigrations []string

	for _, mig := range migrations {
		logger.Info(fmt.Sprintf("Migrating %s...", mig.name))
		migStart := time.Now()

		if err := mig.fn(ctx, fixturesDir); err != nil {
			logger.Error(fmt.Sprintf("Migration failed for %s", mig.name), logger.Err(err))
			failedMigrations = append(failedMigrations, mig.name)
			// Continue with next migration instead of failing completely
			continue
		}

		elapsed := time.Since(migStart)
		logger.Info(fmt.Sprintf("✓ %s migrated successfully", mig.name),
			logger.String("duration", elapsed.String()))
		successCount++
	}

	totalElapsed := time.Since(startTime)
	logger.Info("Migration completed",
		logger.Int("successful", successCount),
		logger.Int("total", len(migrations)),
		logger.String("duration", totalElapsed.String()))

	// Log ID mapping statistics
	stats := m.idMapping.Stats()
	logger.Info("ID Mapping statistics",
		logger.Int("teams", stats["teams"]),
		logger.Int("players", stats["players"]),
		logger.Int("games", stats["games"]),
		logger.Int("fields", stats["fields"]),
		logger.Int("divisions", stats["divisions"]))

	if len(failedMigrations) > 0 {
		logger.Warn("Some migrations failed",
			logger.String("failed", strings.Join(failedMigrations, ", ")))
	}

	return nil
}

// migrateWorld ensures world entity exists
func (m *Migrator) migrateWorld(ctx context.Context, fixturesDir string) error {
	fixtures, err := m.loader.LoadAndValidate("core_world.json")
	if err != nil {
		// If file not found, create default world
		logger.Warn("World fixtures not found, creating default world")
		_, err = m.createDefaultWorld(ctx)
		return err
	}

	if len(fixtures) == 0 {
		// Create default world if no fixture
		_, err = m.createDefaultWorld(ctx)
		return err
	}

	for _, fix := range fixtures {
		slug := parseString(fix.Fields["slug"])
		if slug == "" {
			slug = "earth"
		}

		exists, err := m.client.World.Query().
			Where(world.Slug(slug)).
			Exist(ctx)
		if err != nil {
			return err
		}
		if exists {
			continue
		}

		_, err = m.client.World.Create().
			SetName(parseString(fix.Fields["name"])).
			SetSlug(slug).
			SetDescription(parseString(fix.Fields["description"])).
			Save(ctx)
		if err != nil {
			return err
		}
	}

	return nil
}

// createDefaultWorld creates the default Earth world entity
func (m *Migrator) createDefaultWorld(ctx context.Context) (*ent.World, error) {
	// Check if already exists
	exists, err := m.client.World.Query().
		Where(world.Slug("earth")).
		Exist(ctx)
	if err != nil {
		return nil, err
	}
	if exists {
		return m.client.World.Query().
			Where(world.Slug("earth")).
			Only(ctx)
	}

	return m.client.World.Create().
		SetName("Earth").
		SetSlug("earth").
		SetDescription("Planet Earth").
		Save(ctx)
}

// Legacy Django fixture structures
type DjangoFixture struct {
	Model  string                 `json:"model"`
	PK     interface{}            `json:"pk"`
	Fields map[string]interface{} `json:"fields"`
}

// loadFixtures loads Django JSON fixtures from file
func loadFixtures(path string) ([]DjangoFixture, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []DjangoFixture{}, nil // Empty is OK
		}
		return nil, err
	}

	var fixtures []DjangoFixture
	if err := json.Unmarshal(data, &fixtures); err != nil {
		return nil, fmt.Errorf("failed to parse fixture: %w", err)
	}

	return fixtures, nil
}

// Helper functions for type conversions
func parseInt(v interface{}) int {
	switch val := v.(type) {
	case float64:
		return int(val)
	case int:
		return val
	case string:
		var i int
		fmt.Sscanf(val, "%d", &i)
		return i
	default:
		return 0
	}
}

func parseTime(v interface{}) time.Time {
	if v == nil {
		return time.Now()
	}
	str, ok := v.(string)
	if !ok {
		return time.Now()
	}

	formats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, str); err == nil {
			return t
		}
	}

	return time.Now()
}

func parseString(v interface{}) string {
	if v == nil {
		return ""
	}
	if str, ok := v.(string); ok {
		return strings.TrimSpace(str)
	}
	return fmt.Sprintf("%v", v)
}

func parseUUID(v interface{}) uuid.UUID {
	if v == nil {
		return uuid.New()
	}
	if str, ok := v.(string); ok {
		if id, err := uuid.Parse(str); err == nil {
			return id
		}
	}
	return uuid.New()
}

// GetIDMapping returns the ID mapping for external use
func (m *Migrator) GetIDMapping() *IDMapping {
	return m.idMapping
}
