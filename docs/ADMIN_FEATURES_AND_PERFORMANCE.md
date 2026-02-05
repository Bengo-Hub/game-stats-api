# Sprint 3 Week 3: Admin Features & Performance Optimization

## Overview
This document covers the implementation of administrative features and performance optimizations for the DigiGameStats API, including Redis caching, audit trails, and admin score editing capabilities.

## Components Implemented

### 1. Redis Caching Infrastructure

**File**: `internal/infrastructure/cache/redis_client.go`

#### Features
- **Redis Client Wrapper**: Abstraction layer over go-redis/v9
- **JSON Support**: Automatic marshaling/unmarshaling for complex objects
- **Pattern-based Cache Invalidation**: Delete multiple keys matching patterns
- **TTL Constants**: Predefined expiration times for different cache types

#### Cache TTL Strategy
```go
const (
	TTLGameStats      = 2 * time.Minute   // Individual game statistics
	TTLStandings      = 5 * time.Minute   // Division standings
	TTLSpiritScores   = 5 * time.Minute   // Spirit score rankings
	TTLPlayerStats    = 5 * time.Minute   // Player performance data
	TTLBracket        = 10 * time.Minute  // Tournament brackets
	TTLEventStats     = 10 * time.Minute  // Event-level analytics
	TTLOllamaQuery    = 1 * time.Hour     // LLM query results
	TTLSupersetToken  = 4 * time.Minute   // Superset guest tokens
	TTLSchemaContext  = 24 * time.Hour    // Database schema context
)
```

#### Key Operations
```go
// Simple key-value operations
cache.Set(ctx, key, value, ttl)
cache.Get(ctx, key)
cache.Delete(ctx, key)

// JSON operations
cache.SetJSON(ctx, key, object, ttl)
cache.GetJSON(ctx, key, &object)

// Pattern operations
cache.DeletePattern(ctx, "game-stats:standings:*")

// Atomic operations
cache.Increment(ctx, key)
cache.SetNX(ctx, key, value, ttl) // Set if not exists

// Utility operations
cache.Exists(ctx, key)
cache.Expire(ctx, key, ttl)
```

#### Cache Key Pattern
```go
// Format: "game-stats:entity:id:subtype"
cache.CacheKey("game", gameID)                    // "game-stats:game:uuid"
cache.CacheKey("standings", "division", divID)    // "game-stats:standings:division:uuid"
cache.CacheKey("event", "stats", eventID, "team") // "game-stats:event:stats:uuid:team"
```

### 2. Audit Trail System

**Files**:
- `internal/domain/audit/audit_log.go` - Domain model
- `internal/infrastructure/repository/audit_repository.go` - Persistence layer

#### Audit Log Structure
```go
type AuditLog struct {
	ID         uuid.UUID              // Unique identifier
	EntityType string                  // Type of entity (game, spirit_score, etc.)
	EntityID   uuid.UUID              // ID of the modified entity
	Action     string                  // create, update, delete
	UserID     uuid.UUID              // Who made the change
	Username   string                  // User's display name
	Changes    map[string]ChangeEntry // Field-level change tracking
	Reason     string                  // Mandatory reason for change
	IPAddress  string                  // Request IP for security
	UserAgent  string                  // Client information
	CreatedAt  time.Time              // Timestamp
}

type ChangeEntry struct {
	OldValue string // Previous value
	NewValue string // New value
}
```

#### Usage Example
```go
// Create audit log
auditLog := audit.NewAuditLog(
	"game",
	gameID,
	audit.ActionUpdate,
	adminUserID,
	"admin_username",
)

// Add changes
auditLog.AddChange("home_score", "10", "12")
auditLog.AddChange("away_score", "8", "10")

// Set metadata
auditLog.SetMetadata("Score correction after official review", "127.0.0.1", "Mozilla/5.0")

// Save
createdLog, err := auditRepo.Create(ctx, auditLog)
```

#### Repository Interface
```go
type Repository interface {
	Create(ctx context.Context, log *AuditLog) (*AuditLog, error)
	GetByEntity(ctx context.Context, entityType string, entityID uuid.UUID) ([]*AuditLog, error)
	GetByUser(ctx context.Context, userID uuid.UUID, limit int) ([]*AuditLog, error)
	GetRecent(ctx context.Context, limit int) ([]*AuditLog, error)
}
```

### 3. Admin Score Editing Service

