# Backend Sprint 1: Foundation & Core Models

**Status**: ✅ Completed
**Focus**: Project setup, database design, core models, authentication, infrastructure

---

## Sprint Goals

- ✅ Initialize Go project with proper structure  
- ✅ Set up PostgreSQL 17 with pgvector extension (Shared Infrastructure)
- ✅ Set up Redis 7.2 (Shared Infrastructure)
- ✅ Design and implement Ent ORM schemas for all core entities
- ✅ Create layered architecture (Domain, Infrastructure, Application, Presentation)
- ✅ Implement full Repository layer for all Sprint 1 entities
- ✅ Implement authentication (JWT) with RBAC
- ✅ Generate Comprehensive API Documentation (Swagger)

---

## Tasks

### Week 1: Project Initialization

#### Day 1-2: Project Setup
- [x] Initialize Go module (`go mod init`)
- [x] Create project directory structure
- [x] Set up `.gitignore`, `.env.example`
- [x] Install core dependencies (Ent, Chi, JWT, Viper, Zap)
- [x] Configure Viper for environment management
- [x] Set up structured logging with Zap

**Deliverable**: Working Go project skeleton

#### Day 3-5: Database Setup
- [x] Create Docker Compose for PostgreSQL 17 + Redis (Shared infrastructure folder)
- [x] Initialize Ent and create schemas for:
  - World, Continent, Country, Location, Field
  - Discipline, Event, DivisionPool, EventReconciliation
  - Team, Player
  - User
  - Scoring, SpiritScore, MVP/Spirit Nominations, AnalyticsEmbedding
- [x] Generate Ent code (`go generate ./ent`)
- [x] Implement database connection with retry logic

**Deliverable**: Complete database schema and infrastructure

---

### Week 2: Core Models & Repository Layer

#### Day 6-7: Geographic Hierarchy
- [x] Implement World repository (CRUD, Soft Delete)
- [x] Implement Continent repository (CRUD, World Relationship)
- [x] Implement Country repository (CRUD, Continent Relationship)
- [x] Implement Location repository (CRUD, GPS support)
- [x] Implement Field repository (CRUD, Location Relationship)

#### Day 8-9: Event Management
- [x] Implement Discipline repository
- [x] Implement Event repository (CRUD, Status Management)
- [x] Implement DivisionPool repository (JSON handling)
- [x] Implement EventReconciliation repository

#### Day 10: Team & Player
- [x] Implement Team repository (Unique Constraints)
- [x] Implement Player repository (CRUD)
- [x] Implement Scoring & SpiritScore repositories
- [x] Implement Nominations repositories (MVP/Spirit)

**Deliverable**: Complete foundation repository layer

---

### Week 3: Authentication & API Docs

#### Day 11-12: User Authentication
- [x] Implement password hashing (bcrypt)
- [x] Implement JWT token generation (Access/Refresh)
- [x] Create Auth service (Login/Refresh)
- [x] Create Auth HTTP handlers
- [x] Implement JWT middleware

#### Day 13-14: API Documentation
- [x] Install Swaggo/swag
- [x] Add Swagger annotations to handlers and main entry points
- [x] Resolve library version conflicts
- [x] Register Swagger UI in router
- [x] Generate docs via `swag init`

**Deliverable**: Secure API with interactive documentation

---

## Definition of Done

✅ All schemas defined and code generated  
✅ Database infrastructure (PQ/Redis) operational  
✅ All repositories implemented and instantiated in main  
✅ Authentication system functional (Login/Refresh)  
✅ API documentation generated and accessible  
✅ Build success (`go build ./cmd/api/main.go`)

---

## Notes
- Shared infrastructure moved to `/shared/infrastructure` to support global BengoBox ecosystem.
- PostgreSQL 17 mapped to host port 5433 to avoid local conflicts.
- Redis 7.2 mapped to host port 6380.
- All primary keys use UUIDs for enhanced security and scalability.

---

**Next**: [Backend Sprint 2: Game Management & Scoring](./BACKEND_SPRINT_2.md)
