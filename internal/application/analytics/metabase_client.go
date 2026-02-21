package analytics

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"io"
	"net/http"
	"time"
)

// MetabaseClient implements the SupersetClientInterface but targets Metabase APIs.
// This acts as an adapter so the analytics service can continue using the same
// interface while switching to Metabase.
type MetabaseClient struct {
	BaseURL    string
	Username   string
	Password   string
	httpClient *http.Client
}

// NewMetabaseClient creates a new Metabase API client
func NewMetabaseClient(baseURL, username, password string) *MetabaseClient {
	return &MetabaseClient{
		BaseURL:  baseURL,
		Username: username,
		Password: password,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Login authenticates against Metabase and returns a token-like response
// Note: Metabase returns a session token which we expose in AccessToken.
func (c *MetabaseClient) Login(ctx context.Context) (*LoginResponse, error) {
	payload := map[string]string{
		"username": c.Username,
		"password": c.Password,
	}

	body, err := c.post(ctx, "/api/session", payload, "")
	if err != nil {
		return nil, fmt.Errorf("metabase login failed: %w", err)
	}

	// Try to parse token or id if present
	var resp map[string]interface{}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse metabase login response: %w", err)
	}

	token := ""
	if t, ok := resp["id"]; ok {
		token = fmt.Sprintf("%v", t)
	}
	if t, ok := resp["token"]; ok && token == "" {
		token = fmt.Sprintf("%v", t)
	}

	return &LoginResponse{AccessToken: token}, nil
}

// GenerateGuestToken is not directly supported by Metabase in the same way as Superset.
// Implementing secure signed embedding requires using Metabase's signed JWT approach
// with an embedding secret. For now, return a not implemented error so callers
// get a clear message.
func (c *MetabaseClient) GenerateGuestToken(ctx context.Context, accessToken string, req GuestTokenRequest) (string, error) {
	return "", fmt.Errorf("metabase guest token generation not implemented: use signed embedding with METABASE_EMBED_SECRET")
}

// GetDashboards lists dashboards from Metabase and maps them to the Dashboard type
func (c *MetabaseClient) GetDashboards(ctx context.Context, accessToken string) ([]Dashboard, error) {
	body, err := c.get(ctx, "/api/dashboard", accessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to get dashboards from metabase: %w", err)
	}

	var raw []map[string]interface{}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse metabase dashboards response: %w", err)
	}

	var dashboards []Dashboard
	for _, item := range raw {
		id := 0
		if v, ok := item["id"]; ok {
			switch vv := v.(type) {
			case float64:
				id = int(vv)
			case int:
				id = vv
			}
		}

		title := ""
		if v, ok := item["name"]; ok {
			title = fmt.Sprintf("%v", v)
		}

		slug := ""
		if v, ok := item["slug"]; ok {
			slug = fmt.Sprintf("%v", v)
		}

		dashboards = append(dashboards, Dashboard{
			ID:    id,
			Title: title,
			Slug:  slug,
			URL:   fmt.Sprintf("%s/dashboard/%d", c.BaseURL, id),
		})
	}

	return dashboards, nil
}

// GetDashboard retrieves a dashboard by numeric id (Metabase uses ints)
func (c *MetabaseClient) GetDashboard(ctx context.Context, accessToken string, dashboardUUID uuid.UUID) (*Dashboard, error) {
	// Metabase dashboards are integer IDs; attempt to use dashboardUUID as fallback
	endpoint := fmt.Sprintf("/api/dashboard/%s", dashboardUUID.String())
	body, err := c.get(ctx, endpoint, accessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to get metabase dashboard: %w", err)
	}

	var item map[string]interface{}
	if err := json.Unmarshal(body, &item); err != nil {
		return nil, fmt.Errorf("failed to parse metabase dashboard response: %w", err)
	}

	id := 0
	if v, ok := item["id"]; ok {
		switch vv := v.(type) {
		case float64:
			id = int(vv)
		case int:
			id = vv
		}
	}

	title := ""
	if v, ok := item["name"]; ok {
		title = fmt.Sprintf("%v", v)
	}

	slug := ""
	if v, ok := item["slug"]; ok {
		slug = fmt.Sprintf("%v", v)
	}

	d := &Dashboard{
		ID:    id,
		Title: title,
		Slug:  slug,
		URL:   fmt.Sprintf("%s/dashboard/%d", c.BaseURL, id),
	}

	return d, nil
}

func (c *MetabaseClient) get(ctx context.Context, endpoint, accessToken string) ([]byte, error) {
	url := c.BaseURL + endpoint

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	if accessToken != "" {
		req.Header.Set("X-Metabase-Session", accessToken)
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

func (c *MetabaseClient) post(ctx context.Context, endpoint string, payload interface{}, accessToken string) ([]byte, error) {
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
		req.Header.Set("X-Metabase-Session", accessToken)
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

// HealthCheck verifies Metabase connectivity
func (c *MetabaseClient) HealthCheck(ctx context.Context) error {
	url := c.BaseURL + "/api/health"

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
