package migration

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/bengobox/game-stats-api/internal/pkg/logger"
)

// FixtureLoader handles loading and validating Django JSON fixtures
type FixtureLoader struct {
	fixturesDir string
}

// NewFixtureLoader creates a new fixture loader
func NewFixtureLoader(fixturesDir string) *FixtureLoader {
	return &FixtureLoader{fixturesDir: fixturesDir}
}

// LoadAndValidate loads fixtures from file and validates/normalizes the data
func (fl *FixtureLoader) LoadAndValidate(filename string) ([]DjangoFixture, error) {
	path := filepath.Join(fl.fixturesDir, filename)
	fixtures, err := loadFixtures(path)
	if err != nil {
		return nil, err
	}

	// Validate and normalize fixtures
	for i := range fixtures {
		fl.normalizeFixture(&fixtures[i])
	}

	return fixtures, nil
}

// normalizeFixture normalizes datetime and boolean fields in fixture data
func (fl *FixtureLoader) normalizeFixture(fix *DjangoFixture) {
	for key, val := range fix.Fields {
		// Normalize datetime fields
		if str, ok := val.(string); ok && str != "" {
			if isDateTimeLike(str) {
				normalized := normalizeDatetime(str)
				if normalized != str {
					fix.Fields[key] = normalized
				}
			}
		}

		// Normalize boolean fields
		if key == "is_superuser" || key == "is_staff" || key == "is_active" {
			fix.Fields[key] = normalizeBool(val)
		}
	}
}

// isDateTimeLike checks if a string looks like a datetime
func isDateTimeLike(s string) bool {
	datePattern := regexp.MustCompile(`^\d{4}-\d{2}-\d{2}`)
	return datePattern.MatchString(s)
}

// normalizeDatetime converts various datetime formats to ISO 8601 with timezone
func normalizeDatetime(s string) string {
	// Already has timezone indicator
	if strings.HasSuffix(s, "Z") || regexp.MustCompile(`[+-]\d{2}:\d{2}$`).MatchString(s) {
		return s
	}

	// Replace space with T if needed
	if strings.Contains(s, " ") && !strings.Contains(s, "T") {
		s = strings.Replace(s, " ", "T", 1)
	}

	// Add Z timezone if missing
	if strings.Contains(s, "T") {
		return s + "Z"
	}

	return s
}

// normalizeBool converts various boolean representations to actual booleans
func normalizeBool(v interface{}) bool {
	switch val := v.(type) {
	case bool:
		return val
	case string:
		return val == "1" || val == "true" || val == "True"
	case float64:
		return val == 1
	case int:
		return val == 1
	default:
		return false
	}
}

// GameFixtureNormalizer normalizes game fixture field names to match new schema
type GameFixtureNormalizer struct{}

// NormalizeGameFixture normalizes field names in a game fixture
func (gfn *GameFixtureNormalizer) NormalizeGameFixture(fix *DjangoFixture) {
	if fix.Model != "games.game" {
		return
	}

	fields := fix.Fields

	// Normalize time field names
	if val, ok := fields["start_time"]; ok && fields["date"] == nil {
		fields["date"] = val
		delete(fields, "start_time")
	}

	// Normalize team field names
	if val, ok := fields["team1"]; ok && fields["home_team"] == nil {
		fields["home_team"] = val
		delete(fields, "team1")
	}
	if val, ok := fields["team2"]; ok && fields["away_team"] == nil {
		fields["away_team"] = val
		delete(fields, "team2")
	}

	// Normalize score field names
	if val, ok := fields["team1_score"]; ok && fields["home_team_score"] == nil {
		fields["home_team_score"] = val
		delete(fields, "team1_score")
	}
	if val, ok := fields["team2_score"]; ok && fields["away_team_score"] == nil {
		fields["away_team_score"] = val
		delete(fields, "team2_score")
	}

	// Normalize pool field name
	if val, ok := fields["pool"]; ok && fields["division_pool"] == nil {
		fields["division_pool"] = val
		delete(fields, "pool")
	}

	// Ensure date format is ISO
	if date, ok := fields["date"].(string); ok && !strings.Contains(date, "T") {
		fields["date"] = strings.Replace(date, " ", "T", 1)
	}

	// Add default status if missing
	if fields["status"] == nil {
		fields["status"] = "completed"
	}

	// Add default location if missing
	if fields["location"] == nil {
		fields["location"] = 1
	}
}

// LoadGamesNormalized loads and normalizes game fixtures
func (fl *FixtureLoader) LoadGamesNormalized() ([]DjangoFixture, error) {
	fixtures, err := fl.LoadAndValidate("games_game.json")
	if err != nil {
		return nil, err
	}

	normalizer := &GameFixtureNormalizer{}
	for i := range fixtures {
		normalizer.NormalizeGameFixture(&fixtures[i])
	}

	return fixtures, nil
}

