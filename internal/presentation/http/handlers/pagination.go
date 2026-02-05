package handlers

import (
	"net/http"
	"strconv"
)

const (
	DefaultPageSize = 50
	MaxPageSize     = 100
)

// PaginationParams holds pagination parameters
type PaginationParams struct {
	Limit  int
	Offset int
}

// ParsePagination extracts pagination parameters from request
func ParsePagination(r *http.Request) PaginationParams {
	params := PaginationParams{
		Limit:  DefaultPageSize,
		Offset: 0,
	}

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 {
			if limit > MaxPageSize {
				limit = MaxPageSize
			}
			params.Limit = limit
		}
	}

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil && offset >= 0 {
			params.Offset = offset
		}
	}

	return params
}

// PaginatedResponse wraps data with pagination metadata
type PaginatedResponse struct {
	Data    interface{} `json:"data"`
	Total   int         `json:"total"`
	Limit   int         `json:"limit"`
	Offset  int         `json:"offset"`
	HasMore bool        `json:"hasMore"`
}

// NewPaginatedResponse creates a paginated response
func NewPaginatedResponse(data interface{}, total, limit, offset int) PaginatedResponse {
	return PaginatedResponse{
		Data:    data,
		Total:   total,
		Limit:   limit,
		Offset:  offset,
		HasMore: offset+limit < total,
	}
}
