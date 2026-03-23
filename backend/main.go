package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"
)

// ESPN API types

type ESPNChallenge struct {
	Propositions []ESPNProposition `json:"propositions"`
}

type ESPNProposition struct {
	ID               string        `json:"id"`
	Status           string        `json:"status"`
	Date             *int64        `json:"date"`
	ScoringPeriodID  int           `json:"scoringPeriodId"`
	DisplayOrder     int           `json:"displayOrder"`
	ActualOutcomeIDs []string      `json:"actualOutcomeIds"`
	CorrectOutcomes  []string      `json:"correctOutcomes"`
	PossibleOutcomes []ESPNOutcome `json:"possibleOutcomes"`
}

type ESPNOutcome struct {
	ID              string        `json:"id"`
	Name            string        `json:"name"`
	Abbrev          string        `json:"abbrev"`
	AdditionalInfo  string        `json:"additionalInfo"`
	RegionID        int           `json:"regionId"`
	RegionSeed      int           `json:"regionSeed"`
	MatchupPosition int           `json:"matchupPosition"`
	Mappings        []ESPNMapping `json:"mappings"`
}

type ESPNMapping struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

type ESPNGroup struct {
	Entries       []ESPNEntry       `json:"entries"`
	GroupSettings ESPNGroupSettings `json:"groupSettings"`
}

type ESPNGroupSettings struct {
	Name string `json:"name"`
}

type ESPNEntry struct {
	Name            string               `json:"name"`
	Member          ESPNMember           `json:"member"`
	Picks           []ESPNPick           `json:"picks"`
	Score           ESPNScore            `json:"score"`
	FinalPick       ESPNPick             `json:"finalPick"`
	TiebreakAnswers []ESPNTiebreakAnswer `json:"tiebreakAnswers"`
}

type ESPNTiebreakAnswer struct {
	Answer float64 `json:"answer"`
}

type ESPNMember struct {
	DisplayName string `json:"displayName"`
}

type ESPNPick struct {
	PropositionID  string              `json:"propositionId"`
	PeriodReached  int                 `json:"periodReached"`
	OutcomesPicked []ESPNOutcomePicked `json:"outcomesPicked"`
}

type ESPNOutcomePicked struct {
	OutcomeID string `json:"outcomeId"`
	Result    string `json:"result"`
}

type ESPNScore struct {
	OverallScore      int        `json:"overallScore"`
	PossiblePointsMax int        `json:"possiblePointsMax"`
	Rank              int        `json:"rank"`
	Eliminated        bool       `json:"eliminated"`
	Percentile        float64    `json:"percentile"`
	Record            ESPNRecord `json:"record"`
}

type ESPNRecord struct {
	Wins   int `json:"wins"`
	Losses int `json:"losses"`
}

// Output types

type Team struct {
	Name   string `json:"name"`
	Abbrev string `json:"abbrev"`
	Seed   int    `json:"seed"`
	Region int    `json:"region"`
	Record string `json:"record"`
	Logo   string `json:"logo"`
}

type PickInfo struct {
	Count   int      `json:"count"`
	Entries []string `json:"entries"`
}

type Matchup struct {
	ID           string              `json:"id"`
	Region       int                 `json:"region"`
	DisplayOrder int                 `json:"displayOrder"`
	Team1ID      string              `json:"team1Id"`
	Team2ID      string              `json:"team2Id"`
	WinnerID     string              `json:"winnerId,omitempty"`
	GameTime     *int64              `json:"gameTime,omitempty"`
	Status       string              `json:"status"`
	Picks        map[string]PickInfo `json:"picks"`
}

type Round struct {
	Status   string    `json:"status"`
	Matchups []Matchup `json:"matchups"`
}

type Output struct {
	GroupName      string           `json:"groupName"`
	LastUpdated    string           `json:"lastUpdated"`
	Version        string           `json:"version"`
	PointsPerRound []int            `json:"pointsPerRound"`
	Teams          map[string]Team  `json:"teams"`
	Rounds         map[string]Round `json:"rounds"`
	Brackets       []interface{}    `json:"brackets"`
}

// Leaderboard types

type LeaderboardEntry struct {
	EntryName   string   `json:"entryName"`
	Member      string   `json:"member"`
	Score       int      `json:"score"`
	MaxPossible int      `json:"maxPossible"`
	Correct     int      `json:"correct"`
	Incorrect   int      `json:"incorrect"`
	Rank        int      `json:"rank"`
	Percentile  float64  `json:"percentile"`
	Eliminated  bool     `json:"eliminated"`
	Tiebreaker  *float64 `json:"tiebreaker,omitempty"`
	Champion    string   `json:"champion"`
	FinalFour   []string `json:"finalFour"`
}

