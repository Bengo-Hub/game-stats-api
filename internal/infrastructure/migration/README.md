# Data Migration from Legacy Django System

This directory contains Go-based data migration scripts that migrate data from the legacy Django-based game stats system to the new Go/Ent-based system.

## Overview

The migration system:
- **Idempotent**: Can be run multiple times safely without duplicating data
- **Automatic**: Runs on application startup (configurable)
- **Comprehensive**: Migrates all entities with proper relationship mapping
- **Error-tolerant**: Logs errors but continues processing
- **Thread-safe**: ID mappings use mutex locks for concurrent access

## Architecture

```
migration/
├── migration.go        # Main Migrator struct and RunAll orchestration
├── id_mapping.go       # Thread-safe legacy ID → UUID mapping
├── fixture_loader.go   # JSON fixture loading and validation
├── geography.go        # Continents, countries, locations
├── events.go           # Disciplines, events, divisions, fields, rounds
├── teams_players.go    # Teams and players
├── games_scoring.go    # Games and scoring records
├── spirit_scores.go    # Spirit scores with nominations
└── README.md           # This documentation
```

## Migrated Entities

Migration order (respects foreign key dependencies):

1. **Geography** (`geography.go`)
   - Continents (`core_continent.json`)
   - Countries (`core_country.json`)
   - Locations (`core_location.json`)

2. **Events** (`events.go`)
   - Disciplines (`events_discipline.json`)
   - Events (`events_event.json`)
   - Division Pools (`events_divisionpool.json`)
   - Fields (`core_field.json`)
   - Game Rounds (`games_gameround.json`)

3. **Teams & Players** (`teams_players.go`)
   - Teams (`games_team.json`)
   - Players (`games_player.json`)

4. **Games & Scoring** (`games_scoring.go`)
   - Games (`games_game.json`)
   - Scoring records (`games_scoring.json`)

5. **Spirit Scores** (`spirit_scores.go`)
   - Spirit scores (`games_spiritscore.json`)
   - MVP nominations (male/female)
   - Spirit nominations (male/female)
   - System user for migration (auto-created)

## Field Mapping

### Django → Ent Schema Mapping

#### Teams
```
Django (games.team)          → Ent (Team)
- id                         → UUID (generated)
- name                       → name
- initial_seed               → initial_seed
- origin (FK)                → (ignored - not in new schema)
```

#### Players
```
Django (games.player)        → Ent (Player)
- id                         → UUID (generated)
- name                       → name
- gender                     → gender
- team (FK)                  → team (relationship)
- mvp_nominations            → (tracked via MVPNomination entity)
- spirit_award_nominations   → (tracked via SpiritNomination entity)
```

#### Games
```
Django (games.game)          → Ent (Game)
- id                         → UUID (generated)
- name                       → name
- date/start_time            → scheduled_time
- home_team/team1 (FK)       → home_team (relationship) [REQUIRED]
- away_team/team2 (FK)       → away_team (relationship) [REQUIRED]
- home_team_score/team1_score→ home_team_score
- away_team_score/team2_score→ away_team_score
- pool/division_pool (FK)    → division_pool (relationship) [REQUIRED]
- field (FK)                 → field_location (relationship) [REQUIRED]
- game_round (FK)            → game_round (relationship) [optional]
- (new)                      → allocated_time_minutes (default: 60)
- status                     → "completed" (default for legacy)
```

**Note**: The Ent schema uses `field_location` edge name (not `field`) and requires both `division_pool` and `field_location` relationships. Migration uses fallback to first available entity if lookup fails.

#### Scoring
```
Django (games.scoring)       → Ent (Scoring)
- id                         → UUID (generated)
- game (FK)                  → game (relationship)
- player (FK)                → player (relationship)
- goals                      → goals
- assists                    → assists
- blocks                     → blocks (mapped from Django)
- turns                      → turns (default 0)
- callahans                  → (removed - not in new schema)
```

#### Spirit Scores
```
Django (games.spiritscore)   → Ent (SpiritScore)
- id                         → UUID (generated)
- game (FK)                  → game (relationship) [REQUIRED]
- team (FK)                  → team (relationship) [REQUIRED]
- scored_by (FK)             → scored_by_team (relationship) [REQUIRED]
- (new)                      → submitted_by (relationship) [REQUIRED, uses system user]
- rules_knowledge            → rules_knowledge
- fouls_body_contact         → fouls_body_contact
- fair_mindedness            → fair_mindedness
- attitude                   → attitude
- communication              → communication
```

