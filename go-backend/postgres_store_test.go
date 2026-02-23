package main

import (
	"database/sql"
	"errors"
	"io"
	"log"
	"regexp"
	"strings"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
)

func newMockPostgresStore(t *testing.T) (*PostgresStore, sqlmock.Sqlmock, func()) {
	t.Helper()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}

	store := &PostgresStore{
		db:     db,
		logger: log.New(io.Discard, "", 0),
	}

	cleanup := func() {
		_ = db.Close()
	}

	return store, mock, cleanup
}

func assertMockExpectations(t *testing.T, mock sqlmock.Sqlmock) {
	t.Helper()

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestPostgresStoreCreateUser(t *testing.T) {
	store, mock, cleanup := newMockPostgresStore(t)
	defer cleanup()

	mock.
		ExpectQuery(regexp.QuoteMeta(`
		INSERT INTO users (name, email, role)
		VALUES ($1, $2, $3)
		RETURNING id, name, email, role
	`)).
		WithArgs("Alice", "alice@example.com", "developer").
		WillReturnRows(
			sqlmock.NewRows([]string{"id", "name", "email", "role"}).
				AddRow(4, "Alice", "alice@example.com", "developer"),
		)

	user, err := store.CreateUser("Alice", "alice@example.com", "developer")
	if err != nil {
		t.Fatalf("expected create user to succeed, got %v", err)
	}
	if user.ID != 4 {
		t.Fatalf("expected ID 4, got %d", user.ID)
	}

	assertMockExpectations(t, mock)
}

func TestPostgresStoreCreateUserInsertError(t *testing.T) {
	store, mock, cleanup := newMockPostgresStore(t)
	defer cleanup()

	mock.
		ExpectQuery(regexp.QuoteMeta(`
		INSERT INTO users (name, email, role)
		VALUES ($1, $2, $3)
		RETURNING id, name, email, role
	`)).
		WithArgs("Alice", "alice@example.com", "developer").
		WillReturnError(errors.New("insert failed"))

	_, err := store.CreateUser("Alice", "alice@example.com", "developer")
	if err == nil {
		t.Fatal("expected create user to fail")
	}
	if !strings.Contains(err.Error(), "insert user") {
		t.Fatalf("expected wrapped insert user error, got %v", err)
	}

	assertMockExpectations(t, mock)
}

func TestPostgresStoreCreateTaskValidation(t *testing.T) {
	store, _, cleanup := newMockPostgresStore(t)
	defer cleanup()

	_, err := store.CreateTask("Task", "not-valid", 1, "admin")
	if !errors.Is(err, ErrInvalidTaskStatus) {
		t.Fatalf("expected ErrInvalidTaskStatus, got %v", err)
	}
}

func TestPostgresStoreCreateTaskUnknownUser(t *testing.T) {
	store, mock, cleanup := newMockPostgresStore(t)
	defer cleanup()

	mock.ExpectBegin()
	mock.
		ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM users WHERE id = \$1\)`).
		WithArgs(999).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
	mock.ExpectRollback()

	_, err := store.CreateTask("Task", "pending", 999, "admin")
	if !errors.Is(err, ErrUserDoesNotExist) {
		t.Fatalf("expected ErrUserDoesNotExist, got %v", err)
	}

	assertMockExpectations(t, mock)
}

func TestPostgresStoreCreateTaskSuccess(t *testing.T) {
	store, mock, cleanup := newMockPostgresStore(t)
	defer cleanup()

	mock.ExpectBegin()
	mock.
		ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM users WHERE id = \$1\)`).
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

	mock.
		ExpectQuery(regexp.QuoteMeta(`
		INSERT INTO tasks (title, status, user_id)
		VALUES ($1, $2, $3)
		RETURNING id, title, status, user_id
	`)).
		WithArgs("Task", "pending", 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "title", "status", "user_id"}).AddRow(4, "Task", "pending", 1))
	mock.
		ExpectExec(`INSERT INTO task_history`).
		WithArgs(4, sqlmock.AnyArg(), "admin", "status", nil, "pending").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	task, err := store.CreateTask("Task", "pending", 1, "admin")
	if err != nil {
		t.Fatalf("expected create task to succeed, got %v", err)
	}
	if task.ID != 4 || task.UserID != 1 {
		t.Fatalf("unexpected task response: %+v", task)
	}

	assertMockExpectations(t, mock)
}

