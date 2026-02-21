package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/bengobox/game-stats-api/internal/application/analytics"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// AnalyticsHandler handles analytics-related HTTP requests
type AnalyticsHandler struct {
	analyticsService *analytics.Service
	textToSQLService *analytics.TextToSQLService
}

// NewAnalyticsHandler creates a new analytics handler
func NewAnalyticsHandler(analyticsService *analytics.Service, textToSQLService *analytics.TextToSQLService) *AnalyticsHandler {
	return &AnalyticsHandler{
		analyticsService: analyticsService,
		textToSQLService: textToSQLService,
	}
}

// ListDashboards godoc
// @Summary List all Superset dashboards
// @Description Retrieves all available analytics dashboards
// @Tags analytics
// @Accept json
// @Produce json
// @Success 200 {object} analytics.ListDashboardsResponse
// @Failure 500 {object} ErrorResponse
// @Router /analytics/dashboards [get]
// @Security BearerAuth
func (h *AnalyticsHandler) ListDashboards(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	response, err := h.analyticsService.ListDashboards(ctx)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to list dashboards", err)
		return
	}

	respondWithJSON(w, http.StatusOK, response)
}

// GetDashboard godoc
// @Summary Get dashboard by UUID
// @Description Retrieves a specific dashboard's metadata
// @Tags analytics
// @Accept json
// @Produce json
// @Param dashboard_uuid path string true "Dashboard UUID" format(uuid)
// @Success 200 {object} analytics.Dashboard
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /analytics/dashboards/{dashboard_uuid} [get]
// @Security BearerAuth
func (h *AnalyticsHandler) GetDashboard(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse dashboard UUID
	dashboardUUIDStr := chi.URLParam(r, "dashboard_uuid")
	dashboardUUID, err := uuid.Parse(dashboardUUIDStr)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid dashboard UUID", err)
		return
	}

	dashboard, err := h.analyticsService.GetDashboard(ctx, dashboardUUID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to get dashboard", err)
		return
	}

	respondWithJSON(w, http.StatusOK, dashboard)
}

// GenerateEmbedToken godoc
// @Summary Generate embed token for dashboard
// @Description Creates a guest token for embedding a Superset dashboard with RLS
// @Tags analytics
// @Accept json
// @Produce json
// @Param dashboard_uuid path string true "Dashboard UUID" format(uuid)
// @Param body body GenerateEmbedTokenRequestDTO true "Embed token parameters"
// @Success 200 {object} analytics.GenerateEmbedTokenResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /analytics/embed-token/{dashboard_uuid} [post]
// @Security BearerAuth
func (h *AnalyticsHandler) GenerateEmbedToken(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse dashboard UUID from URL
	dashboardUUIDStr := chi.URLParam(r, "dashboard_uuid")
	dashboardUUID, err := uuid.Parse(dashboardUUIDStr)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid dashboard UUID", err)
		return
	}

	// Parse request body
	var dto GenerateEmbedTokenRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	// Build service request
	request := analytics.GenerateEmbedTokenRequest{
		DashboardUUID: dashboardUUID,
		UserID:        dto.UserID,
		EventID:       dto.EventID,
		TeamIDs:       dto.TeamIDs,
		Username:      dto.Username,
		FirstName:     dto.FirstName,
		LastName:      dto.LastName,
	}

	response, err := h.analyticsService.GenerateEmbedToken(ctx, request)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to generate embed token", err)
		return
	}

	respondWithJSON(w, http.StatusOK, response)
}

// GetEventStatistics godoc
// @Summary Get event statistics
// @Description Retrieves comprehensive analytics for an event
// @Tags analytics
// @Accept json
// @Produce json
// @Param event_id path string true "Event UUID" format(uuid)
// @Success 200 {object} analytics.EventStatistics
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /analytics/events/{event_id}/statistics [get]
// @Security BearerAuth
func (h *AnalyticsHandler) GetEventStatistics(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse event UUID
	eventUUIDStr := chi.URLParam(r, "event_id")
	eventUUID, err := uuid.Parse(eventUUIDStr)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid event UUID", err)
		return
	}

	stats, err := h.analyticsService.GetEventStatistics(ctx, eventUUID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to get event statistics", err)
		return
	}

	respondWithJSON(w, http.StatusOK, stats)
}

// HealthCheck godoc
// @Summary Analytics health check
// @Description Verifies Superset connectivity
// @Tags analytics
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 500 {object} ErrorResponse
// @Router /analytics/health [get]
func (h *AnalyticsHandler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if err := h.analyticsService.HealthCheck(ctx); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Superset health check failed", err)
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]string{
		"status":  "healthy",
		"service": "superset",
	})
}

// NaturalLanguageQuery godoc
// @Summary Process natural language query
// @Description Converts natural language to SQL using Ollama LLM and executes the query
// @Tags analytics
// @Accept json
// @Produce json
// @Param body body NaturalLanguageQueryRequestDTO true "Natural language question"
// @Success 200 {object} analytics.NaturalLanguageQueryResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /analytics/query [post]
// @Security BearerAuth
func (h *AnalyticsHandler) NaturalLanguageQuery(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse request body
	var dto NaturalLanguageQueryRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	// Build service request
	request := analytics.NaturalLanguageQueryRequest{
		Question: dto.Question,
		EventID:  dto.EventID,
		UserID:   dto.UserID,
		Context:  dto.Context,
	}

	response, err := h.textToSQLService.ProcessNaturalLanguageQuery(ctx, request)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to process query", err)
		return
	}

	respondWithJSON(w, http.StatusOK, response)
}

// NaturalLanguageQueryRequestDTO is the DTO for natural language queries
type NaturalLanguageQueryRequestDTO struct {
	Question string     `json:"question" validate:"required"`
	EventID  *uuid.UUID `json:"event_id,omitempty"`
	UserID   uuid.UUID  `json:"user_id" validate:"required"`
	Context  string     `json:"context,omitempty"`
}

// GenerateEmbedTokenRequestDTO is the DTO for embed token generation
type GenerateEmbedTokenRequestDTO struct {
	UserID    uuid.UUID   `json:"user_id" validate:"required"`
	EventID   *uuid.UUID  `json:"event_id,omitempty"`
	TeamIDs   []uuid.UUID `json:"team_ids,omitempty"`
	Username  string      `json:"username" validate:"required,email"`
	FirstName string      `json:"first_name" validate:"required"`
	LastName  string      `json:"last_name" validate:"required"`
}
