package metadata

import (
	"context"

	"github.com/bengobox/game-stats-api/internal/domain/continent"
	"github.com/bengobox/game-stats-api/internal/domain/country"
	"github.com/bengobox/game-stats-api/internal/domain/world"
	"github.com/google/uuid"
)

type Service struct {
	worldRepo     world.Repository
	continentRepo continent.Repository
	countryRepo   country.Repository
}

func NewService(w world.Repository, con continent.Repository, cou country.Repository) *Service {
	return &Service{
		worldRepo:     w,
		continentRepo: con,
		countryRepo:   cou,
	}
}

type WorldDTO struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

func (s *Service) ListWorlds(ctx context.Context) ([]WorldDTO, error) {
	worlds, err := s.worldRepo.List(ctx)
	if err != nil {
		return nil, err
	}

	dtos := make([]WorldDTO, len(worlds))
	for i, w := range worlds {
		dtos[i] = WorldDTO{
			ID:   w.ID.String(),
			Name: w.Name,
			Slug: w.Slug,
		}
	}
	return dtos, nil
}

type ContinentDTO struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Slug    string `json:"slug"`
	WorldID string `json:"world_id"`
}

func (s *Service) ListContinents(ctx context.Context) ([]ContinentDTO, error) {
	conts, err := s.continentRepo.List(ctx)
	if err != nil {
		return nil, err
	}

	dtos := make([]ContinentDTO, len(conts))
	for i, c := range conts {
		dtos[i] = ContinentDTO{
			ID:      c.ID.String(),
			Name:    c.Name,
			Slug:    c.Slug,
			WorldID: c.WorldID.String(),
		}
	}
	return dtos, nil
}

type CountryDTO struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Code        string `json:"code"`
	ContinentID string `json:"continent_id"`
}

func (s *Service) ListCountries(ctx context.Context, continentID *uuid.UUID) ([]CountryDTO, error) {
	countries, err := s.countryRepo.List(ctx)
	if err != nil {
		return nil, err
	}

	dtos := make([]CountryDTO, 0, len(countries))
	for _, c := range countries {
		// Filter by continent if provided
		if continentID != nil && c.ContinentID != *continentID {
			continue
		}
		dtos = append(dtos, CountryDTO{
			ID:          c.ID.String(),
			Name:        c.Name,
			Slug:        c.Slug,
			Code:        c.Code,
			ContinentID: c.ContinentID.String(),
		})
	}
	return dtos, nil
}
