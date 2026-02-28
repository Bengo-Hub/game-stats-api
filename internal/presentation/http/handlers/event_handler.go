package handlers

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/bengobox/game-stats-api/ent"
	"github.com/bengobox/game-stats-api/ent/category"
	"github.com/bengobox/game-stats-api/ent/country"
	"github.com/bengobox/game-stats-api/ent/discipline"
	"github.com/bengobox/game-stats-api/ent/divisionpool"
	"github.com/bengobox/game-stats-api/ent/event"
	entGame "github.com/bengobox/game-stats-api/ent/game"
	"github.com/bengobox/game-stats-api/ent/location"
	"github.com/bengobox/game-stats-api/ent/predicate"
	"github.com/bengobox/game-stats-api/ent/scopedrole"
	entUser "github.com/bengobox/game-stats-api/ent/user"
	"github.com/bengobox/game-stats-api/internal/pkg/logger"
	"github.com/bengobox/game-stats-api/internal/pkg/types"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// Note: Pagination uses ParsePagination from pagination.go

// ============================================
// Request DTOs
// ============================================

type CreateEventRequest struct {
	Name        string         `json:"name" validate:"required"`
	Slug        string         `json:"slug" validate:"required"`
	Year        int            `json:"year"`
	StartDate   types.JSONTime `json:"startDate" swaggertype:"string" example:"2026-02-25"`
	EndDate     types.JSONTime `json:"endDate" swaggertype:"string" example:"2026-02-26"`
	Status      string         `json:"status"`
	Description *string        `json:"description,omitempty"`
	// category IDs (uuid) that should be associated with the event
	CategoryIDs  []string                `json:"categoryIds,omitempty"`
	LogoUrl      *string                 `json:"logoUrl,omitempty"`
	BannerUrl    *string                 `json:"bannerUrl,omitempty"`
	LocationID   *string                 `json:"locationId" validate:"required"`
	DisciplineID *string                 `json:"disciplineId" validate:"required"`
	RulesUrl     *string                 `json:"rulesUrl,omitempty"`
	Metadata     map[string]interface{}  `json:"metadata,omitempty"`
	Divisions    []CreateDivisionRequest `json:"divisions,omitempty"`
	GameRoundIDs []string                `json:"gameRoundIds,omitempty"`
}

type CreateDivisionRequest struct {
	ID            *string    `json:"id,omitempty"`
	Name          string     `json:"name" validate:"required"`
	DivisionType  string     `json:"divisionType" validate:"required,oneof=pool bracket mixed"`
	Description   *string    `json:"description,omitempty"`
	AutoAdvance   bool       `json:"auto_advance,omitempty"`
	TargetRoundID *uuid.UUID `json:"target_round_id,omitempty"`
	TopNTeams     *int       `json:"top_n_teams,omitempty"`
}

type UpdateDivisionRequest struct {
	Name          *string    `json:"name"`
	DivisionType  *string    `json:"divisionType" validate:"omitempty,oneof=pool bracket mixed"`
	Description   *string    `json:"description"`
	AutoAdvance   *bool      `json:"auto_advance,omitempty"`
	TargetRoundID *uuid.UUID `json:"target_round_id,omitempty"`
	TopNTeams     *int       `json:"top_n_teams,omitempty"`
}

type AddCrewMemberRequest struct {
	UserID string `json:"userId" validate:"required"`
	Role   string `json:"role"`
}

type UpdateEventRequest struct {
	ID           *string                 `json:"id,omitempty"`
	Name         *string                 `json:"name"`
	Slug         *string                 `json:"slug"`
	Description  *string                 `json:"description"`
	StartDate    *types.JSONTime         `json:"startDate"`
	EndDate      *types.JSONTime         `json:"endDate"`
	DisciplineID *string                 `json:"disciplineId"`
	LocationID   *string                 `json:"locationId"`
	CategoryIDs  []string                `json:"categoryIds"`
	LogoUrl      *string                 `json:"logoUrl"`
	BannerUrl    *string                 `json:"bannerUrl"`
	RulesUrl     *string                 `json:"rulesUrl"`
	Status       *string                 `json:"status"`
	Metadata     map[string]interface{}  `json:"metadata"`
	Divisions    []CreateDivisionRequest `json:"divisions"`
	GameRoundIDs []string                `json:"gameRoundIds"`
}

type EventHandler struct {
	client *ent.Client
}

func NewEventHandler(client *ent.Client) *EventHandler {
	return &EventHandler{client: client}
}

// ============================================
// Response DTOs
// ============================================

