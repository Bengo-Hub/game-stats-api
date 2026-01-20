# Game Stats API - Backend Implementation Plan

## Executive Summary

This document outlines the comprehensive backend architecture for the Game Stats application, a production-ready tournament management system built with **Go 1.24+**, **Ent ORM v0.11**, and **PostgreSQL 17** with pgvector extension. The system addresses all gaps from the legacy Django implementation while incorporating modern best practices and UltiScore-inspired features.

**Key Features**:
- Real-time score updates via Server-Sent Events (SSE)
- Game timeline with stoppage tracking
- Automatic ranking and round progression
- Tournament brackets visualization
- Spirit of the Game tracking
- AI-powered analytics with natural language queries
- PWA support with offline capabilities

---

## Technology Stack

### Core Technologies

| Component | Technology | Version | Rationale |
|-----------|------------|---------|-----------|
| Language | Go | 1.24+ | High performance, excellent concurrency, static typing, built-in tooling |
| ORM | Ent | v0.11+ | Schema-as-code, type-safe, graph traversal, automatic migrations |
| Database | PostgreSQL | 17 | Latest stable, pgvector support, excellent performance |
| Vector DB | pgvector | v0.8.1 | AI analytics, semantic search |
| Cache | Redis | 7.2+ | Sub-millisecond latency, pub/sub for SSE |
| HTTP Router | Chi | v5+ | Lightweight, idiomatic Go, excellent middleware support, standard library compatible |
| Real-time | SSE | Native | Lower overhead than WebSockets for unidirectional updates |
| Analytics Platform | Metabase | Latest | Open-source, easy embedding, iframe support, lightweight |
| LLM | Ollama + duckdb-nsql | 7B | Local inference, SQL-optimized, low resource usage |

### Supporting Libraries

| Purpose | Library | Version |
|---------|---------|---------|
| Authentication | golang-jwt/jwt | v5+ |
| Password Hashing | golang.org/x/crypto/bcrypt | Latest |
| Validation | go-playground/validator | v10+ |
| Configuration | spf13/viper | v1.18+ |
| Logging | uber-go/zap | v1.26+ |
| Metrics | prometheus/client_golang | v1.18+ |
| Testing | testify | v1.8+ |
| Mocking | mockery | v2+ |
| API Docs | swaggo/swag | v1.16+ |
| Migrations | Ent built-in | - |
| UUID | google/uuid | v1.5+ |
| Rate Limiting | ulule/limiter | v3+ |
| CORS | rs/cors | v1.10+ |
| Graceful Shutdown | oklog/run | v1.1+ |
| **Analytics** |  |  |
| Ollama Client | HTTP native | - |
| Vector Search | pgvector-go | Latest |
| Embeddings | go-llama.cpp (optional) | Latest |

---

## Architecture Overview

### Layered Architecture

```
┌─────────────────────────────────────────┐
│         Presentation Layer              │
│  ┌─────────────────────────────────┐   │
│  │   REST API Handlers             │   │
│  │   - JSON request/response       │   │
│  │   - Input validation            │   │
│  │   - Error handling              │   │
│  └─────────────────────────────────┘   │
└─────────────────────────────────────────┘
                   ↓
┌─────────────────────────────────────────┐
│         Application Layer               │
│  ┌────────────────┬───────────────┐    │
│  │  Service Logic │  SSE Broker   │    │
│  │  - Business    │  - Event      │    │
│  │    rules       │    streaming  │    │
│  │  - Workflows   │  - Client     │    │
│  │               │    management │    │
│  └────────────────┴───────────────┘    │
└─────────────────────────────────────────┘
                   ↓
┌─────────────────────────────────────────┐
│         Domain Layer                    │
│  ┌─────────────────────────────────┐   │
│  │   Domain Models & Interfaces    │   │
│  │   - Core entities               │   │
│  │   - Repository interfaces       │   │
│  │   - Domain events               │   │
│  └─────────────────────────────────┘   │
└─────────────────────────────────────────┘
                   ↓
┌─────────────────────────────────────────┐
│      Infrastructure Layer               │
│  ┌───────────┬──────────┬──────────┐   │
│  │ Ent/DB    │  Redis   │  Vector  │   │
│  │ Repository│  Cache   │  Search  │   │
│  └───────────┴──────────┴──────────┘   │
└─────────────────────────────────────────┘
```

### Project Structure

