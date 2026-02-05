package repository

import (
	"context"
	"fmt"
	"time"

	"entgo.io/ent/dialect/sql"
	"github.com/bengobox/game-stats-api/ent"
	"github.com/bengobox/game-stats-api/ent/auditlog"
	"github.com/bengobox/game-stats-api/internal/domain/audit"
	"github.com/google/uuid"
)

// AuditRepository implements audit log persistence using Ent
type AuditRepository struct {
	client *ent.Client
}

// NewAuditRepository creates a new audit repository
func NewAuditRepository(client *ent.Client) *AuditRepository {
	return &AuditRepository{client: client}
}

// Create stores a new audit log entry
func (r *AuditRepository) Create(ctx context.Context, log *audit.AuditLog) error {
	// Convert domain changes to generic map for Ent storage
	changesMap := make(map[string]interface{})
	for k, v := range log.Changes {
		changesMap[k] = map[string]interface{}{
			"old_value": v.OldValue,
			"new_value": v.NewValue,
		}
	}

	builder := r.client.AuditLog.Create().
		SetEntityType(log.EntityType).
		SetEntityID(log.EntityID).
		SetAction(string(log.Action)).
		SetUserID(log.UserID).
		SetUsername(log.Username).
		SetChanges(changesMap).
		SetReason(log.Reason)

	// Set optional fields
	if log.IPAddress != "" {
		builder.SetIPAddress(log.IPAddress)
	}
	if log.UserAgent != "" {
		builder.SetUserAgent(log.UserAgent)
	}

	_, err := builder.Save(ctx)
	if err != nil {
		return fmt.Errorf("failed to create audit log: %w", err)
	}

	return nil
}

// GetByEntity retrieves audit logs for a specific entity
func (r *AuditRepository) GetByEntity(ctx context.Context, entityType string, entityID uuid.UUID) ([]*audit.AuditLog, error) {
	entLogs, err := r.client.AuditLog.Query().
		Where(
			auditlog.EntityType(entityType),
			auditlog.EntityID(entityID),
		).
		Order(auditlog.ByCreatedAt(sql.OrderDesc())).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get audit logs by entity: %w", err)
	}

	return r.convertEntLogsToDomain(entLogs), nil
}

// GetByUser retrieves recent audit logs for a user
func (r *AuditRepository) GetByUser(ctx context.Context, userID uuid.UUID, limit int) ([]*audit.AuditLog, error) {
	entLogs, err := r.client.AuditLog.Query().
		Where(auditlog.UserID(userID)).
		Order(auditlog.ByCreatedAt(sql.OrderDesc())).
		Limit(limit).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get audit logs by user: %w", err)
	}

	return r.convertEntLogsToDomain(entLogs), nil
}

// GetRecent retrieves the most recent audit logs
func (r *AuditRepository) GetRecent(ctx context.Context, limit int) ([]*audit.AuditLog, error) {
	entLogs, err := r.client.AuditLog.Query().
		Order(auditlog.ByCreatedAt(sql.OrderDesc())).
		Limit(limit).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent audit logs: %w", err)
	}

	return r.convertEntLogsToDomain(entLogs), nil
}

// convertEntLogsToDomain converts Ent AuditLog entities to domain models
func (r *AuditRepository) convertEntLogsToDomain(entLogs []*ent.AuditLog) []*audit.AuditLog {
	result := make([]*audit.AuditLog, len(entLogs))
	for i, entLog := range entLogs {
		result[i] = r.convertEntLogToDomain(entLog)
	}
	return result
}

// convertEntLogToDomain converts a single Ent AuditLog to domain model
func (r *AuditRepository) convertEntLogToDomain(entLog *ent.AuditLog) *audit.AuditLog {
	// Convert stored map back to domain ChangeEntry format
	changes := make(map[string]audit.ChangeEntry)
	for k, v := range entLog.Changes {
		if changeMap, ok := v.(map[string]interface{}); ok {
			changes[k] = audit.ChangeEntry{
				OldValue: changeMap["old_value"],
				NewValue: changeMap["new_value"],
			}
		}
	}

	return &audit.AuditLog{
		ID:         entLog.ID,
		EntityType: entLog.EntityType,
		EntityID:   entLog.EntityID,
		Action:     audit.Action(entLog.Action),
		UserID:     entLog.UserID,
		Username:   entLog.Username,
		Changes:    changes,
		Reason:     entLog.Reason,
		IPAddress:  entLog.IPAddress,
		UserAgent:  entLog.UserAgent,
		CreatedAt:  entLog.CreatedAt,
	}
}

// InMemoryAuditRepository provides an in-memory implementation for testing
type InMemoryAuditRepository struct {
	logs []*audit.AuditLog
}

// NewInMemoryAuditRepository creates an in-memory audit repository
func NewInMemoryAuditRepository() *InMemoryAuditRepository {
	return &InMemoryAuditRepository{
		logs: make([]*audit.AuditLog, 0),
	}
}

// Create stores a log in memory
func (r *InMemoryAuditRepository) Create(_ context.Context, log *audit.AuditLog) error {
	log.CreatedAt = time.Now()
	r.logs = append(r.logs, log)
	return nil
}

// GetByEntity retrieves logs for an entity
func (r *InMemoryAuditRepository) GetByEntity(_ context.Context, entityType string, entityID uuid.UUID) ([]*audit.AuditLog, error) {
	var result []*audit.AuditLog
	for _, log := range r.logs {
		if log.EntityType == entityType && log.EntityID == entityID {
			result = append(result, log)
		}
	}
	return result, nil
}

// GetByUser retrieves logs for a user
func (r *InMemoryAuditRepository) GetByUser(_ context.Context, userID uuid.UUID, limit int) ([]*audit.AuditLog, error) {
	var result []*audit.AuditLog
	count := 0

	// Reverse iteration for most recent first
	for i := len(r.logs) - 1; i >= 0 && count < limit; i-- {
		if r.logs[i].UserID == userID {
			result = append(result, r.logs[i])
			count++
		}
	}

	return result, nil
}

// GetRecent retrieves recent logs
func (r *InMemoryAuditRepository) GetRecent(_ context.Context, limit int) ([]*audit.AuditLog, error) {
	start := len(r.logs) - limit
	if start < 0 {
		start = 0
	}

	result := make([]*audit.AuditLog, 0, limit)
	for i := len(r.logs) - 1; i >= start; i-- {
		result = append(result, r.logs[i])
	}

	return result, nil
}
