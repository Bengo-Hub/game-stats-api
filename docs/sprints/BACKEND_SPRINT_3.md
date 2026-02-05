# Backend Sprint 3: Advanced Features & Analytics

**Duration**: 2-3 weeks
**Focus**: Auto-ranking, brackets, analytics platform, AI-powered queries

---

## Sprint Goals

- ✅ Implement automatic ranking and round progression
- ✅ Build tournament bracket generation and visualization
- ✅ Integrate Metabase for embedded analytics
- ✅ Implement Ollama text-to-SQL with pgvector
- ✅ Create admin score editing with audit trail
- ✅ Build analytics dashboard API

---

## Tasks

### Week 1: Auto-Ranking & Round Progression

#### Day 1-3: Ranking System
- [x] Extend DivisionPool model with ranking_criteria JSONB
- [x] Implement ranking calculation service:
  ```go
  func (s *Service) CalculateStandings(ctx context.Context, divisionID uuid.UUID) (*DivisionStandingsResponse, error)
  ```
  - Parse ranking criteria (points, goal_diff, head_to_head)
  - Calculate team statistics
  - Apply sorting rules
  - Cache results in Redis (pending)
- [x] Build auto-advancement logic:
  ```go
  func (s *Service) AdvanceTeams(ctx context.Context, req AdvanceTeamsRequest) (*AdvanceTeamsResponse, error)
  ```
  - Check pool phase completion
  - Advance top N teams
  - Create bracket round games (pending bracket generation)
  - Send notifications (pending)
- [x] Create ranking API:
  - `GET /api/v1/divisions/{id}/standings` - Current standings
  - `POST /api/v1/divisions/advance` - Trigger advancement
  - `PUT /api/v1/divisions/{id}/criteria` - Update ranking rules
- [ ] Add materialized view for team stats

**Deliverable**: Auto-ranking system ✅

---

#### Day 4-5: Brackets Generation
- [x] Implement bracket generation algorithm:
  - Single elimination
  - Double elimination (optional)
  - Seeding based on pool standings
- [x] Create BracketNode service for tree structure
- [x] Build bracket API:
  - `GET /api/v1/events/{id}/bracket` - Get bracket JSON
  - `POST /api/v1/events/{id}/generate-bracket` - Generate from pools
  - `GET /api/v1/rounds/{id}/bracket` - Get bracket for specific round
- [x] Add bracket visualization data format
- [x] Integrate bracket generation with AdvanceTeams service

**Deliverable**: Bracket generation ✅

---

### Week 2: Analytics Platform Integration

#### Day 6-8: Superset/Metabase Setup
- [ ] Deploy Superset/Metabase via Docker (infrastructure)
- [ ] Configure connection to PostgreSQL (infrastructure)
- [ ] Create base dashboards (infrastructure)
- [x] Implement embed token generation service:
  - `analytics.Service.GenerateEmbedToken()` complete
  - `analytics.Service.ListDashboards()` complete
  - `analytics.Service.GetDashboard()` complete
  - `analytics.Service.GetEventStatistics()` complete with real queries
  - Row-level security (RLS) implementation complete
- [x] Build dashboard management API:
  ```go
  func (s *AnalyticsService) GenerateEmbedToken(dashboardID int, userID uuid.UUID) (string, error)
  ```
- [ ] Build dashboard management API:
  - `GET /api/v1/analytics/dashboards` - List dashboards
  - `POST /api/v1/analytics/embed-token/:id` - Get embed token
- [ ] Add row-level security params

**Deliverable**: Embedded Metabase dashboards

---

#### Day 9-11: Ollama LLM Integration
- [ ] Deploy Ollama container with duckdb-nsql:7b (infrastructure)
- [x] Implement Ollama HTTP client (`ollama_client.go`)
- [x] Create text-to-SQL service (`text_to_sql_service.go`):
  - `OllamaClient.GenerateSQL()` - SQL generation via LLM
  - `OllamaClient.Chat()` - General chat capabilities
  - `OllamaClient.HealthCheck()` - Service availability check
  - Build schema context dynamically
  - Parse SQL + explanation response
  - SQL injection prevention (validateSQL)
  - Row limit enforcement (1000 rows max)
