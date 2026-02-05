package handlers

import (
	"encoding/json"
	"net/http"

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