```
games-stats-api/
├── cmd/
│   └── api/
│       └── main.go                 # Application entry point
├── internal/
│   ├── config/
│   │   ├── config.go               # Configuration loading
│   │   └── config.yaml             # Default configuration
│   ├── domain/
│   │   ├── game/
│   │   │   ├── model.go            # Game entity
│   │   │   ├── repository.go       # Repository  interface
│   │   │   └── service.go          # Service interface
│   │   ├── team/
│   │   ├── player/
│   │   ├── event/
│   │   └── user/
│   ├── application/
│   │   ├── game/
│   │   │   ├── service.go          # Business logic
│   │   │   ├── dto.go              # Data transfer objects
│   │   │   └── mapper.go           # Domain ↔ DTO mapping
│   │   ├── scorekeeper/
│   │   ├── analytics/
│   │   └── sse/
│   │       ├── broker.go           # SSE event broker
│   │       └── events.go           # Event types
│   ├── infrastructure/
│   │   ├── database/
│   │   │   ├── postgres.go         # DB connection
│   │   │   └── migrations/         # Ent migrations
│   │   ├── cache/
│   │   │   └── redis.go            # Redis client
│   │   ├── repository/
│   │   │   ├── game_repository.go  # Implementation
│   │   │   ├── team_repository.go
│   │   │   └── ...
│   │   └── vector/
│   │       └── embeddings.go       # Vector operations
│   ├── presentation/
│   │   ├── http/
│   │   │   ├── router.go           # Route definitions
│   │   │   ├── middleware/
│   │   │   │   ├── auth.go
│   │   │   │   ├── cors.go
│   │   │   │   ├── logger.go
│   │   │   │   └── rate_limit.go
│   │   │   └── handlers/
│   │   │       ├── game_handler.go
│   │   │       ├── team_handler.go
│   │   │       ├── auth_handler.go
│   │   │       └── sse_handler.go
│   │   └── dto/
│   │       └── responses.go        # API response structures
│   └── pkg/
│       ├── errors/
│       │   └── errors.go           # Custom error types
│       ├── logger/
│       │   └── logger.go           # Logging setup
│       └── validator/
│           └── validator.go        # Custom validators
├── ent/
│   ├── schema/
│   │   ├── world.go                # Ent schemas
│   │   ├── continent.go
│   │   ├── country.go
│   │   ├── game.go
│   │   └── ...
│   └── generate.go                 # Go generate directive
├── docs/
│   ├── ERD.md                      # Database schema
│   ├── PLAN.md                     # This file
│   ├── API_REFERENCE.md            # API documentation
│   ├── INTEGRATIONS.md             # Integration specs
│   └── sprints/
│       ├── BACKEND_SPRINT_1.md
│       ├── BACKEND_SPRINT_2.md
│       ├── BACKEND_SPRINT_3.md
│       └── BACKEND_SPRINT_4.md
├── tests/
│   ├── integration/
│   ├── unit/
│   └── e2e/
├── scripts/
│   ├── migrate.sh
│   ├── seed.sh
│   └── test.sh
├── deployments/
│   ├── docker/
│   │   ├── Dockerfile
│   │   └── docker-compose.yml
│   └── k8s/
│       ├── deployment.yaml
│       ├── service.yaml
│       └── ingress.yaml
├── .env.example
├── .dockerignore
├── .gitignore
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

---

## Core Features Implementation

### 1. Game Management with Timer System

**Requirement**: Game timer with start time, countdown, and stoppage tracking

**Implementation**:

```go
// Game state machine
type GameStatus string

const (
    StatusScheduled  GameStatus = "scheduled"
    StatusInProgress GameStatus = "in_progress"
    StatusFinished   GameStatus = "finished"  // Time expired, scores still editable
    StatusEnded      GameStatus = "ended"     // Final scores submitted
    StatusCanceled   GameStatus = "canceled"
)

// Service method
func (s *GameService) StartGame(ctx context.Context, gameID uuid.UUID) error {
    game, err := s.repo.GetByID(ctx, gameID)
    if err != nil {
        return err
    }
    
    if game.Status != StatusScheduled {
        return ErrInvalidGameStatus
    }
    
    now := time.Now()
    game.ActualStartTime = &now
    game.Status = StatusInProgress
    
    // Calculate expected end time
    endTime := now.Add(time.Duration(game.AllocatedTimeMinutes) * time.Minute)
    
    // Broadcast SSE event
    s.ssebroker.Publish(GameEvent{
        Type:    "game_started",
        GameID:  gameID,
        Payload: map[string]interface{}{
            "start_time": now,
            "end_time":   endTime,
        },
    })
    
    // Start background goroutine for auto-finish
    go s.scheduleAutoFinish(ctx, gameID, endTime)
    
    return s.repo.Update(ctx, game)
}

