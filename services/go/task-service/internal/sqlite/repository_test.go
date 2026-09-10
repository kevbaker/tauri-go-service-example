package sqlite

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/kbaker/tauri-go-service-example/services/go/task-service/internal/task"
)

func TestRepositoryCRUDAndRestartPersistence(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "tasks.db")
	repository, err := Open(ctx, path)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	description := "first description"
	now := time.Date(2026, 9, 9, 12, 0, 0, 123, time.UTC)
	created, err := repository.Create(ctx, task.Task{ID: "one", Title: "First", Description: &description, Status: task.StatusTodo, CreatedAt: now, UpdatedAt: now})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.ID != "one" {
		t.Fatalf("Create() = %#v", created)
	}
	if _, err := repository.Create(ctx, created); !errors.Is(err, task.ErrConflict) {
		t.Fatalf("duplicate Create() error = %v", err)
	}
	if err := repository.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	repository, err = Open(ctx, path)
	if err != nil {
		t.Fatalf("Open() after restart error = %v", err)
	}
	defer repository.Close()
	got, err := repository.Get(ctx, "one")
	if err != nil || got.Title != "First" || got.Description == nil || *got.Description != description {
		t.Fatalf("Get() = %#v, %v", got, err)
	}

	done := task.StatusDone
	updated, err := repository.Update(ctx, "one", task.UpdateInput{Description: task.OptionalString{Set: true}, Status: &done}, now.Add(time.Minute))
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if updated.Description != nil || updated.Status != task.StatusDone {
		t.Fatalf("Update() = %#v", updated)
	}
	items, err := repository.List(ctx, task.ListQuery{Status: &done, Limit: 25})
	if err != nil || len(items) != 1 {
		t.Fatalf("List() = %#v, %v", items, err)
	}
	if err := repository.Delete(ctx, "one"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if _, err := repository.Get(ctx, "one"); !errors.Is(err, task.ErrNotFound) {
		t.Fatalf("Get() after delete error = %v", err)
	}
}

func TestMigrationIsIdempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tasks.db")
	for i := 0; i < 2; i++ {
		repository, err := Open(context.Background(), path)
		if err != nil {
			t.Fatalf("Open() iteration %d error = %v", i, err)
		}
		if err := repository.Close(); err != nil {
			t.Fatalf("Close() iteration %d error = %v", i, err)
		}
	}
}

func TestRepositoryListUsesStatusPaginationAndNewestFirst(t *testing.T) {
	ctx := context.Background()
	repository, err := Open(ctx, filepath.Join(t.TempDir(), "tasks.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer repository.Close()

	base := time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC)
	items := []task.Task{
		{ID: "one", Title: "First", Status: task.StatusTodo, CreatedAt: base, UpdatedAt: base},
		{ID: "two", Title: "Second", Status: task.StatusDone, CreatedAt: base.Add(time.Minute), UpdatedAt: base.Add(time.Minute)},
		{ID: "three", Title: "Third", Status: task.StatusDone, CreatedAt: base.Add(2 * time.Minute), UpdatedAt: base.Add(2 * time.Minute)},
	}
	for _, item := range items {
		if _, err := repository.Create(ctx, item); err != nil {
			t.Fatalf("Create(%q) error = %v", item.ID, err)
		}
	}

	done := task.StatusDone
	got, err := repository.List(ctx, task.ListQuery{Status: &done, Limit: 1, Offset: 1})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(got) != 1 || got[0].ID != "two" {
		t.Fatalf("List() = %#v", got)
	}
}

func TestRepositoryReportsMissingMutations(t *testing.T) {
	repository, err := Open(context.Background(), filepath.Join(t.TempDir(), "tasks.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer repository.Close()

	status := task.StatusDone
	if _, err := repository.Update(context.Background(), "missing", task.UpdateInput{Status: &status}, time.Now()); !errors.Is(err, task.ErrNotFound) {
		t.Fatalf("Update() error = %v", err)
	}
	if err := repository.Delete(context.Background(), "missing"); !errors.Is(err, task.ErrNotFound) {
		t.Fatalf("Delete() error = %v", err)
	}
}

func TestOpenRejectsEmptyPath(t *testing.T) {
	if _, err := Open(context.Background(), ""); err == nil {
		t.Fatal("Open() accepted an empty path")
	}
}
