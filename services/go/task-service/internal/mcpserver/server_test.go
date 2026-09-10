package mcpserver

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"slices"
	"strings"
	"testing"

	"github.com/kbaker/tauri-go-service-example/services/go/task-service/internal/task"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type fakeTaskService struct {
	updateInput task.UpdateInput
}

func (f *fakeTaskService) List(context.Context, task.ListQuery) ([]task.Task, error) {
	return []task.Task{{ID: "one", Title: "First", Status: task.StatusTodo}}, nil
}

func (f *fakeTaskService) Get(_ context.Context, id string) (task.Task, error) {
	if id == "missing" {
		return task.Task{}, task.ErrNotFound
	}
	return task.Task{ID: id, Title: "First", Status: task.StatusTodo}, nil
}

func (f *fakeTaskService) Create(_ context.Context, input task.CreateInput) (task.Task, error) {
	if strings.TrimSpace(input.Title) == "" {
		return task.Task{}, &task.ValidationError{Fields: map[string]string{"title": "Title is required"}}
	}
	return task.Task{ID: "created", Title: input.Title, Status: task.StatusTodo}, nil
}

func (f *fakeTaskService) Update(_ context.Context, id string, input task.UpdateInput) (task.Task, error) {
	f.updateInput = input
	return task.Task{ID: id, Title: "Updated", Status: task.StatusTodo}, nil
}

func (f *fakeTaskService) Delete(_ context.Context, id string) error {
	if id == "missing" {
		return task.ErrNotFound
	}
	return nil
}

func connect(t *testing.T, service TaskService) *mcp.ClientSession {
	t.Helper()
	ctx := context.Background()
	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	server := New(service, slog.New(slog.NewTextHandler(io.Discard, nil)))
	serverSession, err := server.Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatalf("server.Connect() error = %v", err)
	}
	client := mcp.NewClient(&mcp.Implementation{Name: "task-mcp-test", Version: "0.1.0"}, nil)
	clientSession, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		_ = serverSession.Close()
		t.Fatalf("client.Connect() error = %v", err)
	}
	t.Cleanup(func() {
		_ = clientSession.Close()
		_ = serverSession.Close()
	})
	return clientSession
}

func TestServerAdvertisesTaskTools(t *testing.T) {
	session := connect(t, &fakeTaskService{})
	var names []string
	for tool, err := range session.Tools(context.Background(), nil) {
		if err != nil {
			t.Fatalf("Tools() error = %v", err)
		}
		names = append(names, tool.Name)
		if tool.Name == "tasks_delete" && (tool.Annotations == nil || tool.Annotations.DestructiveHint == nil || !*tool.Annotations.DestructiveHint) {
			t.Fatalf("tasks_delete annotations = %#v", tool.Annotations)
		}
	}
	want := []string{"tasks_create", "tasks_delete", "tasks_get", "tasks_list", "tasks_update"}
	slices.Sort(names)
	if !slices.Equal(names, want) {
		t.Fatalf("tool names = %v, want %v", names, want)
	}
}

func TestToolsDelegateToTaskService(t *testing.T) {
	service := &fakeTaskService{}
	session := connect(t, service)

	created, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Meta: mcp.Meta{"correlationId": "test-correlation"},
		Name: "tasks_create", Arguments: map[string]any{"title": "Use the MCP adapter"},
	})
	if err != nil || created.IsError {
		t.Fatalf("tasks_create result = %#v, error = %v", created, err)
	}
	structured, ok := created.StructuredContent.(map[string]any)
	if !ok || structured["task"] == nil {
		t.Fatalf("tasks_create structured content = %#v", created.StructuredContent)
	}
	if created.Meta["correlationId"] != "test-correlation" {
		t.Fatalf("tasks_create metadata = %#v", created.Meta)
	}

	updated, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "tasks_update", Arguments: map[string]any{"id": "created", "clearDescription": true},
	})
	if err != nil || updated.IsError || !service.updateInput.Description.Set || service.updateInput.Description.Value != nil {
		t.Fatalf("tasks_update result = %#v, input = %#v, error = %v", updated, service.updateInput, err)
	}
}

func TestToolErrorsAreSafeAndActionable(t *testing.T) {
	session := connect(t, &fakeTaskService{})
	tests := []struct {
		name      string
		tool      string
		arguments map[string]any
		contains  string
	}{
		{name: "validation", tool: "tasks_create", arguments: map[string]any{"title": ""}, contains: "VALIDATION"},
		{name: "not found", tool: "tasks_get", arguments: map[string]any{"id": "missing"}, contains: "NOT_FOUND"},
		{name: "ambiguous description", tool: "tasks_update", arguments: map[string]any{"id": "one", "description": "text", "clearDescription": true}, contains: "cannot be combined"},
		{name: "invalid status schema", tool: "tasks_create", arguments: map[string]any{"title": "Invalid", "status": "blocked"}, contains: "status"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: test.tool, Arguments: test.arguments})
			if err != nil {
				t.Fatalf("CallTool() error = %v", err)
			}
			if !result.IsError || len(result.Content) == 0 || !strings.Contains(result.Content[0].(*mcp.TextContent).Text, test.contains) {
				t.Fatalf("CallTool() result = %#v", result)
			}
		})
	}
}

func TestSafeErrorDoesNotExposeInternalDetails(t *testing.T) {
	err := safeError(errors.New("database /secret/path failed"))
	if strings.Contains(err.Error(), "secret") || !strings.Contains(err.Error(), "INTERNAL") {
		t.Fatalf("safeError() = %v", err)
	}
}
