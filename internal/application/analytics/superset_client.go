package analytics

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// SupersetClientInterface defines the contract for Superset operations
type SupersetClientInterface interface {
	Login(ctx context.Context) (*LoginResponse, error)
	GenerateGuestToken(ctx context.Context, accessToken string, req GuestTokenRequest) (string, error)
	GetDashboards(ctx context.Context, accessToken string) ([]Dashboard, error)
	GetDashboard(ctx context.Context, accessToken string, dashboardUUID uuid.UUID) (*Dashboard, error)
	HealthCheck(ctx context.Context) error
}

// SupersetClient manages interactions with Apache Superset REST API
type SupersetClient struct {
	BaseURL    string
	Username   string
	Password   string
	httpClient *http.Client
}

// NewSupersetClient creates a new Superset API client
func NewSupersetClient(baseURL, username, password string) *SupersetClient {
	return &SupersetClient{
		BaseURL:  baseURL,
		Username: username,
		Password: password,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// LoginRequest for Superset authentication
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Provider string `json:"provider"`
	Refresh  bool   `json:"refresh"`
}

// LoginResponse from Superset
type LoginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// Dashboard represents a Superset dashboard
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

// Resource identifies a Superset resource (dashboard, chart)
type Resource struct {
	Type string `json:"type"` // "dashboard" or "chart"
	ID   string `json:"id"`   // UUID string
}

// RLSRule defines row-level security filter
type RLSRule struct {
	Clause string `json:"clause"` // SQL WHERE clause
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

// Login authenticates with Superset and returns access token
func (c *SupersetClient) Login(ctx context.Context) (*LoginResponse, error) {
	loginReq := LoginRequest{
		Username: c.Username,
		Password: c.Password,
		Provider: "db",
		Refresh:  true,
	}

	body, err := c.post(ctx, "/api/v1/security/login", loginReq, "")
	if err != nil {
		return nil, fmt.Errorf("login failed: %w", err)
	}

	var response LoginResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse login response: %w", err)
	}

	return &response, nil
}

// GenerateGuestToken creates a guest token for embedding dashboards
func (c *SupersetClient) GenerateGuestToken(ctx context.Context, accessToken string, req GuestTokenRequest) (string, error) {
	body, err := c.post(ctx, "/api/v1/security/guest_token", req, accessToken)
	if err != nil {
		return "", fmt.Errorf("failed to generate guest token: %w", err)
	}

	var response GuestTokenResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("failed to parse guest token response: %w", err)
	}

	return response.Token, nil
}

// GetDashboards retrieves list of available dashboards
func (c *SupersetClient) GetDashboards(ctx context.Context, accessToken string) ([]Dashboard, error) {
	body, err := c.get(ctx, "/api/v1/dashboard/", accessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to get dashboards: %w", err)
	}

	var response struct {
		Result []Dashboard `json:"result"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse dashboards response: %w", err)
	}

	return response.Result, nil
}

// GetDashboard retrieves a specific dashboard by UUID
func (c *SupersetClient) GetDashboard(ctx context.Context, accessToken string, dashboardUUID uuid.UUID) (*Dashboard, error) {
	endpoint := fmt.Sprintf("/api/v1/dashboard/%s", dashboardUUID.String())
	body, err := c.get(ctx, endpoint, accessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to get dashboard: %w", err)
	}

	var response struct {
		Result Dashboard `json:"result"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse dashboard response: %w", err)
	}

	return &response.Result, nil
}

// get performs HTTP GET request to Superset API
func (c *SupersetClient) get(ctx context.Context, endpoint, accessToken string) ([]byte, error) {
	url := c.BaseURL + endpoint

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	if accessToken != "" {
		req.Header.Set("Authorization", "Bearer "+accessToken)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	return body, nil
}

// post performs HTTP POST request to Superset API
func (c *SupersetClient) post(ctx context.Context, endpoint string, payload interface{}, accessToken string) ([]byte, error) {
	url := c.BaseURL + endpoint

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	if accessToken != "" {
		req.Header.Set("Authorization", "Bearer "+accessToken)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	return body, nil
}

// HealthCheck verifies Superset connectivity
func (c *SupersetClient) HealthCheck(ctx context.Context) error {
	url := c.BaseURL + "/health"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check returned status %d", resp.StatusCode)
	}

	return nil
}
