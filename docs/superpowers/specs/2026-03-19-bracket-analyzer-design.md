# NCAA Bracket Analyzer — Design Spec

## Overview

A local web app that analyzes 19 NCAA tournament brackets from an ESPN Tournament Challenge group. Built as a Go server with a single-file HTML/CSS/JS dashboard. Hosted as a GitHub repo with a scheduled Actions workflow to keep data fresh.

**Group URL:** `https://fantasy.espn.com/games/tournament-challenge-bracket-2026/group?id=af223df6-96d0-46e7-b00d-1b590dc67888`
**Repo:** `https://github.com/jeffwnelson/march-madness-2026`

## Goals

1. **Consensus picks** — what % of the group picked each team to win each game, across all rounds
2. **Bracket uniqueness** — how different each bracket is from the others
3. **What-if simulator** — select a game winner and see how it ripples through all brackets (points impact, eliminations, leaderboard shifts)

## Architecture

```
march-madness-2026/
├── server.go                          # Go server: ESPN fetcher + HTTP server + JSON API
├── go.mod
├── index.html                         # Single-file dashboard (HTML/CSS/JS)
├── data/
│   └── brackets.json                  # Cached processed bracket data
└── .github/
    └── workflows/
        └── update-brackets.yml        # Scheduled ESPN data refresh
```

### How it works

1. `go run server.go` starts a local server on `localhost:8000`
2. On startup, loads cached data from `data/brackets.json` (or fetches from ESPN if missing)
3. Serves `index.html` as the dashboard
4. Dashboard fetches data from `/api/brackets` and renders the three analysis views
5. "Refresh" button in the UI hits `/api/refresh` to re-fetch from ESPN

## Go Server (`server.go`)

### ESPN Data Pipeline

Two API calls to ESPN's Gambit API:

1. **Challenge data** — `GET gambit-api.fantasy.espn.com/apis/v1/challenges/tournament-challenge-bracket-2026`
   - Contains 32 R64 propositions with `possibleOutcomes` mapping outcome UUIDs to team names, seeds, regions, records, and logos
   - Contains `actualOutcomeIds` for completed games — the array has two elements: `[0]` is the winner, `[1]` is the loser

2. **Group data** — `GET gambit-api.fantasy.espn.com/apis/v1/challenges/tournament-challenge-bracket-2026/groups/af223df6-96d0-46e7-b00d-1b590dc67888?view=entries&limit=50`
   - Contains all 19 entries with full pick data
   - Each entry has 63 picks (32 R64 + 31 later rounds)
   - Each R64 pick has a `periodReached` field (1-6) encoding how far the user picked that team to advance

**Processing:**

1. Build `outcomeID → team` lookup from challenge data
2. For each entry, decode R64 picks using the outcome map
3. Use `periodReached` to reconstruct the full bracket (which teams advance through each round)
4. Extract game results from `actualOutcomeIds` to mark picks as CORRECT/INCORRECT/UNDECIDED
5. Compute scores using ESPN's point values: 10, 20, 40, 80, 160, 320 per round

**Later-round matchups:** The API only provides 32 R64 propositions. Later-round matchups are derived from actual results as games complete. For the consensus view, unresolved later rounds show all teams picked to reach that slot (grouped by bracket position using `regionId`, `regionSeed`, and `displayOrder`), not fixed team-vs-team pairings. Once a game's feeder matchups are resolved, the actual pairing is shown.

**CLI flag:** `--fetch-only` — fetches ESPN data, writes `data/brackets.json`, and exits without starting the HTTP server. Used by the GitHub Actions workflow.

### API Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/` | GET | Serves `index.html` |
| `/api/brackets` | GET | Returns processed bracket JSON |
| `/api/refresh` | POST | Re-fetches from ESPN, updates cache, returns fresh data |

### Processed Data Shape

