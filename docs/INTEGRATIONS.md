# Game Stats API - Integration Specifications

## Overview

This document defines all integration points between the Game Stats backend API and external systems, including the frontend application, third-party services, and infrastructure components.

---

## Frontend Integration

### Base Configuration

**API Base URL**: 
- Development: `http://localhost:4000/api/v1`
- Staging: `https://api-staging.gamestats.com/api/v1`
- Production: `https://api.gamestats.com/api/v1`

**Content Type**: `application/json`
**Character Encoding**: UTF-8

---

### Authentication

#### Login

**Endpoint**: `POST /auth/login`

**Request**:
```json
{
  "email": "user@example.com",
  "password": "SecurePass123!"
}
```

**Response** (200 OK):
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "expires_in": 900,
  "token_type": "Bearer",
  "user": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "email": "user@example.com",
    "full_name": "John Doe",
    "role": "admin"
  }
}
```

**Error Response** (401 Unauthorized):
```json
{
  "code": "INVALID_CREDENTIALS",
  "message": "Invalid email or password"
}
```

#### Token Refresh

**Endpoint**: `POST /auth/refresh`

**Request**:
```json
{
  "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

**Response** (200 OK):
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "expires_in": 900,
  "token_type": "Bearer"
}
```

#### Authorization Header

All authenticated requests must include:
```
Authorization: Bearer {access_token}
```

---

### Server-Sent Events (SSE)

#### Live Game Updates

**Endpoint**: `GET /games/{game_id}/stream`

**Headers**:
```
Authorization: Bearer {access_token}
Accept: text/event-stream
Cache-Control: no-cache
```

**Event Format**:
```
event: goal_scored
data: {"game_id":"550e8400-e29b-41d4-a716-446655440000","player_id":"660e8400-e29b-41d4-a716-446655440001","team":"home","minute":15,"second":30,"score":{"home":6,"away":5}}

event: stoppage_recorded
data: {"game_id":"550e8400-e29b-41d4-a716-446655440000","duration_seconds":45,"reason":"injury"}

event: game_ended
data: {"game_id":"550e8400-e29b-41d4-a716-446655440000","final_score":{"home":15,"away":13}}
```

**Event Types**:
- `game_started` - Game begins
- `goal_scored` - Goal recorded
- `assist_recorded` - Assist recorded
- `stoppage_started` - Stoppage begins
- `stoppage_ended` - Stoppage ends
- `game_finished` - Time expired
- `score_updated` - Admin edit
- `game_ended` - Final submission

**Frontend Implementation**:
```typescript
const eventSource = new EventSource(
  `${API_BASE_URL}/games/${gameId}/stream`,
  {
    headers: {
      'Authorization': `Bearer ${accessToken}`
    }
  }
);

eventSource.addEventListener('goal_scored', (event) => {
  const data = JSON.parse(event.data);
  updateGameState(data);
});

eventSource.addEventListener('error', (error) => {
  console.error('SSE Error:', error);
  eventSource.close();
  // Implement reconnection logic
});
```

---

### API Response Format

#### Success Response

```json
{
  "data": {
    // Resource or array of resources
  },
  "meta": {
    "page": 1,
    "limit": 20,
    "total": 150,
    "total_pages": 8
  }
}
```

#### Error Response

```json
{
  "code": "VALIDATION_ERROR",
  "message": "Invalid input parameters",
  "details": {
    "email": ["must be a valid email address"],
    "password": ["must be at least 8 characters"]
  }
}
```

#### Validation Errors

**Field-Level Errors**:
```json
{
  "code": "VALIDATION_ERROR",
  "message": "Validation failed",
  "details": {
    "name": ["required", "max length is 100"],
    "email": ["must be unique"]
  }
}
```

---

### Pagination

**Query Parameters**:
- `page` (integer, default: 1): Page number
- `limit` (integer, default: 20, max: 100): Items per page
- `sort` (string): Field to sort by
- `order` (string): `asc` or `desc`

**Example Request**:
```
GET /api/v1/games?page=2&limit=50&sort=scheduled_time&order=desc
```

**Response Meta**:
```json
{
  "meta": {
    "page": 2,
    "limit": 50,
    "total": 237,
    "total_pages": 5,
    "has_next": true,
    "has_previous": true
  }
}
```

---

### Filtering

**Supported Operators**:
- Equality: `?field=value`
- In: `?field=value1,value2` (comma-separated)
- Greater than: `?field__gt=value`
- Less than: `?field__lt=value`
- Greater than or equal: `?field__gte=value`
- Less than or equal: `?field__lte=value`
- Like: `?field__like=value` (case-insensitive)

**Example**:
```
GET /api/v1/games?status=in_progress,finished&scheduled_time__gte=2026-01-20T00:00:00Z
```

---

### CORS Configuration

**Allowed Origins**:
- Development: `http://localhost:3000`
- Staging: `https://staging.gamestats.com`
- Production: `https://gamestats.com`

