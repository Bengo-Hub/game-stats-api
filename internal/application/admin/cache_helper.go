package admin

import (
	"context"
	"fmt"

	"github.com/bengobox/game-stats-api/ent"
	"github.com/bengobox/game-stats-api/internal/domain/audit"
	"github.com/bengobox/game-stats-api/internal/infrastructure/cache"
	"github.com/google/uuid"
)

// InvalidateCaches invalidates game-related caches after admin updates
func InvalidateCaches(ctx context.Context, c *cache.RedisClient, gameID uuid.UUID, g *ent.Game) error {
	// Delete specific game cache
	gameKey := cache.CacheKey("game", gameID.String())
	if err := c.Delete(ctx, gameKey); err != nil {
		return fmt.Errorf("failed to delete game cache: %w", err)
	}

	// Delete division standings cache
	if g != nil && g.Edges.DivisionPool != nil {
		standingsPattern := cache.CacheKey("standings", "division", g.Edges.DivisionPool.ID.String())
		if err := c.DeletePattern(ctx, standingsPattern); err != nil {
			return fmt.Errorf("failed to delete standings cache: %w", err)
		}

		// Delete event stats cache if this is tournament game
		if g.Edges.GameRound != nil && g.Edges.GameRound.Edges.Event != nil {
			eventStatsPattern := cache.CacheKey("event", "stats", g.Edges.GameRound.Edges.Event.ID.String(), "*")
			if err := c.DeletePattern(ctx, eventStatsPattern); err != nil {
				return fmt.Errorf("failed to delete event stats cache: %w", err)
			}
		}
	}

	return nil
}

// CreateAuditLog creates an audit trail entry for admin modifications
func CreateAuditLog(
	ctx context.Context,
	auditRepo audit.Repository,
	entityType string,
	entityID uuid.UUID,
	action audit.Action,
	userID uuid.UUID,
	username string,
	changes map[string]audit.ChangeEntry,
	reason string,
	ipAddress string,
	userAgent string,
) (uuid.UUID, error) {
	auditLog := audit.NewAuditLog(
		entityType,
		entityID,
		action,
		userID,
		username,
	)

	for field, change := range changes {
		auditLog.AddChange(field, change.OldValue, change.NewValue)
	}

	auditLog.Reason = reason
	auditLog.SetMetadata(ipAddress, userAgent)

	if err := auditRepo.Create(ctx, auditLog); err != nil {
		return uuid.Nil, fmt.Errorf("failed to create audit log: %w", err)
	}

	return auditLog.ID, nil
}
