package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type Server struct {
	dataStore Store
	logger    *log.Logger
	handler   http.Handler
}

var emailRegex = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)

const maxRequestBodyBytes = 1 << 20
const actorHeaderName = "X-Actor"

type createUserRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

type createTaskRequest struct {
	Title  string `json:"title"`
	Status string `json:"status"`
	UserID *int   `json:"userId"`
}

type updateTaskRequest struct {
	Title  *string `json:"title"`
	Status *string `json:"status"`
	UserID *int    `json:"userId"`
}

// NewServer builds a server instance with routes and middleware.
func NewServer(dataStore Store) *Server {
	if dataStore == nil {
		panic("data store is required")
	}

	s := &Server{
		dataStore: dataStore,
		logger:    log.Default(),
	}

	mux := http.NewServeMux()
	s.setupRoutes(mux)
	s.handler = s.loggingMiddleware(s.recoveryMiddleware(s.corsMiddleware(mux)))

	return s
}

func (s *Server) setupRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/api/users", s.handleUsers)
	mux.HandleFunc("/api/users/", s.handleUserByID)
	mux.HandleFunc("/api/tasks", s.handleTasks)
	mux.HandleFunc("/api/tasks/", s.handleTaskByID)
	mux.HandleFunc("/api/stats", s.handleStats)
}

// Handler returns the fully configured HTTP handler chain.
func (s *Server) Handler() http.Handler {
	return s.handler
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	response := HealthResponse{
		Status:  "ok",
		Message: "Go backend is running",
	}

	s.writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleUsers(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		users, err := s.dataStore.GetUsers()
		if err != nil {
			s.logger.Printf("error loading users: %v", err)
			s.writeError(w, http.StatusInternalServerError, "internal server error")
			return
		}
		response := UsersResponse{
			Users: users,
			Count: len(users),
		}
		s.writeJSON(w, http.StatusOK, response)
	case http.MethodPost:
		s.createUser(w, r)
	default:
		s.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *Server) handleUserByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	id, err := parseIDFromPath(r.URL.Path, "/api/users/")
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid user ID")
		return
	}

	user, ok, err := s.dataStore.GetUserByID(id)
	if err != nil {
		s.logger.Printf("error loading user id=%d: %v", id, err)
		s.writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	if !ok {
		s.writeError(w, http.StatusNotFound, "user not found")
		return
	}

	s.writeJSON(w, http.StatusOK, user)
}

func (s *Server) handleTasks(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		status := r.URL.Query().Get("status")
		userID := r.URL.Query().Get("userId")
		if userID != "" {
			parsedUserID, err := strconv.Atoi(userID)
			if err != nil || parsedUserID <= 0 {
				s.writeError(w, http.StatusBadRequest, "invalid userId query parameter")
				return
			}
		}

		tasks, err := s.dataStore.GetTasks(status, userID)
		if err != nil {
			s.logger.Printf("error loading tasks: %v", err)
			s.writeError(w, http.StatusInternalServerError, "internal server error")
			return
		}
		response := TasksResponse{
			Tasks: tasks,
			Count: len(tasks),
		}

		s.writeJSON(w, http.StatusOK, response)
	case http.MethodPost:
		s.createTask(w, r)
	default:
		s.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *Server) handleTaskByID(w http.ResponseWriter, r *http.Request) {
	if strings.HasSuffix(r.URL.Path, "/history") {
		s.handleTaskHistory(w, r)
		return
	}

	if r.Method != http.MethodPut {
		s.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	taskID, err := parseIDFromPath(r.URL.Path, "/api/tasks/")
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid task ID")
		return
	}

	if err := requireJSONContentType(r); err != nil {
		s.writeError(w, http.StatusUnsupportedMediaType, err.Error())
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)

	var req updateTaskRequest
	if err := decodeJSONBody(r, &req); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			s.writeError(w, http.StatusRequestEntityTooLarge, "request body too large")
			return
		}
		s.writeError(w, http.StatusBadRequest, normalizeJSONError(err))
		return
	}

	if req.Title == nil && req.Status == nil && req.UserID == nil {
		s.writeError(w, http.StatusBadRequest, "at least one field must be provided")
		return
	}

	var update TaskUpdate
	if req.Title != nil {
		title := strings.TrimSpace(*req.Title)
		if title == "" {
			s.writeError(w, http.StatusBadRequest, "title cannot be empty")
			return
		}
		update.Title = &title
	}

	if req.Status != nil {
		status := strings.TrimSpace(*req.Status)
		if !isValidTaskStatus(status) {
			s.writeError(w, http.StatusBadRequest, "invalid status")
			return
		}
		update.Status = &status
	}

	if req.UserID != nil {
		update.UserID = req.UserID
	}

	task, err := s.dataStore.UpdateTask(taskID, update, extractActor(r))
	if err != nil {
		switch {
		case errors.Is(err, ErrTaskNotFound):
			s.writeError(w, http.StatusNotFound, "task not found")
		case errors.Is(err, ErrInvalidTaskStatus), errors.Is(err, ErrUserDoesNotExist):
			s.writeError(w, http.StatusBadRequest, err.Error())
		default:
			s.logger.Printf("error updating task id=%d: %v", taskID, err)
			s.writeError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	s.writeJSON(w, http.StatusOK, task)
}

func (s *Server) handleTaskHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	taskID, err := parseTaskHistoryIDFromPath(r.URL.Path, "/api/tasks/")
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid task ID")
		return
	}

	history, err := s.dataStore.GetTaskHistory(taskID)
	if err != nil {
		if errors.Is(err, ErrTaskNotFound) {
			s.writeError(w, http.StatusNotFound, "task not found")
			return
		}
		s.logger.Printf("error loading task history id=%d: %v", taskID, err)
		s.writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	s.writeJSON(w, http.StatusOK, TaskHistoryResponse{
		TaskID:  taskID,
		History: history,
		Count:   len(history),
	})
}

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	stats, err := s.dataStore.GetStats()
	if err != nil {
		s.logger.Printf("error loading stats: %v", err)
		s.writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	s.writeJSON(w, http.StatusOK, stats)
}

