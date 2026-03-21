## Server

.PHONY: start stop restart status

start: ## Start the Go server (background)
	@PID=$$(lsof -ti :8000 2>/dev/null); \
	if [ -n "$$PID" ]; then \
		echo "Server already running (PID $$PID)"; \
	else \
		go build -o .server ./backend && \
		nohup ./.server > /dev/null 2>&1 & \
		sleep 0.5; \
		PID=$$(lsof -ti :8000 2>/dev/null); \
		echo "Server started (PID $$PID) at http://localhost:8000"; \
	fi

stop: ## Stop the Go server
	@PID=$$(lsof -ti :8000 2>/dev/null); \
	if [ -n "$$PID" ]; then \
		kill $$PID; \
		echo "Server stopped (PID $$PID)"; \
	else \
		echo "Server not running"; \
	fi

restart: stop start ## Restart the Go server

status: ## Check if server is running
	@PID=$$(lsof -ti :8000 2>/dev/null); \
	if [ -n "$$PID" ]; then \
		echo "Server running (PID $$PID)"; \
	else \
		echo "Server not running"; \
	fi

## Test data

.PHONY: generate-scenarios load-r64 load-r32 load-s16 load-e8 load-ff load-champ load-real test

generate-scenarios: ## Generate all scenario files
	cd backend && GENERATE_SCENARIOS=1 go test -run TestGenerateScenarioFiles -v

load-r64: ## Load R64-complete scenario
	cp backend/testdata/scenarios/R64_1/brackets.json data/brackets.json
	@echo "Loaded R64_1 scenario"

load-r32: ## Load R32-complete scenario
	cp backend/testdata/scenarios/R32_1/brackets.json data/brackets.json
	@echo "Loaded R32_1 scenario"

load-s16: ## Load S16-complete scenario
	cp backend/testdata/scenarios/S16_1/brackets.json data/brackets.json
	@echo "Loaded S16_1 scenario"

load-e8: ## Load E8-complete scenario
	cp backend/testdata/scenarios/E8_1/brackets.json data/brackets.json
	@echo "Loaded E8_1 scenario"

load-ff: ## Load FF-complete scenario
	cp backend/testdata/scenarios/FF_1/brackets.json data/brackets.json
	@echo "Loaded FF_1 scenario"

load-champ: ## Load Championship-complete scenario
	cp backend/testdata/scenarios/Champ_1/brackets.json data/brackets.json
	@echo "Loaded Champ_1 scenario"

load-real: ## Restore real ESPN data
	go run ./backend --fetch-only
	@echo "Restored real ESPN data"

test: ## Run all Go tests
	go test ./backend/... -v

test-e2e: ## Run Playwright smoke tests
	cd tests && npx playwright test

help: ## Show this help
	@grep -E '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

.DEFAULT_GOAL := help
