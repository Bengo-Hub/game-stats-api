package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/bengobox/game-stats-api/internal/application/gamemanagement"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type GameRoundHandler struct {
	service *gamemanagement.Service
}

func NewGameRoundHandler(service *gamemanagement.Service) *GameRoundHandler {
	return &GameRoundHandler{
		service: service,
	}
}

// CreateGameRound creates a new game round.
// @Summary Create Game Round
// @Description Create a new round for an event (pool, bracket, etc.)
// @Tags game-rounds
// @Accept json
// @Produce json
// @Param event_id path string true "Event ID"
// @Param request body gamemanagement.CreateGameRoundRequest true "Game Round Request"
// @Success 201 {object} gamemanagement.GameRoundDTO
// @Failure 400 {string} string "bad request"
// @Security BearerAuth
// @Router /events/{event_id}/rounds [post]
func (h *GameRoundHandler) CreateGameRound(w http.ResponseWriter, r *http.Request) {
	var req gamemanagement.CreateGameRoundRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Set event ID from URL parameter
	eventIDStr := chi.URLParam(r, "event_id")
	eventID, err := uuid.Parse(eventIDStr)
	if err != nil {
		http.Error(w, "invalid event ID", http.StatusBadRequest)
		return
	}
	req.EventID = eventID

	round, err := h.service.CreateGameRound(r.Context(), req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(round)
}

// GetGameRound retrieves a game round by ID.
// @Summary Get Game Round
// @Description Get game round details by ID
// @Tags game-rounds
// @Produce json
// @Param id path string true "Round ID"
// @Success 200 {object} gamemanagement.GameRoundDTO
// @Failure 404 {string} string "not found"
// @Router /rounds/{id} [get]
func (h *GameRoundHandler) GetGameRound(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "invalid round ID", http.StatusBadRequest)
		return
	}

	round, err := h.service.GetGameRound(r.Context(), id)
	if err != nil {
		if err == gamemanagement.ErrGameRoundNotFound {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(round)
}

// ListGameRounds retrieves all rounds for an event.
// @Summary List Game Rounds
// @Description List all rounds for an event
// @Tags game-rounds
// @Produce json
// @Param event_id path string true "Event ID"
// @Success 200 {array} gamemanagement.GameRoundDTO
// @Failure 400 {string} string "bad request"
// @Router /events/{event_id}/rounds [get]
func (h *GameRoundHandler) ListGameRounds(w http.ResponseWriter, r *http.Request) {
	eventIDStr := chi.URLParam(r, "event_id")
	eventID, err := uuid.Parse(eventIDStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid event ID")
		return
	}

	pagination := ParsePagination(r)
	rounds, err := h.service.ListGameRounds(r.Context(), eventID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	total := len(rounds)
	start := pagination.Offset
	if start > total {
		start = total
	}
	end := start + pagination.Limit
	if end > total {
		end = total
	}
	paginatedRounds := rounds[start:end]

	respondJSON(w, http.StatusOK, NewPaginatedResponse(paginatedRounds, total, pagination.Limit, pagination.Offset))
}

func (h *GameRoundHandler) ListAllRounds(w http.ResponseWriter, r *http.Request) {
	pagination := ParsePagination(r)
	rounds, err := h.service.ListAllGameRounds(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	total := len(rounds)
	start := pagination.Offset
	if start > total {
		start = total
	}
	end := start + pagination.Limit
	if end > total {
		end = total
	}
	paginatedRounds := rounds[start:end]

	respondJSON(w, http.StatusOK, NewPaginatedResponse(paginatedRounds, total, pagination.Limit, pagination.Offset))
}

// @Summary Update Game Round
// @Description Update game round details
// @Tags game-rounds
// @Accept json
// @Produce json
// @Param id path string true "Round ID"
// @Param request body gamemanagement.UpdateGameRoundRequest true "Update Request"
// @Success 200 {object} gamemanagement.GameRoundDTO
// @Failure 400 {string} string "bad request"
// @Failure 404 {string} string "not found"
// @Security BearerAuth
// @Router /rounds/{id} [put]
func (h *GameRoundHandler) UpdateGameRound(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "invalid round ID", http.StatusBadRequest)
		return
	}

	var req gamemanagement.UpdateGameRoundRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	round, err := h.service.UpdateGameRound(r.Context(), id, req)
	if err != nil {
		if err == gamemanagement.ErrGameRoundNotFound {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(round)
}

// DeleteGameRound deletes a game round.
// @Summary Delete Game Round
// @Description Delete a game round by ID
// @Tags game-rounds
// @Param id path string true "Round ID"
// @Success 204 "no content"
// @Failure 404 {string} string "not found"
// @Security BearerAuth
// @Router /rounds/{id} [delete]
func (h *GameRoundHandler) DeleteGameRound(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "invalid round ID", http.StatusBadRequest)
		return
	}

	err = h.service.DeleteGameRound(r.Context(), id)
	if err != nil {
		if err == gamemanagement.ErrGameRoundNotFound {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// SeedDefaultRounds seeds default rounds for an event.
// @Summary Seed Default Rounds
// @Description Seed common game rounds (Pool, Bracket, etc.) for an event
// @Tags game-rounds
// @Param event_id path string true "Event ID"
// @Success 204 "no content"
// @Failure 400 {string} string "bad request"
// @Security BearerAuth
// @Router /events/{event_id}/rounds/seed [post]
func (h *GameRoundHandler) SeedDefaultRounds(w http.ResponseWriter, r *http.Request) {
	eventIDStr := chi.URLParam(r, "event_id")
	eventID, err := uuid.Parse(eventIDStr)
	if err != nil {
		http.Error(w, "invalid event ID", http.StatusBadRequest)
		return
	}

	err = h.service.SeedDefaultRounds(r.Context(), eventID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
