package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

// Scenario defines which team wins each game, with CompletedThrough controlling
// how far into the tournament results are applied.
// CompletedThrough: 1=R64, 2=R32, 3=S16, 4=E8, 5=FF, 6=Championship
type Scenario struct {
	Name             string
	CompletedThrough int // 1-6
	R64Winners       [32]int
	R32Winners       [16]int
	S16Winners       [8]int
	E8Winners        [4]int
	FFWinners        [2]int
	ChampWinner      int
}

// buildScenarios generates 12 scenarios: 2 per completion level (R64 through Championship).
func buildScenarios() []Scenario {
	roundNames := []string{"R64", "R32", "S16", "E8", "FF", "Champ"}
	var scenarios []Scenario

	for level := 1; level <= 6; level++ {
		for sample := 0; sample < 2; sample++ {
			seed := int64(level*10 + sample + 1)
			rng := rand.New(rand.NewSource(seed))

			var r64 [32]int
			for j := range r64 {
				r64[j] = rng.Intn(2) + 1
			}
			var r32 [16]int
			for j := range r32 {
				r32[j] = rng.Intn(2) + 1
			}
			var s16 [8]int
			for j := range s16 {
				s16[j] = rng.Intn(2) + 1
			}
			var e8 [4]int
			for j := range e8 {
				e8[j] = rng.Intn(2) + 1
			}
			var ff [2]int
			for j := range ff {
				ff[j] = rng.Intn(2) + 1
			}

			scenarios = append(scenarios, Scenario{
				Name:             fmt.Sprintf("%s_%d", roundNames[level-1], sample+1),
				CompletedThrough: level,
				R64Winners:       r64,
				R32Winners:       r32,
				S16Winners:       s16,
				E8Winners:        e8,
				FFWinners:        ff,
				ChampWinner:      rng.Intn(2) + 1,
			})
		}
	}
	return scenarios
}

// applyScenario deep-copies the ESPN data, applies R64 outcomes, runs processData,
// then post-processes the output to apply R32 outcomes.
func applyScenario(challenge *espnChallenge, group *espnGroup, sc Scenario) *BracketData {
	// Deep copy via JSON round-trip
	cBytes, _ := json.Marshal(challenge)
	gBytes, _ := json.Marshal(group)
	var newChallenge espnChallenge
	var newGroup espnGroup
	json.Unmarshal(cBytes, &newChallenge)
	json.Unmarshal(gBytes, &newGroup)

	// Apply R64 outcomes to ESPN source data
	winnerIDs := make(map[string]string) // propositionID → winning outcomeID
	for i, prop := range newChallenge.Propositions {
		var pos1ID, pos2ID string
		for _, o := range prop.PossibleOutcomes {
			if o.MatchupPosition == 1 {
				pos1ID = o.ID
			} else {
				pos2ID = o.ID
			}
		}

		if sc.R64Winners[prop.DisplayOrder] == 1 {
			winnerIDs[prop.ID] = pos1ID
		} else {
			winnerIDs[prop.ID] = pos2ID
		}

		loserID := pos2ID
		if winnerIDs[prop.ID] == pos2ID {
			loserID = pos1ID
		}

		newChallenge.Propositions[i].Status = "COMPLETE"
		newChallenge.Propositions[i].ActualOutcomeIDs = []string{winnerIDs[prop.ID], loserID}
	}

	// Update every entry's R64 pick results
	for i, entry := range newGroup.Entries {
		for j, pick := range entry.Picks {
			winner, ok := winnerIDs[pick.PropositionID]
			if !ok || len(pick.OutcomesPicked) == 0 {
				continue
			}
			if pick.OutcomesPicked[0].OutcomeID == winner {
				newGroup.Entries[i].Picks[j].OutcomesPicked[0].Result = "CORRECT"
			} else {
				newGroup.Entries[i].Picks[j].OutcomesPicked[0].Result = "INCORRECT"
			}
		}
	}

	// Run processData to get base output (R64 decided, R32+ UNDECIDED)
	result := processData(&newChallenge, &newGroup)

	// Post-process: apply round outcomes up to CompletedThrough
	if sc.CompletedThrough >= 2 {
		applyR32Results(result, sc.R32Winners)
	}
	if sc.CompletedThrough >= 3 {
		applyS16Results(result, sc.R32Winners, sc.S16Winners)
	}
	if sc.CompletedThrough >= 4 {
		applyE8Results(result, sc.R32Winners, sc.S16Winners, sc.E8Winners)
	}
	if sc.CompletedThrough >= 5 {
		applyFFResults(result, sc.R32Winners, sc.S16Winners, sc.E8Winners, sc.FFWinners)
	}
	if sc.CompletedThrough >= 6 {
		applyChampResults(result, sc.R32Winners, sc.S16Winners, sc.E8Winners, sc.FFWinners, sc.ChampWinner)
	}

	return result
}