func TestPostgresStoreUpdateTaskNotFound(t *testing.T) {
	store, mock, cleanup := newMockPostgresStore(t)
	defer cleanup()

	mock.ExpectBegin()
	mock.
		ExpectQuery(`SELECT id, title, status, user_id`).
		WithArgs(999).
		WillReturnError(sql.ErrNoRows)
	mock.ExpectRollback()

	status := "completed"
	_, err := store.UpdateTask(999, TaskUpdate{Status: &status}, "admin")
	if !errors.Is(err, ErrTaskNotFound) {
		t.Fatalf("expected ErrTaskNotFound, got %v", err)
	}

	assertMockExpectations(t, mock)
}

func TestPostgresStoreUpdateTaskSuccess(t *testing.T) {
	store, mock, cleanup := newMockPostgresStore(t)
	defer cleanup()

	mock.ExpectBegin()
	mock.
		ExpectQuery(`SELECT id, title, status, user_id`).
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "title", "status", "user_id"}).AddRow(1, "Old", "pending", 1))
	mock.
		ExpectExec(`INSERT INTO task_history`).
		WithArgs(1, sqlmock.AnyArg(), "admin", "title", "Old", "Updated").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.
		ExpectExec(`INSERT INTO task_history`).
		WithArgs(1, sqlmock.AnyArg(), "admin", "status", "pending", "completed").
		WillReturnResult(sqlmock.NewResult(2, 1))
	mock.
		ExpectExec(`UPDATE tasks`).
		WithArgs("Updated", "completed", 1, 1).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	title := "Updated"
	status := "completed"
	task, err := store.UpdateTask(1, TaskUpdate{
		Title:  &title,
		Status: &status,
	}, "admin")
	if err != nil {
		t.Fatalf("expected update task to succeed, got %v", err)
	}
	if task.Title != "Updated" || task.Status != "completed" {
		t.Fatalf("unexpected task after update: %+v", task)
	}
	if task.LastChange == nil {
		t.Fatal("expected lastChange metadata after update")
	}
	if task.LastChange.Field != "status" {
		t.Fatalf("expected latest change field=status, got %q", task.LastChange.Field)
	}

	assertMockExpectations(t, mock)
}

func TestPostgresStoreGetUsers(t *testing.T) {
	store, mock, cleanup := newMockPostgresStore(t)
	defer cleanup()

	mock.
		ExpectQuery(`SELECT id, name, email, role`).
		WillReturnRows(
			sqlmock.NewRows([]string{"id", "name", "email", "role"}).
				AddRow(1, "John Doe", "john@example.com", "developer").
				AddRow(2, "Jane Smith", "jane@example.com", "designer"),
		)

	users, err := store.GetUsers()
	if err != nil {
		t.Fatalf("expected get users to succeed, got %v", err)
	}
	if len(users) != 2 {
		t.Fatalf("expected 2 users, got %d", len(users))
	}

	assertMockExpectations(t, mock)
}

func TestPostgresStoreGetUserByIDNotFound(t *testing.T) {
	store, mock, cleanup := newMockPostgresStore(t)
	defer cleanup()

	mock.
		ExpectQuery(`SELECT id, name, email, role`).
		WithArgs(123).
		WillReturnError(sql.ErrNoRows)

	_, ok, err := store.GetUserByID(123)
	if err != nil {
		t.Fatalf("expected get user by ID to return not found without error, got %v", err)
	}
	if ok {
		t.Fatal("expected user lookup to be not found")
	}

	assertMockExpectations(t, mock)
}

func TestPostgresStoreGetTasksInvalidUserFilter(t *testing.T) {
	store, _, cleanup := newMockPostgresStore(t)
	defer cleanup()

	tasks, err := store.GetTasks("", "not-an-int")
	if err != nil {
		t.Fatalf("expected invalid userId filter to return empty result without error, got %v", err)
	}
	if len(tasks) != 0 {
		t.Fatalf("expected 0 tasks for invalid userId filter, got %d", len(tasks))
	}
}

