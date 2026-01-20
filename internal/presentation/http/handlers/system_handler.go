package handlers

import (
	"net/http"
)

type SystemHandler struct{}

func NewSystemHandler() *SystemHandler {
	return &SystemHandler{}
}

// Health handles the health check request.
// @Summary Health Check
// @Description Get the health status of the API.
// @Tags system
// @Produce plain
// @Success 200 {string} string "OK"
// @Router /health [get]
func (h *SystemHandler) Health(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}
