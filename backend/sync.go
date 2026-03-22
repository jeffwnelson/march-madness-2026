package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// rs is a region+seed pair used as a canonical team identifier.
type rs struct{ r, s int }

// loadJSON loads and unmarshals a JSON file into the given type.
func loadJSON[T any](path string) (*T, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var v T
	if err := json.Unmarshal(b, &v); err != nil {
		return nil, err
	}
	return &v, nil
}

// saveJSON marshals v as indented JSON and writes it to path.
func saveJSON(path string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0644)
}

// saveSnapshot saves timestamped raw JSON files into dir.
func saveSnapshot(dir string, challengeData, groupData []byte) error {
	ts := time.Now().UTC().Format("20060102T150405Z")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, fmt.Sprintf("challenge-%s.json", ts)), challengeData, 0644); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, fmt.Sprintf("group-%s.json", ts)), groupData, 0644)
}

// teamID returns a canonical team ID string from region and seed.
func teamID(region, seed int) string {
	return fmt.Sprintf("r%d-s%d", region, seed)
}

// buildOutputs transforms ESPN raw data into LeaderboardData and BracketPicksData.
func buildOutputs(ch *espnChallenge, g *espnGroup, existingLB *LeaderboardData, existingBP *BracketPicksData) (*LeaderboardData, *BracketPicksData) {
	now := time.Now().UTC().Format(time.RFC3339)

	// Determine current scoring period (minimum across all propositions).
	currentPeriod := 0
	if len(ch.Propositions) > 0 {
		currentPeriod = ch.Propositions[0].ScoringPeriodID
		for _, p := range ch.Propositions[1:] {
			if p.ScoringPeriodID < currentPeriod {
				currentPeriod = p.ScoringPeriodID
			}
		}
	}

	// Build teams from proposition outcomes.
	// Each outcome has a unique region+seed.
	teams := make(map[string]TeamInfo) // keyed by canonical team ID
	outcomeToTeam := make(map[string]string) // ESPN outcome ID → canonical team ID
	rsToTeam := make(map[rs]string)

	for _, prop := range ch.Propositions {
		for _, o := range prop.PossibleOutcomes {
			tid := teamID(o.RegionID, o.RegionSeed)
			if _, exists := teams[tid]; !exists {
				logo := ""
				for _, m := range o.Mappings {
					if m.Type == "IMAGE_PRIMARY" {
						logo = m.Value
						break
					}
				}
				teams[tid] = TeamInfo{
					Name:   o.Name,
					Abbrev: o.Abbrev,
					Seed:   o.RegionSeed,
					Region: o.RegionID,
					Logo:   logo,
				}
			}
			outcomeToTeam[o.ID] = tid
			rsToTeam[rs{o.RegionID, o.RegionSeed}] = tid
		}
	}

	// Build propID set for current challenge propositions.
	propSet := make(map[string]bool)
	propByID := make(map[string]*espnProposition)
	for i := range ch.Propositions {
		p := &ch.Propositions[i]
		propSet[p.ID] = true
		propByID[p.ID] = p
	}

	// Champion resolution: map finalPick outcome IDs to canonical team IDs.
	champMap := resolveChampions(ch, g, propSet, outcomeToTeam, teams, rsToTeam)

	// Build old outcome ID mapping if period >= 2.
	oldOutcomeToTeam := make(map[string]string) // old ESPN outcome ID → canonical team ID
	if currentPeriod >= 2 {
		oldOutcomeToTeam = buildOldOutcomeMapping(ch, g, propSet, outcomeToTeam)
	}

	// Combined outcome resolver: tries current outcomes first, then old mapping.
	resolveOutcome := func(outcomeID string) string {
		if tid, ok := outcomeToTeam[outcomeID]; ok {
			return tid
		}
		if tid, ok := oldOutcomeToTeam[outcomeID]; ok {
			return tid
		}
		return ""
	}

	// Build matchups for current round.
	currentRoundKey := periodToRoundKey(currentPeriod)
	matchups := buildCurrentMatchups(ch, g, outcomeToTeam, resolveOutcome, currentPeriod)

	// Build R64 reconstructed matchups if period >= 2.
	var r64Matchups []MatchupData
	if currentPeriod >= 2 {
		r64Matchups = reconstructR64Matchups(ch, g, outcomeToTeam, oldOutcomeToTeam, resolveOutcome)
	}

	// Build pick aggregation for current round.
	aggregateCurrentPicks(matchups, g, propSet, resolveOutcome, currentPeriod)

	// Build pick aggregation for R64 if period >= 2.
	if currentPeriod >= 2 {
		aggregateR64Picks(r64Matchups, g, propSet, oldOutcomeToTeam)
	}

	// If period == 1, R64 matchups ARE the current matchups.
	if currentPeriod == 1 {
		r64Matchups = matchups
	}

	// Build rounds map.
	rounds := make(map[string]Round)

	// Preserve existing rounds.
	if existingBP != nil {
		for k, v := range existingBP.Rounds {
			rounds[k] = v
		}
	}

	// Add/update R64 round.
	if len(r64Matchups) > 0 {
		rounds["r64"] = Round{
			Status:   deriveRoundStatus(r64Matchups),
			Matchups: r64Matchups,
		}
	}

	// Add/update current round if not R64.
	if currentPeriod >= 2 && currentRoundKey != "" && len(matchups) > 0 {
		rounds[currentRoundKey] = Round{
			Status:   deriveRoundStatus(matchups),
			Matchups: matchups,
		}
	}

	// Ensure all 6 round keys exist.
	for _, rk := range roundKeyOrder {
		if _, ok := rounds[rk]; !ok {
			rounds[rk] = Round{Status: "future", Matchups: []MatchupData{}}
		}
	}

	// Build leaderboard brackets.
	brackets := buildLeaderboardBrackets(g, propSet, resolveOutcome, champMap)

	lb := &LeaderboardData{
		LastUpdated: now,
		GroupName:   g.GroupSettings.Name,
		Teams:       teams,
		Brackets:    brackets,
	}

	bp := &BracketPicksData{
		LastUpdated: now,
		Teams:       teams,
		Rounds:      rounds,
	}

	return lb, bp
}

