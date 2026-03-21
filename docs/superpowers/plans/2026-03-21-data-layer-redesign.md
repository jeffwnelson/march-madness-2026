# Data Layer Redesign Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace single `brackets.json` with two frontend-optimized files (`leaderboard.json`, `bracket-picks.json`) and ESPN snapshot archiving, so the app survives ESPN's rolling proposition model.

**Architecture:** The Go backend syncs ESPN data into raw snapshots and merges into two stable JSON files that accumulate tournament state across rounds. The frontend loads these two files instead of one, with pre-aggregated pick counts eliminating client-side computation.

**Tech Stack:** Go (backend sync/serve), vanilla JS (frontend), GitHub Actions (scheduled sync)

**Spec:** `docs/superpowers/specs/2026-03-21-data-layer-redesign.md`

---

## File Structure

### New Files
- `backend/sync.go` — Sync logic: fetch ESPN, save snapshots, merge into output JSONs
- `backend/sync_test.go` — Tests for sync logic
- `backend/types.go` — Output types for leaderboard.json and bracket-picks.json
- `data/leaderboard.json` — Leaderboard tab data (generated)
- `data/bracket-picks.json` — Bracket picks tab data (generated)
- `data/snapshots/` — Raw ESPN API response archive (generated)

### Modified Files
- `backend/server.go` — Update `--fetch-only` to use new sync, update HTTP handlers
- `backend/process.go` — Remove (replaced by sync.go and types.go)
- `backend/process_test.go` — Remove (replaced by sync_test.go)
- `backend/server_test.go` — Update for new types
- `js/app.js` — Load two JSONs, use pre-aggregated data, simplify rendering
- `.github/workflows/update-brackets.yml` — Check two output files instead of one
- `.gitignore` — Add `data/snapshots/`
- `Makefile` — Update load-* targets for new file format

### Removed Files
- `data/brackets.json` — Replaced by leaderboard.json + bracket-picks.json
- `backend/process.go` — Replaced by sync.go + types.go

---

### Task 1: Define Output Types

**Files:**
- Create: `backend/types.go`

- [ ] **Step 1: Create types.go with output structs**

```go
package main

import "time"

// Output types for leaderboard.json

type LeaderboardData struct {
	LastUpdated string                  `json:"lastUpdated"`
	GroupName   string                  `json:"groupName"`
	Teams       map[string]TeamInfo     `json:"teams"`
	Brackets    []LeaderboardBracket    `json:"brackets"`
}

type TeamInfo struct {
	Name   string `json:"name"`
	Abbrev string `json:"abbrev"`
	Seed   int    `json:"seed"`
	Region int    `json:"region"`
	Logo   string `json:"logo"`
}

type LeaderboardBracket struct {
	EntryName   string   `json:"entryName"`
	Member      string   `json:"member"`
	Score       int      `json:"score"`
	MaxPossible int      `json:"maxPossible"`
	Rank        int      `json:"rank"`
	Percentile  float64  `json:"percentile"`
	Eliminated  bool     `json:"eliminated"`
	Tiebreaker  *float64 `json:"tiebreaker"`
	Champion    string   `json:"champion"`
	FinalFour   []string `json:"finalFour"`
}

// Output types for bracket-picks.json

type BracketPicksData struct {
	LastUpdated string              `json:"lastUpdated"`
	Teams       map[string]TeamInfo `json:"teams"`
	Rounds      map[string]Round    `json:"rounds"`
}

type Round struct {
	Status   string        `json:"status"` // "complete", "in_progress", "future"
	Matchups []MatchupData `json:"matchups"`
}

type MatchupData struct {
	ID           string              `json:"id"`
	Region       int                 `json:"region"`
	DisplayOrder int                 `json:"displayOrder"`
	Team1        string              `json:"team1"`
	Team2        string              `json:"team2"`
	Winner       string              `json:"winner,omitempty"`
	Status       string              `json:"status"`
	GameTime     *int64              `json:"gameTime,omitempty"`
	Picks        map[string]PickData `json:"picks"`
}

type PickData struct {
	Count   int      `json:"count"`
	Entries []string `json:"entries"`
}

// roundKeyOrder defines the canonical round key order and period mapping.
var roundKeyOrder = []string{"r64", "r32", "sweet16", "elite8", "finalFour", "championship"}

func periodToRoundKey(period int) string {
	if period >= 1 && period <= len(roundKeyOrder) {
		return roundKeyOrder[period-1]
	}
	return ""
}

var _ = time.Now // keep import for other files
```