func (s *GameService) RecordStoppage(ctx context.Context, gameID uuid.UUID, duration int) error {
    // Add stoppage time and extend game end time
    // Broadcast update via SSE
}

func (s *GameService) FinishGame(ctx context.Context, gameID uuid.UUID) error {
    // Mark as finished (time expired), but allow score edits
}

func (s *GameService) EndGame(ctx context.Context, gameID uuid.UUID, userID uuid.UUID) error {
    // Final submission by scorekeeper
    // Lock scores, mark as ended
    // Trigger auto-ranking if configured
}
```

**Database Support**:
- `actual_start_time`, `actual_end_time` timestamps
- `allocated_time_minutes` for game duration
- `stoppage_time_seconds` cumulative stoppage
- `game_events` table for timeline with minute/second precision

---

### 2. Real-time Score Updates (SSE)

**Requirement**: Near real-time updates with minimal resource consumption

**Implementation**:

```go
// SSE Broker
type Broker struct {
    clients    map[uuid.UUID]chan Event
    newClients chan chan Event
    defClients chan chan Event
    events     chan Event
    mu         sync.RWMutex
}

func (b *Broker) Listen(gameID uuid.UUID) chan Event {
    clientChan := make(chan Event, 10)
    b.newClients <- clientChan
    return clientChan
}

func (b *Broker) Publish(event Event) {
    b.events <- event
}

// HTTP Handler
func (h *SSEHandler) StreamGameUpdates(w http.ResponseWriter, r *http.Request) {
    gameID := chi.URLParam(r, "gameID")
    
    // Set SSE headers
    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")
    
    // Get event channel
    clientChan := h.broker.Listen(uuid.MustParse(gameID))
    defer h.broker.Remove(clientChan)
    
    flusher, _ := w.(http.Flusher)
    
    for {
        select {
        case event := <-clientChan:
            fmt.Fprintf(w, "data: %s\n\n", event.JSON())
            flusher.Flush()
        case <-r.Context().Done():
            return
        }
    }
}
```

**Event Types**:
- `game_started` - Game begins
- `goal_scored` - Goal recorded with scorer/assister
- `stoppage_recorded` - Game stoppage event
- `game_finished` - Time expired
- `game_ended` - Final submission
- `score_updated` - Score edit by admin

**Redis Pub/Sub** for horizontal scaling across multiple API instances

---

### 3. Automatic Ranking & Round Progression

**Requirement**: Configurable ranking criteria with automatic advancement

**Implementation**:

```go
// Ranking criteria stored in DivisionPool.ranking_criteria JSONB
type RankingCriteria struct {
    Criteria []RankingRule `json:"criteria"`
    AutoAdvance bool       `json:"auto_advance"`
    TopNTeams   int        `json:"top_n_teams"`
}

type RankingRule struct {
    Field string  `json:"field"` // "points", "goal_diff", "goals_for", "head_to_head", "spirit_avg"
    Order string  `json:"order"` // "desc" or "asc"
}

// Service
func (s *DivisionService) CalculateStandings(ctx context.Context, divisionID uuid.UUID) ([]TeamStanding, error) {
    criteria, err := s.repo.GetRankingCriteria(ctx, divisionID)
    if err != nil {
        return nil, err
    }
    
    teams := s.getTeamsWithStats(ctx, divisionID)
    
    // Apply sorting based on criteria
    sort.Slice(teams, func(i, j int) bool {
        for _, rule := range criteria.Criteria {
            cmp := s.compareTeams(teams[i], teams[j], rule)
            if cmp != 0 {
                return rule.Order == "desc" && cmp > 0 || rule.Order == "asc" && cmp < 0
            }
        }
        return false
    })
    
    if criteria.AutoAdvance && s.poolPhaseComplete(ctx, divisionID) {
        go s.advanceTeams(ctx, divisionID, teams[:criteria.TopNTeams])
    }
    
    return teams, nil
}
```

---

### 4. Tournament Brackets

**Requirement**: UltiScore-style bracket visualization

**Implementation**:

```go
type BracketNode struct {
    ID        uuid.UUID      `json:"id"`
    GameID    *uuid.UUID     `json:"game_id,omitempty"`
    Round     string         `json:"round"` // "quarterfinal", "semifinal", "final"
    Position  int            `json:"position"`
    HomeTeam  *TeamSummary   `json:"home_team,omitempty"`
    AwayTeam  *TeamSummary   `json:"away_team,omitempty"`
    Score     *GameScore     `json:"score,omitempty"`
    Status    string         `json:"status"`
    Children  []*BracketNode `json:"children,omitempty"`
}