type LeaderboardOutput struct {
	Entries []LeaderboardEntry `json:"entries"`
}

// parseHex extracts the first hex segment (before '-') and returns its int64 value.
func parseHex(id string) int64 {
	seg := id
	if i := strings.IndexByte(id, '-'); i >= 0 {
		seg = id[:i]
	}
	val := int64(0)
	for _, c := range seg {
		val = val * 16
		if c >= '0' && c <= '9' {
			val += int64(c - '0')
		} else if c >= 'a' && c <= 'f' {
			val += int64(c - 'a' + 10)
		}
	}
	return val
}

func main() {
	// Load ESPN data
	challengeData, err := os.ReadFile("data/espn/challenge.json")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading challenge: %v\n", err)
		os.Exit(1)
	}
	var ch ESPNChallenge
	if err := json.Unmarshal(challengeData, &ch); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing challenge: %v\n", err)
		os.Exit(1)
	}

	groupData, err := os.ReadFile("data/espn/group.json")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading group: %v\n", err)
		os.Exit(1)
	}
	var g ESPNGroup
	if err := json.Unmarshal(groupData, &g); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing group: %v\n", err)
		os.Exit(1)
	}

	// Current outcome ID → team abbrev
	outcomeToAbbrev := make(map[string]string)
	currentOutcomeIDs := make(map[string]bool)
	for _, prop := range ch.Propositions {
		for _, o := range prop.PossibleOutcomes {
			outcomeToAbbrev[o.ID] = o.Abbrev
			currentOutcomeIDs[o.ID] = true
		}
	}

	// Build teams map
	teams := make(map[string]Team)
	for _, prop := range ch.Propositions {
		for _, o := range prop.PossibleOutcomes {
			logo := ""
			for _, m := range o.Mappings {
				if m.Type == "IMAGE_PRIMARY" {
					logo = m.Value
					break
				}
			}
			teams[o.Abbrev] = Team{
				Name:   o.Name,
				Abbrev: o.Abbrev,
				Seed:   o.RegionSeed,
				Region: o.RegionID,
				Record: o.AdditionalInfo,
				Logo:   logo,
			}
		}
	}

	// Detect current period from propositions
	currentPeriod := ch.Propositions[0].ScoringPeriodID

	var r64Matchups, r32Matchups, s16Matchups []Matchup

	if currentPeriod == 2 {
		// ============================================================
		// Period 2 (R32): 4 outcomes per prop → 2 R64 games + 1 R32 game
		// ============================================================
		r64Matchups, r32Matchups = buildFromR32Props(ch, g, outcomeToAbbrev)

	} else if currentPeriod == 3 {
		// ============================================================
		// Period 3 (S16): 8 outcomes per prop → 4 R64 + 2 R32 + 1 S16
		// ============================================================
		r64Matchups, r32Matchups, s16Matchups = buildFromS16Props(ch, g, outcomeToAbbrev, currentOutcomeIDs)
	}

	// Sort matchups
	sortMatchups := func(ms []Matchup) {
		sort.Slice(ms, func(i, j int) bool {
			if ms[i].Region != ms[j].Region {
				return ms[i].Region < ms[j].Region
			}
			return ms[i].DisplayOrder < ms[j].DisplayOrder
		})
	}
	sortMatchups(r64Matchups)
	sortMatchups(r32Matchups)
	sortMatchups(s16Matchups)

	// Derive round status
	deriveStatus := func(matchups []Matchup) string {
		if len(matchups) == 0 {
			return "future"
		}
		allComplete, anyStarted := true, false
		for _, m := range matchups {
			if m.Status == "COMPLETE" || m.Status == "PLAYING" || m.Status == "LOCKED" {
				anyStarted = true
			}
			if m.Status != "COMPLETE" {
				allComplete = false
			}
		}
		if allComplete {
			return "complete"
		}
		if anyStarted {
			return "in_progress"
		}
		return "future"
	}

	version := "dev"
	if vb, err := os.ReadFile("VERSION"); err == nil {
		version = strings.TrimSpace(string(vb))
	}

	out := Output{
		GroupName:      g.GroupSettings.Name,
		LastUpdated:    time.Now().UTC().Format(time.RFC3339),
		Version:        version,
		PointsPerRound: []int{10, 20, 40, 80, 160, 320},
		Teams:          teams,
		Rounds: map[string]Round{
			"r64":          {Status: deriveStatus(r64Matchups), Matchups: r64Matchups},
			"r32":          {Status: deriveStatus(r32Matchups), Matchups: r32Matchups},
			"sweet16":      {Status: deriveStatus(s16Matchups), Matchups: s16Matchups},
			"elite8":       {Status: "future", Matchups: []Matchup{}},
			"finalFour":    {Status: "future", Matchups: []Matchup{}},
			"championship": {Status: "future", Matchups: []Matchup{}},
		},
		Brackets: []interface{}{},
	}

	outBytes, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling output: %v\n", err)
		os.Exit(1)
	}
	js := append([]byte("const DATA = "), outBytes...)
	js = append(js, ';')
	if err := os.WriteFile("data/data.js", js, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing data.js: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Wrote data/data.js")

	// ============================================================
	// Generate leaderboard.js
	// ============================================================
	generateLeaderboard(ch, g, outcomeToAbbrev, currentOutcomeIDs)
}

