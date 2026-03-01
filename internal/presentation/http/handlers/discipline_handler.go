package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/bengobox/game-stats-api/ent"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// ======== request/response DTOs ========

type CreateDisciplineRequest struct {
	Name        string  `json:"name" validate:"required"`
	Slug        string  `json:"slug" validate:"required"`
	Description *string `json:"description,omitempty"`
	RulesPdfURL *string `json:"rulesPdfUrl,omitempty"`
	CountryID   string  `json:"countryId" validate:"required,uuid4"`
}

// Update fields are all pointers so that we can distinguish absent vs zero values.

type UpdateDisciplineRequest struct {
	Name        *string `json:"name,omitempty"`
	Slug        *string `json:"slug,omitempty"`
	Description *string `json:"description,omitempty"`
	RulesPdfURL *string `json:"rulesPdfUrl,omitempty"`
	CountryID   *string `json:"countryId,omitempty"`
}

// DisciplineResponse is returned to clients.

type DisciplineResponse struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Slug        string  `json:"slug"`
	Description *string `json:"description,omitempty"`
	RulesPdfURL *string `json:"rulesPdfUrl,omitempty"`
	CountryID   string  `json:"countryId"`
}

func toDisciplineResponse(d *ent.Discipline) DisciplineResponse {
	resp := DisciplineResponse{
		ID:   d.ID.String(),
		Name: d.Name,
		Slug: d.Slug,
	}
	if d.Edges.Country != nil {
		resp.CountryID = d.Edges.Country.ID.String()
	}
	if d.Description != nil {
		resp.Description = d.Description
	}
	if d.RulesPdfURL != nil {
		resp.RulesPdfURL = d.RulesPdfURL
	}
	return resp
}

// DisciplineService defines the subset of application logic that the HTTP
// handler depends upon. Keeping an interface simplifies testing.

type DisciplineService interface {
	List(ctx context.Context) ([]*ent.Discipline, error)
	GetByID(ctx context.Context, id uuid.UUID) (*ent.Discipline, error)
	Create(ctx context.Context, d *ent.Discipline) (*ent.Discipline, error)
	Update(ctx context.Context, d *ent.Discipline) (*ent.Discipline, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

// DisciplineHandler handles HTTP requests dealing with disciplines.

type DisciplineHandler struct {
	service DisciplineService
}

func NewDisciplineHandler(service DisciplineService) *DisciplineHandler {
	return &DisciplineHandler{service: service}
}

// ListDisciplines returns all disciplines.
// @Summary List disciplines
// @Tags disciplines
// @Produce json
// @Success 200 {array} handlers.DisciplineResponse
// @Router /disciplines [get]
func (h *DisciplineHandler) ListDisciplines(w http.ResponseWriter, r *http.Request) {
	pg := ParsePagination(r)
	list, err := h.service.List(r.Context())
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	total := len(list)
	start := pg.Offset
	if start > total {
		start = total
	}
	end := start + pg.Limit
	if end > total {
		end = total
	}
	paginatedList := list[start:end]

	resp := make([]DisciplineResponse, len(paginatedList))
	for i, d := range paginatedList {
		resp[i] = toDisciplineResponse(d)
	}

	respondJSON(w, http.StatusOK, NewPaginatedResponse(resp, total, pg.Limit, pg.Offset))
}

// GetDiscipline returns a single discipline by ID.
// @Summary Get discipline
// @Tags disciplines
// @Produce json
// @Param id path string true "Discipline ID"
// @Success 200 {object} handlers.DisciplineResponse
// @Failure 404 {string} string "not found"
// @Router /disciplines/{id} [get]
func (h *DisciplineHandler) GetDiscipline(w http.ResponseWriter, r *http.Request) {
	idParam := chi.URLParam(r, "id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	d, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(toDisciplineResponse(d))
}

// CreateDiscipline creates a new discipline.
// @Summary Create discipline
// @Tags disciplines
// @Accept json
// @Produce json
// @Param discipline body CreateDisciplineRequest true "Discipline"
// @Success 201 {object} handlers.DisciplineResponse
// @Router /disciplines [post]
func (h *DisciplineHandler) CreateDiscipline(w http.ResponseWriter, r *http.Request) {
	var req CreateDisciplineRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	countryID, err := uuid.Parse(req.CountryID)
	if err != nil {
		http.Error(w, "invalid country ID", http.StatusBadRequest)
		return
	}
	entObj := &ent.Discipline{
		Name:        req.Name,
		Slug:        req.Slug,
		Description: req.Description,
		RulesPdfURL: req.RulesPdfURL,
		Edges:       ent.DisciplineEdges{Country: &ent.Country{ID: countryID}},
	}
	created, err := h.service.Create(r.Context(), entObj)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(toDisciplineResponse(created))
}

// UpdateDiscipline updates an existing discipline.
// @Summary Update discipline
// @Tags disciplines
// @Accept json
// @Produce json
// @Param id path string true "Discipline ID"
// @Param discipline body UpdateDisciplineRequest true "Discipline"
// @Success 200 {object} handlers.DisciplineResponse
// @Router /disciplines/{id} [put]
func (h *DisciplineHandler) UpdateDiscipline(w http.ResponseWriter, r *http.Request) {
	idParam := chi.URLParam(r, "id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	var req UpdateDisciplineRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	// fetch existing to retain existing fields when not provided
	existing, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if req.Name != nil {
		existing.Name = *req.Name
	}
	if req.Slug != nil {
		existing.Slug = *req.Slug
	}
	if req.Description != nil {
		existing.Description = req.Description
	}
	if req.RulesPdfURL != nil {
		existing.RulesPdfURL = req.RulesPdfURL
	}
	if req.CountryID != nil {
		cid, err := uuid.Parse(*req.CountryID)
		if err != nil {
			http.Error(w, "invalid country ID", http.StatusBadRequest)
			return
		}
		existing.Edges.Country = &ent.Country{ID: cid}
	}
	updated, err := h.service.Update(r.Context(), existing)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(toDisciplineResponse(updated))
}

// DeleteDiscipline soft-deletes a discipline.
// @Summary Delete discipline
// @Tags disciplines
// @Param id path string true "Discipline ID"
// @Success 204 {string} string ""
// @Router /disciplines/{id} [delete]
func (h *DisciplineHandler) DeleteDiscipline(w http.ResponseWriter, r *http.Request) {
	idParam := chi.URLParam(r, "id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	if err := h.service.Delete(r.Context(), id); err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
