# Go Developer Test Requirements

## Overview

This test project evaluates a Go developer's ability to work with HTTP servers, API integration, data structures, error handling, and best practices. The project consists of a three-tier architecture where React calls Node.js, which calls Go.

## Architecture

```
React Frontend (port 5173)
    ↓
Node.js Backend (port 3000) - API Gateway
    ↓
Go Backend (port 8080) - Data Source
```

## Current State

The project is set up with:
- **Go Backend**: Basic HTTP server serving users and tasks data
- **Node.js Backend**: API gateway that proxies requests to Go backend
- **React Frontend**: UI that displays data from Node.js backend

## Test Requirements

### Phase 1: Understanding & Setup (30 minutes)

1. **Fork/Clone the repository**
2. **Set up the development environment**
   - Ensure Go 1.21+ is installed
   - Ensure Node.js 16+ is installed
   - Install dependencies for Node.js and React
3. **Run all three services**
   - Start Go backend on port 8080
   - Start Node.js backend on port 3000
   - Start React frontend on port 5173
4. **Verify the application works**
   - Test all endpoints manually
   - Verify data flows correctly through all layers

### Phase 2: Core Requirements (2-3 hours)

#### 2.1 Add User Creation Endpoint (Required)

**Task**: Implement a POST endpoint in the Go backend to create new users.

**Requirements**:
- Add `POST /api/users` endpoint in `go-backend/server.go`
- Accept JSON body with: `name`, `email`, `role`
- Validate all fields are present and non-empty
- Validate email format (basic validation is acceptable)
- Generate a unique ID (increment from max existing ID)
- Return the created user with status 201
- Handle errors appropriately (400 for validation, 500 for server errors)

**Expected Behavior**:
- Successfully create users via API
- New users should appear in the GET `/api/users` response
- Invalid requests should return appropriate error messages

#### 2.2 Add Task Creation Endpoint (Required)

**Task**: Implement a POST endpoint in the Go backend to create new tasks.

**Requirements**:
- Add `POST /api/tasks` endpoint in `go-backend/server.go`
- Accept JSON body with: `title`, `status`, `userId`
- Validate all fields are present
- Validate `status` is one of: "pending", "in-progress", "completed"
- Validate `userId` exists in the users list
- Generate a unique ID
- Return the created task with status 201
- Handle errors appropriately

**Expected Behavior**:
- Successfully create tasks via API
- New tasks should appear in the GET `/api/tasks` response
- Invalid userId should return 400 error
- Invalid status should return 400 error

#### 2.3 Implement Task Update Endpoint (Required)

**Task**: Implement a PUT/PATCH endpoint to update existing tasks.

**Requirements**:
- Add `PUT /api/tasks/:id` endpoint
- Accept JSON body with optional fields: `title`, `status`, `userId`
- Update only provided fields (partial updates)
- Validate status if provided
- Validate userId exists if provided
- Return 404 if task not found
- Return updated task with status 200
- Handle errors appropriately

**Expected Behavior**:
- Successfully update task fields
- Partial updates work correctly
- Non-existent task IDs return 404

#### 2.4 Add Request Logging (Required)

**Task**: Add structured logging for all HTTP requests.

**Requirements**:
- Log all incoming requests with:
  - HTTP method
  - Request path
  - Response status code
  - Response time (duration)
- Use Go's standard `log` package or a logging library
- Format logs consistently
- Log errors with appropriate detail

**Expected Behavior**:
- All API requests are logged to console
- Logs are readable and informative
- Error logs include relevant context

### Phase 3: Advanced Requirements (2-3 hours)

#### 3.1 Add Data Persistence (Optional but Recommended)

**Task**: Replace in-memory storage with file-based persistence.

**Requirements**:
- Save data to JSON file on changes
- Load data from JSON file on startup
- Handle file read/write errors gracefully
- Ensure thread-safe file operations
- Maintain backward compatibility with existing data structure

**Expected Behavior**:
- Data persists across server restarts
- File operations don't block the server
- Corrupted files are handled gracefully

#### 3.2 Implement Caching Layer (Optional)

**Task**: Add caching to reduce redundant API calls from Node.js.

**Requirements**:
- Cache GET responses for users and tasks
- Implement cache expiration (e.g., 5 minutes)
- Provide cache invalidation on POST/PUT operations
- Use Go's sync package for thread-safe cache
- Add cache statistics endpoint

**Expected Behavior**:
- Repeated requests return cached data
- Cache is invalidated on data mutations
- Cache statistics are available

#### 3.3 Add Request Validation Middleware (Optional)

**Task**: Create middleware for request validation and error handling.

**Requirements**:
- Create reusable validation middleware
- Validate JSON request bodies
- Return consistent error response format
- Handle malformed JSON gracefully
- Add request size limits

**Expected Behavior**:
- Invalid requests are caught early
- Error responses are consistent
- Large requests are rejected appropriately

