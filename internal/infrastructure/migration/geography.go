package migration

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/bengobox/game-stats-api/ent"
	"github.com/bengobox/game-stats-api/ent/continent"
	"github.com/bengobox/game-stats-api/ent/country"
	"github.com/bengobox/game-stats-api/ent/location"
	"github.com/bengobox/game-stats-api/ent/world"
	"github.com/bengobox/game-stats-api/internal/pkg/logger"
)

// migrateContinents migrates continents from core_continent.json
func (m *Migrator) migrateContinents(ctx context.Context, fixturesDir string) error {
	fixtures, err := loadFixtures(filepath.Join(fixturesDir, "core_continent.json"))
	if err != nil {
		return err
	}

	// First ensure world exists
	worldEntity, err := m.client.World.Query().Where(world.Slug("earth")).Only(ctx)
	if err != nil {
		if !ent.IsNotFound(err) {
			return err
		}
		// Create default world
		worldEntity, err = m.client.World.Create().
			SetName("Earth").
			SetSlug("earth").
			SetDescription("Planet Earth").
			Save(ctx)
		if err != nil {
			return err
		}
	}

	for _, fix := range fixtures {
		legacyID := parseInt(fix.PK)
		slug := parseString(fix.Fields["slug"])

		// Check if already exists (idempotent)
		existingContinent, err := m.client.Continent.Query().
			Where(continent.Slug(slug)).
			Only(ctx)
		if err == nil {
			// Already exists, store mapping
			m.idMapping.SetContinent(legacyID, existingContinent.ID)
			continue
		}
		if !ent.IsNotFound(err) {
			return err
		}

		newContinent, err := m.client.Continent.Create().
			SetName(parseString(fix.Fields["name"])).
			SetSlug(slug).
			SetDescription(parseString(fix.Fields["description"])).
			SetWorld(worldEntity).
			Save(ctx)
		if err != nil {
			return err
		}

		// Store ID mapping
		m.idMapping.SetContinent(legacyID, newContinent.ID)
	}

	return nil
}

// migrateCountries migrates countries from core_country.json
func (m *Migrator) migrateCountries(ctx context.Context, fixturesDir string) error {
	fixtures, err := loadFixtures(filepath.Join(fixturesDir, "core_country.json"))
	if err != nil {
		return err
	}

	for _, fix := range fixtures {
		legacyID := parseInt(fix.PK)
		slug := parseString(fix.Fields["slug"])

		// Check if already exists
		existingCountry, err := m.client.Country.Query().
			Where(country.Slug(slug)).
			Only(ctx)
		if err == nil {
			// Already exists, store mapping
			m.idMapping.SetCountry(legacyID, existingCountry.ID)
			continue
		}
		if !ent.IsNotFound(err) {
			return err
		}

		// Find parent continent by ID from legacy system
		continentLegacyID := parseInt(fix.Fields["continent"])
		var parentContinent *ent.Continent

		// First try to get from ID mapping
		if continentUUID, ok := m.idMapping.GetContinent(continentLegacyID); ok {
			parentContinent, err = m.client.Continent.Get(ctx, continentUUID)
			if err != nil {
				logger.Warn("Continent UUID lookup failed", logger.Err(err))
			}
		}

		// Fallback: Map legacy continent IDs to slugs
		if parentContinent == nil {
			continentSlugMap := map[int]string{
				1: "africa",
				2: "europe",
				3: "asia",
				4: "north-america",
				5: "south-america",
				6: "oceania",
			}

			if continentSlug, ok := continentSlugMap[continentLegacyID]; ok {
				parentContinent, err = m.client.Continent.Query().
					Where(continent.Slug(continentSlug)).
					Only(ctx)
				if err != nil {
					logger.Warn("Continent not found by slug", logger.Int("legacy_id", continentLegacyID))
				}
			}
		}

		// Last fallback: use first available continent
		if parentContinent == nil {
			parentContinent, _ = m.client.Continent.Query().First(ctx)
		}

		if parentContinent == nil {
			logger.Warn("No continent available, skipping country", logger.String("slug", slug))
			continue
		}

		// Get country code (use fixture code or derive from name)
		code := parseString(fix.Fields["code"])
		if code == "" {
			// Map country names to ISO codes
			countryCodeMap := map[string]string{
				"kenya":    "KE",
				"uganda":   "UG",
				"tanzania": "TZ",
				"rwanda":   "RW",
				"ethiopia": "ET",
			}
			if c, ok := countryCodeMap[slug]; ok {
				code = c
			} else {
				// Use first 2 letters of slug as fallback
				code = slug
				if len(code) > 2 {
					code = code[:2]
				}
				code = strings.ToUpper(code)
			}
		}

		newCountry, err := m.client.Country.Create().
			SetName(parseString(fix.Fields["name"])).
			SetSlug(slug).
			SetCode(code).
			SetDescription(parseString(fix.Fields["description"])).
			SetContinent(parentContinent).
			Save(ctx)
		if err != nil {
			return err
		}

		// Store ID mapping
		m.idMapping.SetCountry(legacyID, newCountry.ID)
	}

	return nil
}

// migrateLocations migrates locations from core_location.json
func (m *Migrator) migrateLocations(ctx context.Context, fixturesDir string) error {
	fixtures, err := loadFixtures(filepath.Join(fixturesDir, "core_location.json"))
	if err != nil {
		return err
	}

	for _, fix := range fixtures {
		legacyID := parseInt(fix.PK)
		slug := parseString(fix.Fields["slug"])

		// Check if already exists
		existingLocation, err := m.client.Location.Query().
			Where(location.Slug(slug)).
			Only(ctx)
		if err == nil {
			// Already exists, store mapping
			m.idMapping.SetLocation(legacyID, existingLocation.ID)
			continue
		}
		if !ent.IsNotFound(err) {
			return err
		}

		// Find country by legacy ID
		countryLegacyID := parseInt(fix.Fields["country"])
		var parentCountry *ent.Country

		// First try to get from ID mapping
		if countryUUID, ok := m.idMapping.GetCountry(countryLegacyID); ok {
			parentCountry, err = m.client.Country.Get(ctx, countryUUID)
			if err != nil {
				logger.Warn("Country UUID lookup failed", logger.Err(err))
			}
		}

		// Fallback: Map legacy country IDs to slugs
		if parentCountry == nil {
			countrySlugMap := map[int]string{
				1: "kenya",
				2: "uganda",
				3: "tanzania",
			}

			if countrySlug, ok := countrySlugMap[countryLegacyID]; ok {
				parentCountry, err = m.client.Country.Query().
					Where(country.Slug(countrySlug)).
					Only(ctx)
				if err != nil {
					logger.Warn("Country not found by slug", logger.Int("legacy_id", countryLegacyID))
				}
			}
		}

		// Last fallback: use first available country
		if parentCountry == nil {
			parentCountry, _ = m.client.Country.Query().First(ctx)
		}

		if parentCountry == nil {
			logger.Warn("No country available, skipping location", logger.String("slug", slug))
			continue
		}

		newLocation, err := m.client.Location.Create().
			SetName(parseString(fix.Fields["name"])).
			SetSlug(slug).
			SetAddress(parseString(fix.Fields["address"])).
			SetCity(parseString(fix.Fields["city"])).
			SetCountry(parentCountry).
			Save(ctx)
		if err != nil {
			return err
		}

		// Store ID mapping
		m.idMapping.SetLocation(legacyID, newLocation.ID)
	}

	return nil
}
