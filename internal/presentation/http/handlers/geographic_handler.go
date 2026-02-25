package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/bengobox/game-stats-api/ent"
	"github.com/bengobox/game-stats-api/internal/application/metadata"
	"github.com/google/uuid"
)

type GeographicHandler struct {
	service *metadata.Service
}

func NewGeographicHandler(service *metadata.Service) *GeographicHandler {
	return &GeographicHandler{service: service}
}

// ListWorlds handles the request to list all worlds.
// @Summary List Worlds
// @Description Get a list of all geographic worlds.
// @Tags geographic
// @Accept json
// @Produce json
// @Success 200 {array} metadata.WorldDTO
// @Router /geographic/worlds [get]
func (h *GeographicHandler) ListWorlds(w http.ResponseWriter, r *http.Request) {
	worlds, err := h.service.ListWorlds(r.Context())
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(worlds)
}

// ListContinents handles the request to list all continents.
// @Summary List Continents
// @Description Get a list of all geographic continents.
// @Tags geographic
// @Accept json
// @Produce json
// @Success 200 {array} metadata.ContinentDTO
// @Router /geographic/continents [get]
func (h *GeographicHandler) ListContinents(w http.ResponseWriter, r *http.Request) {
	conts, err := h.service.ListContinents(r.Context())
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(conts)
}

// ListCountries handles the request to list all countries.
// @Summary List Countries
// @Description Get a list of all countries, optionally filtered by continent.
// @Tags geographic
// @Accept json
// @Produce json
// @Param continent_id query string false "Filter by continent ID"
// @Success 200 {array} metadata.CountryDTO
// @Router /geographic/countries [get]
func (h *GeographicHandler) ListCountries(w http.ResponseWriter, r *http.Request) {
	var continentID *uuid.UUID
	if cid := r.URL.Query().Get("continent_id"); cid != "" {
		if id, err := uuid.Parse(cid); err == nil {
			continentID = &id
		}
	}

	countries, err := h.service.ListCountries(r.Context(), continentID)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(countries)
}

// ListFields handles the request to list all fields.
// @Summary List Fields
// @Description Get a list of all fields, optionally filtered by location.
// @Tags geographic
// @Accept json
// @Produce json
// @Param location_id query string false "Filter by location ID"
// @Success 200 {array} metadata.FieldDTO
// @Router /geographic/fields [get]
func (h *GeographicHandler) ListFields(w http.ResponseWriter, r *http.Request) {
	var locationID *uuid.UUID
	if lid := r.URL.Query().Get("location_id"); lid != "" {
		if id, err := uuid.Parse(lid); err == nil {
			locationID = &id
		}
	}

	fields, err := h.service.ListFields(r.Context(), locationID)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(fields)
}

// CreateFieldRequest represents the body for creating a field.
type CreateFieldRequest struct {
	Name        string                 `json:"name" validate:"required"`
	LocationID  string                 `json:"location_id" validate:"required"`
	Capacity    *int                   `json:"capacity,omitempty"`
	SurfaceType *string                `json:"surface_type,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// CreateField handles the request to create a new field.
// @Summary Create Field
// @Tags geographic
// @Accept json
// @Produce json
// @Param field body CreateFieldRequest true "Field"
// @Success 201 {object} metadata.FieldDTO
// @Router /geographic/fields [post]
func (h *GeographicHandler) CreateField(w http.ResponseWriter, r *http.Request) {
	var req CreateFieldRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	locationID, err := uuid.Parse(req.LocationID)
	if err != nil {
		http.Error(w, "invalid location ID", http.StatusBadRequest)
		return
	}

	res, err := h.service.CreateField(r.Context(), &ent.Field{
		Name:        req.Name,
		Capacity:    req.Capacity,
		SurfaceType: req.SurfaceType,
		Metadata:    req.Metadata,
		Edges: ent.FieldEdges{
			Location: &ent.Location{ID: locationID},
		},
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(metadata.FieldDTO{
		ID:          res.ID.String(),
		Name:        res.Name,
		Capacity:    res.Capacity,
		SurfaceType: res.SurfaceType,
		LocationID:  req.LocationID,
		Metadata:    res.Metadata,
	})
}

// CreateWorld handles the request to create a new world.
// @Summary Create World
// @Tags geographic
// @Accept json
// @Produce json
// @Param world body metadata.WorldDTO true "World"
// @Success 201 {object} metadata.WorldDTO
// @Router /geographic/worlds [post]
func (h *GeographicHandler) CreateWorld(w http.ResponseWriter, r *http.Request) {
	var dto metadata.WorldDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	res, err := h.service.CreateWorld(r.Context(), &ent.World{
		Name: dto.Name,
		Slug: dto.Slug,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(metadata.WorldDTO{
		ID:   res.ID.String(),
		Name: res.Name,
		Slug: res.Slug,
	})
}

// CreateContinent handles the request to create a new continent.
// @Summary Create Continent
// @Tags geographic
// @Accept json
// @Produce json
// @Param continent body metadata.ContinentDTO true "Continent"
// @Success 201 {object} metadata.ContinentDTO
// @Router /geographic/continents [post]
func (h *GeographicHandler) CreateContinent(w http.ResponseWriter, r *http.Request) {
	var dto metadata.ContinentDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	worldID, err := uuid.Parse(dto.WorldID)
	if err != nil {
		http.Error(w, "invalid world ID", http.StatusBadRequest)
		return
	}

	res, err := h.service.CreateContinent(r.Context(), &ent.Continent{
		Name:    dto.Name,
		Slug:    dto.Slug,
		WorldID: worldID,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(metadata.ContinentDTO{
		ID:      res.ID.String(),
		Name:    res.Name,
		Slug:    res.Slug,
		WorldID: res.WorldID.String(),
	})
}

// CreateCountry handles the request to create a new country.
// @Summary Create Country
// @Tags geographic
// @Accept json
// @Produce json
// @Param country body metadata.CountryDTO true "Country"
// @Success 201 {object} metadata.CountryDTO
// @Router /geographic/countries [post]
func (h *GeographicHandler) CreateCountry(w http.ResponseWriter, r *http.Request) {
	var dto metadata.CountryDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	continentID, err := uuid.Parse(dto.ContinentID)
	if err != nil {
		http.Error(w, "invalid continent ID", http.StatusBadRequest)
		return
	}

	res, err := h.service.CreateCountry(r.Context(), &ent.Country{
		Name:        dto.Name,
		Slug:        dto.Slug,
		Code:        dto.Code,
		ContinentID: continentID,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(metadata.CountryDTO{
		ID:          res.ID.String(),
		Name:        res.Name,
		Slug:        res.Slug,
		Code:        res.Code,
		ContinentID: res.ContinentID.String(),
	})
}
