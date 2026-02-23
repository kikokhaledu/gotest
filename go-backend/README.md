# Go Backend Service

This is my submission for the Go developer test project.

`React (5173) -> Node.js API gateway (3000) -> Go backend (8080)`

## What I Built

I implemented the full end-to-end flow and production-style wiring for this test project, not just the three required Go endpoints.

What I added/improved:
- I implemented the required Go API behavior for `POST /api/users`, `POST /api/tasks`, and `PUT /api/tasks/:id` with input validation and proper status codes.
- I added structured request logging, panic recovery, consistent JSON error responses, and graceful shutdown behavior.
- I integrated PostgreSQL as the runtime data store (no in-memory runtime fallback), with startup schema creation and initial seed data.
- I added a Node gateway authentication flow (`POST /auth/login`), API-key protection for `/api/*`, and rate limiting for both `/api/*` and `/auth/login`.
- I added Swagger/OpenAPI docs on the Node gateway (`/docs` and `/openapi.json`).
- I containerized the whole stack with Docker Compose and added a root `Makefile` for one-command setup, run, test, and reset flows.
- I expanded test coverage across Go, Node, and React, including edge cases (validation, upstream failures/timeouts, auth, rate-limit behavior, and API client behavior).

## How To Use This (Quick Start)

From repository root:

```bash
cp .env.example .env
make up
```

Then open:
- Frontend: `http://localhost:5173`
- Node gateway docs (Swagger): `http://localhost:3000/docs`
- Node OpenAPI JSON: `http://localhost:3000/openapi.json`

Default login for test data:
- username: `admin`
- password: `ChangeMe123@`

Example API flow:

```bash
# 1) Login and get API key from Node gateway
API_KEY=$(curl -s -X POST http://localhost:3000/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"ChangeMe123@"}' | jq -r .apiKey)

# 2) Call protected API endpoint through Node gateway
curl http://localhost:3000/api/users -H "x-api-key: $API_KEY"
```

Run all tests:

```bash
make prepush
```

Stop the stack:

```bash
make down
```

## Requirements

- Go 1.21+
- PostgreSQL 14+ (or use docker compose PostgreSQL service)

## Run

```bash
cd go-backend
POSTGRES_DSN="postgres://gotest:gotest@localhost:5432/gotest?sslmode=disable" go run .
```

Server starts on `http://localhost:8080` by default.

Environment:
- `PORT` (optional, default `8080`)
- `POSTGRES_DSN` (required, no in-memory fallback is configured)

## Docker + Env

For containerized startup, run from the repository root:

```bash
cp .env.example .env
docker compose up --build -d
```

Root `.env` is the single source of compose values, including:
- PostgreSQL runtime (`POSTGRES_DB`, `POSTGRES_USER`, `POSTGRES_PASSWORD`, `POSTGRES_PORT`, `POSTGRES_DSN`)
- service ports (`GO_BACKEND_PORT`, `NODE_BACKEND_PORT`, `FRONTEND_PORT`)
- Node runtime config (`GO_BACKEND_URL`, `NODE_ENV`)
- Node auth + rate limits (`AUTH_*`, `RATE_LIMIT_*`)
- frontend build-time values (`VITE_API_URL`, `VITE_AUTH_USERNAME`, `VITE_AUTH_PASSWORD`, `VITE_API_KEY`)

Default auth test data (Node gateway):
- username: `admin`
- password: `ChangeMe123@`
- default API key: `dev-local-api-key`

These are demo defaults for local/test use and should be overridden in real deployments.

Compose is configured to fail fast if required PostgreSQL variables are missing:
- `POSTGRES_DB`
- `POSTGRES_USER`
- `POSTGRES_PASSWORD`
- `POSTGRES_DSN`

Note: `VITE_*` variables are injected at frontend image build-time through compose build args.

## Node Gateway Security + API Docs

The Node gateway (port `3000`) sits in front of this Go service and adds:

- Authentication: `POST /auth/login` returns API key
- API key protection on `/api/*`
- Rate limiting on `/api/*` and `/auth/login`
- Swagger/OpenAPI at `GET /docs` and `GET /openapi.json`

Default login:
- username: `admin`
- password: `ChangeMe123@`

Example:

```bash
# Login
API_KEY=$(curl -s -X POST http://localhost:3000/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"ChangeMe123@"}' | jq -r .apiKey)

# Call protected gateway endpoint
curl http://localhost:3000/api/users -H "x-api-key: $API_KEY"

# Swagger
curl http://localhost:3000/openapi.json
# open http://localhost:3000/docs
```

## Makefile Commands (Repo Root)

From repository root (`gotest`), available shortcuts:

