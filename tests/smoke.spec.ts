import { test, expect } from '@playwright/test';
import { execSync, spawn, ChildProcess } from 'child_process';
import { copyFileSync } from 'fs';
import { resolve } from 'path';

const ROOT = resolve(__dirname, '..');
const SERVER_BIN = resolve(ROOT, '.server');
const LEADERBOARD_PATH = resolve(ROOT, 'data/leaderboard.json');
const BRACKET_PICKS_PATH = resolve(ROOT, 'data/bracket-picks.json');
const SCENARIOS_DIR = resolve(ROOT, 'backend/testdata/scenarios');

// Round keys match the data-round attributes in the UI
const ROUND_KEYS = ['r64', 'r32', 'sweet16', 'elite8', 'finalFour', 'championship'] as const;

interface Scenario {
  name: string;
  dir: string;
  /** Number of rounds fully decided (1 = R64 done, ..., 6 = Championship done) */
  decidedRounds: number;
}

const scenarios: Scenario[] = [
  { name: 'R64_1',   dir: 'R64_1',   decidedRounds: 1 },
  { name: 'R64_2',   dir: 'R64_2',   decidedRounds: 1 },
  { name: 'R32_1',   dir: 'R32_1',   decidedRounds: 2 },
  { name: 'R32_2',   dir: 'R32_2',   decidedRounds: 2 },
  { name: 'S16_1',   dir: 'S16_1',   decidedRounds: 3 },
  { name: 'S16_2',   dir: 'S16_2',   decidedRounds: 3 },
  { name: 'E8_1',    dir: 'E8_1',    decidedRounds: 4 },
  { name: 'E8_2',    dir: 'E8_2',    decidedRounds: 4 },
  { name: 'FF_1',    dir: 'FF_1',    decidedRounds: 5 },
  { name: 'FF_2',    dir: 'FF_2',    decidedRounds: 5 },
  { name: 'Champ_1', dir: 'Champ_1', decidedRounds: 6 },
  { name: 'Champ_2', dir: 'Champ_2', decidedRounds: 6 },
];

// Build the server binary once before all tests
test.beforeAll(() => {
  execSync('go build -o .server ./backend', { cwd: ROOT, stdio: 'pipe' });
});

for (const scenario of scenarios) {
  test.describe(scenario.name, () => {
    let server: ChildProcess;

    test.beforeAll(async () => {
      // Copy scenario data into place
      const srcLB = resolve(SCENARIOS_DIR, scenario.dir, 'leaderboard.json');
      const srcBP = resolve(SCENARIOS_DIR, scenario.dir, 'bracket-picks.json');
      copyFileSync(srcLB, LEADERBOARD_PATH);
      copyFileSync(srcBP, BRACKET_PICKS_PATH);

      // Start the server
      server = spawn(SERVER_BIN, [], { cwd: ROOT, stdio: 'pipe' });

      // Wait for server to be ready
      await waitForServer('http://localhost:8000', 5000);
    });

    test.afterAll(() => {
      if (server) {
        server.kill();
      }
    });

    test('smoke test', async ({ page }) => {
      await page.goto('/');

      // 1. Leaderboard tab — verify 19 bracket rows render
      const lbRows = page.locator('.lb-row');
      await expect(lbRows).toHaveCount(19);

      // 2. Switch to Bracket Picks tab
      await page.click('.tab-btn[data-tab="consensus"]');

      // 3. For each decided round, verify decided matchups exist
      for (let i = 0; i < scenario.decidedRounds; i++) {
        const roundKey = ROUND_KEYS[i];
        await page.click(`.round-tab[data-round="${roundKey}"]`);
        const decided = page.locator('.bk-matchup.decided');
        await expect(decided.first()).toBeVisible();
      }

      // 4. For the first undecided round (if not all rounds decided), verify no decided matchups
      if (scenario.decidedRounds < ROUND_KEYS.length) {
        const nextRound = ROUND_KEYS[scenario.decidedRounds];
        await page.click(`.round-tab[data-round="${nextRound}"]`);
        const decided = page.locator('.bk-matchup.decided');
        await expect(decided).toHaveCount(0);
      }

      // 5. Open bracket modal — click first leaderboard row
      // Switch back to leaderboard tab first
      await page.click('.tab-btn[data-tab="leaderboard"]');
      await page.click('.lb-row >> nth=0');
      const modal = page.locator('#bracket-modal');
      await expect(modal).toBeVisible();
    });
  });
}

async function waitForServer(url: string, timeoutMs: number): Promise<void> {
  const start = Date.now();
  while (Date.now() - start < timeoutMs) {
    try {
      const res = await fetch(url);
      if (res.ok) return;
    } catch {
      // Server not ready yet
    }
    await new Promise(r => setTimeout(r, 100));
  }
  throw new Error(`Server at ${url} did not become ready within ${timeoutMs}ms`);
}
