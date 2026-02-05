package repository

import (
	"context"
	"time"

	"github.com/bengobox/game-stats-api/ent"
	"github.com/bengobox/game-stats-api/ent/game"
	"github.com/bengobox/game-stats-api/ent/gameevent"
	"github.com/google/uuid"
)

type gameEventRepository struct {
	client *ent.Client
}

// NewGameEventRepository creates a new game event repository.
func NewGameEventRepository(client *ent.Client) *gameEventRepository {
	return &gameEventRepository{client: client}
}

func (r *gameEventRepository) Create(ctx context.Context, event *ent.GameEvent) (*ent.GameEvent, error) {
	query := r.client.GameEvent.Create().
		SetEventType(event.EventType).
		SetMinute(event.Minute).
		SetSecond(event.Second).
		SetDescription(event.Description).
		SetGameID(event.Edges.Game.ID)

	if event.Metadata != nil {
		query.SetMetadata(event.Metadata)
	}

	return query.Save(ctx)
}

func (r *gameEventRepository) GetByID(ctx context.Context, id uuid.UUID) (*ent.GameEvent, error) {
	return r.client.GameEvent.Query().
		Where(gameevent.ID(id)).
		WithGame().
		Only(ctx)
}

func (r *gameEventRepository) ListByGame(ctx context.Context, gameID uuid.UUID) ([]*ent.GameEvent, error) {
	return r.client.GameEvent.Query().
		Where(gameevent.HasGameWith(game.ID(gameID))).
		Where(gameevent.DeletedAtIsNil()).
		Order(ent.Asc(gameevent.FieldMinute), ent.Asc(gameevent.FieldSecond)).
		All(ctx)
}

func (r *gameEventRepository) Update(ctx context.Context, event *ent.GameEvent) (*ent.GameEvent, error) {
	query := r.client.GameEvent.UpdateOneID(event.ID).
		SetEventType(event.EventType).
		SetMinute(event.Minute).
		SetSecond(event.Second).
		SetDescription(event.Description).
		SetUpdatedAt(time.Now())

	if event.Metadata != nil {
		query.SetMetadata(event.Metadata)
	}

	return query.Save(ctx)
}

func (r *gameEventRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.client.GameEvent.UpdateOneID(id).
		SetDeletedAt(time.Now()).
		Exec(ctx)
}
