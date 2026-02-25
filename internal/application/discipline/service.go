package discipline

import (
	"context"

	"github.com/bengobox/game-stats-api/ent"
	disciplineDomain "github.com/bengobox/game-stats-api/internal/domain/discipline"
	"github.com/google/uuid"
)

// Service provides business logic around disciplines. It is a thin wrapper
// over the repository which is used directly by HTTP handlers.
// Keeping a service layer allows us to add validation or caching later without
// tying the handlers to the persistence details.

// DTOs exposed by the service (identical to ent-based fields for now).

type DisciplineDTO struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Slug        string  `json:"slug"`
	Description *string `json:"description,omitempty"`
	RulesPdfURL *string `json:"rulesPdfUrl,omitempty"`
	CountryID   string  `json:"countryId"`
}

// Service orchestrates discipline operations.

type Service struct {
	repo disciplineDomain.Repository
}

// NewService constructs a Discipline Service.
func NewService(repo disciplineDomain.Repository) *Service {
	return &Service{repo: repo}
}

// List retrieves all active disciplines.
func (s *Service) List(ctx context.Context) ([]*ent.Discipline, error) {
	return s.repo.List(ctx)
}

// GetByID retrieves a single discipline by uuid.
func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*ent.Discipline, error) {
	return s.repo.GetByID(ctx, id)
}

// Create adds a new discipline to the system.
func (s *Service) Create(ctx context.Context, d *ent.Discipline) (*ent.Discipline, error) {
	return s.repo.Create(ctx, d)
}

// Update modifies an existing discipline.
func (s *Service) Update(ctx context.Context, d *ent.Discipline) (*ent.Discipline, error) {
	return s.repo.Update(ctx, d)
}

// Delete soft-deletes a discipline.
func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}
