package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

const (
	dbOperationTimeout = 3 * time.Second
	dbPingRetries      = 20
)

// PostgresStore persists users/tasks in PostgreSQL.
type PostgresStore struct {
	db     *sql.DB
	logger *log.Logger
}

// NewPostgresStore initializes the PostgreSQL store, schema, and seed data.
func NewPostgresStore(dsn string) (*PostgresStore, error) {
	if strings.TrimSpace(dsn) == "" {
		return nil, errors.New("POSTGRES_DSN is required")
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("open postgres connection: %w", err)
	}

	db.SetConnMaxIdleTime(5 * time.Minute)
	db.SetConnMaxLifetime(30 * time.Minute)
	db.SetMaxIdleConns(5)
	db.SetMaxOpenConns(20)

	if err := pingWithRetry(db); err != nil {
		_ = db.Close()
		return nil, err
	}

	ps := &PostgresStore{
		db:     db,
		logger: log.Default(),
	}

	if err := ps.initSchema(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("initialize schema: %w", err)
	}

	if err := ps.seedInitialData(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("seed initial data: %w", err)
	}

	return ps, nil
}

// Close releases database resources.
func (ps *PostgresStore) Close() error {
	return ps.db.Close()
}

func (ps *PostgresStore) GetUsers() ([]User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), dbOperationTimeout)
	defer cancel()

	rows, err := ps.db.QueryContext(ctx, `
		SELECT id, name, email, role
		FROM users
		ORDER BY id
	`)
	if err != nil {
		ps.logger.Printf("error querying users: %v", err)
		return nil, fmt.Errorf("query users: %w", err)
	}
	defer rows.Close()

	users := make([]User, 0)
	for rows.Next() {
		var user User
		if err := rows.Scan(&user.ID, &user.Name, &user.Email, &user.Role); err != nil {
			ps.logger.Printf("error scanning user row: %v", err)
			return nil, fmt.Errorf("scan users row: %w", err)
		}
		users = append(users, user)
	}
	if err := rows.Err(); err != nil {
		ps.logger.Printf("error iterating user rows: %v", err)
		return nil, fmt.Errorf("iterate users rows: %w", err)
	}

	return users, nil
}

func (ps *PostgresStore) GetUserByID(id int) (User, bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), dbOperationTimeout)
	defer cancel()

	var user User
	err := ps.db.QueryRowContext(ctx, `
		SELECT id, name, email, role
		FROM users
		WHERE id = $1
	`, id).Scan(&user.ID, &user.Name, &user.Email, &user.Role)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return User{}, false, nil
		}
		ps.logger.Printf("error querying user id=%d: %v", id, err)
		return User{}, false, fmt.Errorf("query user by id=%d: %w", id, err)
	}

	return user, true, nil
}

func (ps *PostgresStore) GetTasks(status, userID string) ([]Task, error) {
	var (
		clauses []string
		args    []any
	)

	if status != "" {
		args = append(args, status)
		clauses = append(clauses, fmt.Sprintf("status = $%d", len(args)))
	}

	if userID != "" {
		parsedUserID, err := strconv.Atoi(userID)
		if err != nil {
			return []Task{}, nil
		}
		args = append(args, parsedUserID)
		clauses = append(clauses, fmt.Sprintf("user_id = $%d", len(args)))
	}

	query := `
		SELECT
			t.id,
			t.title,
			t.status,
			t.user_id,
			h.id,
			h.changed_at,
			h.changed_by,
			h.field,
			h.from_value,
			h.to_value
		FROM tasks t
		LEFT JOIN LATERAL (
			SELECT id, changed_at, changed_by, field, from_value, to_value
			FROM task_history
			WHERE task_id = t.id
			ORDER BY changed_at DESC, id DESC
			LIMIT 1
		) h ON true
	`
	if len(clauses) > 0 {
		query += " WHERE " + strings.Join(clauses, " AND ")
	}
	query += " ORDER BY t.id"

	ctx, cancel := context.WithTimeout(context.Background(), dbOperationTimeout)
	defer cancel()

	rows, err := ps.db.QueryContext(ctx, query, args...)
	if err != nil {
		ps.logger.Printf("error querying tasks: %v", err)
		return nil, fmt.Errorf("query tasks: %w", err)
	}
	defer rows.Close()

	tasks := make([]Task, 0)
	for rows.Next() {
		var (
			task      Task
			changeID  sql.NullInt64
			changedAt sql.NullTime
			changedBy sql.NullString
			field     sql.NullString
			fromValue sql.NullString
			toValue   sql.NullString
		)
		if err := rows.Scan(
			&task.ID,
			&task.Title,
			&task.Status,
			&task.UserID,
			&changeID,
			&changedAt,
			&changedBy,
			&field,
			&fromValue,
			&toValue,
		); err != nil {
			ps.logger.Printf("error scanning task row: %v", err)
			return nil, fmt.Errorf("scan tasks row: %w", err)
		}
		if changeID.Valid {
			entry := TaskHistoryItem{
				ID:        int(changeID.Int64),
				TaskID:    task.ID,
				ChangedAt: changedAt.Time,
				ChangedBy: changedBy.String,
				Field:     field.String,
				ToValue:   toValue.String,
			}
			if fromValue.Valid {
				from := fromValue.String
				entry.FromValue = &from
			}
			task.LastChange = &entry
		}
		tasks = append(tasks, task)
	}
	if err := rows.Err(); err != nil {
		ps.logger.Printf("error iterating task rows: %v", err)
		return nil, fmt.Errorf("iterate tasks rows: %w", err)
	}

	return tasks, nil
}

