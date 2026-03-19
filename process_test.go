package main

import (
	"os"
	"testing"
)

func loadTestData(t *testing.T) (*espnChallenge, *espnGroup) {
	t.Helper()
	cData, _ := os.ReadFile("testdata/challenge.json")
	gData, _ := os.ReadFile("testdata/group.json")
	challenge, _ := parseChallengeData(cData)
	group, _ := parseGroupData(gData)
	return challenge, group
}

func TestProcessData(t *testing.T) {
	challenge, group := loadTestData(t)
	result := processData(challenge, group)

	if len(result.Teams) != 64 {
		t.Errorf("expected 64 teams, got %d", len(result.Teams))
	}

	if len(result.Matchups) != 32 {
		t.Errorf("expected 32 matchups, got %d", len(result.Matchups))
	}

	if len(result.Brackets) != 19 {
		t.Errorf("expected 19 brackets, got %d", len(result.Brackets))
	}

	expectedPoints := []int{10, 20, 40, 80, 160, 320}
	if len(result.PointsPerRound) != len(expectedPoints) {
		t.Errorf("expected %d PointsPerRound, got %d", len(expectedPoints), len(result.PointsPerRound))
	} else {
		for i, v := range expectedPoints {
			if result.PointsPerRound[i] != v {
				t.Errorf("PointsPerRound[%d]: expected %d, got %d", i, v, result.PointsPerRound[i])
			}
		}
	}

	first := result.Brackets[0]

	if len(first.Picks.R64) != 32 {
		t.Errorf("first bracket R64: expected 32 picks, got %d", len(first.Picks.R64))
	}
	if len(first.Picks.R32) != 16 {
		t.Errorf("first bracket R32: expected 16 picks, got %d", len(first.Picks.R32))
	}
	if len(first.Picks.Sweet16) != 8 {
		t.Errorf("first bracket Sweet16: expected 8 picks, got %d", len(first.Picks.Sweet16))
	}
	if len(first.Picks.Elite8) != 4 {
		t.Errorf("first bracket Elite8: expected 4 picks, got %d", len(first.Picks.Elite8))
	}
	if len(first.Picks.FinalFour) != 2 {
		t.Errorf("first bracket FinalFour: expected 2 picks, got %d", len(first.Picks.FinalFour))
	}
	if len(first.Picks.Championship) != 1 {
		t.Errorf("first bracket Championship: expected 1 pick, got %d", len(first.Picks.Championship))
	}

	if first.Champion == "" {
		t.Error("first bracket: Champion should not be empty")
	}

	if len(first.FinalFour) != 4 {
		t.Errorf("first bracket FinalFour teams: expected 4, got %d", len(first.FinalFour))
	}
}