- [ ] **Step 2: Verify it compiles**

Run: `go build -o .server ./backend/`
Expected: Success (no errors)

- [ ] **Step 3: Commit**

```
git add backend/types.go
git commit -m "feat: add output types for leaderboard and bracket-picks JSON files"
```

---

### Task 2: Implement Sync Logic

**Files:**
- Create: `backend/sync.go`
- Create: `backend/sync_test.go`

- [ ] **Step 1: Write sync_test.go with core test**

Test that syncESPNData produces correct leaderboard and bracket-picks data from the existing testdata fixtures.

```go
package main

import (
	"encoding/json"
	"os"
	"testing"
)

func loadTestChallenge(t *testing.T) *espnChallenge {
	t.Helper()
	data, err := os.ReadFile("testdata/challenge.json")
	if err != nil {
		t.Fatal(err)
	}
	ch, err := parseChallengeData(data)
	if err != nil {
		t.Fatal(err)
	}
	return ch
}

func loadTestGroup(t *testing.T) *espnGroup {
	t.Helper()
	data, err := os.ReadFile("testdata/group.json")
	if err != nil {
		t.Fatal(err)
	}
	g, err := parseGroupData(data)
	if err != nil {
		t.Fatal(err)
	}
	return g
}

func TestSyncProducesLeaderboard(t *testing.T) {
	ch := loadTestChallenge(t)
	g := loadTestGroup(t)

	lb, _ := buildOutputs(ch, g, nil, nil)

	if len(lb.Teams) != 64 {
		t.Errorf("expected 64 teams, got %d", len(lb.Teams))
	}
	if len(lb.Brackets) != 19 {
		t.Errorf("expected 19 brackets, got %d", len(lb.Brackets))
	}
	// Every bracket should have a champion
	for _, b := range lb.Brackets {
		if b.Champion == "" {
			t.Errorf("bracket %q has no champion", b.EntryName)
		}
		if len(b.FinalFour) != 4 {
			t.Errorf("bracket %q has %d finalFour, want 4", b.EntryName, len(b.FinalFour))
		}
	}
}

func TestSyncProducesBracketPicks(t *testing.T) {
	ch := loadTestChallenge(t)
	g := loadTestGroup(t)

	_, bp := buildOutputs(ch, g, nil, nil)

	if len(bp.Teams) != 64 {
		t.Errorf("expected 64 teams, got %d", len(bp.Teams))
	}

	// Should have all 6 rounds
	for _, key := range roundKeyOrder {
		if _, ok := bp.Rounds[key]; !ok {
			t.Errorf("missing round %q", key)
		}
	}

	// R64 should have 32 matchups
	r64 := bp.Rounds["r64"]
	if len(r64.Matchups) != 32 {
		t.Errorf("expected 32 R64 matchups, got %d", len(r64.Matchups))
	}

	// Each R64 matchup should have picks totaling 19
	for _, m := range r64.Matchups {
		total := 0
		for _, p := range m.Picks {
			total += p.Count
		}
		if total != 19 {
			t.Errorf("matchup %s picks total %d, want 19", m.ID, total)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend && go test -run TestSyncProduces -v`
Expected: FAIL — `buildOutputs` not defined

- [ ] **Step 3: Write sync.go with buildOutputs function**

This is the core function. It takes ESPN data + optional existing data and produces the two output files. Implementation must handle:
- Building teams map from proposition outcomes
- Building matchups from propositions (handling both 2-outcome R64 and 4-outcome R32+ formats)
- Aggregating picks per matchup from entry data
- Resolving champion picks via cross-entry correlation
- Merging with existing data (preserving matchups from prior rounds)
- Computing round status

