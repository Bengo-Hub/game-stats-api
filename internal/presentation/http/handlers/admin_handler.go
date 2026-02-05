package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/bengobox/game-stats-api/internal/application/admin"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// AdminHandler handles administrative operations
type AdminHandler struct {
	scoreAdminService *admin.ScoreAdminService
}

// NewAdminHandler creates a new admin handler
func NewAdminHandler(scoreAdminService *admin.ScoreAdminService) *AdminHandler {
	return &AdminHandler{
		scoreAdminService: scoreAdminService,
	}
}

// UpdateGameScore godoc
// @Summary Update game score (Admin only)
// @Description Updates a game's score with audit trail
// @Tags admin
// @Accept json
// @Produce json
// @Param body body UpdateGameScoreRequestDTO true "Score update request"
// @Success 200 {object} admin.UpdateGameScoreResponse
// @Failure 400 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /admin/games/{id}/score [put]
// @Security BearerAuth
func (h *AdminHandler) UpdateGameScore(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse game ID from URL
	gameIDStr := chi.URLParam(r, "id")
	gameID, err := uuid.Parse(gameIDStr)
	if err != nil {
		http.Error(w, "Invalid game ID", http.StatusBadRequest)
		return
	}

	// Parse request body
	var dto UpdateGameScoreRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Get user info from context (set by auth middleware)
	userID, _ := ctx.Value("user_id").(uuid.UUID)
	username, _ := ctx.Value("username").(string)

	// Build service request
	request := admin.UpdateGameScoreRequest{
		GameID:      gameID,
		HomeScore:   dto.HomeScore,
		AwayScore:   dto.AwayScore,
		Reason:      dto.Reason,
		AdminUserID: userID,
		AdminName:   username,
		IPAddress:   r.RemoteAddr,
		UserAgent:   r.UserAgent(),
	}

	response, err := h.scoreAdminService.UpdateGameScore(ctx, request)
	if err != nil {
		http.Error(w, "Failed to update game score: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// GetGameAuditHistory godoc
// @Summary Get game audit history (Admin only)
// @Description Retrieves all administrative changes made to a game
// @Tags admin
// @Produce json
// @Param id path string true "Game ID" format(uuid)
// @Success 200 {array} AuditLogDTO
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /admin/games/{id}/audit [get]
// @Security BearerAuth
func (h *AdminHandler) GetGameAuditHistory(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse game ID
	gameIDStr := chi.URLParam(r, "id")
	gameID, err := uuid.Parse(gameIDStr)
	if err != nil {
		http.Error(w, "Invalid game ID", http.StatusBadRequest)
		return
	}

	logs, err := h.scoreAdminService.GetAuditHistory(ctx, gameID)
	if err != nil {
		http.Error(w, "Failed to get audit history: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(logs)
}

// UpdateSpiritScore godoc
// @Summary Update spirit score (Admin only)
// @Description Updates a spirit score with audit trail
// @Tags admin
// @Accept json
// @Produce json
// @Param id path string true "Spirit Score ID" format(uuid)
// @Param body body UpdateSpiritScoreRequestDTO true "Spirit score update request"
// @Success 200 {object} admin.UpdateSpiritScoreResponse
// @Failure 400 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /admin/spirit-scores/{id} [put]
// @Security BearerAuth
func (h *AdminHandler) UpdateSpiritScore(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse spirit score ID
	spiritIDStr := chi.URLParam(r, "id")
	spiritID, err := uuid.Parse(spiritIDStr)
	if err != nil {
		http.Error(w, "Invalid spirit score ID", http.StatusBadRequest)
		return
	}

	// Parse request body
	var dto UpdateSpiritScoreRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Get user info from context
	userID, _ := ctx.Value("user_id").(uuid.UUID)
	username, _ := ctx.Value("username").(string)

	// Build service request
	request := admin.UpdateSpiritScoreRequest{
		SpiritScoreID:  spiritID,
		RulesKnowledge: dto.RulesKnowledge,
		Fouls:          dto.Fouls,
		FairMindedness: dto.FairMindedness,
		Attitude:       dto.Attitude,
		Communication:  dto.Communication,
		Reason:         dto.Reason,
		AdminUserID:    userID,
		AdminName:      username,
		IPAddress:      r.RemoteAddr,
		UserAgent:      r.UserAgent(),
	}

	response, err := h.scoreAdminService.UpdateSpiritScore(ctx, request)
	if err != nil {
		http.Error(w, "Failed to update spirit score: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// UpdateGameScoreRequestDTO is the DTO for game score updates
type UpdateGameScoreRequestDTO struct {
	HomeScore int    `json:"home_score" validate:"min=0"`
	AwayScore int    `json:"away_score" validate:"min=0"`
	Reason    string `json:"reason" validate:"required,min=10"`
}

// UpdateSpiritScoreRequestDTO is the DTO for spirit score updates
type UpdateSpiritScoreRequestDTO struct {
	RulesKnowledge int    `json:"rules_knowledge" validate:"min=0,max=4"`
	Fouls          int    `json:"fouls" validate:"min=0,max=4"`
	FairMindedness int    `json:"fair_mindedness" validate:"min=0,max=4"`
	Attitude       int    `json:"attitude" validate:"min=0,max=4"`
	Communication  int    `json:"communication" validate:"min=0,max=4"`
	Reason         string `json:"reason" validate:"required,min=10"`
}

// AuditLogDTO represents an audit log entry for swagger documentation
type AuditLogDTO struct {
	ID         string                 `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	EntityType string                 `json:"entity_type" example:"game"`
	EntityID   string                 `json:"entity_id" example:"550e8400-e29b-41d4-a716-446655440001"`
	Action     string                 `json:"action" example:"update"`
	UserID     string                 `json:"user_id" example:"550e8400-e29b-41d4-a716-446655440002"`
	Username   string                 `json:"username" example:"admin@example.com"`
	Changes    map[string]interface{} `json:"changes"`
	Reason     string                 `json:"reason,omitempty" example:"Score correction"`
	IPAddress  string                 `json:"ip_address,omitempty" example:"192.168.1.1"`
	UserAgent  string                 `json:"user_agent,omitempty" example:"Mozilla/5.0"`
	CreatedAt  string                 `json:"created_at" example:"2026-02-04T12:00:00Z"`
}