**Files**:
- `internal/application/admin/score_admin_service_v2.go` - Core service
- `internal/application/admin/cache_helper.go` - Cache invalidation utilities

#### Features
- **Game Score Updates**: Modify final scores with full audit trail
- **Spirit Score Updates**: Adjust spirit scores (component-level)
- **Mandatory Reason**: All edits require 10+ character explanation
- **Automatic Cache Invalidation**: Clears affected caches
- **Full Metadata Capture**: User, IP, timestamp, user agent

#### Game Score Update Flow
```go
// 1. Validate request
if err := req.Validate(); err != nil {
	return nil, err
}

// 2. Get current game state
currentGame, err := gameRepo.GetByIDWithRelations(ctx, req.GameID)

// 3. Track changes
changes := make(map[string]audit.ChangeEntry)
if currentGame.HomeTeamScore != req.HomeScore {
	changes["home_score"] = audit.ChangeEntry{
		OldValue: fmt.Sprintf("%d", currentGame.HomeTeamScore),
		NewValue: fmt.Sprintf("%d", req.HomeScore),
	}
}

// 4. Create audit log
auditLogID, err := CreateAuditLog(ctx, auditRepo, "game", gameID, ...)

// 5. Update scores
currentGame.HomeTeamScore = req.HomeScore
updatedGame, err := gameRepo.Update(ctx, currentGame)

// 6. Invalidate caches
InvalidateCaches(ctx, cache, gameID, updatedGame)
```

#### Cache Invalidation Strategy
```go
func InvalidateCaches(ctx context.Context, c *cache.RedisClient, gameID uuid.UUID, g *game.Game) error {
	// Delete specific game cache
	c.Delete(ctx, c.CacheKey("game", gameID.String()))
	
	// Delete division standings (pattern match)
	c.DeletePattern(ctx, c.CacheKey("standings", "division", divisionID, "*"))
	
	// Delete event statistics
	c.DeletePattern(ctx, c.CacheKey("event", "stats", eventID, "*"))
	
	return nil
}
```

### 4. Admin HTTP Handlers

**File**: `internal/presentation/http/handlers/admin_handler.go`

#### Endpoints

##### Update Game Score
```
PUT /api/v1/admin/games/{id}/score
Authorization: Bearer {admin_token}
X-User-ID: {user_id}
X-Username: {username}
X-User-Role: admin

Request Body:
{
	"home_score": 12,
	"away_score": 10,
	"reason": "Score correction after official review of game footage"
}

Response:
{
	"game_id": "uuid",
	"home_score": 12,
	"away_score": 10,
	"updated_at": "2024-01-15T10:30:00Z",
	"audit_log_id": "uuid"
}
```

##### Update Spirit Score
```
PUT /api/v1/admin/spirit-scores/{id}
Authorization: Bearer {admin_token}

Request Body:
{
	"rules_knowledge": 3,
	"fouls": 2,
	"fair_mindedness": 4,
	"attitude": 4,
	"communication": 3,
	"reason": "Correction based on observer report from tournament director"
}

Response:
{
	"spirit_score_id": "uuid",
	"rules_knowledge": 3,
	"fouls": 2,
	"fair_mindedness": 4,
	"attitude": 4,
	"communication": 3,
	"total_score": 16,
	"updated_at": "2024-01-15T10:30:00Z",
	"audit_log_id": "uuid"
}
```

##### Get Game Audit History
```
GET /api/v1/admin/games/{id}/audit
Authorization: Bearer {admin_token}

Response:
[
	{
		"id": "uuid",
		"entity_type": "game",
		"entity_id": "uuid",
		"action": "update",
		"user_id": "uuid",
		"username": "admin_user",
		"changes": {
			"home_score": {
				"old_value": "10",
				"new_value": "12"
			}
		},
		"reason": "Score correction after official review",
		"ip_address": "127.0.0.1",
		"user_agent": "Mozilla/5.0...",
		"created_at": "2024-01-15T10:30:00Z"
	}
]
```

### 5. Admin Middleware

**File**: `internal/presentation/http/middleware/admin.go`

#### AdminOnly Middleware
```go
// Ensures only admin users can access protected endpoints
func AdminOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		role, ok := r.Context().Value("user_role").(string)
		if !ok || strings.ToLower(role) != "admin" {
			http.Error(w, `{"error":"Forbidden: Admin access required"}`, 403)
			return
		}
		next.ServeHTTP(w, r)
	})
}
```

