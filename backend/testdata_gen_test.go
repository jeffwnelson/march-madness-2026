package main

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"os"
	"path/filepath"
	"testing"
)

// TestGenerateScenarioFiles generates leaderboard.json and bracket-picks.json
// for each tournament scenario. Run with GENERATE_SCENARIOS=1.
func TestGenerateScenarioFiles(t *testing.T) {
	if os.Getenv("GENERATE_SCENARIOS") != "1" {
		t.Skip("set GENERATE_SCENARIOS=1 to generate scenario files")
	}

	ch := loadTestChallenge(t)
	g := loadTestGroup(t)
	lb, bp := buildOutputs(ch, g, nil, nil)

	scenarios := []struct {
		name            string
		completedRounds int // 1=R64, 2=R32, ..., 6=Championship
	}{
		{"R64_1", 1}, {"R64_2", 1},
		{"R32_1", 2}, {"R32_2", 2},
		{"S16_1", 3}, {"S16_2", 3},
		{"E8_1", 4}, {"E8_2", 4},
		{"FF_1", 5}, {"FF_2", 5},
		{"Champ_1", 6}, {"Champ_2", 6},
	}

	for _, sc := range scenarios {
		t.Run(sc.name, func(t *testing.T) {
			dir := filepath.Join("testdata", "scenarios", sc.name)
			if err := os.MkdirAll(dir, 0755); err != nil {
				t.Fatal(err)
			}

			// Deep copy via JSON round-trip
			scenarioLB := deepCopyJSON[LeaderboardData](t, lb)
			scenarioBP := deepCopyJSON[BracketPicksData](t, bp)

			// Generate synthetic matchups for rounds beyond the base R64 data
			generateRounds(t, scenarioBP, sc.completedRounds, sc.name)

			if err := saveJSON(filepath.Join(dir, "leaderboard.json"), scenarioLB); err != nil {
				t.Fatal(err)
			}
			if err := saveJSON(filepath.Join(dir, "bracket-picks.json"), scenarioBP); err != nil {
				t.Fatal(err)
			}

			t.Logf("Generated %s (completed rounds: %d)", sc.name, sc.completedRounds)
		})
	}
}

// deepCopyJSON performs a deep copy of a value via JSON marshal/unmarshal.
func deepCopyJSON[T any](t *testing.T, src *T) *T {
	t.Helper()
	b, err := json.Marshal(src)
	if err != nil {
		t.Fatal(err)
	}
	var dst T
	if err := json.Unmarshal(b, &dst); err != nil {
		t.Fatal(err)
	}
	return &dst
}

// hashPick returns a deterministic int from a scenario name + matchup index.
func hashPick(scenario string, index int) uint32 {
	h := fnv.New32a()
	h.Write([]byte(fmt.Sprintf("%s-%d", scenario, index)))
	return h.Sum32()
}

// generateRounds modifies bracket-picks data to simulate tournament progression.
// For completed rounds, matchups get winners and status=COMPLETE.
// The round after completedRounds gets status "in_progress" or stays "future".
func generateRounds(t *testing.T, bp *BracketPicksData, completedRounds int, scenarioName string) {
	t.Helper()

	// Round 1 (R64) comes from the base data and should already have matchups.
	// For rounds 2+, we need to build synthetic matchups by promoting winners.

	// First, ensure R64 matchups all have winners and are COMPLETE if completedRounds >= 1.
	r64 := bp.Rounds["r64"]
	if completedRounds >= 1 {
		for i := range r64.Matchups {
			m := &r64.Matchups[i]
			if m.Winner == "" {
				// Pick a winner deterministically
				if hashPick(scenarioName, i)%2 == 0 {
					m.Winner = m.Team1
				} else {
					m.Winner = m.Team2
				}
			}
			m.Status = "COMPLETE"
		}
		r64.Status = "complete"
	}
	bp.Rounds["r64"] = r64

	// For subsequent rounds, promote winners from the previous round.
	// Each round halves the number of matchups.
	prevMatchups := r64.Matchups

	for roundIdx := 1; roundIdx < 6; roundIdx++ {
		roundKey := roundKeyOrder[roundIdx]

		// Build matchups by pairing consecutive winners from previous round
		var matchups []MatchupData
		for i := 0; i+1 < len(prevMatchups); i += 2 {
			team1 := prevMatchups[i].Winner
			team2 := prevMatchups[i+1].Winner
			if team1 == "" || team2 == "" {
				continue
			}

			// Generate synthetic pick counts (deterministic split of 19 picks)
			h := hashPick(scenarioName, 2000*(roundIdx+1)+i/2)
			team1Picks := int(h%15) + 2 // 2-16 picks for team1
			team2Picks := 19 - team1Picks

			m := MatchupData{
				ID:           fmt.Sprintf("syn-%s-%d", roundKey, i/2),
				Region:       prevMatchups[i].Region,
				DisplayOrder: i/2 + 1,
				Team1:        team1,
				Team2:        team2,
				Status:       "FUTURE",
				Picks: map[string]PickData{
					team1: {Count: team1Picks, Entries: syntheticEntries(team1Picks)},
					team2: {Count: team2Picks, Entries: syntheticEntries(team2Picks)},
				},
			}

			if roundIdx < completedRounds {
				// This round is complete — pick a winner
				if hashPick(scenarioName, 1000*(roundIdx+1)+i/2)%2 == 0 {
					m.Winner = m.Team1
				} else {
					m.Winner = m.Team2
				}
				m.Status = "COMPLETE"
			}

			matchups = append(matchups, m)
		}

		if matchups == nil {
			matchups = []MatchupData{}
		}

		status := "future"
		if roundIdx < completedRounds {
			status = "complete"
		} else if roundIdx == completedRounds {
			// The round right after the last completed one is "in_progress" if completedRounds < 6
			status = "in_progress"
		}

		bp.Rounds[roundKey] = Round{
			Status:   status,
			Matchups: matchups,
		}

		prevMatchups = matchups
	}
}

