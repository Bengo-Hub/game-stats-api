package handlers

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
)

type MediaHandler struct {
	uploadsDir string
	baseUrl    string
}

func NewMediaHandler(uploadsDir string, baseUrl string) *MediaHandler {
	// Ensure uploads directory exists
	if _, err := os.Stat(uploadsDir); os.IsNotExist(err) {
		os.MkdirAll(uploadsDir, 0755)
	}
	return &MediaHandler{
		uploadsDir: uploadsDir,
		baseUrl:    baseUrl,
	}
}

// UploadResponse represents the response after a successful upload
type UploadResponse struct {
	URL      string `json:"url"`
	Filename string `json:"filename"`
	Size     int64  `json:"size"`
}

// Upload handles file uploads
// @Summary Upload a file
// @Description Upload an image or file and get back a URL
// @Tags media
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "File to upload"
// @Success 201 {object} UploadResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /upload [post]
func (h *MediaHandler) Upload(w http.ResponseWriter, r *http.Request) {
	// Parse multipart form (10MB max)
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		respondError(w, http.StatusBadRequest, "Failed to parse multipart form")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		respondError(w, http.StatusBadRequest, "Missing file in request")
		return
	}
	defer file.Close()

	// Generate unique filename
	ext := filepath.Ext(header.Filename)
	newFilename := fmt.Sprintf("%s%s", uuid.New().String(), ext)

	// Create sub-directory based on date to keep it organized
	dateDir := time.Now().Format("2006/01/02")
	fullDir := filepath.Join(h.uploadsDir, dateDir)
	if _, err := os.Stat(fullDir); os.IsNotExist(err) {
		os.MkdirAll(fullDir, 0755)
	}

	filePath := filepath.Join(fullDir, newFilename)
	dst, err := os.Create(filePath)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to create destination file")
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to save file")
		return
	}

	// Construct public URL
	// The path should be relative to the uploads directory for the static server
	relativeUrlPath := fmt.Sprintf("/uploads/%s/%s", dateDir, newFilename)
	publicUrl := fmt.Sprintf("%s%s", h.baseUrl, relativeUrlPath)

	respondJSON(w, http.StatusCreated, UploadResponse{
		URL:      publicUrl,
		Filename: header.Filename,
		Size:     header.Size,
	})
}