// resolveChampions maps finalPick outcome IDs to canonical team IDs.
// Uses cross-entry correlation, iterative resolution, and hex offset fallback.
func resolveChampions(ch *espnChallenge, g *espnGroup, propSet map[string]bool, outcomeToTeam map[string]string, teams map[string]TeamInfo, rsToTeam map[rs]string) map[string]string {
	// Build propPeriod for current props.
	propPeriod := make(map[string]int)
	for _, p := range ch.Propositions {
		propPeriod[p.ID] = p.ScoringPeriodID
	}

	type champCandidate struct {
		teams    map[string]bool
		resolved string
	}
	champCandidates := make(map[string]*champCandidate)

	for _, entry := range g.Entries {
		if len(entry.FinalPick.OutcomesPicked) == 0 {
			continue
		}
		fpOutcome := entry.FinalPick.OutcomesPicked[0].OutcomeID

		// Collect canonical team IDs for picks with periodReached >= 6.
		finalists := make(map[string]bool)
		for _, pick := range entry.Picks {
			if !propSet[pick.PropositionID] {
				continue
			}
			if propPeriod[pick.PropositionID] == ch.Propositions[0].ScoringPeriodID && pick.PeriodReached >= 6 && len(pick.OutcomesPicked) > 0 {
				if tid, ok := outcomeToTeam[pick.OutcomesPicked[0].OutcomeID]; ok {
					finalists[tid] = true
				}
			}
		}

		if existing, ok := champCandidates[fpOutcome]; ok {
			for tid := range existing.teams {
				if !finalists[tid] {
					delete(existing.teams, tid)
				}
			}
		} else {
			champCandidates[fpOutcome] = &champCandidate{teams: finalists}
		}
	}

	// Iterative resolution.
	usedTeams := make(map[string]bool)
	for {
		progress := false
		for _, c := range champCandidates {
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

	// Hex offset fallback.
	bracketOrder := [16]int{1, 16, 8, 9, 5, 12, 4, 13, 6, 11, 3, 14, 7, 10, 2, 15}
	parseFirstSeg := func(id string) (int64, bool) {
		seg := strings.SplitN(id, "-", 2)[0]
		val, err := strconv.ParseInt(seg, 16, 64)
		return val, err == nil
	}

	var champBase int64
	var hasBase bool
	for fpOutcome, c := range champCandidates {
		if c.resolved == "" {
			continue
		}
		fpVal, ok := parseFirstSeg(fpOutcome)
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
		for fpOutcome, c := range champCandidates {
			if c.resolved != "" {
				continue
			}
			fpVal, ok := parseFirstSeg(fpOutcome)
			if !ok {
				continue
			}
			offset := int(fpVal - champBase)
			region := offset/16 + 1
			pos := offset % 16
			if pos >= 0 && pos < 16 && region >= 1 && region <= 4 {
				seed := bracketOrder[pos]
				if tid, ok := rsToTeam[rs{region, seed}]; ok {
					c.resolved = tid
				}
			}
		}
	}

	// Build result map: finalPick outcome ID → canonical team ID.
	result := make(map[string]string)
	for fpOutcome, c := range champCandidates {
		if c.resolved != "" {
			result[fpOutcome] = c.resolved
		}
	}
	return result
}

// buildOldOutcomeMapping maps old R64 outcome IDs (from entry picks) to canonical team IDs.
func buildOldOutcomeMapping(ch *espnChallenge, g *espnGroup, propSet map[string]bool, outcomeToTeam map[string]string) map[string]string {
	// Collect old R64 proposition IDs from entries.
	// These are props NOT in current challenge, NOT finalPick props, where min periodReached across entries >= 2.
	finalPickProps := make(map[string]bool)
	for _, entry := range g.Entries {
		finalPickProps[entry.FinalPick.PropositionID] = true
	}

	// Collect all non-challenge, non-finalPick prop IDs and track min periodReached.
	propMinPeriod := make(map[string]int)
	propOutcomes := make(map[string]map[string]bool) // propID → set of outcome IDs seen
	for _, entry := range g.Entries {
		for _, pick := range entry.Picks {
			if propSet[pick.PropositionID] || finalPickProps[pick.PropositionID] {
				continue
			}
			if min, ok := propMinPeriod[pick.PropositionID]; !ok || pick.PeriodReached < min {
				propMinPeriod[pick.PropositionID] = pick.PeriodReached
			}
			if _, ok := propOutcomes[pick.PropositionID]; !ok {
				propOutcomes[pick.PropositionID] = make(map[string]bool)
			}
			if len(pick.OutcomesPicked) > 0 {
				propOutcomes[pick.PropositionID][pick.OutcomesPicked[0].OutcomeID] = true
			}
		}
	}

	// Filter to old R64 props: minPeriodReached == 2 (these are R64 picks that advance to R32).
	var oldR64Props []string
	for pid, minPR := range propMinPeriod {
		if minPR == 2 {
			oldR64Props = append(oldR64Props, pid)
		}
	}
	sort.Strings(oldR64Props)

	// Sort R32 (current) props by hex value.
	var r32Props []string
	for _, p := range ch.Propositions {
		r32Props = append(r32Props, p.ID)
	}
	sort.Strings(r32Props)

	// Pair every 2 consecutive old R64 props to 1 R32 prop.
	result := make(map[string]string)
	for i := 0; i+1 < len(oldR64Props) && i/2 < len(r32Props); i += 2 {
		r32PropID := r32Props[i/2]
		r32Prop := ch.Propositions[0] // placeholder
		for _, p := range ch.Propositions {
			if p.ID == r32PropID {
				r32Prop = p
				break
			}
		}

		// Get outcomes at positions 1,2 and 3,4.
		pos := make(map[int]*espnOutcome)
		for j := range r32Prop.PossibleOutcomes {
			o := &r32Prop.PossibleOutcomes[j]
			pos[o.MatchupPosition] = o
		}

		// First old prop maps to positions 1,2; second to positions 3,4.
		oldPropA := oldR64Props[i]
		oldPropB := oldR64Props[i+1]

		// Map old outcomes to R32 outcomes using entry correlation.
		// For each old prop, find entries that picked a specific outcome AND also
		// picked a known team for the R32 prop. This tells us which old outcome = which team.
		mapOldPropOutcomes(result, oldPropA, r32PropID, []int{1, 2}, pos, g, propSet, outcomeToTeam)
		mapOldPropOutcomes(result, oldPropB, r32PropID, []int{3, 4}, pos, g, propSet, outcomeToTeam)
	}

	return result
}

// mapOldPropOutcomes maps old R64 outcome IDs to canonical team IDs by correlating
// entries' old R64 picks with their R32 picks for the same bracket slot.
// positions specifies which R32 matchup positions (e.g., [1,2] or [3,4]) this old prop feeds into.
func mapOldPropOutcomes(result map[string]string, oldPropID, r32PropID string, positions []int, pos map[int]*espnOutcome, g *espnGroup, propSet map[string]bool, outcomeToTeam map[string]string) {
	// For each entry, find their pick for this old R64 prop and their pick for the R32 prop.
	// If the R32 pick matches one of the positions we're interested in, we know the old outcome = that team.
	for _, entry := range g.Entries {
		var oldOutcomeID string
		var r32OutcomeID string

		for _, pick := range entry.Picks {
			if pick.PropositionID == oldPropID && len(pick.OutcomesPicked) > 0 {
				oldOutcomeID = pick.OutcomesPicked[0].OutcomeID
			}
			if pick.PropositionID == r32PropID && len(pick.OutcomesPicked) > 0 {
				r32OutcomeID = pick.OutcomesPicked[0].OutcomeID
			}
		}

		if oldOutcomeID == "" || r32OutcomeID == "" {
			continue
		}
		if _, done := result[oldOutcomeID]; done {
			continue
		}

		// Check if the R32 pick corresponds to one of our positions
		r32Team := outcomeToTeam[r32OutcomeID]
		for _, p := range positions {
			if pos[p] != nil && outcomeToTeam[pos[p].ID] == r32Team {
				result[oldOutcomeID] = r32Team
				break
			}
		}
	}

	// Fallback: if we mapped one outcome but not the other, infer the second.
	// Each old prop has exactly 2 outcomes mapping to 2 positions.
	posTeams := make(map[string]string) // canonical team ID → old outcome ID (reverse of result)
	var unmappedOld []string
	for oid := range propOutcomesForProp(g, oldPropID) {
		if _, ok := result[oid]; ok {
			posTeams[result[oid]] = oid
		} else {
			unmappedOld = append(unmappedOld, oid)
		}
	}
	if len(unmappedOld) == 1 {
		// Find which position team is unmapped
		for _, p := range positions {
			if pos[p] == nil {
				continue
			}
			tid := outcomeToTeam[pos[p].ID]
			if _, mapped := posTeams[tid]; !mapped {
				result[unmappedOld[0]] = tid
				break
			}
		}
	}
}

// propOutcomesForProp collects unique outcome IDs for a proposition from all entries.
func propOutcomesForProp(g *espnGroup, propID string) map[string]bool {
	seen := make(map[string]bool)
	for _, entry := range g.Entries {
		for _, pick := range entry.Picks {
			if pick.PropositionID == propID && len(pick.OutcomesPicked) > 0 {
				seen[pick.OutcomesPicked[0].OutcomeID] = true
			}
		}
	}
	return seen
}

func sortedKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// buildCurrentMatchups builds matchup data from current propositions.
func buildCurrentMatchups(ch *espnChallenge, g *espnGroup, outcomeToTeam map[string]string, resolveOutcome func(string) string, currentPeriod int) []MatchupData {
	matchups := make([]MatchupData, 0, len(ch.Propositions))

	for _, prop := range ch.Propositions {
		m := MatchupData{
			ID:           prop.ID,
			DisplayOrder: prop.DisplayOrder,
			Status:       prop.Status,
			Picks:        make(map[string]PickData),
		}

		if prop.Date != nil {
			m.GameTime = prop.Date
		}

		// Determine region from first outcome.
		if len(prop.PossibleOutcomes) > 0 {
			m.Region = prop.PossibleOutcomes[0].RegionID
		}

		if currentPeriod == 1 {
			// R64: 2 outcomes → team1/team2 by matchupPosition.
			for _, o := range prop.PossibleOutcomes {
				tid := outcomeToTeam[o.ID]
				if o.MatchupPosition == 1 {
					m.Team1 = tid
				} else {
					m.Team2 = tid
				}
			}
		} else {
			// R32+: determine actual contestants from entry picks.
			m.Team1, m.Team2 = determineR32Contestants(prop, outcomeToTeam)
		}

		// Determine winner from actualOutcomeIds.
		if len(prop.ActualOutcomeIDs) > 0 {
			m.Winner = outcomeToTeam[prop.ActualOutcomeIDs[0]]
		}

		matchups = append(matchups, m)
	}

	return matchups
}

// determineR32Contestants figures out which 2 of 4 outcomes are the actual R32 contestants
// using actualOutcomeIds from the proposition.
func determineR32Contestants(prop espnProposition, outcomeToTeam map[string]string) (string, string) {
	actualSet := make(map[string]bool)
	for _, oid := range prop.ActualOutcomeIDs {
		actualSet[oid] = true
	}
	var t1, t2 string
	for _, o := range prop.PossibleOutcomes {
		tid := outcomeToTeam[o.ID]
		if actualSet[o.ID] {
			if o.MatchupPosition <= 2 {
				t1 = tid
			} else {
				t2 = tid
			}
		}
	}
	return t1, t2
}

// reconstructR64Matchups builds synthetic R64 matchups from R32 propositions.
func reconstructR64Matchups(ch *espnChallenge, g *espnGroup, outcomeToTeam map[string]string, oldOutcomeToTeam map[string]string, resolveOutcome func(string) string) []MatchupData {
	var matchups []MatchupData

	for _, prop := range ch.Propositions {
		// Each R32 prop's 4 outcomes at positions 1,2,3,4 represent two R64 games.
		pos := make(map[int]string) // matchupPosition → canonical team ID
		regionID := 0
		for _, o := range prop.PossibleOutcomes {
			pos[o.MatchupPosition] = outcomeToTeam[o.ID]
			if regionID == 0 {
				regionID = o.RegionID
			}
		}

		// R64 game A: pos 1 vs pos 2.
		winnerA := determineR64Winner(pos[1], pos[2], prop, outcomeToTeam)
		mA := MatchupData{
			ID:           prop.ID + "-r64a",
			Region:       regionID,
			DisplayOrder: prop.DisplayOrder*2 - 1,
			Team1:        pos[1],
			Team2:        pos[2],
			Winner:       winnerA,
			Status:       "COMPLETE",
			Picks:        make(map[string]PickData),
		}

		// R64 game B: pos 3 vs pos 4.
		winnerB := determineR64Winner(pos[3], pos[4], prop, outcomeToTeam)
		mB := MatchupData{
			ID:           prop.ID + "-r64b",
			Region:       regionID,
			DisplayOrder: prop.DisplayOrder * 2,
			Team1:        pos[3],
			Team2:        pos[4],
			Winner:       winnerB,
			Status:       "COMPLETE",
			Picks:        make(map[string]PickData),
		}

		matchups = append(matchups, mA, mB)
	}

	return matchups
}

// determineR64Winner determines which of two teams won the R64 game
// by checking the R32 proposition's actualOutcomeIds. These list the outcomes
// (teams) that advanced — i.e., the R64 winners who play in R32.
func determineR64Winner(team1, team2 string, r32Prop espnProposition, outcomeToTeam map[string]string) string {
	actualSet := make(map[string]bool)
	for _, oid := range r32Prop.ActualOutcomeIDs {
		actualSet[oid] = true
	}
	for _, o := range r32Prop.PossibleOutcomes {
		tid := outcomeToTeam[o.ID]
		if (tid == team1 || tid == team2) && actualSet[o.ID] {
			return tid
		}
	}
	return ""
}

// aggregateCurrentPicks counts picks per team per matchup for the current round.
func aggregateCurrentPicks(matchups []MatchupData, g *espnGroup, propSet map[string]bool, resolveOutcome func(string) string, currentPeriod int) {
	matchupByID := make(map[string]int) // matchup ID → index in matchups
	for i, m := range matchups {
		matchupByID[m.ID] = i
	}

	for _, entry := range g.Entries {
		for _, pick := range entry.Picks {
			if !propSet[pick.PropositionID] {
				continue
			}
			idx, ok := matchupByID[pick.PropositionID]
			if !ok || len(pick.OutcomesPicked) == 0 {
				continue
			}
			// For R64, periodReached must be >= 1 (which it always is).
			// For R32, picks advance from R64 — entries that pick a team to win
			// R32 are those with periodReached >= currentPeriod+1 for that prop.
			// But actually, the current round picks are entries' picks with
			// propositionID matching the current prop.
			outcomeID := pick.OutcomesPicked[0].OutcomeID
			tid := resolveOutcome(outcomeID)
			if tid == "" {
				continue
			}
			pd := matchups[idx].Picks[tid]
			pd.Count++
			pd.Entries = append(pd.Entries, entry.Name)
			matchups[idx].Picks[tid] = pd
		}
	}
}

// aggregateR64Picks counts R64 picks from old outcome IDs when period >= 2.
func aggregateR64Picks(r64Matchups []MatchupData, g *espnGroup, propSet map[string]bool, oldOutcomeToTeam map[string]string) {
	// Build a lookup: canonical team ID → R64 matchup indices where team plays.
	teamToMatchups := make(map[string][]int)
	for i, m := range r64Matchups {
		teamToMatchups[m.Team1] = append(teamToMatchups[m.Team1], i)
		teamToMatchups[m.Team2] = append(teamToMatchups[m.Team2], i)
	}

	for _, entry := range g.Entries {
		for _, pick := range entry.Picks {
			if propSet[pick.PropositionID] {
				continue // skip current round picks
			}
			if pick.PeriodReached < 2 {
				continue
			}
			if len(pick.OutcomesPicked) == 0 {
				continue
			}
			outcomeID := pick.OutcomesPicked[0].OutcomeID
			tid, ok := oldOutcomeToTeam[outcomeID]
			if !ok {
				continue
			}
			// Find which R64 matchup this team is in and record the pick.
			for _, idx := range teamToMatchups[tid] {
				pd := r64Matchups[idx].Picks[tid]
				pd.Count++
				pd.Entries = append(pd.Entries, entry.Name)
				r64Matchups[idx].Picks[tid] = pd
				break // a team is only in one R64 matchup
			}
		}
	}
}

// deriveRoundStatus determines round status from matchup statuses.
func deriveRoundStatus(matchups []MatchupData) string {
	allComplete := true
	anyActive := false
	for _, m := range matchups {
		if m.Status == "PLAYING" || m.Status == "COMPLETE" {
			anyActive = true
		}
		if m.Status != "COMPLETE" {
			allComplete = false
		}
	}
	if allComplete && len(matchups) > 0 {
		return "complete"
	}
	if anyActive {
		return "in_progress"
	}
	return "future"
}

// buildLeaderboardBrackets creates leaderboard entries from ESPN group data.
func buildLeaderboardBrackets(g *espnGroup, propSet map[string]bool, resolveOutcome func(string) string, champMap map[string]string) []LeaderboardBracket {
	brackets := make([]LeaderboardBracket, 0, len(g.Entries))

	for _, entry := range g.Entries {
		var tiebreaker *float64
		if len(entry.TiebreakAnswers) > 0 {
			t := entry.TiebreakAnswers[0].Answer
			tiebreaker = &t
		}

		// Resolve champion.
		var champion string
		if len(entry.FinalPick.OutcomesPicked) > 0 {
			fpOutcome := entry.FinalPick.OutcomesPicked[0].OutcomeID
			if tid, ok := champMap[fpOutcome]; ok {
				champion = tid
			}
		}

		// Collect Final Four picks: periodReached >= 5.
		var finalFour []string
		for _, pick := range entry.Picks {
			if pick.PeriodReached >= 5 && len(pick.OutcomesPicked) > 0 {
				tid := resolveOutcome(pick.OutcomesPicked[0].OutcomeID)
				if tid != "" {
					finalFour = append(finalFour, tid)
				}
			}
		}
		if finalFour == nil {
			finalFour = []string{}
		}

		b := LeaderboardBracket{
			EntryName:   entry.Name,
			Member:      entry.Member.DisplayName,
			Score:       entry.Score.OverallScore,
			MaxPossible: entry.Score.PossiblePointsMax,
			Rank:        entry.Score.Rank,
			Percentile:  entry.Score.Percentile,
			Eliminated:  entry.Score.Eliminated,
			Tiebreaker:  tiebreaker,
			Champion:    champion,
			FinalFour:   finalFour,
		}
		brackets = append(brackets, b)
	}

	return brackets
}
