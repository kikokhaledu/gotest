# Node.js Backend API


## Setup

1. Install dependencies:
```bash
npm install
```

2. Start the server:
```bash
npm start
```

Or for development with auto-reload:
```bash
npm run dev
```

The server will run on `http://localhost:3000` by default.

For the full stack, run from repository root with Docker Compose:

```bash
make up
```

## Configuration

Environment variables:

- `PORT` (default: `3000`)
- `GO_BACKEND_URL` (default: `http://localhost:8080`)
- `NODE_ENV` (default: `development`)
- `GO_REQUEST_TIMEOUT_MS` (default: `5000`)
- `GO_MAX_RESPONSE_BYTES` (default: `1048576`)
- `SHUTDOWN_TIMEOUT_MS` (default: `10000`)
- `AUTH_ENABLED` (default: `true`)
- `AUTH_USERNAME` (default: `admin`)
- `AUTH_PASSWORD` (default: `ChangeMe123@`)
- `AUTH_API_KEY` (default: `dev-local-api-key`)
- `RATE_LIMIT_ENABLED` (default: `true`)
- `RATE_LIMIT_WINDOW_MS` (default: `60000`)
- `RATE_LIMIT_MAX_REQUESTS` (default: `120`)
- `AUTH_RATE_LIMIT_MAX_REQUESTS` (default: `10`)

If you run through `docker compose`, configure these values in the root `.env` file.

Safety behavior:

- Upstream Go requests have timeout enforcement.
- Upstream response size is bounded.
- Graceful shutdown is handled on `SIGINT`/`SIGTERM`.
- API key auth protects `/api/*` endpoints.
- IP-based rate limiting protects `/api/*` and `/auth/login`.

## Default Test Credentials (Out of the Box)

These credentials are seeded by default via env defaults and compose:

- Username: `admin`
- Password: `ChangeMe123@`
- API key returned by login (default): `dev-local-api-key`

These are demo defaults for local/test usage and should be changed outside local development.

Login endpoint:

- `POST /auth/login`
- Body: `{ "username": "admin", "password": "ChangeMe123@" }`

## API Endpoints

### Health Check
- `GET /health` - Check if the server is running

### API Documentation
- `GET /docs` - Swagger UI
- `GET /openapi.json` - OpenAPI 3.0 spec JSON

### Authentication
- `POST /auth/login` - Validate credentials and return API key

### Users
- `GET /api/users` - Get all users
- `GET /api/users/:id` - Get user by ID
- `POST /api/users` - Create a new user
  - Body: `{ "name": "string", "email": "string", "role": "string" }`

### Tasks
- `GET /api/tasks` - Get all tasks (supports query params: `status`, `userId`)
- `POST /api/tasks` - Create a new task
  - Body: `{ "title": "string", "status": "string", "userId": number }`
- `PUT /api/tasks/:id` - Update an existing task
- `GET /api/tasks/:id/history` - Get full change history for one task

### Statistics
- `GET /api/stats` - Get statistics about users and tasks

### Auth Requirement

All `/api/*` endpoints require either:

- `x-api-key: <api-key>`
- `Authorization: Bearer <api-key>`

Optional actor header for task audit history:

- `x-actor: <name>` to record who made task create/update changes

## Testing

```bash
npm test
```

## Example Requests

```bash
# Health check
curl http://localhost:3000/health

# Swagger
curl http://localhost:3000/openapi.json
# then open http://localhost:3000/docs in browser

# Login and capture API key
API_KEY=$(curl -s -X POST http://localhost:3000/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"ChangeMe123@"}' | jq -r .apiKey)

# Get all users
curl http://localhost:3000/api/users -H "x-api-key: $API_KEY"

# Get user by ID
curl http://localhost:3000/api/users/1 -H "x-api-key: $API_KEY"

# Get tasks by status
curl "http://localhost:3000/api/tasks?status=pending" -H "x-api-key: $API_KEY"

# Update task with actor attribution
curl -X PUT http://localhost:3000/api/tasks/1 \
  -H "Content-Type: application/json" \
  -H "x-api-key: $API_KEY" \
  -H "x-actor: admin" \
  -d '{"status":"completed"}'

# Get task history
curl http://localhost:3000/api/tasks/1/history -H "x-api-key: $API_KEY"

# Get statistics
curl http://localhost:3000/api/stats -H "x-api-key: $API_KEY"
```