// EventResponse represents an event in API responses
type EventResponse struct {
	ID          string           `json:"id"`
	Name        string           `json:"name"`
	Slug        string           `json:"slug"`
	Year        int              `json:"year"`
	StartDate   time.Time        `json:"startDate"`
	EndDate     time.Time        `json:"endDate"`
	Status      string           `json:"status"`
	Description *string          `json:"description,omitempty"`
	Categories  []RefDTO         `json:"categories,omitempty"`
	LogoUrl     *string          `json:"logoUrl,omitempty"`
	BannerUrl   *string          `json:"bannerUrl,omitempty"`
	RulesUrl    *string          `json:"rulesUrl,omitempty"`
	TeamsCount  int              `json:"teamsCount"`
	GamesCount  int              `json:"gamesCount"`
	Discipline  *RefDTO          `json:"discipline,omitempty"`
	Location    *LocationDTO     `json:"location,omitempty"`
	Divisions   []DivisionDTO    `json:"divisions,omitempty"`
	GameRounds  []RefDTO         `json:"gameRounds,omitempty"`
	TeamPreview []TeamPreviewDTO `json:"teamPreview,omitempty"`
}

type UserResponse struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	Email     string  `json:"email"`
	AvatarURL *string `json:"avatarUrl,omitempty"`
	Role      string  `json:"role"`
	IsActive  bool    `json:"isActive"`
}

type RefDTO struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type LocationDTO struct {
	ID      string      `json:"id"`
	Name    string      `json:"name"`
	City    *string     `json:"city,omitempty"`
	Country *CountryDTO `json:"country,omitempty"`
}

type CountryDTO struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Code string `json:"code"`
}

type DivisionDTO struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	DivisionType string `json:"divisionType"`
	TeamsCount   int    `json:"teamsCount"`
}

type TeamPreviewDTO struct {
	ID      string  `json:"id"`
	Name    string  `json:"name"`
	LogoUrl *string `json:"logoUrl,omitempty"`
}

// ============================================
// DTO Transformers
// ============================================

func toDivisionDTO(dp *ent.DivisionPool) DivisionDTO {
	teamsCount := 0
	if dp.Edges.Teams != nil {
		teamsCount = len(dp.Edges.Teams)
	}
	return DivisionDTO{
		ID:           dp.ID.String(),
		Name:         dp.Name,
		DivisionType: dp.DivisionType,
		TeamsCount:   teamsCount,
	}
}

func toEventResponse(e *ent.Event) EventResponse {
	// Calculate actual counts from loaded edges
	actualTeamsCount := 0
	actualGamesCount := 0

	// Count teams and games from all divisions
	for _, dp := range e.Edges.DivisionPools {
		if dp.Edges.Teams != nil {
			actualTeamsCount += len(dp.Edges.Teams)
		}
		if dp.Edges.Games != nil {
			actualGamesCount += len(dp.Edges.Games)
		}
	}

	// Use calculated count if stored count is 0
	teamsCount := e.TeamsCount
	if teamsCount == 0 {
		teamsCount = actualTeamsCount
	}
	gamesCount := e.GamesCount
	if gamesCount == 0 {
		gamesCount = actualGamesCount
	}

	resp := EventResponse{
		ID:          e.ID.String(),
		Name:        e.Name,
		Slug:        e.Slug,
		Year:        e.Year,
		StartDate:   e.StartDate,
		EndDate:     e.EndDate,
		Status:      string(e.Status),
		Description: e.Description,
		LogoUrl:     e.LogoURL,
		BannerUrl:   e.BannerURL,
		RulesUrl:    e.RulesURL,
		TeamsCount:  teamsCount,
		GamesCount:  gamesCount,
	}

	if e.Edges.Discipline != nil {
		resp.Discipline = &RefDTO{
			ID:   e.Edges.Discipline.ID.String(),
			Name: e.Edges.Discipline.Name,
		}
	}

	if e.Edges.Location != nil {
		loc := e.Edges.Location
		locDTO := &LocationDTO{
			ID:   loc.ID.String(),
			Name: loc.Name,
			City: loc.City,
		}
		if loc.Edges.Country != nil {
			locDTO.Country = &CountryDTO{
				ID:   loc.Edges.Country.ID.String(),
				Name: loc.Edges.Country.Name,
				Code: loc.Edges.Country.Code,
			}
		}
		resp.Location = locDTO
	}

	// attach categories
	if len(e.Edges.Categories) > 0 {
		resp.Categories = make([]RefDTO, len(e.Edges.Categories))
		for i, c := range e.Edges.Categories {
			resp.Categories[i] = RefDTO{ID: c.ID.String(), Name: c.Name}
		}
	}

	// Build divisions list
	if len(e.Edges.DivisionPools) > 0 {
		resp.Divisions = make([]DivisionDTO, len(e.Edges.DivisionPools))
		teamPreviewMap := make(map[string]bool)
		var teamPreviews []TeamPreviewDTO

		for i, dp := range e.Edges.DivisionPools {
			if dp.Edges.Teams != nil {
				// Collect team previews (max 5 total)
				for _, t := range dp.Edges.Teams {
					if len(teamPreviews) < 5 && !teamPreviewMap[t.ID.String()] {
						teamPreviewMap[t.ID.String()] = true
						teamPreviews = append(teamPreviews, TeamPreviewDTO{
							ID:      t.ID.String(),
							Name:    t.Name,
							LogoUrl: t.LogoURL,
						})
					}
				}
			}
			resp.Divisions[i] = toDivisionDTO(dp)
		}
		resp.TeamPreview = teamPreviews
	}

	// attach game rounds
	if len(e.Edges.GameRounds) > 0 {
		resp.GameRounds = make([]RefDTO, len(e.Edges.GameRounds))
		for i, r := range e.Edges.GameRounds {
			resp.GameRounds[i] = RefDTO{ID: r.ID.String(), Name: r.Name}
		}
	}

	return resp
}