#### MVP Nominations (created from SpiritScore)
```
Django (embedded in spiritscore) → Ent (MVP_Nomination)
- mvp_female_nomination (FK)     → player + category="female"
- mvp_male_nomination (FK)       → player + category="male"
```

#### Spirit Nominations (created from SpiritScore)
```
Django (embedded in spiritscore)    → Ent (SpiritNomination)
- spirit_female_nomination (FK)     → player + category="female"
- spirit_male_nomination (FK)       → player + category="male"
```

**Note**: The `submitted_by` field is required in the new schema. Migration creates a disabled system user (`migration@system.local`) for this purpose.

## Configuration

### Environment Variables

```bash
# Enable/disable migration on startup
RUN_MIGRATION=true

# Path to Django JSON fixtures
FIXTURES_DIR=./scripts/fixtures
```

### Disable Migration

To skip migration on startup:

```bash
RUN_MIGRATION=false
```

Or in code:
```go
cfg.RunMigration = false
```

## Running Migration

### Automatic (on startup)

Migration runs automatically when the application starts if `RUN_MIGRATION=true`.

```bash
# Start the application
go run ./cmd/api
```

### Manual

You can also run migration manually:

```go
import "github.com/bengobox/game-stats-api/internal/infrastructure/migration"

migrator := migration.NewMigrator(client)
ctx := context.Background()

if err := migrator.RunAll(ctx, "./scripts/fixtures"); err != nil {
    log.Fatal(err)
}
```

## Data Sources

### Legacy Django Fixtures

The migration reads Django JSON fixture files from `scripts/fixtures/`:

```
scripts/fixtures/
├── core_continent.json       # Continents
├── core_country.json         # Countries
├── core_location.json        # Event locations
├── core_field.json           # Playing fields
├── events_discipline.json    # Sports disciplines
├── events_event.json         # Tournament events
├── events_divisionpool.json  # Division pools
├── games_gameround.json      # Tournament rounds
├── games_team.json           # Teams (10 teams)
├── games_player.json         # Players (1960 records)
├── games_game.json           # Games (561 games)
├── games_scoring.json        # Scoring records (4792 records)
└── games_spiritscore.json    # Spirit scores (1244 records)
```

**Important**: Note that division pools are in `events_divisionpool.json` (not `games_divisionpool.json`) and fields are in `core_field.json` (not `games_field.json`).

### Django Fixture Format

Example fixture structure:

```json
[
  {
    "model": "games.team",
    "pk": 1,
    "fields": {
      "name": "KEMRI Sharks",
      "initial_seed": 0,
      "origin": 2
    }
  }
]
```

## Idempotency

The migration is idempotent through several mechanisms:

1. **Slug-based checking**: Entities with slugs are checked by slug before creation
2. **Name-based checking**: Entities are checked by name to prevent duplicates
3. **Relationship mapping**: Legacy integer IDs are mapped to UUIDs in memory during migration
4. **Skip existing**: If an entity already exists, it's skipped without error

Example:
```go
exists, err := m.client.Team.Query().
    Where(team.Name(name)).
    Exist(ctx)
if exists {
    continue // Skip, already migrated
}
```

## Legacy ID Mapping

Since the new system uses UUIDs instead of integer IDs, the migration maintains thread-safe in-memory mappings via the `IDMapping` struct:

```go
type IDMapping struct {
    mu          sync.RWMutex
    Teams       map[int]uuid.UUID
    Players     map[int]uuid.UUID
    Games       map[int]uuid.UUID
    GameRounds  map[int]uuid.UUID
    Fields      map[int]uuid.UUID
    Divisions   map[int]uuid.UUID
    Locations   map[int]uuid.UUID
    Countries   map[int]uuid.UUID
    Continents  map[int]uuid.UUID
    Disciplines map[int]uuid.UUID
    Events      map[int]uuid.UUID
    Users       map[int]uuid.UUID
}
```

### Usage

