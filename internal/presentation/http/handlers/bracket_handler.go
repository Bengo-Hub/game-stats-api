package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/bengobox/game-stats-api/internal/application/bracket"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// BracketHandler handles bracket-related HTTP requests
type BracketHandler struct {
	service *bracket.Service
}

// NewBracketHandler creates a new bracket handler
func NewBracketHandler(service *bracket.Service) *BracketHandler {
	return &BracketHandler{
		service: service,
	}
}

// GenerateBracket godoc
// @Summary Generate tournament bracket
// @Description Generate a tournament bracket structure with games
// @Tags brackets
// @Accept json
// @Produce json
// @Param request body bracket.GenerateBracketRequest true "Bracket generation request"
// @Success 201 {object} bracket.GenerateBracketResponse
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/events/{id}/generate-bracket [post]
func (h *BracketHandler) GenerateBracket(w http.ResponseWriter, r *http.Request) {
	var req bracket.GenerateBracketRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Get event ID from URL
	eventIDStr := chi.URLParam(r, "id")
	eventID, err := uuid.Parse(eventIDStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid event ID")
		return
	}

	req.EventID = eventID

	response, err := h.service.GenerateBracket(r.Context(), req)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, response)
}

// GetEventBracket godoc
// @Summary Get event bracket
// @Description Get the bracket structure for an event's bracket round
// @Tags brackets
// @Produce json
// @Param id path string true "Event ID"
// @Success 200 {object} bracket.GetBracketResponse
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/events/{id}/bracket [get]
func (h *BracketHandler) GetEventBracket(w http.ResponseWriter, r *http.Request) {
	// Get round_id from query parameter
	roundIDStr := r.URL.Query().Get("round_id")
	if roundIDStr == "" {
		respondError(w, http.StatusBadRequest, "round_id query parameter required")
		return
	}

	roundID, err := uuid.Parse(roundIDStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid round ID")
		return
	}

	response, err := h.service.GetBracket(r.Context(), roundID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, response)
}

// GetRoundBracket godoc
// @Summary Get round bracket
// @Description Get the bracket structure for a specific round
// @Tags brackets
// @Produce json
// @Param id path string true "Round ID"
// @Success 200 {object} bracket.GetBracketResponse
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/rounds/{id}/bracket [get]
func (h *BracketHandler) GetRoundBracket(w http.ResponseWriter, r *http.Request) {
	roundIDStr := chi.URLParam(r, "id")
	roundID, err := uuid.Parse(roundIDStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid round ID")
		return
	}

	response, err := h.service.GetBracket(r.Context(), roundID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, response)
}
