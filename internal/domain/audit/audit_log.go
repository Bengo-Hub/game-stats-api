package audit

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// AuditLog represents a change tracking record
type AuditLog struct {
	ID         uuid.UUID              `json:"id"`
	EntityType string                 `json:"entity_type"` // "game", "score", "spirit_score", etc.
	EntityID   uuid.UUID              `json:"entity_id"`
	Action     Action                 `json:"action"` // "create", "update", "delete"
	UserID     uuid.UUID              `json:"user_id"`
	Username   string                 `json:"username"`
	Changes    map[string]ChangeEntry `json:"changes"`
	Reason     string                 `json:"reason,omitempty"`
	IPAddress  string                 `json:"ip_address,omitempty"`
	UserAgent  string                 `json:"user_agent,omitempty"`
	CreatedAt  time.Time              `json:"created_at"`
}

// Action represents the type of audit action
type Action string

const (
	ActionCreate Action = "create"
	ActionUpdate Action = "update"
	ActionDelete Action = "delete"
)

// ChangeEntry represents a single field change
type ChangeEntry struct {
	OldValue interface{} `json:"old_value"`
	NewValue interface{} `json:"new_value"`
}

// Repository defines audit log operations
type Repository interface {
	Create(ctx context.Context, log *AuditLog) error
	GetByEntity(ctx context.Context, entityType string, entityID uuid.UUID) ([]*AuditLog, error)
	GetByUser(ctx context.Context, userID uuid.UUID, limit int) ([]*AuditLog, error)
	GetRecent(ctx context.Context, limit int) ([]*AuditLog, error)
}

// NewAuditLog creates a new audit log entry
func NewAuditLog(entityType string, entityID uuid.UUID, action Action, userID uuid.UUID, username string) *AuditLog {
	return &AuditLog{
		ID:         uuid.New(),
		EntityType: entityType,
		EntityID:   entityID,
		Action:     action,
		UserID:     userID,
		Username:   username,
		Changes:    make(map[string]ChangeEntry),
		CreatedAt:  time.Now(),
	}
}

// AddChange adds a field change to the audit log
func (a *AuditLog) AddChange(field string, oldValue, newValue interface{}) {
	a.Changes[field] = ChangeEntry{
		OldValue: oldValue,
		NewValue: newValue,
	}
}

// SetMetadata sets request metadata
func (a *AuditLog) SetMetadata(ipAddress, userAgent string) {
	a.IPAddress = ipAddress
	a.UserAgent = userAgent
}
