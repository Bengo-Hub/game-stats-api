# Game Stats API - Backend

Modern, high-performance tournament management system built with **Go 1.24+**, **Ent ORM**, and **PostgreSQL 17**.

## Features

- ⚡ **Real-time Score Updates** via Server-Sent Events (SSE)
- ⏱️ **Game Timer System** with stoppage tracking  
- 🏆 **Tournament Brackets** with automatic ranking
- 📊 **AI-Powered Analytics** with natural language queries
- 🎯 **Spirit of the Game** tracking and nominations
- 🔐 **Role-Based Access Control** with hierarchical permissions
- 📱 **RESTful API** with comprehensive OpenAPI documentation

## Tech Stack

| Component | Technology |
|-----------|------------|
| Language | Go 1.24+ |
| ORM | Ent v0.11+ |
| Database | PostgreSQL 17 + pgvector |
| Cache | Redis 7.2+ |
| Real-time | Server-Sent Events |
| Router | Chi v5 |



## Quick Start

### Prerequisites

- Go 1.24+
- Docker & Docker Compose
- PostgreSQL 17 (or use Docker)
- Redis 7.2+ (or use Docker)

### Installation

```bash
# Clone repository
git clone https://github.com/yourusername/game-stats.git
cd game-stats/games-stats-api

# Install dependencies
go mod download

# Copy environment file
cp .env.example .env
# Edit .env with your configuration

# Start dependencies (PostgreSQL + Redis)
docker-compose up -d

# Run database migrations
make migrate

# Seed sample data (optional)
make seed

# Start API server
make run
```

Server runs at `http://localhost:8080`

## Documentation

📚 **[Complete Documentation](./docs/)**

- [ERD.md](./docs/ERD.md) - Database schema and relationships
- [PLAN.md](./docs/PLAN.md) - Architecture and implementation details
- [INTEGRATIONS.md](./docs/INTEGRATIONS.md) - API contracts and third-party integrations
- [API_REFERENCE.md](./docs/API_REFERENCE.md) - Complete API documentation
- [Sprint Files](./docs/sprints/) - Development roadmap

## API Endpoints

**Authentication**
- `POST /api/v1/auth/login` - User login
- `POST /api/v1/auth/refresh` - Refresh access token

**Games**
- `GET /api/v1/games` - List games
- `GET /api/v1/games/{id}` - Get game details
- `POST /api/v1/games/{id}/start` - Start game
- `POST /api/v1/games/{id}/score` - Record score
- `GET /api/v1/games/{id}/stream` - SSE live updates

**Teams & Players**
- `GET /api/v1/teams` - List teams
- `GET /api/v1/teams/{id}/stats` - Team statistics
- `GET /api/v1/players/{id}/stats` - Player statistics

**Brackets**
- `GET /api/v1/events/{id}/bracket` - Tournament bracket

**Analytics**
- `POST /api/v1/analytics/query` - Natural language query

📖 [View Full API Reference](./docs/API_REFERENCE.md)

## Development

### Project Structure

```
internal/
├── config/          # Configuration management
├── domain/          # Domain models and interfaces
├── application/     # Business logic
├── infrastructure/  # DB, cache, external services
└── presentation/    # HTTP handlers, middleware

ent/schema/          # Ent schema definitions
docs/                # Documentation
tests/               # Tests (unit, integration, e2e)
```

### Testing

```bash
# Run unit tests
make test

# Run integration tests
make test-integration

# Run all tests with coverage
make test-coverage

# Lint code
make lint
```

### Database Migrations

```bash
# Generate new migration
make migration-new name=add_field_to_games

# Run pending migrations
make migrate

# Rollback last migration
make migrate-down
```

## Environment Variables

| Variable | Required | Description |
|----------|----------|-------------|
| `DATABASE_URL` | Yes | PostgreSQL connection string |
| `REDIS_URL` | Yes | Redis connection string |
| `JWT_SECRET` | Yes | JWT signing secret |
| `OPENAI_API_KEY` | No | For AI analytics (optional) |
| `PORT` | No | Server port (default: 8080) |

See [.env.example](./.env.example) for full configuration.

## Deployment

### Docker

```bash
# Build image
docker build -t game-stats-api .

# Run container
docker run -p 8080:8080 --env-file .env game-stats-api
```

### Kubernetes

```bash
# Apply manifests
kubectl apply -f deployments/k8s/

# Check status
kubectl get pods -l app=game-stats-api
```

## Performance Targets

| Metric | Target |
|--------|--------|
| API P95 Latency | < 100ms |
| SSE Message Delivery | < 500ms |
| Concurrent Connections | 10,000+ |
| Throughput | 5,000 req/s |

## Contributing

See [CONTRIBUTING.md](../CONTRIBUTING.md)

## License

MIT License - see [LICENSE](../LICENSE)

## Support

- 📧 Email: support@gamestats.com
- 💬 Discord: [Join Community](https://discord.gg/gamestats)
- 🐛 Issues: [GitHub Issues](https://github.com/yourusername/game-stats/issues)
