package migration

import (
	"context"
	"path/filepath"

	"github.com/bengobox/game-stats-api/internal/pkg/logger"
	"github.com/google/uuid"
)

// migrateActivityLogs migrates activity/audit logs from audit_log.json
func (m *Migrator) migrateActivityLogs(ctx context.Context, fixturesDir string) error {
	fixtures, err := loadFixtures(filepath.Join(fixturesDir, "audit_log.json"))
	if err != nil {
		// If audit logs don't exist, it's not a fatal error
		logger.Warn("Audit log fixtures not found, skipping", logger.Err(err))
		return nil
	}

	if len(fixtures) == 0 {
		logger.Info("No activity log fixtures found")
		return nil
	}

	migrated := 0
	skipped := 0

	for _, fix := range fixtures {
		userID := parseInt(fix.Fields["user"])
		action := parseString(fix.Fields["action"])
		objectType := parseString(fix.Fields["object_type"])
		objectID := parseString(fix.Fields["object_id"])
		timestamp := parseTime(fix.Fields["timestamp"])
		details := parseString(fix.Fields["details"])

		// Map legacy user ID to new UUID
		newUserID, ok := m.idMapping.GetUser(userID)
		if !ok {
			// If user doesn't exist, we skip the log
			skipped++
			continue
		}

		// Parse objectID as UUID
		entityID, err := uuid.Parse(objectID)
		if err != nil {
			// If not a valid UUID, we might need a default or skip
			entityID = uuid.Nil
		}

		// Create audit log entry
		_, err = m.client.AuditLog.Create().
			SetUserID(newUserID).
			SetUsername("system_migration").
			SetAction(action).
			SetEntityType(objectType).
			SetEntityID(entityID).
			SetCreatedAt(timestamp).
			SetReason(details).
			SetChanges(map[string]interface{}{"migrated": true}).
			Save(ctx)

		if err != nil {
			logger.Warn("Failed to create audit log entry", logger.Err(err))
			skipped++
			continue
		}

		migrated++
	}

	logger.Info("Activity log migration complete",
		logger.Int("migrated", migrated),
		logger.Int("skipped", skipped))

	return nil
}
