package handlers

import (
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/bengobox/game-stats-api/ent"
	"github.com/bengobox/game-stats-api/ent/divisionpool"
	"github.com/bengobox/game-stats-api/ent/event"
	"github.com/bengobox/game-stats-api/ent/player"
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
	TeamID          *string `json:"teamId,omitempty"`
	TeamName        *string `json:"teamName,omitempty"`
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
	EventID        *string                `json:"eventId,omitempty"`
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
	if p.Edges.Team != nil {
		id := p.Edges.Team.ID.String()
		resp.TeamID = &id
		resp.TeamName = &p.Edges.Team.Name
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

		if t.Edges.DivisionPool.Edges.Event != nil {
			eventID := t.Edges.DivisionPool.Edges.Event.ID.String()
			resp.EventID = &eventID
		}
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
		WithDivisionPool(func(dpq *ent.DivisionPoolQuery) {
			dpq.WithEvent()
		}).
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

// GetPlayer godoc
// @Summary Get a player
// @Description Get a player by ID
// @Tags players
// @Produce json
// @Param id path string true "Player ID" format(uuid)
// @Success 200 {object} PlayerResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /public/players/{id} [get]
func (h *TeamHandler) GetPlayer(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	playerIDStr := chi.URLParam(r, "id")
	playerID, err := uuid.Parse(playerIDStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid player ID")
		return
	}

	p, err := h.client.Player.Query().
		Where(player.ID(playerID)).
		WithTeam().
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			respondError(w, http.StatusNotFound, "Player not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "Failed to get player")
		return
	}

	respondJSON(w, http.StatusOK, toPlayerResponse(p))
}

// ListPlayers godoc
// @Summary List all players
// @Description List all players with pagination and search
// @Tags players
// @Produce json
// @Param search query string false "Search by player name"
// @Param teamId query string false "Filter by team ID" format(uuid)
// @Param limit query int false "Limit results" default(50)
// @Param offset query int false "Offset for pagination" default(0)
// @Success 200 {array} PlayerResponse
// @Failure 500 {object} ErrorResponse
// @Router /public/players [get]
func (h *TeamHandler) ListPlayers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	query := h.client.Player.Query().
		Where(player.DeletedAtIsNil()).
		WithTeam()

	if search := r.URL.Query().Get("search"); search != "" {
		query = query.Where(player.NameContainsFold(search))
	}

	if teamIDStr := r.URL.Query().Get("teamId"); teamIDStr != "" {
		if teamID, err := uuid.Parse(teamIDStr); err == nil {
			query = query.Where(player.HasTeamWith(team.ID(teamID)))
		} else {
			respondError(w, http.StatusBadRequest, "Invalid team ID")
			return
		}
	}

	if eventIDStr := r.URL.Query().Get("eventId"); eventIDStr != "" {
		if eventID, err := uuid.Parse(eventIDStr); err == nil {
			query = query.Where(player.HasTeamWith(team.HasDivisionPoolWith(divisionpool.HasEventWith(event.ID(eventID)))))
		} else {
			respondError(w, http.StatusBadRequest, "Invalid event ID")
			return
		}
	}

	if gender := r.URL.Query().Get("gender"); gender != "" {
		// Normalize gender
		g := strings.ToUpper(gender)
		if strings.HasPrefix(g, "M") {
			gender = "M"
		} else if strings.HasPrefix(g, "F") || strings.HasPrefix(g, "W") {
			gender = "F"
		} else {
			gender = "X"
		}
		query = query.Where(player.GenderEQ(gender))
	}

	pagination := ParsePagination(r)

	players, err := query.
		Limit(pagination.Limit).
		Offset(pagination.Offset).
		Order(ent.Asc(player.FieldName)).
		All(ctx)

	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to list players")
		return
	}

	response := make([]PlayerResponse, len(players))
	for i, p := range players {
		response[i] = toPlayerResponse(p)
	}

	respondJSON(w, http.StatusOK, response)
}

// BulkImportPlayersResponse represents the result of a bulk import
type BulkImportPlayersResponse struct {
	Count  int      `json:"count"`
	Errors []string `json:"errors,omitempty"`
}

// BulkImportPlayers godoc
// @Summary Bulk import players for a team
// @Description Import players from a CSV file (columns: Name, Gender, JerseyNumber)
// @Tags teams
// @Accept multipart/form-data
// @Produce json
// @Param id path string true "Team ID" format(uuid)
// @Param file formData file true "CSV file"
// @Success 200 {object} BulkImportPlayersResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /teams/{id}/players/upload [post]
func (h *TeamHandler) BulkImportPlayers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	teamIDStr := chi.URLParam(r, "id")
	teamID, err := uuid.Parse(teamIDStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid team ID")
		return
	}

	// Parse multipart form (10MB max)
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		respondError(w, http.StatusBadRequest, "Failed to parse multipart form")
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		respondError(w, http.StatusBadRequest, "Missing file in request")
		return
	}
	defer file.Close()

	reader := csv.NewReader(file)
	// Allow variable number of columns
	reader.FieldsPerRecord = -1

	// Skip header
	header, err := reader.Read()
	if err != nil {
		respondError(w, http.StatusBadRequest, "Failed to read CSV header")
		return
	}

	// Simple validation: check if it's a valid CSV
	if len(header) < 1 {
		respondError(w, http.StatusBadRequest, "Invalid CSV format")
		return
	}

	count := 0
	var importErrors []string

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			importErrors = append(importErrors, fmt.Sprintf("Error reading row: %v", err))
			continue
		}

		if len(record) < 1 {
			continue
		}

		name := strings.TrimSpace(record[0])
		if name == "" {
			continue // Skip empty names
		}

		// Skip header if it was repeated or similarly named
		if strings.EqualFold(name, "Name") || strings.EqualFold(name, "Player Name") {
			continue
		}

		gender := "X" // Default
		if len(record) > 1 {
			g := strings.ToUpper(strings.TrimSpace(record[1]))
			if g == "M" || g == "F" || g == "X" {
				gender = g
			}
		}

		builder := h.client.Player.Create().
			SetName(name).
			SetGender(gender).
			SetTeamID(teamID)

		if len(record) > 2 {
			jerseyStr := strings.TrimSpace(record[2])
			if jerseyStr != "" {
				if j, err := strconv.Atoi(jerseyStr); err == nil {
					builder.SetJerseyNumber(j)
				}
			}
		}

		_, err = builder.Save(ctx)
		if err != nil {
			importErrors = append(importErrors, fmt.Sprintf("Failed to save player %s: %v", name, err))
		} else {
			count++
		}
	}

	respondJSON(w, http.StatusOK, BulkImportPlayersResponse{
		Count:  count,
		Errors: importErrors,
	})
}
