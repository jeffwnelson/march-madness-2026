package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseChallengeData(t *testing.T) {
	data, err := os.ReadFile("testdata/challenge.json")
	if err != nil {
		t.Fatalf("failed to read challenge.json: %v", err)
	}

	challenge, err := parseChallengeData(data)
	if err != nil {
		t.Fatalf("parseChallengeData failed: %v", err)
	}

	if len(challenge.Propositions) != 32 {
		t.Errorf("expected 32 propositions, got %d", len(challenge.Propositions))
	}

	for i, prop := range challenge.Propositions {
		if len(prop.PossibleOutcomes) != 2 {
			t.Errorf("proposition %d: expected 2 outcomes, got %d", i, len(prop.PossibleOutcomes))
		}
		for j, outcome := range prop.PossibleOutcomes {
			if outcome.Name == "" {
				t.Errorf("proposition %d outcome %d: name is empty", i, j)
			}
			if outcome.RegionID == 0 {
				t.Errorf("proposition %d outcome %d: regionId is 0", i, j)
			}
		}
	}
}

func TestSaveAndLoadCache(t *testing.T) {
	challenge, group := loadTestData(t)
	original := processData(challenge, group)

	tmpDir := t.TempDir()
	cachefile := filepath.Join(tmpDir, "brackets.json")

	if err := saveCache(cachefile, original); err != nil {
		t.Fatalf("saveCache failed: %v", err)
	}

	loaded, err := loadCache(cachefile)
	if err != nil {
		t.Fatalf("loadCache failed: %v", err)
	}

	if len(loaded.Teams) != len(original.Teams) {
		t.Errorf("teams count mismatch: got %d, want %d", len(loaded.Teams), len(original.Teams))
	}
	if len(loaded.Brackets) != len(original.Brackets) {
		t.Errorf("brackets count mismatch: got %d, want %d", len(loaded.Brackets), len(original.Brackets))
	}
	if len(loaded.Matchups) != len(original.Matchups) {
		t.Errorf("matchups count mismatch: got %d, want %d", len(loaded.Matchups), len(original.Matchups))
	}
}

func TestParseGroupData(t *testing.T) {
	data, err := os.ReadFile("testdata/group.json")
	if err != nil {
		t.Fatalf("failed to read group.json: %v", err)
	}

	group, err := parseGroupData(data)
	if err != nil {
		t.Fatalf("parseGroupData failed: %v", err)
	}

	if len(group.Entries) != 19 {
		t.Errorf("expected 19 entries, got %d", len(group.Entries))
	}

	first := group.Entries[0]
	if len(first.Picks) != 63 {
		t.Errorf("first entry: expected 63 picks, got %d", len(first.Picks))
	}

	for i, pick := range first.Picks {
		if pick.PropositionID == "" {
			t.Errorf("pick %d: propositionId is empty", i)
		}
		if len(pick.OutcomesPicked) == 0 {
			t.Errorf("pick %d: outcomesPicked is empty", i)
		}
	}
}
