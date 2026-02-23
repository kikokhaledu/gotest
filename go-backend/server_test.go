package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestPOSTUsersCreatesUser(t *testing.T) {
	s := newTestServer(t)

	body := `{"name":"New User","email":"new.user@example.com","role":"developer"}`
	res := performRequest(s.Handler(), http.MethodPost, "/api/users", body)
	if res.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusCreated, res.Code, res.Body.String())
	}

	var created User
	decodeJSONResponse(t, res.Body.Bytes(), &created)
	if created.ID != 4 {
		t.Fatalf("expected user ID 4, got %d", created.ID)
	}
	if created.Email != "new.user@example.com" {
		t.Fatalf("unexpected email: %s", created.Email)
	}

	getRes := performRequest(s.Handler(), http.MethodGet, "/api/users", "")
	if getRes.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, getRes.Code)
	}

	var usersResp UsersResponse
	decodeJSONResponse(t, getRes.Body.Bytes(), &usersResp)
	if usersResp.Count != 4 {
		t.Fatalf("expected 4 users after creation, got %d", usersResp.Count)
	}
}

func TestPOSTUsersValidation(t *testing.T) {
	s := newTestServer(t)

	testCases := []struct {
		name string
		body string
	}{
		{
			name: "missing fields",
			body: `{"name":"Only Name"}`,
		},
		{
			name: "invalid email",
			body: `{"name":"User","email":"invalid-email","role":"developer"}`,
		},
		{
			name: "unknown field",
			body: `{"name":"User","email":"user@example.com","role":"developer","bad":true}`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			res := performRequest(s.Handler(), http.MethodPost, "/api/users", tc.body)
			if res.Code != http.StatusBadRequest {
				t.Fatalf("expected status %d, got %d body=%s", http.StatusBadRequest, res.Code, res.Body.String())
			}
		})
	}
}

func TestPOSTUsersTrimsWhitespace(t *testing.T) {
	s := newTestServer(t)

	res := performRequest(
		s.Handler(),
		http.MethodPost,
		"/api/users",
		`{"name":"  New User  ","email":"  new.user@example.com  ","role":"  developer  "}`,
	)
	if res.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusCreated, res.Code, res.Body.String())
	}

	var created User
	decodeJSONResponse(t, res.Body.Bytes(), &created)
	if created.Name != "New User" {
		t.Fatalf("expected trimmed name, got %q", created.Name)
	}
	if created.Email != "new.user@example.com" {
		t.Fatalf("expected trimmed email, got %q", created.Email)
	}
	if created.Role != "developer" {
		t.Fatalf("expected trimmed role, got %q", created.Role)
	}
}

func TestPOSTTasksValidationAndCreate(t *testing.T) {
	s := newTestServer(t)

	invalidStatus := performRequest(s.Handler(), http.MethodPost, "/api/tasks", `{"title":"Task","status":"bad","userId":1}`)
	if invalidStatus.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, invalidStatus.Code)
	}

	unknownUser := performRequest(s.Handler(), http.MethodPost, "/api/tasks", `{"title":"Task","status":"pending","userId":999}`)
	if unknownUser.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, unknownUser.Code)
	}

	createRes := performRequest(s.Handler(), http.MethodPost, "/api/tasks", `{"title":"Task","status":"pending","userId":1}`)
	if createRes.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusCreated, createRes.Code, createRes.Body.String())
	}

	var created Task
	decodeJSONResponse(t, createRes.Body.Bytes(), &created)
	if created.ID != 4 {
		t.Fatalf("expected task ID 4, got %d", created.ID)
	}
	if created.Status != "pending" {
		t.Fatalf("expected status pending, got %s", created.Status)
	}
	if created.LastChange == nil {
		t.Fatal("expected lastChange to be populated on create")
	}
	if created.LastChange.ChangedBy != defaultActorName {
		t.Fatalf("expected default actor %q, got %q", defaultActorName, created.LastChange.ChangedBy)
	}
	if created.LastChange.Field != "status" {
		t.Fatalf("expected status history field, got %q", created.LastChange.Field)
	}
}