**Allowed Methods**: GET, POST, PUT, PATCH, DELETE, OPTIONS
**Allowed Headers**: Authorization, Content-Type, X-Request-ID
**Exposed Headers**: X-Total-Count, X-Page, X-Per-Page
**Credentials**: true
**Max Age**: 86400 (24 hours)

---

## Third-Party Integrations

### OpenAI API (AI Analytics)

**Purpose**: Natural language analytics queries

**Configuration**:
```yaml
openai:
  api_key: ${OPENAI_API_KEY}
  model: gpt-4-turbo
  embedding_model: text-embedding-ada-002
  max_tokens: 2000
  temperature: 0.3
```

**Request Flow**:
1. User submits natural language query
2. Generate embedding for semantic search
3. Retrieve relevant context from pgvector
4. Build prompt with schema context
5. Call OpenAI API for SQL generation
6. Validate and execute SQL
7. Return results with explanation

**Rate Limits**:
- Embeddings: 3,000,000 tokens/min
- Chat: 40,000 tokens/min

**Error Handling**:
- Retry with exponential backoff (max 3 retries)
- Fallback to predefined templates if API unavailable

---

### Redis (Caching & Pub/Sub)

**Purpose**: Session storage, caching, SSE event distribution

**Configuration**:
```yaml
redis:
  host: ${REDIS_HOST}
  port: 6379
  password: ${REDIS_PASSWORD}
  db: 0
  pool_size: 100
  max_retries: 3
  dial_timeout: 5s
  read_timeout: 3s
  write_timeout: 3s
```

**Key Patterns**:
| Pattern | Purpose | TTL |
|---------|---------|-----|
| `session:{token}` | User sessions | 7 days |
| `cache:game:{id}` | Game data | 5 min |
| `cache:team:{id}:stats` | Team statistics | 5 min |
| `cache:player:{id}:stats` | Player statistics | 5 min |
| `ratelimit:{ip}:{endpoint}` | Rate limiting | 1 min |

**Pub/Sub Channels**:
| Channel | Purpose |
|---------|---------|
| `game:{id}:events` | Game-specific SSE events |
| `global:events` | System-wide events |
| `cache:invalidate` | Cache invalidation notifications |

---

### Email Service (SendGrid / AWS SES)

**Purpose**: Transactional emails, notifications

**Configuration**:
```yaml
email:
  provider: sendgrid  # or aws_ses
  api_key: ${EMAIL_API_KEY}
  from_email: noreply@gamestats.com
  from_name: GameStats
  templates:
    welcome: d-abc123
    password_reset: d-def456
    game_reminder: d-ghi789
```

**Email Templates**:
1. **Welcome Email**: Sent on user registration
2. **Password Reset**: Password recovery link
3. **Game Reminder**: 1 hour before scheduled game
4. **Tournament Invitation**: Team added to event
5. **Results Summary**: End-of-tournament recap

**Async Processing**: 
- Use goroutines for non-blocking sends
- Retry failed sends up to 3 times
- Log all send attempts

---

### File Storage (AWS S3 / MinIO)

**Purpose**: Team logos, player images, event banners

**Configuration**:
```yaml
storage:
  provider: s3  # or minio
  bucket: gamestats-media
  region: us-east-1
  access_key: ${S3_ACCESS_KEY}
  secret_key: ${S3_SECRET_KEY}
  public_url: https://media.gamestats.com
```

