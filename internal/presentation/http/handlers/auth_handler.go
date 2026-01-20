package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/bengobox/game-stats-api/internal/application/auth"
)

type AuthHandler struct {
	service *auth.Service
}

func NewAuthHandler(service *auth.Service) *AuthHandler {
	return &AuthHandler{service: service}
}

// Login handles user authentication.
// @Summary Login
// @Description Authenticate user and return access/refresh tokens.
// @Tags auth
// @Accept json
// @Produce json
// @Param request body auth.LoginRequest true "Login Request"
// @Success 200 {object} auth.LoginResponse
// @Failure 401 {string} string "unauthorized"
// @Router /auth/login [post]
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req auth.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	resp, err := h.service.Login(r.Context(), req)
	if err != nil {
		if err == auth.ErrInvalidCredentials {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// Refresh handles token refresh.
// @Summary Refresh Token
// @Description Refresh access token using refresh token.
// @Tags auth
// @Accept json
// @Produce json
// @Param request body auth.RefreshRequest true "Refresh Request"
// @Success 200 {object} auth.TokenResponse
// @Failure 401 {string} string "unauthorized"
// @Router /auth/refresh [post]
func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req auth.RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	resp, err := h.service.Refresh(r.Context(), req)
	if err != nil {
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
