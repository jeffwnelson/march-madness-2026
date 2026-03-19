# March Madness 2026

Bracket tracking and scoring app for the NCAA March Madness tournament. Pulls bracket data from ESPN's Tournament Challenge API, processes picks and scores, and serves a web UI for tracking a private group's brackets.

## Features

- **Leaderboard** with scores, max possible points, and consensus picks
- **Round-by-round matchup view** showing pick distributions and results
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
  │                 │  - Hierarchical pick advancement via periodReached
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

Pick `result` fields (CORRECT/INCORRECT/UNDECIDED) update as games are played. The frontend uses a **hierarchical winner chain** to determine matchup visibility — each round's matchups only appear when both feeder games from the previous round are decided.

## Scoring

| Round        | Points |
|--------------|--------|
| R64          | 10     |
| R32          | 20     |
| Sweet 16     | 40     |
| Elite 8      | 80     |
| Final Four   | 160    |
| Championship | 320    |

## Automated Updates

A GitHub Actions workflow fetches fresh data from ESPN on a cron schedule during tournament play, commits the updated `data/brackets.json`, and pushes.
