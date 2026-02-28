package migration

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/bengobox/game-stats-api/ent"
	"github.com/bengobox/game-stats-api/ent/game"
	"github.com/bengobox/game-stats-api/ent/player"
	"github.com/bengobox/game-stats-api/ent/scoring"
	"github.com/bengobox/game-stats-api/ent/user"
	"github.com/bengobox/game-stats-api/internal/pkg/logger"
	"github.com/google/uuid"
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

		// Extract team IDs
		homeTeamLegacyID := parseInt(fix.Fields["home_team"])
		awayTeamLegacyID := parseInt(fix.Fields["away_team"])

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

		// Extract scores
		homeScore := parseInt(fix.Fields["home_team_score"])
		awayScore := parseInt(fix.Fields["away_team_score"])

		// Parse scheduled time
		scheduledTime := parseTime(fix.Fields["date"])
		if scheduledTime.IsZero() {
			scheduledTime = time.Now()
		}

		// Get required relationships: division_pool and field_location
		divisionLegacyID := parseInt(fix.Fields["division_pool"])

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

		// Find default scorekeeper
		scorekeeper, _ := m.client.User.Query().
			Where(user.Email("scorekeeper@test.com")).
			Only(ctx)

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

		if scorekeeper != nil {
			gameBuilder.SetScorekeeper(scorekeeper)
		}

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

	logger.Info("Recalculating game scores based on player goals...")
	recalcCtx := context.Background()

	// Recalculate all game scores
	games, err := m.client.Game.Query().
		WithHomeTeam().
		WithAwayTeam().
		WithScores(func(q *ent.ScoringQuery) {
			q.WithPlayer(func(pq *ent.PlayerQuery) {
				pq.WithTeams()
			})
		}).
		All(recalcCtx)

	if err != nil {
		logger.Error("Failed to fetch games for score recalculation", logger.Err(err))
		return err
	}

	recalculated := 0
	for _, g := range games {
		if g.Edges.HomeTeam == nil || g.Edges.AwayTeam == nil {
			continue
		}

		homeScore := 0
		awayScore := 0

		for _, s := range g.Edges.Scores {
			if s.Edges.Player == nil {
				continue
			}

			// Figure out which team the player was scoring for
			isHome := false
			isAway := false
			if s.TeamID != uuid.Nil {
				if s.TeamID == g.Edges.HomeTeam.ID {
					isHome = true
				} else if s.TeamID == g.Edges.AwayTeam.ID {
					isAway = true
				}
			} else {
				for _, pt := range s.Edges.Player.Edges.Teams {
					if pt.ID == g.Edges.HomeTeam.ID {
						isHome = true
						break
					}
					if pt.ID == g.Edges.AwayTeam.ID {
						isAway = true
						break
					}
				}
			}

			if isHome {
				homeScore += s.Goals
			} else if isAway {
				awayScore += s.Goals
			}
		}

		// Update game if scores changed or were initially 0
		if g.HomeTeamScore != homeScore || g.AwayTeamScore != awayScore || (g.HomeTeamScore == 0 && g.AwayTeamScore == 0 && (homeScore > 0 || awayScore > 0)) {
			updatedGame, err := m.client.Game.UpdateOne(g).
				SetHomeTeamScore(homeScore).
				SetAwayTeamScore(awayScore).
				Save(recalcCtx)
			if err != nil {
				logger.Error("Failed to update game score during recalculation", logger.Err(err), logger.String("game_id", g.ID.String()))
			} else {
				recalculated++

				// Cache the updated game score data if redis is available
				if m.cache != nil {
					cacheKey := fmt.Sprintf("game-stats:game:%s", updatedGame.ID.String())
					ttl := 5 * time.Minute // Default live game cache

					if updatedGame.Status == "completed" || updatedGame.Status == "final" {
						ttl = 24 * time.Hour
					}

					if err := m.cache.SetJSON(recalcCtx, cacheKey, updatedGame, ttl); err != nil {
						logger.Warn("Failed to cache recalculated game", logger.Err(err), logger.String("game_id", updatedGame.ID.String()))
					}
				}
			}
		}
	}

	logger.Info("Game score recalculation complete", logger.Int("games_updated", recalculated))

	return nil
}
