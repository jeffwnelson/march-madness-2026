package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

const (
	challengeURL    = "https://gambit-api.fantasy.espn.com/apis/v1/challenges/tournament-challenge-bracket-2026"
	groupURL        = "https://gambit-api.fantasy.espn.com/apis/v1/challenges/tournament-challenge-bracket-2026/groups/af223df6-96d0-46e7-b00d-1b590dc67888?view=entries&limit=50"
	leaderboardPath = "data/leaderboard.json"
	bracketPicksPath = "data/bracket-picks.json"
	snapshotDir     = "data/snapshots"
)

func fetchESPNData() (*LeaderboardData, *BracketPicksData, error) {
	challengeResp, err := http.Get(challengeURL)
	if err != nil {
		return nil, nil, fmt.Errorf("fetching challenge data: %w", err)
	}
	defer challengeResp.Body.Close()
	challengeBytes, err := io.ReadAll(challengeResp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("reading challenge response: %w", err)
	}

	groupResp, err := http.Get(groupURL)
	if err != nil {
		return nil, nil, fmt.Errorf("fetching group data: %w", err)
	}
	defer groupResp.Body.Close()
	groupBytes, err := io.ReadAll(groupResp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("reading group response: %w", err)
	}

	// Save raw snapshots
	if err := saveSnapshot(snapshotDir, challengeBytes, groupBytes); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to save snapshot: %v\n", err)
	}

	challenge, err := parseChallengeData(challengeBytes)
	if err != nil {
		return nil, nil, fmt.Errorf("parsing challenge data: %w", err)
	}

	group, err := parseGroupData(groupBytes)
	if err != nil {
		return nil, nil, fmt.Errorf("parsing group data: %w", err)
	}

	// Load existing data (ignore errors — files may not exist yet)
	existingLB, _ := loadJSON[LeaderboardData](leaderboardPath)
	existingBP, _ := loadJSON[BracketPicksData](bracketPicksPath)

	lb, bp := buildOutputs(challenge, group, existingLB, existingBP)
	return lb, bp, nil
}

func main() {
	fetchOnly := false
	for _, arg := range os.Args[1:] {
		if arg == "--fetch-only" {
			fetchOnly = true
		}
	}

	// --fetch-only: always fetch fresh data, save, exit (for GitHub Actions)
	if fetchOnly {
		fmt.Println("Fetching fresh data from ESPN...")
		lb, bp, err := fetchESPNData()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error fetching ESPN data: %v\n", err)
			os.Exit(1)
		}
		if err := saveJSON(leaderboardPath, lb); err != nil {
			fmt.Fprintf(os.Stderr, "Error saving leaderboard: %v\n", err)
			os.Exit(1)
		}
		if err := saveJSON(bracketPicksPath, bp); err != nil {
			fmt.Fprintf(os.Stderr, "Error saving bracket picks: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Data fetched and saved. Exiting.")
		return
	}

	// Normal server mode: serve static files
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			http.ServeFile(w, r, "index.html")
			return
		}
		// Serve static files (data/*.json, css/*, js/*, etc.)
		http.ServeFile(w, r, r.URL.Path[1:])
	})

	http.HandleFunc("/api/refresh", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		lb, bp, err := fetchESPNData()
		if err != nil {
			http.Error(w, fmt.Sprintf("error fetching data: %v", err), http.StatusInternalServerError)
			return
		}
		if err := saveJSON(leaderboardPath, lb); err != nil {
			fmt.Fprintf(os.Stderr, "Error saving leaderboard: %v\n", err)
		}
		if err := saveJSON(bracketPicksPath, bp); err != nil {
			fmt.Fprintf(os.Stderr, "Error saving bracket picks: %v\n", err)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	addr := ":8000"
	fmt.Printf("Server running at http://localhost%s\n", addr)
	http.ListenAndServe(addr, nil)
}

// ESPN raw API types

type espnChallenge struct {
	Propositions []espnProposition `json:"propositions"`
}

type espnProposition struct {
	ID               string        `json:"id"`
	Name             string        `json:"name"`
	Status           string        `json:"status"`
	Date             *int64        `json:"date"`
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
	Status             string        `json:"status"`
	Mappings           []espnMapping `json:"mappings"`
}

type espnMapping struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

type espnGroupSettings struct {
	Name string `json:"name"`
}

type espnGroup struct {
	Entries       []espnEntry       `json:"entries"`
	Size          int               `json:"size"`
	GroupID       string            `json:"groupId"`
	GroupSettings espnGroupSettings `json:"groupSettings"`
}

type espnEntry struct {
	ID               string               `json:"id"`
	Name             string               `json:"name"`
	Member           espnMember           `json:"member"`
	Picks            []espnPick           `json:"picks"`
	Score            espnScore            `json:"score"`
	FinalPick        espnPick             `json:"finalPick"`
	TiebreakAnswers  []espnTiebreakAnswer `json:"tiebreakAnswers"`
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
	OverallScore            int     `json:"overallScore"`
	PossiblePointsMax       int     `json:"possiblePointsMax"`
	PossiblePointsRemaining int     `json:"possiblePointsRemaining"`
	PointsLost              int     `json:"pointsLost"`
	Rank                    int     `json:"rank"`
	Eliminated              bool    `json:"eliminated"`
	Percentile              float64 `json:"percentile"`
}

type espnTiebreakAnswer struct {
	Answer float64 `json:"answer"`
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
