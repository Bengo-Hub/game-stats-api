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
- [ ] Extend DivisionPool model with ranking_criteria JSONB
- [ ] Implement ranking calculation service:
  ```go
  func (s *DivisionService) CalculateStandings(ctx context.Context, divisionID uuid.UUID) ([]TeamStanding, error)
  ```
  - Parse ranking criteria (points, goal_diff, head_to_head)
  - Calculate team statistics
  - Apply sorting rules
  - Cache results in Redis
- [ ] Build auto-advancement logic:
  ```go
  func (s *DivisionService) AdvanceTeams(ctx context.Context, divisionID uuid.UUID, topN int) error
  ```
  - Check pool phase completion
  - Advance top N teams
  - Create bracket round games
  - Send notifications
- [ ] Create ranking API:
  - `GET /api/v1/divisions/{id}/standings` - Current standings
  - `POST /api/v1/divisions/{id}/advance` - Trigger advancement
  - `PUT /api/v1/divisions/{id}/criteria` - Update ranking rules
- [ ] Add materialized view for team stats

**Deliverable**: Auto-ranking system

---

#### Day 4-5: Brackets Generation
- [ ] Implement bracket generation algorithm:
  - Single elimination
  - Double elimination (optional)
  - Seeding based on pool standings
- [ ] Create BracketNode service for tree structure
- [ ] Build bracket API:
  - `GET /api/v1/events/{id}/bracket` - Get bracket JSON
  - `POST /api/v1/events/{id}/generate-bracket` - Generate from pools
- [ ] Add bracket visualization data format

**Deliverable**: Bracket generation

---

### Week 2: Analytics Platform Integration

#### Day 6-8: Metabase Setup
- [ ] Deploy Metabase via Docker
- [ ] Configure Metabase connection to PostgreSQL
- [ ] Create base dashboards:
  - Event Overview (games, teams, scores)
  - Player Statistics (top scorers, assists)
  - Spirit Scores Leaderboard
  - Team Performance
- [ ] Enable embedding in Metabase
- [ ] Implement embed token generation:
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
- [ ] Deploy Ollama container with duckdb-nsql:7b
- [ ] Implement Ollama HTTP client
- [ ] Create text-to-SQL service:
  ```go
  type TextToSQLService struct {
      ollamaClient *OllamaClient
      vectorStore  *VectorStore
  }
  
  func (s *TextToSQLService) GenerateSQL(ctx context.Context, query string) (*SQLResult, error)
  ```
  - Build schema context from pgvector
  - Call Ollama API with prompt
  - Parse SQL + chart type response
  - Validate SQL (prevent injection)
  - Execute with row limit
- [ ] Implement schema embeddings:
  - Pre-compute table/column descriptions
  - Store in analytics_embeddings table
  - Update on schema changes
- [ ] Build natural language query API:
  - `POST /api/v1/analytics/query` - NL to SQL
  - `GET /api/v1/analytics/schema` - Get schema docs

**Deliverable**: AI text-to-SQL system

---

#### Day 12: Analytics Query Execution
- [ ] Create safe SQL execution service:
  - Whitelist allowed tables
  - Block DELETE/UPDATE/DROP
  - Enforce row limits
  - Timeout protection
- [ ] Implement query result caching
- [ ] Add query history tracking
- [ ] Build saved query management

**Deliverable**: Safe analytics execution

---

### Week 3: Admin Features & Optimization

#### Day 13-14: Admin Score Editing
- [ ] Create ScoreEdit audit model
- [ ] Implement edit score service:
  ```go
  func (s *AdminService) EditScore(ctx context.Context, req EditScoreRequest) error
  ```
  - Verify admin permissions
  - Create audit record (old value, new value, reason, editor)
  - Update scoring record
  - Recalculate game totals
  - Broadcast SSE update
  - Send notification to affected teams
- [ ] Build admin API:
  - `PUT /api/v1/admin/scores/{id}` - Edit score
  - `GET /api/v1/admin/audit/scores` - View audit trail
  - `POST /api/v1/admin/games/{id}/recalculate` - Recalc totals
- [ ] Add admin dashboard

**Deliverable**: Score editing with audit

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
