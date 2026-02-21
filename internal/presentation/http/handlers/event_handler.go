package handlers

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqljson"
	"github.com/bengobox/game-stats-api/ent"
	"github.com/bengobox/game-stats-api/ent/country"
	"github.com/bengobox/game-stats-api/ent/event"
	"github.com/bengobox/game-stats-api/ent/location"
	"github.com/bengobox/game-stats-api/ent/predicate"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// Note: Pagination uses ParsePagination from pagination.go

// ============================================
// Request DTOs
// ============================================

type CreateEventRequest struct {
	Name        string     `json:"name" validate:"required"`
	Slug        string     `json:"slug" validate:"required"`
	Year        int        `json:"year"`
	StartDate   time.Time  `json:"startDate"`
	EndDate     time.Time  `json:"endDate"`
	Status      string     `json:"status"`
	Description *string    `json:"description,omitempty"`
	Categories  []string   `json:"categories,omitempty"`
	LogoUrl     *string    `json:"logoUrl,omitempty"`
	BannerUrl   *string    `json:"bannerUrl,omitempty"`
	LocationID  *uuid.UUID `json:"locationId,omitempty"`
}

type CreateDivisionRequest struct {
	Name         string `json:"name" validate:"required"`
	DivisionType string `json:"divisionType" validate:"required,oneof=pool bracket"`
}

type UpdateEventRequest struct {
	Name         *string    `json:"name"`
	Slug         *string    `json:"slug"`
	Description  *string    `json:"description"`
	StartDate    *time.Time `json:"startDate"`
	EndDate      *time.Time `json:"endDate"`
	DisciplineID *uuid.UUID `json:"disciplineId"`
	LocationID   *uuid.UUID `json:"locationId"`
	Categories   []string   `json:"categories"`
	LogoUrl      *string    `json:"logoUrl"`
	BannerUrl    *string    `json:"bannerUrl"`
	Status       *string    `json:"status"`
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
	Categories  []string         `json:"categories,omitempty"`
	LogoUrl     *string          `json:"logoUrl,omitempty"`
	BannerUrl   *string          `json:"bannerUrl,omitempty"`
	TeamsCount  int              `json:"teamsCount"`
	GamesCount  int              `json:"gamesCount"`
	Discipline  *RefDTO          `json:"discipline,omitempty"`
	Location    *LocationDTO     `json:"location,omitempty"`
	Divisions   []DivisionDTO    `json:"divisions,omitempty"`
	TeamPreview []TeamPreviewDTO `json:"teamPreview,omitempty"`
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
		Categories:  e.Categories,
		LogoUrl:     e.LogoURL,
		BannerUrl:   e.BannerURL,
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

	// Build divisions list
	if len(e.Edges.DivisionPools) > 0 {
		resp.Divisions = make([]DivisionDTO, len(e.Edges.DivisionPools))
		teamPreviewMap := make(map[string]bool)
		var teamPreviews []TeamPreviewDTO

		for i, dp := range e.Edges.DivisionPools {
			teamsInDivision := 0
			if dp.Edges.Teams != nil {
				teamsInDivision = len(dp.Edges.Teams)
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
			resp.Divisions[i] = DivisionDTO{
				ID:           dp.ID.String(),
				Name:         dp.Name,
				DivisionType: dp.DivisionType,
				TeamsCount:   teamsInDivision,
			}
		}
		resp.TeamPreview = teamPreviews
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
// @Param request body CreateEventRequest true "Event data"
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

	builder := h.client.Event.Create().
		SetName(req.Name).
		SetSlug(req.Slug).
		SetYear(req.Year).
		SetStartDate(req.StartDate).
		SetEndDate(req.EndDate).
		SetStatus(req.Status)

	if req.Description != nil {
		builder.SetDescription(*req.Description)
	}
	if req.Categories != nil {
		builder.SetCategories(req.Categories)
	}
	if req.LogoUrl != nil {
		builder.SetLogoURL(*req.LogoUrl)
	}
	if req.BannerUrl != nil {
		builder.SetBannerURL(*req.BannerUrl)
	}
	if req.LocationID != nil {
		builder.SetLocationID(*req.LocationID)
	}

	e, err := builder.Save(ctx)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to create event")
		return
	}

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
		updater.SetStartDate(*req.StartDate)
	}
	if req.EndDate != nil {
		updater.SetEndDate(*req.EndDate)
	}
	if req.Status != nil {
		updater.SetStatus(*req.Status)
	}
	if req.Categories != nil {
		updater.SetCategories(req.Categories)
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

	// Handle discipline
	if req.DisciplineID != nil {
		updater.SetDisciplineID(*req.DisciplineID)
	}

	// Handle location
	if req.LocationID != nil {
		updater.SetLocationID(*req.LocationID)
	}

	// Execute update
	eUpdated, err := updater.Save(ctx)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to update event")
		return
	}

	// Reload with edges for response
	eUpdated, err = h.client.Event.Query().
		Where(event.ID(eUpdated.ID)).
		WithDiscipline().
		WithLocation(func(lq *ent.LocationQuery) {
			lq.WithCountry()
		}).
		WithDivisionPools(func(dpq *ent.DivisionPoolQuery) {
			dpq.WithTeams()
			dpq.WithGames()
		}).
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

	dp, err := h.client.DivisionPool.Create().
		SetName(req.Name).
		SetDivisionType(req.DivisionType).
		SetEventID(eventID).
		Save(ctx)

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
// @Param category query []string false "Filter by categories (outdoor, hat, beach, indoor, league)"
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
		})

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

	// Filter by categories
	if categories := r.URL.Query()["category"]; len(categories) > 0 {
		// OR logic: event has ANY of the specified categories
		var categoryPredicates []predicate.Event
		for _, cat := range categories {
			c := cat // capture loop variable
			categoryPredicates = append(categoryPredicates, func(s *sql.Selector) {
				s.Where(sqljson.ValueContains(event.FieldCategories, c))
			})
		}
		if len(categoryPredicates) > 0 {
			query = query.Where(event.Or(categoryPredicates...))
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

	respondJSON(w, http.StatusOK, response)
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
				dpq.WithTeams()
			})
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