```go
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// buildOutputs produces leaderboard and bracket-picks data from ESPN responses.
// existingLB and existingBP are the previously saved files (nil on first sync).
func buildOutputs(ch *espnChallenge, g *espnGroup, existingLB *LeaderboardData, existingBP *BracketPicksData) (*LeaderboardData, *BracketPicksData) {
	now := time.Now().UTC().Format(time.RFC3339)

	currentPeriod := 1
	if len(ch.Propositions) > 0 {
		currentPeriod = ch.Propositions[0].ScoringPeriodID
	}

	// Build teams from proposition outcomes
	teams := make(map[string]TeamInfo)
	for _, prop := range ch.Propositions {
		for _, o := range prop.PossibleOutcomes {
			logo := ""
			for _, m := range o.Mappings {
				if m.Type == "IMAGE_PRIMARY" {
					logo = m.Value
					break
				}
			}
			teams[o.ID] = TeamInfo{
				Name:   o.Name,
				Abbrev: o.Abbrev,
				Seed:   o.RegionSeed,
				Region: o.RegionID,
				Logo:   logo,
			}
		}
	}

	// Build region+seed → teamId lookup
	type rs struct{ r, s int }
	teamByRS := make(map[rs]string)
	for id, t := range teams {
		teamByRS[rs{t.Region, t.Seed}] = id
	}

	// Build matchups for the current round from propositions
	currentRoundKey := periodToRoundKey(currentPeriod)
	var currentMatchups []MatchupData

	for _, prop := range ch.Propositions {
		m := MatchupData{
			ID:           prop.ID,
			DisplayOrder: prop.DisplayOrder,
			Status:       prop.Status,
			GameTime:     prop.Date,
			Picks:        make(map[string]PickData),
		}
		if len(prop.PossibleOutcomes) > 0 {
			m.Region = prop.PossibleOutcomes[0].RegionID
		}

		if currentPeriod == 1 {
			// R64: 2 outcomes per prop
			for _, o := range prop.PossibleOutcomes {
				if o.MatchupPosition == 1 {
					m.Team1 = o.ID
				} else {
					m.Team2 = o.ID
				}
			}
		} else {
			// R32+: 4 outcomes, but the actual R32 contestants are determined
			// by R64 winners. We need to figure out which teams are actually playing.
			// For now, set team1/team2 based on which teams entries actually pick.
			// The two teams that get picked are the R64 winners.
			pickedTeams := make(map[string]bool)
			for _, entry := range g.Entries {
				for _, pick := range entry.Picks {
					if pick.PropositionID == prop.ID && len(pick.OutcomesPicked) > 0 {
						pickedTeams[pick.OutcomesPicked[0].OutcomeID] = true
					}
				}
			}
			// Sort outcomes by position; game A = pos 1,2 and game B = pos 3,4
			// Team1 = winner of game A (whichever of pos 1,2 is in pickedTeams)
			// Team2 = winner of game B (whichever of pos 3,4 is in pickedTeams)
			outcomes := make(map[int]string)
			for _, o := range prop.PossibleOutcomes {
				outcomes[o.MatchupPosition] = o.ID
			}
			for _, pos := range []int{1, 2} {
				if id, ok := outcomes[pos]; ok && pickedTeams[id] {
					m.Team1 = id
					break
				}
			}
			for _, pos := range []int{3, 4} {
				if id, ok := outcomes[pos]; ok && pickedTeams[id] {
					m.Team2 = id
					break
				}
			}
		}

		currentMatchups = append(currentMatchups, m)
	}

	// Determine winners for completed matchups using first entry's pick results
	if len(g.Entries) > 0 {
		for i, m := range currentMatchups {
			if m.Status != "COMPLETE" {
				continue
			}
			for _, pick := range g.Entries[0].Picks {
				if pick.PropositionID != m.ID || len(pick.OutcomesPicked) == 0 {
					continue
				}
				result := pick.OutcomesPicked[0].Result
				picked := pick.OutcomesPicked[0].OutcomeID
				if result == "CORRECT" {
					currentMatchups[i].Winner = picked
				} else if result == "INCORRECT" {
					if picked == m.Team1 {
						currentMatchups[i].Winner = m.Team2
					} else {
						currentMatchups[i].Winner = m.Team1
					}
				}
				break
			}
		}
	}

	// Aggregate picks per matchup for current round
	for i, m := range currentMatchups {
		picks := make(map[string]PickData)
		for _, entry := range g.Entries {
			for _, pick := range entry.Picks {
				if pick.PropositionID != m.ID || len(pick.OutcomesPicked) == 0 {
					continue
				}
				oid := pick.OutcomesPicked[0].OutcomeID
				pd := picks[oid]
				pd.Count++
				pd.Entries = append(pd.Entries, entry.Name)
				picks[oid] = pd
				break
			}
		}
		currentMatchups[i].Picks = picks
	}

	// Build R64 matchups from R32 propositions when currentPeriod >= 2
	var r64Matchups []MatchupData
	if currentPeriod >= 2 {
		for _, prop := range ch.Propositions {
			outcomes := make(map[int]espnOutcome)
			for _, o := range prop.PossibleOutcomes {
				outcomes[o.MatchupPosition] = o
			}
			region := 0
			if len(prop.PossibleOutcomes) > 0 {
				region = prop.PossibleOutcomes[0].RegionID
			}

			// R64 game A: pos 1 vs pos 2
			if t1, ok := outcomes[1]; ok {
				if t2, ok := outcomes[2]; ok {
					ma := MatchupData{
						ID:           fmt.Sprintf("%s-r64a", prop.ID),
						Region:       region,
						DisplayOrder: prop.DisplayOrder*2 - 1,
						Team1:        t1.ID,
						Team2:        t2.ID,
						Status:       "COMPLETE",
						Picks:        make(map[string]PickData),
					}
					r64Matchups = append(r64Matchups, ma)
				}
			}
			// R64 game B: pos 3 vs pos 4
			if t3, ok := outcomes[3]; ok {
				if t4, ok := outcomes[4]; ok {
					mb := MatchupData{
						ID:           fmt.Sprintf("%s-r64b", prop.ID),
						Region:       region,
						DisplayOrder: prop.DisplayOrder * 2,
						Team1:        t3.ID,
						Team2:        t4.ID,
						Status:       "COMPLETE",
						Picks:        make(map[string]PickData),
					}
					r64Matchups = append(r64Matchups, mb)
				}
			}
		}

		// Determine R64 winners: the team that appears in R32 matchups won R64
		for i, m := range r64Matchups {
			for _, cm := range currentMatchups {
				if cm.Team1 == m.Team1 || cm.Team1 == m.Team2 {
					r64Matchups[i].Winner = cm.Team1
				}
				if cm.Team2 == m.Team1 || cm.Team2 == m.Team2 {
					r64Matchups[i].Winner = cm.Team2
				}
			}
		}
	}

	// Build old-outcome-ID → current-team-ID mapping for pick resolution
	oldOutcomeMap := buildOldOutcomeMap(ch, g, teams, teamByRS, currentPeriod)

	// Aggregate R64 picks from old entry picks when currentPeriod >= 2
	if currentPeriod >= 2 && len(r64Matchups) > 0 {
		propPeriod := make(map[string]int)
		for _, prop := range ch.Propositions {
			propPeriod[prop.ID] = prop.ScoringPeriodID
		}

		for _, entry := range g.Entries {
			for _, pick := range entry.Picks {
				if _, inCurrent := propPeriod[pick.PropositionID]; inCurrent {
					continue
				}
				if len(pick.OutcomesPicked) == 0 {
					continue
				}
				oldOID := pick.OutcomesPicked[0].OutcomeID
				teamID := oldOID
				if mapped, ok := oldOutcomeMap[oldOID]; ok {
					teamID = mapped
				}

				// Find matching R64 matchup
				for i, m := range r64Matchups {
					if m.Team1 == teamID || m.Team2 == teamID {
						pd := r64Matchups[i].Picks[teamID]
						pd.Count++
						pd.Entries = append(pd.Entries, entry.Name)
						r64Matchups[i].Picks[teamID] = pd
						break
					}
				}
			}
		}
	}

	// Build rounds map, merging with existing data
	rounds := make(map[string]Round)
	for _, key := range roundKeyOrder {
		rounds[key] = Round{Status: "future", Matchups: []MatchupData{}}
	}

	// Preserve existing rounds (from prior syncs)
	if existingBP != nil {
		for key, r := range existingBP.Rounds {
			rounds[key] = r
		}
	}

	// Set current round's matchups
	if currentRoundKey != "" {
		rounds[currentRoundKey] = Round{Matchups: currentMatchups}
	}

	// Set R64 matchups if reconstructed
	if currentPeriod >= 2 && len(r64Matchups) > 0 {
		existing := rounds["r64"]
		if len(existing.Matchups) == 0 {
			rounds["r64"] = Round{Matchups: r64Matchups}
		}
	}

	// Compute round statuses
	for key, r := range rounds {
		if len(r.Matchups) == 0 {
			r.Status = "future"
		} else {
			allComplete := true
			anyStarted := false
			for _, m := range r.Matchups {
				if m.Status == "COMPLETE" || m.Status == "PLAYING" {
					anyStarted = true
				}
				if m.Status != "COMPLETE" {
					allComplete = false
				}
			}
			if allComplete {
				r.Status = "complete"
			} else if anyStarted {
				r.Status = "in_progress"
			} else {
				r.Status = "future"
			}
		}
		rounds[key] = r
	}

	// Build champion mapping (reuse existing logic)
	champMap := resolveChampions(ch, g, teams, teamByRS, oldOutcomeMap)

	// Build leaderboard brackets
	var brackets []LeaderboardBracket
	for _, entry := range g.Entries {
		var tiebreaker *float64
		if len(entry.TiebreakAnswers) > 0 {
			t := entry.TiebreakAnswers[0].Answer
			tiebreaker = &t
		}

		champion := ""
		if len(entry.FinalPick.OutcomesPicked) > 0 {
			fpOID := entry.FinalPick.OutcomesPicked[0].OutcomeID
			if tid, ok := champMap[fpOID]; ok {
				champion = tid
			}
		}

		// Build finalFour from picks with periodReached >= 5
		var finalFour []string
		propPeriod := make(map[string]int)
		for _, prop := range ch.Propositions {
			propPeriod[prop.ID] = prop.ScoringPeriodID
		}
		for _, pick := range entry.Picks {
			if pick.PeriodReached >= 5 && len(pick.OutcomesPicked) > 0 {
				oid := pick.OutcomesPicked[0].OutcomeID
				teamID := oid
				if mapped, ok := oldOutcomeMap[oid]; ok {
					teamID = mapped
				} else if _, ok := teams[oid]; !ok {
					continue
				}
				finalFour = append(finalFour, teamID)
			}
		}

		brackets = append(brackets, LeaderboardBracket{
			EntryName:   entry.Name,
			Member:      entry.Member.DisplayName,
			Score:       entry.Score.OverallScore,
			MaxPossible: entry.Score.PossiblePointsMax,
			Rank:        entry.Score.Rank,
			Percentile:  entry.Score.Percentile,
			Eliminated:  entry.Score.Eliminated,
			Tiebreaker:  tiebreaker,
			Champion:    champion,
			FinalFour:   finalFour,
		})
	}

	lb := &LeaderboardData{
		LastUpdated: now,
		GroupName:   g.GroupSettings.Name,
		Teams:       teams,
		Brackets:    brackets,
	}

	bp := &BracketPicksData{
		LastUpdated: now,
		Teams:       teams,
		Rounds:      rounds,
	}

	return lb, bp
}

// resolveChampions maps finalPick outcome IDs to canonical team IDs.
func resolveChampions(ch *espnChallenge, g *espnGroup, teams map[string]TeamInfo, teamByRS map[rs]string, oldOutcomeMap map[string]string) map[string]string {
	bracketOrder := [16]int{1, 16, 8, 9, 5, 12, 4, 13, 6, 11, 3, 14, 7, 10, 2, 15}

	parseFirst := func(id string) (int64, bool) {
		seg := strings.SplitN(id, "-", 2)[0]
		val, err := strconv.ParseInt(seg, 16, 64)
		return val, err == nil
	}

	type candidate struct {
		teams    map[string]bool
		resolved string
	}
	champMap := make(map[string]*candidate)

	for _, entry := range g.Entries {
		if len(entry.FinalPick.OutcomesPicked) == 0 {
			continue
		}
		fpOID := entry.FinalPick.OutcomesPicked[0].OutcomeID

		finalists := make(map[string]bool)
		for _, pick := range entry.Picks {
			if pick.PeriodReached >= 6 && len(pick.OutcomesPicked) > 0 {
				oid := pick.OutcomesPicked[0].OutcomeID
				if mapped, ok := oldOutcomeMap[oid]; ok {
					finalists[mapped] = true
				} else if _, ok := teams[oid]; ok {
					finalists[oid] = true
				}
			}
		}

		if existing, ok := champMap[fpOID]; ok {
			for tid := range existing.teams {
				if !finalists[tid] {
					delete(existing.teams, tid)
				}
			}
		} else {
			champMap[fpOID] = &candidate{teams: finalists}
		}
	}

	used := make(map[string]bool)
	for {
		progress := false
		for _, c := range champMap {
			if c.resolved != "" {
				continue
			}
			for tid := range c.teams {
				if used[tid] {
					delete(c.teams, tid)
				}
			}
			if len(c.teams) == 1 {
				for tid := range c.teams {
					c.resolved = tid
					used[tid] = true
					progress = true
				}
			}
		}
		if !progress {
			break
		}
	}

	// Fallback: hex offset pattern
	var base int64
	var hasBase bool
	for fpOID, c := range champMap {
		if c.resolved == "" {
			continue
		}
		fpVal, ok := parseFirst(fpOID)
		if !ok {
			continue
		}
		team := teams[c.resolved]
		pos := 0
		for i, s := range bracketOrder {
			if s == team.Seed {
				pos = i
				break
			}
		}
		base = fpVal - int64((team.Region-1)*16+pos)
		hasBase = true
		break
	}

	if hasBase {
		for fpOID, c := range champMap {
			if c.resolved != "" {
				continue
			}
			fpVal, ok := parseFirst(fpOID)
			if !ok {
				continue
			}
			offset := int(fpVal - base)
			region := offset/16 + 1
			pos := offset % 16
			if pos >= 0 && pos < 16 && region >= 1 && region <= 4 {
				seed := bracketOrder[pos]
				if tid, ok := teamByRS[rs{region, seed}]; ok {
					c.resolved = tid
				}
			}
		}
	}

	result := make(map[string]string)
	for fpOID, c := range champMap {
		if c.resolved != "" {
			result[fpOID] = c.resolved
		}
	}
	return result
}

// buildOldOutcomeMap maps old pick outcome IDs to current canonical team IDs
// using hex-sorted proposition pairing.
func buildOldOutcomeMap(ch *espnChallenge, g *espnGroup, teams map[string]TeamInfo, teamByRS map[rs]string, currentPeriod int) map[string]string {
	mapping := make(map[string]string)
	if currentPeriod == 1 {
		return mapping
	}

	propPeriod := make(map[string]int)
	for _, prop := range ch.Propositions {
		propPeriod[prop.ID] = prop.ScoringPeriodID
	}
	finalPickProps := make(map[string]bool)
	for _, entry := range g.Entries {
		if entry.FinalPick.PropositionID != "" {
			finalPickProps[entry.FinalPick.PropositionID] = true
		}
	}

	// Collect old R64 prop IDs (min periodReached = 2)
	minPR := make(map[string]int)
	for _, entry := range g.Entries {
		for _, pick := range entry.Picks {
			pid := pick.PropositionID
			if _, inCurrent := propPeriod[pid]; inCurrent {
				continue
			}
			if finalPickProps[pid] {
				continue
			}
			if existing, ok := minPR[pid]; !ok || pick.PeriodReached < existing {
				minPR[pid] = pick.PeriodReached
			}
		}
	}

	oldR64Props := make(map[string]bool)
	for pid, pr := range minPR {
		if pr == 2 {
			oldR64Props[pid] = true
		}
	}

	parseHex := func(id string) int64 {
		seg := strings.SplitN(id, "-", 2)[0]
		val, _ := strconv.ParseInt(seg, 16, 64)
		return val
	}

	oldSorted := make([]string, 0, len(oldR64Props))
	for pid := range oldR64Props {
		oldSorted = append(oldSorted, pid)
	}
	sort.Slice(oldSorted, func(i, j int) bool {
		return parseHex(oldSorted[i]) < parseHex(oldSorted[j])
	})

	r32Sorted := make([]string, 0, len(ch.Propositions))
	for _, prop := range ch.Propositions {
		r32Sorted = append(r32Sorted, prop.ID)
	}
	sort.Slice(r32Sorted, func(i, j int) bool {
		return parseHex(r32Sorted[i]) < parseHex(r32Sorted[j])
	})

	// Build R32 prop outcomes sorted by position
	r32Outcomes := make(map[string][]espnOutcome)
	for _, prop := range ch.Propositions {
		sorted := make([]espnOutcome, len(prop.PossibleOutcomes))
		copy(sorted, prop.PossibleOutcomes)
		sort.Slice(sorted, func(i, j int) bool {
			return sorted[i].MatchupPosition < sorted[j].MatchupPosition
		})
		r32Outcomes[prop.ID] = sorted
	}

	// Pair old R64 props to R32 props
	if len(oldSorted) == len(r32Sorted)*2 {
		for i, r32PID := range r32Sorted {
			outcomes := r32Outcomes[r32PID]
			if len(outcomes) < 4 {
				continue
			}
			oldA := collectOutcomes(g, oldSorted[i*2])
			oldB := collectOutcomes(g, oldSorted[i*2+1])
			mapPair(mapping, oldA, outcomes[0], outcomes[1], teamByRS)
			mapPair(mapping, oldB, outcomes[2], outcomes[3], teamByRS)
		}
	}

	return mapping
}

func collectOutcomes(g *espnGroup, propID string) []string {
	seen := make(map[string]bool)
	for _, entry := range g.Entries {
		for _, pick := range entry.Picks {
			if pick.PropositionID == propID && len(pick.OutcomesPicked) > 0 {
				seen[pick.OutcomesPicked[0].OutcomeID] = true
			}
		}
	}
	result := make([]string, 0, len(seen))
	for id := range seen {
		result = append(result, id)
	}
	sort.Strings(result)
	return result
}

func mapPair(mapping map[string]string, oldIDs []string, a, b espnOutcome, teamByRS map[rs]string) {
	if len(oldIDs) != 2 {
		return
	}
	tA := teamByRS[rs{a.RegionID, a.RegionSeed}]
	tB := teamByRS[rs{b.RegionID, b.RegionSeed}]
	if tA != "" {
		mapping[oldIDs[0]] = tA
	}
	if tB != "" {
		mapping[oldIDs[1]] = tB
	}
}

// Sync orchestration

func saveSnapshot(dir string, challengeData, groupData []byte) error {
	ts := time.Now().UTC().Format("2006-01-02T15-04-05Z")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, ts+"-challenge.json"), challengeData, 0644); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, ts+"-group.json"), groupData, 0644)
}

func loadJSON[T any](path string) (*T, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var v T
	if err := json.Unmarshal(data, &v); err != nil {
		return nil, err
	}
	return &v, nil
}

func saveJSON(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
```

