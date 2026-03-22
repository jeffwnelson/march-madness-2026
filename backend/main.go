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
	Name             string               `json:"name"`
	Member           ESPNMember           `json:"member"`
	Picks            []ESPNPick           `json:"picks"`
	Score            ESPNScore            `json:"score"`
	FinalPick        ESPNPick             `json:"finalPick"`
	TiebreakAnswers  []ESPNTiebreakAnswer `json:"tiebreakAnswers"`
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

	// R32 outcome ID → team abbrev (known from current challenge propositions)
	outcomeToAbbrev := make(map[string]string)
	for _, prop := range ch.Propositions {
		for _, o := range prop.PossibleOutcomes {
			outcomeToAbbrev[o.ID] = o.Abbrev
		}
	}

	// Build old outcome ID → team abbrev mapping.
	// Old R64 picks reference outcome IDs not in the current challenge.
	// For each old prop, the outcome with result=CORRECT across entries is the winner.
	// We know all 32 R64 winners from actualOutcomeIds → team abbrev.
	// We match by: for each old prop, find an entry with CORRECT result,
	// then find which R64 winner this entry also picked in their R32 pick.
	r32PropSet := make(map[string]bool)
	finalPickProps := make(map[string]bool)
	for _, prop := range ch.Propositions {
		r32PropSet[prop.ID] = true
	}
	for _, entry := range g.Entries {
		if entry.FinalPick.PropositionID != "" {
			finalPickProps[entry.FinalPick.PropositionID] = true
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

	// Build matchups
	var r64Matchups []Matchup
	var r32Matchups []Matchup

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
		if actual[t1.ID] { winnerA = t1.Abbrev } else if actual[t2.ID] { winnerA = t2.Abbrev }
		winnerB := ""
		if actual[t3.ID] { winnerB = t3.Abbrev } else if actual[t4.ID] { winnerB = t4.Abbrev }

		// Aggregate picks from R32 entries for R64 games.
		// Entry picks one of 4 outcomes. Pos 1 or 2 → game A pick. Pos 3 or 4 → game B pick.
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
					if o.ID == oid { pos = o.MatchupPosition; break }
				}

				// R32 picks — only track picks for the two actual contestants
				if abbrev == winnerA || abbrev == winnerB {
					pi := r32Picks[abbrev]
					pi.Count++
					pi.Entries = append(pi.Entries, entry.Name)
					r32Picks[abbrev] = pi
				}

				// R64 game picks based on position
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

		// R32 matchup
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

	// Sort
	sortMatchups := func(ms []Matchup) {
		sort.Slice(ms, func(i, j int) bool {
			if ms[i].Region != ms[j].Region { return ms[i].Region < ms[j].Region }
			return ms[i].DisplayOrder < ms[j].DisplayOrder
		})
	}
	sortMatchups(r64Matchups)
	sortMatchups(r32Matchups)

	// Derive round status
	deriveStatus := func(matchups []Matchup) string {
		if len(matchups) == 0 { return "future" }
		allComplete, anyStarted := true, false
		for _, m := range matchups {
			if m.Status == "COMPLETE" || m.Status == "PLAYING" { anyStarted = true }
			if m.Status != "COMPLETE" { allComplete = false }
		}
		if allComplete { return "complete" }
		if anyStarted { return "in_progress" }
		return "future"
	}

	// Build version from latest git tag + short commit hash
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
			"sweet16":      {Status: "future", Matchups: []Matchup{}},
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

	// Map championship finalPick outcome IDs to team abbrevs using hex offset.
	// Championship outcomes follow: base + (region-1)*16 + bracketPosition
	bracketOrder := [16]int{1, 16, 8, 9, 5, 12, 4, 13, 6, 11, 3, 14, 7, 10, 2, 15}

	// region+seed → abbrev
	rsToAbbrev := make(map[[2]int]string)
	for _, prop := range ch.Propositions {
		for _, o := range prop.PossibleOutcomes {
			rsToAbbrev[[2]int{o.RegionID, o.RegionSeed}] = o.Abbrev
		}
	}

	// Collect all unique finalPick outcome IDs, find the base (lowest hex)
	fpOutcomes := make(map[string]bool)
	for _, entry := range g.Entries {
		if len(entry.FinalPick.OutcomesPicked) > 0 {
			fpOutcomes[entry.FinalPick.OutcomesPicked[0].OutcomeID] = true
		}
	}

	parseHex := func(id string) int64 {
		seg := id
		if idx := len(id); idx > 0 {
			for i, c := range id {
				if c == '-' {
					seg = id[:i]
					break
				}
				_ = c
			}
		}
		val := int64(0)
		for _, c := range seg {
			val = val*16
			if c >= '0' && c <= '9' {
				val += int64(c - '0')
			} else if c >= 'a' && c <= 'f' {
				val += int64(c - 'a' + 10)
			}
		}
		return val
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

	// Map each finalPick outcome → team abbrev
	champMap := make(map[string]string) // finalPick outcome ID → team abbrev
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

	// Build FinalFour picks per entry.
	// Entries with periodReached >= 5 on their R32 picks = team reaches Final Four.
	// The R32 pick outcome → known abbrev.
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
				abbrev := outcomeToAbbrev[pick.OutcomesPicked[0].OutcomeID]
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
