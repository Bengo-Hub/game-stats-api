package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/bengobox/game-stats-api/ent"
	"github.com/bengobox/game-stats-api/internal/domain/user"
	"github.com/bengobox/game-stats-api/internal/pkg/auth"
	"github.com/bengobox/game-stats-api/internal/presentation/http/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// AdminUsersHandler handles user management for admins
type AdminUsersHandler struct {
	userRepo user.Repository
	client   *ent.Client
}

// NewAdminUsersHandler creates a new admin users handler
func NewAdminUsersHandler(userRepo user.Repository, client *ent.Client) *AdminUsersHandler {
	return &AdminUsersHandler{
		userRepo: userRepo,
		client:   client,
	}
}

// AdminUserDTO is the response DTO for admin user management
type AdminUserDTO struct {
	ID          string  `json:"id"`
	Email       string  `json:"email"`
	Name        string  `json:"name"`
	Role        string  `json:"role"`
	IsActive    bool    `json:"isActive"`
	AvatarURL   *string `json:"avatarUrl,omitempty"`
	LastLoginAt *string `json:"lastLoginAt,omitempty"`
	CreatedAt   string  `json:"createdAt"`
	UpdatedAt   string  `json:"updatedAt"`
}

// CreateUserRequestDTO is the DTO for creating a new user
type CreateUserRequestDTO struct {
	Email    string `json:"email"`
	Name     string `json:"name"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

// UpdateUserRequestDTO is the DTO for updating a user
type UpdateUserRequestDTO struct {
	Name     *string `json:"name,omitempty"`
	Role     *string `json:"role,omitempty"`
	IsActive *bool   `json:"isActive,omitempty"`
}

func mapEntUserToAdminDTO(u *ent.User) AdminUserDTO {
	dto := AdminUserDTO{
		ID:        u.ID.String(),
		Email:     u.Email,
		Name:      u.Name,
		Role:      u.Role,
		IsActive:  u.IsActive,
		AvatarURL: u.AvatarURL,
		CreatedAt: u.CreatedAt.Format(time.RFC3339),
		UpdatedAt: u.UpdatedAt.Format(time.RFC3339),
	}
	if u.LastLoginAt != nil {
		t := u.LastLoginAt.Format(time.RFC3339)
		dto.LastLoginAt = &t
	}
	return dto
}

// ListUsers godoc
// @Summary List all users (Admin only)
// @Description List all users in the system with their roles and status
// @Tags admin
// @Produce json
// @Success 200 {array} AdminUserDTO
// @Failure 500 {object} ErrorResponse
// @Router /admin/users [get]
// @Security BearerAuth
func (h *AdminUsersHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.userRepo.List(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to fetch users")
		return
	}

	result := make([]AdminUserDTO, len(users))
	for i, u := range users {
		result[i] = mapEntUserToAdminDTO(u)
	}

	respondJSON(w, http.StatusOK, result)
}

// GetUser godoc
// @Summary Get user details (Admin only)
// @Description Get a specific user's details by ID
// @Tags admin
// @Produce json
// @Param id path string true "User ID" format(uuid)
// @Success 200 {object} AdminUserDTO
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /admin/users/{id} [get]
// @Security BearerAuth
func (h *AdminUsersHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	userID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	u, err := h.userRepo.GetByID(r.Context(), userID)
	if err != nil {
		if ent.IsNotFound(err) {
			respondError(w, http.StatusNotFound, "User not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "Failed to fetch user")
		return
	}

	respondJSON(w, http.StatusOK, mapEntUserToAdminDTO(u))
}

// CreateUser godoc
// @Summary Create a new user (Admin only)
// @Description Create a new user with specified role
// @Tags admin
// @Accept json
// @Produce json
// @Param body body CreateUserRequestDTO true "User creation request"
// @Success 201 {object} AdminUserDTO
// @Failure 400 {object} ErrorResponse
// @Failure 409 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /admin/users [post]
// @Security BearerAuth
func (h *AdminUsersHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var dto CreateUserRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if dto.Email == "" || dto.Name == "" || dto.Password == "" || dto.Role == "" {
		respondError(w, http.StatusBadRequest, "Email, name, password, and role are required")
		return
	}

	// Validate role
	validRoles := map[string]bool{"admin": true, "scorekeeper": true, "viewer": true, "event_manager": true}
	if !validRoles[dto.Role] {
		respondError(w, http.StatusBadRequest, "Invalid role. Must be: admin, scorekeeper, viewer, or event_manager")
		return
	}

	// Check if email already exists
	if _, err := h.userRepo.GetByEmail(r.Context(), dto.Email); err == nil {
		respondError(w, http.StatusConflict, "A user with this email already exists")
		return
	}

	// Hash password
	hash, err := auth.HashPassword(dto.Password)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to hash password")
		return
	}

	newUser := &ent.User{
		Email:        dto.Email,
		Name:         dto.Name,
		PasswordHash: hash,
		Role:         dto.Role,
		IsActive:     true,
	}

	created, err := h.userRepo.Create(r.Context(), newUser)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to create user: "+err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, mapEntUserToAdminDTO(created))
}

// UpdateUser godoc
// @Summary Update a user (Admin only)
// @Description Update a user's name, role, or active status
// @Tags admin
// @Accept json
// @Produce json
// @Param id path string true "User ID" format(uuid)
// @Param body body UpdateUserRequestDTO true "User update request"
// @Success 200 {object} AdminUserDTO
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /admin/users/{id} [put]
// @Security BearerAuth
func (h *AdminUsersHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	userID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	var dto UpdateUserRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Fetch existing user
	u, err := h.userRepo.GetByID(r.Context(), userID)
	if err != nil {
		if ent.IsNotFound(err) {
			respondError(w, http.StatusNotFound, "User not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "Failed to fetch user")
		return
	}

	// Apply updates
	if dto.Name != nil {
		u.Name = *dto.Name
	}
	if dto.Role != nil {
		validRoles := map[string]bool{"admin": true, "scorekeeper": true, "viewer": true, "event_manager": true}
		if !validRoles[*dto.Role] {
			respondError(w, http.StatusBadRequest, "Invalid role")
			return
		}
		u.Role = *dto.Role
	}
	if dto.IsActive != nil {
		u.IsActive = *dto.IsActive
	}

	updated, err := h.userRepo.Update(r.Context(), u)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to update user: "+err.Error())
		return
	}

	respondJSON(w, http.StatusOK, mapEntUserToAdminDTO(updated))
}

// DeleteUser godoc
// @Summary Delete a user (Admin only)
// @Description Soft deletes a user account
// @Tags admin
// @Param id path string true "User ID" format(uuid)
// @Success 204
// @Failure 400 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /admin/users/{id} [delete]
// @Security BearerAuth
func (h *AdminUsersHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	userID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	// Prevent admin from deleting themselves
	claims, _ := r.Context().Value(middleware.UserContextKey).(*auth.Claims)
	if claims != nil && claims.UserID == userID {
		respondError(w, http.StatusForbidden, "Cannot delete your own account")
		return
	}

	if err := h.userRepo.Delete(r.Context(), userID); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to delete user: "+err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetAuditLogs godoc
// @Summary Get audit logs (Admin only)
// @Description Get all audit log entries, newest first
// @Tags admin
// @Produce json
// @Param limit query int false "Limit results" default(50)
// @Param offset query int false "Offset for pagination" default(0)
// @Success 200 {array} AuditLogDTO
// @Failure 500 {object} ErrorResponse
// @Router /admin/audit-logs [get]
// @Security BearerAuth
func (h *AdminUsersHandler) GetAuditLogs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	pagination := ParsePagination(r)

	logs, err := h.client.AuditLog.Query().
		Order(ent.Desc("created_at")).
		Limit(pagination.Limit).
		Offset(pagination.Offset).
		All(ctx)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to fetch audit logs")
		return
	}

	result := make([]AuditLogDTO, len(logs))
	for i, log := range logs {
		result[i] = AuditLogDTO{
			ID:         log.ID.String(),
			EntityType: log.EntityType,
			EntityID:   log.EntityID.String(),
			Action:     log.Action,
			UserID:     log.UserID.String(),
			Username:   log.Username,
			Changes:    log.Changes,
			Reason:     log.Reason,
			IPAddress:  log.IPAddress,
			UserAgent:  log.UserAgent,
			CreatedAt:  log.CreatedAt.Format(time.RFC3339),
		}
	}

	respondJSON(w, http.StatusOK, result)
}

// GetSystemHealth godoc
// @Summary Get system health (Admin only)
// @Description Get system health information including counts and service status
// @Tags admin
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} ErrorResponse
// @Router /admin/system/health [get]
// @Security BearerAuth
func (h *AdminUsersHandler) GetSystemHealth(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userCount, _ := h.client.User.Query().Count(ctx)
	gameCount, _ := h.client.Game.Query().Count(ctx)
	eventCount, _ := h.client.Event.Query().Count(ctx)
	teamCount, _ := h.client.Team.Query().Count(ctx)

	health := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().Format(time.RFC3339),
		"database": map[string]interface{}{
			"status": "connected",
		},
		"counts": map[string]int{
			"users":  userCount,
			"games":  gameCount,
			"events": eventCount,
			"teams":  teamCount,
		},
	}

	respondJSON(w, http.StatusOK, health)
}
