# DigiGameStats API Security Documentation

## Overview

This document outlines the security measures implemented in the DigiGameStats API to protect against common vulnerabilities and ensure secure access to resources.

## Authentication

### JWT-Based Authentication

The API uses JSON Web Tokens (JWT) for authentication:

- **Access Tokens**: Short-lived (15 minutes) tokens for API access
- **Refresh Tokens**: Long-lived (7 days) tokens for obtaining new access tokens
- **Algorithm**: HS256 (HMAC-SHA256)

#### Token Structure

```json
{
  "user_id": "uuid",
  "role": "string",
  "exp": "timestamp",
  "iat": "timestamp",
  "nbf": "timestamp"
}
```

### Password Security

- Passwords are hashed using **bcrypt** with default cost factor
- Plain text passwords are never stored or logged
- Password validation enforces minimum requirements

## Authorization (RBAC)

### Role Hierarchy

| Role | Level | Description |
|------|-------|-------------|
| admin | 100 | Full system access |
| event_manager | 80 | Manage events, tournaments, games |
| team_manager | 60 | Manage team roster and players |
| scorekeeper | 40 | Record scores, submit spirit |
| spectator | 20 | View-only access |

### Permission Matrix

| Permission | Admin | Event Manager | Team Manager | Scorekeeper | Spectator |
|------------|-------|---------------|--------------|-------------|-----------|
| view_dashboard | ✓ | ✓ | ✓ | ✓ | ✓ |
| view_events | ✓ | ✓ | ✓ | ✓ | ✓ |
| add_events | ✓ | ✓ | | | |
| change_events | ✓ | ✓ | | | |
| delete_events | ✓ | | | | |
| manage_events | ✓ | ✓ | | | |
| view_games | ✓ | ✓ | ✓ | ✓ | ✓ |
| add_games | ✓ | ✓ | | | |
| change_games | ✓ | ✓ | | | |
| delete_games | ✓ | | | | |
| record_scores | ✓ | ✓ | | ✓ | |
| view_teams | ✓ | ✓ | ✓ | ✓ | ✓ |
| manage_teams | ✓ | ✓ | | | |
| view_players | ✓ | ✓ | ✓ | ✓ | ✓ |
| manage_players | ✓ | ✓ | ✓ | | |
| submit_spirit | ✓ | ✓ | ✓ | ✓ | |
| view_analytics | ✓ | ✓ | ✓ | | ✓ |
| view_admin | ✓ | | | | |
| manage_users | ✓ | | | | |

## Rate Limiting

### Rate Limit Tiers

| Tier | Requests/Minute | Burst | Applied To |
|------|-----------------|-------|------------|
| Default | 100 | 20 | All endpoints |
| Auth | 10 | 5 | Login, Refresh |
| Public | 60 | 30 | Public API endpoints |

### Rate Limit Headers

Responses include rate limit information:
- `X-RateLimit-Limit`: Maximum requests allowed
- `X-RateLimit-Remaining`: Requests remaining in window
- `Retry-After`: Seconds until rate limit resets (when exceeded)

### Rate Limit Response

When rate limited, the API returns:
```json
{
  "error": "Rate limit exceeded. Please try again later."
}
```
Status: `429 Too Many Requests`

## API Security

### Public Endpoints

These endpoints require no authentication and are rate-limited:

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/health` | GET | Health check |
| `/api/v1/auth/login` | POST | User login |
| `/api/v1/auth/refresh` | POST | Token refresh |
| `/api/v1/public/*` | GET | Public read-only data |
| `/api/v1/geographic/*` | GET | Geographic metadata |

### Protected Endpoints

All other endpoints require:
1. Valid JWT in `Authorization: Bearer <token>` header
2. Appropriate role/permission for the operation

### CORS Configuration

```go
cors.Options{
    AllowedOrigins:   []string{"*"},  // Configure for production
    AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
    AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
    ExposedHeaders:   []string{"Link", "X-Total-Count"},
    AllowCredentials: true,
    MaxAge:           300,
}
```

**Production Recommendation**: Restrict `AllowedOrigins` to specific domains.

## Security Best Practices

### Input Validation

- All input is validated before processing
- UUID parameters are parsed and validated
- Request body size limits are enforced

### Error Handling

- Internal errors are logged but not exposed to clients
- Generic error messages prevent information leakage
- Stack traces are never returned in responses

### Logging

- Authentication failures are logged
- Rate limit violations are logged
- Sensitive data (passwords, tokens) are never logged

### Headers

Security headers are set on responses:
- `Content-Type: application/json`
- `Cache-Control: no-store` (for authenticated endpoints)

## Swagger/OpenAPI Security

The API uses `BearerAuth` security definition:

```yaml
securityDefinitions:
  BearerAuth:
    type: apiKey
    in: header
    name: Authorization
```

### Using Swagger UI

1. Call `/api/v1/auth/login` with credentials
2. Copy the `access_token` from response
3. Click "Authorize" in Swagger UI
4. Enter: `Bearer <access_token>`
5. All subsequent requests will include the token

## Vulnerability Mitigation

### OWASP Top 10 Protections

| Vulnerability | Mitigation |
|---------------|------------|
| Injection | Parameterized queries via Ent ORM |
| Broken Auth | JWT with short expiry, bcrypt passwords |
| Sensitive Data | HTTPS required, no secrets in logs |
| XXE | No XML parsing |
| Broken Access Control | RBAC with permission middleware |
| Security Misconfiguration | Environment-based config |
| XSS | JSON-only API, no HTML rendering |
| Insecure Deserialization | Standard JSON parsing |
| Known Vulnerabilities | Regular dependency updates |
| Insufficient Logging | Comprehensive audit logging |

## Environment Variables

Required security-related configuration:

| Variable | Description | Example |
|----------|-------------|---------|
| JWT_SECRET | Secret for signing JWTs | 32+ character random string |
| DATABASE_URL | PostgreSQL connection | `postgres://user:pass@host/db?sslmode=require` |
| REDIS_URL | Redis connection | `redis://:password@host:6379/0` |
| ENV | Environment mode | `production` |

## Incident Response

### Suspicious Activity

1. Unusual rate limit violations
2. Multiple failed login attempts
3. Access attempts to unauthorized resources

### Response Steps

1. Review audit logs
2. Block suspicious IPs if necessary
3. Rotate JWT secret if compromised
4. Force password resets if needed

## Updates and Maintenance

- Regularly update dependencies for security patches
- Review and update RBAC permissions as needed
- Monitor security advisories for used packages
- Conduct periodic security audits
