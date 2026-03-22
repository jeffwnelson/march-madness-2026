## ESPN Data

.PHONY: fetch generate help

fetch: ## Download fresh ESPN data
	@mkdir -p data/espn
	@curl -s "https://gambit-api.fantasy.espn.com/apis/v1/challenges/tournament-challenge-bracket-2026" | python3 -m json.tool > data/espn/challenge.json
	@curl -s "https://gambit-api.fantasy.espn.com/apis/v1/challenges/tournament-challenge-bracket-2026/groups/af223df6-96d0-46e7-b00d-1b590dc67888?view=entries&limit=50" | python3 -m json.tool > data/espn/group.json
	@echo "Downloaded ESPN data"

generate: ## Generate data.js and leaderboard.js from ESPN data
	go run ./backend/
	@echo "Generated data files"

update: fetch generate ## Fetch ESPN data and regenerate

help: ## Show this help
	@grep -E '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

.DEFAULT_GOAL := help
