package main

// LeaderboardData is the output structure for leaderboard.json
type LeaderboardData struct {
	LastUpdated string                `json:"lastUpdated"`
	GroupName   string                `json:"groupName"`
	Teams       map[string]TeamInfo   `json:"teams"`
	Brackets    []LeaderboardBracket  `json:"brackets"`
}

// TeamInfo holds metadata about a tournament team
type TeamInfo struct {
	Name   string `json:"name"`
	Abbrev string `json:"abbrev"`
	Seed   int    `json:"seed"`
	Region int    `json:"region"`
	Logo   string `json:"logo"`
}

// LeaderboardBracket holds scoring and pick summary for a single bracket entry
type LeaderboardBracket struct {
	EntryName   string   `json:"entryName"`
	Member      string   `json:"member"`
	Score       int      `json:"score"`
	MaxPossible int      `json:"maxPossible"`
	Rank        int      `json:"rank"`
	Percentile  float64  `json:"percentile"`
	Eliminated  bool     `json:"eliminated"`
	Tiebreaker  *float64 `json:"tiebreaker"`
	Champion    string   `json:"champion"`
	FinalFour   []string `json:"finalFour"`
}

// BracketPicksData is the output structure for bracket-picks.json
type BracketPicksData struct {
	LastUpdated string           `json:"lastUpdated"`
	Teams       map[string]TeamInfo `json:"teams"`
	Rounds      map[string]Round `json:"rounds"`
}

// Round holds all matchup data for a single tournament round
type Round struct {
	Status   string        `json:"status"` // "complete", "in_progress", "future"
	Matchups []MatchupData `json:"matchups"`
}

// MatchupData holds game details and pick breakdowns for a single matchup
type MatchupData struct {
	ID           string              `json:"id"`
	Region       int                 `json:"region"`
	DisplayOrder int                 `json:"displayOrder"`
	Team1        string              `json:"team1"`
	Team2        string              `json:"team2"`
	Winner       string              `json:"winner,omitempty"`
	Status       string              `json:"status"`
	GameTime     *int64              `json:"gameTime,omitempty"`
	Picks        map[string]PickData `json:"picks"`
}

// PickData holds pick counts and entry names for a given team in a matchup
type PickData struct {
	Count   int      `json:"count"`
	Entries []string `json:"entries"`
}

var roundKeyOrder = []string{"r64", "r32", "sweet16", "elite8", "finalFour", "championship"}

func periodToRoundKey(period int) string {
	if period >= 1 && period <= len(roundKeyOrder) {
		return roundKeyOrder[period-1]
	}
	return ""
}
