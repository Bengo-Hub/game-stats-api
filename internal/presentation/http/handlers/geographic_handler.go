package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/bengobox/game-stats-api/internal/application/metadata"
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
