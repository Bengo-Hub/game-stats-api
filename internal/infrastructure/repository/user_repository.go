package repository

import (
	"context"
	"time"

	"github.com/bengobox/game-stats-api/ent"
	"github.com/bengobox/game-stats-api/ent/user"
	"github.com/google/uuid"
)

type userRepository struct {
	client *ent.Client
}

// NewUserRepository creates a new user repository.
func NewUserRepository(client *ent.Client) *userRepository {
	return &userRepository{client: client}
}

func (r *userRepository) Create(ctx context.Context, u *ent.User) (*ent.User, error) {
	return r.client.User.Create().
		SetEmail(u.Email).
		SetPasswordHash(u.PasswordHash).
		SetName(u.Name).
		SetRole(u.Role).
		SetNillableAvatarURL(u.AvatarURL).
		Save(ctx)
}

func (r *userRepository) GetByID(ctx context.Context, id uuid.UUID) (*ent.User, error) {
	return r.client.User.Get(ctx, id)
}

func (r *userRepository) GetByEmail(ctx context.Context, email string) (*ent.User, error) {
	return r.client.User.Query().
		Where(user.EmailEQ(email)).
		Only(ctx)
}

func (r *userRepository) List(ctx context.Context) ([]*ent.User, error) {
	return r.client.User.Query().
		Where(user.DeletedAtIsNil()).
		All(ctx)
}

func (r *userRepository) Update(ctx context.Context, u *ent.User) (*ent.User, error) {
	return r.client.User.UpdateOneID(u.ID).
		SetEmail(u.Email).
		SetPasswordHash(u.PasswordHash).
		SetName(u.Name).
		SetRole(u.Role).
		SetNillableAvatarURL(u.AvatarURL).
		SetUpdatedAt(time.Now()).
		Save(ctx)
}

func (r *userRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.client.User.UpdateOneID(id).
		SetDeletedAt(time.Now()).
		Exec(ctx)
}
