package handlers

import (
	"encoding/json"
	"net/http"
)

// ErrorResponse represents an error response for swagger documentation
// @Description Error response structure
type ErrorResponse struct {
	Error string `json:"error" example:"error message"`
}

// respondJSON writes a JSON response with the given status code and data
func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// respondWithJSON is an alias for respondJSON for consistency
func respondWithJSON(w http.ResponseWriter, status int, data interface{}) {
	respondJSON(w, status, data)
}

// respondError writes a JSON error response with the given status code and message
func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{"error": message})
}

// respondWithError writes a JSON error response with message and error
func respondWithError(w http.ResponseWriter, status int, message string, err error) {
	errorMsg := message
	if err != nil {
		errorMsg = message + ": " + err.Error()
	}
	respondJSON(w, status, map[string]string{"error": errorMsg})
}

// parseJSONBody parses the JSON request body into the provided destination struct
func parseJSONBody(r *http.Request, dst interface{}) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(dst)
}