// buildFromR32Props handles period 2 (R32): each prop has 4 outcomes.
func buildFromR32Props(ch ESPNChallenge, g ESPNGroup, outcomeToAbbrev map[string]string) ([]Matchup, []Matchup) {
	var r64Matchups, r32Matchups []Matchup

	for _, prop := range ch.Propositions {
		byPos := make(map[int]ESPNOutcome)
		for _, o := range prop.PossibleOutcomes {
			byPos[o.MatchupPosition] = o
		}

		actual := make(map[string]bool)
		for _, oid := range prop.ActualOutcomeIDs {
			actual[oid] = true
		}

		region := prop.PossibleOutcomes[0].RegionID
		t1, t2, t3, t4 := byPos[1], byPos[2], byPos[3], byPos[4]

		winnerA := ""
		if actual[t1.ID] {
			winnerA = t1.Abbrev
		} else if actual[t2.ID] {
			winnerA = t2.Abbrev
		}
		winnerB := ""
		if actual[t3.ID] {
			winnerB = t3.Abbrev
		} else if actual[t4.ID] {
			winnerB = t4.Abbrev
		}

		gameAPicks := make(map[string]PickInfo)
		gameBPicks := make(map[string]PickInfo)
		r32Picks := make(map[string]PickInfo)

		for _, entry := range g.Entries {
			for _, pick := range entry.Picks {
				if pick.PropositionID != prop.ID || len(pick.OutcomesPicked) == 0 {
					continue
				}
				oid := pick.OutcomesPicked[0].OutcomeID
				abbrev := outcomeToAbbrev[oid]
				pos := 0
				for _, o := range prop.PossibleOutcomes {
					if o.ID == oid {
						pos = o.MatchupPosition
						break
					}
				}

				if abbrev == winnerA || abbrev == winnerB {
					pi := r32Picks[abbrev]
					pi.Count++
					pi.Entries = append(pi.Entries, entry.Name)
					r32Picks[abbrev] = pi
				}

				if pos == 1 || pos == 2 {
					pi := gameAPicks[abbrev]
					pi.Count++
					pi.Entries = append(pi.Entries, entry.Name)
					gameAPicks[abbrev] = pi
				} else {
					pi := gameBPicks[abbrev]
					pi.Count++
					pi.Entries = append(pi.Entries, entry.Name)
					gameBPicks[abbrev] = pi
				}
			}
		}

		r64Matchups = append(r64Matchups, Matchup{
			ID: prop.ID + "-r64a", Region: region,
			DisplayOrder: prop.DisplayOrder*2 - 1,
			Team1ID: t1.Abbrev, Team2ID: t2.Abbrev,
			WinnerID: winnerA, Status: "COMPLETE",
			Picks: gameAPicks,
		})
		r64Matchups = append(r64Matchups, Matchup{
			ID: prop.ID + "-r64b", Region: region,
			DisplayOrder: prop.DisplayOrder * 2,
			Team1ID: t3.Abbrev, Team2ID: t4.Abbrev,
			WinnerID: winnerB, Status: "COMPLETE",
			Picks: gameBPicks,
		})

		r32Winner := ""
		for _, oid := range prop.CorrectOutcomes {
			if abbrev, ok := outcomeToAbbrev[oid]; ok {
				r32Winner = abbrev
			}
		}
		r32Matchups = append(r32Matchups, Matchup{
			ID: prop.ID, Region: region,
			DisplayOrder: prop.DisplayOrder,
			Team1ID: winnerA, Team2ID: winnerB,
			WinnerID: r32Winner, GameTime: prop.Date,
			Status: prop.Status, Picks: r32Picks,
		})
	}

	return r64Matchups, r32Matchups
}

