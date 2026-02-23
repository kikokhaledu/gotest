package main

import (
	"log"
	"os"
	"strings"
	"time"
)

const (
	defaultPort = "8080"
)

// User represents an application user.
type User struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

// Task represents a work item assigned to a user.
type Task struct {
	ID         int              `json:"id"`
	Title      string           `json:"title"`
	Status     string           `json:"status"`
	UserID     int              `json:"userId"`
	LastChange *TaskHistoryItem `json:"lastChange,omitempty"`
}

// TaskHistoryItem captures a single mutation event for a task.
type TaskHistoryItem struct {
	ID        int       `json:"id"`
	TaskID    int       `json:"taskId"`
	ChangedAt time.Time `json:"changedAt"`
	ChangedBy string    `json:"changedBy"`
	Field     string    `json:"field"`
	FromValue *string   `json:"fromValue,omitempty"`
	ToValue   string    `json:"toValue"`
}

// TaskHistoryResponse is the envelope for task audit history.
type TaskHistoryResponse struct {
	TaskID  int               `json:"taskId"`
	History []TaskHistoryItem `json:"history"`
	Count   int               `json:"count"`
}

// UsersResponse is the envelope for the users collection endpoint.
type UsersResponse struct {
	Users []User `json:"users"`
	Count int    `json:"count"`
}

// TasksResponse is the envelope for the tasks collection endpoint.
type TasksResponse struct {
	Tasks []Task `json:"tasks"`
	Count int    `json:"count"`
}

// StatsResponse contains aggregate counts for users and tasks.
type StatsResponse struct {
	Users struct {
		Total int `json:"total"`
	} `json:"users"`
	Tasks struct {
		Total      int `json:"total"`
		Pending    int `json:"pending"`
		InProgress int `json:"inProgress"`
		Completed  int `json:"completed"`
	} `json:"tasks"`
}

// HealthResponse is returned by the health endpoint.
type HealthResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

func main() {
	// Get port from environment or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	postgresDSN := strings.TrimSpace(os.Getenv("POSTGRES_DSN"))
	if postgresDSN == "" {
		log.Fatal("POSTGRES_DSN is required (no in-memory fallback is configured)")
	}

	postgresStore, err := NewPostgresStore(postgresDSN)
	if err != nil {
		log.Fatalf("failed to initialize postgres store: %v", err)
	}
	defer func() {
		if closeErr := postgresStore.Close(); closeErr != nil {
			log.Printf("error closing postgres store: %v", closeErr)
		}
	}()

	server := NewServer(postgresStore)
	server.Start(port)
}