// Start runs the HTTP server on the provided port.
func (s *Server) Start(port string) {
	if port == "" {
		port = defaultPort
	}

	httpServer := &http.Server{
		Addr:              ":" + port,
		Handler:           s.handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	log.Printf("Go backend server starting on http://localhost:%s", port)
	log.Printf("Serving data from PostgreSQL-backed Go backend")

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := s.runWithContext(ctx, httpServer, httpServer.ListenAndServe); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

func (s *Server) runWithContext(ctx context.Context, httpServer *http.Server, serve func() error) error {
	errCh := make(chan error, 1)
	go func() {
		errCh <- serve()
	}()

	select {
	case err := <-errCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	case <-ctx.Done():
		s.logger.Printf("shutdown signal received, shutting down server")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("graceful shutdown failed: %w", err)
		}

		err := <-errCh
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	}
}

func (s *Server) createUser(w http.ResponseWriter, r *http.Request) {
	if err := requireJSONContentType(r); err != nil {
		s.writeError(w, http.StatusUnsupportedMediaType, err.Error())
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)

	var req createUserRequest
	if err := decodeJSONBody(r, &req); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			s.writeError(w, http.StatusRequestEntityTooLarge, "request body too large")
			return
		}
		s.writeError(w, http.StatusBadRequest, normalizeJSONError(err))
		return
	}

	name := strings.TrimSpace(req.Name)
	email := strings.TrimSpace(req.Email)
	role := strings.TrimSpace(req.Role)

	if name == "" || email == "" || role == "" {
		s.writeError(w, http.StatusBadRequest, "name, email, and role are required")
		return
	}
	if !emailRegex.MatchString(email) {
		s.writeError(w, http.StatusBadRequest, "invalid email format")
		return
	}

	user, err := s.dataStore.CreateUser(name, email, role)
	if err != nil {
		s.logger.Printf("error creating user: %v", err)
		s.writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	s.writeJSON(w, http.StatusCreated, user)
}

func (s *Server) createTask(w http.ResponseWriter, r *http.Request) {
	if err := requireJSONContentType(r); err != nil {
		s.writeError(w, http.StatusUnsupportedMediaType, err.Error())
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)

	var req createTaskRequest
	if err := decodeJSONBody(r, &req); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			s.writeError(w, http.StatusRequestEntityTooLarge, "request body too large")
			return
		}
		s.writeError(w, http.StatusBadRequest, normalizeJSONError(err))
		return
	}

	title := strings.TrimSpace(req.Title)
	status := strings.TrimSpace(req.Status)

	if title == "" || status == "" || req.UserID == nil {
		s.writeError(w, http.StatusBadRequest, "title, status, and userId are required")
		return
	}

	if !isValidTaskStatus(status) {
		s.writeError(w, http.StatusBadRequest, "invalid status")
		return
	}

	task, err := s.dataStore.CreateTask(title, status, *req.UserID, extractActor(r))
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidTaskStatus), errors.Is(err, ErrUserDoesNotExist):
			s.writeError(w, http.StatusBadRequest, err.Error())
		default:
			s.logger.Printf("error creating task: %v", err)
			s.writeError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	s.writeJSON(w, http.StatusCreated, task)
}

