package task

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	MaxTitleLength       = 120
	MaxDescriptionLength = 4000
	DefaultListLimit     = 25
	MaxListLimit         = 100
)

var (
	ErrNotFound = errors.New("task not found")
	ErrConflict = errors.New("task conflict")
)

type Status string

const (
	StatusTodo       Status = "todo"
	StatusInProgress Status = "in_progress"
	StatusDone       Status = "done"
)

func (s Status) Valid() bool {
	return s == StatusTodo || s == StatusInProgress || s == StatusDone
}

type Task struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description *string   `json:"description"`
	Status      Status    `json:"status"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type CreateInput struct {
	Title       string  `json:"title"`
	Description *string `json:"description,omitempty"`
	Status      *Status `json:"status,omitempty"`
}

// OptionalString preserves the difference between an omitted value and an
// explicit JSON null in task update requests.
type OptionalString struct {
	Set   bool
	Value *string
}

type UpdateInput struct {
	Title       *string        `json:"title,omitempty"`
	Description OptionalString `json:"-"`
	Status      *Status        `json:"status,omitempty"`
}

type ListQuery struct {
	Status *Status `json:"status,omitempty"`
	Limit  int     `json:"limit,omitempty"`
	Offset int     `json:"offset,omitempty"`
}

type ValidationError struct {
	Fields map[string]string
}

func (e *ValidationError) Error() string { return "task validation failed" }

func validateCreate(input *CreateInput) error {
	fields := map[string]string{}
	input.Title = strings.TrimSpace(input.Title)
	if input.Title == "" {
		fields["title"] = "Title is required"
	} else if len([]rune(input.Title)) > MaxTitleLength {
		fields["title"] = fmt.Sprintf("Title must be at most %d characters", MaxTitleLength)
	}
	if input.Description != nil && len([]rune(*input.Description)) > MaxDescriptionLength {
		fields["description"] = fmt.Sprintf("Description must be at most %d characters", MaxDescriptionLength)
	}
	if input.Status != nil && !input.Status.Valid() {
		fields["status"] = "Status must be todo, in_progress, or done"
	}
	if len(fields) > 0 {
		return &ValidationError{Fields: fields}
	}
	return nil
}

func validateUpdate(input *UpdateInput) error {
	fields := map[string]string{}
	if input.Title == nil && !input.Description.Set && input.Status == nil {
		fields["input"] = "At least one field is required"
	}
	if input.Title != nil {
		trimmed := strings.TrimSpace(*input.Title)
		input.Title = &trimmed
		if trimmed == "" {
			fields["title"] = "Title is required"
		} else if len([]rune(trimmed)) > MaxTitleLength {
			fields["title"] = fmt.Sprintf("Title must be at most %d characters", MaxTitleLength)
		}
	}
	if input.Description.Set && input.Description.Value != nil && len([]rune(*input.Description.Value)) > MaxDescriptionLength {
		fields["description"] = fmt.Sprintf("Description must be at most %d characters", MaxDescriptionLength)
	}
	if input.Status != nil && !input.Status.Valid() {
		fields["status"] = "Status must be todo, in_progress, or done"
	}
	if len(fields) > 0 {
		return &ValidationError{Fields: fields}
	}
	return nil
}

func validateList(query *ListQuery) error {
	fields := map[string]string{}
	if query.Status != nil && !query.Status.Valid() {
		fields["status"] = "Status must be todo, in_progress, or done"
	}
	if query.Limit == 0 {
		query.Limit = DefaultListLimit
	}
	if query.Limit < 1 || query.Limit > MaxListLimit {
		fields["limit"] = fmt.Sprintf("Limit must be between 1 and %d", MaxListLimit)
	}
	if query.Offset < 0 {
		fields["offset"] = "Offset must not be negative"
	}
	if len(fields) > 0 {
		return &ValidationError{Fields: fields}
	}
	return nil
}

type Repository interface {
	List(context.Context, ListQuery) ([]Task, error)
	Get(context.Context, string) (Task, error)
	Create(context.Context, Task) (Task, error)
	Update(context.Context, string, UpdateInput, time.Time) (Task, error)
	Delete(context.Context, string) error
}

type Service struct {
	repository Repository
	now        func() time.Time
	newID      func() (string, error)
}

func NewService(repository Repository) *Service {
	return &Service{repository: repository, now: func() time.Time { return time.Now().UTC() }, newID: randomID}
}

func (s *Service) List(ctx context.Context, query ListQuery) ([]Task, error) {
	if err := validateList(&query); err != nil {
		return nil, err
	}
	return s.repository.List(ctx, query)
}

func (s *Service) Get(ctx context.Context, id string) (Task, error) {
	if strings.TrimSpace(id) == "" {
		return Task{}, &ValidationError{Fields: map[string]string{"id": "ID is required"}}
	}
	return s.repository.Get(ctx, id)
}

func (s *Service) Create(ctx context.Context, input CreateInput) (Task, error) {
	if err := validateCreate(&input); err != nil {
		return Task{}, err
	}
	id, err := s.newID()
	if err != nil {
		return Task{}, fmt.Errorf("generate task id: %w", err)
	}
	now := s.now().UTC()
	status := StatusTodo
	if input.Status != nil {
		status = *input.Status
	}
	return s.repository.Create(ctx, Task{ID: id, Title: input.Title, Description: input.Description, Status: status, CreatedAt: now, UpdatedAt: now})
}

func (s *Service) Update(ctx context.Context, id string, input UpdateInput) (Task, error) {
	if strings.TrimSpace(id) == "" {
		return Task{}, &ValidationError{Fields: map[string]string{"id": "ID is required"}}
	}
	if err := validateUpdate(&input); err != nil {
		return Task{}, err
	}
	return s.repository.Update(ctx, id, input, s.now().UTC())
}

func (s *Service) Delete(ctx context.Context, id string) error {
	if strings.TrimSpace(id) == "" {
		return &ValidationError{Fields: map[string]string{"id": "ID is required"}}
	}
	return s.repository.Delete(ctx, id)
}

func randomID() (string, error) {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	// RFC 4122 version 4 UUID, generated without a third-party dependency.
	raw[6] = (raw[6] & 0x0f) | 0x40
	raw[8] = (raw[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", raw[0:4], raw[4:6], raw[6:8], raw[8:10], raw[10:16]), nil
}
