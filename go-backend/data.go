package main

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	// ErrTaskNotFound is returned when updating a non-existent task.
	ErrTaskNotFound = errors.New("task not found")
	// ErrInvalidTaskStatus is returned when task status is not one of allowed values.
	ErrInvalidTaskStatus = errors.New("invalid task status")
	// ErrUserDoesNotExist is returned when a task references an unknown user.
	ErrUserDoesNotExist = errors.New("user does not exist")
)

const defaultActorName = "system"

// Store defines data access methods used by HTTP handlers.
type Store interface {
	GetUsers() ([]User, error)
	GetUserByID(id int) (User, bool, error)
	GetTasks(status, userID string) ([]Task, error)
	GetTaskHistory(taskID int) ([]TaskHistoryItem, error)
	GetStats() (StatsResponse, error)
	CreateUser(name, email, role string) (User, error)
	CreateTask(title, status string, userID int, actor string) (Task, error)
	UpdateTask(id int, update TaskUpdate, actor string) (Task, error)
}

// TaskUpdate represents patch semantics for task updates.
type TaskUpdate struct {
	Title  *string
	Status *string
	UserID *int
}

// DataStore holds all application data in memory.
type DataStore struct {
	mu          sync.RWMutex
	users       []User
	tasks       []Task
	taskHistory map[int][]TaskHistoryItem
	nextUserID  int
	nextTaskID  int
	nextHistID  int
}

var initialUsers = []User{
	{ID: 1, Name: "John Doe", Email: "john@example.com", Role: "developer"},
	{ID: 2, Name: "Jane Smith", Email: "jane@example.com", Role: "designer"},
	{ID: 3, Name: "Bob Johnson", Email: "bob@example.com", Role: "manager"},
}

var initialTasks = []Task{
	{ID: 1, Title: "Implement authentication", Status: "pending", UserID: 1},
	{ID: 2, Title: "Design user interface", Status: "in-progress", UserID: 2},
	{ID: 3, Title: "Review code changes", Status: "completed", UserID: 3},
}

// NewDataStore initializes a thread-safe in-memory store.
func NewDataStore(users []User, tasks []Task) *DataStore {
	userCopy := copyUsers(users)
	taskCopy := copyTasks(tasks)
	taskHistory := make(map[int][]TaskHistoryItem, len(taskCopy))
	for _, task := range taskCopy {
		taskHistory[task.ID] = []TaskHistoryItem{}
	}
	return &DataStore{
		users:       userCopy,
		tasks:       taskCopy,
		taskHistory: taskHistory,
		nextUserID:  nextUserID(userCopy),
		nextTaskID:  nextTaskID(taskCopy),
		nextHistID:  1,
	}
}

func (ds *DataStore) GetUsers() ([]User, error) {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	return copyUsers(ds.users), nil
}

func (ds *DataStore) GetUserByID(id int) (User, bool, error) {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	for _, user := range ds.users {
		if user.ID == id {
			return user, true, nil
		}
	}

	return User{}, false, nil
}

func (ds *DataStore) GetTasks(status, userID string) ([]Task, error) {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	filterByUser := false
	parsedUserID := 0
	if userID != "" {
		id, err := strconv.Atoi(userID)
		if err != nil {
			return []Task{}, nil
		}
		filterByUser = true
		parsedUserID = id
	}

	filtered := make([]Task, 0, len(ds.tasks))
	for _, task := range ds.tasks {
		if status != "" && task.Status != status {
			continue
		}
		if filterByUser && task.UserID != parsedUserID {
			continue
		}

		filtered = append(filtered, copyTask(task))
	}

	return filtered, nil
}

func (ds *DataStore) GetTaskHistory(taskID int) ([]TaskHistoryItem, error) {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	if !ds.taskExistsLocked(taskID) {
		return nil, fmt.Errorf("%w: %d", ErrTaskNotFound, taskID)
	}

	history := copyTaskHistory(ds.taskHistory[taskID])
	for left, right := 0, len(history)-1; left < right; left, right = left+1, right-1 {
		history[left], history[right] = history[right], history[left]
	}
	return history, nil
}

func (ds *DataStore) GetStats() (StatsResponse, error) {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	var stats StatsResponse
	stats.Users.Total = len(ds.users)
	stats.Tasks.Total = len(ds.tasks)

	for _, task := range ds.tasks {
		switch task.Status {
		case "pending":
			stats.Tasks.Pending++
		case "in-progress":
			stats.Tasks.InProgress++
		case "completed":
			stats.Tasks.Completed++
		}
	}

	return stats, nil
}

func (ds *DataStore) CreateUser(name, email, role string) (User, error) {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	user := User{
		ID:    ds.nextUserID,
		Name:  name,
		Email: email,
		Role:  role,
	}
	ds.nextUserID++
	ds.users = append(ds.users, user)

	return user, nil
}