func TestPostgresStoreGetTasksQueryErrorReturnsError(t *testing.T) {
	store, mock, cleanup := newMockPostgresStore(t)
	defer cleanup()

	mock.
		ExpectQuery(`FROM tasks t`).
		WillReturnError(errors.New("query failed"))

	_, err := store.GetTasks("", "")
	if err == nil {
		t.Fatal("expected query error from get tasks")
	}

	assertMockExpectations(t, mock)
}

func TestPostgresStoreGetTasksIncludesLastChange(t *testing.T) {
	store, mock, cleanup := newMockPostgresStore(t)
	defer cleanup()

	mock.
		ExpectQuery(`FROM tasks t`).
		WillReturnRows(
			sqlmock.NewRows([]string{
				"id",
				"title",
				"status",
				"user_id",
				"history_id",
				"changed_at",
				"changed_by",
				"field",
				"from_value",
				"to_value",
			}).AddRow(
				1,
				"Task",
				"in-progress",
				2,
				7,
				time.Date(2026, time.January, 1, 10, 0, 0, 0, time.UTC),
				"admin",
				"status",
				"pending",
				"in-progress",
			),
		)

	tasks, err := store.GetTasks("", "")
	if err != nil {
		t.Fatalf("expected get tasks to succeed, got %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(tasks))
	}
	if tasks[0].LastChange == nil {
		t.Fatal("expected task to include lastChange")
	}
	if tasks[0].LastChange.ChangedBy != "admin" {
		t.Fatalf("expected changedBy admin, got %q", tasks[0].LastChange.ChangedBy)
	}
	if tasks[0].LastChange.Field != "status" {
		t.Fatalf("expected status field, got %q", tasks[0].LastChange.Field)
	}

	assertMockExpectations(t, mock)
}

func TestPostgresStoreGetTaskHistory(t *testing.T) {
	store, mock, cleanup := newMockPostgresStore(t)
	defer cleanup()

	mock.
		ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM tasks WHERE id = \$1\)`).
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	mock.
		ExpectQuery(`SELECT id, task_id, changed_at, changed_by, field, from_value, to_value`).
		WithArgs(1).
		WillReturnRows(
			sqlmock.NewRows([]string{"id", "task_id", "changed_at", "changed_by", "field", "from_value", "to_value"}).
				AddRow(11, 1, time.Date(2026, time.January, 2, 10, 0, 0, 0, time.UTC), "admin", "status", "pending", "in-progress"),
		)

	history, err := store.GetTaskHistory(1)
	if err != nil {
		t.Fatalf("expected get task history to succeed, got %v", err)
	}
	if len(history) != 1 {
		t.Fatalf("expected 1 history entry, got %d", len(history))
	}
	if history[0].ChangedBy != "admin" {
		t.Fatalf("expected changedBy admin, got %q", history[0].ChangedBy)
	}

	assertMockExpectations(t, mock)
}

func TestPostgresStoreGetTaskHistoryNotFound(t *testing.T) {
	store, mock, cleanup := newMockPostgresStore(t)
	defer cleanup()

	mock.
		ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM tasks WHERE id = \$1\)`).
		WithArgs(99).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

	_, err := store.GetTaskHistory(99)
	if !errors.Is(err, ErrTaskNotFound) {
		t.Fatalf("expected ErrTaskNotFound, got %v", err)
	}

	assertMockExpectations(t, mock)
}

func TestPostgresStoreGetStats(t *testing.T) {
	store, mock, cleanup := newMockPostgresStore(t)
	defer cleanup()

	mock.
		ExpectQuery(`SELECT COUNT\(\*\)`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(3))

	mock.
		ExpectQuery(`SELECT\s+COUNT\(\*\) AS total`).
		WillReturnRows(sqlmock.NewRows([]string{"total", "pending", "in_progress", "completed"}).AddRow(5, 2, 1, 2))

	stats, err := store.GetStats()
	if err != nil {
		t.Fatalf("expected get stats to succeed, got %v", err)
	}
	if stats.Users.Total != 3 || stats.Tasks.Total != 5 {
		t.Fatalf("unexpected stats response: %+v", stats)
	}
	if stats.Tasks.Pending != 2 || stats.Tasks.InProgress != 1 || stats.Tasks.Completed != 2 {
		t.Fatalf("unexpected task status distribution: %+v", stats.Tasks)
	}

	assertMockExpectations(t, mock)
}

