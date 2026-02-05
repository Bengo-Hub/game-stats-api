package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/bengobox/game-stats-api/internal/application/ranking"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type RankingHandler struct {
	service *ranking.Service
}

func NewRankingHandler(service *ranking.Service) *RankingHandler {
	return &RankingHandler{
		service: service,
	}
}

// GetDivisionStandings godoc
// @Summary Get division standings
// @Description Get current team standings for a division with ranking
// @Tags rankings
// @Produce json
// @Param id path string true "Division ID" format(uuid)
// @Success 200 {object} ranking.DivisionStandingsResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /divisions/{id}/standings [get]
func (h *RankingHandler) GetDivisionStandings(w http.ResponseWriter, r *http.Request) {
	divisionIDStr := chi.URLParam(r, "id")
	divisionID, err := uuid.Parse(divisionIDStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid division ID")
		return
	}

	standings, err := h.service.CalculateStandings(r.Context(), divisionID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to calculate standings")
		return
	}

	respondJSON(w, http.StatusOK, standings)
}

// UpdateRankingCriteria godoc
// @Summary Update division ranking criteria
// @Description Update the ranking rules for a division
// @Tags rankings
// @Accept json
// @Produce json
// @Param id path string true "Division ID" format(uuid)
// @Param request body ranking.UpdateRankingCriteriaRequest true "Ranking criteria"
// @Success 200 {object} map[string]string
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /divisions/{id}/criteria [put]
func (h *RankingHandler) UpdateRankingCriteria(w http.ResponseWriter, r *http.Request) {
	divisionIDStr := chi.URLParam(r, "id")
	divisionID, err := uuid.Parse(divisionIDStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid division ID")
		return
	}

	var req ranking.UpdateRankingCriteriaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := h.service.UpdateRankingCriteria(r.Context(), divisionID, req); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to update ranking criteria")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "Ranking criteria updated successfully"})
}

// AdvanceTeams godoc
// @Summary Advance teams to next round
// @Description Advance top N teams from division to bracket round
// @Tags rankings
// @Accept json
// @Produce json
// @Param request body ranking.AdvanceTeamsRequest true "Advancement request"
// @Success 200 {object} ranking.AdvanceTeamsResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /divisions/advance [post]
func (h *RankingHandler) AdvanceTeams(w http.ResponseWriter, r *http.Request) {
	var req ranking.AdvanceTeamsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	response, err := h.service.AdvanceTeams(r.Context(), req)
	if err != nil {
		if err.Error() == "not enough teams" || err.Error() == "target round not found" {
			respondError(w, http.StatusBadRequest, err.Error())
		} else {
			respondError(w, http.StatusInternalServerError, "Failed to advance teams")
		}
		return
	}

	respondJSON(w, http.StatusOK, response)
}
