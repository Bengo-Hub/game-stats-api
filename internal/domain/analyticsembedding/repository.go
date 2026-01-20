package analyticsembedding

import (
	"context"

	"github.com/bengobox/game-stats-api/ent"
	"github.com/google/uuid"
)

type Repository interface {
	Create(ctx context.Context, e *ent.AnalyticsEmbedding) (*ent.AnalyticsEmbedding, error)
	GetByID(ctx context.Context, id uuid.UUID) (*ent.AnalyticsEmbedding, error)
	GetByEntity(ctx context.Context, entityType string, entityID uuid.UUID) (*ent.AnalyticsEmbedding, error)
	Update(ctx context.Context, e *ent.AnalyticsEmbedding) (*ent.AnalyticsEmbedding, error)
	Delete(ctx context.Context, id uuid.UUID) error
}