func (s *BracketService) GenerateBracket(ctx context.Context, eventID uuid.UUID) (*BracketNode, error) {
    // 1. Get all bracket-stage games ordered by round
    // 2. Build tree structure from finals backwards
    // 3. Return root node (final game)
}
```

---

### 5. Admin Score Editing

**Requirement**: Allow admins to correct scores with audit trail

**Implementation**:

```go
type ScoreEdit struct {
    ID          uuid.UUID
    GameID      uuid.UUID
    PlayerID    uuid.UUID
    Field       string    // "goals", "assists", etc.
    OldValue    int
    NewValue    int
    EditedBy    uuid.UUID
    Reason      string
    EditedAt    time.Time
}

func (s *GameService) EditScore(ctx context.Context, req EditScoreRequest) error {
    // Verify permission
    if !s.authz.CanEditScores(ctx, req.UserID, req.GameID) {
        return ErrUnauthorized
    }
    
   // Get current score
    scoring, err := s.scoringRepo.Get(ctx, req.PlayerID, req.GameID)
    if err != nil {
        return err
    }
    
    // Create audit record
    edit := ScoreEdit{
        GameID:   req.GameID,
        PlayerID: req.PlayerID,
        Field:    req.Field,
        OldValue: getFieldValue(scoring, req.Field),
        NewValue: req.NewValue,
        EditedBy: req.UserID,
        Reason:   req.Reason,
        EditedAt: time.Now(),
    }
    s.auditRepo.Create(ctx, edit)
    
    // Update score
    setFieldValue(scoring, req.Field, req.NewValue)
    
    // Broadcast SSE event
    s.ssebroker.Publish(ScoreEditedEvent{...})
    
    return s.scoringRepo.Update(ctx, scoring)
}
```

**Audit Table** in ERD captures all edits

---

## AI-Powered Analytics

**Requirement**: Natural language analytics queries with pgvector + embedded analytics dashboard platform

**Architecture**:

```
Frontend (Next.js) → Backend API (Go) → Analytics Engine
                         ↓
                   Superset/Metabase
                   (Embedded Dashboards)
                         ↓
                   Ollama + llama.cpp
                   (Text-to-SQL LLM)
                         ↓
                   pgvector + PostgreSQL
                   (Semantic Search + Data)
```

**Implementation**:

### 1. Embedded Analytics Platform

**Platform**: **Metabase** (open-source, easy embedding, excellent Next.js integration)

**Why Metabase**:
- Lightweight and fast
- Simple iframe embedding with signed tokens
- Row-level security out of the box
- Active community and good documentation
- Easy Docker deployment
- Free and open-source

```go
// Metabase embedding configuration
type MetabaseConfig struct {
	URL          string
	SecretKey    string
	EmbedEnabled bool
}

type AnalyticsDashboard struct {
	ID          uuid.UUID
	Name        string
	MetabaseID  int
	Permissions []string
}

func (s *AnalyticsService) GenerateEmbedToken(dashboardID uuid.UUID, userID uuid.UUID) (string, error) {
	// Get user permissions
	permissions := s.getPermissions(userID)
	
	// Generate signed JWT for Metabase embedding
	payload := map[string]interface{}{
		"resource": map[string]interface{}{
			"dashboard": dashboardID,
		},
		"params": map[string]interface{}{
			"user_id": userID,
		},
		"exp": time.Now().Add(10 * time.Minute).Unix(),
	}
	
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims(payload))
	signedToken, err := token.SignedString([]byte(s.config.SecretKey))
	
	return signedToken, err
}
```

**Metabase Embedding Features**:
- iframe embedding with signed tokens
- Row-level permissions
- SSO integration
- Custom branding
- API for dashboard management

### 2. Lightweight LLM for Text-to-SQL

**Model**: **Ollama** with **duckdb-nsql:7b**

**Why duckdb-nsql:7b**:
- Optimized specifically for SQL generation
- 7B parameters - good balance of accuracy and performance
- Runs efficiently on CPU (8GB RAM) or GPU
- Fast inference time (~2-5 seconds per query)
- PostgreSQL/DuckDB dialect support

**Go Integration**: Ollama HTTP API (simple REST calls)

```go
// Ollama client for text-to-SQL
type LLMClient struct {
	ollamaURL string
	model     string
	client    *http.Client
}