#### SetUserContext Middleware
```go
// Extracts user information from headers/JWT
func SetUserContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		
		// Extract from headers (production: from JWT claims)
		ctx = context.WithValue(ctx, "user_id", userID)
		ctx = context.WithValue(ctx, "username", username)
		ctx = context.WithValue(ctx, "user_role", role)
		
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
```

### 6. Ranking Service Caching

**File**: `internal/application/ranking/service.go`

#### Caching Implementation
```go
func (s *Service) CalculateStandings(ctx context.Context, divisionID uuid.UUID) (*DivisionStandingsResponse, error) {
	// Try cache first
	cacheKey := s.cache.CacheKey("standings", "division", divisionID.String())
	var cachedStandings DivisionStandingsResponse
	found, err := s.cache.GetJSON(ctx, cacheKey, &cachedStandings)
	if err == nil && found {
		return &cachedStandings, nil
	}

	// Calculate standings (database queries)
	// ...

	// Cache the result
	if err := s.cache.SetJSON(ctx, cacheKey, response, cache.TTLStandings); err != nil {
		fmt.Printf("Failed to cache standings: %v\n", err)
	}

	return response, nil
}
```

#### Cache Invalidation Triggers
- Game score updates (admin or regular)
- Game status changes (finished → ended)
- Spirit score modifications
- Team composition changes

## Testing

### Unit Tests

#### Redis Client Tests
**File**: `internal/infrastructure/cache/redis_client_test.go`

```bash
go test ./internal/infrastructure/cache -v
```

Tests:
- Cache key generation patterns
- TTL constant values
- Set/Get operations
- JSON marshaling/unmarshaling
- Pattern-based deletion
- Atomic operations (Increment, SetNX)

#### Admin Handler Tests
**File**: `internal/presentation/http/handlers/admin_handler_test.go`

```bash
go test ./internal/presentation/http/handlers -run TestAdminHandler -v
```

Tests:
- Successful game score update
- Invalid game ID handling
- Reason validation (minimum length)
- Audit history retrieval
- Authorization checks

### Integration Tests

#### Test Scenarios
1. **Score Update with Cache Invalidation**
   - Update game score via admin endpoint
   - Verify audit log creation
   - Confirm cache invalidation
   - Check standings recalculation

2. **Concurrent Admin Updates**
   - Multiple admins updating different games
   - Verify audit trail completeness
   - Check for race conditions

3. **Cache Performance**
   - Load 1000 games into cache
   - Measure hit/miss rates
   - Verify TTL expiration

## Configuration

### Environment Variables

```bash
# Redis Configuration
REDIS_URL=redis://localhost:6379
REDIS_PASSWORD=
REDIS_DB=0

# Superset Configuration (existing)
SUPERSET_BASE_URL=http://localhost:8088
SUPERSET_USERNAME=admin
SUPERSET_PASSWORD=admin

# Ollama Configuration (existing)
OLLAMA_BASE_URL=http://localhost:11434
OLLAMA_MODEL=duckdb-nsql:7b

# Database
DATABASE_URL=postgresql://user:pass@localhost:5432/gamestats?sslmode=disable
```

### Docker Compose Addition

```yaml
# Add to existing docker-compose.yml
services:
  redis:
    image: redis:7-alpine
    container_name: gamestats-redis
    ports:
      - "6379:6379"
    volumes:
      - redis-data:/data
    command: redis-server --appendonly yes
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 3s
      retries: 3

volumes:
  redis-data:
```

## Performance Metrics

### Expected Improvements

#### Standings Calculation
- **Before**: 150-300ms (database queries)
- **After**: 2-5ms (cache hit)
- **Cache Hit Rate**: 85-95%

#### Bracket Generation
- **Before**: 200-400ms (complex queries + calculations)
- **After**: 3-8ms (cache hit)
- **Cache Hit Rate**: 90-98%

#### LLM Queries
- **Before**: 2-5 seconds (Ollama inference)
- **After**: 10-50ms (cache hit for repeated queries)
- **Cache Hit Rate**: 40-60%

### Monitoring Metrics

```go
// TODO: Implement Prometheus metrics
metrics.CacheHitRate.WithLabelValues("standings").Inc()
metrics.CacheMissRate.WithLabelValues("standings").Inc()
metrics.APILatency.WithLabelValues("/api/v1/rankings/divisions/{id}/standings").Observe(duration)
```