func TestGETUserByID(t *testing.T) {
	s := newTestServer(t)

	found := performRequest(s.Handler(), http.MethodGet, "/api/users/1", "")
	if found.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, found.Code)
	}

	var user User
	decodeJSONResponse(t, found.Body.Bytes(), &user)
	if user.ID != 1 {
		t.Fatalf("expected user ID 1, got %d", user.ID)
	}

	notFound := performRequest(s.Handler(), http.MethodGet, "/api/users/999", "")
	if notFound.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, notFound.Code)
	}

	invalid := performRequest(s.Handler(), http.MethodGet, "/api/users/abc", "")
	if invalid.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, invalid.Code)
	}
}

func TestGETUsersReadErrorReturnsInternalServerError(t *testing.T) {
	s := NewServer(&errorReadStore{usersErr: errors.New("db unavailable")})
	s.logger = log.New(io.Discard, "", 0)

	res := performRequest(s.Handler(), http.MethodGet, "/api/users", "")
	if res.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusInternalServerError, res.Code, res.Body.String())
	}
}

func TestGETUserByIDReadErrorReturnsInternalServerError(t *testing.T) {
	s := NewServer(&errorReadStore{userByIDErr: errors.New("db unavailable")})
	s.logger = log.New(io.Discard, "", 0)

	res := performRequest(s.Handler(), http.MethodGet, "/api/users/1", "")
	if res.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusInternalServerError, res.Code, res.Body.String())
	}
}

func TestGETTasksAndStats(t *testing.T) {
	s := newTestServer(t)

	pendingTasks := performRequest(s.Handler(), http.MethodGet, "/api/tasks?status=pending", "")
	if pendingTasks.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, pendingTasks.Code)
	}

	var tasksResp TasksResponse
	decodeJSONResponse(t, pendingTasks.Body.Bytes(), &tasksResp)
	if tasksResp.Count != 1 {
		t.Fatalf("expected 1 pending task, got %d", tasksResp.Count)
	}

	userTasks := performRequest(s.Handler(), http.MethodGet, "/api/tasks?userId=2", "")
	if userTasks.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, userTasks.Code)
	}
	decodeJSONResponse(t, userTasks.Body.Bytes(), &tasksResp)
	if tasksResp.Count != 1 {
		t.Fatalf("expected 1 task for user 2, got %d", tasksResp.Count)
	}

	statsRes := performRequest(s.Handler(), http.MethodGet, "/api/stats", "")
	if statsRes.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, statsRes.Code)
	}

	var stats StatsResponse
	decodeJSONResponse(t, statsRes.Body.Bytes(), &stats)
	if stats.Users.Total != 3 {
		t.Fatalf("expected 3 users, got %d", stats.Users.Total)
	}
	if stats.Tasks.Total != 3 {
		t.Fatalf("expected 3 tasks, got %d", stats.Tasks.Total)
	}

	invalidUserQuery := performRequest(s.Handler(), http.MethodGet, "/api/tasks?userId=abc", "")
	if invalidUserQuery.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, invalidUserQuery.Code)
	}

	invalidNonPositiveUserQuery := performRequest(s.Handler(), http.MethodGet, "/api/tasks?userId=0", "")
	if invalidNonPositiveUserQuery.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, invalidNonPositiveUserQuery.Code)
	}
}

func TestGETTasksReadErrorReturnsInternalServerError(t *testing.T) {
	s := NewServer(&errorReadStore{tasksErr: errors.New("db unavailable")})
	s.logger = log.New(io.Discard, "", 0)

	res := performRequest(s.Handler(), http.MethodGet, "/api/tasks", "")
	if res.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusInternalServerError, res.Code, res.Body.String())
	}
}

func TestGETStatsReadErrorReturnsInternalServerError(t *testing.T) {
	s := NewServer(&errorReadStore{statsErr: errors.New("db unavailable")})
	s.logger = log.New(io.Discard, "", 0)

	res := performRequest(s.Handler(), http.MethodGet, "/api/stats", "")
	if res.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusInternalServerError, res.Code, res.Body.String())
	}
}