```go
// Store mapping
m.idMapping.SetTeam(legacyID, newTeam.ID)

// Retrieve mapping
if teamUUID, ok := m.idMapping.GetTeam(legacyID); ok {
    // Use teamUUID
}
```

### Thread Safety

All mapping operations use read/write mutex locks, making them safe for concurrent access if needed.

## Fixture Loader

The `FixtureLoader` utility (`fixture_loader.go`) replaces the Python scripts for loading and validating fixtures:

```go
loader := NewFixtureLoader("./scripts/fixtures")

// Load and validate a fixture file
fixtures, err := loader.LoadAndValidate("games_team.json")

// Load games with normalized field names
games, err := loader.LoadGamesNormalized()

// Validate all fixtures
err := loader.ValidateFixtures()

// Generate unified seed data
seedData, err := loader.GenerateUnifiedSeedData()
```

### Features

- **Datetime normalization**: Handles multiple datetime formats (`2023-01-15T14:30:00Z`, `2023-01-15 14:30:00`, etc.)
- **Boolean normalization**: Converts string booleans (`"true"`, `"1"`) to proper booleans
- **Field name normalization**: Handles legacy field name variations (e.g., `team1`/`home_team`)
- **Validation**: Ensures required fields exist and fixture structure is correct

## Error Handling

The migration is error-tolerant:

- **Non-fatal errors**: Logged but don't stop migration
- **Missing references**: Skipped with warning
- **Partial migration**: Individual entity failures don't block others
- **Graceful degradation**: Application starts even if migration fails

## Logging

Migration progress is logged at each stage:

```
INFO: Starting data migration from legacy system...
INFO: Migrating continents...
✓ continents migrated successfully
INFO: Migrating countries...
✓ countries migrated successfully
...
INFO: All migrations completed successfully!
```

## Performance

Expected migration times (local development):
- 10 teams: < 1s
- 1960 players: ~2-3s
- 561 games: ~1-2s
- 4792 scoring records: ~5-10s
- 1244 spirit scores: ~3-5s

**Total: ~15-25 seconds** for full legacy dataset

## Testing

Test migration locally:

```bash
# 1. Reset database
psql -U postgres -d postgres -c "DROP SCHEMA public CASCADE; CREATE SCHEMA public;"

# 2. Run application (migration will run)
RUN_MIGRATION=true go run ./cmd/api

# 3. Verify data
psql -U postgres -d postgres -c "SELECT COUNT(*) FROM teams;"
psql -U postgres -d postgres -c "SELECT COUNT(*) FROM players;"
```

## Troubleshooting

### Migration fails to find fixtures

```
Error: failed to load fixtures: no such file or directory
```

**Solution**: Set correct `FIXTURES_DIR` path:
```bash
FIXTURES_DIR=/absolute/path/to/fixtures
```

### Duplicate key errors

```
Error: duplicate key value violates unique constraint
```

**Solution**: Migration should be idempotent, but if this occurs:
1. Check if migration ran twice simultaneously
2. Verify idempotency checks in code
3. Clear database and re-run

### Missing foreign key references

```
WARN: Team not found for player
```

**Solution**: Check migration order - teams must be migrated before players.

### Performance issues

If migration is slow:
1. Disable logging temporarily
2. Use batch inserts (future enhancement)
3. Run migration offline before deployment

## Future Enhancements

- [ ] Batch insert support for better performance
- [ ] Progress bar/percentage completion
- [ ] Migration dry-run mode
- [ ] Export migration report (JSON/CSV)
- [ ] Rollback support
- [ ] Incremental migration (only new records)
- [ ] Schema version tracking
- [ ] CLI tool for fixture validation/generation

## Recent Changes

### v2.0 (Migration Overhaul)

- **Added `id_mapping.go`**: Thread-safe ID mapping between legacy Django PKs and new UUIDs
- **Added `fixture_loader.go`**: Go-based fixture loading (replaces Python scripts)
- **Fixed fixture file paths**: Corrected paths for division pools and fields
- **Enabled spirit score migration**: Added system user creation and MVP/Spirit nominations
- **Required relationships**: Added proper handling for required edges (division_pool, field_location)
- **Field name normalization**: Handle legacy field variations (team1/home_team, etc.)
