package task

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

type fakeRepository struct {
	created     Task
	updated     UpdateInput
	listedQuery ListQuery
	gotID       string
	deletedID   string
}

func (f *fakeRepository) List(_ context.Context, query ListQuery) ([]Task, error) {
	f.listedQuery = query
	return nil, nil
}
func (f *fakeRepository) Get(_ context.Context, id string) (Task, error) {
	f.gotID = id
	return Task{ID: id}, nil
}
func (f *fakeRepository) Create(_ context.Context, task Task) (Task, error) {
	f.created = task
	return task, nil
}
func (f *fakeRepository) Update(_ context.Context, _ string, input UpdateInput, _ time.Time) (Task, error) {
	f.updated = input
	return Task{}, nil
}
func (f *fakeRepository) Delete(_ context.Context, id string) error {
	f.deletedID = id
	return nil
}

func TestCreateValidatesAndNormalizesTitle(t *testing.T) {
	repository := &fakeRepository{}
	service := NewService(repository)
	service.now = func() time.Time { return time.Date(2026, 9, 9, 12, 0, 0, 0, time.FixedZone("offset", 3600)) }
	service.newID = func() (string, error) { return "task-1", nil }

	created, err := service.Create(context.Background(), CreateInput{Title: "  Prove bridge  "})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.Title != "Prove bridge" || created.Status != StatusTodo {
		t.Fatalf("Create() = %#v", created)
	}
	if created.CreatedAt.Location() != time.UTC {
		t.Fatalf("CreatedAt location = %v", created.CreatedAt.Location())
	}
}

func TestCreateRejectsBlankTitle(t *testing.T) {
	service := NewService(&fakeRepository{})
	_, err := service.Create(context.Background(), CreateInput{Title: "  "})
	var validation *ValidationError
	if !errors.As(err, &validation) || validation.Fields["title"] == "" {
		t.Fatalf("Create() error = %v", err)
	}
}

func TestCreateAcceptsExplicitStatus(t *testing.T) {
	repository := &fakeRepository{}
	service := NewService(repository)
	service.newID = func() (string, error) { return "task-1", nil }
	status := StatusInProgress
	created, err := service.Create(context.Background(), CreateInput{Title: "Started", Status: &status})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.Status != StatusInProgress {
		t.Fatalf("Create() status = %q", created.Status)
	}
}

func TestUpdateRejectsEmptyPatch(t *testing.T) {
	service := NewService(&fakeRepository{})
	_, err := service.Update(context.Background(), "task-1", UpdateInput{})
	var validation *ValidationError
	if !errors.As(err, &validation) || validation.Fields["input"] == "" {
		t.Fatalf("Update() error = %v", err)
	}
}

func TestCreateRejectsOversizedFields(t *testing.T) {
	tests := []struct {
		name  string
		input CreateInput
		field string
	}{
		{name: "title", input: CreateInput{Title: strings.Repeat("x", MaxTitleLength+1)}, field: "title"},
		{name: "description", input: CreateInput{Title: "Valid", Description: stringPointer(strings.Repeat("x", MaxDescriptionLength+1))}, field: "description"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := NewService(&fakeRepository{}).Create(context.Background(), test.input)
			var validation *ValidationError
			if !errors.As(err, &validation) || validation.Fields[test.field] == "" {
				t.Fatalf("Create() error = %v", err)
			}
		})
	}
}

func TestListAppliesDefaultsAndValidatesQuery(t *testing.T) {
	repository := &fakeRepository{}
	service := NewService(repository)
	if _, err := service.List(context.Background(), ListQuery{}); err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if repository.listedQuery.Limit != DefaultListLimit {
		t.Fatalf("List() limit = %d", repository.listedQuery.Limit)
	}

	badStatus := Status("blocked")
	for name, query := range map[string]ListQuery{
		"status": {Status: &badStatus},
		"limit":  {Limit: MaxListLimit + 1},
		"offset": {Offset: -1},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := service.List(context.Background(), query)
			var validation *ValidationError
			if !errors.As(err, &validation) {
				t.Fatalf("List() error = %v", err)
			}
		})
	}
}

func TestUpdateNormalizesTitleAndRejectsInvalidStatus(t *testing.T) {
	repository := &fakeRepository{}
	service := NewService(repository)
	title := "  Updated  "
	if _, err := service.Update(context.Background(), "task-1", UpdateInput{Title: &title}); err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if repository.updated.Title == nil || *repository.updated.Title != "Updated" {
		t.Fatalf("Update() title = %#v", repository.updated.Title)
	}

	invalid := Status("invalid")
	_, err := service.Update(context.Background(), "task-1", UpdateInput{Status: &invalid})
	var validation *ValidationError
	if !errors.As(err, &validation) || validation.Fields["status"] == "" {
		t.Fatalf("Update() error = %v", err)
	}
}

func TestGetAndDeleteRequireAndForwardID(t *testing.T) {
	repository := &fakeRepository{}
	service := NewService(repository)
	if _, err := service.Get(context.Background(), " "); err == nil {
		t.Fatal("Get() accepted blank ID")
	}
	if err := service.Delete(context.Background(), " "); err == nil {
		t.Fatal("Delete() accepted blank ID")
	}

	if _, err := service.Get(context.Background(), "task-1"); err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if err := service.Delete(context.Background(), "task-1"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if repository.gotID != "task-1" || repository.deletedID != "task-1" {
		t.Fatalf("forwarded IDs = get %q, delete %q", repository.gotID, repository.deletedID)
	}
}

func stringPointer(value string) *string { return &value }
