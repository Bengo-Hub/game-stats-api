package metadata

import (
	"context"

	"github.com/bengobox/game-stats-api/ent"
	"github.com/bengobox/game-stats-api/internal/domain/continent"
	"github.com/bengobox/game-stats-api/internal/domain/country"
	"github.com/bengobox/game-stats-api/internal/domain/field"
	"github.com/bengobox/game-stats-api/internal/domain/location"
	"github.com/bengobox/game-stats-api/internal/domain/world"
	"github.com/google/uuid"
)

type Service struct {
	worldRepo     world.Repository
	continentRepo continent.Repository
	countryRepo   country.Repository
	locationRepo  location.Repository
	fieldRepo     field.Repository
}

func NewService(w world.Repository, con continent.Repository, cou country.Repository, l location.Repository, f field.Repository) *Service {
	return &Service{
		worldRepo:     w,
		continentRepo: con,
		countryRepo:   cou,
		locationRepo:  l,
		fieldRepo:     f,
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

type LocationDTO struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Slug      string   `json:"slug"`
	Address   *string  `json:"address,omitempty"`
	City      *string  `json:"city,omitempty"`
	Latitude  *float64 `json:"latitude,omitempty"`
	Longitude *float64 `json:"longitude,omitempty"`
	CountryID string   `json:"country_id"`
}

func (s *Service) ListLocations(ctx context.Context) ([]LocationDTO, error) {
	locs, err := s.locationRepo.List(ctx)
	if err != nil {
		return nil, err
	}

	dtos := make([]LocationDTO, len(locs))
	for i, l := range locs {
		countryID := ""
		if l.Edges.Country != nil {
			countryID = l.Edges.Country.ID.String()
		}

		dtos[i] = LocationDTO{
			ID:        l.ID.String(),
			Name:      l.Name,
			Slug:      l.Slug,
			Address:   l.Address,
			City:      l.City,
			Latitude:  l.Latitude,
			Longitude: l.Longitude,
			CountryID: countryID,
		}
	}
	return dtos, nil
}

type FieldDTO struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Capacity    *int                   `json:"capacity,omitempty"`
	SurfaceType *string                `json:"surface_type,omitempty"`
	LocationID  string                 `json:"location_id"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

func (s *Service) ListFields(ctx context.Context, locationID *uuid.UUID) ([]FieldDTO, error) {
	fields, err := s.fieldRepo.List(ctx)
	if err != nil {
		return nil, err
	}

	dtos := make([]FieldDTO, 0, len(fields))
	for _, f := range fields {
		if locationID != nil && f.Edges.Location != nil && f.Edges.Location.ID != *locationID {
			continue
		}
		// If f.Edges.Location is nil but filter is provided, skip
		if locationID != nil && f.Edges.Location == nil {
			continue
		}

		locID := ""
		if f.Edges.Location != nil {
			locID = f.Edges.Location.ID.String()
		}

		dtos = append(dtos, FieldDTO{
			ID:          f.ID.String(),
			Name:        f.Name,
			Capacity:    f.Capacity,
			SurfaceType: f.SurfaceType,
			LocationID:  locID,
			Metadata:    f.Metadata,
		})
	}
	return dtos, nil
}

func (s *Service) CreateField(ctx context.Context, f *ent.Field) (*ent.Field, error) {
	return s.fieldRepo.Create(ctx, f)
}

func (s *Service) GetFieldByID(ctx context.Context, id uuid.UUID) (*ent.Field, error) {
	return s.fieldRepo.GetByID(ctx, id)
}

func (s *Service) CreateWorld(ctx context.Context, w *ent.World) (*ent.World, error) {
	return s.worldRepo.Create(ctx, w)
}

func (s *Service) CreateContinent(ctx context.Context, c *ent.Continent) (*ent.Continent, error) {
	return s.continentRepo.Create(ctx, c)
}

func (s *Service) CreateCountry(ctx context.Context, c *ent.Country) (*ent.Country, error) {
	return s.countryRepo.Create(ctx, c)
}

func (s *Service) CreateLocation(ctx context.Context, loc *ent.Location) (*ent.Location, error) {
	return s.locationRepo.Create(ctx, loc)
}

func (s *Service) GetLocationByID(ctx context.Context, id uuid.UUID) (*ent.Location, error) {
	return s.locationRepo.GetByID(ctx, id)
}
