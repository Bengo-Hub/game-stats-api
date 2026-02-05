package migration

import (
	"context"
	"path/filepath"

	"github.com/bengobox/game-stats-api/ent"
	"github.com/bengobox/game-stats-api/ent/game"
	"github.com/bengobox/game-stats-api/ent/spiritscore"
	"github.com/bengobox/game-stats-api/ent/team"
	"github.com/bengobox/game-stats-api/ent/user"
	"github.com/bengobox/game-stats-api/internal/pkg/logger"
)

// migrateSpiritScores migrates spirit scores from games_spiritscore.json
func (m *Migrator) migrateSpiritScores(ctx context.Context, fixturesDir string) error {
	fixtures, err := loadFixtures(filepath.Join(fixturesDir, "games_spiritscore.json"))
	if err != nil {
		return err
	}

	if len(fixtures) == 0 {
		logger.Info("No spirit score fixtures found")
		return nil
	}

	// Get or create a system user for migration (submitted_by is required)
	systemUser, err := m.getOrCreateSystemUser(ctx)
	if err != nil {
		logger.Error("Failed to get/create system user for spirit scores", logger.Err(err))
		return err
	}

	migrated := 0
	skipped := 0

	for _, fix := range fixtures {
		gameLegacyID := parseInt(fix.Fields["game"])
		teamLegacyID := parseInt(fix.Fields["team"])
		scoredByLegacyID := parseInt(fix.Fields["scored_by"])

		gameUUID, ok1 := m.idMapping.GetGame(gameLegacyID)
		teamUUID, ok2 := m.idMapping.GetTeam(teamLegacyID)
		scoredByUUID, ok3 := m.idMapping.GetTeam(scoredByLegacyID)

		if !ok1 || !ok2 || !ok3 {
			skipped++
			continue
		}

		// Check if spirit score already exists
		exists, err := m.client.SpiritScore.Query().
			Where(
				spiritscore.HasGameWith(game.IDEQ(gameUUID)),
				spiritscore.HasTeamWith(team.IDEQ(teamUUID)),
				spiritscore.HasScoredByTeamWith(team.IDEQ(scoredByUUID)),
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

		teamEntity, err := m.client.Team.Get(ctx, teamUUID)
		if err != nil {
			skipped++
			continue
		}

		scoredByTeam, err := m.client.Team.Get(ctx, scoredByUUID)
		if err != nil {
			skipped++
			continue
		}

		// Create spirit score
		spiritScore, err := m.client.SpiritScore.Create().
			SetGame(gameEntity).
			SetTeam(teamEntity).
			SetScoredByTeam(scoredByTeam).
			SetSubmittedBy(systemUser).
			SetRulesKnowledge(parseInt(fix.Fields["rules_knowledge"])).
			SetFoulsBodyContact(parseInt(fix.Fields["fouls_body_contact"])).
			SetFairMindedness(parseInt(fix.Fields["fair_mindedness"])).
			SetAttitude(parseInt(fix.Fields["attitude"])).
			SetCommunication(parseInt(fix.Fields["communication"])).
			Save(ctx)
		if err != nil {
			logger.Error("Failed to create spirit score", logger.Err(err))
			skipped++
			continue
		}

		// Create MVP nominations if present
		m.createMVPNominations(ctx, fix, spiritScore)

		// Create Spirit nominations if present
		m.createSpiritNominations(ctx, fix, spiritScore)

		migrated++
	}

	logger.Info("Spirit score migration complete",
		logger.Int("migrated", migrated),
		logger.Int("skipped", skipped))

	return nil
}

// getOrCreateSystemUser gets an existing admin user or creates a system user for migration purposes
func (m *Migrator) getOrCreateSystemUser(ctx context.Context) (*ent.User, error) {
	// First, try to find an existing admin user from migration
	adminUser, err := m.client.User.Query().
		Where(user.Role("admin")).
		First(ctx)
	if err == nil {
		logger.Info("Using existing admin user for spirit score migration", logger.String("email", adminUser.Email))
		return adminUser, nil
	}

	// Try to find existing system user
	systemUser, err := m.client.User.Query().
		Where(user.Email("migration@system.local")).
		Only(ctx)
	if err == nil {
		return systemUser, nil
	}
	if !ent.IsNotFound(err) {
		return nil, err
	}

	// Create system user for migration as last resort
	systemUser, err = m.client.User.Create().
		SetEmail("migration@system.local").
		SetPasswordHash("$migration$not-a-real-password").
		SetName("Migration System User").
		SetRole("system").
		SetIsActive(false). // Disabled - cannot login
		Save(ctx)
	if err != nil {
		return nil, err
	}

	logger.Info("Created system user for migration", logger.String("email", systemUser.Email))
	return systemUser, nil
}

// createMVPNominations creates MVP nominations for a spirit score
func (m *Migrator) createMVPNominations(ctx context.Context, fix DjangoFixture, spiritScore *ent.SpiritScore) {
	// MVP female nomination
	if mvpFemaleLegacyID := parseInt(fix.Fields["mvp_female_nomination"]); mvpFemaleLegacyID > 0 {
		if playerUUID, ok := m.idMapping.GetPlayer(mvpFemaleLegacyID); ok {
			player, err := m.client.Player.Get(ctx, playerUUID)
			if err == nil {
				_, err = m.client.MVP_Nomination.Create().
					SetSpiritScore(spiritScore).
					SetPlayer(player).
					SetCategory("female").
					Save(ctx)
				if err != nil {
					logger.Warn("Failed to create MVP female nomination", logger.Err(err))
				}
			}
		}
	}

	// MVP male nomination
	if mvpMaleLegacyID := parseInt(fix.Fields["mvp_male_nomination"]); mvpMaleLegacyID > 0 {
		if playerUUID, ok := m.idMapping.GetPlayer(mvpMaleLegacyID); ok {
			player, err := m.client.Player.Get(ctx, playerUUID)
			if err == nil {
				_, err = m.client.MVP_Nomination.Create().
					SetSpiritScore(spiritScore).
					SetPlayer(player).
					SetCategory("male").
					Save(ctx)
				if err != nil {
					logger.Warn("Failed to create MVP male nomination", logger.Err(err))
				}
			}
		}
	}
}

// createSpiritNominations creates spirit nominations for a spirit score
func (m *Migrator) createSpiritNominations(ctx context.Context, fix DjangoFixture, spiritScore *ent.SpiritScore) {
	// Spirit female nomination
	if spiritFemaleLegacyID := parseInt(fix.Fields["spirit_female_nomination"]); spiritFemaleLegacyID > 0 {
		if playerUUID, ok := m.idMapping.GetPlayer(spiritFemaleLegacyID); ok {
			player, err := m.client.Player.Get(ctx, playerUUID)
			if err == nil {
				_, err = m.client.SpiritNomination.Create().
					SetSpiritScore(spiritScore).
					SetPlayer(player).
					SetCategory("female").
					Save(ctx)
				if err != nil {
					logger.Warn("Failed to create spirit female nomination", logger.Err(err))
				}
			}
		}
	}

	// Spirit male nomination
	if spiritMaleLegacyID := parseInt(fix.Fields["spirit_male_nomination"]); spiritMaleLegacyID > 0 {
		if playerUUID, ok := m.idMapping.GetPlayer(spiritMaleLegacyID); ok {
			player, err := m.client.Player.Get(ctx, playerUUID)
			if err == nil {
				_, err = m.client.SpiritNomination.Create().
					SetSpiritScore(spiritScore).
					SetPlayer(player).
					SetCategory("male").
					Save(ctx)
				if err != nil {
					logger.Warn("Failed to create spirit male nomination", logger.Err(err))
				}
			}
		}
	}
}
