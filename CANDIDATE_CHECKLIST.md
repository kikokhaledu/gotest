# Candidate Checklist

Use this checklist to track your progress through the test.

## Phase 1: Setup (30 minutes)

- [x] Forked/cloned the repository
- [x] Installed Go 1.21+
- [x] Installed Node.js 16+
- [x] Installed dependencies (`npm install` in node-backend and react-frontend)
- [x] Started Go backend on port 8080
- [x] Started Node.js backend on port 3000
- [x] Started React frontend on port 5173
- [x] Verified application works end-to-end
- [x] Tested existing endpoints manually

## Phase 2: Core Requirements (2-3 hours)

### User Creation Endpoint
- [x] Added `POST /api/users` endpoint
- [x] Validates name, email, role fields
- [x] Validates email format
- [x] Generates unique ID
- [x] Returns 201 with created user
- [x] Returns 400 for invalid input
- [x] New users appear in GET `/api/users`

### Task Creation Endpoint
- [x] Added `POST /api/tasks` endpoint
- [x] Validates title, status, userId fields
- [x] Validates status enum (pending/in-progress/completed)
- [x] Validates userId exists
- [x] Generates unique ID
- [x] Returns 201 with created task
- [x] Returns 400 for invalid input
- [x] New tasks appear in GET `/api/tasks`

### Task Update Endpoint
- [x] Added `PUT /api/tasks/:id` endpoint
- [x] Supports partial updates
- [x] Validates status if provided
- [x] Validates userId if provided
- [x] Returns 200 with updated task
- [x] Returns 404 if task not found
- [x] Returns 400 for invalid input

### Request Logging
- [x] Added logging for all requests
- [x] Logs HTTP method
- [x] Logs request path
- [x] Logs response status code
- [x] Logs response time/duration
- [x] Logs errors with context
- [x] Logs are readable and consistent

## Phase 3: Advanced Requirements (Optional)

- [x] Data persistence (implemented with PostgreSQL; no JSON-file fallback by design)
- [ ] Caching layer with expiration
- [ ] Request validation middleware
- [x] Enhanced health check with dependency status
- [x] Task status audit history (last change + full history endpoint)
- [x] Docker + Make automation improvements
- [x] API documentation (OpenAPI + Swagger UI)

## Phase 4: Code Quality

### Testing
- [x] Unit tests for data store
- [x] Integration tests for HTTP endpoints
- [x] Test error cases
- [x] Test edge cases
- [x] Code coverage > 70%

### Code Organization
- [x] Follows Go naming conventions
- [x] Functions are focused and single-purpose
- [x] Meaningful comments for complex logic
- [x] No unused code

### Error Handling
- [x] Proper error wrapping
- [x] Appropriate HTTP status codes
- [x] Meaningful error messages
- [x] Errors logged with context
- [x] Internal errors not exposed to clients

### Documentation
- [x] GoDoc comments on exported functions
- [x] API endpoint documentation
- [x] Updated README
- [x] Design decisions documented

## Phase 5: Bonus Tasks (Optional)

- [x] Authentication/API keys
- [x] Rate limiting
- [ ] Metrics/observability
- [x] Database integration

## Submission

- [x] All required features implemented
- [x] Code compiles without errors
- [x] All services run successfully
- [x] Tests written and passing
- [x] Documentation updated
- [ ] Code review notes (optional)

## Notes

Use this space to track any issues, questions, or design decisions:

```
Verified locally on February 23, 2026:
- Go backend: http://localhost:8080
- Node backend: http://localhost:3000
- React frontend: http://localhost:5173
- End-to-end create/update flow validated through Node -> Go:
  - Created user: checklist.user@example.com (id=4)
  - Created task: Checklist Task (id=4)
  - Updated task status to completed
- Added Docker + automation tooling:
  - Root `docker-compose.yml` for all services
  - Root `Makefile` with `up/down/logs/test` shortcuts and cross-platform log helpers (`rg`/`findstr`/`grep`)
  - Dockerfiles and `.dockerignore` files for Go/Node/React services
  - Single root `.env` for compose variables and frontend build-time `VITE_API_URL`
- Added Node gateway security controls:
  - `POST /auth/login` with default seeded credentials (`admin` / `ChangeMe123@`)
  - API key protection for all `/api/*` routes
  - IP-based rate limiting for `/api/*` and `/auth/login`
- Added PostgreSQL integration:
  - Go backend now requires `POSTGRES_DSN` (no in-memory runtime fallback)
  - Auto-creates schema and seeds initial users/tasks on empty DB
  - Docker compose includes dedicated `postgres` service + persistent volume
- Added task audit trail:
  - Task cards include `lastChange` metadata (who/when/what changed)
  - Full task history endpoint: `GET /api/tasks/:id/history`
- Added root project README linking all component docs.
- Added OpenAPI documentation + Swagger UI in Node backend (`/openapi.json`, `/docs`).
- Frontend static favicon 404 noise fixed by shipping `react-frontend/public/vite.svg`.
- Node tests run through a compatibility runner (`node-backend/scripts/run-tests.js`) so test command works on Node 16+ and Node 18+.
- Test coverage: 78.0%
- `go test -race` requires CGO + C compiler (`gcc`) in PATH on this machine.
```
