package main

import (
	"strconv"
	"strings"
	"time"
)

// Processed output types

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
	GameTime     *int64  `json:"gameTime"`
	Status       string  `json:"status"`
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
	Percentile  float64      `json:"percentile"`
	Eliminated  bool         `json:"eliminated"`
	Tiebreaker  *float64     `json:"tiebreaker"`
	Picks       BracketPicks `json:"picks"`
	Champion    string       `json:"champion"`
	FinalFour   []string     `json:"finalFour"`
}

type BracketData struct {
	LastUpdated    string          `json:"lastUpdated"`
	GroupID        string          `json:"groupId"`
	GroupName      string          `json:"groupName"`
	PointsPerRound []int           `json:"pointsPerRound"`
	Teams          map[string]Team `json:"teams"`
	Matchups       []Matchup       `json:"matchups"`
	Brackets       []Bracket       `json:"brackets"`
}

// processData transforms ESPN raw data into a processed bracket structure.
func processData(challenge *espnChallenge, group *espnGroup) *BracketData {
	// Build outcome ID → Team map
	teams := make(map[string]Team)
	for _, prop := range challenge.Propositions {
		for _, outcome := range prop.PossibleOutcomes {
			logo := ""
			for _, m := range outcome.Mappings {
				if m.Type == "IMAGE_PRIMARY" {
					logo = m.Value
					break
				}
			}
			teams[outcome.ID] = Team{
				Name:   outcome.Name,
				Abbrev: outcome.Abbrev,
				Seed:   outcome.RegionSeed,
				Region: outcome.RegionID,
				Record: outcome.AdditionalInfo,
				Logo:   logo,
			}
		}
	}

	// Build region+seed → teamId lookup for resolving championship outcomes
	type regionSeed struct {
		region, seed int
	}
	teamByRegionSeed := make(map[regionSeed]string)
	for id, team := range teams {
		teamByRegionSeed[regionSeed{team.Region, team.Seed}] = id
	}

	// Build proposition ID → scoringPeriod and displayOrder maps
	propPeriod := make(map[string]int)
	propDisplay := make(map[string]int)
	for _, prop := range challenge.Propositions {
		propPeriod[prop.ID] = prop.ScoringPeriodID
		propDisplay[prop.ID] = prop.DisplayOrder
	}

	// Build Matchup objects from propositions
	matchups := make([]Matchup, 0, len(challenge.Propositions))
	for _, prop := range challenge.Propositions {
		matchup := Matchup{
			ID:           prop.ID,
			Round:        1,
			DisplayOrder: prop.DisplayOrder,
			GameTime:     prop.Date,
			Status:       prop.Status,
		}

		for _, outcome := range prop.PossibleOutcomes {
			if matchup.Region == 0 {
				matchup.Region = outcome.RegionID
			}
			if outcome.MatchupPosition == 1 {
				matchup.Team1ID = outcome.ID
			} else {
				matchup.Team2ID = outcome.ID
			}
		}

		// Winner is determined after processing entries (using pick results)
		matchups = append(matchups, matchup)
	}

	// Build a map of proposition ID → matchup index for winner assignment
	matchupIdx := make(map[string]int)
	for i, m := range matchups {
		matchupIdx[m.ID] = i
	}

	// Build championship outcomeId → R64 teamId mapping via cross-entry correlation.
	// The finalPick uses championship-proposition outcomeIds which differ from R64 outcomeIds.
	// For each championship outcomeId, intersect the R64 finalists across all entries that
	// picked it — the common team is the mapping.
	type champCandidate struct {
		teams    map[string]bool
		resolved string
	}
	champMap := make(map[string]*champCandidate)
	for _, entry := range group.Entries {
		if len(entry.FinalPick.OutcomesPicked) == 0 {
			continue
		}
		fpOutcome := entry.FinalPick.OutcomesPicked[0].OutcomeID

		// Collect R64 teams this entry has at periodReached >= 6 (championship finalists)
		finalists := make(map[string]bool)
		for _, pick := range entry.Picks {
			if _, ok := propPeriod[pick.PropositionID]; !ok {
				continue
			}
			if propPeriod[pick.PropositionID] == 1 && pick.PeriodReached >= 6 && len(pick.OutcomesPicked) > 0 {
				finalists[pick.OutcomesPicked[0].OutcomeID] = true
			}
		}

		if existing, ok := champMap[fpOutcome]; ok {
			// Intersect
			for tid := range existing.teams {
				if !finalists[tid] {
					delete(existing.teams, tid)
				}
			}
		} else {
			champMap[fpOutcome] = &champCandidate{teams: finalists}
		}
	}

	// Iteratively resolve: find unambiguous mappings, eliminate those teams from
	// ambiguous ones, repeat until no more progress.
	usedTeams := make(map[string]bool)
	for {
		progress := false
		for _, c := range champMap {
			if c.resolved != "" {
				continue
			}
			for tid := range c.teams {
				if usedTeams[tid] {
					delete(c.teams, tid)
				}
			}
			if len(c.teams) == 1 {
				for tid := range c.teams {
					c.resolved = tid
					usedTeams[tid] = true
					progress = true
				}
			}
		}
		if !progress {
			break
		}
	}

	// Fallback for still-ambiguous outcomes: use championship outcome ID offset pattern.
	// Championship outcomes are ordered by region (16 per region), within each region by
	// standard bracket position: [1,16,8,9,5,12,4,13,6,11,3,14,7,10,2,15].
	// IDs are UUIDs like "c9161f41-0695-..." — the first segment varies.
	bracketOrder := [16]int{1, 16, 8, 9, 5, 12, 4, 13, 6, 11, 3, 14, 7, 10, 2, 15}

	parseFirstSegment := func(id string) (int64, bool) {
		seg := strings.SplitN(id, "-", 2)[0]
		val, err := strconv.ParseInt(seg, 16, 64)
		return val, err == nil
	}

	// Find the base value from any resolved mapping
	var champBase int64
	var hasBase bool
	for fpOutcome, c := range champMap {
		if c.resolved == "" {
			continue
		}
		fpVal, ok := parseFirstSegment(fpOutcome)
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
		champBase = fpVal - int64((team.Region-1)*16+pos)
		hasBase = true
		break
	}

	if hasBase {
		for fpOutcome, c := range champMap {
			if c.resolved != "" {
				continue
			}
			fpVal, ok := parseFirstSegment(fpOutcome)
			if !ok {
				continue
			}
			offset := int(fpVal - champBase)
			region := offset/16 + 1
			pos := offset % 16
			if pos >= 0 && pos < 16 && region >= 1 && region <= 4 {
				seed := bracketOrder[pos]
				if tid, ok := teamByRegionSeed[regionSeed{region, seed}]; ok {
					c.resolved = tid
				}
			}
		}
	}

	// Process each entry into a Bracket
	brackets := make([]Bracket, 0, len(group.Entries))
	for _, entry := range group.Entries {
		var tiebreaker *float64
		if len(entry.TiebreakAnswers) > 0 {
			t := entry.TiebreakAnswers[0].Answer
			tiebreaker = &t
		}

		bracket := Bracket{
			Member:      entry.Member.DisplayName,
			EntryName:   entry.Name,
			Score:       entry.Score.OverallScore,
			MaxPossible: entry.Score.PossiblePointsMax,
			Percentile:  entry.Score.Percentile,
			Eliminated:  entry.Score.Eliminated,
			Tiebreaker:  tiebreaker,
			FinalFour:   []string{},
			Picks: BracketPicks{
				R64:          []Pick{},
				R32:          []Pick{},
				Sweet16:      []Pick{},
				Elite8:       []Pick{},
				FinalFour:    []Pick{},
				Championship: []Pick{},
			},
		}

		// Iterate only R64 picks (propositionId in challenge props with scoringPeriodId==1)
		for _, pick := range entry.Picks {
			period, ok := propPeriod[pick.PropositionID]
			if !ok || period != 1 {
				continue
			}

			if len(pick.OutcomesPicked) == 0 {
				continue
			}

			pickedOutcome := pick.OutcomesPicked[0]
			p := Pick{
				MatchupID:    pick.PropositionID,
				PickedTeamID: pickedOutcome.OutcomeID,
				Result:       pickedOutcome.Result,
			}

			// All R64 picks advance to at least round 2 (periodReached >= 2)
			bracket.Picks.R64 = append(bracket.Picks.R64, p)

			// R32 winners advance to S16 = periodReached >= 3
			if pick.PeriodReached >= 3 {
				bracket.Picks.R32 = append(bracket.Picks.R32, p)
			}

			// S16 winners advance to E8 = periodReached >= 4
			if pick.PeriodReached >= 4 {
				bracket.Picks.Sweet16 = append(bracket.Picks.Sweet16, p)
			}

			// E8 winners reach Final Four = periodReached >= 5
			if pick.PeriodReached >= 5 {
				bracket.Picks.Elite8 = append(bracket.Picks.Elite8, p)
				bracket.FinalFour = append(bracket.FinalFour, pickedOutcome.OutcomeID)
			}

			// Championship finalists = periodReached >= 6
			if pick.PeriodReached >= 6 {
				bracket.Picks.FinalFour = append(bracket.Picks.FinalFour, p)
			}
		}

		// Determine champion from finalPick mapping
		var championID string
		if len(entry.FinalPick.OutcomesPicked) > 0 {
			fpOutcome := entry.FinalPick.OutcomesPicked[0].OutcomeID
			if c, ok := champMap[fpOutcome]; ok && c.resolved != "" {
				championID = c.resolved
			}
		}

		if championID != "" {
			bracket.Champion = championID
			bracket.Picks.Championship = append(bracket.Picks.Championship, Pick{
				MatchupID:    "championship",
				PickedTeamID: championID,
				Result:       "UNDECIDED",
			})
		}

		brackets = append(brackets, bracket)
	}

	// Determine winners for completed propositions using pick results.
	// A pick with result "CORRECT" tells us which team won.
	// We build a set of completed prop IDs, then scan the first entry's picks.
	completedProps := make(map[string]bool)
	for _, prop := range challenge.Propositions {
		if prop.Status == "COMPLETE" {
			completedProps[prop.ID] = true
		}
	}
	if len(brackets) > 0 {
		for _, pick := range brackets[0].Picks.R64 {
			if !completedProps[pick.MatchupID] {
				continue
			}
			// Find the original ESPN pick to check result
			for _, ep := range group.Entries[0].Picks {
				if ep.PropositionID == pick.MatchupID && len(ep.OutcomesPicked) > 0 {
					result := ep.OutcomesPicked[0].Result
					pickedTeam := ep.OutcomesPicked[0].OutcomeID
					if idx, ok := matchupIdx[pick.MatchupID]; ok {
						if result == "CORRECT" {
							matchups[idx].WinnerID = &pickedTeam
						} else if result == "INCORRECT" {
							// The other team won
							m := matchups[idx]
							if pickedTeam == m.Team1ID {
								matchups[idx].WinnerID = &m.Team2ID
							} else {
								matchups[idx].WinnerID = &m.Team1ID
							}
						}
					}
					break
				}
			}
		}
	}

	return &BracketData{
		LastUpdated:    time.Now().UTC().Format(time.RFC3339),
		GroupID:        group.GroupID,
		GroupName:      group.GroupSettings.Name,
		PointsPerRound: []int{10, 20, 40, 80, 160, 320},
		Teams:          teams,
		Matchups:       matchups,
		Brackets:       brackets,
	}
}
