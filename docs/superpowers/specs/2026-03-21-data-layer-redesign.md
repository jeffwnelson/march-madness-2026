# Data Layer Redesign: Stable JSON Format with ESPN Sync

## Problem

ESPN's challenge API uses a rolling proposition model — it only serves propositions for the current round. When R64 completes, those propositions disappear and R32 propositions replace them. Our processing code assumed R64 propositions would always be available, so the app breaks when ESPN rolls forward.

## Solution

Introduce an intermediate data layer between ESPN's API and our frontend. Instead of deriving everything from ESPN's current snapshot, we accumulate tournament state across syncs into two stable, frontend-optimized JSON files.

## Data Files

### Raw Snapshots

Saved on every sync for audit/debugging:

```
data/snapshots/
  2026-03-21T00-25-00Z-challenge.json
  2026-03-21T00-25-00Z-group.json
```

### Processed Frontend Files

```
data/leaderboard.json      # leaderboard tab + bracket modal
data/bracket-picks.json    # bracket picks tab
```

`data/brackets.json` is removed.

## Schemas

### `leaderboard.json`

```json
{
  "lastUpdated": "2026-03-21T00:25:00Z",
  "groupName": "Curtis / Vasquez Madness",
  "teams": {
    "<teamId>": {
      "name": "Duke",
      "abbrev": "DUKE",
      "seed": 1,
      "region": 1,
      "logo": "https://..."
    }
  },
  "brackets": [
    {
      "entryName": "Jeff's Worst Picks Ever",
      "member": "Jeff",
      "score": 190,
      "maxPossible": 1870,
      "rank": 8,
      "percentile": 72.5,
      "eliminated": false,
      "tiebreaker": 152.0,
      "champion": "<teamId>",
      "finalFour": ["<teamId>", "<teamId>", "<teamId>", "<teamId>"]
    }
  ]
}
```

### `bracket-picks.json`

```json
{
  "lastUpdated": "2026-03-21T00:25:00Z",
  "teams": {
    "<teamId>": {
      "name": "Duke",
      "abbrev": "DUKE",
      "seed": 1,
      "region": 1,
      "logo": "https://..."
    }
  },
  "rounds": {
    "r64": {
      "status": "complete",
      "matchups": [
        {
          "id": "...",
          "region": 1,
          "displayOrder": 1,
          "team1": "<teamId>",
          "team2": "<teamId>",
          "winner": "<teamId>",
          "status": "COMPLETE",
          "gameTime": 1774109400000,
          "picks": {
            "<teamId>": {
              "count": 18,
              "entries": ["Jeff's Worst Picks Ever", "Delka's Dunk Dynasty"]
            },
            "<teamId>": {
              "count": 1,
              "entries": ["Lenny 🐶🐾"]
            }
          }
        }
      ]
    },
    "r32": {
      "status": "in_progress",
      "matchups": []
    },
    "sweet16": { "status": "future", "matchups": [] },
    "elite8": { "status": "future", "matchups": [] },
    "finalFour": { "status": "future", "matchups": [] },
    "championship": { "status": "future", "matchups": [] }
  }
}
```

Round status values: `"complete"`, `"in_progress"`, `"future"`.

## Sync Process

The `--fetch-only` flow:

1. **Fetch** — Hit ESPN challenge + group endpoints
2. **Snapshot** — Save raw responses to `data/snapshots/<timestamp>-challenge.json` and `<timestamp>-group.json`
3. **Load existing** — Read current `leaderboard.json` and `bracket-picks.json` (if they exist)
4. **Merge** — Process ESPN data and merge into existing state:
   - **Teams**: always overwrite (ESPN may update records/logos)
   - **Matchups**: add new rounds, update status/winners on existing, never delete
   - **Brackets**: overwrite scores/rank/maxPossible/eliminated (change every sync); champion and finalFour are stable
   - **Pick counts**: recompute from entry picks on every sync
   - **Round status**: derived — all matchups COMPLETE → `"complete"`, any PLAYING or COMPLETE → `"in_progress"`, else `"future"`
5. **Write** — Save `leaderboard.json` and `bracket-picks.json`

### Team ID Stability

Current ESPN outcome IDs serve as team IDs. When ESPN rolls to a new round with new outcome IDs, we map old → new using region+seed (unique per team). All stored data uses the latest canonical IDs.

### Matchup Accumulation

Matchups never disappear. When ESPN rolls from R64 to R32:
- Existing R64 matchups in `bracket-picks.json` are preserved as-is
- New R32 matchups are added from the current ESPN propositions
- R32 matchup team IDs are the R64 winners (resolved from picks or from the R32 proposition structure)

## Frontend Changes

- **Load two files** instead of one
- **Leaderboard tab**: reads `leaderboard.json` — rendering logic unchanged, just data source
- **Bracket picks tab**: reads `bracket-picks.json` — simpler since picks are pre-aggregated (`matchup.picks[teamId].count`)
- **Bracket modal**: reads `leaderboard.json` for scores/champion/finalFour; per-round pick details from `bracket-picks.json` entry lists
- **Pill auto-advance**: `rounds[key].status === "complete"` instead of counting completed matchups
- **Eliminated teams**: derived from matchup winners in `bracket-picks.json` (unchanged logic)

## What Gets Removed

- `data/brackets.json` — replaced by two new files
- Complex outcome ID mapping / championship resolution in `process.go` — replaced by merge-based sync
- Client-side pick aggregation in `app.js` — moved server-side

## Testing

- Existing Go unit tests updated for new processing functions
- Scenario-based tests updated to produce the new JSON format
- Playwright smoke tests updated to load new data files