// applyR32Results sets Sweet16 pick results based on R32 outcomes.
// R32 matchup i pairs R64 winners from displayOrder 2*i and 2*i+1.
// r32Winners[i]=1 means the winner from DO 2*i advances; =2 means DO 2*i+1.
func applyR32Results(data *BracketData, r32Winners [16]int) {
	// Build matchup ID → displayOrder map
	matchupDO := make(map[string]int)
	for _, m := range data.Matchups {
		matchupDO[m.ID] = m.DisplayOrder
	}

	// Build R64 displayOrder → winning teamID
	r64Winners := make(map[int]string)
	matchups := make([]Matchup, len(data.Matchups))
	copy(matchups, data.Matchups)
	sort.Slice(matchups, func(i, j int) bool { return matchups[i].DisplayOrder < matchups[j].DisplayOrder })
	for _, m := range matchups {
		if m.WinnerID != nil {
			r64Winners[m.DisplayOrder] = *m.WinnerID
		}
	}

	// Build R32 slot → winning teamID
	r32WinnerTeams := make(map[int]string) // R32 slot → team that won R32
	for slot := 0; slot < 16; slot++ {
		evenDO := slot * 2
		oddDO := slot*2 + 1
		if r32Winners[slot] == 1 {
			r32WinnerTeams[slot] = r64Winners[evenDO]
		} else {
			r32WinnerTeams[slot] = r64Winners[oddDO]
		}
	}

	// For each bracket, update S16 picks and R32 picks
	for i := range data.Brackets {
		// Update R32 picks: CORRECT if team won their R32 game
		for j, pick := range data.Brackets[i].Picks.R32 {
			do, ok := matchupDO[pick.MatchupID]
			if !ok {
				continue
			}
			slot := do / 2 // R32 slot this pick feeds into
			if r32WinnerTeams[slot] == pick.PickedTeamID {
				data.Brackets[i].Picks.R32[j].Result = "CORRECT"
			} else {
				data.Brackets[i].Picks.R32[j].Result = "INCORRECT"
			}
		}
	}
}

