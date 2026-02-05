package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/bengobox/game-stats-api/internal/application/gamemanagement"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type SpiritScoreHandler struct {
	service *gamemanagement.Service
}

func NewSpiritScoreHandler(service *gamemanagement.Service) *SpiritScoreHandler {
	return &SpiritScoreHandler{
		service: service,
	}
}

// SubmitSpiritScore godoc
// @Summary Submit spirit score for a game
// @Description Submit spirit score for a game by a team captain/manager
// @Tags Spirit Scores
// @Accept json
// @Produce json
// @Param id path string true "Game ID" format(uuid)
// @Param request body gamemanagement.SubmitSpiritScoreRequest true "Spirit score submission"
// @Success 201 {object} gamemanagement.SpiritScoreDTO
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 409 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /games/{id}/spirit [post]
func (h *SpiritScoreHandler) SubmitSpiritScore(w http.ResponseWriter, r *http.Request) {
	gameIDStr := chi.URLParam(r, "id")
	gameID, err := uuid.Parse(gameIDStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid game ID")
		return
	}

	var req gamemanagement.SubmitSpiritScoreRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Get user ID from context (set by auth middleware)
	userID, ok := r.Context().Value("userID").(uuid.UUID)
	if !ok {
		respondError(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	spiritScore, err := h.service.SubmitSpiritScore(r.Context(), gameID, userID, req)
	if err != nil {
		switch err {
		case gamemanagement.ErrGameNotFound:
			respondError(w, http.StatusNotFound, err.Error())
		case gamemanagement.ErrInvalidGameStatus:
			respondError(w, http.StatusBadRequest, err.Error())
		case gamemanagement.ErrTeamNotInGame:
			respondError(w, http.StatusBadRequest, err.Error())
		case gamemanagement.ErrUnauthorized:
			respondError(w, http.StatusForbidden, err.Error())
		default:
			if err.Error() == "spirit scores must be between 0 and 4" ||
				err.Error() == "cannot score your own team" ||
				err.Error() == "spirit score already submitted for this game by this team" {
				respondError(w, http.StatusBadRequest, err.Error())
			} else {
				respondError(w, http.StatusInternalServerError, "Failed to submit spirit score")
			}
		}
		return
	}

	respondJSON(w, http.StatusCreated, spiritScore)
}

// GetGameSpiritScores godoc
// @Summary Get spirit scores for a game
// @Description Get all spirit scores submitted for a specific game
// @Tags Spirit Scores
// @Produce json
// @Param id path string true "Game ID" format(uuid)
// @Success 200 {array} gamemanagement.SpiritScoreDTO
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /games/{id}/spirit [get]
func (h *SpiritScoreHandler) GetGameSpiritScores(w http.ResponseWriter, r *http.Request) {
	gameIDStr := chi.URLParam(r, "id")
	gameID, err := uuid.Parse(gameIDStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid game ID")
		return
	}

	spiritScores, err := h.service.GetGameSpiritScores(r.Context(), gameID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to retrieve spirit scores")
		return
	}

	respondJSON(w, http.StatusOK, spiritScores)
}

// GetTeamSpiritAverage godoc
// @Summary Get team spirit average
// @Description Get average spirit scores received by a team across all games
// @Tags Spirit Scores
// @Produce json
// @Param id path string true "Team ID" format(uuid)
// @Success 200 {object} gamemanagement.TeamSpiritAverageDTO
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /teams/{id}/spirit-average [get]
func (h *SpiritScoreHandler) GetTeamSpiritAverage(w http.ResponseWriter, r *http.Request) {
	teamIDStr := chi.URLParam(r, "id")
	teamID, err := uuid.Parse(teamIDStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid team ID")
		return
	}

	average, err := h.service.GetTeamSpiritAverage(r.Context(), teamID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to retrieve team spirit average")
		return
	}

	respondJSON(w, http.StatusOK, average)
}