## Security Considerations

### Audit Trail
- **Immutable**: Audit logs cannot be edited or deleted
- **Mandatory Reason**: All admin actions require justification
- **IP Tracking**: Record source IP for accountability
- **User Agent**: Capture client information

### Authorization
- **Role-based Access**: Only admin role can access admin endpoints
- **JWT Validation**: Verify token signature and expiration
- **Context Propagation**: User info carried through request lifecycle

### Cache Security
- **No Sensitive Data**: Tokens and credentials not cached (or very short TTL)
- **Redis Authentication**: Use password in production
- **Network Isolation**: Redis on private network only

## Known Limitations

### Current Implementation
1. **In-Memory Audit Repository**: Production requires Ent schema migration
2. **Spirit Score Updates**: Placeholder - needs spirit score repository
3. **No Pagination**: Audit history returns all logs (add limit in production)
4. **Cache Invalidation**: Manual - no automatic triggers on game updates yet

### TODO Items
1. Create Ent schema for audit_logs table
2. Implement spirit score repository and update logic
3. Add pagination to audit history endpoints
4. Wire admin routes into main router
5. Add Prometheus metrics for cache performance
6. Implement distributed locking for concurrent updates
7. Add database indexes for frequently queried fields
8. Create Grafana dashboards for monitoring

## Migration Guide

### Step 1: Create Audit Log Schema
```sql
CREATE TABLE audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    entity_type VARCHAR(50) NOT NULL,
    entity_id UUID NOT NULL,
    action VARCHAR(20) NOT NULL,
    user_id UUID NOT NULL,
    username VARCHAR(100) NOT NULL,
    changes JSONB NOT NULL,
    reason TEXT NOT NULL,
    ip_address VARCHAR(45),
    user_agent TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_audit_logs_entity ON audit_logs(entity_type, entity_id);
CREATE INDEX idx_audit_logs_user ON audit_logs(user_id);
CREATE INDEX idx_audit_logs_created ON audit_logs(created_at DESC);
```

### Step 2: Update Ent Schema
```go
// Add to ent/schema/audit_log.go
func (AuditLog) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(uuid.New),
		field.String("entity_type").NotEmpty(),
		field.UUID("entity_id", uuid.UUID{}),
		field.String("action").NotEmpty(),
		field.UUID("user_id", uuid.UUID{}),
		field.String("username").NotEmpty(),
		field.JSON("changes", map[string]interface{}{}),
		field.Text("reason").NotEmpty(),
		field.String("ip_address").Optional(),
		field.Text("user_agent").Optional(),
		field.Time("created_at").Default(time.Now),
	}
}
```

### Step 3: Initialize Redis in main.go
```go
// Add Redis client initialization
redisClient, err := cache.NewRedisClient(cfg.RedisURL, cfg.RedisPassword, cfg.RedisDB)
if err != nil {
	logger.Fatal("Failed to connect to Redis", logger.Err(err))
}
defer redisClient.Close()
```

### Step 4: Update Service Constructors
```go
// Add cache parameter to ranking service
rankingService := ranking.NewService(
	divisionRepo,
	gameRepo,
	teamRepo,
	gameRoundRepo,
	redisClient, // NEW
)
```

## Appendix

### Dependencies Added
```
github.com/redis/go-redis/v9 v9.17.3
```

### File Structure
```
game-stats-api/
├── internal/
│   ├── application/
│   │   ├── admin/
│   │   │   ├── score_admin_service_v2.go
│   │   │   └── cache_helper.go
│   │   └── ranking/
│   │       └── service.go (updated with caching)
│   ├── domain/
│   │   └── audit/
│   │       └── audit_log.go
│   ├── infrastructure/
│   │   ├── cache/
│   │   │   ├── redis_client.go
│   │   │   └── redis_client_test.go
│   │   └── repository/
│   │       └── audit_repository.go
│   └── presentation/
│       └── http/
│           ├── handlers/
│           │   ├── admin_handler.go
│           │   └── admin_handler_test.go
│           └── middleware/
│               └── admin.go
```

### Related Documentation
- Sprint 2: SPIRIT_SCORE_API.md
- Sprint 3 Week 1: RANKING_AND_BRACKET_SYSTEM.md
- Sprint 3 Week 2: SUPERSET_INTEGRATION.md, OLLAMA_DEPLOYMENT.md
- Database: docs/erd.md
