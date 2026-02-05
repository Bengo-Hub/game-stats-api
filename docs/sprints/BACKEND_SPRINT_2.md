# Backend Sprint 2: Game Management & Scoring System

**Duration**: 2-3 weeks
**Focus**: Game scheduling, real-time scoring, game state management
**Status**: ✅ All features complete (Testing pending)

---

## Sprint Goals

- ✅ Implement complete game management system
- ✅ Build real-time scoring with SSE
- ✅ Create game timer with stoppage tracking  
- ✅ Implement GameRound and tournament structure
- ✅ Build SpiritScore system with full API
- ✅ Add game event timeline

---

## Prerequisites from Sprint 1

- ✅ Database schema and migrations
- ✅ Authentication system
- ✅ Geographic hierarchy models
- ✅ Event and DivisionPool models
- ✅ Team and Player models

---

## Tasks

### Week 1: Game Scheduling & Management

#### Day 1-2: GameRound Implementation
- [x] Implement GameRound model and repository
- [x] Create round types (pool, bracket, semifinal, final)
- [x] Add round sequencing logic
- [x] Build GameRound API endpoints
  - `POST /api/v1/events/{id}/rounds` - Create round
  - `GET /api/v1/events/{id}/rounds` - List rounds
  - `PUT /api/v1/rounds/{id}` - Update round
- [x] Write unit tests for GameRound service

**Deliverable**: Complete round management system

---

#### Day 3-5: Game Model & Scheduling
- [x] Implement Game model with Ent
  - All fields from ERD (status, times, scores, etc.)
  - Version field for optimistic locking
- [x] Create Game repository with queries:
  - By division pool
  - By date range
  - By status
  - By field
- [x] Implement Game service logic:
  - Schedule game
  - Validate no field conflicts
  - Assign scorekeeper
  - Update game status
- [x] Build Game API endpoints:
  - `POST /api/v1/divisions/{id}/games` - Schedule game
  - `GET /api/v1/games` - List with filters
  - `GET /api/v1/games/{id}` - Get details
  - `PUT /api/v1/games/{id}` - Update details
  - `DELETE /api/v1/games/{id}` - Cancel game
- [x] Add field conflict detection

**Deliverable**: Game scheduling system

---

### Week 2: Game Timer & State Management

#### Day 6-7: Timer System
- [x] Implement timer states (scheduled, in_progress, finished, ended)
- [x] Create StartGame service method:
  ```go
  func (s *GameService) StartGame(ctx context.Context, gameID uuid.UUID, userID uuid.UUID) error
  ```
  - Validate scorekeeper permissions
  - Set actual_start_time
  - Calculate expected end time
  - Broadcast SSE event
  - Schedule auto-finish goroutine
- [x] Create FinishGame method (time expired, allow score edits)
- [x] Create EndGame method (final submission by scorekeeper)
- [x] Build timer API endpoints:
  - `POST /api/v1/games/{id}/start` -Start game
  - `POST /api/v1/games/{id}/finish` - Mark finished
  - `POST /api/v1/games/{id}/end` - Final submission
- [x] Add timer background jobs

**Deliverable**: Complete timer system

---

#### Day 8-9: Game Stoppage Tracking
- [x] Create GameEvent model for timeline
- [x] Implement stoppage event types
- [x] Build RecordStoppage method:
  ```go
  func (s *GameService) RecordStoppage(ctx context.Context, gameID uuid.UUID, duration int, reason string) error
  ```
  - Update stoppage_time_seconds
  - Create GameEvent record
  - Extend game end time
  - Broadcast SSE update
- [x] Create GameEvent repository
- [x] Build stoppage API:
  - `POST /api/v1/games/{id}/stoppages` - Record stoppage
  - `GET /api/v1/games/{id}/timeline` - Get complete timeline
- [x] Add stoppage statistics

**Deliverable**: Game stoppage tracking

---

#### Day 10: SSE Implementation Part 1
- [x] Create SSE broker service
  ```go
  type SSEBroker struct {
      clients map[uuid.UUID][]chan Event
      events  chan Event
  }
  ```
- [x] Implement client management:
  - Add client
  - Remove client
  - Broadcast to game clients
- [x] Create SSE handler:
  - `GET /api/v1/games/{id}/stream` - SSE endpoint
  - Set correct headers
  - Handle client disconnection
- [x] Define event types:
  - game_started
  - goal_scored
  - assist_recorded
  - stoppage_recorded
  - game_finished
  - game_ended

**Deliverable**: SSE infrastructure

---