// buildFromS16Props handles period 3 (S16): each prop has 8 outcomes.
// Reconstructs R64, R32, and S16 matchups using current props + old entry picks.
func buildFromS16Props(ch ESPNChallenge, g ESPNGroup, outcomeToAbbrev map[string]string, currentOutcomeIDs map[string]bool) ([]Matchup, []Matchup, []Matchup) {
	// Sort S16 props by displayOrder
	s16Props := make([]ESPNProposition, len(ch.Propositions))
	copy(s16Props, ch.Propositions)
	sort.Slice(s16Props, func(i, j int) bool {
		return s16Props[i].DisplayOrder < s16Props[j].DisplayOrder
	})

	// Collect all old (non-current) proposition IDs from entry picks, sorted by hex.
	// Structure: first 32 = R64 props, next 16 = R32 props, rest = future rounds.
	oldPropSet := make(map[string]bool)
	for _, entry := range g.Entries {
		for _, pick := range entry.Picks {
			if len(pick.OutcomesPicked) == 0 {
				continue
			}
			if !currentOutcomeIDs[pick.OutcomesPicked[0].OutcomeID] {
				oldPropSet[pick.PropositionID] = true
			}
		}
	}
	oldProps := make([]string, 0, len(oldPropSet))
	for pid := range oldPropSet {
		oldProps = append(oldProps, pid)
	}
	sort.Slice(oldProps, func(i, j int) bool {
		return parseHex(oldProps[i]) < parseHex(oldProps[j])
	})

	r64OldProps := oldProps[:32] // First 32 old props = R64 games
	r32OldProps := oldProps[32:48] // Next 16 = R32 games

	// ============================================================
	// Determine R64 winners from old entry picks using hex offset mapping.
	//
	// R64 props map to S16 props in groups of 4 (matching displayOrder sort):
	//   r64OldProps[s16idx*4 + gameIdx] → S16 prop's R64 game
	//   gameIdx 0: pos 1 vs 2, gameIdx 1: pos 3 vs 4,
	//   gameIdx 2: pos 5 vs 6, gameIdx 3: pos 7 vs 8
	//
	// Within each R64 prop, outcome offset 1 = odd position team,
	// offset 2 = even position team.
	// ============================================================

	// r64Winners[s16idx][gameIdx] = winner abbrev
	r64Winners := make([]map[int]string, len(s16Props))
	for i := range r64Winners {
		r64Winners[i] = make(map[int]string)
	}

	for propIdx, r64Pid := range r64OldProps {
		s16Idx := propIdx / 4
		gameIdx := propIdx % 4
		prop := s16Props[s16Idx]

		pos1 := gameIdx*2 + 1
		pos2 := gameIdx*2 + 2
		var team1, team2 ESPNOutcome
		for _, o := range prop.PossibleOutcomes {
			if o.MatchupPosition == pos1 {
				team1 = o
			}
			if o.MatchupPosition == pos2 {
				team2 = o
			}
		}

		r64PidBase := parseHex(r64Pid)
		found := false
		for _, entry := range g.Entries {
			if found {
				break
			}
			for _, pick := range entry.Picks {
				if pick.PropositionID != r64Pid || len(pick.OutcomesPicked) == 0 {
					continue
				}
				oid := pick.OutcomesPicked[0].OutcomeID
				result := pick.OutcomesPicked[0].Result
				offset := int(parseHex(oid) - r64PidBase)

				if result == "CORRECT" {
					if offset == 1 {
						r64Winners[s16Idx][gameIdx] = team1.Abbrev
					} else {
						r64Winners[s16Idx][gameIdx] = team2.Abbrev
					}
					found = true
					break
				} else if result == "INCORRECT" {
					if offset == 1 {
						r64Winners[s16Idx][gameIdx] = team2.Abbrev
					} else {
						r64Winners[s16Idx][gameIdx] = team1.Abbrev
					}
					found = true
					break
				}
			}
		}
	}

	// ============================================================
	// Build old outcome ID → team abbrev mapping for pick aggregation.
	//
	// For R64 old props: offset 1 → odd-position team, offset 2 → even-position team.
	// For R32 old props: offsets 1-4 → S16 positions from the relevant half.
	// ============================================================
	oldOutcomeToAbbrev := make(map[string]string)

	// Map R64 old prop outcomes
	for propIdx, r64Pid := range r64OldProps {
		s16Idx := propIdx / 4
		gameIdx := propIdx % 4
		prop := s16Props[s16Idx]

		pos1 := gameIdx*2 + 1
		pos2 := gameIdx*2 + 2
		var team1Abbrev, team2Abbrev string
		for _, o := range prop.PossibleOutcomes {
			if o.MatchupPosition == pos1 {
				team1Abbrev = o.Abbrev
			}
			if o.MatchupPosition == pos2 {
				team2Abbrev = o.Abbrev
			}
		}

		r64PidBase := parseHex(r64Pid)
		// Collect all outcome IDs used across entries for this prop
		for _, entry := range g.Entries {
			for _, pick := range entry.Picks {
				if pick.PropositionID != r64Pid || len(pick.OutcomesPicked) == 0 {
					continue
				}
				oid := pick.OutcomesPicked[0].OutcomeID
				offset := int(parseHex(oid) - r64PidBase)
				if offset == 1 {
					oldOutcomeToAbbrev[oid] = team1Abbrev
				} else if offset == 2 {
					oldOutcomeToAbbrev[oid] = team2Abbrev
				}
			}
		}
	}

	// Map R32 old prop outcomes
	for propIdx, r32Pid := range r32OldProps {
		s16Idx := propIdx / 2
		half := propIdx % 2 // 0=top (pos 1-4), 1=bottom (pos 5-8)
		prop := s16Props[s16Idx]

		// Build position → abbrev for this half
		posToAbbrev := make(map[int]string)
		for _, o := range prop.PossibleOutcomes {
			posToAbbrev[o.MatchupPosition] = o.Abbrev
		}

		r32PidBase := parseHex(r32Pid)
		for _, entry := range g.Entries {
			for _, pick := range entry.Picks {
				if pick.PropositionID != r32Pid || len(pick.OutcomesPicked) == 0 {
					continue
				}
				oid := pick.OutcomesPicked[0].OutcomeID
				offset := int(parseHex(oid) - r32PidBase)
				// Offsets 1-4 map to positions in the relevant S16 half
				if offset >= 1 && offset <= 4 {
					pos := offset
					if half == 1 {
						pos = offset + 4 // bottom half: offset 1→pos5, 2→pos6, etc.
					}
					if abbrev, ok := posToAbbrev[pos]; ok {
						oldOutcomeToAbbrev[oid] = abbrev
					}
				}
			}
		}
	}

	// Merge old and current outcome mappings
	allOutcomeToAbbrev := make(map[string]string)
	for k, v := range outcomeToAbbrev {
		allOutcomeToAbbrev[k] = v
	}
	for k, v := range oldOutcomeToAbbrev {
		allOutcomeToAbbrev[k] = v
	}

	// ============================================================
	// Build R64 matchups
	// ============================================================
	var r64Matchups []Matchup
	for s16Idx, prop := range s16Props {
		region := prop.PossibleOutcomes[0].RegionID
		byPos := make(map[int]ESPNOutcome)
		for _, o := range prop.PossibleOutcomes {
			byPos[o.MatchupPosition] = o
		}

		for gameIdx := 0; gameIdx < 4; gameIdx++ {
			pos1 := gameIdx*2 + 1
			pos2 := gameIdx*2 + 2
			t1 := byPos[pos1]
			t2 := byPos[pos2]
			winner := r64Winners[s16Idx][gameIdx]

			// Aggregate R64 picks from the corresponding old R64 prop
			r64Picks := make(map[string]PickInfo)
			r64Pid := r64OldProps[s16Idx*4+gameIdx]
			for _, entry := range g.Entries {
				for _, pick := range entry.Picks {
					if pick.PropositionID != r64Pid || len(pick.OutcomesPicked) == 0 {
						continue
					}
					abbrev := allOutcomeToAbbrev[pick.OutcomesPicked[0].OutcomeID]
					if abbrev != "" {
						pi := r64Picks[abbrev]
						pi.Count++
						pi.Entries = append(pi.Entries, entry.Name)
						r64Picks[abbrev] = pi
					}
				}
			}

			r64Matchups = append(r64Matchups, Matchup{
				ID:           prop.ID + fmt.Sprintf("-r64-%d", gameIdx),
				Region:       region,
				DisplayOrder: prop.DisplayOrder*4 + gameIdx,
				Team1ID:      t1.Abbrev,
				Team2ID:      t2.Abbrev,
				WinnerID:     winner,
				Status:       "COMPLETE",
				Picks:        r64Picks,
			})
		}
	}

	// ============================================================
	// Build R32 matchups
	// ============================================================
	var r32Matchups []Matchup
	for s16Idx, prop := range s16Props {
		region := prop.PossibleOutcomes[0].RegionID
		oidToAbbrev := make(map[string]string)
		for _, o := range prop.PossibleOutcomes {
			oidToAbbrev[o.ID] = o.Abbrev
		}

		actualSet := make(map[string]bool)
		for _, aid := range prop.ActualOutcomeIDs {
			actualSet[oidToAbbrev[aid]] = true
		}

		for half := 0; half < 2; half++ {
			// R32 contestants = R64 winners from each pair of games
			game1 := half * 2       // games 0,1 or 2,3
			game2 := half*2 + 1
			teamA := r64Winners[s16Idx][game1]
			teamB := r64Winners[s16Idx][game2]

			r32Winner := ""
			if actualSet[teamA] {
				r32Winner = teamA
			} else if actualSet[teamB] {
				r32Winner = teamB
			}

			// Aggregate R32 picks from the corresponding old R32 prop
			r32Picks := make(map[string]PickInfo)
			r32Pid := r32OldProps[s16Idx*2+half]
			for _, entry := range g.Entries {
				for _, pick := range entry.Picks {
					if pick.PropositionID != r32Pid || len(pick.OutcomesPicked) == 0 {
						continue
					}
					abbrev := allOutcomeToAbbrev[pick.OutcomesPicked[0].OutcomeID]
					// Only count picks for the actual R32 contestants
					if abbrev == teamA || abbrev == teamB {
						pi := r32Picks[abbrev]
						pi.Count++
						pi.Entries = append(pi.Entries, entry.Name)
						r32Picks[abbrev] = pi
					}
				}
			}

			r32Matchups = append(r32Matchups, Matchup{
				ID:           prop.ID + fmt.Sprintf("-r32-%d", half),
				Region:       region,
				DisplayOrder: prop.DisplayOrder*2 + half,
				Team1ID:      teamA,
				Team2ID:      teamB,
				WinnerID:     r32Winner,
				Status:       "COMPLETE",
				Picks:        r32Picks,
			})
		}
	}

	// ============================================================
	// Build S16 matchups
	// ============================================================
	var s16Matchups []Matchup
	for _, prop := range s16Props {
		region := prop.PossibleOutcomes[0].RegionID
		oidToAbbrev := make(map[string]string)
		for _, o := range prop.PossibleOutcomes {
			oidToAbbrev[o.ID] = o.Abbrev
		}

		// S16 contestants = the 2 R32 winners (actualOutcomeIds)
		var team1, team2 string
		if len(prop.ActualOutcomeIDs) >= 2 {
			team1 = oidToAbbrev[prop.ActualOutcomeIDs[0]]
			team2 = oidToAbbrev[prop.ActualOutcomeIDs[1]]
		} else if len(prop.ActualOutcomeIDs) == 1 {
			team1 = oidToAbbrev[prop.ActualOutcomeIDs[0]]
		}

		// S16 winner from correctOutcomes
		s16Winner := ""
		for _, cid := range prop.CorrectOutcomes {
			if abbrev, ok := oidToAbbrev[cid]; ok {
				s16Winner = abbrev
			}
		}

		// Aggregate S16 picks from current prop entries
		s16Picks := make(map[string]PickInfo)
		for _, entry := range g.Entries {
			for _, pick := range entry.Picks {
				if pick.PropositionID != prop.ID || len(pick.OutcomesPicked) == 0 {
					continue
				}
				oid := pick.OutcomesPicked[0].OutcomeID
				abbrev := outcomeToAbbrev[oid]
				// Only count picks for actual S16 contestants
				if abbrev == team1 || abbrev == team2 {
					pi := s16Picks[abbrev]
					pi.Count++
					pi.Entries = append(pi.Entries, entry.Name)
					s16Picks[abbrev] = pi
				}
			}
		}

		s16Matchups = append(s16Matchups, Matchup{
			ID:           prop.ID,
			Region:       region,
			DisplayOrder: prop.DisplayOrder,
			Team1ID:      team1,
			Team2ID:      team2,
			WinnerID:     s16Winner,
			GameTime:     prop.Date,
			Status:       prop.Status,
			Picks:        s16Picks,
		})
	}

	return r64Matchups, r32Matchups, s16Matchups
}

