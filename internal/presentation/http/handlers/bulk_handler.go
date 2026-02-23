package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/bengobox/game-stats-api/internal/application/gamemanagement"
)

type BulkHandler struct {
	service *gamemanagement.Service
}

func NewBulkHandler(service *gamemanagement.Service) *BulkHandler {
	return &BulkHandler{service: service}
}

// BulkTransferPlayers transfers multiple players between teams.
// @Summary Bulk Transfer Players
// @Description Transfer multiple players from different teams to new destinations
// @Tags bulk
// @Accept json
// @Produce json
// @Param request body gamemanagement.BulkTransferRequest true "Bulk Transfer Request"
// @Success 200 {string} string "success"
// @Failure 400 {string} string "bad request"
// @Security BearerAuth
// @Router /bulk/players/transfer [post]
func (h *BulkHandler) BulkTransferPlayers(w http.ResponseWriter, r *http.Request) {
	var req gamemanagement.BulkTransferRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.service.BulkTransferPlayers(r.Context(), req); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("success"))
}

// MassImportPlayers imports multiple players into a team.
// @Summary Mass Import Players
// @Description Import multiple players into a specific team for an event
// @Tags bulk
// @Accept json
// @Produce json
// @Param request body gamemanagement.MassImportPlayersRequest true "Mass Import Request"
// @Success 201 {array} uuid.UUID
// @Failure 400 {string} string "bad request"
// @Security BearerAuth
// @Router /bulk/players/import [post]
func (h *BulkHandler) MassImportPlayers(w http.ResponseWriter, r *http.Request) {
	var req gamemanagement.MassImportPlayersRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	playerIDs, err := h.service.MassImportPlayers(r.Context(), req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(playerIDs)
}