func TestPUTTaskByIDPartialUpdate(t *testing.T) {
	s := newTestServer(t)

	updateRes := performRequestWithHeaders(
		s.Handler(),
		http.MethodPut,
		"/api/tasks/1",
		`{"status":"completed"}`,
		map[string]string{actorHeaderName: "admin"},
	)
	if updateRes.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusOK, updateRes.Code, updateRes.Body.String())
	}

	var updated Task
	decodeJSONResponse(t, updateRes.Body.Bytes(), &updated)
	if updated.Status != "completed" {
		t.Fatalf("expected status completed, got %s", updated.Status)
	}
	if updated.Title == "" {
		t.Fatalf("expected title to remain populated")
	}
	if updated.LastChange == nil {
		t.Fatal("expected lastChange after update")
	}
	if updated.LastChange.ChangedBy != "admin" {
		t.Fatalf("expected changedBy admin, got %q", updated.LastChange.ChangedBy)
	}
	if updated.LastChange.Field != "status" {
		t.Fatalf("expected last change field status, got %q", updated.LastChange.Field)
	}

	notFound := performRequest(s.Handler(), http.MethodPut, "/api/tasks/999", `{"status":"completed"}`)
	if notFound.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, notFound.Code)
	}

	invalidBody := performRequest(s.Handler(), http.MethodPut, "/api/tasks/1", `{}`)
	if invalidBody.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, invalidBody.Code)
	}

	invalidStatus := performRequest(s.Handler(), http.MethodPut, "/api/tasks/1", `{"status":"bad"}`)
	if invalidStatus.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, invalidStatus.Code)
	}

	unknownField := performRequest(s.Handler(), http.MethodPut, "/api/tasks/1", `{"status":"completed","bad":true}`)
	if unknownField.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusBadRequest, unknownField.Code, unknownField.Body.String())
	}

	invalidID := performRequest(s.Handler(), http.MethodPut, "/api/tasks/not-an-id", `{"status":"completed"}`)
	if invalidID.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, invalidID.Code)
	}
}

func TestGETTaskHistory(t *testing.T) {
	s := newTestServer(t)

	_ = performRequestWithHeaders(
		s.Handler(),
		http.MethodPut,
		"/api/tasks/1",
		`{"status":"in-progress"}`,
		map[string]string{actorHeaderName: "alice"},
	)
	_ = performRequestWithHeaders(
		s.Handler(),
		http.MethodPut,
		"/api/tasks/1",
		`{"status":"completed"}`,
		map[string]string{actorHeaderName: "bob"},
	)

	res := performRequest(s.Handler(), http.MethodGet, "/api/tasks/1/history", "")
	if res.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusOK, res.Code, res.Body.String())
	}

	var historyResp TaskHistoryResponse
	decodeJSONResponse(t, res.Body.Bytes(), &historyResp)
	if historyResp.TaskID != 1 {
		t.Fatalf("expected taskId 1, got %d", historyResp.TaskID)
	}
	if historyResp.Count < 2 {
		t.Fatalf("expected at least 2 history entries, got %d", historyResp.Count)
	}
	if historyResp.History[0].ChangedBy != "bob" {
		t.Fatalf("expected most recent change by bob, got %q", historyResp.History[0].ChangedBy)
	}

	notFound := performRequest(s.Handler(), http.MethodGet, "/api/tasks/999/history", "")
	if notFound.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, notFound.Code)
	}

	invalid := performRequest(s.Handler(), http.MethodGet, "/api/tasks/not-int/history", "")
	if invalid.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, invalid.Code)
	}
}

func TestGETTaskHistoryReadErrorReturnsInternalServerError(t *testing.T) {
	s := NewServer(&errorReadStore{historyErr: errors.New("db unavailable")})
	s.logger = log.New(io.Discard, "", 0)

	res := performRequest(s.Handler(), http.MethodGet, "/api/tasks/1/history", "")
	if res.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusInternalServerError, res.Code, res.Body.String())
	}
}

