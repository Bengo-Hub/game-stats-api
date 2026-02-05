# Analytics Service Integration with Apache Superset

## Overview

This document outlines the integration between the Game Stats API and Apache Superset for embedded analytics, dashboards, and AI-powered queries.

## Superset Deployment Architecture

Based on `devops-k8s/apps/superset/app.yaml` and `devops-k8s/docs/superset-deployment.md`:

### Infrastructure Components

**Superset Web Server** (1-3 replicas with HPA)
- Serves web UI and REST API
- Handles authentication and authorization
- Processes synchronous queries
- **Port**: 8088
- **Ingress**: https://superset.codevertexitsolutions.com

**Celery Workers** (1-4 replicas with HPA)
- Execute asynchronous queries
- Handle long-running tasks
- Process scheduled reports

**Celery Beat** (1 replica)
- Manages scheduled tasks
- Triggers periodic cache warming

**External Dependencies**
- PostgreSQL: `postgresql.infra.svc.cluster.local:5432`
- Redis: `redis-master.infra.svc.cluster.local:6379`

### Database Configuration

```yaml
Database: superset
User: superset_user
Schema: public
Connection: postgresql://superset_user@postgresql.infra.svc.cluster.local:5432/superset
```

### Feature Flags

```python
FEATURE_FLAGS = {
    'ENABLE_TEMPLATE_PROCESSING': True,
    'DASHBOARD_NATIVE_FILTERS': True,
    'DASHBOARD_CROSS_FILTERS': True,
    'DASHBOARD_RBAC': True,
    'EMBEDDED_SUPERSET': True,  # Critical for embedding
}
```

## Integration Patterns

### 1. Database Connection

Game Stats API connects to the same PostgreSQL instance as Superset, enabling:
- Direct query access to game statistics
- Real-time dashboard updates
- No data synchronization needed

**Game Stats Database**: `game_stats`
**Superset Database**: `superset`

Both on: `postgresql.infra.svc.cluster.local:5432`

### 2. Embedded Dashboard Pattern

```python
# Superset Configuration (configOverrides.custom_config)
EMBEDDED_SUPERSET = True
SESSION_COOKIE_SECURE = True
SESSION_COOKIE_HTTPONLY = True
SESSION_COOKIE_SAMESITE = 'Lax'

# Enable guest token generation
FEATURE_FLAGS['EMBEDDED_SUPERSET'] = True
```

**Guest Token Flow**:
1. Game Stats API calls Superset REST API
2. Generates short-lived guest token with permissions
3. Returns token to frontend
4. Frontend embeds dashboard using `@superset-ui/embedded-sdk`

### 3. Row-Level Security (RLS)

```python
ROW_LEVEL_SECURITY_ENABLED = True
```

**Implementation**:
- Filter dashboards by event_id, team_id, or user permissions
- Use Jinja templates in Superset SQL queries
- Pass context via guest token claims

**Example RLS Filter**:
```sql
WHERE event_id = '{{ current_user_id() }}'
  OR team_id IN ({{ user_team_ids() }})
```

### 4. API Integration

**Superset REST API**: `https://superset.codevertexitsolutions.com/api/v1/`

**Key Endpoints**:
- `POST /security/login` - Authenticate
- `POST /security/guest_token` - Generate embed token
- `GET /dashboard/{id}` - Get dashboard metadata
- `POST /chart/data` - Execute chart query
- `GET /database/` - List databases

## Analytics Service Implementation

### Service Structure

```go
package analytics

import (
    "context"
    "time"
    "github.com/google/uuid"
)

// Service handles analytics and Superset integration
type Service struct {
    supersetClient *SupersetClient
    dbClient       *ent.Client
}

// SupersetClient manages Superset API interactions
type SupersetClient struct {
    BaseURL    string
    Username   string
    Password   string
    httpClient *http.Client
}

// Dashboard represents a Superset dashboard
type Dashboard struct {
    ID            int       `json:"id"`
    DashboardUUID uuid.UUID `json:"dashboard_uuid"`
    Title         string    `json:"dashboard_title"`
    Slug          string    `json:"slug"`
    Published     bool      `json:"published"`
}

// GuestTokenRequest for embedding dashboards
type GuestTokenRequest struct {
    Resources []Resource         `json:"resources"`
    RLS       []RLSRule          `json:"rls"`
    User      GuestUser          `json:"user"`
}

type Resource struct {
    Type string `json:"type"` // "dashboard"
    ID   string `json:"id"`   // dashboard UUID
}

type RLSRule struct {
    Clause string `json:"clause"` // SQL WHERE clause
}

type GuestUser struct {
    Username  string `json:"username"`
    FirstName string `json:"first_name"`
    LastName  string `json:"last_name"`
}

// GuestTokenResponse contains the embed token
type GuestTokenResponse struct {
    Token string `json:"token"`
}
```