// applyS16Results sets Sweet16 pick results based on S16 outcomes.
// S16 matchup i pairs R32 winners from slots 2*i and 2*i+1.
// s16Winners[i]=1 means the winner from R32 slot 2*i advances; =2 means 2*i+1.
func applyS16Results(data *BracketData, r32Winners [16]int, s16Winners [8]int) {
	// Build matchup ID → displayOrder map
	matchupDO := make(map[string]int)
	for _, m := range data.Matchups {
		matchupDO[m.ID] = m.DisplayOrder
	}

	// Build R64 DO → winning teamID
	r64Winners := make(map[int]string)
	for _, m := range data.Matchups {
		if m.WinnerID != nil {
			r64Winners[m.DisplayOrder] = *m.WinnerID
		}
	}

	// Build R32 slot → winning teamID
	r32WinnerTeams := make(map[int]string)
	for slot := 0; slot < 16; slot++ {
		if r32Winners[slot] == 1 {
			r32WinnerTeams[slot] = r64Winners[slot*2]
		} else {
			r32WinnerTeams[slot] = r64Winners[slot*2+1]
		}
	}

	// Build S16 slot → winning teamID
	s16WinnerTeams := make(map[int]string)
	for slot := 0; slot < 8; slot++ {
		if s16Winners[slot] == 1 {
			s16WinnerTeams[slot] = r32WinnerTeams[slot*2]
		} else {
			s16WinnerTeams[slot] = r32WinnerTeams[slot*2+1]
		}
	}

	// For each bracket, update S16 picks
	for i := range data.Brackets {
		for j, pick := range data.Brackets[i].Picks.Sweet16 {
			do, ok := matchupDO[pick.MatchupID]
			if !ok {
				continue
			}
			slot := do / 4 // S16 slot: each covers 4 R64 display orders
			if s16WinnerTeams[slot] == pick.PickedTeamID {
				data.Brackets[i].Picks.Sweet16[j].Result = "CORRECT"
			} else {
				data.Brackets[i].Picks.Sweet16[j].Result = "INCORRECT"
			}
		}
	}
}

// applyE8Results sets Elite8 pick results based on E8 outcomes.
// E8 matchup i pairs S16 winners from slots 2*i and 2*i+1.
// e8Winners[i]=1 means the winner from S16 slot 2*i advances; =2 means 2*i+1.
func applyE8Results(data *BracketData, r32Winners [16]int, s16Winners [8]int, e8Winners [4]int) {
	matchupDO := make(map[string]int)
	for _, m := range data.Matchups {
		matchupDO[m.ID] = m.DisplayOrder
	}

	// Rebuild the winner chain: R64 → R32 → S16 → E8
	r64Winners := make(map[int]string)
	for _, m := range data.Matchups {
		if m.WinnerID != nil {
			r64Winners[m.DisplayOrder] = *m.WinnerID
		}
	}

	r32WinnerTeams := make(map[int]string)
	for slot := 0; slot < 16; slot++ {
		if r32Winners[slot] == 1 {
			r32WinnerTeams[slot] = r64Winners[slot*2]
		} else {
			r32WinnerTeams[slot] = r64Winners[slot*2+1]
		}
	}

	s16WinnerTeams := make(map[int]string)
	for slot := 0; slot < 8; slot++ {
		if s16Winners[slot] == 1 {
			s16WinnerTeams[slot] = r32WinnerTeams[slot*2]
		} else {
			s16WinnerTeams[slot] = r32WinnerTeams[slot*2+1]
		}
	}

	e8WinnerTeams := make(map[int]string)
	for slot := 0; slot < 4; slot++ {
		if e8Winners[slot] == 1 {
			e8WinnerTeams[slot] = s16WinnerTeams[slot*2]
		} else {
			e8WinnerTeams[slot] = s16WinnerTeams[slot*2+1]
		}
	}

	// For each bracket, update E8 picks
	for i := range data.Brackets {
		for j, pick := range data.Brackets[i].Picks.Elite8 {
			do, ok := matchupDO[pick.MatchupID]
			if !ok {
				continue
			}
			slot := do / 8 // E8 slot: each covers 8 R64 display orders
			if e8WinnerTeams[slot] == pick.PickedTeamID {
				data.Brackets[i].Picks.Elite8[j].Result = "CORRECT"
			} else {
				data.Brackets[i].Picks.Elite8[j].Result = "INCORRECT"
			}
		}
	}
}

