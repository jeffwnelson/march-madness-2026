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

## Project Structure

```
├── backend/               Go server and data processing
│   ├── server.go          HTTP server, ESPN API client
│   ├── process.go         ESPN data → bracket JSON transform
│   ├── *_test.go          Tests and scenario generator
│   └── testdata/          ESPN snapshots and generated scenarios
├── css/style.css          Styles
├── js/app.js              Frontend application
├── index.html             HTML shell
├── data/brackets.json     Processed bracket data (served to frontend)
├── Makefile               Server and test data management
└── .github/workflows/     CI/CD (cron updates, versioning, Pages deploy)
```

## Architecture

```
ESPN Tournament Challenge API
  ├─ /challenges/tournament-challenge-bracket-2026      (matchups, teams, outcomes)
  └─ /challenges/.../groups/{groupId}?view=entries      (bracket entries, picks, scores)
        │
        ▼
  ┌──────────────────────┐
  │  backend/process.go  │  Transforms ESPN data → processed bracket JSON
  │                      │  - Teams, matchups (with winners)
  │                      │  - Per-entry picks bucketed by round (R64→Championship)
  │                      │  - Champion resolution via finalPick cross-entry correlation
  │                      │  - ESPN stats: percentile, eliminated, tiebreaker
  └──────────┬───────────┘
             │
             ▼
  ┌──────────────────────┐
  │  backend/server.go   │  HTTP server (:8000)
  │                      │  - GET  /              → index.html
  │                      │  - GET  /api/brackets  → cached bracket JSON
  │                      │  - POST /api/refresh   → re-fetch from ESPN
  └──────────┬───────────┘
             │
             ▼
  ┌──────────────────────┐
  │  index.html          │  Single-page app (vanilla JS)
  │  css/style.css       │  - Matchup cards with pick counts
  │  js/app.js           │  - Hierarchical round rendering
  │                      │  - Bracket detail modal with stats
  │                      │  - What-if score simulator
  └──────────────────────┘
```

## Quick Start

```bash
make start          # Start server in background
make stop           # Stop server
make restart        # Restart server
make test           # Run all tests
make help           # Show all commands
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

Each round's pick `result` fields (CORRECT/INCORRECT/UNDECIDED) indicate whether the team won that round's game. For example, an R32 pick marked CORRECT means the team won their R32 game.

The frontend uses a **hierarchical winner chain** to determine matchup visibility. Each round's pick results directly determine that round's winners. Later-round matchups only appear once the previous round's games are decided.

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

```bash
make test
```

Tests include structural validation (`TestProcessData`), HTTP endpoint tests, and 12 scenario-based integration tests (`TestScenarios`) that validate pick results at each tournament completion level.

## Mock Tournament Data

`testdata_gen_test.go` generates simulated tournament scenarios for end-to-end testing without waiting for real games. It loads real ESPN snapshots from `testdata/`, mutates them with random outcomes, and produces processed `brackets.json` files.

### Scenarios

12 scenarios are generated — 2 at each completion level:

| Scenarios | Completed through | Decided games |
|-----------|------------------|---------------|
| `R64_1`, `R64_2` | Round of 64 | 32 |
| `R32_1`, `R32_2` | Round of 32 | 48 |
| `S16_1`, `S16_2` | Sweet 16 | 56 |
| `E8_1`, `E8_2` | Elite 8 | 60 |
| `FF_1`, `FF_2` | Final Four | 62 |
| `Champ_1`, `Champ_2` | Championship | 63 |

Each scenario uses a deterministic PRNG seed so results are reproducible.

### Generating scenario files

```bash
make generate-scenarios   # Generate all scenario files
make test                 # Run validation tests
```

### Loading a scenario locally

```bash
make load-e8              # Load Elite 8 scenario (or load-r64, load-r32, etc.)
make restart              # Restart server with new data

make load-real            # Restore real ESPN data
```

## CI/CD

| Workflow | Trigger | Jobs |
|----------|---------|------|
| **Version Bump** | push to master | version → update-version-file → release → deploy |
| **Deploy Pages** | push to master (chore commits) | deploy to GitHub Pages |
| **Update Brackets** | cron (every 10min during games) | fetch ESPN data → commit → push |

- **Versioning**: [cocogitto](https://github.com/cocogitto/cocogitto) with conventional commits — `feat:` bumps minor, `fix:` bumps patch
- **Releases**: GitHub releases with cocogitto-generated changelogs
- **Pages**: Actions-based deployment, runs after version bump and release complete
