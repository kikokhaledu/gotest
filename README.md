# Go Developer Test Project

Monorepo for a 3-tier application:

- React frontend (`react-frontend`)
- Node.js API gateway (`node-backend`)
- Go backend (`go-backend`)
- PostgreSQL (via `docker-compose`)

Architecture:

`React (5173) -> Node (3000) -> Go (8080) -> PostgreSQL (5432)`

## Documentation

- Go backend: [`go-backend/README.md`](go-backend/README.md)
- Node backend: [`node-backend/README.md`](node-backend/README.md)
- React frontend: [`react-frontend/README.md`](react-frontend/README.md)
- Requirements: [`TEST_REQUIREMENTS.md`](TEST_REQUIREMENTS.md)
- Summary: [`TEST_SUMMARY.md`](TEST_SUMMARY.md)
- Checklist: [`CANDIDATE_CHECKLIST.md`](CANDIDATE_CHECKLIST.md)

## Quick Start (Docker Compose)

1. Create env file:

```bash
cp .env.example .env
```

PowerShell equivalent:

```powershell
Copy-Item .env.example .env
```

Or just run `make up`; the Makefile now auto-creates `.env` from `.env.example` if it is missing.

2. Start everything:

```bash
make up
```

3. Open:
- Frontend: `http://localhost:5173`
- Node Swagger UI: `http://localhost:3000/docs`
- Node OpenAPI JSON: `http://localhost:3000/openapi.json`

4. Default login (demo data):
- username: `admin`
- password: `ChangeMe123@`

5. Logs:

```bash
make logs
```

`make logs` prints recent logs and exits.  
Use `make logs-follow` for continuous logs, or `make logs-errors` for error-only logs.

6. Stop stack:

```bash
make down
```

## Testing

Run all checks:

```bash
make prepush
```

Run per component:

```bash
make test-go
make test-node
make test-react
```

List all commands:

```bash
make help
```