func TestPostgresStoreGetStatsUserQueryErrorReturnsError(t *testing.T) {
	store, mock, cleanup := newMockPostgresStore(t)
	defer cleanup()

	mock.
		ExpectQuery(`SELECT COUNT\(\*\)`).
		WillReturnError(errors.New("stats query failed"))

	_, err := store.GetStats()
	if err == nil {
		t.Fatal("expected query error from get stats")
	}

	assertMockExpectations(t, mock)
}

func TestPostgresStoreUpdateTaskInvalidStatus(t *testing.T) {
	store, _, cleanup := newMockPostgresStore(t)
	defer cleanup()

	status := "not-valid"
	_, err := store.UpdateTask(1, TaskUpdate{Status: &status}, "admin")
	if !errors.Is(err, ErrInvalidTaskStatus) {
		t.Fatalf("expected ErrInvalidTaskStatus, got %v", err)
	}
}

func TestPostgresStoreInitSchema(t *testing.T) {
	store, mock, cleanup := newMockPostgresStore(t)
	defer cleanup()

	mock.ExpectExec(`CREATE TABLE IF NOT EXISTS users`).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(`CREATE TABLE IF NOT EXISTS tasks`).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(`CREATE TABLE IF NOT EXISTS task_history`).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(`CREATE INDEX IF NOT EXISTS idx_tasks_status`).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(`CREATE INDEX IF NOT EXISTS idx_tasks_user_id`).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(`CREATE INDEX IF NOT EXISTS idx_task_history_task_id`).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(`CREATE INDEX IF NOT EXISTS idx_task_history_changed_at`).WillReturnResult(sqlmock.NewResult(0, 0))

	if err := store.initSchema(); err != nil {
		t.Fatalf("expected init schema to succeed, got %v", err)
	}

	assertMockExpectations(t, mock)
}

func TestPostgresStoreUpdateTaskUnknownUser(t *testing.T) {
	store, mock, cleanup := newMockPostgresStore(t)
	defer cleanup()

	mock.ExpectBegin()
	mock.
		ExpectQuery(`SELECT id, title, status, user_id`).
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "title", "status", "user_id"}).AddRow(1, "Old", "pending", 1))
	mock.
		ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM users WHERE id = \$1\)`).
		WithArgs(999).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
	mock.ExpectRollback()

	newUserID := 999
	_, err := store.UpdateTask(1, TaskUpdate{
		UserID: &newUserID,
	}, "admin")
	if !errors.Is(err, ErrUserDoesNotExist) {
		t.Fatalf("expected ErrUserDoesNotExist, got %v", err)
	}

	assertMockExpectations(t, mock)
}

func TestPostgresStoreSeedInitialDataOnEmptyTables(t *testing.T) {
	store, mock, cleanup := newMockPostgresStore(t)
	defer cleanup()

	mock.ExpectBegin()

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM users`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	for _, user := range initialUsers {
		mock.ExpectExec(`INSERT INTO users`).
			WithArgs(user.ID, user.Name, user.Email, user.Role).
			WillReturnResult(sqlmock.NewResult(0, 1))
	}
	mock.ExpectExec(`SELECT setval\(`).WillReturnResult(sqlmock.NewResult(0, 1))

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM tasks`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	for _, task := range initialTasks {
		mock.ExpectExec(`INSERT INTO tasks`).
			WithArgs(task.ID, task.Title, task.Status, task.UserID).
			WillReturnResult(sqlmock.NewResult(0, 1))
	}
	mock.ExpectExec(`SELECT setval\(`).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM task_history`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectExec(`INSERT INTO task_history`).
		WithArgs(defaultActorName).
		WillReturnResult(sqlmock.NewResult(0, int64(len(initialTasks))))

	mock.ExpectCommit()

	if err := store.seedInitialData(); err != nil {
		t.Fatalf("expected seed initial data to succeed, got %v", err)
	}

	assertMockExpectations(t, mock)
}

func TestPostgresStoreSeedInitialDataSkipsWhenTablesPopulated(t *testing.T) {
	store, mock, cleanup := newMockPostgresStore(t)
	defer cleanup()

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM users`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM tasks`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM task_history`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectCommit()

	if err := store.seedInitialData(); err != nil {
		t.Fatalf("expected seed initial data to succeed with existing data, got %v", err)
	}

	assertMockExpectations(t, mock)
}