type TextToSQLRequest struct {
	Query     string  `json:"query"`
	Schema    string  `json:"schema"`
	Examples  []string `json:"examples,omitempty"`
}

type TextToSQLResponse struct {
	SQL         string   `json:"sql"`
	Explanation string   `json:"explanation"`
	ChartType   string   `json:"chart_type"` // bar, line, pie, table
	Confidence  float64  `json:"confidence"`
}

func (c *LLMClient) GenerateSQL(ctx context.Context, req TextToSQLRequest) (*TextToSQLResponse, error) {
	// Build prompt with schema context
	prompt := c.buildPrompt(req)
	
	// Call Ollama API
	resp, err := c.client.Post(
		c.ollamaURL+"/api/generate",
		"application/json",
		strings.NewReader(map[string]interface{}{
			"model":  c.model, // "duckdb-nsql:7b" or "sqlcoder"
			"prompt": prompt,
			"stream": false,
		}),
	)
	
	// Parse response (includes SQL + chart suggestion)
	var result TextToSQLResponse
	json.NewDecoder(resp.Body).Decode(&result)
	
	return &result, nil
}

func (c *LLMClient) buildPrompt(req TextToSQLRequest) string {
	return fmt.Sprintf(`
### Task:
Generate a PostgreSQL query for the following natural language request.

### Database Schema:
%s

### Natural Language Query:
"%s"

### Instructions:
1. Generate syntactically correct PostgreSQL SQL
2. Use appropriate JOINs, WHERE clauses, and aggregations
3. Suggest appropriate chart type (bar, line, pie, table, time_series)
4. Return JSON with: {"sql": "...", "chart_type": "...", "explanation": "..."}

### Examples:
%s

### Response (JSON):
`, req.Schema, req.Query, strings.Join(req.Examples, "\n"))
}
```

**Model Options**:
- **duckdb-nsql:7b** - 7B parameters, optimized for SQL generation
- **sqlcoder:15b** - 15B parameters, more accurate but slower
- **codellama:7b** - General code generation, decent SQL

**Deployment**: Run Ollama in Docker container alongside backend

```yaml
services:
  ollama:
    image: ollama/ollama:latest
    ports:
      - "11434:11434"
    volumes:
      - ollama_data:/root/.ollama
    environment:
      - OLLAMA_KEEP_ALIVE=24h
    deploy:
      resources:
        reservations:
          devices:
            - driver: nvidia
              count: 1
              capabilities: [gpu]
```

### 3. Semantic Search with pgvector

**Purpose**: Enhance text-to-SQL with relevant schema context and examples

```go
type VectorStore struct {
	client *ent.Client
}

type SchemaEmbedding struct {
	TableName   string
	ColumnName  string
	Description string
	Embedding   []float32 // 1536 dims for OpenAI ada-002
	Examples    []string
}

func (v *VectorStore) SearchRelevantSchema(ctx context.Context, query string, limit int) ([]SchemaEmbedding, error) {
	// Generate embedding for query
	embedding, err := v.generateEmbedding(query)
	if err != nil {
		return nil, err
	}
	
	// Vector similarity search
	var results []SchemaEmbedding
	err = v.client.AnalyticsEmbedding.Query().
		Where(
			analyticsembedding.EntityType("schema"),
		).
		Order(
			ent.Desc(sqlgraph.OrderBySpec{
				Field: "embedding <=> ?",	// Cosine distance
				Args:  []interface{}{embedding},
			}),
		).
		Limit(limit).
		Scan(ctx, &results)
	
	return results, err
}