// ============================================
// Event Crew/Staff Handler
// ============================================

// CrewMemberDTO represents a staff member in API responses
type CrewMemberDTO struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	Role      string  `json:"role"`
	AvatarUrl *string `json:"avatarUrl,omitempty"`
}

// EventCrewResponse contains event staff information
type EventCrewResponse struct {
	EventID      string          `json:"eventId"`
	EventName    string          `json:"eventName"`
	Admins       []CrewMemberDTO `json:"admins"`
	Scorekeepers []CrewMemberDTO `json:"scorekeepers"`
}

// GetEventCrew godoc
// @Summary Get event crew/staff
// @Description Get tournament admins and scorekeepers for an event
// @Tags events
// @Produce json
// @Param id path string true "Event ID or slug"
// @Success 200 {object} EventCrewResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /public/events/{id}/crew [get]
func (h *EventHandler) GetEventCrew(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	idOrSlug := chi.URLParam(r, "id")

	var e *ent.Event
	var err error

	baseQuery := func() *ent.EventQuery {
		return h.client.Event.Query().
			Where(event.DeletedAtIsNil()).
			WithManagedBy()
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

	// Build crew response
	response := EventCrewResponse{
		EventID:      e.ID.String(),
		EventName:    e.Name,
		Admins:       []CrewMemberDTO{},
		Scorekeepers: []CrewMemberDTO{},
	}

	// Categorize users by role
	for _, user := range e.Edges.ManagedBy {
		member := CrewMemberDTO{
			ID:        user.ID.String(),
			Name:      user.Name,
			Role:      user.Role,
			AvatarUrl: user.AvatarURL,
		}

		// Categorize by role
		switch user.Role {
		case "admin", "event_admin", "tournament_admin", "organizer":
			response.Admins = append(response.Admins, member)
		case "scorekeeper", "referee", "official":
			response.Scorekeepers = append(response.Scorekeepers, member)
		default:
			// Add to admins by default if they're managing the event
			response.Admins = append(response.Admins, member)
		}
	}

	respondJSON(w, http.StatusOK, response)
}