// ValidateFixtures validates all fixture files and logs any issues
func (fl *FixtureLoader) ValidateFixtures() error {
	files := []string{
		"authman_user.json",
		"core_world.json",
		"core_continent.json",
		"core_country.json",
		"core_location.json",
		"core_field.json",
		"events_discipline.json",
		"events_event.json",
		"events_divisionpool.json",
		"games_gameround.json",
		"games_team.json",
		"games_player.json",
		"games_game.json",
		"games_scoring.json",
		"games_spiritscore.json",
	}

	var errors []string

	for _, file := range files {
		path := filepath.Join(fl.fixturesDir, file)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			logger.Warn("Fixture file not found", logger.String("file", file))
			continue
		}

		fixtures, err := fl.LoadAndValidate(file)
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", file, err))
			continue
		}

		logger.Info("Validated fixture file",
			logger.String("file", file),
			logger.Int("records", len(fixtures)))
	}

	if len(errors) > 0 {
		return fmt.Errorf("validation errors: %v", errors)
	}

	return nil
}

// GenerateUnifiedSeedData combines all fixtures into a single unified structure
func (fl *FixtureLoader) GenerateUnifiedSeedData() (*UnifiedSeedData, error) {
	data := &UnifiedSeedData{
		Version:   "1.0",
		Generated: time.Now().UTC().Format(time.RFC3339),
	}

	// Load each fixture type
	if fixtures, err := fl.LoadAndValidate("core_world.json"); err == nil {
		data.Worlds = fixtures
	}
	if fixtures, err := fl.LoadAndValidate("core_continent.json"); err == nil {
		data.Continents = fixtures
	}
	if fixtures, err := fl.LoadAndValidate("core_country.json"); err == nil {
		data.Countries = fixtures
	}
	if fixtures, err := fl.LoadAndValidate("core_location.json"); err == nil {
		data.Locations = fixtures
	}
	if fixtures, err := fl.LoadAndValidate("core_field.json"); err == nil {
		data.Fields = fixtures
	}
	if fixtures, err := fl.LoadAndValidate("events_discipline.json"); err == nil {
		data.Disciplines = fixtures
	}
	if fixtures, err := fl.LoadAndValidate("events_event.json"); err == nil {
		data.Events = fixtures
	}
	if fixtures, err := fl.LoadAndValidate("events_divisionpool.json"); err == nil {
		data.DivisionPools = fixtures
	}
	if fixtures, err := fl.LoadAndValidate("games_gameround.json"); err == nil {
		data.GameRounds = fixtures
	}
	if fixtures, err := fl.LoadAndValidate("games_team.json"); err == nil {
		data.Teams = fixtures
	}
	if fixtures, err := fl.LoadAndValidate("games_player.json"); err == nil {
		data.Players = fixtures
	}
	if fixtures, err := fl.LoadGamesNormalized(); err == nil {
		data.Games = fixtures
	}
	if fixtures, err := fl.LoadAndValidate("games_scoring.json"); err == nil {
		data.Scoring = fixtures
	}
	if fixtures, err := fl.LoadAndValidate("games_spiritscore.json"); err == nil {
		data.SpiritScores = fixtures
	}

	return data, nil
}

// UnifiedSeedData represents all seed data in a single structure
type UnifiedSeedData struct {
	Version       string          `json:"version"`
	Generated     string          `json:"generated"`
	Worlds        []DjangoFixture `json:"worlds"`
	Continents    []DjangoFixture `json:"continents"`
	Countries     []DjangoFixture `json:"countries"`
	Locations     []DjangoFixture `json:"locations"`
	Fields        []DjangoFixture `json:"fields"`
	Disciplines   []DjangoFixture `json:"disciplines"`
	Events        []DjangoFixture `json:"events"`
	DivisionPools []DjangoFixture `json:"division_pools"`
	GameRounds    []DjangoFixture `json:"game_rounds"`
	Teams         []DjangoFixture `json:"teams"`
	Players       []DjangoFixture `json:"players"`
	Games         []DjangoFixture `json:"games"`
	Scoring       []DjangoFixture `json:"scoring"`
	SpiritScores  []DjangoFixture `json:"spirit_scores"`
}

// SaveUnifiedSeedData saves the unified seed data to a JSON file
func (data *UnifiedSeedData) SaveToFile(path string) error {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal seed data: %w", err)
	}

	if err := os.WriteFile(path, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write seed data: %w", err)
	}

	return nil
}

// LoadUnifiedSeedData loads unified seed data from a JSON file
func LoadUnifiedSeedData(path string) (*UnifiedSeedData, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read seed data: %w", err)
	}

	var seedData UnifiedSeedData
	if err := json.Unmarshal(data, &seedData); err != nil {
		return nil, fmt.Errorf("failed to parse seed data: %w", err)
	}

	return &seedData, nil
}
