package bridge

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/kbaker/tauri-go-service-example/services/go/task-service/internal/config"
	"github.com/kbaker/tauri-go-service-example/services/go/task-service/internal/task"
)

type fakeTaskService struct {
	updateInput task.UpdateInput
}

func (f *fakeTaskService) List(context.Context, task.ListQuery) ([]task.Task, error) {
	return []task.Task{}, nil
}
func (f *fakeTaskService) Get(_ context.Context, id string) (task.Task, error) {
	if id == "missing" {
		return task.Task{}, task.ErrNotFound
	}
	return task.Task{ID: id}, nil
}
func (f *fakeTaskService) Create(_ context.Context, input task.CreateInput) (task.Task, error) {
	if strings.TrimSpace(input.Title) == "" {
		return task.Task{}, &task.ValidationError{Fields: map[string]string{"title": "Title is required"}}
	}
	return task.Task{ID: "one", Title: input.Title, Status: task.StatusTodo}, nil
}
func (f *fakeTaskService) Update(_ context.Context, id string, input task.UpdateInput) (task.Task, error) {
	f.updateInput = input
	return task.Task{ID: id}, nil
}
func (f *fakeTaskService) Delete(context.Context, string) error { return nil }

func newTestDispatcher(service TaskService) *Dispatcher {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	dispatcher := NewDispatcher(service, config.PublicConfig{}, logger)
	dispatcher.now = func() time.Time { return time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC) }
	return dispatcher
}

func TestInvokeMapsValidationAndNotFoundErrors(t *testing.T) {
	dispatcher := newTestDispatcher(&fakeTaskService{})
	tests := []struct {
		name      string
		operation string
		payload   string
		code      string
	}{
		{"validation", OperationTasksCreate, `{"title":""}`, "VALIDATION"},
		{"not found", OperationTasksGet, `{"id":"missing"}`, "NOT_FOUND"},
		{"unknown operation", "tasks.publish", `{}`, "VALIDATION"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := dispatcher.Invoke(context.Background(), Request{ProtocolVersion: 1, RequestID: "request-1", Operation: test.operation, Payload: json.RawMessage(test.payload)})
			if response.OK || response.Error == nil || response.Error.Code != test.code {
				t.Fatalf("Invoke() = %#v", response)
			}
		})
	}
}

func TestInvokePreservesExplicitNullDescription(t *testing.T) {
	service := &fakeTaskService{}
	dispatcher := newTestDispatcher(service)
	response := dispatcher.Invoke(context.Background(), Request{
		ProtocolVersion: 1,
		RequestID:       "request-1",
		Operation:       OperationTasksUpdate,
		Payload:         json.RawMessage(`{"id":"one","input":{"description":null}}`),
	})
	if !response.OK || !service.updateInput.Description.Set || service.updateInput.Description.Value != nil {
		t.Fatalf("Invoke() = %#v; input = %#v", response, service.updateInput)
	}
}

func TestInvokeRejectsUnsupportedProtocolAndCanceledContext(t *testing.T) {
	dispatcher := newTestDispatcher(&cancelingService{})
	unsupported := dispatcher.Invoke(context.Background(), Request{ProtocolVersion: 2, RequestID: "request-1", Operation: OperationTasksList})
	if unsupported.Error == nil || unsupported.Error.Code != "VALIDATION" {
		t.Fatalf("unsupported response = %#v", unsupported)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	canceled := dispatcher.Invoke(ctx, Request{ProtocolVersion: 1, RequestID: "request-2", Operation: OperationTasksList})
	if canceled.Error == nil || canceled.Error.Code != "INTERNAL" {
		t.Fatalf("canceled response = %#v", canceled)
	}
}

type cancelingService struct{ fakeTaskService }

func (f *cancelingService) List(ctx context.Context, _ task.ListQuery) ([]task.Task, error) {
	return nil, ctx.Err()
}

func TestDecodePayloadRejectsUnknownFieldsAndMultipleValues(t *testing.T) {
	dispatcher := newTestDispatcher(&fakeTaskService{})
	for _, payload := range []string{`{"id":"one","extra":true}`, `{"id":"one"} {}`} {
		response := dispatcher.Invoke(context.Background(), Request{ProtocolVersion: 1, RequestID: "request-1", Operation: OperationTasksGet, Payload: json.RawMessage(payload)})
		if response.Error == nil || response.Error.Code != "VALIDATION" {
			t.Fatalf("Invoke(%q) = %#v", payload, response)
		}
	}
}

func TestMapErrorDoesNotExposeInternalDetails(t *testing.T) {
	mapped := mapError(errors.New("database /secret/path failed"))
	if strings.Contains(mapped.Message, "secret") || mapped.Code != "INTERNAL" {
		t.Fatalf("mapError() = %#v", mapped)
	}
}
