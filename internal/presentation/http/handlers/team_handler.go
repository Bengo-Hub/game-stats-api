package handlers

import (
	"net/http"

	"github.com/bengobox/game-stats-api/ent"
	"github.com/bengobox/game-stats-api/ent/divisionpool"
	"github.com/bengobox/game-stats-api/ent/event"
	"github.com/bengobox/game-stats-api/ent/team"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// ============================================
// Request DTOs
// ============================================

type CreateTeamRequest struct {
	Name           string                 `json:"name" validate:"required"`
	EventID        uuid.UUID              `json:"eventId" validate:"required"`
	DivisionPoolID uuid.UUID              `json:"divisionPoolId" validate:"required"`
	HomeLocationID *uuid.UUID             `json:"homeLocationId,omitempty"`
	LogoURL        *string                `json:"logoUrl,omitempty"`
	InitialSeed    *int                   `json:"initialSeed,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

type CreatePlayerRequest struct {
	Name            string    `json:"name" validate:"required"`
	EventID         uuid.UUID `json:"eventId" validate:"required"`
	TeamID          uuid.UUID `json:"teamId" validate:"required"`
	Gender          string    `json:"gender" validate:"required,oneof=M F X"`
	JerseyNumber    *int      `json:"jerseyNumber,omitempty"`
	ProfileImageURL *string   `json:"profileImageUrl,omitempty"`
	IsCaptain       bool      `json:"isCaptain"`
	IsSpiritCaptain bool      `json:"isSpiritCaptain"`
}

type TeamHandler struct {
	client *ent.Client
}

func NewTeamHandler(client *ent.Client) *TeamHandler {
	return &TeamHandler{client: client}
}

// PlayerResponse represents a player in API responses
type PlayerResponse struct {
	ID              string  `json:"id"`
	Name            string  `json:"name"`
	Gender          string  `json:"gender"`
	JerseyNumber    *int    `json:"jerseyNumber,omitempty"`
	ProfileImageURL *string `json:"profileImageUrl,omitempty"`
	IsCaptain       bool    `json:"isCaptain"`
	IsSpiritCaptain bool    `json:"isSpiritCaptain"`
}

// TeamResponse represents a team in API responses
type TeamResponse struct {
	ID             string                 `json:"id"`
	Name           string                 `json:"name"`
	InitialSeed    *int                   `json:"initialSeed,omitempty"`
	FinalPlacement *int                   `json:"finalPlacement,omitempty"`
	LogoURL        *string                `json:"logoUrl,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
	DivisionPoolID *string                `json:"divisionPoolId,omitempty"`
	HomeLocationID *string                `json:"homeLocationId,omitempty"`
	LocationName   *string                `json:"locationName,omitempty"`
	DivisionName   *string                `json:"divisionName,omitempty"`
	Players        []PlayerResponse       `json:"players,omitempty"`
	Captain        *PlayerResponse        `json:"captain,omitempty"`
	SpiritCaptain  *PlayerResponse        `json:"spiritCaptain,omitempty"`
	PlayersCount   int                    `json:"playersCount"`
}

func toPlayerResponse(p *ent.Player) PlayerResponse {
	resp := PlayerResponse{
		ID:              p.ID.String(),
		Name:            p.Name,
		Gender:          p.Gender,
		IsCaptain:       p.IsCaptain,
		IsSpiritCaptain: p.IsSpiritCaptain,
	}
	if p.JerseyNumber != nil {
		resp.JerseyNumber = p.JerseyNumber
	}
	if p.ProfileImageURL != nil {
		resp.ProfileImageURL = p.ProfileImageURL
	}
	return resp
}

func toTeamResponse(t *ent.Team) TeamResponse {
	resp := TeamResponse{
		ID:       t.ID.String(),
		Name:     t.Name,
		Metadata: t.Metadata,
	}

	if t.InitialSeed != nil {
		resp.InitialSeed = t.InitialSeed
	}
	if t.FinalPlacement != nil {
		resp.FinalPlacement = t.FinalPlacement
	}
	if t.LogoURL != nil {
		resp.LogoURL = t.LogoURL
	}

	if t.Edges.DivisionPool != nil {
		id := t.Edges.DivisionPool.ID.String()
		resp.DivisionPoolID = &id
		resp.DivisionName = &t.Edges.DivisionPool.Name
	}

	if t.Edges.HomeLocation != nil {
		id := t.Edges.HomeLocation.ID.String()
		resp.HomeLocationID = &id
		resp.LocationName = t.Edges.HomeLocation.City
	}

	// Process players if loaded
	if t.Edges.Players != nil {
		resp.PlayersCount = len(t.Edges.Players)
		resp.Players = make([]PlayerResponse, len(t.Edges.Players))
		for i, p := range t.Edges.Players {
			resp.Players[i] = toPlayerResponse(p)
			// Identify captain and spirit captain
			if p.IsCaptain {
				playerResp := toPlayerResponse(p)
				resp.Captain = &playerResp
			}
			if p.IsSpiritCaptain {
				playerResp := toPlayerResponse(p)
				resp.SpiritCaptain = &playerResp
			}
		}
	}

	return resp
}

// ListTeams godoc
// @Summary List teams
// @Description List all teams with optional filtering
// @Tags teams
// @Produce json
// @Param eventId query string false "Filter by event ID" format(uuid)
// @Param divisionPoolId query string false "Filter by division pool ID" format(uuid)
// @Param search query string false "Search by team name"
// @Param limit query int false "Limit results" default(50)
// @Param offset query int false "Offset for pagination" default(0)
// @Success 200 {array} TeamResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /public/teams [get]
func (h *TeamHandler) ListTeams(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse query parameters
	query := h.client.Team.Query().
		Where(team.DeletedAtIsNil()).
		WithDivisionPool(func(dpq *ent.DivisionPoolQuery) {
			dpq.WithEvent()
		}).
		WithHomeLocation().
		WithPlayers()

	// Filter by event ID (teams in any division pool of this event)
	if eventIDStr := r.URL.Query().Get("eventId"); eventIDStr != "" {
		eventID, err := uuid.Parse(eventIDStr)
		if err != nil {
			respondError(w, http.StatusBadRequest, "Invalid event ID")
			return
		}
		query = query.Where(team.HasDivisionPoolWith(divisionpool.HasEventWith(event.ID(eventID))))
	}

	// Filter by division pool
	if divisionPoolIDStr := r.URL.Query().Get("divisionPoolId"); divisionPoolIDStr != "" {
		divisionPoolID, err := uuid.Parse(divisionPoolIDStr)
		if err != nil {
			respondError(w, http.StatusBadRequest, "Invalid division pool ID")
			return
		}
		query = query.Where(team.HasDivisionPoolWith(divisionpool.ID(divisionPoolID)))
	}

	// Search by name
	if search := r.URL.Query().Get("search"); search != "" {
		query = query.Where(team.NameContainsFold(search))
	}

	// Pagination
	pagination := ParsePagination(r)

	teams, err := query.
		Limit(pagination.Limit).
		Offset(pagination.Offset).
		Order(ent.Asc(team.FieldName)).
		All(ctx)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to list teams")
		return
	}

	// Transform to response
	response := make([]TeamResponse, len(teams))
	for i, t := range teams {
		response[i] = toTeamResponse(t)
	}

	respondJSON(w, http.StatusOK, response)
}

