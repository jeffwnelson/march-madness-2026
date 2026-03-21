# Playwright Smoke Tests

## Goal

Verify the app renders correctly at each tournament completion level using our 12 generated scenarios. Tests run against the live Go server.

## Test Matrix

12 scenarios, 1 smoke test each:

| Scenario | Completed through | Verifications |
|----------|------------------|---------------|
| R64_1, R64_2 | R64 | 32 R64 matchups with winners, R32 shows matchups without winners |
| R32_1, R32_2 | R32 | R64 + R32 decided, S16 undecided |
| S16_1, S16_2 | S16 | Through S16 decided, E8 undecided |
| E8_1, E8_2 | E8 | Through E8 decided, FF undecided |
| FF_1, FF_2 | FF | Through FF decided, Championship undecided |
| Champ_1, Champ_2 | Championship | All rounds decided |

## Per-Test Assertions

Each smoke test:

1. **Navigate** to `http://localhost:8000`
2. **Leaderboard tab** — verify 19 `.lb-row` elements render
3. **Switch to Bracket Picks tab** — click `.tab-btn[data-tab="consensus"]`
4. **For each decided round** — click `.round-tab[data-round="{key}"]`, verify matchup cards with winners exist (`.bk-matchup.decided`)
5. **For the first undecided round** (if not Champ scenario) — click the next round tab, verify no `.bk-matchup.decided` elements
6. **Open bracket modal** — click first `.lb-row`, verify `#bracket-modal` is visible with content

## Setup / Teardown

- **Before each scenario:** Copy the scenario's `brackets.json` to `data/`, start Go server on port 8000
- **After each scenario:** Stop the server
- Server is built once (`go build -o .server ./backend`) and reused

## File Structure

```
tests/
  smoke.spec.ts          Smoke tests for all 12 scenarios
  playwright.config.ts   Playwright config (baseURL, timeouts)
package.json             Playwright dev dependency
```

## Makefile

```makefile
test-e2e: ## Run Playwright smoke tests
	cd tests && npx playwright test
```

## Dependencies

- `@playwright/test` (dev dependency)
- Node.js (for Playwright runner)
- Go toolchain (for building the server)

## Round Keys

Used for tab selectors:

| Round | data-round value |
|-------|-----------------|
| R64 | `r64` |
| R32 | `r32` |
| Sweet 16 | `sweet16` |
| Elite 8 | `elite8` |
| Final Four | `finalFour` |
| Championship | `championship` |
