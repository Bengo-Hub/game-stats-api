package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// AuditLog holds the schema definition for the AuditLog entity.
type AuditLog struct {
	ent.Schema
}

// Fields of the AuditLog.
func (AuditLog) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New).
			Immutable(),
		field.String("entity_type").
			NotEmpty().
			Comment("Type of entity being audited (game, spirit_score, etc.)"),
		field.UUID("entity_id", uuid.UUID{}).
			Comment("ID of the entity being audited"),
		field.String("action").
			NotEmpty().
			Comment("Action performed: create, update, delete"),
		field.UUID("user_id", uuid.UUID{}).
			Comment("ID of user who performed the action"),
		field.String("username").
			NotEmpty().
			Comment("Username of user who performed the action"),
		field.JSON("changes", map[string]interface{}{}).
			Comment("Map of field changes with old and new values"),
		field.Text("reason").
			NotEmpty().
			Comment("Mandatory reason for the change (minimum 10 characters)"),
		field.String("ip_address").
			Optional().
			Comment("IP address of the request"),
		field.Text("user_agent").
			Optional().
			Comment("User agent of the client"),
		field.Time("created_at").
			Default(time.Now).
			Immutable().
			Comment("Timestamp when audit log was created"),
	}
}

// Indexes of the AuditLog.
func (AuditLog) Indexes() []ent.Index {
	return []ent.Index{
		// Index for querying audit logs by entity
		index.Fields("entity_type", "entity_id"),
		// Index for querying audit logs by user
		index.Fields("user_id"),
		// Index for recent audit logs (most common query)
		index.Fields("created_at"),
	}
}
