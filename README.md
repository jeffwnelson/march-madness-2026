# March Madness 2026

Bracket tracking app for the NCAA March Madness tournament. Pulls bracket data from ESPN's Tournament Challenge API, processes it into static JS files, and serves a web UI for tracking a private group's brackets.

## Features

- **Leaderboard** — scores, max possible points, correct picks, champion logos (greyed out if eliminated), bracket detail modal
- **Bracket detail modal** — champion pick, Final Four (2x2 grid by region), national percentile, tiebreaker prediction
- **Bracket Picks** — round-by-round matchup cards showing which group members picked each team
- **Slide-out drawer** — tap any matchup to see who picked each team with bracket names
- **Auto-advancing round pills** — defaults to the first incomplete round
- **Live game detection** — games past their start time show "PLAYING" with blue border
- **Deep linking** — URL hash tracks tab, round, sort mode, and open matchup drawer
- **No server required** — static HTML/JS files, works with GitHub Pages or opening `index.html` directly

## Project Structure

```
├── backend/
│   └── main.go              ESPN data → data.js + leaderboard.js
├── css/style.css             Styles
├── js/app.js                 Frontend application
├── index.html                HTML shell
├── data/
│   ├── espn/                 Raw ESPN API snapshots
│   │   ├── challenge.json    Propositions, teams, outcomes
│   │   └── group.json        Bracket entries, picks, scores
│   ├── data.js               Processed matchup data (teams + rounds)
│   └── leaderboard.js        Processed leaderboard data (entries + scores)
├── Makefile                  Commands
└── .github/workflows/        CI/CD
```

## How It Works

```
ESPN Tournament Challenge API
  ├─ challenge endpoint    (propositions, teams, outcomes, game results)
  └─ group endpoint        (bracket entries, picks, scores)
        │
        ▼
  ┌──────────────────────┐
  │  backend/main.go     │  Reads data/espn/*.json
  │                      │  Outputs data/data.js + data/leaderboard.js
  │                      │
  │  - Teams from R32 proposition outcomes (all 64)
  │  - R64 matchups reconstructed from R32 prop positions
  │  - R64 winners from actualOutcomeIds
  │  - R32 matchups from R32 props, winners from correctOutcomes
  │  - Pick aggregation per matchup from entry picks
  │  - Champion resolution via hex offset on finalPick IDs
  └──────────┬───────────┘
             │
             ▼
  ┌──────────────────────┐
  │  index.html          │  Static site (no server needed)
  │  data/data.js        │  const DATA = { teams, rounds }
  │  data/leaderboard.js │  const LEADERBOARD = { entries }
  │  js/app.js           │  Renders leaderboard + bracket picks
  └──────────────────────┘
```

## ESPN Data Model

ESPN uses a **rolling proposition model** — only the current round's propositions are available. When R64 completes, R64 propositions are replaced with R32 propositions.

Key fields:
- **`scoringPeriodId`** — current round (1=R64, 2=R32, 3=S16, etc.)
- **`actualOutcomeIds`** — teams that won their previous-round game (R64 winners)
- **`correctOutcomes`** — team that won the current-round game (only on COMPLETE props)
- **`possibleOutcomes`** — 4 teams per R32 prop: positions 1,2 = R64 game A, positions 3,4 = R64 game B

Entry picks use `periodReached` to indicate how far a team advances in that bracket:

| periodReached | Meaning |
|---------------|---------|
| 2 | Team wins R64 (reaches R32) |
| 3 | Team wins R32 (reaches S16) |
| 4 | Team wins S16 (reaches E8) |
| 5 | Team wins E8 (reaches FF) |
| 6 | Team wins FF (reaches Championship) |

## Quick Start

```bash
# Download fresh ESPN data
curl -s "https://gambit-api.fantasy.espn.com/apis/v1/challenges/tournament-challenge-bracket-2026" | python3 -m json.tool > data/espn/challenge.json
curl -s "https://gambit-api.fantasy.espn.com/apis/v1/challenges/tournament-challenge-bracket-2026/groups/af223df6-96d0-46e7-b00d-1b590dc67888?view=entries&limit=50" | python3 -m json.tool > data/espn/group.json

# Generate static JS files
go run ./backend/

# Open in browser (no server needed)
open index.html
```

## Scoring

| Round        | Points |
|--------------|--------|
| R64          | 10     |
| R32          | 20     |
| Sweet 16     | 40     |
| Elite 8      | 80     |
| Final Four   | 160    |
| Championship | 320    |
