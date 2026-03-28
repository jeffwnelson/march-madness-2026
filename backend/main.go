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

// Round metadata: key, total games, scoring period
var roundDefs = []struct {
	Key        string
	TotalGames int
	Period     int
}{
	{"r64", 32, 1},
	{"r32", 16, 2},
	{"sweet16", 8, 3},
	{"elite8", 4, 4},
	{"finalFour", 2, 5},
	{"championship", 1, 6},
}

func main() {
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

	roundMatchups := buildAllMatchups(ch, g, outcomeToAbbrev, currentOutcomeIDs)

	// Sort each round's matchups
	for _, matchups := range roundMatchups {
		sort.Slice(matchups, func(i, j int) bool {
			if matchups[i].Region != matchups[j].Region {
				return matchups[i].Region < matchups[j].Region
			}
			return matchups[i].DisplayOrder < matchups[j].DisplayOrder
		})
	}

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

	rounds := make(map[string]Round)
	for _, rd := range roundDefs {
		ms := roundMatchups[rd.Key]
		if ms == nil {
			ms = []Matchup{}
		}
		rounds[rd.Key] = Round{Status: deriveStatus(ms), Matchups: ms}
	}

	out := Output{
		GroupName:      g.GroupSettings.Name,
		LastUpdated:    time.Now().UTC().Format(time.RFC3339),
		Version:        version,
		PointsPerRound: []int{10, 20, 40, 80, 160, 320},
		Teams:          teams,
		Rounds:         rounds,
		Brackets:       []interface{}{},
	}

	outBytes, _ := json.MarshalIndent(out, "", "  ")
	js := append([]byte("const DATA = "), outBytes...)
	js = append(js, ';')
	if err := os.WriteFile("data/data.js", js, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing data.js: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Wrote data/data.js")

	generateLeaderboard(ch, g, outcomeToAbbrev, currentOutcomeIDs)
}

// buildAllMatchups handles any scoring period by reconstructing all rounds generically.
//
// ESPN's rolling proposition model: at period P, each prop has 2^P outcomes covering
// the full bracket subtree. Old entry picks (sorted by hex) map to earlier rounds:
//   - First 32 old props = R64, next 16 = R32, next 8 = S16, etc.
//
// For each old round R (period < current), outcomes per old prop = 2^R.
// Old prop j within current prop i: offset k maps to current prop position = j*(2^R) + k.
func buildAllMatchups(ch ESPNChallenge, g ESPNGroup, outcomeToAbbrev map[string]string, currentOutcomeIDs map[string]bool) map[string][]Matchup {
	period := ch.Propositions[0].ScoringPeriodID

	// Sort current props by displayOrder
	curProps := make([]ESPNProposition, len(ch.Propositions))
	copy(curProps, ch.Propositions)
	sort.Slice(curProps, func(i, j int) bool {
		return parseHex(curProps[i].ID) < parseHex(curProps[j].ID)
	})
	numProps := len(curProps)

	// Position → team abbrev per current prop
	propTeams := make([]map[int]string, numProps)
	for i, prop := range curProps {
		propTeams[i] = make(map[int]string)
		for _, o := range prop.PossibleOutcomes {
			propTeams[i][o.MatchupPosition] = o.Abbrev
		}
	}

	// Collect old prop IDs sorted by hex
	oldPropSet := make(map[string]bool)
	for _, entry := range g.Entries {
		for _, pick := range entry.Picks {
			if len(pick.OutcomesPicked) > 0 && !currentOutcomeIDs[pick.OutcomesPicked[0].OutcomeID] {
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

	// Partition old props by round: R64(32), R32(16), S16(8), E8(4), FF(2), Champ(1)
	oldPropsByRound := make(map[int][]string) // round period → prop IDs
	idx := 0
	for _, rd := range roundDefs {
		if rd.Period >= period {
			break // Current and future rounds aren't in old props
		}
		end := idx + rd.TotalGames
		if end > len(oldProps) {
			break
		}
		oldPropsByRound[rd.Period] = oldProps[idx:end]
		idx = end
	}

	// Build old outcome ID → team abbrev mapping for ALL old rounds.
	// For old round with period R: each prop has 2^R outcomes.
	// Old prop j within current prop i: offset k → current position j*(2^R) + k.
	allOutcomeToAbbrev := make(map[string]string)
	for k, v := range outcomeToAbbrev {
		allOutcomeToAbbrev[k] = v
	}

	for roundPeriod, pids := range oldPropsByRound {
		outcomesPerOldProp := 1 << uint(roundPeriod)
		oldPropsPerCurProp := len(pids) / numProps

		for i, pid := range pids {
			curPropIdx := i / oldPropsPerCurProp
			localIdx := i % oldPropsPerCurProp
			base := parseHex(pid)

			for _, entry := range g.Entries {
				for _, pick := range entry.Picks {
					if pick.PropositionID != pid || len(pick.OutcomesPicked) == 0 {
						continue
					}
					oid := pick.OutcomesPicked[0].OutcomeID
					offset := int(parseHex(oid) - base)
					if offset >= 1 && offset <= outcomesPerOldProp {
						pos := localIdx*outcomesPerOldProp + offset
						if curPropIdx < len(propTeams) {
							if team, ok := propTeams[curPropIdx][pos]; ok {
								allOutcomeToAbbrev[oid] = team
							}
						}
					}
				}
			}
		}
	}

	// Also map future old props (E8/FF/Championship that haven't been played)
	// These are after the current-round old props in the sorted list.
	// Use wider offset ranges for deeper bracket props.
	for futureIdx := idx; futureIdx < len(oldProps); futureIdx++ {
		pid := oldProps[futureIdx]
		base := parseHex(pid)
		for _, entry := range g.Entries {
			for _, pick := range entry.Picks {
				if pick.PropositionID != pid || len(pick.OutcomesPicked) == 0 {
					continue
				}
				oid := pick.OutcomesPicked[0].OutcomeID
				offset := int(parseHex(oid) - base)

				// Figure out which round this future prop belongs to by its position
				// After all old completed rounds, remaining are future bracket games
				// grouped by round size. Determine which current prop + position.
				remaining := futureIdx - idx
				cumulative := 0
				for _, rd := range roundDefs {
					if rd.Period <= period {
						continue // Skip completed + current rounds
					}
					if remaining < cumulative+rd.TotalGames {
						localInRound := remaining - cumulative
						propsPerCur := rd.TotalGames / numProps
						if propsPerCur == 0 {
							propsPerCur = 1
						}
						curPropIdx := localInRound / propsPerCur
						localIdx := localInRound % propsPerCur
						outcomesPerProp := 1 << uint(rd.Period)
						if offset >= 1 && offset <= outcomesPerProp {
							pos := localIdx*outcomesPerProp + offset
							if curPropIdx < len(propTeams) {
								if team, ok := propTeams[curPropIdx][pos]; ok {
									allOutcomeToAbbrev[oid] = team
								}
							}
						}
						break
					}
					cumulative += rd.TotalGames
				}
			}
		}
	}

	// ============================================================
	// Determine winners for each completed round using old picks
	// ============================================================
	// winners[roundPeriod][globalGameIdx] = winner abbrev
	winners := make(map[int]map[int]string)

	for roundPeriod, pids := range oldPropsByRound {
		winners[roundPeriod] = make(map[int]string)
		outcomesPerOldProp := 1 << uint(roundPeriod)
		oldPropsPerCurProp := len(pids) / numProps

		for i, pid := range pids {
			curPropIdx := i / oldPropsPerCurProp
			localIdx := i % oldPropsPerCurProp
			base := parseHex(pid)

			// For R64: each prop has 2 outcomes → 1 game. Winner is the CORRECT one.
			// For R32: 4 outcomes → 1 game (pick 1 of 4 teams). CORRECT = winner.
			// For S16: 8 outcomes → 1 game. CORRECT = winner.
			found := false
			for _, entry := range g.Entries {
				if found {
					break
				}
				for _, pick := range entry.Picks {
					if pick.PropositionID != pid || len(pick.OutcomesPicked) == 0 {
						continue
					}
					oid := pick.OutcomesPicked[0].OutcomeID
					result := pick.OutcomesPicked[0].Result
					offset := int(parseHex(oid) - base)

					if result == "CORRECT" {
						pos := localIdx*outcomesPerOldProp + offset
						if curPropIdx < len(propTeams) {
							if team, ok := propTeams[curPropIdx][pos]; ok {
								winners[roundPeriod][i] = team
								found = true
								break
							}
						}
					} else if result == "INCORRECT" && roundPeriod == 1 {
						// For R64 (2 outcomes), if offset 1 is INCORRECT, winner is offset 2
						otherOffset := 3 - offset // if 1→2, if 2→1
						pos := localIdx*outcomesPerOldProp + otherOffset
						if curPropIdx < len(propTeams) {
							if team, ok := propTeams[curPropIdx][pos]; ok {
								winners[roundPeriod][i] = team
								found = true
								break
							}
						}
					}
				}
			}
		}
	}

	// Previous round winners from actualOutcomeIds
	prevPeriod := period - 1
	if prevPeriod >= 1 {
		winners[prevPeriod] = make(map[int]string)
		prevGamesPerProp := 1 << uint(period-prevPeriod) // games of prev round per current prop
		// For E8 (period 4), prev=S16 (period 3): 2 S16 games per E8 prop
		for propIdx, prop := range curProps {
			oidToPos := make(map[string]int)
			for _, o := range prop.PossibleOutcomes {
				oidToPos[o.ID] = o.MatchupPosition
			}
			for _, aid := range prop.ActualOutcomeIDs {
				pos := oidToPos[aid]
				team := propTeams[propIdx][pos]
				// Determine which game of the previous round this winner is from
				outcomesPerPrevGame := 1 << uint(prevPeriod) // positions covered per prev-round game
				gameInProp := (pos - 1) / outcomesPerPrevGame
				globalGameIdx := propIdx*prevGamesPerProp + gameInProp
				winners[prevPeriod][globalGameIdx] = team
			}
		}
	}

	// ============================================================
	// Build matchups for each round
	// ============================================================
	result := make(map[string][]Matchup)

	for _, rd := range roundDefs {
		if rd.Period > period {
			break // Future rounds
		}

		gamesPerProp := rd.TotalGames / numProps
		if gamesPerProp == 0 {
			gamesPerProp = 1
		}

		var matchups []Matchup

		for propIdx, prop := range curProps {
			region := prop.PossibleOutcomes[0].RegionID

			for gameInProp := 0; gameInProp < gamesPerProp; gameInProp++ {
				globalGameIdx := propIdx*gamesPerProp + gameInProp
				positionsPerGame := 1 << uint(rd.Period) // 2 for R64, 4 for R32, 8 for S16, etc.

				var team1, team2, winner string
				var gameTime *int64
				status := "COMPLETE"

				if rd.Period == 1 {
					// R64: teams from position pairs
					pos1 := gameInProp*2 + 1
					pos2 := gameInProp*2 + 2
					team1 = propTeams[propIdx][pos1]
					team2 = propTeams[propIdx][pos2]
					winner = winners[1][globalGameIdx]
				} else if rd.Period < period {
					// Completed intermediate round: contestants are winners from previous round
					prevGamesPerGame := 2 // each game fed by 2 games from previous round
					prevRoundGamesPerProp := gamesPerProp * prevGamesPerGame
					prevGlobalBase := propIdx * prevRoundGamesPerProp
					game1Idx := prevGlobalBase + gameInProp*prevGamesPerGame
					game2Idx := game1Idx + 1
					team1 = winners[rd.Period-1][game1Idx]
					team2 = winners[rd.Period-1][game2Idx]
					winner = winners[rd.Period][globalGameIdx]
				} else {
					// Current round: contestants from actualOutcomeIds, winner from correctOutcomes
					oidToAbbrev := make(map[string]string)
					oidToPos := make(map[string]int)
					for _, o := range prop.PossibleOutcomes {
						oidToAbbrev[o.ID] = o.Abbrev
						oidToPos[o.ID] = o.MatchupPosition
					}

					startPos := gameInProp*positionsPerGame + 1
					endPos := (gameInProp + 1) * positionsPerGame

					for _, aid := range prop.ActualOutcomeIDs {
						pos := oidToPos[aid]
						if pos >= startPos && pos <= endPos {
							halfSize := positionsPerGame / 2
							if pos <= startPos+halfSize-1 {
								team1 = oidToAbbrev[aid]
							} else {
								team2 = oidToAbbrev[aid]
							}
						}
					}

					for _, cid := range prop.CorrectOutcomes {
						if abbrev, ok := oidToAbbrev[cid]; ok {
							winner = abbrev
						}
					}

					gameTime = prop.Date
					status = prop.Status
				}

				// Aggregate picks
				picks := make(map[string]PickInfo)

				if rd.Period < period {
					// Old round: aggregate from old props
					if pids, ok := oldPropsByRound[rd.Period]; ok && globalGameIdx < len(pids) {
						pid := pids[globalGameIdx]
						for _, entry := range g.Entries {
							for _, pick := range entry.Picks {
								if pick.PropositionID != pid || len(pick.OutcomesPicked) == 0 {
									continue
								}
								abbrev := allOutcomeToAbbrev[pick.OutcomesPicked[0].OutcomeID]
								if abbrev == team1 || abbrev == team2 {
									pi := picks[abbrev]
									pi.Count++
									pi.Entries = append(pi.Entries, entry.Name)
									picks[abbrev] = pi
								}
							}
						}
					}
				} else {
					// Current round: aggregate from current prop picks
					for _, entry := range g.Entries {
						for _, pick := range entry.Picks {
							if pick.PropositionID != prop.ID || len(pick.OutcomesPicked) == 0 {
								continue
							}
							abbrev := outcomeToAbbrev[pick.OutcomesPicked[0].OutcomeID]
							if abbrev == team1 || abbrev == team2 {
								pi := picks[abbrev]
								pi.Count++
								pi.Entries = append(pi.Entries, entry.Name)
								picks[abbrev] = pi
							}
						}
					}
				}

				matchups = append(matchups, Matchup{
					ID:           fmt.Sprintf("%s-%s-%d", prop.ID, rd.Key, gameInProp),
					Region:       region,
					DisplayOrder: propIdx*gamesPerProp + gameInProp,
					Team1ID:      team1,
					Team2ID:      team2,
					WinnerID:     winner,
					GameTime:     gameTime,
					Status:       status,
					Picks:        picks,
				})
			}
		}

		result[rd.Key] = matchups
	}

	return result
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

	// Build old outcome → abbrev for Final Four resolution using the same
	// generalized hex mapping as buildAllMatchups.
	period := ch.Propositions[0].ScoringPeriodID
	curProps := make([]ESPNProposition, len(ch.Propositions))
	copy(curProps, ch.Propositions)
	sort.Slice(curProps, func(i, j int) bool {
		return parseHex(curProps[i].ID) < parseHex(curProps[j].ID)
	})
	numProps := len(curProps)

	propTeams := make([]map[int]string, numProps)
	for i, prop := range curProps {
		propTeams[i] = make(map[int]string)
		for _, o := range prop.PossibleOutcomes {
			propTeams[i][o.MatchupPosition] = o.Abbrev
		}
	}

	oldPropSet := make(map[string]bool)
	for _, entry := range g.Entries {
		for _, pick := range entry.Picks {
			if len(pick.OutcomesPicked) > 0 && !currentOutcomeIDs[pick.OutcomesPicked[0].OutcomeID] {
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

	allOutcomeToAbbrev := make(map[string]string)
	for k, v := range outcomeToAbbrev {
		allOutcomeToAbbrev[k] = v
	}

	// Map old props for completed rounds
	idx := 0
	for _, rd := range roundDefs {
		if rd.Period >= period {
			break
		}
		end := idx + rd.TotalGames
		if end > len(oldProps) {
			break
		}
		outcomesPerOldProp := 1 << uint(rd.Period)
		oldPropsPerCurProp := rd.TotalGames / numProps

		for i := idx; i < end; i++ {
			pid := oldProps[i]
			localI := i - idx
			curPropIdx := localI / oldPropsPerCurProp
			localIdx := localI % oldPropsPerCurProp
			base := parseHex(pid)

			for _, entry := range g.Entries {
				for _, pick := range entry.Picks {
					if pick.PropositionID != pid || len(pick.OutcomesPicked) == 0 {
						continue
					}
					oid := pick.OutcomesPicked[0].OutcomeID
					offset := int(parseHex(oid) - base)
					if offset >= 1 && offset <= outcomesPerOldProp {
						pos := localIdx*outcomesPerOldProp + offset
						if curPropIdx < len(propTeams) {
							if team, ok := propTeams[curPropIdx][pos]; ok {
								allOutcomeToAbbrev[oid] = team
							}
						}
					}
				}
			}
		}
		idx = end
	}

	// Map future round old props
	for i := idx; i < len(oldProps); i++ {
		pid := oldProps[i]
		base := parseHex(pid)
		remaining := i - idx
		cumulative := 0
		for _, rd := range roundDefs {
			if rd.Period <= period {
				continue
			}
			if remaining < cumulative+rd.TotalGames {
				localInRound := remaining - cumulative
				propsPerCur := rd.TotalGames / numProps
				if propsPerCur == 0 {
					propsPerCur = 1
				}
				curPropIdx := localInRound / propsPerCur
				localIdx := localInRound % propsPerCur
				outcomesPerProp := 1 << uint(rd.Period)

				for _, entry := range g.Entries {
					for _, pick := range entry.Picks {
						if pick.PropositionID != pid || len(pick.OutcomesPicked) == 0 {
							continue
						}
						oid := pick.OutcomesPicked[0].OutcomeID
						offset := int(parseHex(oid) - base)
						if offset >= 1 && offset <= outcomesPerProp {
							pos := localIdx*outcomesPerProp + offset
							if curPropIdx < len(propTeams) {
								if team, ok := propTeams[curPropIdx][pos]; ok {
									allOutcomeToAbbrev[oid] = team
								}
							}
						}
					}
				}
				break
			}
			cumulative += rd.TotalGames
		}
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
	lbBytes, _ := json.MarshalIndent(lbOut, "", "  ")
	lbJS := append([]byte("const LEADERBOARD = "), lbBytes...)
	lbJS = append(lbJS, ';')
	if err := os.WriteFile("data/leaderboard.js", lbJS, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing leaderboard.js: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Wrote data/leaderboard.js")
}