### Week 3: Scoring & Spirit Scores

#### Day 11-12: Scoring System
- [x] Implement Scoring model and repository
- [x] Create RecordScore method with validation:
  ```go
  func (s *GameService) RecordScore(ctx context.Context, req RecordScoreRequest) error
  ```
  - Verify game is in_progress or finished
  - Verify player belongs to team
  - Update or create Scoring record
  - Recalculate game scores
  - Create GameEvent for goal/assist
  - Broadcast SSE event
  - Use optimistic locking
- [x] Build scoring API:
  - `POST /api/v1/games/{id}/scores` - Record score
  - `GET /api/v1/games/{id}/scores` - Get all scores
  - `PUT /api/v1/scores/{id}` - Edit score (admin)
- [x] Add score validation rules
- [x] Implement score edit audit trail

**Deliverable**: Complete scoring system

---

#### Day 13-14: Spirit Scores
- [x] Implement SpiritScore model
- [x] Create spirit submission service:
  ```go
  func (s *Service) SubmitSpiritScore(ctx context.Context, gameID uuid.UUID, req SubmitSpiritScoreRequest) (*SpiritScoreDTO, error)
  func (s *Service) GetGameSpiritScores(ctx context.Context, gameID uuid.UUID) ([]*SpiritScoreDTO, error)
  func (s *Service) GetTeamSpiritAverage(ctx context.Context, teamID uuid.UUID) (*TeamSpiritAverageDTO, error)
  ```
  - Validate one submission per team per game
  - Validate score ranges (0-4)
  - Save MVP and Spirit nominations
- [x] Build spirit API:
  - `POST /api/v1/games/{id}/spirit` - Submit spirit score
  - `GET /api/v1/games/{id}/spirit` - Get spirit scores
  - `GET /api/v1/teams/{id}/spirit-average` - Team average
- [x] Add spirit leaderboard queries (team averages)
- [x] Implement MVP/Spirit nomination tracking

**Deliverable**: Spirit score system

---

#### Day 15: Integration & Testing
- [ ] Write integration tests:
  - Complete game workflow (schedule → start → score → finish → end)
  - SSE event delivery
  - Timer expiration
  - Stoppage handling
  - Spirit score submission
- [ ] Test concurrent score updates
- [ ] Test optimistic locking
- [ ] Performance test SSE with 100+ clients
- [ ] Load test scoring endpoints

**Deliverable**: Test suite with >75% coverage

---

## Definition of Done

✅ All game management features implemented  
✅ Timer system with stoppages working  
✅ SSE real-time updates functional  
✅ Scoring system with optimistic locking  
✅ Spirit scores and nominations (full API implementation complete)
⬜ Integration tests passing  
⬜ Code coverage >75%  
✅ API documentation updated (Swagger annotations added)  

---

## API Endpoints Summary

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/api/v1/events/{id}/rounds` | POST | Create round |
| `/api/v1/divisions/{id}/games` | POST | Schedule game |
| `/api/v1/games` | GET | List games with filters |
| `/api/v1/games/{id}` | GET | Get game details |
| `/api/v1/games/{id}/start` | POST | Start game timer |
| `/api/v1/games/{id}/finish` | POST | Mark time expired |
| `/api/v1/games/{id}/end` | POST | Final submission |
| `/api/v1/games/{id}/stoppages` | POST | Record stoppage |
| `/api/v1/games/{id}/timeline` | GET | Get event timeline |
| `/api/v1/games/{id}/stream` | GET | SSE live updates |
| `/api/v1/games/{id}/scores` | POST | Record score |
| `/api/v1/scores/{id}` | PUT | Edit score (admin) |
| `/api/v1/games/{id}/spirit` | POST | Submit spirit score |

---

## Database Migrations

```sql
-- Add indexes for performance
CREATE INDEX idx_games_status ON games(status) WHERE deleted_at IS NULL;
CREATE INDEX idx_games_field_scheduled ON games(field_id, scheduled_time);
CREATE INDEX idx_game_events_game_time ON game_events(game_id, minute, second);
CREATE INDEX idx_scoring_player ON scoring(player_id) WHERE deleted_at IS NULL;
```

---

## Technical Debt / Future Improvements

- Consider Redis pub/sub for SSE horizontal scaling
- Implement WebSocket fallback for SSE
- Add game highlights/summary generation
- Implement automated game scheduling algorithm
- Add push notifications for game start

---

**Next**: [Backend Sprint 3: Advanced Features & Analytics](./BACKEND_SPRINT_3.md)
