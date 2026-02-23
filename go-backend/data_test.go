package main

import (
	"errors"
	"sync"
	"testing"
)

func TestDataStoreGetUsersReturnsCopy(t *testing.T) {
	ds := NewDataStore([]User{
		{ID: 1, Name: "Alice", Email: "alice@example.com", Role: "developer"},
	}, nil)

	users, err := ds.GetUsers()
	if err != nil {
		t.Fatalf("expected get users to succeed, got %v", err)
	}
	users[0].Name = "Mutated"

	user, ok, err := ds.GetUserByID(1)
	if err != nil {
		t.Fatalf("expected get user by ID to succeed, got %v", err)
	}
	if !ok {
		t.Fatal("expected user to exist")
	}
	if user.Name != "Alice" {
		t.Fatalf("expected datastore user to remain unchanged, got %q", user.Name)
	}
}

func TestDataStoreCreateUserAssignsIncrementingID(t *testing.T) {
	ds := NewDataStore([]User{
		{ID: 10, Name: "Alice", Email: "alice@example.com", Role: "developer"},
	}, nil)

	user1, err := ds.CreateUser("Bob", "bob@example.com", "designer")
	if err != nil {
		t.Fatalf("expected first create user to succeed, got %v", err)
	}
	user2, err := ds.CreateUser("Carol", "carol@example.com", "manager")
	if err != nil {
		t.Fatalf("expected second create user to succeed, got %v", err)
	}

	if user1.ID != 11 {
		t.Fatalf("expected first created user ID 11, got %d", user1.ID)
	}
	if user2.ID != 12 {
		t.Fatalf("expected second created user ID 12, got %d", user2.ID)
	}
}

func TestDataStoreCreateTaskValidation(t *testing.T) {
	ds := NewDataStore([]User{
		{ID: 1, Name: "Alice", Email: "alice@example.com", Role: "developer"},
	}, nil)

	if _, err := ds.CreateTask("Task 1", "invalid", 1, "admin"); !errors.Is(err, ErrInvalidTaskStatus) {
		t.Fatalf("expected ErrInvalidTaskStatus, got %v", err)
	}

	if _, err := ds.CreateTask("Task 1", "pending", 999, "admin"); !errors.Is(err, ErrUserDoesNotExist) {
		t.Fatalf("expected ErrUserDoesNotExist, got %v", err)
	}

	task, err := ds.CreateTask("Task 1", "pending", 1, "admin")
	if err != nil {
		t.Fatalf("expected successful task creation, got %v", err)
	}
	if task.ID != 1 {
		t.Fatalf("expected task ID 1, got %d", task.ID)
	}
	if task.LastChange == nil {
		t.Fatal("expected task to include last change")
	}
	if task.LastChange.ChangedBy != "admin" {
		t.Fatalf("expected changedBy admin, got %q", task.LastChange.ChangedBy)
	}
}

func TestDataStoreUpdateTaskPartial(t *testing.T) {
	ds := NewDataStore(
		[]User{
			{ID: 1, Name: "Alice", Email: "alice@example.com", Role: "developer"},
			{ID: 2, Name: "Bob", Email: "bob@example.com", Role: "manager"},
		},
		[]Task{
			{ID: 1, Title: "Original", Status: "pending", UserID: 1},
		},
	)

	newStatus := "completed"
	newUserID := 2
	updated, err := ds.UpdateTask(1, TaskUpdate{
		Status: &newStatus,
		UserID: &newUserID,
	}, "qa-user")
	if err != nil {
		t.Fatalf("expected successful update, got %v", err)
	}

	if updated.Status != "completed" {
		t.Fatalf("expected status completed, got %s", updated.Status)
	}
	if updated.UserID != 2 {
		t.Fatalf("expected userId 2, got %d", updated.UserID)
	}
	if updated.Title != "Original" {
		t.Fatalf("expected title to remain unchanged, got %s", updated.Title)
	}
	if updated.LastChange == nil {
		t.Fatal("expected last change metadata after update")
	}
	if updated.LastChange.ChangedBy != "qa-user" {
		t.Fatalf("expected changedBy qa-user, got %q", updated.LastChange.ChangedBy)
	}

	if _, err := ds.UpdateTask(999, TaskUpdate{Status: &newStatus}, "qa-user"); !errors.Is(err, ErrTaskNotFound) {
		t.Fatalf("expected ErrTaskNotFound, got %v", err)
	}
}