func TestMethodNotAllowedReturnsJSONError(t *testing.T) {
	s := newTestServer(t)

	res := performRequest(s.Handler(), http.MethodPatch, "/api/tasks/1", `{"status":"completed"}`)
	if res.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status %d, got %d", http.StatusMethodNotAllowed, res.Code)
	}

	var errResp map[string]string
	decodeJSONResponse(t, res.Body.Bytes(), &errResp)
	if errResp["error"] == "" {
		t.Fatalf("expected error message in response, got %v", errResp)
	}

	healthMethod := performRequest(s.Handler(), http.MethodPost, "/health", "{}")
	if healthMethod.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status %d, got %d", http.StatusMethodNotAllowed, healthMethod.Code)
	}
}

func TestCORSOptionsRequest(t *testing.T) {
	s := newTestServer(t)

	res := performRequest(s.Handler(), http.MethodOptions, "/api/tasks", "")
	if res.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, res.Code)
	}
	if res.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Fatalf("expected CORS allow-origin header, got %q", res.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestInvalidJSONAndTypeErrors(t *testing.T) {
	s := newTestServer(t)

	malformedJSON := performRequest(s.Handler(), http.MethodPost, "/api/users", `{"name":"x",`)
	if malformedJSON.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, malformedJSON.Code)
	}

	wrongType := performRequest(s.Handler(), http.MethodPost, "/api/tasks", `{"title":"task","status":"pending","userId":"abc"}`)
	if wrongType.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, wrongType.Code)
	}

	multipleObjects := performRequest(
		s.Handler(),
		http.MethodPost,
		"/api/users",
		`{"name":"x","email":"x@example.com","role":"developer"}{"name":"y"}`,
	)
	if multipleObjects.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, multipleObjects.Code)
	}
}

func TestWriteEndpointsRequireJSONContentType(t *testing.T) {
	s := newTestServer(t)

	postUserNoTypeReq := httptest.NewRequest(
		http.MethodPost,
		"/api/users",
		strings.NewReader(`{"name":"x","email":"x@example.com","role":"developer"}`),
	)
	postUserNoTypeRes := httptest.NewRecorder()
	s.Handler().ServeHTTP(postUserNoTypeRes, postUserNoTypeReq)
	if postUserNoTypeRes.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("expected status %d, got %d", http.StatusUnsupportedMediaType, postUserNoTypeRes.Code)
	}

	postTaskWrongTypeReq := httptest.NewRequest(
		http.MethodPost,
		"/api/tasks",
		strings.NewReader(`{"title":"x","status":"pending","userId":1}`),
	)
	postTaskWrongTypeReq.Header.Set("Content-Type", "text/plain")
	postTaskWrongTypeRes := httptest.NewRecorder()
	s.Handler().ServeHTTP(postTaskWrongTypeRes, postTaskWrongTypeReq)
	if postTaskWrongTypeRes.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("expected status %d, got %d", http.StatusUnsupportedMediaType, postTaskWrongTypeRes.Code)
	}

	putTaskWrongTypeReq := httptest.NewRequest(
		http.MethodPut,
		"/api/tasks/1",
		strings.NewReader(`{"status":"completed"}`),
	)
	putTaskWrongTypeReq.Header.Set("Content-Type", "text/plain")
	putTaskWrongTypeRes := httptest.NewRecorder()
	s.Handler().ServeHTTP(putTaskWrongTypeRes, putTaskWrongTypeReq)
	if putTaskWrongTypeRes.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("expected status %d, got %d", http.StatusUnsupportedMediaType, putTaskWrongTypeRes.Code)
	}
}

func TestWriteEndpointsRejectMalformedContentTypeHeader(t *testing.T) {
	s := newTestServer(t)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/users",
		strings.NewReader(`{"name":"x","email":"x@example.com","role":"developer"}`),
	)
	req.Header.Set("Content-Type", "application/json; charset")

	res := httptest.NewRecorder()
	s.Handler().ServeHTTP(res, req)

	if res.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusUnsupportedMediaType, res.Code, res.Body.String())
	}

	var errResp map[string]string
	decodeJSONResponse(t, res.Body.Bytes(), &errResp)
	if errResp["error"] != "invalid content type header" {
		t.Fatalf("expected malformed content-type message, got %q", errResp["error"])
	}
}

