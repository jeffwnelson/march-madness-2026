# March Madness 2026

Bracket tracking and scoring app for the NCAA March Madness tournament. Pulls bracket data from ESPN's Tournament Challenge API, processes picks and scores, and serves a web UI for tracking a private group's brackets.

## Features

- **Leaderboard** with scores, max possible points, champion picks, player names, and bracket detail modal
- **Bracket detail modal** — click any leaderboard entry to see champion pick, Final Four (2x2 grid by region), national percentile, R64 record, and tiebreaker prediction
- **Round-by-round matchup view** showing pick distributions and results
- **Hierarchical round progression** — each round's matchups only appear when the previous round's games are decided
- **Bracket uniqueness** analysis comparing entries across the group
- **What-if simulator** to explore hypothetical outcomes and score impacts
- **Deep linking** via URI fragments for tabs and rounds
- **Automated updates** via GitHub Actions cron (noon–1am CST during the tournament)

## Architecture

```
ESPN Tournament Challenge API
  ├─ /challenges/tournament-challenge-bracket-2026      (matchups, teams, outcomes)
  └─ /challenges/.../groups/{groupId}?view=entries      (bracket entries, picks, scores)
        │
        ▼
  ┌─────────────────┐
  │   process.go    │  Transforms ESPN data → processed bracket JSON
  │                 │  - Teams, matchups (with winners)
  │                 │  - Per-entry picks bucketed by round (R64→Championship)
  │                 │  - Champion resolution via finalPick cross-entry correlation
  │                 │  - ESPN stats: percentile, eliminated, tiebreaker
  └────────┬────────┘
           │
           ▼
  ┌─────────────────┐
  │   server.go     │  HTTP server (:8000)
  │                 │  - GET  /              → index.html
  │                 │  - GET  /api/brackets  → cached bracket JSON
  │                 │  - POST /api/refresh   → re-fetch from ESPN
  └────────┬────────┘
           │
           ▼
  ┌─────────────────┐
  │   index.html    │  Single-page app (vanilla JS)
  │                 │  - Matchup cards with pick counts
  │                 │  - Hierarchical round rendering
  │                 │  - Bracket detail modal with stats
  │                 │  - What-if score simulator
  └─────────────────┘
```

## Quick Start

```bash
# Run the server (loads cached data or fetches from ESPN)
go run .

# Fetch fresh data only (for CI/cron jobs)
go run . --fetch-only

# Run tests
go test ./...
```

The server runs at `http://localhost:8000`.

## Data Flow

ESPN provides 32 R64 propositions. Each bracket entry's picks reference these propositions with a `periodReached` value indicating how far the user predicted that team would advance:

| periodReached | Round          |
|---------------|----------------|
| 2             | Round of 64    |
| 3             | Round of 32    |
| 4             | Sweet 16       |
| 5             | Elite 8        |
| 6             | Final Four     |

### Pick Results

ESPN pick `result` fields (CORRECT/INCORRECT/UNDECIDED) indicate whether a team **reached** that round, not whether they won in that round. For example, an R32 pick marked CORRECT means the team won R64 and reached R32.

The frontend uses a **hierarchical winner chain** to determine matchup visibility. To find who won round N, it checks round N+1's pick results (since reaching round N+1 means winning round N). This prevents later-round matchups from appearing prematurely.

### Champion Resolution

ESPN's `finalPick` field uses championship-proposition outcome IDs that differ from R64 outcome IDs. Champions are resolved via:

1. **Cross-entry correlation** — intersect R64 finalists across entries with the same championship pick
2. **Iterative elimination** — remove already-resolved teams from ambiguous sets
3. **ID offset fallback** — championship outcomes follow bracket ordering (region × 16 + bracket position)

## Scoring

| Round        | Points |
|--------------|--------|
| R64          | 10     |
| R32          | 20     |
| Sweet 16     | 40     |
| Elite 8      | 80     |
| Final Four   | 160    |
| Championship | 320    |

## Testing

12 bracket progression tests validate the hierarchical winner chain:

```bash
go test -v -run "TestR64|TestAll|TestR32|TestS16|TestIncorrect|TestFull|TestPartial|TestUpset|TestCross|TestE8" ./...
```

Tests cover: R64→R32→S16→E8→FF progression, partial results, upset chains, cross-region isolation, and the off-by-one pick result semantics.

## CI/CD

| Workflow | Trigger | Jobs |
|----------|---------|------|
| **Version Bump** | push to master | version → update-version-file → release → deploy |
| **Deploy Pages** | push to master (chore commits) | deploy to GitHub Pages |
| **Update Brackets** | cron (every 10min during games) | fetch ESPN data → commit → push |

- **Versioning**: [cocogitto](https://github.com/cocogitto/cocogitto) with conventional commits — `feat:` bumps minor, `fix:` bumps patch
- **Releases**: GitHub releases with cocogitto-generated changelogs
- **Pages**: Actions-based deployment, runs after version bump and release complete
