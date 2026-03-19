package main

import (
	"encoding/json"
	"fmt"
)

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
