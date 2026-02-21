package analytics

import (
	"context"
	"github.com/google/uuid"
)

// AnalyticsClientInterface defines the contract for analytics providers (Metabase, etc.)
type AnalyticsClientInterface interface {
	Login(ctx context.Context) (*LoginResponse, error)
	GenerateGuestToken(ctx context.Context, accessToken string, req GuestTokenRequest) (string, error)
	GetDashboards(ctx context.Context, accessToken string) ([]Dashboard, error)
	GetDashboard(ctx context.Context, accessToken string, dashboardUUID uuid.UUID) (*Dashboard, error)
	HealthCheck(ctx context.Context) error
}

// Note: import context where needed in other files; kept here for types.

// LoginResponse from analytics provider authentication
type LoginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// Dashboard represents a dashboard
type Dashboard struct {
	ID            int       `json:"id"`
	DashboardUUID uuid.UUID `json:"dashboard_uuid"`
	Title         string    `json:"dashboard_title"`
	Slug          string    `json:"slug"`
	Published     bool      `json:"published"`
	URL           string    `json:"url"`
}

// GuestTokenRequest for generating embed tokens
type GuestTokenRequest struct {
	Resources []Resource `json:"resources"`
	RLS       []RLSRule  `json:"rls"`
	User      GuestUser  `json:"user"`
}

// Resource identifies an analytics resource
type Resource struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}

// RLSRule defines row-level security filter
type RLSRule struct {
	Clause string `json:"clause"`
}

// GuestUser information for embedding
type GuestUser struct {
	Username  string `json:"username"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

// GuestTokenResponse contains the generated token
type GuestTokenResponse struct {
	Token string `json:"token"`
}
