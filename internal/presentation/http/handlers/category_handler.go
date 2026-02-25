package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/bengobox/game-stats-api/ent"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// ======== DTOs ========

type CreateCategoryRequest struct {
	Name        string  `json:"name" validate:"required"`
	Slug        string  `json:"slug" validate:"required"`
	Description *string `json:"description,omitempty"`
}

type UpdateCategoryRequest struct {
	Name        *string `json:"name,omitempty"`
	Slug        *string `json:"slug,omitempty"`
	Description *string `json:"description,omitempty"`
}

type CategoryResponse struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Slug        string  `json:"slug"`
	Description *string `json:"description,omitempty"`
}

func toCategoryResponse(c *ent.Category) CategoryResponse {
	resp := CategoryResponse{
		ID:   c.ID.String(),
		Name: c.Name,
		Slug: c.Slug,
	}
	if c.Description != nil {
		resp.Description = c.Description
	}
	return resp
}

// Service interface required by handler (mirrors application layer but allows stubbing)

type CategoryService interface {
	List(ctx context.Context) ([]*ent.Category, error)
	GetByID(ctx context.Context, id uuid.UUID) (*ent.Category, error)
	Create(ctx context.Context, c *ent.Category) (*ent.Category, error)
	Update(ctx context.Context, c *ent.Category) (*ent.Category, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

// Handler definition

type CategoryHandler struct {
	service CategoryService
}

func NewCategoryHandler(service CategoryService) *CategoryHandler {
	return &CategoryHandler{service: service}
}

// ListCategories returns all categories.
// @Summary List categories
// @Tags categories
// @Produce json
// @Success 200 {array} handlers.CategoryResponse
// @Router /categories [get]
func (h *CategoryHandler) ListCategories(w http.ResponseWriter, r *http.Request) {
	list, err := h.service.List(r.Context())
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	resp := make([]CategoryResponse, len(list))
	for i, c := range list {
		resp[i] = toCategoryResponse(c)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// GetCategory by id
// @Summary Get category
// @Tags categories
// @Produce json
// @Param id path string true "Category ID"
// @Success 200 {object} handlers.CategoryResponse
// @Failure 404 {string} string "not found"
// @Router /categories/{id} [get]
func (h *CategoryHandler) GetCategory(w http.ResponseWriter, r *http.Request) {
	idParam := chi.URLParam(r, "id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	c, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(toCategoryResponse(c))
}

// CreateCategory
// @Summary Create category
// @Tags categories
// @Accept json
// @Produce json
// @Param category body CreateCategoryRequest true "Category"
// @Success 201 {object} handlers.CategoryResponse
// @Router /categories [post]
func (h *CategoryHandler) CreateCategory(w http.ResponseWriter, r *http.Request) {
	var req CreateCategoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	entObj := &ent.Category{
		Name:        req.Name,
		Slug:        req.Slug,
		Description: req.Description,
	}
	created, err := h.service.Create(r.Context(), entObj)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(toCategoryResponse(created))
}

// UpdateCategory
// @Summary Update category
// @Tags categories
// @Accept json
// @Produce json
// @Param id path string true "Category ID"
// @Param category body UpdateCategoryRequest true "Category"
// @Success 200 {object} handlers.CategoryResponse
// @Router /categories/{id} [put]
func (h *CategoryHandler) UpdateCategory(w http.ResponseWriter, r *http.Request) {
	idParam := chi.URLParam(r, "id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	var req UpdateCategoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
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
	updated, err := h.service.Update(r.Context(), existing)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(toCategoryResponse(updated))
}

// DeleteCategory
// @Summary Delete category
// @Tags categories
// @Param id path string true "Category ID"
// @Success 204
// @Router /categories/{id} [delete]
func (h *CategoryHandler) DeleteCategory(w http.ResponseWriter, r *http.Request) {
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
