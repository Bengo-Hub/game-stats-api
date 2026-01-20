package repository

import (
	"context"
	"time"

	"github.com/bengobox/game-stats-api/ent"
	"github.com/bengobox/game-stats-api/ent/analyticsembedding"
	"github.com/google/uuid"
)

type analyticsEmbeddingRepository struct {
	client *ent.Client
}

// NewAnalyticsEmbeddingRepository creates a new analytics embedding repository.
func NewAnalyticsEmbeddingRepository(client *ent.Client) *analyticsEmbeddingRepository {
	return &analyticsEmbeddingRepository{client: client}
}

func (r *analyticsEmbeddingRepository) Create(ctx context.Context, e *ent.AnalyticsEmbedding) (*ent.AnalyticsEmbedding, error) {
	return r.client.AnalyticsEmbedding.Create().
		SetEntityType(e.EntityType).
		SetEntityID(e.EntityID).
		SetEmbedding(e.Embedding).
		SetNillableContent(e.Content).
		SetMetadata(e.Metadata).
		Save(ctx)
}

func (r *analyticsEmbeddingRepository) GetByID(ctx context.Context, id uuid.UUID) (*ent.AnalyticsEmbedding, error) {
	return r.client.AnalyticsEmbedding.Get(ctx, id)
}

func (r *analyticsEmbeddingRepository) GetByEntity(ctx context.Context, entityType string, entityID uuid.UUID) (*ent.AnalyticsEmbedding, error) {
	return r.client.AnalyticsEmbedding.Query().
		Where(analyticsembedding.EntityTypeEQ(entityType)).
		Where(analyticsembedding.EntityID(entityID)).
		Only(ctx)
}

func (r *analyticsEmbeddingRepository) Update(ctx context.Context, e *ent.AnalyticsEmbedding) (*ent.AnalyticsEmbedding, error) {
	return r.client.AnalyticsEmbedding.UpdateOneID(e.ID).
		SetEmbedding(e.Embedding).
		SetNillableContent(e.Content).
		SetMetadata(e.Metadata).
		SetUpdatedAt(time.Now()).
		Save(ctx)
}

func (r *analyticsEmbeddingRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.client.AnalyticsEmbedding.UpdateOneID(id).
		SetDeletedAt(time.Now()).
		Exec(ctx)
}
