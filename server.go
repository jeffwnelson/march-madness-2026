package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
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
	fetchOnly := false
	for _, arg := range os.Args[1:] {
		if arg == "--fetch-only" {
			fetchOnly = true
		}
	}

	// --fetch-only: always fetch fresh data, save, exit (for GitHub Actions)
	if fetchOnly {
		fmt.Println("Fetching fresh data from ESPN...")
		data, err := fetchESPNData()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error fetching ESPN data: %v\n", err)
			os.Exit(1)
		}
		if err := saveCache(cachePath, data); err != nil {
			fmt.Fprintf(os.Stderr, "Error saving cache: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Data fetched and cached. Exiting.")
		return
	}

	// Normal mode: load cache or fetch
	var mu sync.Mutex
	var currentData *BracketData

	cached, err := loadCache(cachePath)
	if err != nil {
		fmt.Println("No cached data found, fetching from ESPN...")
		fetched, fetchErr := fetchESPNData()
		if fetchErr != nil {
			fmt.Fprintf(os.Stderr, "Error fetching ESPN data: %v\n", fetchErr)
			os.Exit(1)
		}
		if saveErr := saveCache(cachePath, fetched); saveErr != nil {
			fmt.Fprintf(os.Stderr, "Error saving cache: %v\n", saveErr)
		}
		currentData = fetched
	} else {
		currentData = cached
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		http.ServeFile(w, r, "index.html")
	})

	http.HandleFunc("/api/brackets", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		mu.Lock()
		d := currentData
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(d)
	})

	http.HandleFunc("/api/refresh", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		fresh, fetchErr := fetchESPNData()
		if fetchErr != nil {
			http.Error(w, fmt.Sprintf("error fetching data: %v", fetchErr), http.StatusInternalServerError)
			return
		}
		if saveErr := saveCache(cachePath, fresh); saveErr != nil {
			fmt.Fprintf(os.Stderr, "Error saving cache: %v\n", saveErr)
		}
		mu.Lock()
		currentData = fresh
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(fresh)
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