func (ps *PostgresStore) GetTaskHistory(taskID int) ([]TaskHistoryItem, error) {
	ctx, cancel := context.WithTimeout(context.Background(), dbOperationTimeout)
	defer cancel()

	var exists bool
	if err := ps.db.QueryRowContext(ctx, `
		SELECT EXISTS(SELECT 1 FROM tasks WHERE id = $1)
	`, taskID).Scan(&exists); err != nil {
		return nil, fmt.Errorf("check task existence: %w", err)
	}
	if !exists {
		return nil, fmt.Errorf("%w: %d", ErrTaskNotFound, taskID)
	}

	rows, err := ps.db.QueryContext(ctx, `
		SELECT id, task_id, changed_at, changed_by, field, from_value, to_value
		FROM task_history
		WHERE task_id = $1
		ORDER BY changed_at DESC, id DESC
	`, taskID)
	if err != nil {
		return nil, fmt.Errorf("query task history: %w", err)
	}
	defer rows.Close()

	history := make([]TaskHistoryItem, 0)
	for rows.Next() {
		var (
			entry     TaskHistoryItem
			fromValue sql.NullString
		)
		if err := rows.Scan(
			&entry.ID,
			&entry.TaskID,
			&entry.ChangedAt,
			&entry.ChangedBy,
			&entry.Field,
			&fromValue,
			&entry.ToValue,
		); err != nil {
			return nil, fmt.Errorf("scan task history row: %w", err)
		}
		if fromValue.Valid {
			from := fromValue.String
			entry.FromValue = &from
		}
		history = append(history, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate task history rows: %w", err)
	}

	return history, nil
}

func (ps *PostgresStore) GetStats() (StatsResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), dbOperationTimeout)
	defer cancel()

	var stats StatsResponse

	if err := ps.db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM users
	`).Scan(&stats.Users.Total); err != nil {
		ps.logger.Printf("error querying user stats: %v", err)
		return StatsResponse{}, fmt.Errorf("query user stats: %w", err)
	}

	if err := ps.db.QueryRowContext(ctx, `
		SELECT
			COUNT(*) AS total,
			COUNT(*) FILTER (WHERE status = 'pending') AS pending,
			COUNT(*) FILTER (WHERE status = 'in-progress') AS in_progress,
			COUNT(*) FILTER (WHERE status = 'completed') AS completed
		FROM tasks
	`).Scan(&stats.Tasks.Total, &stats.Tasks.Pending, &stats.Tasks.InProgress, &stats.Tasks.Completed); err != nil {
		ps.logger.Printf("error querying task stats: %v", err)
		return StatsResponse{}, fmt.Errorf("query task stats: %w", err)
	}

	return stats, nil
}

func (ps *PostgresStore) CreateUser(name, email, role string) (User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), dbOperationTimeout)
	defer cancel()

	var user User
	if err := ps.db.QueryRowContext(ctx, `
		INSERT INTO users (name, email, role)
		VALUES ($1, $2, $3)
		RETURNING id, name, email, role
	`, name, email, role).Scan(&user.ID, &user.Name, &user.Email, &user.Role); err != nil {
		return User{}, fmt.Errorf("insert user: %w", err)
	}

	return user, nil
}

func (ps *PostgresStore) CreateTask(title, status string, userID int, actor string) (Task, error) {
	if !isValidTaskStatus(status) {
		return Task{}, fmt.Errorf("%w: %q", ErrInvalidTaskStatus, status)
	}

	ctx, cancel := context.WithTimeout(context.Background(), dbOperationTimeout)
	defer cancel()

	tx, err := ps.db.BeginTx(ctx, nil)
	if err != nil {
		return Task{}, fmt.Errorf("begin create task transaction: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	var userExists bool
	if err := tx.QueryRowContext(ctx, `
		SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)
	`, userID).Scan(&userExists); err != nil {
		return Task{}, fmt.Errorf("check user existence: %w", err)
	}
	if !userExists {
		return Task{}, fmt.Errorf("%w: %d", ErrUserDoesNotExist, userID)
	}

	var task Task
	if err := tx.QueryRowContext(ctx, `
		INSERT INTO tasks (title, status, user_id)
		VALUES ($1, $2, $3)
		RETURNING id, title, status, user_id
	`, title, status, userID).Scan(&task.ID, &task.Title, &task.Status, &task.UserID); err != nil {
		return Task{}, fmt.Errorf("insert task: %w", err)
	}

	changedAt := time.Now().UTC()
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO task_history (task_id, changed_at, changed_by, field, from_value, to_value)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, task.ID, changedAt, normalizeActor(actor), "status", nil, status); err != nil {
		return Task{}, fmt.Errorf("insert task history: %w", err)
	}
	task.LastChange = &TaskHistoryItem{
		TaskID:    task.ID,
		ChangedAt: changedAt,
		ChangedBy: normalizeActor(actor),
		Field:     "status",
		ToValue:   status,
	}

	if err := tx.Commit(); err != nil {
		return Task{}, fmt.Errorf("commit create task transaction: %w", err)
	}
	committed = true

	return task, nil
}

func (ps *PostgresStore) UpdateTask(id int, update TaskUpdate, actor string) (Task, error) {
	if update.Status != nil && !isValidTaskStatus(*update.Status) {
		return Task{}, fmt.Errorf("%w: %q", ErrInvalidTaskStatus, *update.Status)
	}

	ctx, cancel := context.WithTimeout(context.Background(), dbOperationTimeout)
	defer cancel()

	tx, err := ps.db.BeginTx(ctx, nil)
	if err != nil {
		return Task{}, fmt.Errorf("begin update task transaction: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	var current Task
	if err := tx.QueryRowContext(ctx, `
		SELECT id, title, status, user_id
		FROM tasks
		WHERE id = $1
		FOR UPDATE
	`, id).Scan(&current.ID, &current.Title, &current.Status, &current.UserID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Task{}, fmt.Errorf("%w: %d", ErrTaskNotFound, id)
		}
		return Task{}, fmt.Errorf("load task for update: %w", err)
	}

	if update.UserID != nil {
		var userExists bool
		if err := tx.QueryRowContext(ctx, `
			SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)
		`, *update.UserID).Scan(&userExists); err != nil {
			return Task{}, fmt.Errorf("check user existence: %w", err)
		}
		if !userExists {
			return Task{}, fmt.Errorf("%w: %d", ErrUserDoesNotExist, *update.UserID)
		}
	}

	now := time.Now().UTC()
	actorName := normalizeActor(actor)
	var latestChange *TaskHistoryItem

	if update.Title != nil {
		if current.Title != *update.Title {
			from := current.Title
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO task_history (task_id, changed_at, changed_by, field, from_value, to_value)
				VALUES ($1, $2, $3, $4, $5, $6)
			`, id, now, actorName, "title", from, *update.Title); err != nil {
				return Task{}, fmt.Errorf("insert task history: %w", err)
			}
			fromValue := from
			latestChange = &TaskHistoryItem{
				TaskID:    id,
				ChangedAt: now,
				ChangedBy: actorName,
				Field:     "title",
				FromValue: &fromValue,
				ToValue:   *update.Title,
			}
		}
		current.Title = *update.Title
	}
	if update.Status != nil {
		if current.Status != *update.Status {
			from := current.Status
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO task_history (task_id, changed_at, changed_by, field, from_value, to_value)
				VALUES ($1, $2, $3, $4, $5, $6)
			`, id, now, actorName, "status", from, *update.Status); err != nil {
				return Task{}, fmt.Errorf("insert task history: %w", err)
			}
			fromValue := from
			latestChange = &TaskHistoryItem{
				TaskID:    id,
				ChangedAt: now,
				ChangedBy: actorName,
				Field:     "status",
				FromValue: &fromValue,
				ToValue:   *update.Status,
			}
		}
		current.Status = *update.Status
	}
	if update.UserID != nil {
		if current.UserID != *update.UserID {
			from := strconv.Itoa(current.UserID)
			to := strconv.Itoa(*update.UserID)
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO task_history (task_id, changed_at, changed_by, field, from_value, to_value)
				VALUES ($1, $2, $3, $4, $5, $6)
			`, id, now, actorName, "userId", from, to); err != nil {
				return Task{}, fmt.Errorf("insert task history: %w", err)
			}
			fromValue := from
			latestChange = &TaskHistoryItem{
				TaskID:    id,
				ChangedAt: now,
				ChangedBy: actorName,
				Field:     "userId",
				FromValue: &fromValue,
				ToValue:   to,
			}
		}
		current.UserID = *update.UserID
	}

	if _, err := tx.ExecContext(ctx, `
		UPDATE tasks
		SET title = $1, status = $2, user_id = $3
		WHERE id = $4
	`, current.Title, current.Status, current.UserID, id); err != nil {
		return Task{}, fmt.Errorf("update task row: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return Task{}, fmt.Errorf("commit update task transaction: %w", err)
	}
	committed = true
	if latestChange == nil {
		var (
			entry     TaskHistoryItem
			fromValue sql.NullString
		)
		err := ps.db.QueryRowContext(ctx, `
			SELECT id, task_id, changed_at, changed_by, field, from_value, to_value
			FROM task_history
			WHERE task_id = $1
			ORDER BY changed_at DESC, id DESC
			LIMIT 1
		`, id).Scan(
			&entry.ID,
			&entry.TaskID,
			&entry.ChangedAt,
			&entry.ChangedBy,
			&entry.Field,
			&fromValue,
			&entry.ToValue,
		)
		if err == nil {
			if fromValue.Valid {
				from := fromValue.String
				entry.FromValue = &from
			}
			latestChange = &entry
		}
	}
	current.LastChange = latestChange

	return current, nil
}

