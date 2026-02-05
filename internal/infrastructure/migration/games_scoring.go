package migration

import (
	"context"
	"path/filepath"
	"time"

	"github.com/bengobox/game-stats-api/ent"
	"github.com/bengobox/game-stats-api/ent/game"
	"github.com/bengobox/game-stats-api/ent/player"
	"github.com/bengobox/game-stats-api/ent/scoring"
	"github.com/bengobox/game-stats-api/internal/pkg/logger"
)

// migrateGames migrates games from games_game.json
func (m *Migrator) migrateGames(ctx context.Context, fixturesDir string) error {
	fixtures, err := loadFixtures(filepath.Join(fixturesDir, "games_game.json"))
	if err != nil {
		return err
	}

	for _, fix := range fixtures {
		legacyID := parseInt(fix.PK)
		name := parseString(fix.Fields["name"])

		// Check if already exists
		existingGame, err := m.client.Game.Query().
			Where(game.Name(name)).
			Only(ctx)
		if err == nil {
			m.idMapping.SetGame(legacyID, existingGame.ID)
			continue
		}
		if !ent.IsNotFound(err) {
			return err
		}

		// Extract team IDs - handle both old and new field names
		var homeTeamLegacyID, awayTeamLegacyID int

		// Try both field name formats
		if val, ok := fix.Fields["team1"]; ok {
			homeTeamLegacyID = parseInt(val)
		} else if val, ok := fix.Fields["home_team"]; ok {
			homeTeamLegacyID = parseInt(val)
		}

		if val, ok := fix.Fields["team2"]; ok {
			awayTeamLegacyID = parseInt(val)
		} else if val, ok := fix.Fields["away_team"]; ok {
			awayTeamLegacyID = parseInt(val)
		}

		homeTeamUUID, ok1 := m.idMapping.GetTeam(homeTeamLegacyID)
		awayTeamUUID, ok2 := m.idMapping.GetTeam(awayTeamLegacyID)

		if !ok1 || !ok2 {
			logger.Warn("Teams not found for game",
				logger.String("game", name),
				logger.Int("home_team_id", homeTeamLegacyID),
				logger.Int("away_team_id", awayTeamLegacyID))
			continue
		}

		homeTeam, err := m.client.Team.Get(ctx, homeTeamUUID)
		if err != nil {
			logger.Warn("Home team lookup failed", logger.Err(err))
			continue
		}

		awayTeam, err := m.client.Team.Get(ctx, awayTeamUUID)
		if err != nil {
			logger.Warn("Away team lookup failed", logger.Err(err))
			continue
		}

		// Extract scores - handle both field name formats
		var homeScore, awayScore int
		if val, ok := fix.Fields["team1_score"]; ok {
			homeScore = parseInt(val)
		} else if val, ok := fix.Fields["home_team_score"]; ok {
			homeScore = parseInt(val)
		}

		if val, ok := fix.Fields["team2_score"]; ok {
			awayScore = parseInt(val)
		} else if val, ok := fix.Fields["away_team_score"]; ok {
			awayScore = parseInt(val)
		}

		// Parse scheduled time - handle multiple field names
		var scheduledTime time.Time
		if val, ok := fix.Fields["start_time"]; ok {
			scheduledTime = parseTime(val)
		} else if val, ok := fix.Fields["date"]; ok {
			scheduledTime = parseTime(val)
		} else {
			scheduledTime = time.Now()
		}

		// Get required relationships: division_pool and field_location
		divisionLegacyID := 0
		if val, ok := fix.Fields["pool"]; ok {
			divisionLegacyID = parseInt(val)
		} else if val, ok := fix.Fields["division_pool"]; ok {
			divisionLegacyID = parseInt(val)
		}

		var division *ent.DivisionPool
		if divisionLegacyID > 0 {
			if divisionUUID, ok := m.idMapping.GetDivision(divisionLegacyID); ok {
				division, _ = m.client.DivisionPool.Get(ctx, divisionUUID)
			}
		}
		// Fallback: use first available division pool (required field)
		if division == nil {
			division, _ = m.client.DivisionPool.Query().First(ctx)
		}
		if division == nil {
			logger.Warn("No division pool available, skipping game", logger.String("name", name))
			continue
		}

		// Get field location
		fieldLegacyID := parseInt(fix.Fields["field"])
		var fieldLocation *ent.Field
		if fieldLegacyID > 0 {
			if fieldUUID, ok := m.idMapping.GetField(fieldLegacyID); ok {
				fieldLocation, _ = m.client.Field.Get(ctx, fieldUUID)
			}
		}
		// Fallback: use first available field (required field)
		if fieldLocation == nil {
			fieldLocation, _ = m.client.Field.Query().First(ctx)
		}
		if fieldLocation == nil {
			logger.Warn("No field available, skipping game", logger.String("name", name))
			continue
		}

		// Create game builder with required fields
		gameBuilder := m.client.Game.Create().
			SetName(name).
			SetStatus("completed"). // Assume legacy games are completed
			SetHomeTeam(homeTeam).
			SetAwayTeam(awayTeam).
			SetHomeTeamScore(homeScore).
			SetAwayTeamScore(awayScore).
			SetScheduledTime(scheduledTime).
			SetAllocatedTimeMinutes(60). // Default game duration
			SetDivisionPool(division).
			SetFieldLocation(fieldLocation)

		// Add game round if available
		if roundLegacyID := parseInt(fix.Fields["game_round"]); roundLegacyID > 0 {
			if roundUUID, ok := m.idMapping.GetGameRound(roundLegacyID); ok {
				round, err := m.client.GameRound.Get(ctx, roundUUID)
				if err == nil {
					gameBuilder.SetGameRound(round)
				}
			}
		}

		newGame, err := gameBuilder.Save(ctx)
		if err != nil {
			logger.Error("Failed to create game", logger.Err(err), logger.String("name", name))
			continue
		}

		m.idMapping.SetGame(legacyID, newGame.ID)
	}

	logger.Info("Games migration complete",
		logger.Int("total_mapped", len(m.idMapping.Games)))

	return nil
}