func (s *AnalyticsService) Query(ctx context.Context, query string) (*AnalyticsResult, error) {
	// 1. Search for relevant schema using pgvector
	relevantSchema, err := s.vectorStore.SearchRelevantSchema(ctx, query, 5)
	
	// 2. Build schema context
	schemaContext := s.buildSchemaContext(relevantSchema)
	
	// 3. Generate SQL via Ollama LLM
	sqlResp, err := s.llmClient.GenerateSQL(ctx, TextToSQLRequest{
		Query:    query,
		Schema:   schemaContext,
		Examples: s.getExamples(relevantSchema),
	})
	
	// 4. Validate SQL (prevent injection)
	if !s.validateSQL(sqlResp.SQL) {
		return nil, ErrInvalidSQL
	}
	
	// 5. Execute query with row limit
	result, err := s.executeAnalyticsQuery(ctx, sqlResp.SQL, 1000)
	
	// 6. Generate human-readable summary
	summary := s.generateSummary(result, query)
	
	// 7. Create dashboard if user wants to save
	dashboard := s.createDashboard(ctx, query, sqlResp, result)
	
	return &AnalyticsResult{
		Data:        result,
		SQL:         sqlResp.SQL,
		ChartType:   sqlResp.ChartType,
		Explanation: sqlResp.Explanation,
		Summary:     summary,
		DashboardID: dashboard.ID,
	}, nil
}
```

**Vector Embeddings Table** (already in ERD):
- Pre-compute embeddings for all table/column descriptions
- Add query examples for common patterns
- Update embeddings after schema changes

### 4. Frontend Integration (Next.js)

**Charting Library**: **Tremor**

**Why Tremor**:
- Built specifically for dashboards
- Tailwind CSS styled (matches our stack)
- Excellent TypeScript support
- Responsive by default
- Clean, modern aesthetic
- React 19 compatible

```typescript
// Analytics query component
import { embedDashboard } from '@superset-ui/embedded-sdk';
import { BarChart, LineChart } from '@tremor/react';

export function AnalyticsQuery() {
  const [query, setQuery] = useState('');
  const [result, setResult] = useState(null);
  const [loading, setLoading] = useState(false);

  const handleQuery = async () => {
    setLoading(true);
    const response = await fetch('/api/analytics/query', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ query }),
    });
    
    const data = await response.json();
    setResult(data);
    setLoading(false);
  };

  const renderChart = () => {
    if (!result) return null;
    
    switch (result.chart_type) {
      case 'bar':
        return <BarChart data={result.data} />;
      case 'line':
        return <LineChart data={result.data} />;
      // ... other chart types
      default:
        return <DataTable data={result.data} />;
    }
  };

  return (
    <div>
      <Textarea
        placeholder="Ask a question about your data..."
        value={query}
        onChange={(e) => setQuery(e.target.value)}
      />
      <Button onClick={handleQuery} disabled={loading}>
        {loading ? 'Analyzing...' : 'Generate Insights'}
      </Button>
      
      {result && (
        <Card>
          <Title>{result.explanation}</Title>
          {renderChart()}
          <Text className="text-sm text-gray-500">
            SQL: {result.sql}
          </Text>
        </Card>
      )}
    </div>
  );
}
```

**Embedded Dashboard** (Metabase):

```typescript
export function EmbeddedDashboard({ dashboardId }: { dashboardId: string }) {
  useEffect(() => {
    // Fetch signed token from backend
    const token = await fetch(`/api/analytics/embed-token/${dashboardId}`)
      .then(r => r.json());
    
    // Embed Metabase dashboard
    const iframe = document.getElementById('metabase-frame');
    iframe.src = `${METABASE_URL}/embed/dashboard/${token}`;
  }, [dashboardId]);

  return (
    <iframe
      id="metabase-frame"
      className="w-full h-screen"
      frameBorder="0"
    />
  );
}
```

---

## Authentication & Authorization

### JWT Strategy

```go
type Claims struct {
    UserID    uuid.UUID `json:"user_id"`
    Email     string    `json:"email"`
    Role      string    `json:"role"`
    ManagedID *uuid.UUID `json:"managed_id,omitempty"`
    jwt.RegisteredClaims
}

func (s *AuthService) Login(ctx context.Context, email, password string) (*TokenPair, error) {
    user, err := s.userRepo.GetByEmail(ctx, email)
    if err != nil {
        return nil, ErrInvalidCredentials
    }
    
    if !bcrypt.CompareHashAndPassword(user.PasswordHash, password) {
        return nil, ErrInvalidCredentials
    }
    
    accessToken, err := generateAccessToken(user)  // 15min exp
    refreshToken, err := generateRefreshToken(user) // 7 days exp
    
    // Store refresh token in Redis
    s.redis.Set(ctx, fmt.Sprintf("refresh:%s", refreshToken), user.ID, 7*24*time.Hour)
    
    return &TokenPair{
        AccessToken:  accessToken,
        RefreshToken: refreshToken,
    }, nil
}
```

### Role-Based Access Control (RBAC)

**Roles**:
- `admin` - Full system access
- `continent_manager` - Manage continent's events
- `country_manager` - Manage country's events
- `discipline_manager` - Manage discipline's events
- `event_manager` - Manage specific event
- `team_manager` - Manage team roster
- `scorekeeper` - Record scores for assigned games
- `spectator` - Read-only access

**Permission Check**:

```go
func (a *AuthzService) CanEditGame(ctx context.Context, userID, gameID uuid.UUID) bool {
    user := a.getUserFromContext(ctx)
    game := a.gameRepo.GetByID(ctx, gameID)
    
    switch user.Role {
    case "admin":
        return true
    case "event_manager":
        return game.DivisionPool.Event.ID == user.ManagedEventID
    case "scorekeeper":
        return game.ScorekeeperID == userID
    default:
        return false
    }
}
```

---

## API Design Principles

### RESTful Conventions

- **Resource-based URLs**: `/api/v1/games/{id}`
- **HTTP Methods**: GET (read), POST (create), PUT (update), DELETE (delete)
- **Status Codes**: 200 (success), 201 (created), 400 (bad request), 401 (unauthorized), 403 (forbidden), 404 (not found), 500 (server error)
- **Pagination**: Query params `page`, `limit`, `sort`, `order`
- **Filtering**: Query params for field filters `?status=in_progress&field_id=uuid`

### API Versioning

- **URL-based**: `/api/v1/`, `/api/v2/`
- **Deprecation Headers**: `X-API-Deprecation-Date`, `X-API-Sunset-Date`

### Error Handling

```go
type APIError struct {
    Code    string                 `json:"code"`
    Message string                 `json:"message"`
    Details map[string]interface{} `json:"details,omitempty"`
}

