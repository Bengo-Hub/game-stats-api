# Analytics Service Integration with Metabase

## Overview

This document outlines the integration between the Game Stats API and Metabase for embedded analytics, dashboards, and AI-powered queries.

## Metabase Deployment Architecture

### Infrastructure Components

**Metabase Server**
- Serves web UI and REST API
- Handles authentication and authorization
- Processes queries
- **Ingress**: https://analytics.ultichange.org

**External Dependencies**
- PostgreSQL: `postgresql.infra.svc.cluster.local:5432`

### Database Configuration

Metabase connects directly to the Game Stats database `game_stats`.
This enables direct query access to game statistics, real-time dashboard updates, and no data synchronization needed.

## Integration Patterns

### 1. Embedded Dashboard Pattern (Signed Embedding)

Metabase uses signed embedding with a secret key.

**Embed Token Flow**:
1. Game Stats API generates a JWT token signed with `METABASE_EMBED_SECRET`.
2. The token includes the dashboard ID and relevant parameters (like `event_id`).
3. Game Stats API returns the token/URL to the frontend.
4. Frontend embeds the dashboard via an `<iframe>`.

### 2. Row-Level Security (RLS)

- Filter dashboards by event_id, team_id, or user permissions
- Passed via locked parameters inside the signed JWT.

### 3. API Integration

**Metabase REST API**: `https://analytics.ultichange.org/api/`

**Key Endpoints**:
- `POST /api/session/` - Authenticate
- `GET /api/dashboard/` - List dashboards
- `GET /api/dashboard/:id` - Get dashboard details

## Analytics Service Implementation

### Authentication Flow

The service authenticates using the API credentials to perform admin actions such as listing dashboards.

### Embed Token Generation

The `MetabaseClient` generates the signed URL using the `METABASE_EMBED_SECRET` without needing a network request.

```go
// Example flow inside the service:
token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
    "resource": map[string]interface{}{"dashboard": req.DashboardID},
    "params": map[string]interface{}{
        "event_id": req.EventID,
    },
    "exp": time.Now().Add(10 * time.Minute).Unix(),
})
signedToken, _ := token.SignedString([]byte(cfg.MetabaseEmbedSecret))
```

## Security Considerations

### 1. Token Expiration
- Guest tokens expire after 10 minutes.
- Frontend should fetch a new iframe URL to refresh.

### 2. Parameter Locking
- Parameters like `event_id` and `team_id` must be locked in the JWT payload so users cannot modify them.

## Environment Variables

```bash
# Metabase Configuration
METABASE_BASE_URL=https://analytics.ultichange.org
METABASE_USERNAME=admin@bengobox.com
METABASE_PASSWORD=secret
METABASE_EMBED_SECRET=your_metabase_embed_secret

# Ollama Text-to-SQL
OLLAMA_BASE_URL=http://ollama.infra.svc.cluster.local:11434
OLLAMA_MODEL=duckdb-nsql:7b
```