func (ps *PostgresStore) initSchema() error {
	ctx, cancel := context.WithTimeout(context.Background(), dbOperationTimeout)
	defer cancel()

	statements := []string{
		`
		CREATE TABLE IF NOT EXISTS users (
			id BIGSERIAL PRIMARY KEY,
			name TEXT NOT NULL,
			email TEXT NOT NULL,
			role TEXT NOT NULL
		);
		`,
		`
		CREATE TABLE IF NOT EXISTS tasks (
			id BIGSERIAL PRIMARY KEY,
			title TEXT NOT NULL,
			status TEXT NOT NULL CHECK (status IN ('pending', 'in-progress', 'completed')),
			user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE RESTRICT
		);
		`,
		`
		CREATE TABLE IF NOT EXISTS task_history (
			id BIGSERIAL PRIMARY KEY,
			task_id BIGINT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
			changed_at TIMESTAMPTZ NOT NULL,
			changed_by TEXT NOT NULL,
			field TEXT NOT NULL CHECK (field IN ('title', 'status', 'userId')),
			from_value TEXT,
			to_value TEXT NOT NULL
		);
		`,
		`CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status);`,
		`CREATE INDEX IF NOT EXISTS idx_tasks_user_id ON tasks(user_id);`,
		`CREATE INDEX IF NOT EXISTS idx_task_history_task_id ON task_history(task_id);`,
		`CREATE INDEX IF NOT EXISTS idx_task_history_changed_at ON task_history(changed_at DESC);`,
	}

	for _, statement := range statements {
		if _, err := ps.db.ExecContext(ctx, statement); err != nil {
			return err
		}
	}

	return nil
}