// ============================================
// Create Event Handler
// ============================================

// CreateEvent godoc
// @Summary Create a new event
// @Description Create a new tournament or event
// @Tags events
// @Accept json
// @Produce json
// @Param request body CreateEventRequest true "Event data, categoryIds as list of UUIDs or slugs"
// @Success 201 {object} EventResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /events [post]
func (h *EventHandler) CreateEvent(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req CreateEventRequest
	if err := parseJSONBody(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	year := req.Year
	if year == 0 && !req.StartDate.IsZero() {
		year = req.StartDate.Year()
	}

	builder := h.client.Event.Create().
		SetName(req.Name).
		SetSlug(req.Slug).
		SetYear(year).
		SetStartDate(req.StartDate.Time).
		SetEndDate(req.EndDate.Time).
		SetStatus(req.Status)
	// optionally set rules URL if provided
	if req.RulesUrl != nil {
		builder.SetRulesURL(*req.RulesUrl)
	}

	if req.Description != nil {
		builder.SetDescription(*req.Description)
	}
	// attach categories by id or slug
	if len(req.CategoryIDs) > 0 {
		for _, cid := range req.CategoryIDs {
			if catID, err := uuid.Parse(cid); err == nil {
				builder.AddCategoryIDs(catID)
			} else {
				// try slug
				cat, err := h.client.Category.Query().Where(category.SlugEQ(cid)).Only(ctx)
				if err == nil && cat != nil {
					builder.AddCategories(cat)
				}
			}
		}
	}
	if req.LogoUrl != nil {
		builder.SetLogoURL(*req.LogoUrl)
	}
	if req.BannerUrl != nil {
		builder.SetBannerURL(*req.BannerUrl)
	}
	if req.RulesUrl != nil {
		builder.SetRulesURL(*req.RulesUrl)
	}
	if req.LocationID != nil && *req.LocationID != "" {
		if locID, err := uuid.Parse(*req.LocationID); err == nil {
			// verify existence to avoid FK constraint failures
			if _, err := h.client.Location.Query().Where(location.IDEQ(locID)).Only(ctx); err != nil {
				respondError(w, http.StatusBadRequest, "Location not found: "+*req.LocationID)
				return
			}
			builder.SetLocationID(locID)
		} else {
			// Try lookup by name/slug
			l, err := h.client.Location.Query().Where(location.SlugEQ(*req.LocationID)).Only(ctx)
			if err != nil {
				l, _ = h.client.Location.Query().Where(location.NameEQ(*req.LocationID)).Only(ctx)
			}
			if l != nil {
				builder.SetLocation(l)
			} else {
				respondError(w, http.StatusBadRequest, "Location not found: "+*req.LocationID)
				return
			}
		}
	}
	if req.DisciplineID != nil && *req.DisciplineID != "" {
		if discID, err := uuid.Parse(*req.DisciplineID); err == nil {
			// verify existence
			if _, err := h.client.Discipline.Query().Where(discipline.IDEQ(discID)).Only(ctx); err != nil {
				respondError(w, http.StatusBadRequest, "Discipline not found: "+*req.DisciplineID)
				return
			}
			builder.SetDisciplineID(discID)
		} else {
			// Find discipline by name/slug
			d, err := h.client.Discipline.Query().Where(discipline.SlugEQ(*req.DisciplineID)).Only(ctx)
			if err != nil {
				d, _ = h.client.Discipline.Query().Where(discipline.NameEQ(*req.DisciplineID)).Only(ctx)
			}
			if d != nil {
				builder.SetDiscipline(d)
			} else {
				respondError(w, http.StatusBadRequest, "Discipline not found: "+*req.DisciplineID)
				return
			}
		}
	}
	if req.Metadata != nil {
		builder.SetSettings(req.Metadata)
	}

	e, err := builder.Save(ctx)
	if err != nil {
		logger.Error("Failed to create event",
			logger.Err(err),
			logger.String("name", req.Name),
			logger.String("slug", req.Slug))

		// Map common errors to appropriate responses
		if strings.Contains(err.Error(), "duplicate key") {
			respondError(w, http.StatusConflict, "Event with this slug already exists")
			return
		}

		respondError(w, http.StatusInternalServerError, "Failed to create event: "+err.Error())
		return
	}

	// Create or Associate nested divisions if provided
	if len(req.Divisions) > 0 {
		for _, dReq := range req.Divisions {
			if dReq.ID != nil && *dReq.ID != "" {
				divID, err := uuid.Parse(*dReq.ID)
				if err == nil {
					h.client.DivisionPool.UpdateOneID(divID).AddEvents(e).Exec(ctx)
				}
			} else {
				_, err := h.client.DivisionPool.Create().
					SetName(dReq.Name).
					SetDivisionType(dReq.DivisionType).
					SetNillableDescription(dReq.Description).
					AddEvents(e).
					Save(ctx)
				if err != nil {
					logger.Error("Failed to create nested division", logger.Err(err), logger.String("event_id", e.ID.String()))
				}
			}
		}
	}

	// Associate game rounds if provided
	if len(req.GameRoundIDs) > 0 {
		for _, rid := range req.GameRoundIDs {
			if rUUID, err := uuid.Parse(rid); err == nil {
				h.client.GameRound.UpdateOneID(rUUID).AddEvents(e).Exec(ctx)
			}
		}
	}

	// Reload event to include details in response
	e, _ = h.client.Event.Query().
		Where(event.ID(e.ID)).
		WithDiscipline().
		WithLocation(func(lq *ent.LocationQuery) { lq.WithCountry() }).
		WithDivisionPools().
		WithGameRounds().
		WithCategories().
		Only(ctx)

	respondJSON(w, http.StatusCreated, toEventResponse(e))
}

// UpdateEvent updates an existing event
// @Summary Update an event
// @Tags events
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Event ID" format(uuid)
// @Param request body UpdateEventRequest true "Update Event Request"
// @Success 200 {object} EventResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /events/{id} [put]
func (h *EventHandler) UpdateEvent(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid event ID format")
		return
	}

	var req UpdateEventRequest
	if err := parseJSONBody(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	ctx := r.Context()

	// Check if event exists
	_, err = h.client.Event.Get(ctx, id)
	if err != nil {
		if ent.IsNotFound(err) {
			respondError(w, http.StatusNotFound, "Event not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "Failed to check event")
		return
	}

	// Prepare update
	updater := h.client.Event.UpdateOneID(id)

	if req.Name != nil {
		updater.SetName(*req.Name)
	}
	if req.Slug != nil {
		updater.SetSlug(*req.Slug)
	}
	if req.Description != nil {
		if *req.Description == "" {
			updater.ClearDescription()
		} else {
			updater.SetDescription(*req.Description)
		}
	}
	if req.StartDate != nil {
		updater.SetStartDate(req.StartDate.Time)
	}
	if req.EndDate != nil {
		updater.SetEndDate(req.EndDate.Time)
	}
	if req.Status != nil {
		updater.SetStatus(*req.Status)
	}
	if req.CategoryIDs != nil {
		// clear existing and add new
		updater.ClearCategories()
		for _, cid := range req.CategoryIDs {
			if catID, err := uuid.Parse(cid); err == nil {
				updater.AddCategoryIDs(catID)
			} else {
				cat, err := h.client.Category.Query().Where(category.SlugEQ(cid)).Only(ctx)
				if err == nil && cat != nil {
					updater.AddCategories(cat)
				}
			}
		}
	}
	if req.LogoUrl != nil {
		if *req.LogoUrl == "" {
			updater.ClearLogoURL()
		} else {
			updater.SetLogoURL(*req.LogoUrl)
		}
	}
	if req.BannerUrl != nil {
		if *req.BannerUrl == "" {
			updater.ClearBannerURL()
		} else {
			updater.SetBannerURL(*req.BannerUrl)
		}
	}
	if req.RulesUrl != nil {
		if *req.RulesUrl == "" {
			updater.ClearRulesURL()
		} else {
			updater.SetRulesURL(*req.RulesUrl)
		}
	}

	// Handle discipline
	if req.DisciplineID != nil && *req.DisciplineID != "" {
		if discID, err := uuid.Parse(*req.DisciplineID); err == nil {
			// verify exists
			if _, err := h.client.Discipline.Query().Where(discipline.IDEQ(discID)).Only(ctx); err != nil {
				respondError(w, http.StatusBadRequest, "Discipline not found: "+*req.DisciplineID)
				return
			}
			updater.SetDisciplineID(discID)
		} else {
			d, _ := h.client.Discipline.Query().Where(discipline.SlugEQ(*req.DisciplineID)).Only(ctx)
			if d != nil {
				updater.SetDiscipline(d)
			}
		}
	}

	// Handle location
	if req.LocationID != nil && *req.LocationID != "" {
		if locID, err := uuid.Parse(*req.LocationID); err == nil {
			if _, err := h.client.Location.Query().Where(location.IDEQ(locID)).Only(ctx); err != nil {
				respondError(w, http.StatusBadRequest, "Location not found: "+*req.LocationID)
				return
			}
			updater.SetLocationID(locID)
		} else {
			l, _ := h.client.Location.Query().Where(location.NameEQ(*req.LocationID)).Only(ctx)
			if l != nil {
				updater.SetLocation(l)
			}
		}
	}

	if req.Metadata != nil {
		updater.SetSettings(req.Metadata)
	}

	// Execute update
	eUpdated, err := updater.Save(ctx)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to update event")
		return
	}

	// Handle nested divisions if provided
	if req.Divisions != nil {
		for _, dReq := range req.Divisions {
			if dReq.ID != nil && *dReq.ID != "" {
				divID, err := uuid.Parse(*dReq.ID)
				if err == nil {
					h.client.DivisionPool.UpdateOneID(divID).AddEvents(eUpdated).Exec(ctx)
				}
			} else {
				// check if exists by name for this event
				exists, _ := h.client.DivisionPool.Query().
					Where(
						divisionpool.HasEventsWith(event.ID(eUpdated.ID)),
						divisionpool.NameEQ(dReq.Name),
					).Exist(ctx)
				if !exists {
					_, err := h.client.DivisionPool.Create().
						SetName(dReq.Name).
						SetDivisionType(dReq.DivisionType).
						SetNillableDescription(dReq.Description).
						AddEvents(eUpdated).
						Save(ctx)
					if err != nil {
						logger.Error("Failed to create nested division on update", logger.Err(err))
					}
				}
			}
		}
	}

	// Associate game rounds if provided
	if len(req.GameRoundIDs) > 0 {
		for _, rid := range req.GameRoundIDs {
			if rUUID, err := uuid.Parse(rid); err == nil {
				h.client.GameRound.UpdateOneID(rUUID).AddEvents(eUpdated).Exec(ctx)
			}
		}
	}

	// Reload with edges for response
	eUpdated, err = h.client.Event.Query().
		Where(event.ID(eUpdated.ID)).
		WithDiscipline().
		WithLocation(func(lq *ent.LocationQuery) { lq.WithCountry() }).
		WithDivisionPools().
		WithGameRounds().
		WithCategories().
		Only(ctx)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to load updated event")
		return
	}

	respondJSON(w, http.StatusOK, toEventResponse(eUpdated))
}

// ============================================
// Create Division Handler
// ============================================

// CreateDivisionPool godoc
// @Summary Create a new division pool
// @Description Create a new division pool for an event
// @Tags events
// @Accept json
// @Produce json
// @Param id path string true "Event ID" format(uuid)
// @Param request body CreateDivisionRequest true "Division data"
// @Success 201 {object} DivisionDTO
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /events/{id}/divisions [post]
func (h *EventHandler) CreateDivisionPool(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	eventIDStr := chi.URLParam(r, "id")
	eventID, err := uuid.Parse(eventIDStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid event ID path parameter")
		return
	}

	var req CreateDivisionRequest
	if err := parseJSONBody(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	builder := h.client.DivisionPool.Create().
		SetName(req.Name).
		SetDivisionType(req.DivisionType).
		SetNillableDescription(req.Description).
		AddEventIDs(eventID).
		SetAutoAdvance(req.AutoAdvance)

	if req.TopNTeams != nil {
		builder.SetTopNTeams(*req.TopNTeams)
	}

	if req.TargetRoundID != nil {
		builder.SetTargetRoundID(*req.TargetRoundID)
	}

	dp, err := builder.Save(ctx)

	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to create division pool")
		return
	}

	respondJSON(w, http.StatusCreated, DivisionDTO{
		ID:           dp.ID.String(),
		Name:         dp.Name,
		DivisionType: dp.DivisionType,
		TeamsCount:   0,
	})
}

// UpdateDivisionPool godoc
// @Summary Update a division pool
// @Description Update an existing division pool
// @Tags events
// @Accept json
// @Produce json
// @Param id path string true "Event ID" format(uuid)
// @Param divisionId path string true "Division ID" format(uuid)
// @Param request body UpdateDivisionRequest true "Division data"
// @Success 200 {object} DivisionDTO
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /events/{id}/divisions/{divisionId} [put]
func (h *EventHandler) UpdateDivisionPool(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	divisionID, err := uuid.Parse(chi.URLParam(r, "divisionId"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid division ID")
		return
	}

	var req UpdateDivisionRequest
	if err := parseJSONBody(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	updater := h.client.DivisionPool.UpdateOneID(divisionID)
	if req.Name != nil {
		updater.SetName(*req.Name)
	}
	if req.DivisionType != nil {
		updater.SetDivisionType(*req.DivisionType)
	}
	if req.Description != nil {
		updater.SetNillableDescription(req.Description)
	}
	if req.AutoAdvance != nil {
		updater.SetAutoAdvance(*req.AutoAdvance)
	}
	if req.TopNTeams != nil {
		updater.SetTopNTeams(*req.TopNTeams)
	}
	if req.TargetRoundID != nil {
		updater.SetTargetRoundID(*req.TargetRoundID)
	}

	dp, err := updater.Save(ctx)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to update division pool")
		return
	}

	respondJSON(w, http.StatusOK, DivisionDTO{
		ID:           dp.ID.String(),
		Name:         dp.Name,
		DivisionType: dp.DivisionType,
		TeamsCount:   0, // Would need query to get actual count
	})
}

// DeleteDivisionPool godoc
// @Summary Delete a division pool
// @Description Delete an existing division pool
// @Tags events
// @Param id path string true "Event ID" format(uuid)
// @Param divisionId path string true "Division ID" format(uuid)
// @Success 204 "No Content"
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /events/{id}/divisions/{divisionId} [delete]
func (h *EventHandler) DeleteDivisionPool(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	divisionID, err := uuid.Parse(chi.URLParam(r, "divisionId"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid division ID")
		return
	}

	err = h.client.DivisionPool.DeleteOneID(divisionID).Exec(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			respondError(w, http.StatusNotFound, "Division pool not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "Failed to delete division pool")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ============================================
// List Events Handler
// ============================================

// ListEvents godoc
// @Summary List events
// @Description List all events with optional filtering
// @Tags events
// @Produce json
// @Param status query string false "Filter by status (draft, published, in_progress, completed, canceled)"
// @Param year query int false "Filter by year"
// @Param temporal query string false "Filter by time: past, upcoming, live, all"
// @Param category query []string false "Filter by category ID or slug (can repeat)"
// @Param country query string false "Filter by country code (2-letter ISO)"
// @Param search query string false "Search in name and description"
// @Param startDateFrom query string false "Events starting after this date (RFC3339)"
// @Param startDateTo query string false "Events starting before this date (RFC3339)"
// @Param sortBy query string false "Sort by field: start_date, name, teams_count" default(start_date)
// @Param sortOrder query string false "Sort order: asc, desc" default(desc)
// @Param limit query int false "Limit results" default(50)
// @Param offset query int false "Offset for pagination" default(0)
// @Success 200 {array} EventResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /public/events [get]
func (h *EventHandler) ListEvents(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	query := h.client.Event.Query().
		Where(event.DeletedAtIsNil()).
		WithDiscipline().
		WithLocation(func(lq *ent.LocationQuery) {
			lq.WithCountry()
		}).
		WithDivisionPools(func(dpq *ent.DivisionPoolQuery) {
			dpq.WithTeams().WithGames()
		}).
		WithCategories()

	// Filter by status
	if status := r.URL.Query().Get("status"); status != "" {
		query = query.Where(event.StatusEQ(status))
	}

	// Filter by year
	if yearStr := r.URL.Query().Get("year"); yearStr != "" {
		if year, err := strconv.Atoi(yearStr); err == nil && year > 0 {
			query = query.Where(event.YearEQ(year))
		}
	}

	// Temporal filter (past/upcoming/live)
	if temporal := r.URL.Query().Get("temporal"); temporal != "" {
		now := time.Now()
		switch temporal {
		case "past":
			query = query.Where(event.EndDateLT(now))
		case "upcoming":
			query = query.Where(event.StartDateGT(now))
		case "live":
			query = query.Where(event.StartDateLTE(now), event.EndDateGTE(now))
		}
	}

	// Filter by categories (ids or slugs)
	if cats := r.URL.Query()["category"]; len(cats) > 0 {
		var catPreds []predicate.Category
		for _, c := range cats {
			if id, err := uuid.Parse(c); err == nil {
				catPreds = append(catPreds, category.IDEQ(id))
			} else {
				catPreds = append(catPreds, category.SlugEQ(c))
			}
		}
		if len(catPreds) > 0 {
			query = query.Where(event.HasCategoriesWith(category.Or(catPreds...)))
		}
	}

	// Filter by country code
	if countryCode := r.URL.Query().Get("country"); countryCode != "" {
		query = query.Where(
			event.HasLocationWith(
				location.HasCountryWith(country.CodeEQ(strings.ToUpper(countryCode))),
			),
		)
	}

	// Search filter (name or description)
	if search := r.URL.Query().Get("search"); search != "" {
		query = query.Where(
			event.Or(
				event.NameContainsFold(search),
				event.DescriptionContainsFold(search),
			),
		)
	}

	// Date range filters
	if startDateFrom := r.URL.Query().Get("startDateFrom"); startDateFrom != "" {
		if t, err := time.Parse(time.RFC3339, startDateFrom); err == nil {
			query = query.Where(event.StartDateGTE(t))
		}
	}
	if startDateTo := r.URL.Query().Get("startDateTo"); startDateTo != "" {
		if t, err := time.Parse(time.RFC3339, startDateTo); err == nil {
			query = query.Where(event.StartDateLTE(t))
		}
	}

	// Sorting
	sortBy := r.URL.Query().Get("sortBy")
	sortOrder := r.URL.Query().Get("sortOrder")
	if sortOrder == "" {
		sortOrder = "desc"
	}

	switch sortBy {
	case "name":
		if sortOrder == "asc" {
			query = query.Order(ent.Asc(event.FieldName))
		} else {
			query = query.Order(ent.Desc(event.FieldName))
		}
	case "teams_count":
		if sortOrder == "asc" {
			query = query.Order(ent.Asc(event.FieldTeamsCount))
		} else {
			query = query.Order(ent.Desc(event.FieldTeamsCount))
		}
	default: // start_date
		if sortOrder == "asc" {
			query = query.Order(ent.Asc(event.FieldStartDate))
		} else {
			query = query.Order(ent.Desc(event.FieldStartDate))
		}
	}

	// Pagination
	pagination := ParsePagination(r)

	total, err := query.Count(ctx)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to count events")
		return
	}

	events, err := query.
		Limit(pagination.Limit).
		Offset(pagination.Offset).
		All(ctx)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to list events")
		return
	}

	// Transform to response
	response := make([]EventResponse, len(events))
	for i, e := range events {
		response[i] = toEventResponse(e)
	}

	respondJSON(w, http.StatusOK, NewPaginatedResponse(response, total, pagination.Limit, pagination.Offset))
}

// ============================================
// Get Event Handler
// ============================================

// GetEvent godoc
// @Summary Get an event
// @Description Get an event by ID or slug
// @Tags events
// @Produce json
// @Param id path string true "Event ID or slug"
// @Success 200 {object} EventResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /public/events/{id} [get]
func (h *EventHandler) GetEvent(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	idOrSlug := chi.URLParam(r, "id")

	var e *ent.Event
	var err error

	baseQuery := func() *ent.EventQuery {
		return h.client.Event.Query().
			Where(event.DeletedAtIsNil()).
			WithDiscipline().
			WithLocation(func(lq *ent.LocationQuery) {
				lq.WithCountry()
			}).
			WithDivisionPools(func(dpq *ent.DivisionPoolQuery) {
				dpq.WithTeams().WithGames()
			}).
			WithCategories()
	}

	// Try parsing as UUID first
	if eventID, parseErr := uuid.Parse(idOrSlug); parseErr == nil {
		e, err = baseQuery().
			Where(event.ID(eventID)).
			Only(ctx)
	} else {
		// Fall back to slug lookup
		e, err = baseQuery().
			Where(event.SlugEQ(idOrSlug)).
			Only(ctx)
	}

	if err != nil {
		if ent.IsNotFound(err) {
			respondError(w, http.StatusNotFound, "Event not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "Failed to get event")
		return
	}

	respondJSON(w, http.StatusOK, toEventResponse(e))
}

// AddEventCrewMember adds a user to the event staff

// ListDivisionsByEvent returns all divisions for an event
func (h *EventHandler) ListDivisionsByEvent(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid event ID")
		return
	}

	divisions, err := h.client.DivisionPool.Query().
		Where(divisionpool.HasEventsWith(event.ID(id))).
		WithTeams().
		All(ctx)

	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to list divisions")
		return
	}

	result := make([]DivisionDTO, len(divisions))
	for i, dp := range divisions {
		teamsCount := 0
		if dp.Edges.Teams != nil {
			teamsCount = len(dp.Edges.Teams)
		}
		result[i] = DivisionDTO{
			ID:           dp.ID.String(),
			Name:         dp.Name,
			DivisionType: dp.DivisionType,
			TeamsCount:   teamsCount,
		}
	}

	respondJSON(w, http.StatusOK, result)
}

// GetEventCrew returns all crew members for an event
func (h *EventHandler) GetEventCrew(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	eventID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid event ID")
		return
	}

	// Filtering by division
	divisionIDStr := r.URL.Query().Get("division_pool_id")
	var divisionID *uuid.UUID
	if divisionIDStr != "" {
		if id, err := uuid.Parse(divisionIDStr); err == nil {
			divisionID = &id
		}
	}

	// 1. Get explicitly scoped roles
	rolesQuery := h.client.ScopedRole.Query()

	if divisionID != nil {
		rolesQuery = rolesQuery.Where(
			scopedrole.Or(
				scopedrole.And(
					scopedrole.ScopeTypeEQ("event"),
					scopedrole.ScopeIDEQ(eventID),
				),
				scopedrole.And(
					scopedrole.ScopeTypeEQ("division"),
					scopedrole.ScopeIDEQ(*divisionID),
				),
			),
		)
	} else {
		rolesQuery = rolesQuery.Where(
			scopedrole.ScopeTypeEQ("event"),
			scopedrole.ScopeIDEQ(eventID),
		)
	}

	roles, err := rolesQuery.WithUser().All(ctx)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to fetch event crew")
		return
	}

	userMap := make(map[uuid.UUID]UserResponse)
	admins := []UserResponse{}
	scorekeepers := []UserResponse{}

	for _, sr := range roles {
		if sr.Edges.User == nil {
			continue
		}
		u := UserResponse{
			ID:        sr.Edges.User.ID.String(),
			Name:      sr.Edges.User.Name,
			Email:     sr.Edges.User.Email,
			AvatarURL: sr.Edges.User.AvatarURL,
			Role:      sr.Edges.User.Role,
			IsActive:  sr.Edges.User.IsActive,
		}
		userMap[sr.Edges.User.ID] = u
		if sr.Role == "admin" || sr.Role == "event_manager" {
			admins = append(admins, u)
		} else {
			scorekeepers = append(scorekeepers, u)
		}
	}

	// 2. Get scorekeepers assigned to games in this event/division
	var skUserQuery *ent.UserQuery
	if divisionID != nil {
		skUserQuery = h.client.User.Query().
			Where(
				entUser.HasOfficiatedGamesWith(
					entGame.HasDivisionPoolWith(divisionpool.IDEQ(*divisionID)),
				),
			)
	} else {
		skUserQuery = h.client.User.Query().
			Where(
				entUser.HasOfficiatedGamesWith(
					entGame.HasDivisionPoolWith(
						divisionpool.HasEventsWith(event.IDEQ(eventID)),
					),
				),
			)
	}

	skUsers, err := skUserQuery.All(ctx)
	if err == nil {
		for _, u := range skUsers {
			if _, exists := userMap[u.ID]; !exists {
				resp := UserResponse{
					ID:        u.ID.String(),
					Name:      u.Name,
					Email:     u.Email,
					AvatarURL: u.AvatarURL,
					Role:      u.Role,
					IsActive:  u.IsActive,
				}
				scorekeepers = append(scorekeepers, resp)
				userMap[u.ID] = resp
			}
		}
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"admins":       admins,
		"scorekeepers": scorekeepers,
	})
}

// AddEventCrewMember adds a user to the event crew
func (h *EventHandler) AddEventCrewMember(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	eventID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid event ID")
		return
	}

	var req AddCrewMemberRequest
	if err := parseJSONBody(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	role := req.Role
	if role == "" {
		role = "scorekeeper"
	}

	// Check if role already exists
	exists, err := h.client.ScopedRole.Query().
		Where(
			scopedrole.UserIDEQ(userID),
			scopedrole.ScopeTypeEQ("event"),
			scopedrole.ScopeIDEQ(eventID),
			scopedrole.RoleEQ(role),
		).
		Exist(ctx)

	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to check existing crew member")
		return
	}

	if exists {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	_, err = h.client.ScopedRole.Create().
		SetUserID(userID).
		SetScopeType("event").
		SetScopeID(eventID).
		SetRole(role).
		Save(ctx)

	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to add crew member")
		return
	}

	w.WriteHeader(http.StatusCreated)
}

