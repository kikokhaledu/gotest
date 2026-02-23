.PHONY: help ensure-env install install-go install-node install-react run-go run-node run-react test-go test-node test-react test-all test-go-cover vet-go prepush reset up-db down-db db-logs psql docker-build up down logs logs-follow logs-all logs-errors ps

IS_WINDOWS := $(filter Windows_NT,$(OS))

ifneq ($(wildcard .env),)
ENV_FILE := .env
else ifneq ($(wildcard .env.example),)
ENV_FILE := .env.example
else
$(error ERROR: neither .env nor .env.example exists)
endif

COMPOSE := docker compose --env-file $(ENV_FILE)

help:
	@echo Available commands:
	@echo   make help          Show available commands and descriptions
	@echo   make ensure-env    Show which env file compose will use (.env preferred, else .env.example)
	@echo   make install       Install dependencies for Go, Node backend, and React frontend
	@echo   make install-go    Run go mod tidy in go-backend
	@echo   make install-node  Run npm install in node-backend
	@echo   make install-react Run npm install in react-frontend
	@echo   make run-go        Run Go backend locally
	@echo   make run-node      Run Node backend locally
	@echo   make run-react     Run React frontend locally (Vite dev server)
	@echo   make test-go       Run Go tests (verbose)
	@echo   make test-node     Run Node backend tests
	@echo   make test-react    Run React frontend tests
	@echo   make test-all      Run tests for Go, Node backend, and React frontend
	@echo   make test-go-cover Run Go tests with coverage
	@echo   make vet-go        Run go vet checks
	@echo   make prepush       Run full pre-push quality checks across all components
	@echo   make reset         Reset local stack state (stop containers) and re-install dependencies
	@echo   make up-db         Start only PostgreSQL in docker compose
	@echo   make down-db       Stop only PostgreSQL container
	@echo   make db-logs       Follow PostgreSQL logs
	@echo   make psql          Open psql shell in PostgreSQL container
	@echo   make docker-build  Build all Docker images via docker compose
	@echo   make up            Start full stack with docker compose
	@echo   make down          Stop and remove docker compose stack
	@echo   make logs          Show recent docker compose logs and exit
	@echo   make logs-follow   Follow full docker compose logs
	@echo   make logs-errors   Follow only error-like lines from docker compose logs
	@echo   make logs-all      Follow full raw docker compose logs
	@echo   make ps            Show docker compose service status

install: install-go install-node install-react

ensure-env:
	@echo Using env file: $(ENV_FILE)

install-go:
	cd go-backend && go mod tidy

install-node:
	cd node-backend && npm install

install-react:
	cd react-frontend && npm install

run-go:
	cd go-backend && go run .

run-node:
	cd node-backend && npm start

run-react:
	cd react-frontend && npm run dev

test-go:
	cd go-backend && go test -v ./...

test-node:
	cd node-backend && npm test

test-react:
	cd react-frontend && npm test

test-all: test-go test-node test-react

test-go-cover:
	cd go-backend && go test -cover ./...

vet-go:
	cd go-backend && go vet ./...

prepush:
	@echo Running pre-push checks across Go, Node backend, and React frontend...
	$(MAKE) test-go
	$(MAKE) test-node
	$(MAKE) test-react

reset:
	docker compose down --volumes --remove-orphans
	$(MAKE) install

up-db: ensure-env
	$(COMPOSE) up -d postgres

down-db:
	$(COMPOSE) stop postgres

db-logs:
	$(COMPOSE) logs -f --tail=200 postgres

psql:
	$(COMPOSE) exec postgres psql -U "$${POSTGRES_USER:-gotest}" -d "$${POSTGRES_DB:-gotest}"

docker-build: ensure-env
	$(COMPOSE) build

up: ensure-env
	$(COMPOSE) up --build -d

down:
	$(COMPOSE) down

logs:
	$(COMPOSE) logs --tail=200

logs-follow:
	$(COMPOSE) logs -f --tail=200

ifeq ($(IS_WINDOWS),Windows_NT)
logs-errors:
	@$(COMPOSE) logs -f --tail=200 2>&1 | findstr /I /R "error fatal panic exception failed timeout denied refused unavailable status=4[0-9][0-9] status=5[0-9][0-9]"
else
logs-errors:
	@if command -v rg >/dev/null 2>&1; then \
		$(COMPOSE) logs -f --tail=200 2>&1 | rg -i "(\\berror\\b|\\bfatal\\b|\\bpanic\\b|exception|\\bfailed\\b|timeout|denied|refused|unavailable|status=4[0-9]{2}|status=5[0-9]{2})"; \
	else \
		$(COMPOSE) logs -f --tail=200 2>&1 | grep -Ei "(error|fatal|panic|exception|failed|timeout|denied|refused|unavailable|status=4[0-9]{2}|status=5[0-9]{2})"; \
	fi
endif

logs-all:
	$(COMPOSE) logs -f --tail=200

ps:
	$(COMPOSE) ps