- [ ] **Step 4: Run tests**

Run: `cd backend && go test -run TestSyncProduces -v`
Expected: Both tests PASS

- [ ] **Step 5: Commit**

```
git add backend/sync.go backend/sync_test.go
git commit -m "feat: implement sync logic for leaderboard and bracket-picks output"
```

---

### Task 3: Update Server to Use New Sync

**Files:**
- Modify: `backend/server.go`
- Remove: `backend/process.go`
- Remove: `backend/process_test.go`

- [ ] **Step 1: Update server.go**

Replace `--fetch-only` mode to use the new sync flow. Update HTTP handlers to serve the two new files. Remove references to old `processData` and `BracketData`.

Key changes:
- `--fetch-only`: fetch raw ESPN data, save snapshots, load existing JSONs, call `buildOutputs`, save new JSONs
- Remove `/api/brackets` and `/api/refresh` handlers (frontend loads static JSON files)
- Keep static file serving for `data/*.json`

- [ ] **Step 2: Remove process.go and process_test.go**

```
git rm backend/process.go backend/process_test.go
```

- [ ] **Step 3: Update server_test.go for new types**

Update `TestSaveAndLoadCache` to use new types or remove if no longer relevant.

- [ ] **Step 4: Verify build**

Run: `go build -o .server ./backend/`
Expected: Success

