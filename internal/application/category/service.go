package category

import (
	"context"

	"github.com/bengobox/game-stats-api/ent"
	categoryDomain "github.com/bengobox/game-stats-api/internal/domain/category"
	"github.com/google/uuid"
)

// Service contains business logic around categories.  Mirrors discipline service
// structure for consistency.

type CategoryDTO struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Slug        string  `json:"slug"`
	Description *string `json:"description,omitempty"`
}

type Service struct {
	repo categoryDomain.Repository
}

func NewService(repo categoryDomain.Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) List(ctx context.Context) ([]*ent.Category, error) {
	return s.repo.List(ctx)
}

func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*ent.Category, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *Service) Create(ctx context.Context, c *ent.Category) (*ent.Category, error) {
	return s.repo.Create(ctx, c)
}

func (s *Service) Update(ctx context.Context, c *ent.Category) (*ent.Category, error) {
	return s.repo.Update(ctx, c)
}

func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}
