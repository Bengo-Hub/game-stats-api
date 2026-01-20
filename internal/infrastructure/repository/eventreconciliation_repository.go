package repository

import (
	"context"
	"time"

	"github.com/bengobox/game-stats-api/ent"
	"github.com/bengobox/game-stats-api/ent/event"
	"github.com/bengobox/game-stats-api/ent/eventreconciliation"
	"github.com/google/uuid"
)

type eventReconciliationRepository struct {
	client *ent.Client
}

// NewEventReconciliationRepository creates a new event reconciliation repository.
func NewEventReconciliationRepository(client *ent.Client) *eventReconciliationRepository {
	return &eventReconciliationRepository{client: client}
}

func (r *eventReconciliationRepository) Create(ctx context.Context, rec *ent.EventReconciliation) (*ent.EventReconciliation, error) {
	return r.client.EventReconciliation.Create().
		SetReconciledAt(rec.ReconciledAt).
		SetReconciledBy(rec.ReconciledBy).
		SetStatus(rec.Status).
		SetNillableComments(rec.Comments).
		SetEventID(rec.Edges.Event.ID).
		Save(ctx)
}

func (r *eventReconciliationRepository) GetByID(ctx context.Context, id uuid.UUID) (*ent.EventReconciliation, error) {
	return r.client.EventReconciliation.Query().
		Where(eventreconciliation.ID(id)).
		WithEvent().
		Only(ctx)
}

func (r *eventReconciliationRepository) ListByEvent(ctx context.Context, eventID uuid.UUID) ([]*ent.EventReconciliation, error) {
	return r.client.EventReconciliation.Query().
		Where(eventreconciliation.HasEventWith(event.ID(eventID))).
		Where(eventreconciliation.DeletedAtIsNil()).
		All(ctx)
}

func (r *eventReconciliationRepository) Update(ctx context.Context, rec *ent.EventReconciliation) (*ent.EventReconciliation, error) {
	return r.client.EventReconciliation.UpdateOneID(rec.ID).
		SetReconciledAt(rec.ReconciledAt).
		SetReconciledBy(rec.ReconciledBy).
		SetStatus(rec.Status).
		SetNillableComments(rec.Comments).
		SetEventID(rec.Edges.Event.ID).
		SetUpdatedAt(time.Now()).
		Save(ctx)
}

func (r *eventReconciliationRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.client.EventReconciliation.UpdateOneID(id).
		SetDeletedAt(time.Now()).
		Exec(ctx)
}