- [ ] **Step 5: Test sync with live ESPN data**

Run: `go run ./backend --fetch-only`
Expected: Creates `data/leaderboard.json`, `data/bracket-picks.json`, and snapshot files in `data/snapshots/`

- [ ] **Step 6: Commit**

```
git add -A
git commit -m "feat: update server to use new sync flow, remove old process.go"
```

---

### Task 4: Update Frontend

**Files:**
- Modify: `js/app.js`

- [ ] **Step 1: Update data loading**

Replace single `data/brackets.json` fetch with two fetches:
```js
const [lbRes, bpRes] = await Promise.all([
  fetch('data/leaderboard.json?t=' + Date.now()),
  fetch('data/bracket-picks.json?t=' + Date.now())
]);
const LB_DATA = await lbRes.json();
const BP_DATA = await bpRes.json();
```

- [ ] **Step 2: Update renderLeaderboard**

Use `LB_DATA` instead of `DATA`. Build eliminated teams from `BP_DATA.rounds` matchup winners. Access bracket fields directly (no picks arrays).

- [ ] **Step 3: Update renderConsensus (bracket picks tab)**

Use `BP_DATA.rounds` directly. Replace client-side pick counting with pre-aggregated `matchup.picks[teamId].count`. Use `round.status === "complete"` for pill auto-advance.

