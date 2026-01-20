# Backend Sprint 1: Foundation & Core Models

**Duration**: 2-3 weeks
**Focus**: Project setup, database design, core models, authentication

---

## Sprint Goals

- ✅ Initialize Go project with proper structure  
- ✅ Set up PostgreSQL 17 with pgvector extension
- ✅ Design and implement Ent ORM schemas
- ✅ Create database migration system
- ✅ Implement authentication (JWT)
- ✅ Set up testing framework
- ✅ Create CI/CD pipeline

---

## Tasks

### Week 1: Project Initialization

#### Day 1-2: Project Setup
- [ ] Initialize Go module (`go mod init`)
- [ ] Create project directory structure
  - cmd/api
  - internal/{domain,application,infrastructure,presentation}
  - ent/schema
  - tests/{unit,integration}
- [ ] Set up `.gitignore`, `.env.example`
- [ ] Install core dependencies
  ```bash
  go get entgo.io/ent/cmd/ent@latest
  go get github.com/go-chi/chi/v5
  go get github.com/golang-jwt/jwt/v5
  go get github.com/spf13/viper
  go get go.uber.org/zap
  ```
- [ ] Configure Viper for environment management
- [ ] Set up structured logging with Zap

**Deliverable**: Working Go project skeleton

---

#### Day 3-5: Database Setup
- [ ] Create Docker Compose for PostgreSQL 17 + Redis
  ```yaml
  services:
    postgres:
      image: pgvector/pgvector:pg17
      ports:
        - "5432:5432"
      environment:
        POSTGRES_DB: gamestats
        POSTGRES_USER: postgres
        POSTGRES_PASSWORD: postgres
      volumes:
        - postgres_data:/var/lib/postgresql/data
  
    redis:
      image: redis:7.2-alpine
      ports:
        - "6379:6379"
  ```
- [ ] Initialize Ent (`go run -mod=mod entgo.io/ent/cmd/ent new`)
- [ ] Create Ent schemas for:
  - World, Continent, Country, Location, Field
  - Discipline, Event, DivisionPool, EventReconciliation
  - Team, Player
  - User
- [ ] Generate Ent code (`go generate ./ent`)
- [ ] Create database migration scripts
- [ ] Add UUID, pgvector, pg_trgm extensions

**Deliverable**: Complete database schema with migrations

---

### Week 2: Core Models & Repository Layer

#### Day 6-7: Geographic Hierarchy
- [ ] Implement World repository
  - CRUD operations
  - Soft delete support
- [ ] Implement Continent repository
  - CRUD with World relationship
- [ ] Implement Country repository
  - CRUD with Continent relationship
- [ ] Implement Location repository
  - CRUD with Country relationship
  - GPS coordinate support
- [ ] Implement Field repository
  - CRUD with Location relationship

**Deliverable**: Complete geographic hierarchy implementation

---

#### Day 8-9: Event Management
- [ ] Implement Discipline repository
  - CRUD with Country relationship
- [ ] Implement Event repository
  - CRUD with Discipline and Location relationships
  - Partition table by year
  - Event status management (draft, active, completed, canceled)
- [ ] Implement DivisionPool repository
  - CRUD with Event relationship
  - Ranking criteria JSON handling
- [ ] Implement EventReconciliation repository
  - Many-to-many Event relationships

**Deliverable**: Event management system

---

#### Day 10: Team & Player
- [ ] Implement Team repository
  - CRUD with DivisionPool and Location relationships
  - Unique name constraint
- [ ] Implement Player repository
  - CRUD with Team relationship
  - Gender validation
  - Fuzzy name search (pg_trgm)

**Deliverable**: Team and player management

---

### Week 3: Authentication & Testing

#### Day 11-12: User Authentication
- [ ] Design User model in Ent
  - Email, password_hash, role, managed entities
- [ ] Implement password hashing (bcrypt)
- [ ] Implement JWT token generation
  - Access token (15min expiry)
  - Refresh token (7 days expiry)
- [ ] Create Auth service
  - Login
  - Refresh
  - Logout
- [ ] Create Auth HTTP handlers
  - POST /api/v1/auth/login
  - POST /api/v1/auth/refresh
  - POST /api/v1/auth/logout
- [ ] Implement JWT middleware
  - Token validation
  - Claims extraction
  - Context injection

**Deliverable**: Working authentication system

---

#### Day 13-14: Role-Based Access Control
- [ ] Define permission matrix
  | Role | Permissions |
  |------|-------------|
  | admin | Full access |
  | event_manager | Manage assigned event |
  | team_manager | Manage team roster |
  | scorekeeper | Record scores for assigned games |
  | spectator | Read-only |
  
- [ ] Implement authorization middleware
- [ ] Create permission check functions
  ```go
  func (a *Authz) CanEditGame(userID, gameID uuid.UUID) bool
  func (a *Authz) CanManageTeam(userID, teamID uuid.UUID) bool
  ```
- [ ] Add permission guards to handlers

**Deliverable**: RBAC system

---

#### Day 15: Testing Framework
- [ ] Set up testcontainers for PostgreSQL
- [ ] Create test utilities
  - Database seeding helper
  - HTTP test helpers
  - Mock factories
- [ ] Write unit tests for:
  - Password hashing
  - JWT generation/validation
  - Repository CRUD operations
- [ ] Write integration tests for:
  - Auth endpoints
  - Geographic hierarchy
- [ ] Set up coverage reporting

**Deliverable**: Test framework with >70% coverage

---

## Definition of Done

✅ All schemas defined and code generated  
✅ Database migrations working  
✅ All repositories implemented with tests  
✅ Authentication system functional  
✅ RBAC implemented  
✅ CI pipeline passing  
✅ Code coverage >70%  
✅ Documentation updated  

---

## Testing Checklist

### Unit Tests
- [ ] User service tests
- [ ] Auth service tests (login, refresh, logout)
- [ ] Repository tests (all models)
- [ ] JWT middleware tests
- [ ] Authorization tests

### Integration Tests
- [ ] POST /auth/login with valid credentials
- [ ] POST /auth/login with invalid credentials
- [ ] POST /auth/refresh with valid token
- [ ] POST /auth/refresh with expired token
- [ ] Protected endpoint without token (401)
- [ ] Protected endpoint with valid token (200)

---

## Technical Debt / Future Improvements

- Consider Redis for session blacklisting
- Implement rate limiting for auth endpoints
- Add 2FA support
- Implement password reset flow
- Add email verification

---

## Dependencies for Next Sprint

Sprint 2 requires:
- ✅ User authentication working
- ✅ Team and Player models
- ✅ Event and DivisionPool models
- ✅ Working repository layer

---

## Notes

- Use optimistic locking for concurrent updates
- All timestamps in UTC
- Soft deletes for all entities
- UUID primary keys for security
- Index frequently queried fields

---

## Sprint Retrospective Template

### What Went Well
- 

### What Could Be Improved
- 

### Action Items
- 

---

**Next**: [Backend Sprint 2: Game Management & Scoring](./BACKEND_SPRINT_2.md)
