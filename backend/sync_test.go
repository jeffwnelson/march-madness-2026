package main

import (
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

	// Should have 64 teams.
	if len(lb.Teams) != 64 {
		t.Errorf("expected 64 teams, got %d", len(lb.Teams))
	}

	// Should have 19 brackets.
	if len(lb.Brackets) != 19 {
		t.Errorf("expected 19 brackets, got %d", len(lb.Brackets))
	}

	// Every bracket should have a champion and 4 Final Four entries.
	for i, b := range lb.Brackets {
		if b.Champion == "" {
			t.Errorf("bracket %d (%s): champion should not be empty", i, b.EntryName)
		}
		if len(b.FinalFour) != 4 {
			t.Errorf("bracket %d (%s): expected 4 finalFour, got %d", i, b.EntryName, len(b.FinalFour))
		}
	}

	// GroupName should be set.
	if lb.GroupName == "" {
		t.Error("GroupName should not be empty")
	}

	// LastUpdated should be set.
	if lb.LastUpdated == "" {
		t.Error("LastUpdated should not be empty")
	}
}

func TestSyncProducesBracketPicks(t *testing.T) {
	ch := loadTestChallenge(t)
	g := loadTestGroup(t)

	_, bp := buildOutputs(ch, g, nil, nil)

	// Should have 64 teams.
	if len(bp.Teams) != 64 {
		t.Errorf("expected 64 teams, got %d", len(bp.Teams))
	}

	// All 6 round keys should be present.
	for _, rk := range roundKeyOrder {
		if _, ok := bp.Rounds[rk]; !ok {
			t.Errorf("missing round key: %s", rk)
		}
	}

	// R64 should have 32 matchups.
	r64 := bp.Rounds["r64"]
	if len(r64.Matchups) != 32 {
		t.Errorf("R64: expected 32 matchups, got %d", len(r64.Matchups))
	}

	// Each R64 matchup picks should total 19 (one per entry).
	for i, m := range r64.Matchups {
		total := 0
		for _, pd := range m.Picks {
			total += pd.Count
		}
		if total != 19 {
			t.Errorf("R64 matchup %d (%s): expected 19 total picks, got %d", i, m.ID, total)
		}
	}

	// Each R64 matchup should have two teams set.
	for i, m := range r64.Matchups {
		if m.Team1 == "" {
			t.Errorf("R64 matchup %d: Team1 is empty", i)
		}
		if m.Team2 == "" {
			t.Errorf("R64 matchup %d: Team2 is empty", i)
		}
	}

	// LastUpdated should be set.
	if bp.LastUpdated == "" {
		t.Error("LastUpdated should not be empty")
	}
}
