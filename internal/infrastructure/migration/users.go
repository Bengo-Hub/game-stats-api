package migration

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/bengobox/game-stats-api/ent"
	"github.com/bengobox/game-stats-api/ent/user"
	"github.com/bengobox/game-stats-api/internal/pkg/auth"
	"github.com/bengobox/game-stats-api/internal/pkg/logger"
)

// Default password for migrated users - they should change this on first login
const defaultMigrationPassword = "ChangeMe123!"

// migrateUsers migrates users from authman_user.json
func (m *Migrator) migrateUsers(ctx context.Context, fixturesDir string) error {
	fixtures, err := loadFixtures(filepath.Join(fixturesDir, "authman_user.json"))
	if err != nil {
		return err
	}

	if len(fixtures) == 0 {
		logger.Info("No user fixtures found")
		return nil
	}

	// Hash the default password once for all users
	defaultPasswordHash, err := auth.HashPassword(defaultMigrationPassword)
	if err != nil {
		logger.Error("Failed to hash default password", logger.Err(err))
		return err
	}

	migrated := 0
	skipped := 0

	for _, fix := range fixtures {
		legacyID := parseInt(fix.PK)
		email := parseString(fix.Fields["email"])

		if email == "" {
			logger.Warn("User has no email, skipping", logger.Int("legacy_id", legacyID))
			skipped++
			continue
		}

		// Skip test spectator account
		if email == "man@test.com" {
			logger.Info("Skipping test spectator account", logger.String("email", email))
			skipped++
			continue
		}

		// Check if user already exists (idempotent)
		existingUser, err := m.client.User.Query().
			Where(user.Email(email)).
			Only(ctx)
		if err == nil {
			// Already exists, store mapping
			m.idMapping.SetUser(legacyID, existingUser.ID)
			skipped++
			continue
		}
		if !ent.IsNotFound(err) {
			return err
		}

		// Build user name from first_name + last_name or username
		firstName := parseString(fix.Fields["first_name"])
		lastName := parseString(fix.Fields["last_name"])
		username := parseString(fix.Fields["username"])

		name := strings.TrimSpace(firstName + " " + lastName)
		if name == "" {
			name = username
		}
		if name == "" {
			name = email // Fallback to email if no name
		}

		// Get role (map Django roles to new system)
		role := parseString(fix.Fields["role"])
		if role == "" {
			// Derive role from is_superuser/is_staff
			isSuperuser := parseBool(fix.Fields["is_superuser"])
			isStaff := parseBool(fix.Fields["is_staff"])
			if isSuperuser {
				role = "admin"
			} else if isStaff {
				role = "staff"
			} else {
				role = "user"
			}
		}

		// Get is_active status
		isActive := parseBool(fix.Fields["is_active"])

		// Create user with default password (users should change on first login)
		creator := m.client.User.Create().
			SetEmail(email).
			SetPasswordHash(defaultPasswordHash).
			SetName(name).
			SetRole(role).
			SetIsActive(isActive)

		// Set last login if available
		if lastLogin := fix.Fields["last_login"]; lastLogin != nil {
			lastLoginTime := parseTime(lastLogin)
			if !lastLoginTime.IsZero() {
				creator.SetLastLoginAt(lastLoginTime)
			}
		}

		newUser, err := creator.Save(ctx)
		if err != nil {
			logger.Error("Failed to create user",
				logger.Err(err),
				logger.String("email", email))
			skipped++
			continue
		}

		// Store ID mapping
		m.idMapping.SetUser(legacyID, newUser.ID)
		migrated++

		logger.Debug("Migrated user",
			logger.String("email", email),
			logger.String("role", role))
	}

	logger.Info("User migration complete",
		logger.Int("migrated", migrated),
		logger.Int("skipped", skipped))

	return nil
}

// parseBool parses a boolean value from interface{}
func parseBool(v interface{}) bool {
	if v == nil {
		return false
	}
	switch val := v.(type) {
	case bool:
		return val
	case string:
		return val == "true" || val == "1" || val == "True"
	case float64:
		return val != 0
	case int:
		return val != 0
	default:
		return false
	}
}