#### 3.4 Add Health Check with Dependencies (Optional)

**Task**: Enhance health check to verify internal state.

**Requirements**:
- Extend `/health` endpoint
- Check data store is accessible
- Return detailed health status
- Include version information
- Add readiness vs liveness checks

**Expected Behavior**:
- Health check reflects actual system state
- Detailed status helps with debugging

### Phase 4: Code Quality & Best Practices (Ongoing)

#### 4.1 Code Organization

**Requirements**:
- Follow Go naming conventions
- Organize code into logical packages if needed
- Keep functions focused and single-purpose
- Add meaningful comments for complex logic
- Remove unused code

#### 4.2 Error Handling

**Requirements**:
- Use proper error wrapping (`fmt.Errorf` with `%w`)
- Return appropriate HTTP status codes
- Provide meaningful error messages
- Log errors with context
- Don't expose internal errors to clients

#### 4.3 Testing

**Requirements**:
- Write unit tests for data store operations
- Write integration tests for HTTP endpoints
- Aim for at least 70% code coverage
- Use Go's testing package
- Test error cases and edge cases

**Example Test Structure**:
```go
// Test file: go-backend/data_test.go
func TestDataStore_GetUserByID(t *testing.T) {
    // Test cases
}

// Test file: go-backend/server_test.go
func TestServer_handleUsers(t *testing.T) {
    // HTTP endpoint tests
}
```

#### 4.4 Documentation

**Requirements**:
- Add GoDoc comments to exported functions
- Document API endpoints (can use comments or separate doc)
- Update README with new features
- Document any design decisions

### Phase 5: Bonus Tasks (Optional)

#### 5.1 Add Authentication (Bonus)

- Implement basic API key authentication
- Protect endpoints with middleware
- Add authentication to health check

#### 5.2 Add Rate Limiting (Bonus)

- Implement rate limiting per IP
- Return appropriate headers (X-RateLimit-*)
- Handle rate limit exceeded gracefully

#### 5.3 Add Metrics/Observability (Bonus)

- Add Prometheus metrics
- Track request counts, durations, errors
- Expose metrics endpoint

#### 5.4 Add Database Integration (Bonus)

- Replace file storage with SQLite or PostgreSQL
- Use database migrations
- Add connection pooling

## Submission Requirements

### Code Submission

1. **Fork the repository** or create a new branch
2. **Implement the required features** (Phase 2)
3. **Optionally implement advanced features** (Phase 3)
4. **Write tests** for your code (Phase 4.3)
5. **Update documentation** as needed

### Deliverables

1. **Working Code**
   - All required features implemented
   - Code compiles without errors
   - All services run successfully

2. **Tests**
   - Unit tests for core functionality
   - Integration tests for API endpoints
   - Test coverage report

3. **Documentation**
   - Updated README with new features
   - API documentation (endpoints, request/response formats)
   - Any setup/configuration notes

4. **Code Review Notes** (Optional)
   - Document any design decisions
   - Explain trade-offs made
   - Note any limitations or future improvements

## Evaluation Criteria

### Technical Skills (60%)

- **Correctness**: Does the code work as expected?
- **Code Quality**: Is the code clean, readable, and maintainable?
- **Error Handling**: Are errors handled appropriately?
- **Testing**: Are there adequate tests with good coverage?

### Go Best Practices (25%)

- **Idiomatic Go**: Does the code follow Go conventions?
- **Concurrency**: Proper use of goroutines, channels, mutexes if needed
- **Standard Library**: Effective use of Go standard library
- **Performance**: Efficient algorithms and data structures

### Problem Solving (15%)

- **Architecture**: Good design decisions
- **Edge Cases**: Handling of edge cases and errors
- **Documentation**: Clear documentation and comments
- **Time Management**: Realistic scope and prioritization

## Time Expectations

- **Minimum**: Complete Phase 2 (Core Requirements) - 2-3 hours
- **Recommended**: Complete Phase 2 + Phase 3 (1-2 items) - 4-5 hours
- **Excellent**: Complete Phase 2 + Phase 3 + Phase 4 - 6-8 hours

## Getting Help

- Review Go documentation: https://golang.org/doc/
- Check Go by Example: https://gobyexample.com/
- Review the existing codebase for patterns
- Ask clarifying questions if requirements are unclear

## Notes

- Focus on code quality over quantity
- It's better to complete fewer features well than many features poorly
- Write tests as you develop, not after
- Document your thought process in code comments
- If you run out of time, document what you would do next

## Questions to Consider

1. How would you handle concurrent requests safely?
2. What would you change if this needed to scale to 1000+ requests/second?
3. How would you add authentication/authorization?
4. What would you do differently if this was production code?
5. How would you monitor and debug this in production?

---

**Good luck!** Focus on writing clean, maintainable Go code that demonstrates your understanding of the language and best practices.
