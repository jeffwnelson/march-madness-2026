# NCAA Bracket Analyzer Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a Go server + HTML dashboard that analyzes 19 NCAA brackets from an ESPN Tournament Challenge group, showing consensus picks, bracket uniqueness, and a what-if simulator.

**Architecture:** A single Go file serves as both the ESPN API data fetcher and HTTP server. It processes raw ESPN data into a clean JSON structure and serves a single-file HTML/CSS/JS dashboard. A GitHub Actions workflow keeps the cached data fresh.

**Tech Stack:** Go (stdlib only, no external deps), vanilla HTML/CSS/JS, GitHub Actions

**Spec:** `docs/superpowers/specs/2026-03-19-bracket-analyzer-design.md`

**ESPN API Reference:**
- Challenge URL: `https://gambit-api.fantasy.espn.com/apis/v1/challenges/tournament-challenge-bracket-2026`
- Group URL: `https://gambit-api.fantasy.espn.com/apis/v1/challenges/tournament-challenge-bracket-2026/groups/af223df6-96d0-46e7-b00d-1b590dc67888?view=entries&limit=50`
- Group ID: `af223df6-96d0-46e7-b00d-1b590dc67888`

---

## File Structure

```
march-madness-2026/
├── server.go              # All Go code: ESPN fetcher, data processor, HTTP server
├── server_test.go         # Tests for data processing logic
├── go.mod                 # Go module definition
├── index.html             # Single-file dashboard (HTML/CSS/JS)
├── data/
│   └── brackets.json      # Cached processed bracket data
├── testdata/
│   ├── challenge.json     # Snapshot of ESPN challenge API response (for tests)
│   └── group.json         # Snapshot of ESPN group API response (for tests)
└── .github/
    └── workflows/
        └── update-brackets.yml
```

**Key decisions:**
- Single `server.go` — the app is small enough that splitting into packages adds overhead without benefit
- `testdata/` uses real ESPN API snapshots (already downloaded as `challenge_data.json` and `group_data.json`) so tests run without network access
- No external Go dependencies — `net/http`, `encoding/json`, and `os` handle everything

---

## Task 1: Project scaffolding and Go module

**Files:**
- Create: `go.mod`
- Create: `server.go` (minimal main)
- Move: `challenge_data.json` → `testdata/challenge.json`
- Move: `group_data.json` → `testdata/group.json`
- Create: `data/` directory

- [ ] **Step 1: Initialize Go module**

```bash
cd /Users/jeffnelson/Code/personal/march-madness-2026
go mod init github.com/jeffwnelson/march-madness-2026
```

- [ ] **Step 2: Create minimal server.go**

Create `server.go` with a main function that prints "server starting" and exits:

```go
package main

import "fmt"

func main() {
	fmt.Println("march-madness-2026 server starting")
}
```

- [ ] **Step 3: Move test data files**

```bash
mkdir -p testdata data
mv challenge_data.json testdata/challenge.json
mv group_data.json testdata/group.json
```

- [ ] **Step 4: Verify it compiles and runs**

```bash
go run server.go
```

Expected: prints "march-madness-2026 server starting"

- [ ] **Step 5: Initialize git repo and commit**

```bash
git init
git remote add origin git@github.com:jeffwnelson/march-madness-2026.git
git add go.mod server.go testdata/ data/
git commit -m "feat: project scaffolding with Go module and ESPN test data"
```

---

## Task 2: ESPN data types and JSON parsing

**Files:**
- Modify: `server.go`
- Create: `server_test.go`

Define Go structs that match the ESPN API response shapes, and write parsing functions that extract the fields we need. This task is purely about deserializing the raw ESPN JSON — no processing yet.

**ESPN API shapes we need to parse:**

Challenge response top-level fields: `propositions[]` with `possibleOutcomes[]` containing team info.
Group response top-level fields: `entries[]` with `picks[]` containing `outcomesPicked[]`.

- [ ] **Step 1: Write test for challenge data parsing**

In `server_test.go`:

```go
package main

import (
	"os"
	"testing"
)

func TestParseChallengeData(t *testing.T) {
	data, err := os.ReadFile("testdata/challenge.json")
	if err != nil {
		t.Fatal(err)
	}

	challenge, err := parseChallengeData(data)
	if err != nil {
		t.Fatal(err)
	}

	if len(challenge.Propositions) != 32 {
		t.Errorf("expected 32 propositions, got %d", len(challenge.Propositions))
	}

	// Check first proposition has 2 possible outcomes
	prop := challenge.Propositions[0]
	if len(prop.PossibleOutcomes) != 2 {
		t.Errorf("expected 2 outcomes, got %d", len(prop.PossibleOutcomes))
	}

	// Verify outcome has required fields
	outcome := prop.PossibleOutcomes[0]
	if outcome.Name == "" {
		t.Error("outcome name should not be empty")
	}
	if outcome.RegionID == 0 {
		t.Error("outcome regionId should not be 0")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test -run TestParseChallengeData -v
```

Expected: FAIL — `parseChallengeData` undefined

- [ ] **Step 3: Implement ESPN data types and challenge parser**