// applyFFResults sets FinalFour pick results based on FF outcomes.
// FF matchup i pairs E8 winners from slots 2*i and 2*i+1.
func applyFFResults(data *BracketData, r32Winners [16]int, s16Winners [8]int, e8Winners [4]int, ffWinners [2]int) {
	matchupDO := make(map[string]int)
	for _, m := range data.Matchups {
		matchupDO[m.ID] = m.DisplayOrder
	}

	// Rebuild full winner chain
	r64Winners := make(map[int]string)
	for _, m := range data.Matchups {
		if m.WinnerID != nil {
			r64Winners[m.DisplayOrder] = *m.WinnerID
		}
	}

	r32WinnerTeams := make(map[int]string)
	for slot := 0; slot < 16; slot++ {
		if r32Winners[slot] == 1 {
			r32WinnerTeams[slot] = r64Winners[slot*2]
		} else {
			r32WinnerTeams[slot] = r64Winners[slot*2+1]
		}
	}

	s16WinnerTeams := make(map[int]string)
	for slot := 0; slot < 8; slot++ {
		if s16Winners[slot] == 1 {
			s16WinnerTeams[slot] = r32WinnerTeams[slot*2]
		} else {
			s16WinnerTeams[slot] = r32WinnerTeams[slot*2+1]
		}
	}

	e8WinnerTeams := make(map[int]string)
	for slot := 0; slot < 4; slot++ {
		if e8Winners[slot] == 1 {
			e8WinnerTeams[slot] = s16WinnerTeams[slot*2]
		} else {
			e8WinnerTeams[slot] = s16WinnerTeams[slot*2+1]
		}
	}

	ffWinnerTeams := make(map[int]string)
	for slot := 0; slot < 2; slot++ {
		if ffWinners[slot] == 1 {
			ffWinnerTeams[slot] = e8WinnerTeams[slot*2]
		} else {
			ffWinnerTeams[slot] = e8WinnerTeams[slot*2+1]
		}
	}

	// For each bracket, update FF picks
	for i := range data.Brackets {
		for j, pick := range data.Brackets[i].Picks.FinalFour {
			do, ok := matchupDO[pick.MatchupID]
			if !ok {
				continue
			}
			slot := do / 16 // FF slot: each covers 16 R64 display orders
			if ffWinnerTeams[slot] == pick.PickedTeamID {
				data.Brackets[i].Picks.FinalFour[j].Result = "CORRECT"
			} else {
				data.Brackets[i].Picks.FinalFour[j].Result = "INCORRECT"
			}
		}
	}
}

// applyChampResults sets Championship pick results based on the championship outcome.
// champWinner=1 means FF slot 0 winner wins; =2 means FF slot 1 winner wins.
func applyChampResults(data *BracketData, r32Winners [16]int, s16Winners [8]int, e8Winners [4]int, ffWinners [2]int, champWinner int) {
	// Rebuild full winner chain
	r64Winners := make(map[int]string)
	for _, m := range data.Matchups {
		if m.WinnerID != nil {
			r64Winners[m.DisplayOrder] = *m.WinnerID
		}
	}

	r32WinnerTeams := make(map[int]string)
	for slot := 0; slot < 16; slot++ {
		if r32Winners[slot] == 1 {
			r32WinnerTeams[slot] = r64Winners[slot*2]
		} else {
			r32WinnerTeams[slot] = r64Winners[slot*2+1]
		}
	}

	s16WinnerTeams := make(map[int]string)
	for slot := 0; slot < 8; slot++ {
		if s16Winners[slot] == 1 {
			s16WinnerTeams[slot] = r32WinnerTeams[slot*2]
		} else {
			s16WinnerTeams[slot] = r32WinnerTeams[slot*2+1]
		}
	}

	e8WinnerTeams := make(map[int]string)
	for slot := 0; slot < 4; slot++ {
		if e8Winners[slot] == 1 {
			e8WinnerTeams[slot] = s16WinnerTeams[slot*2]
		} else {
			e8WinnerTeams[slot] = s16WinnerTeams[slot*2+1]
		}
	}

	ffWinnerTeams := make(map[int]string)
	for slot := 0; slot < 2; slot++ {
		if ffWinners[slot] == 1 {
			ffWinnerTeams[slot] = e8WinnerTeams[slot*2]
		} else {
			ffWinnerTeams[slot] = e8WinnerTeams[slot*2+1]
		}
	}

	var champion string
	if champWinner == 1 {
		champion = ffWinnerTeams[0]
	} else {
		champion = ffWinnerTeams[1]
	}

	// Update championship picks
	for i := range data.Brackets {
		for j, pick := range data.Brackets[i].Picks.Championship {
			if pick.PickedTeamID == champion {
				data.Brackets[i].Picks.Championship[j].Result = "CORRECT"
			} else {
				data.Brackets[i].Picks.Championship[j].Result = "INCORRECT"
			}
		}
	}
}

