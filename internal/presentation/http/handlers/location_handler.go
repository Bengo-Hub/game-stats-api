package handlers

import (
	"net/http"
	"strings"

	"github.com/bengobox/game-stats-api/ent"
	"github.com/bengobox/game-stats-api/internal/application/metadata"
	"github.com/bengobox/game-stats-api/internal/pkg/logger"
	"github.com/google/uuid"
)

// ======== DTOs ========

type CreateLocationRequest struct {
	Name      string   `json:"name" validate:"required"`
	Slug      string   `json:"slug"`
	Address   *string  `json:"address,omitempty"`
	City      *string  `json:"city,omitempty"`
	Latitude  *float64 `json:"latitude,omitempty"`
	Longitude *float64 `json:"longitude,omitempty"`
	CountryID string   `json:"countryId" validate:"required"`
}

type UpdateLocationRequest struct {
	Name      *string  `json:"name,omitempty"`
	Slug      *string  `json:"slug,omitempty"`
	Address   *string  `json:"address,omitempty"`
	City      *string  `json:"city,omitempty"`
	Latitude  *float64 `json:"latitude,omitempty"`
	Longitude *float64 `json:"longitude,omitempty"`
	CountryID *string  `json:"countryId,omitempty"`
}

type LocationHandler struct {
	service   *metadata.Service
	entClient *ent.Client
}

func NewLocationHandler(service *metadata.Service, client *ent.Client) *LocationHandler {
	return &LocationHandler{
		service:   service,
		entClient: client,
	}
}

// ListLocations returns all locations.
// @Summary List locations
// @Tags geographic
// @Produce json
// @Success 200 {array} metadata.LocationDTO
// @Router /geographic/locations [get]
func (h *LocationHandler) ListLocations(w http.ResponseWriter, r *http.Request) {
	list, err := h.service.ListLocations(r.Context())
	if err != nil {
		logger.Error("Failed to list locations", logger.Err(err))
		respondError(w, http.StatusInternalServerError, "Failed to list locations")
		return
	}
	respondJSON(w, http.StatusOK, list)
}

// CreateLocation creates a new location.
// @Summary Create location
// @Tags geographic
// @Accept json
// @Produce json
// @Param location body CreateLocationRequest true "Location"
// @Success 201 {object} metadata.LocationDTO
// @Router /geographic/locations [post]
func (h *LocationHandler) CreateLocation(w http.ResponseWriter, r *http.Request) {
	var req CreateLocationRequest
	if err := parseJSONBody(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	countryID, err := uuid.Parse(req.CountryID)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid country ID")
		return
	}

	slug := req.Slug
	if slug == "" {
		slug = strings.ToLower(strings.ReplaceAll(req.Name, " ", "-"))
	}

	builder := h.entClient.Location.Create().
		SetName(req.Name).
		SetSlug(slug).
		SetCountryID(countryID)

	if req.Address != nil {
		builder.SetAddress(*req.Address)
	}
	if req.City != nil {
		builder.SetCity(*req.City)
	}
	if req.Latitude != nil {
		builder.SetLatitude(*req.Latitude)
	}
	if req.Longitude != nil {
		builder.SetLongitude(*req.Longitude)
	}

	loc, err := builder.Save(r.Context())
	if err != nil {
		logger.Error("Failed to create location", logger.Err(err))
		respondError(w, http.StatusInternalServerError, "Failed to create location: "+err.Error())
		return
	}

	// Transform to DTO
	resp := metadata.LocationDTO{
		ID:        loc.ID.String(),
		Name:      loc.Name,
		Slug:      loc.Slug,
		Address:   loc.Address,
		City:      loc.City,
		Latitude:  loc.Latitude,
		Longitude: loc.Longitude,
		CountryID: countryID.String(),
	}

	respondJSON(w, http.StatusCreated, resp)
}