// syntheticEntries generates placeholder entry names for synthetic pick data.
func syntheticEntries(count int) []string {
	entries := make([]string, count)
	for i := range entries {
		entries[i] = fmt.Sprintf("Entry_%d", i+1)
	}
	return entries
}

// TestScenarios loads each generated scenario and validates basic properties.
func TestScenarios(t *testing.T) {
	scenarios := []struct {
		name            string
		completedRounds int
	}{
		{"R64_1", 1}, {"R64_2", 1},
		{"R32_1", 2}, {"R32_2", 2},
		{"S16_1", 3}, {"S16_2", 3},
		{"E8_1", 4}, {"E8_2", 4},
		{"FF_1", 5}, {"FF_2", 5},
		{"Champ_1", 6}, {"Champ_2", 6},
	}

	for _, sc := range scenarios {
		t.Run(sc.name, func(t *testing.T) {
			lbPath := filepath.Join("testdata", "scenarios", sc.name, "leaderboard.json")
			bpPath := filepath.Join("testdata", "scenarios", sc.name, "bracket-picks.json")

			lb, err := loadJSON[LeaderboardData](lbPath)
			if err != nil {
				t.Fatalf("loading leaderboard: %v", err)
			}
			bp, err := loadJSON[BracketPicksData](bpPath)
			if err != nil {
				t.Fatalf("loading bracket-picks: %v", err)
			}

			// Should have 19 brackets
			if len(lb.Brackets) != 19 {
				t.Errorf("expected 19 brackets, got %d", len(lb.Brackets))
			}

			// Should have 64 teams
			if len(lb.Teams) != 64 {
				t.Errorf("expected 64 teams in leaderboard, got %d", len(lb.Teams))
			}
			if len(bp.Teams) != 64 {
				t.Errorf("expected 64 teams in bracket-picks, got %d", len(bp.Teams))
			}

			// All 6 round keys should be present
			for _, rk := range roundKeyOrder {
				if _, ok := bp.Rounds[rk]; !ok {
					t.Errorf("missing round key: %s", rk)
				}
			}

			// Check round statuses match expected completion level
			for i, key := range roundKeyOrder {
				round := bp.Rounds[key]
				if i < sc.completedRounds {
					if round.Status != "complete" {
						t.Errorf("round %s: expected status 'complete', got %q", key, round.Status)
					}
					// All matchups in completed rounds should have winners
					for j, m := range round.Matchups {
						if m.Winner == "" {
							t.Errorf("round %s matchup %d: expected winner in completed round", key, j)
						}
					}
				} else {
					if round.Status == "complete" {
						t.Errorf("round %s: should not be 'complete' (completedRounds=%d)", key, sc.completedRounds)
					}
					// Matchups in incomplete rounds should NOT have winners
					for j, m := range round.Matchups {
						if m.Winner != "" {
							t.Errorf("round %s matchup %d: unexpected winner in incomplete round", key, j)
						}
					}
				}
			}

			// R64 should have 32 matchups
			r64 := bp.Rounds["r64"]
			if len(r64.Matchups) != 32 {
				t.Errorf("R64: expected 32 matchups, got %d", len(r64.Matchups))
			}

			// Check expected matchup counts for later rounds
			expectedMatchups := []int{32, 16, 8, 4, 2, 1}
			for i, key := range roundKeyOrder {
				if i < sc.completedRounds {
					round := bp.Rounds[key]
					if len(round.Matchups) != expectedMatchups[i] {
						t.Errorf("round %s: expected %d matchups, got %d", key, expectedMatchups[i], len(round.Matchups))
					}
				}
			}
		})
	}
}