func TestScenarios(t *testing.T) {
	challenge, group := loadTestData(t)
	scenarios := buildScenarios()

	if len(scenarios) != 12 {
		t.Fatalf("expected 12 scenarios, got %d", len(scenarios))
	}

	for _, sc := range scenarios {
		t.Run(sc.Name, func(t *testing.T) {
			result := applyScenario(challenge, group, sc)

			// Structural checks
			if len(result.Teams) != 64 {
				t.Errorf("expected 64 teams, got %d", len(result.Teams))
			}
			if len(result.Matchups) != 32 {
				t.Errorf("expected 32 matchups, got %d", len(result.Matchups))
			}
			if len(result.Brackets) != 19 {
				t.Errorf("expected 19 brackets, got %d", len(result.Brackets))
			}

			// R64 matchups always have winners
			for _, m := range result.Matchups {
				if m.WinnerID == nil {
					t.Errorf("matchup DO=%d has nil WinnerID", m.DisplayOrder)
				}
			}

			// Validate pick results per round based on CompletedThrough
			// Round N completed → picks decided; round N+1 → UNDECIDED
			type roundCheck struct {
				name  string
				level int
				picks func(b Bracket) []Pick
			}
			checks := []roundCheck{
				{"R64", 1, func(b Bracket) []Pick { return b.Picks.R64 }},
				{"R32", 2, func(b Bracket) []Pick { return b.Picks.R32 }},
				{"S16", 3, func(b Bracket) []Pick { return b.Picks.Sweet16 }},
				{"E8", 4, func(b Bracket) []Pick { return b.Picks.Elite8 }},
				{"FF", 5, func(b Bracket) []Pick { return b.Picks.FinalFour }},
				{"Champ", 6, func(b Bracket) []Pick { return b.Picks.Championship }},
			}

			for _, bracket := range result.Brackets {
				for _, rc := range checks {
					for _, pick := range rc.picks(bracket) {
						if rc.level <= sc.CompletedThrough {
							if pick.Result != "CORRECT" && pick.Result != "INCORRECT" {
								t.Errorf("bracket %q: %s pick %s has result %q, expected decided",
									bracket.EntryName, rc.name, pick.MatchupID, pick.Result)
							}
						} else {
							if pick.Result != "UNDECIDED" {
								t.Errorf("bracket %q: %s pick %s has result %q, expected UNDECIDED",
									bracket.EntryName, rc.name, pick.MatchupID, pick.Result)
							}
						}
					}
				}
			}
		})
	}
}

func TestGenerateScenarioFiles(t *testing.T) {
	if os.Getenv("GENERATE_SCENARIOS") != "1" {
		t.Skip("set GENERATE_SCENARIOS=1 to generate scenario files")
	}

	challenge, group := loadTestData(t)
	scenarios := buildScenarios()

	for _, sc := range scenarios {
		t.Run(sc.Name, func(t *testing.T) {
			result := applyScenario(challenge, group, sc)

			dir := filepath.Join("testdata", "scenarios", sc.Name)
			if err := os.MkdirAll(dir, 0o755); err != nil {
				t.Fatalf("mkdir %s: %v", dir, err)
			}

			data, err := json.MarshalIndent(result, "", "  ")
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}

			outPath := filepath.Join(dir, "brackets.json")
			if err := os.WriteFile(outPath, data, 0o644); err != nil {
				t.Fatalf("write %s: %v", outPath, err)
			}

			t.Logf("wrote %s (%d bytes)", outPath, len(data))
		})
	}
}