**Upload Workflow**:
1. Frontend requests pre-signed URL from backend
2. Backend generates signed URL (valid for 15 minutes)
3. Frontend uploads file directly to S3
4. Frontend notifies backend of successful upload
5. Backend saves file URL in database

**Supported Formats**:
- Images: JPEG, PNG, WebP (max 5MB)
- Documents: PDF (max 10MB)

**Security**:
- Server-side encryption (SSE-S3)
- Bucket policy restricts public access
- Pre-signed URLs for uploads
- CDN (CloudFront) for public assets

---

### Metabase (Analytics Platform)

**Purpose**: Embedded analytics dashboards with iframe integration

**Configuration**:
```yaml
metabase:
  url: ${METABASE_URL}
  secret_key: ${METABASE_SECRET_KEY}
  site_url: ${METABASE_SITE_URL}
  embed_enabled: true
```

**Dashboard Embedding Flow**:
1. Backend generates signed JWT for dashboard
2. Frontend receives embed token
3. Frontend loads Metabase dashboard in iframe
4. Row-level security applied via token params

**Embed Token Generation**:
```go
type MetabaseEmbedPayload struct {
	Resource map[string]interface{} `json:"resource"`
	Params   map[string]interface{} `json:"params"`
	Exp      int64                  `json:"exp"`
}

func GenerateEmbedToken(dashboardID int, userID uuid.UUID) (string, error) {
	payload := MetabaseEmbedPayload{
		Resource: map[string]interface{}{
			"dashboard": dashboardID,
		},
		Params: map[string]interface{}{
			"user_id":  userID.String(),
			"event_id": eventID.String(),
		},
		Exp: time.Now().Add(10 * time.Minute).Unix(),
	}
	
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims(payload))
	return token.SignedString([]byte(metabaseSecretKey))
}
```

**API Endpoints**:
- `GET /api/v1/analytics/dashboards` - List dashboards
- `POST /api/v1/analytics/embed-token/:id` - Generate embed token
- `POST /api/v1/analytics/query` - Natural language query

---

### Ollama (LLM for Text-to-SQL)

**Purpose**: Lightweight local LLM for natural language to SQL conversion

**Configuration**:
```yaml
ollama:
  url: http://localhost:11434
  model: duckdb-nsql:7b  # or sqlcoder:15b
  timeout: 30s
  max_retries: 3
```

**Models**:
| Model | Size | Best For |
|-------|------|----------|
| duckdb-nsql:7b | 4.1GB | PostgreSQL/DuckDB queries |
| sqlcoder:15b | 8.5GB | Complex SQL, higher accuracy |
| codellama:7b | 3.8GB | General code generation |

**API Request**:
```json
POST http://localhost:11434/api/generate
{
  "model": "duckdb-nsql:7b",
  "prompt": "### Task:\nGenerate PostgreSQL query for: 'Top 5 scorers in Men's division'\n\n### Schema:\n...\n\n### Response:",
  "stream": false
}
```

**Response Parsing**:
```go
type OllamaResponse struct {
	Response string `json:"response"`
	Done     bool   `json:"done"`
}

// Parse JSON from LLM response
type SQLGeneration struct {
	SQL         string `json:"sql"`
	ChartType   string `json:"chart_type"`
	Explanation string `json:"explanation"`
}
```

**Docker Deployment**:
```yaml
ollama:
  image: ollama/ollama:latest
  ports:
    - "11434:11434"
  volumes:
    - ollama-models:/root/.ollama
  environment:
    - OLLAMA_KEEP_ALIVE=24h
  deploy:
    resources:
      limits:
        memory: 8GB
      reservations:
        devices:
          - driver: nvidia
            count: 1
            capabilities: [gpu]
```

**Error Handling**:
- Timeout after 30s
- Retry up to 3 times with exponential backoff
- Fallback to predefined query templates
- SQL injection validation before execution

---

## Analytics Query Flow

###Complete Flow:

