package migration

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/bengobox/game-stats-api/ent"
	"github.com/bengobox/game-stats-api/ent/discipline"
	"github.com/bengobox/game-stats-api/ent/divisionpool"
	"github.com/bengobox/game-stats-api/ent/event"
	entfield "github.com/bengobox/game-stats-api/ent/field"
	"github.com/bengobox/game-stats-api/ent/gameround"
	"github.com/bengobox/game-stats-api/internal/pkg/logger"
)

// migrateDisciplines migrates disciplines from events_discipline.json
func (m *Migrator) migrateDisciplines(ctx context.Context, fixturesDir string) error {
	fixtures, err := loadFixtures(filepath.Join(fixturesDir, "events_discipline.json"))
	if err != nil {
		return err
	}

	for _, fix := range fixtures {
		legacyID := parseInt(fix.PK)
		slug := parseString(fix.Fields["slug"])

		existingDiscipline, err := m.client.Discipline.Query().
			Where(discipline.Slug(slug)).
			Only(ctx)
		if err == nil {
			// Already exists, store mapping
			m.idMapping.SetDiscipline(legacyID, existingDiscipline.ID)
			continue
		}
		if !ent.IsNotFound(err) {
			return err
		}

		// Find country using ID mapping first
		countryLegacyID := parseInt(fix.Fields["country"])
		var parentCountry *ent.Country

		if countryUUID, ok := m.idMapping.GetCountry(countryLegacyID); ok {
			parentCountry, _ = m.client.Country.Get(ctx, countryUUID)
		}

		// Fallback: use first available country (discipline requires country)
		if parentCountry == nil {
			parentCountry, _ = m.client.Country.Query().First(ctx)
		}

		if parentCountry == nil {
			logger.Warn("No country available, skipping discipline", logger.String("slug", slug))
			continue
		}

		newDiscipline, err := m.client.Discipline.Create().
			SetName(parseString(fix.Fields["name"])).
			SetSlug(slug).
			SetDescription(parseString(fix.Fields["description"])).
			SetCountry(parentCountry).
			Save(ctx)
		if err != nil {
			return err
		}

		// Store ID mapping
		m.idMapping.SetDiscipline(legacyID, newDiscipline.ID)
	}

	return nil
}

// migrateEvents migrates events from events_event.json
func (m *Migrator) migrateEvents(ctx context.Context, fixturesDir string) error {
	fixtures, err := loadFixtures(filepath.Join(fixturesDir, "events_event.json"))
	if err != nil {
		return err
	}

	for _, fix := range fixtures {
		legacyID := parseInt(fix.PK)
		slug := parseString(fix.Fields["slug"])

		existingEvent, err := m.client.Event.Query().
			Where(event.Slug(slug)).
			Only(ctx)
		if err == nil {
			// Already exists, store mapping
			m.idMapping.SetEvent(legacyID, existingEvent.ID)
			continue
		}
		if !ent.IsNotFound(err) {
			return err
		}

		// Find discipline using ID mapping first
		disciplineLegacyID := parseInt(fix.Fields["discipline"])
		var disc *ent.Discipline

		if disciplineUUID, ok := m.idMapping.GetDiscipline(disciplineLegacyID); ok {
			disc, _ = m.client.Discipline.Get(ctx, disciplineUUID)
		}

		// Fallback to slug map
		if disc == nil {
			disciplineSlugMap := map[int]string{
				1: "ultimate",
				2: "basketball",
				3: "soccer",
			}

			if discSlug, ok := disciplineSlugMap[disciplineLegacyID]; ok {
				disc, _ = m.client.Discipline.Query().
					Where(discipline.Slug(discSlug)).
					Only(ctx)
			}
		}

		if disc == nil {
			logger.Warn("Discipline not found", logger.Int("legacy_id", disciplineLegacyID))
		}

		// Find location using ID mapping first
		locationLegacyID := parseInt(fix.Fields["location"])
		var loc *ent.Location

		if locationUUID, ok := m.idMapping.GetLocation(locationLegacyID); ok {
			loc, _ = m.client.Location.Get(ctx, locationUUID)
		}

		if loc == nil {
			logger.Warn("Location not found for event", logger.Int("legacy_id", locationLegacyID))
		}

		creator := m.client.Event.Create().
			SetName(parseString(fix.Fields["name"])).
			SetSlug(slug).
			SetDescription(parseString(fix.Fields["description"])).
			SetYear(parseInt(fix.Fields["year"])).
			SetStartDate(parseTime(fix.Fields["start_date"])).
			SetEndDate(parseTime(fix.Fields["end_date"]))

		if disc != nil {
			creator.SetDiscipline(disc)
		}
		if loc != nil {
			creator.SetLocation(loc)
		}

		newEvent, err := creator.Save(ctx)
		if err != nil {
			return err
		}

		// Store ID mapping
		m.idMapping.SetEvent(legacyID, newEvent.ID)
	}

	return nil
}

