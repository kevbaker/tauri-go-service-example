package bridge

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"regexp"
	"time"

	"github.com/kbaker/tauri-go-service-example/services/go/task-service/internal/config"
	"github.com/kbaker/tauri-go-service-example/services/go/task-service/internal/task"
)

const ProtocolVersion = 1

const (
	OperationSystemHealth    = "system.health"
	OperationConfigGetPublic = "config.getPublic"
	OperationTasksList       = "tasks.list"
	OperationTasksGet        = "tasks.get"
	OperationTasksCreate     = "tasks.create"
	OperationTasksUpdate     = "tasks.update"
	OperationTasksDelete     = "tasks.delete"
)

var requestIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$`)

type Request struct {
	ProtocolVersion int             `json:"protocolVersion"`
	RequestID       string          `json:"requestId"`
	Operation       string          `json:"operation"`
	Payload         json.RawMessage `json:"payload,omitempty"`
}

type Response struct {
	ProtocolVersion int               `json:"protocolVersion"`
	RequestID       string            `json:"requestId"`
	OK              bool              `json:"ok"`
	Data            any               `json:"data,omitempty"`
	Error           *ApplicationError `json:"error,omitempty"`
}

type ApplicationError struct {
	Code        string            `json:"code"`
	Message     string            `json:"message"`
	FieldErrors map[string]string `json:"fieldErrors,omitempty"`
}

type TaskService interface {
	List(context.Context, task.ListQuery) ([]task.Task, error)
	Get(context.Context, string) (task.Task, error)
	Create(context.Context, task.CreateInput) (task.Task, error)
	Update(context.Context, string, task.UpdateInput) (task.Task, error)
	Delete(context.Context, string) error
}

type Dispatcher struct {
	tasks        TaskService
	publicConfig config.PublicConfig
	logger       *slog.Logger
	now          func() time.Time
}

func NewDispatcher(tasks TaskService, publicConfig config.PublicConfig, logger *slog.Logger) *Dispatcher {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	return &Dispatcher{tasks: tasks, publicConfig: publicConfig, logger: logger, now: func() time.Time { return time.Now().UTC() }}
}

func (d *Dispatcher) Invoke(ctx context.Context, request Request) Response {
	started := time.Now()
	response := Response{ProtocolVersion: ProtocolVersion, RequestID: request.RequestID}
	operation := request.Operation

	if fields := validateEnvelope(request); len(fields) > 0 {
		response.Error = &ApplicationError{Code: "VALIDATION", Message: "The request envelope is invalid", FieldErrors: fields}
		d.logResult(ctx, request.RequestID, operation, started, response.Error.Code)
		return response
	}

	data, err := d.dispatch(ctx, request)
	if err != nil {
		response.Error = mapError(err)
		d.logResult(ctx, request.RequestID, operation, started, response.Error.Code)
		return response
	}
	response.OK = true
	response.Data = data
	d.logResult(ctx, request.RequestID, operation, started, "")
	return response
}

func (d *Dispatcher) dispatch(ctx context.Context, request Request) (any, error) {
	switch request.Operation {
	case OperationSystemHealth:
		if err := decodeNoPayload(request.Payload); err != nil {
			return nil, validation("payload", err.Error())
		}
		return struct {
			Status          string    `json:"status"`
			ProtocolVersion int       `json:"protocolVersion"`
			Time            time.Time `json:"time"`
		}{Status: "ready", ProtocolVersion: ProtocolVersion, Time: d.now().UTC()}, nil
	case OperationConfigGetPublic:
		if err := decodeNoPayload(request.Payload); err != nil {
			return nil, validation("payload", err.Error())
		}
		return d.publicConfig, nil
	case OperationTasksList:
		var query task.ListQuery
		if err := decodePayload(request.Payload, &query); err != nil {
			return nil, validation("payload", err.Error())
		}
		return d.tasks.List(ctx, query)
	case OperationTasksGet:
		var payload struct {
			ID string `json:"id"`
		}
		if err := decodePayload(request.Payload, &payload); err != nil {
			return nil, validation("payload", err.Error())
		}
		return d.tasks.Get(ctx, payload.ID)
	case OperationTasksCreate:
		var input task.CreateInput
		if err := decodePayload(request.Payload, &input); err != nil {
			return nil, validation("payload", err.Error())
		}
		return d.tasks.Create(ctx, input)
	case OperationTasksUpdate:
		var payload updatePayload
		if err := decodePayload(request.Payload, &payload); err != nil {
			return nil, validation("payload", err.Error())
		}
		return d.tasks.Update(ctx, payload.ID, payload.Input.taskInput())
	case OperationTasksDelete:
		var payload struct {
			ID string `json:"id"`
		}
		if err := decodePayload(request.Payload, &payload); err != nil {
			return nil, validation("payload", err.Error())
		}
		if err := d.tasks.Delete(ctx, payload.ID); err != nil {
			return nil, err
		}
		return struct{}{}, nil
	default:
		return nil, validation("operation", "Operation is not supported")
	}
}

type updatePayload struct {
	ID    string          `json:"id"`
	Input updateInputWire `json:"input"`
}

type updateInputWire struct {
	Title       *string         `json:"title,omitempty"`
	Description json.RawMessage `json:"description,omitempty"`
	Status      *task.Status    `json:"status,omitempty"`
}

func (input updateInputWire) taskInput() task.UpdateInput {
	result := task.UpdateInput{Title: input.Title, Status: input.Status}
	if input.Description != nil {
		result.Description.Set = true
		if !bytes.Equal(bytes.TrimSpace(input.Description), []byte("null")) {
			var value string
			// decodePayload already validated this value through UnmarshalJSON.
			_ = json.Unmarshal(input.Description, &value)
			result.Description.Value = &value
		}
	}
	return result
}

func (input *updateInputWire) UnmarshalJSON(data []byte) error {
	type alias updateInputWire
	var decoded alias
	if err := decodeJSON(data, &decoded); err != nil {
		return err
	}
	if decoded.Description != nil && !bytes.Equal(bytes.TrimSpace(decoded.Description), []byte("null")) {
		var value string
		if err := decodeJSON(decoded.Description, &value); err != nil {
			return fmt.Errorf("description must be a string or null")
		}
	}
	*input = updateInputWire(decoded)
	return nil
}

func validateEnvelope(request Request) map[string]string {
	fields := map[string]string{}
	if request.ProtocolVersion != ProtocolVersion {
		fields["protocolVersion"] = "Protocol version is not supported"
	}
	if !requestIDPattern.MatchString(request.RequestID) {
		fields["requestId"] = "Request ID is invalid"
	}
	if request.Operation == "" {
		fields["operation"] = "Operation is required"
	}
	if isTaskOperation(request.Operation) {
		payload := bytes.TrimSpace(request.Payload)
		if len(payload) == 0 || bytes.Equal(payload, []byte("null")) {
			fields["payload"] = "Payload is required"
		}
	}
	return fields
}

func isTaskOperation(operation string) bool {
	switch operation {
	case OperationTasksList, OperationTasksGet, OperationTasksCreate, OperationTasksUpdate, OperationTasksDelete:
		return true
	default:
		return false
	}
}

func decodePayload(raw json.RawMessage, target any) error {
	if len(bytes.TrimSpace(raw)) == 0 || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		raw = []byte("{}")
	}
	return decodeJSON(raw, target)
}

func decodeNoPayload(raw json.RawMessage) error {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) || bytes.Equal(trimmed, []byte("{}")) {
		return nil
	}
	return errors.New("payload must be empty")
}

func decodeJSON(raw []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("expected one JSON value")
	}
	return nil
}

func validation(field, message string) error {
	return &task.ValidationError{Fields: map[string]string{field: message}}
}

func mapError(err error) *ApplicationError {
	var validationError *task.ValidationError
	switch {
	case errors.As(err, &validationError):
		return &ApplicationError{Code: "VALIDATION", Message: "The request is invalid", FieldErrors: validationError.Fields}
	case errors.Is(err, task.ErrNotFound):
		return &ApplicationError{Code: "NOT_FOUND", Message: "The task was not found"}
	case errors.Is(err, task.ErrConflict):
		return &ApplicationError{Code: "CONFLICT", Message: "The task conflicts with existing data"}
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		return &ApplicationError{Code: "INTERNAL", Message: "The request did not complete"}
	default:
		return &ApplicationError{Code: "INTERNAL", Message: "The service could not complete the request"}
	}
}

func (d *Dispatcher) logResult(ctx context.Context, requestID, operation string, started time.Time, errorCode string) {
	attributes := []any{
		"component", "go-service",
		"event", "bridge.request.completed",
		"requestId", requestID,
		"operation", operation,
		"durationMs", time.Since(started).Milliseconds(),
	}
	if errorCode != "" {
		attributes = append(attributes, "errorCode", errorCode)
		d.logger.WarnContext(ctx, "bridge request completed", attributes...)
		return
	}
	d.logger.InfoContext(ctx, "bridge request completed", attributes...)
}