```
1. User submits natural language query
   ↓
2. Frontend → POST /api/v1/analytics/query
   ↓
3. Backend generates embedding for query (optional, for schema search)
   ↓
4. pgvector semantic search for relevant tables/columns
   ↓
5. Build schema context + examples
   ↓
6. Call Ollama LLM with prompt
   ↓
7. Parse SQL + chart type from LLM response
   ↓
8. Validate SQL (prevent injection)
   ↓
9. Execute query with row limit (max 1000)
   ↓
10. Return results + chart type to frontend
   ↓
11. Frontend renders chart using Tremor/Recharts
   ↓
12. (Optional) Save as Metabase dashboard for future use
```

### Example API Call:

**Request**:
```json
POST /api/v1/analytics/query
{
  "query": "Show me the top 5 players with most goals in the Men's Open division",
  "save_dashboard": true,
  "dashboard_name": "Top Scorers - Men's Open"
}
```

**Response**:
```json
{
  "sql": "SELECT p.name, SUM(s.goals) as total_goals FROM players p JOIN scoring s ON s.player_id = p.id JOIN teams t ON p.team_id = t.id JOIN division_pools dp ON t.division_pool_id = dp.id WHERE dp.division_type = 'open' AND p.gender = 'M' GROUP BY p.id, p.name ORDER BY total_goals DESC LIMIT 5",
  "chart_type": "bar",
  "explanation": "This query aggregates goals scored by male players in the Open division, showing the top 5 scorers.",
  "data": [
    {"name": "John Doe", "total_goals": 45},
    {"name": "Mike Smith", "total_goals": 38},
    ...
  ],
  "dashboard_id": "550e8400-e29b-41d4-a716-446655440000"
}
```

---

### Prometheus Metrics

**Endpoint**: `GET /metrics`

**Exported Metrics**:
```
# HTTP Requests
http_requests_total{method, path, status}
http_request_duration_seconds{method, path}

# SSE Connections
sse_active_connections{game_id}
sse_messages_sent_total{event_type}

# Database
db_query_duration_seconds{operation}
db_connections_active
db_connections_idle

# Cache
cache_hits_total{key_pattern}
cache_misses_total{key_pattern}

# Business Metrics
games_active
games_completed_total
users_online
```

### Structured Logging

**Format**: JSON
**Fields**: timestamp, level, message, request_id, user_id, context

**Example**:
```json
{
  "timestamp": "2026-01-20T15:30:45Z",
  "level": "info",
  "message": "Game started successfully",
  "request_id": "req_abc123",
  "user_id": "550e8400-e29b-41d4-a716-446655440000",
  "game_id": "660e8400-e29b-41d4-a716-446655440001",
  "duration_ms": 45
}
```

**Log Levels**:
- `debug`: Development/troubleshooting
- `info`: Normal operations
- `warn`: Potential issues
- `error`: Errors requiring attention
- `fatal`: Critical failures

---

## Health Checks

### Liveness Probe

**Endpoint**: `GET /health`

**Response** (200 OK):
```json
{
  "status": "ok",
  "timestamp": "2026-01-20T15:30:45Z"
}
```

**Purpose**: Indicates the application is running

---

### Readiness Probe

**Endpoint**: `GET /ready`

**Response** (200 OK):
```json
{
  "status": "ready",
  "checks": {
    "database": "ok",
    "redis": "ok",
    "migrations": "ok"
  },
  "timestamp": "2026-01-20T15:30:45Z"
}
```

**Response** (503 Service Unavailable):
```json
{
  "status": "not_ready",
  "checks": {
    "database": "ok",
    "redis": "error",
    "migrations": "ok"
  },
  "timestamp": "2026-01-20T15:30:45Z"
}
```

**Purpose**: Indicates the application is ready to accept traffic

---

## Webhook Support

### Webhook Configuration

**Table**: `webhooks`

| Column | Type | Description |
|--------|------|-------------|
| id | UUID | Primary key |
| event_type | STRING | `game.started`, `game.ended`, `score.updated` |
| url | STRING | Callback URL |
| secret | STRING | HMAC secret for signature verification |
| is_active | BOOLEAN | Enable/disable |
| created_at | TIMESTAMP | Creation time |

### Webhook Delivery

**Headers**:
```
Content-Type: application/json
X-Webhook-Signature: sha256=abc123...
X-Webhook-Event: game.started
X-Webhook-Delivery-ID: delivery_xyz789
```

