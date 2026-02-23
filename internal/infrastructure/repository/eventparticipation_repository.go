package repository

import (
	"context"
	"time"

	"github.com/bengobox/game-stats-api/ent"
	"github.com/bengobox/game-stats-api/ent/event"
	"github.com/bengobox/game-stats-api/ent/eventparticipation"
	"github.com/bengobox/game-stats-api/ent/player"
	"github.com/google/uuid"
)

type eventParticipationRepository struct {
	client *ent.Client
}

// NewEventParticipationRepository creates a new event participation repository.
func NewEventParticipationRepository(client *ent.Client) *eventParticipationRepository {
	return &eventParticipationRepository{client: client}
}

func (r *eventParticipationRepository) Create(ctx context.Context, ep *ent.EventParticipation) (*ent.EventParticipation, error) {
	return r.client.EventParticipation.Create().
		SetRole(ep.Role).
		SetStatus(ep.Status).
		SetPlayerID(ep.Edges.Player.ID).
		SetTeamID(ep.Edges.Team.ID).
		SetEventID(ep.Edges.Event.ID).
		Save(ctx)
}

func (r *eventParticipationRepository) GetByID(ctx context.Context, id uuid.UUID) (*ent.EventParticipation, error) {
	return r.client.EventParticipation.Query().
		Where(eventparticipation.ID(id)).
		WithPlayer().
		WithTeam().
		WithEvent().
		Only(ctx)
}

func (r *eventParticipationRepository) ListByEvent(ctx context.Context, eventID uuid.UUID) ([]*ent.EventParticipation, error) {
	return r.client.EventParticipation.Query().
		Where(eventparticipation.HasEventWith(event.ID(eventID))).
		WithPlayer().
		WithTeam().
		All(ctx)
}

func (r *eventParticipationRepository) ListByPlayer(ctx context.Context, playerID uuid.UUID) ([]*ent.EventParticipation, error) {
	return r.client.EventParticipation.Query().
		Where(eventparticipation.HasPlayerWith(player.ID(playerID))).
		WithTeam().
		WithEvent().
		All(ctx)
}

func (r *eventParticipationRepository) Update(ctx context.Context, ep *ent.EventParticipation) (*ent.EventParticipation, error) {
	return r.client.EventParticipation.UpdateOneID(ep.ID).
		SetRole(ep.Role).
		SetStatus(ep.Status).
		SetUpdatedAt(time.Now()).
		Save(ctx)
}

func (r *eventParticipationRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.client.EventParticipation.DeleteOneID(id).Exec(ctx)
}
