package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/bengobox/game-stats-api/internal/application/gamemanagement"
	"github.com/bengobox/game-stats-api/internal/application/sse"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type GameHandler struct {
	service   *gamemanagement.Service
	sseBroker *sse.Broker
}

func NewGameHandler(service *gamemanagement.Service, sseBroker *sse.Broker) *GameHandler {
	return &GameHandler{
		service:   service,
		sseBroker: sseBroker,
	}
}

// ScheduleGame creates a new game.
// @Summary Schedule Game
// @Description Create a new game schedule
// @Tags games
// @Accept json
// @Produce json
// @Param request body gamemanagement.CreateGameRequest true "Game Schedule Request"
// @Success 201 {object} gamemanagement.GameDTO
// @Failure 400 {string} string "bad request"
// @Failure 409 {string} string "field conflict"
// @Security BearerAuth
// @Router /divisions/{id}/games [post]
func (h *GameHandler) ScheduleGame(w http.ResponseWriter, r *http.Request) {
	var req gamemanagement.CreateGameRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	game, err := h.service.ScheduleGame(r.Context(), req)
	if err != nil {
		if err == gamemanagement.ErrFieldConflict {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(game)
}

// GetGame retrieves a game by ID.
// @Summary Get Game
// @Description Get game details by ID
// @Tags games
// @Produce json
// @Param id path string true "Game ID"
// @Success 200 {object} gamemanagement.GameDTO
// @Failure 404 {string} string "not found"
// @Router /games/{id} [get]
func (h *GameHandler) GetGame(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "invalid game ID", http.StatusBadRequest)
		return
	}

	game, err := h.service.GetGame(r.Context(), id)
	if err != nil {
		if err == gamemanagement.ErrGameNotFound {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(game)
}

// ListGames retrieves games with filters.
// @Summary List Games
// @Description List games with optional filters
// @Tags games
// @Produce json
// @Param division_pool_id query string false "Division Pool ID"
// @Param status query string false "Game Status"
// @Param field_id query string false "Field ID"
// @Param start_date query string false "Start Date (RFC3339)"
// @Param end_date query string false "End Date (RFC3339)"
// @Param limit query int false "Limit results" default(50)
// @Param offset query int false "Offset for pagination" default(0)
// @Success 200 {array} gamemanagement.GameDTO
// @Failure 400 {string} string "bad request"
// @Router /games [get]
func (h *GameHandler) ListGames(w http.ResponseWriter, r *http.Request) {
	pagination := ParsePagination(r)
	filter := gamemanagement.ListGamesFilter{
		Limit:  pagination.Limit,
		Offset: pagination.Offset,
	}

	if divisionStr := r.URL.Query().Get("division_pool_id"); divisionStr != "" {
		divisionID, err := uuid.Parse(divisionStr)
		if err != nil {
			http.Error(w, "invalid division_pool_id", http.StatusBadRequest)
			return
		}
		filter.DivisionPoolID = &divisionID
	}

	if status := r.URL.Query().Get("status"); status != "" {
		filter.Status = &status
	}

	if fieldStr := r.URL.Query().Get("field_id"); fieldStr != "" {
		fieldID, err := uuid.Parse(fieldStr)
		if err != nil {
			http.Error(w, "invalid field_id", http.StatusBadRequest)
			return
		}
		filter.FieldID = &fieldID
	}

	if startStr := r.URL.Query().Get("start_date"); startStr != "" {
		start, err := time.Parse(time.RFC3339, startStr)
		if err != nil {
			http.Error(w, "invalid start_date format (use RFC3339)", http.StatusBadRequest)
			return
		}
		filter.StartDate = &start
	}

	if endStr := r.URL.Query().Get("end_date"); endStr != "" {
		end, err := time.Parse(time.RFC3339, endStr)
		if err != nil {
			http.Error(w, "invalid end_date format (use RFC3339)", http.StatusBadRequest)
			return
		}
		filter.EndDate = &end
	}

	games, err := h.service.ListGames(r.Context(), filter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(games)
}

// UpdateGame updates a game.
// @Summary Update Game
// @Description Update game details
// @Tags games
// @Accept json
// @Produce json
// @Param id path string true "Game ID"
// @Param request body gamemanagement.UpdateGameRequest true "Update Request"
// @Success 200 {object} gamemanagement.GameDTO
// @Failure 400 {string} string "bad request"
// @Failure 404 {string} string "not found"
// @Security BearerAuth
// @Router /games/{id} [put]
func (h *GameHandler) UpdateGame(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "invalid game ID", http.StatusBadRequest)
		return
	}

	var req gamemanagement.UpdateGameRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	game, err := h.service.UpdateGame(r.Context(), id, req)
	if err != nil {
		if err == gamemanagement.ErrGameNotFound {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(game)
}

// CancelGame cancels a game.
// @Summary Cancel Game
// @Description Cancel a scheduled or in-progress game
// @Tags games
// @Param id path string true "Game ID"
// @Success 204
// @Failure 400 {string} string "bad request"
// @Failure 404 {string} string "not found"
// @Security BearerAuth
// @Router /games/{id} [delete]
func (h *GameHandler) CancelGame(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "invalid game ID", http.StatusBadRequest)
		return
	}

	if err := h.service.CancelGame(r.Context(), id); err != nil {
		if err == gamemanagement.ErrGameNotFound {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// StartGame starts a game timer.
// @Summary Start Game
// @Description Start game timer and set status to in_progress
// @Tags games
// @Accept json
// @Produce json
// @Param id path string true "Game ID"
// @Param request body gamemanagement.StartGameRequest true "Start Game Request"
// @Success 200 {object} gamemanagement.GameDTO
// @Failure 400 {string} string "bad request"
// @Failure 401 {string} string "unauthorized"
// @Failure 404 {string} string "not found"
// @Security BearerAuth
// @Router /games/{id}/start [post]
func (h *GameHandler) StartGame(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "invalid game ID", http.StatusBadRequest)
		return
	}

	userID := getUserIDFromContext(r)

	var req gamemanagement.StartGameRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	game, err := h.service.StartGame(r.Context(), id, userID, req)
	if err != nil {
		if err == gamemanagement.ErrGameNotFound {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		if err == gamemanagement.ErrUnauthorized {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Broadcast SSE event
	h.sseBroker.Broadcast(id, sse.EventGameStarted, game)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(game)
}

// FinishGame marks a game as finished (time expired).
// @Summary Finish Game
// @Description Mark game timer as finished (scores can still be edited)
// @Tags games
// @Produce json
// @Param id path string true "Game ID"
// @Success 200 {object} gamemanagement.GameDTO
// @Failure 400 {string} string "bad request"
// @Failure 401 {string} string "unauthorized"
// @Failure 404 {string} string "not found"
// @Security BearerAuth
// @Router /games/{id}/finish [post]
func (h *GameHandler) FinishGame(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "invalid game ID", http.StatusBadRequest)
		return
	}

	userID := getUserIDFromContext(r)

	game, err := h.service.FinishGame(r.Context(), id, userID)
	if err != nil {
		if err == gamemanagement.ErrGameNotFound {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		if err == gamemanagement.ErrUnauthorized {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Broadcast SSE event
	h.sseBroker.Broadcast(id, sse.EventGameFinished, game)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(game)
}

// EndGame finalizes a game (no more edits allowed).
// @Summary End Game
// @Description Finalize game submission by scorekeeper
// @Tags games
// @Produce json
// @Param id path string true "Game ID"
// @Success 200 {object} gamemanagement.GameDTO
// @Failure 400 {string} string "bad request"
// @Failure 401 {string} string "unauthorized"
// @Failure 404 {string} string "not found"
// @Security BearerAuth
// @Router /games/{id}/end [post]
func (h *GameHandler) EndGame(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "invalid game ID", http.StatusBadRequest)
		return
	}

	userID := getUserIDFromContext(r)

	game, err := h.service.EndGame(r.Context(), id, userID)
	if err != nil {
		if err == gamemanagement.ErrGameNotFound {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		if err == gamemanagement.ErrUnauthorized {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Broadcast SSE event
	h.sseBroker.Broadcast(id, sse.EventGameEnded, game)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(game)
}

// RecordStoppage records a game stoppage.
// @Summary Record Stoppage
// @Description Record game stoppage and extend game time
// @Tags games
// @Accept json
// @Produce json
// @Param id path string true "Game ID"
// @Param request body gamemanagement.RecordStoppageRequest true "Stoppage Request"
// @Success 200 {object} gamemanagement.GameDTO
// @Failure 400 {string} string "bad request"
// @Failure 401 {string} string "unauthorized"
// @Failure 404 {string} string "not found"
// @Security BearerAuth
// @Router /games/{id}/stoppages [post]
func (h *GameHandler) RecordStoppage(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "invalid game ID", http.StatusBadRequest)
		return
	}

	userID := getUserIDFromContext(r)

	var req gamemanagement.RecordStoppageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	game, err := h.service.RecordStoppage(r.Context(), id, userID, req)
	if err != nil {
		if err == gamemanagement.ErrGameNotFound {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		if err == gamemanagement.ErrUnauthorized {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Broadcast SSE event
	h.sseBroker.Broadcast(id, sse.EventStoppageRecorded, map[string]interface{}{
		"game":             game,
		"duration_seconds": req.DurationSeconds,
		"reason":           req.Reason,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(game)
}

// GetGameTimeline retrieves game event timeline.
// @Summary Get Game Timeline
// @Description Get complete game event timeline
// @Tags games
// @Produce json
// @Param id path string true "Game ID"
// @Success 200 {object} gamemanagement.GameTimelineDTO
// @Failure 404 {string} string "not found"
// @Router /games/{id}/timeline [get]
func (h *GameHandler) GetGameTimeline(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "invalid game ID", http.StatusBadRequest)
		return
	}

	timeline, err := h.service.GetGameTimeline(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(timeline)
}

// StreamGame streams real-time game updates via SSE.
// @Summary Stream Game Updates
// @Description Get real-time game updates via Server-Sent Events
// @Tags games
// @Produce text/event-stream
// @Param id path string true "Game ID"
// @Success 200 {string} string "event stream"
// @Router /games/{id}/stream [get]
func (h *GameHandler) StreamGame(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	gameID, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "invalid game ID", http.StatusBadRequest)
		return
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Subscribe to game events
	clientID, eventChan := h.sseBroker.Subscribe(gameID)
	defer h.sseBroker.Unsubscribe(gameID, clientID)

	// Create flusher
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	// Send heartbeat every 30 seconds
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return

		case event := <-eventChan:
			fmt.Fprint(w, sse.FormatSSE(event))
			flusher.Flush()

		case <-ticker.C:
			// Send heartbeat
			heartbeat := sse.Event{
				Type: sse.EventHeartbeat,
				Data: map[string]interface{}{
					"timestamp": time.Now(),
					"clients":   h.sseBroker.GetClientCount(gameID),
				},
			}
			fmt.Fprint(w, sse.FormatSSE(heartbeat))
			flusher.Flush()
		}
	}
}

// RecordScore records player score.
// @Summary Record Score
// @Description Record player goals, assists, blocks, etc.
// @Tags games
// @Accept json
// @Produce json
// @Param id path string true "Game ID"
// @Param request body gamemanagement.RecordScoreRequest true "Score Request"
// @Success 200 {object} gamemanagement.GameDTO
// @Failure 400 {string} string "bad request"
// @Failure 401 {string} string "unauthorized"
// @Failure 404 {string} string "not found"
// @Security BearerAuth
// @Router /games/{id}/scores [post]
func (h *GameHandler) RecordScore(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	gameID, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "invalid game ID", http.StatusBadRequest)
		return
	}

	userID := getUserIDFromContext(r)

	var req gamemanagement.RecordScoreRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	game, err := h.service.RecordScore(r.Context(), gameID, userID, req)
	if err != nil {
		if err == gamemanagement.ErrGameNotFound {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		if err == gamemanagement.ErrUnauthorized {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Broadcast SSE event
	h.sseBroker.Broadcast(gameID, sse.EventScoreUpdated, game)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(game)
}

// GetGameScores retrieves all scores for a game.
// @Summary Get Game Scores
// @Description Get all player scores for a game
// @Tags games
// @Produce json
// @Param id path string true "Game ID"
// @Success 200 {array} gamemanagement.ScoringDTO
// @Failure 404 {string} string "not found"
// @Router /games/{id}/scores [get]
func (h *GameHandler) GetGameScores(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	gameID, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "invalid game ID", http.StatusBadRequest)
		return
	}

	scores, err := h.service.GetGameScores(r.Context(), gameID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(scores)
}

// Helper function to get user ID from context (set by auth middleware)
func getUserIDFromContext(r *http.Request) uuid.UUID {
	userIDValue := r.Context().Value("user_id")
	if userIDValue == nil {
		return uuid.Nil
	}

	switch v := userIDValue.(type) {
	case uuid.UUID:
		return v
	case string:
		id, _ := uuid.Parse(v)
		return id
	default:
		return uuid.Nil
	}
}
