package mcpserver

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/kbaker/tauri-go-service-example/services/go/task-service/internal/task"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	Name    = "tauri-go-task-service"
	Version = "0.1.0"
)

// TaskService is the domain-facing boundary required by the MCP adapter.
// The concrete task.Service remains responsible for validation and persistence.
type TaskService interface {
	List(context.Context, task.ListQuery) ([]task.Task, error)
	Get(context.Context, string) (task.Task, error)
	Create(context.Context, task.CreateInput) (task.Task, error)
	Update(context.Context, string, task.UpdateInput) (task.Task, error)
	Delete(context.Context, string) error
}

type adapter struct {
	tasks  TaskService
	logger *slog.Logger
}

type listTasksInput struct {
	Status *task.Status `json:"status,omitempty" jsonschema:"Optional task status filter: todo, in_progress, or done"`
	Limit  int          `json:"limit,omitempty" jsonschema:"Maximum tasks to return; defaults to 25 and cannot exceed 100"`
	Offset int          `json:"offset,omitempty" jsonschema:"Zero-based number of tasks to skip"`
}

type listTasksOutput struct {
	Tasks []task.Task `json:"tasks" jsonschema:"Tasks ordered from newest to oldest"`
}

type taskIDInput struct {
	ID string `json:"id" jsonschema:"Task identifier"`
}

type taskOutput struct {
	Task task.Task `json:"task"`
}

type createTaskInput struct {
	Title       string       `json:"title" jsonschema:"Required task title, at most 120 characters"`
	Description *string      `json:"description,omitempty" jsonschema:"Optional task description, at most 4000 characters"`
	Status      *task.Status `json:"status,omitempty" jsonschema:"Optional initial status: todo, in_progress, or done; defaults to todo"`
}

type updateTaskInput struct {
	ID               string       `json:"id" jsonschema:"Task identifier"`
	Title            *string      `json:"title,omitempty" jsonschema:"Replacement task title, at most 120 characters"`
	Description      *string      `json:"description,omitempty" jsonschema:"Replacement task description, at most 4000 characters"`
	ClearDescription bool         `json:"clearDescription,omitempty" jsonschema:"Set true to remove the current description; cannot be combined with description"`
	Status           *task.Status `json:"status,omitempty" jsonschema:"Replacement status: todo, in_progress, or done"`
}

type deleteTaskOutput struct {
	Deleted bool   `json:"deleted"`
	ID      string `json:"id"`
}

// New creates an MCP server that exposes the existing task application service
// as a deliberately narrow set of model-facing tools.
func New(tasks TaskService, logger *slog.Logger) *mcp.Server {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	a := &adapter{tasks: tasks, logger: logger}
	server := mcp.NewServer(&mcp.Implementation{Name: Name, Version: Version}, nil)

	closedWorld := false
	nonDestructive := false
	destructive := true
	readOnly := &mcp.ToolAnnotations{ReadOnlyHint: true, OpenWorldHint: &closedWorld}
	additive := &mcp.ToolAnnotations{DestructiveHint: &nonDestructive, OpenWorldHint: &closedWorld}
	mutating := &mcp.ToolAnnotations{DestructiveHint: &destructive, OpenWorldHint: &closedWorld}
	deleting := &mcp.ToolAnnotations{DestructiveHint: &destructive, IdempotentHint: true, OpenWorldHint: &closedWorld}

	mcp.AddTool(server, &mcp.Tool{
		Name: "tasks_list", Title: "List tasks",
		Description: "List persisted tasks, optionally filtered by status and paginated.",
		Annotations: readOnly, InputSchema: schemaFor[listTasksInput](), OutputSchema: schemaFor[listTasksOutput](),
	}, a.list)
	mcp.AddTool(server, &mcp.Tool{
		Name: "tasks_get", Title: "Get task",
		Description: "Get one persisted task by its identifier.",
		Annotations: readOnly, InputSchema: schemaFor[taskIDInput](), OutputSchema: schemaFor[taskOutput](),
	}, a.get)
	mcp.AddTool(server, &mcp.Tool{
		Name: "tasks_create", Title: "Create task",
		Description: "Create and persist a task. This adds data but does not overwrite existing tasks.",
		Annotations: additive, InputSchema: schemaFor[createTaskInput](), OutputSchema: schemaFor[taskOutput](),
	}, a.create)
	mcp.AddTool(server, &mcp.Tool{
		Name: "tasks_update", Title: "Update task",
		Description: "Change selected fields on an existing task. Omitted fields remain unchanged.",
		Annotations: mutating, InputSchema: schemaFor[updateTaskInput](), OutputSchema: schemaFor[taskOutput](),
	}, a.update)
	mcp.AddTool(server, &mcp.Tool{
		Name: "tasks_delete", Title: "Delete task",
		Description: "Permanently delete one task. Call only after the user explicitly confirms the deletion.",
		Annotations: deleting, InputSchema: schemaFor[taskIDInput](), OutputSchema: schemaFor[deleteTaskOutput](),
	}, a.delete)

	return server
}

func schemaFor[Value any]() *jsonschema.Schema {
	schema, err := jsonschema.For[Value](&jsonschema.ForOptions{TypeSchemas: map[reflect.Type]*jsonschema.Schema{
		reflect.TypeFor[task.Status](): {
			Type:        "string",
			Description: "Task status",
			Enum:        []any{string(task.StatusTodo), string(task.StatusInProgress), string(task.StatusDone)},
		},
	}})
	if err != nil {
		panic(fmt.Sprintf("build MCP schema: %v", err))
	}
	return schema
}

