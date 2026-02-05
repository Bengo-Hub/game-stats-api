package migration

import (
	"context"
	"path/filepath"

	"github.com/bengobox/game-stats-api/ent"
	"github.com/bengobox/game-stats-api/ent/player"
	"github.com/bengobox/game-stats-api/ent/team"
	"github.com/bengobox/game-stats-api/internal/pkg/logger"
)

// migrateTeams migrates teams from games_team.json
func (m *Migrator) migrateTeams(ctx context.Context, fixturesDir string) error {
	fixtures, err := loadFixtures(filepath.Join(fixturesDir, "games_team.json"))
	if err != nil {
		return err
	}

	for _, fix := range fixtures {
		legacyID := parseInt(fix.PK)
		name := parseString(fix.Fields["name"])

		// Check if already exists
		existingTeam, err := m.client.Team.Query().
			Where(team.Name(name)).
			Only(ctx)
		if err == nil {
			// Already exists, store mapping
			m.idMapping.SetTeam(legacyID, existingTeam.ID)
			continue
		}
		if !ent.IsNotFound(err) {
			return err
		}

		// Find division pool for team (if origin maps to a division)
		originLegacyID := parseInt(fix.Fields["origin"])
		var divisionPool *ent.DivisionPool

		if divisionUUID, ok := m.idMapping.GetDivision(originLegacyID); ok {
			divisionPool, _ = m.client.DivisionPool.Get(ctx, divisionUUID)
		}

		// Fallback: use first available division pool
		if divisionPool == nil {
			divisionPool, _ = m.client.DivisionPool.Query().First(ctx)
		}

		// Find home location (use first available)
		homeLocation, _ := m.client.Location.Query().First(ctx)

		// Create new team
		creator := m.client.Team.Create().
			SetName(name).
			SetNillableInitialSeed(intPtr(parseInt(fix.Fields["initial_seed"])))

		if divisionPool != nil {
			creator.SetDivisionPool(divisionPool)
		}
		if homeLocation != nil {
			creator.SetHomeLocation(homeLocation)
		}

		newTeam, err := creator.Save(ctx)
		if err != nil {
			logger.Error("Failed to create team", logger.Err(err), logger.String("name", name))
			return err
		}

		// Store mapping for later use
		m.idMapping.SetTeam(legacyID, newTeam.ID)
	}

	return nil
}

// migratePlayers migrates players from games_player.json
func (m *Migrator) migratePlayers(ctx context.Context, fixturesDir string) error {
	fixtures, err := loadFixtures(filepath.Join(fixturesDir, "games_player.json"))
	if err != nil {
		return err
	}

	for _, fix := range fixtures {
		legacyID := parseInt(fix.PK)
		name := parseString(fix.Fields["name"])

		// Skip placeholder players named "A"
		if name == "A" || name == "" {
			continue
		}

		// Check if already exists by name and team
		teamLegacyID := parseInt(fix.Fields["team"])
		teamUUID, teamOK := m.idMapping.GetTeam(teamLegacyID)

		if teamOK {
			// Check for existing player with same name on same team
			existingPlayer, err := m.client.Player.Query().
				Where(
					player.Name(name),
					player.HasTeamWith(team.ID(teamUUID)),
				).
				Only(ctx)
			if err == nil {
				m.idMapping.SetPlayer(legacyID, existingPlayer.ID)
				continue
			}
		}

		// Also check by name only (for idempotency)
		existingPlayer, err := m.client.Player.Query().
			Where(player.Name(name)).
			Only(ctx)
		if err == nil {
			m.idMapping.SetPlayer(legacyID, existingPlayer.ID)
			continue
		}
		if !ent.IsNotFound(err) && !ent.IsNotSingular(err) {
			// Multiple players with same name - need to be more specific
			logger.Warn("Multiple players with same name", logger.String("name", name))
		}

		// Find team
		if !teamOK {
			logger.Warn("Team not found for player",
				logger.String("player", name),
				logger.Int("team_id", teamLegacyID))
			continue
		}

		playerTeam, err := m.client.Team.Get(ctx, teamUUID)
		if err != nil {
			logger.Warn("Team lookup failed", logger.Err(err))
			continue
		}

		// Parse gender
		gender := parseString(fix.Fields["gender"])
		if gender == "" {
			gender = "M" // Default to male if not specified
		}

		// Create player
		newPlayer, err := m.client.Player.Create().
			SetName(name).
			SetGender(gender).
			SetTeam(playerTeam).
			Save(ctx)
		if err != nil {
			logger.Error("Failed to create player", logger.Err(err), logger.String("name", name))
			continue
		}

		m.idMapping.SetPlayer(legacyID, newPlayer.ID)
	}

	logger.Info("Players migration complete",
		logger.Int("total_mapped", len(m.idMapping.Players)))

	return nil
}

func intPtr(i int) *int {
	return &i
}