// GetTeam godoc
// @Summary Get a team
// @Description Get a team by ID
// @Tags teams
// @Produce json
// @Param id path string true "Team ID" format(uuid)
// @Success 200 {object} TeamResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /public/teams/{id} [get]
func (h *TeamHandler) GetTeam(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	teamIDStr := chi.URLParam(r, "id")
	teamID, err := uuid.Parse(teamIDStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid team ID")
		return
	}

	t, err := h.client.Team.Query().
		Where(team.ID(teamID)).
		Where(team.DeletedAtIsNil()).
		WithDivisionPool().
		WithHomeLocation().
		WithPlayers().
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			respondError(w, http.StatusNotFound, "Team not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "Failed to get team")
		return
	}

	respondJSON(w, http.StatusOK, toTeamResponse(t))
}

// ============================================
// Create Team Handler
// ============================================

// CreateTeam godoc
// @Summary Create a new team
// @Description Create a new team for an event
// @Tags teams
// @Accept json
// @Produce json
// @Param request body CreateTeamRequest true "Team data"
// @Success 201 {object} TeamResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /teams [post]
func (h *TeamHandler) CreateTeam(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req CreateTeamRequest
	if err := parseJSONBody(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	builder := h.client.Team.Create().
		SetName(req.Name).
		SetDivisionPoolID(req.DivisionPoolID)

	if req.HomeLocationID != nil {
		builder.SetHomeLocationID(*req.HomeLocationID)
	}
	if req.LogoURL != nil {
		builder.SetLogoURL(*req.LogoURL)
	}
	if req.InitialSeed != nil {
		builder.SetInitialSeed(*req.InitialSeed)
	}
	if req.Metadata != nil {
		builder.SetMetadata(req.Metadata)
	}

	t, err := builder.Save(ctx)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to create team")
		return
	}

	// Refetch to get related division pool and event data
	tFull, err := h.client.Team.Query().
		Where(team.ID(t.ID)).
		WithDivisionPool().
		WithHomeLocation().
		WithPlayers().
		Only(ctx)

	if err != nil {
		// Output the basic model if full fetch fails
		respondJSON(w, http.StatusCreated, toTeamResponse(t))
		return
	}

	respondJSON(w, http.StatusCreated, toTeamResponse(tFull))
}

// ============================================
// Create Player Handler
// ============================================

// CreatePlayer godoc
// @Summary Add a player to a team
// @Description Add a new player to an existing team
// @Tags teams
// @Accept json
// @Produce json
// @Param id path string true "Team ID" format(uuid)
// @Param request body CreatePlayerRequest true "Player data"
// @Success 201 {object} PlayerResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /teams/{id}/players [post]
func (h *TeamHandler) CreatePlayer(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	teamIDStr := chi.URLParam(r, "id")
	teamID, err := uuid.Parse(teamIDStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid team ID path parameter")
		return
	}

	var req CreatePlayerRequest
	if err := parseJSONBody(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Verify path matches request
	if req.TeamID != teamID {
		respondError(w, http.StatusBadRequest, "Team ID in path and body must match")
		return
	}

	builder := h.client.Player.Create().
		SetName(req.Name).
		SetGender(req.Gender).
		SetTeamID(teamID).
		SetIsCaptain(req.IsCaptain).
		SetIsSpiritCaptain(req.IsSpiritCaptain)

	if req.JerseyNumber != nil {
		builder.SetJerseyNumber(*req.JerseyNumber)
	}
	if req.ProfileImageURL != nil {
		builder.SetProfileImageURL(*req.ProfileImageURL)
	}

	p, err := builder.Save(ctx)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to create player")
		return
	}

	respondJSON(w, http.StatusCreated, toPlayerResponse(p))
}
