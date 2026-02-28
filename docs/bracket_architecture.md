# Tournament Scheduling & Bracket Architecture

This document outlines the current architecture, data flow, and processing rules for standard tournament progressions (Round Robin Pools -> Play-in Crossovers -> Elimination Brackets) in the `game-stats` application.

## 1. Schema Overview

The tournament progression relies on the following core entities (managed via Ent):

### `Event`
- The top-level container for a tournament.
- Contains `DivisionPool` and `GameRound` records.

### `GameRound`
- Defines a specific stage in the tournament timeline for an Event.
- Fields: `RoundType` (e.g., `pool`, `crossover`, `bracket`, `semifinal`, `final`), `RoundNumber`, `AutoAdvance` (boolean), `TopNTeams` (int).

### `DivisionPool`
- A specific subgroup within an event (e.g., Pool A, Pool B, Open Division Bracket).
- Contains teams and games.
- Fields: `AutoAdvance` (boolean), `TopNTeams` (int).
- Edges:
  - `TargetRound`: Links to a `GameRound`. This dictates where the top teams from this pool will advance after all games in the pool are complete.

### `Game`
- Represents a specific matchup between a Home Team and an Away Team.
- Edges:
  - `GameRound`: The round this game belongs to.
  - `DivisionPool`: The pool (or bracket stage) this game belongs to.
- Key States: `scheduled`, `in_progress`, `ended`, `completed`, `canceled`.

---

## 2. Frontend Triggers & Forms

The scheduling and ranking logic is driven by administrative actions on the UI (`game-stats-ui`):

### `GameForm.tsx`
- **Creation Mode**: Admins select an Event, Division (Pool), and Round. The form then filters the `HomeTeam` and `AwayTeam` dropdowns based on the selected `DivisionPool`.
- **Edit Mode (Overrides)**: Used to manually modify games. Currently, the UI hides team selections in edit mode, preventing manual adjustments of auto-generated "TBD vs TBD" bracket games.
  - *Upcoming Change*: Refactor to fetch teams by `Event` instead of restricting by `DivisionPool`, allowing admins to manually pair cross-pool matchups (e.g., a team from Pool A against a team from Pool B).

### API Bindings (`publicApi`)
- `listTeams`: Fetches teams for a division/event.
- `listGames`, `getEventBracket`: Retrieves tournament data for visualization.

---

## 3. Backend Execution Flow

The automation logic lives primarily in `ranking.Service` and `bracket.Service`.

### 3.1 Round Robin Pool Play
1. Admins use the UI to manually schedule games in a `DivisionPool` (RoundType = `pool`).
2. Scorekeepers submit scores. A background worker (`score_sync_worker`) or manual API trigger transitions games from `ended` to `completed`.
3. **The Trigger**: The `HandleGameEnded` hook in `ranking.Service` fires every time a game completes.
4. **Validation**: It checks the pool's `TargetRound`:
   - If the pool targets a `crossover` or `bracket` round, it verifies if **every game in the pool has ended/completed** AND that there are **at least two completed games** overall (to ensure standings are valid).
5. **Advancement**: Once all pools targeting the specific round are complete, the `AdvanceTeams` logic executes.

### 3.2 Crossover Play-in Stage
1. **Generation Strategy**: When the `pool` stage completes, the `TargetRound` (e.g., "Play-Ins") creates the crossover matches.
2. The logic identifies the top seeds across the interconnected pools.
3. **Structure**: Seeds are paired standardly (e.g., A1 vs D2, B1 vs C2) or dynamically based on `TopNTeams`.
4. The system creates unplayed `Game` entities assigned to the `crossover` target round.

### 3.3 Continuous Elimination Promotion
1. Similar to pools, `HandleGameEnded` evaluates completed crossover or bracket games.
2. In a single elimination tree, the winning `TeamID` is automatically propelled into the `LeftChildNode` or `RightChildNode` of the upper `TargetRound` bracket (e.g., Quarterfinals).
3. The losing `TeamID` is pushed into the lower `targetRound` bracket (Placement/Consolation), if configured.
4. This flow loops automatically from Quarterfinals to Semifinals to Finals as games finish.

---

## 4. Required Structural Changes (Upcoming Implementation)

To robustly support the Ultimate Frisbee style progression detailed above, the following adaptations are necessary:

1. **DTO Updates**: `UpdateGameRequest` must include `HomeTeamID` and `AwayTeamID` to permit manual UI overrides on generated bracket matchups.
2. **Global Pool Tracking**: `handlePoolGameEnded` currently evaluates pools in isolation. It must be refactored to look at the unified `TargetRound` and verify *all* related constituent pools are finished before calculating the crossover.
3. **Frontend decoupling**: The `GameForm` schema must distinguish between `create` restrictions (same pool only) and `edit` realities (allowing cross-pool assignments).