func TestDataStoreGetTaskHistory(t *testing.T) {
	ds := NewDataStore(
		[]User{
			{ID: 1, Name: "Alice", Email: "alice@example.com", Role: "developer"},
		},
		[]Task{
			{ID: 1, Title: "Original", Status: "pending", UserID: 1},
		},
	)

	status1 := "in-progress"
	status2 := "completed"
	if _, err := ds.UpdateTask(1, TaskUpdate{Status: &status1}, "alice"); err != nil {
		t.Fatalf("expected first update to succeed, got %v", err)
	}
	if _, err := ds.UpdateTask(1, TaskUpdate{Status: &status2}, "bob"); err != nil {
		t.Fatalf("expected second update to succeed, got %v", err)
	}

	history, err := ds.GetTaskHistory(1)
	if err != nil {
		t.Fatalf("expected task history lookup to succeed, got %v", err)
	}
	if len(history) != 2 {
		t.Fatalf("expected 2 history entries, got %d", len(history))
	}
	if history[0].ChangedBy != "bob" {
		t.Fatalf("expected most recent history entry by bob, got %q", history[0].ChangedBy)
	}
	if history[0].Field != "status" {
		t.Fatalf("expected status field change, got %q", history[0].Field)
	}
	if history[0].FromValue == nil || *history[0].FromValue != "in-progress" {
		t.Fatalf("unexpected fromValue in history: %+v", history[0].FromValue)
	}

	if _, err := ds.GetTaskHistory(999); !errors.Is(err, ErrTaskNotFound) {
		t.Fatalf("expected ErrTaskNotFound for unknown task, got %v", err)
	}
}

func TestDataStoreConcurrentCreateUserIDsAreUnique(t *testing.T) {
	ds := NewDataStore(nil, nil)

	const total = 100
	var wg sync.WaitGroup
	wg.Add(total)

	ids := make(chan int, total)
	for i := 0; i < total; i++ {
		go func(idx int) {
			defer wg.Done()
			user, err := ds.CreateUser("User", "user@example.com", "developer")
			if err != nil {
				t.Errorf("expected create user to succeed, got %v", err)
				return
			}
			ids <- user.ID
		}(i)
	}

	wg.Wait()
	close(ids)

	seen := make(map[int]struct{}, total)
	for id := range ids {
		if _, exists := seen[id]; exists {
			t.Fatalf("duplicate ID generated: %d", id)
		}
		seen[id] = struct{}{}
	}

	if len(seen) != total {
		t.Fatalf("expected %d unique IDs, got %d", total, len(seen))
	}
}

func TestDataStoreGetTasksFilters(t *testing.T) {
	ds := NewDataStore(
		[]User{
			{ID: 1, Name: "Alice", Email: "alice@example.com", Role: "developer"},
			{ID: 2, Name: "Bob", Email: "bob@example.com", Role: "manager"},
		},
		[]Task{
			{ID: 1, Title: "T1", Status: "pending", UserID: 1},
			{ID: 2, Title: "T2", Status: "completed", UserID: 1},
			{ID: 3, Title: "T3", Status: "pending", UserID: 2},
		},
	)

	all, err := ds.GetTasks("", "")
	if err != nil {
		t.Fatalf("expected get tasks to succeed, got %v", err)
	}
	if len(all) != 3 {
		t.Fatalf("expected 3 tasks, got %d", len(all))
	}

	pending, err := ds.GetTasks("pending", "")
	if err != nil {
		t.Fatalf("expected get tasks with status to succeed, got %v", err)
	}
	if len(pending) != 2 {
		t.Fatalf("expected 2 pending tasks, got %d", len(pending))
	}

	userOneTasks, err := ds.GetTasks("", "1")
	if err != nil {
		t.Fatalf("expected get tasks with user filter to succeed, got %v", err)
	}
	if len(userOneTasks) != 2 {
		t.Fatalf("expected 2 tasks for user 1, got %d", len(userOneTasks))
	}

	invalidUserID, err := ds.GetTasks("", "not-an-int")
	if err != nil {
		t.Fatalf("expected invalid userId filter to return empty result without error, got %v", err)
	}
	if len(invalidUserID) != 0 {
		t.Fatalf("expected 0 tasks for invalid user filter, got %d", len(invalidUserID))
	}
}

func TestDataStoreGetStats(t *testing.T) {
	ds := NewDataStore(
		[]User{
			{ID: 1, Name: "Alice", Email: "alice@example.com", Role: "developer"},
			{ID: 2, Name: "Bob", Email: "bob@example.com", Role: "manager"},
		},
		[]Task{
			{ID: 1, Title: "T1", Status: "pending", UserID: 1},
			{ID: 2, Title: "T2", Status: "in-progress", UserID: 2},
			{ID: 3, Title: "T3", Status: "completed", UserID: 2},
		},
	)

	stats, err := ds.GetStats()
	if err != nil {
		t.Fatalf("expected get stats to succeed, got %v", err)
	}
	if stats.Users.Total != 2 {
		t.Fatalf("expected 2 users, got %d", stats.Users.Total)
	}
	if stats.Tasks.Total != 3 {
		t.Fatalf("expected 3 tasks, got %d", stats.Tasks.Total)
	}
	if stats.Tasks.Pending != 1 || stats.Tasks.InProgress != 1 || stats.Tasks.Completed != 1 {
		t.Fatalf(
			"unexpected status counts: pending=%d inProgress=%d completed=%d",
			stats.Tasks.Pending,
			stats.Tasks.InProgress,
			stats.Tasks.Completed,
		)
	}
}