### Authentication Flow

```go
func (c *SupersetClient) Login(ctx context.Context) (string, error) {
    payload := map[string]string{
        "username": c.Username,
        "password": c.Password,
        "provider": "db",
        "refresh":  "true",
    }
    
    resp, err := c.post(ctx, "/api/v1/security/login", payload)
    if err != nil {
        return "", err
    }
    
    var result struct {
        AccessToken string `json:"access_token"`
    }
    
    if err := json.Unmarshal(resp, &result); err != nil {
        return "", err
    }
    
    return result.AccessToken, nil
}
```

### Guest Token Generation

```go
func (s *Service) GenerateEmbedToken(ctx context.Context, req GenerateEmbedTokenRequest) (*GuestTokenResponse, error) {
    // Authenticate with Superset
    accessToken, err := s.supersetClient.Login(ctx)
    if err != nil {
        return nil, fmt.Errorf("failed to authenticate: %w", err)
    }
    
    // Build RLS rules based on user permissions
    rlsRules := s.buildRLSRules(req.UserID, req.EventID, req.TeamIDs)
    
    // Generate guest token
    guestReq := GuestTokenRequest{
        Resources: []Resource{
            {Type: "dashboard", ID: req.DashboardUUID.String()},
        },
        RLS: rlsRules,
        User: GuestUser{
            Username:  req.Username,
            FirstName: req.FirstName,
            LastName:  req.LastName,
        },
    }
    
    token, err := s.supersetClient.GenerateGuestToken(ctx, accessToken, guestReq)
    if err != nil {
        return nil, fmt.Errorf("failed to generate token: %w", err)
    }
    
    return &GuestTokenResponse{Token: token}, nil
}
```

### Dashboard Management

```go
func (s *Service) ListDashboards(ctx context.Context) ([]Dashboard, error) {
    accessToken, err := s.supersetClient.Login(ctx)
    if err != nil {
        return nil, err
    }
    
    dashboards, err := s.supersetClient.GetDashboards(ctx, accessToken)
    if err != nil {
        return nil, err
    }
    
    return dashboards, nil
}
```

## Predefined Dashboards

### 1. Event Overview Dashboard

**Metrics**:
- Total games played/scheduled
- Teams participating
- Average spirit scores
- Top scorers
- Games by status (scheduled/in-progress/completed)

**Filters**:
- Event selection
- Date range
- Division/Pool

### 2. Player Statistics Dashboard

**Metrics**:
- Goals scored
- Assists
- Callahans
- Spirit scores received
- Games played
- Average points per game

**Filters**:
- Player search
- Team selection
- Event/Season
- Position

### 3. Spirit Leaderboard Dashboard

**Metrics**:
- Team spirit averages
- Individual player spirit nominations
- Spirit score breakdown by category
- Trend over time
- MVP nominations

**Filters**:
- Event
- Division
- Date range

### 4. Team Performance Dashboard

**Metrics**:
- Win/Loss/Draw record
- Goals for/against
- Goal differential
- Win percentage
- Head-to-head records
- Division standings

**Filters**:
- Team selection
- Season/Event
- Division

## API Endpoints

### Analytics Controller

```go
// GET /api/v1/analytics/dashboards
// List all available dashboards
func (h *AnalyticsHandler) ListDashboards(w http.ResponseWriter, r *http.Request)

// POST /api/v1/analytics/embed-token/:dashboard_uuid
// Generate guest token for embedding
func (h *AnalyticsHandler) GenerateEmbedToken(w http.ResponseWriter, r *http.Request)

// GET /api/v1/analytics/dashboards/:dashboard_uuid
// Get dashboard metadata
func (h *AnalyticsHandler) GetDashboard(w http.ResponseWriter, r *http.Request)
```

### Request/Response Examples

**Generate Embed Token**:
```json
POST /api/v1/analytics/embed-token/550e8400-e29b-41d4-a716-446655440000

Request:
{
  "user_id": "123e4567-e89b-12d3-a456-426614174000",
  "event_id": "789e4567-e89b-12d3-a456-426614174111",
  "team_ids": ["456e4567-e89b-12d3-a456-426614174222"],
  "username": "john.doe@example.com",
  "first_name": "John",
  "last_name": "Doe"
}

Response:
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "dashboard_uuid": "550e8400-e29b-41d4-a716-446655440000",
  "expires_at": "2026-02-04T18:00:00Z"
}
```