- [ ] **Step 4: Update openBracketModal**

Use `LB_DATA` for bracket data, `BP_DATA` for eliminated teams and matchup details.

- [ ] **Step 5: Update matchup detail panel**

Use `matchup.picks[teamId].entries` for the list of who picked each team.

- [ ] **Step 6: Verify locally**

Run: `make start`
Navigate to http://localhost:8000, verify leaderboard, bracket picks, and modal all work.

- [ ] **Step 7: Commit**

```
git add js/app.js
git commit -m "feat: update frontend to load leaderboard.json and bracket-picks.json"
```

---

### Task 5: Update CI/CD and Config

**Files:**
- Modify: `.github/workflows/update-brackets.yml`
- Modify: `.gitignore`
- Modify: `Makefile`
- Remove: `data/brackets.json`

- [ ] **Step 1: Update workflow to check new files**

Change the git diff check from `data/brackets.json` to `data/leaderboard.json data/bracket-picks.json`.

- [ ] **Step 2: Update .gitignore**

Add `data/snapshots/` to .gitignore (raw ESPN snapshots should not be committed).

- [ ] **Step 3: Update Makefile**

Update `load-real` target. Update or remove `load-*` scenario targets if needed.

- [ ] **Step 4: Remove old brackets.json**

```
git rm data/brackets.json
```

- [ ] **Step 5: Commit**

```
git add -A
git commit -m "chore: update CI, gitignore, and Makefile for new data format"
```

---

### Task 6: Update Tests

**Files:**
- Modify: `backend/sync_test.go`
- Modify: `backend/server_test.go`
- Modify: `backend/testdata_gen_test.go`
- Modify: `tests/smoke.spec.ts`

- [ ] **Step 1: Ensure sync tests cover edge cases**

Add tests for: merge with existing data, R32+ proposition format, round status computation.

- [ ] **Step 2: Update server_test.go**

Update or replace tests for new JSON types.

- [ ] **Step 3: Update scenario test generator**

Update `testdata_gen_test.go` to produce `leaderboard.json` and `bracket-picks.json` instead of `brackets.json`.

- [ ] **Step 4: Update Playwright smoke tests**

Update `tests/smoke.spec.ts` to copy both JSON files per scenario and verify the UI still renders correctly.

- [ ] **Step 5: Run all tests**

Run: `make test && make test-e2e`
Expected: All tests pass.

- [ ] **Step 6: Commit**

```
git add -A
git commit -m "test: update all tests for new data layer format"
```