func handleError(w http.ResponseWriter, err error) {
    var apiErr *APIError
    status := http.StatusInternalServerError
    
    switch {
    case errors.Is(err, ErrNotFound):
        status = http.StatusNotFound
        apiErr = &APIError{Code: "NOT_FOUND", Message: "Resource not found"}
    case errors.Is(err, ErrUnauthorized):
        status = http.StatusUnauthorized
        apiErr = &APIError{Code: "UNAUTHORIZED", Message: "Authentication required"}
    case errors.Is(err, ErrValidation):
        status = http.StatusBadRequest
        apiErr = &APIError{Code: "VALIDATION_ERROR", Message: err.Error()}
    default:
        apiErr = &APIError{Code: "INTERNAL_ERROR", Message: "An error occurred"}
    }
    
    respondJSON(w, status, apiErr)
}
```

---

## Database Optimization

### Connection Pooling

```go
func NewPostgresClient(cfg *Config) (*ent.Client, error) {
    db, err := sql.Open("postgres", cfg.DatabaseURL)
    if err != nil {
        return nil, err
    }
    
    // Connection pool settings
    db.SetMaxOpenConns(100)
    db.SetMaxIdleConns(10)
    db.SetConnMaxLifetime(time.Hour)
    db.SetConnMaxIdleTime(10 * time.Minute)
    
    driver := entsql.OpenDB("postgres", db)
    return ent.NewClient(ent.Driver(driver)), nil
}
```

### Query Optimization

**Eager Loading**:
```go
games, err := client.Game.Query().
    WithHomeTeam().
    WithAwayTeam().
    WithField(func(q *ent.FieldQuery) {
        q.WithLocation()
    }).
    WithDivisionPool(func(q *ent.DivisionPoolQuery) {
        q.WithEvent()
    }).
    All(ctx)