func TestRequestBodyTooLarge(t *testing.T) {
	s := newTestServer(t)

	oversizedName := strings.Repeat("a", maxRequestBodyBytes)
	body := `{"name":"` + oversizedName + `","email":"big@example.com","role":"developer"}`
	res := performRequest(s.Handler(), http.MethodPost, "/api/users", body)
	if res.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusRequestEntityTooLarge, res.Code, res.Body.String())
	}

	var errResp map[string]string
	decodeJSONResponse(t, res.Body.Bytes(), &errResp)
	if errResp["error"] == "" {
		t.Fatalf("expected error payload, got %v", errResp)
	}
}

func TestRecoveryMiddleware(t *testing.T) {
	s := newTestServer(t)
	panicHandler := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("boom")
	})

	wrapped := s.recoveryMiddleware(panicHandler)
	res := performRequest(wrapped, http.MethodGet, "/panic", "")
	if res.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, res.Code)
	}
}

func TestParseIDFromPath(t *testing.T) {
	id, err := parseIDFromPath("/api/tasks/123", "/api/tasks/")
	if err != nil {
		t.Fatalf("expected valid ID parse, got %v", err)
	}
	if id != 123 {
		t.Fatalf("expected ID 123, got %d", id)
	}

	if _, err := parseIDFromPath("/api/tasks/", "/api/tasks/"); err == nil {
		t.Fatal("expected error for empty ID")
	}
	if _, err := parseIDFromPath("/api/tasks/12/extra", "/api/tasks/"); err == nil {
		t.Fatal("expected error for nested path")
	}
	if _, err := parseIDFromPath("/api/tasks/not-int", "/api/tasks/"); err == nil {
		t.Fatal("expected error for non-integer ID")
	}
}

func TestParseTaskHistoryIDFromPath(t *testing.T) {
	id, err := parseTaskHistoryIDFromPath("/api/tasks/123/history", "/api/tasks/")
	if err != nil {
		t.Fatalf("expected valid ID parse, got %v", err)
	}
	if id != 123 {
		t.Fatalf("expected ID 123, got %d", id)
	}

	if _, err := parseTaskHistoryIDFromPath("/api/tasks/123", "/api/tasks/"); err == nil {
		t.Fatal("expected error for missing /history suffix")
	}
	if _, err := parseTaskHistoryIDFromPath("/api/tasks//history", "/api/tasks/"); err == nil {
		t.Fatal("expected error for empty task ID")
	}
	if _, err := parseTaskHistoryIDFromPath("/api/tasks/12/extra/history", "/api/tasks/"); err == nil {
		t.Fatal("expected error for nested path")
	}
	if _, err := parseTaskHistoryIDFromPath("/api/tasks/not-int/history", "/api/tasks/"); err == nil {
		t.Fatal("expected error for non-integer ID")
	}
}

func TestDecodeJSONBodyNilBody(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/users", nil)
	req.Body = nil

	var payload createUserRequest
	err := decodeJSONBody(req, &payload)
	if err == nil {
		t.Fatal("expected error for nil body")
	}
	if !strings.Contains(err.Error(), "required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDecodeJSONBodyEmptyBody(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/users", strings.NewReader(""))

	var payload createUserRequest
	err := decodeJSONBody(req, &payload)
	if err == nil {
		t.Fatal("expected error for empty body")
	}
	if !strings.Contains(err.Error(), "required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoggingMiddlewareLogsMethodPathStatus(t *testing.T) {
	s := newTestServer(t)

	var logBuffer bytes.Buffer
	s.logger = log.New(&logBuffer, "", 0)

	res := performRequest(s.Handler(), http.MethodGet, "/health", "")
	if res.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, res.Code)
	}

	logOutput := logBuffer.String()
	if !strings.Contains(logOutput, "method=GET") {
		t.Fatalf("expected log output to include method, got: %s", logOutput)
	}
	if !strings.Contains(logOutput, "path=/health") {
		t.Fatalf("expected log output to include path, got: %s", logOutput)
	}
	if !strings.Contains(logOutput, "status=200") {
		t.Fatalf("expected log output to include status code, got: %s", logOutput)
	}
}