**Payload**:
```json
{
  "event": "game.started",
  "timestamp": "2026-01-20T15:30:45Z",
  "data": {
    "game_id": "550e8400-e29b-41d4-a716-446655440000",
    "home_team": {...},
    "away_team": {...}
  }
}
```

**Signature Verification**:
```
HMAC-SHA256(secret, payload) = X-Webhook-Signature
```

**Retry Logic**:
- Retry on 5xx errors or timeout
- Exponential backoff: 1s, 2s, 4s, 8s, 16s
- Max 5 retry attempts
- Mark as failed after max retries

---

## Rate Limiting

### Limits by Tier

| Tier | Rate Limit | Burst | Window |
|------|------------|-------|--------|
| Anonymous | 20 req/min | 30 | 1 min |
| Authenticated | 100 req/min | 150 | 1 min |
| Team Manager | 300 req/min | 500 | 1 min |
| Event Manager | 1000 req/min | 2000 | 1 min |
| Admin | Unlimited | - | - |

### Response Headers

```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 73
X-RateLimit-Reset: 1737381000
```

### Rate Limit Exceeded

**Response** (429 Too Many Requests):
```json
{
  "code": "RATE_LIMIT_EXCEEDED",
  "message": "Rate limit exceeded. Try again in 45 seconds.",
  "retry_after": 45
}
```

---

## WebSocket Fallback (Optional)

For clients that cannot use SSE, WebSocket support is available:

**Endpoint**: `WS /ws/games/{game_id}`

**Message Format**:
```json
{
  "type": "goal_scored",
  "timestamp": "2026-01-20T15:30:45Z",
  "data": {...}
}
```

**Connection**:
```javascript
const ws = new WebSocket(`wss://api.gamestats.com/ws/games/${gameId}?token=${accessToken}`);

ws.onmessage = (event) => {
  const message = JSON.parse(event.data);
  handleGameEvent(message);
};
```

---

## Environment Variables

### Required

| Variable | Description | Example |
|----------|-------------|---------|
| `DATABASE_URL` | PostgreSQL connection string | `postgres://user:pass@host:5432/db` |
| `REDIS_URL` | Redis connection string | `redis://:password@host:6379` |
| `JWT_SECRET` | Secret for JWT signing | `your-256-bit-secret` |
| `JWT_REFRESH_SECRET` | Secret for refresh tokens | `different-256-bit-secret` |
| `OPENAI_API_KEY` | OpenAI API key | `sk-...` |
| `S3_BUCKET` | S3 bucket name | `gamestats-media` |
| `S3_ACCESS_KEY` | AWS access key | `AKIA...` |
| `S3_SECRET_KEY` | AWS secret key | `...` |

### Optional

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | HTTP server port | `4000` |
| `ENV` | Environment (dev, staging, prod) | `dev` |
| `LOG_LEVEL` | Logging level | `info` |
| `CORS_ORIGINS` | Allowed CORS origins | `http://localhost:3000` |
| `MAX_UPLOAD_SIZE` | Max upload size (bytes) | `10485760` (10MB) |

---

## API Versioning Strategy

### Current Version: v1

**URL Pattern**: `/api/v1/{resource}`

### Deprecation Policy

1. **Announcement**: 6 months before deprecation
2. **Warning Headers**: `X-API-Deprecation-Date`, `X-API-Sunset-Date`
3. **Migration Guide**: Provide detailed migration docs
4. **Support Period**: Maintain old version for 12 months after new version release

### Example Deprecation

```
X-API-Deprecation-Date: 2026-06-01
X-API-Sunset-Date: 2027-06-01
X-API-Migration-Guide: https://docs.gamestats.com/api/v1-to-v2-migration
```

---

## Conclusion

This integration specification provides a comprehensive guide for all interactions with the Game Stats API. All integrations follow industry-standard protocols and best practices for security, reliability, and performance.

**For additional details**, refer to:
- [API_REFERENCE.md](./API_REFERENCE.md) - Complete API endpoint documentation
- [PLAN.md](./PLAN.md) - Backend architecture and implementation details
- [ERD.md](./ERD.md) - Database schema reference