```

**Prepared Statements**: Ent handles automatically

**Indexes**: Defined in ERD.md

---

## Caching Strategy

### Cache Layers

| Layer | Technology | Use Case | TTL |
|-------|------------|----------|-----|
| L1 | In-memory map | Hot data in single instance | 30s |
| L2 | Redis | Shared cache across instances | 5min |
| L3 | Materialized views | Pre-aggregated statistics | Refresh on write |

### Cache Keys

```
game:{id}:live           → Live game state
team:{id}:stats          → Team statistics
player:{id}:stats        → Player statistics
division:{id}:standings  → Division standings
event:{id}:bracket       → Tournament bracket
user:{id}:permissions    → User permissions
```

### Cache Invalidation

- **Write-through**: Update cache on write
- **TTL-based**: Automatic expiration
- **Event-driven**: SSE events trigger cache invalidation
- **Manual**: Admin can force refresh

---

## Testing Strategy

### Unit Tests

- **Coverage target**: 80%+
- **Test files**: `*_test.go` alongside source
- **Mocking**: Use `testify/mock` for dependencies
- **Table-driven tests**: For multiple scenarios

```go
func TestGameService_StartGame(t *testing.T) {
    tests := []struct {
        name    string
        gameID  uuid.UUID
        setup   func(*mocks.GameRepository)
        wantErr bool
    }{
        {
            name:   "successful start",
            gameID: uuid.New(),
            setup: func(m *mocks.GameRepository) {
                m.On("GetByID", mock.Anything, mock.Anything).
                    Return(&ent.Game{Status: "scheduled"}, nil)
                m.On("Update", mock.Anything, mock.Anything).
                    Return(nil)
            },
            wantErr: false,
        },
        // More test cases...
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            mockRepo := new(mocks.GameRepository)
            tt.setup(mockRepo)
            svc := NewGameService(mockRepo, nil)
            
            err := svc.StartGame(context.Background(), tt.gameID)
            if tt.wantErr {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
            }
        })
    }
}
```

### Integration Tests

- **Test database**: Use testcontainers for PostgreSQL
- **Transaction rollback**: Each test in isolated transaction
- **Real dependencies**: No mocks for database/Redis

### E2E Tests

- **API tests**: Use `httptest` for HTTP handlers
- **Workflow tests**: Complete user journeys (create event → add teams → play games → view standings)

---

## Deployment Architecture

### Docker

```dockerfile
# Multi-stage build
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /api cmd/api/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /api ./
EXPOSE 8080
CMD ["./api"]
```

### Kubernetes

**Deployment**: 3 replicas for HA
**HPA**: Scale 3-10 based on CPU (70%) and memory (80%)
**Health Checks**: Liveness /health, Readiness /ready
**Secrets**: Mounted from Kubernetes secrets
**ConfigMaps**: Non-sensitive configuration

### CI/CD Pipeline (GitHub Actions)

1. **Lint**: golangci-lint
2. **Test**: Run unit + integration tests
3. **Build**: Docker image
4. **Scan**: Trivy for vulnerabilities
5. **Deploy**: 
   - Dev: Auto-deploy on merge to `develop`
   - Staging: Auto-deploy on merge to `main`
   - Production: Manual approval

---

## Monitoring & Observability

### Metrics (Prometheus)

- **System**: CPU, memory, goroutines
- **HTTP**: Request rate, latency (p50, p95, p99), error rate
- **Database**: Query duration, connection pool stats
- **Business**: Active games, SSE connections, API calls by endpoint

### Logging (Zap)

- **Structured logging**: JSON format
- **Log levels**: Debug, Info, Warn, Error
- **Context propagation**: Request ID, User ID
- **Sensitive data**: Redact passwords, tokens

### Tracing (OpenTelemetry)

- **Distributed tracing**: Track request flow
- **Span instrumentation**: HTTP handlers, DB queries, external calls
- **Export**: Jaeger or Tempo

---

## Security Best Practices

1. **Input Validation**: Validate all user inputs
2. **SQL Injection**: Use parameterized queries (Ent handles)
3. **XSS Protection**: Sanitize outputs (frontend responsibility)
4. **CSRF Protection**: Token-based (frontend)
5. **Rate Limiting**: 100 req/min per IP for public endpoints
6. **HTTPS Only**: TLS 1.3, redirect HTTP → HTTPS
7. **CORS**: Whitelist allowed origins
8. **Secrets Management**: Environment variables, never in code
9. **Least Privilege**: Database user has minimal permissions
10. **Security Headers**: Set via middleware (CSP, X-Frame-Options, etc.)

---

## Migration from Django

### Phase 1: Dual-Run (2 weeks)
- Deploy Go API alongside Django
- Route read traffic to Go API
- Writes still go to Django
- Validate data consistency

### Phase 2: Write Migration (1 week)
- Route all traffic to Go API
- Keep Django as backup
- Monitor for issues

### Phase 3: Decommission (1 week)
- Archive Django codebase
- Remove Django infrastructure

---

## Performance Targets

| Metric | Target | Measurement |
|--------|--------|-------------|
| API P95 Latency | < 100ms | Prometheus |
| API P99 Latency | < 200ms | Prometheus |
| Database Query P95 | < 50ms | pg_stat_statements |
| SSE Message Delivery | < 500ms | Custom metric |
| Concurrent SSE Connections | 10,000+ | Load test |
| API Throughput | 5,000 req/s | Load test |
| Uptime | 99.9% | Monitoring |

---

## Conclusion

This implementation plan provides a comprehensive blueprint for building a production-ready, high-performance Game Stats API. The architecture leverages Go's strengths in concurrency and performance while incorporating modern best practices for security, scalability, and maintainability.

**Next Steps**:
1. Review and approve this plan
2. Set up development environment
3. Initialize project structure
4. Begin Sprint 1: Foundation & Core Models (see [BACKEND_SPRINT_1.md](./sprints/BACKEND_SPRINT_1.md))