func (ps *PostgresStore) seedInitialData() error {
	ctx, cancel := context.WithTimeout(context.Background(), dbOperationTimeout)
	defer cancel()

	tx, err := ps.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	var userCount int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&userCount); err != nil {
		return err
	}

	if userCount == 0 {
		for _, user := range initialUsers {
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO users (id, name, email, role)
				VALUES ($1, $2, $3, $4)
			`, user.ID, user.Name, user.Email, user.Role); err != nil {
				return err
			}
		}

		if _, err := tx.ExecContext(ctx, `
			SELECT setval(
				pg_get_serial_sequence('users', 'id'),
				COALESCE((SELECT MAX(id) FROM users), 1),
				true
			)
		`); err != nil {
			return err
		}
	}

	var taskCount int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM tasks`).Scan(&taskCount); err != nil {
		return err
	}

	if taskCount == 0 {
		for _, task := range initialTasks {
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO tasks (id, title, status, user_id)
				VALUES ($1, $2, $3, $4)
			`, task.ID, task.Title, task.Status, task.UserID); err != nil {
				return err
			}
		}

		if _, err := tx.ExecContext(ctx, `
			SELECT setval(
				pg_get_serial_sequence('tasks', 'id'),
				COALESCE((SELECT MAX(id) FROM tasks), 1),
				true
			)
		`); err != nil {
			return err
		}
	}

	var historyCount int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM task_history`).Scan(&historyCount); err != nil {
		return err
	}
	if historyCount == 0 {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO task_history (task_id, changed_at, changed_by, field, from_value, to_value)
			SELECT id, NOW(), $1, 'status', NULL, status
			FROM tasks
		`, defaultActorName); err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	committed = true

	return nil
}

func pingWithRetry(db *sql.DB) error {
	var lastErr error
	for attempt := 1; attempt <= dbPingRetries; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), dbOperationTimeout)
		err := db.PingContext(ctx)
		cancel()
		if err == nil {
			return nil
		}

		lastErr = err
		time.Sleep(1 * time.Second)
	}

	return fmt.Errorf("ping postgres after %d retries: %w", dbPingRetries, lastErr)
}