// RemoveEventCrewMember removes a user from the event crew
func (h *EventHandler) RemoveEventCrewMember(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	eventID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid event ID")
		return
	}

	userID, err := uuid.Parse(chi.URLParam(r, "userId"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	_, err = h.client.ScopedRole.Delete().
		Where(
			scopedrole.UserIDEQ(userID),
			scopedrole.ScopeTypeEQ("event"),
			scopedrole.ScopeIDEQ(eventID),
		).
		Exec(ctx)

	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to remove crew member")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ListAllDivisions lists all divisions in the system
func (h *EventHandler) ListAllDivisions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	pagination := ParsePagination(r)
	total, err := h.client.DivisionPool.Query().Where(divisionpool.DeletedAtIsNil()).Count(ctx)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to count divisions")
		return
	}

	divisions, err := h.client.DivisionPool.Query().
		Where(divisionpool.DeletedAtIsNil()).
		Limit(pagination.Limit).
		Offset(pagination.Offset).
		All(ctx)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to list divisions")
		return
	}

	res := make([]DivisionDTO, len(divisions))
	for i, d := range divisions {
		res[i] = toDivisionDTO(d)
	}

	respondJSON(w, http.StatusOK, NewPaginatedResponse(res, total, pagination.Limit, pagination.Offset))
}
