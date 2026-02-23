# Getting Started - Go Developer Test

Welcome! This guide will help you get started with the test project.

## Step 1: Read the Requirements

1. **Start here**: Read [TEST_REQUIREMENTS.md](./TEST_REQUIREMENTS.md) for complete requirements
2. **Quick reference**: Check [TEST_SUMMARY.md](./TEST_SUMMARY.md) for a quick overview
3. **Track progress**: Use [CANDIDATE_CHECKLIST.md](./CANDIDATE_CHECKLIST.md) to track your work

## Step 2: Set Up Your Environment

### Prerequisites

- **Go 1.21 or higher**
  ```bash
  go version  # Should show 1.21+
  ```

- **Node.js 16 or higher**
  ```bash
  node --version  # Should show v16+
  npm --version
  ```

- **PostgreSQL 14+** (or Docker Desktop for compose-based PostgreSQL)

### Install Dependencies

1. **Go Backend** (no external dependencies needed)
   ```bash
   cd go-backend
   go mod tidy
   ```

2. **Node.js Backend**
   ```bash
   cd node-backend
   npm install
   ```

3. **React Frontend**
   ```bash
   cd react-frontend
   npm install
   ```

## Step 3: Start the Services

**Important**: Start services in this order:

### Terminal 1: PostgreSQL (if running services locally without full compose)
```bash
docker compose up -d postgres
```

### Terminal 2: Go Backend
```bash
cd go-backend
POSTGRES_DSN="postgres://gotest:gotest@localhost:5432/gotest?sslmode=disable" go run .
```
You should see: `Go backend server starting on http://localhost:8080`

### Terminal 3: Node.js Backend
```bash
cd node-backend
npm start
```
You should see: `Node.js backend server running on http://localhost:3000`

### Terminal 4: React Frontend
```bash
cd react-frontend
npm run dev
```
You should see: `Local: http://localhost:5173`

## Step 4: Verify Everything Works

1. **Open the frontend**: http://localhost:5173
   - You should see users and tasks displayed
   - Health status should show "ok"

2. **Test Go backend directly**:
   ```bash
   curl http://localhost:8080/health
   curl http://localhost:8080/api/users
   curl http://localhost:8080/api/tasks
   ```

3. **Test Node.js backend**:
   ```bash
   curl http://localhost:3000/health
   API_KEY=$(curl -s -X POST http://localhost:3000/auth/login \
     -H "Content-Type: application/json" \
     -d '{"username":"admin","password":"ChangeMe123@"}' | jq -r .apiKey)
   curl http://localhost:3000/api/users -H "x-api-key: $API_KEY"
   ```

Default auth credentials:
- username: `admin`
- password: `ChangeMe123@`

## Step 5: Understand the Codebase

### Go Backend Structure

```
go-backend/
├── main.go          # Entry point, starts server
├── server.go        # HTTP server and route handlers
├── data.go          # Data store with users and tasks
└── go.mod           # Go module file
```

**Key Files to Review**:
- `data.go` - See how data is stored and accessed
- `server.go` - See how HTTP endpoints are handled
- `main.go` - See how the server is started

### Current Endpoints

**GET endpoints** (already implemented):
- `GET /health` - Health check
- `GET /api/users` - Get all users
- `GET /api/users/:id` - Get user by ID
- `GET /api/tasks` - Get all tasks (supports ?status= and ?userId= query params)
- `GET /api/stats` - Get statistics

**POST/PUT endpoints** (you need to implement):
- `POST /api/users` - Create new user
- `POST /api/tasks` - Create new task
- `PUT /api/tasks/:id` - Update existing task

## Step 6: Start Implementing

### Recommended Order

1. **Start with User Creation** (`POST /api/users`)
   - This is the simplest endpoint
   - Good for understanding the codebase structure

2. **Then Task Creation** (`POST /api/tasks`)
   - Similar to user creation
   - Adds validation complexity (userId must exist)

3. **Then Task Update** (`PUT /api/tasks/:id`)
   - More complex (partial updates)
   - Good practice for handling edge cases

4. **Finally Logging**
   - Add throughout the process
   - Helps with debugging

### Tips

- **Read the existing code first** - Understand patterns before adding new code
- **Test as you go** - Use `curl` or Postman to test endpoints
- **Write tests early** - Don't wait until the end
- **Check the checklist** - Use `CANDIDATE_CHECKLIST.md` to track progress

## Step 7: Testing Your Code

### Manual Testing

```bash
# Create a user
curl -X POST http://localhost:8080/api/users \
  -H "Content-Type: application/json" \
  -d '{"name":"Test User","email":"test@example.com","role":"developer"}'

# Create a task
curl -X POST http://localhost:8080/api/tasks \
  -H "Content-Type: application/json" \
  -d '{"title":"Test Task","status":"pending","userId":1}'

# Update a task
curl -X PUT http://localhost:8080/api/tasks/1 \
  -H "Content-Type: application/json" \
  -d '{"status":"completed"}'
```

### Writing Tests

Create test files:
- `go-backend/data_test.go` - Test data store functions
- `go-backend/server_test.go` - Test HTTP endpoints

Run tests:
```bash
cd go-backend
go test ./...
go test -v ./...  # Verbose output
go test -cover ./...  # With coverage

# Node backend
cd ../node-backend
npm test

# React frontend
cd ../react-frontend
npm test
```

## Common Issues

### Port Already in Use
```bash
# Find and kill process on port 8080
lsof -ti:8080 | xargs kill -9

# Or use different port
PORT=8081 go run .
```

### Go Module Issues
```bash
cd go-backend
go mod tidy
go mod download
```

### Node.js Issues
```bash
# Clear cache and reinstall
rm -rf node_modules package-lock.json
npm install
```

## Getting Help

- **Go Documentation**: https://golang.org/doc/
- **Go by Example**: https://gobyexample.com/
- **Review existing code** - The codebase has examples of patterns to follow
- **Ask questions** - If requirements are unclear, ask for clarification

## Optional: Docker + Make

If you prefer containerized startup, the repository now includes:
- `docker-compose.yml`
- `Makefile`
- Dockerfiles for each service
- root `.env` / `.env.example` for compose variables and frontend build args

Examples:
```bash
# Required once: create local compose env file
cp .env.example .env

# Start all services in containers
make up

# Stream logs
make logs

# Stop all services
make down
```

Important: compose fails fast if these PostgreSQL values are missing from `.env`:
- `POSTGRES_DB`
- `POSTGRES_USER`
- `POSTGRES_PASSWORD`
- `POSTGRES_DSN`

Manual Docker equivalent:
```bash
docker compose up --build -d
docker compose logs -f
docker compose down
```

## Next Steps

1. ✅ Read [TEST_REQUIREMENTS.md](./TEST_REQUIREMENTS.md)
2. ✅ Set up your environment
3. ✅ Start all services
4. ✅ Verify everything works
5. ✅ Review the codebase
6. ✅ Start implementing required features
7. ✅ Write tests
8. ✅ Submit your work

**Good luck!** 🚀