func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Actor")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,OPTIONS")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Server) recoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				s.logger.Printf("panic recovered method=%s path=%s err=%v", r.Method, r.URL.Path, rec)
				s.writeError(w, http.StatusInternalServerError, "internal server error")
			}
		}()

		next.ServeHTTP(w, r)
	})
}

func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		recorder := &statusRecorder{ResponseWriter: w}

		next.ServeHTTP(recorder, r)

		status := recorder.status
		if status == 0 {
			status = http.StatusOK
		}

		s.logger.Printf(
			"request method=%s path=%s status=%d duration=%s",
			r.Method,
			r.URL.Path,
			status,
			time.Since(start),
		)
	})
}

func (s *Server) writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(payload); err != nil {
		s.logger.Printf("failed to encode JSON response: %v", err)
	}
}

func (s *Server) writeError(w http.ResponseWriter, status int, message string) {
	s.writeJSON(w, status, map[string]string{
		"error": message,
	})
}

func parseIDFromPath(path, prefix string) (int, error) {
	idPart := strings.TrimPrefix(path, prefix)
	if idPart == "" || strings.Contains(idPart, "/") {
		return 0, errors.New("invalid id")
	}

	id, err := strconv.Atoi(idPart)
	if err != nil || id <= 0 {
		return 0, errors.New("invalid id")
	}

	return id, nil
}

func parseTaskHistoryIDFromPath(path, prefix string) (int, error) {
	idPart := strings.TrimPrefix(path, prefix)
	if idPart == "" || !strings.HasSuffix(idPart, "/history") {
		return 0, errors.New("invalid id")
	}
	idPart = strings.TrimSuffix(idPart, "/history")
	if idPart == "" || strings.Contains(idPart, "/") {
		return 0, errors.New("invalid id")
	}

	id, err := strconv.Atoi(idPart)
	if err != nil || id <= 0 {
		return 0, errors.New("invalid id")
	}

	return id, nil
}

func extractActor(r *http.Request) string {
	actor := strings.TrimSpace(r.Header.Get(actorHeaderName))
	if actor == "" {
		return defaultActorName
	}
	return actor
}

func decodeJSONBody(r *http.Request, dst any) error {
	if r.Body == nil {
		return errors.New("request body is required")
	}
	defer r.Body.Close()

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(dst); err != nil {
		if errors.Is(err, io.EOF) {
			return errors.New("request body is required")
		}
		return err
	}

	var extra json.RawMessage
	if err := decoder.Decode(&extra); err != nil {
		if errors.Is(err, io.EOF) {
			return nil
		}
		return err
	}

	return errors.New("request body must contain a single JSON object")
}

func requireJSONContentType(r *http.Request) error {
	contentType := r.Header.Get("Content-Type")
	if contentType == "" {
		return errors.New("content type must be application/json")
	}

	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return errors.New("invalid content type header")
	}
	if mediaType != "application/json" {
		return errors.New("content type must be application/json")
	}

	return nil
}

func normalizeJSONError(err error) string {
	var syntaxErr *json.SyntaxError
	if errors.As(err, &syntaxErr) {
		return "invalid JSON syntax"
	}

	var typeErr *json.UnmarshalTypeError
	if errors.As(err, &typeErr) {
		if typeErr.Field != "" {
			return fmt.Sprintf("invalid type for field %q", typeErr.Field)
		}
		return "invalid JSON field type"
	}

	if strings.HasPrefix(err.Error(), "json: unknown field ") {
		return err.Error()
	}

	return err.Error()
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (sr *statusRecorder) WriteHeader(status int) {
	sr.status = status
	sr.ResponseWriter.WriteHeader(status)
}

func (sr *statusRecorder) Write(p []byte) (int, error) {
	if sr.status == 0 {
		sr.status = http.StatusOK
	}
	return sr.ResponseWriter.Write(p)
}