**List Dashboards**:
```json
GET /api/v1/analytics/dashboards

Response:
{
  "dashboards": [
    {
      "id": 1,
      "dashboard_uuid": "550e8400-e29b-41d4-a716-446655440000",
      "title": "Event Overview",
      "slug": "event-overview",
      "published": true,
      "description": "Overall event statistics and metrics"
    },
    {
      "id": 2,
      "dashboard_uuid": "660e8400-e29b-41d4-a716-446655440111",
      "title": "Player Statistics",
      "slug": "player-stats",
      "published": true,
      "description": "Individual player performance metrics"
    }
  ]
}
```

## Frontend Integration

### Embedding with @superset-ui/embedded-sdk

```typescript
import { embedDashboard } from '@superset-ui/embedded-sdk';

async function embedSupersetDashboard(
  containerId: string,
  dashboardUuid: string,
  filters?: Record<string, any>
) {
  // Get guest token from backend
  const response = await fetch(
    `/api/v1/analytics/embed-token/${dashboardUuid}`,
    {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        user_id: currentUser.id,
        event_id: selectedEvent.id,
        team_ids: currentUser.teamIds,
        username: currentUser.email,
        first_name: currentUser.firstName,
        last_name: currentUser.lastName,
      }),
    }
  );

  const { token } = await response.json();

  // Embed dashboard
  embedDashboard({
    id: dashboardUuid,
    supersetDomain: 'https://superset.codevertexitsolutions.com',
    mountPoint: document.getElementById(containerId)!,
    fetchGuestToken: () => Promise.resolve(token),
    dashboardUiConfig: {
      hideTitle: false,
      hideChartControls: false,
      hideTab: false,
      filters: {
        expanded: true,
        visible: true,
      },
    },
  });
}
```

## Security Considerations

### 1. Token Expiration

- Guest tokens expire after 5 minutes (configurable)
- Implement token refresh mechanism
- Cache tokens server-side with Redis

### 2. Permission Validation

```go
func (s *Service) buildRLSRules(userID, eventID uuid.UUID, teamIDs []uuid.UUID) []RLSRule {
    rules := []RLSRule{}
    
    // Event-level filtering
    if eventID != uuid.Nil {
        rules = append(rules, RLSRule{
            Clause: fmt.Sprintf("event_id = '%s'", eventID),
        })
    }
    
    // Team-level filtering
    if len(teamIDs) > 0 {
        teamIDsStr := strings.Join(uuidsToStrings(teamIDs), "','")
        rules = append(rules, RLSRule{
            Clause: fmt.Sprintf("team_id IN ('%s')", teamIDsStr),
        })
    }
    
    return rules
}
```

### 3. HTTPS/TLS

- All Superset communication over HTTPS
- Validate SSL certificates
- Use secure session cookies

### 4. Rate Limiting

```go
// Limit embed token generation
// Max 10 tokens per user per minute
rateLimit := 10
rateLimitWindow := time.Minute
```

## Environment Variables

```bash
# Superset Configuration
SUPERSET_URL=https://superset.codevertexitsolutions.com
SUPERSET_API_USERNAME=admin
SUPERSET_API_PASSWORD=${SUPERSET_ADMIN_PASSWORD}
SUPERSET_DATABASE_URI=postgresql://game_stats_user@postgresql.infra.svc.cluster.local:5432/game_stats

# Token Configuration
SUPERSET_GUEST_TOKEN_TTL=300 # 5 minutes
SUPERSET_TOKEN_CACHE_ENABLED=true
SUPERSET_TOKEN_CACHE_TTL=240 # 4 minutes (before expiry)

# Redis for token caching
REDIS_URL=redis://redis-master.infra.svc.cluster.local:6379/2
```

## Next Steps

1. **Sprint 3 Week 2 Day 6-8**: Implement analytics service with Superset client
2. **Sprint 3 Week 2 Day 9-11**: Add Ollama LLM for natural language queries
3. **Sprint 3 Week 3**: Optimize with Redis caching and performance tuning

## References

- [Apache Superset Documentation](https://superset.apache.org/docs/)
- [Superset REST API](https://superset.apache.org/docs/api)
- [Superset Embedded SDK](https://www.npmjs.com/package/@superset-ui/embedded-sdk)
- [DevOps Deployment Guide](../../../devops-k8s/docs/superset-deployment.md)