// migrateScoring migrates scoring records from games_scoring.json
func (m *Migrator) migrateScoring(ctx context.Context, fixturesDir string) error {
	fixtures, err := loadFixtures(filepath.Join(fixturesDir, "games_scoring.json"))
	if err != nil {
		return err
	}

	migrated := 0
	skipped := 0

	for _, fix := range fixtures {
		gameLegacyID := parseInt(fix.Fields["game"])
		playerLegacyID := parseInt(fix.Fields["player"])

		gameUUID, ok1 := m.idMapping.GetGame(gameLegacyID)
		playerUUID, ok2 := m.idMapping.GetPlayer(playerLegacyID)

		if !ok1 || !ok2 {
			skipped++
			continue
		}

		// Check if scoring record already exists
		exists, err := m.client.Scoring.Query().
			Where(
				scoring.HasGameWith(game.IDEQ(gameUUID)),
				scoring.HasPlayerWith(player.IDEQ(playerUUID)),
			).
			Exist(ctx)
		if err != nil {
			return err
		}
		if exists {
			skipped++
			continue
		}

		gameEntity, err := m.client.Game.Get(ctx, gameUUID)
		if err != nil {
			skipped++
			continue
		}

		playerEntity, err := m.client.Player.Get(ctx, playerUUID)
		if err != nil {
			skipped++
			continue
		}

		// Create scoring record
		_, err = m.client.Scoring.Create().
			SetGame(gameEntity).
			SetPlayer(playerEntity).
			SetGoals(parseInt(fix.Fields["goals"])).
			SetAssists(parseInt(fix.Fields["assists"])).
			SetBlocks(parseInt(fix.Fields["blocks"])).
			SetTurns(parseInt(fix.Fields["turns"])).
			Save(ctx)
		if err != nil {
			logger.Error("Failed to create scoring", logger.Err(err))
			skipped++
			continue
		}

		migrated++
	}

	logger.Info("Scoring migration complete",
		logger.Int("migrated", migrated),
		logger.Int("skipped", skipped))

	return nil
}