- [x] Build natural language query API:
  - `POST /api/v1/analytics/query` - NL to SQL with execution
  - Row-level security filtering applied automatically

**Deliverable**: AI text-to-SQL system ✅ (code complete, needs Ollama deployment)

---

#### Day 12: Analytics Query Execution
- [x] Create safe SQL execution service:
  - Block dangerous keywords (DELETE, UPDATE, DROP, INSERT, etc.)
  - Only SELECT/WITH queries allowed
  - Row limit enforcement (1000 rows)
  - Timeout protection via context
- [x] Execute raw SQL queries against PostgreSQL
- [ ] Implement query result caching (future enhancement)
- [ ] Add query history tracking (future enhancement)

**Deliverable**: Safe analytics execution ✅

---

### Week 3: Admin Features & Optimization

#### Day 13-14: Admin Score Editing
- [x] Create ScoreEdit audit model (`ent/schema/audit_log.go`, `internal/domain/audit/audit_log.go`)
- [x] Implement edit score service (`internal/application/admin/score_admin_service.go`):
  - Field-level change tracking
  - Mandatory reason (min 10 chars)
  - User metadata (ID, username, IP, user agent)
  - Automatic cache invalidation
- [x] Build admin API:
  - `PUT /api/v1/admin/games/{id}/score` - Edit game score
  - `GET /api/v1/admin/games/{id}/audit` - View audit trail
  - `PUT /api/v1/admin/spirit-scores/{id}` - Edit spirit score
- [x] Admin-only middleware (`internal/presentation/http/middleware/admin.go`)

**Deliverable**: Score editing with audit ✅

---

#### Day 15: Performance Optimization
- [ ] Implement Redis caching:
  - Team standings (5min TTL)
  - Player stats (5min TTL)
  - Game lists (1min TTL)
  - Bracket data (1min TTL)
- [ ] Add database indexes:
  - Composite indexes for common queries
  - Partial indexes for active games
- [ ] Optimize N+1 queries with eager loading
- [ ] Add query performance monitoring
- [ ] Implement connection pooling tuning
- [ ] Run load tests:
  - 1000 concurrent SSE connections
  - 100 req/s on scoring endpoints
  - Analytics query performance

**Deliverable**: Optimized performance

---

## Definition of Done

✅ Auto-ranking with configurable criteria  
✅ Bracket generation and visualization  
✅ Metabase embedded dashboards  
✅ Ollama text-to-SQL working  
✅ pgvector semantic search  
✅ Admin score editing with audit  
✅ Performance targets met  
✅ Tests passing with >75% coverage  

---

## Performance Targets

| Metric | Target | Test Method |
|--------|--------|-------------|
| Standings calculation | < 100ms | Benchmark |
| Bracket generation | < 500ms | Integration test |
| Analytics query | < 5s | E2E test |
| Text-to-SQL | < 3s | Integration test |
| SSE concurrent clients | 1000+ | Load test |

---

## Environment Variables

```bash
# Metabase
METABASE_URL=http://localhost:3000
METABASE_SECRET_KEY=your-secret-key-here
METABASE_SITE_URL=https://yourdomain.com

# Ollama
OLLAMA_URL=http://localhost:11434
OLLAMA_MODEL=duckdb-nsql:7b
OLLAMA_TIMEOUT=30s

# OpenAI (for embeddings, optional)
OPENAI_API_KEY=sk-...
OPENAI_EMBEDDING_MODEL=text-embedding-ada-002
```

---

## Docker Compose Additions

```yaml
services:
  metabase:
    image: metabase/metabase:latest
    ports:
      - "3000:3000"
    environment:
      MB_DB_TYPE: postgres
      MB_DB_DBNAME: metabase
      MB_DB_PORT: 5432
      MB_DB_USER: metabase
      MB_DB_PASS: metabase
      MB_DB_HOST: postgres
    depends_on:
      - postgres
  
  ollama:
    image: ollama/ollama:latest
    ports:
      - "11434:11434"
    volumes:
      - ollama-data:/root/.ollama
    deploy:
      resources:
        limits:
          memory: 8GB
```

---

**Next**: [Backend Sprint 4: Production Readiness](./BACKEND_SPRINT_4.md)