// migrateDivisionPools migrates division pools from events_divisionpool.json
// NOTE: The correct file is events_divisionpool.json, not games_divisionpool.json (which is empty)
func (m *Migrator) migrateDivisionPools(ctx context.Context, fixturesDir string) error {
	// FIXED: Load from events_divisionpool.json (contains actual data)
	fixtures, err := loadFixtures(filepath.Join(fixturesDir, "events_divisionpool.json"))
	if err != nil {
		return err
	}

	// Empty file is OK
	if len(fixtures) == 0 {
		logger.Info("No division pool fixtures found")
		return nil
	}

	for _, fix := range fixtures {
		legacyID := parseInt(fix.PK)
		name := parseString(fix.Fields["name"])

		existingDivision, err := m.client.DivisionPool.Query().
			Where(divisionpool.Name(name)).
			Only(ctx)
		if err == nil {
			// Already exists, store mapping
			m.idMapping.SetDivision(legacyID, existingDivision.ID)
			continue
		}
		if !ent.IsNotFound(err) {
			return err
		}

		// Find parent event using ID mapping
		eventLegacyID := parseInt(fix.Fields["event"])
		var parentEvent *ent.Event

		if eventUUID, ok := m.idMapping.GetEvent(eventLegacyID); ok {
			parentEvent, _ = m.client.Event.Get(ctx, eventUUID)
		}

		// Fallback: use first available event
		if parentEvent == nil {
			parentEvent, _ = m.client.Event.Query().First(ctx)
		}

		// Determine division type from name
		divisionType := parseString(fix.Fields["division_type"])
		if divisionType == "" {
			// Infer from name
			nameLower := strings.ToLower(name)
			switch {
			case strings.Contains(nameLower, "mixed"):
				divisionType = "mixed"
			case strings.Contains(nameLower, "open"):
				divisionType = "open"
			case strings.Contains(nameLower, "women"):
				divisionType = "women"
			case strings.Contains(nameLower, "pool"):
				divisionType = "pool"
			default:
				divisionType = "pool"
			}
		}

		creator := m.client.DivisionPool.Create().
			SetName(name).
			SetDivisionType(divisionType).
			SetDescription(parseString(fix.Fields["description"]))

		if parentEvent != nil {
			creator.SetEvent(parentEvent)
		}

		newDivision, err := creator.Save(ctx)
		if err != nil {
			return err
		}

		// Store ID mapping
		m.idMapping.SetDivision(legacyID, newDivision.ID)
	}

	return nil
}

// migrateFields migrates fields from core_field.json
// NOTE: The correct file is core_field.json, not games_field.json (which is empty)
func (m *Migrator) migrateFields(ctx context.Context, fixturesDir string) error {
	// FIXED: Load from core_field.json (contains actual data)
	fixtures, err := loadFixtures(filepath.Join(fixturesDir, "core_field.json"))
	if err != nil {
		return err
	}

	// Empty file is OK
	if len(fixtures) == 0 {
		logger.Info("No field fixtures found")
		return nil
	}

	for _, fix := range fixtures {
		legacyID := parseInt(fix.PK)
		name := parseString(fix.Fields["name"])

		existingField, err := m.client.Field.Query().
			Where(entfield.Name(name)).
			Only(ctx)
		if err == nil {
			// Already exists, store mapping
			m.idMapping.SetField(legacyID, existingField.ID)
			continue
		}
		if !ent.IsNotFound(err) {
			return err
		}

		// Find parent location using ID mapping
		locationLegacyID := parseInt(fix.Fields["location"])
		var parentLocation *ent.Location

		if locationUUID, ok := m.idMapping.GetLocation(locationLegacyID); ok {
			parentLocation, _ = m.client.Location.Get(ctx, locationUUID)
		}

		// Fallback: use first available location
		if parentLocation == nil {
			parentLocation, _ = m.client.Location.Query().First(ctx)
		}

		creator := m.client.Field.Create().
			SetName(name).
			SetSurfaceType(parseString(fix.Fields["surface_type"])).
			SetCapacity(parseInt(fix.Fields["capacity"]))

		if parentLocation != nil {
			creator.SetLocation(parentLocation)
		}

		newField, err := creator.Save(ctx)
		if err != nil {
			return err
		}

		// Store ID mapping
		m.idMapping.SetField(legacyID, newField.ID)
	}

	return nil
}

// migrateGameRounds migrates game rounds from games_gameround.json
func (m *Migrator) migrateGameRounds(ctx context.Context, fixturesDir string) error {
	fixtures, err := loadFixtures(filepath.Join(fixturesDir, "games_gameround.json"))
	if err != nil {
		return err
	}

	// Get first event for association
	var firstEvent *ent.Event
	firstEvent, _ = m.client.Event.Query().First(ctx)

	for _, fix := range fixtures {
		legacyID := parseInt(fix.PK)
		name := parseString(fix.Fields["name"])

		existingRound, err := m.client.GameRound.Query().
			Where(gameround.Name(name)).
			Only(ctx)
		if err == nil {
			// Already exists, store mapping
			m.idMapping.SetGameRound(legacyID, existingRound.ID)
			continue
		}
		if !ent.IsNotFound(err) {
			return err
		}

		// Determine round type from name (since fixture doesn't have this)
		roundType := "pool"
		roundNumber := legacyID
		switch name {
		case "Round Robin":
			roundType = "pool"
			roundNumber = 1
		case "Play-Offs":
			roundType = "bracket"
			roundNumber = 2
		case "Quater Finals", "Quarter Finals":
			roundType = "bracket"
			roundNumber = 3
		case "Semi-Finals":
			roundType = "bracket"
			roundNumber = 4
		case "Finals":
			roundType = "final"
			roundNumber = 5
		}

		creator := m.client.GameRound.Create().
			SetName(name).
			SetRoundType(roundType).
			SetRoundNumber(roundNumber)

		// Associate with first event if available
		if firstEvent != nil {
			creator.SetEvent(firstEvent)
		}

		newRound, err := creator.Save(ctx)
		if err != nil {
			return err
		}

		// Store ID mapping
		m.idMapping.SetGameRound(legacyID, newRound.ID)
	}

	return nil
}