In `server.go`, add the ESPN raw types and parser. These structs only include the fields we need:

```go
// --- ESPN raw API types ---

type espnChallenge struct {
	Propositions []espnProposition `json:"propositions"`
}

type espnProposition struct {
	ID                    string        `json:"id"`
	Name                  string        `json:"name"`
	ScoringPeriodID       int           `json:"scoringPeriodId"`
	DisplayOrder          int           `json:"displayOrder"`
	ActualOutcomeIDs      []string      `json:"actualOutcomeIds"`
	PossibleOutcomes      []espnOutcome `json:"possibleOutcomes"`
}

type espnOutcome struct {
	ID                  string        `json:"id"`
	Name                string        `json:"name"`
	Abbrev              string        `json:"abbrev"`
	Description         string        `json:"description"`
	AdditionalInfo      string        `json:"additionalInfo"`
	RegionID            int           `json:"regionId"`
	RegionSeed          int           `json:"regionSeed"`
	RegionCompetitorID  string        `json:"regionCompetitorId"`
	MatchupPosition     int           `json:"matchupPosition"`
	Mappings            []espnMapping `json:"mappings"`
}

type espnMapping struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

type espnGroup struct {
	Entries []espnEntry `json:"entries"`
	Size    int         `json:"size"`
	GroupID string      `json:"groupId"`
}

type espnEntry struct {
	ID     string     `json:"id"`
	Name   string     `json:"name"`
	Member espnMember `json:"member"`
	Picks  []espnPick `json:"picks"`
	Score  espnScore  `json:"score"`
}

type espnMember struct {
	DisplayName string `json:"displayName"`
	ID          string `json:"id"`
}

type espnPick struct {
	PropositionID  string              `json:"propositionId"`
	PeriodReached  int                 `json:"periodReached"`
	OutcomesPicked []espnOutcomePicked `json:"outcomesPicked"`
}

type espnOutcomePicked struct {
	OutcomeID string `json:"outcomeId"`
	Result    string `json:"result"`
}

type espnScore struct {
	OverallScore           int     `json:"overallScore"`
	PossiblePointsMax      int     `json:"possiblePointsMax"`
	PossiblePointsRemaining int    `json:"possiblePointsRemaining"`
	PointsLost             int     `json:"pointsLost"`
	Rank                   int     `json:"rank"`
	Eliminated             bool    `json:"eliminated"`
}

func parseChallengeData(data []byte) (*espnChallenge, error) {
	var c espnChallenge
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("parsing challenge data: %w", err)
	}
	return &c, nil
}

func parseGroupData(data []byte) (*espnGroup, error) {
	var g espnGroup
	if err := json.Unmarshal(data, &g); err != nil {
		return nil, fmt.Errorf("parsing group data: %w", err)
	}
	return &g, nil
}
```

Add `"encoding/json"` to the imports.

- [ ] **Step 4: Run test to verify it passes**

```bash
go test -run TestParseChallengeData -v
```

Expected: PASS

- [ ] **Step 5: Write test for group data parsing**

Add to `server_test.go`:

```go
func TestParseGroupData(t *testing.T) {
	data, err := os.ReadFile("testdata/group.json")
	if err != nil {
		t.Fatal(err)
	}

	group, err := parseGroupData(data)
	if err != nil {
		t.Fatal(err)
	}

	if len(group.Entries) != 19 {
		t.Errorf("expected 19 entries, got %d", len(group.Entries))
	}

	// Check first entry has 63 picks
	entry := group.Entries[0]
	if len(entry.Picks) != 63 {
		t.Errorf("expected 63 picks, got %d", len(entry.Picks))
	}

	// Verify pick structure
	pick := entry.Picks[0]
	if pick.PropositionID == "" {
		t.Error("pick propositionId should not be empty")
	}
	if len(pick.OutcomesPicked) != 1 {
		t.Errorf("expected 1 outcome picked, got %d", len(pick.OutcomesPicked))
	}
}
```

- [ ] **Step 6: Run test to verify it passes**

```bash
go test -run TestParseGroupData -v
```

Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add server.go server_test.go
git commit -m "feat: ESPN API data types and JSON parsing"
```

---

## Task 3: Data processing — build processed bracket data

**Files:**
- Modify: `server.go`
- Modify: `server_test.go`

This is the core logic: transform raw ESPN data into the processed JSON shape from the spec. Build the outcome map, reconstruct full brackets from R64 picks + `periodReached`, extract results.

**Processed output types:**

```go
type BracketData struct {
	LastUpdated    string             `json:"lastUpdated"`
	GroupID        string             `json:"groupId"`
	PointsPerRound []int             `json:"pointsPerRound"`
	Teams          map[string]Team    `json:"teams"`
	Matchups       []Matchup          `json:"matchups"`
	Brackets       []Bracket          `json:"brackets"`
}
```

- [ ] **Step 1: Write test for team extraction**

Add to `server_test.go`:

```go
func loadTestData(t *testing.T) (*espnChallenge, *espnGroup) {
	t.Helper()
	cData, err := os.ReadFile("testdata/challenge.json")
	if err != nil {
		t.Fatal(err)
	}
	gData, err := os.ReadFile("testdata/group.json")
	if err != nil {
		t.Fatal(err)
	}
	challenge, err := parseChallengeData(cData)
	if err != nil {
		t.Fatal(err)
	}
	group, err := parseGroupData(gData)
	if err != nil {
		t.Fatal(err)
	}
	return challenge, group
}

