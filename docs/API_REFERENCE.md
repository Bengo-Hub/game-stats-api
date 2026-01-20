# Game Stats API Reference

Complete REST API documentation for Game Stats backend.

**Base URL**: `https://api.gamestats.com/api/v1`

---

## Authentication

### Login
```http
POST /auth/login
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "password123"
}
```

**Response** (200):
```json
{
  "access_token": "eyJhbG...",
  "refresh_token": "eyJhbG...",
  "expires_in": 900,
  "user": {
    "id": "uuid",
    "email": "user@example.com",
    "role": "admin"
  }
}
```

### Refresh Token
```http
POST /auth/refresh
Content-Type: application/json

{
  "refresh_token": "eyJhbG..."
}
```

---

## Games

### List Games
```http
GET /games?page=1&limit=20&status=in_progress&division_id=uuid
Authorization: Bearer {token}
```

**Query Parameters**:
- `page` (int): Page number (default: 1)
- `limit` (int): Items per page (default: 20, max: 100)
- `status` (string): Filter by status (scheduled, in_progress, finished, ended)
- `division_id` (uuid): Filter by division
- `field_id` (uuid): Filter by field
- `date` (date): Filter by date (YYYY-MM-DD)

**Response** (200):
```json
{
  "data": [
    {
      "id": "uuid",
      "home_team": { "id": "uuid", "name": "Team A" },
      "away_team": { "id": "uuid", "name": "Team B" },
      "status": "in_progress",
      "home_score": 10,
      "away_score": 8,
      "scheduled_time": "2026-01-20T14:00:00Z",
      "field": { "id": "uuid", "name": "Field 1" }
    }
  ],
  "meta": {
    "page": 1,
    "limit": 20,
    "total": 150,
    "total_pages": 8
  }
}
```

### Get Game Details
```http
GET /games/:id
Authorization: Bearer {token}
```

### Schedule Game
```http
POST /divisions/:id/games
Authorization: Bearer {token}
Content-Type: application/json

{
  "home_team_id": "uuid",
  "away_team_id": "uuid",
  "field_id": "uuid",
  "scheduled_time": "2026-01-20T14:00:00Z",
  "allocated_time_minutes": 90,
  "scorekeeper_id": "uuid"
}
```

### Start Game
```http
POST /games/:id/start
Authorization: Bearer {token}
```

**Response** (200):
```json
{
  "id": "uuid",
  "status": "in_progress",
  "actual_start_time": "2026-01-20T14:02:30Z",
  "expected_end_time": "2026-01-20T15:32:30Z"
}
```

### Record Score
```http
POST /games/:id/scores
Authorization: Bearer {token}
Content-Type: application/json

{
  "player_id": "uuid",
  "goals": 1,
  "assists": 0,
  "minute": 15,
  "second": 30
}
```

### SSE Live Updates
```http
GET /games/:id/stream
Authorization: Bearer {token}
Accept: text/event-stream
```

**Events**:
```
event: goal_scored
data: {"game_id":"uuid","player_id":"uuid","team":"home","minute":15,"score":{"home":6,"away":5}}

event: game_finished
data: {"game_id":"uuid","final_score":{"home":15,"away":13}}
```

---

## Teams

### List Teams
```http
GET /teams?division_id=uuid
Authorization: Bearer {token}
```

### Get Team Details
```http
GET /teams/:id
Authorization: Bearer {token}
```

**Response** (200):
```json
{
  "id": "uuid",
  "name": "Team A",
  "division_pool": { "id": "uuid", "name": "Men's Open" },
  "roster": [
    {
      "id": "uuid",
      "name": "John Doe",
      "number": 7,
      "gender": "M"
    }
  ],
  "stats": {
    "games_played": 10,
    "wins": 7,
    "losses": 3,
    "goals_for": 150,
    "goals_against": 120,
    "spirit_average": 12.5
  }
}
```

### Create Team
```http
POST /divisions/:id/teams
Authorization: Bearer {token}
Content-Type: application/json

{
  "name": "Team A",
  "country_id": "uuid",
  "manager_id": "uuid"
}
```

---

## Players

### List Players
```http
GET /players?team_id=uuid&search=john
Authorization: Bearer {token}
```

### Get Player Stats
```http
GET /players/:id/stats
Authorization: Bearer {token}
```

**Response** (200):
```json
{
  "id": "uuid",
  "name": "John Doe",
  "team": { "id": "uuid", "name": "Team A" },
  "stats": {
    "games_played": 10,
    "total_goals": 45,
    "total_assists": 20,
    "mvp_nominations": 3,
    "spirit_nominations": 2
  }
}
```

---

## Analytics

### Natural Language Query
```http
POST /analytics/query
Authorization: Bearer {token}
Content-Type: application/json

{
  "query": "Show me top 5 scorers in Men's Open division",
  "save_dashboard": false
}
```

**Response** (200):
```json
{
  "sql": "SELECT p.name, SUM(s.goals)...",
  "chart_type": "bar",
  "explanation": "This query shows...",
  "data": [
    { "name": "John Doe", "total_goals": 45 }
  ]
}
```

### Get Embed Token
```http
POST /analytics/embed-token/:dashboard_id
Authorization: Bearer {token}
```

**Response** (200):
```json
{
  "token": "eyJhbG...",
  "expires_in": 600,
  "embed_url": "https://metabase.com/embed/dashboard/..."
}
```

---

## Divisions & Standings

### Get Standings
```http
GET /divisions/:id/standings
Authorization: Bearer {token}
```

**Response** (200):
```json
{
  "division": { "id": "uuid", "name": "Men's Open" },
  "standings": [
    {
      "rank": 1,
      "team": { "id": "uuid", "name": "Team A" },
      "games": 10,
      "wins": 8,
      "draws": 1,
      "losses": 1,
      "points": 25,
      "goal_diff": 30,
      "goals_for": 150,
      "goals_against": 120
    }
  ]
}
```

---

## Brackets

### Get Tournament Bracket
```http
GET /events/:id/bracket
Authorization: Bearer {token}
```

**Response** (200):
```json
{
  "event": { "id": "uuid", "name": "Championship 2026" },
  "bracket": {
    "id": "uuid",
    "round": "final",
    "game_id": "uuid",
    "home_team": { "id": "uuid", "name": "Team A" },
    "away_team": { "id": "uuid", "name": "Team B" },
    "score": { "home": 15, "away": 13 },
    "children": [...]
  }
}
```

---

## Error Codes

| Code | Status | Description |
|------|--------|-------------|
| `UNAUTHORIZED` | 401 | Invalid or missing token |
| `FORBIDDEN` | 403 | Insufficient permissions |
| `NOT_FOUND` | 404 | Resource not found |
| `VALIDATION_ERROR` | 400 | Invalid input |
| `RATE_LIMIT_EXCEEDED` | 429 | Too many requests |
| `INTERNAL_ERROR` | 500 | Server error |

**Error Response Format**:
```json
{
  "code": "VALIDATION_ERROR",
  "message": "Invalid input parameters",
  "details": {
    "email": ["must be a valid email"]
  }
}
```

---

## Rate Limits

| Tier | Limit | Window |
|------|-------|--------|
| Anonymous | 20 req/min | 1 min |
| Authenticated | 100 req/min | 1 min |
| Admin | Unlimited | - |

**Headers**:
```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 73
X-RateLimit-Reset: 1737381000
```

---

For complete OpenAPI spec, see: https://api.gamestats.com/swagger