```json
{
  "lastUpdated": "2026-03-19T15:30:00Z",
  "groupId": "af223df6-96d0-46e7-b00d-1b590dc67888",
  "pointsPerRound": [10, 20, 40, 80, 160, 320],
  "teams": {
    "c9068ee1-...": {
      "name": "Duke",
      "abbrev": "DUKE",
      "seed": 1,
      "region": 1,
      "record": "32-2",
      "logo": "https://a.espncdn.com/i/teamlogos/ncaa/500/150.png"
    }
  },
  "matchups": [
    {
      "id": "c9068ee0-...",
      "round": 1,
      "region": 1,
      "displayOrder": 1,
      "team1Id": "c9068ee1-...",
      "team2Id": "c9068ee2-...",
      "winnerId": null
    }
  ],
  "brackets": [
    {
      "member": "Snoopycake",
      "entryName": "9th Time's The Charm",
      "score": 10,
      "maxPossible": 1920,
      "picks": {
        "r64": [{"matchupId": "...", "pickedTeamId": "...", "result": "UNDECIDED"}],
        "r32": [...],
        "sweet16": [...],
        "elite8": [...],
        "finalFour": [...],
        "championship": [{"matchupId": "...", "pickedTeamId": "...", "result": "UNDECIDED"}]
      },
      "champion": "c90e3001-...",
      "finalFour": ["c9068ee1-...", "c90e3001-...", "c90c0d21-...", "c9094e01-..."]
    }
  ]
}
```

## Dashboard (`index.html`)

Single HTML file with embedded CSS and JS. Three tabs accessible via top navigation.

### Tab 1: Consensus Picks

For each round (R64 through Championship), display every matchup with a horizontal stacked bar showing pick distribution.

- Each matchup: two team names with seeds, a bar colored proportionally (e.g., 72% Duke / 28% Siena)
- Color intensity indicates consensus strength: dark green = near-unanimous, yellow = close to 50/50
- Layout grouped by region in bracket-style columns
- Completed games show the actual result with correct/incorrect styling

### Tab 2: Bracket Uniqueness

- **Originality ranking** — each bracket scored by average Hamming distance from all other brackets across 63 picks, normalized to 0-100%
- **Similarity table** — sortable table showing each bracket pair's overlap percentage
- Visual highlighting of the most unique and most mainstream brackets
- Breakdown by round: "Your R64 picks are 85% mainstream but your Final Four is the most unique in the group"

### Tab 3: What-If Simulator

- Display the current bracket in a visual bracket layout
- Click any unresolved matchup to select a winner
- On selection, instantly compute and display:
  - **Points impact per bracket**: how many points each bracket gains (if they picked this winner) or loses access to (if their pick is now eliminated)
  - **Max possible recalculation**: updated maximum possible score for each bracket
  - **Elimination alerts**: which brackets had their champion, Final Four, etc. picks killed by this result
  - **Projected leaderboard**: re-ranked standings based on current score + remaining possible points
- Support chaining multiple what-ifs to simulate a full scenario (with a "Reset" button to clear)

## Algorithms

### Consensus Calculation

For each of the 63 games in the bracket:
1. Count how many brackets picked team A vs team B
2. Express as percentage: `count / 19 * 100`
3. Group by round for the display

### Uniqueness Score (Hamming Distance)

For brackets A and B:
1. Compare their 63 picks position by position
2. Count disagreements: `distance = count(A[i] != B[i])`
3. Normalize: `similarity = 1 - (distance / 63)`
4. Bracket originality = `mean(distance to all other brackets) / 63 * 100`

### What-If Propagation

When team X is selected to win game G:
1. Mark game G result as team X
2. For each bracket:
   - If they picked team X: mark that pick CORRECT, add round points to score
   - If they picked the loser: mark INCORRECT, and cascade — find all later-round picks where they had the losing team advancing, mark those as ELIMINATED
3. Recalculate `maxPossible` for each bracket: `currentScore + sum(points for remaining UNDECIDED picks that aren't ELIMINATED)`
4. Re-sort leaderboard by `maxPossible` (or current score as tiebreaker)

## GitHub Actions Workflow

**File:** `.github/workflows/update-brackets.yml`

**Trigger:** Cron schedule — every 15 minutes during tournament game windows (roughly 12pm-12am ET on game days), every 2 hours otherwise.

**Steps:**
1. Checkout repo
2. Set up Go
3. Run `go run server.go --fetch-only` (a CLI flag that fetches data, writes `data/brackets.json`, and exits without starting the server)
4. Check if `data/brackets.json` changed (`git diff`)
5. If changed: commit and push with message like "Update bracket data — 2026-03-19T15:30:00Z"

**Cron expression:** `*/15 * * * *` during March Madness (can be adjusted manually or via a matrix of game-day schedules).

## Non-Goals

- No authentication or multi-user support
- No persistent database — JSON file is the data store
- No deployment to a public server (local use only, data lives in GitHub)
- No bracket editing or submission — read-only analysis