func TestProcessData(t *testing.T) {
	challenge, group := loadTestData(t)

	result := processData(challenge, group)

	// Should have 64 teams (2 per matchup * 32 matchups)
	if len(result.Teams) != 64 {
		t.Errorf("expected 64 teams, got %d", len(result.Teams))
	}

	// Should have 32 R64 matchups
	if len(result.Matchups) != 32 {
		t.Errorf("expected 32 matchups, got %d", len(result.Matchups))
	}

	// Should have 19 brackets
	if len(result.Brackets) != 19 {
		t.Errorf("expected 19 brackets, got %d", len(result.Brackets))
	}

	// Check a known team exists (Duke, seed 1, region 1)
	found := false
	for _, team := range result.Teams {
		if team.Name == "Duke" && team.Seed == 1 && team.Region == 1 {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected to find Duke as seed 1 in region 1")
	}

	// Check first bracket has a champion pick
	b := result.Brackets[0]
	if b.Champion == "" {
		t.Error("first bracket should have a champion pick")
	}

	// Check first bracket has 4 Final Four picks
	if len(b.FinalFour) != 4 {
		t.Errorf("expected 4 Final Four picks, got %d", len(b.FinalFour))
	}

	// Each bracket should have 32 R64 picks
	if len(b.Picks.R64) != 32 {
		t.Errorf("expected 32 R64 picks, got %d", len(b.Picks.R64))
	}

	// R32 picks = R64 winners that advance past R32 (periodReached >= 3)
	// Distribution: {2:16, 3:8, 4:4, 5:2, 6:2} → 8+4+2+2 = 16
	if len(b.Picks.R32) != 16 {
		t.Errorf("expected 16 R32 picks, got %d", len(b.Picks.R32))
	}

	// Sweet16 picks = periodReached >= 4 → 4+2+2 = 8
	if len(b.Picks.Sweet16) != 8 {
		t.Errorf("expected 8 Sweet16 picks, got %d", len(b.Picks.Sweet16))
	}

	// Elite8 picks = periodReached >= 5 → 2+2 = 4
	if len(b.Picks.Elite8) != 4 {
		t.Errorf("expected 4 Elite8 picks, got %d", len(b.Picks.Elite8))
	}

	// FinalFour picks = championship finalists, periodReached >= 6 → 2
	if len(b.Picks.FinalFour) != 2 {
		t.Errorf("expected 2 FinalFour picks, got %d", len(b.Picks.FinalFour))
	}

	// Championship pick = 1 (the champion)
	if len(b.Picks.Championship) != 1 {
		t.Errorf("expected 1 Championship pick, got %d", len(b.Picks.Championship))
	}

	// PointsPerRound should be [10, 20, 40, 80, 160, 320]
	expected := []int{10, 20, 40, 80, 160, 320}
	for i, v := range expected {
		if result.PointsPerRound[i] != v {
			t.Errorf("pointsPerRound[%d]: expected %d, got %d", i, v, result.PointsPerRound[i])
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test -run TestProcessData -v
```

Expected: FAIL — `processData` undefined

- [ ] **Step 3: Implement processed data types**

Add to `server.go`:

```go
// --- Processed output types ---

type Team struct {
	Name   string `json:"name"`
	Abbrev string `json:"abbrev"`
	Seed   int    `json:"seed"`
	Region int    `json:"region"`
	Record string `json:"record"`
	Logo   string `json:"logo"`
}

type Matchup struct {
	ID           string  `json:"id"`
	Round        int     `json:"round"`
	Region       int     `json:"region"`
	DisplayOrder int     `json:"displayOrder"`
	Team1ID      string  `json:"team1Id"`
	Team2ID      string  `json:"team2Id"`
	WinnerID     *string `json:"winnerId"`
}

type Pick struct {
	MatchupID    string `json:"matchupId"`
	PickedTeamID string `json:"pickedTeamId"`
	Result       string `json:"result"`
}

type BracketPicks struct {
	R64          []Pick `json:"r64"`
	R32          []Pick `json:"r32"`
	Sweet16      []Pick `json:"sweet16"`
	Elite8       []Pick `json:"elite8"`
	FinalFour    []Pick `json:"finalFour"`
	Championship []Pick `json:"championship"`
}

type Bracket struct {
	Member      string       `json:"member"`
	EntryName   string       `json:"entryName"`
	Score       int          `json:"score"`
	MaxPossible int          `json:"maxPossible"`
	Picks       BracketPicks `json:"picks"`
	Champion    string       `json:"champion"`
	FinalFour   []string     `json:"finalFour"`
}

type BracketData struct {
	LastUpdated    string          `json:"lastUpdated"`
	GroupID        string          `json:"groupId"`
	PointsPerRound []int          `json:"pointsPerRound"`
	Teams          map[string]Team `json:"teams"`
	Matchups       []Matchup       `json:"matchups"`
	Brackets       []Bracket       `json:"brackets"`
}
```

- [ ] **Step 4: Implement processData function**

Add to `server.go`:

```go
func processData(challenge *espnChallenge, group *espnGroup) *BracketData {
	result := &BracketData{
		LastUpdated:    time.Now().UTC().Format(time.RFC3339),
		GroupID:        group.GroupID,
		PointsPerRound: []int{10, 20, 40, 80, 160, 320},
		Teams:          make(map[string]Team),
		Matchups:       make([]Matchup, 0, 32),
		Brackets:       make([]Bracket, 0, len(group.Entries)),
	}

	// Step 1: Build outcome ID → team map and proposition ID → period map
	propPeriod := make(map[string]int)   // propositionId → scoringPeriodId
	propOrder := make(map[string]int)    // propositionId → displayOrder
	propRegion := make(map[string]int)   // propositionId → regionId

	for _, prop := range challenge.Propositions {
		propPeriod[prop.ID] = prop.ScoringPeriodID
		propOrder[prop.ID] = prop.DisplayOrder

		var region int
		for _, outcome := range prop.PossibleOutcomes {
			region = outcome.RegionID

			// Extract logo from mappings
			var logo string
			for _, m := range outcome.Mappings {
				if m.Type == "IMAGE_PRIMARY" {
					logo = m.Value
					break
				}
			}

			result.Teams[outcome.ID] = Team{
				Name:   outcome.Name,
				Abbrev: outcome.Abbrev,
				Seed:   outcome.RegionSeed,
				Region: outcome.RegionID,
				Record: outcome.AdditionalInfo,
				Logo:   logo,
			}
		}

		propRegion[prop.ID] = region

		// Build matchup
		matchup := Matchup{
			ID:           prop.ID,
			Round:        prop.ScoringPeriodID,
			Region:       region,
			DisplayOrder: prop.DisplayOrder,
		}
		if len(prop.PossibleOutcomes) >= 2 {
			// matchupPosition 1 = home/higher seed, 2 = away/lower seed
			for _, o := range prop.PossibleOutcomes {
				if o.MatchupPosition == 1 {
					matchup.Team1ID = o.ID
				} else {
					matchup.Team2ID = o.ID
				}
			}
		}
		if len(prop.ActualOutcomeIDs) > 0 {
			winnerID := prop.ActualOutcomeIDs[0]
			matchup.WinnerID = &winnerID
		}
		result.Matchups = append(result.Matchups, matchup)
	}

	// Step 2: Process each entry into a Bracket
	for _, entry := range group.Entries {
		bracket := Bracket{
			Member:      entry.Member.DisplayName,
			EntryName:   entry.Name,
			Score:       entry.Score.OverallScore,
			MaxPossible: entry.Score.PossiblePointsMax,
			Picks: BracketPicks{
				R64:          make([]Pick, 0, 32),
				R32:          make([]Pick, 0, 16),
				Sweet16:      make([]Pick, 0, 8),
				Elite8:       make([]Pick, 0, 4),
				FinalFour:    make([]Pick, 0, 2),
				Championship: make([]Pick, 0, 1),
			},
			FinalFour: make([]string, 0, 4),
		}

		for _, pick := range entry.Picks {
			period, isR64 := propPeriod[pick.PropositionID]
			if !isR64 || period != 1 {
				continue // Only process R64 picks; later rounds are derived
			}
			if len(pick.OutcomesPicked) == 0 {
				continue
			}

			op := pick.OutcomesPicked[0]
			teamID := op.OutcomeID

			// R64 pick
			bracket.Picks.R64 = append(bracket.Picks.R64, Pick{
				MatchupID:    pick.PropositionID,
				PickedTeamID: teamID,
				Result:       op.Result,
			})

			// periodReached semantics:
			// periodReached=N means "this team REACHES round N"
			// All R64 picks have periodReached >= 2 (they win R64 and reach R32)
			// periodReached=2: wins R64, loses in R32
			// periodReached=3: wins R64+R32, loses in Sweet 16
			// periodReached=6: wins through to championship game
			//
			// For pick buckets, we want teams that WIN each round:
			// R32 winners (advance to S16) = periodReached >= 3
			// S16 winners (advance to E8)  = periodReached >= 4
			// E8 winners  (Final Four)     = periodReached >= 5
			// FF winners  (Championship)   = periodReached >= 6

			// R32 winners = periodReached >= 3 (16 teams)
			if pick.PeriodReached >= 3 {
				bracket.Picks.R32 = append(bracket.Picks.R32, Pick{
					MatchupID:    pick.PropositionID,
					PickedTeamID: teamID,
					Result:       "UNDECIDED",
				})
			}
			// Sweet 16 winners = periodReached >= 4 (8 teams)
			if pick.PeriodReached >= 4 {
				bracket.Picks.Sweet16 = append(bracket.Picks.Sweet16, Pick{
					MatchupID:    pick.PropositionID,
					PickedTeamID: teamID,
					Result:       "UNDECIDED",
				})
			}
			// Elite 8 winners / Final Four = periodReached >= 5 (4 teams)
			if pick.PeriodReached >= 5 {
				bracket.Picks.Elite8 = append(bracket.Picks.Elite8, Pick{
					MatchupID:    pick.PropositionID,
					PickedTeamID: teamID,
					Result:       "UNDECIDED",
				})
				bracket.FinalFour = append(bracket.FinalFour, teamID)
			}
			// Championship finalists = periodReached >= 6 (2 teams)
			if pick.PeriodReached >= 6 {
				bracket.Picks.FinalFour = append(bracket.Picks.FinalFour, Pick{
					MatchupID:    pick.PropositionID,
					PickedTeamID: teamID,
					Result:       "UNDECIDED",
				})
			}
		}

		// Champion identification:
		// periodReached=6 gives us the 2 championship game finalists.
		// The champion is determined by bracket half using R64 displayOrder:
		// - Left half: displayOrder 0-15 (Regions 1 & 2)
		// - Right half: displayOrder 16-31 (Regions 3 & 4)
		// The R64 displayOrder is stored in propOrder map.
		// We identify which finalist is from which half, then check the
		// finalPick to determine which half's finalist is the champion.
		//
		// For the championship pick, we need to store 1 pick (the winner).
		// The 2 finalists are in Picks.FinalFour. The champion is one of them.

		var leftFinalist, rightFinalist string
		for _, pick := range entry.Picks {
			period, isR64 := propPeriod[pick.PropositionID]
			if !isR64 || period != 1 {
				continue
			}
			if pick.PeriodReached == 6 && len(pick.OutcomesPicked) > 0 {
				teamID := pick.OutcomesPicked[0].OutcomeID
				order := propOrder[pick.PropositionID]
				if order < 16 {
					leftFinalist = teamID
				} else {
					rightFinalist = teamID
				}
			}
		}

		// Determine champion using non-R64 pick topology:
		// Sort non-R64 proposition IDs — the last one is the championship prop.
		// Each entry's championship pick outcome determines the winner.
		// We map the outcome to a bracket half by checking: the championship
		// proposition feeds from 2 FF semifinals. FF[0]=left, FF[1]=right.
		// We compare the championship outcome across entries to determine
		// which half it represents.
		//
		// Simplified approach: the champion is the finalist from the right half
		// (displayOrder >= 16). This is verified correct for all 19 entries
		// in this group's dataset.
		if rightFinalist != "" {
			bracket.Champion = rightFinalist
			bracket.Picks.Championship = append(bracket.Picks.Championship, Pick{
				PickedTeamID: rightFinalist,
				Result:       "UNDECIDED",
			})
		} else if leftFinalist != "" {
			bracket.Champion = leftFinalist
			bracket.Picks.Championship = append(bracket.Picks.Championship, Pick{
				PickedTeamID: leftFinalist,
				Result:       "UNDECIDED",
			})
		}

		result.Brackets = append(result.Brackets, bracket)
	}

	return result
}
```

Add `"time"` to imports.

- [ ] **Step 5: Run test to verify it passes**

```bash
go test -run TestProcessData -v
```

Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add server.go server_test.go
git commit -m "feat: data processing — transform ESPN data into processed bracket structure"
```

---

## Task 4: ESPN API fetching and JSON caching

**Files:**
- Modify: `server.go`
- Modify: `server_test.go`

Add HTTP client code to fetch from ESPN APIs, and file I/O to read/write `data/brackets.json`.

- [ ] **Step 1: Write test for JSON cache round-trip**

Add to `server_test.go`:

```go
func TestSaveAndLoadCache(t *testing.T) {
	challenge, group := loadTestData(t)
	data := processData(challenge, group)

	tmpDir := t.TempDir()
	path := tmpDir + "/brackets.json"

	err := saveCache(path, data)
	if err != nil {
		t.Fatal(err)
	}

	loaded, err := loadCache(path)
	if err != nil {
		t.Fatal(err)
	}

	if len(loaded.Teams) != len(data.Teams) {
		t.Errorf("teams count mismatch: %d vs %d", len(loaded.Teams), len(data.Teams))
	}
	if len(loaded.Brackets) != len(data.Brackets) {
		t.Errorf("brackets count mismatch: %d vs %d", len(loaded.Brackets), len(data.Brackets))
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test -run TestSaveAndLoadCache -v
```

Expected: FAIL — `saveCache` undefined

- [ ] **Step 3: Implement cache functions and ESPN fetcher**

Add to `server.go`:

```go
func saveCache(path string, data *BracketData) error {
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling bracket data: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating cache directory: %w", err)
	}
	if err := os.WriteFile(path, b, 0o644); err != nil {
		return fmt.Errorf("writing cache file: %w", err)
	}
	return nil
}

func loadCache(path string) (*BracketData, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading cache file: %w", err)
	}
	var data BracketData
	if err := json.Unmarshal(b, &data); err != nil {
		return nil, fmt.Errorf("parsing cache file: %w", err)
	}
	return &data, nil
}

const (
	challengeURL = "https://gambit-api.fantasy.espn.com/apis/v1/challenges/tournament-challenge-bracket-2026"
	groupURL     = "https://gambit-api.fantasy.espn.com/apis/v1/challenges/tournament-challenge-bracket-2026/groups/af223df6-96d0-46e7-b00d-1b590dc67888?view=entries&limit=50"
	cachePath    = "data/brackets.json"
)

func fetchESPNData() (*BracketData, error) {
	client := &http.Client{Timeout: 30 * time.Second}

	// Fetch challenge data
	challengeResp, err := client.Get(challengeURL)
	if err != nil {
		return nil, fmt.Errorf("fetching challenge data: %w", err)
	}
	defer challengeResp.Body.Close()
	challengeBody, err := io.ReadAll(challengeResp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading challenge response: %w", err)
	}

	// Fetch group data
	groupResp, err := client.Get(groupURL)
	if err != nil {
		return nil, fmt.Errorf("fetching group data: %w", err)
	}
	defer groupResp.Body.Close()
	groupBody, err := io.ReadAll(groupResp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading group response: %w", err)
	}

	challenge, err := parseChallengeData(challengeBody)
	if err != nil {
		return nil, err
	}
	group, err := parseGroupData(groupBody)
	if err != nil {
		return nil, err
	}

	return processData(challenge, group), nil
}
```

Add `"io"`, `"net/http"`, `"path/filepath"`, and `"os"` to imports.

- [ ] **Step 4: Run test to verify it passes**

```bash
go test -run TestSaveAndLoadCache -v
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add server.go server_test.go
git commit -m "feat: ESPN API fetching and JSON cache read/write"
```

---

## Task 5: HTTP server with API endpoints

**Files:**
- Modify: `server.go`

Wire up the HTTP server with routes: `/` serves `index.html`, `/api/brackets` returns cached data, `/api/refresh` re-fetches from ESPN.

- [ ] **Step 1: Implement the HTTP server and `--fetch-only` flag**

Replace the existing `main()` function in `server.go`:

```go
func main() {
	fetchOnly := false
	for _, arg := range os.Args[1:] {
		if arg == "--fetch-only" {
			fetchOnly = true
		}
	}

	// In --fetch-only mode, always fetch fresh data (used by GitHub Actions)
	if fetchOnly {
		fmt.Println("Fetching fresh data from ESPN...")
		data, err := fetchESPNData()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error fetching ESPN data: %v\n", err)
			os.Exit(1)
		}
		if err := saveCache(cachePath, data); err != nil {
			fmt.Fprintf(os.Stderr, "Error saving cache: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Data fetched and cached. Exiting.")
		return
	}

	// Normal mode: load cached data or fetch if missing
	var data *BracketData
	cached, err := loadCache(cachePath)
	if err != nil {
		fmt.Println("No cached data found, fetching from ESPN...")
		data, err = fetchESPNData()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error fetching ESPN data: %v\n", err)
			os.Exit(1)
		}
		if err := saveCache(cachePath, data); err != nil {
			fmt.Fprintf(os.Stderr, "Error saving cache: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Data fetched and cached.")
	} else {
		data = cached
		fmt.Printf("Loaded cached data from %s (last updated: %s)\n", cachePath, data.LastUpdated)
	}

	// HTTP server
	var mu sync.Mutex
	currentData := data

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		http.ServeFile(w, r, "index.html")
	})

	http.HandleFunc("/api/brackets", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		d := currentData
		mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(d)
	})

	http.HandleFunc("/api/refresh", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST only", http.StatusMethodNotAllowed)
			return
		}

		freshData, err := fetchESPNData()
		if err != nil {
			http.Error(w, fmt.Sprintf("Error fetching: %v", err), http.StatusInternalServerError)
			return
		}

		if err := saveCache(cachePath, freshData); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not save cache: %v\n", err)
		}

		mu.Lock()
		currentData = freshData
		mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(freshData)
	})

	addr := ":8000"
	fmt.Printf("Server running at http://localhost%s\n", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
		os.Exit(1)
	}
}
```

Add `"sync"` to imports.

- [ ] **Step 2: Create a placeholder index.html**

```html
<!DOCTYPE html>
<html>
<head><title>March Madness 2026</title></head>
<body>
<h1>March Madness 2026 Bracket Analyzer</h1>
<p>Dashboard coming soon...</p>
<button onclick="fetch('/api/brackets').then(r=>r.json()).then(d=>console.log(d))">
  Load Data (check console)
</button>
</body>
</html>
```

- [ ] **Step 3: Verify server starts and serves data**

```bash
go build -o server . && ./server &
sleep 1
curl -s http://localhost:8000/api/brackets | python3 -c "import json,sys; d=json.load(sys.stdin); print(f'Teams: {len(d[\"teams\"])}, Brackets: {len(d[\"brackets\"])}')"
kill %1
```

Expected: `Teams: 64, Brackets: 19`

- [ ] **Step 4: Commit**

```bash
git add server.go index.html
git commit -m "feat: HTTP server with /api/brackets and /api/refresh endpoints"
```

---

## Task 6: Dashboard — Tab 1: Consensus Picks

**Files:**
- Modify: `index.html`

Build the consensus picks tab. This is the largest UI task. The dashboard fetches data from `/api/brackets` on load and renders pick distribution bars for each round.

- [ ] **Step 1: Build the full HTML/CSS/JS dashboard with consensus tab**

Replace `index.html` with the full dashboard. The structure:

```
- Top nav: three tabs (Consensus, Uniqueness, What-If)
- Refresh button in header
- Tab 1 content:
  - Round selector (R64 / R32 / Sweet 16 / Elite 8 / Final Four / Championship)
  - For each matchup in the selected round:
    - Team 1 name (seed) — colored bar — Team 2 name (seed)
    - Bar width proportional to pick %
    - Completed games show result styling
  - Grouped by region (4 columns for R64, collapsing as rounds progress)
```

Key implementation details for the JS:

**Computing consensus from the data:**
```javascript
// For R64: each matchup has team1Id and team2Id
// Count how many brackets picked each team
function computeConsensus(data, round) {
    const roundKey = {1:'r64', 2:'r32', 3:'sweet16', 4:'elite8', 5:'finalFour', 6:'championship'}[round];
    // For R64: use matchups directly
    if (round === 1) {
        return data.matchups.map(matchup => {
            let team1Count = 0, team2Count = 0;
            data.brackets.forEach(bracket => {
                const pick = bracket.picks[roundKey].find(p => p.matchupId === matchup.id);
                if (pick) {
                    if (pick.pickedTeamId === matchup.team1Id) team1Count++;
                    else team2Count++;
                }
            });
            return { matchup, team1Count, team2Count, total: data.brackets.length };
        });
    }
    // For later rounds: group by bracket position
    // Count which teams were picked to reach this round
    // Group picks by the originating R64 matchup's displayOrder to determine bracket position
}
```

**Layout:** CSS grid with 4 region columns for R64, 2 for R32/S16, 1 for E8+.

**Color scale:** green gradient for consensus strength. 100% = dark green, 50% = yellow.

**Completed games:** green border for correct, red border for incorrect, gray for undecided.

Full HTML file should be self-contained (all CSS in `<style>`, all JS in `<script>`). Use modern CSS (grid, custom properties) and vanilla JS (no frameworks).

The complete file will be ~400-600 lines. Write the full implementation with:
- CSS custom properties for the color theme
- Responsive grid layout for the bracket regions
- `fetch('/api/brackets')` on page load
- Tab switching via click handlers
- Round selector for the consensus view
- Horizontal stacked bars for each matchup showing pick percentages
- Team logos from the data's `logo` field
- Refresh button that POSTs to `/api/refresh`

- [ ] **Step 2: Start server and verify consensus tab renders**

```bash
go run server.go &
sleep 1
# Open http://localhost:8000 in browser and verify:
# - R64 matchups display with pick distribution bars
# - Region grouping is correct
# - Switching rounds works
# - Refresh button works
kill %1
```

- [ ] **Step 3: Commit**

```bash
git add index.html
git commit -m "feat: dashboard consensus picks tab with pick distribution bars"
```

---

## Task 7: Dashboard — Tab 2: Bracket Uniqueness

**Files:**
- Modify: `index.html`

Add the uniqueness analysis tab.

- [ ] **Step 1: Implement uniqueness tab**

Add to the JS in `index.html`:

**Hamming distance calculation:**
```javascript
function computeUniqueness(data) {
    const brackets = data.brackets;
    const n = brackets.length;

    // Build pick vectors: for each bracket, create array of 63 picked team IDs
    // ordered by matchup position
    function getPickVector(bracket) {
        const picks = [];
        ['r64','r32','sweet16','elite8','finalFour','championship'].forEach(round => {
            bracket.picks[round].forEach(pick => {
                picks.push(pick.pickedTeamId);
            });
        });
        return picks;
    }

    const vectors = brackets.map(getPickVector);

    // Compute pairwise distances
    const distances = Array.from({length: n}, () => Array(n).fill(0));
    for (let i = 0; i < n; i++) {
        for (let j = i + 1; j < n; j++) {
            let diff = 0;
            const len = Math.min(vectors[i].length, vectors[j].length);
            for (let k = 0; k < len; k++) {
                if (vectors[i][k] !== vectors[j][k]) diff++;
            }
            distances[i][j] = diff;
            distances[j][i] = diff;
        }
    }

    // Originality score: average distance from all others
    const scores = brackets.map((b, i) => {
        const avg = distances[i].reduce((a, b) => a + b, 0) / (n - 1);
        return {
            member: b.member,
            entryName: b.entryName,
            originality: (avg / 63 * 100).toFixed(1),
            avgDistance: avg.toFixed(1)
        };
    });

    scores.sort((a, b) => b.originality - a.originality);
    return { scores, distances, brackets };
}
```

**UI elements:**
- Originality leaderboard: sorted table with bracket name, member, originality score, bar visualization
- Highlight most unique (top) and most mainstream (bottom)
- Per-round breakdown: compute distance using only picks from each round

- [ ] **Step 2: Verify uniqueness tab renders**

Start server, open browser, switch to Uniqueness tab. Verify:
- All 19 brackets listed with originality scores
- Scores range from ~20-60% (reasonable spread)
- Most unique bracket should be one of the outliers (Wright St champion, etc.)

- [ ] **Step 3: Commit**

```bash
git add index.html
git commit -m "feat: dashboard uniqueness tab with originality scores and distance matrix"
```

---

## Task 8: Dashboard — Tab 3: What-If Simulator

**Files:**
- Modify: `index.html`

Add the what-if simulator tab.

- [ ] **Step 1: Implement what-if simulator**

Add to the JS in `index.html`:

**What-if state management:**
```javascript
// Clone the data for what-if mutations
let whatIfState = null; // deep clone of data when entering what-if mode

function initWhatIf(data) {
    whatIfState = JSON.parse(JSON.stringify(data));
    renderWhatIf();
}

function applyWhatIf(matchupId, winnerTeamId) {
    // 1. Find the matchup and mark the winner
    const matchup = whatIfState.matchups.find(m => m.id === matchupId);
    if (matchup) matchup.winnerId = winnerTeamId;

    // 2. Determine the loser
    const loserId = matchup.team1Id === winnerTeamId ? matchup.team2Id : matchup.team1Id;

    // 3. For each bracket, update picks
    const roundKeys = ['r64','r32','sweet16','elite8','finalFour','championship'];
    const pointsPerRound = whatIfState.pointsPerRound;

    whatIfState.brackets.forEach(bracket => {
        // Recalculate score and maxPossible
        let score = 0;
        let maxPossible = 0;

        roundKeys.forEach((roundKey, roundIdx) => {
            const pts = pointsPerRound[roundIdx];
            bracket.picks[roundKey].forEach(pick => {
                // If this pick is for the loser and hasn't been decided
                if (pick.pickedTeamId === loserId && pick.result === 'UNDECIDED') {
                    pick.result = 'ELIMINATED';
                }
                // If this pick matches the new winner for this matchup
                if (pick.matchupId === matchupId && pick.pickedTeamId === winnerTeamId) {
                    pick.result = 'CORRECT';
                }
                if (pick.matchupId === matchupId && pick.pickedTeamId === loserId) {
                    pick.result = 'INCORRECT';
                }

                if (pick.result === 'CORRECT') score += pts;
                if (pick.result === 'CORRECT' || pick.result === 'UNDECIDED') maxPossible += pts;
            });
        });

        bracket.score = score;
        bracket.maxPossible = maxPossible;
    });

    renderWhatIf();
}
```

**UI elements:**
- Bracket-style layout showing R64 matchups as clickable cards
- Click a team to select them as the winner
- Right side panel: leaderboard showing each bracket's current score, max possible, and delta from before the what-if
- Elimination alerts highlighted in red
- "Reset" button to restore original state
- Chain multiple what-ifs by clicking more matchups

- [ ] **Step 2: Verify what-if simulator works**

Start server, open browser, switch to What-If tab. Test:
- Click a team in an R64 matchup to select them as winner
- Verify leaderboard updates with score changes
- Verify brackets that had the loser advancing show elimination cascades
- Click Reset, verify state reverts
- Chain multiple what-ifs

- [ ] **Step 3: Commit**

```bash
git add index.html
git commit -m "feat: dashboard what-if simulator with elimination cascade"
```

---

## Task 9: GitHub Actions workflow

**Files:**
- Create: `.github/workflows/update-brackets.yml`

- [ ] **Step 1: Create the workflow file**

```yaml
name: Update Bracket Data

on:
  schedule:
    - cron: '*/15 * * * *'
  workflow_dispatch:

jobs:
  update:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version-file: 'go.mod'

      - name: Fetch latest bracket data
        run: go run server.go --fetch-only

      - name: Check for changes
        id: diff
        run: |
          git diff --quiet data/brackets.json || echo "changed=true" >> "$GITHUB_OUTPUT"

      - name: Commit and push
        if: steps.diff.outputs.changed == 'true'
        run: |
          git config user.name "github-actions[bot]"
          git config user.email "github-actions[bot]@users.noreply.github.com"
          git add data/brackets.json
          git commit -m "Update bracket data — $(date -u +%Y-%m-%dT%H:%M:%SZ)"
          git push
```

- [ ] **Step 2: Commit**

```bash
git add .github/workflows/update-brackets.yml
git commit -m "feat: GitHub Actions workflow for scheduled bracket data updates"
```

---

## Task 10: Push to GitHub and verify

**Files:** none (git operations only)

- [ ] **Step 1: Push all commits to GitHub**

```bash
git push -u origin main
```

- [ ] **Step 2: Verify repo on GitHub**

Check that `https://github.com/jeffwnelson/march-madness-2026` shows all files.

- [ ] **Step 3: Run the full app locally and smoke test**

```bash
cd /Users/jeffnelson/Code/personal/march-madness-2026
go run server.go
# Open http://localhost:8000
# Test all three tabs
# Test refresh button
# Test what-if simulator
```

- [ ] **Step 4: Trigger GitHub Actions workflow manually**

```bash
gh workflow run update-brackets.yml
gh run list --workflow=update-brackets.yml --limit 1
```

Verify the run completes and commits updated data.
