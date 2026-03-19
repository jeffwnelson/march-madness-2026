package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

const (
	challengeURL = "https://gambit-api.fantasy.espn.com/apis/v1/challenges/tournament-challenge-bracket-2026"
	groupURL     = "https://gambit-api.fantasy.espn.com/apis/v1/challenges/tournament-challenge-bracket-2026/groups/af223df6-96d0-46e7-b00d-1b590dc67888?view=entries&limit=50"
	cachePath    = "data/brackets.json"
)

func saveCache(path string, data *BracketData) error {
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0644)
}

func loadCache(path string) (*BracketData, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var data BracketData
	if err := json.Unmarshal(b, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

func fetchESPNData() (*BracketData, error) {
	challengeResp, err := http.Get(challengeURL)
	if err != nil {
		return nil, fmt.Errorf("fetching challenge data: %w", err)
	}
	defer challengeResp.Body.Close()
	challengeBytes, err := io.ReadAll(challengeResp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading challenge response: %w", err)
	}

	groupResp, err := http.Get(groupURL)
	if err != nil {
		return nil, fmt.Errorf("fetching group data: %w", err)
	}
	defer groupResp.Body.Close()
	groupBytes, err := io.ReadAll(groupResp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading group response: %w", err)
	}

	challenge, err := parseChallengeData(challengeBytes)
	if err != nil {
		return nil, fmt.Errorf("parsing challenge data: %w", err)
	}

	group, err := parseGroupData(groupBytes)
	if err != nil {
		return nil, fmt.Errorf("parsing group data: %w", err)
	}

	return processData(challenge, group), nil
}

func main() {
	fmt.Println("march-madness-2026 server starting")
}

// ESPN raw API types

type espnChallenge struct {
	Propositions []espnProposition `json:"propositions"`
}

type espnProposition struct {
	ID               string        `json:"id"`
	Name             string        `json:"name"`
	ScoringPeriodID  int           `json:"scoringPeriodId"`
	DisplayOrder     int           `json:"displayOrder"`
	ActualOutcomeIDs []string      `json:"actualOutcomeIds"`
	PossibleOutcomes []espnOutcome `json:"possibleOutcomes"`
}

type espnOutcome struct {
	ID                 string        `json:"id"`
	Name               string        `json:"name"`
	Abbrev             string        `json:"abbrev"`
	Description        string        `json:"description"`
	AdditionalInfo     string        `json:"additionalInfo"`
	RegionID           int           `json:"regionId"`
	RegionSeed         int           `json:"regionSeed"`
	RegionCompetitorID string        `json:"regionCompetitorId"`
	MatchupPosition    int           `json:"matchupPosition"`
	Mappings           []espnMapping `json:"mappings"`
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
	OverallScore            int  `json:"overallScore"`
	PossiblePointsMax       int  `json:"possiblePointsMax"`
	PossiblePointsRemaining int  `json:"possiblePointsRemaining"`
	PointsLost              int  `json:"pointsLost"`
	Rank                    int  `json:"rank"`
	Eliminated              bool `json:"eliminated"`
}

// Parsing functions

func parseChallengeData(data []byte) (*espnChallenge, error) {
	var challenge espnChallenge
	if err := json.Unmarshal(data, &challenge); err != nil {
		return nil, err
	}
	return &challenge, nil
}

func parseGroupData(data []byte) (*espnGroup, error) {
	var group espnGroup
	if err := json.Unmarshal(data, &group); err != nil {
		return nil, err
	}
	return &group, nil
}