func (ds *DataStore) CreateTask(title, status string, userID int, actor string) (Task, error) {
	if !isValidTaskStatus(status) {
		return Task{}, fmt.Errorf("%w: %q", ErrInvalidTaskStatus, status)
	}

	ds.mu.Lock()
	defer ds.mu.Unlock()

	if !ds.userExistsLocked(userID) {
		return Task{}, fmt.Errorf("%w: %d", ErrUserDoesNotExist, userID)
	}

	task := Task{
		ID:     ds.nextTaskID,
		Title:  title,
		Status: status,
		UserID: userID,
	}
	ds.nextTaskID++
	history := ds.appendHistoryLocked(
		task.ID,
		normalizeActor(actor),
		"status",
		nil,
		status,
		time.Now().UTC(),
	)
	task.LastChange = &history
	ds.tasks = append(ds.tasks, task)

	return copyTask(task), nil
}

func (ds *DataStore) UpdateTask(id int, update TaskUpdate, actor string) (Task, error) {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	idx := -1
	for i := range ds.tasks {
		if ds.tasks[i].ID == id {
			idx = i
			break
		}
	}
	if idx == -1 {
		return Task{}, fmt.Errorf("%w: %d", ErrTaskNotFound, id)
	}

	if update.Status != nil && !isValidTaskStatus(*update.Status) {
		return Task{}, fmt.Errorf("%w: %q", ErrInvalidTaskStatus, *update.Status)
	}
	if update.UserID != nil && !ds.userExistsLocked(*update.UserID) {
		return Task{}, fmt.Errorf("%w: %d", ErrUserDoesNotExist, *update.UserID)
	}

	var latestChange *TaskHistoryItem
	now := time.Now().UTC()
	normalizedActor := normalizeActor(actor)

	if update.Title != nil {
		if ds.tasks[idx].Title != *update.Title {
			fromValue := ds.tasks[idx].Title
			change := ds.appendHistoryLocked(id, normalizedActor, "title", &fromValue, *update.Title, now)
			latestChange = &change
		}
		ds.tasks[idx].Title = *update.Title
	}
	if update.Status != nil {
		if ds.tasks[idx].Status != *update.Status {
			fromValue := ds.tasks[idx].Status
			change := ds.appendHistoryLocked(id, normalizedActor, "status", &fromValue, *update.Status, now)
			latestChange = &change
		}
		ds.tasks[idx].Status = *update.Status
	}
	if update.UserID != nil {
		if ds.tasks[idx].UserID != *update.UserID {
			fromValue := strconv.Itoa(ds.tasks[idx].UserID)
			toValue := strconv.Itoa(*update.UserID)
			change := ds.appendHistoryLocked(id, normalizedActor, "userId", &fromValue, toValue, now)
			latestChange = &change
		}
		ds.tasks[idx].UserID = *update.UserID
	}
	if latestChange != nil {
		ds.tasks[idx].LastChange = latestChange
	}

	return copyTask(ds.tasks[idx]), nil
}

func (ds *DataStore) userExistsLocked(id int) bool {
	for _, user := range ds.users {
		if user.ID == id {
			return true
		}
	}

	return false
}

func (ds *DataStore) taskExistsLocked(id int) bool {
	for _, task := range ds.tasks {
		if task.ID == id {
			return true
		}
	}
	return false
}

func (ds *DataStore) appendHistoryLocked(
	taskID int,
	actor string,
	field string,
	fromValue *string,
	toValue string,
	changedAt time.Time,
) TaskHistoryItem {
	entry := TaskHistoryItem{
		ID:        ds.nextHistID,
		TaskID:    taskID,
		ChangedAt: changedAt,
		ChangedBy: actor,
		Field:     field,
		FromValue: copyStringPtr(fromValue),
		ToValue:   toValue,
	}
	ds.nextHistID++
	ds.taskHistory[taskID] = append(ds.taskHistory[taskID], entry)
	return entry
}

func normalizeActor(actor string) string {
	trimmed := strings.TrimSpace(actor)
	if trimmed == "" {
		return defaultActorName
	}
	return trimmed
}

func isValidTaskStatus(status string) bool {
	switch status {
	case "pending", "in-progress", "completed":
		return true
	default:
		return false
	}
}

func nextUserID(users []User) int {
	maxID := 0
	for _, user := range users {
		if user.ID > maxID {
			maxID = user.ID
		}
	}

	return maxID + 1
}

func nextTaskID(tasks []Task) int {
	maxID := 0
	for _, task := range tasks {
		if task.ID > maxID {
			maxID = task.ID
		}
	}

	return maxID + 1
}

func copyUsers(users []User) []User {
	out := make([]User, len(users))
	copy(out, users)
	return out
}

func copyTasks(tasks []Task) []Task {
	out := make([]Task, len(tasks))
	for idx, task := range tasks {
		out[idx] = copyTask(task)
	}
	return out
}

func copyTask(task Task) Task {
	copied := task
	if task.LastChange != nil {
		historyCopy := *task.LastChange
		historyCopy.FromValue = copyStringPtr(task.LastChange.FromValue)
		copied.LastChange = &historyCopy
	}
	return copied
}

func copyTaskHistory(history []TaskHistoryItem) []TaskHistoryItem {
	out := make([]TaskHistoryItem, len(history))
	for idx, entry := range history {
		out[idx] = entry
		out[idx].FromValue = copyStringPtr(entry.FromValue)
	}
	return out
}

func copyStringPtr(value *string) *string {
	if value == nil {
		return nil
	}
	copied := *value
	return &copied
}