- `make help` show available commands
- `make install` install dependencies across Go, Node backend, React frontend
- `make run-go` run Go backend locally
- `make run-node` run Node backend locally
- `make run-react` run React frontend locally
- `make test-go` run Go tests
- `make test-node` run Node backend tests
- `make test-react` run React frontend tests
- `make test-all` run all test suites
- `make test-go-cover` run Go tests with coverage
- `make vet-go` run `go vet`
- `make prepush` run all test suites before push
- `make reset` stop compose stack (with volumes) and reinstall dependencies
- `make up-db` start only PostgreSQL in compose
- `make down-db` stop only PostgreSQL container
- `make db-logs` stream PostgreSQL logs
- `make psql` open psql shell in PostgreSQL container
- `make docker-build` build docker images
- `make up` bring up full docker compose stack
- `make down` stop docker compose stack
- `make logs` show recent docker compose logs and exit
- `make logs-follow` follow docker compose logs continuously
- `make logs-errors` follow error-like log lines only
- `make logs-all` follow full raw docker compose logs
- `make ps` show docker compose service status

## Implemented API

### Health

- `GET /health`

### Users

- `GET /api/users`
- `GET /api/users/:id`
- `POST /api/users`

`POST /api/users` body:

```json
{
  "name": "Test User",
  "email": "test@example.com",
  "role": "developer"
}
```

Validation:
- `name`, `email`, `role` required and non-empty
- basic email format validation
- `Content-Type` must be `application/json`

### Tasks

- `GET /api/tasks` (optional query params: `status`, `userId`)
- `POST /api/tasks`
- `PUT /api/tasks/:id`
- `GET /api/tasks/:id/history`

`POST /api/tasks` body:

```json
{
  "title": "Build feature",
  "status": "pending",
  "userId": 1
}
```

`PUT /api/tasks/:id` body (partial updates):

```json
{
  "status": "completed"
}
```

Optional actor header for task audit tracking:

```http
X-Actor: admin
```

Task objects now include optional `lastChange` metadata (field changed, who changed it, and when).
`GET /api/tasks/:id/history` returns the full change timeline for that task.

Validation:
- `status` must be one of: `pending`, `in-progress`, `completed`
- `userId` must exist for create/update
- `PUT` requires at least one field
- `Content-Type` must be `application/json` for `POST`/`PUT` endpoints
- request body size limit is 1MB for JSON write endpoints

### Stats

- `GET /api/stats`

## Response Semantics

- Success responses are JSON.
- Error responses use consistent JSON shape:

```json
{
  "error": "message"
}
```

Common status codes:
- `200` success
- `201` created
- `400` validation / malformed request
- `413` request body too large
- `415` unsupported media type
- `404` resource not found
- `405` method not allowed
- `500` internal server error

## Design Decisions

- `net/http` with `ServeMux` kept intentionally for low dependency surface and easy review.
- Store is abstracted behind a `Store` interface to keep handlers testable and decoupled from storage details.
- Runtime storage is PostgreSQL-only; process startup fails fast if `POSTGRES_DSN` is missing/unreachable.
- Read-path datastore failures are treated as server errors (`500`) instead of returning misleading empty payloads.
- PostgreSQL schema is created on startup and seeded once with initial users/tasks when tables are empty.
- Task updates are audit-logged in PostgreSQL (`task_history`) with actor, timestamp, and before/after values.
- JSON decoding uses `DisallowUnknownFields` and size limits for predictable validation behavior.
- Middleware chain handles CORS, panic recovery, and structured request logging consistently.
- Server handles graceful shutdown on `SIGINT`/`SIGTERM` with a bounded shutdown timeout.

## Request Logging

All requests are logged with:
- method
- path
- status
- duration

Example:

```text
request method=POST path=/api/users status=201 duration=1.2ms
```

## Testing

Run Go tests:

```bash
cd go-backend
go test -v ./...
go test -cover ./...
```

Run full stack test suite from repo root:

```bash
make prepush
```

Current Go coverage result:
- `78.0%` statements

Edge-case coverage includes:
- API validation branches (unknown fields, malformed/invalid JSON, malformed content type, empty body, oversized body)
- task/user validation branches (`userId` parsing, invalid statuses, not-found paths)
- PostgreSQL store error paths (query/insert errors, update rollback behavior, seed-empty vs seed-skip behavior)
- Node gateway auth/rate-limit edge behavior and HTTP client timeout/abort/oversize/error mapping
- React API service request/response behavior with mocked axios client

Race detector:

```bash
go test -race ./...
```

Note: `-race` requires CGO and a C compiler (for example, `gcc`) available in `PATH`.
