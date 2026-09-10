package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/kbaker/tauri-go-service-example/services/go/task-service/internal/task"
	"github.com/kbaker/tauri-go-service-example/services/go/task-service/migrations"
	modernsqlite "modernc.org/sqlite"
)

const timestampLayout = time.RFC3339Nano

type Repository struct {
	db *sql.DB
}

func Open(ctx context.Context, path string) (*Repository, error) {
	if path == "" {
		return nil, errors.New("database path is required")
	}
	if path != ":memory:" {
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			return nil, fmt.Errorf("create database directory: %w", err)
		}
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	// A single embedded application writer is the POC's explicit concurrency
	// model. It also ensures connection-scoped pragmas apply to every query.
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	repository := &Repository{db: db}
	if err := repository.initialize(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	return repository, nil
}

func (r *Repository) initialize(ctx context.Context) error {
	for _, statement := range []string{
		"PRAGMA foreign_keys = ON",
		"PRAGMA busy_timeout = 5000",
		"PRAGMA journal_mode = WAL",
	} {
		if _, err := r.db.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("configure sqlite: %w", err)
		}
	}
	if err := r.applyMigrations(ctx); err != nil {
		return err
	}
	return nil
}

func (r *Repository) applyMigrations(ctx context.Context) error {
	if _, err := r.db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		version TEXT PRIMARY KEY,
		applied_at TEXT NOT NULL
	)`); err != nil {
		return fmt.Errorf("create migration table: %w", err)
	}

	entries, err := fs.ReadDir(migrations.Files, ".")
	if err != nil {
		return fmt.Errorf("read embedded migrations: %w", err)
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".sql") {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)
	for _, name := range names {
		var exists int
		if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM schema_migrations WHERE version = ?", name).Scan(&exists); err != nil {
			return fmt.Errorf("check migration %s: %w", name, err)
		}
		if exists != 0 {
			continue
		}
		script, err := fs.ReadFile(migrations.Files, name)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", name, err)
		}
		tx, err := r.db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("begin migration %s: %w", name, err)
		}
		if _, err = tx.ExecContext(ctx, string(script)); err == nil {
			_, err = tx.ExecContext(ctx, "INSERT INTO schema_migrations(version, applied_at) VALUES (?, ?)", name, time.Now().UTC().Format(timestampLayout))
		}
		if err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("apply migration %s: %w", name, err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migration %s: %w", name, err)
		}
	}
	return nil
}

func (r *Repository) Close() error { return r.db.Close() }

func (r *Repository) List(ctx context.Context, query task.ListQuery) ([]task.Task, error) {
	statement := `SELECT id, title, description, status, created_at, updated_at FROM tasks`
	args := make([]any, 0, 3)
	if query.Status != nil {
		statement += " WHERE status = ?"
		args = append(args, string(*query.Status))
	}
	statement += " ORDER BY created_at DESC, id DESC LIMIT ? OFFSET ?"
	args = append(args, query.Limit, query.Offset)
	rows, err := r.db.QueryContext(ctx, statement, args...)
	if err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}
	defer rows.Close()

	tasks := make([]task.Task, 0)
	for rows.Next() {
		item, err := scanTask(rows)
		if err != nil {
			return nil, fmt.Errorf("scan task list: %w", err)
		}
		tasks = append(tasks, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate task list: %w", err)
	}
	return tasks, nil
}

func (r *Repository) Get(ctx context.Context, id string) (task.Task, error) {
	item, err := scanTask(r.db.QueryRowContext(ctx, `
		SELECT id, title, description, status, created_at, updated_at
		FROM tasks WHERE id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return task.Task{}, task.ErrNotFound
	}
	if err != nil {
		return task.Task{}, fmt.Errorf("get task: %w", err)
	}
	return item, nil
}

func (r *Repository) Create(ctx context.Context, item task.Task) (task.Task, error) {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO tasks(id, title, description, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		item.ID, item.Title, item.Description, item.Status,
		item.CreatedAt.UTC().Format(timestampLayout), item.UpdatedAt.UTC().Format(timestampLayout))
	if err != nil {
		var sqliteError *modernsqlite.Error
		if errors.As(err, &sqliteError) && sqliteError.Code()&0xff == 19 {
			return task.Task{}, task.ErrConflict
		}
		return task.Task{}, fmt.Errorf("create task: %w", err)
	}
	return item, nil
}

func (r *Repository) Update(ctx context.Context, id string, input task.UpdateInput, updatedAt time.Time) (result task.Task, err error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return task.Task{}, fmt.Errorf("begin task update: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	item, err := scanTask(tx.QueryRowContext(ctx, `
		SELECT id, title, description, status, created_at, updated_at
		FROM tasks WHERE id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return task.Task{}, task.ErrNotFound
	}
	if err != nil {
		return task.Task{}, fmt.Errorf("get task for update: %w", err)
	}
	if input.Title != nil {
		item.Title = *input.Title
	}
	if input.Description.Set {
		item.Description = input.Description.Value
	}
	if input.Status != nil {
		item.Status = *input.Status
	}
	item.UpdatedAt = updatedAt.UTC()
	if _, err = tx.ExecContext(ctx, `
		UPDATE tasks SET title = ?, description = ?, status = ?, updated_at = ? WHERE id = ?`,
		item.Title, item.Description, item.Status, item.UpdatedAt.Format(timestampLayout), item.ID); err != nil {
		return task.Task{}, fmt.Errorf("update task: %w", err)
	}
	if err = tx.Commit(); err != nil {
		return task.Task{}, fmt.Errorf("commit task update: %w", err)
	}
	return item, nil
}

func (r *Repository) Delete(ctx context.Context, id string) error {
	result, err := r.db.ExecContext(ctx, "DELETE FROM tasks WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete task: %w", err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read deleted task count: %w", err)
	}
	if count == 0 {
		return task.ErrNotFound
	}
	return nil
}

type scanner interface {
	Scan(...any) error
}

func scanTask(row scanner) (task.Task, error) {
	var item task.Task
	var description sql.NullString
	var status, createdAt, updatedAt string
	if err := row.Scan(&item.ID, &item.Title, &description, &status, &createdAt, &updatedAt); err != nil {
		return task.Task{}, err
	}
	item.Status = task.Status(status)
	if description.Valid {
		item.Description = &description.String
	}
	var err error
	item.CreatedAt, err = time.Parse(timestampLayout, createdAt)
	if err != nil {
		return task.Task{}, fmt.Errorf("parse created timestamp: %w", err)
	}
	item.UpdatedAt, err = time.Parse(timestampLayout, updatedAt)
	if err != nil {
		return task.Task{}, fmt.Errorf("parse updated timestamp: %w", err)
	}
	return item, nil
}