func generateLeaderboard(ch ESPNChallenge, g ESPNGroup, outcomeToAbbrev map[string]string, currentOutcomeIDs map[string]bool) {
	// Map championship finalPick outcome IDs to team abbrevs using hex offset.
	bracketOrder := [16]int{1, 16, 8, 9, 5, 12, 4, 13, 6, 11, 3, 14, 7, 10, 2, 15}

	rsToAbbrev := make(map[[2]int]string)
	for _, prop := range ch.Propositions {
		for _, o := range prop.PossibleOutcomes {
			rsToAbbrev[[2]int{o.RegionID, o.RegionSeed}] = o.Abbrev
		}
	}

	fpOutcomes := make(map[string]bool)
	for _, entry := range g.Entries {
		if len(entry.FinalPick.OutcomesPicked) > 0 {
			fpOutcomes[entry.FinalPick.OutcomesPicked[0].OutcomeID] = true
		}
	}

	var champBase int64
	first := true
	for oid := range fpOutcomes {
		v := parseHex(oid)
		if first || v < champBase {
			champBase = v
			first = false
		}
	}

	champMap := make(map[string]string)
	for oid := range fpOutcomes {
		offset := int(parseHex(oid) - champBase)
		region := offset/16 + 1
		pos := offset % 16
		if region >= 1 && region <= 4 && pos >= 0 && pos < 16 {
			seed := bracketOrder[pos]
			if abbrev, ok := rsToAbbrev[[2]int{region, seed}]; ok {
				champMap[oid] = abbrev
			}
		}
	}

	// Build old outcome ID → team abbrev for Final Four resolution.
	// Need to map old pick outcome IDs to team abbrevs using the same hex approach.
	// Collect all old prop IDs, sort by hex, map R64 props to S16 positions.
	oldPropSet := make(map[string]bool)
	for _, entry := range g.Entries {
		for _, pick := range entry.Picks {
			if len(pick.OutcomesPicked) == 0 {
				continue
			}
			if !currentOutcomeIDs[pick.OutcomesPicked[0].OutcomeID] {
				oldPropSet[pick.PropositionID] = true
			}
		}
	}
	oldProps := make([]string, 0, len(oldPropSet))
	for pid := range oldPropSet {
		oldProps = append(oldProps, pid)
	}
	sort.Slice(oldProps, func(i, j int) bool {
		return parseHex(oldProps[i]) < parseHex(oldProps[j])
	})

	// Build old outcome → abbrev mapping
	oldOutcomeToAbbrev := make(map[string]string)

	s16Props := make([]ESPNProposition, len(ch.Propositions))
	copy(s16Props, ch.Propositions)
	sort.Slice(s16Props, func(i, j int) bool {
		return s16Props[i].DisplayOrder < s16Props[j].DisplayOrder
	})

	if len(oldProps) >= 48 {
		// Map R64 old prop outcomes (first 32)
		for propIdx := 0; propIdx < 32 && propIdx < len(oldProps); propIdx++ {
			r64Pid := oldProps[propIdx]
			s16Idx := propIdx / 4
			gameIdx := propIdx % 4
			if s16Idx >= len(s16Props) {
				continue
			}
			prop := s16Props[s16Idx]

			pos1 := gameIdx*2 + 1
			pos2 := gameIdx*2 + 2
			var t1Abbrev, t2Abbrev string
			for _, o := range prop.PossibleOutcomes {
				if o.MatchupPosition == pos1 {
					t1Abbrev = o.Abbrev
				}
				if o.MatchupPosition == pos2 {
					t2Abbrev = o.Abbrev
				}
			}

			r64Base := parseHex(r64Pid)
			for _, entry := range g.Entries {
				for _, pick := range entry.Picks {
					if pick.PropositionID != r64Pid || len(pick.OutcomesPicked) == 0 {
						continue
					}
					oid := pick.OutcomesPicked[0].OutcomeID
					offset := int(parseHex(oid) - r64Base)
					if offset == 1 {
						oldOutcomeToAbbrev[oid] = t1Abbrev
					} else if offset == 2 {
						oldOutcomeToAbbrev[oid] = t2Abbrev
					}
				}
			}
		}

		// Map R32 old prop outcomes (next 16)
		for propIdx := 0; propIdx < 16 && propIdx+32 < len(oldProps); propIdx++ {
			r32Pid := oldProps[32+propIdx]
			s16Idx := propIdx / 2
			half := propIdx % 2
			if s16Idx >= len(s16Props) {
				continue
			}
			prop := s16Props[s16Idx]

			posToAbbrev := make(map[int]string)
			for _, o := range prop.PossibleOutcomes {
				posToAbbrev[o.MatchupPosition] = o.Abbrev
			}

			r32Base := parseHex(r32Pid)
			for _, entry := range g.Entries {
				for _, pick := range entry.Picks {
					if pick.PropositionID != r32Pid || len(pick.OutcomesPicked) == 0 {
						continue
					}
					oid := pick.OutcomesPicked[0].OutcomeID
					offset := int(parseHex(oid) - r32Base)
					if offset >= 1 && offset <= 4 {
						pos := offset
						if half == 1 {
							pos = offset + 4
						}
						if abbrev, ok := posToAbbrev[pos]; ok {
							oldOutcomeToAbbrev[oid] = abbrev
						}
					}
				}
			}
		}

		// Map remaining old props (E8, FF, Championship) using wider offsets
		// E8 props (4 props) have 8 outcomes each, FF props have 16, Championship has 32+
		// Use the same S16 position mapping but with larger offset ranges
		for propIdx := 48; propIdx < len(oldProps); propIdx++ {
			pid := oldProps[propIdx]
			pidBase := parseHex(pid)

			// Determine which S16 props this covers based on index
			// E8: props 48-51, each covers 2 S16 props (= 1 region quarter)
			// FF: props 52-53, each covers 4 S16 props (= 2 regions)
			// Championship: prop 54, covers all 8 S16 props
			for _, entry := range g.Entries {
				for _, pick := range entry.Picks {
					if pick.PropositionID != pid || len(pick.OutcomesPicked) == 0 {
						continue
					}
					oid := pick.OutcomesPicked[0].OutcomeID
					offset := int(parseHex(oid) - pidBase)

					// For E8/FF/Championship, outcomes follow the same bracket ordering
					// as the full tournament. Each S16 prop contributes 8 positions.
					// We can map by: figure out which S16 prop this offset falls in,
					// then which position within that prop.
					var s16Idx, posInProp int
					switch {
					case propIdx < 52: // E8 (4 props)
						e8Idx := propIdx - 48
						// Each E8 prop covers 2 S16 props (= 16 teams)
						localOffset := offset - 1
						if localOffset < 0 || localOffset >= 16 {
							continue
						}
						s16Idx = e8Idx*2 + localOffset/8
						posInProp = (localOffset % 8) + 1
					case propIdx < 54: // FF (2 props)
						ffIdx := propIdx - 52
						localOffset := offset - 1
						if localOffset < 0 || localOffset >= 32 {
							continue
						}
						s16Idx = ffIdx*4 + localOffset/8
						posInProp = (localOffset % 8) + 1
					default: // Championship (1 prop)
						localOffset := offset - 1
						if localOffset < 0 || localOffset >= 64 {
							continue
						}
						s16Idx = localOffset / 8
						posInProp = (localOffset % 8) + 1
					}

					if s16Idx < len(s16Props) {
						for _, o := range s16Props[s16Idx].PossibleOutcomes {
							if o.MatchupPosition == posInProp {
								oldOutcomeToAbbrev[oid] = o.Abbrev
								break
							}
						}
					}
				}
			}
		}
	}

	// Build combined outcome → abbrev (current + old)
	allOutcomeToAbbrev := make(map[string]string)
	for k, v := range outcomeToAbbrev {
		allOutcomeToAbbrev[k] = v
	}
	for k, v := range oldOutcomeToAbbrev {
		allOutcomeToAbbrev[k] = v
	}

	var lbEntries []LeaderboardEntry
	for _, entry := range g.Entries {
		champion := ""
		if len(entry.FinalPick.OutcomesPicked) > 0 {
			champion = champMap[entry.FinalPick.OutcomesPicked[0].OutcomeID]
		}

		var finalFour []string
		seen := make(map[string]bool)
		for _, pick := range entry.Picks {
			if pick.PeriodReached >= 5 && len(pick.OutcomesPicked) > 0 {
				abbrev := allOutcomeToAbbrev[pick.OutcomesPicked[0].OutcomeID]
				if abbrev != "" && !seen[abbrev] {
					seen[abbrev] = true
					finalFour = append(finalFour, abbrev)
				}
			}
		}

		var tiebreaker *float64
		if len(entry.TiebreakAnswers) > 0 {
			t := entry.TiebreakAnswers[0].Answer
			tiebreaker = &t
		}

		lbEntries = append(lbEntries, LeaderboardEntry{
			EntryName:   entry.Name,
			Member:      entry.Member.DisplayName,
			Score:       entry.Score.OverallScore,
			MaxPossible: entry.Score.PossiblePointsMax,
			Correct:     entry.Score.Record.Wins,
			Incorrect:   entry.Score.Record.Losses,
			Rank:        entry.Score.Rank,
			Percentile:  entry.Score.Percentile,
			Eliminated:  entry.Score.Eliminated,
			Tiebreaker:  tiebreaker,
			Champion:    champion,
			FinalFour:   finalFour,
		})
	}

	lbOut := LeaderboardOutput{Entries: lbEntries}
	lbBytes, err := json.MarshalIndent(lbOut, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling leaderboard: %v\n", err)
		os.Exit(1)
	}
	lbJS := append([]byte("const LEADERBOARD = "), lbBytes...)
	lbJS = append(lbJS, ';')
	if err := os.WriteFile("data/leaderboard.js", lbJS, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing leaderboard.js: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Wrote data/leaderboard.js")
}
