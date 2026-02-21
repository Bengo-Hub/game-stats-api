package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/bengobox/game-stats-api/ent"
	"github.com/bengobox/game-stats-api/internal/domain/user"
	"github.com/bengobox/game-stats-api/internal/pkg/auth"
	"github.com/google/uuid"
)

// SettingsHandler handles user settings and profile management
type SettingsHandler struct {
	userRepo user.Repository
}

// NewSettingsHandler creates a new settings handler
func NewSettingsHandler(userRepo user.Repository) *SettingsHandler {
	return &SettingsHandler{
		userRepo: userRepo,
	}
}

// UpdateProfileRequestDTO is the DTO for profile updates
type UpdateProfileRequestDTO struct {
	Name      *string `json:"name,omitempty"`
	AvatarURL *string `json:"avatarUrl,omitempty"`
}

// ChangePasswordRequestDTO is the DTO for password changes
type ChangePasswordRequestDTO struct {
	CurrentPassword string `json:"currentPassword"`
	NewPassword     string `json:"newPassword"`
}

// ProfileResponseDTO is the response DTO for profile data
type ProfileResponseDTO struct {
	ID        string  `json:"id"`
	Email     string  `json:"email"`
	Name      string  `json:"name"`
	Role      string  `json:"role"`
	AvatarURL *string `json:"avatarUrl,omitempty"`
}

// GetProfile godoc
// @Summary Get current user profile
// @Description Get the authenticated user's profile
// @Tags settings
// @Produce json
// @Success 200 {object} ProfileResponseDTO
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /settings/profile [get]
// @Security BearerAuth
func (h *SettingsHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromContext(r)
	if userID == uuid.Nil {
		respondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	u, err := h.userRepo.GetByID(r.Context(), userID)
	if err != nil {
		if ent.IsNotFound(err) {
			respondError(w, http.StatusNotFound, "User not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "Failed to fetch profile")
		return
	}

	respondJSON(w, http.StatusOK, ProfileResponseDTO{
		ID:        u.ID.String(),
		Email:     u.Email,
		Name:      u.Name,
		Role:      u.Role,
		AvatarURL: u.AvatarURL,
	})
}

// UpdateProfile godoc
// @Summary Update current user profile
// @Description Update the authenticated user's name or avatar
// @Tags settings
// @Accept json
// @Produce json
// @Param body body UpdateProfileRequestDTO true "Profile update request"
// @Success 200 {object} ProfileResponseDTO
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /settings/profile [put]
// @Security BearerAuth
func (h *SettingsHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromContext(r)
	if userID == uuid.Nil {
		respondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var dto UpdateProfileRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	u, err := h.userRepo.GetByID(r.Context(), userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to fetch user")
		return
	}

	if dto.Name != nil && *dto.Name != "" {
		u.Name = *dto.Name
	}
	if dto.AvatarURL != nil {
		u.AvatarURL = dto.AvatarURL
	}

	updated, err := h.userRepo.Update(r.Context(), u)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to update profile: "+err.Error())
		return
	}

	respondJSON(w, http.StatusOK, ProfileResponseDTO{
		ID:        updated.ID.String(),
		Email:     updated.Email,
		Name:      updated.Name,
		Role:      updated.Role,
		AvatarURL: updated.AvatarURL,
	})
}

// ChangePassword godoc
// @Summary Change password
// @Description Change the authenticated user's password
// @Tags settings
// @Accept json
// @Produce json
// @Param body body ChangePasswordRequestDTO true "Password change request"
// @Success 200 {object} map[string]string
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /settings/password [put]
// @Security BearerAuth
func (h *SettingsHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromContext(r)
	if userID == uuid.Nil {
		respondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var dto ChangePasswordRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if dto.CurrentPassword == "" || dto.NewPassword == "" {
		respondError(w, http.StatusBadRequest, "Current and new passwords are required")
		return
	}

	if len(dto.NewPassword) < 8 {
		respondError(w, http.StatusBadRequest, "New password must be at least 8 characters")
		return
	}

	u, err := h.userRepo.GetByID(r.Context(), userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to fetch user")
		return
	}

	// Verify current password
	if !auth.CheckPasswordHash(dto.CurrentPassword, u.PasswordHash) {
		respondError(w, http.StatusUnauthorized, "Current password is incorrect")
		return
	}

	// Hash new password
	hash, err := auth.HashPassword(dto.NewPassword)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to hash password")
		return
	}

	u.PasswordHash = hash
	if _, err := h.userRepo.Update(r.Context(), u); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to update password")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "Password updated successfully"})
}

// DeleteAccount godoc
// @Summary Delete own account
// @Description Permanently delete the authenticated user's account
// @Tags settings
// @Produce json
// @Success 204
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /settings/account [delete]
// @Security BearerAuth
func (h *SettingsHandler) DeleteAccount(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromContext(r)
	if userID == uuid.Nil {
		respondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	if err := h.userRepo.Delete(r.Context(), userID); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to delete account")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