func (a *adapter) list(ctx context.Context, request *mcp.CallToolRequest, input listTasksInput) (*mcp.CallToolResult, listTasksOutput, error) {
	return invoke(ctx, request, a.logger, "tasks_list", func() (listTasksOutput, error) {
		items, err := a.tasks.List(ctx, task.ListQuery{Status: input.Status, Limit: input.Limit, Offset: input.Offset})
		return listTasksOutput{Tasks: items}, err
	})
}

func (a *adapter) get(ctx context.Context, request *mcp.CallToolRequest, input taskIDInput) (*mcp.CallToolResult, taskOutput, error) {
	return invoke(ctx, request, a.logger, "tasks_get", func() (taskOutput, error) {
		item, err := a.tasks.Get(ctx, input.ID)
		return taskOutput{Task: item}, err
	})
}

func (a *adapter) create(ctx context.Context, request *mcp.CallToolRequest, input createTaskInput) (*mcp.CallToolResult, taskOutput, error) {
	return invoke(ctx, request, a.logger, "tasks_create", func() (taskOutput, error) {
		item, err := a.tasks.Create(ctx, task.CreateInput{Title: input.Title, Description: input.Description, Status: input.Status})
		return taskOutput{Task: item}, err
	})
}

func (a *adapter) update(ctx context.Context, request *mcp.CallToolRequest, input updateTaskInput) (*mcp.CallToolResult, taskOutput, error) {
	return invoke(ctx, request, a.logger, "tasks_update", func() (taskOutput, error) {
		if input.Description != nil && input.ClearDescription {
			return taskOutput{}, &task.ValidationError{Fields: map[string]string{
				"clearDescription": "clearDescription cannot be combined with description",
			}}
		}
		domainInput := task.UpdateInput{Title: input.Title, Status: input.Status}
		if input.Description != nil || input.ClearDescription {
			domainInput.Description.Set = true
			domainInput.Description.Value = input.Description
		}
		item, err := a.tasks.Update(ctx, input.ID, domainInput)
		return taskOutput{Task: item}, err
	})
}

func (a *adapter) delete(ctx context.Context, request *mcp.CallToolRequest, input taskIDInput) (*mcp.CallToolResult, deleteTaskOutput, error) {
	return invoke(ctx, request, a.logger, "tasks_delete", func() (deleteTaskOutput, error) {
		err := a.tasks.Delete(ctx, input.ID)
		return deleteTaskOutput{Deleted: err == nil, ID: input.ID}, err
	})
}

func invoke[Output any](ctx context.Context, request *mcp.CallToolRequest, logger *slog.Logger, tool string, operation func() (Output, error)) (*mcp.CallToolResult, Output, error) {
	started := time.Now()
	correlationID := correlationID(request)
	output, err := operation()
	if err != nil {
		code := errorCode(err)
		logger.ErrorContext(ctx, "MCP tool failed", "component", "mcp-server", "event", "mcp.tool.completed", "correlationId", correlationID, "tool", tool, "durationMs", time.Since(started).Milliseconds(), "errorCode", code, "error", err)
		return &mcp.CallToolResult{Meta: mcp.Meta{"correlationId": correlationID}}, output, safeError(err)
	}
	logger.InfoContext(ctx, "MCP tool completed", "component", "mcp-server", "event", "mcp.tool.completed", "correlationId", correlationID, "tool", tool, "durationMs", time.Since(started).Milliseconds())
	return &mcp.CallToolResult{Meta: mcp.Meta{"correlationId": correlationID}}, output, nil
}

func correlationID(request *mcp.CallToolRequest) string {
	if request != nil && request.Params != nil {
		if value, ok := request.Params.Meta["correlationId"].(string); ok && strings.TrimSpace(value) != "" {
			return value
		}
	}
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err == nil {
		return hex.EncodeToString(raw[:])
	}
	return "unavailable"
}

func errorCode(err error) string {
	var validationError *task.ValidationError
	switch {
	case errors.As(err, &validationError):
		return "VALIDATION"
	case errors.Is(err, task.ErrNotFound):
		return "NOT_FOUND"
	case errors.Is(err, task.ErrConflict):
		return "CONFLICT"
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		return "CANCELED"
	default:
		return "INTERNAL"
	}
}

func safeError(err error) error {
	var validationError *task.ValidationError
	switch {
	case errors.As(err, &validationError):
		keys := make([]string, 0, len(validationError.Fields))
		for key := range validationError.Fields {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		parts := make([]string, 0, len(keys))
		for _, key := range keys {
			parts = append(parts, fmt.Sprintf("%s: %s", key, validationError.Fields[key]))
		}
		return fmt.Errorf("VALIDATION: %s", strings.Join(parts, "; "))
	case errors.Is(err, task.ErrNotFound):
		return errors.New("NOT_FOUND: the task was not found")
	case errors.Is(err, task.ErrConflict):
		return errors.New("CONFLICT: the task conflicts with existing data")
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		return errors.New("CANCELED: the request did not complete")
	default:
		return errors.New("INTERNAL: the service could not complete the request")
	}
}