func TestRunWithContextShutsDownOnCancel(t *testing.T) {
	s := newTestServer(t)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}

	httpServer := &http.Server{
		Handler: s.Handler(),
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	runErrCh := make(chan error, 1)
	go func() {
		runErrCh <- s.runWithContext(ctx, httpServer, func() error {
			return httpServer.Serve(listener)
		})
	}()

	address := "http://" + listener.Addr().String() + "/health"
	deadline := time.Now().Add(2 * time.Second)
	for {
		resp, reqErr := http.Get(address)
		if reqErr == nil {
			resp.Body.Close()
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("server never became reachable: %v", reqErr)
		}
		time.Sleep(20 * time.Millisecond)
	}

	cancel()

	select {
	case runErr := <-runErrCh:
		if runErr != nil {
			t.Fatalf("expected clean shutdown, got error: %v", runErr)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for graceful shutdown")
	}
}

func newTestServer(t *testing.T) *Server {
	t.Helper()

	ds := NewDataStore(
		[]User{
			{ID: 1, Name: "John Doe", Email: "john@example.com", Role: "developer"},
			{ID: 2, Name: "Jane Smith", Email: "jane@example.com", Role: "designer"},
			{ID: 3, Name: "Bob Johnson", Email: "bob@example.com", Role: "manager"},
		},
		[]Task{
			{ID: 1, Title: "Implement authentication", Status: "pending", UserID: 1},
			{ID: 2, Title: "Design user interface", Status: "in-progress", UserID: 2},
			{ID: 3, Title: "Review code changes", Status: "completed", UserID: 3},
		},
	)

	s := NewServer(ds)
	s.logger = log.New(io.Discard, "", 0)
	return s
}

func performRequest(handler http.Handler, method, path, body string) *httptest.ResponseRecorder {
	return performRequestWithHeaders(handler, method, path, body, map[string]string{})
}

func performRequestWithHeaders(
	handler http.Handler,
	method string,
	path string,
	body string,
	headers map[string]string,
) *httptest.ResponseRecorder {
	var requestBody io.Reader
	if body != "" {
		requestBody = strings.NewReader(body)
	}

	req := httptest.NewRequest(method, path, requestBody)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	return res
}

func decodeJSONResponse(t *testing.T, body []byte, dst any) {
	t.Helper()

	if err := json.Unmarshal(body, dst); err != nil {
		t.Fatalf("failed to decode JSON response: %v body=%s", err, string(body))
	}
}

type errorReadStore struct {
	usersErr    error
	userByIDErr error
	tasksErr    error
	statsErr    error
	historyErr  error
}

func (s *errorReadStore) GetUsers() ([]User, error) {
	if s.usersErr != nil {
		return nil, s.usersErr
	}
	return []User{}, nil
}

func (s *errorReadStore) GetUserByID(id int) (User, bool, error) {
	if s.userByIDErr != nil {
		return User{}, false, s.userByIDErr
	}
	return User{}, false, nil
}

func (s *errorReadStore) GetTasks(status, userID string) ([]Task, error) {
	if s.tasksErr != nil {
		return nil, s.tasksErr
	}
	return []Task{}, nil
}

func (s *errorReadStore) GetStats() (StatsResponse, error) {
	if s.statsErr != nil {
		return StatsResponse{}, s.statsErr
	}
	return StatsResponse{}, nil
}

func (s *errorReadStore) GetTaskHistory(taskID int) ([]TaskHistoryItem, error) {
	if s.historyErr != nil {
		return nil, s.historyErr
	}
	return []TaskHistoryItem{}, nil
}

func (s *errorReadStore) CreateUser(name, email, role string) (User, error) {
	return User{}, nil
}

func (s *errorReadStore) CreateTask(title, status string, userID int, actor string) (Task, error) {
	return Task{}, nil
}

func (s *errorReadStore) UpdateTask(id int, update TaskUpdate, actor string) (Task, error) {
	return Task{}, nil
}
